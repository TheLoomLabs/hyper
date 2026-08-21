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

// What the corpus cannot say about `hyper store init` (issue #126).
//
// Its goldens hold the two streams, the exit code and the branch's tree, which
// is everything the command answers — but two of the ticket's rules are about
// what the command does *not* leave behind, and one is about a world that
// resists in a way no checked-in fixture can arrange. Those three are here, each
// driven through cli.Main from a complete argv like every golden case is.

// storeCase is one repository built for a case in this file: a git repository
// with a working tree, a pin the gate will pass, and whatever remote the case
// wires. It is deliberately not a golden case — that one is driven from a case
// directory and committed whole before the command runs, and what these cases
// need is a repository they can go on editing after it exists.
//
// It runs its own git through gitFixture rather than through a wrapper of its
// own, which is what keeps the environment stated rather than inherited: git's
// global and system configuration are pointed at files that do not exist and the
// identity is supplied outright, so a machine that signs every commit or has
// never set user.email builds the same repository here.
type storeCase struct {
	t  *testing.T
	fx gitFixture
}

func newStoreCase(t *testing.T) *storeCase {
	t.Helper()
	requireGit(t)

	base := t.TempDir()
	home := filepath.Join(base, "home")
	root := filepath.Join(base, "repo")
	for _, dir := range []string{home, root} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	c := &storeCase{t: t, fx: gitFixture{root: root, env: fixtureEnvironment(home, fixedInstant)}}
	writeFile(t, filepath.Join(root, "hyper.yaml"), "kind: repository-declaration\nversion: 1.4.0\n")
	c.git("init", "--quiet", "--initial-branch="+codeBranchName)
	c.git("add", "--all")
	c.git("commit", "--quiet", "--message", "the working tree")
	return c
}

// root is where the repository sits, which is what --repo-dir names and what
// every assertion resolves a path against.
func (c *storeCase) root() string { return c.fx.root }

// git runs one git command in the repository and answers its stdout as a line.
// A failure is the case's own setup rather than anything under test, so it stops
// the test where it is.
func (c *storeCase) git(args ...string) string {
	c.t.Helper()
	return c.fx.text(c.t, c.fx.root, args...)
}

// origin wires a bare repository as the remote of that name, with the code
// branch pushed to it, and answers where it sits.
func (c *storeCase) origin() string {
	c.t.Helper()

	bare := filepath.Join(filepath.Dir(c.fx.root), "origin.git")
	c.fx.run(c.t, filepath.Dir(c.fx.root), "init", "--quiet", "--bare", bare)
	c.git("remote", "add", "origin", bare)
	c.git("push", "--quiet", "origin", codeBranch+":"+codeBranch)
	return bare
}

// init drives `hyper store init` against the repository, through the one entry
// point and from a complete argv — the same path a golden case takes, including
// the --repo-dir the harness splices in.
func (c *storeCase) init(args ...string) (exit int, stdout, stderr string) {
	c.t.Helper()

	var out, errs bytes.Buffer
	argv := append([]string{"store", "--repo-dir", c.fx.root, "init"}, args...)
	code := cli.Main(argv, &out, &errs, (&process{wd: c.fx.root}).value(), testFacts)
	return code, out.String(), errs.String()
}

// TestRunStore_RunsAgainstADirtyTreeAndLeavesItExactlyAsItWas is ADR-0075 as
// the property an operator notices. The branch is a parentless commit built
// from git objects and nothing about it is ever checked out, so `store init`
// runs against a dirty tree like any read command — and an uncommitted edit and
// an untracked file are both still there afterwards, unchanged and unstaged.
//
// No golden can hold this: a case's repository is committed whole before the
// command runs and is never touched again, and a command that quietly staged,
// stashed or checked something out would answer on all three streams exactly as
// it does here. It is driven through cli.Main rather than through store.Init
// because the criterion is the command's — *it runs against a dirty tree like
// any read command* — and the gate, the root resolution and the exit code are
// all part of that claim.
func TestRunStore_RunsAgainstADirtyTreeAndLeavesItExactlyAsItWas(t *testing.T) {
	c := newStoreCase(t)
	writeFile(t, filepath.Join(c.root(), "hyper.yaml"), "kind: repository-declaration\nversion: 1.4.0\n# edited, and uncommitted\n")
	writeFile(t, filepath.Join(c.root(), "untracked.txt"), "a file nobody committed\n")

	before := c.git("status", "--porcelain")
	head := c.git("rev-parse", "HEAD")

	exit, stdout, stderr := c.init()
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, stderr)
	}
	if !strings.Contains(stdout, store.BranchName) {
		t.Errorf("stdout = %q, want it to name the branch it created", stdout)
	}

	if after := c.git("status", "--porcelain"); after != before {
		t.Errorf("the working tree moved:\n before: %q\n after:  %q", before, after)
	}
	if now := c.git("rev-parse", "HEAD"); now != head {
		t.Errorf("HEAD moved from %s to %s; the Store is never checked out", head, now)
	}
	if branch := c.git("rev-parse", "--abbrev-ref", "HEAD"); branch != "main" {
		t.Errorf("HEAD is on %q, want it left on main", branch)
	}
	if _, err := os.Stat(filepath.Join(c.root(), store.IntroductionPath)); !os.IsNotExist(err) {
		t.Errorf("%s is in the working tree; no byte of Store content is ever an ordinary file on disk", store.IntroductionPath)
	}
	if worktrees := strings.Count(c.git("worktree", "list"), "\n") + 1; worktrees != 1 {
		t.Errorf("the repository has %d worktrees, want the one it started with — `git worktree add` would take the human's checkout away (§7)", worktrees)
	}
}

