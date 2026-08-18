package artefact

import (
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// SecretMarker is the one constant a credential's position is marked with,
// wherever `hyper` writes a position a secret occupies (§7). §7 fixes two such
// positions: a Record field a Manifest declares secret, which arrives with the
// Store, and the Auth scheme a surface rendering a Provider composes, which is
// what reaches it here. One constant rather than one per position is what §7
// requires — it is what keeps the Store's byte comparison honest, a rotated
// secret writing identical bytes — and it is why nothing here needs a
// credential in hand: the marker stands in the position, and the position is
// the scheme's (ADR-0007, ADR-0031).
const SecretMarker = "<secret>"

// authSchemeNone is what a Provider carrying no auth: block composes, and §12
// fixes the word: an undeclared default rendered in words rather than an empty
// cell, because a Provider naming no scheme sends no credential — a fact about
// it, and not a gap in the row. It is unexported because every surface reads it
// through the composition below; the day a second surface needs the word
// itself is the day it is exported.
const authSchemeNone = "none"

// ManifestFacts is what a Manifest declares about itself, in the shape a
// surface reports it — `hyper provider`'s header row, which exists to state
// what a Manifest declares (§9).
//
// It is ReadTargetFacts's counterpart and stands beside ProviderInfo for
// ProviderInfo's own reason: one artefact read for two questions. A check asks
// *may this Definition bind that Provider*, which is a set of memberships; a
// row states *what does this Manifest declare*, which is an enumeration and the
// composition an Auth scheme writes. Neither reading judges the artefact —
// a Manifest that declares a Capability outside the closed set states what it
// states here and earns its problem from check (ADR-0064).
//
// AuthScheme is the header the Manifest's scheme composes, with the
// credential's position marked; the word §12 fixes for a Manifest carrying no
// auth: block, and "" where it carries one `hyper` cannot read as a scheme —
// which is a schema fault check names, and not an absence of authentication a
// row may report as one.
//
// SchemaVersion is a pointer because the Manifest's own schema-version: is an
// integer and a Manifest that wrote something else has stated no version at
// all: 0 would be an answer to a question nothing asked, and the ordinary
// absence rule is what a reader reads off its absence (§7).
//
// OriginRef and OriginDigest are the origin: block's two members, both empty
// where the block is absent — a built-in Provider and a locally authored
// Extension alike, which together are the whole of what distinguishes an
// installed Extension from one an author wrote (ADR-0073).
//
// Operations are in the Manifest's own order, which is the order they were
// authored in. The normative order a listing is ranged over in is the
// surface's rule and is applied there: `hyper operation` writes a Manifest's
// declaring lines back verbatim, so the authored order is preserved exactly
// where it is the answer (§9).
type ManifestFacts struct {
	AuthScheme    string
	Capabilities  []string
	SchemaVersion *int
	OriginRef     string
	OriginDigest  string
	Operations    []OperationFacts
}

// OperationFacts is one Operation as a listing reports it: the name the
// Manifest keys it by, the Kind it declares, whether it is Opaque, and a
// derived summary (§9).
//
// Kind is read from the Manifest's own kind: and never inferred from the name
// — an Operation called delete_thing that declares mutate is a mutate (§12).
// Opaque is not read from the Manifest at all in the sense the other two are:
// opacity is a property of the Capability the Operation's request uses, so it
// is read from the request block, and no artefact anywhere declares it (§12).
type OperationFacts struct {
	Name    string
	Kind    string
	Opaque  bool
	Summary string
}

// ReadManifestFacts reads those facts off a Manifest's own root. Like
// ReadTargetFacts it judges nothing and drops what it cannot read: a member
// this returns empty is a member the Manifest did not legibly state, which is
// check's to report and never this reader's to guess at (ADR-0064).
//
// The digest is not here: it is SHA-256 over a Manifest's exact bytes and not
// over what they parsed to (§7), so it is ManifestDigest's and is taken over
// the bytes the load kept.
func ReadManifestFacts(root *yaml.Node) ManifestFacts {
	fields := topLevelFields(root, "capabilities", "schema-version", "origin", "operations")
	facts := ManifestFacts{
		AuthScheme:    authScheme(root),
		Capabilities:  scalarSequence(fields["capabilities"]),
		SchemaVersion: integerScalar(fields["schema-version"]),
		Operations:    operationFacts(fields["operations"]),
	}
	if originVal := fields["origin"]; originVal != nil && originVal.Kind == yaml.MappingNode {
		origin := topLevelFields(originVal, "ref", "digest")
		facts.OriginRef = scalarValue(origin["ref"])
		facts.OriginDigest = scalarValue(origin["digest"])
	}
	return facts
}

// authScheme composes the header the Manifest's Auth scheme writes, with the
// credential's position marked by SecretMarker — the composition rather than
// the parameters, because a prefix: is concatenated verbatim and the
// load-bearing trailing space in "Bearer " is invisible in the source. A
// Provider that omitted it composes `Authorization: Bearer<secret>` and reads
// that way here, the failure made legible rather than guessed at by a check
// (§9, §12).
//
// The position is read through authOwnedHeaderName, which is the position
// check reserves against an ordinary headers: entry (manifest-inconsistent,
// §4). One read rather than two that agree: the header a row renders the
// credential into and the header check refuses a second writer for are the
// same position, and a second walk here is where the day comes that they name
// different ones.
//
// It returns authSchemeNone where there is no auth: block — a Provider naming
// no scheme sends no credential — and "" where there is a block whose scheme
// cannot be read. The three answers are three different facts and the middle
// one is the one to read carefully: none says no credential is sent, which is a
// claim about the Provider, and `hyper` cannot make that claim about a Manifest
// that declared a scheme it could not parse. That is schema-mismatch, which
// check reports, and the row states nothing in its place (§7, ADR-0064).
//
// Nothing here resolves a credential or reaches a network: the marker stands in
// the position, and the position is the scheme's (ADR-0007).
func authScheme(root *yaml.Node) string {
	if topLevelFields(root, "auth")["auth"] == nil {
		return authSchemeNone
	}
	position := authOwnedHeaderName(root)
	if position == "" {
		return ""
	}
	return position + ": " + schemePrefix(root) + SecretMarker
}

// schemePrefix is what the scheme concatenates verbatim in front of the
// credential: header:'s own prefix:, absent meaning empty, and basic:'s fixed
// "Basic ", which is the scheme's own and takes no parameter a Manifest could
// supply (§3, §12).
//
// It is reached only where authOwnedHeaderName named a position, which is what
// makes header: the branch to test for: a Manifest whose auth: is not a mapping,
// or carries neither scheme, has already answered "" above.
func schemePrefix(root *yaml.Node) string {
	authVal := topLevelFields(root, "auth")["auth"]
	if headerVal := topLevelFields(authVal, "header")["header"]; headerVal != nil {
		return scalarValue(topLevelFields(headerVal, "prefix")["prefix"])
	}
	return "Basic "
}

// operationFacts reads one entry per member of operations:, in the mapping's
// own order. An entry whose key is not a plain scalar has no name to report
// and contributes nothing; everything else about it is read the way every
// check reads it — through operationInfoFromNode, so the Kind a row states and
// the Kind a Step is checked against are one read of one artefact rather than
// two that agree (§4, §9).
func operationFacts(opsVal *yaml.Node) []OperationFacts {
	if opsVal == nil || opsVal.Kind != yaml.MappingNode {
		return nil
	}

	var facts []OperationFacts
	for i := 0; i+1 < len(opsVal.Content); i += 2 {
		key, op := opsVal.Content[i], opsVal.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		info := operationInfoFromNode(op)
		facts = append(facts, OperationFacts{
			Name:    key.Value,
			Kind:    info.Kind,
			Opaque:  info.IsShell,
			Summary: operationSummary(op, info),
		})
	}
	return facts
}

// operationSummary is the derived summary an Operation row carries (§9): the
// request it makes, the Repeatability in force, and what it projects. Its
// shape is ProviderInfo.Summary's — clauses joined by "; ", each derived from
// what the Manifest states and none of them a key an author could write.
//
// The three are chosen for what they answer that the row's other members do
// not. The request is where an agent reads what the Operation does before
// opening its input schema. The Repeatability in force is the one fact here
// with no spelling in the source at all: run-once is what an effectful
// Operation declaring no repeatability: is, and §12 gives it no keyword to
// author, so a reader who scanned the Manifest for it would find nothing
// (issue #96). And what an Operation projects is the difference between a
// Record, a series of them, and none — which is what a caller reads before
// writing a selector against it.
//
// A fact the Manifest never stated is dropped rather than rendered empty,
// exactly as Summary drops one: an Operation that declares no request has no
// request to name, and one whose kind: is not one of the three has no default
// Repeatability to derive (§12, ADR-0064). What it projects is always stated,
// an Operation projecting nothing saying so in words.
func operationSummary(op *yaml.Node, info OperationInfo) string {
	var clauses []string
	if request := operationRequest(op, info); request != "" {
		clauses = append(clauses, request)
	}
	if repeatability := effectiveRepeatability(info); repeatability != "" {
		clauses = append(clauses, repeatability)
	}
	return strings.Join(append(clauses, projection(info)), "; ")
}

// operationRequest names the request the Operation makes: its method and path
// where that request is http, and the Capability's own name where it is shell,
// an Opaque request having no shape `hyper` can describe and no keys to read
// (§3, §12). It is "" where neither block is legible.
func operationRequest(op *yaml.Node, info OperationInfo) string {
	if info.IsShell {
		return "shell"
	}
	httpVal := topLevelFields(op, "http")["http"]
	if httpVal == nil {
		return ""
	}
	request := topLevelFields(httpVal, "method", "path")
	method, path := scalarValue(request["method"]), scalarValue(request["path"])
	if method == "" || path == "" {
		return ""
	}
	return method + " " + path
}

// effectiveRepeatability is the Repeatability in force: the one the Operation
// declares, or the default its Kind fixes — repeatable on a read, run-once on
// a mutate or a destroy (§12). It is "" where kind: is not one of the three,
// there being no default to read off a Kind the Manifest did not state.
func effectiveRepeatability(info OperationInfo) string {
	if info.Repeatability != "" {
		return info.Repeatability
	}
	if !slices.Contains(kindsByBlastRadius, info.Kind) {
		return ""
	}
	if info.IsRunOnce() {
		return "run-once"
	}
	return "repeatable"
}

// projection is what the Operation writes to the Store: one Record, a series
// of them where its record: carries an over:, or none at all — which is what a
// destroy projects, having no record: block by construction (§3, §4).
func projection(info OperationInfo) string {
	switch {
	case info.RecordFields == nil:
		return "projects no Record"
	case info.HasSeries:
		return "projects a Record series"
	default:
		return "projects one Record"
	}
}

// scalarValue is a node's own text where it is a plain scalar, and "" where it
// is absent or is anything else — the one reading every fact above is dropped
// by, on ReadTargetFacts's rule that what cannot be read has no value to
// report.
func scalarValue(val *yaml.Node) string {
	if val == nil || val.Kind != yaml.ScalarNode {
		return ""
	}
	return val.Value
}

// integerScalar is a scalar read as the integer it is required to be, or nil
// where the key is absent or holds something else. Nil rather than zero
// because zero is a value the member could legitimately carry, and a member
// nothing stated is absent from a row rather than written as its zero (§7).
func integerScalar(val *yaml.Node) *int {
	text := scalarValue(val)
	if text == "" {
		return nil
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return nil
	}
	return &n
}
