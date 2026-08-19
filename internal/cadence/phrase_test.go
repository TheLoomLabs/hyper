package cadence_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cadence"
)

// phraseOf reads one expression and answers its phrase, failing the case where
// the grammar admits the expression and this package did not.
func phraseOf(t *testing.T, expression string) string {
	t.Helper()
	gloss, ok := cadence.Read(expression)
	if !ok {
		t.Fatalf("%q is in the grammar and was not read", expression)
	}
	return gloss.Phrase
}

// TestPhrase_AFormSelectingOneValueIsRepairedGrammaticallyAndNoFurther is
// §10's repair: `*/1` is a step that steps over nothing and `1-1` is a range
// that spans one value, and each renders as what it selects rather than as
// `every 1 minutes` or `from :01 to :01`.
func TestPhrase_AFormSelectingOneValueIsRepairedGrammaticallyAndNoFurther(t *testing.T) {
	if got, want := phraseOf(t, "*/1 * * * *"), "every minute"; got != want {
		t.Errorf("*/1 * * * * rendered %q, want %q", got, want)
	}
	if got, want := phraseOf(t, "1-1 * * * *"), "at :01 past every hour"; got != want {
		t.Errorf("1-1 * * * * rendered %q, want %q", got, want)
	}
}

// TestPhrase_TheFormIsNeverNormalised holds the fact §10 renders awkwardly on
// purpose: `0-59` selects every minute and stays a range, because somebody
// wrote a range where `*` was meant and that is a fact about the artefact.
func TestPhrase_TheFormIsNeverNormalised(t *testing.T) {
	written := phraseOf(t, "0-59 * * * *")
	if written == phraseOf(t, "* * * * *") {
		t.Errorf("0-59 rendered as `*` does, %q; the form is never normalised", written)
	}
	if !strings.Contains(written, "from :00 to :59") {
		t.Errorf("0-59 rendered %q, want the range it was written as", written)
	}
}

// TestPhrase_OrderAndRepetitionAreNormalised is the other half of that rule:
// cron attaches no meaning to either, so a field is the set of values its items
// select and three spellings of one set render one phrase.
func TestPhrase_OrderAndRepetitionAreNormalised(t *testing.T) {
	want := phraseOf(t, "0 1,3,5 * * *")
	for _, spelling := range []string{"0 5,3,1 * * *", "0 3,1,5,3 * * *"} {
		if got := phraseOf(t, spelling); got != want {
			t.Errorf("%q rendered %q, want %q", spelling, got, want)
		}
	}
}

// TestPhrase_IntervalLanguageIsAdmittedOnlyWhereItIsExactlyTrue is ADR-0066's
// whole subject. `*/7` on minutes selects nine values and then waits four
// minutes, and `*/10` on day of month is the 1st, 11th, 21st and 31st in
// months most of which have no 31st.
func TestPhrase_IntervalLanguageIsAdmittedOnlyWhereItIsExactlyTrue(t *testing.T) {
	for _, admitted := range []struct{ expression, phrase string }{
		{"*/15 * * * *", "every 15 minutes"},
		{"0 */6 * * *", "at :00 past every 6 hours"},
	} {
		if got := phraseOf(t, admitted.expression); got != admitted.phrase {
			t.Errorf("%q rendered %q, want %q", admitted.expression, got, admitted.phrase)
		}
	}

	for _, refused := range []struct{ expression, members string }{
		{"*/7 * * * *", "at :00, :07, :14, :21, :28, :35, :42, :49 and :56 past every hour"},
		{"0 0 */10 * *", "00:00 UTC on the 1st, the 11th, the 21st and the 31st of the month"},
		{"0 0 * */3 *", "00:00 UTC every day in January, April, July and October"},
		{"0 0 * * */2", "00:00 UTC every Sunday, Tuesday, Thursday and Saturday"},
	} {
		got := phraseOf(t, refused.expression)
		if got != refused.members {
			t.Errorf("%q rendered %q, want its members %q", refused.expression, got, refused.members)
		}
		if strings.Contains(got, "every 10 ") || strings.Contains(got, "every 3 ") ||
			strings.Contains(got, "every 2 ") || strings.Contains(got, "every 7 ") {
			t.Errorf("%q rendered interval language: %q", refused.expression, got)
		}
	}
}

