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
// **What it reads and what it refuses.** The grammar here is read and not
// validated: `cadence-malformed` is §12's static check, deferred to the
// milestone that projects a Cadence into a workflow, and nothing in this
// package refuses an artefact or fails a `check`. What it does is answer
// whether the expression is one the grammar admits — an expression outside it
// has no reading, and a surface handed no gloss renders none rather than
// inventing a fallback for it.
//
// **What it never reads.** No clock, no calendar of record, and nothing about
// the environment it runs in (ADR-0066). Both halves are a function of the five
// fields and of nothing else, so a laptop and a runner render the same gloss
// for the same artefact forever.
package cadence

import "strings"

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
// check that declines such an artefact and it is not implemented here, so what
// a caller does with an unreadable expression is render no gloss and go on.
func Read(expression string) (Gloss, bool) {
	parsed, ok := parse(expression)
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
// not admit. Fields are space-separated; nothing else in the expression is.
func parse(expression string) (expr, bool) {
	fields := strings.Fields(expression)
	if len(fields) != len(spans) {
		return expr{}, false
	}
	var parsed expr
	for i, written := range fields {
		f, ok := parseField(position(i), written)
		if !ok {
			return expr{}, false
		}
		parsed[i] = f
	}
	return parsed, true
}

// parseField reads one field: an item, or a comma-separated list of them. A
// list's members are items and not merely numbers, so each is read by the same
// reader whatever the others are (§10).
func parseField(at position, written string) (field, bool) {
	f := field{at: at, star: written == "*"}
	for _, part := range strings.Split(written, ",") {
		it, ok := parseItem(at, part)
		if !ok {
			return field{}, false
		}
		f.items = append(f.items, it)
	}
	return f, true
}

// parseItem reads one of the four item forms. A step divides one of the two
// forms that span more than one value; a step over a number is not in the
// grammar, and neither is a second `/`.
func parseItem(at position, written string) (item, bool) {
	base, stepped, hasStep := strings.Cut(written, "/")
	step := 1
	if hasStep {
		n, ok := readNumber(stepped)
		if !ok || n < 1 {
			return item{}, false
		}
		step = n
	}

	span := spans[at]
	switch {
	case base == "*":
		return item{form: everyValue, from: span.low, to: span.high, step: step}, true
	case strings.Contains(base, "-"):
		low, high, _ := strings.Cut(base, "-")
		from, fromOK := readNumber(low)
		to, toOK := readNumber(high)
		if !fromOK || !toOK || from > to || from < span.low || to > span.high {
			return item{}, false
		}
		return item{form: aRange, from: from, to: to, step: step}, true
	default:
		v, ok := readNumber(base)
		if !ok || v < span.low || v > span.high || hasStep {
			return item{}, false
		}
		return item{form: oneValue, from: v, to: v, step: step}, true
	}
}

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
		n = n*10 + int(digit-'0')
		if n > 9999 {
			return 0, false
		}
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
