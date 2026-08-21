package store

import (
	"slices"
	"time"
)

// The members each of the Step's two shapes always carries. A closing write
// carries three of the eleven: what a reaper knows without resolving the dead
// Run's revision, everything else being absent where it cannot (§7).
var (
	stepFileMembers = []string{
		"step", "id", "definition", "operation", "provider", "target",
		"kind", "disposition", "started_at", "ended_at", "provenance",
	}
	closedByMembers = []string{"step", "disposition", "ended_at"}
)

// Kind is what an Operation does: §12's closed three, declared per Operation in
// a Manifest and never inferred from the Operation's name (ADR-0025).
type Kind string

const (
	// KindRead observes and changes nothing. It writes Observations.
	KindRead Kind = "read"
	// KindMutate brings something into existence or changes something that
	// already stands. It writes Assets.
	KindMutate Kind = "mutate"
	// KindDestroy removes something. It writes Tombstones.
	KindDestroy Kind = "destroy"
)

// Disposition is what became of one Step: §12's closed seven, each with the
// wire spelling the Store writes.
//
// Six of the seven are borne by a Step file. DispositionNeverReached is read
// from the absence of one inside a closed entry — a forty-Step Procedure that
// halted at Step 3 would otherwise write thirty-seven files saying that nothing
// happened — and it is here because §8 reads Dispositions generically across
// all seven values and a reaper's file writes one of them by name.
type Disposition string

const (
	// DispositionRan is a Step invoked and reaching a conclusion hyper
	// recorded.
	DispositionRan Disposition = "ran"
	// DispositionSkippedAsAlreadyRecorded is skip-if-recorded finding the
	// Asset still standing. The only value that is Repeatability evidence.
	DispositionSkippedAsAlreadyRecorded Disposition = "skipped-as-already-recorded"
	// DispositionSkippedByCondition is a `when:` that did not hold. It says
	// nothing about what the world holds, which is why it is not the value
	// above.
	DispositionSkippedByCondition Disposition = "skipped-by-condition"
	// DispositionRefused is a guardrail declining before any effect reached
	// the world.
	DispositionRefused Disposition = "refused"
	// DispositionNeverReached is the Run ending before the Step. It is read
	// from a silence and written by no Step file.
	DispositionNeverReached Disposition = "never-reached"
	// DispositionAttemptedOutcomeUnknown is a call that went out with no
	// answer coming back. It attaches the uncertainty to the attempt rather
	// than to the thing, and it is the one value a ClosedBy file carries.
	DispositionAttemptedOutcomeUnknown Disposition = "attempted-outcome-unknown"
	// DispositionAttemptedWorldUntouched is a request that provably never
	// left. Effectful-only, and only where no call this Step made reached
	// the world (ADR-0062).
	DispositionAttemptedWorldUntouched Disposition = "attempted-world-untouched"
)

// Selector is the selector a Step resolved, held as authored beside what it
// resolved to, so that what a Step reached is readable back from the entry long
// after the Run without a checkout at the revision its Provenance names.
type Selector struct {
	// Declared is the selector as authored, in whichever of §12's three
	// `over:` forms it was written. It is nil on a Step carrying no
	// selector, which resolved none and holds none.
	Declared Value
	// ExpandedTo is what the Expansion resolved to, in **Expansion order**
	// and not sorted: on a serial destroy it is the only place the halt
	// point is legible, and *which three of the five* is read off it by
	// position (§6, ADR-0044).
	//
	// It is written whenever a selector exists, the empty list included —
	// the other exception to the absence rule, for the reason Members is:
	// an Expansion that resolved to nothing is not a Step with no selector.
	// Nil and empty are therefore one value here, and a decode answers the
	// empty list.
	ExpandedTo []string
	// Bound is what the Expansion was counted against, and zero where the
	// Step declared none — a read Step carries no Bound, having nothing for
	// one to guard (§4).
	Bound int
}

