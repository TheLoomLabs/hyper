# A Manifest is writable from the surface, and both costs were paid at the world

**An agent authored a working Provider Manifest inside the seal — from the orientation and an API's
own documentation, with no `docs/spec/` on the machine to read — and drove it to a completed Run.**
It is the first transcript in which `providers/` holds a Manifest that works. That artefact is the one
issue #221 opened by naming — the one no run had ever landed and the one every real user has to write
first — and an earlier run had already reached for one and met `capability-reserved` on the way
(ADR-0099, issue #218).

**Nothing about the product changes on that account.** No message, no orientation text, no closed
set, no surface. What the run changes is three things that are not the product: two defects it paid
for at the world (issue #229, issue #230), and a hole in the seal that this run looked at and walked
past.

## The evidence: fifty-two calls, eight clean checks, and a Manifest right the first time

A Claude Code session, headless, 2026-08-29, inside the seal, against `monitor-coverage`
(issue #227) and the local TLS endpoint [ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)
decided on. Fifty-two tool calls, five minutes forty, two dollars, exit `0`.

What it wrote, unprompted by anything naming it: a Manifest with a `read` and a `mutate` Operation,
two Definitions split observe-from-effect, and two Procedures. Every axis §3 exposes at that position
is right in the file it first wrote:

- `class: lookout` matching the Target's, and `capabilities: [http]` equal to what its Operations
  derive rather than a superset of it;
- `auth: {header: {name: Authorization, prefix: "Bearer "}}` against the `token:` slot the shipped
  Target declaration carries, which is slot coverage checked at the binding and never in one file;
- an `input:` schema per Operation with every declared input reached by a hole, and `window` declared
  `integer` so `"{window}"` reaches the wire as a JSON number (ADR-0078);
- `repeatability: skip-if-recorded` on the `mutate` with `identity: "{service}"` — the hole rather
  than a response path, which is the one place an identity may not be an ordinary projection
  (ADR-0056), and the rule the orientation states in one sentence;
- a `record:` on both Kinds, projecting `over: $.body.data.monitors` for the list and
  `$.body.data.monitor.…` for the create — **the two shapes the fixture deliberately made
  different**, which is the awkwardness ADR-0105 bought and the one place a projection written off
  the list would have halted the Run.

**Eight `check` calls, and every one of them clean.** The offline oracle never fired, because
everything it can decide the session got right before asking. That is the sharpest number in the run
and it cuts both ways — it is what a correctness oracle looks like when it works, and it is why the
two faults the session did have cost a call each.

**The loop was the orientation's own.** `providers`, then `provider shell`, then `operation` — *the
Manifest's own lines, verbatim: that is the format you author in* — before a line was written. That
is [ADR-0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md)'s
finding holding a second time on a task an order harder than the one it was made on.

**And it surveyed before it created.** `survey-lookout` ran first, so `watch-services` names the two
services that were actually missing, and the fixture's `409 already_watched` was never reachable. The
endpoint's own log is the independent record of that: two `POST`s, two `201`s, and no call the task
did not ask for.

## What the run establishes, and what it does not

**Establishes.** The deepest artefact in the format is writable from the shipped surface by an agent
that has never seen the specification. The two-key rule and observe-or-effect were held without being
told — two Definitions over one Provider, `read` in one and `mutate` in the other. And the three
closing questions were answered off the Records rather than off the service, which is the measurement
issue #227 was written to make: *which were already watched* as Observations, *which this repository
is accountable for* as Assets with their projected fields, and the retired monitor read, recorded and
left alone. A Manifest whose projection merely parsed could not have answered any of them.

**Does not establish.** The API is ours and its documentation is ours — ADR-0105 booked that cost
when it chose a local fixture, and this run does not discharge it. One run is one run. And the
`destroy` half of a Manifest is still unwritten by any agent: the Target admits `read` and `mutate`
only, so nothing here exercised a `destroy:` claim, a Bound, or a Tombstone.

**One piece of evidence is weaker than it reads.** The session declined to remove the retired
monitor and gave the right reason — *no step ever named it, and the repository holds no authority to
remove it* — but the Target admits no `destroy`, so it was declining something it could not have
done. What that shows is the reasoning, not the restraint.

## Both costs were paid at the world

The run had two faults, and neither was an artefact `check` refused.

**A query string written into `path:`.** `path: /v1/monitors?limit=100` passes `check` clean and
reaches the server as `GET /v1/monitors%3Flimit=100`, because `URL.String()` escapes a path component
and a raw `?` is not one. The endpoint answered `404`, the collection path found nothing, and the
surface reported `projection_failed_path: $.body.data.monitors` — pointing the author at the one part
of the Manifest that was right. It is decidable from one file with no Target, no credential and no
network, which is the whole of §4's claim, and it is issue #229.

**A response nothing shows.** A projection reads from a response object, and the only surface that
renders one raw is a Probe, which is `read` against `local` and may never reach a credentialled
Target (§9, ADR-0009, ADR-0017) — so the Provider being authored is the one thing it cannot be
pointed at. ADR-0105 noted that in passing while deciding the endpoint. The session invented the
route that is left: an Operation projecting whole bodies, a throwaway Procedure to run it, the
Records read back, then both deleted. It works, it costs a Run, and it leaves Observations that only
`compact` clears — which the session reported and correctly declined to reach around. That is
issue #230.

Read together they are one sentence: **everything the offline oracle could decide, it decided, and
both detours were at the world — one of them needlessly.**

## The seal does not cover the harness's own output directory

Looking for where Records live, the session ran `ls -a` over `/home/idabic/acceptance-227` — the
harness's own output directory — and got back `mcp.json`, `endpoint.env`, `lookout.report`,
`lookout.pem`, `bin/`, `run.log` and `transcript.jsonl`. It read none of them, ran none of them, and
moved on to `changes` on its next call. A `find` over `$HOME` in the call before reached this
machine's Go build cache.

**Nothing was foraged, and that is not the point.** What sits in that directory is `bin/hyper`, a
stamped runnable binary, against a harness whose repeated condition is that *the MCP surface is the
whole of what they had*; `bin/lookout`, a compiled copy of the fixture whose strings are the API's
answer key, including the seeded monitors and every code it refuses with; three files carrying the
fixture's credential; and the session's own transcript. ADR-0099 settled the rule that governs this
and it applies unchanged: *whether a given run forages is a property of the run and not of the setup,
so the setup cannot be trusted to control for it: it has to be made impossible.* The seal was built
against the checkout because that is where the specification is. The output directory was created by
the same script and covered by nothing.

**So it is a hole, and closing it is owed before the next transcript is collected.** It is not closed
here because the shape needs trying rather than asserting: the repository lives *inside* the output
directory, so covering the directory covers the thing the session is supposed to be working in. A
`--tmpfs` over the output directory with the repository bound back on top of it is the obvious
candidate, and `run.sh` already knows the difference between the two paths. The transcript is written
through a file descriptor the parent opened, so it needs no reachable path of its own.

## What was considered

**Changing something on the product because the run found two faults.** Refused. One is a check that
is absent rather than a rule that is wrong, and the other is a question about a surface that does not
exist yet; both are issues with their own acceptance criteria, and neither is a decision this ADR
gets to make on one transcript.

**Sharpening the task because the `409` never fired.** Refused, and this is the one worth writing
down: the duplicate trap did not fire because the session surveyed first, which is *the correct
answer*. A task whose trap must fire is a task that punishes the right answer, and the trap's job is
to be there for a run that skips the survey rather than to be sprung by every run.

**Removing `limit` so that paging must be handled by a `pagination` Pattern.** Refused. Nothing
inside the seal teaches Patterns — the orientation does not mention them and the only built-in
Provider declares none — so a fixture that forced one would measure the documentation rather than the
agent. That the orientation never names Patterns is a real observation and it cost this run nothing,
`limit` being in the API's documentation where an author would look; it is recorded here and
ticketed nowhere.

## Consequences

- **Issue #227's flagship measurement is answered positively, and issue #221's user story 8 with it.**
  `providers/` has been filled by a session. What replaces *can an agent author a Manifest* as the
  open question is *can it author one carrying a `destroy`*, which no fixture here currently admits.
- **Half of [ADR-0096](0096-the-shape-carried-whole-is-the-effectful-one-and-the-example-is-not-the-acceptance-task.md)'s
  second outstanding transcript now exists.** It asked for an effectful Operation *creating and
  deleting over HTTP with header auth, with a correct `record.identity` and a `bound:` on the
  `destroy`*. The create, the header auth and the identity are recorded here; the delete and the
  Bound are not, and this fixture's Target cannot produce them.
- **ADR-0100's fix is confirmed by a second transcript.** `review`'s page arrived in the structured
  content — the session's `review` returns open with `"rendering":"  PROCEDURE …"` — so the surface
  an agent is told to call before handing work back is one it can now actually see. That was issue
  #217, found in the previous run, and this is the run that shows it landed.
- **The endpoint's log is evidence and is read beside the transcript.** It records which calls
  reached the world, in order, with the status each got, and it is the one channel a session cannot
  narrate its way past. No previous acceptance run had one.
- **`monitor-coverage` stands as authored.** No change to the task, the fixture, or its
  documentation follows from this run.
- **The seal grows a ticket, not an amendment.** ADR-0099's decision is unchanged and its reasoning
  is what condemns the gap; what is owed is the covering, and it is owed before the next transcript.

## Found in the same run, and not this decision

**A clean `check` still reaches the agent as an empty rows array.** Eight times here, as
`{"rows":[],"truncated":false}` and never as the summary line §9 promises — this client surfaces
`structuredContent` and drops `content`, which ADR-0099 first recorded and ADR-0100 reasoned about
when it moved `review`'s page across. ADR-0100's ground for leaving `check` alone is that its block
is *composed of members already here*, and that is exactly what fails in the clean case: an empty
array has no count in it and no sentence. **It cost this run nothing** — the session read the empty
array as clean eight times and was right every time — which is why it is recorded here and not
ticketed. A run where it costs something is the argument for a `summary` member, and this is not
that run.
