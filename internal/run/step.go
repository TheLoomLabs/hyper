package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// One Step: the binding resolved, the calls made, the responses projected, the
// versions the projections moved, and the file that says what became of it (§6,
// §7, issues #136, #140).
//
// **One path, three Kinds.** An effectful Step resolves, expands, calls and
// projects exactly as a `read` does; what a Kind changes is what a call's
// answer *means* and what a version it writes is a version of, and both of
// those are effect.go's (§6, issue #148).

// binding is what a Step is bound to, resolved: every artefact the Step names,
// read once, so that nothing below resolves a name a second time and gets a
// second answer.
//
// They travel together because they are one fact — *what this Step is bound
// to* — and because the Provenance a Step writes is read off three of them: the
// Definition's blob id, the Manifest's digest, and its `origin:` digest where
// it carries one (§7, ADR-0043).
//
// It is not called a Bound. A Bound is the maximum number of Records an
// effectful Step may affect (§5, CONTEXT.md), and this is the Step's binding —
// two words the glossary keeps apart and one letter would not.
type binding struct {
	definition repository.LoadedArtefact
	manifest   repository.LoadedManifest
	provider   artefact.ProviderInfo
	target     artefact.TargetInfo
	operation  artefact.OperationInfo
	// detail is what internal/artefact derived about the Operation, read
	// once here rather than once per member: the deadline that bounds each
	// call and the `concurrency:` limit that dispatched it are two facts
	// about one Operation, and two readings of one Manifest is where the
	// day comes that they disagree (§3, ADR-0045).
	detail artefact.OperationDetail
}

