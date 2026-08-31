# A Requirement halts, and claims nothing to do it

**A `steps:` entry may carry an `id:` and a `require:` and nothing else. It binds no Target, invokes no
Operation and declares no Kind; where its predicate holds the Run goes on, and where it does not the
Run halts — here and in whatever invoked this Procedure.** That is a **Requirement**, the third entry
shape beside the Step and the nested invocation, and it is the only thing in `hyper` that stops a Run
without acting on the world first.

**It is the condition's predicate read for the other answer.** Same root, same `step:` beside `field:`,
same eleven operators, same one instant, same rule that a condition reads what the Steps of *this* Run
acted on and never falls through to the Store (§3, §6, §12). A `when:` that does not hold **skips** the
Step it is written on; a `require:` that does not hold **halts**. §12 gains a fourth position and no
fourth root.

**What it buys is a check that is shared and gating at once**, which is the pair issue #236 states
`hyper` supported neither half of together. A Procedure whose Steps are all `read` and whose last entry
is a Requirement is read-only in `review`'s authority table and still stops everything downstream of
it, because one Run has one outcome however deep the invocation goes (§6).

**ADR-0002 is not reopened.** No graph, no dependency edge, no Step namespace crossing an invocation's
boundary, and nothing an invoked Procedure did becomes referenceable from its caller. A Requirement
roots at a Step of its own Procedure exactly as a `when:` does. What changed is not what a caller can
read; it is that the callee can stop.

## What was wrong (issue #236)

§6 states the mechanism for a precondition: a `read` never halts on what came back, the status is
recorded, *and a later Step's `when:` decides on it* — which puts *what counts as acceptable* on a line
the gutter annotates rather than in the artefact a reviewer reads least (ADR-0050).

**That mechanism stops at a composition boundary, and correctly.** `checkSteps` threads its reference
index forward so a `when:` resolves only against an `id:` written earlier in the same list, and the
invocation itself projects no Record for a predicate to root at. So for a caller, *the invoked
Procedure's verdict* is not a fact any later Step can condition on.

