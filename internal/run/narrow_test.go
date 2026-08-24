package run

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// **The ladder narrows in the direction the operator reaches** (§8, issue
// #169).
//
// It is the exception the milestone's testing note names, for expand_test.go's
// own reason: `internal/run` is driven through `cli.Main` because its interface
// is a Run, and this is a pure function of two strings. What the corpus holds
// beside it is the whole rendering — [a-bound-past-a-relative-predicate] drives
// the proposal, its re-expansion and the `EDIT ONE OF` row it lands in.
//
// The direction is what this is for. `older_than` reaches further back the
// larger its operand and `newer_than` further forward, so one narrows up and
// the other down — and a proposal in the wrong direction would hand a reviewer
// facing a runaway `destroy` a selector that reaches more.

// TestNextRung walks the ladder in both directions, including the two ends
// where there is no rung to propose.
func TestNextRung(t *testing.T) {
	for _, c := range []struct {
		name            string
		operand         string
		operator        string
		want            string
		wantHasProposal bool
	}{
		{"older_than takes the next rung up", "14d", "older_than", "30d", true},
		{"an operand between rungs is placed on it", "21d", "older_than", "30d", true},
		{"an operand equal to a rung passes it", "30d", "older_than", "60d", true},
		{"the top of the ladder proposes nothing", "365d", "older_than", "", false},
		{"past the top proposes nothing", "1000d", "older_than", "", false},
		{"newer_than takes the next rung down", "30d", "newer_than", "14d", true},
		{"an operand between rungs is placed on it, downward", "20d", "newer_than", "14d", true},
		{"the bottom of the ladder proposes nothing", "1m", "newer_than", "", false},
		{"a value that is not a duration proposes nothing", "2026-01-01T00:00:00Z", "older_than", "", false},
		{"the units are compared as lengths and not as characters", "36h", "older_than", "7d", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, has := nextRung(c.operand, c.operator)
			if has != c.wantHasProposal || got != c.want {
				t.Fatalf("nextRung(%q, %q) = %q, %t; want %q, %t",
					c.operand, c.operator, got, has, c.want, c.wantHasProposal)
			}
		})
	}
}

// TestRungs_AreAscending holds the ladder in the order nextRung reads it. The
// walk is a first-match in each direction, so a rung out of place would answer
// a proposal that is not the next one — and the fault would be invisible on
// every case whose operand happens to sit before it.
func TestRungs_AreAscending(t *testing.T) {
	previous := 0
	for _, rung := range rungs {
		seconds, isDuration := schema.DurationSeconds(rung)
		if !isDuration {
			t.Fatalf("rung %q is not a duration the grammar admits", rung)
		}
		if seconds <= previous {
			t.Fatalf("rung %q is %d seconds, which does not follow %d", rung, seconds, previous)
		}
		previous = seconds
	}
}
