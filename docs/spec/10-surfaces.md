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
