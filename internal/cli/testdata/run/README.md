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
- [`repo-effectful/`](repo-effectful) — the effectful spine (issue #148): a
  `cloudflare-dns` Manifest whose `create_dns_record` is a `mutate` of `one`
  cardinality, a Definition claiming `mutate` against a credentialled Target,
  and four Procedures — one `mutate` Step, two of them, one invoked through a
  nested Procedure, and `repo-watch-status`' own read-only Procedure beside
  them, which is what lets one fixture drive both arms of a rule that splits by
  Kind. Each case serves its own `api.cloudflare.com`, so what the world answers
  is the case's and the artefacts are shared.

  Its Operation declares `deadline: 1s` and `retry: {attempts: 3}` for the same
  reason [`repo-drain/`](repo-drain) declares the first: a number an artefact
  declared is a number a case can drive to, and a suite that waited thirty
  seconds for one of them would be a corpus nobody runs. The retry is what makes
  *an exhausted retry leaves the response object for the projection to read* a
  case rather than a sentence — the refused connection below is retried three
  times and is still *attempted, world untouched*.

  It was `repo-not-built-yet/` while the binary declined every effectful Step,
  and it held a Step carrying an `over:` selector until issue #139 built the
  Expansion and a nested invocation until issue #141 built that. Each of those
  moved to the corpus of the thing it is now an example of, and issue #148
  renamed what was left to what its artefacts now demonstrate rather than
  leaving it under a name that had stopped being true.
- [`repo-bounded/`](repo-bounded) — the Bound at Expansion (issue #149): the
  same `cloudflare-dns` shape cut to what a count needs — one `mutate`
  Operation whose `identity:` is a template hole, a Definition claiming
  `mutate`, a credentialled Target, and one Procedure whose Step carries a
  `bound:` beside an `assets:` selector. It is a repository of its own rather
  than a Procedure added to `repo-effectful` because adding one there would
  move the `repo_revision` in every golden that names it.
- [`repo-destroy/`](repo-destroy) — the `destroy` Kind (issue #150): the same
  `cloudflare-dns` shape with a `delete_dns_record` beside the `create` — a
  `destroy` of no `record:` block at all, projecting nothing and declaring no
  identity (ADR-0037) — a Definition claiming the Operation under `destroy:`, a
  Target accepting both effectful Kinds, and three Procedures: one `destroy`
  Step over an `assets:` selector under `bound: 5`, the same Step followed by a
  `mutate` that writes above the Tombstone, and the same selector under a
  `bound: 2` the Expansion is past. It is a repository of its own for
  `repo-bounded`'s reason above.

  Every Step in it carries a `bound:`, which is not this corpus's choice: a
  `destroy` Step with none is `bound-missing` at `check`, and `check` re-runs in
  full at Run start (§4, §5, §6).

  Its cases serve **one** host between all the members of one Expansion, which
  nothing else here does — a serial Expansion is one request at a time, so an
  `answers:` list is walked in Expansion order and *the fourth member is the one
  that was answered `500`* is a fact the case states rather than a race it
  hopes for (§6).
- [`repo-cadence-run-once/`](repo-cadence-run-once) — a Procedure declaring a
  Cadence over a nested Procedure whose Step is run-once (issue #169). It is a
  copy of [`check/cadence-run-once/repo/`](../check/cadence-run-once/repo)
  rather than a `repo-from` reaching across into it: a `check` case's `repo/` is
  that case's own, and a corpus reaching into another's fixture is one where an
  edit made for one command breaks a golden of a different one. It carries the
  workflow its Cadence projects, for the reason the one below does.
- [`repo-cadence-malformed/`](repo-cadence-malformed) — a Procedure declaring a
  Cadence outside §10's five-field grammar, `0 3 * * MON` (issue #174). It is
  `repo-cadence-run-once/` with the nested Procedure dropped and the Definition
  narrowed to `read`, so the Run refuses on **one** code: what this case drives
  is a Procedure's own per-file check reaching a Run's pre-flight, and a second
  code on the page would leave which of the two got it there unsaid.

  **Both carry a `hyper-*.yml` that no `project` could have written**, and that
  is what keeps the *one code* above true (issue #179). A Procedure declaring a
  Cadence with no generated file beside it is `projection-stale`, so either
  repository without one refuses twice — and since `project` refuses these
  repositories for the very codes they exist to drive, the projection they carry
  was written by hand from the generator's own bytes. That is also what it once
  cost: §11's compiled-in constants appear in those bytes, so a runner label or
  a `checkout` SHA that moves takes every checked-in `hyper-*.yml` in the corpora
  with it, and nothing regenerated them.

  **The one flag now carries them** (issue #181).
  `TestGoldenCorpora_AFixtureWhoseProjectionIsCurrentRegeneratesToItself` holds
  these two — and the six other fixture repositories whose projection must stay
  current — to a fresh `verify.Projection` at the pin each one's own `hyper.yaml`
  declares, naming the repository and the file where they part; and `-update`
  regenerates them **before** the first case is driven, an input being settled
  before anything reads it. Nothing about these two makes them a special case for
  it: the generator judges nothing it is handed (ADR-0064), so a Cadence outside
  the grammar and a run-once Step project like any other and a regeneration
  reproduces them exactly — which is what keeps them maintainable rather than
  frozen.

  What a repository that declares a Cadence and holds no file for it looks like
  is [`check/projection-stale`](../check/projection-stale)'s, and a Run's own is
  `a-stale-projection` below.
- [`repo-relative-bound/`](repo-relative-bound) — §8's Refusal in full (issue
  #169): `repo-destroy`'s Manifest, Definition and Target unchanged, over one
  Procedure whose `destroy` Step carries a **relative** predicate — `created_on
  older_than: 14d` — with its `bound:` written *beneath* the `over:` block, so
  that the caret excerpt at the Bound renders the operand in its context and the
  `=` note has something to gloss. Its Store seeds five Assets whose
  `created_on` spread across the ladder: four are older than the Run's start by
  more than fourteen days and one of those by more than thirty, which is what
  puts *expansion resolved 4* on the caret and *expands to 1* on the narrowed
  selector's row. It is a repository of its own for `repo-bounded`'s reason
  above.
- [`repo-literal-destroy/`](repo-literal-destroy) — the `destroy` by literal
  identifier (issue #151): `repo-destroy`'s Manifest, Definition and Target
  unchanged, over four Procedures whose selectors are `values:` lists rather
  than predicates — a `destroy` over two literals, the same over three, a
  `mutate` over one, and a `destroy` over a literal spelled `Preview-42` against
  a standing `preview-42`. It is a repository of its own for `repo-bounded`'s
  reason above, and its cases differ from each other in the **branch they were
  seeded with** rather than in the artefact: one Procedure over an empty Store
  opens two series, and over a Store holding a Tombstone it reaches one member
  fewer.
- [`repo-opaque-destroy/`](repo-opaque-destroy) — §6's own worked example
  (issue #151): the built-in `shell` Provider's `destroy`, a Definition claiming
  it under `destroy:`, a Target that grants `shell` and opts into
  `opaque-destroy:`, and the `purge-releases` Procedure §6 prints — an `over:`
  `values:` list of two paths, `args: {command: [rm, -rf, {item: $}]}`, and no
  `bound:` anywhere, one being `bound-illegal` on such a Step (§4, ADR-0053).
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
- [`repo-drain/`](repo-drain) — the drain (issue #140): one `read` Operation
  declaring `deadline: 1s`, over three granted hosts, and a Procedure expanding
  over all three in an order whose sorted set is not it. The middle host is the
  one the case does not answer for, which is what reaches the deadline.
- [`repo-concurrency/`](repo-concurrency) — how much runs at once (issue #140):
  three `read` Operations over eight granted hosts, identical but for one key —
  one declaring `concurrency: 4`, one declaring `2`, and one declaring nothing
  at all — and four Procedures. What differs between the cases that drive it is
  a Manifest key and nothing else, which is ADR-0045's whole claim.
- [`repo-nested/`](repo-nested) — the nested invocation (issue #141): the
  `uptime` Manifest and the `inventory` one whose `identity:` does not resolve,
  and six Procedures — one invoking a Procedure that invokes a third, one whose
  nested Procedure halts, and one flat four-Step Procedure that halts at its
  second. A Procedure invoking another runs as **one** Run, so what these drive
  is one entry, one outcome and one exit code however deep the invocation goes.
- [`repo-conditions/`](repo-conditions) — the condition (issue #141): the same
  `uptime` Manifest with `check_named` beside `check_http`, and four Procedures —
  one Step guarded by a `when:`, two guarded in sequence, one whose guarded Step
  carries a selector that would Refuse, and one whose condition is handed a value
  it cannot compare. Each case serves its own `status`, so what decides whether a
  condition holds is the case's `serve/` and the artefacts are shared.
- [`repo-expansion/`](repo-expansion) — the Expansion (issue #139): the same
  `uptime` Manifest with two Operations beside `check_http` — `check_named`,
  whose `identity:` is a template hole and therefore resolves **before** the
  call, and `check_ttl`, which wires an `integer` input into a request header
  the fixture echoes back — and eleven Procedures, one per shape the Expansion
  has. Each case seeds its own branch under `store/` and names the Procedure it
  runs, so the population a selector reaches is the case's and the artefacts are
  shared.
- [`repo-shell/`](repo-shell) — the `shell` Capability's `read` half (issue
  #142): the built-in Provider bound by a `read` Definition and a `mutate` one,
  a `local` granting `shell`, a second Target declaration nothing binds whose
  `auth:` names the variable the child must not inherit, and eight Procedures —
  one per shape a command has. The argv is the **Procedure's** and
  the binary is the **case's**: each case holds a `bin/` directory its argv head
  resolves against, so what a command printed, what it exited with and whether
  it could be started at all are the fixture's facts rather than the machine's.
- [`repo-patterns/`](repo-patterns) — the three Patterns (issue #143): an
  `inventory` Manifest whose Operations differ only in the `patterns:` block
  they carry — two paginated, one of them writing its cursor into a `query:`
  and the other its page number into a `header:`; one polled; and two `read`
  Operations identical but for a `retry:`, which is what makes *a Pattern does
  not change the number of Records a Step affects* a comparison rather than a
  claim. Eleven Procedures, one host per Procedure, and each case's own `serve/`
  says what the world answers on the second call and the third — the Patterns
  being the one thing in the tool whose subject is a world that changes between
  calls.
- [`repo-projection/`](repo-projection) — when a projection does not resolve
  (issue #144): an `inventory` Manifest of four `read` Operations differing in
  the one thing this corpus is about — one of `one` cardinality projecting two
  fields, one of `series` cardinality whose `over:` names a collection, one
  whose `over:` names a path no response carries, and one of `series`
  cardinality whose `identity:` is a template hole rather than a response path —
  and six Procedures, one per shape the failure has: one host, a pair, three,
  and the three collections. Each case's own `serve/` is what decides whether a
  path resolves, so the artefacts are shared and the fault is the case's.
- [`repo-shell-unresolvable/`](repo-shell-unresolvable) — `repo-shell` cut down
  to the one artefact that does not load: a Step whose argv word references a
  field the Record it names does not carry. It is a repository of its own
  because **`check` re-runs in full at Run start** (§6) — one artefact that
  Refuses Refuses every Run of every Procedure beside it, so a broken Procedure
  living in a shared repository would leave every case sharing it asserting one
  Refusal under ten different names.
- [`repo-cyclic/`](repo-cyclic) — `repo-watch-status` whose one Procedure
  invokes **itself** after its Step (issue #146). It is a repository of its own
  for the reason above: a cyclic invocation graph is `procedure-cycle` at
  `check`, and `check` re-runs in full at Run start, so it would Refuse every
  Run of every Procedure beside it.
- [`repo-skip-if-recorded/`](repo-skip-if-recorded) — Repeatability that reads
  something (issue #152): §4's own `create_dns_record` with
  `repeatability: skip-if-recorded` in its place and `identity: "{name}"`, a
  Definition claiming `mutate`, a credentialled Target, and three Procedures —
  §4's `publish` Step carrying no selector, its `publish-aliases` Step over a
  `values:` list of three, and the two of them in one Procedure. It is a
  repository of its own for `repo-bounded`'s reason above, and its cases differ
  from each other in the **branch they were seeded with** rather than in the
  artefact, as `repo-literal-destroy`'s do. Every case but one serves
  `api.cloudflare.com`, so what stopped a call is the skip and never a refused
  connection; the exception serves a host it never dials, which is how the
  request that provably never left is driven.
- [`repo-skip-if-recorded-shell/`](repo-skip-if-recorded-shell) — the other form
  the same value's test resolves its identity through (issue #152): the built-in
  `shell` Provider's `mutate_skip_if_recorded`, whose `identity:` is
  `$.command` — a fact about the call rather than about the answer, so it is
  known before one goes out. One Definition, a `local` granting `shell`, and one
  Procedure over a `values:` list of two paths. It is a repository of its own
  rather than a Procedure added to `repo-shell` for `repo-two-reads`' reason.
- [`repo-run-once/`](repo-run-once) — the other value that reads something, and
  the one nobody writes (issue #153): the same `create_dns_record` with
  `repeatability:` **omitted**, which is what makes it run-once, and a
  `refresh_dns_record` declaring `repeatable` beside it — the Step that runs in
  front of a run-once one, so that *a Step the halt never reached* is a
  Procedure rather than an entry with nothing in it. One Definition, a
  credentialled Target, a run-once `delete_dns_record` beside them so that the
  value is driven on **both** effectful Kinds, and six Procedures: the one
  run-once Step alone, a run-once Step with one after it, the
  repeatable-then-run-once pair, one invoking the first of them **twice**, the
  run-once `destroy`, and a run-once Step behind a `when:`. It is
  a repository of its own for `repo-bounded`'s reason above, and its cases
  differ from each other in the **Journal they were seeded with** rather than in
  the artefact — which is the whole of what this value reads. Every case serves
  `api.cloudflare.com`, so what stopped a call is the Refusal and never a
  refused connection.

- [`repo-rehearsal/`](repo-rehearsal) — where a rehearsal stops (issue #155):
  `repo-effectful`'s two Manifests and two Definitions, a `local` granting two
  hosts and a credentialled `cloudflare-prod` accepting both effectful Kinds,
  over three Procedures that each put a `read` Step **in front of** an effectful
  one — `read`, `mutate`, `read`, so that a Step withheld and a Step after it
  are two different rows; `read` then `destroy` over an `assets:` selector, so
  that the stop is driven on both effectful Kinds; and the first of them again
  with a `when:` on the `mutate` that the `read` in front of it does **not**
  satisfy. It is a repository of its own for `repo-bounded`'s reason above.

  Its cases serve the `read`'s host and **not** `api.cloudflare.com`, which is
  what makes *no call went out* an assertion rather than a hope: a rehearsal
  that reached the effectful Step would have its connection refused, and the
  Run would be `failed` at `1` with the Step *attempted, world untouched*
  instead of `completed` at `0`.

- [`repo-unscoped-destroy/`](repo-unscoped-destroy) — `repo-destroy`'s Manifest,
  Definition and Target unchanged, over one Procedure whose `destroy` Step
  carries its mandatory `bound:` and **no `over:` at all** (issue #157). It is a
  repository of its own for the reason above, and it is the one repository here
  written to be refused: the shape it holds is `destroy-unscoped` at `check`,
  which is the whole of what its case asserts.

- [`repo-shell-effects/`](repo-shell-effects) — the `shell` Capability's
  effectful half (issue #156): the built-in Provider bound by one Definition
  claiming `mutate` under `kinds:` and both `destroy` Operations under
  `destroy:`, a `local` granting `shell` and opting into `opaque-destroy:`, and
  six Procedures — a `mutate` Step, two `mutate` Steps so that a drain has a Step
  to withhold, a `mutate` over a `values:` list of three, a `destroy` over one of
  three, a `mutate_once` Step, and a `destroy_once` Step over a `values:` list.
  It is a repository of its own rather than Procedures added to `repo-shell` for
  `repo-two-reads`' reason, and the argv is the Procedure's while the binary is
  the case's, exactly as it is there. The two cases that halt a `destroy` and
  shorten one drive [`repo-opaque-destroy`](repo-opaque-destroy)'s
  `purge-releases` instead, with a `bin/rm` of their own: §6's own worked example
  is where a repeatable `destroy` over authored paths already lives.

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
| `a-run-halted-by-its-step` | a Run the world resisted: `failed`, exit `1`, the Step *ran* with the set it concluded about, and the entry left where it stopped |
| `what-the-run-wrote-reaches-the-remote` | the Run's own commits go out and `remote.golden` shows what arrived |
| `a-repository-with-no-store`, `-json` | `store-absent`, `77`, naming `hyper store init` — a Run never creates the branch. It is one of the two paths that **decline before a Run is identified**, so stdout still carries §8's terminal line and its `outcome` row: what is missing there is the row's `run_id` and never the row (§8, §9, issue #172) |
| `version-pin-mismatch`, `-json` | the other of the two: the gate fires before the positional is resolved or the Store is reached, and the Run answers `refused · exit 77` naming nothing to look up |
| `the-runner-clone-fetches-the-store` | the runner shape: `hyper-store` on `origin` alone, brought down by the Run's own sync, and the Run proceeds normally |
| `a-sync-that-could-not-reach-the-remote` | the sync fails and the Run **tolerates it**, saying so on stderr, reading the branch the clone holds and completing at `0` — never `75` for a sync it could not complete |
| `a-sync-that-could-not-bring-a-branch` | the same failure with no branch in hand: the same stderr line, then `store-absent` at `77`, because what is missing is an act and not a network — and the terminal line beneath it, that Refusal being a decline before a Run is identified like the one above |
| `a-later-run-pushes-what-an-earlier-one-stranded` | an earlier Run's unpushed commit and a second environment's published one, over one root: the push is rejected, the **whole** unpushed set is re-applied, and `remote.golden` holds all three Runs |
| `two-read-steps-push-once` | a two-Step Run with a host each, and what `run_push_test.go` counts the reaches of |
| `two-effectful-steps-push-three-times` | the other rhythm, driven by the same test: two `mutate` Steps, two Assets, and three reaches — the sync, then one per Step |
| `a-read-only-sync-in-the-same-fixture` | the read-only half of the sync that splits by Kind: an unfetchable remote, tolerated, and the Run completes at `0`. The effectful half is `an-effectful-sync-that-could-not-reach-the-remote`, which cannot be a golden — see below |
| `a-mutate-step-lands-an-asset` | the effectful spine end to end (issue #148): one `mutate` Step over `http`, `2xx`, one version with `record_type: asset`, *ran*, exit `0` |
| `an-effectful-step-inside-a-nested-procedure` | the same Step reached through an invocation, running exactly as one written at the top level does — one Run, one entry, and the nested Step's `path` on its file and its version |
| `a-mutate-against-an-unchanged-answer` | the case ADR-0030 exists for, on the effectful side: the branch already holds the Asset the call returns, so the Run **mints nothing** and the Record is in the identity set all the same. `store.golden` holds one version |
| `an-effectful-halt-leaves-what-it-did` | a `mutate` answered `500`: the Run halts at exit `1`, the Step is ***ran*** with `answered` carrying the host and the status and **no** `error_code`, and the Step after it is *never reached* and writes no file |
| `a-mutate-answered-a-redirect` | the same halt on a `302`, which is where *completes on `2xx`* is a rule rather than a synonym for *did not fail*. Its `Location:` names a host the case does **not** serve, so a redirect followed would reach a refused connection and the Step would be *attempted, world untouched* — that the Step is ***ran*** carrying `status: 302` on `api.cloudflare.com` is the assertion that none was followed, a redirect target being reach arriving from data |
| `a-mutate-whose-connection-was-refused` | the request that provably never left: ***attempted, world untouched***, `answered` carrying the host **alone**, no identity set, and `–` in `RECORDS` — the safest state in the tool, rendered as the absence of doubt rather than as `0 of 1` |
| `a-mutate-that-reached-its-deadline` | the other `attempted`: the call went out and nothing came back, so the Step is ***attempted, outcome unknown***, carries no `answered`, and its `pattern` reads `attempts: 1` — the one Disposition §7 writes a single attempt on, which is how *nothing was retried* is legible |
| `a-step-reference-reads-an-earlier-steps-record` | `{step:, path:}` resolved: the second Step's `host:` is the first Step's Record read at its turn, and the two Steps write two versions of one series |
| `a-nested-invocation-is-one-run` | a Procedure invoking a Procedure that invokes a third: four Steps in one written order, **one** `run.json`, one `outcome.json`, one exit code and one Run id. The three nested Steps carry `path` on their files and on the versions they wrote; the two top-level ones carry none, and the two invocations write no file at all |
| `a-halt-inside-a-nested-procedure`, `-json` | the halt inside a nested Procedure is a halt of the whole: the Step after the invocation is *never reached* like the one beside it, **neither writes a Step file**, and both still have a row on the page and a `step` row on the wire rendering `–` |
| `a-run-halted-at-step-two-of-four` | the same arithmetic with no invocation in it: Steps 3 and 4 *never reached*, two Step files in the entry and four rows on the page |
| `a-condition-that-holds`, `a-condition-that-does-not-hold` | one Procedure and two `serve/` directories: the guarded Step runs where `probe` answered `200` and is *skipped by condition* where it answered `503`, reaching no Target. The skipped Step's file carries no identity set and no selector, and its cell renders `–` |
| `a-skip-propagates`, `-json` | two conditions in sequence: `middle` is skipped, so `last`'s condition names a Step that wrote no Record in this Run and is skipped in its turn — and the Run **completes** rather than Refusing. The wire says the rest: a *skipped by condition* `step` row carries no `records` key at all |
| `a-condition-does-not-read-the-store` | the same Procedure over a branch whose head for `middle`'s own Record would satisfy `last`'s condition. It is skipped all the same: a condition reads this Run's Records and never falls through to the Store |
| `a-condition-reads-an-unchanged-record` | the seeded branch already holds `probe`'s answer, so the Run mints no version — and the guarded Step runs, a Record going unchanged not being a Record going missing |
| `a-condition-decides-before-the-expansion`, `the-selector-that-condition-spared` | one Procedure whose guarded Step carries a selector that resolves two members to one identity, and two `serve/` directories. Where the condition does not hold the Step expands over nothing and the Run completes; where it holds, the same selector Refuses `record-identity-collision` — which is what says the first case's Expansion never resolved. The Step after it runs in the first and is *never reached* in the second |
| `a-condition-that-cannot-compare` | a condition handed a value its operator cannot compare: `predicate-type-mismatch` at the second Record root, the Step *refused*, and nothing reached (ADR-0035) |
| `an-expansion-over-values` | the demonstration §5 writes out: two members, `{item: $}` into the Operation's `host-input:`, two Observations, and `RECORDS 2`. Its `declared` is the authored list and its `expanded_to` is the same order, where the identity set beside them is sorted |
| `an-expansion-over-observations` | the record form: two seeded series, `{item: $.host}`, and `expanded_to` in **name** order — `zone-a` before `Über-vm`, which is the opposite of the order their percent-encoded paths sort in (ADR-0044). Two further series are seeded where the Expansion may not reach them, one under another Definition and one under another Target the Definition accepts (§5, ADR-0012) |
| `a-predicate-reads-the-head-version` | a series whose **earlier** version matches the predicate and whose head does not: the Expansion resolves to nothing, `expanded_to` is written `[]`, and `RECORDS` renders `0` rather than the dash |
| `a-tombstoned-series-stands-for-nothing` | an `assets:` selector over three series — one Asset standing, one whose head is a Tombstone, one Observation — reaching the first alone |
| `a-relative-predicate-resolves-against-the-run` | `older_than: 14d` against the instant on `run.json`, over two series either side of it; **two Steps** carry the same predicate and reach the same member, the instant being the Run's and not each Step's |
| `a-predicate-list-does-not-short-circuit`, `-refuses-in-either-order` | the same two conjuncts written in both orders: the one that excludes the candidate first, then the one that cannot compare it. Both Refuse `predicate-type-mismatch`, so whether a Run Refuses does not depend on the order two conjuncts were written in (ADR-0035) |
| `a-stored-value-fills-an-integer-input` | a stored `"2592000"` filling an `integer` input — characters against the declared type, never the stored value's own JSON type — and the number reaching the wire, echoed back by the fixture |
| `a-stored-value-that-will-not-read` | the same wiring over a stored `"thirty"`: `schema-mismatch`, the code §4 fires where the value is on the page |
| `a-reference-resolving-to-nothing` | a member whose head carries no such field at all: the same code again, a reference resolving to nothing supplying no value |
| `two-members-one-identity`, `-json` | two members of one Expansion resolving to one identity under §7's fold: `record-identity-collision`, nothing touched, and the Step *refused* with its selector and no identity set |
| `an-identity-the-store-already-holds` | the second comparand: an identity that folds onto a standing series, refused with the same code |
| `the-sibling-collision-is-named-first` | both comparands available at once — the sibling is named, being reproducible from the artefact alone with no Store in hand |
| `a-step-with-no-selector-meets-the-store` | the Store comparand reaching a Step carrying no `over:`: vacuous against itself, and not against the branch |
| `an-expansion-past-its-bound`, `-json` | the Bound's run-time half (issue #149): three seeded Assets under a `bound: 2`, `bound-exceeded` at `77`, and the Refusal carrying `declared` against `observed` — on `outcome.json` and on the `refusal` row, and on no other code anywhere. **No call goes out**, and the case serves `api.cloudflare.com` so that it is the Bound and not a refused connection that stopped it: had a call gone out it would have been answered, and the branch would hold a version it does not |
| `a-bound-past-a-relative-predicate`, `-json` | §8's Refusal in full (issue #169): the Step table with the refused Step's cell `–`, the caret excerpt at the `bound:` with the `over:` block above it in context, the `=` note glossing `older_than: 14d` against **this Run's start**, `EDIT ONE OF` carrying both remediations — the Bound raised to `≥ 4`, and the selector narrowed a rung and speculatively re-expanded to `1` — the sentence that names no count, and a terminal line whose `show --expansion` pointer is earned by the four members the caret reported and did not name. On the wire it is one `refusal` row, two `remediation` rows, and `resolved` on the two that render an operand |
| `a-destroy-step-tombstones-an-asset` | the `destroy` end to end (issue #150): one seeded Asset, an `assets:` selector, a `204`, and a Tombstone carrying `tombstone: true` and the previous Head's `fields` copied forward under the Operation that destroyed it — *ran*, `RECORDS 1`, exit `0`, and **no** `answered`, the call having given the ordinary answer |
| `a-destroy-answered-a-404-is-still-gone` | the same case answered `404`: the `destroy` **completes**, and its Tombstone is **byte-identical** to the one above — same Run id, same clock, same bytes. What tells the two apart is the Step file's `answered` and nothing else, which is [`../../run_destroy_test.go`](../../run_destroy_test.go)'s |
| `a-destroy-expansion-is-serial` | five seeded Assets and five Tombstones under an Operation that may not declare a `concurrency:` at all. Its page says only that five landed; that **one** connection stood at a time is [`../../run_destroy_test.go`](../../run_destroy_test.go)'s |
| `a-destroy-halted-at-the-fourth-of-five`, `-json` | the halt the whole shape exists for: the fourth member is answered `500`, so three Tombstones are on the branch, two Assets stand, the Step is ***ran*** at `3 of 5` naming no member, `expanded_to` holds all five in Expansion order, the identity set beside it holds the three sorted, and the Run is `failed` at `1`. That the **fifth member is never called** is the drivers' |
| `a-re-run-reaches-what-the-halt-left` | ADR-0011's *the next Run reads exactly that*, driven over the branch the case above left: the three Tombstoned series stand for nothing, the Expansion resolves the two survivors in the same relative order the halted Run reached them in, and the Run completes at `0`. That the order is the Record `name` by code point rather than the percent-encoded path is `an-expansion-over-observations`', over the one function both selector forms sort in (ADR-0044) |
| `a-destroy-then-a-create-reads-alive-again` | a Tombstone is terminal for the Asset's life and not for the series: `destroy` then `mutate` over one identity, three versions, and a Head that reads alive again |
| `a-destroy-past-its-bound` | the Bound counted against a `destroy`'s Expansion: five seeded Assets under a `bound: 2`, `bound-exceeded` at `77`, and no call out — the case serves the host, so what stopped it is the Bound |
| `a-destroy-by-literal-opens-the-series-it-ends` | ADR-0033 end to end (issue #151): a `destroy` over a `values:` list naming two resources the Store holds nothing for, and two Tombstones each opening the series it ends. Neither carries a `fields` key at all — there was no previous Head to copy forward, and the absence means *`hyper` destroyed this and never observed what it was*. That it needs no second marker is [`../../run_literal_destroy_test.go`](../../run_literal_destroy_test.go)'s |
| `a-literal-that-matches-a-standing-series` | the same Procedure over a branch that already holds one of the two: no branch on whether a series was there, so one Tombstone opens a series and the other is an ordinary further version carrying the previous Head's `fields`, both under one Run and one Step. The two are held against each other member by member in the same driver |
| `a-values-member-the-store-already-ended-is-dropped` | *the Store shortens a `destroy`'s list and never lengthens it* (§5): three literals, the middle one already Tombstoned, and `expanded_to` holds the first and the third — three authored, two expanded to, one already gone, readable off the entry beside `declared` and in the authored order the survivors keep |
| `a-mutate-reaches-the-member-a-destroy-drops` | the other half of the same sentence, over the same seeded branch: a `mutate` over the Tombstoned literal is a call the artefact *is* asking for, so it is reached and the version it writes puts the Head above the Tombstone, alive again |
| `a-literal-that-folds-onto-a-standing-series` | the price ADR-0033 names, caught where it can be: a member authored `Preview-42.example.com` against a standing `preview-42.example.com` survives the head lookup — the two names are not equal — and the Store comparand Refuses `record-identity-collision` at the Expansion, naming both spellings verbatim, with nothing touched and no call out |
| `an-opaque-destroy-runs-over-a-values-list` | §6's own worked example run: `rm -rf` over two authored paths, one Tombstone each named by the path a human wrote down, `expanded_to` in Expansion order, and no Bound anywhere claiming to have guarded it |
| `a-step-with-no-selector-skips-what-stands` | `skip-if-recorded` end to end (issue #152): a Step carrying no `over:` runs the test over the one series it would write, its head stands, and the Step is ***skipped as already recorded*** at `RECORDS 1`, exit `0`. **No call goes out** — the case serves an answer carrying a different `id`, so a call that went out would have minted a second version the branch does not hold |
| `a-tombstoned-member-is-created-again` | the same Step over a series whose head is a **Tombstone**: it runs, the series standing for nothing, and the version it writes puts the Head above the Tombstone. Create, destroy, and create again is three Runs that each do what they say (ADR-0011) |
| `a-values-list-skips-two-and-calls-for-one` | the test decided per Record: three members, two standing and one naming no series at all. Two skip, one calls, the Step is ***ran***, `expanded_to` holds all three in the authored order, and the identity set holds all three — nothing is dropped for standing |
| `every-member-already-recorded` | the same Procedure over a branch holding all three: every member skips, the Step is ***skipped as already recorded*** at `RECORDS 3`, and its identity **digest is byte-identical** to the case above's — the two Dispositions being one set at two granularities (ADR-0056). No version is minted anywhere |
| `a-run-whose-every-step-skipped` | two `skip-if-recorded` Steps over four standing series: both *skipped as already recorded*, `completed`, exit `0`. Nothing in the outcome or the exit code distinguishes it from a Run that did all the work |
| `the-skip-test-does-not-read-the-journal` | a branch whose Journal records this very Step as *ran* over this very Record, and whose `records/` holds nothing. The Step **runs**: the test reads the Store's head version and never the Journal, so unlike run-once it consumes no Disposition (§6) |
| `a-skip-then-a-request-that-never-left` | the value the skip must not defeat: the first member skips, the second member's request **provably never left**, and no call this Step made reached the world — so it is ***attempted, world untouched*** with no identity set and `–` in `RECORDS`, at exit `1`. A Step that concluded about something without calling has still touched nothing, which is what keeps *world untouched* literally true (ADR-0062) |
| `a-shell-step-skips-the-command-it-recorded` | the `$.command` arm: the built-in `mutate_skip_if_recorded` over two authored paths, each Record named by the argv that made it. Driven once here and twice by [`../../run_skip_if_recorded_test.go`](../../run_skip_if_recorded_test.go), where the second Run skips both — the name the skip test reads and the name the projection writes being one string |
| `three-runs-of-one-values-list` | driven once here and three times by [`../../run_skip_if_recorded_test.go`](../../run_skip_if_recorded_test.go) |
| `two-runs-of-one-run-once-step` | run-once end to end (issue #153): driven once here — the first Run, which **runs**, the Journal holding no evidence of the Step — and twice by [`../../run_run_once_test.go`](../../run_run_once_test.go), where the second Refuses on what the first wrote. The case serves a **second answer carrying a different `id`**, so a second call that went out would mint a second version the branch does not hold |
| `a-second-run-refuses-run-once` | the Refusal on the page and in the entry: a seeded entry records the Step as *ran*, so the Run Refuses `run-once-recorded` at `77` with **no call out**, the Step is *refused* carrying no identity set and no selector — its Expansion never resolved — and the Step after it is *never reached* and **writes no file**. The Refusal is on `outcome.json` with exactly one member, a Refusal being terminal |
| `an-attempt-with-an-unknown-outcome-refuses` | the second of the two Dispositions that are evidence: the call went out and no answer came back, so a later Run may not treat it as either success or failure — and repeating the effect is the reading run-once declines (ADR-0018) |
| `a-step-the-halt-never-reached-runs` | the exclusion the value would be unusable without: a seeded entry whose first Step *ran* and whose run-once Step has **no file at all**, which is *never reached*. The re-run reaches it and the Run completes — without which one run-once Step would make a whole Procedure permanently un-re-runnable after any halt, with no bypass and nothing but an artefact edit left (ADR-0001) |
| `a-request-that-never-left-is-no-evidence` | the other exclusion, and the one stated rather than left to the list: the request provably never left, so nothing happened that a later Run could be evidence of. A firewall that lapsed for ten minutes leaves every artefact correct and nothing to edit (ADR-0062) |
| `neither-skip-is-evidence` | two seeded entries, one recording the Step as *skipped by condition* and one as *skipped as already recorded*: the Step **runs**. The first ran no Repeatability test at all; the second is `skip-if-recorded`'s finding, which is a test of the Store's head version rather than of anything a Run did (ADR-0056). The two values are one `repeatability:` key's alternatives, so the second reaches a run-once Step only across a **Manifest that changed between the Runs** — which is the state this entry is, and it is a fact about the branch either way |
| `a-rehearsal-is-no-evidence` | a seeded entry marked `dry_run: true` recording the Step as *ran*: the Step **runs**. An entry a rehearsal wrote is evidence that a rehearsal happened and evidence of nothing else, and a reader that counted it would permanently refuse every run-once Step in the Procedure it rehearsed — the review aid disarming the tool (ADR-0001) |
| `an-id-that-moved-has-no-evidence` | the match, and the whole of it: a seeded entry records `published` as *ran* and the Procedure's Step is `publish`. An `id` that moved is a different Step, with no evidence behind it |
| `run-once-does-not-read-the-record` | the mirror of `the-skip-test-does-not-read-the-journal` one value over: a branch whose `records/` holds the very Asset this Step would write and whose Journal holds nothing at all. The Step **runs** and mints a second version — run-once reads the Journal and never the Store's head versions |
| `one-run-that-reaches-a-run-once-step-twice` | the walk reaches **this Run's own entry** like any other: a Procedure invoking the run-once Step's Procedure twice runs the first occurrence and Refuses the second, on the file the first wrote a moment earlier. The Refusal names this Run's own id, and the two Steps carry one `path` between them — told apart by the position each holds in the Run (§7, ADR-0055) |
| `a-run-once-destroy-refuses` | the same value on the other effectful Kind: a `destroy` over a `values:` list whose Step the Journal records as *ran*. It Refuses before its Expansion resolves, so its file carries **no `selector` block** at all — the Store never shortened the list, and nothing was counted against the `bound:` (§6, §12) |
| `a-condition-decides-before-run-once` | the order of the two, and the Journal is what makes it visible: the seeded entry records the guarded Step as *ran*, and the Step is ***skipped by condition*** rather than refused. A Step whose `when:` does not hold makes no call, so refusing one that was going to be skipped anyway would end the Run over an effect nobody was about to repeat |
| `a-rehearsal-then-the-real-run` | driven once here — the rehearsal, which is the Run this golden holds — and twice by [`../../run_run_once_test.go`](../../run_run_once_test.go), where the real Run after it **runs**. Since issue #155 the rehearsal **withholds** the run-once Step rather than performing it, so what the round trip now shows is that a rehearsal reaches no run-once Step at all: the filter `a-rehearsal-is-no-evidence` seeds by hand is what catches an entry this binary did not write |
| `four-runs-of-one-step` | driven once here and four times by [`../../run_expansion_test.go`](../../run_expansion_test.go) |
| `a-procedure-matching-nothing`, `a-definition-rather-than-a-procedure`, `two-positionals`, `a-target-flag` | the four usage errors, all `2`, all with stdout completely silent |
| `a-store-file-this-binary-cannot-read` | the first gate past `run.json`: a Record head written at schema version 2, `store-schema-unsupported`, and the one Refusal that cites a file with no line and no field |
| `check-refuses-the-run`, `-json` | `check` re-run in full: five codes across the five artefact kinds, one `refusal` row each, in `check`'s own order |
| `a-cadence-over-a-run-once-step` | §8's Refusal over a code the walk found and the citation came back from (issue #169): a Procedure declaring a Cadence that reaches a run-once Step through a nested invocation. The fault is the invoked Procedure's Manifest, and the caret sits on the **`cadence:` line** of the Procedure declaring the recurrence — the artefact whose author can act on it. It is the one code here whose file, line and message all come from a walk that ended somewhere else |
| `a-stale-projection` | §10's other static code reaching the same pre-flight (issue #179): a repository whose projection is current but for one line — the install step's URL, pointed at a mirror by hand — refuses `projection-stale` at `77`. It is the shape a runner actually meets, the file being *there and wrong* rather than absent, and it is where §8's rendering for the code is asserted whole: **no caret**, since the comparison is whole-file and the file is not one to edit; the coordinate as a note; and the remedy as a **command**, `hyper project`, rather than an `EDIT ONE OF` table |
| `a-malformed-cadence`, `-json` | §10's grammar closed, one command over (issue #174): a `cadence:` naming a day of the week rather than numbering it, `cadence-malformed` at the Run-start `check`, `77`, and the Refusal on the entry. It is here as well as under `check/` because the whole point of the closure is that an expression no executor's clock could read never reaches one — a rule that held for `check` and not for a Run would make the Run the way past it |
| `a-cyclic-procedure-refuses-the-run` | the invocation graph that closes on itself: `procedure-cycle` at the Run-start `check`, `77`, and the Refusal on the entry. A cycle is `check`'s to refuse — the engine's own arm for one is a precondition no Run reaches (§4, §6, issue #146) |
| `a-destroy-with-no-selector-refuses-the-run` | the `destroy` Step carrying no `over:`: `destroy-unscoped` at the Run-start `check`, `77`, and the Refusal on the entry. It is here rather than under `check/` alone because what issue #157 found was a Run — the call went out and the process died in the Store afterwards — so the case that proves it fixed has to be one that would have made the call (§4, §5, ADR-0085) |
| `a-working-tree-edited-since-check-passed` | the same gate driven the way an operator meets it — one `uncommitted/` line narrows `local`'s `kinds:`, and the Run refuses with the codes the edit earns |
| `a-credential-the-environment-does-not-hold` | one absent slot, `credential-absent`, citing the `env:` line of the declaration whose slot the environment did not fill |
| `every-absent-credential-at-once`, `-json` | three absent slots across two Targets in **one** Refusal, and the two slots `paid` carries that no Step of this Run could send are not among them |
| `one-slot-two-definitions` | two Definitions binding one Target under one scheme require **one** slot between them, so an absent variable earns one member of the array and not one per binding |
| `a-header-scheme-reaches-the-wire` | the `header:` scheme end to end: the Manifest's `name:` and `prefix:`, the variable the Target declaration names, and what arrived at the far end |
| `a-basic-scheme-reaches-the-wire` | the same for `basic:`, whose position and base64 composition are the scheme's and never a Manifest's |
| `a-secret-sink-names-every-step`, `-json` | the sink gate: two Steps declaring secret output, both named at once, neither of them run |
| `usage-secret-out-to-stdout`, `-inside-the-repository`, `-with-no-path` | the three things `--secret-out` will not take, all `2` and all carrying no `error_code` |
| `a-member-that-reaches-the-deadline`, `-json` | the drain (issue #140): three members, the middle one reaching the Operation's `deadline:`, **every** member attempted, the two Observations that succeeded committed, the Step *ran* with the set it concluded about, `RECORDS` reading `2 of 3` and naming no member, and the `step` row carrying `expanded` beside `records` |
| `a-field-that-went-quiet` | a recorded field's path resolving to nothing is an **absence** and not a fault: the seeded head carries `note`, the answer does not, and the Run mints a second version **without** it and completes at `0` — the bytes moved, so the field going quiet renders as a change like any other |
| `a-collection-path-that-does-not-resolve` | the other half of that distinction: an Operation of `series` cardinality whose `over:` names a path the response does not carry. The Run halts at `1`, the Step is *ran* with an empty set, and its file carries `projection_failed_path` — without it `hyper` could not tell a collection that was empty from a path that was wrong |
| `a-series-whose-tenth-member-has-no-identity`, `-json` | the half-projected response: ten Records out of one, the tenth carrying no `id`. The nine that projected are **written**, the tenth is not, `RECORDS` reads `9 of 10` and the `step` row carries `expanded` beside `records` — the entry says expanded to one, the column counts the Records the answer reached |
| `a-projection-failure-drains-the-expansion` | the same failure inside an Expansion of three: every member attempted, the two that projected committed, the Step *ran* at `2 of 3`, and `projection_failed_path` naming the identity path. It is the deadline case's shape with `hyper`'s reading of an answer in place of an answer that never came |
| `two-members-one-identity-after-the-call` | the sibling comparand where the identity reads from the **response**: two members resolving `Crate` and `crate`. There is no Refusal available — the calls have gone out — so the Run **halts**, carries no `error_code`, names both spellings verbatim, and the first member in Expansion order keeps the identity |
| `two-records-of-one-response-one-identity` | the comparand no pre-call form reaches at all: two Records out of one `series` response, decided by the collection's own order. The colliding Record is written under no name and the other two are, at `2 of 3` |
| `an-identity-the-store-holds-after-the-call` | the third comparand, and `an-identity-the-store-already-holds`'s other half: the branch holds `Crate`, the answer resolves `crate`, and there is nothing to decide — the standing series was written by an earlier Run. `0 of 1`, nothing written, no `error_code` |
| `a-series-under-an-identity-that-fills-before-the-call` | the collision no pre-call pass could ever see: an Operation of `series` cardinality whose `identity:` is a template hole, so every Record of one response resolves the one name that hole filled to. The Expansion holds one identity per **member** (ADR-0070) and this one member holds two Records, so the halt is the only place it can be caught — `1 of 2`, the first Record written and the second under no name at all |
| `eight-members-under-a-limit-of-four` | an Expansion of eight under `concurrency: 4`. Its page says only that eight Observations landed; what the limit did is [`../../run_concurrency_test.go`](../../run_concurrency_test.go)'s |
| `an-expansion-with-no-declared-limit` | the same over an Operation declaring no `concurrency:`, which is the serial half of ADR-0045 |
| `two-read-steps-do-not-overlap` | two `read` Steps of two members each, both under `concurrency: 2` — all concurrency lives inside one Step's Expansion (ADR-0002) |
| `a-step-with-no-selector-under-a-limit` | a Step carrying no `over:` bound to an Operation declaring `concurrency: 4`: one call, which is a set of one and inside any limit ever written |
| `a-shell-step-records-what-a-command-printed` | the `shell` tracer bullet: the argv exec'd, the four-member object projected, and one Observation named by the command it ran |
| `a-command-that-exited-non-zero` | a `read` never halts on an exit code: `3` is recorded, the Step is *ran*, and the Run completes at `0` |
| `a-command-that-could-not-be-started` | the binary the Procedure names is not in the case's `bin/`: the object is `command` alone, the Observation carries **no fields at all**, and the Step is still *ran* — the attempt is the answer (ADR-0084) |
| `a-command-answering-in-json-is-recorded-as-text` | stdout is never parsed: what lands in the version is the string the command printed, braces and all |
| `an-argv-words-reference-is-checked-offline` | every argv word after the first is referenceable, and one naming a field the Record does not carry is `reference-unresolvable` at `77` — decided by `check` with no Store and no process, and cited down to `steps[1].args.command[1].path`. It carries no `bin/`: nothing is exec'd, the Run ending before its first Step |
| `an-argv-is-not-a-shell` | a pipe, a glob, an `&&`, a `$HOME` and a `>` reach the process as literal argv words, there being no shell between the artefact and it (ADR-0051) |
| `two-argv-spellings-are-two-series` | `[words, "a b"]` and `[words, a, b]` write two Record series, which is what the JSON encoding of `command` is injective for |
| `two-steps-running-one-argv-write-two-versions` | one argv, two Steps, one Definition and Target: two versions of one series, the command answering differently the second time |
| `the-child-stands-in-the-repository-root` | `cwd` is fixed and unauthorable: the command finds `hyper.yaml` and `procedures/` beside it |
| `the-child-inherits-no-credential-slot` | the environment less every credential-slot variable **in the repository**: the case sets both, the command prints both, and the one a Target declaration names — a declaration no Step of this Run binds — reads `<unset>` |
| `an-expansion-of-shell-steps-is-serial` | an Expansion of three over an Operation declaring no `concurrency:`: each member appends to a file the next one reads, so the three stdouts accumulate in Expansion order — which is serial dispatch shown rather than described |
| `usage-no-concurrency-flag` | there is no `--concurrency`: `2`, an unknown flag, and stdout silent. How much of an Expansion runs at once is a Manifest's and nobody else's |
| `a-cursor-walks-three-pages` | pagination's `cursor:` form (issue #143): three pages, six Records, and each Record carrying the query the server saw — nothing on page one, `cursor=c2` on page two, `cursor=c3` on page three. The walk ends where the third page hands no token back, and the Step file's `pattern` block reads `pages: 3` |
| `a-page-number-walks-until-the-collection-is-empty` | the other form and the other position: an integer `hyper` increments from `1`, written into a `header:`, and a **fourth** page whose collection comes back empty — which is the terminator both forms share. Six Records and `pages: 4`, the empty page having been fetched and being part of what `hyper` did |
| `a-poll-stops-when-its-until-holds` | a `polling:` Pattern over a host that answers `pending` and then `ready`: two calls an `interval:` apart, one Observation, and `polls: 2` |
| `a-poll-is-bounded-by-the-deadline` | a host that answers `pending` for ever under an Operation declaring `deadline: 1s` beside its `interval: 1s`: the poll is bounded by the deadline and by nothing else, and the Run halts at `1`. Its Step file carries **no** `pattern` block — one poll answered is the trivial single call, and the call the deadline cut off produced no observation of state to count |
| `an-until-that-cannot-compare` | the same Procedure with `state` served as a number: the Run **halts** rather than Refusing, exit `1`, no `error_code` anywhere, and stderr naming the field and what was found in it (ADR-0035, ADR-0072) |
| `a-retry-follows-a-refused-connection` | the host refuses its first two connections and answers the third: the Observation carries a `status`, and the Step file reads `attempts: 3` |
| `a-retry-follows-a-name-and-a-handshake` | the class's other two members, driven through a Run rather than only at the Capability: the host's first connection fails to resolve, its second fails its handshake, and its third answers. All three are one class because each provably precedes the request (ADR-0018), and `attempts: 3` says the Pattern followed both |
| `the-same-host-with-no-retry-declared` | the same host under the Operation that differs in one key: no retry, so the first refusal is the answer and the Observation records the silence. Its identity set is **byte-identical** to the case above's, which is what [`../../run_pattern_test.go`](../../run_pattern_test.go) holds |
| `an-exhausted-retry-records-the-silence` | three attempts against a host that is never there: the object is `host` alone, the Observation records it, the Step is *ran* and the Run completes at `0` — an exhausted retry leaves the response object for the projection to read |
| `no-status-is-ever-retried` | a host that answers `503` and then `200`, under an Operation declaring `retry: {attempts: 3}`: the `503` is recorded and the second answer is never asked for. The Step file carries **no** `pattern` block, one attempt and no retry declared being the same silence (§7) |
| `a-deadline-is-not-retried` | the other exclusion: a host that hangs, under the same Operation. The deadline halts the Step at `1` and no second attempt is made — a connect timeout is outside ADR-0018's class, and the Operation's own deadline is `hyper` stopping |
| `four-paginated-members-under-a-limit-of-four` | an Expansion of four under `concurrency: 4`, each member walking three pages: twelve Observations and `pages: 12`. What the limit reached and what it did not is [`../../run_pattern_test.go`](../../run_pattern_test.go)'s |
| `a-rehearsal-performs-the-reads-it-reaches`, `-json` | `--dry-run` over a Procedure of two `read` Steps: every read it reaches is performed, both Observations are recorded as ordinary versions carrying **no** marker of their own, the entry carries `dry_run: true`, the terminal line reads `completed · dry-run · exit 0`, and the `outcome` row carries the marker on the wire |
| `a-rehearsal-refuses-the-sink-it-was-not-given` | the sink gate carries no `--dry-run` exemption: a rehearsal reaching two Steps declaring `secret:` output with no `--secret-out` Refuses `secret-sink-absent` at `77` — the marker is on the line and on the entry, and a rehearsal that Refuses is not a rehearsal that completes (§9, issue #137) |
| `a-rehearsal-stops-at-the-first-effect`, `-json` | `--dry-run` over `read`, `mutate`, `read` (issue #155): the `read` in front of the effect **ran** and its Observation is on the branch, the `mutate` is *never reached* and so is the `read` behind it, neither writes a Step file, the page names the withheld Step and says the Run stopped, and the outcome is `completed` at `0`. The `-json` half is the same Run's rows: three `step` rows, the Run-wide `provenance` and the one Step's, and the `outcome` row carrying `dry_run: true` |
| `a-rehearsal-withholds-a-destroy` | the same stop on the other effectful Kind, against a branch seeded with the Asset the selector would have reached: the `destroy` is withheld before its Expansion resolves, and the branch holds that Asset's one version and **no Tombstone** |
| `a-rehearsal-stops-before-the-condition`, `a-guarded-effect-a-real-run-skips` | **the Kind is the whole of the test**, driven from both sides over one Procedure whose `mutate` carries a `when:` that does not hold. The rehearsal stops at that `mutate` all the same, so the `read` behind it is *never reached*; the same Procedure without the flag skips the `mutate` **by condition** and reaches that `read`. Deciding the `when:` here would be `hyper` reporting that a `mutate` *would* have been skipped, which is the prospective rendering ADR-0010 declines |
| `a-rehearsal-withholds-a-nested-step` | the withheld Step reached through an invocation: the page names it `publish-inner.publish`, under the path the table renders it at, so the sentence and the row point at one Step rather than at two spellings of one (§8) |
| `a-rehearsal-then-the-real-run` | the same stop where the effectful Step is the **first** Step: no Step file is written at all, the entry is `run.json` and `outcome.json`, and no Asset lands. It is listed under the run-once cases above, which is the other half of what it drives |
| `an-open-entry-is-left-open` | the branch is seeded with **another** Run's entry holding no account at all — no `outcome.json` it wrote, no `closed-by/` anybody wrote. This Run is **read-only**, so it reads that branch, completes at `0`, and leaves the entry exactly as open as it found it: a read-only Run holds the shared lock and can find a live effectful Run's entry open with no way to tell it from an abandoned one, so it reads and never reaps (§6) |

### The reap (issue #154)

Every case here is an **effectful** Run against a seeded Journal, and what it
asserts is the closing write that rides out beside its own `run.json`. The pair
no case can hold — a Run went quiet, and the next effectful Run closed what it
left — is [`../../run_reap_test.go`](../../run_reap_test.go)'s, which drives
real Runs at both ends.

| Case | What it is about |
| --- | --- |
| `an-effectful-run-reaps-an-open-entry` | the whole of it: a seeded entry holding `run.json` and one Step file, closed by a `closed-by/<this-run-id>.json` carrying `schema_version`, `ended_at`, `step` and `disposition` — `attempted-outcome-unknown` and no other value — beside the `id` and the code facts the dead Run's `repo_revision` resolves. The Step is the highest ordinal present **plus one**, derived by loading that Run's Procedure at that revision, so the entry reads `publish-again` and not `publish`. Nothing the dead Run wrote is touched, and neither an `outcome.json` nor a file under `steps/` appears |
| `an-open-entry-with-no-step-file-is-closed-at-step-one` | the same entry with its Step file taken away: a Run killed before its first Step finished went quiet on Step `1`, which is the same arithmetic rather than a case in the code |
| `every-open-entry-is-reaped` | three seeded entries — two open, one under the **day before**, and one its own Run closed. Both open entries are closed and the third is left exactly as it was: the reap is every open entry it finds and not a subset, and no criterion of age or liveness appears anywhere (ADR-0076) |
| `a-reaped-entry-whose-run-was-dirty-carries-no-code-facts` | the dead Run recorded `repo_dirty`, so the commit it named is not the code it ran: the closing write carries `step` and `disposition` and **no** `id` and no code facts. §7 names this one outright — *absent where it does not, which is every Run that recorded `repo_dirty`* |
| `a-reaped-entry-at-a-revision-this-clone-lacks-carries-no-code-facts` | the other way the same absence arises, and the one a runner produces: the entry names a commit this clone has never fetched. The reaper omits what it cannot establish rather than guessing at it, and the file it writes is byte-identical to the one above |
| `a-journal-file-this-binary-cannot-read` | the one place the reap and a gate meet: the seeded entry's own `run.json` is at schema version `2`, so the reap reads nothing it could act on and closes nothing — and the Run Refuses `store-schema-unsupported` at `77` one gate later, naming the file. §6 quantifies that gate over the Journal whole, so the condition is reported where it is reported for a read-only Run and the entry is left exactly as open as it was |
| `a-refused-run-has-already-reaped` | the same seeded entry under a Run whose credential is absent: it Refuses `credential-absent` at `77` **having already closed the entry**, which is the same reason §6 puts `run.json` before the gates at all |

Two further directories under `run/` hold no `argv` and are driven by name from
[`../../run_reap_test.go`](../../run_reap_test.go): `two-runs-and-a-kill-between-them`,
which freezes the branch mid-Run and drives the next effectful Run against it,
and `a-reap-that-was-wrong-is-contested`, which overlaps two effectful Runs on
two clones of one repository and reads the contest off the remote they both
pushed to.

### The `shell` Capability's effectful half (issue #156)

An effectful `shell` Operation completes on **`0`** and halts on everything
else, and the `404` has no counterpart here: an exit code is the command's own
vocabulary, so no value completes a `destroy` that would not complete a
`mutate`. The trap the `404` exists to avoid is closed by the `over:` selector
instead, which is the second case below.

Nothing about the Capability moves in these — the process group, the deadline,
the repository root, the empty stdin and the credential-stripped environment are
[`repo-shell`](repo-shell)'s cases' and unchanged. What is new is the Kind
semantics on top.

| Case | What it is about |
| --- | --- |
| `a-shell-mutate-lands-an-asset` | the effectful spine under this Capability: a `mutate` runs a command, one version lands with `record_type: asset` named by the argv that made it, the Step is *ran* and the Run completes at `0` |
| `a-shell-mutate-that-exited-non-zero` | the halt: `1` is not `0`, so the Step is ***ran*** at `0 of 1`, carries **no `error_code`** — nothing declined — and its `answered` holds the command and the exit code, which is §7's own worked example |
| `a-shell-mutate-whose-child-never-started` | the argv names a binary the case's `bin/` does not hold: the object is `command` alone, so the Step is ***attempted, world untouched*** with no identity set and `–` in `RECORDS`, and its `answered` is the command with **no `exit_code` beside it**. It is a request that never left under a different Capability and carries the same Disposition (ADR-0062) |
| `a-shell-destroy-halted-at-the-second-of-two` | `rm -rf` over two paths where the second exits `1`: the first Tombstone is committed, the Step is *ran* at `1 of 2`, and the Run halts at `1`. **No exit code completes a `destroy`** — there is no `404` to be told *already gone* by, and what the `answered` names is the member that ended the Step rather than the one that succeeded |
| `a-shell-destroy-drops-what-it-already-ended` | the other half of that sentence, and the mechanism that stands in for the `404`: the same Procedure over a branch already holding a Tombstone for the first path. It is dropped at Expansion, `expanded_to` holds the second alone, and **no command goes out for it** — the case's `bin/rm` exits `3` on that path, so a call that went out would halt the Run |
| `a-shell-mutate-once-refuses-a-second-run` | run-once under this Capability: a seeded entry records the `mutate_once` Step as *ran*, so the Run Refuses `run-once-recorded` at `77` with no command out |
| `a-shell-destroy-once-refuses-a-second-run` | the same on the other effectful Kind, over a `destroy_once` Step whose Expansion never resolves |
| `a-shell-skip-runs-the-member-a-tombstone-ended` | `skip-if-recorded` decided per Record where the identity is `$.command`: two members, one whose series stands and one whose head is a **Tombstone**. The first skips and the second runs, the identity set holds both, and the case's `bin/mark` exits `3` on the standing one so that *no call went out* is an assertion. Its sibling [`a-shell-step-skips-the-command-it-recorded`](a-shell-step-skips-the-command-it-recorded) drives the head that stands |
| `a-shell-mutate-answering-in-json-is-recorded-as-text` | ADR-0052 on an effectful Kind: the command prints an object and what lands in the Asset is the string it printed, braces and all. `$.stdout.status` is not a path — parsing a command's output would be `hyper` describing what it cannot describe |
| `a-destroy-expansion-of-shell-steps-is-serial` | the same on the Kind where it is worth the most, and it cannot be shown through stdout — a `destroy` projects nothing. So the **outcome** is the evidence: each member's `rm` reads back the path the member before it removed and exits `1` unless it is the one the authored order says, so three Tombstones and exit `0` are only reachable one member at a time and in that order |
| `an-expansion-of-shell-mutates-is-serial` | serial dispatch on an effectful Kind, shown rather than described: three members each appending to a file the next one reads, so the three stdouts accumulate in Expansion order. An effectful Step does not consult `concurrency:` at all (ADR-0045) |
| `two-shell-mutate-steps-land-two-assets` | two `mutate` Steps and one Asset each. Driven once here and once by [`../../run_signal_test.go`](../../run_signal_test.go), where the first interrupt lands on the first child: that Step finishes and is *ran*, the Step after it never starts, and the Run is `failed` at `130` — the child being in a process group of its own is what makes the drain true |

What no case here can drive is the **deadline**, for the reason stated below:
the built-in Provider declares one hour on every Operation and no repository may
edit it. That an effectful `shell` Step which reached one is *attempted, outcome
unknown* is [`../../../run/effect_test.go`](../../../run/effect_test.go)'s, and
the group kill it sends is
[`../../child_test.go`](../../child_test.go)'s.

## How a case reaches a binary, and what it costs

A `shell` Step's argv head is a name, and what a name resolves to is the
machine's `PATH` — which is the one thing a golden may not depend on. So the
harness supplies **name resolution** and nothing else, which is the arrangement
the dialer already has one Capability over: a case's argv head resolves against
that case's own `bin/` directory, and a case that holds none, or whose argv
names a binary its `bin/` does not hold, reaches nothing at all. That is what
[`a-command-that-could-not-be-started`](a-command-that-could-not-be-started)
drives, and its `bin/` deliberately holds a script under a *different* name, so
the case is a directory that exists and a binary that is not in it.

The argv itself is untouched. A `shell` Record is named by the argv as run
(§12), so a harness that rewrote the head into an absolute path under a temp
directory would put a value nobody can check in into every `store.golden`.

The **launcher is the real one** — `cli.Child`, the value the binary wires — so
the process group and the SIGKILL a deadline sends it are exercised by the same
code path a Run takes. What a golden cannot drive is the deadline: the built-in
`shell` Provider declares **one hour** on every Operation and no repository may
edit it, so the case that costs a second is not writable here. The group kill,
the grandchild that does not outlive it, and the response object under a
deadline are [`../../child_test.go`](../../child_test.go)'s and
[`../../../capability/shell_test.go`](../../../capability/shell_test.go)'s.

A case's `env` file is the child's whole environment, less the credential slots.
It carries no `PATH`, which is why every fixture script under a `bin/` uses
shell builtins alone: a script reaching for `cat` would be reaching for the
machine.

## What a Refusal's page looks like

Every Refusal that declined before Step 1 renders the same three blocks:
`nothing ran. no step was reached.` where the Step table would be, §8's Refusal
in full, and §8's terminal line. The middle block is the caret excerpt over the
offending line in its own context, the `=` notes carrying the phase and the
resolved instant of a relative predicate, and the `EDIT ONE OF` table — one row
per remediation, with a narrowed selector speculatively re-expanded beside the
widening (issue #169). It stood as `check`'s problem table for five milestones
and no longer does.

The Step table is omitted rather than rendered empty, on §8's own reading: an
empty table asserts *we looked at the Steps*, which is false. `stderr.golden`
is where that shows twice over — a refusing case narrates `run <id>` and no
`step` line at all, because no Step was reached.

That is the page of a Refusal that declined **before Step 1**, which is most of
the closed set. A Refusal at a Step's own Expansion — the identity comparands,
a predicate that cannot compare, a Bound the count is past — reached a Step, so
its page carries the Step table with that Step *refused* and §8's Refusal
beneath it, and its `stderr.golden` carries the `step` line. The sentence the
absence would have to carry is not needed there: the table says what became of
every Step.

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
the work (§12, ADR-0061, issues #138, #148). Two of the three sit above as
ordinary cases. The rest are in
[`../../run_store_lost_test.go`](../../run_store_lost_test.go), each for a
reason a golden cannot get past:

- **The lock** is not a directory of files. It is held by a *live* process —
  which is exactly why a crash cannot leave one behind — so the three cases that
  drive it take it in the test process and run the command against the same
  repository. Two hold the exclusive lock against a `read` Run; the third holds
  the **shared** one against an effectful Run, which is what proves the *mode*
  rather than the lock — a Run that contends whichever mode it asked for says
  nothing about which one it asked for.
- **The exhausted push** renders git's own account of the rejection, and that
  account names the bare repository by path: a temp directory, different on
  every run of the suite. Its streams are asserted by what they say; its two
  branch goldens, which name no path and no commit, are checked in beside
  [`a-push-rejected-three-times/`](a-push-rejected-three-times) and compared
  like any other case's.
- **An effectful Run's sync** is the same shape one moment earlier: the push of
  that Run's own `run.json` **is** the sync (§7, ADR-0083), so a Run that could
  not complete it lost the Store before it touched the world. Both halves render
  git's words and both are driven there —
  [`an-effectful-sync-that-could-not-reach-the-remote/`](an-effectful-sync-that-could-not-reach-the-remote),
  whose fetch URL points at nothing, and
  [`an-effectful-entry-that-did-not-reach-the-remote/`](an-effectful-entry-that-did-not-reach-the-remote),
  whose push is refused three times running. The first leaves no entry at all,
  the second an entry that stands locally and reached nothing, and the read-only
  half of the first is the ordinary golden
  [`a-read-only-sync-in-the-same-fixture`](a-read-only-sync-in-the-same-fixture),
  which tolerates the failure and completes.

**None of the three writes a terminal row where it has no id, and that is the
rule rather than an omission.** §8 puts `run` on the `outcome` side on every path
a Run was **attempted** on, and on the two that decline before one is identified
— the version pin gate and the bootstrap `store-absent` (issue #172). The lock
and the sync stand before `run.json`, so no Run was attempted on either and
neither declined: stdout is silent, and `run_id` stays absent exactly where §8
says it is. The push that could not land is the other way round — the Run ran,
so its terminal line names the id it wrote.

## How a case reaches the Operation's deadline

A `read` never halts on what came back, so the only thing that fails one short
of its projection is `hyper` stopping: the Operation's own `deadline:` (§6,
ADR-0050). Reaching it needs a host that accepts the connection and then answers
nothing, which is what `serve/<host>.json`'s **`hangs`** key is — the handler
holds the request open until the caller gives up on it, and the caller giving up
is the deadline an artefact declared.

It is a different fact from a host with no `serve/` entry at all, and the two
must not be read as one. A refused connection is *no response arrived*, which a
`read` records as the answer it is
([`a-host-that-answered-nothing`](a-host-that-answered-nothing), exit `0`); a
deadline fails the Step
([`a-member-that-reaches-the-deadline`](a-member-that-reaches-the-deadline),
exit `1`). `repo-drain`'s Operation declares `deadline: 1s`, which is the
smallest a duration can be written as short of the `0s` that would reach every
member at once — and the case costs that second, which is what the one fact
nothing else can drive is worth.

## How a case reaches a world that changed between two calls

The Patterns are the one thing in the tool whose subject is a world that answers
differently the second time, so `serve/` grew two keys for it (issue #143) —
both of them still what a *server* does, which is the only thing a fixture has
any business supplying.

**`answers`** is a list of what the host answers on successive requests, the
last one repeating once the list is exhausted. It is what a paginated read walks
and what a polled Operation waits on. It is deterministic because the thing it
serves is: all three Patterns are serial by construction, so a member is one
request at a time — and no case here drives two members through one host, which
would be depending on something nothing fixes.

**`refuse_first`** is how many connections to the host are refused before any is
accepted, and **`refuse_first_as`** is how each of them fails — `refused`,
`name` or `handshake`. Without the first, a case could show a retry exhausting
and never a retry succeeding: the refused connection a case already had refuses
for ever. Without the second, ADR-0018's three-member class would be asserted at
Run level by its smallest member alone, the other two reaching only
[`../../../capability/sent_test.go`](../../../capability/sent_test.go). Each is
answered in the shape that failure really has — the resolver's error, the TLS
stack's — because `hyper` establishes the class by **where** a failure happened
and never by reading it, and a fixture answering one error type three times
would check that with less than it could.

**What `hyper` put on the wire** is read back the way the credential cases read
theirs — by having the server say what it saw. A body may carry `<<query>>` or
`<<header:name>>`, which the fixture fills from the request it is answering, so
a pagination Pattern's token lands in a **projected field** and therefore in
`store.golden`. That is what makes *the cursor was written into the declared
`query:` position* a checked-in constant rather than an inference from the pages
arriving in order.

## What one golden cannot prove, and what reads all of them

**That `answered` is written on no `read` Step anywhere.** A per-case golden
says what that case wrote, and a hundred goldens that happen not to carry the
key say nothing about the hundred-and-first. So the rule is held over the whole
corpus instead: [`../../run_answered_test.go`](../../run_answered_test.go) walks
every branch golden under `testdata/`, decodes every Step file it finds, and
fails on a `read` Step carrying the key — and on a corpus holding no `read` Step
file, or no effectful one carrying it, either of which would be the rule passing
over nothing (§7, ADR-0010, issue #148).

**That the row stream is a contract and not a per-case accident.** §9 states
three rules over the *whole* stream and each is held over the whole corpus in
[`../../row_stream_test.go`](../../row_stream_test.go): that nothing is
abbreviated on the wire, with the catch-all row's `command` as the one stated
exception; that the page and the wire carry the same rows, over every case with
both a plain and a `--json` twin; and that every stream terminates — a Run in an
`outcome` row on every path it takes, a usage error in no stream at all. Each
fails once more where the corpus holds nothing for it to range over, which is
the failure a rule asserted over eighty-three pairs is written to catch (issue
#172).

**That `declared` and `observed` are written by `bound-exceeded` and by nothing
else.** The same shape of claim, one key over: §7 states the pair for a check
that compared two values and nothing is invented to fill a member that does not
apply, so what has to hold is a property of *every* Refusal the corpus ever
writes.
[`../../error_code_coverage_test.go`](../../error_code_coverage_test.go) walks
every branch golden and every `--json` stdout under `testdata/`, decodes each
`outcome.json` through the Store's own reader, and fails three ways — a code
that is not `bound-exceeded` carrying either member, `bound-exceeded` carrying
neither, and a member written without its pair — plus once more where no
Refusal in the corpus compares anything at all, which would be the rule passing
over nothing (§7, issue #149).

## What no golden here proves

**That a signal arrived while a Step was in flight.** A case directory is an
argv and a set of inputs, and *the interrupt landed while Step 1 was on the
wire* is a fact about **when** — the same thing the concurrency cases below
cannot state. So the delivery is driven instead:
[`../../run_signal_test.go`](../../run_signal_test.go) drives the two-Step cases
here with the tenth process read supplied by the case, hands the signal over
from inside Step 1's own call, and waits for the watch to release before
returning — so the Step in flight is genuinely in flight and the Run's next
boundary is guaranteed to see the drain. What it holds is §6's sentence entire:
the drained Step *ran* and its Record is on the branch, the Steps after it are
*never reached* and wrote no file, the entry is closed by its **own**
`outcome.json`, and the exit code is `130` for an interrupt and `143` for a
termination. It drives the one-Step case too, where there is nothing left to
withhold: the Run is `failed` at `130` all the same — §6 puts an interrupt in
`failed` beside an error and a deadline, so a Run somebody stopped may not
answer `0` however much of it finished.

**What a second interrupt leaves.** It kills the process outright, so there is
no code path to drive and nothing for `hyper` to write — what it leaves is
whatever the branch already held. The same file reads the branch from inside
Step 1's call, which is exactly the state a kill at that instant would freeze:
`run.json`, and no account beside it. That absence **is** the representation.

**That a `shell` Step's own dispatch is the `http` one.** There is one call site
and one limit — `dispatch(bound.detail.ConcurrencyLimit, …)` — reached by both
Capabilities, so *a shell Step honours `concurrency:` exactly as an http one
does* is true by construction rather than by a case.
[`an-expansion-of-shell-steps-is-serial`](an-expansion-of-shell-steps-is-serial)
shows the built-in's own end of it: it declares no `concurrency:`, so its
Expansion is serial, and the accumulating stdout is what serial looks like from
the branch. Its effectful counterpart
[`an-expansion-of-shell-mutates-is-serial`](an-expansion-of-shell-mutates-is-serial)
shows the arm that consults no limit at all, a Kind other than `read` dispatching
one member at a time whatever a Manifest declared (ADR-0045).

**How much of an Expansion ran at once.** A limit, a dispatch order and *two
Steps never overlap* are all facts about **when** calls happened, and a branch
and a page record only that they did: four members in flight and four members
one after another leave the same Observations, the same identity set and the
same table. So the wire is watched instead —
[`../../run_concurrency_test.go`](../../run_concurrency_test.go) drives the four
`repo-concurrency` cases with the harness's dialer wrapped, holds each
connection until as many as it expects are standing together, and reads the peak
off. It also drives one case twice with its connections let go last-first, and
holds the page and the branch byte-identical against the forward Run: **nothing
derives from the order a concurrent Expansion's calls complete in** (§6).

**That a concurrency limit does not bound a Pattern's own calls.** All three
Patterns are serial by construction, so a member is one call at a time from the
moment it is dispatched until its last page, and *members in flight* and
*requests in flight* are one number (ADR-0045). Twelve pages fetched four at a
time and twelve fetched one member at a time leave the same twelve Observations,
the same identity set and the same `pattern` block, so the dials are counted
instead: [`../../run_pattern_test.go`](../../run_pattern_test.go) drives
[`four-paginated-members-under-a-limit-of-four`](four-paginated-members-under-a-limit-of-four)
with each dial held long enough for one standing beside it to be seen, and holds
two numbers — four members together, and never two requests to one host.

**That the interval was waited between calls rather than before the first, and
that a fourth attempt was never made.** Both are facts about *when*, and both
are [`../../../run/pattern_test.go`](../../../run/pattern_test.go)'s for the
reason above. What the cases here hold is everything downstream: the Records
every page projected, the account the Step file carries, and the halt.

**That a retry under the `shell` Capability covers a child that could not be
started and nothing else.** No repository can write it: the Capability is
reserved to Providers `hyper` ships (ADR-0039), and the one it ships declares an
empty `patterns:` — pagination and polling having no meaning against a command,
and retry following only a failure that provably preceded a request (§13). So
the claim is structural instead. One function decides the class for both
Capabilities (`capability.NeverSent`), marking only the failures each performer
knows came before any byte left — the dialler answering nothing, and
`child.Start` failing — and
[`../../../capability/sent_test.go`](../../../capability/sent_test.go) holds
both sides of it, a non-zero exit and a `503` included.

**That a Tombstone opening a series is not a shape of its own, and that an
ordinary version carrying no `fields` is not one.** Both are claims about a key
that is **absent**, which is what a golden asserts least well: a key nobody
wrote looks exactly like a key that may not be there.
[`../../run_literal_destroy_test.go`](../../run_literal_destroy_test.go) reads
them off the branches two cases left. It holds the Tombstone that opened a
series against the one written over a standing Asset in the same Run and Step,
member by member, and fails on any third member differing — so *no second
marker* is a property of the pair rather than of one file. Then it decodes the
fieldless Tombstone beside the fieldless Observation that
`a-command-that-could-not-be-started` writes, and reads what the Store answers
when asked what each one is: it is the written marker and never the missing key
that identifies a Tombstone (§7, ADR-0033, ADR-0084).

**Which rhythm a Run pushes at.** A read-only Run batches to its end and an
effectful one pushes at every Step boundary, and both leave the remote holding
the same commits — so no branch golden can tell them apart. What tells them
apart is how many times the remote was reached, so that is counted instead:
[`../../run_push_test.go`](../../run_push_test.go) installs a receive hook on
the bare origin that accepts a push and tallies it, and drives two two-Step
Runs. [`two-read-steps-push-once`](two-read-steps-push-once) holds the tally at
one and
[`two-effectful-steps-push-three-times`](two-effectful-steps-push-three-times)
at three — the sync that is the push of `run.json`, then one per Step, the last
Step's going out with `outcome.json`.

**That an identity set is written as a digest alone where it did not move.**
What that is about is what the *second* entry writes given the first, and a case
drives one Run. [`../../run_expansion_test.go`](../../run_expansion_test.go)
drives [`four-runs-of-one-step`](four-runs-of-one-step) four times through one
materialised repository — narrowing the Step's `values:` list between the second
Run and the third — and reads the four Step files: members, digest alone,
members, digest alone.

**That a rehearsal leaves an entry the Run after it reads as no evidence.**
Same shape: `a-rehearsal-is-no-evidence` hand-writes the marker, and what no
case can drive is the round trip — `--dry-run`, then the same argv without it.
[`../../run_run_once_test.go`](../../run_run_once_test.go) drives
[`a-rehearsal-then-the-real-run`](a-rehearsal-then-the-real-run) both ways: the
rehearsal **withholds** the Step and writes no file for it, the real Run after
it *ran*, and the entries carry `dry_run: true` and then `false`. It is the
claim the exception to the absence rule is bought against (§7, ADR-0001).

Since issue #155 a rehearsal cannot record a run-once Step even once — run-once
is effectful-only and a rehearsal stops at the first effectful Step — so the
filter itself is reachable only from a seeded entry, which is what
`a-rehearsal-is-no-evidence` is. The round trip is what says the two ends meet
all the same: what the flag leaves behind is an entry the Run after it is not
refused by.

**That the Disposition a run-once Step writes is the one a later Run refuses
on.** Every seeded case here says what one Disposition *means* to the walk; none
of them says that a real Run writes one a real Run reads, and a case drives one
Run. [`../../run_run_once_test.go`](../../run_run_once_test.go) drives
[`two-runs-of-one-run-once-step`](two-runs-of-one-run-once-step) twice through
one materialised repository, editing nothing between them: the first *ran* at
`0` and the second Refuses `run-once-recorded` at `77`, its Refusal naming the
first Run's own id. That **no call went out** is the branch's to say — the case
serves a second answer carrying a different `id`, so a call would have minted a
version, and the branch holds one after both Runs.

**Three of the Expansion's rules are structural, and the corpus shows their
consequence rather than the rule.** They are named here so that a reader does
not look for a case that cannot exist.

- **A `values:` member the Store dropped** is present in `declared` and absent
  from `expanded_to`, and the drop is a `destroy`'s: §5 drops a member whose
  head is a Tombstone on a `destroy` Step and reaches one on a `mutate`. A
  `read` drops nothing, so most of the cases here show the two lists standing
  side by side and issue #151 is where they differ —
  [`a-values-member-the-store-already-ended-is-dropped`](a-values-member-the-store-already-ended-is-dropped)
  against
  [`a-mutate-reaches-the-member-a-destroy-drops`](a-mutate-reaches-the-member-a-destroy-drops),
  one seeded branch and two Kinds.
- **A Tombstone stands for nothing under either form**, and only an `assets:`
  selector can meet one: a Definition observes or effects and never both
  (ADR-0032), so an Observation series never holds a Tombstone and a fixture
  that seeded one would be evidence of a state the Store cannot reach.
- **The two identity checks run once over the resolved set**, before the first
  call. What a golden holds is the consequence — a Refusal with nothing written
  and no version minted — since one check per member and one check over the set
  Refuse the same way when the Refusal comes first either way.

## Why the halted Run halts on an identity path

`a-run-halted-by-its-step` binds an Operation whose `record:` reads its
`identity:` from `$.body.id`, and serves a body with no `id` in it. That is the
second way a `read` Step fails, beside the deadline above: a `read` never halts
on what came back, so what stops it is `hyper`'s reading of the answer rather
than the answer (§6, ADR-0050).

It lands on the drain rather than beside it (issue #140). Its one member is its
whole Expansion, so the Step concludes about nothing, is *ran* all the same, and
renders `0 of 1` — expanded to one and accounted for none. Its file carries
`projection_failed_path`, the member §7 puts the failed path on, and nothing
whatever of the response it failed against: a rendering goes to a terminal that
scrolls, and no surface shows the answer (ADR-0017).

Three faults share that shape and the `repo-projection` cases above drive each
of them.

- **The collection path** an Operation of `series` cardinality reads its Records
  from halts the same way, and it had to: both pagination forms terminate when
  that collection comes back empty, so conflating *there was nothing there* with
  *the path was wrong* would have made the Pattern's own terminator a lie (§6).
- **A recorded field's path is not one of them.** It resolving to nothing is an
  absence, and the version is written without the field — which is why
  `a-field-that-went-quiet` completes at `0` beside these.
- **A half-projected response writes what projected.** The response arrived and
  part of it read, so the nine Records that projected are written and the tenth
  is what the Run halts on (issue #144). The drain is untouched by that: a
  member whose call reached the Operation's deadline holds no answer at all and
  is skipped as it always was, which is what `a-member-that-reaches-the-deadline`
  and `a-projection-failure-drains-the-expansion` hold side by side.

## Why a collision after the call is a halt and not a Refusal

The Expansion's two identity comparands Refuse with nothing touched
(`two-members-one-identity`, `an-identity-the-store-already-holds`), and their
three neighbours here halt with the calls already made. Which of the two a
repository gets is decided by the Manifest and by nothing else: an `identity:`
that is a template hole fills from the resolved inputs before the call, and a
`$`-rooted one names a value that exists only once the answer is in hand
(ADR-0072). The two `repo-projection` Operations that read theirs from a
response are what put these cases on the halting side of that line.

**What each comparand supplies is an order**, and the three cases differ in
which one. Across an Expansion it is Expansion order — read off the drain rather
than off a completion order, a `read` Expansion running concurrently. Across one
`series` response it is the collection's own order, which the response states.
Against the Store there is nothing to decide, the standing series having been
written by an earlier Run.

That the rule is an order and not a race is not a thing a case can show: two
members that always answer in the same order would report the same winner
whichever decided it. [`../../../run/reading_test.go`](../../../run/reading_test.go)
holds the rule; the cases here hold the answer, the halt, and that both spellings
reach the report verbatim.

## Why `a-procedure-matching-nothing` has no repository

It names `../repo-watch-status` with the `--repo-dir` an operator would type,
and that directory is inside no git repository. That is the case: §9 fixes that
a positional resolves before the Store is located, so `hyper run typo` is `2` on
a repository with no Store at all — the typo is repaired before the Store is
missed.
