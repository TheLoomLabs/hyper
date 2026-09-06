# The sink was supplied, both secrets were lost, and `check` was clean throughout

**Both taught repairs landed, and this is the first evidence either has.** A sealed session reached
`secret-sink-absent`, read §8's remedy, took it verbatim on the next call, and got two live
credentials out of `<nnnn>/<name>/<field>` — naming which file belonged to which service, the modes,
the byte count and the absent trailing newline, without being asked for any of it.
[ADR-0146](0146-a-sink-nothing-writes-is-a-refusal-and-not-a-completed-run.md)'s clause and
[ADR-0148](0148-a-secret-sink-is-a-directory-hyper-makes-and-one-file-holds-one-value.md)'s did the
job they were written for. Issue #272, and the run both records deferred.

**And forty calls earlier the same session lost two secrets with a sink named on the invocation.**
It wrote the declaration inside `fields:` — `token: {path: $…, secret: true}` — rather than as the
Operation-level `secret: [token]`. `check` passed it over eleven artefacts. `probe` printed the
projection without the field. `review` rendered the Operation without a secret. The Run **completed**
at exit `0`, two `mutate` Steps `ran`, two Records written, `--secret-out` pointing at a path
`hyper` never created — and the one response carrying each token was discarded. That is the loss
ADR-0146 named, *a green `check` standing behind a Run that quietly destroyed what it was invoked
for*, arriving through a door neither ADR fenced.

**The feared failure did not happen and its neighbour did.** Issue #272's first question asks whether
an author who fails to declare a credential writes a live value into the Record and onto the branch.
Nothing of the sort occurred: the session grepped both tokens against the whole `hyper-store` branch
and the tracked tree and neither appears, because a field the projection drops reaches the Record no
more than a field it suppresses. The undeclared secret was not leaked. **It was destroyed**, and the
two outcomes are told apart by nothing `hyper` prints.

## The evidence: eighty-one calls, six Runs, exit `0`

