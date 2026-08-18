# §9 — Surfaces

Everything `hyper` does is reached two ways over one core: a command line, and an MCP server. This
chapter states both. The CLI is below, and the MCP surface after it, reusing the names this half fixes
rather than restating them — the two surfaces share the verb set, the outcome contract, and the one
renderer §8 states, and differ only in ergonomics.

The CLI is where the long Run actually lives, and where CI invokes `hyper` at all. Every command below
either does what its arguments said or Refuses; nothing asks the operator anything (ADR-0015), and
every surface reads identically in a terminal, in a pipe, and in an Actions log.

## The tree

Sixteen commands, flat, every name a word the glossary already defines. There is one noun group and
no other nesting, no aliases, and no hidden commands.

| | |
| --- | --- |
| **Discovery** | `providers` · `provider <name>` · `operation <provider> <operation>` |
| **The repository** | `targets` |
| **Authoring** | `check [path...]` · `review <artefact>` |
| **Execution** | `run <procedure>` · `probe <provider> <operation>` |
| **Inspection** | `runs` · `show <run-id>` · `changes [procedure]` · `records` |
| **Lifecycle** | `install <ref>` · `project` · `store init` · `compact` |

`show` rather than `run show`, because the latter is ambiguous against a Procedure named `show`. `run`
and `runs` sit one letter apart, which is a readability wart rather than a hazard: `run` requires
arguments and `runs` accepts none positionally, so the typo in either direction is a usage error.

Two more commands exist and are not among the sixteen, because neither reads a repository and neither
says anything about `hyper`'s domain: `version` prints the version of the binary that would act, and
`completions <shell>` writes a shell completion script. They are also the only two exempt from the
version pin gate below (ADR-0020). Neither ever checks whether a newer version exists (ADR-0019).

Every one of the sixteen compares itself against the version pin in the Repository declaration before
reading a second file and Refuses on mismatch, on a laptop and in CI alike; where there is no pin it
Refuses naming `hyper project` (§4, ADR-0020). The pin gate is stated once here and presupposed by
every command below.

## A positional that matches nothing

Nine of the sixteen take a name positionally — `review`, `run`, `provider`, `operation`, `probe`,
`show`, `changes`, `install` and `check` — and eight of the nine answer the same way when that name
resolves to nothing: **a usage error, exit `2`, carrying no `error_code`** (§12). A Refusal is the
artefacts declining an act and a usage error is there being no act to decline (ADR-0036, ADR-0060).
Nothing was reviewed, so nothing refused, and `77`'s promise that a verbatim retry refuses identically
would be false here — the remedy is a different name, not an artefact edit. Like the pin gate above,
this is stated once and presupposed by every command below.

**A positional resolves against its own namespace, and whatever that namespace requires is in place
before the lookup can happen.** The pin gate fires first everywhere. After it, a working-tree name — an
artefact in `definitions/` or `procedures/`, a Provider built in or installed under `providers/` —
needs nothing further, so `hyper run typo` is `2` on a repository with no Store at all: the typo is
repaired before the Store is missed. `show` is the exception that states the rule, the Store being the
namespace its `<run-id>` resolves against, so `store-absent` (`77`, §12) necessarily precedes the lookup
and an unknown id is reachable only where the branch exists.

**A name matches byte-exact over UTF-8, case-sensitive, with no normalisation** (§12), compared against
the artefact's own `name:` — which `name-mismatch` already pins to its file's basename (§4) — rather
than settled by whether the filesystem's open succeeded. A macOS filesystem is case-insensitive and a
runner's is not, so `hyper review Deploy` against a `deploy` artefact would otherwise render on a laptop
and exit `2` in CI. The fold is `hyper`'s rather than the filesystem's, exactly as it is for a Record
identity (§7).

**What it writes is the name that was typed, the namespace it was resolved against, and the command
that enumerates that namespace** — `providers`, `runs`, `targets`. It lists no candidates and suggests
no near miss. A suggestion is a partial name resolved on the caller's behalf, which is what ADR-0047
refused for a Run id, and a human who accepts one has run something they did not type; enumerating a
namespace is the listing commands' job, and they carry `--limit` and a truncation marker because an
unbounded return is the hazard.

**No row stream opens.** stdout carries nothing in either mode, and the rendering goes to stderr like
every other human rendering of an error. There is therefore no terminal row and none is missing: a
usage error is not a path a command takes, it is the command never starting, and the exit code is what
says so.

Two of the nine are stated where they differ. `install <ref>` exits `1` rather than `2` (§11): its ref
names nothing in a registry rather than nothing in this repository, and a name that had to be fetched
before it could be judged is the world resisting rather than the invocation being wrong. And
`check [path...]` stats its paths before it loads a single artefact, so a path naming no file is `2`
and no problems are reported at all — the alternative is worse than unstated, since `check` reports
only the problems positioned in the paths it was given, so a path matching nothing filters to zero
problems and `hyper check definitions/typo.yaml` exits `0` clean on a job that checked nothing.

## Discovery

Three commands rather than one taking optional arguments, because they are three questions asked in
order — *which Provider*, *which Operation*, *how do I call it* — and a return shape that changes with
which argument was omitted is unusable to the caller that most needs it.

`providers` writes one row per Provider `hyper` can load, built-in and Extension alike: its name, its
origin (§12), a summary, how many Operations it exposes, and its digest.

Origin says where the Manifest's bytes load from and nothing else. It does not say whether those bytes
were verified against a registry — a built-in and a locally authored Extension both make no such claim,
and the digest on this row is `manifest_digest`, which every Provider carries (§7). That fact is
`provider`'s, below, where a Manifest's own facts are reported.

`provider <name>` writes one row per Operation the named Provider exposes — its name, its declared
Kind, whether it is `opaque`, and a summary — beside the Manifest's own facts: its Auth scheme, the
Capabilities it requires, its digest, its schema version, and the ref and digest of its `origin:` block
where it carries one. Kind is on every row at this level because it is what answers the two-key
question (§5) before a single input schema has been read.

The two origin members follow the ordinary absence rule: both are written where the block is there and
both are absent where it is not, which is a built-in Provider and a locally authored Extension (§3,
§11). Absent together they say the Manifest makes no digest claim, and that is the whole of what
distinguishes an installed Extension from one an author wrote — the fact `check` enforces as
`origin-digest-mismatch` and Provenance carries as `origin_digest`, reported here because this row
exists to state what a Manifest declares and this is the one block of it no surface rendered.

