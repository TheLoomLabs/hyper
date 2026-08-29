# A tool that declined answers no structured half

**Where a command declines before it opens a row stream, the envelope is the `text` block and
`isError` and nothing else.** No `structuredContent` key at all — not an empty one, not a null one.
That is MCP's own shape for a tool that failed, and it is the only shape on that path that conforms
to the `outputSchema` the tool published, because there is nothing there to conform.

`run` is untouched: §8 puts it on the `outcome` side on every path a Run was attempted, so a `run` a
guardrail declined goes on answering `outcome: refused`, `dry_run` and its rows. The rule is keyed on
whether the command produced a stream, not on `isError` and not on the exit code.

Beside it, one schema is corrected: `truncated` on `runs`, `changes` and `records` now admits the
marker object those commands actually write.

## The MUST, and whether the Refusal path is meant to keep it

It is, and there is no reading of the specification on which it is exempt.

> If an output schema is provided: Servers **MUST** provide structured results that conform to this
> schema.

ADR-0100 rested its whole argument on that sentence — `structuredContent` is *the result*, `content`
is *the serialized JSON* returned **for backwards compatibility**, so a promise kept only in the
block is kept in the half the protocol calls redundant — and then stated, rather than defended, that
this surface breaks the same sentence on the Refusal path. Appealing to a hole as precedent is not
conforming, and the sentence does not carry an exception for a call that went badly.

**What it does carry is a subject.** The obligation is over *structured results*: what the tool
answers when it answers. MCP's account of a tool that did **not** answer is a different mechanism
under a heading of its own — *Tool Execution Errors: reported in tool results with `isError: true`* —
and the specification's worked example of one is the whole of what it puts there:

```json
{ "jsonrpc": "2.0", "id": 4,
  "result": {
    "content": [{ "type": "text", "text": "Failed to fetch weather data: API rate limit exceeded" }],
    "isError": true } }
```

No `structuredContent` member, and no sentence anywhere saying one is owed. The reference Go SDK
cannot produce one there either: its typed handler path returns the error result before it ever
marshals an output value, so a server built the ordinary way answers `content` alone on every failure
it reports.

So the answer to *is the Refusal path meant to conform* is **yes, and it conforms by having no
structured result** — not by acquiring members that make a schema pass.

## What was actually broken, which is not what the ticket said

Issue #219 said `truncated: null` fails `{"type": "boolean"}` on all thirteen tools. Writing the
fence first said otherwise, twice.

**`run` already conformed.** Its schema declares `"truncated": {"type": "null"}` — a Run reports what
it just did rather than ranging over a namespace, so there is no result set for a limit to have cut —
and on the two paths that decline before a Run is identified it writes `outcome`, `dry_run` and
`rows: []` beside it. Every member its schema requires is there. The hole was the **twelve**, and
naming it thirteen made a defect in eleven listings and one authoring tool look like a property of
the envelope.

