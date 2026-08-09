# §12 — The closed sets

Every closed set `hyper` defines lives here, defined once and in full. §3–§11
name members of these sets and cite this chapter; none restates a set it can
cite instead.

A set below is **closed** once writing the sections that draw on it stops
adding members to it, and stays **open** until then.

The process by which a set in this chapter grows — who adds a member, and
when — is undecided. It is not decided here.

## Kind

**Open.**

## Repeatability

Three values, declared per Operation in a Manifest and never inferred:

- `repeatable` — invoke it again. What the Operation does is unchanged by how many times it has already
  run: an absolute-state `mutate`, most reads.
- `skip-if-recorded` — skip while the Asset it would produce still stands. The test reads the head
  version of the Record series `(Target, Definition, name)` identifies, not the existence of that
  series, so a series whose head is a Tombstone runs again (ADR-0011).
- Undeclared — **run-once**. The Operation runs where the Journal holds no evidence it already ran, and
  Refuses where it does. This is the default, and it is the strict one: an effect nobody vouched for is
  not repeated on a guess.

## Disposition

Six values, one per Step of a Run:

- **ran** — the Step was invoked and its outcome came back.
- **skipped as already recorded** — the Step's Operation declared `skip-if-recorded` and the Asset it
  would produce still stands. The only value that is Repeatability evidence.
- **skipped by condition** — the Step's `when:` did not hold. Says nothing about what the world holds,
  which is why it is not the same value as the skip above.
- **refused** — a guardrail declined the Step before any effect reached the world (§5).
- **never reached** — the Run ended before the Step. A run-once Step in this state runs on a re-run.
- **attempted, outcome unknown** — the call went out and no answer came back. It attaches the
  uncertainty to the attempt rather than to the thing, so nothing downstream reads it as either
  success or failure.

Every Disposition also carries the Record identities the Step acted on and what `hyper` itself did to
reach that outcome — a Pattern's attempts, pages, and poll iterations — which is a `hyper`-owned record
no Provider supplies (ADR-0018).

## The outcome triple

Three terminal outcomes, exactly one per Run:

- `completed` — every Step reached a terminal Disposition and none of them refused or failed. A Run
  whose every Step skipped completed.
- `refused` — a guardrail declined a Step before any effect reached the world, and the Run stopped
  there (§5, ADR-0001).
- `failed` — the world resisted, or the Run was stopped: an error from a Step, a deadline, an interrupt,
  contention on the Store lock, or an open entry closed by a later Run.

There is no fourth outcome and no partial one; a Run that halted midway is `failed`, with what it
completed held by its Records and Dispositions rather than by its outcome.

## Exit codes

Seven members, one per way an invocation can end. No member ever spans two outcomes of the triple
above, and the set is finer than that triple rather than coarser: four of the seven are `failed`.

- `0` — the command did what it was asked. A Run that completed, including one whose every Step
  skipped (§6).
- `1` — a Run that failed because the world resisted, or a command that is not a Run reporting
  problems it found.
- `2` — a usage error. No Run began, and no member of the outcome triple applies.
- `75` — a Run that lost the Store: to the lock (§6), or to a push it could not rebase through in
  three attempts (§7). `failed`.
- `77` — a guardrail declined before any effect reached the world. A Run that refused (§5), and the
  code the version pin gate and an absent Store carry from any command that hits them (§9).
- `130` — a Run stopped by an interrupt, having drained (§6, ADR-0015). `failed`.
- `143` — a Run stopped by a termination signal, drained the same way (§9). `failed`.

`75` and `77` are `sysexits`' `EX_TEMPFAIL` and `EX_NOPERM`, and the pairing carries the difference
between them: `75` says retry me, and `77` says a verbatim retry will refuse identically (§9).

## Capabilities

**Open.**

## Auth schemes

**Open.**

## Patterns

**Open.**

## `error_code`

Members named so far, contributed by §4's static checks: `strict-yaml-violation`, `unknown-key`,
`kind-mismatch`, `schema-unsupported`, `credential-slot-malformed`, `hole-illegal`,
`series-reference`, `capability-mismatch`, `identity-undeclared`, `target-class-mismatch`,
`kind-not-granted`, `operation-not-claimed`, `envelope-exceeded`, `opaque-destroy-not-granted`,
`bound-missing`, `host-not-granted`. §6's two run-time checks carry `bound-exceeded`, an Expansion
resolving to more Records than the Step's declared Bound, and `run-once-recorded`, a run-once Step the
Journal already holds as *ran* or *attempted, outcome unknown*. §7's four are the Store's:
`store-absent`, a Run — or any other command that needs the Store (§9) — finding no Store branch;
`store-unsynced`, an effectful Run that could not sync before its first effect;
`record-identity-collision`, a Record identity colliding case-insensitively with one already written;
and `store-schema-unsupported`, a Store file whose schema version is above the reader's (ADR-0028). §9
contributes `credential-absent`, a credential a Target declaration names and the environment does not
hold, checked before a Run's first Step. Further members are named where §11 states the checks that
carry them.

## The path grammar

Two roots: a Manifest's projection reads from an Operation's response, and a Step's reference reads
from a Record. From either root, the grammar is `$`, `.member`, and `["member"]`, and nothing else.
Recursive descent (`..`) is rejected on the same ground as a YAML alias or a JSON Schema `$ref`: a path
whose meaning depends on data the reviewer cannot see while reviewing (ADR-0022). Array indexing
(`[n]`) is rejected because an index into an upstream array is the identity-is-array-position hazard
that a Manifest's declared Record identity exists to close. Iteration (`[*]` or any equivalent) is not
in the grammar at all: it is declared once, in an Operation's Record cardinality, and implied by a
cardinality of `series`.

## The Store path grammar

