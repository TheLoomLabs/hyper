# `hyper` has no query language

The Store is read through a closed set of typed parameters — `records(...)`, `runs(...)`, and the
Comparison — and through nothing else. There is no expression language, no predicate dialect, and no
SQL passthrough. A caller wanting an arbitrary predicate takes the rows and filters them itself.

We chose this because the expression language is a surface that grows and never shrinks, and nothing
here earns one. Swamp's CEL-over-a-catalog with a field allowlist is the strongest version of the
alternative, and its best property — an unknown identifier producing an error that lists the valid
vocabulary — is available more cheaply from typed parameters, which state the vocabulary before the
call rather than after a failed one. The rows come back as NDJSON, and an agent filtering them with
its own tools is better at arbitrary predicates than any dialect we would ship.

## Consequences

- **Some questions become two steps.** "Which Assets in prod are tagged `X`" is a filter on the
  returned rows rather than on the query, which is a real ergonomic cost and the honest price of the
  decision.
- **CLI and MCP parity is trivial**, since both surfaces carry the same parameters and neither has a
  dialect the other must match.
- **Swamp's implicit history opt-in is not inherited.** Naming `version` or `isLatest` in a predicate
  silently opens history there, so the same predicate means different things depending on which
  fields it happens to mention. `hyper` takes an explicit `history` boolean defaulting to Head-only:
  a surface an AI drives should have no behaviour that depends on what a parameter is named.
- **This is not the argument that rejected an evaluator for security decisions.** That one was
  categorical — a grammar that does not exist cannot grow into an authority-widening one. A query
  language would be permitted here and is simply not worth its weight.
