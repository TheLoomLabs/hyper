# A declared-secret field is suppressed on every surface that renders a projection

A field a Manifest names in `secret:` is written to the Store as the constant marker (ADR-0007), and
is now rendered as that same marker by `probe` — the one other surface that renders a projection.
Where an Operation declares any secret output, the raw response object a Probe would show beside the
projection is **withheld whole**, the marker standing in its place, on the page and in
`probe_result.response` alike.

We chose this because ADR-0017's guarantee was argued from the *request*. A Probe is `read` Kind
against `local`, holds no credential and writes no Record, so there was no secret in the request and
no store for a response to leak into — and the conclusion drawn was that a Probe may surface the raw
response. A `secret:` field is a second position, it arrives in the *response*, and it postdates that
reasoning. An Operation declaring one, probed against a supplied object (ADR-0108) or against an
unauthenticated host that answers with one, put a generated credential in a terminal, in an Actions
log, and in the tool result an agent reads back and carries into a transcript. ADR-0007's claim is
about the Store and was never in question; every other surface was.

The response is withheld whole rather than rendered minus its secret fields because rendering it minus
something is ADR-0017's own second considered option, and the objection it was rejected on has not
moved: the projection is a closed allowlist, and subtracting declared fields from a whole response
body turns that allowlist into a denylist over input `hyper` does not control. The path grammar also
has a reader and no writer, so blanking a position means a second traversal that resolves paths in
order to mutate them — two spellings of one grammar, which is the shape refused at every other
position in this tool.

## Considered options

- **Suppress the projection and leave the response.** Rejected, and it is worse than doing nothing: the
  same value renders two blocks below the masked table and in the same tool result, so what the mask
  achieves is telling a reader the page has been made safe. A half-suppression is a claim, and this one
  would be false.
- **Blank the positions `secret:` names inside the rendered response.** The obvious answer, and the one
  the surface invites. Rejected on ADR-0017's denylist argument above, and on the second traversal it
  needs.
- **Refuse a Probe of an Operation that declares secret output.** Cheap, positional, and in this tool's
  idiom — it refuses rather than redacts. Rejected because it fails the shape it would most often meet:
  one `secret:` field beside twenty ordinary ones is the common Manifest, and an author who cannot
  rehearse the twenty is an author who deletes the `secret:` line to debug and forgets to put it back.
  A guardrail whose cost is paid by removing a different guardrail is not one.

## Consequences

- **An Operation declaring secret output stays probeable.** Its projection renders in full with the
  marker in the declared positions, which is the half of *does this Manifest describe its API* no
  static check can reach.
- **What stands in for the withheld response is `unresolved`.** A response block is read to find out
  why a path did not resolve, and every authored path that resolved to nothing is already named there
  by name.
- **The marker is `artefact`'s constant and not a second spelling.** One marker across the Store and
  every surface, so a reader who has met it once has met it everywhere — the rule §7 states for the
  Store's, applied where the Store is not involved at all.
- **The supplied form suppresses a value the caller already holds.** `--response` reads a file the
  caller wrote, so nothing is being kept from them. What is being kept is the write-back: `hyper` has
  no business copying a secret out of a caller's file into a terminal, a log, or a transcript it does
  not control.
- **A corpus that exercises this is a repository of its own.** A Manifest digest and a repository
  revision are Provenance, so a `secret:` added to an existing fixture repository moves the goldens of
  every case that runs against it.
