package artefact

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// The Procedure a Run of one `read` Step performs, and the two members that
// share `steps:` with it: a Step proper, a nested invocation, and the keys a
// milestone that has not built one declines on.
const stepsProcedure = `kind: procedure
procedure: watch
targets: [local, cloudflare-prod]
steps:
  - id: status
    definition: uptime-check
    operation: check_http
    target: local
    args:
      host: status.hyper.dev

  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values: [a.example.com]
    when:
      - step: status
        field: status
        equals: 200
    args:
      record_id: {item: $}
    bound: 1

  - id: nested
    procedure: inner
`

func TestReadProcedureSteps_ReadsTheBindingArgsAndTheThreeOptionalKeys(t *testing.T) {
	steps := ReadProcedureSteps(parse(t, stepsProcedure))

	if len(steps) != 3 {
		t.Fatalf("read %d Steps, want the three the sequence holds", len(steps))
	}

	first := steps[0]
	if first.ID != "status" || first.Definition != "uptime-check" || first.Operation != "check_http" || first.Target != "local" {
		t.Errorf("the first Step reads %+v, want the binding it was authored with", first)
	}
	if host := first.Args["host"]; host == nil || host.Value != "status.hyper.dev" {
		t.Errorf("the first Step's args read %v, want host: status.hyper.dev", first.Args)
	}
	if first.Over != nil || first.When != nil || first.Bound != nil {
		t.Errorf("the first Step carries a selector, a condition or a Bound; it was authored with none")
	}
	if first.IsInvocation() {
		t.Errorf("the first Step reads as a nested invocation; it binds a Definition")
	}
	if first.Line == 0 {
		t.Errorf("the first Step names no line; a Refusal citing it has no caret to draw")
	}

	second := steps[1]
	if second.Over == nil || second.When == nil || second.Bound == nil {
		t.Errorf("the second Step reads over: %v, when: %v, bound: %v; it was authored with all three", second.Over, second.When, second.Bound)
	}

	third := steps[2]
	if !third.IsInvocation() || third.Invocation != "inner" {
		t.Errorf("the third member reads %+v, want the nested invocation it was authored as", third)
	}
	if third.Definition != "" || third.Operation != "" || third.Target != "" {
		t.Errorf("the nested invocation carries a binding; an invocation binds nothing (§3)")
	}
}

// A reader judges nothing and drops nothing, which is the rule every reader in
// this package follows: what is wrong with an artefact is check's to report,
// and a Step the engine cannot perform must still arrive at the engine — a
// Step silently absent from the sequence is a Run that skipped one without
// saying so (ADR-0064).
func TestReadProcedureSteps_AFaultyStepIsReadRatherThanDropped(t *testing.T) {
	faulty := `kind: procedure
procedure: watch
steps:
  - id: nameless
  - definition: uptime-check
    operation: check_http
    target: local
`
	steps := ReadProcedureSteps(parse(t, faulty))
	if len(steps) != 2 {
		t.Fatalf("read %d Steps, want both members of the sequence", len(steps))
	}
	if steps[0].Definition != "" {
		t.Errorf("the Step with no binding reads %q; a reader invents nothing", steps[0].Definition)
	}
	if steps[1].ID != "" {
		t.Errorf("the Step with no id reads %q; a reader invents nothing", steps[1].ID)
	}
}

// A root that is not a Procedure, and one whose steps: is not a sequence, are
// both no Steps at all rather than a fault.
func TestReadProcedureSteps_NoLegibleStepsIsNoSteps(t *testing.T) {
	for name, doc := range map[string]string{
		"no steps: at all":       "kind: procedure\nprocedure: watch\n",
		"steps: is not a list":   "kind: procedure\nprocedure: watch\nsteps: everything\n",
		"not a Procedure at all": "kind: target-declaration\ntarget: local\n",
	} {
		t.Run(name, func(t *testing.T) {
			if steps := ReadProcedureSteps(parse(t, doc)); steps != nil {
				t.Errorf("read %d Steps, want none", len(steps))
			}
		})
	}

	if steps := ReadProcedureSteps((*yaml.Node)(nil)); steps != nil {
		t.Errorf("a nil root read %d Steps, want none", len(steps))
	}
}
