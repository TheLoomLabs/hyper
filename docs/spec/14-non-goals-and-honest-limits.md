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
asserts is that the record was consulted, never that the world was. The test decides per Record
(ADR-0056), so this is paid per member rather than per Step: one hand-deleted member of a list is
skipped while the rest of the list is served correctly, and the Run says nothing about the one.

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

Seventeen victims stand at it, each a thing an author can want, describe precisely, and not write:

- **OIDC federation.** `hyper` reads credentials and never acquires them (ADR-0007), so a federated
  cloud reached from CI needs a long-lived credential in the executor's secrets — worse than the
  ecosystem norm, and stated as such where it lands (§11).
- **Request signing, and with it every hyperscaler.** An Auth scheme is a placement and never a
  computation over the request being placed into (ADR-0031), so AWS SigV4 and its relatives are
  unwritable. This is the largest thing behind the wall: AWS needs signing, and GCP and Azure need the
  token exchange below, so no hyperscaler API is reachable at all. Nothing about it is cheap to route
  around — a signature is not a header a human can put in an environment variable.
- **A token `hyper` fetches.** OAuth2 client credentials, token exchange, and refresh are all a scheme
  performing a request of its own (ADR-0031): a call no Operation declared, no Bound counted, and no
  Disposition recorded, reaching a host named nowhere a reviewer reads. What is authored instead is a
  token obtained out of band into an environment variable, on ADR-0007's shape.
- **A client certificate.** mTLS is a property of the connection rather than a position in the request,
  so it cannot join a set defined by request positions (ADR-0031) — and its private key has no home
  either, `hyper` having no filesystem Capability (§12).
- **An API whose secret lives in the URL.** A credential occupies a request header and nothing else
  (ADR-0031), which is what makes *no secret ever appears in a URL* mechanically true rather than a
  convention. Slack's incoming webhooks are exactly that shape, the whole path being the credential.
  The workaround is usually a token-authenticated API beside the webhook, which is the better artefact
  anyway; where there is none, there is none.
- **Load-shaped retry.** Retry follows only a failure that provably preceded the request (ADR-0018),
  so an API that fails under load, with 5xx, on an effectful call, and genuinely wants retrying, has
  no route but waiting for one to ship.
- **Disjunction in a selector.** A predicate list is always AND, and there is no `or` (§12,
  ADR-0022).
- **A regular-expression match.** `starts_with` and `ends_with` are the bounded form of prefix and
  suffix matching and the whole of what exists (§12, ADR-0022).
- **Arithmetic on a response field.** There is no expression language to compute one in (ADR-0022).
- **A field timestamped as an epoch integer, compared by time.** `older_than` and `newer_than` read a
  value as a timestamp and an integer is a number, so a field holding `1754478199` Refuses rather than
  comparing (§12, ADR-0035). Reading it as an epoch would mean choosing seconds or milliseconds with
  nothing in any artefact saying which, and a heuristic on magnitude is `hyper` guessing about the world
  on the surface that decides what a `destroy` reaches. There is no arithmetic to convert with either
  (ADR-0022). A Manifest can project such a field and every surface renders it; no predicate can compare
  it. Unlike the rest of this list the cause is a value rather than a missing primitive, and what would
  buy it back is a format declared on the projection — which is an output schema, refused where the
  projection is stated (§3).
- **A written value that depends on when the Run happens.** No artefact names the current instant in a
  value position, no arithmetic computes over one, and no invocation supplies one (ADR-0022, ADR-0008),
  so a date a Step writes is the literal its author wrote — and a Procedure on a Cadence writes that
  same literal at every occurrence until somebody edits the artefact and puts it back through review. A
  predicate does name the instant, relatively, in a filter position, and the Run fixes it once for all
  of them (ADR-0034); the two are different positions and only the first is behind this wall.
- **A request body that is not JSON.** A `body:` is a mapping serialised as JSON and nothing else (§3),
  so a form-encoded POST, an XML payload, or a raw upload has no route but waiting for one to ship.
