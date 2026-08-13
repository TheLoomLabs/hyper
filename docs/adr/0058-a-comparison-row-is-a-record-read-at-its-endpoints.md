# A Comparison row is a Record read at its endpoints

The Comparison's Asset and Observation tables render **one row per Record identity**, and that row's
change name is read from what the baseline entry held against what the subject entry holds — never from
the versions between them. Which identities are eligible for a row comes from the identity sets the
Runs' Steps carry (§7) and never from a scan of the Store; which version stands at each end comes from
§7's Head derivation with the entry's own last instant as a cutoff. The two rules are one decision: the
identity sets say *whose* row it is, the endpoints say *what it says*.

The reading an implementer reaches unaided is a version log. Every fact the row needs is on a version
file — its identity, its `run_id`, its `written_at`, its `fields` — so listing the versions written
between two instants and emitting a row for each is the shortest correct-looking path from the Store to
the screen, and it is how a changelog is ordinarily built. It also answers the two hard cases for free:
a Record changed and then destroyed in one Run emits two rows because two versions were written, and a
Record another Procedure created inside the window emits a row because a version exists.

Answering them for free is the problem. Both answers are wrong, and they are wrong in opposite
directions.

**Per-version rows contradict what `ORDINAL` is for.** §8 states that `4 → 7` is "the only place a
Comparison admits that versions exist between its two sides that it is not showing". Under a version log
there is nothing between the sides — every version in the window has its own row — so the column's whole
justification evaporates and it degenerates into `n → n+1` on every line. The chapter also sells the
three-table split as costing "a single chronological scan: there is no one list of everything that
happened, in order", which is a cost only a two-point rendering pays. A version log is that list, minus
the ordering.

**Interval scope contradicts the window rule.** §8 fixes the baseline as the previous Run of the same
Procedure so that "a monitoring Run is never compared against a provisioning one". Scoping rows to the
Store's interval undoes that in one move: everything every other Procedure did between the two instants
lands in this Procedure's tables, and a `read`-only monitoring Procedure grows a populated
`YOU DID THIS` it had no part in. The identity sets are already the mechanism §8 uses for *vanished*,
*appeared* and *nothing moved*; extending them to decide row eligibility outright is the same evidence
doing one job instead of two.

What the endpoint half then costs is that a row may span a version this Procedure did not write, and
that cost is correct rather than tolerated. §8's closing sentence fixes the meaning of every row on the
surface: it "never says *this differs from what we intended*, only *this differs from when we last
looked*". A Record this Procedure concluded about, which another Procedure moved in between, differs
from when this Procedure last looked. Reading `run_id` to decide the change name would make the tables
report authorship, which is a claim about who did what across Procedures — and deriving it means
reasoning about other Procedures' Runs, which is exactly the join the window rule exists to prevent.

## Considered options

- **One row per version written in the window.** Rejected on both arguments above: it strips `ORDINAL`
  of its stated purpose and it is the chronological list §8 names as the accepted cost of the actor
  split.
- **Interval scope — every series in the Store whose head moved between the two instants.** Rejected:
  it puts every Procedure's work in every other Procedure's Comparison and falsifies the sentence that
  makes the window per-Procedure.
- **Endpoints, but with the change name decided by `run_id`** — `created` only where a Run in this
  window wrote the first version, and some other name otherwise. Rejected: it needs the renderer to
  reason about Runs outside the window to name a row inside it, and it means inventing a seventh change
  name for *a version arrived and it was not ours*, which is a distinction the surface has no accountable
  use for.
- **A hybrid: identity-set scope for Observations, interval scope for Assets**, on the ground that
  Assets are `hyper`'s and Observations are the world's. Rejected: it makes `YOU DID THIS` the one table
  that reports work this Procedure did not do, which is the reading its name most obviously forbids.

## Consequences

- **A Record changed and destroyed by one Run is one row**, `destroyed`, spanning both versions, and its
  fields come off the Tombstone — which copied the post-change Head forward, so the row shows the state
  the Record was in when it ended.
- **`ORDINAL` keeps its stated job.** A gap between the two numbers means versions the rendering is not
  showing, whoever wrote them, and that is the only place the Comparison admits them.
- **A row may report a change this Procedure did not cause**, where its Step concluded about an identity
  another Procedure moved. That is the surface saying *this differs from when we last looked*, which is
  what §8 says it says.
- **An endpoint needs an instant, and the instant is the entry's own last one** — `outcome.json`'s
  `ended_at` where the Run wrote it, the last Step file's where a reaper closed it (§7). An open entry
  has neither an outcome nor a settled last instant and is therefore neither baseline nor subject.
- **A Disposition carrying no identity set contributes nothing and asserts nothing**, in both
  directions — §8's partial-set rule at its limit.
- **`TOTALS` counts rows.** The line totals what is on the page above it, so a Record changed and
  destroyed in one window counts once.
