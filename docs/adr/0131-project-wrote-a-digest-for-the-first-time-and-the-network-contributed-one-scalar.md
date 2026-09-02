# `project` wrote a digest for the first time, and the network contributed one scalar

**`hyper project` took its success branch against `https://github.com`, exited `0`, and froze the
published checksum of `hyper-0.0.1-alpha-x86_64-linux.tar.gz` into a repository that had no
`hyper.yaml` at all.** Every invocation the command had ever had — in the suite, in six sealed
acceptance sessions, and twice in the #249 session — had resolved a `404` and Refused
`release-artefact-absent` at exit `77`, or had found the pin already equal to the binary's version
and resolved nothing. The branch that writes a digest had never run against a published file. It has
now, three times, with the fetch proved load-bearing by taking the network away. No defect was
found, and nothing is repaired here except two sections of README. Issue #245.

**What unblocked it is not `hyper`.** #244 cut the tag on 2026-09-01 and this ticket did not clear:
`TheLoomLabs/hyper` was private, `project`'s read carries no credential by construction
([ADR-0007](0007-hyper-never-stores-a-secret.md)), and a release nobody may read unauthenticated
answers `404` exactly as an absent one does
([ADR-0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md),
[ADR-0127](0127-a-remedy-may-not-assert-what-the-answer-could-not-establish.md)). The repository was
made public on 2026-09-02, which is the fourth shape §11 names going away, and the shape this ticket
was waiting to test became reachable.

## The two repositories, and why there are two

Both are copies of `internal/cli/testdata/project/a-repository-with-no-pin/repo` — a Manifest
declaring `header:` auth over a `read` and a `destroy` Operation, a Target granting `http` to
`api.cloudflare.com` with one credential slot, two Definitions, and one Procedure carrying
`cadence: "0 3 * * 1"` and a `destroy` Step. They are the fixture rather than the quickstart's
`shell` shape because a `shell` Procedure projects a workflow with no `env:` block, and the `env:`
key is half of what the tree comparison is for.

| | pin as it stood | what `frozenDigest` does |
|---|---|---|
| the network run | **no `hyper.yaml`** | `declared.version` is empty, is not `0.0.1-alpha`, so the fetch happens |
| the offline run | `version: 0.0.1-alpha` + the published digest, written by hand | the pin already equals the binary's version, so **nothing is fetched** |

The binary is the published `x86_64-linux` archive, downloaded and checked by hand
(`hyper-0.0.1-alpha-x86_64-linux.tar.gz: OK`), never a local build:

    hyper 0.0.1-alpha
    commit  85244dd1703f92c75f7c0915927ef5341479954f
    built   2026-09-01T09:42:11Z

## Claim 1 — exit `0`, and the declaration a repository with none was given

    $ hyper project
    PATH                                            PROCEDURE           CADENCE
    .github/workflows/hyper-retire-preview-dns.yml  retire-preview-dns  0 3 * * 1
                                                                        03:00 UTC every Monday
                                                                        ≈4.3 runs/month
                                                                        scheduled runs happen on the default branch only
                                                                        :00 is the executor's busiest minute — delivery there is likeliest to be delayed or dropped

Nothing on stderr. `git status --short` answered three untracked paths — `.github/`, `AGENTS.md`,
`hyper.yaml` — which is the whole of what §11 says this command writes, and the 130 bytes it created
are three keys and **no `retention:`**:

    kind: repository-declaration
    version: 0.0.1-alpha
    digest: sha256:d9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096

## Claim 2 — the digest equals the published line, compared by hand

    frozen into hyper.yaml   d9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096
    checksums.txt says       d9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096  hyper-0.0.1-alpha-x86_64-linux.tar.gz
    sha256sum of the archive d9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096

Three ways to the same 64 characters: what the command wrote, what the publisher published, and what
the bytes on this machine hash to. **The third is the one the first two do not imply.** `Digest`
reads the checksums file and never the artefact — a few hundred bytes rather than eight megabytes,
both being the same mutable source read in the same instant — so a published line that did not
describe the published archive is a thing `hyper` cannot see and this comparison can.

## Claim 3 — what the world answered, which no injected `Dial` models

Two TLS connections, in one process, observed at the syscall:

    connect(3, {sin_port=htons(443), sin_addr=inet_addr("140.82.121.4")})      lb-140-82-121-4-fra.github.com
    connect(3, {sin_port=htons(443), sin_addr=inet_addr("185.199.110.133")})   cdn-185-199-110-133.github.com

