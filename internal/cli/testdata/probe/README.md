# The first call

`hyper probe <provider> <operation>` is the smallest complete path through the
tool that touches the world: an artefact, a call, a response object, a
projection, a page (issue #135). This corpus drives every one of them.

## The call is real; only the name resolution is a fixture

A case that supplies a `serve/` directory has one in-process TLS server stood for
it, its certificate minted against the case's own `now`, and the dialer
`cli.Main` is handed maps every hostname the case serves to that listener.
`golden_serve_test.go` states it in full.

So a case exercises a real handshake, a real peer certificate, a real status
line, real headers and a real JSON parse. **Nothing about the response object is
written down by a case** — a `serve/<host>.json` says what a *server* answers, a
status, headers and a body, and nothing about what `hyper` does with it. That is
why `tls.days_left` reads `34` in every golden here: it is floored off an x509
chain the harness minted thirty-four and a half days after the case's instant,
not a number a fixture typed.

**A host with no `serve/` entry gets its connection refused**, which is the whole
of how *no response arrived at all* is driven — `a-host-that-answered-nothing`
probes a host `local` grants and the case does not serve, and the response object
comes back as `host` and nothing else.

## What each case is for

Sixteen run against
[`../five-artefact-demo/repo`](../five-artefact-demo/README.md) — §3's own
`uptime` Manifest, one `read` Operation, no credential, `class: local` — named
with the `--repo-dir` an operator would type. The rest carry a repository of
their own, each written for the one edge it drives.

| case | what it holds |
| --- | --- |
| `a-503-and-nothing-else`, `-json` | the demo: a `503` with no body, exit `0`, in both modes |
| `a-200-carrying-a-json-body` | `body` present and parsed |
| `a-host-that-answered-nothing`, `-json` | the object is `host` alone, and the Probe still renders and exits `0` |
| `a-typed-input-filling-a-query-hole` | `--input minutes=0015` reads as the integer 15 and fills a `query:` hole |
| `a-value-that-will-not-read-as-its-type` | the same input as `soon`: a usage error at `2`, carrying no `error_code` |
| `a-host-outside-the-grant`, `-json` | `host-not-granted`, `77`, naming the `hosts:` line to edit — stdout silent in both modes |
| `a-repository-declaring-no-local` | the same code with no declaration to point at: an absent declaration grants nothing |
| `a-local-that-grants-no-http` | the second way to have no line: a `local` that grants no `http` writes no `hosts:` at all, and grants nothing |
| `an-opaque-operation` | `probe shell read` is a usage error at `2` |
| `an-opaque-operation-on-a-repository-that-grants-shell` | the same, on a repository whose `local` grants `shell` and opts into `opaque-destroy:` — *whatever any Target grants* |
| `an-effectful-operation` | a Probe invokes a `read` Operation and nothing else |
| `a-manifest-that-names-no-host-input` | a candidate set and a grant intersecting to two hosts, on a repository that checks clean: the fault is decidable only at a binding, and a Probe is a binding no artefact wrote |
| `a-deadline-the-call-cannot-meet` | the Operation's `deadline:` bounds the call and is reported rather than hung on |
| `a-provider-matching-nothing`, `an-operation-matching-nothing`, `one-positional`, `an-input-the-operation-does-not-declare`, `a-declared-input-left-out`, `an-input-with-no-value`, `usage-unknown-flag` | the seven usage errors, all `2`, all with stdout completely silent — the last of them a near miss of `--input` itself, answered with the flags this command takes (issue #215) |
| `version-pin-mismatch-and-a-bad-operation` | the gate fires before either positional resolves: `77`, not `2` |
| `writes-nothing-at-all`, `-json` | a Probe beside a Store branch, and `store.golden` shows the branch it did not touch |

## Why `a-deadline-the-call-cannot-meet` declares `0s`

The Operation declares a deadline of zero seconds, so the bound is reached
before the dial rather than during it. What the case asserts is the *reporting*:
the deadline named on stderr, a response object of `host` alone, and exit `0`.
The bound cutting a call that is already in flight is
`internal/capability`'s own test, where a handler blocks until the test releases
it — a server that hangs for a golden second is a suite that takes a second
longer on every run to state something a millisecond already states.

## The two `-json` twins written for another corpus

`a-host-that-answered-nothing-json` and `writes-nothing-at-all-json` are here
for the fence rather than for themselves. An envelope's row is held against the
rows the corpus's `--json` streams write, corpus-wide and by the row's own
identity (`TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites`), and a
`probe_result` is identified by its Provider and its Operation — so the three
`uptime check_http` answers under `../mcp/probe/` are three renderings of one
identity and each needs a stream that writes it. A fixture only the second
surface drives has none.
