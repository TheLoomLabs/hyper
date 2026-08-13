# §12 — The closed sets

Every closed set `hyper` defines lives here, defined once and in full. Every section that names a
member of one cites this chapter; none restates a set it can cite instead.

A set below is **closed** once the sections that draw on it stop adding members to it, and stays
**open** until then. Every set below carries its marker, none of them is open, and every one of them
states its membership in full.

The process by which a set in this chapter grows — who adds a member, and when — is undecided. It is
not decided here (ADR-0004).

Two closed grammars are deliberately not here, each being the whole subject of the chapter that owns
it: the YAML subset every artefact is written in (§3) and the cron grammar a Cadence is written in
(§10). Both are stated once there and cited rather than restated everywhere else.

## How a member is spelled

A member of a set below is written in kebab-case wherever it goes on the wire — into a Store file, into
a row of the stream §8 states, into an artefact — and the keys around it are written in snake_case:
`"disposition": "skipped-by-condition"`, `"error_code": "bound-exceeded"`. The split is stated because
it is already in force everywhere and is otherwise a coincidence a reader has to notice. No member is
abbreviated on the wire; a set whose members are shortened for the file is a second set of names.

## Kind

**Closed.** Three values, declared per Operation in a Manifest and never inferred from the Operation's
name or from any other property `hyper` could derive on its own (ADR-0025):

- `read` — the Operation observes and changes nothing. It writes Observations.
- `mutate` — the Operation brings something into existence, or changes something that already stands.
  It writes Assets, `hyper` being accountable for what it made (§2, ADR-0025).
- `destroy` — the Operation removes something. It writes Tombstones (§7).

The order is by severity, and what turns on it is stated where it acts: what a Bound is worth on each
(§4), what a selector may expand over (§5), what runs concurrently and which Operations may declare a
`concurrency:` limit at all (§6, §3). `read` and `mutate` are
claimed at Kind level and `destroy` by named Operation, granularity following severity (§5).

## Repeatability

**Closed.** Three values, declared per Operation in a Manifest and never inferred:

- `repeatable` — invoke it again. What the Operation does is unchanged by how many times it has already
  run: an absolute-state `mutate`, every read.
- `skip-if-recorded` — skip while the Asset it would produce still stands. The test reads the head
  version of the Record series `(Target, Definition, name)` identifies, not the existence of that
  series, so a series whose head is a Tombstone runs again (ADR-0011). It decides **per Record**, one
  series per member of the Step's Expansion, so a Step may skip some members and call for others
  (§6, ADR-0056).
- Undeclared — **run-once**. The Operation runs where the Journal holds no evidence it already ran, and
  Refuses where it does. This is the effectful default, and it is the strict one: an effect nobody
  vouched for is not repeated on a guess.

**Which values a Kind may declare is fixed by what that Kind projects** (ADR-0037), since two of the
three read a projection to decide:

| Kind | `repeatable` | `skip-if-recorded` | run-once | undeclared means |
| --- | --- | --- | --- | --- |
| `read` | yes | no | — | `repeatable` |
| `mutate` | yes | yes | yes | run-once |
| `destroy` | yes | no | yes | run-once |

`skip-if-recorded` is a `mutate`-only value because `mutate` is the only Kind that projects the Asset
its test reads. A `destroy` projects nothing (§3), and the head its test would find is the live Asset
the Step exists to remove, so the value reads exactly backwards there — while the reading that
preserves the intent, *skip what is already gone*, is what §5's Expansion already performs on every
selector and every `values:` member. A `read` projects an Observation, which no `destroy` may ever
Tombstone (ADR-0032), so its head stands from the first Run forever: the value would read the world
once and skip every occurrence after, reporting `completed` each time.

Deciding per Record puts two requirements on a `skip-if-recorded` Operation that no other value carries,
both of them checked before a Run (§4). Its `identity:` must **resolve before the call**, since the name
the test reads is the name the call it is deciding on would write under: a template hole has that
property, resolving to an Operation input like any hole outside a Capability-relevant position (below),
and so does `$.command` on a `shell` Operation, which is in the response object precisely
because it is a fact about the call rather than about the answer (§3). A response path anywhere else
names a value that exists only once the call has gone out, and is `manifest-inconsistent`. And the Step
must be able to reach a member the test can answer *run* for, which an `assets:` selector never
does — an effectful Expansion reaches only standing Assets, so every member stands by construction and
the Step can never call (`skip-if-recorded-unreachable`). A `values:` list is the form that expresses a
population `hyper` may not yet have built, which is what the value is for.

Run-once has no spelling, so a `read`'s only expressible Repeatability is `repeatable`, whether written
or omitted. The strict default is not withheld from `read` as an exception but because its own reason —
*an effect nobody vouched for* — names something a `read` does not do; it is the shape §4 already uses
for a Bound, which a `read` Step carries none of, having nothing for one to guard.

## Record cardinality

**Closed.** Two values, declared per Operation in a Manifest beside the response field that is the
Record's stable identity (§3):

- `one` — the Operation's response projects a single Record.
- `series` — the Operation's response projects many, out of the collection a declared path names.

What turns on it is stated where it acts: what a later Step may reference (§3), where a projection's
paths root and why the path grammar carries no iteration (below), and what a response half-projecting
leaves behind (§6).

## Disposition

**Closed.** Seven values, one per Step of a Run, each with its wire spelling beside it:

- **ran** — `ran`. The Step was invoked and reached a conclusion `hyper` recorded. Usually because its
  outcome came back; also where no answer came back and that absence was itself the answer, a `read`
  against a host that answers nothing recording an Observation whose `status` has gone quiet (§6,
  ADR-0050), and where the answer came back and `hyper` could not read it (§6).
- **skipped as already recorded** — `skipped-as-already-recorded`. The Step's Operation declared
  `skip-if-recorded` and the Asset it would produce still stands. The only value that is Repeatability
  evidence. On an expanding Step it is the value where **every** member skipped; where any call went
  out the Step is *ran*, that value claiming no count (§6, ADR-0056).
