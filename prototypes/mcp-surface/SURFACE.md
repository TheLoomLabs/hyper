# PROTOTYPE — `hyper`'s MCP surface

> **Throwaway.** This is the prototype for issue #10, kept as a primary source on branch
> `prototype/mcp-surface`. It is not the spec and nothing here is production code. The validated
> decisions live in the issue's resolution comment; this file is the artefact those decisions were
> made against.

Run the stub: `python3 prototypes/mcp-surface/server.py --walkthrough` (no dependencies), or point an
MCP client at `python3 prototypes/mcp-surface/server.py` for stdio.

Vocabulary is `CONTEXT.md`. Every capitalised term is a glossary term.

---

## 0. What binds this surface

Settled elsewhere, inherited here rather than re-decided:

| Source | Constraint |
| --- | --- |
| #3 | Nothing is invoked except through a **Definition**; every invocation binds a **Target**. Records are keyed `(Target, Definition, name)`. |
| #4 | No per-run approval, no bypass, no `--force`. Three outcomes: `completed` / `refused` / `failed`, never sharing a signal. **The environment is not an axis.** |
| #5 | The artefact is the review surface. One renderer, one row per table row. A **Refusal** is deliberately the most verbose surface in the tool. |
| #6 | No resumption, no daemon. Records write per Step. A killed Run leaves an **open Journal entry** closed `failed` by the next invocation. |
| #7 | A Provider *is* a **Manifest** — data, not code. Manifests are the same line-oriented format as Definitions. |
| #8 | **Cadence** is projected into Actions. The **Store** is a branch; an effectful Run syncs first or Refuses. |
| #9 | `hyper` never stores a secret. The **Secret sink** is a path, not an fd — *because of this surface*. |

## 1. Shape of the surface

- **Tools only.** No MCP resources, no prompts. Resources are client-controlled and in most clients
  are user-attached rather than agent-reachable, which would make Manifest discovery unavailable
  exactly when the agent needs it. Prompts are per-editor glue — the thing this project departs from.
- **`hyper` writes what it derives; the agent writes what is reviewed.** There is no tool that
  authors a Definition, a Procedure, or a Target declaration. The agent writes those with its own
  file tools. `hyper` supplies discovery good enough to write them correctly, and a `check` whose
  errors are positioned by file and line so the next act is an edit — the same act a human makes.
