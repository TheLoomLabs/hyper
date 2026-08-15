# A body literal is typed by its spelling and a hole by its input

A request `body:` is a JSON value tree, and it is the one position in any artefact `hyper` holds no
schema for. A **literal** scalar in it carries its YAML 1.2 core type onto the wire; a **template
hole** carries the declared type of the Operation input it resolves to, and only where the hole is the
whole of the value. A hole may not name an input declared `object` or `array`, and may not fill a
mapping key.

## The problem this decides

ADR-0023 states as a consequence that **there is no untyped position anywhere in any artefact**, and
that totality is what makes schema-directed typing work: a scalar's type comes from the schema at the
position it occupies, never from what it looks like, so the Norway problem has no position to arise in.

`body:` is a position with no schema. The shape of a request body is the remote API's, authored per
Operation, and it lives outside this repository — there is nothing for `hyper` to type against. The
corpus never noticed, because the worked Manifests happened not to force the question: every body
written before this decision carried strings.

The first real Provider written against the spec forced it twice in one file. A body wanting
`reusable: false` and `expirySeconds: "{expiry_seconds}"` against `{type: integer}` has two positions
neither of which the format could type — and the second is worse than the first, because ADR-0023
**mandates** the quotes around a leading hole (unquoted, `{expiry_seconds}` is a YAML flow mapping), so
the artefact cannot express the difference by spelling even if an author wants to.

The failure is silent in the direction that matters. An API rejecting a stringified integer makes the
Manifest wrong about the world, which §4 states it has no oracle for — so it is found by a `mutate`
reaching the world, not by a check.

## The decision

**A literal is typed by its spelling.** `false` is a JSON boolean, `2592000` a JSON number, `"2592000"`
a JSON string.

This is not a weakening of schema-directed typing but a statement of its boundary. ADR-0023 rejected
implicit type resolution *because schema-directed typing was available instead*; here it is not
available, and the choice is between the artefact's own spelling and a fixed type. YAML 1.2 core does
most of the work the rejection was doing: `NO` is a string, the booleans are `true` and `false` with
their case variants and nothing else, and there are no sexagesimals. What survives as a cost is the
leading zero — `0755` reads as the integer 755, so an identifier of that shape must be quoted, and
getting it wrong is silent on the wire.

**A hole is typed by the input schema.** The hole's quotes are mandatory and therefore say nothing, so
the schema is the only thing left that can — and it is `hyper`'s own, in the same file, a few lines
below the body. A hole must be the **whole** of its value to carry a type: `"preview-{name}"` and
`" {name}"` are compositions, which have no meaning but a string.

**A hole fills a scalar value position and nothing else.** Not a mapping key (`hole-illegal`), and not
an input declared `object` or `array` (`manifest-inconsistent`).

## Considered options

- **Every body value is a string.** Totality survives untouched and no API taking a boolean or a number
  in a body is writable — a far larger wall than the one §13 already carries for form-encoded and XML
  bodies, and one that catches APIs whose bodies are otherwise trivially expressible. It also produces
  an arbitrary asymmetry once the literal question is settled: an author could hardcode
  `expirySeconds: 2592000` and never parameterise the same position, separated only by a quote the
  parser mandates.
- **A parallel type declaration**, stating a JSON type per body position beside the body. Totality
  survives and the Manifest carries a second representation of one fact that can disagree with the
  first — the shape refused for output schemas, for `opaque`, and for `concurrency:`. It is also the
  only one of the three that an AI author can get wrong in a way no reader can see, the two
  declarations being separated on the page.
- **Reading `query:` and `headers:` as precedent for string-always.** §3 fixes those as *mappings of
  name to string, always*, which is the strongest textual argument for the rejected reading. It does
  not survive being read for its reason: a query string and a header field **are text on the wire**, so
  there is no other type to carry into. That sentence fixes a sink, not a hole, and `body:` is the one
  sink with types. Read as precedent it proves the opposite — §3 states string-always exactly where the
  wire forces it and nowhere else.
- **A hole may carry a whole object or array into a body.** Rejected: it puts the *shape* of the
  request in a Step's arguments, inverting what a Manifest is for. A reviewer reads a request's
  structure off the Provider and its values off the Procedure.

## Consequences

- **ADR-0023's totality claim gains a boundary.** *No untyped position anywhere* is true of every
  position whose schema is `hyper`'s, which is every position but this one. Stating the boundary is
  what keeps the claim honest; leaving it unstated is what let the corpus describe a body it could not
  serialise.
- **The mandatory quote now means two things two lines apart.** `description: "{description}"` is a
  string and `expirySeconds: "{expiry_seconds}"` is an integer, and neither line says which. The wire
  type is a two-place read — the body line plus the input schema — and both places are in one file. The
  quote was never available to mean anything else, ADR-0023 having required it for a parsing reason
  before typing was in question.
- **A stray space changes the wire type.** `"{n}"` is typed and `" {n}"` is a string. This is the same
  trap the leading zero is, and it has the same defence and no better one: it is on the line the
  reviewer reads.
- **The absent oracle is narrowed rather than closed.** `check` still cannot know what the API accepts.
  But the Manifest now *states* what it sends, so an API refusing a stringified integer is a wrong
  `type:` a reviewer reads beside the body — the one face of §13's *whether a Manifest describes the
  API it names* that names its own edit.
- **`error_code` does not move.** The two new refusals land on `manifest-inconsistent` and
  `hole-illegal`, each at a site its existing statement already reaches: a Manifest disagreeing with
  itself in one file, and a hole in a position §12 does not list.
- **Two entries join §13's wall**, both narrow: a body whose top level is not a mapping, and an API
  wanting a caller-supplied object inside one.
- **What joins a value to the type its input declares is undecided**, and is not this decision's to
  make. An authored `args:` value disagreeing with its input's declared type has no code in §4, and a
  value arriving through a reference has no schema to disagree with at all, an Operation's output
  carrying none. Before this decision a mismatch there was harmless, everything being stringified;
  after it, the declared type decides the bytes.
