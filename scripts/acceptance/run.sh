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
# pin, a Target, a Definition, a Procedure, and no `providers/` — with the MCP
# server wired, `AGENTS.md` written from the server's own handshake, and a
# headless Claude Code session run inside a mount namespace where the source
# checkout, the neighbouring checkouts, and every cached copy of their text are
# not there to be read. The session's own transcript is the output.
#
# The seal is `bwrap(1)`, which needs no privilege and no daemon. It is not a
# security boundary and is not trying to be one: the sandboxed session runs as
# the same user against the same filesystem, and a determined process inside it
# is not being kept from anything. What it does is make the specification
# absent, which is the one property the evidence depends on.
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
version=${version:-0.0.1-alpha}

repo=$outdir/repo
rm -rf "$repo" "$outdir/bin"
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
# read — brings it with it, in a script beside it. A transcript that failed
# because a path was not there would be evidence about the path.
if [ -x "${task%.md}.setup.sh" ]; then
	"${task%.md}.setup.sh" "$repo"
fi

# `AGENTS.md` is the orientation's second channel (ADR-0095), and `project`
# writes it — except that `project` resolves the release it would freeze a
# digest of and Refuses `release-artefact-absent` while none is published. So
# the text is taken from the other channel, which is the same text: `writeNote`
# writes `mcp.Instructions(version)` and nothing else. A copy kept in a file
# beside this script would be the wrong bytes the moment the orientation
# changes, which is the failure this avoids.
python3 - "$outdir/bin/hyper" "$repo" >"$repo/AGENTS.md" <<'PY'
import json, os, subprocess, sys

binary, repo = sys.argv[1], sys.argv[2]
server = subprocess.Popen(
    [binary, "mcp"],
    stdin=subprocess.PIPE, stdout=subprocess.PIPE,
    env={**os.environ, "HYPER_REPO_DIR": repo}, text=True,
)
server.stdin.write(json.dumps({
    "jsonrpc": "2.0", "id": 1, "method": "initialize",
    "params": {
        "protocolVersion": "2025-06-18", "capabilities": {},
        "clientInfo": {"name": "acceptance", "version": "0"},
    },
}) + "\n")
server.stdin.flush()
sys.stdout.write(json.loads(server.stdout.readline())["result"]["instructions"])
server.stdin.close()
server.wait(timeout=10)
PY

# A Run's Provenance records `repo_revision`, so a repository with no commit
# has nothing to record. The Store is a branch, and `store init` Refuses on an
# unpinned repository — the pin above is what makes this line work.
git -C "$repo" init -b main -q
git -C "$repo" config user.name "acceptance"
git -C "$repo" config user.email "acceptance@localhost"
git -C "$repo" add -A
git -C "$repo" commit -q -m "the quickstart shape"
HYPER_REPO_DIR=$repo "$outdir/bin/hyper" store init >/dev/null

# Passed with `--strict-mcp-config`, so this file is the whole of the session's
# MCP configuration and nothing on the machine can add a second server to it.
# Local scope in `~/.claude.json` would do the same job and would be one more
# piece of state the seal has to reason about.
python3 -c 'import json,sys; json.dump({"mcpServers": {"hyper": {"command": sys.argv[1], "args": ["mcp"], "env": {"HYPER_REPO_DIR": sys.argv[2]}}}}, sys.stdout)' \
	"$outdir/bin/hyper" "$repo" >"$outdir/mcp.json"

# What the seal covers, and why each one.
#
#   the checkout        the specification, the ADRs, `internal/*.go` — the
#                       thing three runs went and read
#   $HOME/bin           where a stamped `hyper` is kept on this machine; the
#                       condition being repeated is that there is none on PATH
#   ~/.claude/projects  every session transcript on this machine, which for
#                       this project quote the specification at length
#   history.jsonl,      the same text again, in the caches Claude Code keeps
#   file-history,       beside it
#   shell-snapshots,
#   session-env
#   ~/.claude.json      per-project prompt history, same reason
#   settings.json,      this machine's hooks, plugins and skills, which are
#   plugins, skills     configuration a user installing `hyper` does not have,
#                       and would be a second uncontrolled variable
#
# The parent of the checkout goes rather than the checkout alone: the sibling
# directories are where a solved `providers/` was found once already.
empty=$outdir/.empty
mkdir -p "$empty" "$outdir/projects"
printf '{"hasCompletedOnboarding":true,"installMethod":"native","autoUpdates":false}\n' >"$outdir/.claude.json"
printf '{}\n' >"$outdir/.settings.json"
: >"$outdir/.history.jsonl"

