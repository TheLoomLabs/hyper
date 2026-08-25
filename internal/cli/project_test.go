package cli_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
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
// criterion the tree goldens can only half state: they render
// `.github/workflows/` and `hyper.yaml` and say nothing about the rest of the
// repository, so what says the rest is untouched is a weighing of it either side
// of the command.
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
			t.Errorf("%s moved; `project` writes inside %s and into %s, and nowhere else", path, workflow.Dir, repository.DeclarationPath)
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
		if slash := filepath.ToSlash(rel); !strings.HasPrefix(slash, workflow.Dir+"/") {
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