- **skipped by condition** — `skipped-by-condition`. The Step's `when:` did not hold. Says nothing about
  what the world holds, which is why it is not the same value as the skip above.
- **refused** — `refused`. A guardrail declined the Step before any effect reached the world (§5).
- **never reached** — `never-reached`. The Run ended before the Step. A run-once Step in this state runs
  on a re-run.
- **attempted, outcome unknown** — `attempted-outcome-unknown`. The call went out and no answer came
  back. It attaches the uncertainty to the attempt rather than to the thing, so nothing downstream reads
  it as either success or failure.
- **attempted, world untouched** — `attempted-world-untouched`. The request provably never left, so
  `hyper` is certain nothing was touched. Effectful-only, and only where **no** call this Step made
  reached the world: a `read` in this state recorded an answer and is *ran*, and a `destroy` that
  confirmed three of five before the fourth request never left is *ran* too (§6, ADR-0062). It concludes
  about nothing and carries no identity set, and it is the one failure that is not Repeatability
  evidence.

The last value's extent is the transport class ADR-0018 fixes and is not re-derived from it: a
**connection refused**, a **name that did not resolve**, a **handshake that failed**, and — the same
fact one Capability over — a **child process that could not be started at all**. A connect timeout is
outside it, ADR-0018 declining to retry one, so a certainty `hyper` will not act on is not a certainty
it asserts. These are the value's boundary and not its content: a Step file says the request never left
and never which of the four it was (§7, §13).

Every Disposition carries more than its value — the identities it concluded about, the selector, and
`hyper`'s own account of the work, which no Provider supplies (ADR-0018, ADR-0030) — and §7 states what
each of them holds and which values carry which. Six of the seven are borne by a Step file; **never
reached** is read from the absence of one inside a closed entry (§7).

## The Trigger's cause and its executor

**Closed.** Two sets of two, carried by every Journal entry's Trigger (§7).

- `cause` — `cron`, a Cadence the executor's clock fired (§10), or `manual`, a person. A dispatched
  workflow run is `manual` on the Actions executor, which is why these are two fields and not one.
- `executor` — `github-actions` or `local`.

`hyper` fills both by reading the environment it finds itself in, and no rule in this specification
reads either one back. Recording where a Run happened is a fact about the occasion; deciding anything
from it would be the authority axis §5 does not have.

## The outcome triple

**Closed.** Three terminal outcomes, exactly one per Run:

- `completed` — every Step reached a terminal Disposition and none of them refused or failed. A Run
  whose every Step skipped completed.
- `refused` — a guardrail declined before any effect reached the world, and the Run stopped there
  (§5, ADR-0001). Most often before any Step existed, `check` re-running in full at Run start (§6).
- `failed` — the world resisted, or the Run was stopped, or it lost the Store: an error from a Step, a
  deadline, an interrupt, contention on the Store lock, a sync or push it could not complete, or an open
  entry closed by a later Run.

There is no fourth outcome and no partial one; a Run that halted midway is `failed`, with what it
completed held by its Records and Dispositions rather than by its outcome.

## Exit codes

**Closed.** Seven members, one per way an invocation can end, each carrying the outcome it maps onto.
No member ever spans two outcomes of the triple above, and the set is finer than that triple rather
than coarser: four of the seven are `failed`.

- `0` — the command did what it was asked. A Run that completed, including one whose every Step
  skipped (§6).
- `1` — a Run that failed because the world resisted, or a command that is not a Run reporting
  problems it found. It is also where `install` lands a ref the registry does not hold, a name in
  somebody else's namespace being the world rather than the invocation (§11, ADR-0060). `failed`.
- `2` — a usage error. No Run began, and no member of the outcome triple applies. It covers a
  positional that matches nothing on eight of the nine commands taking one — `install` is the
  exception above — and no row stream opens on this code at all (§9, ADR-0060).
- `75` — a Run that lost the Store: to the lock (§6), to the sync at Run start (§7), or to a push it
  could not rebase through in three attempts (§7). `failed`.
- `77` — a guardrail declined before any effect reached the world. A Run that refused (§5), and the
  code the version pin gate and an absent Store carry from any command that hits them (§9).
- `130` — a Run stopped by an interrupt, having drained (§6, ADR-0015). `failed`.
- `143` — a Run stopped by a termination signal, drained the same way (§9). `failed`.

`75` and `77` are `sysexits`' `EX_TEMPFAIL` and `EX_NOPERM`, and the pairing carries the difference
between them: `75` says retry me, and `77` says a verbatim retry will refuse identically (§9). What
sorts a stop into one or the other is whether an act is required to clear it — an edit, an `init`, a
`project`, a newer binary, a variable set — and never how severe it was (ADR-0061).

## Capabilities

**Closed.** Two members — the two effects `hyper` performs on a Manifest's behalf (ADR-0004):

- `http` — a request to a host the bound Target grants, reaching that Target's enumerated host set and
  never the network, `local` included (ADR-0024).
- `shell` — a command run on the machine `hyper` runs on, and the Capability behind an `opaque`
  Operation. Its request block carries one key, `command:`, a list of argv words `hyper` execs directly
  with no interpreter between the artefact and the process (§3). It is the one reserved to built-ins,
  so an Extension declaring it is refused at load (`capability-reserved`, §11) — and it is the one no
  Probe may invoke, whatever any Target grants (§9).

**Opacity is a property of the Capability rather than of the Operation.** `http` describes what it does
— a method, a host, a path, a body, every one of them in the artefact — and `shell` cannot describe
anything. An Operation is `opaque` exactly where its Capability is, which is why no artefact declares it
(§3) and every surface still renders it (§9).

A Capability is declared by a Manifest, which the declared-equals-derived check compares against what
`hyper` derives (`capability-mismatch`, §4), and granted by a Target declaration, which the bound
Manifest's requirement is checked against per binding (`capability-not-granted`, §4). It is
also the key an Operation's request is written under, so what `hyper` derives per Operation is read
rather than inferred, and an Operation uses exactly one (§3).
Writing the Store passes no grant and costs no Capability, the Store not being a Target (§7,
ADR-0006), which is what keeps this the set of effects on the world.

