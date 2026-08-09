# §10 — Cadence and projection

A Procedure may declare a **Cadence**: how often it is meant to run. `hyper` keeps no clock of its own
and never runs it. It *projects* the declaration into an executor's clock — a workflow file it
generates, that a human reads in a diff, and that a static check holds to the declaration it came from
(ADR-0005).

This chapter states the grammar a Cadence is written in, what it promises and what it does not, the
gloss every surface renders it through, the workflow `project` generates, the check that fails when
that file no longer matches, and the job summary the projection writes.

## The declaration

A Cadence is authored on a Procedure, in `procedures/`, as one field of the artefact §3 states. A
Procedure declaring none is run by hand and generates no workflow; there is no default recurrence and
nothing inherits one.

One Procedure declares at most one Cadence. Two recurrences are two Procedures, with any shared body
factored into a nested one — which stays one Run (§6) — so *prod every five minutes, staging hourly* is
two artefacts rather than one artefact with two clocks. A Procedure invoked by another may declare a
Cadence of its own; it is projected like any other, and its invocation from elsewhere is still one Run.

A Cadence is a code fact: it appears in `THE CODE MOVED` when it moves between two Runs, on the same
footing as a Bound (§8, §12). It is a blast-radius multiplier, which is why it is declared in the
reviewed artefact and the executor's copy derived from it (ADR-0005).

### The grammar

A Cadence is a five-field POSIX cron expression, UTC only (ADR-0005). The fields are space-separated
and in one order — minute (`0`–`59`), hour (`0`–`23`), day of month (`1`–`31`), month (`1`–`12`), day
of week (`0`–`6`, Sunday `0`).

Each field is `*`, a number, a comma-separated list, a range `a-b`, or a step over either — `*/n` or
`a-b/n`. Nothing else is in the grammar. A sixth seconds field, an `@hourly`-style nickname, a month or
day *name*, `?`, `L`, `W`, `#`, and any timezone or offset are all rejected at load
(`cadence-malformed`, §12). A name is rejected on the ground the nickname is: it is a second spelling
of a Cadence the numeric form already states.

Where the day-of-month and day-of-week fields are both restricted, POSIX matches a day satisfying
*either*, and the gloss below says so in words.

There is nowhere to name another timezone — no field, no flag, and no file (ADR-0014). *3am my time* is
therefore unexpressible, which is stated here as the cost it is and carried forward to §13.

An expression finer than the executor's floor loads like any other. What it costs is legible in the
rate the gloss derives, on the review surface where every other blast-radius fact is read.

## What a Cadence promises

A Cadence is a lower bound on staleness rather than a promise of coverage (ADR-0005). It bounds how
long a fact may go unrefreshed; it says nothing about any particular window having been served.

A missed window is never made up. There is no catch-up, no backlog, and no queue of skipped
occurrences: re-invocation is decided by Repeatability against the Journal's evidence and never by a
clock (§6, ADR-0005).

The floor is the executor's — five minutes on Actions — and delivery is best-effort.

A Cadence is also the declaration the last Journal entry is read against, which is what makes staleness
readable at all. It is derived when something looks, and nothing watches for it.

## The gloss

Cron is write-only for humans and agents alike, so no surface renders a Cadence as the expression
alone. Wherever a Cadence renders, the gloss renders with it (ADR-0005, ADR-0021) — `0 3 * * 1` as:

```
03:00 UTC every Monday   ·   ≈4.3 runs/month   ·   last ran 41 days ago
```

The **phrase** states the times of day, the days, and the months the expression selects, always naming
UTC, and states the day-of-month/day-of-week union in words where both fields are restricted.

The **rate** derives by counting the expression's matches over a calendar year and dividing by twelve.
It is the number that matters beside a Procedure that destroys, which is why it is rendered rather than
left to be inferred from the phrase.

The **last Journal entry** is the most recent Run of that Procedure, dry-run entries filtered out like
every other reading of Journal evidence (§7). What an overdue reading is made of is that entry, its
Trigger (§7), and its age against the declared Cadence — facts placed beside each other, with the human
doing the subtraction.

`hyper` never says *overdue*, and nothing refuses on it. Any threshold would be a claim of the tool's
own on a surface built to admit none (ADR-0021, ADR-0026), and being overdue is a fact about the
executor rather than about the artefact or the world.

Where no Store is reachable the last entry degrades to `last ran: unknown (no Store)` and the surface
renders anyway: `review` reads the Journal where there is one and never requires one, which is what
keeps the offline authoring loop intact (§9).

### Two facts the gloss carries

