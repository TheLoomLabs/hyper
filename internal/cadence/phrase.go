package cadence

import (
	"fmt"
	"sort"
	"strings"
)

// The phrase is a total function of the five fields: every expression the
// grammar admits gets one, and an expression that glosses awkwardly gets an
// awkward phrase rather than a fallback to something shorter. There is no
// phrasebook of recognised shapes, because a phrasebook stops being total
// exactly where the expression is hardest to read, which is where the gloss was
// needed (§10, ADR-0066). Nothing here truncates.
//
// It is three clauses in fixed order — the time, the day, the month — so the
// eye lands in the same place on every rendering. An unrestricted field renders
// nothing, with the one exception the day clause states below.

// phrase is the whole of it: the three clauses, `UTC` on the first one
// carrying a value a timezone moves, and the separators between them.
func (e expr) phrase() string {
	timeText, merged, namesClock := e.timeClause()
	dayText, monthText, dayNamesCalendar := e.dayAndMonthClauses(namesClock)

	// `UTC` attaches once, at the first clause carrying a clock or a
	// calendar value. `* * * * *` carries neither and therefore names no
	// timezone: naming one there would qualify a statement that is true in
	// every timezone, which is a word added.
	//
	// A minute past the hour is not such a value and neither is an
	// interval: `:07 past every hour` is the same instant in every
	// timezone, and it is only a whole clock time or a named day or month
	// that a timezone moves.
	clauses := []struct {
		text      string
		timezoned bool
	}{
		{timeText, namesClock},
		{dayText, dayNamesCalendar},
		{monthText, monthText != ""},
	}
	for i, clause := range clauses {
		if clause.text != "" && clause.timezoned {
			clauses[i].text += " UTC"
			break
		}
	}

	// The day clause is separated from a merged time clause by a space and
	// from an unmerged one by a comma: a merged clause is a run of clock
	// times that reads straight into the day it happens on, where an
	// unmerged one is a construction of its own and needs closing before
	// the next begins.
	phrase := clauses[0].text
	if clauses[1].text != "" {
		if merged {
			phrase += " " + clauses[1].text
		} else {
			phrase += ", " + clauses[1].text
		}
	}
	if clauses[2].text != "" {
		phrase += " " + clauses[2].text
	}
	return phrase
}

// timeClause is the minute and the hour, in the one of two shapes §10 admits.
//
// It merges them into clock times only where the merge does not lengthen it —
// where the minute selects one value and the hour enumerates, a single value or
// a list — the merged form having one member per hour and the hour clause
// having had that many already. A range, a step or `*` on the hour keeps the
// two fields in separate clauses, and so does a minute field selecting more
// than one value: merging further would take the cross product, and
// `*/5 9-17 * * *` is 108 clock times, which is the one way a
// structure-preserving phrase could still explode.
//
// merged says which shape it took, and namesClock says whether the clause
// carries a whole clock time — the fact the day clause and `UTC` both turn on.
func (e expr) timeClause() (text string, merged, namesClock bool) {
	minutes, hours := e[minute], e[hour]

	if at, single := minutes.singleValue(); single && hours.enumerates() {
		var times []string
		for _, h := range hours.values() {
			times = append(times, fmt.Sprintf("%02d:%02d", h, at))
		}
		return listed(times), true, true
	}

	minuteText, _ := minutes.phrase()
	// A minute clause naming values needs the preposition its idiom is
	// written with — `at :07 past every hour` — where one already headed by
	// *every* is a quantified phrase and takes none.
	if !strings.HasPrefix(minuteText, "every ") {
		minuteText = "at " + minuteText
	}

	// Where the minute clause is a complete recurrence in itself and the
	// hour is unrestricted, the hour renders nothing: `every 5 minutes` is
	// the whole sentence, and *past every hour* after it restates it. A
	// minute clause naming values or a window is not complete — `at :07` and
	// `from :00 to :59` are both within an hour they do not name — so the
	// hour stands there however unrestricted it is.
	hourText, hourNamesValue := hours.phrase()
	if hours.star && minutes.selfStanding() {
		return minuteText, false, false
	}
	return minuteText + " past " + hourText, false, hourNamesValue
}

