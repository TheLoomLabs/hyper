# A worked scenario: rotating Tailscale CI auth keys

**Frozen at `hyper` commit `8f4e950`, worked 2026-08-15.** This document is never updated — it is a
conformance exercise against the specification at one revision, and its value is that it records what
that revision did and did not decide. If the spec has moved on, that is expected; re-work the scenario
from scratch rather than editing this file.

One repository, one Run, end to end: the five reviewed artefacts, every Store file the Run writes, and
every rendering the Run produces. Everything derivable is derived — the blob ids, the commit ids, the
Manifest digest, the identity-set digests and the `git diff` count below were computed from the actual
files, not invented, so the numbers agree with each other and can be re-checked.

## Method, and what this is evidence of

This is not research against an external source. It is the specification read as an implementer would
read it — [`docs/spec/`](../spec/) §0–§13, [`CONTEXT.md`](../../CONTEXT.md) and all seventy-six
[ADRs](../adr/) — and then *used*, once, on a scenario nobody had written down. The scenario was
chosen to make several rules meet rather than to exercise any one of them.

What it is evidence of is only what §6 records: **the twenty places the corpus did not decide the
question, or decided it in two places differently.** Everything above §6 is a demonstration that the
model composes; §6 is the finding. Nothing here is normative — the specification is `docs/spec/`, the
rationale is `docs/adr/`.

The scratch git repository the digests and diff counts were computed in is not kept: it is
reproducible from the artefacts in §1 with `git hash-object` and `sha256sum`.

---

## 0. The scenario, and why this one

A solo operator keeps a fleet of self-hosted CI runners on a Tailscale tailnet. Each runner joins with
its own **ephemeral, tagged, 30-day auth key**. Keys expire; expired keys have to be retired and
reissued. The Procedure does exactly that, in one pass: retire what is past its 30 days, then issue
whatever the runner list asks for and the record does not already hold.

The shape was chosen because it exercises the parts of the model that only bite when they meet:

- a Definition that **effects and does not observe** (ADR-0032) — one Definition claiming `mutate`
  beside a named `destroy:`;
- `skip-if-recorded` deciding **per Record, not per Step** (ADR-0056) — one Step that calls for three
  members and skips a fourth;
- **destroy-then-recreate inside one Run** — §5 drops a Tombstoned member from a `destroy` Expansion
  and §6 runs it under a `mutate`, which is what makes rotation authorable at all;
- a **secret output** — `secret: [key]`, the presence-only marker in the Store, and the Secret sink
  the invocation has to supply;
- two versions of one Record series written by one Run at two `<nnnn>`, which is the case §12's path
  grammar exists to disambiguate.

It also ran into one wall that decided the artefact's shape, and that is finding **#1** below: this
Procedure **cannot carry a Cadence**.

---

## 1. The five artefacts

### `providers/tailscale.yaml`

```yaml
kind: provider
provider: tailscale
schema-version: 1
class: tailscale
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_keys:
    kind: read
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /api/v2/tailnet/{tailnet}/keys
    input:
      type: object
      required: [tailnet]
      properties:
        tailnet: {type: string}
    record:
      over: $.body.keys
      identity: $.id
      fields:
        id: $.id
        description: $.description
        created: $.created
        expires: $.expires
  create_key:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    patterns:
      retry: {attempts: 3}
    http:
      method: POST
      host: "{from-target}"
      path: /api/v2/tailnet/{tailnet}/keys
      body:
        description: "{description}"
        expirySeconds: "{expiry_seconds}"
        capabilities:
          devices:
            create:
              reusable: false
              ephemeral: true
              preauthorized: true
              tags: ["tag:ci"]
    input:
      type: object
      required: [tailnet, description, expiry_seconds]
      properties:
        tailnet: {type: string}
        description: {type: string}
        expiry_seconds: {type: integer}
    record:
      identity: "{description}"
      fields:
        id: $.body.id
        description: $.body.description
        created: $.body.created
        expires: $.body.expires
        key: $.body.key
    secret: [key]
  delete_key:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    patterns:
      retry: {attempts: 3}
    http:
      method: DELETE
      host: "{from-target}"
      path: /api/v2/tailnet/{tailnet}/keys/{key_id}
    input:
      type: object
      required: [tailnet, key_id]
      properties:
        tailnet: {type: string}
        key_id: {type: string}
```

`identity: "{description}"` is a **template hole and not a response path**, which is what
`skip-if-recorded` requires: the test reads the head of the series the call would write under, so the
name has to exist before the call it is deciding on (§3, §12, ADR-0056). Tailscale's own key id would
have been the obvious `identity:` and is `manifest-inconsistent`.

`secret: [key]` sits beside the projection rather than inside it, so `fields:` values stay uniformly
scalar (§3). No predicate anywhere names `key` — one against a declared-secret field is
`predicate-type-mismatch` at load (§4).

`repeatability:` is omitted on `list_keys` because a `read` has one legal value (§12). `concurrency:`
is omitted everywhere — declared on anything but a `read` it is `manifest-inconsistent` (ADR-0045),
and `list_keys` has no measured number behind it. `origin:` is absent, so this is a locally authored
Provider and `origin_digest` is absent from every Provenance below (§7, ADR-0073).

Digest over these exact bytes:
`sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635`

### `targets/tailscale-prod.yaml`

```yaml
kind: target-declaration
target: tailscale-prod
class: tailscale
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.tailscale.com]
auth:
  token: {env: TAILSCALE_API_KEY}
```

`hosts:` is present exactly because `capabilities:` grants `http`; either without the other is
`target-inconsistent` (§4). The grant holds one host, so the candidate set from `{from-target}`
intersects to one and `hyper` fills it — no `host-input:` anywhere in the Manifest (ADR-0029).
`kinds:` accepts `read` although this repository's one Definition never claims it; a Target accepts,
a Definition claims, and the Step runs in the intersection (§5). No `opaque-destroy:` — nothing bound
here is opaque.

