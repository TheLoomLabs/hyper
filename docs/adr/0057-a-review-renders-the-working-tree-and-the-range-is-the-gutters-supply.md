# A review renders the working tree, and the range is the gutter's supply

A Definition review renders the artefact as it stands in the working tree and renders no removed line.
Where the artefact has a previous revision, the gutter's change column marks the lines the range touched
and the removed text is nowhere on screen. `FLAGS` may nonetheless read a marked line's counterpart at
the revision the header names: the gutter's supply is the artefact **across the range**, not the pixels
of it (§8).

The reading an implementer reaches unaided is a two-sided patch. That is what §8 drew — `- bound: 3`
above `+ bound: 5` — it is what every review tool they have used renders, and §8's own justification for
`WIDENED` said a direction is readable "by comparing the two values **the marked lines** carry", which
only parses if the removed line is one of them. Having built that, the same implementer reads
ADR-0026 — `FLAGS` "may never introduce a claim of its own" — and concludes that a flag may read only
what is rendered, because under a two-sided patch the two readings coincide and nothing forces them
apart.

Both halves are wrong together, and the second is why the first has to be decided rather than left to
the drawing.

A two-sided patch has no line numbering. `review --json`'s flag rows are contractual and a consumer
resolves `cites_line` against a file it has checked out, which is the working tree and never the
baseline. Once removed lines occupy positions, every citation acquires a silent choice of revision, and
the corpus demonstrated the consequence before anyone made the choice: §8's human `FLAGS` block, its
`review --json` rows, and its Refusal excerpt carried **three mutually inconsistent numberings** of the
same file. One-sided rendering does not answer that question, it removes it — a review has one
numbering because there is one file on screen.

A two-sided patch also breaks the object under review. ADR-0026's argument is that the thing a human
reads and the thing they would edit to change it are the same object on screen, since editing that
artefact is the only way past a Refusal. Half the lines of a two-sided patch cannot be edited; they are
not in the file. The gutter stops being an annotation of the artefact and becomes an annotation of a
history that includes it.

What that costs is `WIDENED`'s evidence, and refusing to pay it is incoherent. Direction is not readable
from the working tree alone, so a surface permitted to say *widened* has read the baseline whatever the
rendering does. Withholding it the two numbers while granting it the word keeps the claim and drops the
evidence for the claim — it makes the surface less checkable, not more disciplined. ADR-0026 forbids a
surface assembled from **somewhere the reviewer is not looking**; the baseline is the same artefact at
the revision printed in this screen's header, and the range is what the entire screen is scoped to.
Naming the supply as the artefact across the range keeps ADR-0026's force exactly where it was and
removes an ambiguity that only looked settled because the drawing hid it.

## Considered options

- **A two-sided patch, with `FLAGS` reading only rendered lines.** Rejected on all three arguments
  above: it has no line numbering, it renders text the reviewer cannot edit, and it is the reading the
  corpus already failed to implement consistently in its own examples.
- **One-sided, with the change column widened to carry the delta** — `bound 3 → 5 since a91f0c2` beside
  the line. Rejected: it puts a per-line history down the left margin of every line of the file, and it
  makes the change column a place that classifies rather than marks, which is the distinction the two
  columns exist to hold (§8). The delta belongs in the one row that already carries it.
- **One-sided, with `FLAGS` dropping the values** — `WIDENED · step retire · bound widened since
  a91f0c2`. Rejected as the wrong economy: the direction is the claim and the numbers are its evidence,
  so this withholds precisely the part a reader would use to check the surface.
- **Keeping the two-sided drawing and stating the numbering separately** — declaring that citations use
  the working tree's numbers while the screen shows both sides. Rejected: the reader then has a screen
  whose visible line positions do not match the numbers cited beneath it, which is a worse failure than
  either consistent option and the one most likely to be silently mis-implemented.

## Consequences

- **A review has one line numbering**, the working tree's, counted from one over every line including
  blanks. `FLAGS`, `gutter` rows, the Refusal's caret excerpt and its `EDIT ONE OF` rows share it. §8's
  three numberings were reconciled to it and its examples corrected.
- **A deleted line has no citation of its own** and anchors to the opening line of the nearest enclosing
  structure (§8) — which is the line the flag carrying its subject cites anyway, so a removed `bound:`
  draws `UNBOUNDED` and `WIDENED` on one line and they read as one story.
- **The change column marks and does not classify.** One sigil covers a changed line, a new line and a
  deletion's anchor, because direction is `FLAGS`' text and a three-sigil alphabet would be §12's closed
  vocabulary arriving in the left margin without a name.
- **`FLAGS` gains no new reach.** It may read a marked line at either end of the range and nothing else.
  It still may not read an unmarked line, a second artefact, the Store, or the world — `AUTHORITY`
  remains the one table because it is assembled from two files (ADR-0026).
- **`review --json` is unchanged in shape.** A `gutter` row is one rendered line carrying `marker`
  and/or `changed`; no row type was added for a removed line, because none is rendered.
