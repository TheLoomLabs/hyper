package run

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Steps a Run holds, flattened: **a Procedure invoking another does not
// start a second Run** (§6, §7, issue #141).
//
// The invoked Procedure's Steps are Steps of the one Run, counted in the one
// written order and recorded under a path. One Run has one outcome, one Journal
// entry and one exit code however deep the invocation goes, and a halt inside a
// nested Procedure is a halt of the whole.
//
// **The invocation itself is not a Step.** None of §12's seven Dispositions
// describes one, it writes no file, and it takes no position in the sequence —
// so this walk emits what it reaches through an invocation and never the
// invocation.
//
// **Nor is a Requirement**, on the same three grounds. It is walked all the
// same, because unlike an invocation it is something that happens: it is
// collected beside the Steps, each one holding the position of the Step it
// stands in front of, so the engine evaluates it in written order without
// giving it a number the Journal would have to account for (§6, ADR-0116).
//
// **There is no depth limit**, because the invocation graph is static and a
// cycle is rejected before the first Step (§6, ADR-0002). What rejects it is
// `check` — `procedure-cycle`, cited at the invocation entry that closes the
// loop (§4, issue #146) — and `check` re-runs in full at Run start, so a
// cyclic repository Refuses at `77` before this walk's answer is read for
// anything. The walk names the cycle all the same, because the alternative is
// not silence but truncation: dropping the recursive invocation and running
// the rest is a Run performing a Procedure nobody wrote, and the name is what
// run.go's own precondition reports for a caller that reached the engine some
// other way.

// sequenced is one Step of a Run's flattened sequence: the Step as authored,
// and the facts about *where it was authored* that only a flattened sequence
// can answer.
type sequenced struct {
	// Step is the member of some Procedure's `steps:`, held exactly as
	// artefact.Step reads it.
	artefact.Step
	// Path is the invocation chain this Step was reached through, ending in
	// its own authored id — `retire.probe` — and "" on a top-level Step. It
	// is what the Step file and every Record version the Step writes carry
	// beside that id (§7).
	Path string
	// Declared is the Procedure file the Step was authored in and Index its
	// position in that file's own `steps:`. They travel together because a
	// Refusal's coordinate is an artefact coordinate: a file and a position
	// in it, which a flattened Run's own position is neither of (§7,
	// ADR-0061).
	Declared repository.LoadedArtefact
	Index    int
	// Namespace is the id namespace this Step was authored in, numbered as
	// the walk reaches it. It is what a `when:` condition and a `{step:,
	// path:}` reference resolve an id against: an id is unique inside one
	// Procedure and says nothing across two (§3), and two invocations of one
	// Procedure are two namespaces — so the second's Steps read the second's
	// Records and never the first's.
	//
	// It is a number rather than the Procedure's name for exactly that
	// reason, and it is neither the path nor a member of any file: it exists
	// while the Run is in flight and is written nowhere.
	Namespace int
}

// requirement is one Requirement of a Run's flattened sequence: the predicate
// it carries, where it was authored, and the Step it stands in front of.
//
// It carries a namespace for the reason a Step does — an id is unique inside
// one Procedure and says nothing across two — and it carries no position,
// because it takes none (§3, §6, sequenced).
type requirement struct {
	// Require is the `require:` mapping as authored: a predicate at the
	// condition's own root, `step:` beside `field:` (§3, §12).
	Require *yaml.Node
	// ID is the Requirement's authored `id:`, and Path the invocation chain
	// it was reached through ending in that id — the two names a halt at
	// this Requirement is reported by, on a Step's own rule (§7).
	ID, Path string
	// Declared is the Procedure file it was authored in, Index its position
	// in that file's own `steps:` and Line where its entry begins: the
	// artefact coordinate a Refusal at it cites, which is the only
	// coordinate it has (§7, ADR-0061).
	Declared repository.LoadedArtefact
	Index    int
	Line     int
	// Namespace is the id namespace its `step:` resolves against, numbered
	// as the walk reached it (sequenced).
	Namespace int
	// Before is the index into the Run's Steps of the Step this Requirement
	// stands in front of, and len(Steps) where it stands after the last one
	// the Run holds. Written order is the whole of what it encodes: a
	// Requirement authored last inside an invoked Procedure stands in front
	// of the caller's next Step, which is exactly what makes a shared check
	// gate the Procedure that invoked it (§6, ADR-0116).
	Before int
}

// identity is one Record identity this Step writes under: the Target and
// Definition it is bound to, and the name a projection resolved.
//
// It is here rather than at each site that builds one because the triple is a
// Record's whole identity (§7) and two of the three are the Step's own: a caller
// filling them one at a time is a caller that can fill one of them from
// somewhere else, and an identity built against another Step's Definition is a
// version written into a series nobody named.
func (s sequenced) identity(name string) store.Identity {
	return store.Identity{Target: s.Target, Definition: s.Definition, Name: name}
}

// name is how one Step of a Run is named where one name is wanted: its path
// where it was reached through an invocation, and its authored id where it sits
// at the top level. A Step with neither is one `check` reports and this
// milestone must still be able to talk about.
func named(step sequenced) string {
	switch {
	case step.Path != "":
		return step.Path
	case step.ID != "":
		return step.ID
	default:
		return fmt.Sprintf("at line %d", step.Line)
	}
}

