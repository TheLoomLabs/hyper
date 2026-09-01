# The second `change-window` run declined the bound, and a taught repair owes a run

**One task was re-run — `change-window`, on 2026-09-01 — and the three repairs its first run caused
were all measured landing. `snapshot-lifecycle` and `monitor-coverage` were not bought, and the
reasons are below. The rule that follows is in
[`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md): a repair that is *taught*
rather than *enforced* is fenced by nothing in the suite, so the ticket that lands one names the run
it owes and either buys it or writes down why not.**

## What was wrong (issue #250)

Six tasks, eight sealed runs, and every one of them predated the repairs it caused. Issues #217
through #242 changed the orientation, `check`'s rows, `review`'s renderings and three sections of the
specification — each change made *because* of a transcript, then verified by the suite and by nothing
else.

That gap is structural rather than accidental.
[`cmd/hyper/acceptance_test.go`](../../cmd/hyper/acceptance_test.go) asserts properties of the
**harness**, and issue #221 fixes that boundary deliberately: *nothing in the suite may assert what an
agent did*. So a clause added to the orientation is verified as text that changed. Whether it
**teaches** is a question only a run answers.

## The decision: one run, and why not three

`change-window` was bought. It is the most recent task, its repairs (#241, #242) are the least
settled, and it is the transcript in which an agent was talked out of a flag — the failure mode with
no other instrument. It is also the cheapest real answer available: hermetic, no Store, nothing run,
ending at the diff. **It cost $0.69, 20 turns and 128 seconds.**

`snapshot-lifecycle` was **not** bought. It is the baseline the harness is measured against and its
baseline is now eighteen ADRs stale, which is a real debt — but it is a debt about *comparability*,
and nothing landed since its last run is a repair to what it exercises. It is the first candidate
when the next taught repair lands.

`monitor-coverage` was **not** bought. It is the only Provider-authoring task and the deepest part of
§3, and none of #236–#242 touched Provider authoring. Buying it here would have spent the instrument
confirming work rather than finding the next thing, which is the trap issue #250 names in its own
words.

## The evidence: three repairs, three before-and-afters

One task, run twice eight days apart against the same fixture, is the cleanest comparison this
harness has produced. In all three the earlier run is 2026-08-31 (ADR-0120, ADR-0121, ADR-0122) and
the later is 2026-09-01.

**The envelope clause holds, twice** (issue #238,
[ADR-0118](0118-the-envelope-cannot-be-trapped-now-that-the-orientation-states-it.md)). Both routes
were authored `targets: [control, local]` in the first draft, `check` was clean over eleven artefacts
on its first call, and `review` renders `envelope ✓`. No `envelope-exceeded` was met at any point.
The fixture's own setup script predicted this: the clause landed before the task did, so *this is not
a trap* — and a second consecutive first-draft pass is the confirmation ADR-0118 could not give
itself.

**The flag was declined, and that is the result of the run** (issue #241,
[ADR-0121](0121-an-opaque-mutate-is-unbounded-and-no-number-clears-the-flag.md)). The 2026-08-31 run
wrote `bound: 1` onto both opaque `mutate` Steps — the one edit that clears the flag without touching
what the flag was about. **The 2026-09-01 run authored no `bound:` anywhere**, and said why, unasked:

> `UNBOUNDED` is not a missing `bound:`; on an opaque `mutate` a bound counts Records rather than
> commands, so writing one would buy nothing and the flag stands either way.

That is ADR-0121's clause read back almost verbatim, from an agent that met it cold. The failure mode
#241 was filed for did not recur.

**The Requirement's value went on the line** (issue #242,
[ADR-0122](0122-a-requirement-roots-at-any-projected-field-and-the-value-goes-on-the-line.md)). The
earlier run transcribed the worked example three times — three `require:` lines on `exit_code`, each
predicate hidden inside a quoted `sh -c` argument. The later run wrote two of the three as value
comparisons on the `require:` line itself:

```yaml
  - id: window-open
    require: {step: read-window, field: stdout, equals: "open\n"}

  - id: freeze-clear
    require: {step: read-freeze, field: stdout, equals: ""}
```

The third is the interesting one. *An approver is named* is not any one value, so the run kept
`exit_code` — and, unprompted, wrote the command as a bare array `[test, -s, control/approver]`
rather than an `sh -c` string, so the predicate still reads in `review`, and then flagged in its
report that this is the one condition where a reviewer must read the `args:` line too. **The clause
was applied with judgement rather than transcribed**, which is more than ADR-0122 asked for.

## What the run did not find, and what that costs

**No Refusal was met, and no `check` error.** Nineteen tool calls, one `check`, five `review`s, and a
diff. As a measurement of the repairs that is the best available answer; as an instrument,
`change-window` has now told us what it can. Its trap is gone by construction — the orientation
teaches the envelope, so the gap the fixture was built around cannot open — and a third run of it
would be buying a regression check, not a finding.

**One run is one sample.** Nothing here says the clauses hold for every agent, only that they held
for this one, cold, on the task they were written for.

The run's own honesty is worth recording as well: it flagged that its `revoke-withdrawn` can leave a
`firewall/allow.next` behind where `grep` fails outright. That is a property of the artefact it
wrote, not of `hyper`, and it is exactly the kind of thing the closing question exists to surface.

## Consequences

- **Three repairs have their first evidence.** #238, #241 and #242 were verified by the suite as text
  and are now verified as behaviour, once each.
- **A taught repair owes a run, and the obligation is written down.**
  `docs/agents/acceptance-re-runs.md` states the taught/enforced test, which task the run falls to,
  and that deferring is a complete answer when it is recorded. `AGENTS.md` and `CONTRIBUTING.md` point
  at it.
- **`snapshot-lifecycle` is the next purchase**, and it is owed for comparability rather than for a
  specific repair — which is a weaker claim on the budget than any taught repair would be, and should
  lose to one.
- **`change-window` is spent as an instrument** until something changes what it exercises. That is not
  a reason to remove it: the fence ranges over `tasks/*.md`, and its setup half runs on every
  `go test ./cmd/hyper`.
