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
the cheapest evidence available narrows it rather than closing it (ADR-0017, ADR-0108). The cost is
that a Manifest can be internally consistent, pass every static check, and still be wrong about the
world. What finds that out is a Run, and the Run that finds it out is a `mutate` or a `destroy` as
readily as a `read`.

The narrowing is worth stating with its own limit. A Probe reads a projection against a response the
author **supplied**, which is how the projection of an Operation no Probe may invoke is legible at all
(§9, ADR-0108) — and a Manifest that reads a supplied response correctly is right about that response
and not about the API. The file is a fixture, and a fixture written by hand can be wrong in exactly
the way the Manifest is wrong. What it buys is the class of evidence a test fixture buys: it catches
the path that addresses nothing, and it cannot catch the response that never looks like the one you
saved.

One face of it is **legible after the fact and the rest are not**, which is worth stating because it
is the only part of this limit an author can act on without instrumenting anything. A body's wire type
is the input schema's declared type (§3, ADR-0078), so an API that wanted `2592000` and received
`"2592000"` is a wrong `type:` sitting a few lines below the body in the same file — a reviewer reading
the rejection reads the edit. Where the same Manifest is wrong about a projection path or about an
Operation's Kind, nothing on any page says which line to change.

**Which of two values a stored one reads as.** A scalar is read against the schema at its position
rather than compared with it, at Expansion as at load (§3, §6, ADR-0081), so an `args:` reference into
a Record holding `"0755"` fills an `{type: integer}` input with **755**. That is ADR-0078's leading-zero
trap arriving where its defence does not: the authored half has the trap on the line a reviewer reads,
and this half has it in the Store, on a value nobody wrote and no surface shows beside the input it
fills. The alternative was binding the stored value's own JSON type, which would make a Refusal turn on
whether an API answered a string or a number — a fact about the API, and one this section opens by
saying `hyper` has no oracle for. The cost is paid to keep *satisfies* meaning one thing at both sites
of one check.

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

Twenty-two victims stand at it, each a thing an author can want, describe precisely, and not write.
The count read *seventeen* against nineteen entries until ADR-0081 counted them, ADR-0078 having added
two without moving the word — which is the shape §12's opening rule refuses in a closed set, arriving
in the prose that introduces one, and the reason the word moves with every entry now:

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
- **A plain-HTTP endpoint.** `hyper` requests `https://` and there is no second scheme
  ([ADR-0082](../adr/0082-the-scheme-is-https-and-there-is-no-second-one.md)): a `hosts:` grant
  enumerates hosts and carries no scheme, so there is no position in any artefact where one could be
  written. An internal service reachable only over plain HTTP is called through a `shell` Step or not
  at all. What buys the limit is that `tls` is then present on every response that arrived (§12), so a
  certificate going quiet means the host answered nothing rather than that the transport differed.
- **A client certificate.** mTLS is a property of the connection rather than a position in the request,
  so it cannot join a set defined by request positions (ADR-0031) — and its private key has no home
  either, `hyper` having no filesystem Capability (§12).
- **An API whose secret lives in the URL.** A credential occupies a request header and nothing else
  (ADR-0031), which is what makes *no secret ever appears in a URL* mechanically true rather than a
  convention. Slack's incoming webhooks are exactly that shape, the whole path being the credential.
  The workaround is usually a token-authenticated API beside the webhook, which is the better artefact
  anyway; where there is none, there is none.
- **A path segment holding a literal `?` or `#`.** `path:` is written as text and `hyper`
  percent-encodes it (§3), so neither character can be authored inline: a `?` there is a query written
  in the wrong key and a `#` is a fragment no request carries, and both are refused at `check`
  (`manifest-inconsistent`, §4, ADR-0107). Unlike the rest of this list the primitive is not missing.
  A hole's value arrives at Run time, is read against no path grammar, and is escaped like any other
  text, so the segment is reachable by naming it in an input — and that route costs what an input
  costs: every input an Operation declares is supplied by every Step that binds it (ADR-0081), so a
  fixed `?` in a path stops being the Manifest's and becomes an argument every call site writes out.
  What is behind the wall is therefore the *constant*, and the wall is the price of deciding offline
  the mistake that character is almost always a symptom of.
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
- **A request body that is not JSON.** A `body:` is serialised as JSON and nothing else (§3), so a
  form-encoded POST, an XML payload, or a raw upload has no route but waiting for one to ship.
  There is no back door either: `Content-Type` is one of the five headers `hyper` computes and no writer
  may name one (`header-reserved`, §4), so the wall cannot be walked round by sending JSON under
  another label.
