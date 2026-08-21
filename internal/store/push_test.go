package store_test

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The push, and what a rejected one does about it (§7, issue #131).
//
// Every case here builds a real bare repository as `origin` and moves it under
// the push it is testing, because that is the only way the condition arises at
// all: a second environment wrote to the branch between this one's sync and its
// push. What is asserted afterwards is what the remote holds, which is the
// whole of what a push is for.

// seedWithout adds one commit to the Store branch of the repository at gitdir
// with those paths off it, and answers the new commit. It is seedFiles' other
// half, and it exists because the case §7 describes only in terms of writes is
// a *removal* arriving from somewhere else (issue #131).
//
// The tree is assembled through a temporary index for seedFiles' own reason:
// the package it is testing never checks the Store out, and a fixture that
// needed a worktree would be testing a shape §7 forbids (ADR-0075).
func (r *repo) seedWithout(gitdir string, paths ...string) string {
	r.t.Helper()

	index := filepath.Join(r.t.TempDir(), "index")
	env := append(slices.Clone(r.env), "GIT_INDEX_FILE="+index)

	parent := r.textIn(gitdir, "rev-parse", store.Ref)
	r.runWith(r.root, env, nil, "read-tree", parent)
	for _, path := range paths {
		r.runWith(r.root, env, nil, "update-index", "--force-remove", path)
	}
	tree := strings.TrimSpace(string(r.runWith(r.root, env, nil, "write-tree")))

	written := r.text("commit-tree", tree, "-p", parent, "-m", "another environment compacted first")
	if gitdir == r.root {
		r.git("update-ref", store.Ref, written)
		return written
	}
	r.git("push", "--quiet", gitdir, written+":"+store.Ref)
	return written
}

// rejectAndAdvance installs a pre-receive hook on the bare repository that
// moves the Store branch on and then refuses the push — one second writer
// getting in first, every time, deterministically.
//
// It is what makes *rejected three times* a case rather than a story: the
// condition is the remote moving under each attempt, and a hook that only
// refused would be a push failing for its own reason, which is the other arm
// and exits through a different door.
func (r *repo) rejectAndAdvance(bare string) {
	r.t.Helper()

	hook := filepath.Join(bare, "hooks", "pre-receive")
	script := "#!/bin/sh\n" +
		// git runs a receive hook with the incoming objects quarantined
		// and ref updates forbidden inside it, so the variables that
		// impose the quarantine are dropped first: the commit this hook
		// builds is made of objects the repository already holds.
		"unset GIT_QUARANTINE_PATH GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES\n" +
		"tree=$(git rev-parse " + store.Ref + "^{tree})\n" +
		"advanced=$(git -c user.name=other -c user.email=other@elsewhere.invalid commit-tree \"$tree\" -p " + store.Ref + " -m 'another environment got there first')\n" +
		"git update-ref " + store.Ref + " \"$advanced\"\n" +
		"exit 1\n"
	if err := os.WriteFile(hook, []byte(script), 0o755); err != nil {
		r.t.Fatal(err)
	}
}

// published is the Store branch's files at the bare remote, by path.
func (r *repo) published(bare string) []string {
	r.t.Helper()
	return slices.Sorted(maps.Keys(r.storeTree(bare)))
}

// TestPush_SendsTheCompactionWhereTheRemoteHasNotMoved is the ordinary case and
// the one every other is measured against: the branch fast-forwards, and what
// the remote holds afterwards is what the clone holds.
func TestPush_SendsTheCompactionWhereTheRemoteHasNotMoved(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))

	r, held := seededStore(t, first, middle, head)
	bare := r.origin()
	r.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)

	if _, err := held.Compact(thePolicy); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	if local, remote := r.text("rev-parse", store.Ref), r.textIn(bare, "rev-parse", store.Ref); local != remote {
		t.Errorf("origin holds %s, want the clone's %s", remote, local)
	}
	want := []string{store.IntroductionPath, pathOf(first), pathOf(head)}
	slices.Sort(want)
	if got := r.published(bare); !slices.Equal(got, want) {
		t.Errorf("origin holds %v, want %v", got, want)
	}
}

