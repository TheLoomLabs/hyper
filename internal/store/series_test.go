package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Head is derived, and everything here is a fact about a listing (§7,
// ADR-0011). Nothing in the Store points at the current version: a series'
// versions are ordered on the `written_at` each file carries, ties broken by
// the file name, so what a case seeds is files and what it reads back is the
// order they fall into (issue #130).

// theSeries is the identity the reader's cases seed versions of. It is §7's own
// worked example, so the file a case seeds is the file the specification
// publishes.
var theSeries = store.Identity{Target: "cloudflare-prod", Definition: "preview-dns", Name: "preview-42.example.com"}

// The Run ids the seeded versions are written by. theEntryRunID is §7's own;
// the other two are further UUIDv7s, ordered as their text is, so a case that
// turns on the tie-break can state which file name comes first.
const (
	theSecondRunID = "01991e22-4d0a-7c15-8e29-6d8f3ba05194"
	theThirdRunID  = "01991e23-5e1b-7d26-9f3a-7e9f4cb16205"
)

// theEarlierInstant and theLaterInstant are theInstant's neighbours, an hour
// either side, for the ordering the Head is derived from.
var (
	theEarlierInstant = theInstant.Add(-time.Hour)
	theLaterInstant   = theInstant.Add(time.Hour)
)

// runID reads one of §12's ids, stopping a case whose own fixture wrote one
// that is not a UUIDv7.
func runID(t *testing.T, text string) store.RunID {
	t.Helper()

	id, err := store.ParseRunID(text)
	if err != nil {
		t.Fatalf("ParseRunID(%q): %v", text, err)
	}
	return id
}

// aVersion is one ordinary version of a series: which series, what wrote it,
// and when. Its content carries the instant, so that two versions of one series
// are two files that differ in the way the Store's own do — a version is minted
// only where the bytes moved (§7).
func aVersion(t *testing.T, id store.Identity, run string, step int, at time.Time) store.RecordVersion {
	t.Helper()

	return store.RecordVersion{
		Metadata: store.Metadata{
			Identity:   id,
			RecordType: store.RecordObservation,
			Run:        runID(t, run),
			Step:       step,
			Operation:  "get_dns_record",
			WrittenAt:  at,
			Provenance: theProvenance,
		},
		Fields: store.Mapping{"observed_at": store.String(at.Format(time.RFC3339))},
	}
}

// aTombstone is the same version recording that what it described was
// destroyed. It is an ordinary version of the series carrying one marker, and
// not a shape of its own (§7, ADR-0011).
func aTombstone(t *testing.T, id store.Identity, run string, step int, at time.Time) store.RecordVersion {
	t.Helper()

	version := aVersion(t, id, run, step, at)
	version.RecordType = store.RecordAsset
	version.Operation = "delete_dns_record"
	version.Tombstone = true
	return version
}

// pathOf is where a seeded version sits: the path its own identity, Run and
// Step build, which is the one a case asserts against and the one the reader
// answers with.
func pathOf(version store.RecordVersion) string {
	return store.RecordPath(version.Identity, version.Run, version.Step)
}

// seededStore is a repository whose Store branch holds the versions handed, and
// the handle on it. It is the shape every case below opens with: a branch that
// exists, files on it, and nothing else anywhere.
func seededStore(t *testing.T, versions ...store.RecordVersion) (*repo, *store.Store) {
	t.Helper()

	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(versions) > 0 {
		r.seedVersions(r.root, versions...)
	}
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r, held
}

// TestSeries_OrdersOnWrittenAtAndDerivesTheHead is the whole of the derivation:
// the versions of a series in the order their own `written_at` puts them in,
// the last of them the Head, and the ordinal a position in that ordering.
//
// The files are seeded out of order — the newest first — so that a reader
// answering the listing's own order rather than the files' would fail here.
func TestSeries_OrdersOnWrittenAtAndDerivesTheHead(t *testing.T) {
	_, held := seededStore(t,
		aVersion(t, theSeries, theThirdRunID, 1, theLaterInstant),
		aVersion(t, theSeries, theEntryRunID, 1, theEarlierInstant),
		aVersion(t, theSeries, theSecondRunID, 1, theInstant),
	)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}

	want := []time.Time{theEarlierInstant, theInstant, theLaterInstant}
	if got := writtenAt(series); !slices.Equal(got, want) {
		t.Errorf("the series is ordered %v, want %v", got, want)
	}
	for i, version := range series.Versions {
		if version.Ordinal != i+1 {
			t.Errorf("the version at %s carries ordinal %d, want %d — the first version of a series is 1", version.WrittenAt, version.Ordinal, i+1)
		}
	}

	head, found := series.Head()
	if !found {
		t.Fatal("the series answers no Head; three versions were seeded")
	}
	if !head.WrittenAt.Equal(theLaterInstant) {
		t.Errorf("the Head is the version at %s, want the greatest written_at, %s", head.WrittenAt, theLaterInstant)
	}
	if head.Run.String() != theThirdRunID {
		t.Errorf("the Head was written by %s, want %s", head.Run, theThirdRunID)
	}
}

