package store_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The open entries a reaper finds (§6, §7, ADR-0076, issue #154).
//
// Every case here seeds the account by **presence** and never by a field: an
// entry holding neither an outcome.json its own Run wrote nor a closing write
// another Run wrote is open, and that absence is the whole representation.

// runsOfOpen is a listing read off as the one thing a case is about, so that an
// assertion reads as the order it states.
func runsOfOpen(entries []store.OpenEntry) []string {
	runs := make([]string, len(entries))
	for i, entry := range entries {
		runs[i] = entry.Run.String()
	}
	return runs
}

// closedBy is a closing write as a case seeds one: the shape a reaper writes,
// under whatever Step it inferred.
func closedBy(at time.Time, step int) store.ClosedBy {
	return store.ClosedBy{EndedAt: at, Step: step}
}

// TestOpenEntries_AreTheEntriesHoldingNoAccountAtAll is the classification the
// reap is quantified over, seeded once in each of §7's four forms: the open one
// is answered and the three that hold an account are not.
func TestOpenEntries_AreTheEntriesHoldingNoAccountAtAll(t *testing.T) {
	completed := store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Minute)}
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart)},
		anEntry{run: runFileAt(t, theCloserRunID, theRunStart.Add(-time.Hour)), outcome: &completed},
		anEntry{
			run:     runFileAt(t, theSecondCloserRunID, theRunStart.Add(-2*time.Hour)),
			closers: map[string]store.ClosedBy{theEntryRunID: closedBy(theRunStart, 4)},
		},
		anEntry{
			run:     runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)),
			outcome: &completed,
			closers: map[string]store.ClosedBy{theEntryRunID: closedBy(theRunStart, 4)},
		},
	)

	open, err := held.OpenEntries()
	if err != nil {
		t.Fatalf("OpenEntries: %v", err)
	}
	if got := runsOfOpen(open); !slices.Equal(got, []string{theEntryRunID}) {
		t.Errorf("the branch answers %v open, want only %s — an entry holding an account of either form is closed", got, theEntryRunID)
	}
}

// TestOpenEntries_AreOrderedNewestFirst is Entries' own order at the reap's
// grain: the instant each entry's own run.json carries, ties broken by the Run
// id. Every one of them is reaped, so the order decides nothing about which —
// what it decides is that two reads of one branch answer the same sequence.
func TestOpenEntries_AreOrderedNewestFirst(t *testing.T) {
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theCloserRunID, theRunStart.Add(-time.Hour))},
		anEntry{run: runFileAt(t, theSecondCloserRunID, theRunStart)},
		anEntry{run: runFileAt(t, theYearBeforeRunID, theRunStart.AddDate(-1, 0, 0))},
	)

	open, err := held.OpenEntries()
	if err != nil {
		t.Fatalf("OpenEntries: %v", err)
	}
	want := []string{theSecondCloserRunID, theCloserRunID, theYearBeforeRunID}
	if got := runsOfOpen(open); !slices.Equal(got, want) {
		t.Errorf("the branch answers %v, want %v newest first", got, want)
	}
}

// TestOpenEntries_CarryTheHighestStepOrdinalPresent is the arithmetic §7 fixes
// for the Step a reaper names: the highest <nnnn> under steps/ is the last Step
// that finished, so the closing write's is the one after it.
//
// It is the ordinal present and never a count of the files: the two agree on
// every entry `hyper` writes, and the reader that took the count would be the
// one that stopped agreeing the day a Run wrote them out of order.
func TestOpenEntries_CarryTheHighestStepOrdinalPresent(t *testing.T) {
	second, third := stepFile(), stepFile()
	second.Step, third.Step = 2, 3

	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theSecondCloserRunID, theRunStart), steps: []store.StepFile{third, second}},
		anEntry{run: runFileAt(t, theCloserRunID, theRunStart.Add(-time.Hour))},
	)

	open, err := held.OpenEntries()
	if err != nil {
		t.Fatalf("OpenEntries: %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("the branch answers %d open entries, want 2", len(open))
	}
	if open[0].Last != 3 {
		t.Errorf("the entry holding steps 2 and 3 answers %d, want 3 — the highest ordinal present", open[0].Last)
	}
	// A killed Run leaving no Step file at all went quiet on Step 1, which
	// is this answer plus one and needs no case of its own in the reaper.
	if open[1].Last != 0 {
		t.Errorf("the entry holding no Step file answers %d, want 0", open[1].Last)
	}
}

