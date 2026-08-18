package artefact

import (
	"slices"
	"testing"
)

// headerSchemeManifest is §3's own worked Manifest as far as this file needs
// it: a header: scheme carrying the trailing space that makes `Bearer ` a
// prefix rather than a word, one Capability, a schema version, and three
// Operations authored out of name order so that nothing here can pass by
// having read the mapping in the order it was written.
const headerSchemeManifest = `kind: provider
provider: cloudflare-dns
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_dns_records:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id}
  create_dns_record:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
    record:
      identity: "{name}"
      fields: {id: $.body.result.id}
  delete_dns_record:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records/{record_id}
`

// operationNames is the facts' Operations in the order they were read, which
// is the Manifest's own: the sort onto name order is the surface's rule and
// not this reader's (§9).
func operationNames(facts ManifestFacts) []string {
	var names []string
	for _, op := range facts.Operations {
		names = append(names, op.Name)
	}
	return names
}

// operationNamed is one Operation's facts by name, for the cases that assert
// about a single row.
func operationNamed(t *testing.T, facts ManifestFacts, name string) OperationFacts {
	t.Helper()
	for _, op := range facts.Operations {
		if op.Name == name {
			return op
		}
	}
	t.Fatalf("no Operation named %q; the Manifest declares %v", name, operationNames(facts))
	return OperationFacts{}
}

// TestReadManifestFacts_ReadsTheManifestsOwnHeaderFacts is the header row's
// whole content bar the digest, which is taken over bytes rather than read off
// a parse tree: what the Manifest declares about itself (§9).
func TestReadManifestFacts_ReadsTheManifestsOwnHeaderFacts(t *testing.T) {
	facts := ReadManifestFacts(parse(t, headerSchemeManifest))

	if got, want := facts.AuthScheme, "Authorization: Bearer <secret>"; got != want {
		t.Errorf("auth scheme = %q, want %q", got, want)
	}
	if got, want := facts.Capabilities, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities = %v, want %v", got, want)
	}
	if facts.SchemaVersion == nil || *facts.SchemaVersion != 1 {
		t.Errorf("schema version = %v, want 1", facts.SchemaVersion)
	}
	if facts.OriginRef != "" || facts.OriginDigest != "" {
		t.Errorf("origin = (%q, %q), want neither: the Manifest carries no origin: block", facts.OriginRef, facts.OriginDigest)
	}
}

// TestReadManifestFacts_AnAuthSchemeRendersAsTheHeaderItComposes is §9's whole
// rule for the Auth scheme: the composition rather than the parameters, with
// the credential's position marked by §7's one constant. The prefix is
// concatenated verbatim, so a Provider that omitted the trailing space in
// "Bearer " reads here as the header it actually writes.
func TestReadManifestFacts_AnAuthSchemeRendersAsTheHeaderItComposes(t *testing.T) {
	for _, c := range []struct {
		name, auth, want string
	}{
		{"a bearer token", "auth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n", "Authorization: Bearer <secret>"},
		{"a prefix missing its space", "auth:\n  header: {name: Authorization, prefix: \"Bearer\"}\n", "Authorization: Bearer<secret>"},
		{"an api key in a header of its own", "auth:\n  header: {name: X-Api-Key}\n", "X-Api-Key: <secret>"},
		{"a compound token", "auth:\n  header: {name: Authorization, prefix: \"PVEAPIToken=\"}\n", "Authorization: PVEAPIToken=<secret>"},
		{"basic", "auth:\n  basic: {}\n", "Authorization: Basic <secret>"},
		{"no auth block at all", "", "none"},
	} {
		t.Run(c.name, func(t *testing.T) {
			facts := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
`+c.auth+`operations: {}
`))
			if got := facts.AuthScheme; got != c.want {
				t.Errorf("auth scheme = %q, want %q", got, c.want)
			}
		})
	}
}

// TestReadManifestFacts_TheMarkerIsTheOneConstant holds the marker every
// rendering shares against the two schemes that write it: a second spelling
// here would be a second thing to recognise (§7).
func TestReadManifestFacts_TheMarkerIsTheOneConstant(t *testing.T) {
	for _, auth := range []string{
		"auth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n",
		"auth:\n  basic: {}\n",
	} {
		facts := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
`+auth+`operations: {}
`))
		if !slices.Contains([]string{
			"Authorization: Bearer " + SecretMarker,
			"Authorization: Basic " + SecretMarker,
		}, facts.AuthScheme) {
			t.Errorf("auth scheme = %q, want it composed from SecretMarker", facts.AuthScheme)
		}
	}
}

// TestReadManifestFacts_AnOriginBlockIsReadWholeOrNotAtAll is the ordinary
// absence rule over the one block no other surface renders: both members where
// the block is there, neither where it is not (§9, ADR-0073).
func TestReadManifestFacts_AnOriginBlockIsReadWholeOrNotAtAll(t *testing.T) {
	installed := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
origin:
  ref: ghcr.io/widgetco/widget:1.2.0
  digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
operations: {}
`))
	if got, want := installed.OriginRef, "ghcr.io/widgetco/widget:1.2.0"; got != want {
		t.Errorf("origin ref = %q, want %q", got, want)
	}
	if got, want := installed.OriginDigest, "sha256:0000000000000000000000000000000000000000000000000000000000000000"; got != want {
		t.Errorf("origin digest = %q, want %q", got, want)
	}

	authored := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations: {}
`))
	if authored.OriginRef != "" || authored.OriginDigest != "" {
		t.Errorf("origin = (%q, %q), want neither", authored.OriginRef, authored.OriginDigest)
	}
}

