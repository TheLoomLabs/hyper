package artefact

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// secretProvider is a synthetic Manifest carrying one Operation that
// declares secret: output — none of the built-ins do, so the two Cadence
// rules' secret-output half needs a fixture of its own (issue #96).
const secretProvider = `kind: provider
provider: vault
schema-version: 1
class: vault
capabilities: [http]
operations:
  issue_token:
    kind: mutate
    repeatability: repeatable
    deadline: 1h
    http:
      method: POST
      host: "{from-target}"
      path: /issue
    record:
      identity: $.id
      fields:
        token: $.token
    secret: [token]
`

func secretProviders(t *testing.T) ProviderIndex {
	t.Helper()
	return BuildProviderIndex([]*yaml.Node{parse(t, secretProvider)})
}

func secretDefinitions() DefinitionIndex {
	return DefinitionIndex{"issue-token": DefinitionInfo{
		ProviderName: "vault",
		Kinds:        map[string]bool{"mutate": true},
		Targets:      map[string]TargetInfo{"vault-prod": fullyGrantedTarget()},
	}}
}

// procedureRoot is a small parse-and-wrap helper — BuildProcedureGraph
// takes file/root pairs rather than bare roots, needing every procedures/
// file's own name to cite a fault against.
func procedureRoot(t *testing.T, file, doc string) ProcedureRoot {
	t.Helper()
	return ProcedureRoot{File: file, Root: parse(t, doc)}
}

// --- envelope-exceeded: the transitive half, invocation reaching outside its caller ---

const outerOnlyLocal = `kind: procedure
procedure: outer
targets: [local]
steps:
  - id: call-inner
    procedure: inner
`

const innerTouchesOther = `kind: procedure
procedure: inner
targets: [local, other]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: other
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_InvokedProcedureExceedingCallerEnvelopeIsEnvelopeExceeded(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/outer.yaml", outerOnlyLocal),
		procedureRoot(t, "procedures/inner.yaml", innerTouchesOther),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeEnvelopeExceeded)
	if p.File != "procedures/outer.yaml" {
		t.Errorf("File = %q, want procedures/outer.yaml — the caller, whose author can widen it", p.File)
	}
	if p.Field != "steps[0].procedure" {
		t.Errorf("Field = %q, want steps[0].procedure", p.Field)
	}
}

const innerSubsetOfOuter = `kind: procedure
procedure: inner
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_InvokedProcedureInsideCallerEnvelopeDrawsNoEnvelopeExceeded(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/outer.yaml", outerOnlyLocal),
		procedureRoot(t, "procedures/inner.yaml", innerSubsetOfOuter),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	mustNoCode(t, got, CodeEnvelopeExceeded)
}

// TestCheckProcedureGraph_ViolationThreeLevelsDeepIsFound proves the walk
// reaches every Procedure reachable to any depth: p1 invokes p2 invokes p3,
// and p3's own Step touches a Target neither p1 nor p2 declares — found at
// the p2-to-p3 edge, two hops below where the check started.
const graphP1 = `kind: procedure
procedure: p1
targets: [local, other]
steps:
  - id: call-p2
    procedure: p2
`

const graphP2 = `kind: procedure
procedure: p2
targets: [local]
steps:
  - id: call-p3
    procedure: p3
`

const graphP3 = `kind: procedure
procedure: p3
targets: [local, other]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: other
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_ViolationThreeLevelsDeepIsFound(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/p1.yaml", graphP1),
		procedureRoot(t, "procedures/p2.yaml", graphP2),
		procedureRoot(t, "procedures/p3.yaml", graphP3),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeEnvelopeExceeded)
	if p.File != "procedures/p2.yaml" {
		t.Errorf("File = %q, want procedures/p2.yaml — the edge where the deeper reach first escapes a declared envelope", p.File)
	}
	for _, prob := range got {
		if prob.File == "procedures/p1.yaml" {
			t.Errorf("got a row on procedures/p1.yaml too — the pairwise edge check at p2-to-p3 already covers p1 transitively")
		}
	}
}

// --- cadence-run-once ---

const cadenceOverRunOnce = `kind: procedure
procedure: nightly
targets: [local]
cadence: "0 3 * * *"
steps:
  - id: change
    definition: uptime
    operation: mutate_once
    target: local
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_CadenceOverOwnRunOnceStepIsCadenceRunOnce(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/nightly.yaml", cadenceOverRunOnce),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeCadenceRunOnce)
	if p.Field != "cadence" {
		t.Errorf("Field = %q, want cadence", p.Field)
	}
	if p.Line != 4 {
		t.Errorf("Line = %d, want 4 — the cadence: line, not the Step's own", p.Line)
	}
}

// TestCheckProcedureGraph_CadenceOverNestedRunOnceStepIsFoundInInvokedProcedure
// proves both Cadence rules ride the same transitive walk envelope-exceeded
// makes: the run-once Step sits in the nested Procedure, not the one
// carrying the clock, and the row still cites the clock's own file and line.
const cadenceCaller = `kind: procedure
procedure: nightly
targets: [local]
cadence: "0 3 * * *"
steps:
  - id: call-inner
    procedure: inner-once
`

const innerRunOnce = `kind: procedure
procedure: inner-once
targets: [local]
steps:
  - id: change
    definition: uptime
    operation: destroy_once
    target: local
    over:
      values: [/tmp/a]
    args:
      command: [rm, -rf, {item: $}]
`

func TestCheckProcedureGraph_CadenceOverNestedRunOnceStepIsFoundInInvokedProcedure(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/nightly.yaml", cadenceCaller),
		procedureRoot(t, "procedures/inner-once.yaml", innerRunOnce),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeCadenceRunOnce)
	if p.File != "procedures/nightly.yaml" {
		t.Errorf("File = %q, want procedures/nightly.yaml — the Procedure declaring the recurrence", p.File)
	}
	if p.Field != "cadence" {
		t.Errorf("Field = %q, want cadence", p.Field)
	}
}

