# §4 — Static verification

`check` is the name for every rule in this chapter, run together: load every artefact in the
repository and evaluate every rule below against them. No credential is resolved, no network reached,
no Operation invoked. `check` runs standalone — from an editor, a pre-commit hook, a CI job — and it
runs again, identically, before the first Step of any Run: pre-flight is `check` re-run with nothing
skipped, never a lighter pass, so the standalone command is never the only thing standing between an
artefact and the world.

`check` Refuses on the version pin like every other command: it compares itself against the Repository
declaration before reading a second file and Refuses on mismatch, on a laptop and in CI alike, with an
absent pin naming `hyper project` (ADR-0020). Every rule below presupposes that gate has already
passed.

Every rejection below carries a closed `error_code`, named here and defined in full in §12.

## Grammar

Loading a file is the first check, and failing it stops every check after. The rules are the grammar
§3 states in full and the sets §12 closes — the strict YAML subset, the input-schema subset, the
credential slot's mapping shape, the three hole positions, the path and reference grammar — and this
chapter is where each rejection gets the name it is refused under, nothing more: a construct the YAML
subset excludes is `strict-yaml-violation` (ADR-0023); a key the schema at that position does not
define is `unknown-key`; a `kind:` disagreeing with its directory or filename, against §12's table, is
`kind-mismatch`; an artefact's name disagreeing with its file's basename is `name-mismatch` (§3); an
input schema outside the closed subset is `schema-unsupported`; a malformed or
misplaced credential slot is `credential-slot-malformed`; a hole resolving outside its position's legal
source is `hole-illegal`; a reference naming an earlier Step of `series` cardinality is
`series-reference`; and a reference naming a field no Operation of that Provider projects is
`reference-unresolvable` (§3).

`name-mismatch` is its own member rather than a widening of `kind-mismatch`. The two carry different
disagreements over four artefacts, and one code standing for both would leave a reader of the rendering
unable to tell which of two files to edit.

## The Manifest's oracle

A Manifest is data, so what it claims can be checked against what its own Operations require, with
nothing but the file itself (ADR-0004). The Capabilities a Manifest declares must equal, not merely
contain, the Capabilities `hyper` derives from every Operation's schema and its own holes —
over-declared or under, either direction is `capability-mismatch`. §3's identity-field requirement and
Target-class type-check get their names here: an Operation projecting a Record and declaring no
identity field for it is `identity-undeclared`, and a Definition naming a Target outside its Provider's
declared class is `target-class-mismatch`.

Four further checks are one fact wearing four shapes — a Manifest whose own declarations disagree with
each other — and they share one name, `manifest-inconsistent`: a `pagination` Pattern on an Operation
whose `record:` carries no collection path, a `host-input:` naming a property the Operation's input
schema does not define, a `headers:` entry taking the request position its Auth scheme owns, and a
Target declaration's credential slots not covering the scheme's slots for a binding a Definition makes
(§3). The last is checked per (Definition, Target) pair rather than on the Target declaration alone,
which is the one place a Target's own artefact is not sufficient — a Target declaration is written
without knowing which Provider will bind it, and the scheme is the Provider's.

## The two keys

A Step runs only where its Definition's claimed Kind and its bound Target's accepted Kinds intersect —
both authored, neither derived, so a claim of "never destroys" is a fact the reviewer can trust rather
than the Manifest's word for it. A Step whose claim and grant do not intersect is `kind-not-granted`. A
`destroy` Step's Operation must be named among the Operations its Definition claims for `destroy` —
granularity follows severity, so `read` and `mutate` check at Kind level and `destroy` checks by name —
and an unnamed Operation is `operation-not-claimed`.

A Procedure's declared Target and Kind envelope must contain everything reachable through every
Procedure it invokes, to any depth. An invoked Procedure's transitive envelope reaching outside its
caller's declared one is `envelope-exceeded`, checked before the first Step of either runs —
composition cannot widen blast radius by accident.

An `opaque` `destroy` Operation may run against a Target only where that Target's declaration has
opted in; a Definition claiming one against a Target that has not is `opaque-destroy-not-granted`. This
is the artefact half of the check — whether the credential in hand also carries the opt-in is resolved
at Run start and belongs to §5.

## The Bound

A `destroy` Step declares the maximum number of Records it may affect. An absent Bound on a `destroy`
Step means unbounded, and unbounded is refused before anything runs: `bound-missing`. A `mutate`
Step's Bound is optional; its absence is not a check's business — it is rendered, unbounded, in the
blast-radius summary, which belongs to §8. A `read` Step carries no Bound at all, having nothing for
one to guard. Whether an Expansion's actual count exceeds a declared Bound is not decidable from the
artefacts alone and belongs to §6.

## The host list

§12 already states two comparisons of a host set against a Target's grant — an Operation's `host:`
template expanded to its candidate set, and an `over:` `values:` list. Both share one name: a member
absent from the grant, from either origin, is `host-not-granted` (ADR-0024, ADR-0029). What the
intersection of a candidate set and a grant then decides at Run time is §3's, and it is the one part of
the host rule no static check performs: where the intersection holds several hosts, the value a
`host-input:` carries is checked for membership when it arrives.

## What `check` cannot know

Every rule above is checkable because it compares one artefact's declared claim against another's, or
against a schema — reviewed text against reviewed text. None of it touches whether a Manifest correctly
describes the API it names: that its schema matches what the API actually accepts, that its projection
paths match what the API actually returns, that its declared Kind matches what an Operation actually
does. `hyper` has no oracle for that question and does not claim one — nothing cross-checks a declared
Kind against what an Operation actually does at runtime (ADR-0025), and the cheapest evidence
available, an unauthenticated Probe against the real response, only narrows the question rather than
closing it (ADR-0017). What a Run does when a projection path does not resolve against a real response
is §6's, that being the only place the question can be put. This is not a gap `check` closes by
growing; it is named here and carried forward as a limit in §13.
