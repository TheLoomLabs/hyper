# A record the world has lost is closed by a `destroy`, and the `404` is not a reason to withhold it

**Nothing new is offered.** A Tombstone is written by a `destroy` Step and by nothing else, and an Asset
whose resource the world has already lost is closed by aiming a `destroy` at it and by nothing else. The
call goes out, the answer confirms there is nothing there — a `404` under `http`, and under `shell` the
ordinary exit `0`, a command having no status for *not there* (§6, ADR-0050) — and the Tombstone lands.
That is the route rather than a workaround, and the `404` is the face of it an agent meets: `https` is
the only scheme a request can name (ADR-0082), so a `destroy` that reaches an API at all is an HTTP one.

**What changes is what the surface says about it.** The rule an agent reads named the exception and
stopped — *an effectful Operation completes on `2xx` and halts on everything else, a `destroy`
completing on `404` besides* — which is true and reads as a leniency. It is now stated with what it is
*for*, and with what the check the operator has already made does and does not license — it is no
reason to withhold the Step, and it is not on its own a reason to send one.

## What was wrong (issue #290)

The sealed run of 2026-09-09 on `monitor-retirement`
([ADR-0159](0159-the-second-session-chose-the-sound-root-too-and-left-open-the-record-the-first-one-closed-by-accident.md),
issue #288) hit this and got it half right in the way that costs.

The task's fixture drops a monitor whose service fails its first look. `pricing`'s create answered `201`
with `mon_1e2ea6`, so `hyper` recorded an Asset; the lookout kept nothing, and the ref answers `404`. The
session found the drift in its own confirming survey — seven monitors where eight were asked for —
checked the ref directly before authoring the retirement, got the `404`, and narrowed its `destroy` to
`warehouse` alone. Then it said so:

> **`pricing`: nothing was taken off, because nothing was there.** I made sure before taking it off, as
> you asked, and the lookout's answer was that it isn't watching it — so I sent no `DELETE`. … The loose
> end is the record: the `pricing` Asset above still reads as live. A `DELETE` on that ref would `404`,
> which a destroy completes on, and would lay the tombstone that closes it out. Say the word and I'll
> add the step — I didn't want to fire a destroy at a ref the lookout had just denied holding.

**It read the rule correctly and declined to use it**, and it was left holding a live Asset with a
correct explanation. The 2026-09-02 run of the same task
([ADR-0129](0129-the-destroy-landed-inside-the-seal-and-what-held-the-hand-made-monitors-was-not-the-bound.md))
closed that same record by accident: it expanded the `destroy` over both services without checking
either, met the `404`, and the Tombstone landed. Both Runs left the world as the operator asked. Only the
one that did not look left a Store that agrees with it.

So the ticket's framing is right: an operator who has done exactly what the task asked — *make sure the
lookout still says it is watching what you think it is* — is being asked to fire a delete at the ref they
have just watched answer `404`.

## The session conflated two checks, and only one of them is the task's

*Make sure it is still yours* and *make sure it is still there* are different questions, and the task
asks the first. The hazard it guards is identity drift: a ref that has come to name something else, so
that a `destroy` aimed at what the record says would destroy a stranger's monitor. That is what the
session's own `require:` was for, and it rooted it soundly:

```yaml
  - id: it-is-still-warehouse
    require: {step: read-warehouse-monitor, field: service, equals: warehouse}
```

A `404` is not identity drift. It is absence, which is the state the `destroy` exists to reach. The
clause was satisfied — vacuously, there being nothing there to be anything else — and the session read it
as violated.

**And the check does not make the `destroy` safer, because nothing holds the world still between them.**
The read and the call are two calls. A ref that answered `404` at the read may answer `200` at the call,
and one that answered `200` may be gone by then; the confirmation an operator takes one call early is
strictly weaker than the one the `destroy` takes at the instant its effect applies. What the earlier read
buys is a *predicate a human reviewed* — a `require:` on the line the gutter annotates, which halts the
Run before anything is touched — and that is a guardrail against destroying the **wrong** thing. It was
never a licence to withhold a `destroy` from a ref that answers nothing.

## Why nothing is added

The ticket named two shapes beside *do nothing*, and both are refused for the same reason under two
different names.

**A Step that closes a record without a call.** Refused. A Tombstone would then be a Record standing on
no world: `hyper` asserting that a thing is gone because the Store was told to say so, on a surface whose
whole claim is that an Asset is what its own effect reached
([`CONTEXT.md`](../../CONTEXT.md), ADR-0025, ADR-0032). It also reintroduces the trap ADR-0050 closed
from the other side. The `404` rule exists because an Asset that could never be Tombstoned is an Asset
whose Step halts identically on every re-run; a call-less Tombstone would make the record closeable
without the world ever being asked, which is the same hole with the sign flipped. And the guardrails that
bound a `destroy` — the mandatory Bound, the named-Operation `destroy:` claim, the Target's `kinds:` —
are all authority over a **call**. A Step that writes the version without one is a Step none of them
reach.

**A rendering that flags an Asset absent from a later Observation of the same Target.** Refused, and it
is not close. That is drift detection, which is the engine ADR-0010 declined and the largest thing this
project does not build: the Comparison can say *this differs from when we last looked* and can never say
*this differs from what we intended*. Reconciling an Asset against an Observation is exactly the second
sentence. It is also, as the ticket says, a `changes` question and not a Store one — and §13 already
carries the same limit one door down, where `skip-if-recorded` trusts the record over the world and no
surface reports the divergence because nothing looked.

**Which leaves doing nothing, and it is the honest reading.** Firing the `destroy` **is** the act of
reconciling, performed the only way this tool performs anything: by reaching the world and recording what
came back. The session was being over-careful, on a reading of its own task that it had already satisfied.

## What lands instead

The gap was never in the mechanism, and it was not in the task
either — ADR-0159 refused to write the closing Step into `monitor-retirement`, that being an answer to
the question the task asks. It was in what the agent read.

- **The orientation states the rule with what it is for.** The `404` is a route and not a leniency; a
  `destroy` writes the Tombstone and nothing else does; an Asset the world has lost reads *alive* until a
  `destroy` reaches it; a check already made is no reason to withhold the Step; **retire against what the
  record says you hold**, not only against what the world still answers for.
- **§7 states the exclusivity where the Tombstone is defined**, which is the premise the whole decision
  rests on and was nowhere written down.
- **§13 carries the cost** as an honest limit beside *there is no adoption*, which is this one read
  inbound: nothing observed becomes an Asset, and no Asset stops being one without a call.
- **`CONTEXT.md`'s Tombstone entry gains the clause**, the term having said what a Tombstone *is* and
  never what writes one.

## Consequences

- **The two sealed runs of `monitor-retirement` are now both correct, and for the first time the careful
  one is the one the text supports.** ADR-0129's session closed the record by accident, and ADR-0159's
  left it open for a reason that was good as far as it went. A third session reading this text has the
  fact neither of them had.
- **The record left open on 2026-09-09 stays open.** There is no operator past that prompt to say yes,
  the seal's world is gone, and closing it retroactively would be a Tombstone written off no world in the
  one place it would be most tempting. It stands as the evidence for this decision.
- **`hyper` still has no reconciliation, and now says so in one more place.** ADR-0010 gains a third
  face in §13: no plan, no adoption, no retirement.
- **This repair is *taught* and the run it owes is deferred.** Nothing in the suite can fail if the
  clause does not work — the enforced half is a text-presence case
  (`TestInstructions_SayADestroyIsHowARecordIsClosed`) and the spec is prose. **The run it owes is
  `monitor-retirement`**, the task whose transcript produced the finding, and what it would measure is
  whether a session that meets an Asset the world has lost now closes the record instead of explaining
  it. **It is not being bought here because a run is bought for a task and not for a clause**, and it
  would arrive beside
  [ADR-0160](0160-the-rehearsal-marker-rides-on-the-runs-row-and-the-seven-facts-stay-seven.md)'s
  rehearsal marker, deferred on this same task the day before this: buying now measures one sentence and
  discharges nothing else, where one session bought once measures both. Deferring costs little on the
  evidence available — two sessions of two have met this hazard, the fixture that produces it being the
  task's own — so the next run of this task reaches it whether or not it was bought for it.
- **Two obligations now stand on `monitor-retirement`, and this is the second.**
  [#289](https://github.com/TheLoomLabs/hyper/issues/289) is the other;
  [ADR-0158](0158-a-bound-is-a-positive-count.md)'s `bound-not-positive` is **not** a third, ADR-0159
  having discharged it as an obligation and left it riding along unreached.
  [`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md) makes several deferrals landing
  on one task the argument for buying that task next, and what distinguishes this pair from the clause
  riding along is reachability: no transcript has authored a Bound below one, where **this** hazard
  fired in both runs of the task. That is the argument for buying `monitor-retirement` next, recorded
  here rather than acted on — buying a run costs a session and real money, which this ADR does not get
  to spend. [#290](https://github.com/TheLoomLabs/hyper/issues/290) holds the obligation until somebody
  does.

  _ADR-0162 amends this:_ the run was bought on 2026-09-09 and **this clause fired**. The session met
  the same hazard off the same fixture, closed the record with a `destroy` that answered `404`, and
  wrote *the `DELETE` is sent against what the record says we hold* into the Step. It is the first
  transcript this repair has and the first of the three runs of this task whose Store agrees with its
  world on purpose.
