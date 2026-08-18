package cli_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// update regenerates the golden files of the check corpus instead of
// checking against them (issue #88: "Golden files are checked in and
// regenerated behind an -update flag").
var update = flag.Bool("update", false, "regenerate golden files")

// checkCorpus is the check command's slice of testdata/. Each command that
// owns a golden corpus gets its own subtree beside it — testdata/version/,
// testdata/completions/ — so that a case directory belongs to exactly one
// harness and no harness runs another's cases (issue #101).
const checkCorpus = "testdata/check"

// checkCases enumerates the check corpus's case directories, in ReadDir
// order. Sibling corpora live beside checkCorpus rather than in it, so they
// are outside what this can see.
func checkCases(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(checkCorpus)
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

// TestCheckGolden_IgnoresSiblingCorpora pins the property the check corpus
// exists to give: a directory under testdata/ that is not check/ belongs to
// another command's harness, and the check harness never runs it as a case
// (issue #101). Were discovery rooted at testdata/ again, the version/ and
// completions/ corpora of the tickets that follow would be handed to
// RunCheck with a --repo-dir neither command takes.
//
// The probe is a real directory under the real testdata/, not a synthetic
// tree: a fence built from a fixture the harness cannot reach would stay
// green through exactly the regression it is here to catch.
func TestCheckGolden_IgnoresSiblingCorpora(t *testing.T) {
	probe, err := os.MkdirTemp("testdata", "sibling-corpus-probe-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(probe) })

	cases := checkCases(t)
	if len(cases) == 0 {
		t.Fatal("check corpus is empty; the probe would go unnoticed for the wrong reason")
	}
	sibling := filepath.Base(probe)
	for _, name := range cases {
		if name == sibling {
			t.Errorf("check corpus enumerated %s, which is a sibling corpus under testdata/, not a check case", name)
		}
	}
}

// TestCheckGolden drives cli.RunCheck end to end, one directory per case
// under testdata/check/, named for the error_code it produces where it
// produces one (issue #88). Each case supplies a fixture repository at repo/
// and, optionally, args (one flag or path per line), version (the simulated
// binary version — defaults to 1.4.0), and wd (the working directory,
// relative to the case directory — defaults to repo/). Its golden files
// (stdout, stderr, exit) are compared byte-for-byte.
func TestCheckGolden(t *testing.T) {
	for _, name := range checkCases(t) {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(checkCorpus, name)
			repoRoot, err := filepath.Abs(filepath.Join(dir, "repo"))
			if err != nil {
				t.Fatal(err)
			}

			args := []string{"--repo-dir", repoRoot}
			args = append(args, readLines(t, filepath.Join(dir, "args"))...)

			version := "1.4.0"
			if v := readFile(t, filepath.Join(dir, "version")); v != "" {
				version = strings.TrimSpace(v)
			}

			wd := repoRoot
			if w := readFile(t, filepath.Join(dir, "wd")); w != "" {
				wd, err = filepath.Abs(filepath.Join(dir, strings.TrimSpace(w)))
				if err != nil {
					t.Fatal(err)
				}
			}

			var stdout, stderr bytes.Buffer
			getenv := func(string) string { return "" }
			exit := cli.RunCheck(args, &stdout, &stderr, getenv, wd, version)

			if *update {
				writeGolden(t, filepath.Join(dir, "stdout.golden"), stdout.Bytes())
				writeGolden(t, filepath.Join(dir, "stderr.golden"), stderr.Bytes())
				writeGolden(t, filepath.Join(dir, "exit.golden"), []byte(strconv.Itoa(exit)+"\n"))
				return
			}

			wantStdout := readFile(t, filepath.Join(dir, "stdout.golden"))
			wantStderr := readFile(t, filepath.Join(dir, "stderr.golden"))
			wantExit, convErr := strconv.Atoi(strings.TrimSpace(readFile(t, filepath.Join(dir, "exit.golden"))))
			if convErr != nil {
				t.Fatalf("exit.golden: %v", convErr)
			}

			if stdout.String() != wantStdout {
				t.Errorf("stdout mismatch:\n got:  %q\n want: %q", stdout.String(), wantStdout)
			}
			if stderr.String() != wantStderr {
				t.Errorf("stderr mismatch:\n got:  %q\n want: %q", stderr.String(), wantStderr)
			}
			if exit != wantExit {
				t.Errorf("exit = %d, want %d", exit, wantExit)
			}
		})
	}
}

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
