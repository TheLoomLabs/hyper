package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// runCommand is the name this command's messages and its gate are spelled with.
// `run` and `runs` sit one letter apart, which §9 calls a readability wart
// rather than a hazard — `run` requires an argument and `runs` accepts none
// positionally, so the typo in either direction is a usage error.
const runCommand = "run"

// RunRun implements `hyper run <procedure>` — the tracer bullet, and the first
// thing in the tool that writes to the record on its own account (issue #136).
//
// The name is the wart §9 already owns: every command in this package is
// `Run<Command>` and this command is `run`, so a reader looking for `hyper
// run`'s entry point finds it where they would look for any other's. Nothing
// else here is named for it.
//
// What it does is decide the occasion and render the answer, and the whole of
// what happens in between is internal/run's. The split is ADR-0026's, and it is
// the reason the engine can be exercised without a subprocess: the Trigger, the
// clock, the mint, the dialer and the streams are read and resolved here, and
// the engine reaches for none of them.
//
// The order past the gate is §9's *positional, then the Store*: `hyper run
// typo` is `2` on a repository with no Store at all, because a working-tree
// name needs nothing further to resolve and the typo is repaired before the
// Store is missed (§9, ADR-0060). Past that it is §6's fixed order, which
// internal/run states.
func RunRun(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
	// `--target` is refused by name rather than as an unknown flag, because
	// what it means is a decision rather than a typo: a Procedure is fully
	// bound and declares its own Target envelope, so a Target supplied at
	// invocation is either redundant with the artefact or it is authority
	// arriving after review (§5, §9, ADR-0008).
	if fault := refusedFlags(args); fault != "" {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, fault)
		return ExitUsage
	}

	// No --limit: a Run reports what it just did rather than ranging over a
	// namespace, so there is no result set for a cap to cut (§9). `--dry-run`
	// and `--secret-out` are the two flags §9 gives this command and no
	// other, and both are later milestones' — until then each is an unknown
	// flag, which is the honest answer for a marker nothing reads.
	parsed, code := parseArgs(runCommand, args, takesNoLimit, process.LookupEnv, stderr)
	if code != 0 {
		return code
	}
	if len(parsed.positional) != 1 {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, arityFault(parsed.positional, "Procedure"))
		return ExitUsage
	}
	name := parsed.positional[0]

	repoRoot, code := resolveRepoRoot(runCommand, parsed.repoDir, process.LookupEnv, wd, stderr)
	if code != 0 {
		return code
	}
	if code := gateOnVersionPin(runCommand, repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, err)
		return ExitUsage
	}
	if fault := unresolvedProcedure(loaded, name); fault != "" {
		fmt.Fprint(stderr, fault)
		return ExitUsage
	}

	// What the binary has not built declines here, before the Store is
	// located and for the reason the positional above resolves before it: a
	// working-tree name needs nothing further, and an operator whose
	// Procedure this binary cannot perform is owed that answer rather than
	// a `hyper store init` that would not help. internal/run states why the
	// decline exists and why it is not a Refusal — no `error_code`, no
	// Journal entry, no row stream, and the milestone that builds the thing
	// deletes its line.
	if declined := run.NotBuilt(loaded, name); len(declined) > 0 {
		return declineNotBuilt(stderr, declined)
	}

	// The Store, located: the second thing §6's order does, and the one that
	// declines before the Run has an id to decline under. A Run never
	// creates the branch, read-only Runs included, because a fetch that
	// failed mid-flight and a branch that never existed look identical from
	// the inside (§6, §7).
	instant := process.Now()
	if err := store.Sync(repoRoot, instant); err != nil {
		return reportRunStoreFault(stderr, err)
	}
	held, err := store.Open(repoRoot, instant)
	if err != nil {
		return reportRunStoreFault(stderr, err)
	}

	// The rehearsal marker, and it is `false` until issue #145 lands
	// `--dry-run`. It is a value rather than a literal at the two places it
	// reaches because those two must never disagree: the entry carries it
	// and the terminal row carries it, and §7's one exception to the
	// absence rule is written always, `false` included, precisely because a
	// reader that gets it wrong cannot recover (§7, §8, ADR-0001).
	dryRun := false

	answer := run.Perform(run.Request{
		Repository: loaded,
		RepoRoot:   repoRoot,
		Store:      held,
		Procedure:  name,
		DryRun:     dryRun,
		Trigger:    readTrigger(process.LookupEnv, process.User, process.Hostname),
		Version:    binaryVersion,
		Now:        process.Now,
		Mint:       process.Mint,
		Dial:       process.Dial,
		Exec:       process.Exec,
		Narrator:   narration{stderr: stderr},
	})

	// The engine's own answer to the same question, which the call above has
	// already made unreachable. It is read rather than assumed away because
	// the decline is the engine's precondition and not this command's: a
	// second caller — the MCP surface, milestone 11 — reaches Perform
	// through its own dispatch, and an answer nobody looked at would be a
	// Run declining into silence there.
	if len(answer.Unbuilt) > 0 {
		return declineNotBuilt(stderr, answer.Unbuilt)
	}

	// The fault is narration's and stdout carries none of it: a failure
	// carries no `error_code`, and what tells two failures apart is the exit
	// code (§9, §12).
	if answer.Fault != nil {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, answer.Fault)
	}

	terminal := outcomeRow{
		Type:    "outcome",
		Outcome: string(answer.Outcome),
		Code:    exitFor(answer.Outcome),
		DryRun:  dryRun,
	}
	if answer.Identified {
		terminal.RunID = answer.Run.String()
	}

	rows := runRows(answer)
	if code := writeAnswer(runCommand, stdout, stderr, parsed.json, rows, terminal, runPage(terminal)); code != 0 {
		return code
	}
	return terminal.Code
}

