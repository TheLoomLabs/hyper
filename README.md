# hyper

> **Nothing reaches the world unreviewed; nothing changes unseen.**

An agent writes the artefact; you verify it offline before anything runs, and read exactly
what changed after — including what the agent changed about the artefact.

`hyper` is a tool for AI-authored, human-reviewable infrastructure automation. Its spine is
procedural rather than desired-state: a **Provider** knows how to talk to a kind of system and
exposes **Operations**, a **Definition** is a named, authority-scoped use of one, a
**Procedure** sequences **Steps**, and every invocation acts against a **Target** and produces
**Records**. A Provider is a Manifest — data, never code — so reviewing the artefact is
reviewing the whole of what will run, and every effect it describes is performed by `hyper`
itself from a closed set of Capabilities that only `hyper` defines.

The two clauses cover each other's blind spot, and [§0](docs/spec/01-what-hyper-is.md) is
where that is argued. Neither is accountability alone: one acts on what has not happened yet
and knows nothing about the world; the other accounts for what has happened and stops
nothing.

## Quickstart

Half of the thesis is free to watch — `check` and a review reach nothing outside the
repository, with no credential and no infrastructure. The sequence below goes further and
runs, against the built-in `shell` Provider on your own machine.

**Build a stamped binary.** `hyper` learns its own version from the linker, and a build that
omits the flag reports `unknown` and Refuses the version-pin gate on every repository it
touches. The stamp is not optional — see [`docs/build/releasing.md`](docs/build/releasing.md).

```bash
mkdir -p ~/bin   # anywhere on your PATH
go build -ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=1.4.0" \
  -o ~/bin/hyper ./cmd/hyper
```

**Make a repository.** The Store is a branch of it, so there has to be one.

```bash
mkdir demo && cd demo && git init -b main
mkdir targets definitions procedures
```

**Write four artefacts.** The Repository declaration, a Target, a Definition, a Procedure —
the fifth artefact `check` counts is the built-in `shell` Manifest, which ships inside the
binary.

`hyper.yaml` — which version of `hyper` may act here, and how long Records are kept:

```yaml
kind: repository-declaration
version: 1.4.0
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
```

`targets/local.yaml` — this machine, and what it grants:

```yaml
kind: target-declaration
target: local
class: local
kinds: [read, mutate]
capabilities: [shell]
```

`definitions/host-ops.yaml` — a named, authority-scoped use of the Provider:

```yaml
kind: definition
definition: host-ops
provider: shell
kinds: [read]
targets: [local]
```

`procedures/say-hello.yaml` — the Steps:

```yaml
kind: procedure
procedure: say-hello
targets: [local]
steps:
  - id: greet
    definition: host-ops
    operation: read
    target: local
    args:
      command: [echo, hello from hyper]
```

**Check them.** Every static rule, offline, against no credential.

```console
$ hyper check
checked 5 artefacts: no problems found
```

**Read the review.** This is the surface the thesis's first clause is made of: the artefact in
a gutter, the authority assembled from `definitions/` and `targets/`, and a `FLAGS` index that
states nothing the gutter does not.

```console
$ hyper review say-hello
  PROCEDURE            │  procedures/say-hello.yaml               no baseline — no Store
  ─────────────────────┼────────────────────────────────────────────────────────────────
                       │ kind: procedure
                       │ procedure: say-hello
  envelope ✓           │ targets: [local]
                       │ steps:
  read  opaque  local  │   - id: greet
                       │     definition: host-ops
                       │     operation: read
                       │     target: local
                       │     args:
                       │       command: [echo, hello from hyper]

  AUTHORITY   assembled from definitions/ and targets/
  DEFINITION  TARGET  DEFINITION KINDS  TARGET KINDS  EFFECTIVE  DESTROY OPS
  host-ops    local   read              read mutate   r          —

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  OPAQUE    line 5  step greet  read reaches an effect hyper cannot describe
  ENVELOPE  line 3  ok          no step reaches a target outside [local]
```

