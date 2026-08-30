# Composition is not conditionable, and the shared check had to claim `mutate` to halt

**An agent inside the seal composed three Procedures, met eight kinds of Refusal, and got there — but
the one thing the task asked for that `hyper` will not do is make a shared check gate its callers.**
A `when:` may root only at an earlier *sibling*, and a nested invocation is not one, so the session
turned the verdict into a halt with an effectful Step: the read-only archive check now claims `mutate`
on `local` in order to be able to fail. It said so to the reviewer in as many words. That is issue
#236, and the second of issue #232's three runs.

**The trap the task was built around never fired.** `release-promotion` was written so that the
natural first draft declares `targets: [local]` on both routes and meets `envelope-exceeded` at the
invocation (issue #225). This session's first draft declared `targets: [archive, local]` and the
Refusal was never reachable, for a reason that is a fact about the task rather than about the agent.
The envelope rule is still unmeasured by any transcript — issue #238.

## The evidence: fifty-nine calls, eighteen of them `check`, and twelve of those refused

A Claude Code session, headless, 2026-08-30, inside the seal
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)), against
`release-promotion` (issue #225). Fifty-nine tool calls, sixty turns, thirteen minutes thirty-five,
$3.65, exit `0` — three times the cost of the `fleet-rollout` run
([ADR-0110](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md))
and about the same as `monitor-coverage`'s (ADR-0106) for half again as many calls.

Twenty-eight `Bash`, **eighteen `check`**, five `review`, five orienting MCP calls, one `ToolSearch`,
two `Write`. Twelve of the eighteen `check` calls came back with problems, against ADR-0106's eight
clean ones out of eight. The two runs are the two faces of the same oracle.

Refusals met, by code: `unknown-key` (32 rows), `schema-mismatch` (5), `reference-unresolvable` (4),
`capability-reserved` (2), `artefact-absent` (1). **No `envelope-exceeded`, and nothing was ever run** —
the task ends at the diff and the session ended there.

**It foraged, and the seal held.** Call 17 was one compound `Bash` — `which hyper`, `ls ~/.hyper`,
then `find / -maxdepth 6 -iname "*hyper*"` with the repository itself excluded — run immediately after
a Refusal it could not read its way past. `which` and `ls` answered nothing, and the `find` came back
with `ax_hypertext.py`, `xen-hypercalls.sh` and the kernel's `hyperv` directories. No checkout, no
specification, no second binary. That is ADR-0109's seal answering the exact question it was built
for, in a run that went looking on purpose.

## The oracle is good enough to reverse-engineer the language from, and this is what that costs

The session did not know the Procedure grammar and could not read it. What it did instead, for
roughly twenty-seven of its fifty-nine calls, was interrogate `check`: write an artefact carrying a
guess, read the Refusal, write the next one.

- **Call 13 broke a name on purpose** — `procedure: verify-nonexistent` — to find out whether the
  invocation position was checked at all. `artefact-absent`, with the path it looked for. That is a
  test, and the session wrote it as one.
- **Call 18 wrote fourteen keys it had invented** — `assert`, `expect`, `gate`, `guard`, `halt-on`,
  `on-failure`, `condition`, `when`, `unless`, `precondition`, `must`, `fail-on`, `continue-on-error`,
  `strict` — on one invocation, and read back fourteen `unknown-key` rows. A closed key set enumerated
  in one round trip.
- **Call 34 did the same to the predicate operators**, and `check` answered with the whole vocabulary
  in the message: *exactly one of `equals`, `not_equals`, `in`, `exists`, `absent`, `starts_with`,
  `ends_with`, `greater_than`, `less_than`, `older_than`, `newer_than` must be present*, and beside it
  *"a predicate list is always AND, and there is no disjunction key"*.
- **Call 38 found the reference form** by writing a bare string where a reference goes:
  *"a reference is a mapping and never a string — `{step:, path:}` and `{item:}` are the only two
  forms"*. Two calls later that form was carrying the release name and the payload bytes between Steps.

**The messages are why this worked.** Every one of those Refusals named the position, the code, and
the closed set the author had missed — `check`'s messages are doing the work a reference manual would,
and ADR-0099's finding that an agent can author blind against this oracle holds a third time under
much harder conditions. What the run adds to that finding is its price: **an agent that has to
discover the grammar spends half its calls on it**, and the two transcripts that did not
(`fleet-rollout`, `monitor-coverage`) are the ones whose artefacts happened to fit what the
orientation shows.

## A shared check cannot gate its callers

The task is explicit about the shape it wants: neither route touches `live/` before the archive has
been checked, the check is four facts today and will grow, and *when it grows I want to edit it in one
place*. One copy, called by both.

The session authored exactly that and then tried to gate on it — which is what §6 says to do:

> An API answering `404` for *absent* is describable for the same reason: the status is recorded and a
> later Step's `when:` decides on it (§3), which puts *what counts as acceptable* on a line the gutter
> annotates rather than in the artefact a reviewer reads least.

All three spellings were refused:

```
reference-unresolvable  step: archive-intact names no id: this Procedure declares earlier
reference-unresolvable  step: verify.archive-intact names no id: this Procedure declares earlier
reference-unresolvable  step: verify names no id: this Procedure declares earlier
```

**And every one of those Refusals is right.** `checkSteps` threads its index forward so a reference
resolves only against an `id:` written earlier in the same list, *"never across a nested invocation's
own boundary, Procedures composing by invoking one another rather than by sharing a Step namespace
(ADR-0002)"*. The third spelling — naming the invocation itself — is refused on a different and equally
sound ground: an invocation writes no Journal file and projects no Record, so there is no `field:` for
a predicate to root at.

So the two things the task asks for are, today, mutually exclusive: **a check may be shared, or it may
gate, and not both.** What the session did with that is the run's best moment. It moved the halt
*inside* the shared check:

```yaml
  # The verdict, as a halt. Runs only when the check above failed.
  - id: halt-unless-sound
    definition: live-ops
    operation: mutate
    target: local
    when: {step: archive-sound, field: exit_code, not_equals: 0}
    bound: 1
    args:
      command: [sh, -c, 'echo "archive verification failed - live/ not touched" >&2; exit 1']
```

A `read` never halts on what came back (§6), so a verdict is not a halt. An effectful Operation halts
on anything but success. Therefore the only way to stop a Run on a condition is a Step that claims
effectful authority — and a check whose whole point is that it writes nothing now claims `mutate` on
the machine it is protecting. The session named the cost to the reviewer itself:

> Worth a reviewer's eye: the shared check is *not* read-only in authority terms. Its last step claims
> `mutate` on `local`, because that is the only way it can halt.

**That inverts the Kind axis on the one artefact where it matters most.** A reviewer reading
`verify-archive` sees `m` in the authority table and an `OPAQUE` flag on a Step that writes nothing at
all. It is issue #236.

**A second consequence of the same boundary**: an invocation admits `id:` and `procedure:` and nothing
else — no `args:`, no `over:`, no `when:`, no `bound:` — so a shared Procedure is a fixed,
unparameterised block. The session found this by probing and redesigned around it: the check derives
*both* release candidates itself and validates each, rather than being handed the one the caller cares
about. It flagged the consequence honestly — *"This is stricter than you asked and you should decide if
you want it"* — because a promotion will now refuse if `live/previous` names a release the checksum
list does not cover, even though promotion never uses it.

## The orientation does not know that Procedures compose

Nothing in `AGENTS.md` says a Procedure may invoke another. Its worked Procedure is a flat list of
Steps, and `targets:` sits one line above that list. So three facts the session needed were reachable
only through `check`, or not at all:

1. **That composition exists.** The session guessed it, wrote `procedure: verify-archive` into a Step
   at call 10, and confirmed the position was real by breaking the name at call 13.
2. **What an invocation admits.** `id:` and `procedure:`, closed. §3 says an invocation *"names an
   `id:` and a `procedure:` in place of `definition:`/`operation:`/`target:`"*, which reads as though
   `args:`, `over:`, `when:` and `bound:` still apply; `invocationDeclaration` in
   `internal/artefact/procedure.go` admits two keys and nothing else. The implementation is stricter
   than the specification's plain reading, and neither says so where an author looks.
3. **That a halt inside a nested Procedure halts the whole Run.** §6 states it in one line — *"a halt
   inside a nested Procedure is a halt of the whole. One Run has one outcome, one Journal entry, and
   one exit code"* — and the orientation does not. The session's entire safety story rests on it, and
   it handed the reviewer an unverified assumption rather than a claim:

   > **The load-bearing assumption**: that an effectful step failing *inside* `verify-archive` halts
   > the whole Run, so `promote`'s later steps never execute. […] I could not confirm it propagates
   > out of a sub-procedure, and it is the property the whole safety story rests on.

It is right, and it could not find that out. That is issue #237.

## The trap did not fire, and the reason is a fact about the task

`release-promotion` was built so that `envelope-exceeded` is reached by writing the obvious thing.
Issue #225's reasoning, in the setup script:

> The shared check reads the archive and reaches only the read-only name; the two routes write `live/`
> and reach only `local`. A route's own Steps therefore all bind `local`, and `targets: [local]` is
> what an author writes with the file in front of them — at which point the invocation carries
> `archive` in past the declaration.

**The premise is that a route's own Steps all bind `local`, and this session's do not.** Its `promote`
reads `archive/wanted` and then the payload bytes *through the archive Target*, so that the Step which
writes `live/` never opens a path under `archive/` — it is handed the bytes as a value. That design
puts `archive` in `promote`'s own envelope before composition is considered at all, and
`targets: [archive, local]` was in its first draft.

The setup script enumerated four answers a run could give. The session gave a fifth that is not on the
list and is not wrong: it declared both Targets on the shared check *as well*, which the script
predicted would be "merely wrong — it says that Procedure may write `live/` when it never does". Here
it does. The check holds a `mutate` Step, so the declaration is accurate.

So: **no transcript has yet met `envelope-exceeded`**, §5's transitive containment rule has never been
in front of an agent, and issue #221's user story about the envelope is still open. Recording that is
issue #232's fifth acceptance criterion — *a task that measured nothing is a fact about the task* — and
the ticket that keeps it from being mistaken for measured is issue #238.

## What the run establishes, and what it does not

**Establishes.** Composition is authorable from the shipped surface by an agent that has never been
told it exists. Three Procedures, two Definitions, `check` clean over eleven artefacts, five `review`
calls read correctly, and a diff handed back with nothing run — which is where the orientation's loop
ends and where this session stopped, unprompted, including declining to `store init`: *"There is no
Store in this repository, so a Run has nowhere to record. That one is yours to run."* The read-only
Target held: `archive-audit` × `archive` renders effective `r`, the session added no Target and
changed neither existing one, and it verified that off `review` rather than off the disk, as the task
demanded.

**It found the wall the task states and `hyper` does not enforce, unprompted.** The task's first
paragraph is an operator's standing rule, not a guarantee — `local` grants `read mutate destroy` over
the same machine, so a route bound to `local` could write into `archive/` with nothing Refusing. The
session said so:

> `shell` is opaque, so `mutate` on `local` is *unconfined* — hyper flags every one of these steps
> `OPAQUE` and cannot tell you which paths a command opens. That […] nothing reaches into `archive/`
> except through the archive target, is true of the command text a reviewer can read […] — it is not
> something hyper's authority model enforces.

That is the honest limit, stated in the right place, to the right audience. It is the strongest single
piece of evidence in the run that the review surface teaches what it is supposed to.

**Does not establish.** Nothing was run, so §6, §7 and the Store are untouched by this transcript —
`fleet-rollout`'s is where those live. `envelope-exceeded` was not met. No Repeatability value was
used: every Operation here is `repeatable`. And the design's load-bearing property — halt propagation
out of a nested Procedure — is asserted by §6 and unverified by any transcript, this one included.

## What was considered

**Changing `release-promotion` so the envelope trap fires.** Refused here, and not because it is a bad
idea — because issue #232 puts it out of scope for a reason that survives this run: the tasks are
artefacts that were reviewed and fenced, and a task that turns out to be wrong is a new ticket with the
transcript as its argument. That ticket is issue #238, and what it has to decide is harder than it
looks: forbidding the route from reading the archive directly would forbid the design that produced the
best artefact any of these runs has authored.

**Ticketing `check`'s acceptance of both `not_equals: 0` and `not_equals: "0"`.** Refused — it is
§12's stated limit, not a defect. See below.

**Reading the session's `mutate`-to-halt as a misuse to be closed off.** Refused. Given the boundary,
it is the *correct* construction: it is the only thing that stops the Run, it is guarded so it fires
only on failure, it is bounded, and it was disclosed to the reviewer. What issue #236 argues about is
the boundary, not the workaround.

## Consequences

- **Issue #221's user story on composed Procedures is answered; the one on the envelope is not.** An
  agent composes correctly, finds the closed key set, and works around the gating boundary. Whether it
  meets `envelope-exceeded` remains unknown after three runs of the harness.
- **`release-promotion` stands as authored**, and issue #238 carries the argument for a task that can
  reach the Refusal.
- **Three defects, ticketed with this transcript as their evidence**: issue #236 (a shared check cannot
  gate its callers, so it must claim effectful authority to halt), issue #237 (the orientation does not
  mention composition, and an invocation's key set is closed and undocumented), issue #238 (the
  envelope rule is unmeasured and no current task reaches it).
