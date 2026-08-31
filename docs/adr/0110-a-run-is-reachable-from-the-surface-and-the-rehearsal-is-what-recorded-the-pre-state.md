# A Run is reachable from the surface, and the rehearsal is what recorded the pre-state

**An agent inside the seal authored a Procedure, rehearsed it, ran it for real, and read the account
back — the first acceptance transcript ever to reach §6 Execution, §7 the Record, and the Store.**
Twenty-seven tool calls, no Refusal, no failing `check`, and both of the task's closing questions
answered off the Records rather than off the disk. That is the measurement issue #223 bought when
[ADR-0104](0104-the-acceptance-fixture-ships-a-store.md) put a Store in the fixture, and issue #232's
first of three.

**Nothing about the product changes on that account.** What the run changes is three things that are
not the product: a wrong sentence in the agent's own answer about where the Record lives (issue #233),
and two surfaces that go quiet around a rehearsal (issues #234, #235).

## The evidence: twenty-seven calls, two Runs, and not one Refusal

A Claude Code session, headless, 2026-08-30, inside the seal ([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)),
against `fleet-rollout` (issue #223). Twenty-seven tool calls — eighteen on the MCP surface, seven
`Bash`, two loading tool schemas — twenty-eight turns, three minutes fifty-four, $1.19, exit `0`.

**Zero Refusals and zero failing checks.** `check` ran twice and was clean both times. Nothing in the
run met a guardrail, a schema error, or an authority wall. That is the sharpest number here and it is
a different number from ADR-0106's eight clean checks: there the offline oracle stayed silent because
the session got a hard artefact right first time; here it stayed silent because the artefact was
small. What the two share is that neither run was ever *taught* by a Refusal, so this transcript says
nothing about what an agent does with one.

What it wrote, before any `check`:

- `definitions/fleet-version-writer.yaml` — a **second** Definition, `kinds: [mutate]`, on the stated
  ground that "`host-ops` is `kinds: [read]` and a Definition observes or effects, never both". The
  two-key rule held again, unprompted, on a third transcript.
- `procedures/roll-fleet-to-wanted.yaml` — four `read` Steps, three `cp` `mutate` Steps each carrying
  `bound: 1`, and a closing `read` over all four files.

**The ordering was the agent's own and it is the whole reason the task has an answer.** The four
observe Steps sit *before* the three mutates in one Procedure, and the session said why in the
artefact's own comments: once `cp` has run, `fleet/` reads alike and those Observations are the
repository's only account of the versions they replaced. Nothing in the task, the orientation or the
fixture names that ordering. It is the thing the task exists to find out whether an agent reaches for,
and this one reached for it in its first draft.

**It copied the file rather than writing the version.** `command: [cp, fleet/wanted, …]` rather than a
`1.4.2` literal, so the wanted version stays stated in one place and convergence is byte-exact. The
same instinct the task's own author had.

**`probe` was declined for the right reason.** The session reached for a Probe to confirm the working
directory a Step runs in, then declined it: *every `shell` Operation is opaque, and a Probe may
invoke neither an effectful nor an opaque Operation*. It substituted `run --dry-run`, which is the
route that is actually left. Unlike ADR-0106's decline, this one was not weakened by the Target — a
Probe here was genuinely available to attempt and genuinely not allowed.

## Both questions were answered, and `changes` is what answered them

The task asks two things and both are answerable only off the Records: *which machines were behind and
what each was on*, and *what the account amounts to — which of it is something you did rather than
something you only looked at*.

The session answered the first from `changes`, correctly: web-02 on `1.3.9`, db-01 on `1.2.0`, web-01
already converged. It answered the second as the Asset/Observation split, which is the rendering
issue #223 wrote the task around — three `asset created` rows against the `cp` Steps and five
`observation appeared` rows against the `cat` Steps, each carrying its projected `stdout`. It named
the Records it caused to exist separately from the artefacts it authored, and listed what it had only
read.

**`records` could not have answered the first question and the session found that out by trying.**
`records` writes identity, ordinal, Run, Step, Observation-or-Asset, Tombstone, secret markers and
Provenance — and no field values, which is §9's contract and not a fault: it is the surface whose job
is finding a version rather than reading a change. `run show` carries none either. The session called
both, said so in as many words — *`records` and `show` give identity and disposition but not field
values* — and moved to `changes`. Three calls to learn a division of labour that holds correctly.

## The rehearsal is what recorded the pre-state

The session rehearsed before it ran, and the Store says something surprising afterwards. **All four**
pre-state Observations exist **only** under the rehearsal's `run_id`, and the effecting Run's one
Observation is the closing read taken after `fleet/` had already converged:

```
records/local/host-ops/["cat","fleet/wanted"]/01a0517c-…-0001.json              ← the dry run
records/local/host-ops/["cat","fleet/web-01/installed"]/01a0517c-…-0002.json    ← the dry run
records/local/host-ops/["cat","fleet/web-02/installed"]/01a0517c-…-0003.json    ← the dry run
records/local/host-ops/["cat","fleet/db-01/installed"]/01a0517c-…-0004.json     ← the dry run
records/local/host-ops/["cat","fleet/wanted",…,"fleet/db-01/installed"]/01a0517d-…-0008.json
```

(Identities are shown decoded; the Store percent-encodes them in the path.)

The effecting Run's Steps 1–4 are `ran` in its Journal entry and minted no version at all, the value
being unchanged. So the repository's account of *what each machine was on* — the task's first question
— is attributed to a Run that §7 tells every consumer of Journal evidence to filter out.

**Every part of that is the specification working as written.** §6: *a dry-run performs the reads it
reaches … those reads really happened, so they record Observations like any other.* §7 filters the
rehearsal's **entry**, and the entry is not the Record. And the answer still renders, because the
rehearsal is not a Comparison subject either: `changes` takes the effecting Run as subject, finds no
non-dry baseline behind it, and shows all five Observations as `appeared` with their values intact.
The design holds end to end.

**What it costs is that the two halves of one fact live under different `run_id`s**, and the session
paid for that twice.

**Once at `records`, which names the Run and not what kind of Run it was.** §7 makes `dry_run`
mandatory on every entry, `false` included, *and is the one marker in the Store that does not follow
the absence rule* — because a reader taking absence for `false` gets a permanent wrong answer. The
Record version carries no such field, and `records` renders no such column, so getting from a row to
*that was a rehearsal* is a `run show` join. The session made that call. It is issue #234. _ADR-0114
decides it:_ the marker stays on the entry, and `records` makes that join inside the one call and
renders it on the row.

**Once at the gap between the two Runs, where nothing renders what the rehearsal read.** `changes`
after the rehearsal alone came back empty — correctly, there being no non-rehearsal Run to be a
subject — and the session drew the conclusion out loud: *the pre-state field values aren't legible
until a second Run exists.* For the length of that gap, four Observations sat in the Store with no
surface on the product that would show their values. An operator who rehearses a rollout precisely to
see what the fleet is on before committing to it is in that gap, and this task is that operator. It is
issue #235, and it sharpens issue #230 rather than duplicating it: #230 is a *response* nothing shows,
this is a *Record* nothing shows.

## The account is in the repository, and the agent said it was not

Asked what the repository's account now amounts to, the session went looking for the Store, ran `ls
-a`, `git status --ignored`, and probes for `.hyper`, `.hyper/store`, `store` and `records`, found
nothing, and answered:

> There is no store directory anywhere under the working tree … right now the repository can say all
> of this, but only on this machine; a clone would get the Procedure and not the history.

**That is wrong.** The Store is a git branch — `hyper-store`, twenty-five commits in this repository:
eight Record versions, twelve Steps, and a `Begin`, an `End` and the branch's own creation — and a
clone gets it like any other branch. The account is exactly as portable as the Procedure.

It is not wrong for want of being told. The orientation says *a Store that is append-only and travels
in the repository* — one subordinate clause, in a paragraph whose subject is the `--response` file and
why not to author a throwaway Operation to look at a body. The session read `AGENTS.md` in full at its
fifth call and the clause did not survive to its answer, which is what a fact stated in the wrong
paragraph does.

Nothing on the surface says it either. `runs`, `records`, `changes` and `run show` render the account's
content and never its location; `git status` shows a clean tree because the branch is not checked out;
and the three commands that would name it — `install`, `store init`, `compact` — are the three the
orientation puts on the far side of the line an agent may not cross. Every route to the fact is
blocked or unluckily placed, so the agent searched the one place the Store is not.

This is the run's most expensive fault and the only one that reached the human as a false statement.
It is issue #233.

## What the run establishes, and what it does not

**Establishes.** A run-capable task is completable from the shipped surface. `run`, `run --dry-run`,
`runs`, `run show`, `records` and `changes` were all reached, all read correctly, and the division of
labour between them — `records` finds a version, `changes` reads a change — was learned inside the run
from their outputs alone. The Store initialised by the fixture was written to by two Runs and survived
being read back four different ways. Observe-before-mutate was authored unprompted, and `bound: 1` was
declared on every effectful Step without the task or the orientation asking.

**Does not establish.** No Refusal was met, so this transcript says nothing about repair. Nothing here
exercised Repeatability: every Operation used was `repeatable`, no Step skipped, and the second Run
that would test any of it is `tenant-onboarding`'s. No `destroy`, no Tombstone, no Bound halt — `bound:
1` was declared and never tested, each Step's Expansion being a single identity. And the fleet is three
directories on one host behind one Target, so *fleet* is the task's word rather than the run's: nothing
here says anything about more than one machine.

**And the ordering the run is praised for was never load-bearing in it.** Observe-before-mutate is
the design this transcript exists to show, and the rehearsal minted every pre-state Observation before
the effecting Run reached its first `cat` — so the Steps that were supposed to capture the pre-state
captured nothing, and the answer survived for a reason the session did not author. Had it not
rehearsed, the ordering would have carried the run. The design is right and this run did not test it.

## What was considered

**Changing something on the product because the account is hard to locate.** Refused, on ADR-0106's
line. Issue #233 has its own acceptance criteria and a real choice inside it — a location fact on a
surface, or a clause moved to the paragraph an agent reads for it — and neither is a decision this ADR
gets to make on one transcript. _ADR-0113 decides it:_ both, one sentence in two registers — `runs` and
`records` begin every answer with where the record is, and the orientation states it in the paragraph
about reading the record back rather than in the one about the `--response` file.

**Sharpening `fleet-rollout` so that a rehearsal is not available.** Refused, and it is the same shape
ADR-0106 refused for the `409` that never fired. The rehearsal was the *correct* answer to an unmet
question about the working directory, arrived at after `probe` was correctly declined. A task that
punishes the right answer measures nothing. What the rehearsal exposed is worth more than the cleaner
transcript it cost.

**Treating the rehearsal's Observations as a defect in the Store.** Refused. §6 states the behaviour
in one sentence and gives the reason; the Records are correct, complete, and rendered correctly by the
surface whose job that is. What is ticketed is the two places a reader cannot tell — a missing marker
and a missing rendering — and not the recording itself.

## Consequences

- **Issue #221's user story 3 is answered, and §6, §7 and the Store have been reached by a
  transcript.** What replaces *can an agent reach a Run* as the open question is *can it reach a
  second one*, which is `tenant-onboarding`'s, and *what does it do with a Refusal*, which is
  `release-promotion`'s.
- **ADR-0104's decision is vindicated on its own terms.** The Store was shipped so that a
  run-capable task could get past the first call, and the whole of §6, §7 and §8 downstream of it is
  what this transcript is made of. The measurement that decision gave up — whether an agent correctly
  reports a Store-less repository — remains given up.
- **`fleet-rollout` stands as authored.** No change to the task follows from this run.
- **Three defects, ticketed with this transcript as their evidence**: issue #233 (the account's
  location is unreachable from every surface an agent is allowed to call — ADR-0113), issue #234 (a
  Record version does not say the Run that wrote it was a rehearsal — ADR-0114), issue #235 (nothing
  renders a Record's fields until a non-rehearsal Run exists).
- **ADR-0100's fix holds on a third transcript.** `review` returned its page in the structured
  content twice and the session read both.

## Found in the same run, and not this decision

**A clean `check` still reaches the agent as an empty rows array.** Twice here, as
`{"rows":[],"truncated":false}`. ADR-0099 recorded it, ADR-0100 reasoned about it, ADR-0106 declined
to ticket it because it cost that run nothing. It cost this run nothing either: the session read the
empty array as clean both times and was right both times. Three transcripts now agree, which makes the
absence of a cost the finding rather than an accident of one run.

**The MCP tools arrive deferred, and two calls of every transcript go on fetching their schemas.**
This client lists all thirteen `hyper` tools at init and loads none of their parameters, so the session
spent calls 2 and 10 on `ToolSearch` before it could call `check` or `run`. The `monitor-coverage` run
spent three the same way. It is a fact about the client rather than about `hyper`, it changes nothing
about what these runs measure, and it is recorded here so that a call count read off a transcript is
read with it in mind.

**`repo_dirty: true` on every Record this run wrote.** The artefacts were authored and never committed
— the orientation ends the loop at the diff and the task asked for no commit — so every version's
Provenance records a dirty tree. The session noticed and reported it accurately. It is the correct
behaviour and the correct report, and it is worth knowing that the honest state of a repository at the
end of the authoring loop is one whose Records all carry the marker.
