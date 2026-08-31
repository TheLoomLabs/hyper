package revision_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/git"
	"github.com/TheLoomLabs/hyper/internal/revision"
)

// The code branch's objects as a reader that never fetches sees them (§8,
// ADR-0071, issue #164).

// TestHeld_AnsweredForAnObjectTheCloneHolds is the ordinary answer, in both
// forms of the name a review asks about: the blob id a Run recorded, and the
// file one commit held at a path.
func TestHeld_AnsweredForAnObjectTheCloneHolds(t *testing.T) {
	r := newRepo(t)
	r.write("targets/staging.yaml", "kind: target-declaration\ntarget: staging\n")
	r.commit()
	head := r.text("rev-parse", "HEAD")
	blob := r.text("rev-parse", "HEAD:targets/staging.yaml")

	for _, named := range []string{blob, revision.AtPath(head, "targets/staging.yaml")} {
		got, held, err := revision.Held(r.root, named)
		if err != nil {
			t.Fatalf("Held(%s): %v", named, err)
		}
		if !held {
			t.Errorf("Held(%s) answered not held; the clone just wrote it", named)
		}
		if got.Blob != blob {
			t.Errorf("Held(%s) = %s, want the file's blob %s", named, got.Blob, blob)
		}
		if want := "kind: target-declaration\ntarget: staging\n"; string(got.Bytes) != want {
			t.Errorf("Held(%s) read %q, want the bytes that blob holds %q", named, got.Bytes, want)
		}
	}
}

// TestHeld_AnObjectTheCloneDoesNotHoldIsAnsweredAndNeverErrored is ADR-0071's
// own rule: a Run recorded on a runner names an object a shallow clone was
// never given, so *not held* is a fact about the clone rather than the world
// resisting — and what the caller does with it is render `not-in-clone`.
func TestHeld_AnObjectTheCloneDoesNotHoldIsAnsweredAndNeverErrored(t *testing.T) {
	r := newRepo(t)
	head := r.text("rev-parse", "HEAD")

	absent := []string{
		"1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01",
		revision.AtPath(head, "targets/no-such-file.yaml"),
		revision.AtPath("2b4c6e8a01f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2", "hyper.yaml"),
	}
	for _, named := range absent {
		got, held, err := revision.Held(r.root, named)
		if err != nil {
			t.Errorf("Held(%s) = %v; an object the clone does not hold is answered, never errored", named, err)
		}
		if held || got.Blob != "" || got.Bytes != nil {
			t.Errorf("Held(%s) = %+v, %t; want the absence answered", named, got, held)
		}
	}
}

// TestHeld_AnObjectThatIsNotABlobIsNotHeld holds the reader to what its caller
// asked for. A range opens at one file's bytes, and a commit standing where a
// blob was named is an object those cannot be read from — which is the reading
// #56 refused arriving as a supply rather than as a rendering.
func TestHeld_AnObjectThatIsNotABlobIsNotHeld(t *testing.T) {
	r := newRepo(t)

	got, held, err := revision.Held(r.root, r.text("rev-parse", "HEAD"))
	if err != nil {
		t.Fatalf("Held: %v", err)
	}
	if held || got.Blob != "" || got.Bytes != nil {
		t.Errorf("Held(a commit) = %+v, %t; want a commit refused as a blob", got, held)
	}
}

// TestHeld_APathHoldingAnySpaceOrNewlineIsAnsweredWhole is why the name goes to
// git NUL-delimited: a repository may hold a path with a space or a newline in
// it, and a reader that split git's answer on either would resolve one file and
// report another.
func TestHeld_APathHoldingAnySpaceOrNewlineIsAnsweredWhole(t *testing.T) {
	r := newRepo(t)
	for _, name := range []string{"targets/two words.yaml", "targets/a\nnewline.yaml"} {
		r.write(name, "kind: target-declaration\ntarget: odd\n")
	}
	r.commit()
	head := r.text("rev-parse", "HEAD")

	for _, name := range []string{"targets/two words.yaml", "targets/a\nnewline.yaml"} {
		got, held, err := revision.Held(r.root, revision.AtPath(head, name))
		if err != nil {
			t.Fatalf("Held(%q): %v", name, err)
		}
		if !held || got.Blob == "" {
			t.Errorf("Held(%q) = %+v, %t; want the file the commit holds at that path", name, got, held)
		}
	}
}

// TestHeld_NoNameReadsNothing: the empty object name is the absence of one
// rather than a name to resolve, and it costs no subprocess to say so.
func TestHeld_NoNameReadsNothing(t *testing.T) {
	got, held, err := revision.Held(t.TempDir(), "")
	if err != nil || held || got.Blob != "" {
		t.Errorf("Held(\"\") = %+v, %t, %v; want the absence of a name answered without a repository", got, held, err)
	}
}

