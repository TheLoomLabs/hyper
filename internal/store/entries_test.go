package store_test

import (
	"errors"
	"iter"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Journal, read (issue #132). Every case here seeds files under an entry's
// own directory and reads them back, because an entry's account is a
// classification over the files present under it and nothing else — there is no
// state key to seed and no growing file to rewrite (§7, ADR-0011).
//
// Nothing here is defined over the branch's commits. A case that seeds four
// entries in one commit and a case that seeds them in four are the same case,
// and one of them below says so.

// The Run ids the entries are seeded under. They are further UUIDv7s of §7's
// own, ordered as their text is, so a case that turns on the ordering can state
// which entry comes first.
const (
	theCloserRunID       = "01991e24-6f2c-7e37-a04b-8fa05dc27316"
	theSecondCloserRunID = "01991e25-703d-7f48-b15c-90b16ed28427"
	theDayBeforeRunID    = "01991e26-814e-7059-8260-a1c27fe39538"
	theMonthBeforeRunID  = "01991e27-925f-716a-9371-b2d380f4a649"
	theYearBeforeRunID   = "01991e28-a360-727b-a482-c3e491056b5a"
)

// anEntry is one Journal entry a case seeds: the files under one Run's
// directory, in the shapes §7 fixes, so what a case puts on the branch is what
// `hyper` would have written there.
//
// The account is seeded by presence and never by a field: an entry with neither
// an outcome nor a closer is open, and that absence is the whole of it.
type anEntry struct {
	run     store.RunFile
	steps   []store.StepFile
	outcome *store.OutcomeFile
	closers map[string]store.ClosedBy
}

// runFileAt is §7's own run.json under another id and another start, which is
// what every entry a case seeds varies.
func runFileAt(t *testing.T, run string, started time.Time) store.RunFile {
	t.Helper()

	file := runFile(t)
	file.Run = runID(t, run)
	file.StartedAt = started
	return file
}

// files is the entry as the branch holds it: one path per file, each built by
// the grammar from the entry's own Run and start (§12).
func (e anEntry) files(t *testing.T) map[string]string {
	t.Helper()

	at := store.JournalEntry{Run: e.run.Run, Started: e.run.StartedAt}
	files := map[string]string{at.RunPath(): string(e.run.Encode())}
	for _, step := range e.steps {
		files[at.StepPath(step.Step)] = string(step.Encode())
	}
	if e.outcome != nil {
		files[at.OutcomePath()] = string(e.outcome.Encode())
	}
	for closer, closed := range e.closers {
		files[at.ClosedByPath(runID(t, closer))] = string(closed.Encode())
	}
	return files
}

// seededJournal is a repository whose Store branch holds the entries handed,
// and the handle on it. Every entry goes into one commit, which is what makes
// the commit-shaped case below a case at all.
func seededJournal(t *testing.T, entries ...anEntry) (*repo, *store.Store) {
	t.Helper()

	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(entries) > 0 {
		files := map[string]string{}
		for _, entry := range entries {
			for path, content := range entry.files(t) {
				files[path] = content
			}
		}
		r.seedFiles(r.root, files)
	}
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return r, held
}

// oneEntry reads back the one entry a case seeded, which most of them do.
func oneEntry(t *testing.T, held *store.Store, run string) store.Entry {
	t.Helper()

	entry, found, err := held.Entry(runID(t, run))
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !found {
		t.Fatalf("the branch holds no entry for %s, want the one seeded", run)
	}
	return entry
}

// runsOfEntries is a listing read off as the one thing a case is about, so that
// an assertion reads as the order it states.
func runsOfEntries(entries []store.Entry) []string {
	runs := make([]string, len(entries))
	for i, entry := range entries {
		runs[i] = entry.Run.String()
	}
	return runs
}

// TestEntries_ListsTheEntriesTheBranchHoldsNewestFirst is the listing: every
// entry under every date partition, ordered on the instant each one's own
// run.json carries. The entries are seeded across a day, a month and a year
// boundary, which is what the backward scan below walks.
func TestEntries_ListsTheEntriesTheBranchHoldsNewestFirst(t *testing.T) {
	// Two of them share the newest day, the later one seeded under the id
	// that sorts second, so the ordering inside a partition is the instant
	// each run.json carries and not the directory the file was found in.
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theCloserRunID, theRunStart.Add(-time.Hour))},
		anEntry{run: runFileAt(t, theSecondCloserRunID, theRunStart)},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1))},
		anEntry{run: runFileAt(t, theMonthBeforeRunID, theRunStart.AddDate(0, -1, 0))},
		anEntry{run: runFileAt(t, theYearBeforeRunID, theRunStart.AddDate(-1, 0, 0))},
	)

	entries, err := held.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	want := []string{theSecondCloserRunID, theCloserRunID, theDayBeforeRunID, theMonthBeforeRunID, theYearBeforeRunID}
	if got := runsOfEntries(entries); !slices.Equal(got, want) {
		t.Errorf("the branch holds %v, want %v newest first", got, want)
	}
}

func TestEntries_OfABranchHoldingNoJournalIsEmpty(t *testing.T) {
	_, held := seededJournal(t)

	entries, err := held.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the branch holds %v, want nothing", entries)
	}
}

// TestEntry_ReadsOneEntryWholeByItsRunID is the second half of the reader's
// listing: an entry named by its Run rather than found by walking, and the Step
// records under it.
func TestEntry_ReadsOneEntryWholeByItsRunID(t *testing.T) {
	step := stepFile()
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1))},
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{step}},
	)

	entry := oneEntry(t, held, theEntryRunID)
	if entry.Procedure != "retire-preview-dns" || !entry.StartedAt.Equal(theRunStart) {
		t.Errorf("the entry is %+v, want the run.json seeded under %s", entry.RunFile, theEntryRunID)
	}

	dispositions, err := held.Dispositions(entry)
	if err != nil {
		t.Fatalf("Dispositions: %v", err)
	}
	if len(dispositions.Steps) != 1 || !reflect.DeepEqual(dispositions.Steps[0], step) {
		t.Errorf("the entry holds %+v, want the one Step file seeded", dispositions.Steps)
	}
}

