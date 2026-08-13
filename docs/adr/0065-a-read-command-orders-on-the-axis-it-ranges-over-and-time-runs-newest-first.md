# A read command orders on the axis it ranges over, and time runs newest-first

The record has two axes. Where a read command ranges over Records it orders by **identity** —
`(Target, Definition, name)`, each by Unicode code point, which is §8's Comparison rule, §7's identity
set rule and §6's Expansion rule reused rather than restated. Where it ranges over Runs or over the
versions of one Record it orders by **time**, and time runs **newest-first**: `runs` on `started_at`
with the `<run-id>` descending behind it, and `records --history` as §7's Head derivation read
backwards, both keys inverting together.

We chose this because §9 gave truncation a marker and no cursor, and truncating an unordered result is
truncating an arbitrary subset — *the first fifty of four thousand Records* is either the answer to a
question or a random sample of one, and nothing decided which. The row stream compounds it: NDJSON goes
out one line at a time behind a single renderer, so a consumer cannot re-sort what it has already
printed. The order is therefore normative rather than incidental.

The rule governs the four commands of §9's *Inspection* and nothing else. **A query chooses an order; a
Run reports one** — a Run's `step` rows arrive as the Steps reach their Dispositions, a Refusal's rows go
in the array's order, and `expanded_to` is a sequence whose order is where a halted `destroy`'s stopping
point is legible (§7). Reversing any of those would reorder events rather than facts.

## Considered options

- **Order by the directory listing.** The Store is files, so a listing is what an implementation holds:
  `records/<target>/<definition>/<name>/` for the identity axis and `journal/<yyyy>/<mm>/<dd>/<run-id>/`
  for time. Rejected on ADR-0044's argument arriving a second time on a different surface — percent-
  encoding drags an escaped character to the left of every unescaped one, so `Über-vm` sorts after
  `zone-a` by name and before it by path. §12 already says the encoding names a file and orders nothing.
- **Oldest-first, in the order the Journal is written.** The reading every log a reader has ever seen
  produces, and the order the files are laid down in. Rejected on the limit: with a modest default and no
  cursor, oldest-first hands back the rows nobody wanted, every time, and the marker's *what was dropped*
  is the whole of what was asked for. The loss is real — a terminal with no pager puts the last row
  nearest the prompt, and under this rule that row is the oldest one.
- **Leave it unstated.** The status quo. Rejected because it makes `--limit` an arbitrary sample and the
  truncation marker's axis a field naming a cut nobody defined. It is also the reading two implementations
  can satisfy while disagreeing, on a tool whose renderings are meant to be byte-identical and diffable.
- **One ordering for everything.** A single rule — identity everywhere, or time everywhere. Rejected
  because neither is total: a Journal entry has no identity to sort on but when it happened, and a listing
  of Records ordered by time is a change feed on the command whose job is finding a version. The two axes
  are one fact stated in two sentences, which is why they are one ADR.
- **A cursor, so the order matters less.** Rejected already by §9 for the reason it is rejected again
  here: pagination is the same context blowout arriving politely, and it would make the order a thing a
  consumer could work around rather than a contract it can rely on.

## Consequences

- **`runs` pays clock skew and nothing else does.** A Journal entry has no second axis, so this is the one
  ordering in the tool derived from two clocks that do not agree — which is exactly what §8 refuses
  `written_at` for, and it can refuse it there because the Comparison has a name axis to fall back on. Two
  adjacent Runs written on two machines can transpose; `hyper` does not detect, warn, or correct, and §13
  names it.
- **The open entry needs no rule.** Ordering on the Run's start rather than its end gives an entry with no
  `outcome.json` a position like any other, because it still carries a `started_at`. Had the ordering been
  on the end, *open* would have needed a defined position among closed entries, and §7's refusal to guess
  whether such a Run is in flight or gone would have had to be re-argued at a rendering.
- **The two orderings over one Store cannot drift.** `records --history` is §7's Head derivation reversed
  whole rather than a second ordering that resembles it, so the rule deciding which version is the Head and
  the rule rendering the series are the same rule read in two directions. The first row of each series is
  the row `records` returns without `--history` at all.
- **Truncation counts identities under `--history`.** A series comes back whole or does not come back: a
  series cut partway through is a partial history wearing a complete one's shape. What bounds one series is
  an unnumbered cap on versions per series, and `records` gains `--since` so the axis that cap cuts has a
  parameter that narrows it — every axis a limit can cut now has one.
- **§12 gains a closed set of two, `identity` and `time`.** The truncation marker's `axis` was already on
  the wire with one value and no enumeration; naming the axes enumerates it.
