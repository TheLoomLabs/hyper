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
| **Execution** | `run <procedure>` \| `run <definition> <operation>` · `probe <provider> <operation>` |
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

## Discovery

Three commands rather than one taking optional arguments, because they are three questions asked in
order — *which Provider*, *which Operation*, *how do I call it* — and a return shape that changes with
which argument was omitted is unusable to the caller that most needs it.

`providers` writes one row per Provider `hyper` can load, built-in and Extension alike: its name, its
origin, a summary, how many Operations it exposes, and its digest.

`provider <name>` writes one row per Operation the named Provider exposes — its name, its declared
Kind, whether it is `opaque`, and a summary — beside the Manifest's own facts: its Auth scheme, the
Capabilities it requires, its digest, and its schema version. Kind is on every row at this level
because it is what answers the two-key question (§5) before a single input schema has been read.

`operation <provider> <operation>` writes the Manifest lines declaring that Operation, verbatim, and
beside them the facts the source does not carry in that form: the Capabilities `hyper` derives from it,
whether a Bound is mandatory, the Patterns it resolves to, its Record cardinality and declared identity
field, its Repeatability, its deadline, and its concurrency limit. The source verbatim, because a
Manifest is written in the format the caller is expected to author Definitions in (§3); the derived
facts beside it, because making the caller re-derive what `hyper` already computed is waste.

## The repository

`targets` writes one row per Target declaration: its name, its endpoint, the Kinds it accepts, the
Capabilities it grants, the environment variables its credentials resolve from — names only, never a
value (§3, ADR-0007) — and whether each of those variables is present. Presence is computed when the
command runs and is the same check a Run performs before its first Step (§6); the value behind a
present name is never read here, and never rendered anywhere.

## Authoring

`check [path...]` runs every rule §4 states, together. With no argument it loads every artefact in the
repository; with paths it still loads every artefact and reports only the problems positioned in the
ones named, because every rule §4 states compares one artefact against another and a subset of the
repository is therefore not checkable on its own. It resolves no credential, reaches no network, and
invokes nothing.

Its output is one row per problem — the file, the line, the field, the `error_code` §12 defines, and a
message — positioned so that the next act is an edit. There is no `check --fix`: a checker that can
also mutate is a checker you stop trusting, and a repair flag on a gate is the shape ADR-0001 removed
elsewhere. What repairs projection drift is `project` below, which is a separate command for that
reason.

`review <artefact>` renders §8's Definition review of the artefact named. It resolves no credential,
reaches no network, and invokes nothing, so it runs offline against a repository whose Store is
unreachable, with the Cadence gloss §10 states degrading rather than the command failing. A `FLAGS` row
is a fact about the artefact rather than a problem with it, so a review that rendered exits 0 however
many flags it carried; only an artefact that would not load exits 1, and what it writes then is
`check`'s row.

## Execution

`run` has two forms and the difference between them is authority, not sugar.

`run <procedure>` runs the named Procedure. It takes no `--target`: a Procedure is fully bound and
declares its own Target envelope (§5), so a Target supplied at invocation is either redundant with the
artefact or it is authority arriving after review, which is what ADR-0008 removed. Supplying one is a
usage error rather than a silently ignored flag.

`run <definition> <operation> --target <target>` invokes a single Operation through a Definition.
`--target` is required here — nothing else names the Target — and omitting it is a usage error.

Neither form carries an argument value. What the invocation carries is the occasion and never
authority: a Secret sink, a dry-run marker, and output formatting (§6, ADR-0008).

`run` renders nothing before executing (ADR-0015). What it writes is the Step table §8 states, each
Step's Disposition and the count of Records it wrote, and, where a guardrail declined, §8's Refusal
rendering in full. Under `--json` it emits §8's rows terminated by the `outcome` row.

**`--dry-run`** is accepted on `run` and on no other command. It performs the reads it reaches and
stops rather than simulating an effect, and it writes a Journal entry marked as a dry-run (§6). A
dry-run that stopped renders that it stopped and why — the withheld Step whose output the next one
would have read — with the Steps after it *never reached*. Its outcome is `completed` and it exits 0,
a halted rehearsal being the correct outcome of a correct operation rather than a failure; the answer
is partial, and it says so on the page rather than in the code. The flag is not global: a `records
--dry-run` or a `check --dry-run` would have to mean something, and neither does.

