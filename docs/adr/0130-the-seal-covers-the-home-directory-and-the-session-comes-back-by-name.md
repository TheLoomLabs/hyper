# The seal covers the home directory and the session comes back by name

**`scripts/acceptance/run.sh` puts a `--tmpfs` over `$HOME` and binds three things back on top of it:
the Claude Code binary, the credential it authenticates with, and a `.claude.json` carrying the
onboarding flag.** Everything else a home directory holds — the caches Claude Code keeps, the prompt
history and its backups, `$HOME/bin`, Go's build cache, and whatever an attended session left lying
about — is not reachable from inside the namespace. This closes issue #257, and it inverts the cover
that ADR-0099 built and ADR-0109 widened: the seal used to name what to hide, and now it names what to
keep.

## What was in `$HOME` when the `monitor-retirement` run was prepared

Six directories, left by three tickets that each owe their evidence to a directory outside git
(#249, #248, #255):

- `hyper-249-hetzner` — a throwaway repository holding a **working Provider Manifest**, five commits
  and a `hyper-store` branch with nine Runs on it.
- `hyper-249-transcripts` — six raw session transcripts, which quote Manifests whole.
- `hyper-249-bin/hyper` — the published `v0.0.1-alpha` archive: a **stamped second binary**, against a
  harness whose written claim is *no source checkout, no second binary, no fixture internals*
  (ADR-0109).
- `hyper-248-install`, `hyper-248-fixtures`, `hyper-248-install-log.md` — a `hyper` repository and the
  install fixtures.
- `hyper-255-artefacts` and `hyper-255-log.md` — the by-hand completion of the task about to be run.
  **The answer key**, an hour old.

**Nothing was foraged.** The material was moved to `dev/hyper-sessions/` — under the checkout's
parent, which the seal already covered wholesale — before the run was bought, and that transcript is
clean (ADR-0129). What the move did not fix is the shape: it is a convention, and a convention is a
setup being trusted.

## Why the list lost rather than gained an entry

The cover was nine `--ro-bind`s over named paths, and **every one of them was added when something
turned out to be reachable**. ADR-0099 covered the checkout's parent, `$HOME/bin` and four Claude Code
caches; ADR-0109 added the build cache, this run's output directory and every previous run's. Issue
#257 is that same finding one directory further out, and the rule ADR-0099 settled is what condemns
the shape rather than the entries:

> whether a given run forages is a property of the run and not of the setup, so the setup cannot be
> trusted to control for it: it has to be made impossible.

A list of what to hide *is* a setup being trusted to control for it. It holds only for the material
somebody thought of, and session material is the class that grows: three tickets have now required it,
the next one will name its directory something else, and the seal's searches match a `go.mod`, an
`mcp.json` and a file called `lookout` — none of which is what an answer key is called.

**The list was also already behind Claude Code.** The four caches ADR-0099 named are still covered;
`~/.claude/backups`, `~/.claude/sessions`, `~/.claude/paste-cache` and `~/.claude/cache` appeared
afterwards and were not, and neither were `~/.claude.json.bak-hyper211` and
`~/.claude.json.bak-hyper-effectful` — two copies of this project's prompt history sitting in `$HOME`
beside the one file the seal did cover. Nobody had done anything wrong. That is what a list costs.

## What comes back, and why each

Three binds, and the first two are the same argument ADR-0109 made for `mcp.json` and `bin/hyper`: a
seal that hid them would be hiding the session rather than sealing it.

- **`~/.local/bin/claude`** — the client. On this machine Claude Code is installed under `$HOME`, and
  the path is read from `command -v claude` rather than written down, so a machine that keeps it
  elsewhere binds nothing and loses nothing. `bwrap` resolves the symlink as it binds, so what lands
  at that path is the binary itself.
- **`~/.claude/.credentials.json`** — what the session authenticates with. It is a secret and it is
  bound deliberately, which costs nothing that was not already spent: it was reachable under the old
  cover too, and the seal has never been a security boundary (ADR-0099) — the sealed session runs as
  the same user against the same filesystem. **It is bound writable**, read-only having been the
  first instinct and the wrong one: Claude Code refreshes its own token, and a refresh a long run
  could not persist would turn an hour of transcript into a report about authentication. What writes
  is the same client doing the same thing it does outside the seal, to the machine's own credential.
- **`~/.claude.json`**, the harness's own, carrying `hasCompletedOnboarding`. Without it the first
  turns go on onboarding and the transcript is about the client.

**`keep` stayed strict, and the two machine paths are guarded where they are named.** Everything
else handed to `keep` is a path `run.sh` created a few lines above — the repository, `bin/hyper`,
`mcp.json`, the `.claude.json`, and a path an `endpoint.env` named that the parser has already
checked exists. A missing one is a defect in the script, and `bwrap`'s hard error is the right report
for it; a `keep` that skipped what it could not find would turn *the repository is not there* into a
sealed session started without one. The client and the credential are the machine's rather than the
script's, so they are the two that may be absent, and each is guarded at its own call site.

**Three of the old covers turned out to be unnecessary rather than to need keeping**, which was
checked by running a session inside the new seal rather than reasoned about. `~/.claude/settings.json`
was covered with `{}` and is now absent, which is the same absence of this machine's hooks, plugins
and skills. `~/.claude/history.jsonl` and `~/.claude/projects` were bound to files in the output
directory and are now the tmpfs's own, which the client creates for itself.

**That last one closes a hazard ADR-0111 wrote down, and gives something up.** A sealed session wrote
its auto-memory to `~/.claude/projects/<slug>/memory/`, the bind landed it in the output directory,
and the ADR noted that *reusing one would carry a previous session's notes into the next*, nothing in
`run.sh` clearing `projects/`. There is no bind now: the note lands in a home directory that dies with
the namespace. What that costs is a copy of what the session wrote to memory, which was readable on
the host afterwards and is not any more. It is a copy rather than the record: the write is three tool
calls in the transcript with their content, which is where ADR-0111 read it from. Restoring it is one
`keep`, and the hazard comes back with it.

## The assertion inventories `$HOME` as well

The seal is asserted from inside itself, and the assertion gains the one root that matters. `$HOME`
and the output directory are now the same statement — a tmpfs with a `keep` list bound back — so one
walk asserts both: everything reachable under either, with the repository pruned, must be something
`keep` or `allow` recorded.

**A fourth `-name` beside the three searches was the other candidate and is the one this replaces.**
It would have gone stale the first time session material was called something new, which is the
failure the `mcp.json` search was itself chosen over a path list to avoid (ADR-0109) — and it is the
failure that produced this ticket. An inventory does not go stale. It goes noisy, and a name it did
not expect is the operator's to explain.

**Both halves were checked against the harness rather than assumed.** With a `go.mod` naming this
module and a file called `lookout` planted in `$HOME`, the old harness *refused to run* and named
them; the new one covers them and runs. With material named `hyper-249-hetzner` and
`hyper-255-artefacts` — the shape of what was actually there — the old harness passed in silence.
And a bind deliberately added inside `$HOME` that `keep` had not recorded is reported by the
inventory, so the walk is not passing vacuously.

## What it simplifies

- **`cover` declines a destination inside `$HOME`.** An empty bind there would not hide a path but
  *create* one — an empty `~/dev` in a home directory the tmpfs has already emptied. One guard retires
  six call sites and keeps the three that can name a path outside `$HOME`: the checkout's parent,
  Go's build cache, and a previous run's output directory.
- **The host-side search for previous output directories no longer walks `$HOME`.** One kept there is
  covered by the tmpfs, so searching for it would buy a `cover` call that `cover` declines — and that
  walk was the expensive half of a fence that runs on every `go test ./cmd/hyper`, once per task.
- **The same search *inside* the namespace now walks a tmpfs holding half a dozen paths.** It stays,
  because what it now says is that the covers came back empty-handed.

## What was considered

**A directory the convention names, covered by path** — issue #257's first candidate, and what was
actually done to unblock the run: session material moved to `dev/hyper-sessions/`. Refused as the
answer: the memory note is not a fence, and the next session that writes to `$HOME` reopens the hole.
The convention is kept because it is tidy, and nothing now depends on it.

**A fourth search rule.** Refused above.

**Covering `$HOME` and refusing to run when something is found there**, as the harness would for a
`hyper` on `PATH`. Refused for ADR-0109's reason one directory in: it would make the operator delete
this project's own evidence, and every attended session would break the next sealed run.

**Binding `~/.local/share/claude` back wholesale** rather than the one binary. Refused: it is 615MB of
three installed versions, and the inventory would have to be taught to expect a directory whose
contents are the client's business. What it buys is robustness against a client that reads its
siblings, and that is bought instead by the failure being loud — a client that cannot start writes no
transcript, which `run.sh` already reports as *the session wrote no transcript*.

**Keeping the `settings.json`, `history.jsonl` and `projects` binds** on the ground that they worked.
Refused: under a tmpfs they are three lines saying what absence already says, and one of them was the
path by which a sealed session's state reached the host.

## Consequences

- **The claim in `run.sh`'s header is unchanged and gains a paragraph.** *No source checkout, no
  second binary, no fixture internals — and the one binary that is reachable is the one the MCP server
  is* still holds, *binary* there meaning a `hyper`; the client and its credential are named beside
  `mcp.json` and `bin/hyper` as things a seal cannot hide without hiding the session.
- **ADR-0099's and ADR-0109's cover lists are amended rather than corrected.** This is a wider seal,
  not a fact that was wrong when it was written.
- **`TestAcceptance_TheSealCoversWhatAnAttendedSessionLeftInTheHomeDirectory` fences the finding
  itself.** It plants session material in a redirected `HOME` and runs the harness against it, so a
  return to a cover list fails under a case rather than under a reader. It writes nothing to the
  machine's own home directory, which costs it the two paths that are not there under a `t.TempDir()`
  — the client and its credential, both of which `keep` skips exactly as it does on a runner with no
  `claude` installed.
- **No acceptance re-run is owed.** This is not a taught repair
  (`docs/agents/acceptance-re-runs.md`): nothing an agent reads changed, and the harness's own
  assertion is what fences it. The one thing a run would have told us — that a session still starts
  inside the tighter seal — was bought directly and cheaply instead: `run.sh` was given a one-line
  task, the seal held, the session started, `mcp_servers` reported `hyper` **connected**, and the
  transcript came back with the answer in one turn. The client, its credential, `mcp.json` and
  `bin/hyper` are reachable; the rest of `$HOME` is not.
- **The suite cannot fence the half that matters most here, and that is deliberate.** Nothing in it
  may assert what an agent did (#221), and *a session still starts inside this* is a claim only a
  session can make. So the fence asserts the cover and the inventory, and the session was run by hand
  once, above.
- **A machine where Claude Code is installed as a package tree under `$HOME`** — an `npm` install
  behind a shim, rather than the native single binary — would bind the shim and not what it loads. The
  failure is loud and the fix is one more `keep`.
- **A machine that keeps its credential somewhere other than `~/.claude/.credentials.json`** binds
  none, and `--clearenv` means an API key in the environment does not reach the session either. The
  report is the empty transcript `run.sh` already refuses on, and the fix is again one more `keep`.
- **No transcript collected before this is invalidated.** What was reachable is on the record in
  ADR-0106 and ADR-0109, and every run made under the old cover was made on a machine whose
  `$HOME` held what it held. What changes is that the next one cannot reach it.
