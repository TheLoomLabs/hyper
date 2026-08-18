package cli_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

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
//   - facts.json or version, optional: the build facts the entry point is
//     handed. facts.json is the whole value, for a case whose subject is the
//     page it renders; version is only the string the pin gate compares, for
//     the many cases that care about nothing else. Absent both, the version is
//     1.4.0.
//   - wd, optional: the working directory, relative to the case directory.
//
// Its stdout.golden, stderr.golden and exit.golden are compared byte for byte,
// and regenerated in place behind -update.
func TestGolden(t *testing.T) {
	for _, c := range goldenCases(t) {
		t.Run(c.name, func(t *testing.T) {
			args, wd := c.invocation(t)

			var stdout, stderr bytes.Buffer
			getenv := func(string) string { return "" }
			getwd := func() (string, error) { return wd, nil }
			exit := cli.Main(args, &stdout, &stderr, getenv, getwd, c.facts(t))

			compareGolden(t, c.dir, stdout.Bytes(), stderr.Bytes(), exit)
		})
	}
}

// invocation resolves what the process would hand the entry point: the
// arguments, and the working directory they are read against.
//
// --repo-dir is synthesised here rather than written into an argv because the
// fixture's path is only known at run time — but the rule is the case's and not
// the command's: whatever command the argv names, a repo/ beside it is the
// repository the invocation stands in, and that is what lets one driver run a
// check case and a providers case alike. It goes directly after the command
// name, which the dispatch reads first and which nothing may precede; the
// case's own arguments follow it, as they would on a real command line.
//
// A case that carries no repo/ is driven with its argv alone, which is how the
// exemption corpus's three invocations share one repository between them: the
// one that can reach a repository at all names it in its own argv, with the
// --repo-dir an operator would type.
func (c goldenCase) invocation(t *testing.T) (args []string, wd string) {
	t.Helper()

	wd = c.abs(t, ".")
	args = c.argv

	if repo := filepath.Join(c.dir, "repo"); isDir(repo) {
		wd = c.abs(t, "repo")
		args = append([]string{c.argv[0], "--repo-dir", wd}, c.argv[1:]...)
	}

	if w := readFile(t, filepath.Join(c.dir, "wd")); w != "" {
		wd = c.abs(t, strings.TrimSpace(w))
	}
	return args, wd
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
		command := c.argv[0]
		if filepath.Base(c.dir) != command && filepath.Base(filepath.Dir(c.dir)) != command {
			t.Errorf("case %s exercises %q; want it in a directory named %q, or one directly beneath it", c.name, command, command)
		}
	}
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
