package store

import "time"

// The members each of the Journal's shapes always carries, stated once and read
// from both directions — the encoder holds a file to them and the decoder holds
// a file to them, which is what keeps a shape from being able to write bytes it
// would itself refuse.
var (
	outcomeFileMembers = []string{"outcome", "ended_at"}
	runFileMembers     = []string{"run_id", "procedure", "trigger", "started_at", "dry_run", "provenance"}
	triggerMembers     = []string{"cause", "executor", "actor"}
)

// Outcome is one of §12's terminal triple: exactly one per Run, and no fourth
// value and no partial one.
type Outcome string

const (
	// OutcomeCompleted is every Step reaching a terminal Disposition with
	// none refused and none failed — a Run whose every Step skipped
	// included.
	OutcomeCompleted Outcome = "completed"
	// OutcomeRefused is a guardrail declining before any effect reached the
	// world, most often before any Step existed.
	OutcomeRefused Outcome = "refused"
	// OutcomeFailed is the world resisting, the Run being stopped, or the
	// Run losing the Store.
	OutcomeFailed Outcome = "failed"
)

// OutcomeFile is outcome.json: the account a Run gives of its own end.
//
// It is written by the Run whose entry it is and by no other, so it carries no
// member naming its author — the <run-id> in its path is that member, and
// another Run's account of the entry is a ClosedBy file (§7, ADR-0076).
//
// It carries no exit code and no duration. Both derive — the exit code from the
// outcome by §12's mapping, the duration from the instants the entry already
// holds — and a stored one is a second representation that can disagree.
type OutcomeFile struct {
	// Outcome is the triple's member this Run reached.
	Outcome Outcome
	// EndedAt is when the Run ended, on the Run's own clock.
	EndedAt time.Time
	// Refusal is the checks that declined, in the order check prints them,
	// and is present exactly where Outcome is OutcomeRefused. What a
	// terminal line names is the first member's ErrorCode, derived there
	// and stored nowhere (§7, ADR-0061).
	Refusal []RefusalMember
}

// Head is what §8's terminal line and outcome row name: the first member's
// error_code, and the empty string where the Run did not refuse.
//
// It is derived here and stored nowhere. A stored head is a second
// representation of the array's first member and the two can disagree, which is
// the reason no exit code, no duration and no Head marker is stored either (§7).
func (f OutcomeFile) Head() string {
	if len(f.Refusal) == 0 {
		return ""
	}
	return f.Refusal[0].ErrorCode
}

// Encode writes outcome.json in §7's canonical encoding.
//
// It declines to write a Refusal that is not one: `refusal` is an ordered array
// of at least one member and is present exactly where the outcome is refused
// (§7). Neither halves of that can arrive from the world — the outcome is
// hyper's own triple and the members are hyper's own checks — so a mismatch is
// hyper's arithmetic being wrong, which paths.go answers the same way.
func (f OutcomeFile) Encode() []byte {
	if (f.Outcome == OutcomeRefused) != (len(f.Refusal) > 0) {
		impossible("an outcome of %q carries %d refusal members: a Refusal is an ordered array of at least one member, present exactly where the outcome is refused (§7)", f.Outcome, len(f.Refusal))
	}
	return file(OutcomeSchemaVersion, "outcome.json", outcomeFileMembers, func(m members) {
		m.text("outcome", string(f.Outcome))
		m.at("ended_at", f.EndedAt)
		if len(f.Refusal) > 0 {
			refusal := make(Array, len(f.Refusal))
			for i, member := range f.Refusal {
				refusal[i] = member.value()
			}
			m["refusal"] = refusal
		}
	})
}

// RefusalMember is one check that declined a Run: what a check problem row
// carries — the error_code, the file, the line, the field and the message —
// plus what a Run adds, the Step it cites and the two values it compared.
//
// A Refusal and a check problem are one shape because they are one thing
// arriving through two commands: what check reports offline is what stops a Run
// online. It is stated here in its own members rather than by embedding §12's
// problem row because the Store's member carries no column — that one rides on
// §8's wire and is read back out of no file — and a shape that dropped a member
// on the way in would not read back equal.
type RefusalMember struct {
	// ErrorCode is the member of §12's closed set naming the check.
	ErrorCode string
	// File is what the check cites: a reviewed artefact on most codes, a
	// generated workflow on projection-stale, a Store file on
	// store-schema-unsupported. It is `file` and not `artefact` for those
	// last two (§7).
	File string
	// Line is the 1-indexed line in File, and Field a path into it in §8's
	// remediation notation. Either is absent where the check has none.
	Line  int
	Field string
	// Message is free text describing the fault.
	Message string
	// Step and StepID are the Step the check cites, where it cites one.
	// Step is an artefact coordinate and never an execution fact: a Step it
	// names may have no file in the entry at all.
	Step   int
	StepID string
	// Declared and Observed are the two values the check compared, where it
	// compared two. Nothing is invented to fill either: a check that
	// compared nothing writes neither.
	Declared, Observed Value
}

