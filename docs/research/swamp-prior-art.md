# Swamp prior art: data model, execution semantics, and the decisions it made

Research for [hyper#2](https://github.com/TheLoomLabs/hyper/issues/2). Investigated 2026-08-06.

## Sources and method

Primary sources, in descending order of trust:

1. **The source tree** — `github.com/swamp-club/swamp` at commit `5bfddaa`, cloned and read
   directly. TypeScript/Deno, ~1,667 `.ts` files under `src/`.
2. **`design/*.md` in that repo** — 27 internal design documents, 13,529 lines. These are the
   maintainers' own architecture notes, including superseded ones. Far more candid than the
   README or the manual.
3. **The manual** at `swamp-club.com/manual` — all 146 pages listed in `sitemap.xml` were fetched
   and read (§13). It is not in the public repo. It is mechanism-heavy and mostly reliable, with the
   specific exceptions catalogued in §13.7.
4. **The README** — *treat as unreliable*. It describes at least one feature that the design docs
   say was removed (§8.2).

**Trust order established by this research: `src/` > `design/` > the manual > bundled skills >
README > marketing pages.**

Every claim below cites a path relative to the Swamp repo root. Anything not directly read is
marked **[inferred]**. Where a question could not be answered, it says so.

---

## Contents

If you read only three sections, read **§9a.2** (implicit ordering), **§9a.7** (approval gates and
the CI exit code), and **§11** (synthesis: what to copy, what not to).

| § | Section |
| --- | --- |
| 1 | What Swamp actually is — thesis, agent interface, storage split |
| 2 | The primitives, corrected — Model / Definition / Workflow / Vault / Data schemas and addressing |
| 3a | Repo config: `.swamp.yaml` |
| 3b | Expressions: CEL, three surfaces, namespaces, the live-state hazard |
| 4 | Inputs — JSON Schema, `forEach`, the `--input` grammar |
| 5 | Execution drivers — an abstraction built, then deleted; vault sentinels |
| 6 | Sensitive data and the vault pre-flight |
| 7 | Run tracking — the SQLite crash-recovery bolt-on |
| 8 | Scar tissue |
| 9 | Supply chain — correcting the map |
| 9a | **Execution semantics** — DAG, concurrency, `forEach`, failure, resume, approvals |
| 9b | **The data layer** — storage, addressing, immutability, query, redaction, provenance, diff |
| 9c | **Vaults** — encryption, keys, refresh hooks, access control, audit |
| 10 | **The extension model** — loading, contract, sandboxing, `npm:`, integrity |
| 11 | **Synthesis** — deliberate decisions, scar tissue, what not to copy |
| 12 | Contradictions with the map's Notes |
| 13 | The manual and the marketing surface |

---

## 1. What Swamp actually is

> "Swamp is an AI Native Automation tool. It has 1:1 models of external APIs or CLI tools (for
> example, AWS or Azure cloud resources, or the GitHub CLI), which it can then validate are
> correct."
> — `design/high-level.md:5-7`

The README's one-liner is "Deterministic Automation for AI Agents" (`README.md:7`). The thesis is
that an AI agent should not run `aws ec2 ...` ad hoc; it should *author a reviewable artifact*
(a YAML definition, a YAML workflow) that a human can read before it touches production, and that
produces versioned data a human can diff afterwards.

That thesis is sound and is the part `hyper` is inspired by. Almost everything else in the repo is
accretion around it.

### The interface is skills + CLI, not MCP

**Swamp ships no MCP server.** Confirmed by exhaustive grep: the only `modelcontextprotocol`
references in the entire tree are transitive npm dependencies of `promptfoo` inside
`evals/promptfoo/package-lock.json`. There is no MCP server, client, or tool definition in `src/`.

Instead, `swamp repo init` writes *skill files* into the agent's own convention directory and
expects the agent to shell out to the CLI:

| Tool | Init flag | Skills dir | Instructions file |
| --- | --- | --- | --- |
| Claude Code | *(default)* | `.claude/skills/` | `CLAUDE.md` |
| Cursor | `--tool cursor` | `.cursor/skills/` | `.cursor/rules/swamp.mdc` |
| OpenCode | `--tool opencode` | `.agents/skills/` | `AGENTS.md` |
| Codex | `--tool codex` | `.agents/skills/` | `AGENTS.md` |

— `README.md:114-119`; `design/repo.md:19-32` adds Copilot and Kiro, so the real list is six tools.

`design/agent.md:28-32` states the intended contract plainly: *"Agents should prefer using CLI
commands for operations, and use the top-level directories for exploration and understanding
context."*

This is a real divergence from hyper's settled "MCP-first" decision (map #1, Notes). Swamp bet on
"teach the agent to use my CLI via a skill file"; it then paid for that bet six times over, once
per supported agent tool, plus per-tool hook-config generators (`design/audit.md:41-58`).

### Storage: git for source-of-truth, a "datastore" for everything else

```
models/{normalized-type}/{id}.yaml        # git-tracked, source of truth
workflows/workflow-{id}.yaml              # git-tracked
vaults/{vault-type}/{id}.yaml             # git-tracked
grants/{name}.yaml                        # git-tracked, access grants

.swamp/data/{normalized-type}/{model-id}/{data-name}/{version}/
.swamp/outputs/{normalized-type}/{method}/{definition-id}-{timestamp}.yaml
.swamp/workflow-runs/{workflow-id}/workflow-run-{run-id}.yaml
```

— `design/repo.md:72-88`, `design/repo.md:189-216`

The split is deliberate: *authored artifacts* are plain files in git (reviewable, diffable by
existing tools); *runtime data* goes through a pluggable datastore abstraction (local FS, external
FS path, or S3 via an extension). `README.md:192-277` documents the datastore backends.

Crucially, the maintainers now think this split was a mistake. See §8.

---

## 2. The primitives — corrected

### 2.1 There are not five, and "Definition" is not a peer of "Model"

The README's "Core Concepts" lists Models, Definitions, Workflows, Data, Vaults, Tags. Swamp's own
bundled skill (`.claude/skills/swamp/SKILL.md:18-28`) lists a *different* set: **Models, Data,
Workflows, Vaults, Extensions, Grants, Serve**. Neither is stable, and the codebase carries several
more first-class kinds — Reports (`design/reports.md`), Datastores, Workers, Grants, Skills.

More importantly, **Definition is not a sibling of Model — it is an instance of one.** The actual
spine:

```
ModelDefinition            a TYPE.     Authored in TypeScript. Code + Zod schemas.
   ▲  referenced by `type` + `typeVersion`
Definition                 an INSTANCE. Authored in YAML. Values only.
   ▲  referenced by name or UUID from StepTask.modelIdOrName
Workflow → Job → Step → StepTask       Authored in YAML. Orchestration.
   │
   ▼ produces
Data                       Immutable, versioned. Owned by a Definition.

VaultConfig                YAML. A named secret backend, referenced by name from CEL.
```

This matters for hyper because the map treats the five as "the starting decomposition". The real
decomposition is three layers (type / instance / orchestration) plus two stores (data / secrets) —
and the type layer is *code*, not configuration.

### 2.2 Model (a model *type*)

TypeScript only. There is no YAML for a model type. Registered by `defineModel()` at module import
time (`src/domain/models/model.ts:1300-1309`).

Schema, `src/domain/models/model.ts:798-887`:

| Field | Type | Req | Meaning |
| --- | --- | --- | --- |
| `type` | `ModelType` | yes | value object; `raw` + `normalized` |
| `version` | `string`, CalVer `YYYY.MM.DD.MICRO` | yes | validated at `model.ts:1089-1094` |
| `globalArguments` | `z.ZodTypeAny` | no | schema for the Definition's `globalArguments` |
| `resources` | `Record<string, ResourceOutputSpec>` | no | keys are **spec names** |
| `files` | `Record<string, FileOutputSpec>` | no | keys are spec names |
| `methods` | `Record<string, MethodDefinition>` | yes | keys are method names |
| `checks` | `Record<string, CheckDefinition>` | no | pre-flight guards |
| `reports` | `string[]` | no | default report names |
| `upgrades` | `VersionUpgrade[]` | no | ordered; last `toVersion` must equal `version` (`model.ts:1122-1129`) |
| `bundleSourceFactory` | `() => Promise<string>` | no | lazy bundle for remote execution |
| `extensionName` | `string` | no | e.g. `@keeb/network`; drives dependency authorization |
| `sourceFingerprint` | `string` | no | SHA-256 of the bundle; recorded in `ExecutionProvenance` |

Sub-schemas worth naming:

- `ResourceOutputSpec` (`model.ts:50-71`): `schema: z.ZodTypeAny` (required), `lifetime`,
  `garbageCollection`, `tags?`, `sensitiveOutput?`, `vaultName?`.
- `MethodDefinition` (`model.ts:732-770`): `description`, `kind?`, `rollbackOnFailure?`,
  `arguments` (Zod, required), `execute(args, context)`.
- **`MethodKind`** (`model.ts:661-667`): `"create" | "read" | "update" | "delete" | "list" |
  "action"`, **inferred from the method name** by `inferMethodKind` (`model.ts:676-700`), with
  `isMutatingKind` = everything except `read`/`list` (`model.ts:725-727`).

That last one is directly relevant to hyper's safety ticket: Swamp has a machine-readable
read-vs-mutate classification on every method, and it is *inferred from the name* by default. A
model author who names a destructive method `check_and_purge` gets whatever the inference heuristic
decides. Explicit declaration exists (`kind:`) but is optional.

`ModelType` normalization (`src/domain/models/model_type.ts:78-86`): lowercase, `::`→`/`,
whitespace→`/`, `.`→`/`, collapse `//`, strip leading/trailing `/`. So `AWS::EC2::VPC` →
`aws/ec2/vpc`. Collectives `swamp` and `si` are reserved (`model_type.ts:161`).

A complete real model type, `src/domain/models/worker/fleet_probe_model.ts:125-146`:

```typescript
export const fleetProbeModel: ModelDefinition = defineModel({
  type: FLEET_PROBE_MODEL_TYPE,
  version: "2026.07.04.1",
  resources: {
    "result": {
      description:
        "Fleet probe verification result (per-seam pass/fail with platform info)",
      schema: ProbeResultSchema,
      lifetime: "ephemeral",
      garbageCollection: 1,
    },
  },
  methods: {
    verify: {
      description:
        "Exercise every seam between worker and orchestrator: dispatch metadata, capability RPC, and data plane",
      kind: "action",
      arguments: VerifyArgsSchema,
      execute: verify,
    },
  },
});
```

### 2.3 Definition (a model *instance*)

YAML at `models/{normalized-type}/{id}.yaml`
(`src/infrastructure/persistence/yaml_definition_repository.ts:56,71,440-445`).

Schema, `src/domain/definitions/definition.ts:155-186`:

| Field | Type | Req | Meaning |
| --- | --- | --- | --- |
| `type` | `string` | always written | normalized model type; falls back to directory path on load (`:229-230`) |
| `typeVersion` | `string` CalVer | no | numeric legacy values coerced to `undefined` = "needs full upgrade" (`definition.ts:157-160`) |
| `id` | uuid | yes | filename stem |
| `name` | `string` min 1 | yes | globally unique handle; no `..`, `\`, NUL; `/` only in `@collective/name` |
| `version` | positive int | yes | definition revision |
| `tags` | `Record<string,string>` | default `{}` | |
| `globalArguments` | `Record<string,unknown>` | default `{}` | validated against the type's Zod schema; may contain `${{ }}` CEL |
| `methods` | `Record<string, {arguments?}>` | default `{}` | keys must be method names on the type |
| `inputs` | `InputsSchema` (JSON Schema draft-07 subset) | no | `definition.ts:93-113` |
| `checks` | `{require?: string[], skip?: string[]}` | no | |
| `reports` | `{require?, skip?}` | no | |
| `resources` | `Record<specName, {lifetime?, garbageCollection?, vaultName?}>` | no | per-spec policy override |

`driver` / `driverConfig` are **rejected** with an actionable error
(`definition.ts:192-195` → `src/domain/removed_driver_fields.ts`) — the deleted driver abstraction
again.

`computeHash()` (`definition.ts:486-506`) SHA-256s the definition with `type`/`typeVersion`
**excluded** and keys recursively sorted, producing the instantiation id / `definitionHash`.

**Auto-definitions.** A model type is usable with no hand-written Definition, via "direct type
execution" (`modelType` + `modelName`). Swamp then auto-creates a Definition into
`.swamp/auto-definitions/` — not git-tracked — purely *so the produced data has an ownership home*
(`design/models.md:149-179`). That is the stated reason they exist. It is a good tell: in Swamp's
model, data cannot be orphaned; something must own it.

### 2.4 Workflow

YAML at `workflows/workflow-{uuid}.yaml`. Aggregate root: `Workflow → Job[] → Step[] → StepTask`.

**Workflow** (`src/domain/workflows/workflow.ts:36-76`): `id` (uuid), `name`, `description?`,
`trigger?` (`{schedule?: croner-validated cron, inputs?}`), `tags`, `inputs?` (`InputsSchema`),
`jobs` (min 1), `version`, `concurrency?` (caps parallel jobs), `reports?`.

Unknown top-level keys are **rejected, not stripped**, with a Levenshtein "did you mean"
(`workflow.ts:85-91`, rationale at `design/workflow.md:364-389`). Good decision for AI-authored
YAML — a hallucinated key fails loudly instead of being silently dropped.

**Job** (`src/domain/workflows/job.ts:64-71`): `name`, `description?`, `steps` (min 1),
`dependsOn: {job, condition}[]`, `weight` (default 0), `concurrency?`.

**Step** (`src/domain/workflows/step.ts:79-97`):

| Field | Type | Meaning |
| --- | --- | --- |
| `name` | string min 1 | unique within job; may contain `${{self.*}}` for forEach |
| `task` | `StepTask` | discriminated union, below |
| `forEach` | `{item: string, in: string}` | `item` = loop var name, `in` = CEL expression |
| `dependsOn` | `{step, condition}[]` | bare-string entries throw an instructive error (`step.ts:58-77`) |
| `weight` | number, default 0 | deterministic topo-sort tiebreak |
| `concurrency` | non-negative int | caps forEach fan-out |
| `dataOutputOverrides` | `DataOutputOverride[]` | including `vary` dimensions |
| `allowFailure` | boolean, default false | |
| `target` / `labels` / `platform` / `queueTimeout` | | remote-worker placement |
| `guard` | string (CEL) | truthy ⇒ **skip** the step (idempotence) |

**`StepTask`** — a four-variant discriminated union (`src/domain/workflows/step_task.ts:41-67`):

- **`model_method`** — `modelIdOrName?` XOR (`modelType?` + `modelName?`), `methodName`, `inputs?`,
  `globalArgs?`. `inputs`/`globalArgs` accept either a record or a single `${{...}}` expression
  (pattern `/^\$\{\{\s*.+?\s*\}\}\s*$/s`). `type: shell` throws with a migration message
  (`:77-89`).
- **`workflow`** — `workflowIdOrName`, `inputs?`. This is how nesting works.
- **`manual_approval`** — `prompt`, `timeout?` (seconds).
- **`assert`** — `expr` (CEL), `message`, `severity: "low"|"medium"|"high"` default `"high"`.

**`TriggerCondition`** is a recursive union (`src/domain/workflows/trigger_condition.ts:69-77`):
`always | succeeded | failed | completed | skipped | and{≥2} | or{≥2} | not{}`. The *reference*
(which step or job) lives on the parent `dependsOn` entry, not inside the condition
(`trigger_condition.ts:61-62`). `completed` = succeeded ∨ failed (`:212-215`).

That separation — reference on the edge, predicate in the condition — is clean and worth copying.

A minimal valid workflow, from `src/libswamp/workflows/doctor_test.ts:25-35`:

```yaml
id: "550e8400-e29b-41d4-a716-446655440000"
name: test-workflow
jobs:
  - name: test-job
    steps:
      - name: test-step
        task:
          type: model_method
          modelIdOrName: my-model
          methodName: validate
```

`forEach` with `vary` dimensions, `design/workflow.md:750-765`:

```yaml
steps:
  - name: scan-${{ self.env }}
    forEach:
      item: env
      in: ${{ inputs.environments }}
    task:
      type: model_method
      modelIdOrName: scanner
      methodName: execute
      inputs:
        environment: ${{ self.env }}
    dataOutputOverrides:
      - specName: result
        vary:
          - environment
```

**`WorkflowRun`** (`src/domain/workflows/workflow_run.ts:107-135`), persisted to
`workflow-runs/{workflow-id}/workflow-run-{run-id}.yaml`: `id`, `workflowId`, `workflowName`,
`status` (`pending | running | suspended | succeeded | failed | cancelled`), `startedAt?`,
`completedAt?`, `jobs: JobRun[]`, `workflowDataArtifacts?`, `logFile?`, `pid?`, `tags`, `inputs?`,
`resumeInputs?: string[]`, `initiatedBy?`, `instanceId?`.

Note `resumeInputs` stores **key names only, never values** (`design/workflow.md:169-171`) — a
deliberate choice to keep secrets out of the run record.

### 2.5 Vault

YAML at `vaults/{vault-type}/{id}.yaml`. Schema, `src/domain/vaults/vault_config.ts:37-56`:

| Field | Type | Req | Meaning |
| --- | --- | --- | --- |
| `id` | string | yes | filename stem — **not** constrained to a UUID, unlike Definition/Workflow |
| `name` | string | yes | the name used in `vault.get("<name>", ...)` |
| `type` | string | yes | `local_encryption` (built-in) or an extension type |
| `config` | `Record<string,unknown>` | default `{}` | provider-specific |
| `createdAt` | ISO string | yes | |
| `auditReads` | boolean | no | when true, every `get()` appends to `.swamp/audit/vault-reads-YYYY-MM-DD.jsonl` |

Only one built-in type: `local_encryption` (`src/domain/vaults/vault_types.ts:44-52`). Everything
else — AWS Secrets Manager, Azure Key Vault, 1Password — is an extension, and the legacy short names
are renamed via a compat map (`vault_types.ts:32-38`).

`VaultProvider` (`src/domain/vaults/vault_provider.ts:31-67`) is a four-method interface —
`get`, `put`, `list`, `getName` — with optional capabilities detected by **runtime type guard**
rather than declaration: `VaultDeleteProvider`, `VaultAnnotationProvider`,
`VaultRefreshHookProvider`.

Secret bytes live outside the config, at
`.swamp/secrets/{vault-type}/{vault-name}/{secret-key}.enc` with `.annotations/` and `.refresh/`
siblings (`design/vaults.md:26-36,168-175`). `secrets` is in `ALWAYS_LOCAL_SUBDIRS`
(`src/domain/datastore/datastore_config.ts:36-42`) so it **never syncs to a remote datastore**.

Vault selection for a sensitive field is a four-tier fallback (`design/vaults.md:488-497`):
field-level `.meta({vaultName})` → spec-level `ResourceOutputSpec.vaultName` (overridable per
definition) → repo-level `defaultVault:` → *first available vault*. That last fallback is
uncomfortable: with no explicit selection, a secret lands in whichever vault happens to enumerate
first.

### 2.6 Data

Never hand-authored. Produced by `context.writeResource()` / `context.createFileWriter()`.
Two kinds: **resource** (JSON validated against the spec's Zod schema) and **file** (bytes + MIME,
optionally line-streaming).

On-disk layout (`design/models.md:574-584`,
`src/infrastructure/persistence/unified_data_repository.ts:70-81`):

```
.swamp/data/{normalized-type}/{model-id}/{data-name}/
  1/
    raw              # Content (binary or text)
    metadata.yaml    # Metadata
  2/
    raw
    metadata.yaml
  latest             # Text file containing current version (e.g. "2")
