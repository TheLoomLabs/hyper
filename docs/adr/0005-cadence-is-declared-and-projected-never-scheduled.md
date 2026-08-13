# Cadence is declared and projected, never scheduled

A Procedure may declare a **Cadence** — a five-field POSIX cron expression, UTC only. `hyper` never
sleeps, never daemonises, and never watches a clock. It *generates* the executor's own schedule file
from that declaration, and a static check fails when the generated file no longer matches. GitHub
Actions supplies the clock; `hyper` supplies the declaration and the projection between them.

We chose this because cadence is a blast-radius multiplier, and it was the last authority-adjacent
fact living outside the review surface. Changing `0 0 1 * *` to `*/5 * * * *` on a Procedure
containing a `destroy` Step multiplies its effect on the world by roughly 8,800 — and if the schedule
lives in a hand-written workflow file, that change is reviewed nowhere, by the same agent that wrote
the Procedure. ADR-0001 and the safety model put authority in the artefact; the oversight model made
the artefact the review surface. A cron line in `.github/workflows` escaped both.

## Considered options

- **No declaration at all**, with the executor owning timing entirely. Genuinely attractive: it keeps
  `hyper` ignorant of a fact it cannot enforce, and adds no grammar. Rejected because it puts
  frequency beyond every guardrail, and because it makes a stopped schedule undetectable — the tool
  cannot say "this was supposed to run hourly and last ran on Tuesday" if it has never been told
  "hourly".
- **A built-in scheduler or daemon.** Excluded by the stateless-binary constraint and by the
  execution model's *no reaper, no daemon, no heartbeat*. Swamp's prior art does not transfer:
  `trigger.schedule` is a croner registry that runs only inside `swamp serve`, the long-lived server
  this project has ruled out of scope — a declaration with no executor behind it.
- **A `hyper`-native interval grammar** (`every: 15m`). Rejected because projection must be exact.
  `every: 7m` has no cron expression, so the grammar would admit cadences the executor cannot honour,
  and the tool would be lying in the one file the human is asked to trust.

## Consequences

- **Cron is the grammar and UTC is the only timezone.** A local-time schedule shifts by an hour twice
  a year — a behaviour change with no artefact edit, which is precisely what ADR-0001 exists to
  prevent. The cost is that "3am my time" is unexpressible.
- **The gloss is mandatory, not a nicety.** Cron is write-only for humans and agents alike, so the
  review surface renders `0 3 * * 1` as `03:00 UTC every Monday` *and* its derived frequency
  (`≈4.3 runs/month`). The frequency is the number that matters beside a Procedure that destroys.
  *Extended by ADR-0063:* it renders in the review's **header** with the last Journal entry beside it,
  and the rule is total — every surface rendering a Cadence glosses it, the `changed` flag on a
  `cadence:` line included, which is where the ≈8,800× above is actually read. *Stated in full by
  ADR-0066:* the phrase renders each field in the form it was written in and says *every n* only where
  the step divides the field's span, and the frequency's denominator is the 400-year Gregorian cycle
  rather than a calendar year — which is where the ≈8,800× above stops being an estimate.
- **A missed window is never made up.** Cadence is a lower bound on staleness, not a promise of
  coverage. An Observation is a fact read at a point in time and a past window is unreadable; for
  effectful Steps, catch-up would mean the clock deciding to repeat an effect, which Repeatability
  decides on evidence instead. GitHub documents that scheduled runs are delayed under load and
  dropped outright when it is severe, so no coverage promise was available to make.
- **`hyper` claims periodic checking, not monitoring.** The floor is the executor's (five minutes on
  Actions) and delivery is best-effort. Continuous probing and alert-on-transition are a different
  product.
- **There is no local clock.** Scheduled recurrence is an Actions property; the laptop authors, runs
  ad hoc, and reviews.
- **Overdue renders, never refuses.** Being overdue is a fact about the executor, not about the
  artefact or the world, and every guardrail refuses only on what `hyper` controls. *Amended by
  ADR-0021:* it renders as facts placed side by side — the gloss and the last Journal entry — and
  `hyper` never asserts the judgement itself.
- **The generated workflow renders into the job summary.** *Added by ADR-0021.* The projection carries
  two `hyper` invocations rather than one — the Run, then the Comparison under `if: always()` — `tee`s
  both to `$GITHUB_STEP_SUMMARY` as fenced text, and names the job after the Procedure, since that
  string is the subject line of the executor's own failure email. The runtime binary is told nothing
  about any of this; only `project` knows the executor.
- **The lock projects too.** A Procedure containing any effectful Step generates a workflow carrying a
  `concurrency` group with `cancel-in-progress: false`, which is how the execution model's
  single-store lock survives two runners that share no filesystem.
- **The generated workflow pins the `hyper` version**, so an auto-updating binary cannot change
  Expansion ordering or Bound checking between two Runs without an artefact edit.
