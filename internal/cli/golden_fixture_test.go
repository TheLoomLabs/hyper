package cli_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/repository"
)

// A golden case can be a real git repository, and this file is the whole of
// what that costs (issue #125).
//
// Every case that has ever landed is a plain directory reached by --repo-dir,
// which is why two paths §9 states have never been driven by anything: the walk
// up to the git root, and the message a command writes when there is no git
// root to find. A checked-in fixture cannot carry a .git — testdata/exemption's
// README says so in as many words, and works around it — so the repository a
// case stands in has to be built at run time or not at all.
//
// It is built at run time. A case that asks for it has its repo/ copied into a
// temp directory, `git init`ed and committed whole, and the command is driven
// against that copy; a case that asks for nothing is driven exactly as before,
// from the same directory with the same argv, which is what leaves the five
// landed corpora untouched by construction rather than by inspection.
//
// Nothing here writes into the checked-in tree. The copy, the branches, the
// bare origin and git's own configuration all live under t.TempDir(), and
// TestMain below holds testdata/ to that byte for byte.

// The two branches a fixture holds, each spelled once as the name a human says
// and once as the ref git takes. `git checkout hyper-store` is what §7 promises
// a reader, and refs/heads/hyper-store is what every plumbing call here names,
// and they are one string so that the two cannot come apart.
//
// hyper-store is a literal here because no package states it yet: internal/store
// is milestone 4.3's and this ticket is the prefactor in front of it. When that
// package lands and exports the name, this pair is what to delete. main is named
// explicitly rather than left to init.defaultBranch, which is a setting on the
// machine running the suite and would otherwise reach into a fixture.
const (
	storeBranchName = "hyper-store"
	codeBranchName  = "main"

	storeBranch = "refs/heads/" + storeBranchName
	codeBranch  = "refs/heads/" + codeBranchName
)

// The identity every commit the fixture makes carries, author and committer
// alike. It is a constant rather than the machine's git configuration for the
// reason the Store's own commit identity will be one (§7, issue #124): a
// checkout that never set user.email would otherwise be unable to build a
// fixture at all, and the identity carries no fact any case reads back.
const (
	fixtureIdentityName  = "hyper golden fixture"
	fixtureIdentityEmail = "fixture@hyper.invalid"
)

// defaultInstant is the clock a case is driven at where it supplies no now
// file: one constant, stated here, so that the dates on a fixture's own commits
// are the same on two machines and a -update run is reproducible across them.
// It is the instant §7's own worked examples are written at.
const defaultInstant = "2026-04-02T09:41:14.221Z"

// absentBranch is what a branch golden holds where the branch is not there. It
// is a stated line rather than an empty file because an absent branch and a
// branch that exists and holds nothing are two different answers — the whole
// distinction store-absent rests on — and a golden that rendered both as
// nothing could not tell them apart (§7).
const absentBranch = "no " + storeBranchName + " branch\n"

// fixtureInputs is what a case asked the harness for, read off its own
// directory. Three of them materialise a repository, one seeds a branch on the
// remote, two decide where the invocation stands, and two are the goldens that
// assert a branch afterwards.
//
// They are read once, together, so that the harness can say what a case meant
// before it builds anything — an input that contradicts another is a fault the
// case is told about, not one it is quietly driven past.
type fixtureInputs struct {
	// repo says the case carries a fixture repository, which is what every
	// other input here is about and what materialisation copies.
	repo bool
	// git is the bare marker: materialise, and seed no branch. A case that
	// wants nothing but a git root asks for it with this and nothing else.
	git bool
	// store is a directory whose files become refs/heads/hyper-store's
	// content before the command runs, built as a parentless commit.
	store bool
	// remote wires a bare repository as origin, with the code branch pushed
	// to it.
	remote bool
	// remoteStore seeds hyper-store on origin and nowhere else — the shape
	// every runner's clone is in, and the one `store init` must not mint a
	// second root against (§7, ADR-0074).
	remoteStore bool
	// findRoot drives the case with no --repo-dir, so the root is the one
	// resolveRepoRoot finds by walking up from the working directory (§9).
	findRoot bool
	// noGitRoot drives the case from a directory that lies under no git
	// root, which is the other half of the same path: the message a command
	// writes when the walk finds nothing.
	noGitRoot bool
	// storeGolden and remoteGolden say the case asserts a branch. A case
	// supplying neither asserts nothing about any branch, which is every
	// case that landed before this ticket.
	storeGolden, remoteGolden bool
}

// materialised says the case is driven against a real git repository: it
// carries a fixture to copy, and asked for at least one of the three inputs
// that make one. Every case that supplies none of them is driven exactly as it
// was before this ticket.
func (in fixtureInputs) materialised() bool {
	return in.repo && (in.git || in.store || in.remote)
}

