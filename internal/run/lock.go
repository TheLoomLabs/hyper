package run

import (
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Which lock a Run takes and at what rhythm it pushes, and why both answers are
// readable before a Run exists (§6, §7, issues #138, #148).
//
// They are one reading because they are one fact — *does this Run touch the
// world* — asked twice: an effectful Run takes the exclusive lock and pushes
// after every Step, a read-only Run takes the shared one and batches its pushes
// to its end (§7, ADR-0006, ADR-0083). Two walks over the same Kinds is where
// the day comes that a Run holds the shared lock and pushes like an effectful
// one.

// LockMode is the lock the named Procedure's Run holds for its duration:
// **exclusive where it contains any effectful Step, shared where every Step is
// `read`**. So a five-minute monitoring cadence is not starved behind a
// forty-minute provision, and two monitoring cadences are not starved behind
// each other.
//
// It is decided from the Kinds `check` already computes and **before any Step
// runs** — before the Store is reached at all, in fact, since the lock is a
// lock on the Store and a Run that cannot take it never opens one. That is what
// makes this a walk over reviewed text with no world in it: the Kind is read
// off the Operation the Step binds and never off the Step, a Kind being
// declared per Operation in a Manifest and never inferred (ADR-0025).
//
// **A Step whose Kind cannot be read is exclusive**, and so is a Procedure
// whose Steps this walk could not reach in full. An unresolvable binding leaves
// no Kind to judge; an invocation naming nothing, and a cycle, leave Steps the
// walk never saw (sequence.go) — and a Run whose blast radius cannot be read is
// not a Run that may share the Store. None of them ever gets as far as its
// first Step: every one is `check`'s to refuse at Run start. The lock is taken
// before that, so what it does with them is a fact of its own rather than one
// that path makes unreachable.
//
// A Procedure name that resolves to nothing is exclusive on the same reading.
// It is unreachable from the CLI, which resolves the positional against the
// namespace first (§9, ADR-0060), and the safe answer costs nothing where
// nothing can reach it.
func LockMode(loaded repository.Loaded, procedure string) store.LockMode {
	if effectful(loaded, flatten(loaded, procedure)) {
		return store.Exclusive
	}
	return store.Shared
}

// effectful says the walked Procedure's Run touches the world: any Step of it
// binds an Operation whose declared Kind is not `read`.
//
// It takes the walk rather than the name because the engine has already made
// one and a second would be a second answer. LockMode is the caller that has
// only a name, the lock being taken before a Run exists at all.
//
// **A Step whose Kind cannot be read counts as effectful**, and so does a
// Procedure whose Steps the walk could not reach in full: the answers above
// state why, and both readings of this value want the same safe one — a Run
// whose blast radius cannot be read may neither share the Store nor keep its
// account to itself until the end.
func effectful(loaded repository.Loaded, walked sequence) bool {
	if !walked.Whole {
		return true
	}
	for _, step := range walked.Steps {
		if effectfulStep(loaded, step) {
			return true
		}
	}
	return false
}
