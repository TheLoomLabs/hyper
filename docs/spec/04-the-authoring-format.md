# §3 — The authoring format

All five reviewed artefacts — Definition, Procedure, Manifest, Target declaration, Repository
declaration — are written in one format: YAML 1.2 core with most of the language rejected at load
(ADR-0023), carrying no expression language anywhere (ADR-0022). The authoring format inherited this
shape from the review surface rather than from taste: the gutter that annotates a Definition assumes a
Step occupies a run of lines a marker can sit beside, which a single-expression or non-line-oriented
format collapses back into a derived table (ADR-0026).

Every artefact lives at a fixed, `hyper`-owned path and carries a `kind:` key that must agree with its
directory. The five values, their directories, and the one exception — `hyper.yaml` at the repository
root, agreeing with its filename rather than a directory — are a closed set defined in §12. The mapping
is declared and derived at once: a file whose directory and `kind:` disagree is a load error.

## The YAML subset

YAML is parsed strictly: anchors, aliases, merge keys, tags, multi-document files, implicit type
resolution, and unknown keys are all rejected at load (ADR-0023). A scalar's type comes from the schema
at the position it occupies, never from what it looks like, so nothing resembling the Norway problem
can arise — there is no untyped position for it to arise in, save the one `body:` is and the boundary
stated with it below (ADR-0078). A file is YAML *syntactically* and not
*semantically*: editors and language servers still parse it, but every construct that would let one
line's meaning depend on another line (an alias) or on data the format's own grammar does not show (a
tag) is refused before anything is read for meaning.

Key order within a file is free and nothing checks it. The layout ADR-0023 fixes is where files live,
not the order of keys inside one, and a reordering renders as what it is: a line that moved, counted by
the Comparison's catch-all (§12) and reviewed against the previous revision like any other edit.

## Lists and mappings

One rule decides the shape of every collection the format itself names. A collection of **named** things
is a mapping keyed by that name — a Manifest's Operations, a projection's fields, a Target declaration's
credential slots, an Operation's Patterns. A **list** appears in exactly two places: where order is the
meaning, which is a Procedure's `steps:` and nowhere else, and where the members are bare scalars, which
is `targets:`, `hosts:`, `kinds:`, `capabilities:`, a Definition's `destroy:` claim, a schema's `enum:`,
and a selector's `values:`. A value an Operation's input schema governs is outside this rule and shaped
by that schema instead, which is how a `shell` Step's `command:` is a list of argv words with references
among them (below).

A mapping makes a name unique by construction, so there is no duplicate-name rule to state, to name a
check for, or to forget. It also gives each name a line of its own, which is what the gutter annotates —
the same reason a Step in an ordered list carries an `id:` on its first line.

## Names

Every artefact but one carries its own name, on a key repeating the word its `kind:` names: `provider:`,
`definition:`, `procedure:`. A Target declaration carries `target:`, because the value names the Target
rather than the declaration and a Definition's `targets:` list names the same thing. `hyper.yaml` carries
no name key: one repository has one Repository declaration, and there is nothing to tell it apart from.

A name agrees with its file's basename, on the argument ADR-0023 makes for `kind:` and the directory,
applied to names rather than kinds: a basename alone leaves the review surface's first lines silent
about what is being read, and an authored name alone lets a file be renamed while the `<definition>`
segment of every path in the Store stays where it was (§12). A disagreement is `name-mismatch` (§4). A
built-in Provider ships inside the binary and has no file, so it authors its name outright (§11).

A name one artefact writes for another resolves against that artefact's own `name:` — byte-exact over
UTF-8, case-sensitive, never against a filename and never settled by whether an `open` succeeded. It is
the rule ADR-0060 states for a name a user types, applied to a name an author wrote, so *resolves to
nothing* means one thing across both and means it identically on a laptop and on a runner. An artefact
whose own file will not parse is present all the same: it declares a name, its own fault is reported
once on its own line, and every artefact naming it correctly is untouched. Failing to resolve is a
check rather than a failure to load — `artefact-absent` where the namespace is the repository's
artefacts and `reference-unresolvable` where it is what an artefact declares (§4, ADR-0064).

## Types

An Operation's input schema is written in a subset of JSON Schema, closed and defined in §12. An
Operation's output carries no schema at all — only the projection a Manifest declares, so cardinality,
identity, and every recorded field are stated exactly once rather than in two representations that can
disagree.

The scalar vocabulary is closed and defined in §12: the common JSON Schema primitives plus two the
domain forces, a `duration` and a `timestamp`. There is no `null` among them — a field's presence is a
predicate fact (`exists` / `absent`), never a type.

**A scalar is read against the schema at its position; it is never compared with it.** ADR-0023 types a
scalar by that schema rather than by what it looks like, and reading is what that means: the value's
characters are read as the declared type, in the text form §12 fixes for it, and the quoting YAML
required is lexical rather than part of the value. `"2592000"` and `2592000` are one value at an
`{type: integer}` position, and `NO` at a `boolean` one reads as nothing at all, `true` and `false`
being the whole of that type's text — which is how the Norway problem stays dead without a blocklist
anywhere (ADR-0081). §12's text-form column states the reading for every scalar type: it was written
for what leaves on the wire and is one rule read in both directions.

Characters that will not read that way, an input the schema declares that the caller did not supply, and
a value outside its `enum` are one check — *a value satisfies the schema at its position* — and one code,
`schema-mismatch` (§4). It reaches every position of every artefact, `hyper`'s own schemas and an
Operation's input alike. `body:` is the one position it does not reach, there being no schema there at
all (below, ADR-0078).

**Every input an Operation declares is supplied.** There is no `null` in the vocabulary above, no
key-omission syntax anywhere in the format, and therefore no sink at which an unsupplied input could
render: a `body:` key with nothing to fill it is not writable, and a request whose keys varied with a
Step's `args:` would put the request's *structure* in the Procedure, which is the reading ADR-0078
refused. `required` is not a keyword of the input-schema subset for that reason — the property list is
the whole of it, and a keyword whose only honest value is *all of them* is the second spelling this
format removes everywhere else. An API with a genuinely optional parameter is two Operations, and §13
states the cost.

## No expression language

There is no interpolation syntax, no operator, no function, no arithmetic, no regular expression, and
no boolean algebra anywhere in the format (ADR-0022). Three closed forms carry everything the five
artefacts need to say instead: a path, a predicate, and a template hole.

A **path** names a value living elsewhere — a field of the response object `hyper` builds from an
Operation's call (§12, ADR-0040), where a Manifest states its projection, or a field of a Record, where
a Step references one. Its grammar is closed and defined in §12. Where a path appears in a value
position rather than a projection, it is embedded as a
**reference** — a mapping, never a string: `{step: <id>, path: <path>}` names an earlier Step's Record,
`{item: <path>}` names the Record the Step is itself ranging over — the one a selector or predicate is
filtering, and, in the `args:` of an expanding Step, the one its Expansion is acting on, which is how
an effectful Step addresses what it expanded to — and there is no third form. A reference may appear
only where the schema at that position expects a scalar, which is what makes "is this a literal or a
reference" a type question rather than a parsing one — a whole object can therefore never be
referenced. A reference naming an earlier Step whose declared cardinality is
`series` is a load error: pairing an expanding Step against a stored series is a join by identity
between two Record series, and no such join is ever performed. The shape that is writable names one
Record — a Step of cardinality `one` produces something, and a later Step references it directly.

