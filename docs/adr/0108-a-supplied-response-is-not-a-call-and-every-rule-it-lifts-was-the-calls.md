# A supplied response is not a call, and every rule it lifts was the call's

**A Manifest author sees the response their projection reads from by fetching it themselves, and
`hyper` answers what the projection makes of it.** `probe <provider> <operation> --response <path>`
reads an Operation's `record:` block against a response object the caller hands over — the Records it
would have written, and, beneath them, every path that resolved to nothing. It performs nothing: no
host is reached, no credential is resolved, no Target is read, and no Record and no Journal entry is
written. Issue #230 opened this as a design question with three shapes and no fix proposed; this is
the fourth, and it is the only one that reaches the projection the question is actually about.

## The question, and why `check` is not where it lives

A `record:` projection reads from the response object `hyper` builds
([ADR-0040](0040-an-http-response-is-an-object-hyper-builds.md)), and nothing in the tool showed that
object for the Provider being authored. §4 states the limit honestly: **no artefact says what an API
returns**, so `check` reaches the grammar of a path and stops there. `$.body.result.id` is well-formed
against a response that carries it and against a response that does not, and the two are the same
artefact.

[ADR-0017](0017-the-wire-is-visible-only-where-no-credential-was-used.md) narrowed half of that and
said so in as many words — *authoring against a public unauthenticated API is now the tightest loop on
the map … authoring against a credentialled API is exactly as hard as it was*. A Probe renders the raw
response beside the projection because it binds `local`, holds no credential and writes nothing, which
is the one place the hazard is absent rather than mitigated. The half it left standing is the half
every real Provider lives in.
[ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md) recorded
the consequence in passing — *a Probe is not the discovery route* — without it being anybody's ticket.