- **A request body whose top level is not a mapping.** A `body:` is a JSON value tree and its root is a
  mapping (§3), so a batch API taking a bare array has no route. Everything below the root nests
  freely, so this is the narrowest entry on the list: one position, and only the outermost one.
- **An API with a genuinely optional parameter.** Every input an Operation declares is supplied, there
  being no `null` in the vocabulary, no key-omission syntax, and therefore no sink at which an
  unsupplied input could render (§3, ADR-0081). So a search endpoint taking any three of five filters
  is one Operation per combination, and a caller who wanted to leave one out writes a second Operation
  instead. It is the only entry here whose cost is **combinatorial** rather than flat: two optional
  parameters are three Operations and five are thirty-one, so the wall is cheap where the option is
  rare and prohibitive where an API is built on them. What buys it is that a request's structure is the
  Manifest's alone — the same sentence the entry below is bought by — and a Step's `args:` can no more
  remove a key than it can add one.
- **An API wanting a caller-supplied object in its body.** A template hole fills a scalar position, so a
  hole may not name an input declared `object` or `array` (§3, ADR-0078), and a body's structure is
  therefore always the Manifest's rather than partly a Step's. What buys the limit is what a reviewer
  gets for it: the shape of every request is readable off the Provider, and only its values move.
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

Its sibling check costs nothing here, and the contrast is what shows where this limit's weight actually
sits. A Cadence is refused over a Step whose Operation declares **secret output** on the same walk and
for a stricter reason — that Procedure works never rather than once (§4, ADR-0077) — and yet no limit
follows, because that remedy runs around the Manifest rather than through it: moving the Step into a
Procedure a person invokes with `--secret-out` is an edit to the consumer's own artefacts, available
whoever wrote the Provider — and the Run that follows it completes, the sink being written and the
value landing in it (§9, ADR-0148). What is unfixable above is therefore not *a Manifest fact a Cadence
forbids*; it is specifically that run-once's only remedy is the Manifest's own `repeatability:`.

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

**The member that collided on an identity leaves a resource nothing records.** An Expansion's members
are one Record identity each, and so are the Records one `series` response projects (§3), and where the
identity reads from the response the collision is
visible nowhere earlier than the answer that carries it — one call has already gone out and there is no
name of its own to write it under, so nothing is written and the Run halts (§6). The resource it created
stands with nothing in the Store reaching it, which is the Orphaned Asset's hazard without the report
that makes it survivable (§7, ADR-0012). Writing it as a further version of the series it collided with
would bury the earlier member's resource beneath it and disguise the fault as an ordinary update, and
inventing a name to distinguish them would be `hyper` minting an identity no Manifest declared. The
Journal names the member and the identity, and the arithmetic on the entry is honest — expanded to
three, concluded about one — so what is lost is a Record and never the fact that it is missing. It costs
one resource once, and it is the price of the offline half being as large as it is: every collision an
artefact can be read for is refused before anything is touched.

**A projected identity that collides with the Store costs a resource on every Run.** It is the same
halt against a different comparand — the name arrives in a response and the series it folds onto is one
an earlier Run wrote (§6, ADR-0072) — and the paragraph above prices the wrong thing for it. Those
collisions are between things this Run produced, and a Run that produces them again is a Run whose
inputs did not change; this one is between something this Run produced and something standing in the
Store, which stays standing. Nothing in `hyper` clears it and nothing remembers it: the Store is
append-only (ADR-0011), a Run is never resumed and reads no earlier Run's halt (§6, ADR-0001), so the
next Run reaches the same member, makes the same call, creates another resource nothing records, and
halts in the same place. Under a Cadence that is one orphaned resource per occurrence, indefinitely, and
the Cadence is the multiplier §5 has always called it — the same `*/5 * * * *` that makes a `destroy`
8,600× the Run it was reviewed as makes this a standing leak rather than an incident. What clears it is
a Manifest identity change, which is a reviewed code change and the same remedy §7 states for a `Foo`
beside a `foo` found any other way; until somebody makes it, every Run pays. The compensating fact is
that the Run is `failed` and loud on every occurrence rather than silent — this is not an Asset a
selector missed — and that the pre-call half of the same check (§6) has already refused every collision
whose identity `hyper` held before it called.