What was left was for the check to halt on its own — and a halt required an effectful Step, because a
`read` never halts. The sealed acceptance run of 2026-08-30 on `release-promotion`
([ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md),
issues #225 and #232) authored exactly the shape the task asked for, met three Refusals trying to gate
on it —

```
reference-unresolvable  step: archive-intact names no id: this Procedure declares earlier
reference-unresolvable  step: verify.archive-intact names no id: this Procedure declares earlier
reference-unresolvable  step: verify names no id: this Procedure declares earlier
```

— and then moved the halt inside the shared check, with a `mutate` Step on `local` whose whole content
was `exit 1`. It told the reviewer so in as many words: *the shared check is not read-only in authority
terms. Its last step claims `mutate` on `local`, because that is the only way it can halt.*

**Every Refusal there was right, and the result was still wrong on three counts.** It inverted the Kind
axis on the artefact where a reviewer most relies on it — `review` rendered effective `m`, with an
`OPAQUE` flag, for a Procedure that touches nothing. It made the Refusal path an effect path: the
halting Step consumes a Bound, is subject to the `destroy`/`mutate` gates, and on a Target that did not
grant the Kind would Refuse `kind-not-granted` rather than halt, so **a check could not protect a Target
it was not also authorised to write**. And it is what every safety-check Procedure in every composing
repository would have had to do, which teaches a reviewer to discount the one column that matters.

## Why not an outcome an invocation's `when:` could root at

Issue #236's first candidate is the smallest-looking one: give an invocation a single fact about itself
— it completed, or it did not — that a caller's `when:` may root at, without a Record and without a
Step namespace.

**It is vacuous, and the reason is a rule that already stands.** A halt inside a nested Procedure is a
halt of the whole (§6). So an invocation that *did not complete* is an invocation whose Run is already
over: there is no later Step of the caller for the fact to decide, and `{step: verify, …}` would be
true on every occasion it could ever be read. To make it non-vacuous the invoked Procedure would have
to be able to fail *without* halting — a per-invocation failure suppression, which §6 refuses outright
(*no `allowFailure`, no `continueOnError`, nothing an author can write to silence one*) and which would
hand every composing repository a way to run the Steps after a check that failed.

The only non-vacuous version of the candidate is therefore not *the invocation's outcome* but *the
verdict of a Step inside it*, which is the Step namespace ADR-0002 closes. The candidate is refused
because it is either nothing or that.

## Why a Requirement is not a Step

A Requirement is an entry of `steps:` and is not a Step, on the nested invocation's own three grounds
(§6, §7): none of §12's seven Dispositions describes it, it writes no Journal file, and it takes no
position in the sequence. Every member of a Step file — Definition, Operation, Provider, Target, Kind,
Provenance, identities — is about a binding it does not have, and giving it a position would put a row
in the Step table whose every column is empty and a `<nnnn>` in the Journal with no file under it.

What it costs is that a Refusal at one cites no Run position. That is the cost §7 already priced: *the
Step a Refusal names is an artefact coordinate and never an execution fact*, and what an author needs
— file, line, `steps[1].require.<operator>`, the `id:` — is all there. §8 gains a third phase note for
it, derived from the shape rather than from a map of codes: a member naming a `step_id` and no `step`
is a Requirement's and nothing else's.

## Two answers, and which is a halt

**A Requirement that does not hold halts.** It roots at an earlier Step by construction, so its verdict
is always reached after that Step's call went out, which is ADR-0072's criterion exactly: *a guardrail
that declines after a call is a halt, never a Refusal*. The Run is `failed` at `1`, the halt carries no
`error_code`, and `77`'s promise that a verbatim retry refuses identically would be false of it — what
moved is the world, not the artefact. §12's closed sets gain nothing: no `error_code`, no Disposition,
no exit code.

**A Requirement whose predicate cannot decide Refuses**, `predicate-type-mismatch`, exactly as a
`when:`'s does. ADR-0035 governs it and does so wherever the predicate stands: a Record that quietly
failed to compare is indistinguishable from one that compared and did not match. The two keys share one
root, one reader and one grammar, and *which key it was written under* is not a ground for `hyper` to
hold two answers about one fault.

The pair reads as one rule rather than two: **ADR-0035 decides what happens when the predicate cannot
answer, and ADR-0072 decides where the answer lands when it can.**

**A Step the requirement names that acted on no Record leaves it unmet**, which is the condition's own
propagation rule reaching this key's outcome. The named Step was skipped, was never reached, or
resolved an Expansion of nothing; there is nothing for the operator to be true of. It fails in the safe
direction — what a check could not confirm does not proceed — and the halt says which of the two
happened, because the edits differ: an unmet requirement points at the world, and an unanswerable one
points at the Step above it.

## The oracle is edited where the wall was met

`check`'s messages are how an agent discovers this grammar, and three Refusals in ADR-0111 are the
transcript of one discovering the wall. Each of the three spellings is now answered with what it found
rather than with *no id: this Procedure declares earlier*:

- **`step:` naming a nested invocation** — the id *is* declared, one line up — says the invocation
  projects no Record, that no Step of an invoked Procedure is referenceable from its caller (ADR-0002),
  and that a shared Procedure states its own verdict with a `require:` entry inside it.
- **`step:` naming a dotted path** — `verify.archive-intact`, which is the Journal's own path spelling
  and the guess this boundary reliably produces — carries the same second half.
- **`step:` naming a Requirement** says a `require:` makes no call and acts on no Record.

An `args:` reference meets the first of those and is told the first half only: no value crosses an
invocation's boundary in either direction, so there is no way across to name.

## What was considered

- **A `require:` key on a Step rather than an entry of its own.** Rejected because it does not reach
  the case. A shared check's verdict is the *last* thing in it, and a key on a Step needs a Step to
  hang off — so the check would have to end in a Step that does something, which is the artefact
  issue #236 is about. The key is on an entry because the entry is what a check ends with.
- **`args:`, `when:` or `over:` on an invocation, so a caller could parameterise or gate the check.**
  Rejected, and not by this decision: an invocation admits `id:` and `procedure:` and nothing else
  (ADR-0111), and every one of those keys is a value or a predicate crossing the boundary ADR-0002
  closes. What this decision changes is the callee's ability to stop, which crosses nothing.
- **Leaving it, and documenting the cost.** Issue #236's third candidate, and it was the honest option
  while nothing better existed. Rejected because the cost is not a limit an author can work around
  once they know it — `review`'s authority table would keep saying `m` about a read-only artefact on
  every such Procedure in every repository, and a reviewer who learns to discount that column has been
  taught to discount the column that matters most.
- **A marker in the gutter on a Requirement's line.** Rejected: the gutter carries what `hyper`
  derived, and a Requirement's whole content is authored on the line being read — an id, a Step of the
  same file, one field name, one operator. An empty cell would be the gutter claiming to have looked
  (ADR-0026).
- **Giving the halt an `error_code` of its own.** Rejected. A halt carries none, a failure has none
  (§12), and the fact is not a check declining: it is a verdict about the world, reported as the Run's
  fault with the Requirement, the field, the Step it read and what it compared all named in it.
- **Naming the observed value in the halt beside the operand.** Rejected, and it is the one thing a
  reader of that sentence might expect. A `require:` holds of **every** Record the named Step acted on,
  the same rule a `when:` is read under, so there is no one observed value: a sentence carrying one
  would be naming whichever member came first, which is the derivation ADR-0035 keeps every other
  predicate report from making. What the halt names is the artefact's half — `equals: 0`, `in: [0, 1]`
  — which is what an author edits, and the Records themselves are one `hyper changes` away.

## Consequences

- **A shared check gates its callers, claiming nothing.** Issue #236's first acceptance criterion, and
  the shape `release-promotion` asks for is now authorable as stated: one copy of the check, both
  routes running that copy, and neither touching `live/` before it passes.
- **§12's closed sets are unchanged.** Forty-seven `error_code` members, seven Dispositions, seven exit
  codes, eleven operators, three predicate roots. What grew is the entry shapes a `steps:` list admits,
  from two to three, and the positions a predicate is written in, from three to four.
- **`review`'s authority table is right by construction on a check that halts**, which is what makes
  the `OPAQUE`-flagged `mutate` on a read-only artefact unnecessary rather than merely discouraged. No
  new flag and no new marker: the table already says `r` once nothing claims otherwise.
- **The orientation gains a paragraph, and it is the first thing in that text to mention composition at
  all.** Three facts in one place — that a Procedure invokes another, that nothing reaches across the
  boundary, and that a check therefore halts on a `require:` — because an agent that meets any two of
  the three lands on the artefact ADR-0111 recorded. The `require:` fragment is held to `check` by a
  case, on the Bound rule's own footing (ADR-0101). **Issue #237 is not closed by this**: what the
  orientation still does not carry is the invocation's own closed key set, and its worked Procedure is
  still a flat list of Steps.
- **A Requirement is invisible to every derivation that walks `steps:`.** No pair in `AUTHORITY`, no
  contribution to either half of the envelope, no `(Definition, Target)` pair in a projection's `env:`
  block, no Kind and no Repeatability in the Cadence walk, and no change fact — a `when:` is named by
  that vocabulary on no Step either, and a `require:` is the same predicate.
- **A rehearsal evaluates the Requirements it reaches.** They make no call and reach no Target, so
  evaluating one is the rehearsal reading what its own `read` Steps recorded — and a rehearsal that
  walked past the check the operator is rehearsing would answer a question nobody asked (ADR-0010). A
  rehearsal can therefore end `failed` at a Requirement, which is the correct answer and the one the
  operator wanted.
- **A Requirement standing after the last Step the Run holds is still evaluated**, at the end, with
  nothing left to be *never reached*. That is the ordinary shape of a check nobody invoked — run
  directly, it is one `read` and one verdict — and it is the reason the engine reads them by the
  position they stand in front of rather than by the Step that follows.
- **The exit codes an operator sees are the two that already existed.** `1` where the verdict was no,
  `77` where the predicate could not compare. A wrapper that branches on them needs no new case.
- **The authority claim is held by a golden rather than by this paragraph.** `review` over a shared
  check that halts renders `EFFECTIVE r`, no marker on the `require:` line and no flag indexing it,
  which is the inversion issue #236 is about read off the surface a reviewer actually meets
  (`testdata/review/a-shared-check-claims-nothing-to-halt`). The rehearsal consequence above is held
  the same way, both ways round: a requirement that holds leaves the rehearsal stopping at the first
  effectful Step as it always did, and one that does not halts it there instead.