// TestSeries_OfOneVersionHasThatVersionAsItsHead, which is the degenerate case
// of the same rule and the one every series passes through.
func TestSeries_OfOneVersionHasThatVersionAsItsHead(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	head, found := series.Head()
	if !found {
		t.Fatal("the series answers no Head; one version was seeded")
	}
	if head.Ordinal != 1 || !head.WrittenAt.Equal(theInstant) {
		t.Errorf("the Head is ordinal %d at %s, want ordinal 1 at %s", head.Ordinal, head.WrittenAt, theInstant)
	}
}

// TestSeries_BreaksATieOnTheFileNameByteWise is §7's rule at §7's grain.
// ADR-0011 says the Run id and §7 says the file name; they are the same rule at
// two grains, and the finer one is the one to implement — two Steps of one Run
// writing one identity write two paths that the coarser reading could not
// order.
func TestSeries_BreaksATieOnTheFileNameByteWise(t *testing.T) {
	_, held := seededStore(t,
		aVersion(t, theSeries, theSecondRunID, 1, theInstant),
		aVersion(t, theSeries, theEntryRunID, 1, theInstant),
	)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	want := []string{theEntryRunID, theSecondRunID}
	if got := runsOf(series); !slices.Equal(got, want) {
		t.Errorf("two versions sharing an instant are ordered %v, want %v — the tie is the file name, byte-wise", got, want)
	}

	// The tie is broken identically on repeated reads, which is what makes
	// it an ordering rather than a coin toss: a Head that moved between two
	// reads of one branch would move an ordinal with it.
	again, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("the second Series: %v", err)
	}
	if got := runsOf(again); !slices.Equal(got, want) {
		t.Errorf("a second read orders them %v, want the first read's %v", got, want)
	}
}

// TestSeries_OrdersTwoStepsOfOneRunByTheirStepNumber is the case the file name
// exists to separate: one Run writing one identity at two Steps writes two
// paths, and the Run id alone could not order them.
func TestSeries_OrdersTwoStepsOfOneRunByTheirStepNumber(t *testing.T) {
	_, held := seededStore(t,
		aVersion(t, theSeries, theEntryRunID, 2, theInstant),
		aVersion(t, theSeries, theEntryRunID, 1, theInstant),
	)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if got := stepsOf(series); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("two Steps of one Run are ordered %v, want [1 2]", got)
	}
	if head, _ := series.Head(); head.Step != 2 {
		t.Errorf("the Head is step %d, want the later Step", head.Step)
	}
}

// TestSeries_AnswersTheMetadataWithoutTheFields. Ordering a series opens every
// version of it, and what a listing answers is what naming and ordering need —
// the content is a read of its own, so a caller that does not need `fields`
// never holds them.
func TestSeries_AnswersTheMetadataWithoutTheFields(t *testing.T) {
	seeded := aVersion(t, theSeries, theEntryRunID, 1, theInstant)
	_, held := seededStore(t, seeded)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	head, _ := series.Head()
	if head.Metadata != seeded.Metadata {
		t.Errorf("the Head's metadata is %+v, want the seeded %+v", head.Metadata, seeded.Metadata)
	}
	if head.File != pathOf(seeded) {
		t.Errorf("the Head sits at %q, want %q", head.File, pathOf(seeded))
	}

	version, err := held.Read(head)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(version, seeded) {
		t.Errorf("the version reads back as %+v, want the seeded %+v", version, seeded)
	}
}