### `definitions/ci-keys.yaml`

```yaml
kind: definition
definition: ci-keys
provider: tailscale
kinds: [mutate]
destroy: [delete_key]
targets: [tailscale-prod]
```

`read` may not appear beside a `destroy:` claim — a Definition observes or it effects
(`definition-kinds-mixed`, ADR-0032). Reading these keys back would be a **second** Definition against
the same Provider and Target, and the two series would never meet. This repository does not have one,
which is why `THE WORLD MOVED` is structurally empty below.

`destroy: [delete_key]` names the Operation rather than a Kind: granularity follows severity (§5).

### `procedures/sync-ci-keys.yaml`

```yaml
kind: procedure
procedure: sync-ci-keys
targets: [tailscale-prod]
steps:
  # retire before issuing, so a key past its 30 days is reissued in this same Run
  - id: retire-expired
    definition: ci-keys
    operation: delete_key
    target: tailscale-prod
    over:
      assets:
        - field: description
          starts_with: ci-
        - field: created
          older_than: 30d
    args:
      tailnet: example.com
      key_id: {item: $.id}
    bound: 5

  - id: issue-runner-keys
    definition: ci-keys
    operation: create_key
    target: tailscale-prod
    over:
      values:
        - ci-arm64
        - ci-x86
        - ci-macos
        - ci-arm64-2
    args:
      tailnet: example.com
      description: {item: $}
      expiry_seconds: 2592000
    bound: 4

  - id: issue-bootstrap-key
    definition: ci-keys
    operation: create_key
    target: tailscale-prod
    args:
      tailnet: example.com
      description: bootstrap-2026
      expiry_seconds: 31536000
```

No `cadence:` — see finding **#1**.

The order is load-bearing rather than stylistic. `retire-expired` Tombstones what is past 30 days;
`issue-runner-keys` then reaches those same members, because a `mutate` runs a member whose head is a
Tombstone (§6, ADR-0056) while a `destroy` drops one (§5). Reverse the two Steps and rotation lags by
a whole occurrence.

`{item: $}` in `description:` makes the `values:` list a list of **identifiers**, not hosts: a member
is a host only where the Step wires it into an Operation's `host-input:`, and this Manifest has none
(§3, ADR-0024). It also satisfies §4's collision check — a `{item:}` reference reaches the value that
fills the identity hole, so the four members cannot project one name.

`bound: 4` on a `mutate` is optional and is written anyway, which buys the offline check: `check`
reads the authored list's length against it (`bound-exceeded`, §4). `issue-bootstrap-key` carries none
and renders `mutate!` in a review's gutter with an `UNBOUNDED` flag. `bound: 5` on the `destroy` is
mandatory — an absent one there means unbounded and is refused before anything runs.

The comment is source. It renders verbatim in place and `hyper` never reads it (§3, §8).

Blob id at the revision this Run read: `9843c17157342973d078a863ab73ecc91ebbd8e4`

### `hyper.yaml`

```yaml
kind: repository-declaration
version: 0.4.1
digest: sha256:a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2
retention: 180d
```

Both keys are the only two facts that govern every Run and belong to no Procedure, Definition or
Target (ADR-0020). `version:` and `digest:` are written only by `hyper project`.

---

## 2. What the Store holds before the Run

Not part of the deliverable, but the Run is unreadable without it. Five Asset series stand under
`(tailscale-prod, ci-keys)`:

| name | `created` | head ordinal | age at this Run |
| --- | --- | --- | --- |
| `bootstrap-2026` | 2026-01-02T14:22:05Z | 1 | 225d |
| `ci-arm64` | 2026-08-01T09:41:11Z | 4 | 14d |
| `ci-macos` | 2026-07-10T09:33:52Z | 4 | 36d |
| `ci-riscv` | 2026-06-20T09:29:14Z | 7 | 56d |
| `ci-x86` | 2026-07-10T09:33:55Z | 5 | 36d |

The previous Run of this Procedure — the Comparison's baseline — is
`019fbcb2-e46a-79c2-82e5-b7a41c0d3f19`, at `2026-08-01T09:41:08.330Z`, on the same laptop. Its
Procedure revision was `9e09748`; between then and now an agent added `ci-arm64-2` to the runner list,
raised that Step's Bound from 3 to 4, and reworded the comment.

## 3. What the Run does

`hyper run sync-ci-keys --secret-out /home/igor/.local/state/hyper/ci-keys-2026-08-15.json`

Run id `01a004b3-6bb5-74d1-81a7-0c93be5482d6` (UUIDv7; its 48-bit prefix is the start instant),
started `2026-08-15T09:14:22.517Z`, trigger manual/local.

**Step 1 `retire-expired`.** The selector reads the **head** version of each series and the two
conjuncts AND. `older_than: 30d` resolves against the instant on `run.json` and against nothing else
(ADR-0034): `2026-07-16T09:14:22Z`. Three heads qualify. Expansion order for an `assets:` selector is
the Record **name** by Unicode code point, never the percent-encoded path (ADR-0044), so it is
`ci-macos`, `ci-riscv`, `ci-x86`. Three ≤ Bound 5. A `destroy` Expansion is strictly serial, so
*which three of the five* has an answer and the halt point would be legible had one halted. All three
confirm; three Tombstones.

**Step 2 `issue-runner-keys`.** Expansion is the authored list, top-first. The skip test runs once per
member at that member's turn:

| member | head at its turn | outcome |
| --- | --- | --- |
| `ci-arm64` | stands | skipped |
| `ci-x86` | Tombstone from Step 1 | called |
| `ci-macos` | Tombstone from Step 1 | called |
| `ci-arm64-2` | no series at all | called |