A reference resolves in both halves, and one code carries both. Its `step:` names an `id:` the same
Procedure declares earlier; its path names a field the Record it points at actually carries, the field
set of a Record series being Manifest-declared. Either naming nothing is `reference-unresolvable` (§4)
rather than a Run that fails partway through — which is what makes the projection rule below safe to
state.

A **predicate** filters a set of Records, in a Step's selector (`over:`), its condition (`when:`), or a
polling Pattern's terminal condition. The operator set is closed and defined in §12, with the operand
types each takes, what the comparisons mean, and the one instant the two temporal operators read
against. Each entry carries a `field:` and exactly one operator; two operators sharing an entry would be
an AND that occupies one line where the list already gives each conjunct its own, which is what the
gutter annotates and what the Comparison renders a selector change from. A predicate list is always AND;
there is no disjunction anywhere in it (ADR-0022).

A `field:` is one of two things, decided by the root it is written under. At the two Record roots — a
selector and a condition — it names one key of the Manifest's `fields:` mapping and nothing else: a
Record's field names are flat and authored, so there is no path there to write. At a polling Pattern's
`until:` it is a path in the grammar above, written without the root marker, a response having paths and
no declared names. A `field:` naming what no Operation of the Provider projects is
`reference-unresolvable` (§4) on the rule a reference already follows, and a predicate that cannot
decide Refuses rather than quietly not matching (§12, ADR-0035).

A **template hole** fills a scalar position inside an otherwise literal value, one hole syntax across
every artefact: `{name}`, naming what fills it and nothing more, and a value beginning with one is
quoted (ADR-0023). What a hole may resolve to is decided by the position it appears in, never by
scanning the resolved value afterward for something dangerous. The legal positions are a closed set
defined in §12, and which positions of a request are Capability-relevant is stated below.

### The credential slot

A credential slot's value is a mapping whose sole key is `env:`, naming the environment variable
`hyper` resolves it from at Run start. A scalar in a credential position is always a load error, with
no exception. `env:` may appear nowhere else in any artefact — a general ambient-input channel is
authority arriving after review under another name (ADR-0008).

### `local` and the host grant

`local` is a Target declaration the repository authors, in `targets/` beside every other, and `hyper`
ships none (ADR-0041). It enumerates the hosts it grants like any other Target's (ADR-0024): a
Capability-relevant hole resolving against it is checked the same way, and a `values:` list of hosts is
checked against the same enumeration (§12, the `over:` forms).

What the name reserves is the Target a Probe binds (§9), and nothing else about the file. A declaration
named `local` declares `class: local` and carries no `auth:` block, and one doing either is
`local-reserved` (§4) — the second being what leaves a Probe no credential to resolve. More than one
declaration claims `class: local` where a repository has reason to: a class only ever rejects a
mismatch, so two of them are two names for the machine `hyper` runs on, each with its own grant, its own
accepted Kinds, its own credential slots, and its own `opaque-destroy:`.

Which host a request reaches is decided by a candidate set, a grant, and their intersection (ADR-0029),
stated with the request below.

## The five artefacts

### Provider

`kind: provider`, in `providers/`. A Provider is a Manifest and nothing else. Its top level carries its
name, an explicit `schema-version:`, the `class:` of Target its Definitions may bind, the
`capabilities:` it requires — once for the Provider and never per Operation, `hyper` deriving the
per-Operation ones (§4) — the `auth:` scheme it authenticates with, if it authenticates at all, any
`enumerations:` its Capability-relevant holes draw on, and `operations:`, a mapping keyed by Operation
name.

A Manifest alone carries an explicit schema-version field; the other four artefacts carry none, since
the repository-wide version pin already fixes which binary reads them and a Manifest is the one artefact
authored by someone outside that pin's reach (ADR-0023). An installed Manifest carries one further
block, written by `hyper` rather than authored: `origin:`, holding the registry `ref:` and `digest:`
`install` verified it against (§11). A Manifest carrying none is a locally authored Provider.

Each Operation declares, on flat keys, the facts `hyper` would otherwise have to guess at: its `kind:`,
its `repeatability:`, its `deadline:`, its `concurrency:` limit, the `patterns:` it uses, its `input:`
schema, its request, and its `record:` projection. None of them is nested under a grouping key: the Kind
is the most review-relevant fact in the file and an indent is what would put it behind something else.

Whether an Operation is `opaque` is not among them, being nothing `hyper` has to guess at: opacity is a
property of the Capability an Operation's request uses (§12), so the request's own key already carries
it and a second spelling could disagree with the first. Every surface still renders it (§9); no artefact
writes it.

`concurrency:` is a `read`'s and no other Kind's. An effectful Expansion runs strictly serially with no
authored knob anywhere in it (§6), so a limit declared on a `mutate` or a `destroy` would govern nothing
from the moment it was written — which the Manifest decides on its own, with no Step in hand — and it is
refused as `manifest-inconsistent` (§4), the code that already refuses a `pagination` Pattern outside an
`over:` and a `record:` on a `destroy`.

Four of them are stated by omission. `repeatability:` omitted is the default the Operation's own
Kind fixes — run-once where it effects, `repeatable` on a `read` — which §12 states in full, and
neither is a value to write. `concurrency:` omitted is **1**, and a `read` whose Manifest says nothing
runs its Expansion serially: how many requests an API tolerates at once is something only the Provider
author has measured, and every other number `hyper` could put there is a guess about a system nobody
described (ADR-0045). An explicit `concurrency: 1` is legal and means exactly what the omission means —
a limit is a number rather than an enumeration, so 1 is an ordinary member of its value space and not a
second spelling of silence, and an author who has established that an API refuses concurrency may say so
instead of leaving it to be read as an oversight. `input:` omitted is an Operation that takes none, and
a Step binding it writes no `args:`: `additionalProperties: false` is forced at every level (§12), so
the omission refuses every argument without a second rule saying so, and an explicit empty schema means
exactly what the silence means. Record cardinality has no key at all — `series` is a
`record:` carrying
`over:`, and `one` is a `record:` without it, since a `series` projection cannot omit the collection
path and a `one` projection has nothing to put there. A `destroy` Operation carries no `record:` and
declares no identity, what it writes being a Tombstone under the series its Expansion acted on — or,
where that Expansion was a literal naming no series, under a series the Tombstone opens (§7,
ADR-0033).

