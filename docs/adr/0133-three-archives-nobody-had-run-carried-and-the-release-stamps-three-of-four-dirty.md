# Three archives nobody had run carried, and the release stamps three of four dirty

**Every archive `v0.0.1-alpha` publishes has now been downloaded and executed on a machine that
matches it.** `aarch64-linux`, `x86_64-darwin` and `aarch64-darwin` — the three that had never been
run by anybody — each fetched the published `.tar.gz` and `checksums.txt` over the network, verified
one against the other, and walked the quickstart: `version`, `check`, three `review`s, a read-only
`run`, an effectful `run`, and `runs`/`records`/`changes` read back. Nine green steps on each of
three machines, and six Runs on one Store branch of a public repository. `hyper` carries.

**Two things were found and neither is `hyper`'s.** Three of the four published archives report
`commit <sha>-dirty`, and the release process wrote the dirt: `dist/` is inside the checkout, so
every build after the first sees an untracked directory. And neither macOS archive passes
Gatekeeper's assessment — one is unsigned outright, the other ad-hoc signed by the Go linker and
notarised by nobody — which is what `releasing.md` already said, said in the wrong place and about
the wrong mechanism. Issue #247.

## What ran, and where

Three GitHub-hosted runners, dispatched from one workflow with `max-parallel: 1`, on 2026-09-02
against the release cut from `85244dd`. The labels were probed before the matrix was written rather
than assumed, and `macos-13` is gone — a job asking for it is never assigned.

| archive | machine | what it is | job |
|---|---|---|---|
| `aarch64-linux` | `ubuntu-24.04-arm` | Ubuntu 24.04.4, `aarch64` | 55s, green |
| `x86_64-darwin` | `macos-15-intel` | macOS 15.7.9 `24G830`, `x86_64` | 1m14s, green |
| `aarch64-darwin` | `macos-15` | macOS 15.7.7 `24G720`, `arm64` | 1m4s, green |

