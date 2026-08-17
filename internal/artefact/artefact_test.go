package artefact

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

func parse(t *testing.T, doc string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &n); err != nil {
		t.Fatalf("Unmarshal(%q) = %v", doc, err)
	}
	if len(n.Content) == 0 {
		return nil
	}
	return n.Content[0]
}

func mustNone(t *testing.T, got []problem.Problem) {
	t.Helper()
	if len(got) != 0 {
		t.Fatalf("CheckRepositoryDeclaration() = %+v, want no problems", got)
	}
}

func mustCode(t *testing.T, got []problem.Problem, code string) problem.Problem {
	t.Helper()
	for _, p := range got {
		if p.ErrorCode == code {
			return p
		}
	}
	t.Fatalf("CheckRepositoryDeclaration() = %+v, want a %s problem", got, code)
	return problem.Problem{}
}

const clean = "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"

func TestCheckRepositoryDeclaration_Clean(t *testing.T) {
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, clean)))
}

func TestCheckRepositoryDeclaration_RetentionIsOptional(t *testing.T) {
	doc := clean + "retention: 90d\n"
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
}

func TestCheckRepositoryDeclaration_KindMismatch(t *testing.T) {
	doc := "kind: definition\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckRepositoryDeclaration_UnknownKey(t *testing.T) {
	doc := clean + "name: my-repo\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "name" {
		t.Errorf("Field = %q, want name", p.Field)
	}
}

func TestCheckRepositoryDeclaration_MissingDigestIsSchemaMismatch(t *testing.T) {
	doc := "kind: repository-declaration\nversion: 1.4.0\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "digest" {
		t.Errorf("Field = %q, want digest", p.Field)
	}
}

func TestCheckRepositoryDeclaration_BadDurationIsSchemaMismatch(t *testing.T) {
	doc := clean + "retention: 1d12h\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "retention" {
		t.Errorf("Field = %q, want retention", p.Field)
	}
}

func TestCheckRepositoryDeclaration_RetentionQuotedOrBareIsOneValue(t *testing.T) {
	for _, doc := range []string{clean + "retention: 90d\n", clean + "retention: \"90d\"\n"} {
		mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
	}
}

func TestCheckRepositoryDeclaration_KeyOrderIsFree(t *testing.T) {
	// Key order within a file is free and nothing checks it (§3).
	doc := "digest: sha256:0000000000000000000000000000000000000000000000000000000000000000\nversion: 1.4.0\nkind: repository-declaration\n"
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
}

func TestCheckRepositoryDeclaration_EmptyFileReportsThreeMissingKeys(t *testing.T) {
	got := CheckRepositoryDeclaration("hyper.yaml", nil)
	if len(got) != 3 {
		t.Fatalf("CheckRepositoryDeclaration(nil) = %+v, want 3 problems", got)
	}
	for _, p := range got {
		if p.ErrorCode != schema.CodeMismatch {
			t.Errorf("ErrorCode = %q, want %q", p.ErrorCode, schema.CodeMismatch)
		}
	}
}

func TestCheckRepositoryDeclaration_EnvElsewhereIsCredentialSlotMalformedNotUnknownKey(t *testing.T) {
	// env: is reserved across every artefact, hyper.yaml included, even
	// though hyper.yaml declares no credential slot of its own (§4).
	doc := clean + "env: OOPS\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	mustCode(t, got, CodeCredentialSlotMalformed)
	for _, p := range got {
		if p.ErrorCode == schema.CodeUnknownKey {
			t.Fatalf("CheckRepositoryDeclaration() = %+v, want no unknown-key alongside credential-slot-malformed", got)
		}
	}
}

func TestCheckRepositoryDeclaration_KindMismatchDoesNotFireWhenKindItselfFailsSchema(t *testing.T) {
	// kind: written as a mapping fails the schema (schema-mismatch); the
	// kind check has nothing legible to compare and says nothing more about
	// the same line.
	doc := "kind:\n  nested: true\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	for _, p := range got {
		if p.ErrorCode == CodeKindMismatch {
			t.Fatalf("CheckRepositoryDeclaration() = %+v, want no kind-mismatch alongside schema-mismatch", got)
		}
	}
	mustCode(t, got, schema.CodeMismatch)
}

