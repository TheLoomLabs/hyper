# A flag renders in line order, never in severity order

`FLAGS` rows render in ascending order of the line each cites, with a file-level row — `ENVELOPE`, the
only one today — last. `hyper` fixes no ranking over the vocabulary, and no surface anywhere orders a
flag by how much it should worry a reader. The `flag` rows of `review --json` carry that same order, so
the terminal block and the stream are the same list.

We chose this because the reading an implementer reaches unaided is that a list of flags sorts by
severity. That is what every alert list, linter, and scanner they have built does, and it is a habit
strong enough to survive reading the rule it breaks: they will have read that a flag introduces no claim
of its own, agreed with it, and then sorted the block anyway, because ordering does not feel like
claiming. It is. A total order over the vocabulary asserts that a widened Bound outranks an `opaque`
`destroy`, or the reverse — and nothing in the gutter says either. It is ADR-0026's rejected prose brief
arriving one level down, wearing a sort comparator instead of a sentence: the surface would be stating
something it can be wrong about, on a screen built so that being wrong about what matters is
structurally impossible.

Line order is the only order that adds nothing. It is derived entirely from a fact already on the
screen, it is stable under any change to the vocabulary, and it makes the block traversable against the
gutter directly above it — read the flags top to bottom and your eye walks the file top to bottom, which
is what *index* means. `FLAGS`'s own header says the block indexes the gutter; an order the gutter does
not have is the header ceasing to be true.

The exception is bounded and is not a severity judgement in disguise. A file-level flag cites a line
whose position in the file is arbitrary — `targets:` sits at line 3 because §3 fixes a Procedure's key
order, not because the envelope is the first thing about it — so sorting it into the middle of a
per-Step list would interleave a claim about the whole artefact with claims about individual lines. It
is pinned last because it is the summary of the ones above it, and there is exactly one kind of it.

## Considered options

- **Severity order, with `hyper` fixing the ranking.** Rejected on the argument above: the ranking is a
  claim the gutter does not carry, and severity is the one axis §5 already establishes `hyper` cannot
  measure — authority is statically decidable and magnitude is not, which is why the Bound counts
  Records and no per-Operation risk score exists anywhere in the tool. A severity sort would be that
  score, reintroduced at the rendering layer and never written down as one.
- **The vocabulary's own declared order, then line order within a name** — grouping every `DESTROY`
  together, then every `OPAQUE`, and so on. Rejected as severity order with the ranking made implicit
  rather than removed: whichever name is written first in §12 is the one the block puts at the top, and
  a reader will read that as *the worst first* because there is no other reason for an order to exist.
  It also breaks the traversal: with the flags grouped by name, walking the block no longer walks the
  file, and the reader has to re-scan the gutter for each group.
- **No order at all — emit in whatever order the renderer computes.** Rejected: `review --json`'s rows
  are contractual, and an unstated order is one a consumer will depend on anyway and one a diff of two
  reviews will churn on. An order nobody stated is not the absence of a claim; it is a claim nobody
  checked.
- **Sorting the file-level row first, as a header for the block.** Rejected as the weaker of the two
  placements, not as wrong. `ENVELOPE` is a statement about every Step's `target:`, and those marks are
  what the rows above it cite; a summary that arrives before its evidence is read as a verdict, and one
  that arrives after is read as a check. The second is what this surface is for.

## Consequences

- **`FLAGS` gains no configuration.** There is no `--sort`, no severity filter, and no way to ask for
  only the `destroy` rows. A consumer wanting a subset filters the row stream — `hyper review … --json |
  jq 'select(.flag=="destroy")'` — which is ADR-0013's shape and the same answer every other surface
  gives.
- **A new flag needs no ranking decided for it.** Adding a marker class to the gutter (§8) brings its
  flag with it and nothing has to be re-litigated about where it sits, which is what makes §12's
  intensional rule cheap to hold. Under a severity order every addition would reopen the whole set.
- **The rule reads the same on all five artefacts.** A Definition, a Manifest, a Target declaration and
  a Repository declaration have no `ENVELOPE` row and therefore no exception to apply; their blocks are
  line order and nothing else.
- **Colour stays non-load-bearing (ADR-0015).** A reader who wants severity gets it from the marker
  names themselves — `DESTROY` and `OPAQUE` are the words — and never from position or from red. The
  Actions log and the terminal carry the identical list in the identical order.