**`--secret-out <path>`** names the Secret sink. A Step whose Operation declares a secret output
Refuses when none was supplied, which is a fact about the invocation and never about the environment it
runs in. The path is written `0600` and is refused where it resolves inside the repository working
tree; `-` is not accepted, stdout being exclusively the answer and a secret written there landing in
the same pipe a CI job logs. It is not a bypass and must not read like one: supplying it weakens no
check, and withholding it produces a Refusal that renders like any other (§8).

`probe <provider> <operation>` invokes a `read` Operation against `local` without a Definition. Inputs
are supplied as repeated `--input <name>=<value>`, each typed by the Operation's declared input schema
at that position rather than by what the value looks like (§3); an unknown name, or a required input
left out, is a usage error. A Probe writes no Record and no Journal entry, has no Trigger, no
Provenance and no Disposition, and can never be scheduled, sequenced into a Procedure, or used as a
Comparison baseline (ADR-0009). Having no outcome triple, it terminates its row stream with `result`
and never with `outcome` (§8). It may surface the raw response beside the projection `hyper` derived
from it, which no credentialled surface does (ADR-0017).

## Inspection

Four commands over the record, taking typed, closed parameters and nothing else. There is no predicate
dialect over them and none behind them: a caller wanting an arbitrary filter takes the rows and applies
it themselves (ADR-0013).

`runs` takes `--since`, `--procedure`, `--target`, `--outcome`, and `--limit`, and writes one row per
Journal entry: the Run id, when it started, its Trigger, its outcome, its Procedure, the Targets it
bound, and the version of `hyper` that performed it. The Trigger is on every row, being the only thing
that distinguishes a world that has not changed from one nobody has looked at (§7).

`show <run-id>` writes one entry in full: each Step's Disposition with the Record identities it acted
on and `hyper`'s own account of what it did to reach that outcome — a Pattern's attempts, its pages,
its poll iterations — beside the entry's Provenance (§7). Under `--expansion` each Step also carries
its selector, what that selector expanded to, and its Bound, which is what §8's Refusal footer points
at.

`changes [procedure]` renders §8's Comparison. Naming a Procedure selects it and omitting one compares
across every Procedure at once, which is why the Procedure is positional here and a parameter on `runs`
— it decides which rendering you get rather than filtering the rows of one. It takes `--since
<timestamp>` or `--between <run-id> <run-id>`, and `--target`, `--kind`, and `--limit`; `--since` and
`--between` together is a usage error, the two being different ways of naming one window.

`records` takes `--target`, `--definition`, `--name`, `--history`, and `--limit`, and writes one row
per Record: its identity, its version, whether it is an Observation or an Asset, whether its head is a
Tombstone, which of its fields carry the presence-only secret marker (§7), and its Provenance. It
returns the Head only unless `--history` is given — an explicit boolean rather than a mode that turns
itself on when some other parameter is named (ADR-0013). An Asset whose Definition no longer exists is
marked Orphaned on every row that carries it, for as long as it stands (§7).

Every command in this section takes `--limit`, with a modest default, and every truncated result
carries a marker naming the axis, what was returned, and what was dropped. There is no cursor and no
pagination: an unbounded return blows a context window on the first interesting month, and walking
three thousand rows a page at a time is the same disaster arriving politely. Truncation is a signal
that the question was too broad, and the marker names the axis so the next call is a narrower question.
A truncated result must never look complete.

## Lifecycle

`install <ref>` fetches an Extension and writes the tracked, digest-verified file §11 states. It is the
single point at which third-party data enters the repository, which is why what it writes is a tracked
file appearing in a diff a human reads. It takes no `--dry-run`; `check` already reports digest drift.

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
reviewed artefact being the line the whole surface does not cross. Every command that needs the Store
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
arise.

**stderr is the narration.** Progress, warnings, and the human rendering of an error, always, in both
modes.

**Progress is one line per Step boundary, in both modes, always on.** With the outcome arriving last, a
silent twenty minutes is indistinguishable from a hang. It is narration, so it carries no machine
contract and has no `--json` variant: a consumer wanting Step-level structure reads the Journal, which
§7 writes per Step as that Step reaches its Disposition.

**The last row is always the terminal row**, and its absence means the stream was cut off. There are
two, `outcome` for a Run and `result` for everything else, and §8 states both. A Probe is on the
`result` side, having no outcome triple to report (ADR-0009).

**`error_code` is closed and has two halves.** The set §12 states holds both: a Refusal's is the
identifier of the check that declined it, minting no vocabulary of its own, and a failure's is one
`hyper` owns. No `error_code` is ever Provider-supplied — a Provider is data and can no more invent one
than it can invent an Auth scheme (ADR-0004) — because a CI-facing contract that grows by extension is
one where an Extension author mints a code somebody's script treats as retryable.

