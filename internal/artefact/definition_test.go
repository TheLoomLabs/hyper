package artefact

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// previewDNS and previewDNSObserved are §3's two worked Definitions, byte
// for byte (issue #93's acceptance criteria: "§3's two worked Definitions —
// preview-dns and preview-dns-observed — both check clean").
const previewDNS = `kind: definition
definition: preview-dns
provider: cloudflare-dns
kinds: [mutate]
destroy: [delete_dns_record]
targets: [cloudflare-prod]
`

const previewDNSObserved = `kind: definition
definition: preview-dns-observed
provider: cloudflare-dns
kinds: [read]
targets: [cloudflare-prod]
`

// cloudflareProviders and cloudflareTargets are the namespaces preview-dns
// and preview-dns-observed resolve against — cloudflareDNS and
// cloudflareProd are already §3's and §4's own worked artefacts, byte for
// byte, reused here rather than restated (manifest_test.go, artefact_test.go).
func cloudflareProviders(t *testing.T) ProviderIndex {
	t.Helper()
	return BuildProviderIndex([]*yaml.Node{parse(t, cloudflareDNS)})
}

func cloudflareTargets(t *testing.T) TargetIndex {
	t.Helper()
	return BuildTargetIndex([]*yaml.Node{parse(t, cloudflareProd)})
}

func TestCheckDefinition_PreviewDNSIsClean(t *testing.T) {
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, previewDNS), cloudflareProviders(t), cloudflareTargets(t))
	if len(got) != 0 {
		t.Fatalf("CheckDefinition() = %+v, want no problems", got)
	}
}

func TestCheckDefinition_PreviewDNSObservedIsClean(t *testing.T) {
	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, previewDNSObserved), cloudflareProviders(t), cloudflareTargets(t))
	if len(got) != 0 {
		t.Fatalf("CheckDefinition() = %+v, want no problems", got)
	}
}

func TestCheckDefinition_KindMismatch(t *testing.T) {
	doc := "kind: provider\ndefinition: uptime\nprovider: shell\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, TargetIndex{})
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckDefinition_NameMismatch(t *testing.T) {
	doc := "kind: definition\ndefinition: not-the-filename\nprovider: shell\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, TargetIndex{})
	p := mustCode(t, got, CodeNameMismatch)
	if p.Field != "definition" {
		t.Errorf("Field = %q, want definition", p.Field)
	}
}

func TestCheckDefinition_DestroyIsNotAMemberOfKinds(t *testing.T) {
	doc := "kind: definition\ndefinition: uptime\nprovider: shell\nkinds: [destroy]\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, TargetIndex{})
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "kinds[0]" {
		t.Errorf("Field = %q, want kinds[0]", p.Field)
	}
}

func TestCheckDefinition_UnknownKey(t *testing.T) {
	doc := "kind: definition\ndefinition: uptime\nprovider: shell\ntargets: [local]\nnickname: foo\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, TargetIndex{})
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "nickname" {
		t.Errorf("Field = %q, want nickname", p.Field)
	}
}

func TestCheckDefinition_ProviderArtefactAbsentCarriesThePathLookedFor(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns\nprovider: no-such-provider\nkinds: [read]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	p := mustCode(t, got, CodeArtefactAbsent)
	if p.Field != "provider" {
		t.Errorf("Field = %q, want provider", p.Field)
	}
	if p.Line != 3 {
		t.Errorf("Line = %d, want 3", p.Line)
	}
	if !strings.Contains(p.Message, "providers/no-such-provider.yaml") {
		t.Errorf("Message = %q, want the path hyper looked for", p.Message)
	}
}

func TestCheckDefinition_TargetsMemberArtefactAbsentCarriesThePathLookedFor(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns\nprovider: cloudflare-dns\nkinds: [read]\ntargets: [no-such-target]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	p := mustCode(t, got, CodeArtefactAbsent)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
	if !strings.Contains(p.Message, "targets/no-such-target.yaml") {
		t.Errorf("Message = %q, want the path hyper looked for", p.Message)
	}
}

// TestCheckDefinition_TargetsLocalWithNoDeclarationIsArtefactAbsent proves
// targets: [local] where no targets/local.yaml exists is artefact-absent
// like any other name an author wrote (issue #93's acceptance criterion):
// local's reservation exempts it from local-reserved's two rules, not from
// resolution — an author who names it still has to write the file.
func TestCheckDefinition_TargetsLocalWithNoDeclarationIsArtefactAbsent(t *testing.T) {
	doc := "kind: definition\ndefinition: uptime\nprovider: shell\nkinds: [read]\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, TargetIndex{})
	p := mustCode(t, got, CodeArtefactAbsent)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
	if !strings.Contains(p.Message, "targets/local.yaml") {
		t.Errorf("Message = %q, want the path hyper looked for", p.Message)
	}
}

// TestCheckDefinition_ProviderWithAFaultOfItsOwnStillResolves proves ADR-
// 0064's "a file that will not parse is still present": a Manifest
// carrying an unrelated fault of its own — here, a stray key its own
// schema does not admit — still declares its provider: name, so a
// Definition naming it correctly resolves clean, and the Manifest's fault
// is reported once, on the Manifest's own line, never repeated on the
// Definition's.
func TestCheckDefinition_ProviderWithAFaultOfItsOwnStillResolves(t *testing.T) {
	faultyManifest := cloudflareDNS + "nickname: dns\n"
	manifestProblems := checkManifest(t, "providers/cloudflare-dns.yaml", faultyManifest)
	mustCode(t, manifestProblems, schema.CodeUnknownKey)

	providers := BuildProviderIndex([]*yaml.Node{parse(t, faultyManifest)})
	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, previewDNSObserved), providers, cloudflareTargets(t))
	if len(got) != 0 {
		t.Fatalf("CheckDefinition() = %+v, want no problems — the Manifest's own fault is the Manifest's row, not this one's", got)
	}
}

