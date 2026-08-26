package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// `records`'s own parameters, its two orderings and the two cuts that bound it
// (§9, ADR-0013, ADR-0049, issue #166). What the corpus drives end to end this
// states at the grain each rule is decided at.

// theAsset and theObservation are the two identities these cases range over.
var (
	theAsset       = store.Identity{Target: "cloudflare-prod", Definition: "legacy-dns", Name: "preview-9.example.com"}
	theObservation = store.Identity{Target: "local", Definition: "uptime", Name: "status.hyper.dev"}
)

// aRecordVersion is one version of a series as a listing answers it: where it
// sits in the ordering, and what it says about itself.
func aRecordVersion(id store.Identity, kind store.RecordType, ordinal int, at time.Time) store.Version {
	return store.Version{
		Metadata: store.Metadata{
			Identity:   id,
			RecordType: kind,
			WrittenAt:  at,
		},
		Ordinal: ordinal,
	}
}

// aSeries is a Record's versions in the order the Head is derived from: oldest
// first, so the last of them is the Head (store.Series).
func aSeries(id store.Identity, kind store.RecordType, instants ...time.Time) store.Series {
	series := store.Series{Identity: id}
	for i, at := range instants {
		series.Versions = append(series.Versions, aRecordVersion(id, kind, i+1, at))
	}
	return series
}

// The instants the seeded series are written at, an hour apart so the ordering
// is legible in a failure message.
var (
	theFirstInstant  = time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	theSecondInstant = theFirstInstant.Add(time.Hour)
	theThirdInstant  = theFirstInstant.Add(2 * time.Hour)
)

// everyDefinition is a repository where every Definition a Record names still
// exists, which is the ordinary case and the one nothing is Orphaned in.
var everyDefinition = artefact.DefinitionIndex{"legacy-dns": {}, "uptime": {}}

// TestParseArgs_ReadsRecordsTypedParameters. Each valued parameter is read in
// both spellings, the spaced one and the `=`-joined one every other valued flag
// takes, and `--history` is the boolean beside them.
func TestParseArgs_ReadsRecordsTypedParameters(t *testing.T) {
	for name, args := range map[string][]string{
		"the spaced spelling": {
			"--target", "local",
			"--definition", "uptime",
			"--name", "status.hyper.dev",
			"--history",
		},
		"the equals spelling, which every other valued flag takes": {
			"--target=local",
			"--definition=uptime",
			"--name=status.hyper.dev",
			"--history",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			parsed, _, code := parseArgs(recordsCommand, args, recordsParameters, environment(nil), streams{stderr: &stderr})

			if code != 0 {
				t.Fatalf("parseArgs() = %d, want 0; stderr said %q", code, stderr.String())
			}
			if parsed.target != "local" || parsed.definition != "uptime" || parsed.name != "status.hyper.dev" {
				t.Errorf("the identity read back as (%q, %q, %q), want (local, uptime, status.hyper.dev)",
					parsed.target, parsed.definition, parsed.name)
			}
			if !parsed.history {
				t.Error("history = false, want true — --history is an explicit boolean (ADR-0013)")
			}
		})
	}
}

// TestParseArgs_LeavesRecordsParametersOffEveryOtherCommand. §9 gives
// `--definition`, `--name` and `--history` to `records` alone, and a command
// that does not take one answers the unknown flag it is rather than accepting a
// parameter it would then have nothing to do with.
func TestParseArgs_LeavesRecordsParametersOffEveryOtherCommand(t *testing.T) {
	for _, flag := range []string{"--definition=uptime", "--name=status.hyper.dev", "--history"} {
		t.Run(flag, func(t *testing.T) {
			var stderr bytes.Buffer
			_, _, code := parseArgs(runsCommand, []string{flag}, runsParameters, environment(nil), streams{stderr: &stderr})

			if code != ExitUsage {
				t.Fatalf("parseArgs() = %d, want %d — %s is `records`'s and nobody else's", code, ExitUsage, flag)
			}
			if said := stderr.String(); !strings.Contains(said, "unknown flag") {
				t.Errorf("stderr said %q, want the unknown flag named", said)
			}
		})
	}
}

