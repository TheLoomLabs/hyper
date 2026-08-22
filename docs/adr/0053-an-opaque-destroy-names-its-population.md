# An `opaque` `destroy` names its population

**An `opaque` `destroy` Step must carry an `over:` selector** (`opaque-destroy-unscoped`, §4). Without
one it reaches the world and writes nothing whatever: no Record, no Tombstone, an empty identity set,
and no row in `YOU DID THIS`.

_ADR-0085 amends this:_ every clause of the argument below is about the Kind and none of it is about
opacity, so the requirement holds on every `destroy` Step and the code is `destroy-unscoped`. What is
written here about the `opaque` Step stays true of it; what changes is that it was never only the
`opaque` one's to satisfy.

The reading a competent implementer reaches unaided is that nothing is wrong. §3 says a Step declaring
no `over:` is invoked once, which is an ordinary and common thing for a Step to be; §3 says a `destroy`
carries no `record:` at all (ADR-0037); and §7 says what a `destroy` writes is a Tombstone under the
series its Expansion acted on. Compose the three and a `destroy` with no Expansion has no series, no
declared identity and nothing to write — each step following from a sentence already in the corpus, and
the composition producing an effectful Step that leaves no trace of itself anywhere in the record.

That is the one thing an effectful path may not do. ADR-0037 made `record:` mandatory on a `mutate`
precisely on this argument — an effect `hyper` is accountable for that puts no row in `YOU DID THIS` —
and ADR-0033 made the same argument for the Tombstone a `destroy` writes against a foreign identifier.
Here it arrives on the most severe Step the tool runs, on the Capability whose reach no grant bounds,
and it arrives silently: nothing refuses, nothing fails, and the Run completes.

Requiring the selector fixes it at the place the review claim already lives. The population is authored
literally in the Procedure, on lines the gutter annotates, so a reviewer reads *what* is being destroyed
rather than inferring it from a command; `expanded_to` holds what the Step resolved to in Expansion
order (§7), so *which three of the five* is legible after a halt; and each member gets a Tombstone,
opening the series it ends (ADR-0033). The practical form writes itself — a `values:` list of the
identifiers, each wired into an argv position the executable is not (ADR-0051):

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

This is not a Bound arriving under another name. A Bound is a claim that at most *n* things will be
affected, and §5 refuses one here because a count of the commands says nothing about what any of them
did — `rm -rf /` is magnitude one. A selector claims nothing about magnitude at all; it says which
things the author meant, and it makes the Store able to record that the Step reached them. The two are
independent, which is why an `opaque` `destroy` now names a population it may still not count.

## Considered options

- **Accept it: a `destroy` with no selector runs once and records nothing.** The unaided reading.
  Rejected because it contradicts ADR-0037 one Kind over and does so on the Kind where the contradiction
  costs most. It would also make `destroy_once` unenforceable in the direction that matters — a
  run-once Step is refused on the Journal's evidence that it *ran* (§6), which survives, but nothing in
  the Store would ever say what it destroyed.
- **Give `destroy` an identity from `$.command`**, writing one Tombstone per destroy command. Rejected
  as a row with no meaning. The Tombstone would be named after the teardown command rather than the
  thing torn down, in a series nothing else ever writes to, and `YOU DID THIS` would report that a
  command is gone. It satisfies the letter of the rule it is trying to satisfy and none of its purpose.
- **Give `destroy` a `record:` after all**, exempting `opaque` from ADR-0037's prohibition. Rejected:
  a `destroy` declaring a projection declares an identity for a Record it does not mint, which is the
  reason the prohibition exists, and opacity is not a reason to invert it.
- **Remove `destroy` and `destroy_once` from the built-in roster.** Rejected because §5 has an entire
  section about what an `opaque` `destroy` requires, and the two independent opt-ins it describes exist
  for a Step this option would delete. The tool either supports destroying things by command or it does
  not; the honest version supports it and states the price.
- **A dedicated code, or `bound-missing` reused.** `bound-missing` was considered and rejected — it
  names an absent Bound, the one thing this Step is *forbidden* to carry, so a reader handed it would
  make the opposite edit. One new member instead, `opaque-destroy-unscoped`, sitting beside
  `opaque-destroy-not-granted` and `bound-illegal` where its check is stated.

## Consequences

- **§5's justification for `bound-illegal` loses a clause and keeps its argument.** It read that an
  opaque Step *expands over nothing and names no population*, which was already false — ADR-0045 has
  the built-in `read` running its Expansion serially — and is now false by rule. What survives is the
  half that was always load-bearing: a count is truthful and still misleading. A second sentence stands
  beside it now, that the population is authored and `expanded_to` records it, so the count is readable
  off the entry without a Bound claiming to have guarded it.
- **An `opaque` Step expands like any other**, all three `over:` forms reading on it under the same Kind
  rule (ADR-0027). Opacity restricts what may be concluded from counting an Expansion, never which
  Records a selector may range over.
- **The `404` has no analogue and needs none.** A `destroy` completes on `404` because an Asset already
  gone would otherwise halt its Step on every re-run forever (ADR-0050); no exit code means *already
  absent* in any vocabulary `hyper` knows. The selector closes the same trap from the other side: a
  `values:` member the Store holds a Tombstone for is dropped from the Expansion before the command goes
  out (§5), so the Step does not re-reach what it already ended.
- **Three things stand where the Bound does not**: the Target declaration's opt-in, the credential's
  opt-in, and a population the author wrote down. §5 said two.