// TestCheckDefinition_NameResolutionIsByteExactAndCaseSensitive proves
// Preview-DNS does not resolve to preview-dns (issue #93's acceptance
// criterion), on ADR-0060's own rule.
func TestCheckDefinition_NameResolutionIsByteExactAndCaseSensitive(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns\nprovider: Cloudflare-DNS\nkinds: [read]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	mustCode(t, got, CodeArtefactAbsent)
}

// TestCheckDefinition_ExtensionNobodyInstalledIsArtefactAbsent proves a
// provider: naming an Extension nobody installed lands at check, offline,
// under the same code an author's typo earns — never a distinct code of
// install's own (§11, ADR-0060, issue #93's acceptance criterion).
func TestCheckDefinition_ExtensionNobodyInstalledIsArtefactAbsent(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns\nprovider: some-extension-nobody-installed\nkinds: [read]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	if len(got) != 1 {
		t.Fatalf("CheckDefinition() = %+v, want exactly one problem", got)
	}
	if got[0].ErrorCode != CodeArtefactAbsent {
		t.Errorf("ErrorCode = %q, want %s", got[0].ErrorCode, CodeArtefactAbsent)
	}
}

func TestCheckDefinition_DestroyMemberIsReferenceUnresolvableNotArtefactAbsent(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns\nprovider: cloudflare-dns\nkinds: [mutate]\ndestroy: [no_such_operation]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	p := mustCode(t, got, CodeReferenceUnresolvable)
	if p.Field != "destroy[0]" {
		t.Errorf("Field = %q, want destroy[0]", p.Field)
	}
	for _, prob := range got {
		if prob.ErrorCode == CodeArtefactAbsent {
			t.Errorf("got artefact-absent for destroy:, want only reference-unresolvable — the Manifest's own namespace")
		}
	}
}

func TestCheckDefinition_UnresolvedProviderSkipsDestroyResolutionButNotOtherRules(t *testing.T) {
	// A failed resolution does not stop the pass: the missing provider:
	// earns its own row, and the unrelated targets: member still resolves
	// or fails to resolve on its own account (issue #93's acceptance
	// criterion).
	doc := "kind: definition\ndefinition: preview-dns\nprovider: no-such-provider\nkinds: [mutate]\ndestroy: [delete_dns_record]\ntargets: [no-such-target]\n"
	got := CheckDefinition("definitions/preview-dns.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))

	var providerRows, targetRows int
	for _, p := range got {
		if p.ErrorCode != CodeArtefactAbsent {
			continue
		}
		switch p.Field {
		case "provider":
			providerRows++
		case "targets[0]":
			targetRows++
		}
	}
	if providerRows != 1 {
		t.Errorf("provider artefact-absent rows = %d, want 1", providerRows)
	}
	if targetRows != 1 {
		t.Errorf("targets artefact-absent rows = %d, want 1", targetRows)
	}
	for _, p := range got {
		if p.ErrorCode == CodeReferenceUnresolvable {
			t.Errorf("got %+v, want no reference-unresolvable — there is no Provider to resolve destroy: against", p)
		}
	}
}

func TestCheckDefinition_ReadBesideMutateIsKindsMixed(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns-observed\nprovider: cloudflare-dns\nkinds: [read, mutate]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	p := mustCode(t, got, CodeDefinitionKindsMixed)
	if p.Field != "kinds" {
		t.Errorf("Field = %q, want kinds", p.Field)
	}
}

func TestCheckDefinition_ReadBesideDestroyClaimIsKindsMixed(t *testing.T) {
	doc := "kind: definition\ndefinition: preview-dns-observed\nprovider: cloudflare-dns\nkinds: [read]\ndestroy: [delete_dns_record]\ntargets: [cloudflare-prod]\n"
	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, doc), cloudflareProviders(t), cloudflareTargets(t))
	mustCode(t, got, CodeDefinitionKindsMixed)
}

