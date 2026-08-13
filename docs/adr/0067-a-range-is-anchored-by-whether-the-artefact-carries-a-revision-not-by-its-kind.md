# A range is anchored by whether the artefact carries a revision, not by its kind

A review's range opens at the last non-rehearsal Run that read the artefact, on all five reviewed
artefacts. What that Run supplies is decided by one test and it is not the artefact's `kind:`: where the
artefact carries a revision of its own in Provenance, the range opens at that member; where it carries
none, at that file's blob resolved under the `repo_revision` the same Run recorded (§8).

Two artefacts pass the test and they are not two kinds. A top-level Procedure carries
`procedure_revision` and a Definition carries `definition_revision` — both git blob ids over the bytes
that Run actually read. A Target declaration, a Repository declaration, a Manifest, **and a Procedure
reached only by invocation** carry none: ADR-0048 gives the member to the top-level Procedure and to no
other, so one of the five kinds falls on both sides of the line at once. A rule keyed on `kind:` cannot
express that, and would be wrong about a nested Procedure without anyone noticing, since the artefact
that fails is the same artefact that passes one invocation up.

The reading an implementer reaches unaided is `HEAD`. It is what every review tool they have used
compares against, it needs no Journal, and §8 before #56 said nothing else. #56 killed it for the
Procedure on a specific argument — an agent that authors a widened Bound *and commits it* leaves the two
sides of a `HEAD` range equal, so the review renders no range and no `WIDENED` flag on the one branch a
human is about to approve. That argument was written about a Bound and is not about Bounds: the same
agent commits `env: STAGING_TOKEN` → `env: PROD_TOKEN`, which is the edit §12 minted the credential
source class for. `HEAD` fails identically on all five and is refused on all five.

Having lost `HEAD`, the unaided implementer writes a switch on `kind:` with five arms, because the five
artefacts genuinely differ in what the Store holds about them — and three of the arms then have nothing
to put in them. That is the wrong decomposition rather than a hard case: what actually varies is which
Runs count as having read the artefact, which is a Journal query, and whether that Run holds a revision
for it, which is a Provenance question. Neither varies with the kind except by coincidence.

## Considered options

- **`HEAD` for the artefacts with no Provenance member.** Rejected on #56's own argument, generalised: an
  agent commits its edit, the two sides go equal, and the review renders an empty range as a *successful*
  one. That is worse than the absence it replaces, because a named absence can be read and an empty range
  reads as *nothing has changed*.
- **The last Run's `repo_revision`, full stop, for all four.** One lookup and no evidence-gathering.
  Rejected: a Target declaration nobody has bound in five months would be compared against yesterday's
  unrelated Run and render *nothing moved* about the credential edit — #56's failure arriving through a
  different door, and in exactly the class §12 minted to catch that edit.
- **No range at all for the artefacts with no member of their own.** Rejected: it discards the range on
  the artefact with the strongest case for one. The credential source is a named change class *because*
  `env: STAGING_TOKEN` → `env: PROD_TOKEN` is a visible one-line edit, which is the kind of edit a range
  exists to mark.
- **Rendering `repo_revision` as the range's endpoint rather than resolving it to a blob.** This is the
  Comparison's existing shape — its catch-all row renders `git diff 1f0a3d7 88bc402`. Rejected here
  because the subjects differ: that row counts lines across the reviewed five, so the repository is
  genuinely its subject, where a review's range has one artefact as its subject and sits on the line
  beside that artefact's path. A commit in that position invites the reading #56 refused — a repository
  revision moves when a README does.
- **A per-`kind:` rule with five arms.** Rejected as the decomposition this ADR exists to name. It is
  silently wrong about a nested Procedure, and it obscures that three of its arms are one rule.

## Consequences

- **The anchor is stated once and reads on all five**, plus the nested Procedure. §8 states which Runs
  count as having read each artefact — `run.json`'s `procedure`, and a Step file's `definition`,
  `provider`, `target` or `path` — and the most recent qualifying entry supplies the range.
- **A Step file carries `provider`** (§7). `manifest_digest` names a Manifest's bytes and never which
  Provider they were, so without the member a Manifest's *anchor* is found by resolving each Step's
  Definition at that Step's own revision — a git object per Step, in a clone that may hold none. It
  closes the same gap in `THE CODE MOVED`'s two Manifest classes.
- **A `repo_dirty` entry supplies no range to an artefact with no member of its own.** The resolution is
  valid only over a clean tree; on a dirty one the blob resolves and names bytes that did not run, so the
  gutter marks the wrong lines. The two members that survive a dirty tree are hashes of what ran and are
  unaffected.
- **A built-in Manifest has no range at any point in its life** and takes a third `baseline_absent` name,
  `built-in` (§12). It is the one absence no act repairs, and it ranks above the other two so that the
  header never promises a range that repairing the Store cannot produce.
- **A renamed artefact needs no rule.** Step files record a Target declaration by name and a name is
  pinned to its basename (§4), so a rename is a new name no Run has read, and the ordinary absence
  renders.
- **Nothing here reaches the network or the credential.** The rule adds a Journal read and a git object
  read to a surface that already performed both (ADR-0057), and a review still resolves no credential,
  reaches no network, and invokes nothing (§8).