The Auth scheme renders as the header it composes, with the credential's position marked:
`Authorization: Bearer <secret>` for a `header:` scheme, `Authorization: Basic <secret>` for `basic:`,
and `none` for a Provider carrying no `auth:` at all (§3). `basic:` takes one marker for its two slots
because what reaches the wire is one derived string, and which two variables filled it is a Target fact
`targets` below carries. The composition is rendered rather than the parameters because a `prefix:` is
concatenated verbatim, so the load-bearing trailing space in `"Bearer "` is invisible in the source and
a Provider that omits it reads here as `Authorization: Bearer<secret>` — the failure is made legible
instead of guessed at by a check. Nothing here resolves a credential or reaches a network, and the
marker is §7's one constant rather than a second one.

`operation <provider> <operation>` writes the Manifest lines declaring that Operation, verbatim, and
beside them the facts the source does not carry in that form: the Capabilities `hyper` derives from it,
whether a Bound is mandatory, the Patterns it resolves to, its Record cardinality and declared identity
field, its Repeatability, its deadline, and its concurrency limit. The source verbatim, because a
Manifest is written in the format the caller is expected to author Definitions in (§3); the derived
facts beside it, because making the caller re-derive what `hyper` already computed is waste.

The concurrency limit reported there is the effective one and is always present: 1 for a `read` whose
Manifest omits the key, and 1 for every `mutate` and `destroy`, whose Expansion is serial and which may
not declare the key at all (§3, §6). A caller asking *how many at once* gets a number for every
Operation, and the rule about what may be **authored** stays where authoring rules live rather than
being inferred from a field that came back empty.

## The repository

`targets` writes one row per Target declaration: its name, the hosts it grants, the Kinds it accepts,
the Capabilities it grants, the environment variables its credentials resolve from — names only, never
a value (§3, ADR-0007) — and whether each of those variables is present. Each variable is paired with
the slot it fills, `token=CLOUDFLARE_API_TOKEN`, rather than listed bare: a declaration may carry slots
for more than one scheme, and a list of names alone does not say which fills what. Presence is computed
when the command runs; the value behind a present name is never read here, and never rendered anywhere.

**The host grant renders as `hosts`, an array, in the declaration's own order.** §3's Target
declaration has no `endpoint:` key: it has `hosts:`, glossed there as *one member where the Target is a
single endpoint*, and never a grant without an enumeration (ADR-0024). A Target granting several hosts
shows all of them — a candidate set, a grant and their intersection is how a host is decided, and a
grant reduced to its first member is not a grant (ADR-0029). The MCP tool below names the field the
same: one fact reaching two wires reaches them under one name (§12).

Presence is reported for every slot the declaration carries, which is wider than what a Run checks. A
Run resolves the slots its bindings require (§6), and this command has no Procedure in hand to narrow
by — so the row answers *what does this Target have in place*, and an absence here is not by itself a
Run that will Refuse.

## Authoring

`check [path...]` runs every rule §4 states, together. With no argument it loads every artefact in the
repository; with paths it still loads every artefact and reports only the problems positioned in the
ones named, because every rule §4 states compares one artefact against another and a subset of the
repository is therefore not checkable on its own. It resolves no credential, reaches no network, and
invokes nothing.

Its output is one row per problem — the file, the line, the field, the `error_code` §12 defines, and a
message — positioned so that the next act is an edit, ordered by file path and then by line. A Run
re-runs every one of these rules at its start (§6) and reports the same problems in the same order under
its own row type, one shape arriving through two commands (§7, ADR-0061); what the two differ in is the
exit code, `1` for a command reporting what it found and `77` for a Run a guardrail declined. There is
no `check --fix`: a checker that can
also mutate is a checker you stop trusting, and a repair flag on a gate is the shape ADR-0001 removed
elsewhere. What repairs projection drift is `project` below, which is a separate command for that
reason.

`review <artefact>` renders §8's Definition review of the artefact named. It resolves no credential,
reaches no network, and invokes nothing, so it runs offline against a repository whose Store is
unreachable — the header naming that absence once, for the range and for the Cadence gloss's last entry
alike, rather than the command failing (§8, §10). A `FLAGS` row
is a fact about the artefact rather than a problem with it, so a review that rendered exits 0 however
many flags it carried; only an artefact that would not load exits 1, and what it writes then is
`check`'s row. *Would not load* means found and faulty: an artefact that is not there at all has no row
to write and is the usage error above, exit `2` (ADR-0060). An artefact that loads and **names** one
that is not there is neither — it renders, marks `unresolved` where a derivation is missing, and exits
0, the fault being `check`'s to report and this surface's to annotate (§8, ADR-0064).

## Execution

`run <procedure>` runs the named Procedure, and it is the whole of what `run` does: every Run is a Run
of a Procedure (ADR-0036). A single Operation invoked directly through a Definition is not a second
form of this command — there is no such invocation, and handing `run` two positionals is a usage error.
Where the one positional names an artefact that exists and is a Definition rather than a Procedure, that
is a usage error too, the two living in different directories and the kind being known before anything
loads (§3).

What is written instead is a Procedure of one Step, and §3's `publish` Step is already that shape: a
`definition:`, an `operation:`, a `target:` and the `args:` the input schema requires, with no `over:`
and no `bound:`. What the artefact costs is those lines; what it buys is the Record, the Comparison row,
the gutter the review annotates, and a Bound where the Kind demands one.

`run` takes no `--target`. A Procedure is fully bound and declares its own Target envelope (§5), so a
Target supplied at invocation is either redundant with the artefact or it is authority arriving after
review, which is what ADR-0008 removed. Supplying one is a usage error rather than a silently ignored
flag.

The invocation carries no argument value at all. What it carries is the occasion and never authority: a
Secret sink, a dry-run marker, and output formatting (§6, ADR-0008).

`run` renders nothing before executing (ADR-0015). What it writes is the Step table §8 states — each
Step's Disposition and the count of Records it concluded about, which is not the count it wrote
(ADR-0030) — and, where a guardrail declined, §8's Refusal rendering in full; last is §8's terminal
line, naming the outcome, the exit code, and — whole — the
entry the Run wrote. Under `--json` it emits those two renderings' rows and the Run's `provenance` in
both its scopes — the Run-wide row and one row per Step file written (§7, §8) — terminated by the
`outcome` row carrying those same facts, and nothing else: what each Record did is the Comparison's
rendering rather than the Run's, and `changes` is what emits it (§8).

**`--dry-run`** is accepted on `run` and on no other command. It performs the reads it reaches and
stops rather than simulating an effect, and it writes a Journal entry marked as a dry-run (§6). A
dry-run that stopped renders that it stopped and why — the withheld Step whose output the next one
would have read — with the Steps after it *never reached*. Its outcome is `completed` and it exits 0,
a halted rehearsal being the correct outcome of a correct operation rather than a failure; the answer
is partial, and it says so on the page rather than in the code. The flag is not global: a `records
--dry-run` or a `check --dry-run` would have to mean something, and neither does.

