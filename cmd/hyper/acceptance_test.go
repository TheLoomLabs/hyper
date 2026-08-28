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
// checkout is not reachable from inside the seal. All three are questions with
// answers, and a harness used a handful of times a year is exactly the kind
// that rots between uses. The seal is the script's own assertion — it searches,
// from inside the namespace, for a checkout and for a `hyper` on `PATH`, and
// exits non-zero on finding either or on the search not having run at all — so
// a case that runs the script to completion has asserted it.
func TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse(t *testing.T) {
	needTools(t, "bash", "bwrap", "git", "go", "python3")
	needSeal(t)

	into := t.TempDir()
	command := exec.Command("bash", "scripts/acceptance/run.sh",
		filepath.Join("scripts", "acceptance", "tasks", "snapshot-lifecycle.md"), into)
	command.Dir = root(t)
	command.Env = append(command.Environ(), "ACCEPTANCE_SETUP_ONLY=1")
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("scripts/acceptance/run.sh: %v\n%s", err, out)
	}
	repo := filepath.Join(into, "repo")

	// The absence is the point. A task that asks for a Provider is asking for
	// the one artefact the repository has no example of, and a harness that
	// scaffolded one would be answering the question it was built to ask.
	if _, err := os.Stat(filepath.Join(repo, "providers")); !os.IsNotExist(err) {
		t.Errorf("the harness left a providers/ directory; its absence is the gap under test")
	}

	stdout, stderr, exit := run(t, filepath.Join(into, "bin", "hyper"), "check", "--repo-dir", repo)
	if exit != 0 || !strings.Contains(stdout, "no problems found") {
		t.Errorf("the repository the harness hands over does not check clean: exit %d\n%s%s", exit, stdout, stderr)
	}

	// `project` would write this file, and cannot while no release is
	// published; the harness takes the bytes from the handshake instead, the
	// two channels carrying one text (ADR-0095). What this catches is the
	// copy-that-went-stale failure that shortcut invites.
	note, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := mcp.Instructions(pinnedBy(t, repo)); string(note) != want {
		t.Errorf("the harness's AGENTS.md is not the orientation this binary holds (%d bytes against %d)", len(note), len(want))
	}
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
