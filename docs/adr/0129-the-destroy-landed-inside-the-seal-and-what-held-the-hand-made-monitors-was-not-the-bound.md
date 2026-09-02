# The `destroy` landed inside the seal, and what held the hand-made monitors was not the Bound

**An agent authored a `destroy` Operation, claimed it on a Definition, bounded a Step at it and ran
it — inside the seal, from the orientation and an API's own documentation, with no `docs/spec/` on
the machine to read.** It is the first transcript in which a Manifest's `destroy` half exists at all.
ADR-0106 closed by naming this as the open question — *the `destroy` half of a Manifest is still
unwritten by any agent: the Target admits `read` and `mutate` only, so nothing here exercised a
`destroy:` claim, a Bound, or a Tombstone* — and issue #255 built the task that asks for one.

**Nothing about the product changes on that account.** No message, no orientation text, no closed set,
no surface. What the run does is discharge three deferred obligations and answer one question the
task was written to ask, and the answer is not the one the task was arranged around.

## The evidence: fifty-three calls, sixteen at the world, six Runs, exit 0

A Claude Code session, headless, 2026-09-02, inside the seal, against `monitor-retirement` (issue
#255) and the local TLS endpoint [ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)
decided on. Fifty-three tool calls, fifty-four turns, seven minutes fifty-five, two dollars
sixty-five, exit `0`.

What it wrote: a Manifest with `list_monitors`, `get_monitor`, `create_monitor` and `delete_monitor`;
two Definitions split observe-from-effect, `kinds: [read]` on one and `kinds: [mutate]` with
`destroy: [delete_monitor]` on the other; and three Procedures. The `destroy` Step is the artefact
that had never existed:

```yaml
  - id: retire
    definition: lookout-fleet
    operation: delete_monitor
    target: lookout
    over:
      assets:
        - field: service
          in: [pricing, warehouse]
    args: {ref: {item: $.ref}}
    bound: 2
```

The endpoint's own log is the independent record of what reached the world — sixteen calls, and the
two that matter are in one Step:

```
DELETE /v1/monitors/mon_1e2ea6 -> 404
DELETE /v1/monitors/mon_ae1fd3 -> 204
```

Two Tombstones, one Run, `completed`. **No hand-made monitor was touched**, which is the failure the
task exists to catch: three were seeded and all three stand, byte-identical across the run's three
surveys.

**It rehearsed both effectful Procedures before running either.** `run --dry-run`, then `run_show`
over the rehearsal, then the Run — twice, on the creates and on the `destroy`. Nothing in the task
asks for that.

## What held the hand-made monitors was the Kind rule, and the session said so

The task's clause — *where this repository has no account of having put a monitor somewhere, it is not
ours to take off* — was written to invite the Bound. The Bound is there, at `2`, and it is not what
did the work. The session's own report:

> It could not have reached anything else: `staging-mirror`, `edge-cache` and `notifier` are not
> Assets under that Definition at all, so they are unreachable by construction, not by filter.

That is §5's Kind-scoped Expansion read correctly off a surface that never states it in those words —
an effectful selector reaches Assets and nothing else, so a monitor `hyper` did not create is
reachable only by a literal `values:` list a reviewer would have seen. **The Bound bounds a runaway
selector; it is not what keeps a selector off other people's things.** The task's fixture measured the
right behaviour and the ticket's reasoning about *why* it would be measured was one rule out.

## Three deferred obligations, discharged

`docs/agents/acceptance-re-runs.md` requires a taught repair to name the run it owes.
[ADR-0126](0126-a-predicate-over-an-expansion-holds-of-all-of-them-and-an-answer-must-name-which.md)
booked two against this task and deferred both until it existed; a third arrived unplanned.

**`answered` naming its member — issue #252, discharged.** The `destroy` Step expanded over two
Assets and one answered `404`, so the Step entry carries the list ADR-0126 made it:

```json
"answered": [{"host": "localhost:40419", "member": "pricing", "status": 404}],
"selector": {"expanded_to": ["pricing", "warehouse"]}
```

The session read it — `run_show` with `expansion` on the real Run — and its report attributes the
`404` to `pricing` and the `204` to `warehouse` in a table, correctly. That is the taught half doing
its job, and it is the first time an agent has been in a position to get it wrong. Before ADR-0126 the
two Tombstones were byte-identical and this key said `404` about neither of them in particular.

**The halt a series-rooted Requirement prints — issue #251, not discharged, and that is a good
result.** The task's sentence is *before you take one off, make sure the lookout still says it is
watching what you think it is*, which §3 quotes when it states that a root that expands is a stricter
test than one that does not. The session wrote **both** shapes and rooted the Requirement at the
narrow one:

```yaml
  - id: still-watching        # series: the whole list, recorded
  - id: confirm-warehouse     # one: a single monitor by ref
  - id: warehouse-is-warehouse
    require: {step: confirm-warehouse, field: service, equals: warehouse}
```

It never entered the hazard, so the sentence never fired. **A run where the taught clause is not
reached is evidence about the hazard's reachability and not about the clause**, and it is recorded as
that: one session, choosing the sound root unaided, is a positive result about the format and leaves
#251's sentence still unmeasured. It stays owed, and the next run of this task is where it is bought.

**`manifest-inconsistent` on a `?` in `path:` — issue #229, discharged, and nobody planned it.** The
session's first `check` was the only one that failed:

```
providers/lookout.yaml  16  operations.list_monitors.http.path  manifest-inconsistent
path: carries a ? — a query is written in the query: key beside it, and a ? here is escaped into
the path rather than opening one
```

ADR-0106 recorded that exact fault costing the previous run a call at the world, a `404` and a
projection failure pointing at the one part of the Manifest that was right. **It is now decided
offline, before anything was sent**, and the session fixed it in one edit and never met it again.
Seven further `check` calls, all clean.

## The drift was found, named, and reasoned about correctly

The fixture drops a monitor whose service fails its first check, so the session is accountable for
five monitors and the world holds four. It surveyed after creating, met seven where it expected
eight, and its report says:

> **pricing never actually got a monitor.** The create returned `201` and a ref, but pricing was
> already down from lunchtime and failed the lookout's first look, so it silently dropped it — the
> survey immediately after showed seven monitors, not eight.

It read the drift off the world, attributed it to the API rather than to `hyper`, and said what the
Tombstone bought: *closes a record that would otherwise claim we hold a monitor we don't*. The task's
third question — *whether anything up there moved that we did not move* — was answered from the Store,
and the answer distinguishes the three untouched monitors from the one movement that was nobody's.

## What the run does not establish, and what it did less well

**One run is one run**, and the API and its documentation are ours — ADR-0105 booked that cost and
this does not discharge it.

**The Requirement is half a check.** `confirm-warehouse` names `ref: mon_ae1fd3` as a literal, read
off the world and pasted into an artefact, and it guards `warehouse` only. `pricing` — the member that
was actually gone — is not what the Requirement is about, so the check that stood between the
`mutate` and the `destroy` could not have caught the thing that had moved. It is well-formed, it
holds, and it is weaker than the sentence that asked for it.

**The creates are five Steps rather than one.** `watch-fleet` unrolls `invoices`, `pricing`,
`search-api`, `session-store` and `warehouse` into five `create_monitor` Steps at `bound: 1` each,
where the format's shape for it is one Step with an `over: values:` list at `bound: 5` — the shape §3
renders in `publish-aliases`. Both are legal and the transcript is not worse for it, but the selector
form is what makes the population one reviewable line, and this session did not reach for it.

**It probed the endpoint with `curl` before writing anything**, twice, and got `401` both times. The
credential is in the MCP server's environment and is unreachable from the session's own shell, which
is the arrangement working; the calls cost nothing and told it nothing it could not read in the
documentation.

## What was considered

**Sharpening the task because the Bound was not what held the line.** Refused, and for ADR-0106's
reason about the duplicate trap: the session reached the correct account of *why* nothing else was
reachable, which is the answer the clause was fishing for. A task rewritten so that only a Bound can
save it would be a task that punishes an agent for understanding the Kind rule.

**Rewriting `monitor-retirement`'s header, which argues the clause invites the Bound.** Refused as an
edit and taken as a correction: the header now says what this run established. The clause does invite
a Bound and the Bound is authored — what the header may not claim is that the Bound is what keeps a
`destroy` off things `hyper` never made.

**Buying a second run to reach #251's sentence.** Refused here. The hazard is reachable — both roots
were written in this very Procedure — and forcing it would mean editing the task until the wrong turn
is the easy one, which is the trap-that-must-fire ADR-0106 already declined.

## Consequences

- **Issue #255's flagship measurement is answered positively.** A `destroy`, a Bound and a Tombstone
  are authorable from the shipped surface by an agent that has never seen the specification, and
  ADR-0096's outstanding transcript — *creating and deleting over HTTP with header auth, with a
  correct `record.identity` and a `bound:` on the `destroy`* — is now closed inside the seal as well
  as outside it (#249).
- **Issue #252's deferred re-run is bought and the repair is confirmed by a session.**
- **Issue #229's repair has its first transcript**, unplanned, and the fault ADR-0106 paid for at the
  world is now paid for at `check`.
- **Issue #251's deferred re-run is still owed**, and this run is why: the sentence was never reached.
  The next run of `monitor-retirement` is where it is bought.
- **`monitor-retirement` stands as authored.** No change to the task, the fixture, or its
  documentation follows from this run; its header gains the by-hand and sealed numbers and one
  corrected sentence about what the Bound does.

## Found in the same run, and not this decision

**The seal does not cover session material kept in `$HOME`, and its assertion does not flag it.**
`run.sh` covers the checkout's *parent* wholesale, `$HOME/bin`, the Claude caches, the Go build cache,
and previous output directories found by searching for an `mcp.json` naming `HYPER_REPO_DIR`. It
covers nothing else under `$HOME`, and the assertion inside the namespace looks for a `go.mod` naming
this module, that same `mcp.json`, and a regular file called `lookout`. Six directories of prior
session material sat beside the home directory when this run was prepared —
`/home/idabic/hyper-249-hetzner`, holding a working Manifest and a Store; `hyper-249-transcripts`;
`hyper-249-bin/hyper`, a **stamped second binary**, which the seal's own claim says is not reachable;
the #248 material; and this ticket's own by-hand completion, which is the answer key to the very task
being run. **None of the three searches matches any of them.** They were moved under the checkout's
parent before the run was bought, so this transcript is clean, and the hole is not.

It is a hole rather than an oversight in the same sense ADR-0106 recorded of the output directory:
whether a given run forages is a property of the run and not of the setup (ADR-0099), so the setup has
to make it impossible. What is owed is a ticket, and the shape needs deciding rather than asserting —
covering `$HOME` wholesale is not available, since the sealed session's own `~/.claude` state has to
work, and a fourth search rule is a rule that goes stale the first time session material is named
something new. **A directory the convention names**, covered by path, is the obvious candidate, and
this run's own preparation is the evidence that a convention is what was missing.

**The rehearsal was used unprompted, twice.** Nothing in the task, and nothing in the orientation's
worked example, asks for `--dry-run` before an effectful Run. The session did it on both effectful
Procedures and read the rehearsal back with `run_show` before committing to either. It is recorded
here because ADR-0110 made the rehearsal reachable from the surface and no transcript had shown an
agent reaching for it on its own.
