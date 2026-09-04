// Package presence holds one closed set: what the environment did with the
// variable a Target declaration's credential slot names (§12, ADR-0145).
//
// It is a package of its own, holding three constants and one function, because
// **two packages ask this question and there must be one answer**. internal/run's
// credential pass decides a Run under these three before Step 1, and internal/cli's
// `targets` reports the same three on both its wires — and neither may import the
// other, `hyper targets` being fenced to the imports that prove it reaches no
// network, no Store and no invocation (§9, targets_test.go). Two spellings is
// where the gate and the column come to disagree, and what they would disagree
// about is whether a Run is about to Refuse, which is the invariant issue #112
// bought the column for.
//
// It imports nothing, which is what lets the fenced command hold it: there is no
// reach here to be granted, only a set to be read the same way twice.
package presence

// Presence is one variable's answer, and the set is closed at three (§12).
//
// **`Empty` is no characters and never a judgement about a credential's
// contents**: no length beyond zero, no shape, no plausibility, no scan. Whether
// a credential works is the endpoint's business and needs the value to decide;
// whether one was supplied at all is `hyper`'s and does not (ADR-0007).
type Presence string

const (
	// Absent is a variable the environment does not hold.
	Absent Presence = "absent"
	// Empty is a variable the environment holds and sets to the empty
	// string — what an upstream produces rather than what an operator
	// forgets: a reader that returned nothing, a CI secret never set on the
	// fork, a vault read against the wrong path.
	Empty Presence = "empty"
	// Set is a variable the environment fills, and the only one of the three
	// a Run proceeds past.
	Set Presence = "set"
)

// Of is the reading, taken from `os.LookupEnv`'s own pair.
//
// It takes the pair rather than a variable name so that nothing here reaches the
// process environment: the caller has already asked, once, and where that
// question is put is §6's own order for a Run and the moment of the call for
// `targets`.
func Of(value string, present bool) Presence {
	switch {
	case !present:
		return Absent
	case value == "":
		return Empty
	default:
		return Set
	}
}