// perform runs one Step and answers what became of it.
//
// **The condition decides first and the Expansion resolves second**, both
// before the Step's first call goes out, and what either declines is a Refusal
// the Run ends on rather than a fault (§6, condition.go, expand.go). Past them
// the members are dispatched in Expansion order, each response projected and
// each version written only where the bytes moved.
//
// The error it answers is what halted the Run, and the Step it answers beside
// it stands whether or not there is one: a Step that reached a Disposition
// wrote its file, and a halted Run leaves what it did (§6, ADR-0011). A zero
// Step is a Step that reached none.
//
// **What a member's fault does to the members beside it follows the Kind.** A
// `read` Expansion drains — every member attempted, every Observation that
// succeeded recorded, and the Run then halting with the rest already on disk —
// and an effectful Expansion stops at the first error, everything it confirmed
// already committed. drain.go states why neither is a preference (§6).
func (r run) perform(position int, authored sequenced) (Step, []Refusal, error) {
	bound, err := resolve(r.request.Repository, authored)
	if err != nil {
		return Step{}, nil, err
	}
	provenance := store.StepProvenance{
		DefinitionRevision: revision.Blob(bound.definition.Bytes),
		ManifestDigest:     artefact.ManifestDigest(bound.manifest.Bytes),
		OriginDigest:       artefact.ReadManifestFacts(bound.manifest.Root).OriginDigest,
	}
	started := r.request.Now()
	file := store.StepFile{
		Step: position,
		Path: authored.Path,
		StepCode: store.StepCode{
			ID:         authored.ID,
			Definition: authored.Definition,
			Operation:  authored.Operation,
			Provider:   bound.manifest.Name,
			Target:     authored.Target,
			Kind:       store.Kind(bound.operation.Kind),
		},
		StartedAt:  started,
		Provenance: provenance,
	}
	reached := Step{
		Position: position, ID: authored.ID, Path: authored.Path,
		Kind: file.Kind, Provenance: provenance,
	}

	// **The condition decides before the Expansion resolves**, so a Step it
	// does not hold for expands over nothing, reaches no Target and cannot
	// Refuse on a selector it never resolved (§6, condition.go). Its file
	// therefore carries no `selector` block at all — a Step that resolved
	// none holds none (§7).
	held, declined := r.decided(authored, position)
	if len(declined) > 0 {
		reached.Disposition, file.Disposition = store.DispositionRefused, store.DispositionRefused
		return reached, declined, r.write(file)
	}
	if !held {
		reached.Disposition = store.DispositionSkippedByCondition
		file.Disposition = reached.Disposition
		return reached, nil, r.write(file)
	}

	// **Run-once decides next**, on the Journal alone and with no selector
	// resolved: a Step the Journal already holds as *ran* or as *attempted,
	// outcome unknown* Refuses `run-once-recorded` before its Expansion
	// reads anything, so what stopped it is what it already did rather than
	// what the Store happens to hold now (§6, §12, once.go).
	//
	// Its file is the *refused* Step's like any other, and it carries no
	// `selector` block: the Expansion below never ran, and a Step that
	// resolved no selector holds none (§7).
	declined, err = r.recordedAlready(bound, authored, position)
	if err != nil {
		return Step{}, nil, err
	}
	if len(declined) > 0 {
		reached.Disposition, file.Disposition = store.DispositionRefused, store.DispositionRefused
		return reached, declined, r.write(file)
	}

	expanded, declined, err := r.expand(bound, authored, position)
	if err != nil {
		return Step{}, nil, err
	}
	file.Selector = store.Selector{Declared: expanded.Selector.Declared, ExpandedTo: expanded.names()}

	// A Refusal at a Step's Expansion. The Step file records what actually
	// happened to that Step — its Disposition, its selector, and what it
	// expanded to — and carries **no identity set**, nothing having been
	// concluded about anything. The Refusal itself is held on
	// `outcome.json` and never here (§7, ADR-0061).
	if len(declined) > 0 {
		reached.Disposition, file.Disposition = store.DispositionRefused, store.DispositionRefused
		return reached, declined, r.write(file)
	}

	// The Store comparand for the identities that can only resolve once the
	// answers are in, read **once** and before the Step's first call goes
	// out — so a member of this Run's own Expansion is the sibling
	// comparand beside it rather than a series that was already standing
	// (§6, reading.go). It is read this early rather than after the calls
	// because an effectful Expansion writes between one call and the next,
	// and a comparand read at the second member's turn would be read
	// against a branch the first member's version is already on. An
	// Operation whose `identity:` resolves before the call reads nothing
	// here: the Expansion has already run both comparands over those
	// identities and Refused, with nothing touched.
	holders, err := r.heldBy(bound)
	if err != nil {
		return Step{}, nil, err
	}

	// What this Step concluded about, and the versions its conclusions
	// moved. A version is written only where the bytes moved: an Operation
	// returning what the head version already holds mints nothing, and the
	// canonical encoding is what makes that an exact test (§7, ADR-0030).
	names := make([]string, 0, len(expanded.Members))
	// `hyper`'s own account of the work, summed over the Expansion: a
	// Pattern's attempts, its pages and its poll iterations, supplied by no
	// Provider (§7, ADR-0018, pattern.go). It is read off every member,
	// faulted or not — a member that polled four times and then reached the
	// deadline did four polls, and the file says what `hyper` did to reach
	// the outcome it reached.
	var acted account
	// What this Step **acted on**, for the conditions and the references of
	// the Steps after it: the fields each call concluded, whether or not the
	// version they would have written moved the bytes. A Record going
	// unchanged is not a Record going missing (§6, ADR-0030, condition.go).
	records := make([]store.Mapping, 0, len(expanded.Members))
	// How many Record identities the Step **reached**, concluded about or
	// not: what `n of m` is read against, and what the arithmetic between it
	// and the set says are unaccounted for (§7, §8). It counts one per
	// Record the answers held and one per member that faulted, and a Record
	// whose identity collides is one reached that will not be concluded
	// about.
	reachedIdentities := 0
	// What a call gave back where it completed the Step without giving the
	// ordinary answer, which in this milestone is a `destroy`'s `404` and
	// nothing else. It is the **first** in Expansion order, and it reaches
	// the Step file only where nothing halted (§7, effect.go).
	var completed store.Answered
	var halted error
	// How many members the walk took, which on a `read` is every one of
	// them and on an effectful Step is every one up to and including the
	// fault. What it is read for is the arithmetic below the loop.
	attempted := 0
	// How many members `skip-if-recorded` found already standing and made
	// no call for. It is read for the Disposition alone: a Step whose
	// **every** member skipped is *skipped as already recorded* and a Step
	// any call went out from is *ran*, which is the whole of what this
	// number decides (§6, §12, ADR-0056, repeat.go).
	skipped := 0
	// The calls, made in Expansion order and bounded by how much of this
	// Step's Expansion may be in flight at once: the Operation's declared
	// `concurrency:` limit on a `read` — which arrives effective, so a
	// Manifest that declared nothing runs its Expansion one member at a
	// time — and **one** on an effectful Step, whatever that limit says
	// (§6, ADR-0045, drain.go, effect.go).
	//
	// **On a `read`, every member is attempted.** A member that faulted is
	// skipped and stops nothing: what it wrote is nothing, and the Run
	// halts on it once the rest of the Expansion has been written down,
	// which is the drain. The fault the Run carries is therefore the
	// **first in Expansion order** and not the first to arrive, which is
	// what keeps *which fault* out of the completion order's reach (§6,
	// drain.go).
	//
	// **An effectful Expansion stops at the first error**, everything it
	// confirmed already committed — one member at a time, each version down
	// before the next call goes out, so *three of five, then halt* is a
	// determinate fact a reviewer reads off `expanded_to` rather than a
	// race (§6, §7).
	//
	// **A member `hyper` could not read the answer back from writes what it
	// projected**, and it is not a widening of either rule. The response
	// arrived and part of it projected, so what projected is written and
	// the tenth member that did not is what the Run halts on: what a
	// half-projected response puts in doubt is the claim that its Records
	// are all of them, and that claim lives in the identity set rather than
	// in any Record (§6, ADR-0011, reading.go).
	//
	// **`skip-if-recorded` decides at each member's turn, in front of that
	// member's call.** The Operation declaring it is a `mutate`, so the walk
	// below is serial and *this member's turn* is a moment: the heads the
	// test reads are the branch as the members before it left it (§6,
	// repeat.go).
	for turn := range dispatch(bound.dispatched(), expanded.Members, func(resolving member) (answer, error) {
		recorded, err := r.recorded(bound, authored, resolving)
		if err != nil {
			return answer{}, err
		}
		if recorded != "" {
			return answer{skipped: recorded}, nil
		}
		return r.call(bound, authored, resolving)
	}) {
		attempted++
		// A fault the skip test raised in front of this member's call
		// halts the **Run** rather than the Step: no call was made, so
		// there is no answer for a Disposition to describe, and it
		// leaves by the door heldBy above leaves by (repeat.go).
		if lost := haltedBeforeTheCall(turn.Fault); lost != nil {
			return Step{}, nil, lost
		}
		// **A member that skipped concluded about its Record and made
		// no call.** So its identity is in the set — the skip test read
		// a head version, which is a conclusion about that identity
		// (§7, ADR-0030) — and it is in neither the Records held for
		// the Steps after it nor the account of the Patterns, a member
		// that called nothing having acted on nothing and done nothing
		// for an account to hold (§6, ADR-0056, condition.go).
		if name := turn.Concluded.skipped; name != "" {
			skipped++
			reachedIdentities++
			names = append(names, name)
			continue
		}
		acted.add(turn.Concluded.account)
		if completed == nil {
			completed = turn.Concluded.answered
		}
		if turn.Fault != nil {
			if halted == nil {
				halted = turn.Fault
			}
			// The one it could not conclude about: the identity it
			// could not read, the collection it could not find, or
			// the answer that never came.
			reachedIdentities++
		}
		// One member is one Record on an Operation of `one` cardinality
		// and one per member of the collection it walked on an Operation
		// of `series` — every page of it, where a pagination Pattern
		// walked several. A Pattern changes the number of Records a Step
		// affects nowhere: what a paginated call reaches is the same
		// collection an unpaginated one would have, arriving a page at a
		// time (§3, pattern.go).
		//
		// A member that faulted holds them only where `hyper` read part
		// of an answer that arrived; a member whose call never answered
		// holds none, and the loop is empty rather than skipped.
		if turn.Fault == nil || wroteWhatProjected(turn.Fault) {
			for _, concluded := range turn.Concluded.records {
				reachedIdentities++
				identity := authored.identity(concluded.name)
				// **An identity that resolved and collides
				// halts the same way**, and the member it
				// collided on is not written: it has no
				// identity of its own to be written under, and
				// a further version of the series it collided
				// with would put its content on the head with
				// the earlier member's beneath it (§6,
				// reading.go).
				if collision := holders.take(identity, projectedBy(expanded.Members[turn.At].Name, concluded.at)); collision != nil {
					if halted == nil {
						halted = fmt.Errorf("step %s: %w", named(authored), collision)
					}
					continue
				}
				names = append(names, concluded.name)
				records = append(records, concluded.fields)

				if err := r.minted(bound, authored, position, provenance, identity, concluded); err != nil {
					return Step{}, nil, err
				}
			}
		}
		// **The stop, and it is the whole of what an effectful
		// Expansion does differently.** Taking no further member starts
		// none, so the members after this one are never called — and
		// what this one confirmed is already on the branch, its version
		// having gone down above (§6, drain.go).
		//
		// It follows **the halt** and not the call: an identity that
		// resolved and collides ends the Run as surely as a `500` does,
		// so an Expansion of three whose second member collides is *1
		// of 3* rather than an Expansion that carried on past it (§6,
		// reading.go). On an effectful Step nothing earlier can have
		// set it, this line having stopped the walk if anything had.
		if halted != nil && bound.effectful() {
			break
		}
	}
	// The members the stop never reached. Each is one Record identity the
	// Expansion resolved (ADR-0070) and one the Step did not conclude
	// about, so the arithmetic §8 renders is *expanded to five, concluded
	// about three, two unaccounted for* — and which two those are is
	// `expanded_to`'s and nowhere else (§7, §8).
	reachedIdentities += len(expanded.Members) - attempted

	// The identity set, and the digest it is written against: the last Run
	// of this Procedure in which this Step carried one, which is never
	// simply the previous Run (§7, ADR-0055).
	previous, err := r.previousDigest(authored)
	if err != nil {
		return Step{}, nil, err
	}
	// The set is built before it is written because the count is read off
	// it here and cannot be read back off what is written: an entry whose
	// digest did not move carries no members at all. It is a **set**, so a
	// Step carrying no selector concludes about the one Record it would
	// write and an Expansion of three concludes about three (§6, §7, §8,
	// ADR-0030).
	concluded := store.Names(names)

	// **A drained Step's Disposition is *ran*.** The calls went out and the
	// answers that came back were recorded; what the set holds is the
	// members it concluded about, and what it does not hold is the rest —
	// which the entry says by the arithmetic between this set and
	// `expanded_to` beside it, and §8 renders as `n of m` (§6, §7, §8).
	//
	// **An effectful halt is the one thing that moves it**, and effect.go
	// decides which of §12's three values it moves to: a status outside
	// `2xx` is *ran* all the same, a deadline is *attempted, outcome
	// unknown*, and a request that provably never left is *attempted, world
	// untouched* (§6, issue #148).
	//
	// ***attempted, world untouched* is a fact about the Step and not about
	// its last call.** A Step that **acted on** anything at all touched the
	// world, whatever stopped the member after it, so it is *ran* carrying
	// what it concluded — which is what keeps *world untouched* literally
	// true of every Step that carries it, and what makes it the one failure
	// that is not Repeatability evidence (§7, ADR-0062).
	//
	// What it reads is what the calls concluded and **never the identity
	// set beside it**, and under `skip-if-recorded` those are two different
	// things: a member that skipped concluded about its Record without a
	// call, so a Step that skipped one member and whose next request
	// provably never left reached the world nowhere and carries this value
	// (§6, ADR-0056, repeat.go). The set is the arithmetic §7 renders and
	// this is the question §6 asks — *did any call this Step made reach the
	// world* — which is the same question one paragraph down decides the
	// two skip Dispositions by.
	reached.Disposition = dispositionAfter(halted)
	if reached.Disposition == store.DispositionAttemptedWorldUntouched && len(records) > 0 {
		reached.Disposition = store.DispositionRan
	}
	// ***skipped as already recorded* is the Step no call went out from**,
	// and *ran* is every Step one did — which claims no count and never
	// did, a `read` Step expanding over five hundred carrying the same
	// value. A **mixed** Step is therefore *ran*, and which of the two it
	// carries decides nothing about a later Run: the test reads the Store's
	// head version and never the Journal, so unlike run-once it consumes no
	// Disposition (§6, §12, ADR-0056, repeat.go).
	//
	// The count is read against the Expansion rather than against the walk
	// so that *every member* means every member the selector resolved to. A
	// Step whose Expansion resolved to nothing skipped nothing and is *ran*
	// with its set written empty, which is the Step §8's `0` is for and not
	// this one (§7, §8).
	if skipped > 0 && skipped == len(expanded.Members) {
		reached.Disposition = store.DispositionSkippedAsAlreadyRecorded
	}
	// The identity set, carried by three of §12's seven Dispositions and by
	// *attempted, world untouched* never: nothing was concluded about
	// anything by construction, and a set written empty there would render
	// the safest state in the tool as a *ran* Step that concluded about
	// nothing (§7, §8, ADR-0062).
	//
	// Expanded is written on the halted Step alone. A Step that reached the
	// end of its Expansion accounted for all of it and has nothing for a
	// second number to say, which is what keeps `n of m` meaning
	// *unaccounted for* rather than *these two counts differ*.
	//
	// What it counts is Record identities and never Expansion members, which
	// is Step.Expanded's own sentence (run.go) and is why the count is built
	// above rather than read off `expanded_to` here.
	if reached.Disposition != store.DispositionAttemptedWorldUntouched {
		reached.Records, reached.Concluded = len(concluded), true
		if halted != nil {
			reached.Expanded = reachedIdentities
		}
	}
	// What this Step acted on is held for the Steps after it at the moment
	// it reaches its Disposition, which is the moment §6 fixes: a Step's
	// Records are written as each call confirms, and all of it before the
	// next Step starts.
	r.acted[stepKey{authored.Namespace, authored.ID}] = records
	file.Disposition = reached.Disposition
	if reached.Concluded {
		file.Identities = store.Concluded(names, previous)
	}
	file.Pattern = acted.written(reached.Disposition)
	// What an effectful call gave back where it did not give the ordinary
	// answer: the host and the status. Its presence is the fact that
	// something other than the ordinary answer decided this Step — the
	// halt, the `404` that completed a `destroy`, or the request that never
	// left — and it is written on effectful Steps and **never on a `read`**
	// (§7, effect.go).
	file.Answered = whatAnswered(halted, completed)
	// The path that failed to project, where that is what halted the Run.
	// The set beside it is then partial and this path is what says so — the
	// digest says nothing about partiality either way — and it is held here
	// and nowhere else: a rendering goes to a terminal that scrolls, and no
	// surface shows the response it failed against (§7, ADR-0017).
	file.ProjectionFailedPath = failedPath(halted)
	// The Step's file goes down whether or not the Expansion drained: a
	// Step that reached a Disposition wrote its file, and a halted Run
	// leaves what it did (§6, ADR-0011). The fault travels beside it and is
	// what makes the Run `failed`.
	if err := r.write(file); err != nil {
		return Step{}, nil, err
	}
	return reached, nil, halted
}

