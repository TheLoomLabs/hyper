# §6 — Execution

A Run begins in a fixed order, and no Step starts until all of it has happened. The version pin gate
runs first and the Store is located second, the two paths that decline before a Run is identified at
all (§9, §7). Then `run.json` is written and pushed, which for an effectful Run is also the Store sync
(§7) — so every gate below declines into an entry that already exists and has already reached the
remote. Then the Store files the Run must read are checked for a schema version above this binary's
(`store-schema-unsupported`, §12), over the Journal and the Record heads under the (Definition, Target)
pairs the Procedure makes. Then `check` is re-run in full with nothing skipped (§4). Then the
credentials of every Target the Run may bind are resolved once (ADR-0007). Then Step 1.

What is resolved is the slots the Run's bindings require rather than every slot each Target declaration
carries: presence is checked over the (Definition, Target) pairs the Procedure makes, exactly as slot
coverage is (§4) and exactly as the schema check above is, so a Target serving two Providers does not
oblige a Run to hold a credential no Step of it could send (`credential-absent`, §12). Two Run-start
gates scoped by one sentence is one rule; two gates scoped by two sentences is a second thing to keep
true.

Every one of those gates declines before Step 1, which is most of the closed `error_code` set rather
than a corner of it: `check` re-runs in full, so all of §4's static codes reach a Run this way
(ADR-0061). Where one declines, the Run is `refused` and the Journal entry holds what declined it
(§7). This chapter states what happens after all of it: the order Steps go in, what re-running one
means, what a condition may read, what runs concurrently, what a failure does to the rest of the Run,
and the three outcomes all of it ends in.

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

What a Step produces is written as each call confirms, and all of it before the next Step starts, so
a crash loses at most the Step in flight and a fresh Run reads exactly the state a resumed Run would
have rebuilt. A serial effectful Expansion therefore leaves what it confirmed rather than nothing:
three Tombstones where the fourth call never returned, which is the state the halting rule below
states and the next Run reads.

## Repeatability

An Operation's Repeatability decides what a re-run does to a Step that already ran. Its three values
are defined in §12, along with which of them each Kind may declare — two of the three read a
projection, so what an Operation projects fixes which of them mean anything on it (ADR-0037). It is
declared in the Manifest and never inferred, and it is a property of the Operation rather than of the
Step calling it: the Provider author knows whether invoking it twice is intended and the Definition
author would be guessing. Nothing downstream may override it, which is what leaves a Manifest omitting
it on an effectful Operation a thing only its own author can correct (§13).

`skip-if-recorded` skips while the Asset a call would produce still stands, which is a fact the head
version of that Record's series carries (§12). The test therefore decides at the granularity of the key
it reads, which is per Record and not per Step: a Step's Expansion holds one such series per member, and
whether the Asset a call would produce still stands is a question about that member (ADR-0056). A member
whose head stands is skipped. A member whose head is a Tombstone runs, the series
standing for nothing, so create, destroy, and create again is three Runs that each do what they say
rather than a third that reports `completed` having built nothing (ADR-0011). A member naming no
series at all runs, there being nothing for it to have been recorded as. A Step carrying no selector is
that same test over the one series it would write.

The decision is taken at each member's turn and not at Expansion, so every member the selector resolved
to is in `expanded_to` (§7) whether its call went out or not. Nothing is dropped for standing: what the
Store shortens is a `destroy`'s list, and only for what it knows is gone (§5). A `values:` list whose
members are filling in one at a time therefore reads as what it is — three authored, three expanded to,
one created this Run — rather than as a list the Store has been quietly shortening.

A Step whose every member skipped is *skipped as already recorded*. A Step any call went out from is
*ran*, which claims no count and never did: a `read` Step expanding over five hundred carries the same
value. Which of the two a mixed Step carries decides nothing about a later Run — the test reads the
Store's head version and never the Journal, so unlike run-once below it consumes no Disposition — and it
is a fact for a reader rather than an input to anything. Either way the identity set (§7) holds every
member, the skip test having concluded about each.