- **ADR-0099's oracle finding holds a third time, with its price attached.** An agent can author blind
  against `check`. What this run adds is that discovering an undocumented corner of the grammar costs
  about half the session, which is the argument issue #237 is made of.
- **ADR-0109's seal has direct evidence.** A `find` over `/` from inside the namespace, run by a
  session that wanted the specification, returned nothing of this project.

## Found in the same run, and not this decision

**`check` accepts both `not_equals: 0` and `not_equals: "0"`, and that is §12's limit rather than a
defect.** The session tried both, saw both pass clean, and flagged it as an open question for the
reviewer — *"If hyper compares it as a string, the gate trips on every run and both procedures refuse
to touch `live/` — visibly, and in the safe direction."* §12 settles it: *a projected field has no
declared type (§3), so a predicate can be handed a value it cannot compare. It Refuses; it never treats
the value as not matching.* The fault is `predicate-type-mismatch` at Expansion, not at `check`,
because there is nothing offline to compare the literal against. The session identified a real property
of the tool, reasoned correctly about the direction it fails in, and escalated it to the human. That is
the behaviour the review loop is for, and it needs no ticket.

**`capability-reserved` fired for the second time in the harness's history.** The session's first
instinct for the shared check was a Provider — a `providers/release-audit.yaml` declaring
`capabilities: [shell]` — and it met the same wall issue #218 and ADR-0099 recorded: *`shell` is
reserved to the Providers `hyper` ships — an Extension may never hold it.* It read the message, deleted
`providers/`, and made the check a Procedure instead, which is the right answer. The message did its
whole job in one call.

**The sealed session wrote to the auto-memory directory, and the seal contained it.** Calls 57–59 read
and wrote `~/.claude/projects/<slug>/memory/`, saving what it had learned about the Procedure grammar
for a next session. The seal binds `$outdir/projects` over `~/.claude/projects`, so the file landed
inside this run's own output directory and is covered from every subsequent run by ADR-0109's search.
No cross-run contamination is possible while each run is given its own output directory — which is
worth writing down, because **reusing one would carry a previous session's notes into the next**, and
nothing in `run.sh` clears `projects/`.

**Three of fifty-nine calls went on that memory write.** Like the `ToolSearch` tax ADR-0110 records, it
is a fact about the client rather than about `hyper`, and it is noted so that a call count read off a
transcript is read with it in mind.
