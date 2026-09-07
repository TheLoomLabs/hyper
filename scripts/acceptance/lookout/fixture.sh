# The lookout, raised: the build, the wait, the `endpoint.env`, the Target
# declaration and the API's documentation — one copy of each, for every task that
# points at a lookout.
#
# Four tasks do, and three of them start a service of their own. Below
# `set -euo pipefail` those three were ~44 lines each and **five things differed,
# every one of them data**: the `-fixture` name, the `kinds:` the Target grants,
# the service list, and the script's own name in a usage string and in a failure
# message (issue #274). Everything else — the `root` derivation, the `go build`,
# the report-wait loop, the three `sed -n` reads, the `endpoint.env`, the other
# seven lines of the declaration, the `while read` that writes the services, and
# the `cp` of `api.md` — was byte-identical.
#
# **The wait loop is why this is a file rather than a tidy-up.** It is the thing
# standing between a green suite and a task run against a service that never came
# up, and as three copies of a `kill -0` guard, a bound and a message, a repair to
# one of them was two silent divergences. `cmd/hyper/acceptance_test.go` runs
# every task's setup on every `go test ./cmd/hyper`, so a copy that rotted would
# fail — but under whichever task happened to be holding the broken one. The
# argument is the one this fixture's documentation already accepted: **two copies
# drift and one file cannot** (issue #255, which is why `api.md` is installed
# below rather than written into a here-doc).
#
# # Where it sits, and why it is sourced
#
# **It is not in `scripts/acceptance/tasks/`, and that is the fence rather than
# taste.** `run.sh` runs `${task%.md}.setup.sh`, and the suite ranges over every
# `.md` in that directory, so a shared file living there is one `.md` away from
# being a task — a sealed session prompted with a library's own prose. Here it
# cannot become one. It sits beside the service it raises, next to the `api.md`
# it installs and the `api.go` whose `-fixture` names it takes.
#
# **It is sourced rather than run, and that is what keeps a failure named.** A
# setup script says `. …/lookout/fixture.sh` and calls `lookout_fixture`, so `$0`
# is still the setup script `bash` was started on: every message below names the
# task, which is what makes a failure report the task rather than the library.
# Nothing is passed to say so, so nothing about the name can go stale. **It
# carries no shebang** for the same reason the name is what it is — a file that
# looked runnable would run, define a function, raise nothing and exit `0`.
#
# **It is more `bash` and no fifth tool.** `run.sh` declares `bwrap git go
# python3` and the fence asserts the same four; this file adds none.
#
# # What stays with the task
#
# Each setup script keeps its header, which is the argument for one world and is
# not duplication, and keeps the three things that are genuinely its own: the
# world it wants, the Kinds it grants, and its services. Everything below is the
# same in every world by construction, which is the property the three copies
# only happened to have.
#
# **`monitor-coverage-empty-credential.setup.sh` is no part of this and stays as
# it is.** It runs `monitor-coverage.setup.sh` rather than copying it, on purpose
# and behind two guards that fail the suite under its own name if the base stops
# producing the state it edits (ae986fd). It empties one line this file writes,
# and it reaches that line through the task that owns the world rather than
# through here.