```yaml
kind: provider
provider: cloudflare-dns
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  create_dns_record:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
      body: {name: "{name}", type: "{type}", content: "{content}"}
    input:
      type: object
      properties:
        zone_id: {type: string}
        name: {type: string}
        type: {type: string, enum: [A, AAAA, CNAME]}
        content: {type: string}
    record:
      identity: "{name}"
      fields:
        id: $.body.result.id
        name: $.body.result.name
        created_on: $.body.result.created_on
  list_dns_records:
    kind: read
    repeatability: repeatable
    deadline: 30s
    concurrency: 4
    http:
      method: GET
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records
      query: {per_page: "100"}
    input:
      type: object
      properties:
        zone_id: {type: string}
    patterns:
      pagination:
        cursor: {from: $.body.result_info.cursor, into: {query: cursor}}
      retry: {attempts: 3}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id, name: $.name, created_on: $.created_on}
  delete_dns_record:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /client/v4/zones/{zone_id}/dns_records/{record_id}
    input:
      type: object
      properties:
        zone_id: {type: string}
        record_id: {type: string}
```

The other Provider this specification renders is `uptime`, whose Definition §8 reviews and whose
Observations §8 compares. It is an ordinary locally authored Manifest rather than anything `hyper` ships
(§11, ADR-0039): no credential, one Operation, and a projection that reads the response object and never
touches a body. Read beside the one above it is the whole of the difference between a check and an API
call.

```yaml
kind: provider
provider: uptime
schema-version: 1
class: local
capabilities: [http]
operations:
  check_http:
    kind: read
    deadline: 10s
    http:
      method: GET
      host: "{from-target}"
      path: /
      host-input: host
    input:
      type: object
      properties:
        host: {type: string}
    record:
      identity: $.host
      fields:
        host: $.host
        status: $.status
        days_left: $.tls.days_left
```

`repeatability:` is omitted because a `read` has one legal value (§12), `auth:` because a public host
takes no credential, and `host-input:` is present because `{from-target}` expands to every host `local`
grants and a Step names which of them this Run is checking. `days_left` is absent from a version written
against a host that answered nothing at all, a field's presence being a predicate fact rather than a
type (§12).

Nothing here declares which statuses are acceptable, and that is not an omission: a `read` never halts
on what came back (§6, ADR-0050), so `status: $.status` records a `503` as readily as a `200` and this
Manifest describes *is this site up* without a second declaration saying what *up* is. Against a host
that answers nothing at all only `host` resolves, and the version carries an identity and no `status` —
the same absence, read the same way.

### Target declaration

`kind: target-declaration`, in `targets/`. The reviewed half of a Target, holding no credentials: the
`kinds:` it accepts, the `capabilities:` it grants, the `hosts:` it grants — one member where the Target
is a single endpoint, and never a grant without an enumeration (ADR-0024) — its `class:`, whether it
opts into `opaque-destroy:`, and an `auth:` mapping naming the environment variable each credential slot
resolves from. Every static check over a Target runs from this artefact alone, with no credential
resolved and no network reached.

`hosts:` is one list rather than a mapping keyed by Capability. The Capability set has two members and
one of them reaches no host at all (§12), so a per-Capability mapping would be a key over the only list
it could ever hold.

It is present exactly where `capabilities:` grants `http` and absent where it does not, either
disagreement being `target-inconsistent` (§4): a grant of `http` over no host reaches nothing, and a list
of hosts beside a Capability that reaches none grants nothing to anything.

```yaml
kind: target-declaration
target: cloudflare-prod
class: cloudflare
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.cloudflare.com]
auth:
  token: {env: CLOUDFLARE_API_TOKEN}
```

The other Target this specification renders is `local` — the Target the Definition §8 reviews binds, and
the one the Observations §8 compares are written against. It is the smallest artefact here: the class its
Providers name, two public hosts, `read` and nothing further, no credential slot to cover, and no
`opaque-destroy:` because nothing bound to it is opaque. A repository that reaches no host without
credentials and runs no command carries no such file at all (ADR-0041).

```yaml
kind: target-declaration
target: local
class: local
kinds: [read]
capabilities: [http]
hosts: [status.hyper.dev, cert.hyper.dev]
```

Read beside the one above it, the whole of what the reserved name changes is the `auth:` block that is
absent because it may not be written. `shell` is absent from `capabilities:` for the ordinary reason any
Capability is: nothing bound to this Target needs it (§4).

### Definition

`kind: definition`, in `definitions/`. A named, authority-scoped use of one `provider:`: the `kinds:` it
claims for `read` and `mutate`, the named `destroy:` Operations it claims — granularity follows severity,
so a `destroy` claim names Operations rather than a Kind — and the `targets:` it may bind, named
literally rather than by class or tag, since a Target class only ever rejects a mismatch and never
expands a Definition's reach.

Both name things that must be there. A `provider:` resolves against the built-in Providers and
`providers/` together (§11), and a `targets:` member against `targets/`; either naming nothing is
`artefact-absent` (§4). A `destroy:` member names an Operation of that Provider, and one naming nothing
it declares is `reference-unresolvable` — the Manifest's own namespace rather than the repository's, and
a different fault from `operation-not-claimed`, which is a Step reaching an Operation that exists and
this Definition did not claim (§4).

`read` may not appear in `kinds:` beside `mutate`, nor beside a `destroy:` claim naming any Operation: a
Definition observes or it effects (ADR-0032). `mutate` beside `destroy:` stays legal and is the ordinary
case, a Tombstone landing in the series the `mutate` created (§7). What a Definition may claim is
therefore `read` alone, or `mutate`, or `destroy:` Operations, or those two together.

`destroy` is not a member of `kinds:`. The two keys are what let the review's `AUTHORITY` table derive
the claimed Kind at that one position rather than read it (§8). A Definition carries no argument value
of its own: the gutter renders what reaches the world from the file being read, and a value living on
the Definition would force a second, `AUTHORITY`-shaped table just to re-show it (ADR-0026). Those
values belong to the Step.

```yaml
kind: definition
definition: preview-dns
provider: cloudflare-dns
kinds: [mutate]
destroy: [delete_dns_record]
targets: [cloudflare-prod]
```

Reading what that Definition creates is a second Definition, against the same Provider and the same
Target. The Definition is the segment of a Record's identity that keeps the two series apart, so the
Observations one writes and the Assets the other owns never meet.

```yaml
kind: definition
definition: preview-dns-observed
provider: cloudflare-dns
kinds: [read]
targets: [cloudflare-prod]
```

### Procedure

`kind: procedure`, in `procedures/`. An ordered `steps:` list and the full set of `targets:` the
Procedure and everything it invokes may touch, authored rather than derived so a reviewer can see the
envelope without tracing every nested invocation. A Procedure's Cadence is authored here too, on
`cadence:`, as a quoted string whose grammar belongs to §10; a Procedure carrying none is run by hand.

Each entry in `steps:` is either a Step or a nested Procedure invocation, in the same list — never a
separate block, since a second list would put ordering between two structures and reintroduce the graph
Steps are sequenced without (ADR-0002). A Step names its `id:`, its `definition:`, its `operation:`, the
`target:` it binds, the `args:` the Operation's input schema requires, an `over:` selector, a `bound:` on
the Records an effectful Step may affect, and a `when:` condition rooted at an earlier Step's Record,
carrying `step:` beside `field:`. A nested invocation names an `id:` and a `procedure:` in place of
`definition:`/`operation:`/`target:`, and its Steps render under the invoking Step's path with the
invoked Procedure's transitive envelope. A comment is permitted on any line, rendered verbatim in place
on the line it was written on and never read by `hyper`; it is source rather than annotation and never
enters the review's gutter, which carries only what `hyper` derived (§8). No directive syntax may ever
exist inside one, since that would be a bypass wearing a comment.

