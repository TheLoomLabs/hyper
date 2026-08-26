package repository

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// TestLoad_ManifestsAreTheProviderNamespacesOtherHalf is the fold issue #111
// needed and #109 did not carry: every name in Providers has an entry here
// naming the file its bytes came from, its origin, and those bytes — which is
// what a manifest_digest covers and what the index alone cannot answer.
func TestLoad_ManifestsAreTheProviderNamespacesOtherHalf(t *testing.T) {
	root := fiveArtefacts(t)

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for name := range loaded.Providers {
		if _, ok := loaded.Manifests[name]; !ok {
			t.Errorf("%s is in the Provider namespace and has no Manifest; the two are one fold", name)
		}
	}
	for name := range loaded.Manifests {
		if _, ok := loaded.Providers[name]; !ok {
			t.Errorf("%s has a Manifest and is not in the Provider namespace; the two are one fold", name)
		}
	}

	builtin := loaded.Manifests["shell"]
	if builtin.Origin != artefact.OriginBuiltIn || builtin.Path != artefact.BuiltinShellProviderPath {
		t.Errorf("the built-in Provider loaded as %+v, want origin %s at %s", builtin, artefact.OriginBuiltIn, artefact.BuiltinShellProviderPath)
	}
	if string(builtin.Bytes) != artefact.BuiltinShellProviderYAML {
		t.Error("the built-in Provider's bytes are not the compiled-in Manifest")
	}

	hetzner := loaded.Manifests["hetzner"]
	if hetzner.Origin != artefact.OriginExtension || hetzner.Path != "providers/hetzner.yaml" {
		t.Errorf("a providers/ file loaded as %+v, want origin %s at providers/hetzner.yaml", hetzner, artefact.OriginExtension)
	}
	if want := "kind: provider\nprovider: hetzner\n"; string(hetzner.Bytes) != want {
		t.Errorf("Bytes = %q, want the file's own bytes %q", hetzner.Bytes, want)
	}
}

// TestLoad_AnExtensionTakingABuiltinsNameContributesNothing is §11's own rule
// at the fold that used to answer the other way: a providers/ file declaring a
// built-in Provider's name is no member of the Provider namespace, and the name
// still means the compiled-in Manifest.
//
// It is the load's half of `provider-name-collision`. The row that says so is
// the static pass's (internal/verify), and what is asserted here is the thing a
// row cannot state: **the namespace did not move**, so a Definition reviewed
// against the built-in resolves through the built-in.
func TestLoad_AnExtensionTakingABuiltinsNameContributesNothing(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "providers/shell.yaml", "kind: provider\nprovider: shell\nclass: local\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	held := loaded.Manifests[artefact.BuiltinShellProviderName]
	if held.Origin != artefact.OriginBuiltIn || held.Path != artefact.BuiltinShellProviderPath {
		t.Errorf("shell loaded as %+v, want the compiled-in Manifest at %s with origin %s",
			held, artefact.BuiltinShellProviderPath, artefact.OriginBuiltIn)
	}
	if string(held.Bytes) != artefact.BuiltinShellProviderYAML {
		t.Error("shell's bytes are the colliding file's; the built-in stands and the Extension contributes nothing")
	}
	if operations := loaded.Providers[artefact.BuiltinShellProviderName].Operations; len(operations) != 6 {
		t.Errorf("shell declares %d Operations, want the built-in's six", len(operations))
	}
	for _, a := range loaded.Artefacts {
		if a.Path == "providers/shell.yaml" {
			return
		}
	}
	t.Error("the colliding file is in no load at all; it contributes no name and is still an artefact check reports against")
}

// TestLoad_AManifestAboveTheSchemaCeilingContributesNothing is the same
// declining one rule over, and for the reason that makes it the sharper of the
// two: an Extension taking a built-in's name is a file this binary understands
// and may not admit, and this is a file this binary does not understand at all.
// Folding a name off it would be reading one key of a shape nobody defined.
//
// It is the load's half of `manifest-schema-unsupported`. The row that says so
// is the file's own check (internal/artefact), and what is asserted here is what
// a row cannot state: the name means nothing, so a Definition that names it
// resolves to nothing and earns `artefact-absent` (§11, ADR-0028).
func TestLoad_AManifestAboveTheSchemaCeilingContributesNothing(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "providers/hetzner.yaml", "kind: provider\nprovider: hetzner\nschema-version: 2\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	if held, named := loaded.Manifests["hetzner"]; named {
		t.Errorf("hetzner loaded as %+v; a Manifest written in a shape this hyper does not know names nothing", held)
	}
	if _, named := loaded.Providers["hetzner"]; named {
		t.Error("hetzner is in the Provider namespace; a Definition naming it would resolve through a shape nobody defined")
	}
	for _, a := range loaded.Artefacts {
		if a.Path == "providers/hetzner.yaml" {
			return
		}
	}
	t.Error("the file is in no load at all; it contributes no name and is still an artefact check reports against")
}

// TestLoad_TargetDeclarationsAreTheTargetNamespacesOtherHalf is the fold issue
// #112 needs, on the shape #111 gave the Provider namespace: every name in
// Targets has the declaration it was read from here, which is what `hyper
// targets` states a row off and what the index — a membership set per name —
// cannot answer.
func TestLoad_TargetDeclarationsAreTheTargetNamespacesOtherHalf(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "targets/staging.yaml", "kind: target-declaration\ntarget: staging\ncapabilities: [http]\nhosts: [b.example, a.example]\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for name := range loaded.Targets {
		if _, ok := loaded.TargetDeclarations[name]; !ok {
			t.Errorf("%s is in the Target namespace and has no declaration; the two are one fold", name)
		}
	}
	for name := range loaded.TargetDeclarations {
		if _, ok := loaded.Targets[name]; !ok {
			t.Errorf("%s has a declaration and is not in the Target namespace; the two are one fold", name)
		}
	}

	// The declaration itself and not a reading of it: the row's own facts
	// are read off this root, in the order the file states them.
	facts := artefact.ReadTargetFacts(loaded.TargetDeclarations["staging"])
	if got, want := facts.Hosts, []string{"b.example", "a.example"}; !slices.Equal(got, want) {
		t.Errorf("hosts = %v, want %v — the declaration's own order", got, want)
	}
}