This set is what the ceiling §13 states costs (ADR-0004).

## The built-in Providers

**Closed.** One member, `shell`, fixed by a criterion rather than by a list: **`hyper` ships a Provider
only where the Capability it needs is one nobody else may declare** (ADR-0039, §11). The set doubles as
the list of names no Extension may take (`provider-name-collision`, §11), and it grows only where the
reserved half of the set above grows.

`shell` declares `capabilities: [shell]` and no Auth scheme, a credential being a property of reaching a
host and a command reaching none (§3). Its `class:` is `local`, so a shell Step binds a class-local
Target and nothing else: a command runs on the machine `hyper` runs on, and a Step naming a Target that
describes somewhere else would be `hyper`'s own Manifest claiming a place it does not reach. How many
such Targets a repository declares is the repository's (§3, ADR-0041) — each is a name for that one
machine, carrying its own accepted Kinds, its own credential slots and its own `opaque-destroy:`.

Six Operations, which is Kind crossed with the Repeatability values each Kind may declare above:

| Operation | Kind | Repeatability |
| --- | --- | --- |
| `read` | `read` | `repeatable` |
| `mutate` | `mutate` | `repeatable` |
| `mutate_once` | `mutate` | run-once, by omission |
| `mutate_skip_if_recorded` | `mutate` | `skip-if-recorded` |
| `destroy` | `destroy` | `repeatable` |
| `destroy_once` | `destroy` | run-once, by omission |

There are six rather than one because a Manifest declares the facts the Provider author knows and the
Definition author would be guessing at (§6) — and here the Provider author is `hyper`, which knows
nothing whatever about the command. No artefact downstream may override a declared fact (§13), so every
declaration an author needs to vary becomes an Operation of its own, exactly as the Kind already does.
The Operation's name is therefore the claim, and it renders where a claim renders: in the gutter beside
the Step, and in `AUTHORITY`'s `DESTROY OPS` column (§8).

Two declarations are the same on all six. `patterns:` is empty: pagination and polling have no meaning
against a command, and retry follows only a failure that provably preceded a request (ADR-0018), which a
command has no equivalent of. `deadline:` is one hour, a deadline bounding `hyper`'s patience rather
than blast radius; nothing downstream may raise it and §13 states what that costs.

A third declaration belongs to `read` alone, that being the one of the six that may carry it (§3):
`concurrency:` is omitted, so a `read` Expansion over commands runs them one after another. It is the
same sentence as the two above — `hyper` knows nothing whatever about the command, and how many of them
a machine will tolerate at once is exactly the sort of thing it would be guessing at (ADR-0045).

The request is the same on all six, and it is the sentence again in its purest form: `command:` is a
hole, and the argv arrives in a Step's `args:` (§3). The projection is the same on the four that carry
one, `destroy` and `destroy_once` being forbidden a `record:` like every other `destroy` (ADR-0037) —
so a shell Record's name is always the command that produced it. No `secret:` is declared anywhere in
it, for the reason §3 gives and at the cost §13 states.

```yaml
kind: provider
provider: shell
schema-version: 1
class: local
capabilities: [shell]
operations:
  read:
    kind: read
    repeatability: repeatable
    deadline: 1h
    shell:
      command: "{command}"
    input:
      type: object
      required: [command]
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  mutate:
    kind: mutate
    repeatability: repeatable
    deadline: 1h
    shell:
      command: "{command}"
    input:
      type: object
      required: [command]
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  destroy:
    kind: destroy
    repeatability: repeatable
    deadline: 1h
    shell:
      command: "{command}"
    input:
      type: object
      required: [command]
      properties:
        command: {type: array, items: {type: string}}
```

`mutate_once` and `destroy_once` are the two above with `repeatability:` omitted, and
`mutate_skip_if_recorded` is `mutate` with `skip-if-recorded` in its place; nothing else in any of the
three differs, which is the point of there being six. The repetition is not factored out because a
Manifest has no factoring construct and would need one invented for its own author's convenience
(ADR-0022), and because `operation` writes these lines back unchanged (§9) — what a reviewer reads is
what `manifest_digest` covers.

## Auth schemes

**Closed.** Two members, both of them a request header:

- **`header:`** — parameters `name:`, the header, and `prefix:`, optional and absent meaning empty,
  concatenated verbatim in front of the credential. One slot, `token`. It covers a bearer token
  (`Authorization` / `"Bearer "`), an API key in any header (`X-Api-Key` / empty), and a vendor's
  compound token (`Authorization` / `"PVEAPIToken="`), which are one placement rather than three schemes.
- **`basic:`** — no parameters, and two slots, `username` and `password`, composed into
  `Authorization: Basic <base64>`. It is not a `header:` with a prefix, because that would leave a human
  base64-encoding credentials by hand into an environment variable that no longer holds what the vendor
  issued.

A Manifest names a scheme and supplies its parameters, a Target declaration names the environment
variable each of that scheme's credential slots resolves from (§3), and no Manifest mints a scheme or a
slot — a Provider author can no more invent either than invent an `error_code` (§9, ADR-0004). Closure
is what lets a secret be suppressed by the position it occupies rather than by scanning a rendering for
something that looks like one (§7, ADR-0007). A scheme naming a header `hyper` computes is refused
(`auth-header-reserved`, §4), and a scheme's parameters carry literals and admit no hole (§3).

`auth:` is optional and its absence means no credential is sent, rendered as `none` wherever a
Provider's auth renders (§3, §9). Absence is not a third member: a scheme is a way of authenticating a
request, and not authenticating one is not a way of doing it.

**An Auth scheme is a header and a placement, never a protocol** (ADR-0031), which is what fixes the
membership at two rather than leaving it to accumulate. Request signing, OAuth2 client credentials, and
a client certificate are each refused by that sentence rather than beside it, and each stands at the
wall §13 states costs for.

## Patterns

