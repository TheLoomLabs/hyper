package repository

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A repository loaded from bytes a caller already holds (§7, issue #154).
//
// The one caller is a reaper: it loads the Procedure the dead Run was running
// at the revision that Run named, which is a read of git rather than of the
// working tree, so the bytes arrive already in hand. What is asserted here is
// that the two doors give one answer — the same namespaces, the same
// built-in — because a load that judged its own artefacts differently by the
// door they came in would be two repositories under one name.

// sources is the five artefacts fiveArtefacts writes, read off the working tree
// so that a case can hand LoadFrom exactly what Load walks.
func sources(t *testing.T, root string, paths ...string) []Source {
	t.Helper()

	held := make([]Source, 0, len(paths))
	for _, path := range paths {
		bytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, Source{Path: path, Bytes: bytes})
	}
	return held
}

// TestLoadFrom_IsTheSameRepositoryTheWalkLoads is the claim in full: the same
// five artefacts through the other door build the same namespaces and the same
// built-in Provider, so a reaper reading a revision resolves a Step exactly as
// the Run that performed it did.
func TestLoadFrom_IsTheSameRepositoryTheWalkLoads(t *testing.T) {
	root := fiveArtefacts(t)

	walked, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	handed := LoadFrom(sources(t, root,
		"definitions/uptime.yaml", "hyper.yaml", "procedures/deploy.yaml",
		"providers/hetzner.yaml", "targets/prod.yaml"))

	if _, held := handed.Procedures["deploy"]; !held {
		t.Errorf("the Procedure namespace holds %v, want deploy", handed.Procedures)
	}
	if _, held := handed.Definitions["uptime"]; !held {
		t.Errorf("the Definition namespace holds %v, want uptime", handed.Definitions)
	}
	if _, held := handed.Targets["prod"]; !held {
		t.Errorf("the Target namespace holds %v, want prod", handed.Targets)
	}
	for _, name := range []string{"hetzner", "shell"} {
		if _, held := handed.Manifests[name]; !held {
			t.Errorf("the Provider namespace holds no %s", name)
		}
	}
	if handed.Declaration() == nil {
		t.Error("the load answers no Repository declaration, want hyper.yaml's root")
	}

	// The one artefact no walk found and no caller handed over: the
	// built-in Provider has no blob in the repository at all, so a load
	// built from a revision's bytes reaches it exactly as the walk does
	// (§3, ADR-0039).
	if got, want := len(handed.Artefacts), len(walked.Artefacts); got != want {
		t.Errorf("the handed load holds %d artefacts and the walked one %d", got, want)
	}
}

// TestLoadFrom_HoldsTheOrderItWasHandedIn is why the input is a slice and not a
// mapping. Where two files declare one name the later one wins, which is the
// fold's rule rather than a precedence this package is entitled to — and a
// mapping's iteration order would make *which* of them wins a fact about the
// run rather than about the repository.
func TestLoadFrom_HoldsTheOrderItWasHandedIn(t *testing.T) {
	first := Source{Path: "targets/a.yaml", Bytes: []byte("kind: target-declaration\ntarget: prod\n")}
	second := Source{Path: "targets/b.yaml", Bytes: []byte("kind: target-declaration\ntarget: prod\n")}

	if got, _ := LoadFrom([]Source{first, second}).TargetDeclaration("prod"); got.Path != second.Path {
		t.Errorf("prod resolved to %q, want the later file %q", got.Path, second.Path)
	}
	if got, _ := LoadFrom([]Source{second, first}).TargetDeclaration("prod"); got.Path != first.Path {
		t.Errorf("prod resolved to %q, want the later file %q", got.Path, first.Path)
	}
}

// TestLoadFrom_ReadsAnArtefactThatWillNotParseExactlyAsTheWalkDoes is the
// load's own rule reaching the other door: what a single file can do wrong is
// carried on that file and stops nothing else (issue #88).
func TestLoadFrom_ReadsAnArtefactThatWillNotParseExactlyAsTheWalkDoes(t *testing.T) {
	loaded := LoadFrom([]Source{
		{Path: "definitions/broken.yaml", Bytes: []byte("kind: definition\n\tdefinition: broken\n")},
		{Path: "targets/prod.yaml", Bytes: []byte("kind: target-declaration\ntarget: prod\n")},
	})

	held := byPath(loaded)
	if held["definitions/broken.yaml"].OK {
		t.Error("an artefact that will not parse loaded OK")
	}
	if len(held["definitions/broken.yaml"].Problems) == 0 {
		t.Error("an artefact that will not parse carries no problem")
	}
	if _, held := loaded.Targets["prod"]; !held {
		t.Error("the file beside one that will not parse contributed no name")
	}
}

// TestIsArtefact is the selection rule, exported because a caller reading a
// revision out of git filters a listing by it where the walk filters a
// directory. Both name the five locations §12 fixes, and they name them once.
func TestIsArtefact(t *testing.T) {
	held := []string{
		"hyper.yaml",
		"definitions/uptime.yaml",
		"procedures/deploy.yaml",
		"providers/hetzner.yaml",
		"targets/prod.yaml",
	}
	absent := []string{
		"README.md",
		"definitions/notes.md",
		"definitions/nested/deep.yaml",
		"docs/definitions/uptime.yaml",
		"nested/hyper.yaml",
		"definitions",
		"records/local/uptime/status/one.json",
	}

	for _, path := range held {
		if !IsArtefact(path) {
			t.Errorf("%q is not read as an artefact, want it read as one", path)
		}
	}
	for _, path := range absent {
		if IsArtefact(path) {
			t.Errorf("%q is read as an artefact, want it read as none", path)
		}
	}
}

// TestIsArtefact_AnswersEveryPathTheWalkFinds holds the two rules to each other
// over a real directory, which is what keeps them one rule rather than two that
// happen to agree today.
func TestIsArtefact_AnswersEveryPathTheWalkFinds(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "README.md", "# not an artefact\n")
	write(t, root, "definitions/notes.md", "not an artefact either\n")

	walked, err := artefactFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(walked)
	for _, path := range walked {
		if !IsArtefact(path) {
			t.Errorf("the walk found %q and IsArtefact reads it as no artefact", path)
		}
	}
	if IsArtefact("README.md") || IsArtefact("definitions/notes.md") {
		t.Error("a file the walk skipped reads as an artefact")
	}
}
