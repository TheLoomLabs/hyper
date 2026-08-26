package artefact

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// ManifestSchemaVersion is the highest Manifest schema version this binary
// reads, and the whole of the mechanism §11 states: `hyper` reads any Manifest
// at or below it and Refuses on one written above it (ADR-0028).
//
// **One today, and reading *down* is built by not building anything.** There is
// one version, which is what the built-in declares and what every Manifest §3
// illustrates carries, so there is no older shape to migrate from — and the
// migration path for a second is written when a second exists rather than
// guessed at now.
//
// It is a compiled-in constant because that is what the fact is: *which shapes
// this binary knows* is answerable before any tree is walked, on
// IsBuiltinProviderName's own footing. And it is **one** constant read from one
// place because a ceiling spelled twice is two answers to *may I read this
// file*, one of which would be wrong the day the integer moves.
//
// A Manifest is the one artefact carrying an explicit schema version at all,
// being the one authored outside this repository's own pin (§3, ADR-0023) —
// which is why this ceiling exists here and nowhere else in internal/artefact.
const ManifestSchemaVersion = 1

// CodeManifestSchemaUnsupported is the code a Manifest declaring a schema
// version above ManifestSchemaVersion earns (§4, §12).
//
// It is spelled here, at the check that decides it, on CodeOriginDigestMismatch's
// own footing — and read from here by internal/cli, which names the remedy for
// it. One string rather than two that happen to agree.
//
// **It is not CodeSchemaUnsupported.** That one is an *input* schema reaching
// outside §4's four-keyword subset, one Operation deep inside a Manifest this
// reader understands, and its remedy is an ordinary artefact edit. This one is
// the whole file in a shape this binary does not know, and its remedy is a
// different binary — nothing in the repository is the fault. The two read alike
// and share nothing else, which is why each is spelled out where it is decided
// rather than either reaching for the other.
const CodeManifestSchemaUnsupported = "manifest-schema-unsupported"

// ManifestSchemaUnsupported says whether root declares a schema version above
// the one this binary reads — the predicate **both halves of the rule read**.
//
// The check below names the file and the load declines it: a Manifest above the
// ceiling contributes nothing to the Provider namespace, which is the mechanism
// `provider-name-collision` introduced one rule over (internal/repository's
// manifestsByName, internal/verify's collisionProblems). One predicate rather
// than two readings, for that rule's own reason: a name the fold declined and
// the check said nothing about is a file that vanished from the namespace with
// no row to explain it.
//
// A root carrying no legible schema-version at all is **not** unsupported here.
// What this answers is *is this version above mine*, and a file that declares no
// version has already earned schema-mismatch from a schema that requires one —
// so it is checked like any other Manifest and its faults are its own.
func ManifestSchemaUnsupported(root *yaml.Node) bool {
	_, declared, legible := manifestSchemaVersion(root)
	return legible && declared > ManifestSchemaVersion
}

// manifestSchemaVersion is the schema version root declares: the scalar that
// carries it, the integer it reads as, and whether it reads as one at all.
//
// It answers **not legible** rather than a guess for a key that is absent, that
// is not a plain scalar, or that carries something other than an integer. Each
// of those is a fault ManifestDeclaration's own top-level schema already names
// under schema-mismatch, and a second opinion here would put two rows on one
// line for one cause — checkKind's rule, one key over.
func manifestSchemaVersion(root *yaml.Node) (declaredAt *yaml.Node, version int, legible bool) {
	val := topLevelFields(root, "schema-version")["schema-version"]
	if val == nil || val.Kind != yaml.ScalarNode {
		return nil, 0, false
	}
	declared, err := strconv.Atoi(val.Value)
	if err != nil {
		return nil, 0, false
	}
	return val, declared, true
}

// checkManifestSchemaVersion reports manifest-schema-unsupported where this
// Manifest is written in a shape above the one this binary knows, cited at the
// `schema-version:` scalar that says so.
//
// **It replaces this Manifest's own checks rather than joining them**, which is
// why CheckManifest returns on it. §11 calls the alternative expensive by name:
// a Manifest read on a partial understanding of its own shape has its
// declared-equals-derived Capability check run against keys the reader could not
// see, and that check is what the whole extension model rests on. So no schema
// check, no checkOperations, no checkCapabilityMismatch, no checkAuth — one
// code, and nothing else said about a file whose shape this binary does not
// know. A reader handed faults derived from keys the reader could not see is
// being handed guesses.
//
// **The reach of that replacement is the file, and it is bounded by the fold
// rather than by suppression.** The rejected alternative was to leave the name
// resolvable and suppress every check that reads *into* the Manifest's
// Operations: that makes suppression a property which has to propagate through
// the Definition and the Procedure checks, and a check that has to know it is
// reading a partially understood file is exactly the shape §11 is warning
// about. Instead the file contributes nothing to the Provider namespace, and a
// Definition naming it earns `artefact-absent` — two rows for one cause, which
// is ADR-0064's precedent and the shape the tool already takes for a file that
// will not parse.
//
// The remedy is a binary and not an edit — *a hyper that reads this schema
// version; nothing in the repository is the fault* — so a Refusal renders it as
// a named remedy rather than an `EDIT ONE OF` table (§8, internal/cli's
// refusalRemedies).
//
// **The comparison is not written here.** It is ManifestSchemaUnsupported's, and
// this reads that predicate, so the check that names a file and the fold that
// declines it cannot come to disagree about which side of the ceiling the file
// is on. What is written here is the citation and the sentence.
func checkManifestSchemaVersion(file string, root *yaml.Node) []problem.Problem {
	if !ManifestSchemaUnsupported(root) {
		return nil
	}
	declaredAt, version, _ := manifestSchemaVersion(root)
	return []problem.Problem{{
		File: file, Line: declaredAt.Line, Column: declaredAt.Column, Field: "schema-version",
		ErrorCode: CodeManifestSchemaUnsupported,
		Message: fmt.Sprintf(
			"schema-version: %d is above the %d this hyper reads — a Manifest shape this binary does not know, so nothing else about this file is checked",
			version, ManifestSchemaVersion),
	}}
}
