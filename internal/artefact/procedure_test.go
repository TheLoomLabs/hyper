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

func uptimeDefinitions() DefinitionIndex {
	return DefinitionIndex{"uptime": "shell"}
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
	return DefinitionIndex{"preview-dns": "cloudflare-dns"}
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, minimalProcedure), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_CommentsAreIgnored(t *testing.T) {
	doc := "# a comment above the document\n" + minimalProcedure + "    # a comment beside a Step's own lines\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_KindMismatch(t *testing.T) {
	doc := "kind: definition\nprocedure: deploy\ntargets: [local]\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckProcedure_NameMismatch(t *testing.T) {
	doc := "kind: procedure\nprocedure: not-the-filename\ntargets: [local]\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
	p := mustCode(t, got, CodeNameMismatch)
	if p.Field != "procedure" {
		t.Errorf("Field = %q, want procedure", p.Field)
	}
}

func TestCheckProcedure_CadenceIsAStringAndUnvalidated(t *testing.T) {
	doc := "kind: procedure\nprocedure: deploy\ntargets: [local]\ncadence: \"whatever hyper likes\"\nsteps: []\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems — cadence:'s grammar is not validated this milestone", got)
	}
}

func TestCheckProcedure_UnknownTopLevelKeyIsUnknownKey(t *testing.T) {
	doc := "kind: procedure\nprocedure: deploy\ntargets: [local]\nsteps: []\nowner: someone\n"
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{"deploy": true})
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
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{"deploy": true})
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
	got := CheckProcedure("procedures/outer.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	definitions := DefinitionIndex{"noop-def": "noop"}
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), providers, definitions, ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {step: publish, path: $.id}
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/disk-free.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/deploy.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), ProcedureIndex{})
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
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, retirePreviewDNS), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_RetirePreviewDNSLegacyVariantIsClean(t *testing.T) {
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, retirePreviewDNSLegacy), cloudflareProcedureProviders(t), previewDNSDefinitions(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}
