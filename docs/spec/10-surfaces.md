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

Three more commands exist and are not among the sixteen, because none of them reads a repository and none
says anything about `hyper`'s domain: `version` prints the version of the binary that would act,
`completions <shell>` writes a shell completion script, and `mcp` starts the server this section states
below, taking no arguments at all. `version` and `completions` are exempt from the version pin gate below
for that reason — a command that reads no repository has no pin to compare itself against (ADR-0020).
`mcp` needs no such exemption, because the invocation is not the act: what acts on a repository is each
tool the server goes on to serve, and every one of those passes the gate exactly as the command it
carries does, at the moment it resolves one. `mcp` is also the protocol's name rather than a word the
glossary defines, where every name in the tree above is one — which is the second reason it stands here
and not as a seventeenth command (ADR-0088). None of the three ever checks whether a newer version exists
(ADR-0019).

**`hyper` with no arguments writes that tree.** The six groups above, the sixteen names carrying the
positionals this section's table states beside them, and the three commands outside the tree named as
such — then, in a block of its own, the three configuration flags below, titled to say that the sixteen
take them and the three outside the tree do not. On stderr, exiting `2`, like the usage error it is. It
is the only place the command line says what it can do: `completions <shell>` emits every name and emits
them as a shell script, which is not a thing anybody runs to find out what a binary does, and the
orientation below reaches this surface only where somebody has already run `project`. A caller told
only
`usage: hyper <command> [args...]` has been given nowhere else to look, and what it does next is read
whatever is nearby.

**An unknown command names where that list is.** A word that is not one of the nineteen writes what was
typed, the namespace it was resolved against, and the invocation that enumerates it — the same three
things a positional matching nothing writes, applied to the command name, and it suggests no near miss
for the same reason (ADR-0047). It is the pointer and not the page: the tree is what the argument-less
invocation answers with, and rendering it after every typo is narration nobody asked for.

**An unknown flag names what that command takes.** A flag is a name resolved against a namespace like
any other, so it writes the first two of those three — the name that was typed, and the namespace it was
resolved against, which here is the command's own parameters together with the three configuration flags
below. There is no third, and that is the one place this differs: the namespace is a handful of words
rather than a page, so the second line **is** the enumeration and there is no further invocation to point
at. A command with parameters of its own names them; a command with none says so rather than rendering an
empty list. The line is composed from the parameters the command declares, so no command carries a list
of its own flags for this message to read and none can drift from what that command accepts. A token
spelled with one hyphen resolves against that same namespace, there being no single-hyphen flag anywhere
here for one to be, so `hyper check -h` is this message and not a path that does not exist; `-` alone is
a file name and stays a positional, and `--` is what names any other file whose own name begins with a
hyphen, exactly as it was before. It suggests no near miss either, and the ground is stronger than
ADR-0047's rather than borrowed from it: the whole namespace is on the line already, so a candidate
picked out of it would be a name resolved on the caller's behalf out of a list they are reading
(ADR-0098).

None of the three adds a command, an alias, a seventeenth entry or a fourth global. `help` and `--help`
are not among the sixteen and answer `unknown command` as any other word does; `--help` after a command
is an unknown flag as any other token does; nothing above is hidden, and printing the list is the
opposite of hiding one. There is no per-command usage text either: a name that resolves to nothing —
a positional, a command or a flag — is the usage error above, naming the namespace it was resolved
against, and what stands in place of a manual page is Discovery — `providers`, `provider` and
`operation` (ADR-0094, ADR-0098).

Fifteen of the sixteen compare themselves against the version pin in the Repository declaration before
reading a second file and Refuse on mismatch, on a laptop and in CI alike; where there is no pin they
Refuse naming `hyper project` (§4, ADR-0020). The pin gate is stated once here and presupposed by
every command below.

