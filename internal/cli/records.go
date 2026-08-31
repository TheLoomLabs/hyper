package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// recordsCommand is the name this command's messages and its gate are spelled
// with.
const recordsCommand = "records"

// recordsParameters is `records`'s argument surface past the three globals:
// §9's five typed, closed parameters and the `--limit` every command in its
// Inspection section takes. There is no predicate dialect over them and none
// behind them (ADR-0013).
var recordsParameters = parameters{
	limit:      defaultListLimit,
	since:      true,
	target:     true,
	definition: true,
	name:       true,
	history:    true,
}

// versionsPerSeries is what bounds one series under `--history`, and it is a
// constant this implementation picks rather than a fact anything reads back
// (§9).
//
// It exists because `--limit` cannot do this job. Under `--history` the limit
// counts **identities and not rows**: a series comes back whole or does not
// come back, a series cut partway through being a partial history wearing a
// complete one's shape. So the limit bounds how many Records answer and this
// bounds how much of one, and neither is the other wearing a different name.
//
// It is deliberately not a parameter. Widening it is not a narrower question,
// which is the only remedy this surface offers, and a flag behind it would be a
// caller asking for more of one series than the page can honestly carry. What a
// caller does when it cuts is read the Runs — a version is named by the Run
// that wrote it, and `hyper runs` and `hyper show` are that namespace (ADR-0049).
const versionsPerSeries = 20

