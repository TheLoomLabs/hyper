package run

import (
	"iter"
	"sync"
)

// Concurrency, and the drain: how much of a `read` Step's Expansion runs at
// once, and what a member that faulted does to the members beside it (§6,
// ADR-0002, ADR-0045, issue #140).
//
// **Concurrency is a function of Kind and is fixed by `hyper`.** A `read`
// Step's Expansion may run concurrently; a `mutate` or `destroy` Expansion runs
// strictly serially. There is no authored knob, no flag and no environment
// override anywhere in it, and this package reads no process fact of its own to
// find one with (run.go). Which of the two a Step gets is decided off its Kind
// one file over and arrives here as a number, so *serial* and *limit one* are
// one mechanism rather than a second code path (effect.go).
//
// **How much runs at once is the Operation's Manifest-declared `concurrency:`
// limit**, since the Provider author is the one who knows where the API
// refuses — and a Manifest declaring none declares 1, so a `read` whose
// Provider author said nothing runs its Expansion serially as well. The number
// arrives here already effective: internal/artefact answers 1 for a `read` that
// omits the key and 1 for every Kind that may not declare one, so this file
// takes a limit rather than a Manifest and there is one place the rule lives.
//
// **All concurrency lives inside one Step's Expansion; two Steps never
// overlap.** dispatch returns when every member it started has answered, and
// Perform reaches the next Step only after this one wrote its file — so the
// limit bounds the members of one Step's Expansion that are in flight at once
// and nothing else. It does not reach a Pattern either: pagination, polling and
// retry are serial by construction, so a member is one call at a time from the
// moment it is dispatched until its last page, and *members in flight* and
// *requests in flight* are one number rather than two that would have to agree
// (pattern.go, issue #143).
//
// **Within a single `read` Step's Expansion, errors drain.** Every member is
// attempted, every Observation that succeeded is recorded, and the Run then
// halts with the rest of the results already on disk. That is not a preference:
// an Expansion of a `read` runs concurrently and the order its calls complete
// in is defined nowhere, so halting at the *first* failure would make which
// Observations were recorded depend on the one thing nothing may derive from.
//
// **An effectful Expansion stops at the first error instead**, everything it
// confirmed already committed — and pushed at the Step boundary the Run's own
// rhythm fixes, a write being committed as the call that produced it confirms
// and pushed after every effectful Step (§7, ADR-0006, run.go). It can stop
// because it is serial: *which three of the five* is a determinate fact rather
// than a race.
//
// Both rules are stated here and **taken** one file over. What dispatch decides
// is how many members run at once and in what order they are handed back; a
// caller that stops taking them starts no more of them, so *drain* and *stop*
// are one sequence read two ways rather than two code paths (§6, step.go).

// dispatch calls every member of an Expansion and hands each one back in
// **Expansion order**, never in the order they answered.
//
// The limit is how many may be in flight at once, and a **slot is held until
// the member it belongs to has been taken**. That is the one sentence the two
// rhythms are read off. Under a limit of one it means the next member is called
// only once the caller has finished with the last, so an effectful Step's
// version is committed before its next call goes out (§7); under a higher limit
// it means the first ten of five hundred are the first ten of the Expansion
// order and not the first ten a scheduler happened to reach.
//
// A caller that stops taking members starts no more of them: the loop that
// takes the slots is the loop that yields, so *stopping at the first error* is
// the caller breaking and nothing here. Every member already in flight is
// waited for before the sequence ends, so a Run never carries on with a call of
// its own still outstanding.
//
// It is generic in what a member's call answers because what that is belongs to
// the caller: this file decides how many members run at once and in what order
// they are read back, and a Step's own answer — the Records it projected and
// the account of the Patterns that reached them — is step.go's (issue #143).
//
// A limit below one runs the Expansion one member at a time. `concurrency: 0`
// is a number no Step could dispatch under and nothing here judges it: what
// governs is 1, the way it does for the Manifest that declared nothing. Whether
// a Manifest may write it at all is §4's, where the authoring rules are.
func dispatch[T any](limit int, members []member, call func(member) (T, error)) iter.Seq[taken[T]] {
	return func(yield func(taken[T]) bool) {
		answered := make([]chan taken[T], len(members))
		for at := range answered {
			// One slot each, so a call that answered never waits
			// for its member's turn to come round: what the limit
			// bounds is how many calls are in flight and not how
			// many answers may be held.
			answered[at] = make(chan taken[T], 1)
		}

		// Every call started is waited for, whether or not its answer
		// was ever taken — a Run that returned with one outstanding
		// would be a Run whose next Step overlapped this one's last
		// call (ADR-0002).
		var running sync.WaitGroup
		defer running.Wait()

		inFlight, started, room := 0, 0, max(limit, 1)
		for at := range members {
			for started < len(members) && inFlight < room {
				position, resolving := started, members[started]
				running.Add(1)
				go func() {
					defer running.Done()
					concluded, fault := call(resolving)
					answered[position] <- taken[T]{At: position, Concluded: concluded, Fault: fault}
				}()
				inFlight, started = inFlight+1, started+1
			}

			held := <-answered[at]
			inFlight--
			if !yield(held) {
				return
			}
		}
	}
}

// taken is one member of an Expansion as the walk hands it back: which member
// it was, what its call concluded, and what stopped it.
//
// The three travel together because a caller reading them apart is a caller
// that can read one member's fault against another's answer — which is exactly
// the mistake Expansion order exists to make impossible.
type taken[T any] struct {
	// At is the member's position in the Expansion, from 0, which is what
	// names it in `expanded_to` and what a collision reports it by.
	At int
	// Concluded is what the call answered, and the zero value where it
	// faulted before answering anything.
	Concluded T
	// Fault is what stopped this member, and nil where nothing did.
	Fault error
}
