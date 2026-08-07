# `local` grants enumerated hosts like any other Target

`local` is reserved because it holds **no credentials**, not because it reaches everything. It carries
a Target declaration like any other Target, enumerating the hosts it grants, and an Operation reaching
a host outside that set Refuses. "Check these fifty preview URLs" is one Target declaration listing
fifty hosts, reviewed and diffable, with the Procedure's Step expanding over a literal list of those
same hosts. There is no unconstrained-reach Target in `hyper`.

We chose this because the alternative is a wildcard, and the extension model's central rule is that a
Manifest's declared Capabilities must equal the Capabilities `hyper` derives from it — *exactly, not by
containment*. A grant of "the public internet" makes that comparison vacuous on the one Target that
looked harmless enough not to need it. The deciding argument is sharper than symmetry, though: an
unconstrained-host `read` is an exfiltration channel the moment a host arrives from a Record rather
than from the artefact. Data read from the world would then decide where the next request goes, which
inverts the thesis — nothing reaches the world unreviewed — on the surface least likely to be
examined. Confining every host to an artefact makes "which hosts may this repository reach without
credentials" a reviewed fact, which it was not previously anywhere.

This amends the glossary. `Local` had read *"the reserved Target meaning this machine and the public
internet"*, and the second half is the wildcard the format no longer has.

## Considered options

- **`local` grants `http` to any host.** The reading the original glossary entry implied. Rejected on
  the exfiltration argument above, and because it would be the only place in the system where a
  Capability check passes without a set to check against.
- **One Target declaration per monitored host.** Exact, and it preserves every rule without amending
  anything. Rejected as unusable at the scale the use case actually has: fifty URLs would be fifty
  files, and the thing being reviewed — the list — would be spread across all of them rather than
  readable in one place.
- **Allowing a Manifest to declare a free-form host for `read` Operations only.** Kind is a claim
  about the world and Capability is mechanical; softening the mechanical one for a Kind that is
  nonetheless a request leaving this machine trades the property that is checkable for the one that is
  not.

## Consequences

- **`from-target` denotes a Target declaration's granted host set, not an endpoint.** A self-hosted
  Provider is simply that set with one member. Where the set has one member `hyper` fills it; where it
  has more, the Operation declares an input marked as the host and `hyper` checks membership. One rule
  with one conditional, replacing what would otherwise have been two meanings for one name.
- **The membership check is static**, because every host is authored rather than data-derived. It is
  the same comparison as the literal-list check below, not a second mechanism, and it runs offline
  with no credentials.
- **`over:` gains a literal form, which the safety model already required.** Expansion takes
  `assets:`, `observations:` (read-only) and `values:` — a literal enumerated list authored in the
  Procedure. This is not a new permission: the glossary's **Expansion** entry has always said that
  anything `hyper` did not create must be named by literal identifier before it can be changed. The
  form is what was missing. `check` can then verify offline that a `values:` list is a subset of the
  Target's granted host set — the fifty URLs and the fifty grants have to agree, member for member.
- **The list enumerates hosts, not URLs.** The path lives in the Manifest. The reviewed artefact and
  the grant therefore compare literally rather than by parsing.
- **Dynamic host discovery is not authorable at all.** A Procedure cannot read a list of endpoints
  from the world and then call them. This is the wall again, taken deliberately: the population may
  come from data, but the *reach* may only come from an artefact.
