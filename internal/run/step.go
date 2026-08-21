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
// What is here is the `read` path and no other Kind's. An effectful Step
// declines before Step 1 — run.go states that and why — so nothing below has to
// carry a Kind it cannot perform, and milestone 6 grows this file rather than
// working around it.

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
// **A member whose call faulted drains rather than halting there.** Every
// member is attempted, every Observation that succeeded is recorded, and the
// Run then halts with the rest of the results already on disk. drain.go states
// why that is not a preference (§6).
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

	// The calls, dispatched in Expansion order and bounded by the
	// Operation's declared `concurrency:` limit — which arrives effective,
	// so a Manifest that declared nothing runs its Expansion one member at
	// a time (§6, ADR-0045, drain.go).
	//
	// **Every member is attempted**, whatever any other member's call did,
	// and what stopped one is read back below in Expansion order.
	answers, faults := dispatch(bound.detail.ConcurrencyLimit, expanded.Members, func(resolving member) (conclusion, error) {
		return r.call(bound, authored, resolving)
	})

	// The versions the calls moved, written in Expansion order. A version
	// is written only where the bytes moved: an Operation returning what
	// the head version already holds mints nothing, and the canonical
	// encoding is what makes that an exact test (§7, ADR-0030).
	//
	// **A member that faulted is skipped and stops nothing.** What it wrote
	// is nothing — there is no Observation to record — and the Run halts on
	// it once the rest of the Expansion has been written down, which is the
	// drain (§6, drain.go). The fault the Run carries is the **first in
	// Expansion order** and not the first to arrive, which is what keeps
	// *which fault* out of the completion order's reach as well.
	names := make([]string, 0, len(expanded.Members))
	// What this Step **acted on**, for the conditions and the references of
	// the Steps after it: the fields each call concluded, whether or not the
	// version they would have written moved the bytes. A Record going
	// unchanged is not a Record going missing (§6, ADR-0030, condition.go).
	records := make([]store.Mapping, 0, len(expanded.Members))
	var halted error
	for at, fault := range faults {
		if fault != nil {
			if halted == nil {
				halted = fault
			}
			continue
		}
		answer := answers[at]
		names = append(names, answer.name)
		records = append(records, answer.fields)

		identity := store.Identity{Target: authored.Target, Definition: authored.Definition, Name: answer.name}
		version := store.RecordVersion{
			Metadata: store.Metadata{
				Identity:   identity,
				RecordType: store.RecordObservation,
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
			},
			Fields: answer.fields,
		}
		moved, err := mints(r.request.Store, version)
		if err != nil {
			return Step{}, nil, err
		}
		if moved {
			if err := r.request.Store.Append([]store.Write{{
				Path:    store.RecordPath(identity, r.id, position),
				Content: version.Encode(),
			}}, fmt.Sprintf("Record %s/%s/%s at run %s step %d", identity.Target, identity.Definition, identity.Name, r.id, position)); err != nil {
				return Step{}, nil, err
			}
		}
	}

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
	// Expanded is written on the drained Step alone. A Step that reached
	// the end of its Expansion accounted for all of it and has nothing for
	// a second number to say, which is what keeps `n of m` meaning
	// *unaccounted for* rather than *these two counts differ*.
	reached.Disposition = store.DispositionRan
	reached.Records, reached.Concluded = len(concluded), true
	if halted != nil {
		reached.Expanded = len(expanded.Members)
	}
	// What this Step acted on is held for the Steps after it at the moment
	// it reaches its Disposition, which is the moment §6 fixes: a Step's
	// Records are written as each call confirms, and all of it before the
	// next Step starts.
	r.acted[stepKey{authored.Namespace, authored.ID}] = records
	file.Disposition = reached.Disposition
	file.Identities = store.Concluded(names, previous)
	// The Step's file goes down whether or not the Expansion drained: a
	// Step that reached a Disposition wrote its file, and a halted Run
	// leaves what it did (§6, ADR-0011). The fault travels beside it and is
	// what makes the Run `failed`.
	if err := r.write(file); err != nil {
		return Step{}, nil, err
	}
	return reached, nil, halted
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
}