// minted writes one Record version, where the bytes moved.
//
// **A version is written only where the bytes moved.** An Operation returning
// what the head version already holds mints nothing and its Record is in the
// identity set all the same, the canonical encoding being what makes *the bytes
// moved* an exact test rather than an approximate one (§7, ADR-0030, mints).
//
// **It is committed as the call that produced it confirms** — one commit per
// confirmed write, and no batching (§7, ADR-0006). On a serial effectful
// Expansion that is the whole of why a Step halted at the fourth of five leaves
// three Tombstones on the branch: each went down before the next call went out.
//
// **What a `destroy` writes is a Tombstone**, which is an ordinary version of
// the Asset's own series carrying `tombstone: true`, the previous Head's
// `fields` copied forward, and the `operation`, `run_id` and `step` every
// version carries anyway (§7, ADR-0033). Its `written_at` is when destruction
// was confirmed, which is the instant read here.
func (r run) minted(bound binding, authored sequenced, position int, provenance store.StepProvenance, identity store.Identity, concluded conclusion) error {
	fields := concluded.fields
	if bound.tombstones() {
		// The Asset's last known state, read off the Store rather than
		// off what came back: a `destroy` projects nothing, and the
		// fields were projected by some earlier Operation while the
		// `operation` beside them names the one that destroyed it —
		// which is the one place in the Store those two keys describe
		// different calls (§7).
		carried, err := lastKnown(r.request.Store, identity)
		if err != nil {
			return err
		}
		fields = carried
	}

	version := store.RecordVersion{
		Metadata: store.Metadata{
			Identity:   identity,
			RecordType: bound.recordType(),
			Run:        r.id,
			Step:       position,
			Operation:  authored.Operation,
			// The invocation chain, where this Step was reached
			// through one. A Record version written by a nested
			// Step carries that Step's `path` as the Step's own
			// file does (§7, issue #141).
			Path:       authored.Path,
			WrittenAt:  r.request.Now(),
			Provenance: store.Provenance{Run: r.provenance, Step: provenance},
			Tombstone:  bound.tombstones(),
		},
		Fields: fields,
	}
	moved, err := mints(r.request.Store, version)
	if err != nil {
		return err
	}
	if !moved {
		return nil
	}
	return r.request.Store.Append([]store.Write{{
		Path:    store.RecordPath(identity, r.id, position),
		Content: version.Encode(),
	}}, fmt.Sprintf("Record %s/%s/%s at run %s step %d", identity.Target, identity.Definition, identity.Name, r.id, position))
}

