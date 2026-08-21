package run

import (
	"fmt"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
)

// What this binary does not implement in a Run, and the artefacts a Run read
// (§6, §9, issues #136, #141).
//
// The Steps themselves are sequence.go's: a Run's Steps are the top-level
// Procedure's and everything it invokes, flattened into one written order.

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
// would make the call and write an Observation. That is a binary doing
// something undefined, which is the one thing `run` may not do on the day it
// lands.
//
// **It walks the flattened sequence**, so an effectful Step reached through a
// nested invocation declines exactly as one written at the top level does. The
// invocation itself is no longer among them: a nested invocation runs as part
// of the one Run (issue #141), and what its Steps are is judged here like any
// other Step's.
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
	var declined []string
	for _, step := range flatten(loaded, procedure).Steps {
		if kind := kindOf(loaded, step); kind != "" && kind != "read" {
			declined = append(declined, fmt.Sprintf(
				"step %s is a %s Step, and effectful Steps are not built yet", named(step), kind))
		}
	}
	return declined
}

// kindOf is the Kind the Step's Operation declares, and "" where the binding
// does not resolve far enough to say.
func kindOf(loaded repository.Loaded, step sequenced) string {
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
// It is the Repository declaration, every Procedure the Run walked — the
// top-level one and each one it invokes, at any depth — and per Step its
// Definition, its Target declaration and its Provider's Manifest. The built-in
// Provider contributes nothing: its bytes are compiled in and it has no blob in
// the repository at all, so there is nothing for it to differ from (§3, §11).
//
// A file named twice is read once: a Procedure whose ten Steps bind one
// Definition read one Definition, and a Procedure invoked from two places is
// one file.
//
// A nested Procedure's own revision is not Provenance's. `procedure_revision`
// is the **top-level** Procedure's, the only reading with exactly one value at
// Run level (ADR-0048); what a nested Procedure's bytes reach here is
// `repo_dirty` and §8's code-change classes, which is where a Bound widened
// inside one is reported.
func artefactsRead(loaded repository.Loaded, walked sequence) []revision.File {
	files := map[string][]byte{}
	if declaration, held := loadedAt(loaded, repository.DeclarationPath); held {
		files[declaration.Path] = declaration.Bytes
	}
	for _, procedure := range walked.Procedures {
		files[procedure.Path] = procedure.Bytes
	}

	for _, step := range walked.Steps {
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
