// Package cadence reads a Cadence: §10's five-field cron grammar, and both
// halves of the gloss every surface renders one through — the phrase, which
// states the times of day, the days and the months the expression selects, and
// the rate, which states how often that comes to in runs per month.
//
// It is a package rather than a function inside a page because three surfaces
// consume it and none of them owns it: a review's header and its `FLAGS` row
// (§8), `THE CODE MOVED`'s `cadence` row (§8), and `project`'s rows (§9).
// Wherever a Cadence renders, the gloss renders with it, and there is no
// surface exempt (ADR-0005, ADR-0021, ADR-0063) — a rule that is total is one
// no consumer may hold a second copy of.
//
// **What it reads and what it refuses.** Nothing here refuses an artefact or
// fails a `check`: what the package answers is whether an expression is one the
// grammar admits, and — where it is not — what is wrong with it. Read is the
// first question and Fault the second, and they are two doors onto one reader
// because §12's `cadence-malformed` and a surface's gloss are two consumers of
// the same closure. A surface handed no gloss renders none rather than
// inventing a fallback for it; a check handed a fault cites it (§10, §12).
//
// **What it never reads.** No clock, no calendar of record, and nothing about
// the environment it runs in (ADR-0066). Both halves are a function of the five
// fields and of nothing else, so a laptop and a runner render the same gloss
// for the same artefact forever.
package cadence

import (
	"fmt"
	"strings"
)

// Gloss is one Cadence in its second reading: the expression as the artefact
// wrote it, the phrase, and the rate in the two forms one rounding produces.
//
// Rate and RateText are one number and not two facts. §10 fixes that the wire
// carries the number the page renders, rounded once, rather than the unrounded
// value beside a rounded rendering — which would be one derived fact in two
// representations that can disagree. They are derived together here so that no
// consumer is in a position to round it a second time.
//
// The composed line — the phrase, the rate and whatever a surface places beside
// them — is the surface's own. How the parts are arranged is the surface's, and
// what they are is not (§10).
type Gloss struct {
	// Expression is the Cadence exactly as the artefact wrote it. It is the
	// one member this package does not derive.
	Expression string
	// Phrase is the time, the day and the month clauses, in that order.
	Phrase string
	// Rate is the number the page renders, at the two significant figures
	// §10 rounds to — the value a `rate` key on the wire carries.
	Rate float64
	// RateText is that number rendered: the `≈` where it was rounded, the
	// digits, and the unit fixed at runs per month.
	RateText string
}

// Read is one expression read as a Gloss. It answers ok=false where the
// expression is not one §10's grammar admits — the wrong number of fields, an
// item form that is not one of the four, a value outside its field's span, or a
// range that runs backwards. That is not a refusal: `cadence-malformed` is the
// check that declines such an artefact (internal/artefact), and what a surface
// does with an unreadable expression is render no gloss and go on.
func Read(expression string) (Gloss, bool) {
	parsed, _, ok := parse(expression)
	if !ok {
		return Gloss{}, false
	}
	text, rate := renderRate(parsed.matches())
	return Gloss{
		Expression: expression,
		Phrase:     parsed.phrase(),
		Rate:       rate,
		RateText:   text,
	}, true
}

// Fault is that same reading asked the other way: what is wrong with an
// expression the grammar does not admit. It answers malformed=false for every
// expression Read glosses, and for one it declines, the reason — one sentence
// naming the field and the text at fault, so a reader handed the code does not
// have to search five fields for the one that is wrong.
//
// The reason is the whole of what this package contributes to the problem.
// `cadence-malformed`, the file, the line and the column are the check's own
// (§12): a reader that named the code here would be one half of a problem
// written twice, and this package refuses nothing.
func Fault(expression string) (reason string, malformed bool) {
	if _, reason, ok := parse(expression); !ok {
		return reason, true
	}
	return "", false
}

// position is one of the five fields, in the order they are written.
type position int

const (
	minute position = iota
	hour
	dayOfMonth
	monthOfYear
	dayOfWeek
)

// spans are the values each field admits, which is also what `*` selects
// there. Day of week runs 0 to 6 with Sunday at 0; there is no seventh
// spelling of Sunday in this grammar.
var spans = [5]struct{ low, high int }{
	{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6},
}

// fieldNames are the five as §10 names them, in the order they are written.
// They are the reader's half of a fault: a message that said *field 4* would
// make the reader count spaces to find the one it meant.
var fieldNames = [5]string{"minute", "hour", "day of month", "month", "day of week"}

