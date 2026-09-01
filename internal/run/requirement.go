package run

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// A Requirement's `require:`: the verdict a Procedure halts on, and the one
// way a Run stops on a `read`'s answer without a Step that claims authority
// over anything (§6, §12, issue #236, ADR-0116).
//
// **It is the same predicate a `when:` is**, at the same root and read by the
// same reader — a named earlier Step's Record, `step:` beside `field:`,
// evaluated against what the Steps of *this* Run acted on and never against
// the Store (condition.go). What differs is what the answer is for. A `when:`
// that does not hold skips the Step it is written on; a `require:` that does
// not hold halts the Run, and every Step after it — in this Procedure and in
// whatever invoked it — is *never reached*.
//
// **That is why a shared check can gate its callers.** A halt inside a nested
// Procedure is a halt of the whole (§6), so a Procedure whose last entry is a
// Requirement stops its caller's later Steps by stopping the Run. Before this
// existed the only thing that halted was an effectful Operation, so the one
// artefact whose point was that it writes nothing had to claim `mutate` on the
// Target it was protecting in order to be able to fail (ADR-0111).
//
// **A halt and not a Refusal.** A Requirement roots at an earlier Step by
// construction, so its verdict is always reached after that Step's call went
// out — which is ADR-0072's criterion exactly. It is `failed` at `1`, it
// carries no `error_code`, and it names what it compared. What `77` would have
// promised beside it is false: a verbatim retry does not refuse identically,
// the world being what moved.
//
// **The one thing that does Refuse is a predicate that cannot decide.** An
// operator handed a value it cannot compare is `predicate-type-mismatch`
// wherever it stands (ADR-0035), and the sibling key already fixes that it is
// a Refusal rather than a halt: a Record that quietly failed to compare is
// indistinguishable from one that compared and did not match, and which of the
// two keys the predicate was written under is not a ground for `hyper` to hold
// two answers about one fault.

// required evaluates every Requirement standing in front of the Step at index
// `before`, in written order, and answers what the first one that did not pass
// did to the Run: an error where it did not hold, and a Refusal where its
// operator could not decide.
//
// A Requirement standing after the last Step the Run holds is evaluated with
// `before` at the length of the sequence, which is where the caller reads it.
func (r run) required(before int) ([]Refusal, error) {
	for _, standing := range r.requirements {
		if standing.Before != before {
			continue
		}
		declined, err := r.verdict(standing)
		if err != nil || len(declined) > 0 {
			return declined, err
		}
	}
	return nil, nil
}

// verdict is one Requirement's own answer.
//
// **A Step that acted on no Record leaves it not holding**, which is
// condition.go's rule arriving at the outcome this key has for it. The named
// Step was skipped, was never reached, or resolved an Expansion of nothing;
// there is no fall-through to the Store and nothing for the operator to be
// true of, and a requirement nothing satisfied is not satisfied. The message
// says which of the two happened, because the edits differ: an unmet
// requirement points at the world and an unanswerable one points at the Step
// above it.
func (r run) verdict(required requirement) ([]Refusal, error) {
	condition, carried := readCondition(required.Require)
	if !carried {
		// Unreachable from a Run: a `require:` that is not a mapping is
		// `check`'s schema-mismatch, and `check` re-runs in full at Run
		// start. It is answered rather than assumed, on this package's
		// own rule for every name a Step writes (§6, ADR-0064).
		return nil, fmt.Errorf("requirement %s carries no legible require: — hyper check reports it", namedRequirement(required))
	}

	acted := r.acted[stepKey{required.Namespace, condition.Step}]
	if len(acted) == 0 {
		return nil, fmt.Errorf("requirement %s did not hold: step %s acted on no Record in this Run, so there was nothing for %s to be read of, and no Step after this line runs",
			namedRequirement(required), condition.Step, condition.Field)
	}

	held, satisfied, mismatch := condition.holds(acted, r.started)
	if mismatch != "" {
		return []Refusal{{RefusalMember: store.RefusalMember{
			ErrorCode: CodePredicateTypeMismatch,
			File:      required.Declared.Path,
			Line:      condition.Line,
			Field:     fmt.Sprintf("steps[%d].require.%s", required.Index, condition.Operator),
			Message:   fmt.Sprintf("%s, %s", rootedAt(condition.Step, len(acted)), mismatch),
			StepID:    required.ID,
		}}}, nil
	}
	if !held {
		return nil, fmt.Errorf("requirement %s did not hold: %s, and no Step after this line runs",
			namedRequirement(required), unmet(condition, satisfied, len(acted)))
	}
	return nil, nil
}

