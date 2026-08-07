# `hyper` never speaks first

`hyper` sends nothing. No webhook, no email, no push, no desktop notification, no terminal bell. It
renders when it is asked to and is otherwise silent. The notification you want is not a capability
`hyper` withholds — it is a **Step**: a Definition against a Slack or PagerDuty Target, `mutate` Kind,
`http` Capability, credentials resolved from the environment like any other, invoked from a Procedure.
Reviewed, Bounded, recorded and Compared exactly like everything else. `hyper` never notifies; you
author a notification, and it goes through the front door.

We chose this because egress `hyper` performs on its own initiative is a destination no reviewed
artefact named, using a credential ADR-0007 says the tool does not hold, over a second redaction path,
under a Capability that cannot join the closed set without being a wildcard. The claim this project
makes is that nothing reaches the world unreviewed — and a notification *is* reaching the world. A
tool that quietly exempts its own outbound messages from the rule it exists to enforce has already
lost the argument. ADR-0016 declined telemetry on this ground and handed egress here to be settled.

The ambient signal in CI already exists and is not ours to build. Exit codes carry the outcome —
`refused` is 77, `failed` is 1, Store contention is 75 — and a nonzero exit fails the job. GitHub then
notifies the user who last modified the cron syntax, which after ADR-0020 is whoever committed
`hyper project`: the person who reviewed the Cadence is the person the executor emails, derived rather
than configured. A channel of our own would duplicate a signal already addressed to the right person,
and would be the only part of the system whose delivery `hyper` could not record.

## Considered options

- **A webhook or SMTP sender behind a flag.** Rejected on what it obliges rather than what it costs
  when unset: a destination that is authority arriving outside every reviewed artefact, retry and
  delivery semantics for a channel with no Record, and a redaction path that has to be got right a
  second time on output nobody reviewed. ADR-0014 also leaves nowhere to configure it.
- **A Step that runs when an earlier Step failed.** The most requested feature in every automation
  tool, and precisely the conditional edge ADR-0002 removed. A failed Step wrote no Record, so the
  condition has nothing to read under the execution model's rule that conditions see only earlier
  Steps' Records of this Run; admitting Dispositions as conditions rebuilds the graph, and it
  rebuilds it into the post-halt world state the safety model refused to reason about.
- **A built-in Provider that reads the Store**, making "every morning, check yesterday's Journal and
  POST to Slack" an ordinary cadenced Procedure. The most elegant of the rejected options, needing no
  new execution semantics at all. It fails twice: it makes the Store a Target by the side door, which
  ADR-0011's data layer ruled out on the grounds that reaching the Store costs no Capability; and it
  shares the failure mode of the thing it watches, since the notifier is scheduled by the same Actions
  that stopped. The one condition it exists to report is the one it cannot report.
- **A watcher Procedure with its own Cadence**, checking that the others ran. Dies with the previous
  option and for the same reason — a daemon in a cron costume, switched off by the outage it monitors.
- **A local ping** — a terminal bell or an OS notification when a long Run finishes. Not egress, so it
  survives the argument above, and it dies on a different one: it is an is-a-TTY axis, the same shape
  as the is-CI axis the safety model deleted and ADR-0015 refused. It would be the first thing that
  behaves differently on the laptop and the runner. Per-Step progress on stderr is the progress story,
  and it is identical in both places.

## Consequences

- **The projected workflow renders into the job summary, and the binary never learns it exists.** The
  workflow carries two invocations — `hyper run <procedure>`, then `hyper changes <procedure>` under
  `if: always()` — and `tee`s both to `$GITHUB_STEP_SUMMARY`. Because a Refusal returns the full
  rendering on stdout, the first invocation already puts the Refusal surface on the summary page, so
  the workflow needs no branching. `hyper project` knows the executor by construction; the runtime
  binary does not, which is what keeps the same command producing the same bytes on a laptop.
- **The summary is a fenced block of the ordinary rendering, not a Markdown mode.** Monospace
  preserves the review gutter and the aligned tables exactly, colour is never load-bearing, and the
  job summary is therefore not a new surface — it is the existing renderings relocated. `tee` rather
  than redirect, because GitHub drops an oversized summary *without failing the step*: the log is
  always complete and the summary is a convenience copy.
- **You cannot author "tell me when this fails."** An error halts, so a trailing notify Step is never
  reached on the failure you wanted to hear about; and no Step can see an outcome that does not exist
  until the Run ends. An authored notification can announce that something **happened**. It can never
  announce that something **failed to happen**. This is the pull-only wall stated exactly, and it is
  structural rather than an omission.
- **A green check means the Run finished, not that anything happened.** A Run whose every Step was
  skipped as already recorded completes and exits 0. Nothing gets a distinct exit code for it — a code
  never spans two outcomes, and "did nothing" is not an outcome — so it is read in the Dispositions,
  on the summary page the projection now writes.
- **Silence is a fact you can read, not a state the tool asserts.** Wherever a Cadence is glossed, the
  last Journal entry is rendered beside it — `03:00 UTC every Monday · ≈4.3 runs/month · last ran 41
  days ago` — and the human does the subtraction. `hyper` never says *overdue*: any threshold would be
  the tool introducing a claim of its own on a surface built to make that impossible, and a missed
  window is documented executor behaviour rather than a fault. Where no Store is reachable the gloss
  degrades to `last ran: unknown (no Store)`; `review` reads the Journal when it is there and does not
  require it, which keeps the offline authoring loop intact.
- **Two executor failures `hyper` can neither see nor prevent.** A scheduled workflow auto-disabled
  after 60 days of repository inactivity produces no run, no error in the Actions tab and no banner —
  only GitHub's own warning email, addressed to whoever last enabled it. And an oversized job summary
  is dropped silently. Both are named in the spec rather than mitigated, because a tool that only ever
  renders cannot see a run that did not happen.
- **There is no `--notify`, no webhook URL anywhere in any artefact, and still no update check.** The
  three globals stand.
