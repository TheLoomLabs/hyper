package run

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// §12's eleven operators, evaluated against a value the Store holds (§5, §6,
// §12, issue #139).
//
// This is the run-time half of a check milestone 1 landed the authored half of.
// `check` reads a predicate's **operand** — the characters an author wrote, and
// the operand types §12's table admits — and it can read no further: a
// projected field has no declared type (§3), so what a `fields:` entry actually
// holds is discovered when it is read and never before.
//
// **A value this cannot compare Refuses.** It never treats it as not matching,
// which is ADR-0035 entire: a Record that quietly failed to compare is
// indistinguishable on every surface from one that compared and did not match,
// and an API that changed a field's type would silently change what a selector
// reaches.
//
// **Nothing coerces.** A value is compared as the type it is. A timestamp
// arrives in a Store file as a string, so `starts_with: "2026-"` against one is
// legal and does what it looks like — `hyper` is not pretending to know more
// about that value than that it is a string.

// predicate is one entry of an `assets:` or `observations:` list as it was
// authored: the field it roots at, the one operator it carries, and that
// operator's operand held as a node.
//
// The operand is a node rather than a read value because §12 gives each
// operator its own operand types and one of them is a list. Reading it is the
// operator's, at the moment the value it is compared against is in hand.
type predicate struct {
	// Field is the `field:` — one declared field name and never a path,
	// there being nothing at a Record root for a path to traverse (§12).
	Field string
	// Operator is the member of §12's closed eleven, and Operand what it
	// was written against.
	Operator string
	Operand  *yaml.Node
	// Line is where the entry begins and Index its position in the list,
	// which is what a Refusal citing this predicate points at (§8).
	Line, Index int
}

// operators is §12's closed set, in the table's own order. It is a slice rather
// than a set because reading a predicate needs the one key that is an operator
// and a mapping's own key order is the author's.
var operators = []string{
	"equals", "not_equals", "in", "exists", "absent",
	"starts_with", "ends_with", "greater_than", "less_than",
	"older_than", "newer_than",
}

// readPredicates reads a predicate list as authored, in written order.
//
// It judges nothing and drops nothing: a list carrying an entry with two
// operators or none is a list `check` has already refused, and this reader's
// answer for one is the first operator it carries or the empty operator, which
// no evaluation reaches (ADR-0064).
func readPredicates(list *yaml.Node) []predicate {
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil
	}

	read := make([]predicate, 0, len(list.Content))
	for index, entry := range list.Content {
		if entry.Kind != yaml.MappingNode {
			continue
		}
		held := predicate{Line: entry.Line, Index: index}
		for i := 0; i+1 < len(entry.Content); i += 2 {
			key, value := entry.Content[i], entry.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			if key.Value == "field" {
				held.Field = value.Value
				continue
			}
			if held.Operator == "" && slices.Contains(operators, key.Value) {
				held.Operator, held.Operand = key.Value, value
				held.Line = key.Line
			}
		}
		read = append(read, held)
	}
	return read
}

// holds answers whether this predicate holds of a version's fields, and the
// mismatch where the operator was handed a value it cannot compare — the empty
// string where it compared, whichever way the comparison went.
//
// instant is the Run's start, read off `run.json` and used verbatim: one
// instant covers every Step, every nested Procedure and all three roots, so
// nothing a Pattern or a slow API does during a Run moves what a later Step
// reaches (ADR-0034).
//
// **A field the version does not carry is not a mismatch.** A field's presence
// is a fact `exists` and `absent` state rather than a nullable type (§7, §12),
// so absence decides every other operator rather than refusing one: what cannot
// be compared is a value of the wrong type and never a value that is not there.
func (p predicate) holds(fields store.Mapping, instant time.Time) (bool, string) {
	value, carried := fields[p.Field]
	switch p.Operator {
	case "exists":
		return carried, ""
	case "absent":
		return !carried, ""
	}
	if !carried {
		return false, ""
	}

	operand := ""
	if p.Operand != nil && p.Operand.Kind == yaml.ScalarNode {
		operand = p.Operand.Value
	}

	switch p.Operator {
	case "equals":
		return p.compares(value, operand)
	case "not_equals":
		equal, mismatch := p.compares(value, operand)
		return !equal && mismatch == "", mismatch
	case "in":
		return p.within(value)
	case "starts_with":
		return p.affixed(value, operand, strings.HasPrefix)
	case "ends_with":
		return p.affixed(value, operand, strings.HasSuffix)
	case "greater_than":
		return p.orders(value, operand, above)
	case "less_than":
		return p.orders(value, operand, below)
	case "older_than":
		return p.times(value, operand, instant, below)
	case "newer_than":
		return p.times(value, operand, instant, above)
	default:
		// An operator outside the closed set, which `check` has already
		// refused under unknown-key. It holds of nothing rather than
		// refusing here, a second opinion on an unreviewed artefact
		// being what ADR-0064 keeps this package out of.
		return false, ""
	}
}

