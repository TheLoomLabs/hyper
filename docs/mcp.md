# The MCP server

`hyper mcp` starts an MCP server over stdio carrying thirteen of the sixteen commands. It takes no
arguments at all: no `--repo-dir`, no transport flag, no port. The server dies with its client and
offers no asynchronous handle, so it owns the author, validate and observe loop and short
effectful Runs. Long unattended work is a Cadence on an executor, not a session held open.

[§9](spec/10-surfaces.md) is the authority on the tool schemas.

## The thirteen tools

Each is named for the command it carries. The set is [the sixteen](cli.md) less `install`, `store`
and `compact`:

`providers` · `provider` · `operation` · `targets` · `check` · `review` · `run` · `probe` ·
`runs` · `run_show` · `changes` · `records` · `project`

One line puts the missing three on the far side of the boundary: an agent may read the record and
add to it, and may not create it, prune it, or bring anything new into the repository. `install`
is the single point at which third-party data enters the repository, `store` creates the record,
and `compact` is the one command that would let an agent prune the account it is itself held to.

`run_show` is the only name that differs from its command. A client holds every server's tools in
one flat namespace, where a bare `show` names nothing.

Ergonomics is the whole of the difference between the two surfaces. An MCP tool builds the command
line its command would have received and hands it to the same dispatch, so there is no second
place for a guardrail to be skipped or a Refusal to be reworded.

## Setup

In Claude Code, register it per project rather than writing a config file by hand:

```bash
cd /path/to/your/repo
claude mcp add --scope local hyper -- "$(command -v hyper)" mcp
claude mcp list        # hyper: … - ✔ Connected
```

In any other MCP client, the same server as configuration:

```json
{
  "mcpServers": {
    "hyper": {
      "command": "/absolute/path/to/hyper",
      "args": ["mcp"]
    }
  }
}
```

## Four things worth knowing

**Name the binary by absolute path.** A server inherits whatever environment launched the client,
and a desktop launcher or IDE that never sourced your shell profile gives you `Executable not
found in $PATH` with no obvious cause. `command -v hyper` above resolves it once, at the moment
you know it is right.

**`HYPER_REPO_DIR` is optional.** Without it `hyper` walks up from the server's working directory
to the git root, which is the repository you opened the client in. Set it (`-e
HYPER_REPO_DIR=/path/to/your/repo`, or the `env` block) only where the client starts the server
somewhere else.

**Put no credential in that file.** A Target declaration names the environment variable a
credential resolves from, and never the credential
([ADR-0007](adr/0007-hyper-never-stores-a-secret.md)). The process that performs a Run is the
server, so export the variable in the shell you launch the client from and let the server inherit
it. Writing it into a config file is the one thing this design is arranged to avoid. A client that
never sourced your shell profile has the same problem here as with `$PATH` above, and the same
answer: start the server through the wrapper that holds your secrets, `"command": "op", "args":
["run", "--", "hyper", "mcp"]`.

**A committed `.mcp.json` is documentation, not configuration.** Most clients treat a file in the
repository as project scope and will not load it until each user approves it, and an unapproved
one loads silently and says nothing. Commit one so a stranger can see how the repository is wired;
register your own at local scope so it actually runs.

## The agent arrives oriented

That is the whole of the setup. MCP's `initialize` result carries an `instructions` field and
`hyper` fills it: what `hyper` is, the five artefacts and where each lives, the loop, the three
commands that are the human's and why, that a Refusal retried unchanged refuses identically, and
one worked example of all five artefacts that checks clean.

`hyper project` writes the same text to `AGENTS.md` where your repository has none, and never
touches one that already stands. A client decides when it surfaces `instructions`, and a file in
the repository has no such contingency. One text, two channels, because two texts would disagree
the first time either was edited
([ADR-0093](adr/0093-orientation-is-a-handshake-field-and-hyper-writes-no-file-to-carry-it.md),
[ADR-0095](adr/0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)).
