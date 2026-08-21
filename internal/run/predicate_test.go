// This file tests issue #139: §12's eleven predicate operators evaluated
// against a **stored** value — the half `check` cannot reach, there being no
// Store in its hand (§4, §5, §6, §12, ADR-0035).
//
// It is a table rather than a corpus case, and the exception is argued in the
// milestone's own testing note: `internal/run` is driven through `cli.Main`
// because its interface is a Run, and what stands here is not an arrangement
// but a **closed table §12 writes down** — eleven operators against the operand
// types they take and the ones they refuse. Driving it through the corpus would
// be one fixture repository, one seeded branch and one golden per cell.
//
// What the corpus does hold is the mechanism: that a mismatch Refuses with
// nothing touched, that the list is AND, and that it does not short-circuit.
package run

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// theInstant is the Run's start, which the two temporal operators resolve
// against — one instant for every Step and every nested Procedure, fixed at the
// Run's start and used verbatim (ADR-0034).
var theInstant = time.Date(2026, 4, 2, 9, 41, 14, 221_000_000, time.UTC)

// parsePredicate reads one predicate the way a Step's selector carries it: one
// entry of a list, a mapping of `field:` and one operator key.
func parsePredicate(t *testing.T, text string) predicate {
	t.Helper()

	var node yaml.Node
	if err := yaml.Unmarshal([]byte("["+text+"]"), &node); err != nil {
		t.Fatalf("the fixture does not parse: %v", err)
	}
	read := readPredicates(node.Content[0])
	if len(read) != 1 {
		t.Fatalf("the fixture reads as %d predicates, want 1", len(read))
	}
	return read[0]
}

// TestPredicate_HoldsAgainstTheOperandTypesEachOperatorTakes walks §12's table
// cell by cell: the value the version carries, the predicate written against
// it, and whether it holds.
func TestPredicate_HoldsAgainstTheOperandTypesEachOperatorTakes(t *testing.T) {
	number := func(literal string) store.Value {
		read, err := store.ParseNumber(literal)
		if err != nil {
			t.Fatalf("%q is not a number: %v", literal, err)
		}
		return read
	}

	for name, c := range map[string]struct {
		value store.Value
		text  string
		holds bool
	}{
		"equals over a string":                 {store.String("preview-42"), "{field: name, equals: preview-42}", true},
		"equals over a string that differs":    {store.String("preview-41"), "{field: name, equals: preview-42}", false},
		"equals is byte-exact over UTF-8":      {store.String("Preview-42"), "{field: name, equals: preview-42}", false},
		"equals over an integer and a number":  {number("1.0"), "{field: ttl, equals: 1}", true},
		"not_equals over a string":             {store.String("preview-41"), "{field: name, not_equals: preview-42}", true},
		"in over a member":                     {store.String("b"), "{field: name, in: [a, b]}", true},
		"in over no member":                    {store.String("c"), "{field: name, in: [a, b]}", false},
		"exists over a field the version has":  {store.String("preview-42"), "{field: name, exists: true}", true},
		"starts_with over the prefix":          {store.String("preview-42"), "{field: name, starts_with: preview-}", true},
		"starts_with over another string":      {store.String("prod-42"), "{field: name, starts_with: preview-}", false},
		"ends_with over the suffix":            {store.String("preview-42"), "{field: name, ends_with: \"-42\"}", true},
		"greater_than over a number":           {number("34"), "{field: days_left, greater_than: 30}", true},
		"greater_than over an equal number":    {number("30"), "{field: days_left, greater_than: 30}", false},
		"less_than over a number":              {number("29"), "{field: days_left, less_than: 30}", true},
		"greater_than compares two durations":  {store.String("10m"), "{field: ttl, greater_than: 300s}", true},
		"less_than compares two durations":     {store.String("10m"), "{field: ttl, less_than: 300s}", false},
		"older_than a duration before the run": {store.String("2026-03-01T00:00:00Z"), "{field: created_at, older_than: 14d}", true},
		"older_than a duration after it":       {store.String("2026-04-01T00:00:00Z"), "{field: created_at, older_than: 14d}", false},
		"older_than an absolute timestamp":     {store.String("2026-03-01T00:00:00Z"), "{field: created_at, older_than: \"2026-03-02T00:00:00Z\"}", true},
		"newer_than a duration":                {store.String("2026-04-01T00:00:00Z"), "{field: created_at, newer_than: 14d}", true},
		"a timestamp is read at any offset":    {store.String("2026-03-01T02:00:00+02:00"), "{field: created_at, equals: \"2026-03-01T02:00:00+02:00\"}", true},
		// The operand is characters read as the type the **value** is, so
		// a numeral against a string is an ordinary string comparison
		// and decides: what cannot be compared is a value of a type the
		// operator does not take, never an operand that looks unlike it.
		"equals a numeral against a string": {store.String("thirty-four"), "{field: name, equals: 34}", false},
		"newer_than reads an offset as UTC": {store.String("2026-04-02T10:41:00+02:00"), "{field: created_at, newer_than: \"2026-04-02T09:00:00Z\"}", false},
	} {
		t.Run(name, func(t *testing.T) {
			held, mismatch := parsePredicate(t, c.text).holds(store.Mapping{
				"name": c.value, "ttl": c.value, "days_left": c.value, "created_at": c.value,
			}, theInstant)
			if mismatch != "" {
				t.Fatalf("%s: %s", c.text, mismatch)
			}
			if held != c.holds {
				t.Errorf("%s over %s holds %v, want %v", c.text, string(store.Encode(c.value)), held, c.holds)
			}
		})
	}
}

