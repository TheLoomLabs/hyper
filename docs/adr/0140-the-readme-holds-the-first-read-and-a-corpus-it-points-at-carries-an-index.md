# The README holds the first read, and a corpus it points at carries an index

The README is 497 lines where it was 743, and 3,263 words where it was 6,001. What left it did not
stop being documented: the install story is [`docs/install.md`](../install.md), the vocabulary is
`CONTEXT.md` alone, and the 139 records this file sits among are named one by one in
[`docs/adr/README.md`](README.md), which a case asserts is complete.

No issue precedes this. It starts from a reading of the README and one sentence in it that was
false.

## What was wrong

**The sentence.** The last section said `docs/adr/` held *ninety-odd records of why*. There are
**139**, numbered `0001`–`0139` with no gaps. The count had been written once and never recomputed,
which is the ordinary failure of a number a human maintains — but the count was the smaller half.

**The link went to a directory listing.** `docs/adr/` carried no `README.md`, so the pointer landed
a reader in 139 filenames of the shape
`0093-orientation-is-a-handshake-field-and-hyper-writes-no-file-to-carry-it.md`, alphabetised by an
accident of numbering into chronological order, with no way in. CONTRIBUTING.md has held the rule
since it was written — *a corpus documents itself; most subtrees carry a `README.md` saying what
they drive* — and the largest corpus in the repository was the one that did not. Nothing could have
caught either defect: a count in prose and an absent file are both invisible to a suite.

**Install was at line 159**, behind the thesis, two Mermaid diagrams and a worked example. Of the
112 lines under that heading, the commands that put a binary on a `PATH` were **nine**. The rest
was a macOS Gatekeeper walk and the archaeology of one `-ldflags` invocation — both correct, both
the product of real measurement ([ADR-0133](0133-three-archives-nobody-had-run-carried-and-the-release-stamps-three-of-four-dirty.md),
[ADR-0137](0137-a-browser-sets-the-attribute-and-the-shell-runs-what-finder-offers-to-delete.md),
[ADR-0138](0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)), and neither
of them what a person who wants the binary is reading for. The `-ldflags` paragraphs were the
sharper case: they exist because `v0.0.1-alpha` predates ADR-0138, they say so, and they occupied
forty lines of the front page while saying *until there is a later release to name*.

**Three things were said more than once.** *An agent may read the record and add to it, and may not
create it, prune it, or bring anything new into the repository* appeared three times; *who may make
an edit stick is who may merge it* three times; the five artefacts were enumerated four times —
prose, diagram, bullet list, and again as a glossary row. The 59-line Glossary restated `CONTEXT.md`
under a heading that called `CONTEXT.md` the authority, which is the shape a drift takes before it
has drifted.

**And a hand-maintained 16-line Contents** duplicated the outline GitHub renders from the headings.

## The decision this records, and what it costs

**The README answers one question — is this for me, and how do I start — and delegates the rest by
name.** It keeps the thesis, the four diagrams, the `review` rendering, the quickstart, the two
surfaces and the non-goals. It loses the glossary, the Contents, the *Your first repository* table
that retold the quickstart as a grid, and every paragraph of install detail past the four commands.

**The count is gone from the prose that can rot and into a file a case reads.** `docs/adr/README.md`
names every one of them in order, with their own headings as titles, groups a dozen of them under *start here*,
and offers reading paths that are explicitly not a partition. The flat list is the part that must be
complete, and the part a case checks; the curation is the part that may be opinionated, because
nothing depends on it being exhaustive.

Three prices, all accepted:

- **The README is no longer self-contained on vocabulary.** A reader who meets *Definition* in the
  second paragraph now follows a link rather than scrolling. That is the trade the glossary was
  hiding: a second definition of a term, one link away from the authoritative one, kept in step by
  nobody.
- **The macOS walk is one line where it was thirty-six.** The README carried the whole of #262's
  finding — three browsers, Archive Utility, the signing asymmetry, the SIP caveat — and
  `docs/install.md` now carries the sentence a person downloading needs (*nobody signed these; run
  it from a shell*) and points at `docs/build/releasing.md` for the rest, which already held all of
  it in more detail. What justifies the cut is
  [ADR-0137](0137-a-browser-sets-the-attribute-and-the-shell-runs-what-finder-offers-to-delete.md)
  Claim 7's own conclusion: *the person this ticket is about probably has Go*, and the `go install`
  path has no Gatekeeper story at all. Documenting the unlikely path at length was the front page
  arguing with its own finding. What it costs is that a reader who wants the measurement follows a
  link — and `releasing.md` is where it was always written down.
- **A curated *start here* is a judgement that will age.** It names ADRs by subject, and a subject
  that gets a better record later leaves the old one on the list. Cheap to correct, and cheaper than
  no way in.

## What constrained the cut

Two couplings, both undocumented at the README end and both silent if broken:

- **`scripts/acceptance/run.sh:118` reads the version out of the README's install block**, with
  `sed -n '0,/^VERSION=/s/^VERSION=//p'`, so that the `hyper.yaml` an acceptance agent sees pins the
  version a reader would have installed. Moving the install block to `docs/install.md` would have
  fallen through to the script's own default and left every transcript pinned to a literal nobody
  maintains. The block stays in the README, at column 0, as the first `VERSION=` in the file; the
  detail is what moved.
- **`scripts/acceptance/run.sh:140` mirrors the quickstart's four artefacts verbatim**, but for the
  Target's `kinds:`. They are unchanged here, byte for byte, and the one documented deviation is
  still the only one.

Neither is a coupling this decision adds or removes. They are recorded because the next person to
shorten this file will meet them, and a `sed` in a shell script is not something a Markdown edit
looks like it could break.

## What fences it

- **`TestDocs_TheADRIndexNamesEveryRecord`** reads `docs/adr/` and `docs/adr/README.md` and requires
  them to be the same set: every record linked exactly once, and every link resolving to a file that
  exists. It is the case the *ninety-odd* sentence could not have had — a count in prose is not
  checkable, and a list of links is. Adding an ADR without indexing it now fails the suite, which is
  the property being bought rather than the tidiness.
- **`TestRelease_TheMacOSArchivesCarrySignaturesNobodyIssued`** is unchanged in what it asserts and
  amended in what it cites: it held the signing asymmetry against *the README and
  `docs/build/releasing.md`*, and now names `docs/install.md` in that position. The bytes it reads
  are the same bytes.
- **The acceptance harness itself** is the fence on the quickstart. `scripts/acceptance/run.sh` fails
  to build a repository if the version it reads is not one the binary can be stamped with, and
  `TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse` runs the setup half in the
  suite.

**No acceptance re-run is owed.** The orientation `hyper project` writes is untouched, and no agent
reads the README — the seal exists precisely so that transcripts cannot
([ADR-0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md),
`docs/agents/acceptance-re-runs.md`).

## What it does not do

- **No claim was softened to make it shorter.** Every sentence removed from the README is either a
  duplicate of one still there, or now stands in `docs/install.md` with its ADR citations intact.
  The non-goals section, which is the highest-value text per line in the file, is unchanged.
- **The spec is still the authority.** Nothing here moves a fact out of `docs/spec/`, and the README
  says as much above the fold rather than in its last section.
- **The index is not a classification of the corpus.** It carries no status field, no
  superseded-by graph and no per-record summary. An ADR here is a record of a decision at a moment,
  and `docs/adr/` has never needed a lifecycle on top of that; what it needed was a way in.
