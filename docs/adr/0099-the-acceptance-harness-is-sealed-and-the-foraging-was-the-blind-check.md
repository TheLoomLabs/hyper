# The acceptance harness is sealed, and the foraging was the blind `check`

**Every acceptance transcript is now run inside a mount namespace where no `hyper` source checkout
exists.** The harness is a script in the repository (`scripts/acceptance/run.sh`) rather than a
directory made by hand, it asserts the seal instead of assuming it, and a case runs its setup half so
that it cannot rot between the handful of runs a year it is used for.

**Nothing else changes.** The orientation is untouched, no fourth channel is built, and no `check`
message gains a sentence — because the run this decision is made on shows the behaviour it was opened
about is gone, and gone for a reason already fixed.

## The evidence: thirty calls, and the repository is never left

A Claude Code session, headless, 2026-08-28, `claude-opus-5`, inside the seal. Issue #214's
conditions repeated after issue #214's fix: a repository in the README's quickstart shape with
`providers/` absent, the MCP server wired and **no `hyper` on `PATH`**, and the same class of task —
one Procedure writing gzipped tars of three log directories as Assets, a second retiring them without
reaching anything `hyper` did not create.

**Seven `check` calls against seventy-six**, which is the count ADR-0097 asked to be measured, and
three of the seven report problems — two, fourteen and one — with an edit against the rows after
each. Thirty tool calls against a hundred and eighty is the headline and is the weaker half of it:
the baseline's are seventy-three `Write` calls where this run writes several artefacts per `Bash`
heredoc, so the totals are not like for like. The `check` ratio is, and so is the outcome.

**What is gone is the oscillation, not the probing.** This run still writes throwaway artefacts —
an Extension whose `destroy` request it hoped would be boundable, then a widened Target under
`# temporary experiment, reverted below` — and that is an agent confirming a rule rather than
guessing at one. The `check` beneath each names the rule (`capability-reserved`, `schema-mismatch:
expected a boolean`), which is the difference from the baseline: there, six throwaway files with a
`zzz_bogus_key: 1` control were the only way to make the count move. A probe answered by a row is a
question; a probe answered by a count is a binary search.

**The repository is never left.** No `strings`, no `ls` of a source tree, no `docs/spec/`, no
`internal/*.go`, no sibling checkout — and none of it because the seal refused: the one path outside
the repository in the whole transcript is a backup the session wrote under `/tmp` before editing a
file of its own. It read `AGENTS.md` on its third tool call, unprompted, and then asked the surface: `providers`,
`provider`, `targets`, and `operation` once for each of the six Operations `shell` exposes.

It also **finished**. The snapshot Procedure was authored and checks clean. The retire Procedure it
declined to ship, reporting back the three codes that stopped it — `opaque-destroy-not-granted`,
`bound-illegal` and, for the Extension it tried authoring to get a boundable `destroy`,
`capability-reserved`. It widened `targets/local.yaml` twice to see what `check` said and reverted
both times, handing back a repository whose Target grants exactly what it was given and the argument
for widening it: opting a Target into `opaque-destroy:` is the human's call. That is §6 working, read
off the surface by an agent that had never seen §6. The baseline for that task is *zero working
Procedures*.

## Which reading the evidence supports, and which one loses

Issue #216 posed two, with opposite fixes.

**A discovery gap** — the agent forages because it does not know `operation` hands it the Manifest's
own lines. **Refuted.** It called `operation` six times, exhaustively, before writing anything, and
what came back was the `source` rows: the Manifest's lines verbatim, which is what it then wrote
against.

**A content gap** — the answer is not on the surface. **Refuted for this task.** Every fact it needed
was in reach, and the three rules it could not have inferred were the three `check` named to it.

**The winning reading is the confound.** The foraging was the blind `check` and nothing else. An
agent told *6 problems* seventy-six times and not which six had a rational reason to go and read the
schema; an agent told the file, line, field, `error_code` and message has none, and does not. What
loses is *the orientation is short of a channel* — the fourth channel issue #216 sized up, whether in
the orientation text, a tool `description`, or a `check` message, is not built, and the 13,191
characters issue #211 cut stay cut.

## The seal, and what it is not

`bwrap(1)`: the filesystem bound at `/`, then an empty directory bound over the parent of the
checkout, over `$HOME/bin`, and over every cache that holds this project's text — the session
transcripts under `~/.claude/projects`, the prompt history, the file history. It needs no privilege
and no daemon.

