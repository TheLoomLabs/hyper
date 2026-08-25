package artefact

import "gopkg.in/yaml.v3"

// RepositoryFacts is what the Repository declaration says that a command acts
// on, in the shape a command reads it: the retention policy that bounds
// Compaction, and the digest the projection freezes.
//
// The version pin is deliberately not here. It is the gate's, and the gate
// reads `hyper.yaml`'s bytes before the repository is loaded at all — a second
// reading of one fact through a second door is exactly what §12's opening rule
// closes (§9, §11, ADR-0020). The digest beside it is not the gate's and never
// was: `hyper` never hashes itself, and what the digest covers is the released
// artefact a runner fetches — a fact about the world outside this checkout,
// read here like any other thing the declaration says (§11, ADR-0020).
//
// It is the third of these readers and stands beside ReadTargetFacts and
// ReadManifestFacts for their reason: a check asks *is this value well formed*,
// which is a schema question, and a command asks *what did the repository
// declare*, which is this. Neither judges — a `retention:` that is not a
// duration states what it states here and earns its `schema-mismatch` from
// `check` (ADR-0064).
type RepositoryFacts struct {
	// Retention is the policy as the artefact spells it, `90d`, and empty
	// where the declaration carries no `retention:` at all.
	//
	// The empty string is the whole of *this repository has agreed to lose
	// nothing*, and it is safe as a sentinel because no duration is empty:
	// §3 admits `<count><unit>` and nothing else, so a key written with no
	// value is a key that declared no policy. That is also the only reading
	// that is safe to get wrong in one direction — a policy read as absent
	// removes nothing (§3, §7).
	Retention string
	// Digest is the released artefact's checksum as the declaration spells
	// it, algorithm inline — `sha256:` and sixty-four hex digits — and
	// empty where the declaration carries none.
	//
	// It is written by `project` and read by `project`: the version pins
	// which release the generated workflow installs and this pins the bytes
	// that release is, literal in the file so that nothing is resolved when
	// the job runs (§11). The schema makes it required, so empty here is a
	// declaration `check` has already refused rather than a state a
	// projection has to have an answer for (§3, ADR-0064).
	Digest string
}

// ReadRepositoryFacts reads them off `hyper.yaml`'s own root.
func ReadRepositoryFacts(root *yaml.Node) RepositoryFacts {
	fields := topLevelFields(root, "retention", "digest")
	return RepositoryFacts{
		Retention: scalarValue(fields["retention"]),
		Digest:    scalarValue(fields["digest"]),
	}
}