// TestOpenEntries_CarryWhatTheDeadRunsOwnRunFileSays is the rest of what a
// reaper reads off an entry it did not write: the Procedure the dead Run was
// performing and the **repository** revision to load it at, which is what makes
// *which Step was it* derived rather than guessed (§7).
func TestOpenEntries_CarryWhatTheDeadRunsOwnRunFileSays(t *testing.T) {
	_, held := seededJournal(t, anEntry{run: runFileAt(t, theEntryRunID, theRunStart)})

	open, err := held.OpenEntries()
	if err != nil {
		t.Fatalf("OpenEntries: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("the branch answers %d open entries, want 1", len(open))
	}
	if got, want := open[0].Procedure, runFile(t).Procedure; got != want {
		t.Errorf("the entry names procedure %q, want %q", got, want)
	}
	if got, want := open[0].Provenance.RepoRevision, theProvenance.Run.RepoRevision; got != want {
		t.Errorf("the entry names repo_revision %q, want %q", got, want)
	}
	if got, want := open[0].At().RunPath(), "journal/2026/08/06/"+theEntryRunID+"/run.json"; got != want {
		t.Errorf("the entry sits at %q, want %q", got, want)
	}
}

// TestOpenEntries_OfABranchHoldingNoJournalIsEmpty is the first Run against a
// Store nobody has run against: a reap over an empty Journal reaps nothing, and
// answers so without a case in the reader for it.
func TestOpenEntries_OfABranchHoldingNoJournalIsEmpty(t *testing.T) {
	_, held := seededJournal(t)

	open, err := held.OpenEntries()
	if err != nil {
		t.Fatalf("OpenEntries: %v", err)
	}
	if len(open) != 0 {
		t.Errorf("the branch answers %v open, want nothing", open)
	}
}

// TestOpenEntries_TellsAFileItCannotReadFromAPathThatDisagrees is the split a
// reap's tolerance turns on, and the reason it is a named error rather than a
// message (entries.go, internal/run/reap.go).
//
// **A file that would not decode is ErrUnreadable**, because §6 puts a gate one
// place further into a Run whose whole job is to report exactly that over the
// Journal — so a reap that met it may leave every entry open and let the gate
// speak. **A path that disagrees with the file standing at it is not**: no gate
// reports it, so a reap that treated it as tolerable would be a Run completing
// having quietly reaped nothing.
func TestOpenEntries_TellsAFileItCannotReadFromAPathThatDisagrees(t *testing.T) {
	open := runFileAt(t, theEntryRunID, theRunStart)
	at := open.At()

	for name, seeded := range map[string]struct {
		files      map[string]string
		unreadable bool
	}{
		"a run.json written above this binary's ceiling": {
			files:      map[string]string{at.RunPath(): bumped(string(open.Encode()))},
			unreadable: true,
		},
		"an entry filed under a date its own run.json does not build": {
			files: map[string]string{
				"journal/2019/07/04/" + theEntryRunID + "/run.json": string(open.Encode()),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := seededJournal(t)
			r.seedFiles(r.root, seeded.files)
			held, err := store.Open(r.root, theInstant)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}

			_, err = held.OpenEntries()
			if err == nil {
				t.Fatal("the branch answered its open entries, and one of its files cannot be read as it stands")
			}
			if errors.Is(err, store.ErrUnreadable) != seeded.unreadable {
				t.Errorf("the read answered %v; want ErrUnreadable to read %v", err, seeded.unreadable)
			}
		})
	}
}
