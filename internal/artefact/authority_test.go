package artefact

import (
	"slices"
	"testing"
)

// demoAuthority is the five-artefact demo's two namespaces as the table reads
// them: two Definitions claiming one Target, and the two Targets §4 works.
func demoAuthority(t *testing.T) Authority {
	t.Helper()
	return Authority{
		Definitions: map[string]DefinitionFacts{
			"preview-dns":          ReadDefinitionFacts(parse(t, previewDNS)),
			"preview-dns-observed": ReadDefinitionFacts(parse(t, previewDNSObserved)),
		},
		Targets: map[string]TargetFacts{
			"cloudflare-prod": ReadTargetFacts(parse(t, cloudflareProd)),
			"local":           ReadTargetFacts(parse(t, localTarget)),
		},
	}
}

// pairsOf is the rows read back as the pairs they are about, which is what
// every case about the *filter* asserts against.
func pairsOf(rows []AuthorityRow) [][2]string {
	pairs := make([][2]string, 0, len(rows))
	for _, row := range rows {
		pairs = append(pairs, [2]string{row.Definition, row.Target})
	}
	return pairs
}

// TestAuthorityRows_ADefinitionRendersARowPerTargetItClaims is the left end of
// the relation: the artefact under review supplies the Definition and the rows
// are its targets: list (§8, ADR-0069).
func TestAuthorityRows_ADefinitionRendersARowPerTargetItClaims(t *testing.T) {
	table := demoAuthority(t).Table(KindDefinition, parse(t, previewDNS))
	rows := table.Rows
	if !table.Renders {
		t.Fatalf("the table did not render on a Definition; it supplies the left end")
	}

	want := [][2]string{{"preview-dns", "cloudflare-prod"}}
	if got := pairsOf(rows); !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	row := rows[0]
	if got, want := row.DefinitionKinds, []string{"mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("definition kinds = %v, want %v — destroy derived from destroy:", got, want)
	}
	if got, want := row.TargetKinds, []string{"read", "mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("target kinds = %v, want %v", got, want)
	}
	if got, want := row.Effective, []string{"mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("effective = %v, want %v — the intersection", got, want)
	}
	if got, want := row.DestroyOperations, []string{"delete_dns_record"}; !slices.Equal(got, want) {
		t.Errorf("destroy operations = %v, want %v", got, want)
	}
}

// TestAuthorityRows_ADefinitionClaimingNoDestroyOperationCarriesNoDestroyKind
// is granularity following severity read the other way: `destroy` is derived at
// the claimed-Kinds column and nowhere else, so a Definition naming none does
// not carry it (§3, §8).
func TestAuthorityRows_ADefinitionClaimingNoDestroyOperationCarriesNoDestroyKind(t *testing.T) {
	rows := demoAuthority(t).Table(KindDefinition, parse(t, previewDNSObserved)).Rows
	if len(rows) != 1 {
		t.Fatalf("rows = %v, want one", pairsOf(rows))
	}
	if got, want := rows[0].DefinitionKinds, []string{"read"}; !slices.Equal(got, want) {
		t.Errorf("definition kinds = %v, want %v", got, want)
	}
	if got, want := rows[0].Effective, []string{"read"}; !slices.Equal(got, want) {
		t.Errorf("effective = %v, want %v", got, want)
	}
	if len(rows[0].DestroyOperations) != 0 {
		t.Errorf("destroy operations = %v, want none", rows[0].DestroyOperations)
	}
}

// TestAuthorityRows_ATargetDeclarationRendersARowPerDefinitionThatClaimsIt is
// the reading an unaided implementation withholds: the right end supplies the
// filter, and the row set is discovered across definitions/ (ADR-0069).
func TestAuthorityRows_ATargetDeclarationRendersARowPerDefinitionThatClaimsIt(t *testing.T) {
	table := demoAuthority(t).Table(KindTargetDeclaration, parse(t, cloudflareProd))
	rows := table.Rows
	if !table.Renders {
		t.Fatalf("the table did not render on a Target declaration; it supplies the right end")
	}

	want := [][2]string{
		{"preview-dns", "cloudflare-prod"},
		{"preview-dns-observed", "cloudflare-prod"},
	}
	if got := pairsOf(rows); !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if got, want := rows[0].DefinitionKinds, []string{"mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("definition kinds = %v, want %v — the claiming Definition's own", got, want)
	}
}

// TestAuthorityRows_ATargetDeclarationNothingClaimsRendersNoRow is the empty
// state's supply: the table renders and has nothing to put in it, which is a
// different answer from not rendering at all (§8, ADR-0012, ADR-0069).
func TestAuthorityRows_ATargetDeclarationNothingClaimsRendersNoRow(t *testing.T) {
	table := demoAuthority(t).Table(KindTargetDeclaration, parse(t, localTarget))
	rows := table.Rows
	if !table.Renders {
		t.Fatalf("the table did not render on a Target declaration nothing claims")
	}
	if len(rows) != 0 {
		t.Errorf("rows = %v, want none", pairsOf(rows))
	}
}

