package cadence_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
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

// admittedExpressions are the awkward expressions §10 admits beside the eleven
// worked ones — the shapes a reader most expects a closing grammar to have shut
// out by accident. A field spelling its whole span, a step that does not divide
// it, a list of stepped ranges, a step of one and a range of one value are all
// in the grammar, and the check below has to leave every one of them alone.
var admittedExpressions = []string{
	"0-59 * * * *",
	"*/7 * * * *",
	"9-11,14-16/2 * * * *",
	"*/1 * * * *",
	"1-1 * * * *",
	"* * * * *",
	"0 0 * * 0",
	"0 0 * * 6",
	"59 23 31 12 6",
	"0 0 1 1 *",
	// A step §10 puts no upper bound on. It selects the first value of
	// the span and nothing else, which is a recurrence like any other —
	// and the reader's own overflow guard saturates rather than refusing
	// it, so the grammar is what decides and not an int's width.
	"*/10000 * * * *",
	"*/99999999999999999999 * * * *",
}

// TestFault_EveryExpressionTheGrammarAdmitsHasNone is the half that matters
// most: `cadence-malformed` is a closure, and a closure that shuts out a
// legitimate recurrence refuses an artefact nobody can repair. Every expression
// §10's own table renders, and every awkward one beside it, answers no fault.
func TestFault_EveryExpressionTheGrammarAdmitsHasNone(t *testing.T) {
	var expressions []string
	for _, worked := range workedExpressions {
		expressions = append(expressions, worked.expression)
	}
	expressions = append(expressions, admittedExpressions...)

	for _, expression := range expressions {
		if reason, malformed := cadence.Fault(expression); malformed {
			t.Errorf("%q is in the grammar and was faulted: %s", expression, reason)
		}
	}
}

// TestFault_WhatTheGrammarShutsOutIsNamed walks §10's own list of what is not in
// the grammar — a nickname, a name, a sixth field, a timezone, `?`, `L`, `W`,
// `#`, a value outside its field's span and a backwards range — and holds each
// one to a reason that names what was wrong. The substring is the claim: a
// reader handed `cadence-malformed` and nothing else has to search five fields
// for the one that is wrong.
func TestFault_WhatTheGrammarShutsOutIsNamed(t *testing.T) {
	for _, want := range []struct{ expression, reason string }{
		{"@hourly", "1 field"},
		{"@daily", "1 field"},
		{"@reboot", "1 field"},
		{"0 3 * * MON", `day of week "MON"`},
		{"0 0 1 JAN *", `month "JAN"`},
		{"0 0 * * * *", "6 fields"},
		{"0 0 * *", "4 fields"},
		{"", "no fields"},
		{"0 3 * * 1 America/New_York", "6 fields"},
		{"0 3 * * 1 UTC", "6 fields"},
		{"TZ=UTC 0 3 * * 1", "6 fields"},
		{"0 3 * * 1+05:00", `day of week "1+05:00"`},
		{"0 0 ? * *", `day of month "?"`},
		{"0 0 L * *", `day of month "L"`},
		{"0 0 15W * *", `day of month "15W"`},
		{"0 0 * * 1#2", `day of week "1#2"`},
		{"60 * * * *", "minute 60 is outside"},
		{"0 24 * * *", "hour 24 is outside"},
		{"0 0 0 * *", "day of month 0 is outside"},
		{"0 0 32 * *", "day of month 32 is outside"},
		{"0 0 1 13 *", "month 13 is outside"},
		{"0 0 * * 7", "day of week 7 is outside"},
		{"5-2 * * * *", "minute range 5-2 runs backwards"},
		{"0 0 * * 6-0", "day of week range 6-0 runs backwards"},
		{"*/0 * * * *", `minute "*/0" steps by zero`},
		{"*/MON * * * *", `minute "*/MON" is not one of`},
		{"*/2/3 * * * *", `minute "*/2/3" is not one of`},
		{"3/2 * * * *", `minute "3/2" steps over a single value`},
		{"5-70 * * * *", "minute 70 is outside"},
		{"70-5 * * * *", "minute 70 is outside"},
		{"0 3 * * 1 America/New_York", "a timezone and an offset are each a sixth"},
	} {
		reason, malformed := cadence.Fault(want.expression)
		if !malformed {
			t.Errorf("%q is outside the grammar and drew no fault", want.expression)
			continue
		}
		if !strings.Contains(reason, want.reason) {
			t.Errorf("%q faulted\n %q\nwant a reason naming %q", want.expression, reason, want.reason)
		}
	}
}

