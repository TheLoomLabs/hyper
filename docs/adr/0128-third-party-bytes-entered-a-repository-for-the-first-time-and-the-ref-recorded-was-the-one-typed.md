# Third-party bytes entered a repository for the first time, and the ref recorded was the one typed

**`hyper install` resolved four refs against a host nobody here controls, wrote
`providers/hetzner.yaml` three times and refused three times, and every branch it landed on held.**
The command is the single point at which third-party data enters a repository, and until now every
fetch it had ever made — in every test — was answered by an injected `capability.Dial`. Nine
invocations of the published `v0.0.1-alpha` `x86_64-linux` binary against
[`TheLoomLabs/hyper-install-fixtures`](https://github.com/TheLoomLabs/hyper-install-fixtures) now
stand behind it. No defect was found and nothing is repaired here. Issue #248.

**Nothing about the product changes.** What this records is that a set of claims which had only ever
been asserted in prose are now measured: that the recorded ref is the one the caller typed rather
than the one that answered ([ADR-0087](0087-a-ref-is-a-location-and-hyper-names-no-registry.md)),
that a publisher who omitted a trailing newline is not punished for the byte `install` itself wrote,
and that the `origin:` block makes the fetch's verification repeatable offline by a reader who never
performed it (§11).

## The host, and why it is not this repository

`TheLoomLabs/hyper` is private, so an unauthenticated read of anything it serves answers `404`, and
`install` sends no credential by construction ([ADR-0007](0007-hyper-never-stores-a-secret.md)). It
could not have served its own fixtures. A separate public repository was stood up instead, and that
is the better shape rather than the tolerable one: the ref points at a location nobody confuses with
the tool's own release, which is what an Extension's ref actually looks like in the world ADR-0087
describes.

Four directories, because `ChecksumsURL` replaces the ref's last path segment — so a **directory is
the unit of publication**, and a Manifest shares one with the digest that covers it. Each
`checksums.txt` is `sha256sum`'s own output, produced in that directory, so its line endings and its
spelling of the basename are the publisher's tool's rather than something composed to be parseable.

| directory | as published | digest |
|---|---|---|
| `good/` | 2691 B, ends `0a` | `785bdc93…`, and its `checksums.txt` says so |
| `mismatch/` | 2689 B, one line altered | `b7bfa872…`, and its `checksums.txt` says `785bdc93…` |
| `no-checksums/` | 2691 B, published with no digest beside it | — |
| `no-trailing-newline/` | 2690 B, ends `7d` | `e3c5b659…` |

The alteration in `mismatch/` is one line — `path: /v1/locations` reads `path: /v1/servers` — chosen
because it is what a substitution of this Manifest can usefully express. **A tampered Manifest cannot
redirect a request to a host the installing repository did not grant**, and the reason is the grant
rather than the Manifest: `ResolveHost` expands the Operation's `host:` template to a candidate set
and admits only its intersection with the Target declaration's `hosts:`, answering `host-not-granted`
where that is empty (§3, [ADR-0029](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md),
[ADR-0042](0042-a-probe-is-bounded-by-the-grant-it-binds.md)). A Manifest writing a literal host is
within the schema — `host:` is a required string and only its *holes* are constrained to
`from-target` or a declared enumeration — so nothing about the Manifest is what stops it; the
Target's grant is. That narrows what a substitution buys an attacker without closing the door: an
altered `path`, `query` or auth header still reaches the granted host under the Target's own
credential, which is why the digest is the check that matters.

Served bytes were confirmed byte-identical to committed bytes before any `install` ran. Both files
come back `text/plain; charset=utf-8` with zero redirects on `raw.githubusercontent.com`, and one
redirect on the `github.com/…/raw/…` alias over the same objects.

## Claim 1 — the two reads were answered, and one file landed

    $ hyper install https://raw.githubusercontent.com/TheLoomLabs/hyper-install-fixtures/main/good/hetzner.yaml
    PATH                    DIGEST
    providers/hetzner.yaml  sha256:785bdc933fe714353ed42bf0d76730c4285310828bcd68b535a27404112ac13b

Exit `0`. The repository held no `providers/` before it, so the directory `writeManifest` creates —
the first-Extension case, which is the common one — is what ran. The file is 2880 bytes: the 2691
published verbatim, and 189 of block.

    origin:
      ref: https://raw.githubusercontent.com/TheLoomLabs/hyper-install-fixtures/main/good/hetzner.yaml
      digest: sha256:785bdc933fe714353ed42bf0d76730c4285310828bcd68b535a27404112ac13b

`hyper check` answered `checked 3 artefacts: no problems found`, and `hyper review
providers/hetzner.yaml` rendered the block verbatim under the gutter with the header line `no
baseline — no Store` — the honest state of a repository that has installed an Extension and run
nothing. The repository was never given a Store: one is the record of Runs, and this session
performed none.

## Claim 2 — the ref recorded is the one typed, across a redirect

The same bytes were installed a second time through the alias that redirects:

    $ hyper install https://github.com/TheLoomLabs/hyper-install-fixtures/raw/main/good/hetzner.yaml
    PATH                    DIGEST
    providers/hetzner.yaml  sha256:785bdc933fe714353ed42bf0d76730c4285310828bcd68b535a27404112ac13b

    $ git diff
    @@ -112,5 +112,5 @@ operations:
     origin:
    -  ref: https://raw.githubusercontent.com/TheLoomLabs/hyper-install-fixtures/main/good/hetzner.yaml
    +  ref: https://github.com/TheLoomLabs/hyper-install-fixtures/raw/main/good/hetzner.yaml
       digest: sha256:785bdc93…

**One line moved.** ADR-0087 says the recorded ref is the caller's *across a redirect and regardless
of where the bytes came from*; the digest is identical because the bytes are, and the ref is not
because the typed strings are not. The second read went through the redirect too — `ChecksumsURL` is
derived from the typed ref, so the checksums fetch followed the alias rather than wherever the first
hop landed.

It also measures §11's *re-installing is how an Extension is updated, and the diff is the review*
without being written to. The file was overwritten whole, and what a reviewer reads is three lines of
diff.

## Claim 3 — the join `hyper` writes, and the only candidate that can clear it

`no-trailing-newline/hetzner.yaml` ends on `}`. Its published digest covers those 2690 bytes.

    $ hyper install https://raw.githubusercontent.com/…/no-trailing-newline/hetzner.yaml
    providers/hetzner.yaml  sha256:e3c5b659c813ab3d6d4d72b26a0c16df3c2700d13c106a3e26fa7d94d00fefae

    $ tail -c 220 providers/hetzner.yaml | cat -A
    {type: integer}$
    origin:$
      ref: https://raw.githubusercontent.com/…/no-trailing-newline/hetzner.yaml$
      digest: sha256:e3c5b659…$

    $ hyper check
    checked 3 artefacts: no problems found

**That clean `check` can only have come from the second candidate.** The byte range
`checkOriginDigest` takes is the file up to the start of the last line beginning `origin:` at column
0 — here 2691 bytes, because `install` wrote a newline the publisher did not — and its digest is not
`e3c5b659…`. The comparison against the prefix *less one trailing newline* is what recovers it. Two
candidates rather than a normalisation, and this is the run where the difference stopped being
hypothetical: normalising would have made the digest cover a canonical form rather than the bytes the
publisher published.

## Claim 4 — the mismatch refused, and the tree did not move

    $ hyper install https://raw.githubusercontent.com/…/mismatch/hetzner.yaml
    refused: origin-digest-mismatch
      https://raw.githubusercontent.com/…/mismatch/hetzner.yaml published sha256:785bdc93… and
      answered bytes that are sha256:b7bfa872…

Exit `77`, and `git status --porcelain` answered nothing. The `providers/hetzner.yaml` already
standing in the tree was byte-identical before and after — a stronger reading of *wrote nothing* than
an empty directory would have given, because the file the command would have overwritten was there
to be overwritten.

**Both digests in that message are digests of real files.** `785bdc93…` is reproducible by anyone
with `sha256sum good/hetzner.yaml`, and `b7bfa872…` is what the host actually served. That is the
consequence of manufacturing the fixture as a substitution after publication rather than as a
made-up hex string: the refusal names two facts a reader can check instead of one fact and one
fiction.

## Claim 5 — the two `404`s, and the one nobody typed

    $ hyper install https://raw.githubusercontent.com/…/good/absent.yaml
    hyper install: https://raw.githubusercontent.com/…/good/absent.yaml answered 404

    $ hyper install https://raw.githubusercontent.com/…/no-checksums/hetzner.yaml
    hyper install: https://raw.githubusercontent.com/…/no-checksums/checksums.txt answered 404

Both exit `1`, both leave the tree unchanged, and they are told apart by the coordinate alone. **The
second names a URL the caller never typed.** `install` derived it by replacing the ref's last
segment, so a reader who typed `…/no-checksums/hetzner.yaml` and was answered about
`…/no-checksums/checksums.txt` learns which of the two reads failed without knowing that there are
two.

**The message is left as it is, deliberately.** The temptation is to append *the Manifest* or *the
checksums file* to the sentence, and that is the mistake
[ADR-0127](0127-a-remedy-may-not-assert-what-the-answer-could-not-establish.md) has just finished
correcting one command over: a `404` cannot establish whether the file is absent, the directory is
absent, or the host is declining an unauthenticated reader. The URL is what the reader acts on, and
it is the one thing the status did establish.

## Claim 6 — the verification, repeated by a reader who never fetched

One line of the installed Manifest was edited, as a local author might:

    $ hyper check
    FILE                    LINE  FIELD          ERROR_CODE              MESSAGE
    providers/hetzner.yaml  116   origin.digest  origin-digest-mismatch  origin: digest: this Manifest's
      published bytes are sha256:b7bfa872… and the block records sha256:e3c5b659… — an edited Manifest is
      re-installed or has its origin: block dropped, never a digest retyped by hand

Exit `1`, positioned at the `origin.digest` scalar, under **the same `error_code` `install` refuses
with**. §11's claim that the block makes the fetch's verification repeatable offline, by anyone
reading the repository, long after the machine that performed it is gone, is measured here: nothing
in this invocation touched the network, and it reached the same verdict about the same bytes.

`b7bfa872…` is worth a sentence. The installed file was the newline-less publication, whose published
range is 2691 bytes once `hyper`'s own newline is counted; the one-line edit shortens it by two bytes
to 2689 — which makes it byte-identical to `mismatch/hetzner.yaml`, and it earns that fixture's
digest. The two ends of the mechanism met by arithmetic rather than by arrangement.

## #244's criterion, re-checked

Issue #244's acceptance criterion read: *`curl -fL` against the two URLs the README prints answers
`200`, fetched from outside the repository.* Unauthenticated, with `GITHUB_TOKEN` and `GH_TOKEN`
unset:

    …/releases/download/v0.0.1-alpha/hyper-0.0.1-alpha-x86_64-linux.tar.gz  →  404, curl exit 22
    …/releases/download/v0.0.1-alpha/checksums.txt                          →  404, curl exit 22

**That box was ticked from a shell holding a token.** The document is already repaired — #256 named
the `gh release download` route beside the `curl` pair — so nothing here is outstanding, and #244 is
not reopened: its deliverable was the tag, and the tag was delivered. What is recorded is that the
criterion measured the operator's credential rather than the world, and that the same fact has now
cost three tickets (#254, #256, this one) because a private release answers `404` in a way that looks
exactly like an absent one.

## What this does not establish

- **It is one host, and that host is GitHub.** Content types, redirect behaviour, line endings and
  `404` semantics are all GitHub's. A publisher serving from a bucket, a CDN with an interstitial, or
  a host that answers `200` with an HTML error page is untested — and the last of those is the one
  worth thinking about, since the digest is the only thing that would catch it.
- **No `429`, no `500`, no body that stops part way, and nothing over `MaxManifest`.** Those branches
  remain fixture-only; provoking them needs a host that misbehaves on request, which this one will
  not.
- **A Manifest published carrying an `origin:` block of its own** — where `install` appends a second
  and the check must key on the last — was deliberately not published. Its rule is a byte scan with
  no counterpart on the write side, and a real host adds nothing to what the suite already drives.
- **Nothing about `install` under concurrency, or into a dirty tree.** Every refusal here ran against
  a clean tree by construction, which is what made *unchanged* measurable.
