package cli

import (
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// runsCommand is the name this command's messages and its gate are spelled
// with.
const runsCommand = "runs"

// runsParameters is `runs`'s argument surface past the three globals: §9's four
// typed, closed parameters and the `--limit` every command in its Inspection
// section takes. There is no predicate dialect over them and none behind them —
// a caller wanting an arbitrary filter takes the rows and applies it themselves
// (ADR-0013).
var runsParameters = parameters{
	limit:     defaultListLimit,
	since:     true,
	procedure: true,
	target:    true,
	outcome:   true,
}

// RunRuns implements `hyper runs` — the Journal listed (issue #165).
//
// It is the surface that enumerates the namespace a `<run-id>` resolves
// against, which is what `show` points a caller at when an id matches nothing
// (§9). One row per Journal entry, and the row is §9's own seven facts: the Run
// id, when it started, its Trigger, its outcome, its Procedure, the Targets it
// bound and the version of `hyper` that performed it.
//
// **The Trigger is on every row**, being the only thing that distinguishes a
// world that has not changed from one nobody has looked at (§7).
//
// **The ordering is time, and time runs newest-first** (ADR-0065), on
// `started_at` with the `<run-id>` descending as the tie-break — §7's Head
// shape, a time key with a name behind it, and a UUIDv7 is total over the tie.
// The order is the Store's own (store.Entries) rather than one applied here:
// the reader that answers a listing is the one that orders it, so a Run's Steps
// and a Journal's entries cannot come to disagree about what *newest* means.
//
// Ordering on the Run's **start** rather than its end is also what gives an
// open entry a position like any other — an entry with no `outcome.json` still
// carries a `started_at`, so nothing here needs a rule for one.
//
// **This is the surface that pays the clock skew, and it pays it knowingly.** A
// Journal entry is a Run, and Runs have nothing to be ordered by except when
// they happened, so two machines whose clocks disagree by more than the gap
// between two Runs can list them in an order neither would agree with. `hyper`
// does not detect it, warn about it, or correct it: every remedy is a clock
// somebody else owns, and a listing that reordered itself on a guess would be
// evidence rearranged to look consistent.
//
// It is not a Run: it writes nothing, terminates its stream with `result`
// rather than `outcome`, and exits 0 whatever outcomes the entries it listed
// record — the exit code is this invocation's and never any Run's.
func RunRuns(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
	parsed, code := parseArgs(runsCommand, args, runsParameters, process.LookupEnv, stderr)
	if code != 0 {
		return code
	}
	// `runs` enumerates a namespace and resolves no name in one, so it
	// takes no positional at all: §9 gives a positional to nine of the
	// sixteen and this is not one of them.
	//
	// A name here has two readings — the parameter spelled without its
	// flag, and `run` with an `s` on it, which §9 names as the readability
	// wart the two commands sitting one letter apart buys — and the message
	// names the **parameter**. Pointing a caller at `hyper run` would be
	// this surface suggesting the one command in the tree that reaches the
	// world, on the strength of a guess about what they meant; pointing
	// them at `--procedure` suggests a listing, and a caller who wanted the
	// Run reads the same line and types the command they meant.
	if len(parsed.positional) > 0 {
		fmt.Fprintf(stderr, "hyper %s: takes no positional argument, got %s — the Procedure is a parameter here, --procedure %s\n",
			runsCommand, parsed.positional[0], parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot(runsCommand, parsed.repoDir, process.LookupEnv, wd, stderr)
	if code != 0 {
		return code
	}
	if code := gateOnVersionPin(runsCommand, repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	held, code := openStoreForReading(runsCommand, repoRoot, process.Now(), stderr)
	if code != 0 {
		return code
	}

	// One walk of the branch, and the three parameters that need nothing but
	// an entry handed down into it: a Target is a fact only a Step file
	// carries, and the Step files of the entries a window excluded are the
	// bulk this command would otherwise read to render nothing
	// (store.Listing). `--target` is the fourth and is applied over what
	// comes back, being the one parameter only those files can answer.
	narrowing := narrowingsOf(parsed)
	listed, err := held.Listing(narrowing.keeps)
	if err != nil {
		return reportReadStoreFault(runsCommand, stderr, err)
	}

	rows := journalRows(listed, narrowing)
	kept, dropped := truncate(rows, parsed.limit)

	// The two renderings are one list of rows written twice (ADR-0026), and
	// the cut is applied before either of them: the table and the --json
	// stream state the same facts because they are built from one row set,
	// cut in one place.
	page := func(w io.Writer, rows []render.Row) error { return runsPage(w, rows, narrowing.any()) }
	if code := writeAnswer(runsCommand, stdout, stderr, parsed.json, kept, runsTerminal(kept, dropped), page); code != 0 {
		return code
	}

	// The human counterpart of the marker is a line on stderr, in both
	// modes, because it is narration rather than an answer (§9). A
	// truncated result must never look complete, and a table that simply
	// stopped after the last row it was allowed would.
	if dropped > 0 {
		fmt.Fprintf(stderr, "hyper %s: %s\n", runsCommand, truncationLine("Runs", len(kept), len(rows), parsed, runsNarrowing))
	}

	return ExitClean
}

// runsNarrowing is what a truncated listing says to do next, in the words §9's
// own MCP sketch writes for this command: `--since` and `--target`, the two
// parameters that narrow what a cap on the time axis cut.
//
// It is the command's own words rather than the renderer's, because the
// parameters that narrow an axis differ by which command was called —
// `--between` is `changes`'s and nobody else's — and naming a flag the caller's
// command does not take would point the remedy at an argument they would go
// looking for in their own command line.
//
// There is no cursor behind this and no way to ask for the next N: the remedy
// for a truncated result is a narrower question, and this names the parameters
// that ask one (§9, ADR-0065).
const runsNarrowing = "narrow with --since or --target"

// runsTerminal is the row every stream here ends with: the marker where a cap
// cut the listing, and the bare `false` where it did not.
//
// The marker names the **time** axis, which is the axis this command orders on
// and therefore the one a cap cuts (§12, ADR-0065). It is the object shape and
// never the bare `true`: a namespace listing has no axis to name and this has
// parameters that narrow the one it cut, so a boolean here would hand back a
// truncated result with no next question in it.
func runsTerminal(kept []render.Row, dropped int) render.Row {
	if dropped == 0 {
		return render.NewResultRow(false)
	}
	return render.NewTruncatedResultRow(render.TruncationMarker{
		Axis:     render.AxisTime,
		Returned: len(kept),
		Dropped:  dropped,
		Hint:     runsNarrowing,
	})
}

// runRow is `runs`'s row, and its members are §9's own, in §9's order:
// {"type":"run","id":…,"started":…,"trigger":…,"outcome":…,"procedure":…,
// "targets":[…],"hyper_version":…}. §9 writes that shape out once and milestone
// 11's MCP tool reuses this contract rather than minting a second one, so the
// declaration order here is the wire's and not a preference.
//
// **`outcome` is absent on an open entry**, the member carrying that absence
// rather than a fourth value: *open* is a state and not a member of §12's
// triple, so writing it into a key named for the triple would relitigate that
// distinction by accident (§7, §9). A `started` beside an absent outcome is the
// whole of what the Store holds about a Run nobody has closed, and the row says
// exactly that much.
//
// **`contested` is the marker a contested entry carries**, and it stands beside
// the outcome rather than inside it. The key is named for the triple, the
// owner's account is a member of it, and a second account of the entry is not a
// fourth value either — so the cell states what the entry's own Run observed
// and this states that another Run drew an inference beside it. `hyper show` is
// where the contest is stated in full, one line per `closed-by/` file. It
// follows the ordinary absence rule: an entry holding one account carries no
// key at all.
//
// **`trigger` is the composed string and never the mapping**, which is §8's own
// window row and not a departure from `show`'s reading of the same fact. This
// is a Run summarised down a column — a clock or a person, which is the whole
// of what §7 says a Trigger distinguishes — where `show` reads one entry whole
// and there the four facts an executor writes are four members. One rendering
// per surface, and the surface that carries the parts is the one whose job is
// the parts.
//
// **`targets` is written always, the empty set included**, which is the one
// member here that departs from the ordinary absence rule — boundTargets below
// says why. Every other absent member of this row is a fact the entry does not
// carry.
//
// **`id` goes out whole here and abbreviated on the page**, like every other id
// on a table read down a column (ADR-0047): the wire abbreviates nothing
// anywhere, and what a consumer does with a Run id is hand it back to `hyper
// show`.
type runRow struct {
	Type         string   `json:"type"`
	ID           string   `json:"id"`
	Started      string   `json:"started"`
	Trigger      string   `json:"trigger"`
	Outcome      string   `json:"outcome,omitempty"`
	Contested    bool     `json:"contested,omitempty"`
	Procedure    string   `json:"procedure"`
	Targets      []string `json:"targets"`
	HyperVersion string   `json:"hyper_version"`
}

// Cells is the row's line on the page, in runsColumns' order — the row's own
// members, so what a consumer filters on and what a reader reads down are one
// list (§8, ADR-0026).
//
// Two of them render differently here and nowhere else: the id is abbreviated,
// and the contest is the word `yes` under a column named for it rather than the
// boolean the wire carries. Both are the page's reading of a fact the row holds
// once.
func (r runRow) Cells() []string {
	return []string{
		abbreviatedRun(r.ID),
		r.Started,
		r.Trigger,
		r.Outcome,
		yesCell(r.Contested),
		r.Procedure,
		namesText(r.Targets),
		r.HyperVersion,
	}
}

// runsColumns is the page's header: the row's own members in the row's own
// order.
var runsColumns = []string{"RUN", "STARTED", "TRIGGER", "OUTCOME", "CONTESTED", "PROCEDURE", "TARGETS", "HYPER"}

// journalRows is the answer: one row per Journal entry that survives the
// parameters, in the order the Store listed them.
//
// **The filters are conjunctive and every one of them is a typed, closed
// parameter.** There is no expression anywhere in them: a caller wanting an
// arbitrary predicate takes the rows and applies it themselves (§9, ADR-0013).
//
// Three of the four were spent on the way in, on the entries the Store opened
// Step files for; `--target` is spent here, being the one that needed those
// files to answer.
func journalRows(listed []store.Listed, narrowing narrowings) []render.Row {
	rows := make([]render.Row, 0, len(listed))
	for _, entry := range listed {
		if !narrowing.bound(entry.Targets) {
			continue
		}
		row := runRow{
			Type:         "run",
			ID:           entry.Run.String(),
			Started:      store.InstantText(entry.StartedAt),
			Trigger:      entry.Trigger.Text(),
			Contested:    entry.Account() == store.AccountContested,
			Procedure:    entry.Procedure,
			Targets:      boundTargets(entry.Targets),
			HyperVersion: entry.Provenance.HyperVersion,
		}
		// The entry's outcome is the owner's wherever one exists, a
		// reaped entry's is `failed`, and an open entry has none —
		// which is the Store's own derivation over the files present
		// (store.Entry.Outcome), read here rather than restated.
		if outcome, closed := entry.Outcome(); closed {
			row.Outcome = string(outcome)
		}
		rows = append(rows, row)
	}
	return rows
}

// boundTargets is the Targets a Run bound as the row carries them: the set, and
// the **empty** set where it bound none.
//
// It is written always, which is the one place this row departs from the
// ordinary absence rule, and §9 is what makes it depart: the row is enumerated
// there as *the Targets it bound* beside the six facts every Run has, and a Run
// that bound none has an answer rather than no answer. The absence rule is for
// a fact an entry does not carry, and every entry carries this one — a Refusal
// that declined before Step 1 bound nothing, and `[]` is what nothing is (§7,
// §9).
//
// The empty slice is minted rather than left nil because a nil slice encodes as
// `null`, which is the one thing a member of this stream may never be.
func boundTargets(bound []string) []string {
	if bound == nil {
		return []string{}
	}
	return bound
}

// narrowings is what the caller narrowed the listing by: §9's four typed,
// closed parameters, read off the arguments once.
//
// They are one value rather than four reads of `commandArgs` because two
// questions are asked of them — which entries survive, and whether anything was
// narrowed at all — and a fifth parameter answered by one and not the other
// would be a page saying *no Journal entry in this Store* about a Store holding
// plenty.
type narrowings struct {
	since      time.Time
	sinceNamed bool
	procedure  string
	target     string
	outcome    store.Outcome
}

// narrowingsOf reads the four off the arguments.
func narrowingsOf(parsed commandArgs) narrowings {
	return narrowings{
		since:      parsed.since,
		sinceNamed: parsed.sinceNamed,
		procedure:  parsed.procedure,
		target:     parsed.target,
		outcome:    parsed.outcome,
	}
}

// any answers whether the caller narrowed the listing at all, which is the
// whole difference between the two sentences an empty page writes.
//
// It is a comparison against the zero value rather than a test per member, so
// it enumerates nothing: a parameter added to this type is a parameter this
// answers, and the two questions asked of these four cannot come apart by
// somebody editing one of them.
func (n narrowings) any() bool { return n != narrowings{} }

// keeps answers whether one entry survives the three parameters an entry alone
// can answer. It is what the Store applies before it opens a Step file, so an
// entry outside the window costs its own account and nothing more.
//
// **`--since` is a lower bound and includes the instant it names**
// (withinWindow), so a timestamp copied off a `STARTED` cell selects the Run it
// was copied from.
//
// **`--procedure` matches byte-exact over UTF-8**, which is the comparison §9
// fixes for matching a name everywhere.
//
// **`--outcome` filters §12's triple and selects neither an open entry nor a
// contest by anything but its owner's outcome.** An open entry has no outcome
// at all, so nothing in the triple selects it; a contested entry has exactly
// one, its own Run's, which is the one the entry has (§7, §9).
func (n narrowings) keeps(entry store.Entry) bool {
	if !withinWindow(entry.StartedAt, n.since, n.sinceNamed) {
		return false
	}
	if n.procedure != "" && entry.Procedure != n.procedure {
		return false
	}
	if n.outcome != "" {
		outcome, closed := entry.Outcome()
		if !closed || outcome != n.outcome {
			return false
		}
	}
	return true
}

// bound answers whether an entry that bound these Targets survives `--target`.
//
// It is the fourth parameter and it stands apart from the three above because
// of what it costs: a Target is a fact only a Step file carries, so this is the
// one narrowing that cannot be spent before those files are read (§7, §9).
//
// The comparison is byte-exact over UTF-8, as `--procedure`'s is.
func (n narrowings) bound(targets []string) bool {
	return n.target == "" || slices.Contains(targets, n.target)
}

// runsPage is `runs`'s page: the table, or the sentence that stands where it
// has no rows.
//
// An empty table is written as nothing at all by the renderer, header included
// — what stands in its place is the command's own (§8) — and here it is a
// sentence naming what was looked for. A header over no rows would read as a
// listing that found some.
func runsPage(w io.Writer, rows []render.Row, narrowed bool) error {
	if len(rows) == 0 {
		if narrowed {
			_, err := fmt.Fprintln(w, "no Journal entry matched")
			return err
		}
		_, err := fmt.Fprintf(w, "no Journal entry in this repository's Store — the %s branch is that namespace\n", store.BranchName)
		return err
	}
	return render.WriteTable(w, runsColumns, rows)
}

// abbreviatedRun is a Run id as this page renders one: the leading digits, and
// an ellipsis saying the rest was cut.
//
// A Run id on a table read down a column is a fact to **recognise** — the eye
// matching one row against another — so it renders short here and whole on the
// wire, which is ADR-0047's rule and the same reading `review`'s revisions
// take. What the ellipsis is there to prevent is the id being retyped: nothing
// anywhere resolves a partial one, so a value that looked complete would be a
// value a caller hands to `hyper show` and gets *no Journal entry* for. What
// supplies an id whole is `--json`, or the terminal line of the Run that wrote
// it (§8, ADR-0047).
//
// The digits are the first two groups of the UUIDv7, which is its
// millisecond timestamp exactly — so what the eye matches on is the fact the
// column is ordered by.
func abbreviatedRun(id string) string {
	if len(id) <= runIDDigits {
		return id
	}
	return id[:runIDDigits] + "…"
}

// runIDDigits is how much of a Run id this page renders: the UUIDv7's first two
// groups, its 48-bit millisecond timestamp and the hyphen inside it.
const runIDDigits = 13
