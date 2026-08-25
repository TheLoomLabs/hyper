package cadence_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cadence"
)

// The two facts §10 states, in the words milestone 9's own specification
// renders them in — copied byte for byte from issue #175, which is where the
// two sentences are written out and the section states them as prose. They are
// the independent source of truth this file holds the package to: a fact that
// differs by a byte is a different sentence, and both are read on a screen by a
// human deciding whether to approve a recurrence.
const (
	defaultBranchFact = "scheduled runs happen on the default branch only"
	hourBoundaryFact  = ":00 is the executor's busiest minute — delivery there is likeliest to be delayed or dropped"
)

// TestFacts_TheDefaultBranchFactIsUnconditional is the half that turns on
// nothing: a Cadence on a feature branch is inert, and every expression the
// grammar admits earns the sentence saying so.
func TestFacts_TheDefaultBranchFactIsUnconditional(t *testing.T) {
	var expressions []string
	for _, worked := range workedExpressions {
		expressions = append(expressions, worked.expression)
	}
	expressions = append(expressions, admittedExpressions...)

	for _, expression := range expressions {
		facts := cadence.Facts(expression)
		if len(facts) == 0 {
			t.Errorf("%q is in the grammar and carried no fact", expression)
			continue
		}
		if facts[0] != defaultBranchFact {
			t.Errorf("%q opened with %q, want %q", expression, facts[0], defaultBranchFact)
		}
	}
}

// TestFacts_TheHourBoundaryFactFollowsTheMinuteField is the conditional half.
// What it turns on is whether the minute field selects `0` at all — however it
// was written, since the four item forms are four spellings of one set of
// values.
func TestFacts_TheHourBoundaryFactFollowsTheMinuteField(t *testing.T) {
	for _, want := range []struct {
		expression string
		lands      bool
	}{
		// A bare `0`, and a whole clock time built on one.
		{"0 3 * * 1", true},
		{"0 0 1 * *", true},
		// A list carrying one.
		{"0,30 * * * *", true},
		{"7,0,21 * * * *", true},
		// A range whose values include it, and one whose do not.
		{"0-29 * * * *", true},
		{"1-59 * * * *", false},
		// A step over the whole span opens at the span's low value,
		// which is `0` — whether or not the step divides 60.
		{"*/5 * * * *", true},
		{"*/7 * * * *", true},
		{"*/15 * * * *", true},
		// `*` is every minute, and every minute includes `:00`.
		{"* * * * *", true},
		// A stepped range that steps over `:00`, and one that opens on
		// it.
		{"1-59/2 * * * *", false},
		{"0-59/2 * * * *", true},
		// A single value that is not `0`.
		{"5 * * * *", false},
		{"59 23 31 12 6", false},
	} {
		facts := cadence.Facts(want.expression)
		if got := slices.Contains(facts, hourBoundaryFact); got != want.lands {
			t.Errorf("%q carried the hour-boundary fact %v, want %v — facts were %q",
				want.expression, got, want.lands, facts)
		}
	}
}

// TestFacts_TheOrderIsTheOneEverySurfaceRenders holds the pair as a sequence
// rather than as a set: three surfaces render them beside a gloss and none of
// them sorts, so the order is this package's to fix once.
func TestFacts_TheOrderIsTheOneEverySurfaceRenders(t *testing.T) {
	if got := cadence.Facts("0 3 * * 1"); !slices.Equal(got, []string{defaultBranchFact, hourBoundaryFact}) {
		t.Errorf("0 3 * * 1 carried %q, want the default-branch fact then the hour-boundary one", got)
	}
	if got := cadence.Facts("5 * * * *"); !slices.Equal(got, []string{defaultBranchFact}) {
		t.Errorf("5 * * * * carried %q, want the default-branch fact alone", got)
	}
}

// TestFacts_AnExpressionWithNoGlossCarriesNone is the seam the three surfaces
// stand on: the facts render *beside* a gloss, and an expression outside §10's
// grammar has no gloss to place them beside. A surface handed no gloss renders
// none, and nothing here refuses anything — `cadence-malformed` is the check's
// (§12, internal/artefact).
func TestFacts_AnExpressionWithNoGlossCarriesNone(t *testing.T) {
	for _, expression := range []string{"@hourly", "0 3 * * MON", "0 3 * * 1 UTC", "60 * * * *", ""} {
		if facts := cadence.Facts(expression); len(facts) != 0 {
			t.Errorf("%q is outside the grammar and carried %q", expression, facts)
		}
	}
}

// TestFacts_NeitherIsAProblemWithTheArtefact holds what the two facts are not.
// Neither carries an `error_code` and neither is a claim about what the world
// holds: they are page text about how the executor will treat a declaration,
// on the footing *last ran* stands on (§10, ADR-0021, ADR-0026).
func TestFacts_NeitherIsAProblemWithTheArtefact(t *testing.T) {
	for _, fact := range cadence.Facts("0 3 * * 1") {
		for _, forbidden := range []string{"error", "warning", "invalid", "should", "must"} {
			if strings.Contains(strings.ToLower(fact), forbidden) {
				t.Errorf("%q reads as a problem with the artefact: it names %q", fact, forbidden)
			}
		}
	}
}