// TestSelectVersions_ReturnsTheHeadOnlyWithoutHistory. `records` returns the
// Head unless `--history` is given — an explicit boolean rather than a mode
// that turns itself on when some other parameter is named (ADR-0013).
func TestSelectVersions_ReturnsTheHeadOnlyWithoutHistory(t *testing.T) {
	records := []store.Series{aSeries(theObservation, store.RecordObservation, theFirstInstant, theSecondInstant, theThirdInstant)}

	selected := selectVersions(records, recordNarrowings{}, false, everyDefinition)

	if len(selected) != 1 || len(selected[0].versions) != 1 {
		t.Fatalf("selectVersions answered %d Records and %d versions of the first, want 1 and 1", len(selected), len(selected[0].versions))
	}
	if head := selected[0].versions[0]; head.Ordinal != 3 || !head.WrittenAt.Equal(theThirdInstant) {
		t.Errorf("the version is ordinal %d at %s, want the Head — ordinal 3 at %s", head.Ordinal, head.WrittenAt, theThirdInstant)
	}
}

// TestSelectVersions_ReadsTheHeadOrderingBackwardsUnderHistory. Under
// `--history` a series is §7's Head ordering read backwards — `written_at`
// descending, ties broken by the file name descending — and the reversal is
// **whole**, both keys inverting together, so the ordering that decides which
// version is the Head and the ordering this surface renders can never drift
// apart.
//
// That is what makes the first row of each series exactly the row `records`
// returns without `--history` at all, which the corpus holds byte for byte
// below.
func TestSelectVersions_ReadsTheHeadOrderingBackwardsUnderHistory(t *testing.T) {
	records := []store.Series{aSeries(theObservation, store.RecordObservation, theFirstInstant, theSecondInstant, theThirdInstant)}

	selected := selectVersions(records, recordNarrowings{}, true, everyDefinition)

	if len(selected) != 1 {
		t.Fatalf("selectVersions answered %d Records, want 1", len(selected))
	}
	want := []int{3, 2, 1}
	got := make([]int, len(selected[0].versions))
	for i, version := range selected[0].versions {
		got[i] = version.Ordinal
	}
	if !slices.Equal(got, want) {
		t.Errorf("the series came back in ordinal order %v, want %v — newest first", got, want)
	}
}

// TestSelectVersions_CutsASeriesAtTheVersionCapAndSaysSo. What bounds a single
// series is a cap on versions per series, and what a cut series carries is the
// count it left out: a caller reading twenty versions cannot tell a series of
// twenty from a series of two hundred, and a truncated result must never look
// complete (§9).
func TestSelectVersions_CutsASeriesAtTheVersionCapAndSaysSo(t *testing.T) {
	instants := make([]time.Time, versionsPerSeries+3)
	for i := range instants {
		instants[i] = theFirstInstant.Add(time.Duration(i) * time.Hour)
	}
	records := []store.Series{aSeries(theObservation, store.RecordObservation, instants...)}

	selected := selectVersions(records, recordNarrowings{}, true, everyDefinition)

	if len(selected) != 1 {
		t.Fatalf("selectVersions answered %d Records, want 1", len(selected))
	}
	if got := len(selected[0].versions); got != versionsPerSeries {
		t.Errorf("the series came back with %d versions, want the cap of %d", got, versionsPerSeries)
	}
	if got := selected[0].found - len(selected[0].versions); got != 3 {
		t.Errorf("the cap left out %d versions, want 3", got)
	}
	if head := selected[0].versions[0]; head.Ordinal != len(instants) {
		t.Errorf("the first version is ordinal %d, want %d — a cap cuts the oldest and never the Head", head.Ordinal, len(instants))
	}
}

