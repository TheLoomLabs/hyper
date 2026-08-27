# A cancelled call drains, and the server catches no signal

A client cancelling a `run` call stops the Run the way the first interrupt does: **the Step in flight
finishes, no further Step starts, and the Run closes its own Journal entry `failed`.** The mapping is
the engine's existing predicate reading the call's context, so no second stopping mechanism enters
internal/run. The server installs **no signal watch**, which makes exit codes `130` and `143`
unreachable from that surface.

Progress lands beside it as the `Narrator`'s second implementation: one `notifications/progress` per
Step boundary, sent where the client supplied a progress token and nowhere else, and nothing at all
where the Run names itself.

## The drain was already a predicate, and that is the whole of the decision

§6 states the stop as a question rather than as a mechanism:

> The first interrupt drains: the Step in flight finishes, no further Step starts, and the Run closes
> its own Journal entry `failed`.

The engine asks it as `Request.Interrupted`, `func() bool`, at one place — where the next Step would
start — and its own comment already said what that shape bought:

> it answers rather than blocks, because the one question a Run asks about a signal is *has one
> arrived by now*

A cancelled MCP call is that same question with a different source of an answer. The SDK cancels the
handler's context when the client cancels the request; the surface composes that context's `Err` with
the signal watch's `interrupted`; the engine reads one predicate and cannot tell which of the two
answered. **A surface supplies one of the two and never both** — the server catches no signal and the
terminal has no call to cancel — so the composition is one live source and one that is inert.

What this rejects is the shape where the engine learns about cancellation: a `context.Context` on
`run.Request`, or a second sentinel beside `ErrInterrupted`. Both would put two stopping mechanisms
inside one Run, and the second one is where the day comes that a Step in flight is asked to stop.

**The call's context is never the parent of anything a command performs.** A Step's context is rooted
at `context.Background()` and bounded only by the Manifest's deadline (§3). Rooting it at the call's
would make a cancelled call stop a Step mid-request — *attempted, outcome unknown* — which is the
ambiguity the drain exists to avoid, arriving as a side effect of plumbing rather than as a decision.

## The server catches no signal

`hyper mcp` is one process per client, and the process's signals belong to the client that spawned it.
A Ctrl-C in the client's terminal is not a statement about whichever call happens to be in flight, and
a server that drained a Run on one would be reading a fact about the session as a fact about a call.

So the dispatch behind every tool clears `Process.Notify`, and internal/cli's watch already had a
reading of that: *a nil Notify is a Run nobody can interrupt*. The consequence is that §12's `130` and
`143` are unreachable from this surface, which the envelope mapping had already written down as an
assumption and can now rely on. It costs nothing. A cancelled Run is `failed` on the ordinary code, the
envelope carries the triple, and the Journal carries the account of which Steps ran — which is more
than the number would have said.

## Progress is narration, and narration is the surface's

The engine declares a `Narrator` with exactly two events because §9's narration is two lines: the Run
naming itself before its first Step, and one line per Step boundary. Until now there was one
implementation, and the interface looked like a courtesy. This is the second, and the two differ in
what they *are* rather than in what they say:

- **`Reached` is a `notifications/progress`**, carrying the position, the total and the Step, sent at
  the same boundary §7 writes a Journal entry at. It is narration, so it carries no machine contract
  and no row of its own: what happened is read off the envelope when the call returns.
- **`Began` sends nothing.** On the CLI it exists because the terminal line is not always reached, and
  the Run that dies before it is the one Run whose identity its own output would otherwise never carry
  (ADR-0047). Here the id arrives in the summary line and in `run_id`, and a client that gives up gets
  no delivery at all — so a notification naming the id would be narration with no reader on the one
  path it was invented for.

**A notification is sent where the client supplied a progress token and nowhere else**, which is the
protocol's rule and ADR-0021's at once: without a token there is nothing to correlate a notification
with, and sending one anyway is the server speaking unasked. The surface answers a **nil** Narrator for
such a call rather than one that drops what it is handed, because the engine already has a reading of a
Run nobody is watching and that is the true thing to say.

**A send that fails does nothing.** A notification the transport could not carry is narration that did
not arrive; a Run that stopped because its progress line could not be written would be a Run whose
effects turned on whether anybody was reading.

## Where the two facts live

Both are the **surface's**, so both cross on the values the surface already owns:

- The dispatch takes a `Call` — the argv, the call's context, and where a Step boundary goes —
  rather than a bare argv. The two additions are facts about the *call* that an argv cannot carry, and
  neither is expressed in the SDK's terms, so internal/mcp remains the one package that can name a
  frame (ADR-0088).
- The destination answers `narrator()` and `stopped()`. A destination is where an answer goes, in which
  form, and where narration goes; progress is narration that cannot wait for a buffer read at the end
  of the call, and **a cancelled call is a destination that has gone away**. The CLI's streams never go
  away and answer `false`; the terminal's own stop is a signal to the process and reaches a Run through
  the watch, which is why the composition happens at the one call site rather than inside either value.

## Considered options

- **An asynchronous handle**: `run` answers an id and the caller polls it. Rejected, and this is where
  it is fixed rather than merely unimplemented: it invents a Run that outlives its caller with nothing
  watching it, which is a daemon with extra steps — and §13 has already refused the daemon twice.
- **A `context.Context` on `run.Request`**, read by the engine directly. Rejected: the engine reaching
  a context is the engine gaining a second stopping mechanism, and the first thing that mechanism would
  be used for is bounding a Step in flight.
- **Watch signals in the server too**, so that Ctrl-C in the client's terminal drains a Run. Rejected:
  a signal to the process is not a statement about one call, the drain would fire for calls nobody
  meant to stop, and the exit code it earned would have nowhere to go on a surface with no exit code.
- **Send progress unconditionally**, token or no token. Rejected by ADR-0021, and by the protocol: a
  notification with no token to name is one no client asked for and none can correlate.
- **Send a notification for `Began`**, naming the Run before its first Step. Rejected: the id is in the
  answer and in the summary line, and the one path where that is not enough — a caller that never
  receives the answer — is exactly the path where the notification is not delivered either.

## Consequences

- **§6 states the gate as a predicate** and names the second surface that reads it; §9 states the
  token rule, the silent `Began`, the cancelled call and the absent signal watch, and its exit-code
  paragraph says which two codes the MCP server cannot reach.
- **No golden moved.** The envelope of a `run` call is unchanged, `Notify` was already nil on every
  corpus case, and progress is narration — which this surface drops on the CLI side and sends as a
  notification here. What the corpus gained is nothing, which is the assertion.
- **Two drivers reach past the goldens**, for run_signal_test.go's reason: a progress notification and
  a cancellation are facts about *when*, which no case directory can state. Both drive
  `mcp/run/a-skip-propagates` — the same case TestGolden drives — with one more input supplied, and the
  cancellation is deterministic rather than timed: the Step in flight does not return until the
  handler's own context is done, so no case passes by racing. *Nothing between calls* is held one
  package over, where two calls share one session and everything the client read is checked in the
  order it arrived.
- **Progress no longer lands in the MCP destination's narration buffer**, and could not: a buffer read
  when the call returns is no use to a caller watching a Run that has not returned. Nothing read it —
  the envelope composes that buffer only on the usage-error arm — so what changes is that the buffer
  now holds what it is for.
- **A twenty-minute provision is still not practically runnable from this surface**, which §9 already
  says and this does not change. What it changes is that the twenty minutes are legible while they pass
  and that giving up has a defined answer.