A Step's `definition:` resolves against `definitions/` and a nested invocation's `procedure:` against
`procedures/`, either naming nothing being `artefact-absent`; its `operation:` names an Operation the
Definition's Provider declares, and one naming nothing is `reference-unresolvable` (§4).

`target:` is written on every Step and is checked for membership of its Definition's `targets:` list —
which is the whole of what it is checked against, the members of that list being where a Target's
existence is already a question. A Step binding one the Definition does not claim is
`target-not-claimed` (§4), on the shape `operation-not-claimed` already has: a Definition claims
Operations and it claims Targets, and a Step naming outside either claim reads the same way. It
is not derived from a Definition naming one Target: the gutter annotates what is in the file being read
and a table is what carries a fact assembled from elsewhere (ADR-0026), so a Target rendered beside a
Step whose source does not name it inverts the one rule the review surface is built on. Nor is it
written only where a Definition names several, which would be a key whose presence in one artefact
depends on the contents of another.

`over:` takes one of three forms, closed and defined in §12, and a Step declaring none is invoked
once. All three range over something already written down — Records in the Store, or identifiers
authored in the Procedure — so a Step calling an Operation for the first thing it will ever record
has nothing to range over and says so by omission.

A `values:` member is a bare scalar and `{item: $}` is the whole of it, `$` with nothing after it being
what the path grammar already allows. A member is not a mapping: a mapping in a scalar position means a
reference in this format, and a compound identity needs none — the shared half of it is an argument and
only the varying half is a population, which is what `retire-preview-dns` below shows with a literal
`zone_id:` beside a `record_id:` that moves. Whether a member is a host or an identifier is decided by
the position the Step wires it into: `{item: $}` in the Operation's `host-input:` makes the list a list
of hosts, checked against the Target's grant like any other candidate set (§4, ADR-0024), and `{item:
$}` in any other input makes it a list of identifiers, which reach nothing on their own. Position
decides it for the reason position decides a hole's legality and a credential slot's: an author
declaring it instead would be stating a fact `hyper` reads off the wiring, and a wrong declaration would
disarm the one offline reach check a `values:` list has.

A member repeated in one list is a load error, two members differing only in case being the same fault
under the fold the Store already applies. It carries the Store's own code, `record-identity-collision`
(§12), because it is the Store's own check — two things that are one identity — found one Run earlier,
against an artefact instead of against a branch. The list is walked as
authored (§6), so a repeat is not a set carrying a redundant element but a second call on one identity —
under a `destroy`, one going out against a thing this same Run entombed, since the drop rule reads heads
once at Expansion (§5) and never between two members of one list. It would also make the entry §7 writes
lie: `expanded_to` holds the sequence and the identity set holds a set, so three expanded to and two
concluded about would read as *one unaccounted for*, the phrase reserved for a call that may have
reached the world. Refusing it at load keeps the authored length the count §4 checks against `bound:`,
with no distinct-member caveat anywhere.

```yaml
kind: procedure
procedure: retire-preview-dns
targets: [cloudflare-prod]
cadence: "0 3 * * 1"
steps:
  - id: publish
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: preview-42.example.com
      type: A
      content: 203.0.113.10

  - id: publish-aliases
    definition: preview-dns
    operation: create_dns_record
    target: cloudflare-prod
    over:
      values:
        - docs.preview-42.example.com
        - api.preview-42.example.com
        - cdn.preview-42.example.com
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      name: {item: $}
      type: CNAME
      content: preview-42.example.com
    bound: 3

  - id: retire
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      assets:
        - field: name
          starts_with: preview-
        - field: created_on
          older_than: 14d
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $.id}
    bound: 5
```

`publish-aliases` is the same `skip-if-recorded` Operation expanding, and it is where that value's test
does more than it can on `publish`. The Operation's identity is the hole `{name}` and the Step fills that
input from the member, so each member is one series and the test asks its question three times. On the
first Run all three are created. On the second nothing is: three heads stand, every member skips, and
the Step is *skipped as already recorded* (§6). Add a fourth member and that Run creates the fourth and
skips the other three — the Step is *ran*, one call went out, and the identity set holds all four
(§7, ADR-0056). A member the Cadence's earlier occurrence left Tombstoned is created again, since the
test asks whether an Asset stands rather than whether the series exists (§12, ADR-0011).

The same Step written against records another tool created reaches them by literal identifier instead.
Each member is one Record name: the Step destroys three things `hyper` never built, and writes three
Tombstones, each opening the series it ends (§7, ADR-0033). `check` reads the list's length against the
Bound here rather than leaving it to the Expansion, the count being authored (§4).

```yaml
  - id: retire-legacy
    definition: preview-dns
    operation: delete_dns_record
    target: cloudflare-prod
    over:
      values:
        - 5b2d84f16c0a39e7d5182bfa604c7e93
        - 8f1a2c4d6e8b0a2c4e6f8a0b2c4d6e8f
        - c3d5e7f9a1b3c5d7e9f1a3b5c7d9e1f3
    args:
      zone_id: 023e105f4ecef8ad9ca31a8372d0c353
      record_id: {item: $}
    bound: 3
```

### Repository declaration

`kind: repository-declaration`, at `hyper.yaml` in the repository root — the one artefact with no
directory, agreeing with its filename instead, and the one with no name. It admits only facts that
govern the repository as a whole and belong to no Procedure, Definition, or Target declaration: the
`version:` pin and its `digest:`, written only by `hyper project`, and the `retention:` policy that
bounds Compaction.

`retention:` is a duration. What the policy bounds is how long a Record's interior versions are kept,
which is what `hyper` is asked and what Compaction acts on (§7); a count would keep a week of a
five-minute Cadence and four years of a monthly one under one number. Omitted, nothing is ever removed —
a repository that has not stated a policy has not agreed to lose anything. The digest carries its
algorithm inline, so a reviewer reads which one produced it rather than trusting the file's silence.

```yaml
kind: repository-declaration
version: 1.4.0
digest: sha256:9f2c1b7a4e6d038c5b1f92a7de40cb83f5710e2d9a6c4b83fe012d75c9a4e6b1
retention: 90d
```

## The request

An Operation's request is written under a key naming the Capability it uses, and an Operation uses
exactly one: `http:` or `shell:` (§12). The block's key *is* the Capability, so the declared-equals-derived
check §4 states reads one key rather than inferring a Capability from the shape of what it finds — and a
Manifest declaring a Capability no Operation's block names, or naming one it did not declare, is
`capability-mismatch` in either direction.

An `http:` block carries `method:`, `host:`, and `path:`, and optionally `query:`, `headers:`, and
`body:`. `query:` and `headers:` are mappings of name to string, always — which is not a rule this
format imposes but one the wire does: a query string and a header field are text, so there is no other
type to carry into and schema-directed typing has nothing to guess at. `body:` is the one position in a
request that is not text, and it is stated below. A form-encoded, XML, or raw body is not writable, and
joins the limits §13 states rather than becoming a second serialisation this format learns.

`path:` is written as text and `hyper` percent-encodes it — `url.URL`'s `Path`, not a second convention
— so an author writes the path a reader would say out loud and the escaping is `hyper`'s. Two characters
therefore cannot mean there what they mean in a URL, and neither is silently accepted. A `?` does not
open a query: it is escaped into the path as `%3F`, and what an author meant by it belongs in the
`query:` key beside `path:`. A `#` does not open a fragment: a fragment is a client-side construct that
is never sent, and there is no key it belongs in. A `path:` carrying either is `manifest-inconsistent`
(§4), cited at the `path:` line, and the `?` is what the row names where a path carries both — in a URL
the `?` opens the query and the `#` is inside what the author wrote as one. The cost is stated rather
than left to be found: a path *segment* holding a literal `?` or `#` is not writable inline and joins the
limits §13 states, and what remains is the hole, whose value arrives at Run time, is read against no path
grammar, and is escaped like any other text ([ADR-0107](../adr/0107-a-query-string-in-path-is-refused-where-it-is-written.md)).