// TestOrphaned is the marker's three clauses, each on its own. An Asset whose
// Definition no longer exists is Orphaned **for as long as it stands**: nothing
// hyper can now do reaches it, and there is no adoption path (§7, ADR-0012).
func TestOrphaned(t *testing.T) {
	tombstoned := aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant)
	tombstoned.Versions[1].Tombstone = true

	for name, held := range map[string]struct {
		series      store.Series
		definitions artefact.DefinitionIndex
		want        bool
	}{
		"an Asset whose Definition is gone, still standing": {
			series:      aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant),
			definitions: artefact.DefinitionIndex{"uptime": {}},
			want:        true,
		},
		"an Asset whose Definition still exists": {
			series:      aSeries(theAsset, store.RecordAsset, theFirstInstant),
			definitions: everyDefinition,
			want:        false,
		},
		"an Asset the Store holds destroyed, whose Definition is gone": {
			series:      tombstoned,
			definitions: artefact.DefinitionIndex{"uptime": {}},
			want:        false,
		},
		"an Observation whose Definition is gone": {
			series:      aSeries(theObservation, store.RecordObservation, theFirstInstant),
			definitions: artefact.DefinitionIndex{},
			want:        false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			head, standing := held.series.Head()
			if !standing {
				t.Fatal("the seeded series answers no Head")
			}
			if got := orphaned(head, held.definitions); got != held.want {
				t.Errorf("orphaned = %v, want %v", got, held.want)
			}
		})
	}
}

// TestSelectVersions_MarksAnOrphanedAssetOnEveryRowItCarries, and not once at
// the moment it was orphaned: otherwise a forgotten resource becomes invisible
// by way of a tidy-up commit (§7).
func TestSelectVersions_MarksAnOrphanedAssetOnEveryRowItCarries(t *testing.T) {
	records := []store.Series{aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant, theThirdInstant)}

	selected := selectVersions(records, recordNarrowings{}, true, artefact.DefinitionIndex{"uptime": {}})

	if len(selected) != 1 {
		t.Fatalf("selectVersions answered %d Records, want 1", len(selected))
	}
	if !selected[0].orphaned {
		t.Fatal("the Record is not Orphaned; its Definition is not in the namespace and its head stands")
	}
	for i, version := range selected[0].versions {
		if row := recordRowOf(version, selected[0], nil); !row.Orphaned {
			t.Errorf("the row at ordinal %d carries no orphaned marker; every row that carries the Asset carries it (row %d)", version.Ordinal, i)
		}
	}
}

// TestSelectVersions_ReadsTombstonedOffTheHeadAndCarriesItDownTheSeries. §9
// gives this row *whether its **head** is a Tombstone*, which is the Record's
// state and not the version's — so it repeats down a series exactly as
// `orphaned` does, and the column means one thing on every row it appears on.
//
// The series here is destroyed at its Head and ordinary beneath it, which is
// the shape a version-grained marker would have marked one row of.
func TestSelectVersions_ReadsTombstonedOffTheHeadAndCarriesItDownTheSeries(t *testing.T) {
	destroyed := aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant, theThirdInstant)
	destroyed.Versions[2].Tombstone = true

	selected := selectVersions([]store.Series{destroyed}, recordNarrowings{}, true, everyDefinition)

	if len(selected) != 1 || !selected[0].tombstoned {
		t.Fatalf("the Record reads tombstoned=%v, want true — its Head is a Tombstone", selected[0].tombstoned)
	}
	for _, version := range selected[0].versions {
		if row := recordRowOf(version, selected[0], nil); !row.Tombstoned {
			t.Errorf("the row at ordinal %d carries no tombstoned marker; the marker is the Head's and stands on every row", version.Ordinal)
		}
	}
}

// TestSelectVersions_ReadsTombstonedOffTheHeadAndNotOffAnInteriorVersion, which
// is destroy-then-recreate: a Tombstone with a further version above it makes
// the series read alive again (store.Series.Head), and the Record's state is
// that it stands.
func TestSelectVersions_ReadsTombstonedOffTheHeadAndNotOffAnInteriorVersion(t *testing.T) {
	recreated := aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant, theThirdInstant)
	recreated.Versions[1].Tombstone = true

	selected := selectVersions([]store.Series{recreated}, recordNarrowings{}, true, everyDefinition)

	if len(selected) != 1 || selected[0].tombstoned {
		t.Fatalf("the Record reads tombstoned=%v, want false — a version above the Tombstone makes it stand again", selected[0].tombstoned)
	}
}

