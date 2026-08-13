# §5 — Authority and safety

Authority is decided before invocation, not at it. Most dangerous states are unreachable from the
five artefacts by construction, and `check` refuses the rest before the first effect reaches the
world. No confirmation happens at execution time and no per-Run approval exists; CI runs unattended,
and every guardrail below is checked before that unattended Run starts. This chapter states what
stands between an authored artefact and the world, in full: the two keys a Step must satisfy, the
envelope composition cannot widen, what an `opaque` `destroy` additionally needs, the Bound a
`destroy` Step cannot omit, the reach a selector is granted by Kind, and the terminal outcome all of
it ends in when it declines rather than runs.

## The two keys

A Step runs only where its Definition's claimed Kind intersects its bound Target's accepted Kinds,
and a `destroy` Step's Operation must be named among those its Definition claims for `destroy` — the
check and its error codes are §4's (`kind-not-granted`, `operation-not-claimed`).

## The envelope

A Procedure's transitive Target and Kind envelope — everything reachable through every Procedure it
invokes, to any depth — is checked before either runs, so composition cannot widen blast radius by
accident (`envelope-exceeded`, §4).

## `opaque` and `destroy`

An `opaque` Operation's effects are ones `hyper` cannot describe, so an `opaque` `destroy` cannot be
shown to touch only Assets and cannot be Bounded — it is unbounded blast radius in exactly those
words, and it is refused unless two separate opt-ins agree. The Target's declaration must opt in —
the artefact half, checked statically with no credential resolved (`opaque-destroy-not-granted`, §4).
The credential resolved for that Target at Run start must carry the same opt-in, or the Step Refuses —
the half no static check can perform, since no credential exists before then. A third requirement is
stated with the Bound below, where the reason for it is: an `opaque` `destroy` Step must name the
population it is destroying. The two opt-ins are
independent by design: a laptop's credential for a Target can carry the opt-in while the credential
Actions holds for the same Target does not, which makes "CI may not do this" a fact about what the CI
credential holds rather than a belief about where the process runs — credentials resolve the same way
regardless of where `hyper` runs (ADR-0007).

## The Bound

A `destroy` Step's Bound is mandatory: an absent Bound means unbounded, and unbounded is refused
before anything runs rather than left unchecked (`bound-missing`, §4).

An `opaque` `destroy` Step is the one Step that carries no Bound, and writing one there is refused
(`bound-illegal`, §4). This is not the rule above softened. A Bound counts the Records an Expansion
resolved to, and a count of the calls an opaque Step made says nothing about what any of them did: the
only Bound a single command could carry is `1`, which would stand in the gutter and in `FLAGS` reading
*at most one thing will be destroyed* while `rm -rf /` is magnitude one. Truthful and still misleading
is the worse failure on the most severe Step the tool runs. Unbounded is the accurate word for it, and
§13 uses that word.

So does the review. Such a Step draws three flags — `DESTROY`, `OPAQUE` and `UNBOUNDED` — and the last
is implied by the first two on every such Step, having no other form to take. It renders regardless
(§12): a surface indexing what is unbounded, silent on the one Step where nothing can be bounded, is
omitting rather than economising.

**An `opaque` `destroy` Step must carry an `over:` selector** (`opaque-destroy-unscoped`, §4). Without
one it is invoked once (§3), so it has no Expansion, no series to write a Tombstone under, and no
declared identity — a `destroy` carries no `record:` at all (§3, ADR-0037). It would reach the world
and write nothing whatever: no Record, no Tombstone, an empty identity set, and no row in `YOU DID
THIS`, which is the one thing an effectful path may not do. The requirement is not a Bound arriving
under another name (ADR-0053). It buys two different things: the population is authored literally, on
lines the gutter annotates, so a reviewer reads what is being destroyed rather than inferring it from a
command; and `expanded_to` records what the Step resolved to in Expansion order (§7), so *which three
of the five* is legible after a halt with no Bound anywhere claiming to have guarded it.

