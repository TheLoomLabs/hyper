package artefact

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// shellProviders and uptimeDefinitions are the namespaces a Step binding
// the built-in shell Provider through the uptime Definition resolves
// against — the same pairing internal/cli's clean fixture checks (§3, §11).
func shellProviders() ProviderIndex {
	return ProviderIndex{"shell": builtinShellProviderInfo()}
}

// fullyGrantedTarget is a synthetic TargetInfo accepting every Kind and
// opted into opaque-destroy: — the two-keys, Bound and opaque-opt-in checks
// are not what most of this file's tests exercise, so their fixtures grant
// everything a Definition might claim (issue #95); the tests that exercise
// those checks build their own narrower TargetInfo instead.
func fullyGrantedTarget() TargetInfo {
	return TargetInfo{Kinds: map[string]bool{"read": true, "mutate": true, "destroy": true}, OpaqueDestroy: true}
}

// localTargets is the TargetIndex uptimeDefinitions' own Targets pairs with —
// the namespace a Procedure's own targets: envelope resolves against wherever
// a test does not need a narrower grant of its own (issue #96).
func localTargets() TargetIndex {
	return TargetIndex{"local": fullyGrantedTarget()}
}

func uptimeDefinitions() DefinitionIndex {
	return DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Destroy:      map[string]bool{"destroy": true, "destroy_once": true},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
}

// cloudflareProcedureProviders and previewDNSDefinitions are the namespaces
// §3's worked Procedure, retire-preview-dns, resolves against —
// cloudflareDNS and previewDNS reused byte for byte (manifest_test.go,
// definition_test.go).
func cloudflareProcedureProviders(t *testing.T) ProviderIndex {
	t.Helper()
	return BuildProviderIndex([]*yaml.Node{parse(t, cloudflareDNS)})
}

func previewDNSDefinitions() DefinitionIndex {
	return DefinitionIndex{"preview-dns": DefinitionInfo{
		ProviderName: "cloudflare-dns",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Destroy:      map[string]bool{"delete_dns_record": true},
		Targets:      map[string]TargetInfo{"cloudflare-prod": fullyGrantedTarget()},
	}}
}

const minimalProcedure = `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
`

func TestCheckProcedure_Clean(t *testing.T) {
	got := CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_CommentsAreIgnored(t *testing.T) {
	doc := "# a comment above the document\n" + minimalProcedure + "    # a comment beside a Step's own lines\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_KindMismatch(t *testing.T) {
	doc := "kind: definition\nprocedure: deploy\ntargets: [local]\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckProcedure_NameMismatch(t *testing.T) {
	doc := "kind: procedure\nprocedure: not-the-filename\ntargets: [local]\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeNameMismatch)
	if p.Field != "procedure" {
		t.Errorf("Field = %q, want procedure", p.Field)
	}
}

func TestCheckProcedure_CadenceIsAStringAndUnvalidated(t *testing.T) {
	doc := "kind: procedure\nprocedure: deploy\ntargets: [local]\ncadence: \"whatever hyper likes\"\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems — cadence:'s grammar is not validated this milestone", got)
	}
}

func TestCheckProcedure_UnknownTopLevelKeyIsUnknownKey(t *testing.T) {
	doc := "kind: procedure\nprocedure: deploy\ntargets: [local]\nsteps: []\nowner: someone\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "owner" {
		t.Errorf("Field = %q, want owner", p.Field)
	}
}

func TestCheckProcedure_StepCarryingBothProcedureAndDefinitionIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: bad
    procedure: deploy
    definition: uptime
    operation: read
    target: local
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

func TestCheckProcedure_NestedInvocationAdmitsOnlyIDAndProcedure(t *testing.T) {
	doc := `kind: procedure
procedure: outer
targets: [local]
steps:
  - id: inner
    procedure: deploy
    bound: 3
`
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"deploy": true})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "steps[0].bound" {
		t.Errorf("Field = %q, want steps[0].bound", p.Field)
	}
}

func TestCheckProcedure_NestedInvocationCleanWhereProcedureResolves(t *testing.T) {
	doc := `kind: procedure
procedure: outer
targets: [local]
steps:
  - id: inner
    procedure: deploy
`
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{"deploy": true})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_NestedInvocationProcedureArtefactAbsent(t *testing.T) {
	doc := `kind: procedure
procedure: outer
targets: [local]
steps:
  - id: inner
    procedure: nowhere
`
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeArtefactAbsent)
	if p.Field != "steps[0].procedure" {
		t.Errorf("Field = %q, want steps[0].procedure", p.Field)
	}
}

