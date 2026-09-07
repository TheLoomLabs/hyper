#!/usr/bin/env bash
# The lookout the push-credential task points at, the two services it watches,
# the documentation for its API, and the Target declaration this script ships —
# plus the service itself, which is what makes this the third of the setup
# scripts that start a process of their own.
#
# **The point of this task is a Step whose Operation declares `secret:` output**
# (issue #271). It was the only task in the set that reached one until
# `push-credential-already-recorded` was written beside it (#277), and until this
# one landed there was no transcript in which anything the Secret sink teaches
# could be measured. Two repairs deferred the run they owed on that ground and
# recorded the deferral in their own last consequence:
#
#   - **#266 / ADR-0146** taught the `run` tool's `secret_sink` description —
#     *supplying a path rescues no Run*, while nothing wrote the file.
#   - **#270 / ADR-0148** taught it again — the description now says `hyper`
#     creates the path as a directory and fills it with one file per value — and
#     taught §8's returning remedy, *the same command again with `--secret-out
#     <path>`*.
#
# Both are **taught** in the sense `docs/agents/acceptance-re-runs.md` fixes:
# enforced in the half a corpus can hold, and fenced by nothing in the half an
# agent reads and then decides on. Two deferrals on one missing task is the
# argument for writing it, and writing it is a task file and the script beside it.
#
# # What the round trip is
#
# Everything this task exists to put in front of an agent is on the far side of a
# Run that Refuses:
#
#   1. A Manifest with an Operation that mints a credential, declaring
#      `secret: [token]`. An author who does not declare it writes a live
#      credential into the Record and onto the branch the Store pushes to, and
#      `hyper` says nothing — suppression is positional and there is nothing in
#      the position (§7, ADR-0007).
#   2. `run`, which Refuses `secret-sink-absent` at `77` before Step 1: the Run
#      reaches a Step declaring secret output and the invocation named no sink.
#   3. §8's remedy under it, which is the **fourth remediation class**, *a
#      different invocation* — the class's only member, and the one remedy in the
#      whole set the Run's own operator can take without leaving the shell: *the
#      same command again with `--secret-out <path>`, naming a directory outside
#      the repository that is not there yet*.
#   4. The same command again, with a path that is outside the working tree and
#      is not there yet. Getting either wrong is a usage error at `2`, and
#      neither is announced in advance.
#   5. The value, which is now in a directory `hyper` made, one file per value at
#      `<nnnn>/<name>/<field>`. Getting it out of there is the last step and it is
#      the one the directory shape either earns its argument on or does not
#      (ADR-0148).
#
# **Two credentials rather than one**, because one secret is the shape the sink
# was nearly given and refused: a Step expanding over two monitors fills two
# directories, and whether the agent can say which is which is the measurement
# a single file could not have carried.
#
# # Why a route and not a field on the create's answer
#
# The ticket offered two shapes and this is neither exactly, for a reason that is
# about the other tasks rather than this one. A `push_token` beside the monitor in
# a `create`'s answer would have been the cheapest change and would have put a
# live credential in front of `monitor-coverage` and `monitor-retirement`, whose
# transcripts are read against one another across years and neither of which is
# about a credential: an agent that projected the new field as an ordinary member
# would have written one into its Store, which is a second variable in two
# experiments that already have their own. It could not have been hidden behind a
# fixture flag either — `main.go` states the invariant that nothing about the
# service varies between worlds, so a Manifest written against one works against
# another, and a `create` that answered differently per world would end that.
#
# So it is a route every world answers and one document describes, and it is a
# **`mutate`**, which is the half of the ticket's first option that mattered: the
# value exists only because the Run made the call, and that is the side ADR-0146's
# loss was found on. The ticket's second option — a `read` that returns an
# existing secret — was passed over as the weaker measurement: it is a shape a
# Provider author writes rarely, and a Run that merely read a value had nothing at
# stake in the sink.
#
# # The world is arranged to put nothing extra in the way
#
# `-fixture push-credential` is two monitors, both already there, both naming a
# service `services/` names. Every trap the other two worlds arrange is
# deliberately absent: two monitors against a page size of two is one page and no
# cursor, nothing has drifted, nothing is unreachable, and there is nothing to
# create. The four awkwardnesses the API has by design are still there and cannot
# be switched off — the envelope, the collection under a name, the `ref`/`service`
# split, and a create's answer that is not an element of the list — but nothing
# here adds a fifth.
#
# That is a decision and not a kindness. A session that spends its turns on a
# paging trap never reaches the Refusal, and a transcript that stopped short would
# measure the trap rather than the thing two ADRs are waiting on. `monitor-coverage`
# is where the traps are measured; this is where what happens after a Refusal is.
#
# # Where a sink can go inside the seal
#
# `--secret-out` refuses a path inside the repository working tree, so the sink
# has to be somewhere else, and the seal decides what else there is. `$HOME` is a
# `--tmpfs` with three paths bound back on top (ADR-0130), which makes it writable
# and empty and outside the tree; the output directory is a `--tmpfs` on the same
# terms. So there is somewhere, and `run.sh` asserts it rather than leaving this
# task to find out: the probe inside the namespace makes a directory under `$HOME`
# and removes it, and a seal in which that fails stops the harness before the
# session starts. A task that failed because there was nowhere to write would be
# evidence about the seal.
#
# # What this script shares, and what it does not
#
# `monitor-coverage-empty-credential` runs `monitor-coverage.setup.sh` rather than
# copying it, because its whole claim is *that fixture with one variable emptied*
# — a world that differed in a second place would be a transcript measuring two
# things. **This task makes no claim about that fixture** and cannot borrow it: it
# needs a world of its own, its own two services, and a `services/` tree with no
# drift in it. So it does what `monitor-retirement.setup.sh` does, which is stand
# on its own.
#
# **The body below was that script's boilerplate copied, and the ticket this
# header named as owed is the one that ended it** (issue #274). The build, the
# wait on the report, the `endpoint.env` and the Target declaration differed from
# `monitor-retirement.setup.sh`'s in the fixture name, the service list and the
# name in the error message — nothing else — and this script was the third copy of
# them. What stood here said so, and said the factoring was worth someone's ticket
# and not this one's, because it would be a change to the seam every lookout task
# is fenced by made in passing by a ticket about a Secret sink. That ticket was
# written and it landed: the five hunks are `scripts/acceptance/lookout/fixture.sh`,
# sourced below, and what is left here is the world, the Kinds and the services.
#
# **The documentation was never one of the copies, being the first thing that
# would rot if it were.** `docs/lookout-api.md` is installed from
# `scripts/acceptance/lookout/api.md` the way every task installs it, so that one
# API is one document and a sealed session is never graded against documentation
# that lies (issue #255). It describes the credential route the way it describes
# the retire route `monitor-coverage` never asks for — as something the service
# does — and it says the one thing about it an author has to know: **the token is
# in that answer and in no other.**
#
# # Reached by hand once before it landed
#
# ae986fd's practice: a task whose whole subject is a state nobody has stood in
# front of is a task about itself. Both moments were reached against this fixture,
# materialised by `ACCEPTANCE_SETUP_ONLY=1 run.sh`, before this file was
# committed. A Provider naming the `header:` scheme with a `read` that lists and a
# `mutate` that mints — `secret: [token]`, `identity: "{ref}"` — two Definitions,
# two Procedures: `check` clean over eleven artefacts.
#
#   - **The Refusal.** `run issue-push-credentials` answered *nothing ran. no step
#     was reached.*, `refused: secret-sink-absent` at
#     `procedures/issue-push-credentials.yaml:5`, exit `77`, with §8's remedy under
#     it — *the same command again with --secret-out <path>, naming a directory
#     outside the repository that is not there yet*.
#   - **The completing Run.** The same command with `--secret-out
#     ../push-credentials` answered `completed`, exit `0`, one `mutate` Step over
#     two members. The sink held `0001/mon_1e6a05/token` and
#     `0001/mon_58c3d1/token`, the directories `0700` and the files `0600`, each
#     holding the value and no newline.
#   - **Both usage errors, since neither is announced in advance.** A path inside
#     the working tree and a path something is already standing at are each a `2`
#     with a sentence naming the fault, and an agent that types either meets it
#     between the Refusal and the Run.
#   - **The third question has a page.** `changes` renders the two Assets as
#     `created` with `id`, `issued`, `service` and `token: <secret>`, which is
#     *what this repository now holds about each of them, and what it does not*
#     in one reading.
#
# **What that established that the ticket did not know: the task takes two Runs.**
# An effectful Step's Expansion ranges over Assets and never over Observations —
# `over: {observations: …}` on a `mutate` is `schema-mismatch` at `check`, *legal
# only on a read Step* — and the refs the mint route takes are on the lookout
# rather than in the repository. So the session lists first, reads the refs back
# off its own Records, and then authors the Expansion as a literal list. That is
# the `ref`/`service` split doing exactly what it was put there to do (ADR-0105)
# rather than an arrangement of this task's, and it is the one thing standing
# between the session and the Refusal. It is left standing: removing it would mean
# a credential route that takes a service name, which is not how the API names a
# monitor.
#
# None of this says the task is winnable in one session. It says it **reaches the
# state it is named for**: no earlier Refusal stands between a checked Manifest
# and the sink gate, and the round trip on the far side of it is walkable.
#
# # Two sealed runs have been bought against this task
#
# **2026-09-06 — ADR-0149.** Eighty-one calls, six Runs, ten minutes thirty-five,
# $4.28. It met `secret-sink-absent`, took §8's fourth remediation class verbatim
# on the next call, and got two credentials out of `<nnnn>/<name>/<field>` naming
# which belonged to which service — so ADR-0146's clause and ADR-0148's did the job
# they were written for. Forty calls earlier the same session had lost two
# credentials to `token: {path: $…, secret: true}` written inside `fields:`, which
# `check`, `probe` and `review` all passed over. That is where ADR-0151 and
# ADR-0152 came from, and thirteen of its `Bash` calls were spent finding the right
# spelling against the binary.
#
# **2026-09-07 — ADR-0153.** Thirty-nine calls, two Runs, four minutes forty-one,
# $1.67, no Refusal anywhere. It read `AGENTS.md` at call 4, authored
# `secret: [token]` correctly at call 21 in one draft, and spent no call at all on
# where the key goes. **ADR-0152's clause is measured and it landed**, and because
# it landed the three surfaces behind it — ADR-0151's `check` message,
# `secret-sink-unfilled`, and the two sink usage errors — were never reached: each
# fires on a mistake this session did not make. The two sink usage errors are now
# unobserved after two runs, both sessions having named a good path first try.
#
# **The task file did not move between them**, which is what makes the two
# transcripts comparable and is the argument #277 declined to spend: the fourth
# axis got `push-credential-already-recorded` beside this file rather than a
# paragraph inside it.
set -euo pipefail

# **One file raises the lookout for all four lookout tasks** (issue #274). It is
# sourced rather than run, so `$0` is still this script and a failure names the
# task; its own header is the argument for where it sits and what it owns.
. "$(dirname "${BASH_SOURCE[0]}")/../lookout/fixture.sh"

# The world, the Kinds and the services — the three things that are this task's
# rather than the fixture's.
#
# **`-fixture push-credential` is two monitors**, both already there and both
# naming a service `services/` names. What that world is arranged to leave out,
# and why, is the header above.
#
# **`kinds:` admits `read` and `mutate` and stops there.** Minting a credential is
# a `mutate` — the value exists because the call was made — and nothing in this
# task retires anything, so a Target granting `destroy` would put a judgement call
# the task never asked for between the session and the Manifest.
#
# **Two services, and there is no third.** Both are watched already: the drift
# `monitor-coverage` is about is exactly what this task does not want in the way.
# Which monitor belongs to which service is still not written down here — it is a
# fact about the lookout, and the session has to go and read it.
lookout_fixture "${1-}" "${2-}" push-credential 'read, mutate' <<-SERVICES
	dispatch  logistics
	ledger    payments
SERVICES
