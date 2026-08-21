package store_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The write half of the branch (issue #136): a Run puts files on the Store one
// commit at a time and pushes them when its own rule says to.

// TestAppend_IsOneCommitCarryingTheFilesAndWhatWasAlreadyThere is the whole of
// what one write does: the paths handed appear, everything already on the
// branch stands, and the branch moves by exactly one commit.
func TestAppend_IsOneCommitCarryingTheFilesAndWhatWasAlreadyThere(t *testing.T) {
	r, held := seededStore(t)
	before := r.text("rev-parse", store.Ref)

	if err := held.Append([]store.Write{
		{Path: "journal/2026/04/02/" + theEntryRunID + "/run.json", Content: []byte("{}\n")},
	}, "a Run began"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	tree := r.storeTree(r.root)
	if got := paths(tree); !slices.Equal(got, []string{"STORE.md", "journal/2026/04/02/" + theEntryRunID + "/run.json"}) {
		t.Errorf("the branch holds %q, want the file written beside the one already there", got)
	}
	if parents := strings.Fields(r.text("rev-list", "--parents", "-1", store.Ref)); len(parents) != 2 || parents[1] != before {
		t.Errorf("the branch is at %q; want one commit whose parent is the tip the handle was opened at (%s)", parents, before)
	}
}

// Two writes are two commits, which is §7's *one commit per confirmed write*:
// a crashed Run's local branch tip is exactly what it confirmed, so a write
// held back to be batched with the next one is a Step whose evidence a crash
// takes with it (§6, §7, ADR-0075).
func TestAppend_EachWriteIsItsOwnCommit(t *testing.T) {
	r, held := seededStore(t)
	before := len(strings.Fields(r.text("rev-list", store.Ref)))

	for _, name := range []string{"first", "second"} {
		if err := held.Append([]store.Write{{Path: "journal/" + name + ".json", Content: []byte("{}\n")}}, "wrote "+name); err != nil {
			t.Fatalf("Append %s: %v", name, err)
		}
	}

	if after := len(strings.Fields(r.text("rev-list", store.Ref))); after != before+2 {
		t.Errorf("two writes left %d commits, want the %d that were there and one each", after, before+2)
	}
}

// The handle moves with the branch, so a read taken through it after a write
// answers about the branch that now stands. A handle left pointing at the
// commit it was opened at would answer that the Run's own Records are not
// there, which is what every read-then-write sequence in a Run does.
func TestAppend_TheHandleReadsWhatItJustWrote(t *testing.T) {
	_, held := seededStore(t)
	run := runID(t, theEntryRunID)
	version := aVersion(t, theSeries, theEntryRunID, 1, theInstant)

	if err := held.Append([]store.Write{
		{Path: store.RecordPath(theSeries, run, 1), Content: version.Encode()},
	}, "a version"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	head, held2, err := held.Head(theSeries)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if !held2 {
		t.Fatalf("the handle answers no Head for the series it just wrote a version of")
	}
	if head.Run != run {
		t.Errorf("the Head was written by %s, want the version just appended (%s)", head.Run, run)
	}
}

// Append-only is the branch's central rule, so a caller writing a path the
// branch already holds is hyper's arithmetic being wrong rather than an
// overwrite performed quietly: every Store path carries the id of the Run that
// wrote it, so two Runs cannot write one path and one Run writing a path twice
// is a bug (§7, ADR-0011, ADR-0076).
func TestAppend_RefusesAPathTheBranchAlreadyHolds(t *testing.T) {
	_, held := seededStore(t)

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Errorf("a write over STORE.md was performed; no file in the Store is ever rewritten (§7)")
		}
	}()
	_ = held.Append([]store.Write{{Path: store.IntroductionPath, Content: []byte("rewritten\n")}}, "a rewrite")
}

// A write of nothing writes no commit at all. It is not an error and not a
// no-op worth a commit: a Step that concluded about nothing new leaves the
// branch byte-identical, which is what makes *a version is written only where
// the bytes moved* visible in `git log` rather than only in the tree (§7).
func TestAppend_WritingNothingLeavesTheBranchWhereItIs(t *testing.T) {
	r, held := seededStore(t)
	before := r.text("rev-parse", store.Ref)

	if err := held.Append(nil, "nothing at all"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if after := r.text("rev-parse", store.Ref); after != before {
		t.Errorf("the branch moved from %s to %s over a write of no files", before, after)
	}
}

// The commit message is the caller's and reaches `git log` intact, which is
// what makes the branch hyper's own account of what happened (§7, §13).
func TestAppend_TheMessageIsTheCallersAndReachesTheLog(t *testing.T) {
	r, held := seededStore(t)

	if err := held.Append([]store.Write{{Path: "journal/one.json", Content: []byte("{}\n")}}, "Begin run "+theEntryRunID); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if got, want := r.text("log", "-1", "--format=%B", store.Ref), "Begin run "+theEntryRunID; got != want {
		t.Errorf("git log says %q, want %q", got, want)
	}
}

// Publish is the push a read-only Run batches to its end: what stands locally
// goes out, and nothing is written by the sending (§6, §7, ADR-0006).
func TestPublish_SendsWhatWasWrittenToTheRemote(t *testing.T) {
	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	origin := r.origin()
	r.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)

	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := held.Append([]store.Write{{Path: "journal/one.json", Content: []byte("{}\n")}}, "a Run"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := held.Publish(); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	remote := r.storeTree(origin)
	if !slices.Equal(paths(remote), []string{"STORE.md", "journal/one.json"}) {
		t.Errorf("origin holds %q, want what the Run wrote", slices.Sorted(maps.Keys(remote)))
	}
}
