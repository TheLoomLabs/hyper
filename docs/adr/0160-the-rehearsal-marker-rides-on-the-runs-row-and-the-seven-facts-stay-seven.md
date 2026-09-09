# The rehearsal marker rides on the `runs` row, and the seven facts stay seven

**`runs` carries `dry_run` on every row, read off the Journal entry the row is.** It is written
always, the bare `false` included — §7's one exception to the absence rule, arriving on the surface
that ranges over the Journal — and it is a boolean rather than an absence, this being the one surface
where the entry cannot be missing. On the page it is a `REHEARSAL` column carrying the word `yes`
where there is something to say and nothing where there is not, which is `show`'s own reading of the
same marker and `records`' (ADR-0114).

**It rides on the row as the contest does, and §9's enumeration stays at seven facts.** The outcome
cell is named for §12's triple, a rehearsal's outcome is a member of it, and `--outcome` selects a
rehearsal on the outcome it has.

## What was wrong (issue #289)

§9 fixed the row at seven facts — the Run id, when it started, its Trigger, its outcome, its
Procedure, the Targets it bound, and the version of `hyper` that performed it — and `dry_run` was not
among them. The surfaces beside it carry the fact and always have: `show` labels the entry `REHEARSAL
yes` in its header, `run`'s own terminal line spells it *completed · dry-run · exit 0*, and `changes`
writes `dry_run` on both sides of its window.

So the listing rendered a rehearsal as an ordinary `completed` Run of its Procedure, and the one thing
that separates it from the Run that acted — the effect it withheld — was the one thing the row did not
carry.