// lastKnown is the Head version's `fields` for one series, and nothing at all
// where the Store holds no series under that identity.
//
// It is what a Tombstone copies forward for the Asset's last known state (§7).
// The absence is not a fault: a `destroy` by literal identifier reaches a member
// naming no series at all — that being what the form exists for — and the
// Tombstone opens one under it carrying no `fields`, which means *hyper
// destroyed this and never observed what it was* (ADR-0033).
func lastKnown(held *store.Store, id store.Identity) (store.Mapping, error) {
	head, standing, err := held.Head(id)
	if err != nil || !standing {
		return nil, err
	}
	previous, err := held.Read(head)
	if err != nil {
		return nil, err
	}
	return previous.Fields, nil
}

// decided evaluates the Step's `when:` and answers whether it holds.
//
// A Step carrying none holds unconditionally, which is what a Step with no
// condition is. Everything else is condition.go's: the Records the named Step
// of **this Run** acted on, the eleven operators against them, and the skip
// that propagates where that Step acted on nothing.
//
// **A predicate handed a value it cannot compare Refuses here**, as it does at
// an Expansion and for the same reason: a Record that quietly failed to compare
// is indistinguishable from one that compared and did not match (§12,
// ADR-0035). It is a Refusal rather than a halt because it decides before the
// Step's first call goes out — earlier, in fact, than the Expansion that would
// have resolved the population (§6, ADR-0072).
//
// It reaches no Store and so answers no error. What it reads is what the Steps
// of this Run already did, which the Run is holding.
func (r run) decided(authored sequenced, position int) (bool, []Refusal) {
	when, carried := readCondition(authored.When)
	if !carried {
		return true, nil
	}

	held, mismatch := when.holds(r.acted[stepKey{authored.Namespace, when.Step}], r.started)
	if mismatch == "" {
		return held, nil
	}

	cited := r.citation(authored, position, selector{})
	return false, []Refusal{r.refusal(CodePredicateTypeMismatch,
		fmt.Sprintf("on the Record step %s acted on, %s", when.Step, mismatch),
		cited.at(when.Line, "when."+when.Operator))}
}

