package verify

import (
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// CodeProviderNameCollision is §11's own static code: a Manifest in providers/
// taking a built-in Provider's name (§12).
//
// It is spelled here, at the check that fires it, on CodeProjectionStale's own
// footing — one string rather than two that happen to agree.
const CodeProviderNameCollision = "provider-name-collision"

// collisionProblems is §11's first rule about what an Extension may never be:
// **it may never shadow a built-in Provider's name.** A Manifest in providers/
// declaring one is refused, and the built-in stands — the compiled-in Manifest
// is what the name means, so a Definition resolving through it resolves through
// the built-in, which is what it was reviewed against.
//
// **A load failure and never a precedence rule**, precedence being how a
// Definition reviewed as one thing runs as another. The declining half is the
// fold's, where a colliding file contributes nothing to the Provider namespace
// (internal/repository's manifestsByName); this is the row that says so, and the
// two read one predicate rather than two agreeing readings of §12's set
// (artefact.IsBuiltinProviderName).
//
// **Its subject is a set rather than a file**, which is what puts it here beside
// the graph's pass and the projection's rather than inside artefactChecks. The
// file is not malformed: a colliding Manifest can be a well-formed one, and
// every check CheckManifest runs over it can pass. What is wrong is that the
// namespace has no room for the member it declares — a fact about the Provider
// namespace and about the fold that declined it, rather than about this file's
// own shape.
//
// **That is not §11's *decidable from the file alone* being contradicted.** That
// sentence is the claim that neither Extension rule reaches a registry or asks
// who published the bytes, and this one does neither: the set it reads against
// is compiled in, so the answer needs the file and this binary and nothing else
// (artefact.IsBuiltinProviderName). *Decidable from* the file and *about* the
// file are two questions, and only the first is what §11 is closing.
//
// **Two Extensions colliding is impossible by construction**, so this is one
// collision rule and not two: `name-mismatch` pins every Manifest's `provider:`
// to its file's basename and two files in providers/ cannot share a basename, so
// the only name an Extension can take from anyone is a built-in's — which is
// exactly the set §12 states the built-in Providers double as, and which grows
// only where the reserved half of the Capability set grows. **This is the one
// site that argument is written at**; the fold cites it rather than restating it.
//
// It is cited on the offending file's `provider:` scalar. The built-in has no
// file to cite and is not the fault; the file whose author can act is the one
// that took the name.
func collisionProblems(loaded repository.Loaded) []problem.Problem {
	var problems []problem.Problem
	for _, a := range loaded.Artefacts {
		// The subject is a file in providers/, which is artefactChecks'
		// own spelling of the same question one file over.
		// artefact.ProviderOrigin is not it and cannot be: it answers
		// `extension` for every path that is not the built-in's
		// pseudo-path, a targets/ file included, and it is meaningful
		// only once a caller already holds a Manifest (ADR-0073).
		if !a.OK || !strings.HasPrefix(a.Path, "providers/") {
			continue
		}
		name := artefact.ManifestProviderName(a.Root)
		if !artefact.IsBuiltinProviderName(name) {
			continue
		}
		line, column := artefact.ManifestProviderNamePosition(a.Root)
		problems = append(problems, problem.Problem{
			File: a.Path, Line: line, Column: column, Field: "provider",
			ErrorCode: CodeProviderNameCollision,
			Message:   "provider: " + name + " is a built-in Provider's name — the built-in stands and this file names nothing in the Provider namespace",
		})
	}
	return problems
}
