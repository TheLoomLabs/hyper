# A release's body is a file in the tree the tag names

`release.yml`'s publish step reads `docs/build/release-notes/$GITHUB_REF_NAME.md` and passes it as
`--notes-file`. Where there is no such file, or it is empty, the step exits `1` and publishes
nothing:

    notes="docs/build/release-notes/$GITHUB_REF_NAME.md"
    if [ ! -s "$notes" ]; then
      echo "no release notes at $notes; write them before tagging" >&2
      exit 1
    fi

`v0.0.2-alpha` is the first release published this way.

## What was wrong

`--generate-notes` asks GitHub to compose the body, and what it composes is the commit subjects
since the previous tag. For `v0.0.2-alpha` that would have been twenty-three lines of the shape

    fix(cadence,cli,mcp,spec,adr): a repeating minute is a promise the executor
    drops, and the gloss now says so (issue #260)

Every one of them is accurate, and none of them is addressed to the reader. A commit subject in
this repository is written for somebody about to read the diff under it: it names the packages
touched and the issue it closes, and it assumes the vocabulary. A release page is read by
somebody deciding whether to install a binary, who wants to know what changed for them and what
it will cost to upgrade. Those are different documents, and one of them was standing in for the
other because it was free.

**The specific thing it buried** is the only change in this release that alters what a person
types: `go install` with no `-ldflags` now produces a working binary, where under `v0.0.1-alpha`
it produced one reporting `unknown` that Refused every repository. That is the first line of the
notes and the twelfth line of the generated body, under a subject naming five packages.

## The decision this records, and what it costs

**The body is written by the person making the change, in the diff that makes it.** A notes file
lands in the same review as the code it describes, at the moment its author still knows why the
change was made — rather than being reconstructed at tag time from a log, which is when nobody
has that context and everybody wants to ship.

**The absence of one stops the release.** This is the half that had to be decided rather than
assumed. A generated body cannot be missing; a written one can, and the failure mode of a soft
fallback is a release published with an empty description that nobody goes back and fixes,
because by then it is published. So the guard is hard, and it fires before `gh release create` —
after the suite, after the build, after the artefact has been checked to report the tag, which
costs those minutes on a tag pushed without notes. That is the right place for it: the cheap
check would be a lint on the pull request, and a tag can be pushed at a commit no pull request
ever covered.

Three prices:

- **A tag is no longer sufficient to cut a release.** `docs/build/releasing.md`'s procedure gains
  a step, and somebody who tags from memory gets a red job. The message names the exact path.
- **The notes are written before the bytes exist.** They describe a build that has not happened,
  so a claim about the archives — *all four are stamped from a clean tree* — is a claim the author
  is making on the strength of the cases, not of the artefact. `TestRelease_…` holds that
  particular one; a claim with no case behind it is one to leave out.
- **They can rot like any other document.** The one shape of that with a case is the copy-paste:
  the notes for one release are the obvious start for the next, and a file renamed without its
  contents following it describes the wrong bytes accurately.

## What fences it

- **`TestDocs_AReleasePublishesNotesFromTheTree`** reads `release.yml`'s publish step by what it
  runs rather than by its name, and requires `--notes-file`, the path these notes actually live
  at, the absence of `--generate-notes`, and the `exit 1` that refuses a tag with no file. A step
  that quietly went back to the generated body would publish something plausible and wrong, which
  is the shape of defect nobody reports.
- **`TestDocs_EveryReleaseNotesFileNamesItsOwnVersion`** is the copy-paste fence: every file in
  the directory is named for a tag, is non-empty, and names its own version somewhere in its
  text.
- **The workflow's own guard** is the fence on the case the suite cannot reach, a tag being a
  thing the suite does not push.

**No acceptance re-run is owed.** Nothing an agent reads changed (`docs/agents/acceptance-re-runs.md`).

## What it does not do

- **It is not a changelog.** There is no `CHANGELOG.md`, no aggregation across releases, and
  nothing generates one file from another. Each release's body is one file, written once, and the
  directory is the archive of them because that is where they already are.
- **It fixes no format.** The file is Markdown and GitHub renders it; nothing parses it, nothing
  requires headings, and the only structural claim any case makes is that it names its own
  version. A release whose notes are three sentences is a release whose notes are three
  sentences.
- **It does not move where a release's version is authored.** That is still the tag and nothing
  else (§11), and the notes filename derives from it the way every published filename does.
