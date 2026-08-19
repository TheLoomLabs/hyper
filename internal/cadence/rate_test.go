package cadence_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cadence"
)

// rateOf reads one expression and answers its rendered rate.
func rateOf(t *testing.T, expression string) string {
	t.Helper()
	gloss, ok := cadence.Read(expression)
	if !ok {
		t.Fatalf("%q is in the grammar and was not read", expression)
	}
	return gloss.RateText
}

// TestRate_TheCycleIsAWholeNumberOfWeeks is what makes 400 years the
// denominator rather than a denominator: every weekday alignment appears in it
// in its true proportion, so all seven weekdays rate the same. Under a calendar
// year they would not — a year is 52 weeks and a day or two over, and which
// weekday gets the remainder depends on the year picked (ADR-0066).
func TestRate_TheCycleIsAWholeNumberOfWeeks(t *testing.T) {
	want := rateOf(t, "0 0 * * 0")
	for _, weekday := range []string{"1", "2", "3", "4", "5", "6"} {
		if got := rateOf(t, "0 0 * * "+weekday); got != want {
			t.Errorf("day of week %s rated %q, want %q — the cycle is 20,871 weeks exactly", weekday, got, want)
		}
	}
	if want != "≈4.3 runs/month" {
		t.Errorf("a weekly Cadence rated %q, want ≈4.3 runs/month — 20,871 ÷ 4,800", want)
	}
}

// TestRate_TheCycleIsAWholeNumberOfMonths is the other half: 4,800 months, so
// a day of the month every month is one run a month exactly, whichever day it
// is, and the `≈` is absent because nothing was rounded.
func TestRate_TheCycleIsAWholeNumberOfMonths(t *testing.T) {
	for _, day := range []string{"1", "9", "17", "28"} {
		if got, want := rateOf(t, "0 0 "+day+" * *"), "1 run/month"; got != want {
			t.Errorf("the %sth of the month rated %q, want %q", day, got, want)
		}
	}
	// The 29th, 30th and 31st are the days some months do not have, and
	// the leap pattern is in the cycle in its true proportion too.
	for _, missing := range []string{"0 0 29 * *", "0 0 30 * *", "0 0 31 * *"} {
		if got := rateOf(t, missing); !strings.HasPrefix(got, "≈") {
			t.Errorf("%q rated %q; a day some months do not have is not one run a month", missing, got)
		}
	}
}

// TestRate_AnExpressionThatNeverFiresRatesZero is the grammar's far end: the
// 30th of February is a date the calendar has no instance of, and the count
// over the cycle says so exactly. It is a rate like any other rather than an
// absence — the expression is one the grammar admits, so it gets a phrase and a
// number, and nothing here refuses it (§10).
func TestRate_AnExpressionThatNeverFiresRatesZero(t *testing.T) {
	for _, never := range []string{"0 0 30 2 *", "0 0 31 4 *", "0 0 31 2 *"} {
		gloss, ok := cadence.Read(never)
		if !ok {
			t.Fatalf("%q is in the grammar and was not read", never)
		}
		if gloss.RateText != "0 runs/month" {
			t.Errorf("%q rated %q, want 0 runs/month, exactly", never, gloss.RateText)
		}
		if gloss.Rate != 0 {
			t.Errorf("%q carried %v on the wire, want 0", never, gloss.Rate)
		}
		if gloss.Phrase == "" {
			t.Errorf("%q rendered no phrase", never)
		}
	}
}

// TestRate_TheApproximationSignRendersOnlyWhereTheNumberWasRounded is what an
// exact denominator buys: the sign becomes informative, and the singular is
// used at exactly 1 and nowhere else (§10, ADR-0066).
func TestRate_TheApproximationSignRendersOnlyWhereTheNumberWasRounded(t *testing.T) {
	for _, exact := range []struct{ expression, rate string }{
		{"0 0 1 * *", "1 run/month"},
		{"0 0 1,15 * *", "2 runs/month"},
		{"0 0,12 1 */2 *", "1 run/month"},
	} {
		if got := rateOf(t, exact.expression); got != exact.rate {
			t.Errorf("%q rated %q, want %q with no ≈", exact.expression, got, exact.rate)
		}
	}
	for _, rounded := range []string{"0 3 * * 1", "0 0 * * *", "*/5 * * * *", "0 0 29 2 *"} {
		if got := rateOf(t, rounded); !strings.HasPrefix(got, "≈") {
			t.Errorf("%q rated %q, want the ≈ that says it was rounded", rounded, got)
		}
	}
}

// TestRate_TheRarestExpressionIsRenderedAtTwoSignificantFigures is why two
// figures rather than one decimal place: `0 0 29 2 *` at 0.0202 would render as
// `0.0 runs/month` beside a Procedure that does run, which is a gloss dropping
// the fact it was rendered to carry.
func TestRate_TheRarestExpressionIsRenderedAtTwoSignificantFigures(t *testing.T) {
	if got := rateOf(t, "0 0 29 2 *"); got != "≈0.020 runs/month" {
		t.Errorf("0 0 29 2 * rated %q, want ≈0.020 runs/month — not the ≈0.0 one decimal place would render", got)
	}
}

// TestRate_TheUnitIsFixedAtRunsPerMonthOnEverySurface holds the unit against
// the value: one that varied with the number would destroy the comparison the
// rate exists for (§10).
func TestRate_TheUnitIsFixedAtRunsPerMonthOnEverySurface(t *testing.T) {
	for _, expression := range []string{"0 0 29 2 *", "0 0 1 * *", "* * * * *", "0 3 * * 1"} {
		got := rateOf(t, expression)
		if !strings.HasSuffix(got, " runs/month") && !strings.HasSuffix(got, " run/month") {
			t.Errorf("%q rated %q, want the unit fixed at runs per month", expression, got)
		}
	}
}
