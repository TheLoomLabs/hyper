package run

import (
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// A Step's `when:`: the condition §6 evaluates before the Step's Expansion
// resolves, and the skip that propagates from it (§6, §12, issue #141).
//
// **A condition reads the Records earlier Steps of this Run acted on, and
// nothing else** — not the world, not another Definition's Records, and not
// another Run's. A fact from elsewhere is a `read` Step: it costs one line, it
// records what it read, and it occupies lines the gutter annotates beside the
// Step it decides (§3). Every fact that influenced a Run is visible twice over,
// in the artefact and in the Run's own Records.
//
// **A Step whose call returned what the head already held acted on its Record
// all the same and minted nothing**, so what a condition reads is that head
// rather than nothing: a Record going unchanged is not a Record going missing
// (§7, ADR-0030). That is why what is held here is the fields each call
// concluded, taken at the Step's turn, rather than a walk of what the Run wrote.
//
// **It does not fall through to the Store.** Reading the head there would be
// the condition quietly reading another Run's Record, which is the one thing
// its rule forbids — so a Step the condition names that wrote no Record in this
// Run leaves the condition not holding, and the Step it decides is *skipped by
// condition* in its turn. **And it does not Refuse**: an earlier optional Step
// being skipped is an ordinary occurrence, and Refusing on it would make the
// Procedure un-runnable with no exit but an edit to a reviewed artefact
// (ADR-0001). A skip propagates, which is what a reader of the artefact would
// predict.

// stepKey is one Step of this Run as a condition and a `{step:, path:}`
// reference name it: the id namespace it was authored in, and its authored id.
// The namespace is half the key because an id is unique inside one Procedure
// and says nothing across two (§3, sequence.go).
type stepKey struct {
	namespace int
	id        string
}

// acted is what the Steps of one Run acted on, by Step: the fields each call
// concluded, in Expansion order.
//
// It holds a Step's Records rather than a Step's versions for the reason above:
// a call that returned what the head already held minted no version and acted
// on its Record all the same.
type acted map[stepKey][]store.Mapping

// condition is a Step's `when:` as authored: the earlier Step it roots at, and
// the one predicate it carries.
//
// It is a predicate with a `step:` beside it, which is exactly what §12's
// second root is — the same eleven operators, the same one declared `field:`
// name, and a root that is a named earlier Step's Record rather than the Record
// being filtered.
type condition struct {
	// Step is the `step:` — an `id:` the same Procedure declares earlier,
	// which `check` has already held to that (§4).
	Step string
	// predicate is the `field:` and the one operator beside it — and the
	// operator's own line, which is what a Refusal cites. A condition has no
	// second line to carry: §8 points a caret at the operator at both Record
	// roots, and a `when:` line beside it would be a second answer to one
	// question.
	predicate
}

// readCondition reads a Step's `when:` as authored, and answers false where the
// Step carries none.
//
// It judges nothing: a condition carrying no `step:`, no `field:` or no
// operator is one `check` has already refused, and this reader answers what it
// found (ADR-0064).
func readCondition(when *yaml.Node) (condition, bool) {
	if when == nil || when.Kind != yaml.MappingNode {
		return condition{}, false
	}

	read := condition{predicate: readPredicate(when, 0)}
	for i := 0; i+1 < len(when.Content); i += 2 {
		key, value := when.Content[i], when.Content[i+1]
		if key.Kind == yaml.ScalarNode && value.Kind == yaml.ScalarNode && key.Value == "step" {
			read.Step = value.Value
		}
	}
	return read, true
}

// holds answers whether the condition holds of what the Step it names acted on,
// and the mismatch where its operator was handed a value it cannot compare.
//
// **A Step that acted on no Record leaves it not holding**, which is the whole
// of §6's skip rule: the named Step was skipped by either Disposition, was
// never reached, or resolved an Expansion of nothing. There is no fall-through
// to the Store and no Refusal.
//
// Where the Step acted on several Records the condition holds of all of them.
// That is the reading the operator set already has — a predicate is a filter
// and a filter over a population is an AND, the same rule §12 fixes for a
// predicate list — and it is why the answer is *the Records earlier Steps of
// this Run acted on* rather than one of them. Every Record is evaluated whether
// or not an earlier one settled the answer, so whether a Run Refuses does not
// depend on which Record a response happened to project first (ADR-0035).
//
// A `{step:, path:}` reference to the same Step answers *resolves to nothing*
// there instead (arguments.go), and the difference is the position rather than
// an inconsistency: a filter takes a population and a value takes one thing.
// That is why `series-reference` declines the reference and not the predicate —
// the two roots are the same words in different positions, and only one of them
// needs one Record to mean anything (§3, §4, ADR-0126).
//
// **How many of them satisfied it travels back beside the verdict**, for the
// halt a `require:` writes. It names no member and no observed value, which is
// what ADR-0035 keeps every predicate report from doing; what it says is that
// the root was a population at all, which is the fact an author rooting at an
// expanding Step by accident has no other way to be told (requirement.go).
func (c condition) holds(records []store.Mapping, instant time.Time) (held bool, satisfied int, mismatch string) {
	if len(records) == 0 {
		return false, 0, ""
	}

	held, found := true, ""
	for _, fields := range records {
		matched, mismatch := c.predicate.holds(fields, instant)
		if mismatch != "" && found == "" {
			found = mismatch
		}
		if matched {
			satisfied++
		}
		held = held && matched
	}
	if found != "" {
		return false, satisfied, found
	}
	return held, satisfied, ""
}
