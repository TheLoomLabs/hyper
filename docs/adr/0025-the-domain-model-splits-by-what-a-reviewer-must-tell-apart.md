# The domain model splits by what a reviewer must tell apart

Three splits carry the load in `hyper`'s domain model, and none of them is the obvious reading of
Swamp's shape. **Observation and Asset are two Record types, never a status field on one** — they
diverge on retention, on what a change in one means versus the other, and on how much safety weight
each carries, and nothing converts one into the other. **A Record's identity excludes the Run that
wrote it** — identity is `(Target, Definition, name)`, so every execution of a Definition against a
Target writes into the same series rather than minting a fresh one, and a Run is recorded only as
**Provenance**, the account of which code produced a given version. **An Operation's Kind is authored
in its Manifest**, never derived from the Operation's name, its verb, or any other property `hyper`
could infer on its own.

A simpler model reads as available on all three points — one `Data` type with a status field, a Record
keyed by the execution that produced it, a Kind guessed from the Operation's name — and a future
reader who reaches for any of them will assume the split was not worth the extra vocabulary. It was.

We chose this because Swamp collapsed each of these into a single field and paid for it in patches.
Its one `Data` primitive needed a `lifetime` enum, a tombstone convention, and an auto-definitions
mechanism to recover a retention rule, a diff meaning, and a safety weight that Observation and Asset
make free consequences of the type itself — three patches around one conflation, where the split makes
the conflation impossible to reintroduce. Keying a Record by the Run that wrote it would mint a fresh
series on every execution and leave nothing to compare against,
which kills the retrospective Comparison before it exists — the destination this whole model serves.
And inferring Kind from a name is closed at the language level: Swamp derives its method kind from
the method name, so an Operation like `check_and_purge` gets whatever the heuristic decides, silently,
with no artefact recording that a judgment call was made at all.

## Considered options

- **One `Data` primitive with a status field**, matching Swamp exactly. Rejected: Swamp needed a
  `lifetime` enum, a tombstone convention, and an auto-definitions mechanism to claw back the
  retention, diff, and safety distinctions a type split gives away for free — three separate patches
  standing in for one type decision never made.
- **Record identity includes the Run.** The obvious first reading of "immutable, versioned" — rejected
  because the point of versioning is to compare one execution against the next, and a key that changes
  with every execution has no "next" to compare against; it only ever has a first.
- **Infer Kind from the Operation's name or verb**, Swamp's actual behaviour. Rejected because it
  makes blast-radius classification a heuristic nobody wrote down and nobody can override without
  renaming the Operation.

## Consequences

- **No Record ever changes type.** An Observation that turns out to describe something `hyper` also
  manages is not promoted to an Asset — adoption was declined as reconciliation entering through a
  side door, and stays open rather than becoming a quiet default.
- **Diffing is possible at all.** Because identity survives across Runs, the Comparison can say "this
  Asset changed since the last Run of this Procedure" rather than only "this Run produced these
  Assets."
- **A misdeclared Kind is a Manifest review problem, not a runtime one.** `hyper` acts on whatever Kind
  the Manifest states; nothing in the tool cross-checks it against the Operation's actual behaviour,
  because there is no oracle to check it against.
- **Folding `Opaque` into the Kind enum was never on the table.** An authored, closed Kind leaves room
  for a second, independent trait precisely because nothing about it needs deriving — and `opaque` plus
  `destroy` is the most dangerous thing `hyper` can express, so the one place a merger would have
  erased a flag is the one place that flag matters most.
- **Retention, diff meaning, and safety weight now read off the type**, not off a value a reviewer has
  to look up. A Comparison renders Observation and Asset changes as separate tables for exactly this
  reason: the type already carries the fact that would otherwise need a second glance.
