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

// update regenerates the golden files under testdata/ instead of checking
// against them (issue #88: "Golden files are checked in and regenerated
// behind an -update flag").
var update = flag.Bool("update", false, "regenerate golden files")

// TestCheckGolden drives cli.RunCheck end to end, one directory per case
// under testdata/, named for the error_code it produces where it produces
// one (issue #88). Each case supplies a fixture repository at repo/ and,
// optionally, args (one flag or path per line), version (the simulated
// binary version — defaults to 1.4.0), and wd (the working directory,
// relative to the case directory — defaults to repo/). Its golden files
// (stdout, stderr, exit) are compared byte-for-byte.
func TestCheckGolden(t *testing.T) {
	cases, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		name := c.Name()
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", name)
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