func TestEntry_AnswersNothingForARunTheBranchDoesNotHold(t *testing.T) {
	_, held := seededJournal(t, anEntry{run: runFileAt(t, theEntryRunID, theRunStart)})

	_, found, err := held.Entry(runID(t, theSecondRunID))
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if found {
		t.Error("the branch answered an entry for a Run it holds none for")
	}
}

// TestEntry_HoldingNoAccountAtAllIsOpen. Neither an outcome.json its own Run
// wrote nor a closing write another Run wrote, and that absence is the whole
// representation: nothing infers an outcome, nothing names an instant, and no
// duration derives (§7).
func TestEntry_HoldingNoAccountAtAllIsOpen(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:   runFileAt(t, theEntryRunID, theRunStart),
		steps: []store.StepFile{stepFile()},
	})

	entry := oneEntry(t, held, theEntryRunID)
	if entry.Account() != store.AccountOpen {
		t.Errorf("the entry is %v, want open where it holds no account at all", entry.Account())
	}
	if outcome, closed := entry.Outcome(); closed {
		t.Errorf("the entry claims an outcome of %q; an open entry has none and nothing infers one", outcome)
	}
	if ended, closed := entry.Ended(); closed {
		t.Errorf("the entry claims it ended at %s; the Run may be in flight and hyper never guesses", ended)
	}
	if _, derives := entry.Duration(); derives {
		t.Error("a duration derives from an open entry; there is no instant to subtract from")
	}
}

// TestEntry_HoldingItsOwnOutcomeIsClosedByItsOwnRun.
func TestEntry_HoldingItsOwnOutcomeIsClosedByItsOwnRun(t *testing.T) {
	ended := theRunStart.Add(2*time.Minute + 31*time.Second)
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: ended},
	})

	entry := oneEntry(t, held, theEntryRunID)
	if entry.Account() != store.AccountOwn {
		t.Errorf("the entry is %v, want closed by its own Run", entry.Account())
	}
	if outcome, closed := entry.Outcome(); !closed || outcome != store.OutcomeCompleted {
		t.Errorf("the outcome is %q (%t), want the file's own %q", outcome, closed, store.OutcomeCompleted)
	}
	if at, closed := entry.Ended(); !closed || !at.Equal(ended) {
		t.Errorf("the entry ended at %s (%t), want the file's own %s", at, closed, ended)
	}
	if took, derives := entry.Duration(); !derives || took != ended.Sub(theRunStart) {
		t.Errorf("the duration is %s (%t), want %s derived inside the one entry", took, derives, ended.Sub(theRunStart))
	}
}

// TestEntry_HoldingClosingWritesAloneIsReaped. The Run really did not come
// back: the entry is `failed`, and where several closers landed the close
// instant is the earliest `ended_at` among them — the first inference, later
// ones adding nothing but their own existence (§7).
func TestEntry_HoldingClosingWritesAloneIsReaped(t *testing.T) {
	first := theRunStart.Add(31 * time.Minute)
	second := theRunStart.Add(48 * time.Minute)
	_, held := seededJournal(t, anEntry{
		run:   runFileAt(t, theEntryRunID, theRunStart),
		steps: []store.StepFile{stepFile()},
		closers: map[string]store.ClosedBy{
			// The later inference is seeded under the id that sorts
			// first, so a reader taking the file name for the
			// ordering would fail here.
			theCloserRunID:       {EndedAt: second, Step: 4, StepCode: store.StepCode{ID: "publish"}},
			theSecondCloserRunID: {EndedAt: first, Step: 4, StepCode: store.StepCode{ID: "publish"}},
		},
	})

	entry := oneEntry(t, held, theEntryRunID)
	if entry.Account() != store.AccountReaped {
		t.Errorf("the entry is %v, want reaped", entry.Account())
	}
	if outcome, closed := entry.Outcome(); !closed || outcome != store.OutcomeFailed {
		t.Errorf("the outcome is %q (%t), want %q", outcome, closed, store.OutcomeFailed)
	}
	if at, closed := entry.Ended(); !closed || !at.Equal(first) {
		t.Errorf("the entry closed at %s (%t), want the earliest inference %s", at, closed, first)
	}
}

// TestEntry_ReportsEveryClosingWriteAndDiscardsNone. However many landed, all
// of them stand and none is removed (§7).
func TestEntry_ReportsEveryClosingWriteAndDiscardsNone(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run: runFileAt(t, theEntryRunID, theRunStart),
		closers: map[string]store.ClosedBy{
			theCloserRunID:       {EndedAt: theRunStart.Add(48 * time.Minute), Step: 4},
			theSecondCloserRunID: {EndedAt: theRunStart.Add(31 * time.Minute), Step: 4},
		},
	})

	entry := oneEntry(t, held, theEntryRunID)
	if len(entry.Closers) != 2 {
		t.Fatalf("the entry reports %d closing writes, want both", len(entry.Closers))
	}
	// Earliest first: the one the account is drawn from is the one at hand,
	// and the ordering is the rule rather than a second lookup.
	if got := []string{entry.Closers[0].Run.String(), entry.Closers[1].Run.String()}; !slices.Equal(got, []string{theSecondCloserRunID, theCloserRunID}) {
		t.Errorf("the closers are %v, want the earliest inference first", got)
	}
	for _, closer := range entry.Closers {
		if closer.Step != 4 {
			t.Errorf("the closer %s names Step %d, want the Step the dead Run went quiet on", closer.Run, closer.Step)
		}
	}
}