`project` is the sixteenth and it stands outside the gate, which is the one exemption inside the tree.
It is exempt not for being read-only — nothing is exempted for that, and §11 says so — but for being
**the pin's only writer**: a gated `project` on an unpinned repository would Refuse naming itself, and
a gated `project` under a newer binary would Refuse naming itself, which makes the upgrade ritual
unperformable at its second step. A writer gated on what it writes is a bootstrap with no bootstrap.
Nothing about ADR-0001 is softened by it: `project` does not proceed under a pin it disagrees with, it
*replaces* that pin and writes the replacement into a tracked file whose diff is the review.

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
problems and `hyper check definitions/typo.yaml` exits `0` clean on a job that checked nothing. **A
path resolving outside the repository is the same `2` for the same reason**, and it is decided before
the stat rather than by it: such a path names nothing this command could report on, however well it
names a file on the caller's own disk (ADR-0089).

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
beside them the facts the source does not carry in that form: the one Capability `hyper` derives from
it, whether a Bound is mandatory, illegal or neither, the Patterns it resolves to, its Record
cardinality and declared identity field, its Repeatability, its deadline, and its concurrency limit.
The source verbatim, because a Manifest is written in the format the caller is expected to author
Definitions in (§3); the derived facts beside it, because making the caller re-derive what `hyper` has
already computed is waste.

**The Bound fact has three members and is not a boolean.** §5 gives it three states: a `destroy` Step's
Bound is mandatory, an `opaque` `destroy` Step is the one Step that carries no Bound — writing one there
is refused (`bound-missing`, `bound-illegal`) — and on the rest none is mandatory, a `read` Step having
nothing for one to guard and a `mutate` Step's being its author's to write or leave out. `false` would
carry both *you need not write one* and *writing one is refused* under one value, on the most severe
Operation the tool runs, so the member is named `bound` and renders `mandatory`, `illegal` or `none`.
The MCP tool below carries the same name and the same set.

`patterns_resolved` goes out as a list and is empty rather than absent where the Operation declares no
Pattern: a caller asking which Patterns run around this call is answered *none of them*, which is a
fact, where an absent member would say the question was not asked. The page states the same thing by
having no line, which is the rule every labelled value on it follows. The three members `hyper` derives
from a declaration — the Capability, the Bound and the Repeatability — are absent only where the
Manifest states nothing legible to derive them from, which is a fault `check` reports and not a value
this row may substitute for (§4).

**The Repeatability reported there is the effective one**, on the same ground as the limit below: an
Operation whose Manifest omits `repeatability:` is run-once where it effects and `repeatable` where it
reads (§12). Run-once is rendered even though no artefact may write that word, which makes it exactly
parallel to `opaque` — a fact no artefact declares and every surface renders.

The concurrency limit reported there is the effective one and is always present: 1 for a `read` whose
Manifest omits the key, and 1 for every `mutate` and `destroy`, whose Expansion is serial and which may
not declare the key at all (§3, §6). A caller asking *how many at once* gets a number for every
Operation, and the rule about what may be **authored** stays where authoring rules live rather than
being inferred from a field that came back empty.

The deadline goes out as `deadline_seconds` and renders on the page as the duration the Manifest wrote —
`30s`, not `30`. The wire fixes a unit so nothing downstream parses a suffix; the page stands directly
beneath the source, and a second spelling of one fact on one screen is what the verbatim lines are there
to prevent.

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

**A slot naming no variable carries neither the variable nor a presence.** There is nothing to ask the
environment about, so `false` would answer a question nothing asked: an `env:` absent or unreadable is
`credential-slot-malformed`, which `check` reports, and no zero value here may stand in for it (§7,
ADR-0064). Where a variable is named, presence is written whichever it is. Both surfaces carry the row
that way.

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

**A path positional is read against the repository root, and the repository bounds it** (ADR-0089). A
relative path is joined to that root, an absolute one is read as itself, and either way a path that
resolves outside the repository names nothing to report on and is the usage error above. The caller's
working directory decides *which repository* is being checked and never *which file inside it* is
being reported on — so one string names one file wherever it is typed, and the spelling a path
positional takes is the spelling `check`'s own rows come back in. The relative form is the one
`review <artefact>` already takes — its path resolves against the load's own paths and never against
a working directory — and `check` admits the absolute spelling beside it because it stats a file
rather than matching a loaded artefact's own path.