Two facts about how the executor will treat the declaration render beside the three above, wherever the
gloss renders. Neither is a problem with the artefact: neither carries an `error_code`, neither fails
`check`, and neither is a claim `hyper` makes about what the world holds.

**Scheduled workflows run only on the default branch.** A Cadence on a feature branch is inert, and
saying so where the Cadence renders is what keeps it from being discovered three weeks later.

**An hour-boundary collision.** Actions names the start of every hour as its high-load window, so a
Cadence landing there is the one most likely to be delayed or dropped. `project` never adjusts a
declared time to avoid it: a generated time nobody wrote would be a small lie in a file whose whole
value is that it says what was declared.

## The projection

`project` generates one workflow file per Procedure that declares a Cadence, at
`.github/workflows/hyper-<procedure>.yml`, carrying the Procedure's name verbatim — one file per
Procedure, so run history in the Actions UI is per-Procedure and nothing is ambiguous about which
expression fired. Generation being whole-file, repo-wide and all-or-nothing (§9), a Procedure that has
dropped its Cadence loses its file to the same command that writes the rest.

### The generated workflow

```yaml
# generated by hyper 0.4.1 — edits are overwritten by `hyper project`
name: retire-preview-envs

on:
  schedule:
    - cron: '0 3 * * 1'

permissions:
  contents: write

concurrency:
  group: hyper-store
  cancel-in-progress: false

jobs:
  run:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8

      - name: install hyper 0.4.1
        run: |
          curl -fsSL -o hyper.tar.gz \
            https://github.com/TheLoomLabs/hyper/releases/download/v0.4.1/hyper-0.4.1-x86_64-linux.tar.gz
          echo 'a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2  hyper.tar.gz' \
            | sha256sum -c -
          tar -xzf hyper.tar.gz

      - name: hyper run retire-preview-envs
        run: |
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          set +e
          ./hyper run retire-preview-envs | tee -a "$GITHUB_STEP_SUMMARY"
          code=${PIPESTATUS[0]}
          set -e
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          exit $code

      - name: hyper changes retire-preview-envs
        if: always()
        run: |
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          ./hyper changes retire-preview-envs | tee -a "$GITHUB_STEP_SUMMARY"
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
```

### What the projection carries

**The workflow and its job are named after the Procedure**, because that string is the subject line of
the executor's own failure email (ADR-0005) and the only part of a failure visible on a phone. GitHub
addresses that email to whoever last committed the projection, which is whoever reviewed the Cadence
(ADR-0021).

**`on:` carries the recurrence and nothing else.** The reviewed artefact declares a recurrence and
nothing more, so the projection derives no second occasion for a Run to start from; a Run started by a
person is started from a laptop and records that Trigger (§7).

**`permissions: contents: write`** is what a runner needs in order to write the Store, and the whole of
what the projection grants. Writing the Store is not an Operation and costs no Capability (§7,
ADR-0006).

**The `concurrency` group is the Store's**, one group for the repository rather than one per Procedure,
with `cancel-in-progress: false`. It is what stands in for the single-store lock §6 states across two
runners that share no filesystem (ADR-0005). A second effectful workflow firing in the same minute
queues behind the first, and where a third piles up the oldest pending run is dropped — which the lower
bound above already accepts. A Procedure whose every Step is `read` carries no group at all: it takes
the shared lock and reaps nothing, and serialising it would starve the five-minute cadence behind the
forty-minute provision (§6, ADR-0006).

**The checkout leaves the token behind**, which is what `hyper` fetches and pushes the Store branch with
— that sync is `hyper`'s own work rather than a step of the workflow (§7). `actions/checkout` is the
one action the projection names, pinned by commit SHA, and `runs-on` names one pinned platform.

**The install step carries the version, the URL, and the digest** of the artefact for the platform
`runs-on` names — the pin and its verification in one reviewed file, with nothing resolved at run time
(§11, ADR-0020). The binary the runner fetches compares itself against the same pin before it reads a
second file (§9), so a workflow left behind by an older projection Refuses rather than acting.

**Two `hyper` invocations, and no branching.** `run` first, then `changes` under `if: always()`. One
invocation would put the Step narration on the summary page and leave the Comparison — half of what
there is to see — unrendered. A Refusal returns the full Refusal rendering on stdout and exits `77`
(§8, §9), so the first invocation already puts the remediation surface on the page and no conditional
is needed to get it there (ADR-0021).

**`${PIPESTATUS[0]}` is what fails the job.** The status the step exits with is the `hyper`
invocation's rather than `tee`'s, which is what makes a Refusal, a failure, and Store contention land
as a red job carrying the code §12 fixes. The `set +e` around it exists so the closing fence is written
before the step exits.

