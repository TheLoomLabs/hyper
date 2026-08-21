package cli_test

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **A Pattern's own concurrency, watched on the wire** (§3, §6, ADR-0045, issue
// #143).
//
// No golden can hold it, for the reason `run_concurrency_test.go`'s cannot hold
// its own: *how many requests one member had in flight* is a fact about **when**
// calls happened, and a branch and a page record only that they did. Twelve
// pages fetched four at a time and twelve fetched one member at a time leave the
// same twelve Observations, the same identity set and the same `pattern` block.
//
// So the dials are counted instead. `hyper`'s client disables keep-alives — one
// call is one connection, so that the certificate a response reports is the one
// that call was answered over — which makes one dial to a host exactly one
// request to it. Each dial is **held** briefly before it is let through, so that
// two requests a member had in flight together would be two dials standing
// together and would be seen; with the Pattern serial, page *n+1* is not asked
// for until page *n* has answered, so a member's own dials cannot overlap at
// all.
//
// The hold is the instrument and not the subject. What a connection does after
// it is made is not watched, because a transport closes a spent connection on
// its own schedule and *the old one is not shut yet* is not *two requests are in
// flight*.
//
// What it drives is [testdata/run/four-paginated-members-under-a-limit-of-four]:
// an Expansion of four members over an Operation declaring `concurrency: 4`,
// each member walking three pages. The two claims it separates are the ones
// ADR-0045 keeps apart — the limit governs the **Expansion**, and it reaches no
// Pattern at all.

// TestPattern_AConcurrencyLimitDoesNotReachAPattern is the pair of claims in
// one Run.
//
// **Four members stand together**, which is the declared limit doing what it
// declares — held by a barrier on each host's first connection, so that *the
// Expansion ran wide* is an observation rather than a hope about scheduling.
//
// **No member ever stands twice**, which is the Pattern's own serialism: a
// member is one request at a time from the moment it is dispatched until its
// last page, so *members in flight* and *requests in flight* are one number
// rather than two that would have to agree. It needs no barrier — a second
// connection to one host is the violation, and counting is enough to see one.
func TestPattern_AConcurrencyLimitDoesNotReachAPattern(t *testing.T) {
	const (
		members = 4
		pages   = 3
	)

	dir := filepath.Join("testdata", "run", "four-paginated-members-under-a-limit-of-four")
	c := goldenCase{dir: dir, name: "run/" + filepath.Base(dir), argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	counted := &connectionCount{
		open: map[string]int{}, peak: map[string]int{}, made: map[string]int{},
		width: members, standing: make(chan struct{}),
	}
	process.Dial = counted.watching(process.Dial)

	var stdout, stderr bytes.Buffer
	if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
		t.Fatalf("exit = %d; stderr: %s", exit, stderr.String())
	}

	if counted.widest != members {
		t.Errorf("%d connections stood together, want exactly %d — the Operation declares concurrency: %d over an Expansion of %d",
			counted.widest, members, members, members)
	}
	for host, peak := range counted.peak {
		if peak != 1 {
			t.Errorf("%s had %d requests in flight at once, want 1 — a Pattern is serial by construction and the concurrency: limit reaches none of them", host, peak)
		}
	}
	// With one request per host in flight and one host per member, the
	// width above can be no more than the Expansion — so the two
	// assertions together are *exactly four, one each* rather than a
	// ceiling nothing reached.
	if len(counted.made) != members {
		t.Errorf("%d hosts were reached, want %d — one per member of the Expansion", len(counted.made), members)
	}
	for host, made := range counted.made {
		if made != pages {
			t.Errorf("%s answered %d requests, want %d — one per page the member walked", host, made, pages)
		}
	}
}

// connectionCount is what the Run's connections are counted through: how many
// are open to each host, how many were ever open to one at once, and how many
// stood together across all of them.
//
// The barrier is the one thing here that is not a count. Each host's **first**
// connection waits until as many hosts as the case expects have one standing,
// which is what makes *four members ran together* an observation; every
// connection after that passes straight through, so nothing a Pattern does is
// held up by the instrument watching it.
type connectionCount struct {
	mutex    sync.Mutex
	open     map[string]int
	peak     map[string]int
	made     map[string]int
	inside   int
	widest   int
	width    int
	arrived  int
	standing chan struct{}
}

// watching wraps a dialler so that every dial through it is counted, and held
// long enough for a dial standing beside it to be seen.
func (c *connectionCount) watching(dial capability.Dial) capability.Dial {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		c.enter(host)
		c.leave(host)
		return dial(ctx, network, address)
	}
}

// overlap is how long every dial past a host's first is held. It is long enough
// that two dials one member made together would be seen standing together, and
// short enough that holding twelve of them three deep costs a tenth of a second.
const overlap = 40 * time.Millisecond

