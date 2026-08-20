# The Store is append-only

No file in the Store is ever rewritten, by anyone, in any path. Every write creates a new path: a
Record version is a file named by the Run that wrote it, a Journal entry is a directory that gains a
file per Step Disposition and a terminal outcome file when the Run ends. Nothing declares which
version of a Record is current — the **Head** is derived by ordering versions on the `written_at`
each carries.

We chose this because ADR-0006 puts the Store on a git branch with two writers and a
fetch-rebase-retry on contention, and a rewritten file is a rebase conflict waiting to happen. Under
append-only every path carries a unique Run id, so two Runs can never target the same path and almost
every rebase is trivially clean. The one surviving conflict — two reapers closing the same crashed
entry — is a genuine disagreement about what happened, and `hyper` fails rather than picking a side.

## Considered options

- **A `latest` marker file**, as Swamp has. Rejected: it is destructively rewritten on every write,
  which is precisely the contended path. Swamp allocates versions by mkdir-as-mutex and rewrites the
  marker in place; both assume one writer on one filesystem, and neither survives two environments.
- **A monotonic per-Record counter**, with the version number as the file name. Tempting, because a
  path collision would be caught by git rather than silently wrong. Rejected because ADR-0006 already
  ruled it out and the derived head removes the need for one entirely — a counter buys ordering that
  `written_at` already provides.
- **Deriving the Head from the branch's commit order**, which is a genuine total order created by the
  rebase-retry protocol itself, and free. Rejected because the working tree must be self-describing:
  a fresh checkout, or a human reading the branch in a browser, has to be able to see which version
  is current without git plumbing.

## Consequences

- **Finding the Head is a directory listing rather than reading one file**, and finding the previous
  Run of a given Step is a backward scan through date-partitioned Journal directories, stopping at
  the first match. This is the named workload any future local index exists to serve.
- **Ordering trusts two synchronised clocks.** `written_at` is UTC from the writer, ties broken by
  the file name, byte-wise. This is only load-bearing when two writers race on the same Record, which
  the Actions concurrency group and the store lock already make rare. *Amended:* this said the tie is
  broken by the Run id, written before §12 fixed what a version's file name holds. The file name is
  `<run-id>-<nnnn>`, so the two are one rule at two grains — and the finer one is the rule, since two
  Steps of one Run writing one identity write two paths the Run id alone could not order. §7 states
  the file name and is authoritative.
- **Reaping stays append-only.** A Run that closes another Run's crashed entry creates its
  `outcome.json`; it does not edit anything the dead Run wrote.
- **A Tombstone does not end a series.** It is an ordinary version carrying `tombstone: true`, so
  recreating under the same identity writes a further version above it and the Head is alive again —
  which is what makes destroy-then-recreate work under `skip-if-recorded`.
- **Hand-editing the Store is editing evidence**, and there is no legitimate case for it. The branch
  says so in its own `STORE.md`.
