# Orientation is a handshake field, and `hyper` writes no file to carry it

An agent standing in a repository `hyper` manages learns what `hyper` is from **the `instructions`
field of the `initialize` result**, filled by the server and delivered before the first tool call.

`hyper` writes **no orientation file into the working tree**. There is no scaffolded `AGENTS.md`, no
seventeenth command that writes one, and `project`'s namespace does not widen past
`.github/workflows/` and `hyper.yaml`. A repository that wants such a file gets it the way it gets
every other file an agent authors: somebody asks, the agent writes it, and it lands in a diff.

## The gap

The tool set teaches more than it looks like. Every tool carries a description; `operation` answers
*the Manifest's own lines, verbatim*, which §9 states is deliberate — a Manifest is written in the
format the caller is expected to author Definitions in, so returning it verbatim teaches that format
at the moment the caller needs it; `check` answers file, line, column and an `error_code`, positioned
so that the next act is an edit.

All of it arrives **with a call already in mind**. What is missing is everything above a tool call:
that `hyper` is here at all, that there are five artefacts and what four of them look like, that the
order is author → `check` → `review` → *hand it to a human* → `run`, that three commands have no tool
on purpose, and that a Refusal retried unchanged refuses identically. An agent that has none of those
shells out, or invents a config file, or retries a Refusal until its budget runs out — and each of
those is a failure the rest of this surface was built to make impossible.

A fresh repository has a sixth hole beside them, and it is concrete. The only Provider that ships is
the built-in `shell`, whose request block is `shell: {}`. An agent asked for an HTTP Provider needs
`method`, `host: "{from-target}"`, a path with holes, a `body`, the `auth:` header scheme and the
`record:` projection, and there is no worked example of any of it anywhere in reach. `hyper operation
shell mutate` teaches the wrong half.

## `instructions` is in the protocol, and it costs nothing

MCP's `initialize` result carries an `instructions` field — *instructions describing how to use the
server and its features* — which clients surface to the model. `hyper` set `Capabilities` and nothing
else, so the handshake announced a name, a version and a tool capability, and said nothing.

Filling it puts the orientation **in the protocol**: delivered before any tool call, with no file in
the user's repository, no setup beyond the `.mcp.json` they already write, and nothing for them to
know to do. It is the only mechanism available that a user cannot forget to enable, because enabling
the server *is* enabling it. It reaches every MCP-speaking harness at once rather than one vendor's.

**It is not `hyper` speaking first.** ADR-0021 refuses egress on `hyper`'s own initiative — a webhook,
a notification, a message the tool sends because it decided to. This is a field of the **response to a
request the client made**: the client started the process and asked `initialize`. Nothing is
initiated, and the ADR's own test — *a destination no reviewed artefact named, using a credential the
tool does not hold* — is not engaged at all.

**It is a function of the version rather than a constant**, because the worked example carries a
Repository declaration and that artefact pins which version of `hyper` may act. The version that would
act is the version of the server the client started (§9), so a constant would teach every agent to
author a pin that Refuses the gate on every repository but the one the text was written in
(ADR-0020).

## Why `hyper` writes no file

`instructions` reaches an agent that speaks MCP. It does nothing for one driving the CLI, and the
convention that covers that case is `AGENTS.md` — a file the harnesses read on their own. Something
has to write it, and the answer here is that **`hyper` does not**.

An `AGENTS.md` would not have crossed §9's line by being written. The five reviewed artefacts are the
Manifest, the Target declaration, the Definition, the Procedure and the Repository declaration; a file
that orients an agent is none of them, carries no authority, and nothing about a Run reads it. So
*`store init` scaffolds nothing, `hyper` authoring a reviewed artefact being the line the whole
surface does not cross* is not what refuses this. What refuses it is that **none of the three ways to
write one solves the problem the issue states**, and each costs something real:

- **`hyper project` writes it.** `project` is already the command that writes derived files into a
  tracked path and lands them in a diff, so the mechanism fits. The cold start does not: `project`
  runs against a repository that already holds artefacts, and by the time one does, an agent has
  already authored them without the orientation. It would also widen `project`'s namespace past
  `.github/workflows/` — a §9 change — and put whole-file, always-overwriting, never-merging semantics
  onto a file users will want to edit, which is correct for a generated workflow and wrong for a note
  addressed to a reader.
- **A seventeenth command.** The largest of the three. §9's tree is *sixteen commands, flat, no
  aliases and no hidden commands*, and the count is load-bearing — §9 counts to sixteen throughout,
  and ADR-0088 refused a seventeenth on exactly that ground. It would also be the first name in the
  tree that `CONTEXT.md` does not define. And it still depends on the user knowing the command
  exists, which is the *user has to know to do it* this decision exists to remove.
- **A section in the README the user pastes.** Free, and honest about what it is. It also does not
  reach the cold start: a user who has not read the README is exactly the user this is for.

