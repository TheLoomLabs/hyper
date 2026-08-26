// This file tests issue #98: the host grant — an Operation's host:
// template expanded at load into a finite candidate set and compared
// against the bound Target's hosts: grant (host-not-granted, one code over
// the template's candidates and a values: list wired into host-input:,
// read off the wiring and never off a declaration), the intersection
// deciding whether host-input: is required (manifest-inconsistent where it
// holds several and none is declared), and the one Expansion-identity
// fault decidable with no Store: an over: values: list of two or more
// whose members can only ever project one Record identity
// (record-identity-collision, the same code the load site fires for a
// duplicate values: member, found here against the wiring rather than
// against the list).
package artefact

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// hostco is the Manifest fixture this file's host-grant tests resolve
// against: two enumeration-hole host: templates — one single-hole, one
// carrying two holes so the cross-product is exercised — each naming
// host-input: endpoint, and one {from-target} Operation declaring no
// host-input:, the shape the intersection rule refuses wherever the grant
// holds several hosts. create_widget's identity: is a template hole, so it
// doubles as this file's resolves-before-the-call fixture for the
// Expansion-identity tests.
const hostco = `kind: provider
provider: hostco
schema-version: 1
class: hostco
capabilities: [http]
enumerations:
  region: [us-east-1, eu-central-1]
  tier: [prod, stage]
operations:
  list_buckets:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "s3.{region}.amazonaws.com"
      path: /
      host-input: endpoint
    input:
      type: object
      properties:
        endpoint: {type: string}
    record:
      identity: $.host
      fields: {host: $.host}
  probe:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "{region}.{tier}.hostco.dev"
      path: /
      host-input: endpoint
    input:
      type: object
      properties:
        endpoint: {type: string}
    record:
      identity: $.host
      fields: {host: $.host}
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
      fields: {name: $.body.name}
  rename_widget:
    kind: mutate
    repeatability: repeatable
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /widgets/{name}
      body: {rename_to: "{rename_to}"}
    input:
      type: object
      properties:
        name: {type: string}
        rename_to: {type: string}
    record:
      identity: "{name}"
      fields: {name: $.body.name}
`

func hostcoProviders(t *testing.T) ProviderIndex {
	t.Helper()
	return BuildProviderIndex([]*yaml.Node{parse(t, hostco)})
}

// hostcoTarget builds the TargetInfo a test's own grant needs — every Kind
// accepted, the authority checks not being what this file exercises, and
// hosts the granted host set the grant checks read.
func hostcoTarget(hosts ...string) TargetInfo {
	granted := map[string]bool{}
	for _, h := range hosts {
		granted[h] = true
	}
	return TargetInfo{
		Kinds: map[string]bool{"read": true, "mutate": true, "destroy": true},
		Hosts: granted,
	}
}

// hostcoDefinitions pairs the hostco Provider with whichever Targets a
// test's Steps bind, each granting its own hosts: set.
func hostcoDefinitions(targets map[string]TargetInfo) DefinitionIndex {
	return DefinitionIndex{"hostco": DefinitionInfo{
		ProviderName: "hostco",
		Kinds:        map[string]bool{"read": true, "mutate": true},
		Targets:      targets,
	}}
}

// hostcoStep builds a one-Step Procedure binding hostco's operation against
// target, with the args and over: blocks a test supplies — the Step being
// the binding site every host-grant and Expansion-identity check reads.
func hostcoStep(operation, target, args, over string) string {
	doc := `kind: procedure
procedure: hostco-run
targets: [` + target + `]
steps:
  - id: step
    definition: hostco
    operation: ` + operation + `
    target: ` + target + `
`
	if args != "" {
		doc += "    args:\n" + args
	}
	if over != "" {
		doc += "    over:\n" + over
	}
	return doc
}