// write puts one Step's file down, which is the last thing that happens at a
// Step's own turn: a Step writes its file as it reaches its Disposition (§6,
// §7).
//
// It takes the file and nothing beside it. The position the entry names it by
// and the id the commit message carries are members of the file already, and a
// caller passing either a second time is a caller that can pass a different
// one.
func (r run) write(file store.StepFile) error {
	file.EndedAt = r.request.Now()
	return r.request.Store.Append([]store.Write{{
		Path:    r.entry.StepPath(file.Step),
		Content: file.Encode(),
	}}, fmt.Sprintf("Step %d %s of run %s: %s", file.Step, file.ID, r.id, file.Disposition))
}

// resolve reads every artefact the Step names, and answers the one fault a Step
// can have that this milestone reaches: a name that resolves to nothing.
//
// **It is unreachable from a Run**, and it is written anyway. `check` re-runs in
// full at Run start (§6, gates.go), and every name a Step writes that resolves
// to nothing is `artefact-absent` or `reference-unresolvable` there — so a Run
// that reaches Step 1 is a Run whose every binding resolved. What stands here
// is the honest answer for a caller that reached the engine another way: the
// Step cannot be performed, and a halt says so without claiming an `error_code`
// no check produced (§12, ADR-0060).
func resolve(loaded repository.Loaded, authored sequenced) (binding, error) {
	// The two halves of the Definition namespace: what the name resolves to,
	// and the file it was read from. They are folded from one walk, so one
	// answering and the other not is a repository nobody could have loaded
	// (issue #121) — hence one test over both.
	info, declared := loaded.Definitions[authored.Definition]
	definition, held := loaded.Definition(authored.Definition)
	if !declared || !held {
		return binding{}, fmt.Errorf("step %s names definition %s, which resolves to nothing — hyper check reports it", named(authored), authored.Definition)
	}
	manifest, published := loaded.Manifests[info.ProviderName]
	provider, exposed := loaded.Providers[info.ProviderName]
	if !published || !exposed {
		return binding{}, fmt.Errorf("step %s binds definition %s, whose provider %s resolves to nothing — hyper check reports it", named(authored), authored.Definition, info.ProviderName)
	}
	target, granted := loaded.Targets[authored.Target]
	if !granted {
		return binding{}, fmt.Errorf("step %s names target %s, which resolves to nothing — hyper check reports it", named(authored), authored.Target)
	}
	operation, declares := provider.Operations[authored.Operation]
	if !declares {
		return binding{}, fmt.Errorf("step %s names operation %s, which %s declares nothing of that name — hyper check reports it", named(authored), authored.Operation, info.ProviderName)
	}
	return binding{
		definition: definition, manifest: manifest, provider: provider, target: target, operation: operation,
		detail: artefact.ReadOperationDetail(manifest.Root, authored.Operation),
	}, nil
}

// conclusion is what one call concluded about one Record: the identity it
// projected, and the fields under it.
type conclusion struct {
	name   string
	fields store.Mapping
	// at is where this Record was read out of the response: its position in
	// the collection an Operation of `series` cardinality named, counted
	// from 1 across the member's whole walk, and 0 on an Operation of `one`
	// cardinality, which projects one Record out of the response itself.
	//
	// It is carried for exactly one reader: an identity collision names the
	// Record that projected it, and *the tenth of what this member read* is
	// the only name such a Record has (§6, reading.go).
	at int
}

// answer is what one member of an Expansion concluded: the Records its calls
// projected, and `hyper`'s own account of what the Patterns around those calls
// did to reach them (§7, pattern.go).
//
// The Records are a list because two facts about an Operation multiply into it
// and neither is the member's: an Operation of `series` cardinality projects
// many Records out of one response, and a pagination Pattern walks many
// responses. Neither changes what the member **is** — one Record identity each
// is the Expansion's rule and holds unchanged (ADR-0070) — and both change how
// many Records one call reaches.
//
// The account travels beside them rather than being derived from them, because
// it is a fact about `hyper`'s own conduct and not about what came back: three
// attempts that reached one Record and one attempt that reached one Record
// leave the same Record and are not the same fact on the page (§7, ADR-0018).
type answer struct {
	records []conclusion
	account account
	// answered is what a call gave back where it **completed** the Step
	// without giving the ordinary answer, and nil everywhere else — which
	// in this milestone is a `destroy`'s `404` and nothing besides. What an
	// answer that halted gave back travels on the fault instead, there
	// being no member to carry it back on (§7, effect.go).
	answered store.Answered
	// skipped is the Record name `skip-if-recorded` found still standing,
	// and "" where this member's call went out — which on every Operation
	// that does not declare the value is every member (§6, repeat.go).
	//
	// It is the name rather than a flag because the name is what the answer
	// **is**: the member concluded about that identity and about nothing
	// else, and the set the Step carries is built out of it exactly as it is
	// built out of a projection's (§7, ADR-0030). A member that skipped
	// carries no Records, no account and no answer, none of the three having
	// a call to have come from.
	skipped string
}

