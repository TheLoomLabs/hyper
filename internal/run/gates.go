package run

import (
	"fmt"
	"slices"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/presence"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/verify"
)

// Everything between `run.json` and Step 1 (§6, issue #137).
//
// §6 fixes the order and no Step starts until all of it has happened: the Store
// files the Run must read are held to this binary's schema versions, `check` is
// re-run in full with nothing skipped, the credentials of every Target the Run
// may bind are resolved once, and the Procedure is tested for a Step whose
// Operation declares secret output. Then Step 1.
//
// Each of them declines into the entry that already exists, which is why they
// sit here rather than beside the gates the CLI runs before a Run is identified
// at all: the pin gate and the Store's own location decline with no id to
// decline under, and everything below writes `outcome.json` under one (§7, §8).
//
// **Three of the four are scoped by one sentence**: the (Definition, Target)
// pairs the Procedure makes. The schema test walks the Record heads under
// those pairs, the credential pass resolves the slots those pairs require, and
// §4's own slot-coverage check quantifies over the same set — so a Target
// serving two Providers does not oblige a Run to read a series or hold a
// credential no Step of it could reach (§6).

// The three codes §9 contributes, alike in being neither the environment's nor
// the artefacts', all three checked before a Run's first Step and all three
// reported exhaustively rather than at the first (§12, ADR-0007).
//
// The first two are the **occasion's** supply. The third is this **binary's**,
// and that is not a widening of the group: it is checked here, at this moment,
// with the other two, and it is what §9 says about a sink today rather than a
// standing fact about invocations (ADR-0146).
//
// They are spelled here because this is where the checks that carry them are.
// internal/store reports the schema condition and does not name a code, and
// §4's thirty-two arrive already carrying theirs.
//
// The first two are one gate reading one variable, and they are **two codes
// rather than one message**, on the set's own test: a reader handed
// `credential-absent` for a variable that is exported checks the export, finds
// it, and is out of moves. The remedies differ and §8 holds one remedy per code
// (refusal.go), so a single code could only ever have offered the wrong one to
// whichever half it was not written for (§12, ADR-0145).
const (
	// CodeCredentialAbsent is a credential a Target declaration names and
	// the environment does not hold.
	CodeCredentialAbsent = "credential-absent"
	// CodeCredentialEmpty is a credential a Target declaration names that
	// the environment holds and sets to the empty string — a variable an
	// upstream produced rather than one nobody exported.
	CodeCredentialEmpty = "credential-empty"
	// CodeSecretSinkUnwritten is a Run reaching a Step whose Operation
	// declares secret output, this hyper having no Secret sink to put the
	// value in.
	//
	// It is one code rather than two because the supply that is missing is
	// the binary's rather than the invocation's: a sink named and a sink
	// withheld are the same Run here, and §8 holds one remedy per code
	// (refusal.go, issue #266).
	CodeSecretSinkUnwritten = "secret-sink-unwritten"
)

// Refusal is one check that declined a Run: everything its Store counterpart
// carries, plus the two members a row adds where it cites a Step (§7, §8).
//
// The two are on the wire and never in the entry, and that is §7's own split
// rather than an omission: `operation` and `target` are what the Step the check
// cites was bound to, and the entry already holds them on the Step's own file
// wherever a Step file exists. A Refusal before Step 1 writes none, so a
// consumer reading the row is handed the binding and a consumer reading the
// entry reads the artefact that made it.
type Refusal struct {
	store.RefusalMember
	// Operation and Target are the binding of the Step this check cites,
	// and are empty on a Refusal that cites no Step.
	Operation, Target string
	// Narrowed is the second remediation §8's `EDIT ONE OF` table renders
	// beside a Bound, and nil on every check that has no second edit to
	// offer — which is every one of them but `bound-exceeded`, and that one
	// only where its selector carries an operand a rung can be proposed for
	// (§8, narrow.go).
	//
	// It rides here rather than in the entry for the reason `resolved` rides
	// on a row: it is derived, it is a hypothetical, and a Store holding one
	// would be a Store holding a count no Run ever acted on (§7, ADR-0034).
	Narrowed *Narrowing
}