**An Asset a selector misses is abandoned silently.** The Bound catches a selector that reaches too
far and nothing catches one that reaches too little: an Asset whose field holds something other than
what the author's predicate expects is not expanded over, not acted on, and not counted. The Step
completes, the Run completes, and every Disposition is truthful — the Step did what it reached. Its
Definition still exists, so the Asset is not an Orphaned Asset and no rendering reports it as
unowned (§7, ADR-0012). What distinguishes *there were three and three were destroyed* from *there
were four* is reading the Record series, which is a thing a person does (§9), not a check any
artefact can carry: a Bound is a ceiling on what may be touched and never a statement of what was
expected to be.

**The record says the request never left and never why.** A Step that is *attempted, world untouched*
carries the host it was reaching, or the command it was starting, and nothing more (§7): a refused
connection, a name that did not resolve, a handshake that failed and a binary that was not there are one
entry in the Journal, and they are four different things to fix. The three transport members are
normative in §12 as that Disposition's boundary and are written on no Step file (ADR-0062), which is a
deliberate refusal of ADR-0050's rejected `error` member arriving through the Journal rather than the
response object. It is paid where it hurts most — an unattended Run under a Cadence, whose terminal
rendering nobody saw and whose entry is the whole account. The consolation is that the host is right
there and the four remediations all begin at the same place, and the fix if that proves too thin is a
closed enumeration ADR-0018 has already written.

**Two clocks can list two Runs in an order neither machine would agree with.** `runs` orders newest-first
on `started_at`, and there is no second axis for it to fall back on: a Journal entry is a Run, and Runs
have nothing to be ordered by except when they happened (§9, ADR-0065). Everywhere else the clock is
avoidable and is avoided — the Comparison orders by identity and refuses `written_at` for this reason
(§8), no rendering subtracts timestamps from two entries, and nothing derived from this ordering is a
check, a Disposition, or a Repeatability test. What it costs is a listing that can transpose two adjacent
rows written on two machines whose clocks disagree by more than the gap between them, and `hyper` does not
detect it, warn about it, or correct it: the record holds each writer's own stamp, and inventing an
authoritative clock to reconcile them is a shared service the tool does not have (§7, ADR-0006).

**Two effectful Runs across the laptop and a runner are serialised by nothing.** §6's lock is one
filesystem's and §10's `concurrency` group covers runner-to-runner, and this pair falls between them:
they can bind the same Target and mutate the same resource at the same moment. It is the same shape as
the two clocks above — two machines with no shared authority to appeal to — and the remedy is the same
one, a lock server ADR-0006 rules out on the ground that CI shares no hidden state with the laptop.

**What it does not cost is the record.** ADR-0076 settled that separately and completely: every Store
path carries the id of the Run that wrote it, so two overlapping Runs cannot write one path, neither can
take a write from the other, and a Run reaped while it was alive finishes on its own terms and leaves a
**contested** entry holding both accounts (§7). Every Run's own account of what it did survives the
overlap in full, and the surfaces show the disagreement rather than resolving it. So the exposure here
is the **world** and never the evidence — which is the one shape of this limit `hyper` can actually
stand behind, the tool's claim having always been about what the record says rather than about what it
could prevent.

Nothing detects the overlap, and nothing could without the lock: an open entry is indistinguishable from
an abandoned one by design (§6), which is why the reap is unconditional and why it had to be made
harmless rather than made accurate.

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
radius of such a Step is stated with the accurate word: unbounded, whatever `bound:` it carries — no
Bound at all on a `destroy`, where one is refused, and on a `mutate` a count of the Records it minted,
which the review flags as unbounded all the same (§5).

**A command's structured output is recorded as a blob.** `stdout` and `stderr` are text and are never
parsed (§12, ADR-0052), so a command answering in JSON is one string in one field and no path reaches
inside it. §7 says a blob nobody reviews has no business on a branch whose whole point is that it can
be read, and the `shell` Provider bends that rule by construction — one more reason it is the escape
hatch rather than the road. The cost used to land unevenly by Kind — a shell `read`'s output is an
Observation, which Compaction reclaims interior versions of, and a shell `mutate`'s was an Asset, which
Compaction never removes — and what closed that half is the effectful Operations no longer projecting
`stdout` at all (§12, ADR-0143). What is left is the `read`, and it is a question of volume rather than
of permanence: no cap stands between a chatty command and the Store, a byte limit being a number
`hyper` would be guessing at on a Provider that knows nothing whatever about the command (ADR-0045),
and Compaction reclaims those versions only where the repository declared a `retention:` — omitted,
nothing is ever removed (§3).

