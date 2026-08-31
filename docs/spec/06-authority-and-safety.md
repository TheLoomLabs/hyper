# §5 — Authority and safety

Authority is decided before invocation, not at it. Most dangerous states are unreachable from the
five artefacts by construction, and `check` refuses the rest before the first effect reaches the
world. No confirmation happens at execution time and no per-Run approval exists; CI runs unattended,
and every guardrail below is checked before that unattended Run starts. This chapter states what
stands between an authored artefact and the world, in full: the two keys a Step must satisfy, the
envelope composition cannot widen, the halt a check performs without claiming anything, what an
`opaque` `destroy` additionally needs, the Bound a
`destroy` Step cannot omit, the reach a selector is granted by Kind, and the terminal outcome all of
it ends in when it declines rather than runs.

## The two keys

A Step runs only where its Definition's claimed Kind intersects its bound Target's accepted Kinds,
and a `destroy` Step's Operation must be named among those its Definition claims for `destroy` — the
check and its error codes are §4's (`kind-not-granted`, `operation-not-claimed`).

## The envelope

A Procedure's transitive Target and Kind envelope — everything reachable through every Procedure it
invokes, to any depth — is checked before either runs, so composition cannot widen blast radius by
accident (`envelope-exceeded`, §4).

## A check needs no authority to stop the work

A Requirement halts the Run where its predicate does not hold (§3, §6), and it claims nothing to do it:
no Kind, no Target, no Bound, no call. A Procedure whose Steps are all `read` and whose last entry is a
Requirement is read-only in authority terms and is still able to stop everything downstream of it,
including in whatever invoked it — one Run has one outcome however deep the invocation goes (§6).

**That is stated in this chapter because its absence was an authority fault.** Until it existed the only
thing that halted was an effectful Operation, so a shared check that had to be able to fail had to end
in a Step claiming `mutate` on the Target it was protecting. Everything about that was correct and it
read as the opposite of what it was: `review`'s authority table rendered effective `m` for a Procedure
that writes nothing, on the one artefact where a reviewer most relies on that column, and the Refusal
path became an effect path — a check could not protect a Target it was not also authorised to write
(ADR-0111, ADR-0116).

A Requirement is not a guardrail and its halt is not a Refusal. It reaches its verdict against a Record
an earlier Step produced, so a call has already gone out, which is exactly what makes it a halt
(ADR-0072): the Run is `failed`, it carries no `error_code`, and what the Steps before it did stands.
The one thing at a Requirement that *is* a Refusal is a predicate that cannot decide, which Refuses
wherever it stands (`predicate-type-mismatch`, ADR-0035).

**What a Requirement asks for is its predicate, so the predicate is where the reviewable fact
belongs.** A Requirement's whole content is on the line being read — a Step of this same file, one
field name, one operator, and no derived cell beside it (§8) — and a `require:` roots at any field the
Step it names projected (§3, §12), so which of two spellings an author writes is a live choice at
every Requirement. A Step whose command embeds the comparison, gated on a `require:` reading `exit_code`,
halts identically to one that projects the value and compares it — and its `require:` line states only
that a status was zero, the fact under review having been written into a quoted argument of an Opaque
request. That is the fault above read off the other axis and on the same artefact, the one whose whole
purpose is to be read before anything runs. It is the author's to avoid rather than something `check`
can refuse: both spellings are well-formed, and an Opaque request is by definition one `hyper` cannot
tell the two apart in (ADR-0122).

## `opaque` and `destroy`

An `opaque` Operation's effects are ones `hyper` cannot describe, so an `opaque` `destroy` cannot be
shown to touch only Assets and cannot be Bounded — it is unbounded blast radius in exactly those
words, and it is refused unless two separate opt-ins agree. The Target's declaration must opt in —
the artefact half, checked statically with no credential resolved (`opaque-destroy-not-granted`, §4).
The credential resolved for that Target at Run start must carry the same opt-in, or the Step Refuses —
the half no static check can perform, since no credential exists before then. A third requirement is
stated with the Bound below, where the reason for it is: such a Step must name the population it is
destroying. That one is not an opt-in and is not the `opaque` Step's alone — it holds on every
`destroy` (`destroy-unscoped`, §4, ADR-0085) — and it is stated there because the `opaque` Step is
where it does the most work. The two opt-ins are independent by design: a laptop's credential for a
Target can carry the opt-in while the credential
Actions holds for the same Target does not, which makes "CI may not do this" a fact about what the CI
credential holds rather than a belief about where the process runs — credentials resolve the same way
regardless of where `hyper` runs (ADR-0007).

