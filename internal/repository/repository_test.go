package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRoot_WalksUpToGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := FindGitRoot(nested)
	if !ok {
		t.Fatalf("FindGitRoot() ok = false, want true")
	}
	if got != root {
		t.Errorf("FindGitRoot() = %q, want %q", got, root)
	}
}

func TestFindGitRoot_NoGitDirAnywhere(t *testing.T) {
	// A temp dir has no .git in any ancestor by construction of the test
	// sandbox, so this exercises the "bounded by the git root" failure path.
	root := t.TempDir()
	_, ok := FindGitRoot(root)
	if ok {
		t.Fatalf("FindGitRoot() ok = true, want false with no .git present")
	}
}

func TestArtefactFiles_CollectsFromAllFiveLocations(t *testing.T) {
	root := t.TempDir()
	write(t, root, "hyper.yaml", "kind: repository-declaration\n")
	write(t, root, "definitions/uptime.yaml", "kind: definition\n")
	write(t, root, "definitions/notes.txt", "not yaml\n") // must be ignored
	write(t, root, "procedures/deploy.yaml", "kind: procedure\n")
	write(t, root, "targets/prod.yaml", "kind: target-declaration\n")
	write(t, root, "providers/hetzner.yaml", "kind: provider\n")

	got, err := artefactFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"hyper.yaml":              true,
		"definitions/uptime.yaml": true,
		"procedures/deploy.yaml":  true,
		"targets/prod.yaml":       true,
		"providers/hetzner.yaml":  true,
	}
	if len(got) != len(want) {
		t.Fatalf("artefactFiles() = %v, want exactly %v", got, want)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("artefactFiles() included unexpected file %q", f)
		}
	}
}

func TestArtefactFiles_MissingDirectoriesAreNotAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, "hyper.yaml", "kind: repository-declaration\n")

	got, err := artefactFiles(root)
	if err != nil {
		t.Fatalf("artefactFiles() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != "hyper.yaml" {
		t.Fatalf("artefactFiles() = %v, want just [hyper.yaml]", got)
	}
}

func TestArtefactFiles_MissingHyperYAMLIsNotAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, "definitions/uptime.yaml", "kind: definition\n")

	got, err := artefactFiles(root)
	if err != nil {
		t.Fatalf("artefactFiles() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != "definitions/uptime.yaml" {
		t.Fatalf("artefactFiles() = %v, want just [definitions/uptime.yaml]", got)
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
