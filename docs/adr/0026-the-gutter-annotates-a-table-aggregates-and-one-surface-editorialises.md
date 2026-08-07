# The gutter annotates, a table aggregates, and one surface editorialises

A Definition review renders the artefact itself, not a table derived from it. Each Step's Kind, its
Target envelope check (ADR-0001), and its Bound are marked in a **gutter** beside the line that makes
the claim, so the thing a human reads and the thing they would edit to change it are the same object
on screen. A table is used only
for a fact that is genuinely assembled from elsewhere — **AUTHORITY**, built from a Definition and a
Target declaration together, is the one fact no single file's gutter could ever show. And exactly one
surface is permitted an editorial voice at all: **FLAGS**, which may repeat a fact the gutter has
already marked, and may never introduce a claim of its own.

Three surfaces for one review reads as unbuilt discipline rather than a decision, and a future reader
maintaining the renderer will be tempted to fold the gutter and `FLAGS` into one view, or to let
`AUTHORITY` grow a second column that restates something the gutter already shows. Neither is
tidying; both reopen the exact failure mode this decision closes.

We chose this because a review surface that can be wrong about what matters is worse than one that
ranks nothing. Three renderings were built to force the choice — a prose brief, a tabular ledger, and
a diff-native patch — and the prose brief's editorial voice, "three things worth checking before you
approve," was rejected on exactly that ground: an editorial claim with nothing behind it is a claim
the surface can simply get wrong. A derived table has the same failure mode one level down — it is
assembled separately from the artefact under review, so it can state something the file does not, or
stay silent about something the file does. Since nothing overrides a Refusal at invocation time, the
only way past one is to edit the exact artefact under review, which makes annotating that artefact in
place, rather than summarising it, the only rendering that cannot drift from what is actually being
approved.

## Considered options

- **A single prose brief.** Rejected: an editorial summary with no citation is a claim the surface can
  be wrong about, and being wrong about what matters is worse than ranking nothing at all.
- **A derived step table, with no gutter.** Rejected: built separately from the file under review, it
  can lie by omission or drift from an edit the reviewer has not yet seen, which is the exact failure
  a review surface exists to prevent.
- **A diff-native patch view alone.** Rejected as the whole answer, not as a component: AUTHORITY is
  assembled from two files at once, and no annotation of either file in isolation has anywhere to put
  it. The decision keeps the patch view for what is in one file and adds a table only for what is not.

## Consequences

- **The derived step table is deleted outright.** The gutter is a table column embedded in the source
  — read down it and the step table falls out for free — but it cannot lie by omission the way a table
  built separately can.
- **A flag with no citation is a defect in the renderer, not a valid rendering.** `FLAGS` has no
  vocabulary the gutter lacks, which turns "the summary said the wrong three things" from a live
  failure mode into a structural impossibility.
- **Asset and Observation changes render as two tables in the Comparison, not one column with two
  values**, and a third table, `THE CODE MOVED`, renders Provenance drift between two Runs as an event
  of its own. Organising the diff by actor rather than by field puts an AI widening a destroy Bound at
  the same visual weight as a server disappearing, which is the weight it deserves.
- **A Refusal is deliberately the most verbose surface in the tool.** Since no bypass exists, its
  caret excerpt and remediation table are the entire path back to a passing review, and verbosity
  there is bought and paid for rather than a lapse in restraint elsewhere.
- **The authoring format inherited a hard constraint from this rendering.** The gutter assumes a Step
  occupies a run of lines a marker can sit beside; a single-expression or non-line-oriented format
  would have collapsed the gutter back into the derived table this decision rejected. The authoring
  format later satisfied it by being a strict, line-oriented YAML subset with no expression language.
- **The terminal and JSON forms share one renderer.** JSON is one row per rendered row with a `type`
  discriminator, so the two surfaces cannot state different things about the same Run; the cost is
  that an NDJSON consumer only knows the outcome at the last line.
