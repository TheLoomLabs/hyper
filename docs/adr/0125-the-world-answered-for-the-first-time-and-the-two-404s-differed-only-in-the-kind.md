# The world answered for the first time, and the two `404`s differed only in the Kind

**Nine `hyper` Runs reached a vendor nobody here controls — `api.hetzner.cloud` and `hetzner.com`,
from the published `v0.0.1-alpha` `x86_64-linux` archive, against an account a human owns with a
credential a human supplied.** Five headless authoring sessions wrote the artefacts, none of them
holding the credential and none of them reading `docs/spec/`. Four non-`2xx` answers came back, one
more than was asked for, and every branch of
[ADR-0050](0050-a-status-is-an-answer-not-an-error.md) they land on held: a `read` halted on its
projection and not on its status, a `destroy` completed on `404` and wrote an ordinary Tombstone
indistinguishable from the one written beside it in the same Step for a key that was still there, and
a `mutate` answered `404` halted where the `destroy` completed. Issue #249.

**Nothing about the product changes, and this does not re-litigate
[ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md).** That
decision is right and its reasoning stands; the harness should keep talking to the lookout. What this
run answers is the question ADR-0105 was not asked: `internal/capability/http.go` had never been
answered by a server nobody here wrote, and now it has been. Two defect candidates were observed and
neither is repaired here.

## The session, and what it was not

**It ran outside the sealed harness**, against an endpoint no task file has, on the deliberate ground
that the seal is where a *fixture* is reached and this run's whole subject is a vendor. What that
costs is stated under *A consequence for anything this session spawns* below.

- **The operator session** was attended — a human watching, holding the Hetzner token. It ran every
  `hyper` command that reached the world and wrote the Repository declaration, both Target
  declarations and `keys/`.
- **Five authoring sessions**, separate `claude -p` invocations, headless, cwd a throwaway git
  repository outside any `hyper` checkout. Each was prompted with the operator's brief plus one line
  confining it to that directory. Round 4 additionally carried the verbatim stdout of the `hyper run`
  that had just failed, and nothing else.
- **No authoring session ever held the credential.** The launcher does `env -u` on both token
  variables unconditionally, whether or not the operator session held one. Every round's own report
  independently records `HETZNER_API_TOKEN` as `(absent)` off `hyper targets`.
- **Authoring ran under an explicit `--allowedTools` allowlist rather than
  `--permission-mode bypassPermissions`** — the harness the operator session ran under refused to
  spawn a `bypassPermissions` child. The allowlist is strictly narrower than bypass, so ADR-0105's
  objection — *a live credential in a headless session running under `--permission-mode
  bypassPermissions`, asked to perform an effect nobody reviewed* — was answered twice over: no
  credential, and no bypass.

The binary was the published `v0.0.1-alpha` `x86_64-linux` archive, `sha256`-checked against the
release's own `checksums.txt` and verified `OK`. `hyper version` reported `0.0.1-alpha`, commit
`85244dd`, `go1.25.0`, `linux/amd64`. `scripts/release.sh` builds that archive `CGO_ENABLED=0` and
`-trimpath`, which is the axis the suite cannot reach.

**Two things went wrong before an agent was started, and both are about a private repository rather
than about `hyper`.** The README's install block could not be followed as written: an unauthenticated
`curl` of the release asset answers `404`, so `gh release download` was used and the checksum verified
by hand. And `hyper project` Refused on first use with `release-artefact-absent`, exit `77`, naming
two remedies — *publish a release for 0.0.1-alpha, or install a released hyper* — neither of which
applies to a release that exists and is private. `hyper.yaml` was written by hand carrying the two
values `project` derives, after which `project` exited `0`.

**`hyper project` wrote `AGENTS.md` into the repository, and the operator committed it, so every
authoring session had the orientation in front of it.** No `docs/spec/`, no ADRs and no `CONTEXT.md`
were in the throwaway repository. A `hyper` checkout existed elsewhere on the machine and each session
was told to work only in the throwaway directory — which is an instruction where
[ADR-0106](0106-a-manifest-is-writable-from-the-surface-and-both-costs-were-paid-at-the-world.md)'s
session had *no `docs/spec/` on the machine to read*. That is the weaker of the two and it is stated
here rather than glossed.

## Claim 1 — what the agents did, off the transcript

