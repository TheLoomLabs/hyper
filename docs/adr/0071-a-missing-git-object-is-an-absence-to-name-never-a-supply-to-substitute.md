# A missing git object is an absence to name, never a supply to substitute

Two surfaces read the repository's history. A review's range opens at the revision the last non-rehearsal
Run recorded for the artefact (§8, ADR-0067), and the gutter's supply is that artefact *at that revision*
(ADR-0057). `THE CODE MOVED` computes eight of its nine classes and its whole line count by reading
bytes at two revisions (ADR-0086). Neither stated what it needs, nor what happens when the clone does not hold it. When
the object is absent, `hyper` names the absence — `not-in-clone`, a fourth `baseline_absent` member (§12)
— renders whatever else it could read, and substitutes nothing. It does not fall back, it does not reach
for the object, and it does not refuse. What it does instead is fix the supply where the supply is
`hyper`'s to fix: the projected workflow deepens the runner's checkout before anything runs (§10).

The two readings an implementer reaches unaided are both silently catastrophic, which is what earns this
an ADR at all.

**`HEAD` as the fallback.** It is what every review tool compares against and it needs no Journal. It is
also the exact failure ADR-0067 refused for the anchor, arriving through the back door: an agent that
authors a widened `destroy` Bound *and commits it* leaves both sides of a `HEAD` range equal, so the
review renders clean on the one branch a human is about to approve. Reaching for `HEAD` only when the
real baseline is missing is worse than adopting it outright, because the substitution happens precisely
where the artefact moved — see below — so the fallback fires on every review that had something to say
and never on one that did not.

**`fetch-depth: 0` as the fix.** It is one line, it is the documented answer, and by the action's own
words it is *all history for all branches and tags*. The Store is an orphan branch in the same repository
(ADR-0006), append-only, machine-written, and named in §13 as growing without bound and never reclaimable
from a clone. `fetch-depth: 0` therefore fetches the whole Store on every scheduled Run of every
Procedure, to read a handful of blobs off the code branch. A guarded `git fetch --unshallow` was taken
instead, on the ground that `checkout` leaves `remote.origin.fetch` pinned to the ref it checked out, so
an `--unshallow` inheriting that refspec touches nothing else — one more line, and the one that does not
pay the cost this project has already measured.

**That ground is false, measured on a runner** (#246,
[ADR-0132](0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md)):
`checkout` does not clone, it runs `git init` and `git remote add`, and `git remote add` writes the
wildcard `+refs/heads/*:refs/remotes/origin/*`. The `--unshallow` inherits *that* and fetches every
branch with complete history. The decision below stands — a missing object is still an absence to name —
and so does the line, with a refspec argument on it: `git fetch --unshallow origin "$GITHUB_REF"` names
what it wants instead of asking what the checkout left, and the cost this paragraph declined is declined
again (#258,
[ADR-0134](0134-the-deepen-step-names-one-ref-and-what-deepens-the-code-branch-is-the-clones-own-boundary.md)).
What was wrong was the ground, not the choice standing on it.

## Why this is not a corner

A blob id is content-addressed. `procedure_revision` and `definition_revision` are `git hash-object` over
the bytes that Run read, so in a shallow clone the object is present exactly where the artefact's bytes
have not moved since — the case with nothing to mark — and absent exactly where they have. The absence
and the marks coincide. Left at `fetch-depth: 1`, the range on a runner would be missing on every review
that would have carried a `~`, and `THE CODE MOVED` would lose eight classes and its line count on every
window that had anything in them.

The worst instance is not a missing row but a false one. A **Target declaration** carries no Provenance
member at all — no revision, no digest — so its `env: STAGING_TOKEN` → `env: PROD_TOKEN` edit is visible
only by reading bytes at `repo_revision`. On a shallow clone that produces no classed row, no catch-all
count, and a `TOTALS` line reading *the code did not move*: the credential source class §12 minted,
rendered as its own negation, on the half of the thesis that says nothing changes unseen. That is the
outcome this decision exists to make unreachable, and it is why the depth is fixed at projection time
rather than left to the rendering to apologise for.

## Considered options