**Create the Store, and commit.** `store init` is a human's act — no MCP tool can perform it.
The commit is not ceremony: a Run's Provenance records `repo_revision`, so a Run against a
tree with no commit has nothing to record and fails.

```bash
hyper store init
git add -A && git commit -m artefacts
```

**Run it, and read the record back.**

```console
$ hyper run say-hello
run 01a043df-521e-7a0a-b723-05eaa2bb0588
step 1/1 greet
STEP  ID     KIND  DISPOSITION  RECORDS
1     greet  read  ran          1

completed · exit 0 · run 01a043df-521e-7a0a-b723-05eaa2bb0588

$ hyper runs
RUN             STARTED                   TRIGGER      OUTCOME    CONTESTED  PROCEDURE  TARGETS  HYPER
01a043df-521e…  2026-08-27T15:38:24.158Z  you@machine  completed             say-hello  local    1.4.0

$ hyper records
TARGET  DEFINITION  RECORD                       ORDINAL  RUN             STEP  KIND         TOMBSTONE  ORPHANED  SECRETS  HYPER
local   host-ops    ["echo","hello from hyper"]  1        01a043df-521e…  1     observation                                1.4.0
```

### Three things that will catch you, and why each is a rule

- **A bare `go build` reports `unknown`, and fifteen of the sixteen commands Refuse.**
  `version-pin-mismatch`, exit `77`. The repository pins a version and the gate compares it for
  exact equality against what the binary was stamped with — `hyper` never hashes itself
  ([§11](docs/spec/12-distribution-and-version-pinning.md),
  [ADR-0020](docs/adr/0020-the-hyper-version-is-pinned-by-the-repository.md)). `project` is the
  sixteenth and stands outside the gate, for being the pin's only writer — it Refuses
  `release-artefact-absent` instead, which is the next section. `go install` and a flagless
  `go build` are both unstamped builds.
- **A Run needs a commit.** Provenance is the record of which code produced something, and
  `repo_revision` is one of its members; a `HEAD` resolving to no commit leaves it nothing to
  write, and the Run fails rather than inventing one.
- **A Target's `hosts:` and its `http` Capability go together or not at all.** `hosts:`
  present where `capabilities:` does not grant `http`, *or absent where it does*, is
  `target-inconsistent` ([§4](docs/spec/05-static-verification.md)) — one file, two adjacent
  keys, disagreeing with each other. Correct, and surprising the first time, which is why the
  Target above carries `capabilities: [shell]` and no `hosts:` at all.

### One step here is a workaround, and it is the `digest:`

The documented first act on a new repository is `hyper project`, which writes the version pin
and freezes the digest of the released artefact beside it
([§11](docs/spec/12-distribution-and-version-pinning.md)). It cannot succeed today, because no
release of `hyper` has been published:

```console
$ hyper project
refused: release-artefact-absent
  https://github.com/TheLoomLabs/hyper/releases/download/v1.4.0/checksums.txt answered 404 — publish a release for 1.4.0, or install a released hyper
```

So the quickstart writes `hyper.yaml` by hand, with a placeholder digest. That placeholder is
inert for everything above — the gate compares the *version* and nothing local ever reads the
digest — and it is **not** inert in a generated workflow, where the digest is the line a runner
checks fetched bytes against. `hyper project` on this repository would happily write a
workflow that verifies against sixty-four zeros.

Once `v1.4.0` is cut ([`docs/build/releasing.md`](docs/build/releasing.md)), the first step
becomes `hyper project` and the hand-written pin goes away.

## The two surfaces

One core, two ways in. [§9](docs/spec/10-surfaces.md) fixes that *ergonomics is the whole of
the difference between them*: an MCP tool builds the command line its command would have
received and hands it to the same dispatch, so there is no second place for a guardrail to be
skipped or a Refusal to be reworded.

**Sixteen commands**, flat, one noun group, no aliases and no hidden commands:

| | |
| --- | --- |
| Discovery | `providers` · `provider` · `operation` |
| The repository | `targets` |
| Authoring | `check` · `review` |
| Execution | `run` · `probe` |
| Inspection | `runs` · `show` · `changes` · `records` |
| Lifecycle | `install` · `project` · `store` · `compact` |

Three more stand outside the tree, because none of them reads a repository: `version`,
`completions <shell>`, and `mcp`.

**Thirteen MCP tools**, over stdio, each named for the command it carries:

`providers` · `provider` · `operation` · `targets` · `check` · `review` · `run` · `probe` ·
`runs` · `run_show` · `changes` · `records` · `project`

`run_show` is the one name that differs from its command — a client holds every server's tools
in one flat namespace, where a bare `show` names nothing.

They are the sixteen less `install`, `store` and `compact`, and one line puts all three on the
far side of the boundary: *an agent may read the record and add to it, and may not create it,
prune it, or bring anything new into the repository.* `install` is the single point at which
third-party data enters the repository; `store` creates the record; `compact` is the one
command that would let an agent prune the account it is itself held to.

```json
{
  "mcpServers": {
    "hyper": {
      "command": "hyper",
      "args": ["mcp"],
      "env": { "HYPER_REPO_DIR": "/path/to/your/repo" }
    }
  }
}
```

`hyper mcp` takes no arguments — no `--repo-dir`, no transport flag, no port. The server dies
with its client and offers no asynchronous handle, so it owns the author→validate→observe loop
and short effectful Runs; long unattended work is a Cadence on an executor.

## Where the real documentation is

- [`CONTEXT.md`](CONTEXT.md) — the vocabulary. Every term above is defined there, with the
  synonyms it deliberately avoids.
- [`docs/spec/`](docs/spec/) — the specification, in fourteen sections. It is the authority:
  where the code and the spec disagree, the spec is right.
- [`docs/adr/`](docs/adr/) — ninety-odd records of why, including the options that lost.
- [`docs/build/milestones.md`](docs/build/milestones.md) — the build sequence, and the only
  place it is written down.

**Together they are about 270k tokens, and no session reads them whole.** Do not point a tool
at `docs/spec/` in one pass; it does not fit, and the sections are layers where any one change
is a slice through all of them. `docs/spec/README.md` states which file carries which section.

## What `hyper` deliberately is not

These are decisions, and [§13](docs/spec/14-non-goals-and-honest-limits.md) records each with
its reason. There is **no desired state and no plan** — nothing anywhere renders a proposed
change before it happens, and a Comparison is retrospective by construction
([ADR-0010](docs/adr/0010-hyper-has-no-plan.md)). There is **no query language**
([ADR-0013](docs/adr/0013-hyper-has-no-query-language.md)), **no configuration file** beyond
the reviewed Repository declaration
([ADR-0014](docs/adr/0014-hyper-has-no-configuration-files.md)), **no telemetry** of any kind —
no exporter, no metrics, no trace context, no logging framework
([ADR-0016](docs/adr/0016-hyper-has-no-telemetry.md)) — and **no daemon**: nothing listens on a
port and nothing outlives the invocation that started it. There is **no ad-hoc invocation**:
every Run is a Run of a Procedure, a Probe reaches `local` and `read` alone and is not a Run
([ADR-0009](docs/adr/0009-a-probe-is-not-a-run.md)), and a one-off act against a credentialled
Target is an artefact you have not written yet. There are **no team features** — no accounts,
no roles, no per-Run approval — because who may change what `hyper` does is who may merge a
change to the reviewed artefacts, and a second authority axis inside the tool would be a way
past a Refusal that no artefact records. And `hyper` **never updates itself**
([ADR-0019](docs/adr/0019-hyper-never-updates-itself.md)).

Half the people who would bounce off `hyper` should bounce off it here, rather than three
weeks in. That is what this section is for.

## Contributing, security, licence

[`CONTRIBUTING.md`](CONTRIBUTING.md) · [`SECURITY.md`](SECURITY.md) ·
[Apache-2.0](LICENSE), copyright TheLoomLabs.
