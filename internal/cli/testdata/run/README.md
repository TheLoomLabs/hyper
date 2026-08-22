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
| `a-repository-with-no-store` | `store-absent`, `77`, naming `hyper store init` — a Run never creates the branch |
| `the-runner-clone-fetches-the-store` | the runner shape: `hyper-store` on `origin` alone, brought down by the Run's own sync, and the Run proceeds normally |
| `a-sync-that-could-not-reach-the-remote` | the sync fails and the Run **tolerates it**, saying so on stderr, reading the branch the clone holds and completing at `0` — never `75` for a sync it could not complete |
| `a-sync-that-could-not-bring-a-branch` | the same failure with no branch in hand: the same stderr line, then `store-absent` at `77`, because what is missing is an act and not a network |
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
| `four-runs-of-one-step` | driven once here and four times by [`../../run_expansion_test.go`](../../run_expansion_test.go) |
| `a-procedure-matching-nothing`, `a-definition-rather-than-a-procedure`, `two-positionals`, `a-target-flag` | the four usage errors, all `2`, all with stdout completely silent |
| `a-store-file-this-binary-cannot-read` | the first gate past `run.json`: a Record head written at schema version 2, `store-schema-unsupported`, and the one Refusal that cites a file with no line and no field |
| `check-refuses-the-run`, `-json` | `check` re-run in full: five codes across the five artefact kinds, one `refusal` row each, in `check`'s own order |
| `a-cyclic-procedure-refuses-the-run` | the invocation graph that closes on itself: `procedure-cycle` at the Run-start `check`, `77`, and the Refusal on the entry. A cycle is `check`'s to refuse — the engine's own arm for one is a precondition no Run reaches (§4, §6, issue #146) |
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
| `an-open-entry-is-left-open` | the branch is seeded with **another** Run's entry holding no account at all — no `outcome.json` it wrote, no `closed-by/` anybody wrote. This Run reads that branch, completes at `0`, and leaves the entry exactly as open as it found it: nothing in this milestone reaps one, closes one, or infers anything from an absent account |

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

That is the page of a Refusal that declined **before Step 1**, which is most of
the closed set. A Refusal at a Step's own Expansion — the identity comparands,
a predicate that cannot compare, a Bound the count is past — reached a Step, so
its page carries the Step table with that Step *refused* and the problem table
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
the branch.

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

**Three of the Expansion's rules are structural, and the corpus shows their
consequence rather than the rule.** They are named here so that a reader does
not look for a case that cannot exist.

- **A `values:` member the Store dropped** is present in `declared` and absent
  from `expanded_to`, and the drop is a `destroy`'s: §5 drops a member whose
  head is a Tombstone on a `destroy` Step and reaches one on a `mutate`. A
  `read` drops nothing, so what the cases here show is the two lists standing
  side by side, and milestone 6 is where they differ.
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