// TestEntry_HoldingBothIsContestedAndTakesTheOwnersOutcome. It is what a reap
// of a Run that was alive after all leaves behind. The entry's outcome is the
// owner's wherever one exists, the inference stays true of the Run that drew
// it, and neither file is removed (§7, ADR-0076).
func TestEntry_HoldingBothIsContestedAndTakesTheOwnersOutcome(t *testing.T) {
	owned := theRunStart.Add(52 * time.Minute)
	inferred := theRunStart.Add(31 * time.Minute)
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: owned},
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: inferred, Step: 4}},
	})

	entry := oneEntry(t, held, theEntryRunID)
	if entry.Account() != store.AccountContested {
		t.Errorf("the entry is %v, want contested", entry.Account())
	}
	if outcome, closed := entry.Outcome(); !closed || outcome != store.OutcomeCompleted {
		t.Errorf("the outcome is %q (%t), want the owner's observation %q", outcome, closed, store.OutcomeCompleted)
	}
	if at, closed := entry.Ended(); !closed || !at.Equal(owned) {
		t.Errorf("the entry ended at %s (%t), want the owner's own %s", at, closed, owned)
	}
	if len(entry.Closers) != 1 || !entry.Closers[0].EndedAt.Equal(inferred) {
		t.Errorf("the entry reports %+v, want the closer's inference standing alongside the owner's account", entry.Closers)
	}
}

// TestEntry_ReapedDerivesNoDurationAndContestedDerivesOne. A reaped entry's
// `ended_at` is the closing Run's instant on the closing Run's clock, so
// subtracting the dead Run's `started_at` from it is the cross-entry
// subtraction §7 forbids, wearing one entry's directory. The reader surfaces
// the fact and the flag together, which is what keeps them from coming apart.
func TestEntry_ReapedDerivesNoDurationAndContestedDerivesOne(t *testing.T) {
	owned := &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(52 * time.Minute)}
	closers := map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(31 * time.Minute), Step: 4}}

	for name, tc := range map[string]struct {
		entry   anEntry
		derives bool
	}{
		"reaped": {
			entry:   anEntry{run: runFileAt(t, theEntryRunID, theRunStart), closers: closers},
			derives: false,
		},
		"contested": {
			entry:   anEntry{run: runFileAt(t, theEntryRunID, theRunStart), outcome: owned, closers: closers},
			derives: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, held := seededJournal(t, tc.entry)

			took, derives := oneEntry(t, held, theEntryRunID).Duration()
			if derives != tc.derives {
				t.Fatalf("a duration derives = %t, want %t", derives, tc.derives)
			}
			if derives && took != owned.EndedAt.Sub(theRunStart) {
				t.Errorf("the duration is %s, want %s taken from the owner's file", took, owned.EndedAt.Sub(theRunStart))
			}
		})
	}
}

// TestEntry_ExposesDryRunOnEveryEntryAndFiltersNothing. Four consumers filter
// rehearsals out and which readings exclude one is each consumer's, so the
// reader reports the marker and never acts on it (§7, ADR-0001).
func TestEntry_ExposesDryRunOnEveryEntryAndFiltersNothing(t *testing.T) {
	rehearsal := runFileAt(t, theEntryRunID, theRunStart)
	rehearsal.DryRun = true
	_, held := seededJournal(t,
		anEntry{run: rehearsal},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1))},
	)

	entries, err := held.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("the branch holds %d entries, want the rehearsal listed beside the Run", len(entries))
	}
	if !entries[0].DryRun {
		t.Errorf("the entry under %s reports dry_run = false, want the rehearsal it was seeded as", theEntryRunID)
	}
	if entries[1].DryRun {
		t.Errorf("the entry under %s reports dry_run = true, want the ordinary Run it was seeded as", theDayBeforeRunID)
	}
}

// TestDispositions_ReadsAStepsDispositionFromItsFile.
func TestDispositions_ReadsAStepsDispositionFromItsFile(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{stepFile()},
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Minute)},
	})

	dispositions := dispositionsOf(t, held, theEntryRunID)
	if got, held := dispositions.Of("retire"); !held || got != store.DispositionRan {
		t.Errorf("the Step `retire` is %q (%t), want its own file's %q", got, held, store.DispositionRan)
	}
}

// TestDispositions_ReadsNeverReachedFromASilenceInsideAClosedEntry. Seven
// Dispositions, six borne by a file and one read from a silence: a forty-Step
// Procedure that halted at Step 3 writes no file for the thirty-seven that
// never happened (§7, §12).
func TestDispositions_ReadsNeverReachedFromASilenceInsideAClosedEntry(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{stepFile()},
		outcome: &store.OutcomeFile{Outcome: store.OutcomeFailed, EndedAt: theRunStart.Add(time.Minute)},
	})

	dispositions := dispositionsOf(t, held, theEntryRunID)
	if got, held := dispositions.Of("publish"); !held || got != store.DispositionNeverReached {
		t.Errorf("the Step `publish` is %q (%t), want %q read from the silence", got, held, store.DispositionNeverReached)
	}
}

// TestDispositions_AnswerNothingForAStepAbsentFromAnOpenEntry. The absence
// means something different there and guessing is exactly what §7 forbids.
func TestDispositions_AnswerNothingForAStepAbsentFromAnOpenEntry(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:   runFileAt(t, theEntryRunID, theRunStart),
		steps: []store.StepFile{stepFile()},
	})

	dispositions := dispositionsOf(t, held, theEntryRunID)
	if got, held := dispositions.Of("publish"); held {
		t.Errorf("the Step `publish` is %q in an open entry, want no answer at all", got)
	}
	if got, held := dispositions.Of("retire"); !held || got != store.DispositionRan {
		t.Errorf("the Step `retire` is %q (%t), want the file it does hold to answer", got, held)
	}
}

