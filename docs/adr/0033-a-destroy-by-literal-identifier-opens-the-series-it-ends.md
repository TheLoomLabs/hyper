# A destroy by literal identifier opens the series it ends

A `destroy` Operation carries no `record:` and declares no identity, and what it writes is a Tombstone
under the series its Expansion acted on (§3, §7). An `over:` `values:` list is the one selector form that
acts on no series at all — it is a literal enumerated list authored in the Procedure, and it exists
precisely so that an effectful Step can reach infrastructure `hyper` never created (§12, ADR-0027). A
`destroy` over such a list therefore has nothing to write a Tombstone into and no identity to write one
under. **It opens one.** The `values:` member is the Record name, the series begins with the Tombstone as
its first version, and `hyper` is accountable for the destruction from the moment its effect reached the
thing.

The reading a competent implementer reaches unaided is that **there is nothing to write, so nothing is
written**: the Expansion resolved to a string rather than a series, no prior version exists to append to,
and the destruction is already recorded in the Journal, whose Disposition holds what the Step confirmed
destroyed (ADR-0030). Every one of those observations is true, and the conclusion is still wrong. It
makes the single most consequential act the tool performs the one act that leaves no row in `YOU DID
THIS` — the table built to answer *what did you do* — and it fails there silently, on a Run that
completes. `hyper`'s second clause is *nothing changes unseen*; a destruction the Comparison cannot
render is that clause failing on its hardest case. It is also the reverse of the gradient the record
should have: a `mutate` over the same list writes an Asset (ADR-0032), so the record would grow less
complete as the act grew more severe.

We chose to open the series because ADR-0032 already settled what accountability is earned by, one Kind
short of here. `hyper` answers for an Asset because **its own effect reached the thing**, not because it
created it — which is why an effectful Step naming a foreign identifier is not adoption. A `destroy` is
the sharpest effect there is, so the same test returns the same answer, and the alternative would have
the test hold for a `mutate` and lapse for a `destroy` with nothing to distinguish them but the
inconvenience of there being no series ready.

That the member is authored rather than projected is the price, and it is a real one. Every other Record
name in the system is a Manifest-declared field of an upstream response (§7); a `values:` member is the
first with an author for its origin. Where the author writes a spelling the Manifest's `identity:` path
would not have projected, a series opens under that spelling, the Tombstone lands in it, and a real Asset
series for the same resource stays standing and reads alive. `check` cannot catch this — it would need to
know what the API returns, which is the question §4 already declines to own — so it is stated as a limit
in §13 rather than defended by a rule that cannot be written.

## Considered options

- **Write nothing to the Store, leaving the destruction in the Journal alone.** The Journal genuinely
  holds it: a `destroy` Step's identity set is what it confirmed destroyed, and a literal serves that as
  well as a projected identity does. Rejected because the Journal and the Comparison answer different
  questions. The Journal says what a Run did; `YOU DID THIS` says what changed in the world between two
  Runs, and it is derived from Records. A destruction visible only to a reader who opens a Journal entry
  and reads a Disposition is not *nothing changes unseen* — it is a fact filed where the surface built to
  surface it will not look.
- **Refuse `values:` on a `destroy` Step**, leaving foreign infrastructure destroyable only by hand and
  adding a victim to §13's wall. Rejected on scope rather than on principle: deletion of existing
  infrastructure is named in the first line of what this tool is for, and §13's own sentence — a foreign
  resource is reachable by literal identifier *for as long as it stands* — presumes something eventually
  makes it stop standing. A wall that removes the tool's stated purpose is not a ceiling accepted
  deliberately; it is the feature declining to exist.
- **Refuse a `values:` member that names an existing Asset series of the same `(Target, Definition)`**,
  forcing `assets:` where a series is already there. Rejected because the collision it guards against is
  the correct behaviour: when the literal matches, the Tombstone lands in the series that thing already
  has, and `values:` and `assets:` produce an identical Store for an identical act. The rule would forbid
  a legitimate Step, and it would still need the unavailable knowledge to catch the case that actually
  hurts — a spelling that matches nothing.

## Consequences

- **The `values:` member is the Record name, with no branch on whether a series already exists.** Where it
  matches one, the Tombstone is an ordinary further version of it. Where it matches none, the series
  begins. One rule covers a resource `hyper` built and a resource it never saw, and the Store cannot tell
  which route reached an identical outcome — which is correct, because nothing distinguishes them.
- **The Expansion drops a member whose series head is already a Tombstone**, so §5's existing sentence
  reads on `values:` once a member has a series to have a head: what one Run destroyed the next does not
  reach again and does not count against its Bound. The rule this states is that `hyper` drops what it
  knows is gone and reaches what it has no record of, and a member it never touched is in the second
  class. A stale `values:` list is therefore self-limiting rather than a Run that fails on a 404 the
  artefact never asked for.
- **A Tombstone may be a series' first version and may carry no `fields`.** §7 has a Tombstone copy the
  previous Head's fields forward as the Asset's last known state; there is no previous Head, so the key is
  absent, and absence there means `hyper` destroyed this and never observed it. A Tombstone is the one
  version whose `fields` can be missing for no other reason.
- **`YOU DID THIS` renders it `destroyed`, with no marker and no fourth class.** A comparison reading
  *absent in the baseline, present in the subject* would render `created`, which is exactly backwards and
  is the unaided reading here too. The row needs no marker to be distinguishable: an ordinary `destroyed`
  row carries the last known state's fields and this one carries none, so *destroyed, and `hyper` never
  saw what it was* reads off the empty column rather than out of a second representation of it.
- **Nothing here softens the no-adoption rule.** An Asset opened by a Tombstone is a thing `hyper`
  destroyed, which is a fact about what happened. Adoption is `hyper` deciding that a Record it only ever
  observed is now one it answers for, which is a claim about the past, and no Step in this decision makes
  one (ADR-0010, ADR-0032).
