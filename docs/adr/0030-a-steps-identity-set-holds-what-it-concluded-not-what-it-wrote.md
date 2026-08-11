# A Step's identity set holds what it concluded, not what it wrote

Every Step's **Disposition** carries the Record identities that Step reached a recorded conclusion
about — what it projected from a response under `read` and `mutate`, and what it confirmed destroyed
under `destroy`. It is not the set of Record versions the Step wrote, and it is not everything the
Step's Expansion reached.

The reading a competent implementer arrives at unaided is *the Records this Step wrote*, because that
is the set a Step obviously produces and the one a file listing hands you for free. It is wrong, and it
is wrong in the direction that silently disables the feature the set exists for: a Record whose bytes
did not move mints no file, so under that reading an unchanged listing of five hundred Records has an
empty identity set, its digest never moves from the last Run's, and the Comparison can no longer tell a
Record that vanished from one that did not change. That is the hole the identity set was introduced to
close, and *what it wrote* reopens it while looking correct on every Run where something happened to
change.

We chose *concluded about* because it is the only definition that holds across all three Kinds. `read`
and `mutate` project a Record from a response, and the projection happens whether or not a version is
minted. `destroy` projects nothing at all — a `destroy` Operation carries no `record:` and declares no
identity, its Tombstone landing under the series its Expansion acted on — so *projected* alone does not
cover it, and *confirmed destroyed* is the same act in the one Kind that has no projection to name.

## Considered options

- **The Record versions the Step wrote.** The natural reading, and the one a file listing makes cheapest
  to implement. Rejected because a version is minted only where the bytes moved: the unchanged case is
  the case the set exists to distinguish, and this reading is empty in exactly that case.
- **Everything the Step's Expansion reached.** Attractive because it is knowable before the first call
  and is what a refused Step can honestly report. Rejected because it says nothing about what came back:
  a Step that expanded to five and got three answers would report five, which asserts three conclusions
  it never reached. That set is real and is held separately, as the selector's `expanded_to`.
- **A different definition per Kind.** Rejected as three rules where one will do, and because the
  Comparison reads the set without knowing which Kind wrote it — a set whose meaning depends on the
  Operation behind it is a set the reader has to resolve a Manifest to interpret.

## Consequences

- **A Step that concluded nothing carries no set at all**, rather than an empty one. `refused`,
  `skipped by condition` and `never reached` hold none; the last writes no file to hold one in.
- **`attempted, outcome unknown` holds a partial set and is honest about it being partial.** A destroy
  that confirmed three of five holds three, and the two unaccounted for are read as `expanded_to` minus
  the set. `hyper` does not name which of the two was in flight, the uncertainty attaching to the
  attempt rather than to the thing.
- **The identity set and the count of Records written are two different numbers**, and any rendering
  that shows both has to say which it is showing. They differ every time a Record comes back unchanged,
  which on a monitoring Cadence is nearly every Run.
- **A `read` Step's set is as load-bearing as an effectful one's.** The set is not an audit of what
  `hyper` changed; it is the evidence that `hyper` looked, and it is the only thing separating *nothing
  changed* from *nobody checked*.