func TestCheckDefinition_MutateBesideDestroyDrawsNoCode(t *testing.T) {
	mustNoCode(t, CheckDefinition("definitions/preview-dns.yaml", parse(t, previewDNS), cloudflareProviders(t), cloudflareTargets(t)), CodeDefinitionKindsMixed)
}

func TestCheckDefinition_TargetClassMismatch(t *testing.T) {
	// local's own class is local, and cloudflare-dns's declared class is
	// cloudflare — binding it from a Definition using cloudflare-dns is
	// target-class-mismatch.
	doc := "kind: definition\ndefinition: preview-dns-observed\nprovider: cloudflare-dns\nkinds: [read]\ntargets: [local]\n"
	providers := cloudflareProviders(t)
	targets := BuildTargetIndex([]*yaml.Node{parse(t, localTarget)})
	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, doc), providers, targets)
	p := mustCode(t, got, CodeTargetClassMismatch)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
}

func TestCheckDefinition_CapabilityNotGrantedOnEveryDefinitionBindingShellWhereTheTargetOmitsIt(t *testing.T) {
	// A repository omitting shell from every class-local declaration
	// reports capability-not-granted for every Definition binding the
	// built-in shell Provider (issue #93's acceptance criterion).
	httpOnlyLocal := "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncapabilities: [http]\nhosts: [status.hyper.dev]\n"
	providers := ProviderIndex{"shell": builtinShellProviderInfo()}
	targets := BuildTargetIndex([]*yaml.Node{parse(t, httpOnlyLocal)})

	doc := "kind: definition\ndefinition: uptime\nprovider: shell\nkinds: [read]\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), providers, targets)
	p := mustCode(t, got, CodeCapabilityNotGranted)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
}

func TestCheckDefinition_CapabilityGrantedDrawsNoCode(t *testing.T) {
	providers := ProviderIndex{"shell": builtinShellProviderInfo()}
	targets := BuildTargetIndex([]*yaml.Node{parse(t, localTarget2)})
	doc := "kind: definition\ndefinition: uptime\nprovider: shell\nkinds: [read]\ntargets: [local]\n"
	mustNoCode(t, CheckDefinition("definitions/uptime.yaml", parse(t, doc), providers, targets), CodeCapabilityNotGranted)
}

// localTarget2 grants shell, unlike artefact_test.go's own localTarget
// fixture (which grants http, for the ordinary read Target it was written
// for).
const localTarget2 = `kind: target-declaration
target: local
class: local
kinds: [read]
capabilities: [shell]
`

func TestCheckDefinition_SlotCoverageMissingIsManifestInconsistent(t *testing.T) {
	// cloudflare-dns's header: scheme needs one slot, token; a Target
	// declaring no auth: block at all does not cover it.
	noAuthCloudflareProd := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\n"
	providers := cloudflareProviders(t)
	targets := BuildTargetIndex([]*yaml.Node{parse(t, noAuthCloudflareProd)})

	got := CheckDefinition("definitions/preview-dns-observed.yaml", parse(t, previewDNSObserved), providers, targets)
	p := mustCode(t, got, CodeManifestInconsistent)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
}

