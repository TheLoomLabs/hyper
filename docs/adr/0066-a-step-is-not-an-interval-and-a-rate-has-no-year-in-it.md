# A step is not an interval, and a rate has no year in it

A Cadence gloss is a function of the five cron fields and reads nothing else. Its **phrase** renders
each field in the form it was written in — `*` as *every*, a number as itself, a list as its members, a
range as *from a to b*, a step as *every n* **only where n divides the field's span**, and as the values
it selects everywhere else. Its **rate** counts the expression's matches over one full Gregorian cycle —
146,097 days, 4,800 months, exactly 20,871 weeks — and divides by 4,800. No clock is read at render
time and no calendar year is named.

We chose this because a gloss introduces nothing (ADR-0063), and both halves of the obvious
implementation introduce something. `*/7` on minutes selects 0, 7, 14, 21, 28, 35, 42, 49 and 56, then
waits four minutes: *every 7 minutes* is not a second notation for that, it is a false sentence, printed
on the surface ADR-0005 made mandatory because cadence is a blast-radius multiplier. And a rate counted
over *a year* has no value until somebody says which — 4.33 or 4.42 for `0 3 * * 1`, depending on leap
day and on the weekday the year opens on — while a rate counted over *this* year changes on 1 January
with no edit anywhere, and lets a laptop and a runner render different numbers for the same artefact
across that boundary, which is the environment-as-axis §5 deleted arriving through arithmetic.

The Gregorian cycle is the denominator rather than a denominator: 400 years is a whole number of weeks
as well as of leap cycles, so every leap pattern and every weekday alignment appears in it in its true
proportion. There is nothing left to pick.

## Considered options

- **`every n` for every step form.** The unaided reading, and the reason this is an ADR rather than a
  sentence in §10. It is right for `*/5` and `*/15`, which is what makes it dangerous: it is wrong only
  where the step does not divide, and a reader has no way to tell those cases apart from the phrase,
  which is the one artefact they were given so they would not have to read the cron.
- **`every n` on month and day of week too, where the arithmetic divides.** 12 ÷ 3 is exact and the
  gaps between January, April, July and October are equal, so the rule as stated would admit it.
  Rejected because *every 3 months* does not say **which** three, and unlike minutes the offset is the
  thing being reviewed. Naming the four months is complete and shorter both, so the exclusion costs
  nothing; on day of month the rule never applies at all, the span varying from 28 to 31.
- **A phrasebook of recognised shapes, falling back for the rest.** Recognise `0 3 * * 1` and
  `*/15 * * * *`, and render the expression alone or the rate alone for what is left. Rejected because
  it stops being total exactly where the expression is hardest to read, which is the case the gloss was
  made mandatory for; and because §10 states that no surface renders a Cadence as the expression alone,
  which a fallback contradicts by construction.
- **A truncated phrase, or a short form for a table cell.** The shape ADR-0063 handed forward, having
  given the gloss its own header line *because* this grammar admits expressions that gloss at very
  different lengths. Rejected because truncating drops, and §8's truncation marker names *an axis
  dropped*, which has no meaning inside a phrase. The cell wraps instead.
- **Normalise the fields to the values they select.** Render `*`, `0-59` and `*/1` identically as *every
  minute*. Rejected because it hides the authoring on the surface built to show it: somebody who wrote
  `0-59` where `*` was meant should see that they did. What is normalised instead is order and
  repetition alone, cron attaching no meaning to either.
- **A named reference year, stated in §10.** Reproducible, and honest about being a choice. Rejected
  because the choice is unnecessary: the cycle removes it rather than fixing it, and a named year is a
  constant a later reader would reasonably want re-litigated.
- **A band rather than a number** — *weekly-ish*, *daily-ish*. Rejected on the row it feeds: §8's
  `artefact` row carries `rate` as a JSON number so §12's reviewing agent is not parsing prose to decide
  whether to escalate, and a band whose only form is words has nothing to put in that key.

## Consequences

- **The phrase's length is the expression's own.** Preserving each field's form rather than recomputing a
  canonical one means a field with four list members glosses to four items, and nothing can explode: the
  cross product of minute and hour is never taken beyond the merge that shortens it. It also makes
  ADR-0063's *a reader who disagrees with a gloss can check it* mechanically true — every clause maps
  back to the field that produced it.
- **The two halves of `*/n` land in different places.** `*/5 * * * *` renders *every 5 minutes* and
  `*/7 * * * *` renders nine values past every hour, so two expressions one character apart gloss at very
  different lengths. That is the geometry ADR-0063 gave the gloss its own header line for, and it is why
  the table cells wrap.
- **`≈` becomes informative.** With an exact denominator, a rate is either exact or rounded, so the sign
  can render only where it rounded: `0 0 1 * *` is `1 run/month` and says so. Under a named year every
  rate would carry it, and the one Cadence in common use that really is exact would have been rendered
  as an approximation.
- **The rendered number is the number on the wire.** Rounding once and carrying the result is what keeps
  the page and the row from being two representations of one derived fact — the refusal already made for
  stored durations, for output schemas and for a Head marker. Two significant figures spans the whole
  range the grammar admits, from 0.0202 for `0 0 29 2 *` to 43,829 for `* * * * *`, and never renders a
  Cadence that runs as `0.0`.
- **ADR-0005's own illustration gets a value.** *Roughly 8,600×* was an estimate made before there was a
  denominator; `0 0 1 * *` → `*/5 * * * *` is 8,765.8, and the two rendered rates put ≈8,800× on the
  screen. The number in the argument and the number on the surface the argument is about are now the
  same number.
- **§10 owns the phrase's vocabulary, and §12 says so.** The gloss is a reading of the cron grammar, and
  §12's preamble already keeps that grammar out of the closed-sets chapter as the whole subject of the
  chapter that owns it. The vocabulary joins it there rather than becoming a set of its own.
- **The words the phrase writes are the words the grammar refuses to read.** `Monday` and `January` are
  rejected at load as second spellings of numbers, and emitted by the gloss as the reading those numbers
  have. Abbreviating them would make the phrase look like something that could be pasted back into an
  artefact, which it must never be.
