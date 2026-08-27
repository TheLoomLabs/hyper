# A path argument is read against the repository, never against the caller's directory

`check [path...]` reads each path positional **against the repository root**: a relative path is
joined to that root, an absolute one is read as itself, and either way the path must resolve inside
the repository or it names nothing this command can report on. The stat and the filter use that one
root. The caller's working directory decides *which repository* is being checked and never *which
file inside it* is being reported on, and the rule is the same on both surfaces.

## The two roots, and what they cost

`check` resolved a path argument against the **process's working directory** and then made the
result relative to the **repository root** — two roots, one argument. Where the caller stood in the
repository root the two agreed, which is every case the corpus held and every ordinary `hyper check`
typed inside a repository. Where they differed the argument meant one thing to the stat and another
to the filter, and the two failures that follow are the reason this is written down rather than
fixed quietly.

**The command could report clean over a repository with problems.** `hyper check --repo-dir ../other
definitions/a.yaml`, typed in a directory that happens to hold `definitions/a.yaml`, stats the local
file, makes it relative to the *other* repository's root, gets something shaped like
`../<cwd>/definitions/a.yaml`, matches no problem's file, and exits `0` reporting nothing. That is
precisely what §9 refuses by name when it says why a path naming no file is `2` — *`hyper check
definitions/typo.yaml` exits `0` clean on a job that checked nothing* — arriving through the door
the sentence was not watching.

**And the MCP tool could not honour what §9 says its argument is.** Over MCP the client picks the
directory it starts the server in, and no argument overrides it: §9 is flat about it — *no tool
takes an override argument of any kind, under any name*. So an argument read against that directory
means something different per client for one string, and the caller cannot see the difference, state
it, or correct for it. A server started anywhere but the root — which is the ordinary case,
`HYPER_REPO_DIR` naming a repository elsewhere or the walk up starting *inside* one — refused paths
that exist.

