# The record travels in the repository

Records and the Journal are written to a dedicated orphan branch — the **Store** — in the same
repository, and both the laptop and CI push to it. CI runs remain self-contained in the sense that
matters: no resume file, no lock server, no remote database, nothing hidden. They are not, however,
stateless. The record persists across Runs and across environments, in the open, reviewable like
everything else.

We chose this because a CI run that starts with an empty store disarms the execution model. Run-once
refuses on *evidence* — the Journal's Disposition, `never reached → run it` — and `skip-if-recorded`
reads the head version of a Record. With neither present, every run-once Step is permanently
never-reached and every `skip-if-recorded` Step never skips, so a nightly provisioning Procedure
builds a fresh VM every night, and each one is an Asset in a store that evaporates when the runner
does. The Bound does not catch it: magnitude one per Run is the correct magnitude. The change review
has no baseline either, since there is no previous Run to diff against. Persistence is therefore
load-bearing for the safety model, not a convenience for the human.

## Considered options

- **Read-only CI**, with everything effectful reserved for a human at a laptop. This dissolves the
  problem entirely and is the safest thing on the table. Rejected because it would retroactively gut
  the safety model, which was premised on *there being no dangerous moment because CI is unattended*,
  and which built two keys, mandatory Bounds on destroy, and named-Operation granularity to make
  exactly that case safe.
- **An object store behind credentials.** Rejected: the store would need authority in order to record
  authority, and a fresh clone would no longer be self-sufficient.
- **Actions artifacts or cache.** Retention-capped and not fetchable with `git`.
- **The default branch rather than an orphan branch.** Tempting — one `git pull` would give you code
  and record together, and "the code moved" and "the world moved" would literally be one log.
  Rejected on volume: a Journal entry per Run puts roughly 8,800 machine commits a year onto the
  branch humans open pull requests against at an hourly cadence, and 105,000 at the five-minute
  floor.

## Consequences

- **The Store is not a Target.** Writing to it is not an Operation and consumes no Capability — it
  sits below the layer Providers exist at. This keeps the closed Capability set closed: it enumerates
  effects on *the world*, and the Store is the account of the world, not part of it. The runner needs
  `contents: write` on `GITHUB_TOKEN`, which is a fact about the executor, rendered in the projection.
- **One Store, two writers.** An effectful Run syncs before its first effect and **refuses** if it
  cannot — acting effectfully against a record known to be stale is the same blindness in a different
  costume. Read-only Runs may proceed offline and push when able.
- **Durability is per effectful Step.** On a runner nothing is durable until it is pushed, so the
  guarantee that a crash loses at most the in-flight Step has to be re-earned: the open Journal entry
  is pushed at Run start, the Store is pushed after every effectful Step, and reads batch to the end.
  Without this, a runner dying after five creates leaves five VMs with no Assets — invisible to the
  change review, unreachable by Expansion, and rebuilt by the next Run.
- **Only effectful Runs close open Journal entries.** Read-only Runs carry no `concurrency` group, so
  a monitoring cadence would otherwise find a live provisioning Run's open entry and write a false
  `failed` outcome for a Run still in progress. A read-only Run has nothing to protect: it reads and
  never reaps.
- **A version identifier cannot be a monotonic counter**, because two writers would mint the same one.
- **Retention may compact but may never expire evidence.** Pruning the entry that says a Step `ran`
  makes it `never reached` again, which is a bypass, and ADR-0001 says there is no bypass.
- **The record is permanent and `git` history is not editable.** A private repository is a required
  assumption, and redaction cannot be a view applied at query time — it must happen before the Record
  is written, at the boundary where a response becomes a Record.
- **Laptop and CI are not mutually excluded.** The `concurrency` group serialises Actions against
  itself and nothing else, and distinguishing a live open entry from a crashed one would need the
  heartbeat the execution model refused. Accepted as a limit: no record is lost (a non-fast-forward
  push forces fetch-rebase-retry), the overlap is visible in the change review afterwards, and there
  is one user.
