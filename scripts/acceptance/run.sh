#!/usr/bin/env bash
#
# The acceptance harness: one headless agent session against a repository that
# has `hyper` and nothing else (issue #216).
#
# Every acceptance transcript is evidence about what an agent can do with the
# surface `hyper` ships. That claim is only worth what the harness is worth,
# and the harness this replaces was a directory made by hand on a machine that
# also holds the `hyper` source checkout. Three of the runs recorded so far
# left the repository and read that checkout — `docs/spec/04-the-authoring-
# format.md`, `internal/*.go`, a solved `providers/` in a sibling directory —
# and a run that reads the specification and then writes a correct Manifest has
# been handed the answer. The transcript records a success the shipped product
# cannot reproduce, because on the machine a user installs `hyper` on there is
# no specification to read. Whether a given run forages is a property of the
# run and not of the setup, so the setup cannot be trusted to control for it:
# it has to be made impossible.
#
#   scripts/acceptance/run.sh <task-file> <output-directory>
#
# What that gets is a repository in the README's quickstart shape — the version
# pin, a Target, a Definition, a Procedure, and no `providers/` — with a Store
# initialised (ADR-0104), the MCP server wired, `AGENTS.md` written by
# `hyper project`, and a headless Claude Code session run inside a mount
# namespace whose home directory holds three paths and nothing else — the client
# running the session, the credential it authenticates with, and an onboarding
# flag. So the source checkout, the neighbouring checkouts, every cached copy of
# their text, this script's own output directory and whatever an attended session
# happened to leave in `$HOME` are none of them there to be read. The session's
# own transcript is the output.
#
# The seal is `bwrap(1)`, which needs no privilege and no daemon. It is not a
# security boundary and is not trying to be one: the sandboxed session runs as
# the same user against the same filesystem, and a determined process inside it
# is not being kept from anything. What it does is make the specification
# absent, which is the one property the evidence depends on.
#
# **Two things about `hyper` in here cannot be hidden, and the claim is written
# to say so** (issue #231). §9 states one transport — the server is the same
# binary, started by the client over stdio, one process per client, dying with
# it — and there is no `serve`, no daemon and no remote transport (ADR-0088).
# So `hyper mcp` is a
# *child of the sealed session*: `claude` reads `mcp.json` from inside the
# namespace and execs the binary that file names from inside it. Both stay
# reachable and no amount of binding changes that, which makes ADR-0099's *no
# `hyper` to invoke* true of `PATH` and of nothing else. The credential in
# `mcp.json` is reachable for the same reason and costs nothing: it is a
# fixture's, worth nothing outside the process that checks it (ADR-0105). The
# certificate that process trusts is bound back on a different ground — it
# *could* be hidden and the fixture would simply stop working, so it is kept
# deliberately, by the rule below rather than by necessity.
#
# **The client and its credential cannot be hidden either** (issue #257). `$HOME`
# goes wholesale now, and on this machine Claude Code is installed inside it — so
# the binary running the session and the credential it authenticates with are
# bound back there by name. A seal that hid them would be hiding the session, and
# neither is anything a task is about.
#
# So what this harness claims, and what the assertion below holds it to, is
# narrower than *nothing is reachable*: **no source checkout, no second binary,
# no fixture internals — and the one binary that is reachable is the one the MCP
# server is** (ADR-0109) — *binary* there meaning a `hyper`, the client being the
# session rather than something inside it. The gap is not an oversight and
# closing it is not available.
set -euo pipefail

task=${1:?usage: run.sh <task-file> <output-directory>}
outdir=${2:?usage: run.sh <task-file> <output-directory>}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
task=$(cd "$(dirname "$task")" && pwd)/$(basename "$task")
mkdir -p "$outdir"
outdir=$(cd "$outdir" && pwd)

[ -r "$task" ] || {
	echo "run.sh: $task is not a readable task file" >&2
	exit 2
}

