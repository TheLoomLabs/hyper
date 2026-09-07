package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/repository"
)

// **The Secret sink gate reads two operands and both of them can be wrong**
// (§6, §9, §12, issue #275, ADR-0151).
//
// One direction has been held since the writer landed: a Run reaching a Step
// whose Operation declares secret output with no sink named Refuses
// `secret-sink-absent`, and the corpus drives it. The converse was silence by
// design — a sink named against a Procedure that produces none left no empty
// directory behind, which is a tidiness argument written before there was a
// reason to speak. It is what made ADR-0149's loss invisible: `hyper` held both
// facts at run start and said nothing.
//
// These are unit tests for the gate's four combinations, where a corpus case
// would need a fixture repository, a served host and a checked-in Store per
// combination to say the same four things.

// sinkGateFor is the gate's answer over one repository, for one Procedure, with
// a sink named or not. The two Procedures of sinkGateFixture are the gate's
// other operand: `mint` binds an Operation declaring `secret:` output and
// `watch` binds one that does not.
func sinkGateFor(t *testing.T, loaded repository.Loaded, procedure string, sink secretSink) []Refusal {
	t.Helper()
	entry, held := loaded.Procedure(procedure)
	if !held {
		t.Fatalf("the fixture holds no Procedure named %s", procedure)
	}
	return sinkRefusals(loaded, entry, secretOutputSteps(loaded, flatten(loaded, procedure).Steps), sink)
}

// sinkGateFixture is one repository holding both operands: a Manifest whose
// `mint_token` declares `secret: [token]` and whose `check_http` does not, and
// one Procedure binding each.
//
// It is written here rather than borrowed from sequenceFixture because that one
// resolves no Definition at all, so every Step of it answers *declares no
// secret output* for the wrong reason — which would leave two of the gate's four
// combinations unreachable from this package.
func sinkGateFixture(t *testing.T) repository.Loaded {
	t.Helper()

	root := t.TempDir()
	for path, content := range map[string]string{
		"hyper.yaml": "kind: repository-declaration\nversion: 1.4.0\nretention: 90d\n",
		"targets/local.yaml": "kind: target-declaration\ntarget: local\nclass: local\n" +
			"kinds: [read]\ncapabilities: [http]\nhosts: [status.hyper.dev]\n",
		"definitions/session.yaml": "kind: definition\ndefinition: session\nprovider: session\n" +
			"kinds: [read]\ntargets: [local]\n",
		"providers/session.yaml": `kind: provider
provider: session
schema-version: 1
class: local
capabilities: [http]
operations:
  mint_token:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /session}
    record:
      identity: $.host
      fields: {token: $.body.token}
    secret: [token]
  check_http:
    kind: read
    deadline: 10s
    http: {method: GET, host: "{from-target}", path: /}
    record:
      identity: $.host
      fields: {status: $.status}
`,
		"procedures/mint.yaml": "kind: procedure\nprocedure: mint\ntargets: [local]\nsteps:\n" +
			"  - {id: token, definition: session, operation: mint_token, target: local}\n",
		"procedures/watch.yaml": "kind: procedure\nprocedure: watch\ntargets: [local]\nsteps:\n" +
			"  - {id: up, definition: session, operation: check_http, target: local}\n",
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := repository.Load(root)
	if err != nil {
		t.Fatalf("loading the fixture: %v", err)
	}
	return loaded
}

// TestSinkGate_ANamedSinkNoStepCanFillRefusesTheRun is the converse gate. The
// Refusal cites the Procedure the invocation named, at the line naming it: the
// two operands are an invocation and a Procedure, and the Procedure is the half
// with a file.
func TestSinkGate_ANamedSinkNoStepCanFillRefusesTheRun(t *testing.T) {
	declined := sinkGateFor(t, sinkGateFixture(t), "watch", secretSink{root: "/somewhere/outside"})
	if len(declined) != 1 {
		t.Fatalf("the gate answered %d Refusals, want 1: %+v", len(declined), declined)
	}

	member := declined[0]
	if member.ErrorCode != CodeSecretSinkUnfilled {
		t.Errorf("ErrorCode = %q, want %q", member.ErrorCode, CodeSecretSinkUnfilled)
	}
	if member.File != "procedures/watch.yaml" || member.Line != 2 || member.Field != "procedure" {
		t.Errorf("cited %s:%d %s, want procedures/watch.yaml:2 procedure", member.File, member.Line, member.Field)
	}
	// It cites no Step, which is the truthful coordinate: no Step of the
	// Run is at fault and naming one would send a reader to edit a Step
	// that is correct (§7, ADR-0061).
	if member.Step != 0 || member.StepID != "" {
		t.Errorf("the Refusal cites step %d %q, want no Step", member.Step, member.StepID)
	}
	if !strings.Contains(member.Message, "secret:") {
		t.Errorf("Message = %q, want it to name the secret: declaration", member.Message)
	}
}

// TestSinkGate_AProducingStepWithNoSinkNamesEveryStep is the older half, held
// here beside its converse so the gate's four combinations read as four rather
// than as one rule and an exception. The corpus drives its rendering; what is
// asserted here is that it is the same walk answering both.
func TestSinkGate_AProducingStepWithNoSinkNamesEveryStep(t *testing.T) {
	declined := sinkGateFor(t, sinkGateFixture(t), "mint", secretSink{})
	if len(declined) != 1 {
		t.Fatalf("the gate answered %d Refusals, want 1: %+v", len(declined), declined)
	}
	if got := declined[0].ErrorCode; got != CodeSecretSinkAbsent {
		t.Errorf("ErrorCode = %q, want %q", got, CodeSecretSinkAbsent)
	}
	// This half cites the Step, and it is the one code in the set that cites
	// a Step it did not reach (§7, refusal.go).
	if declined[0].Step != 1 || declined[0].StepID != "token" {
		t.Errorf("cited step %d %q, want step 1 token", declined[0].Step, declined[0].StepID)
	}
}

// TestSinkGate_TheTwoAgreeingCombinationsAreOrdinaryRuns is the fence the check
// is worth nothing without: a sink for the Steps that will fill it, and neither.
// The second is the overwhelming majority of Runs, and the gate has nothing to
// say about either.
func TestSinkGate_TheTwoAgreeingCombinationsAreOrdinaryRuns(t *testing.T) {
	loaded := sinkGateFixture(t)
	for named, c := range map[string]struct {
		procedure string
		sink      secretSink
	}{
		"neither a sink nor a producing Step": {procedure: "watch"},
		"a sink and the Step that fills it":   {procedure: "mint", sink: secretSink{root: "/somewhere/outside"}},
	} {
		if declined := sinkGateFor(t, loaded, c.procedure, c.sink); len(declined) != 0 {
			t.Errorf("%s was refused: %+v", named, declined)
		}
	}
}