// TestPush_ReAppliesARemovalOntoTheFetchedTipAndRetries is the case §7
// describes only in terms of writes. The unpushed set is a set of path
// *operations*, and this one's is a removal: another environment wrote to the
// branch between the sync and the push, the push is rejected, and what goes out
// is the removal applied to what that environment left rather than a tree that
// would take their write back off (issue #131).
func TestPush_ReAppliesARemovalOntoTheFetchedTipAndRetries(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))

	r, held := seededStore(t, first, middle, head)
	bare := r.origin()
	r.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)

	// The second environment, writing a version of another series while
	// this one is deciding what to remove.
	elsewhere := aVersion(t, theSecondSeries, theFourthRunID, 1, ageing(5))
	r.seedVersions(bare, elsewhere)

	tree := r.workingTree()

	if _, err := held.Compact(thePolicy); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// **The re-application uses no worktree, no checkout and no `git
	// rebase`.** It rebuilds a known path set onto a fetched tree with
	// plumbing, which is what lets it happen at all: `git rebase` needs a
	// worktree, and `git worktree add` would take the human's `git checkout
	// hyper-store` away (§7, ADR-0075). Nothing on disk moves, which is the
	// observable half of that sentence.
	if after := r.workingTree(); !slices.Equal(tree, after) {
		t.Errorf("the repository root holds %v, want it left as %v", after, tree)
	}
	if worktrees := r.text("worktree", "list"); strings.Count(worktrees, "\n") != 0 {
		t.Errorf("the repository has more than one worktree:\n%s", worktrees)
	}

	want := []string{store.IntroductionPath, pathOf(first), pathOf(head), pathOf(elsewhere)}
	slices.Sort(want)
	if got := r.published(bare); !slices.Equal(got, want) {
		t.Errorf("origin holds %v, want %v — the removal is re-applied onto the fetched tip, not over it", got, want)
	}
	if local, remote := r.text("rev-parse", store.Ref), r.textIn(bare, "rev-parse", store.Ref); local != remote {
		t.Errorf("origin holds %s and the clone %s; a re-applied push leaves the two agreeing", remote, local)
	}

	// The account survives the retry: the Compaction's own commit message is
	// what `git log` on the branch says about it, and a re-application that
	// collapsed the unpushed set would have lost it (§7, §13).
	if message := r.text("log", "-1", "--format=%B", store.Ref); !strings.Contains(message, thePolicy.Declared) {
		t.Errorf("the tip's message is %q, want the Compaction's own account of what it removed", message)
	}
}

// TestPush_ARemovalThePathIsAlreadyOffIsANoOp is the other half of the same
// sentence. Two environments compacting one branch remove the same version, and
// the second one to push finds its path already gone — which is one removal
// rather than a conflict.
func TestPush_ARemovalThePathIsAlreadyOffIsANoOp(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))

	r, held := seededStore(t, first, middle, head)
	bare := r.origin()
	r.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)

	// The same version taken off the branch by somebody else first, and a
	// write beside it so that the remote's tip is genuinely ahead.
	r.seedWithout(bare, pathOf(middle))
	elsewhere := aVersion(t, theSecondSeries, theFourthRunID, 1, ageing(5))
	r.seedVersions(bare, elsewhere)

	if _, err := held.Compact(thePolicy); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	want := []string{store.IntroductionPath, pathOf(first), pathOf(head), pathOf(elsewhere)}
	slices.Sort(want)
	if got := r.published(bare); !slices.Equal(got, want) {
		t.Errorf("origin holds %v, want %v — a removal whose path is already gone is a no-op", got, want)
	}
}

// TestPush_ExhaustedAfterThreeAttemptsReportsTheCondition is the remote moving
// under every attempt. Three pushes, and then the condition — which this
// package reports and never maps: `compact` renders it as the world resisting
// and a Run will render it as `failed` (§9, §12).
//
// What it leaves behind is asserted too, because it is the half §7 promises: the
// commits stand locally, re-applied onto the last tip that was fetched, and go
// out with the next push that gets through.
func TestPush_ExhaustedAfterThreeAttemptsReportsTheCondition(t *testing.T) {
	first := aVersion(t, theSeries, theEntryRunID, 1, ageing(900))
	middle := aVersion(t, theSeries, theSecondRunID, 1, ageing(800))
	head := aVersion(t, theSeries, theThirdRunID, 1, ageing(700))

	r, held := seededStore(t, first, middle, head)
	bare := r.origin()
	r.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)
	r.rejectAndAdvance(bare)

	_, err := held.Compact(thePolicy)
	if !errors.Is(err, store.ErrPushExhausted) {
		t.Fatalf("Compact: %v, want %v", err, store.ErrPushExhausted)
	}

	tree := r.storeTree(r.root)
	if _, stands := tree[pathOf(middle)]; stands {
		t.Errorf("%s stands locally; a push that could not complete does not unwind what it removed", pathOf(middle))
	}
	if _, stands := tree[pathOf(head)]; !stands {
		t.Errorf("%s is gone locally; nothing about an exhausted push touches the Head", pathOf(head))
	}
}
