package cli_test

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **How much of a `read` Step's Expansion runs at once, watched on the wire**
// (§6, ADR-0002, ADR-0045, issue #140).
//
// No golden can hold any of this, and for the reason `run_push_test.go`'s
// cannot hold its own: what a limit, a dispatch order and a *two Steps never
// overlap* are about is **when** calls happened, and a branch and a page record
// only that they did — [testdata/run/README.md] states it there beside the
// cases these drive.
//
// So the wire is watched instead. Each case drives a corpus case through
// `cli.Main` with the harness's own dialer wrapped: every connection the Run
// makes announces itself, waits for as many others as the case says should be
// able to stand beside it, and the peak is read off afterwards. A limit that is
// never reached would report a peak of one whatever the Manifest declared,
// which is why every case here says how wide it expects the Expansion to stand
// and holds each connection until that many are there.
//
// The corpus case supplies the repository, the hosts and the mint;
// [testdata/run/repo-concurrency] holds three Operations — one declaring
// `concurrency: 4`, one declaring `2`, and one declaring nothing at all — over
// eight granted hosts, so what differs between these cases is one Manifest key
// and nothing else.

// dialWatch is what the Run's connections are watched through: how many stood
// together at the widest, the order they arrived in, and the host each one
// reached.
//
// width is how many the case expects to be able to stand at once, and hold is
// how long a connection waits for them before giving up and going on. A case
// that expects one waits the hold out on every connection, which is what makes
// *nothing stood beside it* an observation rather than an absence of evidence.
type dialWatch struct {
	mutex   sync.Mutex
	inside  int
	peak    int
	arrived []string
	events  []string
	width   int
	hold    time.Duration
	// pending is the connections standing at the gate that have not been
	// let go yet, and it is what the rendezvous counts. It counts
	// **arrivals** rather than what happens to be in flight: a connection
	// the previous batch has not finished with is not a member of the next
	// one, and counting occupancy instead would let one batch borrow the
	// tail of the one before it and leave the last few waiting for a width
	// that can never arrive.
	pending []string
	gates   map[string]chan struct{}
	// made is one channel per connection, closed when that connection has
	// been made. It is what lets a release wait for the connection it just
	// let go before letting the next one go, which is what makes the order
	// a case names the order the dials actually return in rather than an
	// order it hoped a sleep would produce.
	made map[string]chan struct{}
	// release, where a case supplies one, is asked for the order a batch is
	// let go in. It is how a Run's calls are made to complete in an order
	// the Expansion did not fix.
	release func(batch []string) []string
}

// watching wraps a dialer, answering the watch beside it.
func watching(dial capability.Dial, width int, hold time.Duration) (capability.Dial, *dialWatch) {
	watch := &dialWatch{width: width, hold: hold, gates: map[string]chan struct{}{}, made: map[string]chan struct{}{}}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		watch.enter(host)
		defer watch.leave(host)
		return dial(ctx, network, address)
	}, watch
}

// enter records one connection arriving and holds it at the gate until the
// width the case named is standing together, or the hold elapses.
func (w *dialWatch) enter(host string) {
	w.mutex.Lock()
	w.inside++
	w.peak = max(w.peak, w.inside)
	w.arrived = append(w.arrived, host)
	w.events = append(w.events, "dial "+host)
	gate := make(chan struct{})
	w.gates[host] = gate
	w.made[host] = make(chan struct{})
	w.pending = append(w.pending, host)
	batch, full, order := w.pending, len(w.pending) >= w.width, w.release
	if full {
		w.pending = nil
	}
	w.mutex.Unlock()

	if !full {
		select {
		case <-gate:
		case <-time.After(w.hold):
			// Nothing ever stood beside it, which is the case's
			// finding rather than a hang: it drops out of the batch
			// and lets itself go.
			w.mutex.Lock()
			w.pending = slices.DeleteFunc(w.pending, func(waiting string) bool { return waiting == host })
			w.mutex.Unlock()
			w.open([]string{host})
		}
		return
	}

	// The batch is complete. Where the case named no order they all go at
	// once; where it named one they are let go one at a time, each waiting
	// for the connection before it to have been made — so the order the
	// dials return in is exactly the order the case named, and the calls
	// that follow run in it rather than in the one the Expansion dispatched
	// them in.
	if order == nil {
		w.open(batch)
		return
	}
	go func() {
		for _, held := range order(batch) {
			w.mutex.Lock()
			settled := w.made[held]
			w.mutex.Unlock()
			w.open([]string{held})
			<-settled
		}
	}()
	<-gate
}

// open lets the named connections go.
func (w *dialWatch) open(hosts []string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for _, host := range hosts {
		if gate, held := w.gates[host]; held {
			close(gate)
			delete(w.gates, host)
		}
	}
}

// leave records one connection being made.
func (w *dialWatch) leave(host string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.inside--
	w.events = append(w.events, "made "+host)
	if settled, held := w.made[host]; held {
		close(settled)
		delete(w.made, host)
	}
}