// dayAndMonthClauses are the last two, which are read together because the
// month decides how the day of month ends: a day-of-month clause says *of the
// month* only where the month clause is silent, the months it names doing that
// work otherwise.
//
// namesClock is the time clause's, and it is what admits the day clause's one
// exception: where both day fields are `*` the clause renders *every day*, but
// only where the time clause names clock times. Where the time clause already
// recurs within the day the recurrence is stated and *every day* would restate
// it, so `0 3 * * *` is `03:00 UTC every day` and `*/15 * * * *` is
// `every 15 minutes`. Without the exception the commonest expression in the
// grammar would render `03:00 UTC` and read as something that happens once.
func (e expr) dayAndMonthClauses(namesClock bool) (day, month string, dayNamesCalendar bool) {
	dom, dow, months := e[dayOfMonth], e[dayOfWeek], e[monthOfYear]
	domSpeaks, dowSpeaks := e.restrictedDayFields()
	monthText, _ := months.phrase()
	// A month field selecting exactly one value is a singular period a day
	// of the month sits inside, which is the same shape as *the month*
	// itself: `0 0 29 2 *` is `on the 29th of February`.
	singleMonth := !months.star && len(months.values()) == 1

	// The day of month, with whatever the month attaches to it. It is built
	// before the disjunction is, because *of the month* qualifies the day of
	// month and not the whole clause: `0 0 1 * 1` is `on the 1st of the
	// month or any Monday`.
	attached := months.star || (singleMonth && domSpeaks && !dowSpeaks)
	domText := ""
	if domSpeaks {
		domText = dom.onTheDay()
		switch {
		case months.star:
			domText += " of the month"
		case attached:
			domText += " of " + monthText
		}
	}
	if !months.star && !attached {
		month = "in " + monthText
	}

	switch {
	case !domSpeaks && !dowSpeaks:
		if namesClock {
			day = "every day"
		}
	case domSpeaks && !dowSpeaks:
		day = domText
	case !domSpeaks && dowSpeaks:
		day = "every " + dow.weekdays()
	default:
		// Where both day fields are restricted the clause is a
		// disjunction, and it is written `or`, with both sides in full.
		// Never a comma and never *and* — *and* states the
		// intersection, which is the wrong answer and the one that
		// reads like the right one. This is the single most misread
		// thing in cron and the phrase spends the words on it.
		day = domText + " or any " + dow.weekdays()
	}
	return day, month, domSpeaks || dowSpeaks
}

// onTheDay is the day-of-month field as a clause: the preposition its ordinals
// are read with, and none where the field already rendered a quantified phrase
// of its own.
func (f field) onTheDay() string {
	text, _ := f.phrase()
	if strings.HasPrefix(text, "every ") {
		return text
	}
	return "on " + text
}

// weekdays is the day-of-week field with the quantifier taken off, so that the
// clause can supply its own: *every Monday* where the field is the whole day
// clause, *any Monday* where it is the right-hand side of a disjunction.
func (f field) weekdays() string {
	text, _ := f.phrase()
	return strings.TrimPrefix(text, "every ")
}

// phrase is one field rendered in the form it was written in, and namesValue
// says whether it named a value at all.
//
// Order and repetition are the only things normalised, because cron attaches no
// meaning to either: a field is the set of values its items select, so
// rendering `5,1,3` ascending drops nothing a reader could map back. The form
// is never normalised — `0-59` stays a range and glosses as one, awkwardly,
// rather than collapsing into *every minute*, because somebody wrote a range
// where `*` was meant and that is a fact about the artefact on the surface
// built to show facts about the artefact (§10, ADR-0066).
func (f field) phrase() (text string, namesValue bool) {
	var parts []string
	for _, it := range f.sorted() {
		written, named := it.phrase(f.at)
		parts = append(parts, written...)
		namesValue = namesValue || named
	}
	return listed(parts), namesValue
}

// sorted is the field's items ascending with duplicates collapsed — the only
// two things §10 normalises. Items are ordered by what they span and stepped
// by, which is an order over the items themselves rather than over the values
// they select: a list's members are items and not merely numbers, and a field
// with four list members glosses to four items (§10, ADR-0066).
func (f field) sorted() []item {
	sorted := append([]item(nil), f.items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.from != b.from {
			return a.from < b.from
		}
		if a.to != b.to {
			return a.to < b.to
		}
		return a.step < b.step
	})
	var kept []item
	for _, it := range sorted {
		if len(kept) > 0 && kept[len(kept)-1] == it {
			continue
		}
		kept = append(kept, it)
	}
	return kept
}

