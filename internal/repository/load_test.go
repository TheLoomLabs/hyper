package repository

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
)

// fiveArtefacts writes one artefact of every location §12 fixes, in shapes the
// load itself reads: the load runs yamlsubset's grammar and builds the four
// namespaces, and nothing here declares an Operation or a Step, so every
// fixture below is a clean load without being a clean check.
func fiveArtefacts(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "hyper.yaml", "kind: repository-declaration\nversion: 1.4.0\n")
	write(t, root, "definitions/uptime.yaml", "kind: definition\ndefinition: uptime\nprovider: shell\ntargets: [prod]\n")
	write(t, root, "procedures/deploy.yaml", "kind: procedure\nprocedure: deploy\n")
	write(t, root, "targets/prod.yaml", "kind: target-declaration\ntarget: prod\n")
	write(t, root, "providers/hetzner.yaml", "kind: provider\nprovider: hetzner\n")
	return root
}

// byPath indexes a load by each artefact's own path, which is what every
// assertion below names an artefact by — the walk's order is the load's
// business and no criterion in issue #109 is about it.
func byPath(loaded Loaded) map[string]LoadedArtefact {
	idx := make(map[string]LoadedArtefact, len(loaded.Artefacts))
	for _, a := range loaded.Artefacts {
		idx[a.Path] = a
	}
	return idx
}

func TestLoad_LoadsEveryArtefactFilePlusTheBuiltinProvider(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		artefact.BuiltinShellProviderPath,
		"hyper.yaml",
		"definitions/uptime.yaml",
		"procedures/deploy.yaml",
		"targets/prod.yaml",
		"providers/hetzner.yaml",
	}
	got := byPath(loaded)
	if len(got) != len(want) {
		t.Fatalf("Load() loaded %d artefacts, want %d", len(got), len(want))
	}
	for _, path := range want {
		if _, ok := got[path]; !ok {
			t.Errorf("Load() loaded no artefact at %q", path)
		}
	}
}

// TestLoad_BytesAreTheFilesOwnBytes is the round trip issue #109 asks for:
// manifest_digest is SHA-256 over a Manifest's exact bytes and never over a
// canonical form of what they parse to (§7), so the load's bytes are the
// file's, unmodified.
func TestLoad_BytesAreTheFilesOwnBytes(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, a := range loaded.Artefacts {
		if a.Path == artefact.BuiltinShellProviderPath {
			continue // no file to read it back from; asserted below
		}
		onDisk, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(a.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a.Bytes, onDisk) {
			t.Errorf("%s: loaded bytes = %q, want the file's own %q", a.Path, a.Bytes, onDisk)
		}
	}
}

func TestLoad_EveryLoadedArtefactCarriesItsParsedRoot(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, a := range loaded.Artefacts {
		if !a.OK {
			t.Errorf("%s: OK = false, want every fixture here to parse; problems = %v", a.Path, a.Problems)
			continue
		}
		if a.Root == nil {
			t.Errorf("%s: Root = nil, want the parsed root of a file that supplied a document", a.Path)
		}
	}
}

// TestLoad_BuiltinShellProviderIsALoadedArtefactLikeAnyOther is ADR-0039's own
// consequence: the one Provider with no blob in the repository is a member of
// the load on the same footing as any other, its bytes being the compiled-in
// constant and its path the pseudo-path §9 renders (§3, §11).
func TestLoad_BuiltinShellProviderIsALoadedArtefactLikeAnyOther(t *testing.T) {
	loaded, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	builtin, ok := byPath(loaded)[artefact.BuiltinShellProviderPath]
	if !ok {
		t.Fatalf("Load() loaded no artefact at %q", artefact.BuiltinShellProviderPath)
	}
	if got, want := string(builtin.Bytes), artefact.BuiltinShellProviderYAML; got != want {
		t.Errorf("built-in bytes = %q, want the compiled-in constant %q", got, want)
	}
	if !builtin.OK {
		t.Errorf("built-in OK = false, want hyper's own bytes to load; problems = %v", builtin.Problems)
	}
	if builtin.Root == nil {
		t.Errorf("built-in Root = nil, want the parsed root of the compiled-in Manifest")
	}
	if len(builtin.Problems) != 0 {
		t.Errorf("built-in problems = %v, want none: the load reports what it read, and the built-in's schema is CheckBuiltinShellProvider's", builtin.Problems)
	}
}

