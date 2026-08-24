package revision_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/revision"
)

// What git says moved between two revisions (§8, §12, issue #171).
//
// Every case is a fact about a real `git diff`, for the reason the rest of this
// suite's are: the Comparison's catch-all row **names the command** — `N other
// lines changed · git diff <rev> <rev>` — so a count that disagreed with what
// the reader runs would be a row disagreeing with its own evidence.

// TestBetween_CountsAddedAndRemovedAsGitCountsThem is the unit: added and
// removed lines, **a modified line being two**.
func TestBetween_CountsAddedAndRemovedAsGitCountsThem(t *testing.T) {
	r := newRepo(t)
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\nbound: 3\n")
	r.commit()
	before := r.text("rev-parse", "HEAD")

	// One line modified — two — and one added.
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\nbound: 5\ncadence: \"0 3 * * 1\"\n")
	r.commit()
	after := r.text("rev-parse", "HEAD")

	moved, held, err := revision.Between(r.root, before, after, wanted)
	if err != nil {
		t.Fatalf("Between: %v", err)
	}
	if !held {
		t.Fatal("the two commits the case just made do not resolve")
	}
	if moved.Count != 3 {
		t.Errorf("git counts %d moved lines, want 3: one modified line is two and one added line is one", moved.Count)
	}
	if !moved.Before["procedures/publish.yaml"][3] {
		t.Errorf("line 3 of the earlier revision is not marked; the lines are %v", moved.Before)
	}
	if !moved.After["procedures/publish.yaml"][3] || !moved.After["procedures/publish.yaml"][4] {
		t.Errorf("lines 3 and 4 of the later revision are not both marked; the lines are %v", moved.After)
	}
}

// TestBetween_CountsTheWantedPathsAndNothingElse is the file set: the reviewed
// five and nothing else, which is §7's `repo_dirty` file set arriving from the
// other side — "so the marker and the count agree on what code is by
// construction" (§8, §12).
//
// The generated workflow is out particularly: it is projected rather than
// authored and byte-exact against what `project` would write (ADR-0046), so a
// change in it is a `hyper` version move already in Provenance, a Procedure
// move already classed, or a hand-edit — and a hand-edit is `check`'s Refusal
// rather than a row here.
func TestBetween_CountsTheWantedPathsAndNothingElse(t *testing.T) {
	r := newRepo(t)
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\n")
	r.write(".github/workflows/hyper.yml", "name: hyper\n")
	r.write("README.md", "# a repository\n")
	r.commit()
	before := r.text("rev-parse", "HEAD")

	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\nbound: 5\n")
	r.write(".github/workflows/hyper.yml", "name: hyper\non: {schedule: []}\n")
	r.write("README.md", "# a repository\n\nand a paragraph\n")
	r.commit()
	after := r.text("rev-parse", "HEAD")

	moved, held, err := revision.Between(r.root, before, after, wanted)
	if err != nil || !held {
		t.Fatalf("Between: %v (held %v)", err, held)
	}
	if moved.Count != 1 {
		t.Errorf("git counts %d moved lines, want the one added to the Procedure: the workflow and the README are outside the reviewed five", moved.Count)
	}
	for _, outside := range []string{".github/workflows/hyper.yml", "README.md"} {
		if len(moved.After[outside]) != 0 {
			t.Errorf("%s is marked, and it is not an artefact", outside)
		}
	}
}

// TestBetween_ADeletionAndACreationAreCountedWhereGitCountsThem is the two
// one-sided files: `--- /dev/null` and `+++ /dev/null` each name the path on
// the other line, and renames are off so a moved file is both.
func TestBetween_ADeletionAndACreationAreCountedWhereGitCountsThem(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/gone.yaml", "kind: definition\ndefinition: gone\n")
	r.commit()
	before := r.text("rev-parse", "HEAD")

	r.remove("definitions/gone.yaml")
	r.write("procedures/new.yaml", "kind: procedure\nprocedure: new\nbound: 1\n")
	r.commit()
	after := r.text("rev-parse", "HEAD")

	moved, held, err := revision.Between(r.root, before, after, wanted)
	if err != nil || !held {
		t.Fatalf("Between: %v (held %v)", err, held)
	}
	if moved.Count != 5 {
		t.Errorf("git counts %d moved lines, want 5: two removed and three added", moved.Count)
	}
	if !moved.Before["definitions/gone.yaml"][1] || !moved.Before["definitions/gone.yaml"][2] {
		t.Errorf("the deleted file's lines are not marked at the earlier revision; the lines are %v", moved.Before)
	}
	if len(moved.After["procedures/new.yaml"]) != 3 {
		t.Errorf("the created file's three lines are not marked at the later revision; the lines are %v", moved.After)
	}
}

// TestBetween_ARevisionTheCloneDoesNotHoldIsAnsweredAndNeverErrored is this
// package's standing rule: a Run recorded on a runner names a commit a laptop
// may never have fetched, and *not held* is an ordinary fact about the clone
// rather than the world resisting (ADR-0071).
func TestBetween_ARevisionTheCloneDoesNotHoldIsAnsweredAndNeverErrored(t *testing.T) {
	r := newRepo(t)
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\n")
	r.commit()
	head := r.text("rev-parse", "HEAD")

	const absent = "1f0a3d78c2e5b91467af03d28b5c9e610473fa8d"
	for _, pair := range [][2]string{{absent, head}, {head, absent}} {
		moved, held, err := revision.Between(r.root, pair[0], pair[1], wanted)
		if err != nil {
			t.Fatalf("Between(%s, %s): %v", pair[0], pair[1], err)
		}
		if held {
			t.Errorf("Between(%s, %s) says the clone holds both, and one of them was never made", pair[0], pair[1])
		}
		if moved.Count != 0 {
			t.Errorf("a revision the clone does not hold counted %d lines", moved.Count)
		}
	}
}

// TestBetween_TwoNamesForOneCommitMoveNothing is the ordinary zero: a window
// whose two ends read one revision counts nothing, which is what `0 other lines
// changed` renders (§8).
func TestBetween_TwoNamesForOneCommitMoveNothing(t *testing.T) {
	r := newRepo(t)
	r.write("procedures/publish.yaml", "kind: procedure\nprocedure: publish\n")
	r.commit()
	head := r.text("rev-parse", "HEAD")

	moved, held, err := revision.Between(r.root, head, head, wanted)
	if err != nil || !held {
		t.Fatalf("Between: %v (held %v)", err, held)
	}
	if moved.Count != 0 {
		t.Errorf("one commit against itself counted %d lines, want none", moved.Count)
	}
}