// enter records one dial to host and holds it: the **first** dial to each host
// waits until the barrier's width has arrived, which is what makes *four members
// ran together* an observation, and every dial after that waits the overlap out,
// which is what makes *this member stood alone* one.
func (c *connectionCount) enter(host string) {
	c.mutex.Lock()
	c.open[host]++
	c.peak[host] = max(c.peak[host], c.open[host])
	c.made[host]++
	c.inside++
	c.widest = max(c.widest, c.inside)
	first := c.made[host] == 1
	if first {
		c.arrived++
		if c.arrived == c.width {
			close(c.standing)
		}
	}
	standing := c.standing
	c.mutex.Unlock()

	if !first {
		time.Sleep(overlap)
		return
	}
	// A width that never arrives is a Run that did not run its Expansion
	// wide, and the wait is bounded so that it reads as a failing assertion
	// rather than as a suite that hangs.
	select {
	case <-standing:
	case <-time.After(5 * time.Second):
	}
}

// leave records one held dial being let through.
func (c *connectionCount) leave(host string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.open[host]--
	c.inside--
}

// TestPattern_TheIdentitySetIsTheSameWithAndWithoutOne is §3's rule that no
// surface can state on its own: **a Pattern may never change an Operation's
// Kind or the number of Records it affects**.
//
// The two cases it reads are one Step against one host under two Operations
// that differ in one key. The host refuses its first two connections, so the
// Pattern is not decorative: with `retry: {attempts: 3}` the third call answers
// and the Observation carries a `status`, and with no retry at all the
// Observation records the silence. What `hyper` was accountable for is
// identical either way — one Record, under one name — which is what the digest
// being the same byte for byte says.
//
// It is a test rather than a sentence in the corpus README because the claim is
// a comparison between two cases, and a golden holds one.
func TestPattern_TheIdentitySetIsTheSameWithAndWithoutOne(t *testing.T) {
	withRetry := identitySetOf(t, "a-retry-follows-a-refused-connection")
	without := identitySetOf(t, "the-same-host-with-no-retry-declared")

	if withRetry != without {
		t.Errorf("the identity set with a retry Pattern is\n%s\nand without one is\n%s\n— a Pattern may not change the number of Records a Step affects", withRetry, without)
	}
	if withRetry == "" {
		t.Fatal("neither case's Step file carries an identity set; the comparison asserted nothing")
	}
}

// identitySetOf is the `identities` block of one case's first Step, read out of
// the branch golden the case checks in.
//
// It reads the golden rather than driving the Run again for the reason the
// golden exists: what a Run left on the branch is already a checked-in
// constant, and a second Run of it would be asserting that two Runs of one
// fixture agree.
func identitySetOf(t *testing.T, name string) string {
	t.Helper()

	rendered := readFile(t, filepath.Join("testdata", "run", name, "store.golden"))
	step := strings.Index(rendered, "/steps/0001.json")
	if step < 0 {
		t.Fatalf("%s: the branch holds no first Step file", name)
	}
	block := strings.Index(rendered[step:], `"identities": {`)
	if block < 0 {
		t.Fatalf("%s: the first Step file carries no identity set", name)
	}
	held := rendered[step+block:]
	end := strings.Index(held, "\n  },")
	if end < 0 {
		t.Fatalf("%s: the identity set does not close", name)
	}
	return held[:end]
}

// TestPolling_TheIntervalFallsBetweenTheCallsAndNeverBeforeTheFirst is the one
// claim about a Pattern that needs a clock, asserted where a test may read one:
// the elapsed time of a whole Run.
//
// It is two cases because the claim is two. `repo-patterns` declares
// `interval: 1s`, which is the smallest a duration can be written as short of
// the `0s` that would wait for nothing, and:
//
//   - the host that answers `pending` and then `ready` takes **two** calls, so
//     a Run of it cannot finish inside the interval;
//   - the host whose `state` cannot be compared halts on the **first** call, so
//     a Run of it must finish well inside one — a Pattern that slept before
//     asking would put its interval on every call an artefact makes.
//
// It drives the Runs rather than reading the goldens, elapsed time being the
// one thing a branch does not record. Their pages and branches are asserted by
// the corpus (testdata/run/README.md); what is read here is only the clock.
func TestPolling_TheIntervalFallsBetweenTheCallsAndNeverBeforeTheFirst(t *testing.T) {
	const interval = time.Second

	if polled := elapsed(t, "a-poll-stops-when-its-until-holds"); polled < interval {
		t.Errorf("two polls took %v, want at least the declared %v between them", polled, interval)
	}
	if halted := elapsed(t, "an-until-that-cannot-compare"); halted >= interval {
		t.Errorf("one poll took %v, want well inside the declared %v — the interval is waited between calls and not before the first", halted, interval)
	}
}

// elapsed drives one corpus case and answers how long the Run took. It asserts
// nothing about what the Run did: the case's own goldens hold that, and this
// reads the one fact they cannot.
func elapsed(t *testing.T, name string) time.Duration {
	t.Helper()

	dir := filepath.Join("testdata", "run", name)
	c := goldenCase{dir: dir, name: "run/" + name, argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)

	var stdout, stderr bytes.Buffer
	began := time.Now()
	cli.Main(invocation.args, &stdout, &stderr, c.process(t, invocation), c.facts(t))
	return time.Since(began)
}
