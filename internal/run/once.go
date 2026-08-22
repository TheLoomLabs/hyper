package run

import (
	"fmt"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Repeatability at the Run, the other value that reads something: **run-once**,
// the effectful default, and the Refusal it earns where the Journal already
// holds the Step (§6, §7, §12, ADR-0001, ADR-0062, issue #153).
//
// **It is the value nobody writes.** An Operation declaring no
// `repeatability:` is run-once where its Kind effects, and run-once has no
// spelling of its own — so this is the reading a Manifest gets by saying
// nothing, and it is the strict one: an effect nobody vouched for is not
// repeated on a guess (§12, ADR-0037).
//
// **It refuses on evidence rather than on suspicion, and the evidence is what
// the Journal holds for that Step.** Where no Run records it as *ran* or as
// *attempted, outcome unknown*, it runs; where one does, the Run Refuses
// `run-once-recorded`. Nothing about the world is consulted and nothing about
// the Store's Records is: a run-once Step writes what it likes, or writes
// nothing at all, and what decides a re-run is the Disposition beside it (§6,
// §7, repeat.go).
//
// **Two of §12's seven Dispositions are read as evidence and two are
// deliberately not**, and the exclusions carry the weight.
//
// A Step the Journal only ever records as ***never reached*** runs on a re-run.
// Without that one run-once Step would make a whole Procedure permanently
// un-re-runnable after any halt, and with no bypass the only exit would be an
// edit to a reviewed artefact (§6, ADR-0001).
//
// A Step recorded as ***attempted, world untouched*** runs on a re-run, and it
// is stated rather than left to the set: the request provably never left, so
// nothing happened that a later Run could be evidence of (§6, §12, ADR-0062).
// Both cases the value covers need it. A hostname somebody mistyped invites the
// artefact edit anyway; a firewall that lapsed for ten minutes does not, and an
// evidentiary reading there would leave a Procedure whose every artefact is
// correct permanently un-runnable with nothing to edit.
//
// The two skips are outside the set for their own reasons. *Skipped by
// condition* ran no test and says nothing about what the world holds, and
// *skipped as already recorded* is `skip-if-recorded`'s finding, which is a
// test of the **Store's head version** rather than of anything a Run did (§6,
// ADR-0056, repeat.go).
//
// The second is reachable rather than vacuous, and it is worth saying how. The
// two values are one `repeatability:` key's alternatives, so no Operation is
// both at once — but a Manifest is a file, and an Operation that declared
// `skip-if-recorded` on Tuesday and declares nothing on Wednesday leaves a
// Journal holding that Disposition for a Step that is now run-once. What the
// entry then records is that a Run found an Asset standing, which is a fact
// about the branch and not about an effect, so it is not evidence here either.
//
// **A rehearsal counts as no evidence at all.** An entry a dry-run wrote is
// evidence that a rehearsal happened and evidence of nothing else, and this
// walk filters it out on the rule the identity digest's back-walk already
// applies one file over (§7, ADR-0001, step.go). A rehearsal that counted would
// permanently refuse every run-once Step in the Procedure it rehearsed, with no
// bypass to recover through: the review aid would disarm the tool.
//
// **The Refusal is the Run's and it is terminal**, which is what a run-once
// Step and a Cadence being refused together is about: nobody is present to read
// it, so the Procedure's remaining Steps stop with it at every occurrence after
// the first. `check` refuses that combination before either runs
// (`cadence-run-once`), which is what leaves this value meaning what it says on
// a Procedure a person invokes, and it is why nothing is re-read here (§4, §11,
// ADR-0038).

// CodeRunOnceRecorded is §6's second run-time check: a run-once Step the
// Journal already holds as *ran* or as *attempted, outcome unknown*.
//
// It is one of the few members of the closed set that require a Step to have
// been reached at all, and like every one of them it declines **before a call
// goes out** — a guardrail that declines after one is a halt and has no
// `error_code` to carry (§12, ADR-0072).
const CodeRunOnceRecorded = "run-once-recorded"

// runsOnce says this Step's Operation is the effectful default.
//
// It reads internal/artefact's own answer rather than restating the two
// conditions, which is what keeps *what run-once is* one sentence in the
// corpus: `check` refuses a Cadence over exactly the Operations this reports,
// and a Run refusing a different set would be two readings of one silence (§4,
// §12, artefact.OperationInfo.IsRunOnce).
func (b binding) runsOnce() bool { return b.operation.IsRunOnce() }

// recordedAlready is the Refusal a run-once Step earns where the Journal holds
// it, and nothing at all where it does not — which, on an Operation that
// declares a Repeatability at all, is every Step.
//
// It decides **before the Expansion resolves**. §12 puts all five of its
// Step-scoped codes *at or before* Expansion and this is the one that is
// before, which is a position rather than a convenience: what the Journal holds
// for this Step is answerable with no selector resolved, and resolving one
// first would let the branch the Expansion reads decide **which** Refusal a
// Step that has already run earns (§6, §12, expand.go).
//
// It decides **after the condition**, which is the other half of the same
// sentence. A Step whose `when:` does not hold makes no call, and Refusing one
// that was going to be skipped anyway would end the Run over an effect nobody
// was about to repeat (§6, condition.go).
func (r run) recordedAlready(bound binding, authored sequenced, position int) ([]Refusal, error) {
	if !bound.runsOnce() {
		return nil, nil
	}

	held, found, err := r.recordedBy(authored)
	if err != nil || !found {
		return nil, err
	}
	cited := r.citation(authored, position, selector{})
	return []Refusal{r.refusal(CodeRunOnceRecorded,
		fmt.Sprintf("%s binds %s, which is run-once, and run %s records it as %s",
			named(authored), authored.Operation, held.Entry.Run, held.Step.Disposition),
		cited.wholeStep())}, nil
}

// recordedBy is the Run whose entry holds this Step as *ran* or as *attempted,
// outcome unknown*, and whether the Journal holds one at all.
//
// It is a backward walk over the Journal's date partitions and it **stops at
// the first record it finds carrying either value**, which is what makes the
// scan cheap: a Step the last Run ran costs that entry's files, and a Step
// nothing ever ran costs the Journal (§7, store.Scan).
//
// **It matches on the authored `id` and on nothing else**, the way the identity
// digest's back-walk matches a Step: an `id` that moved is a different Step,
// with no evidence behind it. Where the two walks part is the filters they put
// in front of that match, and the reason is what each of them is deciding.
//
// The digest's comparand is narrowed to this Procedure and this invocation
// chain, because a set read off another Step's entry would be written down as
// this Step's (§7, ADR-0055). Neither narrowing is applied here, and the second
// would be **wrong**: a Step of a nested Procedure carries a path when the Run
// reached it through an invocation and carries none when the Run named that
// Procedure directly, so filtering on one would answer *no evidence* for the
// Step that ran an hour ago under another Procedure's name — a repeated effect,
// which is the one answer this value exists to prevent. What the wider match
// costs is the other direction and it is the safe one: two Procedures declaring
// one `id` are one Step to this walk, and the second of them Refuses.
//
// **A rehearsal is filtered out**, and that filter is this reading's rather
// than the walk's: internal/store reaches every entry and reports each one's
// `dry_run` marker, because which entries a reading keeps is its own (§7,
// ADR-0001).
//
// **This Run's own entry is reached like any other**, and the entry being open
// changes nothing: the walk reads the records an entry holds and never infers
// one from a silence (§7, store.Dispositions.Of). A Procedure invoking one
// run-once Step twice in a single Run therefore Refuses at the second
// occurrence, the first having written its file before the second's turn came —
// which is the same effect repeated, and this value's own rule arriving at it
// (§6, sequence.go).
func (r run) recordedBy(authored sequenced) (store.Evidence, bool, error) {
	for held, err := range r.request.Store.Scan(authored.ID) {
		if err != nil {
			return store.Evidence{}, false, err
		}
		if isEvidence(held) {
			return held, true, nil
		}
	}
	return store.Evidence{}, false, nil
}

// isEvidence says one entry's record of the Step is Repeatability evidence:
// **a Run that touched the world, or may have.**
//
// Two of §12's seven Dispositions are that, and it is the whole of the rule.
// *ran* is a Step that reached a conclusion `hyper` recorded, and *attempted,
// outcome unknown* attaches the doubt to the attempt — the call went out and no
// answer came back, so a later Run may not treat it as either success or
// failure, and repeating the effect is the reading run-once declines (§12,
// ADR-0018).
//
// Of the five outside the set, *never reached* and *attempted, world untouched*
// are the two the value's own text names, each for a reason this file's header
// states.
// *refused* joins them by the same rule — a guardrail declined before any
// effect reached the world — and the two skips by theirs: *skipped by
// condition* ran no test, and *skipped as already recorded* is
// `skip-if-recorded`'s finding, which a run-once Step meets only across a
// Manifest that changed and which is a fact about the branch either way (§6,
// §12, ADR-0056, ADR-0062).
//
// **A rehearsal is none of it**, whatever it recorded. An entry a dry-run wrote
// is evidence that a rehearsal happened and evidence of nothing else, and the
// marker is read here rather than trusted to be absent: `dry_run` is written on
// every entry, `false` included, for exactly this reader (§7, ADR-0001).
//
// No rehearsal **this** binary runs can reach the state that would matter:
// run-once is effectful-only and a rehearsal stops at the first effectful Step,
// so a rehearsal records no run-once Step even once (§9, ADR-0010, run.go).
// What this line still filters is an entry written by something else — a binary
// before issue #155, a hand-edited branch — and it stays because the cost of
// getting it wrong is unrecoverable while the cost of holding it is one read of
// a marker that is always there (§7, ADR-0001).
func isEvidence(held store.Evidence) bool {
	if held.Entry.RunFile.DryRun {
		return false
	}
	switch held.Step.Disposition {
	case store.DispositionRan, store.DispositionAttemptedOutcomeUnknown:
		return true
	default:
		return false
	}
}
