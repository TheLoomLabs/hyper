# §13 — Non-goals and honest limits

Most of what follows was named by the chapter that stated the mechanism it belongs to; the rest is
named for the first time here. Either way it is collected in one place, with what each one costs
stated rather than implied — so that a limit is something read in the spec rather than discovered in
use.

Nothing here is an apology and nothing here is a roadmap item. Each of these is a decision that was
made, and the reasoning for each is in the ADR it cites; what this chapter adds is the price.

## What `hyper` cannot know

**Whether a Manifest describes the API it names.** `check` compares reviewed text against reviewed
text and reaches no API, and `hyper` has no oracle for the question and claims none (§4, ADR-0025);
the cheapest evidence available narrows it rather than closing it (ADR-0017). The cost is that a
Manifest can be internally consistent, pass every static check, and still be wrong about the world.
What finds that out is a Run, and the Run that finds it out is a `mutate` or a `destroy` as readily
as a `read`.

**Whether the world still holds what the record says.** `skip-if-recorded` trusts the record over the
world, so an Asset somebody deleted by hand is skipped and the Run reports `completed` with nothing
standing (§6). Nothing reconciles the two, that engine being the one `hyper` declined to build
(ADR-0010), and no surface reports the divergence because nothing looked. What a `completed` Run
asserts is that the record was consulted, never that the world was.

**That a scheduled Run did not happen.** A workflow auto-disabled after 60 days of repository
inactivity produces no run and no error anywhere `hyper` can read, and an oversized job summary is
dropped without failing the step (§10). `hyper` is not running at either moment and has nothing to
notice with; what carries the first of them is an email the executor sends to whoever last enabled
the workflow, and what shows it inside the repository is the gloss's *last ran* read against the
declared Cadence, when somebody looks (§10, ADR-0021).

## The ceiling

When a Provider needs an effect `hyper` does not have, no amount of authoring routes around it: the
Provider is unwritable until `hyper` grows the primitive and ships one. The ceiling is a wall rather
than a slope, and it is what the closed sets §12 states cost rather than an accident of them
(ADR-0004).

Eight victims stand at it, each a thing an author can want, describe precisely, and not write:

- **OIDC federation.** `hyper` reads credentials and never acquires them (ADR-0007), so a federated
  cloud reached from CI needs a long-lived credential in the executor's secrets — worse than the
  ecosystem norm, and stated as such where it lands (§11).
- **Load-shaped retry.** Retry follows only a failure that provably preceded the request (ADR-0018),
  so an API that fails under load, with 5xx, on an effectful call, and genuinely wants retrying, has
  no route but waiting for one to ship.
- **Disjunction in a selector.** A predicate list is always AND, and there is no `or` (§12,
  ADR-0022).
- **A regular-expression match.** `starts_with` and `ends_with` are the bounded form of prefix and
  suffix matching and the whole of what exists (§12, ADR-0022).
- **Arithmetic on a response field.** There is no expression language to compute one in (ADR-0022).
- **A written value that depends on when the Run happens.** No artefact names the current instant, no
  arithmetic computes over one, and no invocation supplies one (ADR-0022, ADR-0008), so a date a Step
  writes is the literal its author wrote — and a Procedure on a Cadence writes that same literal at
  every occurrence until somebody edits the artefact and puts it back through review.
- **A request body that is not JSON.** A `body:` is a mapping serialised as JSON and nothing else (§3),
  so a form-encoded POST, an XML payload, or a raw upload has no route but waiting for one to ship.
- **An API paginated by a URL it hands back.** Pagination reads a cursor or counts pages (§12); a
  `Link` header or a `next` field is reach arriving from data, which no rule in the model permits
  (ADR-0024, ADR-0029). An API offering only that form is unwritable, and unlike the rest of this list
  it is unwritable on purpose rather than for want of a primitive.

The process by which those sets grow — who adds a member, and when — is undecided, and §12 records it
as undecided rather than answering it.

The closed grammars charge in the same currency without extending that list. A Cadence is UTC-only
cron, with no field, no flag, and no file that could name a zone (§10, ADR-0005, ADR-0014), so *3am
my time* is unexpressible: what is authored instead is the UTC hour that means it today, and it stays
that hour when the clocks move.

## What the record does not reach

**There is no adoption.** An Asset is a Record of something `hyper` created, and having been created
by `hyper` is the whole test (§7, ADR-0025). Nothing observed, imported, or named by hand becomes
one, because promotion is reconciliation and the domain model declined it. A resource created by
another tool is therefore reachable only by literal identifier, one `values:` entry at a time, for as
long as it stands — an effectful selector has nowhere else to reach (§5, ADR-0027). Moving existing
infrastructure under `hyper` means recreating it through a Step, or carrying an enumerated list
forever; the same holds for recovering an Orphaned Asset, where restoring the deleted Definition is
the other way back (§7, ADR-0012).

**An Asset a selector misses is abandoned silently.** The Bound catches a selector that reaches too
far and nothing catches one that reaches too little: an Asset whose field holds something other than
what the author's predicate expects is not expanded over, not acted on, and not counted. The Step
completes, the Run completes, and every Disposition is truthful — the Step did what it reached. Its
Definition still exists, so the Asset is not an Orphaned Asset and no rendering reports it as
unowned (§7, ADR-0012). What distinguishes *there were three and three were destroyed* from *there
were four* is reading the Record series, which is a thing a person does (§9), not a check any
artefact can carry: a Bound is a ceiling on what may be touched and never a statement of what was
expected to be.

## What the guardrails do not cover