**A command that prints a credential puts it in the Store, and nothing notices.** ADR-0007's *`hyper`
never stores a secret* holds because every credential `hyper` handles occupies a position `hyper` owns
— a scheme's header, a declared slot, a `secret:` field — and it is suppressed by that position rather
than by anything recognising a value. A command's stdout is not such a position. §11 keeps `hyper`'s
own resolved credentials out of the child, so this is never a secret the repository named; it is one
the command obtained elsewhere and printed. The built-in declares no `secret:` and could not usefully
declare one, `hyper` being the Provider author and knowing nothing about the output (§3). This is the
one place the headline claim is qualified rather than merely bounded.

What bounds it now is Kind and Compaction rather than anything noticing. Only a shell `read` projects
`stdout`, so what a printed credential reaches is an Observation and never an Asset or a Tombstone
(§12, ADR-0143) — a version Compaction may remove where the repository declared a `retention:`, which
is a policy an author states rather than a thing `hyper` does. It is still in `git` history, which is
not editable, so what changed is that the branch's live tree can stop holding it; rotating the
credential remains the only remedy that is one.

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

**A command sees the environment `hyper` was invoked with, less what a declaration names.** What `hyper`
removes from the child is exactly the variables a Target declaration names — as a credential slot, which
it knows positionally, or in a `withhold:` list, which an author writes (§11, §3, ADR-0144). Everything
else passes through, `hyper` neither reading nor recording it, so **a variable no declaration names is
reachable by any command and appears on no surface**. That is the residue, and `withhold:` is what makes
it a line an author did not write rather than a thing that could not be written: the common case is a
wrapper's own credential — `OP_SERVICE_ACCOUNT_TOKEN` under `op run --` — which is strictly more
powerful than the token it was fetching and which `hyper` has no position for. The rule cuts the other
way too: a command needing a credential that *is* named cannot have it, and what is authored instead is
a second variable holding a second secret, which is one more thing outside the record.

**Bound to `local` itself, the positional half of that removal is empty.** A declaration named `local`
carries no `auth:` block at all (§3, ADR-0041), so it names no credential slot and there is nothing
positional for `hyper` to take out of the child — the whole environment would otherwise pass to a
command run against the one Target a repository is likeliest to bind. A `withhold:` on that same
declaration is what answers it directly, `local` being allowed one precisely here; the older answer, a
second class-local declaration naming the slots, is the same plurality the `opaque-destroy:` opt-in uses
to stop being one switch over every command in the repository, and it remains available. All are
reviewed artefacts an author writes or does not, and none is a default.

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

**The Store grows monotonically, forever, and it is paid for twice on two different curves.** There is
no rollover to a fresh branch (§7, ADR-0011, ADR-0001), so nothing ever gets smaller on either.

**A clone pays for the whole history, and Compaction reclaims nothing from it.** Git history is not
editable, so every byte ever written to the branch is fetched by the next clone of it: a repository
that runs often enough, for long enough, pays for its whole history every time somebody clones it, and
no command in the tool stops that.

**`hyper`'s own sync pays for the live tree, and pays it once per environment that lacks the branch.**
The sync is a depth-1 fetch (§7, ADR-0074), so it costs what the branch currently holds rather than
what it has ever held — but on a runner the branch is always absent, `actions/checkout` taking one ref
(§10), so **every scheduled occurrence of every Procedure fetches the whole live Store from scratch**.
This is the curve that recurs, and the clause above names the rarer act. Compaction does reclaim from
this one, which is the only place in the tool where it buys anything on the wire.

**Which of the two curves recurs was measured, and for a while it was the wrong one.** A bare
`git fetch --unshallow` in the deepen step inherits the wildcard refspec `checkout` leaves behind, so
until the refspec was named the runner took the Store branch's **entire history** before `hyper`
started — the clause above, on every Run, at whatever Cadence was declared (#246,
[ADR-0132](../adr/0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md),
[ADR-0134](../adr/0134-the-deepen-step-names-one-ref-and-what-deepens-the-code-branch-is-the-clones-own-boundary.md)).
The two curves are as stated because one line of the projection says so, and nothing about a runner
enforces it.

**What neither curve reclaims is the Journal**, and it is the term that grows with the Cadence rather
than with the world. Compaction removes interior Observation versions only, and a version is minted
only where the bytes moved, so a Record checked every five minutes and never changing sits at one file
— while each of those Runs writes an entry, and every entry stands forever (§7). At a five-minute
recurrence the Journal is the dominant term in what a runner fetches, it is untouchable by the one
command that reclaims anything, and the recurrence that makes it grow is the same declaration §10
projects.