Run-once refuses on evidence rather than on suspicion, and the evidence is what the Journal (§7)
holds for that Step: where no Run records it as *ran* or as *attempted, outcome unknown*, it runs;
where one does, it Refuses (`run-once-recorded`, §12). A Step the Journal only ever records as
*never reached* therefore runs on a re-run — without which one run-once Step would make a whole
Procedure permanently un-re-runnable after any halt, and with no bypass (ADR-0001) the only exit
would be an edit to a reviewed artefact.

That exit is why a run-once Step and a Cadence cannot be authored together: nobody is present to make
the edit, and the Refusal is terminal, so the Procedure's remaining Steps stop with it at every
occurrence after the first. `check` refuses the combination before either runs (`cadence-run-once`,
§4, ADR-0038), which leaves run-once meaning what it says on a Procedure a person invokes.

## Conditions

A `when:` condition reads the Records earlier Steps of this Run acted on, and nothing else: not the
world, not another Definition's Records, and not another Run's. A Step whose call returned what the
head already held acted on its Record all the same and minted nothing (§7), so what a condition reads
is that head rather than nothing — a Record going unchanged is not a Record going missing. A fact
from elsewhere is a `read` Step — it costs one line, it records what it read, and it occupies lines
the gutter annotates beside the Step it decides (§3). Every fact that influenced a Run is therefore
visible twice over, in the artefact and in the Run's own Records.

A condition is evaluated before the Step's Expansion resolves, so a Step it does not hold for expands
over nothing, reaches no Target, and cannot Refuse on a Bound it never counted against.

A Step whose condition does not hold is skipped by condition, which is a different Disposition from
skipped as already recorded because only the latter is Repeatability evidence.

Where the Step a condition names wrote no Record in this Run — it was skipped by either Disposition, or
never reached — the condition does not hold and the Step is skipped by condition in its turn. It does
not fall through to the Store: reading the head there would be the condition quietly reading another
Run's Record, which is the one thing the rule above states it may not do. And it does not Refuse: an
earlier optional Step being skipped is an ordinary occurrence, and Refusing on it would make the
Procedure un-runnable with no exit but an edit to a reviewed artefact (ADR-0001). A skip propagates,
which is what a reader of the artefact would predict.

## Expansion and concurrency

A Step's Expansion (§5) resolves before the Step runs, in a deterministic order. *Which three of the
five* therefore has an answer, and a re-run attempts them in the same order.

The order is the one a reviewer can predict from what is in front of them, which makes it two rules
rather than one. Where the selector is an `over:` `values:` list, the artefact states the order and
that is the order: the list top-first, as authored, the members the Store dropped (§5) leaving the
survivors in the sequence the page has them in. Where it is `assets:` or `observations:` there is no
page to read an order off, so the Record `name` supplies one, sorted **by Unicode code point** — the
name the Store holds, never the percent-encoded path segment §12 builds from it to reach a file
(ADR-0044), and the same rule §7 sorts an identity set under rather than a second ordering. The sort
is total and needs no tie-break, one Expansion being one Target and one Definition, and two names
colliding case-insensitively being a Refusal the Store raises before a second series can exist
(`record-identity-collision`, §12).

Because it resolves first, a predicate handed a value it cannot compare Refuses here
(`predicate-type-mismatch`, §12) before any effect reaches the world, and a predicate list does not
short-circuit — every conjunct is evaluated against every candidate, so whether a Run Refuses does not
depend on the order an author happened to write two conjuncts in (ADR-0035). The two temporal operators
read against the instant on this Run's `run.json`, fixed at its start and shared by every Step and
every nested Procedure, so nothing a Pattern does moves what a later Step reaches (ADR-0034).

Concurrency is a function of Kind and is fixed by `hyper`: a `read` Step's Expansion may run
concurrently, and a `mutate` or `destroy` Expansion runs strictly serially. There is no authored
knob, no flag, and no environment override anywhere in it. How much of a concurrent Expansion runs
at once is the Operation's Manifest-declared `concurrency:` limit (§3), since the Provider author is
the one who knows where the API refuses — and a Manifest declaring none declares 1, so a `read` whose
Provider author said nothing runs its Expansion serially as well (ADR-0045). Serial destruction is what
makes *three of five, then halt* a determinate fact a reviewer can read rather than a race.

