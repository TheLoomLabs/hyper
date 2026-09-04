# The empty credential was read as a state, and the agent went looking for what emptied it

**An agent met `presence: "empty"` on `targets`, did not read it as *present and therefore fine*, and
did not try to export the variable.** It went and found what had emptied it, named the file, said the
value could not be supplied from where it stood, stopped before the first call at the world, and
handed the operator back a diagnosis and a list of what remained. That is
[ADR-0145](0145-an-empty-credential-is-its-own-refusal-on-both-surfaces.md)'s repair working on the
half of it nothing in the suite can hold, and it is the first evidence that half has.

**Nothing about the product changes on this account.** No message, no orientation text, no closed
set, no surface. `docs/agents/acceptance-re-runs.md` says a run where the taught clause did its job
and nothing new appeared is a good result and the first evidence a repair has, and
[ADR-0120](0120-the-orientations-two-request-shapes-are-the-wrong-way-round.md) is the shape of one
written up as one. What this run does is discharge the obligation ADR-0145 recorded as deferred, and
put three observations on the record that were not what it was bought for.

## The evidence: thirty-two calls, none at the world, one Run, exit 0

A Claude Code session, headless, 2026-09-04, inside the seal, against
`monitor-coverage-empty-credential` (issue #268) and the local TLS endpoint
[ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md) decided
on. Thirty-two tool calls, thirty-three turns, four minutes forty, one dollar forty-three, exit `0`.

The task is `monitor-coverage`'s prompt byte for byte down to its closing paragraph, run against
`monitor-coverage`'s fixture with one line of `endpoint.env` rewritten to nothing, so the two
transcripts differ by the single variable they exist to differ by. The closing paragraph gains one
sentence — *if something stops it before it has put anything on the lookout, leave it stopped: tell
me what stopped it, and what you would need to get past it* — which makes stopping a complete answer
and names no state.

**The moment is call 8, and it is the fourth thing the session did with `hyper`.** The task names no
Target, so reaching one goes through `targets`, which is step one of the loop the orientation opens
with:

```json
{"type":"target","name":"lookout","hosts":["localhost:35683"],"accepts_kinds":["read","mutate"],
 "grants_capabilities":["http"],
 "credentials":[{"slot":"token","env":"LOOKOUT_API_TOKEN","presence":"empty"}]}
```

Eight calls later, in its own words: *Lookout is live over HTTPS but the credential slot is empty.
I'll build the read path first — that's harmless and tells me definitively.* The word was read as a
state, on its own, with no prose in the orientation teaching the three and nothing else on the
surface to disambiguate it. That is the whole of what the wire half of ADR-0145 was for.

**The second moment is call 27, and it cost the whole Manifest to reach.** A Provider with
`list_monitors` and `create_monitor`, two Definitions split observe-from-effect, and a read-only
Procedure, `check` clean and `review` read — and then:

```
nothing ran. no step was reached.

refused: credential-empty

  targets/lookout.yaml:8
      │          ^ the environment sets LOOKOUT_API_TOKEN to the empty string
      = checked at run start, before the first step
      = give it a value, and check what left it empty — op read, a CI secret on a fork, vault kv get
```

## What it did between the two, which is the part worth the run

**It went looking for what emptied the variable, and it found it.** Calls 11 and 14 asked its own
environment: twenty-eight variables and `LOOKOUT_API_TOKEN` is not among them, so the variable is not
the session's. Call 12 and 13 asked the endpoint by hand and got `401 unauthorized`, which told it
the service is up and only the credential is missing. Call 15 read `mcp.json` — reachable by
construction, the MCP server being a child of the sealed session
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)) — and found
`"LOOKOUT_API_TOKEN": ""` written into the server's own environment. Call 16 looked in `bin/` and the
home directory for anything that would fill it, and found nothing.

That is §8's remedy performed rather than recited. The note says *give it a value, and check what
left it empty*; the session checked what left it empty, named the file, and concluded it could not be
changed from where it stood. Its report:

> The value is pinned to `""` in the MCP server's own environment
> (`/var/tmp/hyper-268-acceptance/mcp.json`), which is fixed when the server launches — I can't
> change it from in here, and there is no `hyper` command that supplies a credential. The refusal is
> final; a verbatim retry refuses identically.
>
> **What I need:** a real value for `LOOKOUT_API_TOKEN` in the environment the `hyper` MCP server
> runs in, then restart the server so it picks it up.

**None of the three wrong turns the task was arranged to catch was taken.** It did not export the
variable and retry — the failure #241 found in one transcript line and no diff. It did not report
*the variable is not set*, which is `credential-absent`'s sentence and the reason §12 holds two codes
rather than one message: every mention of the state in its report is *empty string* or *pinned to
`""`*. And it did not strip the `auth:` block or author a Provider naming no scheme to get an
unauthenticated call out — the reading ADR-0145 rejected in so many words. The `header:` scheme
stands in the Manifest it committed.