// declineNotBuilt renders what this binary does not implement and answers the
// exit code: one line each on stderr, `2`, and stdout completely silent — no
// `error_code`, no row stream, and no Journal entry, which is what makes it a
// usage error rather than a Refusal (§9, ADR-0060).
func declineNotBuilt(stderr io.Writer, declined []string) int {
	for _, line := range declined {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, line)
	}
	return ExitUsage
}

// exitFor maps an outcome onto the exit code §12 fixes for it. The code space
// is finer than the triple and never coarser, so this is the one direction that
// is a function: `failed` takes four codes and the three this milestone cannot
// reach — 75 for a Store lost past Run start, 130 and 143 for the two signals —
// are decided where they happen rather than derived from the outcome (§9, §12).
func exitFor(outcome store.Outcome) int {
	switch outcome {
	case store.OutcomeCompleted:
		return ExitClean
	case store.OutcomeRefused:
		return ExitRefused
	default:
		return ExitProblems
	}
}

// refusedFlags is the flag `run` refuses by name, and the reason it gives.
//
// It runs before parseArgs for splitInputs' own reason: the three globals are
// every command's and this is one command's, so a parser that knew about both
// is one every other command's signature would have to admit. `--` ends the
// flags, so a positional spelled like a flag is still reachable (§9).
func refusedFlags(args []string) string {
	for _, argument := range args {
		if argument == "--" {
			return ""
		}
		if argument == "--target" || strings.HasPrefix(argument, "--target=") {
			return "takes no --target: a Procedure is fully bound and declares its own Target envelope,\n" +
				"  so a Target named here is either redundant with the artefact or authority arriving after review"
		}
	}
	return ""
}

