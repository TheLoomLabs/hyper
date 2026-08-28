# `project` writes the orientation to `AGENTS.md`, and the handshake is not the only channel

**`hyper project` writes an `AGENTS.md` where the repository holds none**, rendered from the
orientation `internal/mcp.Instructions` already states, at the version of the binary that ran. It is
created if absent and never overwritten, on any run. No artefact is scaffolded, no command is added,
and the file carries no authority.

This **amends ADR-0093 rather than reversing it**. Two of that decision's three refusals stand
untouched. The third moved, because a fact about `project` moved.

## The evidence: an agent recovered the orientation with `strings`

A Claude Code session, 2026-08-28, in a fresh repository built to issue #209's shape — `hyper.yaml`,
one Target, one Definition, one Procedure over the built-in `shell`, **no `providers/`** — with the
MCP server wired at local scope. It was asked for an extension that gets the HTTP status of a list of
websites and says whether they are online.

It did the task well. It authored four artefacts, `check`ed, `probe`d the Operation live, `review`ed
all four, and ran: four sites, four Observations, exit `0`. It never read a neighbouring repository,
and two solved copies in sibling directories went untouched.

**Six of its twenty-eight calls were binary archaeology.** `strings` over the executable, grepped for
`capabilit`, then for the request block's keys, then for `host-input` and `expansion` and `selector`;
one call's own description was *"Dump embedded doc region"*, and what came back was the orientation —
the worked example, `from-target`, `host-input`, the `AGENTS.md` paragraph. Three calls later it wrote
a Manifest that is the orientation's multi-host example **verbatim**, plus one added
`repeatability: repeatable`.

**This does not prove any client drops the field.** Issue #209 already established that the harness's
own transcript records no system prompt, so absence there is not evidence. What it establishes is
behavioural: an agent holding that text in context does not spend six calls excavating it from an
executable and then copy the example it dug out. Combined with #209's Codex finding — delivery
contingent on the model issuing a tool search — the handshake is demonstrably not the unconditional
channel ADR-0093 took it for.

## The shipped fallback was circular

The orientation said: *if this repository has no `AGENTS.md`, offer to write one. Do not assume the
next agent sees this text.* The agent did offer — in its closing line, **after** it had dug the
orientation out of the binary.

A fallback for an orientation that does not arrive, gated on the orientation arriving, fires exactly
when it is not needed and is silent when it is. That is not a paragraph that can be rewritten into
working; the mechanism has to move.

## What ADR-0093 got right, and the one part that moved

Its refusal was never about authority, and it says so: an `AGENTS.md` *"is none of the five reviewed
artefacts, carries no authority, and nothing about a Run reads it."* It refused on the practical
ground that none of three mechanisms reaches the cold start.

Two of the three still do not. **A seventeenth command** still collides with the count ADR-0088
defended, and still depends on the user knowing it exists. **A README section** still needs a user who
has read the README.

The third has moved. ADR-0093 refused `project` partly because *"`project` runs against a repository
that already holds artefacts, and by the time one does, an agent has already authored them without the
orientation."* §9 and the README say the opposite: **`hyper project` is the documented first act on a
new repository.** Once a release exists, `project` *is* the cold start — the exact moment the ADR said
it could not reach.

## Why the write is create-if-absent and nothing else

**`AGENTS.md` is a shared file most repositories already hold**, for reasons having nothing to do with
`hyper`. ADR-0093 is right that whole-file, always-overwriting semantics are *"correct for a generated
workflow and wrong for a note addressed to a reader"*, and that half is kept rather than argued with:
`project` creates the file or leaves it exactly as it stands, on every run, forever.

The absence is tested by the write. `O_EXCL` is the one form in which *is it there* and *claim it* are
one syscall, and the file this is about is one somebody may be authoring in the next pane.

**The bytes are the orientation's own.** A hand-maintained file beside `internal/mcp`'s text would
disagree the first time either was edited, and a reader of the one that drifted has no way to tell
which drifted — the same argument that put §9's tree in one list in `tree.go`.