// TestCutIdentities_ReportsWhatTheVersionCapLeftOut. A cap that cut a series is
// a truncated result, and a terminal row saying `false` over one is a truncated
// result that looks complete — the one thing §9 says this surface may never
// produce. The counts are versions and the axis is `time`, which is the axis a
// series is ordered on and the one `--since` narrows.
func TestCutIdentities_ReportsWhatTheVersionCapLeftOut(t *testing.T) {
	instants := make([]time.Time, versionsPerSeries+3)
	for i := range instants {
		instants[i] = theFirstInstant.Add(time.Duration(i) * time.Hour)
	}
	selected := selectVersions([]store.Series{aSeries(theObservation, store.RecordObservation, instants...)},
		recordNarrowings{}, true, everyDefinition)

	_, left := cutIdentities(selected, defaultListLimit)

	if left.identitiesDropped != 0 {
		t.Fatalf("the limit dropped %d identities, want 0 — this case is about the cap alone", left.identitiesDropped)
	}
	if left.versions != versionsPerSeries || left.versionsDropped != 3 {
		t.Errorf("the cut returned %d versions and dropped %d, want %d and 3", left.versions, left.versionsDropped, versionsPerSeries)
	}

	terminal, err := json.Marshal(recordsTerminal(left))
	if err != nil {
		t.Fatalf("marshalling the terminal row: %v", err)
	}
	if got := string(terminal); !strings.Contains(got, `"axis":"time"`) || !strings.Contains(got, `"dropped":3`) {
		t.Errorf("the terminal row is %s, want the time axis and the three versions it dropped", got)
	}
}

// TestRecordsTerminal_NamesTheCoarserCutWhereBothCut. The marker names one axis
// and two things can cut this answer, so it names the one to narrow first: a
// caller who narrows identity and calls again reads the time cut on the next
// answer, where it is the only one left.
func TestRecordsTerminal_NamesTheCoarserCutWhereBothCut(t *testing.T) {
	both := cut{identities: 2, identitiesDropped: 3, versions: 40, versionsDropped: 7}

	terminal, err := json.Marshal(recordsTerminal(both))
	if err != nil {
		t.Fatalf("marshalling the terminal row: %v", err)
	}
	if got := string(terminal); !strings.Contains(got, `"axis":"identity"`) || !strings.Contains(got, `"dropped":3`) {
		t.Errorf("the terminal row is %s, want the identity axis and the three Records it dropped", got)
	}
}

// TestCutIdentities_DropsWholeIdentitiesAndNeverCutsASeries. Under `--history`
// the limit counts identities and not rows: a series comes back whole or does
// not come back, a series cut partway through being a partial history wearing a
// complete one's shape (§9).
func TestCutIdentities_DropsWholeIdentitiesAndNeverCutsASeries(t *testing.T) {
	records := []store.Series{
		aSeries(theAsset, store.RecordAsset, theFirstInstant, theSecondInstant, theThirdInstant),
		aSeries(theObservation, store.RecordObservation, theFirstInstant, theSecondInstant),
	}
	selected := selectVersions(records, recordNarrowings{}, true, everyDefinition)

	kept, left := cutIdentities(selected, 1)

	if len(kept) != 1 || left.identitiesDropped != 1 {
		t.Fatalf("cutIdentities kept %d and dropped %d, want 1 and 1 — the counts are identities", len(kept), left.identitiesDropped)
	}
	if got := len(kept[0].versions); got != 3 {
		t.Errorf("the Record that survived came back with %d versions, want all 3 — a cap on identities never cuts a series", got)
	}
}

// TestRecordNarrowings_MatchByteExactOnEachColumnOfTheIdentity, which is the
// comparison §9 fixes for matching a name everywhere. The three are
// conjunctive, so naming all three names one Record.
func TestRecordNarrowings_MatchByteExactOnEachColumnOfTheIdentity(t *testing.T) {
	for name, held := range map[string]struct {
		narrowing recordNarrowings
		want      bool
	}{
		"nothing named at all":         {recordNarrowings{}, true},
		"the Target named":             {recordNarrowings{target: "local"}, true},
		"the Target in another case":   {recordNarrowings{target: "Local"}, false},
		"the Definition named":         {recordNarrowings{definition: "uptime"}, true},
		"another Definition":           {recordNarrowings{definition: "legacy-dns"}, false},
		"the whole identity named":     {recordNarrowings{target: "local", definition: "uptime", name: "status.hyper.dev"}, true},
		"two columns of three matched": {recordNarrowings{target: "local", definition: "uptime", name: "cert.hyper.dev"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := held.narrowing.keeps(theObservation); got != held.want {
				t.Errorf("keeps(%v) = %v, want %v", theObservation, got, held.want)
			}
		})
	}
}