// write puts the selector into its block, and writes nothing at all where the
// Step declared none.
func (s Selector) write(m members) {
	if s.Declared == nil {
		return
	}
	m.value("declared", s.Declared)
	m["expanded_to"] = Always(namesArray(s.ExpandedTo))
	m.count("bound", s.Bound)
}

// Pattern is hyper's own account of the work, supplied by no Provider
// (ADR-0018): a retry's attempts, a paginated read's pages, a poll's
// iterations. It is what makes *attempted, outcome unknown* after five attempts
// a different fact on the page from the same Disposition after one.
//
// It is written where a Pattern did more than the trivial single call and
// absent otherwise — except on DispositionAttemptedOutcomeUnknown, where it is
// written whenever a Pattern was declared at all. How many times hyper may have
// touched the world is the fact that Disposition exists to carry, and *one
// attempt* and *no retry declared* are the same silence everywhere else and
// must not be here. Which of the two rules applies is the writer's, and the
// zero value of each member is that member's absence.
type Pattern struct {
	Attempts int
	Pages    int
	Polls    int
}

// write puts the Pattern account into its block.
func (p Pattern) write(m members) {
	m.count("attempts", p.Attempts)
	m.count("pages", p.Pages)
	m.count("polls", p.Polls)
}

// Answered is what an effectful Step's call gave back where it did not give the
// ordinary answer. Its presence is the fact that something other than the
// ordinary answer decided this Step, and which of §6's three cases it was is
// read from the Disposition beside it.
//
// It is effectful-only. A read's status is the answer, and the answer belongs
// in the Record wherever its Manifest projected it; a Journal copy would add
// only a claim that hyper thought a 503 was untoward, which on a read it does
// not (§6, ADR-0010).
//
// The set is closed by the unexported method for the reason Value's is: two
// Capabilities, two member sets, and no third.
type Answered interface {
	// write puts the answer's own members into the block.
	write(m members)
}

// Answer is an integer a call may not have given back: an HTTP status, a shell
// exit code. The zero value is *no answer arrived*, which is the case §7 states
// the key is absent for, and Arrived is the one door a value comes through — so
// a caller cannot mean 0 by leaving a field unset.
type Answer struct {
	code    int
	arrived bool
}

// Arrived is an answer that came back.
func Arrived(code int) Answer { return Answer{code: code, arrived: true} }

// Code is the answer and whether one arrived. The two come back together
// because reading the first without the second is how a request that never left
// acquires a status of 0 — the reading §7 states the absent key exists to
// prevent.
func (a Answer) Code() (code int, arrived bool) { return a.code, a.arrived }

// HTTPAnswer is the http Capability's: the host reached and the status it gave.
// Where no response arrived the status is absent, on the rule §3's response
// object carries (ADR-0050), so a Step that is *attempted, world untouched*
// writes the host alone. It says the request did not arrive and never which of
// ADR-0018's members stopped it.
type HTTPAnswer struct {
	Host   string
	Status Answer
}

func (a HTTPAnswer) write(m members) {
	m.text("host", a.Host)
	m.answer("status", a.Status)
}

// ShellAnswer is the shell Capability's: the command it ran and the code it
// exited with, the code absent where the command could not be started at all.
// Its threshold is 0 rather than 2xx, and it covers two of §6's three cases
// rather than all of them, there being no 404 for a command to answer with.
//
// Command is written rather than left to the identity set beside it: a destroy
// projects nothing and declares no identity, so on the Kind where this key
// matters most there is no projected command anywhere in the entry — and it is
// what keeps the key from ever being written empty, a failed exec otherwise
// leaving an `answered: {}` the encoding suppresses outright.
type ShellAnswer struct {
	Command  string
	ExitCode Answer
}

func (a ShellAnswer) write(m members) {
	m.text("command", a.Command)
	m.answer("exit_code", a.ExitCode)
}