- **An API that answers in anything but JSON.** The response object's `body` is a parsed JSON body and
  is absent where the answer is XML, HTML, or bytes (§12, ADR-0040) — so such an API is callable and its
  answer is unprojectable, and everything an Operation records of it comes from the status, the headers,
  and the certificate. The absence is deliberate rather than a gap: it is what lets a check against a
  web page work at all.
- **A command that outlives the built-in's deadline.** The `shell` Provider declares one hour on every
  Operation and no artefact downstream may raise it (§12, and the override rule below), so a migration
  that takes four hours is unwritable as a Step. `hyper` is the Provider author here, and a deadline it
  guessed too low is corrected by a release rather than by an edit.
- **A pinned third party that needs to move.** The runner image and the `actions/checkout` commit the
  projection names are compiled into the binary (§11, ADR-0046), and a generated file is byte-exact
  against what `project` would write now (§10), so a hand-edit is caught rather than kept. When that
  action carries a vulnerability, or GitHub retires that image, every repository waits for a `hyper`
  release and has no route to repair in the meantime. Unlike the rest of this list what cannot be
  written is a line in a *generated* file rather than an artefact or a Provider, and what is waited on
  is a *third party's* clock rather than `hyper`'s — the wait begins whether or not `hyper` has
  anything of its own to ship.
- **An API paginated by a URL it hands back.** Pagination reads a cursor or counts pages (§12); a
  `Link` header or a `next` field is reach arriving from data, which no rule in the model permits
  (ADR-0024, ADR-0029). An API offering only that form is unwritable, and unlike the rest of this list
  it is unwritable on purpose rather than for want of a primitive.
- **An effectful Operation whose API answers anything but `2xx`.** A `mutate` or `destroy` completes on
  `2xx` and halts on everything else, a `destroy` on `404` besides, and no artefact declares otherwise
  (§6, ADR-0050). So a create answering `303 See Other` is unwritable, and so is one against an API that
  answers a transient `5xx` mid-poll, every call being judged and retry reaching only failures that
  provably preceded the request (ADR-0018). It is the one entry here that costs nothing on the other
  Kind: a `read` never halts on a status at all, which is what makes the same rule the reason `uptime`
  is writable rather than a further thing it cannot say.

The process by which those sets grow — who adds a member, and when — is undecided, and §12 records it
as undecided rather than answering it.

One further cost of the Auth schemes is a rendering loss rather than an unwritable Provider, and it
belongs beside that list rather than in it. A credential slot holds one opaque string, so where a
vendor bundles an identifier into it — Proxmox issues `root@pam!hyper=<uuid>`, of which only the UUID
is secret — the identifying half is suppressed with the secret half and no surface reports which token
a Run used. Splitting it would need a Manifest composing two slots into one value, which is the
Manifest choosing the placement that ADR-0031 gives to the scheme.

A second cost sits beside that list rather than in it, and it is not a thing `hyper` has failed to
ship. **A Manifest's declared facts are the Provider author's, and no artefact downstream may override
one.** A Definition claims Kinds and a Step supplies arguments and a Bound; neither may restate what
the Manifest said about the Operation itself — its Kind, its Repeatability, its deadline, its
concurrency limit — because those are the facts §6 gives the Provider author precisely on the ground
that the Definition author would be guessing at them, and an override arriving at the Step is
authority arriving after review (ADR-0008).

Repeatability is where that costs something visible. A Manifest omitting `repeatability:` on an
effectful Operation declares run-once, so that Operation cannot appear in a Procedure carrying a
Cadence at all (§4, ADR-0038) — and where the Manifest was installed rather than authored here, it is
verified by digest (§11), so editing it locally breaks the verification instead of fixing the Provider.
The correction belongs to whoever wrote the Manifest, and there is no local route to it. What `hyper`
gives back is the timing: `cadence-run-once` names the Operation and the file at `check`, offline,
before anything is projected or run, rather than at the second occurrence some night after the first
one worked.

The closed grammars charge in the same currency without extending that list. A Cadence is UTC-only
cron, with no field, no flag, and no file that could name a zone (§10, ADR-0005, ADR-0014), so *3am
my time* is unexpressible: what is authored instead is the UTC hour that means it today, and it stays
that hour when the clocks move.

## What the record does not reach