// unmet is the middle of the halt: what did not satisfy what, and — where the
// Step the Requirement roots at acted on more than one Record — how many of
// them did.
//
// **The count is the whole of what the expanding root adds**, and it is here
// because nothing else tells an author their root was a population. A `require:`
// roots at a Step and a Step of `series` cardinality, or one ranging over an
// Expansion, is as legal a root as any other and holds of every Record it acted
// on (§3, condition.go). An author who meant *this one* and wrote the list read
// has authored a stricter predicate than the one they meant, and the sentence
// they get on the halt is the first place that is visible — *of the Record*,
// singular, said the opposite (ADR-0126).
//
// It names no member and no observed value: what an author edits is the
// operand, and naming whichever member came first is the thing ADR-0035 keeps
// every predicate report from doing.
func unmet(against condition, satisfied, records int) string {
	if records == 1 {
		return fmt.Sprintf("%s of the Record step %s acted on does not satisfy %s",
			against.Field, against.Step, asWritten(against.predicate))
	}
	return fmt.Sprintf("step %s acted on %d Records and %s satisfies %s on %d of them — a require: holds of every one",
		against.Step, records, against.Field, asWritten(against.predicate), satisfied)
}

// rootedAt is how a predicate report names the root it read: the Records the
// Step it names acted on, counted where there is more than one.
//
// It is shared by the two Record roots because the roots are one root — a
// `when:` and a `require:` are the same predicate at the same place (§3,
// condition.go) — and a reader told *the Record* where there were nine has been
// told something false about which values the operator was handed (ADR-0126).
func rootedAt(step string, records int) string {
	if records == 1 {
		return fmt.Sprintf("on the Record step %s acted on", step)
	}
	return fmt.Sprintf("on the %d Records step %s acted on", records, step)
}

// asWritten is the test a Requirement carries, spelled the way its author wrote
// it: the operator, and the operand beside it where the operator takes one.
//
// It is the **artefact's** half of the comparison and not the Store's, which is
// a choice rather than an economy. A `require:` holds of every Record the named
// Step acted on (condition.go), so there is no one observed value to name — a
// sentence carrying one would be naming whichever member came first, which is
// the thing ADR-0035 keeps every other predicate report from doing. What an
// author edits is the operand, and that is what the halt hands them (§7, §12).
func asWritten(test predicate) string {
	switch {
	case test.Operand == nil:
		return test.Operator
	case test.Operand.Kind == yaml.ScalarNode:
		return test.Operator + ": " + test.Operand.Value
	case test.Operand.Kind == yaml.SequenceNode:
		members := make([]string, 0, len(test.Operand.Content))
		for _, member := range test.Operand.Content {
			members = append(members, member.Value)
		}
		return test.Operator + ": [" + strings.Join(members, ", ") + "]"
	default:
		return test.Operator
	}
}

// namedRequirement is how one Requirement is named where one name is wanted:
// its path where it was reached through an invocation, and its authored id
// where it sits at the top level. It is `named`'s answer for the entry shape
// that is not a Step, and it is spelled separately because a Requirement holds
// no Step to be named by (sequence.go).
func namedRequirement(required requirement) string {
	switch {
	case required.Path != "":
		return required.Path
	case required.ID != "":
		return required.ID
	default:
		return fmt.Sprintf("at line %d", required.Line)
	}
}