Every path the Store branch holds, and no other. Distinct from the path grammar above, which is the
one a Manifest projects with and a Step references with; this one is filenames.

| path | written |
| --- | --- |
| `STORE.md` | once, when the Store is created |
| `records/<target>/<definition>/<name>/<run-id>-<nnnn>.json` | one per Record version |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/run.json` | at Run start |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/steps/<nnnn>.json` | one per Step reaching a Disposition |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/outcome.json` | when the Run ends, or by the Run that closes it |

`<run-id>` is a UUIDv7, lowercase and hyphenated: time-ordered, and mintable by either environment
alone, which a counter is not (ADR-0006). `<yyyy>/<mm>/<dd>` is the UTC date of the Run's start.
`<nnnn>` is the Step's position in the Run's written order, nested invocations counted in that order,
zero-padded to four digits and widening beyond four rather than wrapping; it names a Record version as
well as a Step file, so two Steps of one Run writing one identity write two paths rather than one
twice.

`<target>`, `<definition>` and `<name>` are one path segment each, percent-encoded: every byte outside
`A`–`Z`, `a`–`z`, `0`–`9`, `-`, `_` and `.` is written as `%` and two uppercase hexadecimal digits over
its UTF-8 bytes, as is a leading `.`. Case is preserved rather than folded (§7). An encoded segment
longer than 200 bytes is cut at 200 on an escape boundary and suffixed with `~` and the first 16
hexadecimal digits of the SHA-256 of the whole encoded segment; `~` is outside the unreserved set
above, so it never occurs in an encoding and the suffix is unambiguous.

## The predicate operators

`equals`, `not_equals`, `in`, `exists`, `absent`, `starts_with`, `ends_with`, `greater_than`,
`less_than`, `older_than`, `newer_than`. A predicate list is always AND; there is no disjunction
(ADR-0022). Two scopes root the same operator set differently: a selector (`over:`) roots at the Record
being filtered, and a condition (`when:`) roots at a named earlier Step's Record and so carries `step:`
beside `field:`. There is no regular-expression operator — `starts_with` and `ends_with` are the
bounded form of prefix and suffix matching, in place of the unbounded one (ADR-0022).

## The template-hole positions

One hole syntax, three legal positions:

- A **Capability-relevant position** — a host, an Auth scheme's parameters, anything that determines
  what a request may reach — resolves only to a declared closed enumeration or to `from-target`, never
  to an Operation input and never to a bare wildcard. `hyper` expands the cross-product of every such
  hole against its enumeration at load and compares the derived finite set against the Target's grant
  (ADR-0024).
- Every other position resolves only to an Operation input.
- A position inside `auth:` resolves only to a declared credential slot, and a credential-slot hole is
  legal nowhere else.

A hole resolving to none of these, or one filled from the wrong source for its position, is a load
error.

## The `over:` forms

`assets:` and `observations:` expand over the Step's own Definition (ADR-0012) and Target's Record
series; `observations:` is legal only on a `read` Step, since Expansion is scoped by Kind rather than
by Record type (ADR-0027). `values:` is a literal enumerated list authored in the Procedure, occupying
lines the gutter annotates and counted by the Bound like any other selector — the syntax for reaching
something `hyper` did not create by literal identifier before it can be changed. Where a `values:`
list's members are hosts, `check` verifies offline that it is a subset of the bound Target's granted
host set (ADR-0024).

## `THE CODE MOVED` change classes

Eight facts about code, plus a catch-all. The Comparison's third table (§8) emits one row for each of
these that differs between the baseline Run and the subject Run, and no row for one that does not:

- **declared Kinds** — a Definition's claimed Kinds, and a Target declaration's accepted ones.
- **selector** — a Step's `over:`, in any of its three forms. It belongs here on the same ground the
  Bound does: a Bound going 10 → 500 rendered while a selector going from one server to every server
  stays silent is truthful and still misleading.
- **Target set** — a Procedure's declared envelope, and the Targets a Definition may bind.
- **Bounds** — a Step's `bound:`, its appearance and its disappearance included.
- **Cadence** — a Procedure's declared recurrence (§10).
- **required Capabilities** — what a Manifest declares it needs.
- **the Operation set** — the Operations a Manifest exposes, and the `destroy` Operations a Definition
  names.
- **the digests** — every member of the Provenance each Record version carries (§7).

The catch-all terminates the table and is not optional: `N other lines changed · git diff <rev> <rev>`,
counting every line of every reviewed artefact that moved and that no row above reports — a widened
retention policy (§7) among them. The enumeration is what makes the table checkable; the catch-all is
what makes omission impossible.

## Scalar types

`string`, `integer`, `number`, `boolean`, `object`, `array`, plus two the domain forces: `duration`
(one integer and one unit from `s m h d`, no compounding — `14d`, never `1d12h`, so a value renders back
byte-identical to what was authored; no weeks, months, or years, since a month is not a duration) and
`timestamp` (RFC 3339, UTC, `Z` mandatory). `enum` and `const` are constraints on a type, not types of
their own. There is no `null`: a field's presence is a fact stated by the `exists` / `absent` predicate
operators, never by a nullable type.

## The artefact `kind:` values

Five values, each fixed to one directory, with `hyper.yaml` at the repository root the one exception,
agreeing with its filename rather than a directory (ADR-0023):

| directory | `kind:` |
| --- | --- |
| `definitions/` | `definition` |
| `procedures/` | `procedure` |
| `targets/` | `target-declaration` |
| `providers/` | `provider` |
| *(root)* `hyper.yaml` | `repository-declaration` |

The mapping is declared and derived at once: a file whose directory and `kind:` disagree is a load
error. `target-declaration` rather than `target`, since a Target is the concrete system and this
artefact is only its reviewed half.
