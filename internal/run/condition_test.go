package run

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// **A condition reads the Records an earlier Step of this Run acted on, and
// nothing else** (§6, §12, issue #141).
//
// What the corpus drives is a Run: the Step that skips, the skip that
// propagates, and the Store head a condition does not fall through to. What is
// held here is the decision itself, which is a pure function of a predicate and
// a set of Records — including the two readings that would need a Procedure of
// four Steps to reach through a page.

// conditionOf reads a `when:` the way a Step carries one, from the text an
// author would write.
func conditionOf(t *testing.T, text string) condition {
	t.Helper()

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(text), &node); err != nil {
		t.Fatalf("reading %q: %v", text, err)
	}
	read, carried := readCondition(node.Content[0])
	if !carried {
		t.Fatalf("%q carries no condition", text)
	}
	return read
}

// TestCondition_ReadsTheStepItRootsAtAndTheOperatorBesideIt holds the shape:
// §12's second root is a predicate with a `step:` beside it.
func TestCondition_ReadsTheStepItRootsAtAndTheOperatorBesideIt(t *testing.T) {
	read := conditionOf(t, "{step: probe, field: status, equals: \"200\"}")

	if read.Step != "probe" {
		t.Errorf("step: %q, want probe", read.Step)
	}
	if read.Field != "status" || read.Operator != "equals" {
		t.Errorf("field: %q %s:, want status equals:", read.Field, read.Operator)
	}
}

// TestCondition_ReadsNoneWhereTheStepCarriesNone is the ordinary Step: no
// `when:`, nothing to decide, and the caller runs it.
func TestCondition_ReadsNoneWhereTheStepCarriesNone(t *testing.T) {
	if _, carried := readCondition(nil); carried {
		t.Error("a Step carrying no when: read a condition")
	}
}

// TestCondition_HoldsOfTheRecordTheNamedStepActedOn is the ordinary case in
// both directions.
func TestCondition_HoldsOfTheRecordTheNamedStepActedOn(t *testing.T) {
	when := conditionOf(t, "{step: probe, field: status, equals: \"200\"}")

	for name, c := range map[string]struct {
		acted []store.Mapping
		holds bool
	}{
		"the Record matches":         {[]store.Mapping{{"status": store.String("200")}}, true},
		"the Record does not match":  {[]store.Mapping{{"status": store.String("503")}}, false},
		"the Record lacks the field": {[]store.Mapping{{"host": store.String("a")}}, false},
	} {
		t.Run(name, func(t *testing.T) {
			held, mismatch := when.holds(c.acted, time.Time{})
			if mismatch != "" {
				t.Fatalf("mismatch %q, want a comparison", mismatch)
			}
			if held != c.holds {
				t.Errorf("holds = %v, want %v", held, c.holds)
			}
		})
	}
}

// TestCondition_DoesNotHoldWhereTheNamedStepActedOnNothing is §6's skip rule,
// and it is the whole of what makes a skip propagate.
//
// The named Step was skipped by either Disposition, was never reached, or
// resolved an Expansion of nothing. **It does not fall through to the Store** —
// that would be the condition reading another Run's Record — and **it does not
// Refuse**: an earlier optional Step being skipped is an ordinary occurrence,
// and Refusing on it would leave the Procedure un-runnable with no exit but an
// edit to a reviewed artefact (ADR-0001).
func TestCondition_DoesNotHoldWhereTheNamedStepActedOnNothing(t *testing.T) {
	for name, acted := range map[string][]store.Mapping{
		"the Step wrote no Record at all":  nil,
		"the Step concluded about nothing": {},
	} {
		t.Run(name, func(t *testing.T) {
			held, mismatch := conditionOf(t, "{step: probe, field: status, absent: true}").holds(acted, time.Time{})
			if held {
				t.Error("the condition held over a Step that acted on no Record")
			}
			if mismatch != "" {
				t.Errorf("mismatch %q — a skip is an ordinary occurrence and never a Refusal", mismatch)
			}
		})
	}
}

// TestCondition_RefusesAValueItCannotCompare is ADR-0035 at the second root. A
// predicate that cannot decide Refuses; it never treats the value as not
// matching, which would be indistinguishable on every surface from a Record
// that compared and did not match.
func TestCondition_RefusesAValueItCannotCompare(t *testing.T) {
	when := conditionOf(t, "{step: probe, field: status, greater_than: 10}")

	held, mismatch := when.holds([]store.Mapping{{"status": store.String("up")}}, time.Time{})
	if held {
		t.Error("the condition held over a value it cannot compare")
	}
	if mismatch == "" {
		t.Fatal("a value the operator cannot compare compared silently")
	}
}

// TestCondition_HoldsOfEveryRecordTheStepActedOn is the reading for a Step that
// expanded: a predicate is a filter, and a filter over a population is an AND —
// the rule §12 already fixes for a predicate list, one root over.
//
// Every Record is evaluated whether or not an earlier one settled the answer,
// so a value that cannot compare Refuses wherever it sits in the set rather
// than depending on which Record a response happened to project first
// (ADR-0035).
func TestCondition_HoldsOfEveryRecordTheStepActedOn(t *testing.T) {
	when := conditionOf(t, "{step: probe, field: status, equals: \"200\"}")

	all := []store.Mapping{{"status": store.String("200")}, {"status": store.String("200")}}
	if held, _ := when.holds(all, time.Time{}); !held {
		t.Error("the condition did not hold where every Record matched")
	}

	one := []store.Mapping{{"status": store.String("200")}, {"status": store.String("503")}}
	if held, _ := when.holds(one, time.Time{}); held {
		t.Error("the condition held where one of two Records did not match")
	}

	uncomparable := []store.Mapping{{"status": store.String("503")}, {"status": store.Bool(true)}}
	if _, mismatch := conditionOf(t, "{step: probe, field: status, starts_with: \"5\"}").holds(uncomparable, time.Time{}); mismatch == "" {
		t.Error("a value the operator cannot compare passed unreported behind a Record that already excluded the set")
	}
}

// TestCondition_ReadsTheInstantTheRunSuppliesRatherThanAClock holds ADR-0034 at
// this root: one instant covers every Step, every nested Procedure and all
// three roots.
func TestCondition_ReadsTheInstantTheRunSuppliesRatherThanAClock(t *testing.T) {
	when := conditionOf(t, "{step: probe, field: seen_at, older_than: 1h}")
	acted := []store.Mapping{{"seen_at": store.String("2026-04-02T09:00:00Z")}}

	later, _ := time.Parse(time.RFC3339, "2026-04-02T11:00:00Z")
	if held, _ := when.holds(acted, later); !held {
		t.Error("the condition did not hold two hours after the value it was compared against")
	}
	sooner, _ := time.Parse(time.RFC3339, "2026-04-02T09:30:00Z")
	if held, _ := when.holds(acted, sooner); held {
		t.Error("the condition held half an hour after the value it was compared against")
	}
}
