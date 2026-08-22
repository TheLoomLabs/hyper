package cli_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **The two Runs it takes to observe a reap** (§6, §7, ADR-0003, ADR-0076,
// issue #154).
//
// The corpus says what one Run does with a Journal it was handed: the closing
// writes a reap makes, the Step each one names, and the code facts it omits.
// What no case can hold is the pair — *a Run went quiet, and the next effectful
// Run closed what it left* — because it is two Runs, and a case drives one.
//
// Both cases here drive real Runs at both ends. The first is the round trip: a
// Run whose process is gone leaves an entry holding no account at all, and the
// next effectful Run closes it. The second is the contest: a Run reaped while
// it was **alive** finishes on its own terms, and the entry ends up holding
// both accounts with neither removed.

// TestReap_TheNextEffectfulRunClosesWhatAKilledRunLeft is §6's sentence driven
// end to end: the two interrupts, and the Run that comes after them.
//
// **The first interrupt is delivered by the driver `run_signal_test.go` states
// in full** — handed over from inside Step 1's own call, with `deliver` not
// returning until the watch has taken itself down. There is no sleep anywhere
// here and no case can pass by racing.
//
// **The second interrupt kills the process outright**, and it lands on the
// kernel's own answer because the watch has just released. There is no code
// path to drive and nothing for `hyper` to write: what it leaves is whatever
// the branch held at that instant, which is the tip read one line above the
// delivery and before the drained Step could commit anything. Putting the
// branch back at that commit is what the kill did — the ref is moved to a
// commit this Run itself wrote, and nothing is edited and nothing invented.
func TestReap_TheNextEffectfulRunClosesWhatAKilledRunLeft(t *testing.T) {
	c := corpusCase(t, "run/two-runs-and-a-kill-between-them", "run", "publish-preview")
	invocation := c.invocation(t)
	// One process for the two Runs, so the mint answers the two ids the
	// case names in the order it names them (run_run_once_test.go).
	process := c.process(t, invocation)
	killed, closer := twoRunIDs(t, c)

	signals := watchedBy()
	process.Notify = signals.notify

	// The tip and the signal, in that order and from inside the Step's own
	// call: the branch as it stands at the instant the terminal's Ctrl-C
	// lands. Both run on the Run's own goroutine, so a fault is carried out
	// rather than fataled where a t.Fatal would leave the Run running
	// (run_signal_test.go, branchPaths).
	var once sync.Once
	var frozen string
	var faulted error
	dial := process.Dial
	process.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		once.Do(func() {
			frozen = invocation.fixture.text(t, invocation.fixture.root, "rev-parse", store.Ref)
			faulted = signals.deliver(os.Interrupt)
		})
		return dial(ctx, network, address)
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != cli.ExitInterrupted {
		t.Fatalf("the interrupted Run: exit = %d, want %d; stderr: %s", exit, cli.ExitInterrupted, stderr.String())
	}
	if faulted != nil {
		t.Fatal(faulted)
	}
	if frozen == "" {
		t.Fatal("the first Run reached no Step, so there is no instant for the second interrupt to have landed at")
	}

	// What the second interrupt left: this Run's `run.json` and no account
	// beside it, neither an `outcome.json` it wrote nor a `closed-by/` file
	// anybody wrote. That absence **is** the representation (§7).
	invocation.fixture.run(t, invocation.fixture.root, "update-ref", store.Ref, frozen)
	assertBranch(t, invocation.fixture.render(t, invocation.fixture.root),
		present(killed+"/run.json"), absent("outcome.json"), absent("closed-by"))

	// The next effectful Run, against exactly that branch, and nobody
	// interrupts this one.
	stdout.Reset()
	stderr.Reset()
	process.Dial, process.Notify = dial, nil
	if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != cli.ExitClean {
		t.Fatalf("the second Run: exit = %d; stderr: %s", exit, stderr.String())
	}

	// **One path, and it is the closer's own.** The reaper writes neither
	// `outcome.json` nor a file under `steps/` — those are the owner's
	// paths, and a closer that could take one is the same-path write §12's
	// grammar makes impossible (§7, ADR-0076).
	entry := entryOf(t, c, invocation, killed)
	if entry.Account() != store.AccountReaped {
		t.Errorf("the killed Run's entry is %v, want reaped", entry.Account())
	}
	if len(entry.Closers) != 1 || entry.Closers[0].Run.String() != closer {
		t.Errorf("the entry holds %+v, want one closing write by %s", entry.Closers, closer)
	}
	if outcome, has := entry.Outcome(); !has || outcome != store.OutcomeFailed {
		t.Errorf("the entry's outcome is %q, want failed — the Run really did not come back", outcome)
	}

	// **The Step the dead Run went quiet on is `attempted, outcome
	// unknown`**, and it is that value rather than *never reached* — without
	// which the crashed Step reads as one that never happened, which re-runs
	// an effect nobody vouched for (§6, ADR-0003).
	closing := entry.Closers[0]
	if closing.Step != 1 {
		t.Errorf("the closing write names Step %d, want 1 — the killed Run wrote no Step file", closing.Step)
	}
	if closing.ID != "publish" {
		t.Errorf("the closing write names Step %q, want publish — derived at the dead Run's own revision", closing.ID)
	}
	reading := closing.Reading()
	if reading.Disposition != store.DispositionAttemptedOutcomeUnknown {
		t.Errorf("the Step reads %q, want %q", reading.Disposition, store.DispositionAttemptedOutcomeUnknown)
	}
}

// TestReap_AReapedRunThatWasAliveLeavesBothAccounts is ADR-0076's whole case,
// and the one shape §6 says it is: an effectful Run on one clone overlapping an
// effectful Run on another.
//
// The lock is one filesystem's, so two clones of one repository are two locks
// and neither Run waits on the other — which is exactly the laptop-and-runner
// overlap the contest arises in. The second Run is driven from inside the
// first's own Step, so the first is genuinely in flight when it is reaped, and
// there is no sleep anywhere: the reap happens while the read that triggered it
// has not returned.
//
// **A wrong reap costs the reaped Run nothing.** It holds the only paths its
// own account can be written at, so it finishes on its own terms and pushes
// what it wrote; the entry ends up **contested**, with both accounts standing
// and neither removed. The entry's outcome is the owner's — an `outcome.json`
// is its own Run's observation and a closing write is another Run's inference
// drawn from a silence (§7, ADR-0076).
func TestReap_AReapedRunThatWasAliveLeavesBothAccounts(t *testing.T) {
	c := corpusCase(t, "run/a-reap-that-was-wrong-is-contested", "run", "publish-preview")
	invocation := c.invocation(t)
	process := c.process(t, invocation)
	alive, reaper := twoRunIDs(t, c)

	// The second clone: origin, cloned again. It holds the code branch at
	// the same commit and the Store as a remote-tracking ref, which is the
	// state a runner's fresh checkout is always in (§7, ADR-0074).
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	invocation.fixture.run(t, t.TempDir(), "clone", "--quiet", "--branch", codeBranchName, invocation.fixture.origin, elsewhere)

	// The overlapping Run, driven from inside the first Run's own call. It
	// dials through the harness's dialer rather than the wrapper below, so
	// it reaches the same served host and starts nothing recursive.
	overlapping := process
	var stdout, stderr bytes.Buffer
	var once sync.Once
	var overlapped int
	dial := process.Dial
	process.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		once.Do(func() {
			overlapped = cli.Main([]string{"run", "--repo-dir", elsewhere, "publish-preview"},
				&stdout, &stderr, overlapping, c.facts(t))
		})
		return dial(ctx, network, address)
	}

	var first, firstErr bytes.Buffer
	exit := cli.Main(invocation.args, &first, &firstErr, process, c.facts(t))

	if overlapped != cli.ExitClean {
		t.Fatalf("the overlapping Run: exit = %d; stderr: %s", overlapped, stderr.String())
	}
	// **The reaped Run finishes on its own terms.** Nothing the reaper did
	// reached a path this Run was going to write, so it completes and its
	// push lands — re-applied onto the tip the reaper moved the remote to,
	// which is the retry ADR-0076 made clean in every case (§7).
	if exit != cli.ExitClean {
		t.Fatalf("the reaped Run: exit = %d; stderr: %s", exit, firstErr.String())
	}

	// The contest, read off the remote both Runs pushed to: the entry holds
	// **both** files and neither is removed.
	rendered := invocation.fixture.render(t, invocation.fixture.origin)
	assertBranch(t, rendered,
		present(alive+"/outcome.json"),
		present(alive+"/closed-by/"+reaper+".json"))

	entry := entryOf(t, c, invocation, alive)
	if entry.Account() != store.AccountContested {
		t.Fatalf("the overlapped Run's entry is %v, want contested", entry.Account())
	}
	// **The entry's outcome is the owner's**, an observation being what an
	// inference was an inference about — and the inference stays on the
	// branch beside it, true of the Run that drew it (§7).
	if outcome, has := entry.Outcome(); !has || outcome != store.OutcomeCompleted {
		t.Errorf("the contested entry's outcome is %q, want the owner's completed", outcome)
	}
	if len(entry.Closers) != 1 || entry.Closers[0].Run.String() != reaper {
		t.Errorf("the entry holds %+v, want the one inference %s drew", entry.Closers, reaper)
	}
	// A contested entry derives a duration normally: the account is the
	// owner's, written on the owner's clock inside the owner's entry, and
	// the closing write beside it is not an endpoint of anything (§7, §8).
	if _, derives := entry.Duration(); !derives {
		t.Error("the contested entry derives no duration; there the account is the owner's")
	}
}

// twoRunIDs is the pair of Run ids a case names, which both cases here read
// back to say *this* Run's entry rather than *an* entry.
func twoRunIDs(t *testing.T, c goldenCase) (first, second string) {
	t.Helper()

	named := strings.Fields(readFile(t, filepath.Join(c.dir, "mint")))
	if len(named) != 2 {
		t.Fatalf("case %s names %d Run ids; these cases drive two Runs", c.name, len(named))
	}
	return named[0], named[1]
}

// entryOf reads one entry back off the branch the Run left, by the id of the
// Run whose entry it is.
func entryOf(t *testing.T, c goldenCase, invocation goldenRun, run string) store.Entry {
	t.Helper()

	held, err := store.Open(invocation.wd, c.instant(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id, err := store.ParseRunID(run)
	if err != nil {
		t.Fatalf("ParseRunID(%q): %v", run, err)
	}
	entry, found, err := held.Entry(id)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !found {
		t.Fatalf("the branch holds no entry for %s", run)
	}
	return entry
}