**`--secret-out <path>`** names the Secret sink. A Run reaching a Step whose Operation declares a
secret output Refuses when none was supplied (`secret-sink-absent`, §12), which is a fact about the
invocation and never about the environment it runs in (ADR-0007). The path is written `0600` and is
refused where it resolves inside the repository working tree; `-` is not accepted, stdout being
exclusively the answer and a secret written there landing in the same pipe a CI job logs. It is not a
bypass and must not read like one: supplying it weakens no check, and withholding it produces a
Refusal that renders like any other (§8).

The Refusal declines before Step 1 rather than at the Step, beside the credential gate and on the same
ground — both operands are the occasion's and both are in hand — and it names every Step that would
have needed a sink rather than the first (§6). It carries no Kind axis: a `read` declaring secret
output is as refused as a `create`, because the sink is the only route by which a secret value ever
leaves `hyper`, and a Run that suppresses one into the Store with nowhere to hand it is useless
without saying so. For the same reason `--dry-run` earns no exemption — the rehearsal performs the
reads it reaches, and one of them may be the Step in question.

Because the projected workflow supplies no sink and cannot sensibly be made to (ADR-0007, ADR-0077),
a Procedure that declares a Cadence and reaches such a Step is refused at `check` (`cadence-secret-output`,
§4) rather than left to Refuse unattended at every occurrence.

`probe <provider> <operation>` invokes a `read` Operation against `local` without a Definition. Inputs
are supplied as repeated `--input <name>=<value>`, each read against the Operation's declared input
schema at that position rather than by what the value looks like (§3); an unknown name, an input left
out — every declared input being supplied (ADR-0081) — or a value whose characters will not read as
its declared type is a usage error, exit `2`, carrying no `error_code`. That is the same fault §4
refuses as `schema-mismatch` and it is not that code here, on ADR-0060's line: an `error_code` names a
check that declined an **artefact**, and a value typed at a command line is not one. Its two positionals resolve against the Provider set exactly as
`operation`'s do, and a Provider or Operation matching nothing is the same usage error the tree above
states. A Probe writes no Record and no Journal entry, has no Trigger, no
Provenance and no Disposition, and can never be scheduled, sequenced into a Procedure, or used as a
Comparison baseline (ADR-0009). Having no outcome triple, it terminates its row stream with `result`
and never with `outcome` (§8). It may surface the raw response beside the projection `hyper` derived
from it, which no credentialled surface does (ADR-0017).

A Probe exits `0` whatever came back — a `503` as readily as a `200`, and a host that answered nothing
as readily as either, that being a response object of `host` alone (§3, ADR-0050). A `read` never halts
on what came back, so there is no failure to map, and a nonzero exit would be `hyper` deciding that a
`503` is bad news: the judgement this surface refuses when it renders a Cadence's staleness and never
says *overdue*. The exit code says whether the command did what it was asked; the rendering says what
came back. What can still fail a Probe is its projection, on the one path every surface fails it
(§6).

A Probe's `--input` is the only place in `hyper` where a value arrives at invocation, and it is not the
door ADR-0036 closed wearing another name. It chooses what is *looked at*: a Probe is not a Run, is
`read` Kind against `local`, and writes no Record and no Journal entry, so nothing it carries can widen
what a reviewed artefact permits, reach a credentialled Target, or leave evidence a later Run reads.
An input on `run` would be none of those things.

What holds the first of those is the grant. A Probe's host is checked against the `hosts:` the Target
named `local` declares, exactly as a Step's is, and one outside that set is `host-not-granted` (§4,
ADR-0042): the reach comes from an artefact even where no artefact named the Operation. A repository
declaring no `local` grants no host, so a Probe there reaches nothing and declines the same way — an
absent declaration being a grant of nothing rather than a fault of its own — and the Refusal renders the
artefact to author (§8).

**A Probe may never invoke an `opaque` Operation**, whatever any Target grants. `shell`'s `read` is a
`read` Kind and its `class:` is `local`, so `probe shell read` satisfies every other rule on this page
— and what it would run is a command supplied at invocation, with no Definition, no Journal entry and
no Record, which is to say with nothing reviewed anywhere and no evidence afterwards that it happened.
The grant that holds the `http` case does not reach it: `hosts:` is present exactly where
`capabilities:` grants `http` (§3), so there is no host to check, and checking the Capability grant
instead would turn on whether a repository happened to grant `shell` on the Target it named `local` —
which is what an author with shell Steps and one class-local declaration does by default. A guardrail
whose failure mode is the obvious authoring is not one.

It is a **usage error** rather than a Refusal, and it names no `error_code` (§12). A Refusal's
remediation points at an artefact to edit (§5, §8), and there is no edit that would make this work: the
surface declines the invocation, on the reading that puts `install` outside MCP. Nothing is lost that
`hyper` was ever the right tool for — a human, and an agent, can each run the command directly, and
`hyper` has no business offering a worse terminal than the one it is invoked from.

## Inspection

Four commands over the record, taking typed, closed parameters and nothing else. There is no predicate
dialect over them and none behind them: a caller wanting an arbitrary filter takes the rows and applies
it themselves (ADR-0013).

`runs` takes `--since`, `--procedure`, `--target`, `--outcome`, and `--limit`, and writes one row per
Journal entry: the Run id, when it started, its Trigger, its outcome, its Procedure, the Targets it
bound, and the version of `hyper` that performed it. The Trigger is on every row, being the only thing
that distinguishes a world that has not changed from one nobody has looked at (§7).

An open entry (§7) renders no outcome at all, the cell carrying that absence rather than a fourth value:
*open* is a state and not a member of §12's triple, so writing it into a column named for the triple
would relitigate that distinction by accident. `--outcome` filters the triple and therefore never selects
one. A `started` beside an absent outcome is the whole of what the Store holds about a Run nobody has
closed, and the row says exactly that much.

A **contested** entry (§7) renders its owner's outcome in that cell and carries a marker on the row. The
cell is not where the contest goes, for the reason the open entry's is not: the column is named for the
triple, the owner's account is a member of it, and a second account of the entry is not a fourth value
either. `--outcome` therefore selects a contested entry on its owner's outcome, which is the one the
entry has. `show <run-id>` is where the contest is stated in full, one line per `closed-by/` file in the
form §8's header uses.

`show <run-id>` takes a Run id whole — nothing anywhere resolves a partial one (ADR-0047) — and writes
one entry in full: each Step's Disposition with the Record identities it acted on, `hyper`'s own account
of what it did to reach that outcome — a Pattern's attempts, its pages, its poll iterations — and, on a
Step a projection failure halted, the path that failed to project beside the partial set it wrote
(§6, §7). On an effectful Step whose call answered anything but `2xx` it writes the host reached and the
status got, which is as true of the `404` that completed a `destroy` as of the `500` that halted one
(§7, ADR-0050). Each of those sits beside that Step's own Provenance and all of it beside the Run's
(§7). Under
`--expansion` each Step
also carries its selector, what that selector expanded to, and its Bound, which is what §8's Refusal
footer points at.

