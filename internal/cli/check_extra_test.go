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

	getenv := func(k string) string {
		if k == "HYPER_REPO_DIR" {
			return root
		}
		return ""
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck(nil, &stdout, &stderr, getenv, elsewhere, "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}
}

func TestRunCheck_RepoDirFlagOverridesEnv(t *testing.T) {
	root := newRepo(t) // the correct repo, pin matches
	wrongEnvRoot := t.TempDir()
	writeFile(t, filepath.Join(wrongEnvRoot, "hyper.yaml"), "kind: repository-declaration\nversion: 9.9.9\n")

	getenv := func(k string) string {
		if k == "HYPER_REPO_DIR" {
			return wrongEnvRoot
		}
		return ""
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root}, &stdout, &stderr, getenv, t.TempDir(), "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d (flag should win over HYPER_REPO_DIR); stderr=%q", exit, cli.ExitClean, stderr.String())
	}
}

func TestRunCheck_NoColorFlagAndEnvProduceIdenticalBytes(t *testing.T) {
	root := newRepo(t)
	writeFile(t, filepath.Join(root, "definitions", "broken.yaml"), "kind: definition\nbase: &b\n  x: 1\ntargets: *b\n")
	getenv := func(string) string { return "" }

	var plain bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root}, &plain, &plain, getenv, root, "1.4.0")

	var flagged bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root, "--no-color"}, &flagged, &flagged, getenv, root, "1.4.0")

	getenvNoColor := func(k string) string {
		if k == "NO_COLOR" {
			return "1"
		}
		return ""
	}
	var envd bytes.Buffer
	cli.RunCheck([]string{"--repo-dir", root}, &envd, &envd, getenvNoColor, root, "1.4.0")

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
	getenv := func(string) string { return "" }

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "definitions/broken.yaml", "definitions/typo.yaml"}, &stdout, &stderr, getenv, root, "1.4.0")

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
	getenv := func(string) string { return "" }

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json", "definitions"}, &stdout, &stderr, getenv, root, "1.4.0")

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

	getenv := func(string) string { return "" }
	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root, "--json"}, &stdout, &stderr, getenv, root, "1.4.0")

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
