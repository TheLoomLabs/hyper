# A rehearsal is the Comparison's subject where a caller names it as one

**`changes` takes a third way of naming a window: `--subject <run-id>`, which names the subject and
leaves the baseline to §8's rule — the Run before it, a rehearsal and an open entry passed over.** It
is the one place a rehearsal may be a side of a window, and it is what makes the Observations a
`run --dry-run` recorded legible before any effectful Run exists.

**Nothing else about a rehearsal moves.** It is never a baseline, under any form. It is never the
subject a window chose for itself. Run-once Repeatability and the identity digest filter its entry
exactly as they did. §7 named four independent readers that filter a rehearsal out; three are
untouched and the fourth is now *the subject the Comparison chooses for itself* rather than *the
subject*.

**Both sides of a `window` row carry `dry_run`, written always**, the bare `false` included — §7's one
exception to the absence rule arriving on the surface that names two Runs, as ADR-0114 brought it to
the surface that names one. On the page it is a stated line beneath the header rather than a column.

## What was wrong (issue #235)

Between a rehearsal and the effecting Run that follows it, **nothing on the product would show what
the rehearsal read**. The Observations were in the Store, complete and correct, and every surface that
could have rendered their values either did not carry values or had no window to render them in:

- `records` carries no field values, and that is §9's contract rather than a fault — *it is the one
  whose job is finding a version rather than reading a change*.
- `run` and `run show` carry none either: *what each Record did is the Comparison's rendering rather
  than the Run's*.
- `changes` is the one surface that renders fields, and it had no subject to take. §8's subject was
  the most recent Run that is neither a rehearsal nor dirty, and with only a rehearsal in the Journal
  there is none.

So the surface that had the values could not select the Run, and the surfaces that could select the
Run did not have the values. The gap closed only when an effectful Run happened — only after the point
the rehearsal existed to inform.

