package run

import (
	"fmt"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
)

// The Steps a Run holds, and the boundary this milestone has to state out loud
// (§6, §9, issue #136).

// readSteps is the Procedure's Steps in written order — which is the order they
// run in, the order the Step table renders them in and the order `<nnnn>`
// counts them by (§6, §8, §12).
//
// A nested invocation's own Steps are Steps of the one Run and are counted in
// that order, the invocation itself being no Step and writing no file (§6, §7).
// Flattening them is issue #141's; until it lands, an invocation is one of the
// things declineUnbuilt below stops the Run on, so this walk reads the top-level
// sequence and no deeper.
func readSteps(procedure repository.LoadedArtefact) []artefact.Step {
	return artefact.ReadProcedureSteps(procedure.Root)
}

// NotBuilt is what this binary does not implement in the named Procedure, one
// line each, and nothing at all where every Step of it is one this milestone
// performs.
//
// **It is not a Refusal.** It carries no `error_code`, writes no Journal entry,
// opens no stream, renders on stderr and exits `2` — and the milestone that
// builds the thing deletes its arm rather than reclassifying it. The precedent
// is internal/cli/tree.go's, stated there in as many words: a name in §9's tree
// is a name the spec fixes and not a claim that the binary implements it yet.
//
// It exists because the alternative is worse than an unimplemented feature. The
// engine implements the `read` path's semantics; handed a `mutate` Step it
// would make the call and write an Observation, and handed a Step carrying an
// `over:` it would make one call and ignore the selector. Both are a binary
// doing something undefined, which is the one thing `run` may not do on the day
// it lands.
//
// It is exported because it decides an **order**, and the order is §9's: a
// working-tree name resolves against the working tree and needs nothing
// further, so this is asked before the Store is located and an operator whose
// Procedure this binary cannot perform is told so rather than sent to run
// `hyper store init` first. The engine asks it again as its own precondition —
// one door, and a two-call contract is a contract that can be got wrong.
//
// Every reached Step is reported rather than the first, which is the reading
// §6's credential gate already takes: an operator sent round the loop once per
// Step earns one decline per round trip.
//
// The Kind is read off the Operation the Step binds and never off the Step: a
// Kind is declared per Operation in a Manifest and never inferred (ADR-0025),
// so a Step whose binding does not resolve carries no Kind here and is left to
// the resolution that will report it.
func NotBuilt(loaded repository.Loaded, procedure string) []string {
	file, resolved := loaded.Procedure(procedure)
	if !resolved {
		return nil
	}

	var declined []string
	steps := readSteps(file)
	for _, step := range steps {
		switch {
		case step.IsInvocation():
			declined = append(declined, fmt.Sprintf(
				"step %s invokes the Procedure %s, and a nested invocation is not built yet", named(step), step.Invocation))
		case step.Over != nil:
			declined = append(declined, fmt.Sprintf(
				"step %s carries an over: selector, and Expansion is not built yet", named(step)))
		case step.When != nil:
			declined = append(declined, fmt.Sprintf(
				"step %s carries a when: condition, and conditions are not built yet", named(step)))
		default:
			if kind := kindOf(loaded, step); kind != "" && kind != "read" {
				declined = append(declined, fmt.Sprintf(
					"step %s is a %s Step, and effectful Steps are not built yet", named(step), kind))
			}
		}
	}
	return declined
}

// named is how a decline names a Step: its authored id, or its position in the
// sequence where it wrote none. A Step with no id is `check`'s to report and
// this milestone's to still be able to talk about.
func named(step artefact.Step) string {
	if step.ID == "" {
		return fmt.Sprintf("at line %d", step.Line)
	}
	return step.ID
}

// kindOf is the Kind the Step's Operation declares, and "" where the binding
// does not resolve far enough to say.
func kindOf(loaded repository.Loaded, step artefact.Step) string {
	definition, declared := loaded.Definitions[step.Definition]
	if !declared {
		return ""
	}
	return loaded.Providers[definition.ProviderName].Operations[step.Operation].Kind
}

// artefactsRead is the reviewed artefacts the Run read, which is exactly the
// file set `repo_dirty` is decided over and exactly the file set §8's catch-all
// row counts the moved lines of — one sentence, so the marker and the count
// agree by construction (§7, §8).
//
// It is the Repository declaration, the top-level Procedure, and per Step its
// Definition, its Target declaration and its Provider's Manifest. The built-in
// Provider contributes nothing: its bytes are compiled in and it has no blob in
// the repository at all, so there is nothing for it to differ from (§3, §11).
//
// A file named twice is read once: a Procedure whose ten Steps bind one
// Definition read one Definition.
func artefactsRead(loaded repository.Loaded, procedure repository.LoadedArtefact, steps []artefact.Step) []revision.File {
	files := map[string][]byte{procedure.Path: procedure.Bytes}
	if declaration, held := loadedAt(loaded, repository.DeclarationPath); held {
		files[declaration.Path] = declaration.Bytes
	}

	for _, step := range steps {
		if definition, held := loaded.Definition(step.Definition); held {
			files[definition.Path] = definition.Bytes
		}
		if target, held := loaded.TargetDeclaration(step.Target); held {
			files[target.Path] = target.Bytes
		}
		if info, declared := loaded.Definitions[step.Definition]; declared {
			if manifest, held := loaded.Manifests[info.ProviderName]; held && strings.HasPrefix(manifest.Path, "providers/") {
				files[manifest.Path] = manifest.Bytes
			}
		}
	}

	read := make([]revision.File, 0, len(files))
	for _, path := range slices.Sorted(keys(files)) {
		read = append(read, revision.File{Path: path, Bytes: files[path]})
	}
	return read
}

// loadedAt is the artefact at one path, which is how the Repository declaration
// is reached: it is the one artefact keyed by its filename rather than by a
// name it declares (§3, §12).
func loadedAt(loaded repository.Loaded, path string) (repository.LoadedArtefact, bool) {
	for _, a := range loaded.Artefacts {
		if a.Path == path {
			return a, true
		}
	}
	return repository.LoadedArtefact{}, false
}

// keys is a mapping's keys as a sequence, for the one sort above. Go's maps
// package answers it and this is spelled out rather than imported under a name
// that would shadow the parameter it is used beside.
func keys[V any](m map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for key := range m {
			if !yield(key) {
				return
			}
		}
	}
}
