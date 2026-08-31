# The envelope cannot be trapped now that the orientation states it

**Issue #238 asked for an acceptance task whose `envelope-exceeded` trap cannot be designed around.
`change-window` is that task, and building it turned up the reason the ticket cannot get what it
asked for: the orientation now states the rule.** Between issue #238's filing and this, two tickets
put it there — issue #236's shared-check section and issue #237's closure of the invocation's key
set — and the sentence they added ends *and its `targets:` count against the caller's declared
envelope* (`internal/mcp/instructions.go`, commits 22bd8a8 and aa550b5). Issue #238 could not have
known: its own text argues about issue #236 as an open design question.

A trap needs an author who does not know the rule. There is no such author now, and no task can make
one. So what lands is the task, and a changed claim about what it is for: **`change-window` measures
whether that sentence was enough.**

`release-promotion` is not edited. That was issue #238's second answer, and it is the one issue #232
put out of scope and issue #238's fourth criterion forbids, on ADR-0106's precedent.

## The confound a task can remove

`release-promotion` was written so the obvious draft declares `targets: [local]` on both routes and
meets `envelope-exceeded` at the invocation (issue #225). Its premise was that a route's own Steps all
bind `local`. The 2026-08-30 session's did not: its `promote` read the payload bytes *through* the
archive Target and handed them between Steps, which put `archive` in `promote`'s own envelope before
composition was considered at all
([ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)).
The premise was the defect — the thing the shared Procedure reached was also the thing the route's work
was made of, so the route could always invent a reason to name it.

`change-window` gives the shared check a Target the callers want nothing out of. Two routes edit a
firewall allow-list on the machine — one grants what has been requested, one revokes what has been
withdrawn — and both must first pass one change-control check that reads, through a second read-only
name, files kept by somebody else. What that check produces is not a value the routes consume. It is a
halt.

That holds because of two rules already in the model:

- **No value crosses an invocation's boundary in either direction** (ADR-0002). A route cannot read
  what the check read, and it cannot gate on the invocation — `require: {step: <the invocation>, …}` is
  `reference-unresolvable`, and the message names the way across: *a `require:` entry inside it halts
  the whole Run, and its later Steps and its caller's alike never run*
  ([ADR-0116](0116-a-requirement-halts-and-claims-nothing-to-do-it.md)).
- **A Requirement claims nothing** (§5, ADR-0116). The check is `read` Steps and Requirements, so it is
  read-only in authority terms and still stops everything downstream of it. `review` renders it
  effective `r`.

**So there is no correct design in which a route's own Steps bind `control`**, and that half of issue
#238 is answered. The only way to get that name into a route's file is to copy the check's Steps into
it, which gives up the one copy the task asks for in its own words.

## The confound a task cannot remove

The orientation's own section on the shared check now reads:

> **No `when:` and no reference reaches across that boundary**: what an invoked Procedure did is not a
> fact its caller can condition on, and its `targets:` count against the caller's declared envelope.

That is one clause in a section about how a check halts, and an agent reading for the halt may not
carry it to the `targets:` line. But it is *there*, and a task built on the assumption that it is not is
`release-promotion`'s mistake in a second form: **a premise about what the agent cannot know, held
against a surface that has since been taught it.** The setup script does not make that assumption, and
neither does this.

**Issue #238's third answer is therefore partly right, and for a reason it did not have.** It offered
*accept it as unmeasurable this way, and say so* on the ground that the rule may not be reachable by a
task an agent designs well. What is true is narrower and firmer: `envelope-exceeded` is not reachable
*by design* at all. A first-draft Refusal is now a fact about whether an agent applied a rule its
orientation states, which is not something a fixture can arrange.

**Removing the clause to keep the trap was considered and is not available.** It is issue #237's fix,
ADR-0117 prices it, and a surface degraded to preserve a measurement is the measurement eating the
product.

## What a run of it now answers

Both outcomes are worth the session, which is what makes the task still worth landing:

- **The routes declare `targets: [local, control]` first time.** The orientation taught the rule, which
  is a direct positive result for ADR-0117's clause and the best available answer to issue #221's sixth
  user story — *§5's envelope rule meets an agent before a user meets it*. It meets the agent as a
  sentence rather than as a Refusal, and `envelope-exceeded` stays unmet. That is then a fact about the
  documentation rather than about the task, and this ADR is where a reader learns it without re-running.
- **They declare `targets: [local]`.** The Refusal is in front of an agent for the first time, and
  because the route has no reason of its own to name `control`, nothing about the design can explain it
  away. Two rows, one per route, and the repair is one edit per file.

**And a Requirement is measured either way.** Issue #236 landed the shape and ADR-0116 records it; no
transcript has met it. §5 states it with this exact artefact — *a Procedure whose Steps are all `read`
and whose last entry is a Requirement is read-only in authority terms and is still able to stop
everything downstream of it* — and `change-window` asks for that Procedure without naming it. The
orientation now works an example of it, so this measures recall rather than invention; that is still
the first transcript either way.

**Three answers pass `check` and are not the repair**, each authored by hand and each clean: copying
the check's Steps into both routes, which gives up the one copy; rebinding its reads to `local`, which
gives up the wall the task states and which `check` cannot refuse, `local` granting `read` over the
same machine; and dropping the halt, which gives up *the job stops and `firewall/allow` is left exactly
as it was* and which `review` will not flag. Each is a clean repository, and the only place any of them
shows is the transcript — which is why the task closes by asking what this repository now says each
route may touch and where it says it.

## What was considered

**Landing nothing and recording the envelope as unmeasurable.** Refused. The reasoning above is worth
recording and the task is worth having anyway: it removes the design-around, it is the only fixture in
which a Requirement is asked for, and it is the only way to find out whether issue #237's sentence
lands.

**Making the second Target one the routes are forbidden to read** — a task that says *do not read
`control/` directly* and traps an agent that obeys. Refused for ADR-0106's reason: a trap that punishes
the right answer measures the task. `change-window` does not forbid the route from reading those
files; it gives the route nothing to read them for.

**Asking for the second Target rather than shipping it.** Refused on issue #225's own ground, which
this task inherits: it is a fact about the repository an operator hands over, and an agent that
declined to create it would take the composition with it. It still has to be found, through
`hyper targets`.

## Consequences

- **Issue #238's first criterion is not closed here, and cannot be closed by a task.** No transcript
  exists, and the decision recorded is neither of the two the criterion names: it is that the rule is
  no longer trappable, that the task lands anyway, and that a sealed run answers a different question
  than the one issue #238 asked. The run is issue #240, on issue #221's tenth story — a sealed
  run's output is not the running agent's to interpret.
- **Issue #221's sixth user story is marked on the issue**, against `change-window` rather than
  `release-promotion`, with what it can and cannot now get. Its closing table also names the wrong task:
  the composition task issue #225 produced is `release-promotion`, and `tenant-onboarding` is issue
  #224's second-Run task. Issue #232's table repeats the swap. Both are corrected on the issues.
- **The fence covers the task because it exists.** `TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse`
  ranges over `scripts/acceptance/tasks/` (issue #222), so `change-window` is asserted — the setup ran,
  `providers/` is absent, the repository checks clean, `AGENTS.md` is this binary's orientation — with
  no edit to the seam. That is issue #238's third criterion.
- **`release-promotion` stands as authored**, which is issue #238's fourth criterion, met by not
  touching it.
- **A task's premise about what an agent cannot know has a shelf life.** `release-promotion`'s expired
  against a design, and this one expired against a documentation fix in the eight days between the
  ticket and the work. A task written around an absence should say which absence, so the next person
  can check whether it is still absent — which is what `change-window.setup.sh` now does in as many
  words.
- **Nothing in `hyper` changed.** The rule was correct and the messages were correct; what moved was
  the orientation, in someone else's ticket, and this is a harness decision that costs the surface
  nothing.

## Completed by hand before it landed

The draft above; both `envelope-exceeded` rows and nothing else; `targets: [local, control]` on each
route; `check` clean over eleven artefacts; `review grant-pending` rendering `envelope ✓` with
`control` in the gutter against the invocation and `ENVELOPE line 3 ok` under it; the same file at
`targets: [local]` rendering `envelope ✗` and `ENVELOPE line 3 exceeded`; and `review
change-permitted` rendering the check effective `r` against a Target granting `read`. The three answers
that are not the repair were each authored and each passed `check`. The seven commands the check and
the two routes are written out of were run against the fixture and all seven hold.
