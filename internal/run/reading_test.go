package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **What a projection that does not resolve does to a Run**, and what an
// identity that resolves and collides does (§6, §7, issue #144).
//
// The corpus one package up holds the consequence of all of it — the halt, the
// Disposition, the `projection_failed_path`, the `n of m` and the three
// comparands each in a case of its own — and what is here is the two things a
// page cannot show.
//
// **That no surface renders the response a projection failed against**
// (ADR-0017). A golden asserts the line that was written; only a case holding a
// response full of things the message could have leaked can assert the line
// that was not.
//
// **That whatever held an identity first keeps it, decided by an order and
// never by a race.** A `read` Expansion runs concurrently and the order its
// calls complete in is defined nowhere (drain.go), so a corpus case proves the
// answer and not the rule: two members that always answer in the same order
// would report the same winner whichever order decided it.

// TestReadingFault_NamesNothingOfTheResponse drives the two projections that
// halt against a response every member of which is a distinctive string, and
// holds that not one of them reaches the message.
func TestReadingFault_NamesNothingOfTheResponse(t *testing.T) {
	// Every one of these is in the response and none of them is in a path,
	// so a message carrying any of them is a message carrying the answer.
	const secretive = `{"records":[{"ident":"leaked-identity","state":"leaked-state"}],"note":"leaked-note"}`
	response := responseOf(t, secretive)

	for _, c := range []struct {
		name     string
		identity string
		record   string
		want     string
	}{
		{
			name:     "the identity path",
			identity: "$.body.id",
			record:   "kind: read\nrecord:\n  identity: $.body.id\n  fields:\n    note: $.body.note\n",
			want:     "$.body.id",
		},
		{
			name:     "the collection path",
			identity: "$.ident",
			record:   "kind: read\nrecord:\n  identity: $.ident\n  over: $.body.items\n  fields:\n    state: $.state\n",
			want:     "$.body.items",
		},
		{
			name:     "one member's identity path",
			identity: "$.id",
			record:   "kind: read\nrecord:\n  identity: $.id\n  over: $.body.records\n  fields:\n    state: $.state\n",
			want:     "$.id",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			held, fault := concludedFrom(t, c.identity, c.record, response, 0)
			if fault == nil {
				t.Fatalf("concluded %#v and no fault; want a halt", held)
			}
			if path := failedPath(fault); path != c.want {
				t.Errorf("projection_failed_path = %q, want %q", path, c.want)
			}
			if !wroteWhatProjected(fault) {
				t.Error("the fault is not one the drain writes what projected for")
			}
			for _, leaked := range []string{"leaked-identity", "leaked-state", "leaked-note"} {
				if strings.Contains(fault.Error(), leaked) {
					t.Errorf("the fault names %q, which is the response and not the path: %s", leaked, fault)
				}
			}
		})
	}
}

// TestConcluded_WhatProjectedIsWritten is the half-projected response: the
// collection resolves, the members before the fault project, and the fault
// arrives beside them rather than instead of them (§6).
func TestConcluded_WhatProjectedIsWritten(t *testing.T) {
	response := responseOf(t, `{"records":[{"id":"r1"},{"id":"r2"},{"state":"nameless"}]}`)

	held, fault := concludedFrom(t, "$.id", "kind: read\nrecord:\n  identity: $.id\n  over: $.body.records\n  fields:\n    state: $.state\n", response, 0)
	if fault == nil {
		t.Fatal("no fault; the third member has no identity to be written under")
	}
	if len(held) != 2 || held[0].name != "r1" || held[1].name != "r2" {
		t.Fatalf("concluded %#v; want the two members that projected", held)
	}
	// The position is what a collision names the Record by, and it counts
	// the collection rather than the answers: the third member is the third
	// whether or not the first two were written.
	if held[0].at != 1 || held[1].at != 2 {
		t.Errorf("positions = %d, %d; want 1, 2", held[0].at, held[1].at)
	}
	if !strings.Contains(fault.Error(), "record 3 of $.body.records") {
		t.Errorf("the fault does not name which Record failed: %s", fault)
	}
}

// TestConcluded_PagesCarryOnCounting holds the positions across a pagination
// Pattern's pages: they are one collection arriving in instalments, so the
// first Record of the second page is the third and not the first.
func TestConcluded_PagesCarryOnCounting(t *testing.T) {
	held, fault := concludedFrom(t, "$.id", "kind: read\nrecord:\n  identity: $.id\n  over: $.body.records\n",
		responseOf(t, `{"records":[{"id":"r3"},{"id":"r4"}]}`), 2)
	if fault != nil {
		t.Fatalf("fault %v; want none", fault)
	}
	if len(held) != 2 || held[0].at != 3 || held[1].at != 4 {
		t.Fatalf("positions = %#v; want the third and fourth Records of the walk", held)
	}
}

// TestWroteWhatProjected is the line between the two ways a Step halts: an
// answer `hyper` could not read back is one it holds, and a deadline is one it
// never got.
func TestWroteWhatProjected(t *testing.T) {
	if wroteWhatProjected(context.DeadlineExceeded) {
		t.Error("a deadline writes what projected; there is nothing to write")
	}
	if !wroteWhatProjected(unreadable("$.id", "no")) {
		t.Error("a projection that did not resolve writes nothing")
	}
	// The drain wraps the fault with the Step it happened at, so both
	// readers have to survive one.
	wrapped := fmt.Errorf("step items: %w", collided("two names"))
	if !wroteWhatProjected(wrapped) || failedPath(wrapped) != "" {
		t.Errorf("a wrapped collision reads as %v / %q", wroteWhatProjected(wrapped), failedPath(wrapped))
	}
	if failedPath(fmt.Errorf("step items: %w", unreadable("$.id", "no"))) != "$.id" {
		t.Error("a wrapped projection fault carries no path")
	}
	if failedPath(errors.New("something else")) != "" {
		t.Error("a fault that is not a reading carries a path")
	}
}

