# Safety refusals have no bypass flag

`hyper`'s guardrails on destructive work — the Kind intersection between a Definition and its Target,
the transitive Target and Kind envelope, the mandatory Bound on a `destroy` Step — all end in a
Refusal. We decided that **nothing overrides a Refusal at invocation time**: no `--force`, no
`--yes`, no `--skip-checks`, no per-Run bypass of any kind. The only way past one is to change the
authored artefact — widen the Target's accepted Kinds, name the Operation, raise the Bound — which
puts the change back through the single human review the whole oversight model rests on.

This runs against the grain: comparable tools all ship an escape flag, and a future reader will
assume its absence is an oversight. It is not. Every guarantee `hyper` makes about AI-authored
automation is conditional on the guardrail being unskippable, and the party most likely to reach for
the flag is an agent in a retry loop at 3am with no human watching. Swamp is the cautionary case —
its pre-flight checks are real, and skippable via `--skip-checks` with no documented restriction, so
its guarantee is opt-out by default.

## Consequences

- A Refusal in CI cannot be cleared from CI. It requires an artefact edit and a review — which is the
  intended cost, not a gap to close later.
- There is no breakglass path. If one is ever genuinely needed, it must be designed as a reviewable
  artefact (an explicit, recorded widening of a Target's policy), never as a flag on the run command.
- Because the escape hatch does not exist, the checks must be right. An over-eager guardrail becomes
  an outage rather than an annoyance, so each one is deliberately scoped to a failure mode we can
  state precisely.
