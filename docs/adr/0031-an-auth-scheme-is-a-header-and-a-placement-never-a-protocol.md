# An Auth scheme is a header and a placement, never a protocol

An Auth scheme decorates a request `hyper` was already making, by writing one header. It does not fetch,
exchange, refresh, or sign, and it does not reach outside the request it is decorating. The set is two
members — `header:`, parameterised by the header's name and an optional literal prefix, and `basic:` —
and it is fixed at two by that sentence rather than by a judgement about how many schemes are enough.

We chose this because the whole reason the set is closed is that `hyper` must know, mechanically, which
position of a request carries the secret, so a credential is suppressed by where it sits rather than by
scanning a rendering for something that looks like one. Every scheme a competent implementer would reach
for unaided breaks that in a different place: OAuth2 client credentials by making a request nobody
declared, SigV4 by making the secret a computation over the whole request rather than a value in a
position, a query-parameter API key by putting the secret somewhere `hyper` cannot suppress at all —
every proxy, gateway, and access log on the path records a URL, and none of them is ours.

A header and a placement is what survives all three. The position class is `hyper`'s; the header's name
is the Provider author's, which is a fact about the API they know and a Definition author would be
guessing at; and the value is the environment's, resolved once per Run and written nowhere. That is the
same *declare, grant, resolve* shape the rest of the model already uses, and it is why parameterising
the header name does not reopen anything: `hyper` wrote the header and knows it by name, so suppression
stays as mechanical as it was when the name was fixed.

## Considered options

- **A member per vendor idiom** — `bearer`, `api-key-header`, `pve-api-token`. Impossible to keep
  closed, since every vendor's header name would be a member, and it hides that they are one placement
  with three parameterisations. `bearer` surviving beside a parameterised `header:` was rejected on the
  same ground the rest of this spec rejects a second representation of a declared fact: the two spellings
  produce a byte-identical request, and swapping between them would render as a change while nothing
  moved.
- **A scheme that obtains its own credential** — OAuth2 client credentials, token exchange, refresh.
  This is what every auth library does and it is the reading an implementer reaches unaided. It puts an
  HTTP call into the tool that no Operation declared, no Bound counts, no Disposition records, and whose
  host is named in a scheme parameter rather than in anything a reviewer reads — inside a tool whose
  claim is that nothing reaches the world unreviewed.
- **Request signing.** The one rejection that is a genuine loss rather than a design objection, and it
  has a real property in its favour: under SigV4 the raw secret never leaves the process, only a derived
  signature does. It is rejected because a signature is a computation over the entire canonical request,
  which makes the scheme a protocol implementation rather than a placement, and because a `region:` and
  `service:` parameter would have to agree with a `host:` nothing could check them against. The cost is
  named where costs are named: no hyperscaler API is reachable.
- **A secret in a query parameter, in URL userinfo, or in a body field.** Rejected for a guarantee it
  buys rather than a principle it offends: with the credential confined to a header, and a credential
  slot legal nowhere but a Target declaration's `auth:`, *no secret ever appears in a URL* is
  mechanically true of every request `hyper` makes.
- **A client certificate.** Not a request position at all, so it cannot join a set defined by request
  positions without redefining the set — and its private key has no home, `hyper` having no filesystem
  Capability and no store of its own.
- **`none` as a third member.** A scheme is a way of authenticating a request, and not authenticating one
  is not a way of doing it. `auth:` is optional instead, its absence rendered as `none` wherever a
  Provider's auth renders, which is the *undeclared default* shape `repeatability:` already has.

## Consequences

- **A scheme's parameters are literals and admit no hole.** They were a Capability-relevant position
  while a scheme might name a host of its own; with that gone, the position does not fall back to the
  default of resolving to an Operation input, because that would let a Step's arguments choose the header
  a credential lands in. It becomes the one position in the format where a hole is refused outright.
- **A Capability-relevant position is exactly one**, an Operation's `host:`. This narrows ADR-0029's
  fourth consequence, which named two while a scheme could reach somewhere.
- **A scheme may not name a header `hyper` computes.** Five are reserved. `Host` is the one that makes it
  a guardrail rather than hygiene: it is derived from the value the Target's grant was checked against,
  so a scheme setting it would dial a granted host while claiming another.
- **A Target holds one credential per scheme, and two credentials are two Targets.** Slot names belong to
  the scheme, so one declaration cannot carry two different secrets for the same scheme. This is the
  domain model asserting itself rather than a limitation: a Target is the unit of both blast radius and
  credentials.
- **A bundled credential is suppressed whole.** Where a vendor packs an identifier and a secret into one
  string, the identifier is suppressed with it and nothing renders which token a Run used. Separating
  them would need a Manifest composing two slots into one value, which is the Manifest taking the
  placement this decision gives to the scheme.