**Blast radius is a count, not a severity.** A Bound counts Records and never weighs what happened to
each one, so a runaway selector is caught and a single correctly-Bounded call against the most
important row in the system is not (§5). What stands there instead is entirely static and entirely
before the Run: the two keys, the named-Operation requirement on `destroy`, and the review of the
Definition that claimed it.

**The Comparison prevents nothing.** It is an accountability instrument, never a guardrail, and it
reports what changed rather than what is wrong (§8, ADR-0010). It says *this differs from when we
last looked* and it never says *this differs from what we intended*, there being nothing in the
system that holds an intention to differ from. A surface that implied otherwise would have lied.

**A destructive Run started by mistake is not catchable at the keyboard.** No confirmation happens at
execution time, no per-Run approval exists, and nothing overrides a Refusal at invocation (§5, §9,
ADR-0001, ADR-0015). Nothing renders what a destructive Step would touch before it touches it either,
there being no prospective counterpart to the Comparison (§8, ADR-0010). The safety net is relocated
rather than absent — the two keys, the mandatory Bound, the named Operation, and the Definition
review, every one of them acting on the authored claim rather than on the world — and it is stated
plainly because the moment a person most wants a prompt is exactly the moment these decisions deny
them one.

**A Refusal is recorded in full only where there is a Store to record it in.** In the bootstrap case
— a repository whose Store branch has never been created — the Refusal renders, exits `77`, and
writes nothing (§9). A scheduled Run that refuses there therefore leaves no trace in the repository
at all, and what happened is in the executor's log or nowhere.

## What the record costs

**The Store grows monotonically, forever.** Compaction reclaims tree size and scan cost and nothing
from a clone, and there is no rollover to a fresh branch (§7, ADR-0011, ADR-0001). Every byte ever
written to the branch is fetched by the next clone of it: a repository that runs often enough, for
long enough, pays for its whole history every time somebody clones it, and no command in the tool
stops that.

**Reading an old commit's Store needs that commit's binary.** Checking out history moves the pin, and
every command that reads the record is gated on it (§11, ADR-0020). Reading a year-old Run therefore
begins with installing the year-old binary, by hand: nothing downloads one, and no flag reads the
record without one.

**There is no one list of everything that happened, in order.** The Comparison is three tables split
by actor — what `hyper` did, what the world did, what the code did — and that split is what makes
each row attributable; a single chronological scan is what it costs (§8). Nothing renders one, on any
surface.

**Half of what there is to see costs infrastructure.** The surfaces that come before a Run are free:
`check` and a Definition review resolve no credential, reach no network, and invoke nothing (§4, §8),
so an artefact can be verified and read from a clone and nothing else. The Comparison is not. It
renders one Run against the Run before it, its code table included — those facts being the Provenance
each of the two Runs recorded (§7, §8) — so everything on that side of the thesis is bought with two
Runs against real systems.

## What `hyper` never says first

`hyper` is pull-only. It has nothing to push from, no clock of its own, and no process alive between
invocations (§10, ADR-0021), so everything below is one decision seen from several sides: the
rendering is complete and hides nothing, and it exists at the moment somebody asks for it.

**A green check means the Run finished, not that anything happened.** A Run whose every Step was
skipped as already recorded completes and exits `0` like any other, and no exit code distinguishes
the two (§6). The Dispositions are where the difference is legible, which is why the job summary is
worth writing at all (§10).

**You cannot author *tell me when this fails*.** Nothing watches a Record for a value crossing a line
and nothing emits when one does. The notification you want is a Step you author — a Definition
against a Target that carries messages, reviewed, Bounded and recorded like everything else
(ADR-0021). It can announce that something happened; it can never announce that something failed to
happen.

**A stale pin is silent.** Nothing checks whether a newer version exists and no surface says you are
three releases behind (§11, ADR-0019, ADR-0016), an update check being egress performed on nobody's
behalf by a tool that otherwise reaches the network only where a reviewed artefact asked it to.

## The non-goals

These are decisions, not gaps. Each was taken for a reason recorded elsewhere, and none of them is
waiting on effort.

**No registry as a product.** `install` consumes a registry, and what a registry is beyond a place
bytes and checksums are published is not `hyper`'s concern: no publishing command, no account, and no
promise about a Provider's availability (§11). Finding a Provider, and living with whoever hosts it,
are the user's; the only claim `hyper` makes about fetched bytes is that they match a published
digest, never that what they describe is benign (ADR-0004).

**No `serve`, no daemon, and no remote API.** Nothing listens on a port and nothing outlives the
invocation that started it (§9, §10). The MCP surface is stdio, dies with its client, and offers no
asynchronous handle, so a twenty-minute provision is not practically runnable from it: that surface
owns the author→validate→observe loop and short effectful Runs, and long unattended work is a Cadence
on an executor (§10).

**No telemetry.** No exporter, no metrics, no trace context, and no logging framework (ADR-0016). The
Journal is the trace, so per-Step timing lives in it or nowhere and a duration is derived at render
from timestamps within one entry. A question about how the tool behaves across many Runs is answered
by reading the branch, by whatever reads it; nothing aggregates one and nothing is sent anywhere.

**No team features.** There are no accounts, no roles, no per-Run approval, and no shared server to
hold any of them (§5, §9, ADR-0001). Who may change what `hyper` does is decided by who may merge a
change to the reviewed artefacts, which is the repository's own review, and adding a second authority
axis inside the tool would be a way past a Refusal that no artefact records.

**No continuous monitoring.** What a Cadence claims is periodic checking against a floor the executor
sets, with best-effort delivery and no invented window (§10, ADR-0005). A sub-minute prober is a
hosted service rather than a stateless binary an external clock invokes, and calling this monitoring
would promise a coverage guarantee no part of the design supplies.
