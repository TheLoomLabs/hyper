package artefact

import (
	"slices"
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

// --- procedure-cycle: an invocation graph that closes on itself ---

// TestCheckProcedureGraph_ProcedureInvokingItselfIsProcedureCycle is the
// smallest cycle there is: a file whose own `steps:` invoke the Procedure
// that file declares. §6 says a cycle is rejected **before the first Step**,
// and this is the walk that can say so — it reads every procedures/ file at
// once and already knows which Procedures it is inside of (issue #146).
const selfInvoking = `kind: procedure
procedure: loop
targets: [local]
steps:
  - id: call-itself
    procedure: loop
`

func TestCheckProcedureGraph_ProcedureInvokingItselfIsProcedureCycle(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/loop.yaml", selfInvoking),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeProcedureCycle)
	if p.File != "procedures/loop.yaml" {
		t.Errorf("File = %q, want procedures/loop.yaml", p.File)
	}
	// The invocation entry itself — `    procedure: loop` on the sixth
	// line, its value at the sixteenth column — which is the line an
	// author edits to break the loop.
	if p.Line != 6 || p.Column != 16 {
		t.Errorf("cited %d:%d, want 6:16 — the invocation entry that closes the loop", p.Line, p.Column)
	}
	if p.Field != "steps[0].procedure" {
		t.Errorf("Field = %q, want steps[0].procedure", p.Field)
	}
}

// TestCheckProcedureGraph_CycleThroughTwoIsCitedWhereItCloses is the same
// fault one hop longer, and it is where the citation rule earns its keep: the
// walk enters at `cycle-a` because that is the name it reaches first, and the
// entry that closes the loop is in `cycle-b`. A row against the Procedure the
// walk happened to enter at would send an author to a file whose invocation
// is fine.
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

func TestCheckProcedureGraph_CycleThroughTwoIsCitedWhereItCloses(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/cycle-a.yaml", cycleA),
		procedureRoot(t, "procedures/cycle-b.yaml", cycleB),
	}, shellProviders(), uptimeDefinitions())

	// A hang here fails the test on go test's own default timeout: the walk
	// terminating is the property every "to any depth" rule below needs,
	// and the row is what this issue adds to it.
	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeProcedureCycle)
	if p.File != "procedures/cycle-b.yaml" {
		t.Errorf("File = %q, want procedures/cycle-b.yaml — the invocation that closes the loop, not the one the walk entered at", p.File)
	}
	if p.Line != 6 || p.Column != 16 {
		t.Errorf("cited %d:%d, want 6:16", p.Line, p.Column)
	}
}

// TestCheckProcedureGraph_CycleThroughThreeIsCitedWhereItCloses is the
// chain one hop longer again, which is what says the walk is transitive
// rather than a test of the two names in hand.
const chainA = `kind: procedure
procedure: chain-a
targets: [local]
steps:
  - id: call-b
    procedure: chain-b
`

const chainB = `kind: procedure
procedure: chain-b
targets: [local]
steps:
  - id: call-c
    procedure: chain-c
`

const chainC = `kind: procedure
procedure: chain-c
targets: [local]
steps:
  - id: call-a
    procedure: chain-a
`

func TestCheckProcedureGraph_CycleThroughThreeIsCitedWhereItCloses(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/chain-a.yaml", chainA),
		procedureRoot(t, "procedures/chain-b.yaml", chainB),
		procedureRoot(t, "procedures/chain-c.yaml", chainC),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	p := mustCode(t, got, CodeProcedureCycle)
	if p.File != "procedures/chain-c.yaml" {
		t.Errorf("File = %q, want procedures/chain-c.yaml — the third hop, which is where the loop closes", p.File)
	}
	if len(got) != 1 {
		t.Errorf("CheckProcedureGraph() = %+v, want one row — a cycle is one fault however many hops it takes", got)
	}
}

// TestCheckProcedureGraph_DiamondIsNotACycle is the case a naive
// already-seen test refuses and this one must not: `top` invokes `left` and
// `leaf`, and `left` invokes `leaf` too. The walk meets `leaf` twice and the
// graph is acyclic, which is a composition an author is entitled to write.
const diamondTop = `kind: procedure
procedure: top
targets: [local]
steps:
  - id: call-left
    procedure: left
  - id: call-leaf
    procedure: leaf
`

const diamondLeft = `kind: procedure
procedure: left
targets: [local]
steps:
  - id: call-leaf
    procedure: leaf
`

const diamondLeaf = `kind: procedure
procedure: leaf
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestCheckProcedureGraph_DiamondIsNotACycle(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/top.yaml", diamondTop),
		procedureRoot(t, "procedures/left.yaml", diamondLeft),
		procedureRoot(t, "procedures/leaf.yaml", diamondLeaf),
	}, shellProviders(), uptimeDefinitions())

	got := CheckProcedureGraph(graph)
	mustNoCode(t, got, CodeProcedureCycle)
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

// --- Reached: the two facts the projection reads off the same walk (issue #176) ---

// twoStepsOneBinding is a Procedure whose two Steps bind one (Definition,
// Target) pair between them — the shape that proves Reached answers what a
// Procedure *binds* rather than what it does.
const twoStepsOneBinding = `kind: procedure
procedure: watch
targets: [local]
cadence: "*/15 * * * *"
steps:
  - id: first
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: second
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestProcedureGraph_ReachedNamesEachBindingOnce(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/watch.yaml", twoStepsOneBinding),
	}, shellProviders(), uptimeDefinitions())

	got := graph.Reached("watch")
	want := []Binding{{Definition: "uptime", Target: "local"}}
	if !slices.Equal(got.Bindings, want) {
		t.Errorf("Bindings = %+v, want %+v — two Steps against one pair bind it once", got.Bindings, want)
	}
}