// RunRecords implements `hyper records` — the surface whose job is finding a
// version (issue #166).
//
// `changes` reads a change and this finds the version that change is of. One
// row per Record: its identity, its ordinal, the Run **and** Step that wrote
// the version, whether that Run was a rehearsal, whether it is an Observation
// or an Asset, whether it is a Tombstone, which of its fields carry the
// presence-only secret marker, and its Provenance (§9).
//
// **The Run and the Step together are the version's identity, and this is the
// surface that carries them.** Two Steps of one Run writing one identity write
// two paths (§12), so the Run alone would not name one — which is a distinction
// only the command whose job is finding a version has to make.
//
// **The ordinal is rendered here and derived nowhere near here.** It is read
// off the Version the listing answered with, where it was built by the same
// ordering that derives the Head (store.order) — one derivation over the one
// listing, and never a second count taken at the surface, which is the whole
// reason a `records` ordinal and a Comparison's are the same number.
//
// It is each version's position in that ordering, stored nowhere, and never the
// version's identifier — which is the Run that wrote it (ADR-0011, ADR-0049).
// It is unstable under Compaction and under a laptop's Observations slotting in
// beneath a runner's, and that is affordable for exactly one reason: **nothing
// anywhere accepts an ordinal as input.**
//
// **The ordering is identity**: `(Target, Definition, name)`, each by Unicode
// code point, the columns read left to right — the one ordering `hyper` has
// over Record names, reused rather than stated a fourth time (ADR-0044). It is
// the Store's own (store.Records) rather than one applied here, for the reason
// `runs`'s is: the reader that answers a listing is the one that orders it.
//
// It is not a Run: it writes nothing, terminates its stream with `result`
// rather than `outcome`, and exits 0 whatever the Records it listed hold.
func RunRecords(args []string, to destination, process Process, wd, binaryVersion string) int {
	parsed, to, code := parseArgs(recordsCommand, args, recordsParameters, process.LookupEnv, to)
	if code != 0 {
		return code
	}
	// `records` ranges over a namespace and resolves no name in one, so it
	// takes no positional at all: §9 gives one to nine of the sixteen and
	// this is not one of them. The Record's name is a parameter here, and
	// the message names it — a caller who typed the name bare reads the
	// flag that takes it rather than a bare refusal.
	if len(parsed.positional) > 0 {
		fmt.Fprintf(to.narrate(), "hyper %s: takes no positional argument, got %s — a Record's name is a parameter here, --name %s\n",
			recordsCommand, parsed.positional[0], parsed.positional[0])
		return ExitUsage
	}
	// **`--since` is legal only with `--history`.** Without it the parameter
	// would filter Heads by when they last moved, which is a change read on
	// the command whose job is finding a version; and having it turn
	// `--history` on instead would be exactly the mode ADR-0013 refused. It
	// is a usage error carrying **no error_code**, like `--since` and
	// `--between` together on `changes`: an error_code names a check that
	// declined an artefact, and an argument list is not one (§9, ADR-0060).
	//
	// It is decided from the argument list alone and so stands beside the
	// positional, before a repository root is resolved: there is nothing to
	// load before this invocation can be judged wrong.
	if parsed.sinceNamed && !parsed.history {
		fmt.Fprintf(to.narrate(), "hyper %s: --since is legal only with --history — a Head has no window to bound, and naming one would not open a history\n", recordsCommand)
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot(recordsCommand, parsed.repoDir, process.LookupEnv, wd, to.narrate())
	if code != 0 {
		return code
	}
	if code, _ := gateOnVersionPin(recordsCommand, repoRoot, binaryVersion, to); code != 0 {
		return code
	}

	// The repository, for the one column the record cannot answer: an Asset
	// is **Orphaned** where the Definition that owned it no longer exists,
	// and what exists is a fact about the working tree rather than about the
	// branch (§7, ADR-0012). It is the only reason this command reads an
	// artefact at all.
	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", recordsCommand, err)
		return ExitUsage
	}

	held, code := openStoreForReading(recordsCommand, repoRoot, process.Now(), to)
	if code != 0 {
		return code
	}

	// One listing of the branch, and the narrowing applied over what came
	// back rather than over the paths it was found at. That is not a cost
	// left unpaid: the identity a version belongs to is **inside** the file
	// — the grammar truncates an over-long segment and suffixes a digest
	// (§12) — so a prefix listing would be filtering on the encoding rather
	// than on the names anybody wrote, and on a truncated segment it would
	// not even be a filter. It is the same reading that makes store.Records
	// order on the identity and never on the listing (ADR-0044).
	records, err := held.Records()
	if err != nil {
		return reportReadStoreFault(recordsCommand, to, err)
	}

	narrowing := recordNarrowingsOf(parsed)
	selected := selectVersions(records, narrowing, parsed.history, loaded.Definitions)
	kept, left := cutIdentities(selected, parsed.limit)

	// The versions this answer renders, flattened once and read three ways:
	// the Journal is asked about their Runs, the branch about their
	// suppressed fields, and the narration counts them. One flattening
	// rather than three is what keeps `suppressed` pairing back to the rows
	// by position (recordRows).
	versions := versionsOf(kept)

	// The Journal, for the one column the record cannot answer either: what
	// kind of Run wrote a version. §7 writes `dry_run` on the entry and on
	// nothing else, so this is the join a caller would otherwise make by
	// hand with `hyper show`, made inside the one call — and it is the
	// narrow door, one run.json per entry asked about and no Step file,
	// because that is the whole of what this column reads (store.Rehearsals,
	// ADR-0114).
	//
	// It stands after the cut so that it is asked only about the Runs this
	// answer names: a listing of one Record opens one entry, where a walk of
	// the whole Journal would make a `--limit 1` cost what a year of Runs
	// costs.
	rehearsals, err := held.Rehearsals(runsOf(versions))
	if err != nil {
		return reportReadStoreFault(recordsCommand, to, err)
	}

	rows, err := recordRows(held, kept, versions, rehearsals)
	if err != nil {
		return reportReadStoreFault(recordsCommand, to, err)
	}

	// The two renderings are one list of rows written twice (ADR-0026), and
	// the cut is applied before either of them: the table and the --json
	// stream state the same facts because they are built from one row set,
	// cut in one place.
	page := func(w io.Writer, rows []render.Row) error { return recordsPage(w, rows, narrowing.any()) }
	if code := writeAnswer(recordsCommand, to, rows, recordsTerminal(left), page); code != 0 {
		return code
	}

	// The human counterparts of the marker, all three on the narration
	// whichever form the answer took, because they are narration rather than
	// an answer (§9, destination.go). A truncated result must never look
	// complete, and a page that simply stopped where it was allowed to would
	// be one.
	if left.identitiesDropped > 0 {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", recordsCommand, truncationLine("Records", left.identities, len(selected), parsed, recordsNarrowing))
	}
	if left.versionsDropped > 0 {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", recordsCommand, cappedSeriesLine(left))
	}
	// A Definition file that did not load is in no namespace, so an Asset it
	// owns would read as Orphaned on a fault in the file that owns it. The
	// count is stated rather than the column being silently wrong, which is
	// the reading `review`'s AUTHORITY table already takes over the same
	// absence (§8, ADR-0069).
	if notLoaded := definitionsNotLoaded(loaded); notLoaded > 0 {
		fmt.Fprintf(to.narrate(), "hyper %s: %s; ORPHANED is read against the Definitions that did\n", recordsCommand, definitionsNotLoadedLine(notLoaded))
	}
	// A version whose Run the Journal holds no entry for has no marker to
	// carry, and the count is stated rather than the blank being read as
	// *this was not a rehearsal* — which is the one reading §7 says a reader
	// of this marker may never take. It is the same reading the line above
	// takes over the same shape of absence (§7, ADR-0114).
	if unattributed := unattributedVersions(versions, rehearsals); unattributed > 0 {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", recordsCommand, unattributedVersionsLine(unattributed))
	}

	return ExitClean
}

