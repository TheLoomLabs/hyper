# §6 — Execution

A Run begins with `check` re-run in full with nothing skipped (§4) and with the credentials of every
Target it may bind resolved once (ADR-0007); no Step starts until both have happened. This chapter
states what happens after that: the order Steps go in, what re-running one means, what a condition
may read, what runs concurrently, what a failure does to the rest of the Run, and the three outcomes
all of it ends in.

## The sequence

Steps run in written order. There is no dependency edge anywhere in the model, declared or inferred,
and no topological sort, no level scheduling, and no parallelism between Steps (ADR-0002). A Step
referencing an earlier Step's Record reads what that Step wrote in this Run, and a reference to a
later one is a load error rather than a race (§3).

A Procedure invoking another does not start a second Run. The invoked Procedure's Steps are Steps of
the one Run, recorded under a path — `deploy.provision.create-vm` — and a halt inside a nested
Procedure is a halt of the whole. One Run has one outcome, one Journal entry, and one exit code
however deep the invocation goes. The invocation graph is static, so a cycle is rejected before the
first Step and no depth limit exists (ADR-0002).

## No inputs

A Procedure is fully bound: `hyper run <procedure>` and the tool call behind it carry no argument
that can change what the Procedure does, because every value a Run needs is already written in the
artefact (ADR-0008). What the invocation does carry — a Secret sink path, a dry-run marker, output
formatting — is a property of the occasion and never authority: none of it can change which
Operation runs against which Target, or how many Records it may affect.

## No resumption

There is no resume. No `--from`, no downstream-reset, no run-state file, and nothing a halted Run
leaves behind for a later one to continue out of: re-running the Procedure is the retry (ADR-0003).
What a re-run does to a Step that already ran is decided by the Repeatability below rather than by
any record of where the last Run stopped.

What a Step produces is written as the Step completes and before the next Step starts, so a crash
loses at most the Step in flight and a fresh Run reads exactly the state a resumed Run would have
rebuilt.

## Repeatability

An Operation's Repeatability decides what a re-run does to a Step that already ran. It takes one of
three values — `repeatable`, `skip-if-recorded`, or, undeclared, run-once — named here and defined
in §12. It is declared in the Manifest and never inferred, and it is a property of the Operation
rather than of the Step calling it: the Provider author knows whether invoking it twice is intended
and the Definition author would be guessing.

A Step skipped under `skip-if-recorded` is one whose Asset still stands, which is a fact the head
version of its Record series carries (§12). A series whose head is a Tombstone stands for nothing,
so create, destroy, and create again is three Runs that each do what they say rather than a third
that reports `completed` having built nothing (ADR-0011).

Run-once refuses on evidence rather than on suspicion, and the evidence is what the Journal (§7)
holds for that Step: where no Run records it as *ran* or as *attempted, outcome unknown*, it runs;
where one does, it Refuses (`run-once-recorded`, §12). A Step the Journal only ever records as
*never reached* therefore runs on a re-run — without which one run-once Step would make a whole
Procedure permanently un-re-runnable after any halt, and with no bypass (ADR-0001) the only exit
would be an edit to a reviewed artefact.

## Conditions

A `when:` condition reads Records produced by earlier Steps of this Run, and nothing else: not the
world, not another Definition's Records, and not another Run's. A fact from elsewhere is a `read`
Step — it costs one line, it records what it read, and it occupies lines the gutter annotates beside
the Step it decides (§3). Every fact that influenced a Run is therefore visible twice over, in the
artefact and in the Run's own Records.

A Step whose condition does not hold is skipped by condition, which is a different Disposition from
skipped as already recorded because only the latter is Repeatability evidence.

## Expansion and concurrency

A Step's Expansion (§5) resolves before the Step runs, in a deterministic order fixed by Record
name. *Which three of the five* therefore has an answer, and a re-run attempts them in the same
order.

Concurrency is a function of Kind and is fixed by `hyper`: a `read` Step's Expansion runs
concurrently, and a `mutate` or `destroy` Expansion runs strictly serially. There is no authored
knob, no flag, and no environment override anywhere in it. How much of a concurrent Expansion runs
at once is the Operation's Manifest-declared concurrency limit (§3), since the Provider author is
the one who knows where the API refuses. Serial destruction is what makes *three of five, then halt*
a determinate fact a reviewer can read rather than a race.

All concurrency lives inside one Step's Expansion; two Steps never overlap (ADR-0002). An
Expansion's count is also where a Bound becomes decidable: an effectful Step whose Expansion
resolves to more Records than its declared Bound Refuses before the first call (`bound-exceeded`,
§12), which is the runtime half of the check §4 states statically.

## Errors and deadlines