Its output is one row per problem — the file, the line, the field, the `error_code` §12 defines, and a
message — positioned so that the next act is an edit, ordered by file path and then by line. A Run
re-runs every one of these rules at its start (§6) and reports the same problems in the same order under
its own row type, one shape arriving through two commands (§7, ADR-0061); what the two differ in is the
exit code, `1` for a command reporting what it found and `77` for a Run a guardrail declined. There is
no `check --fix`: a checker that can
also mutate is a checker you stop trusting, and a repair flag on a gate is the shape ADR-0001 removed
elsewhere. What repairs projection drift is `project` below, which is a separate command for that
reason.

`review <artefact>` renders §8's Definition review of the artefact named. **Its positional takes two
forms**: one containing `/` or ending `.yaml` is a repository-relative path, resolved against the load's
own paths; anything else is a name, resolved against the artefacts' own `name:` by the rule above. They
are two namespaces rather than two spellings, and each reaches what the other cannot — a name here is
matched across all four artefact namespaces at once and can be ambiguous in them, and an artefact whose
file will not parse declares no name at all and is reachable only by its path. A form that fell back to
the other would make which artefact `hyper review deploy` renders depend on what else is in the
repository, so the discriminator decides which namespace is looked in and never what is tried first
(ADR-0060, ADR-0090). It resolves no credential,
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
of a Procedure (ADR-0036). **Its positional is that name and takes no second form** — `deploy` names the
Procedure and `procedures/deploy.yaml` names the file — so a path-shaped positional is a name that
resolves to nothing like any other, exit `2`, with a sentence saying which grammar this one takes. That
`review` takes both forms and this takes one is the two *commands* differing rather than the two
positionals: `procedures/` is one namespace, so a Procedure's name is unambiguous in it, and a Procedure
whose file will not parse cannot be run at all — the Run-start `check` declines the load before Step 1 —
so a path here would reach no Procedure the name does not already reach. What a caller holding a path needs
is the rule rather than a second argument: `name-mismatch` pins a name to its file's basename (§4), and
that name is what the Journal entry carries and what `changes <procedure>` takes back (§8, ADR-0090).

A single Operation invoked directly through a Definition is not a second form of this command — there is
no such invocation, and handing `run` two positionals is a usage error. Where the one positional names an
artefact that exists and is a Definition rather than a Procedure, that is a usage error too, the two
living in different directories and the kind being known before anything loads (§3).

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
would have read — with the Steps after it *never reached*. **Which Step that was is a row's fact and
not the page's alone**: that Step's `step` row carries `withheld: true`, and no other row carries the
key (§8). Its outcome is `completed` and it exits 0, a halted rehearsal being the correct outcome of a
correct operation rather than a failure; the answer is partial, and it says so on the page rather than
in the code. The flag is not global: a `records --dry-run` or a `check --dry-run` would have to mean
something, and neither does.

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

**The Repository declaration is the second thing it writes, and it is edited rather than regenerated.**
Two scalars move in it — the `version:` pin and the `digest:` beside it — and every other byte is
carried through: `retention:` is authored, and so are the comments and the layout. The whole-file rule
above governs files that are *entirely* derived, and `hyper.yaml` is a reviewed artefact carrying two
derived facts. Where the repository holds none, `project` creates one carrying `kind:`, `version:` and
`digest:` and **no `retention:`** — a repository that has not stated a policy has not agreed to lose
anything (§3, §11).

**`AGENTS.md` is the third thing it writes, and it is created or left alone.** Where the repository
holds none, `project` writes the orientation below into it, at the version of the binary that ran;
where one stands it is not touched, on any run, ever. This is the one path in the namespace that is
not derived from an artefact and the one that is not regenerated, and both follow from what the file
is: a note addressed to a reader, which most repositories already hold for reasons having nothing to
do with `hyper`, and which whole-file overwriting would take. It carries no authority, no Run reads
it, `check` does not count it, and it writes no row — like the declaration, it lands in the diff
`project` is read in (ADR-0093, ADR-0095).

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
`store init` answers, `compact` included. `probe` reaches `77` too, and by one route: it touches no
Store and against `local` the two-key check is vacuous, but the **grant** is a guardrail all the same,
so a host outside the `hosts:` the Target named `local` declares is `host-not-granted` and exits `77`
(above, ADR-0042). A repository declaring no `local` grants no host and declines identically. What a
Probe can never exit is `1`: it is not a Run, and a status is an answer rather than a problem found.

