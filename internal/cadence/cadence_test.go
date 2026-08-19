package cadence_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cadence"
)

// workedExpressions is §10's own table of eleven expressions and the phrase
// each one renders, copied byte for byte from the specification. It is the
// independent source of truth this package is held to: every entry was written
// before the code was, and no entry is derived from what the code does.
var workedExpressions = []struct{ expression, phrase string }{
	{"0 3 * * 1", "03:00 UTC every Monday"},
	{"0 0 1 * *", "00:00 UTC on the 1st of the month"},
	{"0 0 * * *", "00:00 UTC every day"},
	{"*/5 * * * *", "every 5 minutes"},
	{"*/7 * * * *", "at :00, :07, :14, :21, :28, :35, :42, :49 and :56 past every hour"},
	{"0-59 * * * *", "every minute from :00 to :59 past every hour"},
	{"0 9,17 * * 1-5", "09:00 and 17:00 UTC every day from Monday to Friday"},
	{"0 9-17 * * *", "at :00 past every hour from 09:00 to 17:00 UTC, every day"},
	{"0 0 1 * 1", "00:00 UTC on the 1st of the month or any Monday"},
	{"0 0 1 */3 *", "00:00 UTC on the 1st in January, April, July and October"},
	{"0 0 29 2 *", "00:00 UTC on the 29th of February"},
}

// TestRead_TheWorkedExpressionsRenderTheirStatedPhrase is §10's table held as a
// contract. A phrase that differs by a byte is a different sentence.
func TestRead_TheWorkedExpressionsRenderTheirStatedPhrase(t *testing.T) {
	for _, worked := range workedExpressions {
		gloss, ok := cadence.Read(worked.expression)
		if !ok {
			t.Errorf("%q is in the grammar and was not read", worked.expression)
			continue
		}
		if gloss.Phrase != worked.phrase {
			t.Errorf("%q rendered\n %q\nwant\n %q", worked.expression, gloss.Phrase, worked.phrase)
		}
	}
}

// TestRead_TheExpressionIsCarriedAsWritten holds the part the gloss does not
// derive: what the artefact wrote is what the gloss carries.
func TestRead_TheExpressionIsCarriedAsWritten(t *testing.T) {
	gloss, ok := cadence.Read("0 3 * * 1")
	if !ok {
		t.Fatal("0 3 * * 1 is in the grammar and was not read")
	}
	if gloss.Expression != "0 3 * * 1" {
		t.Errorf("the expression is %q, want the five fields as written", gloss.Expression)
	}
}

// TestRead_TheRateIsCountedOverTheGregorianCycle drives the rate through the
// values §10 and ADR-0066 state for them: 20,871 Mondays over 400 years, one
// run a month exactly on the 1st, 97 leap days over the cycle. The rendered
// text and the number are one rounding, so both are asserted together.
func TestRead_TheRateIsCountedOverTheGregorianCycle(t *testing.T) {
	for _, want := range []struct {
		expression string
		text       string
		rate       float64
	}{
		{"0 3 * * 1", "≈4.3 runs/month", 4.3},
		{"0 0 1 * *", "1 run/month", 1},
		{"0 0 * * *", "≈30 runs/month", 30},
		{"*/5 * * * *", "≈8800 runs/month", 8800},
		{"0 0 29 2 *", "≈0.020 runs/month", 0.02},
		{"* * * * *", "≈44000 runs/month", 44000},
	} {
		gloss, ok := cadence.Read(want.expression)
		if !ok {
			t.Errorf("%q is in the grammar and was not read", want.expression)
			continue
		}
		if gloss.RateText != want.text {
			t.Errorf("%q rated %q, want %q", want.expression, gloss.RateText, want.text)
		}
		if gloss.Rate != want.rate {
			t.Errorf("%q carried %v on the wire, want %v", want.expression, gloss.Rate, want.rate)
		}
	}
}

// TestGloss_NothingReadsAClock is §10's claim held structurally rather than by
// habit: the gloss is a function of the five fields and of nothing else, so a
// laptop and a runner render the same rate for the same artefact forever, and
// the number does not change on 1 January with no edit anywhere (ADR-0066).
//
// A package that never reaches the standard library's clock cannot read one,
// and holding the import is what makes that checkable in one place rather than
// re-argued at every call.
func TestGloss_NothingReadsAClock(t *testing.T) {
	parsed, err := parser.ParseDir(token.NewFileSet(), ".", func(f fs.FileInfo) bool {
		return !strings.HasSuffix(f.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for name, pkg := range parsed {
		for file, tree := range pkg.Files {
			for _, imported := range tree.Imports {
				if imported.Path.Value == `"time"` {
					t.Errorf("%s (%s) imports time; the gloss reads no clock", file, name)
				}
			}
		}
	}
	if len(parsed) == 0 {
		t.Fatal("no package was parsed; the case would pass having read nothing")
	}
}