All three fail the same test, and noticing that is the decision. The cold start is reachable from
**one** place — the handshake — because that is the only channel that opens without anybody choosing
to open it. Everything else is a file somebody has to want, and a file somebody wants is a file an
agent can write for them once it has been oriented. So the README says that, and says nothing about a
format `hyper` would then own.

### What the transcripts qualified, and what they did not

**The handshake is not as unconditional as this decision assumed.** Two harnesses were run against it
(issue #209). Codex delivers a server's `instructions` inside a `tool_search_output` — the model sees
it when it searches for the tools, and an agent that reaches for one without searching does not see it
at all. Claude Code's session log records no system prompt, so its delivery cannot be observed from
the harness's own artefacts. In neither case is *before the first tool call* a property the protocol
guarantees; it is a property each client decides.

That does not revive any of the three options above — every one of them still needs somebody to know
to do something, and `hyper` writing into a working tree is still refused. What it changes is one
sentence of the orientation: it now **tells the agent to offer to write an `AGENTS.md`** where the
repository has none. Harnesses read that file up front, unprompted, whether or not a server is
configured, so the second session in a repository is oriented by a mechanism with no contingency in it.

The line holds because the **agent** offers and the **human** accepts: `hyper` writes nothing, the file
carries no authority, `check` does not count it, and it lands in a diff like every other file an agent
authors. An orientation that told an agent to write one unasked would be this decision's own refusal
re-acquired by way of its prose, which is why the text says *offer*.

## Where the worked `http` example lives

**In the `instructions` text**, as five artefacts written out under the repository paths they belong
at, and held to checking clean by a case that writes them into a repository and runs `check`
(`internal/cli/instructions_test.go`).

The alternatives were to cite a path and to publish a real Provider. A path is unreachable from the
fresh directory this is for. Publishing a Provider is the version that compounds — the teaching
example is also a thing somebody uses — but `install` is deliberately unreachable from this surface,
so a human would have to install it before the agent could read it back through `operation`, and the
registry and release it needs do not exist. Neither is refused for good; the first is refused because
a fresh directory holds no docs, and the second is a thing to do *as well*, later.

**What the example must not become is a second specification.** It is two Providers, two Targets, a
Definition, a Procedure and a Repository declaration — enough shape to author against, carried because
the format cannot be inferred, and not a tour of the thirty-two static codes. The text is paid for on
every session whether the model reads it or not, so anything a tool call already answers stays a tool
call.

**The second Manifest was bought with evidence, and it is the shape of the argument for anything else
going in.** The first transcript run against this text (issue #209) was asked for a multi-host `read`
where the example taught a single-host `mutate`. The agent read the orientation, found it did not
cover the shape it needed, and **disassembled the binary** — `strings`, `objdump --dwarf=info`, raw
byte scans — to recover `host-input:` and `enumerations:`. It never read `docs/spec/`, which was on the
same machine; it went to the executable. That is what an insufficient example costs, and it is not a
cost the prose can pay off: the agent did not need a better sentence, it needed the other request
shape. A third goes in the same way or not at all — a transcript showing an agent hunting for it.

## Consequences

- **The handshake carries a text this repository is on the hook for, and the cases hold less of it than
  it looks.** They hold the **list** — that each of the five facts above is stated at all — and a
  `check` holds the worked example. Every sentence between those is on a reader, and one that drifts
  from §9 is worse than an absent one: an agent believes it, and spends its next turns repairing an
  artefact the tool taught it. This is the accepted cost of putting prose in the binary, and the reason
  the text stays as short as it is.
- **`Instructions` is exported from `internal/mcp`.** The corpus that runs `check` over the example
  lives in `internal/cli`, where the commands are, and the surface must not learn to reach a command
  for itself.
- **The text states runtime semantics, which is a widening of what it is for and is deliberate.** What
  halts a Step, what a Disposition means and what deleting a Definition abandons are not answerable by
  any tool call — they are consequences, and an agent learns them by causing one. The first transcript
  reported a Run that halted on an unreachable host when none had; the agent had read `probe`'s
  narration, which prints `no response arrived` and exits `0`, and generalised it. Facts of that class
  earn their bytes; facts a tool already answers do not.
- **Nothing widens.** No tool is added, no command is added, `project`'s namespace does not move,
  `install`, `store init` and `compact` stay unreachable, and no artefact permits anything it did not
  permit before. What changed is a field in a handshake.
- **`CONTEXT.md` gains no term.** `instructions` is the protocol's word for a protocol's field.
- **Harness coverage is a claim about clients, not about `hyper`.** The field is in the specification
  and the Go SDK carries it; whether a given harness surfaces it to the model varies, and a harness
  that drops it leaves that agent exactly where it was before this decision — no worse. That is the
  accepted cost of putting the orientation in the one place a user cannot forget to enable, and it is
  worth re-checking per harness before more prose is written into the field.