Where an entry holds a digest and no members — the set did not move since the Run that last carried one
(§7, ADR-0055) — `show` resolves the members from that Run and names it, rendering them as *unchanged
since* it. It reads another entry's bytes only by saying so: rendering them bare would present a set
this entry does not hold as though it did, which is the omission these surfaces are built to make
impossible (ADR-0026). Nothing is stored to spare it the walk, and the walk always terminates (§7).

`changes [procedure]` renders §8's Comparison. Naming a Procedure selects it and omitting one compares
across every Procedure at once, which is why the Procedure is positional here and a parameter on `runs`
— it decides which rendering you get rather than filtering the rows of one. It takes `--since
<timestamp>` or `--between <run-id> <run-id>`, and `--target`, `--kind`, and `--limit`; `--since` and
`--between` together is a usage error, the two being different ways of naming one window.

`records` takes `--target`, `--definition`, `--name`, `--history`, `--since`, and `--limit`, and writes one row
per Record: its identity, its ordinal, the Run and Step that wrote the version, whether it is an
Observation or an Asset, whether its head is a Tombstone, which of its fields carry the presence-only
secret marker (§7), and its Provenance. It returns the Head only unless `--history` is given — an
explicit boolean rather than a mode that turns itself on when some other parameter is named (ADR-0013).
An Asset whose Definition no longer exists is marked Orphaned on every row that carries it, for as long
as it stands (§7).

The ordinal is §8's and unstable for the reasons stated there. The Run and Step are the version's
identity, and this is the surface that carries them: it is the one whose job is finding a version rather
than reading a change, and two Steps of one Run writing one identity write two paths (§12), so the Run
alone would not name one. They render abbreviated here and whole under `--json`, like every other id on
a table read down a column (ADR-0047), and nothing takes an ordinal as input at all — naming a version
is naming its Run (ADR-0049).

**The record has two axes, and every command above orders on the one it ranges over** (ADR-0065). An
**identity** ordering sorts by `(Target, Definition, name)`, each by Unicode code point, the columns read
left to right — §8's Comparison rule, §7's identity set rule and §6's Expansion rule, the one ordering
`hyper` has over Record names reused rather than stated a fourth time (ADR-0044). A **time** ordering runs
**newest-first**.

`runs` orders on `started_at`, ties broken on the `<run-id>` descending: §7's Head shape, a time key with
a name behind it, and a UUIDv7 is total over the tie. Ordering on the Run's start rather than its end is
also what gives an open entry a position like any other — an entry with no `outcome.json` still carries a
`started_at`, so nothing here needs a rule for one.

`records` orders on identity, and under `--history` it is identity-major: every version of one Record
together, the identities in name order, and the versions inside each in **§7's Head ordering read
backwards** — `written_at` descending, ties broken by the file name descending. The reversal is whole,
both keys inverting together, so the ordering that decides which version is the Head and the ordering this
surface renders can never drift apart. The first row of each series is therefore the row `records` returns
without `--history` at all.

`changes` renders §8's tables, whose rows §8 orders. `show` reads one entry back and orders nothing: its
Step rows are the Run's written order, `<nnnn>` (§12), and `expanded_to` under `--expansion` comes back in
Expansion order because §7 holds it that way and not sorted — the sequence being where a halted `destroy`'s
stopping point is legible, and the identity set beside it being a fact about membership instead.

**A query chooses an order; a Run reports one.** The rule governs the four commands here and nothing else.
A Run's `step` rows arrive as the Steps reach their Dispositions, a Refusal's rows go in the array's order
(§8), and `expanded_to` is a sequence: none of the three is a result set being ranged over, and reversing
any of them would reorder events rather than facts.

The order is **normative**. Two renderings of one result are byte-identical and diffable, as §8 requires of
a Comparison, and the row stream is why it is a contract rather than a convenience: NDJSON goes out one
line at a time with no cursor behind it, so a consumer cannot re-sort a row it has already printed (§8).

Every command in this section takes `--limit`, with a modest default, and truncation keeps the **first N of
the ordering above**. That is what the order buys: the first fifty of four thousand Records is the answer to
a question rather than an arbitrary sample of one. Every truncated result carries a marker naming the axis
— `time` or `identity`, §12 — what was returned, and what was dropped. There is no cursor and no
pagination: an unbounded return blows a context window on the first interesting month, and walking three
thousand rows a page at a time is the same disaster arriving politely. Truncation is a signal that the
question was too broad, and the marker names the axis so the next call is a narrower question. A truncated
result must never look complete.

Under `--history` the limit counts **identities and not rows**: a series comes back whole or does not come
back, because a series cut partway through is a partial history wearing a complete one's shape, which is
the one thing the sentence above forbids. What bounds a single series instead is a cap on versions per
series, and it is unnumbered here for the same reason the default above is — both are constants an
implementation picks, and neither is a fact any artefact, Record or check depends on.

`records` takes `--since` so that the axis a cap can cut has a parameter that narrows it: the marker names
an axis so the next call can be narrower, and a caller who has already named one Record has no other
narrowing left to do. It is legal only with `--history`. Without it the parameter would filter heads by when
they last moved, which is a change read on the command whose job is finding a version, and having it turn
`--history` on instead would be exactly the mode ADR-0013 refused; `--since` without `--history` is a usage
error, like `--since` and `--between` together above.

## Lifecycle

`install <ref>` fetches an Extension and writes the tracked, digest-verified file §11 states. It is the
single point at which third-party data enters the repository, which is why what it writes is a tracked
file appearing in a diff a human reads. It takes no `--dry-run`; `check` already reports digest drift.
It is the one command whose positional matching nothing is not a usage error: a ref names nothing in a
registry rather than nothing here, and it exits `1` (§11, ADR-0060).

`project` regenerates the projection §10 states — whole-file, always overwriting, never merging, one
file per Procedure — and derives the version pin from the binary that ran it (ADR-0020). It is repo-wide
and all-or-nothing: there is no `project <procedure>`, since per-Procedure projection would let two
Procedures pin different versions against one Store. Overwriting a hand-edited workflow is correct
rather than regrettable — a hand-edit to a projected file is authority living outside every reviewed
artefact, and the git diff `project` writes is where it gets reviewed. It Refuses where it cannot
resolve a published artefact for its own version, which is exactly the case where it would otherwise
write a workflow fetching a binary nobody can download (ADR-0020).