// TestDispositions_DecodeEveryOneOfTheSeven. Six from a Step file, one from a
// silence, and `attempted-outcome-unknown` from a closing write as well — which
// is the value §6's rule lands on, without which a crashed Step reads as never
// reached and an effect nobody vouched for is re-run.
func TestDispositions_DecodeEveryOneOfTheSeven(t *testing.T) {
	borne := []store.Disposition{
		store.DispositionRan,
		store.DispositionSkippedAsAlreadyRecorded,
		store.DispositionSkippedByCondition,
		store.DispositionRefused,
		store.DispositionAttemptedOutcomeUnknown,
		store.DispositionAttemptedWorldUntouched,
	}

	var steps []store.StepFile
	for i, disposition := range borne {
		step := stepFile()
		step.Step = i + 1
		step.ID = string(disposition)
		step.Disposition = disposition
		steps = append(steps, step)
	}
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   steps,
		outcome: &store.OutcomeFile{Outcome: store.OutcomeFailed, EndedAt: theRunStart.Add(time.Minute)},
	})

	dispositions := dispositionsOf(t, held, theEntryRunID)
	for _, want := range borne {
		if got, held := dispositions.Of(string(want)); !held || got != want {
			t.Errorf("the Step %q is %q (%t), want %q off its own file", want, got, held, want)
		}
	}
	if got, held := dispositions.Of("never authored"); !held || got != store.DispositionNeverReached {
		t.Errorf("a Step with no file is %q (%t), want %q", got, held, store.DispositionNeverReached)
	}

	// The seventh carrier: a reaped entry's closing write, whose one
	// Disposition is written out rather than assumed from its existence.
	_, reaped := seededJournal(t, anEntry{
		run:     runFileAt(t, theSecondRunID, theRunStart),
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(time.Hour), Step: 4, StepCode: store.StepCode{ID: "publish"}}},
	})
	if got, held := dispositionsOf(t, reaped, theSecondRunID).Of("publish"); !held || got != store.DispositionAttemptedOutcomeUnknown {
		t.Errorf("the Step the reaper named is %q (%t), want %q off the closing write", got, held, store.DispositionAttemptedOutcomeUnknown)
	}
}

// TestDispositions_TakeTheOwnersFileOverAClosersInferenceAndKeepBoth. On a
// contested entry the owner's observation is what the Step reads as, and the
// inference stands beside it rather than being removed (§7).
func TestDispositions_TakeTheOwnersFileOverAClosersInferenceAndKeepBoth(t *testing.T) {
	owned := stepFile()
	owned.Step, owned.ID, owned.Disposition = 4, "publish", store.DispositionRan
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{owned},
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Hour)},
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(31 * time.Minute), Step: 4, StepCode: store.StepCode{ID: "publish"}}},
	})

	dispositions := dispositionsOf(t, held, theEntryRunID)
	if got, isHeld := dispositions.Of("publish"); !isHeld || got != store.DispositionRan {
		t.Errorf("the Step `publish` is %q (%t), want the owner's observation %q", got, isHeld, store.DispositionRan)
	}
	if len(dispositions.Entry.Closers) != 1 {
		t.Errorf("the entry reports %d closers, want the inference kept", len(dispositions.Entry.Closers))
	}
}

// TestScan_FindsAStepsPreviousRunAcrossADayAMonthAndAYearBoundary. The backward
// scan is total: it walks the date partitions, crosses every boundary, and
// stops at the first match (§7).
func TestScan_FindsAStepsPreviousRunAcrossADayAMonthAndAYearBoundary(t *testing.T) {
	for name, tc := range map[string]struct {
		earlier time.Time
		run     string
	}{
		"a day":   {earlier: theRunStart.AddDate(0, 0, -1), run: theDayBeforeRunID},
		"a month": {earlier: theRunStart.AddDate(0, -1, 0), run: theMonthBeforeRunID},
		"a year":  {earlier: theRunStart.AddDate(-1, 0, 0), run: theYearBeforeRunID},
	} {
		t.Run(name, func(t *testing.T) {
			earlier := stepFile()
			earlier.StartedAt, earlier.EndedAt = tc.earlier, tc.earlier

			// The newest entry holds a different Step, so a scan
			// that stopped at the newest entry rather than the
			// first match would fail here.
			other := stepFile()
			other.ID = "publish"

			_, held := seededJournal(t,
				anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{other}},
				anEntry{run: runFileAt(t, tc.run, tc.earlier), steps: []store.StepFile{earlier}},
			)

			found, ok := firstOf(t, held.Scan("retire"))
			if !ok {
				t.Fatalf("the scan found nothing, want the Run %s across %s", tc.run, name)
			}
			if found.Entry.Run.String() != tc.run {
				t.Errorf("the scan stopped at %s, want %s", found.Entry.Run, tc.run)
			}
			if !found.Step.StartedAt.Equal(tc.earlier) {
				t.Errorf("the Step began at %s, want the file seeded at %s", found.Step.StartedAt, tc.earlier)
			}
		})
	}
}

