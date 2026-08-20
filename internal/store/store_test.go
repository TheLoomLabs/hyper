package store_test

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Store is a git branch, so every assertion here is a fact about a real
// repository built in a temp directory and read back with git. Nothing is
// mocked: §7 writes down bytes on a branch, and the test of a rule that writes
// bytes is the bytes it wrote (issue #124, issue #126).
//
// The repository a case builds is its own, under t.TempDir(), and no test in
// this package ever names hyper's own root — `store.Init` creates a branch, and
// a fixture that resolved the repository it is being developed in would create
// one there.

// theInstant is the clock every case here is driven at unless it names another:
// one constant, so a commit's dates are the same on two machines. It is the
// instant §7's own worked examples are written at, and the harness in
// internal/cli drives its fixtures at the same one.
var theInstant = time.Date(2026, time.April, 2, 9, 41, 14, 221_000_000, time.UTC)

// repo is one git repository a case built: where it sits, and the environment
// its own git calls are made with. The environment is stated rather than
// inherited so that a machine whose git is configured to sign every commit, or
// which has never set user.email at all, builds the same bytes here — the same
// property internal/cli's golden fixture holds, for the same reason.
//
// What it deliberately does not control is the environment `store.Init` itself
// uses: that one is the process's own, because the git `hyper` shells out to is
// the git that resolves the credential a checkout left behind (§7).
type repo struct {
	t    *testing.T
	root string
	env  []string
}

// newRepo builds an empty git repository with one ordinary commit on `main`,
// which is every case's starting point: a code branch to stand beside, and no
// Store.
func newRepo(t *testing.T) *repo {
	t.Helper()
	requireGit(t)

	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	r := &repo{t: t, root: root, env: []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "absent-global-config"),
		"GIT_CONFIG_SYSTEM=" + filepath.Join(home, "absent-system-config"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=somebody else",
		"GIT_AUTHOR_EMAIL=somebody@elsewhere.invalid",
		"GIT_COMMITTER_NAME=somebody else",
		"GIT_COMMITTER_EMAIL=somebody@elsewhere.invalid",
		"GIT_AUTHOR_DATE=2001-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2001-01-01T00:00:00Z",
		"GIT_TERMINAL_PROMPT=0",
		"TZ=UTC",
		"LC_ALL=C",
	}}

	r.git("init", "--quiet", "--initial-branch=main")
	r.write("hyper.yaml", "kind: repository-declaration\nversion: 1.4.0\n")
	r.git("add", "--all")
	r.git("commit", "--quiet", "--message", "the working tree")
	return r
}

// write puts one file in the working tree, which is the half of the repository
// `store.Init` must never touch.
func (r *repo) write(name, content string) {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.root, name), []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// git runs one git command in the repository and returns its stdout untouched.
// A failure is the fixture's own rather than a case's, so it stops the test
// where it happened.
func (r *repo) git(args ...string) []byte {
	r.t.Helper()
	return r.run(r.root, nil, args...)
}

// gitIn is the same in another repository — the bare origin, for the assertions
// that read what arrived there.
func (r *repo) gitIn(dir string, args ...string) []byte {
	r.t.Helper()
	return r.run(dir, nil, args...)
}

// run is the one place this file starts a git subprocess. stdin is nil for
// every call but the two that write an object from bytes rather than from a
// path, which is the only axis the callers above differ on.
func (r *repo) run(dir string, stdin []byte, args ...string) []byte {
	r.t.Helper()
	return r.runWith(dir, r.env, stdin, args...)
}

// runWith is the same with the environment named, which one caller needs: a
// tree is built here through a temporary index, and GIT_INDEX_FILE is how a
// fixture points git at one without a worktree.
func (r *repo) runWith(dir string, env []string, stdin []byte, args ...string) []byte {
	r.t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes()
}

// text is git's answer where the whole of it is a line to read back — an object
// id, a formatted log line. Anything whose answer is bytes to keep goes through
// git, which trims nothing.
func (r *repo) text(args ...string) string {
	r.t.Helper()
	return strings.TrimSpace(string(r.git(args...)))
}

func (r *repo) textIn(dir string, args ...string) string {
	r.t.Helper()
	return strings.TrimSpace(string(r.gitIn(dir, args...)))
}

