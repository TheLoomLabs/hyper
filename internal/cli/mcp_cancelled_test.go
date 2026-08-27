package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
)

// **A cancelled call drains** (§6, §9, ADR-0015, ADR-0092, issue #202).
//
// The client cancels a call by cancelling the request; the SDK cancels the
// handler's context; and that is mapped onto the drain the first interrupt
// already performs — the Step in flight finishes, no further Step starts, and
// the Run closes its own Journal entry `failed`. The engine's gate for it is
// already a predicate rather than a signal, so what the mapping adds is a
// predicate that reads a context, and **no second stopping mechanism enters the
// engine**.
//
// The delivery is driven the way [run_signal_test.go](run_signal_test.go)
// delivers a signal: the same corpus cases, the same door a client comes
// through, with one more input supplied by the driver rather than by a file. A
// signal is a fact about *when* it arrived and a cancellation is the same,
// which no file beside a `call` can state.
//
// It is deterministic rather than timed, and that is what makes it assertable.
// The cancellation is handed over from inside the Step's own call, and the
// driver does not let that call return until the **handler's** context is
// actually done — the SDK carrying the client's cancellation across on a
// goroutine of its own, so a driver that let the Step finish first would be
// racing it. There is no sleep anywhere here, and no case can pass by racing.

// cancellation is one call's stop as a driver delivers it: the context the
// dispatch was handed, and the cancel the driver fires from inside a Step.
//
// The context is the **handler's own** and not the client's. A driver that
// waited on the one it cancelled would be waiting on the moment it asked, where
// what the drain reads is the moment the server was told.
type cancellation struct {
	called chan context.Context
	cancel context.CancelFunc
	once   sync.Once
	fault  error
}

// deliver cancels the call and waits until the call's own context is done,
// which is the instant the drain became readable. It is the client giving up,
// at a moment a case can name.
func (c *cancellation) deliver() {
	c.once.Do(func() {
		c.cancel()
		select {
		case <-(<-c.called).Done():
		case <-time.After(10 * time.Second):
			c.fault = errNeverCancelled
		}
	})
}

// errNeverCancelled is the client's cancellation failing to reach the handler,
// which would leave this driver asserting a drain that never had anything to
// read.
var errNeverCancelled = errors.New("the call's context was never cancelled: the client gave up and the handler was not told")

// cancelledMidStep drives one case of the MCP corpus as a tool call, cancels
// the call from inside its first Step, and answers the error the client was
// left with and the branch the Run left behind.
//
// **The server is the one the binary starts**, with one observer in front of
// the dispatch: the tool set is internal/mcp's own table and the dispatch is
// `cli.MCPDispatch`, exactly as `cli.MCPServer` assembles them. What the
// observer does is hold the call's context so the delivery can wait on it,
// which is the same fixture in kind as the tee the corpus reads its envelopes
// off (mcp.Server.Call).
func cancelledMidStep(t *testing.T, name string) (branch string, err error) {
	t.Helper()

	c := corpusCase(t, name)
	invocation := c.invocation(t)
	process := c.process(t, invocation)
	facts := c.facts(t)

	ctx, cancel := context.WithCancel(t.Context())
	stopping := &cancellation{called: make(chan context.Context, 1), cancel: cancel}

	// **The signals, supplied so that reaching for them fails the case.**
	// The server installs no signal watch — the process's signals belong to
	// the client that spawned it and not to any one call in flight — so a
	// Run reached through a tool is one nobody interrupts by signal, and
	// exit codes `130` and `143` are unreachable from here (§6, §12,
	// ADR-0015). A watch installed anyway would take this arm.
	process.Notify = func(...os.Signal) (<-chan os.Signal, func()) {
		t.Error("the server installed a signal watch: the process's signals belong to the client, not to a call in flight")
		return nil, func() {}
	}

	// The delivery, wrapped around the first connection the Run's first Step
	// makes — the performer run_signal_test.go names, on the cases whose
	// Steps reach a host. It fires once: what a case is about is the client
	// giving up, and a second delivery would be cancelling a call that is
	// already over.
	dial := process.Dial
	process.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		stopping.deliver()
		return dial(ctx, network, address)
	}

	dispatch := cli.MCPDispatch(process, facts)
	server := mcp.NewServer(facts.Version, func(call mcp.Call) mcp.Answer {
		stopping.called <- call.Context
		return dispatch(call)
	})

	// Call returns only once the handler has returned, the server session
	// being closed last and waiting for what is in flight (mcp.Server.Watched)
	// — so the branch read below is the branch the Run finished leaving.
	_, err = server.Call(ctx, c.call.Tool, c.call.Arguments)
	if stopping.fault != nil {
		t.Fatal(stopping.fault)
	}
	return invocation.fixture.render(t, invocation.fixture.root), err
}

