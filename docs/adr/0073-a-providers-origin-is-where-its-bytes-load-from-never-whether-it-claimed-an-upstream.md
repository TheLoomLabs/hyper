# A Provider's origin is where its bytes load from, never whether it claimed an upstream

A `providers` row carries `origin`, a closed two-member set — `built-in` and `extension` — and its
criterion is **where the Manifest's bytes load from**: inside the binary, or a tracked file in
`providers/` (§12). Whether those bytes were ever verified against a registry is a **second and
orthogonal fact**, carried by the `origin:` block a Manifest holds or does not, and it is not a third
member of this set.

## The reading an implementer reaches unaided is three

The corpus names three sources, in three places, none of them wrong on its own:

- §7: `origin_digest` "is absent for a built-in Provider and for a locally authored one, neither
  having an upstream to have come from."
- §11: "a Manifest carrying no origin block is a locally authored Provider: checked like any other and
  making no digest claim."
- §11: "a built-in ships inside the binary, and an Extension is a tracked file in `providers/`."

An implementer holding those three sentences and asked to write the enum reaches **`built-in`,
`installed`, `local`**. That is a conclusion arrived at by reasoning from the specification's own text
rather than by failing to read it, which is what makes it worth refusing in writing. Left unrefused, it
would also have been reached asymmetrically: §12 would carry a set whose middle member is a fact about
the past, on a row whose other members are all facts about the present.

## Why the partition stays at two

**The glossary already fixed it.** `CONTEXT.md` defines an Extension as *a Provider authored and
distributed by someone other than `hyper` itself* — not one fetched from a registry. A Manifest an
author typed into `providers/` this morning satisfies that in full. A three-member set would put a wire
value at odds with a settled term, and the term is the one the spec cites everywhere.

**The two facts have different shapes.** *Where do the bytes load from* is a property of this
invocation, decided by the loader, with exactly two answers and no way to grow one: a Manifest is in
the binary or it is in the tree. *Did this Manifest claim an upstream* is a property of the file's
history, decided by whether a block is present, and it is already carried by `origin_digest` in
Provenance and enforced by `origin-digest-mismatch` in `check`. Folding them onto one axis makes a set
that is exhaustive today and breaks the moment either fact moves independently.

**Only one of them is constrained by the other, and weakly.** A built-in can never claim an upstream,
which is why the three-way reading looks like a partition at all. It is a cross with one cell empty,
not a line with three points.

## What it costs, and what pays it

The cost is real: a `providers` row cannot tell an installed Extension from a locally authored one.
The row's `digest` is `manifest_digest`, which §7 states is present for all three — the embedded bytes
for a built-in, the file for either kind of Extension — so nothing on that row distinguishes them.

That is paid at the level where a Manifest's own facts are reported rather than by widening `origin`.
`provider <name>` gains `origin_ref` and `origin_digest`, both written where the block is there and
both absent where it is not (§9). The row already existed to state what a Manifest declares, and the
`origin:` block was the one key §3 defines that no surface rendered — so this is a member the row was
short of, discovered by asking what `origin` does not answer.

## The spelling

`built-in`, not `builtin`. §12's `baseline_absent` already carries `built-in` on §8's wire for the same
fact about the same Manifest — ADR-0068 found the two biconditional with `path`'s absence — and §12
opens by refusing a second set of names for one set of members. One fact reaching two wires reaches
them under one name.

## Consequences

- §12 gains a set and states it in full; §9 cites it where its two sibling closed vocabularies are
  already cited.
- The `providers` return shape changes in one character. It is a pre-implementation spec, so nothing
  is migrated.
- `provider`'s header row grows two members that follow the ordinary absence rule.
- A consumer wanting *was this verified* asks `provider`, one call deeper than the one that lists them.
  That is the same shape as every other Manifest fact: `providers` answers *which*, `provider` answers
  *what does it declare*.
