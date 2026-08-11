# A predicate is evaluated against one instant, fixed at the Run's start

`older_than` and `newer_than` compare a projected timestamp against *now*, and every other operator in
§12's set compares two values that are both already in hand. These two are the only place a predicate
reads something no artefact wrote, so what *now* means is a decision rather than an implementation
detail. **`hyper` reads the clock once, at the start of the invocation, and every predicate in that
invocation is evaluated against that one instant** — every Step, every nested Procedure, every
selector, condition and polling `until:`. For a Run it is the `started_at` already on `run.json` (§7),
used verbatim rather than approximated, so the instant that decided what a `destroy` reached is a fact
the Store already holds and no key is added to hold it.

The reading a competent implementer reaches unaided is **the wall clock at evaluation**: the predicate
is a function over a value, the value arrives, the clock is read. It is what every language's date
library invites, it costs nothing to write, and it is invisible in every test that finishes in under a
second. It is wrong because it makes **how long the earlier Steps took** part of what a later Step
reaches. A polling Pattern on Step 3 that waits forty minutes for a VM to come up moves the boundary
`older_than: 14d` draws for the `destroy` at Step 40; so does a retry, a slow API, and a runner under
load. §3 states that a Pattern may never change the number of Records an Operation affects, and
ADR-0018 confines retry to provably pre-send failures for the same reason — that `hyper`'s own
behaviour must not change how many times the world is touched. Reading the clock late is that rule
failing on an axis nobody had looked at: the Pattern does not touch an extra Record itself, it moves
the set a later Step is allowed to.

Fixing the instant at the Step's Expansion rather than at the Run's start is the near miss. It removes
nothing: the Steps before it still move it, which is the whole objection. It also stores an
approximation of the value it used, the Step file's `started_at` being when the Step began rather than
when its Expansion read the clock, where the Run's `started_at` **is** the value.

We chose the Run's start because a Run is the unit against which change is reviewed (§2), and the time
it happened is a fact about the occasion — which is the Trigger's territory, and the Trigger is
Run-wide. It sits with `repo_revision` and `hyper_version` in Provenance: one Run, one reading, and no
Step disagreeing with another about what day it is under a Cadence.

## Considered options

- **The wall clock at each evaluation.** Rejected above, and it fails a second time on concurrency: a
  `read` Step's Expansion runs concurrently (§6), so nothing would fix which *now* each item was judged
  against, and *which three of the five* stops having the determinate answer §6 promises.
- **The moment the Step's Expansion resolves.** The serious alternative, and it is defensible on the
  ground that a Step should act on the world as it is when it acts. Rejected because the objection is
  not staleness but *whose behaviour decides the boundary*, and this reading leaves that with the
  preceding Steps.
- **An instant supplied at invocation** — a `--now` flag, or an input. Rejected outright: an input is
  authority arriving after review (ADR-0008), and an artefact whose reach depends on a flag is the
  shape ADR-0001 removed. It would also make the boundary a thing the Store cannot reconstruct.

## Consequences

- **Nothing is stored for it.** The instant is `run.json`'s `started_at`, which every Run writes
  already. A reader reconstructing why five expanded to three reads the entry they are already holding.
- **A long Run judges against a stale instant, and the staleness is in the safe direction for
  `destroy`.** An Asset that crosses the `older_than: 14d` boundary while Step 1 is provisioning is not
  reached by Step 40, so the set is a subset of what a fresher clock would have given, never a superset.
  A `newer_than` window shifts rather than shrinks; it is not the operator a `destroy` is usually
  written with, and the shift is bounded by the Run's own duration either way.
- **A Probe uses its own start instant and records it nowhere.** A Probe is an invocation and this rule
  is about invocations, so a polling `until:` inside one behaves identically; that the instant is
  unrecorded is ADR-0009's existing position — a Probe writes no Record and no Journal entry — rather
  than an exception here. Refusing a relative predicate inside a Probe was rejected: it would make an
  Operation runnable and un-probeable on a fact about the Manifest the caller cannot see.
- **The resolved instant is a rendered fact.** Wherever a Run's surfaces render a selector,
  `older_than: 14d` is glossed with the absolute instant it resolved to. This is ADR-0005's Cadence
  gloss one axis over, and it is derived arithmetic rather than a claim, so it leaves §8's
  one-editorial-surface rule where it stands. `check` renders no gloss, being offline with no Run and
  therefore no instant.
- **§13's *written value that depends on when the Run happens* keeps its entry and narrows its
  reason.** It argued from *no artefact names the current instant*, which is no longer true: a
  predicate names one, relatively, in a filter. What survives is the distinction that matters — nothing
  names the instant in a **value** position, so a date a Step writes is still the literal its author
  wrote, and a Procedure on a Cadence writes that same literal at every occurrence.
