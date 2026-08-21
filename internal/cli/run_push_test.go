package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **A read-only Run's pushes batch to its end** (§6, §7, ADR-0006, issue #138).
//
// No golden can hold this. One push at the end and a push per Step leave the
// remote holding the same commits, so what a branch golden renders is identical
// either way — testdata/run/README.md says so in as many words, and this file
// is what it now says it instead of.
//
// What tells the two apart is how many times the remote was reached, so that is
// what is counted: a receive hook on the bare origin appends a line per push and
// accepts it, and a Run of **two** `read` Steps must leave one line behind. Two
// is the smallest number that can tell the rhythms apart, which is why the case
// this drives is the corpus's only two-Step Run.

// TestRunPush_AReadOnlyRunsPushesBatchToItsEnd counts the reaches.
func TestRunPush_AReadOnlyRunsPushesBatchToItsEnd(t *testing.T) {
	dir := filepath.Join("testdata", "run", "two-read-steps-push-once")
	c := goldenCase{dir: dir, name: "run/two-read-steps-push-once", argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)

	// The hook is installed after the fixture is built and before the
	// command runs, so what it counts is the Run's own pushes and none of
	// the fixture's. It writes into the bare repository, which is the
	// directory a receive hook is run from.
	tally := filepath.Join(invocation.fixture.origin, "pushes")
	hook := "#!/bin/sh\necho reached >> pushes\nexit 0\n"
	if err := os.WriteFile(filepath.Join(invocation.fixture.origin, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t)); exit != 0 {
		t.Fatalf("exit = %d; stderr: %s", exit, stderr.String())
	}
	if steps := strings.Count(stderr.String(), "step "); steps != 2 {
		t.Fatalf("stderr narrates %d Steps, want 2 — one push is only a claim where there is more than one Step", steps)
	}

	counted, err := os.ReadFile(tally)
	if err != nil {
		t.Fatalf("reading what the hook counted: %v", err)
	}
	if reaches := len(strings.Fields(string(counted))); reaches != 1 {
		t.Errorf("the Run reached the remote %d times, want 1 — a read-only Run's pushes batch to its end", reaches)
	}
}