Three calls went out, so the Step is **ran** and not *skipped as already recorded* — that value is for
a Step where every member skipped (§12, ADR-0056). Every member is in `expanded_to` whether its call
went out or not, and the identity set holds all four, the skip test having concluded about each.

**Step 3 `issue-bootstrap-key`.** No selector, so a set of one and one skip test. The head stands;
the Step is **skipped as already recorded**, which is the only Disposition that is Repeatability
evidence, and it carries a set of one.

Outcome `completed`, exit `0`.

---

## 4. The Store files this Run writes

Eleven paths, every one of them new; nothing in the Store is ever rewritten (ADR-0011). Every path
carries the id of the Run that wrote it (ADR-0076). No segment here needs percent-encoding — every
name is `[a-z0-9-]` — and none approaches the 200-byte truncation (§12).

```
journal/2026/08/15/01a004b3-6bb5-74d1-81a7-0c93be5482d6/run.json
journal/2026/08/15/01a004b3-6bb5-74d1-81a7-0c93be5482d6/steps/0001.json
journal/2026/08/15/01a004b3-6bb5-74d1-81a7-0c93be5482d6/steps/0002.json
journal/2026/08/15/01a004b3-6bb5-74d1-81a7-0c93be5482d6/steps/0003.json
journal/2026/08/15/01a004b3-6bb5-74d1-81a7-0c93be5482d6/outcome.json
records/tailscale-prod/ci-keys/ci-macos/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0001.json
records/tailscale-prod/ci-keys/ci-riscv/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0001.json
records/tailscale-prod/ci-keys/ci-x86/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0001.json
records/tailscale-prod/ci-keys/ci-macos/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0002.json
records/tailscale-prod/ci-keys/ci-x86/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0002.json
records/tailscale-prod/ci-keys/ci-arm64-2/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0002.json
```

`ci-macos` and `ci-x86` each gain **two versions from one Run**, at `-0001` and `-0002`. That is
exactly what §12's `<nnnn>` disambiguates: two Steps of one Run writing one identity write two paths
rather than one path twice.

`ci-arm64` and `bootstrap-2026` write nothing at all. They were skipped, so no call went out and no
bytes moved — and they are still in their Steps' identity sets, which is the whole reason the set is
*what the Step concluded about* and not *what it wrote* (ADR-0030).

Commits: one per confirmed write, six of them, pushed after every effectful Step. The Store is never
checked out and has no uncommitted local state at any moment (ADR-0075).

### `run.json`

```json
{
  "dry_run": false,
  "procedure": "sync-ci-keys",
  "provenance": {
    "hyper_version": "0.4.1",
    "procedure_revision": "9843c17157342973d078a863ab73ecc91ebbd8e4",
    "repo_revision": "b31703fe796ba46511758a7cf118dfc9b789bb6e"
  },
  "run_id": "01a004b3-6bb5-74d1-81a7-0c93be5482d6",
  "schema_version": 1,
  "started_at": "2026-08-15T09:14:22.517Z",
  "trigger": {
    "actor": "igor",
    "cause": "manual",
    "executor": "local",
    "host": "thinkpad"
  }
}
```

`dry_run: false` is written out — the one marker in the Store that does not follow the absence rule
(§7). Run-level Provenance carries only the members that have exactly one value at Run level: no
`definition_revision`, no `manifest_digest` (ADR-0043). No `repo_dirty`, the tree being clean.

### `steps/0001.json` — `retire-expired`

```json
{
  "definition": "ci-keys",
  "disposition": "ran",
  "ended_at": "2026-08-15T09:14:26.884Z",
  "id": "retire-expired",
  "identities": {
    "digest": "sha256:328a273b5d99a744f9c2afc4e9aaa694e631036e75136e0692625d88a138556d",
    "members": [
      "ci-macos",
      "ci-riscv",
      "ci-x86"
    ]
  },
  "kind": "destroy",
  "operation": "delete_key",
  "provenance": {
    "definition_revision": "03d962a71e11cd83d62d8ac60b111e17277338cc",
    "manifest_digest": "sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"
  },
  "provider": "tailscale",
  "schema_version": 1,
  "selector": {
    "bound": 5,
    "declared": {
      "assets": [
        {"field": "description", "starts_with": "ci-"},
        {"field": "created", "older_than": "30d"}
      ]
    },
    "expanded_to": [
      "ci-macos",
      "ci-riscv",
      "ci-x86"
    ]
  },
  "started_at": "2026-08-15T09:14:23.902Z",
  "step": 1,
  "target": "tailscale-prod"
}
```

`members` is written because the digest moved: at the baseline Run this Step confirmed one Asset
(`ci-arm64`), digest
`sha256:95a3fb31249d304ff7696bcfb05979032e07da4dc1fc23043fe743bcbac394d2`. The digest is
`sha256:` over the canonical JSON encoding of the sorted array with its trailing LF, so a reader
recomputes it with `sha256sum` over those exact bytes:

```
[
  "ci-macos",
  "ci-riscv",
  "ci-x86"
]
```

No `origin_digest` — the Provider is locally authored. No `answered` — every call was `2xx`. No
Pattern account: `retry` is declared and did the trivial single call, so nothing is written (§7).

### `steps/0002.json` — `issue-runner-keys`