The flagship acceptance task (issue #227) is where that came due. It asks an agent to author a Manifest
against an API it has never seen, and the claim it exists to measure is that the Manifest *describes*
the API rather than merely parsing — which turns on the projection, the one part no static check can
reach. The sealed run of 2026-08-29 succeeded, and spent roughly half of a five-minute session
inventing a way to look at a response.

## The loop we had is the bucket this corpus already closed

What that session invented, and what an author with no orientation invents next: **author an Operation
whose projection captures the body, run it, read the Records back, then delete the Operation.** It
works. It also cannot be blessed, and the reason is not taste.

A projected field is written out as *the JSON it is* wherever it is not a string
(`internal/projection`, `Text`), so `fields: {raw: $.body}` is writable today and puts the whole
response body on a Record version. That version goes into the Store, which is append-only
([ADR-0011](0011-the-store-is-append-only.md)) and travels in the repository
([ADR-0006](0006-the-record-travels-in-the-repository.md)), and what takes it out again is `compact`
and nothing else. So the loop is exactly the catch-all bucket ADR-0017 refused — *there is no
catch-all bucket for an undeclared token to hide in* — reinstated by the author, on our own
instructions, and aimed at the **worse** of the two sinks. A terminal scrolls; a git branch does not.
The session that found this out named the residue precisely and correctly declined to reach around
`compact` to remove it.

Documenting that loop was the cheapest of issue #230's three shapes and possibly the right one. It is
not: what it documents is a hazard the design spent two decisions closing.

## Widening the Probe answers the smaller half of the question

Issue #230's first shape — a Probe that binds a Target, or a Definition — spends
ADR-0017 whole. That is a real price and it is not the decisive one. **A Probe that calls can never
reach a `mutate`'s projection**, because looking would be the effect: invoking a create to find out
what a create answers is an unreviewed change to somebody's account, which is the one thing this tool
exists to make impossible.

And a `mutate`'s projection is precisely where the difficulty is. ADR-0105's own fourth constraint on
the acceptance API names it — *a create whose response differs from an element of the list* — because
that is where real APIs are awkward and where a Manifest that merely parses comes apart. A widened
Probe buys the `read` half of the wall, at the cost of the argument that made the Probe safe, and
leaves the half that matters exactly where it was.

## The decision: the world is the author's to call, and the projection is `hyper`'s to read

The author already holds the credential and already has an HTTP client. What they cannot get from
`curl` is the object §12 closes and the grammar §12 closes over it: that `body` is a member and not
the root, that a `series` Operation reads from **two** roots written with one marker, that a header
name is lowered, that `tls.days_left` exists because no artefact could compute one. Those are
`hyper`'s construction, and whether a path addresses anything in a given response is a question about
that construction and about nothing else.

So `hyper` does not fetch it. It reads it.

**Every rule the supplied form lifts bounded a request leaving this machine.** A Probe invokes a
`read` and nothing else ([ADR-0009](0009-a-probe-is-not-a-run.md)); it may never invoke an `opaque`
Operation whatever any Target grants (§9); its host is checked against what `local` grants
([ADR-0042](0042-a-probe-is-bounded-by-the-grant-it-binds.md)). With no request, each of those is
**vacuous rather than skipped** — which is the shape ADR-0009 used for the two-key authority check
against `local` and ADR-0042 used again for the Kind half of it. There is nothing on one side of the
intersection because there is no call to bound. What is left is a Manifest, a grammar, and a JSON file
the caller wrote, and none of the three touches anything.

The same reading decides the inputs. Every declared input is supplied because there is no null and no
key-omission syntax, so an input left out has no sink to render at
([ADR-0081](0081-a-value-is-read-against-the-schema-at-its-position.md)) — and **the sink is the
request**. With no request there is no sink, so inputs are optional here; what one still reaches is an
`identity:` written as a template hole rather than a path, which resolves before a call is made at all
and is the shape `skip-if-recorded` requires. An input nothing supplied leaves that identity
unresolved, and the page says so.

## What it costs against ADR-0017 and ADR-0007

**Against ADR-0017, nothing, and the reason is positional rather than a judgement.** What that decision
forbids is `hyper` rendering a wire *it fetched* on a call where a credential was used — the hazard
being that `hyper` becomes the channel and a terminal or an Actions log becomes the store. Here
`hyper` fetched nothing. It resolved no credential, read no Target declaration, opened no socket, and
the bytes on the page are bytes the caller already holds in a file they wrote. Echoing them back is
not a channel; there is nowhere for the response to travel that it has not already been. ADR-0017's
sentence survives unedited: the wire is visible only where no credential was used, and no credential
was used because no request was made.

**Against ADR-0007, nothing, and less than nothing to arrange.** `hyper never stores a secret` is
untouched because no secret is resolved: `--response` binds no Target, so no `env:` slot is read, and
there is no credential in the process to store, suppress or leak. Nothing is written anywhere — no
Record, no Journal entry, no file.

**The cost that is real is the author's own file.** A response saved off a live API can carry a token —
`Set-Cookie` is the ordinary case, and a create's body is the other — and a path argument is read
against the repository ([ADR-0089](0089-a-path-argument-is-read-against-the-repository-never-against-the-callers-directory.md)),
so that file sits in a git working tree. `hyper` neither writes it nor loads it: the repository walk
reads `.yaml` and nothing else, so a sample enters no namespace, is checked by nothing, and is
rendered by no other command. What it is, is a scratch file an author may commit by accident, and the
answer to that is the same answer every scratch file gets — **it belongs in `.gitignore`**, and the
orientation says so. Refusing the feature over it would be refusing to read a file the author already
made, on the same disk, to protect them from a copy of what their own client already wrote there.

## The object is §12's, and the path is the repository's

Two details that could each have gone the other way.

**The file is the response object, not the body.** A projection reads `$.status`, `$.headers[…]` and
`$.tls.days_left` as readily as `$.body.…`, so a file holding only a body would answer three of those
with an absence the real call would never produce. So it carries the members §12 closes — read against
that closed set, in that order, with an unknown member refused by name and the members enumerated, and
`host` (or `command`) required because it is the member that survives where nothing came back at all
([ADR-0050](0050-a-status-is-an-answer-not-an-error.md)). Header names are lowered exactly as they are
off the wire, for the reason they are lowered there: a header name is case-insensitive on the wire and
a path is exact. The friction is real and it is the teaching: an author who supplied `data` at the top
level has written a path root no Capability has, and being told so here is the cheapest that fact ever
gets.

**The path is read against the repository**, like `check`'s positionals and `review`'s, and refused
where it resolves outside. ADR-0089's argument carries whole: it is the only root an agent can name,
the MCP tool builds a command line and holds no directory to re-root against, and a root that only one
of the two surfaces can compute is not ergonomics. The alternative considered was the JSON inline on
the command line, which needs no root at all — rejected because a response body on an argument line is
a line nobody can read in a transcript, and because a response arrives in a file, `curl > file` being
how it got there.

## Considered options

- **Widen `probe` to bind a Target or a Definition.** Issue #230's first shape. Rejected on the
  argument above: it spends ADR-0017 and still cannot see a `mutate`'s response.
- **A surface that performs one call and renders the response without writing Records.** Issue #230's
  second. It is the first wearing another name — the credential is still resolved and the wire `hyper`
  fetched is still rendered — and it owes a name, a place in §9's tree of sixteen, and an answer for
  what authority it runs under, none of which buys anything the first did not.
- **Document the loop we have.** Issue #230's third and cheapest. Rejected above: it blesses writing a
  whole response body into an append-only Store that travels in git.
- **A response schema declared in the Manifest**, so `check` could reach the projection offline.
  Refused where the projection is stated and already recorded as a limit in §13: it is an output
  schema, and an artefact stating what an API returns is a claim no reviewer can check and every
  upstream can invalidate.
- **Make the sample an artefact**, loaded and checked. Rejected twice: a reviewed artefact is a claim
  somebody stands behind, and a copy of what an API answered on Tuesday is not one; and §4's *no
  credential is resolved, no network reached* would come to depend on a file that is a photograph of
  the network.
- **Read the response from stdin.** Rejected on the surfaces: a tool builds an argv and hands it to
  the same dispatch, so a stream the CLI can be handed and the MCP surface cannot is the two surfaces
  differing in something other than ergonomics.

## Consequences

- **The tightest loop on the map now reaches the credentialled half and the effectful half.**
  ADR-0017's closing sentence — *this does not close the standing question of whether a Manifest
  correctly describes its API* — still stands, and what has moved is that the evidence is now
  obtainable without a Run for every Operation rather than for the public `read` ones alone.
- **A Manifest that reads a supplied response correctly is right about that response and not about the
  API.** The file is a fixture, and a fixture an author wrote by hand can be wrong in exactly the way
  the Manifest is wrong. This buys the same class of evidence a test fixture buys and no more, which
  is worth stating because the previous paragraph is the sentence people will remember.
- **`probe` has two forms and the tree still holds sixteen commands.** It is one flag and one page,
  not a seventeenth name (§9, [ADR-0094](0094-the-argument-less-invocation-writes-the-tree-and-there-is-no-help.md)).
- **A Probe on a `series` Operation was wrong, and is not any more.** The projection was read from the
  response root under both cardinalities, where a `series` Operation's `identity:` and `fields:` root
  at a **member** of the collection `over:` named (§3, §12). No case drove it because no case probed a
  `series` Operation; the supplied form's first real use is a collection, which is how it surfaced.
- **The `probe_result` row grew three members**, and one of them is a correction rather than an
  addition. `projection` is now a list under either cardinality, one entry per Record the response
  would have produced. `unresolved` names every authored path that failed to resolve against at
  least one of the roots it was read from, once however many it failed against — the half a Run has
  nowhere to put, a field going quiet being an absence on a version and an invisibility to an author.
  Failing on one member of a collection and not another is named, that member's version being the one
  written without the field. `supplied` says which of the two claims the response beneath is, because *this is what the
  world answered* and *this is what you handed me* may not render identically.
- **The MCP tool takes one more argument**, and it widens no reach: it names a file rather than a
  host, and the call it would have made is the call it does not make.
- **A response sample belongs in `.gitignore`.** It is the one thing this decision asks of an author
  that nothing enforces, and the orientation states it beside the loop it is part of.
