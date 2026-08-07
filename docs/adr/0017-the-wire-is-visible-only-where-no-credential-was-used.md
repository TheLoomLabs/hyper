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

## Considered options

- **A global verbose or debug flag that prints requests and responses.** The obvious answer, and the
  one every HTTP tool ships. Rejected on the redaction argument above, and because ADR-0014 leaves
  three globals that are presentation only — a flag that changes what leaves the process is not
  presentation.
- **Dumping only fields the Manifest did not declare secret.** Rejected: it inverts the guarantee.
  Declared-only redaction is safe precisely because the projection is a closed allowlist; applying it
  to the whole response body makes it a denylist, and a denylist over hostile input is the advisory
  scan this project refuses to perform anywhere else.
- **A scrubber over the response body.** Rejected for the same reason ADR-0007 rejected scrubbing
  credentials out of requests: suppression works positionally, and there are no positions here.

## Consequences

- **A projection failure must say what failed, precisely.** With no wire to inspect, "the Manifest says
  the identity is `.data.items[].id` and I recorded nothing" is the authoring failure mode, so a
  failure of that shape carries a closed `error_code` and the path that failed to project — positional,
  as ADR-0007 is positional, rather than a scan.
- **Authoring against a public unauthenticated API is now the tightest loop on the map**, joining the
  static Manifest checks and `check`/`review`. Authoring against a credentialled API is exactly as
  hard as it was.
- **This does not close the standing question of whether a Manifest correctly describes its API.** It
  makes the cheap half cheap and leaves the rest where it was.
