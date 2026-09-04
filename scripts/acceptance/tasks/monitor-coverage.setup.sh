#!/usr/bin/env bash
# The lookout the coverage task points at, the five services it is supposed to be
# watching, the documentation for its API, and the one artefact this script ships
# — plus the service itself, which is what makes this one of the two setup scripts
# that start a process of their own (`monitor-retirement` is the other, and starts
# the same binary in a world of its own; `monitor-coverage-empty-credential` gets
# one by running this script).
#
# **The point of this task is the Manifest** (issue #227). `providers/` is absent
# from the fixture on purpose — `run.sh` says so of itself, *that absence is the
# gap under test whenever the task is author a Provider* — and it is the first
# artefact a real user has to write and the one no transcript has ever seen
# written. The 2026-08-29 run reached for one and died on `capability-reserved`
# trying to give itself a boundable `destroy`; we know agents go there and we have
# never seen one land. It is also the deepest part of §3 by a distance:
# `operations:` with a Kind and a Repeatability each, a request block, an `input:`
# schema, template holes, an `auth:` scheme whose slots a Target must cover, and a
# `record:` projection with an `identity:` and its `fields:`. Every one of those
# has a `check` code waiting behind it, which is the claim under test — that a
# Manifest can be checked statically, offline, and therefore gives an agent a
# correctness oracle a program does not have.
#
# **What it talks to is a TLS server this script starts** (ADR-0105, issue #226).
# The scheme is `https` and there is no second one (ADR-0082), so a plaintext
# fixture was never available; a real vendor's API would put a live credential in
# a headless session running under `--permission-mode bypassPermissions`; and
# either way a transcript produced a handful of times a year would carry a second
# variable — a rate limit, a schema change, an outage. The endpoint is
# `scripts/acceptance/lookout`, built here the way `run.sh` builds `hyper` and for
# the same reason: the seal hides the source and not the binary. Its certificate
# is trusted through `SSL_CERT_FILE` in the environment `run.sh` folds into the
# MCP server's, which is a position no artefact can name and no agent can author —
# so a Manifest that works here is one that would work against a vendor, and the
# fixture cannot flatter it. **`run.sh` owns the lifetime**, killing the pid this
# script writes on `EXIT`, `INT` and `TERM`: the fence runs the setup half on
# every `go test ./cmd/hyper`, so a service nobody stops is one process per test
# run.
#
# **The Target declaration is shipped rather than asked for**, on the ground issue
# #225 already used for its second Target: it is a fact about the repository an
# operator hands over, and it is not the question under test. Two further things
# ride on that here. The port is the kernel's, taken fresh at every start, and a
# Target the agent authored could not carry a number the harness only learns at
# startup — `host:` is `"{from-target}"`, so the port appears in nothing the agent
# writes and transcripts stay comparable. And a declaration carries the slots the
# scheme it binds requires, so shipping `token:` fixes the scheme at `header:`
# without a word of the task saying so; a declaration carrying `username` and
# `password` beside it would serve `basic:` as well and measure which one an agent
# picks, which is a second question and not this one's. It has to be found, too:
# the task names no Target, so reaching it goes through `hyper targets`, which is
# step one of the loop the orientation opens with.
#
# **`kinds:` admits `read` and `mutate` and stops there.** The task leaves the one
# stale monitor alone by name, and a Target admitting `destroy` would turn *should
# I clean that up* into a judgement call standing between the session and the
# Manifest — `destroy`'s own surface is what `snapshot-lifecycle` is for.
#
# **The documentation still describes the retire route, and that is deliberate.**
# It documents the API rather than the task (ADR-0105), and a fixture whose
# documentation hid a route would be teaching an agent that a service has only
# the calls its operator wants made today. What an agent that tidies up anyway
# meets is `kind-not-granted` — the operator's instruction with a code behind it,
# which is the one place this task measures whether a stated policy is held.
#
# **The initial state is named now, and this task's is unchanged** (issue #255).
# A second task against the same service needed its own fiction, so the state the
# lookout starts in is `-fixture coverage` rather than the only state it had; the
# four seeded monitors, their order and the page boundaries they fall on are the
# same bytes they have always been, and `api_test.go` holds them against a golden
# so a later fixture cannot move them. Transcripts against this task are compared
# with one another across years and that comparison is what the flag protects.
#
# **What did move is two sentences of the documentation, and both had to.** The
# service now looks at a monitor as soon as it is added and does not keep one
# whose service fails that look. Nothing in this task's fiction is switched off,
# so nothing here can reach that behaviour — but `docs/lookout-api.md` documents
# the API rather than the task (ADR-0105), which is the same ground the retire
# route is described on, and a copy of it that omitted what the service does
# would be the documentation that lies the fixture's own fence warns about. So it
# gained a paragraph under *Adding one*, and an agent reading it here learns one
# more fact about a service and meets it nowhere.
#
# The second is `state`'s own sentence, and it is the first one's consequence
# rather than a second decision. It read *`pending` on a monitor we have not
# looked at yet and `active` on one we have*, and moving the first look to the
# moment of adding made that false of every monitor the service keeps: it has
# been looked at and it is still `pending`. Either the sentence moved or the
# field did, and moving the field would have changed what this task's agent
# observes of a world it is being measured in. The sentence moved. `pending` on
# a create's answer and `active` on the seeded four are the same bytes they were.
#
# **What the API asks of an author, and where each one bites.** None of it is
# advertised as a trap: they are properties of an API, and `docs/lookout-api.md`
# describes them the way a vendor's documentation would.
#
#   - **The list pages at two, and there are four monitors.** An agent that reads
#     the first page concludes two services are unwatched that are not, and the
#     lookout answers `409 already_watched` — which halts the Step where it
#     stands (§6), with whatever members preceded it already created. A
#     `pagination` Pattern is §3's shape for this and nothing inside the seal
#     teaches one: the orientation does not mention Patterns and the only
#     built-in Provider declares none. So the reachable answers are a wider
#     `limit`, a second call carrying the cursor, or the Pattern found by other
#     means, and which one it reaches for is a measurement rather than a trap.
#   - **`window` is a whole number of seconds, and the task says *a one-minute
#     window*.** A hole is typed by the input schema it names and only where it is
#     the whole of the value (ADR-0078), so `"{window} "` or `"every {window}"`
#     reaches the wire as a string and is refused `400 invalid_window` — the rule
#     that is invisible in a diff meeting an API that checks. `window: 1` is
#     refused `400 window_out_of_range` on the same call.
#   - **A create's answer is not an element of the list.** `POST` answers one
#     object under `data.monitor`; the list carries them under `data.monitors`.
#     A projection written off the list does not resolve against the create, and a
#     `record:` whose `identity:` does not resolve halts the Run naming the path
#     (§6) — after the call went out, which is the honest cost of a projection no
#     static check can reach.
#   - **A monitor is handled by `ref` and describes a `service`.** Which of the
#     two a Record is named by is the author's choice, and an Operation declaring
#     `skip-if-recorded` has it made for it: that test reads the head of the
#     series before deciding whether to call, so the identity must resolve before
#     the call — a hole, not a `$.` path, `manifest-inconsistent` otherwise (§4,
#     ADR-0056).
#
# **The observe-or-effect split is unavoidable here**, and it is the second
# artefact-level rule the task walks into: listing what the lookout holds and
# creating what it does not are two Definitions over one Provider and one Target,
# because a Definition claiming `read` beside `mutate` is `definition-kinds-mixed`
# (§4, ADR-0032). It is also what makes the task's closing questions answerable —
# the Definition is the segment of a Record's identity that keeps the two series
# apart, so what the session merely looked at and what it is accountable for are
# two sets of Records rather than one set with a flag.
#
# **The three questions are the scoring, and none of them can be answered off the
# disk.** *Which were already being watched* is the Observations; *which monitors
# this repository is accountable for, and what it holds about each* is the Assets
# and their projected fields; *what it can still tell me about the ones it only
# looked at* is those same Observations, still standing after the Run that wrote
# them. A Manifest whose projection merely parses answers none of them, which is
# the difference this task exists to put in front of an agent.
#
# **A second task runs this script and empties one value it writes** (issue
# #268). `monitor-coverage-empty-credential.setup.sh` calls this one, asserts
# that the `LOOKOUT_API_TOKEN` line below was written filled and that the
# declaration still names it, and then rewrites that line to nothing — which is
# the one arrangement in the set that puts an agent in front of the third member
# of credential presence (ADR-0145). Nothing here is arranged for it and nothing
# here may be: what that task claims is *this fixture with one variable emptied*,
# so a change made here reaches both, which is the point of it running this
# script rather than a copy.
#
# **Completed by hand once before it landed** — a Manifest nobody has written is a
# transcript about the task. What that established, in order: `check` clean over
# ten artefacts with the port in the grant; a Run whose `read` Step wrote four
# Observations and whose `mutate` Step wrote two Assets; `changes` rendering
# those as `YOU DID THIS` beside `THE WORLD MOVED`, which is the whole of the
# task's three questions in one page; and a second Run reporting
# `skipped-as-already-recorded` on the create.
#
# Each of the three failures above was reached the same way, and each one is clean
# under `check` first, which is the point of stating them:
#
#   - `window` declared `{type: string}` and sent as `"60"` — `check` clean, the
#     Run halted on `400`, no Asset written.
#   - the create's `identity:` written off the list's shape
#     (`$.body.data.monitors.service`) — `check` clean, the `201` in the service's
#     own log, and the Run halted naming the path that did not resolve. The world
#     was touched and nothing was recorded, which is the cost of a projection no
#     static check can reach.
#   - the first page read as the whole list — the `read` Step wrote two
#     Observations rather than four, and the create halted `409 already_watched`
#     on its first member with nothing created.
set -euo pipefail
repo=${1:?usage: monitor-coverage.setup.sh <repository> <output-directory>}
outdir=${2:?usage: monitor-coverage.setup.sh <repository> <output-directory>}

