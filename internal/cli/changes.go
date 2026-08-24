package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// changesCommand is the name this command's messages and its gate are spelled
// with.
const changesCommand = "changes"

// changesParameters is `changes`'s argument surface past the three globals:
// §9's typed, closed parameters for this command and nothing else. There is no
// predicate dialect over them and none behind them — a caller wanting an
// arbitrary filter takes the rows and applies it themselves (ADR-0013).
//
// `--procedure` is absent deliberately, and it is the one difference between
// this surface and `runs`'. The Procedure is **positional** here because it
// decides which rendering you get rather than filtering the rows of one:
// naming one selects it, and naming none compares across every Procedure at
// once (§9).
var changesParameters = parameters{
	limit:   defaultListLimit,
	since:   true,
	between: true,
	target:  true,
	kind:    true,
}

// RunChanges implements `hyper changes [procedure]` — §8's Comparison: which
// two Runs are being compared, and everything the header says about them
// (issue #167).
//
// **This ticket is the window and the header, and no table stands beneath them
// yet.** §8 requires three — `YOU DID THIS`, `THE WORLD MOVED` and `THE CODE
// MOVED` — and they arrive in the two tickets this one blocks:
// [#170](https://github.com/TheLoomLabs/hyper/issues/170) for the two Record
// tables and [#171](https://github.com/TheLoomLabs/hyper/issues/171) for the
// code facts, the catch-all and `TOTALS`. Until they land the surface renders
// nothing where they will sit rather than an empty header, which is the
// deferral convention `review`'s own absent range followed for five
// milestones.
//
// **The derivation is `internal/compare`'s and this is flags, the lookups and
// the page.** That package opens no file, starts no subprocess and reads no
// clock: it takes the two Journal entries and the Store reads those entries
// need, and answers the ordered row list `internal/render` writes twice.
//
// **The order past the gate is the positional and then the Store**, which is
// §9's general rule and the one `show` is the exception to: a Procedure name
// resolves against the working tree, so a typo is `2` on a repository with no
// Store at all, and `store-absent` Refuses `77` only once the name is known to
// be one (§9, ADR-0060).
//
// `--target` and `--kind` narrow the rows of the tables above. They are read
// and validated here — the set `--kind` closes is where a third name becomes a
// usage error rather than an empty answer — and they narrow nothing yet
// because there is nothing yet to narrow; the header they stand over is not
// theirs to cut. `--limit` is the same: it cuts the first N of §8's table
// ordering, and this milestone's fifth ticket emits no row that ordering
// ranges over, so every stream here terminates in the bare `false`.
//
// It is not a Run: it writes nothing, terminates its stream with `result`
// rather than `outcome`, and exits 0 whatever outcomes the entries it named
// record — the exit code is this invocation's and never any Run's.
func RunChanges(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
	parsed, code := parseArgs(changesCommand, args, changesParameters, process.LookupEnv, stderr)
	if code != 0 {
		return code
	}

	// The two ways of naming one window, and naming it both ways at once is
	// a usage error. It is decided from the argument list alone and so
	// stands before a repository root is resolved: there is nothing to load
	// before this invocation can be judged wrong. It carries **no
	// error_code** — an error_code names a check that declined an artefact,
	// and an argument list is not one (§9, §12, ADR-0060).
	if parsed.sinceNamed && parsed.betweenNamed {
		fmt.Fprintf(stderr, "hyper %s: --since and --between name one window two ways; give one of them\n", changesCommand)
		return ExitUsage
	}
	// At most one positional. **Naming none is not a usage error** — it is
	// the whole-Store mode, and naming nothing and fetching nothing stay
	// different things (ADR-0060) — so only a second name is a fault.
	if len(parsed.positional) > 1 {
		fmt.Fprintf(stderr, "hyper %s: %s\n", changesCommand, arityFault(parsed.positional, "Procedure"))
		return ExitUsage
	}
	named := ""
	if len(parsed.positional) == 1 {
		named = parsed.positional[0]
	}

	repoRoot, code := resolveRepoRoot(changesCommand, parsed.repoDir, process.LookupEnv, wd, stderr)
	if code != 0 {
		return code
	}
	if code := gateOnVersionPin(changesCommand, repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}
	if code := resolveProcedureName(named, repoRoot, stderr); code != 0 {
		return code
	}

	held, code := openStoreForReading(changesCommand, repoRoot, process.Now(), stderr)
	if code != 0 {
		return code
	}
	// One walk of the branch and no Step file with it. Which Runs a window
	// names is a question `run.json` and the accounts beside it answer in
	// full, and the Step files of every entry a window excluded are the bulk
	// this command would otherwise read to render nothing (store.Entries,
	// store.Listing).
	entries, err := held.Entries()
	if err != nil {
		return reportReadStoreFault(changesCommand, stderr, err)
	}

	windows, code := changeWindows(entries, parsed, named, stderr)
	if code != 0 {
		return code
	}
	rows, err := comparisonRows(held, windows)
	if err != nil {
		return reportReadStoreFault(changesCommand, stderr, err)
	}

	// The two renderings are one list of rows written twice (ADR-0026): the
	// page and the --json stream state the same facts because they are built
	// from one row set.
	page := func(w io.Writer, rows []render.Row) error { return changesPage(w, rows, noWindowLine(named, parsed)) }
	if code := writeAnswer(changesCommand, stdout, stderr, parsed.json, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}
	return ExitClean
}

