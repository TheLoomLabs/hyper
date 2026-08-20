package store_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The sync is what puts the branch in hand, and the whole of what it decides is
// the depth (§7, ADR-0074). A clone that lacks the Store takes the tip and no
// history; a clone that holds it fetches incrementally and names no depth, so a
// Store held whole stays whole and one held shallow is never deepened. Neither
// path is ever filtered.
//
// Every assertion here is read back with git out of a real repository and a real
// bare remote, on store_test.go's own reasoning: §7 writes down what a fetch
// leaves behind, and the test of it is what the fetch left behind (issue #130).

// TestSync_TakesTheTipAndNoHistoryWhereTheCloneLacksTheBranch is the runner's
// case, and it is every runner: `actions/checkout` takes one ref, so the Store
// is absent from the clone and arrives here. The depth is decided at this one
// moment, which is the moment there is nothing to preserve.
func TestSync_TakesTheTipAndNoHistoryWhereTheCloneLacksTheBranch(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	r.seedStore(bare, store.IntroductionPath, store.Introduction)
	tip := r.seedVersions(bare, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if got := r.text("rev-parse", store.Ref); got != tip {
		t.Errorf("%s is %s, want the remote's %s", store.Ref, got, tip)
	}
	if shallow := r.text("rev-parse", "--is-shallow-repository"); shallow != "true" {
		t.Errorf("the clone is shallow: %s, want the tip and no history", shallow)
	}
	if boundary := r.shallowBoundary(); len(boundary) != 1 || boundary[0] != tip {
		t.Errorf("the shallow boundary is %v, want %s alone", boundary, tip)
	}
	if depth := len(strings.Fields(r.text("rev-list", store.Ref))); depth != 1 {
		t.Errorf("the branch holds %d commits, want the tip alone", depth)
	}
}

// TestSync_LeavesAStoreItHoldsWholeWhole is the other arm and the rule that
// makes it one decision rather than two: a clone that holds the Store's history
// is fetched incrementally, with no depth named, so nothing shortens a branch
// `hyper` did not create shallow.
//
// It runs against both ways a clone comes to hold one. The second is the one
// that matters: an ordinary `git clone` leaves every branch as a remote-tracking
// ref and its history whole, and a depth-1 fetch there would cut a history the
// clone already had — through the one path where it looks like an arrival.
func TestSync_LeavesAStoreItHoldsWholeWhole(t *testing.T) {
	for name, hold := range map[string]func(r *repo){
		"as its own branch": func(r *repo) {
			r.git("fetch", "--quiet", "origin", store.Ref+":"+store.Ref)
		},
		"as a remote-tracking ref alone": func(r *repo) {
			r.git("fetch", "--quiet", "origin")
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			bare := r.origin()
			r.seedStore(bare, store.IntroductionPath, store.Introduction)
			r.seedVersions(bare, aVersion(t, theSeries, theEntryRunID, 1, theInstant))
			hold(r)
			tip := r.seedVersions(bare, aVersion(t, theSeries, theSecondRunID, 1, theLaterInstant))

			if err := store.Sync(r.root, theInstant); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			if got := r.text("rev-parse", store.Ref); got != tip {
				t.Errorf("%s is %s, want the remote's %s — the fetch is incremental, not absent", store.Ref, got, tip)
			}
			if shallow := r.text("rev-parse", "--is-shallow-repository"); shallow != "false" {
				t.Errorf("the clone is shallow: %s; a Store held whole is left whole", shallow)
			}
			if depth := len(strings.Fields(r.text("rev-list", store.Ref))); depth != 3 {
				t.Errorf("the branch holds %d commits, want all three — nothing was shortened", depth)
			}
		})
	}
}

// TestSync_NeverDeepensAStoreItCreated is the same rule read from the other
// end. A shallow Store stays shallow: `hyper` decides the depth once, and a
// second sync is not a second chance to decide it.
func TestSync_NeverDeepensAStoreItCreated(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	r.seedStore(bare, store.IntroductionPath, store.Introduction)
	r.seedVersions(bare, aVersion(t, theSeries, theEntryRunID, 1, theInstant))
	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("the first Sync: %v", err)
	}
	tip := r.seedVersions(bare, aVersion(t, theSeries, theSecondRunID, 1, theLaterInstant))

	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("the second Sync: %v", err)
	}

	if got := r.text("rev-parse", store.Ref); got != tip {
		t.Errorf("%s is %s, want the remote's %s", store.Ref, got, tip)
	}
	if shallow := r.text("rev-parse", "--is-shallow-repository"); shallow != "true" {
		t.Errorf("the clone is shallow: %s, want a Store hyper fetched shallow left shallow", shallow)
	}
	if depth := len(strings.Fields(r.text("rev-list", store.Ref))); depth > 2 {
		t.Errorf("the branch holds %d commits; nothing here deepens the Store", depth)
	}
}