A signal is handled as §6 states: the first interrupt drains, the Step in flight finishes, no further
Step starts, and the Run closes its own Journal entry `failed`, exiting `130` or `143` according to the
signal it received. Only a second signal kills the process, which is what leaves the open entry a later
Run closes. Both codes are the CLI's alone: the MCP server installs no signal watch, so neither is
reachable from it and what stops a call there is the client cancelling it (below, ADR-0092).

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

## The handshake carries the orientation

**`initialize` answers an `instructions` field, and `hyper` fills it.** It is the one thing on this
surface that reaches an agent *before its first tool call*, and it needs no setup beyond the client
config a user already writes.

**It is the same text `project` writes to `AGENTS.md`, and the two channels are one text for that
reason.** A client decides when it surfaces `instructions` — one harness carries them only inside a
tool search, and a session observed against a fresh repository recovered the orientation by running
`strings` over the binary rather than reading it — so *before the first tool call* is a property of
the client and not of the protocol. A file in the repository has no such contingency; a repository
nobody has run `project` in has no file. Neither channel reaches every cold start alone, and a
second orientation maintained beside the first disagrees with it the first time either is edited
(ADR-0095).

**So it is worded for a reader on either surface.** It names commands rather than tools — `show` and
not `run_show` — and it puts `install`, `store init` and `compact` out of reach as *the human's*,
which is true of both, rather than as *absent from this surface*, which is true only of the server and
is read as permission by the reader holding a terminal.

It states nine things, each of them something the tool set cannot teach. Every tool carries a
description and `operation` goes further — it answers *the Manifest's own lines, verbatim*, which
teaches the authoring format at the moment a caller needs it — but all of them arrive with a call
already in mind, and none of these is about a call:

- **What `hyper` is**: the agent authors, a human reviews, every effect is recorded.
- **The five artefacts** (§2) and where each lives, since `operation` teaches one of the five and
  nothing anywhere teaches the other four.
- **The loop**: read what is here, author with your own file tools, `check`, repair, `review`, hand
  the diff to the human, and `run` only once they have read it.
- **The four verbs an operator asks for** — author, change, retire, operate — and the fact each needs
  before it starts: that a changed artefact is reviewed again, that deleting a Definition abandons its
  Assets rather than destroying them (ADR-0012), and that a declared Cadence left unprojected is
  `projection-stale`.
- **What halts and what is merely an answer.** A `read` never halts on what came back and records an
  absence as readily as a status (§6, ADR-0050); an effectful Operation completes on `2xx` and halts
  otherwise. Narration is not an outcome — `probe` prints `no response arrived` and exits `0` — and an
  agent that reads the prose beside a result rather than its Disposition reports a Run that halted when
  none did.
- **The three commands that are the human's, and why** — an agent that does not know runs them, which
  is the exact bypass their absence from the tool set exists to prevent. They are stated as *the
  human's* rather than as *absent here*, because the second is true only of this surface and the same
  text is read by an agent holding a terminal, where absence is read as permission (ADR-0095).
- **That a Refusal is final**: the same call retried refuses identically, and no argument anywhere
  widens the caller's own authority.
- **One worked example of all five artefacts, which checks clean.** A fresh repository holds only the
  built-in `shell` Manifest, whose request block is `shell: {}` — so an agent asked for an HTTP
  Provider has no worked example of a request, an `auth:` scheme or a `record:` projection anywhere in
  reach. The example carries the Repository declaration at **the version of the binary that would
  act** (ADR-0020). It carries **both request shapes**, and the second is not decoration: a
  single-host request with an `auth:` scheme, and a multi-host one (`host: "{from-target}"` with
  `host-input:`) are different shapes, and an example that taught only the first sent the first agent
  that needed the second to disassemble the binary for it. **One of the two is carried whole and the
  other as a fragment, and which is which is fixed here**: the whole one is effectful — a `mutate`
  beside a `destroy`, an `auth:` scheme, a `record:` whose `identity:` is a hole, a Step with a
  selector and a Bound — because those are the rules no tool call and no format prose state, and
  because the multi-host `read` is the task a fresh repository's first agent is asked for, which an
  example carrying it whole answers by transcription. Carrying both whole is refused for length, this
  text being paid for on every session in every harness (ADR-0096).

  **A rule stated here is stated with its exception, or it is not stated.** The prose beside the
  example is what an agent authors from, and a rule carrying half of itself is worse than an absent
  one: the agent authors the half it was given, `check` declines the artefact, and the orientation has
  spent a repair loop on the one surface whose whole job is to save one. The Bound is where this was
  learned — *mandatory on a `destroy`* is true and is half the rule, an opaque `destroy` refusing one
  (§5) — and the constraint the exception is stated under is the budget above, so it is a clause of
  the sentence that already stood rather than a paragraph beside it. A claim this text makes about a
  `check` is held to that `check` by a case, so that the two cannot come to disagree
  (`TestInstructions_TheBoundRuleIsTheOneCheckHolds`, ADR-0101).

