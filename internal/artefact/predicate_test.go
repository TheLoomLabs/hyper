// This file tests issue #97: the closed eleven-member operator set's own
// operand-type rules, the three roots a field: reads against, the three
// over: forms, skip-if-recorded-unreachable, and the offline half of
// bound-exceeded. It reuses retirePreviewDNS's own fixtures
// (cloudflareProcedureProviders, previewDNSDefinitions, cloudflareTargets)
// for the Record-root checks, and a small secretco fixture of its own for
// the skip-if-recorded and secret: field cases neither of retirePreviewDNS's
// two Operations carries.
package artefact

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// secretco is a small Manifest fixture carrying a skip-if-recorded mutate
// with a secret: field — the two facts neither of retirePreviewDNS's two
// Operations carries, and the two this file's own tests need.
const secretco = `kind: provider
provider: secretco
schema-version: 1
class: secretco
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  create_widget:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /widgets
      body: {name: "{name}"}
    input:
      type: object
      properties:
        name: {type: string}
    record:
      identity: "{name}"
      fields:
        id: $.body.id
        name: $.body.name
        token: $.body.token
    secret: [token]
  destroy_widget:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /widgets/{widget_id}
    input:
      type: object
      properties:
        widget_id: {type: string}
`

func secretcoProviders(t *testing.T) ProviderIndex {
	t.Helper()
	return BuildProviderIndex([]*yaml.Node{parse(t, secretco)})
}

func secretcoDefinitions() DefinitionIndex {
	return DefinitionIndex{"widget": DefinitionInfo{
		ProviderName: "secretco",
		Kinds:        map[string]bool{"mutate": true},
		Destroy:      map[string]bool{"destroy_widget": true},
		Targets:      map[string]TargetInfo{"secretco-prod": fullyGrantedTarget()},
	}}
}

func secretcoTargets() TargetIndex {
	return TargetIndex{"secretco-prod": fullyGrantedTarget()}
}

// findProblem returns the first problem in got carrying both code and
// field, so a test asserting a specific position does not depend on order
// among several problems a doc might carry.
func findProblem(got []problem.Problem, code, field string) (problem.Problem, bool) {
	for _, p := range got {
		if p.ErrorCode == code && p.Field == field {
			return p, true
		}
	}
	return problem.Problem{}, false
}

func mustProblemAt(t *testing.T, got []problem.Problem, code, field string) {
	t.Helper()
	if _, ok := findProblem(got, code, field); !ok {
		t.Fatalf("got %+v, want a %s at %s", got, code, field)
	}
}

// retireDocWithPredicate builds retire-preview-dns with its retire Step's
// over: assets: replaced by a single predicate entry, written in flow style
// so each test states only the one predicate it means to exercise — the
// resolution namespace stays previewDNS's own (id, name, created_on, none
// of it secret), so every operand-type test below is isolated from the
// field-resolution and secret: checks exercised separately.
func retireDocWithPredicate(predicate string) string {
	return fmt.Sprintf(`kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      assets:
        - %s
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $.id}
    bound: 5
`, predicate)
}

func checkRetirePredicate(t *testing.T, predicate string) []problem.Problem {
	t.Helper()
	doc := retireDocWithPredicate(predicate)
	return CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
}

// --- The eleven operators load, and a two-operator entry is schema-mismatch. ---

func TestCheckProcedure_PredicateOneCleanOperatorPerLine(t *testing.T) {
	for _, predicate := range []string{
		`{field: name, equals: preview-42.example.com}`,
		`{field: name, not_equals: something-else}`,
		`{field: name, in: [preview-42.example.com, preview-43.example.com]}`,
		`{field: name, exists: true}`,
		`{field: name, absent: true}`,
		`{field: name, starts_with: preview-}`,
		`{field: name, ends_with: .example.com}`,
		`{field: id, greater_than: 5}`,
		`{field: id, less_than: 5}`,
		`{field: created_on, older_than: 14d}`,
		`{field: created_on, newer_than: 14d}`,
	} {
		got := checkRetirePredicate(t, predicate)
		if len(got) != 0 {
			t.Errorf("predicate %s: CheckProcedure() = %+v, want no problems", predicate, got)
		}
	}
}

func TestCheckProcedure_PredicateTwoOperatorsIsSchemaMismatch(t *testing.T) {
	got := checkRetirePredicate(t, `{field: name, equals: preview-42.example.com, not_equals: other}`)
	mustCode(t, got, schema.CodeMismatch)
}