// TestSync_LeavesABranchHoldingWhatTheRemoteDoesNot is the state §7 calls
// ordinary and a sync must survive: a Run wrote to the Store and could not push
// it. What it wrote stands locally and goes out with the next Run that pushes,
// which re-applies the whole unpushed set onto the fetched tip — so the sync
// brings the remote's tip down and leaves the branch exactly where it is.
func TestSync_LeavesABranchHoldingWhatTheRemoteDoesNot(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	r.seedStore(bare, store.IntroductionPath, store.Introduction)
	r.git("fetch", "--quiet", "origin", store.Ref+":"+store.Ref)

	unpushed := r.seedVersions(r.root, aVersion(t, theSeries, theEntryRunID, 1, theInstant))
	published := r.seedVersions(bare, aVersion(t, theSeries, theSecondRunID, 1, theLaterInstant))

	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("Sync over a branch holding what the remote does not: %v", err)
	}
	if got := r.text("rev-parse", store.Ref); got != unpushed {
		t.Errorf("%s is %s, want the unpushed %s left standing", store.Ref, got, unpushed)
	}
	if got := r.text("rev-parse", "refs/remotes/"+store.RemoteName+"/"+store.BranchName); got != published {
		t.Errorf("the tracking ref is %s, want the remote's %s — the tip came down", got, published)
	}
}

// TestSync_IsNeverFiltered is ADR-0074's second half, on both paths. A version's
// `written_at` sits inside the file, so ordering a series opens every version of
// it — and under a blob or tree filter each of those is a lazy fetch, which is
// what would make *a read-only Run proceeds offline* false wherever the network
// is.
func TestSync_IsNeverFiltered(t *testing.T) {
	for name, hold := range map[string]func(r *repo){
		"on the branch's arrival": func(*repo) {},
		"on a branch already here": func(r *repo) {
			r.git("fetch", "--quiet", "origin", store.Ref+":"+store.Ref)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			bare := r.origin()
			r.seedStore(bare, store.IntroductionPath, store.Introduction)
			r.seedVersions(bare, aVersion(t, theSeries, theEntryRunID, 1, theInstant))
			hold(r)

			if err := store.Sync(r.root, theInstant); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			for _, setting := range []string{"remote.origin.promisor", "remote.origin.partialclonefilter"} {
				if value, set := r.setting(setting); set {
					t.Errorf("%s is %q; the Store's content is always read and the fetch is never filtered", setting, value)
				}
			}
			// A promisor pack is the other half of the same fact:
			// every object the branch names is here, so nothing is
			// a lazy fetch.
			if missing := r.text("rev-list", "--objects", "--missing=print", store.Ref); strings.Contains(missing, "?") {
				t.Errorf("the branch names objects the clone does not hold:\n%s", missing)
			}
		})
	}
}

// TestSync_ReportsNoFailureForARemoteThatHoldsNoBranch. The Store here stands
// and publishing it is the next push's — the state an `init` whose push was
// rejected leaves behind, which §7 says is reachable without anybody doing
// anything wrong.
func TestSync_ReportsNoFailureForARemoteThatHoldsNoBranch(t *testing.T) {
	r := newRepo(t)
	r.origin()
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	before := r.text("rev-parse", store.Ref)
	r.gitIn(r.text("config", "remote.origin.url"), "update-ref", "-d", store.Ref)

	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("Sync against a remote holding no branch: %v", err)
	}
	if after := r.text("rev-parse", store.Ref); after != before {
		t.Errorf("%s moved from %s to %s", store.Ref, before, after)
	}
}