// origin wires a bare repository as the remote of that name, with the code
// branch pushed to it, and answers where it sits.
func (r *repo) origin() string {
	r.t.Helper()

	bare := filepath.Join(filepath.Dir(r.root), "origin.git")
	r.gitIn(filepath.Dir(r.root), "init", "--quiet", "--bare", bare)
	r.git("remote", "add", "origin", bare)
	r.git("push", "--quiet", "origin", "refs/heads/main:refs/heads/main")
	return bare
}

// seedStore builds a parentless commit whose tree holds one file and puts it on
// the named ref of the repository at gitdir — the shape a Store already in
// existence has, wherever it is.
func (r *repo) seedStore(gitdir, name, content string) string {
	r.t.Helper()

	blob := strings.TrimSpace(string(r.run(r.root, []byte(content), "hash-object", "-w", "--stdin")))
	tree := strings.TrimSpace(string(r.run(r.root, []byte("100644 blob "+blob+"\t"+name+"\n"), "mktree")))
	commit := r.text("commit-tree", tree, "-m", "a Store that already existed")
	if gitdir == r.root {
		r.git("update-ref", store.Ref, commit)
		return commit
	}
	r.git("push", "--quiet", gitdir, commit+":"+store.Ref)
	return commit
}

// seedFiles adds one commit to the Store branch of the repository at gitdir,
// holding the files handed on top of whatever the branch already holds, and
// answers the new commit.
//
// The tree is assembled through a temporary index rather than a checkout, for
// the reason the package itself never checks the Store out: a fixture that
// needed a worktree would be testing a shape §7 forbids (ADR-0075). The objects
// are written in the clone and pushed where the branch being seeded is the bare
// remote's, which is what seedStore does one file at a time.
func (r *repo) seedFiles(gitdir string, files map[string]string) string {
	r.t.Helper()

	index := filepath.Join(r.t.TempDir(), "index")
	env := append(slices.Clone(r.env), "GIT_INDEX_FILE="+index)

	parent := ""
	if r.hasStoreBranch(gitdir) {
		parent = r.textIn(gitdir, "rev-parse", store.Ref)
		r.runWith(r.root, env, nil, "read-tree", parent)
	}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		blob := strings.TrimSpace(string(r.run(r.root, []byte(files[path]), "hash-object", "-w", "--stdin")))
		r.runWith(r.root, env, nil, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+path)
	}
	tree := strings.TrimSpace(string(r.runWith(r.root, env, nil, "write-tree")))

	commit := []string{"commit-tree", tree, "-m", "a Run that already happened"}
	if parent != "" {
		commit = append(commit, "-p", parent)
	}
	written := r.text(commit...)
	if gitdir == r.root {
		r.git("update-ref", store.Ref, written)
		return written
	}
	r.git("push", "--quiet", gitdir, written+":"+store.Ref)
	return written
}

// seedVersions is seedFiles over Record versions, each written to the path the
// grammar gives it and in the encoding §7 fixes — which is the only shape the
// reader admits, so a case that seeds one is seeding a file `hyper` could have
// written.
func (r *repo) seedVersions(gitdir string, versions ...store.RecordVersion) string {
	r.t.Helper()

	files := map[string]string{}
	for _, version := range versions {
		files[store.RecordPath(version.Identity, version.Run, version.Step)] = string(version.Encode())
	}
	return r.seedFiles(gitdir, files)
}

// storeTree is the Store branch's files at gitdir, rendered as one string per
// entry: the path, and the bytes under it.
func (r *repo) storeTree(gitdir string) map[string]string {
	r.t.Helper()

	tree := map[string]string{}
	listing := strings.TrimSuffix(r.textIn(gitdir, "ls-tree", "-r", "--format=%(objectname) %(path)", store.Ref), "\n")
	if listing == "" {
		return tree
	}
	for _, line := range strings.Split(listing, "\n") {
		object, path, named := strings.Cut(line, " ")
		if !named {
			r.t.Fatalf("ls-tree line %q is not <object> <path>", line)
		}
		tree[path] = string(r.gitIn(gitdir, "cat-file", "blob", object))
	}
	return tree
}

// hasStoreBranch says whether the repository at gitdir holds the Store branch.
// It is the one git call whose failure is an answer rather than a fault.
func (r *repo) hasStoreBranch(gitdir string) bool {
	r.t.Helper()

	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", store.Ref)
	cmd.Dir = gitdir
	cmd.Env = r.env
	return cmd.Run() == nil
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("git is not on PATH: %v; hyper's record is kept in git and the suite assumes it (§13)", err)
	}
}