// TestPredicate_AbsenceIsWhatExistsAndAbsentState. A field's presence is a fact
// the two operators state and never a nullable type (§7, §12), so an absent
// field decides every other operator rather than refusing one: what `hyper`
// cannot compare is a value of the wrong type and never a value that is not
// there.
func TestPredicate_AbsenceIsWhatExistsAndAbsentState(t *testing.T) {
	for text, want := range map[string]bool{
		"{field: name, absent: true}":          true,
		"{field: name, exists: true}":          false,
		"{field: name, equals: preview-42}":    false,
		"{field: name, not_equals: preview-4}": false,
		"{field: name, starts_with: preview-}": false,
	} {
		held, mismatch := parsePredicate(t, text).holds(store.Mapping{"other": store.String("x")}, theInstant)
		if mismatch != "" {
			t.Errorf("%s over a version carrying no such field: %s", text, mismatch)
			continue
		}
		if held != want {
			t.Errorf("%s over a version carrying no such field holds %v, want %v", text, held, want)
		}
	}
}

// TestPredicate_RefusesTheValueItCannotCompare is ADR-0035 itself: it never
// treats the value as not matching, because a Record that quietly failed to
// compare is indistinguishable from one that compared and did not match.
func TestPredicate_RefusesTheValueItCannotCompare(t *testing.T) {
	number, err := store.ParseNumber("34")
	if err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct {
		value store.Value
		text  string
	}{
		"older_than against a number":        {number, "{field: f, older_than: 14d}"},
		"newer_than against a string":        {store.String("soon"), "{field: f, newer_than: 14d}"},
		"greater_than against a string":      {store.String("many"), "{field: f, greater_than: 10}"},
		"greater_than against a duration":    {store.String("10m"), "{field: f, greater_than: 10}"},
		"less_than against a number operand": {number, "{field: f, less_than: 10m}"},
		"starts_with against an object":      {store.Mapping{"a": store.String("b")}, "{field: f, starts_with: preview-}"},
		"starts_with against a number":       {number, "{field: f, starts_with: \"3\"}"},
		"ends_with against a list":           {store.Array{store.String("a")}, "{field: f, ends_with: a}"},
		"equals against an object":           {store.Mapping{}, "{field: f, equals: preview-42}"},
		"equals a string against a number":   {number, "{field: f, equals: thirty-four}"},
		"equals a boolean against a number":  {number, "{field: f, equals: true}"},
		"in against a value of another type": {number, "{field: f, in: [a, b]}"},
	} {
		t.Run(name, func(t *testing.T) {
			held, mismatch := parsePredicate(t, c.text).holds(store.Mapping{"f": c.value}, theInstant)
			if mismatch == "" {
				t.Fatalf("%s over %s answered %v and refused nothing", c.text, string(store.Encode(c.value)), held)
			}
		})
	}
}

// TestPredicate_ComparesANumberBeyondAFloat64sExactRange. A Record identity on
// plenty of upstreams is an integer past that range (§7), and a comparison that
// went through a float would answer that two of them are equal.
func TestPredicate_ComparesANumberBeyondAFloat64sExactRange(t *testing.T) {
	value, err := store.ParseNumber("9007199254740993")
	if err != nil {
		t.Fatal(err)
	}

	held, mismatch := parsePredicate(t, "{field: f, equals: 9007199254740992}").holds(store.Mapping{"f": value}, theInstant)
	if mismatch != "" {
		t.Fatalf("two integers did not compare: %s", mismatch)
	}
	if held {
		t.Error("9007199254740993 equals 9007199254740992; the comparison went through a float64")
	}
}