The evidence is the sealed acceptance run of 2026-08-30 on `fleet-rollout`
([ADR-0110](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md),
issues #223 and #232). The task is a rollout across three machines whose closing question is *which
machines were behind, and what each of them was on*. The session authored its four `read` Steps
deliberately ahead of its three `mutate` Steps so the pre-state would be recorded before it was
overwritten, rehearsed, and then went looking for what the rehearsal had captured:

```
records --definition host-ops        → identities, ordinals, run ids. No values.
run show <rehearsal run id>          → dispositions and provenance. No values.
changes --procedure roll-fleet-…     → {"rows":[],"truncated":false}
```

and drew the conclusion out loud:

> `changes` is empty because it compares two Runs and there is only one — no baseline yet. **So the
> pre-state field values aren't legible until a second Run exists.**

It then ran the Procedure for real, which the task authorised, and `changes` rendered all five
Observations with their values, `1.3.9` and `1.2.0` among them. **The operator this describes is the
operator `--dry-run` is for**, and the only way to see what they were deciding on was to do the thing
they were deciding whether to do.

## Why the two ends part company

ADR-0010 is the constraint, and it is not weakened here. `hyper` has no plan: nothing renders a
proposed change before it happens, and a rehearsal is no evidence of what the world **became**.

That is a statement about a **baseline**. A baseline is the thing a later Run is measured against, so
a rehearsal standing there would retire the warning a real Run earned — the Comparison would report
*nothing moved* over an effect nobody performed. Nothing in this decision touches it: a rehearsal is
refused as a baseline under every form, including the one behind a subject a caller named.

A **subject** asserts something different. §8's own sentence for these tables is *this differs from
when we last looked*, and a subject is the later of the two looks. §6 is explicit that a rehearsal
looks: *a dry-run performs the reads it reaches … those reads really happened, so they record
Observations like any other*. Asking *what did this rehearsal see* is therefore answered off versions
that exist, in the surface built to render versions, and the answer is a claim about a read rather
than about a change. No proposed change is rendered anywhere: `YOU DID THIS` is empty by construction
where the window has no baseline, a rehearsal reaching no effectful Step, and where there is a
baseline that table holds what moved between two instants — which is what it holds on every window,
including the ones §8 already admits another Procedure moved a Record inside.

**The rule is the position and never the spelling.** `--between` names both ends, so a rehearsal in
its second id is a rehearsal named as a subject and is taken; in its first it is a baseline and is
refused, with the remedy on the same line:

```
hyper changes: run 01991d00-… is a rehearsal, which is never a window's baseline — a --dry-run is no
evidence of what the world became; --subject 01991d00-… renders what it read
```

## Why a third window form rather than the two other candidates

The ticket named three and decided none. Its acceptance criteria enumerate three of §7's four
rehearsal filters as invariant and leave the fourth — the Comparison's subject — off the list, which
is the shape this is.

**A value-carrying flag on `records`, e.g. `--fields`.** Simpler, and it puts values on the surface §9
says is not for reading changes. §9's division of labour is stated with its reason and the reason
still holds; widening `records` is a decision against a stated one where this is a distinction inside
one. It would also have given `records` a second job on a page that already carries twelve columns,
and left `changes` — the surface a reader goes to for values — still answering nothing for the window
that has them.

**Rendering the projected fields in `run`'s Step rows under `--dry-run`.** Narrowest, and it answers
the operator where they are already looking. It was refused on **re-readability**: it answers at the
moment of the Run and never afterwards, so an operator who rehearsed yesterday, or an agent whose
context has moved on, is back in the gap with the versions still in the Store. It would also make
`--dry-run` the one path on which a Run's rendering carries field values, which is the division §8
states in as many words.

**Naming the subject by id is also the general thing that was missing.** Before this, no caller could
ask for a Comparison of any Run other than the newest without knowing its predecessor's id to write
`--between` with. `--subject` is that question, and the rehearsal is the case that forced it.

## The baseline behind a named subject

`compare.Preceding` is the newest nameable Run of the same Procedure to have started before the
subject, on §9's ordering for Runs — `started_at` descending, ties broken on the Run id descending —
and an absent side where there is none, which renders §8's existing *no baseline — first Run of
`<Procedure>`* state.

It is a function in `internal/compare` beside `Select` and not a rule the command writes, because it
**is** `Select`'s rule: the same standing filter, the same one-Procedure window, the same ordering. A
baseline chosen one way where the subject was typed and another way where it was derived is two
answers to *which Run is the baseline*, and the two would drift.

The subject is excluded from its own baseline search by the comparison rather than by identity: the
ordering is total over both keys, so a nameable subject present in the listing compares equal to
itself and drops out with everything after it. A window whose two ends were one Run would render a
Comparison of a Run against itself.

## A stated line, and not a seventh column

The page marks a rehearsal subject beneath the header:

```
  retire-preview-envs
  BASELINE  01991c3a-7d40…  cron           Tue 4 Aug 09:12  completed  1m48s  procedure rev a91f0c2
  SUBJECT   01991d00-0000…  igor@thinkpad  Wed 5 Aug 10:00  completed  12s    procedure rev b0c94f1
  rehearsal — the SUBJECT entry is a --dry-run: it reached no effect, so nothing below is a change it made
```

`records` gave this marker a column and `show` gives it a word in its header; this page gives it a
line, and the three are one discipline rather than three readings. The header here is a two-line
aligned block whose columns are the six facts §8 names each Run with. A seventh, blank on every
Comparison but the ones a caller asked for by id, is a column the page carries in order to say
nothing — and the page already has a shape for *a qualification of a side that is not one of those six
facts*: the contest line, one per `closed-by/` file. The rehearsal line stands above those because it
says what kind of Run this was where they say how one ended, and a reader who has not been told the
subject is a rehearsal is reading the tables wrong already.

There is no loop over the two sides. A baseline is never a rehearsal under any form, so a line for the
other end would be a sentence for a window that cannot exist.

## What was considered

**Letting `Select` take a rehearsal as subject where a Procedure has no other Run.** Refused. It would
make the answer depend on what else is in the Journal: the same command against the same Procedure
would silently stop reading the rehearsal the moment a real Run landed, which is the surface deciding
for the caller rather than answering them. *Named by id* is a fact about the question asked.

**Refusing a rehearsal in `--between`'s second position, so `--subject` is the only door.** Refused.
Both name a subject by id; the difference is only whether the baseline was typed too, and refusing one
spelling of one question is the surface deciding on the spelling.

**A marker on the `TOTALS` line, or on each row of the tables.** Refused. The fact is about the
**window**, not about any row in it, and repeating it per row would put it where §8 says a row's cells
are the Record's own facts.

**Leaving `dry_run` off the baseline's side of the wire, it being always `false`.** Refused. §7's
exception exists because a reader that takes absence for `false` gets a permanent wrong answer; an
exception that held on one member of a pair and not the other is a shape a consumer has to learn
twice, and this is the same decision ADR-0114 made one surface over.

**Fixing issue #230 with this.** It is not fixed. #230 is *a Manifest author cannot see the response
their projection reads from*, and its documented workaround — author an Operation that projects the
body, run it, read the Records back, delete the Operation — needed a **real** Run because
`changes` needed a non-rehearsal subject. That step is now a `--dry-run`, so the workaround is one
step less bad; the fault it works around is untouched.

## Consequences

- **`changes` has a third window parameter and the usage error generalises.** `--since`, `--between`
  and `--subject` name one window three ways, and the message names which two — or three — were given.
- **`compare.Preceding` is the second exported rule about which Run is a side of a window**, beside
  `Select`, and it is the whole of what `--subject` leaves derived. `StandingOf` is unchanged: what
  moved is who consults it, not what it says.
- **Two refusal sentences are now written once and called from both forms** — the unknown Run id and
  the open entry — where `--between` held the only copy.
- **The `--between` rehearsal message changed** and now carries a remedy. A caller who named a
  rehearsal as a baseline is told the flag that renders what it read.
- **Every `changes` window row on the wire gains `dry_run` on both sides**, which is a schema change
  to `windowSide` and a required member. A consumer reading the row by name is unaffected; one
  asserting the exact key set is not.
- **`YOU DID THIS` renders empty on a baseline-less rehearsal window**, header and count, as every
  empty table on this page does. That is the honest rendering of *this Run did nothing*, and it is
  the same three-zero-row shape a Run whose every Step skipped reads as.
- **ADR-0110's second cost closes.** That transcript paid twice: once at `records`, which ADR-0114
  answered, and once at the gap between the two Runs, which is this.
