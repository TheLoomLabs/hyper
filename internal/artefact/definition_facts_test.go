package artefact

import (
	"slices"
	"testing"
)

// TestReadDefinitionFacts_ReadsTheDefinitionsOwnLists is the left end of §5's
// two-key check read off one file: what it claims, what it names for `destroy`,
// and which Targets it claims.
func TestReadDefinitionFacts_ReadsTheDefinitionsOwnLists(t *testing.T) {
	facts := ReadDefinitionFacts(parse(t, previewDNS))

	if got, want := facts.Kinds, []string{"mutate"}; !slices.Equal(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
	if got, want := facts.Destroy, []string{"delete_dns_record"}; !slices.Equal(got, want) {
		t.Errorf("destroy = %v, want %v", got, want)
	}
	if got, want := facts.Targets, []string{"cloudflare-prod"}; !slices.Equal(got, want) {
		t.Errorf("targets = %v, want %v", got, want)
	}
}

// TestReadDefinitionFacts_KeepsTheDefinitionsOwnOrder is ReadTargetFacts's own
// rule on the other end of the relation: a claim re-sorted here is a second
// reading of an artefact the reviewer has open beside the table (§3, §8).
func TestReadDefinitionFacts_KeepsTheDefinitionsOwnOrder(t *testing.T) {
	facts := ReadDefinitionFacts(parse(t, `kind: definition
definition: many
provider: hostco
kinds: [mutate, read]
destroy: [delete_server, delete_volume]
targets: [staging, production]
`))

	if got, want := facts.Kinds, []string{"mutate", "read"}; !slices.Equal(got, want) {
		t.Errorf("kinds = %v, want %v — the Definition's own order", got, want)
	}
	if got, want := facts.Destroy, []string{"delete_server", "delete_volume"}; !slices.Equal(got, want) {
		t.Errorf("destroy = %v, want %v — the Definition's own order", got, want)
	}
	if got, want := facts.Targets, []string{"staging", "production"}; !slices.Equal(got, want) {
		t.Errorf("targets = %v, want %v — the Definition's own order", got, want)
	}
}

// TestReadDefinitionFacts_AnAbsentKeyReadsAsNoMember is the ordinary absence
// rule: a Definition claiming no `destroy` Operation carries no destroy: at
// all, and what stands there is nothing rather than a member to interpret (§7).
func TestReadDefinitionFacts_AnAbsentKeyReadsAsNoMember(t *testing.T) {
	facts := ReadDefinitionFacts(parse(t, previewDNSObserved))

	if len(facts.Destroy) != 0 {
		t.Errorf("destroy = %v, want none", facts.Destroy)
	}
}