// fault names what a case asked for that the harness cannot honour, or "" where
// the inputs are coherent. It is a message rather than a boolean because every
// one of these is a mistake in a case directory, and the case's author is the
// reader: an input silently ignored is a fixture that asserts less than its
// directory says it does, which is the failure mode a golden corpus is least
// able to notice.
func (in fixtureInputs) fault() string {
	switch {
	case !in.repo && (in.git || in.store || in.remote || in.remoteStore):
		return "asks for a git fixture and carries no repo/ to make one from"
	case in.remoteStore && !in.remote:
		return "seeds hyper-store on origin and wires no origin; remote-store/ needs a remote marker beside it"
	case in.noGitRoot && in.repo:
		return "asks to be driven from under no git root and carries a repository to stand in; those are two different cases"
	case in.findRoot && !in.materialised():
		return "asks for the repository root to be found by walking up and materialises no repository; the walk would climb out of testdata/ and resolve hyper's own root"
	case in.storeGolden && !in.materialised():
		return "holds a store.golden and materialises no repository; there is no branch to render"
	case in.remoteGolden && !in.remote:
		return "holds a remote.golden and wires no origin; there is no remote branch to render"
	}
	return ""
}

// fixtureInputs reads the case directory for every optional input this file
// knows about. A marker is a file and a seed is a directory, and each is
// checked as the kind it is: a `store` file or a `git/` directory is a case
// that meant something the harness would otherwise silently not do.
func (c goldenCase) fixtureInputs() fixtureInputs {
	return fixtureInputs{
		repo:         isDir(filepath.Join(c.dir, "repo")),
		git:          isFile(filepath.Join(c.dir, "git")),
		store:        isDir(filepath.Join(c.dir, "store")),
		remote:       isFile(filepath.Join(c.dir, "remote")),
		remoteStore:  isDir(filepath.Join(c.dir, "remote-store")),
		findRoot:     isFile(filepath.Join(c.dir, "find-root")),
		noGitRoot:    isFile(filepath.Join(c.dir, "no-git-root")),
		storeGolden:  isFile(filepath.Join(c.dir, "store.golden")),
		remoteGolden: isFile(filepath.Join(c.dir, "remote.golden")),
	}
}

// instant is the clock the case is driven at: the RFC 3339 time in its now
// file, or the harness's stated constant. It is the value cli.Main is handed
// and the date on every commit the fixture itself makes, so that a materialised
// branch is the same branch twice.
func (c goldenCase) instant(t *testing.T) time.Time {
	t.Helper()

	named := strings.TrimSpace(readFile(t, filepath.Join(c.dir, "now")))
	if named == "" {
		named = defaultInstant
	}
	instant, err := time.Parse(time.RFC3339, named)
	if err != nil {
		t.Fatalf("%s: now is %q; want an RFC 3339 instant", c.name, named)
	}
	return instant
}

// gitFixture is one materialised case's git state: the working tree the command
// is driven against, the bare repository wired as origin where the case asked
// for one, and the environment every git subprocess is run with.
//
// The environment is carried rather than inherited because a fixture must not
// depend on the machine that builds it. git's system and global configuration
// are pointed at files that do not exist, HOME is a temp directory of its own,
// and the identity and both dates are supplied outright — so a repository whose
// owner signs every commit, or has never set user.email at all, builds the same
// bytes here.
type gitFixture struct {
	root   string
	origin string
	env    []string
}

// materialise builds the case's repository: repo/ copied into a temp directory,
// `git init`ed on a named branch and committed whole, then whatever branches
// and remote the case asked for. The command runs against the copy, so a case
// can be materialised while supplying nothing but the marker.
func (c goldenCase) materialise(t *testing.T, in fixtureInputs) gitFixture {
	t.Helper()
	requireGit(t)

	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	fx := gitFixture{
		root: filepath.Join(base, "repo"),
		env:  fixtureEnvironment(home, c.instant(t)),
	}
	if err := os.CopyFS(fx.root, os.DirFS(filepath.Join(c.dir, "repo"))); err != nil {
		t.Fatal(err)
	}

	fx.run(t, fx.root, "init", "--quiet", "--initial-branch="+codeBranchName)
	fx.run(t, fx.root, "add", "--all")
	// --allow-empty, because a fixture repository is allowed to be empty —
	// an assertion about the fixture itself needs no artefact in it — and a
	// code branch has to exist either way for the remote to hold one.
	fx.run(t, fx.root, "commit", "--quiet", "--message", "the fixture's working tree", "--allow-empty")

	if in.store {
		fx.run(t, fx.root, "update-ref", storeBranch, fx.orphan(t, filepath.Join(c.dir, "store")))
	}
	if in.remote {
		fx.origin = filepath.Join(base, "origin.git")
		fx.run(t, base, "init", "--quiet", "--bare", fx.origin)
		fx.run(t, fx.root, "remote", "add", "origin", fx.origin)
		fx.run(t, fx.root, "push", "--quiet", "origin", codeBranch+":"+codeBranch)
		if in.remoteStore {
			// The commit is built in the local repository and pushed by
			// its id, so origin gains the branch and the clone gains no
			// ref to it: hyper-store on the remote only, which is the
			// state a runner's fresh clone is always in (§7).
			fx.run(t, fx.root, "push", "--quiet", "origin", fx.orphan(t, filepath.Join(c.dir, "remote-store"))+":"+storeBranch)
		}
	}
	return fx
}

