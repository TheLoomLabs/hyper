package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
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
//   - git, store/, remote, remote-store/, find-root, no-git-root, optional:
//     the git fixture, which golden_fixture_test.go states in full. A case
//     supplying none of them is driven exactly as it was before issue #125 —
//     same directory, same argv — which is every case that landed before it.
//
// Its stdout.golden, stderr.golden and exit.golden are compared byte for byte,
// and regenerated in place behind -update; so are store.golden and
// remote.golden, where the case supplies them.
func TestGolden(t *testing.T) {
	for _, c := range goldenCases(t) {
		t.Run(c.name, func(t *testing.T) {
			run := c.invocation(t)

			var stdout, stderr bytes.Buffer
			getwd := func() (string, error) { return run.wd, nil }
			instant := c.instant(t)
			now := func() time.Time { return instant }
			exit := cli.Main(run.args, &stdout, &stderr, c.environment(t), getwd, now, c.facts(t))

			compareGolden(t, c.dir, stdout.Bytes(), stderr.Bytes(), exit)
			run.compareBranches(t, c.dir)
		})
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

// environment is the environment the case is driven against, read from its own
// env file: one NAME=value line per variable, and everything past the first "="
// is the value, empty included. A case that supplies no env file is driven
// against an environment with nothing in it, which is what every case that
// names no variable means.
//
// It answers presence as well as value, os.LookupEnv's shape rather than
// os.Getenv's, because a variable set to the empty string and one that was
// never set are two different answers on `targets`'s presence column and a
// corpus that could not state the difference could not hold the rule (§9,
// issue #112).
func (c goldenCase) environment(t *testing.T) func(string) (string, bool) {
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

	return func(name string) (string, bool) {
		value, present := env[name]
		return value, present
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
// asserted over every corpus at once: stdout is the answer and nothing else
// ever goes there, so an invocation that ended badly wrote nothing to it. Exit
// 1 is the sole exception, being a command that is not a Run reporting problems
// it found — which *is* the answer. A usage error opens no row stream at all,
// and a Refusal renders on stderr (§9, ADR-0060, issue #105).
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
		if stdout := readFile(t, filepath.Join(dir, "stdout.golden")); stdout != "" {
			t.Errorf("%s exits %d and wrote %q to stdout; only exit %d has an answer to write there",
				dir, exit, stdout, cli.ExitProblems)
		}
	})

	// A corpus of nothing but clean runs and problem reports would hold this
	// invariant without ever reaching it. The fence is that the walk found the
	// exits it is here to judge, not a count of them: cases are added freely,
	// and a number here would be a registration by another name.
	if judged == 0 {
		t.Fatal("no case in any corpus exits non-zero and not 1; the invariant held vacuously")
	}
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