// resolveProcedureName holds the positional to the namespace it resolves
// against: the Procedures in the working tree.
//
// A name matching nothing is a usage error at `2` carrying **no error_code**,
// writing nothing to stdout and opening no row stream at all — nothing was
// reviewed, so nothing refused, and there is no terminal row to be absent from
// (§9, ADR-0060). It suggests no near miss, for `review`'s own reason: a
// suggestion is a partial name resolved on the caller's behalf, and a human who
// accepts one reads a Comparison of something they did not type (ADR-0047).
//
// **It runs before the Store is opened**, which is what makes a typo `2` on a
// repository with no Store at all: the working-tree namespace needs nothing
// further to answer, and blaming the missing branch for a misspelt Procedure
// would point the remedy at `hyper store init` (§9).
//
// Naming nothing resolves nothing, and loads nothing: the whole-Store mode
// ranges over the Procedures the **Journal** holds, a Procedure deleted from
// the tree still having Runs that reached the world (ADR-0012).
func resolveProcedureName(named, repoRoot string, stderr io.Writer) int {
	if named == "" {
		return 0
	}
	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper %s: %s\n", changesCommand, err)
		return ExitUsage
	}
	if _, declared := loaded.Procedure(named); !declared {
		fmt.Fprintf(stderr, "hyper %s: no Procedure named %q in this repository\n"+
			"  procedures/ is that namespace, and hyper check walks it\n", changesCommand, named)
		return ExitUsage
	}
	return 0
}

// changeWindows is the windows this invocation renders: the two ends chosen by
// rule, or the two the caller named directly.
func changeWindows(entries []store.Entry, parsed commandArgs, named string, stderr io.Writer) ([]compare.Window, int) {
	if parsed.betweenNamed {
		window, code := betweenWindow(entries, parsed.between, named, stderr)
		if code != 0 {
			return nil, code
		}
		return []compare.Window{window}, 0
	}
	return compare.Select(entries, compare.Selection{
		Procedure:  named,
		Since:      parsed.since,
		SinceNamed: parsed.sinceNamed,
	}), 0
}

