package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// checkCorpus is the check command's slice of testdata/. Each command that
// owns a golden corpus gets its own subtree beside it — testdata/version/,
// testdata/completions/ — so that a case directory belongs to exactly one
// harness and no harness runs another's cases (issue #101). What every corpus
// shares is in golden_test.go.
const checkCorpus = "testdata/check"

// checkCases enumerates the check corpus's case directories. Sibling corpora
// live beside checkCorpus rather than in it, so they are outside what this can
// see.
func checkCases(t *testing.T) []string {
	t.Helper()
	return corpusCases(t, checkCorpus)
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

			compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)
		})
	}
}
