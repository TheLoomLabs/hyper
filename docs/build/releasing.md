# Cutting a release

`hyper` learns its own version at build time. Nothing at run time reads it from
a file, a flag, or an environment variable, and nothing ever asks whether a
newer one exists (ADR-0019) — so what a binary claims to be is decided by the
invocation that built it, and this file is where that invocation is written
down.

It is a **process document, not a specification**. What a release must publish
so that the version pin works is §11's, and where it disagrees with
[`docs/spec/12-distribution-and-version-pinning.md`](../spec/12-distribution-and-version-pinning.md)
the spec is right and this file is stale.

## The stamp

One symbol, written by the linker:

```
go build -ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=0.0.1-alpha" ./cmd/hyper
```

`internal/version.Version` is a `var` for exactly this reason. `-X` writes a
string *variable*; a constant is inlined at compile time and the flag is ignored
without complaint, which is the failure mode that hid behind the declaration
until [#191](https://github.com/TheLoomLabs/hyper/issues/191) — every binary this
repository had ever produced reported the same placeholder however it was built.

**A build that omits the flag reports `unknown`**, which is the word every fact
the build did not stamp renders as ([#103](https://github.com/TheLoomLabs/hyper/issues/103)).
`hyper version` prints `hyper unknown`, and the pin gate's Refusal reads *this
binary is unknown*. That is the honest answer and it is also a dead end:
`hyper project` on such a binary asks the release host for a tag named for it,
is answered `404`, and Refuses `release-artefact-absent` — so an unstamped
binary runs and checks and cannot project, which §11 states as the consequence
it is. `go install`, a `go build` with no flags, and a `go test` binary are all
unstamped builds.

## What a release publishes

Two kinds of file under one tag, and the binary names both by a template it
holds and cannot be argued out of (§11's four compiled-in constants):

```
https://github.com/TheLoomLabs/hyper/releases/download/v0.0.1-alpha/hyper-0.0.1-alpha-x86_64-linux.tar.gz
https://github.com/TheLoomLabs/hyper/releases/download/v0.0.1-alpha/checksums.txt
```

The **tag carries the `v` and no filename under it does.** Each archive holds
the binary as `hyper` at its root, because the install step in a generated
workflow untars into the checkout and invokes `./hyper` (§10). `checksums.txt`
is `sha256sum`'s own output, one line per archive naming a bare filename —
which is the line `hyper project` reads to freeze the digest into the Repository
declaration, and the only thing the pin ever reaches the network for.

Exactly one platform is `hyper`'s business: the one the projection's `runs-on`
names, `x86_64-linux`. What a release publishes beyond it is the release
process's own, and three are published beside it for laptops —
`aarch64-linux`, `x86_64-darwin` and `aarch64-darwin`. There is no Windows
build: no `runs-on` here names one, so it would be a platform the release
carried and nothing tested.

The macOS archives are **cross-compiled from the Linux runner and are neither
signed nor notarised.** `curl` sets no quarantine attribute so a fetched
archive runs, and a browser download does — `xattr -d com.apple.quarantine
./hyper` is the repair, and the README says so where it says to download.

## Cutting one

1. Land everything, on `main`, with the tests green.
2. Push the tag: `git tag v0.0.1-alpha && git push origin v0.0.1-alpha`.
3. [`.github/workflows/release.yml`](../../.github/workflows/release.yml) does
   the rest — it pins the toolchain to go.mod's directive, builds through
   [`scripts/release.sh`](../../scripts/release.sh), checks that the binary
   inside the artefact reports the tag, and creates the release with the
   archives and `checksums.txt`.

Nothing else publishes, and the version is never written into a tracked file
here: the tag is the only place a release's version is authored, and every
filename under it derives from that one string.

The same script builds locally, which is what the cases in
[`cmd/hyper/release_test.go`](../../cmd/hyper/release_test.go) run:

```
scripts/release.sh 0.0.1-alpha dist x86_64-linux
```

## The first release, and what it unlocks

Until a release exists, `hyper project` cannot bootstrap a pin in any
repository — the checksums file it reads answers `404` and it Refuses
`release-artefact-absent` at exit `77`, correctly. The upgrade ritual ADR-0020
fixes — install, `project`, read the diff — needs one published release before
it can run end to end, and the tag above is what supplies it.

After that, an upgrade is the same three acts every time, and the pin's two arms
are both reachable with binaries anyone can download: the repository's `version:`
against what the running binary was stamped with, compared for exact equality
and nothing else (§11).
