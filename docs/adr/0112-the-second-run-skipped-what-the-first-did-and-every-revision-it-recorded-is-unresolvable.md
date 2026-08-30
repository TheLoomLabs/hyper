# The second Run skipped what the first did, and every revision it recorded is unresolvable

**An agent inside the seal ran one Procedure twice over a roster that grew between the two, and the
second Run skipped all seven Steps the first had done and ran only the two that were new.** That is
Repeatability measured — the thing issue #224 wrote `tenant-onboarding` to find out and the hardest
thing in the model to hold from the orientation alone. Thirty tool calls, one Refusal, $1.23, and
both of the task's closing questions answered off the Records.

**It also authored a Cadence, and then told the operator the clock cannot do what they wanted it
for.** That is the third of issue #232's three runs, and the only one that reached §10.

**One defect, and it is under every transcript this harness has collected**: not one of the artefact
revisions the Store recorded in this run resolves in the repository that recorded it. Issue #239.

## The evidence: thirty calls, one Refusal, two Runs

A Claude Code session, headless, 2026-08-30, inside the seal
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)), against
`tenant-onboarding` (issue #224). Thirty tool calls, thirty-one turns, three minutes twenty-seven,
$1.23, exit `0` — the cheapest of the three, against the task with the most in it.

Eight `Bash`, four `operation`, four `check`, four `review`, two `run`, two `ToolSearch`, and one each
of `providers`, `provider`, `project`, `changes`, `run_show`, `runs`.

**One Refusal in the whole run, and it took one call to repair.** `projection-stale` at
`.github/workflows/hyper-onboard-tenants.yml` — *the Procedure `onboard-tenants` declares a Cadence
and the working tree holds no file here — `hyper project` writes it* — and the session fetched
`project`'s schema, called it, re-`check`ed clean and moved on. The orientation says that in one line
and the line worked.

**It read the Repeatability axis before it wrote anything.** Calls 7, 9 and 10 fetched
`mutate_skip_if_recorded`, `mutate_once` and `mutate` in that order, off the summaries `provider shell`
had just given it — *shell; run-once; projects one Record* against *shell; skip-if-recorded; projects
one Record*. Choosing between them is the whole task and it went to the source before choosing.

## The second Run is the measurement, and it is the first one ever taken

The Procedure was run, `hooli` was added to the roster and to the artefact, and it was run again:

```
run 1  01a05194-6729…   7 steps, all `ran`
run 2  01a05194-d314…   shared-conf, acme-dir, acme-conf, globex-dir, globex-conf,
                        initech-dir, initech-conf   →  skipped-as-already-recorded
                        hooli-dir, hooli-conf       →  ran
```

**`run show --expansion` is what carries the answer**, and it carries the half the task specifically
asked for — *I want this repository able to tell me the other three were considered and left alone
rather than never asked about*. Each skipped Step's row holds its identity set and
`"unchanged_since": "01a05194-6729-73f2-ac90-0e8e6a9841ce"`, which is §6's distinction between the two
skips arriving intact at an agent: skipped-as-already-recorded *ran the test and reached a conclusion
about every identity it read*. The session read exactly that and reported it as
*recorded as considered-and-skipped, not as never-asked*.

**`changes --between` answered the second half.** Two `asset created` rows for `hooli`, and the `THE
CODE MOVED` rows beside them — the Procedure revision moving and the catch-all count. The session
asked for the window by naming both Run ids rather than taking the default, which is the form §9
provides for exactly this question.

**It avoided both of the task's traps by choosing the third value.** `skel/shared.conf` is written
once and edited in place afterwards, and the whole thing is on a 06:00 clock — which is
`cadence-run-once` if the Step is `run-once`, and `run-once-recorded` on the second Run if the Cadence
is dropped instead (§4, ADR-0038). The session used `mutate_skip_if_recorded` with `cp -n`, kept one
Procedure, kept the Cadence, and neither Refusal was reachable. On the second Run `shared-conf` skipped
like the rest. That is a correct answer that the task's author had not enumerated, and it is a better
one than the split it expected: one artefact, one clock, and the belt-and-braces `-n` under the
recorded skip.

**`review` taught it to bound.** The first `review` rendered `mutate!` in the gutter on all seven Steps
— the UNBOUNDED flag — and the session added `bound: 1` to each and re-reviewed, watching the gutter
drop to `mutate`. It said so plainly, and named what was left: *"Only OPAQUE remains, which is inherent
to the `shell` Provider and not something I can author away."* A review surface that changed an
artefact, in a run where nothing else did.

## The clock was authored, and then honestly disclaimed

The task asks for 06:00 daily *so a name somebody adds to the roster overnight is set up before anyone
is awake*. No Cadence delivers that: the population is authored in the Procedure, an `assets:` selector
under `skip-if-recorded` being `skip-if-recorded-unreachable` (§4), so every occurrence re-runs the
same authored names and skips every one of them. A name reaching the roster still costs an artefact
edit and a review.

The session found this without being pointed at it, and gave the operator the reason rather than the
symptom:

> **The clock will not pick up a new name by itself.** This is the significant one, and it defeats the
> reason you wanted the schedule. […] The format is a strict YAML subset with no expression language,
> and a Step's `over:` is a selector over **Records**, not over lines of a file — there is no construct
> that reads `tenants/roster` and expands to one Step per name. […] What the cadence genuinely buys you
> is a daily proof that the four existing tenants are still untouched — not autonomous onboarding.

And it put the route it could not take on the right side of the line: *"that needs a Provider Operation
whose request enumerates the file, and installing a Provider is `install`, which is the human's
command, not mine."*

**This is the answer the task's last paragraph invites without naming**, and it is the strongest
evidence any of these three runs produced that the tool's honest limits reach an operator through an
agent rather than being smoothed over. Nothing about it is a defect and nothing is ticketed.

## Every revision the Store recorded is a dangling reference

The Records this run wrote carry Provenance, and Provenance carries revisions. None of them resolve:

```
$ git cat-file -t 46b387bfe071df2e70f5ab52c5c7ff8e59cefb0a   # procedure_revision, run 1
fatal: git cat-file: could not get object info
$ git cat-file -t 6aec6e87134577ab787efc0ae26de1d3d29d0b40   # procedure_revision, run 2
fatal: git cat-file: could not get object info
$ git cat-file -t 1a738a8d8376cafa6dd705708eed97e87eb3057a   # definition_revision, both
fatal: git cat-file: could not get object info
```

`git hash-object procedures/onboard-tenants.yaml` returns `6aec6e87…`, so the ids are right. They are
blob ids computed from the working tree, and nothing ever wrote those blobs into the object database —
because nothing was ever committed, and the orientation's loop does not commit. It ends at the diff.

**§7's promise is half-kept.** *`definition_revision` is the git blob id of the Definition file:
content-addressed, computable offline from the working tree, unmoved by a rebase, and equal exactly
where the content is.* Equality holds. Resolution does not: you can confirm an id against a file you
already have, and you cannot get the bytes back from the id.

**It cost this run the thing the run was for.** The second `review` — the surface the orientation tells
an agent to call before handing work back — could not render a baseline:

```
PROCEDURE │ procedures/onboard-tenants.yaml
          │ no baseline — 46b387bfe071df2e70f5ab52c5c7ff8e59cefb0a is not in this clone
```

The one moment in three runs where a `review` had a real baseline to draw — a Procedure that had run,
being reviewed again after an edit — is the moment it had none. `not-in-clone` is a designed rendering
(§8), and what this transcript shows is that the loop the orientation prescribes reaches it *by
construction* rather than by accident.

**The same root cause reaches `changes`.** Both Runs recorded `repo_revision: 991887a…` with
`repo_dirty: true`, that revision being the fixture's initial commit and the only one there is. §8
computes the catch-all between the two committed revisions and suppresses the command where either
side was dirty, so the row came out `{"type":"code","fact":"other lines changed","count":0}` — while
`tenants/roster`, the edit that caused `hooli`'s Steps to exist, sat modified in the tree. The session
read it correctly and flagged it anyway:

> That last line is wrong about the world, and worth your attention: I did edit `tenants/roster`
> between the runs. […] `hyper`'s code facts anchor on artefact revisions and the repo revision,
> `tenants/roster` is not one of the five artefacts, and the repo revision did not move — so the roster
> edit that *caused* hooli's steps is invisible to the account.

Its diagnosis is exactly right and its verdict is one word too strong: the row is right about the code,
which is what `THE CODE MOVED` is a table of, and `repo_dirty: true` is on the wire on both window
rows, where §8 and `internal/compare/rows.go` deliberately put it — *the marker that stops a consumer
resolving the revision and believing it read what ran*. The session read that marker and did the join.
But it had to, and `count: 0` sits three rows away from the only thing that qualifies it.

Both symptoms are one fact: **an agent that follows the orientation exactly produces a repository whose
Records point at bytes that exist nowhere.** It is issue #239.

## What the run establishes, and what it does not

**Establishes.** Repeatability is authorable and readable from the shipped surface. `skip-if-recorded`
was chosen deliberately over `run-once` after reading both, held across a second Run, and the
`skipped-as-already-recorded` Disposition with its `unchanged_since` was read back and reported as the
distinction §6 built it to be. A Cadence was authored, `projection-stale` was met and repaired by the
one line the orientation gives for it, and the projected workflow was read. `review`'s UNBOUNDED flag
changed an artefact. And the tool's central limit — that a Cadence cannot expand a population — reached
the operator with its reason.

**Does not establish.** `over: {values: […]}` was never used: the session transcribed the roster into
one Step pair per tenant rather than into an Expansion, so §6's Expansion rule and the `THE CODE MOVED`
rendering of a `values:` list growing from three members to four — which is what issue #224 expected to
see — are still unmeasured. Neither `cadence-run-once` nor `run-once-recorded` fired, so the `run-once`
value has still never been in front of an agent. And the Cadence was projected and never executed:
nothing here says a scheduled occurrence works, only that one is written correctly.

## What was considered

**Ticketing the Cadence's inability to expand a roster.** Refused. It is stated in §4
(`skip-if-recorded-unreachable`), it is the tool working rather than failing, and the run's own answer
to it is the behaviour that was being measured.

**Ticketing the projected workflow's sixty-four zero digest.** Refused — it is the fixture's. The
session reported that `project` wrote `echo '000…000  hyper.tar.gz' | sha256sum -c -` and that the
scheduled run would fail its own integrity check, which is accurate and is a fact about
`run.sh`'s placeholder `hyper.yaml`, not about `project`. The orientation warns against hand-writing
the digest and the session correctly declined to.

**Ticketing `other lines changed: 0` on its own.** Refused as a separate ticket, and folded into issue
#239 instead. It is the same root cause as the missing baseline, it has the same repair, and §8's
design for it — the marker on the window row rather than a `+` the wire has no room for — is defensible
on its own terms. What is not defensible is a loop that guarantees the state it degrades in.

**Sharpening `tenant-onboarding` so the `values:` Expansion is forced.** Refused, on the line
ADR-0106 and ADR-0111 both took. The session's per-tenant Steps are more verbose and are not wrong, and
a task that forbids a working design to preserve a rendering is measuring the task.

## Consequences

- **Issue #221's user story 5 is answered and issue #224's flagship measurement with it.** The second
  Run exists, it skipped what the first did, and the agent authored that rather than discovering it.
  What replaces it as the open question is `run-once` — the one Repeatability value no transcript has
  reached — and a Cadence that actually fires.
- **All three of issue #232's runs are done, and §10 has now been reached once.** `project`,
  `projection-stale` and a generated workflow are in a transcript for the first time.
- **One defect, ticketed with this transcript as its evidence**: issue #239 — the artefact revisions in
  Provenance are unresolvable in the repository that recorded them, because the orientation's loop
  never commits.
- **`tenant-onboarding` stands as authored.** No change to the task follows from this run.
- **The `not-in-clone` degradation is under every acceptance transcript, not just this one.** The
  `fleet-rollout` run's Records carry the same dangling ids for the same reason; it did not notice
  because it never reviewed an artefact after running it. This is the run where the loop got long
  enough for the fault to show.

## Found in the same run, and not this decision

**A clean `check` still reaches the agent as an empty rows array**, three times here. Four transcripts
now agree that it costs nothing (ADR-0099, ADR-0100, ADR-0106, ADR-0110).

**The Record captured a real-world detail nothing else would have.** `cp -n` emitted
`cp: warning: behavior of -n is non-portable and may change in future; use --update=none instead` on
stderr, the projection captured it, and `changes` rendered it inside the `asset created` row for
`hooli`'s config. The Step completed and the Run did not halt, which is correct — an effectful
Operation halts on a non-zero exit and this one exited `0`. It is a small demonstration that the
Record holds what a human running the command by hand would have seen and lost.

**The session verified its own account against the world and said which was which.** Its closing
paragraph reads the four `tenant.conf` files' mtimes off disk — the three originals at `09:31:15`,
`hooli` alone at `09:31:43` — and offers them as *independent* corroboration of what the Records say,
having answered the question off the Records first as the task demanded. That is the distinction the
task was written to test and it was held without being reminded of it.
