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
the half no static check can perform, since no credential exists before then. The two opt-ins are
independent by design: a laptop's credential for a Target can carry the opt-in while the credential
Actions holds for the same Target does not, which makes "CI may not do this" a fact about what the CI
credential holds rather than a belief about where the process runs — credentials resolve the same way
regardless of where `hyper` runs (ADR-0007).

## The Bound

A `destroy` Step's Bound is mandatory: an absent Bound means unbounded, and unbounded is refused
before anything runs rather than left unchecked (`bound-missing`, §4).

## Expansion

A selector's Expansion — the resolution of `over:` to the concrete Records a Step will act on — is
scoped by the Step's own Kind: a `read` Step may expand over Observations as well as Assets, and an
effectful Step may expand only over Assets (ADR-0027). A series whose head is a Tombstone stands for
nothing and is expanded over by neither, so what one Run destroyed the next does not reach again and
does not count against its Bound (§7). Anything `hyper` did not create is therefore
reachable only by literal identifier — a `values:` list, occupying lines the gutter annotates and
counted by the Bound like any other selector (§12) — never by a selector ranging over it: a Record
`hyper` never created is not an Asset, and an effectful selector has nowhere else to reach.

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