// StepCode is a Step's code facts: its authored id, and what that id resolved
// to at the Run's revision. They are one shape because two files carry them —
// a Step file, which holds them rather than resolving a Manifest at read time,
// and a ClosedBy file, whose reaper resolves them from the dead Run's revision
// or writes none of them at all (§7, §8).
type StepCode struct {
	// ID is the Step's authored id — what a later Run matches this Step by
	// when it looks for the last identity set it carried (ADR-0055).
	ID string
	// Definition, Operation, Provider, Target and Kind are what the Step
	// was going to do. Provider is the Provider's **name** where a
	// ManifestDigest is the Provider's **bytes**: the digest identifies
	// what ran and answers nothing about what it was, and deriving the name
	// instead costs git objects a shallow clone does not have.
	Definition string
	Operation  string
	Provider   string
	Target     string
	Kind       Kind
}

// write puts the code facts into a file's members. Each is absent where the
// caller has none, which on a ClosedBy file is all of them together.
func (c StepCode) write(m members) {
	m.text("id", c.ID)
	m.text("definition", c.Definition)
	m.text("operation", c.Operation)
	m.text("provider", c.Provider)
	m.text("target", c.Target)
	m.text("kind", string(c.Kind))
}

// readStepCode reads a Step's code facts back, which two of the five shapes
// carry.
func readStepCode(f *fields) StepCode {
	return StepCode{
		ID:         f.text("id"),
		Definition: f.text("definition"),
		Operation:  f.text("operation"),
		Provider:   f.text("provider"),
		Target:     f.text("target"),
		Kind:       oneOf(f, "kind", KindRead, KindMutate, KindDestroy),
	}
}

// StepFile is one Step of a Run reaching a Disposition: what the Step was, what
// became of it, and the three things a Disposition holds.
//
// Its code facts are held rather than resolved from a Manifest at the Run's
// revision. A Journal whose Dispositions cannot be read without fetching three
// artefacts is evidence with a dependency, and the Kind in particular is the
// fact §8's third table exists to report as moving — read back from a Manifest
// it would be today's Kind wearing that Run's date.
type StepFile struct {
	// Step is the Step's position in the Run's written order, the first
	// Step 1. A nested Procedure's Steps are counted in that order, the
	// invocation itself being no Step and writing no file.
	Step int
	// Path is the invocation chain where this Step was reached through a
	// nested Procedure invocation, `retire.probe`, and empty on a top-level
	// Step.
	Path string
	// StepCode is what this Step was going to do, held rather than
	// resolved.
	StepCode
	// Disposition is what became of the Step, and it is one of the six
	// §12 values a file can bear. DispositionNeverReached is the seventh
	// and is read from the absence of a file inside a closed entry, so it
	// is the one value this member can never hold (§7).
	Disposition Disposition
	// StartedAt and EndedAt are the instants this Step began and ended. No
	// duration is stored: it derives, and only within one entry, the laptop
	// and the runner not sharing a clock.
	StartedAt, EndedAt time.Time
	// Provenance is the Step's half. This file sits beside run.json and
	// reads the Run-wide members one file over, so it carries none of them.
	Provenance StepProvenance
	// Identities is what the Step concluded about, absent on the three
	// Dispositions that conclude about nothing.
	Identities Identities
	// Selector is the selector as authored beside what it expanded to,
	// absent where the Step declared none.
	Selector Selector
	// Pattern is hyper's own account of the work.
	Pattern Pattern
	// Answered is what an effectful call gave back where it did not give
	// the ordinary answer, and nil otherwise.
	Answered Answered
	// ProjectionFailedPath is the path that failed to project on a Step
	// halted by a projection that did not resolve (§6). The identity set
	// beside it is then partial and this path is what says so — the digest
	// says nothing about partiality either way. It is held here and nowhere
	// else: a rendering goes to a terminal that scrolls, and no surface
	// shows the response it failed against (ADR-0017).
	ProjectionFailedPath string
}

