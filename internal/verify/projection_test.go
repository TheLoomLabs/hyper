package verify_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/verify"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// projectedDigest is the checksum the cases below project at. It is the
// caller's fact rather than the repository's — `project` freezes it in the same
// act that writes the pin — so it is stated here once rather than read back off
// each case's own declaration.
const projectedDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// projected is one repository written into a temp directory, loaded and
// projected. The files are the least a Procedure needs to reach a Step and a
// credential slot, and every case below is that repository with one thing
// changed.
func projected(t *testing.T, files map[string]string) []verify.ProjectedWorkflow {
	t.Helper()

	loaded, err := repository.Load(write(t, files))
	if err != nil {
		t.Fatal(err)
	}
	return verify.Projection(loaded, "1.4.0", projectedDigest)
}

// write lays one repository's files out in a temp directory and answers its
// root. It is projected's other half, split out for the one case that projects
// at a version and a digest of its own.
func write(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// bound is the repository the cases below vary: a Manifest with `header:` auth
// over a `read` Operation, a Target naming the variable its `token` slot reads,
// a Definition binding the two, and one Procedure with one Step.
func bound(procedures map[string]string) map[string]string {
	files := map[string]string{
		"hyper.yaml": "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:" + strings.Repeat("0", 64) + "\n",
		"providers/uptime.yaml": `kind: provider
provider: uptime
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  check_http:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /}
    input: {type: object, properties: {}}
    record: {identity: $.status, fields: {status: $.status}}
`,
		"targets/staging.yaml": `kind: target-declaration
target: staging
class: cloudflare
kinds: [read]
capabilities: [http]
hosts: [status.hyper.dev]
auth:
  token: {env: STAGING_TOKEN}
`,
		"definitions/heartbeat.yaml": "kind: definition\ndefinition: heartbeat\nprovider: uptime\nkinds: [read]\ntargets: [staging]\n",
	}
	for path, content := range procedures {
		files[path] = content
	}
	return files
}

// procedure is one Procedure declaring name, with cadence where it is not empty
// and one `read` Step.
func procedure(name, cadence string) string {
	declared := ""
	if cadence != "" {
		declared = "cadence: \"" + cadence + "\"\n"
	}
	return "kind: procedure\nprocedure: " + name + "\ntargets: [staging]\n" + declared +
		"steps:\n  - id: beat\n    definition: heartbeat\n    operation: check_http\n    target: staging\n"
}

// TestProjection_OneFilePerProcedureDeclaringACadence is §10's own sentence, and
// the two halves of it: a Cadence puts a file in the namespace, and no Cadence
// puts nothing there.
func TestProjection_OneFilePerProcedureDeclaringACadence(t *testing.T) {
	files := projected(t, bound(map[string]string{
		"procedures/beat.yaml":  procedure("beat", "0 3 * * 1"),
		"procedures/quiet.yaml": procedure("quiet", ""),
	}))

	if len(files) != 1 {
		t.Fatalf("projected %d files, want the one Procedure that declares a recurrence: %+v", len(files), files)
	}
	if files[0].Procedure != "beat" || files[0].Cadence != "0 3 * * 1" {
		t.Errorf("projected %q at %q, want beat's own 0 3 * * 1", files[0].Procedure, files[0].Cadence)
	}
	if want := workflow.Path("beat"); files[0].Path != want {
		t.Errorf("path is %q, want %q", files[0].Path, want)
	}
}

// TestProjection_IsOrderedByProcedureName is what makes two projections of one
// repository the same list: the map the names come out of has no order, and this
// is where one is imposed.
func TestProjection_IsOrderedByProcedureName(t *testing.T) {
	files := projected(t, bound(map[string]string{
		"procedures/zebra.yaml":  procedure("zebra", "0 3 * * 1"),
		"procedures/apple.yaml":  procedure("apple", "0 3 * * 1"),
		"procedures/middle.yaml": procedure("middle", "0 3 * * 1"),
	}))

	var names []string
	for _, file := range files {
		names = append(names, file.Procedure)
	}
	if len(names) != 3 || names[0] != "apple" || names[1] != "middle" || names[2] != "zebra" {
		t.Errorf("projected %v, want them by name by Unicode code point", names)
	}
}

// TestProjection_TheEnvBlockIsTheBindingsOwnSlot holds the one fact this
// derivation adds to the generator: which credential variables a Procedure's
// bindings reach. The generator is held to its own bytes elsewhere; what is
// asserted here is that the block carries the Target's variable at all.
func TestProjection_TheEnvBlockIsTheBindingsOwnSlot(t *testing.T) {
	files := projected(t, bound(map[string]string{"procedures/beat.yaml": procedure("beat", "0 3 * * 1")}))
	if len(files) != 1 {
		t.Fatalf("projected %d files, want one", len(files))
	}
	if got := string(files[0].Bytes); !strings.Contains(got, "STAGING_TOKEN: ${{ secrets.STAGING_TOKEN }}") {
		t.Errorf("the generated file carries no secret for the slot its binding requires:\n%s", got)
	}
}

// TestProjection_TheVersionAndDigestAreTheOnesItWasProjectedAt is where §11
// puts the two facts a workflow's install step carries: the version the binary
// derived and the checksum frozen for it, literal in the file, with nothing
// resolved when the job runs.
//
// They are asserted against what the **caller** handed over rather than against
// what the declaration on disk says, which is the whole of what makes one
// `project` one edit: the file below still pins the version being replaced, and
// a projection that read either fact off it would generate the workflow of the
// binary that is being upgraded away from (issue #178).
func TestProjection_TheVersionAndDigestAreTheOnesItWasProjectedAt(t *testing.T) {
	files := bound(map[string]string{"procedures/beat.yaml": procedure("beat", "0 3 * * 1")})
	files["hyper.yaml"] = "kind: repository-declaration\nversion: 1.3.0\ndigest: sha256:" + strings.Repeat("00", 32) + "\n"

	loaded, err := repository.Load(write(t, files))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(verify.Projection(loaded, "1.4.0", "sha256:"+strings.Repeat("ab", 32))[0].Bytes)

	if !strings.Contains(generated, "echo '"+strings.Repeat("ab", 32)+"  hyper.tar.gz'") {
		t.Errorf("the checksum line does not carry the digest it was projected at:\n%s", generated)
	}
	if !strings.Contains(generated, "install hyper 1.4.0") {
		t.Errorf("the install step does not name the version it was projected at:\n%s", generated)
	}
}

// TestProjection_TheSameRepositoryProjectsTheSameBytes is what generate-and-
// verify rests on one level up from the generator: a projection that answered
// twice would fail a repository nobody had edited.
func TestProjection_TheSameRepositoryProjectsTheSameBytes(t *testing.T) {
	files := bound(map[string]string{
		"procedures/beat.yaml":  procedure("beat", "0 3 * * 1"),
		"procedures/again.yaml": procedure("again", "15 4 * * *"),
	})

	first, second := projected(t, files), projected(t, files)
	if len(first) != len(second) {
		t.Fatalf("projected %d files and then %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Path != second[i].Path || string(first[i].Bytes) != string(second[i].Bytes) {
			t.Errorf("%s was projected two different ways", first[i].Path)
		}
	}
}