```

`latest` is read as a text file, falling back to `Deno.readLinkSync` for the legacy symlink format
(`unified_data_repository.ts:1389-1400`) — more symlink scar tissue. Versions are auto-incrementing
integers from 1.

`DataMetadata` (`src/domain/data/data_metadata.ts:121-147`):

| Field | Type | Req | Meaning |
| --- | --- | --- | --- |
| `name` | string min 1 | yes | instance name; no `..`, `/`, `\`, NUL. `"latest"` is reserved |
| `id` | uuid | yes | branded `DataId` |
| `version` | positive int | yes | |
| `contentType` | string | yes | MIME |
| `lifetime` | `Lifetime` | yes | `/^\d+(mo\|y\|h\|m\|d\|w)$/` \| `ephemeral` \| `infinite` \| `job` \| `workflow` |
| `garbageCollection` | `number>0` \| duration string | yes | version retention |
| `streaming` | boolean | default false | |
| `tags` | `Record<string,string>` | yes | **must contain a `type` key**; auto-set to `resource`/`file` |
| `ownerDefinition` | `OwnerDefinition` | yes | provenance, below |
| `createdAt` | ISO datetime | yes | |
| `size` / `checksum` | int / string | no | |
| `lifecycle` | `"active"\|"deleted"` | no | omitted when active; `deleted` = tombstone |
| `renamedTo` | string | no | forward reference on a rename tombstone |

`OwnerDefinition` (`data_metadata.ts:103-113`) is the provenance record: `definitionHash?` (legacy),
`ownerType: "model-method" | "workflow-step" | "manual"`, `ownerRef`, `workflowId?`,
`workflowRunId?`, `workflowName?`, `jobName?`, `stepName?`, `source?`.

Ownership is enforced on `ownerType` + `ownerRef` only (`src/domain/data/data.ts:218-221`).
`ownerRef` is the Definition UUID for model-method writes (`src/domain/models/data_writer.ts:287`),
or `` `${workflowId}:${jobName}:${stepName}` `` for workflow-step writes
(`unified_data_repository.ts:2045`).

`DataRecord` (`src/domain/data/data_record.ts:29-60`) is the read/query projection: `id, name,
version, isLatest, createdAt, namespace, attributes, tags, modelName, modelId, modelType, specName,
dataType, contentType, lifetime, ownerType, streaming, size, content`, plus promoted provenance
fields `ownerRef, workflowRunId, workflowName, jobName, stepName, source` (empty string when not
workflow-produced).

### 2.7 Addressing, end to end

| Thing | Addressed by |
| --- | --- |
| Model type | normalized string, e.g. `aws/ec2/vpc`; `@collective/...` for extensions |
| Definition | UUID `id` (filename) **or** unique `name`; **or a ≥3-hex-char UUID prefix** with ambiguity detection, Docker-style (`src/domain/models/model_lookup.ts:26-35,76-95`) |
| Job | name, unique within workflow |
| Step | name, unique within job; `forEach` names are templates |
| Data | `{normalized-type}/{definition-id}/{data-name}/{version}` |
| Data (varied) | `data-name` becomes composite: `` `${baseName}-${varyValues.join("-")}` `` (`src/domain/data/composite_name.ts:66`), e.g. `result-dev`, `result-prod`, each with its own `latest` |
| Vault | plain `name` string from CEL |
| Vault secret key (auto-generated) | `{sanitized modelType}-{modelId}-{methodName}-{fieldPath}`, `@` stripped, `/`,`\` → `-` (`design/vaults.md:439-450`) |
| Namespace | branded string `/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/`, ≤64 chars; `SOLO_NAMESPACE = ""` (`src/domain/data/namespace.ts:20-46`) |

Two distinctions that are easy to miss and are load-bearing:

- **`specName` vs `data-name`.** `specName` names the *declaration* on the model type
  (`resources: { result: {...} }`). `data-name` is the *instance* name passed at write time:
  `writeResource(specName, name, ...)`. One spec produces many named instances.
- **A step referencing a nonexistent model instance is a warning**, because an upstream step may
  create it mid-run; an unresolvable model *type* is a hard failure
  (`design/workflow.md:337-347`).

### 2.8 Documentation reliability

The agent that mapped this found the bundled skill references (`.claude/skills/swamp/references/**`)
contradict `design/` and the code in at least three places: definitions shown at
`models/{name}/input.yaml` (actually `models/{normalized-type}/{id}.yaml`), definitions under
`.swamp/definitions/` (moved to top-level `models/`), and `latest` described as a symlink (now a
text file).

`design/high-level.md:20-23` self-flags applications, environments, and drift detection as
unimplemented. `design/vaults.md:645-702` self-flags its `swamp/lets-get-sensitive` vault model as
not present in the codebase.

**Trust order for anyone reading Swamp: `src/` > `design/` > the manual > bundled skills > README >
marketing pages.** (The manual ranks above the bundled skills because it is maintained as a
deliverable; see §13.)

---

## 3a. Repo config: `.swamp.yaml`

The full marker schema, verbatim from
`src/infrastructure/persistence/repo_marker_repository.ts:41-67`:

```typescript
export interface RepoMarkerData {
  swampVersion: string;
  initializedAt: string;
  upgradedAt?: string;
  modelsDir?: string;
  workflowsDir?: string;
  vaultsDir?: string;
  driversDir?: string;
  datastoresDir?: string;
  reportsDir?: string;
  repoId?: string;
  telemetryEndpoint?: string;
  telemetryDisabled?: boolean;
  telemetryKeepFlushed?: boolean;
  tools?: string[];
  /** @deprecated legacy single-tool field; promoted to `tools` on read. */
  tool?: AiTool;
  logLevel?: string;
  gitignoreManaged?: boolean;
  datastore?: DatastoreConfigData;
  trustedCollectives?: string[];
  trustMemberCollectives?: boolean;
  lastSkillMigrationWarning?: string;
  skillMigrationDismissed?: boolean;
  autoGc?: boolean;
  defaultVault?: string;
}
```

Note what this file has become: alongside genuine configuration it now carries UI-dismissal state
(`skillMigrationDismissed`, `lastSkillMigrationWarning`), a deprecated field kept alive by a read
normalizer (`tool`), and two removed fields that are *explicitly rejected with an error* rather
than ignored — `defaultDriver` and `defaultDriverConfig`
(`src/infrastructure/persistence/repo_marker_repository.ts:84-97`).

Precedence is per-setting and hand-rolled, not a general mechanism. For log level
(`README.md:322-323`):

```
-q / --log-level flag  →  SWAMP_LOG_LEVEL  →  .swamp.yaml logLevel  →  info
```

For repo directory (`README.md:292-293`): `--repo-dir` → `SWAMP_REPO_DIR` → cwd.
For telemetry (`README.md:480-482`): flag → env → repo yaml → user-global yaml.

Each setting re-implements its own chain. There is no single precedence resolver.

---

## 3b. Expressions: CEL with `${{ }}` delimiters

Swamp embeds Google CEL in YAML using `${{ ... }}`. Source: `design/expressions.md`.

There are **three separate CEL environments** (`design/expressions.md:10-43`):

1. **Internal** — `CelEvaluator` (`src/infrastructure/cel/cel_evaluator.ts`). Used for workflow
   conditions, data queries, `forEach` expansion, definition evaluation. Registers swamp's
   namespace types (`file.contents`, `data.latest`, …).
2. **Extension-author-facing** — `createExtensionCelEnvironment()`, exposed to extension methods as
   `ctx.createCelEnvironment()`. Same arithmetic baseline, no swamp-internal namespaces.
   Extensions register their own functions/types/operators; registrations are per-instance.
3. **Grant-condition** — `createGrantConditionEnvironment()`
   (`src/infrastructure/cel/grant_condition_environment.ts`). A **sealed** environment for
   authorization conditions:

   > "It declares explicit variables per resource kind (workflow, model, data, access) and a
   > `principal.*` namespace, with `unlistedVariablesAreDyn: false` so references to undeclared
   > fields fail at write-time validation. No I/O receivers (`data.*`, `file.*`, `vault.*`,
   > `env.*`), no extension registrations, no host functions beyond the arithmetic baseline. The
   > seal is permanent — conditions are deterministic pure functions over (resource fields,
   > principal context)."
   > — `design/expressions.md:29-38`

That third one is the single best design decision in the repo: *the security-decision expression
language is a strictly weaker, sealed dialect of the automation expression language, and the seal
is documented as permanent.* Worth copying wholesale in spirit.

CEL is `@marcbachmann/cel-js@7.6.1` from npm (`deno.json:42`).

### Namespaces available

| Namespace | Where available | Source |
| --- | --- | --- |
| `model.<name>.definition.attributes.*` | definitions, workflows | `design/expressions.md:66-73` |
| `self.*` | own definition — name, version, tags, attributes; also `forEach` item | `design/expressions.md:94-95` |
| `inputs.*` | definitions and workflows | `design/expressions.md:137-155` |
| `data.latest / version / listVersions / query / findByTag / findBySpec` | model globalArguments, workflow step inputs, `forEach.in` | `design/expressions.md:247-398` |
| `workers.connected()` | same contexts as `data.*` | `design/expressions.md:103-135` |
| `run.id / workflowId / workflowName / startedAt / tags` | step execution only | `design/expressions.md:168-183` |
| `webhook.body / headers / route` | **only** inside `trigger.inputs` | `design/expressions.md:185-208` |
| `vault.get(...)` | definitions, step inputs | `design/execution-drivers.md:299-303` |

Notable language details:

- No `??` operator. Optional payload fields must use `has(x.y) ? x.y : fallback`
  (`design/expressions.md:204-206`). A hard reference to a missing field aborts the run.
- Optional-select `.?` and `.orValue(default)` exist for possibly-absent data
  (`design/expressions.md:369-390`).
- `ns` is the namespace field name in queries because *"`namespace` is a reserved identifier in the
  CEL language"* (`design/expressions.md:354-355`).
- `model.*.resource` and `model.*.file` are **deprecated** (`design/expressions.md:244-245`).

### A hazard worth naming: expressions read live disk state

> "These accessors read directly from disk on every call, so they always reflect the latest on-disk
> state with no cache staleness."
> — `design/expressions.md:242-243`

This is framed as a feature. It means a workflow has **no snapshot isolation**: two steps in the
same run evaluating `data.latest('m','n')` can observe different values if anything wrote in
between, and re-running an "evaluated" workflow is not guaranteed to reproduce. Swamp partly
concedes this by adding a `--last-evaluated` flag that re-runs from the previously evaluated
expansion instead of re-evaluating (`design/inputs.md:186-189`).

---

## 4. Inputs

Inputs are JSON Schema, declared at the top level of both model definitions and workflows
(`design/inputs.md:3-13`):

```yaml
inputs:
  environment:
    type: string
    enum: ["dev", "staging", "production"]
    description: "Target environment for deployment"
```

A complete model definition (`design/inputs.md:21-37`):

```yaml
type: command/shell
typeVersion: 1
id: b015aac3-fdc6-41c5-9d91-b130fb65e78d
name: shell-env
version: 1
tags: {}
inputs:
  environment:
    type: string
    enum: ["dev", "staging", "production"]
    description: "Target environment for deployment"
methods:
  execute:
    arguments:
      run: echo "Deploying to ${{ inputs.environment }}"
```

A complete workflow with dependencies (`design/inputs.md:41-71`):

```yaml
id: abc123
name: deploy-application
jobs:
  - name: shell-environments
    description: run shell commands for environments
    steps:
      - name: first-env
        description: the first env
        task:
          type: model_method
          modelIdOrName: shell-env
          methodName: execute
          inputs:
            environment: "dev"
        dependsOn: []
        weight: 0
      - name: second-env
        description: the second env
        task:
          type: model_method
          modelIdOrName: shell-env
          methodName: execute
          inputs:
            environment: "qa"
        dependsOn:
          - step: first-env
            condition:
              type: succeeded
        weight: 0
```

`forEach` fan-out (`design/inputs.md:136-149`):

```yaml
      - name: shell-env-${{self.env}}
        description: Deploy to environment
        forEach:
          item: env
          in: ${{ inputs.environments }}
        task:
          type: model_method
          modelIdOrName: shell-env
          methodName: execute
          inputs:
            environment: ${{ self.env }}
```

Iterating an object yields `.key` / `.value` (`design/inputs.md:153-175`).

### The `--input` CLI syntax is scar tissue

`design/inputs.md:209-264` documents a value grammar that has been patched repeatedly:

- `@/path/to/file` reads a file.
- `\@literal` escapes it.
- But `@namespace/name` is a *scoped type identifier*, not a path. The disambiguation is a
  heuristic: *"if the value after `@` starts with a letter, contains at least one `/`, and has no
  `.` characters, it is a scoped identifier"* (`design/inputs.md:225-229`).
- `key:json=...` parses as JSON, and *"bypasses the `@file` shorthand and the `\@` escape"*
  (`design/inputs.md:253-254`).
- The suffix attaches to the leaf segment only: `server.config:json={...}` →
  `{server:{config:{port:8080}}}`.

Four overlapping sigils on one value slot, resolved by a heuristic that inspects for dots. This is
the clearest "do not copy" in the repo.

---

## 5. Execution drivers: an abstraction built, then deleted

`design/execution-drivers.md` opens with:

> "# Execution Drivers (superseded)
>
> **Superseded by [remote execution](./remote-execution.md)** (swamp issue #535). The driver
> abstraction — `raw`/`docker` selection, the `driver:`/`driverConfig:` fields, the driver
> extension kind, and the docker bundle-mounting machinery — has been removed. Methods run
> in-process on whichever executor holds them: the orchestrator's loopback executor, or a remote
> worker selected by step `target`/`labels`/`platform` placement. **Isolation is a worker
> deployment property** (run a containerized worker)."
> — `design/execution-drivers.md:1-12`

What was removed is instructive, because it is exactly the kind of thing a spec-writing exercise
invents on day one:

- Two built-in drivers (`raw`, `docker`) plus a third-party `ExecutionDriver` interface with
  `type` / `execute` / `initialize?` / `shutdown?` (`design/execution-drivers.md:209-220`).
- A `DriverTypeRegistry` Map-backed singleton with `@collective/name` naming
  (`design/execution-drivers.md:198-206`).
- A **six-tier config precedence chain**: CLI flag → step → job → workflow → definition → repo →
  default `raw`, with the explicit note that *"configs are **not** merged across levels"*
  (`design/execution-drivers.md:44-56`).
- A twelve-field `driverConfig` schema (image, bundleImage, command, timeout, network, memory,
  cpus, volumes, env, extraArgs …) (`design/execution-drivers.md:144-155`).
- Self-contained JS bundling with zod inlined, mounted into a container at `/swamp/`, run as
  `deno run --allow-all /swamp/runner.js` (`design/execution-drivers.md:179-189`).
- Content-fingerprint bundle caching, because *"mtime-based freshness was unreliable under
  atomic-rename saves, mtime-preserving sync tools, and sub-millisecond edits (issue #125)"*
  (`design/execution-drivers.md:229-232`).

All of it, replaced by "run a containerized worker."

### One idea from it that survived and is worth keeping: vault sentinels

> "Vault expressions (`${{ vault.get(...) }}`) produce sentinel tokens during runtime expression
> resolution. These sentinels must be replaced with actual secret values before execution. […] The
> resolution operates on cloned data — the original definition is never mutated, so sentinel tokens
> remain in the persisted definition while only the in-flight request carries resolved values."
> — `design/execution-drivers.md:299-316`

And specifically for shell: *"The shell model additionally resolves sentinels via environment
variables (`resolveForShell`) to prevent shell injection"* (`design/execution-drivers.md:305-308`).

Secrets exist as opaque sentinels through the whole authoring, persistence, and expression layer,
and are materialised only in the in-flight request to the executing method — never in the artifact
on disk. That is a genuinely good, non-obvious design.

---

## 6. Sensitive data and the vault pre-flight

Sensitive output is declared as **Zod schema metadata on the model type**, not as a redaction rule
in the data layer (`design/doctor-vaults.md:41-44`):

> A spec requires a vault when:
> - Any field in the spec's Zod schema has `.meta({ sensitive: true })`, or
> - The spec has `sensitiveOutput: true` (all fields treated as sensitive).

Three enforcement points (`design/doctor-vaults.md:17-30`):

1. **Pre-flight**, in `MethodExecutionService.executeWorkflow()` — if the model's output specs
   contain sensitive fields and no vault is configured, fail *before* the method runs.
2. **Defense-in-depth**, in `createResourceWriter()` — refuse the write rather than silently
   writing plaintext.
3. **Validation-time**, `swamp doctor vaults` — scan all definitions and exit non-zero, so CI can
   gate on it.

The reason the pre-flight exists is worth quoting, because it is precisely hyper's
destructive-operations problem:

> "Without a vault, the method execution fails at persist time — but only after the method has
> already run, potentially creating cloud resources that cannot be recorded."
> — `design/doctor-vaults.md:14-16`

That is: Swamp shipped a version where a mutation succeeded against real infrastructure and then
could not be recorded. The fix was to move the check before the side effect. Any hyper design that
validates *after* the effectful call has the same bug.

---

## 7. Run tracking: a SQLite database bolted on to fix crash recovery

`design/run-tracker.md` is a short, honest post-mortem.

> "`model method run` writes a `ModelOutput` YAML file with `status: "running"` at start, then
> updates it to a terminal state on completion. Process death (OOM, SIGKILL, power failure) leaves
> the YAML permanently stuck in "running" with no mechanism for detection."
> — `design/run-tracker.md:6-12`

The fix — a **second, local-only storage engine** alongside the YAML files:

```sql
CREATE TABLE active_runs (
  id            TEXT PRIMARY KEY,
  run_kind      TEXT NOT NULL,        -- 'model_method' | 'workflow'
  model_type    TEXT,
  method_name   TEXT,
  workflow_name TEXT,
  pid           INTEGER NOT NULL,
  hostname      TEXT NOT NULL,
  started_at    TEXT NOT NULL,
  heartbeat_at  TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'running',
  completed_at  TEXT
);
```
— `design/run-tracker.md:21-34`, at `.swamp/run_tracker.db`

Lifecycle (`design/run-tracker.md:40-52`): register with pid+hostname → heartbeat every 30s → mark
terminal (guarded by `AND status IN ('running','suspended')` to prevent a TOCTOU race) → reap stale
rows on the *next* CLI invocation (heartbeat older than 90s; same-machine checks
`isProcessDead(pid)` first, cross-machine falls back to TTL alone).

Two things follow that matter for hyper:

- The invariant that made it work was **write-once terminal YAMLs**: output files are only written
  once, in a terminal state. Mutable-status files were the root cause.
- The DB is deliberately **not synced** to remote datastores: *"PIDs and heartbeats are inherently
  local — a PID from machine A is meaningless on machine B"* (`design/run-tracker.md:91-93`).

Also from that doc, direct evidence about extension trust:

> "`swamp serve` installs a global `unhandledrejection` and `error` event handler at startup
> (`src/serve/unhandled_rejection_guard.ts`). This prevents detached rejecting promises or uncaught
> exceptions **in extension code** from terminating the server process."
> — `design/run-tracker.md:76-82`

Extensions run in the host process with enough authority to crash it. The mitigation is a global
error swallower.

---

## 8. Scar tissue

### 8.1 The maintainers are contemplating a storage rewrite

`design/unification.md` is 41 lines and reads as a confession:

> "Today, swamp has 3 different storage shapes
>
> 1. A datastore tier. Operational Data only. […] No definitions.
> 2. filesystem - source of truth for models, workflows, vaults, pulled extensions.
> 3. swamp-club - registry for extensions. pushed/pulled from with swamp push/pull
>
> There's a lot of tension across these 3 dimensions […] Reconstituting a usable environment is a
> minimum of 3 steps"
> — `design/unification.md:3-14`

The proposed fix is to collapse everything — extensions, models, workflows, reports, telemetry,
audit — into one versioned datastore, with git as a possible backend:

> "Git could be a datastore implementation. Right now it's half git half something else in
> swamp-club half also git for the extensions themselves."
> — `design/unification.md:32`

The lesson for hyper: the git/datastore split *looks* clean on a whiteboard (reviewable artifacts
in git, run data elsewhere) and is the thing they most want to undo.

### 8.2 The README documents a feature that was removed

`README.md:67-68` claims: *"Everything lives in a `.swamp/` directory inside a Git repository, with
human-friendly symlink views under `/models/` and `/workflows/`."*

`design/repo.md:134-140` says: *"The RepoIndexService is a domain event handler […] It is currently
a noop implementation (`NoopRepoIndexService`) — the old symlink-based logical views have been
removed."*

Verified in code: `src/infrastructure/repo/noop_repo_index_service.ts:50` and
`src/infrastructure/persistence/repository_factory.ts:465`. What remains are *fallback* read paths
for legacy repos — e.g. `src/libswamp/workflows/edit.ts:191` logs
`"Using symlink fallback for broken workflow"`.

Related: a full domain-event system (`ModelCreated`, `DefinitionUpdated`, `WorkflowRunFailed`,
`VaultSecretRead`, 17 events in `design/repo.md:146-178`) is still emitted by every repository —
into a noop handler. An abstraction whose only consumer was deleted.

### 8.3 Six agent-tool integrations

Each of Claude Code, Cursor, Kiro, OpenCode, Copilot, and Codex needs its own skills directory, its
own instructions file, and its own hook-config generator — `updateClaudeSettings`,
`updateCursorHooks`, `updateKiroHooks`, `updateKiroAgentConfig`, `ensureKiroCliDefaultAgent`,
`updateOpenCodePlugin`, `updateCopilotHooks`, `createCopilotHooksIfNotExists`
(`design/audit.md:41-58`). Plus per-tool `postToolUse` payload normalizers, because the four hook
payload shapes differ (`design/audit.md:11-15`).

There is also a `SUPERSEDED_SKILLS` constant and a startup check that warns when a repo carries
skill directories from an older binary (`design/repo.md:54-70`).

This is the cost of the "teach every agent to use my CLI" bet.

### 8.4 A gamification and support-funnel subsystem

`design/quests.md` (252 lines) specifies a seasonal 5×5 bingo board of objectives, points, bingo
lines, and seasonal badges, driven by `POST /api/v1/quest/events` to a backend after each
successful CLI command. The CLI has `swamp quest`, and `src/cli/commands/` contains a full
`swamp issue bug|feature|security|search|submit|edit|get|ripple` family — a support funnel that
files issues from the CLI.

Also `~/.config/swamp/telemetry/`, on by default, with a `distinct_id`, spooled and flushed
(`README.md:388-432`).

The map already puts all of this out of scope. The research confirms that call: it is roughly the
same order of magnitude of surface as the workflow engine.

### 8.5 The CLI surface

`src/cli/commands/` holds ~230 files. Command families: `access`, `audit`, `auth`, `completion`,
`config`, `data`, `datastore`, `doctor`, `extension`, `help`, `issue`, `model`, `quest`, `repo`,
`report`, `run`, `serve`, `source`, `summarise`, `telemetry`, `type`, `update`, `vault`, `version`,
`worker`, `workflow`.

Approximate non-test LOC by area:

| Area | LOC |
| --- | --- |
| `src/infrastructure/persistence/` | 16,301 |
| `src/domain/models/` | 11,794 |
| `src/domain/extensions/` | 11,278 |
| `src/libswamp/extensions/` | 9,588 |
| `src/domain/workflows/` | 9,181 |
| `src/libswamp/models/` | 5,339 |
| `src/domain/data/` | 5,329 |
| `src/libswamp/workflows/` | 5,263 |
| `src/domain/expressions/` | 3,283 |
| `src/domain/vaults/` | 2,851 |
| `src/domain/access/` | 2,173 |
| `src/domain/audit/` | 2,143 |

The extension subsystem (20,866 LOC across `domain/` + `libswamp/`) is larger than the workflow
engine (14,444 LOC). Distribution machinery outgrew the thing being distributed.

### 8.6 Security scar tissue, self-documented

`agent-constraints/adversarial-dimensions.md` is the checklist the maintainers' own review agents
run. Its security section is written in the specific, wounded register of someone who has shipped
these bugs:

> "**Authorization/execution identity consistency**: If the change authorizes an operation using
> one value and executes using another (different payload fields, raw vs. normalized forms), that
> is a critical finding. The identity used for the security decision must be the exact same
> canonical value used for execution."
>
> "**Canonicalization before security decisions**: If input can be represented in multiple forms
> (case variants, separator substitution like `.` for `/`, whitespace, `::`, `//`), is it
> normalized to a single canonical form *before* any authorization or access-control check?
> Raw-vs-canonical mismatches are bypass vectors."
> — `agent-constraints/adversarial-dimensions.md`

