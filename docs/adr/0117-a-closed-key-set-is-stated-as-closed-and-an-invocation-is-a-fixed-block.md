# A closed key set is stated as closed, and an invocation is a fixed block

**The orientation and §3 now state that a nested invocation admits an `id:` and a `procedure:` and no
other key, what that closure costs an author — a shared Procedure is a fixed block rather than a
parameterised one — and that a halt inside an invoked Procedure halts the whole Run.** Two clauses on
the sentence that already stood, one sentence beside it, and a case that holds the closure to what
`check` actually answers.

Nothing about composition moved. `invocationDeclaration` is untouched, §6 already stated the halt, and
the shared check the orientation teaches is the one issue #236 landed. What changed is a pair of
documents that left open a set the binary closes, and left the halt stated where the reader who needs
it does not stand.

## The claim as it stood, and the set underneath it

Both channels of the orientation (ADR-0093, ADR-0095) and §3 said the same thing about the shape:

> A nested invocation names an `id:` and a `procedure:` in place of `definition:`/`operation:`/`target:`

That is true, and it reads as *those three are replaced and the rest still apply* — `args:`, `over:`,
`bound:` and `when:` among them. The binary reads it the other way. `invocationDeclaration` in
`internal/artefact/procedure.go` declares exactly two properties, and `additionalProperties: false` is
forced rather than authored (§12), so **every other key is `unknown-key` at the position it is written
on**.

The consequence is not syntactic. A Procedure is fully bound (ADR-0008), and an invocation carries no
values across, so a Procedure invoked by two callers does the same thing for both: **what varies
between two callers is two Procedures, not two invocations.** That is a fact an author designs around
before writing anything, and neither document stated it.

## The evidence: the enumeration the text made someone pay for

