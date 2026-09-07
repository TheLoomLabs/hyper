#!/usr/bin/env bash
# `push-credential`'s world exactly, and its prompt with one paragraph added
# (issue #277, ADR-0150).
#
# **The point of this task is a member a `skip-if-recorded` Step found already
# recorded.** A Run over such a Step makes no call for that member and writes
# nothing to the sink for it, and ADR-0150 decided what the Run says about that:
# one line beneath the Step table, one `secrets_skipped` count on the `step` row,
# and no `check` anywhere. **That repair is taught in the half that matters.**
# The line and the member are fenced by four corpus cases and by `internal/cli`'s
# own; whether an operator handed a short sink and a sentence explaining it reads
# the sentence, rather than reporting a failure, is a thing only a transcript can
# say (`docs/agents/acceptance-re-runs.md`).
#
# **And `push-credential` could not put an agent in front of it.** It is the one
# task in the set that reaches a Step declaring `secret:` output, so it is the
# only task the repair could be measured against — but the state needs a **third**
# Run of the agent's own Procedure over Records its own earlier Run wrote, and
# `push-credential` asks for the credentials once. Its two Runs are the Refusal
# and the round trip past it, the Manifest is the agent's to write, and nothing in
# the prompt asks for the Repeatability or for a Run after the one that minted.
# ADR-0150 named that and deferred it; ADR-0152 named it again as the one axis of
# four that was not buyable as the task stood. This file is that gap closed, and
# closing it is one task file and the script beside it (#222).
#
# # Why a task beside it rather than a clause in `push-credential.md`
#
# **A prompt is what a transcript is measured against, and `push-credential`'s is
# already holding three measurements.** Three other axes stand on that task — the
# two sink usage errors ADR-0149's session never met, ADR-0151's `check` message
# and its converse Refusal, and ADR-0152's orientation clause — and the last of
# those is measured by *the call count and the first draft*, read against
# ADR-0149's eighty-one calls and thirteen of reverse-engineering. A prompt with a
# paragraph added is a prompt a session spends different turns on, and the
# comparison that answers *did the clause do its job* would be against a task
# nobody has run. ADR-0150 said as much when it deferred: extending the task
# **moves the baseline** ADR-0149 established.
#
# So `push-credential.md` does not move, and the fourth axis gets a file of its
# own. That is `monitor-coverage-empty-credential`'s shape (#268) and it is here
# for the same reason: two prompts differing in one place can be read against each
# other, and one prompt rewritten between two runs cannot.
#
# # What the paragraph asks for, and why each half of it is there
#
#   Do one of them first and on its own. … Then get the rest by running the same
#   thing again rather than by writing a second one … Minting another credential
#   for something we already hold does not help me — the one it replaces does not
#   go anywhere, and I would have to come back and paste it again.
#
# **The first half is what makes the third Run happen**, and it is the operator's
# own reason rather than an instruction to re-run: someone pasting credentials
# into a deploy environment by hand would rather find out on one of them that
# something is wrong than on both. It asks for the rest through the *same* thing
# widened, because a second Procedure for the second service reaches no member
# that is already recorded.
#
# **The second half is what makes `skip-if-recorded` the natural declaration**,
# and it is true of the API rather than of `hyper`: `docs/lookout-api.md` says
# *ask again and you get another one, with an `id` of its own*, and carries no
# route that takes one back. So a re-run under `repeatable` mints a second live
# credential for a service that already had one, and leaves the first standing
# with nobody able to revoke it — which is the consequence ADR-0149's session
# reported unprompted after `run-once` made its own loss permanent. The default is
# `run-once`, so a session that declares nothing meets `run-once-recorded` on the
# third Run and has to choose; the paragraph is what makes one of the two choices
# the operator's.
#
# **Neither half names a Repeatability, a Step, a Run or a sink.** What is being
# measured is what an agent reads and then decides, and a prompt that spelled the
# answer would measure our own prose (#255, ADR-0105).
#
# # What this task costs a session that `push-credential` does not
#
# `skip-if-recorded` tests the Record before the call, so the identity has to
# resolve before the call too — §4 Refuses `manifest-inconsistent` otherwise
# (ADR-0056) — and the only input that reaches this request is the `ref` in its
# path. So the Manifest that reaches this state is keyed by `ref` and its sink
# fills `<nnnn>/mon_1e6a05/token` where ADR-0149's filled `<nnnn>/ledger/token`.
#
# **That is a real cost and it is left standing.** The task's first question is
# *which of them belongs to which service*, and ADR-0148's argument for a
# directory per identity was that the paths answer it; here they do not, and the
# answer has to come off the Records, `changes` or `show`. Arranging it away would
# mean a credential route that takes a service name, which is not how the API
# names a monitor — the same `ref`/`service` split ADR-0105 put there on purpose.
# **Whether the session notices that it has to answer that question elsewhere is
# the second thing this task measures**, and it is one `push-credential` cannot
# ask, its own Manifest being free to key the sink by service.
#
# # Why it runs `push-credential`'s setup, and changes nothing in it
#
# **Two copies drift and one file cannot** (#255), and the whole claim of this
# task is *that task asked for twice over*. The world, the two services, the
# documentation, the Target declaration and the lifetime of the service are all
# `push-credential.setup.sh`'s, run as it stands.
#
# It differs from `monitor-coverage-empty-credential` in having nothing to edit:
# that task's variable is a value in `endpoint.env`, and this task's is a
# paragraph in a prompt. The world it wants is the world that is already there —
# **two monitors, both already watched, and no trap in the way** — because the
# state it is named for is on the far side of three Runs and a session that spent
# its turns somewhere else would not reach it.
#
# # The two guards, and what they are guarding
#
# **The prose is the thing that goes quiet.** Everything above the added paragraph
# is `push-credential.md` byte for byte, and that is what makes the two
# transcripts comparable — but nothing about a `.md` file makes it stay that way,
# and #277 itself came within one decision of editing the base prompt. So every
# line of `push-credential.md` is asserted to be a line of this one: a base that
# moved without this file moving with it fails the suite under this task's name
# rather than producing a second transcript nobody can read the first against.
#
# **And the world has to hold more than one service.** `push-credential` reaches
# its own state with one; this one cannot. *Do one of them first, then the rest*
# has no rest in a world with a single service, and the re-run would reach the
# wholly skipped Step rather than the mixed one — which is the easier half of what
# ADR-0150 decided, and not the half that decided it.
#
# Both are asserted rather than assumed, and `run.sh`'s setup half runs under
# every `go test ./cmd/hyper` (`cmd/hyper/acceptance_test.go`), so either failing
# fails the suite here.
#
# # Reached by hand once before it landed
#
# ae986fd's practice, which `push-credential.setup.sh` kept: a task whose whole
# subject is a state nobody has stood in front of is a task about itself. Every
# moment below was reached against this fixture, materialised by
# `ACCEPTANCE_SETUP_ONLY=1 run.sh`, before this file was committed — a Provider
# naming the `header:` scheme with a `read` that lists and a `mutate` that mints,
# `secret: [token]`, `repeatability: skip-if-recorded`, `identity: "{ref}"`, two
# Definitions, two Procedures, `check` clean over eleven artefacts.
#
#   - **`identity` first.** `identity: $.body.data.credential.service` is
#     `manifest-inconsistent` — *resolves only from the response — a
#     skip-if-recorded test must resolve before the call*. That Refusal is what
#     stands between this task and a sink keyed by service name, and it is
#     `check` rather than a Run.
#   - **The Refusal, unchanged.** `run issue-push-credentials` over one member and
#     no sink answered `refused: secret-sink-absent` at `77`, with §8's remedy
#     under it. `push-credential`'s first Run is this task's first Run.
#   - **The second Run.** `--secret-out ../sink-a` over the one member: completed,
#     exit `0`, `0001/mon_1e6a05/token`, `0700` and `0600`.
#   - **A sink path something is already standing at**, which this task reaches by
#     construction rather than by a session's mistake: the third Run cannot reuse
#     the second's directory, and `--secret-out ../sink-a` again answered *hyper
#     run: --secret-out ../sink-a: something is already there* at `2`. That is one
#     of the two usage errors ADR-0149 left unobserved, met here without being
#     asked for. The other, a path inside the working tree, answers *the path
#     resolves inside the repository working tree* at `2`.
#   - **The third Run, which is the state this task is named for.** The Expansion
#     widened to both refs, `--secret-out ../sink-b`: one call, one file, and
#
#         STEP  ID     KIND    DISPOSITION  RECORDS
#         1     issue  mutate  ran          2
#
#         step 1 skipped 1 record and wrote no secret for it.
#         a member skip-if-recorded found already recorded makes no call, so there was no value for the sink to hold.
#
#     `RECORDS 2` beside a sink holding one file is exactly the row ADR-0150 said
#     is byte-identical to a Step that wrote everything, and the line is the only
#     surface that says otherwise.
#   - **And a fourth Run says the other half.** Both members recorded:
#     `"disposition":"skipped-as-already-recorded"`, `"records":2`,
#     `"secrets_skipped":2` on the wire the sealed session reads, and a sink
#     directory `hyper` made and left empty.
#
# None of this says the task is winnable in one session, and it is a longer walk
# than `push-credential`'s. It says the state **is reachable**: no earlier Refusal
# stands between a checked Manifest and the third Run, and both halves of
# ADR-0150's report are on the far side of it.
#
# # The run this task exists for is not bought here
#
# #277 bought `push-credential`'s run and left this one, which is the answer that
# ticket says is complete so long as it is written down: the three axes standing
# on the unchanged task are measured against the baseline ADR-0149 set, and the
# fourth is fenced by this file and unspent. `docs/agents/acceptance-re-runs.md`
# owns the decision and it costs a session and real money.
set -euo pipefail
me=$(basename "${BASH_SOURCE[0]}")
repo=${1:?usage: $me <repository> <output-directory>}
outdir=${2:?usage: $me <repository> <output-directory>}

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
base=$here/push-credential.md
variant=$here/${me%.setup.sh}.md