func checkHostcoStep(t *testing.T, operation, target, args, over string, targets map[string]TargetInfo) []problem.Problem {
	t.Helper()
	doc := hostcoStep(operation, target, args, over)
	return CheckProcedure("procedures/hostco-run.yaml", parse(t, doc), hostcoProviders(t), hostcoDefinitions(targets), TargetIndex(targets), ProcedureIndex{})
}

func TestCheckManifest_HostcoIsClean(t *testing.T) {
	mustNone(t, checkManifest(t, "providers/hostco.yaml", hostco))
}

// --- The candidate set ---

func TestBuildTargetIndex_ReadsTheGrantedHostSet(t *testing.T) {
	idx := BuildTargetIndex([]*yaml.Node{parse(t, cloudflareProd)})
	if !idx["cloudflare-prod"].Hosts["api.cloudflare.com"] {
		t.Errorf("BuildTargetIndex() Hosts = %v, want api.cloudflare.com granted", idx["cloudflare-prod"].Hosts)
	}
}

func TestCheckProcedure_HostTemplateFromTargetExpandsToTheGrantItself(t *testing.T) {
	// {from-target} expands to the bound Target's granted host set, so the
	// candidate set is the grant and nothing can be absent from it
	// (ADR-0029).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one", "      name: widget-one\n", "", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_HostTemplateCandidateAbsentFromGrant(t *testing.T) {
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("s3.us-east-1.amazonaws.com")}
	got := checkHostcoStep(t, "list_buckets", "hostco-one", "      endpoint: s3.us-east-1.amazonaws.com\n", "", targets)
	p := mustCode(t, got, CodeHostNotGranted)
	if !strings.Contains(p.Message, "s3.eu-central-1.amazonaws.com") {
		t.Errorf("Message = %q, want it to name the absent candidate s3.eu-central-1.amazonaws.com", p.Message)
	}
}

func TestCheckProcedure_HostTemplateCrossProductExpandsEveryHole(t *testing.T) {
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("us-east-1.prod.hostco.dev", "eu-central-1.prod.hostco.dev")}
	got := checkHostcoStep(t, "probe", "hostco-one", "      endpoint: us-east-1.prod.hostco.dev\n", "", targets)
	absent := []string{"us-east-1.stage.hostco.dev", "eu-central-1.stage.hostco.dev"}
	count := 0
	for _, p := range got {
		if p.ErrorCode != CodeHostNotGranted {
			continue
		}
		count++
		found := false
		for _, want := range absent {
			if strings.Contains(p.Message, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("host-not-granted Message = %q, want it to name one of %v", p.Message, absent)
		}
	}
	if count != 2 {
		t.Fatalf("CheckProcedure() = %+v, want exactly 2 host-not-granted rows, one per absent cross-product member", got)
	}
}

// --- The intersection ---

