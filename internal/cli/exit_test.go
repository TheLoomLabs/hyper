package cli_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// TestExitCodes_AreTheClosedSetOfSeven pins the numbers §12 fixes against the
// names internal/cli gives them. Three of the seven — 75, 130 and 143 — are
// unreachable until the Store and the Run exist, so no golden case can carry
// them and this is the only place a typo in one would surface before the
// milestone that reaches it (issue #102). The wanted numbers come from §12's
// closed set rather than from the constants themselves.
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