`store init` creates the orphan branch §7 names and writes `STORE.md`, and does nothing else: there is
no configuration to write (ADR-0014), and no example Definition is scaffolded, `hyper` authoring a
reviewed artefact being the line the whole surface does not cross. It touches no file in the working
tree — the branch is a parentless commit built from objects, and nothing about it is ever checked out
(§7, ADR-0075) — so it runs against a dirty tree like any read command. Every command that needs the Store
and does not find it Refuses naming this one (`store-absent`, §12) rather than failing: an absent Store
is a guardrail declining, not the world resisting.

`store` is the one noun-grouped command in the tree. The noun is what makes *this creates a git branch*
legible at the point of use, where a bare `init` would read as initialising a repository, and the group
is where a second verb goes if one is ever earned.

`compact` removes what §7 permits it to remove and nothing else. It never runs automatically and never
on a Cadence, it is not a Run and writes no Journal entry — it is an ordinary commit on the Store
branch, so `git log` there is its own account of what it removed — and it takes no `--dry-run`.

None of the four is a Run, so each writes what it changed and terminates its row stream with `result`
rather than `outcome` (§8): the file `install` wrote and its digest, the workflows `project`
regenerated, the branch `store init` created, and the versions `compact` removed.

## The three configuration layers

Configuration is **flags → environment → defaults**, in that precedence, and it governs presentation
only. There are three globals and no more: `--json`, `--repo-dir` (`HYPER_REPO_DIR`), and `--no-color`
(honouring `NO_COLOR`). Terminal width is auto-detected and is never a flag. The repository root is
found by walking up from the working directory, bounded by the git root.

There is no configuration file of any kind — no user-level file, no repository-level file, and nowhere
a future setting could go (ADR-0014). Nothing `hyper` reads can change what a Run does except a
reviewed artefact and the environment's credentials, which is the property the absence exists to
protect: a user-level file would make `hyper` behave differently on one machine than another, and a
repository-level file would be an unreviewed artefact competing for authority with the reviewed ones.
The Repository declaration is not a counterexample — it is authority, it is reviewed, and `review`
renders it (ADR-0020).

Everything that survives is invisible to the outcome, so its precedence is uninteresting by
construction. The ergonomic cost is accepted and is real: a repeated `--repo-dir` cannot be made
sticky, and there is no way to set a personal default for anything.

Six flags prior art would lead a reader to expect are absent, each for its own reason. `--quiet` is
`2>/dev/null` now that stderr is narration; `--log-level` and `--show-properties` are a logging
framework's surface rather than this tool's; `--junit` is a second document shape to keep in sync by
hand, which the single-renderer rule already rejected (§8); and `--server` and `--token` belong to a
`serve` that does not exist.

## Stream discipline

**stdout is the answer, and nothing else ever goes there.** The human tables or the row stream, in
whichever mode was asked for, and no diagnostic of any kind. `hyper changes --json | jq` is therefore
safe by construction, and the classic CI hazard of a warning interleaved into parsed output cannot
arise. §8's terminal line is the answer's last line and goes there like the rest of it, which is what
puts the Run id on the job summary (§10).

**stderr is the narration.** Progress, warnings, and the human rendering of an error, always, in both
modes.

**Progress is one line per Step boundary, in both modes, always on.** With the outcome arriving last, a
silent twenty minutes is indistinguishable from a hang. It is narration, so it carries no machine
contract and has no `--json` variant: a consumer wanting Step-level structure reads the Journal, which
§7 writes per Step as that Step reaches its Disposition.

**A Run names itself on stderr before its first Step**, as the first line of that narration and in both
modes. The id is available there — `run.json` is written at Run start (§7) — and the terminal line is
not always reached: a process killed outright renders nothing and leaves the open entry §7 states, which
is the one Run whose identity its own output would otherwise never carry, and the one §7 says `hyper`
never guesses about. It is narration like the rest, so it carries no machine contract, has no `--json`
variant, and never reaches the job summary, which takes stdout alone. The id therefore renders twice on
an interactive Run that finishes, and that repetition is what the Run that does not finish costs.

**The last row is always the terminal row**, and its absence means the stream was cut off. There are
two, `outcome` for a Run and `result` for everything else, and §8 states both. A Probe is on the
`result` side, having no outcome triple to report (ADR-0009). `run` is on the `outcome` side on every
path on which a Run was attempted, the two that decline before a Run is identified included: what is
missing there is the row's `run_id` and never the row (§8). A usage error is not one of those paths —
it opens no stream at all, so there is no terminal row to be absent from and nothing was cut off
(above, ADR-0060). It is the one case where stdout is silent, and the exit code is what distinguishes
it.

**`error_code` names a check, and a failure carries none.** Every member of the set §12 states is the
identifier of a check that declined, stated where §12 attributes it, so the code mints no vocabulary of
its own and a `refused` Run carries one where a `failed` one carries nothing. There is no second half
for failure: the ways the world can resist are not a set `hyper` could close over, and what tells two
failures apart is the exit code below — closed, finer than the outcome triple, and four of its seven
members `failed`, which is where a Store lost to contention is told from a Step that errored. No
`error_code` is ever Provider-supplied — a Provider is data and can no more invent one than it can
invent an Auth scheme (ADR-0004) — because a CI-facing contract that grows by extension is one where an
Extension author mints a code somebody's script treats as retryable.

**Colour and width are the only differences between a terminal and a CI log**, and colour never carries
a fact of its own (§8, ADR-0015). The job summary §10 states is those same renderings relocated rather
than a surface of its own.

## Exit codes

Seven members, defined in full in §12, each carrying there the outcome of the triple it maps onto.

**No exit code ever spans two outcomes.** The code space is finer than the triple and never coarser,
which is the whole point: a caller that reads `refused` as success has been told the wrong thing about
what reached the world.

A caller that retries on `75` is right to, and one that retries on `77` loops forever: with no bypass
anywhere (ADR-0001) a verbatim retry refuses identically. A shell script has the same reflex as an
agent, and now the same signal.

**What separates them is whether an act is required, not how severe the stop was.** Past a `77` lies an
artefact edit, a `hyper store init`, a `hyper project`, a newer binary, or a variable set in the
environment — an act of somebody's, and until it happens nothing changes. Past a `75` lies time: a lock
released, a branch reachable again, a push that rebases. That is why a Run that could not sync the Store
is `failed` at `75` and not a Refusal (§7, ADR-0061): the network coming back is not an act, and telling
a caller not to retry the one thing retrying fixes is the single most expensive thing this pair of codes
could get wrong.

The mapping is what keeps the two credential failures apart. Presence is checked where §6 resolves the
credentials the Run's bindings require, before the first Step: one a binding requires and the
environment does not hold is a Refusal and exits `77` (`credential-absent`, §12), while one that is
present and the endpoint rejects is the world resisting and exits `1`. Nothing about where the process
runs enters either decision (ADR-0007). Every absent slot is reported, one member of the Refusal's array
each (§7), because the pass resolves them all in one go and knows them all at once — reporting the first
would send an operator round the loop once per variable, each `export` earning another `77`.

