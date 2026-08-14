# §4 — Static verification

`check` is the name for every rule in this chapter, run together: load every artefact in the
repository and evaluate every rule below against them. No credential is resolved, no network reached,
no Operation invoked. `check` runs standalone — from an editor, a pre-commit hook, a CI job — and it
runs again, identically, before the first Step of any Run: pre-flight is `check` re-run with nothing
skipped, never a lighter pass, so the standalone command is never the only thing standing between an
artefact and the world.

`check` Refuses on the version pin like every other command: it compares itself against the Repository
declaration before reading a second file and Refuses on mismatch, on a laptop and in CI alike, with an
absent pin naming `hyper project` (ADR-0020). Every rule below presupposes that gate has already
passed.

Every rejection below carries a closed `error_code`, named here and defined in full in §12.

## Grammar

Loading a file is the first check, and failing it stops every check after. The rules are the grammar
§3 states in full and the sets §12 closes — the strict YAML subset, the input-schema subset, the
credential slot's mapping shape, the three hole positions, the path and reference grammar — and this
chapter is where each rejection gets the name it is refused under, nothing more: a construct the YAML
subset excludes is `strict-yaml-violation` (ADR-0023); a key the schema at that position does not
define is `unknown-key`; a `kind:` disagreeing with its directory or filename, against §12's table, is
`kind-mismatch`; an artefact's name disagreeing with its file's basename is `name-mismatch` (§3); an
input schema outside the closed subset is `schema-unsupported`; a malformed or
misplaced credential slot is `credential-slot-malformed`; a hole resolving outside its position's legal
source is `hole-illegal`, which covers a hole written inside an Auth scheme's parameters, the one
position with no legal source at all (§3); a reference naming an earlier Step of `series` cardinality is
`series-reference`; a reference naming a field no Operation of that Provider projects is
`reference-unresolvable` (§3); and a shell Step's `command:` that is empty, or whose first member is a
reference rather than a literal, is `command-malformed` (§3, ADR-0051).

`command-malformed` is one code over two faults because it is one check — *a shell Step names its
executable literally* — and both faults are the same absence of one. It is stated here rather than
under the hole positions above because what it constrains is a Step's argument rather than a Manifest's
template, and it is a member of its own rather than a widening of `hole-illegal` for the same reason: a
reader handed that code would go looking for a hole.

`name-mismatch` is its own member rather than a widening of `kind-mismatch`. The two carry different
disagreements over four artefacts, and one code standing for both would leave a reader of the rendering
unable to tell which of two files to edit.

## Resolution

Every name an artefact writes for something outside itself resolves, and the namespace it resolves
against decides which code refuses it. Matching is §3's, which is ADR-0060's: byte-exact over UTF-8,
case-sensitive, against the named artefact's own `name:`. An artefact whose own file will not parse is
present for this check — it declares a name, and the fault in it is reported once, on its own line
(ADR-0064).

**Four positions name an artefact of this repository**: a Definition's `provider:`, resolving against
the built-in Providers and `providers/` together (§11); a Definition's `targets:` members, against
`targets/`; a Step's `definition:`, against `definitions/`; and a nested invocation's `procedure:`,
against `procedures/`. One naming nothing is `artefact-absent`, and the row carries the file and line
the name was written on and the path it looked for — the two edits it points at being *fix the name* and
*write that file*. An Extension the repository never installed lands here, at `check`, with no network
reached: a `provider:` naming what `providers/` does not hold is this code and not `install`'s (§11,
ADR-0060).

**Four more name a member of an artefact**: a Step's `operation:` and a Definition's `destroy:` members,
each against the Operations the bound Provider declares; a `field:` at either Record root, against that
Provider's `fields:` mapping; and the `step:` half of a reference, against the `id:`s its own Procedure
declares earlier. One naming nothing is `reference-unresolvable`, which is what that code has always
meant — §3 already carries it for a bare `field:`, which is no reference in the format's sense, so the
scope is the namespace rather than the syntax. The split from `artefact-absent` is the split
`name-mismatch` was taken out of `kind-mismatch` for: which file the next act touches. A missing
artefact is one the reader may have to write; a missing member is a key inside an artefact that already
exists, and where that artefact is a built-in or somebody else's Extension it is not theirs to write at
all.

A Step's `target:` is in neither list. It resolves against its Definition's `targets:` list and nothing
wider (§3), so a Target's existence is asked once, where the Definition claims it, rather than again at
every Step that binds it — one absent `targets/prod.yaml` bound by six Steps is one row and not seven.

