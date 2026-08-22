package cli_test

import (
	"bytes"
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **The two signals, delivered while a Run is in flight** (§6, §12, ADR-0015,
// issue #145).
//
// No golden can hold any of this, and for the reason `run_push_test.go`'s
// cannot hold its own: a case directory is an argv and a set of inputs, and
// *a signal arrived while Step 1 was talking to the world* is a fact about
// **when**, which no file beside an argv can state. So the delivery is driven
// here instead — the same corpus cases, the same entry point, with the tenth
// process read supplied by the case rather than by the terminal.
//
// The delivery is deterministic rather than timed, and that is what makes these
// cases assertable at all. The signal is handed over from inside the Step's own
// call — the first connection a `read` Step makes, or the first child a `shell`
// Step starts — and `deliver` does not return until the watch has taken itself
// down, which is the moment after the drain became readable. So the Step in
// flight is genuinely in flight when the signal lands, and the Run's next
// boundary is guaranteed to see it: there is no sleep anywhere here, and no
// case can pass by racing.
//
// What is asserted at that point is §6's whole sentence. The Step in flight
// finishes — its Disposition is *ran* and its Record is on the branch. No
// further Step starts — the Steps after it are *never reached* and write no
// file. The Run closes its **own** entry `failed` — `outcome.json` is there, and
// it was written by this Run rather than by anybody's reap. And the exit code
// is the signal's: `130` for an interrupt, `143` for a termination.

// signalling is the process's signals as a case delivers them: one channel the
// case writes into, and the release the watch takes down.
//
// stopped is what makes `deliver` synchronous. internal/cli's watch closes the
// drain and then releases, in that order, so a case that has seen the release
// has seen a drain that is already readable — which is what lets the assertion
// below be *the Step after this one did not start* rather than *it usually does
// not*.
type signalling struct {
	arriving chan os.Signal
	stopped  chan struct{}
	once     sync.Once
}

func watchedBy() *signalling {
	return &signalling{arriving: make(chan os.Signal, 1), stopped: make(chan struct{})}
}

// notify is cli.Notify as this case stands it: the channel the case writes
// into, and a release that says so. It watches for nothing itself — no signal
// of the suite's own process reaches it, which is the whole point of the read
// being threaded.
func (s *signalling) notify(...os.Signal) (<-chan os.Signal, func()) {
	return s.arriving, func() { s.once.Do(func() { close(s.stopped) }) }
}

// deliver hands one signal over and waits for the watch to release, which is
// the instant the drain became readable. It is the terminal's Ctrl-C, at a
// moment a case can name.
func (s *signalling) deliver(caught os.Signal) error {
	s.arriving <- caught
	select {
	case <-s.stopped:
		return nil
	case <-time.After(10 * time.Second):
		return errNeverReleased
	}
}

// errNeverReleased is the watch failing to take itself down, which would be a
// second interrupt landing on a handler that had already drained — the one
// thing this design must not do (signals.go).
var errNeverReleased = &neverReleased{}

type neverReleased struct{}

func (*neverReleased) Error() string {
	return "the watch never released after the first signal: a second interrupt would not reach the process"
}

// performer is which of the two ways a Step reaches the world hands the signal
// over: a connection for a Run whose Steps reach a host, a child for one whose
// Steps run a command.
//
// It is the case's own shape rather than an option — a `shell` case dials
// nothing — and it is named by the case rather than inferred so that a case
// that started reaching the world differently fails outright rather than
// quietly stopping delivering anything.
type performer int

const (
	atTheFirstConnection performer = iota
	atTheFirstChild
)

// stopped drives one corpus case with a signal delivered from inside its first
// Step, and answers the exit code, the page, and the branch the Run left.
func stopped(t *testing.T, name string, caught os.Signal, deliverAt performer) (exit int, page, branch string) {
	t.Helper()

	dir := filepath.Join("testdata", "run", name)
	c := goldenCase{dir: dir, name: "run/" + name, argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	signals := watchedBy()
	process.Notify = signals.notify

	// The delivery, wrapped around whichever performer the case's first Step
	// reaches the world through. It fires once: what a case is about is the
	// **first** interrupt, and a second delivery would be a second signal
	// arriving at a watch that is already gone.
	var once sync.Once
	var faulted error
	fire := func() {
		once.Do(func() {
			if err := signals.deliver(caught); err != nil {
				faulted = err
			}
		})
	}
	switch deliverAt {
	case atTheFirstConnection:
		dial := process.Dial
		process.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			fire()
			return dial(ctx, network, address)
		}
	case atTheFirstChild:
		launch := process.Exec
		process.Exec = func(ctx context.Context, argv []string) *exec.Cmd {
			fire()
			return launch(ctx, argv)
		}
	}

	var stdout, stderr bytes.Buffer
	exit = cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t))
	if faulted != nil {
		t.Fatal(faulted)
	}
	return exit, stdout.String(), invocation.fixture.render(t, invocation.fixture.root)
}