// TestInit_CreatesAParentlessCommitHoldingSTOREMdAlone is the whole of what
// `store init` does to a repository that has no Store: one orphan branch, one
// file on it, and no history behind it (§7, ADR-0075).
func TestInit_CreatesAParentlessCommitHoldingSTOREMdAlone(t *testing.T) {
	r := newRepo(t)

	done, err := store.Init(r.root, theInstant)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !done.Created {
		t.Error("Created = false, want the branch created where nothing held it")
	}
	if done.Pushed {
		t.Error("Pushed = true, want no push where no remote is configured")
	}

	if !r.hasStoreBranch(r.root) {
		t.Fatalf("%s is not there after Init", store.Ref)
	}
	if parents := r.text("rev-list", "--parents", "-1", store.Ref); len(strings.Fields(parents)) != 1 {
		t.Errorf("%s is %q; want a commit id alone, a parentless root carrying no history", store.Ref, parents)
	}
	if want := map[string]string{store.IntroductionPath: store.Introduction}; !equalTrees(r.storeTree(r.root), want) {
		t.Errorf("the branch holds %v, want %v", r.storeTree(r.root), want)
	}
}

// TestInit_CarriesHypersOwnIdentityAndTheThreadedClock is the pair of rules the
// git layer lands with (§7, issue #126). The identity is `hyper`'s constant and
// not the repository's git configuration — a runner that never set user.email
// would otherwise be unable to write the record at all, and who ran something
// is already the Journal's trigger.actor — and both dates come from the clock
// the caller threaded, so a fixture's branch is reproducible and `git log` on
// the Store is honest.
//
// The repository this runs against is configured with an identity of its own,
// so a commit that read the configuration would be visibly the wrong one rather
// than accidentally the right one.
func TestInit_CarriesHypersOwnIdentityAndTheThreadedClock(t *testing.T) {
	r := newRepo(t)
	r.git("config", "user.name", "somebody else")
	r.git("config", "user.email", "somebody@elsewhere.invalid")

	instant := time.Date(2019, time.July, 4, 12, 0, 0, 0, time.UTC)
	if _, err := store.Init(r.root, instant); err != nil {
		t.Fatalf("Init: %v", err)
	}

	stamp := strconv.FormatInt(instant.Unix(), 10)
	want := strings.Join([]string{
		store.CommitName, store.CommitEmail, stamp,
		store.CommitName, store.CommitEmail, stamp,
	}, "\n")
	if got := r.text("log", "-1", "--format=%an%n%ae%n%at%n%cn%n%ce%n%ct", store.Ref); got != want {
		t.Errorf("the commit carries:\n%s\nwant:\n%s", got, want)
	}
}

// TestInit_CreatesNothingWhereTheBranchIsAlreadyLocal is idempotence, and it is
// the one rewrite append-only forbids arriving through the file that looks least
// dangerous to touch: a second `init` writes no STORE.md and mints no commit
// (§7, §12, ADR-0011).
//
// This repository has no remote, so nothing at all is left for the second call
// to do; where one is configured, the push above is what it still does.
func TestInit_CreatesNothingWhereTheBranchIsAlreadyLocal(t *testing.T) {
	r := newRepo(t)

	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("the first Init: %v", err)
	}
	first := r.text("rev-parse", store.Ref)

	done, err := store.Init(r.root, theInstant.Add(time.Hour))
	if err != nil {
		t.Fatalf("the second Init: %v", err)
	}
	if done.Created {
		t.Error("Created = true on a second Init, want the Store found rather than made")
	}
	if done.Pushed {
		t.Error("Pushed = true with no remote configured")
	}
	if second := r.text("rev-parse", store.Ref); second != first {
		t.Errorf("%s moved from %s to %s; a second init writes nothing", store.Ref, first, second)
	}
}

