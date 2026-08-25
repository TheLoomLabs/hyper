package cli

import "github.com/TheLoomLabs/hyper/internal/cadence"

// cadenceGloss is the gloss's parts as a row carries them, and the reading that
// fills them: the three §9 closes a row at, and the two the page renders from
// but the wire derives (§8, §9, §10).
//
// It is one value rather than five members on each row because two of §9's rows
// carry exactly this group and neither may carry a different one — `review`'s
// `artefact` row and `project`'s `workflow` row (review.go, project.go). Wherever
// a Cadence renders, the gloss renders with it, and there is no surface exempt
// (ADR-0063); a rule that is total is one no consumer may hold a second copy of,
// and a second copy is what two rows reading an expression for themselves would
// be.
//
// **It is embedded rather than nested**, so the three wire members stand at the
// row's own top level in the position the row declares it: §9 fixes a row's keys
// flat, and `cadence`, `phrase` and `rate` are the row's own keys rather than a
// block beneath one.
//
// **What it does not carry is how the parts are arranged.** How the three are
// laid out is the surface's and what they are is not (§10) — a header joins them
// with `·` and hangs *last ran* off the end, a table cell stacks them — so each
// page composes its own line and this carries no rendering of its own.
type cadenceGloss struct {
	// Cadence is the expression exactly as the artefact wrote it, Phrase
	// its reading, and Rate the number at the two significant figures §10
	// rounds to. All three are absent where the row's subject declares no
	// recurrence, and absent together where it declares one the grammar
	// does not admit — a gloss is a reading of the grammar, and what is not
	// in the grammar has no reading (§9, §10).
	Cadence string   `json:"cadence,omitempty"`
	Phrase  string   `json:"phrase,omitempty"`
	Rate    *float64 `json:"rate,omitempty"`

	// rateText is the rate in the notation the page renders it in — the
	// `≈` where it was rounded, and the unit fixed at runs per month (§10).
	// It is off the wire because the wire carries the number, and it is on
	// the row because the page is written from the rows (ADR-0026): both
	// come out of one reading of the expression, so the digits the page
	// renders and the number the row carries are one rounding and cannot
	// disagree.
	rateText string
	// facts are §10's two facts about how the executor will treat the
	// declaration — the default-branch one always, the hour-boundary one
	// where the minute field selects `0` (internal/cadence).
	//
	// They are off the wire because §9 closes a row at the gloss's three
	// parts, and both are derived from `cadence` and `phrase`, which the
	// row already carries: a consumer derives them exactly as a page does,
	// so widening a row would be one derived fact carried twice (§8, §10).
	// They are on the row for rateText's reason — the page is written from
	// the rows (ADR-0026) — and they are a member of their own rather than
	// folded into rateText, neither being part of the gloss.
	facts []string
}

// read fills the parts from one expression and answers whether the grammar
// admitted it. An expression it does not leaves every member as it was, which
// is the absence a row with no Cadence beneath it already carries.
//
// The answer is returned rather than left to be inferred from an empty member,
// because one caller has a second fact hanging off the same condition: a
// review's *last ran* renders where the gloss does and nowhere else (§8, §10).
func (g *cadenceGloss) read(expression string) bool {
	gloss, readable := cadence.Read(expression)
	if !readable {
		return false
	}
	g.Cadence, g.Phrase, g.Rate = gloss.Expression, gloss.Phrase, &gloss.Rate
	g.rateText, g.facts = gloss.RateText, cadence.Facts(expression)
	return true
}