The five headers `hyper` computes for itself — `Host`, `Content-Length`, `Content-Type`,
`Transfer-Encoding`, `Connection` — are reserved against every writer rather than against an Auth
scheme alone, and a `headers:` entry naming one is `header-reserved` (§4), compared case-insensitively
as an HTTP header name is. `Host` is what shows the rule is not hygiene, `hyper` deriving it from the
value the Target's grant was checked against (ADR-0029). `Content-Type` is where it holds the line §13
draws: `hyper` writes it because it serialised JSON, and an author able to overwrite it would reach a
form-encoded body not by writing one but by mislabelling the one `hyper` sent.

### The host

`host:` is a template, and it is always written. The value it carries is either `"{from-target}"`, which
is the ordinary case, or a literal with enumeration holes in it — `"s3.{region}.amazonaws.com"`. From it
three things follow in order (ADR-0029):

- **The candidate set.** `hyper` expands every hole's cross-product at load. `{from-target}` expands to
  the bound Target's granted host set; an enumeration hole expands against the `enumerations:` entry it
  names. The result is finite and known before anything runs.
- **The grant.** The candidate set is compared against the Target's `hosts:`, and a member absent from
  the grant is `host-not-granted` (§4, ADR-0024) — the same name the `values:` check carries, from the
  same comparison.
- **The intersection.** What a Run may reach is the candidates intersected with the grant. Where that
  is one host `hyper` fills it. Where it is several, the Operation's `host-input:` names which of its
  inputs carries the host, and the value that input carries is checked for grant membership like any
  other.

`host-input:` is a sibling key inside `http:` rather than a hole in `host:`, because a Capability-relevant
position resolves only to an enumeration or to `from-target` and never to an Operation input (§12). The
input it names always carries a whole host: an enumeration hole is a compact way of writing a large
candidate set, never a second thing filled at Run time. Naming an input the schema does not define is
`manifest-inconsistent` (§4).

**The scheme is `https` and there is no second one** ([ADR-0082](../adr/0082-the-scheme-is-https-and-there-is-no-second-one.md)).
No artefact chooses it: a `hosts:` grant enumerates hosts and carries no scheme, `host:` is a template
over those hosts, and there is no position in any of the five artefacts where `http://` could be
written. So `tls` is present on every response that arrived and absent exactly where the object is
`host` and nothing else (§12) — §12's *present where the scheme was HTTPS* is a condition with one
branch. A plain-HTTP endpoint is not reachable and joins the limits §13 states.

### Which positions are Capability-relevant

`host:` and an Auth scheme's parameters, and nothing else in a request. A grant enumerates hosts and
nothing finer, so reach is a host-level fact by construction and a hole in `path:`, `query:`, `headers:`
or `body:` cannot widen it past what the Target already granted; those positions resolve to an Operation
input, like every other position in the format (§12).

### `enumerations:`

A Capability-relevant hole resolves to a *declared* closed enumeration, and `enumerations:` is where a
Manifest declares one: a mapping of name to a list of bare scalars, at the Manifest's top level beside
`capabilities:`. It sits there rather than on an Operation because several Operations reach the same
hosts and because the cross-product `hyper` expands is a property of the Manifest. It is not the
`input:` schema's `enum`, which constrains a value a caller supplies and can never fill a
Capability-relevant position.

```yaml
enumerations:
  region: [us-east-1, eu-central-1, ap-southeast-2]
operations:
  list_buckets:
    kind: read
    http:
      method: GET
      host: "s3.{region}.amazonaws.com"
      path: /
      host-input: endpoint
    input:
      type: object
      properties:
        endpoint: {type: string}
```

### The body

`body:` is a JSON value tree, and it is the one position in any artefact `hyper` holds no schema for:
the shape is the API's, authored per Operation, and it lives outside this repository. The top level is
a mapping; a value is a scalar, a mapping, or a list; nesting is unbounded and a list member is itself
any of the three. A body whose top level is a list or a scalar is not writable and joins §13.

Because there is no schema at that position, **a literal scalar in `body:` carries its YAML 1.2 core
type onto the wire** — `false` is a JSON boolean, `2592000` a JSON number, `"2592000"` a JSON string.
This is the one place in the format where the artefact's own spelling types a value, and it is
schema-directed typing's boundary rather than an exception to it: the rule is refused nowhere, it is
unavailable here (ADR-0078). YAML 1.2 core is what keeps the Norway problem out — `NO` is a string, the
booleans are `true` and `false` with their case variants and nothing else, and there are no
sexagesimals. What it still costs is a leading zero: `0755` is the integer 755, so an identifier of that
shape is quoted.

**A template hole carries the declared type of the Operation input it resolves to.** A hole cannot
spell its own type — unquoted, `{expiry_seconds}` is a YAML flow mapping, so the quote is mandatory
(ADR-0023) and therefore says nothing — which leaves the input schema as the only thing that can, and
it is `hyper`'s to read, in the same file, a few lines below. An input declared `{type: integer}`
reaches the wire as a JSON number.

The hole is the **whole** of the value or it is not typed. `"{expiry_seconds}"` carries the input's
type; `"preview-{name}"`, `"{a}-{b}"` and `" {name}"` are compositions, which have no meaning but a
string, and the input is rendered into them in the text form §12 fixes for its type. The boundary is on
the line the reviewer reads, and so is its cost: a stray space changes the type on the wire and nothing
else changes with it.

A hole fills a value position only — every value of the tree, a list member included — and never a
mapping key. A key hole would put the request's own shape in a Step's arguments, which is ADR-0024's
*reach arriving from data* one aisle over on content instead of reach, and it would leave the body's key
set unenumerable by anything downstream. It is `hole-illegal` (§4), §12's hole positions being positions
and a key not being among them.