// TestPhrase_AListRendersItsItemsInTheFormsTheyWereWrittenIn is §10's list: a
// list's members are items and not merely numbers, so a field with four list
// members glosses to four items and a range inside one stays a range
// (ADR-0066). An item that renders several values contributes them to the
// field's own list rather than as a list inside it.
func TestPhrase_AListRendersItsItemsInTheFormsTheyWereWrittenIn(t *testing.T) {
	for _, listed := range []struct{ expression, phrase string }{
		{"9-11,14-16/2 * * * *", "every minute from :09 to :11, :14 and :16 past every hour"},
		{"0 0 1,10-12,20 * *", "00:00 UTC on the 1st, every day from the 10th to the 12th and the 20th of the month"},
	} {
		if got := phraseOf(t, listed.expression); got != listed.phrase {
			t.Errorf("%q rendered\n %q\nwant\n %q", listed.expression, got, listed.phrase)
		}
	}
}

// TestPhrase_TheTimeClauseMergesOnlyWhereTheMergeDoesNotLengthenIt is §10's
// merge rule read from both sides: one minute and an enumerating hour merge
// into clock times, and a range, a step or `*` on the hour keeps the two fields
// in separate clauses.
func TestPhrase_TheTimeClauseMergesOnlyWhereTheMergeDoesNotLengthenIt(t *testing.T) {
	for _, merged := range []struct{ expression, phrase string }{
		{"30 6 * * *", "06:30 UTC every day"},
		{"0 9,17 * * *", "09:00 and 17:00 UTC every day"},
		{"30-30 6 * * *", "06:30 UTC every day"},
	} {
		if got := phraseOf(t, merged.expression); got != merged.phrase {
			t.Errorf("%q rendered %q, want %q", merged.expression, got, merged.phrase)
		}
	}
	// The hour's half of the rule reads the spelling and the minute's reads
	// the values, which is how §10 words each of them: a range or a step on
	// the hour keeps the two clauses apart however few values it spans,
	// where a minute selecting one value merges however it was spelled.
	for _, apart := range []struct{ expression, phrase string }{
		{"0 */2 * * *", "at :00 past every 2 hours"},
		{"0,30 6 * * *", "at :00 and :30 past 06:00 UTC, every day"},
		{"0 1-1 * * *", "at :00 past 01:00 UTC, every day"},
		{"0 */60 * * *", "at :00 past 00:00 UTC, every day"},
	} {
		if got := phraseOf(t, apart.expression); got != apart.phrase {
			t.Errorf("%q rendered %q, want %q", apart.expression, got, apart.phrase)
		}
	}
}

// TestPhrase_EveryDayIsStatedOnlyWhereTheTimeClauseNamesClockTimes is the day
// clause's one exception and its limit: `0 3 * * *` would otherwise read as
// something that happens once, and `*/15 * * * *` already states its own
// recurrence.
func TestPhrase_EveryDayIsStatedOnlyWhereTheTimeClauseNamesClockTimes(t *testing.T) {
	if got, want := phraseOf(t, "0 3 * * *"), "03:00 UTC every day"; got != want {
		t.Errorf("0 3 * * * rendered %q, want %q", got, want)
	}
	for _, recurring := range []string{"*/15 * * * *", "*/7 * * * *", "0-59 * * * *", "* * * * *"} {
		if got := phraseOf(t, recurring); strings.Contains(got, "every day") {
			t.Errorf("%q rendered %q; the recurrence is stated and every day would restate it", recurring, got)
		}
	}
}