// betweenWindow is the window `--between <run-id> <run-id>` names: the two Runs
// read off the Journal in the order the header renders them, baseline first.
//
// Every way this can fail is a usage error at `2` with no `error_code`, because
// every one of them is a name that resolves to nothing a window can be made of
// (§9, ADR-0060). An id no entry carries and a partial id arrive at one
// message, nothing anywhere resolving a partial one (ADR-0047).
//
// **A rehearsal and an open entry are refused in their own words.** They are
// not one fault: a rehearsal is disqualified — letting one be a side would
// retire the warning a real Run earned — and an open entry is *not yet
// nameable*, an entry whose Run may be in flight and whose outcome the header
// would have to render without having one (§7, §8). A caller who named either
// is told which, because the remedies differ: one is a Run to re-run for real
// and the other is a Run to wait for.
//
// **Two Runs of two Procedures are refused too.** A Comparison is a window over
// one Procedure — the rule that keeps a monitoring Run from being compared
// against a provisioning one — so a pair spanning two names no window this
// surface can render, however well each half resolves.
func betweenWindow(entries []store.Entry, ids [2]string, named string, stderr io.Writer) (compare.Window, int) {
	var sides [2]store.Entry
	for i, id := range ids {
		found, held := entryNamed(entries, id)
		if !held {
			fmt.Fprintf(stderr, "hyper %s: no Journal entry for run %q in this repository's Store\n"+
				"  the %s branch is that namespace, and hyper runs enumerates it\n", changesCommand, id, store.BranchName)
			return compare.Window{}, ExitUsage
		}
		switch compare.StandingOf(found) {
		case compare.Rehearsal:
			fmt.Fprintf(stderr, "hyper %s: run %s is a rehearsal, which is neither side of a window — a --dry-run reaches no effect a Comparison reports\n", changesCommand, id)
			return compare.Window{}, ExitUsage
		case compare.Unclosed:
			fmt.Fprintf(stderr, "hyper %s: run %s holds no account of how it ended, so it is not yet an entry a window can name\n", changesCommand, id)
			return compare.Window{}, ExitUsage
		}
		sides[i] = found
	}
	if ids[0] == ids[1] {
		fmt.Fprintf(stderr, "hyper %s: --between names run %s twice; a window has two ends\n", changesCommand, ids[0])
		return compare.Window{}, ExitUsage
	}
	if sides[0].Procedure != sides[1].Procedure {
		fmt.Fprintf(stderr, "hyper %s: run %s is a Run of %s and run %s of %s; a window is over one Procedure\n",
			changesCommand, ids[0], sides[0].Procedure, ids[1], sides[1].Procedure)
		return compare.Window{}, ExitUsage
	}
	// The order is the header's — baseline, then subject — so a pair given
	// the other way round is refused rather than quietly reordered or
	// quietly rendered backwards. Reordering would compare two Runs the
	// caller did not ask for in the order they did not ask for; rendering
	// it would put a later Run in the column the page says is the earlier
	// one, on the surface whose whole job is *this differs from when we
	// last looked*.
	if sides[1].StartedAt.Before(sides[0].StartedAt) {
		fmt.Fprintf(stderr, "hyper %s: run %s started after run %s; --between names the baseline first and the subject second\n",
			changesCommand, ids[0], ids[1])
		return compare.Window{}, ExitUsage
	}
	if named != "" && sides[0].Procedure != named {
		fmt.Fprintf(stderr, "hyper %s: the positional names %s and both Runs are of %s; a window is over one Procedure\n",
			changesCommand, named, sides[0].Procedure)
		return compare.Window{}, ExitUsage
	}
	return compare.Window{
		Procedure: sides[0].Procedure,
		Baseline:  compare.Side{Present: true, Entry: sides[0]},
		Subject:   compare.Side{Present: true, Entry: sides[1]},
	}, 0
}

// entryNamed is the entry one id names, and whether the Journal holds it. The
// comparison is byte-exact over the id as it is written, which is the only form
// there is: nothing anywhere resolves a partial one (ADR-0047).
func entryNamed(entries []store.Entry, id string) (store.Entry, bool) {
	for _, entry := range entries {
		if entry.Run.String() == id {
			return entry, true
		}
	}
	return store.Entry{}, false
}

// comparisonRows is every window's rows, one block after another in the order
// the windows were chosen.
//
// This is where the Store reads `internal/compare` is handed happen, and it
// makes exactly the one that is needed. A window's end is an instant, and on
// every entry that gave an account of its own end that instant is in the
// account: only a **reaped** end — one whose sole account is a `closed-by/`
// file — is the last Step file's `ended_at`, and that is the one end whose
// Step records have to be read (compare.Side.Instant).
//
// So a Comparison of two ordinary Runs opens no Step file at all. The tables
// #170 and #171 add read the identity sets off both ends and will widen this
// to every end; widening it now would pay a Journal's Step files to fill a
// member nothing on this page renders.
func comparisonRows(held *store.Store, windows []compare.Window) ([]render.Row, error) {
	var rows []render.Row
	for _, window := range windows {
		for _, side := range []*compare.Side{&window.Baseline, &window.Subject} {
			if !side.Present || !side.Reaped() {
				continue
			}
			records, err := held.Dispositions(side.Entry)
			if err != nil {
				return nil, err
			}
			side.Steps = records.Steps
		}
		rows = append(rows, compare.Rows(window)...)
	}
	return rows, nil
}