A hole may not name an input declared `object` or `array`. That is the rule a reference already carries
— a hole fills a scalar position, so a whole object can no more be interpolated than referenced — and
what it holds is that a reviewer reads a request's structure off the Manifest and its values off the
Procedure. One that does is `manifest-inconsistent` (§4): a Manifest's body and its own input schema
disagreeing, both in one file. An API needing a caller-supplied object in its body is not writable, and
joins §13.

`hyper` sets `Content-Type: application/json` because it serialised JSON, and the bytes it sends are
compact — no insignificant whitespace, keys in the order they were authored, and a number written as
the shortest decimal that round-trips, which is §7's own encoding rule reused rather than a second one
minted beside it.

```yaml
      body:
        description: "{description}"          # string in, JSON string out
        expirySeconds: "{expiry_seconds}"     # integer in, JSON number out
        label: "ci-{description}"             # a composition, so a JSON string
        capabilities:
          devices:
            create:
              reusable: false                 # spelled a boolean, sent a boolean
              tags: ["tag:ci"]
    input:
      type: object
      properties:
        description: {type: string}
        expiry_seconds: {type: integer}
```

reaching the wire, against `description: ci-runner` and `expiry_seconds: 2592000`, as:

```json
{"description":"ci-runner","expirySeconds":2592000,"label":"ci-ci-runner","capabilities":{"devices":{"create":{"reusable":false,"tags":["tag:ci"]}}}}
```

Every difference between those two blocks is readable from them: `expirySeconds` loses its quotes
because the schema three lines down says `integer`, `label` keeps them because a hole beside other
characters is a composition, and `reusable` keeps its type because nobody parameterised it.

None of this reaches whether the API wants the type the schema declares. That is the absence §4 states
and §13 names, and what this changes is its symptom rather than its existence: the Manifest now says
what it sends, so an API refusing a stringified integer is a wrong `type:` a reviewer reads beside the
body rather than bytes no surface shows.

### The command

A `shell:` block carries **no keys at all**, and is written `shell: {}`. The block's key *is* the
Capability (above), and there is nothing else for it to say. The request is a list of argv words, and
`hyper` execs it directly: nothing stands between the artefact and the process, so a pipe, a
redirection, a glob and an `&&` are not writable and join the limits §13 states (ADR-0051).

The words are the Step's rather than the Manifest's. `hyper`'s own `shell` Provider is the only one
that may declare this Capability (§11, ADR-0039) and it knows nothing whatever about the command, so
the argv **is** the Operation input named `command`, arriving in a Step's `args:` — which is what §13
means by *what bounds a shell Step is the words a reviewer read in the Procedure*. The Capability names
that input the way it fixes `cwd:`, `stdin:` and `env:` below, and for the reason each of those is
fixed: a key whose only legal content is the one `hyper` requires is a second spelling that can only
ever disagree with the first.

It is not a template hole, and it could not be one. A hole fills a scalar position and may not name an
input declared `array` (ADR-0078), so `command: "{command}"` against `{type: array, items: {type:
string}}` is a Manifest that does not load — `manifest-inconsistent` (§4) — which is what the built-in
Provider was written as before ADR-0081. What looked like an exception the hole rule needed is none: a
hole may not carry a request's shape into a Step's arguments, and `shell` is the one Capability whose
request has no shape of the Manifest's to carry.

**The first member is the reach axis, and it is a literal.** A reference there would put the choice of
binary in a value the world supplied, which is the arrival ADR-0029 closed for a host, reappearing on
the one Capability no grant bounds. A Step whose `command:` names its executable by reference — or
whose `command:` is empty, there being no executable to name — is `command-malformed` (§4), decidable
from the Procedure alone with no Store and no credential. Every member after the first is
referenceable, which is what makes an Expansion writable at all.

```yaml
  - id: disk-free
    definition: host-checks
    operation: read
    target: local
    over:
      values: [web-01, web-02, db-01]
    args:
      command: [ssh, {item: $}, df, -h, /srv]
```

Three commands, three Observations, one series per host, the Record's name being the command that
produced it — and `ssh` is read off the line rather than off a Record. The hosts here are argv words
and are checked against no grant: a `values:` member is a host only where the Step wires it into an
Operation's `host-input:`, and a `shell` Operation has none, which is §13's *the one Capability whose
reach no grant bounds* in its most ordinary form. §5 states what the same shape requires on a
`destroy`.

`cwd:`, `stdin:` and `env:` are not keys, and each is fixed instead. The working directory is the
repository root, so a laptop and a runner agree without a line saying so; stdin is empty; and the
environment is the one §11 states, the invoking environment with every credential-slot variable in the
repository removed. Each would otherwise be a further position a hole could fill on the Capability that
has no grant, and an authored `env:` would route a secret through an argument list.

## The Patterns

A Pattern is behaviour `hyper` performs around a call, which a Manifest parameterises and does not
implement. The three are closed (§12) and each is a key under an Operation's `patterns:` mapping.

All three are serial, and by construction rather than by a rule imposed on them: pagination's two forms
both terminate when the collection comes back empty, so there is no page after this one until this one
has answered; polling waits an `interval:` between calls; and a retry follows a failure. No Pattern
therefore has any concurrency to govern, and an Operation's `concurrency:` limit reaches none of them
(§6). That the serialism is constructional rather than declared is what makes it durable: a Pattern is
`hyper`'s own code, so a policy could be widened by a release and touch the world more times than the
artefact says with nothing appearing in a diff (ADR-0018), where a construction has nothing to widen.

**Pagination** takes one of two forms and no others. `cursor:` reads a token from a response path and
writes it into a named request position on the next call; `page:` writes an integer `hyper` increments,
from a declared starting value. Both carry `into:`, a single-key mapping naming the position — `query:`
or `header:` — and the name within it, which is a mapping rather than two flat keys so that exactly one
of a closed two-member choice is writable. Both terminate when the collection `record.over` names comes
back empty, and `cursor:` also when its path stops resolving. Pagination is legal only where `record:`
carries `over:`; declaring it elsewhere is `manifest-inconsistent` (§4).

A next-page URL read from a `Link` header or a response field is not a form. Reach arriving from data is
what ADR-0024 closed, and a URL a response hands back is exactly that — the population may come from
data and the reach only from an artefact.

**Polling** carries an `interval:` and an `until:` predicate list whose `field:` roots at the response
object §12 states, the same root a projection reads from.
It reuses the operator set §12 closes, rooted at a third scope, rather than growing a matcher of its own
— §12 already documents one operator set rooting differently by scope. It carries no attempt count and no
timeout of its own: the Operation's `deadline:` bounds the whole call, polls included, and a second bound
can disagree with the first.

**Retry** carries `attempts:` and nothing else. The failure class is fixed and provably pre-send
(ADR-0018), and backoff after a DNS failure or a refused connection is not a fact about the API a
Provider author describes — it is `hyper`'s, like the account of the attempts the Disposition carries
(§7).

```yaml
patterns:
  pagination:
    page: {from: 1, into: {query: page}}
  polling:
    interval: 5s
    until:
      - field: status
        equals: running
  retry: {attempts: 3}
```

