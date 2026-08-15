# A code fact renders whole, in the shape it was written

The Comparison's `THE CODE MOVED` renders a `FROM`/`TO` value whole — the column widens and the row
wraps to as many lines as the two values need — and both the rendering and the comparison follow the
shape of the value at that row rather than the class it sits under. A set of names compares by set
equality and renders as one code-point-sorted ` · `-separated run; a `values:` selector compares by
list equality and renders as authored; a predicate selector compares by set equality and renders one
`field operator operand` line per conjunct; a scalar renders as written; an absent side renders `–`.
The wire carries the artefact's parsed shape, and the review's `FLAGS` uses the same notation.

Before this, §8 rendered exactly one of the nine change classes in full — the `cadence` row — and four
of the remaining eight carry values that are not scalars. Nothing said what went in `FROM` and `TO`
for a selector, a Target set, an Operation set or a list of declared Kinds.

## `changed` is unavailable here, so ADR-0059 cannot be extended

[ADR-0059](0059-a-projected-value-renders-whole-or-renders-changed.md) is the nearest rule and is
scoped to the Comparison's `FIELDS` column. Extending it is what an implementer reaches unaided, and it
fails twice.

It fails **mechanically**: its disqualifications are over 120 characters, a newline, or **nested**, and
a predicate selector is nested by construction. The extension therefore renders every predicate
selector `changed` and every `values:` list of any length whole — a rule that fires on the structure
rather than on the size, in a column where structure is what the values have.

It fails **on meaning**, which is the deeper half. In `FIELDS`, `path: changed` still carries news:
*this field, of the several on this row, moved*. In `THE CODE MOVED` the fact is the row — a row exists
if and only if the fact differs — so `FROM: changed` restates the row's own existence and says nothing
at all. The two columns refuse truncation for the same reason, ADR-0059's: a truncated pair renders two
different values as identical bytes on a row asserting they differ. What they do *instead* differs
because one column has somewhere to fall back to and the other does not.

## Why whole, and not the membership delta

The attractive alternative is to render what each side holds that the other does not — `– → create_key`
where whole renders 25 names. It generalises with no seam: for a scalar the symmetric difference of
`{old}` and `{new}` is exactly today's `old → new`, and an appearance falls out as an empty side.

It was rejected because it is a second rendering `hyper` would have to define and hold closed, and
because nothing else in the tool does it. `FLAGS` renders full before-and-after text everywhere,
including the numeric `bound 3 → 5`; the `AUTHORITY` table renders whole sets; §8 already refuses to
cap the `FIELDS` column's width on the ground that "a wide cell is a Manifest author's projection
choice rendered honestly, and capping it is `hyper` guessing at a number". A 40-Operation Manifest is
that same author's choice. The cost is real and is paid on the widest sets: a reader wanting the edit
rather than the resulting state must difference two cells by eye, which is why both sides of a
set-shaped fact are sorted.

## Why the comparison is by meaning and not by text

§12 refuses a one-member `in:` on the explicit ground that it is a second spelling of `equals` — "one
filter, two ways to write it, which would render as a change in `THE CODE MOVED` with nothing moved".
The format eliminates redundant spellings *so that* comparing what was written is safe. Two
second-spellings survive that programme and reach these classes: reordering a list of bare scalars like
`targets:`, and reordering a predicate list, which is always AND and does not short-circuit. Comparing
the text would produce exactly the outcome the `in:` rule exists to prevent, so each fact compares by
its own equality. A `values:` selector is the one list whose order is meaning — §6 orders an Expansion
by the artefact where the selector is a literal list — so it compares as a list and renders as
authored.

## Why an absence renders as an absence

Three classes give omission a value: an absent `bound:` is unbounded, an absent `over:` is a Step
invoked once, an absent `cadence:` is no recurrence. Rendering those meanings was rejected because the
Comparison is not an editorial surface — §8 fixes `FLAGS` as "the one editorial surface", and
`unbounded` is already a flag name, read off the file where the line is. `step retire · bound  3  –`
already reads as *the Bound was removed*; what that removal means is §5's rule and the review's to
point at. It would also put names-as-values into a table, which is §12's boundary.

For a set-shaped fact the question does not arise twice: an absent key and an empty list are one value,
the empty set, and there is no counterpart here to the Step table's `–` against `0`, which distinguishes
a Disposition that concluded about nothing from one that concluded about zero things.

## Consequences

- **The `cadence` row is not an instance of this rule.** Its value is a scalar; its cell stacks because
  §10 mandates a gloss (ADR-0005, ADR-0063). §8 says so, so that no reader derives a stacking form for
  non-scalars from a row that stacks for an unrelated reason.
- **The `FACT` cell names the authored key**, coordinate-qualified where the key repeats inside one
  artefact — `over`, `step retire · bound`, `credential token`. §12's class names are the grouping and
  reach no screen: a cell naming the class misdescribes a Step's `target:` and a Definition's `destroy:`
  claim. `the digests` names its Provenance member instead, being the one class with no line in any
  artefact.
- **The credential source emits one row per slot**, so the one-line edit §6 named variables explicitly
  to make visible is a row of its own rather than a mapping cell.
- **`FLAGS` and the Comparison share one notation**, which is what §12 already claims of the change
  names: the same fault read off a file on one surface and off two Runs on the other.
- **The catch-all subtracts a fact's own lines and not its block.** A Manifest gaining an Operation
  subtracts the key line the `operations` row reports, leaving the request, projection and Kind beneath
  it in the count, where no classed row reports them.
- **Direction in `FLAGS` is quantified rather than listed** — set inclusion wherever the fact compares
  as a set — which reaches the Manifest's exposed Operation set that §12's list omitted, and gives an
  edit that both gains and loses a member `changed` without a second clause.
- **No closed set moves.** The notation, the `FACT` cell's rule and the shape rule are all §8's, and
  §12 stays at 26 sets.