The tool could not settle it: a tool builds the command line its command would have received, holds
no logic of its own, and has no repository in hand to re-root a path against even if it did (issue
#198, `internal/mcp/tools.go`). So the decision is the command's, and it is a decision rather than a
repair — the working directory is not obviously the wrong root, and a person typing `hyper check
a.yaml` from inside `definitions/` means the file beside them.

## Why the repository is the root

**A path goes back in the way it came out.** Every `problem` row `check` writes carries `file` as a
repository-relative path, on both surfaces, and the next act after reading one is naming it: `hyper
check definitions/a.yaml`, copied off the table, or the same string handed back through the tool. A
positional read against the caller's directory round-trips only from the root and silently filters
to nothing everywhere else. Reading it against the repository makes the output's spelling and the
input's spelling one spelling, which is what §12's opening rule asks of one fact reaching two wires.

**The other positional that names a file already works this way.** `review <artefact>` takes a
*repository-relative path* — its own word — and resolves it against the load's own paths, having
never seen a working directory at all. `check` was the only command that read a path against the
caller, and the Authoring pair reading one string two ways is the sharpest form of it: `hyper review
definitions/a.yaml` and `hyper check definitions/a.yaml`, typed side by side in `definitions/`,
named two different files.

**It is the only root an agent can name.** This is where the surfaces stop being a matter of taste.
A human can see the directory they are standing in; a client that started a server cannot, and
nothing in the protocol lets it ask. An argument whose meaning is a fact about the process's
environment is one an agent has to guess at, and the guess is invisible when it is wrong — a filter
that matches nothing looks exactly like a repository with nothing to report. *Ergonomics is the
whole of the difference between the two surfaces* (§9): a root that only one of them can compute is
not ergonomics, it is a second argument the other one cannot supply.

## The bound, and why the rule needs one

A root alone does not close the hole. `..` is spellable and so is an absolute path, so `hyper check
../other/definitions/a.yaml` would resolve, stat clean, and filter to zero problems exactly as the
working-directory reading did. **So a path that resolves outside the repository is refused**, and
refused *before* the stat: the repository is what a problem's `file` is positioned against, so a
path outside it names nothing this command could report on however well it names a file on the
caller's disk. It is a parse rather than a disk read, in ADR-0087's sense — everything it decides,
it decides from the argument and the root.

The empty string is refused on the same reading, in the same breath: `check ""` would resolve to the
root and quietly become *every problem there is*, which is not what a caller who wrote it meant. `.`
is the root and does mean that, because it was typed.

**The bound is lexical, and that has a price worth naming.** It compares the argument against the
root as written and reads no link, so an absolute argument that spells a symlinked prefix
differently from the root — `/tmp` against `/private/tmp`, a checkout reached through a link, a
`HYPER_REPO_DIR` set one way and a path typed the other — is refused for a file that is genuinely
inside. Resolving links would fix that and would cost the property the rule rests on: the bound
would become a disk read, deciding on what the filesystem answers rather than on the argument, and
it would take a second arm for the case where the resolution disagrees with the parse. What the
caller gets instead is a decline with a sentence — never a silent zero — and the repository-relative
form, which is the spelling the tool advertises and the rows come back in, cannot reach the case at
all.

**What stays the caller's is what points outside the repository.** `--repo-dir` names the repository
itself and cannot be read against what it establishes; `--secret-out` is refused unless it resolves
outside the working tree (ADR-0007). Both remain relative to the working directory, and that is the
same rule read from the other end rather than an exception to it: a path naming something *inside*
the repository is the repository's, and a path naming the repository or something beyond it is the
caller's.

## Considered options

- **Working-directory-relative everywhere**, filtering by the same absolute path the stat used, and
  §9's *repository-relative* amended to say so. Keeps the terminal ergonomics whole. Rejected on the
  MCP argument above — it leaves the tool's paths dependent on a directory no caller can see or set
  — and on the round trip: `check`'s own rows would name files in a spelling its own argument would
  not accept.
- **Working-directory-relative, falling back to the repository root where the working directory is
  outside it.** Rejected. It costs a rule with two arms, and it does not even buy the case it was
  drawn for: a server started in `definitions/` is *inside* the repository, so the fallback never
  fires and repository-relative paths are still refused. Where both arms could resolve, which file
  was checked would depend on what else happened to be on disk — the failure mode this ADR exists to
  delete, wearing a fallback.
- **A `--paths-relative-to` flag, or a tool argument naming the root.** Rejected twice over: §9
  fixes three globals and no more, and admits no repository argument on any tool under any name.
- **Leave it and let the corpus pin the agreeing case.** The status quo, which is what the `wd` file
  beside every MCP path case was doing. Rejected: a fixture that only drives the case where two
  roots agree documents an accident as a contract.

## Consequences

- **A path typed from a subdirectory is refused rather than resolved.** `hyper check a.yaml` from
  inside `definitions/` exits `2`; what works there is `definitions/a.yaml`, the same string the
  table prints, or the absolute path the shell completes. This is the accepted cost and it is a
  decline with a sentence, never a silent zero — which is the whole reason the trade goes this way.
  A case in the corpus drives it.
- **`hyper check --repo-dir <elsewhere> <path>` can no longer report clean over a repository with
  problems.** It reports that repository's problems, or it declines.
- **§9's `check(paths?)` sketch is true as written**, and the tool's schema says the same root the
  command uses. The `wd` file each MCP path case carried is gone: those cases now run with a working
  directory that is not the repository at all, which is the claim rather than a convention.
- **No new exit code and no new `error_code`.** Both declines are §9's usage error — exit `2`, no
  row stream, the rendering on stderr — which is where every other positional that names nothing
  already lands (ADR-0060).
- **`check` gains no flag, and nothing about the repository's own resolution moves.** `--repo-dir`,
  `HYPER_REPO_DIR` and the walk up from the working directory are untouched (§9, ADR-0014).
- **A path that exists and holds no problem still exits `0`, and that is the answer rather than the
  hazard.** `hyper check README.md` reports nothing because nothing is positioned there, exactly as
  a clean artefact does; what §9 refuses is a path naming *no file*, and that is the `2` above. The
  hazard this ADR closes is the narrower one — a path that named a file and could not name a problem
  — and closing it does not turn a quiet answer into an error.
