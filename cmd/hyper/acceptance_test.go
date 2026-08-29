package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/mcp"
)

// TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse runs
// the setup half of scripts/acceptance/run.sh — everything up to and including
// the seal, and not the session behind it (issue #216).
//
// The transcripts that decide whether this tool can be used by an agent are
// only worth what the harness that produced them is worth, and the harness has
// three properties the evidence rests on: the repository it hands over is the
// quickstart shape and checks clean, its `AGENTS.md` is the orientation this
// binary holds rather than a copy that went stale, and the `hyper` source
// checkout is not reachable from inside the seal — nor, in the output directory
// the harness writes, anything but the repository and the files the sealed
// session's own processes must open. All three are questions with answers, and
// a harness used a handful of times a year is exactly the kind that rots
// between uses. The seal is the script's own assertion — it searches, from
// inside the namespace, for a checkout and for a `hyper` on `PATH`, inventories
// what is reachable in the output directory against what was deliberately bound
// back there (issue #231, ADR-0109), and exits non-zero on any of them or on
// the search not having run at all — so a case that runs the script to
// completion has asserted it.
//
// **The setup half runs in no namespace, and that is what this case needs.**
// The cover goes up around the session; everything below runs against the
// output directory on the host, where the repository the harness materialised
// and the binary it stamped are both still where the script left them.
//
// **It ranges over the tasks directory rather than naming a task** (issue
// #222). A task file and the `.setup.sh` beside it are one artefact, and the
// setup script runs on the repository an agent is handed — so a task named
// here and the rest fenced by nothing is the same rot in a narrower place:
// the second task's setup could leave a repository that does not check clean
// and the suite would stay green. Adding a task file is the whole of what
// fencing it takes, which is what makes the directory the right thing to
// drive.
//
// What each task is asserted to hold, once its own setup has run: the script
// ran to completion, which is the seal's assertion above; the setup script is
// one `run.sh` would actually run; `providers/` is absent; the repository
// checks clean; and `AGENTS.md` is the orientation this binary holds. Each
// task gets a subtest of its own so that a failure names the task that caused
// it, and an empty directory is a failure, since a loop over nothing passes
// for the wrong reason.
func TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse(t *testing.T) {
	needTools(t, "bash", "bwrap", "git", "go", "python3")
	needSeal(t)

	directory := filepath.Join(root(t), "scripts", "acceptance", "tasks")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var tasks []string
	for _, entry := range entries {
		// Every `.md` here is a task but one: a subtree documents itself
		// (CONTRIBUTING), and a `README.md` handed to `run.sh` as a task would
		// be a sealed session prompted with the directory's own prose.
		if !strings.HasSuffix(entry.Name(), ".md") || entry.Name() == "README.md" {
			continue
		}
		tasks = append(tasks, filepath.Join(directory, entry.Name()))
	}
	if len(tasks) == 0 {
		t.Fatal("scripts/acceptance/tasks holds no task file; a fence that ranges over nothing is green for the wrong reason")
	}

	for _, task := range tasks {
		t.Run(strings.TrimSuffix(filepath.Base(task), ".md"), func(t *testing.T) {
			// `run.sh` runs a task's setup script only if it is executable,
			// and skips it in silence otherwise — so a setup script committed
			// without its bit is a task whose repository is missing what the
			// task names, fenced by a loop that stays green. The mode is the
			// one part of "a task and its setup script are one artefact" the
			// harness cannot assert for itself.
			if setup := strings.TrimSuffix(task, ".md") + ".setup.sh"; !executable(t, setup) {
				t.Errorf("%s exists but is not executable; run.sh would skip it and this task would run against a repository without it", filepath.Base(setup))
			}

			into := t.TempDir()
			command := exec.Command("bash", "scripts/acceptance/run.sh", task, into)
			command.Dir = root(t)
			command.Env = append(command.Environ(), "ACCEPTANCE_SETUP_ONLY=1")
			if out, err := command.CombinedOutput(); err != nil {
				t.Fatalf("scripts/acceptance/run.sh: %v\n%s", err, out)
			}
			repo := filepath.Join(into, "repo")

			// The absence is the point, and it is asserted for every task
			// rather than only the ones that ask for a Provider: it is a
			// property of the fixture the harness materialises, and a `.setup.sh`
			// that scaffolded one would be answering the question the flagship
			// task is being written to ask.
			if _, err := os.Stat(filepath.Join(repo, "providers")); !os.IsNotExist(err) {
				t.Errorf("the harness left a providers/ directory; its absence is the gap under test")
			}

			stdout, stderr, exit := run(t, filepath.Join(into, "bin", "hyper"), "check", "--repo-dir", repo)
			if exit != 0 || !strings.Contains(stdout, "no problems found") {
				t.Errorf("the repository the harness hands over does not check clean: exit %d\n%s%s", exit, stdout, stderr)
			}

			// The harness writes this file with `hyper project`, the channel
			// ADR-0095 states, and what stands here is that the repository
			// handed over holds that note at the version it pins. What fails
			// it is a harness that stopped calling `project` or went back to
			// keeping a copy of the orientation beside itself, and a note
			// written at a version the fixture did not pin.
			//
			// **It is not the two-channels assertion, and reads as one.**
			// Both sides now derive from `mcp.Instructions`, so the handshake
			// carrying the same text is true here by construction rather than
			// by this case; `internal/mcp`'s own case is where that is fenced
			// (instructions_test.go, `InitializeResult().Instructions`).
			note, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
			if err != nil {
				t.Fatal(err)
			}
			if want := mcp.Instructions(pinnedBy(t, repo)); string(note) != want {
				t.Errorf("the harness's AGENTS.md is not the orientation this binary holds (%d bytes against %d)", len(note), len(want))
			}
		})
	}
}

// executable answers whether a task's setup script is one `run.sh` would run:
// a script that is not there is in order — a task naming nothing on the machine
// brings nothing with it — and one that is there with no executable bit is not.
func executable(t *testing.T, path string) bool {
	t.Helper()

	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return true
	case err != nil:
		t.Fatal(err)
	}
	return info.Mode().Perm()&0o111 != 0
}

// pinnedBy answers the version a Repository declaration pins, which is also
// the version the harness stamped the binary with — the two being one value is
// what keeps the version gate out of the transcript (§11).
func pinnedBy(t *testing.T, repo string) string {
	t.Helper()

	declaration, err := os.ReadFile(filepath.Join(repo, "hyper.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(declaration), "\n") {
		if pin, found := strings.CutPrefix(line, "version: "); found {
			return strings.TrimSpace(pin)
		}
	}
	t.Fatalf("the harness wrote a Repository declaration with no version:\n%s", declaration)
	return ""
}

// needSeal skips where `bwrap` is present but cannot build a namespace — a
// container with user namespaces switched off, which is a property of the
// machine rather than of anything this repository can fix.
func needSeal(t *testing.T) {
	t.Helper()

	if err := exec.Command("bwrap", "--bind", "/", "/", "--dev", "/dev", "true").Run(); err != nil {
		t.Skipf("bwrap cannot build a namespace here (%v); the acceptance harness cannot be sealed", err)
	}
}
