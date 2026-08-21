package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// What the corpus cannot say about `hyper compact` (issue #131).
//
// Its cases hold the two streams, the exit code and the branch's tree, which is
// everything the command answers about a world that behaves. What is here is
// the world resisting in ways no checked-in fixture can arrange — a push that
// reaches nowhere, a remote that will not answer — and the rule about what the
// command does *not* leave behind, which a case directory cannot state either.
//
// Each is driven through cli.Main from a complete argv, like every golden case,
// and each builds on store_test.go's storeCase: a repository that can go on
// being edited after it exists.

// declare rewrites the case repository's hyper.yaml with the retention policy
// handed, so a case can state the policy it is acting under.
func (c *storeCase) declare(retention string) {
	c.t.Helper()

	declaration := "kind: repository-declaration\nversion: 1.4.0\n"
	if retention != "" {
		declaration += "retention: " + retention + "\n"
	}
	writeFile(c.t, filepath.Join(c.fx.root, "hyper.yaml"), declaration)
}

// seed puts a Store branch on the repository holding the versions handed, at
// the paths and in the encoding the grammar and the canonical encoding give
// them — a branch `hyper` could have written, which is the only kind these
// cases are about.
//
// It goes through the golden fixture's own orphan builder, so a branch a case
// here seeds and a branch a case directory seeds are built by one code path and
// neither is checked out (ADR-0075).
func (c *storeCase) seed(versions ...store.RecordVersion) {
	c.t.Helper()

	dir := c.t.TempDir()
	writeFile(c.t, filepath.Join(dir, store.IntroductionPath), store.Introduction)
	for _, version := range versions {
		path := store.RecordPath(version.Identity, version.Run, version.Step)
		writeFile(c.t, filepath.Join(dir, filepath.FromSlash(path)), string(version.Encode()))
	}
	c.git("update-ref", store.Ref, c.fx.orphan(c.t, dir))
}

// compact drives `hyper compact` against the repository, through the one entry
// point and from a complete argv — the same path a golden case takes, including
// the --repo-dir the harness splices in.
func (c *storeCase) compact(args ...string) (exit int, stdout, stderr string) {
	c.t.Helper()

	var out, errs bytes.Buffer
	argv := append([]string{"compact", "--repo-dir", c.fx.root}, args...)
	code := cli.Main(argv, &out, &errs, (&process{wd: c.fx.root}).value(), testFacts)
	return code, out.String(), errs.String()
}

// aged is one version of the case series, that many days before the clock every
// case here is driven at.
func aged(t *testing.T, run string, days int) store.RecordVersion {
	t.Helper()

	id, err := store.ParseRunID(run)
	if err != nil {
		t.Fatal(err)
	}
	instant := fixedInstant.AddDate(0, 0, -days)
	return store.RecordVersion{
		Metadata: store.Metadata{
			Identity:   store.Identity{Target: "local", Definition: "uptime", Name: "status.hyper.dev"},
			RecordType: store.RecordObservation,
			Run:        id,
			Step:       1,
			Operation:  "get_status",
			WrittenAt:  instant,
			Provenance: caseProvenance,
		},
		Fields: store.Mapping{"observed_at": store.String(store.InstantText(instant))},
	}
}

// caseProvenance is the Provenance every seeded version carries. Every member
// a Record version requires has a value and none is read back by anything here:
// what these cases are about is the branch, and a version missing its
// Provenance is a version the encoder declines to write at all (§7, ADR-0043).
var caseProvenance = store.Provenance{
	Run: store.RunProvenance{
		HyperVersion:      "1.4.0",
		ProcedureRevision: "b7f1c0a2d3e4f5061728394a5b6c7d8e9f001122",
		RepoRevision:      "3f2e1d0c9b8a77665544332211009988ffeeddcc",
	},
	Step: store.StepProvenance{
		DefinitionRevision: "9a8b7c6d5e4f30211203948576a7b8c9d0e1f223",
		ManifestDigest:     "sha256:a118a517431e241eac83559919ae969346bf5a3bf6e06c6db3e636f378fcdf12",
	},
}

// The Run ids these cases seed with, and the whole of what they are: three
// UUIDv7s, so that a series has an interior at all.
const (
	firstRun  = "01984f1a-3c9f-7b04-9c2e-4f0b8d61a3e7"
	middleRun = "01984f2b-4da0-7c15-8d41-6b2f7ae05c19"
	headRun   = "01984f3c-5eb1-7d26-9e52-7c3f8bf16d2a"
)