| round | brief | turns | cost | duration | what it changed |
|---|---|---|---|---|---|
| 1 | v1 | 78 | $5.88 | 736 s | both Manifests, three Definitions, three Procedures, from nothing |
| 2 | v2 | 50 | $3.90 | 554 s | split the mark into its own `mutate`; `per_page` 50 → 5; two Cadences |
| 3 | v2 | 66 | $4.85 | 732 s | generalised the mark to one Step over the whole ledger |
| 4 | v2 + the failure's stdout | 50 | $2.90 | 536 s | diagnosed that generalisation and reverted it; `bound` 5 → 2 |
| 5 | v2 | 65 | $4.64 | 713 s | `keys/mallory`, copied not repaired; `bound` 2 → 1 |

309 turns, $22.17, 54 minutes 31 seconds of wall clock across the five.

### The order of the calls

Every round opened the same way and in the same order: the tree and the git log, then `AGENTS.md`,
then every artefact file, then `hyper --help`, `hyper providers`, `hyper targets`, `hyper check`.
Rounds 2, 3 and 5 read `hyper provider hetzner` and `hyper operation hetzner list_locations` before
editing anything. That is ADR-0099's finding and ADR-0106's second sighting of it — *the Manifest's
own lines, verbatim: that is the format you author in* — holding a third time, outside the seal, on an
orientation this repository wrote and an API it did not.

Round 4 is the exception worth naming: it read fourteen files individually before running a single
`hyper` command, then went straight to `hyper runs`, `hyper show` on the failed Run, `hyper records`
and `hyper changes`. It had been handed a failure and it went to the record first.

### Round 1 and round 2 reached different documentation, and the difference is not cosmetic

**Round 1 never reached Hetzner's own machine-readable specification.** Its `WebFetch` of
`https://docs.hetzner.cloud/reference/cloud#locations-get-all-locations` returned navigation chrome
only — the docs site is client-rendered, so the fetched HTML carries no endpoint content. It then
spent **sixteen consecutive `Bash` calls** hunting the OpenAPI document: `spec.json`, `cloud.json`,
`api.hetzner.cloud/v1/spec.json`, `openapi-spec/cloud.json` under nine different prefixes, a grep of
the rendered HTML for any `.json` reference, a `specPath` variable pulled out of a minified Next.js
bundle and then out of three downloaded chunk files, a request carrying an `RSC: 1` header,
`sitemap.xml` and `robots.txt`. Every one of them dead.

It then fell back to `https://api.apis.guru/v2/specs/hetzner.cloud/1.0.0/openapi.json` — a
**third-party mirror**. That document's `x-origin` names `https://docs.hetzner.cloud/spec.json`, which
the operator confirmed answers `404`. Hetzner's own documentation prose — Pagination, Authentication,
Errors, Rate Limiting — is embedded verbatim in the mirror's `info.description`, and that is where
round 1 read it from.

**Round 2 found the vendor's live specification on its second attempt**:
`https://docs.hetzner.cloud/cloud.spec.json`. Operator-verified: `200`, 3,453,181 bytes, `servers:
['https://api.hetzner.cloud/v1']`. Rounds 2 through 5 authored from it, and rounds 3 and 5 each
re-fetched and re-read its Pagination and Errors sections rather than trusting the artefacts in front
of them.

**The mirror is materially stale on exactly the axis this task turns on.** Its `GET /ssh_keys`
declares `sort`, `name`, `fingerprint`, `label_selector` and **no `page` or `per_page`**. The vendor's
live specification declares all six.

**And it cost the artefact nothing observable, which is the finding rather than a licence.** Round 1
wrote `query: {per_page: "5"}`'s ancestor — `per_page: "50"` — and a `cursor:` Pattern onto
`list_ssh_keys` anyway, because it took the pagination rule from the mirror's embedded *prose* rather
than from that Operation's parameter list. An agent reading an OpenAPI document as documentation
rather than as a schema routed around the staleness without noticing it was there. Nothing in the
transcript shows round 1 comparing the two, and nothing shows it knowing there was anything to
compare.

Round 4 fetched `docs.hetzner.cloud/reference/cloud` and `/changelog` as its **last two calls**, after
it had written and checked its file — a confirmation pass rather than a source.

### What they had in front of them, and what they did not

They had: `AGENTS.md`, the operator's brief, the operator's half of the artefacts, `keys/`, the
0.0.1-alpha binary, the whole public internet, and — from round 2 on — the Store of every Run that had
already happened. They did not have: `docs/spec/`, the ADRs, `CONTEXT.md`, a credential, or any
`hyper` documentation for `patterns:`.

**`AGENTS.md` says nothing about `patterns:`.** ADR-0106 recorded that the orientation never names
Patterns and recorded it costing that run nothing, `limit` being in the fixture's documentation where
an author would look. Here it cost a large fraction of round 1. The grammar was recovered by three
routes in succession:

- `strings` over the 0.0.1-alpha binary, repeatedly and with progressively narrower filters (calls
  11–13, 20–21, 53–54, 75);
- driving `hyper mcp` over JSON-RPC from a Python script to read the tool schemas back (calls 14–19);
- **using `check` as an oracle**: deliberately writing `into: {bogus: page}`, reading the Refusal, and
  learning from it that `cursor.into` wants exactly one of `query` or `header` and that a second form
  spelled `page:` exists (call 52).

Round 2 reached the same place by a fourth route — locating the literal
`"items": {"enum": ["pagination", "polling", "retry"]}` inside the binary and reading six kilobytes
either side of it (call 26).

### Every dead end

- **Round 1's seventeen-call specification hunt**, above.
- **Round 3's selector-grammar battery.** It copied the repository to a scratch directory and drove
  `check` over roughly ten fabricated Procedures to find out what `over:` admits — including six
  spellings of a `values:` item in a single call (`{file: …}`, `{path: …}`, `{ref: …}`, `{from: …}`,
  `{step: …}`, `{asset: …}`), then a matrix of read-over-assets, require-over-series and
  mutate-over-observations cases. It deleted the scratch copy afterwards (call 63).
- **Round 5's three attempts at `probe`'s input flag** — `--host hetzner.com`, then
  `--input '{"host": "hetzner.com"}'`, then the spelling that works, `--input host=hetzner.com`. Round
  1 and round 3 had each found the same spelling independently.
- **Round 5 read a tool-result file out of its own session's transcript directory** (call 25) to
  re-grep strings it had already collected rather than re-running `strings`. It read its own output
  and not an answer key, but the class of path is the one ADR-0106 and
  [ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md) closed *inside* the
  seal, and outside the seal nothing closes it.
- **Deliberate falsification edits, made and reverted, in four of the five rounds.** Round 1:
  `into: {bogus: page}`, `into: {query: page, body: x}`, `nonsense_op:` for an operator,
  `field: no_such_field`, and a swap of the pagination form. Round 2: repointing a `require:` at a
  series-cardinality Step. Round 3: the scratch battery. Round 5: `bound: 0` on the `destroy`. Every
  one was restored in the same call or the next.

### What the rounds actually decided

Round 1 put `labels: {managed_by: hyper}` in the create body and said so proudly — the label is fixed
in the Manifest, no Step can vary it. Brief v2 said that was the wrong shape: *putting one on is not
the same act as vouching for it*. Round 2 removed the labels from the create, added a second `mutate`
(`mark_ssh_key`, `PUT`) whose identity resolves to the same name the create wrote under, and put two
per-key `require:` entries between the marking and the `destroy`.

**Round 3 generalised the two per-key mark Steps into one Step over the whole ledger** —
`over: {assets: [{field: id, exists: true}]}` — with the argument that per-person Steps disappear when
`keys/` empties, leaving the `destroy` with no gate above it. That is the one authoring decision in
this session that reached the world and was wrong, and it is wrong for a reason no `check` can hold:
`assets:` is every Record the Definition has ever written, a history rather than a project.

**Round 4, holding only the failed Run's stdout, found it from the record.** It ran `hyper changes`,
read `vanished … ada`, named the cause in one sentence — *`assets:` is every Record the Definition has
ever written, a history, not the project* — and reverted to per-key scoping. It also narrowed the
Bound from 5 to 2 *because the ledger held two*, and said so. That is the loop this product exists to
make possible, performed once, by an agent that had not written the thing it diagnosed.

Round 5 was handed a `keys/` holding one file with a deliberately truncated public key. It counted the
base64 characters, decoded the blob, read the internal length prefix, ran `ssh-keygen -l`, **copied
the bytes in verbatim rather than repairing them**, and wrote eleven lines of comment above the Step
saying the Run would end there. It did.

## Claim 2 — four non-`2xx` answers, read line by line against ADR-0050

Three were asked for. The fourth was not, and it is the one that makes the third mean something.

| Run | Step | Kind | status | Disposition | outcome |
|---|---|---|---|---|---|
| `01a05d0e-ec35` | `locations` | `read` | `401` | ran | **failed** |
| `01a05d1f-1c3b` | `mark-ours` | `mutate` | `404` | ran | **failed** |
| `01a05d29-175b` | `retire` | `destroy` | `404` recorded, a second call unrecorded | ran | **completed** |
| `01a05d36-942b` | `publish-mallory` | `mutate` | `422` | ran | **failed** |