The artefact half is a property of the Target and never of the pairing that took it. A declaration
carrying the opt-in admits an `opaque` `destroy` from every Definition bound to that Target, and the
second Definition to claim one widens nothing on the declaration's own line — its own claim is the
reviewed edit, named Operation by named Operation, which is where §3 puts a `destroy` claim's
granularity. That is the grain a Target *is*: the unit of both blast radius and credentials at once
(§2), and every authority fact about a (Definition, Target) pairing is derived from its two ends
rather than authored at one — a Step binds a pairing and grants it nothing (§8, ADR-0069). What
narrows the grant is a second class-local declaration — two names for the machine `hyper` runs on,
one opting in and one not, confining command-`destroy` authority to the Definitions that bind the
first (§3, ADR-0041). `opaque-destroy-not-granted` states both edits, since they differ in blast
radius and only one of them is on the line the row cites (ADR-0103).

## The Bound

A `destroy` Step's Bound is mandatory: an absent Bound means unbounded, and unbounded is refused
before anything runs rather than left unchecked (`bound-missing`, §4).

An `opaque` `destroy` Step is the one Step that carries no Bound, and writing one there is refused
(`bound-illegal`, §4). This is not the rule above softened. A Bound counts the Records an Expansion
resolved to, and a count of the calls an opaque Step made says nothing about what any of them did: the
only Bound a single command could carry is `1`, which would stand in the gutter and in `FLAGS` reading
*at most one thing will be destroyed* while `rm -rf /` is magnitude one. Truthful and still misleading
is the worse failure on the most severe Step the tool runs. Unbounded is the accurate word for it, and
§13 uses that word.

So does the review. Such a Step draws three flags — `DESTROY`, `OPAQUE` and `UNBOUNDED` — and the last
is implied by the first two on every such Step, having no other form to take. It renders regardless
(§12): a surface indexing what is unbounded, silent on the one Step where nothing can be bounded, is
omitting rather than economising.

**The argument is about opacity and not about severity, so it reaches an `opaque` `mutate` too — and
there it lands on the rendering rather than on the check.** A Bound written on one is truthful: it
counts Records, the Step mints them, and `check` accepts it (§4). It is also silent about everything
the flag beside it exists to say. `bound: 1` on a Step whose command reads `cat requests/pending >>
firewall/allow && : > requests/pending` is the record count, and the two firewall rules appended and
the file truncated are not in it. So the flag has a third form and renders regardless there as well,
on the second's own footing: implied by `mutate` and `opaque` together, which the gutter marks in
place, and cleared by no edit to `bound:`. What differs between the two Kinds is what may be written,
not what may be read off it — `bound-illegal` on the one where no honest count exists, and a flag on
the one where an honest count exists and bounds nothing (ADR-0121). A flag an author can clear by
writing a number that changes nothing is a mark that teaches the wrong edit, which is the one failure
this surface may not have.

**A `destroy` Step must carry an `over:` selector** (`destroy-unscoped`, §4). Without one it is
invoked once (§3), so it has no Expansion, no series to write a Tombstone under, and no declared
identity — a `destroy` carries no `record:` at all (§3, ADR-0037). It would reach the world and write
nothing whatever: no Record, no Tombstone, an empty identity set, and no row in `YOU DID THIS`, which
is the one thing an effectful path may not do. The requirement is not a Bound arriving under another
name (ADR-0053).

**None of that sentence is about opacity**, and the rule is stated here rather than scoped here: an
`http` `destroy` with no selector reaches the world and writes nothing by the same three steps. It is
one requirement on the Kind, and one code (ADR-0085).

What the `opaque` Step adds is the second thing the selector buys, which is why the requirement is
stated in this section at all. The population is authored literally, on lines the gutter annotates, so
a reviewer reads what is being destroyed rather than inferring it from a command; and `expanded_to`
records what the Step resolved to in Expansion order (§7), so *which three of the five* is legible
after a halt with no Bound anywhere claiming to have guarded it. On an `opaque` Step that is the only
account of the population there is. On every other `destroy` it is a second one beside the Bound.

