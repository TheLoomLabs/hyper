package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// showCommand is the name this command's messages and its gate are spelled
// with.
const showCommand = "show"

// RunShow implements `hyper show <run-id>` — one Journal entry read back whole
// (issue #163).
//
// It is the first command a person can type that reads the record back. The
// Journal reader in internal/store has been complete since milestone 4 and has
// been called by the reap, by the run-once evidence walk and by nothing anybody
// could invoke; this is the surface for it.
//
// **What it renders is what the entry holds and nothing reconstructed.** Its
// Step rows are the records the entry holds about its Steps, in the Run's own
// written order — the Step files it wrote, `<nnnn>` by §12's grammar, and,
// where a reaper closed it, the reading its earliest closing write carries
// beside them (§7). That is why this command orders nothing (§9).
//
// A Step the Run never reached wrote no file, so it has no row here and no
// `provenance` row either: that Disposition is read from a silence inside a
// closed entry (§7), and reading it back would mean loading the Procedure
// sequence at the Run's own revision — which is the reaper's act, on the four
// honest absences it costs, and not a thing this surface performs to fill a
// table.
//
// **The order past the gate is the Store and then the id**, which is the
// exception to §9's *positional, then the Store*: the Store is the namespace a
// `<run-id>` resolves against, so an absent branch is reported rather than the
// id blamed, and an unknown id is reachable only where the branch exists (§9,
// ADR-0060). A partial id resolves to nothing anywhere (ADR-0047), so a
// prefix and a typo arrive at one message.
//
// It is not a Run: it writes nothing, terminates its stream with `result`
// rather than `outcome`, and exits 0 whatever outcome the entry it read
// records — the exit code is this invocation's and never the Run's.
func RunShow(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
	// `--expansion` comes off before the globals, for `--dry-run`'s reason
	// one command over: the three globals are every command's and this is
	// one command's, so a parser that knew about both is one every other
	// command's call site would have to admit. `--` ends the flags, so a
	// positional spelled like a flag is still reachable (§9).
	expansion, rest := splitExpansion(args)

	// **No --limit**, and §9 says two things about that. Its Inspection
	// section enumerates each command's parameters and gives `--limit` to
	// three of the four — `runs`, `changes`, `records` — while `show`'s own
	// sentence gives it a Run id and `--expansion` and nothing else; a
	// paragraph further down then says every command in the section takes
	// one. The enumeration governs, because the general sentence is stated
	// in terms of the ordering — *truncation keeps the first N of the
	// ordering above* — and the paragraph immediately before it is the one
	// that says `show` **orders nothing**. There is no ordering here for a
	// cut to keep the first N of, and no axis for the marker to name, §12's
	// pair being the record's two and a Run's Steps neither.
	//
	// What a cap would do instead is hand back a Run's account with its last
	// Steps dropped, which is the partial answer wearing a complete one's
	// shape that §9 forbids `--history` for in as many words.
	parsed, code := parseArgs(showCommand, rest, parameters{limit: takesNoLimit}, process.LookupEnv, stderr)
	if code != 0 {
		return code
	}
	if len(parsed.positional) != 1 {
		fmt.Fprintf(stderr, "hyper %s: %s\n", showCommand, arityFault(parsed.positional, "Run id"))
		return ExitUsage
	}
	named := parsed.positional[0]

	repoRoot, code := resolveRepoRoot(showCommand, parsed.repoDir, process.LookupEnv, wd, stderr)
	if code != 0 {
		return code
	}
	if code := gateOnVersionPin(showCommand, repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	held, code := openStoreForReading(showCommand, repoRoot, process.Now(), stderr)
	if code != 0 {
		return code
	}

	entry, dispositions, found, err := readEntry(held, named)
	if err != nil {
		return reportReadStoreFault(showCommand, stderr, err)
	}
	if !found {
		fmt.Fprint(stderr, unresolvedRun(named))
		return ExitUsage
	}

	rows, err := showRows(held, entry, dispositions, expansion)
	if err != nil {
		return reportReadStoreFault(showCommand, stderr, err)
	}
	// `truncated` is always false: `show` takes no --limit and cuts
	// nothing, so the axes §12 closes name nothing here.
	if code := writeAnswer(showCommand, stdout, stderr, parsed.json, rows, render.NewResultRow(false), showPage); code != 0 {
		return code
	}
	return ExitClean
}

// splitExpansion takes `--expansion` off the argument list and answers whether
// it was given, leaving everything else for the shared parser.
//
// It is `show`'s alone. §9 gives it to this command and to `run_show`'s MCP
// twin and to nothing else, so `hyper runs --expansion` stays the unknown flag
// it is.
func splitExpansion(args []string) (expansion bool, rest []string) {
	for i, argument := range args {
		switch {
		case argument == "--":
			return expansion, append(rest, args[i:]...)
		case argument == "--expansion":
			expansion = true
		default:
			rest = append(rest, argument)
		}
	}
	return expansion, rest
}

// readEntry puts one entry and its Step records in hand, and answers whether
// the Store holds the id at all.
//
// The id is parsed here rather than beside the positional, and that placement
// is the rule §9 states: the Store is the namespace a `<run-id>` resolves
// against, so nothing about the id is judged before the branch is in hand. A
// value that is not a UUIDv7 resolves to nothing, exactly as a well-formed id
// no entry stands under does — **including a prefix of a real one**, since
// nothing anywhere resolves a partial id (ADR-0047) — so the two arrive at one
// answer and one message.
func readEntry(held *store.Store, named string) (store.Entry, store.Dispositions, bool, error) {
	id, err := store.ParseRunID(named)
	if err != nil {
		return store.Entry{}, store.Dispositions{}, false, nil
	}
	entry, found, err := held.Entry(id)
	if err != nil || !found {
		return store.Entry{}, store.Dispositions{}, false, err
	}
	dispositions, err := held.Dispositions(entry)
	if err != nil {
		return store.Entry{}, store.Dispositions{}, false, err
	}
	return entry, dispositions, true, nil
}

// unresolvedRun is what a `<run-id>` matching nothing says: what was typed, the
// namespace it resolved against, and the command that enumerates it.
//
// It **suggests no near miss**. A Run id is a UUIDv7 a person retypes, and a
// surface offering *did you mean* over one would be offering to run something
// the caller did not type — which is the whole of why nothing anywhere resolves
// a partial id (§9, ADR-0047).
//
// It is a usage error carrying no `error_code`: an id matching nothing is a
// name that resolves to nothing and never an artefact declining an act (§12,
// ADR-0060).
func unresolvedRun(named string) string {
	return fmt.Sprintf("hyper %s: no Journal entry for run %q in this repository's Store\n"+
		"  the %s branch is that namespace, and hyper runs enumerates it\n", showCommand, named, store.BranchName)
}

// showRows is the entry as rows: the `entry` header, the Refusal its own Run
// recorded where it recorded one, the Run's `provenance`, and then one `step`
// row per Step record with that Step's own `provenance` beside it (§8, §9).
//
// The Provenance split is §7's, held here rather than restated: one row
// carrying the Run-wide members and one per **Step file written**, told apart
// by the `step` the second carries and never by a key naming a scope
// (ADR-0043). Two records carry no Step-scoped Provenance and neither gets a
// row: a Step the Run never reached wrote no file and is not among these
// records at all, and a reaped entry's account of the Step the dead Run went
// quiet on is a closing write, which carries the Step's code facts and none of
// its revisions — the reaper could not establish them (§7, ADR-0076). A row
// with every member omitted would state that absence twice and state it as a
// row, which is the shape §8 says this stream does not have.
//
// Each Step's Provenance goes **beside that Step** rather than in a block of
// its own after the table, which is `run`'s order and not this one's: a Run
// reports a table of Steps and this reads one entry whole, so the page renders
// a block per Step and the stream is that page's order (§8, ADR-0026).
func showRows(held *store.Store, entry store.Entry, dispositions store.Dispositions, expansion bool) ([]render.Row, error) {
	rows := make([]render.Row, 0, 2*len(dispositions.Steps)+len(entry.Owner.Refusal)+2)
	rows = append(rows, entryRowOf(entry))
	// One `refusal` row per problem, in the array's order, and never one row
	// carrying an array: a consumer's `select(.type=="refusal")` returns one
	// problem per line here exactly as it does on the Run that wrote them
	// (§7, §8).
	for _, member := range entry.Owner.Refusal {
		operation, target := bindingOf(dispositions, member.Step)
		rows = append(rows, refusalRowOf(member, operation, target))
	}
	rows = append(rows, runProvenanceRow(entry.Provenance))
	for _, step := range dispositions.Steps {
		row, err := entryStepRowOf(held, entry, step, expansion)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
		if step.Provenance != (store.StepProvenance{}) {
			rows = append(rows, stepProvenanceRow(step.Step, step.Provenance))
		}
	}
	return rows, nil
}

// bindingOf is what the Step a Refusal cites was bound to, and two empty
// strings where the entry holds no file for it.
//
// The `step` a Refusal carries is an **artefact coordinate and never an
// execution fact** — the Step it names may have no file in the entry at all,
// every Refusal that declined before Step 1 leaving none (§7, ADR-0061) — so
// this is a lookup that is allowed to find nothing, and finding nothing is what
// §8 says a Refusal that reached no Step renders.
func bindingOf(dispositions store.Dispositions, step int) (operation, target string) {
	for _, record := range dispositions.Steps {
		if record.Step == step {
			return record.Operation, record.Target
		}
	}
	return "", ""
}

// entryRow is the header of one Journal entry: what its own run.json holds, and
// the account or accounts the entry carries of how it ended.
//
// It is the one row type this command mints, and it names its content rather
// than its position: it is the entry, not the header of a page (§8's rule for
// every type it fixes).
//
// **`outcome` and `ended_at` are the entry's own Run's**, present exactly where
// it wrote an outcome.json, and `closed_by` is every inference another Run drew
// about it. The four states §7 classifies an entry into are therefore read off
// which of the two stand: neither is open, the first alone is closed by its own
// Run, the second alone is reaped, and both together is contested — the same
// derivation the Store makes from the files present, with nothing minted beside
// it to carry the answer twice.
//
// `ended_at` is the owner's and never a closer's. A closing write's instant is
// on the **closing** Run's clock, so putting it here would invite the
// cross-entry subtraction §7 forbids; where a closer is the entry's only
// account, the instant is inside the `closed_by` member that names whose clock
// it is.
//
// `dry_run` is written always, `false` included — §7's one exception to the
// absence rule holding on the wire, because what a reader that takes its
// absence for `false` gets wrong is unrecoverable.
type entryRow struct {
	Type      string      `json:"type"`
	RunID     string      `json:"run_id"`
	Procedure string      `json:"procedure"`
	Trigger   triggerRow  `json:"trigger"`
	StartedAt string      `json:"started_at"`
	DryRun    bool        `json:"dry_run"`
	Outcome   string      `json:"outcome,omitempty"`
	EndedAt   string      `json:"ended_at,omitempty"`
	ClosedBy  []closerRow `json:"closed_by,omitempty"`
}

// triggerRow is the Trigger as the entry holds it: a mapping and never a
// composed string, four facts whose shape differs by executor not packing into
// one without a grammar and a parser, and a job URL carrying every separator
// such a packing would use (§9).
type triggerRow struct {
	Cause       string `json:"cause"`
	Executor    string `json:"executor"`
	Actor       string `json:"actor,omitempty"`
	Host        string `json:"host,omitempty"`
	ExecutorRun string `json:"run_id,omitempty"`
	Attempt     int    `json:"run_attempt,omitempty"`
	JobURL      string `json:"job_url,omitempty"`
}

// closerRow is one closing write: another Run's inference that this entry's Run
// had died, and the whole of what the stated line beneath the header renders.
//
// The Run is the file's name and not one of its members — a closing write
// carries none naming its author, its path being that member (§7, ADR-0076).
// `outcome` is what a closing write records the entry as, which §7 fixes at
// `failed` for every one of them; it is written rather than left implied
// because the page states it, and a fact the page states and the wire does not
// is the two surfaces disagreeing (ADR-0026).
type closerRow struct {
	RunID   string `json:"run_id"`
	Outcome string `json:"outcome"`
	Step    int    `json:"step,omitempty"`
	EndedAt string `json:"ended_at"`
}

// entryRowOf reads the header off the entry.
func entryRowOf(entry store.Entry) entryRow {
	row := entryRow{
		Type:      "entry",
		RunID:     entry.Run.String(),
		Procedure: entry.Procedure,
		Trigger: triggerRow{
			Cause:       string(entry.Trigger.Cause),
			Executor:    string(entry.Trigger.Executor),
			Actor:       entry.Trigger.Actor,
			Host:        entry.Trigger.Host,
			ExecutorRun: entry.Trigger.ExecutorRun,
			Attempt:     entry.Trigger.Attempt,
			JobURL:      entry.Trigger.JobURL,
		},
		StartedAt: store.InstantText(entry.StartedAt),
		DryRun:    entry.DryRun,
	}
	if entry.Owner.Outcome != "" {
		row.Outcome = string(entry.Owner.Outcome)
		row.EndedAt = store.InstantText(entry.Owner.EndedAt)
	}
	for _, closer := range entry.Closers {
		row.ClosedBy = append(row.ClosedBy, closerRow{
			RunID:   closer.Run.String(),
			Outcome: string(store.OutcomeFailed),
			Step:    closer.Step,
			EndedAt: store.InstantText(closer.EndedAt),
		})
	}
	return row
}

// Cells is empty: the header is a block of labelled values rather than a line
// of a table, and showPage writes it.
func (r entryRow) Cells() []string { return nil }

// entryStepRow is one Step of the entry: what the Step was, what became of it,
// and the three things a Disposition holds — the identities it concluded about,
// `hyper`'s own account of the work, and what an effectful call gave back where
// it did not give the ordinary answer (§7).
//
// **`records` carries the members and not a count**, under the key a Run's own
// `step` row writes a number to: one set in the two shapes the two surfaces are
// for (§8). It is a pointer to a slice because the three states are three: a
// Disposition carrying no set writes no key at all, a *ran* Step whose
// Expansion resolved to nothing writes `[]`, and everything else writes the
// members — and `[]` and the absence are the distinction §7's own exception to
// the absence rule exists to keep.
//
// `unchanged_since` names the Run the members were resolved from, and stands
// exactly where the entry holds a digest and no members of its own (§7,
// ADR-0055). It is what keeps `show` from presenting a set this entry does not
// hold as though it did: another entry's bytes are read only by saying so.
//
// `selector`, `pattern` and `answered` are the blocks the Step file itself
// holds, in the Store's own member order and under the Store's own names. One
// reading rather than two that have to agree (ADR-0026); `records` is the one
// member here that is renamed, and §8 is what renames it.
//
// `selector` is written under `--expansion` and nowhere else. It is the
// Refusal footer's destination — §8's `bound-exceeded` page points a reader at
// `hyper show <run-id> --expansion` — and it is behind a parameter because an
// Expansion of five hundred members is the whole of what a Step reached and
// almost never what a reader of a Disposition came for.
type entryStepRow struct {
	Type                 string       `json:"type"`
	Step                 int          `json:"step"`
	ID                   string       `json:"id,omitempty"`
	Path                 string       `json:"path,omitempty"`
	Definition           string       `json:"definition,omitempty"`
	Operation            string       `json:"operation,omitempty"`
	Provider             string       `json:"provider,omitempty"`
	Target               string       `json:"target,omitempty"`
	Kind                 string       `json:"kind,omitempty"`
	Disposition          string       `json:"disposition"`
	StartedAt            string       `json:"started_at,omitempty"`
	EndedAt              string       `json:"ended_at,omitempty"`
	Records              *[]string    `json:"records,omitempty"`
	UnchangedSince       string       `json:"unchanged_since,omitempty"`
	Selector             *selectorRow `json:"selector,omitempty"`
	Pattern              *patternRow  `json:"pattern,omitempty"`
	Answered             *answeredRow `json:"answered,omitempty"`
	ProjectionFailedPath string       `json:"projection_failed_path,omitempty"`
}

// selectorRow is the Step's `over:` as authored beside what it expanded to.
//
// `declared` rides as the canonical bytes the entry holds for it rather than as
// a second reading of the same fact, which is what keeps the wire and the file
// from disagreeing about a selector (§7, ADR-0026).
//
// **`expanded_to` comes back in Expansion order and is never sorted.** It is a
// sequence and not a result set being ranged over, so reversing or sorting it
// would reorder events rather than facts: on a serial `destroy` the halt point
// is legible by position and nowhere else (§6, §9, ADR-0044). It is written
// whenever a selector exists, the empty list included.
type selectorRow struct {
	Declared   json.RawMessage `json:"declared"`
	ExpandedTo []string        `json:"expanded_to"`
	Bound      int             `json:"bound,omitempty"`
}

// patternRow is hyper's own account of the work, supplied by no Provider: a
// retry's attempts, a paginated read's pages, a poll's iterations. It is what
// makes a Step that took four minutes legible as four minutes of something
// (§7, ADR-0018).
type patternRow struct {
	Attempts int `json:"attempts,omitempty"`
	Pages    int `json:"pages,omitempty"`
	Polls    int `json:"polls,omitempty"`
}

// answeredRow is what an effectful call gave back where it did not give the
// ordinary answer: the host reached and the status got, or the command run and
// the code it exited with.
//
// Its presence is the fact that something other than the ordinary answer
// decided the Step, and it is effectful-only — a read's status is the answer,
// and the answer is in the Record wherever the Manifest projected it (§7,
// ADR-0050). The status is absent where no response arrived at all, which is
// the Step whose request provably never left.
type answeredRow struct {
	Host     string `json:"host,omitempty"`
	Status   *int   `json:"status,omitempty"`
	Command  string `json:"command,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

// entryStepRowOf is one Step record as a row, resolving the identity set where
// the entry holds a digest and no members.
func entryStepRowOf(held *store.Store, entry store.Entry, step store.StepFile, expansion bool) (entryStepRow, error) {
	row := entryStepRow{
		Type:                 "step",
		Step:                 step.Step,
		ID:                   step.ID,
		Path:                 step.Path,
		Definition:           step.Definition,
		Operation:            step.Operation,
		Provider:             step.Provider,
		Target:               step.Target,
		Kind:                 string(step.Kind),
		Disposition:          string(step.Disposition),
		StartedAt:            store.InstantText(step.StartedAt),
		EndedAt:              store.InstantText(step.EndedAt),
		ProjectionFailedPath: step.ProjectionFailedPath,
	}
	if step.StartedAt.IsZero() {
		// A closing write does not know when the Step began and writes
		// no `started_at` at all, which is an honest absence rather
		// than the year 1 (§7, ADR-0076).
		row.StartedAt = ""
	}
	if step.Identities.Digest != "" {
		members, from, err := resolveMembers(held, entry, step)
		if err != nil {
			return entryStepRow{}, err
		}
		row.Records = &members
		if from != entry.Run {
			row.UnchangedSince = from.String()
		}
	}
	if pattern := (patternRow{Attempts: step.Pattern.Attempts, Pages: step.Pattern.Pages, Polls: step.Pattern.Polls}); pattern != (patternRow{}) {
		row.Pattern = &pattern
	}
	if answered := answeredRowOf(step.Answered); answered != nil {
		row.Answered = answered
	}
	if expansion && step.Selector.Declared != nil {
		expanded := step.Selector.ExpandedTo
		if expanded == nil {
			expanded = []string{}
		}
		row.Selector = &selectorRow{
			Declared:   wireValue(step.Selector.Declared),
			ExpandedTo: expanded,
			Bound:      step.Selector.Bound,
		}
	}
	return row, nil
}

// answeredRowOf is the Capability's own answer as a row, and nil where the call
// gave the ordinary answer or the Step made none.
func answeredRowOf(answered store.Answered) *answeredRow {
	switch answer := answered.(type) {
	case store.HTTPAnswer:
		row := answeredRow{Host: answer.Host}
		if status, arrived := answer.Status.Code(); arrived {
			row.Status = &status
		}
		return &row
	case store.ShellAnswer:
		row := answeredRow{Command: answer.Command}
		if code, arrived := answer.ExitCode.Code(); arrived {
			row.ExitCode = &code
		}
		return &row
	}
	return nil
}

// resolveMembers is the identity set this Step concluded about, and the Run the
// members came back from.
//
// **Where the entry holds them there is no walk at all.** That is the ordinary
// case and it costs one file — the one already in hand — where a walk would
// cost a listing of the branch per Step. What the shortcut skips is nothing:
// the decoder already holds an identity set to being the sorted set its own
// digest is taken over, so a file that disagrees with itself fails on the way
// in rather than here (§7).
//
// Where the entry holds a digest and no members the set did not move since the
// Run that last carried one (ADR-0055), and the walk reads that Run's bytes —
// which `show` renders as *unchanged since* it, another entry's bytes being
// read only by saying so (§9, ADR-0026).
//
// **The walk is filtered exactly as the digest was computed**, and it has to
// be: the comparand a Run writes is the digest of the last Run **of this
// Procedure, and of this invocation chain, that was not a rehearsal** in which
// this Step carried a set (internal/run's previousDigest). A walk here that
// kept an entry that one skipped would reach a different digest and report the
// Store as disagreeing with itself.
//
// It also begins at the entry in hand rather than at the newest entry the
// branch holds. `Scan` walks the whole Journal newest-first, so an entry read
// back a month later would otherwise start at a Run whose digest has since
// moved — which is the same contradiction from the other end. The entry in hand
// is found by its Run id and yielded first, whatever it holds, because a
// rehearsal's own entry is a legitimate thing to ask about and the filter below
// would drop it.
func resolveMembers(held *store.Store, entry store.Entry, step store.StepFile) ([]string, store.RunID, error) {
	if step.Identities.Members != nil {
		return step.Identities.Members, entry.Run, nil
	}
	scan := func(yield func(store.Evidence, error) bool) {
		reached := false
		for evidence, err := range held.Scan(step.ID) {
			if err != nil {
				yield(store.Evidence{}, err)
				return
			}
			if !reached {
				if evidence.Entry.Run != entry.Run || evidence.Step.Step != step.Step {
					continue
				}
				reached = true
			} else if evidence.Entry.DryRun ||
				evidence.Entry.Procedure != entry.Procedure ||
				evidence.Step.Path != step.Path {
				continue
			}
			if !yield(evidence, nil) {
				return
			}
		}
	}
	return store.ReadIdentitySet(step.ID, scan)
}

// stepFileProvenanceRow is one Step's Provenance as the entry holds it, and the
// `step` it carries is what tells it from the Run's row on the wire (ADR-0043).
//
// It builds `run`'s own row type rather than a second one, so that the
// Provenance a Run reports as it happens and the Provenance an entry reports
// when it is read back cannot come out in two shapes (ADR-0026).
func stepFileProvenanceRow(step store.StepFile) provenanceRow {
	position := step.Step
	return provenanceRow{
		Type:               "provenance",
		Step:               &position,
		DefinitionRevision: step.Provenance.DefinitionRevision,
		ManifestDigest:     step.Provenance.ManifestDigest,
		OriginDigest:       step.Provenance.OriginDigest,
	}
}

// Cells is empty: a Step is a block of labelled values here rather than a line
// of a table, an identity set of five hundred members having no column to sit
// in. What tabulates a Step is `run`'s own page, where the cell is a count
// (§8).
func (r entryStepRow) Cells() []string { return nil }

// showPage is `show`'s page: the entry's header, the stated line the entry's
// account earns, the Refusal its own Run recorded, and then one block per Step
// record — each headed by the Step it is, and each carrying that Step's own
// Provenance beneath it.
//
// It is blocks rather than a table because what a Step carries here is the
// identity set in full, and an Expansion of five hundred members is not a cell.
// The page reads its blocks off the rows rather than off the entry, which is
// ADR-0026's rule as `run`'s page already applies it: a block the page draws
// and a row the stream emits come from one reading of what happened.
func showPage(w io.Writer, rows []render.Row) error {
	blocks := []showBlock{}
	steps := 0
	for _, row := range rows {
		switch held := row.(type) {
		case entryRow:
			// The Run-wide Provenance stands inside the header
			// block rather than after it: §9 puts the Run's
			// Provenance beside the Run, and a block of its own
			// under a heading of its own would be a second header
			// for one entry.
			values := entryValues(held)
			values = append(values, provenanceValues(runProvenanceOf(rows))...)
			blocks = append(blocks, showBlock{values: values})
			for _, line := range accountLines(held) {
				blocks = append(blocks, showBlock{heading: line})
			}
		case entryStepRow:
			steps++
			values := stepValues(held)
			values = append(values, provenanceValues(provenanceOfStep(rows, held.Step))...)
			blocks = append(blocks, showBlock{heading: stepHeading(held), values: values})
		}
	}
	if steps == 0 {
		// An entry holding no Step record at all — a Run that Refused
		// before Step 1, or one that went quiet before it wrote a file.
		// The absence is the most important thing on the page and an
		// empty page cannot carry it, which is the reading `run`'s own
		// *nothing ran* line already takes (§8).
		blocks = append(blocks, showBlock{heading: noStepRecord})
	}
	if err := writeShowBlocks(w, blocks); err != nil {
		return err
	}

	// The Refusal, beneath everything the entry says about itself, in the
	// problem table `check` already renders. §8's caret excerpt and its
	// `EDIT ONE OF` table are milestone 8's own renderer and land with the
	// ticket that builds them, which is the same deferral gate.go and
	// run.go both state — and this is the second site that inherits it.
	refusals := rowsOf[refusalRow](rows)
	if len(refusals) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return render.WriteTable(w, checkColumns, refusals)
}

// showBlock is one block of the page: a heading, a run of labelled values, or
// both. A stated line is a block with a heading and nothing under it, which is
// what it is — a line above values that turned out to be none.
type showBlock struct {
	heading string
	values  []labelledValue
}

// writeShowBlocks writes the blocks in order, a blank line between them, with
// **one label column across the whole page**.
//
// The alignment is the page's rather than each block's, which is why this is
// here and not writeLabelledValues: a column that narrows halfway down because
// one Step's block happens to carry shorter labels reads as a renderer that
// broke, and the blocks of one entry are one document rather than several
// pages that happen to be adjacent.
//
// A value a row does not carry writes no line at all, which is the ordinary
// absence rule the wire applies to the same member — a page carrying a label
// against nothing would state a claim the entry never made (§7, ADR-0064). No
// line ends in padding: a value is its line's last cell.
func writeShowBlocks(w io.Writer, blocks []showBlock) error {
	width := 0
	for _, block := range blocks {
		for _, stated := range block.values {
			if stated.value != "" {
				width = max(width, len(stated.label))
			}
		}
	}

	for i, block := range blocks {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if block.heading != "" {
			if _, err := fmt.Fprintln(w, block.heading); err != nil {
				return err
			}
		}
		for _, stated := range block.values {
			if stated.value == "" {
				continue
			}
			padding := strings.Repeat(" ", width-len(stated.label)+labelGutter)
			if _, err := fmt.Fprintln(w, stated.label+padding+stated.value); err != nil {
				return err
			}
		}
	}
	return nil
}

// labelGutter is the space between the label column and the values, and it is
// the two spaces tabwriter leaves everywhere else in this package: one page
// aligned by hand beside a dozen aligned by a writer should not be legible as
// the odd one out.
const labelGutter = 2

// accountLines is what the entry's own account earns beneath the header: one
// stated line per closing write, and one line for an entry that holds no
// account at all.
//
// The three states it renders are §7's four less the ordinary one. An entry
// closed by its own Run says so in its `OUTCOME` and needs no line; an **open**
// entry has no outcome to write there and its absence alone would read as a
// renderer that dropped a row, so the state is named — *open* being a state and
// not a fourth member of §12's triple, which is exactly why it may not go in
// that slot (§7, §9). A **reaped** or **contested** entry states the inference
// in full, one line per `closed-by/` file.
func accountLines(row entryRow) []string {
	if row.Outcome == "" && len(row.ClosedBy) == 0 {
		return []string{entryOpen}
	}
	lines := make([]string, 0, len(row.ClosedBy))
	for _, closer := range row.ClosedBy {
		lines = append(lines, closedByLine(closer, row.Outcome != ""))
	}
	return lines
}

// entryOpen is what an entry holding neither an outcome.json nor a closing
// write says: the Run may be in flight or its process may be gone, and `hyper`
// never guesses which (§7).
const entryOpen = "open — this entry holds no account of how it ended."

// noStepRecord is what stands where an entry holds no Step file: the Run
// reached no Step, or reached one and never wrote it (§7).
const noStepRecord = "this entry holds no step file."

// entryValues is the header's labelled values, in the row's own member order —
// the wire's order, so a reader moving between the page and the stream reads
// the same facts in the same sequence.
//
// A rehearsal writes a line and an ordinary Run writes none, which is the page
// half of §7's exception: the marker is written always on the wire, because a
// reader that takes its absence for `false` cannot recover, and on the page it
// is a word that stands where there is something to say — the reading §8's own
// terminal line already takes for the same marker.
func entryValues(row entryRow) []labelledValue {
	values := []labelledValue{
		{"ENTRY", row.RunID},
		{"PROCEDURE", row.Procedure},
		{"CAUSE", row.Trigger.Cause},
		{"EXECUTOR", row.Trigger.Executor},
		{"ACTOR", row.Trigger.Actor},
		{"HOST", row.Trigger.Host},
		{"EXECUTOR RUN", row.Trigger.ExecutorRun},
		{"ATTEMPT", numberText(row.Trigger.Attempt)},
		{"JOB URL", row.Trigger.JobURL},
		{"STARTED", row.StartedAt},
	}
	if row.DryRun {
		values = append(values, labelledValue{"REHEARSAL", "yes"})
	}
	return append(values,
		labelledValue{"OUTCOME", row.Outcome},
		labelledValue{"ENDED", row.EndedAt})
}

// closedByLine is the stated line one closing write earns, in the form §8's
// Comparison header uses: *contested — Run `<id>` recorded this entry `failed`
// at `<instant>`*.
//
// The prefix stands exactly where the entry also holds its own Run's account,
// which is what a contest is: two accounts of one Run's end, both standing
// (§7, ADR-0076). Where a closing write is the entry's only account there is no
// disagreement to name, and the line states the inference alone.
//
// Nothing is minted for it. The Run is the file's name, the instant is its
// `ended_at`, and `failed` is what §7 fixes a closing write records an entry as
// — putting any of it in the header's outcome would be this surface deciding
// between two accounts of what the world did, which §7 is precise that `hyper`
// does not do.
func closedByLine(closer closerRow, owned bool) string {
	line := fmt.Sprintf("Run %s recorded this entry %s at %s", closer.RunID, closer.Outcome, closer.EndedAt)
	if owned {
		return "contested — " + line
	}
	return line
}

// stepHeading is the line a Step's block is headed by: its position, what it is
// called, its Kind and its Disposition — the four cells §8's Step table renders
// and the one thing on this page that is not a labelled value.
//
// It names the Step as §8's table names it: the invocation chain where it was
// reached through a nested Procedure, the authored id where it sits at the top
// level. A closing write may resolve neither, and there the position is the
// whole of what is known (§7).
func stepHeading(row entryStepRow) string {
	parts := []string{}
	if named := namedStep(row.Path, row.ID); named != "" {
		parts = append(parts, named)
	}
	if row.Kind != "" {
		parts = append(parts, row.Kind)
	}
	parts = append(parts, row.Disposition)
	return fmt.Sprintf("STEP %d  %s", row.Step, strings.Join(parts, " · "))
}

// stepValues is one Step's labelled values, in the row's own member order.
func stepValues(row entryStepRow) []labelledValue {
	values := []labelledValue{
		{"DEFINITION", row.Definition},
		{"OPERATION", row.Operation},
		{"PROVIDER", row.Provider},
		{"TARGET", row.Target},
		{"STARTED", row.StartedAt},
		{"ENDED", row.EndedAt},
		{"RECORDS", recordsText(row)},
	}
	if row.Selector != nil {
		values = append(values,
			labelledValue{"SELECTOR", selectorText(row.Selector.Declared)},
			labelledValue{"EXPANDED TO", namesText(row.Selector.ExpandedTo)},
			labelledValue{"BOUND", numberText(row.Selector.Bound)})
	}
	if p := row.Pattern; p != nil {
		values = append(values,
			labelledValue{"ATTEMPTS", numberText(p.Attempts)},
			labelledValue{"PAGES", numberText(p.Pages)},
			labelledValue{"POLLS", numberText(p.Polls)})
	}
	if a := row.Answered; a != nil {
		values = append(values,
			labelledValue{"HOST", a.Host},
			labelledValue{"STATUS", answerText(a.Status)},
			labelledValue{"COMMAND", a.Command},
			labelledValue{"EXIT CODE", answerText(a.ExitCode)})
	}
	return append(values, labelledValue{"PROJECTION FAILED", row.ProjectionFailedPath})
}

// recordsText is what the `RECORDS` line renders: the members the Step
// concluded about, and where they were resolved from another entry, the Run
// that supplied them.
//
// The three states are three renderings. A Disposition carrying no set writes
// no line at all, which is the page's ordinary absence rule; a set that moved
// to empty renders `none`, because the empty line the members would leave is a
// label standing against nothing; and a set resolved from an earlier entry
// renders the members and says whose they are, `show` reading another entry's
// bytes only by saying so (§9, ADR-0026).
func recordsText(row entryStepRow) string {
	if row.Records == nil {
		return ""
	}
	text := namesText(*row.Records)
	if text == "" {
		text = "none"
	}
	if row.UnchangedSince != "" {
		text += " — unchanged since run " + row.UnchangedSince
	}
	return text
}

// selectorText is a selector in §8's own notation: the form heading it, then a
// ` · `-separated run of what that form carries — the literals as authored
// under `values:`, and one `field operator operand` conjunct per predicate,
// colons dropped and sorted by Unicode code point on the rendered text.
//
// It reads the canonical bytes the entry holds rather than an artefact at a
// revision, which is why a selector is readable back long after the Run without
// a checkout (§7). One notation and not two: what §8's `FROM` and `TO` state
// about a selector is what this states, since it is the same value read off a
// file rather than off two Runs.
func selectorText(declared json.RawMessage) string {
	var selector map[string]any
	if err := json.Unmarshal(declared, &selector); err != nil || len(selector) == 0 {
		return ""
	}
	form := slices.Sorted(maps.Keys(selector))[0]
	parts := []string{form}
	switch carried := selector[form].(type) {
	case []any:
		conjuncts := make([]string, 0, len(carried))
		for _, member := range carried {
			conjuncts = append(conjuncts, conjunctText(member))
		}
		if form == "values" {
			// A `values:` selector renders **as authored**: §6
			// orders an Expansion by the artefact where the
			// selector is a literal list, so sorting it would hide
			// which member a Run reaches first (§8).
			parts = append(parts, conjuncts...)
			break
		}
		parts = append(parts, slices.Sorted(slices.Values(conjuncts))...)
	}
	return strings.Join(parts, " · ")
}

// conjunctText is one member of a selector's list: a bare literal under
// `values:`, and `field operator operand` under the two predicate forms, with
// `exists` and `absent` rendering bare — their operand being the only one
// either takes (§8).
func conjunctText(member any) string {
	predicate, isMapping := member.(map[string]any)
	if !isMapping {
		return scalarText(member)
	}
	field, _ := predicate["field"].(string)
	words := []string{field}
	for _, key := range slices.Sorted(maps.Keys(predicate)) {
		if key == "field" {
			continue
		}
		words = append(words, key)
		if operand := scalarText(predicate[key]); operand != "" && operand != "true" {
			words = append(words, operand)
		}
	}
	return strings.Join(words, " ")
}

// scalarText is a JSON scalar as the page writes it, and the empty string for
// anything composed — nothing composed goes out composed, and a nested shape on
// this line would be a notation this chapter does not have (ADR-0059).
func scalarText(scalar any) string {
	switch value := scalar.(type) {
	case string:
		return value
	case bool:
		if value {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return value.String()
	}
	return ""
}

// namesText is a set or a sequence of names as one line: ` · `-separated, in
// the order it was handed over, which for `expanded_to` is Expansion order and
// not a sort (§6, ADR-0044).
func namesText(named []string) string { return strings.Join(named, " · ") }

// numberText is a number as a page cell, and the empty string at zero — the
// absence rule holding on the page for the members whose zero value is their
// absence in the entry (§7).
func numberText(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// answerText is a number a call may not have given back at all: the value where
// one arrived, and nothing where none did. It is a pointer rather than a zero
// because 0 is an answer a shell command gives — the ordinary one — and reading
// an unset field as one is how a request that never left acquires an exit code
// (§7).
func answerText(answer *int) string {
	if answer == nil {
		return ""
	}
	return strconv.Itoa(*answer)
}

// provenanceValues is a Provenance row's labelled values, in the row's own
// member order. The two scopes share one rendering because they share one row
// type: a member with no value at a level writes no line, which is the same
// absence rule that leaves it off the wire (§7, ADR-0043).
//
// **Every revision and every digest renders whole**, which is `providers`' own
// reading of ADR-0047 rather than a departure from it: the abbreviation is for
// a fact to be *recognised* — a revision the eye matches against the one beside
// it, in the Comparison's header and down a `runs` column — and there is no
// second value on this page to match one against. What a reader does with a
// revision here is `git show` it and with a digest is `sha256sum` a file, and
// neither is a thing an ellipsis can be pasted into (§8, §9).
//
// A revision supplied by an entry that recorded `repo_dirty` renders with a `+`
// suffix. The bytes that Run read are not the bytes at that revision and are
// nowhere in git, and the marker is what stops the page asserting otherwise
// (§8, §7).
func provenanceValues(row provenanceRow) []labelledValue {
	revision := row.RepoRevision
	if revision != "" && row.RepoDirty {
		revision += "+"
	}
	return []labelledValue{
		{"HYPER", row.HyperVersion},
		{"PROCEDURE REVISION", row.ProcedureRevision},
		{"REPOSITORY REVISION", revision},
		{"DEFINITION REVISION", row.DefinitionRevision},
		{"MANIFEST DIGEST", row.ManifestDigest},
		{"ORIGIN DIGEST", row.OriginDigest},
	}
}

// runProvenanceOf is the Run-wide Provenance row among the rows: the one
// carrying no `step`, which is the split the wire already states (ADR-0043).
func runProvenanceOf(rows []render.Row) provenanceRow {
	for _, row := range rows {
		if provenance, is := row.(provenanceRow); is && provenance.Step == nil {
			return provenance
		}
	}
	return provenanceRow{}
}

// provenanceOfStep is the Provenance row belonging to the Step at position, and
// the zero value where that Step has none — a Step whose record is a closing
// write rather than a file of its own (§7, ADR-0076).
//
// It matches on `step` rather than on the row that happens to follow, because
// `step` is what tells the two scopes apart on the wire already (ADR-0043):
// reading the pairing off the discriminator the stream states is one fact read
// once, where reading it off adjacency would be a second, unstated contract
// between the builder and the page.
func provenanceOfStep(rows []render.Row, position int) provenanceRow {
	for _, row := range rows {
		if provenance, is := row.(provenanceRow); is && provenance.Step != nil && *provenance.Step == position {
			return provenance
		}
	}
	return provenanceRow{}
}
