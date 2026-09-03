package cadence

// Three facts about how the executor will treat a declaration render beside the
// gloss, **wherever the gloss renders** (§10) — the same total rule the gloss
// itself stands under, and one no surface is exempt from. They are derived
// here, from the same five fields the phrase and the rate are derived from and
// from nothing else, because a rule that is total is one no consumer may hold a
// second copy of — and because the two conditional ones are readings of the
// minute field, which is this package's subject and no page's.
//
// **None is a problem with the artefact.** None carries an `error_code`, none
// fails `check`, and none is a claim `hyper` makes about what the world holds
// (§12, ADR-0026). They are not parts of the gloss either: they stand *beside*
// it, on the footing *last ran* stands on, with the human doing the reading —
// which is why they are a door of their own here rather than three more members
// of Gloss.
//
// **They carry no wire member.** §9 closes the `artefact` row, the `code` row
// and the `workflow` row at the gloss's three parts, and all three facts are
// derived from `cadence` and `phrase`, which are already on the wire — so a
// consumer derives them exactly as a page does, and no consumer and no page can
// disagree about them (§8, §10).

// The three facts, as the surfaces render them.
const (
	// defaultBranchFact is unconditional. A scheduled workflow runs on the
	// default branch only, so a Cadence on a feature branch is inert, and
	// saying so where the Cadence renders is what keeps that from being
	// discovered three weeks later.
	defaultBranchFact = "scheduled runs happen on the default branch only"
	// withinTheHourFact stands where the minute field selects more than
	// one value, which is exactly where the expression declares two
	// occurrences less than sixty minutes apart.
	//
	// It is the one fact here that was measured rather than read off the
	// executor's documentation. A repository carrying four workflows at
	// `*/5 * * * *` was watched for twenty hours: each of the four had 2.5%
	// of its declared occurrences delivered, and the gaps between the ones
	// that arrived ran from 1.9 to 5.1 hours and moved together across all
	// four — a repository polled every few hours rather than an expression
	// honoured (ADR-0139). An author writing `*/5` is not writing a
	// five-minute recurrence; they are writing one the executor delivers on
	// the order of once every few hours, and the number beside it — the
	// rate, at 8,766 runs per month — is the declaration and not a forecast.
	//
	// Nothing here narrows the expression to what will be delivered, for
	// the reason `project` does not move a time off the hour: a generated
	// recurrence nobody wrote is a small lie in a file whose whole value is
	// that it says what was declared (§10).
	withinTheHourFact = "more than one run an hour is more than the executor delivers — most occurrences will never fire"
	// hourBoundaryFact stands where the minute field selects `0`. The
	// executor names the start of every hour as its high-load window, so a
	// Cadence landing there is the one likeliest to be delayed or dropped.
	//
	// Nothing adjusts a declared time to dodge it. `project` writes the
	// expression that was declared, a generated time nobody wrote being a
	// small lie in a file whose whole value is that it says what was
	// declared (§10) — so the fact is stated and the choice is left with
	// the reader, which is the only move a surface admitting no claims of
	// its own has.
	hourBoundaryFact = ":00 is the executor's busiest minute — delivery there is likeliest to be delayed or dropped"
)

// Facts are the three, in the order every surface renders them: the
// default-branch fact, then — where the minute field selects more than one
// value — the within-the-hour one, then — where it selects `0` — the
// hour-boundary one. The order is fixed here because no surface sorts them and
// every surface renders the same set — the three that render a gloss today, and
// `project`'s rows, which are the fourth (cadence.go).
//
// The order is **decreasing blast radius**, which is what makes it derivable
// rather than remembered, and what decided where the second one went in: the
// branch fact says the Cadence may not fire at all, the within-the-hour one
// says most of its occurrences will not, and the hour-boundary one says one of
// them may be late. A reader who reads one line reads the largest.
//
// An expression outside §10's grammar carries none. The facts render beside a
// gloss and there is no gloss for one the grammar does not admit, so the empty
// answer is the same answer Read gives, reached the same way — and a readable
// expression always carries at least the default-branch fact, so nothing but an
// unreadable one is ever empty.
func Facts(expression string) []string {
	parsed, _, ok := parse(expression)
	if !ok {
		return nil
	}
	facts := []string{defaultBranchFact}
	if parsed.repeatsWithinTheHour() {
		facts = append(facts, withinTheHourFact)
	}
	if parsed.landsOnTheHour() {
		facts = append(facts, hourBoundaryFact)
	}
	return facts
}

// landsOnTheHour says whether the minute field selects `0` at all.
//
// It reads the values the field selects rather than the form it was written in,
// which is what makes the four item forms four spellings of one question: `0`,
// a list carrying one, a range spanning it and a step opening on it all land on
// the hour, and the reader that already collapses a field to its values is the
// one that answers it. A predicate written over the forms would have to decide
// four times what one lookup decides once — and would be the place `*/7` got
// answered differently from `*/5` for a reason the grammar does not have.
//
// The lookup is indexed by the value itself and sized to the field's own span,
// so index 0 is the value `0` and is always there — the minute's span opens at
// `0`, as every field's does but the two the calendar numbers from one
// (field.taken, spans).
func (e expr) landsOnTheHour() bool {
	return e[minute].taken()[0]
}

// repeatsWithinTheHour says whether the minute field selects more than one
// value.
//
// That is the whole of the question, and it is the minute field's alone: two
// values selected in one hour are two occurrences less than sixty minutes
// apart, and one value is a declaration whose consecutive occurrences are a
// full hour apart at the closest — however many hours, days or months the
// fields below it select, since those only ever move an occurrence further from
// its neighbour. `0 0,12 * * *` runs twice a day and does not repeat within an
// hour; `0,30 3 * * 1` runs twice a year and does.
//
// It reads the values the field selects rather than the form it was written in,
// for landsOnTheHour's reason: a list, a range and a step are three spellings
// of one set of values, and the reader that already collapses a field to its
// values is the one that answers it.
func (e expr) repeatsWithinTheHour() bool {
	return len(e[minute].values()) > 1
}
