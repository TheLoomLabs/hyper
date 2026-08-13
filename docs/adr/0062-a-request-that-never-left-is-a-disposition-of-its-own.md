# A request that never left is a Disposition of its own

An effectful Step whose request provably never left carries a seventh Disposition, **attempted, world
untouched** — a refused connection, a name that did not resolve, a handshake that failed, and one
Capability over a child that could not be started at all. It is **not Repeatability evidence**, it
concludes about nothing and carries no identity set, and its extent is exactly the transport class
ADR-0018 already fixed.

The reading a competent implementer reaches unaided is *attempted, outcome unknown*: the Step was
reached, it did not succeed, and that value is the one the six hold for a Step that failed without an
answer. It is wrong on the one thing that value exists to say. §7 states its content plainly — *how many
times `hyper` may have touched the world* — and these three failures are the exact class ADR-0018
retries **because** `hyper` is certain the world was not touched. Recording them there writes a doubt
`hyper` does not have onto the record that exists to carry doubt, and §6 then refuses a run-once Step on
it: a typo in a hostname bricks the Step permanently, with no bypass (ADR-0001).

The other five fit no better. *ran* means the outcome came back and it did not. *refused* is a guardrail
declining, and a host that will not answer is the world — which is why the Run is `failed` at `1` and
not `refused`. *never reached* means the Run ended before the Step. Both skips say a test was run and a
call was deliberately not made.

So the set grows, for the first time. It is not a case the specification failed to close: it is a
partition ADR-0018 has drawn since long before there was a Disposition to hang it on, surfacing on the
record rather than being invented for it. The set is closed against authors adding to it, which is not
the same as closed against the specification naming a state of the world it had not written down.

**The extent is ADR-0018's and is not re-derived.** A **connect timeout is outside it**, and a reader
will object that no request can have been serialised onto a connection that never established. ADR-0018
declines to retry one all the same, and two classifications of one event that must be kept in agreement
forever is worse than one boundary drawn slightly conservatively. `hyper` does not assert a certainty on
the record that it refuses to act on in its own classifier. The three members therefore become normative
in §12 as the **value's boundary**, and are written on no Step file: a reader learns which failures carry
the value, and an entry never says which one this was.

**It is effectful-only, and only where no call reached the world.** On a `read` a non-answer is the
answer (ADR-0050) — `uptime` records *down* against a host that answers nothing, so the Step minted an
Observation and *ran*, whose definition widens from *its outcome came back* to reaching a conclusion
`hyper` recorded, which the projection-failure case already strained. And a `destroy` that confirmed
three of five before the fourth connection was refused is *ran*: the value is the one where **no** call
this Step made reached the world, on the shape *skipped as already recorded* already carries. That is
what keeps *world untouched* literally true, which is what makes the run-once exclusion safe to state as
a fact about the value rather than as a fact about a set.

## Considered options

- **Widen *attempted, outcome unknown*.** The unaided reading, rejected above. The DNS typo is the case
  that decides it: an artefact edit is the only exit, and while a typo does invite one, a firewall that
  lapsed for ten minutes does not — there the artefact is correct and a correct Procedure is left
  permanently un-runnable with nothing to edit.
- **Widen *ran*** to *invoked, and `hyper` knows what it did*. Coherent — `hyper` does know: nothing.
  Rejected because *ran* is Repeatability evidence, so this reinstates the brick through a second door
  unless §6's rule stops reading values and starts reading identity sets, which is a larger change with
  a worse blast radius bought to avoid a smaller one.
- **Widen *refused*.** Rejected on the outcome triple: `refused` is a guardrail declining and `failed` is
  the world resisting, so this puts a Step-level `refused` inside a Run that is `failed`.
- **A wider *nothing was sent* class** admitting connect timeouts. Rejected above: it forks ADR-0018's
  classifier for an accuracy gain in one case.
- **Record which of the three it was**, promoting ADR-0018's class from a boundary to a member on
  `answered`. Rejected on the ground ADR-0050 rejected an `error` member: the catch-all the record's
  shape exists to close, arriving through the Journal instead of the response object. The host is
  already on the entry and the Disposition already says the request never left; §13 carries what that
  costs, and the fix if it bites is an enumeration ADR-0018 has already written.
- **`0 of m` in the Step table** rather than `–`. Rejected: `n of m` means *unaccounted for*, and the
  whole content of this Disposition is that nothing is unaccounted for. Rendering the safest state in
  the tool through the column form reserved for doubt inverts it.

## Consequences

- **A hostname typo is recoverable and a lapsed firewall is survivable.** §6 names the exclusion rather
  than leaving it as an absence from a whitelist — the shape of silence that let this hole sit unnoticed
  through fourteen sections.
- **§12 gains its first member.** Five earlier rounds closed a case rather than growing a set; each found
  that an existing value already answered. None did here.
- **Three Dispositions carry an identity set and four do not** (§7), and §8's `–` gains a fourth member
  beside *refused*, *skipped by condition* and *never reached*. §7 read *four and two* before this,
  which its own enumeration beneath already contradicted; the enumeration was right and the count was
  a slip, corrected in the same edit.
- **`answered` is reworded, not extended.** *Answered anything but `2xx`* does not reach a call that
  answered nothing; the key is written with the host alone, or with the command alone on a `shell` Step
  whose child never started, and no member is added to it.
- **`hyper` records that the request never left and never why**, which §13 carries as a named cost.
- **Nothing moves in the outcome triple or the exit codes.** The Run is `failed` at `1` — the world
  resisted. `75` is a Run that lost the Store, and this Run lost a host.
- **§4 gains nothing.** Nothing is declared, so there is nothing to check offline.
