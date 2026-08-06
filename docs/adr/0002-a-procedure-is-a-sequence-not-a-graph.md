# A Procedure is a sequence, not a graph

Every comparable automation tool models work as a DAG, so a future reader will assume `hyper`'s flat,
top-to-bottom Step list is an unfinished version of one. It is not. A Procedure is an **ordered set of
Steps** that execute in written order, and there is no dependency edge, explicit or inferred,
anywhere in the model.

The reason is that `hyper`'s artefacts are written by an AI and reviewed by a human. A DAG needs the
author to declare an edge for every data reference, and Swamp is the case study for what happens when
they don't: its edges come *only* from explicit `dependsOn` (the machinery to infer them from data
references was written and then abandoned unused), so a step that reads another step's output without
declaring the dependency passes validation, usually works, and intermittently reads stale
infrastructure state. That is the worst possible failure mode for an artefact nobody hand-wrote. In a
sequence there is no edge to forget — "Step 5 references a Record that Step 6 produces" is a static
error caught at review time rather than a race at runtime.

## Consequences

- Independent Steps do not run in parallel. All concurrency comes from Expansion — one Step over many
  Records — which is where the real fan-out lives (200 certificates, not 3 unrelated Steps).
- The execution order is the reading order, which is what makes the review gutter in the oversight
  model work: a graph cannot be read down the side of a file.
- Because the invocation graph between Procedures is static and not expression-driven, cycles are
  rejected before the first Step and no depth limit is needed.