// oneSpaced collapses a table's column padding, so that a case says which
// cells a row holds rather than how wide the widest cell in it happened to be.
// What the padding is is §8's and the golden corpus asserts it; what these
// cases are about is the Disposition each Step reached.
func oneSpaced(page string) string {
	return strings.Join(strings.Fields(page), " ")
}

// TestSignals_TheFirstInterruptDrains is §6's sentence in full, driven over the
// corpus's two-Step `read` Run: the signal lands while Step 1 is on the wire,
// Step 1 finishes and is recorded, Step 2 never starts, and the Run closes its
// own entry `failed` at `130`.
func TestSignals_TheFirstInterruptDrains(t *testing.T) {
	exit, page, branch := stopped(t, "two-read-steps-push-once", os.Interrupt, atTheFirstConnection)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d — a Run stopped by an interrupt, having drained", exit, cli.ExitInterrupted)
	}

	// The drained Step's Disposition is *ran* like any other completed
	// Step's: its outcome came back, and a Run that stopped after it is not
	// a reason to record it as anything else.
	if !strings.Contains(oneSpaced(page), "1 status read ran 1") {
		t.Errorf("the Step in flight does not render *ran* with its one Record:\n%s", page)
	}
	// Every Step after it is *never reached*, and the dash beside it is *no
	// set exists* rather than a set with nothing in it (§8, ADR-0030).
	if !strings.Contains(oneSpaced(page), "2 cert read never-reached –") {
		t.Errorf("the Step after the drain does not render *never reached*:\n%s", page)
	}
	if !strings.Contains(page, "failed · exit 130 · run ") {
		t.Errorf("the terminal line does not name the stop:\n%s", page)
	}
	// It is not a rehearsal, so the marker is not on the line. The one path
	// where a `dry-run` could appear without the flag is this one, the
	// terminal line being built after the Run rather than before it.
	if strings.Contains(page, "dry-run") {
		t.Errorf("the terminal line carries the rehearsal marker on an ordinary Run:\n%s", page)
	}

	// The record: Step 1's file and its Record are on the branch, Step 2
	// wrote nothing at all, and the entry is **closed** — `outcome.json` is
	// the Run's own account, written by the Run rather than by anybody's
	// reap.
	assertBranch(t, branch, present("steps/0001.json"), present("status.hyper.dev"),
		absent("steps/0002.json"), absent("cert.hyper.dev"),
		present("outcome.json"), present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_AnInterruptDuringTheLastStepStillStops is the drain where there
// is nothing left to withhold: a Run of **one** Step, interrupted while that
// Step is on the wire.
//
// The Step finishes and is recorded like any other drained Step, and the Run is
// `failed` at `130` all the same. §6 puts an interrupt in `failed` beside an
// error and a deadline — "the world resisting **or the Run being stopped**" —
// so a Run somebody stopped may not answer `0`, whatever it managed to finish
// on the way. A wrapper reading the code would otherwise read a Run that was
// never stopped at all.
func TestSignals_AnInterruptDuringTheLastStepStillStops(t *testing.T) {
	exit, page, branch := stopped(t, "the-tracer-bullet", os.Interrupt, atTheFirstConnection)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d — the Run was stopped, and a Step having finished does not unstop it", exit, cli.ExitInterrupted)
	}
	if !strings.Contains(oneSpaced(page), "1 status read ran 1") {
		t.Errorf("the Step in flight does not render *ran* with its one Record:\n%s", page)
	}
	if !strings.Contains(page, "failed · exit 130 · run ") {
		t.Errorf("the terminal line does not name the stop:\n%s", page)
	}
	// Every Step ran, so there is no *never reached* row and no Step file
	// missing from the entry: what carries the stop is the outcome and the
	// exit code, and nothing else differs from a Run that completed.
	if strings.Contains(page, "never-reached") {
		t.Errorf("a Run of one Step renders a Step it never reached:\n%s", page)
	}
	assertBranch(t, branch, present("steps/0001.json"), present("status.hyper.dev"),
		present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_TheFirstInterruptDrainsASerialDestroy is the same sentence where
// it is worth the most: the Step in flight is a `destroy`, so what draining
// costs is a bounded wait and what it buys is a stop that is recorded in full
// rather than an ambiguity (§6, ADR-0015, issue #150).
//
// The signal lands on the destroy Step's first connection. That Step finishes
// and is *ran*, its Tombstone is on the branch, the `mutate` Step after it never
// starts and writes no file, and the Run closes its **own** entry `failed` at
// `130`.
func TestSignals_TheFirstInterruptDrainsASerialDestroy(t *testing.T) {
	exit, page, branch := stopped(t, "a-destroy-then-a-create-reads-alive-again", os.Interrupt, atTheFirstConnection)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d — a Run stopped by an interrupt, having drained", exit, cli.ExitInterrupted)
	}
	if !strings.Contains(oneSpaced(page), "1 retire destroy ran 1") {
		t.Errorf("the destroy in flight does not render *ran* with the Asset it confirmed:\n%s", page)
	}
	if !strings.Contains(oneSpaced(page), "2 publish mutate never-reached –") {
		t.Errorf("the Step after the drain does not render *never reached*:\n%s", page)
	}
	if !strings.Contains(page, "failed · exit 130 · run ") {
		t.Errorf("the terminal line does not name the stop:\n%s", page)
	}
	assertBranch(t, branch, present("steps/0001.json"), absent("steps/0002.json"),
		present(`"tombstone": true`), present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_ADrainedDestroyFinishesItsWholeExpansion is what *the Step in
// flight finishes* means where that Step has five members: the interrupt lands
// on the **first** member's connection, and the four after it are called all
// the same.
//
// A drain is read where the next Step would start and nowhere else (§6, run.go),
// so an Expansion is not a place a Run stops half-way — which is what keeps
// *three of five* the account of a world that resisted rather than of a
// terminal somebody pressed Ctrl-C at.
func TestSignals_ADrainedDestroyFinishesItsWholeExpansion(t *testing.T) {
	exit, page, branch := stopped(t, "a-destroy-expansion-is-serial", os.Interrupt, atTheFirstConnection)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d", exit, cli.ExitInterrupted)
	}
	if !strings.Contains(oneSpaced(page), "1 retire destroy ran 5") {
		t.Errorf("the destroy in flight did not finish its Expansion:\n%s", page)
	}
	// Five Tombstones, one per member — and the count is read off the page
	// above rather than off the branch, `RECORDS` being the size of the
	// identity set (§8, ADR-0030).
	if !strings.Contains(branch, `"tombstone": true`) {
		t.Errorf("the drained destroy left no Tombstone on the branch")
	}
	assertBranch(t, branch, present("steps/0001.json"), present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_ATerminationSignalDrainsTheSameWay is the second of §12's two
// codes, and the only thing that differs is the number: a termination signal is
// handled exactly as an interrupt is and exits `143`.
func TestSignals_ATerminationSignalDrainsTheSameWay(t *testing.T) {
	exit, page, branch := stopped(t, "two-read-steps-do-not-overlap", syscall.SIGTERM, atTheFirstConnection)

	if exit != cli.ExitTerminated {
		t.Errorf("exit = %d, want %d — a Run stopped by a termination signal, drained the same way", exit, cli.ExitTerminated)
	}
	if !strings.Contains(page, "failed · exit 143 · run ") {
		t.Errorf("the terminal line does not name the stop:\n%s", page)
	}
	assertBranch(t, branch, present("steps/0001.json"), absent("steps/0002.json"),
		present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_AShellStepInFlightFinishes is the claim m5.1 put the child's own
// process group in `Exec` for, made assertable here (§6, issue #142).
//
// In `hyper`'s group a terminal's interrupt would reach the child directly and
// it would die at once, so the Step in flight would not finish and the drain
// would be a sentence the implementation contradicts. The child being in a
// group of its own is what makes it true — and what it costs is the other half
// of the same fact: a second interrupt kills `hyper` and leaves that command
// running with nothing watching it. `hyper` never claims to have stopped a
// command it started.
func TestSignals_AShellStepInFlightFinishes(t *testing.T) {
	exit, page, branch := stopped(t, "two-steps-running-one-argv-write-two-versions", os.Interrupt, atTheFirstChild)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d", exit, cli.ExitInterrupted)
	}
	if !strings.Contains(oneSpaced(page), "1 first read ran 1") {
		t.Errorf("the `shell` Step in flight did not finish:\n%s", page)
	}
	if !strings.Contains(oneSpaced(page), "2 second read never-reached –") {
		t.Errorf("the Step after the drain does not render *never reached*:\n%s", page)
	}
	assertBranch(t, branch, present("steps/0001.json"), absent("steps/0002.json"),
		present(`"outcome": "failed"`))
}

// TestSignals_AnEffectfulShellStepInFlightFinishes is the same claim where the
// child is changing the machine rather than reading it (§6, issue #156).
//
// The drain is what makes an effectful `shell` Step's stop readable at all: the
// child is in a process group of its own, so the terminal's interrupt does not
// reach it, and `hyper` waits for the command it started rather than leaving a
// half-finished effect behind an *attempted, outcome unknown* nobody can act on.
// The Step is *ran*, its Asset is on the branch, the Step after it never starts,
// and the Run closes its own entry `failed` at `130`.
func TestSignals_AnEffectfulShellStepInFlightFinishes(t *testing.T) {
	exit, page, branch := stopped(t, "two-shell-mutate-steps-land-two-assets", os.Interrupt, atTheFirstChild)

	if exit != cli.ExitInterrupted {
		t.Errorf("exit = %d, want %d", exit, cli.ExitInterrupted)
	}
	if !strings.Contains(oneSpaced(page), "1 deploy mutate ran 1") {
		t.Errorf("the effectful `shell` Step in flight did not finish:\n%s", page)
	}
	if !strings.Contains(oneSpaced(page), "2 tag mutate never-reached –") {
		t.Errorf("the Step after the drain does not render *never reached*:\n%s", page)
	}
	if !strings.Contains(page, "failed · exit 130 · run ") {
		t.Errorf("the terminal line does not name the stop:\n%s", page)
	}
	// The Asset the drained Step landed is on the branch under the argv that
	// made it, and the Step after it wrote nothing at all.
	assertBranch(t, branch, present("steps/0001.json"), absent("steps/0002.json"),
		present(`%5B%22deploy%22%2C%22r41%22%5D`), absent(`%5B%22tag%22%2C%22r41%22%5D`),
		present(`"record_type": "asset"`), present(`"outcome": "failed"`), absent("closed-by"))
}

// TestSignals_AnEntryHoldsNoAccountUntilItsRunWritesOne is what the **second**
// interrupt leaves, asserted at the one moment it can be: mid-Run (§6, §7,
// ADR-0003).
//
// A second interrupt kills the process outright, so there is no code path to
// drive and nothing for `hyper` to write — what it leaves is whatever the
// branch already held. This reads the branch from inside Step 1's own call,
// which is exactly the state a kill at that instant would freeze: `run.json`
// and no account beside it, neither an `outcome.json` its own Run wrote nor a
// `closed-by/` file another Run wrote. That absence **is** the representation:
// there is no reaper, no daemon and no heartbeat, and `hyper` never guesses
// whether the Run is in flight or its process is gone.
func TestSignals_AnEntryHoldsNoAccountUntilItsRunWritesOne(t *testing.T) {
	dir := filepath.Join("testdata", "run", "two-read-steps-push-once")
	c := goldenCase{dir: dir, name: "run/two-read-steps-push-once", argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	var once sync.Once
	var midRun string
	var faulted error
	dial := process.Dial
	process.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		once.Do(func() { midRun, faulted = branchPaths(invocation.fixture) })
		return dial(ctx, network, address)
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != cli.ExitClean {
		t.Fatalf("exit = %d; stderr: %s", exit, stderr.String())
	}
	if faulted != nil {
		t.Fatal(faulted)
	}

	assertBranch(t, midRun, present("run.json"), absent("outcome.json"), absent("closed-by"))

	// And the same Run, having reached its end, closed its own entry: the
	// account is `outcome.json` and it is the Run's own. This Run is
	// read-only, so it reaps nothing whatever it finds open — and the entry
	// it left is the one a later effectful Run would close, which
	// [run_reap_test.go](run_reap_test.go) drives end to end.
	after, err := branchPaths(invocation.fixture)
	if err != nil {
		t.Fatal(err)
	}
	assertBranch(t, after, present("outcome.json"), absent("closed-by"))
}

// branchPaths is the Store branch's file list, read without stopping the test:
// it is called from inside a Run's own call, on a goroutine that is not the
// test's, where a t.Fatal would leave the Run running and the suite guessing.
func branchPaths(fixture gitFixture) (string, error) {
	listing := exec.Command("git", "ls-tree", "-r", "--name-only", "refs/heads/hyper-store")
	listing.Dir = fixture.root
	listing.Env = fixture.env
	out, err := listing.Output()
	return string(out), err
}

// assertBranch holds a rendering of the branch — its file list, or its files
// and their bytes — against what a case says must and must not be in it.
func assertBranch(t *testing.T, branch string, expected ...expectation) {
	t.Helper()
	for _, want := range expected {
		if strings.Contains(branch, want.text) != want.present {
			t.Errorf("the branch %s %q:\n%s", map[bool]string{true: "does not hold", false: "holds"}[want.present], want.text, branch)
		}
	}
}

// expectation is one thing a case says about the branch, and whether it is
// there. The pair is one type rather than two lists so that a case reads as the
// sentence it is asserting, in the order it says it.
type expectation struct {
	text    string
	present bool
}

func present(text string) expectation { return expectation{text: text, present: true} }

func absent(text string) expectation { return expectation{text: text} }
