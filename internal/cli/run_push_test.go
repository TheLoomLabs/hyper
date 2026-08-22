package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **The push rhythm splits by Kind** (§6, §7, ADR-0006, issues #138, #148): an
// effectful Run pushes at every Step boundary and a read-only Run batches to its
// end.
//
// No golden can hold this. One push at the end and a push per Step leave the
// remote holding the same commits, so what a branch golden renders is identical
// either way — testdata/run/README.md says so in as many words, and this file
// is what it now says it instead of.
//
// What tells the two apart is how many times the remote was reached, so that is
// what is counted: a receive hook on the bare origin appends a line per push and
// accepts it. A Run of **two** `read` Steps must leave one line behind, and a
// Run of two `mutate` Steps three — the sync that is the push of `run.json`,
// then one per Step, the last Step's going out with `outcome.json`. Two is the
// smallest number of Steps that can tell the rhythms apart, which is why both
// cases this drives are two-Step Runs.

// TestRunPush_AReadOnlyRunsPushesBatchToItsEnd counts the reaches.
func TestRunPush_AReadOnlyRunsPushesBatchToItsEnd(t *testing.T) {
	reaches := reachesTheRemote(t, "two-read-steps-push-once", 2)
	if reaches != 1 {
		t.Errorf("the Run reached the remote %d times, want 1 — a read-only Run's pushes batch to its end", reaches)
	}
}

// TestRunPush_AnEffectfulRunPushesAtEveryStepBoundary counts the same reaches
// on the other arm, and three is the whole of the claim: the entry goes out
// before the gates, Step 1's writes go out before Step 2 starts, and Step 2's go
// out with `outcome.json`. Nothing a runner writes is ever more than one Step
// behind the remote, which is what makes §6's *a crash loses at most the Step in
// flight* a fact about when the pushes happen (§7, ADR-0006).
func TestRunPush_AnEffectfulRunPushesAtEveryStepBoundary(t *testing.T) {
	reaches := reachesTheRemote(t, "two-effectful-steps-push-three-times", 2)
	if reaches != 3 {
		t.Errorf("the Run reached the remote %d times, want 3 — the sync, then one per Step", reaches)
	}
}

// reachesTheRemote drives one corpus case with a counting receive hook on its
// bare origin and answers how many pushes reached it.
//
// The hook is installed after the fixture is built and before the command runs,
// so what it counts is the Run's own pushes and none of the fixture's. It writes
// into the bare repository, which is the directory a receive hook is run from.
//
// steps is how many Steps the case's Run holds, checked against the narration
// rather than assumed: a rhythm is only a claim where there is more than one
// Step, and a case whose Procedure quietly lost one would assert nothing while
// still passing.
func reachesTheRemote(t *testing.T, name string, steps int) int {
	t.Helper()

	c := corpusCase(t, "run/"+name)
	invocation := c.invocation(t)

	tally := filepath.Join(invocation.fixture.origin, "pushes")
	hook := "#!/bin/sh\necho reached >> pushes\nexit 0\n"
	if err := os.WriteFile(filepath.Join(invocation.fixture.origin, "hooks", "pre-receive"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t)); exit != 0 {
		t.Fatalf("exit = %d; stderr: %s", exit, stderr.String())
	}
	if narrated := strings.Count(stderr.String(), "step "); narrated != steps {
		t.Fatalf("stderr narrates %d Steps, want %d — a rhythm is only a claim where there is more than one Step", narrated, steps)
	}

	counted, err := os.ReadFile(tally)
	if err != nil {
		t.Fatalf("reading what the hook counted: %v", err)
	}
	return len(strings.Fields(string(counted)))
}