# The prose guard runs before the world is raised, because it is a property of
# two files and needs neither a repository nor a service — and a failure here
# leaves nothing running to be killed.
#
# Every line of the base has to be a line of this task's prompt. It is `grep`
# rather than `diff` so that the four tools `run.sh` declares stay four, and it
# is unordered for the same reason: what it is guarding against is a paragraph
# rewritten on one side only, and a base line that appears nowhere in this file
# is that, whatever order the rest is in.
if dropped=$(grep -Fxv -f "$variant" "$base" | grep .); then
	echo "$me: push-credential.md has lines this task's prompt does not:" >&2
	echo "$dropped" >&2
	echo "$me: the two prompts differ in more than the one paragraph, so their transcripts cannot be read against each other" >&2
	exit 2
fi

# The whole of the fixture: the world, the services, the documentation, the
# Target declaration, and the service whose lifetime is `run.sh`'s.
"$here/push-credential.setup.sh" "$repo" "$outdir"

# *Do one of them first, then the rest* needs a rest. One service would reach the
# wholly skipped Step and never the mixed one, which is the case ADR-0150 turned
# on.
services=$(find "$repo/services" -mindepth 1 -maxdepth 1 -type d | grep -c . || true)
if [ "$services" -lt 2 ]; then
	echo "$me: push-credential now ships $services service(s) and this task needs at least two" >&2
	echo "$me: with one there is no second member to skip, so the Run this task is named for is the empty sink and not the short one" >&2
	exit 2
fi
