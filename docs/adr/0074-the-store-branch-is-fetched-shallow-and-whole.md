# The Store branch is fetched shallow and whole

The Store is an orphan branch of the same repository that `hyper` fetches and pushes itself, that sync
being `hyper`'s own work rather than a step of the projected workflow (ADR-0006, §7, §10). No section
said how much of it `hyper` takes. It takes the tip and no history: where the branch is absent from the
clone, `hyper` creates it with a depth-1 fetch of that one ref, and where the branch is present it
fetches incrementally and names no depth at all. It never deepens the Store and never shortens it. And
it never filters: no blob filter, no tree filter, no partial clone. **The Store's history is never read
and its content always is**, and those are the two halves of one thought rather than two decisions.

The history half is already decided everywhere except in a sentence about fetching. ADR-0011 rejected
deriving the Head from the branch's commit order — a genuine total order, created by the rebase-retry
protocol itself, and free — because the working tree must be self-describing. §7 states that files are
authoritative: finding a Head is a directory listing, finding a Step's previous Run is a backward scan
through date partitions. Provenance carries `procedure_revision`, `definition_revision` and
`repo_revision` and no `store_revision`, so every git object `hyper` reads *at a revision* belongs to
the **code** branch — the branch ADR-0071 deepens — and the Store supplies only its tip. Append-only is
what makes a year-old Run a tip read.

The content half runs the other way and for the same reason. A version's `written_at` is inside the
file, so ordering a series means opening every version of it; a partial clone turns each of those into
a lazy fetch, mid-Run, and *a read-only Run proceeds offline* (§7) becomes false wherever the network
is. ADR-0071 already made *reaches no network* an enforced rule rather than an assumption, on a review;
a blob-filtered Store would have `hyper` manufacturing the condition it enforces against.

## The two readings an implementer reaches unaided

**`git fetch origin hyper-store`, the whole branch.** It is what git does when nobody says otherwise,
and §13's own text primes the reader for it by naming history as the cost of the record. On a runner it
is the cost ADR-0071 declined `fetch-depth: 0` to avoid, arriving one step later and by a different
door: `actions/checkout` takes one ref, so the Store is absent and `hyper` fetches it from scratch on
**every scheduled occurrence of every Procedure**, and at full depth that is the whole history each
time.

**A blob filter as the economy.** It is the obvious optimisation for a branch this large — a Run reads a
tiny subset of what the branch holds — and it is the one that costs the guarantee rather than the bytes.
Its failure arrives as latency and timeouts on a laptop with no connection, in a tool whose read path is
specified to have none.

## Why the depth is decided by the branch's absence and not by the repository's state

A depth-1 fetch into a clone that already holds the branch in full *truncates* it, and `git log` on the
Store is Compaction's own account of what it removed (§7, §8, §9, ADR-0049). Matching the repository's
own shallowness fails on the exact path that matters: ADR-0071's deepen step runs before `hyper` and
leaves the runner un-shallow, so `hyper` would fetch the Store in full precisely where it must not. What
survives both is a rule about creation rather than about state — **`hyper` never changes the depth of a
branch it did not create** — which is ADR-0071's *never deepens on its own behalf* read from the other
end, and which decides the depth exactly once, at the moment there is nothing to preserve.

## Why the rebase is safe below the boundary

§7's push retry fetches, rebases and retries, which is a history operation on a shallow branch and the
first thing that looks like it should not work. It works, for a reason nobody had written down: the
Store is append-only and is never force-pushed, so every fetch lands on a **descendant** of the boundary
the previous fetch set. A local unpushed commit is rooted at the tip `hyper` last saw, so the merge base
of it and the new remote tip is at or above that boundary and is always present. The one conflict §7
keeps — two Runs closing the same open entry — is a conflict about content at the tip and is unmoved by
depth. The push is a fast-forward whose parent the remote already holds, so the boundary never reaches
the wire.

## Considered options

- **Always shallow.** Rejected: it truncates the laptop clone that already had the branch whole, which
  is the ordinary laptop, and takes `git log` from the one surface that is specified to be Compaction's
  account.
- **Always full.** Rejected above: it reinstates on every scheduled Run the cost ADR-0071 rejected
  `fetch-depth: 0` for.
- **Match the repository's shallowness.** Rejected above: the deepen step makes the runner un-shallow
  before `hyper` runs, so this reads *fetch the Store in full* on the only clone where the choice bites.
- **A blob or tree filter.** Rejected above: it converts an offline guarantee into a network dependency
  at the granularity of a file nobody named.
- **`hyper` deepens the Store where something turns out to need history.** Rejected because nothing does
  — see the consequence below — and because ADR-0071 already declined deepening on `hyper`'s own behalf
  for a reason that is unchanged here: it makes a command behave differently on a laptop that is offline
  from one that is not.

## Consequences

- **There is no Store-side `not-in-clone` and no new `error_code`.** Every Store read is the tip tree or
  a descendant of the boundary, so no absence exists to name and §12's sets do not move. A fetch that
  cannot complete is already the contention code `75` (§7), and ADR-0061's test has nothing new to fire
  on. This is the fourth shape of a closed set staying closed by not being asked.
- **`hyper` names the Store ref explicitly.** ADR-0071 records that `checkout` leaves
  `remote.origin.fetch` pinned to the single ref it took; a fetch relying on the configured refspec
  therefore reaches nothing on a runner. The same trap, at a second site, and written down rather than
  relied on.
- **On a runner the Store fetch re-creates `.git/shallow` after the deepen step has cleared it**, with a
  boundary naming the Store branch alone. Nothing on the code branch is truncated, and no later step
  reads that file.
- **A `--single-branch` laptop clone gains a shallow Store**, its owner having opted out of the branch
  already. This is the one clone where Compaction and shallowness can meet, and there `git log` is short
  by exactly the history that was never fetched. On every path `hyper` projects they cannot meet at all:
  Compaction is an explicit command that never runs on a Cadence, and the shallow Store is the runner's.
- **Compaction reclaims from a shallow fetch what it cannot reclaim from a clone**, which corrects the
  reading §7's *it reclaims nothing from a clone* invites. What it does not reclaim, at any depth, is the
  Journal: §7 lets it remove interior Observation versions only, so the term that grows with the
  **Cadence** rather than with the world is the term nothing reclaims, and at a five-minute recurrence it
  is the dominant one. §13 carries the limit as two curves with two payers rather than one.