// drive runs one corpus case with its dialer watched, and answers what stdout
// held, the branch the Run left, and the watch.
func drive(t *testing.T, name string, width int, hold time.Duration, release func([]string) []string) (string, string, *dialWatch) {
	t.Helper()

	dir := filepath.Join("testdata", "run", name)
	c := goldenCase{dir: dir, name: "run/" + name, argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	dial, watch := watching(process.Dial, width, hold)
	watch.release = release
	process.Dial = dial

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
		t.Fatalf("%s: exit = %d; stderr: %s", name, exit, stderr.String())
	}
	return stdout.String(), invocation.fixture.render(t, invocation.fixture.root), watch
}

// TestConcurrency_TheDeclaredLimitBoundsWhatIsInFlight is ADR-0045's limit on
// the wire: an Operation declaring `concurrency: 4`, expanded over eight
// members, stands exactly four connections together and never a fifth.
func TestConcurrency_TheDeclaredLimitBoundsWhatIsInFlight(t *testing.T) {
	_, _, watch := drive(t, "eight-members-under-a-limit-of-four", 4, 2*time.Second, nil)

	if watch.peak != 4 {
		t.Errorf("%d connections stood together, want exactly 4 — the Manifest declares concurrency: 4", watch.peak)
	}
	if len(watch.arrived) != 8 {
		t.Errorf("%d connections were made, want 8 — one per member of the Expansion", len(watch.arrived))
	}
}

// TestConcurrency_MembersAreDispatchedInExpansionOrder is §6's *the order above
// is the order members are dispatched in*: under a limit of four the first four
// connections are the first four members of the Expansion, which is what fixes
// the first ten of five hundred.
func TestConcurrency_MembersAreDispatchedInExpansionOrder(t *testing.T) {
	_, _, watch := drive(t, "eight-members-under-a-limit-of-four", 4, 2*time.Second, nil)

	// Which of the four arrived first is a scheduler's business and nothing
	// derives from it; **which four** is the Expansion's, and is what this
	// reads.
	first := slices.Clone(watch.arrived[:4])
	slices.Sort(first)
	want := []string{"site-1.hyper.dev", "site-2.hyper.dev", "site-3.hyper.dev", "site-4.hyper.dev"}
	if !slices.Equal(first, want) {
		t.Errorf("the first four members dispatched were %v, want %v — the Expansion's own first four", first, want)
	}
}

// TestConcurrency_AManifestDeclaringNothingRunsSerially is the other half of
// ADR-0045: a `read` whose Provider author said nothing runs its Expansion one
// member at a time, and there is nothing anywhere else to say otherwise.
//
// The environment is loaded with every name an implementer might reach for, so
// that *no environment override* is an observation rather than a claim: the
// engine reads no environment variable of its own at all (run_test.go holds
// that over its source), and this holds the consequence at the wire.
func TestConcurrency_AManifestDeclaringNothingRunsSerially(t *testing.T) {
	t.Setenv("HYPER_CONCURRENCY", "8")
	t.Setenv("HYPER_MAX_IN_FLIGHT", "8")
	t.Setenv("CONCURRENCY", "8")

	// A width of two, so a second connection standing beside the first
	// would be seen. None does, and each connection pays the hold out.
	_, _, watch := drive(t, "an-expansion-with-no-declared-limit", 2, 30*time.Millisecond, nil)

	if watch.peak != 1 {
		t.Errorf("%d connections stood together under a Manifest declaring no concurrency:, want 1", watch.peak)
	}
	if len(watch.arrived) != 3 {
		t.Errorf("%d connections were made, want 3 — one per member", len(watch.arrived))
	}
}

// TestConcurrency_TwoStepsNeverOverlap is ADR-0002 on the wire: all concurrency
// lives inside one Step's Expansion, so the second Step starts no member before
// the first has finished all of its.
//
// Both Steps bind an Operation declaring `concurrency: 2` and hold two members
// each. Within a Step the two stand together — which is what makes the
// partition below a fact about Steps rather than a Run that was serial
// throughout — and across the two nothing interleaves.
func TestConcurrency_TwoStepsNeverOverlap(t *testing.T) {
	_, _, watch := drive(t, "two-read-steps-do-not-overlap", 2, 2*time.Second, nil)

	if watch.peak != 2 {
		t.Fatalf("%d connections stood together, want 2 — two members of one Step and never a Step's beside another's", watch.peak)
	}

	// The two Steps' hosts, and where the first Step's last event sits
	// against the second Step's first.
	step := map[string]int{
		"site-1.hyper.dev": 1, "site-2.hyper.dev": 1,
		"site-3.hyper.dev": 2, "site-4.hyper.dev": 2,
	}
	lastOfFirst, firstOfSecond := -1, len(watch.events)
	for at, event := range watch.events {
		host := strings.TrimPrefix(strings.TrimPrefix(event, "dial "), "made ")
		switch step[host] {
		case 1:
			lastOfFirst = max(lastOfFirst, at)
		case 2:
			firstOfSecond = min(firstOfSecond, at)
		}
	}
	if lastOfFirst > firstOfSecond {
		t.Errorf("step 2 reached the wire before step 1 had left it:\n%s", strings.Join(watch.events, "\n"))
	}
}