// itemForm is which of the grammar's item shapes an item was written in. A
// step is not a form of its own: it is a divisor over one of these two, which
// is what makes `*/n` and `a-b/n` gloss by different rules (§10).
type itemForm int

const (
	// everyValue is `*` — the whole of the field's span.
	everyValue itemForm = iota
	// oneValue is a number.
	oneValue
	// aRange is `a-b`.
	aRange
)

// item is one member of a field: its form, the values it spans, and the step
// written over it.
//
// step is 1 where none was written, and a step of 1 is folded into that same 1
// — the two select the same values and `every 1 minutes` is not a sentence, so
// the fold is §10's grammatical repair and not a normalisation of the form.
type item struct {
	form     itemForm
	from, to int
	step     int
}

// values are the values the item selects, ascending.
func (it item) values() []int {
	var vals []int
	for v := it.from; v <= it.to; v += it.step {
		vals = append(vals, v)
	}
	return vals
}

// field is one of the five as it was written: the items, in the order the
// author wrote them, and whether the whole field is the bare `*`.
//
// star is the spelling and not the value set, because §10 decides restriction
// by the spelling: `0-6` selects every day of the week and is still a
// restriction, which is what makes `0 0 1 * 0-6` a disjunction that unions to
// every day of the month.
type field struct {
	at    position
	items []item
	star  bool
}

// expr is the five fields, in order.
type expr [5]field

// parse reads the five fields, and answers false for anything the grammar does
// not admit, with the reason beside it. Fields are space-separated; nothing
// else in the expression is, which is what makes a timezone, an offset and a
// sixth seconds field one fault rather than three — each of them is a field
// where the grammar has none.
//
// The reason travels with the answer rather than being derived from it by a
// second pass: what is wrong with an expression is known exactly where it is
// found, and a fault re-derived from a bare false is a second reader.
func parse(expression string) (parsed expr, reason string, ok bool) {
	fields := strings.Fields(expression)
	if len(fields) != len(spans) {
		return expr{}, fieldCountReason(len(fields)), false
	}
	for i, written := range fields {
		f, reason, ok := parseField(position(i), written)
		if !ok {
			return expr{}, reason, false
		}
		parsed[i] = f
	}
	return parsed, "", true
}

// fieldCountReason is the fault an expression with the wrong number of fields
// earns, naming the five so the next act is a rewrite rather than a count of
// spaces.
//
// Where there are too many it names what the extra one usually is. A seconds
// field, a timezone and an offset are each a sixth field here and each earn
// this same reason, so a message that stopped at *six fields* would leave the
// author of `0 3 * * 1 America/New_York` reading a sentence about arithmetic
// (§10, §13).
func fieldCountReason(n int) string {
	counted := fmt.Sprintf("%d fields", n)
	switch n {
	case 0:
		counted = "no fields"
	case 1:
		counted = "1 field"
	}
	reason := "it carries " + counted + " where the grammar states five: " + strings.Join(fieldNames[:], ", ")
	if n > len(spans) {
		reason += " — a seconds field, a timezone and an offset are each a sixth"
	}
	return reason
}

// parseField reads one field: an item, or a comma-separated list of them. A
// list's members are items and not merely numbers, so each is read by the same
// reader whatever the others are (§10).
func parseField(at position, written string) (parsed field, reason string, ok bool) {
	parsed = field{at: at, star: written == "*"}
	for _, part := range strings.Split(written, ",") {
		it, reason, ok := parseItem(at, part)
		if !ok {
			return field{}, reason, false
		}
		parsed.items = append(parsed.items, it)
	}
	return parsed, "", true
}

// parseItem reads one of the four item forms. A step divides one of the two
// forms that span more than one value; a step over a number is not in the
// grammar, and neither is a second `/`.
func parseItem(at position, written string) (parsed item, reason string, ok bool) {
	base, stepped, hasStep := strings.Cut(written, "/")
	step := 1
	if hasStep {
		n, isNumber := readNumber(stepped)
		switch {
		case !isNumber:
			return item{}, unknownItemForm(at, written), false
		case n < 1:
			return item{}, fmt.Sprintf(
				"%s %q steps by zero: a step is /n with n above zero",
				fieldNames[at], written,
			), false
		}
		step = n
	}

	span := spans[at]
	switch {
	case base == "*":
		return item{form: everyValue, from: span.low, to: span.high, step: step}, "", true
	case strings.Contains(base, "-"):
		low, high, _ := strings.Cut(base, "-")
		from, reason, ok := readValue(at, written, low)
		if !ok {
			return item{}, reason, false
		}
		to, reason, ok := readValue(at, written, high)
		if !ok {
			return item{}, reason, false
		}
		if from > to {
			return item{}, fmt.Sprintf(
				"%s range %s runs backwards: a-b needs a no greater than b",
				fieldNames[at], base,
			), false
		}
		return item{form: aRange, from: from, to: to, step: step}, "", true
	default:
		v, reason, ok := readValue(at, written, base)
		switch {
		case !ok:
			return item{}, reason, false
		case hasStep:
			return item{}, fmt.Sprintf(
				"%s %q steps over a single value: a step stands over * or a range a-b",
				fieldNames[at], written,
			), false
		}
		return item{form: oneValue, from: v, to: v, step: step}, "", true
	}
}