### `401` on a `read` — the Step halted on the projection, and the record does not say `401`

ADR-0050's central claim is that *the status was never what decided a `read`; the projection was*. The
Journal entry is that sentence and nothing else: `projection_failed_path: "$.body.locations"`,
`identities.members: []`, `disposition: "ran"`, and **no `answered` key at all**, because `answered` is
effectful-only. The Run's `outcome.json` reads `failed`. The Step ran twenty-four seconds after an
identical Run that completed with two pages, on the same procedure revision and the same manifest
digest.

**The consequence of that rule is legible here for the first time, because three separate agents ran
into it.** Rounds 3, 4 and 5 each read this entry cold and each reported, independently and in
different words, that they could not say what came back: *the Store does not keep response bodies, so
I cannot tell you what came back* (round 3); *I'm not going to guess at a cause* (round 4); *I'd
rather say that than guess* (round 5). ADR-0050 explains exactly why — a `read`'s status belongs in
the Record where its Manifest projected it, and this Manifest projected no status. The rule held; what
it costs is three agents unable to name a `401` from a Journal that recorded it as a missing key.

**One limit and one gap.** ADR-0050's *the Expansion drains and the Run halts after it* was **not**
exercised: `locations` declares no `over:`, so there was one call and nothing to drain.

And **the Journal does not record that this Run met a `401` at all** — it records a path that did not
resolve, and nothing about a status. That the status was `401` is established outside `hyper`, by the
operator afterwards: `GET /v1/locations?per_page=5` under a malformed token answers `HTTP 401` with
`{"error":{"code":"unauthorized","details":null,"message":"the token you have provided is invalid"}}`
and no `locations` key. That is the same call the Step made, and it is the reason the projection could
not resolve. **The rule is vindicated and the record is thin**: a reader holding only the Store cannot
recover why the path was missing, which is precisely what the three agents below discovered.

### `404` on the `destroy` — completed, and the two Tombstones are indistinguishable

The `retire` Step of `01a05d29-175b` expanded to two members. One had been deleted out of band by a
human in Hetzner's web console; the other was live. **The Run completed and both Records carry
`"tombstone": true`.**

**What the Store settles and what it does not.** Observed: the Expansion was two members, two
`DELETE` calls were made, one `answered` was written, its status is `404`, and both series received a
Tombstone. **Not observed:** which member the `404` belongs to, and what the other call answered. The
reading below — that the deleted key answered `404` and the live one `204` — rests on the key having
been confirmed gone by hand beforehand, on the other being present immediately before the Run, and on
the vendor's own specification documenting `DELETE /ssh_keys/{id}` as `204`. It is a sound inference
and it is not a record. **That the Store cannot settle it is the finding**, and it is the second
defect candidate below.

ADR-0050 calls this *the alternative is not a limit but a trap*, and the trap is visible in the
evidence around it: the out-of-band deletion left a standing Asset at ordinal 2 with no Tombstone
while the vendor answered `404` for it, and under a `< 400` rule that Asset could never have been
Tombstoned by anything.

The sharpest line in that ADR is *nothing distinguishes a Tombstone written on `404` from one written
on `204`*, and this is the first evidence for it: the two Record blobs differ in the name, the id and
the fingerprint, and **in nothing else** — no marker, no status, nothing saying that one of these two
things was already gone when `hyper` reached for it. No marker, no status, no hint of which was which. The
distinction is where ADR-0050 said it would be relocated to — the `answered` key on the Step file —
and that is where the second defect candidate lives, below.

### `404` on a `mutate` — halted, and the Kind is the whole of the difference

Not asked for by the ticket, and it belongs beside the `destroy`. The `mark-ours` Step of
`01a05d1f-1c3b` sent `PUT` to the same host, from the same Definition, at the same object, and
answered the same `404`. `answered: {"host": "api.hetzner.cloud", "status": 404}`, `disposition:
"ran"`, `identities.members: []`, outcome **failed**.

Two Steps, one status, one host, one Definition, opposite outcomes. ADR-0050 scopes the `404` rule to
`destroy` in one clause — *an effectful Operation completes on `2xx`, a `destroy` on `404` besides* —
and this pair is the proof that the scoping is load-bearing rather than decorative. A `mutate` told
there is nothing there has **not** reached the state it exists to reach.

It also halted before `retire`, which is what the brief asked for and what the ADR predicts: nothing
was taken off the project by a Run whose marking had not landed.

### `422` on the `mutate` — *ran*, no `error_code`, and the vendor's own spec said `400`

