// Package run is the engine: what happens between `hyper run <procedure>` and
// the Journal entry it leaves behind (§6, issue #136).
//
// It takes a loaded repository, a Store handle, the reads the process supplies
// and the Capabilities' performers, and answers a value the CLI renders. **It
// reaches no process fact of its own**: no clock, no randomness, no environment
// variable. Every one of them is threaded through Request, which is what makes
// a Run's every path in the Store a checked-in constant rather than a value
// normalised out of a golden (§8, ADR-0047), and run_test.go holds it over this
// package's own source.
//
// That is a claim about **this package** and not about the tool beneath it, and
// the difference is worth stating rather than leaving to be discovered. Two
// packages under it start a git subprocess with the process's own environment,
// deliberately: the git hyper shells out to is the same git that resolves the
// credential a checkout left behind (§7, §11), and internal/git holds the one
// rule that keeps such a subprocess acting on the repository the caller named.
// So the engine performs no read of its own; what it reaches through does, and
// where it reaches is the repository root it was handed.
//
// It renders nothing either. What a Step table looks like, what the terminal
// line says and which exit code an outcome maps onto are §8's and §9's, and
// they live in internal/cli beside every other surface. What is here is what
// happened; what is said about it is one package up (ADR-0026).
//
// **The order is §6's fixed order**, and no Step starts until all of it has
// happened: the pin gate, the lock and the Store's location, all three the
// CLI's and all three declining before a Run is identified at all; then
// `run.json`; then the Store schema test, `check` re-run in full, the
// credential pass and the Secret sink, each of which declines into the entry
// that already exists; then Step 1. Perform states the order and gates.go
// states the four in the middle (issues #137, #138).
//
// The lock is the CLI's for the reason the Store's location is: it is a lock on
// the Store, a Run that cannot take it never opens one, and a Run with no Store
// has no entry to decline into. Which of the two locks it takes is this
// package's, though — it is read off the Kinds and nothing else, which is
// lock.go.
package run

