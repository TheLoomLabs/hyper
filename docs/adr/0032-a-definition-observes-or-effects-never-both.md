# A Definition observes or effects, never both

A Record's identity is `(Target, Definition, name)` and excludes the Record type (ADR-0025), so a
Definition claiming `read` beside `mutate` or a `destroy` Operation can write an Observation version and
an Asset version into one series — which §2 forbids in the words *neither ever becomes the other* and
which nothing checked. `read` may therefore not appear in a Definition's `kinds:` beside `mutate` or
beside a `destroy:` claim, refused at load as `definition-kinds-mixed`. Reading what a Definition creates
is a second Definition against the same Provider and the same Target, and the Definition segment of the
identity is what keeps the two series apart.

The reading a competent implementer reaches unaided is that **nothing stops it**: `record_type` is
derived from the Operation's Kind (§12), the version file carries it, and every write succeeds. The
conflation is invisible on the surface and every reader of a series downstream of it silently returns a
wrong answer rather than an error. A `read` decides `skip-if-recorded`, which tests the head version of
the series and finds an Observation where it looks for a standing Asset. A `read` above a Tombstone makes
the Head alive again (§7), so an identity `hyper` destroyed reads as standing and the rebuild is skipped.
A `read` moves what an effectful selector reaches, Expansion being scoped to Assets for a `mutate` or a
`destroy` (ADR-0027). And Compaction removes interior Observation versions and never an Asset (§7), so it
prunes a mixed series by type. In every one of these a Step that only looks changes what an effectful
Step does, which is the property the safety model exists to deny.

We chose a static rule on the Definition's claim because it is the one shape that puts this in §5's
first clause — *most dangerous states are unreachable from the five artefacts by construction* — rather
than after an effect. The narrower rule the hole seems to ask for is unavailable: *one Definition may not
span a `read` and an effectful Operation projecting the same identity field* cannot be checked, because
in `hyper`'s own worked Manifest those paths are `$.result.id` and `$.id`, and knowing that two paths
name one upstream field is knowledge about the API that `hyper` has no oracle for (§4). A claim-level
rule needs one authored line and no Manifest reading at all. It is coarser than the hole — a Definition
reading zones and creating DNS records has two disjoint identity spaces and no collision available to it,
and is refused too — and that cost is taken on the same ground ADR-0023 takes its restrictions: a
restriction is reviewable, and this one is one line to state, impossible to misapply, and legible on the
review surface as *this Definition observes* or *this Definition acts*.

## Considered options

- **Identity carries the Record type**, making the triple a quadruple and the collision unrepresentable.
  Rejected for what it stores rather than what it costs: two directories differing only in
  `observation`/`asset` — same Target, same Definition, same name, adjacent on a branch built to be read
  in a browser — is the adoption pair §8 forbids joining, manufactured as a first-class shape with only a
  rendering rule between a reader and the join. It is also silent: no artefact changes, no review
  happens, and the author never learns their Definition does two incompatible things. Both this and the
  chosen rule produce the same two series in the real workload; only one of them makes the author write
  the split down.
- **Refuse the collision at write time**, with an `error_code` beside `record-identity-collision`. The
  identity is projected from the response, so the collision is knowable only after the call has gone out —
  and §5 defines a Refusal as a guardrail declining before any effect reached the world. Under this the
  `mutate` has already created the resource, no Record is written for it, and what stands in the world is
  an Orphaned Asset that a guardrail manufactured. The case-insensitive collision accepts a post-response
  Refusal because it is a freak; this one is reachable by any Definition that reads and writes one
  resource class.
- **Identity carries the Operation**, separating the series by what projected them. Rejected because a
  `create` and a later `update` of one resource would become two series, which ends the Comparison's
  ability to say *this Asset changed* — the destination ADR-0025 keyed identity for in the first place.

## Consequences

- **Reading a thing you created is two Definitions and two series, and `hyper` never relates them.** The
  Comparison shows the Asset under `YOU DID THIS` and the Observation under `THE WORLD MOVED`, under the
  same name, unjoined — because joining them is the drift detection ADR-0010 declined. The rule makes the
  pair honest rather than making it disappear.
- **An effectful Operation reaching a resource `hyper` did not create writes an Asset for it**, since
  `values:` lets an effectful Step name a foreign identifier literally (§5). This is not adoption arriving
  by the side door and the difference is not a technicality: `hyper` is accountable because **its own
  effect reached the thing**, which is a fact about what happened, where adoption is `hyper` deciding a
  Record it only ever observed should now be one it answers for. The first is earned by acting; the second
  is a claim about the past. `CONTEXT.md`'s Asset entry states the test as the effect for this reason.
- **`mutate` beside `destroy:` remains legal and must.** A Tombstone lands in the series the `mutate`
  created, so that pairing is the one the record positively requires in a single Definition. The rule is
  about `read`'s company alone.
- **A Definition is now legible as an observer or an actor** on `AUTHORITY`, which is a narrowing that
  buys clarity: a row reading `mutate destroy` says what the artefact is for, where `read mutate destroy`
  said only that it was permitted a great deal.
- **What destroying a `values:` member writes** is not settled here. A Tombstone belongs under an Asset's
  identity, and a `destroy` reaching a foreign resource by literal identifier has no Asset series waiting
  for one; this rule narrows that question — the Definition that destroys can never be the one that read
  the thing — without answering it.
