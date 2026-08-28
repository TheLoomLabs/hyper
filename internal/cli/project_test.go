package cli_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// The things about `hyper project` that no case directory can state (§10, §11,
// issues #177 and #178).
//
// Everything else it does is a golden: what it wrote is testdata/project's
// `tree.golden`, what it reported is the two streams beside it. What is here is
// what a corpus cannot reach — a write the filesystem refuses part-way through,
// and a **read that never happened** — because a case directory says what a
// repository holds and not what the disk will do with it, and an absence of
// egress is not a byte anything renders.

// projectRepository writes the least a Procedure needs to project: one
// Procedure declaring a Cadence, and the artefacts its one Step resolves
// through. It is deliberately not a copy of a corpus fixture — a case here is
// about a filesystem and not about a rendering, and the repository it stands on
// only has to check clean.
func projectRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for path, content := range map[string]string{
		"hyper.yaml": "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:" + strings.Repeat("0", 64) + "\n",
		"providers/uptime.yaml": `kind: provider
provider: uptime
schema-version: 1
class: local
capabilities: [http]
operations:
  check_http:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /}
    input: {type: object, properties: {}}
    record: {identity: $.status, fields: {status: $.status}}
`,
		"targets/staging.yaml":       "kind: target-declaration\ntarget: staging\nclass: local\nkinds: [read]\ncapabilities: [http]\nhosts: [status.hyper.dev]\n",
		"definitions/heartbeat.yaml": "kind: definition\ndefinition: heartbeat\nprovider: uptime\nkinds: [read]\ntargets: [staging]\n",
		"procedures/beat.yaml": "kind: procedure\nprocedure: beat\ntargets: [staging]\ncadence: \"0 3 * * 1\"\n" +
			"steps:\n  - id: beat\n    definition: heartbeat\n    operation: check_http\n    target: staging\n",
	} {
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

// TestRunProject_AWriteThatFailsNamesTheFileItDiedOn is §10's own account of a
// projection that could not finish: exit `1`, the path on stderr, and the tree
// left as it stands — git is the undo, the tree is under review, and a rollback
// path is code that runs only when something has already gone wrong and is
// therefore the least-tested thing in the command.
//
// The failure is arranged by standing a **directory** where the file goes, which
// is a write no permission bit has to be set for and one that behaves the same
// way for every account the suite might run as.
func TestRunProject_AWriteThatFailsNamesTheFileItDiedOn(t *testing.T) {
	root := projectRepository(t)
	wanted := workflow.Path("beat")
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(wanted)), 0o755); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts)

	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — the world resisted a write", exit, cli.ExitProblems)
	}
	if !strings.Contains(stderr.String(), wanted) {
		t.Errorf("stderr = %q, want it to name %q, the file the write died on", stderr.String(), wanted)
	}
	// Once, and in the repository's own vocabulary. os names the file
	// absolutely in every *os.PathError it hands back, and the message has
	// already named it relative to the root — one fault reported as two
	// files is worse than either of them alone (§9).
	if strings.Contains(stderr.String(), root) {
		t.Errorf("stderr = %q, want the path named once and repo-relative, not again as %q", stderr.String(), root)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it silent: there is no answer to a projection that did not finish", stdout.String())
	}
}