// readValue reads one value position of an item: a number, inside its own
// field's span. The three positions that hold one — a range's two ends and a
// bare number — are read here rather than each at its own site, so that
// `5-70`, `70-5` and `70` all fault the same way for the same cause.
//
// whole is the item as it was written and part is the position inside it. An
// item form outside the grammar is named by the whole — `MON-FRI` is one
// intruder and not two — where a number outside its span is named by the part,
// which is the number the author has to change.
func readValue(at position, whole, part string) (value int, reason string, ok bool) {
	n, isNumber := readNumber(part)
	span := spans[at]
	switch {
	case !isNumber:
		return 0, unknownItemForm(at, whole), false
	case n < span.low || n > span.high:
		return 0, outsideSpan(at, part), false
	}
	return n, "", true
}

// unknownItemForm is the fault a name, a nickname, `?`, `L`, `W`, `#`, a
// timezone offset and every other spelling outside the grammar earns. They are
// one reason and not eight because the grammar is closed by what it admits
// rather than by a list of what it rejects, and a message enumerating the four
// forms tells a reader what to write where a message naming the intruder only
// tells them what not to (§10).
func unknownItemForm(at position, written string) string {
	return fmt.Sprintf(
		"%s %q is not one of the grammar's four item forms: *, a number, a range a-b, or a step over either",
		fieldNames[at], written,
	)
}

// outsideSpan is the fault a legible number outside its own field's span
// earns, naming the span so the edit is the reader's next act rather than a
// look at the specification.
func outsideSpan(at position, written string) string {
	return fmt.Sprintf(
		"%s %s is outside the field's span %d–%d",
		fieldNames[at], written, spans[at].low, spans[at].high,
	)
}

// numberCeiling is where this reader stops accumulating digits. It is an
// overflow guard and no rule of the grammar, so it **saturates rather than
// refuses**: §10 puts no upper bound on a step, and a reader that declined
// `*/10000` would be refusing an expression the grammar admits.
//
// Saturating costs nothing that can be observed. Every field's span ends more
// than two orders of magnitude below it, so a saturated number is outside its
// span exactly as the number written was; and a step at or above the ceiling
// selects the first value of what it steps over and nothing else, exactly as
// any larger step would.
const numberCeiling = 10000

// readNumber reads a run of decimal digits and nothing else: no sign, no
// space, and no empty string. A name, a nickname and a `?` all fail here,
// which is where the grammar's closure is enforced.
func readNumber(written string) (int, bool) {
	if written == "" {
		return 0, false
	}
	n := 0
	for _, digit := range written {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		if n < numberCeiling {
			n = n*10 + int(digit-'0')
		}
	}
	if n > numberCeiling {
		n = numberCeiling
	}
	return n, true
}

// taken is what a field selects, as a lookup indexed by the value itself: what
// a field means is the set of values its items select together, its items being
// free to overlap, repeat and arrive in any order.
func (f field) taken() []bool {
	taken := make([]bool, spans[f.at].high+1)
	for _, it := range f.items {
		for _, v := range it.values() {
			taken[v] = true
		}
	}
	return taken
}

// values are the same set read as the values themselves, ascending and with
// duplicates collapsed.
func (f field) values() []int {
	var vals []int
	for v, selected := range f.taken() {
		if selected {
			vals = append(vals, v)
		}
	}
	return vals
}

// restrictedDayFields says which of the two day fields speak, which is the one
// question both halves of the gloss ask of them: the phrase renders a
// disjunction where both do, and the rate matches a day satisfying either.
//
// Restricted means *not `*`*, and it is the spelling that decides rather than
// the values: `0-6` selects every day of the week and is still a restriction,
// so `0 0 1 * 0-6` unions to every day of the month. Deriving it once is what
// keeps the phrase and the number it renders beside from reading POSIX's rule
// two different ways.
func (e expr) restrictedDayFields() (dayOfMonthSpeaks, dayOfWeekSpeaks bool) {
	return !e[dayOfMonth].star, !e[dayOfWeek].star
}
