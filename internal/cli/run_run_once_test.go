package cli_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **A run-once Step runs once, and the evidence it refuses on the second time
// is the one the first Run wrote** (§6, §7, §12, issue #153).
//
// No golden can hold this. What it is about is what the *second* Run does given
// what the first left, and a case drives one Run — so the corpus cases that
// seed a Journal say what one Disposition means, and this says that a real Run
// writes one a real Run reads.
//
// The two halves matter separately. A seeded entry proves the walk reads what a
// Journal holds; this proves the two ends meet — that the Disposition the
// engine writes for a Step that ran is the Disposition the engine refuses on,
// under the same authored `id`, with nothing edited in between.
//
// It drives [testdata/run/two-runs-of-one-run-once-step] twice through one
// materialised repository. The case serves **two different answers**, so a
// second call that went out would mint a second version: that the branch holds
// one version after both Runs is what says no call went out at all.

// TestRunOnce_TheSecondRunRefusesOnWhatTheFirstWrote drives the two Runs and
// reads what each of them left.
func TestRunOnce_TheSecondRunRefusesOnWhatTheFirstWrote(t *testing.T) {
	c := corpusCase(t, "run/two-runs-of-one-run-once-step")
	invocation := c.invocation(t)
	// One process for the two Runs, so the mint answers the two ids the
	// case names in the order it names them — a fresh one per Run would
	// answer the first id twice and write one entry over another.
	process := c.process(t, invocation)

	for nth, expected := range []struct {
		exit        int
		disposition store.Disposition
	}{
		{0, store.DispositionRan},
		{77, store.DispositionRefused},
	} {
		var stdout, stderr bytes.Buffer
		if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != expected.exit {
			t.Fatalf("run %d: exit = %d, want %d; stderr: %s", nth+1, exit, expected.exit, stderr.String())
		}

		file := stepFileOf(t, c, invocation, nth+1)
		if file.Disposition != expected.disposition {
			t.Errorf("run %d is %s, want %s", nth+1, file.Disposition, expected.disposition)
		}

		// **A Step no call went out from minted nothing.** The case
		// serves a second answer carrying a different `id`, so a call
		// that went out on the second Run would have moved the bytes
		// and put a second version of this series on the branch (§7,
		// ADR-0030).
		branch := invocation.fixture.render(t, invocation.fixture.root)
		if held := strings.Count(branch, "=== records/"); held != 1 {
			t.Errorf("run %d leaves %d Record versions on the branch, want 1", nth+1, held)
		}
	}

	// **The Refusal is the Run's**, held on `outcome.json` under `refusal`
	// and on no Step file — and it carries exactly one member, a Refusal
	// being terminal, so there is no second check to reach (§7, ADR-0061).
	refusal := outcomeOf(t, c, invocation, 2).Refusal
	if len(refusal) != 1 {
		t.Fatalf("the Refusal holds %d members, want exactly 1: %+v", len(refusal), refusal)
	}
	if refusal[0].ErrorCode != run.CodeRunOnceRecorded {
		t.Errorf("the Refusal is %q, want %q", refusal[0].ErrorCode, run.CodeRunOnceRecorded)
	}
	// It names the Run whose entry the evidence was read off, which is the
	// first of the two: a reader handed this row can open that entry and
	// see the Step that ran.
	if first := strings.Fields(readFile(t, filepath.Join(c.dir, "mint")))[0]; !strings.Contains(refusal[0].Message, first) {
		t.Errorf("the Refusal reads %q, and names no run; want run %s in it", refusal[0].Message, first)
	}
	if file := stepFileOf(t, c, invocation, 2); file.Identities.Digest != "" || len(file.Identities.Members) != 0 {
		t.Errorf("the refused Step carries an identity set (%+v); a Step that concluded about nothing carries none", file.Identities)
	}
}

