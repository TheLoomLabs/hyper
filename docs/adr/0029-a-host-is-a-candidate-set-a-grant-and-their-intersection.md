# A host is a candidate set, a grant, and their intersection

An Operation always writes a `host:`, and what a request reaches is decided in three steps rather than
by one declaration. `hyper` expands the template's holes at load into a finite **candidate set**;
compares that set against the bound Target's **grant**; and reaches, at Run time, the **intersection** of
the two — filling it where the intersection is one host, and taking it from the input the Operation's
`host-input:` names where the intersection is several.

We chose this because the corpus had three mechanisms that read as three features and never said they
were one. ADR-0024 has a Manifest write `s3.{region}.amazonaws.com` over 35 enumerated regions and check
the derived set against the grant. The model has `hyper` fill the host where a Target grants one and an
Operation mark an input as the host where it grants several. The Capability `http` reaches "a host the
bound Target grants". Each is true, none subsumes the others, and the reading a competent implementer
reaches unaided is that a Manifest either writes a host or does not — after which either the enumerated
template has no run-time meaning or the marked input has no static one.

The three steps are what make both true at once. The template is what a Provider author knows: the shape
of the API's hosts. The grant is what a Target's reviewer authorises. The intersection is what the two
agree on, and it is smaller than either — which is the property that matters, because it means neither
the Manifest nor the Target can widen reach alone. `{from-target}` is not a special case in it: the
candidate set is then the grant itself, the intersection is the grant, and the rule reads out as the
model already described.

## Considered options

- **The Manifest writes no host; the Target's grant is the whole of reach.** Simplest, and it deletes
  ADR-0024's enumerated template along with the only way a Provider states which hosts its API actually
  has. It also makes a Target grant do double duty as a description of an upstream service, which is a
  fact about the Provider living in the artefact that authorises it.
- **The Manifest writes the host and the Target grant is advisory.** Rejected on the shape rather than
  the detail: a grant that only warns is the advisory check the extension model exists to eliminate, and
  it puts reach on the side of the artefact a third party authors.
- **The marked input fills a hole in `host:`.** The natural-looking unification, and it collides head-on
  with the rule that a Capability-relevant position never resolves to an Operation input. Reinstating
  that would make reach depend on a value arriving at Run time with no finite set to check at load,
  which is the static-decidability the whole authority model rests on.
- **A per-Operation host grant on the Target declaration**, so each Operation's reach is authorised
  separately. Rejected for multiplying the reviewed surface by the size of the Provider while adding no
  distinction anybody was asking to draw: reach is a host-level fact and every Operation of a Provider
  reaches the same service.

## Consequences

- **The marked input always carries a whole host.** An enumeration hole is a compact way of writing a
  large candidate set, never a second thing filled at Run time, so there is no position where a fragment
  of a host arrives from data.
- **A Capability-relevant position is exactly two positions** — `host:` and an Auth scheme's parameters.
  The wide reading of "anything that determines what a request may reach" would take in `path:`, which
  forbids `{zone_id}` and makes a parameterised REST API unwritable. A grant enumerates hosts and
  nothing finer, so nothing below the host can widen what the grant already allows.
- **`host-input:` is a sibling key, not a hole.** It names which of the Operation's inputs carries the
  host, and naming a property the input schema does not define is a Manifest contradicting itself,
  refused at load.
- **One error code covers both origins.** A candidate set with a member outside the grant and a `values:`
  list with a host outside it are the same comparison, and they carry the same name — which is what keeps
  the check readable as one rule rather than two that happen to agree.
- **Dynamic host discovery stays impossible.** A next-page URL, a redirect target, or a host read from a
  response is not reachable through any of the three steps, and pagination's forms are closed for that
  reason. The population may come from data; the reach only from an artefact.