`publish-mallory` of `01a05d36-942b`: `answered: {"host": "api.hetzner.cloud", "status": 422}`,
`disposition: "ran"`, `identities.members: []`, no `error_code` anywhere in the entry, outcome
`failed`, nothing created. ADR-0050 in three clauses — *halts on everything else*, *a status halt
carries no `error_code`, and its Disposition is ran*, *a response arrived, which is what ran means*.

**And the status is not the one the agent predicted.** Round 5 wrote, in its report and in eleven
lines of comment above the Step, that Hetzner answers a malformed `public_key` with `400` and code
`invalid_input`. The vendor answered `422`, with code `invalid_input`. The evidence does not record
where the `400` came from — the spec, an inference, or the error table it read in round 3's call 54 —
so what is established is the discrepancy and not its origin. This is the class of fact a fixture we
write cannot produce: our fixture answers the status our documentation says it answers, because one
person wrote both.

## Claim 3 — what the lookout had been letting pass

The lookout is `scripts/acceptance/lookout/api.go` and `main.go`: one route prefix, a bearer check
before the route, and eight statuses it can ever produce — `200`, `201`, `204`, `400`, `401`, `404`,
`405`, `409`. It is a good fixture and ADR-0105's argument for it stands. What follows is what a
fixture-answered transcript could not have said, read against the world-answered one.

**A `destroy` had never been authored, run, or answered.** ADR-0106 states it: *the `destroy` half of
a Manifest is still unwritten by any agent — the Target admits `read` and `mutate` only, so nothing
here exercised a `destroy:` claim, a Bound, or a Tombstone*. `api.go` has a `remove` route and no task
reaches it. This session wrote all three — `destroy: [delete_ssh_key]` on the Definition, `bound:` on
every effectful Step, and two Tombstones — and closed the second half of
[ADR-0096](0096-the-shape-carried-whole-is-the-effectful-one-and-the-example-is-not-the-acceptance-task.md)'s
outstanding transcript, the delete and the Bound.

**TLS to a public CA chain.** The suite's dialer builds its own `tls.Config` carrying an explicit
`RootCAs` pool and a frozen `Time`, so no test has ever verified against the system root pool at real
wall-clock time. The lookout's route is a PEM named by `SSL_CERT_FILE`. Between them, the verifier
that a user's `CGO_ENABLED=0` `-trimpath` build actually runs had no coverage. This session's Runs
verified `api.hetzner.cloud` and `hetzner.com` from that archive and completed, and round 5's probe
read `days_left: 150` off a certificate expiring 2027-01-29. What that establishes is that the
published archive verifies real public chains; **nothing in the evidence records which verifier ran or
what the root store was**, and that follows from the build flags rather than from an observation here.

**A redirect nobody here served.** `client()`'s `CheckRedirect` returns `http.ErrUseLastResponse`, and
until now every `3xx` it declined was one this repository answered with. `hetzner.com` answered
`HTTP/2 301`, `location: https://www.hetzner.com/`, `server: HeRay`, and the Observation carries all
three as projected fields. The guardrail earned its keep on the way past: `targets/local.yaml` grants
`hetzner.com` and **not** `www.hetzner.com`, so a client that followed would have reached a host
outside the grant — which is exactly the reach-arriving-from-data that
[ADR-0029](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md) closed. The same
Provider declares no `auth:` at all and its Target carries no credential slot, so *a Provider naming
no scheme sends no credential* was exercised against a real host.

**A Pattern, at all.** ADR-0106 refused removing the fixture's `limit` *so that paging must be handled
by a `pagination` Pattern*, on the ground that nothing inside the seal teaches Patterns. The
consequence, unstated until now, is that **no acceptance transcript has ever declared a Pattern of any
kind**. This one declared `pagination` on both list Operations, and the Journal carries
`"pattern": {"pages": 2}` on the Step that walked six locations at `per_page: 5`. The lookout's cursor
is an opaque base64 token under `data.cursor`; Hetzner's is a page *number* under
`meta.pagination.next_page` written into a `query: page` — the `cursor:` form carrying an integer,
which the fixture's shape cannot ask for.

**A `record:` over a response shape nobody here designed.** The awkwardness ADR-0105 built into the
lookout — an envelope key, a collection under a name, an identity that is not the name, a create whose
answer differs from a list element — is four guesses at what a vendor does. Hetzner has the fourth and
not the first: `GET /v1/locations` answers a bare `{"locations": [...]}` with no envelope, while
`POST` and `PUT /v1/ssh_keys` answer `{"ssh_key": {…}}`, so `create_ssh_key` and `mark_ssh_key`
project `$.body.ssh_key.*` and `list_ssh_keys` projects `over: $.body.ssh_keys`. The guess was right
about the one that matters and wrong about the envelope, which is worth knowing about a fixture whose
shape is ours.