**There is no adoption.** An Asset is a Record of something `hyper`'s own effect reached, and that
effect is the whole test (§7, ADR-0025, ADR-0032). Nothing observed becomes one, because promotion is
reconciliation and the domain model declined it: an effect reaching a thing is a fact about what
happened, where adoption is a claim about the past. A resource created by another tool is therefore
reachable only by literal identifier, one `values:` entry at a time, for as long as it stands — an
effectful selector has nowhere else to reach (§5, ADR-0027) — and naming it does not make it an Asset;
acting on it does. Moving existing infrastructure under `hyper` means recreating it through a Step, or
carrying an enumerated list forever; the same holds for recovering an Orphaned Asset, where restoring
the deleted Definition is the other way back (§7, ADR-0012).

**A literal identifier is a Record name nothing can check.** Every other Record name is a
Manifest-declared field of an upstream response; a `values:` member is authored, and where a `destroy`
over one opens the series it ends, the author's spelling becomes the name (§7, ADR-0033). Where that
spelling is not what the Manifest's `identity:` path would have projected — an API that deletes by name
while its identity is an internal id — the Tombstone lands in a series that describes nothing, and a
real Asset series for the same resource stays standing and reads alive. `check` would need to know what
the API returns to catch it, which is the oracle §4 states it does not have. It is the same under-reach
as an Asset a selector misses, arriving through a literal rather than a predicate, and it is bounded the
same way: by the review of the artefact that names it.

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

**`shell` is the one Capability whose reach no grant bounds.** An `http` Operation reaches the hosts its
Target granted and nothing else, checked before the Run and enforced at the call (§4, §12). A command
reaches whatever the machine reaches: any host, any file, anything already in the environment. The
Target it binds is class-local, and such a Target's host grant governs its `http` Operations and governs
nothing about a command (§12, ADR-0024). What bounds a shell Step is the words a reviewer read in the Procedure —
the executable among them, which may not arrive by reference (§3, ADR-0051) — plus, on an `opaque`
`destroy`, the two opt-ins and the population the author wrote down (§5, ADR-0053), and nothing else in
the system. Which is why the
Capability is granted to no Extension, and why *a third party can never ship a Provider that runs
commands on your machine* is the honest form of that guarantee rather than *nobody can*. The blast
radius of such a Step is stated with the accurate word: unbounded, carrying no Bound at all (§5).

**A command's structured output is recorded as a blob.** `stdout` and `stderr` are text and are never
parsed (§12, ADR-0052), so a command answering in JSON is one string in one field and no path reaches
inside it. §7 says a blob nobody reviews has no business on a branch whose whole point is that it can
be read, and the `shell` Provider bends that rule by construction — one more reason it is the escape
hatch rather than the road. The cost lands unevenly by Kind and the asymmetry is the part worth
knowing: a shell `read`'s output is an Observation, and Compaction reclaims its interior versions; a
shell `mutate`'s is an Asset, and Compaction never removes one, so that output is on the branch
permanently. No cap stands between a chatty command and the Store, a byte limit being a number `hyper`
would be guessing at on a Provider that knows nothing whatever about the command (ADR-0045).

**A command that prints a credential puts it in the Store, and nothing notices.** ADR-0007's *`hyper`
never stores a secret* holds because every credential `hyper` handles occupies a position `hyper` owns
— a scheme's header, a declared slot, a `secret:` field — and it is suppressed by that position rather
than by anything recognising a value. A command's stdout is not such a position. §11 keeps `hyper`'s
own resolved credentials out of the child, so this is never a secret the repository named; it is one
the command obtained elsewhere and printed. The built-in declares no `secret:` and could not usefully
declare one, `hyper` being the Provider author and knowing nothing about the output (§3). This is the
one place the headline claim is qualified rather than merely bounded.

**No pipe, no redirection, no glob, no `&&`.** A `command:` is a list of argv words `hyper` execs
directly, with no interpreter in between (§3, ADR-0051), so `aws … | jq …` is not writable and neither
is `cmd > file`. What is written instead is a script in the repository, invoked as one word — which is
the substitution this limit is really making: the shell logic moves from a line nobody reviews into a
file that is reviewed like any other, at the cost of being a file at all.