`local` is where both halves of this rule meet, and they answer differently on purpose. Where nobody
named the Target — a Probe, whose `local` comes from `hyper` — an absent declaration is a grant of
nothing and the request declines `host-not-granted` (§9, ADR-0042). Where an author wrote
`targets: [local]`, it is a name that resolved to nothing and it is `artefact-absent` like any other.
That is not two rules about `local` but one rule about who wrote the name, which is the line ADR-0060
already drew between a name the user typed and a name an author wrote.

## The Manifest's oracle

A Manifest is data, so what it claims can be checked against what its own Operations require, with
nothing but the file itself (ADR-0004). The Capabilities a Manifest declares must equal, not merely
contain, the Capabilities `hyper` derives from every Operation's schema and its own holes —
over-declared or under, either direction is `capability-mismatch`. §3's identity-field requirement and
Target-class type-check get their names here: an Operation projecting a Record and declaring no
identity field for it is `identity-undeclared`, and a Definition naming a Target outside its Provider's
declared class is `target-class-mismatch`.

Ten further checks are one fact wearing ten shapes — a Manifest whose own declarations disagree with
each other — and they share one name, `manifest-inconsistent`: a `pagination` Pattern on an Operation
whose `record:` carries no collection path, a `host-input:` naming a property the Operation's input
schema does not define, a `headers:` entry taking the request position its Auth scheme owns, a Provider
declaring only the `shell` Capability while carrying an `auth:` block, a
Target declaration's credential slots not covering the scheme's slots for a binding a Definition makes
(§3), a `read` or `mutate` Operation carrying no `record:`, a `destroy` Operation carrying one,
`skip-if-recorded` declared on an Operation that is not a `mutate` (ADR-0037), a `concurrency:`
limit declared on an Operation that is not a `read` (ADR-0045), and a `skip-if-recorded` Operation whose
`identity:` resolves only from the response, which is a test that cannot run before the call it decides
(§3, ADR-0056). The last five are the
Kind disagreeing with what the Operation projects, with what its Repeatability would have to read, or
with what its Expansion could ever do,
and they earn no code of their own because they point a reader at one file, one Operation, and two
adjacent keys — which is the discrimination `name-mismatch` was split out to preserve and this does not
lose. The fifth is checked per (Definition, Target) pair rather than on the Target declaration alone,
which is the one place a Target's own artefact is not sufficient — a Target declaration is written
without knowing which Provider will bind it, and the scheme is the Provider's.

A Capability an Operation's request names and the bound Target's declaration does not grant is
`capability-not-granted`, checked per (Definition, Target) pair for the reason slot coverage is: a Target
declaration is written without knowing which Provider will bind it. It is what makes `capabilities:` on a
Target a grant rather than a description of one, and it is where a repository omitting `shell` from every
class-local declaration stops every command it could have run, offline and in one line.

One check reads a Target declaration against itself. `hosts:` present where `capabilities:` does not
grant `http`, or absent where it does, is `target-inconsistent` — `manifest-inconsistent`'s shape one
artefact-class over, and it points a reader at one file and two adjacent keys for the same reason.

`local` is credential-free because a declaration named `local` carries no `auth:` block, refused as
`local-reserved` together with one whose `class:` is not `local` (§3, ADR-0041). The reservation is what
holds that rather than slot coverage: coverage compares what an author wrote against the scheme a Provider
declared, and the author of this artefact could write a scheme's slots into it. A Probe therefore resolves
no credential, which is the ground ADR-0017 stands on.

A separate check refuses a Manifest naming something reserved rather than disagreeing with itself. An
Auth scheme may not name a header `hyper` computes — `Host`, `Content-Length`, `Content-Type`,
`Transfer-Encoding`, `Connection` — and one that does is `auth-header-reserved`, compared
case-insensitively as an HTTP header name is. It is its own code on `capability-reserved`'s shape (§11):
what is refused is drawing on a name the tool holds, not an internal contradiction. `Host` is what makes
it a guardrail rather than hygiene, being the one header whose value decides which service a granted
host answers as (§3).

## The two keys

A Definition's own claim is checked before it is compared with any grant. `read` in `kinds:` beside
`mutate`, or beside a `destroy:` claim naming any Operation, is `definition-kinds-mixed` (§3, ADR-0032).
It reads one file and needs no Target, which makes it the one Kind rule below that refuses a claim
before anything has been asked about what would grant it.