// Encode writes a Step file in §7's canonical encoding.
func (f StepFile) Encode() []byte {
	if f.Disposition == DispositionNeverReached {
		impossible("a Step file carries %q: that Disposition is read from the absence of one inside a closed entry (§7, §12)", DispositionNeverReached)
	}
	return file(StepSchemaVersion, "a Step file", stepFileMembers, func(m members) {
		m.count("step", f.Step)
		m.text("path", f.Path)
		f.StepCode.write(m)
		m.text("disposition", string(f.Disposition))
		m.at("started_at", f.StartedAt)
		m.at("ended_at", f.EndedAt)
		m.block("provenance", f.Provenance.write)
		m.block("identities", f.Identities.write)
		m.block("selector", f.Selector.write)
		m.block("pattern", f.Pattern.write)
		if f.Answered != nil {
			m.block("answered", f.Answered.write)
			// Which of the two it names is what a reader tells the
			// Capabilities apart by, and naming neither is the
			// empty `answered` §7 says is never written — the
			// failed exec whose fact would vanish exactly where it
			// is least ordinary.
			if named, _ := m["answered"].(Mapping); named["host"] == nil && named["command"] == nil {
				impossible("an answer names neither the host it reached nor the command it ran (§7)")
			}
		}
		m.text("projection_failed_path", f.ProjectionFailedPath)
	})
}

// ClosedBy is a closing write: one Run's account of another Run's entry, drawn
// from a silence. It carries what a reaper knows and omits what it cannot
// establish (§7, ADR-0076).
//
// It carries no member naming its author — its path is that member — and never
// `started_at`: the reaper does not know when the Step began, and filling it
// would be hyper asserting something about a Run it did not perform, on the
// surface built to hold what happened.
//
// It carries no `outcome` either. The entry's outcome is a question about the
// **entry**, which this file's existence answers in full; a Disposition is a
// fact about a **Step** that this file is the only carrier of.
type ClosedBy struct {
	// EndedAt is the closing Run's instant on the closing Run's clock,
	// which is why a reaped entry renders no duration at all: subtracting
	// the dead Run's started_at from it is a cross-entry subtraction (§7).
	EndedAt time.Time
	// Step is the Step the dead Run went quiet on: the one after the
	// highest <nnnn> its entry holds. Which Step that is is not a guess —
	// run.json names the Procedure and the repository revision to load it
	// at — but the answer depends on that revision resolving.
	Step int
	// StepCode is the Step's id and its code facts where the dead Run's
	// revision resolves them, and the zero value where it does not, which
	// is every Run that recorded repo_dirty.
	StepCode
}

// Encode writes a closed-by/ file in §7's canonical encoding.
//
// Its `disposition` is DispositionAttemptedOutcomeUnknown and no other value
// can appear there, so it is written by this encoder rather than taken from the
// caller: without it §6's rule has nowhere to land and the crashed Step reads
// as never reached, which re-runs an effect nobody vouched for. A field would
// be a way to write something else there.
func (f ClosedBy) Encode() []byte {
	return file(ClosedBySchemaVersion, "a closing write", closedByMembers, func(m members) {
		m.count("step", f.Step)
		f.StepCode.write(m)
		m.text("disposition", string(DispositionAttemptedOutcomeUnknown))
		m.at("ended_at", f.EndedAt)
	})
}

// Reading is what this closing write records about the Step the dead Run went
// quiet on, in the shape a Step file records one — which is what lets §8 read
// Dispositions generically across all seven values, and what makes *attempted,
// outcome unknown* evidence for run-once whichever of the two files carries it
// (§6, §7).
//
// It is a reading and not a file, and the difference is in the value rather
// than in a comment: StartedAt is zero because the reaper does not know when
// the Step began, and the Provenance, the identity set and the rest are absent
// because it could not establish them. What that leaves would panic if it were
// encoded, a Step file carrying members this one has none of — so a reading
// cannot quietly become a file nobody wrote.
func (f ClosedBy) Reading() StepFile {
	return StepFile{
		Step:        f.Step,
		StepCode:    f.StepCode,
		Disposition: DispositionAttemptedOutcomeUnknown,
		EndedAt:     f.EndedAt,
	}
}

