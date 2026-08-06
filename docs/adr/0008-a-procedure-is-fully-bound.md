# A Procedure is fully bound

A Procedure takes no inputs at invocation. Everything a Run needs — the Operation, the Definition, the
Target, the selector, the Bound — is written in the artefact, so `hyper run <procedure>` and the tool
call behind it carry no argument that could change what the Procedure does. The ad-hoc question that
would otherwise want an argument is answered by a Probe (ADR-0009) instead.

We chose this because an input is authority arriving after review. The whole oversight model rests on
the artefact being the thing a human reads and the thing they approve; a value supplied at call time
is Step behaviour appearing on no reviewed line, and the review surface cannot annotate what is not
in the file. That is the same shape as the `--force` flag ADR-0001 removed, and it would arrive
through a door nobody was watching. Three decisions had already pointed here without being about
inputs at all: a Procedure carrying a Cadence must have no unbound inputs, because there is nobody to
supply them on a schedule; the review gutter needs every consequential fact to occupy a line it can
sit beside; and the friction that made call-time arguments attractive — a one-off read needing a whole
artefact — now has its own answer.

## Consequences

- **Parameterisation is duplication, deliberately.** "Retire stale VMs in prod" and "…in staging" are
  two Procedures, not one Procedure invoked twice. They may share a nested Procedure for the common
  part, which is the same shape scheduling already settled for "prod every 5 minutes, staging
  hourly". The cost is real: a change that should apply to both must be made in both, or factored
  into the shared Procedure by hand.
- **The blast radius of an invocation is knowable before it happens.** With no arguments, the static
  authority check and the rendered review are complete descriptions of what a Run can do. Nothing a
  caller types can widen them, which is what lets the same guardrails hold identically on a laptop and
  on a runner.
- **This constrains the authoring language.** Whatever format survives needs no parameter
  declaration, no interpolation of caller-supplied values, and no defaulting rules — a smaller
  surface than it would otherwise carry, in the same way the language was already constrained to
  line-oriented text and to conditions referencing only earlier Steps of the same Run.
- **The invocation is not empty, but what remains is never authority.** A Secret sink path, a dry-run
  marker, and output formatting are properties of the occasion rather than of the work; none of them
  can change which Operation runs against which Target, or how many Records it may affect.
