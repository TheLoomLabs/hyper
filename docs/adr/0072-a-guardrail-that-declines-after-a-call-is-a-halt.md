# A guardrail that declines after a call is a halt, never a Refusal

A Refusal declines before any effect reached the world. That is not a description of most Refusals, it
is the property that makes one — so a check reaching its verdict after a call has gone out does not
Refuse, whatever it found. It halts, carries no `error_code`, and leaves the Step's Disposition saying
what the Step did. Where one check can reach its verdict at either moment, the moment decides which
outcome it gets, and the check's `error_code` names it only on the side that declines.

`hyper` has asserted this three times and enforced it nowhere. `CONTEXT.md` defines a Refusal as *a
guardrail declining before any effect reached the world*; ADR-0061 sharpens it to *a fact about the Run
rather than about a Step*; §12 defines exit `77` as *a guardrail declined before any effect reached the
world*. All three are statements about what a Refusal is, and none of them is a rule anything checks a
new site against.

One site of one code broke all three. §7 stated the Store's collision test at the **write** — *writing a
Record whose identity collides case-insensitively with one already in the Store is a Refusal* — and a
write is downstream of the call that produced the name. A `mutate` calls, the world is touched, the
response is projected, the projected identity folds onto a `foo` some earlier Run wrote, and `hyper`
Refuses. It was the only post-call member of a forty-seven-member set, and the only one of
`record-identity-collision`'s four sites on the far side of a call.

**It was also the option ADR-0070 rejected, running in production prose.** Deciding the sibling case one
ticket earlier, ADR-0070 weighed *refuse per member at its turn* and rejected it: *"a member's turn comes
after the member before it has run, so declining there declines after an effect — and refused carries no
identity set (§7), so the Step would render `–` and the write that did happen would vanish from the
surface."* The Store site refuses per member at its turn. Only the comparand differs — the sibling set
there, the Store here — and the consequence lands identically. Three `mutate` members, serial: the first
projects `alpha` and writes a version, the second projects `Foo` against a standing `foo`, and the Step's
Disposition is *refused*, carrying no identity set, rendering `–` on a surface whose `expanded_to` says
three. *Concluded about nothing*, over a Run that wrote a version and created two resources.

The reason ADR-0070 did not catch it is worth recording, because it is the shape this map keeps finding:
the ticket that settled the fault had no reason to visit the site. It needed the Expansion-internal
collision to decline before any effect, established that it did, and never asked what the Store's own
site does with the same fault. A rule stated three times and applied case by case is a rule that holds
wherever somebody thought to apply it.

## Considered options

- **Widen the Refusal to admit a post-effect case.** Rejected. It costs the definition in `CONTEXT.md`,
  ADR-0061's *never on a Step's*, and §12's definition of `77`, and it buys nothing: the fact still has
  to be recorded, and a halt records it better. A Refusal carries no identity set, so the writes that did
  happen vanish; *ran* carries one, so the entry says expanded to three and concluded about one and the
  arithmetic stays true. It would also leave `77` promising *a verbatim retry refuses identically* over a
  retry that makes the call, creates another resource, and only then refuses — identical in outcome and
  not in cost, which is the distinction the code exists to carry.
- **Defend the asymmetry on the remedy.** The Store collision is cleared by a Manifest identity change,
  which is a reviewed edit — and ADR-0061's line between `75` and `77` is *whether an act is required*.
  Rejected: that line sorts stops that have already been classified, it does not classify them. A halt is
  `failed` at `1` and also lies behind an edit; the Journal names the identity and the colliding name
  either way. Nothing an author needs is lost by halting, and the reader holding `–` over a Run that
  wrote is told something false.
- **Move the whole Store site to a halt.** Rejected as giving up ground already held. Where `identity:`
  resolves before the call — a template hole, a `shell` Operation's `$.command`, an authored `values:`
  member — `hyper` holds the name before it dispatches anything and can read the Store at Expansion, so
  the collision is refusable with nothing touched. Halting there would charge an effect for a fault a
  Store read decides, against the standing preference that everything decidable before the call is
  decided before the call.
