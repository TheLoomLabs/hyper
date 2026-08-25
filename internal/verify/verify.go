// Package verify is §4's static verification over an already-loaded
// repository: the second pass, which runs after internal/repository has walked
// the artefact locations and parsed every file, and which reports every problem
// it finds rather than the first.
//
// It is a package of its own because it has two callers and one of them is not
// a command. `hyper check` is the surface §9 gives it; a Run re-runs it **in
// full with nothing skipped** at Run start, which is how all thirty-two of §4's
// static codes reach a Run and why most of the closed `error_code` set declines
// before Step 1 (§6, ADR-0061). Two callers spelling that pass for themselves
// is two readings of §4, and the day comes that a Run admits a repository
// `check` refuses.
//
// It owns two rules and no more. Every other problem it reports comes from the
// load (internal/repository, over internal/yamlsubset's own grammar) and from
// an artefact's own schema and checks (internal/artefact); what is here is the
// walk — which checks run over which artefact, and the one graph-wide pass that
// needs every procedures/ file at once.
//
// The rules that are its own are the two whose subject is not an artefact, and
// so could belong to no artefact's checks: §11's `provider-name-collision`, a
// Manifest in providers/ taking a name a built-in Provider already means
// (collision.go), and §10's `projection-stale`, the generated workflows the load
// found against the projection this repository's artefacts ask for
// (projection.go).
//
// Two derivations stand beside those rules and are exported for one reason: a
// surface renders what this pass checks, and two builders of one thing is where
// the day comes that they disagree. ProcedureGraph is the invocation graph a
// review's roster quantifies over; Projection is the set of workflow files
// `hyper project` writes, which is the same set §10's own check holds a working
// tree to — generate and verify being one derivation called from two places
// (projection.go).
package verify

import (
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// Repository runs every static check over an already-loaded repository and
// answers what it found, unsorted and unfiltered.
//
// The order is the walk's, and it is deliberately not the answer: §9 fixes the
// order problems are reported in — by file path, then by line — and
// problem.Sort is what puts them there. A caller that reports a subset filters
// after this rather than narrowing it, because every rule compares one artefact
// against another and a subset of the repository is not checkable on its own
// (§9).
//
// A failed load does not stop it. Reading or parsing one file stops every check
// after it for that file and never for the repository, which is why the load's
// own problems are carried through here beside the checks' rather than short-
// circuiting them (issue #88).
func Repository(loaded repository.Loaded) []problem.Problem {
	var problems []problem.Problem
	for _, a := range loaded.Artefacts {
		problems = append(problems, a.Problems...)
		if !a.OK {
			continue
		}
		problems = append(problems, artefactChecks(a, loaded)...)
	}

	// The transitive walk — an invoked Procedure's own envelope against its
	// caller's, and the two Cadence rules that ride the same walk — needs
	// every procedures/ file at once, so it runs once here rather than per
	// file inside artefactChecks (issue #96).
	problems = append(problems, artefact.CheckProcedureGraph(ProcedureGraph(loaded))...)

	// The second pass whose subject is a set: a Manifest in providers/
	// taking a built-in Provider's name. *This name is taken* is a fact
	// about the Provider namespace, which no one file's checks can see, so
	// it runs here rather than inside artefactChecks (§11, issue #185).
	problems = append(problems, collisionProblems(loaded)...)

	// And the one rule whose subject is not an artefact at all: the
	// generated workflows the load found, against the projection this
	// repository asks for. It runs here rather than beside a file's own
	// checks because the comparison is over a **set** — a file nobody asks
	// for is a fault of the namespace rather than of any artefact — and
	// because it reaches `check` and a Run's pre-flight through this one
	// signature, neither having to be taught the rule (§10, issue #179).
	return append(problems, projectionProblems(loaded)...)
}

// ProcedureGraph is the invocation graph an already-loaded repository makes:
// every procedures/ file at once, which is what the transitive checks and a
// Procedure's own review roster both need (issue #96).
//
// It is exported beside Repository because the review renders the same graph
// this pass checks — §8's roster on a Procedure quantifies over every Step's
// `target:` to any depth — and two builders of one graph is where the day comes
// that a review's marks and a `check`'s problems disagree about what a nested
// invocation reaches.
func ProcedureGraph(loaded repository.Loaded) artefact.ProcedureGraph {
	return artefact.BuildProcedureGraph(procedureRoots(loaded.Artefacts), loaded.Providers, loaded.Definitions)
}

// artefactChecks runs one already-parsed artefact's own schema and the checks
// that read it against itself or against the repository: hyper.yaml, a file in
// targets/, a file in providers/, a file in definitions/ and a file in
// procedures/ — the five artefacts §3 states — plus the built-in shell
// Provider, which the load carries like any other artefact and which is checked
// here with no exemption (§3, ADR-0081, issues #99, #109): a Provider is data,
// and data check may not read is an advisory analyzer wearing the tool's own
// badge.
//
// loaded is passed whole rather than as its four namespaces, which travel
// together everywhere and are what an artefact's authored names resolve
// against: providers and targets are the namespaces a Definition's provider:
// and targets: resolve against, definitions and procedures the namespaces a
// Step's definition: and a nested invocation's procedure: resolve against.
func artefactChecks(a repository.LoadedArtefact, loaded repository.Loaded) []problem.Problem {
	switch {
	case a.Path == artefact.BuiltinShellProviderPath:
		return artefact.CheckBuiltinShellProvider()
	case a.Path == repository.DeclarationPath:
		return artefact.CheckRepositoryDeclaration(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "targets/"):
		return artefact.CheckTargetDeclaration(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "providers/"):
		return artefact.CheckManifest(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "definitions/"):
		return artefact.CheckDefinition(a.Path, a.Root, loaded.Providers, loaded.Targets)
	case strings.HasPrefix(a.Path, "procedures/"):
		return artefact.CheckProcedure(a.Path, a.Root, loaded.Providers, loaded.Definitions, loaded.Targets, loaded.Procedures)
	}
	return nil
}

// procedureRoots is every loaded Procedure's root paired with its own file —
// what BuildProcedureGraph needs to cite a fault against the file that carries
// it (issue #96). A file that failed to parse contributes no root, on
// ADR-0064's own rule. The graph is this pass's and not the load's, so the
// shaping its input needs is here.
func procedureRoots(artefacts []repository.LoadedArtefact) []artefact.ProcedureRoot {
	var roots []artefact.ProcedureRoot
	for _, a := range artefacts {
		if a.OK && strings.HasPrefix(a.Path, "procedures/") {
			roots = append(roots, artefact.ProcedureRoot{File: a.Path, Root: a.Root})
		}
	}
	return roots
}