The sealed acceptance run of 2026-08-30 — `scripts/acceptance/run.sh`, `release-promotion`, read in
[ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)
(issues #225, #232). The task asks for one shared archive check invoked by two routes, and the session
had to establish the whole grammar of composition from `check`:

- **That composition exists at all** it guessed, wrote `procedure: verify-archive` into a Step, and
  confirmed by deliberately breaking the name and reading `artefact-absent` back.
- **What an invocation admits** it enumerated the only way the sentence left open: fourteen invented
  keys written onto one invocation in a single call, and fourteen `unknown-key` rows back — a closed
  key set read out in one round trip.
- **That a halt propagates out of a callee** it could not establish, and said so in its handback rather
  than claiming it:

  > **The load-bearing assumption**: that an effectful step failing *inside* `verify-archive` halts the
  > whole Run, so `promote`'s later steps never execute. […] I could not confirm it propagates out of a
  > sub-procedure, and it is the property the whole safety story rests on.

Roughly twenty-seven of the run's fifty-nine calls went on discovering the grammar this way, against a
`fleet-rollout` run that authored a flat Procedure in one draft and spent twenty-seven calls on the
whole task (ADR-0110). The run cost $3.65 and thirteen and a half minutes.

**That the oracle absorbed it is the finding rather than the consolation.** `check`'s messages are good
enough that the set is recoverable from them, which is ADR-0099's finding holding under pressure; the
price is half a session, paid every time an author reaches a corner the orientation skipped. And the
third fact was not recoverable at all offline: the artefact the run produced is correct, and its
reviewer was handed its central guarantee marked *I could not confirm this*.

## What was decided, and against what

**A set the binary closes is stated as closed.** That is the general form, and it is
[ADR-0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)'s rule read off the
other axis: a rule stated without its exception is believed and authored against, and a set stated
without its closure is *enumerated* — by an agent, against the oracle, at whatever a call costs. Both
failures are the same document teaching half of what the binary holds.

**The closure is stated with what it costs, not only with its keys.** *No other key* is the syntax;
*a shared Procedure is a fixed block rather than a parameterised one* is the sentence that changes what
an author builds. The run that met this redesigned around it — the check derives both release
candidates itself rather than being handed the one its caller cares about — which is the design the
first sentence would have produced a call earlier.

**The halt is stated as the whole Run's, and not only as the `require:`'s.** Issue #236 landed *a
`require:` that does not hold halts the Run — here and in whatever invoked this Procedure*, which is
true and is narrower than §6: **whatever** halts inside an invoked Procedure halts the whole Run, an
effectful Step's non-`2xx` included. Stating it of the `require:` alone leaves exactly the assumption
the sealed run could not confirm. It is now one clause of the composition sentence, and the `require:`
sentence loses the depth clause it was carrying rather than restating it.

**A section about composition was rejected**, as it was for the Bound. The orientation's length is a
design constraint rather than an aesthetic one (ADR-0093, ADR-0096) — it is paid on every session in
every harness, as a handshake field whether or not the model reads it and as a file the harness reads
up front — and the acceptance criterion issue #237 wrote for itself is that the facts land where an
author authoring a Procedure already is. They are clauses of the paragraph issue #236 added.

**Enumerating the refused keys in the orientation was rejected; §3 enumerates them.** *No other key* is
the whole rule, and an agent that reads it needs no list; §3 names `args:`, `over:`, `bound:` and
`when:` because the specification is where a reader goes to be told what a position admits, and because
the Requirement's sentence one clause later already reads that way (§3, ADR-0116).

**Making an invocation parameterised was not considered here.** It is a change to the model, not to a
document: a Procedure is fully bound (ADR-0008), an argument crossing an invocation would be authority
travelling in a key a reviewer reads last, and the ticket that would carry it is not this one. What
this ADR decides is that the constraint is stated where it is designed against.

## The seam the fence is expressed at

`TestInstructions_TheInvocationKeySetItStatesIsTheOneCheckHolds`, in `internal/cli` for the reason the
worked example's cases are there: a `check` is not reachable from `internal/mcp` and must not become
so, the surface handing a Call to a dispatch and knowing no command.

It drives **three repositories through `check`** — an invocation carrying nothing beside its two keys,
one carrying an `args:`, one carrying a `when:` — and asserts the answers: clean, `unknown-key`,
`unknown-key`. Then it pairs each answer `check` declines with the word the orientation must carry for
an author to have avoided writing it: the `args:` case requires the text to name `args:`, the `when:`
case requires *no other key*. `invocationSentence` holds the budget beside it — **exactly one**
sentence of the orientation states what an invocation admits, so the repair cannot grow into the
section the text may not become — and it is `boundSentence`'s own assertion, both now reading the one
`theOneSentence` walk under their own marks and their own failure line.

**The halt is fenced elsewhere, and by a Run rather than by a `check`.** No `check` can hold it — the
claim is about what a Run does — and `internal/cli/testdata/run/a-halt-inside-a-nested-procedure`
already holds it: a halt inside the callee, and the caller's own next Step rendered `never-reached`.
What the case beside the text adds is that the sentence is present at all
(`TestInstructions_SayHowAProcedureComposesAndHowASharedCheckHalts`). The narrower shape ADR-0111's
session actually asked about — an *effectful* Step failing inside a callee — has no golden of its own,
the halt path being the Run's rather than the Step's, and that is worth one when a fixture next needs
building rather than on its own.

**What the fence cannot do is read.** Prose has no schema, and a sentence carrying every required word
and still misleading would pass. What it holds is that the orientation names the outcome `check` has
for the two keys an author reaches for first, in one sentence, in the vocabulary the other surfaces
answer in — which is the specific failure that happened.

## Consequences

- **The orientation grows by two clauses and a hundred and forty-nine characters**, on both channels
  at once, the two being one text (ADR-0095). No golden carries the body — a tree golden holds the
  line *the orientation, verbatim* — `project`'s behaviour is unchanged, and an `AGENTS.md` already
  written keeps its old text until somebody regenerates it, which is `project` never overwriting a
  file that stands.
- **§3 says the key set is closed**, so the specification and `invocationDeclaration` agree. A careful
  reader of §3 would have written `when:` on an invocation and been refused; now the sentence says what
  the position admits and what the closure costs.
- **§9 states the rule beside the exception rule**, and the count of what the orientation states is
  unchanged: *a set the binary closes is stated as closed* is a constraint on the eleven, not a twelfth
  thing to state. The composition bullet's own count moves from three facts to five.
- **`CONTEXT.md`'s **Procedure** entry gains the same two clauses**, in the register the entries are
  written in — a nested invocation names the Procedure it invokes and passes it nothing, and a halt
  anywhere inside it is a halt of the whole Run. No term is added and none is renamed.
- **Nothing in the binary moved.** `check`, `run`, `review` and the schemas are untouched; the two
  cases read them rather than change them.
- **The next transcript is what says whether this worked.** ADR-0111's run reconstructed the key set
  from fourteen Refusals and handed its reviewer an assumption; the claim here is that a run against
  the same task now writes the invocation once and states the halt as a fact. A run that never composes
  would say nothing either way.
