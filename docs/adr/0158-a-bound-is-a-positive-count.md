# A Bound is a positive count

A `bound:` below `1` is refused where it is written: `bound-not-positive`, a new member of §12's
closed `error_code` set, reported at `steps[N].bound` on every Step whose Kind admits a Bound at all.
The zero and the negatives are one code, `1` is the strictest Bound there is, and a Journal entry's
absent Bound therefore means *this Step declared none* and nothing else. Issue
[#287](https://github.com/TheLoomLabs/hyper/issues/287).

No artefact key moves, no rendering moves, and nothing moves at Run time. **One `check` row appears
for a Procedure that used to pass.**

## What the state was

`bound: 0` checked clean on any Step that may carry a Bound — every `mutate`, and every non-opaque
`destroy`. `internal/artefact/procedure.go` typed the key as a plain `schema.Integer` with no minimum,
and `checkStepBound` read only *where* a Bound may stand: `unknown-key` on a `read`, `bound-illegal`
on an opaque `destroy`, `bound-missing` on a non-opaque `destroy` carrying none. Nothing read the
value.

So the Step reached a Run, and Refused. `exceededBound` counts the Expansion against
`declaredBound(authored.Bound)`, and every Expansion resolving a member exceeds zero — so
`bound-exceeded` was such a Step's only reachable outcome, on every Run, for the life of the artefact.
The Step could act on no Run and no call it guarded could ever go out.

**And the Journal could not say which of two Steps it had been.** `store.Selector` writes the Bound
through `members.count`, which writes the key only where the value is non-zero — the Store's own
absence rule, which is right about every other counted thing it covers, a position, an attempt, a page
all starting at one. Under it a `bound: 0` Step's entry carried no `bound` key, byte for byte the entry
a Step declaring no Bound leaves, and `show --expansion` rendered no `BOUND` line over either. A reader
could not tell the strictest Bound there is from no Bound at all, and `hyper show` said *this Step
declared no Bound* about a Step that had declared the tightest one it could.

Issue [#286](https://github.com/TheLoomLabs/hyper/issues/286) is what made this reachable and is what
found it: before it no Step recorded a Bound at all, so the collision had nothing to collide with. Its
fix is correct as it stands and this is the case it deliberately did not widen into — both files it
touched say where the zero stops.

**Nothing in the corpus authored one.** Every Bound in the tree is 1 or more — 1, 2, 3, 4, 5 and 9 —
and no `show` seed carried a zero either. A case that authored one below that would have been the
first, and the fixture this ticket adds is it.

## The decision

**A Bound is a count of Records a Step may affect, and a count that admits none is not a guardrail.**
It is a Step that can do nothing, written in a file a human reviews for what it will do. `check`
refuses it at the value: `bound-not-positive`, on the line the author edits.

The zero and the negatives are **one code because they are one fault**. A Bound errs in the declining
direction by design — §4 holds `bound:` to an integer and refuses one only where it may not stand at
all, so a guardrail nobody meant declines rather than admits — and below `1` is where that direction
runs out: the value stops being a Bound that is too tight and becomes one nothing can satisfy.
Splitting them would have cost §4 an argument for why `0` differs from `-1` that nothing in the
behaviour supports; both Refuse every Expansion, and both are the same edit.

It is checked **only where the key may stand**, as the switch's last arm. A `bound: 0` on a `read` Step
is `unknown-key` and on an opaque `destroy` is `bound-illegal`, each the answer to the question that
comes first — and a second row naming the count would teach an edit (*write a positive one*) that the
first row has already refused outright.

**And it is the only row such a Step draws.** The offline `bound-exceeded` check compares an authored
`over:` `values:` list against the Bound, and every list exceeds a Bound below one — so a `bound: 0`
Step with two members would have drawn both codes at once, on two lines, for one edit. The count is
not the fact that is wrong there: the number it was compared against is, and `checkBoundExceeded`
stands down where the Bound is not a Bound. The comparison is untouched everywhere the rule admits the
value, `bound: 1` over a two-member list still being `bound-exceeded`.

**What it settles, and the reason the ticket was opened, is the record.** With the value refused, the
zero is a Bound no checked repository holds, so no Run can write one: `check` re-runs in full at Run
start, and the smallest Bound that reaches `store.Selector` is `1`. The absence rule keeps its one
reading — *this Step declared no Bound* — and it keeps it because `check` guarantees it rather than
because the Store's encoder was taught a second value.

## What was considered

- **Make the Store member optional** — `*int`, or an `Optional` wrapper, so *absent* and *zero* are two
  values a Step file can hold. Rejected as the largest of the three and the one that buys the least: it
  moves `Selector.write`, the decode, `show`'s `numberText` call and §7's account of what the selector
  holds, so that a Journal can faithfully record a guardrail that declines every Expansion it ever
  guards. Recording an unauthorable thing well is worse than not admitting it, and the absence rule is
  older than this member and right about the rest of what it covers.
- **Refuse the zero and leave `bound: -1` standing**, which is issue #287's own wording of this option.
  Rejected on the argument above: the two behave identically at the guardrail, and a set that admits
  one and refuses the other owes §4 a distinction the tool does not make.
- **Leave it, documented** — where issue #286 left it, both files stating what the zero cannot
  distinguish. Rejected because the documentation is in the source and the reader who needs it is
  holding a `show` rendering that is the opposite of true.
- **Refuse it at Run time instead**, as a Refusal of its own. Rejected for the reason §4 and ADR-0061
  exist: the fault is on the page, knowable offline, with no Store, no credential and no response
  needed to decide it. A Run that Refuses where `check` was clean is the arrangement they both avoid.

## Consequences

- **`error_code` grows by one, to fifty-four**, and §4's static contribution to it from thirty-two to
  thirty-three. Every count that names either moves with it: §12's three, and the ones carried in
  `internal/verify`, `internal/run/gates.go`, `internal/cli/check.go`, `internal/cli/instructions_test.go`
  and the coverage test whose list the new member joins.
  `internal/cli/testdata/check/a-bound-below-one` is its fixture — one Procedure, a `mutate` at
  `bound: 0` and a `destroy` at `bound: -1`, two rows and one code.
- **One `check` row on a Procedure that used to pass**, and one however the Step is written: the
  offline `bound-exceeded` comparison stands down where the Bound is below one, so a Step carrying an
  authored `values:` list draws the same single row. Nothing in the corpus authored a Bound below one,
  so no golden moved and no fixture repository changed.
- **`declaredBound` still reads a negative and still counts against it.** No checked repository can
  reach it with one, and it is not this file's to reinterpret a value it was handed — a guardrail that
  widened where it could not read is the failure ADR-0064 names.
- **The MCP Step-selector schema's `bound` is `minimum: 1`** where it was `minimum: 0`, and carries the
  sentence that says what its absence means. Zero was never a value any binary wrote; the schema now
  says so.
- **The orientation's Bound sentence carries *at least `1`*** and the fence over it
  (`TestInstructions_TheBoundRuleIsTheOneCheckHolds`) gains a fifth combination — the rule `check` holds
  is the rule the text teaches, and this is the fourth thing `check` does with a `bound:`.
- **A sealed acceptance run is owed and is not being bought.** The repair is *taught* as well as
  enforced ([`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md)): the enforcement is
  fenced by the golden and the package cases, and the clause in `internal/mcp/instructions.go` is not.
  The task is **`monitor-retirement`**, the one whose Steps a `destroy`'s Bound is authored in and the
  task ADR-0129 read back. It is deferred because the clause it adds is five words inside a sentence an
  agent already reads for the Bound's two harder rules, and because no transcript has ever shown an
  agent authoring a Bound below one — the run is worth buying the next time that task is bought for a
  reason of its own, not on this clause alone.
  [#288](https://github.com/TheLoomLabs/hyper/issues/288) holds the obligation, beside ADR-0126's,
  which the 2026-09-02 run of the same task never reached: two deferrals now stand on one run, which
  is what `acceptance-re-runs.md` calls the argument for buying that task next.