func TestCheckProcedure_IntersectionOfSeveralAndNoHostInputIsManifestInconsistent(t *testing.T) {
	// {from-target} expands to the grant itself, so a two-host grant is a
	// two-member candidate set and a two-member intersection — and an
	// Operation declaring no host-input: leaves which of them a request
	// reaches undecidable (§3, ADR-0029).
	targets := map[string]TargetInfo{"hostco-multi": hostcoTarget("a.hostco.dev", "b.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-multi", "      name: widget-one\n", "", targets)
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "steps[0]" {
		t.Errorf("Field = %q, want steps[0]", p.Field)
	}
}

func TestCheckProcedure_IntersectionOfOneIsFilledAndNeedsNoHostInput(t *testing.T) {
	// Where the candidate set and the grant intersect to exactly one host
	// hyper fills it, and host-input: is not required (§3).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("only.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one", "      name: widget-one\n", "", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_IntersectionOfSeveralWithHostInputIsClean(t *testing.T) {
	// The same several-member intersection with host-input: declared is
	// the shape host-input: exists for — the value it carries is checked
	// for membership when it arrives, which is milestone 5's (§3, §4).
	targets := map[string]TargetInfo{"hostco-both": hostcoTarget("s3.us-east-1.amazonaws.com", "s3.eu-central-1.amazonaws.com")}
	got := checkHostcoStep(t, "list_buckets", "hostco-both", "      endpoint: s3.us-east-1.amazonaws.com\n", "", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// --- The values: host list ---

func TestCheckProcedure_ValuesWiredIntoHostInputAreCheckedAgainstTheGrant(t *testing.T) {
	// {item: $} in the Operation's host-input: makes the values: list a
	// host list, compared against the same grant under the same code as
	// the template's candidate set (§3, §4, ADR-0024). The grant covers
	// both of the template's own candidates, so the only comparison that
	// can fire here is the list's own.
	targets := map[string]TargetInfo{"hostco-both": hostcoTarget("s3.us-east-1.amazonaws.com", "s3.eu-central-1.amazonaws.com")}
	got := checkHostcoStep(t, "list_buckets", "hostco-both",
		"      endpoint: {item: $}\n",
		"      values: [s3.us-east-1.amazonaws.com, s3.eu-central-2.amazonaws.com]\n", targets)
	p := mustCode(t, got, CodeHostNotGranted)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
	if !strings.Contains(p.Message, "s3.eu-central-2.amazonaws.com") {
		t.Errorf("Message = %q, want it to name the absent member s3.eu-central-2.amazonaws.com", p.Message)
	}
}

func TestCheckProcedure_ValuesWiredIntoHostInputAllGrantedIsClean(t *testing.T) {
	targets := map[string]TargetInfo{"hostco-both": hostcoTarget("s3.us-east-1.amazonaws.com", "s3.eu-central-1.amazonaws.com")}
	got := checkHostcoStep(t, "list_buckets", "hostco-both",
		"      endpoint: {item: $}\n",
		"      values: [s3.us-east-1.amazonaws.com, s3.eu-central-1.amazonaws.com]\n", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_ValuesWiredIntoAnyOtherInputAreIdentifiers(t *testing.T) {
	// A values: member wired into any other input is an identifier,
	// reaches nothing on its own, and is compared against no grant — the
	// wiring decides, never a declaration (§3, §12). {item: $} in name:
	// is also the Expansion-identity remedy, so no collision code either.
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one",
		"      name: {item: $}\n",
		"      values: [web-01.internal, web-02.internal]\n", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_ShellStepValuesAreComparedAgainstNoGrant(t *testing.T) {
	// §3's own worked disk-free Step: a shell Operation has no
	// host-input:, so a values: list on a shell Step is never compared
	// against a grant — the one Capability whose reach no grant bounds
	// (§3, §13).
	doc := `kind: procedure
procedure: disk-free
targets: [local]
steps:
  - id: disk-free
    definition: uptime
    operation: read
    target: local
    over:
      values: [web-01, web-02]
    args:
      command: [ssh, {item: $}, df, -h, /srv]
`
	got := CheckProcedure("procedures/disk-free.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

// --- An Expansion with one identity in it ---

func TestCheckProcedure_ValuesExpansionWithOneIdentityIsRecordIdentityCollision(t *testing.T) {
	// identity: "{name}" resolves before the call, and name: is a literal
	// — one value for the whole Expansion, so two members are two calls
	// writing two versions of one series (§4).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one",
		"      name: widget-one\n",
		"      values: [widget-a, widget-b]\n", targets)
	p := mustCode(t, got, CodeRecordIdentityCollision)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
}

func TestCheckProcedure_ValuesExpansionIdentityFromAStepReferenceIsStillOneIdentity(t *testing.T) {
	// A reference to another Step's output is one value for the whole
	// Expansion by construction, exactly as a literal is (§4).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	doc := `kind: procedure
procedure: hostco-run
targets: [hostco-one]
steps:
  - id: first
    definition: hostco
    operation: create_widget
    target: hostco-one
    args:
      name: widget-one
  - id: again
    definition: hostco
    operation: create_widget
    target: hostco-one
    over:
      values: [widget-a, widget-b]
    args:
      name: {step: first, path: $.name}
`
	got := CheckProcedure("procedures/hostco-run.yaml", parse(t, doc), hostcoProviders(t), hostcoDefinitions(targets), TargetIndex(targets), ProcedureIndex{})
	p := mustCode(t, got, CodeRecordIdentityCollision)
	if p.Field != "steps[1].over.values" {
		t.Errorf("Field = %q, want steps[1].over.values", p.Field)
	}
}

func TestCheckProcedure_ValuesExpansionWithMemberWiredIntoTheIdentityIsClean(t *testing.T) {
	// The remedy: wire the member into the input the identity reads, and
	// distinct members resolve to distinct identities (§4).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one",
		"      name: {item: $}\n",
		"      values: [widget-a, widget-b]\n", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_OneMemberValuesExpansionHasNoSiblingToCollideWith(t *testing.T) {
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "create_widget", "hostco-one",
		"      name: widget-one\n",
		"      values: [widget-a]\n", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_AssetsSelectorSizeIsOnNoFile(t *testing.T) {
	// The member count must be authored for the collision to be decidable
	// offline: an assets: selector's size is on no file, so a literal
	// identity here is §6's to catch at Expansion, not §4's (§4).
	targets := map[string]TargetInfo{"hostco-one": hostcoTarget("one.hostco.dev")}
	got := checkHostcoStep(t, "rename_widget", "hostco-one",
		"      name: widget-one\n      rename_to: widget-two\n",
		"      assets:\n        - field: name\n          starts_with: widget-\n", targets)
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_ResponseIdentityDrawsNoCollisionCode(t *testing.T) {
	// list_dns_records' identity: is $.id — a response path, nowhere
	// earlier than the response to read it off — so a two-member values:
	// list under a literal zone_id draws no code here (§4).
	definitions := DefinitionIndex{"preview-dns-observed": DefinitionInfo{
		ProviderName: "cloudflare-dns",
		Kinds:        map[string]bool{"read": true},
		Targets:      map[string]TargetInfo{"cloudflare-prod": fullyGrantedTarget()},
	}}
	doc := `kind: procedure
procedure: list-preview-dns
targets: [cloudflare-prod]
steps:
  - id: list
    definition: preview-dns-observed
    operation: list_dns_records
    target: cloudflare-prod
    over:
      values: [zone-a, zone-b]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
`
	got := CheckProcedure("procedures/list-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), definitions, cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_DestroyExpansionMemberIsTheName(t *testing.T) {
	// A destroy projects nothing and writes its Tombstone under the
	// Asset's own identity, so the member is the name and distinct
	// members are distinct identities by construction (§3).
	doc := `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values: [record-a, record-b]
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 5
`
	got := CheckProcedure("procedures/retire-preview-dns.yaml", parse(t, doc), cloudflareProcedureProviders(t), previewDNSDefinitions(), cloudflareTargets(t), ProcedureIndex{})
	if len(got) != 0 {
		t.Fatalf("CheckProcedure() = %+v, want no problems", got)
	}
}

func TestCheckProcedure_ShellExpansionWithOneCommandIsRecordIdentityCollision(t *testing.T) {
	// $.command on a shell Operation resolves before the call — it is a
	// fact about the call rather than about the answer — so two members
	// running one literal argv are one identity (§3, §4).
	doc := `kind: procedure
procedure: disk-free
targets: [local]
steps:
  - id: disk-free
    definition: uptime
    operation: read
    target: local
    over:
      values: [web-01, web-02]
    args:
      command: [df, -h, /srv]
`
	got := CheckProcedure("procedures/disk-free.yaml", parse(t, doc), shellProviders(), uptimeDefinitions(), localTargets(), ProcedureIndex{})
	p := mustCode(t, got, CodeRecordIdentityCollision)
	if p.Field != "steps[0].over.values" {
		t.Errorf("Field = %q, want steps[0].over.values", p.Field)
	}
}