// recordsNarrowing is what a truncated listing says to do next: the three
// parameters that narrow the identity axis, which is the axis this command
// orders on and therefore the one a cap cuts.
//
// It is the command's own parameters rather than the renderer's, because the
// parameters that narrow an axis differ by which command was called —
// `--between` is `changes`'s and nobody else's — and naming a flag the caller's
// command does not take would point the remedy at an argument they would go
// looking for in their own command line. It is the parameters and not the
// sentence for runsNarrowing's reason: §9 gives this marker two surfaces, and
// the hint is the one member of it each spells for itself (render.Narrowing,
// issue #199).
//
// `--since` is not among them, and its absence is the rule rather than an
// oversight: it narrows the versions **inside** a series, and what a cap cut
// here is whole Records (§9, ADR-0065).
var recordsNarrowing = render.Narrowing{
	{Flag: "--target", Argument: "target"},
	{Flag: "--definition", Argument: "definition"},
	{Flag: "--name", Argument: "name"},
}

// recordsTerminal is the row every stream here ends with: the marker where
// something cut the answer, and the bare `false` only where nothing did.
//
// **Two things can cut this answer and the marker names one axis**, so it names
// the one to narrow first. §9 gives the record two axes and gives this command
// a parameter set for each: `--target`, `--definition` and `--name` narrow
// **identity**, which is what the limit cuts, and `--since` narrows **time**,
// which is what the cap on versions per series cuts — §9's own reason for
// giving `records` a `--since` at all.
//
// Where the limit dropped whole Records the axis is `identity`, that being the
// coarser cut and the one whose remedy comes first; a caller who narrows it and
// calls again reads the time cut on the next answer, where it is the only one
// left. Where the limit dropped nothing and the cap did, the axis is `time` and
// the counts are versions.
//
// The identity counts are **identities in both modes**, which is what makes
// them one number a reader can subtract: under `--history` a series is many
// rows and the limit still counts identities, so counting rows would report a
// cut of three Records as a cut of sixty (§12, ADR-0065).
func recordsTerminal(left cut) render.Row {
	switch {
	case left.identitiesDropped > 0:
		return render.NewTruncatedResultRow(render.TruncationMarker{
			Axis:     render.AxisIdentity,
			Returned: left.identities,
			Dropped:  left.identitiesDropped,
			Narrows:  recordsNarrowing,
		})
	case left.versionsDropped > 0:
		return render.NewTruncatedResultRow(render.TruncationMarker{
			Axis:     render.AxisTime,
			Returned: left.versions,
			Dropped:  left.versionsDropped,
			Narrows:  versionsNarrowing,
		})
	}
	return render.NewResultRow(false)
}

// versionsNarrowing is what a series the cap cut says to do next: `--since`,
// the one parameter that narrows the time axis a series is ordered on.
//
// It is the whole remedy and no larger cap stands beside it. §9 gives this
// command a `--since` **so that the axis a cap can cut has a parameter that
// narrows it**, and a window small enough to fit under the cap comes back
// whole — where widening the cap would be a bigger answer, which is not a thing
// this surface offers on either axis (§9, ADR-0065).
var versionsNarrowing = render.Narrowing{{Flag: "--since", Argument: "since"}}