// orphan builds a parentless commit whose tree is dir's files, and returns its
// id. Nothing is checked out and no working tree is touched: the blobs are
// hashed straight into the object database and the tree is assembled through an
// index file of the commit's own, which is how §7 says every byte the Store
// ever holds is written (ADR-0075).
//
// Every entry is a regular file at 100644. The Store holds files and no
// directories of its own making, no symlinks and nothing executable, so a mode
// a fixture could vary is a fact no case has to state.
func (fx gitFixture) orphan(t *testing.T, dir string) string {
	t.Helper()

	// The paths are absolute because every git subprocess runs inside the
	// materialised copy, and a case's seed directory is named relative to the
	// package the suite is running from.
	seed, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	indexed := fx
	indexed.env = append(slices.Clone(fx.env), "GIT_INDEX_FILE="+filepath.Join(t.TempDir(), "index"))
	indexed.run(t, fx.root, "read-tree", "--empty")

	for _, rel := range filesUnder(t, seed) {
		blob := indexed.text(t, fx.root, "hash-object", "-w", "--no-filters", "--", filepath.Join(seed, filepath.FromSlash(rel)))
		indexed.run(t, fx.root, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+rel)
	}
	return indexed.text(t, fx.root, "commit-tree", indexed.text(t, fx.root, "write-tree"), "-m", "the fixture's "+storeBranchName)
}

// render is the Store branch of the repository at gitdir, as a branch golden
// holds it: every path the branch's tree carries, in sorted order, each under a
// header line naming it and its length. A branch that is not there renders the
// stated marker instead.
//
// The tree is rendered and the commits are not. Nothing hyper answers about the
// record is defined over the branch's commits (§7, ADR-0074), so a golden
// pinning a commit id, a tree id, an author or a date would hold the
// implementation to a fact the specification does not state — and would move
// every time the fixture's clock did. What the tree holds is exactly what §7
// calls authoritative.
//
// The length is on the header line because the bytes below it are verbatim: a
// file that does not end in a newline would otherwise run into the next header,
// and a golden a reader cannot parse is one they cannot check.
func (fx gitFixture) render(t *testing.T, gitdir string) string {
	t.Helper()

	if !fx.hasStoreBranch(t, gitdir) {
		return absentBranch
	}

	var rendered strings.Builder
	for _, entry := range fx.storeTree(t, gitdir) {
		blob := fx.run(t, gitdir, "cat-file", "blob", entry.object)
		fmt.Fprintf(&rendered, "=== %s (%d bytes)\n", entry.path, len(blob))
		rendered.Write(blob)
	}
	return rendered.String()
}

// treeEntry is one file on the Store branch: where it sits, and the blob whose
// bytes the golden writes out.
type treeEntry struct {
	path   string
	object string
}

// storeTree lists the Store branch's files, sorted by path. The listing is
// asked for NUL-separated so that a path git would otherwise quote arrives
// whole, and it is sorted here rather than trusted from git so that the order
// the golden holds is one this file states.
func (fx gitFixture) storeTree(t *testing.T, gitdir string) []treeEntry {
	t.Helper()

	var entries []treeEntry
	for _, record := range nulSeparated(fx.run(t, gitdir, "ls-tree", "-r", "-z", storeBranch)) {
		meta, path, named := strings.Cut(record, "\t")
		fields := strings.Fields(meta)
		if !named || len(fields) != 3 {
			t.Fatalf("ls-tree record %q is not <mode> <type> <object>\\t<path>", record)
		}
		entries = append(entries, treeEntry{path: path, object: fields[2]})
	}
	slices.SortFunc(entries, func(a, b treeEntry) int { return strings.Compare(a.path, b.path) })
	return entries
}

// hasStoreBranch says whether the repository at gitdir holds the Store branch.
// It is the one git call whose failure is an answer rather than a fault, which
// is why it does not go through run.
func (fx gitFixture) hasStoreBranch(t *testing.T, gitdir string) bool {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", storeBranch)
	cmd.Dir = gitdir
	cmd.Env = fx.env
	return cmd.Run() == nil
}

// run runs one git command in gitdir and returns its stdout untouched. Any
// failure is the harness's own — a fixture that cannot be built is not a case
// that failed — so it stops the test where it happened, with git's stderr
// quoted.
func (fx gitFixture) run(t *testing.T, gitdir string, args ...string) []byte {
	t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = gitdir
	cmd.Env = fx.env
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes()
}

// text runs a git command whose whole answer is text to read back — an object
// id, a formatted log line — and returns it with the trailing newline off.
// Anything whose answer is bytes to keep goes through run, which touches
// nothing: a blob is not text and must not be trimmed.
func (fx gitFixture) text(t *testing.T, gitdir string, args ...string) string {
	t.Helper()
	return strings.TrimSpace(string(fx.run(t, gitdir, args...)))
}