import (
	"fmt"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Request is everything a Run is performed from.
//
// The four process reads are functions rather than resolved values, which is
// cli.Process's own rule one layer down: a read a Run never makes is a read
// that never happens, and a Run of one `read` Step starts no child and dials
// once.
type Request struct {
	// Repository is the load every name in the Run resolves against, and
	// RepoRoot is where it was read from — the second being what the code
	// branch's revisions are read out of.
	Repository repository.Loaded
	RepoRoot   string
	// Store is the record, already located. Locating it is the caller's,
	// because a Run that cannot find one Refuses `store-absent` before it
	// has an id to refuse under (§6, §7).
	Store *store.Store
	// Procedure is the top-level Procedure's name, already resolved against
	// the repository: a positional matching nothing is a usage error and
	// never reaches here (§9, ADR-0060).
	Procedure string
	// Trigger is what caused this Run and where it happened, already read
	// off the environment. It is filled rather than derived here for the
	// reason this package reads no environment variable at all — and
	// nothing in the engine reads it back, recording where a Run happened
	// being a fact about the occasion rather than an authority axis (§5,
	// §7).
	Trigger store.Trigger
	// DryRun says this Run is a rehearsal. It is carried into the entry on
	// every Run, false included (§7, ADR-0001).
	DryRun bool
	// Version is the binary's, and it is Provenance's `hyper_version`. It is
	// always a release string: the pin gate refuses any binary whose version
	// differs from the repository's in either direction, so there is no
	// development form to write (§7, §11).
	Version string
	// Now is the clock. Every instant the Run records comes through it, so a
	// fixture's entry is reproducible and the dates a golden holds are the
	// case's own.
	Now func() time.Time
	// Mint mints the Run id at the instant it is handed. It is threaded
	// rather than called for the reason the clock is, and one more: a Run id
	// lands on the terminal line, in the `outcome` row, in `run.json` and in
	// every Store path a Run writes, so an id minted here would make every
	// golden of a Run unassertable (§8, ADR-0047).
	Mint func(now time.Time) store.RunID
	// LookupEnv is the environment, read. It is threaded for the reason the
	// clock is — the engine reaches no process fact of its own — and it is
	// reached exactly once per credential slot the Run's bindings require,
	// at the credential pass §6 puts before Step 1 (§6, ADR-0007).
	//
	// It answers a value **and whether the variable is set at all**, which
	// is os.LookupEnv's shape rather than os.Getenv's: presence is the whole
	// of what the gate asks, and a variable set to the empty string is
	// present (§9).
	LookupEnv func(string) (string, bool)
	// Environ is the environment read whole, and it is reached for exactly
	// one thing: a `shell` Operation's child inherits it, less every
	// credential-slot variable in the repository (§3, §11, issue #142).
	//
	// It is a second read of one subject rather than a widening of the
	// first. LookupEnv answers *what does this name hold*, which is the
	// whole of what the credential pass asks; composing a child's
	// environment is a subtraction, and a set of names nothing enumerates is
	// not a set anything can subtract from.
	Environ func() []string
	// SecretSink is the path `--secret-out` named, and "" where the
	// invocation supplied none. It is the **occasion's** supply rather than
	// the environment's or the artefacts', which is why it arrives here
	// beside DryRun rather than being read off anything (§6, §9, ADR-0008).
	//
	// The engine reads its presence and never its value: what a Run does
	// with a sink is the milestone that writes one's, and #133 flags the ADR
	// that has to state the file's format first.
	SecretSink string
	// Dial and Exec are the two Capabilities' performers. Neither is reached
	// for: internal/capability is handed one, so a case exercises a real
	// handshake against a server standing in the test process and a real
	// child against a script a fixture checked in (§5, issues #133, #142).
	Dial capability.Dial
	Exec capability.Exec
	// Narrator is where progress goes as it happens. It may be nil, which is
	// a Run nobody is watching.
	Narrator Narrator
}

// Narrator is a Run's progress as it happens, and it is two events because §9's
// narration is two lines: the Run naming itself before its first Step, and one
// line per Step boundary, in both modes, always on.
//
// It is an interface rather than a stream this package writes to, because what
// those lines *say* is a rendering and renderings live in internal/cli. The
// engine reports the boundary; the surface writes the words. It carries no
// machine contract and has no `--json` variant, so nothing here derives from it
// and nothing reads it back.
type Narrator interface {
	// Began is the Run naming itself, before its first Step. It is the one
	// place a Run's identity reaches an operator whose process is killed
	// outright, which is why it is narrated rather than left to the
	// terminal line (§9).
	Began(run store.RunID)
	// Reached is one Step boundary: the Step's position, how many the Run
	// holds, and its authored id.
	Reached(position, of int, id string)
}

// Answer is what a Run did, in the shape the CLI renders (§8).
//
// It carries no exit code and no rendered line. §12 maps an outcome onto a
// code and §8 states what the line says; a value that carried either would be a
// second representation of a fact one layer up, which is the rule the Store
// itself is written under (§7).
type Answer struct {
	// Run is the Run's id, and Identified says an entry was written under
	// it. The pair is §8's *where no entry was written the id is absent, and
	// its absence is the fact*: two paths decline before a Run is identified
	// at all, and on both the terminal line names nothing to look up.
	Run        store.RunID
	Identified bool
	// Outcome is §12's triple, and every Run reaching this package ends in
	// exactly one member of it.
	Outcome store.Outcome
	// Steps is one entry per Step that reached a Disposition, in the Run's
	// written order — which is the Step table's order and the `<nnnn>` the
	// entry names its files by (§8, §12).
	Steps []Step
	// Provenance is the Run-wide half: what `run.json` carries, and what the
	// Run-wide `provenance` row renders (§7, §8, ADR-0043).
	Provenance store.RunProvenance
	// Refusal is the checks that declined, in the order `check` prints them,
	// and is non-empty exactly where Outcome is OutcomeRefused. It is one
	// array and never several Refusals: a Run has at most one Refusal ever,
	// the outcome being terminal, and the members are the checks one phase
	// evaluated together (§7, §8, ADR-0061).
	//
	// What the terminal line and the `outcome` row name is the **first**
	// member's `error_code`, derived where it is rendered and stored nowhere
	// (§7, §8).
	Refusal []Refusal
	// Fault is what stopped the Run, and nil where nothing did. It is
	// narration's — a failure carries no `error_code` (§9, §12) — and it is
	// the whole of what the surface has to say about a `failed` Run beyond
	// the outcome itself.
	Fault error
	// Unbuilt is what the Run reached that this binary does not implement,
	// one line each. It is **not a Refusal**: it carries no `error_code`, it
	// writes no Journal entry, it opens no stream, and the milestone that
	// builds the thing deletes its line rather than reclassifying it. A name
	// in §9's tree is a name the spec fixes and not a claim that the binary
	// implements it yet, which is the precedent internal/cli/tree.go already
	// states in as many words.
	Unbuilt []string
}

// Step is one Step of a Run as a surface reads it back.
type Step struct {
	// Position is the Step's place in the Run's written order, the first
	// Step 1.
	Position int
	// ID is the Step's authored `id:`, Kind its Operation's declared Kind
	// held rather than resolved, and Disposition what became of it (§7).
	ID          string
	Kind        store.Kind
	Disposition store.Disposition
	// Path is the invocation chain this Step was reached through, and empty
	// on a top-level Step. It is the same member the Step file carries, and
	// it is what the Step table renders a nested Step under (§7, §8).
	Path string
	// Records is the size of the identity set — what the Step concluded
	// about, and never the versions it wrote (ADR-0030) — and Concluded says
	// a set exists at all. Three of §12's seven Dispositions conclude about
	// nothing, and the dash that renders for them is what tells them from
	// the *ran* Step whose set is written empty (§8).
	Records   int
	Concluded bool
	// Expanded is how many Record identities the Step's calls **reached**,
	// and it is written **only where the Step stopped short of them** — a
	// `read` Expansion that drained and then halted (§6). It is what
	// `n of m` is read against: `n` is Records and `m` is this, and the
	// Records between them are the ones unaccounted for (§7, §8, issue
	// #140). A Step carrying no `over:` resolved no selector and makes its
	// call under a set of one, so a halted one is `0 of 1`.
	//
	// It counts Records and never the Expansion's members, which are the
	// same number only where an Operation projects one Record per member.
	// A `series` response whose tenth member's identity path did not resolve
	// is one member of an Expansion and ten Records reached, and the column
	// reads `9 of 10` — the entry says expanded to one beside it,
	// `expanded_to` being what the *selector* resolved to and this being
	// what the answers held (issue #144).
	//
	// Zero is *nothing stood short*, which is every Step that accounted for
	// everything it reached. It is written from the drain rather than
	// derived from Records, because *unaccounted for* is a fact about
	// Records that were never concluded about and the two counts are equal
	// for other reasons too — an identity two members resolve to is one
	// member of the set and two Records reached, which is a collision §6
	// halts on rather than a Step that stopped short (ADR-0070, ADR-0072).
	//
	// It is a count and never the names. Which Records are unaccounted for
	// is `expanded_to`'s and nowhere else, and a Step value carrying them
	// would be the second place a surface could read them from (§7, §8).
	Expanded int
	// Provenance is the Step's half: what the Step file carries, and what
	// that Step's `provenance` row renders (§7, ADR-0043).
	Provenance store.StepProvenance
}

// Perform runs the Procedure and answers what it did.
//
// **The order is §6's, and it is the whole of this function.** What the
// milestone has not built declines first, before anything is minted; then the
// code branch is read, because a Run that cannot name a revision has no
// Provenance to write and nothing to write it on; then the Run is identified
// and `run.json` goes down; then the Steps, in written order; then
// `outcome.json`; then the push.
//
// It answers rather than returning an error, because a Run that stopped is
// still a Run that happened: what it did before it stopped lives in its Records
// and its Dispositions rather than in its outcome (§6), and every one of them
// is on the Answer.
func Perform(request Request) Answer {
	loaded := request.Repository
	procedure, resolved := loaded.Procedure(request.Procedure)
	if !resolved {
		// Unreachable from the CLI, which resolves the positional
		// against the same namespace before it locates the Store (§9).
		// It is answered rather than assumed so that a second caller —
		// the MCP surface, milestone 11 — cannot reach a Run of a
		// Procedure that is not there.
		return failedBeforeTheRun(fmt.Errorf("no Procedure named %s", request.Procedure))
	}
	// The Run's Steps: the top-level Procedure's and every nested
	// invocation's, flattened into the one written order. **A Procedure
	// invoking another does not start a second Run** (§6, sequence.go).
	walked := flatten(loaded, request.Procedure)
	steps := walked.Steps
	if walked.Cycle != "" {
		// A Procedure that invokes one it is already inside of. §6 says
		// a cycle is rejected before the first Step, and §4 states no
		// code to reject it under — so it stops here, as a fault rather
		// than a Refusal: a Refusal is `hyper` declining and has a check
		// to name, and there is no check (§12, ADR-0002).
		return failedBeforeTheRun(fmt.Errorf(
			"the invocation graph of %s reaches %s, which is already invoking it — a cycle, and no Run performs one",
			request.Procedure, walked.Cycle))
	}

	// What this binary does not implement, restated here as this call's own
	// precondition. The caller has already asked — NotBuilt is what decides
	// the order, and it is asked before the Store is even located — and it
	// is asked again because there is one door into the engine and a
	// two-call contract is a contract that can be got wrong: without it the
	// engine would perform the `read` path's semantics against a Step that
	// is not a `read`, which is a binary doing something undefined.
	if unbuilt := NotBuilt(loaded, request.Procedure); len(unbuilt) > 0 {
		return Answer{Unbuilt: unbuilt}
	}

	// The code branch, read once over the artefacts the Run read. It is here
	// rather than beside the first Step that needs it because `run.json`
	// carries the Run-wide half, and `run.json` is written before Step 1
	// (§6, §7).
	facts, err := revision.Read(request.RepoRoot, artefactsRead(loaded, walked))
	if err != nil {
		return failedBeforeTheRun(err)
	}
	provenance := store.RunProvenance{
		HyperVersion:      request.Version,
		ProcedureRevision: revision.Blob(procedure.Bytes),
		RepoRevision:      facts.Head,
		RepoDirty:         facts.Dirty,
	}

	started := request.Now()
	inFlight := run{
		request:    request,
		id:         request.Mint(started),
		provenance: provenance,
		started:    started,
		acted:      acted{},
	}
	inFlight.entry = store.JournalEntry{Run: inFlight.id, Started: started}
	// What a `shell` Step's child inherits, composed once and before the
	// first of them runs. The withheld set is the **repository's** and not
	// this Run's — every credential-slot variable any Target declaration
	// names, whether or not a Step reached it — so it is decided offline and
	// does not turn on which Steps a Run walked (§3, §11, gates.go).
	inFlight.environment = capability.Inherited(request.Environ(), credentialVariables(loaded))
	answer := Answer{Run: inFlight.id, Identified: true, Outcome: store.OutcomeCompleted, Provenance: provenance}

	narrator := watching(request.Narrator)
	narrator.Began(inFlight.id)

	if err := request.Store.Append([]store.Write{{
		Path: inFlight.entry.RunPath(),
		Content: store.RunFile{
			Run:        inFlight.id,
			Procedure:  request.Procedure,
			Trigger:    request.Trigger,
			StartedAt:  started,
			DryRun:     request.DryRun,
			Provenance: provenance,
		}.Encode(),
	}}, "Begin run "+inFlight.id.String()); err != nil {
		// Nothing was written, so the entry the terminal line would name
		// is not there: the Run is identified by its id and by nothing on
		// the branch, which is what Identified says (§8).
		answer.Identified = false
		return failed(answer, err)
	}

	// The gates §6 states between `run.json` and Step 1, in its order: the
	// Store schema test over the files the Run will read, `check` re-run in
	// full, the credential pass, and the Secret sink. Each declines into the
	// entry that now exists, which is why they are here and not beside the
	// two the CLI runs before a Run is identified at all (issue #137).
	held, declined, err := inFlight.gates(steps)
	if err != nil {
		return inFlight.closed(failed(answer, err))
	}
	if len(declined) > 0 {
		return inFlight.closed(refused(answer, declined))
	}
	inFlight.credentials = held

	for position, step := range steps {
		narrator.Reached(position+1, len(steps), named(step))

		performed, declined, err := inFlight.perform(position+1, step)
		if performed.Position != 0 {
			answer.Steps = append(answer.Steps, performed)
		}
		if err == nil && len(declined) == 0 {
			continue
		}

		// The Run ends here, and the Steps after this one are *never
		// reached* — a Disposition each, no file for any of them, and a
		// row on the page all the same (§7, §12, neverReached below).
		// Both ways out share it because both are the Run ending: a
		// fault is the world resisting, and a guardrail declining at
		// this Step's Expansion or its condition is the one Refusal a
		// Run reaches past Step 1. Either way it is terminal — the
		// Steps before it ran and what they did stands, and this one
		// wrote no Record (§6, §8, ADR-0061).
		answer.Steps = append(answer.Steps, neverReached(loaded, steps, position+1)...)
		if err != nil {
			return inFlight.closed(failed(answer, err))
		}
		return inFlight.closed(refused(answer, declined))
	}

	return inFlight.closed(answer)
}

// neverReached is the Steps the Run ended before, one entry each, from the
// position after the one that ended it (§6, §12).
//
// **It is the one Disposition of the seven that writes no file at all**, and
// within a closed entry that absence is its whole representation: a forty-Step
// Procedure that halted at Step 3 would otherwise write thirty-seven files
// saying that nothing happened (§7). What it still has is a row on the page and
// a `step` row on the wire — the Step has a cell — and both render `–` for
// `RECORDS`, no set existing rather than a set with nothing in it (§8).
//
// It reaches every way a Run can end past Step 1, a halt and a Refusal alike:
// the value says *the Run ended before the Step* and neither of the two is more
// ended than the other. A Refusal **before** Step 1 leaves none of them, and
// that is §7's own sentence rather than an exception carved here — such an
// entry is `run.json` and `outcome.json` and nothing else, and the surface
// renders no Step table over a Run where no Step was reached (§7, ADR-0061).
//
// The Kind is the Operation's, read off the Manifest exactly as a Step that ran
// reads it: a Step that never ran still binds what it binds, and the column
// says what this Run would have done (ADR-0025).
func neverReached(loaded repository.Loaded, steps []sequenced, from int) []Step {
	unreached := make([]Step, 0, len(steps)-from)
	for position, step := range steps[from:] {
		unreached = append(unreached, Step{
			Position:    from + position + 1,
			ID:          step.ID,
			Path:        step.Path,
			Kind:        store.Kind(kindOf(loaded, step)),
			Disposition: store.DispositionNeverReached,
		})
	}
	return unreached
}

// run is one Run in flight: what the engine holds between `run.json` and
// `outcome.json`, and what every Step of it is performed against.
//
// The Run's own start instant is a member because it is the instant the whole
// Run resolves against: `run.json` carries it, the entry's date partition is
// the UTC date of it, and it is what a certificate's remaining life is counted
// from — so nothing a later Step does moves what an earlier one recorded
// (ADR-0034). The clock is read again for each Step's own `started_at` and
// `ended_at`, those being facts about the Step rather than about the Run.
type run struct {
	request    Request
	id         store.RunID
	entry      store.JournalEntry
	provenance store.RunProvenance
	started    time.Time
	// credentials is what the credential pass resolved, by Target and then
	// by the scheme's slot. It is a member because §6 says the credentials
	// of every Target the Run may bind are resolved **once**: a Step reading
	// the environment again would be a Run whose second call could send a
	// credential its own gate never saw (§6, ADR-0007).
	credentials credentials
	// environment is what a `shell` Step's child inherits, composed once
	// before Step 1 for the reason the credentials above are resolved once:
	// what a child may read is decided by the repository and the invocation
	// rather than by which Step happens to be running, and a second
	// composition is where the day comes that two children of one Run
	// disagree about what was withheld (§3, §11).
	environment capability.Environment
	// acted is what each Step of this Run acted on, filled at that Step's
	// turn and read by the `when:` conditions and the `{step:, path:}`
	// references of the Steps after it. It is a mapping rather than a value
	// on the Run because every method here is a reader of one run value: the
	// Steps write into what the Run holds and never into the Run
	// (condition.go).
	acted acted
}

// closed writes `outcome.json` and publishes, and answers whatever the Run had
// already concluded about itself.
//
// The two acts are one function because they are the Run's end whichever way it
// reached it: a Run halted by a Step's error closes its own entry `failed`
// exactly as a completed one closes it `completed`, and both send what stands
// locally. A Run that could not write its own outcome is `failed` and says so,
// which is the one place this can change the answer it was handed.
//
// The push is here rather than after each Step because this milestone's Run is
// read-only, and **a read-only Run's pushes batch to its end** (§7, ADR-0006).
// The Store carries no uncommitted local state at any moment either way — every
// write is already a commit — so what batching decides is when the commits
// leave and never whether they exist. Pushing after every effectful Step is
// milestone 6's.
//
// A push the remote moved under three times running is ErrPushExhausted, and it
// arrives here as any other fault does: the Run is `failed`, and what it wrote
// stands on the local branch and goes out with the next Run that syncs (§7,
// ADR-0076). That it exits 75 rather than 1 is the surface's, one package up —
// this one holds no exit code and maps none (§12, ADR-0026, issue #138).
func (r run) closed(reached Answer) Answer {
	ended := r.request.Now()
	if err := r.request.Store.Append([]store.Write{{
		Path: r.entry.OutcomePath(),
		Content: store.OutcomeFile{
			Outcome: reached.Outcome,
			EndedAt: ended,
			Refusal: storedRefusal(reached.Refusal),
		}.Encode(),
	}}, "End run "+r.id.String()); err != nil {
		return failed(reached, err)
	}
	if err := r.request.Store.Publish(); err != nil {
		return failed(reached, err)
	}
	return reached
}

// failed is the Answer a Run that the world resisted carries: the triple's
// member, and the fault as narration. It keeps the first fault where one is
// already held — what stopped the Run is what stopped it, and a failure while
// closing an already-failed Run is that Run still, reported by the fault a
// reader can act on.
func failed(answer Answer, err error) Answer {
	answer.Outcome = store.OutcomeFailed
	if answer.Fault == nil {
		answer.Fault = err
	}
	return answer
}

// refused is the Answer a Run a guardrail declined carries: the triple's
// member, and the array of checks that declined it.
//
// It is the counterpart of failed one line up and the two are deliberately not
// one function. A failure is the world resisting and carries a fault a reader
// acts on; a Refusal is `hyper` declining and carries a check it can name, and
// a value that could hold both would be a Run claiming an `error_code` for
// something nothing checked (§9, §12).
func refused(answer Answer, declined []Refusal) Answer {
	answer.Outcome = store.OutcomeRefused
	answer.Refusal = declined
	return answer
}

// failedBeforeTheRun is a Run that stopped before it was identified: no id, no
// entry, and nothing to look up. §8 states that the terminal line names nothing
// there, and that the `outcome` row is emitted regardless — what is missing is
// the row's `run_id` and never the row.
func failedBeforeTheRun(err error) Answer {
	return Answer{Outcome: store.OutcomeFailed, Fault: err}
}

// watching is the Narrator a Run reports to, or a silent one where nobody
// supplied one. It is one nil check rather than one per event: an engine
// guarding its own narration at every call site is an engine where the day
// comes that one site forgets.
func watching(narrator Narrator) Narrator {
	if narrator == nil {
		return silent{}
	}
	return narrator
}

// silent is the Narrator of a Run nobody is watching.
type silent struct{}

func (silent) Began(store.RunID)        {}
func (silent) Reached(int, int, string) {}
