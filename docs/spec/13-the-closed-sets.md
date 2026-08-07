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

**Open.**

## Disposition

**Open.**

## The outcome triple

**Open.**

## Exit codes

**Open.**

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
`bound-missing`, `host-not-granted`. Further members are named where §5, §7, and §11 state the checks
that carry them.

## The path grammar

Two roots: a Manifest's projection reads from an Operation's response, and a Step's reference reads
from a Record. From either root, the grammar is `$`, `.member`, and `["member"]`, and nothing else.
Recursive descent (`..`) is rejected on the same ground as a YAML alias or a JSON Schema `$ref`: a path
whose meaning depends on data the reviewer cannot see while reviewing (ADR-0022). Array indexing
(`[n]`) is rejected because an index into an upstream array is the identity-is-array-position hazard
that a Manifest's declared Record identity exists to close. Iteration (`[*]` or any equivalent) is not
in the grammar at all: it is declared once, in an Operation's Record cardinality, and implied by a
cardinality of `series`.

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

**Open.**

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
