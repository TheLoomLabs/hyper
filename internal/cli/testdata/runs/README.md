# `hyper runs`

The Journal listed (issue #165) — the surface that enumerates the namespace a
`<run-id>` resolves against, and the one `show` points a caller at when an id
matches nothing. Every case here drives `hyper runs` through `cli.Main` from its
own `argv` and asserts the two streams and the exit code. No case asserts a
branch: `runs` writes nothing, so a `store.golden` here would hold the seed it
was handed.

## What a case supplies

Every case that reaches the record carries three inputs and nothing else, which
is [`show`](../show)'s list for `show`'s reasons:

- **`repo-from`** naming [`repo/`](repo) — a repository declaration and no
  artefact at all. `runs` loads none: it reads the Journal, and the only thing
  in the working tree it looks at is the version pin the gate compares.
  [`version-pin-mismatch/`](version-pin-mismatch) is the one case with a `repo/`
  of its own.
- **`git`**, so the fixture is materialised and the Store has a branch to be.
- **`store/`**, whose files are `internal/store`'s own canonical encoding at the
  paths its grammar builds.

No case supplies a `now`, a `mint` or a `serve/`. `runs` mints nothing, dials
nothing, and reads the clock only to open the Store's handle at one instant, so
every timestamp in a golden came out of a seeded file.

## The Journal every listing case reads

Nine cases share one seeded Journal of six entries, which is one of each account
§7 classifies an entry into and all three of §12's outcomes, across two
Procedures, three Targets, both executors and two versions of `hyper`:

| Run | started | trigger | account | procedure | targets |
| --- | --- | --- | --- | --- | --- |
| `0199206d-4e15…` | 6 Aug 11:03 | `igor@thinkpad` | completed | `retire-preview-envs` | `local`, `staging` |
| `01991ea6-b118…` | 6 Aug 09:41 | `cron` | failed | `retire-preview-envs` | `cloudflare-prod` |
| `01991d70-6a2f…` | 5 Aug 20:00 | `igor@thinkpad` | refused | `retire-preview-envs` | *none* |
| `01991c3a-7d40…` | 5 Aug 14:20 | `igor` | **open** | `publish-preview` | `staging` |
| `019917f2-2c81…` | 4 Aug 09:12 | `cron` | **reaped** | `retire-preview-envs` | `cloudflare-prod`, `local` |
| `019912ab-5e60…` | 3 Aug 08:00 | `igor@thinkpad` | **contested** | `publish-preview` | `local`, `staging` |

Four of those rows are the ones the ticket is about:

- The **open** entry renders no outcome at all and still holds its position in
  the ordering, which is what ordering on the Run's start rather than its end
  buys: an entry with no `outcome.json` still carries a `started_at`.
- The **reaped** entry renders `failed`, which is what §7 fixes a closing write
  records an entry as, and it binds `cloudflare-prod` off that closing write —
  the reaper's reading being a record of the Step the dead Run went quiet on.
- The **contested** entry renders its **owner's** outcome and carries the marker
  beside it. The cell is named for §12's triple, the owner's account is a member
  of it, and a second account is not a fourth value.
- The **refused** entry declined before Step 1, so it holds no Step file and
  bound nothing. Its `TARGETS` cell is empty and its wire member is `[]` and
  never absent: a Run that bound none has an answer rather than no answer, which
  is the one member of this row that departs from the ordinary absence rule.

The `igor` row is the Trigger's third form: a person on the Actions executor,
where no `host` is written and a runner is a machine nobody will look for again.

- [`the-journal-listed/`](the-journal-listed) — all five, newest-first on
  `started_at`. Its `-json` twin is the same rows on the wire, where the ids go
  out **whole** and `contested` is the boolean the page renders as a word.
- [`ties-break-on-the-run-id-descending/`](ties-break-on-the-run-id-descending)
  — two Runs that began in the same millisecond, under a Journal of their own.
  A UUIDv7 is total over the tie, and nothing else here could decide it.

## The four parameters

Four typed, closed parameters and no predicate dialect (ADR-0013). Each has a
case, and each drives the shared Journal:

- [`narrowed-by-procedure/`](narrowed-by-procedure) — the two `publish-preview`
  entries. The Procedure is a parameter here and a positional on `changes`,
  because there it decides which rendering you get.
- [`narrowed-by-target/`](narrowed-by-target) — the two entries that bound
  `cloudflare-prod`, one of them by way of a closing write.
- [`narrowed-by-outcome/`](narrowed-by-outcome) — `completed` selects the two
  entries whose outcome is that, and **never the open one**: `--outcome` filters
  §12's triple, and *open* is a state and not a member of it. The contested
  entry is selected on its owner's outcome, which is the one the entry has.
- [`narrowed-by-since/`](narrowed-by-since) — the bound **includes the instant
  it names**, the fourth row's own `started_at`, which is why the page renders
  an instant in the spelling `--since` reads rather than a friendlier date.

## Truncation

- [`a-cut-listing-names-the-time-axis/`](a-cut-listing-names-the-time-axis) and
  its `-json` twin — `--limit 2` over five entries. The marker names `axis:
  "time"`, what came back, what did not, and `--since` and `--target` as the
  narrowings; the page's counterpart is the line on stderr, in both modes. There
  is no cursor and no pagination, and a truncated result never looks complete.

The other form of that line — a cap nobody named — needs fifty-one entries to
reach and is asserted at [runs_test.go](../../runs_test.go) instead.

## Where the record is

Every case here that renders a page begins with the same line: *the record is
the `hyper-store` branch of this repository — never checked out, and it travels
with a clone* (ADR-0113, issue #233). It is above the table on every page this
command writes, including the two empty ones below, and it is the reason those
two no longer name the branch a second time. The cases that render no page carry
none of it — `store-absent`, `version-pin-mismatch` and the usage errors below
never open the Store, and a command that did not read the record has nothing to
say about where it is. No `-json` golden carries it either: the sentence is prose
on the page and no row on the wire.

## Where there are no rows

An empty table is written as nothing at all, header included, so what stands in
its place is this command's own sentence — and there are two of them, because a
listing that came back empty because the branch holds nothing and one that came
back empty because a filter matched nothing are two different answers:

- [`an-empty-journal/`](an-empty-journal) — a branch that exists and holds no
  entry. It is `0` and a sentence, not a Refusal.
- [`nothing-matched/`](nothing-matched) — the same exit code, and the sentence
  that says a parameter is what excluded everything.

## The failure paths

- [`store-absent/`](store-absent) — `77`. A branch on neither side is a claim
  about the record rather than about this clone, so it Refuses rather than
  answering an empty listing.
- [`version-pin-mismatch/`](version-pin-mismatch) — `77`, before the Store is
  opened and before any row exists.
- [`usage-a-positional/`](usage-a-positional) — `runs` resolves no name in a
  namespace and takes no positional, so a bare Procedure is the parameter
  spelled without its flag, and the message says which flag.
- [`usage-an-outcome-outside-the-triple/`](usage-an-outcome-outside-the-triple)
  — `--outcome open`, which is the value that refusal is really for: a
  parameter accepting the word would relitigate by accident the distinction the
  outcome column exists to hold. `2`, and no `error_code` — a value typed at a
  command line is not a check that declined an artefact.
- [`usage-unknown-flag/`](usage-unknown-flag) — `--procedures`, one letter out.
