package run

import "sync"

// Concurrency, and the drain: how much of a `read` Step's Expansion runs at
// once, and what a member that faulted does to the members beside it (§6,
// ADR-0002, ADR-0045, issue #140).
//
// **Concurrency is a function of Kind and is fixed by `hyper`.** A `read`
// Step's Expansion may run concurrently; a `mutate` or `destroy` Expansion runs
// strictly serially. There is no authored knob, no flag and no environment
// override anywhere in it, and this package reads no process fact of its own to
// find one with (run.go).
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
// (issue #143 builds the Patterns onto that ground).
//
// **Within a single `read` Step's Expansion, errors drain.** Every member is
// attempted, every Observation that succeeded is recorded, and the Run then
// halts with the rest of the results already on disk. That is not a preference:
// an Expansion of a `read` runs concurrently and the order its calls complete
// in is defined nowhere, so halting at the *first* failure would make which
// Observations were recorded depend on the one thing nothing may derive from.

// dispatch calls every member of an Expansion and answers what each of them
// concluded, in **Expansion order** and never in the order they answered.
//
// The limit is how many may be in flight at once. Slots are taken in the loop
// rather than inside each call, which is what makes the Expansion order the
// **dispatch** order: member three does not start until a slot is free, and the
// first slots freed go to the members after it in the order the Expansion
// resolved them in — so the first ten of five hundred under a limit of ten are
// the first ten of that order and not the first ten a scheduler happened to
// reach.
//
// **Every member is attempted**, whatever any other member's call did. The
// answers are two slices indexed by member rather than a first fault and a
// short list: what a member concluded and what stopped it are facts about that
// member, and the caller reads them back in Expansion order — which is what
// keeps *which Observations were recorded* and *which fault the Run carries*
// out of the completion order's reach.
//
// A limit below one runs the Expansion one member at a time. `concurrency: 0`
// is a number no Step could dispatch under — the slots would be a channel with
// no room in it and the first member would wait for a receiver that starts only
// once it has been sent — and nothing here judges it: what governs is 1, the
// way it does for the Manifest that declared nothing. Whether a Manifest may
// write it at all is §4's, where the authoring rules are.
func dispatch(limit int, members []member, call func(member) (conclusion, error)) ([]conclusion, []error) {
	concluded := make([]conclusion, len(members))
	faults := make([]error, len(members))

	inFlight := make(chan struct{}, max(limit, 1))
	var running sync.WaitGroup
	for position, resolving := range members {
		inFlight <- struct{}{}
		running.Add(1)
		go func() {
			defer running.Done()
			defer func() { <-inFlight }()
			concluded[position], faults[position] = call(resolving)
		}()
	}
	running.Wait()

	return concluded, faults
}