A Claude Code session, headless, 2026-09-06, inside the seal
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md),
[ADR-0130](0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md)), against
`push-credential` (issue #271) and the local TLS endpoint
([ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)).
Eighty-one tool calls, eighty-two turns, ten minutes thirty-five, **$4.28**, exit `0`,
`subtype: success`.

Forty-two `Bash`, seven `check`, six `review`, six `run`, four `changes`, three `operation`, three
`probe`, three `Write`, two `ToolSearch`, and one each of `providers`, `targets`, `provider`,
`records` and `run_show`.

**It is the most expensive sealed run recorded here, and the one with the most calls** — against
`release-promotion`'s 59 calls and $3.65
([ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)),
`monitor-retirement`'s 53 and $2.65
([ADR-0129](0129-the-destroy-landed-inside-the-seal-and-what-held-the-hand-made-monitors-was-not-the-bound.md)),
`monitor-coverage`'s 52 and about two dollars
([ADR-0106](0106-a-manifest-is-writable-from-the-surface-and-both-costs-were-paid-at-the-world.md)),
the empty-credential run's 32 and $1.43
([ADR-0147](0147-the-empty-credential-was-read-as-a-state-and-the-agent-went-looking-for-what-emptied-it.md)),
and `change-window`'s 20 and $0.73
([ADR-0120](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md)).
**It is not the longest and not the dearest session in the corpus**, and saying otherwise would
contradict a record that stands: `release-promotion` ran thirteen minutes thirty-five, and three of
[ADR-0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md)'s
five rounds cost more than this one at $5.88, $4.85 and $4.64. Those five ran **outside** the harness
against a vendor, which is why they are not what the sealed figures are ranked against.

Thirteen of the forty-two `Bash` calls are one investigation, and it is the one the loss caused.

**Six Runs — four completed, two Refused.** `survey-lookout` twice (calls 23 and 74, both `read`,
two Records each); `issue-service-credentials` four times: completed-and-lost (41),
`run-once-recorded` (49), `secret-sink-absent` (69), completed-and-written (70).

**The task's own prediction held.** `push-credential.setup.sh` records that the task takes two Runs,
an effectful Step's Expansion ranging over Assets and never over Observations while the mint route
takes a `ref` that lives on the lookout. The session did exactly that: `survey-lookout` at call 23,
`records` and `run_show` at 24–25 to read the refs back off its own Records, the Expansion authored
as a literal list at call 35. It cost four calls and no Refusal. **That is the one thing the task's
author expected to stand between a session and the Refusal, and it did not stand.**

## Both repairs landed

**ADR-0146's clause.** Call 69, the Run with the declaration finally correct and no sink. Both
members, verbatim but for the second citation block's elided gutter:

```
nothing ran. no step was reached.

refused: secret-sink-absent

  procedures/issue-service-credentials.yaml:5
    4 │ steps:
    5 │   - id: ledger-reissue
      │         ^ lookout issue_credential declares secret: output and this Run was given no Secret sink to write it to
      │
      = checked at run start, before the first step
      = the same command again with --secret-out <path>, naming a directory outside the repository that is not there yet

refused: secret-sink-absent

  procedures/issue-service-credentials.yaml:11
   11 │   - id: dispatch-reissue
      │         ^ lookout issue_credential declares secret: output and this Run was given no Secret sink to write it to
      │
      = checked at run start, before the first step
      = the same command again with --secret-out <path>, naming a directory outside the repository that is not there yet

refused · exit 77 · run 01a075e6-d17e-7d9a-9a03-b49c5bea4c15
```

**Two members, one per Step** — §9's two-member rendering, which issue #269 fixed the specification
to match, is what the session actually met.

**ADR-0148's remedy.** Call 70 is call 69 with `secret_sink` added and nothing else changed — **the
fourth remediation class taken verbatim, on the next call, with no intervening exploration.** *A
different invocation* has had one member since ADR-0148 returned it to §8, and this is the first time
an agent has been observed taking it. It completed: two `mutate` Steps `ran`, two Records, exit `0`.

**ADR-0148's directory shape.** Call 71 read the tree back with `find -printf '%M %u:%g %6s %p'`:

```
drwx------ idabic:idabic     80  …/lookout-push-credentials
drwx------ idabic:idabic     60  …/lookout-push-credentials/0001
drwx------ idabic:idabic     60  …/lookout-push-credentials/0001/ledger
-rw------- idabic:idabic     26  …/lookout-push-credentials/0001/ledger/token
drwx------ idabic:idabic     60  …/lookout-push-credentials/0002
drwx------ idabic:idabic     60  …/lookout-push-credentials/0002/dispatch
-rw------- idabic:idabic     26  …/lookout-push-credentials/0002/dispatch/token
```

`0700` on both directory kinds and `0600` on the files — what `internal/run/sink_test.go` holds and
`sink.golden` renders, met in the world for the first time. The session **printed no token value into
its own transcript**: call 71 answers the byte count and a `^[A-Z0-9]+$` match, call 81 the absent
trailing newline. Nothing asked it to be careful.

**The shape answered the question it was chosen for.** The task asks *which of them belongs to which
service*, and the answer came off the paths — `0001/ledger/token`, `0002/dispatch/token`, ordinal
from the Step and name from the directory beneath — with the session's own sentence under it: *the
whole file is the token, so `cat` it and paste*. That is the axis ADR-0148 said one keyed document
could not carry, carried once.

**The third question was answered off `changes`.** Call 73, the reissue against the lost Run, one of
two Asset rows:

```json
{"type":"asset","change":"changed","target":"lookout","definition":"lookout-credentials",
 "name":"ledger","from_ordinal":1,"to_ordinal":2,
 "fields":{"id":["cred_5u3yprgq","cred_7zvkpy34"],
           "issued":["2026-09-06T08:43:08Z","2026-09-06T08:47:40Z"],
           "token":[null,"<secret>"]}}
```

`token: [null, "<secret>"]` is the loss and the recovery in one row, and the session handed it back
as *the audit trail of exactly this*.

## The sink was supplied before it was needed, and the Run completed without one

Call 27 appended the mint Operation, verbatim:

```yaml
  issue_credential:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /v1/monitors/{ref}/credential
    input:
      type: object
      properties:
        ref: {type: string}
        service: {type: string}
    record:
      identity: "{service}"
      fields:
        id: $.body.data.credential.id
        monitor: $.body.data.credential.monitor
        service: $.body.data.credential.service
        issued: $.body.data.credential.issued
        token: {path: $.body.data.credential.token, secret: true}
```

**Every surface passed the last line.** `check` at call 28 reported one problem — `input "service" is
declared and no position of this Operation's request reaches it` — and nothing about the token; call
31 dropped the `service` input and the `skip-if-recorded`, and call 32 was clean. `review` at calls
37–38 rendered Procedure and Manifest. Nothing named a secret because, as far as `hyper` was
concerned, nothing had declared one.

Call 41 is the Run, and **the session named a sink without having been refused into one** — it had
read the `run` tool's `secret_sink` description and supplied a path in advance, which is the
behaviour ADR-0148 set out to teach:

```json
{"procedure": "issue-service-credentials",
 "secret_sink": "/home/idabic/acceptance-272/lookout-push-credentials"}
```

It answered `"outcome":"completed"`, two Steps `"disposition":"ran"`, `"records":1` each. Call 42
went looking for the sink:

```
bfs: error: /home/idabic/acceptance-272/lookout-push-credentials: No such file or directory.
```

The Records hold `id`, `issued`, `monitor` and `service`, and **no `token` key at all** — not
`<secret>`, absent. Two credentials had been minted on the lookout and both tokens were gone.

**This is the state ADR-0146 exists to make unreachable, reached with the flag ADR-0148 added held
correctly in the caller's hand.**

## Why `check` said nothing

`internal/artefact/manifest.go:196` — `fields:` is an **Open** object, so a field whose value is a
mapping is a shape the schema does not describe and does not refuse:

```go
{Name: "fields", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
```

`internal/projection/projection.go:143` — the reader takes scalars and skips everything else:

```go
if key.Kind != yaml.ScalarNode || value.Kind != yaml.ScalarNode {
    continue
}
```

The reader is **right by its own rule**, and the comment above it says so: *it judges nothing and
drops what it cannot read, which is every reader's rule in this tool: what is wrong with a Manifest
is `check`'s to report and never a reader's to guess at*
([ADR-0064](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md)). The
division of labour is the right one. **The hole is that the other half declines to look**: `check`
never reports it, Open being the schema saying it has no opinion here.

**The session's own experiment is the sharpest statement of it.** Calls 61–62 drove six wrong
spellings through `check` and `probe`, and the results split cleanly by *position*:

| | candidate | `check` |
|---|---|---|
| B | `fields: {token: {secret: $…}}` | **clean**, token dropped |
| — | `fields: {token: {path: $…, secret: true}}` (the one it had shipped) | **clean**, token dropped |
| C | `record.secret: [token]` | `unknown-key` at `operations.issue.record.secret` |
| D | `record.secret: {token: $…}` | `unknown-key` at `operations.issue.record.secret` |
| F | `secret: {token: true}` | `schema-mismatch`, *expected an array at this position* |
| G | `secret: token` | `schema-mismatch`, *expected an array at this position* |

Only the correct `secret: [token]` (candidate E) checked clean *and* withheld the response. **The two
that pass silently are the two written inside `fields:`, and the four that are caught are the four
written anywhere else.** `check` is not weak here in general — it is blind at exactly one position,
and that position is where an author reaching for *this field is secret* naturally puts it.

It reproduces with no session and no service. Against the fixture repository this run left, using
the binary the harness built from `0db9fbe`, with one ordinary Operation carrying that one field:

```
$ hyper check providers/repro.yaml
checked 12 artefacts: no problems found

$ hyper probe repro issue --response .scratch/r.json
FIELD      VALUE
identity:  ledger
id         cred_x
```

Twelve is the fixture's eleven artefacts plus the added one. The token is in the supplied response
and is not in the projection, and neither surface says a word.

**The converse gate does not exist, and its absence is deliberate.** `internal/run/gates.go:497`
opens `sinkRefusals` with `if sink.named() { return nil }`, and `gates.go:163` returns from the whole
sink block when no Step declares secret output — so a named sink is neither refused nor created.
`internal/run/sink.go` states the intent plainly: *a Run that reaches no Step declaring secret output
never calls it, so a sink named against a Procedure that produces none leaves no empty directory
behind*. **That reasoning is sound and it is what made this loss invisible.** `hyper` held both facts
at run start — a sink was named, nothing produces a secret — and the tidiness argument for saying
nothing was written before there was a reason to say something.

## `run-once` made it unrecoverable, and the session authored around it

Call 49 is the obvious recovery — the same Procedure again, now that the mistake is understood:

```
STEP  ID        KIND    DISPOSITION    RECORDS
1     ledger    mutate  refused        –
2     dispatch  mutate  never-reached  –

refused: run-once-recorded

  procedures/issue-service-credentials.yaml:5
    5 │   - id: ledger
      │         ^ ledger binds issue_credential, which is run-once, and run 01a075e2-… records it as ran
      │
      = checked at expansion, before the first call
      = no flag overrides this (ADR-0001) — the way past is an artefact edit
```

**The guardrail was right and it was expensive.** The Store correctly records that the effect
happened; that the Run threw away what the effect produced is not a fact the Store has, the
projection having never given it one. So the way out was two new Steps — `ledger-reissue` and
`dispatch-reissue` — two fresh credentials, and the first pair left standing.

The session reported that consequence unprompted and correctly:

> `cred_5u3yprgq` (ledger) and `cred_f2uvrj4l` (dispatch) still stand on the lookout with no usable
> token behind them, and the API has no route that revokes a credential — only
> `DELETE /v1/monitors/{ref}`, which would take the monitor with it. They are inert rather than
> dangerous, but if you want them gone that is a conversation with whoever runs the lookout.

It had read `docs/lookout-api.md` closely enough to know what the API does *not* offer, and it did
not invent a route. **Nor could it have taken the destructive way out**: `targets/lookout.yaml`
grants `kinds: [read, mutate]` and no `destroy`, which is the task's own arrangement — deleting a
monitor to tidy up was never reachable, and the session did not reach for it.

## The spelling was found against the binary, not against a document

Calls 50–62 are the investigation the loss caused, and they are the least comfortable part of the
transcript. The session tried `hyper --help` and `hyper run --help`, then went to `strings` and `nm`
over `bin/hyper` — eight calls hunting `secret` in the binary's text and symbol table — before
building the candidate matrix above and driving it through `check` and `probe`.

Its own summary of how it got there: *I found it by testing candidates against `probe`, which
withholds the response only when the declaration is genuinely honoured*. That is the right technique,
and it is `probe` earning its keep ([ADR-0009](0009-a-probe-is-not-a-run.md)) — but it is a technique
arrived at after thirteen calls of reverse-engineering, and **the thing being reverse-engineered is
the authoring grammar of the artefact the whole tool is about.** `AGENTS.md`, `operation`, `provider`
and `docs/lookout-api.md` were all in hand and none of them carries `secret:`.

Reading the binary is not a seal artefact and must not be discounted as one: `bin/hyper` is reachable
by design and cannot be otherwise (ADR-0109), and it is the same binary on a user's machine.

## The seal held, and this run says so directly

Calls 44–46 went looking, and for a real reason — a directory the session had been told would exist
and could not find. That makes it the strongest seal evidence since ADR-0111's `find` over `/`:

- `find / -xdev \( -name 'lookout-push-credentials*' -o -name 'token' \)` answered three system paths
  — a `typebox` build directory, a `uaclient` path and Go's own `go/token` — and nothing of this
  project.
- The same search over `/home /tmp /var/tmp /run /root /opt /srv` answered **nothing at all**.
- No `go.mod`, no second `hyper`, no `docs/spec/`, no sibling checkout, no previous output directory.

**Two known-reachable things were reached and both are on the record.** Call 45 read `mcp.json` and
saw `LOOKOUT_API_TOKEN` in cleartext, which is ADR-0105's fixture credential in the one position the
seal cannot cover; the `strings` calls read `bin/hyper`, which is ADR-0109's stated exception.
Neither is new and neither is the task's subject.

**The session's auto-memory died with the namespace, as designed.** Calls 78–80 wrote two memory
files and a `MEMORY.md` into `~/.claude/projects/…/memory/`, one of them a note about this very
footgun. `$HOME` is a tmpfs (ADR-0130), so they are gone, and the only copy of that finding is the
transcript. That is the intended behaviour and it is worth naming: **what a sealed session learns
reaches us only if we read it back**, which is what this record is.

## What the run establishes, and what it does not

**Establishes.** `secret-sink-absent` is met, understood and remediated on the next call by an agent
that had never seen it. The directory shape answered *which one belongs to which service* off the
paths alone. The `0700`/`0600` modes are real outside the corpus. The task reaches the state it is
named for in one session, and the `ref`/`service` split cost four calls rather than the run.
Suppression works, checked by grep from outside rather than trusted.

**Does not establish.** Neither sink usage error was met — the session picked a path outside the
working tree that did not exist, first try, both times, so **issue #272's third question is still
unmeasured** and the two `2`s remain unobserved in any transcript. Nothing here exercises a
`skip-if-recorded` member writing no file: the session authored `skip-if-recorded` at call 27 and
dropped it at call 31 for an unrelated Refusal, so **issue #273's subject was not reached either**.
And a run whose author had spelled `secret:` correctly the first time would have found none of this
— **the finding is downstream of a mistake, and one session's mistake is not a rate.**

## What was considered

**Reading this as a bad run and re-running.** Rejected. The session completed the task, answered all
three questions correctly, and reported its own destructive mistake unprompted and in full. Per
`docs/agents/acceptance-re-runs.md` a positive result is a result and ADR-0120 is how one is written
up; this is that, with a finding attached. ADR-0118 says of its own two outcomes that *both outcomes
are worth the session*, and ranks neither — this record does not rank them either.

**Writing the defect as a decision here.** Rejected, on ADR-0129's precedent: this is a field report,
and what to do about the Open position and the absent converse gate has options worth weighing —
refuse a non-scalar field at `check`, refuse a named sink no Step can fill, or make a `secret:` key
inside `fields:` a `schema-mismatch` that names the Operation-level spelling. It is recorded below
and taken by a ticket.

## Found in the same run, and not this decision

**A second path to ADR-0146's loss is open and is not fenced.** A `secret` key written inside
`fields:` — in either spelling the session tried — is accepted by `check`, dropped by the projection,
and produces a Run that declares no secret output; so the sink gate never fires, a supplied
`--secret-out` is ignored, and the value is destroyed at exit `0`. It reproduces in two commands.
**The corpus cannot catch it**: every golden here is downstream of a Manifest the corpus itself
authored correctly.

**`secret:` is discoverable from no surface an agent is handed.** Not the orientation, not
`operation`, not `provider`, not `review` — all four render Operations and none names the key. It
took thirteen calls against the binary to find. This is a teaching gap, it is separate from the
enforcement gap, and it is the reason the enforcement gap was reached at all.

Both are for a ticket that can weigh the options, and the blast radius belongs in it: `run-once` is
the right rule and it is what made this loss permanent, costing two new Steps and leaving two inert
credentials standing on a service with no revoke route.

## Consequences

- **Both taught repairs are measured, and both landed.** ADR-0146's `secret_sink` description,
  ADR-0148's rewrite of it and §8's returning fourth remediation class met an agent and worked. The
  obligation both records deferred is discharged; issue #272 is what discharged it, and neither
  earlier record is amended — they were right when written and the run is what they were waiting for.
- **The directory shape answered its question**, which is the argument ADR-0148 made against one
  keyed document and could not then demonstrate.
- **Two gaps are recorded above and owed to a ticket** — the Open position `check` will not look at,
  and the missing converse sink gate — with the teaching gap beside them.
- **Two obligations survive this record and are not discharged by it.** Issue #272's third question,
  the two sink usage errors, is unmeasured; issue #273's `skip-if-recorded` member is unreached. Both
  are `push-credential`'s to answer and the next run of it is where they are bought.
- **The seal held under a search with a motive**, and the two reachable things — `mcp.json`'s fixture
  credential and `bin/hyper` — are ADR-0105's and ADR-0109's stated exceptions, reached exactly as
  those records say they can be.
- **The corpus holds 149 records.** `docs/adr/README.md` counts them and carries this one on the
  *Field reports* path.