```yaml
  - id: purge-releases
    definition: host-ops
    operation: destroy
    target: local
    over:
      values: [/srv/app/releases/r41, /srv/app/releases/r42]
    args:
      command: [rm, -rf, {item: $}]
```

Two Tombstones, each opening the series it ends (§7, ADR-0033), each named by a path a human
recognises. What stands in the Bound's place is therefore three things and not two: the two independent
opt-ins above, a Definition that named the Operation, and a population the author wrote down.

## Expansion

A selector's Expansion — the resolution of `over:` to the concrete Records a Step will act on — is
scoped by the Step's own Kind: a `read` Step may expand over Observations as well as Assets, and an
effectful Step may expand only over Assets (ADR-0027). A predicate reads the **head** version of each
series and no other: *any version* would have a `destroy` reach a thing for what it used to be, and
would make one artefact reach further every month the Store grows. A series whose head is a Tombstone
stands for nothing and is expanded over by neither, so what one Run destroyed the next does not reach
again and does not count against its Bound (§7) — which is that same rule applied to the one version
type that stands for nothing.

An `opaque` Step expands like any other, all three `over:` forms reading on it under the same Kind
rule. Nothing about opacity restricts which Records a selector may range over; what it restricts is
what may be concluded from counting them, which is the Bound's business above.

An Expansion is where a predicate meets a value, so it is where a value of the wrong type is found. It
Refuses (`predicate-type-mismatch`, §12) rather than excluding the Record quietly, which it can do
because an Expansion resolves before the Step's call goes out (§6, ADR-0035). Anything `hyper` did not create is therefore
reachable only by literal identifier — a `values:` list, occupying lines the gutter annotates and
counted by the Bound like any other selector (§12) — never by a selector ranging over it: a Record
`hyper` never created is not an Asset, and an effectful selector has nowhere else to reach.

The Tombstone rule reads on a `values:` list too, once a member has a series to have a head, and there
it is a `destroy`'s: on a `destroy` Step a member whose head is a Tombstone is dropped from the
Expansion, and a member naming no series at all is reached. What that states is that `hyper` drops what
it knows is gone and reaches what it has no record of — a member it never touched being in the second
class — so a `values:` list left standing in a Procedure that runs on a Cadence is self-limiting rather
than a Run that fails on a call the artefact never asked it to repeat.

A `mutate` reaches such a member instead, and the same sentence is the reason: a create over a
Tombstoned series is a call the artefact *is* asking for, and §6 skips a member only while its Asset
stands. Dropping it here would leave destroy-then-recreate reachable from a Step with no selector and
from nowhere else, which nothing states and ADR-0011 contradicts. This is ADR-0027's Kind-scoping
arriving at a case it did not enumerate rather than a second rule standing beside it.

It is the Store that shortens the list and never lengthens it, which is what lets §4 count the authored
length against the Bound offline.

## No bypass

Nothing overrides a Refusal at invocation time — no flag, no confirmation, no per-Run override of any
check in this chapter — and the only way past one is an edit to the authored artefact, put back
through review (ADR-0001).

## Refusal

A Refusal is a terminal Run outcome distinct from failure: a guardrail declined a Step before any
effect reached the world, rather than the world resisting one. Every check this chapter and §4 state
ends in a Refusal when it does not pass. A Refusal is recorded in full — which check, which Step,
which Target, what was declared against what was found — and no Record is written for the Step it
stopped, since nothing happened to the world; both belong to §7.

## Blast radius

A Bound counts Records; it does not weigh what happened to each one. It catches the runaway selector,
which is the failure it exists for, and says nothing about how severe a single, correctly-Bounded call
is. Nothing above reopens on that account — the two keys, the named-Operation requirement on
`destroy`, and the Definition review are what stand between an author and that outcome. Blast radius
measured in counts rather than severity is named here as the honest limit it is, carried forward to
§13.
