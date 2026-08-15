# A Cadence and a secret-producing Step are refused together

A Step whose Operation declares secret output Refuses where the invocation supplied no Secret sink, and
the workflow `project` generates supplies none — ADR-0007 says so outright: *in Actions the generated
workflow simply supplies nothing*. Compose the two and a Procedure that declares a Cadence and reaches
such a Step passes `check`, projects a workflow, and Refuses at every occurrence for as long as the
clock keeps firing. **`check` refuses the combination** (`cadence-secret-output`, §4), walking every
Procedure reachable from the one declaring the Cadence, to any depth.

This is ADR-0038's argument arriving on a second door, and the reading a competent implementer reaches
unaided is not merely available here — it is what the corpus says out loud in two places. §9 fixes the
absent sink as *a fact about the invocation and never about the environment it runs in*, and `check`
never sees an invocation, so an implementer reading only the prose concludes this cannot be a static
check. What that reasoning misses is the quantifier. What is refused is not *this Run has no sink* but
*this Procedure's recurrence has no invocation that could ever supply one* — a fact about a reviewed
artefact, the Manifests its Steps reach, and a compiled-in projection template (§10, ADR-0046), which
are three things `check` already compares.

ADR-0038's three clauses all hold, and two of them have got worse:

- **Nobody is present when it happens.** ADR-0005 puts the clock on GitHub Actions, so every occurrence
  Refuses unattended. Unchanged.
- **It arrives through an omission rather than a decision.** `secret:` is the Provider author's claim
  about output that author understands (§3) and `cadence:` is the Procedure author's. The two artefacts
  are separately correct and the combination is fatal — ADR-0038's clause verbatim, one key over.
- **There is no way out at all.** ADR-0038's consumer facing a digest-verified Manifest at least has a
  Provider author to petition. Here the remedy is §8's fifth remediation class — *a different
  invocation* — and the invocation is a byte-exact projection with a compiled-in executor (§10,
  ADR-0046). A remedy that is a flag is a remedy no generated workflow can take.

And where ADR-0038 ships a Procedure that **works once**, this ships one that **works never**: a
repository that passes every check and never once does the thing it was written to do.

The walk costs nothing. `cadence-run-once` already walks every reachable Procedure to any depth and
reads each Step's Operation out of its Manifest to find `repeatability:`; this reads `secret:` off the
same Operation on the same walk. §4 gains a rule and not a traversal.

## Considered options

- **Have the projection supply a sink.** The obvious fix, and it is rejected in both its forms. Supply
  a path under `$RUNNER_TEMP` and `hyper` makes the call, mints a credential whose only handle is the
  value that came back, writes it to a disk destroyed minutes later, and exits `0` — an Asset built with
  its handle deliberately destroyed, which is ADR-0070's collision hazard arrived at on purpose rather
  than by accident. Supply a path and have the workflow forward it — an artifact upload, a repository
  secret, a line in the step summary — and the first publishes a secret and the second is `hyper`
  acquiring one, which is the product ADR-0007 exists to refuse. ADR-0007's *simply supplies nothing*
  stands, and this decision is the reason it stands rather than the illustration it was written as.
- **Carry it as an honest limit in §13**, on ADR-0072's precedent: meet a fault with no guardrail in
  front of it and answer with a limit rather than a guard. Rejected because ADR-0072's fault was a
  collision between what an API answered and what the Store already held — undecidable offline, and its
  Refusal fires after a call has reached the world. This one is decidable with both operands in hand and
  nothing touched, which is the line ADR-0072 itself drew.
- **Widen `cadence-run-once` rather than mint a second code**, on ADR-0067's and ADR-0070's precedent,
  to *a Cadence reaching a Step that cannot recur*. This is the strongest case for widening the map has
  yet produced, because the two faults share a remedy exactly — §10's two Procedures, the run-by-hand
  Steps split from the recurring ones. Rejected on the standing objection: a reader handed
  `cadence-run-once` on a `secret:` clash goes looking for a `repeatability:` that is correct. ADR-0070's
  widening worked because the invariant underneath was genuinely one; here there are two — one Step
  recurs wrongly, the other never runs at all. Two codes with one remedy is the honest shape.