# This script's own path, three levels down from the checkout, rather than a
# third argument: `run.sh`'s `root` is a local it does not export, and a task
# that needs the source needs it to build with rather than to read.
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)

# Built outside the seal, like `hyper` above it, because the seal hides the
# source and not the binary. `go` rather than a fifth tool: `run.sh` declares
# `bwrap git go python3` and the fence asserts the same four, so anything else
# here would be an edit to the seam this task is fenced by (ADR-0105).
mkdir -p "$outdir/bin"
go build -C "$root" -o "$outdir/bin/lookout" ./scripts/acceptance/lookout

# The report is written atomically by the endpoint once it is listening, so
# waiting for the file is waiting for the service — no sleep long enough to be
# wrong on a loaded machine, and no port guessed before the kernel handed one
# out. A dead process is not waited on for the full ten seconds: what a task
# owes a reader here is the log, and what it must not do is hang the suite.
rm -f "$outdir/lookout.report"
"$outdir/bin/lookout" -dir "$outdir" -fixture coverage >>"$outdir/lookout.log" 2>&1 &
echo $! >"$outdir/endpoint.pid"
for _ in $(seq 1 200); do
	[ -f "$outdir/lookout.report" ] && break
	kill -0 "$(cat "$outdir/endpoint.pid")" 2>/dev/null || break
	sleep 0.05
