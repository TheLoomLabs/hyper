# The selector a `destroy` carries is required by the Kind, not by the Capability

**Every `destroy` Step must carry an `over:` selector**, whatever Capability its
Operation binds. The requirement ADR-0053 landed on the `opaque` `destroy` is
one requirement on the Kind, and `opaque-destroy-unscoped` is renamed
`destroy-unscoped` to say so.

## The problem this decides

ADR-0053's argument is three steps long and none of them mentions opacity:

> Without one it is invoked once (§3), so it has no Expansion, no series to
> write a Tombstone under, and no declared identity — a `destroy` carries no
> `record:` at all (§3, ADR-0037). It would reach the world and write nothing
> whatever: no Record, no Tombstone, an empty identity set, and no row in `YOU
> DID THIS`, which is the one thing an effectful path may not do.

Every clause of it is true of an `http` `destroy` with no `over:`. §3's *a Step
declaring no selector is invoked once* is a rule about Steps. ADR-0037's *a
`destroy` declares no `identity:`* is a rule about the Kind. §7's *what a
`destroy` writes is a Tombstone under the series its Expansion acted on* is a
rule about the Kind too. Nothing in the composition asks what the Operation
speaks.

The check was written the other way round. `opaque-destroy-unscoped` gates on
`op.IsShell`, because that is the shape the argument was noticed against — §5
states the requirement in the Bound section, beside the two opt-ins, and the
Bound section is about the `opaque` Step. So the guardrail was scoped by the
paragraph it happened to be written in.

What that leaves reachable is issue #157. A `destroy` Step against any `http`
Operation, carrying its mandatory Bound and no selector, passes `check` with no
problem row of any kind. The Run resolves the empty selector to a set of one
whose member has no name; the `destroy` concludes about that member, because a
`destroy` concludes about the Asset the Expansion acted on and never a
projection of the destroying response; and `minted` asks the Store for the Head
under an identity whose name is `""`. `store.seriesDir` reaches its own
`impossible` guard and the process dies — **after** the `DELETE` has been
answered `2xx`.

So the failure §5 predicted arrives, on a review-clean repository, as a Go panic
rather than as the silent gap: a destruction reaches the world, nothing records
it, and the Journal entry is left open for a later Run to reap as *died*.

## The decision

**The requirement is on the Kind. One check, one code, and the code is
`destroy-unscoped`.**

- It fires wherever `op.Kind == "destroy"` and the Step carries no `over:`,
  citing the Step as it always did. The `opaque` Step is refused by it exactly as
  before — the same line, the same field, the same argument — and is no longer
  the only Step refused by it.
- **The rename is the honest edit.** §12's rule for its own membership is that
  each member is one code because it is one check, and what names a Refusal is
  the check that declined. A code called `opaque-destroy-unscoped` firing mostly
  on Steps that are not opaque asserts something false on the row a reader is
  holding, and sends them to a Target's `opaque-destroy:` opt-in when the edit
  is an `over:` line.
- **The remedy is one edit for both**, which is what makes one code right here
  where ADR-0053 found reuse wrong. `bound-missing` was rejected there because a
  reader handed it would have added the one key the Step is forbidden to carry.
  `destroy-unscoped` demands the same key of every Step it fires on.
- **`store.seriesDir`'s `impossible` stays exactly as it is.** It is the guard
  that caught this, and the state it guards is now unreachable rather than
  tolerated. A Store invariant that holds is not softened because something
  upstream violated it.

## Considered options

- **Refuse at the Run instead of at `check`.** Rejected. A Run re-runs §4 in
  full before its first Step, so nothing is bought — and reached any later, the
  decline would be arriving after an effect, which §5 calls a halt and not a
  Refusal (ADR-0072). The guardrail belongs where it can still decline before
  anything runs.
- **Decide the Step is legal and give the Tombstone a name.** Rejected for want
  of a candidate. A `destroy` declares no `identity:` and carries no `record:`
  block, which is precisely what ADR-0037 fixes; the only names in reach are the
  Step's id and the argv, and a Tombstone under either is `hyper` naming a
  Record after something that is not the thing destroyed — the row with no
  meaning ADR-0053 already rejected one Capability over.
- **Keep `opaque-destroy-unscoped` and widen what it fires on.** The cheapest
  edit, and every landed golden would have stayed byte-identical. Rejected on
  the code's own name: it would say *opaque* on rows about Steps that are not,
  and §12's set is read by operators off a Refusal row rather than out of the
  specification.
- **Mint a second member beside it, split on `op.IsShell`.** Rejected: it is one
  check. Two codes would have a reader believe the two Steps fail differently,
  and the second code's whole content would be *and also when it is not opaque*
  — the moment it ran rather than the check that declined, which is the
  distinction §12 names outright.

## Consequences

- **ADR-0053 is amended in its title's direction and nowhere else.** *An
  `opaque` `destroy` names its population* remains true and remains the argument
  that produced the rule; what changes is that it was never only the `opaque`
  one's to name. Its code-selection bullet stands as written about
  `bound-missing`, and its rejection of an invented identity stands for every
  Capability now.
- **§5 keeps the requirement in the Bound section and states its extent there.**
  The paragraph is where the *reason* lives — the population is authored on
  lines the gutter annotates, and `expanded_to` records what the Step resolved
  to — and on an `opaque` Step that reason is the only account of the population
  there is. On every other `destroy` it is a second account beside the Bound.
  Location was never scope.
- **§12's closed set keeps its cardinality.** One member is renamed, none added
  and none removed, so `check/error-code-coverage` holds the same number of
  codes to the same rule.
- **One authoring shape stops being legal**: a `destroy` by reference — publish,
  then delete what was published, with the `record_id` read off the earlier
  Step. It gains an `over:` `values:` list naming what is being destroyed, which
  is the form ADR-0033 and issue #151 built for exactly this, and the Tombstone
  it writes now opens a series under a name a human recognises rather than under
  none.
- **The other authored route to a nameless identity closes in the same
  change.** An `over:` `values:` member that is an empty scalar carries a member
  whose name is `""`, and a `destroy`'s head lookup — *is this one already
  gone?* — reaches `store.seriesDir` with it before the first call goes out. It
  is refused at `check` under `schema-mismatch`, the code §4 already fires on a
  member that is not a bare scalar: the selector is there and the fault is the
  value. No closed set moves for it either. The two shapes are one sentence read
  twice — *what a `destroy`'s Expansion acts on has a name, or there is nothing
  to check* — and they are the whole of what reaches that guard from an
  artefact.
- **The `opaque` Step's three static requirements are untouched.**
  `opaque-destroy-not-granted`, `bound-illegal` and the selector are all still
  required of it, still checked at `check`, and `opaque-destroy-clean` passes
  unchanged.
