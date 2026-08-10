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

A **predicate** filters a set of Records, in a Step's selector (`over:`) or its condition (`when:`).
The operator set is closed and defined in §12. A predicate list is always AND; there is no disjunction
anywhere in it (ADR-0022).

A **template hole** fills a scalar position inside an otherwise literal value, one hole syntax across
every artefact: `{name}`, naming what fills it and nothing more, and a value beginning with one is
quoted (ADR-0023). What a hole may resolve to is decided by the position it appears in, never by
scanning the resolved value afterward for something dangerous. The legal positions are a closed set
defined in §12.

### The credential slot

A credential slot's value is a mapping whose sole key is `env:`, naming the environment variable
`hyper` resolves it from at Run start. A scalar in a credential position is always a load error, with
no exception. `env:` may appear nowhere else in any artefact — a general ambient-input channel is
authority arriving after review under another name (ADR-0008).

### `local` and the host grant

`local`'s Target declaration enumerates the hosts it grants like any other Target's (ADR-0024): a
Capability-relevant hole resolving against it is checked the same way, and a `values:` list of hosts is
checked against the same enumeration (§12, the `over:` forms).

Which host a request reaches follows from the grant's size rather than from a second declaration.
Where the bound Target grants one host `hyper` fills it; where it grants several the Operation marks
one of its inputs as the host, and the value that input carries is checked for membership of the
grant like any other (ADR-0024). The marking is a property of an input rather than a further fact
about the Operation.

## The five artefacts

### Provider

`kind: provider`, in `providers/`. A Provider is a Manifest and nothing else. It declares the closed
input schema and output projection for each Operation it exposes, and, per Operation, the seven facts
`hyper` would otherwise have to guess at: the Kind, whether it is `opaque`, its Repeatability, its
deadline, its concurrency limit, the Patterns it uses, and its Record cardinality (§12) together with
the response field that is the Record's stable identity. A Record's name is the value that field
holds, so an Operation whose response a Record is projected from and which declares no identity field
produces a Record nothing can identify, and declaring none is a load error. A `destroy` Operation
projects no Record of its own and declares neither, what it writes being a Tombstone (§7). A Manifest
also declares the Capabilities it requires, once for the Provider and never per Operation, `hyper`
deriving the per-Operation ones (§4); which output fields are secret; the Auth scheme it
authenticates with and that scheme's own declared parameters; and the class of Target its Definitions
may bind, a static type-check rather than a set that could expand a Definition's reach. A Manifest
alone carries an explicit schema-version field; the other four artefacts carry none, since the
repository-wide version pin already fixes which binary reads them and a Manifest is the one artefact
authored by someone outside that pin's reach (ADR-0023). An installed Manifest carries one further
block, written by `hyper` rather than authored: the registry ref and digest `install` verified it
against (§11).

### Target declaration

`kind: target-declaration`, in `targets/`. The reviewed half of a Target, holding no credentials: the
Kinds it accepts, the Capabilities it grants, the host set it grants — one member where the Target is
a single endpoint, and never a grant without an enumeration (ADR-0024) — its class,
whether it opts into `opaque` plus `destroy`, and the environment variable name for each
credential slot its Auth scheme requires. Every static check over a Target runs from this artefact
alone, with no credential resolved and no network reached.

### Definition

`kind: definition`, in `definitions/`. A named, authority-scoped use of one Provider: the Kinds it
claims for `read` and `mutate`, the named `destroy` Operations it claims — granularity follows
severity, so a `destroy` claim names Operations rather than a Kind — and the Targets it may bind, named
literally rather than by class or tag, since a Target class only ever rejects a mismatch and never
expands a Definition's reach. A Definition carries no argument value of its own: the gutter renders what
reaches the world from the file being read, and a value living on the Definition would force a second,
`AUTHORITY`-shaped table just to re-show it (ADR-0026). Those values belong to the Step.

### Procedure

`kind: procedure`, in `procedures/`. An ordered `steps:` list and the full set of Targets the Procedure
and everything it invokes may touch, authored rather than derived so a reviewer can see the envelope
without tracing every nested invocation. A Procedure's Cadence is authored here too; its grammar
belongs to §10.

Each entry in `steps:` is either a Step or a nested Procedure invocation, in the same list — never a
separate block, since a second list would put ordering between two structures and reintroduce the graph
Steps are sequenced without (ADR-0002). A Step names its `definition:` and `operation:`, the `args:` the
Operation's input schema requires, an `over:` selector, a `bound:` on the Records an effectful Step may
affect, and a `when:` condition rooted at an earlier Step's Record, carrying `step:` beside `field:`. A
nested invocation names a `procedure:` in place of `definition:`/`operation:`, and its Steps render
under the invoking Step's path with the invoked Procedure's transitive envelope. A comment is permitted
on any line, rendered verbatim in the gutter and never read by `hyper`; no directive syntax may ever
exist inside one, since that would be a bypass wearing a comment.

`over:` takes one of three forms, closed and defined in §12, and a Step declaring none is invoked
once. All three range over something already written down — Records in the Store, or identifiers
authored in the Procedure — so a Step calling an Operation for the first thing it will ever record
has nothing to range over and says so by omission.

### Repository declaration

`kind: repository-declaration`, at `hyper.yaml` in the repository root — the one artefact with no
directory, agreeing with its filename instead. It admits only facts that govern the repository as a
whole and belong to no Procedure, Definition, or Target declaration: the `hyper` version pin and its
digest, written only by `hyper project`, and the retention policy that bounds Compaction.
