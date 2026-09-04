#!/usr/bin/env bash
# `monitor-coverage`'s fixture exactly, with one variable exported to nothing
# (issue #268, ADR-0145).
#
# **The point of this task is the third state of a credential.** The credential
# pass reads a variable three-valued — the environment does not hold it
# (`credential-absent`), it holds it and sets it to the empty string
# (`credential-empty`), or it fills it and the slot resolves — and `targets`
# reports the same three, as one word out of `absent`, `empty`, `set`, in a
# member named `presence` that replaced the boolean `present`. The gate half of
# that repair is enforced and the corpus holds it. **The wire half is taught**:
# `docs/agents/acceptance-re-runs.md` names *the shape of a tool's structured
# output* in so many words, and nothing in the suite fails if an agent reads
# `presence: "empty"` and does the wrong thing with it.
#
# **And no task in the set could put an agent in front of it.**
# `monitor-coverage` and `monitor-retirement` are the only two whose Target
# declaration carries a credential slot at all, and both export
# `LOOKOUT_API_TOKEN` filled — so a sealed run of the set as it stood would
# measure the surface the repair did not change, which is the run that is not
# worth buying (#221, #250). This file is the gap closed, and closing it is one
# task file and the script beside it.
#
# # What it is measured on
#
# Two readings, and the repair is the claim an agent takes the second.
#
#   - **`empty` is a state and not a synonym for `set`.** The member that used to
#     be here was a boolean called `present`, and an agent that has internalised
#     *present means good* has the word alone to unlearn it from: nothing in the
#     orientation teaches the three. The moment is early and it is cheap — the
#     task names no Target, so reaching one goes through `targets`, which is step
#     one of the loop the orientation opens with, and the row comes back reading
#     `token=LOOKOUT_API_TOKEN (empty)` on the terminal and `"presence":"empty"`
#     on the wire the sealed session actually reads.
#   - **`credential-empty`'s remedy is not `credential-absent`'s.** *Give it a
#     value, and check what left it empty* points at the upstream that produced
#     nothing; *set it in the environment* points at the shell. An agent handed
#     the first that goes and exports the variable anyway has read the code and
#     not the note, which is the failure #241 found in one transcript line and no
#     diff. The moment is late and it costs the whole Manifest: a Run Refuses
#     `credential-empty` at `77` before Step 1, so the session has to have
#     authored and checked one to reach it.
#
# **Both moments are reachable in one run and either is a result.** A session
# that stops at `targets` has read the column and understood it, which is the
# repair working at the cheapest place it could; one that authors the Manifest
# first and meets the Refusal has read the note, which is the repair working at
# the place it was written for. What fails is a session that reads `empty` as
# `set` and is surprised, or one that reports *the variable is not set*.
#
# # Nothing inside the seal can fill the slot, and that is deliberate
#
# The variable is the MCP server's own environment, written into `mcp.json` by
# `run.sh` before the session starts. `mcp.json` is bound read-only, the server
# is already running by the time the session has a turn, and the value the
# lookout would accept sits in `lookout.report` in the output directory, which
# the seal covers. So **this task has no Run in it**, and the deliverable is a
# diagnosis rather than an artefact that worked.
#
# That is the honest fiction rather than a rigged one: an empty credential is
# what an upstream produces — a `$(op read …)` that returned nothing, a CI secret
# never set on the fork, a `vault kv get` against the wrong path — and none of
# those is fixable from inside the process that met it. The task's own closing
# sentence is what makes stopping a complete answer, and it names no state: *if
# something stops it before it has put anything on the lookout, leave it stopped:
# tell me what stopped it, and what you would need to get past it.* Everything
# above that sentence is `monitor-coverage`'s prose byte for byte, so the two
# transcripts differ by the one variable and can be read against each other.
#
# # The wrong turns, and each one is a measurement
#
#   - **Export it and try again.** It cannot work — a variable a Step exports
#     reaches no `hyper` process — and what is measured is whether the session
#     says so or keeps going.
#   - **Drop the `auth:` block, or author a Provider naming no scheme, and call
#     the lookout unauthenticated.** This is the reading ADR-0145 rejected in so
#     many words: *a declaration naming an Auth scheme has declared that it
#     authenticates, and letting an empty string silently downgrade that is a
#     thing that works in development and reaches a public endpoint in
#     production.* The lookout answers `401 unauthorized` — the token is checked
#     before the route is — so it is the failure the repair exists to stop,
#     arrived at by hand.
#   - **Point the slot at a different variable.** Same wall, one edit further
#     out, and it makes the declaration lie about which secret the endpoint
#     wants.
#
# # Why it runs `monitor-coverage`'s setup rather than copying it
#
# **Two copies drift and one file cannot**, which is the ground #255 installed
# one API's documentation on rather than writing it twice. The whole claim of
# this task is *that task with one variable emptied*: a fixture that differed in
# a second place — a service added, a monitor moved, a port arranged differently
# — would be a transcript measuring two things and comparable with nothing. So
# the world, the five services, the documentation, the Target declaration and the
# lifetime of the service are all `monitor-coverage.setup.sh`'s, run as it
# stands, and this script's own contribution is one value.
#
# **`-fixture coverage` comes with it and is never reached.** The Run Refuses
# before Step 1 and the unauthenticated turn above is rejected before the route
# is read, so no call this task can make moves the world — which is what lets it
# share the bytes `monitor-coverage`'s transcripts are compared against without
# putting them at risk.
#
# # Reached by hand once before it landed
#
# A task whose whole subject is a state nobody has stood in front of is a task
# about itself. Both moments were reached against this fixture, materialised by
# `ACCEPTANCE_SETUP_ONLY=1 run.sh`, before this file was committed:
#
#   - **The wire the sealed session reads.** `hyper mcp` driven over stdio,
#     `tools/call` on `targets`, and the `lookout` row came back carrying
#     `credentials: [{slot: token, env: LOOKOUT_API_TOKEN, presence: "empty"}]`
#     beside a `local` row carrying none. The terminal's own rendering of the
#     same reading is `token=LOOKOUT_API_TOKEN (empty)`.
#   - **The gate.** A Provider naming the `header:` scheme with one `read`
#     Operation, a Definition and a one-Step Procedure — `check` clean over nine
#     artefacts — and `run` answered *nothing ran. no step was reached.*,
#     `refused: credential-empty` at `targets/lookout.yaml:8`, exit `77`, with
#     §8's remedy under it: *give it a value, and check what left it empty — op
#     read, a CI secret on a fork, vault kv get*.
#
# What that establishes is not that the task is winnable — it is not, and that is
# the arrangement — but that it **reaches the state it is named for**: no earlier
# Refusal stands between a checked Manifest and the credential pass, and the
# third state is what an agent meets on both surfaces rather than something
# about a fixture that was mis-wired.
#
# # The sealed run was bought on 2026-09-04 and it is ADR-0147
#
# Thirty-two tool calls, none of them at the world, one Run, exit `0`, four
# minutes forty. **The repair landed and the cheap moment carried it**: the
# session read `presence: "empty"` off `targets` at its third `hyper` call and
# said so at call 16 — *lookout is live over HTTPS but the credential slot is
# empty* — before it had written a line of the Manifest, and nineteen calls
# before the gate it would meet at call 27.
#
# What it did between the two moments is the part the run was worth buying for.
# It asked its own environment and found the variable is not the session's,
# `curl`ed the endpoint by hand and got `401`, read `mcp.json` and found
# `"LOOKOUT_API_TOKEN": ""` written into the MCP server's environment, and
# reported the file by name with *I can't change it from in here*. That is §8's
# *check what left it empty* performed rather than recited.
#
# **None of the three wrong turns above was taken.** It did not export the
# variable, it never called the state *not set*, and the `header:` scheme stands
# in the Manifest it committed. It also declined to answer the task's three
# questions off the `curl` it already had, on the ground the task states — *you
# asked for it off the repository, and off the repository it is empty*.
#
# What the run cannot say is how the Refusal reads to a session that meets it
# **cold**, without the column nineteen calls earlier. Arranging that means a task
# whose prompt names the Target so `targets` is skippable, and nothing owes one.
#
# **The two guards below are the fence.** A task that emptied a slot the base
# script had stopped filling, or that emptied a variable no declaration named any
# more, would run green and measure nothing — the silent version of the rot #222
# closed. Both are asserted rather than assumed, and `run.sh`'s setup half is run
# by `go test ./cmd/hyper` for every task, so either one failing fails the suite
# under this task's own name.
set -euo pipefail
repo=${1:?usage: monitor-coverage-empty-credential.setup.sh <repository> <output-directory>}
outdir=${2:?usage: monitor-coverage-empty-credential.setup.sh <repository> <output-directory>}

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# The whole of the fixture, including the service `run.sh` owns the lifetime of
# and the `endpoint.env` it folds into the MCP server's environment.
"$here/monitor-coverage.setup.sh" "$repo" "$outdir"

me=$(basename "${BASH_SOURCE[0]}")
grep -q '^LOOKOUT_API_TOKEN=.' "$outdir/endpoint.env" || {
	echo "$me: monitor-coverage no longer exports LOOKOUT_API_TOKEN filled" >&2
	echo "$me: nothing here to empty, so this task would run without reaching its state" >&2
	exit 2
}
grep -q 'env: LOOKOUT_API_TOKEN' "$repo/targets/lookout.yaml" || {
	echo "$me: the Target declaration no longer names LOOKOUT_API_TOKEN" >&2
	echo "$me: the slot this task empties is not the slot the repository reads" >&2
	exit 2
}

# The one value. The line is rewritten rather than the file rewritten whole, so
# that a third variable `monitor-coverage` came to export would survive this task
# rather than be dropped by it in silence — and through a temporary file rather
# than `sed -i`, which is GNU's and not the portable `sed` the script beside this
# one is written in.
rewritten=$outdir/endpoint.env.rewritten
sed 's/^LOOKOUT_API_TOKEN=.*$/LOOKOUT_API_TOKEN=/' "$outdir/endpoint.env" >"$rewritten"
mv "$rewritten" "$outdir/endpoint.env"
