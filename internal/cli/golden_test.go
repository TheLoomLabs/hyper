package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/version"
)

// update regenerates the golden files of every case instead of checking
// against them (issue #88: "Golden files are checked in and regenerated behind
// an -update flag"). One flag serves them all — a corpus that regenerated
// behind a switch of its own would be one a `-update` run silently left stale.
var update = flag.Bool("update", false, "regenerate golden files")

// There is one harness, and it drives every case under testdata/ through
// cli.Main from that case's own argv (issue #108). Four harnesses stood here
// while each command had an entry point of its own; with the complete argv
// reaching one function they were four copies of one thing, and this milestone
// would otherwise have added four more.
//
// The per-command subtree survives the collapse (issue #101): testdata/check/,
// testdata/version/, testdata/completions/ and whatever lands beside them hold
// one directory per case, and a case's directory says which command it
// exercises. What changed is that a case belongs to a *command* by where it
// sits and to no harness at all — TestGolden walks testdata/ whole and needs no
// list of the commands it will find there.

// goldenCase is one case directory as the harness reads it: its argv, already
// parsed, and whatever optional inputs lie beside it on disk.
//
// The argv is read once, here, rather than at each place that wants something
// out of it. The command a case exercises is argv[0] and three things ask for
// it — the dispatch, which is handed it; the fence below, which holds the case
// directory's name against it; and the splice that puts --repo-dir after it —
// and a case that answered the question differently depending on who asked
// would be one whose subtree said one thing and whose run did another.
type goldenCase struct {
	// dir is the case directory, relative to the package, and name is the
	// same path with testdata/ trimmed off — what the subtest is called.
	dir, name string
	// argv is what the entry point receives: the case's complete command
	// line with the program name off the front, so argv[0] is the command.
	argv []string
}

// goldenCases enumerates every case under testdata/, in walk order. A case is
// a directory holding an argv, wherever it sits: the corpora are directories
// like any other, so a corpus that lands in a later milestone is driven by
// being written rather than by being registered here.
func goldenCases(t *testing.T) []goldenCase {
	t.Helper()

	var cases []goldenCase
	walkTestdata(t, "argv", func(dir string) {
		cases = append(cases, goldenCase{
			dir:  dir,
			name: filepath.ToSlash(strings.TrimPrefix(dir, "testdata"+string(filepath.Separator))),
			argv: readArgv(t, filepath.Join(dir, "argv")),
		})
	})
	if len(cases) == 0 {
		t.Fatal("no case under testdata/ holds an argv; the harness would pass having driven nothing")
	}
	return cases
}

// corpusCase is one case under testdata/ as a driver beside TestGolden reads
// it: its directory, the subtest name it is known by, and the argv it is driven
// from.
//
// It exists because the drivers that reach past a golden — the signal, the push
// tally, the three ways a Run loses the Store — each drive a case TestGolden
// already knows, and reading its directory a second way is how the day comes
// that one of them drives something else.
//
// argv is the case's own `argv` file where it has one, and the case names it
// here where it has none. Those cases carry none deliberately: a directory
// holding an argv is one TestGolden walks, and a case whose streams name a temp
// directory is one no golden can hold (run_store_lost_test.go).
func corpusCase(t *testing.T, name string, argv ...string) goldenCase {
	t.Helper()

	dir := filepath.Join("testdata", filepath.FromSlash(name))
	if len(argv) == 0 {
		argv = readArgv(t, filepath.Join(dir, "argv"))
	}
	return goldenCase{dir: dir, name: name, argv: argv}
}

