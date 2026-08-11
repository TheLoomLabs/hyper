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
can arise — there is no untyped position for it to arise in. A file is YAML *syntactically* and not
*semantically*: editors and language servers still parse it, but every construct that would let one
line's meaning depend on another line (an alias) or on data the format's own grammar does not show (a
tag) is refused before anything is read for meaning.

Key order within a file is free and nothing checks it. The layout ADR-0023 fixes is where files live,
not the order of keys inside one, and a reordering renders as what it is: a line that moved, counted by
the Comparison's catch-all (§12) and reviewed against the previous revision like any other edit.

## Lists and mappings

One rule decides the shape of every collection in every artefact. A collection of **named** things is a
mapping keyed by that name — a Manifest's Operations, a projection's fields, a Target declaration's
credential slots, an Operation's Patterns. A **list** appears in exactly two places: where order is the
meaning, which is a Procedure's `steps:` and nowhere else, and where the members are bare scalars, which
is `targets:`, `hosts:`, `kinds:`, `capabilities:`, a Definition's `destroy:` claim, a schema's
`required:`, and a selector's `values:`.

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

## Types

An Operation's input schema is written in a subset of JSON Schema, closed and defined in §12. An
Operation's output carries no schema at all — only the projection a Manifest declares, so cardinality,
identity, and every recorded field are stated exactly once rather than in two representations that can
disagree.

The scalar vocabulary is closed and defined in §12: the common JSON Schema primitives plus two the
domain forces, a `duration` and a `timestamp`. There is no `null` among them — a field's presence is a
predicate fact (`exists` / `absent`), never a type.

## No expression language

There is no interpolation syntax, no operator, no function, no arithmetic, no regular expression, and
no boolean algebra anywhere in the format (ADR-0022). Three closed forms carry everything the five
artefacts need to say instead: a path, a predicate, and a template hole.

A **path** names a value living elsewhere — a field of an Operation's response, where a Manifest states
its projection, or a field of a Record, where a Step references one. Its grammar is closed and defined
in §12. Where a path appears in a value position rather than a projection, it is embedded as a
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

A reference names a field the Record it points at actually carries. The field set of a Record series is
Manifest-declared, so a reference naming one no Operation of that Provider projects is
`reference-unresolvable` (§4) rather than a Run that fails partway through — which is what makes the
projection rule below safe to state.

A **predicate** filters a set of Records, in a Step's selector (`over:`), its condition (`when:`), or a
polling Pattern's terminal condition. The operator set is closed and defined in §12. Each entry carries
a `field:` and exactly one operator; two operators sharing an entry would be an AND that occupies one
line where the list already gives each conjunct its own, which is what the gutter annotates and what the
Comparison renders a selector change from. A predicate list is always AND; there is no disjunction
anywhere in it (ADR-0022).

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

`local`'s Target declaration enumerates the hosts it grants like any other Target's (ADR-0024): a
Capability-relevant hole resolving against it is checked the same way, and a `values:` list of hosts is
checked against the same enumeration (§12, the `over:` forms).

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
whether it is `opaque:`, its `repeatability:`, its `deadline:`, its `concurrency:` limit, the `patterns:`
it uses, its `input:` schema, its request, and its `record:` projection. None of them is nested under a
grouping key: the Kind is the most review-relevant fact in the file and an indent is what would put it
behind something else.

Two of the seven are stated by omission. `repeatability:` omitted is run-once, which §12 fixes as the
undeclared default rather than a value to write. Record cardinality has no key at all — `series` is a
`record:` carrying `over:`, and `one` is a `record:` without it, since a `series` projection cannot omit
the collection path and a `one` projection has nothing to put there. A `destroy` Operation carries no
`record:` and declares no identity, what it writes being a Tombstone under the series its Expansion
acted on — or, where that Expansion was a literal naming no series, under a series the Tombstone opens
(§7, ADR-0033).

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
      required: [zone_id, name, type, content]
      properties:
        zone_id: {type: string}
        name: {type: string}
        type: {type: string, enum: [A, AAAA, CNAME]}
        content: {type: string}
    record:
      identity: $.result.id
      fields:
        id: $.result.id
        name: $.result.name
        created_on: $.result.created_on
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
      required: [zone_id]
      properties:
        zone_id: {type: string}
    patterns:
      pagination:
        cursor: {from: $.result_info.cursor, into: {query: cursor}}
      retry: {attempts: 3}
    record:
      over: $.result
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
      required: [zone_id, record_id]
      properties:
        zone_id: {type: string}
        record_id: {type: string}
```

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

### Definition

`kind: definition`, in `definitions/`. A named, authority-scoped use of one `provider:`: the `kinds:` it
claims for `read` and `mutate`, the named `destroy:` Operations it claims — granularity follows severity,
so a `destroy` claim names Operations rather than a Kind — and the `targets:` it may bind, named
literally rather than by class or tag, since a Target class only ever rejects a mismatch and never
expands a Definition's reach.

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
invoked Procedure's transitive envelope. A comment is permitted on any line, rendered verbatim in the
gutter and never read by `hyper`; no directive syntax may ever exist inside one, since that would be a
bypass wearing a comment.

`target:` is written on every Step and is checked for membership of its Definition's `targets:` list. It
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
`body:`. `query:` and `headers:` are mappings of name to string, always: a query string is text on the
wire, and typing it as text leaves schema-directed typing nothing to guess at. `body:` is a mapping
serialised as JSON, and `hyper` sets `Content-Type` accordingly. A form-encoded, XML, or raw body is
not writable, and joins the limits §13 states rather than becoming a second serialisation this format
learns.

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
      required: [endpoint]
      properties:
        endpoint: {type: string}
```

## The Patterns

A Pattern is behaviour `hyper` performs around a call, which a Manifest parameterises and does not
implement. The three are closed (§12) and each is a key under an Operation's `patterns:` mapping.

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

**Polling** carries an `interval:` and an `until:` predicate list whose `field:` roots at the response.
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

`record:` is where a Manifest states what an Operation's response becomes. It carries `identity:`, a path
to the response field that is the Record's stable identity; `fields:`, a mapping of recorded field name
to path, which is the whole of a Record's projected content; and, for an Operation of `series`
cardinality, `over:`, the path naming the collection the Records are projected out of.

A Record's name is the value the identity field holds, so an Operation projecting a Record and declaring
no identity produces one nothing can identify, and declaring none is `identity-undeclared` (§4).
`fields:` is optional: a Record carrying only its identity and its metadata is a perfectly good Asset,
and an empty mapping would be a thing written to mean nothing.

`identity:` and a `fields:` entry may name the same path, and an expanding effectful Step is the reason
they usually do — `{item: $.id}` addresses what an Expansion resolved to, and the handle an API knows a
thing by is normally its identity. This is not a second representation of one fact: both read the same
response and cannot disagree, and where the paths differ the Record's name and its recorded field are
genuinely two facts. A reference naming a field nothing projects is caught before the Run
(`reference-unresolvable`, §4), which is what keeps the arrangement from being a trap.

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
`Transfer-Encoding`, `Connection` — and a scheme naming one is refused at load as
`auth-header-reserved` (§4). `Host` is the reason the check is not merely tidiness: `hyper` derives it
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