// recordRow is `records`'s row, and its members are §9's own, in §9's order:
// {"type":"record","key":{…},"ordinal":…,"run_id":…,"step":…,
// "record_kind":…,"tombstoned":…,"orphaned":…,"secret_fields":[…],
// "provenance":{…}}. §9 writes that shape out once and milestone 11's MCP tool
// reuses this contract rather than minting a second one, so the declaration
// order here is the wire's and not a preference.
//
// **`key` is a nested object rather than three flat members**, which is §9's
// own shape and is the identity travelling as the one fact it is: a Record is
// identified by its Target, its Definition and a name together, and three
// siblings of `ordinal` would be three arguments that happen to be adjacent
// (§2).
//
// **`run_id` and `step` are the version's identity**, and this is the one row
// in the tool that carries both: two Steps of one Run writing one identity
// write two paths, so the Run alone would not name a version (§12).
//
// **`dry_run` says what kind of Run that was, and it is written always** — the
// bare `false` included, which is §7's one exception to the absence rule
// carried onto the surface that renders the Run. §6 records a rehearsal's reads
// like any other Run's, so a version this row names can be the whole of what a
// reader came for and still have been written by a Run §7 tells every consumer
// of Journal evidence to filter out; a reader who carries that rule here
// without the marker discards exactly the versions holding the account (§7,
// ADR-0114, issue #234).
//
// It is read off the Journal entry of the Run that wrote the version, which is
// the one place the marker is written — the same join the caller would make by
// hand with `hyper show`, made inside the one call. It is **absent** where the
// branch holds no entry for that Run at all: a Run writes its entry at start
// and Compaction never removes one, so that is a Store missing evidence rather
// than a Run that was not a rehearsal, and the two may not spell the same
// (store.Rehearsals). The narration counts those rows, exactly as it counts the
// Definitions `orphaned` could not be read against.
//
// **`tombstoned` and `orphaned` are the Record's state, and both are the
// series' rather than the version's.** §9 gives this row *whether its **head**
// is a Tombstone*, and that is one grain with `orphaned`'s *on every row that
// carries it*: the two together say whether the thing the Record names is still
// there, which is the question a surface for finding a version is asked
// alongside *where is it*. Every other member of this row — the ordinal, the
// Run, the Step, the suppressed fields, the Provenance — is the version's, and
// the split is exactly the one §9 draws.
//
// Under `--history` that means both markers repeat down a series, which is what
// `orphaned` is stated to do and what `tombstoned` reading the head buys: the
// column means one thing on every row it appears on, where a version-grained
// marker would mean *this Record is destroyed* on the first row of a series and
// *this particular version was a destruction* on the rest.
//
// They follow the ordinary absence rule, which is how the Store writes its own
// `tombstone` — a marker is written where it is true and absent where it is
// not, and a `false` there would state a fact against a Record that has nothing
// to say (§7).
//
// **`secret_fields` is derived on read.** §7's decoder does not read back which
// fields were suppressed; the marker is a constant standing in the position the
// value would occupy, so the fields whose decoded value is that constant are
// exactly the fields that were suppressed (store.SuppressedFields). It is
// absent where nothing was suppressed, on the same absence rule.
//
// **`provenance` is the whole of it**, the Run-wide half and the Step's half
// under one key, which is what a Record version carries and what no Journal
// file does (§7, ADR-0043). It is written always: a version file states which
// code performed the Run that wrote it, and a Record with no `provenance` would
// be a version this reader could not have read.
//
// **`run_id` goes out whole here and abbreviated on the page**, like every
// other id on a table read down a column (ADR-0047).
type recordRow struct {
	Type         string          `json:"type"`
	Key          recordKey       `json:"key"`
	Ordinal      int             `json:"ordinal"`
	RunID        string          `json:"run_id"`
	Step         int             `json:"step"`
	DryRun       *bool           `json:"dry_run,omitempty"`
	RecordKind   string          `json:"record_kind"`
	Tombstoned   bool            `json:"tombstoned,omitempty"`
	Orphaned     bool            `json:"orphaned,omitempty"`
	SecretFields []string        `json:"secret_fields,omitempty"`
	Provenance   provenanceBlock `json:"provenance"`
}

// recordKey is a Record's identity as the row carries it: the three columns the
// ordering reads left to right, in that order (§2, ADR-0044).
type recordKey struct {
	Target     string `json:"target"`
	Definition string `json:"definition"`
	Name       string `json:"name"`
}