// walkTestdata walks the corpora for files of one name and hands each one's
// directory to visit, in walk order. Both of the questions asked of testdata/
// are this walk — which directories are cases, and which hold checked-in
// golden files — and neither knows a corpus by name.
func walkTestdata(t *testing.T, filename string, visit func(dir string)) {
	t.Helper()

	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == filename {
			visit(filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestGolden drives every golden case end to end through cli.Main — the one
// entry point, taking the complete argv, deciding for itself which command runs
// (issue #107). A case supplies:
//
//   - argv, required: the complete command line as typed, `hyper check --json`.
//   - repo/, optional: a fixture repository. A case that supplies one has
//     --repo-dir resolved to it and stands in it; a case that supplies none is
//     driven with its argv alone, from the case directory.
//   - env, optional: the environment the invocation is read against, one
//     NAME=value line per variable. A variable the file does not list is
//     absent, and a line whose value is empty is a variable set to the empty
//     string — two states a case must be able to tell apart, `targets`
//     reporting whether a credential's variable is present (issue #112).
//   - facts.json or version, optional: the build facts the entry point is
//     handed. facts.json is the whole value, for a case whose subject is the
//     page it renders; version is only the string the pin gate compares, for
//     the many cases that care about nothing else. Absent both, the version is
//     1.4.0.
//   - wd, optional: the working directory. It is relative to the case
//     directory, or — where the case materialises a repository, the fixture
//     then being a copy somewhere else entirely — relative to that copy's root.
//   - now, optional: an RFC 3339 instant, the clock the entry point is handed
//     and the date on every commit the fixture itself makes. Absent, the
//     harness's stated constant.
//   - mint, optional: the Run ids the process answers, one per line and in
//     order. A case whose command mints one and names none fails; that is the
//     axis, and it is what makes every Store path and every terminal line a
//     Run writes a checked-in constant (issue #136).
//   - actor and hostname, optional: who is running hyper and on which machine,
//     which a Journal entry's Trigger carries — `actor` on both executors and
//     `host` on `local`. Absent, the harness's stated constants.
//   - git, store/, store-unpushed/, remote, remote-store/, remote-ahead/,
//     reject-pushes, unfetchable-remote, find-root, no-git-root, optional: the
//     git fixture, which golden_fixture_test.go states in full. A case
//     supplying none of them is driven exactly as it was before issue #125 —
//     same directory, same argv — which is every case that landed before it.
//   - bin/, optional: the executables a `shell` Step's argv may reach. A case's
//     argv head resolves against it and against nothing else, so an exit code,
//     a stdout and a stderr are the fixture's rather than the machine's; a case
//     with no bin/, or one naming a binary its bin/ does not hold, reaches no
//     binary at all.
//   - serve/, optional: what the world answers, one `<host>.json` per host —
//     a status, headers and a body, or the host that accepts the connection and
//     answers nothing at all. The harness stands one in-process TLS
//     server, mints its certificate against the case's `now`, and hands the
//     entry point a dialer that maps every served hostname to it; a host with
//     no entry has its connection refused. golden_serve_test.go states it in
//     full.
//
// Its stdout.golden, stderr.golden and exit.golden are compared byte for byte,
// and regenerated in place behind -update; so are store.golden and
// remote.golden, where the case supplies them.
func TestGolden(t *testing.T) {
	for _, c := range goldenCases(t) {
		t.Run(c.name, func(t *testing.T) {
			run := c.invocation(t)

			var stdout, stderr bytes.Buffer
			exit := cli.Main(run.args, &stdout, &stderr, c.process(t, run), c.facts(t))

			compareGolden(t, c.dir, stdout.Bytes(), stderr.Bytes(), exit)
			run.compareBranches(t, c.dir)
		})
	}
}

// process is what the harness hands the entry point in the process's place: the
// case's own environment, the working directory its invocation was resolved
// against, the instant its clock answers with, and the dialer its serve/
// directory stands. The four are the case's `env`, `wd`, `now` and `serve/`
// inputs and nothing else, which is what makes a golden a statement about the
// command rather than about the machine the suite ran on (issues #134, #135).
//
// All three of the threaded reads have now been claimed, each opted into by a
// file a case writes: the dialer by `probe` and `run` through a serve/ entry,
// the mint by `run` through a mint line, and the launcher by a `shell` Step
// through a bin/ directory (issue #142). The launcher is the real one and the
// opt-in is that directory rather than a stand-in — what a case may exec is
// what its bin/ holds, so a case that names a binary it does not hold drives
// *the command could not be started at all* and a case that names nothing execs
// nothing.
//
// The tenth read is left nil, and that is what a case directory can say about
// signals: a corpus case is a Run **nobody interrupts**. A signal is a fact
// about *when* it arrived, which no file beside an argv can state, so the
// delivery is driven by [run_signal_test.go](run_signal_test.go) — the same
// cases, the same entry point, with this one member supplied there (issue
// #145).
func (c goldenCase) process(t *testing.T, run goldenRun) cli.Process {
	t.Helper()

	instant := c.instant(t)
	env := c.variables(t)
	return cli.Process{
		LookupEnv: c.environment(env),
		Environ:   c.environ(env),
		Getwd:     func() (string, error) { return run.wd, nil },
		User:      func() (string, error) { return c.actor(t), nil },
		Hostname:  func() (string, error) { return c.hostname(t), nil },
		Now:       func() time.Time { return instant },
		Mint:      c.mint(t),
		Dial:      c.dialer(t, instant),
		// The real launcher, with the fixture supplying name
		// resolution: a case that started its child through a stand-in
		// would leave the process group and the SIGKILL unchecked, and
		// what a golden asserted would be the harness's account of a
		// command rather than a command's (issue #142).
		Exec: c.launcher(t),
	}
}

// fixtureActor is who a case says is running hyper where it names nobody, and
// fixtureHostname the machine they are on. Both are stated constants rather
// than the suite's own for the reason every other read of the process here is
// one: a Journal entry carries both on the `local` executor (§7), so an entry
// built from the account and machine the suite ran on is a store.golden nobody
// can check in.
const fixtureActor = "igor"

const fixtureHostname = "thinkpad"

// actor is who the case is driven by: what its actor file names, or the stated
// constant.
func (c goldenCase) actor(t *testing.T) string {
	t.Helper()

	if named := strings.TrimSpace(readFile(t, filepath.Join(c.dir, "actor"))); named != "" {
		return named
	}
	return fixtureActor
}

// hostname is the machine the case is driven on: what its hostname file names,
// or the stated constant.
func (c goldenCase) hostname(t *testing.T) string {
	t.Helper()

	if named := strings.TrimSpace(readFile(t, filepath.Join(c.dir, "hostname"))); named != "" {
		return named
	}
	return fixtureHostname
}

// mint is the Run ids the case's process answers, in order: one per line of its
// mint file, and the last determinism axis the corpus has (issue #136).
//
// A Run id is minted from the clock and from crypto/rand, and it lands on the
// terminal line, in the `outcome` row, in `run.json` and in every Store path a
// Run writes. Threading it is what makes every one of those a checked-in
// constant — §8 states that a Run id renders **whole** (ADR-0047), and a corpus
// asserting `<run-id>` could not check the one rendering rule that surface has.
//
// A case that mints and lists nothing, or mints more ids than it listed, fails
// rather than being handed something: the whole point is that the case says
// which ids its Run has, and a harness that invented one would put a value
// nobody wrote into a golden.
func (c goldenCase) mint(t *testing.T) func(time.Time) store.RunID {
	t.Helper()

	var ids []store.RunID
	for _, line := range strings.Split(readFile(t, filepath.Join(c.dir, "mint")), "\n") {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		id, err := store.ParseRunID(text)
		if err != nil {
			t.Fatalf("case %s: mint names %q: %v", c.name, text, err)
		}
		ids = append(ids, id)
	}

	minted := 0
	return func(time.Time) store.RunID {
		if minted >= len(ids) {
			t.Errorf("case %s minted %d Run ids and its mint file names %d; a case says which ids its Runs have", c.name, minted+1, len(ids))
			return store.RunID{}
		}
		minted++
		return ids[minted-1]
	}
}

// goldenRun is one case resolved into what it takes to drive it: the arguments
// the entry point receives, the working directory they are read against, and
// the git fixture the case asked for — which for every case that asks for none
// is the zero value, built by nothing and rendering nothing.
type goldenRun struct {
	args    []string
	wd      string
	inputs  fixtureInputs
	fixture gitFixture
}

// invocation resolves what the process would hand the entry point: the
// arguments, and the working directory they are read against — materialising
// the case's repository first, where it asked for one.
//
// --repo-dir is synthesised here rather than written into an argv because the
// fixture's path is only known at run time — but the rule is the case's and not
// the command's: whatever command the argv names, a repo/ beside it is the
// repository the invocation stands in, and that is what lets one driver run a
// check case and a providers case alike. It goes directly after the command
// name, which the dispatch reads first and which nothing may precede; the
// case's own arguments follow it, as they would on a real command line.
//
// A case that carries no repo/ is driven with its argv alone, which is how one
// repository is shared between cases: the case names it in its own argv, with
// the --repo-dir an operator would type, resolved against the case's own
// directory like every other path a case writes. The exemption corpus's three
// invocations share one that way, only the one that can reach a repository at
// all naming it (issue #105); the twelve cases against the five-artefact
// demonstration repository share testdata/five-artefact-demo/repo the same way,
// across five corpora, each of them naming it (issue #116).
//
// The two paths §9 states and no case could reach until now are here too, and
// both are the absence of that synthesis rather than an argument added to it: a
// case carrying find-root is driven with no --repo-dir at all, so the root is
// the one resolveRepoRoot walks up to, and a case carrying no-git-root is
// driven from a directory with no git root above it, which is the walk finding
// nothing (issue #125).
func (c goldenCase) invocation(t *testing.T) goldenRun {
	t.Helper()

	run := goldenRun{args: c.argv, wd: c.abs(t, "."), inputs: c.fixtureInputs()}
	if fault := run.inputs.fault(); fault != "" {
		t.Fatalf("case %s %s", c.name, fault)
	}

	// The repository the invocation stands in, wherever it turned out to be:
	// the copy, for a materialised case, and the checked-in repo/ for the
	// cases that have always been driven straight off the disk.
	root := ""
	switch {
	case run.inputs.materialised():
		run.fixture = c.materialise(t, run.inputs)
		root = run.fixture.root
	case run.inputs.repo:
		root = c.abs(t, "repo")
	}
	if root != "" {
		run.wd = root
		if !run.inputs.findRoot {
			run.args = append([]string{c.argv[0], "--repo-dir", root}, c.argv[1:]...)
		}
	}
	if run.inputs.noGitRoot {
		run.wd = outsideAnyRepository(t)
	}

	if w := readFile(t, filepath.Join(c.dir, "wd")); w != "" {
		named := strings.TrimSpace(w)
		// A materialised case's fixture is a copy in a temp directory, so
		// a path into it is relative to that copy's root; every other
		// case's wd is relative to its own directory, which is what
		// testdata/exemption's three ../repo mean.
		if run.inputs.materialised() {
			run.wd = filepath.Join(run.fixture.root, filepath.FromSlash(named))
		} else {
			run.wd = c.abs(t, named)
		}
	}
	return run
}

// compareBranches holds the case's two branch goldens against what the run
// left on the branch — store.golden the local refs/heads/hyper-store, and
// remote.golden the same ref on origin — or, under -update, rewrites the ones
// the case supplies from what just ran.
//
// A golden the case does not supply is not written, which is what leaves every
// landed case untouched by a -update run: the axis is opted into by the file
// being there, so a new case asserting a branch starts by creating an empty one
// and regenerating it. A case supplying neither asserts nothing about any
// branch.
func (r goldenRun) compareBranches(t *testing.T, dir string) {
	t.Helper()

	if r.inputs.storeGolden {
		compareBranchGolden(t, filepath.Join(dir, "store.golden"), r.fixture.render(t, r.fixture.root))
	}
	if r.inputs.remoteGolden {
		compareBranchGolden(t, filepath.Join(dir, "remote.golden"), r.fixture.render(t, r.fixture.origin))
	}
}

// compareBranchGolden holds one rendered branch against its golden file, byte
// for byte, on compareGolden's own footing and behind the same one flag.
func compareBranchGolden(t *testing.T, path, rendered string) {
	t.Helper()

	if *update {
		writeGolden(t, path, []byte(rendered))
		return
	}
	if want := readFile(t, path); rendered != want {
		t.Errorf("%s mismatch:\n got:  %q\n want: %q", filepath.Base(path), rendered, want)
	}
}

// abs resolves a path named by a case against the case's own directory, which
// is what every path a case writes is relative to.
func (c goldenCase) abs(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.Abs(filepath.Join(c.dir, path))
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// variables is the environment the case is driven against, read from its own
// env file: one NAME=value line per variable, and everything past the first "="
// is the value, empty included. A case that supplies no env file is driven
// against an environment with nothing in it, which is what every case that
// names no variable means.
//
// It is the whole environment and not a starting point the harness adds to. A
// `shell` Operation's child inherits it less the repository's credential slots
// (§3, §11), so a case that prints a variable prints exactly what it wrote —
// which is what lets a golden hold *this one was withheld and that one was not*
// (issue #142). Nothing about where a binary is found comes through here: that
// is name resolution, and the harness supplies it beside the launcher below.
func (c goldenCase) variables(t *testing.T) map[string]string {
	t.Helper()

	env := map[string]string{}
	for _, line := range strings.Split(readFile(t, filepath.Join(c.dir, "env")), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		name, value, named := strings.Cut(line, "=")
		if !named {
			t.Fatalf("env line %q names no value; a variable a case sets is written NAME=value, and one it leaves unset is a line it does not write", line)
		}
		env[name] = value
	}
	return env
}

// environment is that map read one name at a time, which is os.LookupEnv's
// shape rather than os.Getenv's: a variable set to the empty string and one
// that was never set are two different answers on `targets`'s presence column,
// and a corpus that could not state the difference could not hold the rule (§9,
// issue #112).
func (c goldenCase) environment(env map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, present := env[name]
		return value, present
	}
}

// environ is the same map read whole, which is what a `shell` Operation's child
// inherits less the repository's credential slots (§3, §11). It is sorted so
// that a child's environment is one value however the map iterated.
func (c goldenCase) environ(env map[string]string) func() []string {
	return func() []string {
		whole := make([]string, 0, len(env))
		for _, name := range slices.Sorted(maps.Keys(env)) {
			whole = append(whole, name+"="+env[name])
		}
		return whole
	}
}

// launcher is the real one with the fixture supplying name resolution, which is
// the arrangement the dialer already has one Capability over: a case's argv head
// resolves against the case's own bin/ directory and never against the machine's
// PATH, so what a command printed, what it exited with and whether it could be
// started at all are the fixture's facts (issue #142).
//
// **The argv itself is untouched**, which is what makes a golden readable: the
// Record a `shell` Step writes is named by the argv as run (§12), and a harness
// that rewrote the head into an absolute path under a temp directory would put
// a value nobody can check in into every store.golden. os/exec keeps the two
// apart already — Path is the file that is executed and Args[0] is the word the
// child is told it was invoked as — so this sets the first and leaves the
// second.
//
// A case with no bin/, or whose argv names a binary its bin/ does not hold,
// resolves to a path that is not there and the child cannot be started at all,
// which is §12's one-member response object driven rather than described.
func (c goldenCase) launcher(t *testing.T) capability.Exec {
	t.Helper()

	bin := c.abs(t, "bin")
	return func(ctx context.Context, argv []string) *exec.Cmd {
		child := cli.Child(ctx, argv)
		child.Path = filepath.Join(bin, argv[0])
		// LookPath's answer, discarded: what this case may exec is its
		// bin/ and the machine's PATH has no say in it, including when
		// it happens to hold a binary of the same name.
		child.Err = nil
		return child
	}
}

// defaultVersion is the version a case is driven with where it names none: the
// pin every fixture repository that means to match writes in its hyper.yaml.
const defaultVersion = "1.4.0"

// facts reads the build facts a case hands the entry point, from facts.json or
// from version, which are two spellings of one input and never both: a case
// supplying each would be stating the binary's version twice, and the day they
// disagreed the golden files would hold whichever the harness happened to
// prefer.
func (c goldenCase) facts(t *testing.T) version.Facts {
	t.Helper()

	path := filepath.Join(c.dir, "facts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		v := defaultVersion
		if named := readFile(t, filepath.Join(c.dir, "version")); named != "" {
			v = strings.TrimSpace(named)
		}
		return version.Facts{Version: v}
	}
	if readFile(t, filepath.Join(c.dir, "version")) != "" {
		t.Fatal("the case supplies both facts.json and version; the binary's version is stated in one of them or the other")
	}

	// Unknown fields are an error: a fixture with a misspelt key would
	// otherwise render `unknown` and be frozen into a golden file as the very
	// rendering it meant to avoid.
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var fixture factsFixture
	if err := dec.Decode(&fixture); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return version.Facts{
		Version:   fixture.Version,
		Commit:    fixture.Commit,
		Built:     fixture.Built,
		Modified:  fixture.Modified,
		Toolchain: fixture.Toolchain,
		OS:        fixture.OS,
		Arch:      fixture.Arch,
	}
}

// factsFixture is the on-disk shape of a case's facts.json, stated here rather
// than by hanging json tags off version.Facts: the fixture format is the
// harness's business, and a domain value should not carry a serialisation it
// has no other use for. The tags are what make the file's keys a contract
// instead of a coincidence of Go's case-insensitive field matching.
type factsFixture struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Built     string `json:"built"`
	Modified  bool   `json:"modified"`
	Toolchain string `json:"toolchain"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// readArgv reads a case's complete argv — `hyper <command>` and whatever
// follows — and returns what the entry point receives, which is everything
// past the program name. Storing the whole line rather than only the tail is
// what makes a case directory readable as the invocation it stands for; the
// token it always starts with is asserted rather than assumed, and the command
// that follows is now the dispatch's own input rather than something the
// harness pins the case against.
//
// Tokens are whitespace-separated, so no case can express an argument that
// carries whitespace of its own. The day a command needs such a case is the day
// this file's grammar changes; nothing in the corpora needs it yet, and a case
// is free to write one token per line in the meantime.
func readArgv(t *testing.T, path string) []string {
	t.Helper()
	argv := strings.Fields(readFile(t, path))
	if len(argv) < 2 || argv[0] != "hyper" {
		t.Fatalf("argv is %q, want a complete argv beginning `hyper <command>`", argv)
	}
	return argv[1:]
}

// compareGolden holds one case's outcome against its stdout.golden,
// stderr.golden and exit.golden, byte for byte on the two streams and by value
// on the code — or, under -update, rewrites all three from what just ran.
// Every case is compared the same way, so no command can quietly hold itself
// to a looser reading of its own golden files than its neighbour does.
func compareGolden(t *testing.T, dir string, stdout, stderr []byte, exit int) {
	t.Helper()

	if *update {
		writeGolden(t, filepath.Join(dir, "stdout.golden"), stdout)
		writeGolden(t, filepath.Join(dir, "stderr.golden"), stderr)
		writeGolden(t, filepath.Join(dir, "exit.golden"), []byte(strconv.Itoa(exit)+"\n"))
		return
	}

	wantExit, err := strconv.Atoi(strings.TrimSpace(readFile(t, filepath.Join(dir, "exit.golden"))))
	if err != nil {
		t.Fatalf("exit.golden: %v", err)
	}

	if got, want := string(stdout), readFile(t, filepath.Join(dir, "stdout.golden")); got != want {
		t.Errorf("stdout mismatch:\n got:  %q\n want: %q", got, want)
	}
	if got, want := string(stderr), readFile(t, filepath.Join(dir, "stderr.golden")); got != want {
		t.Errorf("stderr mismatch:\n got:  %q\n want: %q", got, want)
	}
	if exit != wantExit {
		t.Errorf("exit = %d, want %d", exit, wantExit)
	}
}

// TestMain holds the criterion no single case can: nothing a test run does
// reaches the checked-in testdata/ tree. Every fixture golden_fixture_test.go
// builds lives under t.TempDir() — the copy, its .git, the bare origin, git's
// own configuration — and the way to know that stayed true is to weigh the
// corpora before the suite and after it.
//
// It sits here rather than beside the fixture it is watching, because it is the
// package's one entry point and governs every test in it, this file's corpus
// assertions included.
//
// Under -update it does not, because rewriting golden files is exactly what
// that flag is for.
func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		os.Exit(m.Run())
	}

	before := testdataDigest()
	code := m.Run()
	if after := testdataDigest(); after != before {
		fmt.Fprintln(os.Stderr, "testdata/ moved during the run; a fixture is being built inside the checked-in corpora rather than in a temp directory")
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

// testdataDigest is one digest over every file under testdata/ — each path and
// then its bytes — so that a file added, removed, rewritten or renamed moves it.
func testdataDigest() string {
	sum := sha256.New()
	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Fprintf(sum, "%s\n", filepath.ToSlash(path))
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum.Write(data)
		return nil
	})
	if err != nil {
		return "unreadable: " + err.Error()
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

// TestGoldenCorpora_NoCaseCarriesACheckedInGitDirectory is the reason the git
// fixture is built at run time, standing where the workaround used to: a case's
// repository is materialised into a temp directory, so a .git under testdata/ is
// either a fixture built in the wrong place or one committed by hand — and git
// will not carry a nested repository's contents anyway, so the second would
// arrive at a fresh clone as something else entirely (issue #125).
func TestGoldenCorpora_NoCaseCarriesACheckedInGitDirectory(t *testing.T) {
	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == ".git" {
			t.Errorf("%s is checked in; a case's repository is materialised at run time, in a temp directory", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestGoldenCorpora_EveryGoldenTripleIsDrivenBySomething is the fence the four
// harnesses used to get from owning their own corpora: each knew its cases by
// name, so a corpus nothing drove was a corpus nobody had written. With one
// harness reading its cases off the disk, a case whose argv went missing would
// simply stop being run, and its golden files would sit there green and
// unexercised. So the walk runs the other way — every directory holding an
// exit.golden must be one the harness enumerated (issue #108).
//
// It is held against goldenCases rather than against an argv file beside the
// golden files, because what the criterion asks is that the harness drives the
// case, and the enumeration is the only thing that answers that.
func TestGoldenCorpora_EveryGoldenTripleIsDrivenBySomething(t *testing.T) {
	driven := make(map[string]bool)
	for _, c := range goldenCases(t) {
		driven[c.dir] = true
	}

	forEachGoldenTriple(t, func(dir string, exit int) {
		if !driven[dir] {
			t.Errorf("%s holds golden files and is not a case the harness drives; it needs an argv", dir)
		}
	})
}

// TestGoldenCorpora_ACasesDirectorySaysWhichCommandItExercises pins the
// convention the per-command subtree exists for (issue #101, issue #108): a
// case sits in the directory named for the command its argv invokes, or one
// directly beneath it. testdata/check/clean/ is the ordinary shape; the
// exemption corpus's check/, version/ and completions/ are the other, three
// cases named for their commands and sharing one repository between them.
//
// The harness itself needs none of this — it reads the command out of the argv
// like the dispatch does — which is exactly why the convention needs a test: it
// is a rule for readers, and nothing else would notice it lapsing.
func TestGoldenCorpora_ACasesDirectorySaysWhichCommandItExercises(t *testing.T) {
	for _, c := range goldenCases(t) {
		names := corpusNames(c.argv[0])
		if !slices.Contains(names, filepath.Base(c.dir)) && !slices.Contains(names, filepath.Base(filepath.Dir(c.dir))) {
			t.Errorf("case %s exercises %q; want it in a directory named one of %q, or one directly beneath one", c.name, c.argv[0], names)
		}
	}
}

// corpusNames is what a command's corpus may be called: its own name, and — for
// the one command in §9's tree that has a sub-verb — the noun and the verb
// hyphenated.
//
// The second form exists because a noun-grouped command's name is two words. A
// caller types `hyper store init`, so `store-init` is what that command is
// called, and a corpus filed under the bare noun would be the right name only
// while `init` is the only verb (§9, issue #126). The bare noun stays admissible
// for the same reason the group exists: a case about the group's own grammar —
// `hyper store` with no verb at all — belongs beside the verb's cases rather
// than in a corpus of its own.
func corpusNames(command string) []string {
	names := []string{command}
	if command == "store" {
		for _, verb := range cli.StoreSubVerbs() {
			names = append(names, command+"-"+verb)
		}
	}
	return names
}

// TestGoldenCorpora_StdoutCarriesNothingButTheAnswer is §9's stream discipline
// asserted over every corpus at once: **stdout is the answer, and nothing else
// ever goes there.**
//
// What counts as the answer is the exit code's own question, and §9 gives it two
// arms. A command that is **not a Run** answers on stdout where it did what it
// was asked (`0`) or reported problems it found (`1`), and is silent otherwise:
// a usage error opens no row stream at all, and the Refusals such a command
// makes — the pin gate's, `store init`'s absent Store — render on stderr
// (ADR-0060, issue #105).
//
// A **Run answers on stdout at every exit on which a Run was attempted**,
// because a Run's answer *is* its outcome: §8's Step table, its Refusal
// rendering and its terminal line, terminated on the wire by the `outcome` row,
// which §9 says `run` is on "on every path on which a Run was attempted". That
// is what puts a refused Run's id on a job summary exactly as a completed one's
// (§8, §9, issue #137). The one path where a Run is silent is the one every
// other command is silent on: a usage error, where no Run was attempted at all.
//
// So the invariant is stated as *what may stand there* rather than as *which
// codes may write*: a non-clean exit writes nothing, or writes an answer that
// ends in §8's terminal line.
//
// This is the one property that spans every command the tool will ever have, so
// it is asserted from the checked-in golden files rather than by driving
// anything: a corpus that lands in a later milestone is found by the shape of
// its case directories — an exit.golden, and whatever stdout.golden lies beside
// it — and has to tell this test nothing about how its own command is driven or
// where its inputs live. Written now over six cases it is one walk; written in
// milestone 8 over two hundred it is an archaeology exercise, and by then
// something will already have violated it.
func TestGoldenCorpora_StdoutCarriesNothingButTheAnswer(t *testing.T) {
	var judged int

	forEachGoldenTriple(t, func(dir string, exit int) {
		if exit == cli.ExitClean || exit == cli.ExitProblems {
			return
		}

		judged++
		stdout := readFile(t, filepath.Join(dir, "stdout.golden"))
		if stdout == "" || endsInAnOutcome(stdout) {
			return
		}
		t.Errorf("%s exits %d and wrote %q to stdout; past exit %d the only answer stdout carries is a Run's outcome",
			dir, exit, stdout, cli.ExitProblems)
	})

	// A corpus of nothing but clean runs and problem reports would hold this
	// invariant without ever reaching it. The fence is that the walk found the
	// exits it is here to judge, not a count of them: cases are added freely,
	// and a number here would be a registration by another name.
	if judged == 0 {
		t.Fatal("no case in any corpus exits non-zero and not 1; the invariant held vacuously")
	}
}

// endsInAnOutcome says whether an answer ends the way a Run's does, in either
// mode: §8's terminal line, or the `outcome` row that terminates its stream.
//
// **The last row is always the terminal row, and its absence means the stream
// was cut off** (§9) — so the two forms are one question asked of two
// renderings, and asking it is how this file tells a Run's answer from a
// diagnostic that leaked onto stdout.
//
// The line is read by the outcome it opens with rather than matched whole,
// because what follows differs by path: the rehearsal marker, and the
// remediation pointer a Refusal may carry in place of the bare Run id.
func endsInAnOutcome(answer string) bool {
	lines := strings.Split(strings.TrimSuffix(answer, "\n"), "\n")
	last := lines[len(lines)-1]

	var row struct {
		Type string `json:"type"`
	}
	if json.Unmarshal([]byte(last), &row) == nil {
		return row.Type == "outcome"
	}
	for _, outcome := range []store.Outcome{store.OutcomeCompleted, store.OutcomeRefused, store.OutcomeFailed} {
		if strings.HasPrefix(last, string(outcome)+" · ") {
			return true
		}
	}
	return false
}

// forEachGoldenTriple walks testdata/ for checked-in golden files and hands
// each directory that holds one to visit, along with the exit it recorded. It
// reads the corpora rather than driving them, so a case is found by the shape
// of its directory and tells these assertions nothing about its own command.
func forEachGoldenTriple(t *testing.T, visit func(dir string, exit int)) {
	t.Helper()

	walkTestdata(t, "exit.golden", func(dir string) {
		path := filepath.Join(dir, "exit.golden")
		exit, err := strconv.Atoi(strings.TrimSpace(readFile(t, path)))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return
		}
		visit(dir, exit)
	})
}

// isDir says whether a case supplied an optional directory input, an absent one
// being the ordinary reading rather than an error.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// readFile reads a case's input file, treating an absent one as empty: most
// case inputs are optional, and their default is the empty reading.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatal(err)
	}
	return string(data)
}

func writeGolden(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestGoldenCorpora_EveryFlagCitesALineTheGutterMarked is the relation `FLAGS`
// rests on, held over every checked-in stream at once: a flag cites a line the
// gutter already marked, and introduces no claim of its own (§8, §12,
// ADR-0026).
//
// It is asserted here rather than per case, and that is the point of putting
// `cites_line` on every row: a flag citing a line no `gutter` row marked is
// mechanically detectable across the whole corpus, where a per-case eyeball
// would find a breach only where somebody happened to look. A name added to the
// vocabulary, or a roster pointed at the wrong line, fails here without anyone
// having written a case for it.
//
// Like the two assertions beside it, it reads the golden files rather than
// driving anything: a stream is found by its case's argv carrying --json, and
// the streams that carry no flag row are the commands that render no flags.
func TestGoldenCorpora_EveryFlagCitesALineTheGutterMarked(t *testing.T) {
	var judged int
	for _, c := range goldenCases(t) {
		if !slices.Contains(c.argv, "--json") {
			continue
		}
		stdout := readFile(t, filepath.Join(c.dir, "stdout.golden"))
		if stdout == "" {
			continue
		}

		marked := map[int]bool{}
		var flags []struct {
			flag string
			line int
		}
		for i, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
			var row struct {
				Type      string `json:"type"`
				Line      int    `json:"line"`
				Flag      string `json:"flag"`
				CitesLine int    `json:"cites_line"`
			}
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				t.Errorf("%s: line %d is not one JSON object: %v", c.name, i+1, err)
				continue
			}
			switch row.Type {
			case "gutter":
				marked[row.Line] = true
			case "flag":
				flags = append(flags, struct {
					flag string
					line int
				}{row.Flag, row.CitesLine})
			}
		}

		judged += len(flags)
		for _, f := range flags {
			if !marked[f.line] {
				t.Errorf("%s: the %s flag cites line %d, which no gutter row in the same stream marked", c.name, f.flag, f.line)
			}
		}
	}

	// A corpus with no flag row in any stream would hold this vacuously. The
	// fence is that the walk found rows to judge, not how many: cases are
	// added freely, and a number here would be a registration by another
	// name.
	if judged == 0 {
		t.Fatal("no --json case in any corpus carries a flag row; the relation held vacuously")
	}
}

// TestGoldenCorpora_AJSONStreamIsTypedRowsEndingInItsTerminalRow is §8's two
// rules about the wire, asserted over every corpus at once: every row opens
// with the type a consumer discriminates on, and the last row is the terminal
// row, whose absence is what says the stream was cut off.
//
// It is the corpus that holds them rather than the renderer, and deliberately
// (issue #110). Type-first is a property of how a row type declares itself, and
// a writer that stopped mid-stream to report a row declared wrong would leave
// the wire unterminated — the very thing the second rule says a consumer must
// not trust. Held here, a row type declared wrong is a fixed cost paid once by
// whoever wrote it, and nothing on the wire is cut off to say so.
//
// Like the stream-discipline test above it, it reads the checked-in golden
// files rather than driving anything: a case is found by its argv carrying
// --json and by whatever stdout.golden lies beside it, and tells this test
// nothing about its own command. That is what makes it the assertion the row
// stream needs — the invariant decays silently as row types are added, and a
// command that added one and got either rule wrong would otherwise have a green
// corpus.
//
// A case that wrote nothing to stdout opened no stream at all, which is what a
// usage error and a Refusal do (§9, ADR-0060): there is no terminal row to be
// missing where no row was written.
func TestGoldenCorpora_AJSONStreamIsTypedRowsEndingInItsTerminalRow(t *testing.T) {
	// The two terminal types, and the type is itself the discriminator (§8):
	// a Run emits outcome and everything else emits result. They are named
	// from the spec rather than read off the renderer, a fence that took its
	// expectation from the code it fences being one that cannot disagree with
	// it.
	terminal := map[string]bool{"result": true, "outcome": true}

	var judged int
	for _, c := range goldenCases(t) {
		if !slices.Contains(c.argv, "--json") {
			continue
		}
		stdout := readFile(t, filepath.Join(c.dir, "stdout.golden"))
		if stdout == "" {
			continue
		}

		judged++
		var last string
		for i, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
			var row struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				t.Errorf("%s: line %d is not one JSON object: %v", c.name, i+1, err)
				continue
			}
			if !strings.HasPrefix(line, `{"type":`) {
				t.Errorf("%s: line %d opens %.16q; every row carries its type as its first key, which is what a consumer discriminates on", c.name, i+1, line)
			}
			last = row.Type
		}
		if !terminal[last] {
			t.Errorf("%s: the stream ends in a %q row; a stream ends in its terminal row, and its absence says the stream was cut off", c.name, last)
		}
	}

	// A corpus with no --json case at all would hold this vacuously. The
	// fence is that the walk found streams to judge, not how many: cases are
	// added freely, and a number here would be a registration by another name.
	if judged == 0 {
		t.Fatal("no case in any corpus wrote a --json stream; the invariant held vacuously")
	}
}

// corpusReportsFactsNotProblems holds one command's corpus against a rule three
// of the discovery commands share: they report facts rather than problems
// found, so exit 1 is unreachable from them however faulty the repository they
// read, and whatever name they were handed (ADR-0064).
//
// It reads the recorded exits rather than driving anything, on the same footing
// as the corpus-wide assertions above: a case is found by the shape of its
// directory, and a case added to any of the three corpora is judged by having
// been written. command is both the subtree the corpus sits in and the name the
// failure reads back, those being one string by the convention a case
// directory says which command it exercises (issue #101, issue #108).
func corpusReportsFactsNotProblems(t *testing.T, command string) {
	t.Helper()

	corpus := filepath.Join("testdata", command)
	cases, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatalf("the %s corpus is empty; the invariant would hold vacuously", command)
	}

	for _, entry := range cases {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(corpus, entry.Name(), "exit.golden")
		recorded, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if exit := strings.TrimSpace(string(recorded)); exit == strconv.Itoa(cli.ExitProblems) {
			t.Errorf("%s records exit %s; `hyper %s` reports facts, not problems found", path, exit, command)
		}
	}
}