// storedRefusal is the array as `outcome.json` holds it: the Store's own shape,
// with the two members that ride on the wire alone left off (§7).
func storedRefusal(refusals []Refusal) []store.RefusalMember {
	held := make([]store.RefusalMember, len(refusals))
	for i, refusal := range refusals {
		held[i] = refusal.RefusalMember
	}
	return held
}

// gates runs §6's order between `run.json` and Step 1, and answers what
// declined the Run.
//
// It stops at the **first gate that declines** rather than collecting across
// all four. A Refusal is one phase's finding: §7 fixes that the array has more
// than one member only where the phase evaluates many checks together, and the
// two that do — `check` re-run, and the credential pass — each report every
// problem they found. Running the credential pass over a repository `check` has
// just refused would report absent variables for bindings that do not resolve.
//
// It answers the credential pass's own result beside the Refusals rather than
// writing it onto the Run, so that every method on a Run in flight stays a
// reader of one. What §6 requires is that the environment is read **once**,
// before Step 1; where that answer is then kept is Perform's, and Perform is
// where the Run in flight is assembled.
func (r run) gates(steps []sequenced) (credentials, []Refusal, error) {
	loaded := r.request.Repository

	unreadable, unsupported, err := r.request.Store.Readable(pairsOf(steps))
	if err != nil {
		return nil, nil, err
	}
	if unsupported {
		return nil, []Refusal{schemaRefusal(unreadable)}, nil
	}

	if declined := staticRefusals(loaded); len(declined) > 0 {
		return nil, declined, nil
	}

	resolved, declined := resolveCredentials(loaded, steps, r.request.LookupEnv)
	if len(declined) > 0 {
		return nil, declined, nil
	}

	return resolved, sinkRefusals(loaded, steps), nil
}

// pairsOf is the (Definition, Target) pairs the Procedure makes, each once and
// in the Steps' own order. It is the scope §6 quantifies both the schema test
// and the credential pass over, read in one place so that the two gates cannot
// come to mean different sets.
//
// **Each once** is what keeps a Procedure of ten Steps against one Target from
// reading one series ten times and reporting one absent variable ten times: the
// pairs are what the Run binds, not what it does.
//
// A nested invocation contributes nothing of its own and needs to contribute
// nothing: it binds no Definition and no Target, and its Steps are Steps of the
// one Run, already flattened into the sequence this walks (issue #141). So a
// Target a nested Procedure alone binds is one this Run's credential pass
// resolves, and a Record head under it one the schema test reads.
func pairsOf(steps []sequenced) []store.Pair {
	seen := map[store.Pair]bool{}
	pairs := make([]store.Pair, 0, len(steps))
	for _, step := range steps {
		pair := store.Pair{Target: step.Target, Definition: step.Definition}
		if seen[pair] {
			continue
		}
		seen[pair] = true
		pairs = append(pairs, pair)
	}
	return pairs
}

// schemaRefusal is the Refusal a Store file above this binary's ceiling earns.
//
// It is the one member of the closed set whose subject is **evidence rather
// than an artefact**, so it carries a file and neither a line nor a field: §8
// renders no caret over a Store path, editing evidence being the one act that
// surface must not invite (§7, §8, ADR-0011, ADR-0028).
//
// The remedy is a different binary and the message names it. A check knows its
// own remedy, so saying it states a fact rather than editorialising.
func schemaRefusal(unreadable store.Unreadable) Refusal {
	return Refusal{RefusalMember: store.RefusalMember{
		ErrorCode: store.SchemaUnsupportedCode,
		File:      unreadable.File,
		Message:   unreadable.SchemaUnsupported.Error() + " — install a hyper that reads it",
	}}
}