// changesPage is the Comparison's page: one block per window, and the sentence
// that stands where there is no window at all.
//
// The blocks are separated by a blank line and never summed. A fold across
// several Procedures renders one block each, with its own header, in
// Procedure-name order — and there is **no grand total**, which would sum
// across windows with different baselines (§8).
func changesPage(w io.Writer, rows []render.Row, noWindow string) error {
	written := 0
	for _, row := range rows {
		window, is := row.(compare.WindowRow)
		if !is {
			continue
		}
		if written > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := writeWindowBlock(w, window); err != nil {
			return err
		}
		written++
	}
	if written > 0 {
		return nil
	}
	_, err := fmt.Fprintln(w, noWindow)
	return err
}

// noWindowLine is what stands where no window could be named, and it says which
// of the two it is: a Store holding no Run of what was asked about, or a Store
// holding Runs and none inside the span `--since` bounded.
//
// A header over no Runs would read as a Comparison that found some, which is
// the ambiguity every named absence on these surfaces refuses to leave standing
// (§8). It is an answer and not an error: the name resolved, and fetching
// nothing is not naming nothing (ADR-0050, ADR-0060), so the exit code beside
// it is 0.
func noWindowLine(named string, parsed commandArgs) string {
	of := "no Run"
	if named != "" {
		of = "no Run of " + named
	}
	if parsed.sinceNamed {
		return of + " since " + store.InstantText(parsed.since)
	}
	return of + " in this repository's Store — the " + store.BranchName + " branch is that namespace"
}

// writeWindowBlock writes one window's header: the Procedure it is of, the two
// Runs, and one stated line per closing write beneath them.
func writeWindowBlock(w io.Writer, window compare.WindowRow) error {
	if _, err := fmt.Fprintln(w, changesIndent+window.Procedure); err != nil {
		return err
	}
	if err := writeHeaderLines(w, window); err != nil {
		return err
	}
	for _, line := range contestLines(window) {
		if _, err := fmt.Fprintln(w, changesIndent+line); err != nil {
			return err
		}
	}
	return nil
}

// writeHeaderLines writes the two lines that name the Runs, aligned against one
// another.
//
// They are aligned by hand rather than through render.WriteTable because this
// block has no column header: `BASELINE` and `SUBJECT` are the rows' own
// labels, and a table whose header row was one of its rows would be a header
// over one line of data (render.WriteTable). What the renderer holds is that
// the page and the wire come out of one row list, which they do.
//
// **Where there is no baseline the first line is the named state** — *no
// baseline — first Run of `<Procedure>`* — standing in the position the Run's
// facts would have taken, which is where `review`'s header already puts an
// absence: whitespace where a member was is omission wearing a rendering
// (§8, ADR-0068).
func writeHeaderLines(w io.Writer, window compare.WindowRow) error {
	lines := [][]string{}
	if window.Baseline != nil {
		lines = append(lines, append([]string{baselineLabel}, sideCells(window.Baseline)...))
	} else {
		lines = append(lines, []string{baselineLabel, "no baseline — first Run of " + window.Procedure})
	}
	lines = append(lines, append([]string{subjectLabel}, sideCells(window.Subject)...))
	return render.WriteAligned(w, changesIndent, lines)
}

// sideCells is one Run as the header renders it: its id, its Trigger, when it
// started, its outcome, how long it took and the `procedure_revision` it
// recorded (§8).
func sideCells(side *compare.SideRow) []string {
	return []string{
		abbreviatedRun(side.Run),
		side.Trigger,
		startedText(side.Started),
		side.Outcome,
		durationText(side),
		revisionCell(side),
	}
}

// startedText is when a Run started, as the header renders it: a weekday, a
// date and a time, on the one line whose job is orienting a reader between two
// Runs rather than supplying a value to paste back.
//
// The instant on the wire is the record's own spelling and this is the page's,
// which is one fact in two notations rather than two facts. It is read back off
// the row's own member so that the page cannot render an instant the stream did
// not carry (ADR-0026).
func startedText(instant string) string {
	started, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		return instant
	}
	return started.UTC().Format("Mon 2 Jan 15:04")
}