**An interrupt during a shell Step can wait an hour.** §6's first interrupt drains, and the drain is
bounded by the Operation's deadline, which on every shell Operation is the one hour §12 fixes and
nothing downstream may lower. Where §6 calls that *a bounded wait* it was written with a
thirty-second HTTP deadline in view.

**A second interrupt leaves the command running.** The child is in a process group of its own, which is
what makes draining true of a shell Step at all (§6); a second interrupt kills `hyper`, which is then
not there to kill the group. `hyper` never claims to have stopped a command it started, and the Journal
entry it leaves open says only that the Run's outcome is unknown — which is accurate about the Run and
says nothing about the process.

**A command sees the environment `hyper` was invoked with.** What `hyper` removes from the child is
exactly the variables a Target declaration names as a credential slot, which it knows positionally
(§11). Everything else passes through, `hyper` neither reading nor recording it, so a credential the
repository does not name is reachable by any command and appears on no surface. The rule cuts the other
way too: a command needing a credential that *is* named cannot have it, and what is authored instead is
a second variable holding a second secret, which is one more thing outside the record.

**Bound to `local` itself, that removal is empty.** A declaration named `local` carries no `auth:` block
at all (§3, ADR-0041), so it names no credential slot and there is nothing positional for `hyper` to
take out of the child — the whole environment passes to a command run against the one Target a
repository is likeliest to bind. What buys the removal back is a second class-local declaration naming
the slots, which is the same plurality the `opaque-destroy:` opt-in uses to stop being one switch over
every command in the repository. Both are reviewed artefacts an author writes or does not, and neither
is a default.

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
surface. **Nor does a row show a Record's intermediate states**: a row is one Record read at its two
ends (ADR-0058), so a Record changed three times and then destroyed between two Runs renders as
`destroyed`, and the only admission that anything sat in between is the gap in `ORDINAL`.

**A large or multi-line projected value never renders on a Comparison.** `FIELDS` shows a value whole
or shows `changed` (ADR-0059), which is the only rendering that cannot show two different values as
identical bytes — but it means a `shell` `read` whose `stdout` moved reports that it moved and nothing
about how. The Store has the bytes and `hyper show` reads them back; the surface built to be scanned
is not where a Provider that projects unparsed text (ADR-0052) is legible, and `--json` carrying every
value whole is the way out for anything automated.

**Half of what there is to see costs infrastructure.** The surfaces that come before a Run are free:
`check` and a Definition review resolve no credential, reach no network, and invoke nothing (§4, §8),
so an artefact can be verified and read from a clone and nothing else. One part of a review is not
free, and it is the part that says what moved: the range it opens against is the revision the last Run
recorded (§8), so a Procedure nobody has run yet is reviewed with no range and a gutter that marks
nothing. The Comparison is not free at all. It
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

**No ad-hoc invocation.** There is no way to invoke one Operation directly: every Run is a Run of a
Procedure, and a one-off act against a credentialled Target is an artefact you have not written yet
(§9, ADR-0036). Nothing grows to cover it — a Probe reaches `local` and `read` alone and is not a Run
(ADR-0009) — so the price is real and it is the ritual working: author, check, review, run. It is not a
ceiling victim; nothing here is unwritable, only unwritten.

**No executor axis.** `hyper` projects onto one runner, GitHub-hosted x86-64 Linux, and there is no
`runs-on` setting, no platform field, and no flag (§11, ADR-0046). A repository whose runners are
self-hosted, ARM, or pinned to an older image cannot be projected onto them. What stands in for a
Cadence there is `hyper run` invoked by whatever clock that executor already has — a Run in every
respect except the one this costs: it reads its own environment and records `local` and `manual` (§12),
so the Journal says a person ran it on a laptop, and the Cadence's *last ran* is read against entries
that never name the clock that fired. The alternative was a field that lets two repositories run two
binaries against one Store, which is the skew the pin exists to prevent arriving through packaging
rather than through policy.

**No continuous monitoring.** What a Cadence claims is periodic checking against a floor the executor
sets, with best-effort delivery and no invented window (§10, ADR-0005). A sub-minute prober is a
hosted service rather than a stateless binary an external clock invokes, and calling this monitoring
would promise a coverage guarantee no part of the design supplies.
