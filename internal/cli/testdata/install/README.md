# What `install` fetched, and what landed in `providers/`

Thirty-eight cases, driven through `cli.Main` from each case's own `argv` by the
one harness in `golden_test.go` (issues #187 and #188). Eight of them write a
file, four Refuse, twelve fail a fetch, and fourteen never reach a repository at
all — the ref grammar and the flags are decided from the argument list alone.

Every case that stands in a repository holds a `tree.golden`, on `project`'s own
rule one command over: the two text streams say what the command **reported**,
and only the tree says what it **did**. A case checking its stdout alone would
pass on a command that printed the right row and wrote nothing, or wrote it
somewhere else. `providers/` is the namespace this command writes into and the
reason that golden renders it at all (issue #184).

That golden is load-bearing on the failing cases rather than incidental. **A
failed install must never leave a partial Provider behind**, and the fourteen
below that fetch and do not write each carry the same three lines — no
`.github/workflows/`, the declaration untouched, **no `providers/` directory at
all** — which is the only place in the case directory where *nothing was
written* is said (issue #188).

## The world a case stands

A case that fetches carries `serve/providers.example.com.json`, and what it
serves is what a *static file host* would — which is the whole of what a
registry is (§11, ADR-0087). The two reads are two entries in the host's
`answers` list, **in order**: the Manifest at the ref, then `checksums.txt` in
the ref's own directory. Nothing about the response is supplied by a case beyond
what a server would supply: the handshake, the certificate, the status line, the
body and the digest computed over bytes that crossed a socket are all real, and
only the name resolution is a fixture (`golden_serve_test.go`).

A case whose **first** read is the one that fails needs no list at all: it states
one response and the host answers it, the command having stopped before there was
a second request to answer differently.

The checksums body is `sha256sum`'s own output and names two files — the ref's
basename and one other — so that a case says *the line naming this file* rather
than *the only line there was*. `into-a-repository-that-does-not-check` writes
its line in the binary spelling, ` *name`, which is the other half of the one
grammar `internal/release` exports.

## The ref, and what is recorded

The ref every case is written against is
`https://providers.example.com/acme/dns.yaml`. There is no registry host in the
binary and no namespace `hyper` owns: the ref **is** the location, the digest is
published beside the bytes, and what is written into the `origin:` block is the
coordinate the caller typed (ADR-0087).

`the-recorded-ref-is-what-was-typed` is the case that makes that last clause
something other than a claim. Its ref answers a `302` to a second host, the
bytes arrive from `cdn.example.net`, and the block records
`providers.example.com` — because a recorded ref naming a redirect target is a
coordinate the publisher never published, and it would put a fact about somebody's
CDN into a reviewed file. The checksums read is derived from the typed ref for
the same reason, and its case shows that too: the second host is asked for
nothing at all.

## Every way it does not write

**Two answers and only two, and no sort is built between them.** §11 puts *a ref
the registry does not hold* and *a fetch that did not complete* on **one** exit
code deliberately: a ref names something in a registry's namespace, and *matches
nothing* is an answer that had to be fetched — it can differ between two
invocations of an identical command line, it is unavailable offline, and it
arrives beside the answers that are unambiguously the world resisting. So the
twelve cases below are all exit `1`: three status lines that mean *not here* —
two on the Manifest and one on the checksums file — a checksums file naming
every published file but this one, three that are the registry having a bad day,
three that never got a request onto the wire, a body that stopped part way and a
body over the cap.

This is where `install` parts company with `project`, and the corpora say so one
directory apart. `internal/release` splits the same status line, because
`release-artefact-absent` is a Refusal at `77` and `77` promises that a verbatim
retry Refuses identically — which a `404` keeps and a `429` does not. `install`
has no `77` on this path at all, so building the sort would be inventing a
distinction §11 spent a paragraph collapsing.

**The message names which of the two reads failed and what it answered.** *The
Manifest 404'd* and *the checksums file 404'd* are different acts for whoever has
to fix it, and one code carrying both is not a reason to render one sentence.

**`origin-digest-mismatch` is the one `77`**, and it is a check declining bytes
that did arrive: the read completed, the digest was published, and the remedy is
the publisher's rather than another attempt. Both digests render whole — a digest
is verified with `sha256sum` rather than recognised by eye, and ADR-0047's rule
that an id a human retypes renders whole forbids abbreviating either (ADR-0047).

The three that never reached the wire are stood by the host's own
`refuse_first`, and by the `refuse_first_as` beside it wherever the failure is
not the default: a connection nothing accepted is what an entry naming none
means, and `a-name-that-did-not-resolve` and `a-handshake-that-did-not-complete`
each name theirs. A case serving **no** host at all could not stand any of them —
the harness hands a case with no `serve/` entry at all a dialer that fails it
rather than one that refuses — so the host is named and its entry says it refuses
connections, which is the same silence written down.

`a-body-that-stopped-part-way` declares the Manifest's whole length and sends
fifteen bytes of it, which is what a connection dying behind a `200` looks like
from the far end. It is the one failure here whose error carries no coordinate of
its own — an unexpected EOF names nothing — so the read is what gives it one, and
the golden is where *which of the two reads was it* is held to that.

`a-manifest-over-the-cap` is the one case whose subject is a body's **size**: a
response larger than a Manifest read admits must be reported as a fetch that did
not complete rather than read into memory. It states its body once and a
`repeat` beside it, because the alternative is four megabytes checked into the
corpus — a fixture costing more to carry than the claim it makes.

## The cases

| Case | What it holds |
| --- | --- |
| `writes-the-manifest` | the whole act: two reads, a verified digest, `providers/` created, one row |
| `writes-the-manifest-json` | the same invocation's other rendering — one `manifest` row, then `result` |
| `the-recorded-ref-is-what-was-typed` | a `302` to a second host: the bytes come from there and the block records the ref that was typed |
| `a-published-manifest-with-no-trailing-newline` | `hyper` writes one newline of its own, and only there: the file is well-formed and the digest still covers the published bytes exactly |
| `an-existing-manifest-is-overwritten-whole` | re-installing over a `providers/` file already carrying an older block: whole-file, never merging |
| `a-published-origin-block-lands` | a publisher who shipped a block of their own gets a file with a duplicate key, which `check` reports — `install` reconciles nothing |
| `a-name-that-disagrees-with-the-ref-lands` | the path comes from the ref and never from the Manifest: a `provider:` that disagrees lands and is `check`'s to report |
| `into-a-repository-that-does-not-check` | the case the pre-write decision rests on: a Definition whose `provider:` names what `providers/` does not hold, and the install is the repair |
| `the-pin-the-binary-disagrees-with` | `version-pin-mismatch`: `install` stands behind the gate like the other fifteen, and nothing is dialled |
| `a-repository-with-no-pin` | `version-pin-absent`, the gate's other code |
| `usage-a-plaintext-scheme` | `http://` — the scheme is `https` and there is no second one |
| `usage-a-bare-path` | a ref is an absolute URL, not a name |
| `usage-an-absolute-path` | a path on this machine is not a location a registry serves |
| `usage-a-path-that-is-not-yaml` | the loader reads `providers/*.yaml` and nothing else |
| `usage-a-traversing-basename` | `..` is not a `providers/` filename — the clause is read before the `.yaml` one, or every traversal would be answered *this does not end in .yaml* |
| `usage-an-escaped-separator` | `%2F` in the last segment: an escape is judged as the character it decodes to |
| `usage-a-query` | a signed URL would put a token into a tracked file |
| `usage-a-fragment` | a fragment names a position inside a document, where a Manifest is a whole file |
| `usage-userinfo` | the one place this tool would write a secret down |
| `usage-no-ref` / `usage-two-refs` | the arity, decided from the argument list alone |
| `usage-dry-run` | `check` already reports digest drift and the diff is the rehearsal |
| `usage-limit` | `install` names no namespace to range over |
| `usage-unknown-flag` | the three globals and no fourth |
| `a-404-on-the-manifest` | a ref the registry does not hold: exit `1`, not `2` and not `77` |
| `a-410-on-the-manifest` | the other *not here*, on the same code |
| `a-404-on-the-checksums-file` | the second read, named as the second read |
| `a-checksums-file-naming-every-file-but-this-one` | the file is there and this ref's line is not |
| `a-429-from-the-registry` | an answer that arrived and is still not an answer about the Manifest |
| `a-500-from-the-registry` / `a-502-from-the-registry` | the world resisting, on the code it already lives at |
| `a-connection-nothing-accepted` | no request ever left the machine |
| `a-name-that-did-not-resolve` | the resolver's own failure, arriving through the dialler |
| `a-handshake-that-did-not-complete` | the TLS stack's, arriving the same way |
| `a-body-that-stopped-part-way` | a `200` whose bytes never all arrived — the one failure whose own error names nothing, so the read names it |
| `a-manifest-over-the-cap` | a body over the read's cap is a fetch that did not complete, not a laptop reading megabytes |
| `bytes-that-are-not-what-was-published` | `origin-digest-mismatch` at `77`, both digests whole, nothing written |
| `bytes-that-are-not-what-was-published-json` | the same Refusal in the other mode: a Refusal is not a row, so `--json` opens no stream to carry it |

## The fourteen cases with no repository and no `serve/`

That is itself the assertion. **Every clause of the ref grammar is a parse**, so
a ref outside it is decided with no network reached and no repository resolved —
which is the property ADR-0060 keeps exit `2` for, and a case that dialled
without a `serve/` entry fails on that alone (`golden_serve_test.go`). Each of
them is exit `2`, stderr, no `error_code` and no row stream: a usage error is
there being no act to decline, where a Refusal is the artefacts declining one.

## What `install` never does before it writes

It runs **no static pass**. `project` refuses to write where `check` would
report anything; `install` inherits none of that, and
`into-a-repository-that-does-not-check` is why: §4 puts an Extension the
repository never installed at `check` as `artefact-absent` on the Definition's
`provider:`, so the repository you install into is very often one that does not
check and the thing you are installing is the repair. A pre-write pass would
make the command unrunnable exactly when it is wanted.

It follows that `install` may write a file that immediately fails `check`, and
three cases here are exactly that. What each of them lands is asserted under
`testdata/check/` rather than here, one command over, because the code and the
line it is cited at are `check`'s:

```
a-published-origin-block-lands          → check/origin-block-published-twice
a-name-that-disagrees-with-the-ref-lands → check/installed-manifest-name-mismatch
into-a-repository-that-does-not-check   → check/an-extension-the-repository-never-installed (before)
                                        → check/an-installed-extension-checks-clean (after)
```

## What no case here can state

Two things, and they sit beside `project`'s own two in
[`install_test.go`](../../install_test.go). A **write the filesystem refuses
part-way** — arranged by standing a directory where the file goes, which needs
no permission bit set and behaves the same for every account the suite might run
as — naming the path it died on at exit `1`, with the tree left as it stands. And
a `2` for a malformed ref **dialling nothing at all**, which is an absence of
egress and therefore not a byte any golden renders: the fourteen cases above
assert it by the `serve/` they do not carry, and the count is asserted where a
count can be.

`install`'s whole code set is held there too — `2`, `1` and `77`, and no path to
a fourth — read off these cases' `exit.golden` files rather than driven again.
