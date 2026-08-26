package artefact

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// CodeOriginDigestMismatch is the code an installed Manifest earns when the
// bytes standing in providers/ are not the bytes `install` verified against the
// digest recorded beside them (§4, §12).
//
// It is spelled here, at the check that recomputes it offline, and read from
// here by `install`, which answers the same code over the same fact at the
// other end of one mechanism: a fetch verifies published bytes once, and this
// is that verification made repeatable by anyone reading the repository, long
// after the machine that performed it is gone (§11). One string rather than two
// that happen to agree.
const CodeOriginDigestMismatch = "origin-digest-mismatch"

// originKeyPrefix is how the block `install` appends begins: the key at column
// 0, which is what a line carrying it starts with and what no line nested
// inside anything can start with. It is the one byte pattern this file scans
// for, and it is a scan rather than a read off the parse tree because what is
// being located is a **byte offset** — where the published half of the file
// ends — and a parse tree holds line numbers over a document it has already
// normalised away from.
const originKeyPrefix = "origin:"

// checkOriginDigest recomputes the digest an installed Manifest records, over
// the bytes it covers, and reports origin-digest-mismatch where it no longer
// holds. It is the verification half of `install`: §11 says the origin: block
// is what makes the fetch's verification repeatable offline, by anyone reading
// the repository, and this is where that claim stops being prose.
//
// **The byte range is the one `install` fixed.** The digest covers the
// published bytes — the file without the block naming them, a digest being
// unable to cover itself — and `install` appends that block as the last thing
// in the file, so the range is the file up to the start of the last line
// beginning `origin:` at column 0 (§11, ADR-0087). The last rather than the
// first: a published Manifest carrying an origin: block of its own acquires a
// second the moment `install` appends the one it verified, and the block naming
// the bytes is the one written after them.
//
// **The comparison is against two candidates, and that is exactness rather than
// laxity.** The prefix, and the prefix less one trailing newline. They differ
// only by the byte `install` itself may have written where a published Manifest
// did not end in one, so a publisher who omitted a trailing newline is not
// punished for it. Two candidates rather than a normalisation, because
// normalising means the digest covers a canonical form rather than the bytes —
// which is what ManifestDigest already refuses to do, and for the same reason: a
// second digest of one Manifest is a second representation that can disagree
// with the one `install` verified.
//
// **Three files reach it never, and each for its own reason.** A Manifest
// carrying no block is a locally authored Provider, checked like any other and
// making no digest claim — which is what dropping the block buys an author who
// deliberately modified an installed Manifest, and the one visible, readable
// edit §11 gives them (ADR-0073). A built-in carries no block and makes no
// claim against a registry, and reaches this pass never: CheckManifest is the
// providers/ file's checks, and the built-in runs the body they layer over
// (§11, ADR-0039). And a Manifest whose block is malformed has already earned
// schema-mismatch from a schema that requires both members where the block is
// present; a second opinion here would put two rows on the page for one fault
// (§4, ADR-0064).
//
// **The remediation is an edit and renders as one.** Unlike projection-stale,
// the repair is a file in the tree the reader may act on — re-install it, or
// drop the block — so the Refusal takes §8's ordinary EDIT ONE OF shape and the
// code earns no entry in the remedy table (§8, internal/cli/refusal.go).
func checkOriginDigest(file string, root *yaml.Node, manifest []byte) []problem.Problem {
	digest := recordedDigest(root)
	if digest == nil {
		return nil
	}
	published, found := publishedBytes(manifest)
	if !found {
		return nil
	}

	recomputed := ManifestDigest(published)
	if digest.Value == recomputed {
		return nil
	}
	if trimmed, ended := bytes.CutSuffix(published, []byte("\n")); ended && digest.Value == ManifestDigest(trimmed) {
		return nil
	}

	return []problem.Problem{{
		File: file, Line: digest.Line, Column: digest.Column, Field: "origin.digest",
		ErrorCode: CodeOriginDigestMismatch,
		Message: fmt.Sprintf("origin: digest: this Manifest's published bytes are %s and the block records %s"+
			" — an edited Manifest is re-installed or has its origin: block dropped, never a digest retyped by hand",
			recomputed, digest.Value),
	}}
}

// recordedDigest is the digest: scalar of a legible origin: block, and nil
// wherever there is no claim to recompute: no block at all, a block that is not
// a mapping, or one whose two members are not both plain scalars. The last of
// those is the schema's row and not this check's, which is why what is returned
// is the node rather than the string — a check that reported a fault would have
// to decide which of the two spellings was wrong, and the schema has already
// said (§4, ADR-0064).
func recordedDigest(root *yaml.Node) *yaml.Node {
	originVal := topLevelFields(root, "origin")["origin"]
	if originVal == nil || originVal.Kind != yaml.MappingNode {
		return nil
	}
	origin := topLevelFields(originVal, "ref", "digest")
	ref, digest := origin["ref"], origin["digest"]
	if ref == nil || ref.Kind != yaml.ScalarNode || digest == nil || digest.Kind != yaml.ScalarNode {
		return nil
	}
	return digest
}

// publishedBytes is the half of the file the recorded digest covers: everything
// before the start of the last line beginning `origin:` at column 0.
//
// It answers false where the file holds no such line, which is a Manifest whose
// block is not in the position `install` writes it — a whole-document flow
// mapping being the one spelling that reaches here. There is no prefix to take
// and therefore nothing to recompute: what such a file carries is a claim
// `hyper` never wrote, and §11's mechanism against one is the diff a human
// reads.
//
// The scan is over line starts rather than a search for the pattern anywhere,
// so a `origin:` written inside a scalar or indented under another key is not
// mistaken for the block: column 0 is what the block is written at and what
// nothing nested can be written at.
func publishedBytes(manifest []byte) ([]byte, bool) {
	key := []byte(originKeyPrefix)
	end, found := 0, false
	for at := 0; at < len(manifest); {
		if bytes.HasPrefix(manifest[at:], key) {
			end, found = at, true
		}
		next := bytes.IndexByte(manifest[at:], '\n')
		if next < 0 {
			break
		}
		at += next + 1
	}
	if !found {
		return nil, false
	}
	return manifest[:end], true
}
