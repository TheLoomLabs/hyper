# The build sequence

`docs/spec/` says what `hyper` does. It is organised by subject and deliberately
not by build order — §18's rule was *the spec is the what, the ADRs are the why*,
and neither is *what to build first*. This file is that third thing, and it is the
only place it is written down.

It is a **sequencing document, not a specification**. It adds no decision, states
no behaviour, and is not authority: where it disagrees with `docs/spec/`, the spec
is right and this file is stale. Nothing here may be implemented from — every
milestone's content lives in the sections it names.

## Why this exists

The corpus is about 270k tokens — `docs/spec/` at 581 KB, `docs/adr/` at 443 KB,
`CONTEXT.md` at 15 KB. No session can hold it, so no session can turn the whole
spec into tickets in one pass. Each milestone below names the slice a session
actually needs to read, which is what makes `/to-spec` and then `/to-tickets`
runnable against it.

Section numbers are the spec's own: file `NN` carries heading `§NN-1`, so §3 is
`docs/spec/04-the-authoring-format.md`. `docs/spec/README.md` states the mapping.

## The sequence

| # | Milestone | Demoable end state | Reads | Blocked by |
| --- | --- | --- | --- | --- |
| 0 | Binary and gate | `hyper version`, `completions <shell>`; the version pin gate Refuses on skew | §9 (the tree), §11 (the pin) | — |
| 1 | Load and `check` | `hyper check` over a repository of five artefacts, rows out, exit 0/1 | §3, §4, §12 (most of it) | 0 |
| 2 | Discovery | `providers`, `provider <name>`, `operation <p> <o>`, `targets` | §9 (Discovery, The repository) | 1 |
| 3 | The review | `hyper review <artefact>` — the gutter and the four renderings | §8 (the Definition review) | 1 |
| 4 | The Store | `store init`, `compact`; canonical JSON, paths, the Head, the Journal | §7, §9 (Lifecycle) | 1 |
| 5 | The read Run | `probe`, then `run` over a Procedure of `read` Steps; Observations land in the Store | §5, §6 (the `read` half) | 1, 4 |
| 6 | The effectful Run | `mutate` and `destroy`: the Bound, Repeatability, Tombstones, Assets | §5, §6 (the rest), §7 (the Asset half) | 5 |
| 7 | The `shell` Capability | argv exec, the process group, the deadline, the built-in Provider's six Operations | §3 (the command), §6 (execution) | 5 for `read`, 6 for effectful |
| 8 | Comparison and inspection | `changes`, `records`, `runs`, `show`; the NDJSON row stream | §8 (the rest) | 3, 6 |
| 9 | Cadence and projection | `project` writes the workflow; the gloss; the job summary | §10 | 6 |
| 10 | Distribution | `install <ref>`, digest verification, `origin:` | §11 | 1 |
| 11 | MCP | the thirteen tools, the return envelope, long Runs | §9 (the MCP half) | 2, 3, 8 |

Every milestone also reads `CONTEXT.md` and the ADRs its sections cite. §12 is a
reference rather than a milestone — most of them read part of it. §13 is not a
milestone at all: it is the list of things that must **not** work, and belongs in
each milestone's spec as an out-of-scope section rather than as work.

## What the order is for

**Milestones 1 and 3 are the thesis, and they finish before anything reaches the
world.** *Nothing reaches the world unreviewed* is delivered entire by `check` plus
`review` — both offline, both credential-free, neither reaching a network. §9 fixes
that `review` "runs offline against a repository whose Store is unreachable", which
is why it sits at 3 rather than waiting on the Store: the differentiating claim is
demoable before a single HTTP request exists in the codebase.

**Milestone 5 is the tracer bullet.** Artefact → `check` → call → projection →
Record → Store, every layer, one thin path. Build it against `uptime`, which §3
renders in full: no credential, one Operation, `class: local`.

## Two that will not fit one `/to-tickets` pass

Split these before ticketing, or the slicing happens on a degraded window.

- **Milestone 1** carries §3 (63 KB) and §4 (25 KB) and most of §12. Split three
  ways: the YAML subset and the loader (the reading rule, names, resolution); the
  five artefact schemas and the request; then the rules — §4's thirty-one static
  codes. The first two are prefactoring for the third.
- **Milestone 8** reads §8, the largest section in the spec at 113 KB. Split by
  rendering: the Comparison's three tables, the Step table, the Refusal and the
  terminal line, and the row stream.

## Two dependency notes the table does not show

- **Milestone 4 barely depends on 1** — only enough loader to read `hyper.yaml`'s
  `retention:`. That is the seam if you want two tracks running at once.
- **Milestone 7 straddles 5 and 6** rather than following either. The cleanest cut
  is one `read` slice inside 5 and one effectful slice inside 6, not a milestone of
  its own.

## How to work it

One milestone per pair of sessions.

1. **A spec session.** Read only the sections the milestone names, plus
   `CONTEXT.md` and the ADRs they cite. Run `/to-spec`. It publishes a spec issue
   to the tracker (`TheLoomLabs/hyper`, GitHub — see `docs/agents/issue-tracker.md`).
   Cite §13's relevant limits in its Out of Scope section.
2. **A ticket session.** Fresh window. Run `/to-tickets <that issue>`. It publishes
   tracer-bullet issues with native blocking edges and the `ready-for-agent` label.
3. **An implement session per ticket.** Fresh window each time — `/clear` between
   them. Run `/implement <ticket>`. It drives `/tdd` internally and closes with
   `/code-review`.

Do not run `/to-spec` or `/to-tickets` against `docs/spec/` whole. It does not fit,
and the spec's sections are layers where a tracer bullet is a slice through all of
them.

## Provenance

Written when the wayfinder map
([#1](https://github.com/TheLoomLabs/hyper/issues/1)) reached an empty frontier —
fifty-four decision tickets closed, none open. The map's destination was a spec, and
its Out of scope section states that building the tool is the effort that follows
this one. This file is the hand-off between them.