The order above is the order members are **dispatched** in, and it binds every Expansion: a serial
`mutate` or `destroy` runs in it outright, and a concurrent `read` starts its members in it, which is
what fixes the first ten of five hundred under a limit of ten. The order a concurrent Expansion's
calls *complete* in is not defined, and nothing derives from it — no Record, no Disposition, no
rendering. That is why the identity set §7 writes is sorted and `expanded_to` beside it is not: one is
a fact about a set, the other the account of a sequence.

All concurrency lives inside one Step's Expansion; two Steps never overlap (ADR-0002). The limit
therefore bounds the members of one `read` Step's Expansion that are in flight at once, and nothing
else. It does not reach a Pattern either: pagination, polling and retry are serial by construction (§3),
so a member is one call at a time from the moment it is dispatched until its last page, and *members in
flight* and *requests in flight* are one number rather than two that would have to agree. And where a
Step carries no `over:` there is no selector to resolve: the Step makes one call, which is a set of one
and inside any limit that has ever been written. A declared limit is a fact about the Operation, live
wherever a Step expands it, in the way a Kind is a fact about the Operation whether or not any Step
invokes it — nothing visible in a Manifest could tell an Operation that will be expanded over from one
that will not, expanding being the Step author's choice and not the Provider author's.

What declaring nothing costs is legible, and it is paid by the one person positioned to fix it: a `read`
over five hundred granted hosts whose Manifest omits the limit makes five hundred calls one after
another, each bounded by that Operation's deadline and by no whole-Run deadline. What buys the
concurrency back is one key in the Manifest, written by the author who measured the API — not a flag,
and not a number `hyper` chose on their behalf.

An Expansion's count is also where a Bound becomes decidable: an effectful Step whose Expansion
resolves to more Records than its declared Bound Refuses before the first call (`bound-exceeded`,
§12), which is the runtime half of the check §4 states statically.

## Errors and deadlines

An error halts the Run. There is no per-Step failure suppression — no `allowFailure`, no
`continueOnError`, nothing an author can write to silence one.

**A status is an answer, not an error** (ADR-0050), and which answers halt follows Kind, as what an
Operation projects does (§3, ADR-0037). No artefact declares it.

A **`read`** never halts on what came back. Whatever the status, the response object §12 states is
what the projection reads, so a monitoring Provider records *down* as an Observation and `hyper`'s own
rule stays one line. An API answering `404` for *absent* is describable for the same reason: the status
is recorded and a later Step's `when:` decides on it (§3), which puts *what counts as acceptable* on a
line the gutter annotates rather than in the artefact a reviewer reads least. What still halts a `read`
is its projection, below — `list_records` against a `401` has no `$.body.result`, and a collection that
was empty and a path that was wrong are not the same fact.

An **effectful** Operation completes on `2xx` and halts on everything else, a **`destroy`** completing
on `404` besides. A `mutate` or `destroy` the server did not accept did not do what the Step said, and
`hyper` does not read the shape of an error body to decide whether its own effect happened. `3xx` is on
the halting side because `hyper` follows no redirect, a redirect target being reach arriving from data
(ADR-0029). `404` completes a `destroy` because a `destroy` told there is nothing there has reached the
state it exists to reach, and because the alternative halts that Step identically on every re-run,
leaving an Asset that can never be Tombstoned and Steps after it *never reached* for good.

Where **no response arrived at all** — a refused connection, a name that does not resolve, a handshake
that failed — the response object is the host and nothing else (§3, §12). A `read` records an
Observation whose `status` has gone quiet, which is how *down* is recorded against a host that answers
nothing; an effectful Operation halts, no status being not `2xx`. Retry is unaffected either way: it
follows only a failure that provably preceded the request (ADR-0018), so no status is ever retried, and
an exhausted retry leaves the object above for the projection to read.

**An exit code is an answer too**, and the rule above is the same rule (ADR-0050). A `read` never halts
on it: the code is recorded, so a check script whose exit status *is* the finding is describable
without a second declaration saying what success means. An effectful Operation completes on **`0`** and
halts on everything else.