// value is the member as it is written. Every key it does not carry is absent,
// which is the absence rule applied to a check rather than to a file.
func (r RefusalMember) value() Value {
	m := members{}
	m.text("error_code", r.ErrorCode)
	m.text("file", r.File)
	m.count("line", r.Line)
	m.text("field", r.Field)
	m.text("message", r.Message)
	m.count("step", r.Step)
	m.text("step_id", r.StepID)
	m.value("declared", r.Declared)
	m.value("observed", r.Observed)
	return Mapping(m)
}

// Cause is what caused a Run: §12's closed pair. A dispatched workflow run is
// CauseManual on the Actions executor, which is why the cause and the executor
// are two fields and not one.
type Cause string

const (
	// CauseCron is a Cadence the executor's clock fired (§10).
	CauseCron Cause = "cron"
	// CauseManual is a person.
	CauseManual Cause = "manual"
)

// Executor is where a Run happened: §12's other closed pair. hyper fills it by
// reading the environment it finds itself in and branches on nothing it finds —
// recording which executor ran is not an authority axis, and behaving
// differently on one would be (§5).
type Executor string

const (
	// ExecutorGitHubActions is a Run on a runner.
	ExecutorGitHubActions Executor = "github-actions"
	// ExecutorLocal is a Run on somebody's machine.
	ExecutorLocal Executor = "local"
)

// Trigger names what caused a Run, which executor it happened on, and which
// occasion on that executor.
//
// It is a mapping rather than a string: four facts whose shape differs by
// executor do not pack into one without a grammar and a parser, and a job URL
// carries every separator such a packing would use.
type Trigger struct {
	// Cause and Executor are §12's two closed pairs, and both are always
	// written.
	Cause    Cause
	Executor Executor
	// Actor is written on both executors — the Actions actor, or the
	// operating system user.
	Actor string
	// Host is the machine, and is written on ExecutorLocal only. §8's
	// header renders `igor@thinkpad` from it and Actor, so both forms come
	// from stored facts with nothing invented at render.
	Host string
	// ExecutorRun, Attempt and JobURL are the occasion on Actions: the
	// executor's own run id, its run attempt, and the URL of the job. They
	// are what links an entry to the narration that produced it — without
	// them a Run id and the job that emitted it are unrelatable.
	//
	// ExecutorRun is the executor's id and never a hyper RunID: it is a
	// decimal counter GitHub minted, and typing it as one would be this
	// package claiming a UUIDv7 where a foreign id stands.
	ExecutorRun string
	Attempt     int
	JobURL      string
}

// Text is the Trigger as one line: a clock or a person, which is what §7 says a
// Trigger names and the whole of what a surface reading one down a column
// renders of it.
//
// A `cron` Run renders the cause, because there is nobody to name: the clock
// that fired is the executor's, and the actor an executor happens to have set
// on a scheduled occasion is not who caused the Run. Everything else renders
// the person, with the machine beside them where the entry carries one —
// `igor@thinkpad` on a laptop and `igor` on a runner.
//
// It is a derivation over the entry's own stored facts and lives here beside
// Entry.Outcome and Entry.Ended for the same reason they do: §8's Comparison
// header and §9's `runs` column render one fact, and two compositions of it are
// two chances to disagree about who caused a Run. What a surface still decides
// is where the string goes — `show` reads one entry whole and renders the four
// members an executor writes, its job being the parts.
func (t Trigger) Text() string {
	if t.Cause == CauseCron {
		return string(CauseCron)
	}
	if t.Host == "" {
		return t.Actor
	}
	return t.Actor + "@" + t.Host
}

// write puts the Trigger into its block.
func (t Trigger) write(m members) {
	m.text("cause", string(t.Cause))
	m.text("executor", string(t.Executor))
	m.text("actor", t.Actor)
	m.text("host", t.Host)
	m.text("run_id", t.ExecutorRun)
	m.count("run_attempt", t.Attempt)
	m.text("job_url", t.JobURL)
	m.require("a Trigger", triggerMembers)
}