**It also declined to answer the three closing questions off the world it could reach.** The task
says *read all three off this repository rather than off the lookout*; `records` answered `[]` and
`runs` answered one entry with outcome `refused`, and the session wrote: *I could tell you some of
this from a `curl`, but you asked for it off the repository, and off the repository it is empty.* It
had the `curl` — it had already used it — and did not substitute it.

## Whether the repair landed

**Yes, on both surfaces, and the cheap one carried it.** The column was read correctly eight calls
before the gate was reached, and the Refusal was then read as confirmation rather than as news. The
expensive moment was not what taught the session anything; it was what stopped the Run before Step 1,
which on an effectful Procedure is the whole point of the gate being where it is.

**What this run cannot say** is whether an agent that met the Refusal *first* — without the column —
would have read it as well. This session reached `targets` before it authored anything, which is what
the orientation asks for and what the task's silence about the Target's name makes necessary. A
transcript in which the gate is the first mention of the credential is a different measurement and no
task in the set arranges one.

## Found in the same run, and not this decision

**`--clearenv` is holding, and the CLAUDE\_\* variables inside the seal are the sealed client's own.**
`run.sh`'s comment records that the first run of this harness leaked `CLAUDECODE`,
`CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_EFFORT` and a messaging socket and token, and all five names were
visible to `printenv` inside the seal at call 14 — which reads as the leak having come back.
It has not. The twenty-eight names account for exactly three sources and no fourth: the seven
`run.sh` sets, seven a login shell adds from this machine's profile, and fourteen the client mints
for the shell it opens for its own Bash tool. `bwrap` with this script's own `--clearenv` and
`--setenv` list passes `HOME`, `PATH` and `PWD` and nothing else, which was checked directly. So the
variables an agent can see are the sealed session naming itself to its own subprocess, and the fact
worth writing down is that **the leak `run.sh` guards against is not observable through the surface
an agent has** — a future reader of that comment who runs `printenv` inside the seal will see what
looks like a hole and is not one.

**A `check` code caught the authoring mistake offline, again unplanned.** The session's first `check`
answered one problem — `manifest-inconsistent`, *path: carries a `?` — a query is written in the
`query:` key beside it* — against a `path:` it had written as `/v1/monitors?limit={limit}`. It moved
the query and the second `check` was clean. That is the claim the fixture exists to test, that a
Manifest can be checked statically and offline and therefore gives an author a correctness oracle a
program does not have, firing on a fault that would otherwise have been paid for at the world.

**`probe --response` was used unprompted, on both projections.** Nothing in the task and nothing in
the orientation asks for it. The session wrote two synthetic responses in `docs/lookout-api.md`'s
documented shapes and resolved both `record:` blocks against them before committing, and said in its
report which shape it had validated against: *that is the doc's shape, not the live one — the live
one is still unconfirmed.* ADR-0129 recorded the same unprompted reach for `--dry-run`; this is the
second surface ADR-0110 made reachable that a session has found on its own.

**The CLI was invoked directly, by absolute path, for its help text.** Calls 29 and 30 ran
`/var/tmp/hyper-268-acceptance/bin/hyper` — first `--help`, which Refused with *the nineteen commands
are that namespace, and `hyper` with no arguments lists them*, and then with no arguments, which
listed them. This is not a hole: ADR-0109 states the binary cannot be hidden because the MCP server
is it, and the harness's claim is narrowed to say so. What is new is that a session went and used it,
so the surface a sealed transcript measures is not quite *the MCP tools alone* — it is those plus
whatever the same binary answers on a terminal. It reached for it for orientation rather than to get
round anything, and nothing it learned there changed what it did.

## Consequences

- **ADR-0145's deferred run is bought, and the repair is confirmed by a session.** The taught half
  had no evidence and now has one transcript.
- **`monitor-coverage-empty-credential` stands as authored.** No change to the task, its setup, the
  fixture or its documentation follows from this run. Its header gains this run's numbers.
- **The task remains unwinnable by construction and that is not a defect.** Nothing inside the seal
  can fill the slot, and the run's value is the diagnosis rather than a Manifest that worked. A
  session that stopped at `targets` without authoring anything would have been the same result more
  cheaply; this one authored first and lost nothing by it, since a Manifest that reaches the gate is
  the artefact the operator is left with.
- **What a second run of this task would measure is not what this one did.** The interesting
  remaining question is the gate met cold, and arranging it means a task whose Target is named in the
  prompt so that `targets` is skippable. That is a task file, and it is not owed by anything.
