# The orientation taught the envelope, and the first Requirement was authored from one sentence

**A sealed session read `change-window`, wrote five artefacts in one call, and declared
`targets: [control, local]` on both routes before `check` ever saw them.** `envelope-exceeded` was
reachable — the same repository with those two lines cut to `[local]` refuses on both, one row each,
exactly where issue #238 said it would — and the session never got near it. That is the first of
[ADR-0118](0118-the-envelope-cannot-be-trapped-now-that-the-orientation-states-it.md)'s two outcomes:
**the clause landed.** §5's containment rule met an agent as a sentence rather than as a Refusal,
which is what issue #221's sixth user story now has.

**And the Requirement is measured.** The shared check is three `read` Steps and three `require:`
entries, `targets: [control]`, `review` rendering it effective `r` — the artefact
[ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)
could not have, written in a first draft by a session that was never told the shape existed except by
`AGENTS.md`. Issue #240, and the run half of issue #238.

## The evidence: twenty calls, three `check`s, and not one Refusal

A Claude Code session, headless, 2026-08-31, inside the seal
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)), against
`change-window` (issue #238, ADR-0118). Twenty tool calls, twenty-one turns, **two minutes fifteen,
$0.73**, exit `0`.

Six `Bash`, six `review`, three `check`, two `operation`, one `providers`, one `provider`, one
`ToolSearch`. It is the fewest calls and the lowest cost of any sealed run recorded here — against
`fleet-rollout`'s 27 calls and $1.19
([ADR-0110](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md)),
`tenant-onboarding`'s 30 and $1.23
([ADR-0112](0112-the-second-run-skipped-what-the-first-did-and-every-revision-it-recorded-is-unresolvable.md)),
`monitor-coverage`'s 52 calls and two dollars (ADR-0106), and `release-promotion`'s 59 calls and $3.65
(ADR-0111).

**Not one Refusal, and not one repair driven by `check`.** All three `check` calls came back with
zero rows, the first of them over the whole repository immediately after the artefacts were written.
That makes it the third transcript never *taught* by a Refusal, after `fleet-rollout` and
`monitor-coverage` (ADR-0110), and the opposite face of ADR-0111's eighteen `check` calls with twelve
refused. What separates the two groups is not obviously the agent: it is at least whether the
orientation happens to state the thing the task needs.

**Nothing was run and nothing foraged.** The task ends at the diff and the session ended there,
handing back `git diff` with `firewall/allow`, `requests/` and `control/` untouched on disk. Its only
`find` was `find . -path ./.git -prune -o -print` inside the repository at call 1. **So this run says
nothing about the seal** — it never went looking, and the direct evidence stays where it was:
ADR-0111's `find` over `/` from inside the namespace, and ADR-0106's `find` over `$HOME` and `ls -a`
of the output directory, which is what ADR-0109 was decided on.

## The clause landed, and the trap was live behind it

Call 9 is a single `Bash` heredoc writing all five artefacts — two Definitions and three Procedures —
with no intervening call. Both routes open:

```yaml
kind: procedure
procedure: grant-pending
targets: [control, local]
steps:
  - id: permitted
    procedure: change-control-permits
```

The declaration was correct before anything checked it, so nothing had a chance to refuse. **The
result is not an accident of the design, and that is what the fixture bought.** `change-window` gives
the routes a shared check whose product is a halt rather than a value (ADR-0118), so neither route's
own Steps bind `control`; the name is in the envelope for one reason only, which is the invocation.
`release-promotion`'s transcript could not say that (ADR-0111) and this one can.

**The Refusal was reachable.** Against the session's own artefacts, with the two `targets:` lines cut
to `[local]` and nothing else touched:

```
FILE                              LINE  FIELD               ERROR_CODE         MESSAGE
procedures/grant-pending.yaml     6     steps[0].procedure  envelope-exceeded  procedure: change-control-permits's transitive envelope reaches control, outside this Procedure's own declared targets:
procedures/revoke-withdrawn.yaml  6     steps[0].procedure  envelope-exceeded  procedure: change-control-permits's transitive envelope reaches control, outside this Procedure's own declared targets:
```

and `review` renders `envelope ✗` against line 3 with `ENVELOPE line 3 exceeded — a step reaches
outside [local]`. Two rows, one per route, the repair one edit per file: the whole of what
`change-window.setup.sh` predicted, sitting one line away from what the session wrote.

**The reading is demonstrated rather than inferred from the result.** The session's closing report
states the rule back in its own words, on the line the task asked it to account for:

> **The envelope.** `targets: [control, local]` on line 3 of each procedure; review confirms
> `ENVELOPE line 3 ok — no step reaches a target outside [control, local]`. The gate's own
> `targets: [control]` counts against that envelope, which is why `control` is listed on the callers.

That is [ADR-0117](0117-a-closed-key-set-is-stated-as-closed-and-an-invocation-is-a-fixed-block.md)'s
clause paraphrased — *its `targets:` count against the caller's declared envelope* — carried from a
sentence in a section about how a check halts to a `targets:` line three keys away. It is the one
thing ADR-0118 said a run could establish and a fixture could not arrange.

**So `envelope-exceeded` is unmet by every transcript, and is now unmet on purpose.** It is a fact
about the documentation, and this ADR is where a reader learns it without paying for a sixth run.

## The first Requirement, and the first halt it made

`procedures/change-control-permits.yaml`, its first two entries as authored:

```yaml
kind: procedure
procedure: change-control-permits
targets: [control]
steps:
  - id: read-window
    definition: change-control
    operation: read
    target: control
    args:
      command: [sh, -c, 'test "$(cat control/window)" = open']

  - id: window-open
    require: {step: read-window, field: exit_code, equals: 0}
```

— and twice more for the freeze and the approver. `review` renders it
`change-control | control | read | read | r | —`: **read-only in authority terms, and it stops both
its callers.** That is §5's sentence with this exact artefact under it, and ADR-0116's shape meeting a
transcript for the first time.

**The session named the alternative it declined, and said why:**

> I used `require:` rather than an effectful step that exits non-zero deliberately — the gate's
> AUTHORITY table is `read` only, and stays that way.

The effectful step that exits non-zero is what ADR-0111's session had to write, and what the
orientation now warns against in one sentence — *never reach for an effectful Step that exits
non-zero in order to make a check able to fail*. Both halves of issue #236's fix are therefore
answered by the same call: the shape is authorable from the orientation, and the workaround it
replaced was not reached for.

**Run afterwards by hand, against the artefacts as the session left them.** Committed, `check` clean
over eleven artefacts, then:

- `run grant-pending` — `completed · exit 0`, four Steps `ran`, both pending rules appended to
  `firewall/allow` and `requests/pending` left empty.
- `run revoke-withdrawn` — `completed · exit 0`, the withdrawn rule off the allow-list,
  `requests/withdrawn` empty, no temporary file left behind.
- `control/window` set to `closed`, then `run grant-pending` again:

```
step 1/4 change-control-permits.read-window
hyper run: requirement change-control-permits.window-open did not hold: exit_code of the Record step
read-window acted on does not satisfy equals: 0, and no Step after this line runs
STEP  ID                                    KIND    DISPOSITION    RECORDS
1     change-control-permits.read-window    read    ran            1
2     change-control-permits.read-freeze    read    never-reached  –
3     change-control-permits.read-approver  read    never-reached  –
4     grant                                 mutate  never-reached  –

failed · exit 1
```

`firewall/allow` byte-identical, `requests/pending` untouched. **A Requirement inside an invoked
Procedure halted its caller's later Steps**, which is the load-bearing property ADR-0111's session
could not confirm and had to hand the reviewer as an assumption. It is now demonstrated end to end,
against artefacts an agent authored blind.

## None of the three wrong answers, and the limit `hyper` does not enforce

`change-window.setup.sh` enumerates three repositories that pass `check` and are not the repair. The
session gave none of them: one copy of the check, its reads bound to `control`, and the halt kept —
so the closing question about what each route may touch is answered off the artefacts rather than off
a design that gave the answer away.

It also found the limit the task states and `hyper` does not enforce, without being asked:

> One thing the artefacts do *not* say, and I want to be plain about it: the effect is **opaque**.
> […] `hyper` can tell you these steps hold `mutate` on `local`; that they touch `firewall/allow` and
> `requests/` and nothing else is knowable only by reading the `command:` line. That is the shell
> Provider, not something I can tighten away.

That is ADR-0111's best moment reproduced by a different session against a different task, which
makes it a property of the review surface rather than of one run.

## `hyper targets` was never called, and the fixture assumed it would be

The setup script's ground for shipping the second Target rather than asking for it ends: *it still
has to be found: the task names no Target, so reaching it goes through `hyper targets`, which is step
one of the loop the orientation opens with.* It was found at calls 1 and 3, by `find` and `cat`. The
tool was loaded — `ToolSearch` selected `mcp__hyper__targets` by name at call 2 — and never used,
while `providers`, `provider` and `operation` were all called, because those answer things the
filesystem does not.

**An agent with file tools reads a small repository off disk**, and step 1 of the loop is a route
to the Targets rather than the route. Nothing is wrong: the second Target was found, bound correctly,
and rendered in the AUTHORITY table the session quoted back. What is worth writing down is that a
fixture cannot buy a measurement of a *listing surface* by declining to name what it lists.

## Two defects

Both are rendering-and-orientation findings rather than faults in a rule, and both are ticketed with
this transcript as their evidence.

**Issue #241 — an opaque `mutate` buys off its `UNBOUNDED` flag with the one number `bound-illegal`
forbids on an opaque `destroy`.** The session's first draft carried no `bound:` and `review` answered
`mutate!` in the gutter with `UNBOUNDED  line 8  step grant  mutate with no declared bound`. It added
`bound: 1` to both effectful Steps, the flag cleared, and the gutter dropped to `mutate` — with the
`command:` line unchanged. §4's argument against a Bound on an opaque `destroy` is that *the only
value a single command could carry is `1`, which would render as a promise the Step cannot make*, and
that argument does not stop at `destroy`: `cat requests/pending >> firewall/allow` appends two rules
and truncates a file, and the reviewer of the second version sees strictly less alarm than the
reviewer of the first. §12's `unbounded` bullet has two forms and neither is this one.

**Issue #242 — the orientation's only worked Requirement gates on `exit_code`, so the reviewable fact
ends up inside the opaque command.** All three of the session's pairs are `sh -c 'test …'` plus
`require: {field: exit_code, equals: 0}`, faithfully following the `sha256sum -c` example. The result
is that *the window must read `open`* is stated nowhere a rendering can show it: `review` marks each
Step `OPAQUE` and the `require:` line says only that an exit code was zero. The stronger spelling
works — `command: [cat, control/window]` with
`require: {step: read-window, field: stdout, equals: "open\n"}` passes `check` and runs — and nothing
in the orientation points at it.

## What the run establishes, and what it does not

**Establishes.** The envelope clause in the orientation is carried to the `targets:` line by an agent
that has never read §5. A Requirement is authorable from `AGENTS.md` alone, and the effectful-halt
workaround it replaced is not reached for. A shared read-only check halts its caller, end to end, in
a Run. Composition, the read-only Target and the declared policy `hyper` does not enforce all hold a second time under a
second task. And a task whose answer the orientation covers costs a fifth of one whose answer it does
not — $0.73 against $3.65, twenty calls against fifty-nine.

**Does not establish.** `envelope-exceeded` has still never been met by an agent, and after this run
it is fair to say it never will be by design. Nothing was run inside the seal, so §6, §7 and the
Store are untouched by this transcript. The seal was not tested, the session not having gone looking.
No Repeatability value was used, no `destroy` was authored, and no Cadence was written, so the three
things ADR-0112 leaves unmeasured — `over: {values:}`, the `run-once` value, an executed Cadence —
are unmeasured still.

## What was considered

**Reading `[control, local]` as evidence about the agent rather than about the documentation.** The
harness cannot separate *read the clause* from *guessed right* by the artefact alone, which is why
the closing report matters: the session states the rule in its own words and cites the surface it
read it from. That is as close to the distinction as a transcript gets, and it is close enough to
report.

**Re-running with the clause removed, to isolate it.** Refused, on ADR-0118's own ground: a surface
degraded to preserve a measurement is the measurement eating the product. The counterfactual that was
run instead — the session's artefacts with two lines edited — costs nothing and answers the narrower
question a reader actually has, which is whether the trap was live.

**Ticketing `check`'s acceptance of `bound: 1` on an opaque `mutate` as a `check` defect.** Refused.
The value is truthful about Records, which is what a Bound counts; what issue #241 is about is the
flag it clears and the gutter mark it removes.

**Editing `change-window` now that its measurement has been taken.** Refused, and out of scope by
issue #240's own text. It measured what ADR-0118 said it would, including the half about the second
Target that the run then falsified — which is a note for the next task's setup script, not an edit
to this one.

## Consequences

- **Issue #238's first criterion is closed against this ADR.** Its two alternatives were *a
  transcript in which an agent meets `envelope-exceeded`* or *the decision that no task can reach
  it*. ADR-0118 recorded the second and this run supplies what that decision needs: the rule is
  reachable, the fixture removed the design-around, and the agent went past it because the
  orientation says so.
- **Issue #221's sixth user story has its answer**, against `change-window`, and it is the positive
  one: the envelope meets an agent before it meets a user, as a sentence.
- **ADR-0117's clause has direct positive evidence**, which is the first time a sentence added to the
  orientation has been measured rather than argued for.
- **ADR-0116 has its first transcript and its first end-to-end halt.** The Requirement is authorable,
  is preferred over the workaround for the reason the orientation gives, renders `r`, and stops a
  caller's Steps from inside an invoked Procedure.
- **`change-window` stands as authored**, and its predictions have been checked one by one against a
  run: the three wrong answers were all avoided, the Requirement was asked for and delivered, and the
  one that did not hold is the route by which the second Target gets found.
- **Two defects ticketed**: issue #241 and issue #242.
- **A run whose answer the orientation already contains is cheap**, and that is worth saying beside
  ADR-0111's price for one whose answer it does not. Roughly twenty-seven of that run's fifty-nine
  calls went on interrogating `check` for a grammar no document states; this one spent none.
