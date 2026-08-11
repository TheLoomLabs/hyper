# §12 — The closed sets

Every closed set `hyper` defines lives here, defined once and in full. Every section that names a
member of one cites this chapter; none restates a set it can cite instead.

A set below is **closed** once the sections that draw on it stop adding members to it, and stays
**open** until then. Every set below carries its marker, and none of them is open; the two whose
membership this specification does not state say so where they stand.

The process by which a set in this chapter grows — who adds a member, and when — is undecided. It is
not decided here (ADR-0004).

Two closed grammars are deliberately not here, each being the whole subject of the chapter that owns
it: the YAML subset every artefact is written in (§3) and the cron grammar a Cadence is written in
(§10). Both are stated once there and cited rather than restated everywhere else.

## Kind

**Closed.** Three values, declared per Operation in a Manifest and never inferred from the Operation's
name or from any other property `hyper` could derive on its own (ADR-0025):

- `read` — the Operation observes and changes nothing. It writes Observations.
- `mutate` — the Operation brings something into existence, or changes something that already stands.
  It writes Assets, `hyper` being accountable for what it made (§2, ADR-0025).
- `destroy` — the Operation removes something. It writes Tombstones (§7).

The order is by severity, and what turns on it is stated where it acts: what a Bound is worth on each
(§4), what a selector may expand over (§5), what runs concurrently (§6). `read` and `mutate` are
claimed at Kind level and `destroy` by named Operation, granularity following severity (§5).

## Repeatability

**Closed.** Three values, declared per Operation in a Manifest and never inferred:

- `repeatable` — invoke it again. What the Operation does is unchanged by how many times it has already
  run: an absolute-state `mutate`, most reads.
- `skip-if-recorded` — skip while the Asset it would produce still stands. The test reads the head
  version of the Record series `(Target, Definition, name)` identifies, not the existence of that
  series, so a series whose head is a Tombstone runs again (ADR-0011).
- Undeclared — **run-once**. The Operation runs where the Journal holds no evidence it already ran, and
  Refuses where it does. This is the default, and it is the strict one: an effect nobody vouched for is
  not repeated on a guess.

## Record cardinality

**Closed.** Two values, declared per Operation in a Manifest beside the response field that is the
Record's stable identity (§3):

- `one` — the Operation's response projects a single Record.
- `series` — the Operation's response projects many, out of the collection a declared path names.

What turns on it is stated where it acts: what a later Step may reference (§3), where a projection's
paths root and why the path grammar carries no iteration (below), and what a response half-projecting
leaves behind (§6).

## Disposition

**Closed.** Six values, one per Step of a Run:

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

Every Disposition carries more than its value — the Record identities, the selector, and `hyper`'s own
account of the work, which no Provider supplies (ADR-0018) — and §7 states what each of them holds.

## The outcome triple

**Closed.** Three terminal outcomes, exactly one per Run:

- `completed` — every Step reached a terminal Disposition and none of them refused or failed. A Run
  whose every Step skipped completed.
- `refused` — a guardrail declined a Step before any effect reached the world, and the Run stopped
  there (§5, ADR-0001).
- `failed` — the world resisted, or the Run was stopped: an error from a Step, a deadline, an interrupt,
  contention on the Store lock, or an open entry closed by a later Run.

There is no fourth outcome and no partial one; a Run that halted midway is `failed`, with what it
completed held by its Records and Dispositions rather than by its outcome.

## Exit codes

**Closed.** Seven members, one per way an invocation can end, each carrying the outcome it maps onto.
No member ever spans two outcomes of the triple above, and the set is finer than that triple rather
than coarser: four of the seven are `failed`.

- `0` — the command did what it was asked. A Run that completed, including one whose every Step
  skipped (§6).
- `1` — a Run that failed because the world resisted, or a command that is not a Run reporting
  problems it found. `failed`.
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

**Closed.** Two members — the two effects `hyper` performs on a Manifest's behalf (ADR-0004):

- `http` — a request to a host the bound Target grants, reaching that Target's enumerated host set and
  never the network, `local` included (ADR-0024).