// TestInit_PushesAStoreTheRemoteDoesNotHold is the second half of the push
// rule, and the one that decides what `init` is *for* once a Store exists. A
// branch that is here and nowhere else is the state every scheduled Run reads as
// no Store at all, so it goes out — this call created nothing and pushed
// something, which is a combination the row has to be able to state.
//
// It is also the only way back from a first `init` whose push was rejected: that
// leaves exactly this state, and a second `init` that stopped on finding the
// branch underfoot would leave it standing forever (§7).
func TestInit_PushesAStoreTheRemoteDoesNotHold(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("the first Init: %v", err)
	}
	// The branch is here and not there, which is the state under test.
	r.gitIn(bare, "update-ref", "-d", store.Ref)

	done, err := store.Init(r.root, theInstant)
	if err != nil {
		t.Fatalf("the second Init: %v", err)
	}
	if done.Created {
		t.Error("Created = true, want nothing created — the branch was already here")
	}
	if !done.Pushed {
		t.Error("Pushed = false, want the branch sent to a remote that did not hold it")
	}
	if local, remote := r.text("rev-parse", store.Ref), r.textIn(bare, "rev-parse", store.Ref); local != remote {
		t.Errorf("origin holds %s at %s, want the clone's %s", store.Ref, remote, local)
	}
}

// TestInit_PushesWhereARemoteIsConfigured is the largest thing this command
// mints and the reason it is not optional: a runner's clone never holds the
// branch and fetches it from the remote, so a Store that exists only on the
// laptop that ran `init` refuses every scheduled Run forever (§7).
func TestInit_PushesWhereARemoteIsConfigured(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()

	done, err := store.Init(r.root, theInstant)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !done.Created || !done.Pushed {
		t.Errorf("Init = %+v, want the branch created and pushed", done)
	}
	if !r.hasStoreBranch(bare) {
		t.Fatalf("origin does not hold %s", store.Ref)
	}
	if local, remote := r.text("rev-parse", store.Ref), r.textIn(bare, "rev-parse", store.Ref); local != remote {
		t.Errorf("origin holds %s at %s, want the clone's %s", store.Ref, remote, local)
	}
}

// TestInit_FetchesTheBranchOriginAlreadyHolds is the ordering rule this ticket
// is about. Two clones each minting an orphan root produce two histories that
// can never fast-forward into one another, and the second operator's every push
// fails forever with nothing to diagnose it by — so a branch on `origin` and
// not local is fetched rather than re-created (§7, ADR-0074).
func TestInit_FetchesTheBranchOriginAlreadyHolds(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	seeded := r.seedStore(bare, "STORE.md", "a Store somebody else created\n")

	done, err := store.Init(r.root, theInstant)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if done.Created {
		t.Error("Created = true, want the Store found on the remote rather than made — nothing was minted and no file was written")
	}
	if done.Pushed {
		t.Error("Pushed = true, want no push where origin already holds the branch")
	}
	if got := r.text("rev-parse", store.Ref); got != seeded {
		t.Errorf("%s is %s, want origin's %s — the branch is fetched, never re-created", store.Ref, got, seeded)
	}
	if want := map[string]string{"STORE.md": "a Store somebody else created\n"}; !equalTrees(r.storeTree(r.root), want) {
		t.Errorf("the branch holds %v, want the remote's %v", r.storeTree(r.root), want)
	}
}

// TestInit_TheFetchIsShallowAndUnfiltered is ADR-0074 stated as the two facts
// it decides: the sync takes the tip and no history, and it is never a filtered
// one. A blob or tree filter would make a version's `written_at` a lazy fetch,
// which is what would make *a read-only Run proceeds offline* false wherever
// the network is.
func TestInit_TheFetchIsShallowAndUnfiltered(t *testing.T) {
	r := newRepo(t)
	bare := r.origin()
	r.seedStore(bare, "STORE.md", "a Store somebody else created\n")

	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if shallow := r.text("rev-parse", "--is-shallow-repository"); shallow != "true" {
		t.Errorf("the clone is shallow: %s, want the tip and no history", shallow)
	}
	for _, setting := range []string{"remote.origin.promisor", "remote.origin.partialclonefilter"} {
		cmd := exec.Command("git", "config", "--get", setting)
		cmd.Dir, cmd.Env = r.root, r.env
		if out, err := cmd.Output(); err == nil {
			t.Errorf("%s is set to %q; the Store's content is always read and the fetch is never filtered", setting, strings.TrimSpace(string(out)))
		}
	}
}

// TestInit_NamesARepositoryRootThatIsNotOne is the usage error: there is no
// branch to create and no repository to refuse on behalf of, so it is `2`'s
// sentinel and not the world resisting.
func TestInit_NamesARepositoryRootThatIsNotOne(t *testing.T) {
	plain := t.TempDir()

	_, err := store.Init(plain, theInstant)
	if err == nil {
		t.Fatal("Init on a directory holding no git repository returned no error")
	}
	if !errors.Is(err, store.ErrNoRepository) {
		t.Errorf("Init returned %v, want it to be ErrNoRepository", err)
	}
	if entries, readErr := os.ReadDir(plain); readErr != nil || len(entries) != 0 {
		t.Errorf("the directory holds %v (%v), want it left exactly as it was", entries, readErr)
	}
}