An error halts the Run. There is no per-Step failure suppression — no `allowFailure`, no
`continueOnError`, nothing an author can write to silence one. Whether an unreachable host or a 503
is a result or an error is the Provider's declaration, made in the Manifest the way Kind is: a
monitoring Provider records *down* as an Observation, and `hyper`'s own rule stays one line.

Within a single `read` Step's Expansion, errors drain: every item is attempted, every Observation
that succeeded is recorded, and the Step then fails with the rest of the results already on disk. An
effectful Expansion is serial and stops at the first error, with everything it confirmed before that
recorded.

A deadline is declared per Operation in the Manifest; there is no whole-Run deadline, which could
only ever fire mid-effect. A deadline reached on a `read` fails the Step. Reached on a `mutate` or
`destroy` it is ambiguous — the call went out and no answer came back — so the Run halts `failed`
and the Step's Disposition is *attempted, outcome unknown*: no Tombstone, which would be a lie in
the safe direction, and no silence, which would be a lie in the other. Nothing retries its way out
of that state, since a retry Pattern follows only a failure that provably preceded the request
(ADR-0018).

## Halting

A halted Run leaves what it did. Nothing is compensated, rewound, or removed from the record: a
`destroy` Step that confirmed three of five Assets leaves three Tombstones and two live Assets and
stops there, and the next Run reads exactly that (ADR-0011). A Tombstone is written on confirmed
destruction only, one per Asset as each confirms.

## Signals

The first interrupt drains: the Step in flight finishes, no further Step starts, and the Run closes
its own Journal entry `failed` and exits 130 (ADR-0015). For a serial destroy that is a bounded
wait, and it turns most cancellations into a stop that is recorded in full rather than into an
ambiguity.

A second interrupt kills the process, and what that leaves is an open Journal entry rather than no
entry at all — the next effectful Run closes it `failed` with the in-flight Step marked *attempted,
outcome unknown* (ADR-0003). There is no reaper, no daemon, and no heartbeat: an abandoned entry is
noticed by the next Run that looks. Draining is a laptop property; an executor that kills after a
short grace period lands in the open-entry path instead, which is what that path is for.

## The lock

A Run holds a lock on the Store for its duration: exclusive if it contains any effectful Step,
shared if every Step is `read`. Which one it takes is statically decidable from the Kinds §4 already
computed, so a five-minute monitoring cadence is not starved behind a forty-minute provision.
Contention is neither a Refusal — no guardrail declined — nor a failure of the work, and it
collapses into `failed` with an exit code of its own (§12).

Only an effectful Run closes another Run's open entry. A read-only Run holds the shared lock, so it
can find a live effectful Run's entry open with no way to tell it from an abandoned one; it reads
and never reaps. The lock is one filesystem's, and two executors share no filesystem: what stands in
for it across runners is the projection §10 states.

## Dry-run

A dry-run performs the reads it reaches and stops rather than simulating an effect (ADR-0010). Those
reads really happened, so they record Observations like any other, and the Run writes a Journal
entry (§7) marked as a dry-run and carrying where it stopped. That entry is never a Comparison
baseline: a Comparison reads back to the last non-dry Run (§8, ADR-0010).

## The outcome triple

Every Run ends in exactly one of three outcomes — `completed`, `refused`, `failed` — named here and
defined in §12. `refused` is §5's: a guardrail declined a Step before any effect reached the world.
`failed` is the world resisting or the Run being stopped, which is where every halt above lands — an
error, a deadline, an interrupt, lock contention, an entry closed by a later Run. `completed` is
neither.

A Run that halted at the third Step of nine is `failed`, and what it did before it halted lives in
its Records and its Dispositions rather than in its outcome.

## Disposition

What each Step did in a Run is its Disposition, held by the Journal (§7) rather than by any Record:
ran, skipped as already recorded, skipped by condition, refused, never reached, or attempted with
outcome unknown. The six are named here and defined in §12.

The two skips are distinct because only one of them is Repeatability evidence. Skipped as already
recorded is the fact a later Run's Repeatability test reads; skipped by condition ran no such test
and says nothing about what the world holds. Collapsing them into one value would make the second
look like the first to every later Run.

## What completion does not mean

A Run whose every Step skips completes and exits 0. Nothing in the outcome or the exit code
distinguishes it from a Run that did all the work; the Dispositions are where the difference is
legible, and reading them back is the Comparison's job (§8) rather than the shell's.

The same holds one layer down. `skip-if-recorded` trusts the record over the world, so an Asset
somebody deleted by hand is skipped and the Run reports `completed` with nothing standing. That is
the price of having no drift detection, and designing around it is the reconciliation engine `hyper`
declined to build (ADR-0010). Both are named here as the honest limits they are, carried forward to
§13.