- **That the agent should offer to add a section to an `AGENTS.md` that already stands.** `project`
  writes the file where a repository holds none and never overwrites one that does, so a repository
  that held the file for its own reasons is the single case neither channel reaches — and offering a
  section is what closes it. The agent offers and the human accepts, which is the same act as any
  other file it authors (ADR-0093, ADR-0095).

**It is not `hyper` speaking first.** ADR-0021 governs egress on `hyper`'s own initiative; this is a
field of the answer to a request the client made. Nothing is initiated, and no destination is named.

**No artefact is scaffolded.** `project` writes the orientation and nothing else: the worked example
is a fenced block in a Markdown file, which teaches the format while granting nothing and being
counted by no `check`, and a Target or a Definition on disk would be `hyper` authoring authority — the
line the whole surface does not cross (ADR-0093, ADR-0095).

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
The three commands outside the sixteen get no tool either: a client writes no completion script, the
version of the binary that would act is the version of the server the client started, and `mcp` is
that invocation.

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
artefacts already reviewed — the note beside them being the one exception, and an orientation carrying
no authority is not a thing to guard a surface against — and all of it lands in a diff like everything
else.

## The return envelope

Every tool returns one shape.

```jsonc
{
  "content": [{ "type": "text", "text": "…" }],
  "structuredContent": {   // absent where a tool declined before it opened a row stream
    "outcome": "completed",   // §12's triple, on the execution tools only; absent elsewhere
    "run_id": "01991ea6-b118-7c93-8d41-6b2f7ae05c19",  // whole; absent where no entry was written
    "dry_run": false,         // beside it, `false` included, wherever `outcome` is present
    "rendering": "…",         // the text block's page, on `review` alone; absent elsewhere
    "rows": [ … ],            // §8's rows, as an array
    "truncated": false        // the marker, or the bare boolean, or null on a Run
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

**The hint names the tool's arguments where the terminal's names its flags, and that is the only
wording an answer changes between the two surfaces.** A hint reading `--since` where nothing takes a
flag would point a caller at an argument no schema declares; the axis, both counts and every row
beside them are the command's own, unchanged. All four members are written always — they are counts a
reader subtracts, and an absent key would read as *unknown* where the fact is *none*. There is no
cursor here and no way to ask for the next N: the remedy for a truncated result is a narrower
question, and a truncated result must never look complete.

**Each tool's `outputSchema` admits exactly the shapes its command writes.** The three tools whose
command orders on an axis a `limit` can cut — `runs`, `changes`, `records` — declare `truncated` as
the bare `false` or the marker above; `run` declares it `null`; the rest declare the boolean §8's
terminal row carries there. A schema admitting less than its tool answers is the same failure as a
Refusal answering past one (ADR-0102).

**The `text` block is asymmetric, and the asymmetry is the point.**

| case | `text` carries |
| --- | --- |
| any ordinary return | one summary line, outcome first |
| `check` | that line, and beneath it the rows as §8's renderer drew them |
| `review` | the full rendered review surface — the gutter, `AUTHORITY`, `FLAGS` |
| a Refusal | the full rendered Refusal — Step table, caret excerpt, `EDIT ONE OF`, retry sentence |

Outcome first repairs what §8's row stream accepts knowingly: with rows the outcome arrives last, and
the terminal compensates with an exit code, which this surface does not have. On `run` that line names
the Run id after the outcome, §8's terminal line arriving here as a sentence: a client has no exit code
to read and no scrollback to search, so an entry the envelope does not name is one an agent can reach
only by asking which Run was the last, on a Store two environments write. The two full renderings
are the same trade §8 made for the same reason — with no bypass anywhere, the Refusal rendering is the
entire remediation path (ADR-0001).

**`check` has a row of its own because a `problem` row is a remediation and not a result** (ADR-0097). A
client is not obliged to surface `structuredContent` to the model behind it, and most do not; an agent
told *how many* problems there are and not *what* they are has, in place of the edit the row already
describes, only the count to guess against. That is the Refusal's argument arriving on the return an
agent meets far more often — it is `check` an agent calls after every write — and it is what MCP's own
guidance asks of a server returning structured content: serialise it into the `text` block as well.
`structuredContent.rows` is unchanged; this is one row set on two channels, drawn by §8's one renderer,
and the terminal and this surface still cannot drift apart.

The summary line survives above the rows, and carries the truncation marker where there is one: a
result cut on an axis says so on the line a reader meets **before** the table rather than after it,
which is how *a truncated result must never look complete* stays true once the rows are in the block.
What goes beneath the line is the row set, so a `check` that found nothing puts nothing there — the
clean answer is the summary line alone, `check`'s own page being a sentence about a count and not a
table. No other ordinary return carries its rows: a listing is a result set, and a table in its `text`
block would say twice what `structuredContent` says once.

**`review`'s page is written into the structured content as well** (ADR-0100). MCP's two halves are
asymmetric, and not in the direction the table above assumes: `structuredContent` is the result —
*servers MUST provide structured results that conform to this schema*, wherever an `outputSchema` is
declared — and the `text` block beside it is *the serialized JSON*, returned **for backwards
compatibility**. A promise kept only in `content` is therefore kept in the half the protocol itself
calls redundant, and a client that surfaces the structured half in preference to it is reading the
half the protocol made normative rather than misbehaving.

`review` is the one tool with a page to lose there. Its rows are what the page is drawn *from*, so an
agent handed them is handed the ingredients and not the surface a human will read — and `review` is
the tool an agent is told to call before handing work back, which it cannot do against a page it
never sees. So the envelope carries `rendering`: the same string the block carries, byte for byte,
standing above the rows for the reason the summary line stands above them in the block, and required
by `review`'s `outputSchema`. **Nothing else carries it.** A listing's summary line and a `check`'s
block are composed of members already here — the counts are the rows, the triple and the id are keys
of their own, the marker is `truncated` — and a Refusal needs no second channel, MCP naming no
structured one for an error at all: its whole error mechanism is `isError` and the `content` beside
it, so `content` is the only place a Refusal is ever put. That is why `rendering` can be `required`
without a `review` the version pin gate declines failing its own schema: a declined call carries no
structured half at all (below), so there is no half for the member to be missing from (ADR-0102).

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

**Such a call carries no `structuredContent` at all** (ADR-0102). Every tool here declares an
`outputSchema`, and MCP is flat about what that obliges: *servers MUST provide structured results that
conform to this schema*. A tool that declined before it opened a row stream produced no result to
conform — and a half composed for it anyway says `rows: []`, which reads as *this tool ranged over a
namespace and found nothing* where the fact is that it never ranged over one. So the envelope on that
path is the `text` block and the bit, which is MCP's own shape for a tool that failed: the protocol's
error mechanism is `isError` and the `content` beside it, and it names no structured channel for one.

**The rule is keyed on the stream and not on the bit or the code.** `run` is on §8's `outcome` side on
every path a Run was attempted, the two that decline before a Run is identified included, so a `run` a
guardrail declined answers a structured half carrying `outcome: refused` — and a Run that lost the
Store before `run.json` answers one carrying `failed`, with `truncated: null` beside it, a Run having
no result set for a limit to have cut. Both are `isError: true`. What decides is whether the command
produced anything for the half to hold, which is why the twelve tools that are not `run` are the ones
that answer `content` alone.

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
//              bound,                    // three members: mandatory | illegal | none
//              patterns_resolved: [ … ],
//              record_cardinality, record_identity,   // both absent on a destroy, which projects
//                                                     // no Record of its own (§3)
//              repeatability,            // §12, and effective: run-once where the Manifest omits it
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
//            credentials: [{ slot, env: "PROD_TOKEN", present }] }]   // names, never values
//                                                                    // absent where the declaration
//                                                                    // carries no auth: block
```