// call makes one member's calls and projects the answers.
//
// **A `read` never halts on what came back.** Whatever the status, whatever the
// exit code, the response object §12 states is what the projection reads — so a
// host that answered nothing records an Observation whose status has gone quiet
// and a command that exited `1` records the code, rather than either stopping
// the Run (§6, ADR-0050). What halts it is the two things that are not an
// answer: the Operation's **deadline**, and the projection — in this milestone
// the identity path and the collection path, the recorded fields being an
// absence a version simply does not carry and the rest of what a projection
// that does not resolve does being issue #144's.
//
// **The Operation's `deadline:` bounds the whole call**, its Patterns' pages and
// polls included: it is taken once here and every call the Patterns make below
// is made under it, so there is no second bound anywhere to disagree with the
// first (§3, §6).
//
// It is one member's and never the Step's, and the error it answers halts
// nothing on its own: what a fault here does to the members beside it is the
// drain, one caller up (§6, drain.go). The account is answered whether or not
// it halted, a Step's file saying what `hyper` did to reach the outcome it
// reached (§7).
//
// **The two Capabilities part at requesting and rejoin at the projection.**
// Which of them an Operation declares is the Manifest's one fact about its
// request (§3), and it is read in exactly two places in this package: there,
// and at the Expansion, where a `shell` Operation's `command:` resolves to an
// argv rather than to an input (arguments.go). Everything past the call — the
// identity, the projection, the version — reads a response object and never a
// socket or a process.
func (r run) call(bound binding, authored sequenced, resolving member) (answer, error) {
	// **A `destroy` declares none, and needs none.** It projects nothing,
	// and the series its Tombstone goes into is the one the Expansion acted
	// on rather than anything read out of what came back (§3, §7, ADR-0037,
	// effect.go). Every other Kind must say which Record a call is holding.
	if bound.operation.Identity == "" && !bound.tombstones() {
		return answer{}, fmt.Errorf("step %s binds %s %s, whose record: declares no identity, so hyper cannot say which Record a call would be holding — hyper check reports it",
			named(authored), bound.manifest.Name, authored.Operation)
	}

	declaration := artefact.OperationNode(bound.manifest.Root, authored.Operation)
	reading := projection.Read(declaration)

	ctx, cancel := capability.Deadline(context.Background(), bound.detail.DeadlineSeconds)
	defer cancel()

	send, err := r.requesting(bound, authored, resolving, declaration)
	if err != nil {
		return answer{}, err
	}

	// Each page's response is projected as it arrives, and what it
	// projected is what pagination terminates on: **both forms terminate
	// when the collection `record.over` names comes back empty** (§3), and
	// the one reading of that collection is this one. A second reading for
	// the Pattern's own sake would be a second chance for *the collection
	// was empty* and *nothing was projected* to disagree.
	var projected []conclusion
	// What this member's call completed on where it was not the ordinary
	// answer: the first such page's, a Pattern's calls being one member's
	// walk and the key naming what decided the Step (§7).
	var answered store.Answered
	acted, halted := readPatterns(declaration).perform(ctx, r.started, send,
		func(response capability.Object) (int, error) {
			// **What an effectful Step makes of the answer, before
			// anything is projected off it.** A `mutate` completes
			// on `2xx` and halts on everything else, and a
			// response that never arrived halts with the world
			// untouched — so nothing below reads a body `hyper`
			// has already decided says its effect did not happen
			// (§6, effect.go). A `read` is judged by nothing and
			// falls straight through.
			completed, fault := bound.judged(authored, response)
			if fault != nil {
				return 0, fault
			}
			if completed != nil && answered == nil {
				answered = completed
			}
			// The Records already read out of this member's walk,
			// which is where this page's collection carries on
			// counting: the pages are one collection arriving in
			// instalments, so the position a collision names is the
			// one a reader counting them would reach (§3, §6).
			held, err := r.concluded(bound, authored, reading, resolving, response, len(projected))
			projected = append(projected, held...)
			return len(held), err
		})
	return answer{records: projected, account: acted, answered: answered}, halted
}

// concluded is what one response became: the Records it projected, and the halt
// where `hyper` could not read the answer back.
//
// **An Operation of `series` cardinality reads from two roots.** `over:` reads
// from the response, and `identity:` and every `fields:` path read from each
// member of the collection it named — both written `$`, the position deciding
// which root it means (§3, §12, internal/projection).
//
// The collection path failing to resolve halts the Run, and it is not the same
// fact as a collection that came back empty: without it `hyper` cannot tell
// *there was nothing there* from *the path was wrong*, which is the *I recorded
// nothing* an absent wire would otherwise be needed to diagnose (§6, ADR-0017).
// An empty collection is an ordinary answer and projects nothing.
//
// **The failure can be one member's, and what projected is written.** The
// collection path resolves, nine members project, and the tenth's identity path
// does not: the nine are answered beside the halt and the tenth is not, there
// being no identity to write it under, and the Run halts leaving what it did
// (§6, ADR-0011, issue #144). The drain rule does not decide this — that rule
// is scoped to a Step's Expansion, and a `series` response is one call the
// Expansion resolved to.
//
// already is how many Records this member's walk has read before this response,
// which is what the positions below count on from.
func (r run) concluded(bound binding, authored sequenced, reading projection.Projection, resolving member, response capability.Object, already int) ([]conclusion, error) {
	// **A `destroy` reads nothing out of the answer.** What it concluded is
	// the Asset the Expansion acted on and no projected content at all, so
	// there is no path here that could fail to resolve — a `destroy` carries
	// no `record:` block for one to be written in (§3, §7, ADR-0037,
	// effect.go).
	if bound.tombstones() {
		return destroyed(resolving), nil
	}
	if reading.Over == "" {
		name, resolved := identityOf(bound.operation, resolving.Inputs, response)
		if !resolved {
			return nil, unreadable(bound.operation.Identity,
				"step %s: the identity path %s did not resolve against what came back, so hyper cannot say which Record it is holding",
				named(authored), bound.operation.Identity)
		}
		return []conclusion{{name: name, fields: projected(bound.operation, reading.Project(response))}}, nil
	}

	members, resolved := projection.Collection(reading.Over, response)
	if !resolved {
		return nil, unreadable(reading.Over,
			"step %s: the collection path %s did not resolve against what came back, so hyper cannot tell a collection that was empty from a path that was wrong",
			named(authored), reading.Over)
	}

	held := make([]conclusion, 0, len(members))
	for at, item := range members {
		name, resolved := identityOf(bound.operation, resolving.Inputs, item)
		if !resolved {
			return held, unreadable(bound.operation.Identity,
				"step %s: the identity path %s did not resolve against record %d of %s, so hyper cannot say which Record it is holding",
				named(authored), bound.operation.Identity, already+at+1, reading.Over)
		}
		held = append(held, conclusion{
			name:   name,
			fields: projected(bound.operation, reading.Project(item)),
			at:     already + at + 1,
		})
	}
	return held, nil
}

