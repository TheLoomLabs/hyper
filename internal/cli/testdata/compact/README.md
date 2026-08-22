# `hyper compact`

The second command to touch the record, and the first thing in the tool that
removes anything (issue #131). Every case here drives `hyper compact` through
`cli.Main` from its own `argv`, and asserts the two streams, the exit code, and
— where there is a branch at all — the tree the run left behind.

Every seeded `store/` here holds files `internal/store` itself wrote: the Record
versions are its canonical encoding at the paths its grammar builds, and the
Journal entry's `run.json` and `outcome.json` are the shapes issue #129 landed.
That is what makes a `store.golden` an assertion rather than a transcript — a
version the reader could not decode would fail the run rather than survive it.

No case supplies a `now`. The harness's stated instant is the clock retention is
measured against, and every seeded version's age is written relative to it, so
the cases say what they mean without pinning a date in a golden.

## The predicate, one case per clause

`compact` removes a version exactly where it is not the Head, not the series'
first version, and older than `retention:`. The corpus is the four ways that can
come out.

- `removes-interior-observations/` — the tracer bullet, and the seed §7's own
  worked figures describe: an Observation series of five versions across a year,
  an Asset series whose every version is years old, a series ending in a
  Tombstone, and a Journal entry older than any policy. Two interior
  Observations go and **everything else stands**, which `store.golden` holds
  path by path. The Journal directory it left is the same two files it started
  with, which is also how the corpus states that this command writes no Journal
  entry: `compact` is not a Run, and a run of it that had written one would
  render a third path here.
- `never-removes-evidence/` — the same seed built to tempt every removal the
  predicate forbids, and from which nothing is removable at all: the Asset
  series, the Tombstone and the Journal entry again, plus an Observation series
  of exactly one version and one of exactly two whose Head is the file that
  sorts *first* by name. `store.golden` is byte-identical to the seed.
- `everything-younger/` — a policy in force and nothing old enough to fall
  under it. Nothing is removed and the branch does not move.
- `no-retention/` — `hyper.yaml` declares no `retention:`, so nothing is read
  and nothing is removed. §3 is explicit that omitted means nothing is ever
  removed: a repository that has not stated a policy has not agreed to lose
  anything.

The last two are the same exit code by different roads, and the corpus is what
holds them apart: `removed nothing · retention 90d · …` against `hyper.yaml
declares no retention: — nothing is ever removed`. Both exit `0`, and a reader
who could only see the code would have to guess which happened.

`retention-not-a-duration/` is the third road to the same place. A `retention:`
that the duration grammar does not admit is `schema-mismatch` and `check` is
what reports it (ADR-0064); what `compact` does about it is remove nothing and
say where to look.

`no-retention-with-environment/` is the rule that no variable stands behind the
flags this command does not have. Its `env` sets `HYPER_RETENTION` and
`HYPER_KEEP_VERSIONS`, and its `store.golden` is byte-identical to
`no-retention/`'s: retention is read-time and lives in one reviewed artefact,
and an invocation that could widen it from the environment would remove more
than the repository ever agreed to (§7, ADR-0001, ADR-0014).

## The two forms

`removes-interior-observations-json/` and `no-retention-json/` are the wire: one
`version` row per removed version in path order, and the terminal `result` with
`truncated` always `false` — `compact` takes no `--limit` and answers no
question with a result set to cut. A row names its version by its Run and its
Step and **never by its ordinal**, which this command is precisely the thing
that moves (ADR-0049); `step` rides on the wire only, having no column on the
page.

The empty forms are `no-retention-json/` and `everything-younger-json/`: the
`result` row alone on stdout, and nothing else — a stream that opened is a
stream that terminates, whether or not it carried a row. What tells the two
apart there is **stderr**, which carries the command's own line where no row was
written. That is `targets`' truncation line read once more (§9, ADR-0026): a
fact that is not a row has no place on stdout, and a stream of no rows says
which nothing happened to nobody. It is written only where there are no rows —
where there are, the rows are the answer, and this is the one thing the wire
cannot otherwise state.

## The Refusals

`store-absent/` and `store-absent-json/` are the Refusal that unblocks milestone
5 — the branch was never created, so the command Refuses at `77` naming `hyper
store init`, renders two lines on stderr with stdout silent, and opens no row
stream in either mode. `--json` carries no Refusal because a Refusal is not a
row (§9); §8's caret form is milestone 8's renderer.

`schema-unsupported/` is the other one this command can reach: a Record version
carrying `schema_version: 2`, which is above this reader's ceiling. It Refuses
at the same code with a different remedy — a different binary rather than an act
of somebody's — and cites the file, which is the one Refusal whose subject is
evidence rather than an artefact (ADR-0028).

It declares a `retention:`, and that is load-bearing rather than incidental. The
ceiling is a **reader's** rule and `compact` tests the files it will read, so the
same file under a repository that declared no policy is never opened and the
command exits `0` — which is why this case states a policy and `no-retention/`
does not.

## The gate

`version-pin-mismatch/` and `version-pin-absent/` Refuse at `77` before the
branch is read. The first seeds a whole Store and its `store.golden` is
byte-identical to that seed, which is the half a stderr comparison cannot show:
the gate fired before a single git subprocess ran, and nothing was removed. The
second carries no `hyper.yaml` at all.

## The usage errors

`usage-positional/`, `usage-dry-run/`, `usage-keep-versions/` and
`usage-retention/` all exit `2` with stdout silent. The last three are the
flags this command does not have and will not grow: retention is read-time and
lives in the Repository declaration alone, because a flag would let one
invocation remove more than the repository ever agreed to (ADR-0001), and there
is no `--dry-run` on this command: §9 gives that flag to `run` and to no other,
a `compact --dry-run` having nothing it could mean — the branch is append-only
and `git log` on it is its own account of what was removed (§9, ADR-0015). None
of them carries a repository, because none of them needs one — the fault is
decided from the argument list alone and before any root is resolved.

## What is not here

The three rules a case directory cannot state — a push that reaches nowhere, a
remote that will not answer while the branch is absent locally, and a dirty
working tree left exactly as it was — are in `../../compact_test.go`, driven
through the same entry point. The push re-application, the removal that is a
no-op on a tip that no longer holds the path, and the push exhausted after three
attempts are `internal/store`'s own, in `push_test.go`.
