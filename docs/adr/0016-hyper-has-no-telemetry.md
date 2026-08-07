# `hyper` has no telemetry

`hyper` ships no OpenTelemetry, no exporter, no metrics, no trace context and no logging framework.
The Journal is the trace: a Run is one process running one sequence of Steps, so there is a single
root, no concurrency to reconstruct, and — after ADR-0004 removed code from Providers — no second
process to propagate a trace context into. Adding a telemetry subsystem would mean adding a
dependency tree and an exporter, and an exporter implies a collector, which is infrastructure this
tool exists to avoid.

This is recorded because *nothing was added* is the hardest kind of decision to reconstruct later, and
because the prior art makes it the reflex. Swamp ships traces and native OTLP log records spanning
CLI → workflow → job → step → model method, with `TRACEPARENT` propagation into extensions and
containers. Its own README says the value of this is "most useful for long-running `swamp serve`
daemons" — the architecture this project put out of scope.

## Considered options

- **Optional OTel behind `OTEL_EXPORTER_OTLP_ENDPOINT`**, as Swamp has it, with zero overhead when
  unset. Rejected on what it obliges rather than what it costs when off: a span vocabulary to keep
  stable, secret redaction on a second egress path, and a standing invitation to answer questions
  from the collector that the Journal is supposed to answer from the repository. The Journal would
  become the thing you check when the collector disagrees.
- **A metrics endpoint or a stats file.** Rejected: there is no process alive to scrape, and a
  counter is a derived reading of Journal entries that anything can compute by looking.
- **A logging framework with levels.** Already removed by ADR-0014's three-globals rule — `--quiet`
  is `2>/dev/null` once stderr is narration.

## Consequences

- **Per-Step timing lives in the Journal or nowhere.** A Journal entry stamps when the Run began, and
  every Step Disposition stamps when that Step began and ended. Durations are derived at render and
  never stored — a stored duration is a second representation of a fact the timestamps already carry,
  and it can disagree with them.
- **A duration is only ever computed inside one Journal entry.** Same process, same clock. Timestamps
  from two entries are never subtracted, because the laptop and the runner do not share a clock; this
  is the limit ADR-0011 names as a consequence of deriving the Head from `written_at`, and it is why
  no rendering may present a cross-entry interval as a measurement.
- **If anything ever leaves the machine, it leaves as a rendering, not as a span.** This is the
  constraint handed to the notification question, which owns egress.
