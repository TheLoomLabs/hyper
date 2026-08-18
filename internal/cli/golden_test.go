package cli_test

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// update regenerates the golden files of every corpus instead of checking
// against them (issue #88: "Golden files are checked in and regenerated behind
// an -update flag"). One flag serves them all — a corpus that regenerated
// behind a switch of its own would be one a `-update` run silently left stale.
var update = flag.Bool("update", false, "regenerate golden files")

// Each command that owns a golden corpus gets its own subtree under testdata/ —
// check/, version/, and completions/ when it lands — so that a case directory
// belongs to exactly one harness and no harness runs another's cases (issue
// #101). What the harnesses share is this file: how a case directory is found,
// and how its three golden files are compared or rewritten. What they do not
// share is how a case is driven, which is each command's own entry point and
// each corpus's own input files.
//
// A corpus is one harness's, then, but not necessarily one command's:
// exemption/ holds a single case whose whole subject is the contrast between
// three commands standing in one repository, and it could not be filed under
// any one of them without becoming the three unrelated cases it exists to not
// be (issue #105).

// readArgv reads a case's complete argv — `hyper <command>` and whatever
// follows — and returns what the entry point receives, which is everything
// past the command name. Storing the whole line rather than only the tail is
// what makes a case directory readable as the invocation it stands for; the
// two tokens it always starts with are asserted rather than assumed, so a case
// that meant to test another command cannot be run as this one's.
//
// Tokens are whitespace-separated, so no case can express an argument that
// carries whitespace of its own. That costs the corpora that use it nothing —
// `version` rejects every argument by length before looking at one, and
// `completions` matches its one positional against three words — and the day a
// command needs such a case is the day the file becomes one token per line, as
// testdata/check/'s args already is.
func readArgv(t *testing.T, path, command string) []string {
	t.Helper()
	argv := strings.Fields(readFile(t, path))
	if len(argv) < 2 || argv[0] != "hyper" || argv[1] != command {
		t.Fatalf("argv is %q, want a complete argv beginning `hyper %s`", argv, command)
	}
	return argv[2:]
}

// corpusCases enumerates a corpus's case directories, in ReadDir order.
// Sibling corpora live beside it rather than in it, so they are outside what
// any one harness can see.
func corpusCases(t *testing.T, corpus string) []string {
	t.Helper()
	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// compareGolden holds one case's outcome against its stdout.golden,
// stderr.golden and exit.golden, byte for byte on the two streams and by value
// on the code — or, under -update, rewrites all three from what just ran.
// Every corpus is compared the same way, so no command can quietly hold itself
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

	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "exit.golden" {
			return nil
		}
		dir := filepath.Dir(path)

		exit, err := strconv.Atoi(strings.TrimSpace(readFile(t, path)))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		if exit == cli.ExitClean || exit == cli.ExitProblems {
			return nil
		}

		judged++
		if stdout := readFile(t, filepath.Join(dir, "stdout.golden")); stdout != "" {
			t.Errorf("%s exits %d and wrote %q to stdout; only exit %d has an answer to write there",
				dir, exit, stdout, cli.ExitProblems)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A corpus of nothing but clean runs and problem reports would hold this
	// invariant without ever reaching it. The fence is that the walk found the
	// exits it is here to judge, not a count of them: cases are added freely,
	// and a number here would be a registration by another name.
	if judged == 0 {
		t.Fatal("no case in any corpus exits non-zero and not 1; the invariant held vacuously")
	}
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

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data := readFile(t, path)
	if data == "" {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(data, "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func writeGolden(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