`env` is exactly what an agent must write into a Target declaration while never seeing a value, which
is the shape §3 fixed when it made a literal in a credential position a load error. Presence is
computed when the tool runs; the value behind a present name is never read here and never rendered
anywhere.

**The credential grant renders as `credentials`, one member per slot, each pairing the slot with its
variable and that variable's presence.** An earlier sketch of this row named two flat members,
`credential_env` beside a `credentials_present`, and the CLI half above has always required the pair:
a declaration may carry slots for more than one scheme, and a list of names alone does not say which
fills what. §12's opening rule — one fact reaching two wires reaches them under one name — decides it
in favour of the shape that can state the fact, exactly as it decided `hosts` above. `env` and
`present` are absent together on a slot naming no variable, on the rule the CLI half states.

### Authoring

`check` and `review` are the two tools that reach nothing: no credential resolves, no network is
touched, and nothing is invoked, so both answer with no credential present in the environment at all
and neither can move the world however it is called. Together they are the whole author-and-validate
loop, and they are the reason the surface is worth attaching to a repository whose Store is
unreachable.

```jsonc
check(paths?)
// args: paths — an array of repository-relative paths, read against the repository root exactly
//               as the command reads them: an absolute path inside the repository is admitted,
//               and one resolving outside it is refused (ADR-0089). Every artefact still loads;
//               only the problems positioned in the ones named are reported
// text: the summary line, and beneath it the rows as §8's renderer drew them (ADR-0097)
// → rows: [{ type: "problem", file, line, column, field, error_code, message }]   // error_code: §12
```