// compares is `equals`, and the comparison `in` and `not_equals` are written
// over: the operand's characters read as **the type the value is**, which is
// what makes this a statement about types rather than a coercion table.
//
// `integer` and `number` are one domain here — two scalar types where an input
// schema constrains what a caller supplies, and one comparison where a value
// has already come back — so `equals: 1` holds against `1.0` (§12).
func (p predicate) compares(value store.Value, operand string) (bool, string) {
	switch held := value.(type) {
	case store.String:
		// Byte-exact over UTF-8, case-sensitive, with no normalisation,
		// on the ground §7 folds no case in a Record identity: the rule
		// is `hyper`'s rather than the locale's, and it is the same *the
		// bytes moved* test the canonical encoding already runs (§12).
		return string(held) == operand, ""
	case store.Number:
		wanted, reads := rational(operand)
		if !reads {
			return false, p.cannot(value, "a number")
		}
		found, exact := rational(held.Text())
		if !exact {
			return false, p.cannot(value, "a number")
		}
		return found.Cmp(wanted) == 0, ""
	case store.Bool:
		if operand != "true" && operand != "false" {
			return false, p.cannot(value, "a boolean")
		}
		return bool(held) == (operand == "true"), ""
	case store.Timestamp:
		// Unreachable from a projection, which writes a timestamp as
		// the string the wire carried (§7, internal/run/value.go). It is
		// answered rather than left to the default so that a Store value
		// this package can compare is never reported as one it cannot.
		return store.InstantText(time.Time(held)) == operand, ""
	default:
		return false, p.cannot(value, "a scalar")
	}
}

// within is `in`: the value equals a member of a list of two or more literals,
// all one type (§12).
//
// It reads every member rather than stopping at the first that matches, so a
// list whose members are not the value's type Refuses whether or not an earlier
// member happened to equal it — the same reason the list of predicates around
// it does not short-circuit (ADR-0035).
func (p predicate) within(value store.Value) (bool, string) {
	if p.Operand == nil || p.Operand.Kind != yaml.SequenceNode {
		return false, ""
	}

	held := false
	for _, member := range p.Operand.Content {
		if member.Kind != yaml.ScalarNode {
			continue
		}
		equal, mismatch := p.compares(value, member.Value)
		if mismatch != "" {
			return false, mismatch
		}
		held = held || equal
	}
	return held, ""
}

// The side of the comparand a value falls on for one of §12's paired operators
// to hold: `above` for the operator naming the greater or the later value,
// `below` for the one naming the lesser or the earlier.
//
// It is decided at the switch that has just read the operator's name and handed
// down, so that no comparison below reads that name a second time. The four
// ordered operators are strict and the pairs are how negation is written (§12),
// which is what makes a side and an exact comparison the whole of them.
const (
	above = 1
	below = -1
)

// affixed is `starts_with` and `ends_with`, which take a non-empty string and a
// value that is one — the bounded form of prefix and suffix matching, in place
// of the unbounded one §12 declined (ADR-0022).
func (p predicate) affixed(value store.Value, operand string, has func(string, string) bool) (bool, string) {
	text, isText := value.(store.String)
	if !isText {
		return false, p.cannot(value, "a string")
	}
	return has(string(text), operand), ""
}

// orders is `greater_than` and `less_than`: the two operators that compare two
// lengths, over the two domains §12 gives them.
//
// Which domain it is is the **operand's** to say, the value having no declared
// type to read one off: `300s` is a duration and `10` is a number, told apart by
// the grammar rather than by what the Record happens to hold. A value that is
// not of that domain is the mismatch.
//
// Durations compare by normalised length, so `10m` is greater than `300s` — the
// no-compounding rule §12 states is about a value rendering back byte-identical
// to what was authored, which is a fact about writing rather than about
// ordering.
func (p predicate) orders(value store.Value, operand string, side int) (bool, string) {
	if wanted, isDuration := schema.DurationSeconds(operand); isDuration {
		text, isText := value.(store.String)
		if !isText {
			return false, p.cannot(value, "a duration")
		}
		found, reads := schema.DurationSeconds(string(text))
		if !reads {
			return false, p.cannot(value, "a duration")
		}
		return big.NewRat(int64(found), 1).Cmp(big.NewRat(int64(wanted), 1)) == side, ""
	}

	wanted, reads := rational(operand)
	if !reads {
		// An operand that is neither a duration nor a number, which §4
		// has already refused where it is on the page.
		return false, ""
	}
	held, isNumber := value.(store.Number)
	if !isNumber {
		return false, p.cannot(value, "a number")
	}
	found, exact := rational(held.Text())
	if !exact {
		return false, p.cannot(value, "a number")
	}
	return found.Cmp(wanted) == side, ""
}

