# The fixture delivered one tick in forty, and the gaps moved together across four files

`TheLoomLabs/hyper-runner-fixture` — #246's scratch repository, still standing with its three
projected workflows unedited — was watched across a second span, wider than #246's: from creation at
11:16 UTC on 2026-09-02 through 12:11 UTC on 2026-09-03, just short of twenty-five hours. In that
window `*/5 * * * *` delivered `schedule` events for the first time. **What it delivered was 2.5% of
what it declared**, and the pattern across four independently-scheduled files answers more of #260's
three questions than any one of them was written to ask alone. Issue #260.

## What was measured

Four files carry `*/5 * * * *`: the three `hyper-*.yml` projections (`observe`, `mark-alpha`,
`mark-beta`) and `canary.yml`, the hand-written non-`hyper-*` file #246 left beside them for the same
reason it exists here — separating *did the scheduler fire* from *did the generated file run*. All
four are still `active`, still byte-exact, and still on `main`.

| workflow   | first tick | last tick (of this window) | ticks | declared occurrences | delivered |
|---|---|---|---|---|---|
| `observe` | 09-02 15:08 | 09-03 10:07 | 7 | 228 | 2.63% |
| `mark-beta` | 09-02 15:08 | 09-03 10:40 | 7 | 234 | 2.56% |
| `mark-alpha` | 09-02 15:09 | 09-03 11:16 | 7 | 241 | 2.49% |
| `canary` | 09-02 15:54 | 09-03 12:11 | 7 | 243 | 2.47% |

*Declared occurrences* is the span each workflow was watched over, in minutes, divided by five —
what `*/5 * * * *` promises over that span. *Delivered* is ticks received divided by that number.

**The first tick took 3.86 hours** — the repository was 11:16–15:08 old before GitHub delivered
anything to any of the four files, which is squarely inside what #246 measured (an hour of nothing)
and inside what this ticket's gloss already says (*delayed*). What #246 could not have measured is
what came after: the rate never recovered. Twenty-one hours past the first tick it was still 2.5%,
which is not a new-repository's warm-up — a repository that has been receiving pushes, dispatches and
scheduled ticks continuously for a day is not new by any definition the platform documents, and the
rate at hour 21 is statistically the rate at hour 4.

**The four files' ticks moved together, not independently.** Listing every delivery in order:

    09-02 15:08  observe
    09-02 15:08  mark-beta
    09-02 15:09  mark-alpha
    09-02 15:54  canary
    09-02 18:38  observe
    09-02 18:40  mark-beta
    09-02 18:43  mark-alpha
    09-02 19:16  canary
    09-02 21:16  observe
    09-02 21:19  mark-beta
    09-02 21:21  mark-alpha
    09-02 21:53  canary
    …

Every round is `observe`, `mark-beta`, `mark-alpha` within one to thirteen minutes of each other, then
`canary` trailing by thirty-one minutes to two hours. Four files each declaring their own `*/5 * * * *`
do not cluster like that by chance six times running — a per-file scheduler evaluating four
independent five-minute expressions would interleave them, not deliver them in the same order in the
same few minutes every time. What the pattern shows is a queue with room for about one delivery to
this repository every couple of hours, servicing whatever is due when its turn comes, and `canary`
consistently coming due after the other three within one round. The six gaps between rounds, per file,
ran 1.86 to 5.11 hours — never once landing at, or near, five minutes.

## What this settles, and what it does not

**"Check back" was answered, and the ticket's framing was half right.** Ticks began arriving — the
repository was never permanently starved — but *delayed*, in #10's gloss, undersells what was
measured by two orders of magnitude. A fact that says a Cadence *landing on the hour* is *likeliest to
be delayed* is describing a risk at the scale of one occurrence in sixty being late. What was measured
is 39 occurrences in 40 never arriving at all, for a five-minute expression, for a full day, on a
repository indistinguishable from an established one by any signal `hyper` or a human can read.

**"Compare against a repository with Actions history" is not settled.** `TheLoomLabs` has no
repository whose scheduled-workflow history predates this ticket by more than a day; the account
itself is new. This ADR cannot separate *this account* from *every account on a free plan* from *every
account, full stop* — only a comparison this session had no standing repository to make.

