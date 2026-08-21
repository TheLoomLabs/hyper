package revision_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/revision"
)

// Everything here is a fact about a real git repository built in a temp
// directory, for internal/store's own reason: this package answers what git
// says about the working tree, and the test of that is what git says.

// TestBlob is the whole claim the pure half makes: the id this computes is the
// id `git hash-object` would answer, so a Run's Provenance can be checked with
// git and never with hyper.
func TestBlob(t *testing.T) {
	r := newRepo(t)

	for name, content := range map[string]string{
		"the empty file":         "",
		"an ordinary artefact":   "kind: definition\ndefinition: uptime-check\n",
		"a file with no newline": "no trailing newline",
		"bytes outside ASCII":    "name: über-vm\n",
		"a NUL and a lone CR":    "a\x00b\rc",
	} {
		t.Run(name, func(t *testing.T) {
			if got, want := revision.Blob([]byte(content)), r.hashObject(content); got != want {
				t.Errorf("Blob = %s, want git hash-object's %s", got, want)
			}
		})
	}
}

// TestRead_HeadIsTheCommitAndACleanTreeIsNotDirty is the ordinary Run: every
// artefact it read is committed and unmodified, so `repo_revision` is HEAD and
// `repo_dirty` is written nowhere (§7).
func TestRead_HeadIsTheCommitAndACleanTreeIsNotDirty(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\n")
	r.commit()

	facts, err := revision.Read(r.root, r.read("hyper.yaml", "definitions/uptime-check.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := r.text("rev-parse", "HEAD"); facts.Head != want {
		t.Errorf("Head = %q, want HEAD at %q", facts.Head, want)
	}
	if facts.Dirty {
		t.Errorf("a repository whose every read artefact is committed reads dirty")
	}
}

// A reviewed artefact the Run read that differs from HEAD is `repo_dirty`, and
// so is one that is untracked — the two halves of §7's one sentence.
func TestRead_AModifiedOrUntrackedArtefactIsDirty(t *testing.T) {
	t.Run("modified", func(t *testing.T) {
		r := newRepo(t)
		r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\n")
		r.commit()
		r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\nkinds: [read]\n")

		facts, err := revision.Read(r.root, r.read("definitions/uptime-check.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !facts.Dirty {
			t.Errorf("an artefact differing from HEAD reads clean")
		}
	})

	t.Run("untracked", func(t *testing.T) {
		r := newRepo(t)
		r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\n")

		facts, err := revision.Read(r.root, r.read("definitions/uptime-check.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if !facts.Dirty {
			t.Errorf("an untracked artefact reads clean")
		}
	})
}

// The file set is the Run's own and not the repository's: an artefact no Step
// read is not part of the code the Run performed, so editing it moves nothing.
// That is what makes the marker and §8's catch-all count agree by construction
// rather than by two walks reaching the same answer (§7, §8).
func TestRead_AnArtefactTheRunDidNotReadDoesNotMakeItDirty(t *testing.T) {
	r := newRepo(t)
	r.write("definitions/uptime-check.yaml", "kind: definition\ndefinition: uptime-check\n")
	r.write("definitions/elsewhere.yaml", "kind: definition\ndefinition: elsewhere\n")
	r.commit()
	r.write("definitions/elsewhere.yaml", "kind: definition\ndefinition: elsewhere\nkinds: [read]\n")

	facts, err := revision.Read(r.root, r.read("definitions/uptime-check.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if facts.Dirty {
		t.Errorf("an artefact the Run never read made it dirty")
	}
}

// A repository with no commit at all has no revision to name, and the answer is
// an error rather than an empty string: `repo_revision` is a member every Run's
// Provenance carries, so a Run that cannot name one has nothing to write (§7).
func TestRead_ARepositoryWithNoCommitAnswersAnError(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &repo{t: t, root: root, env: fixtureEnvironment(t, base)}
	r.git("init", "--quiet", "--initial-branch=main")

	if _, err := revision.Read(root, nil); err == nil {
		t.Errorf("a repository with no commit answered facts; want the error that says it has no revision")
	}
}

// --- the fixture ---

type repo struct {
	t    *testing.T
	root string
	env  []string
}

func newRepo(t *testing.T) *repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Fatalf("git is not on PATH: %v; hyper reads the code branch with it and the suite assumes it (§13)", err)
	}

	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	r := &repo{t: t, root: root, env: fixtureEnvironment(t, base)}
	r.git("init", "--quiet", "--initial-branch=main")
	r.write("hyper.yaml", "kind: repository-declaration\nversion: 1.4.0\n")
	r.commit()
	return r
}

// fixtureEnvironment is stated rather than inherited so that a machine whose
// git signs every commit, or has never set user.email, builds the same
// repository here.
func fixtureEnvironment(t *testing.T, base string) []string {
	t.Helper()

	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + home,
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "absent-global-config"),
		"GIT_CONFIG_SYSTEM=" + filepath.Join(home, "absent-system-config"),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=somebody else",
		"GIT_AUTHOR_EMAIL=somebody@elsewhere.invalid",
		"GIT_COMMITTER_NAME=somebody else",
		"GIT_COMMITTER_EMAIL=somebody@elsewhere.invalid",
		"GIT_AUTHOR_DATE=2001-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2001-01-01T00:00:00Z",
		"GIT_TERMINAL_PROMPT=0",
		"TZ=UTC",
		"LC_ALL=C",
	}
}

func (r *repo) write(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *repo) commit() {
	r.t.Helper()
	r.git("add", "--all")
	r.git("commit", "--quiet", "--message", "the working tree", "--allow-empty")
}

// read is the file set a Run would hand over: the artefacts it read, with the
// bytes it read them as.
func (r *repo) read(paths ...string) []revision.File {
	r.t.Helper()

	files := make([]revision.File, 0, len(paths))
	for _, path := range paths {
		bytes, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(path)))
		if err != nil {
			r.t.Fatal(err)
		}
		files = append(files, revision.File{Path: path, Bytes: bytes})
	}
	return files
}

func (r *repo) hashObject(content string) string {
	r.t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", "hash-object", "--stdin")
	cmd.Dir = r.root
	cmd.Env = r.env
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("git hash-object: %v\n%s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

func (r *repo) git(args ...string) []byte {
	r.t.Helper()

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	cmd.Env = r.env
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes()
}

func (r *repo) text(args ...string) string {
	r.t.Helper()
	return strings.TrimSpace(string(r.git(args...)))
}