// staticRefusals is `check` re-run in full with nothing skipped, as a Refusal.
//
// This is how **all thirty-two of §4's static codes reach a Run**, which is
// most of the closed `error_code` set rather than a corner of it (ADR-0061).
// Nothing is implemented for them here: milestone 1 built every one, and what
// this adds is the path — a Run against a working tree edited since `check`
// last passed declines with whatever code the edit earns.
//
// Nothing is skipped and nothing is narrowed. `check [path...]` filters what it
// *reports* after checking the repository whole, because every rule compares
// one artefact against another (§9); a Run has no paths to filter by and would
// have no reason to.
//
// The order is `check`'s own — by file path, then by line — which is what §7
// fixes for the array and what puts the first member's code on the terminal
// line (§7, §8).
func staticRefusals(loaded repository.Loaded) []Refusal {
	problems := verify.Repository(loaded)
	if len(problems) == 0 {
		return nil
	}
	problem.Sort(problems)

	declined := make([]Refusal, 0, len(problems))
	for _, found := range problems {
		declined = append(declined, Refusal{RefusalMember: store.RefusalMember{
			ErrorCode: found.ErrorCode,
			File:      found.File,
			Line:      found.Line,
			Field:     found.Field,
			Message:   found.Message,
		}})
	}
	return declined
}

// credentials is every credential the Run's bindings require, resolved: by
// Target name, then by the scheme's slot.
//
// It is the values and not their presence, and it exists so that the
// environment is read **once** — §6 says the credentials of every Target the
// Run may bind are resolved once, before the first Step, and a Step reading a
// variable again would be a Run whose second call could send a credential its
// first gate never saw.
type credentials map[string]map[string]string

// credentialSlot is one slot of one Target: what the credential pass resolves
// at, and therefore what it asks the environment about at most once. It is the
// Target and not the pair, because a slot belongs to the Target declaration
// that names its variable and two Definitions reaching it reach one variable.
type credentialSlot struct{ target, slot string }

// resolveCredentials resolves every slot the Run's bindings require and answers
// which of them the environment did not fill.
//
// **Every unfilled slot is reported at once**, one member of the array each,
// because the pass resolves them all in one go and knows them all at once.
// Reporting the first would send an operator round the loop once per variable,
// each `export` earning another `77` (§6, §9, ADR-0007).
//
// **The question is three-valued and the value is still not judged**: §12's
// credential presence, read once per slot, with two of its three members
// declining — `absent` earning `credential-absent` and `empty` earning
// `credential-empty`. What is read of a value is which of the three it falls
// under and nothing else — no length beyond zero, no shape, no plausibility —
// the endpoint owning whether a credential works and `hyper` owning only whether
// one was supplied (§12, ADR-0007, ADR-0145).
//
// The empty half is here rather than at the wire because an empty string
// composes into a header that is present and blank, and the `401` it earns
// reads as the world resisting when what happened is that the invocation was
// never ready. A variable exported to nothing is not a typo anyone makes twice;
// it is what an upstream produces — a `$(op read …)` that returned nothing, a
// CI secret never set on the fork — and on an effectful Procedure the Steps
// before the first authenticated one have already run by the time the `401`
// lands (§9, ADR-0145).
//
// `targets` reports the same three under the same three names, which is the
// invariant issue #112 bought: the gate and `targets` ask one question, so
// `set` there means this pass will proceed (§9, targets.go).
//
// Two shapes are passed over rather than reported, and both are `check`'s,
// which has already run: a Target whose slots do not cover the bound Provider's
// scheme is `manifest-inconsistent` (§4), and a slot naming no variable is
// `credential-slot-malformed` (§4). A second opinion here would put two rows
// on the page for one fault.
func resolveCredentials(loaded repository.Loaded, steps []sequenced, lookupEnv func(string) (string, bool)) (credentials, []Refusal) {
	resolved := credentials{}
	// asked is the slots the pass has already put to the environment. Two
	// Definitions naming one Target under one scheme require one slot
	// between them, so without this an absent variable would earn a member
	// of the array per Definition — and §7's array is every absent *slot*,
	// not every Step that wanted one.
	asked := map[credentialSlot]bool{}
	var declined []Refusal

	for _, pair := range pairsOf(steps) {
		// Which slots the binding requires, and where the declaration
		// that names their variables was read from, is the load's own
		// answer: `project` writes the same slots' variables into the
		// generated workflow's `env:` block, and one walk is what keeps
		// the job's block and this gate quantified over one set
		// (repository.Loaded.CredentialSlots, §10).
		file, slots := loaded.CredentialSlots(pair)

		for _, named := range slots {
			if named.Env == "" || asked[credentialSlot{pair.Target, named.Slot}] {
				continue
			}
			asked[credentialSlot{pair.Target, named.Slot}] = true

			value, present := lookupEnv(named.Env)
			if code, message := credentialRefusal(presence.Of(value, present), named.Env); code != "" {
				declined = append(declined, Refusal{RefusalMember: store.RefusalMember{
					ErrorCode: code,
					File:      file,
					Line:      named.Line,
					Field:     "auth." + named.Slot,
					Message:   message,
				}})
				continue
			}
			if resolved[pair.Target] == nil {
				resolved[pair.Target] = map[string]string{}
			}
			resolved[pair.Target][named.Slot] = value
		}
	}

	sortRefusals(declined)
	return resolved, declined
}