Commands that are not Runs carry no outcome. They use `0` for clean, `1` for problems found, and `2`
for usage, plus `77` where a guardrail declines — the version pin gate above, and the absent Store
`store init` answers, `compact` included. `probe` is the one command that can never exit `77` past the
pin gate: it touches no Store, and against `local` the two-key check is vacuous, so there is no
guardrail there to decline.

A signal is handled as §6 states: the first interrupt drains, the Step in flight finishes, no further
Step starts, and the Run closes its own Journal entry `failed`, exiting `130` or `143` according to the
signal it received. Only a second signal kills the process, which is what leaves the open entry a later
Run closes.

A Run whose every Step skipped exits `0` like any other completed Run. *Did nothing* is not an outcome
and gets no code of its own; the Dispositions are where the difference is legible (§6).

## No prompt, and no bypass

`hyper` asks the operator nothing: no confirmation before a `destroy`, no *are you sure*, no
interactive picker, and no TTY-conditional behaviour anywhere except colour and width (ADR-0015). A
prompt is a bypass with better manners — it puts the decision back at the keyboard, unreviewed,
unrecorded, and answered in the second before it is understood — and it cannot exist in the environment
this tool was built for, where an unattended effectful Run on a Cadence is normal: a prompt is either
skipped on a runner, making the guardrails different in two places, or it hangs a scheduled Run until
it times out.

The pre-flight summary dies with it. `hyper run` renders nothing before executing, because a summary
with no question after it is decoration that reads like a checkpoint and scrolls past unread in CI.
Review happens at review time, on the artefact, through `hyper review` — before the commit, not before
the process.

There is no `--force`, no `--yes`, and no `--skip-checks`, on `run` or anywhere else (ADR-0001).
`--secret-out` is not one of them wearing a different name: withholding it produces a Refusal and
supplying it weakens no check, which is the opposite direction of travel.

Ctrl-C is the only interactive control there is, and it is handled as a signal rather than as a
question.

## What the CLI does not do

There is no `serve`, no daemon, and no remote transport: nothing in `hyper` listens on a port, and
nothing outlives the invocation that started it.

There is no command that authors a reviewed artefact. `hyper` writes what it derives — the projection,
the pin, the Store branch, the Extension file — and the agent or the human writes what is reviewed,
with their own tools. Discovery good enough to write a Definition correctly, and a `check` positioned by
file and line, are what stand in place of a generator.

**A destructive Run started by mistake is not catchable at the keyboard.** What stands in its place is
entirely static and entirely before the invocation: the two keys, the named-Operation requirement on
`destroy`, the mandatory Bound, and the Definition review (§4, §5, §8). This is a relocation of the
safety net rather than a gap in it, and it is stated plainly because the moment a human most wants a
prompt is exactly the moment this decision denies them one. Carried forward to §13.

**A Refusal is recorded in full only where there is a Store to record it in.** §5 and §7 state a
Refusal as held in the Journal with its Provenance, and that holds wherever the Store is reachable and
the Run has been identified. Where it is not — the bootstrap case, a repository whose Store branch has
never been created — the Refusal is rendered, exits `77`, and writes nothing at all, so a scheduled Run
that refuses there leaves no trace in the repository and is found in the Actions log. Its terminal line
and its `outcome` row name no entry for the same reason (§8), there being none to name. The alternative
is creating the Store implicitly in order to record why it could not be found, which is the implicit
creation §7 forbids arrived at by the back door. Carried forward to §13.

## The MCP server

The server is the same binary, started by the client over stdio: one process per client, dying with it.
It is not the `serve` above wearing another name — nothing listens on a port and nothing outlives the
invocation that started it — and every tool passes the version pin gate exactly as its command does.

**Tools only: no resources and no prompts.** The read-only half of this surface is shaped like MCP
resources and is deliberately not served as them, a resource being client-controlled and, in most
clients, user-attached rather than agent-reachable — which would put Manifest discovery out of reach
exactly where an agent needs it. Prompts are per-editor glue, which is the thing this tool departs
from.

Ergonomics is the whole of the difference between the two: where the CLI takes a flag this surface
takes a typed argument, and where the CLI disambiguates a name against a Procedure this surface
disambiguates it against every other server the client has loaded.

## The tool set

Thirteen tools, each named for the command above that it carries:

| | |
| --- | --- |
| **Discovery** | `providers` · `provider` · `operation` |
| **The repository** | `targets` |
| **Authoring** | `check` · `review` |
| **Execution** | `run` · `probe` |
| **Inspection** | `runs` · `run_show` · `changes` · `records` |
| **Lifecycle** | `project` |

`run_show` is the one name that differs from its command. A client holds every server's tools in one
flat namespace, where a bare `show` names nothing; the ambiguity the CLI resolved was a different one.
The two commands outside the sixteen get no tool either: a client writes no completion script, and the
version of the binary that would act is the version of the server the client started.

Three of the sixteen commands are absent, and one line puts all three on the far side of it: **an agent
may read the record and add to it, and may not create it, prune it, or bring anything new into the
repository.**

`install` is the single point at which third-party data enters the repository, and §11 makes what it
writes a tracked file precisely so that it lands in a diff a human reads. The claim that a hostile
Extension reaches nothing you did not grant (ADR-0004) survives an agent installing one; the review
moment does not. An agent that can install can author against what it installed and run it in the same
turn, which is the whole supply-chain sequence with no human between acquisition and effect. A required
digest argument in place of the exclusion is theatre — the agent reads the digest from the registry it
was already trusting.

`store init` creates the record and `compact` removes from it permanently (§7). Neither is a derivation
an agent could be asked to repair, and `compact` is the one command that would let an agent prune the
account it is itself held to. A tool finding no Store therefore Refuses naming a command its caller
cannot reach, which is correct: creating the record is the human's act, and an agent's part in it is to
say that it has not happened.

`project` is on the reachable side, and it is why the line falls where it does rather than around
writing at all: a Cadence declared in a reviewed artefact and left unprojected is the drift §10 states
a check for, and an agent must be able to repair what it caused. What `project` writes is derived from
artefacts already reviewed and lands in a diff like everything else.

## The return envelope

Every tool returns one shape.

```jsonc
{
  "content": [{ "type": "text", "text": "…" }],
  "structuredContent": {
    "outcome": "completed",   // §12's triple, on the execution tools only; absent elsewhere
    "run_id": "01991ea6-b118-7c93-8d41-6b2f7ae05c19",  // whole; absent where no entry was written
    "dry_run": false,         // beside it, `false` included, wherever `outcome` is present
    "rows": [ … ],            // §8's rows, as an array
    "truncated": null         // the truncation marker, or null
  },
  "isError": false
}
```

