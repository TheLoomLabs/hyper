package store_test

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Compaction is the one act in the tool that removes anything, so the corpus it
// is tested against is built to tempt it (§7, issue #131). Every case here
// seeds a branch, compacts it, and reads back what is still on the branch with
// git — the predicate is a rule about files, and the test of it is the files
// that survived.

// thePolicy is the retention every case here acts under unless it names
// another: §7's own worked figure, and the one `hyper.yaml` in the fixtures
// declares.
var thePolicy = store.Retention{Declared: "90d", Age: 90 * 24 * time.Hour}

// The Run ids the interior cases seed with. They are further UUIDv7s beside
// series_test.go's three, one per version of a series long enough to have an
// interior at all.
const (
	theFourthRunID = "01991e24-6f2c-7e37-a04b-8fa05dc27316"
	theFifthRunID  = "01991e25-703d-7f48-b15c-90b16ed38427"
	theSixthRunID  = "01991e26-814e-7059-8260-a1c27fe49538"
)

// theAssetSeries and theSecondSeries are identities beside theSeries, so that a
// case can hold what one series lost against what another kept.
var (
	theAssetSeries  = store.Identity{Target: "cloudflare-prod", Definition: "preview-dns", Name: "asset-42.example.com"}
	theSecondSeries = store.Identity{Target: "cloudflare-prod", Definition: "uptime", Name: "status.hyper.dev"}
)

// ageing is one instant a number of days before another, which is how every
// case here states a version's age against the policy.
func ageing(days int) time.Time { return theInstant.AddDate(0, 0, -days) }

// compacted seeds a branch with the versions handed, compacts it under the
// policy at theInstant, and answers the repository, what was removed, and the
// branch's files afterwards.
//
// It opens the Store at theInstant, which is the clock retention is measured
// against: an age is a comparison between a file and a moment, and a case that
// let the moment be the wall clock would pass in April and fail in July.
func compacted(t *testing.T, retention store.Retention, versions ...store.RecordVersion) (*repo, store.Compaction, map[string]string) {
	t.Helper()

	r, held := seededStore(t, versions...)
	compaction, err := held.Compact(retention)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	return r, compaction, r.storeTree(r.root)
}

// paths is a branch's files by path alone, sorted — what a case asserts where
// what it is about is which files stand rather than what they hold.
func paths(tree map[string]string) []string { return slices.Sorted(maps.Keys(tree)) }

// TestCompact_RemovesTheInteriorObservationsOlderThanRetention is the
// predicate's positive half: a version that is not the Head, not the series'
// first, and older than the policy goes — and one younger than the policy stays,
// which is the same series answering both.
func TestCompact_RemovesTheInteriorObservationsOlderThanRetention(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(365))
	stale := aVersion(t, theSeries, theSecondRunID, 1, ageing(300))
	older := aVersion(t, theSeries, theThirdRunID, 1, ageing(200))
	young := aVersion(t, theSeries, theFourthRunID, 1, ageing(30))
	head := aVersion(t, theSeries, theFifthRunID, 1, ageing(1))

	_, compaction, tree := compacted(t, thePolicy, first, stale, older, young, head)

	want := []string{store.IntroductionPath, pathOf(first), pathOf(young), pathOf(head)}
	slices.Sort(want)
	if got := paths(tree); !slices.Equal(got, want) {
		t.Errorf("the branch holds %v, want %v", got, want)
	}

	removed := make([]string, 0, len(compaction.Removed))
	for _, version := range compaction.Removed {
		removed = append(removed, version.File)
	}
	if wanted := []string{pathOf(stale), pathOf(older)}; !slices.Equal(removed, wanted) {
		t.Errorf("removed %v, want %v in path order", removed, wanted)
	}
	if compaction.Untouched != 0 {
		t.Errorf("%d series untouched, want 0: the one series here lost two versions", compaction.Untouched)
	}
}

// TestCompact_NeverRemovesAHeadOrAFirstVersion is the predicate's two
// exclusions, at the age that would take everything else: every version in the
// series is years older than the policy, and the two at the ends still stand.
//
// A series of one version and a series of two are the same rule read at the
// ends, where the first version *is* the Head or sits directly beneath it, and
// they are here so that the interior being empty is answered by the range
// rather than by a comparison that could be off by one.
func TestCompact_NeverRemovesAHeadOrAFirstVersion(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))
	alone := aVersion(t, theSecondSeries, theFourthRunID, 1, ageing(900))
	pair := aVersion(t, theSecondSeries, theFifthRunID, 1, ageing(800))

	_, compaction, tree := compacted(t, thePolicy, first, middle, head, alone, pair)

	want := []string{store.IntroductionPath, pathOf(first), pathOf(head), pathOf(alone), pathOf(pair)}
	slices.Sort(want)
	if got := paths(tree); !slices.Equal(got, want) {
		t.Errorf("the branch holds %v, want %v — the ends of a series are never removable", got, want)
	}
	if compaction.Untouched != 1 {
		t.Errorf("%d series untouched, want 1: the series of two lost nothing", compaction.Untouched)
	}
}