// TestRunStore_TheLocalRefIsOneAHumanCanCheckOut is §7's promise to a reader,
// and the reason the ref is refs/heads/hyper-store and not a private namespace:
// the record being in the open is the thesis, and the branch a human already has
// is the one they read it on.
func TestRunStore_TheLocalRefIsOneAHumanCanCheckOut(t *testing.T) {
	c := newStoreCase(t)
	if exit, _, stderr := c.init(); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want 0; stderr=%q", exit, stderr)
	}

	c.git("checkout", "--quiet", store.BranchName)

	read, err := os.ReadFile(filepath.Join(c.root(), store.IntroductionPath))
	if err != nil {
		t.Fatalf("%s is not readable after the checkout: %v", store.IntroductionPath, err)
	}
	if string(read) != store.Introduction {
		t.Errorf("the checked-out %s is %q, want the stated %q", store.IntroductionPath, read, store.Introduction)
	}
	if listed := c.git("ls-tree", "-r", "--name-only", store.Ref); listed != store.IntroductionPath {
		t.Errorf("the branch holds %q, want %s and nothing else", listed, store.IntroductionPath)
	}
}

// TestRunStore_APushThatCannotCompleteIsTheWorldResisting is the one exit code
// no golden case can arrange: a remote that answers when it is asked what it
// holds and refuses the push that follows.
//
// It is `1`, and the two codes it is not are the point (§9, §12, ADR-0061). It
// is not `75`, which is a *Run* that lost the Store and `store init` is not a
// Run. And it is not `77`, which promises that a verbatim retry Refuses
// identically — false the moment the remote comes back, which is a network
// returning and not an act of anybody's.
func TestRunStore_APushThatCannotCompleteIsTheWorldResisting(t *testing.T) {
	c := newStoreCase(t)

	// A reachable remote for the look and an unreachable one for the push,
	// which is what separates this case from a remote that was never there:
	// the branch is not on origin, so the command mints a root and then
	// cannot send it.
	bare := c.origin()
	c.git("config", "remote.origin.pushurl", filepath.Join(t.TempDir(), "no-repository-here.git"))

	exit, stdout, stderr := c.init()
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

	// What it wrote before it stopped stands locally (§7).
	if c.git("rev-parse", "--verify", store.Ref) == "" {
		t.Fatalf("%s is not there; a push that failed does not unwind the branch it was sending", store.Ref)
	}

	// And there is a way back from here, which is the reason `init` still
	// looks at the remote on finding the branch underfoot: with the push
	// reaching somewhere again, a second `init` creates nothing and sends the
	// branch that was stranded. A no-op on the local ref would leave this
	// repository refusing every scheduled Run forever, with no command in the
	// tree able to repair it (§7, issue #126).
	c.git("config", "--unset", "remote.origin.pushurl")

	exit, stdout, stderr = c.init()
	if exit != cli.ExitClean {
		t.Fatalf("the second init exits %d, want 0; stderr=%q", exit, stderr)
	}
	if stdout != "pushed  origin\n" {
		t.Errorf("stdout = %q, want the push alone: nothing was created the second time", stdout)
	}
	if local, remote := c.git("rev-parse", store.Ref), c.fx.text(t, bare, "rev-parse", store.Ref); local != remote {
		t.Errorf("origin holds %s at %s, want the clone's %s", store.Ref, remote, local)
	}
}