func TestCheckProcedure_StepDefinitionArtefactAbsent(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: nowhere
    operation: read
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeArtefactAbsent)
	if p.Field != "steps[0].definition" {
		t.Errorf("Field = %q, want steps[0].definition", p.Field)
	}
}

func TestCheckProcedure_StepOperationReferenceUnresolvable(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: nonexistent
    target: local
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[0].operation" {
		t.Errorf("Field = %q, want steps[0].operation", p.Field)
	}
}

func TestCheckProcedure_ArgsMissingRequiredInputIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0].args.content" {
		t.Errorf("Field = %q, want steps[0].args.content", p.Field)
	}
}

func TestCheckProcedure_ArgsEnumViolationIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: MX
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0].args.type" {
		t.Errorf("Field = %q, want steps[0].args.type", p.Field)
	}
}

const noInputProvider = `kind: provider
provider: noop
schema-version: 1
class: cloudflare
capabilities: [http]
operations:
  ping:
    kind: read
    deadline: 1h
    http:
      method: GET
      host: "{from-target}"
      path: /ping
    record:
      identity: $.status
      fields:
        status: $.status
`

func TestCheckProcedure_ArgsWhereOperationDeclaresNoInputIsUnknownKey(t *testing.T) {
	providers := BuildProviderIndex([]*yaml.Node{parse(t, noInputProvider)})
	definitions := DefinitionIndex{"noop-def": DefinitionInfo{
		ProviderName: "noop",
		Kinds:        map[string]bool{"read": true},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: noop-def
    operation: ping
    target: local
    args:
      foo: bar
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), providers, definitions, localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "steps[0].args.foo" {
		t.Errorf("Field = %q, want steps[0].args.foo", p.Field)
	}
}

func TestCheckProcedure_ReferenceThirdFormIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {foo: bar}
      type: A
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0].args.name" {
		t.Errorf("Field = %q, want steps[0].args.name", p.Field)
	}
}

func TestCheckProcedure_ReferenceStepHalfUnresolvable(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {step: nowhere, path: $.id}
      type: A
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[0].args.name.step" {
		t.Errorf("Field = %q, want steps[0].args.name.step", p.Field)
	}
}

func TestCheckProcedure_ReferenceCannotNameItsOwnStep(t *testing.T) {
	// A Step's id: is registered only once its own args: have been checked,
	// so a reference inside a Step's args: naming that same Step's own id:
	// is not "earlier" and is reference-unresolvable, on the same rule as
	// a reference naming an id: no Step ever declares.
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {step: publish, path: $.id}
      type: A
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[0].args.name.step" {
		t.Errorf("Field = %q, want steps[0].args.name.step", p.Field)
	}
}

func TestCheckProcedure_ReferencePathUnresolvable(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
      content: 203.0.113.10
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {step: publish, path: $.bogus}
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "steps[1].args.record_id.path" {
		t.Errorf("Field = %q, want steps[1].args.record_id.path", p.Field)
	}
}

func TestCheckProcedure_ReferenceResolvesAgainstEarlierRecordField(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
      content: 203.0.113.10
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values: [preview-42.example.com]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {step: publish, path: $.id}
    bound: 1
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_SeriesReference(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: list
    definition: preview-dns
    operation: list_dns_records
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {step: list, path: $.id}
      type: A
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeSeriesReference)
	if p.Field != "steps[1].args.name" {
		t.Errorf("Field = %q, want steps[1].args.name", p.Field)
	}
}

func TestCheckProcedure_ReferenceWhereArrayExpectedIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: earlier
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: {step: earlier, path: $.exit_code}
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[1].args.command" {
		t.Errorf("Field = %q, want steps[1].args.command", p.Field)
	}
}

func TestCheckProcedure_ShellCommandEmptyIsCommandMalformed(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: []
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeCommandMalformed)
	if p.Field != "steps[0].args.command" {
		t.Errorf("Field = %q, want steps[0].args.command", p.Field)
	}
}