# The output directory is written by the sealed session and read afterwards, so
# it cannot be inside the thing the seal hides — and what the seal hides is the
# checkout's *parent*, the sibling directories being where a solved `providers/`
# was found once already. A sibling of the checkout is the natural reading of
# "outside the checkout" and is the mistake this catches: the session would
# start with its own MCP configuration and repository missing.
hidden=$(dirname "$root")
case $outdir/ in
"$hidden"/*)
	echo "run.sh: $outdir is under $hidden, which the seal hides; put it somewhere else" >&2
	exit 2
	;;
esac
# `ACCEPTANCE_SETUP_ONLY` stops after the seal is asserted and before the
# session, which is the half a test can run: whether the repository this hands
# an agent checks clean, whether its `AGENTS.md` is the orientation the binary
# holds, and whether the seal is still covering the checkout are all questions
# with answers, and a harness nothing exercises is one that rots between the
# runs it is used for.
setup_only=${ACCEPTANCE_SETUP_ONLY:-}

tools=(bwrap git go python3)
[ -n "$setup_only" ] || tools+=(claude)
for tool in "${tools[@]}"; do
	command -v "$tool" >/dev/null || {
		echo "run.sh: $tool is not on PATH" >&2
		exit 2
	}
done

# The version the repository pins and the binary is stamped with. *Which*
# version is arbitrary — nothing here resolves a release — but the two halves
# agreeing is not: fifteen of the sixteen commands compare them for exact
# equality and Refuse `version-pin-mismatch` when they differ (§11), and a
# harness that let them drift would produce a transcript about the version gate
# rather than about the task. It is read off the README's install block so that
# what an agent sees in `hyper.yaml` is the version a reader would have
# installed; falling back costs nothing but the resemblance.
version=$(sed -n '0,/^VERSION=/s/^VERSION=//p' "$root/README.md")
version=${version:-0.0.2-alpha}

repo=$outdir/repo
# The two files a setup script may leave for this one to read go with the
# repository and the binaries, and for a sharper reason than tidiness: an
# output directory reused by a task that ships no service would otherwise fold a
# previous task's credential into `mcp.json` and kill a pid the machine has
# since given to something else.
rm -rf "$repo" "$outdir/bin"
rm -f "$outdir/endpoint.pid" "$outdir/endpoint.env"
mkdir -p "$repo/targets" "$repo/definitions" "$repo/procedures" "$outdir/bin"

# Built before the seal goes up, because the seal hides the source. The binary
# lands outside the repository and off the sandbox's PATH: issue #214's runs
# had no `hyper` to invoke and the MCP surface was the whole of what they had,
# which is the condition being repeated. `hyper mcp` is reached by the absolute
# path the MCP configuration names, and by nothing else.
go build -C "$root" \
	-ldflags "-X github.com/TheLoomLabs/hyper/internal/version.Version=$version" \
	-o "$outdir/bin/hyper" ./cmd/hyper

# The README's quickstart, verbatim but for the Target's `kinds:`. The tasks
# these transcripts are for reach the `destroy` corner, and a Target that
# granted `read, mutate` would have the agent meet an authority Refusal rather
# than the question being asked. `providers/` is absent, and that absence is
# the gap under test whenever the task is *author a Provider*.
cat >"$repo/hyper.yaml" <<YAML
kind: repository-declaration
version: $version
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
YAML
cat >"$repo/targets/local.yaml" <<'YAML'
kind: target-declaration
target: local
class: local
kinds: [read, mutate, destroy]
capabilities: [shell]
YAML
cat >"$repo/definitions/host-ops.yaml" <<'YAML'
kind: definition
definition: host-ops
provider: shell
kinds: [read]
targets: [local]
YAML
cat >"$repo/procedures/say-hello.yaml" <<'YAML'
kind: procedure
procedure: say-hello
targets: [local]
steps:
  - id: greet
    definition: host-ops
    operation: read
    target: local
    args:
      command: [echo, hello from hyper]
YAML

# A task that names something on the machine — a directory to archive, a file to
# read, an API to call — brings it with it, in a script beside it. A transcript
# that failed because a path was not there would be evidence about the path.
#
# **It takes the output directory as well as the repository, and that is what
# lets a task ship a service** (ADR-0105, issue #227). The repository is what the
# agent sees; the output directory is where anything the harness needs to know
# about it goes, and two files there are read by name afterwards:
# `endpoint.pid`, whose process this script kills on the way out, and
# `endpoint.env`, whose `NAME=value` lines are folded into the environment the
# MCP server runs with. That is the whole of the contract, and a task that needs
# neither writes neither.
#
# **The lifetime is this script's rather than the setup script's**, because the
# fence runs the setup half on every `go test ./cmd/hyper` and a service nobody
# stops is one process per test run. The trap is armed before the script runs, so
# a setup that fails halfway still takes its service with it; a `SIGKILL` of this
# script leaks one process, and the pidfile names it until the next run over this
# output directory clears it above.
trap 'if [ -s "$outdir/endpoint.pid" ]; then kill "$(cat "$outdir/endpoint.pid")" 2>/dev/null || true; fi' EXIT INT TERM
if [ -x "${task%.md}.setup.sh" ]; then
	"${task%.md}.setup.sh" "$repo" "$outdir"
fi

# `AGENTS.md` is the orientation's second channel (ADR-0095), and `project`
# writes it, so the harness runs `project` rather than reproducing the text. A
# copy kept in a file beside this script would be the wrong bytes the moment
# the orientation changes, which is the failure this avoids.
#
# **ADR-0095's *dormant until the first release* does not reach this fixture.**
# The Refusal it names turns on the pin rather than on the release: the pin
# here and the binary's stamp are one value by construction above, so
# `frozenDigest` returns the declared digest, resolves nothing, and reaches no
# network. Nothing here can Refuse `release-artefact-absent`.
#
# **Nothing forces the position either.** `project` walks up for a git root
# only where neither `--repo-dir` nor `HYPER_REPO_DIR` names one, so it would
# run after `git init` below just as well; what the position has to satisfy is
# `git add -A`, the note belonging to the commit the agent is handed. It sits
# after the task's setup script, and `project` is create-if-absent — a setup
# script writing an `AGENTS.md` of its own keeps it, and `cmd/hyper`'s case
# compares these bytes for every task, so a task that did would fail under its
# own name.
#
# The answer goes to stderr rather than `/dev/null`: `project` runs `check`
# first and a failing check *is* its answer, on stdout, so discarding it would
# stop this script at `set -e` with the table that says why thrown away. The
# `store init` line below discards its stdout safely because its faults are
# Refusals, and §8 puts a Refusal on stderr with stdout left silent.
HYPER_REPO_DIR=$repo "$outdir/bin/hyper" project >&2

# A Run's Provenance records `repo_revision`, so a repository with no commit
# has nothing to record. The Store is a branch, and `store init` Refuses on an
# unpinned repository — the pin above is what makes this line work.
git -C "$repo" init -b main -q
git -C "$repo" config user.name "acceptance"
git -C "$repo" config user.email "acceptance@localhost"
git -C "$repo" add -A
git -C "$repo" commit -q -m "the quickstart shape"

# **The fixture ships a Store, and that is a decision** (ADR-0104, issue #223).
# A repository with no Store refuses every Run `store-absent`, and the
# orientation puts `store init` on the far side of the line it draws with
# `install` and `compact`: *creating the record is the human's act; your part is
# to say it has not happened*. So an agent handed a Store-less repository and a
# task whose deliverable is a Run answers correctly by stopping at the first
# call, and the transcript reaches none of Execution, the Record, or the branch
# that holds them — §6, §7 and the Store, which is the whole of what a
# run-capable task exists to put in front of an agent. Keeping the wall and
# asking for a Run are not both available, and what removing it costs is the
# ADR's to say.
HYPER_REPO_DIR=$repo "$outdir/bin/hyper" store init >/dev/null

# Passed with `--strict-mcp-config`, so this file is the whole of the session's
# MCP configuration and nothing on the machine can add a second server to it.
# Local scope in `~/.claude.json` would do the same job and would be one more
# piece of state the seal has to reason about.
#
# **A task's `endpoint.env` lands here and nowhere else** (ADR-0105). It is where
# a fixture's credential and an `SSL_CERT_FILE` reach the `hyper` the server
# runs, and that position is the point rather than a convenience: no artefact
# carries a root, a pin or a verification mode, so trust is a property of the
# process and is unreachable from anywhere the agent writes. A missing file is
# the ordinary case and reads as no additions.
#
# **This is the only reading of `endpoint.env` there is**, and what it prints
# back is the values that are existing paths inside the output directory. The
# seal has to bind those through, the process that opens them being the sealed
# session's own child (issue #231) — and a second reading of this file in shell
# would be a second `strip()` and a second answer to *what is a line carrying no
# `=`*, with nothing keeping the two in agreement.
kept=$(python3 -c '
import json, os, sys

command, repo, additions, destination = sys.argv[1:]
environment = {"HYPER_REPO_DIR": repo}
supplied = []
try:
	with open(additions) as lines:
		for line in lines:
			name, separator, value = line.strip().partition("=")
			if separator:
				environment[name] = value
				supplied.append(value)
except FileNotFoundError:
	pass
with open(destination, "w") as configuration:
	json.dump({"mcpServers": {"hyper": {"command": command, "args": ["mcp"], "env": environment}}}, configuration)

# What the task supplied, and nothing else: `HYPER_REPO_DIR` is written here
# rather than read, and the repository it names is bound back by name.
directory = os.path.dirname(destination)
for value in supplied:
	if value.startswith(directory + os.sep) and os.path.exists(value):
		print(value)' \
	"$outdir/bin/hyper" "$repo" "$outdir/endpoint.env" "$outdir/mcp.json")

# What the seal covers, and why each one.
#
#   $HOME               **wholesale** (issue #257). A `--tmpfs`, with the three
#                       paths the sealed session's own processes must open bound
#                       back on top by name. Every entry below that names a path
#                       inside it is a consequence of this line rather than a
#                       cover of its own.
#   the checkout        the specification, the ADRs, `internal/*.go` — the thing
#                       three runs went and read. Its *parent* goes rather than
#                       the checkout alone, the sibling directories being where a
#                       solved `providers/` was found once already; the line is
#                       here for a checkout kept outside `$HOME`.
#   the build cache     a linked `bin/lookout` and the archives behind it: the
#                       fixture's compiled text, not the specification's. Same
#                       case — on this machine it is `~/.cache/go-build` and the
#                       first line has it.
#   the output          everything this script writes, but for the repository
#   directory           and the files `keep` names below, with reasons
#   previous output     the same list again, one run of this harness ago; found
#   directories         by search, since nothing here remembers where they went
#
# **The list used to run the other way, and that is what issue #257 closed.**
# Nine covers named the places this project's text was known to collect —
# `$HOME/bin`, five Claude Code caches, the prompt history, the build cache —
# and each of them was added when something turned out to be reachable. What a
# list of that shape cannot cover is the thing that grows: when the
# `monitor-retirement` run was prepared, `$HOME` held six directories of session
# material left by three earlier tickets — a working Provider Manifest and nine
# Runs, six raw transcripts quoting Manifests whole, the published
# `v0.0.1-alpha` archive, and the by-hand completion of the very task about to be
# run, an hour old. None of it matched any search here and none of it was ever
# going to: the next one is named something else.
#
# ADR-0099's rule is what condemns the shape rather than the entries — *whether a
# given run forages is a property of the run and not of the setup, so the setup
# cannot be trusted to control for it: it has to be made impossible* — and a list
# of what to hide is a setup being trusted to control for it. So the home
# directory goes and what the session needs comes back, which is the same
# inversion ADR-0109 made one directory in.
empty=$outdir/.empty
mkdir -p "$empty"
# The one fact about this machine the sealed session is handed rather than
# left to discover. Without it the first turns go on onboarding, and the
# transcript is about the client.
printf '{"hasCompletedOnboarding":true,"installMethod":"native","autoUpdates":false}\n' >"$outdir/.claude.json"

# `--clearenv`, because the harness is usually invoked from inside an agent
# session and that session's environment is not a fact about a user's machine.
# What leaks otherwise is `CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`,
# `CLAUDE_EFFORT` and a messaging socket and token — enough for the sealed
# session to be running as a child of the one that started it, at an effort
# level it did not choose. The first run of this harness leaked all of those.
seal=(
	bwrap --bind / /
	--clearenv
	--setenv HOME "$HOME" --setenv USER "${USER:-$(id -un)}" --setenv LOGNAME "${USER:-$(id -un)}"
	--setenv PATH /usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin
	--setenv LANG "${LANG:-C.UTF-8}" --setenv TERM "${TERM:-dumb}"
	--setenv DISABLE_AUTOUPDATER 1
)

# The rules the covers below are written in, stated once each.
#
# `cover` hides a path outside `$HOME`. Operands in `bwrap`'s own order —
# option, source, destination — so that the calls read against the man page
# rather than backwards from it. A destination this machine does not have is
# skipped, `bwrap` treating a missing destination as a hard error rather than
# something to create; and so is a destination inside `$HOME`, where an empty
# bind would not hide a path but **create** one — an empty `~/dev` in a home
# directory the tmpfs below has already emptied, and a line the inventory would
# then have to be taught to expect.
cover() {
	case $3/ in
	"$HOME"/*) return 0 ;;
	esac
	[ -e "$3" ] && seal+=("$1" "$2" "$3")
	return 0
}

# `keep` is one fact written once: a path the sealed session's own processes must
# open is bound back over a tmpfs, *and* is what the assertion below expects to
# find reachable there. Two lists would be two lists to keep in agreement. The
# directories on the way to a kept path come with it, `bwrap` creating them in
# the tmpfs as it binds, so `allow` records the whole chain rather than the path
# alone. The chain is what an output directory nested under `$HOME` needs: the
# walk below descends from `$HOME` to it and prints every directory on the way.
# Past `$HOME` it keeps walking to `/`, which is looser than it needs to be and
# costs nothing — those are ancestors of a walk's own root, which `find` never
# prints, so nothing is ever compared against them.
#
# **`keep` is strict, unlike `cover`, and the difference is whose path it is.**
# Everything handed to `keep` here is one this script created a few lines above —
# the `.claude.json`, the repository, `bin/hyper`, `mcp.json`, and a path an
# `endpoint.env` named that the parser above already checked exists. A missing
# one is a defect in this script, and `bwrap`'s hard error is the right report
# for it: a silent skip would be a sealed session started without its repository.
# The two paths that belong to the *machine* rather than to this script are
# guarded where they are named, below.
allowed=()
allow() {
	local path=$1
	while [ "$path" != / ]; do
		allowed+=("$path")
		case $path in
		"$HOME") break ;;
		esac
		path=$(dirname "$path")
	done
}
keep() {
	local source=$2 destination=${3:-$2}
	seal+=("$1" "$source" "$destination")
	allow "$destination"
}

# **`$HOME` goes whole, and three things come back** (issue #257).
#
# Two of them are the session itself and cannot be anything else, for the reason
# `mcp.json` and `bin/hyper` cannot (ADR-0109): a seal that hid them would hide
# the thing being measured. Claude Code is installed under `$HOME` on this
# machine — `~/.local/bin/claude`, a symlink `bwrap` resolves as it binds, so
# what lands at that path is the binary itself — and its credential is what lets
# the session authenticate at all. The path is read once and used twice: the
# client bound here is the one `bwrap` execs at the bottom of this script. Neither is new inside the seal; what is new is
# that they are named rather than merely surviving a list of what to hide.
# Neither is anything the task is about, and the credential is worth exactly what
# it was worth before, the seal never having been a security boundary.
#
# The third is the onboarding flag above, which is a fact about the client and
# not about the machine.
#
# **What is deliberately not bound back is everything the old list named.** The
# Claude caches, the prompt history and its backups, `$HOME/bin`, the build
# cache: all of them are inside `$HOME` and are gone by construction rather than
# by nine lines that have to stay abreast of what Claude Code writes and what a
# session leaves behind. `~/.claude/settings.json` was covered with `{}` and is
# now simply absent, which is the same absence of hooks, plugins and skills;
# `~/.claude/history.jsonl` and `~/.claude/projects` were bound to files in the
# output directory and are now the tmpfs's, which is what ADR-0111 wanted — a
# sealed session's auto-memory write lands in a home directory that dies with the
# namespace, so reusing an output directory can no longer carry one session's
# notes into the next.
#
# **Both machine paths are guarded, and neither is a fact this script can assume.**
# A client kept outside `$HOME` needs no bind and gets none; a machine with no
# client and no credential at all — the runner that runs the setup half, where
# `ACCEPTANCE_SETUP_ONLY` stops before the session — keeps neither, and a session
# it cannot start is not one it is starting. On a machine where the session *is*
# about to run, a credential that is not here is a session that cannot
# authenticate, and the report for that is the empty transcript at the bottom of
# this script.
#
# **The credential is bound writable.** Claude Code refreshes its own token, and
# a refresh it cannot persist would turn a long run into a transcript about
# authentication. The writer is the same client doing the same thing it does
# outside the seal, and what it writes is the machine's own credential.
seal+=(--tmpfs "$HOME")
client=$(command -v claude || true)
case $client/ in
"$HOME"/*) keep --ro-bind "$client" ;;
esac
credential=$HOME/.claude/.credentials.json
if [ -e "$credential" ]; then
	keep --bind "$credential"
fi
keep --bind "$outdir/.claude.json" "$HOME/.claude.json"

# The checkout, where it is kept outside `$HOME`. The parent of it goes rather
# than the checkout alone: the sibling directories are where a solved
# `providers/` was found once already.
cover --ro-bind "$empty" "$(dirname "$root")"

# Go's build cache, on the same terms, and the reason is the fixture rather than
# the specification (issue #231). It holds no source — compiled archives, linked
# binaries and cached `go test` output — but one of those linked binaries is
# `bin/lookout`, built minutes earlier by a setup script, and hiding the
# fixture's binary in the output directory while leaving a copy of it here would
# be the same hole one `find` further away. The 2026-08-29 run's `find` over
# `$HOME` reached this directory. The module cache is left alone: it holds
# third-party source and nothing of this project's text.
cover --ro-bind "$empty" "$(go env GOCACHE)"

# **A previous run's output directory is covered too, and it has to be found
# rather than known** (issue #231). The cover below is over the directory this
# script was handed; a machine that has run this harness before has others, and
# `/home/idabic/acceptance-217` and `-227` were both sitting in `$HOME` when this
# was written, each holding a `bin/lookout`, an `endpoint.env` and a transcript.
# Covering one and leaving the rest is the mistake that covering the checkout's
# *parent* rather than the checkout alone already answers, and a harness whose
# every run left the next one a directory to read would be a hole that widens
# with use.
#
# A harness output directory is an `mcp.json` naming `HYPER_REPO_DIR`, exactly as
# a checkout is a `go.mod` naming this module — the thing is looked for rather
# than a list of paths trusted. The search runs **here, outside the namespace**,
# where what it finds can still be covered; the same search runs inside as the
# assertion, where it could only complain. This run's own directory is skipped,
# and so is any directory this run's output is inside of.
#
# **`$HOME` is no longer one of the roots** (issue #257). An output directory
# kept there is covered by the tmpfs above, so searching for one would buy a
# `cover` call that `cover` now declines — and the walk was the expensive half of
# a fence that runs on every `go test ./cmd/hyper`, once per task.
while IFS= read -r configuration; do
	previous=$(dirname "$configuration")
	case $outdir/ in
	"$previous"/*) continue ;;
	esac
	cover --ro-bind "$empty" "$previous"
done < <(find /opt /srv /var/tmp -name mcp.json -readable \
	-exec grep -l "HYPER_REPO_DIR" {} + 2>/dev/null)

# **The output directory is covered too, and it is the one this script writes
# itself** (issue #231, ADR-0106). `bin/lookout`, whose strings are the
# fixture's answer key — the seeded monitors, the page size, every code it
# refuses with — sits here, beside `endpoint.env` and `lookout.report` carrying
# the fixture's credential in cleartext, both logs, and the transcript being
# written as the session runs. The path is not obscure: `mcp.json` names the
# binary's absolute path and the repository the session works in is
# `$outdir/repo`, so `..` is the whole of the discovery.
#
# A `--tmpfs` rather than an empty bind, because the repository lives *inside*
# this directory and has to come back on top of it. The binds above take their
# sources from here and keep working: `bwrap` resolves a source against the old
# root, so a tmpfs over the destination side does not hide `$outdir/.empty` or
# `$outdir/.claude.json` from the operands naming them. The transcript needs no
# reachable path — it is written through a redirect this shell opens before
# `bwrap` runs.
#
# **The cover is `allow`ed rather than `keep`t**, because it is a tmpfs and
# not a bind: where the output directory sits inside `$HOME` this line creates
# the path in the home directory's tmpfs, and the walk below meets it and the
# chain down to it exactly as it meets a kept path's.
seal+=(--tmpfs "$outdir")
allow "$outdir"

keep --bind "$repo"
keep --ro-bind "$outdir/bin/hyper"
keep --ro-bind "$outdir/mcp.json"

# **A task's `endpoint.env` may name a file rather than only a value**, and
# `SSL_CERT_FILE` is the case that exists (ADR-0105). The general rule is the one
# worth holding: the MCP server is a child of the sealed session, so a path this
# file hands it has to survive the cover — and the harness learns which path from
# the contract it already reads rather than from a task's filename hardcoded here.
# `$kept` is that reading, done once where the configuration was written.
if [ -n "$kept" ]; then
	while IFS= read -r path; do
		keep --ro-bind "$path"
	done <<<"$kept"
fi

seal+=(--proc /proc --dev /dev --die-with-parent --chdir "$repo")

# The conditions are asserted rather than assumed, and asserted by looking for
# the thing rather than by trusting the lists above. A checkout of `hyper` is
# a `go.mod` naming its module path, so that is what is searched for — under the
# home directory and the handful of places a second checkout is plausibly kept.
# A `hyper` on `PATH` is the other condition issue #214's runs had, and an empty
# `$HOME` is not the same fact as there being none.
#
# The three run as one walk rather than three: they share a root list and the
# `-name` tests are disjoint, so a second pass over `$HOME` would buy nothing but
# the seconds this case pays on every `go test ./cmd/hyper`. Since issue #257
# that leg walks a tmpfs holding half a dozen paths, so what it costs is nothing
# and what it says is that the covers came back empty-handed.
#
# **Two directories are inventoried, and the inventory is the half of this that
# survives a list going stale** (issue #231, issue #257).
#
# `$HOME` and the output directory are each a tmpfs with a `keep` list bound back
# on top, and one walk asserts both: it prints everything reachable under them
# with the repository pruned, and what `keep` and `allow` recorded is the
# whole of what may come back. A list of forbidden names — a fourth `-name`
# beside the three above — goes stale the first time a task leaves a file nobody
# thought of, or the first time session material is kept under a name nobody has
# used yet, which is how `$HOME` came to be uncovered for every sealed run so
# far. An inventory does not go stale. It goes noisy, and a name it did not
# expect is the operator's to explain.
#
# **A previous run's directory is asserted by name**, because it is covered by a
# search rather than by a path and a search that quietly matched nothing would
# be a cover that quietly covered nothing. So the same `mcp.json` rule runs
# again in here, and beside it the fixture's binary as a regular file called
# `lookout` — which catches a copy that is *not* in an output directory and so
# was never covered at all. Either one is fatal: the operator moves it, deletes
# it, or finds out why the cover above missed it.
#
# **This run's own `mcp.json` matches that search**, and is not a finding: it is
# on `keep`'s list, and the filter below drops everything on that list before
# anything is concluded. `endpoint.env` is deliberately *not* searched for by
# name — it is too ordinary a filename to fire on only this harness's copies
# (this machine has one under `~/.config`), and every copy of it that matters
# sits in a directory the `mcp.json` rule already names.
#
# **The sentinel is what makes this an assertion.** Without it an empty answer
# reads as *sealed* whether the search found nothing or never ran, and a `bwrap`
# that could not build a namespace would be reported as the strongest result the
# harness has. Nothing is concluded from silence: the search prints `SEALED` on
# a line of its own once it has actually run, and the absence of that line is a
# failure rather than a pass. The inventory is the one search whose own failure
# would be silence too — a covered directory is readable to us or it is nothing
# — so it takes the sentinel with it on the way out rather than hiding its
# errors, which is the `|| exit 1` and the `2>/dev/null` the other three have and
# it does not.
report=$("${seal[@]}" /bin/sh -c '
	find "$HOME" /opt /srv /var/tmp \
		\( -name go.mod -readable -exec grep -l "^module github.com/TheLoomLabs/hyper$" {} + \) -o \
		\( -name mcp.json -readable -exec grep -l "HYPER_REPO_DIR" {} + \) -o \
		\( -type f -name lookout -print \) 2>/dev/null
	find "$HOME" "$1" -mindepth 1 -path "$2" -prune -o -print || exit 1
	command -v hyper 2>/dev/null
	echo SEALED
' sh "$outdir" "$repo" || true)
if ! printf '%s\n' "$report" | grep -qx SEALED; then
	echo "run.sh: the seal could not be built — the search inside it never ran" >&2
	exit 2
fi
if reachable=$(printf '%s\n' "$report" | grep -vx SEALED |
	grep -vxF -f <(printf '%s\n' "${allowed[@]}") | grep .); then
	echo "run.sh: the seal is not holding — this is reachable inside it:" >&2
	echo "$reachable" >&2
	exit 2
fi

if [ -n "$setup_only" ]; then
	echo "repository: $repo"
	exit 0
fi

# `stream-json` because the question these transcripts answer is which tools
# were called, in which order, against which paths — and `--verbose` is what
# makes print mode emit every message rather than the last one. The turn cap is
# well above the 180 calls of the run being repeated; it is there so a session
# that has stopped making progress stops rather than runs all night.
status=0
"${seal[@]}" "$client" \
	--print --verbose --output-format stream-json \
	--strict-mcp-config --mcp-config "$outdir/mcp.json" \
	--permission-mode bypassPermissions \
	--max-turns 220 \
	"$(cat "$task")" >"$outdir/transcript.jsonl" || status=$?

# **The session's exit is not the harness's.** A session that hits the turn cap
# exits non-zero and its transcript is the evidence anyway — a run that stopped
# making progress is a result and quite possibly the interesting one. What is
# not a result is an empty file, which is `claude` or `bwrap` failing to start,
# and reporting the path to one as though it were a transcript is how a harness
# tells you it measured something when it measured nothing.
if [ ! -s "$outdir/transcript.jsonl" ]; then
	echo "run.sh: the session wrote no transcript (exit $status)" >&2
	exit 2
fi
[ "$status" -eq 0 ] || echo "run.sh: the session exited $status; the transcript below is still the record" >&2

echo "transcript: $outdir/transcript.jsonl"
echo "repository: $repo"
