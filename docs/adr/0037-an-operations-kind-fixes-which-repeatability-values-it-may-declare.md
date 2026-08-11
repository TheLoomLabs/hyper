# An Operation's Kind fixes which Repeatability values it may declare

Two of Repeatability's three values decide by reading a projection. `skip-if-recorded` reads the head
version of the Record series an Operation would produce and skips while it stands; run-once reads the
Journal instead, and is the only one of the three that needs nothing projected. So the question *which
values may an Operation declare* is answered by *what does this Operation project* — and that is fixed
by its Kind. **`record:` is mandatory on a `read` and on a `mutate` and forbidden on a `destroy`, and
`skip-if-recorded` is a `mutate`-only value.** A `read`'s Repeatability is `repeatable`, written or
omitted; run-once is the undeclared default on the two effectful Kinds and nowhere else.

The reading a competent implementer reaches unaided is that Repeatability and Kind are orthogonal —
three values against three Kinds, nine combinations, each meaning something. It is what §12 looked like
before this decision, and it survives contact with the corpus because the combination that breaks it is
never written down: the canonical Manifest's `delete_dns_record` declares `repeatable`, and no `destroy`
under `skip-if-recorded` appears anywhere in fourteen sections. What that combination would mean is
worse than undefined in both directions it can be read:

- **Literally** — *skip while the Asset it would produce still stands* — the head a `destroy` Step finds
  is the live Asset it exists to remove, so the Step skips exactly what it was authored to destroy and
  runs only against what is already gone. The value inverts.
- **Charitably** — *skip what is already destroyed* — it is a no-op. §5 already drops a series whose
  head is a Tombstone from every Expansion, selectors and `values:` members alike. The value would name
  a test that has already run, on the surface where a reviewer is deciding whether a `destroy` is safe.

The same hole is on `read` and it is the more dangerous one, because it never Refuses. A `read` projects
an Observation, and after ADR-0032 no `destroy` can ever write a Tombstone into an Observation series —
a Definition observes or effects, never both — so once a read's head exists it stands forever. A nightly
uptime check declaring `skip-if-recorded` therefore reads the world once, skips every occurrence after,
and reports `completed` and exits 0 each time. That is *nothing changes unseen* failing silently and
permanently, on the workload §0 opens the specification with, and no rule in the corpus stopped an
author writing it.

Making `record:` mandatory on a `mutate` is the premise the rest of this rests on, and it is worth on
its own account. A `mutate` projecting nothing performs an effect `hyper` is accountable for and leaves
no row in `YOU DID THIS`. That is ADR-0033's argument one Kind over: the objection there was that the
most consequential act the tool performs would be the one act leaving no row, and it does not weaken
because the act is creation rather than destruction. Forbidding `record:` on a `destroy` is the same
sentence read backwards — a projection there would declare an identity for a Record the Operation does
not mint, §3 having already said what a `destroy` writes and under whose name.

## Considered options

- **Leave the combinations undefined.** Rejected: `hyper` has no undefined behaviour to leave them in.
  Every value an artefact can carry is either refused at load or means something at Run time, and a
  Provider author writing `skip-if-recorded` on a `read` would get the silent-forever skip above with
  nothing declining and nothing rendered.
- **Give `skip-if-recorded` a second meaning on `destroy`** — *skip if already a Tombstone*. Rejected on
  the charitable reading above: it is a no-op, since §5 performs it at Expansion. Beyond being useless
  it would make one word name two different tests according to the Kind beside it, which is the second
  representation of one fact this specification has refused everywhere else, arriving in the one place a
  reviewer is reading to decide whether a `destroy` is bounded.
- **Ignore the value where it cannot apply.** Rejected as the same failure `predicate-type-mismatch`
  exists to prevent (ADR-0035): a declaration that quietly does nothing is indistinguishable, on every
  surface the tool has, from one that did what it says.
- **Let a `mutate` omit `record:`.** Rejected above. The tempting case is an API returning `204 No
  Content`, and the answer there is that the Provider author declares what the Operation affected from
  what it was given — an Operation `hyper` cannot describe the effects of is what `opaque` is for, and
  it still owes a Record.
- **Keep run-once as the default on `read` too.** Rejected. Run-once's own justification is *an effect
  nobody vouched for is not repeated on a guess*, and a `read` performs no effect, so the sentence is
  vacuous there while the consequence is not: `hyper list-vms` would work once and Refuse forever after,
  with no Cadence anywhere for ADR-0038's check to catch it.
- **Make run-once illegal on `read` and require `repeatability: repeatable` to be written.** Rejected as
  a field whose only legal value is fixed, which is a thing written to mean nothing — §3's objection to
  an empty `fields:` mapping, and it would put that noise on the commonest Kind in the tool.

## Consequences

- **Three new shapes of `manifest-inconsistent`, and no new `error_code`** (§4): a `read` or `mutate`
  carrying no `record:`, a `destroy` carrying one, and `skip-if-recorded` outside `mutate`. The code goes
  from five shapes to eight. `name-mismatch` was kept out of `kind-mismatch` because one code for both
  would leave a reader unable to tell which of two files to edit; that argument does not reach here,
  where all three point at one file, one Operation, and two adjacent keys.
- **A `read`'s Repeatability is `repeatable`, by omission or written out.** Run-once has no spelling
  (§3), so it is not merely defaulted away on a `read` — it is inexpressible there.
- **The default is now read off a declared Kind rather than being one value.** This is not inference in
  the sense ADR-0025 forbids: the Kind is authored and never derived, and reading another fact off it is
  what §4 already does for a Bound, mandatory on a `destroy` Step and absent from a `read` Step
  entirely, *having nothing for one to guard*.
- **ADR-0011's destroy-then-recreate case is untouched.** It rests on `skip-if-recorded` reading a head
  that is a Tombstone, which is a `mutate` reading the series a `destroy` closed — both Kinds behaving
  as they do here, and the value staying where it was legal all along.
- **Nothing in the model now depends on a Record type a Kind cannot produce.** §3 fixes the projection
  from the Kind, §12 fixes the Repeatability from the projection, and §5 already scoped Expansion by
  Kind (ADR-0027). The three read as one rule applied at three positions.
