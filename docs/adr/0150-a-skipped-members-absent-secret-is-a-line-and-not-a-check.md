# A skipped member's absent secret is a line, and not a `check`

`check` gains no member. An Operation declaring both `repeatability: skip-if-recorded` and `secret:`
output stays legal, and what changes is that the Run **says so**: beneath the Step table, one line per
Step that concluded about a Record without calling for it while declaring secret output, and one line
under them carrying the reason. The same fact rides on that Step's `step` row as `secrets_skipped`.
Issue #273.

This is the decision [ADR-0148](0148-a-secret-sink-is-a-directory-hyper-makes-and-one-file-holds-one-value.md)
declined in the sentence that stated the behaviour: *whether §4 should refuse `skip-if-recorded` on a
secret-producing Operation outright is a separate decision and is not taken here*. It is taken here, and
the answer is the third of the three the issue put on the table — neither refuse nor stay silent.

## What the state was

A Run over such an Operation behaves like this. On the first Run the member has no Asset, the call goes
out, and the value lands at `<sink>/<nnnn>/<name>/<field>`. On the second the Asset stands, the Step
reports *skipped as already recorded*, no call is made, and **nothing is written to the sink for that
member** — while the Run reports `completed` at exit `0` and the sink directory it was given was still
created.

**Nothing is lost, and that is what tells this from ADR-0146's defect.** There the value was produced
and discarded; here it was never produced. Every guarantee holds and the tree is correct.

What was wrong is that the tree was correct and unreadable. An operator who typed `--secret-out <path>`
said *I want the value on disk*; on the second Run they get part of a tree and exit `0`, and a wrapper
doing `cat "$sink/0001/db-1/password"` gets *no such file* where it expected a credential. The absence
was legible only to a reader who already knew the Repeatability — three artefacts away from the shell
they are standing in.

## Why `check` is not the place

**The rule would refuse a Manifest that works.** §4 charges that cost nowhere else. Every member of
`manifest-inconsistent` refuses a shape that could not do what it says: a `skip-if-recorded` on a
non-`mutate` Kind reads a projection its Kind cannot produce (ADR-0037), and one whose `identity:`
resolves only from the response declares a test that cannot run before the call it decides
(ADR-0056). A `skip-if-recorded` Operation declaring `secret:` output runs, produces the value the
first time, and does exactly what its author wrote.

**And the author may have meant it.** *Mint a credential only where we hold no record of one* is a
coherent declaration and a common one, and on the second Run the empty sink is then the **correct
report**: we did not rotate, so there is nothing to hand you. Refusing it strands that author with no
spelling for a thing the tool can plainly do.

**§4 is already careful in exactly this neighbourhood.** It says of the Cadence rules that a `when:`
that never holds and *a `skip-if-recorded` that always skips are alike beyond `check`'s reach, and
refusing them is refusing a shape somebody meant* (ADR-0038). The refusal proposed here is the same
shape of guess with a smaller radius, and the sentence one paragraph up is the argument against it.

## Why silence is not the place either

The case for a rule was never that the artefact is broken. It was that the **operator has no sentence to
read**, and that half of the argument survives the rule being declined. §9 said *the absence is the
answer* and left the answer somewhere no surface stated it.

**The wholly skipped Step is the easier half and the mixed Step is the case that decides it.** A Step
whose every member skipped renders *skipped as already recorded*, which explains the empty tree to a
reader who already knows the Operation declares `secret:` — incomplete, but adjacent. A Step that
skipped two members and called for one renders `ran` and a `RECORDS` of `3` (ADR-0056), which is
**byte-identical** to the row of a Step that wrote every value it concluded about. On that Run there is
no surface anywhere — page, `--json` stream, envelope, entry, `show` — from which the shortfall can be
derived. That is not a legibility complaint; it is a fact the tool holds and does not report.

## The decision

**One line per Step, and the reason once.**

```
  STEP  ID    KIND    DISPOSITION  RECORDS
  1     mint  mutate  ran          3

  step 1 skipped 2 records and wrote no secret for them.
  a member skip-if-recorded found already recorded makes no call, so there was no value for the sink to hold.
```

**It stands where a rehearsal's *stopped at* line stands**, between the table and the terminal line, and
for the same reason: the cells above are true and incomplete, and what the line says is what their
silence means. That is the precedent this follows rather than an analogy — the rehearsal's sentence
exists because *never reached* is a Disposition several Steps share and one of them is the boundary,
which is this page's problem with `ran` (§8, ADR-0091).

