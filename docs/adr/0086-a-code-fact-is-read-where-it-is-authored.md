# A code fact is read where it is authored

`THE CODE MOVED` reads §12's nine change classes from two supplies, and which supply a class takes is
fixed by where the fact is written rather than by which reading is cheaper. **`the digests` is read off
the two Journal entries**, being the one class with no line in any artefact (§12); **the other eight are
read off the reviewed artefacts at the two revisions the window names**, being the eight that are
authored in an artefact's own lines. There is no class with two supplies and none whose supply depends
on what else could be read.

Before this, §8 said four of the nine read off two Journal entries — `the digests`, the selector, the
Bounds and the Target set — on the ground that a Step file records `selector.declared`, `selector.bound`
and `target` beside the Provenance (§7). Three of those four are now read where they are authored, and
§8, §10, §13 and
[ADR-0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md) are
corrected in place.

## The count was never right

**The Target set class spans both supplies and always did.** §12 states it as *a Procedure's declared
envelope, the Targets a Definition may bind, and the `target:` a Step binds*. The first two appear in no
Step file and in no Provenance member; only the third does. So "four of the nine classes read off two
Journal entries and are unaffected" was true of one fact inside one of the four, and a shallow clone
would have taken a Procedure's `targets:` down while the sentence promised the class survived.

That is what makes this a correction rather than a preference. The 4/5 split could not be implemented as
written, because one of the four cannot be answered from the Journal at all; something had to move, and
what moved is stated here.

## Why the artefacts, and not the entries

**One reader answers both surfaces.** §12 fixes one vocabulary of code facts and §8 makes the review's
`FLAGS` and the Comparison share one notation for it — "the same fault read off a file here and off two
Runs there" ([ADR-0080](0080-a-code-fact-renders-whole-in-the-shape-it-was-written.md)). A review reads
one artefact; a Comparison reading a selector off a Step file instead would be two readers of one
vocabulary, free to disagree about what a conjunct says on two screens a reviewer reads in one sitting.
That is the argument this decision rests on.

**A Step file is written only by a Step that was reached.** *Never reached* is read off a silence and
writes no file at all (§7, §12), a closing write carries no code facts, and a Run that refused at its
second Step wrote nothing for its third. A Bound widened on a Step neither Run reached would therefore
emit no row, on the surface whose whole job is that an AI widening a Bound is a first-class fact
(ADR-0048). The artefacts at the two revisions carry every Step both revisions declared, reached or not.

**The Store's canonical encoding is not the shape §8 puts on the wire.** A `code` row carries the
artefact's own parsed shape for a value that is not a scalar — `{"assets":[{"field":"labels.role",
"equals":"preview"}]}` — and §7 writes a mapping's keys in Unicode code-point order, which would send
that conjunct as `{"equals":…,"field":…}`. It is the narrowest of the three: §12 already says a conjunct
*list*'s order carries no meaning, and a `values:` list survives canonicalisation intact. It is stated
because it is the one place where reading the entry instead of the file changes bytes a consumer sees.

## What it costs, and where the cost is paid

**A revision the clone does not hold takes eight classes down rather than five.** What survives is `the
digests` and the catch-all's replacement line. That is a real loss and it is smaller than it reads: the
surviving class is the one that answers *which Procedure, which Definition, which Manifest and which
commit*, so a window in which any reviewed artefact moved still renders rows saying so, and the
replacement line names the gap the count could not close — which is ADR-0071's rule holding, an
absence named rather than a supply substituted.

It also **strengthens §10's deepen step** rather than weakening it. The projected workflow deepens the
runner's checkout before anything runs precisely because a shallow clone reads no bytes, and the number
of classes that argument is about goes from five to eight.

## Considered options

- **Read the three Step-level facts off the entries, as §8 said.** Rejected on the three grounds above,
  and it would still leave the Target set class split across two supplies — the defect that started this.
- **Read them off the entries only where the clone lacks a revision, and off the artefacts otherwise.**
  Rejected outright. It is a fact whose supply depends on whether a *different* fact could be read, so
  one class has two readings that never coexist and can never be compared against each other; a Bound
  row would mean *what the Run used* on a deficient clone and *what the file said* on a good one, with
  nothing on the page saying which. §7's discipline is that a surface does not decide between two
  accounts of what happened, and this would have it decide silently and by clone state.
- **Read every class off the entries.** Rejected: `the digests` aside, a Target declaration carries no
  Provenance member at all (§12), so its credential source would have no supply anywhere and the class
  §12 minted would be unrenderable.

## Consequences

- **`repo_dirty` and this rule meet without conflict.** §8 already says the classed rows are computed
  between the two committed revisions where a side recorded `repo_dirty`; reading them off the artefacts
  at those revisions is that sentence carried out, where reading a Step file would have reported bytes
  that are nowhere in git beside a count that is not about them.
- **The table ranges over the artefacts the two Runs read**, which is §8's own rule for the Manifests —
  *which Manifests a Run read is the Step files' `provider`* — answered off the same files for the other
  four kinds. What that cannot supply is an artefact only an unreached Step named, whose moved lines fall
  to the catch-all's count instead of drawing a classed row. Nothing is dropped: the word *other* is what
  guarantees the enumeration and the count sum to the whole.
- **A Step present at one revision only pairs against a side with nothing**, and renders `–` there, which
  is the ordinary rule ADR-0080 states rather than an exception for this supply.
- **No closed set moves.** §12's nine classes are unchanged, `the digests` is still stated intensionally,
  and the four `baseline_absent` members are still four.