// TestRunCompact_APushThatCannotCompleteIsTheWorldResisting is the exit code no
// golden case can arrange: the remote answers when it is asked what it holds,
// and refuses the push that follows.
//
// It is `1`, and the two codes it is not are the point (§9, §12, ADR-0061). It
// is not `75`, which is a *Run* that lost the Store and `compact` is not a Run.
// And it is not `77`, which promises that a verbatim retry Refuses identically
// — false the moment the remote comes back.
func TestRunCompact_APushThatCannotCompleteIsTheWorldResisting(t *testing.T) {
	c := newStoreCase(t)
	c.declare("90d")
	c.seed(aged(t, firstRun, 900), aged(t, middleRun, 800), aged(t, headRun, 700))
	c.origin()
	c.git("push", "--quiet", "origin", store.Ref+":"+store.Ref)
	c.git("config", "remote.origin.pushurl", filepath.Join(t.TempDir(), "no-repository-here.git"))

	exit, stdout, stderr := c.compact()

	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — the world resisted", exit, cli.ExitProblems)
	}
	for _, wrong := range []int{cli.ExitStoreLost, cli.ExitRefused} {
		if exit == wrong {
			t.Errorf("exit = %d; a push that could not complete is neither a Run that lost the Store nor a guardrail declining", exit)
		}
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want silence: the command has no answer to write", stdout)
	}
	if stderr == "" {
		t.Error("stderr is empty, want the reason the push could not complete")
	}

	// What it removed before it stopped stands locally, and goes out with
	// the next push that gets through (§7).
	if listed := c.git("ls-tree", "-r", "--name-only", store.Ref); strings.Contains(listed, middleRun) {
		t.Errorf("the branch still holds the interior version; a push that failed does not unwind what it removed")
	}
}

// TestRunCompact_AStoreAbsentWithTheRemoteUnreachableIsNotStoreAbsent is the
// distinction `store-absent` rests on. The branch is in neither place this
// clone can see, but one of the two places could not be asked — so claiming
// `store-absent` would tell a caller that a verbatim retry Refuses identically,
// which is false the moment the network returns (§7, §12, ADR-0061).
func TestRunCompact_AStoreAbsentWithTheRemoteUnreachableIsNotStoreAbsent(t *testing.T) {
	c := newStoreCase(t)
	c.declare("90d")
	c.origin()
	c.git("config", "remote.origin.url", filepath.Join(t.TempDir(), "no-repository-here.git"))

	exit, stdout, stderr := c.compact()

	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — the remote could not be asked", exit, cli.ExitProblems)
	}
	if strings.Contains(stderr, "store-absent") {
		t.Errorf("stderr = %q, want it not to claim store-absent: one of the two places was never asked", stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want silence", stdout)
	}
}

// TestRunCompact_RunsAgainstADirtyTreeAndLeavesItExactlyAsItWas is ADR-0075 as
// the property an operator notices, on the one command that removes. The
// removal is a commit built from git objects and nothing about the branch is
// ever checked out, so an uncommitted edit and an untracked file are both still
// there afterwards, unchanged and unstaged.
//
// No golden can hold this: a case's repository is committed whole before the
// command runs and is never touched again, and a command that quietly staged,
// stashed or checked something out would answer on all three streams exactly as
// it does here.
func TestRunCompact_RunsAgainstADirtyTreeAndLeavesItExactlyAsItWas(t *testing.T) {
	c := newStoreCase(t)
	c.declare("90d")
	c.seed(aged(t, firstRun, 900), aged(t, middleRun, 800), aged(t, headRun, 700))
	writeFile(t, filepath.Join(c.root(), "untracked.txt"), "a file nobody committed\n")

	before := c.git("status", "--porcelain")
	head := c.git("rev-parse", "HEAD")

	exit, stdout, stderr := c.compact()
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, stderr)
	}
	if !strings.Contains(stdout, "removed 1 interior observation version") {
		t.Errorf("stdout = %q, want it to name the one version it removed", stdout)
	}

	if after := c.git("status", "--porcelain"); after != before {
		t.Errorf("the working tree moved:\n before: %q\n after:  %q", before, after)
	}
	if now := c.git("rev-parse", "HEAD"); now != head {
		t.Errorf("HEAD moved from %s to %s; the Store is never checked out", head, now)
	}
	if branch := c.git("rev-parse", "--abbrev-ref", "HEAD"); branch != codeBranchName {
		t.Errorf("HEAD is on %q, want it left on %s", branch, codeBranchName)
	}
	if _, err := os.Stat(filepath.Join(c.root(), store.IntroductionPath)); !os.IsNotExist(err) {
		t.Errorf("%s is in the working tree; no byte of Store content is ever an ordinary file on disk", store.IntroductionPath)
	}
	if worktrees := strings.Count(c.git("worktree", "list"), "\n") + 1; worktrees != 1 {
		t.Errorf("the repository has %d worktrees, want the one it started with", worktrees)
	}
}