# `--clearenv`, because the harness is usually invoked from inside an agent
# session and that session's environment is not a fact about a user's machine.
# What leaks otherwise is `CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`,
# `CLAUDE_EFFORT` and a messaging socket and token — enough for the sealed
# session to be running as a child of the one that started it, at an effort
# level it did not choose. The first run of this harness leaked all of those.
seal=(
	bwrap --bind / / --ro-bind "$empty" "$(dirname "$root")"
	--clearenv
	--setenv HOME "$HOME" --setenv USER "${USER:-$(id -un)}" --setenv LOGNAME "${USER:-$(id -un)}"
	--setenv PATH /usr/local/bin:/usr/bin:/bin:/usr/local/sbin:/usr/sbin:/sbin
	--setenv LANG "${LANG:-C.UTF-8}" --setenv TERM "${TERM:-dumb}"
	--setenv DISABLE_AUTOUPDATER 1
)
# Operands in `bwrap`'s own order — option, source, destination — so that this
# list reads against the man page rather than backwards from it. A destination
# this machine does not have is skipped: there is no history to hide in a
# `~/.claude` that has none, and `bwrap` treats a missing destination as a hard
# error rather than something to create.
cover() {
	[ -e "$3" ] && seal+=("$1" "$2" "$3")
	return 0
}
cover --ro-bind "$empty" "$HOME/bin"
cover --ro-bind "$empty" "$HOME/.claude/file-history"
cover --ro-bind "$empty" "$HOME/.claude/shell-snapshots"
cover --ro-bind "$empty" "$HOME/.claude/session-env"
cover --ro-bind "$empty" "$HOME/.claude/plugins"
cover --ro-bind "$empty" "$HOME/.claude/skills"
cover --ro-bind "$outdir/.settings.json" "$HOME/.claude/settings.json"
cover --bind "$outdir/.history.jsonl" "$HOME/.claude/history.jsonl"
cover --bind "$outdir/.claude.json" "$HOME/.claude.json"
cover --bind "$outdir/projects" "$HOME/.claude/projects"
seal+=(--proc /proc --dev /dev --die-with-parent --chdir "$repo")

# The two conditions are asserted rather than assumed, and asserted by looking
# for the thing rather than by trusting the list above. A checkout of `hyper` is
# a `go.mod` naming its module path, so that is what is searched for — under the
# home directory and the handful of places a second checkout is plausibly kept.
# A `hyper` on `PATH` is the other condition issue #214's runs had, and covering
# `$HOME/bin` is not the same fact as there being none.
#
# **The sentinel is what makes this an assertion.** Without it an empty answer
# reads as *sealed* whether the search found nothing or never ran, and a `bwrap`
# that could not build a namespace would be reported as the strongest result the
# harness has. Nothing is concluded from silence: the search prints `SEALED` on
# a line of its own once it has actually run, and the absence of that line is a
# failure rather than a pass.
report=$("${seal[@]}" /bin/sh -c '
	find "$HOME" /opt /srv /var/tmp -name go.mod -readable \
		-exec grep -l "^module github.com/TheLoomLabs/hyper$" {} + 2>/dev/null
	command -v hyper 2>/dev/null
	echo SEALED
' || true)
if ! printf '%s\n' "$report" | grep -qx SEALED; then
	echo "run.sh: the seal could not be built — the search inside it never ran" >&2
	exit 2
fi
if reachable=$(printf '%s\n' "$report" | grep -vx SEALED | grep .); then
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
"${seal[@]}" "$(command -v claude)" \
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
