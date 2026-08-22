package cli_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// TestExitCodes_AreTheClosedSetOfSeven pins the numbers §12 fixes against the
// names internal/cli gives them. The wanted numbers come from §12's closed set
// rather than from the constants themselves (issue #102).
//
// Every one of the seven is now reachable. 75 and 77 are driven by the corpus;
// 130 and 143 are driven by [run_signal_test.go](run_signal_test.go), which
// hands a Run a signal from inside its first Step and holds both the code and
// the outcome it maps onto — `failed`, and no other member of the triple.
func TestExitCodes_AreTheClosedSetOfSeven(t *testing.T) {
	closedSet := []struct {
		outcome string
		got     int
		want    int
	}{
		{"the command did what it was asked", cli.ExitClean, 0},
		{"a Run the world resisted, or problems reported", cli.ExitProblems, 1},
		{"a usage error; no row stream opens", cli.ExitUsage, 2},
		{"a Run that lost the Store", cli.ExitStoreLost, 75},
		{"a guardrail declined before any effect", cli.ExitRefused, 77},
		{"a Run stopped by an interrupt, having drained", cli.ExitInterrupted, 130},
		{"a Run stopped by a termination signal", cli.ExitTerminated, 143},
	}

	// No member spans two outcomes of the triple, so no two outcomes may
	// share a number either — a constant copied from the row above it would
	// otherwise be caught only where the two numbers happen to differ.
	seen := make(map[int]string, len(closedSet))
	for _, member := range closedSet {
		if member.got != member.want {
			t.Errorf("%s: exit code = %d, want %d", member.outcome, member.got, member.want)
		}
		if other, dup := seen[member.got]; dup {
			t.Errorf("exit code %d names two outcomes: %q and %q", member.got, other, member.outcome)
		}
		seen[member.got] = member.outcome
	}
}