func TestCheckProcedureGraph_RunOnceStepWithNoCadenceOnItsProcedureDrawsNoCode(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/inner-once.yaml", innerRunOnce),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	mustNoCode(t, got, CodeCadenceRunOnce)
}

const cadenceOverRepeatableOnly = `kind: procedure
procedure: nightly
targets: [local]
cadence: "0 3 * * *"
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: change
    definition: uptime
    operation: mutate
    target: local
    args:
      command: [uptime]
  - id: idempotent-change
    definition: uptime
    operation: mutate_skip_if_recorded
    target: local
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_CadenceOverRepeatableStepsOnlyDrawsNoCadenceRunOnce(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/nightly.yaml", cadenceOverRepeatableOnly),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	mustNoCode(t, got, CodeCadenceRunOnce)
	mustNoCode(t, got, CodeCadenceSecretOutput)
}

// --- cadence-secret-output ---

const cadenceOverSecretOutput = `kind: procedure
procedure: rotate
targets: [vault-prod]
cadence: "0 4 * * *"
steps:
  - id: issue
    definition: issue-token
    operation: issue_token
    target: vault-prod
    args: {}
`

func TestCheckProcedureGraph_CadenceOverOwnSecretOutputStepIsCadenceSecretOutput(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/rotate.yaml", cadenceOverSecretOutput),
	}, secretProviders(t), secretDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeCadenceSecretOutput)
	if p.Field != "cadence" {
		t.Errorf("Field = %q, want cadence", p.Field)
	}
}

// TestCheckProcedureGraph_CadenceOverNestedSecretOutputStepIsFoundInInvokedProcedure
// proves cadence-secret-output rides the same transitive walk
// cadence-run-once does: the secret-output Step sits in the nested
// Procedure, not the one carrying the clock, and the row still cites the
// clock's own file and line — cadence-run-once's own acceptance criterion
// (§4, §5), read for the second Cadence rule.
const cadenceSecretCaller = `kind: procedure
procedure: rotate
targets: [vault-prod]
cadence: "0 4 * * *"
steps:
  - id: call-inner
    procedure: issue-inner
`

const innerSecretOutput = `kind: procedure
procedure: issue-inner
targets: [vault-prod]
steps:
  - id: issue
    definition: issue-token
    operation: issue_token
    target: vault-prod
    args: {}
`

func TestCheckProcedureGraph_CadenceOverNestedSecretOutputStepIsFoundInInvokedProcedure(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/rotate.yaml", cadenceSecretCaller),
		procedureRoot(t, "procedures/issue-inner.yaml", innerSecretOutput),
	}, secretProviders(t), secretDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeCadenceSecretOutput)
	if p.File != "procedures/rotate.yaml" {
		t.Errorf("File = %q, want procedures/rotate.yaml — the Procedure declaring the recurrence", p.File)
	}
	if p.Field != "cadence" {
		t.Errorf("Field = %q, want cadence", p.Field)
	}
}

// TestCheckProcedureGraph_TwoCadenceCodesAreNeverMerged proves
// cadence-run-once and cadence-secret-output are two codes and are never
// merged: a Procedure whose Cadence reaches both a run-once Step and a
// secret-output Step earns both rows.
func TestCheckProcedureGraph_TwoCadenceCodesAreNeverMerged(t *testing.T) {
	providers := shellProviders()
	for name, info := range secretProviders(t) {
		providers[name] = info
	}
	definitions := uptimeDefinitions()
	for name, info := range secretDefinitions() {
		definitions[name] = info
	}

	doc := `kind: procedure
procedure: both
targets: [local, vault-prod]
cadence: "0 5 * * *"
steps:
  - id: change
    definition: uptime
    operation: mutate_once
    target: local
    args:
      command: [uptime]
  - id: issue
    definition: issue-token
    operation: issue_token
    target: vault-prod
    args: {}
`
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/both.yaml", doc),
	}, providers, definitions)

	got := CheckProcedureGraph(graph)
	mustCode(t, got, CodeCadenceRunOnce)
	mustCode(t, got, CodeCadenceSecretOutput)
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2 — one row per code, never merged", len(got))
	}
}

func TestCheckProcedureGraph_CadenceOverNoSecretOutputDrawsNoCadenceSecretOutput(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/nightly.yaml", cadenceOverRepeatableOnly),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	mustNoCode(t, got, CodeCadenceSecretOutput)
}

// TestCheckProcedureGraph_InvocationCycleDoesNotHang proves the walk
// guards against a Procedure invoking itself, directly or through a cycle,
// rather than recursing forever — a defensive property this issue does not
// name a code for, but that every genuine "to any depth" walk needs.
const cycleA = `kind: procedure
procedure: cycle-a
targets: [local]
steps:
  - id: call-b
    procedure: cycle-b
`

const cycleB = `kind: procedure
procedure: cycle-b
targets: [local]
steps:
  - id: call-a
    procedure: cycle-a
`

func TestCheckProcedureGraph_InvocationCycleDoesNotHang(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/cycle-a.yaml", cycleA),
		procedureRoot(t, "procedures/cycle-b.yaml", cycleB),
	}, shellProviders(), uptimeDefinitions())

	// A hang here fails the test on go test's own default timeout — the
	// property under test is termination, not any particular row.
	CheckProcedureGraph(graph)
}

func TestCheckProcedureGraph_NoInvocationsAndNoCadenceIsClean(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/deploy.yaml", minimalProcedure),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	if len(got) != 0 {
		t.Fatalf("CheckProcedureGraph() = %+v, want no problems", got)
	}
}