The second address is in `release-assets.githubusercontent.com`'s A set. **The redirect is real and
it crosses hosts**, which is `Client`'s documented departure from a Capability's call — there is no
grant here and no authored host, so a fetch that refused to follow one would resolve nothing
anywhere ([ADR-0029](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md)). The hop:

    HTTP/2 302   content-type: text/html; charset=utf-8   content-length: 0
                 location: https://release-assets.githubusercontent.com/…?<signed query>
    HTTP/2 200   content-type: application/octet-stream   content-length: 420

**The `200` is `application/octet-stream`, and nothing in `Digest` consults it.** That is worth
stating as a thing that held rather than a thing nobody thought about: a checksums file is text, the
obvious defensive reflex is to require `text/plain`, and a `Digest` that had would Refuse
`release-artefact-absent` against a file that names the artefact perfectly well. The `MaxChecksums`
cap is what actually covers *a body that is not the file it should be*, and it is the right check
because it is about size rather than about a header the publisher did not choose.

The 420 bytes are `sha256sum`'s own output and nothing else: four lines, ASCII, LF endings with no
`CR` anywhere, the two-space text-mode separator, a trailing newline, bare basenames with no `v`.
`DigestIn`'s tolerance of the ` *` binary-mode spelling was never exercised — the release runner used
text mode — and the tolerance is what makes that an observation rather than a dependency.

## Claim 4 — take the network away and the same command reaches a different exit

A fresh copy of the same artefacts, still with no `hyper.yaml`, the same binary, under
`bwrap --unshare-net`:

    $ hyper project
    hyper project: the checksum for 0.0.1-alpha did not arrive: Get "https://github.com/…/checksums.txt":
      dial tcp: lookup github.com on 127.0.0.53:53: read udp …: connection refused

Exit **`1`**, not `77`, and no Refusal line. This is the distinction §11 draws and `release.go` is
explicit about, arriving from the world rather than from a fixture: a Refusal promises that a
verbatim retry refuses identically, and a host that never answered promises the opposite — so a fetch
that did not complete is `1`, and it is `install`'s own rule one command over
([ADR-0060](0060-naming-nothing-is-a-usage-error-fetching-nothing-is-not.md)). It also
settles what Claim 1 could otherwise only infer: **the fetch is load-bearing.** The digest is not
lying about somewhere on disk, because without a network there is no digest and no `hyper.yaml`.

## Claim 5 — the network run wrote the offline run's bytes, exactly

The offline repository was projected under the same `--unshare-net`, where `curl` cannot resolve
`github.com`, and its tree compared with the network run's:

    $ diff -r --exclude=.git hyper-245-net hyper-245-offline
    (no output)

| path | bytes | sha256 |
|---|---|---|
| `hyper.yaml` | 130 | `e4c81a9e…d229` |
| `AGENTS.md` | 16083 | `aab47072…04b4` |
| `.github/workflows/hyper-retire-preview-dns.yml` | 1564 | `3546512b…e3a3` |

Identical on both sides, and the two stdout streams are identical too. The `hyper.yaml` line is the
one that had to be argued for rather than observed: the two files were produced by **different code
paths** — `pin.Written` creating a declaration where none stood on the network side, and editing two
scalars in place in a file a hand had written on the offline side — and they agree to the byte
because the scalars agree. **The network's entire contribution to a projection is one scalar.** Everything else — the workflow's `cron`, its
`concurrency` group, its `env:` key derived from the Target's credential slot, the pinned
`actions/checkout` SHA, the orientation's sixteen kilobytes — is derived from reviewed artefacts and
from constants compiled into the binary, and the fetch supplies 64 hex characters that then appear in
exactly two places. `AGENTS.md` is the sharpest case: it is not a copy of anything on disk and it
came out of the same binary with no host reachable.

## Claim 6 — the install step it wrote installs

The four shell lines of the generated workflow's `install hyper 0.0.1-alpha` step, taken out of the
file and run verbatim on this machine:

    curl -fsSL -o hyper.tar.gz \
      https://github.com/TheLoomLabs/hyper/releases/download/v0.0.1-alpha/hyper-0.0.1-alpha-x86_64-linux.tar.gz
    echo 'd9a64425368560358e5931b8de389a36f207d275e935c54a4bd5eb59c3db4096  hyper.tar.gz' \
      | sha256sum -c -
    tar -xzf hyper.tar.gz

    hyper.tar.gz: OK
    $ ./hyper version
    hyper 0.0.1-alpha

**The digest `project` froze is the digest a runner will check, and it clears.** This is not #246 —
the workflow has still never run on a runner — but it removes the one thing about the install step
that could only have been asserted: that the URL the template composes and the checksum the fetch
froze describe the same bytes, and that `sha256sum -c -` is satisfied by them.

## Claim 7 — the quickstart, run as the README will now print it

The third success-branch invocation was a bare repository — `git init`, three empty artefact
directories, nothing else — which is the shape the README's quickstart starts from:

    $ hyper project
    no Procedure declares a Cadence, and no generated workflow stands