// TestScan_StopsAtTheMostRecentRunCarryingTheStep. Every entry here carries the
// Step, so the answer turns entirely on the order the walk goes in: the
// partitions newest first, and the entries inside one on the instant each
// run.json carries. What the two callers want is the *last* Run the Step did
// something in, and a walk that found any of them would satisfy neither.
func TestScan_StopsAtTheMostRecentRunCarryingTheStep(t *testing.T) {
	// Two Runs on the newest day, the later one seeded under the id that
	// sorts second — so a walk ordering entries within a partition by their
	// directory name rather than by the instant each run.json carries would
	// stop at the wrong one.
	sameDayEarlier := runFileAt(t, theCloserRunID, theRunStart.Add(-time.Hour))
	sameDayLater := runFileAt(t, theSecondCloserRunID, theRunStart)

	_, held := seededJournal(t,
		anEntry{run: sameDayEarlier, steps: []store.StepFile{stepFile()}},
		anEntry{run: sameDayLater, steps: []store.StepFile{stepFile()}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{stepFile()}},
		anEntry{run: runFileAt(t, theYearBeforeRunID, theRunStart.AddDate(-1, 0, 0)), steps: []store.StepFile{stepFile()}},
	)

	var reached []string
	for found, err := range held.Scan("retire") {
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		reached = append(reached, found.Entry.Run.String())
	}
	want := []string{theSecondCloserRunID, theCloserRunID, theDayBeforeRunID, theYearBeforeRunID}
	if !slices.Equal(reached, want) {
		t.Errorf("the scan walked %v, want %v — newest first, across every partition", reached, want)
	}
}

// TestScan_TerminatesCleanlyWhereNoEarlierEntryCarriesThatStep.
func TestScan_TerminatesCleanlyWhereNoEarlierEntryCarriesThatStep(t *testing.T) {
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{stepFile()}},
		anEntry{run: runFileAt(t, theYearBeforeRunID, theRunStart.AddDate(-1, 0, 0)), steps: []store.StepFile{stepFile()}},
	)

	for found, err := range held.Scan("never authored") {
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		t.Fatalf("the scan found %+v, want nothing for a Step no entry carries", found.Step)
	}
}

// TestScan_MatchesAStepByItsAuthoredID. An `id` that moved is a different Step,
// with no Run anywhere behind it (ADR-0055).
func TestScan_MatchesAStepByItsAuthoredID(t *testing.T) {
	moved := stepFile()
	moved.ID = "retire-preview"
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{moved}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{stepFile()}},
	)

	found, ok := firstOf(t, held.Scan("retire-preview"))
	if !ok || found.Entry.Run.String() != theEntryRunID {
		t.Errorf("the scan found %v (%t), want the entry carrying the authored id", found.Entry.Run, ok)
	}
	if _, ok := firstOf(t, held.Scan("retire-previews")); ok {
		t.Error("a Step whose id moved found an earlier match; an id that moved is a different Step")
	}
}

// TestScan_SurfacesAReapersInferenceAsEvidenceOfTheStepItNamed. Run-once
// refuses on evidence, and *attempted, outcome unknown* is evidence whichever
// file carries it — the crashed Step's only record is the closing write (§6).
func TestScan_SurfacesAReapersInferenceAsEvidenceOfTheStepItNamed(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{stepFile()},
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(time.Hour), Step: 4, StepCode: store.StepCode{ID: "publish"}}},
	})

	found, ok := firstOf(t, held.Scan("publish"))
	if !ok {
		t.Fatal("the scan found nothing for the Step the reaper named")
	}
	if found.Step.Disposition != store.DispositionAttemptedOutcomeUnknown {
		t.Errorf("the Step is %q, want %q", found.Step.Disposition, store.DispositionAttemptedOutcomeUnknown)
	}
	if !found.Step.StartedAt.IsZero() {
		t.Errorf("the reading claims the Step began at %s; a reaper does not know that", found.Step.StartedAt)
	}
}

// TestScan_TakesTheOwnersFileOverAClosersInferenceOnAContestedEntry. The rule
// Dispositions.Of reads forward, asserted in the direction the scan reads: the
// owner's observation is what became of the Step whichever way a reader walks
// the entry. A contested entry is a reap of a Run that was alive after all, and
// its Step went on to finish — so the evidence run-once and §8 read off it is
// the file that Run wrote, and the inference stands where §7 puts it, among the
// entry's own closers (§6, §7, ADR-0076).
func TestScan_TakesTheOwnersFileOverAClosersInferenceOnAContestedEntry(t *testing.T) {
	owned := stepFile()
	owned.Step, owned.ID, owned.Disposition = 4, "publish", store.DispositionRan
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{owned},
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Hour)},
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(31 * time.Minute), Step: 4, StepCode: store.StepCode{ID: "publish"}}},
	})

	var reached []store.Disposition
	for found, err := range held.Scan("publish") {
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		reached = append(reached, found.Step.Disposition)
	}
	if !slices.Equal(reached, []store.Disposition{store.DispositionRan}) {
		t.Errorf("the scan reached %v, want the owner's observation alone — one record per Step, and the inference among the entry's closers", reached)
	}

	entry := oneEntry(t, held, theEntryRunID)
	if len(entry.Closers) != 1 {
		t.Errorf("the entry reports %d closers, want the inference standing where nothing removes it", len(entry.Closers))
	}
}

// TestScan_FiltersNoEntryOnAnyConsumersBehalf. A rehearsal is reached like any
// other entry, and which readings exclude one is each of the four consumers'
// (§7, ADR-0001).
func TestScan_FiltersNoEntryOnAnyConsumersBehalf(t *testing.T) {
	rehearsal := runFileAt(t, theEntryRunID, theRunStart)
	rehearsal.DryRun = true
	_, held := seededJournal(t,
		anEntry{run: rehearsal, steps: []store.StepFile{stepFile()}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{stepFile()}},
	)

	found, ok := firstOf(t, held.Scan("retire"))
	if !ok || found.Entry.Run.String() != theEntryRunID {
		t.Fatalf("the scan reached %v (%t), want the rehearsal reported rather than skipped", found.Entry.Run, ok)
	}
	if !found.Entry.DryRun {
		t.Error("the entry reports dry_run = false; the marker is what a consumer filters on")
	}
}