**It is not a security boundary and is not trying to be one.** The sealed session runs as the same
user against the same filesystem and is not being kept from anything it sets out to reach. What it
does is make the specification *absent*, which is the one property the evidence depends on: a
transcript that reads `docs/spec/04-the-authoring-format.md` and then writes a correct Manifest
records a success the shipped product cannot reproduce, because on the machine a user installs
`hyper` on there is no `docs/spec/` to read.

**The seal is asserted rather than assumed.** Inside it the script searches for a `go.mod` naming
this module and for a `hyper` on `PATH` — issue #214's other condition, and one that covering
`$HOME/bin` is not the same fact as — and exits non-zero on finding either. It says `SEALED` when the
search has run, and an answer arriving without that line is a failure rather than a pass: a `bwrap`
that could not build a namespace returns nothing, and nothing is what a held seal also returns. That
is the check that survives the list going stale,
and it is the difference between a harness that is sealed and a harness that was sealed on the day it
was written. ADR-0095 logged the same hazard from the other side — two solved copies in sibling
directories went untouched *that time* — and the answer to *that time* is not a better setup, it is a
setup where the outcome is not a property of the run.

**`--clearenv` is part of it, and was learned the hard way.** The first run of this harness was
started from inside an agent session and inherited `CLAUDECODE`, `CLAUDE_CODE_ENTRYPOINT`,
`CLAUDE_EFFORT` and a messaging socket and token, so the sealed session was a child of the session
measuring it, at an effort level it did not choose. Its numbers agree with the record run — thirty-two
calls, seven `check`, the repository never left — and it is not the record, because a harness whose
result depends on who started it is the same measurement-validity defect in a smaller font.

## What was considered

**Doing nothing at all**, on the ground that the behaviour is gone. Refused, because the reason the
behaviour was visible on this machine is that this machine has a checkout to forage; every transcript
ADR-0096 is still waiting on would be collected the same optimistic way, and the cost of the seal is
one script and a case.

**A container.** It seals more and costs more — an image to build and keep, and a Claude Code
installation and credential inside it. `bwrap` reuses the machine's, and the property being bought is
absence of a directory rather than isolation of a process.

## Consequences

- **ADR-0096's two outstanding transcripts are run in this harness or they are not evidence.** The
  multi-host `read` composed from a fragment, and an effectful Operation creating and deleting over
  HTTP with header auth — neither has been run, and both were always going to be run on the machine
  that holds the answer to them.
- **A case runs the setup half** (`cmd/hyper/acceptance_test.go`): the repository handed over checks
  clean and has no `providers/`, its `AGENTS.md` is `mcp.Instructions` for the version it pins rather
  than a copy that went stale, and the script's own seal assertion has to pass for the case to
  finish. `ACCEPTANCE_SETUP_ONLY` is what stops it before the session, a session not being something
  a test can run.
- **The orientation is untouched, and so is every message.** Issue #216's remaining acceptance
  criterion was *whatever follows from that, or an explicit decision that nothing does*. Nothing
  does.
- **The residue ticket issue #216 anticipated is declined, and this is where that is recorded.** It
  expected *a much smaller ticket about the residue* if the pressure to forage had largely gone.
  There is no residue to write one about: the run left the repository zero times rather than fewer
  times, and the two behaviours from the same session that did leave something behind already have
  tickets — `hyper <command> --help` was issue #215 and is closed, and what this run turned up is
  issue #217, which is about a block that never arrives rather than about an agent going looking.
- **The count-as-an-oracle question ADR-0097 left open is answered.** *Whether the fix actually closes
  the repair loop is a question for a transcript, not for a test* — seven calls and a clean
  repository, against seventy-six and never.
- **The task's own artefacts are kept beside the harness** (`scripts/acceptance/tasks/`), a task that
  names three log directories bringing them with it. A transcript that failed because a path was not
  there would be evidence about the path.

## Found in the same run, and not this decision

**This client shows the model `structuredContent` and discards `content` on every return that is not
an error.** In the transcript, `review`'s three calls arrive as a JSON rows array — the rendered page
§9 promises byte for byte never reaches the agent — and a clean `check` arrives as
`{"rows":[],"truncated":false}` rather than its summary line. The server is doing exactly what §9
says; `hyper mcp` driven by hand returns the whole rendering in the `text` block, and it is the
client that drops it. This is issue #214's defect on the other branch of the same table, it was
invisible while `check`'s failing return was the only one anybody looked at, and it is issue #217. It
also strengthens the finding above rather than weakening it: the session reached a clean
repository and a correct refusal **without ever seeing a rendered review page**.