// TestRecordNarrowings_SinceIsALowerBoundThatIncludesTheInstantItNames, so a
// timestamp copied off a page selects the version it was copied from — the same
// reading `runs`'s `--since` takes.
func TestRecordNarrowings_SinceIsALowerBoundThatIncludesTheInstantItNames(t *testing.T) {
	versions := []store.Version{
		aRecordVersion(theObservation, store.RecordObservation, 3, theThirdInstant),
		aRecordVersion(theObservation, store.RecordObservation, 2, theSecondInstant),
		aRecordVersion(theObservation, store.RecordObservation, 1, theFirstInstant),
	}

	narrowing := recordNarrowings{since: theSecondInstant, sinceNamed: true}
	kept := narrowing.window(versions)

	if len(kept) != 2 || kept[0].Ordinal != 3 || kept[1].Ordinal != 2 {
		t.Errorf("the window kept %d versions, want the two at or after %s", len(kept), theSecondInstant)
	}
}

// TestSelectVersions_DropsARecordTheWindowLeftEmpty. A Record is in this answer
// because it has something to say in the window asked about, so one the window
// admits nothing of is not a Record the limit counted and not a blank line on
// the page.
func TestSelectVersions_DropsARecordTheWindowLeftEmpty(t *testing.T) {
	records := []store.Series{
		aSeries(theAsset, store.RecordAsset, theFirstInstant),
		aSeries(theObservation, store.RecordObservation, theThirdInstant),
	}

	narrowing := recordNarrowings{since: theSecondInstant, sinceNamed: true}
	selected := selectVersions(records, narrowing, true, everyDefinition)

	if len(selected) != 1 {
		t.Fatalf("selectVersions answered %d Records, want 1 — the Record with nothing in the window is not one", len(selected))
	}
	if selected[0].versions[0].Identity != theObservation {
		t.Errorf("the Record that survived is %v, want %v", selected[0].versions[0].Identity, theObservation)
	}
}

// TestTheFirstRowOfASeriesIsTheRowRecordsReturnsWithoutHistory, held over the
// corpus's own checked-in bytes.
//
// It is the wire that carries the claim. A row is one object on the --json
// stream, and two renderings of one result are byte-identical and diffable
// (§9); the page's cells are the same facts laid out against a table whose
// column widths are a property of every row in it, so the row is where the
// identity is stated and where it is asserted.
//
// Reading the goldens rather than driving the command is deliberate: what the
// two cases assert against is what a reader of the corpus sees, and a
// derivation that agreed with itself while both goldens were wrong is the one
// failure this cannot have.
func TestTheFirstRowOfASeriesIsTheRowRecordsReturnsWithoutHistory(t *testing.T) {
	heads := recordStream(t, filepath.Join("testdata", "records", "the-heads-listed-json", "stdout.golden"))
	history := recordStream(t, filepath.Join("testdata", "records", "a-history-is-identity-major-json", "stdout.golden"))

	seen := map[recordKey]bool{}
	var firsts []string
	for _, line := range history {
		key := recordKeyOf(t, line)
		if seen[key] {
			continue
		}
		seen[key] = true
		firsts = append(firsts, line)
	}

	if !slices.Equal(firsts, heads) {
		t.Errorf("the first row of each series under --history is\n%s\nand `records` returns\n%s",
			strings.Join(firsts, "\n"), strings.Join(heads, "\n"))
	}
}

// recordStream is a case's `record` rows, in order, with the terminal row left
// off: the terminal row is the wire's framing rather than an answer, and it
// carries a truncation marker the two cases have no reason to share.
func recordStream(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var rows []string
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if strings.HasPrefix(line, `{"type":"record",`) {
			rows = append(rows, line)
		}
	}
	if len(rows) == 0 {
		t.Fatalf("%s holds no record row; the case would pass having compared nothing", path)
	}
	return rows
}

// recordKeyOf reads a row's identity back off the wire, which is the one member
// of it these two streams are grouped by.
func recordKeyOf(t *testing.T, line string) recordKey {
	t.Helper()

	var row struct {
		Key recordKey `json:"key"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		t.Fatalf("reading a record row back: %v", err)
	}
	return row.Key
}