// TestScan_ReadsTheIdentitySetBackFromAnEntryHoldingOnlyADigest. The comparand
// is the last Run in which that Step carried a set **at all**, and the walk
// that reads one back terminates at the Run where the Step first carried one
// (§7, ADR-0055).
func TestScan_ReadsTheIdentitySetBackFromAnEntryHoldingOnlyADigest(t *testing.T) {
	// Three Runs, oldest first: the one that wrote the members, one that
	// carried the digest alone, and one where the Step refused and carried
	// no set at all — so a walk comparing against the previous Run rather
	// than against the last one carrying a set would find nothing.
	first := stepFile()
	first.Identities = store.Concluded(theExpansion, "")

	unchanged := stepFile()
	unchanged.Identities = store.Concluded(theExpansion, first.Identities.Digest)

	refused := stepFile()
	refused.Disposition = store.DispositionRefused
	refused.Identities = store.Identities{}

	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{refused}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{unchanged}},
		anEntry{run: runFileAt(t, theMonthBeforeRunID, theRunStart.AddDate(0, -1, 0)), steps: []store.StepFile{first}},
	)

	// The comparand: the last Run in which the Step carried a set at all,
	// which is the digest-only entry and not the Refusal above it.
	comparand, ok := firstOf(t, carryingASet(held.Scan("retire")))
	if !ok {
		t.Fatal("the scan found no Run in which the Step carried a set")
	}
	if comparand.Entry.Run.String() != theDayBeforeRunID {
		t.Errorf("the comparand is %s, want the last Run carrying a set at all", comparand.Entry.Run)
	}
	if comparand.Step.Identities.Members != nil {
		t.Error("the comparand carries members; the digest did not move and their absence is what says so")
	}

	set, from, err := store.ReadIdentitySet("retire", held.Scan("retire"))
	if err != nil {
		t.Fatalf("ReadIdentitySet: %v", err)
	}
	if !slices.Equal(set, theExpansion) {
		t.Errorf("the set reads back as %v, want %v in full from the entry that holds it", set, theExpansion)
	}
	// **Which** Run supplied them, which is the fact `show` renders as
	// *unchanged since* that Run: neither the entry in hand nor the
	// comparand above, but the one the walk stopped at (§9, issue #163).
	if from.String() != theMonthBeforeRunID {
		t.Errorf("the members came back from %s, want the Run that wrote them, %s", from, theMonthBeforeRunID)
	}
}

// TestJournal_SurfacesTheSchemaCeilingRatherThanGuessing. A Journal file above
// the reader's ceiling is neither read nor skipped, on every shape an entry
// holds (§7, ADR-0028).
func TestJournal_SurfacesTheSchemaCeilingRatherThanGuessing(t *testing.T) {
	at := store.JournalEntry{Run: runID(t, theEntryRunID), Started: theRunStart}
	step := stepFile()

	for name, tc := range map[string]struct {
		path  string
		known int
	}{
		"run.json":     {path: at.RunPath(), known: store.RunSchemaVersion},
		"a Step file":  {path: at.StepPath(step.Step), known: store.StepSchemaVersion},
		"outcome.json": {path: at.OutcomePath(), known: store.OutcomeSchemaVersion},
		"a closing write": {
			path:  at.ClosedByPath(runID(t, theCloserRunID)),
			known: store.ClosedBySchemaVersion,
		},
	} {
		t.Run(name, func(t *testing.T) {
			seeded := anEntry{
				run:     runFileAt(t, theEntryRunID, theRunStart),
				steps:   []store.StepFile{step},
				outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Minute)},
				closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(time.Hour), Step: 4}},
			}
			files := seeded.files(t)
			files[tc.path] = strings.Replace(files[tc.path], `"schema_version": 1`, `"schema_version": 2`, 1)

			r := newRepo(t)
			if _, err := store.Init(r.root, theInstant); err != nil {
				t.Fatalf("Init: %v", err)
			}
			r.seedFiles(r.root, files)
			held, err := store.Open(r.root, theInstant)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}

			err = readWhole(held)
			var unsupported store.SchemaUnsupported
			if !errors.As(err, &unsupported) {
				t.Fatalf("the read answered %v, want the store-schema-unsupported condition", err)
			}
			if unsupported.Written != 2 || unsupported.Known != tc.known {
				t.Errorf("the condition is %+v, want the file's 2 against this reader's %d", unsupported, tc.known)
			}
			if !strings.Contains(err.Error(), tc.path) {
				t.Errorf("the error is %q, want it to name the file it could not read", err)
			}
		})
	}
}

// TestJournal_RefusesAFileThatDoesNotSitWhereItsOwnContentsPutIt. The Journal's
// half of the rule a Record version is read under: an entry sits under the UTC
// date its own run.json carries and a Step file at the position its own `step`
// names, so a file that does not is a file `hyper` did not write. It is a fault
// rather than a skip — an entry filed under a date nothing put it under is one
// a listing would report and a backward scan would miss, which is two answers
// about one Journal that disagree (§7, §12).
func TestJournal_RefusesAFileThatDoesNotSitWhereItsOwnContentsPutIt(t *testing.T) {
	run := runID(t, theEntryRunID)
	at := store.JournalEntry{Run: run, Started: theRunStart}
	elsewhere := store.JournalEntry{Run: run, Started: theRunStart.AddDate(0, 0, -1)}
	step := stepFile()

	for name, files := range map[string]map[string]string{
		"an entry under a date nothing filed it under": {
			elsewhere.RunPath(): string(runFileAt(t, theEntryRunID, theRunStart).Encode()),
		},
		"a Step file at a position that is not its own": {
			at.RunPath():               string(runFileAt(t, theEntryRunID, theRunStart).Encode()),
			at.StepPath(step.Step + 1): string(step.Encode()),
		},
		"a directory holding no run.json at all": {
			at.StepPath(step.Step): string(step.Encode()),
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)
			if _, err := store.Init(r.root, theInstant); err != nil {
				t.Fatalf("Init: %v", err)
			}
			r.seedFiles(r.root, files)
			held, err := store.Open(r.root, theInstant)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}

			if err := readWhole(held); err == nil {
				t.Error("the read answered without a fault, want the file surfaced rather than read or skipped")
			}
		})
	}
}