// TestCompact_RemovesNothingFromASeriesHoldingAnAssetOrATombstone is the
// evidence half, and the reason the test reads the whole series rather than the
// version in front of it. An Asset is what hyper's effect reached and a
// Tombstone is the destruction of one; the Observations beneath either are the
// account of that Asset, and a maintenance command that pruned them would leave
// a destruction of a thing nothing ever described (§7, ADR-0006, ADR-0001).
func TestCompact_RemovesNothingFromASeriesHoldingAnAssetOrATombstone(t *testing.T) {
	asset := aVersion(t, theAssetSeries, theEntryRunID, 1, ageing(900))
	asset.RecordType = store.RecordAsset
	interior := aVersion(t, theAssetSeries, theSecondRunID, 1, ageing(800))
	interior.RecordType = store.RecordAsset
	assetHead := aVersion(t, theAssetSeries, theThirdRunID, 1, ageing(700))
	assetHead.RecordType = store.RecordAsset

	observed := aVersion(t, theSeries, theFourthRunID, 1, ageing(900))
	beneath := aVersion(t, theSeries, theFifthRunID, 1, ageing(800))
	tomb := aTombstone(t, theSeries, theSixthRunID, 1, ageing(700))

	_, compaction, tree := compacted(t, thePolicy, asset, interior, assetHead, observed, beneath, tomb)

	want := []string{store.IntroductionPath, pathOf(asset), pathOf(interior), pathOf(assetHead), pathOf(observed), pathOf(beneath), pathOf(tomb)}
	slices.Sort(want)
	if got := paths(tree); !slices.Equal(got, want) {
		t.Errorf("the branch holds %v, want every file it started with %v", got, want)
	}
	if len(compaction.Removed) != 0 {
		t.Errorf("removed %d versions, want none: no Asset version and no version of an Asset series is removable", len(compaction.Removed))
	}
	if compaction.Untouched != 2 {
		t.Errorf("%d series untouched, want both", compaction.Untouched)
	}
}

// TestCompact_LeavesTheJournalAndTheIntroductionWhereTheyAre is the rest of
// what is not removable, at an age no policy would keep. A Journal entry is the
// evidence a Step ran, and `STORE.md` is the branch introducing itself; neither
// is a Record version, and Compaction removes nothing else.
func TestCompact_LeavesTheJournalAndTheIntroductionWhereTheyAre(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))

	r, held := seededStore(t, first, middle, head)
	entry := store.JournalEntry{Run: runID(t, theFourthRunID), Started: ageing(900)}
	r.seedFiles(r.root, map[string]string{
		entry.RunPath():     "{\n  \"schema_version\": 1\n}\n",
		entry.StepPath(1):   "{\n  \"schema_version\": 1\n}\n",
		entry.OutcomePath(): "{\n  \"schema_version\": 1\n}\n",
	})
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := held.Compact(thePolicy); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	tree := r.storeTree(r.root)
	for _, path := range []string{store.IntroductionPath, entry.RunPath(), entry.StepPath(1), entry.OutcomePath()} {
		if _, held := tree[path]; !held {
			t.Errorf("%s is gone; Compaction removes interior Observation versions and nothing else", path)
		}
	}
	if _, held := tree[pathOf(middle)]; held {
		t.Errorf("%s stands; the interior Observation the policy covers is what this command removes", pathOf(middle))
	}
}

// TestCompact_RemovesNothingWhereEveryInteriorVersionIsYounger writes no commit
// at all. A Compaction that found nothing to remove has nothing to say in `git
// log`, and a branch whose tip moved would be an account of an act that did not
// happen.
func TestCompact_RemovesNothingWhereEveryInteriorVersionIsYounger(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(30))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(20))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(10))

	r, held := seededStore(t, first, middle, head)
	before := r.text("rev-parse", store.Ref)

	compaction, err := held.Compact(thePolicy)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}

	if len(compaction.Removed) != 0 {
		t.Errorf("removed %d versions, want none: every interior version is younger than the policy", len(compaction.Removed))
	}
	if after := r.text("rev-parse", store.Ref); after != before {
		t.Errorf("the branch moved from %s to %s; a Compaction that removed nothing commits nothing", before, after)
	}
}