// TestLoad_ATargetDeclarationThatNamesNothingIsInNoNamespace is ADR-0064's rule
// at the Target fold, exactly as it stands at the Provider one: a file that
// will not parse, and one that parses without naming itself, each contribute no
// name — so neither is a row `hyper targets` writes and neither is a name a
// targets: member resolves to.
func TestLoad_ATargetDeclarationThatNamesNothingIsInNoNamespace(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "targets/broken.yaml", "kind: target-declaration\ntarget: broken\n  bad: [\n")
	write(t, root, "targets/nameless.yaml", "kind: target-declaration\nclass: local\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"broken", "nameless"} {
		if _, ok := loaded.TargetDeclarations[name]; ok {
			t.Errorf("%s named itself into the Target namespace; a file that names nothing is in no namespace", name)
		}
	}
	if _, ok := loaded.TargetDeclarations["prod"]; !ok {
		t.Error("the declaration that does name itself is missing; one faulty file stops nothing else")
	}
}

// TestLoad_AManifestThatNamesNothingIsInNoNamespace is ADR-0064's rule at the
// fold: a file that will not parse, and one that parses without naming itself,
// each contribute no name — so neither can be reported as a Provider and
// neither can be resolved against.
func TestLoad_AManifestThatNamesNothingIsInNoNamespace(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, "providers/broken.yaml", "kind: provider\nprovider: broken\n  bad: [\n")
	write(t, root, "providers/nameless.yaml", "kind: provider\nschema-version: 1\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := loaded.Manifests["broken"]; ok {
		t.Error("a Manifest that will not parse is in the Provider namespace")
	}
	if len(loaded.Manifests) != 2 {
		t.Errorf("Manifests holds %d entries, want the built-in and hetzner alone", len(loaded.Manifests))
	}
}

// TestLoad_CarriesTheGeneratedWorkflowsAsBytes is the namespace beside the five
// artefact locations: `hyper-*.yml` under `.github/workflows/`, read whole and
// carried as bytes, which is what §10's projection check compares against a
// fresh regeneration and what `project` reads to know what already stands
// (issue #179).
//
// What is asserted beside the bytes is the boundary: the namespace is
// `hyper-`-prefixed files **directly** under that directory, so a hand-written
// workflow beside them and a file one directory down are both somebody else's.
func TestLoad_CarriesTheGeneratedWorkflowsAsBytes(t *testing.T) {
	root := fiveArtefacts(t)
	write(t, root, ".github/workflows/hyper-deploy.yml", "name: deploy\n")
	write(t, root, ".github/workflows/hyper-nightly.yml", "name: nightly\n")
	write(t, root, ".github/workflows/release.yml", "name: release\n")
	write(t, root, ".github/workflows/nested/hyper-deeper.yml", "name: deeper\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}

	want := []LoadedWorkflow{
		{Path: ".github/workflows/hyper-deploy.yml", Bytes: []byte("name: deploy\n")},
		{Path: ".github/workflows/hyper-nightly.yml", Bytes: []byte("name: nightly\n")},
	}
	if len(loaded.Workflows) != len(want) {
		t.Fatalf("Workflows = %+v, want the two files in the namespace", loaded.Workflows)
	}
	for i, held := range loaded.Workflows {
		if held.Path != want[i].Path || !bytes.Equal(held.Bytes, want[i].Bytes) {
			t.Errorf("Workflows[%d] = %+v, want %+v", i, held, want[i])
		}
	}
}

// TestLoad_AGeneratedWorkflowIsNotAnArtefact is the other half of the sentence
// above, and it is what keeps a generated file out of everything an artefact is
// in: no LoadedArtefact, so nothing parses it, no namespace holds it, no schema
// check reads it, and `check`'s own count of what it checked does not move
// (§10, §12).
func TestLoad_AGeneratedWorkflowIsNotAnArtefact(t *testing.T) {
	root := fiveArtefacts(t)

	before, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, ".github/workflows/hyper-deploy.yml", "this: [is not, even, valid: yaml\n")

	after, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Artefacts) != len(before.Artefacts) {
		t.Errorf("the load holds %d artefacts with a generated workflow in the tree and %d without", len(after.Artefacts), len(before.Artefacts))
	}
	for _, a := range after.Artefacts {
		if strings.HasPrefix(a.Path, ".github/") {
			t.Errorf("%s loaded as an artefact; a generated file is derived from artefacts rather than being one", a.Path)
		}
	}
}

// TestLoadFrom_HoldsNoGeneratedWorkflow is the door that supplies none. The
// reaper and `changes` read artefacts out of a revision and verify nothing, so
// there is no directory to walk and nothing that would ask (§7, issue #154).
func TestLoadFrom_HoldsNoGeneratedWorkflow(t *testing.T) {
	loaded := LoadFrom([]Source{{Path: "hyper.yaml", Bytes: []byte("kind: repository-declaration\n")}})
	if len(loaded.Workflows) != 0 {
		t.Errorf("LoadFrom() carries %+v, want no generated workflow", loaded.Workflows)
	}
}
