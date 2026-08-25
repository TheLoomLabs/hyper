package verify

import (
	"maps"
	"slices"

	"github.com/TheLoomLabs/hyper/internal/artefact"
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