**Closed.** Three: **pagination**, **polling** to a terminal condition, and **retry**, which follows
only a failure that provably preceded the request (ADR-0018). An Operation declares which of them it
uses and parameterises each one there (§3), and what each one did on a given Step is carried by that
Step's Disposition above. All three are serial by construction — each learns whether there is another
call to make only from the answer to the one before it — so an Operation's `concurrency:` limit governs
its Expansion and never a Pattern (§3, §6, ADR-0045). Pagination's two forms are closed with it — a cursor read from a response, or
a page number `hyper` increments — and a next-page URL read from a response is neither, reach arriving
from data being what ADR-0024 closed.

## `error_code`

**Closed.** Forty-seven members, each the identifier of a check that declined, named where that check is
stated, and none of them ever Provider-supplied (§9, ADR-0004).

No failure carries one. A Refusal is `hyper` declining and has a check to name; a failure is the world
resisting and has none, and the ways it can resist are not a set anything could close over. Two
failures are told apart by the exit code above rather than here.

**Most of the set declines before Step 1.** A Run re-runs `check` in full at its start (§6), so all
thirty of §4's static codes reach a Run that way, beside the credential pass, the Cadence's and
the Store's. Where one does, it is held on `outcome.json` and never on a Step file, `step` being an
artefact coordinate rather than an execution fact (§7, ADR-0061). Only `bound-exceeded`,
`run-once-recorded`, `record-identity-collision` and §6's `predicate-type-mismatch` require a Step to
have been reached at all.

Thirty are contributed by §4's static checks alone: `strict-yaml-violation`, `unknown-key`,
`kind-mismatch`, `name-mismatch`, `schema-unsupported`, `credential-slot-malformed`, `hole-illegal`,
`series-reference`, `artefact-absent`, a name an artefact writes for one of this repository's artefacts
resolving to nothing — a Definition's `provider:` or `targets:` member, a Step's `definition:`, a nested
invocation's `procedure:` — which is a check with a file and a line rather than a failure to load
(§4, ADR-0064); `reference-unresolvable`, the same fault where the namespace is what an artefact
declares rather than what the repository holds — a Step's `operation:`, a Definition's `destroy:`
member, a `field:` at either Record root, and the `step:` half of a reference (§3);
`capability-mismatch`, `manifest-inconsistent`,
`target-inconsistent`, `auth-header-reserved`, `local-reserved`, `identity-undeclared`,
`target-class-mismatch`, `definition-kinds-mixed`,
`kind-not-granted`, `capability-not-granted`, `operation-not-claimed`, `target-not-claimed`, a Step
binding a Target its Definition does not claim, which is `operation-not-claimed`'s shape one key over
and its own member because a reader handed that code on a `target:` line would edit `destroy:` (§4);
`envelope-exceeded`,
`opaque-destroy-not-granted`, `bound-missing`, `bound-illegal`, `host-not-granted`,
`command-malformed`, a shell Step's `command:` that is empty or names its executable by reference
(§3, ADR-0051), `opaque-destroy-unscoped`, an `opaque` `destroy` Step carrying no `over:` selector
and therefore reaching the world with nothing to write a Tombstone under (§5, ADR-0053), and
`skip-if-recorded-unreachable`, a `skip-if-recorded` Step expanding over `assets:`, whose every member
stands by construction and whose test can therefore only ever answer *skip* (above, ADR-0056). §6's two run-time checks carry `bound-exceeded`, an Expansion
resolving to more Records than the Step's declared Bound, and `run-once-recorded`, a run-once Step the
Journal already holds as *ran* or *attempted, outcome unknown*.

Two members are stated by §4 and §6 both. `bound-exceeded` is one, §4 firing it where the Expansion's
count is authored and therefore known offline (a `values:` list longer than the Bound) and §6 firing it
everywhere else. `predicate-type-mismatch` is the other, an operator handed a type it does not take:
§4 fires it where the fault is authored — a `timestamp` under `greater_than`, `exists: false`, an
`in:` that is empty, of one member or of mixed types, an empty `starts_with:`, a predicate against a
declared-secret field — and §6 fires it against a stored value at Expansion (ADR-0035). Each is one code because it is one check:
what names a Refusal is the check that declined, never the moment it ran, and a reader is never holding
one without knowing whether they asked `check` or a Run. §7's three are the Store's:
`store-absent`, a Run — or any other command that needs the Store (§9) — finding no Store branch;
`record-identity-collision`, a Record identity colliding case-insensitively with one already written —
and, on the same shared-code rule as the two above but across a different pair, §3 firing it at load
where two members of one `values:` list are one identity under that same fold (§3), which is the one
place the collision is authored and therefore catchable with no Store at all;
and `store-schema-unsupported`, a Store file whose schema version is above the reader's (ADR-0028),
tested at Run start over the files the Run will read (§6). A Run that could not sync the Store
contributes no member: it is `failed` at `75` rather than a Refusal, the network coming back being no
act of anyone's, and `77` promising above that a verbatim retry refuses identically (§7, ADR-0061). §9
contributes `credential-absent`, a credential a Target declaration names and the environment does not
hold, checked before a Run's first Step and reported for every absent slot at once. §10's three are the Cadence's: `cadence-malformed`,
a Cadence outside the cron grammar §10 states; `projection-stale`, a generated workflow that is
not what `project` would write now; and `cadence-run-once`, a Procedure declaring a Cadence that
reaches a run-once Step at any depth (§4, ADR-0038) — a recurrence whose second occurrence Refuses,
which is why it is refused before the first. §11's seven are distribution's. Three are the pin's:
`version-pin-mismatch`, a binary whose version differs from the Repository declaration's pin in either
direction;
`version-pin-absent`, a command that needs the pin and finds none; and `release-artefact-absent`,
`project` unable to resolve a published artefact for its own version — no release under the tag, no
checksums file beside it, or no line in that file for the artefact the compiled-in template names
(§11). Four are the Extension's:
`origin-digest-mismatch`, fetched bytes or an installed Manifest that no longer match the digest
`install` verified — the check and the Provenance field it guards name one fact (§7);
`provider-name-collision`, a Manifest taking a built-in Provider's name;
`capability-reserved`, a Manifest in `providers/` declaring a Capability reserved to built-ins; and
`manifest-schema-unsupported`, a Manifest whose schema version is above the reader's (ADR-0028).

