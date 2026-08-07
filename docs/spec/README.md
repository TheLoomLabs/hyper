# `docs/spec/`

The specification for `hyper`: what it does, not why. Read the sections in file
order — each depends on the ones before it. Rationale lives in
[`docs/adr/`](../adr/); vocabulary lives in [`CONTEXT.md`](../../CONTEXT.md);
neither is restated here.

1. [What `hyper` is](01-what-hyper-is.md) — the thesis and the one-screen demo.
2. [Vocabulary](02-vocabulary.md) — a pointer to `CONTEXT.md`.
3. [The model](03-the-model.md) — Provider, Operation, Kind, the five artefacts, Target, Record.
4. [The authoring format](04-the-authoring-format.md) — the YAML subset, the five artefacts, holes, paths.
5. [Static verification](05-static-verification.md) — everything `check` refuses before anything runs.
6. [Authority and safety](06-authority-and-safety.md) — two keys, envelope, Bound, Expansion, Refusal.
7. [Execution](07-execution.md) — sequence, Repeatability, conditions, concurrency, outcomes.
8. [The record](08-the-record.md) — Store, Head, Journal, Disposition, Provenance, retention.
9. [Review and Comparison](09-review-and-comparison.md) — the renderings verbatim, the NDJSON contract.
10. [Surfaces](10-surfaces.md) — the CLI's commands and exit codes, the MCP tool schemas.
11. [Cadence and projection](11-cadence-and-projection.md) — cron, gloss, generated workflow, job summary.
12. [Distribution and version pinning](12-distribution-and-version-pinning.md) — install, digest, `project`, skew.
13. [The closed sets](13-the-closed-sets.md) — every closed set `hyper` defines, in full, once.
14. [Non-goals and honest limits](14-non-goals-and-honest-limits.md) — what `hyper` will not do and what it costs.

This list glosses the order the filenames already fix; the numbering in
`docs/spec/` is the one representation of it. Filenames run `01`–`14`; each
file's own heading names it `§0`–`§13`, one lower, because the map that fixed
this order numbered its first section `§0`.