// TestInit_ReportsARemoteItCannotReach is the other side of the same sort: the
// world resisting rather than a guardrail declining, so it is an ordinary error
// and never ErrNoRepository. §7's own reason is that a network coming back is
// not an act of anybody's (ADR-0061).
func TestInit_ReportsARemoteItCannotReach(t *testing.T) {
	r := newRepo(t)
	r.git("remote", "add", "origin", filepath.Join(t.TempDir(), "no-repository-here.git"))

	_, err := store.Init(r.root, theInstant)
	if err == nil {
		t.Fatal("Init against an unreachable remote returned no error")
	}
	if errors.Is(err, store.ErrNoRepository) {
		t.Errorf("Init returned %v, want an ordinary error and not ErrNoRepository", err)
	}
	if r.hasStoreBranch(r.root) {
		t.Errorf("%s was created against a remote that could not be read; the remote is looked at before anything is created", store.Ref)
	}
}

// TestInit_WritesToTheRepositoryItWasNamed rather than to the one the
// environment names. git sets GIT_DIR for every hook it runs, so a `hyper store
// init` invoked from one inherits a pointer to some other repository — and a
// command that let that decide where the record goes would create the Store
// somewhere the caller never named, in silence.
func TestInit_WritesToTheRepositoryItWasNamed(t *testing.T) {
	named, elsewhere := newRepo(t), newRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(elsewhere.root, ".git"))
	t.Setenv("GIT_WORK_TREE", elsewhere.root)

	done, err := store.Init(named.root, theInstant)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !done.Created {
		t.Error("Created = false, want the branch created in the repository that was named")
	}
	if !named.hasStoreBranch(named.root) {
		t.Errorf("%s is not in the repository Init was handed", store.Ref)
	}
	if elsewhere.hasStoreBranch(elsewhere.root) {
		t.Errorf("%s landed in the repository GIT_DIR named; the Store is a branch of the repository the artefacts sit in", store.Ref)
	}
}

// TestIntroduction_IsByteIdenticalInEveryRepository is what makes STORE.md a
// golden: it is written once, when the Store is created, and carries no schema
// version, no timestamp and no repository-specific fact (§7, §12).
func TestIntroduction_IsByteIdenticalInEveryRepository(t *testing.T) {
	first, second := newRepo(t), newRepo(t)
	for _, r := range []*repo{first, second} {
		if _, err := store.Init(r.root, theInstant); err != nil {
			t.Fatalf("Init: %v", err)
		}
	}

	a, b := first.storeTree(first.root), second.storeTree(second.root)
	if !equalTrees(a, b) {
		t.Errorf("two repositories hold different Stores:\n %v\n %v", a, b)
	}
	if a[store.IntroductionPath] != store.Introduction {
		t.Errorf("the branch holds %q, want the stated %q", a[store.IntroductionPath], store.Introduction)
	}
}

// TestIntroduction_MakesItsThreeClaims holds the prose to what §7 fixes about
// it — that every other file on the branch is machine-written, that the branch
// is the account of the world rather than part of it, and that editing it by
// hand is editing evidence (ADR-0011). §7 fixes the three claims and not the
// words, so what is asserted is that each one is made rather than how.
func TestIntroduction_MakesItsThreeClaims(t *testing.T) {
	for claim, phrase := range map[string]string{
		"every other file on the branch is machine-written": "machine-written",
		"the branch is the account of the world":            "account of the world",
		"editing it by hand is editing evidence":            "editing evidence",
	} {
		if !strings.Contains(store.Introduction, phrase) {
			t.Errorf("STORE.md does not say %s; it reads:\n%s", claim, store.Introduction)
		}
	}
	if strings.ContainsAny(store.Introduction, "0123456789") {
		t.Errorf("STORE.md carries a digit; it holds no timestamp, no schema version and no repository-specific fact:\n%s", store.Introduction)
	}
}

// equalTrees compares two path-against-content listings, which is what both a
// branch's tree and a working tree are read back as here.
func equalTrees(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for path, content := range a {
		if b[path] != content {
			return false
		}
	}
	return true
}