A Manifest's projection of a response is a different thing wearing the same word as the two projection
codes above, and it contributes no member: a path failing to resolve against a response is read after
the call went out, so nothing declined and there is no check to name (§6, ADR-0017).

## The response objects

**Closed**, and there is one per Capability. A Manifest's projection and a polling Pattern's `until:`
read from the object belonging to the Capability the Operation's request is written under, and never
from the bytes that came back (ADR-0040, §3). The two share no member: `http` describes what it did and
`shell` describes nothing, so what each can be asked about afterwards differs the same way.

### `http`

**Five members.**

- `host` — the host the request reached, which is the one host the candidate set and the grant
  intersected to (ADR-0029). It is a fact about the call rather than the answer, and it is here because
  an Operation whose answer carries no identity of its own has nowhere else to project a Record identity
  from.
- `status` — the HTTP status, an integer, **absent** where no response arrived.
- `headers` — a mapping of header name to value, names lowercased: a header name is case-insensitive on
  the wire and a path is exact, so the lowering is what makes one path mean one thing. **Absent** where
  no response arrived.
- `body` — the parsed JSON body, **absent** where the response carried none or carried something else,
  and its absence is not an error. A site that is down answers with no body at all, and an uptime check
  is pointed at hosts that answer in HTML.
- `tls` — present where the scheme was HTTPS, carrying `not_after`, `days_left`, `subject`, and
  `issuer`. `days_left` is a member because no artefact could compute one: there is no arithmetic in the
  format (ADR-0022), and what it counts from is the instant the Run fixed (ADR-0034).

Where **no response arrived at all** the object is `host` and nothing else (ADR-0050). `host` is the
member that survives because it is a fact about the call rather than about the answer, which is what
lets a `read` record a host that answered nothing rather than halting on it (§6). No member says what
went wrong: that is the catch-all bucket ADR-0017 closed, and what stands in its place is the same
absence a projection already reads.

A status is an answer and never an `error_code`. Which statuses halt follows Kind and is stated in §6,
and no set here has an opinion about it: nothing is declared, so there is nothing to enumerate, and a
range `hyper`'s own code applies is not a closed set an artefact draws members from.

There is no duration or latency member. A Record versions only on change (§7), so a timing field would
mint a version on every Run that projected it and fill the record with evidence that `hyper` ran rather
than that anything moved; a duration is computed inside one Journal entry, which is where this one
already lives (§7). There are no raw bytes either, on the ground ADR-0017 settled for rendering.

### `shell`

**Four members.**

- `command` — the argv as run, JSON-encoded on one line. It is `host`'s member argument one Capability
  over: a fact about the call rather than about the answer, here because an Operation whose answer
  carries no identity of its own has nowhere else to project one from. JSON rather than a joining rule
  because it must be injective — `[echo, "a b"]` and `[echo, a, b]` are two commands, and a joining
  rule makes them one series that `record-identity-collision` can never catch, the two names being
  genuinely equal.
- `exit_code` — the code the command exited with, an integer, **absent** where it could not be started.
- `stdout` — what the command wrote to standard output, **text**, never parsed (ADR-0052). **Absent**
  where the command could not be started.
- `stderr` — the same, for standard error.

Where the command **could not be started at all** the object is `command` and nothing else, on the rule
the `http` object carries for a call that got no answer (ADR-0050). No member says why: that is the
catch-all bucket ADR-0017 closed, arriving on the object every projection reads from.

There is no parsed form of `stdout`, and its absence is the `opaque` trait arriving in the projection.
`hyper` cannot describe what a command does, and parsing its output is a description of exactly that.
Nothing enforces it beyond the grammar below, which reaches no further inside a scalar than inside a
string anywhere else — so a shell projection reaches `$.exit_code`, `$.stdout` and `$.stderr` and
nothing finer, and a shell response holds no collection, so every shell Operation is of `one`
cardinality by construction.

An exit code is an answer and never an `error_code`, on the sentence the `http` object carries above.
Which codes halt follows Kind and is stated in §6, and no set here has an opinion about it.

## The path grammar

**Closed.** Two roots: a Manifest's projection reads from its Capability's response object above, and a
Step's reference reads from a Record. From either root, the grammar is `$`, `.member`, and
`["member"]`, and nothing else.

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
`<nnnn>` is the Step's position in the Run's written order, the first Step `0001`, a nested Procedure's
Steps counted in that order — the invocation itself being no Step and writing no file (§7) —
zero-padded to four digits and widening beyond four rather than wrapping; it names a Record version as
well as a Step file, so two Steps of one Run writing one identity write two paths rather than one
twice.

`<target>`, `<definition>` and `<name>` are one path segment each, percent-encoded: every byte outside
`A`–`Z`, `a`–`z`, `0`–`9`, `-`, `_` and `.` is written as `%` and two uppercase hexadecimal digits over
its UTF-8 bytes, as is a leading `.`. Case is preserved rather than folded (§7). An encoded segment
longer than 200 bytes is cut at 200 on an escape boundary and suffixed with `~` and the first 16
hexadecimal digits of the SHA-256 of the whole encoded segment; `~` is outside the unreserved set
above, so it never occurs in an encoding and the suffix is unambiguous.

The encoding names a file and orders nothing. An Expansion is ordered by the Record `name` itself (§6,
ADR-0044), which a listing of one of these directories is not: escaping drags every escaped character
to the left of every unreserved one, so `Über-vm` sorts after `zone-a` by name and before it by path.

## The predicate operators

**Closed.** Eleven operators, each taking the operand types below and no others:

| operator | operand | holds where |
| --- | --- | --- |
| `equals` | `string`, `integer`/`number`, `boolean`, `duration`, `timestamp` | the value is that value |
| `not_equals` | as `equals` | the value is not that value |
| `in` | a list of two or more literals, all one type | the value equals a member |
| `exists` | `true` | the version carries the field |
| `absent` | `true` | the version does not carry the field |
| `starts_with` | a non-empty `string` | the value has that prefix |
| `ends_with` | a non-empty `string` | the value has that suffix |
| `greater_than` | `integer`/`number`, `duration` | the value is strictly greater |
| `less_than` | `integer`/`number`, `duration` | the value is strictly less |
| `older_than` | `duration`, `timestamp` | the value is strictly before the instant it names |
| `newer_than` | `duration`, `timestamp` | the value is strictly after the instant it names |

A `duration` operand under `older_than` or `newer_than` names the instant that far before the one below;
a `timestamp` operand names itself. Neither operator takes a `duration`-valued field, an interval having
no age; `greater_than` is what compares two lengths.

A predicate list is always AND; there is no disjunction (ADR-0022). `in` is not a reopening of it: an
`any_of` disjoins whole predicates and so puts two disjoint populations under one Bound, where `in`
disjoins the values of one field over one population, every alternative on the line the gutter
annotates. There is no regular-expression operator — `starts_with` and `ends_with` are the bounded form
of prefix and suffix matching, in place of the unbounded one (ADR-0022). There is no `>=`: the four
ordered operators are strict, and the pairs are how negation is written. **Negation lives in an
operator's name and never in its operand**, which is why `exists` and `absent` are two members rather
than one taking a boolean, and why `exists: false` is refused: it is a second spelling of `absent:
true` producing an identical filter, and it is the spelling where one character in a value inverts a
`destroy` selector while the operator a reviewer reads does not move.

### The three roots

A selector (`over:`) roots at the Record being filtered; a condition (`when:`) roots at a named earlier
Step's Record and carries `step:` beside `field:`; a polling Pattern's `until:` roots at the response
object in hand (§3, and the object above).

At the two Record roots a `field:` is **one declared field name** — a key of the Manifest's `fields:`
mapping (§3) — and nothing else: no descent, no brackets, no path. There is nothing there for a path to
traverse, a Record's field names being flat and authored, and naming one is what makes the check below
total. A projected value that is an object is therefore unfilterable, which is paid in the right place:
the Provider author writes `region: $.body.config.region` once in the Manifest, reviewed, rather than every
selector spelling the descent out. At the response root a `field:` **is** a path in the grammar above,
written without the root marker, a response having paths and no declared names. Which of the two it is
follows from the position, as every other legality question in this format does (§3).

A `field:` naming what no Operation of the Provider projects is `reference-unresolvable` (§4), the same
code and the same check a reference gets. It is a load error rather than a predicate that never holds
because `absent` over an undeclared field is true for every version in the series (ADR-0035). The
`over:` form `values:` carries no predicate list at all: its members are bare scalars with no declared
fields to name, so there is nothing there to filter on (§7, ADR-0033).

A selector reads the **head** version of each series and no other (§5). *Any version* would have a
`destroy` reach a thing for what it used to be, and would make one artefact's reach grow with the Store
every month.

### What the comparisons mean

Numbers are one domain: `integer` and `number` are two scalar types where an input schema constrains
what a caller supplies, and one comparison where a value has already come back, so `equals: 1` holds
against `1.0`. Durations compare by normalised length, so `10m` is greater than `300s` — the
no-compounding rule above is about a value rendering back byte-identical to what was authored, which is
a fact about writing rather than about ordering. Strings compare **byte-exact over UTF-8**,
case-sensitive, with no normalisation, on the ground §7 folds no case in a Record identity: the rule is
`hyper`'s rather than the locale's, and it is the same *the bytes moved* test the canonical encoding
already runs.

Time is `older_than` and `newer_than` entirely, and both take either a `duration`, read relative to the
instant below, or an absolute `timestamp`. `greater_than` against a timestamp is refused rather than
allowed as a synonym: it is a second spelling of a byte-identical comparison, and it is the spelling
that reads backwards, *greater* standing in for *later*. A projected value is read as any RFC 3339
timestamp, an offset included, and normalised to UTC; the `Z` above binds a value an author writes,
where two artefacts could otherwise disagree about what a local time meant, and there is no author
involved when an API hands one back. An epoch integer is a number rather than a timestamp and no
operator reads it as one — §13 states the cost.

**Every predicate in an invocation is evaluated against one instant, read once at its start**
(ADR-0034). For a Run it is the `started_at` on `run.json` (§7), used verbatim, so nothing is stored to
hold it; for a Probe it is the Probe's own start, recorded nowhere as a Probe records nothing
(ADR-0009). One instant covers every Step, every nested Procedure and all three roots, so nothing a
Pattern or a slow API does during a Run can move what a later Step reaches.

### What a predicate cannot decide

A projected field has no declared type (§3), so a predicate can be handed a value it cannot compare.
**It Refuses; it never treats the value as not matching** (ADR-0035). A predicate list does not
short-circuit — every conjunct is evaluated against every candidate — and nothing coerces. Refused with
them are the predicates whose truth cannot depend on the value: an empty `in:`, an empty `starts_with:`
or `ends_with:`, and a predicate against a field the Manifest declares secret, that field being written
as the constant `"<secret>"` (§7). Refused beside those are the operand faults that are not comparisons
at all: an `in:` of mixed types, `exists: false`, and a `timestamp` operand under `greater_than` or
`less_than`. A one-member `in:` is refused too, being a second spelling of `equals` — one filter, two
ways to write it, which would render as a change in `THE CODE MOVED` with nothing moved.

The code is `predicate-type-mismatch`, stated by §4 where the fault is authored and knowable offline
and by §6 where it is a stored value at Expansion. A polling `until:` contributes no Refusal: it reads
a response after the call went out, so what applies there is §6's *when a projection does not resolve*
— the Run halts, no `error_code` is carried, and what is named is the field and what was found in it.

## The template-hole positions

**Closed.** One hole syntax, three legal positions:

- A **Capability-relevant position** resolves only to a declared closed enumeration or to `from-target`,
  never to an Operation input and never to a bare wildcard. `hyper` expands the cross-product of every
  such hole against its enumeration at load and compares the derived finite set against the Target's
  grant (ADR-0024, ADR-0029). There is exactly one: an Operation's `host:`. A request's `path:`,
  `query:`, `headers:` and `body:` are not among them — a grant
  enumerates hosts and nothing finer, so no hole in one of those can widen reach past what the Target
  already granted (§3). The enumeration a hole names is declared in a Manifest's `enumerations:` and
  never in an Operation's input schema, whose `enum` constrains a value a caller supplies.
- Every other position resolves only to an Operation input.
- Inside `auth:` no hole is legal at all. A Manifest writes an Auth scheme's parameters as literals, and
  the credential itself is never written there in any form: the scheme owns the position it occupies, so
  there is nothing for a hole to stand in for (§3, ADR-0031). This is the one position where a hole is
  refused outright rather than restricted to a source.

A hole resolving to none of these, or one filled from the wrong source for its position, is a load
error.

An Auth scheme's parameters were a Capability-relevant position while a scheme might name a host of its
own — a token endpoint to exchange against. ADR-0031 removed that possibility rather than constraining
it, and the position went with it: a scheme decorates a request and never performs one, so nothing in
`auth:` reaches anywhere. A credential slot survives as a value shape in a Target declaration
(`credential-slot-malformed`, §4) rather than as a hole position anywhere.

## The `over:` forms

**Closed.** Three forms — `assets:`, `observations:`, and `values:`.

`assets:` and `observations:` expand over the Step's own Definition (ADR-0012) and Target's Record
series; `observations:` is legal only on a `read` Step, since Expansion is scoped by Kind rather than
by Record type (ADR-0027). `values:` is a literal enumerated list of bare scalars authored in the
Procedure, occupying lines the gutter annotates and counted by the Bound like any other selector — the
syntax for reaching something `hyper` did not create by literal identifier before it can be changed or
ended. It is legal on all three Kinds: the Kind rule scopes which Record types a selector may range
over, and `values:` ranges over no Record type at all, so there is nothing there for it to restrict.

A `values:` member is a host where the Step wires it into the Operation's `host-input:` and an
identifier where it fills any other input — decided by position, like every other question this format
asks of a value (§3). Where they are hosts, `check` verifies offline that the list is a subset of the
bound Target's granted host set (ADR-0024).

Where a `destroy` Step's `values:` member names no series, the Tombstone it writes opens one under that
member as the Record name (§7, ADR-0033). It is the one Record name in the system whose origin is an
author rather than an upstream response, and §13 states the limit that follows.

## `THE CODE MOVED` change classes

**Closed.** Nine facts about code, plus a catch-all. The Comparison's third table (§8) emits one row
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
- **the credential source** — the environment variable each of a Target declaration's credential slots
  resolves from. Names only, never values: §6's whole reason for naming variables explicitly rather than
  deriving them is that `env: STAGING_TOKEN` → `env: PROD_TOKEN` is a visible one-line edit, and without
  a row here that edit reaches the Comparison only as one line in the catch-all count. A Target
  declaration is in no Record's Provenance, so `the digests` below does not carry it. The secret behind
  an unchanged name rotating is a world fact `hyper` deliberately cannot see (§7, ADR-0007), which is
  why this class is about the name and stops there. The fact belongs to a Target rather than to a
  Definition, as the repository revision belongs to neither, and §8's `SUBJECT` column is where that
  lands: a kind-qualified name, `target staging`, one row per subject.
- **the Operation set** — the Operations a Manifest exposes, and the `destroy` Operations a Definition
  names.
- **the digests** — every member of the Provenance, the full set a Record version carries (§7). Stated
  intensionally on purpose: a member joining that field set brings a row here without this enumeration
  moving, which is how the Procedure revision arrived (ADR-0048) and why there are still nine. A class
  reading *every member except one* would be the enumeration ceasing to be checkable. Each member takes
  its own subject in §8's column, the Procedure revision's being a Procedure, `hyper_version`'s the
  Repository declaration whose pin it cannot differ from (§11), and `repo_revision`'s no artefact at
  all — the one row in the table that renders `—`.

A class emits **one row per `(subject, fact)` pair** rather than one row: a class is a kind of fact, so
a Definition's declared Kinds and its Target declaration's accepted ones moving in one window is two
rows under one name here. §8's `SUBJECT` column carries a kind-qualified name — `target staging`,
`procedure retire-preview-envs` — and fixes each class's subject.

The catch-all terminates the table and is not optional: `N other lines changed · git diff <rev> <rev>`,
counting every line of every reviewed artefact that moved and that no row above reports — a widened
retention policy (§7) among them. `N` is in **`git diff` lines** as git counts them, a modified line
being two, because the row names the command and any other unit makes the row disagree with its own
evidence; the file set is the reviewed five and nothing else, the generated workflow among the
exclusions (§8). The command is suppressed where either side recorded `repo_dirty`, that being the one
case it cannot reproduce (§7, §8). The enumeration is what makes the table checkable; the catch-all is
what makes omission impossible.

## The `FLAGS` vocabulary

**Closed to `hyper`.** Eight names; no Manifest mints a flag any more than it mints an `error_code`
(ADR-0004). `FLAGS` is the one surface in the whole tool permitted to say *look here*, and what binds
every name is the relation §8 states: a flag cites a line the gutter already marked, and introduces no
claim of its own.

**The set is a rule before it is a list.** Every marker class the gutter carries indexes here, and the
eight names below are what that rule yields today: a marker class arriving in §8 brings its flag
without this enumeration moving, exactly as a Provenance member joining brings a `THE CODE MOVED` row.
`unresolved` is the first name to arrive that way, and it is evidence the rule works rather than a
decision to grow the set — §8 gained a marker class and this listing followed it (ADR-0064).
The names are written out anyway, so the set stays checkable against a rendering — an intensional rule
nobody can enumerate is a set that has stopped being closed.

