# Provenance names the top-level Procedure's revision

Provenance carries `procedure_revision`, the git blob id of the file `run.json`'s `procedure` names.
It is Run-wide on `run.json` under ADR-0043, present on every Record version, and absent from a Step
file by the locality rule. It is never absent anywhere it belongs: every Run is a Run of exactly one
top-level Procedure (ADR-0036), so unlike every other member there is no level at which it has no value.

We chose this because the unaided reading is that `definition_revision` names the artefact that ran, and
this corpus not only reached that reading but rendered it: §8's `THE CODE MOVED` reported
`retire-preview-envs · definition revision`, a Procedure name under a Definition's fact. Three other
renderings had the value right and no name for it — the review's `a91f0c2 → working tree`, `FLAGS`'
`bound 3 → 5 since a91f0c2`, and the Comparison header's per-Run `rev`, which no Journal file held, so
the header could not be produced from the Store at all.

What the member buys is the asymmetry the mislabel was covering. A Definition declares which Kinds it
claims and which Targets it may bind and carries no argument values; the Procedure holds the Bound, the
selector, the `target:` a Step binds, the Cadence and every argument value. The Definition had a
revision in Provenance and the Procedure had none, so a Record's Provenance named the code that
performed the effect and not the code that decided its extent. `THE CODE MOVED` is the surface making an
AI widening a Bound between two Runs a first-class fact, and the Bound lives in the Procedure.

## Considered options

- **The innermost Procedure holding the Step.** The `path` distinction, one layer up. Rejected because
  a Run spans nested Procedures as one Run (§6), so at Run level this reading has *several* values, not
  one — the multiplicity ADR-0043 exists to refuse — and omitting it from `run.json` withholds the
  header's revision in exactly the Runs that need it most.
- **A digest over the transitive closure** of the Procedure and every Procedure it invokes. Single-valued
  always, and it catches nested edits. Rejected because it is a digest `hyper` computes rather than a git
  blob id, so no reader verifies it with `git hash-object`, and §7 refuses a second digest of one artefact
  on that ground.
- **`run.json` carrying it beside `procedure`, outside Provenance.** Cheap, and nesting is trivially
  single-valued. Rejected because it keeps a code fact next to the field set defined as *the record of
  which code produced something*, and it leaves a Record version unable to name the Procedure that
  produced it without reading the entry.
- **Derived at render** — `git hash-object` the file named by `procedure` at `repo_revision`, storing
  nothing. Rejected because it fails precisely where §7's reaper already falls back: a Run that recorded
  `repo_dirty` has no committed blob to derive from, and that is the case where knowing what the Run
  actually read matters.
- **The repository revision under another name.** The cheapest reading and the one §7's reaper appeared
  to depend on. Rejected as a *change* fact — a repository revision moves when a README does — while kept
  as the reaper's *load* anchor, the two now stated as different jobs: a commit resolves every Procedure
  in an invocation tree and a blob id resolves one file.

## Consequences

- **A nested Procedure's own revision is not recorded.** A Bound widened inside one moves nothing in
  Provenance. It is caught by §12's `Bounds` class and counted by the catch-all, which is the treatment
  every line of every artefact that is not one of the nine classes already gets. This is the deliberate
  cost of the member being single-valued without amending ADR-0043, and it is the part a future reader
  is most likely to reopen.
- **Irreversible from the first Run.** No file in the Store is ever rewritten (ADR-0011) and migration
  in place is impossible, so every Record version carries this member forever and a change of shape can
  only arrive as a schema version on a Store that accretes format handling (ADR-0028).
- **No change class was minted.** §12's `the digests` is *every member of the Provenance*, stated
  intensionally, so the row arrives with the member and the enumeration is still nine.
- **The review's range is a Run fact.** Naming the revision once made the review's `since a91f0c2` and
  the Comparison header's revision one value from one place: the last non-rehearsal Run of that
  Procedure (§8). A review against `HEAD` would render nothing on a branch where an agent authored and
  committed the widened Bound, which is the branch a human is about to approve.
