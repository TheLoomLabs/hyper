package artefact

import (
	"slices"
	"testing"
)

// TestReadTargetFacts_ReadsTheDeclarationsOwnLists is the row's four
// list-shaped facts read off one declaration — §4's own worked cloudflare-prod:
// what it grants, what it accepts, and which variable fills each credential
// slot.
func TestReadTargetFacts_ReadsTheDeclarationsOwnLists(t *testing.T) {
	facts := ReadTargetFacts(parse(t, cloudflareProd))

	if got, want := facts.Hosts, []string{"api.cloudflare.com"}; !slices.Equal(got, want) {
		t.Errorf("hosts = %v, want %v", got, want)
	}
	if got, want := facts.Kinds, []string{"read", "mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
	if got, want := facts.Capabilities, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities = %v, want %v", got, want)
	}
	if got, want := facts.Credentials, []CredentialSlot{{Slot: "token", Env: "CLOUDFLARE_API_TOKEN"}}; !slices.Equal(got, want) {
		t.Errorf("credentials = %v, want %v", got, want)
	}
}

// TestReadTargetFacts_KeepsTheDeclarationsOwnOrder is what makes a grant
// reportable at all: a Target granting several hosts grants all of them, and a
// list re-sorted here would be a second reading of an artefact the reader has
// open beside it (§3, ADR-0024, ADR-0029).
func TestReadTargetFacts_KeepsTheDeclarationsOwnOrder(t *testing.T) {
	facts := ReadTargetFacts(parse(t, `kind: target-declaration
target: multi
class: hostco
kinds: [mutate, read, destroy]
capabilities: [shell, http]
hosts: [zulu.hostco.dev, alpha.hostco.dev, mike.hostco.dev]
auth:
  password: {env: HOSTCO_PASSWORD}
  username: {env: HOSTCO_USERNAME}
`))

	if got, want := facts.Hosts, []string{"zulu.hostco.dev", "alpha.hostco.dev", "mike.hostco.dev"}; !slices.Equal(got, want) {
		t.Errorf("hosts = %v, want %v — the declaration's own order", got, want)
	}
	if got, want := facts.Kinds, []string{"mutate", "read", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("kinds = %v, want %v — the declaration's own order", got, want)
	}
	if got, want := facts.Capabilities, []string{"shell", "http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities = %v, want %v — the declaration's own order", got, want)
	}
	if got, want := facts.Credentials, []CredentialSlot{
		{Slot: "password", Env: "HOSTCO_PASSWORD"},
		{Slot: "username", Env: "HOSTCO_USERNAME"},
	}; !slices.Equal(got, want) {
		t.Errorf("credentials = %v, want %v — the auth: mapping's own order", got, want)
	}
}

// TestReadTargetFacts_ADeclarationWithNoAuthBlockCarriesNoCredentialSlot is
// `local`'s one visible consequence: the reserved name carries no auth: block
// at all — writing one is local-reserved — so there is no slot to pair with a
// variable (§4, ADR-0041).
func TestReadTargetFacts_ADeclarationWithNoAuthBlockCarriesNoCredentialSlot(t *testing.T) {
	facts := ReadTargetFacts(parse(t, localTarget))

	if len(facts.Credentials) != 0 {
		t.Errorf("credentials = %v, want none", facts.Credentials)
	}
	if got, want := facts.Hosts, []string{"status.hyper.dev", "cert.hyper.dev"}; !slices.Equal(got, want) {
		t.Errorf("hosts = %v, want %v: local grants enumerated hosts like any other Target", got, want)
	}
}

// TestReadTargetFacts_ADeclarationGrantingNoHostEnumeratesNone is the other
// half of hosts:, which is the one key here that a declaration may leave out: a
// Target granting no http carries no host grant, and what stands there is
// nothing rather than an empty enumeration (§3, §4).
func TestReadTargetFacts_ADeclarationGrantingNoHostEnumeratesNone(t *testing.T) {
	facts := ReadTargetFacts(parse(t, `kind: target-declaration
target: shell-only
class: local
kinds: [read]
capabilities: [shell]
`))

	if facts.Hosts != nil {
		t.Errorf("hosts = %v, want none: this declaration grants no http and enumerates none", facts.Hosts)
	}
}

// TestReadTargetFacts_ASlotNamingNoVariableIsCarriedWithoutOne is ADR-0064's
// rule read onto a credential slot: a slot whose env: is missing or malformed
// resolves from no variable, and check names that fault under
// credential-slot-malformed. The slot is still a slot the declaration carries,
// so it is reported — with nothing standing in for the variable it does not
// name.
func TestReadTargetFacts_ASlotNamingNoVariableIsCarriedWithoutOne(t *testing.T) {
	facts := ReadTargetFacts(parse(t, `kind: target-declaration
target: malformed
class: hostco
kinds: [read]
capabilities: [http]
hosts: [api.hostco.dev]
auth:
  token: PLAINTEXT
  other: {name: NOT_ENV}
`))

	want := []CredentialSlot{{Slot: "token"}, {Slot: "other"}}
	if !slices.Equal(facts.Credentials, want) {
		t.Errorf("credentials = %v, want %v", facts.Credentials, want)
	}
}

// TestReadTargetFacts_ReadsNothingOffADocumentThatIsNotAMapping is the
// tolerance every reader in this package has: a root that is nil, or a scalar,
// or a list, states none of these facts, and what is wrong with it is a
// problem check reports rather than one this reader guesses at.
func TestReadTargetFacts_ReadsNothingOffADocumentThatIsNotAMapping(t *testing.T) {
	for name, doc := range map[string]string{
		"empty":  "",
		"scalar": "just-a-scalar\n",
		"list":   "- one\n- two\n",
	} {
		t.Run(name, func(t *testing.T) {
			facts := ReadTargetFacts(parse(t, doc))
			if facts.Hosts != nil || facts.Kinds != nil || facts.Capabilities != nil || facts.Credentials != nil {
				t.Errorf("facts = %+v, want nothing read", facts)
			}
		})
	}
}

// TestTargetDeclarationName_IsTheDeclarationsOwnTargetKey is the rule the
// Target namespace is folded on, exported so the fold that decides which file
// a name means and the index a targets: resolves against read one rule rather
// than two copies of it (issue #112).
func TestTargetDeclarationName_IsTheDeclarationsOwnTargetKey(t *testing.T) {
	if got := TargetDeclarationName(parse(t, cloudflareProd)); got != "cloudflare-prod" {
		t.Errorf("name = %q, want %q", got, "cloudflare-prod")
	}
	for name, doc := range map[string]string{
		"absent":     "kind: target-declaration\nclass: local\n",
		"not scalar": "kind: target-declaration\ntarget: [a, b]\n",
	} {
		t.Run(name, func(t *testing.T) {
			if got := TargetDeclarationName(parse(t, doc)); got != "" {
				t.Errorf("name = %q, want \"\": a declaration that does not name itself names nothing", got)
			}
		})
	}
}