// cloudflareProd and localTarget are §4's two worked Target declarations,
// byte for byte (issue #90's acceptance criteria: "§3's two worked Target
// declarations — cloudflare-prod and local — both check clean").
const cloudflareProd = `kind: target-declaration
target: cloudflare-prod
class: cloudflare
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.cloudflare.com]
auth:
  token: {env: CLOUDFLARE_API_TOKEN}
`

const localTarget = `kind: target-declaration
target: local
class: local
kinds: [read]
capabilities: [http]
hosts: [status.hyper.dev, cert.hyper.dev]
`

func TestCheckTargetDeclaration_CloudflareProdIsClean(t *testing.T) {
	mustNone(t, CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, cloudflareProd)))
}

func TestCheckTargetDeclaration_LocalIsClean(t *testing.T) {
	mustNone(t, CheckTargetDeclaration("targets/local.yaml", parse(t, localTarget)))
}

func TestCheckTargetDeclaration_KindMismatch(t *testing.T) {
	doc := "kind: definition\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read]\ncapabilities: []\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckTargetDeclaration_NameMismatch(t *testing.T) {
	got := CheckTargetDeclaration("targets/prod.yaml", parse(t, cloudflareProd))
	p := mustCode(t, got, CodeNameMismatch)
	if p.Field != "target" {
		t.Errorf("Field = %q, want target", p.Field)
	}
}

func TestCheckTargetDeclaration_NameMismatchIsNeverWidenedIntoKindMismatch(t *testing.T) {
	got := CheckTargetDeclaration("targets/prod.yaml", parse(t, cloudflareProd))
	for _, p := range got {
		if p.ErrorCode == CodeKindMismatch {
			t.Fatalf("CheckTargetDeclaration() = %+v, want no kind-mismatch alongside name-mismatch", got)
		}
	}
	mustCode(t, got, CodeNameMismatch)
}

func TestCheckTargetDeclaration_SchemaAdmitsExactlyTheDocumentedKeys(t *testing.T) {
	doc := cloudflareProd + "opaque-destroy: true\n"
	mustNone(t, CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc)))

	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, cloudflareProd+"extra: 1\n"))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "extra" {
		t.Errorf("Field = %q, want extra", p.Field)
	}
}

func TestCheckTargetDeclaration_ScalarCredentialSlotIsMalformedWithNoException(t *testing.T) {
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [mutate]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\nauth:\n  token: not-a-mapping\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	p := mustCode(t, got, CodeCredentialSlotMalformed)
	if p.Field != "auth.token" {
		t.Errorf("Field = %q, want auth.token", p.Field)
	}
}

func TestCheckTargetDeclaration_CredentialSlotWithWrongKeyIsMalformed(t *testing.T) {
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [mutate]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\nauth:\n  token: {name: CLOUDFLARE_API_TOKEN}\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	mustCode(t, got, CodeCredentialSlotMalformed)
}

func TestCheckTargetDeclaration_CredentialSlotWithMoreThanOneKeyIsMalformed(t *testing.T) {
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [mutate]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\nauth:\n  token: {env: CLOUDFLARE_API_TOKEN, extra: 1}\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	// Exactly one problem: the malformed slot itself, not a second report
	// for the env: key its own shape still carries.
	if len(got) != 1 {
		t.Fatalf("CheckTargetDeclaration() = %+v, want exactly one problem", got)
	}
	if got[0].ErrorCode != CodeCredentialSlotMalformed || got[0].Field != "auth.token" {
		t.Errorf("got %+v, want one credential-slot-malformed at auth.token", got[0])
	}
}

func TestCheckTargetDeclaration_EnvWrittenOutsideACredentialSlotIsMalformed(t *testing.T) {
	doc := cloudflareProd + "env: OOPS\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	mustCode(t, got, CodeCredentialSlotMalformed)
	for _, p := range got {
		if p.ErrorCode == schema.CodeUnknownKey {
			t.Fatalf("CheckTargetDeclaration() = %+v, want no unknown-key alongside credential-slot-malformed", got)
		}
	}
}