A Step runs only where its Definition's claimed Kind and its bound Target's accepted Kinds intersect —
both authored, neither derived, so a claim of "never destroys" is a fact the reviewer can trust rather
than the Manifest's word for it. A Step whose claim and grant do not intersect is `kind-not-granted`. A
`destroy` Step's Operation must be named among the Operations its Definition claims for `destroy` —
granularity follows severity, so `read` and `mutate` check at Kind level and `destroy` checks by name —
and an unnamed Operation is `operation-not-claimed`.

A Definition claims Targets the same way it claims Operations, and a Step binding one it does not claim
is `target-not-claimed`. It is its own member rather than a widening of `operation-not-claimed` on that
code's own test: a reader handed *operation not claimed* on a `target:` line goes looking at `destroy:`,
which is the wrong edit. What the two share is the artefact pair and the remedy's shape — widen the
Definition's claim, in a reviewed edit, or bind something it already claims.

A Procedure's declared Target and Kind envelope must contain everything reachable from it: every Step's
own Kind and bound Target, and everything reachable through every Procedure it invokes, to any depth.
A Step outside its own Procedure's declared envelope, or an invoked Procedure's transitive envelope
reaching outside its caller's declared one, is `envelope-exceeded`, checked before the first Step of
either runs — composition cannot widen blast radius by accident, and neither can a Step written past
the envelope its own file declares (§12).

An `opaque` `destroy` Operation may run against a Target only where that Target's declaration has
opted in; a Definition claiming one against a Target that has not is `opaque-destroy-not-granted`. This
is the artefact half of the check — whether the credential in hand also carries the opt-in is resolved
at Run start and belongs to §5.

Such a Step must also name the population it destroys. An `opaque` `destroy` Step carrying no `over:`
selector is `opaque-destroy-unscoped`: it is invoked once (§3), has no Expansion to write a Tombstone
under and declares no identity, so it would reach the world and leave nothing in the record at all
(§5, ADR-0053).

## A skip that can only skip

A `skip-if-recorded` Step expanding over `assets:` is `skip-if-recorded-unreachable`. An effectful
Expansion reaches only Assets whose head stands (§5), and the value's test skips exactly while a head
stands, so every member skips on every Run and no call can ever go out — a Step refused for what it can
never do rather than for what it might. It renders a `RECORDS` count off a Disposition that made no call
(§8), which is the *truthful and still misleading* shape §5 declines on the Bound.

The remedy is one of two edits and the code points at both: an `over:` `values:` list, which is the form
that names a population `hyper` may not yet have built, or a Repeatability the Step's population can
answer. The check needs no Store and no credential — the selector form and the Operation's declared
Repeatability are both authored (ADR-0056).

## An Expansion with one identity in it

An Expansion's members are one Record identity each (§3). Whether they are is not decidable from the
artefacts alone except in one case, and there it is decided here: where the Operation's `identity:`
resolves before the call — a template hole, or `$.command` on a `shell` Operation (§3, §12) — and no
`{item:}` reference reaches the value that fills it, every member projects one name however the Run
goes. A literal in that position, or a reference to another Step's output, is one value for the whole
Expansion by construction, so a three-member list is three calls into one series and the Step's entry
will report a halt nobody performed.

Two things must both be authored for that certainty to hold, and the second is the one the Bound below
also needs. The identity must resolve before the call, or there is nothing on any file to read it off.
And the member count must be authored — an `over:` `values:` list of two or more — since a one-member
Expansion has no sibling to collide with and an `assets:` selector's size is on no file. It may still
collide with the Store, which is the same code against a comparand no file holds and is therefore §6's
(§6, ADR-0072). It is
`record-identity-collision`, the code §3 already fires at load where two members of one `values:` list
are one identity, and the same check found against the wiring rather than against the list; §6 carries
it everywhere else, over the identities an Expansion actually resolved.

The remedy is one of two edits and the code points at both: wire the member into the input the identity
reads, which is what an `{item: $}` in an `args:` value does, or drop the selector and write the calls
out as Steps. Neither needs a Store, a credential, or a response.

## The Bound

A `destroy` Step declares the maximum number of Records it may affect. An absent Bound on a `destroy`
Step means unbounded, and unbounded is refused before anything runs: `bound-missing`. An `opaque`
`destroy` Step is where that rule stops and the opposite one starts: it carries no Bound, and one
written there is `bound-illegal`. A count of the commands it ran says nothing about what any of them
did, and the only value a single command could carry is `1`, which would render as a promise the Step
cannot make (§5). It names a population all the same, under the check above. A `mutate`
Step's Bound is optional; its absence is not a check's business — it is rendered, unbounded, in the
blast-radius summary, which belongs to §8. A `read` Step carries no Bound at all, having nothing for
one to guard.

