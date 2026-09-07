package run

import (
	"errors"
	"fmt"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **`skip-if-recorded`'s test resolves its identity before the call, and there
// are exactly two forms that do** (§4, §6, ADR-0056, issue #152).
//
// It is a unit test for identityBeforeTheCall's reason one file over: the
// answer is a pure function of an Operation's `identity:`, its Capability and
// the member in front of it, where the corpus would need one Manifest per form
// to say the same three things.

// skipIdentityCase is one Operation's `identity:` met by one member: what the
// Manifest declared, whether the Operation is a `shell` one, the argv and the
// pre-call identity the Expansion left on the member, and the series the skip
// test would read the head of.
type skipIdentityCase struct {
	identity string
	isShell  bool
	argv     []string
	resolved string
	holds    string
}

func TestSkipIdentity_ResolvesTheSeriesTheCallWouldWriteUnder(t *testing.T) {
	for named, c := range map[string]skipIdentityCase{
		// The ordinary form: a template hole, filled from this member's
		// inputs at the Expansion and read straight off the member.
		"a template hole the Expansion filled": {
			identity: "{name}", resolved: "preview-42.example.com", holds: "preview-42.example.com",
		},
		// The second form, and the one the built-in
		// `mutate_skip_if_recorded` is: `$.command` is in the response
		// object because it is a fact about the call, so it is known
		// before one goes out — and it is spelled by the Capability
		// rather than joined here.
		"$.command on a shell Operation": {
			identity: "$.command", isShell: true, argv: []string{"mkdir", "-p", "/srv/app"},
			holds: `["mkdir","-p","/srv/app"]`,
		},
		// The same path on an `http` Operation names a value that
		// exists only once the call has gone out. `check` refuses it
		// (`manifest-inconsistent`), and nothing resolves here.
		"$.command on an http Operation": {
			identity: "$.command", holds: "",
		},
		// A `shell` Operation whose argv did not resolve has no command
		// to be named by. It is unreachable — an empty `command:` is
		// `command-malformed` at load — and it answers nothing rather
		// than naming the empty argv's own spelling.
		"a shell Operation with no argv": {
			identity: "$.command", isShell: true, holds: "",
		},
	} {
		t.Run(named, func(t *testing.T) {
			bound := binding{operation: artefact.OperationInfo{
				Kind: "mutate", Repeatability: "skip-if-recorded",
				Identity: c.identity, IsShell: c.isShell,
			}}
			held := skipIdentity(bound, member{Argv: c.argv, Identity: c.resolved})
			if held != c.holds {
				t.Errorf("resolved %q, want %q", held, c.holds)
			}
		})
	}
}

// TestSkipsIfRecorded_ReadsTheDeclaredValue holds the one fact the walk asks of
// an Operation's Repeatability: whether this is the value whose test runs at
// all. Every other value and the silence alike answer no, and the silence is
// the one that matters — run-once is what a `mutate` declaring nothing is, and
// it is a different mechanism reading a different thing (§12).
func TestSkipsIfRecorded_ReadsTheDeclaredValue(t *testing.T) {
	for value, want := range map[string]bool{
		"skip-if-recorded": true,
		"repeatable":       false,
		"":                 false,
	} {
		bound := binding{operation: artefact.OperationInfo{Kind: "mutate", Repeatability: value}}
		if held := bound.skipsIfRecorded(); held != want {
			t.Errorf("repeatability: %q reads %v, want %v", value, held, want)
		}
	}
}

// TestProducesSecret_ReadsTheDeclaration holds the second half of *this Step
// concluded about a Record and wrote no value for it*: whether the Operation
// declares `secret:` output at all (§9, ADR-0150, issue #273).
//
// **What it reads is `HasSecret` and never the set beside it**, and the third
// case is what fences that: the two members answer the same question and the §6
// gate reads the first, so a second reading over `SecretFields` would be two
// readings of one declaration and the day they disagree is already written down
// (gates.go, artefact.OperationInfo).
//
// The empty declaration is the case worth writing down beside it. `secret: []`
// names no field, so nothing is suppressed and nothing reaches a sink, and a
// Step under it that skipped every member is short of nothing.
func TestProducesSecret_ReadsTheDeclaration(t *testing.T) {
	for name, c := range map[string]struct {
		operation artefact.OperationInfo
		want      bool
	}{
		"an Operation declaring a secret field": {
			operation: artefact.OperationInfo{Kind: "mutate", HasSecret: true, SecretFields: map[string]bool{"password": true}},
			want:      true,
		},
		"an Operation declaring none": {
			operation: artefact.OperationInfo{Kind: "mutate"},
			want:      false,
		},
		"an Operation whose secret: names no field": {
			operation: artefact.OperationInfo{Kind: "mutate", SecretFields: map[string]bool{}},
			want:      false,
		},
		"the declaration read off HasSecret alone": {
			// The set is not the operand. A reading over
			// `SecretFields` would answer no here and the gate
			// would answer yes, which is the Run declining for a
			// Step this reports nothing about.
			operation: artefact.OperationInfo{Kind: "mutate", HasSecret: true},
			want:      true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if held := (binding{operation: c.operation}).producesSecret(); held != c.want {
				t.Errorf("producesSecret reads %v, want %v", held, c.want)
			}
		})
	}
}

// TestHaltedBeforeTheCall_TellsAPreCallFaultFromEveryOtherWayAMemberEnds is the
// reason skipFault is a type. A member's turn answers one error channel and two
// unrelated things travel down it: what a call answered, which is this Step's
// Disposition to describe, and a fault raised in front of the call, which is
// the Run's to end on — no request having been made for a Disposition to
// describe (§6, §7).
func TestHaltedBeforeTheCall_TellsAPreCallFaultFromEveryOtherWayAMemberEnds(t *testing.T) {
	raised := errors.New("the branch is not there")

	if held := haltedBeforeTheCall(beforeTheCall(raised)); !errors.Is(held, raised) {
		t.Errorf("a pre-call fault reads back as %v, want %v", held, raised)
	}
	for named, fault := range map[string]error{
		"nothing at all":       nil,
		"a status that halted": answeredOtherwise(store.HTTPAnswer{Host: "api.example.com"}, "500"),
		"a projection":         unreadable("$.id", "the identity path did not resolve"),
		"a plain halt":         fmt.Errorf("step publish: %w", raised),
	} {
		if held := haltedBeforeTheCall(fault); held != nil {
			t.Errorf("%s reads back as a pre-call fault: %v", named, held)
		}
	}
}