// sequence is a Run's Steps in the order they run, together with the two facts
// the walk answers that no Step of it carries.
type sequence struct {
	// Steps is every Step the Run holds, in written order — which is the
	// order they run in, the order the Step table renders them in and the
	// order `<nnnn>` counts them by (§6, §8, §12).
	Steps []sequenced
	// Procedures is every Procedure file the walk read, the top-level one
	// first and each one once. It is part of the file set `repo_dirty` is
	// decided over, a nested Procedure being an artefact the Run read (§7,
	// §8).
	Procedures []repository.LoadedArtefact
	// Whole says the walk reached every Step the Procedure holds. It is
	// false where an invocation named a Procedure this walk could not
	// descend into — one that resolves to nothing, which is `check`'s
	// `artefact-absent`, and one that is the cycle below. Neither reaches
	// Step 1; the reading that is not unreachable is the lock, which is
	// taken before `check` has re-run (lock.go).
	Whole bool
	// Requirements is every Requirement the Run holds, in written order,
	// each carrying the index of the Step it stands in front of. They are
	// beside the Steps rather than among them because a Requirement takes
	// no position in the sequence (§6, ADR-0116).
	Requirements []requirement
	// Cycle is the Procedure an invocation named that the walk was already
	// inside of, and "" where the graph is acyclic. Both are `check`'s to
	// refuse at the gate §6 puts before Step 1 — an invocation naming
	// nothing is `artefact-absent` and a cycle is `procedure-cycle` — so
	// what this member is for is the precondition run.go states past that
	// gate rather than the Refusal itself.
	Cycle string
}

// flatten walks the named Procedure and everything it invokes, in written
// order.
//
// It resolves names and judges none of them, which is this package's rule for
// reading an artefact: an invocation naming nothing is `artefact-absent` at
// `check`, and a Run that reaches Step 1 is a Run whose every name resolved
// (ADR-0064).
func flatten(loaded repository.Loaded, procedure string) sequence {
	top, resolved := loaded.Procedure(procedure)
	if !resolved {
		return sequence{}
	}

	walked := sequence{Whole: true}
	read := map[string]bool{}
	// namespaces numbers the id namespaces as the walk reaches them, the
	// top-level Procedure's being 0. It counts occurrences rather than names
	// because two invocations of one Procedure are two namespaces.
	namespaces := 0
	visiting := map[string]bool{procedure: true}

	var walk func(file repository.LoadedArtefact, prefix string, namespace int)
	walk = func(file repository.LoadedArtefact, prefix string, namespace int) {
		if !read[file.Path] {
			read[file.Path] = true
			walked.Procedures = append(walked.Procedures, file)
		}
		for index, step := range artefact.ReadProcedureSteps(file.Root) {
			if step.IsRequirement() {
				walked.Requirements = append(walked.Requirements, requirement{
					Require:   step.Require,
					ID:        step.ID,
					Path:      pathUnder(prefix, step.ID),
					Declared:  file,
					Index:     index,
					Line:      step.Line,
					Namespace: namespace,
					Before:    len(walked.Steps),
				})
				continue
			}
			if !step.IsInvocation() {
				walked.Steps = append(walked.Steps, sequenced{
					Step:      step,
					Path:      pathUnder(prefix, step.ID),
					Declared:  file,
					Index:     index,
					Namespace: namespace,
				})
				continue
			}
			invoked, held := loaded.Procedure(step.Invocation)
			if !held {
				walked.Whole = false
				continue
			}
			if visiting[step.Invocation] {
				walked.Whole = false
				if walked.Cycle == "" {
					walked.Cycle = step.Invocation
				}
				continue
			}
			visiting[step.Invocation] = true
			namespaces++
			walk(invoked, prefix+step.Invocation+".", namespaces)
			delete(visiting, step.Invocation)
		}
	}
	walk(top, "", 0)

	return walked
}

// pathUnder is the path a Step reached under one invocation chain carries, and
// "" at the top level — where §7 states a Step carries none, its own `id`
// being the whole of what names it.
//
// The chain is the **Procedures** invoked and not the ids that invoked them.
// That is what makes §8's *a nested Procedure is read by every Run carrying it
// in a Step file's `path`* a reading a surface can perform: the range a review
// anchors for `procedures/provision.yaml` is found by the name in the path,
// where an invocation's own id would name the caller's line instead — and §7
// refuses the alternative reading outright, a Journal whose entries cannot be
// read without loading three artefacts at the revision that Run names being
// evidence with a dependency.
//
// What it costs is named rather than hidden: a Procedure invoking one other
// Procedure **twice** gives both invocations' Steps one path, told apart by the
// position each holds in the Run. An invocation id would have kept them apart,
// and it would have cost the reading above on every Run that has one.
//
// The Step's own id is the last segment, which is what both of §6's and §7's
// examples spell — `deploy.provision.create-vm`, `retire.probe` — and the id is
// written beside it all the same: the path is where a Step sat and the id is
// what its author called it, and a surface reading one back out of the other
// would be parsing a name (§7).
func pathUnder(prefix, id string) string {
	if prefix == "" {
		return ""
	}
	return prefix + id
}
