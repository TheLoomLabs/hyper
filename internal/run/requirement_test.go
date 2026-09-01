package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **A `require:` halts where it does not hold**, which is what lets a shared
// read-only check stop the Procedure that invoked it (§6, issue #236,
// ADR-0116).
//
// What the corpus drives is the Run: the halt, the Steps after it *never
// reached*, and the exit code. What is held here is the walk that places a
// Requirement in written order without giving it a position, and the verdict
// itself — which is a pure function of a predicate and what a Step acted on.

// requiring reads a `require:` the way a Requirement carries one, from the
// text an author would write.
func requiring(t *testing.T, text string) requirement {
	t.Helper()

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(text), &node); err != nil {
		t.Fatalf("reading %q: %v", text, err)
	}
	return requirement{Require: node.Content[0], ID: "sound", Index: 1, Line: 9}
}

// actingOn is a Run holding what one Step acted on, which is the whole of what
// a verdict reads.
func actingOn(records ...store.Mapping) run {
	return run{
		started: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		acted:   acted{{namespace: 0, id: "probe"}: records},
	}
}

func TestRequirement_HoldsOfTheRecordTheNamedStepActedOn(t *testing.T) {
	declined, err := actingOn(store.Mapping{"exit_code": store.Int(0)}).
		verdict(requiring(t, "{step: probe, field: exit_code, equals: 0}"))
	if err != nil {
		t.Fatalf("verdict() = %v, want the Run to go on", err)
	}
	if len(declined) > 0 {
		t.Fatalf("verdict() declined %+v, want nothing", declined)
	}
}

// The halt, and the sentence a reader acts on: which Requirement, which Step
// it read, and that nothing after it runs.
func TestRequirement_HaltsWhereItDoesNotHold(t *testing.T) {
	declined, err := actingOn(store.Mapping{"exit_code": store.Int(1)}).
		verdict(requiring(t, "{step: probe, field: exit_code, equals: 0}"))
	if len(declined) > 0 {
		t.Fatalf("verdict() declined %+v, want a halt and no Refusal", declined)
	}
	if err == nil {
		t.Fatal("verdict() = nil, want a halt")
	}
	for _, want := range []string{"sound", "exit_code", "probe", "equals: 0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the halt reads %q, want it to name %q", err, want)
		}
	}
}

// **A Step that acted on nothing leaves the requirement unmet**, which is
// condition.go's rule reaching this key's outcome — and the message says so
// rather than reading as a comparison that failed.
// The operand is what the halt names as *what it compared*: the artefact's half
// of the test, which is what an author edits. An `in:` list renders whole for
// the same reason a scalar does.
func TestRequirement_TheHaltNamesTheTestAsItWasWritten(t *testing.T) {
	_, err := actingOn(store.Mapping{"exit_code": store.Int(3)}).
		verdict(requiring(t, "{step: probe, field: exit_code, in: [0, 1]}"))
	if err == nil {
		t.Fatal("verdict() = nil, want a halt")
	}
	if !strings.Contains(err.Error(), "in: [0, 1]") {
		t.Errorf("the halt reads %q, want it to name the operand as written", err)
	}
}

func TestRequirement_HaltsWhereTheNamedStepActedOnNothing(t *testing.T) {
	_, err := actingOn().verdict(requiring(t, "{step: probe, field: exit_code, equals: 0}"))
	if err == nil {
		t.Fatal("verdict() = nil, want a halt")
	}
	if !strings.Contains(err.Error(), "acted on no Record") {
		t.Errorf("the halt reads %q, want it to say the Step acted on no Record", err)
	}
}

// **A `require:` rooted at a Step that expanded holds of every Record it acted
// on** — a Step of `series` cardinality, or one ranging over an Expansion, is as
// legal a root as any other (§3, ADR-0126).
//
// The Run corpus drives the halt end to end; what is held here is the verdict,
// and the sentence it hands an author. That sentence is the whole of what makes
// the wider reading legible: an author who meant the Record of one Step and
// wrote the list read has authored a stricter test than the one they meant, and
// the count is where the difference shows.
func TestRequirement_HoldsOfEveryRecordTheStepActedOn(t *testing.T) {
	required := requiring(t, "{step: probe, field: state, equals: ready}")

	ready := actingOn(store.Mapping{"state": store.String("ready")}, store.Mapping{"state": store.String("ready")})
	if _, err := ready.verdict(required); err != nil {
		t.Fatalf("verdict() = %v, want the Run to go on where every Record satisfied the predicate", err)
	}

	mixed := actingOn(
		store.Mapping{"state": store.String("ready")},
		store.Mapping{"state": store.String("degraded")},
		store.Mapping{"state": store.String("ready")},
	)
	_, err := mixed.verdict(required)
	if err == nil {
		t.Fatal("verdict() = nil, want a halt where one of three Records did not satisfy the predicate")
	}
	// The population, the count, and the rule — and no member and no
	// observed value, which is what ADR-0035 keeps every predicate report
	// from naming.
	for _, want := range []string{"acted on 3 Records", "on 2 of them", "holds of every one"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the halt reads %q, want it to carry %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "degraded") {
		t.Errorf("the halt reads %q, want it to name no observed value (ADR-0035)", err)
	}
}

