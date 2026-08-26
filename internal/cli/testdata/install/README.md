# What `install` fetched, and what landed in `providers/`

Twenty-four cases, driven through `cli.Main` from each case's own `argv` by the
one harness in `golden_test.go` (issue #187). Eight of them write a file, two
Refuse at the pin gate, and fourteen never reach a repository at all — the ref
grammar and the flags are decided from the argument list alone.

Every case that stands in a repository holds a `tree.golden`, on `project`'s own
rule one command over: the two text streams say what the command **reported**,
and only the tree says what it **did**. A case checking its stdout alone would
pass on a command that printed the right row and wrote nothing, or wrote it
somewhere else. `providers/` is the namespace this command writes into and the
reason that golden renders it at all (issue #184).

## The world a case stands

A case that fetches carries `serve/providers.example.com.json`, and what it
serves is what a *static file host* would — which is the whole of what a
registry is (§11, ADR-0087). The two reads are two entries in the host's
`answers` list, **in order**: the Manifest at the ref, then `checksums.txt` in
the ref's own directory. Nothing about the response is supplied by a case beyond
what a server would supply: the handshake, the certificate, the status line, the
body and the digest computed over bytes that crossed a socket are all real, and
only the name resolution is a fixture (`golden_serve_test.go`).

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
