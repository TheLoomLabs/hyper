# Retry only follows a failure that provably preceded the request

A retry **Pattern** may re-send a call only when the failure proves the request never left: connection
refused, DNS resolution failure, TLS handshake failure. Never on a timeout, never on a connection
reset mid-flight, never on a 5xx — in each of those the server may already have acted. The
classification is `hyper`'s own closed set, decided at the transport layer, and no Manifest can widen
it or add to it.

We chose this because a retry Pattern is the one place `hyper` itself can touch the world more than
the artefact says. ADR-0004 removed code from Providers, which means the retry is `hyper`'s code, not
the Provider's — so a retry that re-sends a `mutate` whose response was lost is an effect reaching the
world that no review ever saw. Recording the attempt is not sufficient: the Comparison is
accountability rather than guardrail, and it reports after the fact. The guardrail therefore has to be
static and mechanical, which the transport-level classification is.

This is written down for a reason the other Pattern rules do not share: the retry class is `hyper`'s
own code, so widening it later moves nothing a human reviews. No Definition changes, no `THE CODE
MOVED` row appears, no digest shifts. An ADR is the only artefact that can hold it.

## Considered options

- **Retry is `read`-only, full stop.** The strictest reading, and the fallback if the transport
  classification ever proves unmaintainable. Rejected as the primary answer because infrastructure
  APIs rate-limit and 503 on effectful calls routinely, and sending every one of those to the wall —
  unwritable until `hyper` ships something — buys no safety over the pre-send class, which is provably
  harmless.
- **Gate retry on the Operation's declared Repeatability**, allowing it wherever the Manifest says
  `repeatable`. Tempting, because the Manifest already declares something adjacent. Rejected because
  the two facts are not the same: `repeatable` means *invoking it again on a re-run is intended*, which
  presumes the first call **completed**; retry presumes it **may** have completed. Overloading the term
  would make a declared safety property quietly mean something stronger than the author agreed to.
- **Allow retry anywhere and record the attempt count.** Rejected: this is the accountability half of
  the thesis being asked to do the guardrail half's job.

## Consequences

- **Ambiguous failures stay ambiguous.** A timeout or a reset on an effectful Step halts the Run
  `failed` with the Step's Disposition *attempted, outcome unknown*, exactly as before. Nothing retries
  its way out of that state, and nothing pretends to know.
- **Every Disposition carries what `hyper` did** — attempts, pages, poll iterations — as a closed,
  `hyper`-owned record that no Provider supplies, on the same grounds as `error_code` and Auth schemes.
  It renders only when it is not the trivial single call.
- **An API that fails only under load, with 5xx, and genuinely wants a retry has no route** but waiting
  for `hyper` to ship one. That is the ceiling, with a second named victim beside deferred OIDC
  federation.
- **Two `hyper` versions can classify the same failure differently** and the Journal will record both
  faithfully as if they were one behaviour. This is behavioural version skew across a shared Store,
  which a store layout version cannot catch, and it belongs to the distribution question.