**The reason is written once however many Steps carry the fact.** Two secret-producing Steps in one
Procedure is a Run `check` permits (ADR-0148), so two of them skipping is a page that exists. Repeating
the reason under each would be one sentence about one Repeatability value said twice.

**The Step is named by position and not by its authored id**, which is the one place the line departs
from the rehearsal's beside it. What a reader does with the line is go and look at the sink, and the
sink's leading segment is that position spelled `<nnnn>` (§9, §12, ADR-0148). Naming the Step the way
the directory names it is worth more here than matching the neighbouring sentence's habit.

**The line says the absence and never the value.** A line naming the credential a member did not mint
would be the prospective rendering `hyper` does not have, arriving on the surface of a Run that
deliberately did not act (ADR-0010).

**`secrets_skipped` is a count on the `step` row and carries no key where it would be zero.** §7's
absence rule is the whole of its semantics, as it is `withheld`'s: a Step that skipped nothing, and
every Step whose Operation declares no `secret:` at all, has no absence to account for. It is written
on the mixed Step and on the wholly skipped one alike, because the Disposition discriminates neither —
*skipped as already recorded* is carried by Steps that produce no secret, and the mixed Step reports
`ran`.

**It is a count against the identity set, and a Step carrying no set carries no count.** That is not a
detail of where the line sits in the engine; it is what the number means. *attempted, world untouched*
concluded about nothing by construction and renders the dash (ADR-0062) — and a Step that skipped one
member and whose next request provably never left is exactly that Disposition, having reached the world
nowhere. A row claiming a skipped Record beside a cell saying no set exists would be one Step saying
both. `a-step-with-no-set-counts-no-skipped-secret` is that case rather than that sentence.

**On a Step that halted and renders `n of m`, the count is of the skips among the `n`.** Members the
halt never reached were neither skipped nor called, and *unaccounted for* is the arithmetic between the
set and `expanded_to` (§7, §8). Folding them in would make one number answer two questions, and the
second already has a column.

**It is scoped to a secret-producing Step, and it is not the skip split.** How a mixed Step divided
into members that called and members that skipped is derivable — under this value a member that runs
always mints a version, a standing head having skipped it — and ADR-0056 declined to carry it on the
Step for that reason. What is not derivable from anything is *which sink entries are missing*, and that
is the fact scoped here.

**It rides on the row a Run writes and not on the entry `show` reads back.** The sink is a directory on
the machine the Run was invoked from: it never reaches the Store, no Run reads it, and it is not on the
branch (ADR-0007, ADR-0011, ADR-0148). Which values a Run put on one operator's disk is not a fact a
branch anybody can clone has any business carrying, and the Store schema does not move.

## Considered options

- **Refuse it at `check`, as a fourth shape of `manifest-inconsistent`.** The issue's own case for it,
  and the shape reads naturally beside ADR-0037's and ADR-0056's members. Rejected above on two counts
  that are one: it refuses an artefact that works, which §4 does nowhere, and the artefact it refuses is
  one a credential-rotation author meant to write. The cost is not hypothetical — `skip-if-recorded` is
  a `mutate`-only value (ADR-0037) and *mint only what we have no record of* is what a `mutate` under it
  is for.
- **Stay silent, and let §9's *the absence is the answer* stand as the whole account.** Rejected on the
  mixed Step. A wholly skipped Step at least renders a Disposition a reader can chase; a mixed one
  renders the same row as a Step that wrote everything, and the shortfall is then a fact no surface in
  the tool carries. *The absence is the answer* is true and is not an answer anybody was given.
- **A Refusal, on ADR-0146's own ground.** Rejected because the ground is not the same. That record
  refused a Run that *destroyed* a value it was invoked for, and one way of saying *I will not do this*
  at exit `77`. Nothing is destroyed here, so a Refusal would decline a Run whose every act was correct
  — and it would decline the second Run of a rotation Procedure whose first Run was the point.
- **A seventh Disposition, or a Disposition that splits.** Rejected on ADR-0056's own argument, which
  this record does not reopen: *ran* already covers a Step that made five hundred calls, and a partial
  value would invent a boundary case where the two existing values cover the range. The fact is a
  member and a line, which is where facts that are not Dispositions go.