Corroborated by the git log: `9f0a184 fix(serve): close five security vulnerabilities in serve HA
and vault subsystems (#2069)`.

There is also a warning about module-scope state not unwinding on failure paths, where *"the
cleanup's own failure shadows the original error"* — a bug class they clearly hit.

---

## 9. Supply chain — correcting the map

Map #1's Notes say: *"Swamp is **Deno**, not npm (no lifecycle scripts, permissions off by
default, likely `deno compile`d)."* Two of those three are wrong.

**Swamp depends on npm, heavily.** `deno.json:21-58` maps 36 direct imports, of which **24 are
`npm:` specifiers** and 12 are `jsr:` — the npm side includes `zod`, `react`, `ink`,
`@aws-sdk/client-cloudcontrol`, `@marcbachmann/cel-js`, `croner`, `marked`, `fast-json-patch`,
`fast-check`, `@vercel/beautiful-mermaid`, and nine `@opentelemetry/*` packages. `deno.lock`
(version 5) resolves **200 npm packages and 18 JSR packages**.

**Lifecycle scripts:** correct, they do not run. `deno.json` sets no `nodeModulesDir` and no task
passes `--allow-scripts`, so Deno resolves npm packages into its global cache without executing
`postinstall`. **[inferred, but strongly]** — from the absence of those settings, not from an
explicit statement.

**Permissions are not "off by default."** `scripts/compile.ts:157-171` produces the shipped binary
with:

```
deno compile --allow-read --allow-write --allow-env --allow-run --allow-sys --allow-net
```

with this comment:

> "Individual flags instead of `--allow-all`: least-privilege, and prevents auto-granting future
> Deno permission categories. Trade-off: `Deno.open()` on device nodes (e.g. `/dev/ttyUSB0`)
> requires `--allow-all` in compiled binaries."

That is every permission category Deno currently has except `--allow-ffi` and `--allow-import`. The
"least-privilege" framing is about *future* categories, not present ones. The compiled `swamp`
binary has ambient read/write/exec/network authority over the machine, and anything running inside
its process inherits that.

The `deno run dev` task is narrower (`deno.json:8` — no `--allow-net`), but that is the dev path,
not the shipped one.

**Conclusion the map's premise still survives on:** Go's single static binary and absence of a
package-manager-with-lifecycle-scripts is still a real difference. But "Deno therefore sandboxed"
is not a claim Swamp's own build supports, and the npm dependency surface (200 packages) is not
meaningfully smaller than an equivalent Node tool's.

---

## 9a. Execution semantics

### 9a.1 Two nested DAGs, both level-synchronous

A workflow is jobs → steps. **Jobs form one DAG; the steps inside each job form another. There is no
global step DAG.** Both are scheduled level-synchronously: compute topological levels, run a whole
level in parallel, wait for all of it, save, advance.

The sort is Kahn's algorithm in `TopologicalSortService.sort()`
(`src/domain/workflows/topological_sort_service.ts:73-166`), producing `levels: string[][]`. Within
a level, nodes are ordered by `weight` ascending then `name.localeCompare` (`:141-147`), and
determinism is an explicit contract ("Identical inputs produce identical outputs", `:71`).

- Duplicate node names throw `DuplicateNodeNameError` before any work (`:82-93`) — this is what a
  `forEach` producing colliding expanded names hits.
- Cycles throw `CyclicDependencyError` with a DFS-recovered path (`:135-139,171-214`).
- **Dangling `dependsOn` references to unknown nodes are silently ignored** (`:117-118`). The
  validation service is expected to catch them.

The job loop (`src/domain/workflows/execution_service.ts:1813-1876`):

```typescript
for (const level of sortedJobs.levels) {
  const jobStreams = level.map((jobName) => this.runJob(...));
  for await (const event of mergeWithConcurrency(jobStreams, jobConcurrency, options?.signal)) { ... yield event; }
  await this.saveRun(workflow.id, run);
  ...
}
```

The step loop (`:2447-2538`) is the same shape, with `if (jobFailed) break;` at `:2448`.

`design/remote-execution.md:888-897` states the limitation of this shape explicitly:

> "fan-out breadth at any moment is bounded by the steps ready in the current topological level and
> their concurrency caps. A free-running ready-step queue that dispatches across level boundaries is
> future work, not v1."

That is the honest trade: level-synchronous scheduling is dramatically simpler to implement,
checkpoint, and reason about, at the cost of head-of-line blocking within a level. For hyper's
workloads it is probably the right call — but it should be a *chosen* trade, recorded as such.

One naming trap: `src/domain/workflows/workflow_scheduler.ts` is **not** the DAG scheduler. It is a
croner-based cron registry for `trigger.schedule`.

### 9a.2 The DAG comes only from explicit `dependsOn` — nothing is inferred

This is the most important execution finding, and it is a hazard.

Job edges come from `job.getDependencyNames()` (`execution_service.ts:1775-1781`); step edges from
`step.getDependencyNames()` (`:2406-2410`). **Expression references create no edges.**

The machinery to infer them was written and abandoned.
`src/domain/expressions/dependency_extractor.ts:36-39`:

```typescript
/** Artifact types that create implicit workflow dependencies. */
export const ArtifactDependencyTypes: readonly DependencyType[] = [
  "resource", "data", "file",
] as const;
```

`ArtifactDependencyTypes` is **never referenced anywhere else in `src/`** — a repo-wide grep finds
the declaration and nothing more. Expression analysis does exist, but only to *defer* evaluation
(`hasStepOutputDependency`, `dependency_extractor.ts:316`), never to add graph edges.

**The consequence:** if you write `${{ data.latest('scanner','result') }}` in step B without
`dependsOn: scanner-step`, B may run concurrently with A and read stale or absent data. Ordering is
entirely the author's responsibility, and the failure is silent and non-deterministic.

For a tool whose premise is that **an AI writes the workflow**, this is exactly the wrong default.
An AI that writes a data reference and forgets the dependency edge produces a workflow that passes
validation, usually works, and intermittently reads stale infrastructure state. Combined with the
"expressions read live disk state" finding in §3b, this is the sharpest "do not copy" in the report.

### 9a.3 Concurrency

All parallelism is in-process async generators merged by `src/infrastructure/stream/merge.ts`.
`merge()` (`:32-98`) spawns one unbounded drain task per stream into an `AsyncQueue`;
`mergeWithConcurrency()` (`:106-173`) gates each drain behind a counting `Semaphore`, and
short-circuits to plain `merge()` when the limit is absent or exceeds the stream count (`:111-114`)
— zero overhead on the default unbounded path. There is no worker pool for local execution.

Three optional `concurrency` fields, all `z.number().int().nonnegative().optional()`:
workflow-level caps parallel jobs (`workflow.ts:74`), job-level caps parallel steps (`job.ts:70`),
step-level caps `forEach` iterations (`step.ts:86`). Plus a global env override
(`execution_service.ts:3560-3575`):

```typescript
function readGlobalConcurrencyLimit(): number | undefined {
  const raw = Deno.env.get("SWAMP_MAX_CONCURRENT_STEPS");
  ...
}
function resolveEffectiveConcurrency(local, global) {
  const l = local && local > 0 ? local : undefined;
  const g = global && global > 0 ? global : undefined;
  if (l && g) return Math.min(l, g);
  return l ?? g;
}
```

Two non-obvious consequences:

- **Step-level `concurrency` is not per-step.** Resolution at `:2499-2506` takes the `min` over
  every step in the topological level and applies it to the *whole level*. One step with
  `concurrency: 2` throttles all of its level-mates.
- **The resume path drops the global cap.** `run()` uses
  `resolveEffectiveConcurrency(workflow.concurrency, readGlobalConcurrencyLimit())` (`:1785-1788`),
  but `resume()` at `:2183` is bare: `const jobConcurrency = resolvedWorkflow.concurrency;`.
  `SWAMP_MAX_CONCURRENT_STEPS` is not applied at job level on resume. **[inferred: a bug, not a
  documented distinction — the step level inside `runJob` still applies `globalLimit`.]**

For *remote* steps there is a real pool: workers advertise `capacity` (`src/worker/connect.ts:104-105`,
`--concurrency N`, default 1), reject overflow with `worker_busy`
(`src/worker/dispatch_handler.ts:117-121`), and the orchestrator re-queues.

### 9a.4 `forEach`

**Structural expansion at job start, before sorting** (`execution_service.ts:2386-2403`):

```typescript
expandedStepsMap = await new ForEachExpansionService(new CelEvaluator()).expand(job, expressionContext);
for (const step of job.steps) {
  if (!step.forEach) continue;
  const expanded = expandedStepsMap.get(step.name);
  const names = expanded ? expanded.map((e) => e.expandedName) : [];
  jobRun.replaceExpandedSteps(step.name, names);
}
```

`forEach.in` must be a single `${{ }}` (`for_each_expansion_service.ts:118-123`) and is evaluated
with **`evaluateAsync`** (`:127`), so `data.latest` / `findBySpec` / `findByTag` / `query` work
there. Arrays iterate items; objects iterate entries binding `self.<item> = {key, value}`
(`:132-157`). Anything else throws
`UserError: forEach.in must evaluate to an array or object` (`:158-162`).

**Naming.** `resolveForEachStepName` (`:57-85`) interpolates `${{ }}` in the step name. With no
expression in the name, the suffix is the item value for primitives, or the index for objects (with
a warning, `:192-198`). Eval failure appends a fallback suffix and warns (`:207-214`). Colliding
names surface as `DuplicateNodeNameError` from the sort.

**DAG edges** (`execution_service.ts:2414-2440`): each expansion becomes its own node carrying the
template's weight, and every dependency is fanned out to *all* expansions of the upstream template
(`:2425-2430`). So `A(forEach n) → B(forEach m)` is a **full n×m barrier, not a zip.** A template
that expands to zero is dropped from the graph and its `StepRun` removed
(`workflow_run.ts:438-456`).

**Undocumented and surprising: trigger conditions are not evaluated for expanded iterations.**
`execution_service.ts:2638-2654`:

```typescript
if (!forEachVar || !forEachVar.name) {
  const shouldRun = this.shouldStepRun(step, jobRun);
  ...
}
```

For a `forEach` step `forEachVar.name` is the item name and therefore non-empty, so `shouldStepRun`
is skipped entirely. **A `forEach` step's `dependsOn.condition` shapes the DAG but is never checked
at runtime** — the iterations run unconditionally once their level is reached. **[inferred: this is
not documented anywhere found.]** A `condition: {type: succeeded}` on a fan-out step is a lie.

**Results are not addressable.** There is no step-output namespace (see §9a.7). By default all N
iterations write the same instance name and **clobber each other**. The `vary` mechanism is the fix
— `execution_service.ts:1025-1050` resolves a suffix from named step-input keys and appends it to
the instance name; readers use the 3-arg form `data.latest('scanner', 'result', [self.env])`. All
step data is also tagged with `workflowRunId`, `job`, `step` (`:1014-1023`) for query-based
retrieval.

The three concurrency tiers together, `design/workflow.md:283-296`:

```yaml
concurrency: 10  # workflow level — caps parallel jobs

jobs:
  - name: fan-out
    concurrency: 5  # job level — caps parallel steps in this job
    steps:
      - name: per-item
        forEach:
          item: target
          in: ${{ inputs.targets }}
        concurrency: 3  # step level — caps forEach iterations
        task: { ... }
```

And a complete, actually-executed one from
`integration/workflow_foreach_concurrency_test.ts:177-199` — fan-out to child workflows, 6 items,
cap 3:

```yaml
jobs:
  - name: fan-out
    steps:
      - name: child-${{self.item}}
        forEach:
          item: item
          in: ${{ inputs.items }}
        concurrency: 3
        task:
          type: workflow
          workflowIdOrName: issue-718-child
          inputs: { id: "${{ self.item }}" }
        dependsOn: []
        weight: 0
```

### 9a.5 Failure semantics

States:

| Level | Enum | Source |
| --- | --- | --- |
| Step | `pending \| running \| waiting_approval \| succeeded \| failed \| skipped` | `workflow_run.ts:58-65` |
| Job | same six | `workflow_run.ts:86-93` (but `waiting_approval` is never written at job level — no `JobRun.waitForApproval()` exists) |
| Run | `pending \| running \| suspended \| succeeded \| failed \| cancelled` | `workflow_run.ts:111-118` |

**Sibling isolation is `allSettled`.** `runStep` catches everything except `WorkflowSuspendedError`
and does not rethrow (`execution_service.ts:3159-3160`):

```typescript
// Do not re-throw: merge() continues draining all step generators
// (allSettled semantics). The job generator tracks failure via step_failed events.
```

So a failed step — or a failed `forEach` iteration — does not stop its level-mates. Same at job
level.

**Propagation:**

- Step fails → `stepRun.fail(msg)` (`:3128`), `step_failed` yielded. `runJob` sets
  `jobFailed = true` only when `!event.allowedFailure` (`:2516-2518`).
- `jobFailed` → `break` before the *next* topological level (`:2448`). Later-level steps in that job
  are **never started and stay `pending`** — `jobRun.fail()` does not skip them (only
  `JobRun.skip()` does). Those pending steps persist in the run YAML, which is a slightly awkward
  record: "pending" forever.
- Downstream *jobs* get their `dependsOn` conditions evaluated (`shouldJobRun`, `:3364-3378`); an
  unsatisfied condition → `jobRun.skip()`, which *does* cascade `skip()` to its pending steps. Note
  `run()` does not break out of the job-level loop — every level is entered, and gating is purely
  via conditions.
- Run outcome (`workflow_run.ts:702-711`): `anyNonTerminal = jobs.some(j => j.status !== "succeeded"
  && j.status !== "skipped")` → `failed`, else `succeeded`. **Computed from job statuses only.**

**`completed` does not include `skipped`** (`trigger_condition.ts:212-215`) — it means
`succeeded || failed`. Downstream of an `allowFailure` step: `succeeded` is false, `failed` is true,
`completed` is true.

**`continueOnError` is spelled `allowFailure`** (`step.ts:88`, default false). The step still
records `failed` with its error and gets `allowedFailure: true` (`:3129-3132`); it just does not set
`jobFailed`.

**There are no retries.** A repo-wide grep for `retry|retries|backoff|maxAttempts` in
`src/domain/workflows/` returns **zero hits**. The only retries anywhere are worker reconnect,
lockfile acquisition, and run-tracker DB init. There is no `retry:`, `backoff:`, or `maxAttempts:`
field in any workflow schema.

For an infrastructure-automation tool this is a striking omission — transient API failures are the
common case — and it is worth deciding deliberately for hyper rather than inheriting by silence.

**Timeouts: three, and none of them bound step execution.**

1. `swamp workflow run --timeout <duration>` — a whole-run `AbortController`
   (`src/cli/commands/workflow_run.ts:605-607`). The help text is admirably honest (`:199`):
   *"Cancellation deadline … **Cooperative — only honored by methods that check AbortSignal.**"*
2. `step.queueTimeout` (`step.ts:95`) — **queue-wait only** for remote dispatch, default 600s
   (`src/serve/dispatch_service.ts:85,221-222`). Validation warns if set without placement.
3. `manual_approval.timeout` — a passive deadline, never a timer (see below).

A hard-coded `AbortSignal.timeout(30_000)` exists only for `model.method()` probes inside guards
(`execution_service.ts:2710`).

**Cancellation vs crash reaping differ, and the docs are wrong about it.** `abort` → merge closes
its queue, `run.cancel(reason)` sets `cancelled` and no-ops on terminal states
(`workflow_run.ts:720-732`). But crash reaping — `reapOrphanedWorkflowRuns`
(`src/cli/commands/serve.ts:391-467`) — calls `run.interrupt("server_crash")`, which fails in-flight
steps and jobs and sets the run to **`failed`** with tag `interrupt_reason`
(`workflow_run.ts:734-751`). `design/workflow.md:849-851` claims reaped runs are marked `cancelled`.
The code marks them `failed`.

