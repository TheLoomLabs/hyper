# A flagless build answers with the version the toolchain recorded

`internal/version.Current` reads a second stamper. `-ldflags -X` still decides wherever it wrote;
where it wrote nothing, the version is the one Go derived from the repository the source sat in and
recorded in the build information:

    func stampedVersion(linked, module string) string {
        if linked != unknown {
            return linked
        }
        if module == "" || module == "(devel)" {
            return unknown
        }
        return strings.TrimPrefix(module, "v")
    }

That closes [#263](https://github.com/TheLoomLabs/hyper/issues/263).

## What was wrong

`hyper` learned its own version from one flag and nothing else (#191, ADR-0020). A build that omitted
it reported `hyper unknown`, Refused `version-pin-mismatch` at exit `77` in every repository it
touched, and could not `project` — `hyper project` asked the release host for a tag named for
`unknown`, was answered `404`, and Refused `release-artefact-absent`.

**The toolchain had already written the answer into the same binary.** `debug.ReadBuildInfo` carries
`Main.Version`; `Current` was calling that function to read `vcs.revision`, `vcs.time` and
`vcs.modified`, and walking past the version. Measured on `linux/amd64` with go1.26.0 against the
published `v0.0.1-alpha` ([ADR-0137](0137-a-browser-sets-the-attribute-and-the-shell-runs-what-finder-offers-to-delete.md)
Claim 7 is where the chase started):

| how the binary was built | `Main.Version` |
|---|---|
| `go install …/cmd/hyper@v0.0.1-alpha`, no flags | `v0.0.1-alpha` |
| clone at the tag, clean tree, no flags | `v0.0.1-alpha` |
| clone at the tag, **dirty** tree, no flags | `v0.0.1-alpha+dirty` |
| clone at a later commit, no flags | `v0.0.1-alpha.0.20260902184134-c9cf477bd361` |
| a `go test` binary | `(devel)` |

**Go stamps this well**, and every row is a fact about the bytes rather than about the machine —
which is the line ADR-0020 draws and the reason this is a second stamper rather than a version
resolved after the build. It is the exact tag where the source is the tag, it carries `+dirty` where
the tree was edited, and it degrades to a pseudo-version where the commit is not a release.

The row that costs something is the first. It is the README's own *From source* command minus a flag,
it is what anybody typing `go install` from memory gets, and ADR-0137 Claim 7 argues that the macOS
user this project is most likely to have is exactly that person — a developer who already has Go,
taking the path with no Gatekeeper story on it. They got a binary that ran, checked, and Refused
every repository it was pointed at.

## The decision this records, and what it costs

**It softens what #191 bought, deliberately.** The point of making `Version` a `var` the linker
writes was that a build nobody stamped says so, and after this a flagless build claims a version. The
question the ticket put is whether *this binary is 0.0.1-alpha, derived from the module the toolchain
recorded* is a weaker sentence than *this binary is unknown*, and it is not: it is a stronger one, and
it is checkable. What #191 was defending was that the page may not identify bytes it cannot vouch for
(#103) — and the module version is vouched for by the same toolchain that wrote `vcs.revision` on the
line below it, from the same repository, at the same instant.

The guarantee that does not survive is narrower than it looks: it was never *an unstamped build
cannot act*, but *a build with no version cannot act*. That still holds. A binary with no version from
either stamper reports `unknown` and Refuses exactly as before, and the build that gets there is a
`go test` binary, whose module version is `(devel)`.

**And the gate did not move.** `+dirty` and a pseudo-version are versions no release published, so a
build from an edited tree or from a commit no tag points at Refuses in every repository, and `project`
on one is still `release-artefact-absent`. What changed is which builds have an answer, not what the
pin accepts.

Two prices, both named in the ticket and both accepted:

- **The page's first line can get long** — `hyper 0.0.1-alpha.0.20260902184134-c9cf477bd361`. It is
  still one line and still cut by a script, which is the shape §9 fixed it for; the pin Refusal
  quoting it is verbose.
- **Two dirt markers can appear at once**: `hyper 0.0.1-alpha+dirty` on the first line and
  `commit 85244dd…-dirty` on the second. They are the same fact from two stampers and neither is
  wrong.

## Why the flag still wins

`scripts/release.sh` passes `-X`, so every archive a release publishes reports what the tag says and
this decision reaches none of them. That ordering is not a preference between two good answers: a
release's version is authored by the person cutting it, and the module version is derived from
whatever repository the build happened to stand in — which for a release is the same string, and for
a cross-compile from a laptop mid-change would not be.

The `v` is stripped because the tag carries it and no version `hyper` states does (§11), which is the
same rule `docs/build/releasing.md` states for the archive filenames under a tag.

## What fences it

- **`stampedVersion`'s own table** (`internal/version/stamped_test.go`) holds every row measured
  above, plus a build information carrying no module version at all — a guard rather than a state
  anything produces. Two of them answer `unknown`.
- **`TestStamp_AFlaglessBuildAtATagReportsTheTag`** builds `cmd/hyper` with no flags in a git
  repository the case makes out of this module's source, at a tag, and asserts the page and that a
  repository pinning that version clears the gate. The checkout a case runs in cannot supply that
  state — it sits at whatever commit its author left it at — which is why the repository is built
  rather than assumed. Go declines to stamp a major it would need an import-path suffix for, so the
  tag is `v1.4.0-test` and not `v9.9.9-test`.
- **`TestStamp_AFlaglessBuildFromAnEditedTreeDoesNotClaimTheRelease`** is where the fallback stops:
  the same repository with one file added reports `1.4.0-test+dirty` and Refuses the pin naming the
  tag, quoting both.
- **`TestStamp_TheLinkerWritesTheVersionTheBuildNames`** now also reads `Main.Version` off the binary
  it built, so the case can say which of the two stampers the page quoted;
  **`TestRelease_TheArtefactCarriesTheBinaryTheTagNames`** holds the same fact on bytes
  `scripts/release.sh` actually published.
- **`TestStamp_TheReleaseScriptNamesTheSymbolThatWorks`** is a fence this decision **moved**, and the
  one thing here that is strictly worse than before. `-X` names its symbol by import path and the
  linker checks it against nothing, so a typo publishes a binary nobody stamped — which used to
  report `unknown` and fail the case above. A release is cut from a clean checkout standing on the
  tag, and there the module version is the same string the flag would have written, so that failure
  is now invisible in exactly the state a release is cut in. The binary it publishes is still
  correct; what is lost is the detection, and what replaces it is comparing the script's spelling of
  the symbol against one the suite builds with and reads back out of a running process.
- **`internal/version`'s existing cases assert what they always did.** They run in a `go test`
  binary, whose module version is `(devel)`, so `TestVersion_AnUnstampedBuildSaysUnknown` and
  `TestCurrent_CarriesTheOneVersion` are unchanged. The first gained a paragraph saying why its name
  is now narrower than the rule it reads: the answer is a property of test binaries and no longer of
  every flagless build.

**No acceptance re-run is owed.** Nothing here is a repair to a sentence an agent reads and then
decides on: the change is enforced by the binary and held by the cases above, and the orientation
`project` writes is untouched (`docs/agents/acceptance-re-runs.md`).

## What was measured

The end of the chain was run rather than reasoned. This source was served as `v0.0.1-alpha` from a
`file://` module proxy — the module cache path, where there is no `.git` at all — and installed with
no flags:

    $ go install github.com/TheLoomLabs/hyper/cmd/hyper@v0.0.1-alpha
    $ hyper version
    hyper 0.0.1-alpha
    commit  unknown
    built   unknown

`hyper check` against a repository pinning `0.0.1-alpha` exits `0`, and `hyper project` in an empty
repository reached the real release host, read the published `checksums.txt`, and wrote

    version: 0.0.1-alpha
    digest: sha256:d9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096

which is the line `checksums.txt` publishes for `hyper-0.0.1-alpha-x86_64-linux.tar.gz`. That is the
ticket's first acceptance criterion end to end: reports the tag's version, passes the pin gate, and
can `project`.

**What it is not is a measurement against the published binary.** The proxy served *this* source
under that tag, because the fix is not in the release that tag names. The first release cut after
this is where a person can run the command as written.

## What it does not do

- **Nothing is read at run time.** No file, no flag, no environment variable, and no network. Both
  stampers write at build time and the page reports what the bytes carry, which is ADR-0014 and
  ADR-0019 untouched.
- **`hyper` still never hashes itself.** The gate compares a version string; the digest keeps doing
  its work where bytes cross a network (ADR-0020).
- **A build with a version is not a build that can `project`.** A pseudo-version names no release, so
  the second half of §11's *an unreleased binary runs and checks and cannot project* is the same
  sentence it was — it just now names the commit instead of saying `unknown`.
