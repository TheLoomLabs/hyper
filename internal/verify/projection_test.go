package verify_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/problem"
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

// current is files with the projection those files ask for written into them:
// the repository as `hyper project` would leave it, which is what every case
// below varies by one edit.
//
// It projects at the version and the digest the declaration in files already
// carries, because that is what the check reads — the two facts reach the
// generator as parameters where `project` derives them, and here the fixture is
// the repository a `project` has already been run against (§11, issue #178).
func current(t *testing.T, files map[string]string) map[string]string {
	t.Helper()

	for _, file := range projected(t, files) {
		files[file.Path] = string(file.Bytes)
	}
	return files
}

// checked is one repository written into a temp directory, loaded and put
// through the whole static pass — the same call `check` and a Run's pre-flight
// make, so what these cases assert is what both of them report.
func checked(t *testing.T, files map[string]string) []problem.Problem {
	t.Helper()

	loaded, err := repository.Load(write(t, files))
	if err != nil {
		t.Fatal(err)
	}
	return verify.Repository(loaded)
}

// stale is the projection problems one pass found, and nothing else it found.
func stale(problems []problem.Problem) []problem.Problem {
	var found []problem.Problem
	for _, held := range problems {
		if held.ErrorCode == verify.CodeProjectionStale {
			found = append(found, held)
		}
	}
	return found
}

// reported says the pass found code, which is how the case above asserts what a
// declaration missing a required key earns instead of a projection problem.
func reported(problems []problem.Problem, code string) bool {
	for _, held := range problems {
		if held.ErrorCode == code {
			return true
		}
	}
	return false
}

// beating is the repository the cases below vary: one Procedure declaring a
// Cadence, and the artefacts its one Step resolves through.
func beating() map[string]string {
	return bound(map[string]string{"procedures/beat.yaml": procedure("beat", "0 3 * * 1")})
}

// TestRepository_AProjectionThatIsCurrentReportsNothing is the case every other
// one below is that case with one thing changed, and it is the one that has to
// hold first: a check that fired on a repository `project` had just written
// would be a rule nobody could clear.
func TestRepository_AProjectionThatIsCurrentReportsNothing(t *testing.T) {
	if problems := checked(t, current(t, beating())); len(problems) > 0 {
		t.Errorf("a repository whose projection is current reports %v", problems)
	}
}

// TestRepository_AWantedFileThatDoesNotStandIsStale is the first of the code's
// three shapes, and the one a repository that has never been projected is in.
// It is also where the citation is asserted: the file, no line, no field.
func TestRepository_AWantedFileThatDoesNotStandIsStale(t *testing.T) {
	found := stale(checked(t, beating()))
	if len(found) != 1 {
		t.Fatalf("reported %v, want one problem: the Procedure declares a Cadence and no file stands", found)
	}
	if want := workflow.Path("beat"); found[0].File != want {
		t.Errorf("cites %q, want %q", found[0].File, want)
	}
	if found[0].Line != 0 || found[0].Column != 0 || found[0].Field != "" {
		t.Errorf("cites %s:%d:%d %q, want the file alone: the comparison is whole-file and a diff hunk is not a citation",
			found[0].File, found[0].Line, found[0].Column, found[0].Field)
	}
	if !strings.Contains(found[0].Message, "hyper project") {
		t.Errorf("message is %q, want it to name `hyper project`, which is the remedy", found[0].Message)
	}
}

// TestRepository_AFileNoProcedureAsksForIsStale is the second shape, and both
// ways a repository gets into it: a Procedure that has dropped its `cadence:`,
// and a `hyper-*.yml` naming no Procedure at all. The namespace is answered as
// a set, so neither is a shape of leftover the rule has to recognise.
func TestRepository_AFileNoProcedureAsksForIsStale(t *testing.T) {
	files := current(t, beating())
	files["procedures/beat.yaml"] = procedure("beat", "")
	files[workflow.Path("ghost")] = "# a file naming no Procedure at all\n"

	found := stale(checked(t, files))
	if len(found) != 2 {
		t.Fatalf("reported %v, want two problems: a dropped Cadence's file and one naming no Procedure", found)
	}
	for _, held := range found {
		if !strings.Contains(held.Message, "no Procedure asks for this file") {
			t.Errorf("%s: message is %q, want it to say no Procedure asks for the file", held.File, held.Message)
		}
	}
}

