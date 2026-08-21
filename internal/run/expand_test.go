package run

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **Four checks decide at a Step's Expansion and they have an order, and it is
// the causal one** (§5, §6, §12, issue #139).
//
// A predicate resolves the set; the set has the count a Bound is read against;
// the members' arguments fill the inputs an identity is projected from; and the
// identities are projected off the members the set holds, against each other
// and against the Store. Where two are available at once the earlier in that
// sequence is the one named, and the sibling collision is named before the
// Store's — being reproducible from the artefact alone, and therefore pointing
// at an edit with no Store in hand.
//
// This is the exception the milestone's testing note names: `internal/run` is
// driven through `cli.Main` because its interface is a Run, and the order these
// checks decide in is a pure function of a resolved set. What the corpus holds
// beside it is each check declining on its own — and the two that can be
// available at once, in [the-sibling-collision-is-named-first].

// found is one Refusal in the shape only its code is read out of, since what
// this table is about is which check answered and not what it said.
func found(code string) []Refusal {
	return []Refusal{{RefusalMember: store.RefusalMember{ErrorCode: code}}}
}

// TestChecks_DeclineInTheCausalOrder walks the order cell by cell: which checks
// found something, and which one the Refusal names.
func TestChecks_DeclineInTheCausalOrder(t *testing.T) {
	predicate := found(CodePredicateTypeMismatch)
	bound := found("bound-exceeded")
	arguments := found(schema.CodeMismatch)
	collision := found(CodeRecordIdentityCollision)

	for name, c := range map[string]struct {
		checks checks
		names  string
	}{
		"nothing declined": {
			checks{}, "",
		},
		"a predicate that cannot compare, alone": {
			checks{Predicate: predicate}, CodePredicateTypeMismatch,
		},
		"a predicate before the count the Bound is read against": {
			checks{Predicate: predicate, Bound: bound}, CodePredicateTypeMismatch,
		},
		"a predicate before the arguments the identity is projected from": {
			checks{Predicate: predicate, Arguments: arguments}, CodePredicateTypeMismatch,
		},
		"a predicate before either identity comparand": {
			checks{Predicate: predicate, Sibling: collision, Stored: collision}, CodePredicateTypeMismatch,
		},
		"the count before the identities projected off the set": {
			checks{Bound: bound, Sibling: collision, Stored: collision}, "bound-exceeded",
		},
		"an argument before the identity it fills": {
			checks{Arguments: arguments, Sibling: collision, Stored: collision}, schema.CodeMismatch,
		},
		"the sibling collision before the Store's": {
			checks{Sibling: collision, Stored: collision}, CodeRecordIdentityCollision,
		},
		"the Store's comparand alone": {
			checks{Stored: collision}, CodeRecordIdentityCollision,
		},
	} {
		t.Run(name, func(t *testing.T) {
			declined := c.checks.declined()
			if c.names == "" {
				if len(declined) != 0 {
					t.Fatalf("declined %v, want nothing", declined)
				}
				return
			}
			// **A Refusal holds exactly one member** at this
			// moment, which is what having an order is for (§7).
			if len(declined) != 1 {
				t.Fatalf("declined %d members, want exactly 1: %v", len(declined), declined)
			}
			if declined[0].ErrorCode != c.names {
				t.Errorf("declined %s, want %s", declined[0].ErrorCode, c.names)
			}
		})
	}
}

// TestChecks_TheSiblingIsNamedBeforeTheStoresOverTheSameSet holds the one pair
// the corpus can drive both halves of at once, over the values a resolution
// would build: two members that are one identity, one of which is also one the
// Store already holds. The sibling is named, and the Store's comparand — which
// is just as true — is not.
func TestChecks_TheSiblingIsNamedBeforeTheStoresOverTheSameSet(t *testing.T) {
	sibling := Refusal{RefusalMember: store.RefusalMember{
		ErrorCode: CodeRecordIdentityCollision,
		Message:   "alpha resolves to status.hyper.dev and beta resolves to Status.hyper.dev",
	}}
	stored := Refusal{RefusalMember: store.RefusalMember{
		ErrorCode: CodeRecordIdentityCollision,
		Message:   "alpha resolves to status.hyper.dev, and the Store already holds STATUS.HYPER.DEV",
	}}

	declined := checks{Sibling: []Refusal{sibling}, Stored: []Refusal{stored}}.declined()
	if len(declined) != 1 || declined[0].Message != sibling.Message {
		t.Errorf("declined %v, want the sibling alone — it is reproducible from the artefact with no Store in hand", declined)
	}
}
