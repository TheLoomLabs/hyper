package artefact

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// A Requirement: the third shape a `steps:` entry takes, carrying `id:` and
// `require:` and nothing else, and the read-only halt it is (§3, §4, §6,
// issue #236).
//
// The Steps it stands beside are the same `shell` `read` the rest of this
// file binds, so what these cases exercise is the entry and never the
// binding.

const requirementProcedure = `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]

  - id: sound
    require: {step: probe, field: exit_code, equals: 0}
`

func TestCheckProcedure_RequirementIsClean(t *testing.T) {
	got := CheckProcedure("procedures/deploy.yaml", parse(t, requirementProcedure), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// A Requirement claims no Kind and binds no Target, so it contributes nothing
// to the envelope its Procedure declares — including on a Procedure whose
// `targets:` grants only `read`, which is the whole point of the shape (§5).
func TestCheckProcedure_RequirementContributesNothingToTheEnvelope(t *testing.T) {
	readOnly := TargetIndex{"local": {Kinds: map[string]bool{"read": true}}}
	got := CheckProcedure("procedures/deploy.yaml", parse(t, requirementProcedure), shellProviders(), uptimeDefinitions(), readOnly, ProcedureIndex{})
	mustNoCode(t, got, CodeEnvelopeExceeded)
}

func TestCheckProcedure_RequirementCarriesNoBinding(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]

  - id: sound
    definition: uptime
    operation: read
    target: local
    require: {step: probe, field: exit_code, equals: 0}
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[1]" {
		t.Errorf("Field = %q, want steps[1]", p.Field)
	}
	if !strings.Contains(p.Message, "require:") {
		t.Errorf("Message = %q, want it to name require:", p.Message)
	}
}

func TestCheckProcedure_RequirementIsNotAnInvocation(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: verify
    procedure: verify-archive
    require: {step: probe, field: exit_code, equals: 0}
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"verify-archive": true})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

func TestCheckProcedure_RequirementAdmitsNoOtherKey(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]

  - id: sound
    require: {step: probe, field: exit_code, equals: 0}
    bound: 1
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "steps[1].bound" {
		t.Errorf("Field = %q, want steps[1].bound", p.Field)
	}
}

func TestCheckProcedure_RequirementRootsAtAnEarlierStep(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: sound
    require: {step: probe, field: exit_code, equals: 0}

  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[0].require.step" {
		t.Errorf("Field = %q, want steps[0].require.step", p.Field)
	}
}

func TestCheckProcedure_RequirementFieldResolvesAgainstTheEarlierStepsProvider(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]

  - id: sound
    require: {step: probe, field: bogus, equals: 0}
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[1].require.field" {
		t.Errorf("Field = %q, want steps[1].require.field", p.Field)
	}
}

// A Requirement acts on no Record, so nothing roots at one: not a later
// Requirement, and not a `when:`. The message says which of the three shapes
// the id names rather than leaving an author to read *no id: this Procedure
// declares earlier* against a line that plainly declares it (§3, ADR-0099).
func TestCheckProcedure_NothingRootsAtARequirement(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]

  - id: sound
    require: {step: probe, field: exit_code, equals: 0}

  - id: again
    definition: uptime
    operation: read
    target: local
    when: {step: sound, field: exit_code, equals: 0}
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[2].when.step" {
		t.Errorf("Field = %q, want steps[2].when.step", p.Field)
	}
	if !strings.Contains(p.Message, "requirement") {
		t.Errorf("Message = %q, want it to say the id names a requirement", p.Message)
	}
}

// The wall issue #236 was found at: a caller naming the invocation it wants
// to gate on. It stays `reference-unresolvable` — an invocation writes no
// Record — and the message now names the way out (§3, ADR-0002, ADR-0111).
func TestCheckProcedure_ConditionCannotRootAtAnInvocation(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: verify
    procedure: verify-archive

  - id: swap
    definition: uptime
    operation: read
    target: local
    when: {step: verify, field: exit_code, equals: 0}
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"verify-archive": true})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[1].when.step" {
		t.Errorf("Field = %q, want steps[1].when.step", p.Field)
	}
	if !strings.Contains(p.Message, "require:") {
		t.Errorf("Message = %q, want it to name require:", p.Message)
	}
}

// The second spelling the same session tried: a Step of the invoked
// Procedure, reached through the invocation's own id.
func TestCheckProcedure_ConditionNamingANestedStepSaysWhy(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: verify
    procedure: verify-archive

  - id: swap
    definition: uptime
    operation: read
    target: local
    when: {step: verify.archive-intact, field: exit_code, equals: 0}
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"verify-archive": true})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if !strings.Contains(p.Message, "require:") {
		t.Errorf("Message = %q, want it to name require:", p.Message)
	}
}

// An `args:` reference meets the same boundary and is told the same thing
// about it — minus the way out, a value never crossing the boundary at all.
func TestCheckProcedure_ReferenceCannotNameAnInvocation(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: verify
    procedure: verify-archive

  - id: swap
    definition: uptime
    operation: read
    target: local
    args:
      command: [echo, {step: verify, path: $.exit_code}]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"verify-archive": true})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if !strings.Contains(p.Message, "invocation") {
		t.Errorf("Message = %q, want it to say the id names a nested invocation", p.Message)
	}
}

// A dotted name in an `args:` reference meets the same boundary and is told
// the same first half — and not the second. A `require:` states a verdict; it
// carries no value across, and pointing an author at a key that cannot solve
// their problem is worse than saying only what is true (ADR-0116).
func TestCheckProcedure_AReferenceIsNotOfferedARequirement(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: verify
    procedure: verify-archive

  - id: swap
    definition: uptime
    operation: read
    target: local
    args:
      command: [echo, {step: verify.archive-intact, path: $.exit_code}]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"verify-archive": true})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if !strings.Contains(p.Message, "referenceable from its caller") {
		t.Errorf("Message = %q, want it to name the boundary", p.Message)
	}
	if strings.Contains(p.Message, "require:") {
		t.Errorf("Message = %q, want it not to offer require: — a reference carries no value across", p.Message)
	}
}

func TestReadProcedureSteps_ReadsARequirement(t *testing.T) {
	steps := ReadProcedureSteps(parse(t, requirementProcedure))
	if len(steps) != 2 {
		t.Fatalf("ReadProcedureSteps() read %d entries, want 2", len(steps))
	}
	requirement := steps[1]
	if !requirement.IsRequirement() {
		t.Errorf("IsRequirement() = false, want true")
	}
	if requirement.IsInvocation() {
		t.Errorf("IsInvocation() = true, want false")
	}
	if requirement.ID != "sound" {
		t.Errorf("ID = %q, want sound", requirement.ID)
	}
	if requirement.Require == nil {
		t.Errorf("Require = nil, want the predicate node")
	}
}