// TestRepository_OneByteChangedAnywhereIsStale is what whole-file and
// byte-exact means: the comparison has no notion of a significant line, so a
// hand-edit to a generated file is caught wherever in it the reader made one.
//
// The runner label is the edit because it is the one a reader actually makes —
// *let me just bump the image* — and it is one of §11's four compiled-in
// constants, so this test names a string internal/workflow owns and cannot
// import (ADR-0046). **That the edit happened is therefore asserted before its
// effect is**: a constant that moved would leave strings.Replace matching
// nothing, and the assertion below would then report a clean tree and blame the
// check for a file nobody edited.
func TestRepository_OneByteChangedAnywhereIsStale(t *testing.T) {
	files := current(t, beating())
	generated := files[workflow.Path("beat")]
	edited := strings.Replace(generated, "ubuntu-24.04", "ubuntu-22.04", 1)
	if edited == generated {
		t.Fatal("no generated file carries the runner label this edits; §11's constant has moved and this test's own edit is what needs updating, not the check below")
	}
	files[workflow.Path("beat")] = edited

	if found := stale(checked(t, files)); len(found) != 1 {
		t.Fatalf("reported %v, want the one file whose bytes were edited", found)
	}
}

// TestRepository_ACadenceEditedAndNotProjectedIsStale is the drift the check
// exists for: the artefact moved and the file derived from it did not, which is
// the half of generate-and-verify a generator alone cannot state.
func TestRepository_ACadenceEditedAndNotProjectedIsStale(t *testing.T) {
	files := current(t, beating())
	files["procedures/beat.yaml"] = procedure("beat", "15,45 * * * *")

	found := stale(checked(t, files))
	if len(found) != 1 {
		t.Fatalf("reported %v, want the one file whose Cadence moved under it", found)
	}
	if !strings.Contains(found[0].Message, "hyper project") {
		t.Errorf("message is %q, want it to name `hyper project`", found[0].Message)
	}
}

// TestRepository_AHandEditedPinInsideAGeneratedFileIsStale is the version pin
// caught **twice**, here and by the fetched binary's own gate — neither
// detection depending on the other having run (§11, ADR-0020).
func TestRepository_AHandEditedPinInsideAGeneratedFileIsStale(t *testing.T) {
	for _, edit := range []struct{ what, from, to string }{
		{"the version", "1.4.0", "1.3.0"},
		{"the digest", strings.Repeat("0", 64), strings.Repeat("a", 64)},
	} {
		t.Run(edit.what, func(t *testing.T) {
			// Every occurrence, which for the version is the four
			// places §11 counts — the header comment, the install
			// step's name, the release tag and the artefact's
			// filename — so what the case edits is the pin rather
			// than one line that happens to mention it.
			files := current(t, beating())
			path := workflow.Path("beat")
			files[path] = strings.ReplaceAll(files[path], edit.from, edit.to)

			if found := stale(checked(t, files)); len(found) != 1 {
				t.Fatalf("reported %v, want %s edited inside the generated file to be caught", found, edit.what)
			}
		})
	}
}

// TestRepository_ADeclarationWithNoLegiblePinReportsItsOwnFaultAlone is the one
// place this rule declines to have an opinion. The version and the digest are
// what a regeneration is derived from, so a declaration carrying neither has
// nothing to regenerate against — and that is the declaration's own
// `schema-mismatch`, already reported. A second opinion would put two rows on
// the page for one fault (§3, §4, ADR-0064).
func TestRepository_ADeclarationWithNoLegiblePinReportsItsOwnFaultAlone(t *testing.T) {
	for _, declaration := range map[string]string{
		"no version": "kind: repository-declaration\ndigest: sha256:" + strings.Repeat("0", 64) + "\n",
		"no digest":  "kind: repository-declaration\nversion: 1.4.0\n",
		"neither":    "kind: repository-declaration\n",
	} {
		files := beating()
		files["hyper.yaml"] = declaration

		problems := checked(t, files)
		if found := stale(problems); len(found) > 0 {
			t.Errorf("%q: reported %v, want the declaration's own fault alone", declaration, found)
		}
		if !reported(problems, "schema-mismatch") {
			t.Errorf("%q: reported %v, want the declaration's own schema fault: a required key it does not carry", declaration, problems)
		}
	}
}