// TestSeries_ReadsTheIdentityFromTheFileAndNotFromThePath. The path grammar
// truncates an over-long segment and suffixes a digest (§12), so *which series
// is this* is answered inside the file and nowhere else — and the identity a
// long one answers is the whole of it, unencoded.
func TestSeries_ReadsTheIdentityFromTheFileAndNotFromThePath(t *testing.T) {
	long := store.Identity{
		Target:     theSeries.Target,
		Definition: theSeries.Definition,
		Name:       strings.Repeat("preview-", 40) + "42.example.com",
	}
	_, held := seededStore(t, aVersion(t, long, theEntryRunID, 1, theInstant))

	series, err := held.Series(long)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	head, found := series.Head()
	if !found {
		t.Fatal("the series answers no Head; one version was seeded")
	}
	if head.Identity != long {
		t.Errorf("the version's identity is %+v, want the whole of %+v", head.Identity, long)
	}
	if !strings.Contains(head.File, "~") {
		t.Errorf("the seeded path is %q, which is not the truncated one this case is about", head.File)
	}

	// And the file opens by the name it sits under, escapes, digest suffix
	// and all — the path a truncated identity builds is a path, not a
	// spelling only a listing can use.
	if _, err := held.Read(head); err != nil {
		t.Errorf("Read over %q: %v", head.File, err)
	}
}

// TestSeries_AnswersNothingForASeriesTheStoreDoesNotHold. A series with no
// versions is not a fault: it is the answer *hyper has never recorded this*,
// which is what every first Run of every Step reads.
func TestSeries_AnswersNothingForASeriesTheStoreDoesNotHold(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	other := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "preview-99.example.com"}
	series, err := held.Series(other)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if len(series.Versions) != 0 {
		t.Errorf("the series holds %d versions, want none", len(series.Versions))
	}
	if _, found := series.Head(); found {
		t.Error("a series the Store does not hold answered a Head")
	}
}

// TestHead_ReadsDeadUnderATombstoneAndAliveAgainAboveOne. A Tombstone is
// terminal for the Asset's life and not for the series: a further version above
// it makes the Head alive again, which is what makes destroy-then-recreate
// behave as §6 states under `skip-if-recorded`.
func TestHead_ReadsDeadUnderATombstoneAndAliveAgainAboveOne(t *testing.T) {
	r, held := seededStore(t,
		aVersion(t, theSeries, theEntryRunID, 1, theEarlierInstant),
		aTombstone(t, theSeries, theSecondRunID, 1, theInstant),
	)

	head, found, err := held.Head(theSeries)
	if err != nil || !found {
		t.Fatalf("Head = %v, %v", found, err)
	}
	if !head.Tombstone {
		t.Errorf("the Head at %s carries no tombstone; the series reads alive after a destruction", head.WrittenAt)
	}

	r.seedVersions(r.root, aVersion(t, theSeries, theThirdRunID, 1, theLaterInstant))
	reopened, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	head, found, err = reopened.Head(theSeries)
	if err != nil || !found {
		t.Fatalf("Head = %v, %v", found, err)
	}
	if head.Tombstone {
		t.Error("the Head is still the Tombstone; a further version above one makes the series read alive again")
	}
}

// TestRead_TakesNoOrdinal is ADR-0049's rule at this seam. Nothing anywhere
// accepts an ordinal as an input — naming a version is naming its Run — so a
// Version carrying an ordinal that is not its own still reads the file it
// names, the position being derived and never consulted.
func TestRead_TakesNoOrdinal(t *testing.T) {
	seeded := aVersion(t, theSeries, theEntryRunID, 1, theInstant)
	_, held := seededStore(t, seeded, aVersion(t, theSeries, theSecondRunID, 1, theLaterInstant))

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	first := series.Versions[0]
	first.Ordinal = 99

	version, err := held.Read(first)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !reflect.DeepEqual(version, seeded) {
		t.Errorf("the version reads back as %+v, want the one its file names, %+v", version, seeded)
	}
}

