# A `review`'s page travels in the structured content

**`review` over MCP carries its rendered page in `structuredContent.rendering` as well as in the
`text` block.** It is the same string on both channels — the gutter, the `AUTHORITY` table, the
`FLAGS` index, byte for byte as the command writes them to stdout — composed once and written twice.
`rows` is unchanged and `review`'s `outputSchema` gains one required member. This adds a channel; it
moves none, and it is ADR-0097's decision read on the other side of the same envelope.

Nothing else on this surface gains the member. The `text` block's asymmetry table is untouched.

## The evidence: three `review` calls, zero pages

The sealed acceptance run recorded in
[ADR-0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md) (issue
#216): a headless Claude Code session, MCP the whole of what it had, thirty tool calls against a
repository in the README's quickstart shape. It called `review` three times. What it was shown all
three times was a JSON rows array.

```jsonc
// what hyper answered, driving `hyper mcp` by hand
{ "content": [{ "type": "text", "text": "  PROCEDURE  │  procedures/say-hello.yaml … FLAGS …" }],
  "structuredContent": { "rows": [ … ], "truncated": false } }

// what the model was given, for that same call
{"rows":[{"type":"artefact","kind":"procedure","path":"procedures/snapshot-logs.yaml", … }],"truncated":false}
```

**This client shows the model `structuredContent` and discards `content` on every return that is not
an error.** The discriminator is `isError`, and the same transcript shows both branches of it: a
`check` that found problems arrived with its summary line and the table beneath it — ADR-0097
working — and every clean return in the session arrived as its rows alone.

The session still finished. It authored a clean Procedure and correctly declined an unsafe one, the
row set carrying the same facts and `operation`'s `source` rows carrying the Manifest's own lines. So
this is not the repair loop issue #214 was: **what was lost is the surface, not the information**. But
`review` is the artefact in a gutter with `FLAGS` indexing into it, and an agent handed the rows is
handed what the page is composed *from*; and `review` is the tool the orientation tells an agent to
call before handing work back, which it cannot do against a page it never sees.

## Whether the rendering is hyper's to guarantee past the `text` block

It is, and the argument is MCP's own rather than this client's.

MCP states its two halves **asymmetrically**, and not in the direction §9 was written against. Of the
structured half: *Servers MUST provide structured results that conform to this schema*, wherever an
`outputSchema` is declared. Of the block beside it: *For backwards compatibility, a tool that returns
structured content SHOULD also return the serialized JSON in a TextContent block* — and the
specification's own example fills that block with the serialised JSON and nothing else.

Read as written, `structuredContent` is **the result** and `content` is a **compatibility
serialisation of that same result**. There is one answer, and the block is a second copy of it for
clients that predate the field. That is the whole of the protocol's account of the pair.

§9 inverted it. It put hyper's strongest promise — *the full rendered review surface, byte for byte on
every path the tool answers at all* — in the channel the protocol calls redundant, and put the copy in
the channel the protocol calls the result. A client that prefers the structured half is not
misbehaving under that reading; it is reading the half MCP made normative, and dropping a block the
protocol told it was a duplicate of what it already had. **On the reading that licenses ADR-0097 in
one direction, this is what it licenses in the other**: a server is the only party that loses by not
writing its answer where the protocol says the answer goes.

So the rule is not *this client drops `content`*. It is: **wherever a tool declares an `outputSchema`,
what that tool promises has to be true of the structured half.** Everything below follows from that
one sentence, and the sentence is keyed on the protocol rather than on any client.

## What was decided, and against what

The rule picks out exactly one tool, because it asks a question about each of §9's four text-block
cases: *can this block be composed back out of the members already in `structuredContent`?*

- **Any ordinary return** — one summary line — yes. The counts are the rows, §12's triple and the Run
  id are keys of their own, the truncation marker is `truncated`. Nothing to add.
- **`check`** — the line, and the rows beneath it — yes. What goes beneath the line is the row set,
  and the row set is `rows` (ADR-0097).
- **`review`** — the page — **no.** The rows are the page's ingredients. A gutter row is a line number
  and a marker; the page is the artefact with the gutter beside it. There is no reconstruction, and
  this is the gap.
- **A Refusal** — the rendering whole, and the retry sentence after it — no, and it needs no second
  channel anyway. **MCP names no structured channel for an error at all.** Its whole error mechanism
  is `isError` and the `content` beside it; a client that dropped `content` there would be dropping
  the only thing the protocol puts there, and would have nothing to show for a failed call. `content`
  is the normative channel on that path for the same reason it is the redundant one on the others.

That last arm is why the decision is **not** keyed on `isError`, which is what the observed client
keys on. Keying on the bit would have been arguing from a client's behaviour; keying on *what the
structured half already carries* is arguing from what each channel is for, and the two happen to agree
here only because the protocol's error mechanism has no structured half.

**Dropping `structuredContent` from `review` was rejected.** It is the obvious other way to make the
page arrive — with no `outputSchema` and no structured result, `content` is the only thing in the
envelope and every client shows it — and it costs the machine-readable half of the surface. A caller
that wants the rows is not hypothetical: §8's row set is one row set on two surfaces (ADR-0026), and a
tool answering a page and nothing else would be the one place a consumer has to parse a rendering to
get at facts the server already had as data. Issue #217 put `rows` out of scope, and this keeps it.

**Writing the block into the structured half on every tool was rejected**, for ADR-0097's own reason
read backwards. There, *every ordinary return with rows* would have put a full table in the text block
of every listing, saying twice what the structured half says once. Here, a `rendering` on every tool
would put a summary line beside the members it is composed of — `"rendering": "3 Providers"` above a
three-element `rows` — which is the same redundancy with the channels swapped.

## The two questions issue #217 raised and this closes without a change

**Whether the summary line matters on an ordinary return.** It does not, on this channel. Outcome-first
exists because this surface has no exit code (§9), and every member of that line is already a key of
the structured half: a client reading only `structuredContent` is told the outcome, the Run id, the
rehearsal marker and the truncation marker by name, and can count the rows it was given. The line is a
composition for a reader of prose. Nothing is lost where it does not arrive, and nothing follows.

**Whether a `check` that finds nothing should say more than `no rows`.** No. On the structured channel
that answer is `"rows": []`, which is the same fact stated in the shape a machine reads it in; on the
text channel the line is `no rows` and the transcript shows an agent acting on it correctly. The
alternative — carrying the command's clean page, a sentence about a count — is the one ADR-0097
already declined for `check`, and this surface would state the count twice and call one of them rows.
The question was worth asking against the first transcript anyone had of that line in use; the answer
is that it was working.

## The seam it is expressed at

**The member is written where the text block is composed, and from the block itself** (`envelopeOf`).
Whatever `answerText` decided the block is, that string is what `structuredContent.rendering` carries
— not a second read of the command's buffer. Two channels carrying one page cannot come to say
different things if there is only one composition, which is ADR-0026's discipline applied inside one
envelope instead of across two surfaces.

**It is keyed on the `textBlock` case and not on the tool's name.** `wholeRendering` is `review`'s
alone today; the rule is that a tool declaring that case declares the member, and
`TestToolSet_TheRenderingMemberIsDeclaredWhereTheTextBlockIsAPage` holds both directions of it — a
tool granted the case without the member fails, and a tool declaring the member without the case fails
too. The next tool whose block is a page arrives with the member or arrives failing.

**The member is `required` in `review`'s output schema**, on the footing `truncated` is required in all
thirteen: an `outputSchema` states what the **tool** answers, and a guardrail declining is §9's own row
standing outside every schema on this surface rather than a path any of them describes. A `review` the
version pin gate declines carries the Refusal in `content`, `rows: []`, `truncated: null` and no
`rendering`, which is the shape every other tool already answers a Refusal in.

**That is a price and it is stated rather than defended.** The premise this decision rests on is
*servers MUST provide structured results that conform to this schema*, and on the Refusal path this
surface does not: `truncated: null` fails `{"type": "boolean"}` on all thirteen tools today, and
`rendering` absent now fails `review`'s `required` alongside it. The non-conformance **predates this
decision** — the Refusal path has stood outside every `outputSchema` here since the schemas were
written — and what this adds is a second member to an existing hole rather than a new one. It is
still a hole, and appealing to it as precedent is not the same as conforming. **The fix is not to
make `rendering` optional**, which would buy nothing: the path would still answer `truncated: null`
against a required boolean, and `review`'s schema would have stopped stating what the tool answers in
exchange. The fix is for the Refusal path to conform on all thirteen at once, which is a ticket of its
own and not this one.

**It stands above `rows`.** A reader of the structured half meets the keys in the order they are
written, and a page beneath a hundred-row array is one met after the thing it exists to be read
instead of. That is the summary line's *outcome first* holding on the other channel, for the same
reason and by the same means.

## The transcript this shipped with

The same harness, the same task, 2026-08-29: `scripts/acceptance/run.sh` against
`tasks/snapshot-lifecycle.md`, sealed, with the MCP surface the whole of what the session had.
Thirty-seven turns, thirty-six tool calls, **five of them `review`**. All five arrived with
`rendering` as the first key of the structured half, carrying the gutter, the `AUTHORITY` table and
the `FLAGS` index — against three calls and zero pages in the run above.

What makes it evidence rather than a smoke test is what the session did with the page. It was asked
for a retire Procedure that *could not reach anything `hyper` did not create*, could not get one, and
said so in its handback by **quoting `FLAGS` back verbatim**:

```
OPAQUE     line 5  step retire  destroy reaches an effect hyper cannot describe
UNBOUNDED  line 5  step retire  an opaque destroy takes no bound
```

That is the whole argument for this decision arriving as behaviour. The rows carried those two facts
in the run above as well, and what the session did with them there was nothing: `review` is *the tool
an agent is told to call before handing work back*, and handing work back means handing over the
surface the human reviews it on. Here it did, unprompted, in the form the human will read it in — and
it used the page to decline a guarantee rather than to claim one.

**The page's arrival is what this decision claims, and that claim is now held by a transcript.** That
an agent reads the page *better* than it reads the rows is a stronger claim and one run does not carry
it; ADR-0099's session reached a clean repository without ever seeing one.

## Consequences

- **§9's return envelope gains a member and its asymmetry table does not change.** The table is about
  `text` and stays exactly as ADR-0097 left it; what §9 gains is the paragraph saying the page is
  written into the structured half as well, and why the other three cases are not.
- **`review`'s published `outputSchema` moved**, which is a change a client can see without any answer
  changing. It gains one required string, so a consumer validating against the old schema still
  validates every member it knew about, and `internal/cli/testdata/mcp/tools.golden` holds the new
  one.
- **Three corpus cases pin the new member** (`internal/cli/testdata/mcp/review/`): the five-artefact
  demo, the gutter-marks case and the artefact that will not load. The last is the one worth reading —
  it is `check`'s table on both channels, which is what *keyed on the tool and not on what the tool
  found* means when it is the same rule read twice.
- **The CLI is untouched**, and so is the row set. There is still one renderer behind the terminal and
  this surface (ADR-0026), and now behind both halves of one envelope.
- **The envelope is larger on `review`**, and by roughly the size of the page. That is the honest cost
  of two channels and it is the cost ADR-0097 already accepted for `check`'s rows; a page a client
  reads twice is cheaper than a page it never reads.
- **`CONTEXT.md` gains no term.** *Rendering* is already the word §8 and §9 use for the thing.
- **What this does not close is whether an agent needs the page.** ADR-0099's session finished its
  task without ever seeing one, and one run in which the page was used well is not evidence that the
  rows would have failed. What is claimed is that §9's strongest promise is now kept on the channel
  the protocol makes normative, which it was not.
- **Three things are left open, each with a ticket.** The Refusal path's non-conformance above is
  issue #219, and it is the surface's rather than `review`'s — `truncated: null` has failed a
  required boolean on all thirteen tools since the schemas were written. Two more came out of the
  2026-08-29 run and are not about this surface at all: the orientation states that a `bound:` is
  *mandatory* on a `destroy` while an opaque `destroy` refuses one (issue #218, a document and a
  binary disagreeing on the channel a user's machine has instead of a specification), and
  `opaque-destroy:` on a Target is a bare boolean, so opting one Definition in opts every Definition
  on that Target in (issue #220, a question rather than a defect — the grant travelling with the
  credential is ADR-0004's decision, and what is asked is whether the grain is right).