// TestConcurrency_AStepWithNoSelectorMakesOneCall is §6's last clause on the
// limit: where a Step carries no `over:` there is no selector to resolve, so
// the Step makes one call — a set of one, inside any limit that has ever been
// written. The Operation it binds declares `concurrency: 4` all the same.
func TestConcurrency_AStepWithNoSelectorMakesOneCall(t *testing.T) {
	_, _, watch := drive(t, "a-step-with-no-selector-under-a-limit", 2, 30*time.Millisecond, nil)

	if len(watch.arrived) != 1 {
		t.Errorf("%d connections were made, want 1 — a Step with no selector is a set of one", len(watch.arrived))
	}
	if watch.peak != 1 {
		t.Errorf("%d connections stood together over a set of one", watch.peak)
	}
}

// TestConcurrency_NothingDerivesFromTheOrderTheCallsCompletedIn is the rule
// everything a concurrent Expansion writes rests on: the same Run, driven once
// with its connections let go in the order they arrived and once with them let
// go last-first, produces the same page and the same branch **byte for byte**.
//
// The reversal is real rather than hoped for, and it is exact rather than
// timed: each batch of four rendezvous before any of them is let go, and they
// are then let go one at a time, each waiting for the connection before it to
// have been made. So the order the dials return in is the order the case named
// — asserted below off the watch's own events — and the calls that follow run
// in it.
func TestConcurrency_NothingDerivesFromTheOrderTheCallsCompletedIn(t *testing.T) {
	const expansion = "eight-members-under-a-limit-of-four"

	forwardPage, forwardBranch, forward := drive(t, expansion, 4, 2*time.Second, func(hosts []string) []string {
		return hosts
	})
	reversedPage, reversedBranch, reversed := drive(t, expansion, 4, 2*time.Second, func(hosts []string) []string {
		last := slices.Clone(hosts)
		slices.Reverse(last)
		return last
	})

	if len(forward.arrived) != 8 || len(reversed.arrived) != 8 {
		t.Fatalf("the two Runs made %d and %d connections, want 8 each", len(forward.arrived), len(reversed.arrived))
	}
	// The reversal took effect, which is what makes the comparison below a
	// finding rather than two identical Runs.
	for _, batch := range [][]string{reversed.arrived[:4], reversed.arrived[4:]} {
		want := slices.Clone(batch)
		slices.Reverse(want)
		if got := reversed.settled(batch); !slices.Equal(got, want) {
			t.Fatalf("a batch dispatched %v was made in %v, want %v", batch, got, want)
		}
	}

	if forwardPage != reversedPage {
		t.Errorf("the page moved with the completion order:\n forward: %q\nreversed: %q", forwardPage, reversedPage)
	}
	if forwardBranch != reversedBranch {
		t.Errorf("the branch moved with the completion order:\n forward: %q\nreversed: %q", forwardBranch, reversedBranch)
	}
}

// settled is the order the named connections were made in, read off the watch's
// events.
func (w *dialWatch) settled(hosts []string) []string {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	order := make([]string, 0, len(hosts))
	for _, event := range w.events {
		host, made := strings.CutPrefix(event, "made ")
		if made && slices.Contains(hosts, host) {
			order = append(order, host)
		}
	}
	return order
}

// TestConcurrency_ThereIsNoFlagThatChangesIt is the third of §6's three: no
// flag anywhere changes how much of an Expansion runs at once.
//
// [testdata/run/usage-no-concurrency-flag] holds the same fact as a page — the
// flag is unknown and the invocation is a usage error — and this holds it over
// four more spellings, so a knob that arrived under any of them would be caught
// wherever it was added. The process the invocation is handed dials nothing and
// mints nothing, which is the second half of the answer: a usage error is
// decided before either is reached.
func TestConcurrency_ThereIsNoFlagThatChangesIt(t *testing.T) {
	dir := filepath.Join("testdata", "run", "usage-no-concurrency-flag")
	c := goldenCase{dir: dir, name: "run/usage-no-concurrency-flag", argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	repo := c.abs(t, filepath.Join("..", "repo-concurrency"))

	for _, spelled := range []string{"--concurrency", "--concurrency=8", "--in-flight", "--parallel", "--max-in-flight"} {
		var stdout, stderr bytes.Buffer
		args := []string{"run", "watch-eight", spelled, "8", "--repo-dir", repo}
		exit := cli.Main(args, &stdout, &stderr, c.process(t, invocation), c.facts(t))

		if exit != cli.ExitUsage {
			t.Errorf("hyper run %s exited %d, want %d — no flag reaches the limit", spelled, exit, cli.ExitUsage)
		}
		if !strings.Contains(stderr.String(), "unknown flag") {
			t.Errorf("hyper run %s said %q, want an unknown flag", spelled, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("hyper run %s wrote %q to stdout; a usage error writes nothing there", spelled, stdout.String())
		}
	}
}