// unresolvedProcedure is the message a positional that is not a Procedure
// earns, and "" where the name resolves.
//
// Both forms are usage errors at `2` carrying no `error_code`: a Refusal is the
// artefacts declining an act and a usage error is there being no act to decline
// (§9, ADR-0060). They are two messages because the remedies differ — one
// caller mistyped a name and the other named the wrong kind of artefact — and
// the second is worth its own sentence: every Run is a Run of a Procedure, and
// what a caller reaching for a Definition wants is a Procedure of one Step
// (ADR-0036).
func unresolvedProcedure(loaded repository.Loaded, name string) string {
	if loaded.Procedures[name] {
		return ""
	}
	if _, isDefinition := loaded.Definitions[name]; isDefinition {
		return fmt.Sprintf("hyper %s: %q is a Definition, and every Run is a Run of a Procedure\n"+
			"  what runs a single Operation is a Procedure of one Step: a definition:, an operation:, a target: and its args:\n",
			runCommand, name)
	}
	return fmt.Sprintf("hyper %s: no Procedure named %q in this repository's Procedure namespace\n"+
		"  procedures/ is that namespace, and no command in the tree enumerates it\n", runCommand, name)
}

// reportRunStoreFault renders whichever way the record stopped a Run before it
// began, and answers the exit code.
//
// The two differ by what it would take to clear them, which is exactly what
// sorts 77 from 75 (§12, ADR-0061). A branch neither side holds Refuses
// `store-absent` naming `hyper store init` — the remedy is an act of
// somebody's. Everything else is a Run that **lost** the Store: a remote it
// could not reach, a fetch that would not land, and past those lies time rather
// than an act, so each is `failed` at 75 and never a Refusal.
//
// `store-schema-unsupported` is not answered here and could not be: locating
// the Store opens no file, and the test over the files a Run will read is a
// gate of its own, one place further down §6's order (issue #137). It arrives
// with that gate.
//
// stdout is silent on all three, which is the milestone-5 Refusal rendering
// this package already has: §8's caret excerpt, its `=` notes and its
// `EDIT ONE OF` table are milestone 8's, and so is the `outcome` row §8 says a
// Refusal's stream carries. What is deferred is the shape; what is on the page
// already is the code and the remedy, which is the whole of the path back from
// this one (gate.go states the same deferral for the pin gate). 75 is not a
// Refusal at all and has no Step table to write either.
func reportRunStoreFault(stderr io.Writer, err error) int {
	if errors.Is(err, store.ErrAbsent) {
		return refuse(stderr, storeAbsentCode, "no "+store.BranchName+" branch in this repository — hyper store init")
	}
	fmt.Fprintf(stderr, "hyper %s: the Store could not be synced: %s\n", runCommand, err)
	return ExitStoreLost
}

// narration is §9's stderr narration: the Run naming itself before its first
// Step, and one line per Step boundary, in both modes, always on.
//
// It carries no machine contract and has no `--json` variant — a consumer
// wanting Step-level structure reads the Journal, which §7 writes per Step as
// that Step reaches its Disposition. With the outcome arriving last, a silent
// twenty minutes is otherwise indistinguishable from a hang.
type narration struct{ stderr io.Writer }

// Began writes the Run's id before its first Step. The id renders here as well
// as on the terminal line, and that repetition is deliberate: the terminal line
// is not always reached — a process killed outright renders nothing and leaves
// the open entry §7 states — and this is the one Run whose identity its own
// output would otherwise never carry (§9, ADR-0047).
func (n narration) Began(id store.RunID) {
	fmt.Fprintf(n.stderr, "run %s\n", id)
}

// Reached writes one Step boundary: which Step, of how many, and its authored
// id.
func (n narration) Reached(position, of int, id string) {
	fmt.Fprintf(n.stderr, "step %d/%d %s\n", position, of, id)
}

// runColumns is the Step table's header (§8).
var runColumns = []string{"STEP", "ID", "KIND", "DISPOSITION", "RECORDS"}

// noRecords is what the RECORDS column renders where **no set exists**, rather
// than a set with nothing in it. *refused*, *skipped by condition*, *never
// reached* and *attempted, world untouched* conclude about nothing, and the
// dash is what tells them from the *ran* Step whose Expansion resolved to
// nothing, whose set is written empty and whose cell reads `0` (§8, ADR-0030).
const noRecords = "–"