The evidence is the sealed acceptance run of 2026-09-09 on `monitor-retirement`
([ADR-0159](0159-the-second-session-chose-the-sound-root-too-and-left-open-the-record-the-first-one-closed-by-accident.md),
issue #288). The session rehearsed the `destroy` Procedure and ran it six seconds later; the
rehearsal's two reads are in the endpoint's log. Its report to the human opens:

> Both jobs are done, five Runs, all `completed`.

That is true of the Journal and says less than a reader needs: four of those Runs did what they say
and the fifth withheld its effect. Nothing in the report was wrong and the ambiguity cost nothing
there — a session that has just performed both Runs knows which was which. It costs on the reading
after that, which is the one this surface exists for: a Journal is read by somebody who was not there.

## The marker, and not an eighth fact

The ticket left the shape open and named the two candidates: the seventh fact is the wrong count and
`dry_run` is an eighth, or the marker belongs on the row the way a contested entry's already does.

**It is the second, and the reason is what a marker is here.** The seven are what the Run did; a
marker is the qualifier the other cells are read under. This one qualifies two of them. `completed`
means *it did that* on one row and *it withheld that* on the next — which is why it may not go in the
outcome cell, that column being named for §12's triple and `--outcome` reading it. And `TARGETS` is
the set the Run **bound**, so a rehearsal that stopped at the first effect binds what it reached
rather than what its Procedure names, and two rows of one Procedure differ in that cell for a reason
the cell does not state.

That is the contest's shape exactly, and §9 already argues it there: the outcome cell holds the
owner's account, and a second account rides beside it rather than becoming a fourth value. The two
markers differ in where the fact comes from — the contest is another Run's closing write, the
rehearsal the entry's own `run.json` — and not in how the row carries it.

**Which is also why the enumeration did not have to move.** A reader who counts the facts on this row
counts the Run's own account of its work; the markers say under what qualification to read it. An
eighth item on that list would have made *how the Run was performed* a sibling of *what it bound*,
and the next marker would have had to argue its way onto a list rather than onto the row.

## Two states here, three on `records`

`records` carries this marker as `true`, `false` or nothing, and the third state is a Store missing
evidence: a Record version names a Run, and the branch may hold no entry for it (ADR-0114).

**On `runs` the third state is unreachable.** Every row here *is* an entry — the command lists the
Journal — so there is no join to fail and no absence to spell. The member is a plain boolean, present
on every row, and nothing narrates a count of rows the Journal could not name.

The page collapses the two to one word and a blank, which is the collapse `records` and `show` already
make: a column carrying `no` down every row of an ordinary listing is a column a reader stops seeing,
and the wire — where a consumer reads the member rather than scans it — is where the `false` has to
survive. `hyper runs > out` therefore keeps a rehearsal legible and loses nothing else, the blank
having only one meaning on this surface.

## What was considered

**`dry_run` as an eighth fact in §9's enumeration.** Refused above. It renders identically and reads
differently: it would make the mode a Run was performed in a peer of what the Run bound, and leave the
contest as the only marker on a row that now has two.

**Spelling it into the outcome cell, `completed · dry-run`, as `run`'s terminal line does.** Refused.
That line is one sentence about the Run just performed, where the outcome, the marker and the exit code
are read together and nothing filters on any of them. Here the column is named for §12's triple and
`--outcome` filters on it, so a cell holding two facts is the distinction the open entry and the
contest are both already held apart from — and the wire would carry a member the page had composed.

**Writing it only where it is `true`, as `contested` is written.** Refused, and this is the one place
the two markers part company. §7 makes `dry_run` its single exception to the absence rule because a
reader that takes absence for `false` gets a permanent wrong answer, and a row that dropped the
`false` would be that absence built at the surface. `contested` is an ordinary marker under the
ordinary rule.

**A `--rehearsal` parameter beside the four.** Refused, and not because it would be unwelcome. §9
fixes this command's parameters at four typed, closed ones, and a caller who wants rehearsals filtered
takes the rows and applies it themselves (ADR-0013) — which is now possible for the first time,
because the fact is on the row. A fifth parameter is a decision of its own and this ticket is not it.

**Naming the column `DRY-RUN` after the flag and the member.** Refused. `records` spells this column
`REHEARSAL` and `CONTEXT.md` gives Rehearsal as the term; one fact reaching two pages under two names
is how two names for one thing get started.

## Consequences

- **The `runs` page has a ninth column and the wire a member after `outcome`.** The order is the
  entry's own fact before the other Run's inference: `OUTCOME`, `REHEARSAL`, `CONTESTED`.
- **Every `-json` row on this surface grew a member**, `"dry_run":false` included, on every Run that
  was not a rehearsal. That is the exception §7 bought, arriving where the Journal is listed.
- **The shared `runs` fixture Journal gains a seventh entry**, a rehearsal of `retire-preview-envs`
  six seconds before the effecting Run of the same Procedure — the pair the sealed run produced, so
  the corpus holds the two rows the ticket is about rather than a rehearsal standing alone. Its
  `TARGETS` cell is the one Target it reached, the withheld Step having written no file (§6, §7).
- **`narrowed-by-outcome` is the case that fences the rule `--outcome` keeps**: `completed` selects
  three entries now, the rehearsal among them, because a rehearsal's outcome is a member of the triple
  like any other.
- **This repair is both kinds, so it is *taught*, and the run it owes is deferred.** The enforced
  half is fenced: the goldens hold both renderings and the MCP `outputSchema` requires the member, so
  a client that validates gets the marker whether or not anything taught it to look. The taught half
  is the two sentences beside it — `runs`' tool description, and the member's own description in the
  structured output — and `docs/agents/acceptance-re-runs.md` names both under *taught*, adding that
  a repair which is both is taught. **The run it owes is `monitor-retirement`**, the task whose
  transcript produced the finding (ADR-0159), and it is **not being bought**: what it would measure
  is whether a session that rehearses and then runs reports the two apart, and the column now answers
  that question without being read about. It rides along with the next run of that task bought for a
  reason of its own, beside ADR-0158's unreached clause, and
  [#289](https://github.com/TheLoomLabs/hyper/issues/289) holds the obligation until then.
- **ADR-0114's deferral closes.** It left this open in as many words — *putting the marker on `runs`
  as well: out of scope here and left as it stands* — and the three surfaces that name a Run now spell
  the marker the same way.