- **Shared core, independently designed surfaces.** The MCP surface and the CLI (#13) share the verb
  set and the outcome contract, normatively. Ergonomics may differ: the CLI adds flags, the MCP
  surface splits flag-laden commands into typed tools.
- **stdio.** One server process per client, dying with it. There is no daemon and no `serve`.

## 2. The return envelope

Every tool returns the same three-part shape.

```jsonc
{
  "content": [{ "type": "text", "text": "<see below>" }],
  "structuredContent": {
    "ok": true,
    "outcome": "completed",        // execution tools only; absent elsewhere
    "rows": [ { "type": "asset", "...": "..." } ],
    "truncated": null              // see §6
  },
  "isError": false
}
```

**`rows` is #5's renderer, unchanged.** #5 chose NDJSON with one JSON object per terminal table row
and a `type` discriminator. This surface serves that same row set as an array. There is still exactly
one renderer, so the terminal and the MCP surface cannot drift apart.

**The `text` block is asymmetric, on purpose.**

| Case | `text` carries |
| --- | --- |
| Ordinary return | One summary line, **outcome first**. |
| `review` | The full rendered review surface — gutter, `AUTHORITY`, `FLAGS`. |
| **Refusal** | The full rendered Refusal — caret excerpt, `EDIT ONE OF`, and the retry sentence (§4). |

The one-line-outcome-first rule repairs a cost #5 knowingly accepted: with NDJSON the outcome is only
known at the last row, and the terminal compensates with an exit code. MCP has no exit code, so the
outcome moves to the front of the text block.

The verbosity rule is the same trade #5 made and for the same reason — with no bypass, the Refusal
rendering *is* the entire remediation path.

## 3. The tools

Twelve. Discovery is three tools rather than one overloaded `provider(name?, operation?)`, because a
return type that changes shape depending on which arguments were omitted is the thing `outputSchema`
exists to prevent.

### Discovery

Three levels, because they are the agent's three questions in order: *which Provider*, *which
Operation*, *how do I call it*.

#### `providers()`

```jsonc
// → rows: [{ type: "provider", name, origin: "builtin"|"extension",
//            summary, operation_count, digest }]
```

#### `provider(name)`

```jsonc
// → rows: [{ type: "operation", name, kind: "read"|"mutate"|"destroy",
//            opaque: bool, summary }]
// → meta: { auth_scheme, capabilities_required: [...], digest, schema_version }
```

**`kind` is on every row here, at level 2.** That is deliberate: Kind is what answers #4's two-key
check — *can a Definition even declare this against my intended Target?* — and it must be answerable
before a single schema is fetched.

#### `operation(provider, name)`

Returns **the Manifest source text for this Operation**, plus a small projection of the facts that
are not in the source verbatim.

```jsonc
// → rows: [{ type: "operation_detail",
//            source: "<the Manifest lines declaring this Operation>",
//            derived: {
//              capabilities: [...],      // derived, not declared — #7 requires exact equality
//              bound_required: bool,     // true iff kind == "destroy"
//              patterns_resolved: [...], // host-owned; may not change Kind or count
//              record_cardinality, record_identity,
//              repeatability: "repeatable"|"skip-if-recorded"|"run-once",
//              deadline_seconds, concurrency_limit
//            } }]
```

Source, because #7 made a Manifest the same line-oriented format as a Definition — so returning it
teaches the agent the format it is expected to author in, which is the burden §1 put on `hyper`.
Projection alongside, because making the agent re-derive what `hyper` already computed is waste.

`repeatability` and `bound_required` live at level 3 rather than level 2 because they are
Definition-*authoring* facts, not call-time facts.

### Repo state

#### `targets()`

```jsonc
// → rows: [{ type: "target", name, endpoint,
//            accepts_kinds: [...], grants_capabilities: [...],
//            credential_env: ["PROD_TOKEN"],   // NAMES ONLY, never values
//            credentials_present: bool }]
```

Not repo-reading in disguise. Three of these fields are computed rather than read: the two-key
intersection against a Definition, #9's credential **presence** check, and the Capability grant set.
And `credential_env` is exactly what an agent must write into a Target declaration while never seeing
a value — #9 made a literal in a credential position a load-time error, so the agent needs the name
and must not have the secret.

### Author support

#### `check(paths?)`

The static oracle. Runs offline, with no credentials: declared-equals-derived Capabilities,
cardinality, identity, unknown-key rejection, enumeration closure, Target class, envelope
containment, the two-key intersection, Cadence projection drift.

```jsonc
// → rows: [{ type: "problem", severity: "error"|"warning",
//            file, line, column, code, message, artefact }]
```

Positioned by file and line, because §1 says the agent's next act is an edit.

#### `review(artefact)`

#5's definition-review surface. Separate from `check` because they answer different questions at
different moments — `check` returns pass/fail, `review` returns a rendering — and merging them would
make every check call pay for a render.

```jsonc
// text: the full rendered surface
// → rows: [{ type: "gutter", line, marker, source }]
//         [{ type: "authority", declared_kinds, target_accepts, intersection }]
//         [{ type: "flag", code, cites_line, text }]
```

**Every `flag` row carries `cites_line`, and it is not optional.** #5 allowed exactly one surface to
editorialise and constrained it so that it cannot introduce a claim the gutter has not already made.
A flag with no line to cite is a bug, not a summary.

### Execution

#### `run(...)`

```jsonc
// args: { procedure: string } | { definition: string, operation: string },
//       target: string,
//       dry_run?: bool,
//       secret_sink?: string    // absolute path, outside the working tree
// → outcome: "completed" | "refused" | "failed"
// → rows: [{ type: "step", index, disposition, definition, operation, target, kind }]
//         [{ type: "asset", ... }] [{ type: "observation", ... }]
//         [{ type: "refusal", check, step, target, declared, observed }]
//         [{ type: "edit_option", file, line, field, from, to, effect }]
```

**There is no `inputs` argument.** A Procedure is fully bound by its artefact. Three settled decisions
converge here and none of them were about inputs: #8 requires a Cadence-carrying Procedure to have no
unbound inputs; #5 makes the artefact the review surface, and a value supplied at call time is Step
behaviour appearing on no reviewed line; and `probe` now owns the ad-hoc case. An input is authority
arriving after review — the same shape as the `--force` that ADR-0001 removed. This is handed to the
authoring-language fog as a hard constraint, the way #5 handed it *line-oriented text*.

`secret_sink` is chosen by the **caller**, not by `hyper`. #9 designed the sink so that omitting it
produces a Refusal; a `hyper` that supplies one by default deletes that guardrail, and it makes
`hyper` a place where secrets live. `hyper` still enforces `0600` and refuses a path inside the
repository working tree, whoever named it.

Long Runs are handled in §5.

#### `probe(provider, operation, inputs)`

The answer to #3's one-off friction — restricted hard enough to be safe by construction:

- **`read` Kind only.**
- **`local` Target only** — the reserved Target, which holds no credentials and grants nothing.
- **Writes no Record.**
- **Writes no Journal entry.**

> **A probe is not a Run.** No Trigger, no Provenance, no Disposition, no outcome triple. It is a
> lookup, not an execution.

That framing is what makes it a door rather than a loophole. With no Record there is no
`(Target, Definition, name)` key to need, so the reason Definitions are mandatory does not apply; and
against `local` the two-key check is **vacuous rather than skipped**, since there is nothing to
protect. It follows that a probe cannot carry a Cadence, cannot appear in a Procedure, and can never
be a diff baseline — weaker even than #6's dry-run, which at least writes a marked entry.

The argument for it is not agent convenience. **#5's review model dies by volume**: if every
throwaway question becomes a reviewed artefact, the set of Definitions stops being something a human
reads, and the oversight story goes with it. The probe protects the review surface.

Honest residual, of the map's own three trivial-read examples:

| Example | Target | Smoothed? |
| --- | --- | --- |
| Is this site up? | `local` | yes |
| When does this cert expire? | `local` | yes |
| What are my VM IPs? | `proxmox-prod` | **no — needs a Definition** |

The third is correct as it stands. It needs a credential, and that is precisely where authority
review earns its keep.

### Inspection

#### `runs(since?, procedure?, target?, outcome?, limit?)`

```jsonc
// → rows: [{ type: "run", id, started, trigger, outcome, procedure, targets, hyper_version }]
```

`trigger` is on every row because #8 made it the only thing separating *nothing changed* from
*nothing ran*.

#### `run_show(id)`

```jsonc
// → rows: [{ type: "disposition", step, state: "ran"|"skipped-recorded"|"refused"
//                                         |"never-reached"|"attempted-outcome-unknown" }]
//         [{ type: "provenance", definition_revision, manifest_digest,
//            extension_digest, repo_revision, hyper_version }]
//         [{ type: "expansion", step, selector, expanded_to, bound }]
```

#### `diff(since | between, target?, record_kind?, limit?)`

#5's three tables, one per actor.

```jsonc
// → rows: [{ type: "asset",       change: "new"|"mut"|"tomb"|"moved", ... }]  // YOU DID THIS
//         [{ type: "observation", change: ..., ... }]                        // THE WORLD MOVED
//         [{ type: "code",        definition_revision_from, ..., summary }]  // THE CODE MOVED
```

`code` rows make provenance drift a first-class diff event — #5's decision 4. An AI widening a
destroy Bound between two Runs is the same *kind* of fact as a server disappearing, and usually the
more important one.

#### `records(target?, definition?, name?, history?, limit?)`

```jsonc
// → rows: [{ type: "record", key: {target, definition, name}, version,
//            record_kind: "observation"|"asset", tombstoned: bool,
//            secret_fields: ["api_key"],   // presence-only marker, per #9
//            provenance: {...} }]
```

## 4. Outcomes and errors

| Outcome | `isError` | `structuredContent.outcome` |
| --- | --- | --- |
| completed | `false` | `"completed"` |
| refused | **`true`** | `"refused"` |
| failed | **`true`** | `"failed"` |

`isError` is one bit and cannot carry three states, so it was never the discriminator — **`outcome`
is**, checkable without parsing prose, exactly as #4 required. `isError` means only *you did not get
what you asked for*, which is true of both. Burying a Refusal inside a successful-looking return
would undo the work #5 spent making it unskimmable.

JSON-RPC protocol errors are reserved for **malformed calls only** — unknown tool, schema violation,
server fault. A domain outcome is never a protocol error.

### The retry sentence

Every Refusal's `text` block ends with an explicit statement that **a verbatim retry will refuse
identically**, naming the artefacts to edit.

This is load-bearing, not politeness. `isError: true` conventionally invites a retry; with no bypass,
a retrying agent loops forever. The sentence is ADR-0001's MCP analogue — the protocol has no way to
express *this will never work until a human-reviewable file changes*, so the rendering must.

## 5. Long Runs

**Synchronous, with `notifications/progress` at Step boundaries.** #6's per-Step Record writes give
the progress ticks a natural boundary.

An async handle — `run` returning an id the agent polls — was rejected: it invents a Run that
outlives its caller with nothing watching it, which is a daemon with extra steps, and #6 and #8 killed
daemons independently.

A client timeout needs no new machinery, because **#6 already specified it**: the client gives up, the
stdio server dies, the open Journal entry is closed `failed` by the next invocation with the in-flight
Step recorded *attempted, outcome unknown*. That is the truthful answer to *what happened*, and it
falls out of decisions already made.

**Accepted cost, named rather than discovered later: a twenty-minute provision is not practically
runnable from this surface.** That is not a gap — it is #8 saying so. MCP owns the
author→validate→observe loop and short effectful Runs; long unattended work is a Cadence on Actions,
where there is no interactive client to time out.

## 6. Truncation

Inspect tools take `limit` with a modest default and return a truncation marker. **There is no
cursor.**

```jsonc
"truncated": {
  "axis": "time",
  "returned": 200,
  "dropped": 2840,
  "hint": "narrow with `since` or `target`; 2840 rows fall outside the returned window"
}
```

Unbounded returns blow a context window on the first interesting month. Pagination is the same
disaster arriving politely — an agent walking three thousand rows a page at a time. Truncation is
therefore a signal that the *question* was too broad, and the marker names the axis so the next call
is a narrower query rather than a page two.

This mirrors #5's rule that a surface may only index into facts already present: **a truncated result
must never look complete.**

## 7. What is deliberately absent

### `install` — excluded from this surface

#7's `install` fetches a third-party extension and writes the tracked, digest-verified file. It is
reachable from the CLI and **not from MCP**.

#7's claim — that a hostile extension reaches nothing you did not grant — survives an agent
installing one. What does not survive is the *review moment*: `install` is the single point where new
third-party data enters the repository, and #7 made it a tracked file precisely so it appears in a
diff a human reads. An agent that can install can also, in the same turn, author against what it
installed and run it — the entire supply-chain sequence with no human between acquisition and effect.
Excluding one tool restores that gap at negligible cost, since installing is rare and manual by
nature.

Requiring a digest argument instead was considered and rejected as theatre: the agent would read the
digest from the registry it was already trusting.

**The line is acquisition, not derivation.** #8's Cadence projection stays reachable — a Cadence
declared in a reviewed artefact and not projected is a loud drift failure, and the agent must be able
to repair what it caused.

### Everything else absent

- No authoring tools (§1).
- No approval or confirmation tool — #4 declined per-run approval, and adding one here would make the
  caller an authority axis.
- No `--force`, no `--skip-checks`, no override argument anywhere (ADR-0001).
- No resources, no prompts (§1).
- No `serve`, no remote transport — out of scope on the map.
