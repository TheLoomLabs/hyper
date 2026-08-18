package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// exemptionCorpus holds milestone 0's thesis in one fixture: a repository
// pinning a version the binary is not, and the three invocations that stand in
// it. It is one case directory with one repo/ and one facts.json rather than
// three sibling cases, because the claim is about one repository at one moment —
// three cases that merely happened to share a pin would not state it (issue
// #105). Each invocation keeps a golden triple of its own beneath it, so the
// contrast is read as a diff and so every corpus-wide assertion sees these
// three like any other case.
const exemptionCorpus = "testdata/exemption"

// TestExemption_OneRepositoryTwoAnswers is §9's exemption paragraph as a
// fixture (ADR-0020). The repository pins 9.9.9 and the binary is not 9.9.9:
// `hyper check` Refuses `version-pin-mismatch` at 77 with stdout silent, while
// `hyper version` and `hyper completions bash` — in that same repository, in
// that same working directory — exit 0 with their full output on stdout and
// nothing on stderr.
//
// That the gate Refuses is already proven six times over in testdata/check/.
// What is proven only here is the contrast: three invocations, one repository,
// one working directory, and the difference between them is the exemption and
// nothing else. The two exempt commands are handed no working directory and no
// environment — their signatures take neither — so the exemption is not a branch
// they take but a repository they cannot reach.
func TestExemption_OneRepositoryTwoAnswers(t *testing.T) {
	caseDir, err := filepath.Abs(exemptionCorpus)
	if err != nil {
		t.Fatal(err)
	}

	// One binary for all three invocations: the facts `version` renders carry
	// the version the gate compares against the pin, so the page's first line
	// and the Refusal's *this binary is* are two readings of one constant
	// (§11, issue #103).
	facts := readFacts(t, filepath.Join(caseDir, "facts.json"))
	repo := standIn(t, filepath.Join(caseDir, "repo"))

	t.Run("check refuses the repository it is standing in", func(t *testing.T) {
		dir := filepath.Join(caseDir, "check")

		var stdout, stderr bytes.Buffer
		exit := cli.RunCheck(readArgv(t, filepath.Join(dir, "argv"), "check"),
			&stdout, &stderr, func(string) string { return "" }, repo, facts.Version)

		compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)

		// What the golden file holds byte for byte, said out loud: the
		// Refusal names both versions and both remedies, so an operator who
		// hits it learns which binary refused, what the repository wanted,
		// and the two ways out — running `hyper project` or installing the
		// pinned version (§9, ADR-0020).
		for _, want := range []string{facts.Version, "9.9.9", "hyper project", "install 9.9.9"} {
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("the Refusal is %q, want it to name %q", stderr.String(), want)
			}
		}
	})

	t.Run("version answers in the same repository", func(t *testing.T) {
		dir := filepath.Join(caseDir, "version")

		var stdout, stderr bytes.Buffer
		exit := cli.RunVersion(readArgv(t, filepath.Join(dir, "argv"), "version"), &stdout, &stderr, facts)

		compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)
	})

	t.Run("completions answers in the same repository", func(t *testing.T) {
		dir := filepath.Join(caseDir, "completions")

		var stdout, stderr bytes.Buffer
		exit := cli.RunCompletions(readArgv(t, filepath.Join(dir, "argv"), "completions"), &stdout, &stderr)

		compareGolden(t, dir, stdout.Bytes(), stderr.Bytes(), exit)
	})
}

// standIn puts the process inside a copy of a fixture repository and returns
// where it now stands, so that all three invocations are handed nothing about
// their surroundings but the working directory itself — no --repo-dir, no
// HYPER_REPO_DIR, nothing a harness could point at one repository while the
// caller stood in another.
//
// The fixture is copied out of this tree rather than stood in where it lies,
// and the copy is given a .git, because repository-root resolution walks up
// from the working directory looking for one. A checked-in fixture cannot carry
// a .git of its own, so a process standing in testdata/exemption/repo would
// climb straight past it and resolve hyper's own repository — and the case would
// prove something about this tree rather than about the pin.
func standIn(t *testing.T, fixture string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}