It also produced a case the fixture cannot: `managed_by: $.labels.managed_by` resolves to nothing on
an unlabelled key, and **that does not halt** — only `over:` and `identity:` paths do — so the field
is simply absent from the version. The lookout's fields are always present. That absence is what makes
*which keys are ours* answerable a year later, and it is the load-bearing behaviour of the whole
Procedure.

**An `auth:` scheme against a server that enforces it.** This is the axis the world changed least.
`header: {name: Authorization, prefix: "Bearer "}` is byte-identical to the scheme the lookout checks,
because Hetzner and the fixture agree. What the world added is the shape of a real refusal, which is
the `401` above.

### Two defect candidates, observed and not repaired

Filed as #251 and #252.

**1. `check` accepts a `require:` rooted at a `series`-cardinality Step (#251).** Round 2 found it
deliberately — repointing a Requirement at `list_ssh_keys` passes `checked 12 artefacts: no problems
found` — and wrote it up as *a deliberate choice, not something the tooling enforced*. Round 3, with
no memory of round 2, authored `require: {step: mark-ours, …}` where `mark-ours` expands over two
Assets, and `check` passed that clean too. The operator reproduced both. **Run-time behaviour was
never exercised**, so what is established is that the offline oracle does not decline it and nothing
more.

This is not reachable from the seal. [ADR-0122](0122-a-requirement-roots-at-any-projected-field-and-the-value-goes-on-the-line.md)'s
Requirement evidence came from `change-window`, whose reads are `shell` Steps of `one` cardinality;
`monitor-coverage` is the only task with a `series` read and its task text asks for no `require:` at
all.

**2. `answered` is singular and unattributed on an expanding effectful Step (#252).** The `retire` Step above
expanded to two members and made two calls, of which one answered `404`. Its Journal entry carries one
`"answered": {"host": "api.hetzner.cloud", "status": 404}` with **no member named**. §8 describes
`answered` in the singular — *the host it reached and the status it got* — and is silent on the
expanding case. The rule as written was followed: the non-`2xx` answer is the one recorded. What is
missing is which member it was about, and that is precisely the fact ADR-0050 relocated onto this key
when it decided that Tombstones would not carry it.

Also not reachable from the seal: `monitor-coverage`'s effectful Step does not expand over Assets.

### Two softer observations, offered as facts

`hyper project`'s `release-artefact-absent` message names two remedies and neither applies to a
release that exists and is private (#254). And the `TOMBSTONE` column of `records --history` renders `yes` on
every row of a series whose head is a Tombstone, including the create and the mark — which
`internal/cli/records.go` documents as deliberate, *the Record's state, and both are the series'
rather than the version's*, and which the operator misread anyway on first reading.

### One thing the record does not carry, by design

Two Requirements passed in the effectful Run, and **the Journal is silent about both**. A Requirement
writes no Journal file and takes no Step number ([ADR-0116](0116-a-requirement-halts-and-claims-nothing-to-do-it.md)),
which is why a nine-entry Procedure left seven Step files numbered 1 to 7. So *the Requirement passed*
is not a claim the Store carries; what the Store carries is that `retire` ran, from which it follows.
That is the decision working as decided, and it is written down here because a reader verifying the
acceptance criterion against the Store will look for it and not find it.

## Claim 4 — the axes that remain untested, named

ADR-0105's *one honest limit* is the model: written down so the next reader does not discover it as a
surprise.

**`429` is declined, and here is the reason.** Hetzner's documented limit is 3600 requests per hour
per project. Provoking it means thousands of requests against a vendor's API to serve a fixture —
slow, rude, and it spends somebody else's capacity to produce a status an injected `capability.Dial`
produces for nothing. What stays untested as a result is narrow and worth naming: ADR-0050's *every
call is judged, a Pattern's included* against a `429` arriving mid-walk, and
[ADR-0018](0018-retry-only-follows-a-failure-that-provably-preceded-the-request.md)'s confinement
of retry against a vendor that would rather be retried later.

**Following a redirect** — declined here, never followed anywhere, and never will be by design.

**`basic:` as a scheme.** ADR-0031's set has two members and only `header:` has ever been authored, in
the seal or out of it.

**`polling` and `retry` as Patterns.** `pagination` is now one of the three; the other two remain
declared by no transcript.

**The `page:` form of `pagination:`.** Both Operations used `cursor:`, and `internal/run/pattern.go`
is explicit that a `cursor:` walk terminates on the empty collection *and* on a cursor that stopped
resolving, while a `page:` walk terminates on the empty collection alone and therefore always makes
one request past the last page. Every round said the same thing about this, unprompted and in its own
words — round 2's is the cleanest: *a correct pager never asks for a page past the end, so nothing in
this design will ever produce that answer.* **So `hyper` never asked.** The operator answered it with
`curl` instead, and the answers are recorded because they are what a `page:` walk would meet: page 3
of 2 answers `200` with an empty collection and `next_page: null`, and page 99 answers `200` with an
empty collection and **`previous_page: 98`** — a vendor computing rather than reporting. A `page:`
walk over this API would therefore terminate correctly and cheaply. That `hyper` does so is untested.

**A single-page Pattern walk is indistinguishable in the Journal from no Pattern at all.** Checked
both ways here: a successful single-page `read` and the failed one each wrote no `pattern` block, and
the two-page walk wrote `"pattern": {"pages": 2}`. §8 states the rule — the account is written where a
Pattern did more than the trivial single call — so this is the spec being followed. What is untested
is whether the same silence obtains on a `polling` or `retry` Pattern, and what it would cost a
Comparison to be unable to tell *declared and made one call* from *not declared*.

**A `read` against a host that answered nothing at all.** ADR-0050's response-object-is-`host`-and-
nothing-else case, and the `attempted-world-untouched` Disposition beside it. Nothing here provoked a
refused connection, a DNS failure or a deadline.

**A non-JSON body.** ADR-0040's *`body` is absent rather than an error*. The `301` from `hetzner.com`
is the obvious candidate and the Manifest projects only `$.host`, `$.status` and two headers, so
nothing read a body there.

**Chunked transfer and `Content-Encoding: gzip`.** `client()` sets `DialTLSContext` and
`DisableKeepAlives` and nothing else, so Go's transport advertises `Accept-Encoding: gzip` and
decompresses transparently. **Whether either happened is not recorded anywhere in the evidence** — the
Store keeps no request or response headers for these calls — so this session is not evidence either
way.

**`skip-if-recorded` against a vendor.** `create_ssh_key` declares it and round 4 warned about its
interaction with a stale ledger. No Run ever re-ran a create over a standing Asset, so the
`skipped-as-already-recorded` Disposition was never reached.

**A Cadence firing.** `hyper project` generated two workflows and every Journal entry in the Store
carries `cause: "manual"` and `executor: "local"`. Nothing scheduled ever ran.

**Three of the four published archives.** `aarch64-linux`, `x86_64-darwin` and `aarch64-darwin` are
untouched here; that is #247's subject and there is no dependency in either direction.

**`hyper install` over a real network** is #248 and not this. The one place this session brushed
`internal/release` — `hyper project`'s digest fetch — Refused, and for a private-repository reason.

**Vendor behaviour not provoked**: a `5xx`, a `403`, a `Link`-header pager (§13: an API paginated that
way can be called and cannot be paged), a schema change mid-walk, a redirect on an effectful call, and
anything at all that Hetzner does when nobody is provoking it. **Every non-`2xx` in this session was
arranged by a human**, and none was met.

## What this run does not establish

- **One session.** Five authoring rounds, one vendor, one API, one credential, one machine, one day.
- **The absence of the specification was an instruction rather than a namespace.** ADR-0106's session
  had no `docs/spec/` on the machine; this one had a checkout elsewhere and was told not to read it.
  Nothing in the call order suggests any round went looking, and nothing prevented it.
- **The `hetzner.com` half is thin by construction.** One `read`, one host, one status, no credential
  — which is what makes it safe to run and what makes it evidence about the redirect guardrail and
  little else.
- **No repair.** Both defect candidates are named as observed and are filed separately, as #251 and
  #252, with #254 for the softer one. A repair landing beside this record would make it a report on a
  surface that no longer exists, which is #241 → ADR-0120's worked shape.

## A consequence for anything this session spawns

**A taught repair out of this session owes a run to no existing task.**
[`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md) requires the ticket that lands
a taught repair to name the run it owes — *the one whose transcript produced the repair*, or where the
repair came from elsewhere, *the task whose surface it touches*. This session ran deliberately outside
the harness, against an endpoint no task file has, so neither clause reaches. The doc anticipates
exactly this and states the answer: *where none does, say so — that is a gap in the task set.*

Named to the line, because the nearest candidate is not the right one. `monitor-coverage` is the only
Provider-authoring task, and none of the three surfaces this session's observations touch is reachable
from it: its Target admits `read` and `mutate` only, so no `destroy` and no Tombstone; its task text
asks for no `require:`, so a series-rooted Requirement cannot be written against it; and its effectful
Step does not expand over Assets, so `answered` cannot be made ambiguous there.

So a follow-up ticket must say so in writing — #253 — and **whether to add a task file is that
ticket's decision rather than this one's** — adding the `.md` and the executable `.setup.sh` beside it is the
whole of what fencing one takes (#222), and it is a decision about what the harness is for rather than
a consequence of this run.

## The two Manifests, whole

Neither is committed to this repository, and that is deliberate: a Manifest for an API we do not
control, committed here, is an Extension whose staleness we would then own — and it *is* an Extension,
authored here and distributed to nobody, which is the gloss issue #249 corrected in `CONTEXT.md`
along the way. This ADR is where they live. The throwaway repository stays on the machine, outside the
checkout, unpublished, and its path is not cited here, matching what `CONTRIBUTING.md` already says of
transcripts.

### `providers/hetzner.yaml`

```yaml
kind: provider
provider: hetzner
schema-version: 1
class: hetzner
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_locations:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /v1/locations
      query: {per_page: "5"}
    patterns:
      pagination:
        cursor: {from: $.body.meta.pagination.next_page, into: {query: page}}
    input:
      type: object
      properties: {}
    record:
      over: $.body.locations
      identity: $.name
      fields:
        name: $.name
        description: $.description
        network_zone: $.network_zone
        country: $.country
        city: $.city
  list_ssh_keys:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /v1/ssh_keys
      query: {per_page: "5"}
    patterns:
      pagination:
        cursor: {from: $.body.meta.pagination.next_page, into: {query: page}}
    input:
      type: object
      properties: {}
    record:
      over: $.body.ssh_keys
      identity: $.name
      fields:
        id: $.id
        name: $.name
        fingerprint: $.fingerprint
        managed_by: $.labels.managed_by
        created: $.created
  create_ssh_key:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /v1/ssh_keys
      body:
        name: "{name}"
        public_key: "{public_key}"
    input:
      type: object
      properties:
        name: {type: string}
        public_key: {type: string}
    record:
      identity: "{name}"
      fields:
        id: $.body.ssh_key.id
        name: $.body.ssh_key.name
        fingerprint: $.body.ssh_key.fingerprint
        created: $.body.ssh_key.created
  mark_ssh_key:
    kind: mutate
    repeatability: repeatable
    deadline: 30s
    http:
      method: PUT
      host: "{from-target}"
      path: /v1/ssh_keys/{ssh_key_id}
      body:
        labels: {managed_by: hyper}
    input:
      type: object
      properties:
        ssh_key_id: {type: integer}
    record:
      identity: $.body.ssh_key.name
      fields:
        id: $.body.ssh_key.id
        name: $.body.ssh_key.name
        fingerprint: $.body.ssh_key.fingerprint
        managed_by: $.body.ssh_key.labels.managed_by
        created: $.body.ssh_key.created
  delete_ssh_key:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /v1/ssh_keys/{ssh_key_id}
    input:
      type: object
      properties:
        ssh_key_id: {type: integer}
```

Read against §3 and the ADRs: `class:` matches the Target's, `capabilities:` equals what the
Operations derive, `identity: "{name}"` on the `skip-if-recorded` `mutate` is the hole rather than a
response path (ADR-0056), `ssh_key_id` is declared `integer` so the hole reaches the wire as a JSON
number (ADR-0078), the `destroy` projects nothing at all (ADR-0037), and every path carries the
`body.` segment ADR-0040 requires. `check` called it clean on every round.

### `providers/site-reachability.yaml`

```yaml
kind: provider
provider: site-reachability
schema-version: 1
class: local
capabilities: [http]
operations:
  fetch_root:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http: {method: GET, host: "{from-target}", path: /, host-input: host}
    input: {type: object, properties: {host: {type: string}}}
    record:
      identity: $.host
      fields:
        host: $.host
        status: $.status
        server: $.headers.server
        location: $.headers.location
```

Four of its five projected fields are outside the body, which is the whole of ADR-0040's argument
standing up in an artefact an agent wrote: `$.host` is the fact about the call rather than the answer,
and it is what the Record is identified by. It declares no `auth:`, its Definition binds only a Target
carrying no credential slot, and the `301` it recorded is the first redirect `hyper` has declined that
this repository did not serve.