## The projection

### What a response is

A response is not the bytes a server sent back: it is an object `hyper` assembles from the call it made,
and every path in a Manifest roots at that object rather than at the body (ADR-0040). There is one such
object per Capability, each closed and stated in §12, and which one a projection reads from is decided
by the key the Operation's request is written under. The `http` object has five members — `host`,
`status`, `headers`, `body`, and `tls`. A body field is therefore `$.body.result.id`, and a projection
reading nothing but `$.status` is a complete one.

`host` is the host the request reached — the one member of the object that is a fact about the call
rather than about the answer. It is there because a Record's identity is projected from the response and
an Operation whose answer carries no identity of its own has nowhere else to find one: an uptime check
over the hosts a Target grants writes one series per host, and the host is what tells them apart. It is
not a second spelling of the request's `host:`, which is a template; this is the single host the
intersection resolved to, which is also the only value a grant was ever checked against (ADR-0029).

`body` is the parsed JSON body and is **absent** rather than an error where the response carries none or
carries something else: a site that is down answers `503` with no body at all, and an uptime check is
pointed at hosts that answer in HTML, so the other reading makes the workload §0 opens with unrecordable.
The cost of that lands in §13 — an API that answers in XML can be called and cannot be projected. `tls`
is present where the scheme was HTTPS, and it carries a remaining-days figure beside the expiry because
no artefact could compute one, arithmetic being refused (ADR-0022), and because what it counts from is
the instant the Run fixed (ADR-0034) rather than anything a Manifest can name.

`status` and `headers` carry the same absence, and it is the one case that reaches every member at once:
where **no response arrived at all** — a refused connection, a name that does not resolve, a handshake
that failed — the object is `host` and nothing else (ADR-0050). `host` survives because it is the fact
about the call rather than about the answer, which is what lets a `read` record a host that answered
nothing: the Observation carries its identity and its `status` has gone quiet, and a field going quiet
renders as a change like any other (§6, §8). An effectful Operation halts there instead, no status being
not `2xx` (§6). There is no member saying *what* went wrong, on the ground ADR-0017 settled for
rendering — it would be the catch-all bucket, arriving on the object every projection reads from.

The `shell` object has four: `command`, `exit_code`, `stdout`, and `stderr`. **`stdout` and `stderr`
are text and are never parsed** (ADR-0052). A command that answers in JSON is recorded as the string it
printed, and `$.stdout.result.id` is not a path — which is the `opaque` trait arriving in the
projection, `hyper` being unable to describe what a command does and parsing its output being a
description of exactly that. Nothing new is needed to enforce it: the grammar §12 closes has three
productions and none of them reaches inside a scalar, so a shell projection reaches `$.exit_code`,
`$.stdout` and `$.stderr` and nothing finer. A shell response holds no collection either, so every
shell Operation is of `one` cardinality by construction rather than by a rule.

`command` is the argv as run, JSON-encoded on one line, and it is `host`'s member argument one Capability
over: a fact about the call rather than about the answer, present because an Operation whose answer
carries no identity of its own has nowhere else to project one from. The encoding is JSON rather than a
joining rule because it is injective — `[echo, "a b"]` and `[echo, a, b]` are two commands and must be
two identities, and a joining rule silently makes them one series that `record-identity-collision`
could never catch, the two names being genuinely equal.

Where the command **could not be started at all** — no such binary, not executable — the object is
`command` and nothing else, on the rule the `http` object already carries for a call that got no answer
(ADR-0050). `command` survives for the reason `host` does, so a `read` records the attempt and an
effectful Operation halts (§6).

### `record:`

`record:` is where a Manifest states what an Operation's response becomes. It carries `identity:`, a path
to the response field that is the Record's stable identity; `fields:`, a mapping of recorded field name
to path, which is the whole of a Record's projected content; and, for an Operation of `series`
cardinality, `over:`, the path naming the collection the Records are projected out of.

`record:` is **mandatory on a `read` and on a `mutate` and forbidden on a `destroy`**, either way round
`manifest-inconsistent` (§4). A `mutate` projecting nothing performs an effect `hyper` is accountable
for and puts no row in `YOU DID THIS` (§8), which is the one thing an effectful path may not do — the
argument ADR-0033 makes for a Tombstone, one Kind over. A `destroy` declaring a projection would be
declaring an identity for a Record it does not mint. What an Operation projects is therefore fixed by
its Kind, which is what lets §12 state which Repeatability values each Kind may declare (ADR-0037).

A Record's name is the value the identity field holds, so an Operation projecting a Record and declaring
no identity produces one nothing can identify, and declaring none is `identity-undeclared` (§4).

An Operation declaring `skip-if-recorded` carries one further requirement, and it is the only place
`identity:` is not free to be an ordinary response path. That value's test reads the head of the series
the call would write under, before deciding whether to make the call (§6, §12) — so the identity must
**resolve before the call**. A **template hole** has that property, resolving to an Operation input like
any hole outside a Capability-relevant position (§12), which is why `create_dns_record` above writes
`identity: "{name}"` and takes the DNS name as its Record name rather than the opaque id Cloudflare
answers with. The two forms are told apart where every scalar in this grammar is: a path opens with `$`
and a hole with `{`. So does `$.command` on a `shell` Operation, which sits in the response object
precisely because it is a fact about the call rather than about the answer — the built-in
`mutate_skip_if_recorded` needs nothing further. A response path anywhere else names a value that exists
only once the call has gone out, which is a Manifest declaring a test it cannot perform:
`manifest-inconsistent` (§4), the code that already refuses a `pagination` Pattern outside an `over:`
and a `record:` on a `destroy`.
`fields:` is optional: a Record carrying only its identity and its metadata is a perfectly good Asset,
and an empty mapping would be a thing written to mean nothing.

`identity:` and a `fields:` entry may name the same path, and an expanding effectful Step is the reason
they usually do — `{item: $.id}` addresses what an Expansion resolved to, and the handle an API knows a
thing by is normally its identity. This is not a second representation of one fact: both read the same
response and cannot disagree, and where the paths differ the Record's name and its recorded field are
genuinely two facts. A reference naming a field nothing projects is caught before the Run
(`reference-unresolvable`, §4), which is what keeps the arrangement from being a trap.

### An Expansion's members are one Record identity each

Every member of an Expansion is a call, so several members projecting one identity is several calls
writing several versions of one series. The Store takes that in its stride, two versions of one series
being the ordinary case, and the entry §7 writes then says something else entirely: the identity set
holds a set where `expanded_to` holds a sequence, so three expanded to and one concluded about reads as
*two unaccounted for*, the phrase reserved for a call that may have reached the world. That is the same
sentence the `values:` duplicate above is refused on, one Run later and against a projection rather than
against a list, and it carries the same code (`record-identity-collision`, §12) because it is the same
check: two things that must be distinct identities being one.