// fixtureEnvironment is the environment every git subprocess a fixture runs is
// given, and it is the complete one: nothing of the machine's is inherited
// except PATH, which is how git is found at all (§13 — the record costs one
// runtime dependency, and this is it). It is a POSIX environment and stated as
// one; a platform whose git needs more of the process than its PATH is a
// platform this list grows for, deliberately, rather than by inheriting the
// whole of one and losing the property above.
//
// Both dates come from the case's clock and both identities from the constants
// above, so the commits a fixture makes are the same commits on two machines. A
// git date is whole seconds, so a clock carrying milliseconds reaches a commit
// truncated — the sub-second half is for the entry point, which is handed the
// instant itself and where retention will read it (§7).
func fixtureEnvironment(home string, instant time.Time) []string {
	stamp := instant.UTC().Format(time.RFC3339)
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		// Two paths under a directory that holds neither file: git reads
		// an absent configuration as an empty one, so the machine's own
		// global and system git config reach no fixture.
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "absent-global-config"),
		"GIT_CONFIG_SYSTEM=" + filepath.Join(home, "absent-system-config"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=" + fixtureIdentityName,
		"GIT_AUTHOR_EMAIL=" + fixtureIdentityEmail,
		"GIT_COMMITTER_NAME=" + fixtureIdentityName,
		"GIT_COMMITTER_EMAIL=" + fixtureIdentityEmail,
		"GIT_AUTHOR_DATE=" + stamp,
		"GIT_COMMITTER_DATE=" + stamp,
		// A fixture is built without a human present: a git that stopped
		// to ask for a credential would hang the suite rather than fail it.
		"GIT_TERMINAL_PROMPT=0",
		"TZ=UTC",
		"LC_ALL=C",
	}
}

// requireGit stops the suite where git is not on PATH, rather than skipping.
// §13 states that the record costs one runtime dependency and that it is the
// only one, so a suite that could not assume git would be testing a binary that
// cannot ship.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("git is not on PATH: %v; hyper's record is kept in git and the suite assumes it (§13)", err)
	}
}

// outsideAnyRepository is a directory under no git root, which is where a case
// carrying the no-git-root marker stands. It is the one fixture the harness
// cannot build on demand — a temp directory is outside a repository or it is
// not — so where the platform's temp directory turns out to sit inside one the
// case skips rather than passing vacuously, on cmd/hyper's own precedent for a
// probe that cannot be built.
func outsideAnyRepository(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if root, found := repository.FindGitRoot(dir); found {
		t.Skipf("the platform's temp directory lies inside the git repository at %s; a case standing under no git root cannot be built here", root)
	}
	return dir
}

// filesUnder lists dir's files, relative to it with forward slashes and sorted,
// which is the order their blobs are hashed and their paths written into a
// tree.
func filesUnder(t *testing.T, dir string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)
	return files
}

// isFile says whether a case supplied an optional file input — a marker, or a
// golden it means to assert. It is isDir's other half, and the two are separate
// so that an input written as the wrong kind is an input the harness does not
// find rather than one it half-honours.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// nulSeparated reads a listing git wrote NUL-separated, which is how every
// listing here is asked for: a path git would otherwise quote arrives whole,
// and a path carrying a newline cannot be mistaken for two.
func nulSeparated(listing []byte) []string {
	trimmed := strings.TrimSuffix(string(listing), "\x00")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\x00")
}