// TestSeries_TheOrdinalMovesWhenAnInteriorVersionIsRemoved is ADR-0049 as the
// property it is. The ordinal is a position and is stored nowhere, so removing
// an interior version renumbers every version above it — which is what
// Compaction does and why nothing takes one as input.
func TestSeries_TheOrdinalMovesWhenAnInteriorVersionIsRemoved(t *testing.T) {
	interior := aVersion(t, theSeries, theSecondRunID, 1, theInstant)
	top := aVersion(t, theSeries, theThirdRunID, 1, theLaterInstant)
	r, held := seededStore(t,
		aVersion(t, theSeries, theEntryRunID, 1, theEarlierInstant),
		interior,
		top,
	)

	before, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	if got, _ := before.Head(); got.Ordinal != 3 {
		t.Fatalf("the Head is ordinal %d of three versions, want 3", got.Ordinal)
	}

	r.removeFromStore(pathOf(interior))
	reopened, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	after, err := reopened.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}

	head, _ := after.Head()
	if head.Ordinal != 2 {
		t.Errorf("the Head is ordinal %d, want 2 — an interior version was removed and the ordinal moved", head.Ordinal)
	}
	if head.Run.String() != top.Run.String() {
		t.Errorf("the Head was written by %s, want the same version as before, %s", head.Run, top.Run)
	}
}

// TestRecords_ListsEverySeriesTheBranchHolds, each identity read from the files
// rather than decoded from the lossy path.
func TestRecords_ListsEverySeriesTheBranchHolds(t *testing.T) {
	other := store.Identity{Target: "cloudflare-staging", Definition: "preview-dns", Name: "preview-7.example.com"}
	third := store.Identity{Target: theSeries.Target, Definition: "zone-facts", Name: "example.com"}
	_, held := seededStore(t,
		aVersion(t, theSeries, theEntryRunID, 1, theEarlierInstant),
		aVersion(t, theSeries, theSecondRunID, 1, theInstant),
		aVersion(t, other, theEntryRunID, 2, theInstant),
		aVersion(t, third, theEntryRunID, 3, theInstant),
	)

	records, err := held.Records()
	if err != nil {
		t.Fatalf("Records: %v", err)
	}

	// The order is the identity's own — Target, then Definition, then name —
	// and never the listing's: escaping drags every escaped character to the
	// left of every unreserved one, so a path order is an order over the
	// encoding rather than over the names anybody wrote (§12, ADR-0044).
	want := []store.Identity{theSeries, third, other}
	got := make([]store.Identity, len(records))
	for i, series := range records {
		got[i] = series.Identity
	}
	if !slices.Equal(got, want) {
		t.Errorf("the branch holds %v, want %v", got, want)
	}
	if versions := len(records[0].Versions); versions != 2 {
		t.Errorf("%v holds %d versions, want 2", records[0].Identity, versions)
	}
}

