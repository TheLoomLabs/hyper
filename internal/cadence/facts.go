package cadence

// Two facts about how the executor will treat a declaration render beside the
// gloss, **wherever the gloss renders** (§10) — the same total rule the gloss
// itself stands under, and one no surface is exempt from. They are derived
// here, from the same five fields the phrase and the rate are derived from and
// from nothing else, because a rule that is total is one no consumer may hold a
// second copy of — and because the hour-boundary one is a reading of the minute
// field, which is this package's subject and no page's.
//
// **Neither is a problem with the artefact.** Neither carries an `error_code`,
// neither fails `check`, and neither is a claim `hyper` makes about what the
// world holds (§12, ADR-0026). They are not parts of the gloss either: they
// stand *beside* it, on the footing *last ran* stands on, with the human doing
// the reading — which is why they are a door of their own here rather than two
// more members of Gloss.
//
// **They carry no wire member.** §9 closes the `artefact` row, the `code` row
// and the `workflow` row at the gloss's three parts, and both facts are derived
// from `cadence` and `phrase`, which are already on the wire — so a consumer
// derives them exactly as a page does, and no consumer and no page can disagree
// about them (§8, §10).

// The two facts, as the surfaces render them.
const (
	// defaultBranchFact is unconditional. A scheduled workflow runs on the
	// default branch only, so a Cadence on a feature branch is inert, and
	// saying so where the Cadence renders is what keeps that from being
	// discovered three weeks later.
	defaultBranchFact = "scheduled runs happen on the default branch only"
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

// Facts are the two, in the order every surface renders them: the
// default-branch fact, and — where the minute field selects `0` — the
// hour-boundary one. The order is fixed here because no surface sorts them and
// every surface renders the same pair — the three that render a gloss today,
// and `project`'s rows, which are the fourth (cadence.go).
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