// TestCompact_TheBoundaryIsInsideThePolicy is the comparison at the one instant
// it is decided at: a version exactly the policy's age stands, and one a
// millisecond older goes. A repository that agreed to keep its Records ninety
// days has not agreed to lose one on the ninetieth.
func TestCompact_TheBoundaryIsInsideThePolicy(t *testing.T) {
	for name, born := range map[string]struct {
		at      time.Time
		removed bool
	}{
		"exactly the policy's age": {theInstant.Add(-thePolicy.Age), false},
		"a millisecond older":      {theInstant.Add(-thePolicy.Age - time.Millisecond), true},
	} {
		t.Run(name, func(t *testing.T) {
			first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
			edge := aVersion(t, theSeries, theSecondRunID, 1, born.at)
			head := aVersion(t, theSeries, theThirdRunID, 1, theInstant)

			_, compaction, tree := compacted(t, thePolicy, first, edge, head)

			_, stands := tree[pathOf(edge)]
			if stands == born.removed {
				t.Errorf("%s stands: %v, want removed: %v", pathOf(edge), stands, born.removed)
			}
			if got := len(compaction.Removed) > 0; got != born.removed {
				t.Errorf("removed %d versions, want removed: %v", len(compaction.Removed), born.removed)
			}
		})
	}
}

// TestCompact_IsOneCommitWhoseMessageNamesTheCountAndThePolicy is §7's own
// account: `git log` on the branch is what says what a Compaction removed, so
// the message is a rendering with a reader. It names the count and the policy
// it acted under, in the artefact's own spelling of it, and nothing else.
func TestCompact_IsOneCommitWhoseMessageNamesTheCountAndThePolicy(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	other := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))
	head := aVersion(t, theSeries, theFourthRunID, 1, ageing(600))

	r, compaction, _ := compacted(t, thePolicy, first, middle, other, head)
	if len(compaction.Removed) != 2 {
		t.Fatalf("removed %d versions, want the two interior ones", len(compaction.Removed))
	}

	if depth := len(strings.Fields(r.text("rev-list", store.Ref))); depth != 3 {
		t.Errorf("the branch holds %d commits, want the root, the seed and one Compaction", depth)
	}
	message := r.text("log", "-1", "--format=%B", store.Ref)
	for _, want := range []string{"2 interior observation versions", thePolicy.Declared} {
		if !strings.Contains(message, want) {
			t.Errorf("the commit message is %q, want it to name %q", message, want)
		}
	}
	if identity := r.text("log", "-1", "--format=%an <%ae>", store.Ref); identity != store.CommitName+" <"+store.CommitEmail+">" {
		t.Errorf("the Compaction was authored by %q, want hyper's own identity", identity)
	}

	// The objects the removal wrote are objects git will read back forever.
	// The tree it builds is assembled a directory at a time rather than
	// through an index, and a tree's entries have a sort order of their own —
	// a directory sorts as though its name ended in a slash — so the check
	// that the branch is well formed is git's own (§7, ADR-0075).
	// The assertion is the exit code: `fsck` writes its complaints to stderr
	// and exits non-zero, and the fixture stops the test on any git call that
	// does — so a tree whose entries came out in the wrong order fails here
	// with git's own words rather than silently.
	r.git("fsck", "--strict", "--no-dangling", "--no-progress")
}

// TestCompact_MovesTheOrdinal is ADR-0049's consequence stated as a property,
// and it is the reason no removed version is ever named by one: an ordinal is a
// position in a series' ordering, and this command is precisely the thing that
// renumbers it.
func TestCompact_MovesTheOrdinal(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(10))

	r, held := seededStore(t, first, middle, head)
	before, _, err := held.Head(theSeries)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if before.Ordinal != 3 {
		t.Fatalf("the Head is ordinal %d, want 3", before.Ordinal)
	}

	if _, err := held.Compact(thePolicy); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	after, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	head2, found, err := after.Head(theSeries)
	if err != nil || !found {
		t.Fatalf("Head after the Compaction: %v, found=%v", err, found)
	}
	if head2.File != before.File {
		t.Errorf("the Head is %s, want the same version %s: a Head is never removable", head2.File, before.File)
	}
	if head2.Ordinal != 2 {
		t.Errorf("the Head is ordinal %d, want 2 — removing an interior version renumbers every version above it", head2.Ordinal)
	}
}