- **A general `skipped` count on every `skip-if-recorded` Step's row.** Tempting, and one member rather
  than a scoped one. Rejected because it is the split ADR-0056 declined, arriving through a different
  door: it is derivable from the Comparison, and putting it on the row makes the Run's stream the second
  place to read a fact the Comparison already renders. What is not derivable anywhere is this record's
  scope, and the member is cut to it.
- **A warning on stderr rather than a line on the page.** Rejected. The page is the answer and stderr is
  narration (§9), and this is a fact about what the Run did — the same reason the rehearsal's sentence
  is on the page. It would also be the one fact about the sink that a `--json` consumer could not reach,
  the stream carrying no narration at all.
- **Carry it in the Store, on the Step file.** Rejected above: it is a fact about a directory on one
  machine, and the entry is pushed to a remote.

## Consequences

- **`check`'s closed set does not move.** No new `error_code` and no new shape of
  `manifest-inconsistent`; §4 states the non-refusal where it states the two Cadence rules, so a reader
  who expects the family to hold this member finds out why it does not rather than finding nothing.
- **The `step` row gains a member and the MCP output schema declares it.** `secrets_skipped`, an
  integer, minimum 1, absent everywhere else. It is additive on the wire — a consumer matching nothing
  new is unaffected — which is what this repair being a *report* rather than a rule buys.
- **Two Dispositions now carry it and no Disposition implies it.** Nothing infers the member from
  *skipped as already recorded*, and nothing may: that value is carried by every skipping Step in every
  Procedure that produces no secret at all.
- **The corpus gains five cases and loses none.** `a-skipped-member-writes-no-secret` is the mixed Step
  — three members, two already recorded, one call, one directory in the sink — with its `-json` and MCP
  twins carrying the row member; `every-member-skips-and-the-sink-holds-nothing` is the wholly skipped
  Step, whose `sink.golden` is the one node the sink has; `a-step-with-no-set-counts-no-skipped-secret`
  is the boundary above, a skip followed by a request that never left, rendering the dash and writing
  neither the line nor the member. `repo-skip-secret` is the fixture all four argv cases drive: the
  first Manifest under `testdata/run/` declaring `skip-if-recorded` and `secret:` together —
  `check/error-code-coverage` holds the only other one, where the pair is incidental to a repository
  carrying one fault per code — and it is the combination this record decided not to refuse, so the
  fixture is what keeps it running as well as what drives the report.
- **`sink.golden` earns its fourth reading.** The other three goldens are blind to this the way ADR-0148
  said they were blind to a lost secret: `store.golden` holds the marker on all three Records whether
  the values landed anywhere or nowhere, and the empty-sink case's tree is what says two of them did
  not.
- **§13's honest limit gains its second half.** *`skip-if-recorded` trusts the record over the world*
  now also says what that costs an operator holding a sink: the value an earlier Run minted is beyond
  `hyper`'s reach the moment that Run ended (ADR-0007), so the only route to it is minting another —
  which is the Repeatability being declined. Stating the absence does not make it recoverable and does
  not pretend to.
- **The repair is taught, and the run it owes is `push-credential` (#271) — the same run ADR-0146 and
  ADR-0148 owe, now bought and read back in
  [ADR-0149](0149-the-sink-was-supplied-both-secrets-were-lost-and-check-was-clean-throughout.md).** Per
  `docs/agents/acceptance-re-runs.md` this repair is both taught and enforced, so it counts as taught:
  the line, the member and the schema are fenced by the corpus and by `internal/cli`'s cases, and what
  is taught is the `run` tool's `secret_sink` description — an agent that reads *a member a
  skip-if-recorded Step found already recorded made no call and produced no value* and then treats an
  empty path as a failure gets no test failure anywhere. **It is not being bought here, and the task it
  would be a run of does not yet reach the surface.** `push-credential` is the one task in the set that
  reaches a Step declaring `secret:` output, and what this repair changes is met only on a **second**
  Run over a standing Asset: its two Runs are the Refusal and the round trip past it, and the Manifest
  is the agent's own to write, so nothing in the task asks for `skip-if-recorded` or for a Run after the
  one that minted. Closing that is therefore **not a task-file line but a change to what
  `push-credential` measures** — a third Run, and a prompt that makes the Repeatability the agent's
  natural choice — which would move the baseline
  [ADR-0149](0149-the-sink-was-supplied-both-secrets-were-lost-and-check-was-clean-throughout.md) just
  established. That is a decision for the session that buys the run rather than a cheap half taken
  here, and this is the record of it being named and deferred.