# lookout_fixture raises the service, hands the repository what it needs to reach
# it, and takes the service list on stdin:
#
#     lookout_fixture "${1-}" "${2-}" <fixture> <kinds> <<-SERVICES
#         <service>  <owner>
#         …
#     SERVICES
#
# The repository and the output directory are passed through from the setup
# script's own arguments rather than read off `$@` here, because `$@` inside a
# function is the function's own — a sourced file's definitions never see the
# arguments the script was started with. One visible line, and no rule to
# remember.
#
# The Kinds are the text between the brackets of the declaration's `kinds:`, so
# what a task writes is what a reader of `targets/lookout.yaml` will find. A list
# that is not one costs the task its own subtest: the fence runs every setup and
# then `hyper check` over what it left (`cmd/hyper/acceptance_test.go`).
lookout_fixture() {
	local me repo outdir fixture kinds services root port certificate token service owner
	me=$(basename "$0")

	# Tested rather than left to `${1:?…}`, which is what a setup script did while
	# it held these lines: bash prefixes that diagnostic with the file the
	# expansion is written in, so the moment they moved here the library named
	# itself and the task did not. A failure reports the task, and that has to
	# hold for the argument nobody passed as well as for the service that never
	# came up.
	repo=${1-}
	outdir=${2-}
	fixture=${3-}
	kinds=${4-}
	[ -n "$repo" ] && [ -n "$outdir" ] || {
		echo "usage: $me <repository> <output-directory>" >&2
		exit 2
	}
	[ -n "$fixture" ] && [ -n "$kinds" ] || {
		echo "$me: lookout_fixture takes the -fixture name of the world it wants and the Kinds its Target grants" >&2
		exit 2
	}

	# The service list is taken whole here rather than left on stdin for the loop
	# that writes it out below. Everything in between reads no stdin today, and
	# keeping that true by hand is the kind of thing that goes quiet: a call that
	# lost its here-doc writes no `services/` and exits `0`, and under `go test`,
	# where stdin is `/dev/null`, the fence would stay green over a repository
	# missing what the task names.
	services=$(cat)
	[ -n "$services" ] || {
		echo "$me: lookout_fixture was given no services; the repository would answer none of the task's questions" >&2
		exit 2
	}

	# This file's own path, three levels down from the checkout, rather than an
	# argument: `run.sh`'s `root` is a local it does not export, and a task that
	# needs the source needs it to build with rather than to read.
	root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)

	# Built outside the seal, like `hyper` above it, because the seal hides the
	# source and not the binary. `go` rather than a fifth tool: `run.sh` declares
	# `bwrap git go python3` and the fence asserts the same four, so anything
	# else here would be an edit to the seam every lookout task is fenced by
	# (ADR-0105).
	mkdir -p "$outdir/bin"
	go build -C "$root" -o "$outdir/bin/lookout" ./scripts/acceptance/lookout

	# The report is written atomically by the endpoint once it is listening, so
	# waiting for the file is waiting for the service — no sleep long enough to
	# be wrong on a loaded machine, and no port guessed before the kernel handed
	# one out. A dead process is not waited on for the full ten seconds: what a
	# task owes a reader here is the log, and what it must not do is hang the
	# suite.
	#
	# The `-fixture` name is the whole of what a task says about the world it
	# wants; which monitors each one is and what it is arranged to ask is the
	# comment beside it in `api.go`, one directory along from here. The service is
	# started on `/dev/null` because it outlives this function and has no business
	# holding the caller's stdin open.
	rm -f "$outdir/lookout.report"
	"$outdir/bin/lookout" -dir "$outdir" -fixture "$fixture" >>"$outdir/lookout.log" 2>&1 </dev/null &
	echo $! >"$outdir/endpoint.pid"
	for _ in $(seq 1 200); do
		[ -f "$outdir/lookout.report" ] && break
		kill -0 "$(cat "$outdir/endpoint.pid")" 2>/dev/null || break
		sleep 0.05
	done
	[ -f "$outdir/lookout.report" ] || {
		echo "$me: the lookout did not start; $outdir/lookout.log is why" >&2
		exit 2
	}
	port=$(sed -n 's/^port=//p' "$outdir/lookout.report")
	certificate=$(sed -n 's/^certificate=//p' "$outdir/lookout.report")
	token=$(sed -n 's/^token=//p' "$outdir/lookout.report")

	# What `run.sh` folds into the MCP server's environment, which is the whole
	# of how the sealed session comes to trust this certificate and hold this
	# credential. Neither is hidden by the seal and neither is worth anything
	# outside the process that checks it; `hyper` still stores no secret
	# (ADR-0007), resolving the slot from its own environment at Run start
	# exactly as it would against a vendor.
	cat >"$outdir/endpoint.env" <<ENV
SSL_CERT_FILE=$certificate
LOOKOUT_API_TOKEN=$token
ENV

	# The declaration is shipped rather than asked for, on issue #225's ground:
	# it is a fact about the repository an operator hands over, it carries a port
	# the harness only learns at startup, and its `token:` slot fixes the Auth
	# scheme at `header:` without a word of the task saying so. No task names a
	# Target, so reaching it goes through `hyper targets`, which is step one of
	# the loop the orientation opens with.
	#
	# **`kinds:` is the task's**, and the argument for the Kinds one world grants
	# is in the header of the script that passes them: it is the line that
	# decides which judgement calls stand between a session and its Manifest.
	cat >"$repo/targets/lookout.yaml" <<YAML
kind: target-declaration
target: lookout
class: lookout
kinds: [$kinds]
capabilities: [http]
hosts: [localhost:$port]
auth:
  token: {env: LOOKOUT_API_TOKEN}
YAML

	# The services, one directory each, with the sort of file a service directory
	# has in it so that *what is a service here* is answered by the shape of the
	# tree rather than by a list a script also has to keep true. **Which of them
	# the lookout is already watching is never written down here**: that is a
	# fact about the lookout, and a task whose questions start there is a task
	# whose session has to go and read it.
	while read -r service owner; do
		mkdir -p "$repo/services/$service"
		printf 'owner = %s\nrestart = on-failure\n' "$owner" >"$repo/services/$service/service.conf"
	done <<<"$services"

	# The API's documentation, which is the fixture's own because there are no
	# public docs to point at, and which is **installed rather than written here**
	# (issue #255). **It documents the API and never the Manifest** (ADR-0105): no
	# §3 vocabulary, no artefact keys, no talk of projections, Kinds or Patterns,
	# and no mention of any task. A transcript that succeeded because this file
	# described a Manifest would be one that measured our prose. What it does
	# carry is everything a vendor's reference would — the envelope, the paging,
	# the field types, and every way the service says no — because an author who
	# cannot read the API cannot write a Manifest for it, and that is not the
	# thing being measured either.
	#
	# **One API is one document, and every task that reads it ships the same
	# bytes.** It lived in a here-doc while there was one task. The file written
	# into the repository is the only description of this API anyone inside the
	# seal can read, so a drift between it and the service is a sealed session
	# graded against documentation that lies. It sits beside the service it
	# describes and `api_test.go` reads it.
	mkdir -p "$repo/docs"
	cp "$root/scripts/acceptance/lookout/api.md" "$repo/docs/lookout-api.md"
}
