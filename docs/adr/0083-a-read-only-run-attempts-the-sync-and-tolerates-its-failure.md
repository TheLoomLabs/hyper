# A read-only Run attempts the sync and tolerates its failure

A Run whose every Step is `read` reaches the remote at Run start like any other
Run, and **a sync that could not complete does not stop it**. It proceeds
against whatever branch the clone holds; it Refuses `store-absent` only where no
branch is in hand *after* the attempt; and it is never `75` for a sync it could
not complete, that code being the effectful Run's.

## The problem this decides

§7 states the sync in one sentence per Run and the second one is a fragment:

> An effectful Run syncs the Store once, at Run start, and that sync **is** the
> push of its open Journal entry […]. A read-only Run proceeds offline and
> pushes when it can.

*Proceeds offline* has two readings and the section that states it never picks
one. It can mean **a read-only Run does not reach the remote at all** — offline
is where it works, and syncing is the effectful Run's act. Or it can mean **a
read-only Run survives a reach that failed** — it goes when it can and carries
on when it cannot.

The first reading is the one the words most naturally carry, and it breaks the
tool on its own deployment. §7 spends a paragraph on the runner's clone:

> Where the branch is absent from the clone — which is every runner,
> `actions/checkout` taking one ref (§10) — `hyper` creates it with a depth-1
> fetch of that one ref.

A runner's clone holds one ref and the Store is not it. A read-only Run that
never fetches therefore finds no branch, and §7 is unambiguous about what
follows: *A Run that cannot find it Refuses (`store-absent`, §12) rather than
proceeding against an empty record.* So under the first reading **every
scheduled monitoring Run Refuses at `77`, forever**, on the executor §10 exists
to project onto — and the remedy the code names, `hyper store init`, would not
help, the branch being already there on `origin` and merely unfetched.

That cannot be the intent, and one sentence three paragraphs earlier says why in
as many words:

> Offline-tolerant means the branch exists and is unreachable now; it never
> means it has never existed.

A Run that never reached the remote cannot tell those two apart. It holds no
branch either way, which is exactly the confusion that sentence exists to
forbid. Tolerance is a property of a reach that was made and failed; declining
to reach is not tolerance of anything.

The gap is small and it is load-bearing. Left unstated, an implementation picks
whichever reading its author assumed, and the reading nobody reviewed decides
whether a five-minute cadence on a runner works at all.

## The decision

**A read-only Run attempts the sync and tolerates its failure.**

- It **attempts** it, at Run start, through the same call and the same depth
  rules an effectful Run's sync uses (ADR-0074). Nothing about the fetch differs
  by Kind.
- It **tolerates** the failure: it proceeds against whatever branch the clone
  holds, and its pushes batch to its end and go out when they can.
- It **Refuses `store-absent` only where no branch is in hand after the
  attempt** — which is a question about this clone, asked once the reach has
  been made and not instead of making it.
- It is **never `75` for a sync it could not complete.** That code is the
  effectful Run's, and the asymmetry is the point of the next section.
- It **says so**, once, on stderr, before its first Step: the condition and what
  it did about it, in words that name no remote and no URL. It is narration and
  not a Refusal — no `error_code`, no row, and stdout carries none of it.

### Why the two Runs part here at all

An effectful Run's sync **is** the push of its own open Journal entry (§7). Its
failure therefore means the Run cannot record what it is about to do, and the
Run is about to reach the world. Stopping is the only honest answer: `75`, and
the world untouched.

A read-only Run's sync is a fetch and nothing else. Its own writes are
Observations, and **nothing a `read` Step does is gated on the record**:
`skip-if-recorded` is `mutate`-only and run-once is effectful-only (§12,
ADR-0037), so a stale Store cannot disarm a test, cannot suppress a call, and
cannot cause an effect to happen twice. The failure mode of reading a stale
Store is a redundant Record version — one whose content matches a head this Run
could not see — and §7's Head derivation is a directory listing over an
append-only branch, so that version stands beside the one it duplicates rather
than contending with it.

That is the whole asymmetry: the effectful Run's sync gates an effect, and the
read-only Run's gates a comparison. One is worth stopping for.

## Considered options

- **The literal reading: a read-only Run never reaches the remote.** The option
  the words most nearly say, and the reason this ADR exists. Rejected: it
  Refuses `store-absent` at `77` on every scheduled Run on every runner, which
  is the executor §10 projects a Cadence onto, and it makes *offline-tolerant
  means the branch exists and is unreachable now* unsatisfiable — a Run that
  never reaches cannot be in a position to know either.
- **Symmetry: a read-only Run syncs, and a failed sync is `75` like the
  effectful one's.** Rejected: it empties the sentence it was meant to honour.
  The whole content of *proceeds offline and pushes when it can* is that a
  `read` Run survives an unreachable network, and this converts a network
  outage into a monitoring cadence that reports failure for as long as the
  outage lasts — which is the one moment an operator most wants a reading.
- **Let a read-only Run create the branch where its sync brought none.** It
  removes the Refusal outright and is rejected outright. The branch is created
  by an explicit act and never by a Run, read-only Runs included (§7,
  ADR-0006), because a fetch that failed mid-flight and a branch that never
  existed look identical from the inside — and reading either as *there is
  nothing recorded* is the reading ADR-0006 refuses.
- **A flag: `--offline`, or `--no-sync`.** Rejected: it puts the decision on
  whoever typed the command, and the occasion this is about is the one nobody
  types. A scheduled Run's argv is written once, in a projection (§10), months
  before the network goes down.
- **Tell the causes apart: tolerate a remote that could not be reached, fail on
  anything else.** Rejected: the distinction is not available. `git fetch`
  reports a refused connection, a rejected credential, a hook that declined and
  a ref that is not there through one exit status and prose meant for a human,
  and a tool sorting them on that would be reading a message rather than asking
  a question. They also clear the same way — by trying again — which is what
  `75` and `77` sort on (ADR-0061).

## Consequences

- **`store-absent` moves from the sync's answer to the open that follows it.**
  It is what this clone holds once the reach has been made, and never what the
  reach itself answered. The Refusal, its code and its remedy are unchanged.
- **`75` is now precisely the effectful Run's for the sync at Run start.** §12's
  entry gains that scope; the lock and the exhausted push are unchanged and
  belong to both Runs.
- **A read-only Run can record against a Store it could not refresh, and says
  so.** The cost is a redundant Record version, bounded by the Cadence and
  reclaimed by retention like any other (§7). It is not a repeated effect and
  cannot become one, on the reasoning above.
- **The stderr line names no remote.** The failure is tolerated, so what an
  operator needs from it is *this Run may be reading a stale Store*, not a
  diagnosis of the network — and a fetch's error names a remote by URL, which is
  a fact about the machine. An operator who wants git's words runs git.
- **`store-absent` after a failed sync says both things.** The clone holds no
  branch *and* the reach that would have brought one did not land, which is two
  lines rather than one and is the difference between *there has never been a
  Store* and *this machine cannot see it*.
- **No closed set moves.** No `error_code`, no exit code, no outcome, no
  Disposition. The decision picks one of two readings of a sentence already in
  the specification and adds nothing to any of them.