- **`HEAD` where the recorded revision is absent.** Rejected above: it substitutes the one baseline that
  is guaranteed to render an edit as no edit, in exactly the case where an edit happened.
- **`fetch-depth: 0`.** Rejected above: it repairs the code branch by fetching the Store branch.
- **`filter: blob:none`, a partial clone.** Rejected: it trades a stated cost for an unstated one. Every
  object read becomes a lazy network fetch, so a review would reach the network without a line of `hyper`
  intending to, and the failure would arrive as latency and timeouts rather than as an absence anyone
  named. It is also what makes the enforcement below necessary rather than optional.
- **`hyper` deepens or fetches the object itself.** Tempting on `changes`, which already fetches the Store
  branch and so would gain no new class of act, and the failure would be `75` rather than `77` by
  ADR-0061's test — past a `75` lies time. Rejected because it makes the command behave differently on a
  laptop that is offline from one that is not, which is the environment-as-authority-axis §5 deleted, and
  because it makes the absence rendering unreachable in testing. Both surfaces stay pure readers of what
  the clone holds.
- **Refusing.** Rejected on both surfaces for different reasons. A review resolves nothing, reaches
  nothing, and has to work in a fresh clone, and §9 fixes exit `1` for the artefact under review failing
  to load and nothing else. A Comparison runs under `if: always()` after a Run that already reached the
  world, and refusing the report of what happened punishes the reader for the shape of a clone.
- **Folding it into `not-run`.** Rejected: `not-run` means *there is no Run to anchor on* and has widened
  four times inside that meaning. Here a Run exists, it is the right one, and it recorded the revision;
  what failed is the clone, and the edit it points at is a fetch rather than a Run.
- **Naming the repairing act in the sentence**, as the other three absences do. Rejected: three causes
  reach this absence and no single act repairs them. A shallow clone wants `--unshallow`, a partial clone
  wants a refetch, and a rewritten history wants nothing that exists. `hyper` cannot tell them apart
  without asking a remote, so any act it named would be a guess, and a sentence promising a repair that
  does not work is the defect §8's named absences exist to refuse.

## Consequences

- **`not-in-clone` is a fourth `baseline_absent` name and ranks last** (§12). The ranking is restated as
  the pipeline it now literally is — no file at all, then nothing to ask, then asked and empty, then
  answered and the bytes absent — because the two comparisons that ordered the first three do not reach a
  fact that is transient *and* answered-and-supplied.
- **It is the first absence that does not suppress the gloss.** §8 held that one absence serves both the
  range and *last ran*; here the Journal entry is present and dated and only its bytes are missing, so
  `last ran: 3 days ago` renders in full and `last_run` goes out on the wire. The rule is narrowed to
  what it always meant — one Journal entry, one lookup — and ADR-0068's statement of it is corrected.
- **It is one name in two positions.** The Comparison's catch-all row is replaced by
  `other lines could not be counted`, keeping `git diff <rev> <rev>` and carrying
  `baseline_absent: not-in-clone` in place of `count`. The command each surface renders is the one its own
  reader can run: a reviewer is standing in the deficient clone, a job summary's reader is usually not.
- **`TOTALS`' last segment gains a third form and loses the right to its negative.** Any classed row
  rendered → *the code moved*; otherwise the absence line rendered → *the code could not be fully read*;
  otherwise → *the code did not move*. The negative may never be asserted over bytes nobody read.
- **The projection carries a deepen step** — guarded on `.git/shallow`, no `|| true`, failing the step —
  placed before the `run` invocation so nothing has reached the world when it fails. It is the file's
  shape rather than a fifth compiled-in constant (§11): it names no version, no host and no third party.
- **A review's *reaches no network* is enforced rather than assumed.** Every object read runs with lazy
  fetching off, and a promisor object that would need fetching renders `not-in-clone` like any other. The
  Store's sync is the one place `hyper` reaches a git remote on purpose and is untouched. The line is
  stated as the line, so an implementer reading git through a library is bound by the same rule.
- **One cause is unrepairable and §13 says so.** A code branch whose history was rewritten leaves an
  object the Store names and nothing produces. The Store outlives the history it points into, which is
  the price of `git hash-object` being the anchor: it buys a range that survives any number of commits,
  and it cannot survive a commit that was unmade.