// RunFile is run.json: written at Run start, before any Step has been reached.
//
// It carries no account of how the Run ended. An entry's account is a
// classification over the files present under its directory — an outcome.json
// its own Run wrote, a ClosedBy file another Run wrote, or neither — and
// nothing in this file moves once it is written (§7, ADR-0011).
type RunFile struct {
	// Run is the Run's own id, restated here though the entry's path
	// carries it: the working tree describes itself, and a file read out of
	// a browser is read without its path in hand.
	Run RunID
	// Procedure is the top-level Procedure's name, which is also what
	// Provenance's ProcedureRevision is the revision of.
	Procedure string
	// Trigger is what caused this Run and where it happened.
	Trigger Trigger
	// StartedAt is when the Run began, and the instant the entry's date
	// partition is the UTC date of.
	StartedAt time.Time
	// DryRun says this Run was a rehearsal. It is written on every entry,
	// false included, and is the one marker in the Store that does not
	// follow the absence rule: four independent readers filter rehearsals
	// out, and one that takes absence for false refuses every run-once Step
	// in the Procedure it rehearsed, permanently, with nothing but an
	// artefact edit left (§6, §7, §8, ADR-0001).
	DryRun bool
	// Provenance is the Run-wide half and never the whole: a Step file one
	// directory over carries the Step's, and neither restates the other's.
	Provenance RunProvenance
}

// At is where this Run's entry sits: the coordinate §12's grammar builds every
// path under the entry from, built from what the file itself says rather than
// from the path it was found at. Every reader of an entry holds the two to
// agreeing, so this is that agreement used rather than restated.
func (f RunFile) At() JournalEntry {
	return JournalEntry{Run: f.Run, Started: f.StartedAt}
}

// Encode writes run.json in §7's canonical encoding.
func (f RunFile) Encode() []byte {
	return file(RunSchemaVersion, "run.json", runFileMembers, func(m members) {
		m.text("run_id", f.Run.String())
		m.text("procedure", f.Procedure)
		m.block("trigger", f.Trigger.write)
		m.at("started_at", f.StartedAt)
		m["dry_run"] = Always(Bool(f.DryRun))
		m.block("provenance", f.Provenance.write)
	})
}

// DecodeOutcomeFile reads outcome.json back to the value it was written from.
//
// It answers SchemaUnsupported where the file was written above this binary's
// ceiling, which the caller renders as a Refusal naming the path it read (§7,
// ADR-0028).
func DecodeOutcomeFile(data []byte) (OutcomeFile, error) {
	return decodeFile(data, OutcomeSchemaVersion, func(r *fields, f *OutcomeFile) {
		r.require(outcomeFileMembers...)
		f.Outcome = oneOf(r, "outcome", OutcomeCompleted, OutcomeRefused, OutcomeFailed)
		f.EndedAt = r.at("ended_at")

		refusal, present := r.array("refusal")
		if present == (f.Outcome != OutcomeRefused) {
			r.fault("an outcome of %q carries a refusal of %d members", f.Outcome, len(refusal))
			return
		}
		if !present {
			return
		}
		f.Refusal = make([]RefusalMember, len(refusal))
		for i, member := range refusal {
			f.Refusal[i] = readRefusalMember(r, member)
		}
	})
}

// readRefusalMember reads one check that declined. Its faults are the file's,
// so it reads through the array's own reader rather than carrying one of its
// own.
func readRefusalMember(r *fields, value Value) RefusalMember {
	mapping, ok := value.(Mapping)
	if !ok {
		r.fault("a refusal member is a mapping")
		return RefusalMember{}
	}

	member := newFields(mapping, r.err)
	member.require("error_code", "file", "message")
	read := RefusalMember{
		ErrorCode: member.text("error_code"),
		File:      member.text("file"),
		Line:      member.position("line"),
		Field:     member.text("field"),
		Message:   member.text("message"),
		Step:      member.position("step"),
		StepID:    member.text("step_id"),
		Declared:  member.value("declared"),
		Observed:  member.value("observed"),
	}
	r.join(member, "refusal")
	return read
}

// DecodeRunFile reads run.json back to the value it was written from.
func DecodeRunFile(data []byte) (RunFile, error) {
	return decodeFile(data, RunSchemaVersion, func(r *fields, f *RunFile) {
		r.require(runFileMembers...)
		f.Run = r.run("run_id")
		f.Procedure = r.text("procedure")
		f.StartedAt = r.at("started_at")
		f.DryRun = r.mark("dry_run")

		if trigger := r.block("trigger"); trigger != nil {
			trigger.require(triggerMembers...)
			f.Trigger = Trigger{
				Cause:       oneOf(trigger, "cause", CauseCron, CauseManual),
				Executor:    oneOf(trigger, "executor", ExecutorGitHubActions, ExecutorLocal),
				Actor:       trigger.text("actor"),
				Host:        trigger.text("host"),
				ExecutorRun: trigger.text("run_id"),
				Attempt:     trigger.count("run_attempt"),
				JobURL:      trigger.text("job_url"),
			}
			r.join(trigger, "trigger")
		}
		if provenance := r.block("provenance"); provenance != nil {
			provenance.require(runProvenanceMembers...)
			f.Provenance = readRunProvenance(provenance)
			r.join(provenance, "provenance")
		}
	})
}