```yaml
  - id: purge-releases
    definition: host-ops
    operation: destroy
    target: local
    over:
      values: [/srv/app/releases/r41, /srv/app/releases/r42]
    args:
      command: [rm, -rf, {item: $}]
```

Two Tombstones, each opening the series it ends (§7, ADR-0033), each named by a path a human
recognises. What stands in the Bound's place is therefore three things and not two: the two independent
opt-ins above, a Definition that named the Operation, and a population the author wrote down.

## Expansion

A selector's Expansion — the resolution of `over:` to the concrete Records a Step will act on — is
scoped by the Step's own Kind: a `read` Step may expand over Observations as well as Assets, and an
effectful Step may expand only over Assets (ADR-0027). A predicate reads the **head** version of each
series and no other: *any version* would have a `destroy` reach a thing for what it used to be, and
would make one artefact reach further every month the Store grows. A series whose head is a Tombstone
stands for nothing and is expanded over by neither, so what one Run destroyed the next does not reach
again and does not count against its Bound (§7) — which is that same rule applied to the one version
type that stands for nothing.

An `opaque` Step expands like any other, all three `over:` forms reading on it under the same Kind
rule. Nothing about opacity restricts which Records a selector may range over; what it restricts is
what may be concluded from counting them, which is the Bound's business above.

An Expansion is where a predicate meets a value, so it is where a value of the wrong type is found. It
Refuses (`predicate-type-mismatch`, §12) rather than excluding the Record quietly, which it can do
because an Expansion resolves before the Step's call goes out (§6, ADR-0035). Anything `hyper` did not create is therefore
reachable only by literal identifier — a `values:` list, occupying lines the gutter annotates and
counted by the Bound like any other selector (§12) — never by a selector ranging over it: a Record
`hyper` never created is not an Asset, and an effectful selector has nowhere else to reach.

The Tombstone rule reads on a `values:` list too, once a member has a series to have a head, and there
it is a `destroy`'s: on a `destroy` Step a member whose head is a Tombstone is dropped from the
Expansion, and a member naming no series at all is reached. What that states is that `hyper` drops what
it knows is gone and reaches what it has no record of — a member it never touched being in the second
class — so a `values:` list left standing in a Procedure that runs on a Cadence is self-limiting rather
than a Run that fails on a call the artefact never asked it to repeat.

A `mutate` reaches such a member instead, and the same sentence is the reason: a create over a
Tombstoned series is a call the artefact *is* asking for, and §6 skips a member only while its Asset
stands. Dropping it here would leave destroy-then-recreate reachable from a Step with no selector and
from nowhere else, which nothing states and ADR-0011 contradicts. This is ADR-0027's Kind-scoping
arriving at a case it did not enumerate rather than a second rule standing beside it.

It is the Store that shortens the list and never lengthens it, which is what lets §4 count the authored
length against the Bound offline.

## No bypass

Nothing overrides a Refusal at invocation time — no flag, no confirmation, no per-Run override of any
check in this chapter — and the only way past one is an edit to the authored artefact, put back
through review (ADR-0001).

## Refusal

A Refusal is a terminal Run outcome distinct from failure: a guardrail declined before any effect
reached the world, rather than the world resisting one. Every check this chapter and §4 state ends in a
Refusal when it does not pass — and since a Run re-runs every one of §4's checks at its start (§6), a
Refusal usually declines before any Step exists to have been declined. It is therefore a fact about the
Run: recorded in full on the Run's outcome and never on a Step's, and the Step it may cite is a position
in an artefact rather than something that ran (§7, ADR-0061). No Record is written for a Step it
stopped, since nothing happened to the world.

Distinct from failure in one further direction: a Run that lost the Store has not been declined by a
guardrail and is not a Refusal, because nothing anyone does is required to clear it (§7).

## Blast radius

A Bound counts Records; it does not weigh what happened to each one. It catches the runaway selector,
which is the failure it exists for, and says nothing about how severe a single, correctly-Bounded call
is. Nothing above reopens on that account — the two keys, the named-Operation requirement on
`destroy`, and the Definition review are what stand between an author and that outcome. Blast radius
measured in counts rather than severity is named here as the honest limit it is, carried forward to
§13.