```json
{
  "definition": "ci-keys",
  "disposition": "ran",
  "ended_at": "2026-08-15T09:14:31.407Z",
  "id": "issue-runner-keys",
  "identities": {
    "digest": "sha256:5e738edb6a4d1ba58d5167244bbe1fe5e53fc5bd98cd9e46af15ef91eb525b8c",
    "members": [
      "ci-arm64",
      "ci-arm64-2",
      "ci-macos",
      "ci-x86"
    ]
  },
  "kind": "mutate",
  "operation": "create_key",
  "provenance": {
    "definition_revision": "03d962a71e11cd83d62d8ac60b111e17277338cc",
    "manifest_digest": "sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"
  },
  "provider": "tailscale",
  "schema_version": 1,
  "selector": {
    "bound": 4,
    "declared": {
      "values": [
        "ci-arm64",
        "ci-x86",
        "ci-macos",
        "ci-arm64-2"
      ]
    },
    "expanded_to": [
      "ci-arm64",
      "ci-x86",
      "ci-macos",
      "ci-arm64-2"
    ]
  },
  "started_at": "2026-08-15T09:14:26.891Z",
  "step": 2,
  "target": "tailscale-prod"
}
```

The two lists differ in order, and that is the point: `expanded_to` is a **sequence** in Expansion
order, which for a `values:` list is the page as authored; `members` is a **set**, sorted by Unicode
code point. `declared` and `expanded_to` are identical here, which says no member was dropped — under
a `destroy` a member present in `declared` and absent from `expanded_to` is one the Store already
holds a Tombstone for (§7).

The digest moved: at the baseline this Step held three members,
`sha256:a3dc9f1f3f557587de73eebcef05bfb24dcde48ea6e3a07367d65ca7f0f793b4`. It moved because the
**artefact** gained a member, not because the world did.

### `steps/0003.json` — `issue-bootstrap-key`

```json
{
  "definition": "ci-keys",
  "disposition": "skipped-as-already-recorded",
  "ended_at": "2026-08-15T09:14:31.418Z",
  "id": "issue-bootstrap-key",
  "identities": {
    "digest": "sha256:e5953f18c542a4770c98fb41deac17002570dae37f8467cffb73882295cfe5b8"
  },
  "kind": "mutate",
  "operation": "create_key",
  "provenance": {
    "definition_revision": "03d962a71e11cd83d62d8ac60b111e17277338cc",
    "manifest_digest": "sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"
  },
  "provider": "tailscale",
  "schema_version": 1,
  "started_at": "2026-08-15T09:14:31.412Z",
  "step": 3,
  "target": "tailscale-prod"
}
```

No `members`: the digest is the one this Step carried in the last Run of the Procedure that carried
one, and it did not move (ADR-0055). Resolving the set means walking back to that Run, which `show`
does by saying so rather than rendering them bare. No `selector` key at all — a Step carrying no
`over:` resolved none and holds none.

### `outcome.json`

```json
{
  "ended_at": "2026-08-15T09:14:31.463Z",
  "outcome": "completed",
  "schema_version": 1
}
```

No exit code, no duration, no `refusal`. The exit code is a mapping the CLI applies, and the Store
does not restate a rendering (§7).

### Record versions — the three Tombstones

`records/tailscale-prod/ci-keys/ci-riscv/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0001.json`

```json
{
  "definition": "ci-keys",
  "fields": {
    "created": "2026-06-20T09:29:14Z",
    "description": "ci-riscv",
    "expires": "2026-07-20T09:29:14Z",
    "id": "kR4nTbC1CNTRL",
    "key": "<secret>"
  },
  "name": "ci-riscv",
  "operation": "delete_key",
  "provenance": {
    "definition_revision": "03d962a71e11cd83d62d8ac60b111e17277338cc",
    "hyper_version": "0.4.1",
    "manifest_digest": "sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635",
    "procedure_revision": "9843c17157342973d078a863ab73ecc91ebbd8e4",
    "repo_revision": "b31703fe796ba46511758a7cf118dfc9b789bb6e"
  },
  "record_type": "asset",
  "run_id": "01a004b3-6bb5-74d1-81a7-0c93be5482d6",
  "schema_version": 1,
  "step": 1,
  "target": "tailscale-prod",
  "tombstone": true,
  "written_at": "2026-08-15T09:14:25.902Z"
}
```

`fields` are the previous Head's, copied forward as the Asset's last known state — including
`key: "<secret>"`, which is a constant and carries forward like any other value. `operation` names
what destroyed it while `fields` were projected by an earlier call: the one place in the Store those
two keys describe different calls. A Record version carries the **whole** of Provenance, unlike the
Step file beside it, because it sits under a Record path with no entry next to it (ADR-0043).

`ci-macos` and `ci-x86` write the same shape at `written_at` `09:14:24.771Z` and `09:14:26.845Z`,
with their own last known fields — ascending in Expansion order, which is what a serial `destroy`
guarantees.

### Record versions — the three creations

`records/tailscale-prod/ci-keys/ci-arm64-2/01a004b3-6bb5-74d1-81a7-0c93be5482d6-0002.json`

```json
{
  "definition": "ci-keys",
  "fields": {
    "created": "2026-08-15T09:14:30Z",
    "description": "ci-arm64-2",
    "expires": "2026-09-14T09:14:30Z",
    "id": "kT8mLd3CNTRL",
    "key": "<secret>"
  },
  "name": "ci-arm64-2",
  "operation": "create_key",
  "provenance": {
    "definition_revision": "03d962a71e11cd83d62d8ac60b111e17277338cc",
    "hyper_version": "0.4.1",
    "manifest_digest": "sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635",
    "procedure_revision": "9843c17157342973d078a863ab73ecc91ebbd8e4",
    "repo_revision": "b31703fe796ba46511758a7cf118dfc9b789bb6e"
  },
  "record_type": "asset",
  "run_id": "01a004b3-6bb5-74d1-81a7-0c93be5482d6",
  "schema_version": 1,
  "step": 2,
  "target": "tailscale-prod",
  "written_at": "2026-08-15T09:14:31.288Z"
}
```

The actual key never reaches the Store. `"<secret>"` occupies the position the value would have had —
no digest, no length, no sibling list of what was suppressed. It is a constant, which is what keeps
the *bytes moved* test honest: a rotated key that changed nothing else would write identical bytes and
correctly mint no version. The key itself went to the Secret sink the invocation named.