// TestCheckDefinition_OneTargetServesAHeaderProviderAndABasicProviderDrawsNoCode
// proves a Target carrying more slots than any one Provider needs draws no
// code (issue #93's acceptance criterion): one Target, bound from two
// Definitions against two Providers whose Auth schemes take different
// slots, covers both.
func TestCheckDefinition_OneTargetServesAHeaderProviderAndABasicProviderDrawsNoCode(t *testing.T) {
	headerProvider := "kind: provider\nprovider: header-svc\nschema-version: 1\nclass: shared\ncapabilities: [http]\nauth:\n  header: {name: Authorization, prefix: \"Bearer \"}\noperations:\n  read_it:\n    kind: read\n    deadline: 10s\n    http: {method: GET, host: \"{from-target}\", path: /}\n    record: {identity: $.id}\n"
	basicProvider := "kind: provider\nprovider: basic-svc\nschema-version: 1\nclass: shared\ncapabilities: [http]\nauth:\n  basic: {}\noperations:\n  read_it:\n    kind: read\n    deadline: 10s\n    http: {method: GET, host: \"{from-target}\", path: /}\n    record: {identity: $.id}\n"
	sharedTarget := "kind: target-declaration\ntarget: shared\nclass: shared\nkinds: [read]\ncapabilities: [http]\nhosts: [svc.example.com]\nauth:\n  token: {env: HEADER_TOKEN}\n  username: {env: BASIC_USER}\n  password: {env: BASIC_PASS}\n"

	providers := BuildProviderIndex([]*yaml.Node{parse(t, headerProvider), parse(t, basicProvider)})
	targets := BuildTargetIndex([]*yaml.Node{parse(t, sharedTarget)})

	headerDef := "kind: definition\ndefinition: uses-header\nprovider: header-svc\nkinds: [read]\ntargets: [shared]\n"
	basicDef := "kind: definition\ndefinition: uses-basic\nprovider: basic-svc\nkinds: [read]\ntargets: [shared]\n"

	got1 := CheckDefinition("definitions/uses-header.yaml", parse(t, headerDef), providers, targets)
	if len(got1) != 0 {
		t.Fatalf("CheckDefinition(header) = %+v, want no problems", got1)
	}
	got2 := CheckDefinition("definitions/uses-basic.yaml", parse(t, basicDef), providers, targets)
	if len(got2) != 0 {
		t.Fatalf("CheckDefinition(basic) = %+v, want no problems", got2)
	}
}

// --- issue #95: the two keys, the Bound, and the opaque destroy opt-ins ---

func TestCheckDefinition_OpaqueDestroyNotGranted(t *testing.T) {
	// shell's destroy Operation is opaque — its request is the shell
	// Capability — and localTarget2 (definition_test.go's own fixture)
	// carries no opaque-destroy: opt-in.
	doc := "kind: definition\ndefinition: cleanup\nprovider: shell\ndestroy: [destroy]\ntargets: [local]\n"
	targets := BuildTargetIndex([]*yaml.Node{parse(t, localTarget2)})
	got := CheckDefinition("definitions/cleanup.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, targets)
	p := mustCode(t, got, CodeOpaqueDestroyNotGranted)
	if p.Field != "targets[0]" {
		t.Errorf("Field = %q, want targets[0]", p.Field)
	}
}

func TestCheckDefinition_OpaqueDestroyGrantedDrawsNoCode(t *testing.T) {
	doc := "kind: definition\ndefinition: cleanup\nprovider: shell\ndestroy: [destroy]\ntargets: [local]\n"
	localOptedIn := "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncapabilities: [shell]\nopaque-destroy: true\n"
	targets := BuildTargetIndex([]*yaml.Node{parse(t, localOptedIn)})
	got := CheckDefinition("definitions/cleanup.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, targets)
	if len(got) != 0 {
		t.Fatalf("CheckDefinition() = %+v, want no problems", got)
	}
}

// TestCheckDefinition_NonOpaqueDestroyDrawsNoOpaqueDestroyCode proves
// opaque-destroy-not-granted reads opacity off the Capability an Operation's
// request uses rather than off destroy: alone — delete_dns_record is an
// http Operation, and cloudflare-prod (artefact_test.go's own fixture)
// carries no opaque-destroy: opt-in either.
func TestCheckDefinition_NonOpaqueDestroyDrawsNoOpaqueDestroyCode(t *testing.T) {
	mustNoCode(t, CheckDefinition("definitions/preview-dns.yaml", parse(t, previewDNS), cloudflareProviders(t), cloudflareTargets(t)), CodeOpaqueDestroyNotGranted)
}

// TestCheckDefinition_ClaimingNoKindsAndNoDestroyLoadsClean proves issue
// #95's acceptance criterion: "A Definition claiming no Kinds and no
// destroy: Operations loads clean" — the Step-level consequence, that every
// Step through it draws kind-not-granted, is procedure_test.go's own.
func TestCheckDefinition_ClaimingNoKindsAndNoDestroyLoadsClean(t *testing.T) {
	doc := "kind: definition\ndefinition: uptime\nprovider: shell\ntargets: [local]\n"
	got := CheckDefinition("definitions/uptime.yaml", parse(t, doc), ProviderIndex{"shell": builtinShellProviderInfo()}, BuildTargetIndex([]*yaml.Node{parse(t, localTarget2)}))
	if len(got) != 0 {
		t.Fatalf("CheckDefinition() = %+v, want no problems", got)
	}
}