// under says whether path is dir or something inside it. It is a directory
// walk and not a string prefix, which would read /tmp/x/repo-other as being
// inside /tmp/x/repo.
func under(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// testdataRoot is the checked-in corpora, absolute: what a fixture must be
// built outside of, and what every case that materialises nothing stands in.
func testdataRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// What follows is what the fixture promises, asserted rather than assumed.
// Most of it is invisible in a golden: a case whose --repo-dir was spliced when
// it asked for the walk renders exactly the same bytes as one where the walk
// ran, a fixture's commit dates appear in no rendering at all, and an input the
// harness quietly ignored looks like a case that never asked for it.
//
// None of it names a case. Every assertion below finds its subjects by what
// their directories supply and fences on having found any, which is the rule
// the corpus-wide assertions in golden_test.go already state twice: a case is
// judged by having been written, and a list of names here would be a
// registration by another name — one that a case renamed or retired unhooks in
// silence.

// eachMaterialisedCase materialises every case whose inputs wants accepts and
// hands each one to visit as its own subtest, answering how many it found. The
// count is the caller's fence: an assertion that ran against nothing is one
// that held vacuously, and the corpora move.
func eachMaterialisedCase(t *testing.T, wants func(fixtureInputs) bool, visit func(*testing.T, goldenCase, gitFixture)) int {
	t.Helper()

	var judged int
	for _, c := range goldenCases(t) {
		in := c.fixtureInputs()
		if !in.materialised() || !wants(in) {
			continue
		}
		judged++
		t.Run(c.name, func(t *testing.T) {
			visit(t, c, c.materialise(t, in))
		})
	}
	return judged
}

// anyFixture accepts every materialised case, for the assertions that hold of
// all of them rather than of one input.
func anyFixture(fixtureInputs) bool { return true }

// TestGoldenFixture_AFindRootCaseNamesNoRepository is the assertion the walk-up
// cases cannot make for themselves. Their answer is a clean check either way:
// spliced with a --repo-dir naming the fixture root, or handed no repository at
// all and left to walk up to the same place, `hyper check` writes the same line
// — so the golden is silent about which of the two happened, and the thing
// under test is the one the golden cannot see (§9, issue #125).
func TestGoldenFixture_AFindRootCaseNamesNoRepository(t *testing.T) {
	testdata := testdataRoot(t)

	// The criterion is two cases and not one — the walk is driven from the
	// fixture root and from a subdirectory of it — and the two are separated
	// by a wd file alone, so nothing but this would notice the second losing
	// it and becoming a copy of the first.
	var atTheRoot, below int
	for _, c := range goldenCases(t) {
		if !c.fixtureInputs().findRoot {
			continue
		}
		// The case is driven once and read twice: a second invocation
		// would materialise a second copy, and the two would agree about
		// nothing.
		run := c.invocation(t)
		if run.wd == run.fixture.root {
			atTheRoot++
		} else {
			below++
		}

		t.Run(c.name, func(t *testing.T) {
			if slices.ContainsFunc(run.args, func(a string) bool {
				return a == "--repo-dir" || strings.HasPrefix(a, "--repo-dir=")
			}) {
				t.Errorf("args = %q; a find-root case is driven with no --repo-dir, so the root is the one the walk finds", run.args)
			}
			if !under(run.wd, run.fixture.root) {
				t.Errorf("wd = %q; want it inside the materialised copy at %q", run.wd, run.fixture.root)
			}
			if under(run.wd, testdata) {
				t.Errorf("wd = %q is inside the checked-in corpora; the walk would climb past it to hyper's own root", run.wd)
			}
		})
	}

	if atTheRoot == 0 {
		t.Error("no find-root case stands at the fixture root; the walk that finds it underfoot is driven by nothing")
	}
	if below == 0 {
		t.Error("no find-root case stands in a subdirectory of the fixture; the walk that has somewhere to climb from is driven by nothing")
	}
}

// TestGoldenFixture_ANonMaterialisedCaseIsDrivenFromTheCheckedInTree is the
// criterion the five landed corpora rest on, held forwards rather than by
// having noticed that their goldens did not move: a case supplying none of the
// git inputs is driven from the same directory it always was, off the disk,
// with the argv it always had.
func TestGoldenFixture_ANonMaterialisedCaseIsDrivenFromTheCheckedInTree(t *testing.T) {
	testdata := testdataRoot(t)

	var judged int
	for _, c := range goldenCases(t) {
		in := c.fixtureInputs()
		if in.materialised() || in.noGitRoot {
			continue
		}
		judged++
		t.Run(c.name, func(t *testing.T) {
			run := c.invocation(t)

			if !under(run.wd, testdata) {
				t.Errorf("wd = %q; a case that materialises nothing stands in the checked-in tree under %q", run.wd, testdata)
			}
			want := c.argv
			if in.repo {
				want = append([]string{c.argv[0], "--repo-dir", c.abs(t, "repo")}, c.argv[1:]...)
			}
			if !slices.Equal(run.args, want) {
				t.Errorf("args = %q, want %q", run.args, want)
			}
		})
	}

	// The five landed corpora are every one of them a case that materialises
	// nothing, so this holding vacuously would mean the harness had decided
	// they all materialise something — which is the regression it is here to
	// catch, wearing the one disguise a walk cannot see through.
	if judged == 0 {
		t.Fatal("every case under testdata/ materialises a repository; the corpora that predate the fixture are being driven as something else")
	}
}

// TestGoldenFixture_AMaterialisedCaseIsDrivenAgainstTheCopy is the other half
// of the walk-up assertion, and invisible in a golden for the same reason: a
// --repo-dir naming the checked-in repo/ and one naming its copy make `check`
// write the same line, because the copy is a copy. What separates them is that
// only one of the two is inside a git repository, which is the whole point of
// materialising anything.
func TestGoldenFixture_AMaterialisedCaseIsDrivenAgainstTheCopy(t *testing.T) {
	testdata := testdataRoot(t)

	var judged int
	for _, c := range goldenCases(t) {
		in := c.fixtureInputs()
		if !in.materialised() || in.findRoot {
			continue
		}
		judged++
		t.Run(c.name, func(t *testing.T) {
			run := c.invocation(t)

			want := append([]string{c.argv[0], "--repo-dir", run.fixture.root}, c.argv[1:]...)
			if !slices.Equal(run.args, want) {
				t.Errorf("args = %q, want %q", run.args, want)
			}
			if under(run.fixture.root, testdata) {
				t.Errorf("the fixture root is %q, inside the checked-in corpora; a materialised case is a copy in a temp directory", run.fixture.root)
			}
			if root, found := repository.FindGitRoot(run.fixture.root); !found || root != run.fixture.root {
				t.Errorf("FindGitRoot(%q) = %q, %v; the copy is the git root the command resolves", run.fixture.root, root, found)
			}
		})
	}

	if judged == 0 {
		t.Fatal("no case materialises a repository and names it; the copy is driven by nothing")
	}
}

// TestGoldenFixture_TheCommitHoldsTheWholeWorkingTree is the half of
// materialisation nothing else would notice. The command reads the working
// tree, so a fixture that copied every file and committed none of them — or
// committed some — answers identically on all three streams; what the commit
// is for is the code branch the remote holds and the history a Store branch
// stands beside, and both are silent about their own contents.
func TestGoldenFixture_TheCommitHoldsTheWholeWorkingTree(t *testing.T) {
	judged := eachMaterialisedCase(t, anyFixture, func(t *testing.T, c goldenCase, fx gitFixture) {
		committed := nulSeparated(fx.run(t, fx.root, "ls-tree", "-r", "--name-only", "-z", codeBranch))
		if want := filesUnder(t, c.abs(t, "repo")); !slices.Equal(committed, want) {
			t.Errorf("%s holds %q, want the whole of repo/: %q", codeBranch, committed, want)
		}
	})

	if judged == 0 {
		t.Fatal("no case materialises a repository; the commit is driven by nothing")
	}
}

// TestGoldenFixture_ItsCommitsCarryOneIdentityAndTheCasesClock is what makes a
// -update run reproducible across machines: the fixture supplies both
// identities and both dates outright, so the branch a case materialises on one
// checkout is the branch it materialises on another. A commit taking its author
// from the machine's git configuration, or its date from the wall clock, would
// be a fixture whose bytes are nobody's to check.
//
// Every branch a case built is read, because they are written by two different
// git mechanisms — a commit against a worktree, and a parentless commit-tree
// against an index of its own — and only one of them going through the
// environment would be a difference nothing else notices.
func TestGoldenFixture_ItsCommitsCarryOneIdentityAndTheCasesClock(t *testing.T) {
	judged := eachMaterialisedCase(t, anyFixture, func(t *testing.T, c goldenCase, fx gitFixture) {
		fx.assertCommitsAt(t, c.instant(t))
	})
	if judged == 0 {
		t.Fatal("no case materialises a repository; the fixture's own commits are driven by nothing")
	}

	// The other reading of the clock, which no landed case exercises: a case
	// that names an instant is materialised at that one rather than at the
	// constant. It is fabricated rather than added to the corpora, because a
	// case whose only subject is the fixture's clock would assert nothing
	// about any command.
	t.Run("an instant the case names", func(t *testing.T) {
		named := goldenCase{dir: t.TempDir(), name: "an instant the case names"}
		writeInput(t, named.dir, "repo/")
		writeInput(t, named.dir, "git")
		if err := os.WriteFile(filepath.Join(named.dir, "now"), []byte("2019-07-04T12:00:00Z\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		instant := named.instant(t)
		if want := time.Date(2019, time.July, 4, 12, 0, 0, 0, time.UTC); !instant.Equal(want) {
			t.Fatalf("the case names %s and is driven at %s", want, instant)
		}
		named.materialise(t, named.fixtureInputs()).assertCommitsAt(t, instant)
	})

	// The constant is the one this file states, read back through a case that
	// supplies no now file: a default that drifted from the line above it
	// would be a -update run whose bytes nobody could reproduce.
	stated, err := time.Parse(time.RFC3339, defaultInstant)
	if err != nil {
		t.Fatalf("defaultInstant is %q; want an RFC 3339 instant", defaultInstant)
	}
	if got := (goldenCase{dir: t.TempDir(), name: "a case naming no instant"}).instant(t); !got.Equal(stated) {
		t.Errorf("a case naming no instant is driven at %s, want the stated %s", got, stated)
	}
}

// assertCommitsAt holds every branch the fixture built to the one identity and
// the one instant. The dates are read as seconds since the epoch rather than as
// git's own rendering of them, which varies with the notation the date was
// handed to it in: what the fixture promises is the instant, and a golden
// nowhere renders one.
func (fx gitFixture) assertCommitsAt(t *testing.T, instant time.Time) {
	t.Helper()

	stamp := strconv.FormatInt(instant.Unix(), 10)
	want := strings.Join([]string{
		fixtureIdentityName, fixtureIdentityEmail, stamp,
		fixtureIdentityName, fixtureIdentityEmail, stamp,
	}, "\n")

	branches := []string{codeBranch}
	if fx.hasStoreBranch(t, fx.root) {
		branches = append(branches, storeBranch)
	}
	for _, branch := range branches {
		if got := fx.text(t, fx.root, "log", "-1", "--format=%an%n%ae%n%at%n%cn%n%ce%n%ct", branch); got != want {
			t.Errorf("%s carries:\n%s\nwant:\n%s", branch, got, want)
		}
	}
}

// TestGoldenFixture_TheStoreBranchIsAParentlessCommit is what makes the seeded
// branch the branch §7 describes rather than a commit on the code history. The
// Store is an orphan branch — a parentless root whose tree is the record and
// nothing else — and a fixture that hung it off the working tree's commit would
// be seeding a shape the tool will never meet (§7, ADR-0075).
func TestGoldenFixture_TheStoreBranchIsAParentlessCommit(t *testing.T) {
	judged := eachMaterialisedCase(t, func(in fixtureInputs) bool { return in.store }, func(t *testing.T, c goldenCase, fx gitFixture) {
		if parents := fx.text(t, fx.root, "rev-list", "--parents", "-1", storeBranch); len(strings.Fields(parents)) != 1 {
			t.Errorf("%s is %q; want a commit id alone, a parentless root carrying no history", storeBranch, parents)
		}
		if code, store := fx.text(t, fx.root, "rev-parse", codeBranch), fx.text(t, fx.root, "rev-parse", storeBranch); code == store {
			t.Errorf("%s and %s are one commit; the Store is a branch of its own", codeBranch, storeBranch)
		}
	})

	if judged == 0 {
		t.Fatal("no case seeds a store/; the parentless root is driven by nothing")
	}
}

// TestGoldenFixture_TheRenderingIsSortedPathsAndTheirBytes reads a branch
// golden back the way a reader does, and holds it to the two things it claims:
// the paths are the seed's, in sorted order, and the bytes under each header
// are that file's. It parses by the byte counts rather than by looking for the
// next header, which is the whole reason a count is on the header line — a
// rendering a reader cannot walk is one they cannot check.
func TestGoldenFixture_TheRenderingIsSortedPathsAndTheirBytes(t *testing.T) {
	judged := eachMaterialisedCase(t, func(in fixtureInputs) bool { return in.store }, func(t *testing.T, c goldenCase, fx gitFixture) {
		seed := c.abs(t, "store")
		rendered := parseRendering(t, fx.render(t, fx.root))

		var paths []string
		for _, file := range rendered {
			paths = append(paths, file.path)
			want, err := os.ReadFile(filepath.Join(seed, filepath.FromSlash(file.path)))
			if err != nil {
				t.Errorf("the rendering carries %s, which the case does not seed: %v", file.path, err)
				continue
			}
			if file.bytes != string(want) {
				t.Errorf("%s renders %q, want the seeded %q", file.path, file.bytes, want)
			}
		}
		if !slices.IsSorted(paths) {
			t.Errorf("the rendering is %q; want the branch's paths in sorted order", paths)
		}
		if want := filesUnder(t, seed); !slices.Equal(paths, want) {
			t.Errorf("the rendering carries %q, want every path the case seeds: %q", paths, want)
		}
	})

	if judged == 0 {
		t.Fatal("no case seeds a store/; the rendering is driven by nothing")
	}
}

// renderedFile is one entry read back out of a branch golden.
type renderedFile struct {
	path  string
	bytes string
}

// parseRendering walks a branch golden — a header line, then exactly the bytes
// it names, then the next header — and fails where it does not hold. It is the
// reader the format is for, and a format only its writer can read is one the
// golden proves nothing to.
func parseRendering(t *testing.T, rendered string) []renderedFile {
	t.Helper()

	var files []renderedFile
	for rest := rendered; rest != ""; {
		header, body, complete := strings.Cut(rest, "\n")
		if !complete {
			t.Fatalf("the rendering ends mid-header at %q", header)
		}
		named, isHeader := strings.CutPrefix(header, "=== ")
		at := strings.LastIndex(named, " (")
		if !isHeader || at < 0 {
			t.Fatalf("%q is not a header line naming a path and its length", header)
		}
		size, err := strconv.Atoi(strings.TrimSuffix(named[at+2:], " bytes)"))
		if err != nil {
			t.Fatalf("%q names no length: %v", header, err)
		}
		if len(body) < size {
			t.Fatalf("%q names %d bytes and %d follow it", header, size, len(body))
		}
		files = append(files, renderedFile{path: named[:at], bytes: body[:size]})
		rest = body[size:]
	}
	return files
}

// TestGoldenFixture_AnAbsentBranchAndAnEmptyOneAreDifferentAnswers is the
// distinction the whole marker line exists for. §7 makes a Run that cannot find
// the branch Refuse store-absent rather than read it as empty — a fetch that
// failed mid-flight and a branch that was never created look identical from the
// inside — so a golden that rendered both as nothing would be unable to hold
// the one fact the record's first command is about.
//
// The empty branch is built here rather than by a case, because a directory
// with nothing in it is not something a corpus can check in: git tracks files,
// so an empty store/ would arrive at a fresh clone as no store/ at all.
func TestGoldenFixture_AnAbsentBranchAndAnEmptyOneAreDifferentAnswers(t *testing.T) {
	judged := eachMaterialisedCase(t, func(in fixtureInputs) bool { return !in.store }, func(t *testing.T, c goldenCase, fx gitFixture) {
		if got := fx.render(t, fx.root); got != absentBranch {
			t.Errorf("a repository with no Store renders %q, want the marker %q", got, absentBranch)
		}

		fx.run(t, fx.root, "update-ref", storeBranch, fx.orphan(t, t.TempDir()))
		if got := fx.render(t, fx.root); got != "" {
			t.Errorf("a Store branch holding nothing renders %q, want it empty and so distinguishable from an absent one", got)
		}
	})

	if judged == 0 {
		t.Fatal("no case materialises a repository without seeding a store/; the absent branch is driven by nothing")
	}
}

// TestGoldenFixture_NeitherGoldenRendersACommitOrADate holds the rendering to
// the tree. Nothing hyper answers about the record is defined over the branch's
// commits (§7, ADR-0074), so a golden carrying a commit id, a tree id, an
// author or a date would pin the implementation to a fact the specification
// does not state — and would move on every machine whose fixture clock did.
func TestGoldenFixture_NeitherGoldenRendersACommitOrADate(t *testing.T) {
	judged := eachMaterialisedCase(t, func(in fixtureInputs) bool { return in.store }, func(t *testing.T, c goldenCase, fx gitFixture) {
		rendered := fx.render(t, fx.root)

		for what, unwanted := range map[string]string{
			"the commit id":        fx.text(t, fx.root, "rev-parse", storeBranch),
			"the tree id":          fx.text(t, fx.root, "rev-parse", storeBranch+"^{tree}"),
			"the author":           fixtureIdentityName,
			"the author's address": fixtureIdentityEmail,
			"the commit date":      c.instant(t).UTC().Format(time.RFC3339),
		} {
			if strings.Contains(rendered, unwanted) {
				t.Errorf("the rendering carries %s (%q); a branch golden renders the tree and nothing about the commits that built it", what, unwanted)
			}
		}
	})

	if judged == 0 {
		t.Fatal("no case seeds a store/; the rendering is driven by nothing")
	}
}

// TestGoldenFixture_TheCodeBranchReachesTheRemote is the half of the remote
// fixture neither golden can show. store.golden and remote.golden render one
// ref between them, so a bare repository wired as origin and holding nothing
// would satisfy both — and `store init`'s push, which is what the remote
// fixture is being built for, needs a remote a push can reach.
func TestGoldenFixture_TheCodeBranchReachesTheRemote(t *testing.T) {
	judged := eachMaterialisedCase(t, func(in fixtureInputs) bool { return in.remote }, func(t *testing.T, c goldenCase, fx gitFixture) {
		if fx.origin == "" {
			t.Fatal("the case wires a remote and the fixture has none")
		}
		if local, remote := fx.text(t, fx.root, "rev-parse", codeBranch), fx.text(t, fx.origin, "rev-parse", codeBranch); local != remote {
			t.Errorf("origin holds %s at %s, want the clone's %s", codeBranch, remote, local)
		}

		// Which side holds the Store is the case's to say, and the pair
		// is the point: remote-store/ seeds the branch on origin alone,
		// which is the state a runner's fresh clone is always in and the
		// one this fixture exists to reach (§7).
		in := c.fixtureInputs()
		if held, want := fx.hasStoreBranch(t, fx.origin), in.remoteStore; held != want {
			t.Errorf("origin holds %s: %v, want %v — the case %s a remote-store/", storeBranch, held, want, seedsOrNot(want))
		}
		if held, want := fx.hasStoreBranch(t, fx.root), in.store; held != want {
			t.Errorf("the clone holds %s: %v, want %v — the case %s a store/", storeBranch, held, want, seedsOrNot(want))
		}
	})

	if judged == 0 {
		t.Fatal("no case wires a remote; origin is driven by nothing")
	}
}

// seedsOrNot reads a case's own input back into the failure above, so that a
// mismatch says which side the case asked for rather than only which side it
// got.
func seedsOrNot(seeded bool) string {
	if seeded {
		return "seeds"
	}
	return "seeds no"
}

// TestGoldenFixture_AnInputWithNothingToActOnIsNamed is the loud failure. Every
// one of these is a case directory whose author meant something the harness
// cannot do, and the alternative to naming it is a fixture that asserts less
// than its directory says it does — a store/ nothing seeds, a remote-store/
// with no remote to put it on — which is the one failure a green corpus is
// least able to reveal.
func TestGoldenFixture_AnInputWithNothingToActOnIsNamed(t *testing.T) {
	for name, inputs := range map[string][]string{
		"a store with no repository to put it on":  {"store/"},
		"a remote with no repository to clone":     {"remote"},
		"a git marker with nothing to initialise":  {"git"},
		"a remote store and no remote":             {"repo/", "remote-store/"},
		"a walk with nothing to walk up to":        {"repo/", "find-root"},
		"a store golden over no branch":            {"repo/", "store.golden"},
		"a remote golden over no remote":           {"repo/", "git", "remote.golden"},
		"a repository and nowhere to stand beside": {"repo/", "git", "no-git-root"},
	} {
		t.Run(name, func(t *testing.T) {
			if fault := fabricate(t, name, inputs).fixtureInputs().fault(); fault == "" {
				t.Errorf("the harness accepts a case that %s; want it named rather than ignored", name)
			}
		})
	}

	// The whole of what a landed case supplies, which must not be a fault:
	// the fence is only worth having if it lets the corpora through.
	for name, inputs := range map[string][]string{
		"a case supplying nothing at all": nil,
		"a case supplying a repository":   {"repo/"},
		"a seeded store":                  {"repo/", "store/", "store.golden"},
		"a store on origin alone":         {"repo/", "remote", "remote-store/", "store.golden", "remote.golden"},
		"a walk up to the git root":       {"repo/", "git", "find-root"},
		"a directory under no git root":   {"no-git-root"},
	} {
		t.Run(name, func(t *testing.T) {
			if fault := fabricate(t, name, inputs).fixtureInputs().fault(); fault != "" {
				t.Errorf("the harness rejects %s: %s", name, fault)
			}
		})
	}

	// And the corpora themselves, which is the reading that matters: a case
	// that has landed carrying an incoherent set of inputs is one the harness
	// stops the whole run on, so it must be found here rather than there.
	for _, c := range goldenCases(t) {
		if fault := c.fixtureInputs().fault(); fault != "" {
			t.Errorf("case %s %s", c.name, fault)
		}
	}
}

// fabricate is a case directory built for an assertion rather than checked in:
// the inputs named, and nothing else.
func fabricate(t *testing.T, name string, inputs []string) goldenCase {
	t.Helper()

	c := goldenCase{dir: t.TempDir(), name: name, argv: []string{"check"}}
	for _, input := range inputs {
		writeInput(t, c.dir, input)
	}
	return c
}

// writeInput creates one case input in dir, as the kind its name says it is: a
// trailing slash is a directory a case seeds from, and anything else a file.
func writeInput(t *testing.T, dir, input string) {
	t.Helper()

	if path, isDirectory := strings.CutSuffix(input, "/"); isDirectory {
		if err := os.MkdirAll(filepath.Join(dir, path), 0o755); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.WriteFile(filepath.Join(dir, input), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