**`rows` is §8's row set unchanged** — one object per table row, carrying the same `type`
discriminator — served as an array rather than as a line stream. There is one renderer behind both
forms, so the terminal and this surface cannot drift apart (ADR-0026). A header is a row with its own
`type` and never a key beside `rows`.

The terminal row is the one row §8 states that this surface does not emit: an array's end is already
its own end-of-stream marker, so `outcome` moves into the structured content and `truncated` carries
what a `result` row carried. `run_id` and `dry_run` move with it — the terminal fact is one fact and
does not arrive in two places — and the id is absent on the two paths that decline before a Run is
identified, exactly as the row's is (§8). Its shape names the same axis, count and remainder the CLI's
marker does:

```jsonc
"truncated": { "axis": "time", "returned": 200, "dropped": 2840,
               "hint": "narrow with `since` or `target`" }
```

**The `text` block is asymmetric, and the asymmetry is the point.**

| case | `text` carries |
| --- | --- |
| any ordinary return | one summary line, outcome first |
| `review` | the full rendered review surface — the gutter, `AUTHORITY`, `FLAGS` |
| a Refusal | the full rendered Refusal — Step table, caret excerpt, `EDIT ONE OF`, retry sentence |

Outcome first repairs what §8's row stream accepts knowingly: with rows the outcome arrives last, and
the terminal compensates with an exit code, which this surface does not have. On `run` that line names
the Run id after the outcome, §8's terminal line arriving here as a sentence: a client has no exit code
to read and no scrollback to search, so an entry the envelope does not name is one an agent can reach
only by asking which Run was the last, on a Store two environments write. The two full renderings
are the same trade §8 made for the same reason — with no bypass anywhere, the Refusal rendering is the
entire remediation path (ADR-0001).

**`outcome` is the discriminator, and `isError` is not.**

| outcome | `isError` |
| --- | --- |
| `completed` | `false` |
| `refused` | `true` |
| `failed` | `true` |

One bit cannot carry three states, so `isError` was never going to separate the triple; it means only
*you did not get what you asked for*, which is true of a Refusal and of a failure alike. Returning a
Refusal as a success would undo everything §8 spends making it unskimmable. Nothing in the structured
content restates that bit, and no row restates the outcome.

A tool that is not a Run carries no `outcome` key at all, and a guardrail declining one — the version
pin gate, an absent Store — returns `isError: true` with the Refusal rendered in full, exactly where
the command exits `77`.

**A domain outcome is never a protocol error.** JSON-RPC errors are reserved for malformed calls — an
unknown tool, an argument violating a schema, an argument that is well-typed and names nothing, a fault
in the server — and a Refusal is an answer to a well-formed call. Every usage error the CLI half states
arrives here as one of those, which is what this surface has in place of exit code `2`.

The third member is this surface's whole account of a positional that matches nothing (above): a
`run("typo")` or a `run_show` against an id the Store lacks satisfies every schema and still names
nothing, and returning it as a domain answer would give it `isError: true` with no `outcome` key —
which is exactly the shape a guardrail declining already returns. That would make a usage error
indistinguishable from a Refusal on the one surface with no exit code to tell them apart, which is the
distinction the CLI half spends `2` to draw. `install` has no tool here, so its `1` has no MCP half
(ADR-0060).

**Every Refusal's `text` block ends by stating that a verbatim retry will refuse identically**, naming
the artefacts to edit. This is load-bearing rather than manners: `isError: true` conventionally invites
a retry, and this surface has no exit code `77` with which to say otherwise (ADR-0001). The rendering
is the only place the protocol leaves for saying it.

## The thirteen tools

Every argument is typed and closed exactly as the flag or the positional it carries is, and its name is
that flag's with the hyphens turned to underscores. Two differ, and each says why where it appears. No
tool takes an override argument of any kind, under any name.

### Discovery

Three tools rather than one taking optional arguments, for the reason the three commands are three, and
here the protocol says it too: an `outputSchema` is declared once and for every call of the tool.

```jsonc
providers()
// → rows: [{ type: "provider", name, origin: "built-in" | "extension",   // origin: §12
//            summary, operation_count, digest }]
```

```jsonc
provider(name)
// → rows: [{ type: "manifest", auth_scheme, capabilities_required: [ … ],
//            digest, schema_version,
//            origin_ref, origin_digest }]   // both absent where there is no origin: block
//                                           // the header row, emitted first
//         [{ type: "operation", name, kind, opaque, summary }]   // kind: §12
```

```jsonc
operation(provider, operation)
// → rows: [{ type: "operation_detail",
//            source,                     // the Manifest lines declaring this Operation, verbatim
//            derived: {
//              capabilities: [ … ],      // derived from the Manifest, never declared beside it
//              bound_required,
//              patterns_resolved: [ … ],
//              record_cardinality, record_identity,
//              repeatability,            // §12
//              deadline_seconds,
//              concurrency_limit } }]    // effective: 1 where absent, and on every effectful Operation
```

`source` is the Manifest's own lines and not a re-rendering of them: a Manifest is written in the
format the caller is expected to author Definitions in (§3), so returning it verbatim teaches that
format at the moment the caller needs it.

### The repository

```jsonc
targets()
// → rows: [{ type: "target", name, hosts: [ … ],
//            accepts_kinds: [ … ], grants_capabilities: [ … ],
//            credential_env: [ "PROD_TOKEN" ],   // variable names, never values
//            credentials_present }]
```

`credential_env` is exactly what an agent must write into a Target declaration while never seeing a
value, which is the shape §3 fixed when it made a literal in a credential position a load error.

### Authoring

`check` and `review` are the two tools that reach nothing: no credential resolves, no network is
touched, and nothing is invoked, so both answer with no credential present in the environment at all
and neither can move the world however it is called. Together they are the whole author-and-validate
loop, and they are the reason the surface is worth attaching to a repository whose Store is
unreachable.

```jsonc
check(paths?)
// args: paths — an array of repository-relative paths. Every artefact still loads; only the
//               problems positioned in the ones named are reported
// → rows: [{ type: "problem", file, line, column, field, error_code, message }]   // error_code: §12
```

```jsonc
review(artefact)
// text: the full rendered review surface
// → rows: §8's gutter, authority and flag rows, unchanged
```

They stay two tools rather than one because they answer different questions at different moments —
`check` answers pass or fail, `review` answers with a rendering — and merging them would make every
validation pay for a render.

### Execution

The MCP surface executes, destructively included. Restricting it to authoring and reads would make who
is calling an axis of authority, and no guardrail §5 states is a function of that: an unattended Run on
a Cadence is already accepted there, and a call made by an agent with a human watching it is strictly
safer than that. An agent that cannot run also cannot read back the Record it just caused, which is the
loop this surface exists to close.