func TestLoad_UnreadableFileCarriesOneProblemAndEveryOtherArtefactIsUntouched(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission bits are not enforced when running as root")
	}
	root := fiveArtefacts(t)
	locked := filepath.Join(root, "definitions", "locked.yaml")
	write(t, root, "definitions/locked.yaml", "kind: definition\ndefinition: locked\n")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil: a file hyper cannot read is one problem, not a failed load", err)
	}

	got := byPath(loaded)
	a := got["definitions/locked.yaml"]
	if a.OK {
		t.Errorf("locked.yaml OK = true, want false")
	}
	if len(a.Problems) != 1 {
		t.Fatalf("locked.yaml problems = %v, want exactly one", a.Problems)
	}
	if a.Problems[0].File != "definitions/locked.yaml" || a.Problems[0].Line != 1 {
		t.Errorf("locked.yaml problem = %+v, want it positioned at its own file, line 1", a.Problems[0])
	}
	if other := got["definitions/uptime.yaml"]; !other.OK || len(other.Problems) != 0 {
		t.Errorf("uptime.yaml = %+v, want it untouched by its neighbour's failure", other)
	}
	if _, ok := loaded.Definitions["uptime"]; !ok {
		t.Error("Definitions has no uptime; a file that cannot be read stops every check after it for that file, never for the repository")
	}
}

func TestLoad_UnparseableFileCarriesItsProblemsAndEveryOtherArtefactIsUntouched(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "definitions/broken.yaml", "kind: definition\n  : [unclosed\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil: a file that will not parse is a problem, not a failed load", err)
	}

	got := byPath(loaded)
	a := got["definitions/broken.yaml"]
	if a.OK {
		t.Errorf("broken.yaml OK = true, want false")
	}
	if len(a.Problems) != 1 {
		t.Fatalf("broken.yaml problems = %v, want exactly one", a.Problems)
	}
	if a.Problems[0].File != "definitions/broken.yaml" {
		t.Errorf("broken.yaml problem = %+v, want it positioned at its own file", a.Problems[0])
	}
	if other := got["definitions/uptime.yaml"]; !other.OK || len(other.Problems) != 0 {
		t.Errorf("uptime.yaml = %+v, want it untouched by its neighbour's failure", other)
	}
	if _, ok := loaded.Providers["hetzner"]; !ok {
		t.Error("Providers has no hetzner; one unparseable file does not stop the pass")
	}
}

// TestLoad_AFileThatWillNotParseContributesNoNameToAnyNamespace is ADR-0064's
// rule, held at the load: a name nothing legible declared is not in the
// namespace other artefacts resolve against, and the file's faults are check's
// to report.
func TestLoad_AFileThatWillNotParseContributesNoNameToAnyNamespace(t *testing.T) {
	root := t.TempDir()
	write(t, root, "providers/hetzner.yaml", "provider: hetzner\n  : [unclosed\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Providers["hetzner"]; ok {
		t.Error("Providers has hetzner; a file that will not parse contributes no name")
	}
}

// TestLoad_NamesResolveAgainstTheWholeRepository is the two-pass ordering issue
// #93 fixed and issue #109 must not lose: every file is parsed before a single
// name is resolved, so a Definition's targets: resolves against the whole
// repository's Target namespace rather than against the files walked before it.
// targets/ is walked after definitions/, which is what makes this an assertion
// rather than a coincidence.
func TestLoad_NamesResolveAgainstTheWholeRepository(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	definition, ok := loaded.Definitions["uptime"]
	if !ok {
		t.Fatalf("Definitions = %v, want an entry for uptime", loaded.Definitions)
	}
	if _, ok := definition.Targets["prod"]; !ok {
		t.Errorf("uptime's Targets = %v, want prod resolved from targets/, which is walked after definitions/", definition.Targets)
	}
}

func TestLoad_BuildsTheFourNamespaces(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := loaded.Providers["hetzner"]; !ok {
		t.Errorf("Providers = %v, want an entry for hetzner", loaded.Providers)
	}
	if _, ok := loaded.Providers["shell"]; !ok {
		t.Errorf("Providers = %v, want the built-in shell entry", loaded.Providers)
	}
	if _, ok := loaded.Targets["prod"]; !ok {
		t.Errorf("Targets = %v, want an entry for prod", loaded.Targets)
	}
	if _, ok := loaded.Definitions["uptime"]; !ok {
		t.Errorf("Definitions = %v, want an entry for uptime", loaded.Definitions)
	}
	if !loaded.Procedures["deploy"] {
		t.Errorf("Procedures = %v, want an entry for deploy", loaded.Procedures)
	}
}

func TestLoad_AnUnreadableArtefactDirectoryIsAFailedLoad(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission bits are not enforced when running as root")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "definitions")
	if err := os.Mkdir(dir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if _, err := Load(root); err == nil {
		t.Error("Load() error = nil, want the walk's own failure: a directory hyper cannot list is not an artefact's problem to carry")
	}
}