// Cells is the row's line on the page, in recordsColumns' order.
//
// Three of them render differently here and nowhere else: the Run id is
// abbreviated, the markers are the word `yes` under columns named for them
// rather than the booleans the wire carries, and Provenance renders its
// `hyper_version` alone. All of them are the page's reading of facts the row
// holds once.
//
// **`REHEARSAL` is the word where there is something to say and nothing where
// there is not**, which is the page half of §7's exception and the reading
// `show`'s own header already takes over the same marker: the wire carries it
// always because a reader that takes its absence for `false` cannot recover,
// and a column carrying `no` down every row of an ordinary listing says
// nothing a reader scans for. A blank therefore covers both an ordinary Run
// and a Run the branch holds no entry for, and the second is what the
// narration counts (entryValues, show.go).
//
// **Provenance is seven values and this is a table read down a column**, so
// what stands here is the one member a reader scans a listing for — which
// `hyper` wrote the version — and the wire carries the block whole beside it.
// Provenance **drift** is a change rather than a version, and §8 gives it a
// table of its own on the surface whose job is changes; the whole of it for one
// version is `hyper show`'s, on the surface whose job is one entry read back.
func (r recordRow) Cells() []string {
	return []string{
		r.Key.Target,
		r.Key.Definition,
		r.Key.Name,
		strconv.Itoa(r.Ordinal),
		abbreviatedRun(r.RunID),
		strconv.Itoa(r.Step),
		yesCell(r.DryRun != nil && *r.DryRun),
		r.RecordKind,
		yesCell(r.Tombstoned),
		yesCell(r.Orphaned),
		namesText(r.SecretFields),
		r.Provenance.HyperVersion,
	}
}

// recordsColumns is the page's header: the row's own members in the row's own
// order, the identity's three columns spelled as §8's Comparison spells them.
var recordsColumns = []string{"TARGET", "DEFINITION", "RECORD", "ORDINAL", "RUN", "STEP", "REHEARSAL", "KIND", "TOMBSTONE", "ORPHANED", "SECRETS", "HYPER"}

// selection is one Record's contribution to the answer: which of its versions
// this call renders, in the order it renders them, and the two facts that
// belong to the series rather than to any version of it.
//
// It is a value rather than a flat list of versions because the limit counts
// **identities**: a series comes back whole or does not come back, and a cut
// applied to a flat list could not honour that. What the type holds is exactly
// what the cut ranges over.
type selection struct {
	// versions are this Record's versions as this call renders them: the
	// Head alone, or the whole series newest-first under `--history`.
	versions []store.Version
	// found is how many versions the window admitted before the cap cut
	// them, so `found - len(versions)` is what the cap left out. It is
	// counted rather than inferred because a caller reading twenty versions
	// cannot tell a series of twenty from a series of two hundred, and a
	// truncated result must never look complete (§9).
	found int
	// tombstoned and orphaned are the Record's state rather than any
	// version's, and they sit here for that reason: both are read off the
	// Head, both go on **every** row the series contributes, and no version
	// carries either as a fact about itself.
	//
	// tombstoned is *the Head is a Tombstone*, which is what §9 gives this
	// row. orphaned is an Asset whose Definition no longer exists, still
	// standing — reported for as long as it stands rather than once, at the
	// moment it was orphaned, otherwise a forgotten resource becomes
	// invisible by way of a tidy-up commit (§7, ADR-0012).
	tombstoned bool
	orphaned   bool
}

// selectVersions is the answer's shape before it is cut: one selection per
// Record the narrowing kept, in the identity order the Store listed them.
//
// **Without `--history` it is the Head alone**, which is what `records` returns
// unless the boolean is given — an explicit boolean and never a mode some other
// parameter turns on (ADR-0013).
//
// **Under `--history` it is §7's Head ordering read backwards**: `written_at`
// descending, ties broken by the file name descending. The reversal is whole,
// both keys inverting together, and it is literally a reversal of the slice the
// Head was derived from — so the ordering that decides which version is the
// Head and the ordering this surface renders cannot drift apart, which is what
// makes the first row of each series exactly the row `records` returns without
// `--history` at all.
//
// A Record whose versions the window left empty contributes no selection, so it
// is not a Record the limit counted and not a blank line on the page.
func selectVersions(records []store.Series, narrowing recordNarrowings, history bool, definitions artefact.DefinitionIndex) []selection {
	selected := make([]selection, 0, len(records))
	for _, series := range records {
		if !narrowing.keeps(series.Identity) {
			continue
		}
		head, standing := series.Head()
		if !standing {
			// A series the branch listed has versions by
			// construction; this is the empty listing no reader
			// answers, kept as a skip rather than a claim.
			continue
		}

		chosen := selection{
			versions:   []store.Version{head},
			found:      1,
			tombstoned: head.Tombstone,
			orphaned:   orphaned(head, definitions),
		}
		if history {
			versions := slices.Clone(series.Versions)
			slices.Reverse(versions)
			versions = narrowing.window(versions)
			if len(versions) == 0 {
				continue
			}
			chosen.found = len(versions)
			if len(versions) > versionsPerSeries {
				versions = versions[:versionsPerSeries]
			}
			chosen.versions = versions
		}
		selected = append(selected, chosen)
	}
	return selected
}