- **Split by Kind**, letting a `read` keep Refusing since its collision creates no resource to orphan.
  Rejected on ADR-0070's own precedent — *a `read` collapses identically and touches nothing while doing
  it, so the rule turns on Kind nowhere* — and because the criterion is *the call went out*, not *a
  resource exists*. That is why a polling `until:` halts on a `read` too (ADR-0035). Splitting here would
  make *effect reached the world* mean one thing in §5 and another in §7, and hand the reader a table
  where there was a sentence.
- **File the four-way order at Expansion as its own ticket.** Rejected: its whole content is one
  sentence, in a section this decision already edits.
- **An ADR on the Store site rather than on the rule.** Rejected. What is hard to reverse is not what one
  site does but that the definition is enforced — the next site to arrive is decided by the rule rather
  than by another ticket noticing.

## Consequences

- **The Store comparand splits by when the identity resolves, and both halves are `record-identity-collision`.**
  Pre-call, the Store is read at §6's Expansion beside the members' comparison against each other and a
  collision Refuses with nothing touched. Response-path, the Run halts. §7 states the check and §6 states
  the moment, on the rule `bound-exceeded` and `predicate-type-mismatch` already carry across two sections
  — *what names a Refusal is the check that declined, never the moment it ran* — which this code was
  already carrying across four.
- **No closed set moves.** Forty-seven `error_code` members, four sites, seven Dispositions, seven exit
  codes, thirty-seven glossary terms. `CONTEXT.md` is untouched because its definition is being enforced
  rather than amended. The one edit to §12's membership prose is a clause: the sites requiring a Step are
  now *at its Expansion site* rather than *at its Store and Expansion sites*, both comparands having
  moved to one moment.
- **§12 gains a property of the whole set.** Every member declines before a call goes out. It is now
  stated as a fact about the set rather than left to be observed of its members, which is what decides a
  forty-eighth without another ticket.
- **The pre-call half reaches two Steps the sibling test cannot.** A Step carrying no `over:` resolves no
  selector and holds a set of one — vacuous against itself, not against the Store. And a `destroy` by
  literal identifier: a `values:` member authored `Foo` against a standing `foo` is not dropped by §5's
  head lookup, the names not being equal, and would otherwise open a colliding series with the call gone
  out. ADR-0070's *a `destroy` cannot reach the case* was true of the sibling comparand and is not true
  of this one.
- **`series` cardinality is named as a third comparand at the halt, and has no pre-call half at all.**
  Two Records projected out of one response that are one identity are not Expansion members — a `series`
  response is one call the Expansion resolved to — and a `series` Operation reads its identities from a
  response by construction, so every such collision halts. Whatever held the identity first keeps it, and
  each comparand supplies the order: Expansion order across an Expansion, the collection's order across
  one `series` response, and nothing to decide against the Store, whose series was written by an earlier
  Run.
- **Expansion has four checks and a stated order**, causal rather than arbitrary: a predicate resolves the
  set, the set has the count the Bound is read against, and the identities are projected off the members
  the set holds. The sibling collision is named before the Store's, being reproducible from the artefact
  alone and therefore pointing at an edit with no Store in hand. A Refusal holds exactly one member
  outside the two phases §7 names, so the order is required rather than cosmetic.
- **The halt names both spellings verbatim.** `Foo` beside `foo` is the whole content of the fault and a
  folded report says nothing. Which Run wrote the standing series is not in the message: the Head is
  derived by listing (§7), so it is one directory listing away and freezing it would store a second
  representation of a derived fact.
- **This case moves from `refused`/`77` to `failed`/`1`**, which is what `77`'s own definition asked for
  all along.
- **§13 carries the Store comparand as a separate limit**, not folded into the existing one. They share a
  shape and not a price: the existing paragraph closes on *it costs one resource once*, which stays exactly
  true of a collision between things one Run produced. The Store's collides with something that stays
  standing, and nothing clears or remembers it — the Store is append-only and a Run reads no earlier Run's
  halt — so every Run reaches the same member, makes the same call, and orphans another resource. Under a
  Cadence that is a standing leak rather than an incident, at the multiplier §5 has always said a Cadence
  is. It is loud on every occurrence, which is what keeps it survivable.
