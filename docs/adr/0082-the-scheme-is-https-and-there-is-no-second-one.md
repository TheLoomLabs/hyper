# The scheme is `https`, and there is no second one

The `http` Capability requests `https://` and nothing else. No artefact chooses
the scheme, no flag overrides it, and there is no plaintext path to configure:
the read the process supplies is a **TLS** dialer, so `tls` is present on every
response that arrived and absent only where none did.

## The problem this decides

Nothing in fourteen sections says how the scheme is chosen.

§12 fixes the `tls` member of the `http` response object as *present where the
scheme was HTTPS* — a conditional whose condition nothing states. §3's `uptime`
Manifest, the one worked example the whole tracer bullet is built on, says
`days_left` is *absent from a version written against a plain-HTTP host*, which
describes a state and names nothing that could reach it. And §3's *The host*
states the three steps by which a request's host is decided — the candidate set,
the grant, the intersection — with no fourth step anywhere for the scheme.

Read together they imply a switch: something, somewhere, decides `http://` or
`https://`, and two sections describe what follows from each answer. Nothing is
that switch. `hosts:` enumerates **hosts** and carries no scheme; `host:` is a
template over those hosts; `{from-target}` expands to a granted host set. There
is no position in any of the five artefacts where the string `http://` could be
written, and no key that could be added to one without reopening a format
milestone 1 froze.

The gap is small and it is load-bearing. An implementation reaching this with no
statement writes whichever scheme its author assumed, and `days_left` — the
member §12 argues for at length, because no artefact could compute one — becomes
present or absent depending on a decision nobody reviewed.

## The decision

**`hyper` requests `https://` and only `https://`.**

Every consequence follows from that one sentence:

- **`tls` is present on every response that arrived**, and absent exactly where
  the object is `host` and nothing else. §12's conditional is now a conditional
  on one branch, which is a fact worth having: `days_left` going quiet means *no
  response arrived*, not *the scheme was different*.
- **The scheme is not a hole, a key, or a grant.** A `hosts:` entry is a host,
  the candidate set is hosts, and the intersection is a host. Nothing about
  reach changes.
- **The read the process supplies is a TLS dialer.** `cli.Process.Dial` answers
  a connection already past its handshake, wired as `http.Transport`'s
  `DialTLSContext`. That is what makes the decision structural rather than a
  branch somebody could add an `if` to: there is no plaintext dialer in the
  binary to reach for, and `internal/capability` holds no TLS configuration of
  its own.
- **§3's plain-HTTP sentence is amended in the same change.** A sentence
  describing a state the tool cannot reach is worse than no sentence: it invites
  a reader to look for the switch that produces it.

## Considered options

- **Extend the `hosts:` grammar to admit a plain-HTTP grant** — `http://legacy.internal`
  beside the bare hosts, or a sibling `scheme:` key on the Target declaration.
  This is the option that keeps §3's sentence true, and it is why it was
  considered at all. Rejected: it reopens an artefact milestone 1 froze in order
  to keep one sentence true, and it puts a **transport** decision in the
  declaration whose subject is **reach** — a grant enumerates hosts and nothing
  finer (§3), and a scheme is exactly the *finer* thing that rule exists to keep
  out. It also makes the grant comparison two-place: `status.hyper.dev` and
  `http://status.hyper.dev` would be one host and two grants, and
  `host-not-granted` would turn on a prefix rather than on a name.
- **A Manifest-level `scheme:` key**, beside `method:` inside the `http:` block.
  Rejected on the same ground one aisle over: it moves the decision to a Provider
  author, so whether a credential crosses the network in the clear would be a
  fact about a Manifest rather than about the repository that granted the host —
  and an Auth scheme always writes a request header (§3), which is the one thing
  a plaintext transport would hand to anybody watching.
- **Infer it: `https` unless the host resolves to a loopback or private
  address.** Rejected outright — it is reach deciding itself from data, which
  ADR-0029 refuses for redirects, and it would make the response object's `tls`
  member depend on DNS.
- **Say nothing and let the implementation pick.** The status quo, and the reason
  this ADR exists. It leaves §12's conditional unresolvable from the corpus and
  makes the first Provider written against a plain-HTTP host a bug report rather
  than a wall entry.

## Consequences

- **A plain-HTTP endpoint is not reachable, and that joins §13's wall.** It is a
  real cost and a narrow one: an internal service on `http://` is called through
  a `shell` Step or not at all. It is the same shape as the wall entries already
  there — a form-encoded body, an XML response, an API paginated by a URL it
  hands back — a capability declined because admitting it would cost more of the
  format than it buys.
- **`tls` gains a second reading.** It was *present where the scheme was HTTPS*
  and is now, equivalently, *present where a response arrived*. §12's sentence
  stays as written: it is still true, and it is the sentence that says what the
  member is about. What changed is that its condition is no longer a question a
  reader has to go looking for an answer to.
- **A self-signed or expired certificate is a call that got no answer.** The
  handshake fails, so the response object is `host` and nothing else — the same
  absence a refused connection produces, on the rule §12 already states for the
  three causes it enumerates. No member says which of them it was (ADR-0017).
- **The golden corpus needs a certificate, and mints one.** A case's TLS server
  is minted against that case's own instant and the client verifies against the
  same clock, which is what makes `tls.days_left` a checked-in constant read off
  a real chain rather than a number a fixture wrote down.
- **No closed set moves.** No `error_code`, no Capability, no scalar type, no
  response-object member. The decision removes a branch rather than adding one,
  which is the rarer direction and the reason it is cheap.