// runRows is the Run's answer as rows: one `step` row per Step that reached a
// Disposition, then the Run's `provenance` in both its scopes — the Run-wide
// row and one per Step file written — and the terminal `outcome` row is
// writeAnswer's (§8, §9).
//
// The order is the page's, and here it is a contract rather than a consequence:
// a row goes out on its own line, there is no cursor behind the stream, and a
// consumer cannot re-sort what it has already printed (§8).
func runRows(answer run.Answer) []render.Row {
	rows := make([]render.Row, 0, 2*len(answer.Steps)+1)
	for _, step := range answer.Steps {
		rows = append(rows, stepRowOf(step))
	}
	if answer.Identified {
		rows = append(rows, runProvenanceRow(answer.Provenance))
	}
	for _, step := range answer.Steps {
		rows = append(rows, stepProvenanceRow(step))
	}
	return rows
}

// runPage is the Run's page: §8's Step table, and beneath it §8's terminal
// line, which is the answer's last line and goes to stdout like the rest of it
// — which is what puts the Run id on a job summary (§9, §10).
//
// The terminal row is captured rather than found among the rows, because the
// two forms are one fact rendered twice: what the row carries the line carries,
// and a fact the stream states and the page does not is the two surfaces
// disagreeing about what happened rather than differing in shape (ADR-0026).
func runPage(terminal outcomeRow) func(io.Writer, []render.Row) error {
	return func(w io.Writer, rows []render.Row) error {
		if err := render.WriteTable(w, runColumns, rows); err != nil {
			return err
		}
		if slices.ContainsFunc(rows, func(row render.Row) bool { return len(row.Cells()) > 0 }) {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(w, terminal.line())
		return err
	}
}

// stepRow is one Step of the Run: its position, its authored id, its Kind, its
// Disposition and how many Records it concluded about (§8).
//
// records is a pointer because it is a member the row must be able to *not*
// carry and whose zero value is a value it must be able to state: a *ran* Step
// whose Expansion resolved to nothing concluded about nothing and writes `0`,
// and a *refused* Step has no set at all and writes no key.
//
// §8's third form of the cell — `n of m`, where the Expansion reached `m` and
// the rest are unaccounted for — is not here, and neither is the `expanded`
// member beside it. Nothing this milestone runs can stop short of an Expansion:
// a Step carrying an `over:` declines before Step 1 (internal/run). It arrives
// with the Expansion that makes it reachable (issue #139, issue #144).
type stepRow struct {
	Type        string `json:"type"`
	Step        int    `json:"step"`
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Disposition string `json:"disposition"`
	Records     *int   `json:"records,omitempty"`
}

// stepRowOf is one Step of the answer as a row. Records is written where the
// Step concluded about anything at all and absent where it concluded about
// nothing, which is the distinction §8's dash renders (ADR-0030).
func stepRowOf(step run.Step) stepRow {
	row := stepRow{
		Type:        "step",
		Step:        step.Position,
		ID:          step.ID,
		Kind:        string(step.Kind),
		Disposition: string(step.Disposition),
	}
	if step.Concluded {
		records := step.Records
		row.Records = &records
	}
	return row
}

// Cells is the Step's line in §8's table: `n` where the set is all the Step
// reached, and the dash where no set exists at all.
func (r stepRow) Cells() []string {
	records := noRecords
	if r.Records != nil {
		records = fmt.Sprintf("%d", *r.Records)
	}
	return []string{fmt.Sprintf("%d", r.Step), r.ID, r.Kind, r.Disposition, records}
}

// provenanceRow is which code performed the Run, at one of the two scopes §7
// splits it across: the Run-wide members, and one row per Step file written.
//
// Which scope a row is is read off the row itself and never off a key naming
// one. A Step's row carries `step` and the Run's does not, which is exactly the
// split the Store already writes — a member is written at the level where it
// has one value and omitted from every level where it has none (ADR-0043) — and
// a discriminator beside it would carry that fact twice.
//
// Nothing is abbreviated: every revision and every digest goes out whole (§8,
// ADR-0047).
type provenanceRow struct {
	Type              string `json:"type"`
	Step              *int   `json:"step,omitempty"`
	HyperVersion      string `json:"hyper_version,omitempty"`
	ProcedureRevision string `json:"procedure_revision,omitempty"`
	RepoRevision      string `json:"repo_revision,omitempty"`
	RepoDirty         bool   `json:"repo_dirty,omitempty"`

	DefinitionRevision string `json:"definition_revision,omitempty"`
	ManifestDigest     string `json:"manifest_digest,omitempty"`
	OriginDigest       string `json:"origin_digest,omitempty"`
}

// runProvenanceRow is the Run-wide scope: the members that have exactly one
// value across a Run, however many Definitions its Steps span (ADR-0043).
func runProvenanceRow(p store.RunProvenance) provenanceRow {
	return provenanceRow{
		Type:              "provenance",
		HyperVersion:      p.HyperVersion,
		ProcedureRevision: p.ProcedureRevision,
		RepoRevision:      p.RepoRevision,
		RepoDirty:         p.RepoDirty,
	}
}

// stepProvenanceRow is one Step's scope, and the `step` it carries is what
// tells the two apart on the wire.
func stepProvenanceRow(step run.Step) provenanceRow {
	position := step.Position
	return provenanceRow{
		Type:               "provenance",
		Step:               &position,
		DefinitionRevision: step.Provenance.DefinitionRevision,
		ManifestDigest:     step.Provenance.ManifestDigest,
		OriginDigest:       step.Provenance.OriginDigest,
	}
}

// Cells is empty: Provenance is not on §8's Step table. What renders it to a
// human is `hyper show`, which reads the entry back (§9); the page a Run writes
// is the table and the terminal line, and a row with no line is the shape the
// terminal row already has (ADR-0026).
func (r provenanceRow) Cells() []string { return nil }

// outcomeRow is the terminal row of a Run's stream, and the only stream in the
// tool that ends in one: everything that is not a Run ends in `result` (§8,
// §9).
//
// `dry_run` is written always, `false` included — §7's one exception to the
// absence rule holding on the wire for the reason it holds in the Store: what a
// reader that takes its absence for `false` gets wrong is unrecoverable.
// `run_id` is written whole and is absent exactly where no entry was written.
// `error_code` names the check that declined a `refused` Run and is absent on
// the other two, a failure having no check to name (§12).
type outcomeRow struct {
	Type      string `json:"type"`
	Outcome   string `json:"outcome"`
	Code      int    `json:"code"`
	DryRun    bool   `json:"dry_run"`
	RunID     string `json:"run_id,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

// Cells is empty: the terminal row's line on the page is §8's terminal line,
// which runPage writes beneath the table rather than inside it.
func (r outcomeRow) Cells() []string { return nil }

// line is §8's terminal line: what happened, the exit code §9 fixes, and the
// entry that holds it — with the Run id **whole**, this being the one id an
// operator retypes (ADR-0047).
//
// A rehearsal carries the marker its entry carries. Without it the line a Run
// that reached the world writes and the line a rehearsal writes are the same
// bytes, on the one path where the difference is the whole point.
//
// Where no entry was written the id is absent and its absence is the fact: the
// line says what happened and names nothing to look up, there being nothing to
// look up.
func (r outcomeRow) line() string {
	parts := []string{r.Outcome}
	if r.DryRun {
		parts = append(parts, "dry-run")
	}
	parts = append(parts, fmt.Sprintf("exit %d", r.Code))
	if r.RunID != "" {
		parts = append(parts, "run "+r.RunID)
	}
	return strings.Join(parts, " · ")
}