```jsonc
review(artefact)
// text: the full rendered review surface
// → rendering: that same surface, byte for byte — the page on the structured channel too (ADR-0100)
// → rows: [{ type: "artefact", kind, path, baseline | baseline_absent,   // the header row,
//            cadence, phrase, rate, last_run }]                          // emitted first
//         §8's gutter, authority and flag rows, unchanged
//         [{ type: "problem", … }]   // check's own rows, where the artefact under review
//                                    // is found and will not load
```

`review`'s `text` is what the command writes to stdout, byte for byte, on **every** path that answers
at all, and `rendering` is that same string beside it. That includes the one where the artefact is
found and will not load: the tool answers `check`'s rows and `check`'s table, on both channels, with
`isError: true` where the command exits `1`. The text-block table in the return envelope above is
keyed on the **tool** rather than on what the tool found, and a reading that swapped in a summary line
on one of the command's own paths would break the promise exactly where an agent is least able to
check it. An artefact that is not there **at all** has no row to write and is the usage error, which
arrives here as a JSON-RPC error like every other. A guardrail declining is the Refusal row, and it
carries **no structured half at all** — so `rendering` is absent there along with every other member,
and this schema goes on stating what the tool answers without qualification (ADR-0102).

The two extra row types are the sketch's list made complete rather than widened. The header is a row
like any other, on the rule the envelope states above, and it is the row carrying the range the
review opened at. The `problem` rows are the CLI half's *would not load* arm, which was always
`check`'s row rendered under `review`; an `outputSchema` is declared once and for every call of the
tool, so both belong in it.

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
// args: procedure    — the Procedure's name, byte-exact and never a path, carrying no target
//       dry_run      — boolean
//       secret_sink  — the Secret sink: an absolute path, outside the repository working tree
// → outcome: §12's triple
// → rows: §8's step, refusal, remediation and provenance rows, unchanged
//         provenance in both scopes: the Run-wide row, then one per Step file written
//         a rehearsal that stopped: withheld: true on the Step it withheld, and on no other row
```

**`run` takes a Procedure and nothing else**, as the command does: every Run is a Run of a Procedure
(ADR-0036), so there is no single-Operation arm of this tool and a call carrying a `definition` is an
argument violating a schema — a protocol error, which is what this surface has in place of exit `2`. It
takes that Procedure **by name**, in the command's one form and not `review`'s two, and its schema says
so: a tool builds the command line its command would have received, so a form advertised here and
resolved by no command is a malformed call for a Procedure that is there (ADR-0090).

**There is no `inputs` argument.** A Procedure is fully bound by its artefact, and a value supplied at
call time is Step behaviour appearing on no reviewed line — authority arriving after review, which is
the shape ADR-0008 removed and the same shape as the `--force` that is absent everywhere else.

**A `dry_run` call is answered in both halves**, which is what the `withheld` member on §8's `step` row
is for. An agent calling it asks *what would this do, and where does it stop*, and the second half is a
fact about one Step that no Disposition carries: the Step a rehearsal withheld is *never reached* like
every Step behind it. The rows carry it on that one Step, so this surface reads the boundary of a
partial answer rather than inferring it from an outcome and a marker — an inference sound only under
`completed` and `dry_run: true`, and one a Run the world resisted breaks (§8, ADR-0091).

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
// → rows: [{ type: "entry", run_id, procedure, trigger: { … }, started_at, dry_run,
//            outcome, ended_at, closed_by: [ … ] }]   // the header row, emitted first
//         §8's refusal and remediation rows, where the entry's own Run recorded a Refusal
//         §8's provenance rows, unchanged: the Run-wide row, then one per Step file written
//         [{ type: "step", step, id, path, definition, operation, provider, target, kind,
//            disposition,                        // §12's set
//            records: [ … ],                     // the identities the Step concluded about — the set
//                                                // §8's `records` count is the size of; absent where
//                                                // the Disposition carries none
//            pattern: { attempts, pages, polls },
//            selector: { declared, expanded_to, bound },   // expansion: true only
//            projection_failed_path }]           // §6's projection failure only; records is partial
```