- `shell` — a command run on the machine `hyper` runs on, and the Capability behind an `opaque`
  Operation. It is the one reserved to built-ins, so an Extension declaring it is refused at load
  (`capability-reserved`, §11).

A Capability is declared by a Manifest and granted by a Target declaration, both of which the
declared-equals-derived check compares against what `hyper` derives (`capability-mismatch`, §4). It is
also the key an Operation's request is written under, so what `hyper` derives per Operation is read
rather than inferred, and an Operation uses exactly one (§3).
Writing the Store passes no grant and costs no Capability, the Store not being a Target (§7,
ADR-0006), which is what keeps this the set of effects on the world.

This set is what the ceiling §13 states costs (ADR-0004).

## Auth schemes

**Closed to `hyper`.** A Manifest names a scheme and supplies its parameters, a Target declaration
names the environment variable each of that scheme's credential slots resolves from (§3), and no
Manifest mints one — a Provider author can no more invent a scheme than invent an `error_code` (§9,
ADR-0004). Closure is what lets a secret be suppressed by the position it occupies rather than by
scanning a rendering for something that looks like one (§7, ADR-0007).

Which schemes `hyper` implements is not stated here, and no section names one: §3 states how a scheme
is chosen and parameterised, §9 what a Target declaration exposes of its slots, and §11 that federation
would arrive as a scheme `hyper` owns rather than as a third-party action. The membership stands where
the growth process above stands — undecided, and not decided here.

## Patterns

**Closed.** Three: **pagination**, **polling** to a terminal condition, and **retry**, which follows
only a failure that provably preceded the request (ADR-0018). An Operation declares which of them it
uses and parameterises each one there (§3), and what each one did on a given Step is carried by that
Step's Disposition above. Pagination's two forms are closed with it — a cursor read from a response, or
a page number `hyper` increments — and a next-page URL read from a response is neither, reach arriving
from data being what ADR-0024 closed.

## `error_code`

**Closed.** Thirty-five members, each the identifier of a check that declined, named where that check is
stated, and none of them ever Provider-supplied (§9, ADR-0004).

No failure carries one. A Refusal is `hyper` declining and has a check to name; a failure is the world
resisting and has none, and the ways it can resist are not a set anything could close over. Two
failures are told apart by the exit code above rather than here.

Nineteen are contributed by §4's static checks: `strict-yaml-violation`, `unknown-key`,
`kind-mismatch`, `name-mismatch`, `schema-unsupported`, `credential-slot-malformed`, `hole-illegal`,
`series-reference`, `reference-unresolvable`, `capability-mismatch`, `manifest-inconsistent`,
`identity-undeclared`, `target-class-mismatch`,
`kind-not-granted`, `operation-not-claimed`, `envelope-exceeded`, `opaque-destroy-not-granted`,
`bound-missing`, `host-not-granted`. §6's two run-time checks carry `bound-exceeded`, an Expansion
resolving to more Records than the Step's declared Bound, and `run-once-recorded`, a run-once Step the
Journal already holds as *ran* or *attempted, outcome unknown*. §7's four are the Store's:
`store-absent`, a Run — or any other command that needs the Store (§9) — finding no Store branch;
`store-unsynced`, an effectful Run that could not sync before its first effect;
`record-identity-collision`, a Record identity colliding case-insensitively with one already written;
and `store-schema-unsupported`, a Store file whose schema version is above the reader's (ADR-0028). §9
contributes `credential-absent`, a credential a Target declaration names and the environment does not
hold, checked before a Run's first Step. §10's two are the Cadence projection's: `cadence-malformed`,
a Cadence outside the cron grammar §10 states, and `projection-stale`, a generated workflow that is
not what `project` would write now. §11's seven are distribution's. Three are the pin's:
`version-pin-mismatch`, a binary whose version differs from the Repository declaration's pin in either
direction;
`version-pin-absent`, a command that needs the pin and finds none; and `release-artefact-absent`,
`project` unable to resolve a published artefact for its own version. Four are the Extension's:
`extension-digest-mismatch`, fetched bytes or an installed Manifest that no longer match the digest
`install` verified; `provider-name-collision`, a Manifest taking a built-in Provider's name;
`capability-reserved`, a Manifest in `providers/` declaring a Capability reserved to built-ins; and
`manifest-schema-unsupported`, a Manifest whose schema version is above the reader's (ADR-0028).