func TestCheckProcedure_ShellCommandWrongShapeIsCommandMalformedNotReportedEmpty(t *testing.T) {
	// command: written as a bare scalar is neither an argv list nor a
	// reference to one — command-malformed on the same code as an empty
	// list, but a different message: it is not empty, it is the wrong shape.
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: uptime
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeCommandMalformed)
	if p.Field != "steps[0].args.command" {
		t.Errorf("Field = %q, want steps[0].args.command", p.Field)
	}
	if p.Message == "a shell Step's command: is empty — there is no executable to name" {
		t.Errorf("Message = %q, want a message that does not call a non-empty value empty", p.Message)
	}
}

func TestCheckProcedure_ShellCommandFirstMemberReferenceIsCommandMalformed(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: earlier
    definition: uptime
    operation: read
    target: local
    args:
      command: [uptime]
  - id: probe
    definition: uptime
    operation: read
    target: local
    args:
      command: [{step: earlier, path: $.exit_code}, uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeCommandMalformed)
	if p.Field != "steps[1].args.command[0]" {
		t.Errorf("Field = %q, want steps[1].args.command[0]", p.Field)
	}
	for _, prob := range got {
		if prob.ErrorCode == CodeHoleIllegal {
			t.Fatalf("got %+v, want no hole-illegal — command-malformed is its own code", got)
		}
	}
}

func TestCheckProcedure_ShellCommandLaterMemberReferenceDrawsNoCode(t *testing.T) {
	doc := `kind: procedure
procedure: disk-free
targets: [local]
steps:
  - id: disk-free
    definition: uptime
    operation: mutate
    target: local
    over:
      values: [web-01, web-02, db-01]
    args:
      command: [ssh, {item: $}, df, -h, /srv]
`
	got := CheckProcedure("procedures/disk-free.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_OverValuesDuplicateCaseInsensitiveIsRecordIdentityCollision(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: mutate
    target: local
    over:
      values: [web-01, Web-01]
    args:
      command: [ssh, {item: $}, df, -h, /srv]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeRecordIdentityCollision)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
}

func TestCheckProcedure_OverValuesDistinctMembersAreClean(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: mutate
    target: local
    over:
      values: [web-01, web-02]
    args:
      command: [ssh, {item: $}, df, -h, /srv]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// retirePreviewDNS and retirePreviewDNSLegacy are §3's worked Procedure,
// byte for byte, and its retire-legacy variant (issue #94's acceptance
// criteria: "§3's worked Procedure retire-preview-dns checks clean,
// including its retire-legacy variant").
const retirePreviewDNS = `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
cadence: "0 3 * * 1"
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
      content: 203.0.113.10

  - id: publish-aliases
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    over:
      values:
        - docs.preview-42.example.com
        - api.preview-42.example.com
        - cdn.preview-42.example.com
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {item: $}
      type: CNAME
      content: preview-42.example.com
    bound: 3

  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      assets:
        - field: name
          starts_with: preview-
        - field: created_on
          older_than: 14d
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $.id}
    bound: 5
`

const retirePreviewDNSLegacy = `kind: procedure
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
    bound: 3
`

func TestCheckProcedure_RetirePreviewDNSIsClean(t *testing.T) {
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, retirePreviewDNS), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_RetirePreviewDNSLegacyVariantIsClean(t *testing.T) {
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, retirePreviewDNSLegacy), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// --- issue #95: the two keys, the Bound, and the opaque destroy opt-ins ---

func TestCheckProcedure_KindNotGrantedWhereDefinitionClaimsNothing(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{},
		Destroy:      map[string]bool{},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
	got := CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), definitions, localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeKindNotGranted)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

// TestCheckProcedure_DefinitionClaimingNothingDrawsKindNotGrantedOnEveryStep
// proves issue #95's acceptance criterion: "A Definition claiming no Kinds
// and no destroy: Operations loads clean, and every Step through it draws
// kind-not-granted."
func TestCheckProcedure_DefinitionClaimingNothingDrawsKindNotGrantedOnEveryStep(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{},
		Destroy:      map[string]bool{},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
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
  - id: change
    definition: uptime
    operation: mutate
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), definitions, localTargets(), ProcedureIndex{})
	var kindNotGranted int
	for _, p := range got {
		if p.ErrorCode == CodeKindNotGranted {
			kindNotGranted++
		}
	}
	if kindNotGranted != 2 {
		t.Errorf("kind-not-granted rows = %d, want 2 — a Definition claiming nothing grants no Step through it", kindNotGranted)
	}
}

