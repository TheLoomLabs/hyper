# Installing `hyper`

`hyper` is one binary and installs by being put on your `PATH`. It has no installer, no daemon
and no post-install step, and it never updates itself
([ADR-0019](adr/0019-hyper-never-updates-itself.md)).

The four commands that get you one are in the
[README's install section](../README.md#install). This page is everything else: what a release
publishes, how to check it, how to build from source, and what to do when a command Refuses
because of the version it read.

## What a release publishes

| Archive | Platform |
| --- | --- |
| `hyper-<version>-x86_64-linux.tar.gz` | Linux, Intel/AMD 64-bit |
| `hyper-<version>-aarch64-linux.tar.gz` | Linux, ARM 64-bit |
| `hyper-<version>-x86_64-darwin.tar.gz` | macOS, Intel |
| `hyper-<version>-aarch64-darwin.tar.gz` | macOS, Apple Silicon |

**There is no Windows build.** Each archive holds one file, `hyper`. Alongside them a release
publishes `checksums.txt`, which is `sha256sum`'s own output over all four
([ADR-0136](adr/0136-the-release-builds-every-platform-before-it-publishes-one.md) is why every
platform is built before any of them is published).

**Nobody signed or notarised these**, and on macOS that shows up in one place: a binary you
downloaded *in a browser* carries the quarantine attribute the browser set, so Finder refuses to
open it and offers **Move to Trash**. That is the attribute, not a verdict on the bytes — run it
from a Terminal, where it exits `0`, or install it with `curl` or `go install`, neither of which
sets one. What was measured, and on which machines, is
[`docs/build/releasing.md`](build/releasing.md).

## Checking the download

```bash
grep " hyper-$VERSION-$PLATFORM.tar.gz$" checksums.txt | sha256sum -c -
# where there is no sha256sum:                         | shasum -a 256 -c -
```

**Neither download carries a credential and neither needs one.** The release is public, and
reading `checksums.txt` by hand is the same act `hyper project` performs when it freezes the
digest for your platform into your Repository declaration
([ADR-0131](adr/0131-project-wrote-a-digest-for-the-first-time-and-the-network-contributed-one-scalar.md)).

That digest is inert on this machine — the version pin gate compares the *version*, and nothing
local ever reads the digest. It is not inert in a generated workflow, where it is the line a
runner checks fetched bytes against before it executes them. That is the reason not to
hand-write one.

## From source

Go 1.25 or newer, which `go.mod` carries.

```bash
go install github.com/TheLoomLabs/hyper/cmd/hyper@v<tag>
```

A binary reports the module version Go recorded wherever nothing stamped one
([ADR-0138](adr/0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)), so
that install reports the tag's version, clears the version pin gate in a repository pinned to
it, and can `project`.

**`v0.0.1-alpha` is the exception, and it is older than the change that made the flag
unnecessary.** Its published source reads only the linker, so a flagless install of *that* tag
reports `unknown` and Refuses everything. Until there is a later release to name, pass the flag:

```bash
go install -ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=0.0.1-alpha" \
  github.com/TheLoomLabs/hyper/cmd/hyper@v0.0.1-alpha
```

### From a clone

The same holds, and only **at such a tag, with a clean tree**:

```bash
mkdir -p ~/bin   # anywhere on your PATH
git checkout v<tag>
go build -o ~/bin/hyper ./cmd/hyper
```

Anywhere else that build reports what it honestly is — `0.0.1-alpha+dirty` from an edited tree,
`0.0.1-alpha.0.20260902184134-c9cf477bd361` from a later commit — and Refuses every repository,
neither being a version a release published. To make a binary from such a tree act on one, stamp
it, which is the release's own invocation and what
[`docs/build/releasing.md`](build/releasing.md) owns:

```bash
go build -ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=0.0.1-alpha" \
  -o ~/bin/hyper ./cmd/hyper
```

**The flag wins wherever it is given**, so nothing about a release changes: `release.sh` passes
it and the archives report what the tag says. What it no longer decides is whether a build has a
version at all.

### A `go install` at a version reports no commit

That is not a missing flag. Go stamps `vcs.revision` and `vcs.time` from the repository a
build's source sits in, and module mode builds from the module cache — a zip the proxy served,
with no `.git` in it — so `hyper version` answers `commit unknown` and `built unknown` under a
version that is right. The pin gate reads the first line and nothing else, so the binary
installs, checks and projects. Building from a clone stamps all three.

## When a command Refuses over the version

Fifteen of the sixteen commands compare the version the repository pins against the version the
binary reports, for exact equality — **`hyper` never hashes itself**
([§11](spec/12-distribution-and-version-pinning.md),
[ADR-0020](adr/0020-the-hyper-version-is-pinned-by-the-repository.md)). `project` is the
sixteenth and stands outside the gate, for being the pin's only writer.

| Refusal | Exit | What it means, and what fixes it |
| --- | --- | --- |
| `version-pin-absent` | `77` | `hyper.yaml` carries no pin. Run `hyper project`. Only `version`, `completions` and `mcp` stand outside this. |
| `version-pin-mismatch` | `77` | The binary is not the version the repository pins. Install the release the pin names, or run `hyper project` with the binary you mean to use and read the diff. A `+dirty` or pseudo-version here means the build came from an edited tree or an untagged commit — see [From a clone](#from-a-clone). |
| `release-artefact-absent` | `77` | `hyper project` asked the release host for a tag named for the version this binary reports, and there is none. Install from a release, build from the tag, or stamp the build. |

## Upgrading

Install the new binary, run `hyper project`, read the diff. That is the whole of it: `project`
rewrites the version pin and the digest, both land in a file you review, and nothing else on the
machine changes. `hyper` never updates itself.
