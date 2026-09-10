# The command reference

Sixteen commands, flat, in one noun group. No aliases and no hidden commands. Every name is a
word [`CONTEXT.md`](../CONTEXT.md) already defines.

This page is the working reference. [§9](spec/10-surfaces.md) is the authority, and where the two
disagree the spec is right.

## The sixteen

| Command | What it does | Its own flags |
| --- | --- | --- |
| **Discovery** | | |
| `providers` | Every Provider this repository can use: the built-in `shell` plus whatever `providers/` holds. | `--limit` |
| `provider <name>` | One Provider's Manifest in full: its Operations, the Kind each declares, the Capabilities it requires, its auth scheme. | |
| `operation <provider> <operation>` | One Operation: its inputs, the request it builds, and the Record it projects. | |
| **The repository** | | |
| `targets` | The Target declarations and what each accepts and grants. | `--limit` |
| **Authoring** | | |
| `check [path...]` | Every static rule, offline, against no credential. The whole repository, or only the paths named. | |
| `review <artefact>` | The artefact in a gutter, with `AUTHORITY` and `FLAGS`, ranged against the last Run that read it. | |
| **Execution** | | |
| `run <procedure>` | A Run of one Procedure. The command that reaches the world. | `--dry-run` · `--secret-out <path>` |
| `probe <provider> <operation>` | One `read` Operation against `local`, through no Definition, writing no Record and no Journal entry. | `--input` · `--response` |
| **Inspection** | | |
| `runs` | The Journal, newest first. | `--limit` · `--since` · `--procedure` · `--target` · `--outcome` |
| `show <run-id>` | One Run's entry in full: its Provenance, and every Step's Disposition. | `--expansion` |
| `changes [procedure]` | The Comparison: one Run read against the previous Run of the same Procedure. | `--limit` · `--since` · `--between <run-id> <run-id>` · `--subject <run-id>` · `--target` · `--kind` |
| `records` | Records: the Heads by default, the versions with `--history`. | `--limit` · `--since` · `--target` · `--definition` · `--name` · `--history` |
| **Lifecycle** | | |
| `install <ref>` | Fetch a Manifest from an `https` ref into `providers/`, against the digest and nothing else. | |
| `project` | Write the version pin and the release digest into `hyper.yaml`, generate the workflow any Cadence declares, and leave an `AGENTS.md` where there is none. | |
| `store init` | Create the `hyper-store` branch. The tree's one noun group, and its one sub-verb. | |
| `compact` | Prune the record to the retention the Repository declaration states. | |

## The three outside the tree

None of these reads a repository and none says anything about `hyper`'s domain, which is also why
the first two are exempt from the version pin gate. `mcp` needs no exemption: starting the server
is not the act, and every tool it goes on to serve passes the gate exactly as its command does.

| | |
| --- | --- |
| `version` | The version of the binary that would act, its commit, and its build. |
| `completions <shell>` | A completion script for `bash`, `fish` or `zsh`. |
| `mcp` | Start the MCP server. Takes no arguments at all. See [`mcp.md`](mcp.md). |

## Configuration flags

Three, on the sixteen alone. Never on the three above.

| | |
| --- | --- |
| `--json` | NDJSON, one row per table row, from the same renderer. |
| `--repo-dir <dir>` | Which repository to act on. `HYPER_REPO_DIR` is the same fact. Without either, `hyper` walks up from the working directory to the git root. |
| `--no-color` | No ANSI. `NO_COLOR` in the environment does the same. |

## There is no help command

`hyper` with no arguments writes the whole command tree on stderr and exits `2`. There is no
`help` command and no `--help`: neither is among the sixteen, and the list is printed rather than
hidden behind one. A word that names no command says where the list is, and a flag a command does
not take names the flags that command does take
([ADR-0094](adr/0094-the-argument-less-invocation-writes-the-tree-and-there-is-no-help.md),
[ADR-0098](adr/0098-an-unknown-flag-names-the-flags-that-command-takes.md)).

## Exit codes

An exit code says what a caller must do to clear the state, never how severe the problem was
([ADR-0061](adr/0061-a-refusal-belongs-to-the-run-not-to-the-step.md)).

| | |
| --- | --- |
| `0` | The command did what it was asked, including a Run whose every Step skipped. |
| `1` | A Run the world resisted, or a command reporting problems it found. `check` lands here with one problem row or a thousand. |
| `2` | A usage error. No Run began: an unknown flag, an unresolvable repository root, a positional matching nothing. |
| `75` | A Run that lost the Store, to the lock, to the sync at Run start, or to a push it could not rebase through. Retryable. |
| `77` | A guardrail declined **before any effect reached the world**. A verbatim retry refuses identically. |
| `130` · `143` | A Run stopped by an interrupt or a termination, having drained: the Step in flight finished, no further Step started, and the Run closed its own entry `failed`. |