// TestRecords_OfABranchHoldingNoSeriesIsEmpty. A Store that has recorded
// nothing is the state every Store begins in — `store init` creates a branch
// holding STORE.md and nothing else — and reading it is an empty enumeration
// rather than a fault.
func TestRecords_OfABranchHoldingNoSeriesIsEmpty(t *testing.T) {
	_, held := seededStore(t)

	records, err := held.Records()
	if err != nil {
		t.Fatalf("Records: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("the branch holds %v, want nothing", records)
	}
}

// TestRead_SurfacesTheSchemaCeilingRatherThanGuessing. A file above the
// reader's ceiling is neither read nor skipped: the condition goes to the
// caller, which names the path and renders the Refusal (§7, ADR-0028).
func TestRead_SurfacesTheSchemaCeilingRatherThanGuessing(t *testing.T) {
	seeded := aVersion(t, theSeries, theEntryRunID, 1, theInstant)
	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	ahead := strings.Replace(string(seeded.Encode()), `"schema_version": 1`, `"schema_version": 2`, 1)
	r.seedFiles(r.root, map[string]string{pathOf(seeded): ahead})

	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, err = held.Series(theSeries)

	var unsupported store.SchemaUnsupported
	if !errors.As(err, &unsupported) {
		t.Fatalf("Series returned %v, want the store-schema-unsupported condition", err)
	}
	if unsupported.Written != 2 || unsupported.Known != store.RecordSchemaVersion {
		t.Errorf("the condition is %+v, want the file's 2 against this reader's %d", unsupported, store.RecordSchemaVersion)
	}
	if !strings.Contains(err.Error(), pathOf(seeded)) {
		t.Errorf("the error is %q, want it to name the file it could not read", err)
	}
}

// TestReading_WritesNothingToDisk is ADR-0075 as the observable fact it is. No
// worktree, no temporary directory, no hidden checkout, and no byte of Store
// content as an ordinary file anywhere — including under `.git/hyper/`, which
// §7 permits derived state in and this package builds none of.
//
// The lock is the one thing that puts `.git/hyper/` there, and it is not
// reached from here: reading the Store takes no lock, a Run does (§6,
// lock.go). So the assertion below is about what a read leaves behind and
// stays exactly as strict as it was.
func TestReading_WritesNothingToDisk(t *testing.T) {
	r, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))
	before := r.workingTree()

	if _, err := held.Records(); err != nil {
		t.Fatalf("Records: %v", err)
	}
	if _, _, err := held.Head(theSeries); err != nil {
		t.Fatalf("Head: %v", err)
	}

	if after := r.workingTree(); !slices.Equal(before, after) {
		t.Errorf("the repository root holds %v, want it left as %v", after, before)
	}
	if _, err := os.Stat(filepath.Join(r.root, ".git", "hyper")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".git/hyper exists (%v); no answer here depends on derived state and none is built (§7)", err)
	}
	if worktrees := r.text("worktree", "list"); strings.Count(worktrees, "\n") != 0 {
		t.Errorf("the repository has more than one worktree:\n%s", worktrees)
	}
}

// writtenAt, runsOf and stepsOf are a series' ordering read off as the one
// thing a case is about, so that an assertion reads as the order it states.
func writtenAt(series store.Series) []time.Time {
	instants := make([]time.Time, len(series.Versions))
	for i, version := range series.Versions {
		instants[i] = version.WrittenAt
	}
	return instants
}

func runsOf(series store.Series) []string {
	runs := make([]string, len(series.Versions))
	for i, version := range series.Versions {
		runs[i] = version.Run.String()
	}
	return runs
}

func stepsOf(series store.Series) []int {
	steps := make([]int, len(series.Versions))
	for i, version := range series.Versions {
		steps[i] = version.Step
	}
	return steps
}

// workingTree is every entry in the repository root, `.git` included, as one
// sorted listing: what a read must leave exactly as it found.
func (r *repo) workingTree() []string {
	r.t.Helper()

	var paths []string
	root := filepath.Clean(r.root)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(path, filepath.Join(root, ".git")+string(filepath.Separator)) {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(path, root))
		return nil
	}); err != nil {
		r.t.Fatal(err)
	}
	slices.Sort(paths)
	return paths
}

// removeFromStore takes one path off the Store branch as a further commit,
// which is what a Compaction is and what an ordinal moves under.
func (r *repo) removeFromStore(path string) {
	r.t.Helper()

	index := filepath.Join(r.t.TempDir(), "index")
	env := append(slices.Clone(r.env), "GIT_INDEX_FILE="+index)
	parent := r.text("rev-parse", store.Ref)
	r.runWith(r.root, env, nil, "read-tree", parent)
	r.runWith(r.root, env, nil, "update-index", "--force-remove", path)
	tree := strings.TrimSpace(string(r.runWith(r.root, env, nil, "write-tree")))
	r.git("update-ref", store.Ref, r.text("commit-tree", tree, "-m", "a Compaction", "-p", parent))
}