// orphaned answers whether a Record is an Orphaned Asset: an Asset the Store
// holds, still standing, whose Definition no longer exists (§7, ADR-0012).
//
// All three clauses earn their place. **Asset**, because an Observation is a
// reading rather than something `hyper` is accountable for having made, and a
// Definition deleted out from under one leaves no resource anywhere.
// **Standing**, because a Tombstoned Asset was destroyed and there is nothing
// left to be unreachable — the marker reports a live resource nothing `hyper`
// can now do reaches, and there is no adoption path (§7). **Definition
// missing**, which is the whole of what orphans one: Expansion needs a
// Definition (§5), so its absence is what puts the Asset out of reach.
//
// It takes the Head rather than the series, all three clauses being facts the
// Head carries: its identity names the Definition, its `record_type` is `asset`
// wherever `hyper`'s effect reached the thing — a Tombstone's included — and
// its marker is what says the Asset no longer stands.
func orphaned(head store.Version, definitions artefact.DefinitionIndex) bool {
	if head.RecordType != store.RecordAsset || head.Tombstone {
		return false
	}
	_, exists := definitions[head.Identity.Definition]
	return !exists
}

// cut is what this answer left out, on each of the two axes something can cut
// it on: the Records the limit dropped, and the versions the cap dropped from
// the series that survived.
//
// The two are counted together because one terminal row carries them, and a
// consumer reading `truncated: false` off a stream that dropped either is
// reading a truncated result that looks complete — the one thing §9 says this
// surface may never produce.
type cut struct {
	// identities is how many Records came back and how many the limit
	// dropped, on the identity axis this command orders on.
	identities, identitiesDropped int
	// versions is how many versions came back and how many the cap dropped
	// from the series that did, on the time axis a series is ordered on.
	versions, versionsDropped int
}

// cutIdentities keeps the first N Records of the identity ordering and answers
// what both axes left out.
//
// It cuts **identities and never rows**, which is §9's rule under `--history`
// and the reason the cut is applied here rather than to the flat row list every
// other listing cuts: a series cut partway through is a partial history wearing
// a complete one's shape, which is the one thing a truncated result may never
// look like. Without `--history` a Record is one row and this is the ordinary
// cut, so there is one rule rather than two that agree in the common case.
//
// The version counts are taken over the Records that **survived** the limit,
// because those are the series this answer carries: versions dropped along with
// a Record the limit dropped were never on this page to be missing from it.
func cutIdentities(selected []selection, limit int) (kept []selection, left cut) {
	kept = selected
	if limit > 0 && len(selected) > limit {
		kept = selected[:limit]
	}
	left = cut{identities: len(kept), identitiesDropped: len(selected) - len(kept)}
	for _, chosen := range kept {
		left.versions += len(chosen.versions)
		left.versionsDropped += chosen.found - len(chosen.versions)
	}
	return kept, left
}

// cappedSeriesLine is what a cap that cut a series says to a human: what came
// back, out of what, and the parameter that narrows the axis it cut.
//
// The remedy is `--since` and not a larger cap, because there is no larger cap
// to name. §9 gives `records` a `--since` **so that the axis a cap can cut has
// a parameter that narrows it**: a caller who has already named one Record has
// no identity narrowing left to do, and a window small enough to fit under the
// cap comes back whole. What it does not offer is versionsPerSeries as a flag —
// widening a cap is a bigger answer where every remedy this surface has is a
// narrower question (§9, ADR-0065).
//
// It reports the counts rather than the constant, so what a reader takes away
// is that this series was cut and by how much, and not a number to depend on:
// the cap is a constant an implementation picks and nothing reads it back.
func cappedSeriesLine(left cut) string {
	return fmt.Sprintf("returned %d of %d versions across %d series; %d dropped by the cap on versions per series — %s",
		left.versions, left.versions+left.versionsDropped, left.identities, left.versionsDropped, versionsNarrowing.Flags())
}

