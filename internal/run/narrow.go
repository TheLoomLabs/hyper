package run

import (
	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// The second remediation §8's `EDIT ONE OF` table renders beside a Bound: the
// selector narrowed by one rung, **speculatively re-expanded so its count is
// on the page** (§8, issue #169).
//
// It is a read `hyper` did not have to perform, and §8 says why it is worth
// it: the alternative is a reviewer widening a `destroy` Bound for want of the
// other number. A Refusal is the entire path back (ADR-0001), and a page
// offering *raise the Bound* and nothing else offers one direction out of a
// runaway selector — the direction that makes the next Run destroy more.
//
// **It is offline.** The re-expansion walks the Store's own Record heads, the
// same walk the Expansion just made, and reaches no network and no Capability:
// a page cannot be worth a request no artefact asked for (ADR-0045). It
// therefore costs one more pass over heads already in hand.
//
// **A speculative read that fails renders nothing rather than failing the
// Run.** What stopped the Run is the Bound, and a Refusal whose remediation
// table went missing because a hypothetical could not be evaluated is a Refusal
// that lost its own answer to a question nobody asked.

// Narrowing is the narrowed selector as the `EDIT ONE OF` table renders it and
// the `remediation` row carries it: where the operand is, what it says now,
// what it would say, and what that would expand to (§8).
//
// It is the **first relative operand** the selector carries and no other. §12's
// eleven operators are not all narrowable in a direction a tool can name — an
// `equals` has no next rung, and a `starts_with` narrowed by a character is a
// guess about names rather than arithmetic — where a duration has an order and
// a direction that shrinks the set. So the proposal exists where the arithmetic
// does, and where it does not the table renders the Bound row alone rather than
// inventing a second one.
type Narrowing struct {
	// Line is where the operand is authored and Field its path in §8's
	// remediation notation. They are the predicate's own coordinate rather
	// than the Bound's: the two rows of `EDIT ONE OF` are two edits, and
	// they are on different lines.
	Line  int
	Field string
	// From is the operand as authored and To the rung proposed.
	From, To string
	// Expansion is what the proposal resolved to — the count the page
	// carries, and the whole reason this read was performed.
	Expansion int
}

// rungs is the ladder a relative operand is narrowed along: ordinary durations
// a reviewer would have written by hand, in ascending order.
//
// It is a ladder rather than arithmetic on the authored value — a doubled `14d`
// is `28d`, which is a number nobody writes — and it is closed rather than
// derived so that the proposal a Refusal makes is one a reader recognises as an
// interval rather than as a computation. What §8 requires of the proposal is
// that its count is on the page; what makes it *useful* is that it is a value
// somebody would commit.
var rungs = []string{"1m", "5m", "15m", "1h", "6h", "12h", "1d", "7d", "14d", "30d", "60d", "90d", "180d", "365d"}

// narrowedSelector is the Expansion's selector narrowed a rung and re-expanded,
// and nil where there is no rung to propose or no relative operand to move.
//
// **Which direction narrows depends on the operator**, which is the whole of
// why this is not one comparison: `older_than` reaches further back the larger
// its operand, so narrowing it is the next rung **up**; `newer_than` reaches
// further forward, so narrowing it is the next rung **down**. Proposing the
// wrong direction would hand a reviewer facing a runaway `destroy` a selector
// that reaches more.
func (r run) narrowedSelector(held expansion, authored sequenced, cited citation) *Narrowing {
	conjunct, found := firstRelative(held.Selector)
	if !found {
		return nil
	}
	authoredOperand := conjunct.Operand.Value
	proposed, exists := nextRung(authoredOperand, conjunct.Operator)
	if !exists {
		return nil
	}

	narrowed := held.Selector
	narrowed.List = withOperand(held.Selector.List, conjunct.Index, proposed)
	members, declined, err := r.seriesMembers(narrowed, authored, cited)
	if err != nil || len(declined) > 0 {
		// A speculative read that would not resolve renders nothing.
		// The Run is refused either way and by the Bound, not by this.
		return nil
	}

	// **The field names the selector and the line names the operand**, and
	// the two are not the same coordinate on purpose: what a reader narrows
	// is the population, which is `over:`, and where they type is the
	// conjunct that bounds it (§8).
	return &Narrowing{
		Line:      conjunct.Line,
		Field:     cited.selector().field,
		From:      authoredOperand,
		To:        proposed,
		Expansion: len(members),
	}
}

// firstRelative is the selector's first `older_than` or `newer_than` written
// against a duration, and false where it carries none.
//
// A **timestamp** operand is passed over rather than narrowed: it names an
// instant an author chose, and moving it a rung along a ladder of durations
// would propose a date nothing derived. `older_than: 2026-01-01T00:00:00Z` is
// an edit a reader makes knowing what the date meant.
func firstRelative(over selector) (predicate, bool) {
	for _, conjunct := range over.List {
		if conjunct.Operator != "older_than" && conjunct.Operator != "newer_than" {
			continue
		}
		if conjunct.Operand == nil || conjunct.Operand.Kind != yaml.ScalarNode {
			continue
		}
		if _, isDuration := schema.DurationSeconds(conjunct.Operand.Value); isDuration {
			return conjunct, true
		}
	}
	return predicate{}, false
}

// nextRung is the ladder's answer for one operator: the smallest rung longer
// than the authored operand under `older_than`, the largest rung shorter under
// `newer_than`, and false where the operand already stands at the end of the
// ladder in the direction that narrows.
//
// The comparison is by **length in seconds** rather than by position, so an
// operand nobody wrote as a rung — `21d`, `36h` — is placed on the ladder
// rather than falling off it.
func nextRung(operand, operator string) (string, bool) {
	authored, isDuration := schema.DurationSeconds(operand)
	if !isDuration {
		return "", false
	}

	if operator == "newer_than" {
		for i := len(rungs) - 1; i >= 0; i-- {
			if seconds, _ := schema.DurationSeconds(rungs[i]); seconds < authored {
				return rungs[i], true
			}
		}
		return "", false
	}
	for _, rung := range rungs {
		if seconds, _ := schema.DurationSeconds(rung); seconds > authored {
			return rung, true
		}
	}
	return "", false
}

// withOperand is the predicate list with one conjunct's operand replaced, the
// rest untouched and in their authored order.
//
// The replacement is a **new node** rather than a write into the authored one:
// the artefact's own nodes are what every other reading of this Step resolves
// from, and a speculative expansion that mutated one would leave the Step's
// entry holding a selector nobody wrote.
func withOperand(list []predicate, index int, operand string) []predicate {
	narrowed := make([]predicate, len(list))
	copy(narrowed, list)
	for i := range narrowed {
		if narrowed[i].Index != index {
			continue
		}
		narrowed[i].Operand = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: operand}
	}
	return narrowed
}