// TestReadManifestFacts_KeepsTheManifestsOwnOperationOrder is what makes the
// sort provable elsewhere: the reader states what the artefact states, and the
// normative order a listing is ranged over in is applied by the surface (§9).
func TestReadManifestFacts_KeepsTheManifestsOwnOperationOrder(t *testing.T) {
	facts := ReadManifestFacts(parse(t, headerSchemeManifest))

	want := []string{"list_dns_records", "create_dns_record", "delete_dns_record"}
	if got := operationNames(facts); !slices.Equal(got, want) {
		t.Errorf("operations = %v, want %v — the Manifest's own order", got, want)
	}
}

// TestReadManifestFacts_KindIsReadFromTheManifestAndNeverFromTheName is §12's
// rule at the one place a reader is tempted to infer it: an Operation called
// delete_thing that declares mutate is a mutate.
func TestReadManifestFacts_KindIsReadFromTheManifestAndNeverFromTheName(t *testing.T) {
	facts := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  delete_thing:
    kind: mutate
    deadline: 30s
    http: {method: POST, host: "{from-target}", path: /things}
`))

	if got, want := operationNamed(t, facts, "delete_thing").Kind, "mutate"; got != want {
		t.Errorf("kind = %q, want %q — read from kind:, never inferred from the name", got, want)
	}
}

// TestReadManifestFacts_OpacityIsReadFromTheRequestBlock is §12's other
// derived fact: opacity is a property of the Capability the request uses, so
// it is read from shell: or http: and declared by no artefact anywhere.
func TestReadManifestFacts_OpacityIsReadFromTheRequestBlock(t *testing.T) {
	shell := ReadManifestFacts(BuiltinShellProviderRoot())
	if !operationNamed(t, shell, "read").Opaque {
		t.Error("the built-in shell Provider's read is not opaque; a shell request is")
	}

	http := ReadManifestFacts(parse(t, headerSchemeManifest))
	if operationNamed(t, http, "list_dns_records").Opaque {
		t.Error("an http Operation is opaque; hyper describes what it does")
	}
}

// TestReadManifestFacts_TheSummaryIsDerivedFromWhatTheManifestStates is the
// row's fourth member: the request, the Repeatability in force, and what the
// Operation projects — three facts a caller would otherwise re-derive (§9).
func TestReadManifestFacts_TheSummaryIsDerivedFromWhatTheManifestStates(t *testing.T) {
	facts := ReadManifestFacts(parse(t, headerSchemeManifest))

	for _, c := range []struct{ operation, want string }{
		{"list_dns_records", "GET /client/v4/zones/{zone_id}/dns_records; repeatable; projects a Record series"},
		{"create_dns_record", "POST /client/v4/zones/{zone_id}/dns_records; skip-if-recorded; projects one Record"},
		{"delete_dns_record", "DELETE /client/v4/zones/{zone_id}/dns_records/{record_id}; repeatable; projects no Record"},
	} {
		if got := operationNamed(t, facts, c.operation).Summary; got != c.want {
			t.Errorf("%s summary = %q, want %q", c.operation, got, c.want)
		}
	}
}

// TestReadManifestFacts_TheSummaryStatesTheRepeatabilityInForce is the one
// summary clause with no spelling in the source: run-once is what an effectful
// Operation declaring no repeatability: is, and §12 gives it no keyword to
// author (§12, issue #96).
func TestReadManifestFacts_TheSummaryStatesTheRepeatabilityInForce(t *testing.T) {
	facts := ReadManifestFacts(BuiltinShellProviderRoot())

	for _, c := range []struct{ operation, want string }{
		{"read", "shell; repeatable; projects one Record"},
		{"mutate_once", "shell; run-once; projects one Record"},
		{"mutate_skip_if_recorded", "shell; skip-if-recorded; projects one Record"},
		{"destroy_once", "shell; run-once; projects no Record"},
	} {
		if got := operationNamed(t, facts, c.operation).Summary; got != c.want {
			t.Errorf("%s summary = %q, want %q", c.operation, got, c.want)
		}
	}
}

// TestReadManifestFacts_AFactTheManifestNeverStatedIsDropped is Summary's own
// rule applied one level down: a Manifest that parses and names itself is in
// the Provider namespace whatever else it left out, and a summary reading
// "; ; projects no Record" would state declarations that are not there
// (ADR-0064).
func TestReadManifestFacts_AFactTheManifestNeverStatedIsDropped(t *testing.T) {
	facts := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  half_written: {}
`))

	op := operationNamed(t, facts, "half_written")
	if got, want := op.Summary, "projects no Record"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if op.Kind != "" {
		t.Errorf("kind = %q, want empty: the Operation declares none", op.Kind)
	}
}

// TestReadManifestFacts_AManifestHyperCannotReadStatesWhatItCan is ADR-0064
// at this reader: a fault is check's to report, and a surface reports the
// facts that are legible rather than guessing at the ones that are not.
func TestReadManifestFacts_AManifestHyperCannotReadStatesWhatItCan(t *testing.T) {
	facts := ReadManifestFacts(parse(t, `kind: provider
provider: widget
schema-version: one
class: widgetco
capabilities: [http]
auth:
  header: {prefix: "Bearer "}
operations: {}
`))

	if facts.SchemaVersion != nil {
		t.Errorf("schema version = %v, want none: the Manifest's own is not an integer", *facts.SchemaVersion)
	}
	if facts.AuthScheme != "" {
		t.Errorf("auth scheme = %q, want none: the scheme names no header to compose", facts.AuthScheme)
	}
}