`ci-x86` (`written_at` `09:14:28.019Z`, ordinal 7) and `ci-macos` (`09:14:29.640Z`, ordinal 6) write
the same shape at `-0002`, landing **above** the Tombstones their own Run wrote three seconds earlier.
A further version above a Tombstone makes the Head alive again — which is what makes this Procedure a
rotation rather than a demolition.

---

## 5. The renderings

### `hyper run`

```
$ hyper run sync-ci-keys --secret-out /home/igor/.local/state/hyper/ci-keys-2026-08-15.json

  STEP  ID                   KIND     DISPOSITION                  RECORDS
  1     retire-expired       destroy  ran                          3
  2     issue-runner-keys    mutate   ran                          4
  3     issue-bootstrap-key  mutate   skipped as already recorded  1

  completed · exit 0 · run 01a004b3-6bb5-74d1-81a7-0c93be5482d6
```

`RECORDS` is the size of the identity set — what each Step concluded about, never what it wrote. The
Run wrote six versions and the column reads 3, 4, 1. Step 2 renders `4` and not `3 of 4`: the skip
test reached a conclusion about every member, so nothing is unaccounted for, and `n of m` is reserved
for a Step that stopped short of its Expansion. Step 3 made no call at all and still renders `1`,
which is the evidentiary content of that Disposition rather than an artefact of it.

Nothing rendered before the Run. There is no pre-flight summary and no confirmation, on any Kind
(ADR-0015).

### `hyper run --json`

```
{"type":"step","index":1,"id":"retire-expired","kind":"destroy","disposition":"ran","records":3}
{"type":"step","index":2,"id":"issue-runner-keys","kind":"mutate","disposition":"ran","records":4}
{"type":"step","index":3,"id":"issue-bootstrap-key","kind":"mutate","disposition":"skipped-as-already-recorded","records":1}
{"type":"provenance","procedure_revision":"9843c17157342973d078a863ab73ecc91ebbd8e4","repo_revision":"b31703fe796ba46511758a7cf118dfc9b789bb6e","hyper_version":"0.4.1"}
{"type":"provenance","step":1,"definition_revision":"03d962a71e11cd83d62d8ac60b111e17277338cc","manifest_digest":"sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"}
{"type":"provenance","step":2,"definition_revision":"03d962a71e11cd83d62d8ac60b111e17277338cc","manifest_digest":"sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"}
{"type":"provenance","step":3,"definition_revision":"03d962a71e11cd83d62d8ac60b111e17277338cc","manifest_digest":"sha256:4757bb8f7f9536aa2b66e5bea49f037e158fc98eae298fd3eb030aab20072635"}
{"type":"outcome","outcome":"completed","code":0,"dry_run":false,"run_id":"01a004b3-6bb5-74d1-81a7-0c93be5482d6"}
```

One `provenance` row per Step file written, distinguished by `step`, and one Run-wide row — the Store's
split arriving on the wire. `origin_digest` is absent rather than `null`. The `outcome` row carries no
`error_code`: a completed Run has no check to name. See finding **#9** on whether these rows abbreviate.

### `hyper changes sync-ci-keys`

```
$ hyper changes sync-ci-keys

  sync-ci-keys
  BASELINE  019fbcb2-e46a…  igor@thinkpad  Sat 1 Aug 09:41   completed  11s  procedure rev 9e09748
  SUBJECT   01a004b3-6bb5…  igor@thinkpad  Sat 15 Aug 09:14  completed   9s  procedure rev 9843c17

  YOU DID THIS   4 assets
  CHANGE     TARGET          DEFINITION  RECORD      ORDINAL  FIELDS
  created    tailscale-prod  ci-keys     ci-arm64-2  – → 1    created: 2026-08-15T09:14:30Z · description: ci-arm64-2 · expires: 2026-09-14T09:14:30Z · id: kT8mLd3CNTRL · key: <secret>
  changed    tailscale-prod  ci-keys     ci-macos    4 → 6    created: 2026-07-10T09:33:52Z → 2026-08-15T09:14:29Z · expires: 2026-08-09T09:33:52Z → 2026-09-14T09:14:29Z · id: kM2wHf5CNTRL → kQ4vNp7CNTRL
  destroyed  tailscale-prod  ci-keys     ci-riscv    7 → 8    † confirmed 09:14 · created: 2026-06-20T09:29:14Z · description: ci-riscv · expires: 2026-07-20T09:29:14Z · id: kR4nTbC1CNTRL · key: <secret>
  changed    tailscale-prod  ci-keys     ci-x86      5 → 7    created: 2026-07-10T09:33:55Z → 2026-08-15T09:14:27Z · expires: 2026-08-09T09:33:55Z → 2026-09-14T09:14:27Z · id: kX7pJr9CNTRL → kSbYs2Q1CNTRL

  THE WORLD MOVED   0 observations

  THE CODE MOVED   4 facts
  SUBJECT                 FACT                               FROM                                 TO
  procedure sync-ci-keys  procedure revision                 9e09748                              9843c17
  procedure sync-ci-keys  step issue-runner-keys · bound     3                                    4
  procedure sync-ci-keys  step issue-runner-keys · selector  values: ci-arm64, ci-x86, ci-macos   values: ci-arm64, ci-x86, ci-macos, ci-arm64-2
  —                       repository revision                3a4fcac                              b31703f
  2 other lines changed · git diff 3a4fcac b31703f

  TOTALS  4 changes · 4 asset · 0 observation · 1 tombstone · the code moved
```

Four things in this rendering are worth reading twice.