// phrase is one item in the form it was written in.
//
// A step is not an interval, and *every n* renders only where it is exactly
// true (ADR-0066). `*/7` on minutes selects 0, 7, 14, 21, 28, 35, 42, 49 and 56
// and then waits four minutes, so *every 7 minutes* is a false sentence rather
// than a second notation. Interval language is admitted on the minute and hour
// fields alone, where the span is fixed and its minimum is zero, and only where
// the step divides the span; it is never admitted on day of month, whose span
// varies with the month, and never on month or day of week, where naming the
// values is complete and shorter both.
//
// A form selecting exactly one value is repaired grammatically and no further:
// `1-1` renders `:01`, and a step of one renders as the form it steps over, so
// `*/1` is *every minute*.
//
// An item rendering its members contributes them to the field's own list rather
// than as a list inside it: `9-11,14-16/2` is one field selecting five values,
// and a phrase that grouped two of them under a second *and* would be reporting
// a structure cron does not have.
func (it item) phrase(at position) (written []string, namesValue bool) {
	vals := it.values()
	if len(vals) == 1 {
		return []string{spell(at, vals[0])}, true
	}
	switch it.form {
	case everyValue:
		if it.step == 1 {
			return []string{"every " + unit(at)}, false
		}
		if intervalAdmitted(at, it.step) {
			return []string{fmt.Sprintf("every %d %ss", it.step, unit(at))}, false
		}
	case aRange:
		if it.step == 1 {
			return []string{"every " + unit(at) + " from " + spell(at, it.from) + " to " + spell(at, it.to)}, true
		}
	}
	for _, v := range vals {
		written = append(written, spell(at, v))
	}
	return written, true
}

// intervalAdmitted says whether a step may be read as an interval: the minute
// and hour fields, and a step that divides 60 or 24.
func intervalAdmitted(at position, step int) bool {
	switch at {
	case minute:
		return 60%step == 0
	case hour:
		return 24%step == 0
	}
	return false
}

// singleValue is the one value a field selects, where it selects one. It reads
// the values rather than the spelling, `0-0` and `0,0` each selecting one value
// however they were written.
func (f field) singleValue() (int, bool) {
	vals := f.values()
	if len(vals) != 1 {
		return 0, false
	}
	return vals[0], true
}

// enumerates says whether a field is a value or a list of them — the shape the
// hour must be in for the time clause to merge.
//
// It reads the spelling and not the values, which is how §10 words this half of
// the rule: a range, a step or `*` on the hour keeps the two fields in separate
// clauses, and a range that happens to span one value is still a range. The
// minute's half of the rule is worded the other way — *the minute selects one
// value* — and is read the other way in singleValue below. The two are
// deliberately not one test: what the merge costs on the hour is a member per
// hour, and what it costs on the minute is the cross product.
func (f field) enumerates() bool {
	for _, it := range f.items {
		if it.form != oneValue {
			return false
		}
	}
	return len(f.items) > 0
}

// selfStanding says whether a minute field is a complete recurrence on its own
// — `every minute` or `every 5 minutes`, which name no value and need no hour
// to sit inside.
func (f field) selfStanding() bool {
	if len(f.items) != 1 || f.items[0].form != everyValue {
		return false
	}
	_, namesValue := f.items[0].phrase(f.at)
	return !namesValue
}

// unit is the noun a field's values are counted in.
func unit(at position) string {
	switch at {
	case minute:
		return "minute"
	case hour:
		return "hour"
	case monthOfYear:
		return "month"
	}
	return "day"
}

// The vocabulary is full English names and ordinals, fixed, with no locale and
// nothing to configure (ADR-0014). They are exactly the words the grammar
// refuses to read on input, and that is the point rather than a tension: a name
// is a second spelling of a number an artefact must not be written in, and a
// gloss is the reading that number has. Abbreviating them would also make the
// phrase look like something that could be pasted back in.
var (
	monthNames = []string{"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}
	weekdayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

// spell is one value of one field in that field's own vocabulary.
func spell(at position, v int) string {
	switch at {
	case minute:
		return fmt.Sprintf(":%02d", v)
	case hour:
		return fmt.Sprintf("%02d:00", v)
	case dayOfMonth:
		return "the " + ordinal(v)
	case monthOfYear:
		return monthNames[v-1]
	}
	return weekdayNames[v]
}

// ordinal is a day of the month as it is read aloud.
func ordinal(v int) string {
	suffix := "th"
	if v/10 != 1 {
		switch v % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", v, suffix)
}

// listed joins what a field or a clause enumerates, in the one form this
// package writes a list in.
func listed(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
