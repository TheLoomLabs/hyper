package artefact_test

import (
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// The code facts a review reads a direction off (§8, §12, issue #168): eight
// of §12's nine classes, each read off one artefact's own lines.

// facts reads one artefact's change facts, keyed by the coordinate and key a
// row is paired under — which is the pairing the comparison itself makes, so a
// case asserts against what the two sides are matched by.
func facts(t *testing.T, kind, source string) map[string]artefact.ChangeFact {
	t.Helper()

	root, _, ok := yamlsubset.Parse("artefact.yaml", []byte(source))
	if !ok {
		t.Fatalf("the fixture did not parse")
	}
	read := map[string]artefact.ChangeFact{}
	for _, fact := range artefact.ReadChangeFacts(kind, root) {
		key := fact.Key
		if fact.Step != "" {
			key = fact.Step + " · " + fact.Key
		}
		read[key] = fact
	}
	return read
}

// TestChangeFacts_ADefinitionsThreeClasses is what a Definition supplies: its
// claimed Kinds, the Targets it may bind, and the `destroy` Operations it
// names. All three compare as sets and all three are sorted, which is what
// makes two cells differenceable by eye (§8).
func TestChangeFacts_ADefinitionsThreeClasses(t *testing.T) {
	read := facts(t, artefact.KindDefinition, `kind: definition
definition: preview-dns
provider: cloudflare-dns
kinds: [mutate, read]
destroy: [delete_dns_record]
targets: [cloudflare-prod]
`)
	for key, want := range map[string][]string{
		"kinds":   {"mutate", "read"},
		"destroy": {"delete_dns_record"},
		"targets": {"cloudflare-prod"},
	} {
		fact, read := read[key]
		if !read {
			t.Fatalf("no fact at %s; a Definition supplies three", key)
		}
		if fact.Shape != artefact.FactSet {
			t.Errorf("%s is not read as a set", key)
		}
		if !slices.Equal(fact.Members, want) {
			t.Errorf("%s = %q, want %q", key, fact.Members, want)
		}
	}
}

// TestChangeFacts_AKeyTheArtefactNeverWroteIsAFactWithNoLine is the absence a
// row renders `–`: an artefact is a subject on both sides of a range always, so
// its own keys are compared whether either side wrote them (§8).
func TestChangeFacts_AKeyTheArtefactNeverWroteIsAFactWithNoLine(t *testing.T) {
	read := facts(t, artefact.KindDefinition, `kind: definition
definition: preview-dns
provider: cloudflare-dns
kinds: [read]
`)
	fact, held := read["destroy"]
	if !held {
		t.Fatalf("a Definition writing no destroy: supplies no fact for it; the key is compared either way")
	}
	if fact.Written() || len(fact.Members) != 0 {
		t.Errorf("the absent key reads as %+v, want a fact with no line and no members", fact)
	}
}

// TestChangeFacts_AReorderedSetIsTheSameFact is §12's own argument arriving at
// the surface it was made about: **a fact that did not move emits no row,
// however its bytes moved**. Reordering `targets: [staging, local]` changes the
// file and changes nothing this reports.
func TestChangeFacts_AReorderedSetIsTheSameFact(t *testing.T) {
	was := facts(t, artefact.KindTargetDeclaration, `kind: target-declaration
target: staging
kinds: [read, mutate, destroy]
`)
	is := facts(t, artefact.KindTargetDeclaration, `kind: target-declaration
target: staging
kinds: [destroy, read, mutate]
`)
	if !was["kinds"].Same(is["kinds"]) {
		t.Errorf("a reordered set reads as a different fact: %q against %q", was["kinds"].Members, is["kinds"].Members)
	}
}

// TestChangeFacts_AValuesSelectorsOrderIsTheFact is the other half of that
// rule, and the one place it inverts: §6 orders an Expansion by the artefact
// where the selector is a literal list, so a reordering moves which member a
// Run reaches first and sorting it would hide the whole of what changed (§8).
func TestChangeFacts_AValuesSelectorsOrderIsTheFact(t *testing.T) {
	selector := func(first, second string) artefact.ChangeFact {
		return facts(t, artefact.KindProcedure, `kind: procedure
procedure: sync
steps:
  - id: issue
    definition: keys
    operation: create_key
    target: prod
    over:
      values:
        - `+first+`
        - `+second+`
`)["issue · over"]
	}
	was, is := selector("ci-arm64", "ci-x86"), selector("ci-x86", "ci-arm64")
	if was.Value != "values" {
		t.Fatalf("the selector's form reads %q, want the form heading the cell", was.Value)
	}
	if was.Same(is) {
		t.Errorf("a reordered values: selector reads as the same fact; its order is the fact (§6, §8)")
	}
}

// TestChangeFacts_APredicateSelectorRendersOneConjunctPerLineSorted is the
// notation §8 fixes for both surfaces: `field operator operand`, colons
// dropped, sorted by Unicode code point on the rendered line — a predicate list
// being always AND, so conjunct order carries no meaning.
func TestChangeFacts_APredicateSelectorRendersOneConjunctPerLineSorted(t *testing.T) {
	fact := facts(t, artefact.KindProcedure, `kind: procedure
procedure: retire
steps:
  - id: retire
    definition: hetzner
    operation: delete_server
    target: staging
    over:
      assets:
        - field: expires
          older_than: 0s
        - field: labels.role
          equals: preview
        - field: retired
          absent: true
    bound: 5
`)["retire · over"]

	if fact.Value != "assets" {
		t.Errorf("the form reads %q, want assets — a cell dropping it could not tell an assets selector from an observations one", fact.Value)
	}
	want := []string{"expires older_than 0s", "labels.role equals preview", "retired absent"}
	if !slices.Equal(fact.Members, want) {
		t.Errorf("the conjuncts read %q, want %q", fact.Members, want)
	}
}

// TestChangeFacts_AManifestSuppliesItsCapabilitiesAndTheOperationsItExposes
// holds the Operation set to the keys and nothing beneath them: what moved when
// a request changed is the digest a Run records, which is `the digests` and has
// no line in any artefact (§12).
func TestChangeFacts_AManifestSuppliesItsCapabilitiesAndTheOperationsItExposes(t *testing.T) {
	read := facts(t, artefact.KindProvider, `kind: provider
provider: tailscale
schema-version: 1
class: tailscale
capabilities: [http]
operations:
  list_keys:
    kind: read
    deadline: 10s
  delete_key:
    kind: destroy
    deadline: 10s
`)
	if want := []string{"http"}; !slices.Equal(read["capabilities"].Members, want) {
		t.Errorf("capabilities = %q, want %q", read["capabilities"].Members, want)
	}
	if want := []string{"delete_key", "list_keys"}; !slices.Equal(read["operations"].Members, want) {
		t.Errorf("operations = %q, want %q", read["operations"].Members, want)
	}
}

// TestChangeFacts_ACredentialSourceIsOneFactPerSlotNamedInsideItsKey is §12's
// own pairing: the class's pair is a **slot** rather than a Target declaration,
// so a declaration carrying two of them supplies two facts — and the slot is
// named inside the key, a slot being a coordinate the wire carries no member
// for.
func TestChangeFacts_ACredentialSourceIsOneFactPerSlotNamedInsideItsKey(t *testing.T) {
	read := facts(t, artefact.KindTargetDeclaration, `kind: target-declaration
target: prod
kinds: [read]
auth:
  token: {env: TAILSCALE_PROD}
  fallback: {env: TAILSCALE_SPARE}
`)
	for key, want := range map[string]string{
		"credential token":    "TAILSCALE_PROD",
		"credential fallback": "TAILSCALE_SPARE",
	} {
		fact, held := read[key]
		if !held {
			t.Fatalf("no fact at %q; each slot is a pair of its own (§12)", key)
		}
		if fact.Value != want {
			t.Errorf("%s = %q, want the variable it names %q", key, fact.Value, want)
		}
		if fact.Step != "" {
			t.Errorf("%s carries the coordinate %q; only a Procedure's Step has one (§12)", key, fact.Step)
		}
	}
}

// TestChangeFacts_ARepositoryDeclarationSuppliesNone is the enumeration holding
// rather than a roster left short: its `version:` is `the digests`'
// `hyper_version` and its `retention:` is one of the lines §12's catch-all
// counts, and neither is a fact this vocabulary names.
func TestChangeFacts_ARepositoryDeclarationSuppliesNone(t *testing.T) {
	read := facts(t, artefact.KindRepositoryDeclaration, `kind: repository-declaration
version: 1.4.0
retention: 90d
`)
	if len(read) != 0 {
		t.Errorf("a Repository declaration supplies %d facts, want none", len(read))
	}
}

// TestChangeFacts_ANestedInvocationSuppliesNoneOfTheThree is what a Step's
// three classes are a Step's: an invocation binds nothing, expands nothing and
// bounds nothing, and the envelope it reaches is the invoked Procedure's own
// fact on that Procedure's own review (§3, §8).
func TestChangeFacts_ANestedInvocationSuppliesNoneOfTheThree(t *testing.T) {
	read := facts(t, artefact.KindProcedure, `kind: procedure
procedure: watch
targets: [staging]
steps:
  - id: descend
    procedure: probe
`)
	for _, key := range []string{"descend · target", "descend · over", "descend · bound"} {
		if _, held := read[key]; held {
			t.Errorf("a nested invocation supplies %s", key)
		}
	}
	if _, held := read["targets"]; !held {
		t.Errorf("the Procedure's own declared envelope went with it")
	}
}

// TestChangeFacts_WireIsTheArtefactsOwnParsedShape is what §8's row stream
// carries for a value that is not a scalar (issue #171).
//
// **A `from` or `to` that is not a scalar carries the artefact's own parsed
// shape** — `{"values":[…]}`, `{"assets":[{"field":…,"equals":…}]}`,
// `["read","mutate"]` — in the order the page renders it. The page's notation
// is that chapter's geometry and never a fact either surface states
// (ADR-0059), so nothing composed goes out composed: a ` · `-separated run and
// a `field operator operand` line are renderings, and a reader composing the
// wire out of them could not tell an `in:` list from a bare operand.
func TestChangeFacts_WireIsTheArtefactsOwnParsedShape(t *testing.T) {
	read := facts(t, artefact.KindProcedure, `kind: procedure
procedure: sync
targets: [staging, local]
cadence: "0 3 * * 1"
steps:
  - id: issue
    bound: 3
    over:
      values: [ci-x86, ci-arm64]
  - id: retire
    over:
      assets:
        - field: labels.role
          in: [preview, scratch]
        - field: created
          older_than: 30d
`)
	for key, want := range map[string]string{
		// A set carries the sorted, deduplicated run the page renders:
		// a set compares by set equality, so the order the author
		// happened to write it in is not a fact.
		"targets": `["local","staging"]`,
		// A Cadence is a scalar under the shape rule; what stacks its
		// cell is the mandatory gloss, which rides beside it as parts.
		"cadence": `"0 3 * * 1"`,
		// A Bound is a number because the author wrote an integer.
		// `"3"` would be this surface re-typing a value it was handed.
		"issue · bound": `3`,
		// A `values:` selector is **as authored**: its order is the
		// fact, so sorting it would hide the whole of what changed.
		"issue · over": `{"values":["ci-x86","ci-arm64"]}`,
		// A predicate selector's conjuncts are in the order the page
		// renders them — sorted on the rendered line — and each one
		// keeps the keys the author wrote, in the order they wrote
		// them, an `in:` list included.
		"retire · over": `{"assets":[{"field":"created","older_than":"30d"},{"field":"labels.role","in":["preview","scratch"]}]}`,
	} {
		if got := string(read[key].Wire); got != want {
			t.Errorf("%s carries %s on the wire, want %s", key, got, want)
		}
	}
}

// TestChangeFacts_AFactStatedByOmissionCarriesNoWire is the absence rule on the
// wire: the key is absent where the page renders `–`, rather than written as
// null — which is `from_ordinal`'s rule two row types up (§7, §8).
//
// For a set-shaped fact an absent key and an empty list are one value, so a
// `destroy: []` carries no wire either.
func TestChangeFacts_AFactStatedByOmissionCarriesNoWire(t *testing.T) {
	read := facts(t, artefact.KindDefinition, `kind: definition
definition: ci-keys
kinds: [mutate]
destroy: []
`)
	for _, key := range []string{"destroy", "targets"} {
		if wire := read[key].Wire; wire != nil {
			t.Errorf("%s carries %s on the wire, want no member at all", key, wire)
		}
	}
}

// TestChangeFacts_TheWireDoesNotEscapeHTML is the stream's own rule holding
// here: the wire carries an artefact's own bytes, and a predicate operand
// quoting a `&` or a `<` is one a consumer reads back as it was written (§8,
// render.WriteJSON).
func TestChangeFacts_TheWireDoesNotEscapeHTML(t *testing.T) {
	read := facts(t, artefact.KindProcedure, `kind: procedure
procedure: sync
steps:
  - id: retire
    over:
      observations:
        - field: title
          equals: "a & b < c"
`)
	if got, want := string(read["retire · over"].Wire), `{"observations":[{"field":"title","equals":"a & b < c"}]}`; got != want {
		t.Errorf("the selector carries %s on the wire, want %s", got, want)
	}
}