// TestEveryReadRunsWithLazyFetchingOff is §8's *a review reaches no network*
// enforced rather than assumed, held at the call site (ADR-0071).
//
// It is asserted against the environment every subprocess this package starts
// is run with, because that is where the rule lives: a golden could only show
// what a read answered, and what is at stake is what the read *did* on a clone
// no case can build.
func TestEveryReadRunsWithLazyFetchingOff(t *testing.T) {
	// An ambient value, because that is the case the enforcement has to
	// survive: a machine configured its way past the switch is where a
	// review would silently start reaching the network.
	t.Setenv("GIT_NO_LAZY_FETCH", "0")

	env := revision.Environment()
	if !slices.Contains(env, git.NoLazyFetch) {
		t.Errorf("the environment is %v, want lazy fetching off on every read", env)
	}
	// os/exec keeps the final value of a repeated name, so what git reads
	// is the last entry naming it — which is the rule internal/store's own
	// identity and dates already rest on.
	last := ""
	for _, entry := range env {
		if name, _, named := strings.Cut(entry, "="); named && name == "GIT_NO_LAZY_FETCH" {
			last = entry
		}
	}
	if last != git.NoLazyFetch {
		t.Errorf("git would read %q for the switch, want %q", last, git.NoLazyFetch)
	}
	if os.Getenv("GIT_NO_LAZY_FETCH") != "0" {
		t.Error("building the environment moved the process's own variable")
	}
}

// TestCommitted_TheBlobACommitHoldsAtAPath is the ordinary answer: a file that
// was committed, read back as the id its commit names for it, without the blob
// itself being read at all.
func TestCommitted_TheBlobACommitHoldsAtAPath(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/dns.yaml", "kind: definition\ndefinition: dns\n")
	r.commit()
	head := r.text("rev-parse", "HEAD")

	got, err := revision.Committed(r.root, head, "definitions/dns.yaml")
	if err != nil {
		t.Fatalf("Committed: %v", err)
	}
	if want := r.text("rev-parse", "HEAD:definitions/dns.yaml"); got != want {
		t.Errorf("Committed = %s, want the id the commit names at that path %s", got, want)
	}
}

// TestCommitted_APathTheCommitDoesNotCarryIsAnsweredEmpty is the answer that
// makes `never committed` readable: the Run recorded a revision, the commit it
// named beside it holds nothing at that path, and the two together say the
// bytes were read out of a working tree and written into no commit (§8).
func TestCommitted_APathTheCommitDoesNotCarryIsAnsweredEmpty(t *testing.T) {
	r := newRepo(t)
	head := r.text("rev-parse", "HEAD")

	got, err := revision.Committed(r.root, head, "definitions/untracked.yaml")
	if err != nil {
		t.Errorf("Committed = %v; a path the commit does not carry is answered, never errored", err)
	}
	if got != "" {
		t.Errorf("Committed = %s, want the absence answered empty", got)
	}
}

// TestCommitted_ACommitThisCloneDoesNotHoldIsAnError, and that is the whole of
// how the two sentences stay apart: a commit that cannot be read says nothing
// about whether the file under it was committed, so the caller falls back to
// the sentence about the clone rather than claiming the stronger one (§8).
func TestCommitted_ACommitThisCloneDoesNotHoldIsAnError(t *testing.T) {
	r := newRepo(t)

	if _, err := revision.Committed(r.root, "2b4c6e8a01f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2", "hyper.yaml"); err == nil {
		t.Error("Committed answered for a commit this clone does not hold; want the read reported as one that could not be performed")
	}
}

// TestCommitted_APathHoldingAnySpaceOrNewlineIsAnsweredWhole is Held's own rule
// one reader over: the listing is asked for NUL-separated, so a path git would
// otherwise quote arrives whole.
func TestCommitted_APathHoldingAnySpaceOrNewlineIsAnsweredWhole(t *testing.T) {
	r := newRepo(t)
	for _, name := range []string{"targets/two words.yaml", "targets/a\nnewline.yaml"} {
		r.write(name, "kind: target-declaration\ntarget: odd\n")
	}
	r.commit()
	head := r.text("rev-parse", "HEAD")

	for _, name := range []string{"targets/two words.yaml", "targets/a\nnewline.yaml"} {
		got, err := revision.Committed(r.root, head, name)
		if err != nil {
			t.Fatalf("Committed(%q): %v", name, err)
		}
		if got == "" {
			t.Errorf("Committed(%q) answered empty; want the file the commit holds at that path", name)
		}
	}
}

// TestCommitted_ADirectoryIsNotAFileTheCommitHolds: a tree standing where a
// file was named is not the artefact's revision, and answering its id would
// hand the caller an object that is not a file's bytes at all.
func TestCommitted_ADirectoryIsNotAFileTheCommitHolds(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/dns.yaml", "kind: definition\ndefinition: dns\n")
	r.commit()

	got, err := revision.Committed(r.root, r.text("rev-parse", "HEAD"), "definitions")
	if err != nil {
		t.Fatalf("Committed: %v", err)
	}
	if got != "" {
		t.Errorf("Committed(a directory) = %s, want a tree refused as a file", got)
	}
}