// TestRunProject_TouchesNothingOutsideTheNamespaceAndTheDeclaration is the
// criterion the tree goldens can only half state: they render the four places
// `hyper` writes — `.github/workflows/`, `hyper.yaml`, `AGENTS.md` and
// `providers/` — and say nothing about the rest of the repository, so what says
// the rest is untouched is a weighing of it either side of the command.
//
// One of the four used to be part of what this test alone said. Since the
// goldens render `providers/`, every `project` case in the corpus states that a
// Manifest is not touched (issue #184); what is left here is `definitions/`,
// `procedures/`, `targets/` and everything else the repository holds.
//
// It weighs the artefacts rather than trusting the paths, because *which paths
// this command composes* is exactly what the criterion is about — a projection
// that resolved a Procedure's name into a path with a `..` in it would compose
// its way out of the directory it is confined to.
//
// `hyper.yaml` is weighed with the rest rather than exempted from it, and that
// is deliberate: the repository below already pins the version this binary is,
// so the two facts `project` writes into it are the two facts already there and
// the file it writes is byte-identical to the file it read. **Re-projection
// changes no byte of the declaration**, which is the property that makes a
// `project` on a current repository an empty diff rather than a whitespace one
// (§11).
//
// `AGENTS.md` cannot be weighed that way and is left out of the weighing for
// the one reason that matters: it is **created**, so there is no before to hold
// an after against. What it does instead is stated by the two cases that own it
// — the bytes it lands with, and that a file already standing is not taken
// (ADR-0095, issue #211).
func TestRunProject_TouchesNothingOutsideTheNamespaceAndTheDeclaration(t *testing.T) {
	root := projectRepository(t)
	before := treeOutsideTheNamespace(t, root)

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	after := treeOutsideTheNamespace(t, root)
	if len(before) != len(after) {
		t.Fatalf("the repository held %d files outside the namespace and now holds %d", len(before), len(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Errorf("%s moved; `project` writes inside %s, into %s and into %s, and nowhere else", path, workflow.Dir, repository.DeclarationPath, notePath)
		}
	}
}

// treeOutsideTheNamespace is every file in the repository that is not one
// `project` owns, by path, with its bytes.
func treeOutsideTheNamespace(t *testing.T, root string) map[string]string {
	t.Helper()

	held := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if slash := filepath.ToSlash(rel); !strings.HasPrefix(slash, workflow.Dir+"/") && slash != notePath {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			held[slash] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return held
}

// TestRunProject_ARepositoryThatDoesNotCheckIsLeftUntouched is the rule stated
// as a property of the **tree** rather than of the page. The corpus holds the
// problem table and the exit code; what it cannot hold is that the tree is
// untouched for a reason — the write never ran — rather than by the projection
// happening to be current.
//
// So the repository here is one whose projection is **absent and wanted**: the
// Cadence is malformed, so `check` reports, and a command that projected first
// and checked second would leave a file behind.
func TestRunProject_ARepositoryThatDoesNotCheckIsLeftUntouched(t *testing.T) {
	root := projectRepository(t)
	beat := filepath.Join(root, "procedures", "beat.yaml")
	declared, err := os.ReadFile(beat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(beat, bytes.Replace(declared, []byte(`"0 3 * * 1"`), []byte(`"0 3 * *"`), 1), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts)

	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — `project` reports what `check` would report", exit, cli.ExitProblems)
	}
	if !strings.Contains(stdout.String(), "cadence-malformed") {
		t.Errorf("stdout = %q, want `check`'s own problem table", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(workflow.Dir))); !os.IsNotExist(err) {
		t.Errorf("stat %s = %v, want it never created: nothing is written where `check` would report anything", workflow.Dir, err)
	}
}

// TestRunProject_APinThatAlreadyAgreesResolvesNothing is §11's own sentence
// about the one network read the pin ever makes: it happens **only where the
// version differs from the pin already in the declaration**. Re-projection
// resolves nothing, and the digest already there is copied into every workflow.
//
// It is asserted by counting the dials, which is the only way an absence of
// egress can be asserted rather than assumed: a `project` that fetched on every
// invocation would write exactly the same bytes and say exactly the same thing,
// and the only trace of it is a connection nobody needed.
func TestRunProject_APinThatAlreadyAgreesResolvesNothing(t *testing.T) {
	root := projectRepository(t)

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	if p.dial != 0 {
		t.Errorf("a host was dialled %d times, want none: the pin already names this binary", p.dial)
	}
	notARun(t, p)
}

// notARun holds the reads `project` never makes, whichever path it took: it
// mints no Run id, starts no child, watches no signal and never reads the
// environment whole for one.
//
// It is asserted on **both** sides of the fetch, which is where it earns its
// keep. The fetch is `hyper`'s own and not an Operation — no Capability, no
// Target, no credential, no Journal entry, no Store — and the invocation that
// makes one must be as small as the invocation that does not (§9, §11,
// ADR-0009).
//
// The Store is held by the repository rather than by a counter: neither case
// stands a git fixture at all, so a Store opened would be a Store found absent
// and the command would have declined instead of doing what it did.
func notARun(t *testing.T, p *process) {
	t.Helper()

	for _, read := range []struct {
		what  string
		count int
	}{
		{"a Run id was minted", p.mint},
		{"a child process was started", p.exec},
		{"the signals were watched", p.notify},
		{"the environment was read whole", p.environ},
	} {
		if read.count != 0 {
			t.Errorf("%s %d times, want it left alone: `project` is not a Run", read.what, read.count)
		}
	}
}

// TestRunProject_APinThatDisagreesReachesForTheChecksum is the other side of it,
// and the smallest statement of the exemption: the repository below pins a
// version this binary is not, which every other command in the tree Refuses on
// before it reads a second file — and this one loads the repository, checks it,
// and goes looking for the checksum it would freeze (§9, §11, ADR-0020).
//
// The dial fails, because the stand-in process has no network under it, so what
// is asserted is the reach rather than the answer: the corpus is where a served
// checksum is read and written down.
func TestRunProject_APinThatDisagreesReachesForTheChecksum(t *testing.T) {
	root := projectRepository(t)
	declaration := filepath.Join(root, repository.DeclarationPath)
	pinned, err := os.ReadFile(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declaration, bytes.Replace(pinned, []byte("1.4.0"), []byte("1.3.0"), 1), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts)

	if strings.Contains(stderr.String(), "version-pin") {
		t.Errorf("stderr = %q, want no pin Refusal: `project` is the pin's only writer and calls no gate", stderr.String())
	}
	if p.dial == 0 {
		t.Error("no host was dialled, want the checksum for the version this invocation would pin")
	}
	notARun(t, p)
	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — the fetch did not complete, which is the world resisting", exit, cli.ExitProblems)
	}
	// And nothing was written on the way out: the declaration still pins the
	// version it pinned, and the namespace was never created.
	after, err := os.ReadFile(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, []byte("1.3.0")) {
		t.Errorf("%s = %q, want the pin untouched: nothing is written where the checksum did not arrive", repository.DeclarationPath, after)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(workflow.Dir))); !os.IsNotExist(err) {
		t.Errorf("stat %s = %v, want it never created", workflow.Dir, err)
	}
}