func TestCheckProcedure_PredicateDisjunctionKeyIsUnknownKey(t *testing.T) {
	got := checkRetirePredicate(t, `{field: name, equals: preview-42.example.com, any_of: []}`)
	mustProblemAt(t, got, schema.CodeUnknownKey, "steps[0].over.assets[0].any_of")
}

// --- field: at a selector or condition root is one declared field name. ---

func TestCheckProcedure_SelectorFieldAsPathIsSchemaMismatch(t *testing.T) {
	got := checkRetirePredicate(t, `{field: $.name, equals: preview-42.example.com}`)
	mustProblemAt(t, got, schema.CodeMismatch, "steps[0].over.assets[0].field")
}

func TestCheckProcedure_SelectorFieldUnresolvableIsReferenceUnresolvable(t *testing.T) {
	got := checkRetirePredicate(t, `{field: nonexistent, equals: x}`)
	mustProblemAt(t, got, CodeReferenceUnresolvable, "steps[0].over.assets[0].field")
}

func TestCheckProcedure_SelectorFieldResolvesAgainstProviderUnion(t *testing.T) {
	// delete_dns_record itself declares no record: at all — name and
	// created_on are projected only by create_dns_record and
	// list_dns_records, the Provider's other two Operations. A selector on
	// the destroy Step still resolves both, which is what the retire Step
	// in retirePreviewDNS (already exercised by
	// TestCheckProcedure_RetirePreviewDNSIsClean) proves in full; this test
	// isolates the one fact that proof depends on.
	got := checkRetirePredicate(t, `{field: created_on, older_than: 14d}`)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// --- A predicate against a field the Manifest declares secret. ---

func TestCheckProcedure_PredicateAgainstSecretFieldIsTypeMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [secretco-prod]
steps:
  - id: destroy
    definition: widget
    operation: destroy_widget
    target: secretco-prod
    over:
      assets:
        - {field: token, equals: abc}
    args:
      widget_id: {item: $.id}
    bound: 5
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), secretcoProviders(t), secretcoDefinitions(), secretcoTargets(), ProcedureIndex{})
	mustProblemAt(t, got, CodePredicateTypeMismatch, "steps[0].over.assets[0].field")
}

// --- Every operand fault §12 fixes. ---

func TestCheckProcedure_PredicateOperandFaultsAreTypeMismatch(t *testing.T) {
	cases := []string{
		`{field: created_on, greater_than: "2024-01-01T00:00:00Z"}`, // timestamp under greater_than
		`{field: created_on, less_than: "2024-01-01T00:00:00Z"}`,    // timestamp under less_than
		`{field: name, in: []}`,                                     // empty in:
		`{field: name, in: [preview-42]}`,                           // one-member in:
		`{field: name, in: [preview-42, 5]}`,                        // mixed-type in:
		`{field: name, exists: false}`,                              // negation in the operand
		`{field: name, absent: false}`,                              // same fault, the other operator
		`{field: name, starts_with: ""}`,                            // empty starts_with:
		`{field: name, ends_with: ""}`,                              // empty ends_with:
	}
	for _, predicate := range cases {
		got := checkRetirePredicate(t, predicate)
		found := false
		for _, p := range got {
			if p.ErrorCode == CodePredicateTypeMismatch {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("predicate %s: got %+v, want a %s", predicate, got, CodePredicateTypeMismatch)
		}
	}
}

func TestCheckProcedure_PredicateGreaterThanDurationIsClean(t *testing.T) {
	// greater_than/less_than take integer/number and duration, never a
	// timestamp; a duration operand is legal on both, unlike older_than/
	// newer_than's own pair.
	got := checkRetirePredicate(t, `{field: id, greater_than: 5m}`)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_PredicateOlderThanNumberIsTypeMismatch(t *testing.T) {
	// older_than/newer_than take duration and timestamp only, never a bare
	// number — the pair greater_than/less_than own instead.
	got := checkRetirePredicate(t, `{field: id, older_than: 5}`)
	mustCode(t, got, CodePredicateTypeMismatch)
}

// --- A condition (when:) carries step: beside field:; a selector does not. ---

func TestCheckProcedure_SelectorEntryWithStepIsUnknownKey(t *testing.T) {
	got := checkRetirePredicate(t, `{field: name, step: retire, equals: preview-42.example.com}`)
	mustProblemAt(t, got, schema.CodeUnknownKey, "steps[0].over.assets[0].step")
}

func TestCheckProcedure_WhenConditionClean(t *testing.T) {
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
  - id: notify
    definition: uptime
    operation: read
    target: local
    when:
      step: probe
      field: exit_code
      equals: "0"
    args:
      command: [echo, ok]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_WhenWithoutStepIsSchemaMismatch(t *testing.T) {
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
  - id: notify
    definition: uptime
    operation: read
    target: local
    when:
      field: exit_code
      equals: "0"
    args:
      command: [echo, ok]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	mustProblemAt(t, got, schema.CodeMismatch, "steps[1].when.step")
}

func TestCheckProcedure_WhenStepUnresolvableIsReferenceUnresolvable(t *testing.T) {
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
  - id: notify
    definition: uptime
    operation: read
    target: local
    when:
      step: nowhere
      field: exit_code
      equals: "0"
    args:
      command: [echo, ok]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	mustProblemAt(t, got, CodeReferenceUnresolvable, "steps[1].when.step")
}

// --- The three over: forms. ---

func TestCheckProcedure_OverTwoFormsAtOnceIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values: [5b2d84f16c0a39e7d5182bfa604c7e93]
      assets:
        - {field: name, equals: preview-42.example.com}
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 3
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustProblemAt(t, got, schema.CodeMismatch, "steps[0].over")
}

func TestCheckProcedure_OverFourthFormIsUnknownKey(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values: [5b2d84f16c0a39e7d5182bfa604c7e93]
      any_of: []
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 3
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustProblemAt(t, got, schema.CodeUnknownKey, "steps[0].over.any_of")
}

func TestCheckProcedure_ObservationsOnNonReadIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      observations:
        - {field: name, equals: preview-42.example.com}
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $.id}
    bound: 5
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustProblemAt(t, got, schema.CodeMismatch, "steps[0].over.observations")
}