// TestCancelled_ACancelledCallDrains is §6's sentence in full, on the surface
// that has no signal to state it with: the client cancels while Step 1 is on
// the wire, Step 1 finishes and is recorded, the Steps after it never start,
// and the Run closes its **own** entry `failed`.
//
// The Run it drives is the three-Step Procedure the corpus already holds as a
// completed call, which is what makes the branch legible: with nobody
// cancelling, all three Steps write a file, and here the second and third write
// nothing at all.
func TestCancelled_ACancelledCallDrains(t *testing.T) {
	branch, err := cancelledMidStep(t, "mcp/run/a-skip-propagates")

	// **A client that gives up gets no delivery at all**, which is the whole
	// of what it gets: the envelope the handler composed has nowhere to go,
	// and what the client is left with is its own cancellation.
	if err == nil {
		t.Error("the cancelled call came back with an answer; a client that gives up is not delivered one")
	}

	// The Step in flight finished and is recorded, the Steps behind it wrote
	// nothing, and the account of how the Run ended is the Run's own —
	// `outcome.json`, and no `closed-by/` file anybody's reap wrote (§6, §7).
	assertBranch(t, branch,
		present("steps/0001.json"), present("status.hyper.dev"),
		absent("steps/0002.json"), absent("steps/0003.json"),
		present("outcome.json"), present(`"outcome": "failed"`), absent("closed-by"))
}

// TestCancelled_TheStopCarriesNoSignalsCode is the other half of *the server
// installs no signal watch*: what a drained Run answers with here.
//
// §12's `130` and `143` are decided where a signal was caught, and nothing on
// this surface catches one — so a cancelled Run is `failed` on the ordinary
// code, and the envelope carries the triple while the Journal carries the
// truthful account. That costs nothing: a caller that reads the outcome learns
// the same thing the number would have told it, and one that reads the entry
// learns more.
//
// It drives the dispatch with a context that is **already** done, which is the
// same predicate answering at the same place: the drain is read where the next
// Step would start, and before Step 1 is one of those places (§6, run.go).
func TestCancelled_TheStopCarriesNoSignalsCode(t *testing.T) {
	answered, branch := cancelledBeforeTheFirstStep(t, "mcp/run/a-skip-propagates")

	switch answered.Exit {
	case cli.ExitInterrupted, cli.ExitTerminated:
		t.Errorf("a cancelled call exited %d; the two signals' codes are unreachable from a server that watches none", answered.Exit)
	case cli.ExitProblems:
	default:
		t.Errorf("a cancelled call exited %d, want %d — the Run was stopped, and a stopped Run is failed", answered.Exit, cli.ExitProblems)
	}

	// No Step started at all, so the entry names the Run and its Steps are
	// every one of them *never reached* — and it is closed all the same.
	assertBranch(t, branch, absent("steps/0001.json"), present("run.json"),
		present("outcome.json"), present(`"outcome": "failed"`))
}

// TestCancelled_TheDrainIsTheEnginesOwnPredicate is the claim that no second
// stopping mechanism entered the engine, held where it can be: the sentinel a
// drained Run carries.
//
// internal/run answers one question at one place — *has this Run been stopped
// by now*, asked where the next Step would start — and `ErrInterrupted` is what
// it stops with. A cancelled call reaching that sentinel is the mapping working
// as stated: the surface composed the predicate and the engine gained nothing
// (run.Request.Interrupted, cli.stopping).
func TestCancelled_TheDrainIsTheEnginesOwnPredicate(t *testing.T) {
	answered, _ := cancelledBeforeTheFirstStep(t, "mcp/run/a-skip-propagates")

	// The narration is where a Run's fault is written, a failure carrying no
	// `error_code` (§9, §12). What it says is the engine's own sentence for
	// a Run that drained.
	if !strings.Contains(answered.Narration, "no further Step was started") {
		t.Errorf("the cancelled Run did not stop on the engine's drain:\n%s", answered.Narration)
	}
}

// cancelledBeforeTheFirstStep drives one case's Run through the dispatch under a
// context that is **already** done, and answers what the command returned and
// the branch it left.
//
// It reaches the dispatch rather than the wire because what these two cases are
// about is the answer a cancelled call composes — the exit code §12 fixes, and
// the fault the engine stopped with — and a client that has given up receives
// neither: it is delivered nothing at all, which is the case above.
//
// The drain is read where the next Step would start, and **before Step 1 is one
// of those places**, so a context that was done on arrival stops the Run there
// (§6, run.go). The Procedure is read off the case's own `call` rather than
// spelled here, so a corpus file that changes takes the driver with it.
func cancelledBeforeTheFirstStep(t *testing.T, name string) (mcp.Answer, string) {
	t.Helper()

	c := corpusCase(t, name)
	invocation := c.invocation(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	answered := cli.MCPDispatch(c.process(t, invocation), c.facts(t))(mcp.Call{
		Context: ctx,
		Argv:    []string{runCommandName, "--", procedureOf(t, c)},
	})
	return answered, invocation.fixture.render(t, invocation.fixture.root)
}

// runCommandName is the command every `run` tool call builds a line for, and it
// is spelled here rather than reached out of internal/cli: what a driver states
// is the command line a client's call becomes, which is the thing under test
// rather than a constant to borrow.
const runCommandName = "run"

// procedureOf is the Procedure a `run` case names, read out of the arguments its
// `call` file holds.
func procedureOf(t *testing.T, c goldenCase) string {
	t.Helper()

	var named struct {
		Procedure string `json:"procedure"`
	}
	if err := json.Unmarshal(c.call.Arguments, &named); err != nil {
		t.Fatal(err)
	}
	if named.Procedure == "" {
		t.Fatalf("case %s names no Procedure; it is not a `run` this driver can drive", c.name)
	}
	return named.Procedure
}
