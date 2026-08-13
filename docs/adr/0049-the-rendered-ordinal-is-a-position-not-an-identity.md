# The rendered ordinal is a position, not an identity

The number §8 renders beside a Record — `4 → 5`, `– → 1`, `22 → 23` — is a version's ordinal position in
the `written_at` ordering of what the Store currently holds, derived at read time from the same directory
listing that derives the Head. Nothing stores it, no version carries it, no file is named by it, and no
surface accepts one as input: a version is named by its Run. It is unstable by construction, and §7 and §8
say so where a reader stands rather than leaving it to be discovered.

We chose this because the reading an implementer reaches unaided is that the number identifies the
version, and the corpus depicted it: the build rehearsal rendered `4 → 6` in a table and `from_version: 4`
as a typed integer on the wire, out of fourteen sections, twenty-eight ADRs and a `CONTEXT.md` containing
no counter anywhere — so it invented one, which is the only thing that number could have come from.
ADR-0011 rejected a monotonic per-Record counter at the Store layer, on the ground that a counter is not
mintable by two writers alone and `written_at` already supplies the ordering, and §7 says so in as many
words: *a version identifier is mintable by either writer alone and never a counter*. That sentence stops
one word short of the rendering, and the rendering is where the counter came back. This is ADR-0011's
rejection arriving at the layer that needed it.

What the ordinal earns its column with is **adjacency**, not magnitude. A Definition is reachable from more
than one Procedure, and `--since` folds several Runs into one rendering, so `4 → 7` is the only place a
Comparison admits that versions exist between its two sides that it is not showing. It is not a count of
times the Record was checked: a version is minted only where the bytes moved (§7), so a Record read hourly
for a month and never changing sits at `1`.

What makes the instability affordable rather than merely admitted is that nothing consumes an ordinal. No
artefact references a version, `--between` takes Run ids, a selector reads the head, and the Store path is
`<run-id>-<nnnn>`. A renumber therefore costs a stale rendering and never a wrong answer — which holds only
for as long as it stays true, and is why it is stated as a rule rather than observed of today's flag set.

## Considered options

- **Render the version's identity instead of a number** — `01991c3a…-0002 → 01991ea6…-0001`, which is the
  filename, is stable forever, and under a `--since` fold names which Run in the fold last touched the
  Record. Rejected: it retires adjacency, the one fact the column carries that no other column does, in
  exchange for an identity §7 already states — and `records` now carries that identity anyway, at the one
  surface whose job is finding the file rather than reading a change.
- **A gap marker, or a count of what the window hides** — `4 ⋯ 7`, or `4 → 7 (2 hidden)`, making adjacency
  explicit rather than arithmetic. Rejected twice over. Both are damaged by exactly the thing the ordinal
  is, so neither buys a guarantee the bare number lacks: after a Compaction the marker vanishes and the
  count is wrong, and a gap that was real closes silently either way. And both would be **sound in the
  Asset table and unsound in the Observation table beside it**, Compaction never removing an Asset version
  — two guarantees under one column head, which a reader has to hold in their head to read a table
  correctly.
- **Naming Compaction as the whole instability**, as the question arrived. Rejected as stating less than is
  true: a read-only Run proceeds offline and pushes when it can (§7), so a laptop's Observations slot in
  beneath versions a runner already pushed and every ordinal above them moves, with no Compaction anywhere.
  It is the worse of the two — Compaction is an explicit command with a `git log` behind it, where this
  announces nothing at all — and a spec naming only Compaction would leave an implementer believing a
  freshly-pushed number is settled.
- **Two rules, one per table** — the Asset ordinal stable, the Observation ordinal not, which is what
  Compaction alone would license. Rejected: it is false under a late push, so it would be a guarantee that
  does not hold.
- **Leaving the column called `VERSION`.** Rejected: over `4 → 5` that head asserts *version 4 became
  version 5*, which is the identity reading, and it is the word that carried the rehearsal into inventing a
  counter. `SEQ` reinstates the same reading under a shorter spelling. `POSITION` was chosen and then
  withdrawn: this format has already spent ADR-0031 on what a position *in a request* is, and uses the word
  for template holes, credential slots, positional arguments and the Step's own position in the Run's
  written order — a `records` row reading `{ ordinal, run_id, step }` is legible where `{ position, run_id,
  step }` puts two senses of one word two keys apart. `ORDINAL` occurs nowhere else in the corpus and is
  `VERSION`'s width exactly, so the table's geometry does not move.

## Consequences

- **Two renderings of one Store can disagree, and neither is wrong.** A Comparison read before a laptop's
  push and one read after report different ordinals for the same two versions. This is stated at the
  rendering, at the Head it is derived from, and at the Compaction that moves it.
- **An ordinal is never input.** No flag, no positional, no artefact reference takes one, and none may be
  added. The rule has a positive form that is already true everywhere else — a version is named by its Run
  — which is what the Store path grammar does, what `--between` does, and what §7 already says of a version
  identifier.
- **`records` carries what an ordinal cannot.** Its rows gain the version's `run_id` and `step`, mirroring
  the keys the version file itself holds, because that is the one surface whose purpose is fetching rather
  than reading a change; two Steps of one Run writing one identity write two paths (§12), so the Run alone
  would not name a version. ADR-0047 applies unchanged — abbreviated in the table, whole under `--json` —
  and is unstrained here, an ordinal being nothing a human retypes and a Run id on this row being read
  rather than typed.
- **No closed set moves.** §12 enumerates no wire keys, so the rename reaches four rendering sites and
  nothing enumerated. No `error_code` names a stale ordinal: a rendering read once has no staleness to
  report, and there is no operation that could fail on one.
- **Widening this later is possible; narrowing it is not.** Nothing stops a future `hyper` from minting a
  stable version identity if one is ever earned. Withdrawing an ordinal that scripts have started to
  address versions by could not be done at all — which is the same asymmetry ADR-0047 closes on.
