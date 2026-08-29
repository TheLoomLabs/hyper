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

Thirty-one run against
[`../five-artefact-demo/repo`](../five-artefact-demo/README.md) — §3's own
`uptime` Manifest, one `read` Operation, no credential, `class: local`, and, for
the fifteen that make no call, the `cloudflare-dns` Manifest beside it — named
with the `--repo-dir` an operator would type. The rest carry a repository of
their own, each written for the one edge it drives, or fail before one is read
at all.

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

## The fifteen that make no call

`--response <path>` supplies the response object instead of fetching one, and
nothing in these cases dials anything (§9, ADR-0108). Fourteen of them read the
samples the demo repository carries — the fifteenth fails on its own argument
list, before a repository is read — a seventh kind of file in it, and not an artefact —
against the `cloudflare-dns` Manifest's three Operations, which is where this
corpus's `uptime` cannot reach: a credentialled Provider, a `mutate` whose
identity is a hole, a `series` whose fields root at a member, and a `destroy`
with no `record:` block at all.

| case | what it holds |
| --- | --- |
| `a-supplied-response-to-a-create`, `-json` | the `mutate` no Probe may invoke: `identity:` filled from `--input`, three fields off `$.body.result`, and the response marked supplied |
| `a-supplied-response-over-a-collection` | `over: $.body.result` named first, then one table per member — the two roots, with the second one actually used |
| `a-supplied-response-whose-paths-miss` | the same collection under another member name: `identity:` and two fields in `UNRESOLVED`, named once rather than once per member |
| `a-collection-one-member-of-which-is-short` | a field that resolved against one member and not the other: named **once**, and the Record that has it still renders it |
| `a-collection-path-that-landed-on-an-object` | the third answer a Run has no use for — an `over:` that resolved to something with no members is not an empty collection, and the line says which |
| `a-supplied-response-named-with-one-token` | `--response=<path>`, the second spelling of a value flag |
| `a-supplied-response-to-an-opaque-operation` | `probe shell read` answers, where the calling form refuses it: the opaque rule bounds an invocation and there is none |
| `an-operation-that-declares-no-record` | a `destroy` carries no `record:`, so there is nothing to project — a usage error rather than an empty page |
| `a-supplied-response-carrying-a-member-no-object-has` | `data` at the top level names a path root no Capability has, answered with the members that object carries |
| `a-supplied-response-that-names-no-host` | `host` is the member that survives a call that answered nothing, so an object without one is one no call could have produced |
| `a-supplied-response-outside-the-repository`, `a-supplied-response-that-names-no-file`, `a-supplied-response-that-names-a-directory`, `usage-response-with-no-path` | the four ways the path itself fails, all `2`: ADR-0089's bound, a name that is nothing, a name that is a directory and says so rather than saying it is missing, and no path at all |

The samples are checked in beside the repository rather than written by a case,
for the reason every other fixture there is shared: a response an author saved is
a file, and one copy of it read by fourteen cases is one that cannot drift
between them.

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
