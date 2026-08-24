package cli

import (
	"fmt"
	"io"
	"strconv"
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
// two Runs are being compared, everything the header says about them, and the
// two Record tables beneath it (issues #167 and #170).
//
// **`THE CODE MOVED` is still absent, and so is `TOTALS`' last segment.** Both
// are [#171](https://github.com/TheLoomLabs/hyper/issues/171)'s, and until they
// land the surface renders nothing where the third table will sit rather than
// an empty header, which is the deferral convention `review`'s own absent range
// followed for five milestones.
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
// **`--target`, `--kind` and `--limit` narrow the two Record tables and never
// the header.** The header orients a reader between two Runs, and a narrowing
// that cut it would answer a question about a window the caller did not ask
// about (§8). `--kind` is validated at the flag rather than at the table, which
// is where a third name becomes a usage error instead of an empty answer, and
// the cap is applied to one row list before either rendering (ADR-0026).
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
	rows, err := comparisonRows(held, windows, changeNarrowingsOf(parsed))
	if err != nil {
		return reportReadStoreFault(changesCommand, stderr, err)
	}
	kept, left := cutChangeRows(rows, parsed.limit)

	// The two renderings are one list of rows written twice (ADR-0026), and
	// the cut is applied before either of them: the page and the --json
	// stream state the same facts because they are built from one row set,
	// cut in one place.
	page := func(w io.Writer, rows []render.Row) error { return changesPage(w, rows, noWindowLine(named, parsed)) }
	if code := writeAnswer(changesCommand, stdout, stderr, parsed.json, kept, changesTerminal(left), page); code != 0 {
		return code
	}

	// The human counterpart of the marker is a line on stderr, in both
	// modes, because it is narration rather than an answer (§9). A truncated
	// result must never look complete, and a table that simply stopped after
	// the last row it was allowed would be one.
	if left.dropped > 0 {
		fmt.Fprintf(stderr, "hyper %s: %s\n", changesCommand, truncationLine("Records", left.returned, left.returned+left.dropped, parsed, changesNarrowing))
	}
	return ExitClean
}

// changesNarrowing is what a truncated Comparison says to do next: the two
// parameters that narrow the identity axis, which is the axis the two Record
// tables order on and therefore the one a cap cuts.
//
// It is the command's own words rather than the renderer's, because the
// parameters that narrow an axis differ by which command was called — a caller
// pointed at `--definition` here would go looking for a flag `changes` does not
// take (§9, §12, ADR-0065).
//
// `--since` and `--between` are not among them, and their absence is the rule
// rather than an oversight: they move the **window**, which is which two Runs
// are compared, and what a cap cut here is Records inside one window.
const changesNarrowing = "narrow with --target or --kind"

// changeNarrowing is what the caller narrowed the two Record tables by: §9's
// two typed, closed parameters for this surface, read off the arguments once.
//
// They are one value rather than two reads of `commandArgs` because they are
// applied in two different places for one reason each, and a page that narrowed
// on one of them in only one of those places would report a Comparison the
// caller did not ask for.
type changeNarrowing struct {
	// target is `--target`, matched byte-exact over UTF-8 as every name
	// comparison in §9 is. It is a fact about the **identity**, so it is
	// spent before a series is read.
	target string
	// recordKind is `--kind`, one of §7's two Record types. It is a fact the
	// **version** carries — which of the two tables a row belongs to — so it
	// is spent over the rows rather than over the identities (§8, ADR-0025).
	recordKind store.RecordType
}

// changeNarrowingsOf reads the two off the arguments, which is where every
// other command in §9's Inspection section reads its own (recordNarrowingsOf).
func changeNarrowingsOf(parsed commandArgs) changeNarrowing {
	return changeNarrowing{target: parsed.target, recordKind: parsed.recordKind}
}

// keepsIdentity answers whether an identity survives `--target`.
func (n changeNarrowing) keepsIdentity(id store.Identity) bool {
	return n.target == "" || id.Target == n.target
}