// DecodeStepFile reads a Step file back to the value it was written from.
func DecodeStepFile(data []byte) (StepFile, error) {
	return decodeFile(data, StepSchemaVersion, func(r *fields, f *StepFile) {
		r.require(stepFileMembers...)
		f.Step = r.position("step")
		f.Path = r.text("path")
		f.StepCode = readStepCode(r)
		f.Disposition = oneOf(r, "disposition",
			DispositionRan,
			DispositionSkippedAsAlreadyRecorded,
			DispositionSkippedByCondition,
			DispositionRefused,
			DispositionAttemptedOutcomeUnknown,
			DispositionAttemptedWorldUntouched,
		)
		f.StartedAt = r.at("started_at")
		f.EndedAt = r.at("ended_at")
		f.ProjectionFailedPath = r.text("projection_failed_path")

		if provenance := r.block("provenance"); provenance != nil {
			provenance.require(stepProvenanceMembers...)
			f.Provenance = readStepProvenance(provenance)
			r.join(provenance, "provenance")
		}
		if identities := r.block("identities"); identities != nil {
			identities.require("digest")
			f.Identities = Identities{Digest: identities.text("digest"), Members: identities.names("members")}
			if held := f.Identities.Members; held != nil && !slices.Equal(held, Concluded(held, "").Members) {
				identities.fault("the members are not the sorted set the digest is taken over")
			}
			r.join(identities, "identities")
		}
		if selector := r.block("selector"); selector != nil {
			selector.require("declared", "expanded_to")
			f.Selector = Selector{
				Declared:   selector.value("declared"),
				ExpandedTo: selector.names("expanded_to"),
				Bound:      selector.count("bound"),
			}
			r.join(selector, "selector")
		}
		if pattern := r.block("pattern"); pattern != nil {
			f.Pattern = Pattern{
				Attempts: pattern.count("attempts"),
				Pages:    pattern.count("pages"),
				Polls:    pattern.count("polls"),
			}
			r.join(pattern, "pattern")
		}
		if answered := r.block("answered"); answered != nil {
			switch {
			case answered.carries("host"):
				f.Answered = HTTPAnswer{Host: answered.text("host"), Status: answered.answer("status")}
			case answered.carries("command"):
				f.Answered = ShellAnswer{Command: answered.text("command"), ExitCode: answered.answer("exit_code")}
			default:
				answered.fault("an answer names the host it reached or the command it ran")
			}
			r.join(answered, "answered")
		}
	})
}

// DecodeClosedBy reads a closed-by/ file back to the value it was written from.
//
// Its `disposition` is checked rather than read: the one value that can appear
// there is written by the encoder, so a file carrying another was written by
// something that is not hyper — and reading it would let a crashed Step arrive
// wearing a Disposition §6's rule has nowhere to land on.
func DecodeClosedBy(data []byte) (ClosedBy, error) {
	return decodeFile(data, ClosedBySchemaVersion, func(r *fields, f *ClosedBy) {
		r.require(closedByMembers...)
		f.Step = r.position("step")
		f.EndedAt = r.at("ended_at")
		f.StepCode = readStepCode(r)

		// Read against the one value rather than through the closed set:
		// the set has seven members and this file may carry one, so what
		// a reader wants told is which value it is missing.
		if r.text("disposition") != string(DispositionAttemptedOutcomeUnknown) {
			r.fault("a closing write's disposition is %q and no other value", DispositionAttemptedOutcomeUnknown)
		}
	})
}
