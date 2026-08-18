package artefact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// The two members of §12's Provider origin set, and the criterion is where the
// Manifest's bytes load from: inside the binary, or a tracked file in
// providers/ (ADR-0073). It says nothing about whether those bytes were ever
// verified against a registry — a built-in and a locally authored Extension
// both make no such claim, and whether a Manifest claimed an upstream is the
// second, orthogonal fact its origin: block carries.
//
// The spelling is built-in and not builtin. §12's baseline_absent already
// carries built-in on §8's wire for the same fact about the same Manifest, and
// one fact reaching two wires reaches them under one name.
const (
	OriginBuiltIn   = "built-in"
	OriginExtension = "extension"
)

// ProviderOrigin reads a loaded Manifest's origin off the path its bytes came
// from, which is the whole of §12's criterion. The pseudo-path the built-in
// carries is the one that answers built-in — hyper ships a Provider only where
// nobody else could write it, so that set is closed and enumerated with the
// paths (ADR-0039) — and every other Manifest hyper can load is a file in
// providers/, whether it arrived by `hyper install` or was typed there by hand:
// an Extension is a Provider authored by someone other than hyper rather than
// one fetched from a registry.
func ProviderOrigin(path string) string {
	if path == BuiltinShellProviderPath {
		return OriginBuiltIn
	}
	return OriginExtension
}

// ManifestDigest is manifest_digest: SHA-256 over a Manifest's exact bytes,
// with sha256: inline because hyper chose the algorithm (§7). The bytes are the
// file in providers/ for an Extension and the compiled-in constant for the
// built-in, which has no blob in the repository at all.
//
// Over the bytes rather than a canonical form of what they parse to, because a
// second digest of one Manifest is a second representation that can disagree
// with the one `install` verified, and because a reader checks bytes with
// sha256sum and a canonical form only where something has written that form out
// for them. Reformatting a Manifest moves it, and moves every later Record's
// Provenance with it — correct rather than noisy: the reviewed artefact moved.
//
// It is never abbreviated by anything that renders it (§8, ADR-0047): a digest
// is verified with sha256sum rather than recognised by eye, so a shortened one
// is a value the reader has to go somewhere else to complete.
func ManifestDigest(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// kindsByBlastRadius is §12's Kind set in the order §12 states it — ascending
// blast radius rather than alphabetical — which is the order a summary counts
// them in.
var kindsByBlastRadius = []string{"read", "mutate", "destroy"}

// Summary is the derived summary a `providers` row carries (§9). §3's Manifest
// schema has no summary: key and §12 forces additionalProperties: false at
// every level, so no Manifest can carry one and no Manifest author can write
// one; what stands in its place is built from the facts the Manifest does
// state — its class:, the Capabilities it requires, and its Operation count by
// Kind — on §9's own rule that making the caller re-derive what hyper already
// computed is waste.
//
// A fact the Manifest never stated is dropped rather than rendered empty: a
// file that parses and names itself is in the Provider namespace whatever else
// it left out (ADR-0064), and a summary reading "class ; requires " would state
// two declarations that are not there. An Operation count is the one part
// always written, a Manifest declaring none saying so in words.
func (p ProviderInfo) Summary() string {
	var parts []string
	if p.Class != "" {
		parts = append(parts, "class "+p.Class)
	}
	if len(p.Capabilities) > 0 {
		parts = append(parts, "requires "+strings.Join(slices.Sorted(maps.Keys(p.Capabilities)), ", "))
	}
	return strings.Join(append(parts, p.operationCounts()), "; ")
}

// operationCounts is the Operation count by Kind, in §12's order, naming only
// the Kinds this Manifest declares an Operation of: a zero is not a fact anyone
// reads off a summary, and "0 destroy" beside two counts that mean something
// reads as a column rather than a sentence. An Operation whose kind: is absent
// or is not one of the three is counted by none of them — the schema has
// already named that fault, and a summary is not where it is reported.
func (p ProviderInfo) operationCounts() string {
	counts := map[string]int{}
	for _, op := range p.Operations {
		counts[op.Kind]++
	}

	var stated []string
	for _, kind := range kindsByBlastRadius {
		if counts[kind] > 0 {
			stated = append(stated, fmt.Sprintf("%d %s", counts[kind], kind))
		}
	}
	if len(stated) == 0 {
		return "no Operations"
	}
	return strings.Join(stated, ", ")
}