// credentialRefusal is what one slot's Presence costs the Run: the code it
// Refuses under and the message that names its variable, or no code at all where
// the environment filled it.
//
// **The reading is not taken here.** §12's closed three is internal/presence's,
// and so is the step that decides which member a variable falls under: what this
// adds is the two Refusals two of those three earn. A second reading here would be the gate and `targets`'s column
// answering *what did the environment do with this name* separately, which is
// where the two come to disagree — and the thing they would disagree about is
// whether a Run is about to Refuse (§9, ADR-0145).
//
// The messages name the variable and state what the environment did, and
// neither of them says what to do about it: the remedy is §8's, one per code,
// and a message carrying one would be the same fact rendered twice on one screen
// (§8, ADR-0026).
func credentialRefusal(held presence.Presence, name string) (code, message string) {
	switch held {
	case presence.Absent:
		return CodeCredentialAbsent, "the environment does not hold " + name
	case presence.Empty:
		return CodeCredentialEmpty, "the environment sets " + name + " to the empty string"
	}
	return "", ""
}

// withheldVariables is every environment variable the repository names — as a
// credential slot, or in a `withhold:` list — in no order anything reads: the set
// a `shell` Operation's child inherits the invoking environment **less** (§3,
// §11, issue #142, ADR-0144).
//
// It is every Target declaration in the repository rather than the ones this
// Run's Steps bind, which is the rule §11 states and not a widening for safety:
// decided that way the set is a fact about the tree a reviewer can read off it,
// where a set derived from the Steps a Run walked would differ between two Runs
// of one repository and would grow a hole the day a Procedure stopped binding a
// Target whose variable was still set. `withhold:` is unioned in on the same
// rule and for the same reason, and the two halves are one set here because the
// child inherits one environment: a name in both contributes once, and a
// declaration is not obliged to know which of the two a variable was.
//
// A slot naming no variable contributes nothing — that is
// `credential-slot-malformed`, which is `check`'s to report rather than a
// reader's to repeat (ADR-0064). Nothing here reads a value: `hyper` knows those
// names by position, which is the same knowledge that lets it suppress a
// credential rather than scan for one (ADR-0007) — and a withheld variable it
// never had a position for, which is why the key names one rather than
// describing it.
func withheldVariables(loaded repository.Loaded) []string {
	var named []string
	for _, declaration := range loaded.TargetDeclarations {
		facts := artefact.ReadTargetFacts(declaration)
		for _, slot := range facts.Credentials {
			if slot.Env != "" {
				named = append(named, slot.Env)
			}
		}
		for _, name := range facts.Withheld {
			if name != "" {
				named = append(named, name)
			}
		}
	}
	return named
}

// sortRefusals puts an array in the order §7 fixes for one: the order `check`
// prints in, by file path and then by line — which is what makes the first
// member, and so the code the terminal line and the `outcome` row name, the
// same on two runs of one command (§7, §8).
//
// The comparison is internal/problem's own rather than a second spelling of it
// here: a Refusal and a `check` over one repository list the same faults, and
// two orderings is where the day comes that they list them differently.
func sortRefusals(refusals []Refusal) {
	slices.SortStableFunc(refusals, func(a, b Refusal) int {
		return problem.Compare(a.coordinate(), b.coordinate())
	})
}