// times is `older_than` and `newer_than`, which are the whole of time in this
// operator set (§12).
//
// The operand names an instant: a `duration` names the one that far before the
// Run's start, and a `timestamp` names itself. The value is read as **any RFC
// 3339 timestamp, an offset included, and normalised to UTC** — the `Z` §3
// mandates binds a value an author writes, where two artefacts could otherwise
// disagree about what a local time meant, and there is no author involved when
// an API hands one back.
//
// An epoch integer is a number rather than a timestamp and neither operator
// reads one as one, which is §13's stated cost and this function's mismatch.
func (p predicate) times(value store.Value, operand string, instant time.Time, side int) (bool, string) {
	threshold, named := p.instant(operand, instant)
	if !named {
		// An operand that is neither a duration nor a timestamp, which
		// §4 has already refused where it is on the page.
		return false, ""
	}

	found, reads := instantOf(value)
	if !reads {
		return false, p.cannot(value, "a timestamp")
	}
	return found.Compare(threshold) == side, ""
}

// instant is the instant the operand names, and false where it names none.
func (p predicate) instant(operand string, runStarted time.Time) (time.Time, bool) {
	if seconds, isDuration := schema.DurationSeconds(operand); isDuration {
		return runStarted.Add(-time.Duration(seconds) * time.Second), true
	}
	return readTimestamp(operand)
}

// instantOf is a stored value read as the instant it names, and false where it
// names none.
//
// A projection writes a timestamp as the string the wire carried (§7,
// internal/run/value.go), which is the case every Run reaches; a Store
// Timestamp is answered beside it for the reason compares answers one — a value
// this package can compare must never be reported as one it cannot.
func instantOf(value store.Value) (time.Time, bool) {
	switch held := value.(type) {
	case store.String:
		return readTimestamp(string(held))
	case store.Timestamp:
		return time.Time(held).UTC(), true
	default:
		return time.Time{}, false
	}
}

// readTimestamp reads a value as RFC 3339 and normalises it to UTC. Both forms
// are read — with and without a fractional second — which is the reading §3
// already performs over an authored one, minus the mandatory `Z`.
func readTimestamp(text string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if read, err := time.Parse(layout, text); err == nil {
			return read.UTC(), true
		}
	}
	return time.Time{}, false
}

// rational reads a number's text exactly. It is a big.Rat rather than a float64
// because an integer past a float64's exact range is a Record identity on
// plenty of upstreams (§7), and a comparison through a float would answer that
// two of them are equal.
//
// A `/` is refused rather than read: big.Rat's own grammar admits a fraction
// and neither JSON nor §12's `number` does, so a value nobody could have
// written would otherwise compare.
func rational(text string) (*big.Rat, bool) {
	if strings.ContainsRune(text, '/') {
		return nil, false
	}
	return new(big.Rat).SetString(text)
}

// cannot is the mismatch this predicate reports: what the version holds, and
// what the operator takes.
//
// It names the field, the value **and** the operator, because a Refusal carries
// exactly one member (§7) and this sentence is the whole of what a reader is
// handed about a Store they have no checkout of.
func (p predicate) cannot(value store.Value, takes string) string {
	return fmt.Sprintf("field: %s holds %s, and %s: takes %s", p.Field, describe(value), p.Operator, takes)
}

// describe is a stored value as a sentence names it. It renders a scalar with
// its type beside it and a container with its type alone: what a reader needs
// is which type arrived, and an object's whole content is the Record's to show.
//
// It is read by the Refusal a reference earns as well as by the one a predicate
// does (expand.go). Both are a value the Store held meeting a type an artefact
// declared, and one vocabulary for *what arrived* is what keeps two Refusals
// about one Record from describing it two ways.
func describe(value store.Value) string {
	switch held := value.(type) {
	case store.String:
		return fmt.Sprintf("the string %q", string(held))
	case store.Number:
		return "the number " + held.Text()
	case store.Bool:
		return fmt.Sprintf("the boolean %v", bool(held))
	case store.Timestamp:
		return "the timestamp " + store.InstantText(time.Time(held))
	case store.Mapping:
		return "an object"
	case store.Array:
		return "a list"
	default:
		return "a value no comparison reads"
	}
}
