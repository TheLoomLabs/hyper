package cli

import (
	"os"
	"syscall"
)

// The two signals a Run answers to, and what each of them exits with (§6, §12,
// ADR-0015, issue #145).
//
// **The first interrupt drains.** The Step in flight finishes, no further Step
// starts, and the Run closes its own Journal entry `failed` — so most
// cancellations become a stop that is recorded in full rather than an
// ambiguity. The drain itself is the engine's, read where the next Step would
// start (internal/run); what is here is the watch that tells it, and the exit
// code the caught signal earns.
//
// **The second interrupt kills the process**, and it does so because this watch
// gets out of the way the moment the first arrives. Go's signal package
// replaces the default disposition while a channel is registered for a signal
// and restores it when the last one goes, so a Run that has drained releases
// the handler and the next signal lands on the kernel's own answer. What that
// leaves is §6's **open** Journal entry — `run.json` and no account beside it,
// neither an `outcome.json` this Run wrote nor a `closed-by/` file another Run
// wrote. Closing one is the **next effectful Run's**, inside the push that
// sends its own `run.json` (internal/run/reap.go): there is no reaper here, no
// daemon and no heartbeat, and an abandoned entry is noticed by the next Run
// that looks.
//
// `hyper` never claims to have stopped a command it started. A `shell` Step's
// child runs in a process group of its own, which is what makes the drain true
// of one at all — in `hyper`'s group a terminal's interrupt would reach the
// child directly and it would die at once, so the Step in flight would not
// finish. The cost is the other half of that same fact: a second interrupt
// kills `hyper` and leaves the child running with nothing watching it (§3, §6).

// signalWatch is one Run's watch: whether the first interrupt has arrived, and
// the exit code it earned.
//
// The two are read through one channel rather than held as two fields a caller
// may read in either order. Every read of the code goes through a receive on
// drained first, which is what publishes the write beside it: the code is
// written before the close and read only after it, so there is no moment at
// which a caller can see *a signal arrived* and not the signal that arrived.
type signalWatch struct {
	// drained is closed when the first signal arrives, and nil for a Run
	// nobody is watching — a receive on a nil channel blocks, so the
	// non-blocking select below takes its default and answers *no signal*
	// without a branch of its own.
	drained chan struct{}
	// code is what the caught signal exits with, written before drained is
	// closed and read only after.
	code int
}

// watchForTheFirstInterrupt installs the watch and answers it, beside the
// release that takes it down.
//
// A nil Notify is a Run nobody can interrupt: nothing is installed, nothing is
// caught, and the watch answers false forever. That is what the MCP surface and
// every test that supplies no signals get, and it is a value rather than a nil
// pointer so that no caller has to ask which it holds.
//
// The MCP surface's nil is written where its dispatch is, and it is a decision
// rather than an omission: the process's signals belong to the client that
// spawned the server and not to any one call in flight, so exit codes `130` and
// `143` are unreachable from there and what stops a Run is the cancelled call
// instead (§9, §12, ADR-0092, mcp.go).
//
// The watch is taken down on the way out whichever way the Run ended, which is
// what keeps a signal after the last Step from reaching a handler with no Run
// behind it.
func watchForTheFirstInterrupt(notify Notify) (*signalWatch, func()) {
	if notify == nil {
		return &signalWatch{}, func() {}
	}

	arriving, stop := notify(os.Interrupt, syscall.SIGTERM)
	watch := &signalWatch{drained: make(chan struct{})}
	released := make(chan struct{})
	go func() {
		select {
		case caught := <-arriving:
			// The code first, the close second, the release
			// third. The first two are what makes the watch safe
			// to read: a caller that has seen the close has seen
			// the code, and there is no moment at which it can
			// see one and not the other.
			//
			// The release is last, and what that costs is worth
			// stating rather than leaving to be found. Between
			// the receive here and `stop()` returning there is a
			// window — a few instructions wide — in which a
			// second signal lands in the channel's one slot,
			// where nothing reads it, instead of killing the
			// process. It is not closable: releasing first would
			// leave the drain unreadable to a caller that has
			// seen the release, and every `os/signal` design has
			// a window of its own between the kernel delivering
			// and this goroutine being scheduled. What the
			// second interrupt answers to is a terminal, where
			// the two are a keystroke apart rather than a few
			// instructions.
			watch.code = exitForSignal(caught)
			close(watch.drained)
			stop()
		case <-released:
			stop()
		}
	}()
	return watch, func() { close(released) }
}

// interrupted says the first signal has arrived. It is what the engine is
// handed, and it answers rather than blocks: the question a Run asks is *has
// one arrived by now*, asked where the next Step would start.
func (w *signalWatch) interrupted() bool {
	select {
	case <-w.drained:
		return true
	default:
		return false
	}
}

// exit is the code §12 fixes for the signal that arrived, and whether one
// arrived at all. It reads the code through the same receive interrupted does,
// which is what publishes it.
//
// It answers the pair rather than a code alone so that no caller has to read a
// zero as *no signal* — on a Run that is already `failed`, a code of `0` read
// that way would put `failed · exit 0` on the terminal line.
func (w *signalWatch) exit() (int, bool) {
	if !w.interrupted() {
		return 0, false
	}
	return w.code, true
}

// exitForSignal is §12's mapping, and it is the only place in the tool that
// reads a signal for anything: a termination signal is `143` and everything
// else this watch is registered for is the interrupt's `130`. Both are `failed`
// and neither is a Refusal — nothing declined, and the world did not resist
// (§12, ADR-0061).
func exitForSignal(caught os.Signal) int {
	if caught == syscall.SIGTERM {
		return ExitTerminated
	}
	return ExitInterrupted
}
