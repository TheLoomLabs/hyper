# A Probe is bounded by the grant it binds

A Probe's host is checked against the `hosts:` the Target named `local` declares, exactly as a Step's
is, and a host outside that set is refused before the request goes out. A repository declaring no `local`
grants no host, so a Probe there reaches nothing.

We chose this because the alternative is the exfiltration channel ADR-0024 was written to close, sitting
on the surface an agent drives. ADR-0009 exempts a Probe from a Definition, and §9 says the two keys are
vacuous against `local` for want of a claim to intersect — from which the unaided reading follows that
nothing checks a Probe at all. A Probe's `--input` is the only place in `hyper` where a value arrives at
invocation, an Operation may name an input as its host, and a Probe is reachable from MCP: under that
reading `probe uptime check_http --input host=elsewhere.example.com` reaches any host in the world and
ADR-0017 returns the raw response. §9 already asserts that nothing a Probe carries can widen what a
reviewed artefact permits. This is the mechanism that makes the assertion true; without it the sentence
was a claim about a check nobody had written.

The Kind check being vacuous and the host check being live is not an inconsistency. A Kind claim is the
Definition's, and a Probe has no Definition to make one — there is nothing on one side of that
intersection. A grant is the Target's, and the Target is there.

## Considered options

- **Nothing checks a Probe.** The reading ADR-0009 and §9 leave standing. Rejected on the argument
  above: it puts the one unbounded reach in the system on the tightest-loop surface, where reach would
  come from an invocation rather than from an artefact.
- **A Probe reaches any host, and ADR-0017's visibility is what narrows.** Rejected because it protects
  the wrong thing. Hiding the response would leave the request itself unreviewed and unrecorded, and what
  ADR-0024 refused was reach arriving from somewhere other than an artefact, not reach being reported.
- **A Probe takes a host and `hyper` checks it against nothing, on the ground that a `read` changes
  nothing.** Rejected on ADR-0024's own reasoning: a `read` to an authored host is a request leaving this
  machine, and where the host is not authored it is a channel out of it.
- **A dedicated `error_code` for a Probe declining.** Rejected as a second name for one check. What
  names a Refusal is the check that declined, never the moment or the surface it ran on.

## Consequences

- **The tightest loop on the map costs one reviewed edit first.** Authoring a Manifest against a public
  API means granting its host in `targets/local.yaml` before probing it. This is ADR-0024's *the
  population may come from data, the reach only from an artefact* applied to the one surface that had
  escaped it, and it does not narrow ADR-0017: what is visible is still the wire of a call no credential
  was used on.
- **An absent `local` declines a Probe with the ordinary code.** No host is granted because nothing grants
  one, so the refusal is the grant check's rather than a rule about absence, and the rendering names the
  artefact to author.
- **A Probe still writes no Record and no Journal entry**, so a refused Probe leaves no trace in the
  repository, exactly as a completed one leaves none.
