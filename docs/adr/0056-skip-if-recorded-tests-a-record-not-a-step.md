# `skip-if-recorded` tests a Record, not a Step

`skip-if-recorded` asks whether the Asset a call would produce still stands, and reads the head version
of the series `(Target, Definition, name)` identifies to answer (§12). An expanding Step has one such
series **per member of its Expansion**, so the test runs once per member: a member whose head stands is
skipped, a member whose head is a Tombstone runs, and a member naming no series runs. A Step may
therefore skip some members and call for others.

The reading an implementer reaches unaided is that the Step is the unit. §6 stated the skip in the
singular — *a Step skipped under `skip-if-recorded` is one whose Asset still stands* — a Disposition is
one value per Step, and none of the six is partial. Nothing said what to do when three members disagree,
so the unaided implementation picks a quantifier nothing states: skip while **all** stand, or while
**any** does.

It is wrong, and the cost is not a rendering. `skip-if-recorded` is what a Procedure under a Cadence
declares so a weekly Run does not rebuild what it built last week. Under a wholesale skip, an author who
adds a member to a reviewed `values:` list gets nothing — the members already standing carry the skip,
the new one is never created, on that Run and every later one, and the Run reports `completed` and exits
`0` (§13). A reviewed edit to an authored artefact with no effect and no rendering anywhere is the
failure ADR-0001 is built to make impossible, arriving through a Repeatability value rather than through
a bypass. The Step even looks busy: its `RECORDS` count is the whole population it skipped (§8).

Three further things follow from the test being per Record, and each of them is a place the unaided
reading takes the Step as the unit.

**The identity must resolve before the call.** The test reads the head of the series the call would
write under, so the name has to exist before the call it is deciding on. A template hole has that
property, resolving to an Operation input like any hole outside a Capability-relevant position (§12); so
does `$.command` on a `shell` Operation, which is in the response object precisely because it is a fact
about the call rather than about the answer (§3). A response path anywhere else names a value that
exists only after the call, which is a Manifest declaring a test it cannot perform —
`manifest-inconsistent` (§4). This needs no new syntax: the hole grammar is closed at one form and
already resolves to an input everywhere outside `host:` and `auth:`, and a path and a hole are told
apart by their first character. `hyper`'s own `mutate_skip_if_recorded` already satisfies the rule; the
worked third-party Manifest in §3 did not, and now writes `identity: "{name}"`.

**A population that stands by construction leaves nothing to decide.** An effectful Expansion reaches
only standing Assets, so a `skip-if-recorded` Step over `assets:` skips every member on every Run and
can never call. It is refused statically as `skip-if-recorded-unreachable` (§4), the forty-sixth
`error_code`. A `values:` list is the form that names a population `hyper` may not yet have built, which
is the population this value exists to fill in.

**The Tombstone-drop in Expansion is a `destroy`'s.** §5 dropped a Tombstone-headed `values:` member
from every Expansion, which contradicts §12 and ADR-0011 on an expanding Step: if the Expansion drops it
first, the skip test never sees it and destroy-then-recreate is reachable only from a Step with no
selector. §5's own justification — *a call the artefact never asked it to repeat* — is a `destroy`'s
argument and false of a create. A `mutate` reaches such a member and creates it again. This is
ADR-0027's Kind-scoping arriving at a case it did not enumerate.

## Considered options

- **Skip wholesale.** The Step is *skipped as already recorded* and no call goes out while any (or
  every) member stands. Rejected above: a member added to a reviewed list is never created, silently and
  permanently, and no surface says so.
- **Run wholesale.** Any member missing makes the whole Step call, for every member. Rejected because
  calling again over a standing Asset is precisely what the value exists to prevent, and `mutate` carries
  no idempotency guarantee — that is what `repeatable` means and it is a different value.
- **A seventh Disposition.** *Partially skipped*, carrying the split. Rejected because the Disposition
  never claimed a count: *ran* already covers a `read` Step that made five hundred calls, and the two
  existing values cover the range with no boundary case invented. The split is derivable — under this
  value a member that runs always mints a version, since a standing head would have skipped it and a
  Tombstoned head has no value to be unchanged from — and §7 refuses a second representation of a
  derivable fact.
- **Drop skipped members from the Expansion**, making the skip and the Tombstone-drop one mechanism.
  Rejected because it inverts §7's arithmetic: *a member present in `declared` and absent from
  `expanded_to` is one the Store already held a Tombstone for*. A member absent for standing and a member
  absent for being gone are opposite states, and the entry would no longer say which — truthful and still
  misleading, on the surface built to be read after a halt.
- **Hold only the members that ran in the identity set.** Rejected because §8 renders *concluded about n
  of m* as **unaccounted for** — a halt. Two members the Store answered a question about are the most
  accounted-for things on the page, and rendering them as `1 of 3` would report a halt that did not
  happen.

## Consequences

- **A mixed Step is *ran*; a Step where every member skipped is *skipped as already recorded*.** Six
  Dispositions stand. The choice costs nothing downstream: `skip-if-recorded` reads the Store's head
  version and never the Journal, so unlike run-once it consumes no Disposition, and the value is a fact
  for a reader rather than an input to a later Run.
- **`expanded_to` holds every member and the identity set holds every member.** `RECORDS` renders `n`
  rather than `n of m`, and the digest does not move as a `values:` list fills in one member at a time —
  which is the behaviour the digest exists for and would be lost if the set shrank to whatever ran.
- **A condition naming a mixed Step holds.** The Step wrote a Record in this Run (§6), so the skip does
  not propagate from a Step that acted — which is what a reader of the artefact would predict.
- **The value's honest limit is unchanged and now paid per member.** `skip-if-recorded` still trusts the
  record over the world (§13), so a member hand-deleted between Runs is skipped rather than rebuilt. What
  changes is that a member the artefact newly asks for is no longer hidden behind the members it already
  had.