// TestIdentityHolders_WhateverHeldTheIdentityFirstKeepsIt is the rule over all
// three comparands: the first to hold a folded identity keeps it, the later one
// is the collision, and the report names both spellings verbatim.
func TestIdentityHolders_WhateverHeldTheIdentityFirstKeepsIt(t *testing.T) {
	standing := identity("Crate")
	holders := identityHolders{
		first:    map[store.Identity]projectedIdentityBy{},
		standing: map[store.Identity]store.Identity{identity("crate"): standing},
	}

	// The Store comparand: nothing here supplies an order, the standing
	// series having been written by an earlier Run.
	stored := holders.take(identity("crate"), "what came back")
	if stored == nil {
		t.Fatal("an identity the Store already holds under another case did not collide")
	}
	for _, verbatim := range []string{"crate", "Crate"} {
		if !strings.Contains(stored.Error(), verbatim) {
			t.Errorf("the report does not name %q verbatim: %s", verbatim, stored)
		}
	}
	if !strings.Contains(stored.Error(), "local/inventory-read") {
		t.Errorf("the report does not name the series it collided with: %s", stored)
	}

	// The colliding identity is **not** taken: it was not written, so a
	// third member spelling it a third way collides with whatever did hold
	// it and not with the member that failed.
	if _, taken := holders.first[store.Folded(identity("crate"))]; taken {
		t.Error("the colliding identity was recorded as a holder")
	}

	// The sibling comparand, in the order the drain reads the answers back.
	if first := holders.take(identity("Tin"), "one.hyper.dev"); first != nil {
		t.Fatalf("the first member to resolve an identity collided: %v", first)
	}
	sibling := holders.take(identity("tin"), "two.hyper.dev")
	if sibling == nil {
		t.Fatal("a second member resolving one identity under the fold did not collide")
	}
	if !strings.Contains(sibling.Error(), "two.hyper.dev resolved the identity tin") ||
		!strings.Contains(sibling.Error(), "one.hyper.dev already resolved Tin") {
		t.Errorf("the report does not name both members and both spellings: %s", sibling)
	}
	// A collision carries no failed path: the identity resolved, and what
	// is wrong with it is what it resolved to (§7).
	if failedPath(sibling) != "" {
		t.Errorf("a collision carries projection_failed_path %q", failedPath(sibling))
	}
	// An identity byte-equal to nothing standing and nothing seen is the
	// ordinary answer.
	if fresh := holders.take(identity("barrel"), "three.hyper.dev"); fresh != nil {
		t.Errorf("an identity nothing holds collided: %v", fresh)
	}
}

// TestProjectedBy is how a collision names what projected an identity, over the
// four shapes a Step has: a member or none, and one Record out of a response or
// many.
func TestProjectedBy(t *testing.T) {
	for _, c := range []struct {
		member string
		at     int
		want   string
	}{
		{member: "shard-2.hyper.dev", at: 0, want: "shard-2.hyper.dev"},
		{member: "shard-2.hyper.dev", at: 10, want: "record 10 of shard-2.hyper.dev"},
		{member: "", at: 0, want: "what came back"},
		{member: "", at: 3, want: "record 3 of what came back"},
	} {
		if got := projectedBy(c.member, c.at); got != c.want {
			t.Errorf("projectedBy(%q, %d) = %q, want %q", c.member, c.at, got, c.want)
		}
	}
}

// identity is one Record identity of the Target and Definition these cases
// share, which is the pair a collision is decided within.
func identity(name string) store.Identity {
	return store.Identity{Target: "local", Definition: "inventory-read", Name: name}
}

// concludedFrom projects one response through one Operation's record: block and
// answers what concluded beside the halt, which is what step.go's page callback
// does with the two.
//
// The `identity:` arrives beside the block rather than being read back out of
// it: at a Step it comes off internal/artefact's reading of the Manifest, and a
// second reader here would be this file answering a question the engine asks
// somewhere else.
func concludedFrom(t *testing.T, identityPath, record string, response capability.Object, already int) ([]conclusion, error) {
	t.Helper()

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(record), &root); err != nil {
		t.Fatal(err)
	}
	bound := binding{operation: artefact.OperationInfo{Identity: identityPath}}

	return run{}.concluded(bound, sequenced{Step: artefact.Step{ID: "items"}}, projection.Read(root.Content[0]), member{}, response, already)
}

// responseOf is a response object carrying one parsed body, which is the only
// member the paths in these cases reach.
func responseOf(t *testing.T, encoded string) capability.Object {
	t.Helper()

	var body any
	if err := json.Unmarshal([]byte(encoded), &body); err != nil {
		t.Fatal(err)
	}
	return capability.Object{
		{Name: capability.MemberHost, Value: "series.hyper.dev"},
		{Name: capability.MemberStatus, Value: 200},
		{Name: capability.MemberBody, Value: body},
	}
}
