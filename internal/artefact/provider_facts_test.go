package artefact

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// TestManifestDigest_IsSHA256OverTheExactBytesWithTheAlgorithmInline holds
// §7's two rules about a manifest_digest against a published test vector
// rather than against a value computed the way the code computes it: SHA-256
// over the bytes as they stand, and `sha256:` inline because hyper chose the
// algorithm.
func TestManifestDigest_IsSHA256OverTheExactBytesWithTheAlgorithmInline(t *testing.T) {
	// NIST's own SHA-256 vector for "abc", which is where the expectation
	// comes from: a test recomputing the hash the way ManifestDigest does
	// would agree with it however wrong both were.
	const want = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"

	if got := ManifestDigest([]byte("abc")); got != want {
		t.Errorf("ManifestDigest = %q, want %q", got, want)
	}
}

// TestManifestDigest_CoversTheBytesAndNotWhatTheyParseTo is the reason §7
// states the digest over bytes: two Manifests that parse to one document have
// two digests, and reformatting one moves every later Record's Provenance
// with it.
func TestManifestDigest_CoversTheBytesAndNotWhatTheyParseTo(t *testing.T) {
	compact := ManifestDigest([]byte("kind: provider\nprovider: widget\n"))
	spaced := ManifestDigest([]byte("kind:     provider\nprovider: widget\n"))

	if compact == spaced {
		t.Errorf("two spellings of one document share the digest %q; the digest covers the bytes", compact)
	}
}

// TestProviderOrigin_IsWhereTheBytesLoadFromAndIsSpelledWithTheHyphen is
// §12's closed two-member set and ADR-0073's criterion: inside the binary, or
// a tracked file in providers/. Nothing about an upstream claim reaches it.
func TestProviderOrigin_IsWhereTheBytesLoadFromAndIsSpelledWithTheHyphen(t *testing.T) {
	if got := ProviderOrigin(BuiltinShellProviderPath); got != "built-in" {
		t.Errorf("the built-in Provider's origin = %q, want %q", got, "built-in")
	}
	if got := ProviderOrigin("providers/cloudflare-dns.yaml"); got != "extension" {
		t.Errorf("a providers/ file's origin = %q, want %q", got, "extension")
	}
	if OriginBuiltIn != "built-in" || OriginExtension != "extension" {
		t.Errorf("the set is spelled %q and %q; §12 fixes built-in and extension", OriginBuiltIn, OriginExtension)
	}
}

// TestProviderSummary_IsDerivedFromTheManifestsOwnFacts is §9's derived
// summary: a Manifest can carry no summary: key at all (§3, §12), so the row's
// summary is built from the facts the Manifest does state — its class:, the
// Capability it requires, and its Operation count by Kind.
func TestProviderSummary_IsDerivedFromTheManifestsOwnFacts(t *testing.T) {
	root := parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    http: {method: GET, host: "{from-target}", path: /widgets}
  create_widget:
    kind: mutate
    http: {method: POST, host: "{from-target}", path: /widgets}
  delete_widget:
    kind: destroy
    http: {method: DELETE, host: "{from-target}", path: "/widgets/{id}"}
  purge_widgets:
    kind: destroy
    http: {method: DELETE, host: "{from-target}", path: /widgets}
`)

	const want = "class widgetco; requires http; 1 read, 1 mutate, 2 destroy"
	if got := providerInfoFromManifest(root).Summary(); got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
}

// TestProviderSummary_CountsByKindInSectionTwelvesOrderAndOmitsTheKindsAbsent
// keeps the Kinds in the order §12 states the set — read, mutate, destroy,
// ascending blast radius — and says nothing about a Kind the Manifest declares
// no Operation of, a zero being a fact nobody reads off a summary.
func TestProviderSummary_CountsByKindInSectionTwelvesOrderAndOmitsTheKindsAbsent(t *testing.T) {
	root := parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    http: {method: GET, host: "{from-target}", path: /widgets}
`)

	const want = "class widgetco; requires http; 1 read"
	if got := providerInfoFromManifest(root).Summary(); got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
}

// TestProviderSummary_SaysSoWhereTheManifestDeclaresNoOperation is the row a
// Manifest with an empty operations: mapping earns: operation_count is 0 and
// the summary says that in words rather than trailing off after the Capability.
func TestProviderSummary_SaysSoWhereTheManifestDeclaresNoOperation(t *testing.T) {
	root := parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations: {}
`)

	const want = "class widgetco; requires http; no Operations"
	if got := providerInfoFromManifest(root).Summary(); got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
}

// TestProviderSummary_DropsTheFactsAManifestNeverStated is the degenerate
// Manifest a row still has to carry: a file that parses and names itself is in
// the Provider namespace whatever else it left out (ADR-0064), and the summary
// states the facts it has rather than rendering an empty class or an empty
// Capability list as though they were declarations.
func TestProviderSummary_DropsTheFactsAManifestNeverStated(t *testing.T) {
	root := parse(t, "kind: provider\nprovider: widget\n")

	const want = "no Operations"
	if got := providerInfoFromManifest(root).Summary(); got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
}

// TestProviderSummary_ReadsTheBuiltInProvidersOwnCompiledBytes is the one
// Provider with no file: its summary is derived from the compiled-in Manifest
// the same way an Extension's is from the file it loaded from, six Operations
// being Kind crossed with the Repeatability values each Kind may declare (§12).
func TestProviderSummary_ReadsTheBuiltInProvidersOwnCompiledBytes(t *testing.T) {
	const want = "class local; requires shell; 1 read, 3 mutate, 2 destroy"
	if got := builtinShellProviderInfo().Summary(); got != want {
		t.Errorf("the built-in shell Provider's summary = %q, want %q", got, want)
	}
}

// TestProviderSummary_NamesEveryCapabilityAManifestRequires is the second
// member of §12's Capability set arriving on a row: a Manifest requiring two
// names both, in an order that does not depend on a map's iteration.
func TestProviderSummary_NamesEveryCapabilityAManifestRequires(t *testing.T) {
	info := ProviderInfo{
		Class:        "widgetco",
		Capabilities: map[string]bool{"shell": true, "http": true},
		Operations:   map[string]OperationInfo{"list_widgets": {Kind: "read"}},
	}

	const want = "class widgetco; requires http, shell; 1 read"
	for i := range 8 {
		if got := info.Summary(); got != want {
			t.Fatalf("Summary = %q on call %d, want %q", got, i, want)
		}
	}
}

// TestManifest_GainsNoSummaryKey is the other half of the derived summary: §3's
// Manifest schema has no summary: and §12 forces additionalProperties: false at
// every level, so a Manifest author who writes one earns unknown-key like any
// other key the schema does not define. What `providers` renders is derived,
// and there is nowhere to author it instead.
func TestManifest_GainsNoSummaryKey(t *testing.T) {
	root := parse(t, `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
summary: a Provider for widgets
operations:
  list_widgets:
    kind: read
    http: {method: GET, host: "{from-target}", path: /widgets}
    record: {identity: $.id, fields: {id: $.id}}
`)

	problems := CheckManifest("providers/widget.yaml", root)
	for _, p := range problems {
		if p.Field == "summary" && p.ErrorCode == schema.CodeUnknownKey {
			return
		}
	}
	t.Errorf("a Manifest carrying summary: earned %v, want %s on summary", problems, schema.CodeUnknownKey)
}
