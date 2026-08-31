# A revision resolves where the artefact was committed, and the loop commits

**Every artefact revision the acceptance transcripts recorded is a git blob id that nothing ever wrote
to an object database, and the orientation's own loop is what put the repositories in that state.** The
loop ran `providers` → author → `check` → `review` → hand back the diff, and then a Run; `git commit`
appeared nowhere in it. A Run computes each artefact's blob id from the working tree, so a file nobody
committed has no such object, and the Record's Provenance points at bytes that exist in no repository —
including the one that recorded them.

**`hyper` does not write the blob, and it does not refuse the Run.** It states what the member is,
names the act in the loop where an author meets it before running, warns on stderr where a Run is
about to record ids nothing will resolve, and — where a review meets one of those revisions — says
which of `not-in-clone`'s causes it met. Issue #239.

## What was observed

The sealed acceptance run of 2026-08-30 against `tenant-onboarding`, read in
[ADR-0112](0112-the-second-run-skipped-what-the-first-did-and-every-revision-it-recorded-is-unresolvable.md)
(issues #224, #232). The session authored a Procedure, ran it over three tenants, added a fourth to the
roster and to the artefact, and ran it again. In the repository it left behind:

```
$ git cat-file -t 46b387bfe071df2e70f5ab52c5c7ff8e59cefb0a   # procedure_revision, run 1
fatal: git cat-file: could not get object info
$ git hash-object procedures/onboard-tenants.yaml
6aec6e87134577ab787efc0ae26de1d3d29d0b40                      # the id is right; the blob is absent
```

Two surfaces degraded, and the second Run is where both showed:

- **`review` lost its baseline.** *no baseline — 46b387bfe071df2e70f5ab52c5c7ff8e59cefb0a is not in
  this clone*, on the one moment in three acceptance runs where a `review` had a real baseline to draw
  — an artefact that had run, being reviewed again after an edit.
- **`changes`' catch-all reported `0`.** Both Runs recorded the same `repo_revision` with `repo_dirty:
  true`, so §8 counted the lines between one commit and itself while `tenants/roster` sat modified in
  the tree.

The session diagnosed both correctly and said the thing that made this a ticket rather than a note:
*Commit the tree and future windows carry real provenance; I did not commit, since you did not ask me
to.* It is under every acceptance transcript, not only this one — the `fleet-rollout` run
([ADR-0110](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md))
recorded the same dangling ids and did not notice, having never reviewed an artefact after running it.

## What the member is, and what it is not

§7 said:

> `definition_revision` is the git blob id of the Definition file: content-addressed, computable
> offline from the working tree, unmoved by a rebase, and equal exactly where the content is.

Every clause of that is true and the sentence reads as though one more were: that the id is a
**pointer**. It is an identity. It compares equal to a later `git hash-object` of the same file whether
or not any object database holds it, and it comes back from a `git show` only where somebody committed
those bytes. §7 now says so outright, and says where the difference is paid.

## What was decided, and against what

**`hyper` does not write the blob.** `git hash-object -w` writes the same object under the same id, it
is one flag, and it would make every recorded revision resolve — which is why it is the candidate the
ticket lists first. It is refused on two grounds:

- **The promise would expire.** An object nothing references is unreachable, and unreachable loose
  objects are what `git gc` prunes. A revision that resolves for two weeks and then does not is worse
  than one that never did: the failure becomes intermittent, on the one member whose stated virtue is
  that a rebase does not move it.
- **It writes into a repository `hyper` only reads.** `internal/revision` starts no subprocess that
  writes an object, moves a ref, reaches a remote or names a commit identity; the branch `hyper` writes
  is its own orphan one, and it never checks it out (ADR-0075). Filling the caller's object database
  with objects no commit reaches, on every Run, is a side effect on the code branch — and it is
  [ADR-0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md)'s rule
  inverted: an absent object is named rather than supplied.

**`run` does not refuse a dirty tree.** It would remove the state rather than the defect: `repo_dirty`
marks a Run that read bytes differing from `HEAD`, and a Run that cannot happen leaves nothing to mark
— the marker and §8's suppression of the `git diff` command, both of which the ticket requires
untouched, would become unreachable. A rehearsal against an uncommitted draft is also exactly what a
rehearsal is for.

**The act is stated in the loop, because the loop is what produced the state.** The orientation's step
6 was *`run`, once they have*; it is now *Commit the artefacts, then `run`* — with what the omission
costs beside it, three surfaces named: the Record points at a revision that resolves nowhere, the next
`review` opens at *no baseline*, and `changes` sees no moved lines. A rule with no consequence beside
it is one an agent reorders against whatever else it is doing, which is
[ADR-0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)'s lesson on a third
axis. It is a step of the loop rather than a twelfth thing §9 says the orientation states: what was
missing was an act between approval and `run`, not a fact about either.

**`run` warns where the tree it read is in no commit.** §9's narration gains its first conditional
line, beneath the Run's own id and above its first Step, naming the act and what its absence costs. It
is worded about the **bytes** and not about the files: what fires it is `repo_dirty`, and the commoner
half of that marker is a committed artefact edited since, of which *an artefact here is in no commit*
would be false. It
fires exactly where the entry records `repo_dirty` — one read of the code branch, two readings of it —
so a case narrating one without the other would be a warning about a state the record does not agree it
was in, which is what holds them together (`TestRunNarration_TheWarningAndTheMarkerAreOneFact`).

**It sends nothing on the MCP surface**, as `Began` does, and for a reason of its own: a warning there
would arrive in the same envelope as the Run it is about, with the Records already written. That
surface's channel for this is the orientation, which reaches an agent *before its first tool call* —
which is the only place *before the Run* is a true description of. What the envelope does carry is the
Run's own account: `repo_dirty` on the `provenance` row, whose schema now says what the marker costs
and what repairs the next Run.

**`not-in-clone` says which cause it met, and stays one name.** Three of its causes are facts about the
clone and no one act repairs all three, which is why the sentence names none; the fourth is a fact
about the Run, and it is separable without asking a remote.

**Two signals separate it and neither does it alone.** The entry's own `repo_dirty` licenses the
sentence — it marks that the Run read bytes differing from the commit it recorded, and an artefact
whose recorded revision is in no commit differs from `HEAD` by construction, so the marker is always
there where this is true. It names no file, which is what the second signal is for: the commit's tree
at the artefact's path, exactly that revision or not. The marker alone would carry the sentence to
every artefact of a Run one file of which was dirty; the tree read alone resolves **today's** path
against that commit, so an artefact renamed since the Run would read as bytes nobody committed. Both
together keep that rename on the weaker sentence in every Run that was otherwise clean, which is every
Run the loop above produces.

The wire name stays `not-in-clone` for `not-run`'s reason: a consumer filtering on it wants a stable
string, and a fifth member would put a `baseline_absent` a client already switches on into a shape its
existing arm no longer covers. The sentence pays no such cost, so it carries which cause stands — one
name over two sentences, as `not-run` is one name over five. **The machine caller gets both**: a
review's page travels in the structured content beside the rows (ADR-0100), so an agent on the MCP
surface reads the sentence in the same answer that carries the name.

**Neither sentence names an act, and the second one could not.** Committing now writes today's bytes
under a new id; it does not produce the ones that ran. The baseline of the draft under review is gone,
and saying otherwise would be the repair-that-does-not-work §8 refuses one layer in. What is repairable
is the next Run, and that is said where an author stands before running rather than after.

## The seam the reading is expressed at

**`revision.Committed`** answers the blob id one commit holds at one path, and "" where it holds no
file there. It sits beside `Held` because it is the same subject — what this clone can be asked about
an object — and it differs in the one way that matters here: **it reads the tree and never the blob**,
so a blobless partial clone answers *committed* rather than the stronger sentence. A commit this clone
does not hold is reported as the read it is, and the caller keeps the weaker sentence: a clone that
cannot answer is itself a clone that is behind.

**The question is asked of the two artefacts carrying a revision of their own and of no other.** The
other four anchor on the commit, and `supplyingEntry` already passes over every entry that recorded
`repo_dirty` for them, so an entry that anchors one of those read it exactly as that commit holds it
and there is no working-tree revision left to be absent.

**What the two signals cannot rule out is stated rather than hidden.** A rename, plus a Run that was
dirty for some other artefact, plus an object this clone does not hold, still reads as *never
committed*. All three at once is a corner, and the alternative is a full-tree search for the blob at
that commit — a scan for a revision the clone has already said it does not hold, to move one sentence
onto the other. The screen ADR-0026 says may not lie is not lying there: both sentences name an absence
and the range is missing either way; what is wrong is which cause it attributes.

## Consequences

- **§7 states what the member is.** A blob id resolves where the artefact was committed, `hyper` never
  writes one, and the `repo_dirty` marker is the same fact on the entry. The sentence that read as a
  pointer now says which half of it is a pointer.
- **The orientation grows by three lines on both channels at once**, the two being one text
  (ADR-0095). An `AGENTS.md` already written keeps its old text until somebody regenerates it, which is
  `project` never overwriting a file that stands.
- **§9's narration is three lines rather than two**, and the third is the only conditional one. Three
  golden cases carry it — the three whose fixtures hold an `uncommitted/` — and the two hundred and two
  beside them hold its absence.
- **§8 renders six absence sentences under four names**, and §12 says why the fifth name was not
  minted. `repo_dirty` and the catch-all's suppression on a dirty side are untouched, which the ticket
  required.
- **§13 names the fourth cause among the honest limits.** It is the one of the four `hyper` can name
  and the one it can do nothing about after the fact, which is the shape the chapter is for.
- **A repository built by following the orientation renders a real baseline.** The draft that ran was
  committed before it ran, so its blob is in the object database and the next `review` of the artefact
  opens at it. That is the criterion issue #239 wrote for itself, and the transcript that says whether
  it holds is the next acceptance run that edits an artefact between two Runs.