**And the Refusal path was not the only hole.** `runs`, `changes` and `records` order on an axis a
`limit` can cut, and where one does they answer §9's marker — `{"axis", "returned", "dropped",
"hint"}` — against a published `{"type": "boolean"}`. That is a non-conformance on a **successful**
call, on the one answer §9 says must never look complete: a client doing what the protocol asks of it
(*clients SHOULD validate structured results against this schema*) was told the server had broken its
contract exactly when the server was working. Four corpus cases held that shape, checked in and
green, because nothing read a golden against a schema.

Both were found the same way, which is the argument for the fence being part of this decision rather
than a test beside it.

## What was decided, and against what

Issue #219 named three shapes. Each fails, and each fails for a reason worth writing down, because
the fourth is only obvious once they have.

**Declare the Refusal shape in every schema** — `truncated` gains `null`, the required members shrink
to the ones a Refusal also writes. It closes the twelve and cannot close `review`: the only way to
stop `rendering` failing a `required` it is absent from is to make it optional, which ADR-0100
refused for a reason that has not changed — the schema would stop stating what the tool answers, and
buy a conformance the path still would not have. It also spends the member's meaning. `null` would
become a value every ordinary call's consumer must handle, in exchange for legalising a half that
carries no information.

**Write conforming members on the Refusal path** — `truncated: false`, and `review`'s Refusal
rendering in `rendering`. §9 refuses `false` on its own grounds: it claims a stream was complete that
was never a stream. And the `rendering` half of it is worse than redundant — MCP names no structured
channel for an error, so it would put a second copy of the Refusal in the one place no client reading
an error looks.

**Drop `outputSchema` where a tool can be declined** — all thirteen can be, so this is *no structured
contract on this surface* wearing a smaller hat.

**What all three have in common is that they treat `{"rows": [], "truncated": null}` as an answer to
be legalised.** It is not an answer. `rows: []` on a `providers` call the version pin gate declined
says *this tool ranged over the namespace and found no Providers*; the fact is that it never looked.
That is not merely unvalidatable — it is **wrong**, and wrong in the direction a consumer cannot
detect, because an empty result and no result are the same bytes. §7's own absence discipline says an
absent key must not read as *unknown* where the fact is *none*; here the key was present and read as
*none* where the fact is *nothing was asked*.

Removing it is therefore not a concession to the schema. The schema was the thing that noticed.

## The seam it is expressed at

**It is keyed on the command's stream** (`structuredOf`): no terminal row, and no execution half, is
the whole condition. Nothing in it reads `isError`, and nothing reads the exit code.

That matters because the two obvious keys are both wrong, and the corpus holds a case against each.
`isError: true` is the shape a `check` reporting problems returns, with rows a consumer needs, and it
is the shape `run` returns on a Refusal that wrote a Journal entry. Exit `77` is the same story: a
`run` the pin gate declined exits `77` and still carries §12's triple. And a `run` that lost the
Store to the lock exits `75`, writes no terminal row at all, and *still* carries an execution half —
§12 fixes what the code carries and the call names the rehearsal, so the members are there without a
row to lift them from (`executionOf`). Read against the stream, all three keep their halves and only
the twelve lose one.

**Rows with no terminal row is a fault rather than a shape.** §8's terminal row is written always,
including after zero rows, so a stream that ends without one is one a consumer must not trust. The
composition refuses it there rather than answering an envelope, which is the same call `envelopeOf`
already makes about an unmapped exit code: a wrong envelope is harder to notice than a missing one.

**`truncated` is a schema fix and not an answer fix.** Nothing about what those three commands write
changed; what changed is that the published schema now admits the two shapes each of them writes —
the bare `false` and the marker — and it is one declaration behind all three, because which shape a
command writes is a fact about what it ranges over rather than a choice (`render.Truncation`). The
bare `true` stays out: it is the namespace listings', which have no axis to name.

The seven tools that answer only `false` keep `{"type": "boolean"}`. Narrowing them to `{"const":
false}` would state more, and it would state it about tools none of which has a cap; the fence holds
what is answered against what is published, and nothing there is unheld.

## The fence

`TestToolSet_EveryAnswerConformsToTheSchemaItsToolPublished` walks every `envelope.golden` in the
corpus and validates its structured half against the `outputSchema` that tool publishes, **with the
validator a client would use** — `github.com/google/jsonschema-go`, which is the library the MCP Go
SDK itself resolves and validates with.

This is the fence the two golden files could not be between them. `tools.golden` holds what a client
is told it may expect and an `envelope.golden` holds what a call answered; until now nothing read the
second against the first, so a member no schema admitted and a schema no answer satisfied both passed.
Both had shipped.

An envelope with no structured half is the one thing it does not validate, and that absence is
checked rather than skipped: it is held to `isError: true`, so an *ordinary* return arriving without
one fails here. Both counts are asserted at the end — an answer validated and an answer declined —
because either at zero is this rule held over nothing.

**The same validator reaches the three answers no case can drive.** A Run that lost the Store to the
lock, to the sync at Run start, or to a rejected push needs a contended repository rather than a
fixture, so those envelopes live in `run_store_lost_test.go` and never in the corpus — and they are
the only `isError: true` answers left on this surface that still carry a structured half.
`assertRunLost` validates each of them against `run`'s schema for that reason, which is also what
holds this decision's claim that `run` conformed all along rather than merely looking as though it
did.

## Consequences

- **Seven corpus envelopes lose their structured half**, across four tools: the pin gate on
  `provider` (three cases, one of them filed under `exemption/`), a Target granting no host on
  `probe` (two), an absent Store on `runs`, and no release under the tag on `project`. Each is now
  `content` and the bit.
- **`review/version-pin-mismatch` is new**, and it is the case #217 left uncovered: a `review` whose
  artefact is present and whose page is withheld by the gate. It is what holds `rendering` being
  `required` honest — the member is absent, and so is the half it would have been absent from. With
  it the corpus holds the shape eight times, across five tools.
- **Three published schemas moved**, and a client can see it without any answer changing: `runs`,
  `changes` and `records` declare `truncated` as `false` or the marker. A consumer validating against
  the old one was already failing on every cut listing, so this widens what validates rather than
  narrowing it.
- **§9's *every tool returns one shape* gains a stated exception**, and the envelope block says so on
  the key itself. A client reading `structuredContent` unconditionally must handle its absence — which
  MCP's own error example already required of it.
- **`mcp.Envelope.StructuredContent` is a pointer**, and the nil crosses to the SDK untyped: that
  member is an `any` with `omitempty` there, and an interface holding a typed nil marshals as `null`,
  which is a structured result claiming to be one.
- **The corpus renders the absence as the wire carries it** — the key is not written at all. A golden
  holding `null` or `{}` would hold the shape this removed.
- **`CONTEXT.md` gains no term**, and neither does §12. Nothing here is a new fact about the domain;
  it is one channel's account of a Refusal the domain already had.
- **ADR-0100's price is paid rather than restated.** *The seam it is expressed at* named this as a
  ticket of its own and left the shape open, weighing two it did not take; the amendment note there
  now says which one it was.
- **What this does not close is whether a client reads the absence well.** ADR-0099's and ADR-0100's
  sealed runs both show the observed client surfacing `content` on `isError: true`, so the Refusal
  arrives; that it arrives *because* the structured half is gone is not a thing one transcript can
  distinguish from it arriving anyway. What is claimed is narrower and checkable: every answer this
  surface gives now conforms to the schema it published, and a fence says so.
