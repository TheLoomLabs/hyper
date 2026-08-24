# `hyper changes`

§8's Comparison — the window and the header (issue #167), the two Record
tables beneath them (issue #170), and `THE CODE MOVED` with `TOTALS`' last
segment (issue #171). Every case here drives `hyper changes` through `cli.Main`
from its own `argv` and asserts the two streams and the exit code. No case
asserts a branch: `changes` writes nothing, so a `store.golden` here would hold
the seed it was handed.

**All three tables and the `TOTALS` line render on every case, whether or not
there is a row.** An absent block is ambiguous between *nothing to report* and
*the renderer had nothing to say*, so every case here that seeds no Record
renders `0 assets`, `0 observations` and a `TOTALS` of zeros beneath its header
— which is how a Run whose every Step skipped reads (§8).

## `THE CODE MOVED`, and the half a checked-in case can hold

The eight artefact-authored classes and the catch-all's whole count read
**bytes at two revisions**, and a `repo_revision` is a commit — a function
of the tree, the message, the identity and the dates, which no case directory
can name. Every seed here therefore names revisions this fixture never
committed (`1f0a3d78…` and `88bc402f…`, §8's own), so every case renders the
half that needs no bytes:

- `the digests`, the one class read off the **two Journal entries** (ADR-0086),
  whose rows here are `procedure revision`, `definition revision`, `manifest
  digest` and the `—`-subject `repository revision`;
- the catch-all in its **replaced** form, `other lines could not be counted ·
  git diff 1f0a3d7 88bc402`, which keeps the command and carries
  `baseline_absent: not-in-clone` on the wire in place of `count`;
- `TOTALS`' last segment reading *the code moved*, which is the ordering §8
  fixes: a surviving classed row is positive proof where the absence line is
  proof of nothing either way.

The other half — the eight classes over a repository at two real commits, the
count against `git diff` itself, the two suppressions and the *did not move*
and *could not be fully read* forms of the phrase — is driven from
[changes_code_test.go](../../changes_code_test.go) against
[`a-code-window-seeded-at-run-time/`](a-code-window-seeded-at-run-time), which
materialises two code commits and seeds the Journal against both. Both halves
are the specification, and neither can be asserted where the other is.

[`since-names-the-specs-own-window/`](since-names-the-specs-own-window) is
therefore byte-identical to §8's own printed block **except its catch-all
line**, which §8 prints as `2 other lines changed · git diff 1f0a3d7 88bc402`
and which no clone holding neither commit can count.

[`no-baseline-is-a-named-state/`](no-baseline-is-a-named-state) renders the
table's head over `0 facts` and **no catch-all at all**: its subject is the
first Run of its Procedure, so there is no earlier revision for code to have
moved from, no pair for `git diff` to name and nothing for a count to count.

## What a case supplies

Every case that reaches the record carries three inputs, which is
[`runs`](../runs)' and [`records`](../records)' list:

- **`repo-from`** naming [`repo/`](repo), which like `records`' and unlike the
  other two Inspection corpora's is **not** empty of artefacts. The Procedure
  positional resolves against the working tree, so the repository declares two
  Procedures — `retire-preview-envs` and `watch` — and a Definition, a Target
  declaration and a Manifest for them to load against.
- **`git`**, so the fixture is materialised and the Store has a branch to be.
- **`store/`**, whose files are `internal/store`'s own canonical encoding at
  the paths its grammar builds — or a **`store-from`** naming a seed shared
  with another case, which is `repo-from`'s shape one branch over. Two seeds
  here are shared: [`spec-store/`](spec-store), which is §8's own worked
  Comparison, and [`moved-store/`](moved-store), which is the four forms of a
  row that example has none of. A page case and its `--json` twin assert two
  renderings of **one** Store (ADR-0026), which two copies of it would let
  drift apart.

[`version-pin-mismatch/`](version-pin-mismatch) is the one case with a `repo/`
of its own, pinning a version this binary is not, and
[`store-absent/`](store-absent) is the one that seeds no branch.

No case supplies a `now`, a `mint` or a `serve/`. `changes` mints nothing,
dials nothing, and reads the clock only to open the Store's handle at one
instant, so every timestamp behind a golden came out of a seeded file.

## The two Runs the corpus is built on

They are **§8's own worked example**, id for id and instant for instant: the
`cron` Run of `retire-preview-envs` that started `2026-08-04T09:12:03Z` and ran
`1m48s`, and the `igor@thinkpad` Run that started `2026-08-06T11:03:18Z` and
ran `2m31s`. [`a-window-and-its-header/`](a-window-and-its-header) is therefore
byte-identical to the header §8 renders, which is what makes it an assertion
against the specification rather than against this implementation's taste.

## The window, and who may be a side of it

- [`a-window-and-its-header/`](a-window-and-its-header) — the two-line header:
  each Run's id, Trigger, start, outcome, duration and `procedure_revision`.
  Its `-json` twin is the one `window` row that page comes out of.
- [`no-baseline-is-a-named-state/`](no-baseline-is-a-named-state) — *no
  baseline — first Run of `<Procedure>`*, standing in the position the Run's
  facts would have taken.
- [`a-rehearsal-is-neither-side/`](a-rehearsal-is-neither-side) and
  [`an-open-entry-is-neither-side/`](an-open-entry-is-neither-side) — a
  `dry_run` entry between the two Runs and an open entry above them, each
  passed over, so the window is the two Runs on either side of it. They are two
  cases and not one because they are two facts: a rehearsal is **disqualified**
  and an open entry is **not yet nameable** (§8).
- [`a-refused-run-still-stands-as-a-baseline/`](a-refused-run-still-stands-as-a-baseline)
  — an outcome does not disqualify a baseline, a refused Run's completed Steps
  having reached the world like any other's.

## The four accounts, where the header renders one

- [`a-reaped-side-renders-reaped/`](a-reaped-side-renders-reaped) — the word in
  the duration cell, and never a dash: no duration derives, and the entry's
  account being a `closed-by/` file is what says so (§7). Its `-json` twin
  carries the absence as an absent `ended` beside the `closed_by` that explains
  it.
- [`a-reaped-baseline-names-its-own-side/`](a-reaped-baseline-names-its-own-side)
  — the same entry on the other line, where the stated line names `BASELINE`.
  That noun is the one word this page does not share with
  [`show`](../show)'s: `show` holds one entry and says *this entry*, and a
  header holding two names which of its two labels the inference is about.
- [`a-contested-side-is-the-owners/`](a-contested-side-is-the-owners) — both
  accounts stand. The outcome and duration cells are the owner's, unqualified,
  and the contest is a `contested — ` line beneath the header rather than a
  cell value.
- An **open** entry has no case of its own here, having no line: it is neither
  side of a window, which is `an-open-entry-is-neither-side/` above.

## The revision, and the `+`

[`a-dirty-side-renders-the-plus/`](a-dirty-side-renders-the-plus) — a side
whose entry recorded `repo_dirty` renders `procedure rev b0c94f1+`. The bytes
that Run read are nowhere in git, and the marker is what stops the header
asserting otherwise; the `-json` twin carries `repo_dirty: true` and the
revision whole, one fact in the two notations.

## The two ways of naming one window

- [`since-folds-everything-after-the-last-run-before-it/`](since-folds-everything-after-the-last-run-before-it)
  — three Runs and one `--since`: the baseline is the last Run before the
  instant and the subject is the newest, everything between them folded into
  one rendering.
- [`since-includes-the-instant-it-names/`](since-includes-the-instant-it-names)
  — the boundary itself: `--since` naming the exact instant a Run started. The
  bound is a lower bound on `started_at` and includes the instant it names, so
  that Run is **inside** the window and the baseline is the Run before it — a
  caller who copied a `started` off the wire to write the argument gets that
  Run reported rather than skipped over (§8).
- [`since-after-every-run/`](since-after-every-run) — a window nothing happened
  in. It is an answer at `0` and not an error: the name resolved, and fetching
  nothing is not naming nothing (ADR-0060).
- [`since-names-the-specs-own-window/`](since-names-the-specs-own-window) and
  its [`-json` twin](since-names-the-specs-own-window-json) — §8's own printed
  command, over §8's own two Runs and one below them so the instant has a Run
  on either side of it. Its `stdout.golden` is the block §8 prints beside that
  command, byte for byte, its catch-all line aside — which is the one line no
  clone holding neither of §8's two commits can count, and which is asserted in
  its counted form one file over (above). §8's example named
  `2026-08-04T09:12:00Z`, three seconds **before** its own `BASELINE` started,
  where the prose beside it says *take the last Run **before** that instant*;
  the timestamp was corrected in place in `docs/spec/09-review-and-comparison.md`
  and this case is what holds the two together. §8's three `† confirmed`
  instants were corrected the same way and for the same kind of reason:
  `11:02:41`, `11:02:52` and `11:03:09` are **before** the subject Run started
  at `11:03:18`, and a Tombstone's `written_at` is when the Run that wrote it
  confirmed the destruction (§7). They now read `11:04:41`, `11:04:52` and
  `11:05:09`.
- [`between-names-two-runs/`](between-names-two-runs) — the two Runs named
  directly, baseline first, skipping the Run between them.
- [`usage-since-and-between/`](usage-since-and-between) — the two together is a
  usage error at `2` with no `error_code`, the two being different ways of
  naming one window.

## Naming a Procedure, and naming none

- [`naming-no-procedure-compares-across-all/`](naming-no-procedure-compares-across-all)
  — two Procedures, one block each with its own header, in Procedure-name
  code-point order, and **no grand total**: a total would sum across windows
  with different baselines. Naming nothing is the whole-Store mode and not a
  usage error.
- [`a-procedure-with-no-run/`](a-procedure-with-no-run) and
  [`an-empty-store/`](an-empty-store) — the sentence that stands where there is
  no window, naming the Procedure where one was named. Both exit `0`.

## The two Record tables

[`spec-store/`](spec-store) is §8's own worked Comparison seeded whole: three
Runs of `retire-preview-envs`, twelve Observations and eleven Assets, with the
ordinals §8 prints. Twenty-three conclusions and seven rows is what the
identity sets buy — a Record that came back unchanged is in the set and mints
nothing, so it draws no row (ADR-0030, ADR-0058). `TOTALS` counts the rows, so
the tombstone count is a subset of the asset count and is never added to it.

- [`since-names-the-specs-own-window/`](since-names-the-specs-own-window) and
  its `-json` twin are that seed rendered: `created`, `changed` and `destroyed`
  in `YOU DID THIS`, `changed` in `THE WORLD MOVED`, and `ORDINAL` in the forms
  `n → m` and `– → 1`.
- [`the-other-forms-of-a-row/`](the-other-forms-of-a-row) and its
  [`-json` twin](the-other-forms-of-a-row-json), over
  [`moved-store/`](moved-store), are the four that example has none of:
  `appeared` at `– → n`, `vanished` at `n → –`, the `destroyed` row of a series
  a **Tombstone opened** — `– → 1` with an empty `FIELDS`, which is *`hyper`
  ended a thing it never built* — and a value the budget disqualifies. The
  budget's three disqualifications are one class: the `user_data` field is over
  120 characters **and** carries a newline and renders `changed` on the
  two-sided row, and `stdout` renders its bare `path` on the one-sided one,
  where `changed` would be false (ADR-0059). Both go out whole on the wire.
- `cert.hyper.dev` in `moved-store/` stood at both ends and did not move, so it
  draws no row at all: this surface reports what differs and not what was
  looked at.

**A partial set and a Disposition carrying no set are not cases here.** Their
whole content is a row that is *not* drawn, which a golden asserts by being
identical to one that seeds nothing — so they are held in
`internal/compare/records_test.go`, where the two sides can be stated beside the
row they do not draw.

## The narrowing parameters, now that there are rows to narrow

- [`narrowed-by-target/`](narrowed-by-target) — `--target local` is a fact
  about the **identity**, so it is spent before a series is read and empties
  `YOU DID THIS` outright.
- [`narrowed-by-kind/`](narrowed-by-kind) — `--kind asset` is a fact the
  **version** carries, so it is spent over the rows: the table it names keeps
  its rows and the other renders its head over none.
- [`the-limit-cuts-the-tables/`](the-limit-cuts-the-tables) — `--limit 3` cuts
  the rows of the tables and never a `window` row, and the marker names the
  **identity** axis with the two parameters that narrow it. The line on stderr
  is the page's half of the same fact (§9, ADR-0065).

## The failure paths, and the order between them

`changes` resolves its positional against the **working tree**, so a typo is
`2` on a repository with no Store at all — §9's general rule, and the one
[`show`](../show) is the exception to.

- [`usage-an-unknown-procedure/`](usage-an-unknown-procedure) — `2`, no
  `error_code`, naming what was typed and the namespace it resolved against,
  and suggesting no near miss (ADR-0047).
- [`store-absent/`](store-absent) — `77` on a Procedure that resolves, the
  missing branch named rather than an empty answer returned.
- [`version-pin-mismatch/`](version-pin-mismatch) — the gate fires before
  either.

## The usage errors `--between` has of its own

Every one is `2` with no `error_code`, because every one is a name that
resolves to nothing a window can be made of:

- [`usage-between-an-unknown-run/`](usage-between-an-unknown-run) — an id no
  entry carries, pointed at `hyper runs`, which enumerates the namespace.
- [`usage-between-a-rehearsal/`](usage-between-a-rehearsal) and
  [`usage-between-an-open-entry/`](usage-between-an-open-entry) — the two
  standings, refused in their own words, because the remedies differ: one is a
  Run to perform for real and the other is a Run to wait for.
- [`usage-between-two-procedures/`](usage-between-two-procedures) and
  [`usage-between-disagrees-with-the-positional/`](usage-between-disagrees-with-the-positional)
  — a window is over one Procedure, which is the rule that keeps a monitoring
  Run from being compared against a provisioning one.
- [`usage-between-one-run-twice/`](usage-between-one-run-twice) and
  [`usage-between-the-wrong-way-round/`](usage-between-the-wrong-way-round) —
  a window has two ends, and its order is the header's. A pair given the other
  way round is refused rather than quietly reordered or quietly rendered
  backwards, on the surface whose whole job is *this differs from when we last
  looked*.
- [`usage-between-one-value/`](usage-between-one-value) and
  [`usage-between-joined-with-equals/`](usage-between-joined-with-equals) — the
  flag takes two values, so one end of a window and the `=`-joined spelling
  that can only carry one are each told what the flag takes rather than told it
  does not exist.

Beside them, [`usage-two-positionals/`](usage-two-positionals),
[`usage-unknown-flag/`](usage-unknown-flag) and
[`usage-unknown-kind/`](usage-unknown-kind) are the ordinary three;
`--kind`'s closed pair is `asset` and `observation`, and `tombstone` is the
name it is refused for — a Tombstone is a marker inside the Asset table rather
than a class of its own (§8).