Nothing required the projection to depend on the member before this, and the three Kinds fail
differently enough that no example showed it. A `destroy` cannot reach the case at all, projecting
nothing and writing its Tombstone under the Asset's own identity (§7, ADR-0033), so the member *is* the
name. A `mutate` under `repeatable` writes the versions and the surface reports a halt that did not
happen. Under `skip-if-recorded` the first member runs and the rest skip, the test having become true of
them — which is coherent, reports `completed`, and does one third of what a reviewed artefact asked for,
with no surface saying so. A `read` collapses identically and touches nothing while doing it. The rule
therefore turns on Kind nowhere, which is what §6 already says of the projection failure beside it.

The rule is that **distinct members resolve to distinct identities**, not that the member reaches the
identity. The narrower statement is the one the arithmetic needs, and the wider one is both too strong
and too weak: `mutate_skip_if_recorded` tells its members apart perfectly with an `identity: $.command`
that is not the member, and `{item: $.id}` over an `assets:` selector reaches the identity from the
member and still collides where two Assets hold one value in that field. The boundary is one Step's
Expansion and nothing wider — two Steps running the same argv against one Definition and Target still
write two versions of one series, as above.

Where the identity resolves before the call, this is decided before anything is touched: §4 refuses it
offline wherever the member count is authored, and §6 over the resolved set at Expansion. Where
`identity:` reads from the response there is nowhere earlier than the response to decide it, and §6
states what happens then.

The built-in `shell` Provider writes one projection, shared by all four of its Operations that carry
one, and §12 states it in full: `identity: $.command`, with `exit_code`, `stdout` and `stderr` as
`fields:`. A Definition author cannot vary it, a Manifest's declared facts being the Provider author's
and no artefact downstream overriding one (§13) — and here the Provider author is `hyper`. The Record's
name is therefore the command, so two Steps running the same argv against one Definition and Target
write two versions of one series, and `mutate_skip_if_recorded` means *skip while the Asset this exact
command produced still stands*.

### Two roots, one marker

A `series` Operation reads from two roots. `over:` reads from the response; `identity:` and every
`fields:` path read from each member of the collection `over:` named. Both are written `$`, and the
position decides which root it means — the grammar §12 closes has three productions and gains no fourth
for this. The cost is real and is the price of not growing it: `identity: $.id` reads identically under
either cardinality, and the sibling `over:` is what says which root it started from.

### Secret output

A Manifest names its secret output fields in `secret:`, a list of `fields:` names beside the projection
rather than a marking inside it. Each is written to the Store as a presence-only marker (§7, ADR-0007).

The list is separate for two reasons. `fields:` values stay uniformly scalar, so a mapping in that
position keeps meaning a reference and nothing else (ADR-0022). And a reviewer asking what a Provider
handles that never reaches the Store reads one line rather than scanning a projection for marks — the
argument §12 makes for keeping a closed set readable in one place.

The built-in `shell` Provider declares none, and that is a decision rather than an omission. `secret:`
is a Provider author's claim about output that author understands, and here the author is `hyper`,
which knows nothing whatever about the command: declaring `stdout` secret on every command would be
`hyper` asserting a fact it cannot have, and declaring it on none is the honest reading of the same
ignorance. What it costs is stated in §13, where it qualifies ADR-0007 rather than sitting beside it.

## Auth

The set is closed at two, `header:` and `basic:`, stated in full in §12. An Auth scheme is a header and
a placement and never a protocol (ADR-0031): it decorates a request `hyper` was already making, and
nothing in it fetches, exchanges, refreshes, or signs.

A Manifest names its scheme as a key: `auth:` carries one key, the scheme's name, over that scheme's
parameters. A scheme taking none carries an empty mapping — `auth: {basic: {}}` — rather than a bare
scalar, since a key whose value is sometimes a scalar and sometimes a mapping is the ambiguity
schema-directed typing exists to remove and there is no `null` to write instead.

`auth:` is optional. A Provider omitting it sends no credential, which is what an uptime check against a
public host is, and every surface rendering a Provider's auth renders that absence as `none` — an
undeclared default rather than a value to write, on the reading `repeatability:` already has (§12). A
Provider declaring only the `shell` Capability and carrying an `auth:` block is a Manifest disagreeing
with itself, since auth is a property of reaching a host: `manifest-inconsistent` (§4).

The scheme owns the position in the request that carries the secret; a Manifest never chooses it. That is
what closure buys (§12): `hyper` suppresses a credential by the position it occupies rather than by
scanning a rendering for something that looks like one (ADR-0007), which is only true while the position
is `hyper`'s. A `headers:` entry naming a position its scheme owns is `manifest-inconsistent` (§4), and
the comparison is case-insensitive, as an HTTP header name is.

Naming a header is not choosing a position. The position class is the scheme's — a request header, for
both members — and what a parameter supplies is which header inside that class, which leaves suppression
exactly as mechanical as it was: `hyper` wrote the header and knows it by name. What it may not name is a
header `hyper` computes for itself. There are five — `Host`, `Content-Length`, `Content-Type`,
`Transfer-Encoding`, `Connection` — and a scheme naming one is refused at load as `header-reserved`
(§4), which is the same code a `headers:` entry naming one carries above: one rule with two writers,
not two rules. `Host` is the reason the check is not merely tidiness: `hyper` derives it
from `host:`, which is the value the Target's grant was checked against, so a scheme setting it would
dial a granted host while claiming another.

A scheme's parameters carry literals and admit no template hole of any kind, which makes them the one
position in the format where a hole is illegal outright rather than restricted to a source
(`hole-illegal`, §4). A hole here resolving to an Operation input would let a Step's arguments choose
the header a credential lands in, which is the Manifest-chooses-placement door reopened from the far
side.

A scheme's credential slots are the scheme's, and neither a Manifest nor a Target declaration invents
one. A Target declaration's `auth:` is a mapping of slot name to credential slot, and it names no scheme:
a Target declaration is written without knowing which Provider will bind it, and every static check over
one runs from that artefact alone (§4). Coverage is checked per binding instead — for each
(Definition, Target) pair, the Target's mapping covers the slots the Provider's scheme requires, and a
gap is `manifest-inconsistent` (§4). A Target may carry more slots than any one Provider needs, which is
what lets one Target serve two: a declaration carrying `token`, `username` and `password` serves a
`header:` Provider and a `basic:` Provider at once. It cannot hold two *different* secrets for one
scheme, and that is the model asserting itself rather than a limit — a Target is the unit of both blast
radius and credentials, so two credentials are two Targets.

Everything downstream of the slots reads the binding rather than the declaration. The presence check
before a Run's first Step (§6) and the `env:` block a projection derives (§10) both resolve over the
(Definition, Target) pairs a Procedure actually makes, so a slot a declaration carries and this Procedure
never binds is neither required to be present nor written into a runner's environment.

```yaml
# in a Manifest — the scheme and its parameters
auth:
  header: {name: Authorization, prefix: "Bearer "}

# in the Target declaration that binds it — the scheme's slot, and where it resolves from
auth:
  token: {env: CLOUDFLARE_API_TOKEN}
```

A Manifest reaching a public host writes no `auth:` at all, and the Target declaration it binds carries
none either — which is what `local` is (§4, ADR-0024).