// requesting is the one request this member makes, ready to be made again: the
// reach resolved and the holes filled once, and a closure that performs it
// under whatever page position a pagination Pattern hands down.
//
// It is built **once per member** and called once per page, poll and attempt,
// which is what keeps a Pattern from re-resolving a host or re-filling a hole
// between two calls of one member: the reach is the artefact's and cannot move
// during a call (§3, ADR-0029).
//
// What each arm answers beside the object is what a Pattern reads: whether the
// failure provably preceded the request, which is the only thing a retry
// follows (ADR-0018), and the **halting** fault, worded for a reader and nil
// where nothing halted. Everything else stays narration's and is dropped there:
// no member of the response object says what went wrong, that being the
// catch-all bucket ADR-0017 closed, and a `read` records a call that got no
// answer as the answer it is (§6, ADR-0050).
func (r run) requesting(bound binding, authored sequenced, resolving member, declaration *yaml.Node) (request, error) {
	if bound.operation.IsShell {
		return r.running(bound, authored, resolving)
	}
	return r.requested(bound, authored, resolving, declaration)
}

// requested is the `http` half: the reach resolved, the holes filled, and the
// call made — decorated, where a pagination Pattern is walking, with the token
// or the number written into the `query:` or `header:` position `into:` names.
func (r run) requested(bound binding, authored sequenced, resolving member, declaration *yaml.Node) (request, error) {
	// The inputs are the Expansion's, resolved for this member before the
	// first call of the Step went out: an `args:` value arriving from a
	// reference is read there, where a value it cannot read is still a
	// Refusal (§6, expand.go).
	inputs := resolving.Inputs

	reach := artefact.ResolveHost(bound.provider, bound.operation, bound.target,
		bound.operation.SuppliedHost(inputs))
	if reach.Reach != artefact.ReachGranted {
		return nil, fmt.Errorf("step %s reaches no host %s grants — hyper check reports it", named(authored), authored.Target)
	}

	declared, legible := capability.ReadRequest(declaration)
	if !legible {
		return nil, fmt.Errorf("step %s binds %s %s, which declares no legible http: block — hyper check reports what is wrong with it",
			named(authored), bound.manifest.Name, authored.Operation)
	}
	built, err := declared.Build(reach.Host, inputs)
	if err != nil {
		return nil, err
	}
	credential := r.credential(bound, authored.Target)

	return func(ctx context.Context, at page) (capability.Object, bool, error) {
		// The instant handed to the call is the **Run's** start and not
		// a fresh read: it is what a certificate's remaining life is
		// counted from, so two Steps of one Run that reach one host
		// record one `days_left`, and nothing a later Step — or a poll
		// of this one — does moves what an earlier one recorded
		// (ADR-0034).
		response, err := at.write(built).Perform(ctx, r.request.Dial, r.started, credential)

		// **A deadline reached fails the Step** (§6). It is the one
		// error beside the response object this reads as a halt, and it
		// is read because it is the one an artefact declared: a refused
		// connection, a name that does not resolve and a handshake that
		// failed are all *no response arrived*, which a `read` records
		// as the answer it is and an effectful Step reads off the
		// object one layer up — and which a retry Pattern follows,
		// those three being exactly the class the request provably
		// never left under (ADR-0018, effect.go).
		//
		// Which Disposition the halt carries is the Step's Kind's: a
		// `read` is *ran* and an effectful Step is *attempted, outcome
		// unknown*, the call having gone out with no answer to say
		// whether it landed (§12, effect.go).
		//
		// The deadline is named as itself rather than as the transport's
		// word for it, and beside the host it was reached on — the two
		// facts a reader can act on, one an edit to the Manifest and one
		// a look at the far end. Which **member** drained is
		// `expanded_to`'s and nowhere else (§7, §8).
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, false, bound.haltedByDeadline("step %s: the Operation's deadline of %s was reached on %s and no response arrived",
				named(authored), bound.detail.Deadline, reach.Host)
		}
		return response, capability.NeverSent(err), nil
	}, nil
}

// running is the `shell` half: the argv exec'd, and the four-member object §12
// closes over what the child did (issue #142).
//
// **No grant is consulted and there is no host to resolve.** `shell` is the one
// Capability whose reach no grant bounds (§13): what bounds a shell Step is the
// words a reviewer read in the Procedure, and its first word — the reach axis —
// is a literal `command-malformed` already refused anything else at, offline and
// with no Store in hand (§3, ADR-0051).
//
// The three things no artefact decides arrive here rather than off the Manifest:
// the working directory is the repository root, so a laptop and a runner agree
// without a line saying so; stdin is empty; and the environment is the one the
// Run composed once before Step 1 (§3, §11, run.go).
//
// **The page position reaches nothing here**, and it is unreachable rather than
// ignored: a `shell:` block has no `query:` and no `header:` for an `into:` to
// name, only `hyper` may write a Provider declaring this Capability (ADR-0039),
// and the one it ships declares no Pattern at all — pagination and polling
// having no meaning against a command, and retry following only a failure that
// provably preceded a request (§12, ADR-0018). What does reach here is retry,
// whose one member under this Capability is a child that could not be started
// at all (ADR-0018, capability.NeverSent).
func (r run) running(bound binding, authored sequenced, resolving member) (request, error) {
	if len(resolving.Argv) == 0 {
		// Unreachable from a Run: `command-malformed` refuses an empty
		// `command:` at load and the Expansion refuses a shape it could
		// not read. It says so rather than exec'ing nothing, an argv
		// with no head being a call `hyper` cannot describe (ADR-0064).
		return nil, fmt.Errorf("step %s resolved no argv, and there is no executable to name — hyper check reports it", named(authored))
	}

	command := capability.Command{Argv: resolving.Argv}
	return func(ctx context.Context, _ page) (capability.Object, bool, error) {
		response, err := command.Perform(ctx, r.request.Exec, r.request.RepoRoot, r.environment)

		// **A deadline reached fails the Step** (§6), and it is the one
		// error beside the object this reads as a halt; which
		// Disposition it carries is the Step's Kind's (effect.go). A
		// command that could not be started at all is *no answer*
		// rather than a fault — the object is `command` and nothing
		// else, and a `read` records the attempt with its `exit_code`
		// gone quiet (§12, ADR-0050).
		//
		// The child's whole process group has been killed with SIGKILL
		// and no grace period by the time this line runs, so a command's
		// own children do not outlive the deadline that bounded it (§6,
		// cli.Child). The deadline is named as itself and beside the
		// command it bounded — the two facts a reader can act on, one an
		// edit to the Manifest and one a look at what the Procedure
		// asked for.
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, false, bound.haltedByDeadline("step %s: the Operation's deadline of %s was reached and %s was killed",
				named(authored), bound.detail.Deadline, command.Text())
		}
		return response, capability.NeverSent(err), nil
	}, nil
}