**The rows are `show`'s own, and an earlier sketch of this tool named others.** It gave each Step a
`disposition` row carrying a `state` and a `failed_path`, with the Expansion split off into rows of
its own, where the command has written `entry`, `refusal`, `remediation`, `provenance` and `step`
rows since it landed — and what that sketch called `state` is the `disposition` member of the row
that carries it. §12's opening rule — one fact reaching two wires reaches them under one name —
decides it in favour of the shape the command already writes, exactly as it decided `credentials`
above: a second shape for one entry is where the two surfaces come to disagree about what a Run did.
The Expansion rides on the Step it belongs to for the same reason, under the argument that asks for
it, and the header and the Refusal rows are the sketch's list made complete rather than widened —
`show` renders both, and an `outputSchema` is declared once and for every call of the tool.

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
carries no machine contract and no row of its own. It carries the Step's position, how many the Run
holds, and the Step's authored id; with the outcome arriving only when the call returns, a silent
twenty minutes is otherwise indistinguishable from a hang.

**A notification is sent where the client supplied a progress token and nowhere else.** That is the
protocol's rule and `hyper`'s at the same time: without a token there is nothing for a notification to
be correlated with, and sending one anyway would be the server speaking unasked (ADR-0021).

**A Run naming itself before its first Step sends nothing here.** On the CLI that line exists because
the terminal line is not always reached, and the Run that dies before it is the one Run whose identity
its own output would otherwise never carry (ADR-0047). On this surface the id arrives in the summary
line and in `run_id`, and a client that gives up gets no delivery at all — so a notification naming it
would be narration with no reader on the one path it was invented for.

An asynchronous handle — `run` returning an id the caller polls — is not offered. It invents a Run that
outlives its caller with nothing watching it, which is a daemon with extra steps.

**A cancelled call drains.** The client cancels a call by cancelling the request, and that is mapped
onto the drain §6 already states for the first interrupt: the Step in flight finishes, no further Step
starts, and the Run closes its own Journal entry `failed` (ADR-0015, ADR-0092). It is one stopping
mechanism read a second way rather than a second mechanism — the gate §6 states is a predicate asked
where the next Step would start, and a cancelled call is that predicate answering.

**No signal watch is installed by the server.** The process's signals belong to the client that spawned
it and not to any one call in flight, so a Run on this surface is one nobody interrupts by signal and
the exit codes `130` and `143` are unreachable from it. That costs nothing: the envelope carries
`failed` and the Journal carries the truthful account, which is what the code would have added a number
to.

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