// durationText is how long a Run took, and the word `reaped` where no duration
// exists to render.
//
// A duration derives **within one entry**: the Run's start subtracted from the
// end its own `outcome.json` recorded, both on that Run's clock. Two entries'
// timestamps are never subtracted, and neither is a closing write's — that
// instant is the *closing* Run's on the closing Run's clock (§7).
//
// **A reaped entry's cell renders `reaped`**, which is the whole reason this is
// not a blank. A column whose every other cell is a duration and this one names
// why there is none is the same discipline the named absences elsewhere on
// these surfaces follow; a bare dash reads as a renderer that broke, and the
// outcome cell beside it says `failed` on plenty of Runs whose duration derives
// perfectly well (§8).
//
// It is derived off the row's own two instants rather than off the entry, so
// the page and the wire cannot come to disagree about what was subtracted: the
// wire carries the two instants and never the subtraction, §7's *times, not
// durations* holding on a stream as it holds in the Store.
func durationText(side *compare.SideRow) string {
	if side.Ended == "" {
		return changesReaped
	}
	// Both members were written by store.InstantText, so neither read can
	// fail on anything this command built. It is still not left to blank:
	// this is the one column where an empty cell reads as a renderer that
	// broke, which is the whole reason `reaped` is a word, so an instant
	// that would not read renders itself and the eye sees a fact rather
	// than a gap.
	started, startErr := time.Parse(time.RFC3339, side.Started)
	ended, endErr := time.Parse(time.RFC3339, side.Ended)
	if startErr != nil || endErr != nil {
		return side.Started + " → " + side.Ended
	}
	// Truncated to the second, which is the grain the page reads a Run at.
	// The record stamps milliseconds and rendering them here would put
	// three digits nobody compares between two Runs.
	return ended.Sub(started).Truncate(time.Second).String()
}

// revisionCell is the `procedure_revision` a Run recorded, named in full as a
// Procedure's.
//
// It is named rather than left as a bare revision because an unqualified `rev`
// sitting one table above a row reading `repository revision` is two facts
// inviting one reading, and it renders on **both** lines whether or not it
// moved — the header orienting a reader before they read a table that reports
// only what changed (§8).
//
// **A revision supplied by an entry that recorded `repo_dirty` renders with a
// `+` suffix.** The bytes that Run read are not the bytes at that revision and
// are nowhere in git, and the marker is what stops the header asserting
// otherwise (§7, §8). It is the page's notation for the `repo_dirty` the row
// carries, exactly as `changed` and `~` are one fact in two notations.
func revisionCell(side *compare.SideRow) string {
	if side.ProcedureRevision == "" {
		return ""
	}
	cell := "procedure rev " + abbreviatedRevision(side.ProcedureRevision)
	if side.RepoDirty {
		cell += "+"
	}
	return cell
}

// contestLines is what a contested or reaped end of the window earns beneath
// the header: one stated line per `closed-by/` file, baseline's first.
//
// **A contested entry's outcome and duration cells are the owner's,
// unqualified**, and the contest is a line rather than a cell value. Putting it
// in the outcome cell would be the surface deciding between two accounts of
// what the world did, which §7 is precise `hyper` does not do; putting it
// nowhere would be the tool holding a disagreement it never shows anyone.
//
// Nothing is minted for it. The Run is the file's name, the instant is its
// `ended_at`, and `failed` is what §7 fixes a closing write records an entry as
// — every fact read off the file's own keys.
//
// The one word this does not share with `show`'s line is the noun: there the
// page holds one entry and the line says *this entry*, and here it holds two,
// so it names which of the header's two labels the inference is about. A
// demonstrative with two candidates on the screen above it is the ambiguity the
// label removes.
func contestLines(window compare.WindowRow) []string {
	var lines []string
	for _, side := range []struct {
		label string
		row   *compare.SideRow
	}{{baselineLabel, window.Baseline}, {subjectLabel, window.Subject}} {
		if side.row == nil {
			continue
		}
		for _, closer := range side.row.ClosedBy {
			line := fmt.Sprintf("Run %s recorded the %s entry %s at %s", closer.Run, side.label, closer.Outcome, closer.Ended)
			if side.row.Ended != "" {
				line = "contested — " + line
			}
			lines = append(lines, line)
		}
	}
	return lines
}

const (
	// changesIndent is the left margin every line of this page carries. It
	// is `review`'s, which is the other surface §8 renders as a block rather
	// than as a table, and §8's own rendering of this page writes it.
	changesIndent = "  "
	// changesReaped is what stands in the duration cell of an entry whose
	// only account is a closing write: the word, and never a dash (§8).
	changesReaped = "reaped"
	// baselineLabel and subjectLabel are the header's two rows, named once.
	// They are the label a line carries and the noun a stated line beneath
	// the header names it by, and two spellings of one label is where the
	// day comes that a contest line points at a row that is not there.
	baselineLabel = "BASELINE"
	subjectLabel  = "SUBJECT"
)