There is one asymmetry, and it is the `404`. A status code is a protocol's shared vocabulary and `404`
means *not there* in every API that speaks it, which is why a `destroy` completes on one. An exit code
is the command's own vocabulary and means whatever that command decided; nothing in any artefact says
which value stands for *already absent*, and the Provider author who would declare it is `hyper`, which
knows nothing whatever about the command. **A `destroy` completes on `0` alone**, and the trap the
`404` exists to avoid is closed here by the `over:` selector instead: a `values:` member the Store
already holds a Tombstone for is dropped from the Expansion before the command goes out (§5), so the
Step does not re-reach what it already ended.

Where the command **could not be started at all** — no such binary, not executable — the response
object is `command` and nothing else (§3, §12), which is the no-answer case one Capability over. A
`read` records the attempt with its `exit_code` gone quiet; an effectful Operation halts, no exit code
being not `0`.

Every call is judged, a Pattern's included. There is no final call a rule could privilege without
inventing one, and a Pattern may not change what an Operation does (§3).

A Step halted by a status carries no `error_code` — nothing declined, and a failure has none (§12) —
and its Disposition is *ran*. A response arrived, which is what that value means, and it is *ran*
whether the status was `400` or `500`: the residual doubt about whether a `500` left something behind
is real and is not what *attempted, outcome unknown* carries, that value meaning no answer came back at
all. What the Step names instead is the host it reached and the status it got, held by the Step file
(§7).

Within a single `read` Step's Expansion, errors drain: every item is attempted, every Observation
that succeeded is recorded, and the Run then halts with the rest of the results already on disk. After
the rule above that is one case rather than several — a projection that did not resolve, below — and
draining it is not a preference: an Expansion of a `read` runs concurrently and the order its calls
complete in is defined nowhere, so halting at the *first* failure would make which Observations were
recorded depend on the one thing nothing may derive from. An effectful Expansion is serial and stops at
the first error, with everything it confirmed before that recorded.

A deadline is declared per Operation in the Manifest; there is no whole-Run deadline, which could
only ever fire mid-effect. A deadline reached on a `read` fails the Step. Reached on a `mutate` or
`destroy` it is ambiguous — the call went out and no answer came back — so the Run halts `failed`
and the Step's Disposition is *attempted, outcome unknown*: no Tombstone, which would be a lie in
the safe direction, and no silence, which would be a lie in the other. Nothing retries its way out
of that state, since a retry Pattern follows only a failure that provably preceded the request
(ADR-0018).

A `shell` Operation's child runs in **its own process group**, and a deadline reached kills that group
with `SIGKILL` and no grace period. The group rather than the process, so a command's own children do
not outlive the deadline that bounded it; `SIGKILL` rather than a `SIGTERM` and a wait, because the
wait is a guessed constant on a Provider that knows nothing whatever about the command, which is the
ground `concurrency:` is 1 on (ADR-0045). A killed effectful child is *attempted, outcome unknown* like
any other deadline, which is the accurate thing to say about it. The one hour §12 fixes is `hyper`'s
patience and not a blast radius, and §13 states what it costs.

## When a projection does not resolve

A response can arrive and still not be readable: the Manifest declares a projection (§3) and a path
in it does not resolve against what came back. Nothing static decides this — no artefact states what
an API returns (§4) — so it is decided where it can be, against the response in hand.

Which path failed is what decides. A path a recorded field is read from resolving to nothing is
absence: the field is not written on that version, which is the fact the `exists` and `absent`
operators read (§12), and it is not silent — the bytes moved, so a version is minted and the field
going quiet renders as a change like any other (§8). A path a Record's identity is read from is an
error in the sense above, and so is the path an Operation of `series` cardinality reads its Records
from: without the first `hyper` cannot say which Record it is holding, and without the second it
cannot tell a collection that was empty from a path that was wrong — the *I recorded nothing* the
absent wire would otherwise be needed to diagnose (ADR-0017). Either halts the Run, and on a `read`
Expansion after that Expansion has drained (above): this is the one way a `read` fails at all
(ADR-0050), so the drain is what decides which Observations were recorded rather than a completion
order nothing derives from.

The Step's Disposition is *ran*: the call went out and the answer came back, and what failed is
`hyper`'s reading of it rather than the call. It is never *attempted, outcome unknown* — an
effectful Step that got this far holds a response saying the world was touched, and there is nothing
ambiguous to attach — and it is Repeatability evidence like any other *ran*, the call having happened
whether or not `hyper` could read the answer back. It carries no `error_code`, nothing having
declined (§12); what it names instead is the path that failed to project, and the surface that goes
out on is §8's.