// call makes one member's call and projects the answer.
//
// **A `read` never halts on what came back.** Whatever the status, whatever the
// exit code, the response object §12 states is what the projection reads — so a
// host that answered nothing records an Observation whose status has gone quiet
// and a command that exited `1` records the code, rather than either stopping
// the Run (§6, ADR-0050). What halts it is the two things that are not an
// answer: the Operation's **deadline**, and the projection — in this milestone
// the identity path alone, the recorded fields being an absence a version
// simply does not carry and the rest of what a projection that does not resolve
// does being issue #144's.
//
// It is one member's and never the Step's, and the error it answers halts
// nothing on its own: what a fault here does to the members beside it is the
// drain, one caller up (§6, drain.go).
func (r run) call(bound binding, authored sequenced, resolving member) (conclusion, error) {
	if bound.operation.Identity == "" {
		return conclusion{}, fmt.Errorf("step %s binds %s %s, whose record: declares no identity, so hyper cannot say which Record a call would be holding — hyper check reports it",
			named(authored), bound.manifest.Name, authored.Operation)
	}

	declaration := artefact.OperationNode(bound.manifest.Root, authored.Operation)
	response, halted := r.answer(bound, authored, resolving, declaration)
	if halted != nil {
		return conclusion{}, halted
	}

	name, resolvedName := identityOf(bound.operation, resolving.Inputs, response)
	if !resolvedName {
		return conclusion{}, fmt.Errorf("step %s: the identity path %s did not resolve against what came back, so hyper cannot say which Record it is holding",
			named(authored), bound.operation.Identity)
	}
	return conclusion{name: name, fields: projected(bound.operation, projection.Read(declaration).Project(response))}, nil
}

// answer is what the member's Capability came back with, and the fault that
// halts the Run where one came back with nothing it could use.
//
// **The two Capabilities meet here and nowhere else above this line.** Which of
// them an Operation declares is the Manifest's one fact about its request (§3),
// and everything past this function — the identity, the projection, the version
// — reads a response object and never a socket or a process. What differs
// between the two arms is the reach rule and the wording of the deadline, and
// both differences are real: an `http` call is bounded by a grant this checks
// against, and a `shell` command is the one Capability whose reach no grant
// bounds (§13), its words being the ones a reviewer read in the Procedure.
//
// The error it answers is the **halting** one, worded for a reader, and nil
// where nothing halted. Everything else beside the object stays narration's and
// is dropped here: no member of the response object says what went wrong, that
// being the catch-all bucket ADR-0017 closed, and a `read` records a call that
// got no answer as the answer it is (§6, ADR-0050).
func (r run) answer(bound binding, authored sequenced, resolving member, declaration *yaml.Node) (capability.Object, error) {
	if bound.operation.IsShell {
		return r.ran(bound, authored, resolving)
	}
	return r.requested(bound, authored, resolving, declaration)
}

// requested is the `http` half: the reach resolved, the holes filled, the call
// made and bounded by the Operation's own `deadline:`.
func (r run) requested(bound binding, authored sequenced, resolving member, declaration *yaml.Node) (capability.Object, error) {
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

	ctx, cancel := capability.Deadline(context.Background(), bound.detail.DeadlineSeconds)
	defer cancel()

	// The instant handed to the call is the **Run's** start and not a fresh
	// read: it is what a certificate's remaining life is counted from, so
	// two Steps of one Run that reach one host record one `days_left`, and
	// nothing a later Step does moves what an earlier one recorded
	// (ADR-0034).
	response, err := built.Perform(ctx, r.request.Dial, r.started, r.credential(bound, authored.Target))

	// **A deadline reached on a `read` fails the Step** (§6). It is the one
	// error beside the response object this reads, and it is read because
	// it is the one an artefact declared: a refused connection, a name that
	// does not resolve and a handshake that failed are all *no response
	// arrived*, which a `read` records as the answer it is, and a deadline
	// is `hyper` stopping rather than the world answering nothing.
	//
	// The deadline is named as itself rather than as the transport's word
	// for it, and beside the host it was reached on — the two facts a
	// reader can act on, one an edit to the Manifest and one a look at the
	// far end. Which **member** drained is `expanded_to`'s and nowhere else
	// (§7, §8).
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("step %s: the Operation's deadline of %s was reached on %s and no response arrived",
			named(authored), bound.detail.Deadline, reach.Host)
	}
	return response, nil
}