func TestCheckProcedure_KindNotGrantedWhereTargetDoesNotAcceptIt(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Targets:      map[string]TargetInfo{"local": {Kinds: map[string]bool{"read": true}}},
	}}
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: mutate
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), definitions, TargetIndex{"local": {Kinds: map[string]bool{"read": true}}}, ProcedureIndex{})
	mustCode(t, got, CodeKindNotGranted)
}

func TestCheckProcedure_KindGrantedDrawsNoCode(t *testing.T) {
	mustNoCode(t, CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{}), CodeKindNotGranted)
}

// TestCheckProcedure_MutateOperationNeedsNoNamedClaimOnlyKind proves issue
// #95's acceptance criterion: "A read or mutate Step checks at Kind level
// and needs no named-Operation claim" — mutate_once names no member of any
// claim by name, and the Kind claim alone is enough.
func TestCheckProcedure_MutateOperationNeedsNoNamedClaimOnlyKind(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: mutate_once
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems — mutate checks at Kind level and needs no per-Operation claim", got)
	}
}

func TestCheckProcedure_DestroyOperationNotClaimedIsOperationNotClaimed(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Destroy:      map[string]bool{"destroy_once": true},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/tmp/a]
    args:
      command: [rm, -rf, {item: $}]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), definitions, localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeOperationNotClaimed)
	if p.Field != "steps[0].operation" {
		t.Errorf("Field = %q, want steps[0].operation", p.Field)
	}
}

func TestCheckProcedure_DestroyOperationClaimedDrawsNoOperationNotClaimed(t *testing.T) {
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/tmp/a]
    args:
      command: [rm, -rf, {item: $}]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	mustNoCode(t, got, CodeOperationNotClaimed)
}

// TestCheckProcedure_TargetNotClaimedIsItsOwnCodeNotOperationNotClaimed
// proves issue #95's acceptance criterion: target-not-claimed is its own
// code and is never folded into operation-not-claimed — a reader handed
// that code on a target: line would go looking at destroy:, which is the
// wrong edit. The Definition here claims the destroy: Operation by name and
// no Target at all, so operation-not-claimed has nothing left to fire on.
func TestCheckProcedure_TargetNotClaimedIsItsOwnCodeNotOperationNotClaimed(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{},
		Destroy:      map[string]bool{"destroy": true},
		Targets:      map[string]TargetInfo{},
	}}
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/tmp/a]
    args:
      command: [rm, -rf, {item: $}]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), definitions, localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeTargetNotClaimed)
	if p.Field != "steps[0].target" {
		t.Errorf("Field = %q, want steps[0].target", p.Field)
	}
	for _, prob := range got {
		if prob.ErrorCode == CodeOperationNotClaimed {
			t.Errorf("got operation-not-claimed for a target: line, want only target-not-claimed — the wrong edit")
		}
	}
}

func TestCheckProcedure_DestroyWithNoBoundIsBoundMissing(t *testing.T) {
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: some-record-id
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeBoundMissing)
	if p.Field != "steps[0].bound" {
		t.Errorf("Field = %q, want steps[0].bound", p.Field)
	}
}

func TestCheckProcedure_OpaqueDestroyWithBoundIsBoundIllegal(t *testing.T) {
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/srv/app/releases/r41]
    bound: 1
    args:
      command: [rm, -rf, {item: $}]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeBoundIllegal)
	if p.Field != "steps[0].bound" {
		t.Errorf("Field = %q, want steps[0].bound", p.Field)
	}
}

// TestCheckProcedure_OpaqueDestroyWithNoBoundIsClean proves the worked
// example §5 states, byte for byte for its Steps' shape: two Tombstones,
// each opening the series it ends, and no Bound anywhere in sight — the one
// combination the Bound rule and its opaque exception both agree on.
func TestCheckProcedure_OpaqueDestroyWithNoBoundIsClean(t *testing.T) {
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge-releases
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/srv/app/releases/r41, /srv/app/releases/r42]
    args:
      command: [rm, -rf, {item: $}]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems — an opaque destroy Step carries no Bound and this one carries none", got)
	}
}

func TestCheckProcedure_ReadWithBoundIsUnknownKey(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: local
    bound: 1
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "steps[0].bound" {
		t.Errorf("Field = %q, want steps[0].bound", p.Field)
	}
}

