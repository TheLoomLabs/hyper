# A gloss is a notation, and the header is the review's fourth rendering

A **gloss** is a value in a second reading — nothing added, nothing dropped, no judgement. It is
therefore admissible on every surface that renders the value, `FLAGS` included, and it leaves
ADR-0026's one-editorial-surface rule exactly where it stands. And ADR-0026 fixes **three disciplines**
— annotate in place, aggregate what is assembled from elsewhere, editorialise only by citation — never
three surfaces. The review screen has a fourth surface, the **header**, which is now named and given a
rule of its own: it states facts about the artefact as a whole, each read from one supply, and it cites
no line. The Cadence gloss renders there.

Two readings a competent implementer reaches unaided are wrong, and one of them is the reading this
decision was originally posed under.

**That ADR-0026 counts surfaces.** §8 opened with "three renderings sit inside one screen" and then
rendered a fourth — `PROCEDURE │ procedures/retire-preview-envs.yaml   a91f0c2 → working tree`. That
line is not a gutter mark, not a table and not a flag; it carries a Journal-derived fact, the range's
left side being the `procedure_revision` the last non-rehearsal Run recorded (ADR-0057), and it carries
the two named absences. Under the counting reading the corpus has been in violation of its own ADR since
§8 was written, and the Cadence gloss — which §10 makes mandatory wherever a Cadence renders — has
nowhere on the screen to go. Under the discipline reading nothing was ever wrong: the header aggregates
what is assembled from elsewhere, which is a discipline ADR-0026 already admits, and `AUTHORITY` being
"the one table" was a fact about what the corpus held rather than a cardinality rule.

**That a gloss is a claim about a value.** It reads like one: it is text `hyper` writes beside something
an author wrote, on a screen whose whole force is that exactly one surface may editorialise. But
ADR-0026 forbids a surface *introducing* something — a claim with nothing behind it, or a fact assembled
from where the reviewer is not looking. A gloss introduces nothing. `03:00 UTC every Monday` is `0 3 * *
1`, and `≈4.3 runs/month` is arithmetic over the same five fields; a reader who disagrees with a gloss
disagrees with the expression. ADR-0034 already said this for the sibling case in one clause —
"derived arithmetic rather than a claim, so it leaves §8's one-editorial-surface rule where it stands" —
and this generalises it.

The two readings are one decision and not two. The discipline reading is what admits the header; the
notation reading is what admits the gloss into `FLAGS` and into `THE CODE MOVED`. Either alone leaves
the other half of the problem standing.

## Considered options

- **The gloss is a gutter mark.** Not absurd: the gutter carries what `hyper` derived about a line, and
  it already marks non-Step, whole-artefact facts — `envelope ✓` beside `targets:`, and since ADR-0057
  a Repository declaration's version pin and retention policy, where no step table exists to read down.
  Rejected on **supply**: ADR-0057 fixes the gutter's supply as the artefact across the range, and the
  last Journal entry is no part of it. And on geometry: a marker is never truncated, so a gloss in that
  column pushes every line of source sideways by its own length, on a phrase grammar whose length varies
  with how awkwardly an expression glosses.
- **A second table.** Consistent with the discipline reading, and rejected as worse: a table is for a
  fact assembled from more than one artefact, which is what `AUTHORITY` is and what a gloss is not. A
  one-row table for one artefact's own value is a heavier rendering saying a smaller thing.
- **The gloss does not render on a review at all.** This is what the counting reading forces, and it
  contradicts §9 — `review` runs offline "with the Cadence gloss §10 states degrading" — and defeats
  ADR-0005, whose whole argument is that cadence is a blast-radius multiplier belonging on the review
  surface beside every other one.
- **Raw cron in `FLAGS`, glossed only in the header.** Rejected because it fails ADR-0005 on ADR-0005's
  own example. Reviewing `0 0 1 * *` → `*/5 * * * *`, the header glosses the working tree's value and
  the baseline stays cron, so the ≈8,800× is exactly what the reviewer cannot read.

## Consequences

- **The header is named, ruled, and enumerated.** Its members are the artefact's kind, its path, the
  range, and on a Procedure declaring a Cadence the gloss with the last Journal entry beside it. The
  rule admits a further member and the written list keeps the set checkable — `FLAGS`' own pattern —
  but the list lives in §8 beside the rendering rather than in §12, which enumerates vocabularies whose
  names travel as values and has never enumerated a wire key or a rendering.
- **It cites no line, and that clause is what bounds it.** A header fact pointing at a line is an index
  into the gutter, which is `FLAGS`' job under a rule `FLAGS` is held to. Without the clause the header
  is the prose brief this ADR's parent rejected, relocated to the top of the screen.
- **It is a block, one fact per line.** Composing the gloss onto the path-and-range line makes the
  screen's width depend on a cron expression's awkwardness; putting it below the rule makes it read as
  an annotation of `kind: procedure`, which is the one thing the header may not be.
- **The gloss rule is total.** Four surfaces render a Cadence and all four gloss it: a review's header,
  a review's `changed` flag, `THE CODE MOVED`'s `cadence` row, and `project`'s rows. Cron is write-only
  wherever it is read, and a per-surface exemption would be a judgement about which readers deserve the
  number ADR-0005 says matters.
- **The last Journal entry is beside the gloss, not inside it.** ADR-0021's own words. It renders on the
  review header alone; the other three are about a value moving between two revisions or about a file
  just written, and have no artefact-under-review to hang *what stands now* on.
- **Two absences become one.** The range and *last ran* read one entry under one filter, so where it is
  missing the header says so once, in §8's existing wording, and §10's `last ran: unknown (no Store)`
  retires. §8's distinction between *has not run* and *no Store* is untouched — those two are different
  facts, where these were one fact in two notations.
- **The wire gains an `artefact` row carrying the parts.** One row for the block, on the `window` row's
  precedent rather than the per-line `gutter` row's, since a header cites no line and per-line rows
  would invent an anchor. It carries the expression, the phrase and the rate as a number, and never the
  composed string — the `gutter` row's anti-decomposition argument is about one derived fact in one
  cell, and a gloss is several facts with several supplies sharing a line.
- **§12 gains a two-name vocabulary and nothing else.** `not-run` and `no-store`, so the absence the
  page renders as two sentences does not collapse into one missing key on the wire. Everything else this
  decision touches is a rendering, and §12 does not hold those.
- **The predicate gloss is unaffected by any of this, and lands by the same rule.** A gloss renders
  where its supply is: ADR-0034's resolved instant comes from a Run, so it renders on a Run's surfaces
  and never on a review or a `check`, which have none. The Cadence gloss runs the other way — its supply
  is the artefact's own bytes, so it renders wherever a Cadence does, offline included.