**`ci-arm64` and `bootstrap-2026` get no row.** Both are in the subject Run's identity sets, so both
were eligible; neither head moved, so neither is a change. That is how *nothing changed* is told from
*nobody looked* — the identity sets say a Step concluded about them, and the absence of a row says the
Record stood still.

**The two rotations render `changed`, not `destroyed` then `created`.** A row is a Record read at its
two endpoints and never the versions between them (ADR-0058). `ci-macos` held a live Asset at the
baseline end and holds a live Asset now, so the Tombstone this Run wrote in between is admitted only
by the ORDINAL gap of two. This is §13's *nor does a row show a Record's intermediate states*, met on
the first realistic Procedure I wrote. See finding **#12**.

**The key demonstrably rotated and nothing says so.** Every `changed` row's `FIELDS` omits `key`,
because the marker is a constant and constants do not move. What makes the rotation legible at all is
that this Manifest also projects `id`, `created` and `expires` — non-secret metadata that moves with
the secret. A Provider projecting the secret and nothing else would render a row with no fields.

**The catch-all is the comment.** The diff between the two revisions is five `git diff` lines: one
added `values:` member, and one modified `bound:` and one modified comment at two lines each. The
selector class reports the first, the Bounds class the second, and `2 other lines changed` is the
comment — counted in `git diff` lines because the row names the command.

`TOTALS` counts **rows**: 4 changes, of which 1 is a tombstone row. Three Tombstones were written.
See finding **#13**.

### `hyper changes sync-ci-keys --json`

```
{"type":"window","procedure":"sync-ci-keys","baseline":{"run":"019fbcb2-e46a-79c2-82e5-b7a41c0d3f19","trigger":"igor@thinkpad","started":"2026-08-01T09:41:08Z","outcome":"completed","procedure_revision":"9e0974802fd6592f3df7035f6520e66e17bc9611"},"subject":{"run":"01a004b3-6bb5-74d1-81a7-0c93be5482d6","trigger":"igor@thinkpad","started":"2026-08-15T09:14:22Z","outcome":"completed","procedure_revision":"9843c17157342973d078a863ab73ecc91ebbd8e4"}}
{"type":"asset","change":"created","target":"tailscale-prod","definition":"ci-keys","name":"ci-arm64-2","to_ordinal":1,"fields":{"created":"2026-08-15T09:14:30Z","description":"ci-arm64-2","expires":"2026-09-14T09:14:30Z","id":"kT8mLd3CNTRL","key":"<secret>"}}
{"type":"asset","change":"changed","target":"tailscale-prod","definition":"ci-keys","name":"ci-macos","from_ordinal":4,"to_ordinal":6,"fields":{"created":["2026-07-10T09:33:52Z","2026-08-15T09:14:29Z"],"expires":["2026-08-09T09:33:52Z","2026-09-14T09:14:29Z"],"id":["kM2wHf5CNTRL","kQ4vNp7CNTRL"]}}
{"type":"asset","change":"destroyed","target":"tailscale-prod","definition":"ci-keys","name":"ci-riscv","from_ordinal":7,"to_ordinal":8,"confirmed_at":"2026-08-15T09:14:25.902Z","fields":{"created":"2026-06-20T09:29:14Z","description":"ci-riscv","expires":"2026-07-20T09:29:14Z","id":"kR4nTbC1CNTRL","key":"<secret>"}}
{"type":"asset","change":"changed","target":"tailscale-prod","definition":"ci-keys","name":"ci-x86","from_ordinal":5,"to_ordinal":7,"fields":{"created":["2026-07-10T09:33:55Z","2026-08-15T09:14:27Z"],"expires":["2026-08-09T09:33:55Z","2026-09-14T09:14:27Z"],"id":["kX7pJr9CNTRL","kSbYs2Q1CNTRL"]}}
{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"procedure revision","from":"9e0974802fd6592f3df7035f6520e66e17bc9611","to":"9843c17157342973d078a863ab73ecc91ebbd8e4"}
{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"step issue-runner-keys · bound","from":3,"to":4}
{"type":"code","subject_kind":"procedure","subject":"sync-ci-keys","fact":"step issue-runner-keys · selector","from":{"values":["ci-arm64","ci-x86","ci-macos"]},"to":{"values":["ci-arm64","ci-x86","ci-macos","ci-arm64-2"]}}
{"type":"code","fact":"repository revision","from":"3a4fcac66bed9be2d9830e4e9cbc842d8504815d","to":"b31703fe796ba46511758a7cf118dfc9b789bb6e"}
{"type":"code","fact":"other lines changed","count":2,"command":"git diff 3a4fcac b31703f"}
{"type":"result","truncated":false}
```

`from_ordinal` is absent on the `created` row exactly where the column renders `–`. The `changed`
rows carry every value whole, including any the page would have elided. `changes` is not a Run, so
the stream terminates with `result` and not `outcome`.

### `hyper show 01a004b3-6bb5-74d1-81a7-0c93be5482d6 --expansion`