The scratch repository is [`TheLoomLabs/hyper-runner-fixture`](https://github.com/TheLoomLabs/hyper-runner-fixture),
#246's, reused: three Procedures over the built-in `shell` Provider and a `local` Target, and a
Store that already held three Runs. Nothing about the workflow is projected — it sits outside
hyper's `hyper-*.yml` namespace, where `project` does not speak for it and the projection check does
not see it, and `hyper check` answered *checked 8 artefacts: no problems found* on all three
machines throughout.

**The machine is named in band rather than by the label that asked for it.** `hyper version`'s fifth
line is `os/arch`, and each job asserted it with `grep -qx` — `os/arch linux/arm64`,
`os/arch darwin/amd64`, `os/arch darwin/arm64`. A matrix entry that had been handed the wrong
runner, or an archive built for the wrong platform, fails there rather than being written up here.

## Claim 1 — the three archives run, and the whole quickstart runs on them

Identical on all three, and the checksum was checked the way the README says to check it, with
`shasum -a 256 -c -` rather than `sha256sum`:

    hyper-0.0.1-alpha-aarch64-darwin.tar.gz: OK
    hyper 0.0.1-alpha
    …
    os/arch darwin/arm64
    checked 8 artefacts: no problems found

`review` rendered a Procedure with a Cadence, a Procedure with an effectful Step, and a Target
declaration — the gutter, the `AUTHORITY` table and `FLAGS`, in a terminal that is not this one's.
Then `run observe` (a `read` Step, `uname -sr`) and `run mark-alpha` (a `mutate` Step, `sh -c`),
both `completed · exit 0`, and `runs`, `records` and `changes observe` read the record back.

**Nothing about the pin gate is platform-shaped, and this is where that stops being a reading of the
code.** `hyper.yaml` pins `0.0.1-alpha` and freezes the digest of the **`x86_64-linux`** archive,
which is the only one §11's compiled-in template names. Three binaries that are not that archive ran
against that pin without complaint, because the gate compares the version string and nothing local
ever reads the digest — the digest is for a runner to check fetched bytes with, and a laptop is not
one.

## Claim 2 — one Store branch took writes from three platforms in turn

Nineteen commits landed on `hyper-store` between `511a4e6`, where #246 left it, and `0865cf7`. Six
Runs, two per machine, each pushed over the credential `actions/checkout` left behind. Because
`max-parallel: 1` held, the jobs are strictly ordered — 13:00:15–13:01:10, 13:01:26–13:02:40,
13:02:44–13:03:48 — and **each job's `hyper runs` lists every Run the machines before it wrote**.
The arm64 Mac's listing carries the Intel Mac's Runs, which carry the arm64 Linux machine's, which
carry #246's from a `x86_64` Ubuntu runner an hour earlier. The Store is bytes in a git branch and
crossing an architecture costs it nothing.

Serialising was the point of the flag rather than a finding: three jobs pushing at once would
contend for the branch, and a push rejected three times is `hyper`'s own failure mode
(ADR-0076) which would have been read here as a platform's.

**Six Runs wrote one Record commit between them**, and that is the Record model rather than a fault.
A Record's identity is the argv and its content is what came back, so `["uname","-sr"]` moved exactly
once in the sequence: `Linux 6.17.0-1022-azure` on the arm64 Ubuntu runner was the string #246's
`x86_64` one had already recorded — `THE WORLD MOVED  0 observations` — and the two macOS runners
both answer `Darwin 24.6.0`, so the Intel Mac moved it and the Apple Silicon one did not:

    CHANGE   TARGET  DEFINITION  RECORD           ORDINAL  FIELDS
    changed  local   host-reads  ["uname","-sr"]  1 → 2    stdout: changed

`uname -sr` names the system and its kernel release and says nothing about the machine, which is why
one line of `changes` output distinguishes Linux from Darwin and distinguishes neither architecture
from the other.

## Claim 3 — three of the four published archives claim a dirty tree, and none was built from one

`hyper version`'s second line is the commit the binary was built from, and `commit()` appends
`-dirty` where Go's `vcs.modified` is set, so that *a hash on the page is never a claim about bytes
edited after it*. On three of the four published archives that claim is false:

| archive | `vcs.revision` | `vcs.modified` |
|---|---|---|
| `x86_64-linux` | `85244dd` | `false` |
| `aarch64-linux` | `85244dd` | **`true`** |
| `x86_64-darwin` | `85244dd` | **`true`** |
| `aarch64-darwin` | `85244dd` | **`true`** |

Read out of the archives with `go version -m`, and read again off the runners, where every one of
the three printed `commit 85244dd1703f92c75f7c0915927ef5341479954f-dirty`.

**The release process wrote the dirt.** `release.yml` runs `bash scripts/release.sh "$VERSION" dist`
from the repository root, the repository has no `.gitignore` at all, and `release.sh` builds the four
platforms in a loop. The first build writes `dist/hyper-0.0.1-alpha-x86_64-linux.tar.gz` into the
checkout; every build after it sees an untracked `dist/` and Go stamps `vcs.modified=true`.
Reproduced from a clean clone at `85244dd`, two platforms, nothing else touched:

    $ git status --porcelain          # empty
    $ bash scripts/release.sh 0.0.1-alpha dist x86_64-linux aarch64-linux
    x86_64-linux   vcs.modified=false
    aarch64-linux  vcs.modified=true
    $ git status --porcelain
    ?? dist/

It is the ordering of a loop and nothing else: the platform that goes first is clean, and the three
laptop archives are dirty because they are not first. `cmd/hyper/release_test.go` cannot see it —
it builds one platform per case, and one platform is always the first one.

Nothing downstream reads it. `vcs.modified` reaches `version.Facts.Modified`, which reaches
`Page()`'s second line and nothing else; it is not `repo_dirty`, which is Provenance about the
**user's** repository and is computed per Run. What it costs is the one page an operator reads when
they want to know what they are running — §9's *this binary is 1.4.0* read on its own — telling them
the release was built from an edited tree. **#261.**

## Claim 4 — neither macOS archive passes Gatekeeper, and quarantine is not what stops one

`releasing.md` has said since #191 that the macOS archives are *neither signed nor notarised*. Half
of that is not quite right, and the sentence beside it is about the wrong mechanism.

**The two macOS archives are not signed the same way.** Read on the machines themselves:

    aarch64-darwin   CodeDirectory flags=0x20002(adhoc,linker-signed)
                     Signature=adhoc · TeamIdentifier=not set
                     codesign --verify --strict → valid on disk, satisfies its Designated Requirement
    x86_64-darwin    code object is not signed at all

That is Go's linker, which ad-hoc signs a `darwin/arm64` binary because Apple Silicon will not exec
an unsigned one at all, and leaves `darwin/amd64` alone because Intel will. Neither is notarised, and
`spctl` rejects both:

    aarch64-darwin   ./hyper: rejected
    x86_64-darwin    ./hyper: rejected  (source=no usable signature)

**And the quarantine attribute did not stop either binary from running.** `curl` set none, as
`releasing.md` says — that half is confirmed on both machines. Written by hand in the shape
LaunchServices writes, `0081;<timestamp>;Safari;<uuid>`, it did not change what happened:

    $ xattr -p com.apple.quarantine hyper-q
    0081;6a981ec8;Safari;1C3BE964-F3EB-45B5-AF5F-B70248DCBDD6
    $ spctl --assess --type execute --verbose=4 ./hyper-q
    ./hyper-q: rejected
    $ ./hyper-q version
    hyper 0.0.1-alpha
    …
    exit 0

Both architectures, `exit 0` quarantined and `exit 0` after `xattr -d`. **`spctl` assesses and the
kernel executes, and they disagree**: assessment is what Finder and `open` consult, and a binary
`exec`ed from a shell is not assessed. So the repair `releasing.md` names removes an attribute that
was not, here, blocking anything.

**This is evidence and it is not the walk the ticket asked for.** Both runners had
`spctl --status` → *assessments enabled*, which is why the question was worth asking at all, but both
also had **SIP disabled**, which is the runner image's doing and not a stock Mac's. And no browser
touched either machine: the attribute was written by hand, and what a real download sets — with what
flags, propagated through Safari's own unpacking rather than through `tar` — is untested. One
incidental measurement makes that gap concrete: a quarantined `.tar.gz` unpacked with `tar -xzf`
carried the attribute onto the extracted binary on the Intel runner and **did not** on the Apple
Silicon one, two macOS 15.7 machines differing on the propagation step the whole story runs through.
**#262** owns the walk, and needs a Mac.

`releasing.md` also said *the README says so where it says to download*. The README has never carried
an `xattr` line. Corrected in place, together with what the archives actually are.

## What this does not establish

- **The browser download has not been walked.** #262. Every quarantine fact above comes from an
  attribute written with `xattr -w` on a machine with SIP off.
- **No archive has run on a machine somebody owns.** Three ephemeral runners, created for the job
  and destroyed after it, all in one datacentre. A laptop's `$HOME`, its shell, its git and its
  filesystem case-sensitivity are still untried, and the runners' filesystems were never asked what
  they are.
- **`hyper install` never ran, and no Target reached the network.** Both Procedures are `shell` over
  `local`, so no `http` Capability, no credential and no TLS was exercised on any of the three
  platforms.
- **One macOS version, one point release apart.** 15.7.7 and 15.7.9. Nothing was run on macOS 26, on
  macOS 14, or on any Linux that is not Ubuntu 24.04.

## The decision this records

**A platform that publishes is a platform that has been executed.** All four now have, so nothing is
dropped from `release.sh`'s set — which was the alternative the ticket named and the one that would
have been taken if any of the three had failed. What follows from that is a standing cost rather
than a rule: a release publishes four archives and this session's evidence is about the release cut
from `85244dd`. #261 is a defect in how they are built; its repair is
[ADR-0136](0136-the-release-builds-every-platform-before-it-publishes-one.md), and it is only
visible in the release after next.