done
[ -f "$outdir/lookout.report" ] || {
	echo "monitor-coverage.setup.sh: the lookout did not start; $outdir/lookout.log is why" >&2
	exit 2
}
port=$(sed -n 's/^port=//p' "$outdir/lookout.report")
certificate=$(sed -n 's/^certificate=//p' "$outdir/lookout.report")
token=$(sed -n 's/^token=//p' "$outdir/lookout.report")

# What `run.sh` folds into the MCP server's environment, which is the whole of
# how the sealed session comes to trust this certificate and hold this
# credential. Neither is hidden by the seal and neither is worth anything
# outside the process that checks it; `hyper` still stores no secret (ADR-0007),
# resolving the slot from its own environment at Run start exactly as it would
# against a vendor.
cat >"$outdir/endpoint.env" <<ENV
SSL_CERT_FILE=$certificate
LOOKOUT_API_TOKEN=$token
ENV

cat >"$repo/targets/lookout.yaml" <<YAML
kind: target-declaration
target: lookout
class: lookout
kinds: [read, mutate]
capabilities: [http]
hosts: [localhost:$port]
auth:
  token: {env: LOOKOUT_API_TOKEN}
YAML

# The five services, one directory each, with the sort of file a service
# directory has in it so that *what is a service here* is answered by the shape
# of the tree rather than by a list this script also has to keep true. Three of
# the five are already watched and two are not, and which two is not written
# down anywhere in the repository — it is a fact about the lookout, which is
# where the task's first question has to be answered from.
while read -r service owner; do
	mkdir -p "$repo/services/$service"
	printf 'owner = %s\nrestart = on-failure\n' "$owner" >"$repo/services/$service/service.conf"
done <<-SERVICES
	billing   payments
	checkout  storefront
	ingest    data
	mailer    platform
	search    data
SERVICES

# The API's documentation, which is the fixture's own because there are no public
# docs to point at, and which is **installed rather than written here** (issue
# #255). **It documents the API and never the Manifest** (ADR-0105): no §3
# vocabulary, no artefact keys, no talk of projections, Kinds or Patterns. A
# transcript that succeeded because this file described a Manifest would be one
# that measured our prose. What it does carry is everything a vendor's reference
# would — the envelope, the paging, the field types, and every way the service
# says no — because an author who cannot read the API cannot write a Manifest for
# it, and that is not the thing being measured either.
#
# **One API is one document, and two tasks ship the same bytes.** It lived in a
# heredoc here while there was one task. A second task needing the same reference
# would have made it two copies of ~110 lines of prose, and the fixture's own
# fence already states what that costs: the file written into the repository is
# the only description of this API anyone inside the seal can read, so a drift
# between it and the service is a sealed session graded against documentation
# that lies. Two copies drift; one file cannot. It sits beside the service it
# describes and `api_test.go` reads it.
mkdir -p "$repo/docs"
cp "$root/scripts/acceptance/lookout/api.md" "$repo/docs/lookout-api.md"
