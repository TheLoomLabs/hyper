# The spelling was right the first time, and the three repairs behind it were never reached

**[ADR-0152](0152-secret-is-named-on-every-surface-that-renders-an-operation.md)'s clause landed, and
this is the first evidence it has.** A sealed session read `AGENTS.md` at its fourth call, authored
`secret: [token]` at its twenty-first in the position §3 names, and never went looking. Against
[ADR-0149](0149-the-sink-was-supplied-both-secrets-were-lost-and-check-was-clean-throughout.md)'s
session on the same task — thirteen `Bash` calls hunting the key, eight of them `strings` and `nm`
over `bin/hyper`, two live credentials destroyed by the wrong spelling — this one spent **zero**.
Issue #277, and the run four repairs deferred.

**And that took the other three axes with it.** ADR-0151's `check` message fires only for an author
who writes the declaration inside `fields:`; `secret-sink-unfilled` fires only for a sink nothing can
fill; both sink usage errors are met only by an invocation that gets the path wrong. This session did
none of those things, so none of those surfaces was reached. **The run measured the one axis it could,
and the other three were unreachable because the first one worked** — which is a result rather than a
shortfall, and is stated as one below.

## The evidence: thirty-nine calls, two Runs, no Refusal, exit `0`

A Claude Code session, headless, 2026-09-07, inside the seal
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md),
[ADR-0130](0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md)), against
`push-credential` (issue #271) and the local TLS endpoint
([ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)).
Thirty-nine tool calls, forty turns, four minutes forty-one, **$1.67**, exit `0`, `subtype: success`.

Eighteen `Bash`, five `check`, three `review`, two each of `operation`, `run`, `changes`, `probe` and
`ToolSearch`, and one each of `providers`, `records` and `run_show`.

**It is the second cheapest sealed run in the corpus and has the second fewest calls** — behind the
empty-credential run's 32 and $1.43
([ADR-0147](0147-the-empty-credential-was-read-as-a-state-and-the-agent-went-looking-for-what-emptied-it.md))
and `change-window`'s 20 and $0.73
([ADR-0120](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md)),
which is a smaller task. What matters is not the rank: **it is the same task as ADR-0149's, and it cost
39 calls against 81 and $1.67 against $4.28.**

**Two Runs, both completed, and nothing Refused anywhere.** `survey-lookout` (call 17, one `read` Step,
two Records) and `enrol-push-heartbeats` (call 32, four Steps, all `ran`). Five `check` calls, every one
of them zero rows.

## The repair landed

Call 4 is `cat AGENTS.md`. Call 21 appends the mint Operation, verbatim in the part that matters:

```yaml
  issue_credential:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /v1/monitors/{ref}/credential
    input:
      type: object
      properties: {ref: {type: string}}
    record:
      identity: "{ref}"
      fields:
        id: $.body.data.credential.id
        monitor: $.body.data.credential.monitor
        service: $.body.data.credential.service
        issued: $.body.data.credential.issued
        token: $.body.data.credential.token
    secret: [token]
```

**One call, first draft, correct position, and nothing between the reading and the writing.** No
`--help`, no `strings`, no `nm`, no candidate matrix. The transcript holds not one call whose subject
is *where does this key go*.

**`AGENTS.md` is the only surface in hand that carries the spelling.** `docs/lookout-api.md` documents
the API and never the Manifest (ADR-0105) — it says *put it somewhere before you lose the answer* and
names no key. `providers` at call 6 answered the built-in `shell` Provider, which declares no `secret:`
and does so deliberately (§3). `operation` on `shell read` at call 9 answered `"secret_fields":[]`,
which carries the concept and not the authoring key. The clause ADR-0152 put in the orientation is the
one place the session could have got this, and it read that file seventeen calls before it needed it.

**ADR-0152's other three surfaces were all reached, and each rendered what that record said it would.**

- `operation` at call 23, on the session's own Operation: `"secret_fields":["token"]` in the derived
  block, beside the verbatim source.
- `probe` at call 25: `"token":"<secret>"` in the projection and `"response":"<secret>"` for the whole
  object — the session's own words, *both projections resolve with nothing unresolved, and `token`
  renders as `<secret>`*.
- `review` at call 30: `secret token` in the gutter on the Operation's key line, and
  `SECRET   line 47   issue_credential declares secret output: token` in the flag block. The session
  read it back as *reads as intended — `SECRET` flagged on `issue_credential`*.

**The sink was named up front and the session was never Refused into it.** Call 32 is the Run, carrying
`secret_sink` on the first attempt, and it completed: four Steps `ran`, two credentials minted,
`0002/mon_58c3d1/token` and `0004/mon_1e6a05/token`, `0700` on the directories and `0600` on the files.
That is ADR-0148's shape met for the second time and the sink gate not met at all — the same behaviour
ADR-0149's session showed at *its* call 41, and the one thing the two sessions did identically.

## What was not reached, and why it is the same fact

**ADR-0151's `check` message did not fire.** It fires on a `secret` key written inside `fields:`, which
is the spelling ADR-0149's session shipped. This one never wrote it, so the message that would have
caught it had nothing to catch. That record's own framing is the reading: the `check` rule catches an
author *after* writing the wrong thing, and ADR-0152's clause is about their not writing it. **The
clause worked, so the net was never touched.**

**`secret-sink-unfilled` did not fire.** Its condition is a sink named against a Run no Step can fill;
this session named a sink against a Procedure with two secret-producing Steps.

**Neither sink usage error was met, for the second run in a row.** The session picked
`/home/idabic/acceptance-277-run/lookout-credentials` — outside the working tree, not there yet —
first try, exactly as ADR-0149's did. **Issue #272's third question is unmeasured after two runs of
this task**, and two sessions choosing correctly without hesitation is now the finding rather than the
gap: the `secret_sink` description says what the argument wants and both agents supplied it.

**None of the three is evidence that its repair does not work.** They are three surfaces whose
condition is an author's mistake, met by two sessions of which one made a mistake — before the clause
— and one did not. What would measure them is a session that gets the spelling wrong *with the clause
in hand*, and nothing can arrange that.

## The fourth axis arrived through the description rather than through the state

ADR-0150's line renders beneath the Step table when a `skip-if-recorded` Step skips a member. This
session ran the Procedure once, so no member was ever already recorded and the line never rendered.

**It reported the behaviour anyway, unprompted, in its closing paragraph:**

> One thing to know before you paste: re-running `enrol-push-heartbeats` will *skip* both credential
> steps, because `skip-if-recorded` finds the Records already there — it reports `secrets_skipped` and
> writes no file. That is the safe behaviour, not a failure, but it means the sink will not refill
> itself.

It names the member by name having never seen one. The only place it can have come from is the `run`
tool's `secret_sink` description, which is where ADR-0148 and ADR-0150 put it — so **the taught half of
ADR-0150 reached an agent that never met the state**, and reached it well enough to be handed on to the
operator with the right reading attached: *the safe behaviour, not a failure* is the sentence that
record was written to make available.

**That is not the measurement the line owes.** Whether a session that meets a short sink and the line
beneath a `ran` row believes it is a different question, and it needs a Run over standing Records.
`push-credential-already-recorded` is the task that reaches it, landed with this ticket, and its run is
not bought here.

## The two sessions differ in more than the clause

ADR-0149's session authored one Step with an Expansion over a literal list of refs, having read them
back off its own Records. This one wrote **four literal Steps** — a `read` of each monitor and a
`mutate` behind it, each credential call gated by
`require: {step: dispatch-monitor, field: service, equals: dispatch}` — so that *a ref pointing at the
wrong service halts the run instead of handing a token to the wrong place*. Both guards held.

**So the call counts are not a controlled comparison and are not offered as one.** What is attributable
is narrower and is the thing the ticket asked for: ADR-0149's thirteen calls had one subject, finding
the key, and this transcript spends none on it. **One session's mistake was not a rate and one
session's success is not one either** — that sentence is ADR-0149's and it holds in both directions.

## What the run establishes, and what it does not

**Establishes.** The orientation clause is sufficient on its own: an agent with no Manifest to copy and
no specification on disk authored `secret: [token]` correctly from it, in one draft, in the position §3
fixes. All four of ADR-0152's surfaces render the fact and three of them were read back by the session
in its own words. The `secret_sink` description carries ADR-0150's `secrets_skipped` far enough that an
agent volunteers it to an operator. `<secret>` holds on `probe`, `changes` and the Store — the session
grepped both token values against the working tree and every object on every branch, `hyper-store`
included, and found neither.

**Does not establish.** ADR-0151's `check` message and `secret-sink-unfilled` are unfired and remain
unmeasured. The two sink usage errors are unobserved for the second time. ADR-0150's line has still
never been rendered in front of an agent. And a run whose author had spelled `secret:` wrongly would
have measured three of those and this one measured none — **the finding is upstream of a mistake that
was not made.**

## What was considered

**Reading this as a thin run.** Rejected, and `docs/agents/acceptance-re-runs.md` settles it: *a run
where the taught clause did its job and nothing new appeared is a good result and the first evidence
the repair has — it is not a wasted run and it is not written up as one.* ADR-0120 is the worked
example and this is a second. The clause under test is the one that produced the whole family: ADR-0151
and ADR-0152 both exist because a session could not find `secret:`, and a session that finds it in one
call is the answer to the question all four records were waiting on.

**Buying a second run immediately, against `push-credential-already-recorded`.** Rejected here and left
to the ordinary decision. These runs are a handful a year and each costs a session and real money
(#221, #250); the fence is landed, the axis is named, and the deferral is written down, which is what
the rule asks for.

**Amending ADR-0149, ADR-0150, ADR-0151 or ADR-0152.** Not available and not wanted.
[`docs/adr/README.md`](README.md) fixes the convention — a later record that revises an earlier one says
so in its own text and the earlier one stays as it was written — and none of those four is revised by
this. They were right when written; this is what they were waiting for.

## Consequences

- **ADR-0152's obligation is discharged.** The clause is measured and it landed. Issue #277 is what
  discharged it.
- **Three obligations survive this record.** ADR-0151's two surfaces are unfired; ADR-0150's line is
  unrendered; issue #272's third question is unmeasured for the second run running. The first two are
  unfired *because* the repair above worked, and the third is two sessions choosing the same correct
  path. None is a defect and all three stay open.
- **`push-credential-already-recorded` is the task the unrendered line is now fenced by**, landed with
  this ticket: `push-credential`'s prompt with one paragraph asking for the credentials one at a time
  and then for the rest by running the same thing again. Its own run is deferred in writing, which
  `docs/agents/acceptance-re-runs.md` names as a complete answer.
- **`push-credential` now has two transcripts and they can be read against each other.** That is what
  made this run worth its money: the baseline existed, the surface between them moved by one clause,
  and the difference is legible. The task file did not move, which is why.
- **The seal held and nothing new was reachable.** The `CLAUDE_*` variable the session found in its own
  environment is the messaging token `run.sh`'s `--clearenv` comment already names, and `bin/hyper` was
  not read this time — there was nothing to reverse-engineer.
- **The corpus holds 153 records.** `docs/adr/README.md` counts them and carries this one on the *Field
  reports* path.