The runtime binary is told nothing about any of this. Only `project` knows the executor, which is what
keeps `hyper run <procedure>` producing the same bytes on a laptop as on a runner (ADR-0021).

## The check

A generated file that is not what `project` would write now fails `check` (`projection-stale`, §12) —
the verification half of generate-and-verify, and the reason the declaration stays the source rather
than becoming a comment on the executor's copy (ADR-0005). The comparison is whole-file and byte-exact
against a fresh regeneration, so it catches every way the two can part: a Cadence edited and not
projected, a hand-edit to a generated file, a generated file deleted, one left behind by a Procedure
that no longer declares a Cadence, and a hand-edited version pin — which is therefore caught twice,
here and by the fetched binary's own pin gate (ADR-0020).

It is one of `check`'s rules and runs with the rest of them, standalone and in the pre-flight of every
Run alike (§4, §9); it is stated here because it compares a reviewed artefact against a file derived
from it, and the derivation is stated here. Regeneration reaches no network — a version change is the
only thing that ever does (ADR-0020) — so the check runs wherever `check` runs.

What repairs it is `project`, always, and never a flag on the check itself (§9). A hand-edit to a
generated file does not survive the next `project`, which is correct rather than regrettable: it is
authority living outside every reviewed artefact.

## The job summary

The projection writes the ordinary renderings to `$GITHUB_STEP_SUMMARY`, fenced rather than converted
to Markdown, so the review gutter and the aligned columns survive and the page carries what the
terminal carries (§9, ADR-0021).

What lands there is the Step table `run` writes — one row per Step carrying its index, its id, its
Kind, the count of Records it wrote, and its **Disposition**, one of the six §12 defines: *ran*,
*skipped as already recorded*, *skipped by condition*, *refused*, *never reached*, and *attempted,
outcome unknown*. Where a guardrail declined, §8's Refusal rendering follows in full, remediation table
included. Then the Comparison, under `if: always()`, so a Run that failed still renders what reached
the world before it stopped.

The Dispositions are why the page is worth writing. A green check means the Run finished, not that
anything happened: a Run whose every Step was skipped as already recorded completes and exits `0` like
any other (§6), no exit code distinguishes it, and this page is where the difference is legible — as it
is where *ran* three times and *never reached* twice reads as the partial Run it was.

`tee` rather than a redirect, because a summary over the executor's per-step limit is dropped *without
failing the step*: the Actions log is always complete and the summary is a convenience copy of it
(ADR-0021).

## No clock of its own

`hyper` never sleeps, never daemonises, and never watches a clock. There is no scheduler in the binary,
no timer, no background thread, no `serve`, and nothing that outlives the invocation that started it
(§9).

Scheduled recurrence is a property of the executor. The laptop authors, runs ad hoc, and reviews; the
runner is the only thing that keeps time. Nothing is lost to that division — whatever the laptop holds
credentials for it can run by hand, and that Run is a first-class Run recording its own Trigger (§7).

A scheduled Run may be of any Kind. Unattended destruction is what the two keys, the mandatory Bound on
`destroy`, named-Operation granularity, and the absence of any bypass exist to make safe (§4, §5,
ADR-0001), and reserving the clock for `read` Procedures would undo all of it.

## What a Cadence does not buy

**Continuous monitoring is not what this is.** `hyper` claims periodic checking against a floor the
executor sets, with best-effort delivery and no made-up window. A sub-minute prober is a hosted service
rather than a stateless binary an external clock invokes. Named here as a non-goal and carried forward
to §13.

**Alert-on-transition does not exist.** Nothing watches a Record for a value crossing a line and
nothing emits when one does. The Comparison renders `status: 200 → 503` when someone reads it (§8), and
the notification you want is a Step you author — a Definition against a Slack or PagerDuty Target,
reviewed, Bounded and recorded like everything else (ADR-0021). It can announce that something
happened; it can never announce that something failed to happen. Carried forward to §13.

**Two executor failures `hyper` can neither see nor prevent.** A scheduled workflow auto-disabled after
60 days of repository inactivity produces no run, no error in the Actions tab, and no banner — only
GitHub's own warning email, addressed to whoever last enabled it, with the gloss's *last ran* the only
thing that shows it, and only when somebody looks. And an oversized job summary is dropped silently,
which is what the `tee` above exists to survive. A tool that only ever renders cannot see a run that did
not happen; both are stated rather than mitigated, and carried forward to §13.