That relation is what sizes the vocabulary rather than any judgement about what is worth saying: the
gutter is the whole supply, so `FLAGS` cannot name a fact the review does not already annotate in
place, and a name for one would be the editorial claim ADR-0026 removed.

### The standing names

Five, each reading on every artefact whose gutter marks the fact. They are facts rather than artefact
kinds, which is why there are five rather than five per artefact:

- **`destroy`** — `destroy` authority is claimed, granted, or exercised on the cited line. It reads on
  a Step whose Operation declares that Kind, on a Definition's claimed Kinds, on a Target declaration's
  accepted Kinds, and on a Manifest Operation's declared Kind.
- **`opaque`** — the cited line reaches an effect `hyper` cannot describe. It reads on a Manifest
  Operation whose request uses an Opaque Capability, on a Step invoking one, and on the opt-in by which
  a Target declaration admits an `opaque` `destroy` at all (§4).
- **`unbounded`** — an effectful Step with no Bound standing behind it: a `mutate` Step carrying no
  `bound:`, which the gutter marks `mutate!` (§8), and an `opaque` `destroy` Step, which may carry no
  Bound at all and where a Bound is refused (`bound-illegal`, §4). The second is implied by `destroy`
  and `opaque` together and renders regardless: §5's argument is that *unbounded* is the accurate word
  for it, and a surface silent on the strongest instance of the fact it indexes is omitting rather than
  economising. It is Procedure-only, `bound:` being a Step's key.
- **`envelope`** — a Procedure's declared Target and Kind envelope, and whether every Step is inside it.
  Procedure-only likewise, and the one name whose all-clear form renders (below). A review does not run
  `check` (§9), so an artefact carrying `envelope-exceeded` renders like any other and this flag has two
  states rather than one.
- **`unresolved`** — a name on the cited line resolves to nothing, so the gutter had no derivation to
  mark there (§8). Its row names which name failed and the path `hyper` looked for, the gutter marking
  and not classifying exactly as it does for a change. Procedure-only, and for a reason that is not
  `unbounded`'s: a Definition names a Provider too, and nothing on a Definition's screen is derived
  from a Manifest, so an absent one costs that rendering nothing (§8). It renders only where the fact
  holds, `envelope` remaining the one name with an all-clear form — and like `envelope` it indexes a
  fault without becoming one, a review not running `check` and exiting 0 however many flags it carried.

### The change names

Three, reading on the lines the gutter's change column marks as touched since the review's range opened
(§8) — including the line a deletion anchors to, which the column marks and which did not itself move.
They are directions rather than classes, the class being carried in the row's text:

- **`widened`** — the cited line grew what the artefact may reach.
- **`narrowed`** — it shrank it.
- **`changed`** — it moved, and no direction is claimed.

All three read the baseline as well as the marked line, which is the gutter's supply and not a reach
past it (ADR-0057): a review renders the working tree, so the value a direction is measured against is
never on screen and the flag's text is where it renders.

**Direction is claimed exactly where it is mechanically decidable, and never where it is not.** That is
numeric comparison for a Bound and set inclusion for declared Kinds, Target sets, required Capabilities,
and the `destroy` Operations a Definition names. It is not available for a selector, a credential
source, or a Cadence, each of which takes `changed` and its full before-and-after text: predicate
subsumption is undecidable in general, so a surface calling `equals: preview` → `starts_with: preview-`
a widening would be inventing the one thing it may not invent. The classes these read on are the nine
`THE CODE MOVED` classes above less **the digests**, which is Run-recorded and has no line in any
artefact — one vocabulary across both surfaces, and the difference between them is that a review reads
a file and a Comparison reads two Runs.

`narrowed` earns its place on symmetry rather than on usefulness. Rendering `widened` while folding
every narrowing into `changed` would be the surface deciding that one direction is worth a name, which
is a judgement about severity and not a fact the gutter carries.

### Order, and the empty state

**Rows render in line order, with a file-level row last** (ADR-0054). `envelope` is the only file-level
name today, so it is the last row of every Procedure review.

A review whose artefact draws no flag renders the block with an explicit empty state rather than
omitting it. Only a Procedure is guaranteed a row — `envelope` always renders and always names its
state — so on the other four artefacts an absent block would be ambiguous between *nothing to flag* and
*the renderer had nothing to say*, which is the ambiguity §8 already refuses to leave standing between
its two named absences.

### Why the set is closed and stays small

`FLAGS` is an index, not a voice. Every row it carries is derivable from the `gutter` rows directly
above it, and the surface exists because derivable is not legible: the gutter is per-line and unranked,
and on a long Procedure the flag block is the only thing that puts the lines that matter on one screen.
It carries a second reader the gutter is the wrong granularity for — the `flag` rows of `review --json`
are what a reviewing agent reads to decide whether to escalate to a human, on artefacts another agent
wrote.

What is rejected is anything that reaches past the gutter for a reason. An Operation's Repeatability is
the nearest miss: `skip-if-recorded` trusts the record over the world (§13), which is worth a reviewer's
attention and is not blast radius — it changes nothing about what a Run may reach. Admitting it starts
the slide from *an index of what the gutter marked* to *a list of things worth thinking about*, which is
the slope ADR-0026 sits at the top of.

## The review header's absent baseline

**Closed to `hyper`.** Two names, carried by an `artefact` row's `baseline_absent` where a review has no
range: `not-run`, the Procedure has no non-rehearsal Run in the Journal; and `no-store`, the Store is
unreachable. They are the two absences §8 renders as two sentences on the argument that an artefact
that has never run and a repository that cannot answer are different facts — a key merely missing would
put them back under one reading on the wire, which is the two surfaces stating different things.

Reuse was considered and rejected. `store-absent` is an `error_code` and a review does not decline, so
naming this with one would point a reader at a Refusal that did not happen. What earns the pair a place
here rather than in §8 is that these are names travelling as values, which is what this chapter holds;
the header's own membership is a list of keys and is stated beside the rendering in §8, where the
gutter's marks already are.

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