// TestFault_TheReasonIsOneSentenceAndNamesNoCode holds the seam: this package
// answers what is wrong with an expression, and `cadence-malformed`, the file
// and the line are the check's own. A reason carrying the code would be one
// half of the problem written twice.
func TestFault_TheReasonIsOneSentenceAndNamesNoCode(t *testing.T) {
	reason, malformed := cadence.Fault("@hourly")
	if !malformed {
		t.Fatal("@hourly is outside the grammar and drew no fault")
	}
	if strings.Contains(reason, "cadence-malformed") {
		t.Errorf("the reason names the code: %q", reason)
	}
	if strings.Contains(reason, "\n") {
		t.Errorf("the reason is more than one line: %q", reason)
	}
}

// TestFault_AgreesWithRead is the one invariant the two entry points have: an
// expression Read declines is one Fault names, and an expression Read glosses
// is one Fault leaves alone. Two readers of one grammar is where the day comes
// that a check refuses an artefact a review glosses.
func TestFault_AgreesWithRead(t *testing.T) {
	for _, expression := range []string{
		"0 3 * * 1", "*/7 * * * *", "0-59 * * * *", "1-1 * * * *",
		"@hourly", "0 3 * * MON", "60 * * * *", "5-2 * * * *", "0 0 * * * *",
	} {
		_, readable := cadence.Read(expression)
		_, malformed := cadence.Fault(expression)
		if readable == malformed {
			t.Errorf("%q: Read readable=%v and Fault malformed=%v — the two disagree", expression, readable, malformed)
		}
	}
}

// grammarVocabulary is the naming a five-field cron reader cannot be written
// without: the three field positions the grammar orders, and the item form it
// closes on. A second reader would declare its own, whatever else it was
// called.
var grammarVocabulary = []string{"dayOfMonth", "dayOfWeek", "monthOfYear", "itemForm"}

// TestGrammar_IsReadInOnePlace is §10's grammar held to one reader. The check
// `cadence-malformed` is (internal/artefact) writes no parser of its own: it
// asks this package what is wrong with an expression and cites the answer, and
// every surface that renders a gloss asks the same package whether there is one
// to render. A second reader is how the day comes that a `check` refuses an
// artefact a review glosses, and it is a fault nothing in a run of the tests
// would otherwise notice (issue #174).
//
// The tell is the vocabulary rather than the behaviour, because a second parser
// is caught by this test only if it exists — and one written to be caught by a
// behavioural test would have to be found first.
func TestGrammar_IsReadInOnePlace(t *testing.T) {
	var here, elsewhere int
	err := filepath.WalkDir("../..", func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.IsDir() && entry.Name() == ".git":
			return fs.SkipDir
		case entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go"):
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mine := filepath.Dir(path) == filepath.FromSlash("../../internal/cadence")
		for _, word := range grammarVocabulary {
			if !strings.Contains(string(source), word) {
				continue
			}
			if mine {
				here++
				continue
			}
			elsewhere++
			t.Errorf("%s names %s; §10's grammar has one reader, and it is internal/cadence", path, word)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if here == 0 {
		t.Fatalf("no file in internal/cadence names any of %v; the case would pass having read nothing", grammarVocabulary)
	}
}