// recordRows is the answer: one row per version of each Record that came back,
// in the order the selections hold them.
//
// The suppressed fields of every row are read in **one** batch, off the flat
// list of versions the selections hold, and paired back by position. That is
// the whole reason this is one function rather than a row builder called per
// version: a read per row is a git subprocess per row, and the cut has already
// bounded how many there are (store.SuppressedFields).
//
// The flat list is handed in rather than rebuilt here, and that is the pairing
// rather than a saved loop: the caller read the Journal against this order and
// the batch below is paired back by position, so a second flattening would be a
// second order for one row set to be built from.
func recordRows(held *store.Store, kept []selection, versions []store.Version, rehearsals map[store.RunID]bool) ([]render.Row, error) {
	suppressed, err := held.SuppressedFields(versions)
	if err != nil {
		return nil, err
	}

	rows := make([]render.Row, 0, len(versions))
	at := 0
	for _, chosen := range kept {
		for _, version := range chosen.versions {
			rows = append(rows, recordRowOf(version, chosen, suppressed[at], rehearsals))
			at++
		}
	}
	return rows, nil
}

// recordRowOf is one version as the row carries it: the version's own facts,
// the Record's two state markers, and the fields the secret marker stood in
// for.
func recordRowOf(version store.Version, of selection, secret []string, rehearsals map[store.RunID]bool) recordRow {
	return recordRow{
		Type: "record",
		Key: recordKey{
			Target:     version.Identity.Target,
			Definition: version.Identity.Definition,
			Name:       version.Identity.Name,
		},
		Ordinal:      version.Ordinal,
		RunID:        version.Run.String(),
		Step:         version.Step,
		DryRun:       rehearsalOf(version.Run, rehearsals),
		RecordKind:   string(version.RecordType),
		Tombstoned:   of.tombstoned,
		Orphaned:     of.orphaned,
		SecretFields: secret,
		Provenance: provenanceBlock{
			HyperVersion:       version.Provenance.Run.HyperVersion,
			ProcedureRevision:  version.Provenance.Run.ProcedureRevision,
			RepoRevision:       version.Provenance.Run.RepoRevision,
			RepoDirty:          version.Provenance.Run.RepoDirty,
			DefinitionRevision: version.Provenance.Step.DefinitionRevision,
			ManifestDigest:     version.Provenance.Step.ManifestDigest,
			OriginDigest:       version.Provenance.Step.OriginDigest,
		},
	}
}

// rehearsalOf is the marker one version carries: what the Journal entry of the
// Run that wrote it says, and nothing at all where the branch holds no entry
// for that Run.
//
// The pointer is the three states kept apart. `true` and `false` are both
// written on the wire — §7's exception, because a reader that takes absence for
// `false` gets a permanent wrong answer — and the third state is the branch
// having nothing to say, which is a Store missing evidence rather than a Run of
// either kind. Collapsing it to `false` would be exactly the reading the
// exception exists to prevent, one layer further out.
func rehearsalOf(run store.RunID, rehearsals map[store.RunID]bool) *bool {
	marker, held := rehearsals[run]
	if !held {
		return nil
	}
	return &marker
}

// versionsOf is every version this answer renders, flattened in the order the
// selections hold them — which is the order the rows go out in, and therefore
// the order anything read in a batch against it is paired back by.
func versionsOf(kept []selection) []store.Version {
	var versions []store.Version
	for _, chosen := range kept {
		versions = append(versions, chosen.versions...)
	}
	return versions
}

// runsOf is every Run that wrote one of them, each once. It is what the Journal
// is asked about, and it is a set because one Run writing forty versions is one
// entry to open.
func runsOf(versions []store.Version) []store.RunID {
	named := map[store.RunID]bool{}
	runs := make([]store.RunID, 0, len(versions))
	for _, version := range versions {
		if !named[version.Run] {
			named[version.Run] = true
			runs = append(runs, version.Run)
		}
	}
	return runs
}

// unattributedVersions is how many rows of this answer name a Run the Journal
// holds no entry for.
//
// It counts versions and not Records, because the marker is the version's: a
// history whose Runs are half in the Journal and half not has one row of each,
// and a count of Records could not say how many rows the reader must go and
// find another way.
//
// It reads the absence through rehearsalOf rather than through the map, so the
// count and the member it is a count of cannot come to disagree about what
// *the branch has nothing to say* is.
func unattributedVersions(versions []store.Version, rehearsals map[store.RunID]bool) int {
	unattributed := 0
	for _, version := range versions {
		if rehearsalOf(version.Run, rehearsals) == nil {
			unattributed++
		}
	}
	return unattributed
}