Whether an Expansion's actual count exceeds a declared Bound is not decidable from the artefacts alone
except in one case, and there it is decided here: an `over:` `values:` list is authored in the
Procedure, so its length is read off the file, and a list longer than the Step's Bound is
`bound-exceeded` before anything runs. The authored length is an upper bound on what the Expansion can
reach — the Store only ever removes members from it (§5) — so a list of seven under `bound: 5` cannot
fit however the Run goes, and refusing it is refusing a certainty rather than a guess. It is the same
code §6 carries because it is the same check; the run-time one is untouched and still guards `assets:`
and `observations:`, which no file can count.

## Predicates

A `field:` at a Record root names one key of the Manifest's `fields:` mapping (§3), so `check` compares
every selector's and every condition's against the Provider's declared field set: one naming nothing
projected is `reference-unresolvable`, the code a reference already carries for the same check. It has
to be a load error rather than a predicate that never holds, because `absent` over an undeclared field
is true for every version in the series — one typo turning a filtered `destroy` into an unfiltered one
with only the Bound in front of it.

`check` also refuses the operand faults that are authored and need no Store: an operator handed a type
it does not take — a `timestamp` under `greater_than` or `less_than`, an `in:` whose members are not all
one type, `exists: false`, an `in:` of one member, which is `equals` spelled twice — and a predicate
whose truth cannot depend on the value, which is an empty `in:`, an empty `starts_with:` or `ends_with:`,
and a predicate against a field the Manifest declares secret, that field reaching the Store as a
constant (§7). All are `predicate-type-mismatch`,
the code §6 also carries for a stored value of the wrong type — the same check, and §12 states which
half fires where (ADR-0035). The always-holds cases are the ones worth the code: a `starts_with: ""` is
a `destroy` selector that reaches the whole series and reads like a filter.

## The host list

§12 already states two comparisons of a host set against a Target's grant — an Operation's `host:`
template expanded to its candidate set, and an `over:` `values:` list. Both share one name: a member
absent from the grant, from either origin, is `host-not-granted` (ADR-0024, ADR-0029). Which `values:`
lists are host lists is not declared: a list is one where the Step wires `{item: $}` into the
Operation's `host-input:` and not otherwise (§3), so what `check` compares against the grant is read off
the wiring rather than off an author's word for it. What the intersection of a candidate set and a grant
then decides at Run time is §3's, and it is the one part of the host rule no static check performs:
where the intersection holds several hosts, the value a `host-input:` carries is checked for membership
when it arrives. A `values:` list wired there is the case where that run-time check has nothing left to
find, every value it will ever carry having been compared against the grant offline.

## The Cadence

A Procedure declaring a Cadence may reach no run-once Step, at any depth: `cadence-run-once`
(ADR-0038). Run-once Refuses where the Journal already holds the Step as *ran*, and a Refusal is
terminal for the Run (§5), so the second occurrence and every one after it stops at that Step and the
Procedure's remaining Steps never run. A Cadence over a run-once Step therefore declares a recurrence
with a lifespan of one occurrence, which is not a thing an author can have meant.

The walk is the transitive one `envelope-exceeded` already makes — every Procedure reachable from the
one declaring the Cadence, to any depth — because §10 makes *any shared body factored into a nested
one* the way two recurrences are authored, so the run-once Step will typically sit in the nested
Procedure rather than in the one carrying the clock.

`check` is the pre-flight of every Run, so what this refuses is the combination and not merely its
recurrence: such a Procedure does not run a first time either. That is the point rather than a side
effect — the alternative is a Procedure that works once, and a repository that looks correct until the
clock comes round. What is authored instead is two Procedures: the run-once Steps in one that is run
by hand, and the recurring Steps in one that carries the Cadence.

## What `check` cannot know

Every rule above is checkable because it compares one artefact's declared claim against another's, or
against a schema — reviewed text against reviewed text. None of it touches whether a Manifest correctly
describes the API it names: that its schema matches what the API actually accepts, that its projection
paths match what the API actually returns, that its declared Kind matches what an Operation actually
does. `hyper` has no oracle for that question and does not claim one — nothing cross-checks a declared
Kind against what an Operation actually does at runtime (ADR-0025), and the cheapest evidence
available, an unauthenticated Probe against the real response, only narrows the question rather than
closing it (ADR-0017). What a Run does when a projection path does not resolve against a real response
is §6's, that being the only place the question can be put. This is not a gap `check` closes by
growing; it is named here and carried forward as a limit in §13.