**A clone that never held the Store gets Compaction's account one commit deep.** `git log` on the
branch is what says what a Compaction removed (§7), and where `hyper` created the branch itself the
history behind that commit was never fetched. It reaches exactly one clone — a `--single-branch` laptop,
whose owner opted out of the branch already — because a runner never compacts and an ordinary `git
clone` has the whole branch before `hyper` touches it (ADR-0074).

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
free, and it is the part that says what moved: the range it opens against is what the last Run that
read the artefact recorded (§8), so **any of the five reviewed artefacts nothing has yet run against is
reviewed with no range and a gutter that marks nothing** — a Procedure nobody has run, a Definition no
Step has named, a Target declaration no Step has bound. Two of them pay more than that. A Target
declaration, a Repository declaration and a Manifest carry no revision of their own, so their range is
resolved from the repository revision a Run recorded, and a Run that recorded `repo_dirty` supplies none
at all: an author who habitually runs against uncommitted edits gets no range on those three, the
alternative being a gutter that marks the wrong lines (§8). The Comparison is not free at all. It
renders one Run against the Run before it, its code table included — those facts being the Provenance
each of the two Runs recorded (§7, §8) — so everything on that side of the thesis is bought with two
Runs against real systems.

**And both surfaces are bought a third time, with the clone.** A range and eight of the nine code
classes read bytes at a revision the Store names, so a clone that does not hold the object renders
`not-in-clone` in place of what it would have said (§8, ADR-0071, ADR-0086). Two causes are repaired by
an act — a shallow clone deepens, a partial clone refetches, and a projected workflow deepens the
runner's before anything runs (§10) — and one is not: a code branch whose history was rewritten or force-pushed leaves an object the
Store names and nothing produces. The Store outlives the history it points into, which is the price of
`git hash-object` being the anchor at all; it buys a range that survives any number of commits, and it
cannot survive a commit that was unmade. `hyper` names no repair on that line for the same reason it
names no *overdue*: it cannot tell which of the three it is looking at without asking a remote, and a
review asks nothing.

**A fourth cause is the author's own, and it is not repairable either.** A Run reads the artefacts out
of the working tree, so a Run against bytes nobody committed records ids that were never written
anywhere (§7), and committing afterwards writes today's bytes under a new id rather than producing the
ones that ran. What that costs is the review of the next draft: the baseline is the previous draft, and
the previous draft is gone. It is the one of the four `hyper` can name — the commit the entry recorded
stands beside the revision, and reading it separates this cause from the other three — and §8 gives it a
sentence of its own; but naming it is all `hyper` does about a Run that has already happened. The act is
the author's and it is before the Run:
the orientation puts a commit in the loop, and `run` warns on stderr where the tree it read has not
been committed (§9, ADR-0119).

**`hyper review shell` never says what changed, and never will.** A built-in Provider's Manifest ships
inside the binary and has no file in the repository (§4, §11), so there is nothing for a Run to have
recorded a revision of and no number of Runs repairs it: alone among the five reviewed artefacts, a
built-in Manifest has no range at any point in its life, and the header says so as a named absence
rather than as an empty column (§8, §12). What that hides is less than it looks and the page points at
where it is readable — a built-in's bytes move only when the binary does, which is `hyper_version`,
which is the pin, which is a one-line edit to a Repository declaration that has a range like any other
file. The limit is that the reading takes two screens rather than one, and that the artefact you asked
about is not the one that answers.

**And it never says where to go.** With no file there is no path to state either, so the header's first
line carries the absence alone and that member goes silent rather than rendering a locator nothing
opens (§8, [ADR-0068](../adr/0068-one-supply-is-stated-once-and-the-member-it-silences-is-not-omitted.md)).
The two halves of that line are one limit and not two: the artefact you asked about is not the one that
answers, and it is not one you can edit.

**The record costs one runtime dependency, and it is the only one.** `hyper` reads and writes the Store
by invoking `git` as a subprocess (§7, [ADR-0075](../adr/0075-hyper-never-checks-the-store-out.md)), so
a single self-contained binary is not quite what ships: it needs `git` on `PATH`, on a laptop as much as
on a runner. §11 exempts the image's tools from the pinning ritual, and that exemption is about what a
**generated workflow** consumes, so this is named here rather than folded into it. What makes it
survivable is that `hyper` operates inside a git clone by construction — the repository root is found by
walking up to the git root (§9) — and what it buys is that the credential a checkout leaves behind
resolves the same way for `hyper` as for the human, rather than through a second implementation of
git's config that would agree with the first only until it did not. No version is stated: every command
on that path predates 2010.

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
