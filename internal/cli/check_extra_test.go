package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

const validHyperYAML = "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"

func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hyper.yaml"), validHyperYAML)
	return root
}

// emptyEnvironment is an environment with nothing in it: every variable
// answers absent, which is what a case that names no variable means. It is the
// shape os.LookupEnv has — the value, and whether the variable is set at all —
// because an empty value and an unset variable are two different answers
// wherever presence is the question (issue #112).
func emptyEnvironment(string) (string, bool) { return "", false }

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunCheck_HYPER_REPO_DIR_IsUsedWhenNoFlagGiven(t *testing.T) {
	root := newRepo(t)
	elsewhere := t.TempDir() // wd has no repository of its own

	lookupenv := func(k string) (string, bool) {
		if k == "HYPER_REPO_DIR" {
			return root, true
		}
		return "", false
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck(nil, cli.Streams(&stdout, &stderr), lookupenv, elsewhere, "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}
}

func TestRunCheck_RepoDirFlagOverridesEnv(t *testing.T) {
	root := newRepo(t) // the correct repo, pin matches
	wrongEnvRoot := t.TempDir()
	writeFile(t, filepath.Join(wrongEnvRoot, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	lookupenv := func(k string) (string, bool) {
		if k == "HYPER_REPO_DIR" {
			return wrongEnvRoot, true
		}
		return "", false
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root}, cli.Streams(&stdout, &stderr), lookupenv, t.TempDir(), "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d (flag should win over HYPER_REPO_DIR); stderr=%q", exit, cli.ExitClean, stderr.String())
	}
}

func TestRunCheck_NoColorFlagAndEnvProduceIdenticalBytes(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	lookupenv := emptyEnvironment

	var plain bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root}, cli.Streams(&plain, &plain), lookupenv, root, "1.4.0")

	var flagged bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root, "--no-color"}, cli.Streams(&flagged, &flagged), lookupenv, root, "1.4.0")

	lookupenvNoColor := func(k string) (string, bool) {
		if k == "NO_COLOR" {
			return "1", true
		}
		return "", false
	}
	var envd bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root}, cli.Streams(&envd, &envd), lookupenvNoColor, root, "1.4.0")

	if plain.String() != flagged.String() {
		t.Errorf("--no-color changed the bytes:\n plain: %q\nflagged: %q", plain.String(), flagged.String())
	}
	if plain.String() != envd.String() {
		t.Errorf("NO_COLOR changed the bytes:\n plain: %q\n envd: %q", plain.String(), envd.String())
	}
}

func TestRunCheck_MultiplePathsOneMissingExitsTwoAndReportsNothing(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	lookupenv := emptyEnvironment

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "definitions/broken.yaml", "definitions/typo.yaml"}, cli.Streams(&stdout, &stderr), lookupenv, root, "1.4.0")

	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Errorf("stderr is empty, want a usage error naming the missing path")
	}
}

func TestRunCheck_DirectoryPathFiltersToThatDirectory(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	writeFile(t, filepath.Join(root, "procedures", "broken.yaml"), "kind: procedure\nbase: &c\n  x: 1\ntargets: *c\n")
	lookupenv := emptyEnvironment

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json", "definitions"}, cli.Streams(&stdout, &stderr), lookupenv, root, "1.4.0")

	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("procedures/broken.yaml")) {
		t.Errorf("stdout = %q, want no rows from outside the named directory", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("definitions/broken.yaml")) {
		t.Errorf("stdout = %q, want a row from the named directory", stdout.String())
	}
}

func TestRunCheck_UnreadableArtefactDoesNotAbortThePass(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission bits are not enforced when running as root")
	}
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "healthy.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	unreadable := filepath.Join(root, "definitions", "locked.yaml")
	writeFile(t, unreadable, "kind: definition\n")
	if err := os.Chmod(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(unreadable, 0o644) })

	lookupenv := emptyEnvironment
	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json"}, cli.Streams(&stdout, &stderr), lookupenv, root, "1.4.0")

	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("definitions/healthy.yaml")) {
		t.Errorf("stdout = %q, want the healthy artefact's problems still reported", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("definitions/locked.yaml")) {
		t.Errorf("stdout = %q, want a problem row for the unreadable artefact", stdout.String())
	}
}

// The three arms of the path positional a golden case cannot hold: an absolute
// path, which no checked-in argv can spell; the empty string, which strings.
// Fields cannot carry into an argv; and the repository root itself, which is
// the one path that names every problem there is (ADR-0089).

func TestRunCheck_AnAbsolutePathInsideTheRepositoryFilters(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	writeFile(t, filepath.Join(root, "procedures", "broken.yaml"), "kind: procedure\nbase: &c\n  x: 1\ntargets: *c\n")

	var stdout, stderr bytes.Buffer
	named := filepath.Join(root, "definitions", "broken.yaml")
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json", named}, cli.Streams(&stdout, &stderr), emptyEnvironment, t.TempDir(), "1.4.0")

	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("definitions/broken.yaml")) {
		t.Errorf("stdout = %q, want the named artefact's rows", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("procedures/broken.yaml")) {
		t.Errorf("stdout = %q, want no rows from outside the path named", stdout.String())
	}
}

func TestRunCheck_AnAbsolutePathOutsideTheRepositoryExitsTwo(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	elsewhere := t.TempDir()
	named := filepath.Join(elsewhere, "definitions", "broken.yaml")
	writeFile(t, named, "kind: definition\n")

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, named}, cli.Streams(&stdout, &stderr), emptyEnvironment, elsewhere, "1.4.0")

	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d; stdout=%q", exit, cli.ExitUsage, stdout.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty; a repository with problems must not report clean", stdout.String())
	}
	if want := "hyper check: " + named + ": outside the repository\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunCheck_TheEmptyPathNamesNothing(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, ""}, cli.Streams(&stdout, &stderr), emptyEnvironment, root, "1.4.0")

	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d; stdout=%q", exit, cli.ExitUsage, stdout.String())
	}
	if want := "hyper check: the empty string names no path\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunCheck_TheRepositoryRootAsAPathReportsEveryProblem(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	writeFile(t, filepath.Join(root, "procedures", "broken.yaml"), "kind: procedure\nbase: &c\n  x: 1\ntargets: *c\n")

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json", "."}, cli.Streams(&stdout, &stderr), emptyEnvironment, root, "1.4.0")

	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitProblems, stderr.String())
	}
	for _, want := range []string{"definitions/broken.yaml", "procedures/broken.yaml"} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Errorf("stdout = %q, want %s among its rows; the root names every problem there is", stdout.String(), want)
		}
	}
}