// TestJournal_MatchesNoStepUnderTheEmptyID. A Step file always carries its
// authored id and a closing write carries one only where the dead Run's
// revision resolved it, so the empty string is the absence of an id rather than
// an id to match on (§7).
func TestJournal_MatchesNoStepUnderTheEmptyID(t *testing.T) {
	_, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		outcome: &store.OutcomeFile{Outcome: store.OutcomeFailed, EndedAt: theRunStart.Add(time.Minute)},
		// The reaper resolved nothing, which is every Run that
		// recorded repo_dirty: the closing write carries a Step's
		// position and no id at all.
		closers: map[string]store.ClosedBy{theCloserRunID: {EndedAt: theRunStart.Add(time.Hour), Step: 4}},
	})

	if got, known := dispositionsOf(t, held, theEntryRunID).Of(""); known {
		t.Errorf("the empty id is %q, want no answer at all", got)
	}
	if _, ok := firstOf(t, held.Scan("")); ok {
		t.Error("the scan matched a Step under the empty id")
	}
}

// TestJournal_AnswersNothingOverTheBranchsCommits. Append-only makes a year-old
// Run a read of the tip like any other, and Provenance names revisions on the
// code branch and none on this one — so the same files answer the same way
// whether they arrived in one commit or in four (§7).
func TestJournal_AnswersNothingOverTheBranchsCommits(t *testing.T) {
	entries := []anEntry{
		{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{stepFile()}},
		{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1))},
		{run: runFileAt(t, theMonthBeforeRunID, theRunStart.AddDate(0, -1, 0))},
		{run: runFileAt(t, theYearBeforeRunID, theRunStart.AddDate(-1, 0, 0)), steps: []store.StepFile{stepFile()}},
	}

	_, once := seededJournal(t, entries...)

	// The same four entries, one commit each, and the oldest Run committed
	// last — so a reader defined over the history rather than over the tree
	// would order them by when they landed.
	r := newRepo(t)
	if _, err := store.Init(r.root, theInstant); err != nil {
		t.Fatalf("Init: %v", err)
	}
	for _, entry := range slices.Backward(entries) {
		r.seedFiles(r.root, entry.files(t))
	}
	commitByCommit, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	first, err := once.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	second, err := commitByCommit.Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("one commit answers %v and four answer %v; no answer about the Journal is defined over the branch's commits",
			runsOfEntries(first), runsOfEntries(second))
	}
}

// TestJournal_ReadingWritesNothingToDisk is §7's own sentence as the observable
// fact it is: no local index, no derived state, and nothing under `.git/hyper/`
// — this milestone builds none, so the scan is a scan.
func TestJournal_ReadingWritesNothingToDisk(t *testing.T) {
	r, held := seededJournal(t, anEntry{
		run:     runFileAt(t, theEntryRunID, theRunStart),
		steps:   []store.StepFile{stepFile()},
		outcome: &store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: theRunStart.Add(time.Minute)},
	})
	before := r.workingTree()

	if err := readWhole(held); err != nil {
		t.Fatalf("reading the Journal: %v", err)
	}

	if after := r.workingTree(); !slices.Equal(before, after) {
		t.Errorf("the repository root holds %v, want it left as %v", after, before)
	}
	if _, err := os.Stat(filepath.Join(r.root, ".git", "hyper")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".git/hyper exists (%v); no answer here depends on derived state and none is built (§7)", err)
	}
}

// dispositionsOf reads one entry whole: the entry named by its Run, and the
// Step records under it.
func dispositionsOf(t *testing.T, held *store.Store, run string) store.Dispositions {
	t.Helper()

	dispositions, err := held.Dispositions(oneEntry(t, held, run))
	if err != nil {
		t.Fatalf("Dispositions: %v", err)
	}
	return dispositions
}

// firstOf is the backward scan stopping at the first match, which is what both
// of its callers do and what makes a set read off a recent entry cost one file.
func firstOf(t *testing.T, scan iter.Seq2[store.Evidence, error]) (store.Evidence, bool) {
	t.Helper()

	for found, err := range scan {
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		return found, true
	}
	return store.Evidence{}, false
}

// carryingASet is the scan narrowed to the Runs in which the Step carried an
// identity set at all, which is the comparand's own rule and not the previous
// Run's (ADR-0055).
func carryingASet(scan iter.Seq2[store.Evidence, error]) iter.Seq2[store.Evidence, error] {
	return func(yield func(store.Evidence, error) bool) {
		for found, err := range scan {
			if err == nil && found.Step.Identities.Digest == "" {
				continue
			}
			if !yield(found, err) {
				return
			}
		}
	}
}

// readWhole reads everything the Journal reader answers, which is what a case
// about a condition every read must surface drives it with.
func readWhole(held *store.Store) error {
	entries, err := held.Entries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if _, err := held.Dispositions(entry); err != nil {
			return err
		}
	}
	for _, err := range held.Scan("retire") {
		if err != nil {
			return err
		}
	}
	return nil
}

