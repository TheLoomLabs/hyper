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

## The shared repositories

A materialised case cannot reach a repository through a `--repo-dir` in its
argv — it is driven against a copy in a temp directory, and a checked-in path
would stand it in a directory that is inside no git repository at all. So a case
names one in a **`repo-from`** file instead, and the ones here are:

- [`repo-watch-status/`](repo-watch-status) — §3's own `uptime` Manifest, one
  `read` Operation, no credential, `class: local`, bound by a Definition and a
  Procedure of one Step with no selector. It is the tracer bullet's repository.
- [`repo-not-built-yet/`](repo-not-built-yet) — the same, plus the artefacts
  three later milestones need: a `mutate` Step, a `read` Step carrying an
  `over:` selector, and a nested invocation.
- [`repo-untracked/`](repo-untracked) — `repo-watch-status` with no
  `definitions/` in it, so the case that adds one through `uncommitted/` is
  running against an artefact git has never seen.
- [`repo-credentialled/`](repo-credentialled) — the credential half (issue
  #137): a `header:` Provider and a `basic:` one, a Target declaration for
  each naming the variable every slot resolves from, and three Procedures —
  one per Provider, one binding both, and one binding two Definitions to the
  same Target. `paid` carries two slots beyond the one its Provider's scheme
  needs, which is what makes *a slot no Step of this Run could send is not
  required* a case rather than a sentence.
- [`repo-check-refuses/`](repo-check-refuses) — a repository carrying one
  static fault in each of §3's five artefacts, plus a clean one-Step Procedure
  to run. `check` re-runs in full at Run start, so a Run of the clean
  Procedure Refuses with all five.
- [`repo-two-secrets/`](repo-two-secrets) — `repo-secret` with a second Step
  whose Operation also declares `secret:` output, for the gate that names
  **every** such Step rather than the first.
- [`repo-two-reads/`](repo-two-reads) — `repo-watch-status` with its one-Step
  Procedure replaced by a two-Step one, both `read`, one host each. Two is the
  smallest number of Steps that can tell one push at the end from a push per
  Step (issue #138), and it is a repository of its own because adding a
  Procedure to `repo-watch-status` would move the `repo_revision` in every
  golden that names it.

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
| `a-secret-field-is-the-marker` | a Manifest declaring `secret:` — the version carries the constant marker and the value reaches no file. It supplies `--secret-out`, without which the sink gate below would decline it before Step 1 |
| `a-host-that-answered-nothing` | the `read` that never halts on what came back: the host is granted and the case serves it nothing, so the Observation records the silence and the Run completes at `0` |
| `a-run-halted-by-its-step` | a Run the world resisted: `failed`, exit `1`, leaving `run.json` and its own `outcome.json` |
| `what-the-run-wrote-reaches-the-remote` | the Run's own commits go out and `remote.golden` shows what arrived |
| `a-repository-with-no-store` | `store-absent`, `77`, naming `hyper store init` — a Run never creates the branch |
| `the-runner-clone-fetches-the-store` | the runner shape: `hyper-store` on `origin` alone, brought down by the Run's own sync, and the Run proceeds normally |
| `a-sync-that-could-not-reach-the-remote` | the sync fails and the Run **tolerates it**, saying so on stderr, reading the branch the clone holds and completing at `0` — never `75` for a sync it could not complete |
| `a-sync-that-could-not-bring-a-branch` | the same failure with no branch in hand: the same stderr line, then `store-absent` at `77`, because what is missing is an act and not a network |
| `a-later-run-pushes-what-an-earlier-one-stranded` | an earlier Run's unpushed commit and a second environment's published one, over one root: the push is rejected, the **whole** unpushed set is re-applied, and `remote.golden` holds all three Runs |
| `two-read-steps-push-once` | the corpus's only two-Step Run, and what `run_push_test.go` counts the reaches of |
| `an-effectful-step-declines`, `a-selector-declines`, `a-nested-invocation-declines` | the three things this binary does not implement, each declining before Step 1 with no entry written |
| `an-effectful-step-declines-before-the-store` | the same decline in a repository with no Store: `2` and not `77`, because a working-tree name is judged before the Store is located |
| `a-procedure-matching-nothing`, `a-definition-rather-than-a-procedure`, `two-positionals`, `a-target-flag` | the four usage errors, all `2`, all with stdout completely silent |
| `a-store-file-this-binary-cannot-read` | the first gate past `run.json`: a Record head written at schema version 2, `store-schema-unsupported`, and the one Refusal that cites a file with no line and no field |
| `check-refuses-the-run`, `-json` | `check` re-run in full: five codes across the five artefact kinds, one `refusal` row each, in `check`'s own order |
| `a-working-tree-edited-since-check-passed` | the same gate driven the way an operator meets it — one `uncommitted/` line narrows `local`'s `kinds:`, and the Run refuses with the codes the edit earns |
| `a-credential-the-environment-does-not-hold` | one absent slot, `credential-absent`, citing the `env:` line of the declaration whose slot the environment did not fill |
| `every-absent-credential-at-once`, `-json` | three absent slots across two Targets in **one** Refusal, and the two slots `paid` carries that no Step of this Run could send are not among them |
| `one-slot-two-definitions` | two Definitions binding one Target under one scheme require **one** slot between them, so an absent variable earns one member of the array and not one per binding |
| `a-header-scheme-reaches-the-wire` | the `header:` scheme end to end: the Manifest's `name:` and `prefix:`, the variable the Target declaration names, and what arrived at the far end |
| `a-basic-scheme-reaches-the-wire` | the same for `basic:`, whose position and base64 composition are the scheme's and never a Manifest's |
| `a-secret-sink-names-every-step`, `-json` | the sink gate: two Steps declaring secret output, both named at once, neither of them run |
| `usage-secret-out-to-stdout`, `-inside-the-repository`, `-with-no-path` | the three things `--secret-out` will not take, all `2` and all carrying no `error_code` |

## What a Refusal's page looks like, and what stands in for §8's

Every Refusal here renders the same three blocks: `nothing ran. no step was
reached.` where the Step table would be, the problem table `check` already
renders, and §8's terminal line. §8 puts a caret excerpt and an `EDIT ONE OF`
table where the middle block is, and that is milestone 8's — every fact §8
requires is on the page already, and what is deferred is the shape
(`internal/cli/gate.go` states the same deferral for the pin gate).

The Step table is omitted rather than rendered empty, on §8's own reading: an
empty table asserts *we looked at the Steps*, which is false. `stderr.golden`
is where that shows twice over — a refusing case narrates `run <id>` and no
`step` line at all, because no Step was reached.

## How the two credential cases see the wire

A credential is composed from a Manifest's scheme parameters and a Target
declaration's environment variable and then **leaves**. It reaches no file, no
row and no rendering (§7, ADR-0007), so the only place a corpus can observe it
is at the far end — which is why `serve/` grew one key that is not what a server
would supply, `echo_request_headers`, and why the two cases that use it serve no
body of their own.

What lands in each case's `store.golden` is therefore the **server's** account
of what arrived, projected by a Manifest that asked for it — which is `hyper`
recording a response like any other. `hyper` writes no credential anywhere on
its own account: `capability.Credential` has no exported member, no accessor,
no `String` and no `MarshalJSON`, and its only path is the environment, through
the composed header, onto the wire.

A real Manifest would name that projected field in `secret:` and the Store
would hold the constant marker — [`a-secret-field-is-the-marker`](a-secret-field-is-the-marker)
is that case. These two deliberately do not, because a marker proves nothing
about what left, and what left is the only thing they are for. Their values are
each case's own `env` file and are spelled to be unmistakable in both
directions: nothing under `repo-credentialled/` is a credential and neither is
anything in a golden beside it.

## The three ways a Run loses the Store, and where each one is driven

`75` is a Run that lost the Store — to the lock, to the sync at Run start, or to
a push it could not land — and none of the three is a Refusal or a failure of
the work (§12, ADR-0061, issue #138). Two of the three sit above as ordinary
cases. The rest are in [`../../run_store_lost_test.go`](../../run_store_lost_test.go),
each for a reason a golden cannot get past:

- **The lock** is not a directory of files. It is held by a *live* process —
  which is exactly why a crash cannot leave one behind — so the two cases that
  drive it take it in the test process and run the command against the same
  repository.
- **The exhausted push** renders git's own account of the rejection, and that
  account names the bare repository by path: a temp directory, different on
  every run of the suite. Its streams are asserted by what they say; its two
  branch goldens, which name no path and no commit, are checked in beside
  [`a-push-rejected-three-times/`](a-push-rejected-three-times) and compared
  like any other case's.

## What no golden here proves

**That a read-only Run's pushes batch to its end.** One push at the end and a
push per Step leave the remote holding the same commits, so no branch golden can
tell them apart. What tells them apart is how many times the remote was reached,
so that is counted instead:
[`../../run_push_test.go`](../../run_push_test.go) installs a receive hook on
the bare origin that accepts a push and tallies it, drives
[`two-read-steps-push-once`](two-read-steps-push-once), and holds the tally at
one.

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
