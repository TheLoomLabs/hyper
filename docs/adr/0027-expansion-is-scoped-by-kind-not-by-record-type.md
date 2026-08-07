# Expansion is scoped by Kind, not by Record type

A selector's **Expansion** reaches further for a `read` Step than for a `mutate` or `destroy` one, and
that asymmetry is deliberate rather than an oversight waiting to be tidied away. `hyper` first bounded
Expansion to Assets everywhere, argued on blast radius alone, then found the bound wrong for half the
workload once real read scenarios were run against it — so the reach now tracks the Step's own Kind
rather than one rule applied to every Step regardless of what it does.

An asymmetric rule reads as unfinished — a future reader meeting "reads reach anything, effectful
Steps reach only Assets" will assume the two halves were never reconciled and one of them is a bug
waiting to be filed. Neither is true; the asymmetry is the whole point.

We chose this because the blast-radius argument that justified Assets-only never reached `read` in the
first place. A selector ranging over an unbounded set of things `hyper` is accountable for is exactly
the danger the safety model exists to bound, and that holds completely for `mutate` and `destroy` — but
blast radius has nothing to say about a Step that only looks. Applied to reads anyway, the rule makes
"list the VMs in this account, read each one's disk usage" unwritable without naming two hundred literal
identifiers in the Definition, which is exactly the case a selector exists to serve and half of
`hyper`'s stated workload besides.

## Considered options

- **Expansion reaches Assets only, without exception.** The rule as first written, and the reading a
  reviewer meeting only the safety argument would reach and stop at. Rejected once ordinary read
  workloads were tried against it: a `read` Step almost always operates over things `hyper` has only
  observed, never created, and an Assets-only rule makes that class of Step unwritable.
- **Expansion reaches any Record, for any Kind.** The other uniform reading. Rejected because the
  blast-radius argument is exactly right for `mutate` and `destroy`: an effectful selector ranging over
  Observations `hyper` never created has no Bound to check it against, since a Bound is stated against
  the Assets a Definition owns.

## Consequences

- **The Assets-only sentence is the one a competent implementer reaches unaided, and it is only half
  true.** It is exactly right for effectful Steps and silently wrong for reads, which is what makes it
  the natural first draft of the rule rather than an obviously incomplete one. This decision is that
  sentence's permanent correction, stated with its Kind qualifier attached rather than left implicit.
- **A `read` Step can select over Observations that no Definition ever created**, which is the one
  place Expansion crosses that boundary. This is safe only because a `read` Step has nothing
  destructive for a Bound to guard.
- **An effectful selector still has nowhere to reach but the Assets a Definition owns.** The Bound,
  mandatory on `destroy`, is stated against exactly that set — widening Expansion for effectful Steps
  would leave the Bound checking a set it was never defined against.
- **The two halves never merge into one sentence that is still short.** Any future rewording of
  Expansion has to keep the Kind qualifier explicit; the price of a shorter sentence is the same
  silent narrowing this decision exists to correct.