// TestListing_AnswersTheTargetsEachEntryBound is the second door beside Entries
// (issue #165). A Target is a fact only a Step file carries, and `runs` is the
// one surface §9 gives it to — so what this answers is the entries with the
// Targets beside them, and never the Step records they were read off.
//
// Each Target appears once however many Steps bound it, and the set is ordered
// by Unicode code point: it is a set read down a cell rather than a sequence of
// events, so nothing here is in the Run's written order.
func TestListing_AnswersTheTargetsEachEntryBound(t *testing.T) {
	first, second, third := stepFile(), stepFile(), stepFile()
	first.Step, first.Target = 1, "local"
	second.Step, second.Target = 2, "cloudflare-prod"
	third.Step, third.Target = 3, "local"

	elsewhere := stepFile()
	elsewhere.Step, elsewhere.Target = 1, "staging"

	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{first, second, third}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{elsewhere}},
	)

	listed := listing(t, held, nil)
	if got := boundBy(listed, theEntryRunID); !slices.Equal(got, []string{"cloudflare-prod", "local"}) {
		t.Errorf("the entry bound %v, want each Target once in code-point order", got)
	}
	if got := boundBy(listed, theDayBeforeRunID); !slices.Equal(got, []string{"staging"}) {
		t.Errorf("the day before bound %v, want [staging]", got)
	}
}

// TestListing_IsOrderedNewestFirstLikeEveryListingOfTheJournal. The two doors
// answer one order: what a listing of Runs renders and what a walk of the
// Journal reaches are the same sequence (§7, ADR-0065).
func TestListing_IsOrderedNewestFirstLikeEveryListingOfTheJournal(t *testing.T) {
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1))},
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart)},
		anEntry{run: runFileAt(t, theMonthBeforeRunID, theRunStart.AddDate(0, -1, 0))},
	)

	listed := listing(t, held, nil)
	want := []string{theEntryRunID, theDayBeforeRunID, theMonthBeforeRunID}
	got := make([]string, len(listed))
	for i, entry := range listed {
		got[i] = entry.Run.String()
	}
	if !slices.Equal(got, want) {
		t.Errorf("the listing is %v, want %v newest first", got, want)
	}
}

// TestListing_ReadsTheStepFilesOfTheEntriesTheCallerWanted. The predicate is
// applied after the accounts are read and before the Step files are, which is
// what makes narrowing the time axis cheap: an entry nobody wanted contributes
// no row and costs none of its Step files.
func TestListing_ReadsTheStepFilesOfTheEntriesTheCallerWanted(t *testing.T) {
	step := stepFile()
	step.Step, step.Target = 1, "staging"
	_, held := seededJournal(t,
		anEntry{run: runFileAt(t, theEntryRunID, theRunStart), steps: []store.StepFile{step}},
		anEntry{run: runFileAt(t, theDayBeforeRunID, theRunStart.AddDate(0, 0, -1)), steps: []store.StepFile{step}},
	)

	listed := listing(t, held, func(entry store.Entry) bool { return entry.Run == runID(t, theEntryRunID) })
	if len(listed) != 1 || listed[0].Run != runID(t, theEntryRunID) {
		t.Fatalf("the listing is %v, want the one entry the predicate kept", listed)
	}
	if !slices.Equal(listed[0].Targets, []string{"staging"}) {
		t.Errorf("the kept entry bound %v, want [staging]", listed[0].Targets)
	}
}

// TestListing_OfAnEntryHoldingNoStepRecordIsNothing. A Run that Refused before
// Step 1, or went quiet before it wrote a file, bound nothing — and the answer
// is the absence rather than a name nobody recorded (§7).
func TestListing_OfAnEntryHoldingNoStepRecordIsNothing(t *testing.T) {
	_, held := seededJournal(t, anEntry{run: runFileAt(t, theEntryRunID, theRunStart)})

	if got := boundBy(listing(t, held, nil), theEntryRunID); len(got) != 0 {
		t.Errorf("the entry bound %v, want nothing: it holds no Step record", got)
	}
}

// TestListing_ReadsAReapersInferenceAsTheStepItNamed. A reaped entry's account
// of the Step the dead Run went quiet on is a closing write, and it is a record
// of that Step in the shape a Step file records one — so the Target it resolved
// is a Target that Run bound (§7, ADR-0076).
func TestListing_ReadsAReapersInferenceAsTheStepItNamed(t *testing.T) {
	reached := stepFile()
	reached.Step, reached.Target = 1, "local"

	_, held := seededJournal(t, anEntry{
		run:   runFileAt(t, theEntryRunID, theRunStart),
		steps: []store.StepFile{reached},
		closers: map[string]store.ClosedBy{theCloserRunID: {
			EndedAt:  theRunStart.Add(time.Hour),
			Step:     2,
			StepCode: store.StepCode{ID: "publish", Target: "cloudflare-prod"},
		}},
	})

	if got := boundBy(listing(t, held, nil), theEntryRunID); !slices.Equal(got, []string{"cloudflare-prod", "local"}) {
		t.Errorf("the entry bound %v, want the reaper's Step counted with the owner's", got)
	}
}

// listing reads the Journal back with the Targets beside each entry, failing
// the case where it could not be read at all.
func listing(t *testing.T, held *store.Store, wanted func(store.Entry) bool) []store.Listed {
	t.Helper()

	listed, err := held.Listing(wanted)
	if err != nil {
		t.Fatalf("Listing: %v", err)
	}
	return listed
}

// boundBy is what one Run of a listing bound, found by its id so that an
// assertion names the entry it is about rather than a position.
func boundBy(listed []store.Listed, run string) []string {
	for _, entry := range listed {
		if entry.Run.String() == run {
			return entry.Targets
		}
	}
	return nil
}
