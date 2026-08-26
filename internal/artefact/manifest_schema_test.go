package artefact

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// manifestAboveTheCeiling is a Manifest declaring a schema version above the
// one this binary reads and carrying, beneath it, four faults its own checks
// would each report: a kind: no providers/ file may declare, a provider: that is
// not its file's basename, a sixth top-level key, and a capabilities: list
// disagreeing with what its one Operation's request derives.
//
// Every one of those four is read off a key positioned by a shape this binary
// does not know, which is what the cases below are about: they are not faults
// the reader found, they are faults the reader guessed at.
const manifestAboveTheCeiling = `kind: definition
provider: elsewhere
schema-version: 2
class: local
capabilities: []
extra: 1
operations:
  check_http:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /}
    input: {type: object, properties: {}}
    record: {identity: $.status, fields: {status: $.status}}
`

// manifestAtTheCeiling is that same file with one digit changed, and it is
// derived rather than written out: the digit **is** the whole of the mechanism,
// so a second copy that drifted by a key would make the pair prove something
// neither case is about.
var manifestAtTheCeiling = strings.Replace(manifestAboveTheCeiling, "schema-version: 2", "schema-version: 1", 1)

// TestCheckManifest_ASchemaVersionAboveTheCeilingIsRefused is §11's rule as
// ADR-0028 already states it one artefact-class over: read at or below, Refuse
// above, and never guess.
func TestCheckManifest_ASchemaVersionAboveTheCeilingIsRefused(t *testing.T) {
	got := checkManifest(t, "providers/uptime.yaml", manifestAboveTheCeiling)
	p := mustCode(t, got, CodeManifestSchemaUnsupported)
	if p.Field != "schema-version" {
		t.Errorf("Field = %q, want schema-version", p.Field)
	}
	if p.Line != 3 || p.Column != 17 {
		t.Errorf("cites %d:%d, want the schema-version: scalar at 3:17", p.Line, p.Column)
	}
}

// TestCheckManifest_TheUnsupportedSchemaReplacesEveryOtherCheck is the rule
// §11 calls expensive by name: a Manifest read on a partial understanding of
// its own shape has its checks run against keys the reader could not see, so
// this file's checks are not run at all. One code, and nothing else said.
func TestCheckManifest_TheUnsupportedSchemaReplacesEveryOtherCheck(t *testing.T) {
	got := checkManifest(t, "providers/uptime.yaml", manifestAboveTheCeiling)
	if len(got) != 1 {
		t.Fatalf("reported %+v, want the unsupported schema alone", got)
	}
	if got[0].ErrorCode != CodeManifestSchemaUnsupported {
		t.Errorf("reported %s, want %s alone", got[0].ErrorCode, CodeManifestSchemaUnsupported)
	}
}

// TestCheckManifest_AtTheCeilingEveryCheckRuns is the other half, and the one
// that says the suppression is the version's and not a file's: the same four
// faults, one digit lower, are every one of them reported.
func TestCheckManifest_AtTheCeilingEveryCheckRuns(t *testing.T) {
	got := checkManifest(t, "providers/uptime.yaml", manifestAtTheCeiling)
	mustNoCode(t, got, CodeManifestSchemaUnsupported)
	for _, code := range []string{CodeKindMismatch, CodeNameMismatch, schema.CodeUnknownKey, CodeCapabilityMismatch} {
		mustCode(t, got, code)
	}
}

// TestCheckManifest_ASchemaVersionTheReaderCannotReadIsSchemaMismatchsOwn is
// the boundary this check declines to guess at from the other side: a
// schema-version: that is absent, or is not an integer at all, is a fault the
// top-level schema already names, and a second opinion here would put two rows
// on one line for one cause.
func TestCheckManifest_ASchemaVersionTheReaderCannotReadIsSchemaMismatchsOwn(t *testing.T) {
	for name, doc := range map[string]string{
		"absent":      "kind: provider\nprovider: uptime\nclass: local\ncapabilities: [http]\noperations: {}\n",
		"not-integer": "kind: provider\nprovider: uptime\nschema-version: two\nclass: local\ncapabilities: [http]\noperations: {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			got := checkManifest(t, "providers/uptime.yaml", doc)
			mustNoCode(t, got, CodeManifestSchemaUnsupported)
			mustCode(t, got, schema.CodeMismatch)
		})
	}
}

// TestManifestSchemaVersion_TheBuiltinSatisfiesTheCeiling holds the compiled-in
// Manifest to the compiled-in ceiling. The two move together or the binary
// refuses its own Provider, which is a state no repository author could clear.
func TestManifestSchemaVersion_TheBuiltinSatisfiesTheCeiling(t *testing.T) {
	root := BuiltinShellProviderRoot()
	if ManifestSchemaUnsupported(root) {
		t.Fatalf("the built-in declares a schema version above the %d this hyper reads", ManifestSchemaVersion)
	}
	if _, declared, legible := manifestSchemaVersion(root); !legible || declared != ManifestSchemaVersion {
		t.Errorf("the built-in declares schema-version %d (legible: %v), want the ceiling %d", declared, legible, ManifestSchemaVersion)
	}
}

// TestManifestSchemaUnsupported_IsTheOnePredicateBothHalvesRead is the shape
// provider-name-collision already has: the fold declines a file and the check
// names it, over one predicate, so a Manifest that vanished from the Provider
// namespace can never be one nothing said a word about.
func TestManifestSchemaUnsupported_IsTheOnePredicateBothHalvesRead(t *testing.T) {
	if !ManifestSchemaUnsupported(parse(t, manifestAboveTheCeiling)) {
		t.Error("a Manifest above the ceiling reads as supported")
	}
	if ManifestSchemaUnsupported(parse(t, manifestAtTheCeiling)) {
		t.Error("a Manifest at the ceiling reads as unsupported")
	}
	if ManifestSchemaUnsupported(nil) {
		t.Error("a file that parsed to no document reads as unsupported; it carries no version at all")
	}
}

// TestCodeManifestSchemaUnsupported_IsNotTheInputSchemasCode is the distinction
// §11 spells rather than reaches for. The two read alike and are different
// checks: one is an input schema reaching outside §4's four-keyword subset,
// whose remedy is an ordinary artefact edit, and this one is a whole file in a
// shape this binary does not know, whose remedy is a different binary.
func TestCodeManifestSchemaUnsupported_IsNotTheInputSchemasCode(t *testing.T) {
	if CodeManifestSchemaUnsupported != "manifest-schema-unsupported" {
		t.Errorf("code = %q, want manifest-schema-unsupported", CodeManifestSchemaUnsupported)
	}
	if CodeManifestSchemaUnsupported == CodeSchemaUnsupported {
		t.Error("the two schema codes are one string; they are different checks with different remedies")
	}
}
