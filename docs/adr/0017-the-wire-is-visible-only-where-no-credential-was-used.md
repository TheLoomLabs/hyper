# The wire is visible only where no credential was used

A **Probe** may surface the raw response beside the projection `hyper` derived from it. Against a
credentialled Target, nothing does — there is no `--dump-wire`, no verbose mode that prints a response
body, and no debug channel that reaches the transport.

We chose this because ADR-0011's data layer records only what a Manifest **projects**, and that is the
whole reason declared-only redaction is sufficient: there is no catch-all bucket for an undeclared
token to hide in. A wire dump puts the bucket back, on the surface with the widest blast radius — a
terminal, or an Actions log that is retained and readable. A Probe is the one place the hazard is
absent rather than mitigated: ADR-0009 makes it `read` Kind against `local`, holding no credentials
and writing no Record and no Journal entry, so there is no secret in the request and no store for the
response to leak into. That is a door rather than a loophole, on the same reasoning ADR-0009 already
used.

**Amended by ADR-0142:** this said a Probe may surface the raw response, full stop, and it was written with the
*request* in view — a Probe holds no credential, so the one thing `hyper` must not render was absent
by construction. A `secret:` field is a second position and it arrives in the **response**. An
Operation declaring one, probed against a supplied object (ADR-0108) or against an unauthenticated
host that answers with one, rendered that value in the FIELD/VALUE table and in
`probe_result.response` — a terminal, an Actions log, and the tool result an agent reads back. So the
projection now carries the constant marker in the position, as a Record's does (§7), and **where the
Operation declares any secret output the response object is withheld whole**, the marker standing in
its place. Withheld rather than blanked at the positions `secret:` names: that is the second option
below, rejected there on the guarantee and rejected still, and the path grammar has a reader and no
writer besides — blanking would need a second traversal of it, which is the second spelling this tool
declines everywhere else.

## Considered options

- **A global verbose or debug flag that prints requests and responses.** The obvious answer, and the
  one every HTTP tool ships. Rejected on the redaction argument above, and because ADR-0014 leaves
  three globals that are presentation only — a flag that changes what leaves the process is not
  presentation.
- **Dumping only fields the Manifest did not declare secret.** Rejected: it inverts the guarantee.
  Declared-only redaction is safe precisely because the projection is a closed allowlist; applying it
  to the whole response body makes it a denylist, and a denylist over hostile input is the advisory
  scan this project refuses to perform anywhere else. *Revisited by ADR-0142*, which closed the
  same hole from the other side, and it stands: what changed is that the object is withheld whole
  rather than rendered minus something, which needs no denylist and reads no position.
- **A scrubber over the response body.** Rejected for the same reason ADR-0007 rejected scrubbing
  credentials out of requests: suppression works positionally, and there are no positions here.

## Consequences

- **A projection failure must say what failed, precisely.** With no wire to inspect, "the Manifest says
  the identity is `.data.items[].id` and I recorded nothing" is the authoring failure mode, so a
  failure of that shape names the path that failed to project — positional, as ADR-0007 is positional,
  rather than a scan. *Amended:* this said the failure also carries a closed `error_code`, written
  before the spec fixed what that set holds. An `error_code` is the identifier of a check that
  declined before any effect reached the world, and a projection is read from a response that has
  already arrived, so there is no check to name; what the argument here needs is the path, and the
  path is what the Run halts with and the surface carries.
- **Authoring against a public unauthenticated API is now the tightest loop on the map**, joining the
  static Manifest checks and `check`/`review`. Authoring against a credentialled API is exactly as
  hard as it was.
- **An Operation that declares secret output is still probeable, and its projection still legible.**
  That is what the amendment above buys over refusing such a Probe outright: one `secret:` field
  beside twenty ordinary ones is the common shape, and an author who cannot rehearse the twenty is an
  author who comments out the `secret:` line to debug. What they give up is the response block, and
  what stands in for it is `unresolved`, which names every authored path that resolved to nothing.
- **This does not close the standing question of whether a Manifest correctly describes its API.** It
  makes the cheap half cheap and leaves the rest where it was.
