# An Expansion's members are one Record identity each

Every member of a Step's Expansion is a call, and each must project a Record identity no other member of
that Expansion projects. Two members that are one identity under the case fold §7 applies are
`record-identity-collision` (§12) — refused before anything is touched wherever the identity resolves
before the call, and halting the Run at the projection where it does not. The boundary is one Step's
Expansion and nothing wider.

The reading an implementer reaches unaided is that there is nothing here to decide. §1 states that a
Record's identity excludes the Run that wrote it, so *a Definition invoked twice against the same Target
writes a further version into one series rather than starting a new one*; §3 blesses the same shape one
level down, *two Steps running the same argv against one Definition and Target write two versions of one
series*. An implementer holding those two sentences watches three members of one Expansion write three
versions of one series and concludes the Store is behaving exactly as specified — because it is. What
makes it a fault is an arithmetic two sections away: §7's identity set is a set, `expanded_to` beside it
is a sequence, and §8 subtracts one from the other. Nobody reaches that unaided, and no example in the
corpus showed it, because the three Kinds fail differently enough to hide it.

**A `destroy` cannot reach the case.** It projects nothing and declares no identity (§3), and its
Tombstone is written under the Asset's own identity (§7, ADR-0033), so the member *is* the name. The
`values:` duplicate refused at load (§3) is the only way two members there could ever be one name, and it
was already refused — on this argument, one Run earlier and against a list rather than a projection.

**A `mutate` under `repeatable` reports a halt that did not happen.** Three calls write three versions of
one series, the identity set holds one member where `expanded_to` holds three, and §8 renders `1 of 3` —
*two unaccounted for*, the phrase reserved for a call that may have reached the world, on the surface
built to be read after a halt. All three calls certainly landed. The Comparison is worse than the Run
page: its rows come from the identity sets (ADR-0058), so three effects render as one row and two of them
are invisible, which is the second clause of the thesis running backwards.

**Under `skip-if-recorded` it is coherent and still wrong.** The identity resolves before the call
(ADR-0056), so the first member runs, the rest find a head standing and skip, and every member is
concluded about — `RECORDS` renders `1`, the Run reports `completed`, exit `0`. A reviewed artefact asked
for three things and got one, and nothing anywhere says so. That is ADR-0056's own failure mode reached
by a different road, and it is the case that decides this cannot be left as a stated consequence.

**A `read` collapses identically** and touches nothing while doing it, so the rule turns on Kind nowhere
— which is what §6 already says of the projection failure beside it: *nothing in it turns on Kind, one
response projecting the same way whichever Kind read it*.

## Considered options

- **State it as a consequence and check nothing.** Rejected: the damage is not a missing oracle but a
  surface asserting doubt about calls that certainly landed, and under `skip-if-recorded` an artefact
  silently doing a third of what it says. The precedent for a stated limit here is the Tombstone-spelling
  case (§13), where catching it *would* need to know what the API returns. This does not: where the
  identity resolves before the call, `hyper` holds every name before it calls anything.
- **The member must reach the identity.** A wiring rule requiring the Operation's `identity:` to depend on
  the Expansion member. Rejected as both too strong and too weak. It cannot be stated where `identity:` is
  a response path, since `$.id` syntactically depends on nothing — so it would forbid an ordinary `create`
  under any Expansion. And it is not the property the arithmetic needs:
  `mutate_skip_if_recorded` tells its members apart perfectly with an `identity: $.command` that is not
  the member, while `{item: $.id}` over an `assets:` selector reaches the identity from the member and
  still collides where two Assets hold one value in that field. It survives inside the chosen rule as a
  sound but incomplete static detector, which is what §4 fires on.
- **Require every expanding effectful Step's `identity:` to resolve before the call.** Extends
  ADR-0056's requirement from one Repeatability value to every Expansion, making the whole fault
  pre-call and every site a Refusal, with no halt anywhere. Genuinely tempting — everything offline is
  this spec's standing preference. Rejected because it charges a Manifest-shaped price for a Step-shaped
  mistake: a correct `create` Operation whose API mints its own identity becomes unusable under any
  Expansion because of what a different, careless Step might do with it. §12 already carries *an API that
  mints its own identity* as a named victim of the closed sets; widening that to every Expansion buys
  determinism the layered rule already gets for every case a file can decide.
- **Refuse per member at its turn.** Rejected because a member's turn comes after the member before it
  has run, so declining there declines after an effect — and *refused* carries no identity set (§7), so
  the Step would render `–` and the write that did happen would vanish from the surface. Testing the whole
  resolved set at Expansion keeps the Refusal meaning what §1 says it means.
- **Write the colliding member as a further version of the series it collided with.** Rejected: it puts
  the later member's resource on the head with the earlier member's beneath it, disguising the fault as an
  ordinary update, and the cell still reads `1 of 3` because the identity set is a set. The collapse would
  be performed once and then complained about.
- **Mint a new `error_code`.** Rejected. §12's rule is that one check is one code however many moments it
  fires at, and the check that declines here is the one already named: two things that must be distinct
  identities are one. The standing objection to reuse — that a shared code points a reader at the wrong
  edit — does not apply, because this code already spans two different edits: the Store site is remedied
  by a Manifest identity change and the authored `values:` site by editing the Procedure. It has never
  promised which file.

## Consequences

- **`record-identity-collision` is stated at four sites and the closed set does not move.** §3 at load on
  an authored `values:` duplicate, §4 at load on the wiring, §6 at Expansion over the resolved set, §7 at
  the Store. Its §12 definition widens from the Store's site — *a Record identity colliding
  case-insensitively with one already written* — to the invariant it was always naming. That is
  ADR-0067's shape: the name already meant the general fact, and the sites arriving under it were inside
  what it meant.
- **The offline half grows again and the run-time half is not new machinery.** §4 refuses where the
  member count is authored and the identity cannot vary — no Store, no credential, no response. §6's
  Expansion Refusal sits beside `bound-exceeded` and `predicate-type-mismatch`, and its halt is ADR-0035's
  split unchanged: a fault found before the call Refuses, one found in the response halts and carries no
  `error_code`.
- **The `n of m` cell stops lying without being changed.** Nothing in §7 or §8 moves. The rendering was
  always correct about what the entry held; what was wrong was that the entry could hold a halt nobody
  performed. Four closed sets are in reach of this decision and none of them moves.
- **The earliest member in Expansion order keeps the identity.** On an effectful Expansion this is not a
  choice — it is serial, and the earlier version was written before the later member was called. Taking
  the same rule on a concurrent `read`, after the drain, keeps which Observation was recorded a fact about
  the Expansion order rather than about a completion order nothing derives from.
- **One resource can be created and not recorded, once per collision.** The colliding member's call has
  already gone out and has no name of its own; nothing is written and the Run halts. §13 carries it as the
  Orphaned Asset's hazard without the report, and the Journal names the member and the identity, so what
  is lost is a Record and never the fact that it is missing.
- **`skip-if-recorded`'s per-member timing is untouched.** Whether two members are one identity is a fact
  about the set of names and is settled at Expansion; whether each name's head still stands is a question
  about that name and is still asked at that member's turn (ADR-0056). The two tests read the same names
  at different moments and are not competing for one.
