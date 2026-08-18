package cli_test

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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