// TestRunProject_TheRoundTripIsProjectThenCheck is generate-and-verify closed:
// `project` writes, `check` passes, one byte moves, `check` declines (§10,
// issue #179).
//
// It is one case rather than four because what it asserts is the **loop** —
// each half of it is held elsewhere, `project`'s bytes by testdata/project's
// tree goldens and the check's three shapes by testdata/check/projection-stale,
// and neither says that the writer and the reader agree about one repository.
// A generator and a check that were each right about a different set of bytes
// would pass both of those corpora and fail every user on the first invocation.
//
// The byte moved is inside the generated file rather than in an artefact,
// because that is the edit no other surface catches: an artefact edit shows up
// in a review's gutter, and this one shows up nowhere but here (§8, §10).
func TestRunProject_TheRoundTripIsProjectThenCheck(t *testing.T) {
	root := projectRepository(t)
	p := &process{wd: root}

	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("project exited %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exit := cli.Main([]string{"check", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("check exited %d over a repository `project` had just written: %s%s", exit, stdout.String(), stderr.String())
	}

	generated := filepath.Join(root, filepath.FromSlash(workflow.Path("beat")))
	written, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	// One byte, in the last place a reader would look for one: the `n` of
	// the runner's own name. Nothing about it is legible in a diff of the
	// repository's artefacts, and the whole-file comparison does not care
	// where it fell.
	if err := os.WriteFile(generated, bytes.Replace(written, []byte("ubuntu"), []byte("ubuntv"), 1), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if exit := cli.Main([]string{"check", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitProblems {
		t.Errorf("check exited %d after one byte moved, want %d", exit, cli.ExitProblems)
	}
	if !strings.Contains(stdout.String(), "projection-stale") {
		t.Errorf("stdout = %q, want the code the byte earned", stdout.String())
	}
	if !strings.Contains(stdout.String(), workflow.Path("beat")) {
		t.Errorf("stdout = %q, want it to cite the file that moved", stdout.String())
	}

	// And `project` runs on it, which is the exclusion the pre-write pass
	// makes for this one code: the repair is not refused on the ground that
	// there is something to repair.
	stdout.Reset()
	stderr.Reset()
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("project exited %d over a repository whose only problem is the drift it repairs: %s%s", exit, stdout.String(), stderr.String())
	}
	repaired, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repaired, written) {
		t.Errorf("the file `project` rewrote is not the file it wrote the first time:\n%s", repaired)
	}
}

// TestRunProject_WritesTheOrientationWhereTheRepositoryHasNone is the third
// place this command writes, and the reason it is a place at all (ADR-0095,
// issue #211).
//
// **The handshake is not the unconditional channel ADR-0093 took it for.** A
// client decides when it surfaces `instructions`; one harness carries them only
// inside a tool search, and a session observed against a fresh repository spent
// six of its twenty-eight calls running `strings` over the binary and then
// copied, verbatim, the Manifest it dug out of the orientation. An `AGENTS.md`
// has no such contingency — every harness reads it up front, unprompted,
// whether or not a server is configured — and `project` is where it is written
// because `project` is the documented first act on a new repository (§9, §11).
//
// **The bytes are the orientation's own, and that is what makes one text one
// text.** A hand-maintained file beside `internal/mcp`'s would disagree the
// first time either was edited, and the reader of the one that drifted has no
// way to tell which (§9).
func TestRunProject_WritesTheOrientationWhereTheRepositoryHasNone(t *testing.T) {
	root := projectRepository(t)

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	written, err := os.ReadFile(filepath.Join(root, notePath))
	if err != nil {
		t.Fatalf("the repository held no %s and `project` left none: %s", notePath, err)
	}
	// At the binary's own version, like everything else this command
	// derives: the orientation carries a Repository declaration, and a pin
	// naming any other version is one that Refuses the gate on the
	// repository the file is standing in (§11, ADR-0020).
	if string(written) != mcp.Instructions(testFacts.Version) {
		t.Error("AGENTS.md is not the orientation at the binary's version; two orientations disagree the first time either is edited")
	}
}

// TestRunProject_NeverOverwritesAnAGENTSFileThatStands is the half of the rule
// that is not the write, and the half ADR-0093 was right about: whole-file,
// always-overwriting semantics are correct for a generated workflow and wrong
// for a note addressed to a reader (§10, ADR-0095).
//
// `AGENTS.md` is a shared file most repositories already hold for reasons
// having nothing to do with `hyper`, so the one thing this command may not do
// is take it. Where one stands, the orientation reaches that agent through the
// handshake, and the text's own closing paragraph is what covers the gap: the
// agent offers to add a section, and the human decides.
func TestRunProject_NeverOverwritesAnAGENTSFileThatStands(t *testing.T) {
	root := projectRepository(t)
	standing := "# this repository\n\nRun the linter before you push.\n"
	if err := os.WriteFile(filepath.Join(root, notePath), []byte(standing), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	after, err := os.ReadFile(filepath.Join(root, notePath))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != standing {
		t.Errorf("the note moved:\n%s\nit is a note somebody wrote, and `project` creates it or leaves it alone", after)
	}
}

// TestRunProject_ARepositoryThatDoesNotCheckGetsNoOrientationEither is the
// pre-write rule read over the new path: a projection derived from a repository
// nobody could review is not written, and the note is written in the same act
// (§10, issue #179).
//
// It is stated of this path rather than left to the case above because the
// note is the one thing `project` writes that does not derive from an artefact
// — an orientation is as true of a repository that fails `check` as of one that
// passes — so *nothing is written* has to be asserted of it rather than
// inferred.
func TestRunProject_ARepositoryThatDoesNotCheckGetsNoOrientationEither(t *testing.T) {
	root := projectRepository(t)
	if err := os.WriteFile(filepath.Join(root, "definitions", "heartbeat.yaml"), []byte("kind: definition\ndefinition: heartbeat\nprovider: nothing-of-that-name\nkinds: [read]\ntargets: [staging]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"project", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d: %s%s", exit, cli.ExitProblems, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, notePath)); !os.IsNotExist(err) {
		t.Errorf("stat %[2]s = %[1]v, want it never written: the pass that stops the projection stops the whole act", err, notePath)
	}
}