A polling Pattern's `until:` fails the same way and is read here rather than in §5. Its predicate roots
at a response (§3), so a value it cannot compare is found after the call went out and there is no
Refusal available: the Run halts as above, carries no `error_code`, and names the field and what was
found in it. That is why two of the three predicate roots contribute a Refusal and this one contributes
a halt (ADR-0035).

An Operation of `series` cardinality projects many Records out of one response, so the failure can be
one member's: the path the Operation reads its Records from resolves, nine members project, and the
tenth's identity path does not. What projected is written, on a `read` Step and an effectful one
alike; the tenth is not, there being no identity to write it under; and the Run halts as above,
leaving what it did (ADR-0011). The drain rule above does not decide this — that rule is scoped to a
Step's Expansion, and a `series` response is one call the Expansion resolved to.

Nine Assets written are nine things `hyper` created and is accountable for, and discarded they are
nine resources standing that nothing in the record reaches and nothing reports — the Orphaned Asset's
hazard (§7) without the report that makes it survivable. What a half-projected response puts in doubt
is not the Records but the claim that they are all of them, and that claim lives in the identity set
the Step's Disposition carries rather than in any Record: §7 states what that set holds here, and §8
what reads it. Nothing in it turns on Kind, one response projecting the same way whichever Kind read
it.

## Halting

A halted Run leaves what it did. Nothing is compensated, rewound, or removed from the record: a
`destroy` Step that confirmed three of five Assets leaves three Tombstones and two live Assets and
stops there, and the next Run reads exactly that (ADR-0011). A Tombstone is written on confirmed
destruction only, one per Asset as each confirms — and a `404` confirms it, the Asset's resource being
gone whether this Run removed it or found it already absent (ADR-0050). Nothing on the Tombstone tells
the two apart: what `hyper` is accountable for is that the thing is gone, and recording *already gone*
as a fact about the Asset would be the reconciliation `hyper` declined to build (ADR-0010). The status
that confirmed it is on the Step file (§7), which is where a fact about the Run rather than about the
thing belongs.

## Signals

The first interrupt drains: the Step in flight finishes, no further Step starts, and the Run closes
its own Journal entry `failed` and exits 130 (ADR-0015). The drained Step's outcome came back, so its
Disposition is *ran* like any other completed Step's. For a serial destroy that is a bounded wait,
and it turns most cancellations into a stop that is recorded in full rather than into an ambiguity.

A second interrupt kills the process, and what that leaves is an open Journal entry rather than no
entry at all — the next effectful Run closes it `failed` with the in-flight Step marked *attempted,
outcome unknown* (ADR-0003). There is no reaper, no daemon, and no heartbeat: an abandoned entry is
noticed by the next Run that looks. Draining is a laptop property; an executor that kills after a
short grace period lands in the open-entry path instead, which is what that path is for.

The child's own process group is what makes draining true of a `shell` Step at all: in `hyper`'s group
a terminal's interrupt reaches the child directly and it dies at once, so the Step in flight would not
finish and the drain would be a sentence the implementation contradicts. Two costs follow and §13
carries both. The bounded wait is bounded by the Operation's deadline, which on every shell Operation
is **one hour**. And a second interrupt kills `hyper`, which — the child being in a group of its own —
leaves that command running with nothing watching it: `hyper` never claims to have stopped a command it
started.

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

Every Run ends in exactly one of the three outcomes §12 defines. `refused` is §5's: a guardrail
declined before any effect reached the world, most often before any Step existed. `failed` is the world
resisting or the Run being stopped, which is where every halt above lands — an error, a deadline, an
interrupt, lock contention, a Store it could not sync, an entry closed by a later Run. `completed` is
neither.

A Run that halted at the third Step of nine is `failed`, and what it did before it halted lives in
its Records and its Dispositions rather than in its outcome.

## Disposition

What each Step did in a Run is its Disposition, held by the Journal (§7) rather than by any Record
and drawn from the six values §12 defines.

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
