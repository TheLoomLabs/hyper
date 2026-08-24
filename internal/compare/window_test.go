package compare_test

import (
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The window (§8, issue #167): which two Runs a Comparison names, and what
// instant each side of it is.

// at is an instant on the day §7's own worked examples are written at, so a
// test reads as a sequence of Runs rather than as a sequence of timestamps.
func at(hour, minute int) time.Time {
	return time.Date(2026, 8, 6, hour, minute, 0, 0, time.UTC)
}

// run is one Journal entry as a case builds it: a Run of a Procedure, started
// at an instant, closed by its own Run at another.
func run(id, procedure string, started, ended time.Time) store.Entry {
	return store.Entry{
		RunFile: store.RunFile{
			Run:       runID(id),
			Procedure: procedure,
			StartedAt: started,
			Trigger:   store.Trigger{Cause: store.CauseManual, Executor: store.ExecutorLocal, Actor: "igor", Host: "thinkpad"},
		},
		Owner: store.OutcomeFile{Outcome: store.OutcomeCompleted, EndedAt: ended},
	}
}

// runID parses an id a case spells as its two leading digits, the rest of the
// UUIDv7 being of no interest to a window.
func runID(digits string) store.RunID {
	id, err := store.ParseRunID("019920" + digits + "-0000-7000-8000-000000000000")
	if err != nil {
		panic(err)
	}
	return id
}

func TestInstant_IsTheOwnersEndedAt(t *testing.T) {
	side := compare.Side{Present: true, Entry: run("01", "watch", at(9, 0), at(9, 5))}
	if got, want := side.Instant(), at(9, 5); !got.Equal(want) {
		t.Errorf("Instant() = %s, want the owner's ended_at %s", got, want)
	}
}

func TestInstant_OnAReapedEntryIsTheLastStepFilesAndNeverTheClosers(t *testing.T) {
	// The closing write's instant is on the *closing* Run's clock, so a Run
	// reaped a week later would otherwise sweep every intervening Run's
	// versions into its side of the window (§8).
	entry := run("02", "watch", at(9, 0), time.Time{})
	entry.Owner = store.OutcomeFile{}
	entry.Closers = []store.Closer{{Run: runID("09"), ClosedBy: store.ClosedBy{EndedAt: at(23, 0), Step: 3}}}

	side := compare.Side{
		Present: true,
		Entry:   entry,
		Steps: []store.StepFile{
			{Step: 1, EndedAt: at(9, 1)},
			{Step: 2, EndedAt: at(9, 4)},
		},
	}
	if got, want := side.Instant(), at(9, 4); !got.Equal(want) {
		t.Errorf("Instant() = %s, want the last Step file's ended_at %s", got, want)
	}
}

func TestInstant_OnAContestedEntryIsTheOwners(t *testing.T) {
	entry := run("03", "watch", at(9, 0), at(9, 5))
	entry.Closers = []store.Closer{{Run: runID("09"), ClosedBy: store.ClosedBy{EndedAt: at(23, 0), Step: 3}}}

	side := compare.Side{Present: true, Entry: entry, Steps: []store.StepFile{{Step: 1, EndedAt: at(9, 1)}}}
	if got, want := side.Instant(), at(9, 5); !got.Equal(want) {
		t.Errorf("Instant() = %s, want the owner's ended_at %s", got, want)
	}
}

func TestStandingOf_ARehearsalIsDisqualifiedAndAnOpenEntryIsNotYetNameable(t *testing.T) {
	rehearsal := run("04", "watch", at(9, 0), at(9, 5))
	rehearsal.DryRun = true

	open := run("05", "watch", at(9, 0), time.Time{})
	open.Owner = store.OutcomeFile{}

	refused := run("06", "watch", at(9, 0), at(9, 5))
	refused.Owner.Outcome = store.OutcomeRefused

	for _, c := range []struct {
		name  string
		entry store.Entry
		want  compare.Standing
	}{
		{"a rehearsal", rehearsal, compare.Rehearsal},
		{"an open entry", open, compare.Unclosed},
		{"a refused Run", refused, compare.Nameable},
		{"a completed Run", run("07", "watch", at(9, 0), at(9, 5)), compare.Nameable},
	} {
		if got := compare.StandingOf(c.entry); got != c.want {
			t.Errorf("StandingOf(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestSelect_TheBaselineIsThePreviousRunOfTheSameProcedure(t *testing.T) {
	// A monitoring Run is never compared against a provisioning one, so the
	// Run of another Procedure standing between the two is passed over.
	windows := compare.Select([]store.Entry{
		run("13", "watch", at(11, 0), at(11, 2)),
		run("12", "retire", at(10, 0), at(10, 2)),
		run("11", "watch", at(9, 0), at(9, 2)),
	}, compare.Selection{Procedure: "watch"})

	if len(windows) != 1 {
		t.Fatalf("Select() answered %d windows, want 1", len(windows))
	}
	if got, want := windows[0].Subject.Entry.Run.String(), runID("13").String(); got != want {
		t.Errorf("subject = %s, want %s", got, want)
	}
	if !windows[0].Baseline.Present {
		t.Fatal("the window has no baseline; the Procedure has an earlier Run")
	}
	if got, want := windows[0].Baseline.Entry.Run.String(), runID("11").String(); got != want {
		t.Errorf("baseline = %s, want %s", got, want)
	}
}

func TestSelect_ARehearsalAndAnOpenEntryAreNeitherSide(t *testing.T) {
	rehearsal := run("12", "watch", at(10, 0), at(10, 2))
	rehearsal.DryRun = true
	open := run("14", "watch", at(12, 0), time.Time{})
	open.Owner = store.OutcomeFile{}

	windows := compare.Select([]store.Entry{
		open,
		run("13", "watch", at(11, 0), at(11, 2)),
		rehearsal,
		run("11", "watch", at(9, 0), at(9, 2)),
	}, compare.Selection{Procedure: "watch"})

	if len(windows) != 1 {
		t.Fatalf("Select() answered %d windows, want 1", len(windows))
	}
	if got, want := windows[0].Subject.Entry.Run.String(), runID("13").String(); got != want {
		t.Errorf("subject = %s, want %s — an open entry is not yet an entry a window can name", got, want)
	}
	if got, want := windows[0].Baseline.Entry.Run.String(), runID("11").String(); got != want {
		t.Errorf("baseline = %s, want %s — a rehearsal is disqualified", got, want)
	}
}

func TestSelect_AFirstRunHasNoBaseline(t *testing.T) {
	windows := compare.Select([]store.Entry{
		run("11", "watch", at(9, 0), at(9, 2)),
	}, compare.Selection{Procedure: "watch"})

	if len(windows) != 1 {
		t.Fatalf("Select() answered %d windows, want 1", len(windows))
	}
	if windows[0].Baseline.Present {
		t.Error("the first Run of a Procedure has a baseline; want none")
	}
}

func TestSelect_SinceTakesTheLastRunBeforeTheInstantAndFoldsTheRest(t *testing.T) {
	windows := compare.Select([]store.Entry{
		run("14", "watch", at(12, 0), at(12, 2)),
		run("13", "watch", at(11, 0), at(11, 2)),
		run("12", "watch", at(10, 0), at(10, 2)),
		run("11", "watch", at(9, 0), at(9, 2)),
	}, compare.Selection{Procedure: "watch", Since: at(10, 30), SinceNamed: true})

	if len(windows) != 1 {
		t.Fatalf("Select() answered %d windows, want 1", len(windows))
	}
	if got, want := windows[0].Baseline.Entry.Run.String(), runID("12").String(); got != want {
		t.Errorf("baseline = %s, want the last Run before the instant, %s", got, want)
	}
	if got, want := windows[0].Subject.Entry.Run.String(), runID("14").String(); got != want {
		t.Errorf("subject = %s, want the newest Run, %s — everything after the baseline folds into one rendering", got, want)
	}
}

func TestSelect_SinceAfterEveryRunNamesNoWindow(t *testing.T) {
	windows := compare.Select([]store.Entry{
		run("11", "watch", at(9, 0), at(9, 2)),
	}, compare.Selection{Procedure: "watch", Since: at(10, 0), SinceNamed: true})

	if len(windows) != 0 {
		t.Errorf("Select() answered %d windows, want none — no Run happened in the window named", len(windows))
	}
}

func TestSelect_NamingNoProcedureAnswersOneWindowEachInNameOrder(t *testing.T) {
	windows := compare.Select([]store.Entry{
		run("13", "watch", at(11, 0), at(11, 2)),
		run("12", "alpha", at(10, 0), at(10, 2)),
		run("11", "retire", at(9, 0), at(9, 2)),
	}, compare.Selection{})

	var named []string
	for _, window := range windows {
		named = append(named, window.Procedure)
	}
	want := []string{"alpha", "retire", "watch"}
	if len(named) != len(want) {
		t.Fatalf("Select() answered %v, want one window per Procedure %v", named, want)
	}
	for i := range want {
		if named[i] != want[i] {
			t.Fatalf("Select() answered %v, want Procedure-name code-point order %v", named, want)
		}
	}
}

func TestSelect_AProcedureThatRanOnlyRehearsalsNamesNoWindow(t *testing.T) {
	rehearsal := run("11", "watch", at(9, 0), at(9, 2))
	rehearsal.DryRun = true

	if windows := compare.Select([]store.Entry{rehearsal}, compare.Selection{Procedure: "watch"}); len(windows) != 0 {
		t.Errorf("Select() answered %d windows, want none — a rehearsal is neither side", len(windows))
	}
}
