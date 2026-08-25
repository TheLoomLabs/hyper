package verify_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/verify"
)

// forked is a Manifest in providers/ declaring name — the built-in shell
// Provider's shape with the reserved Capability left out, which is the fork
// §11 admits: a built-in is forkable in form, and the copy may not declare the
// Capability nobody but a built-in may hold.
//
// It is written at whatever name a case hands it so that the collision and the
// fork are one file with one word changed, that word being the whole of what
// the check reads.
func forked(name string) string {
	return `kind: provider
provider: ` + name + `
schema-version: 1
class: local
capabilities: [http]
operations:
  ping:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /}
    input: {type: object, properties: {}}
    record: {identity: $.status, fields: {status: $.status}}
`
}

// colliding is the repository every case below varies: the fork above written
// at the built-in's own name, in the file that name obliges it to sit in.
func colliding() map[string]string {
	files := bound(nil)
	files["providers/shell.yaml"] = forked("shell")
	return files
}

// collisions is the name-collision problems one pass found, and nothing else it
// found.
func collisions(problems []problem.Problem) []problem.Problem {
	var found []problem.Problem
	for _, held := range problems {
		if held.ErrorCode == verify.CodeProviderNameCollision {
			found = append(found, held)
		}
	}
	return found
}

// TestRepository_AManifestTakingABuiltinsNameIsACollision is §11's own
// sentence, and the citation it lands on: the offending file's provider:
// scalar, the built-in having no file to cite and being no part of the fault.
func TestRepository_AManifestTakingABuiltinsNameIsACollision(t *testing.T) {
	found := collisions(checked(t, colliding()))
	if len(found) != 1 {
		t.Fatalf("reported %v, want one problem: providers/shell.yaml takes the built-in shell Provider's name", found)
	}
	if found[0].File != "providers/shell.yaml" {
		t.Errorf("cites %q, want providers/shell.yaml — the file whose author can act", found[0].File)
	}
	if found[0].Line != 2 || found[0].Column != 11 || found[0].Field != "provider" {
		t.Errorf("cites %s:%d:%d %q, want the provider: scalar at 2:11", found[0].File, found[0].Line, found[0].Column, found[0].Field)
	}
	if !strings.Contains(found[0].Message, "built-in") {
		t.Errorf("message is %q, want it to say the name is a built-in Provider's", found[0].Message)
	}
}

// TestRepository_AnExtensionTakingNoBuiltinNameReportsNothing is the case the
// one above is that case with one word changed: a Provider author forking a
// built-in under a name of their own is exactly what §11 admits, and a rule
// that fired on it would be one nobody could clear.
func TestRepository_AnExtensionTakingNoBuiltinNameReportsNothing(t *testing.T) {
	files := bound(nil)
	files["providers/local-shell.yaml"] = forked("local-shell")

	if found := checked(t, files); len(found) > 0 {
		t.Errorf("a repository whose Extensions take no built-in name reports %v", found)
	}
}

// TestRepository_TheBuiltinStandsWhileTheCollidingFileIsInTheTree is the
// assertion that would have caught the silent shadowing, and the one a corpus
// checking only the error code would not have: the name still means the
// compiled-in Manifest, with its origin and its six Operations.
func TestRepository_TheBuiltinStandsWhileTheCollidingFileIsInTheTree(t *testing.T) {
	loaded := load(t, colliding())

	held, stands := loaded.Manifests["shell"]
	if !stands {
		t.Fatal("the Provider namespace holds no shell at all; the built-in stands and a colliding file contributes nothing")
	}
	if held.Origin != artefact.OriginBuiltIn || held.Path != artefact.BuiltinShellProviderPath {
		t.Errorf("shell loaded from %s with origin %s, want the compiled-in Manifest at %s with origin %s",
			held.Path, held.Origin, artefact.BuiltinShellProviderPath, artefact.OriginBuiltIn)
	}
	if operations := loaded.Providers["shell"].Operations; len(operations) != 6 {
		t.Errorf("shell declares %d Operations, want the built-in's six", len(operations))
	}
}

// TestRepository_ADefinitionNamingACollidedNameResolvesThroughTheBuiltin is
// what the built-in standing is *for*: a Definition reviewed against the
// built-in runs against the built-in, and nothing further is said about it.
func TestRepository_ADefinitionNamingACollidedNameResolvesThroughTheBuiltin(t *testing.T) {
	files := colliding()
	files["definitions/heartbeat.yaml"] = "kind: definition\ndefinition: heartbeat\nprovider: shell\nkinds: [read]\ntargets: [staging]\n"
	files["targets/staging.yaml"] = "kind: target-declaration\ntarget: staging\nclass: local\nkinds: [read]\ncapabilities: [shell]\n"

	found := checked(t, files)
	if len(found) != 1 {
		t.Fatalf("reported %v, want the collision alone: a Definition naming shell resolves through the built-in", found)
	}
	if found[0].ErrorCode != verify.CodeProviderNameCollision {
		t.Errorf("reported %s, want the collision alone", found[0].ErrorCode)
	}
}