func TestCheckProcedure_ObservationsOnReadIsClean(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: list
    definition: preview-dns
    operation: list_dns_records
    target: cloudflare-prod
    over:
      observations:
        - {field: name, equals: preview-42.example.com}
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_ValuesMappingMemberIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire-legacy
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values:
        - {foo: bar}
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 3
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustProblemAt(t, got, schema.CodeMismatch, "steps[0].over.values[0]")
}

func TestCheckProcedure_ValuesLegalOnReadKind(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: list
    definition: preview-dns
    operation: list_dns_records
    target: cloudflare-prod
    over:
      values: [dummy]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustNoCode(t, got, schema.CodeMismatch)
}

// --- A Step declaring no over: loads clean and is invoked once. ---

func TestCheckProcedure_NoOverDrawsNoOverRelatedCode(t *testing.T) {
	got := CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// --- A skip-if-recorded Step expanding over assets: is unreachable. ---

func TestCheckProcedure_SkipIfRecordedOverAssetsIsUnreachable(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [secretco-prod]
steps:
  - id: create
    definition: widget
    operation: create_widget
    target: secretco-prod
    over:
      assets:
        - {field: name, equals: foo}
    args:
      name: foo
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), secretcoProviders(t), secretcoDefinitions(), secretcoTargets(), ProcedureIndex{})
	mustProblemAt(t, got, CodeSkipIfRecordedUnreachable, "steps[0]")
}

func TestCheckProcedure_SkipIfRecordedOverValuesIsReachable(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [secretco-prod]
steps:
  - id: create
    definition: widget
    operation: create_widget
    target: secretco-prod
    over:
      values: [widget-a, widget-b]
    args:
      name: {item: $}
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), secretcoProviders(t), secretcoDefinitions(), secretcoTargets(), ProcedureIndex{})
	mustNoCode(t, got, CodeSkipIfRecordedUnreachable)
}

// --- bound-exceeded, decided offline from an over: values: list's length. ---

func TestCheckProcedure_OverValuesLongerThanBoundIsBoundExceeded(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire-legacy
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values:
        - 5b2d84f16c0a39e7d5182bfa604c7e93
        - 8f1a2c4d6e8b0a2c4e6f8a0b2c4d6e8f
        - c3d5e7f9a1b3c5d7e9f1a3b5c7d9e1f3
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 2
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustProblemAt(t, got, CodeBoundExceeded, "steps[0].over.values")
}

func TestCheckProcedure_BoundExceededNotEmittedAgainstAssets(t *testing.T) {
	// retire's own bound: 5 covers only two assets: predicates and no
	// authored count at all — bound-exceeded needs an authored length, and
	// an assets: selector's size is on no file (§5, §6, §12).
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, retirePreviewDNS), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	mustNoCode(t, got, CodeBoundExceeded)
}
