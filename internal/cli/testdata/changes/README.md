# `hyper changes`

§8's Comparison — the window and the header (issue #167). Every case here
drives `hyper changes` through `cli.Main` from its own `argv` and asserts the
two streams and the exit code. No case asserts a branch: `changes` writes
nothing, so a `store.golden` here would hold the seed it was handed.

**No table stands beneath the header yet.** §8 requires three —
`YOU DID THIS`, `THE WORLD MOVED` and `THE CODE MOVED` — and they arrive in
the two tickets this one blocks:
[#170](https://github.com/TheLoomLabs/hyper/issues/170) for the two Record
tables and [#171](https://github.com/TheLoomLabs/hyper/issues/171) for the code
facts, the catch-all and `TOTALS`. Until they land the surface renders nothing
where they will sit, which is the deferral convention
[`review`](../review)'s own absent range followed for five milestones — so
every `stdout.golden` here ends at the header and the case files gain rows
rather than being rewritten.

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
  the paths its grammar builds.

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
- [`since-after-every-run/`](since-after-every-run) — a window nothing happened
  in. It is an answer at `0` and not an error: the name resolved, and fetching
  nothing is not naming nothing (ADR-0060).
- [`the-specs-own-since-takes-the-run-before-it/`](the-specs-own-since-takes-the-run-before-it)
  — §8's own printed command over §8's own two Runs, and its header is **not**
  the block §8 prints beside it. The example's `--since 2026-08-04T09:12:00Z`
  falls three seconds before its `BASELINE` started; the prose says *take the
  last Run **before** that instant and fold everything after it*, and the code
  follows the rule rather than the illustration. The case is checked in so the
  divergence is asserted rather than sidestepped — `docs/spec/09`'s prose and
  its worked example disagree, and one of them wants correcting.
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
- [`the-narrowing-parameters-are-accepted/`](the-narrowing-parameters-are-accepted)
  — `--target`, `--kind` and `--limit` narrow the rows of the tables above, and
  narrow nothing on a page that has none of them yet. The case is what says
  they are accepted now rather than arriving with the tables.

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