func TestCheckTargetDeclaration_LocalWithAuthBlockIsReserved(t *testing.T) {
	doc := "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncapabilities: [http]\nhosts: [status.hyper.dev]\nauth:\n  token: {env: SOME_TOKEN}\n"
	got := CheckTargetDeclaration("targets/local.yaml", parse(t, doc))
	p := mustCode(t, got, CodeLocalReserved)
	if p.Field != "auth" {
		t.Errorf("Field = %q, want auth", p.Field)
	}
}

func TestCheckTargetDeclaration_LocalWithNonLocalClassIsReserved(t *testing.T) {
	doc := "kind: target-declaration\ntarget: local\nclass: cloudflare\nkinds: [read]\ncapabilities: [http]\nhosts: [status.hyper.dev]\n"
	got := CheckTargetDeclaration("targets/local.yaml", parse(t, doc))
	p := mustCode(t, got, CodeLocalReserved)
	if p.Field != "class" {
		t.Errorf("Field = %q, want class", p.Field)
	}
}

func TestCheckTargetDeclaration_NonLocalNameClaimingClassLocalIsLegal(t *testing.T) {
	// More than one declaration may claim class: local — two names for the
	// machine hyper runs on, each with its own grant (ADR-0041).
	doc := "kind: target-declaration\ntarget: build-box\nclass: local\nkinds: [read, mutate]\ncapabilities: [http]\nhosts: [status.hyper.dev]\nauth:\n  token: {env: BUILD_BOX_TOKEN}\nopaque-destroy: true\n"
	mustNone(t, CheckTargetDeclaration("targets/build-box.yaml", parse(t, doc)))
}

func TestCheckTargetDeclaration_HostsPresentWithoutHTTPIsInconsistent(t *testing.T) {
	doc := "kind: target-declaration\ntarget: shell-only\nclass: local\nkinds: [mutate]\ncapabilities: [shell]\nhosts: [example.com]\n"
	got := CheckTargetDeclaration("targets/shell-only.yaml", parse(t, doc))
	p := mustCode(t, got, CodeTargetInconsistent)
	if p.Field != "hosts" {
		t.Errorf("Field = %q, want hosts", p.Field)
	}
}

func TestCheckTargetDeclaration_HostsAbsentWithHTTPIsInconsistent(t *testing.T) {
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read]\ncapabilities: [http]\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	p := mustCode(t, got, CodeTargetInconsistent)
	if p.Field != "hosts" {
		t.Errorf("Field = %q, want hosts", p.Field)
	}
}

func TestCheckTargetDeclaration_KindsAndCapabilitiesEnumsAreClosed(t *testing.T) {
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [delete]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "kinds[0]" {
		t.Errorf("Field = %q, want kinds[0]", p.Field)
	}

	doc = "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read]\ncapabilities: [ftp]\n"
	got = CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))
	p = mustCode(t, got, schema.CodeMismatch)
	if p.Field != "capabilities[0]" {
		t.Errorf("Field = %q, want capabilities[0]", p.Field)
	}
}

func TestCheckTargetDeclaration_EnvNestedInsideAnotherKeyOfAMalformedSlotIsFlaggedSeparately(t *testing.T) {
	// The malformed slot itself earns one problem; the env: nested inside
	// its unrelated extra key still reads as env: written outside a
	// credential slot and earns its own — env: is reserved everywhere but a
	// credential slot's own sole key, with no carve-out for where the
	// mapping around it came from.
	doc := "kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [mutate]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\nauth:\n  token:\n    env: X\n    extra:\n      env: Y\n"
	got := CheckTargetDeclaration("targets/cloudflare-prod.yaml", parse(t, doc))

	var malformed []problem.Problem
	for _, p := range got {
		if p.ErrorCode == CodeCredentialSlotMalformed {
			malformed = append(malformed, p)
		}
	}
	if len(malformed) != 2 {
		t.Fatalf("CheckTargetDeclaration() = %+v, want exactly 2 credential-slot-malformed problems", got)
	}
	if malformed[0].Field != "auth.token" {
		t.Errorf("Field = %q, want auth.token for the malformed slot itself", malformed[0].Field)
	}
	if malformed[1].Field != "auth.token.extra.env" {
		t.Errorf("Field = %q, want auth.token.extra.env for the nested env:", malformed[1].Field)
	}
}