// TestSync_ReadsTheLocalBranchWithNoRemoteConfigured: a repository that has
// never had a remote is not a repository whose Store is missing. Nothing here
// reaches a network and nothing reports the absent remote as a failure.
func TestSync_ReadsTheLocalBranchWithNoRemoteConfigured(t *testing.T) {
	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	before := r.text("rev-parse", store.Ref)

	if err := store.Sync(r.root, theInstant); err != nil {
		t.Fatalf("Sync with no remote configured: %v", err)
	}
	if after := r.text("rev-parse", store.Ref); after != before {
		t.Errorf("%s moved from %s to %s", store.Ref, before, after)
	}
}

// TestSync_AnswersAbsentWhereNeitherSideHoldsTheBranch is the sentinel a caller
// renders as `store-absent`. The branch is created by an explicit act and never
// by a Run, read-only Runs included (§7), so a sync that finds nothing anywhere
// says so rather than creating one.
func TestSync_AnswersAbsentWhereNeitherSideHoldsTheBranch(t *testing.T) {
	for name, wire := range map[string]func(r *repo){
		"with no remote configured": func(*repo) {},
		"with a remote that does not hold it": func(r *repo) {
			r.origin()
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			wire(r)

			err := store.Sync(r.root, theInstant)
			if !errors.Is(err, store.ErrAbsent) {
				t.Errorf("Sync returned %v, want it to be ErrAbsent", err)
			}
			if r.hasStoreBranch(r.root) {
				t.Errorf("%s was created by a sync; the branch is created by an explicit act and never by a Run (§7)", store.Ref)
			}
		})
	}
}

// TestSync_ReportsARemoteItCannotReach is the world resisting rather than a
// guardrail declining, and the two are never folded together: a remote that
// could not be reached, read as a remote holding nothing, is exactly how a
// second orphan root gets minted (ADR-0074).
func TestSync_ReportsARemoteItCannotReach(t *testing.T) {
	r := newRepo(t)
	r.git("remote", "add", "origin", filepath.Join(t.TempDir(), "no-repository-here.git"))

	err := store.Sync(r.root, theInstant)
	if err == nil {
		t.Fatal("Sync against an unreachable remote returned no error")
	}
	if errors.Is(err, store.ErrAbsent) {
		t.Errorf("Sync returned %v, want the world resisting and not ErrAbsent", err)
	}
}

// TestSync_NamesARepositoryRootThatIsNotOne answers Init's own sentinel, this
// being the same usage error one call over.
func TestSync_NamesARepositoryRootThatIsNotOne(t *testing.T) {
	if err := store.Sync(t.TempDir(), theInstant); !errors.Is(err, store.ErrNoRepository) {
		t.Errorf("Sync returned %v, want it to be ErrNoRepository", err)
	}
}

// TestOpen_AnswersAbsentWhereTheBranchIsNotInTheClone. Open reaches no network:
// putting the branch in hand is Sync's, and what Open answers is what the clone
// holds now.
func TestOpen_AnswersAbsentWhereTheBranchIsNotInTheClone(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	r.seedStore(bare, store.IntroductionPath, store.Introduction)

	_, err := store.Open(r.root, theInstant)
	if !errors.Is(err, store.ErrAbsent) {
		t.Errorf("Open returned %v, want it to be ErrAbsent — the branch is on the remote and not here", err)
	}
}

// TestOpen_ReadsTheBranchTheCloneHolds, and reaches nothing else while doing it.
func TestOpen_ReadsTheBranchTheCloneHolds(t *testing.T) {
	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// A remote that could not be reached at all, to hold Open to reaching
	// no network: a read of the clone is a read of the clone.
	r.git("remote", "add", "origin", filepath.Join(t.TempDir(), "no-repository-here.git"))

	if _, err := store.Open(r.root, theInstant); err != nil {
		t.Errorf("Open: %v", err)
	}
}

// shallowBoundary is what `.git/shallow` holds: the commits the clone has no
// history beneath. A depth-1 fetch of one ref leaves that ref's tip alone.
func (r *repo) shallowBoundary() []string {
	r.t.Helper()

	content, err := os.ReadFile(filepath.Join(r.root, ".git", "shallow"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		r.t.Fatal(err)
	}
	return strings.Fields(string(content))
}

// setting reads one git configuration value, and whether it is set at all.
func (r *repo) setting(name string) (string, bool) {
	r.t.Helper()

	cmd := exec.Command("git", "config", "--get", name)
	cmd.Dir, cmd.Env = r.root, r.env
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
