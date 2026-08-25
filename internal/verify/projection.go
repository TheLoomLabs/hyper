package verify

import (
	"bytes"
	"maps"
	"slices"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/pin"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// ProjectedWorkflow is one file the projection asks for: which Procedure asked
// for it, where it sits relative to the repository root, the Cadence that put
// it there, and its exact bytes.
//
// The Cadence rides along because every surface that reports a projected file
// glosses it (§10) — `project`'s own rows are the fourth surface that renders a
// gloss — and re-reading the artefact to find it again would be a second
// reading of the fact the bytes were derived from.
type ProjectedWorkflow struct {
	Procedure string
	Path      string
	Cadence   string
	Bytes     []byte
}

// Projection is every workflow file a repository's reviewed artefacts ask for:
// one per Procedure declaring a Cadence, ordered by Procedure name by Unicode
// code point (§10).
//
// It is exported beside Repository and ProcedureGraph, and for ProcedureGraph's
// own reason: **the command that writes the projection and the check that holds
// the working tree to it are one derivation**, and two of them is the day a
// repository `project` has just written fails its own check. What varies
// between the two callers is only what they do with the bytes — write them, or
// compare them against what stands.
//
// version and digest are the two facts here that are not read off the loaded
// repository, and they are parameters to say so. `project` derives the pin from
// the binary that ran it and freezes the checksum for that version in the same
// act (§11, ADR-0020) — so at the moment these bytes are generated the
// declaration on disk still carries the pin they are replacing, and reading
// either fact off it would project the version being left behind. What the
// caller hands over is what it is about to write into `hyper.yaml`, which is
// what makes the workflow and the declaration one edit rather than two
// (issue #178).
//
// **The other caller reads them off the declaration**, and is right to: §10's
// projection check regenerates against what `hyper.yaml` says, because under
// the pin gate the declaration's version and the running binary's are the same
// string and reading the file is what keeps the check inside a pass that
// touches nothing outside the load (projectionProblems below, issue #179).
//
// **Nothing here reads a file, a clock or a network**, and nothing here judges a
// repository: an artefact that would earn a problem still projects, because what
// stops a projection is `check` reporting something and that is its caller's to
// ask (§10, ADR-0064).
func Projection(loaded repository.Loaded, version, digest string) []ProjectedWorkflow {
	graph := ProcedureGraph(loaded)

	var projected []ProjectedWorkflow
	for _, name := range slices.Sorted(maps.Keys(loaded.Procedures)) {
		declaration, held := loaded.Procedure(name)
		if !held {
			continue
		}
		declared := artefact.ProcedureCadence(declaration.Root)
		if declared == "" {
			// A Procedure run by hand declares no recurrence, and
			// there is nothing for an executor's clock to hold (§10).
			continue
		}

		// Both conditional sections of the file come off the walk that
		// already exists — the pairs the Procedure reaches and whether
		// every Step it reaches is a `read` — asked once here so that
		// the `env:` block and the concurrency group cannot disagree
		// about what a Procedure runs (§10, issue #176).
		reach := graph.Reaches(name)
		projected = append(projected, ProjectedWorkflow{
			Procedure: name,
			Path:      workflow.Path(name),
			Cadence:   declared,
			Bytes: workflow.Generate(workflow.Facts{
				Procedure: name,
				Cadence:   declared,
				Effects:   !reach.EveryStepReads,
				Variables: projectedVariables(loaded, reach.Pairs),
				Version:   version,
				Digest:    digest,
			}),
		})
	}
	return projected
}

// projectedVariables is the environment variables the credential slots these
// pairs' bindings require resolve from, in the pairs' own order and with
// repeats.
//
// Ordering them and writing each once is the generator's, the block being a
// function of the repository rather than of the walk that found the pairs
// (workflow.Facts.Variables). A slot naming no variable contributes nothing —
// that is `credential-slot-malformed`, `check`'s to report rather than a
// reader's to repeat (§4, ADR-0064).
func projectedVariables(loaded repository.Loaded, pairs []store.Pair) []string {
	var named []string
	for _, pair := range pairs {
		_, slots := loaded.CredentialSlots(pair)
		for _, slot := range slots {
			if slot.Env != "" {
				named = append(named, slot.Env)
			}
		}
	}
	return named
}

// DeclaredPin is the version and the digest a loaded repository's own
// declaration carries — the two facts a projection regenerated against what
// `hyper.yaml` says is derived from — and the empty strings where it carries
// neither legibly.
//
// **The two facts come through two doors and that is the pin's own shape**: the
// version is internal/pin's, which is where the gate reads it, and the digest is
// internal/artefact's, which is where every fact the declaration *says* is read
// (§9, §11, ADR-0020).
//
// It is exported beside Projection, and for Projection's own reason: the check
// below regenerates at the declared pin, and so does the corpus guard that holds
// a fixture repository to the same derivation — **two readings of the pin is the
// day a `-update` writes bytes `check` disagrees with** (issue #181,
// internal/cli/golden_test.go).
//
// It is not the reading `project` does. That command derives the version from
// the binary that ran it and freezes a digest for it, which is a different
// question about a different moment and is stated at its own site (§11,
// internal/cli/project.go).
func DeclaredPin(loaded repository.Loaded) (version, digest string) {
	declaration, _ := loaded.DeclarationBytes()
	return pin.Declared(declaration), artefact.ReadRepositoryFacts(loaded.Declaration()).Digest
}

// CodeProjectionStale is §10's own static code: a generated workflow that is
// not what `project` would write now (§12).
//
// It is spelled here, at the check that fires it, and read from here by the
// Refusal rendering that knows its remedy — one string rather than two that
// happen to agree (§8, internal/cli/refusal.go).
const CodeProjectionStale = "projection-stale"

// projectionProblems is the verification half of generate-and-verify: the
// working tree's own `.github/workflows/hyper-*.yml` against a fresh
// regeneration of the same set, whole-file and byte-exact (§10, §12, issue
// #179).
//
// **Byte-exact is what catches every way the two can part**, and one comparison
// is why there is no shape of drift that has to be recognised for what it is: a
// Cadence edited and not projected, a hand-edit to a generated file, a
// generated file deleted, one left behind by a Procedure that no longer
// declares a Cadence, and a hand-edited version pin — which is therefore caught
// twice, here and by the fetched binary's own gate, neither detection depending
// on the other having run (§11, ADR-0020).
//
// **Regeneration reaches no network, and that is structural rather than
// disciplinary.** The version and the digest are read off `hyper.yaml` rather
// than off the binary's stamped facts: under the gate they are the same string,
// and reading the declaration is what keeps this rule inside a pass that takes
// a loaded repository and nothing else. Where the declaration carries neither
// legibly there is nothing to regenerate against and this reports **nothing** —
// that is the declaration's own `schema-mismatch`, already reported, and a
// second opinion would put two rows on the page for one fault (§3, ADR-0064).
//
// Which two facts those are, and where each is read, is DeclaredPin's.
func projectionProblems(loaded repository.Loaded) []problem.Problem {
	version, digest := DeclaredPin(loaded)
	if version == "" || digest == "" {
		return nil
	}

	standing := make(map[string][]byte, len(loaded.Workflows))
	for _, file := range loaded.Workflows {
		standing[file.Path] = file.Bytes
	}

	var problems []problem.Problem
	wanted := Projection(loaded, version, digest)
	asked := make(map[string]bool, len(wanted))
	for _, file := range wanted {
		asked[file.Path] = true
		held, stands := standing[file.Path]
		switch {
		case !stands:
			problems = append(problems, stale(file.Path,
				"the Procedure "+file.Procedure+" declares a Cadence and the working tree holds no file here — hyper project writes it"))
		case !bytes.Equal(held, file.Bytes):
			problems = append(problems, stale(file.Path,
				"this file is not what hyper project would write now — the comparison is whole-file and byte-exact"))
		}
	}
	for _, path := range unwanted(loaded, asked) {
		problems = append(problems, stale(path, "no Procedure asks for this file — hyper project removes it"))
	}
	return problems
}

// UnwantedWorkflows is the files standing in the namespace that this
// repository's projection does not ask for: what `hyper project` takes away in
// the same act that writes the rest, and what the check above reports as the
// second of the code's three shapes.
//
// **The namespace is answered as a set**, which is why there is no shape of
// leftover that has to be recognised for what it is: a Procedure that has
// dropped its `cadence:`, a Procedure that has been deleted, and a
// `hyper-*.yml` naming no Procedure at all reach it the same way (§10).
//
// It is exported beside Projection, and for Projection's own reason: **the
// command that removes a file and the check that says the file should not be
// there are one derivation**, and two of them is the day a `project` leaves
// behind a repository that fails its own check.
//
// wanted is the caller's projection rather than one computed here, because the
// two callers project at two versions — `project` at the binary's, the check at
// the declaration's — and which files are asked for is the same set either way
// (issue #178).
func UnwantedWorkflows(loaded repository.Loaded, wanted []ProjectedWorkflow) []string {
	asked := make(map[string]bool, len(wanted))
	for _, file := range wanted {
		asked[file.Path] = true
	}
	return unwanted(loaded, asked)
}

// unwanted is that set over an index the caller already holds.
func unwanted(loaded repository.Loaded, asked map[string]bool) []string {
	var held []string
	for _, file := range loaded.Workflows {
		if !asked[file.Path] {
			held = append(held, file.Path)
		}
	}
	return held
}

// stale is one shape of the code, cited at the file and nowhere further in.
//
// **It carries no line, no column and no field**, and renders no caret on a
// Refusal for the same reason (§8, internal/cli/refusal.go). The comparison is
// whole-file, so a diff hunk is not a citation — and pointing a reader at line
// 14 of a file they must not hand-edit is worse than pointing at the file, the
// file being derived from the artefacts rather than authored (§9, ADR-0011's
// own ground one artefact over).
//
// Each shape names `hyper project` verbatim, which is a check knowing its own
// remedy stated as a fact rather than editorialised (§8).
func stale(path, message string) problem.Problem {
	return problem.Problem{File: path, ErrorCode: CodeProjectionStale, Message: message}
}