// TestPhrase_BothDayFieldsRestrictedIsADisjunctionWrittenOr is cron's most
// misread rule. *and* states the intersection, which is the wrong answer and
// the one that reads like the right one.
func TestPhrase_BothDayFieldsRestrictedIsADisjunctionWrittenOr(t *testing.T) {
	for _, disjoined := range []struct{ expression, phrase string }{
		{"0 0 1 * 1", "00:00 UTC on the 1st of the month or any Monday"},
		{"0 0 1 * 0-6", "00:00 UTC on the 1st of the month or any day from Sunday to Saturday"},
		{"0 0 1,15 * 1,4", "00:00 UTC on the 1st and the 15th of the month or any Monday and Thursday"},
	} {
		got := phraseOf(t, disjoined.expression)
		if got != disjoined.phrase {
			t.Errorf("%q rendered %q, want %q", disjoined.expression, got, disjoined.phrase)
		}
	}
}

// TestPhrase_UTCAttachesOnceAtTheFirstClauseCarryingAValueATimezoneMoves is
// §10's rule, and its one expression that names no timezone at all: `* * * * *`
// carries neither a clock nor a calendar value, and is true in every timezone.
func TestPhrase_UTCAttachesOnceAtTheFirstClauseCarryingAValueATimezoneMoves(t *testing.T) {
	for _, timezoned := range []string{"0 3 * * 1", "0 0 1 * *", "0 9-17 * * *"} {
		if got := phraseOf(t, timezoned); strings.Count(got, "UTC") != 1 {
			t.Errorf("%q rendered %q, want UTC exactly once", timezoned, got)
		}
	}
	for _, untimezoned := range []string{"* * * * *", "*/5 * * * *", "*/7 * * * *", "0-59 * * * *"} {
		if got := phraseOf(t, untimezoned); strings.Contains(got, "UTC") {
			t.Errorf("%q rendered %q; it carries no clock or calendar value and names no timezone", untimezoned, got)
		}
	}
	if got, want := phraseOf(t, "* * * * *"), "every minute"; got != want {
		t.Errorf("* * * * * rendered %q, want %q", got, want)
	}
}

// TestPhrase_TheVocabularyIsFullNamesAndOrdinals holds the words §10 fixes,
// against the abbreviations that would make a phrase look like something that
// could be pasted back into an artefact.
func TestPhrase_TheVocabularyIsFullNamesAndOrdinals(t *testing.T) {
	if got, want := phraseOf(t, "0 0 22 1 *"), "00:00 UTC on the 22nd of January"; got != want {
		t.Errorf("0 0 22 1 * rendered %q, want %q", got, want)
	}
	if got, want := phraseOf(t, "0 0 3 12 *"), "00:00 UTC on the 3rd of December"; got != want {
		t.Errorf("0 0 3 12 * rendered %q, want %q", got, want)
	}
	if got, want := phraseOf(t, "0 0 11 11 *"), "00:00 UTC on the 11th of November"; got != want {
		t.Errorf("0 0 11 11 * rendered %q, want %q", got, want)
	}
	// The names render whole, which is what a word-by-word reading holds:
	// an abbreviation is a word of its own and `Monday` is not one.
	for _, expression := range []string{"0 0 1 1 1", "0 0 22 12 5"} {
		for _, word := range strings.FieldsFunc(phraseOf(t, expression), func(r rune) bool { return r == ' ' || r == ',' }) {
			for _, abbreviated := range []string{"Mon", "Jan", "Feb", "Dec", "Fri", "am", "pm"} {
				if word == abbreviated {
					t.Errorf("%q rendered the abbreviation %q", expression, word)
				}
			}
		}
	}
}

