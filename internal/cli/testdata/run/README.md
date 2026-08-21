# The tracer bullet

`hyper run <procedure>` is the whole tool on one thin path: an artefact, a
`check`, a call, a projection, a Record, a Store, a Journal entry, a page
(issue #136). This corpus drives every one of them.

## What a Run's determinism costs, and who pays it

Three of the reads `cli.Process` names land in what a Run writes, and all three
are supplied by the case rather than by the machine the suite runs on.

- **`mint`** names the Run ids the process answers, one per line and in order.
  A Run id lands on the terminal line, in the `outcome` row, in `run.json` and
  in every Store path a Run writes, so this file is what makes each of those a
  checked-in constant. §8 states that a Run id renders **whole** (ADR-0047), and
  a corpus normalising it to `<run-id>` could not check the one rendering rule
  that surface has.
- **`actor`** and **`hostname`** are who is running `hyper` and on which
  machine, which a Journal entry's Trigger carries. Absent, the harness's stated
  `igor` and `thinkpad`.
- **`now`** is the clock, as everywhere else, and every instant a Run records
  comes through it — so a case's `started_at`, `written_at` and `ended_at` are
  one value and that value is the case's.

`repo_revision` is not supplied: it is the fixture's own commit, which is
reproducible because `golden_fixture_test.go` states both identities and both
dates outright. `procedure_revision`, `definition_revision` and
`manifest_digest` are content-addressed over bytes the case checks in.

## Two repositories, shared

A materialised case cannot reach a repository through a `--repo-dir` in its
argv — it is driven against a copy in a temp directory, and a checked-in path
would stand it in a directory that is inside no git repository at all. So a case
names one in a **`repo-from`** file instead, and the two here are:

- [`repo-watch-status/`](repo-watch-status) — §3's own `uptime` Manifest, one
  `read` Operation, no credential, `class: local`, bound by a Definition and a
  Procedure of one Step with no selector. It is the tracer bullet's repository.
- [`repo-not-built-yet/`](repo-not-built-yet) — the same, plus the artefacts
  three later milestones need: a `mutate` Step, a `read` Step carrying an
  `over:` selector, and a nested invocation.
- [`repo-untracked/`](repo-untracked) — `repo-watch-status` with no
  `definitions/` in it, so the case that adds one through `uncommitted/` is
  running against an artefact git has never seen.

The rest carry a repository of their own, each written for the one edge it
drives.

## What each case is for

| case | what it holds |
| --- | --- |
| `the-tracer-bullet`, `-json` | the Run end to end: one Observation, one entry, the Step table and the terminal line, in both modes |
| `a-second-run-against-an-unchanged-answer` | the seeded branch already holds the answer, so the Run mints **no** second version and its Step file carries the digest alone |
| `a-changed-answer-mints-a-second-version` | the same seed with a moved `status`: a second version, and the first untouched |
| `a-working-tree-that-moved` | an artefact the Run read differs from `HEAD`, so the entry carries `repo_dirty: true` and the Provenance names the working tree's blob |
| `an-untracked-artefact-is-dirty` | the other half of the same sentence: the Definition the Step binds is not committed at all, and the entry says so |
| `a-run-on-a-runner` | the Trigger's other executor: `cause: cron`, the occasion, and no `host` |
| `a-secret-field-is-the-marker` | a Manifest declaring `secret:` — the version carries the constant marker and the value reaches no file |
| `a-host-that-answered-nothing` | the `read` that never halts on what came back: the host is granted and the case serves it nothing, so the Observation records the silence and the Run completes at `0` |
| `a-run-halted-by-its-step` | a Run the world resisted: `failed`, exit `1`, leaving `run.json` and its own `outcome.json` |
| `what-the-run-wrote-reaches-the-remote` | the Run's own commits go out and `remote.golden` shows what arrived |
| `a-repository-with-no-store` | `store-absent`, `77`, naming `hyper store init` — a Run never creates the branch |
| `an-effectful-step-declines`, `a-selector-declines`, `a-nested-invocation-declines` | the three things this binary does not implement, each declining before Step 1 with no entry written |
| `an-effectful-step-declines-before-the-store` | the same decline in a repository with no Store: `2` and not `77`, because a working-tree name is judged before the Store is located |
| `a-procedure-matching-nothing`, `a-definition-rather-than-a-procedure`, `two-positionals`, `a-target-flag` | the four usage errors, all `2`, all with stdout completely silent |

## What no golden here proves

**That a read-only Run's pushes batch to its end.** They do — `internal/run`'s
`closed` is the only place `Publish` is called — but one push at the end and a
push per write leave the remote holding the same commits, so
`what-the-run-wrote-reaches-the-remote` asserts what arrived and not how many
reaches it took. The claim is stated where it is made and is not goldenable
here.

## Why the halted Run halts on an identity path

`a-run-halted-by-its-step` binds an Operation whose `record:` reads its
`identity:` from `$.body.id`, and serves a body with no `id` in it. That is the
one way a `read` Step fails at all: a `read` never halts on what came back, so
what stops it is `hyper`'s reading of the answer rather than the answer (§6,
ADR-0050).

What that case does **not** yet assert is the Disposition. §6 says the Step is
*ran* and carries `projection_failed_path`, and issue #144 is where that lands;
here the Run halts before the Step file is written, so the entry holds
`run.json` and `outcome.json` and no Step file at all.

## Why `a-procedure-matching-nothing` has no repository

It names `../repo-watch-status` with the `--repo-dir` an operator would type,
and that directory is inside no git repository. That is the case: §9 fixes that
a positional resolves before the Store is located, so `hyper run typo` is `2` on
a repository with no Store at all — the typo is repaired before the Store is
missed.