- **Scope the sink requirement to effectful Operations**, which would take every read-only Procedure —
  the Cadence's main customer — out of the fault entirely, on the ground that ADR-0007's justification is
  recoverability and a `read` has no *once* (ADR-0037 forbids a run-once `read`). Rejected. The sink is
  the sole route by which a secret value ever leaves `hyper`, so a `read` whose point is fetching one,
  invoked without a sink, writes `"<secret>"` to the Store and produces nothing — silently useless rather
  than Refused. §9's sentence keeps no Kind axis, and the price is stated: this is the first Cadence check
  that can refuse a read-only Procedure, where `cadence-run-once` is effectful-only by consequence of
  ADR-0037 rather than by clause.
- **Exempt `--dry-run`.** Reads generous — rehearse without effects, and refuse nothing for a secret that
  cannot be produced. Rejected because the exemption cannot be written without the Kind axis the previous
  option just refused: §9's dry-run *performs the reads it reaches*, so a `read` declaring secret output
  is reached, produces the secret, and has nowhere to put it. It also costs one flag that §9 already
  states weakens no check.

## Consequences

- **Two new `error_code` members, and the set goes to forty-nine.** `cadence-secret-output` is §4's
  static check, counted with §10's on `cadence-run-once`'s precedent — the check runs in `check` and the
  fact it is about is the Cadence, taking §10's Cadence group from three to four. `secret-sink-absent` is
  the Run-start check the absent sink always needed and never had: §9 has stated that Refusal since it was
  written, and §12's closed set carried no name for it, so the spec described a Refusal nothing could
  name, render, or exit on. It is §9's second contribution beside `credential-absent`.
- **The absent sink declines before Step 1**, in the credential pass's company rather than at the Step
  that needs one, and reports every such Step at once as the credential pass reports every absent slot
  (§6). Both operands are in hand at Run start: the invocation carries a sink or it does not, and which
  reachable Steps declare secret output is a walk over reviewed artefacts. The alternative is worse than
  untidy — a Refusal at the Step under a Cadence runs the Steps *before* it at every occurrence and never
  reaches the tail, which is an effectful prefix repeated indefinitely with no completion, and the
  multiplier ADR-0072 named arriving on a third door. Declining at Run start makes the Cadence failure
  total and inert instead: nothing ran, every time.
- **A fifth remediation class in §8, *a different invocation*.** Beside the artefact edit, the command,
  the different binary and the act on the environment. `secret-sink-absent` renders no `EDIT ONE OF`
  table and names its remedy verbatim in the `=` notes, exactly as `credential-absent` does.
- **The carets cite the repository's own artefacts and never the Manifest.** `cadence-secret-output`
  cites the `cadence:` line, on `cadence-run-once`'s precedent — the check walks to the Manifest and
  cites the artefact the reader controls. `secret-sink-absent` cites one Step line per Step needing a
  sink, `credential-absent`'s one-per-slot shape. Neither points at a `secret:` line, so the
  digest-verified-file trap that costs `store-schema-unsupported` its caret never arises.
- **No named limit in §13, and the asymmetry is the point.** ADR-0038 earned one because its remedy runs
  *through* the Manifest, so an installed one strands the consumer with no local route at all. This
  remedy never touches the Manifest: §10's two Procedures moves the secret Step into one a person
  invokes, entirely within the consumer's own artefacts, whoever wrote the Provider. This is the
  friendlier of the two checks and the first Cadence check whose remedy is always local.
- **ADR-0007 is amended.** Its *in Actions the generated workflow simply supplies nothing* was written as
  a neutral illustration of the invocation-not-environment rule. It is now the load-bearing fact behind a
  whole refused combination, and a reader must not be able to leave that sentence thinking nothing
  follows from it.
- **The replacement idiom is ADR-0038's, unchanged.** The secret-producing Steps in a Procedure run by
  hand with `--secret-out`, the recurring Steps in one that carries the Cadence, with any shared body
  factored into a nested Procedure (§10).