// ran is the `shell` half: the argv exec'd, and the four-member object §12
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
func (r run) ran(bound binding, authored sequenced, resolving member) (capability.Object, error) {
	if len(resolving.Argv) == 0 {
		// Unreachable from a Run: `command-malformed` refuses an empty
		// `command:` at load and the Expansion refuses a shape it could
		// not read. It says so rather than exec'ing nothing, an argv
		// with no head being a call `hyper` cannot describe (ADR-0064).
		return nil, fmt.Errorf("step %s resolved no argv, and there is no executable to name — hyper check reports it", named(authored))
	}

	ctx, cancel := capability.Deadline(context.Background(), bound.detail.DeadlineSeconds)
	defer cancel()

	command := capability.Command{Argv: resolving.Argv}
	response, err := command.Perform(ctx, r.request.Exec, r.request.RepoRoot, r.environment)

	// **A deadline reached on a `read` fails the Step** (§6), and it is the
	// one error beside the object this reads. A command that could not be
	// started at all is *no answer* rather than a fault — the object is
	// `command` and nothing else, and a `read` records the attempt with its
	// `exit_code` gone quiet (§12, ADR-0050).
	//
	// The child's whole process group has been killed with SIGKILL and no
	// grace period by the time this line runs, so a command's own children
	// do not outlive the deadline that bounded it (§6, cli.Child). The
	// deadline is named as itself and beside the command it bounded — the
	// two facts a reader can act on, one an edit to the Manifest and one a
	// look at what the Procedure asked for.
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("step %s: the Operation's deadline of %s was reached and %s was killed",
			named(authored), bound.detail.Deadline, command.Text())
	}
	return response, nil
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
// inputs before the call, and a `$`-rooted path resolves against the response
// after it. Which of the two a Manifest declares is what decides whether an
// identity collision Refuses at Expansion or halts the Run (§6, ADR-0072).
func identityOf(operation artefact.OperationInfo, inputs map[string]schema.Scalar, response capability.Object) (string, bool) {
	declared := operation.Identity
	if declared == "" {
		return "", false
	}
	if !strings.HasPrefix(declared, "$") {
		filled, err := capability.Fill("identity:", declared, inputs)
		return filled, err == nil && filled != ""
	}

	value, resolved := projection.Resolve(declared, response)
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
// Two filters stand between the scan and the answer, and both are this
// consumer's rather than the walk's — internal/store's Scan matches on the
// authored id and on nothing else, and states that which entries a reading
// keeps is its own.
//
// **A rehearsal is filtered out.** An entry a dry-run wrote is evidence that a
// rehearsal happened and evidence of nothing else, and every consumer of
// Journal evidence filters it out (§7, ADR-0001).
//
// **So is another Procedure's entry.** An authored id is unique inside one
// Procedure and says nothing across two, so a `status` Step in `watch-status`
// and a `status` Step in `watch-many` are two Steps that would otherwise share
// a digest — and each would then read the other's set as its own, writing no
// members while the digest never moved. §7's rule is *the last Run in which
// that Step carried a set*, and a Step belongs to the Procedure that wrote it.
//
// **And so is another invocation chain's.** One Run holds the Steps of every
// Procedure it invokes, so two nested Procedures may each declare a `status`
// and both be Steps of one Run — told apart by the `path` their files carry
// beside that id, which is the same filter one Procedure over (§7, issue #141).
// Where one Procedure is invoked twice the two chains are one path
// (sequence.go), and the comparand is then the more recent of the two: §7
// matches a Step by what it was authored as, and those two Steps were authored
// as one.
func (r run) previousDigest(authored sequenced) (string, error) {
	for evidence, err := range r.request.Store.Scan(authored.ID) {
		if err != nil {
			return "", err
		}
		if evidence.Entry.RunFile.DryRun || evidence.Entry.RunFile.Procedure != r.request.Procedure {
			continue
		}
		if evidence.Step.Path != authored.Path {
			continue
		}
		if digest := evidence.Step.Identities.Digest; digest != "" {
			return digest, nil
		}
	}
	return "", nil
}