// outerReadsInnerEffects is the pair the concurrency group turns on: an outer
// Procedure whose own Steps are all `read`, reaching a `mutate` only through
// the Procedure it invokes.
const outerReadsInnerEffects = `kind: procedure
procedure: outer
targets: [local, other]
cadence: "0 3 * * 1"
steps:
  - id: look
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: call-inner
    procedure: inner
`

const innerMutates = `kind: procedure
procedure: inner
targets: [other]
steps:
  - id: touch
    definition: uptime
    operation: mutate
    target: other
    args:
      command: [touch, /tmp/x]
`

func TestProcedureGraph_ReachedEveryStepReadsIsTrueWhereEveryReachableStepIsRead(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/watch.yaml", twoStepsOneBinding),
	}, shellProviders(), uptimeDefinitions())

	if !graph.Reached("watch").EveryStepReads {
		t.Error("EveryStepReads = false, want true — every reachable Step declares kind: read")
	}
}

func TestProcedureGraph_ReachedEveryStepReadsIsFalseWhereANestedInvocationEffects(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/outer.yaml", outerReadsInnerEffects),
		procedureRoot(t, "procedures/inner.yaml", innerMutates),
	}, shellProviders(), uptimeDefinitions())

	if graph.Reached("outer").EveryStepReads {
		t.Error("EveryStepReads = true, want false — reachability decides, and the invoked Procedure mutates")
	}
}

func TestProcedureGraph_ReachedFollowsANestedInvocationsBindings(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/outer.yaml", outerReadsInnerEffects),
		procedureRoot(t, "procedures/inner.yaml", innerMutates),
	}, shellProviders(), uptimeDefinitions())

	got := graph.Reached("outer")
	want := []Binding{
		{Definition: "uptime", Target: "local"},
		{Definition: "uptime", Target: "other"},
	}
	if !slices.Equal(got.Bindings, want) {
		t.Errorf("Bindings = %+v, want %+v — an invocation binds nothing of its own and its Steps' pairs are the caller's", got.Bindings, want)
	}
}

// twoTargetsOneDefinition is what orders the walk's answer: two Steps binding
// one Definition against two Targets, in the order the Steps are written.
const twoTargetsOneDefinition = `kind: procedure
procedure: sweep
targets: [local, other]
steps:
  - id: second-written
    definition: uptime
    operation: read
    target: other
    args:
      command: [uptime]
  - id: first-written
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestProcedureGraph_ReachedHoldsBindingsInTheStepsOwnOrder(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/sweep.yaml", twoTargetsOneDefinition),
	}, shellProviders(), uptimeDefinitions())

	got := graph.Reached("sweep")
	want := []Binding{
		{Definition: "uptime", Target: "other"},
		{Definition: "uptime", Target: "local"},
	}
	if !slices.Equal(got.Bindings, want) {
		t.Errorf("Bindings = %+v, want %+v — the Steps' own order, so one repository answers one way", got.Bindings, want)
	}
}

// bindsNothingLegible is the Step whose binding does not resolve: its
// Operation carries no Kind for this walk to read.
const bindsNothingLegible = `kind: procedure
procedure: broken
targets: [local]
steps:
  - id: unresolved
    definition: no-such-definition
    operation: read
    target: local
`

func TestProcedureGraph_ReachedCountsAnUnresolvedStepAsEffecting(t *testing.T) {
	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/broken.yaml", bindsNothingLegible),
	}, shellProviders(), uptimeDefinitions())

	if graph.Reached("broken").EveryStepReads {
		t.Error("EveryStepReads = true, want false — an Operation that does not resolve declares no read for this walk to read")
	}
}

func TestProcedureGraph_ReachedOfANameTheGraphDoesNotHoldIsEmpty(t *testing.T) {
	graph := BuildProcedureGraph(nil, shellProviders(), uptimeDefinitions())

	got := graph.Reached("absent")
	if len(got.Bindings) != 0 {
		t.Errorf("Bindings = %+v, want none", got.Bindings)
	}
	if !got.EveryStepReads {
		t.Error("EveryStepReads = false, want true — a Procedure with no Steps reaches no Step that effects")
	}
}

// twoDefinitionsOneTarget is what §10's dedup rule is quantified over: two
// Definitions binding one Target. The pairs stay two here — a slot belongs to
// the Target declaration, and it is the slots the two resolve to that come to
// one entry in the `env:` block, which is the caller's reading and not this
// walk's.
const twoDefinitionsOneTarget = `kind: procedure
procedure: pair
targets: [local]
steps:
  - id: first
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: second
    definition: heartbeat
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestProcedureGraph_ReachedKeepsTwoDefinitionsAgainstOneTargetAsTwoPairs(t *testing.T) {
	definitions := uptimeDefinitions()
	definitions["heartbeat"] = definitions["uptime"]

	graph := BuildProcedureGraph([]ProcedureRoot{
		procedureRoot(t, "procedures/pair.yaml", twoDefinitionsOneTarget),
	}, shellProviders(), definitions)

	got := graph.Reached("pair")
	want := []Binding{
		{Definition: "uptime", Target: "local"},
		{Definition: "heartbeat", Target: "local"},
	}
	if !slices.Equal(got.Bindings, want) {
		t.Errorf("Bindings = %+v, want %+v — the pair is what a Run binds, and both are pairs", got.Bindings, want)
	}
}