The layout below is **invented** — §9 states what this command carries, not how it draws it (finding
**#10**).

```
  RUN         01a004b3-6bb5-74d1-81a7-0c93be5482d6
              sync-ci-keys · igor@thinkpad · Sat 15 Aug 09:14 · completed · 9s
  PROVENANCE  hyper 0.4.1 · procedure rev 9843c17 · repo rev b31703f

  STEP 1  retire-expired  destroy  delete_key  tailscale-prod  ran
    provenance   definition rev 03d962a · manifest sha256:4757bb…
    records      ci-macos · ci-riscv · ci-x86
    selector     assets: description starts_with ci- · created older_than 30d
                 = older_than: 30d resolved to 2026-07-16T09:14:22Z
    expanded to  ci-macos · ci-riscv · ci-x86
    bound        5

  STEP 2  issue-runner-keys  mutate  create_key  tailscale-prod  ran
    provenance   definition rev 03d962a · manifest sha256:4757bb…
    records      ci-arm64 · ci-arm64-2 · ci-macos · ci-x86
    selector     values: ci-arm64 · ci-x86 · ci-macos · ci-arm64-2
    expanded to  ci-arm64 · ci-x86 · ci-macos · ci-arm64-2
    bound        4

  STEP 3  issue-bootstrap-key  mutate  create_key  tailscale-prod  skipped as already recorded
    provenance   definition rev 03d962a · manifest sha256:4757bb…
    records      bootstrap-2026   unchanged since run 019fbcb2-e46a-79c2-82e5-b7a41c0d3f19
```

Step 3's entry holds a digest and no members, so `show` resolves them from the Run that last carried
one and **names it** rather than rendering them bare. The relative predicate on Step 1 is glossed with
the instant it resolved to; this and a Refusal's caret are the only two surfaces where that gloss
appears, both being surfaces a Run supplies an instant to.

### What this Run does not render

No job summary — there is no Cadence, so no workflow was projected. No Refusal rendering — nothing
declined. No `THE WORLD MOVED` rows — this repository has no observing Definition. No Pattern account
anywhere: `retry` is declared on both effectful Operations and neither did more than the trivial
single call.

---

## 6. Every place I guessed, and every place I wanted to ask

Ordered by how much the answer would change.

### The four that are decisions

Four of the twenty are questions the corpus does not answer and cannot be answered by editing prose:
each changes what an author may write, and one of them is irreversible once a Store exists. They are
ticket-shaped and are stated here in that form, for the map ([#1]) to carry — one ticket each, none of
them restated by this file once it has one.

| # | The question | Why it is a decision and not an edit |
| --- | --- | --- |
| **1** | May a Procedure declare a Cadence when any Step it reaches produces secret output? | Every projected occurrence Refuses for want of a sink the workflow never supplies. It is ADR-0038's argument on a second door, and the answer is either a new `error_code` or a named limit in §13 — both move a closed set or the limits chapter. |
| **2** | What type does a template hole carry into a JSON `body:`? | Decides whether an API taking a non-string scalar in a body is writable at all. `query:` and `headers:` are stated to be strings always; `body:` is given no such sentence, and schema-directed typing points the other way. |
| **4** | What are the exact canonical bytes an identity digest is taken over? | The Store is append-only, so a reading adopted after the first Run is a reading that cannot be changed. Every Journal entry ever written depends on it. |
| **5** | How does `THE CODE MOVED` render a non-scalar fact? | The `cadence` row is specified in full and its four siblings — selector, Target set, Operation set, `destroy` Operations — are not. ADR-0059's whole-or-`changed` rule is explicitly scoped to the `FIELDS` column and does not reach here. |

The remaining sixteen are corrections rather than decisions: **#9** is two sections of the corpus
contradicting each other, **#3**, **#6**, **#8**, **#10**, **#11**, **#14**, **#15** and **#16** are
questions prose can settle, **#12** and **#13** are consequences the spec gets right and undernames,
and **#7**, **#17**, **#18**, **#19** and **#20** are notes on this document rather than on the spec.

[#1]: https://github.com/TheLoomLabs/hyper/issues/1

### The twenty, in full

**1. A Procedure whose Steps produce secret output cannot carry a Cadence, and nothing says so.** This
decided the artefact. A Step whose Operation declares a secret output Refuses when the invocation
supplies no sink (§9, ADR-0007), and the projected workflow supplies none (§10). So a `cadence:` on
this Procedure would pass `check`, project a workflow, and Refuse at **every** occurrence — the exact
shape `cadence-run-once` exists to refuse one aisle over, with the same *nobody is present to read the
Refusal* argument (ADR-0038), and with no check catching it. I dropped the Cadence. **Is this
intended, and does it want a `cadence-secret-sink` member?** If it does, the check is as cheap as
ADR-0038's: both facts are authored and offline.

**2. The type of a template hole inside a JSON `body:`.** `expirySeconds: "{expiry_seconds}"` must be
quoted in YAML (ADR-0023) and the input is declared `{type: integer}`. Nothing states whether the
serialised body carries `2592000` or `"2592000"`. I assumed the input schema's declared type governs
the JSON, since schema-directed typing is the format's rule everywhere else — but `query:` and
`headers:` are stated to be "mappings of name to string, always" and `body:` is given no such
sentence. **This one is worth pinning down: an API that rejects a stringified integer makes the
Manifest silently wrong in a way `check` cannot see.**

**3. Whether `input:` may be omitted on an Operation that takes no inputs.** §3 lists exactly three
per-Operation facts as stated by omission — `repeatability:`, `concurrency:` and Record cardinality —
and `input:` is not among them. I dodged it by giving every Operation a required `tailnet`, so the
question never arose in the artefacts above. It will arise on the first Operation that genuinely takes
nothing.

**4. The canonical JSON bytes an identity digest is taken over.** §7 fixes it as "the canonical JSON
encoding of the sorted array, trailing LF included". I read that as two-space indent, one element per
line, no trailing comma:

```
[
  "ci-macos",
  "ci-riscv",
  "ci-x86"
]
```

Every digest in this document is `sha256sum` over exactly those bytes and is re-checkable. A different
reading — no indent, or a single line — changes every digest in every Journal entry ever written, and
the Store is append-only. **Worth an unambiguous example in §7.**

**5. How `THE CODE MOVED` renders a non-scalar fact.** The `cadence` row's stacked expression /
phrase / rate form is stated in full; nothing states how a `selector`, a `Target set`, an
`Operation set` or a `destroy` Operations list renders in `FROM` and `TO`. ADR-0059's whole-or-
`changed` rule is explicitly scoped to the Asset/Observation `FIELDS` column, so it does not reach
here. I rendered the selector as a one-line `values: a, b, c` and put the structure on the wire. A
twenty-member `values:` list, or a five-conjunct predicate, has no stated rendering at all.

**6. Whether `field:` at a Record root resolves against the Provider's whole field set or the bound
Operation's.** My `destroy` Step's selector names `description` and `created`, and `delete_key`
projects nothing whatever — a `destroy` carries no `record:` by rule. §4 says the check is "against
that Provider's `fields:` mapping", which is Provider-level and makes this legal; under the narrower
reading no `destroy` Step could ever carry a predicate at all, which would remove the tool's stated
purpose. I took the Provider-level reading and believe it is right, but §3's phrasing ("one key of the
Manifest's `fields:` mapping") is doing work a reader could miss.

**7. Nested mappings and lists inside `body:`.** §3 says `body:` "is a mapping serialised as JSON" and
does not say whether it may nest. Tailscale's create-key body is three levels deep with a list of
strings at the bottom, so I assumed yes. If nesting is not intended, this Provider is unwritable and
joins §13's wall.

**8. The Secret sink's file.** §9 fixes the mode (`0600`), that the path may not resolve inside the
working tree, and that `-` is refused. It does not say what is written into it — one file per Run, one
per Record, what keys, what shape. I named a path and wrote nothing about its contents.

**9. `--json` abbreviation is self-contradictory in the spec.** ADR-0047 says `--json` "abbreviates
nothing anywhere", and §8's own worked stream abbreviates:
`"definition_revision":"c3a17b0"`, `"manifest_digest":"sha256:2b7e…"`, and `"baseline":"a91f0c2"` on
the `artefact` row. I wrote every revision and digest whole on the wire and abbreviated only on the
page. **I think §8's examples are the bug, but they are what an implementer would copy.**

**10. `show`'s human layout is stated nowhere.** §9 says what the command carries — each Step's
Disposition, its identities, the Pattern account, the failed path, `answered`, and under `--expansion`
the selector, `expanded_to` and the Bound — and §8 states the verbatim form of five other renderings
but not this one. The block above is mine.

**11. Sub-minute durations, and rounding.** §8 renders `1m48s` and `2m31s`. I rendered `9s` and `11s`
for 8.946s and 10.776s, guessing at both the form and the rounding direction.

**12. A destroy-then-recreate pair renders `changed`.** ADR-0058 makes this correct and I am not
disputing it — but on the first realistic Procedure I wrote, it means the Run's most consequential act
against two of its three retired keys is visible in the Comparison only as an ORDINAL moving by two.
`hyper show` has it in full and `records --history` has the versions. **I would want this named in
§13's *nor does a row show a Record's intermediate states* sentence with rotation as the example**,
because rotation is not an edge case — it is what `skip-if-recorded` plus a `destroy` selector is
*for*.

**13. `TOTALS`' tombstone count is rows, not Tombstones.** §8 is explicit that `TOTALS` counts rows
and that the tombstone count is a subset of the asset count, so `1 tombstone` is right for a Run that
wrote three. The line under-reports the destruction by two, on the surface built to report
destruction, and it is correct.

**14. Does an empty change table render its column-header row?** §8 says all three tables render
"header and count" whether or not they hold a row. I rendered `THE WORLD MOVED   0 observations` and
stopped, on the reading that a `CHANGE TARGET DEFINITION …` line above nothing is the empty header the
`AUTHORITY` discussion rejects. The other reading is defensible.

**15. Does the ordinal count Tombstones?** `ci-macos` runs 4 (live) → 5 (Tombstone) → 6 (live), giving
the `4 → 6` above. A Tombstone is an ordinary version of the series (§7), so I believe it does; §8
never says so where the ordinal is defined, and the answer is the difference between `4 → 6` and
`4 → 5` on the row.

**16. Timestamps on a Step that made no call.** §7 says each Step file stamps the instants that Step
began and ended. I wrote a six-millisecond `started_at`/`ended_at` pair on the skipped Step 3. The
alternative — a skipped Step carrying no instants — would need saying.

**17. The Tailscale API facts.** The request shapes and response fields are from memory:
`POST /api/v2/tailnet/{tailnet}/keys` with that `capabilities` body, returning `id`, `key`, `created`,
`expires` and `description`; `DELETE …/keys/{keyId}`; `Authorization: Bearer`. The key ids
(`kT8mLd3CNTRL` and friends) are invented in the right shape. This is precisely the class of error
§4 states it has no oracle for and §13 carries as the first thing `hyper` cannot know — which is
itself the point: nothing in the artefacts above would catch it, and the Run that finds out is a
`destroy` as readily as a `read`.

**18. `hyper.yaml`'s `digest:`.** Only `hyper project` writes it, resolved from a published checksums
file for the pinned version. I copied the shape from §10's example. `version: 0.4.1` matches §10/§11's
projection example; §3 and §7 use `1.4.0` elsewhere.

**19. Dates without years.** `Sat 15 Aug 09:14` follows §8's `Tue 4 Aug 09:12`. A `--since` window
spanning a year boundary renders two rows whose years are nowhere on the line.

**20. What this scenario never reached.** Worth stating so the coverage is not read as complete: no
Observation and no `read` Step (one Definition cannot do both, ADR-0032), so no pagination, no
concurrency limit, no `THE WORLD MOVED` row; no `when:` condition — a reference to a `series` Step is
a load error, so conditioning on the Provider's one `read` Operation is unwritable and would need a
second `one`-cardinality Operation; no nested Procedure, so no Step `path`; no Cadence, so no gloss,
no rate, no projected workflow and no job summary; no `opaque`, no `shell`, no Probe, no Refusal, no
halt, and no percent-encoded or over-long Record name — every name here is `[a-z0-9-]`, and a real
Tailscale key description is free text that would exercise both.