// coordinate is the Refusal's position as internal/problem states one, which is
// what orders the array. It carries no column: §12's problem row has one and
// §7's Refusal member does not, the column riding on `check`'s wire alone.
func (r Refusal) coordinate() problem.Problem {
	return problem.Problem{File: r.File, Line: r.Line, ErrorCode: r.ErrorCode}
}

// sinkRefusals is the Secret sink gate: where the Procedure reaches a Step
// whose Operation declares secret output, the Run Refuses, naming **every such
// Step at once** rather than the first (§6, §9, §12).
//
// **No sink is written by this hyper**, and that is what the gate reads. The
// sink is the only route by which a secret value ever leaves `hyper` (§9,
// ADR-0007), and a Run that reached such a Step would produce the value,
// suppress it into the Store as the constant marker, and discard it — a clean
// Run, a Record that reads correctly, and no secret. The format the file holds
// has yet to be decided and is ADR-shaped, so what this states in the meantime
// is the limit rather than the loss (issue #266, ADR-0146).
//
// So it is handed **no sink at all**: a path named and a path withheld are the
// same Run here, and reading the flag would be this gate offering a remedy that
// leads to another `77` — the loop the credential codes were split to prevent
// (§8, ADR-0145). The remedy is a different binary and §8 names it as one
// (refusal.go). When the format lands, the sink's presence becomes the gate's
// second operand again and `secret-sink-absent` returns to the closed set.
//
// It is stated at Run start rather than at the Step because its operand is
// already in hand: which reachable Steps declare secret output is a walk over
// reviewed text. Declining at the Step instead would run the Steps before it
// and never reach the tail — which under a Cadence is an effectful prefix
// repeated for as long as the clock fires, and is the second reason the
// combination is refused at `check` before it can arise (§4, ADR-0077).
//
// **It carries no Kind axis and no `--dry-run` exemption**, and the second is
// held by this function's own signature: it is handed no rehearsal marker, so
// there is nothing here to exempt on. A `read` declaring secret output is
// reached by a rehearsal and produces a secret with nowhere to go (§9).
//
// The `step` it carries is an **artefact coordinate and never an execution
// fact**: a Refusal before Step 1 writes no Step file, so the Step it names has
// no file in the entry at all (§7, ADR-0061).
//
// It reaches every Step the Run holds, a nested Procedure's included: the
// walk is over reviewed text and the invocation graph is static, so *which
// reachable Steps declare secret output* is answered at any depth (§6, issue
// #141). What each one cites is the file it was **authored** in and its
// position in that file's own `steps:`, a coordinate being an artefact's and
// never the Run's flattened order.
func sinkRefusals(loaded repository.Loaded, steps []sequenced) []Refusal {
	var declined []Refusal
	for position, step := range steps {
		operation, secret := secretOutputOf(loaded, step)
		if !secret {
			continue
		}
		declined = append(declined, Refusal{
			RefusalMember: store.RefusalMember{
				ErrorCode: CodeSecretSinkUnwritten,
				File:      step.Declared.Path,
				Line:      step.Line,
				Field:     fmt.Sprintf("steps[%d]", step.Index),
				Message: fmt.Sprintf("%s %s declares secret: output and this hyper does not write a Secret sink — the value would be produced and discarded",
					loaded.Definitions[step.Definition].ProviderName, step.Operation),
				Step:   position + 1,
				StepID: step.ID,
			},
			Operation: operation,
			Target:    step.Target,
		})
	}
	return declined
}

// secretOutputOf says whether the Step's Operation declares secret output, and
// names the Operation where it does.
//
// A binding that does not resolve declares nothing here, which is `check`'s to
// report and has already been reported: this gate runs after `check` re-ran.
func secretOutputOf(loaded repository.Loaded, step sequenced) (string, bool) {
	info, declared := loaded.Definitions[step.Definition]
	if !declared {
		return "", false
	}
	operation, declares := loaded.Providers[info.ProviderName].Operations[step.Operation]
	if !declares {
		return "", false
	}
	return step.Operation, len(operation.SecretFields) > 0
}
