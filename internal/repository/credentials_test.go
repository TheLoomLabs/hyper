package repository

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// boundRepository writes the least a (Definition, Target) pair needs to resolve
// a credential slot: a Manifest declaring an auth scheme, a Target declaring
// what its slots are read from, and a Definition binding the two.
//
// The Target carries a slot the scheme does not name — `password`, under a
// `header:` scheme, which reaches `token` and nothing else — because that is the
// half of the rule a fixture without one cannot state: what a job's `env:` block
// carries is what the **binding** requires, not what a declaration happens to
// hold (§10, ADR-0007).
func boundRepository(t *testing.T) Loaded {
	t.Helper()
	root := t.TempDir()
	write(t, root, "hyper.yaml", "kind: repository-declaration\nversion: 1.4.0\n")
	write(t, root, "providers/cloudflare-dns.yaml",
		"kind: provider\nprovider: cloudflare-dns\nauth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n")
	write(t, root, "providers/uptime.yaml", "kind: provider\nprovider: uptime\n")
	write(t, root, "targets/cloudflare-prod.yaml",
		"kind: target-declaration\ntarget: cloudflare-prod\nauth:\n  token: {env: CLOUDFLARE_API_TOKEN}\n  password: {env: NOBODY_ASKED}\n")
	write(t, root, "definitions/preview-dns.yaml",
		"kind: definition\ndefinition: preview-dns\nprovider: cloudflare-dns\ntargets: [cloudflare-prod]\n")
	write(t, root, "definitions/heartbeat.yaml",
		"kind: definition\ndefinition: heartbeat\nprovider: uptime\ntargets: [cloudflare-prod]\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

// TestCredentialSlots_IsTheSchemesSlotsAndNotTheDeclarationsWholeAuth is the
// rule both callers rest on: a Target may carry slots for a scheme this binding
// never uses, and neither a Run's gate nor a generated job may reach one.
func TestCredentialSlots_IsTheSchemesSlotsAndNotTheDeclarationsWholeAuth(t *testing.T) {
	file, slots := boundRepository(t).CredentialSlots(store.Pair{Definition: "preview-dns", Target: "cloudflare-prod"})

	if want := "targets/cloudflare-prod.yaml"; file != want {
		t.Errorf("file is %q, want the declaration's own %q", file, want)
	}
	if len(slots) != 1 || slots[0].Slot != "token" || slots[0].Env != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("slots are %+v, want the header scheme's token alone", slots)
	}
	if slots[0].Line == 0 {
		t.Error("the slot carries no line; a Refusal cites the line an author edits")
	}
}

// TestCredentialSlots_ASchemelessProviderRequiresNothing is the other end of the
// same reading: a Manifest declaring no auth reaches no slot, however much the
// Target declares.
func TestCredentialSlots_ASchemelessProviderRequiresNothing(t *testing.T) {
	_, slots := boundRepository(t).CredentialSlots(store.Pair{Definition: "heartbeat", Target: "cloudflare-prod"})
	if len(slots) != 0 {
		t.Errorf("slots are %+v, want none: the Provider declares no scheme", slots)
	}
}

// TestCredentialSlots_APairThatDoesNotResolveRequiresNothing is ADR-0064 at this
// reader: what is wrong with a name is `check`'s to report, and a second opinion
// here would put two answers on one fault. Either end of the pair failing to
// resolve leaves the binding requiring no slot at all.
func TestCredentialSlots_APairThatDoesNotResolveRequiresNothing(t *testing.T) {
	loaded := boundRepository(t)
	for what, pair := range map[string]store.Pair{
		"no such Definition": {Definition: "nobody", Target: "cloudflare-prod"},
		"no such Target":     {Definition: "preview-dns", Target: "nowhere"},
	} {
		t.Run(what, func(t *testing.T) {
			if _, slots := loaded.CredentialSlots(pair); len(slots) != 0 {
				t.Errorf("slots are %+v, want none", slots)
			}
		})
	}
}

// TestCredentialSlots_AnUnresolvableTargetNamesNoFile is the one answer that is
// two absences rather than one: with no declaration there is no path for a
// Refusal to cite, and nothing to read a slot off.
func TestCredentialSlots_AnUnresolvableTargetNamesNoFile(t *testing.T) {
	if file, _ := boundRepository(t).CredentialSlots(store.Pair{Definition: "preview-dns", Target: "nowhere"}); file != "" {
		t.Errorf("file is %q, want none: no declaration was read", file)
	}
}
