# `hyper records`

The surface whose job is finding a version (issue #166) — where `changes` reads
a change. Every case here drives `hyper records` through `cli.Main` from its own
`argv` and asserts the two streams and the exit code. No case asserts a branch:
`records` writes nothing, so a `store.golden` here would hold the seed it was
handed.

## What a case supplies

Every case that reaches the record carries three inputs, which is
[`runs`](../runs)' list for `runs`' reasons — and one more that no listing
corpus has needed before:

- **`repo-from`** naming [`repo/`](repo), which unlike every other Inspection
  corpus's is **not** empty of artefacts. `records` reads the Definition
  namespace, and for one column only: an Asset whose Definition no longer exists
  is **Orphaned**, and what exists is a fact about the working tree rather than
  about the branch (§7, ADR-0012). The repository declares `preview-dns` and
  `uptime` and does **not** declare `legacy-dns`, which is what makes the
  `legacy-dns` Assets below orphaned.
- **`git`**, so the fixture is materialised and the Store has a branch to be.
- **`store/`**, whose files are `internal/store`'s own canonical encoding at the
  paths its grammar builds.

Two cases carry a `repo/` of their own: [`version-pin-mismatch/`](version-pin-mismatch),
and [`a-definition-that-did-not-load/`](a-definition-that-did-not-load), whose
`definitions/legacy-dns.yaml` will not parse.

No case supplies a `now`, a `mint` or a `serve/`. `records` mints nothing, dials
nothing, and reads the clock only to open the Store's handle at one instant, so
every timestamp behind a golden came out of a seeded file.

## The Store every listing case reads

Five Records across three Targets and three Definitions, in the identity order
`(Target, Definition, name)` the surface renders them in:

| Record | versions | head | what it is here for |
| --- | --- | --- | --- |
| `cloudflare-prod / legacy-dns / preview-8.example.com` | 2 | **Tombstone** | the `TOMBSTONE` marker, and a destroyed Asset whose Definition is gone and is therefore **not** Orphaned |
| `cloudflare-prod / legacy-dns / preview-9.example.com` | 2 | standing | the **Orphaned** Asset |
| `cloudflare-prod / preview-dns / preview-42.example.com` | 3 | standing | three versions, two of them written by Step 2 of their Run |
| `local / uptime / status.hyper.dev` | 2 | standing | the **secret marker**, on one field and then on two |
| `staging / uptime / cert.hyper.dev` | 1 | standing | a series of one, whose Head is ordinal 1 |

One Run — `01991a00-0000…` — wrote the first version of four of the five, at
Steps 1, 1, 2, 3 and 4. That is what makes the `RUN` and `STEP` columns worth
two columns: **the Run and the Step together are the version's identity**, and
two Steps of one Run writing one identity write two paths (§12), so the Run
alone would not name one.

## The listing, and the two orderings

- [`the-heads-listed/`](the-heads-listed) — the Head of each of the five, in
  identity order. Its `-json` twin is the same rows on the wire, where the Run
  ids go out **whole**, the two markers are booleans the page renders as words,
  and `provenance` is carried whole where the page renders its `hyper_version`
  alone.
- [`a-history-is-identity-major/`](a-history-is-identity-major) — every version
  of every Record: the identities in name order, and the versions inside each
  newest-first. The reversal is **whole**, both ordering keys inverting
  together, which is why the first row of each series is exactly the row
  `records` returns without `--history`. That claim is held byte for byte
  against these two cases' JSON goldens by
  [`records_test.go`](../../records_test.go).
- [`ordered-by-identity-and-not-by-the-encoding/`](ordered-by-identity-and-not-by-the-encoding)
  — `zurich.example.com` before `émile.example.com`. Escaping drags every
  escaped character to the left of every unreserved one, so the two sit the
  other way round in the branch's own listing (`%C3%A9mile` before `zurich`):
  the order is over the names anybody wrote and never over the paths they were
  found at (§12, ADR-0044).

## The parameters

- [`narrowed-by-target/`](narrowed-by-target),
  [`narrowed-by-definition/`](narrowed-by-definition) — one column of the
  identity each, matched byte-exact.
- [`one-record-named-whole/`](one-record-named-whole) — all three columns and
  `--history`, which is the whole reason `--since` is legal only there: a caller
  who has already named one Record has no other narrowing left to do.
- [`narrowed-by-since/`](narrowed-by-since) — the window, and the two Records it
  admits nothing of dropping out of the answer entirely rather than standing
  there with no versions.
- [`an-orphaned-asset-is-marked-on-every-row/`](an-orphaned-asset-is-marked-on-every-row)
  — the marker on **every** row the Asset carries, and not once at the moment it
  was orphaned: otherwise a forgotten resource becomes invisible by way of a
  tidy-up commit (§7). The Tombstoned neighbour in the same Definition carries no
  `ORPHANED` marker at all, an Asset that was destroyed having nothing left to be
  out of reach — and it carries `TOMBSTONE` on **both** its rows, that marker
  being *whether its head is a Tombstone* and so the Record's state rather than
  any version's. Those two columns are the only ones on this row that are the
  series' rather than the version's, and they are the two §9 assigns that grain.

## The two cuts

- [`a-cut-listing-names-the-identity-axis/`](a-cut-listing-names-the-identity-axis)
  and its `-json` twin — the marker with `axis: "identity"`, the two counts and
  the hint naming `--target`, `--definition` and `--name`.
- [`the-limit-drops-whole-identities/`](the-limit-drops-whole-identities) —
  `--history --limit 2` returns **four** rows, being two whole series. The limit
  counts identities and not rows: a series cut partway through is a partial
  history wearing a complete one's shape.
- [`a-series-is-cut-at-the-version-cap/`](a-series-is-cut-at-the-version-cap)
  and its `-json` twin — a series of 23 versions cut to the constant. The
  **wire** carries a marker for it, on the `time` axis with version counts and
  `--since` as the hint: a cap that cut a series is a truncated result, and a
  terminal row saying `false` over one would be a truncated result that looks
  complete. §9 gives this command a `--since` precisely *so that the axis a cap
  can cut has a parameter that narrows it* — a window small enough to fit under
  the cap comes back whole, where a larger cap is a bigger answer and not a
  narrower question. The narration reports the counts and never the constant,
  which is an implementation's to pick and nothing's to read back.

## Where the record is

Every case here that renders a page begins with the same line: *the record is
the `hyper-store` branch of this repository — never checked out, and it travels
with a clone* (ADR-0113, issue #233). It is `runs`'s own and stands here for the
same reason — this is the other command whose job is finding something in the
Store — above the table on every page, the empty ones included. The Refusals and
the usage errors carry none of it, having read no record. No `-json` golden
carries it either: the sentence is prose on the page and no row on the wire.

## The empty states, the usage errors and the Refusals

- [`an-empty-store/`](an-empty-store) and [`nothing-matched/`](nothing-matched) —
  the two sentences that stand where a table has no rows, told apart by whether
  the caller narrowed anything at all. Neither names the branch any more, the
  line above them naming it.
- [`usage-since-without-history/`](usage-since-without-history) — exit `2` with
  no `error_code`. Without `--history` the parameter would filter Heads by when
  they last moved, which is a change read on the command whose job is finding a
  version; having it turn `--history` on instead would be the mode ADR-0013
  refused.
- [`usage-a-positional/`](usage-a-positional) — `records` resolves no name, and
  the message names the parameter that takes the one the caller typed.
- [`usage-unknown-flag/`](usage-unknown-flag) — the misspelling, refused before
  a repository is resolved.
- [`store-absent/`](store-absent) — `77`, naming the act that clears it.
- [`version-pin-mismatch/`](version-pin-mismatch) — `77` before a single git
  subprocess runs, the gate firing first everywhere.
- [`a-definition-that-did-not-load/`](a-definition-that-did-not-load) — a
  `definitions/` file that will not parse is in no namespace, so an Asset it
  owns reads as Orphaned. The count is stated on stderr rather than the column
  being quietly wrong, which is the reading `review`'s AUTHORITY table already
  takes over the same absence (§8, ADR-0069).