**No artefact is scaffolded, and that line does not move.** Everything the observed agent dug for —
`host-input`, the request block, `capabilities`, the `record:` projection — is **format knowledge**,
and a fenced YAML block in a Markdown file teaches all of it while granting nothing: it is counted by
no `check`, resolves as no artefact, and confers no authority. A Target or a Definition on disk would
be `hyper` authoring authority, which is the line the whole surface does not cross (§9).

## One text, worded for two readers

The same bytes now reach an agent through the handshake and through a file a harness reads up front,
and the second reader may hold nothing but a terminal. So the text names **commands** rather than
tools — `show`, not `run_show`, which names nothing on a command line — and it puts `install`, `store
init` and `compact` out of reach as **the human's** rather than as *absent from this surface*. The
old wording rested the guardrail on the tool set's shape, and an agent holding a terminal reads *there
is no tool for it here* as permission.

**The text was cut by 28% in the same change**, from 13,191 characters to about 9,440. It is now paid
for twice — once as a handshake field on every session, once as a file every harness reads — so length
is a design constraint rather than a matter of taste. What went was prose, not facts: the second
worked Provider became a four-line fragment showing the single-host request and the `auth:` scheme,
and every fact §9 lists as a thing the orientation must state is still there, the four verbs
included.

Getting to the ~5,000 characters issue #211 measured a hand-written draft at would have meant dropping
the `hyper.yaml`, the Definition and the Procedure from the worked example — which §9 requires it to
carry, and which is a spec change rather than an edit. The reduction stops where the facts start.

## The one case left over, and what the orientation now says about it

A repository that **already holds** an `AGENTS.md` saying nothing about `hyper` gets no file, by the
never-overwrite rule. Neither channel reliably reaches that agent.

So the orientation's paragraph is narrowed rather than deleted: **where an `AGENTS.md` stands and says
nothing about `hyper`, offer to add a section to it.** The agent offers and the human accepts, which
is ADR-0093's own line and the same act as any other file an agent authors. The old sentence — *offer
to write one* — is gone, the file itself now being `project`'s to write.

## Consequences

- **§9's account of `project` gains a paragraph**, and the namespace widens by one path:
  `.github/workflows/`, `hyper.yaml`, `AGENTS.md`. It is the only path in it that is not derived from
  an artefact and the only one that is not regenerated.
- **It writes no row.** §9 fixes `project`'s row at one per workflow, and the declaration has never
  had one either. Both are files that land in the diff `project` is read in, and a row would be this
  command's answer growing a shape §9 does not state (§9, §10).
- **The note goes last in the write order.** It is the only thing written there that no artefact
  derives, so a tree interrupted before it is a repository that projected correctly and lacks a note —
  the reading that costs a reader least, and one that running the command again repairs.
- **It is dormant until the first release.** `project` Refuses `release-artefact-absent` while no
  release is published, so nothing here reaches a fresh directory until `v1.4.0` is cut. `store init`
  was the alternative writer and is refused: §9 says it *"touches no file in the working tree"*, and
  buying a few weeks with that sentence is a worse trade than waiting.
- **Acceptance is a transcript, not a case.** The criterion is an agent in a fresh repository
  authoring a correct multi-host `read` Manifest **without running `strings`**. What the cases hold
  is that the file renders from one source, that a file already standing is not taken, and that the
  example still checks clean (`internal/cli/instructions_test.go`, `internal/cli/project_test.go`,
  and the `tree.golden` of every `project` case).
- **The tree goldens render a fourth place**, and render `AGENTS.md` **by name** where it holds what
  `project` wrote. The bytes are a constant one package states and one case checks; copying them into
every `project` case would move fifty files whenever a sentence was re-flowed. Anything else
  standing at that path renders verbatim, which is what makes the never-overwrite case legible.
- **The text stays in `internal/mcp`**, which is now read by a command that is not the server. It is
  where ADR-0093 put it and where the handshake fills the field from, and a package holding one
  exported constant, imported by both, would be a package invented so that a name reads tidily. The
  dependency runs `internal/cli` → `internal/mcp`, which already existed and is the direction that
  decision fixed: the surface must not learn to reach a command for itself.
- **`CONTEXT.md` gains no term.** `AGENTS.md` is a convention's filename, not a `hyper` concept.
- **ADR-0021 is still not engaged.** Nothing is sent anywhere. This is a file written into a working
  tree the user is standing in, by a command they invoked.
