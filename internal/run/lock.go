package run

import (
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Which lock a Run takes, and why the answer is readable before a Run exists
// (§6, §7, issue #138).

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
// **A Step whose Kind cannot be read is exclusive.** An unresolvable binding
// and a nested invocation each leave this walk with no Kind to judge — the
// first because the Definition or the Operation is not there, the second
// because the invoked Procedure's Steps are not reached until flattening lands
// (issue #141) — and a Run whose blast radius cannot be read is not a Run that
// may share the Store. Neither ever gets as far as its first Step: the first is
// `check`'s to refuse at Run start and the second declines before the Store is
// located. The lock is taken before both, so what it does with them is a fact
// of its own rather than one those two paths make unreachable.
//
// A Procedure name that resolves to nothing is exclusive on the same reading.
// It is unreachable from the CLI, which resolves the positional against the
// namespace first (§9, ADR-0060), and the safe answer costs nothing where
// nothing can reach it.
func LockMode(loaded repository.Loaded, procedure string) store.LockMode {
	file, resolved := loaded.Procedure(procedure)
	if !resolved {
		return store.Exclusive
	}
	for _, step := range readSteps(file) {
		if kindOf(loaded, step) != "read" {
			return store.Exclusive
		}
	}
	return store.Shared
}