// unattributedVersionsLine is what those rows say to a human: how many, and
// that the column is read off the entries the Journal does hold.
//
// It names `hyper runs` as the act that shows what the Journal holds, for the
// reason the Definition line names `hyper check`: a count with nothing to do
// about it is a count a reader carries away and cannot act on.
func unattributedVersionsLine(unattributed int) string {
	versions, name := "versions", "name"
	if unattributed == 1 {
		versions, name = "version", "names"
	}
	return fmt.Sprintf("%d %s %s a Run the Journal holds no entry for; REHEARSAL is read off the entries it does · hyper runs", unattributed, versions, name)
}

// recordNarrowings is what the caller narrowed the listing by: §9's typed,
// closed parameters, read off the arguments once.
//
// They are one value rather than four reads of `commandArgs` for the reason
// `runs`'s are: two questions are asked of them — which Records survive, and
// whether anything was narrowed at all — and a parameter answered by one and
// not the other would be a page saying *no Record matched* about a Store
// holding plenty.
//
// `--history` is not among them. It widens what comes back rather than
// narrowing it, so a caller who typed it alone and got nothing has an empty
// Store rather than a question that matched none.
type recordNarrowings struct {
	target     string
	definition string
	name       string
	since      time.Time
	sinceNamed bool
}

// recordNarrowingsOf reads them off the arguments.
func recordNarrowingsOf(parsed commandArgs) recordNarrowings {
	return recordNarrowings{
		target:     parsed.target,
		definition: parsed.definition,
		name:       parsed.name,
		since:      parsed.since,
		sinceNamed: parsed.sinceNamed,
	}
}

// any answers whether the caller narrowed the listing at all, which is the
// whole difference between the two sentences an empty page writes.
//
// It is a comparison against the zero value rather than a test per member, so
// it enumerates nothing: a parameter added to this type is a parameter this
// answers, and the two questions asked of these cannot come apart by somebody
// editing one of them.
func (n recordNarrowings) any() bool { return n != recordNarrowings{} }

// keeps answers whether one Record survives the identity narrowing.
//
// The three are conjunctive and each is matched **byte-exact over UTF-8**,
// which is the comparison §9 fixes for matching a name everywhere. They are the
// identity's own three columns, so naming all three is naming one Record and
// naming none is the whole branch.
func (n recordNarrowings) keeps(id store.Identity) bool {
	return (n.target == "" || id.Target == n.target) &&
		(n.definition == "" || id.Definition == n.definition) &&
		(n.name == "" || id.Name == n.name)
}

// window is the versions of one series that fall inside the window `--since`
// named, keeping the order they arrived in.
//
// The bound is withinWindow's, one package file over, so `records` and `runs`
// cannot come to disagree about which side of `--since` an instant equal to it
// falls on. A window that admits nothing removes the Record from the answer
// rather than leaving it there with no versions: a Record is in this answer
// because it has something to say in the window asked about.
func (n recordNarrowings) window(versions []store.Version) []store.Version {
	if !n.sinceNamed {
		return versions
	}
	kept := make([]store.Version, 0, len(versions))
	for _, version := range versions {
		if withinWindow(version.WrittenAt, n.since, n.sinceNamed) {
			kept = append(kept, version)
		}
	}
	return kept
}

// recordsPage is `records`'s page: where the record is, and under it the table
// or the sentence that stands where it has no rows.
//
// An empty table is written as nothing at all by the renderer, header included
// — what stands in its place is the command's own (§8) — and here it is a
// sentence naming what was looked for. A header over no rows would read as a
// listing that found some.
//
// The location line is `runs`'s own and stands here for the same reason: this
// is the other command whose job is finding something in the Store, and a
// listing of Record versions says what the account holds and never where it is
// held (writeRecordLocation, ADR-0113, issue #233).
func recordsPage(w io.Writer, rows []render.Row, narrowed bool) error {
	if err := writeRecordLocation(w); err != nil {
		return err
	}
	if len(rows) == 0 {
		if narrowed {
			_, err := fmt.Fprintln(w, "no Record matched")
			return err
		}
		_, err := fmt.Fprintln(w, "no Record in this repository's Store")
		return err
	}
	return render.WriteTable(w, recordsColumns, rows)
}