// outcomeOf reads one Run's outcome.json back off the branch, the way
// stepFileOf reads its Step file: by the id the case's mint named it, at the
// instant the case's clock answers.
func outcomeOf(t *testing.T, c goldenCase, invocation goldenRun, nth int) store.OutcomeFile {
	t.Helper()

	ids := strings.Fields(readFile(t, filepath.Join(c.dir, "mint")))
	if nth > len(ids) {
		t.Fatalf("the case names %d Run ids and this is Run %d", len(ids), nth)
	}
	id, err := store.ParseRunID(ids[nth-1])
	if err != nil {
		t.Fatal(err)
	}

	entry := store.JournalEntry{Run: id, Started: c.instant(t)}
	branch := invocation.fixture.render(t, invocation.fixture.root)
	content, held := blobOf(branch, entry.OutcomePath())
	if !held {
		t.Fatalf("run %d wrote no %s; the branch holds:\n%s", nth, entry.OutcomePath(), branch)
	}
	file, err := store.DecodeOutcomeFile([]byte(content))
	if err != nil {
		t.Fatalf("run %d's outcome.json: %v", nth, err)
	}
	return file
}

// **A rehearsal counts as no evidence at all, driven as the round trip rather
// than as a seeded marker** (§7, §8, ADR-0001, issue #153).
//
// The corpus case beside it hand-writes `"dry_run": true` on an entry, which
// says what the walk does with the marker. This says what a rehearsal actually
// leaves: `--dry-run` writes its own entry, the real Run after it reads that
// entry like any other, and the run-once Step **runs**.
//
// It is the claim the exception to the absence rule is bought against. `dry_run`
// is written on every entry, `false` included, because a reader that took its
// absence for `false` would refuse every run-once Step in the Procedure it
// rehearsed — permanently, with no bypass to recover through and nothing but an
// edit to a reviewed artefact left. The review aid would disarm the tool.
//
// It drives [testdata/run/a-rehearsal-then-the-real-run] twice: the case's own
// argv, which carries `--dry-run`, and then the same argv with the flag taken
// off. Nothing else moves between them.
func TestRunOnce_ARehearsalIsNoEvidenceForTheRunAfterIt(t *testing.T) {
	c := corpusCase(t, "run/a-rehearsal-then-the-real-run")
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	for nth, rehearsal := range []bool{true, false} {
		args := invocation.args
		if !rehearsal {
			args = withoutDryRun(args)
		}

		var stdout, stderr bytes.Buffer
		if exit := cli.Main(args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
			t.Fatalf("run %d (rehearsal %t): exit = %d, want 0; stderr: %s", nth+1, rehearsal, exit, stderr.String())
		}

		// **Both Runs *ran*.** The second is the one the claim is about:
		// the Journal holds the first recording this very Step under this
		// very `id`, and it is a rehearsal, so it is evidence of nothing.
		file := stepFileOf(t, c, invocation, nth+1)
		if file.Disposition != store.DispositionRan {
			t.Errorf("run %d (rehearsal %t) is %s, want %s", nth+1, rehearsal, file.Disposition, store.DispositionRan)
		}
	}

	// The entry the rehearsal left says it was one, which is what the walk
	// reads — and the real Run's says it was not.
	for nth, marked := range []bool{true, false} {
		if held := runFileOf(t, c, invocation, nth+1).DryRun; held != marked {
			t.Errorf("run %d writes dry_run: %t, want %t", nth+1, held, marked)
		}
	}
}

// withoutDryRun is the same invocation with the rehearsal flag taken off, which
// is the one thing that moves between this driver's two Runs.
func withoutDryRun(args []string) []string {
	real := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "--dry-run" {
			real = append(real, arg)
		}
	}
	if len(real) == len(args) {
		panic("the case's argv carries no --dry-run: the first Run of this driver is the rehearsal")
	}
	return real
}

// runFileOf reads one Run's run.json back off the branch, the way outcomeOf
// reads its outcome.json.
func runFileOf(t *testing.T, c goldenCase, invocation goldenRun, nth int) store.RunFile {
	t.Helper()

	ids := strings.Fields(readFile(t, filepath.Join(c.dir, "mint")))
	if nth > len(ids) {
		t.Fatalf("the case names %d Run ids and this is Run %d", len(ids), nth)
	}
	id, err := store.ParseRunID(ids[nth-1])
	if err != nil {
		t.Fatal(err)
	}

	entry := store.JournalEntry{Run: id, Started: c.instant(t)}
	branch := invocation.fixture.render(t, invocation.fixture.root)
	content, held := blobOf(branch, entry.RunPath())
	if !held {
		t.Fatalf("run %d wrote no %s; the branch holds:\n%s", nth, entry.RunPath(), branch)
	}
	file, err := store.DecodeRunFile([]byte(content))
	if err != nil {
		t.Fatalf("run %d's run.json: %v", nth, err)
	}
	return file
}
