# The seal covers the output directory the harness writes

**`scripts/acceptance/run.sh` puts a `--tmpfs` over its own output directory and binds four things
back on top of it: the repository, `bin/hyper`, `mcp.json`, and any path a task's `endpoint.env`
names inside that directory.** Everything else it writes there — `bin/lookout`, `endpoint.env`,
`lookout.report`, both logs and the transcript being written as the session runs — is not reachable
from inside the namespace. Go's build cache is covered beside it, and so is **every output directory
a previous run left on the machine**, found by searching for the `mcp.json` this harness writes. This
closes the hole ADR-0106 recorded and issue #231 booked.

_ADR-0130 amends this:_ `$HOME` is covered the same way, for the same reason one directory further
out — a tmpfs, with the client, its credential and the onboarding flag bound back on top (issue
#257). That retires the cover list this decision extended, and with it the two lines below that name
`$HOME/bin` and `~/.claude/projects` as covers of their own.

**The two things that stay were never buyable.** §9 states one transport — *the server is the same
binary, started by the client over stdio: one process per client, dying with it* — and there is no
`serve`, no daemon and no remote transport (ADR-0088). So `hyper mcp` is a **child of the sealed
session**: `claude` reads `mcp.json` from inside the namespace and execs the binary that file names
from inside it. The credential in `mcp.json` and the certificate `SSL_CERT_FILE` points at are
reachable for the same reason and cost nothing — both are a fixture's, worth nothing outside the
process that checks them (ADR-0105, ADR-0007).

So the claim `run.sh` now states of itself is narrower than the one ADR-0099 left standing, and it is
the honest one: **no source checkout, no second binary, no fixture internals — and the one binary
that is reachable is the one the MCP server is.**

## What was in there, and why *nothing was foraged* is not the answer

The 2026-08-29 `monitor-coverage` run (ADR-0106) ran `ls -a` over the output directory looking for
where Records live, got back the whole list, read none of it, and moved on. That `ls` and a `find`
over `$HOME` are the whole of the foraging in a fifty-two-call run.

What it could have read is the point. `bin/lookout` is a compiled copy of the fixture's API, and its
strings are the answer key: the seeded monitors, the page size at two, the window bounds, every code
it refuses with — every one of which the task exists to measure an author discovering from the
documentation and the wire. Three files carry the fixture's credential in cleartext, one of them
`mcp.json`, which stays for the reason above and costs nothing by staying. The transcript is the
session's own record of itself.

ADR-0099 settled the rule that governs this and it applies unchanged: *whether a given run forages is
a property of the run and not of the setup, so the setup cannot be trusted to control for it: it has
to be made impossible.* A transcript in which a session read `bin/lookout`'s strings would be
evidence about an agent handed the answer key, and it would be found by **reading the transcript
afterwards** — which is the position ADR-0099 spent a mount namespace to get out of. The seal was
built against the checkout because that is where the specification is; the output directory was
created by the same script and covered by nothing.

The path was not obscure either. `mcp.json` names the binary's absolute path, and the repository the
session works in is `$outdir/repo` — so `..` is the whole of the discovery.

## A tmpfs, because the repository lives inside the thing being covered

The other covers in this script are an empty directory bound over a path. That shape does not work
here: the repository the session is *supposed* to be working in is a subdirectory of the thing to
cover. A `--tmpfs` does, and the four exceptions bind back on top of it.

Three behaviours this rests on were checked rather than assumed, `run.sh`'s existing `cover` helper
existing because `bwrap` treats a missing destination as a hard error rather than something to
create:

- **A bind destination inside a tmpfs is created**, parents included, so `$outdir/bin/hyper` needs no
  `bin/` prepared for it.
- **A bind source is resolved against the old root**, so the seal's other operands — `$outdir/.empty`
  bound over `$HOME/bin`, `$outdir/.claude.json` over `~/.claude.json`, `$outdir/projects` over
  `~/.claude/projects` — keep working with a tmpfs over the directory they are read from.
- **The transcript needs no reachable path.** It is written through a redirect the parent shell opens
  before `bwrap` runs.

## What a task may leave here, and the one rule that keeps it hidden

ADR-0105 gave a task two files it may write for `run.sh` to read afterwards: `endpoint.pid` and
`endpoint.env`. Both are read by the harness, outside the namespace, and neither needs to be
reachable inside it.

What can need to be reachable is a **path** one of them names. `SSL_CERT_FILE=$outdir/lookout.pem` is
the case that exists, and the process that opens it is the MCP server, which is a child of the sealed
session. So the rule is stated where the contract already is: **a value in `endpoint.env` that is a
path inside the output directory is bound back read-only, and nothing else a task writes there is.**
The harness learns which file that is from the contract it already reads rather than from a task's
filename written into the seal — `lookout.pem` hardcoded in `run.sh` would be the second task's bug.

**And it is read once.** `endpoint.env` already had one parser, the Python that folds it into
`mcp.json`, so the seal takes the paths from that parser rather than adding a shell one beside it.
Two readers of one file is two answers to *what does a trailing space do* and *what is a line
carrying no `=`* — and the failure they compose into is silent: a server handed an `SSL_CERT_FILE`
the seal did not bind, which is a TLS error inside a sealed session and a transcript about a
certificate.

## A previous run's directory is the same hole, and it is found rather than known

The cover above is over the directory the script was handed. A machine that has run this harness
before has others: `/home/idabic/acceptance-217` and `/home/idabic/acceptance-227` were both sitting
in `$HOME` when this was written, each with a `bin/lookout`, an `endpoint.env` and a transcript in it,
and `acceptance-227` is the very directory the 2026-08-29 run listed. Covering one and leaving the
rest is the mistake that covering the checkout's *parent* rather than the checkout alone already
answers — and worse, it is a hole that widens with use: every run of the harness would leave the next
one a directory to read.

**A harness output directory is an `mcp.json` naming `HYPER_REPO_DIR`**, exactly as a checkout is a
`go.mod` naming this module. The search runs on the host, before the seal is built, so that what it
finds can be *covered* rather than merely complained about — this run's own directory skipped, and
any directory this run's output sits inside skipped with it. _ADR-0130 amends this:_ the search no
longer walks `$HOME`, an output directory kept there being covered wholesale; `/opt`, `/srv` and
`/var/tmp` are what is left of the roots. `endpoint.env` is not searched for by
name anywhere: it is too ordinary a filename to fire on only this harness's copies, this machine
carrying an unrelated one under `~/.config`, and every copy that matters is in a directory the
`mcp.json` rule already names.

**Covering rather than refusing is the whole point.** A harness that failed on finding a previous
run's directory would make the operator delete the transcripts this project keeps as evidence, and
would turn every acceptance run into the next run's broken test.

## The assertion is an inventory *and* a list of names

Issue #231 asked for the fixture's binary and its environment file to join the search that already
runs inside the namespace. Both forms are here, because they catch different things.

**For this run's directory, an inventory.** The search prints everything reachable under it with the
repository pruned, and what was bound back is the whole of what may come back. A list of forbidden
names would go stale the first time a task leaves a file nobody thought of, and ADR-0099 built this
assertion precisely so that *the list going stale* is survivable. The bind and the expectation are
one statement in the script (`keep`), because two lists are two lists to keep in agreement.

**For everything outside it, the names.** An inventory of one directory says nothing about a previous
run's, and the cover for those is a *search* rather than a path — a search that quietly matched
nothing would be a cover that quietly covered nothing. So the `mcp.json` rule runs again inside the
namespace, and beside it `lookout` as a regular file, which catches a copy that is in no output
directory and so was covered by nothing. This run's own `mcp.json` matches and is not a finding: it
is on `keep`'s list, and the list is subtracted before anything is concluded.

The three searches share one walk of `$HOME`. Their `-name` tests are disjoint, and a second pass
would buy nothing but the seconds the fence pays on every `go test ./cmd/hyper`.

## Go's build cache is covered, and it is the fixture's problem rather than the specification's

Issue #231 asked for this to be decided either way. It is covered.

The cache holds no source. What it holds is compiled archives, cached `go test` output, and **linked
binaries** — and one of those linked binaries is `bin/lookout`, put there minutes earlier by the same
`go build` the setup script runs. Hiding the fixture's binary in the output directory while leaving a
copy of it one `find` away would be a fix that reads as one. The 2026-08-29 run's `find` over `$HOME`
reached this directory.

The module cache stays. It holds third-party source and nothing of this project's text, and it is not
what the seal is about.

## What was considered

**An empty directory bound over the output directory, like every other cover.** Not available: the
repository is inside it, and covering the directory would cover the thing the session is handed.

**Moving the repository out of the output directory instead.** That is a bigger change to a script
whose two-argument shape is documented in CONTRIBUTING, in ADR-0105's contract and in every task's
setup script, and it buys nothing the tmpfs does not: the other files would still need covering.

**Hardcoding `lookout.pem` in the seal**, as issue #231's sketch had it. Refused: it is one task's
filename in the generic harness, and the next task shipping a file its server must read would find
the seal silently wrong rather than loudly wrong.

**The names in the assertion instead of the inventory**, which is what the ticket asked for. Taken as
well rather than instead: the inventory is stronger inside the one directory and blind everywhere
else.

**Refusing to run when a previous run's directory is reachable**, rather than covering it. Refused
above: it would make the operator delete this project's own evidence, and every acceptance run would
break the next one's test.

**Leaving the build cache**, on the ground that it holds compiled artefacts of the binary under test
and that binary is reachable anyway. Refused: `bin/lookout` is not the binary under test.

## Consequences

- **The claim is written in `run.sh`'s own header**, including *why* `bin/hyper` and `mcp.json` cannot
  be hidden, so the next reader does not take the gap for an oversight or try to close it.
- **ADR-0099's list of what the seal covers is amended** rather than corrected: this is a wider seal,
  not a fact that was wrong when it was written.
- **`TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse` fences it for every
  task.** The assertion is `run.sh`'s own and the case runs the script to completion, so a task that
  leaves a new file in the output directory and a seal that stops holding both fail under the task's
  name. The setup half itself runs in no namespace, which is what lets the repository the fence then
  checks be the one the session was handed.
- **A sealed session can still write into the output directory**, the tmpfs being writable, and what
  it writes goes nowhere: the namespace dies with the session and the harness reads the host's copy.
  Nothing depends on this either way; it is recorded so that a reader does not mistake it for a leak.
- **No transcript collected before this is invalidated.** The 2026-08-29 run listed the directory and
  read nothing from it, which is the whole of the exposure on the record. What changes is that the
  next one cannot.