// A root that acted on one Record says so in the singular, which is every
// Requirement authored before an Expansion was a legal root.
func TestRequirement_NamesOneRecordInTheSingular(t *testing.T) {
	_, err := actingOn(store.Mapping{"exit_code": store.Int(1)}).
		verdict(requiring(t, "{step: probe, field: exit_code, equals: 0}"))
	if err == nil {
		t.Fatal("verdict() = nil, want a halt")
	}
	if !strings.Contains(err.Error(), "of the Record step probe acted on") {
		t.Errorf("the halt reads %q, want it to name one Record in the singular", err)
	}
}

// A predicate that cannot decide Refuses wherever it stands (ADR-0035), and it
// cites the artefact coordinate: the file, the Requirement's own entry, and
// the operator inside it. It carries no `step` — a Requirement takes no
// position in the Run to name (§7, ADR-0061).
func TestRequirement_RefusesAValueItCannotCompare(t *testing.T) {
	required := requiring(t, "{step: probe, field: age, older_than: 7d}")
	required.Declared = repository.LoadedArtefact{Path: "procedures/verify-archive.yaml"}

	declined, err := actingOn(store.Mapping{"age": store.Int(41)}).verdict(required)
	if err != nil {
		t.Fatalf("verdict() = %v, want a Refusal rather than a halt", err)
	}
	if len(declined) != 1 {
		t.Fatalf("verdict() declined %+v, want one Refusal", declined)
	}
	refusal := declined[0]
	if refusal.ErrorCode != CodePredicateTypeMismatch {
		t.Errorf("error_code = %q, want %q", refusal.ErrorCode, CodePredicateTypeMismatch)
	}
	if refusal.File != "procedures/verify-archive.yaml" {
		t.Errorf("file = %q, want procedures/verify-archive.yaml", refusal.File)
	}
	if refusal.Field != "steps[1].require.older_than" {
		t.Errorf("field = %q, want steps[1].require.older_than", refusal.Field)
	}
	if refusal.StepID != "sound" {
		t.Errorf("step_id = %q, want sound", refusal.StepID)
	}
	if refusal.Step != 0 {
		t.Errorf("step = %d, want none — a Requirement takes no position in the Run", refusal.Step)
	}
}

// **A Requirement takes no position in the sequence**, and it stands in front
// of the Step that follows it in written order — which, for one authored last
// inside an invoked Procedure, is the caller's next Step. That is the whole of
// how a shared check gates its callers.
func TestFlatten_PlacesARequirementInFrontOfTheStepThatFollowsIt(t *testing.T) {
	walked := flatten(requirementFixture(t), "promote")

	var order []string
	for _, step := range walked.Steps {
		order = append(order, named(step))
	}
	want := []string{"stage", "verify-archive.archive-sound", "swap"}
	if strings.Join(order, " ") != strings.Join(want, " ") {
		t.Fatalf("the walk reached %v, want %v", order, want)
	}

	if len(walked.Requirements) != 1 {
		t.Fatalf("the walk reached %d Requirements, want 1", len(walked.Requirements))
	}
	required := walked.Requirements[0]
	if namedRequirement(required) != "verify-archive.sound" {
		t.Errorf("the Requirement is named %q, want verify-archive.sound", namedRequirement(required))
	}
	if required.Before != 2 {
		t.Errorf("it stands before step %d, want 2 — the caller's own next Step", required.Before)
	}
	if required.Declared.Path != "procedures/verify-archive.yaml" || required.Index != 1 {
		t.Errorf("it is cited at %s steps[%d], want procedures/verify-archive.yaml steps[1]", required.Declared.Path, required.Index)
	}
	if required.Namespace != 1 {
		t.Errorf("it resolves ids in namespace %d, want the invoked Procedure's own", required.Namespace)
	}
}

// A Requirement authored after the last Step the Run holds stands after all of
// them, which is where the engine reads it: a Procedure nothing invoked, whose
// last entry is its own verdict.
func TestFlatten_PlacesATrailingRequirementAfterEveryStep(t *testing.T) {
	walked := flatten(requirementFixture(t), "verify-archive")
	if len(walked.Steps) != 1 {
		t.Fatalf("the walk reached %d Steps, want 1", len(walked.Steps))
	}
	if len(walked.Requirements) != 1 || walked.Requirements[0].Before != 1 {
		t.Fatalf("the Requirement stands before %+v, want the end of a one-Step sequence", walked.Requirements)
	}
	if walked.Requirements[0].Path != "" {
		t.Errorf("path = %q, want none on a top-level entry", walked.Requirements[0].Path)
	}
}

// requirementFixture is the shape issue #236 states: one shared, read-only
// check, invoked by a Procedure that writes — and the check halting on its own
// verdict rather than claiming `mutate` to be able to fail (ADR-0111).
func requirementFixture(t *testing.T) repository.Loaded {
	t.Helper()

	root := t.TempDir()
	for path, content := range map[string]string{
		"hyper.yaml": "kind: repository-declaration\nversion: 1.4.0\nretention: 90d\n",
		"procedures/promote.yaml": "kind: procedure\nprocedure: promote\ntargets: [local]\nsteps:\n" +
			"  - {id: stage, definition: d, operation: o, target: local}\n" +
			"  - {id: verify, procedure: verify-archive}\n" +
			"  - {id: swap, definition: d, operation: o, target: local}\n",
		"procedures/verify-archive.yaml": "kind: procedure\nprocedure: verify-archive\ntargets: [local]\nsteps:\n" +
			"  - {id: archive-sound, definition: d, operation: o, target: local}\n" +
			"  - {id: sound, require: {step: archive-sound, field: exit_code, equals: 0}}\n",
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