// credential is the header this Step's call carries: the Auth scheme the
// Provider's Manifest names, composed out of the slots the credential pass
// already resolved for this Target.
//
// **Nothing is read here.** §6 resolves the credentials of every Target the Run
// may bind once, before Step 1, so what this does is compose — and the
// environment is not reachable from this package at all, every process read
// being threaded through Request. A Provider naming no `auth:` composes the
// empty Credential, which is what an uptime check against a public host is (§3,
// §6, §12, ADR-0007).
//
// The scheme is read a second time here, and it is the same reading: gates.go's
// credential pass asked `capability.ReadAuth` which slots to resolve, and this
// asks it which header to write them into. Both go through that one function
// over that one Manifest root — the gate reaches it through the Definition's
// name because no Step has resolved yet, and this reaches it through the
// binding that just did — so the slots a Run holds and the scheme it sends them
// under cannot come apart.
//
// The value goes from here onto the wire and reaches nothing else. It is not
// held on the binding, not written into the Call, and has no accessor: a
// credential is suppressed by the position it occupies rather than by every
// surface remembering to (ADR-0007, ADR-0031).
func (r run) credential(bound binding, target string) capability.Credential {
	return capability.ReadAuth(bound.manifest.Root).Credential(r.credentials[target])
}

// identityOf is the name the Record is held under: what the Operation's
// `identity:` resolved to.
//
// It reads from whichever of the two roots the Manifest wrote, which is decided
// by the spelling and by nothing else (§3): a template fills from the resolved
// inputs before the call, and a `$`-rooted path resolves against what came back
// after it. Which of the two a Manifest declares is what decides whether an
// identity collision Refuses at Expansion or halts the Run (§6, ADR-0072).
//
// root is the response object on an Operation of `one` cardinality and one
// member of the collection `over:` named on an Operation of `series` — the two
// roots §12 gives a path, told apart by the position and never by the path
// (internal/projection).
func identityOf(operation artefact.OperationInfo, inputs map[string]schema.Scalar, root any) (string, bool) {
	declared := operation.Identity
	if declared == "" {
		return "", false
	}
	if !strings.HasPrefix(declared, "$") {
		filled, err := capability.Fill("identity:", declared, inputs)
		return filled, err == nil && filled != ""
	}

	value, resolved := projection.Resolve(declared, root)
	if !resolved {
		return "", false
	}
	name, isText := value.(string)
	if !isText {
		name = projection.Text(value)
	}
	return name, name != ""
}

// projected is what the version's `fields` holds: every field that resolved, in
// the Store's own value types, with the ones the Manifest declares `secret:`
// written as the constant marker in the position the value would occupy (§7,
// ADR-0007).
//
// A field whose path resolved to nothing is not here at all — absence is the
// answer, and the field not being written is what carries it (§6, §12).
func projected(operation artefact.OperationInfo, fields projection.Fields) store.Mapping {
	mapping := store.Mapping{}
	for _, field := range fields {
		value, holdable := stored(field.Value)
		if !holdable {
			continue
		}
		if operation.SecretFields[field.Name] {
			value = store.Secret(value)
		}
		mapping[field.Name] = value
	}
	return mapping
}

// mints says whether this version's content differs from what the series' head
// version already holds — *the bytes moved*, made an exact test by the
// canonical encoding rather than an approximate one (§7).
//
// It compares the content and never the whole file. A version restates the Run,
// the Step and the instant that wrote it, so two files of one unchanged Record
// differ in every case and a comparison of them would mint a version on every
// Run — which is precisely the reading ADR-0030 exists to refuse.
func mints(held *store.Store, version store.RecordVersion) (bool, error) {
	head, standing, err := held.Head(version.Identity)
	if err != nil {
		return false, err
	}
	if !standing {
		return true, nil
	}
	previous, err := held.Read(head)
	if err != nil {
		return false, err
	}
	if previous.RecordType != version.RecordType || previous.Tombstone != version.Tombstone {
		return true, nil
	}
	return !slices.Equal(store.Encode(previous.Fields), store.Encode(version.Fields)), nil
}

// previousDigest is the identity digest this Step carried in the last Run **of
// this Procedure** in which it carried one, and "" where there is no such Run —
// a Step's first, and a Step whose authored id moved, which is a different Step
// with no digest behind it (§7, ADR-0055).
//
// It is a backward walk over the Journal's date partitions, and it terminates
// at the first record it finds carrying a set: three of §12's seven
// Dispositions carry none and a fourth writes no file, so the comparand is the
// last Run that carried one rather than the previous Run.
//
// The filter between the scan and the answer is store.Evidence.Comparable's
// and is stated there: a rehearsal, another Procedure's entry and another
// invocation chain's are each out, and each for a reason that holds as much for
// the reader that resolves a set back as for this, which is why the rule sits
// beside the walk rather than at either end of it (§7, ADR-0001, ADR-0055).
//
// Where one Procedure is invoked twice the two chains are one path
// (sequence.go), and the comparand is then the more recent of the two: §7
// matches a Step by what it was authored as, and those two Steps were authored
// as one.
func (r run) previousDigest(authored sequenced) (string, error) {
	for evidence, err := range r.request.Store.Scan(authored.ID) {
		if err != nil {
			return "", err
		}
		if !evidence.Comparable(r.request.Procedure, authored.Path) {
			continue
		}
		if digest := evidence.Step.Identities.Digest; digest != "" {
			return digest, nil
		}
	}
	return "", nil
}