**Colour and width are the only differences between a terminal and a CI log**, and colour never carries
a fact of its own (§8, ADR-0015). The job summary §10 states is those same renderings relocated rather
than a surface of its own.

## Exit codes

Seven members, defined in full in §12 and mapped here onto the outcome triple §6 states:

| code | outcome |
| --- | --- |
| `0` | `completed` |
| `1` | `failed` |
| `2` | none — no Run began |
| `75` | `failed` |
| `77` | `refused` |
| `130` | `failed` |
| `143` | `failed` |

**No exit code ever spans two outcomes.** The code space is finer than the triple — `1`, `75`, `130`
and `143` are all `failed` — and it is never coarser, which is the whole point: a caller that reads
`refused` as success has been told the wrong thing about what reached the world.

A caller that retries on `75` is right to, and one that retries on `77` loops forever: with no bypass
anywhere (ADR-0001) a verbatim retry refuses identically, and the way past is an artefact edit. A shell
script has the same reflex as an agent, and now the same signal.

The mapping is what keeps the two credential failures apart. Presence is checked where §6 resolves
every bound Target's credentials, before the first Step: one a Target declaration names and the
environment does not hold is a Refusal and exits `77` (`credential-absent`, §12), while one that is
present and the endpoint rejects is the world resisting and exits `1`. Nothing about where the process
runs enters either decision (ADR-0007).

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
that refuses there leaves no trace in the repository and is found in the Actions log. The alternative
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
what a `result` row carried. Its shape names the same axis, count and remainder the CLI's marker does:

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
the terminal compensates with an exit code, which this surface does not have. The two full renderings
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
unknown tool, an argument violating a schema, a fault in the server — and a Refusal is an answer to a
well-formed call. Every usage error the CLI half states arrives here as one of those, which is what
this surface has in place of exit code `2`.

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
// → rows: [{ type: "provider", name, origin: "builtin" | "extension",
//            summary, operation_count, digest }]
```

```jsonc
provider(name)
// → rows: [{ type: "manifest", auth_scheme, capabilities_required: [ … ],
//            digest, schema_version }]                       // the header row, emitted first
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
//              deadline_seconds, concurrency_limit } }]
```

`source` is the Manifest's own lines and not a re-rendering of them: a Manifest is written in the
format the caller is expected to author Definitions in (§3), so returning it verbatim teaches that
format at the moment the caller needs it.

### The repository

```jsonc
targets()
// → rows: [{ type: "target", name, endpoint,
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
run({ procedure } | { definition, operation, target }, dry_run?, secret_sink?)
// args: procedure    — the Procedure form, carrying no target
//       definition, operation, target — the single-Operation form, target required
//       dry_run      — boolean
//       secret_sink  — the Secret sink: an absolute path, outside the repository working tree
// → outcome: §12's triple
// → rows: §8's step, asset, observation, refusal, remediation and provenance rows, unchanged
```

**There is no `inputs` argument, on either form.** A Procedure is fully bound by its artefact, and a
value supplied at call time is Step behaviour appearing on no reviewed line — authority arriving after
review, which is the shape ADR-0008 removed and the same shape as the `--force` that is absent
everywhere else.

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
//            records: [ … ],                     // the Record identities the Step acted on
//            pattern: { attempts, pages, polls } }]
//         §8's provenance row, unchanged
//         [{ type: "expansion", step, selector, expanded_to, bound }]   // expansion: true only
```

```jsonc
changes(procedure?, since?, between?, target?, record_kind?, limit?)
// → rows: §8's window, asset, observation and code rows, unchanged
```

`record_kind` is the CLI's `--kind`, spelled out: in a flat argument object beside tools carrying an
Operation's Kind, one name cannot hold two senses.

```jsonc
records(target?, definition?, name?, history?, limit?)
// → rows: [{ type: "record", key: { target, definition, name }, version,
//            record_kind: "observation" | "asset", tombstoned, orphaned,
//            secret_fields: [ "api_key" ],   // the presence-only marker, per §7
//            provenance: { … } }]
```

### Lifecycle

```jsonc
project()
// → rows: [{ type: "workflow", path, procedure, cadence }]   // one per Procedure, all of them
```

`project` is not a Run and carries no `outcome` key, and it takes no arguments at all: it is repo-wide
and all-or-nothing above, and there is nothing here for a per-Procedure argument to name.

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
