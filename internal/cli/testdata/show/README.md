# `hyper show`

One Journal entry read back whole (issue #163), and the first command a person
can type that reads the record at all. Every case here drives `hyper show`
through `cli.Main` from its own `argv` and asserts the two streams and the exit
code. No case asserts a branch: `show` writes nothing, so a `store.golden` here
would hold the seed it was handed.

## What a case supplies

Every case that reaches the record carries three inputs and nothing else:

- **`repo-from`** naming [`repo/`](repo) — a repository declaration and no
  artefact at all. `show` loads none: it reads the Journal, and the only thing
  in the working tree it looks at is the version pin the gate compares.
  [`version-pin-mismatch/`](version-pin-mismatch) is the one case with a `repo/`
  of its own, pinning a version this binary is not.
- **`git`**, so the fixture is materialised and the Store has a branch to be.
- **`store/`**, whose files are `internal/store`'s own canonical encoding at the
  paths its grammar builds. That is what makes a golden an assertion rather than
  a transcript — a Journal file the reader could not decode fails the run rather
  than surviving it, which
  [`a-journal-file-this-binary-cannot-read/`](a-journal-file-this-binary-cannot-read)
  is the case for.

No case supplies a `now`, a `mint` or a `serve/`. `show` mints nothing, dials
nothing, and reads the clock only to open the Store's handle at one instant, so
every timestamp in a golden came out of a seeded file.

## The entry, and what each case is about

- [`an-entry-read-back/`](an-entry-read-back) — the whole of it on one page: a
  `read` Step and a `mutate` Step, each Disposition with the Record identities
  it acted on **as members**, `hyper`'s own account of the work beside them, each
  Step's Provenance beside that Step and the Run's beside the Run. Its `-json`
  twin is the same rows on the wire.
- [`an-expansion-in-expansion-order/`](an-expansion-in-expansion-order) — the
  same entry under `--expansion`: the selector, what it expanded to, and the
  Bound. `expanded_to` comes back in **Expansion order and is not sorted**, which
  the members beside it — sorted, being a set — is what shows.
- [`a-values-selector-as-authored/`](a-values-selector-as-authored) — the
  literal-list form, rendered as authored, because §6 orders an Expansion by the
  artefact where the selector is a list and sorting it would hide which member a
  Run reached first.
- [`a-status-that-is-not-the-ordinary-answer/`](a-status-that-is-not-the-ordinary-answer)
  — both halves of ADR-0050 in one entry: the `404` that **completed** a
  `destroy` and the `500` that **halted** one, each carrying the host reached and
  the status got. The `read` Steps everywhere else in this corpus carry neither.
- [`a-halted-destroy-under-expansion/`](a-halted-destroy-under-expansion) — that
  entry under `--expansion`, where the halt point of a serial `destroy` is
  legible by position: two members of five in the identity set, and the sequence
  saying which two.
- [`a-projection-failure-names-the-path/`](a-projection-failure-names-the-path) —
  the path that failed to project, beside the partial set the Step wrote. The
  entry holds one Step file and the Run halted there, so the Steps after it
  wrote nothing and have no row here and no `provenance` row either — which is
  the whole of what `show` can say about a Step that was never reached, that
  Disposition being read from a silence rather than from a file (§7). The
  `-json` twin asserts the wire half of that absence.
- [`a-digest-with-no-members/`](a-digest-with-no-members) — two entries of one
  Procedure, the newer holding a digest and no members. `show` resolves them
  from the Run that last carried them and **names it**: another entry's bytes are
  read only by saying so.
- [`dispositions-that-conclude-about-nothing/`](dispositions-that-conclude-about-nothing)
  — the distinction §7's own exception to the absence rule exists to keep. Three
  Steps carrying no identity set write no `records` key at all; the fourth
  concluded about nothing and writes `[]`.
- [`a-shell-answer-and-a-nested-step/`](a-shell-answer-and-a-nested-step) — the
  other Capability's answer, the command beside the code it exited with, and a
  Step reached through a nested Procedure named by its invocation chain.
- [`a-refused-entry/`](a-refused-entry) — the entry's account includes the
  checks that declined it: one `refusal` row per problem, in the array's order,
  and never one row carrying an array. The page renders `check`'s problem table
  beneath the header, which is the deferral `run` and the pin gate both already
  state — §8's caret excerpt and its `EDIT ONE OF` table are
  [#169](https://github.com/TheLoomLabs/hyper/issues/169)'s, and it is blocked
  on this ticket so that it moves both sites at once.

## The four accounts an entry can have

§7 classifies an entry by which files stand under it, and each of the four is a
case:

- **closed by its own Run** — every case above.
- **open** — [`an-open-entry/`](an-open-entry): no outcome and no closer, so the
  state is named beneath the header rather than written into a cell named for
  §12's triple.
- **reaped** — [`a-reaped-entry/`](a-reaped-entry): the closing write is the
  entry's only account, and the Step the dead Run went quiet on arrives in the
  shape a Step file records one.
- **contested** — [`a-contested-entry/`](a-contested-entry): both stand. The
  header is the owner's, unqualified, and the contest is one stated line per
  `closed-by/` file in the form §8's Comparison header uses.

The last two carry a `-json` twin each, because the page and the wire may not
state different things and the closing writes are where they most easily could:
`closed_by` is the wire's whole account of a contest, and the Step a closing
write reports carries **no `provenance` row at all** — a reaper establishes a
Step's code facts and never its revisions, and a row with every member omitted
would state that absence twice.

Beside them, [`an-entry-with-no-step-file/`](an-entry-with-no-step-file) is a
Run that ended before it wrote one, [`a-rehearsal-says-so/`](a-rehearsal-says-so)
carries the marker every consumer of Journal evidence filters on, and
[`a-run-on-a-runner/`](a-run-on-a-runner) carries the Trigger members only the
Actions executor has — and a `repo_dirty` revision, which renders with the `+`
suffix that stops the page asserting those were the bytes at that commit.

## The two failure paths, and the order between them

The Store is the namespace a `<run-id>` resolves against, so `store-absent`
necessarily precedes the lookup:

- [`store-absent/`](store-absent) and
  [`store-absent-precedes-the-lookup/`](store-absent-precedes-the-lookup) —
  `77`, on a well-formed id and on a partial one alike. The missing branch is
  reported rather than the id blamed.
- [`an-unknown-run-id/`](an-unknown-run-id) and
  [`a-partial-run-id/`](a-partial-run-id) — `2`, no `error_code`, naming what was
  typed, the namespace it resolved against and `hyper runs` as the command that
  enumerates it. **Both messages are the same shape**, which is the corpus
  stating that nothing anywhere resolves a partial id: a prefix of a real id is
  a name matching nothing, exactly like a typo.
- [`a-remote-that-cannot-be-read/`](a-remote-that-cannot-be-read) — the sync is
  attempted and its failure tolerated. The branch this clone holds is read, and
  the line saying it may be stale is narration on stderr rather than a Refusal.
