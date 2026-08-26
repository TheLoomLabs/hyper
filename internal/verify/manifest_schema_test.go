package verify_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
)

// aboveTheCeiling is §3's own `uptime` Manifest declaring a schema version this
// binary does not read, and carrying beneath it a sixth top-level key and a
// `capabilities:` list its one Operation's request disagrees with — two faults
// this reader would report on any file whose shape it knew.
const aboveTheCeiling = `kind: provider
provider: uptime
schema-version: 2
class: cloudflare
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

// forFile is the problems one pass found against one file, which is what every
// case here asks: a Manifest above the ceiling earns one row and no other, and
// the rows about *other* files are the point of the second case rather than
// noise in the first.
func forFile(problems []problem.Problem, file string) []problem.Problem {
	var found []problem.Problem
	for _, held := range problems {
		if held.File == file {
			found = append(found, held)
		}
	}
	return found
}

// TestRepository_AManifestAboveTheCeilingEarnsOneRowAndNoOther is the failure
// §11 calls expensive by name, refused: a file whose shape this binary does not
// know has nothing said about it but that, so the two deliberate faults beneath
// the version are not reported — they are keys the reader could not see.
func TestRepository_AManifestAboveTheCeilingEarnsOneRowAndNoOther(t *testing.T) {
	files := bound(nil)
	files["providers/uptime.yaml"] = aboveTheCeiling

	found := forFile(checked(t, files), "providers/uptime.yaml")
	if len(found) != 1 {
		t.Fatalf("reported %+v, want the unsupported schema alone", found)
	}
	if found[0].ErrorCode != artefact.CodeManifestSchemaUnsupported {
		t.Errorf("reported %s, want %s", found[0].ErrorCode, artefact.CodeManifestSchemaUnsupported)
	}
	if found[0].Line != 3 || found[0].Field != "schema-version" {
		t.Errorf("cites line %d field %q, want the schema-version: scalar on line 3", found[0].Line, found[0].Field)
	}
}

// TestRepository_ADefinitionNamingItReportsArtefactAbsent is the reach of that
// silence, and it is bounded by the fold rather than by suppression: the
// Manifest contributes nothing to the Provider namespace, so a Definition
// naming it resolves to nothing and says so in its own words. Two rows for one
// cause, which is ADR-0064's shape for a file that will not parse.
func TestRepository_ADefinitionNamingItReportsArtefactAbsent(t *testing.T) {
	files := bound(nil)
	files["providers/uptime.yaml"] = aboveTheCeiling

	found := forFile(checked(t, files), "definitions/heartbeat.yaml")
	if len(found) != 1 || found[0].ErrorCode != artefact.CodeArtefactAbsent {
		t.Fatalf("reported %+v, want artefact-absent alone against the Definition naming uptime", found)
	}
}

// TestRepository_AManifestAboveTheCeilingTakesNoBuiltinsName is the one place
// two of this milestone's rules could have landed on one file. They do not: a
// Manifest that contributes nothing to the Provider namespace takes no name
// inside it, so the collision has no subject — the check whose subject *is* the
// namespace says nothing, and the built-in stands as it would anyway.
func TestRepository_AManifestAboveTheCeilingTakesNoBuiltinsName(t *testing.T) {
	files := bound(nil)
	files["providers/shell.yaml"] = `kind: provider
provider: shell
schema-version: 2
class: local
capabilities: [shell]
operations: {}
`

	problems := checked(t, files)
	found := forFile(problems, "providers/shell.yaml")
	if len(found) != 1 || found[0].ErrorCode != artefact.CodeManifestSchemaUnsupported {
		t.Fatalf("reported %+v, want the unsupported schema alone — the name is taken from nobody", found)
	}
	if collisions(problems) != nil {
		t.Error("a Manifest this binary cannot read is reported as taking a name; it names nothing at all")
	}

	loaded := load(t, files)
	if held := loaded.Manifests[artefact.BuiltinShellProviderName]; held.Origin != artefact.OriginBuiltIn {
		t.Errorf("shell loaded with origin %s, want the compiled-in Manifest's %s", held.Origin, artefact.OriginBuiltIn)
	}
}
