# The acceptance endpoint is a local TLS server, and no artefact trusts it

**The Provider acceptance task talks to an HTTPS server the harness mints, starts, and stops on the
loopback, and the sealed session comes to trust its certificate through the `hyper` process's
environment — a position no artefact can name and no agent can author.** Issue #221 booked this as its
own ticket rather than a coin flip inside an implementation one, and issue #226 is where it is spent.

**Nothing about the product changes.** No Capability, no artefact key, no closed set, no flag. `hyper`
requests `https://` and only `https://` ([ADR-0082](0082-the-scheme-is-https-and-there-is-no-second-one.md)),
and the whole of what this decision needs is already true of it — which is the argument the trust
section below makes, and the reason a local endpoint is affordable at all.

## The question, and what makes it one

An acceptance task that asks an agent to author a Provider Manifest (issue #227) needs something for
that Manifest to talk to. Three facts bound it, and all three were established before the ticket
opened: the scheme is `https` and there is no second one, so a plaintext local server is not an option;
the seal keeps the network — `scripts/acceptance/run.sh` passes no `--unshare-net`, it hides the
checkout rather than the internet; and a local server in a task's setup script costs no new dependency.

So the question was never *can it* but *should it*, between a public endpoint, a local one, and
neither. What decides it is the list issue #227 actually needs: an `auth:` scheme, more than one
Operation across at least two Kinds — so one of them effects — and a `record:` projection that
**resolves**, which is what separates a Manifest that describes something from one that merely parses.
Read together that is a live credential, a real effect, and a response body whose shape a projection
addresses.

## The public options that are safe are not real, and the ones that are real are not safe

A real vendor's API satisfies the list by putting a live credential in a headless session running under
`--permission-mode bypassPermissions`, and asking it to perform an effect nobody reviewed. The seal is
a mount namespace and says so of itself — *it is not a security boundary and is not trying to be one* —
so nothing outside the repository bounds what that session does with the credential once it has one.
The worst outcome stops being a bad transcript and becomes a change to somebody's account that no
review stood in front of.

An echo service — `httpbin.org` and its kin — is safe because nothing it accepts means anything, which
is the same sentence as *it is not an API*. What it buys is somebody else's fixture reached over the
internet: the fragility of a network dependency without the realism that was the whole argument for
one.

And either way the transcript pays for a second variable in an experiment run a handful of times a
year. A rate limit, a schema change, a certificate rotation or an outage lands on exactly the axis
[ADR-0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md) spent a mount
namespace to control: a transcript is evidence about what an agent can do with the surface `hyper`
ships, and a run that died on a `429` costs a full session and answers nothing. A credential the
harness holds and rotates is a fourth cost on top.

**Running our own public service** was considered and is this decision with a DNS name, a hosting bill
and an internet-facing attack surface, bought for a fixture two people read.

## The port is forced, and the loopback is free

The sealed session is a `claude` process: it reaches an API or it does not exist. So `--unshare-net` is
not available, the sandbox shares the host's network namespace, and a server listening on `127.0.0.1`
outside the seal is reachable from inside it with no harness change at all. That was verified against a
seal of the harness's own shape — the checkout hidden, `--clearenv`, the same binds — where a full Run
against a loopback server completed and wrote its Records.

The same fact costs the port. Sharing the host's network namespace means an unprivileged process, and
an unprivileged process does not get `443` on an ordinary Linux, so the endpoint listens above 1024 and
the Target's grant carries the port: `hosts: [localhost:<port>]`.

**The number is taken from the kernel at startup rather than written down.** A fixed port is one
another process on the machine may already hold, and a task that failed on it would have failed for a
reason that is not the task's — the rule the golden corpus is already held to, where *a case that
reached the machine for one of them would be a case that fails on somebody else's*. What a free port
could cost is comparability between transcripts, and it costs none: `host:` is `"{from-target}"`, so
the port lives in the Target declaration the setup script writes and appears in nothing the agent
authors.

That a port in a grant works at all was checked rather than assumed. `check` calls the repository
clean, the candidate set and the grant compare by equality like any other host, `hyper` derives the
`Host` header from the value the grant was checked against, and the dialler is handed `host:port`
because the authority is what a URL's host is.

It is a corner the format neither admits nor rules out. `hosts:` is glossed as *enumerating hosts*, and
nothing inspects the string; [ADR-0087](0087-a-ref-is-a-location-and-hyper-names-no-registry.md) already
admits a port where a location is written, for the reason that a registry is wherever it is served.
**No transcript will be evidence about that corner**, because the Target declaration is shipped by the
setup script rather than authored by the agent — the shape issue #225 already used for its second
Target, a fact about the repository an operator hands over.

## The trust is the process's, which is what keeps the evidence clean

The server's certificate is self-signed, its own root, minted at startup with `localhost` and
`127.0.0.1` in its SANs and a validity in days. The sealed session trusts it because `SSL_CERT_FILE`
names it in the environment of the `hyper` the MCP server runs — Go's standard library reading its
roots, and nothing of `hyper`'s own. Without the variable the same repository answers *no response
arrived … so the request never left*, the Disposition on that Step `attempted-world-untouched`; with
it, the Run completes.

A two-level chain buys nothing a fixture needs, and one certificate is one file for the setup script to
write and the environment to name. Its validity is a number the harness chose rather than one a vendor's
rotation decided, so `tls.days_left` — the member ADR-0082 spends its argument on — is a known
quantity: a `one` projection over a thirty-day certificate recorded `days_left: 29`.

**The decisive property is that the trust is unreachable from where the agent writes.** There is no
position in any of the five artefacts that carries a root, a pin, or a verification mode; there is no
flag; and `internal/capability` holds no TLS configuration to override. So a Manifest that works against
this fixture is one that would work against a vendor, and the fixture cannot flatter the Manifest. Had
trust required a key in an artefact or a switch on the command line, the answer to this ticket would
have been *neither*: the transcript would have recorded an agent authoring something the shipped
product does not have.

One honest limit. Go reads `SSL_CERT_FILE` for its root *file* and `SSL_CERT_DIR` for its root
*directories*, so on a machine whose roots live in a hashed directory the fixture's root is **added**
and public hosts still verify — checked here, `example.com` verifying with the variable set. On a
machine whose roots live only in a bundle file it is **substituted**, and for the length of that run the
fixture's root is the only one. Nothing in this task reaches a public host, so it costs nothing; it is
written down so the next reader does not discover it as a surprise.

## What the setup script owns, and what `run.sh` owns

Stated to the line, and in the imperative: **none of it exists yet**, and issue #227 is what builds
it. A knob with no caller is a knob nothing holds true
([ADR-0104](0104-the-acceptance-fixture-ships-a-store.md)), so the seam lands with the task that uses
it rather than here. Harness work landing in the same commit as the task that needs it is the shape
issue #223 already took, `scripts/acceptance/run.sh` changing beside `fleet-rollout` — #221's *the
ticket's deliverable is the task file and its fence and nothing else* is about not running a sealed
session, not about leaving the harness untouched.

- **The endpoint is a Go program in the checkout**, beside the harness, built by the setup script the
  way `run.sh` already builds `hyper`: `go build -C "$root" -o "$outdir/bin/…"`, outside the seal,
  because the seal hides the source and not the binary. The setup script takes `$root` from its own
  path — it sits at `scripts/acceptance/tasks/`, three levels down — rather than from a third argument,
  `run.sh`'s own `root` being a local it does not export. The program mints its certificate at startup,
  listens on a free loopback port, and reports the port and the certificate's path.
- **`go` rather than `openssl` or `python3`, and that is a rule rather than a taste.** `run.sh` declares
  the tools it needs — `bwrap git go python3` — and the fence names the same four beside `bash`; issue
  #225 already turned down `sha256sum` for `python3` on this ground. `openssl` would be a fifth, and
  declaring it is an edit to the seam that issue #227 is required not to make, while reaching for it
  undeclared is a task that dies inside `run.sh` with no named cause on a host that lacks it.
  `python3` is declared and cannot mint a certificate — its standard library has no X.509 writer, and
  `cryptography` is not in it. `go` is declared, already builds the binary under test, and
  `crypto/x509` is the shape the suite's own TLS cases are written in. It buys one thing more:
  `go build ./...` compiles the endpoint on every change, where a script in another language is text
  nothing reads until a sealed run.
- **The endpoint checks a bearer token in `Authorization`, and the Target ships the `token` slot.**
  The scheme is fixed here because the Target declaration is shipped rather than authored, and a
  declaration carries the slots the scheme it binds requires: `auth: {token: {env: …}}` in the Target,
  `auth: {header: {name: Authorization, prefix: "Bearer "}}` for the agent to reach in the Manifest,
  and coverage checked at the binding (§3). A Target carrying `username` and `password` beside `token`
  would serve both schemes at once and measure which one an agent picks — a second question, and not
  one this task needs.
- **The setup script starts the service** and writes, into the output directory: the Target declaration
  granting `localhost:<port>` with that slot, the API's documentation as a file in the repository, a
  pidfile, and a file of `NAME=value` lines — `SSL_CERT_FILE` and the credential variable — for
  `run.sh` to fold into the MCP server's environment.
- **`run.sh` takes the lifetime.** It grows a second argument to the setup script, the output
  directory; folds that environment file into the `env` it writes into `mcp.json`, which is a fixed
  one-liner today; and kills the pid in a trap on `EXIT`, `INT` and `TERM`. The lifetime cannot be the
  setup script's: the fence runs the setup half on **every `go test ./cmd/hyper`**, so a service
  nobody stops is one process per test run, and `run.sh` is the only party that knows when the session
  is over. A `SIGKILL` of the harness leaks one process, and the pidfile is where a human finds it.

## `wontfix` — no endpoint, and the task ends at `check`

The live third answer, and the closest one. Most of §3's surface is checkable offline, which is issue
#227's own claim: a Manifest is data all the way down, so `operations:`, holes, `host-input:`,
`enumerations:`, `patterns:`, a scheme's slots against a Target's coverage, and the two keys a Step
must satisfy (§5) all have a `check` code waiting behind them with no host in sight.

What it cannot reach is the projection. **A Manifest declares no response schema** — `input:` is the
only schema in an Operation, and a `record:` path names a position in a body that exists only once the
call has gone out. So nothing offline can say whether `$.body.result.id` addresses anything, and an
offline task measures the grammar while leaving *does this Manifest describe the thing it points at*
exactly as assumed as it is today. That is the half the flagship exists to see.

It is recorded here as rejected rather than dropped, because it is the fallback: the offline half is
untouched by anything on this page, so a run whose endpoint failed to start still produces the grammar
evidence and loses only the last measurement.

## How this was checked

By hand, on 2026-08-29, against a `0.0.1-alpha` binary stamped the way `run.sh` stamps one, and a
repository in the fixture's shape carrying an authored Manifest, a Target granting `localhost:8443`
with a `token` slot, and the `read` and `mutate` Definitions that bind it. The endpoint was a TLS
server on the loopback holding a self-signed thirty-day certificate and checking `Authorization`.

What that established, in order: `check` clean over seven artefacts, the port included; a Run without
`SSL_CERT_FILE` stopping at the first Step, `attempted-world-untouched`, *the request never left*; the
same Run with it completing, its `over:` projection writing a Record per collection member and its
`one` projection recording `status: 200` and `days_left: 29`; a second Run reporting
`skipped-as-already-recorded` on the `mutate`; the whole of it repeated inside a `bwrap` namespace of
the harness's own shape, the checkout hidden and `--clearenv` set, with the same result; and
`example.com` still verifying with `SSL_CERT_FILE` pointed at the fixture's root.

**The verification server was a stand-in**, written in the shortest thing to hand rather than in the Go
this file settles on. Nothing it established is a fact about its language: the certificate is a PEM,
the trust is an environment variable Go's standard library reads, and the port is a socket.

## Consequences

- **Issue #227 is writable**, and what it inherits is an endpoint, a port, a trust story, a scheme, an
  owner for each half, and a language. What the API *says* is the task's to design; this file leaves
  it three constraints — JSON, an auth header it actually checks, and enough of a body for a `record:`
  projection to address — and the one below.
- **The fixture's shape is ours, and that is the cost paid.** An agent authoring a Manifest against an
  API we designed is being graded against our own idea of one, and a fixture that fits §3 too neatly
  would flatter the transcript. Hence the fourth constraint: be awkward where real APIs are awkward — an envelope key, a collection under a name, an
  identity that is not the name, a create whose response differs from an element of the list — and
  never show the agent the format.
- **There are no public docs, so the documentation is the fixture's**, and the file that carries it is
  where the answer can leak. It documents the API and never the Manifest: no §3 vocabulary, no artefact
  keys, no talk of projections or Kinds. A transcript that succeeded because the docs described a
  Manifest is one that measured our prose.
- **A Probe is not the discovery route.** A Probe is `read` Kind against `local` and may not reach a
  credentialled Target (§9), so the authenticated Provider under test cannot be probed. Whatever the
  agent learns about the response shape it learns from the documentation file or by calling the
  endpoint itself, which is a thing the sealed session's own tools can do.
- **`go test` will bind a loopback port** once the task lands, in that task's subtest, for as long as
  its setup half takes. That is the fence doing its job: a service that failed to start, or a
  certificate that failed to mint, fails the suite under the task's own name rather than in a session
  nobody is watching.
- **The credential is a fixture's.** It sits in the environment file and in `mcp.json`, both beside the
  repository and neither hidden by the seal, and it is worth nothing outside the process that checks it.
  `hyper` still never stores one ([ADR-0007](0007-hyper-never-stores-a-secret.md)): it resolves the slot
  from its own environment at Run start, exactly as it would against a vendor.
