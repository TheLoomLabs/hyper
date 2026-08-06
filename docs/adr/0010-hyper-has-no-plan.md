# `hyper` has no plan

`hyper`'s **Comparison** is retrospective and has no prospective counterpart. It renders one Run
against the Run before it — the Assets `hyper` changed, the Observations the world changed, and the
code that changed between the two — and nothing in the tool will ever render a proposed change
before it happens. A Terraform-style `plan` requires reconciling a desired state against an observed
one; `hyper` is procedural, has no desired state, and deliberately declined the reconciliation
engine that would produce one.

We chose this because the reconciliation engine is the single largest thing this project does not
build, and a `plan` is its user-visible face — shipping the face without the engine would be the
first dishonest surface in the tool. The retrospective direction is also the one that answers the
question actually being asked. *What will this do* is a question about code, and the artefact review
answers it by being readable. *What changed since Tuesday, and did the AI change the code between
those two Runs* is a question no `plan` in the field answers at all, and it is the half of the
thesis that says nothing changes unseen.

## Considered options

- **A pre-flight rendering before a destructive Step**, computing what a selector would reach and
  showing it for approval. Rejected on two independent grounds: it needs the reconciliation the
  domain model declined, and per-Run execution approval was already rejected — CI is unattended, so
  a prompt at the dangerous moment has nobody to answer it. What replaces it is that the dangerous
  state is unreachable from the artefacts rather than confirmed at the moment it arrives.
- **Dry-run as a plan in everything but name.** Rejected because a dry-run stops where it would
  otherwise have to lie: the moment a later Step's behaviour depends on an earlier Step's real
  output, it has nothing truthful to say. It is a review aid, not a guarantee, and promoting it to a
  `plan` would promote its silences to promises.

## Consequences

- **The Comparison prevents nothing.** It is an accountability instrument, not a guardrail. Every
  guardrail in `hyper` is static and sits before the Run: the two-key authority check, the
  named-Operation requirement on destroy, the Bound checked at Expansion, and a Refusal with no
  bypass. If any surface ever implies the Comparison is protecting you, it has lied.
- **There is no pre-flight before a destructive change**, and that is a real gap rather than a
  solved problem. It is carried by the artefact review and by the Bound, both of which act on the
  authored claim rather than on the world.
- **The Comparison needs a baseline, so Runs must persist.** This is one of the reasons the record
  travels in the repository (ADR-0006). A dry-run writes a marked Journal entry and is never a
  baseline, and a Probe writes nothing and can never be one (ADR-0009).
- **The Comparison never reports disagreement, only change.** Observations and Assets are never
  reconciled, so the tool can say *this differs from when we last looked* and can never say *this
  differs from what we intended*. The second sentence is drift detection, which is the engine this
  ADR declines to build.