// TestAuthorityRows_AProcedureRendersARowPerDistinctPairItsStepsBind is the
// artefact that supplies neither end: two Steps sharing a pairing are one row,
// and a Step is not an occurrence to be counted (§8, ADR-0069).
func TestAuthorityRows_AProcedureRendersARowPerDistinctPairItsStepsBind(t *testing.T) {
	table := demoAuthority(t).Table(KindProcedure, parse(t, `kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
  - id: publish-aliases
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
  - id: observe
    definition: preview-dns-observed
    operation: list_dns_records
    target: cloudflare-prod
`))
	rows := table.Rows
	if !table.Renders {
		t.Fatalf("the table did not render on a Procedure; its Steps bind pairs")
	}

	want := [][2]string{
		{"preview-dns", "cloudflare-prod"},
		{"preview-dns-observed", "cloudflare-prod"},
	}
	if got := pairsOf(rows); !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v — two Steps sharing a pairing are one row", got, want)
	}
}

// TestAuthorityRows_RowsSortByTargetThenDefinition is §7's identity-set
// ordering reused rather than reinvented, so two renderings of one review are
// byte-identical and step order is refused where it exists (§8, ADR-0069).
func TestAuthorityRows_RowsSortByTargetThenDefinition(t *testing.T) {
	supply := Authority{
		Definitions: map[string]DefinitionFacts{
			"alpha": {Kinds: []string{"read"}, Targets: []string{"zulu", "alpha"}},
			"zulu":  {Kinds: []string{"read"}, Targets: []string{"zulu", "alpha"}},
		},
		Targets: map[string]TargetFacts{
			"alpha": {Kinds: []string{"read"}},
			"zulu":  {Kinds: []string{"read"}},
		},
	}
	rows := supply.Table(KindProcedure, parse(t, `kind: procedure
procedure: out-of-order
targets: [zulu, alpha]
steps:
  - id: one
    definition: zulu
    operation: op
    target: zulu
  - id: two
    definition: zulu
    operation: op
    target: alpha
  - id: three
    definition: alpha
    operation: op
    target: zulu
  - id: four
    definition: alpha
    operation: op
    target: alpha
`)).Rows

	want := [][2]string{{"alpha", "alpha"}, {"zulu", "alpha"}, {"alpha", "zulu"}, {"zulu", "zulu"}}
	if got := pairsOf(rows); !slices.Equal(got, want) {
		t.Errorf("rows = %v, want %v — (Target, Definition), each by code point", got, want)
	}
}

// TestAuthorityRows_AManifestAndARepositoryDeclarationRenderNoTable is the
// absence that is not an empty state: neither artefact is a member of any pair,
// so there is no end to read the relation from (§8, ADR-0069).
func TestAuthorityRows_AManifestAndARepositoryDeclarationRenderNoTable(t *testing.T) {
	for _, kind := range []string{KindProvider, KindRepositoryDeclaration} {
		if demoAuthority(t).Table(kind, parse(t, cloudflareProd)).Renders {
			t.Errorf("the table rendered on a %s; it is a member of no pair", kind)
		}
	}
}

// TestAuthorityRows_AnAbsentTargetDeclarationEmptiesTwoCellsNotTheRow is §8's
// own rule: dropping the row would say the Target was never claimed, and the
// Definition's own claims are supplied whatever the far end did (ADR-0026).
func TestAuthorityRows_AnAbsentTargetDeclarationEmptiesTwoCellsNotTheRow(t *testing.T) {
	rows := demoAuthority(t).Table(KindDefinition, parse(t, `kind: definition
definition: preview-dns
provider: cloudflare-dns
kinds: [mutate]
destroy: [delete_dns_record]
targets: [cloudflare-prod, nowhere]
`)).Rows

	want := [][2]string{{"preview-dns", "cloudflare-prod"}, {"preview-dns", "nowhere"}}
	if got := pairsOf(rows); !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v — a claim that resolves to nothing keeps its row", got, want)
	}
	absent := rows[1]
	if absent.TargetKinds != nil || absent.Effective != nil {
		t.Errorf("target kinds = %v and effective = %v, want neither supplied", absent.TargetKinds, absent.Effective)
	}
	if got, want := absent.DefinitionKinds, []string{"mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("definition kinds = %v, want %v — this artefact's own, whatever the far end did", got, want)
	}
	if got, want := absent.DestroyOperations, []string{"delete_dns_record"}; !slices.Equal(got, want) {
		t.Errorf("destroy operations = %v, want %v — this artefact's own", got, want)
	}
}

// TestAuthorityRows_AProcedureStepNamingNoDefinitionEmptiesTheOtherHalf is the
// same rule read from the end a Procedure supplies: the pairing is authored on
// the Step and renders, and the cells with no supply carry the absence.
func TestAuthorityRows_AProcedureStepNamingNoDefinitionEmptiesTheOtherHalf(t *testing.T) {
	rows := demoAuthority(t).Table(KindProcedure, parse(t, `kind: procedure
procedure: half-written
targets: [cloudflare-prod]
steps:
  - id: one
    definition: nowhere
    operation: op
    target: cloudflare-prod
`)).Rows

	if got, want := pairsOf(rows), [][2]string{{"nowhere", "cloudflare-prod"}}; !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	row := rows[0]
	if row.DefinitionKinds != nil || row.DestroyOperations != nil || row.Effective != nil {
		t.Errorf("row = %+v, want the Definition's three cells unsupplied", row)
	}
	if got, want := row.TargetKinds, []string{"read", "mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("target kinds = %v, want %v — the declaration's own, whatever the far end did", got, want)
	}
}