func TestCheckProcedure_OpaqueDestroyWithNoOverIsUnscoped(t *testing.T) {
	doc := `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    args:
      command: [rm, -rf, /srv/app/releases/r41]
`
	got := CheckProcedure("procedures/cleanup.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeDestroyUnscoped)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

func TestCheckProcedure_OpaqueDestroyWithOverIsScoped(t *testing.T) {
	mustNoCode(t, CheckProcedure("procedures/cleanup.yaml", parse(t, `kind: procedure
procedure: cleanup
targets: [local]
steps:
  - id: purge
    definition: uptime
    operation: destroy
    target: local
    over:
      values: [/srv/app/releases/r41]
    args:
      command: [rm, -rf, {item: $}]
`), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{}), CodeDestroyUnscoped)
}

// TestCheckProcedure_NonOpaqueDestroyWithNoOverIsUnscoped is issue #157's
// reproduction, refused where §5's own argument holds rather than where it was
// noticed: an `http` `destroy` carrying a mandatory Bound and no `over:` at
// all. It is invoked once, so its one member has no name, and what it would
// conclude about is that member — an identity the Store cannot hold. Every
// clause of the requirement is true here and none of them is about opacity
// (§4, §5, ADR-0085).
func TestCheckProcedure_NonOpaqueDestroyWithNoOverIsUnscoped(t *testing.T) {
	doc := `kind: procedure
procedure: purge-unscoped
targets: [cloudflare-prod]
steps:
  - id: purge
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    bound: 1
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: 372e67954025e0ba6aaa6d586b9e0b59
`
	got := CheckProcedure("procedures/purge-unscoped.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, CodeDestroyUnscoped)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
	for _, prob := range got {
		if prob.ErrorCode == CodeBoundMissing || prob.ErrorCode == CodeBoundIllegal {
			t.Errorf("got %s beside destroy-unscoped, want the Bound left alone — this Step's bound: is correct", prob.ErrorCode)
		}
	}
}

// TestCheckProcedure_NonOpaqueDestroyWithOverIsScoped is the widened check's
// other side: the ordinary scoped `destroy` every landed corpus already holds
// draws nothing new.
func TestCheckProcedure_NonOpaqueDestroyWithOverIsScoped(t *testing.T) {
	mustNoCode(t, CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      assets:
        - field: name
          starts_with: preview-
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $.id}
    bound: 5
`), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{}), CodeDestroyUnscoped)
}

// TestCheckProcedure_MutateWithNoOverIsClean holds the widening to the Kind it
// is stated on. A `mutate` invoked once mints a Record under an identity its
// Operation declares (§3, ADR-0037), so it has a name and needs no selector to
// find one; only a `destroy`, which declares no identity at all, is left with
// nothing.
func TestCheckProcedure_MutateWithNoOverIsClean(t *testing.T) {
	mustNoCode(t, CheckProcedure("procedures/publish.yaml", parse(t, `kind: procedure
procedure: publish
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
      content: 203.0.113.10
`), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{}), CodeDestroyUnscoped)
}

// TestCheckProcedure_EmptyValuesMemberIsSchemaMismatch closes the other
// direction on the same Store guard (issue #157). A `values:` member that is an
// empty scalar carries the Expansion a member whose Name is "", and a `destroy`
// concludes about the member — so the head lookup that decides whether it is
// already gone asks the Store for a series under an identity with no name,
// which is `store.seriesDir`'s own `impossible`. It is refused on the page
// instead: an empty scalar names nothing a Record can be held under, which is
// §4's schema-mismatch where the value is authored (§4, §7, ADR-0085).
func TestCheckProcedure_EmptyValuesMemberIsSchemaMismatch(t *testing.T) {
	doc := `kind: procedure
procedure: purge-empty
targets: [cloudflare-prod]
steps:
  - id: purge
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    bound: 1
    over:
      values: [""]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
`
	got := CheckProcedure("procedures/purge-empty.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
}

// TestCheckProcedure_EmptyValuesMemberIsRefusedOnEveryKind holds the check to
// the page rather than to the Kind. A `mutate` never reaches the head lookup
// above — a Tombstoned series is one a `mutate` is asking for, so it drops
// nothing (§5) — but an empty member is not an identifier on any Kind, and a
// check that fired only on `destroy` would leave the same authored nonsense
// legal one key over.
func TestCheckProcedure_EmptyValuesMemberIsRefusedOnEveryKind(t *testing.T) {
	doc := `kind: procedure
procedure: publish-empty
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    over:
      values: ["", other.example.com]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {item: $}
      type: A
      content: 203.0.113.10
`
	got := CheckProcedure("procedures/publish-empty.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
}

// TestCheckProcedure_ValuesMembersThatNameSomethingAreClean is the check's
// other side: an ordinary list draws nothing.
func TestCheckProcedure_ValuesMembersThatNameSomethingAreClean(t *testing.T) {
	mustNoCode(t, CheckProcedure("procedures/purge-two.yaml", parse(t, `kind: procedure
procedure: purge-two
targets: [cloudflare-prod]
steps:
  - id: purge
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    bound: 2
    over:
      values: [preview-41.example.com, preview-42.example.com]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
`), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{}), schema.CodeMismatch)
}

// --- issue #96: the transitive walks — the envelope and the two Cadence rules ---

// TestCheckProcedure_StepTargetOutsideDeclaredTargetsIsEnvelopeExceeded
// proves the file-local half of the envelope rule: a Step whose bound
// Target the Procedure's own targets: never named is envelope-exceeded,
// whether or not the binding is otherwise authorised.
func TestCheckProcedure_StepTargetOutsideDeclaredTargetsIsEnvelopeExceeded(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Destroy:      map[string]bool{"destroy": true, "destroy_once": true},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget(), "other": fullyGrantedTarget()},
	}}
	targets := TargetIndex{"local": fullyGrantedTarget(), "other": fullyGrantedTarget()}
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: other
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), definitions, targets, ProcedureIndex{})
	p := mustCode(t, got, CodeEnvelopeExceeded)
	if p.Field != "steps[0].target" {
		t.Errorf("Field = %q, want steps[0].target", p.Field)
	}
}

// TestCheckProcedure_StepTargetInsideDeclaredTargetsDrawsNoEnvelopeExceeded
// proves the clean side of the same rule: a Step bound to a Target the
// Procedure's own targets: does name draws no envelope-exceeded.
func TestCheckProcedure_StepTargetInsideDeclaredTargetsDrawsNoEnvelopeExceeded(t *testing.T) {
	got := CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	mustNoCode(t, got, CodeEnvelopeExceeded)
}

// TestCheckProcedure_StepKindOutsideDeclaredKindEnvelopeIsEnvelopeExceeded
// proves the Kind half of the same file-local rule: a Step whose Operation's
// Kind no declared target's own accepted Kinds grant is envelope-exceeded —
// read here off a Target the Procedure's targets: does name, so the Target
// envelope half of the rule stays clean and only the Kind half fires.
func TestCheckProcedure_StepKindOutsideDeclaredKindEnvelopeIsEnvelopeExceeded(t *testing.T) {
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Targets:      map[string]TargetInfo{"local": fullyGrantedTarget()},
	}}
	targets := TargetIndex{"local": {Kinds: map[string]bool{"read": true}}}
	doc := `kind: procedure
procedure: deploy
targets: [local]
steps:
  - id: probe
    definition: uptime
    operation: mutate
    target: local
    args:
      command: [uptime]
`
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), definitions, targets, ProcedureIndex{})
	p := mustCode(t, got, CodeEnvelopeExceeded)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

// TestCheckProcedure_UnresolvedDeclaredTargetContributesNoKind proves an
// unresolved targets: member contributes no accepted Kinds to the Kind
// envelope — it names no Target declaration for a union to read, the same
// way an unresolved name is read absent everywhere else in this package.
func TestCheckProcedure_UnresolvedDeclaredTargetContributesNoKind(t *testing.T) {
	doc := `kind: procedure
procedure: deploy
targets: [nowhere]
steps:
  - id: probe
    definition: uptime
    operation: read
    target: nowhere
    args:
      command: [uptime]
`
	definitions := DefinitionIndex{"uptime": DefinitionInfo{
		ProviderName: "shell",
		Kinds:        map[string]bool{"read": true},
		Targets:      map[string]TargetInfo{"nowhere": fullyGrantedTarget()},
	}}
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), definitions, TargetIndex{}, ProcedureIndex{})
	p := mustCode(t, got, CodeEnvelopeExceeded)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}