// keepsRow answers whether a row survives `--kind`.
//
// A row's type is the Record type of the version standing at it, which is what
// the two tables are split by: `asset` keeps `YOU DID THIS` and `observation`
// keeps `THE WORLD MOVED` (ADR-0026). A row that is neither — the `window` row
// — is not what this narrows and is kept whatever was named.
func (n changeNarrowing) keepsRow(row render.Row) bool {
	change, is := tableRow(row)
	return !is || n.recordKind == "" || change.Type == string(n.recordKind)
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
// makes three: the Step records of every end, the branch's Record listing, and
// the content of the versions the endpoints landed on.
//
// **The Step records are read for every end now, where #167 read only a reaped
// one.** The two Record tables above are eligible over the identity sets those
// records carry (ADR-0058), so a Comparison of two ordinary Runs opens their
// Step files where it used to open none — and where a record carries a digest
// and no members, the set is read off the Run that last wrote them, which is a
// walk of the Journal (§7, ADR-0055, resolveMembers).
//
// **The versions are one listing for the invocation and one batch read per
// window.** `store.Store.Records` lists the branch once however many windows a
// fold renders, the Head derivation reading a whole series either way, and
// `store.Store.Contents` opens one window's endpoint versions in one go — so
// the cost is bytes rather than a subprocess per identity, which is the trade
// the Version/RecordVersion split exists for (§7).
func comparisonRows(held *store.Store, windows []compare.Window, narrowing changeNarrowing) ([]render.Row, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	for i := range windows {
		for _, side := range []*compare.Side{&windows[i].Baseline, &windows[i].Subject} {
			if err := readDispositions(held, side); err != nil {
				return nil, err
			}
		}
	}

	// One listing of the branch, however many windows this invocation
	// renders, and never one per identity: the Head derivation reads a
	// whole series either way, and a per-identity listing would be a
	// subprocess per Record (§7, store.Records).
	listed, err := held.Records()
	if err != nil {
		return nil, err
	}
	series := map[store.Identity]store.Series{}
	for _, record := range listed {
		series[record.Identity] = record
	}

	var rows []render.Row
	for _, window := range windows {
		records, err := comparisonRecords(held, window, series, narrowing)
		if err != nil {
			return nil, err
		}
		for _, row := range compare.Rows(window, records) {
			if narrowing.keepsRow(row) {
				rows = append(rows, row)
			}
		}
	}
	return rows, nil
}

// tableRow answers whether a row is one of the two Record tables' — and which
// table, the type being the split (§8, compare.ChangeRow).
//
// It is one predicate rather than a type assertion at each of the four sites
// that ask: the cut, the count, the rows one window's block renders and
// `--kind`. A stream carries the `window` row beside them and will carry
// #171's, so *is this a row a table holds* is a question this page asks
// repeatedly and must answer identically each time.
func tableRow(row render.Row) (compare.ChangeRow, bool) {
	change, is := row.(compare.ChangeRow)
	return change, is
}

// changeCut is what a cap left of the two Record tables: how many of their rows
// came back and how many it dropped.
//
// It is a value rather than two returns for `records`' own reason one file
// over: the same pair is read by the marker on the wire and by the line on
// stderr, and a caller recomputing one of them from the row list is a caller
// free to count something else (`cut`, records.go).
type changeCut struct{ returned, dropped int }

// cutChangeRows applies `--limit` to the two Record tables and to nothing else.
//
// The cap counts the rows of the tables, in the order the page renders them,
// and every `window` row survives it: a header is not a row a cap cuts, and a
// Comparison whose cap fell on its own headers would report a window it did not
// render (§8, §9). What that leaves is a page whose last window may render its
// tables empty where the cap fell before them, and the marker on the wire and
// the line on stderr are what say so — a truncated result must never look
// complete, and there is no cursor behind this stream (ADR-0065).
//
// A limit of none is the flag left off and cuts nothing, which is what every
// other cut on these surfaces means by it (truncate, cutIdentities).
func cutChangeRows(rows []render.Row, limit int) (kept []render.Row, left changeCut) {
	for _, row := range rows {
		if _, is := tableRow(row); !is {
			kept = append(kept, row)
			continue
		}
		if limit > 0 && left.returned == limit {
			left.dropped++
			continue
		}
		left.returned++
		kept = append(kept, row)
	}
	return kept, left
}

// changesTerminal is the row every stream here ends with: the marker where the
// cap cut the tables, and the bare `false` where it did not.
//
// The marker names the **identity** axis, which is the axis the two Record
// tables order on and therefore the one a cap cuts (§12, ADR-0065). It is the
// object shape and never the bare `true`: this command has parameters that
// narrow what it cut, so a boolean here would hand back a truncated result with
// no next question in it.
func changesTerminal(left changeCut) render.Row {
	if left.dropped == 0 {
		return render.NewResultRow(false)
	}
	return render.NewTruncatedResultRow(render.TruncationMarker{
		Axis:     render.AxisIdentity,
		Returned: left.returned,
		Dropped:  left.dropped,
		Hint:     changesNarrowing,
	})
}

// readDispositions fills one side's Step records, with every identity set
// resolved to its members.
//
// A set whose digest did not move is written in an earlier entry and reading it
// back is a walk of the Journal (§7, ADR-0055). `internal/compare` is handed
// the answer rather than the Store, so the walk happens here — the same walk
// `show` makes for the same member, through the same door.
func readDispositions(held *store.Store, side *compare.Side) error {
	if !side.Present {
		return nil
	}
	dispositions, err := held.Dispositions(side.Entry)
	if err != nil {
		return err
	}
	side.Steps = dispositions.Steps
	for i, step := range side.Steps {
		if step.Identities.Digest == "" || step.Identities.Members != nil {
			continue
		}
		members, _, err := resolveMembers(held, side.Entry, step)
		if err != nil {
			return err
		}
		side.Steps[i].Identities.Members = members
	}
	return nil
}

// comparisonRecords is one window's eligible identities with both ends read for
// each: the version standing at each end's instant, and what that version
// projected.
//
// **`--target` narrows here rather than over the rows**, being a fact about the
// identity and not about what happened to it: an identity another Target owns
// is one this call never reads a series for. `--kind` narrows over the rows,
// the Record type being a fact the version carries and not the identity
// (changeNarrowing).
func comparisonRecords(held *store.Store, window compare.Window, series map[store.Identity]store.Series, narrowing changeNarrowing) ([]compare.Record, error) {
	var records []compare.Record
	var wanted []store.Version
	for _, id := range compare.Eligible(window) {
		if !narrowing.keepsIdentity(id) {
			continue
		}
		standing := series[id]
		record := compare.Record{Identity: id}
		record.Baseline.Version, record.Baseline.Held = compare.Endpoint(window.Baseline, standing)
		record.Subject.Version, record.Subject.Held = compare.Endpoint(window.Subject, standing)
		if record.Baseline.Held {
			wanted = append(wanted, record.Baseline.Version)
		}
		if record.Subject.Held {
			wanted = append(wanted, record.Subject.Version)
		}
		records = append(records, record)
	}

	projected, err := projectedFields(held, wanted)
	if err != nil {
		return nil, err
	}
	for i, record := range records {
		if record.Baseline.Held {
			records[i].Baseline.Fields = projected[record.Baseline.Version.File]
		}
		if record.Subject.Held {
			records[i].Subject.Fields = projected[record.Subject.Version.File]
		}
	}
	return records, nil
}

// projectedFields is the content of every version named, keyed by the file it
// sits at.
//
// It is one batch read for the whole answer, so what an endpoint's `fields`
// costs is bytes and never a subprocess per row (store.Contents). The two ends
// of an unchanged Record are one version, and it is read once: the key is the
// file, which is what names a version to git.
func projectedFields(held *store.Store, versions []store.Version) (map[string]store.Mapping, error) {
	var wanted []store.Version
	seen := map[string]bool{}
	for _, version := range versions {
		if seen[version.File] {
			continue
		}
		seen[version.File] = true
		wanted = append(wanted, version)
	}

	read, err := held.Contents(wanted)
	if err != nil {
		return nil, err
	}
	projected := make(map[string]store.Mapping, len(read))
	for i, version := range read {
		projected[wanted[i].File] = version.Fields
	}
	return projected, nil
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
	for i, row := range rows {
		window, is := row.(compare.WindowRow)
		if !is {
			continue
		}
		if written > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := writeWindowBlock(w, window, changeRowsUnder(rows[i+1:])); err != nil {
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

// changeRowsUnder is the rows of one window's tables: everything up to the next
// window and no further.
//
// The stream's order is the page's, so a window's rows are the run that follows
// it — which is the contract §8 fixes rather than a consequence of how this
// happens to be built (ADR-0026).
func changeRowsUnder(rows []render.Row) []compare.ChangeRow {
	var under []compare.ChangeRow
	for _, row := range rows {
		change, is := tableRow(row)
		if !is {
			break
		}
		under = append(under, change)
	}
	return under
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

// writeWindowBlock writes one window: the Procedure it is of, the two Runs, one
// stated line per closing write beneath them, and the two Record tables.
//
// `THE CODE MOVED` and `TOTALS` are #171's, and until they land the block ends
// at the second table rather than at an empty header.
func writeWindowBlock(w io.Writer, window compare.WindowRow, rows []compare.ChangeRow) error {
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
	for _, table := range []struct {
		head string
		kind store.RecordType
	}{
		{youDidThis, store.RecordAsset},
		{theWorldMoved, store.RecordObservation},
	} {
		if err := writeChangeTable(w, table.head, table.kind, rowsOfKind(rows, table.kind)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, changesIndent+totalsLine(rows))
	return err
}

// totalsLine is what stands beneath a window's tables: how many rows are on the
// page above it.
//
// **It counts rows** (ADR-0058), so the line totals what a reader can see and a
// Record changed and destroyed in one window counts once. `changes` is the
// asset rows plus the observation rows; **the tombstone count is a subset of
// the asset count and is never added to it**, a Tombstone being a marker inside
// the Asset table rather than a class of its own (§7, §8, ADR-0033).
//
// **All four numbers render, `0` included**, so the line is scanned rather than
// parsed. There is no grand total across windows: a fold renders this line per
// block, and summing across windows with different baselines is the
// cross-Procedure reading the window rule refuses (§8).
//
// **The last segment is #171's.** It is a phrase and not a count — summing a
// classed fact, a repository revision and a line count into one integer is
// three incommensurable things under one head — and it arrives with the table
// it reports on, `TOTALS` gaining a segment rather than being rewritten.
//
// It has no row of its own on the wire. §8's stream carries the rows of the
// tables and the `window` row above them and no `totals` object, this line
// being those rows counted rather than a fact of its own — and a consumer
// counting what it was handed cannot disagree with the page about it
// (ADR-0026).
func totalsLine(rows []compare.ChangeRow) string {
	assets, observations, tombstones := 0, 0, 0
	for _, row := range rows {
		if row.Type != string(store.RecordAsset) {
			observations++
			continue
		}
		assets++
		if row.Change == string(compare.ChangeDestroyed) {
			tombstones++
		}
	}
	return fmt.Sprintf("%s%s%d changes%s%d asset%s%d observation%s%d tombstone",
		totalsLabel, totalsGap,
		assets+observations, factMemberGap,
		assets, factMemberGap,
		observations, factMemberGap,
		tombstones)
}

// rowsOfKind is one table's rows: the ones whose Record type names it.
func rowsOfKind(rows []compare.ChangeRow, kind store.RecordType) []render.Row {
	var held []render.Row
	for _, row := range rows {
		if row.Type == string(kind) {
			held = append(held, row)
		}
	}
	return held
}

// writeChangeTable writes one of the two Record tables: a blank line, the
// table's own head and count, and then the rows.
//
// **The head and the count render whether or not the table holds a row.** An
// absent block is ambiguous between *nothing to report* and *the renderer had
// nothing to say*, which is the ambiguity every named absence on these surfaces
// refuses to leave standing — and two zero-row tables is how a Run whose every
// Step skipped reads (§8).
//
// The column header stands over the rows and not over their absence, which is
// the renderer's own rule: a page whose rows contribute no line is written as
// nothing, header included, and what stands in its place is the command's own
// (render.WriteTable). Here that is the count line above, which has already
// said *none*.
func writeChangeTable(w io.Writer, head string, kind store.RecordType, rows []render.Row) error {
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s%s%s%s\n", changesIndent, head, changeTableGap, countOf(len(rows), string(kind))); err != nil {
		return err
	}
	return render.WriteTable(indented{w: w, indent: changesIndent}, changeColumns, rows)
}

// countOf is a count and the noun it counts, pluralised. Zero takes the plural,
// which is what English does and what keeps the line scanned rather than read.
func countOf(counted int, noun string) string {
	if counted == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(counted) + " " + noun + "s"
}

// indented is a writer that puts this page's left margin in front of every line
// written through it.
//
// It exists because `render.WriteTable` writes a table at no indent and this
// page is a block with a margin — the same margin `review` renders with, §8's
// other block-shaped page. The alternative is a second alignment path for one
// prefix, and one path from cells to bytes is what the renderer holds
// (ADR-0026).
type indented struct {
	w      io.Writer
	indent string
}

// Write puts the indent in front of the line and hands the rest on.
//
// It is called once per line by the renderer, which writes through
// `fmt.Fprintln` — a line at a time, terminated — so nothing here reassembles a
// line out of pieces. A write that is not a line is passed through unchanged
// rather than guessed at.
func (i indented) Write(p []byte) (int, error) {
	if len(p) == 0 || p[len(p)-1] != '\n' {
		return i.w.Write(p)
	}
	if _, err := io.WriteString(i.w, i.indent); err != nil {
		return 0, err
	}
	return i.w.Write(p)
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
	// youDidThis and theWorldMoved are the two Record tables' heads, one per
	// actor and Assets first. The split is by actor rather than by field,
	// and Asset against Observation is two tables rather than one column
	// with two values (§8, ADR-0026).
	youDidThis    = "YOU DID THIS"
	theWorldMoved = "THE WORLD MOVED"
	// totalsLabel and totalsGap are the line beneath the tables. The gap is
	// two spaces where a table's head takes three, which is §8's own
	// rendering: the label is shorter and the line is one segment run
	// rather than a head over a table.
	totalsLabel = "TOTALS"
	totalsGap   = "  "
	// changeTableGap stands between a table's head and its count. It is a
	// fixed gap and not an alignment: the two heads are different lengths
	// and their counts are not a column, each line being a sentence about
	// its own table.
	changeTableGap = "   "
	// baselineLabel and subjectLabel are the header's two rows, named once.
	// They are the label a line carries and the noun a stated line beneath
	// the header names it by, and two spellings of one label is where the
	// day comes that a contest line points at a row that is not there.
	baselineLabel = "BASELINE"
	subjectLabel  = "SUBJECT"
)

// changeColumns is the head of both Record tables: the row's own members in the
// row's own order. One head for the two tables, because one row type carries
// them and the split is by actor rather than by column (§8, compare.ChangeRow).
var changeColumns = []string{"CHANGE", "TARGET", "DEFINITION", "RECORD", "ORDINAL", "FIELDS"}
