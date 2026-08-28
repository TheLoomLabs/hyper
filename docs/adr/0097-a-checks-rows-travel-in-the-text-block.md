# A `check`'s rows travel in the `text` block

**`check` over MCP carries its rows in the `text` block, beneath the summary line.** §9's asymmetric
table gains a row of its own: *any ordinary return* still carries one summary line, `review` still carries
its page whole, a Refusal still carries its rendering whole, and `check` now carries the line and then
the rows — file, line, field, `error_code` and message, drawn by §8's renderer, exactly as the terminal
draws them. `structuredContent.rows` is unchanged. This adds a channel; it moves none.

## The evidence: 180 tool calls, zero working Procedures

A Claude Code session, headless, 2026-08-28. A repository in the README's quickstart shape — `hyper.yaml`,
a `local` Target granting `read, mutate, destroy` over `shell`, `providers/` absent — with the MCP server
wired at local scope and **no `hyper` on `PATH`**, so this surface was the whole of what the agent had. It
was asked for a snapshot lifecycle: one Procedure writing gzipped tars of three log directories as Assets,
and a second retiring them without reaching anything `hyper` did not create.

**180 tool calls, 76 of them `check`, 73 of them `Write`, and not one working Procedure.** It never got
past the Definition schema, and the task had been chosen to reach the `opaque` `destroy` corner.

What it received from every one of those 76 calls was this:

```json
{ "content": [{ "type": "text", "text": "4 problems" }], "isError": true }
```

It degenerated into a write/`check` oscillation and then began **using the problem count as an oracle**,
writing throwaway artefacts to binary-search the schema one guess per file — `kinds: mutate` as a scalar,
`provider:` on a Definition, `binding:`, `scope:`, `via:`, and a `zzz_bogus_key: 1` control to watch the
count move. Every one of those guesses is answered exactly by a row `check` had already computed and did
not say.

**This was §9 faithfully implemented, not a defect in the code.** The rows went to `structuredContent`,
where §9 puts them, and the client did not surface `structuredContent` to the model — which no client is
obliged to do, and most do not.

## What was decided, and against what

**A `check` reporting problems is a remediation, not a result.** That is the whole of it. §9 already
grants the full rendering to a Refusal for exactly this reason — *with no bypass anywhere, the Refusal
rendering is the entire remediation path* (ADR-0001) — and a `problem` row is the same kind of event on
the return an agent meets far more often: it is `check` an agent calls after every write, and the row it
is not shown is the edit it is being asked to make.

Two arguments stand beside the analogy. MCP's own guidance asks a server returning `structuredContent` to
serialise it into a `text` block as well, for clients that do not read structured content; `hyper` is the
only party that loses by not doing it. And there is nothing new to render — §8's renderer already draws
these rows for the terminal, and this decides which block they are written into rather than how they look.

**Every ordinary return with rows** was rejected. It would put a full table in the `text` block of every
listing, saying twice what `structuredContent` says once, on returns where the rows are an answer rather
than an instruction. **Every return with rows and `isError: true`** was rejected for a narrower reason: it
is keyed on the path rather than the tool, and it would append a Run's page to the summary line §9 composes
for `run` on purpose — an entry an agent reads back with `run_show`, not a remediation it acts on in place.

**The summary line survives, above the rows.** Outcome-first exists because this surface has no exit code
(§9), and a table arriving with no line above it would put an agent back to counting rows to learn whether
it had been told about problems at all. It costs one line. The truncation marker rides on that line, which
is what keeps §9's *a truncated result must never look complete* true once the rows are in the block: the
marker is met **before** the table rather than after it. `check` carries no `--limit` and its command cuts
nothing, so no repository produces that answer today; the rule is about the composition and holds wherever
the row is granted next.

**What the line carries is the bare marker and not its members.** `summary` appends the word *truncated*;
the axis, the two counts and the hint stay in `structuredContent.truncated`, where §9 puts them. A reader
of `content` alone therefore learns that the answer was cut and not how to narrow the question. That is
the pre-existing shape of every ordinary return's summary line rather than something this decision
introduces, and it is left alone deliberately: the tool that would need it is one that cuts, `check` is
not one, and writing a hint into the block for a tool that never emits one would be composing a remedy
against no evidence. The day a cutting tool is granted this row, the members are what to grant with it.

## The seam it is expressed at

The bit on the tool becomes a **case** on the tool — `summaryLine`, `rowsBeneathSummary`, `wholeRendering`
— because what §9's table has is cases and not axes: *the whole page* and *the rows beneath the line* are
alternatives, and a second boolean beside `rendersInFull` would have had a fourth state naming no row of
the table at all. The zero value is the ordinary return, so the eleven tools that declare nothing are the
common case rather than tools that forgot to declare one.

**`check`'s row is keyed on the tool, and what varies under it is the row set rather than the path.** That
distinction is load-bearing. §9's `review` row promises that page *byte for byte on every path the tool
answers at all* — a `review` against an artefact that will not load writes `check`'s table, and the block is
that table — and swapping in a summary line on one of the command's own paths would break the promise
exactly where an agent is least able to check it. `check` promises something different and keeps it the same
way: *the summary line, and beneath it the rows*. A repository with nothing wrong with it has no rows, so
nothing goes beneath the line. Carrying the command's clean page instead — a sentence about a count — would
have this surface state the count twice and call one of them rows.

## Consequences

- **§9's asymmetry table has four rows**, and `check`'s tool block names its `text`.
- **Three corpus cases pin the new block** (`internal/cli/testdata/mcp/check/`): the five-artefact demo, the
  ordering case and the narrowed report all now hold the summary line and the table beneath it, byte for
  byte, beside the unchanged `structuredContent.rows`.
- **The CLI is untouched**, and so is the row set. `error_code`, `field` and the messages are §4's and this
  changes none of them; the terminal and this surface still cannot drift apart, there being one renderer
  behind both (ADR-0026).
- **The orientation's third step becomes true on this surface.** It has always said `check` *answers file,
  line, column and an `error_code`* — advice an agent reading only `content` could not act on until now.
- **The count as an oracle is not a bug this closes by itself.** An agent that cannot see the rows will find
  another oracle; what this decision removes is the reason to look for one.