```jsonc
run(procedure, dry_run?, secret_sink?)
// args: procedure    — the Procedure to run, carrying no target
//       dry_run      — boolean
//       secret_sink  — the Secret sink: an absolute path, outside the repository working tree
// → outcome: §12's triple
// → rows: §8's step, refusal, remediation and provenance rows, unchanged
//         provenance in both scopes: the Run-wide row, then one per Step file written
```

**`run` takes a Procedure and nothing else**, as the command does: every Run is a Run of a Procedure
(ADR-0036), so there is no single-Operation arm of this tool and a call carrying a `definition` is an
argument violating a schema — a protocol error, which is what this surface has in place of exit `2`.

**There is no `inputs` argument.** A Procedure is fully bound by its artefact, and a value supplied at
call time is Step behaviour appearing on no reviewed line — authority arriving after review, which is
the shape ADR-0008 removed and the same shape as the `--force` that is absent everywhere else.

`secret_sink` is the CLI's `--secret-out` under the name of the thing it supplies, a flag named for a
direction having no direction to name in an argument object. It is chosen by the caller and never
defaulted by `hyper`: a sink supplied automatically deletes the guardrail that makes its absence a
Refusal, and makes `hyper` a place a secret lives (ADR-0007). Everything the CLI half states about the
path holds whoever named it. Returning the secret in the tool result is not one of the sink's forms —
it would put a generated credential into an agent's context and from there into whatever transcript
that agent writes to, which is the failure the sink exists to prevent.

```jsonc
probe(provider, operation, inputs?)
// args: inputs — one object keyed by input name, in place of the CLI's repeated flag, and typed
//                at each position by the same schema (§3)
// → rows: [{ type: "probe_result", provider, operation,
//            projection: { … },   // what hyper derived, in the shape a Record would have held
//            response: { … } }]   // the raw response beside it (ADR-0017)
```

A Probe is available here, `read` Kind against `local` only, and it **writes no Record and no Journal
entry** (ADR-0009). Having no outcome triple, its return carries no `outcome` key. The reason this
surface needs it is not agent convenience — writing a file is cheap for an agent — it is that §8's
review model dies by volume: if every throwaway question becomes a reviewed artefact, the set of
Definitions stops being something a human reads, and the oversight story goes with it. The Probe
protects the review surface.

### Inspection

Four tools over the record, taking the typed, closed parameters their commands take and nothing else
(ADR-0013).

```jsonc
runs(since?, procedure?, target?, outcome?, limit?)
// → rows: [{ type: "run", id, started, trigger, outcome, procedure, targets, hyper_version }]
```

```jsonc
run_show(run_id, expansion?)
// → rows: [{ type: "disposition", step, state,   // state: §12
//            records: [ … ],                     // the identities the Step concluded about — the set
//                                                // §8's `records` count is the size of; absent where
//                                                // the Disposition carries none
//            pattern: { attempts, pages, polls },
//            failed_path }]                      // §6's projection failure only; records is partial
//         §8's provenance rows, unchanged: the Run-wide row, then one per Step file written
//         [{ type: "expansion", step, selector, expanded_to, bound }]   // expansion: true only
```

```jsonc
changes(procedure?, since?, between?, target?, record_kind?, limit?)
// → rows: §8's window, asset, observation and code rows, unchanged
```

`record_kind` is the CLI's `--kind`, spelled out: in a flat argument object beside tools carrying an
Operation's Kind, one name cannot hold two senses.

```jsonc
records(target?, definition?, name?, history?, since?, limit?)
// → rows: [{ type: "record", key: { target, definition, name }, ordinal,
//            run_id, step,                   // the version's identity, §8
//            record_kind: "observation" | "asset", tombstoned, orphaned,
//            secret_fields: [ "api_key" ],   // the presence-only marker, per §7
//            provenance: { … } }]
```

### Lifecycle

```jsonc
project()
// → rows: [{ type: "workflow", path, procedure,
//            cadence, phrase, rate }]        // the gloss's parts, §10 — one per Procedure, all of them
```

`project` is not a Run and carries no `outcome` key, and it takes no arguments at all: it is repo-wide
and all-or-nothing above, and there is nothing here for a per-Procedure argument to name.

Its row glosses the expression like every other surface that renders one (§10), carrying the parts
rather than the composed phrase-and-rate line (§8). It carries no last Journal entry: `project` writes a
file and reports what it wrote, and what stands in the Store is no part of that.

## Long Runs

A `run` call is synchronous, and progress arrives as a notification at each Step boundary — the same
boundary §7 writes a Journal entry at, and the same narration stderr carries. It is narration, so it
carries no machine contract and no row of its own.

An asynchronous handle — `run` returning an id the caller polls — is not offered. It invents a Run that
outlives its caller with nothing watching it, which is a daemon with extra steps.

A client that gives up needs no machinery of its own, because §6 already states what happens: the stdio
server dies with the client, and the open Journal entry is closed `failed` by the next invocation with
the Step in flight recorded *attempted, outcome unknown*. That is the truthful account of what happened,
and it falls out of a decision already made.

**A twenty-minute provision is therefore not practically runnable from this surface.** That is not a
gap in it: this surface owns the author→validate→observe loop and short effectful Runs, and long
unattended work is a Cadence on an executor (§10), where there is no interactive client to time out.
Carried forward to §13.

## `hyper` never speaks first

The server sends nothing it was not asked for (ADR-0021). It has no logging channel, it initiates no
message between calls, and the progress notifications above belong to the call that is in flight and
stop when it does. There is no server-initiated request of any kind: `hyper` never asks the client's
model for a completion, since a tool that decides anything by asking a model has moved authority off
the reviewed artefact and onto a prompt; and it never asks the client's user a question, an elicitation
being a prompt and no surface prompting (ADR-0015). A notification you want is a Step you author — a
Definition against a Slack or PagerDuty Target, reviewed, Bounded and recorded like everything else —
and it goes through the front door (ADR-0021).

## What the MCP surface does not do

There is no tool that authors a Definition, a Procedure, or a Target declaration, for the reason no
command does: `hyper` writes what it derives and the agent writes what is reviewed, with its own file
tools. `check` positioned by file and line is what makes that practical, since the next act after a
failed check is an edit — the same act ADR-0001 forces on a human.

There is no approval tool and no confirmation tool. Adding one would make the caller an authority axis,
and it is the per-Run approval §5 states does not exist, reached from a second direction (ADR-0001,
ADR-0015).

**An agent can drive the whole tool and cannot widen its own authority.** No tool takes a bypass
argument, every Refusal returns as an error carrying its own remediation, and the one place third-party
data enters the repository is not reachable from here at all. What an agent can do is author an
artefact and ask a human to read it, which is the same thing a human colleague can do.