// TestPhrase_ADayOfMonthSaysOfTheMonthOnlyWhereTheMonthClauseIsSilent is the
// clause's tail: the months it names do that work otherwise.
func TestPhrase_ADayOfMonthSaysOfTheMonthOnlyWhereTheMonthClauseIsSilent(t *testing.T) {
	if got := phraseOf(t, "0 0 1 * *"); !strings.Contains(got, "of the month") {
		t.Errorf("0 0 1 * * rendered %q, want the month clause silent and the day saying so", got)
	}
	for _, spoken := range []string{"0 0 1 2 *", "0 0 1 */3 *"} {
		if got := phraseOf(t, spoken); strings.Contains(got, "of the month") {
			t.Errorf("%q rendered %q; the months it names do that work", spoken, got)
		}
	}
}

// TestRead_TheGrammarIsWhatIsRead holds the closure the four item forms make.
// What is outside it has no reading — and refusing it is `cadence-malformed`'s
// job, which is §12's static check and is not this package's.
func TestRead_TheGrammarIsWhatIsRead(t *testing.T) {
	for _, admitted := range []string{
		"* * * * *", "0 3 * * 1", "9-11,14-16/2 * * * *", "0 0 1,15 1-3 0",
		"*/5 */4 */10 */2 */3", "59 23 31 12 6", "0-0 0-0 1-1 1-1 0-0",
	} {
		if _, ok := cadence.Read(admitted); !ok {
			t.Errorf("%q is in the grammar and was not read", admitted)
		}
	}
	for _, outside := range []string{
		"", "@hourly", "0 3 * *", "0 3 * * 1 *", "0 3 * * MON", "0 3 * JAN *",
		"0 3 * * 7", "60 * * * *", "0 24 * * *", "0 0 0 * *", "0 0 * 13 *",
		"5-1 * * * *", "0 3 * * ?", "0 3 L * *", "*/0 * * * *", "3/2 * * * *",
		"0 3 * * 1 UTC", "0 3 * * -1", "* * * * *,",
	} {
		if _, ok := cadence.Read(outside); ok {
			t.Errorf("%q is not in the grammar and was read", outside)
		}
	}
}

// TestPhrase_IsTotalOverTheGrammar drives every field form against every other
// and holds the two properties §10 states of the whole function: every
// expression the grammar admits gets a phrase, and nothing is ever truncated.
//
// It asserts the properties rather than the sentences: what each of these
// renders is the business of the cases above, and what this holds is that none
// of them falls back to something shorter.
func TestPhrase_IsTotalOverTheGrammar(t *testing.T) {
	forms := [5][]string{
		{"*", "7", "0-59", "*/5", "*/7", "9-11,14-16/2"},
		{"*", "9", "9-17", "*/2", "9,17", "1-1"},
		{"*", "29", "1-15", "*/10", "1,15,31", "30"},
		{"*", "2", "1-6", "*/3", "1,4,7,10", "6-9/2"},
		{"*", "1", "1-5", "*/2", "0,6", "2-4"},
	}
	seen := map[string]bool{}
	for _, m := range forms[0] {
		for _, h := range forms[1] {
			for _, dom := range forms[2] {
				for _, mon := range forms[3] {
					for _, dow := range forms[4] {
						expression := strings.Join([]string{m, h, dom, mon, dow}, " ")
						gloss, ok := cadence.Read(expression)
						if !ok {
							t.Fatalf("%q is in the grammar and was not read", expression)
						}
						if gloss.Phrase == "" {
							t.Fatalf("%q rendered no phrase", expression)
						}
						for _, dropped := range []string{"…", "...", "and more", "etc"} {
							if strings.Contains(gloss.Phrase, dropped) {
								t.Fatalf("%q rendered %q, which drops something", expression, gloss.Phrase)
							}
						}
						if gloss.RateText == "" || gloss.Rate < 0 {
							t.Fatalf("%q rated %q / %v", expression, gloss.RateText, gloss.Rate)
						}
						seen[gloss.Phrase] = true
					}
				}
			}
		}
	}
	if len(seen) < 5000 {
		t.Errorf("7,776 expressions rendered %d distinct phrases; the phrase is meant to be a function of the fields", len(seen))
	}
}
