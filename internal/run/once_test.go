package run

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **Run-once is a silence on an effectful Operation, and its evidence is two of
// §12's seven Dispositions** (§6, §12, issue #153).
//
// Both are unit tests for repeat_test.go's reason: each answer is a pure
// function of one declared value, where the corpus would need a Manifest per
// Kind and an entry per Disposition to say the same nine and eight things.

// TestRunsOnce_IsTheSilenceOnAnEffectfulOperation walks Repeatability's legality
// table (§12): three Kinds crossed with the values each may declare, and the
// silence beside them. Run-once has no spelling, so the only cell that answers
// yes is the one nobody wrote.
func TestRunsOnce_IsTheSilenceOnAnEffectfulOperation(t *testing.T) {
	for kind, values := range map[string]map[string]bool{
		// A `read`'s undeclared Repeatability is `repeatable` and never
		// run-once: *an effect nobody vouched for* names something a
		// `read` does not do, so the value is inexpressible there
		// rather than defaulted away (§12, ADR-0037).
		"read": {"repeatable": false, "": false},
		// `mutate` is the Kind all three values reach.
		"mutate": {"repeatable": false, "skip-if-recorded": false, "": true},
		// A `destroy` may not declare `skip-if-recorded` — the head its
		// test would find is the live Asset the Step exists to remove —
		// so its silence is run-once like a `mutate`'s (§12, ADR-0037).
		"destroy": {"repeatable": false, "": true},
	} {
		for value, want := range values {
			bound := binding{operation: artefact.OperationInfo{Kind: kind, Repeatability: value}}
			if held := bound.runsOnce(); held != want {
				t.Errorf("a %s declaring repeatability: %q reads %v, want %v", kind, value, held, want)
			}
		}
	}
}

// TestIsEvidence_ReadsTwoOfTheSevenDispositions holds the closed set the walk
// decides on: the two values that say the world was touched or may have been,
// and the five that do not.
//
// *never reached* is in the table though no entry ever holds a record carrying
// it — it is read from the absence of a Step file, which this walk meets as an
// entry yielding nothing (§7, store.Dispositions.Of). It is here so that the
// value the rule is most load-bearing about is stated rather than implied: a
// Step the Journal only ever records that way runs on a re-run, without which
// one run-once Step would make a whole Procedure un-re-runnable after any halt
// (§6, ADR-0001).
func TestIsEvidence_ReadsTwoOfTheSevenDispositions(t *testing.T) {
	for disposition, want := range map[store.Disposition]bool{
		store.DispositionRan:                      true,
		store.DispositionAttemptedOutcomeUnknown:  true,
		store.DispositionAttemptedWorldUntouched:  false,
		store.DispositionNeverReached:             false,
		store.DispositionRefused:                  false,
		store.DispositionSkippedByCondition:       false,
		store.DispositionSkippedAsAlreadyRecorded: false,
	} {
		held := store.Evidence{Step: store.StepFile{Disposition: disposition}}
		if reads := isEvidence(held); reads != want {
			t.Errorf("%s reads %v, want %v", disposition, reads, want)
		}

		// **A rehearsal is no evidence at all**, whatever it recorded:
		// the same record inside a dry-run entry answers no on every
		// one of the seven (§7, ADR-0001).
		held.Entry.RunFile.DryRun = true
		if reads := isEvidence(held); reads {
			t.Errorf("%s inside a rehearsal reads as evidence; a dry-run entry is evidence that a rehearsal happened and of nothing else", disposition)
		}
	}
}