**"Try a less frequent expression" was started and left running.** Two more diagnostic workflows —
`canary-hourly-on.yml` at `0 * * * *` and `canary-hourly-off.yml` at `37 * * * *`, otherwise identical
— were pushed to the fixture at 13:14 UTC on 2026-09-03, after the window this ADR reports on. Whether
an hourly cadence is delivered at anything near 100% is still open; a later ADR should read it once
enough hours have passed for the comparison to mean something, and this one commits to nothing about
hourly cadences beyond what §10 already said.

**The existing hour-boundary fact is left unrevised, and that is a decision and not an oversight.** The
ticket's own complaint names it directly — *"delayed or dropped"* about one minute, read beside a
measurement of an entire hour going unserved, is a gloss describing the wrong risk. This ADR does not
touch its wording anyway, because the twenty-eight delivered ticks this window holds cannot answer the
question the sentence makes: every one of them arrived at an irregular minute — `:07`, `:38`, `:51` —
never on a multiple of five, which means `created_at` is the delivery instant and not the scheduled
one, and the delay between them is unknown and almost certainly not uniform across the twelve
candidates a `*/5` expression offers each hour. A tick recorded at `:38` could be the `:35` candidate
three minutes late or the `:00` candidate thirty-eight minutes late, and this data cannot tell the two
apart. Isolating *does landing on `:00` fare worse than landing off it* from *how much does the
executor deliver at all* needs an experiment that varies only the first, holding the second fixed —
which is exactly what pairs `canary-hourly-on.yml` with `canary-hourly-off.yml`: both hourly, so
neither trips the new within-the-hour fact, and they differ in nothing but which minute they land on.
Until that comparison has enough hours in it to read, rewriting a sentence this measurement cannot
check would be trading one unchecked claim for another.

## The decision this records

**The gloss gets a third fact, derived and not configured**, from the one field the first two already
read: `internal/cadence.Facts` now states *more than one run an hour is more than the executor
delivers — most occurrences will never fire* wherever the minute field selects more than one value —
list, range or step, however written, for `landsOnTheHour`'s own reason (`internal/cadence/facts.go`).
It renders on the same footing as the other two: no `error_code`, no failed `check`, beside the gloss
wherever the gloss renders, in the order **decreasing blast radius** — the branch fact first because it
can silence a Cadence entirely, the new fact second because it silences most of one, the hour-boundary
fact last because it delays at most one occurrence in the pattern this measured.

**Nothing here narrows what `project` writes.** `*/5 * * * *`, once authored, is still projected
verbatim — the fact is a sentence beside the declaration and never an adjustment to it, for the reason
already stated for the hour-boundary fact: a generated expression nobody wrote is a small lie in a file
whose whole value is that it says what was declared.

**Why a threshold of "more than one" rather than a number closer to what was measured.** 2.5% is what
one repository, one day, on one account showed, at one declared density — twelve occurrences an hour —
and this measurement is not positioned to generalise that figure into a rate `hyper` asserts as a
promise about the executor (§14, ADR-0066: no clock, no calendar of record, nothing about the
environment it runs in). But the *trigger* the field reads — more than one value in the minute field —
covers expressions far sparser than `*/5`, down to two values a minute apart from wrapping past the
hour, and the argument for reaching all of them without a second measurement is arithmetic rather than
an extrapolation of the 2.5% figure. Any minute field selecting `k ≥ 2` values divides the hour into
`k` gaps summing to sixty minutes, so the smallest of them is at most `60 ÷ k ≤ 30` — every qualifying
expression, however sparse, declares at least one pair of consecutive occurrences no more than thirty
minutes apart. The narrowest gap this window measured between two delivered ticks, across all four
files, was 1.86 hours — 112 minutes, nearly four times that ceiling. An expression can only fail to
trip the fact by holding every occurrence at least sixty minutes from its neighbours, which is a single
value in the minute field and nothing this field was ever asked to read: the trigger's reach is bounded
by field arithmetic on the same five fields the rest of this package reads, not by how far past `*/5`
the fixture happened to test.

## What was left standing

The fixture repository, its four workflows, and the two new hourly canaries remain live for whoever
reads this next. `hyper-260-fixture/` under `dev/hyper-sessions/` is the clone this session worked
from.