A Manifest's projection of a response is a different thing wearing the same word as the two projection
codes above, and it contributes no member: a path failing to resolve against a response is read after
the call went out, so nothing declined and there is no check to name (§6, ADR-0017).

## The path grammar

**Closed.** Two roots: a Manifest's projection reads from an Operation's response, and a Step's
reference reads from a Record. From either root, the grammar is `$`, `.member`, and `["member"]`, and
nothing else.

An Operation of `series` cardinality carries one further root inside the first: the path naming the
collection reads from the response, and the identity and field paths read from each member of that
collection. That is the only position a member's field is nameable from at all, the two forms that
would otherwise name one — `[n]` and `[*]` — being outside the grammar for the reasons below. Both roots
are written `$` and the position decides which one it is, on the rule that decides every other legality
question in this format; a second marker would be a fourth production in a grammar that has three (§3).

Recursive descent (`..`) is rejected on the same ground as a YAML alias or a JSON Schema `$ref`: a path
whose meaning depends on data the reviewer cannot see while reviewing (ADR-0022). Array indexing (`[n]`)
is rejected because an index into an upstream array is the identity-is-array-position hazard that a
Manifest's declared Record identity exists to close. Iteration (`[*]` or any equivalent) is not in the
grammar at all: it is declared once, in an Operation's Record cardinality, and implied by a cardinality
of `series`.

## The Store path grammar

**Closed.** Every path the Store branch holds, and no other. Distinct from the path grammar above,
which is the one a Manifest projects with and a Step references with; this one is filenames.

| path | written |
| --- | --- |
| `STORE.md` | once, when the Store is created |
| `records/<target>/<definition>/<name>/<run-id>-<nnnn>.json` | one per Record version |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/run.json` | at Run start |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/steps/<nnnn>.json` | one per Step reaching a Disposition |
| `journal/<yyyy>/<mm>/<dd>/<run-id>/outcome.json` | when the Run ends, or by the Run that closes it |

`<run-id>` is a UUIDv7, lowercase and hyphenated: time-ordered, and mintable by either environment
alone, which a counter is not (ADR-0006). `<yyyy>/<mm>/<dd>` is the UTC date of the Run's start.
`<nnnn>` is the Step's position in the Run's written order, the first Step `0001`, nested invocations
counted in that order, zero-padded to four digits and widening beyond four rather than wrapping; it
names a Record version as well as a Step file, so two Steps of one Run writing one identity write two
paths rather than one twice.

`<target>`, `<definition>` and `<name>` are one path segment each, percent-encoded: every byte outside
`A`–`Z`, `a`–`z`, `0`–`9`, `-`, `_` and `.` is written as `%` and two uppercase hexadecimal digits over
its UTF-8 bytes, as is a leading `.`. Case is preserved rather than folded (§7). An encoded segment
longer than 200 bytes is cut at 200 on an escape boundary and suffixed with `~` and the first 16
hexadecimal digits of the SHA-256 of the whole encoded segment; `~` is outside the unreserved set
above, so it never occurs in an encoding and the suffix is unambiguous.

## The predicate operators

**Closed.** Eleven operators: `equals`, `not_equals`, `in`, `exists`, `absent`, `starts_with`,
`ends_with`, `greater_than`, `less_than`, `older_than`, `newer_than`.

A predicate list is always AND; there is no disjunction (ADR-0022). Two scopes root the same operator
set differently: a selector (`over:`) roots at the Record being filtered, and a condition (`when:`)
roots at a named earlier Step's Record and so carries `step:` beside `field:`. A `field:` is a path in
the grammar above, written from that root without the root marker. There is no
regular-expression operator — `starts_with` and `ends_with` are the bounded form of prefix and suffix
matching, in place of the unbounded one (ADR-0022).

## The template-hole positions

**Closed.** One hole syntax, three legal positions:

- A **Capability-relevant position** resolves only to a declared closed enumeration or to `from-target`,
  never to an Operation input and never to a bare wildcard. `hyper` expands the cross-product of every
  such hole against its enumeration at load and compares the derived finite set against the Target's
  grant (ADR-0024, ADR-0029). There are exactly two: an Operation's `host:` and an Auth scheme's
  parameters. A request's `path:`, `query:`, `headers:` and `body:` are not among them — a grant
  enumerates hosts and nothing finer, so no hole in one of those can widen reach past what the Target
  already granted (§3). The enumeration a hole names is declared in a Manifest's `enumerations:` and
  never in an Operation's input schema, whose `enum` constrains a value a caller supplies.
- Every other position resolves only to an Operation input.
- A position inside `auth:` resolves only to a declared credential slot, and a credential-slot hole is
  legal nowhere else. The slots are the Auth scheme's own, so no Manifest mints one (§3).

A hole resolving to none of these, or one filled from the wrong source for its position, is a load
error.

## The `over:` forms

**Closed.** Three forms — `assets:`, `observations:`, and `values:`.

`assets:` and `observations:` expand over the Step's own Definition (ADR-0012) and Target's Record
series; `observations:` is legal only on a `read` Step, since Expansion is scoped by Kind rather than
by Record type (ADR-0027). `values:` is a literal enumerated list authored in the Procedure, occupying
lines the gutter annotates and counted by the Bound like any other selector — the syntax for reaching
something `hyper` did not create by literal identifier before it can be changed. Where a `values:`
list's members are hosts, `check` verifies offline that it is a subset of the bound Target's granted
host set (ADR-0024).

## `THE CODE MOVED` change classes

**Closed.** Eight facts about code, plus a catch-all. The Comparison's third table (§8) emits one row
for each of these that differs between the baseline Run and the subject Run, and no row for one that
does not:

- **declared Kinds** — a Definition's claimed Kinds, and a Target declaration's accepted ones.
- **selector** — a Step's `over:`, in any of its three forms. It belongs here on the same ground the
  Bound does: a Bound going 10 → 500 rendered while a selector going from one server to every server
  stays silent is truthful and still misleading.
- **Target set** — a Procedure's declared envelope, the Targets a Definition may bind, and the `target:`
  a Step binds. The last belongs here for the reason the selector does: a Step moving from staging to
  production inside an envelope that already declared both changes everything about what the Run
  touches, and the envelope row would not move.
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

## The `FLAGS` vocabulary

**Closed to `hyper`.** `FLAGS` is the one vocabulary on the one editorial surface a Definition review
carries (§8); no Manifest mints a flag any more than it mints an `error_code`. Which names it draws
on is not stated here, and stands where the growth process above stands — undecided, and not decided
here. What binds every name whatever they turn out to be is the relation §8 states.

## The input-schema subset

**Closed.** Six keywords are the whole of the JSON Schema an Operation's input schema is written in:
`type`, `required`, `enum`, `const`, `properties`, and `items`. `additionalProperties: false` is forced
at every level rather than authored, so an unknown key is refused wherever it appears (`unknown-key`,
§4). A schema reaching outside this subset is `schema-unsupported`.

`$ref` is rejected on the ground a YAML alias is, a schema whose meaning lives in a document the
reviewer is not reading (ADR-0023); `allOf`, `oneOf`, and `if`/`then`/`else` are rejected because a
schema that composes or branches is an expression language in a second costume (ADR-0022). `enum` and
`const` are constraints on a type rather than types of their own, which is why they sit here and not in
the set below. This subset covers an Operation's input; its output has no schema at all (§3).

## Scalar types

**Closed.** Eight types: `string`, `integer`, `number`, `boolean`, `object`, and `array`, plus two the
domain forces: `duration` (one integer and one unit from `s m h d`, no compounding — `14d`, never
`1d12h`, so a value renders back byte-identical to what was authored; no weeks, months, or years, since
a month is not a duration) and `timestamp` (RFC 3339, UTC, `Z` mandatory). There is no `null`: a
field's presence is a fact stated by the `exists` / `absent` predicate operators, never by a nullable
type.

## The artefact `kind:` values

**Closed.** Five values, each fixed to one directory, with `hyper.yaml` at the repository root the one
exception, agreeing with its filename rather than a directory (ADR-0023):

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