Exit `0`, `hyper.yaml` and `AGENTS.md` written, no `.github/`. The three quickstart artefacts were
then written against the built-in `shell` Provider and the rest of the quickstart run in order:

    $ hyper check
    checked 5 artefacts: no problems found

    $ hyper store init
    created  hyper-store
    wrote    STORE.md

    $ hyper run say-hello
    completed · exit 0 · run 01a061af-0174-7a2c-b840-96f79b480350

**`check` counts five with no `retention:` in the declaration**, which is the one thing about the
new first step that could have surprised a reader: `project` writes `kind:`, `version:` and
`digest:` and authors no policy on the repository's behalf, and nothing downstream minds.

## #244's criterion, ticked a third time and true this time

ADR-0128 records that #244's *`curl -fL` against the two URLs the README prints answers `200`,
fetched from outside the repository* was ticked from a shell holding a token, and answered `404`
unauthenticated. From an environment with no token and no credential file reachable at all
(`env -i … HOME=/nonexistent`):

    …/releases/download/v0.0.1-alpha/hyper-0.0.1-alpha-x86_64-linux.tar.gz  →  200, curl exit 0
    …/releases/download/v0.0.1-alpha/checksums.txt                          →  200, curl exit 0

The README's install block was then run start to finish exactly as it is printed — both `curl -fLO`s,
the `grep … | sha256sum -c -`, the `tar`, and `hyper version` reporting `hyper 0.0.1-alpha`. **The
criterion is now met by the world rather than by the operator**, and it took four tickets (#254,
#256, #248, this one) to get there.

## What moved in the repository

Two sections of README, and nothing else. `hyper` is unchanged: no code, no test, no golden, no
constant. The README documented the private-repository state as current, in the two places it is
load-bearing for a reader following instructions —

- the install block's *while this repository is private, the two `curl`s answer `404`*, with the `gh
  release download` route #256 added beside it. The block itself closed on *the `curl` pair is what
  will be right for every reader the day this repository opens*; that day is the subject of this
  ADR, so the paragraph and its `gh` fallback are gone and the `curl` pair stands alone.
- the quickstart's *one step here is a workaround, and it is the `digest:`*, which had the reader
  write `hyper.yaml` by hand with sixty-four zeros because `project` could not succeed. It can, so
  step 1 is now `hyper project` — as §11 and `docs/build/releasing.md` have said throughout — the
  quickstart writes three files rather than four, and the placeholder digest is gone from the
  document that was propagating it. The section itself becomes *step 1 is the only one that reaches
  the network*, which keeps the one paragraph worth keeping: the digest is inert on this machine
  and is not inert in a generated workflow, which is why nobody should hand-write one.

Every command block in both edits was run against the published release before it was written down,
which is Claim 7.

**ADR-0125, ADR-0127 and ADR-0128 are left exactly as they are.** Each states what was true when it
was written and each is a record of a measurement rather than a description of the present; the
convention that factual errors are corrected in place is about errors, and none of those was one.

## What this does not establish

- **A rate limit told apart from an absence — still unobserved, and it is the one `release.go` is
  most explicit about.** `absentAt`'s rule that `404` and `410` are the absence and every other
  refusal is not — that a `429` or `503` must reach exit `1` rather than tell an author to publish a
  release that is already published — remains fixture-only. Provoking it needs a host that will
  misbehave on request, and this one will not. The `1` branch *was* reached against the world, by
  Claim 4, but through a dialer that never connected rather than through a status line.
- **`410` was never seen.** GitHub answered `404` for an absent tag and `404` for an absent asset
  under a real tag; nothing here produced the other member of the pair.
- **One host, and it is GitHub.** The redirect shape, the signed second hop, the content type and the
  `404` semantics are all GitHub's. A publisher answering `200` with an HTML error page is untested,
  and is the case where only the digest would catch it — which for `project` means the diff a human
  reads, since `project` verifies nothing about the line it freezes beyond the line it is on.
- **One platform and one line of one file.** `ArtefactName` names `x86_64-linux`; the other three
  published archives are #247 and were not read here beyond their `checksums.txt` lines.
- **The upgrade, as opposed to the bootstrap.** Both runs here had a declaration that was absent or
  already correct. A pin at a version the binary disagrees with, fetched and replaced against a real
  release, cannot be run at all until a second release exists — `the-pin-the-binary-disagrees-with`
  stays fixture-only, and what it adds over the bootstrap is `pin.Written` editing an existing file
  rather than creating one.
- **The generated workflow has still never executed on a runner** (#246). Claim 6 is its install step
  run by hand on a laptop, which is a different thing from `ubuntu-24.04` running the whole file.
