package run

import (
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The artefacts a Run read, and the Kind a Step binds (§6, §9, issues #136,
// #141).
//
// The Steps themselves are sequence.go's: a Run's Steps are the top-level
// Procedure's and everything it invokes, flattened into one written order.
//
// **Nothing here declines a Step for being effectful.** The arm that did —
// NotBuilt, and the engine's own precondition on it — was the interim milestone
// 5 shipped on the stated understanding that milestone 6 deletes it, and issue
// #148 is where it went: a `mutate` or `destroy` Step resolves, expands, calls
// and records like a `read` Step, and what an effect *means* is effect.go's
// (§6).
//
// **The three Kinds are all here.** #148 landed the spine and the `mutate`
// semantics on it, and #150 the `destroy`: it completes on `404` besides,
// writes a Tombstone where an Asset would go, dispatches strictly serially
// whatever any Manifest declares, and stops its Expansion at the first error
// rather than draining it. All four live where the Kind is read — effect.go's
// judgement, its recordType and its tombstones, and the walk step.go takes
// over what drain.go hands back. The decline this file used to carry was
// deleted rather than narrowed to `destroy`, because narrowing is what the
// ticket called reclassifying: a reviewer of that diff should see a gate
// removed, not a behaviour changed.
//
// What is still `read`-shaped is the `shell` Capability's effectful half, which
// completes on `0` alone and carries the command and the exit code under
// `answered`, and that is issue #156's (effect.go).

// kindOf is the Kind the Step's Operation declares, and "" where the binding
// does not resolve far enough to say.
//
// It answers store.Kind rather than the string internal/artefact holds, because
// §12's set is closed and every reader of it here compares against a member:
// the lock mode, the push rhythm, what a call's answer means and what a version
// a Step writes is a version of. A bare string compared against a literal in
// one place and a constant in another is one set spelled two ways (§12,
// effect.go, lock.go).
func kindOf(loaded repository.Loaded, step sequenced) store.Kind {
	definition, declared := loaded.Definitions[step.Definition]
	if !declared {
		return ""
	}
	return store.Kind(loaded.Providers[definition.ProviderName].Operations[step.Operation].Kind)
}

// effectfulStep says this one Step binds an Operation whose declared Kind is not
// `read` — the per-Step half of the reading lock.go asks of a whole walk, and
// the one three things now ask: which lock the Run takes, which rhythm it
// pushes at, and where a rehearsal stops (§6, §9, ADR-0010).
//
// It carries the Step in its name rather than in its signature alone, because
// *effectful* is the word for both readings and the one over a walk already
// holds it (lock.go). Two spellings of one predicate is where the day comes that
// a Run is effectful and its Steps are not.
//
// **A Step whose Kind cannot be read counts as effectful.** An unresolvable
// binding leaves no Kind to judge, and every reader here wants the same safe
// answer — a Step whose blast radius cannot be read may not share the Store,
// may not keep its account to itself, and is not one a rehearsal walks past. No
// such Step reaches a Run at all: `check` re-runs in full at Run start and
// refuses it, so the answer costs nothing where nothing can reach it (lock.go).
func effectfulStep(loaded repository.Loaded, step sequenced) bool {
	return kindOf(loaded, step) != store.KindRead
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