**Nested workflows** (`runWorkflowStep`, `:3170-3362`): depth cap of 10
(`MAX_WORKFLOW_NESTING_DEPTH`, checked `:3210-3229`), ancestor-set cycle detection (`:3231-3249`),
child failure surfaces the first non-allowed child step error (`:3335-3353`). Both structural
failures respect `allowFailure`.

Event ordering is pinned by contract tests (`execution_service_test.ts:3311,3350`). Note that a
failing run still emits `completed` — with `run.status === "failed"` — never a distinct `failed`
terminal event.

### 9a.6 Resumability

**Yes — but only at job-level checkpoints, and only for `suspended` or `failed` runs.**

Runs persist to `{repoDir}/.swamp/workflow-runs/{workflowId}/workflow-run-{runId}.yaml`
(`src/infrastructure/persistence/yaml_workflow_run_repository.ts:67-73`), atomic write at `:416`,
plus a status index. Content is the full `WorkflowRunSchema` job/step tree.

**Checkpoint granularity is the key limitation**, and it is pinned by a contract test
(`execution_service_test.ts:3395`, *"CONTRACT: run is persisted at start, after each level, and at
completion"*) which asserts exactly **four** saves for a two-job / two-level workflow: after
`run.start()`, after each job level, and after `run.complete()` — at `:1741`, `:1853`, `:1920`, plus
an extra save at the suspend site (`:2821`).

**There is no per-step save.** A crash mid-level loses every step outcome in that level. `interrupt()`
at least marks running steps `failed` on reap, so the record is consistent rather than stuck in
"running".

Live state that is deliberately *not* persisted: the mutated `expressionContext.model[...]`
accumulated as steps complete (`:3044-3086`). On resume the whole context is rebuilt from disk via
`modelResolver.buildContext()` (`:2090`) — which is correct only because step outputs live in the
data store rather than in memory. That is a good property, and it falls out of "everything a step
produces is durable data".

**Two resume mechanisms** (`execution_service.ts:2017-2343`):

- **Gate resume** (no `--from`): requires `status === "suspended"` (`:2056-2060`) and no remaining
  `waiting_approval` step (`:2063-2071`).
- **`--from <step>`**: requires `status === "failed"` (`:2049-2054`). `computeStepsToReset`
  (`:3582-3697`) validates the name is a *template* name, builds a combined step+job downstream
  graph (job deps become all-steps-to-all-steps edges), BFS-collects transitive dependents, then
  maps template names onto persisted names — including `forEach`-expanded ones via prefix match,
  guarded against collisions (`:3658-3693`).

**Skip logic:** a job whose persisted status is `succeeded|failed|skipped` is replaced with an empty
generator (`:2205-2221`); any step already in one of those states returns immediately (`:2607-2629`).
Note **`failed` steps are skipped too** — a gate resume does not retry earlier failures; only
`--from` does.

**The whole workflow is re-evaluated on resume** (`:2109-2113`). Only *execution* is skipped, not
expression evaluation. `--input` deep-merges over the suspended inputs (arrays replace, they do not
concatenate; `src/domain/inputs/input_merge.ts:24-49`), resume winning on collision. Only **key
names** are persisted for audit (`workflow_run.ts:791-797`) — values never touch disk.

**The idempotency primitive is `guard`** (`step.ts:96`): a CEL expression evaluated per step (and per
`forEach` iteration, with `self.*` bound) *after* dependency checks and *before* execution; truthy →
`skip()` with `reason: "guarded"` (`:2755-2773`). Guards may invoke `model.method(...)` as a live
probe (`:2693-2750`). A guard error fails the step (`:2776-2789`).

The stated design intent (`design/workflow.md:206-208`) is worth quoting because it is an explicit
refusal to be clever:

> "Steps without a guard always execute on resume. If you didn't write a guard, you're saying
> 'always run this step.'"

### 9a.7 Approval gates: a run that suspends, persists, and exits

**Not a prompt.** This is the mechanism, end to end.

**1. Suspend.** `runStep` starts the step (`:2793` — this timestamp is the deadline anchor), then
(`:2806-2828`):

```typescript
if (task.type === "manual_approval") {
  stepRun.waitForApproval();
  yield { kind: "approval_requested", runId: run.id, jobId: job.name, stepId: stepName, prompt: task.prompt, timeout: task.timeout };
  run.suspend(stepExprContext?.inputs);
  await this.saveRun(workflow.id, run);
  throw new WorkflowSuspendedError(job.name, stepName, task.prompt, task.timeout);
}
```

Step → `waiting_approval`; run → `suspended` with inputs captured; job stays `running`.

**2. Unwind.** `WorkflowSuspendedError` (`:191-201`) exists because a generator `yield` can only
travel one frame. It is rethrown untouched by `runStep` (`:3104-3108`) and `runJob` (`:2553-2557`),
and converted to the terminal `suspended` event in `run()`'s catch (`:1930-1944`), which also marks
the run-tracker row `suspended`. Because siblings run under `merge`, there are two belt-and-braces
re-detections that reconstruct the error from `run.findWaitingApprovalStep()` (`:2521-2537` and
`:1855-1875`).

**3. Approve** (`src/libswamp/workflows/approve.ts:68-171`) resolves the run via
`resolveSuspendedRun` (explicit `--run`, else the exactly-one-suspended run, else an error listing
ids), finds the step in `waiting_approval`, checks the deadline, then (`:147-156`):

```typescript
const decidedBy = input.decidedBy ?? Deno.env.get("USER") ?? Deno.env.get("USERNAME") ?? "unknown";
step.recordApprovalDecision({ approved: true, reason: input.reason, decidedBy, decidedAt: new Date().toISOString() });
step.succeed();
await deps.runRepo.save(createWorkflowId(run.workflowId), run);
```

**This is a pure record edit — the run stays `suspended` and nothing executes.** Over the wire the
authenticated principal overrides the client-supplied `decidedBy`
(`src/serve/handlers/workflow_handlers.ts` ~854). Locally, `decidedBy` falls back to `$USER` — i.e.
it is an unauthenticated self-assertion, which is honest for a single-user tool but should not be
mistaken for an attestation.

**4. Reject** (`src/libswamp/workflows/reject.ts:156-170`): `step.fail(reason)`, `matchedJob.fail()`,
`run.complete()` → the run becomes `failed`. Terminal.

**5. Resume is always a separate invocation.** Nothing auto-resumes: the only callers of
`service.resume` are `workflow_resume.ts:446` and two explicit `workflow.resume` request handlers.

**6. The timeout is passive.** `src/domain/workflows/approval_timeout.ts:47-67` is a pure function;
`expired = elapsedSeconds > taskData.timeout`. **No timer, no reaper.** It is consulted only by
`approve`, `reject`, and `approvals`. Once the window lapses, approve **and** reject both refuse and
`approvals` hides the row — the run is then reachable only via `swamp workflow cancel`. A gate that
times out therefore strands its run rather than failing it.

**7. Programmatic gates** exist: `src/domain/models/workflow_gate_service.ts:27-41` (port),
`src/libswamp/models/workflow_gate.ts:37-125` (adapter). Identity is *stamped, not passed* —
``decidedBy = `model:${definitionName}/${methodName}` `` (`:49-50,91-92`), and the option types have
no `decidedBy` field. Remote workers hard-error (`src/worker/remote_method_context.ts:604-618`). So
a model can approve a gate, but it cannot impersonate a human.

#### The CI failure mode

This is the finding hyper most needs to absorb.

- **There is no TTY prompt anywhere on the run path.** `promptConfirmation`
  (`src/cli/prompt_helpers.ts:35`) is imported only by destructive *management* commands
  (`vault_delete`, `workflow_delete`, `extension_push`, …) — never by `workflow_run.ts`,
  `workflow_resume.ts`, or `workflow_approve.ts`. `workflow_run.ts:101` says: *"Blocks until the run
  completes, suspends (manual approval), fails, or is cancelled. There is no async/detached mode."*
- **The exit code on suspension is 0.** The console renderer's `suspended` handler
  (`src/presentation/renderers/workflow_run.ts:646-676`) prints a "Suspended" line plus
  approve/resume hints and **never sets `this._failed`** — in contrast to `cancelled` (`:631`) and
  a failed `completed` (`:577`), which do. `workflowFailed()` (`:684`) is the sole gate on
  `Deno.exitCode = 1` (`src/cli/commands/workflow_run.ts:518-526`). JSON mode behaves the same
  (`:783-797`) though it at least emits an `approvalRequired: {stepId, jobId, prompt, timeout}`
  block.

**A CI job that gates on exit code reads a half-executed, gated workflow as success.** No
`--fail-on-suspend` flag exists, and no test anywhere asserts the exit code on suspension.

Given the map's decision that hyper's two execution environments are the laptop and GitHub Actions,
this is a concrete, copyable bug: any "pause for approval" concept must define its non-interactive
exit-code contract *first*, not as a rendering detail.

**`forEach` + `manual_approval`** is documented at `design/workflow.md:182-184` as "N parallel
approval gates, each independently approvable". Nothing forbids it and nothing tests it. Per-name
approval does work (`approve.ts:110-117` matches expanded names), but `findWaitingApprovalStep`
(`workflow_run.ts:838-847`) returns only the **first** match, so `swamp workflow approvals` shows
one row per run at a time and resume refuses until all are cleared. Whether all N iterations
reliably reach `waiting_approval` is a race — the first throw calls `queue.abort()` (`merge.ts:73`),
tearing down siblings that had not started. **Could not determine** how many are reliably marked; no
test pins it.

### 9a.8 Expression evaluation in the execution path

**Real CEL**, `@marcbachmann/cel-js@7.6.1` (`deno.json:42`), wrapped by `CelEvaluator`
(`src/infrastructure/cel/cel_evaluator.ts:315`).

Registered custom functions, all receiver methods on registered types (`cel_evaluator.ts:330-441`):
`data.latest(model, name)`, `.version(model, name, int)`, `.listVersions(model, name)` — each with a
vary-aware overload taking `list<dyn>` as a third argument — plus `data.findByTag(k, v)`,
`data.findBySpec(model, spec)`, `data.query(predicate[, select])`, `file.contents(model, spec)`,
`workers.connected()`, and `model.method(name, method[, inputs])`.

**`range()` is NOT registered by Swamp**, despite `design/workflow.md:698-707` using
`in: ${{ range(1, 97) }}` as a guard example. Whether `@marcbachmann/cel-js` provides it natively
**could not be determined** — the dependency is not vendored in the clone.

**Sync vs async, handled well.** `evaluate()` (`:455`) is synchronous and **detects a leaked Promise
and throws** (`:475-486`) rather than silently coercing it to `{}`. `evaluateAsync()` (`:521`)
awaits. Anything touching `data.*` (except `listVersions`, deliberately sync), `file.contents`, or
`model.method` requires the async form. That explicit Promise-leak check is a small thing done right.

**The context shape** (`src/domain/expressions/model_resolver.ts:300-353`):

| Key | Contents |
| --- | --- |
| `model` | `Record<modelName, {input, resource?, file?, execution?, definition?}>` — **this is how prior step outputs are reached** |
| `self` | `{id,name,version,tags,globalArguments}` plus `forEach` iteration vars |
| `inputs` | workflow inputs |
| `env` | **the full process environment, not redacted** (there is a security note at `:325-331`) |
| `vault` | `vault.get(vaultName, key)` |
| `data` | `DataNamespace` (`:195-241`) |
| `file`, `workers` | lazy file contents / worker query |
| `workflowRunId`, `run` | `run.{id, workflowId, workflowName, startedAt, tags}` |
| `webhook` | `{body, headers, route}` — only inside `trigger.inputs` |
| `workflow` | workflow metadata |

**There is no `steps.*` or `jobs.*` namespace.** Prior step outputs are never addressed as
`steps.foo.outputs.bar`. They are reached either in memory via
`model.<name>.resource.<spec>.<instance>` — mutated after each step at `:3044-3086` — or from disk
via `data.latest(...)` / `data.query(...)`.

This is a deliberate and interesting choice: **the only channel between steps is durable data.**
There is no ephemeral step-output bus. It is what makes resume-from-disk correct, and what makes
every intermediate value queryable after the fact. It is also why `forEach` iterations clobber each
other without `vary` — the channel is a keyed store, not a per-step slot.

Per-step context is a **shallow copy** (`:2658-2676`) with `self` replaced, so `model` is shared by
reference across parallel siblings while `self` is isolated (pinned by
`execution_service_test.ts:897`).

**Staged evaluation.** `WorkflowExpressionEvaluator.evaluate`
(`src/domain/workflows/expression_evaluators.ts:62-149`) evaluates the whole workflow body up front
and **deliberately leaves raw**: runtime/vault expressions (`:93`), anything matching `\bself\??\.`
(`:97`), `run.*` / `workflowRunId` (`:102-107`), `forEach.in` (`:109`), `trigger.inputs` (`:116`),
`guard` (`:122`), and `task.inputs` / assert `message` with a step-output dependency (`:128-133`).
Those resolve later, at step-execution time, from the live context.

**Error semantics are asymmetric by design** (`:56-60,158-166`):

- **Workflow evaluation is strict** — any throw propagates. The stated rationale: *"workflows declare
  their evaluation order explicitly, so a thrown CEL error is a real bug."*
- **Definition evaluation is lenient** — catches and leaves unresolved, surfacing later through a
  `globalArgs` Proxy.
- Guard errors fail the step; `forEach` step-*name* eval failure warns and appends an index fallback;
  assert `message` interpolation failure is **silently swallowed**, leaving the raw text (`:2855-2857`).

Also worth noting: `dependsOn` is **not** expression-driven — it is the structural `TriggerCondition`
DSL. Two languages, one file. That is arguably the right split (structure is data, values are
expressions), and it is why the DAG is statically analysable even though values are not.

### 9a.9 Doc/code divergences found in the execution path

- `design/workflow.md:849-851` says orphan reaping marks runs `cancelled`; `serve.ts:422,465` calls
  `run.interrupt()`, which sets `failed`.
- `execution_service.ts:1856-1858` claims `merge()` swallows `WorkflowSuspendedError`;
  `merge.ts:95-97,170-172` rethrow `firstStreamError` unless the abort was induced.
- `design/workflow.md:698-707` uses `range()`, which Swamp does not register.

### 9a.10 What could not be determined in the execution path

- Whether `range()` resolves natively via `@marcbachmann/cel-js`.
- How many of N `forEach`-expanded `manual_approval` gates reliably reach `waiting_approval` before
  `merge`'s `queue.abort()` tears down siblings. No test covers the combination.
- Whether `JobRun`'s `waiting_approval` enum member (`workflow_run.ts:89`) is dead schema or has a
  writer that was missed — no `JobRun.waitForApproval()` was found.
- Whether the missing global concurrency cap on the resume job path (`:2183`) is an oversight or
  intentional. No comment or test either way. **[inferred: oversight.]**

---

## 9b. The data layer

This is the part of Swamp most directly relevant to hyper's "what changed since Tuesday" half of
oversight, so it is worth going slowly.

### 9b.1 Two tiers: authoritative files, derived index

**The files are the truth. SQLite is a cache.** This is the single most important structural fact
about Swamp's data layer, and it is a good decision.

Tier A — the authoritative store, a versioned directory tree
(`src/infrastructure/persistence/unified_data_repository.ts:70-82`):

```
.swamp/data/{normalized-type}/{model-id}/{data-name}/
  1/
    raw              # Content (binary or text)
    metadata.yaml    # Metadata
  2/
    raw
    metadata.yaml
  latest             # Text file containing current version (e.g. "2")
```

Tier B — `_catalog.db`, a SQLite index that is **repo-local, never namespaced, and excluded from
datastore sync**, with this doc comment (`catalog_store.ts:58-67`):

> "Content is NOT stored — it remains on disk in the existing versioned file layout. The catalog is
> local-only and excluded from datastore sync. It self-heals by triggering a backfill when missing
> or not yet populated."

`node:sqlite` `DatabaseSync`, `PRAGMA busy_timeout=5000`, `PRAGMA journal_mode=WAL`
(`catalog_store.ts:110-115`).

Because the index is derived, **schema evolution is drop-and-rebuild, not migration**. There is no
migrations directory. `CATALOG_SCHEMA_VERSION = "4"` (`catalog_store.ts:84`); `migrateIfNeeded()`
(`:217-232`) does `DROP TABLE IF EXISTS catalog`, recreates, and clears the `populated` flag so the
next query triggers a full backfill from the YAML files. That is only possible because the files
are authoritative — and it removes an entire class of migration bugs. Worth copying.

### 9b.2 The record schema

`metadata.yaml` is defined by `DataMetadataSchema` (`src/domain/data/data_metadata.ts:121-149`) —
see §2.6 above for the field table. The catalog DDL, verbatim from `catalog_store.ts:166-209`:

```sql
CREATE TABLE IF NOT EXISTS catalog (
  namespace       TEXT NOT NULL DEFAULT '',
  type_normalized TEXT NOT NULL,
  model_id        TEXT NOT NULL,
  data_name       TEXT NOT NULL,
  id              TEXT NOT NULL,
  version         INTEGER NOT NULL,
  is_latest       INTEGER NOT NULL DEFAULT 1,
  model_name      TEXT NOT NULL,
  spec_name       TEXT NOT NULL DEFAULT '',
  data_type       TEXT NOT NULL DEFAULT '',
  content_type    TEXT NOT NULL DEFAULT '',
  lifetime        TEXT NOT NULL DEFAULT '',
  owner_type      TEXT NOT NULL DEFAULT '',
  streaming       INTEGER NOT NULL DEFAULT 0,
  size            INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL,
  tags            TEXT NOT NULL DEFAULT '{}',
  owner_ref       TEXT NOT NULL DEFAULT '',
  workflow_run_id TEXT NOT NULL DEFAULT '',
  workflow_name   TEXT NOT NULL DEFAULT '',
  job_name        TEXT NOT NULL DEFAULT '',
  step_name       TEXT NOT NULL DEFAULT '',
  source          TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (namespace, type_normalized, model_id, data_name, version)
);
```

Plus nine indexes. The catalog carries **no `checksum` and no `lifecycle`/`renamedTo` column** —
those live only in `metadata.yaml`.

### 9b.3 Addressing: path-addressed, not content-addressed

The primary key is a five-tuple that is *literally the directory path*:
`(namespace, type_normalized, model_id, data_name, version)`.

- `id` is a random UUIDv4 (`src/domain/data/data_id.ts:35-37`), stable across versions of the same
  name (`data.ts:294`), but **not the lookup key** — `findById` scans; `findByName` is the fast
  path.
- `checksum` is SHA-256 hex of the content (`unified_data_repository.ts:2006-2014`) but is
  **advisory only** — nothing addresses or dedupes by it, and `append()` *deletes* it rather than
  recompute (`:777-779`).
- `"latest"` is a reserved data name (case-insensitive) because it would collide with the marker
  file (`data.ts:36-45`, enforced at `unified_data_repository.ts:564-569`).

So: **Swamp is not content-addressed.** The map's phrasing "versioned, immutable, provenanced
artifacts" is accurate about intent, but if hyper wants content addressing that is a design hyper
would be adding, not inheriting.

### 9b.4 What "immutable" actually means

Version allocation is **mkdir-as-mutex** — monotonic, gap-tolerant, and pleasingly simple
(`unified_data_repository.ts:1862-1892`):

```typescript
const maxRetries = 100;
for (let attempt = 0; attempt < maxRetries; attempt++) {
  const versionDir = this.getPath(type, modelId, dataName, nextVersion);
  try {
    await Deno.mkdir(versionDir);
    return { version: nextVersion, versionDir, priorVersions: versions };
  } catch (error) {
    if (error instanceof Deno.errors.AlreadyExists) {
      nextVersion++;
      continue;
    }
```

Every `save()` writes a brand-new version directory; existing `{n}/raw` and `{n}/metadata.yaml` are
never rewritten (`:588-629`).

**But there are two documented exceptions where existing bytes are mutated:**

1. `append()` — for `streaming: true` data, opens `{latest}/raw` with `{ append: true }` and
   rewrites that version's `metadata.yaml` with a new `size` and a *deleted* `checksum`
   (`:735-789`). Append-only within a version, not copy-on-write.
2. The `latest` marker is destructively replaced on each write (`:1985-2004`).

And **deletion is a hard `Deno.remove(..., { recursive: true })`**, not a tombstone (`:865-934`) —
`swamp data delete --force` removes the version directory (or the whole subtree) and drops the
catalog rows. So the store is append-mostly, not append-only.

Tombstones are a *separate, additive* concept, and are used well:

- **Cloud-resource deletion**: `withDeletionMarker({ version: data.version + 1 })` is written after
  a successful `delete`-kind method, with content `{...lastKnownState, deletedAt, deletedByMethod}`,
  *so that `data.latest()` still resolves after deletion and re-runs stay idempotent*
  (`src/domain/models/method_execution_service.ts:949-1017`, `data.ts:248-261`). This is a genuinely
  clever answer to "how does a declarative system represent a thing that no longer exists".
- **Rename**: `withRenameMarker({version, renamedTo})` writes a forward-reference tombstone on the
  old name; the new name gets a fresh `id` and a fresh version chain; lookups follow the chain, max
  depth 5 (`unified_data_repository.ts:962-1039,1440-1459`).

### 9b.5 Two-phase commit for run outputs

`saveDeferred()` writes the version directory but inserts the catalog row with `is_latest: 0`
("invisible to latest-based queries", `:705`) and does **not** advance the `latest` marker.
`advanceLatestMarkers(receipts)` commits (`:1300-1325`); `rollbackVersions(receipts)` deletes the
version directory and its row (`:1327-1351`). A receipt is `{type, modelId, dataName, version}`
(`src/domain/data/repositories.ts:74-79`).

The catalog's latest invariant is maintained transactionally (`catalog_store.ts:287-301`):
`BEGIN IMMEDIATE` → `UPDATE catalog SET is_latest = 0 WHERE …` → `INSERT OR REPLACE … is_latest = 1`
→ `COMMIT`.

Writes also enforce ownership: `save()` throws `OwnershipValidationError` if
`existing.isOwnedBy(data.ownerDefinition)` is false, comparing `ownerType` + `ownerRef`
(`:576-586`).

### 9b.6 Querying: CEL over the catalog, with an allowlist

There is **no SQL passthrough** exposed to users. The query language is CEL over the catalog rows.

The queryable field allowlist (`src/domain/data/query_predicate.ts:23-48`):

```typescript
export const QUERY_FIELDS = new Set([
  "id", "name", "version", "isLatest", "createdAt", "attributes", "tags",
  "modelName", "modelType", "specName", "dataType", "contentType",
  "lifetime", "ownerType", "streaming", "size", "content",
  "ownerRef", "workflowRunId", "workflowName", "jobName", "stepName",
  "source", "ns",
]);
```

Unknown identifiers throw a `UserError` that lists the available set (`:227-240`). For an
AI-authored query language that is exactly right: a hallucinated field name produces an actionable
error containing the correct vocabulary.

**Implicit latest-only, with explicit opt-in to history** (`data_query_service.ts:344-357`):

```typescript
const opensHistory = rootIds.some((id) => HISTORY_OPT_IN_FIELDS.has(id));
const effectivePredicate = opensHistory
  ? predicate
  : `(${predicate}) && isLatest == true`;
```

with `HISTORY_OPT_IN_FIELDS = new Set(["version", "isLatest"])` (`query_predicate.ts:55`).
Mentioning `version` or `isLatest` anywhere at the AST root opens history. Neat, and a little
magic — the same predicate means different things depending on which fields it happens to name.

**SQL pushdown is deliberately minimal**: only `is_latest = ?` and a top-level
`modelName == "literal"` equality (walking `&&` conjuncts, never descending into `||`) are pushed
into SQLite; the full CEL still runs per row (`data_query_service.ts:382-404`,
`query_predicate.ts:196-221`). Paged at 1000 rows (`catalog_store.ts:547-566`).

**Lazy content loading driven by AST analysis**: `raw` is read from disk only if the predicate or
`--select` references `attributes` or `content` (`data_query_service.ts:368-380`).

Real queries, from the command's own `.example()` declarations (`src/cli/commands/data_query.ts:65-77`)
and the bundled skill (`.claude/skills/swamp/references/data/reference.md:5-27`):

```bash
swamp data query 'modelName == "scanner"'
swamp data query 'size > 1048576'
swamp data query 'dataType == "resource" && tags.env == "prod"'
swamp data query 'modelName == "scanner"' --select '{"name": name, "os": attributes.os}'
swamp data query 'attributes.status == "failed"' --select 'name'
```

**Every other `data` subcommand is documented sugar over `query`**
(`.claude/skills/swamp/references/data/guide.md:19-33`):

| Shortcut | Underlying query |
| --- | --- |
| `data get <m> <n>` | `data query 'modelName == "<m>" && name == "<n>"' --select content` |
| `data list <m>` | `data query 'modelName == "<m>"'` |
| `data list --run <id>` | `data query 'workflowRunId == "<id>"'` |
| `data versions <m> <n>` | `data query 'modelName == "<m>" && name == "<n>" && version >= 0' --select 'version'` |
| `data search --tag env=prod` | `data query 'tags.env == "prod"'` |

One primitive, many named shortcuts, with the mapping documented. That is a good pattern for a CLI
an AI has to drive.

**Caveat:** `swamp data search` is *not* catalog-backed. It is a linear in-memory scan over
`findAllGlobal()` with a lowercase substring match on four fields only — no index, no FTS, no
content search (`src/libswamp/data/search.ts:169-225`).

### 9b.7 Redaction: write-time, via vault indirection

There is **no `redact:` or `sensitive:` key in any YAML**. The annotation is Zod schema metadata
read at runtime from `z.globalRegistry`.

`SensitiveMetadata` (`src/domain/models/sensitive_field_extractor.ts:35-42`):

```typescript
interface SensitiveMetadata {
  sensitive?: boolean;
  vaultName?: string;
  vaultKey?: string;
}
```

Used as `z.string().meta({ sensitive: true })`. Spec-level blanket switch on `ResourceOutputSpec`
(`model.ts:66-70`).

**Write path** (`processSensitiveResourceData`, `data_writer.ts:381-474`): snapshot the value →
`vaultService.put(targetVault, vaultKey, stringValue)` → replace the value in place with a vault
reference:

```typescript
const vaultRef = `\${{ vault.get('${targetVault}', '${vaultKey}') }}`;
if (field.path.includes(".")) {
  setNestedValue(data, field.path, vaultRef);
} else {
  data[field.path] = vaultRef;
}
```

Default key: `sanitizeVaultKey(\`${modelType.normalized}/${modelId}/${methodName}/${specName}/${instanceName}/${field.path}\`)`
(`:445-448`). **Fail-closed** if no vault is configured (`:421-431`, `:644-654`).

So the bytes at `{version}/raw` contain `${{ vault.get(...) }}` — never the secret. The artifact is
safe to commit, sync to S3, or hand to a reviewer. That is the right shape.

**Read path** re-resolves and registers each value with the redactor
(`data_writer.ts:742-786`, key line `:764: redactor?.addSecret(value);`), called from
`data_query_service.ts:191,298` and `data_record_mapper.ts:83`. Note that `querySync()` does **not**
resolve vault refs (`data_query_service.ts:318`).

**Second, independent layer: `SecretRedactor`** (`src/domain/secrets/secret_redactor.ts:24-60`) — a
set of known secret strings, longest-first replacement with `***`, values under 3 characters
ignored, JSON-escaped variants auto-added. Applied to data *before it hits disk*
(`data_writer.ts:704-707`):

```typescript
const serialized = redactor
  ? redactor.redact(JSON.stringify(data))
  : JSON.stringify(data);
const handle = await writer.writeText(serialized);
```

and to per-run log files, subprocess output, and OTLP exports.

**A third chokepoint at persistence**: `yaml_definition_repository.ts:347-364` refuses to persist a
definition whose `sensitive` global argument holds a literal value —

```typescript
const leakedArgs = findLiteralSensitiveGlobalArgs(
  modelDef?.globalArguments, data.globalArguments,
);
if (leakedArgs.length > 0) {
  throw new UserError(
    literalSensitiveGlobalArgsMessage(leakedArgs),
    LITERAL_SENSITIVE_GLOBAL_ARG_CODE,
  );
}
```

Only `undefined`/`null`/empty/expression-only values pass. Mirrored in `models/create.ts:222`,
`direct_execution.ts:238,300`, and scanned read-only by `swamp doctor secrets`.

**Documented gaps in redaction** (from the code and docs, not speculation):

- **Audit logs are explicitly not covered.** `design/vaults.md:603`:
  `| Audit log | Not covered | Not covered (see follow-up #244) |`. Confirmed — zero `redact` /
  `sensitive` references in `src/domain/audit/`.
- `extractSensitiveFields()` only walks object shapes, so sensitive fields inside `z.record()` or
  `z.union()` are missed — self-documented at `sensitive_field_extractor.ts:252-255`.
- Report redaction returns args unchanged for workflow scope and for unregistered model types
  (`report_execution_service.ts:299,302`).
- **[inferred, unconfirmed]** `handle.attributes = { ...data }` (`data_writer.ts:708`) is assigned
  *after* the `redactor.redact()` serialize, so the in-memory handle carries data that passed vault
  substitution but not the value-based scrub. Consumers of `handle.attributes` were not fully
  traced.

### 9b.8 Provenance: what is actually recorded

Per record version, in `metadata.yaml` under `ownerDefinition`, mirrored into seven catalog columns:

| Field | In YAML | Catalog column | Source |
| --- | --- | --- | --- |
| `ownerType` | yes | `owner_type` | `"model-method"` \| `"workflow-step"` \| `"manual"` |
| `ownerRef` | yes | `owner_ref` | `modelId`, or `${modelType}:${methodName}`, or `${workflowId}:${jobName}:${stepName}` |
| `definitionHash` | optional, legacy | — | back-compat only |
| `workflowId` | yes | — | |
| `workflowRunId` | yes | `workflow_run_id` | |
| `workflowName` | yes | `workflow_name` | |
| `jobName` | yes | `job_name` | |
| `stepName` | yes | `step_name` | |
| `source` | yes | `source` | |

Plus per version: `createdAt`, stable `id`, `version`, `size`, SHA-256 `checksum`, `contentType`,
`tags`, and `namespace`. All six workflow-provenance fields are first-class queryable, so
`workflowRunId == "…" && stepName == "dedup"` works.

**What is NOT recorded, and this matters for hyper:**

- **No definition version or git commit SHA on the data record.** `definitionHash` is optional and
  legacy. You cannot ask "which revision of the definition produced this?" from the record alone.
- **No `methodName` as a distinct field** — only embedded inside `ownerRef` for one owner shape.
- **No actor or user identity.**
- **No extension or bundle digest** on the record. (`sourceFingerprint` exists on the model type and
  goes into `ExecutionProvenance`, and a bundle fingerprint was added to execution provenance as
  recently as commit `f1cb668`, but it does not land on the data record.)
- `swampSha` — the repo git SHA — exists only on `ReportContext`
  (`src/domain/reports/report_context.ts:35`), not on the data record.

So Swamp's provenance answers "which run, job and step wrote this" but **not** "what code and what
configuration produced it". For a tool whose thesis is human review of AI-authored automation, that
is the more important question, and it is the gap hyper should close.

### 9b.9 There is no diff

**Swamp has no diff feature.** No version-to-version comparison, no `swamp data diff`, no drift
detection over stored data. Every `diff`/`drift` hit in `src/` concerns extension versions, repo
upgrades, or schema-version mismatches. `design/high-level.md:20-23` self-flags drift detection as
aspirational and not implemented.

What exists instead is time-window filtering:

- `findAllGlobalSince(cutoff)` (`unified_data_repository.ts:236-247,315-371`) — a two-stage filter:
  a `Deno.stat` mtime pre-filter on `metadata.yaml`, then parse and verify `createdAt >= cutoff`.
  The comment at `:231-234` explains the catalog cannot serve this because it does not track
  `created_at` for that path. (This mtime pre-filter is also *why* output YAMLs must be write-once —
  see §7.)
- `swamp summarise --since` — activity counts grouped by model/method and workflow run. Not a diff.
- `swamp data search --since <duration>`.

History is readable (`version >= 0` opens the full chain), so a diff is *composable* client-side —
but nothing in the repo composes it.

This is the single largest gap between Swamp and what the map wants from hyper. The "what changed
since Tuesday" half of oversight is **not** solved by Swamp; it is only made possible in principle
by keeping versions around.

### 9b.10 Retention

Three per-record policies, frozen into `metadata.yaml` at creation and inherited by `withNewVersion`
(`data.ts:287-309`):

- **`lifetime`** — expiry of the whole item. `"infinite"` never expires (and is the documented
  default for resources); `"ephemeral"`; `"job"` / `"workflow"` expire when the owning run
  completes; duration strings expire at `createdAt + duration`.
- **`garbageCollection`** — version retention. A number keeps the N most recent; a duration keeps
  versions newer than that, never removing the max version (`unified_data_repository.ts:1698-1830`).
- **Optional write-time cap** — `pruneExcessVersions()` inline on save when `enableWriteGc` is set
  and `garbageCollection` is numeric (`:633-641,1894-1924`).

Commands: `swamp data gc` (expiry + version pruning; expired items are **soft-deleted** via
`removeLatestMarker()` — version directories stay on disk but become unreachable, `:936-960`),
`swamp data prune` (reclaims data orphaned by a deleted definition — needed because `delete` throws
`Model not found` and `gc` honours the frozen `infinite` lifetime, so orphans would accumulate
forever, `design/datastores.md:279-300`), `swamp datastore compact` (`VACUUM` +
`wal_checkpoint(TRUNCATE)`), and `swamp run gc` for the separate `workflow-runs/` and `outputs/`
stores (30-day default, terminal runs only, `design/repo.md:118-132`).

**There is no scheduler.** GC is CLI-triggered only; `autoGc` in `.swamp.yaml` runs it after model
method runs (`design/repo.md:108-114`), and `design/repo.md:131` states plainly: *"Manual-only:
There is no automated or post-run GC for these stores yet."*

Note the shape here: **retention policy is frozen into each record at write time, not applied from
current config at read time.** Changing the policy does not affect data already written. That is a
real decision with real consequences and it is not obviously the right one.

---

## 9c. Vaults: encryption, refresh hooks, audit

Headline: **AES-256-GCM + PBKDF2-HMAC-SHA256 at 100,000 iterations, no AAD, no key wrapping.** No
Argon2id, no scrypt, no XChaCha20, no age/GPG, no OS keychain, no cloud KMS anywhere in the tree.
**No key rotation and no recovery path.** The audit trail has **no tamper-evidence**.

### 9c.1 The cipher

`src/domain/vaults/local_encryption_vault_provider.ts:700-723`:

```typescript
private async encrypt(
  value: string,
  key: CryptoKey,
  salt: Uint8Array,
): Promise<EncryptedData> {
  const iv = crypto.getRandomValues(new Uint8Array(12)); // 96-bit IV for AES-GCM
  const encodedValue = new TextEncoder().encode(value);

  const encrypted = await crypto.subtle.encrypt(
    {
      name: "AES-GCM",
      iv: iv,
    },
    key,
    encodedValue,
  );

  return {
    iv: this.arrayBufferToBase64(iv),
    data: this.arrayBufferToBase64(encrypted),
    salt: this.arrayBufferToBase64(salt),
    version: 2,
  };
}
```

The KDF (`:420-440`):

```typescript
const key = await crypto.subtle.deriveKey(
  {
    name: "PBKDF2",
    salt: salt,
    iterations: 100000,
    hash: "SHA-256",
  },
  keyMaterial,
  { name: "AES-GCM", length: 256 },
  false,
  ["encrypt", "decrypt"],
);
```

Salt is 16 bytes from `crypto.getRandomValues`, fresh **per put**, per secret, and separately for
annotations and refresh hooks (`:133,260,354`).

Three observations:

- **No AAD.** Grepping `additionalData` and `tagLength` across `src/` returns zero hits. The
  `salt` and `version` fields in the on-disk envelope are therefore unauthenticated. The GCM tag
  still prevents forgery of the ciphertext, so a swapped salt yields a decrypt failure rather than
  a silent wrong answer — but the envelope metadata is not bound to the plaintext.
- **No key wrapping / envelope encryption.** There is no DEK/KEK split; the PBKDF2 output is used
  directly as the content-encryption key. Because each secret uses its own salt, **100,000 PBKDF2
  iterations are paid on every single get and every single put.**
- **PBKDF2-SHA256 at 100k is dated** for a passphrase-derived key. It is defensible here only
  because the key material is a high-entropy file (an SSH private key or a generated 244-bit
  string), not a human passphrase — so the KDF is doing much less work than usual. Worth naming as
  a decision rather than a default.

Tag length is the WebCrypto default of 128 bits.

### 9c.2 A second, weaker crypto path

`ControlPlaneVaultProvider` (`src/domain/vaults/control_plane_vault_provider.ts`, using
`src/domain/crypto/aes_gcm.ts`) has **no KDF at all**. A 256-bit AES-GCM key is generated by
`crypto.subtle.generateKey(..., /* extractable */ true, ...)` (`aes_gcm.ts:65-71`) and stored **raw
and unencrypted** in the control-plane store at `token-secrets/encryption-key`, with the secrets
next to it at `token-secrets/values/<key>` (`control_plane_vault_provider.ts:41,101-130`).

This is the vault that holds `swamp serve` bearer tokens. The key sits in plaintext beside the
ciphertext, which makes the encryption an obfuscation layer rather than a control. It was
introduced recently — commit `0dfdbef feat(serve): move per-login token secrets from vault to
encrypted control-plane store` — and hardened twice in the following commits.

### 9c.3 Key management

Exactly two sources of key material (`local_encryption_vault_provider.ts:460-574`):

1. **An SSH private key file** — `config.ssh_key_path`, defaulting to `~/.ssh/id_rsa` when neither
   `ssh_key_path` nor `auto_generate` is set (`:479`). The PEM is base64-decoded and the raw
   DER/OpenSSH bytes are imported as PBKDF2 key material (`:601-607`). **Passphrase-protected keys
   are rejected**, both legacy PEM (`Proc-Type: 4,ENCRYPTED`) and OpenSSH (`openssh-key-v1\0` with
   a cipher name other than `"none"`) (`:651-695`). Permissions are validated —
   `checkFileNotBroadlyReadable` → POSIX `(stat.mode & 0o077) !== 0`, and on Windows an `icacls`
   ACE scan (`src/infrastructure/security/file_security_check.ts:57-104`).
2. **An auto-generated key** (`auto_generate: true`), `:517`:

   ```typescript
   const generatedKey = crypto.randomUUID() + crypto.randomUUID(); // 72 chars
   ```

   Two concatenated UUIDv4s — about 122 bits each, 244 bits total — written as ASCII to
   `{vaultDir}/.key` with `O_CREAT|O_EXCL` and `mode: 0o600` (`:523-527`), with a 20×5ms read-back
   retry for the loser of a creation race (`:548-552`).

**No passphrase prompt. No OS keychain. No env var. No age/GPG. No cloud KMS.** These do not exist.

**No key rotation.** Grepping `rekey|re-encrypt|reencrypt|key.rotation|rotateKey|escrow` across
`src/` and `design/` returns only unrelated hits. There is no `swamp vault rotate-key`, no
re-encrypt-all, no key versioning. Changing `ssh_key_path` or losing `.key` **permanently destroys
every secret in that vault.**

**No recovery path.** No escrow, no recovery key, no export or backup command. `.key` lives under
`.swamp/`, which is gitignored, so the only copy of the key-encryption key is on one machine and is
not in version control. The nearest thing to rotation is `swamp vault migrate --to-type`, which
copies plaintext to a *different* backend and is explicitly rejected for same-type migrations
(`design/vaults.md:835`).

**And the `.key` file's permissions are never validated on read** — only SSH keys go through
`checkFileNotBroadlyReadable` (`:588`). A `.key` chmod'd to 0644 out of band is used silently.

### 9c.4 On-disk format

`src/domain/vaults/local_encryption_vault_provider.ts:52-64`:

```typescript
/**
 * Encrypted data format stored in files.
 */
interface EncryptedData {
  /** Base64-encoded initialization vector */
  iv: string;
  /** Base64-encoded encrypted data */
  data: string;
  /** Salt used for key derivation (base64) */
  salt: string;
  /** Format version for future compatibility */
  version: number;
}
```

Written as `JSON.stringify(encryptedData, null, 2)` via `atomicWriteTextFile(..., { mode: 0o600 })`
(`:139-143`). `data` is ciphertext with the 16-byte GCM tag appended. `version` is always `2`, and
**nothing reads or branches on it** — the "for future compatibility" field is inert.

Layout (`design/vaults.md:26-36,168-175`):

```
vaults/
  {vault-type}/
    {id}.yaml                        # Vault configuration (tracked in git)

.swamp/secrets/
  {vault-type}/
    {vault-name}/
      .key                           # Encryption key (auto_generate mode)
      {secret-key}.enc               # Encrypted secret files
      .annotations/{secret-key}.enc  # encrypted annotation metadata
      .refresh/{secret-key}.enc      # encrypted refresh hook config
```

Directories are `0o700`. Every write goes through `assertSafePath(path, this.secretsBoundary)` and
`validateSecretKey` rejects `..`, `/`, `\`, `\0` (`:752-763`). Annotations and refresh hooks use the
identical envelope and the same PBKDF2/AES-GCM path.

Note that the vault **config** YAML is written with `atomicWriteTextFile(path, content)` — **no
mode argument** (`yaml_vault_config_repository.ts:266-268`) — so it lands at the default umask. It
contains no secrets, but the asymmetry with the deliberate `0o600` elsewhere is notable.

### 9c.5 Refresh hooks

Declared per secret, at put time:

```bash
swamp vault put my-vault GCP_TOKEN --refresh-from "gcloud auth print-access-token" --refresh-ttl 50m
swamp vault put my-vault GCP_TOKEN --clear-refresh
```
— `src/cli/commands/vault_put.ts:158-161,183-194`

The value object (`src/domain/vaults/refresh_hook.ts:32-37`):

```typescript
export interface RefreshHookData {
  command: string;
  ttlMs: number;
  ttl: string;
  lastRefreshedAt: string | null;
}
```

**Trigger is TTL only, evaluated lazily on read.** There is no scheduler, no daemon, no expiry field
on the secret, and **no on-failure retry** (`refresh_hook.ts:75-78`):

```typescript
isStale(now: number = Date.now()): boolean {
  if (this.lastRefreshedAt === null) return true;
  return (now - this.lastRefreshedAt.getTime()) >= this.ttlMs;
}
```

A freshly created hook has `lastRefreshedAt: null`, so it is stale on first read.

Execution (`src/domain/vaults/vault_service.ts:228-262`): if stale, run the command, and on success
`provider.put(secretKey, freshValue)` plus `provider.putRefreshHook(secretKey,
hook.withRefreshedAt(new Date()))`. For `local_encryption` that is a full re-encrypt with a new salt
and IV.

The command runs **through a shell** — `sh -c <command>` on POSIX, `cmd /c <command>` on Windows,
with a 30-second timeout (`src/infrastructure/vaults/vault_refresh.ts:26-34`).

**Refresh is fail-soft**: failure, empty output, or a thrown error all log a warning and return the
**stale value** (`vault_service.ts:236-260`). Defensible — an expired token that fails closed would
break every workflow — but it means a silently-broken refresh command degrades to "quietly using an
expired credential" rather than a hard error.

**Activation is conditional and easy to miss.** `refreshOptions` is wired at only two call sites:
`src/libswamp/vaults/read_secret.ts:76` and `src/domain/expressions/model_resolver.ts:494-498`.
Every other `VaultService.fromRepository()` omits it, so **refresh hooks do not fire** for
`vault inspect`, `vault migrate`, token verification, or serve/WebSocket reads — those silently
return the stale stored value. **[inferred from an exhaustive grep for `createVaultRefreshOptions`.]**

### 9c.6 Access control does not cover secrets

The access-control resource kinds are `["workflow", "model", "data", "access"]`
(`src/domain/access/resource_selector.ts:22-27`). `resource_selector_test.ts:63-70` explicitly
asserts that `parseResourceSelector("secret:my-secret")` **throws**. Actions are
`["run", "read", "write", "admin"]`, non-hierarchical.

Enforcement exists at exactly three places, **all under `swamp serve`**:
`src/serve/handlers/shared.ts:268` (`authorizeOrReject`), `src/cli/commands/serve.ts:2625`, and
`src/serve/device_auth_handler.ts:207`. Default mode is `"none"`
(`src/domain/access/serve_auth_config.ts:86`; `shared.ts:243` returns `true` immediately).

**Locally there is no enforcement at all.** `VaultService.get(vaultName, secretKey, callerContext?)`
(`vault_service.ts:221-225`) takes **no principal** and performs no check. There is no `secrets:` or
`permissions:` block on workflows, jobs, or steps — verified against `step.ts:79-96`, `job.ts:64-70`,
`workflow.ts:36-74`. Secret resolution is a **regex scan over arbitrary strings**
(`src/domain/expressions/model_resolver.ts:1097-1121`): whatever `vault.get(...)` the author writes
is fetched. And extensions hold the raw `VaultService` on their `MethodContext`.

Under `serve`, vault handlers are shoehorned into `kind: "data"`, and most authorize against the
**literal string `"vault"`** rather than the vault name — `handleVaultGet`, `handleVaultListKeys`,
`handleVaultInspect`, `handleVaultSearch` all use `{ kind: "data", name: "vault" }`
(`src/serve/handlers/vault_handlers.ts:113-117,467-471,527-531,579-583`). Only
`handleVaultReadSecret` is per-vault (`:875-881`), and the `key` field it passes **cannot be
referenced by any grant condition**: the sealed CEL environment declares only
`["name","ns","tags","owner"]` for `data` with `unlistedVariablesAreDyn: false`
(`src/infrastructure/cel/grant_condition_environment.ts:39-44,244`). **[inferred]** per-key scoping
is therefore structurally impossible.

**Concrete consequence, traced hop by hop but not executed [inferred]:** server bearer tokens are
stored as plaintext in a vault under the well-known key `server-token-<name>`
(`src/domain/models/access/server_token_model.ts:68-72,114-118`). So a grant of `read` on `data:*` —
the natural "developers can read data" grant — also yields every server token; and a local workflow
step containing `${{ vault.get('local', 'server-token-admin-1') }}` exfiltrates it with no check
whatsoever.

### 9c.7 The audit trail

**Two unrelated subsystems share the `.swamp/audit/` directory.** They share no code, no schema, and
no interface.

**(a) Agent-session recording** — `src/domain/audit/`. Despite the name, this is not a security
audit; it records the bash/tool-use commands the *AI coding agent* runs
(`design/audit.md:1-7`). Schema (`src/domain/audit/audit_command_entry.ts:25-34`):

```typescript
export interface BashCommandEntry {
  readonly timestamp: string;
  readonly sessionId?: string;
  readonly command: string;
  readonly cwd: string;
  readonly exitCode?: number;
  readonly error?: string;
}
```

At `.swamp/audit/commands-YYYY-MM-DD.jsonl`. **Full command strings are persisted unredacted** — an
agent that types a secret inline lands it here. Retention is 7 days, triggered only from the hook
and **not awaited** (`src/cli/commands/audit.ts:158`).

This is a genuinely interesting idea and worth flagging for hyper: it captures *what the AI actually
did*, which is a different and complementary record to *what the AI wrote*. The map's oversight
model (review the definition, review the diff) has no equivalent of this third stream.

**(b) Vault read audit** — `src/domain/vaults/vault_audit_entry.ts:20-26`:

```typescript
export interface VaultAuditEntry {
  readonly timestamp: string;
  readonly vaultName: string;
  readonly vaultType: string;
  readonly secretKey: string;
  readonly callerContext: string;
}
```

At `.swamp/audit/vault-reads-YYYY-MM-DD.jsonl`. One `write()` per line on an `O_APPEND` fd, opened
and closed per entry, **no fsync, no locking, no file mode**
(`src/infrastructure/persistence/jsonl_vault_audit_repository.ts:43-63`).

Its weaknesses, all confirmed in code:

- **Off by default**, opt-in per vault via `auditReads` (`vault_config.ts:87`).
- **Fail-open** — an append failure is downgraded to a warning and the read still succeeds
  (`vault_service.ts:278,288-291`).
- **Only `get` is audited.** `put`, `delete`, `list`, and every annotation/refresh-hook mutation
  emit nothing (`vault_service.ts:322-398,435-458`). So `swamp vault put`, `swamp vault delete`, and
  `swamp vault list-keys` — an unaudited enumeration of every secret name — leave no trace.
- **Granularity is useless for the stated purpose.** Every CEL/workflow/model-driven read records
  the identical literal `"expression:vault-resolve"`
  (`src/domain/expressions/model_resolver.ts:1120`), so the trail cannot say which run, step, or
  model read the secret. `design/vaults.md:315` states the goal as *"proving which automation read
  which secret, when"*. It does not achieve it.
- No principal identity, no PID/host/session, no success flag, no run correlation.

### 9c.8 Tamper-evidence: none

Grepping `prev_hash|prevHash|hmac|signature|merkle|checksum|integrity|append-only|tamper` across the
repo:

- **No hash chain, no sequence numbers, no signatures, no HMAC.** Neither entry type has a `hash`,
  `prevHash`, `seq`, or `id` field. Records are fully independent.
- The only HMAC in the codebase is inbound webhook verification
  (`src/serve/webhook_verifiers.ts:87,102`). The single `crypto` reference in the whole audit
  subtree is `crypto.randomUUID().slice(0,8)` — a test nonce.
- "Append-only" means **only `O_APPEND`**. No `chattr +a`, no WORM, no external sink. The files sit
  in gitignored `.swamp/` at the default umask and can be truncated or rewritten with a text editor.
- Readers **silently skip malformed lines** (`jsonl_audit_repository.ts:118-120`,
  `jsonl_vault_audit_repository.ts:108-110`), so partial tampering is indistinguishable from normal
  operation.
- `deleteOlderThan` is part of the `AuditRepository` interface itself
  (`src/domain/audit/audit_repository.ts:44`) — deletion is a first-class capability.
- **Anti-forensic footgun:** `DIAGNOSTIC_COMMAND_PREFIX = "echo swamp-doctor-smoke-test"`
  (`audit_service.ts:110`) is filtered out of the default `swamp audit` view by raw string prefix
  (`:149-154`). `design/audit.md:66` says *"User shell invocations must not start with that
  prefix"* — an unenforced convention. Any command so prefixed is invisible by default.
- **[inferred]** Neither JSONL writer passes a `mode`, while `auth_repository.ts:147` and
  `server_credential_repository.ts:98` deliberately use `{ mode: 0o600 }`. So audit files are likely
  group/world-readable under a typical 022 umask — and they contain the name of every secret read.

**Retention:** the vault read log is **never pruned**. `VaultAuditRepository` declares only `append`
and `findByTimeRange` (`src/domain/vaults/vault_audit_repository.ts:28-36`); there is no delete at
any layer, and the command-log cleanup cannot touch it because its filter requires
`entry.name.startsWith("commands-")` (`jsonl_audit_repository.ts:157`). `MAX_RANGE_DAYS = 365` caps
only the query window. Unbounded growth for the life of the repo, and no command to prune, export,
verify, or rotate either trail.

### 9c.9 The sentinel mechanism, in full

This is the part of the vault subsystem that is genuinely good, and it is worth stating precisely.

`VaultSecretBag` (`src/domain/vaults/vault_secret_bag.ts`) replaces resolved secrets with
`__SWAMP_VSEC_<8hex>_<n>__` sentinel tokens (`:61,72-76`), so raw values never enter persisted
definitions. For shell commands, `resolveForShell` (`:150-168`) swaps sentinels for
`${__SWAMP_VAULT_N}` **environment-variable references** — quoting-context-aware — so secrets are
passed via the process environment and never appear in the command string or the process argument
list (where any other user on the box could read them from `/proc`). There is a PowerShell sibling
at `:195-213`, and single-quoted sentinels are detected and warned about, since `'…'` blocks
expansion and would leak the literal placeholder (`design/vaults.md:467-486`).

That is careful, correct work, and it is the design idea most worth carrying into hyper.

The redactor that backs it (`src/domain/secrets/secret_redactor.ts:31-53`) is a plain
registered-string replacer — longest-first, values under 3 characters ignored, JSON-escaped variants
auto-registered. Its limits are inherent: exact substring only, so a secret that is base64'd,
URL-encoded, split across log lines, or partially printed is not caught. And `design/vaults.md:603`
states plainly that the audit log is **"Not covered"** by redaction in either direction.

### 9c.10 What could not be determined about vaults

- No literal `.enc` fixture exists in-tree; tests assert the shape and generate files at runtime.
- Whether `EncryptedData.version` was ever `1`. There is no v1 branch and no migration code for this
  envelope. **[inferred]** v1 files would decrypt fine since the field is unread — unverified.
- **Extension vault providers** (`@swamp/aws-sm`, `@swamp/azure-kv`, `@swamp/1password`) are not in
  this repo — only the type names in `RENAMED_VAULT_TYPES`. Their crypto, key management, and
  whether they implement `VaultRefreshHookProvider` are outside this tree.
- Whether any in-tree extension actually calls `context.vaultService.get()` directly. The type and
  wiring permit it; no in-tree example was found.
- Concurrent-append atomicity of the JSONL writers. A single short `write()` on an `O_APPEND` fd is
  atomic in practice on POSIX, but nothing in the code guarantees it, there is no locking, and there
  is no fsync — a crash loses the tail. **[inferred.]**

---

## 10. The extension model

### 10.1 How an extension gets into a repo

Four routes:

| Route | Declared in | Code |
| --- | --- | --- |
| Registry pull | `swamp extension pull @c/n[@ver]` | `src/libswamp/extensions/pull.ts` |
| Lockfile restore | `upstream_extensions.json` (committed) | `src/libswamp/extensions/install.ts:165-215` |
| Auto-resolve on first use | `trustedCollectives` in `.swamp.yaml` | `src/domain/extensions/extension_auto_resolver.ts` |
| Local source dir | `.swamp-sources.yaml` | `src/domain/repo/swamp_sources.ts:47-101` |

Trust default is `["swamp"]` only, and collectives the user *belongs to* are not auto-trusted
unless `trustMemberCollectives` is set (`design/extension.md:818-846`). Auto-resolve prefers the
lockfile-pinned version and checksum when one exists (`design/extension.md:865-885`). Those are
sensible defaults.

Install (`installExtension()`, `src/libswamp/extensions/pull.ts:692-1200`):

1. Fetch metadata and archive from the registry (`GET /api/v1/extensions/{name}@{v}/download`).
2. SHA-256 the archive, compare against the registry-served checksum. **If the registry returns no
   checksum, `integrityStatus = "unverified"` and the install proceeds anyway**
   (`pull.ts:752-757`).
3. If a lockfile checksum exists, byte-exact compare or hard `UserError` (`pull.ts:765-779`).
4. Tar entry pre-scan rejects `..` and leading `/` (`pull.ts:800-808`); after extraction,
   `validateNoSymlinkEscape` (`pull.ts:818-825`).
5. `analyzeExtensionSafety(tsFiles)` over the `.ts` sources (`pull.ts:849-886`).
6. Files land in `.swamp/pulled-extensions/<@c/n>/`; **prebuilt `bundles/*.js` are copied to
   `.swamp/bundles/<bundleNamespace>/`** (`pull.ts:466-600`).

### 10.2 How it is loaded and run

`src/cli/mod.ts:590-670` builds an `ExtensionLoader` and registers a lazy type loader on the global
model registry. Execution is a bare dynamic `import()` into the host isolate —
`src/domain/extensions/extension_loader.ts:780-806`:

```typescript
const baseUrl = toFileUrl(paths.bundlePath).href;
…
return await import(importUrl);
```

with a fallback (`extension_loader.ts:1229-1276`):

```typescript
return await import(`data:application/javascript;base64,${encoded}`);
```

Before import, the JS is rewritten in place (`rewriteZodImports`, `fixCjsEsmInterop`) so that zod
resolves to `globalThis.__swamp_zod` — extensions must share the host's zod instance for
`instanceof` schema checks to work.

**Pulled extensions are never rebundled locally.** *"They rely entirely on pre-built bundles
shipped with the registry package"* (`design/extension.md:1291-1296`). Local `.ts` sources do get
bundled, by shelling out to `deno bundle` via `Deno.Command` (`src/domain/models/bundle.ts:583-673`).

### 10.3 The authoring contract

There is **no SDK to import**. `src/libswamp/` is *not* the extension SDK — it is the internal
application layer between presentation and domain (`design/libswamp.md:1-29`). Nothing under
`extensions/` imports it.

The contract is structural duck-typing validated by zod at load time
(`src/domain/extensions/model_kind_adapter.ts:140-164`):

```typescript
const UserModelSchema = z.object({
  type: z.string(),
  version: z.string().refine(CalVer.isValid, …),
  globalArguments: z.custom<z.ZodTypeAny>(isZodSchemaLike).optional(),
  resources: z.record(z.string(), ResourceOutputSpecSchema).optional(),
  files: z.record(z.string(), FileOutputSpecSchema).optional(),
  methods: z.record(z.string(), UserMethodSchema),
  checks: z.record(z.string(), UserCheckSchema).optional(),
  upgrades: z.array(UserUpgradeSchema).optional(),
  reports: z.array(z.string()).optional(),
})
```

The loader looks for a named export per kind: `model` / `extension`, `vault`, `datastore`,
`report` (`model_kind_adapter.ts:394-396`).

A real extension, from `extensions/models/issue_lifecycle.ts`:

```typescript
import { z } from "zod";
import { …, GlobalArgsSchema, … } from "./_lib/schemas.ts";
import { createSwampClubClient, loadAuthFile } from "./_lib/swamp_club.ts";

export const model = {
  type: "@swamp/issue-lifecycle",
  version: "2026.07.30.1",
  globalArguments: GlobalArgsSchema,
  …
  methods: {
    start: {
      description: "Ensure the swamp-club issue exists and begin the lifecycle",
      arguments: z.object({}),
      execute: async (_args, context) => {
        const { issueNumber } = context.globalArgs;
        const sc = await createSwampClubClient(context.globalArgs, context.logger);
        const issue = await sc.fetchIssue();
        …
        await context.writeResource("context", "context-main", { … });
```

Versions are CalVer (`2026.07.30.1`). The manifest (`manifestVersion: 1`, scoped `name`, CalVer
`version`, at least one of `models|workflows|vaults|drivers|datastores|reports|skills`, plus
`dependencies`, `binaries`, `include`, `additionalFiles`) is specified in
`design/extension.md:267-360` and `src/domain/extensions/extension_manifest.ts`.

**The manifest has no permissions or capabilities field.** An extension declares no intent about
filesystem, network, environment, or subprocess access.

### 10.4 Sandboxing: there is none

This is the blunt finding, and it is well evidenced.

> "Extension model methods run **in-process in the host Deno process** (via `InProcessExecutor`).
> They **share the process-level permissions baked into the compiled binary**."
> — `design/extension.md:1026-1032`

Corroborating greps over the whole tree:

- `new Worker(` — **zero hits** repo-wide. No worker isolate, so no per-worker
  `deno.permissions` descriptor.
- `Deno.permissions` / `PermissionOptions` — **zero hits**. No runtime permission query, request,
  or revocation.
- No `--deny-*` flag anywhere; no permission prompts (the compiled binary is pre-granted).

So: **Swamp does not use Deno's permission model to constrain extensions.** It uses it exactly once,
at the process boundary, to grant the whole binary everything except FFI (see §9). Anything loaded
into that process inherits the lot. `--allow-run` is unscoped, which independently defeats any
narrower grant that might have been applied — an extension can just spawn a subprocess.

The out-of-process paths that exist are **placement, not isolation**. The remote worker subprocess
is spawned with the same full flag set (`src/worker/dispatch_handler.ts:366-378`):

```
"run","--unstable-bundle","--allow-read","--allow-write","--allow-env",
"--allow-run","--allow-net","--allow-sys"
```

And the now-removed Docker driver ran `deno run --allow-all /swamp/runner.js`
(`design/execution-drivers.md:186`). The design doc is explicit that *"Isolation is a worker
deployment property (run a containerized worker)"* (`design/execution-drivers.md:8-10`) — i.e.
isolation is the operator's problem, not the tool's.

What an extension can reach, concretely:

- **Everything Deno can**: `Deno.readTextFile`, `Deno.writeFile`, `Deno.env`, `Deno.Command`,
  `fetch`. The design doc's own examples show extensions reading arbitrary files
  (`design/extension.md:630-644`) and spawning `cat`/`dd`/`stty` against `/dev/ttyUSB0`
  (`design/extension.md:1055-1077`).
- **The injected `MethodContext`** (`src/domain/models/model.ts:147-430`): `dataRepository` (the
  whole datastore), `definitionRepository`, `outputRepository`, `queryData` (CEL across *all*
  models), `readModelData`, `redactor`, `createCelEnvironment`, `runModel`,
  `approveWorkflowGate` / `rejectWorkflowGate`, and — note — **`vaultService?: VaultService`**
  (`model.ts:212`), a live handle with `get/put/delete/list/getAnnotation/putRefreshHook`
  (`src/domain/vaults/vault_service.ts:221-452`), plus `vaultSecrets: VaultSecretBag`
  (`model.ts:351`).

None of these are scoped per-extension except `runModel`. That one check —
`ModelInvocationService.#checkAuthorization`, `src/domain/models/model_invocation_service.ts:423-479`
— requires the caller's `manifest.yaml` to list the target in `dependencies`, fails closed if the
manifest is unreadable, and is bypassed for same-extension and same-collective calls. An extension
that wants the effect anyway can `import`, `fetch`, or `Deno.Command` around it.

### 10.5 The safety analyzer is advisory, and scans the wrong artifact

`analyzeExtensionSafety` (`src/domain/extensions/extension_safety_analyzer.ts:128-290`) runs at pull
time.

Hard errors: hidden files, file extensions outside `.ts .json .md .yaml .yml .txt`, symlinks,
>1 MB per file, >10 MB total, >150 files, and `content.includes("eval(")` /
`content.includes("new Function(")`.

Warnings only: 500+ character lines, base64 blobs, and — critically —
`content.includes("Deno.Command(")` (`:245-250`). Spawning arbitrary subprocesses is a *warning*.

Two structural problems:

1. **The warnings never prompt — despite the design doc saying they do.**
   `design/extension.md:998` heads the list "### Warnings (prompt user)". But
   `src/cli/commands/extension_pull.ts` contains no confirmation on
   safety findings; the renderer logs them *after* the install completes
   (`src/presentation/renderers/extension_pull.ts:60-71`) — including the line
   `"This extension includes executable binaries — inspect before use:"`. The only
   `promptConfirmation` in the pull path is for file-overwrite conflicts (`extension_pull.ts:174`).
2. **It scans `.ts` sources; the `.js` bundle is what executes.** Pull scans only
   `models/ vaults/ drivers/ datastores/ reports/` `*.ts` (`pull.ts:850-870`), while the artifact
   that gets `import()`ed is the registry-supplied prebuilt `bundles/*.js`. Nothing verifies the
   bundle was built from the shipped source. **[inferred]** — a publisher could ship benign
   TypeScript and a divergent bundle, and nothing in the pull or load path would notice. This
   could not be ruled out server-side (the registry is a separate, non-public repo), so treat it as
   unverified rather than confirmed.

The `binaries:` manifest field additionally lets an extension ship arbitrary non-allowlisted
executables, with exec bits restored on POSIX (`pull.ts:1046-1050`,
`design/extension.md:1007-1017`).

### 10.6 `npm:` specifiers: allowed, first-class, inlined at publish time

> "Each model entry point is compiled using `deno bundle` with zod externalized. All other
> non-local specifiers (`npm:`, `jsr:`, `https:`) are resolved and **inlined into the bundle** …
> **First-class specifier kinds:** `npm:`, `jsr:`, and `https:` are peers — `deno bundle` resolves
> all three natively with identical treatment."
> — `design/extension.md:472-486`

Author guidance mandates the inline form —
`.claude/skills/swamp/references/extension-publish/references/publishing.md:230-263` gives
`import { z } from "npm:zod@4";` as canonical and lists `npm:lodash-es@4.17.21`,
`jsr:@std/assert@1.0.0`, and `https://deno.land/std@0.224.0/async/delay.ts` as all supported.

Consequences worth stating plainly:

- **No allowlist, no denylist, no host restriction on imports.** The only import rules enforced are
  that dynamic `import()` is rejected in extension source
  (`src/domain/extensions/extension_quality_checker.ts:262-277`) and that local imports must stay
  inside the kind directory (`src/domain/extensions/extension_import_resolver.ts`).
- **Extension dependencies never enter `deno.lock`.** They are inlined on the *author's* machine at
  push time. The consumer's lockfile pins the extension archive, not its transitive npm tree.
- **Version pinning is advisory** (`design/extension.md:488-493`). An unpinned `npm:foo` resolves
  to whatever was latest when the author pushed.
- No `--allow-scripts` anywhere, so npm lifecycle scripts do not execute during bundling.

### 10.7 Verification and integrity

| Mechanism | Where | Strength |
| --- | --- | --- |
| SHA-256 archive vs registry checksum | `pull.ts:749-757` | Same-origin — registry serves both bytes and hash. Catches transport corruption, not a malicious or compromised registry. Missing checksum ⇒ installs anyway as `"unverified"`. |
| Lockfile checksum anchor | `pull.ts:765-779`, `install.ts:207-215` | Real TOFU pinning, but only on *restore* flows. An explicit `extension pull` is the documented way to accept drifted bytes. |
| `filesChecksum` on-disk digest | `src/domain/extensions/installed_extension_digest.ts:45-54` | Covers `.swamp/pulled-extensions/` only — **not** `.swamp/bundles/`, where the executed JS lives. Not checked at import time. |
| Tar path / symlink escape guards | `pull.ts:800-825` | Properly enforced. |
| npm/jsr dependency trust audit (OSV.dev, deprecation, license, downloads, maintainers) | `src/domain/extensions/extension_dependency_trust_checker.ts:403` | **Push-time only, on the publisher's machine** (`push.ts:679`). Never re-run on pull. |
| Code signing / provenance | — | **Absent.** Grepping `signature\|sigstore\|gpg\|provenance\|ed25519\|minisign` across `src/` and `design/` returns only HMAC webhook verifiers. |
| Registry auth | `design/extension.md:1121-1126` | Push needs a Bearer key scoped to your collective. **Pull is unauthenticated.** |

Registry protocol: push is 3-phase (`POST /api/v1/extensions/push` → presigned S3 `PUT` →
`POST /api/v1/extensions/confirm`); pull is `GET /api/v1/extensions/{name}` → `…/download` (302) →
`…/checksum` (`design/extension.md:1128-1145`). Because the upload is a presigned S3 `PUT`, all
client-side push gating (quality rubric, dependency trust audit) is bypassable by a hostile
publisher writing to the URL directly — unless the server re-verifies, which could not be
determined from the public repo.

### 10.7b One decision here is exemplary

`design/extension.md:243-266`, "Why no passive on-load warning", is the best-argued section in the
repo. A feature request asked for a per-extension staleness warning on every command. Instead of
shipping it, they surveyed how Terraform, OpenTofu, Ansible, Pulumi, and `gh` handled the same
question, cited specific issue numbers and the pain each caused, and declined:

> "No surveyed tool has shipped a generally-loved per-extension passive warning. gh's extension
> version checker is the closest attempt and is the documented cautionary tale (issue #10235:
> blocking PostRun call hangs commands for minutes)."

They then documented the design they *would* ship if demand appeared — "an end-of-command,
single-line, aggregated, info-level (not warn) advisory with 24h display cooldown, TTY gating,
env-var suppression (`SWAMP_NO_UPDATE_NOTIFIER`), and non-blocking time-budgeted registry calls" —
and closed with "Per-extension warnings at the start of every command are explicitly out."

Deciding *not* to build, in writing, with evidence and a documented escape hatch, is a practice
worth copying independent of anything else in Swamp.

### 10.8 What could not be determined about extensions

- **Server-side registry behaviour.** Push-time rejection, the rubric scorer, and whether the
  registry re-bundles or re-verifies uploads live in a non-public repo.
- **Whether the shipped `bundles/*.js` is ever cross-checked against the shipped `.ts`.** No
  client-side check exists; a server-side one cannot be ruled out.
- **`SWAMP_EXTENSION_REVIEW_DIR`** (`design/extension.md:764-788`) — described as a CI review
  surface, not traced to implementation.
- **Prompt-injection surface of skills.** Skills are explicitly non-executable — *"Skills are
  passive markdown guidance documents — swamp never executes them"* (`design/extension.md:311-313`)
  — but they are installed into `.claude/skills/` and *are* read by coding agents. Not evaluated.

---

## 11. Synthesis

### 11.1 Deliberate and non-obvious decisions worth taking seriously

**1. The sealed grant-condition CEL dialect.** Three CEL surfaces, of which the one used for
authorization decisions is strictly weaker: explicit variable declarations,
`unlistedVariablesAreDyn: false` so undeclared references fail at *write* time, no I/O receivers, no
extension registrations. And the seal is documented as **permanent**
(`design/expressions.md:29-38`). Separating "the expression language users author in" from "the
expression language security decisions are made in", and refusing to let them converge, is the best
idea in the repo.

**2. Vault sentinels.** Secrets are opaque `__SWAMP_VSEC_<hex>_<n>__` tokens through authoring,
persistence, and expression evaluation, materialised only in the in-flight request. For shell
execution they become `${__SWAMP_VAULT_N}` environment references — quoting-context-aware — so they
never appear in a command string or a process argument list
(`src/domain/vaults/vault_secret_bag.ts:150-168`). The persisted artifact is always safe to read,
commit, and review.

**3. Sensitive output routes to a vault at write time, and fails closed.** A `z.string().meta({
sensitive: true })` on a model's output schema means the value is `put` into a vault and the data
record gets `${{ vault.get(...) }}` in its place (`data_writer.ts:381-474`). Three enforcement
points, including a **pre-flight check before the method runs**, added specifically because the
old behaviour *"fails at persist time — but only after the method has already run, potentially
creating cloud resources that cannot be recorded"* (`design/doctor-vaults.md:14-16`).

**4. Files authoritative, SQLite derived.** The index can be dropped and rebuilt
(`CATALOG_SCHEMA_VERSION`, drop-and-backfill at `catalog_store.ts:217-232`), which eliminates the
entire category of index-migration bugs. Any hyper design with a database should ask whether the
database can be deleted without data loss.

**5. Unknown keys are rejected, not stripped** (`design/workflow.md:364-389`). With the failure
story recorded:

> "A step-level placement property (`labels`, `target`, `platform`, `queueTimeout`) found on a job
> or workflow names the misplaced key and shows the step-level form. **Silent stripping here failed
> open: the placement intent was discarded and the work ran on the orchestrator.**"

Plus a Levenshtein did-you-mean, and a stated schema-evolution consequence: because suspended runs
are re-parsed through the same schemas on resume, removing a field is a breaking change requiring
*"an explicit migration or tolerance decision … never a silent strip"*. For AI-authored YAML this is
the correct default and the reasoning is fully worked out.

**6. Deletion tombstones that keep `data.latest()` resolvable.** After a successful `delete`-kind
method, a new version is written containing `{...lastKnownState, deletedAt, deletedByMethod}`
(`method_execution_service.ts:949-1017`), so re-runs stay idempotent and the record of what was
destroyed survives. A genuinely good answer to "how does an immutable log represent a thing that no
longer exists".

**7. The only channel between steps is durable data.** No `steps.*` namespace, no ephemeral output
bus. This is what makes resume-from-disk correct (the expression context is rebuilt from the store,
`:2090`) and what makes every intermediate value queryable afterwards.

**8. `guard` instead of implicit idempotence.** A per-step CEL predicate that may probe live
infrastructure via `model.method(...)`, with an explicit refusal to be clever:
*"Steps without a guard always execute on resume. If you didn't write a guard, you're saying 'always
run this step.'"* (`design/workflow.md:206-208`).

**9. Structure is data; only values are expressions.** `dependsOn` is a structural
`TriggerCondition` DSL, not a CEL expression, with the *reference* on the edge and the *predicate*
in the condition (`trigger_condition.ts:61-62`). The DAG stays statically analysable even though
values are not.

**10. Refusing to build, in writing.** `design/extension.md:243-266` declines a requested feature
after surveying Terraform, OpenTofu, Ansible, Pulumi and `gh`, cites the specific issues each caused,
documents the design it *would* ship if demand appeared, and closes the question. See §10.7b.

**11. Run-tracker DB is deliberately not synced.** *"PIDs and heartbeats are inherently local — a PID
from machine A is meaningless on machine B"* (`design/run-tracker.md:91-93`). Knowing which state is
machine-local and refusing to replicate it is a discipline worth keeping.

**12. `swamp audit` records what the AI agent did.** Not swamp operations — the bash/tool-use
commands the coding agent ran, captured via each tool's hook system (`design/audit.md:1-7`). This is
a third oversight stream that the map's model (review the definition; review the diff) does not
have. Worth considering on its merits, separately from the fact that Swamp's implementation of it
is unredacted and untrustworthy.

### 11.2 Scar tissue, in one place

| Evidence | What it says |
| --- | --- |
| `design/unification.md` (whole file) | Three storage shapes create enough pain that the maintainers are contemplating collapsing filesystem + datastore + registry into one versioned store, with git as a possible backend. |
| `design/execution-drivers.md:1-12` | An entire pluggable driver abstraction — registry, 12-field config, 6-tier precedence, docker bundling — built, then deleted. Replaced by "run a containerized worker". |
| `definition.ts:192-195`, `repo_marker_repository.ts:84-97` | The removed `driver`/`driverConfig` fields must still be *actively rejected* in two schemas. |
| `design/run-tracker.md:6-12` | A SQLite DB with PIDs, heartbeats and a reaper, bolted on because status-mutable YAML files got stuck in "running" after process death. |
| `design/repo.md:134-140` + `noop_repo_index_service.ts` | Symlink views removed; a 17-event domain-event system still fires into a noop handler. Symlink *fallback* read paths remain (`libswamp/workflows/edit.ts:191`). |
| `unified_data_repository.ts:1389-1400` | `latest` is a text file now, with a `Deno.readLinkSync` fallback for legacy symlink repos. |
| `repo_marker_repository.ts:41-67` | `.swamp.yaml` carries a deprecated `tool` field kept alive by a read normalizer, plus UI-dismissal state (`skillMigrationDismissed`, `lastSkillMigrationWarning`). |
| `design/repo.md:54-70` | A `SUPERSEDED_SKILLS` constant and a startup scan warning about skill directories left by older binaries. |
| `design/inputs.md:209-264` | Four overlapping sigils on one `--input` value slot (`@file`, `\@`, `@scoped/name`, `:json`), disambiguated by a heuristic that checks for dots. |
| `design/audit.md:41-58` | Six AI-tool integrations, each with its own skills dir, instructions file, hook-config generator, and `postToolUse` payload normalizer. |
| `design/extension.md:229-232` | Bundle caching moved from mtime to content fingerprint because *"mtime-based freshness was unreliable under atomic-rename saves, mtime-preserving sync tools, and sub-millisecond edits"*. |
| `agent-constraints/adversarial-dimensions.md` + commit `9f0a184` | A review checklist written in the register of someone who has shipped canonicalization and authorize-vs-execute identity bugs. "fix(serve): close five security vulnerabilities in serve HA and vault subsystems". |
| `design/run-tracker.md:76-82` | A global `unhandledrejection` swallower installed so extension code cannot kill the server. |
| `dependency_extractor.ts:36-39` | `ArtifactDependencyTypes` — dependency inference declared and never used. |
| Doc/code divergences | Reaped runs (`cancelled` vs `failed`); safety warnings ("prompt user" vs logged after install); `range()` documented but unregistered; README symlink views; skill references showing three obsolete paths. |

The compound lesson: **the abstractions Swamp most regrets are the ones it added for flexibility it
did not yet need** — execution drivers, datastore backends, the storage-tier split, the domain-event
bus. The abstractions it does not regret are the ones that constrain (sealed grant CEL, unknown-key
rejection, write-once terminal records, sentinels).

### 11.3 What `hyper` should most carefully NOT copy

Ordered by how much damage the mistake would do.

**1. Implicit ordering — a DAG derived only from `dependsOn` while expressions read live state.**
Swamp lets you write `${{ data.latest('scanner','result') }}` in a step with no dependency edge; the
step may run concurrently with its producer and read stale or absent data, silently and
non-deterministically (§9a.2). Compounded by accessors that *"read directly from disk on every
call"* (`design/expressions.md:242-243`), so there is no snapshot isolation within a run either.
When an **AI** writes the workflow, "the author is responsible for declaring the edge they already
implied by referencing the data" is the wrong contract. hyper should either infer edges from
references, or reject a reference without an edge — but not silently permit it.

**2. Approval that suspends and exits 0.** A gated run in CI is indistinguishable from a successful
one by exit code (§9a.7). No `--fail-on-suspend`, no test asserting it. If hyper ever grows a pause
concept, the non-interactive exit-code contract must be designed first. (The map has already
declined per-run approval; this is the reason to keep declining it, and the trap to avoid if the
destructive-operations ticket reintroduces anything gate-shaped.)

**3. An extension model with no sandbox and no declared permissions.** Extensions are `import()`ed
into the host isolate, which holds every Deno permission except FFI; the manifest has no
capabilities field; the safety analyzer treats `Deno.Command(` as a *warning*, never prompts, and
scans the `.ts` sources while the prebuilt `.js` bundle is what actually runs (§10.4, §10.5). Every
extension gets a live `VaultService`. The map keeps extensions in scope — then the permission model
must be designed *before* the loader, because Swamp shows it cannot be retrofitted: the whole
process is already fully privileged.

**4. Secrets outside the access-control model.** `parseResourceSelector("secret:my-secret")` throws
by design; `VaultService.get()` takes no principal; secret resolution is a regex scan over arbitrary
strings, so whatever the author writes is fetched (§9c.6). Any hyper design where a definition can
name a secret should make secret access a declared, checkable property of the definition — not an
emergent consequence of string content.

**5. An audit trail with no tamper-evidence, off by default, that only logs reads.** No hash chain,
no signatures, `O_APPEND` only, gitignored, default umask, `deleteOlderThan` in the interface, and a
filtered-by-default diagnostic prefix that is an unenforced convention (§9c.7, §9c.8). Plus the
`callerContext` is the constant string `"expression:vault-resolve"`, so it cannot answer the
question it exists to answer. Either build a trail that is trustworthy or do not claim one.

**6. Encryption with no rotation and no recovery.** Losing `.key` or changing `ssh_key_path`
permanently destroys every secret, and there is no `rotate-key`, no escrow, no export (§9c.3). Also
avoid the second, weaker path: `ControlPlaneVaultProvider` stores its AES key in plaintext beside
the ciphertext. If hyper encrypts anything locally, rotation and recovery are day-one requirements,
not follow-ups.

**7. Provenance that records the run but not the code.** Swamp records `workflowRunId`, `jobName`,
`stepName`, `ownerRef` — but no definition version, no git SHA, no extension digest on the data
record (§9b.8). For a tool whose thesis is human review of AI-authored automation, "which revision
of the definition produced this?" is *the* question, and Swamp cannot answer it from the artifact.

**8. Pluggable-everything before there is a second implementation.** Execution drivers were built
with a registry, a 12-field config, and a six-level precedence chain, then deleted wholesale. The
datastore abstraction now spans filesystem, S3, distributed locks, sync, hydration strategies, and
namespaces — for a tool the map says will have one user and two environments. The `.swamp.yaml`
per-setting precedence chains are each hand-rolled. Prefer one implementation until a second is
actually required.

**9. Overloaded value syntax.** The `--input` grammar needs a heuristic ("starts with a letter,
contains a `/`, has no `.`") to tell a file path from a scoped identifier (§4). One sigil per
meaning, or an explicit flag.

**10. `forEach` conditions that are silently ignored.** A `dependsOn.condition` on a fan-out step
shapes the DAG but is never evaluated at runtime (§9a.4). Either honour it or reject it at parse
time.

**11. A typed model layer with an unschematised shell escape hatch.** Swamp's pitch is typed models
with Zod schemas, validated inputs and outputs. But `command/shell` accepts an arbitrary `run:`
string, pre-flight checks are skippable (`--skip-checks`), and — per the manual's own reference —
schema mismatches on write "produce a warning, not an error" (§13.7). Nothing stops an AI agent from
routing every task through `command/shell` and getting none of the guarantees. **A typed layer with
a shell escape hatch provides the safety of a shell.** If hyper wants the typed-artifact property to
mean anything, the escape hatch has to be either absent, or visibly and irreversibly marked in the
artifact so a reviewer sees it.

**12. Rollback that rewinds the record but not the effect.** The manual warns against its own
`rollbackOnFailure` feature for exactly this: rolling back the data-layer writes for three
successfully-provisioned VMs while the VMs still exist is *"worse than partial writes: it is a silent
divergence between what Swamp knows and what is actually running"* (§13.5). Any transactional
framing hyper adopts must distinguish "undo the record" from "undo the effect", and refuse the first
without the second.

**13. Retention policy frozen into each record at write time.** Changing the policy does not affect
data already written (§9b.10). Defensible, but it should be a decision, and read-time policy is
probably the better default for a single-user tool.

**12. Everything the map already declared out of scope** — quests and bingo boards, telemetry with a
`distinct_id`, `swamp issue` support funnels, `serve` with HA and OAuth, six agent-tool
integrations, an extension registry. The research confirms the call: the extension subsystem alone
(20,866 LOC) is larger than the workflow engine (14,444 LOC).

### 11.4 Two things hyper should copy the *reasoning* of, not the mechanism

- **Isolation is a deployment property, not a config field.** Swamp's conclusion after deleting its
  driver abstraction (`design/execution-drivers.md:8-10`). The mechanism (remote workers) is out of
  scope for hyper; the conclusion — do not put an isolation selector in the authored artifact —
  transfers directly.
- **Pre-flight the check that would otherwise fail after the side effect.** `doctor vaults` exists
  because a mutation succeeded and then could not be recorded (`design/doctor-vaults.md:14-16`).
  Generalised: any hyper validation that runs *after* an effectful call is a latent version of that
  bug.

---

## 12. Contradictions with the map's Notes

Recorded explicitly so the map can be corrected.

**1. "Swamp is Deno, not npm."** Half right, and the half that is wrong matters.
Swamp is Deno-hosted, but it depends on npm heavily: 24 of 36 direct imports in `deno.json:21-58`
are `npm:` specifiers (zod, react, ink, `@aws-sdk/client-cloudcontrol`, cel-js, croner, marked,
nine `@opentelemetry/*`, …), and `deno.lock` resolves **200 npm packages** to 18 JSR ones. The
extension system makes this worse rather than better: `npm:`, `jsr:` and `https:` are documented
first-class peers, **inlined into the bundle on the author's machine at push time**, so extension
dependencies never appear in the consumer's lockfile at all and version pinning is advisory
(`design/extension.md:472-493`). The npm supply-chain surface is not smaller than an equivalent Node
tool's; it is less visible.

**2. "Permissions off by default."** Wrong. `scripts/compile.ts:157-171` ships the binary with
`--allow-read --allow-write --allow-env --allow-run --allow-sys --allow-net` — every permission
category Deno currently has except `--allow-ffi` and `--allow-import`. The accompanying comment
frames "individual flags instead of `--allow-all`" as least-privilege, but that is about *future*
categories, not present ones. And Swamp uses Deno's permission model nowhere else: zero occurrences
of `new Worker(`, `Deno.permissions`, or `PermissionOptions` in the entire tree.

**3. "No lifecycle scripts."** Correct, and worth keeping. No `nodeModulesDir` is set and no task
passes `--allow-scripts`, so npm packages resolve into Deno's global cache without executing
`postinstall`. **[inferred from the absence of those settings, not from an explicit statement.]**

**4. "Likely `deno compile`d."** Confirmed — `scripts/compile.ts` runs `deno compile` with an
embedded Deno runtime and `--include .claude/skills`, and the Dockerfile just copies the resulting
binary (`FROM denoland/deno:2.7.5`).

**5. Not a contradiction, but an unrecorded divergence: Swamp has no MCP server.** Its agent
interface is skills-files-plus-CLI, across six tools. The only `modelcontextprotocol` references in
the tree are transitive deps of `promptfoo` under `evals/`. hyper's "MCP-first" decision is a
genuine departure from the prior art, not an inheritance — which is fine, but the map currently
reads as though Swamp established that ground.

**6. "Its five primitives (Models, Definitions, Workflows, Vaults, Data) are the starting
decomposition."** The five are a marketing artifact, not the model. Swamp's own bundled skill lists
a different five-plus-two (Models, Data, Workflows, Vaults, Extensions, Grants, Serve), and the code
carries Reports, Datastores, Workers and Grants as first-class kinds too. Critically, **Definition
is not a peer of Model — it is an instance of one** (§2.1). The real decomposition is three layers
(type in code / instance in YAML / orchestration in YAML) plus two stores (data, secrets).

**7. "Data layer — how versioned, immutable, provenanced artifacts are … diffed."** Swamp has **no
diff**. No `swamp data diff`, no drift detection; `design/high-level.md:20-23` self-flags drift
detection as unimplemented. Only `--since` time-window filtering exists. The "what changed since
Tuesday" half of oversight is not prior art to be adapted — hyper would be designing it from
scratch.

**8. "Swamp ships OpenTelemetry"** (listed under Observability). Confirmed, and more than the note
suggests: traces *and* native OTLP log records, with secrets redacted before export, spanning CLI →
workflow → job → step → model method, and `TRACEPARENT` propagation into extensions and containers
(`README.md:326-386`). It is opt-in via `OTEL_EXPORTER_OTLP_ENDPOINT` with zero overhead when unset.
Note the framing though: the README says its main value is *"most useful for long-running `swamp
serve` daemons"* — i.e. it earns its place in the architecture the map has explicitly declined.

**9. "Authoring language — YAML? CEL expressions like Swamp?"** Worth recording that Swamp runs
**three** CEL dialects, not one (§3b), and that its CEL has no `??` operator, cannot use `namespace`
as an identifier, and has a documented parsing limitation where an expression containing a literal
`}}` splits prematurely (`design/workflow.md:271-275`).

---

## 13. The manual and the marketing surface

The manual is at `swamp-club.com/manual`. **It is not in the public GitHub repo** — no `docs/`,
`site/`, or `www/` directory, and greps for its slugs find nothing. `design/*.md` and
`.claude/skills/swamp/references/**` are different documents.

Retrieval notes: `/llms.txt` and `/manual/reference/*.md` both 404, so there are no markdown
alternates. `sitemap.xml` lists 163 URLs, **146 under `/manual`** — of which the manual index links
only about 61, the rest being nested sub-pages. All 146 were fetched and converted (≈1.02 MB of
markdown, 1,452 fenced code blocks). Coverage of the manual is therefore complete; the quotes below
are a selection, not the whole corpus.

### 13.1 Swamp's own thesis, in its own words

From `/manual/explanation/how-swamp-works`:

> "Swamp is an adaptive automation framework designed to be operated by AI agents. It decomposes
> automation into a small set of composable primitives: models, workflows, vaults, extensions,
> skills, and a data layer — also known as The Swamp. Each primitive does one thing. What matters is
> how they fit together."

Note that this is **six primitives, and the list differs again** from both the README and the
bundled skill. "Definitions" is not among them — consistent with §2.1.

From the marketing site:

> "Deterministic automation for AI agents."
> "Your agent nails the task, then can't do it twice."
> "Instead of *performing* the work each time, it *builds* a workflow. Typed models, secrets pulled
> from a vault, every result kept. The agent decides once; Swamp *runs* it the same way every time."

And from `/manual/explanation/ai-agent-integration`, the clearest statement of the bet:

> "Swamp is operated through AI agents. Not as an optional integration or a convenience layer on top
> of a human-first interface, but as the primary design target. The CLI, the file formats, the data
> model, and the extension system were all shaped by the question: what does an AI agent need to do
> this well?"
>
> "Automation frameworks traditionally give humans a GUI or a CLI and expect them to learn the
> system's concepts, memorize commands, and wire things together manually. Swamp inverts this. The
> human describes what they want."

This is the thesis hyper shares. Worth noting where hyper's map already differs: Swamp's answer to
"what does an agent need" is *skills files + a CLI*; hyper's is MCP.

### 13.2 Configuration precedence — this closes a map open question

The map lists "Configuration precedence — Swamp layers repo/user/env/flag. Likely needed, not yet
sharp." The manual states it exactly (`/manual/explanation/configuration-layers`):

> "…that configuration comes from four sources: `.swamp.yaml` (repo-level, checked into version
> control), `swamp config set` (user-level, stored in `~/.config/swamp/`), environment variables
> prefixed with `SWAMP_`, and CLI flags. Precedence follows a strict order: **CLI flags override
> environment variables, which override user config, which overrides repo config, which overrides
> built-in defaults.**"

With stated rationale per layer: repo config "holds settings the whole team shares"; user config
"holds personal preferences… should not be version-controlled"; env vars are "the override layer for
CI/CD pipelines"; flags are "for one-off overrides."

A fifth source exists for the server — `.swamp/serve.yaml`, occupying the same precedence slot as
`.swamp.yaml` — with one carve-out worth copying:

> "Security-sensitive settings (authentication mode, admin identity, TLS certificates) are excluded
> from the serve config file entirely and remain CLI-only or environment-variable-only."

Repo root resolution (`/manual/reference/repository-configuration`): `--repo-dir` → `SWAMP_REPO_DIR`
→ walk up from cwd, **bounded by the git repo root, or 10 ancestor levels outside a git repo**.

Caveat for hyper: the manual describes one clean four-layer order, but §3a of this document shows the
implementation hand-rolls a separate chain per setting. The documented model is the one to copy; the
implementation is not evidence that it was easy.

### 13.3 CLI and output contract

Global flags, verbatim from `/manual/reference/doctor`:

| Flag | Description |
| --- | --- |
| `--json` | "Output in JSON format (non-interactive)." |
| `--log` | "Force flat log output (no interactive tree)." |
| `--log-level <level>` | `trace`, `debug`, `info`, `warning`, `error`, `fatal` |
| `-q`, `--quiet` | "Suppress non-essential output." |
| `--no-telemetry` | |
| `--show-properties` | "Show structured properties in log output." |
| `--no-color` | |
| `--server` | "Run against a remote `swamp serve` instance (env: `SWAMP_SERVE_URL`)." |
| `--token` | "Server token in `<name>.<secret>` format; only with `--server`" |
| `--repo-dir <dir>` | env: `SWAMP_REPO_DIR` |

Output-mode conventions worth stealing:

- `--json` is universal, and **implies non-interactive**: `data prune` — "Output in JSON format;
  implies non-interactive mode (skips confirmation prompt)"; `vault read-secret` — "In `--json`
  mode, outputs the value directly without prompting."
- TTY-sensitivity is explicit rather than implicit: `data query` with no predicate opens a TUI on a
  TTY, and **returns an error** in non-interactive mode rather than doing something arbitrary.
- Long-running commands stream **NDJSON, not a wrapped array**: `extension pull --json` emits "a
  stream of JSON objects, one per pull stage… the stream is not wrapped in an outer array."
- `--junit` (with `--out`) for CI, and "`--junit` and `--json` cannot be combined."

**Exit codes** — the only place Swamp gets specific, and it is a good model:

- `swamp model method run`: `0` success, `1` general error, **`75` lock contention — "retry with
  exponential backoff"**. Machine-readable error codes on stderr: `lock_timeout`, `model_not_found`,
  `unknown_model_type`, `unknown_method`, `no_evaluated_definition`, `missing_deps`,
  `method_execution_failed`, `not_authenticated`, `cancelled`.
- All seven `doctor` subcommands: `0` pass / `1` fail — so CI can gate on any of them.
- `swamp workflow run` blocks and its exit code reflects the outcome: "There is no async or detached
  mode on the CLI." (Except for suspension — see §9a.7, where the code returns 0.)

A distinct exit code for a *retryable* condition, plus a stable error-code vocabulary alongside the
human message, is exactly what the map's "CLI surface and JSON output contract" item needs.

Documented command families: `model`, `workflow`, `data`, `vault`, `extension`, `repo`, `config`,
`doctor`, `run`, `auth`, `access`, `worker`, `serve`, `datastore`, `issue`, `agent`, `update`,
`completions`, `version`, `report`, `source`, `summarise`, `telemetry`, `audit`. Note the manual
mentions `workflow list`, `workflow status`, `workflow runs`, `report run`, and `report search` only
in prose, with no flags table — treat those as unverified.

### 13.4 Dry-run: per-command, never global

**There is no global `--dry-run` or plan mode.** `--dry-run` exists on `data gc`, `data prune`,
`run gc`, `vault migrate`, `extension push`, and `doctor extensions` — i.e. on *housekeeping*
commands only.

`swamp model method run` and `swamp workflow run` have **no `--dry-run`**. The nearest equivalents
are `model validate` / `model evaluate` and `workflow validate` / `workflow evaluate` ("Test CEL
expressions in a workflow without executing it"), plus `workflow get --graph`. **[inferred]** that
these are the intended substitute — the manual never calls them a dry-run.

For a tool that provisions and deletes infrastructure, that is a notable gap, and it bears directly
on hyper's destructive-operations ticket: Swamp's answer to "what will this do?" is "evaluate the
expressions and look at the DAG", not "simulate the effects".

### 13.5 The `rollbackOnFailure` warning

The single most valuable safety passage in the manual
(`/manual/explanation/models-types-and-methods`):

> "**Do not use `rollbackOnFailure` on methods that interact with external systems.** If a method
> provisions three VMs in a cloud provider, writes their IDs to the data layer, then fails
> provisioning a fourth, rollback discards the records of VMs 1–3 — but those VMs still exist in the
> cloud. The data layer now says they do not exist. **This is worse than partial writes: it is a
> silent divergence between what Swamp knows and what is actually running.**"

Swamp shipped a transactional-rollback feature and then had to document that using it against real
infrastructure produces silent drift. The general principle for hyper: **a rollback that only
rewinds the record, not the effect, is worse than no rollback**, because it converts a visible
partial failure into an invisible inconsistency. This should inform whatever hyper does about
partial failure of destructive operations.

Related, and good: the manual's rationale for keeping `prune` out of `gc` — "gc runs unattended
(including as an automatic post-method hook), while prune defaults to a confirmation prompt and
supports `--dry-run` for preview." Splitting operations by *whether they may run unattended* is a
cleaner axis than splitting by what they delete.

### 13.6 Other safety mechanisms the manual documents

- **Pre-flight vault validation**: a mutating method with sensitive outputs and no vault "is rejected
  immediately — before any method logic runs." (Matches §6.)
- **Checks run before execution**, with stated reasoning: "A check that fails before execution begins
  leaves nothing to clean up." But they are skippable via `--skip-checks`, `--skip-check`, and
  `--skip-check-label`, with no documented restriction — so the guarantee is opt-out by default.
- **Guards + `resume --from`**: "Steps that completed successfully before the failure are **not
  re-run** — this prevents repeating irreversible side effects."
- **Assert severities** `low|medium|high` with a `--fail-on` threshold (default `low`).
- **Uniform destructive-command convention**: `-y, --yes` (with `--force`/`-f` accepted) on every
  delete-shaped command, and explicit irreversibility language — `data prune`: "Deletion is
  irreversible."; `datastore lock release --force` sits under a heading literally called
  "Breakglass".
- **`swamp serve` hard refusal**, and the reasoning is worth quoting: off-loopback binding "requires
  both `--cert-file` / `--key-file` and `--auth-mode` other than `none`. **This is not
  configurable.**" Because "An unauthenticated, unencrypted control plane on the network is arbitrary
  remote execution." A non-negotiable refusal, stated as such.
- **`runModel` limits**: max call depth 10, max 100 total invocations, cycle detection, and "Vault
  secrets are isolated per extension" — the one place per-extension scoping does exist.
- **Remote workers**: "Workers never receive vault credentials or connection details — they see only
  the resolved plaintext for secrets referenced by the current step."

### 13.7 Claims to treat as marketing, not mechanism

The manual is unusually mechanism-heavy — a corpus-wide grep for "seamless", "effortless",
"enterprise-grade", "just works", "guarantee" finds essentially nothing under `/manual`; the hits
are all on the marketing pages. That makes the remaining unsupported claims easier to isolate:

1. **"Deterministic automation for AI agents."** The strongest claim on the site, and nowhere
   defined. Nothing in the manual describes determinism enforcement — no pinned execution
   environment for `command/shell`, no replay, no input hashing. Our own findings (§3b: expressions
   read live disk state; §9a.2: no inferred ordering) suggest the opposite. Treat "deterministic" as
   meaning "the same YAML is executed again", nothing stronger.
2. **"Extreme Token Efficiency"** — no measurement, baseline, or mechanism anywhere.
3. **"adaptive automation framework"** — "adaptive" is never defined across 146 pages.
4. **"It plugs into any agent harness, with any model"** — the docs actually cover six built-in
   tools plus a custom-tool definition that captures three facts, and concede custom tools "get the
   substance (skills and instructions) without the polish (audit recording, diagnostics, harness
   awareness)."
5. **"Nothing leaves your environment unless you configure a datastore"** — true of *produced
   artifacts*, not of the process: telemetry is on by default and posts to `api.swamp-club.com`, and
   extension pull/push, `swamp issue`, and `auth` are all network operations.
6. **"Swamp is interesting because it doesn't trust AI." / "The agent reasons freely. Swamp keeps it
   honest."** The actual enforcement surface is schema validation, sensitive-literal rejection,
   pre-flight checks, and approval gates. But checks are skippable and `command/shell` is an
   unschematised escape hatch — **nothing prevents an agent from routing everything through
   `command/shell`**, which dissolves the typed-model guarantee entirely. This is the most important
   gap between Swamp's pitch and its mechanism, and it is a trap hyper can fall into identically:
   *a typed model layer with a shell escape hatch provides the safety of a shell.*
7. **"Secrets are never frozen into YAML files, never written to `.swamp` data, and never cached
   between runs."** There *is* a real mechanism behind this (§9c.9), so it is checkable rather than
   vapor — but "never" is absolute, and §9b.7 found documented gaps (audit logs explicitly not
   covered; `z.record()`/`z.union()` sensitive fields missed).
8. **An internal contradiction worth noting.** The manual says "Schemas are validated at creation
   time and execution time, so type mismatches surface early." But `/manual/reference/data` says:
   "Resource data is validated against the spec's Zod schema on write. **Schema mismatches produce a
   warning, not an error.**" The typed-data guarantee is softer than advertised.
9. **Unsourced aggregates** — a "153,577,370 automation events" homepage counter, and testimonial
   claims ("shipped 900 times in four weeks", "Killing n8n") attributed to individuals and podcasts.

### 13.8 Incidental facts worth recording

- License is **AGPL-3.0** with a "Swamp Extension and Definition Exception" (`COPYING`,
  `COPYING-EXCEPTION`), plus a trademark policy that reserves the name.
- The canonical source is self-hosted at `git.swamp-club.com/swamp-club/swamp`; GitHub is a mirror.
- The vendor entity is **System Initiative** (named on the extension-scorecard page), which explains
  the reserved `si` collective alongside `swamp`.
- Contribution model: **fork PRs are automatically closed** — issue-driven only, justified as
  "supply chain security in the age of AI-generated code" (`README.md:167-172`).
- The manual enforces its own terminology: "Always use 'operative' in prose — never 'operator'";
  "Always use 'collective' in prose — never 'organization'" (`/manual/reference/glossary`).
