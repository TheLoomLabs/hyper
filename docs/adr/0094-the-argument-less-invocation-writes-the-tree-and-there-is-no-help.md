# The argument-less invocation writes the tree, and there is no `help`

`hyper` with no arguments writes **§9's tree** on stderr and exits `2`: the six groups, the sixteen
names with the positionals §9's table states beside them, and the three commands outside the tree
named as such.

No command is added. `help` and `--help` are still not among the sixteen, still answer `unknown
command`, and there is still no per-command usage text anywhere. What changed is that the one message
the binary already wrote when asked what it is now says something.

## What the terminal could not answer

An agent in a fresh repository, with the MCP server wired and asked to author a Provider extension,
made three tool calls and then went to the command line to find out what the binary could do:

```console
$ hyper --help
hyper: unknown command "--help"          exit 2

$ hyper help
hyper: unknown command "help"            exit 2

$ hyper
usage: hyper <command> [args...]         exit 2
```

Three attempts, nothing learned. It then **began reading directories outside the repository** — two
neighbouring checkouts, and a `docs/spec/` section in a `hyper` clone that happened to sit on the same
machine — before the operator stopped it. A second harness, on a different model, shelled out to
`hyper changes --help`, `hyper records --help`, `hyper show --help` and `hyper help` after a `changes`
tool call (issue #209, issue #210).

The foraging is the symptom. The cause is that after those three commands there was nowhere left to
look. `hyper completions bash` does enumerate every name — it is where the surface lives (tree.go) —
but it enumerates them as a shell script, and nobody runs a completion generator to find out what a
tool does.

**The two surfaces were asymmetric, and the asymmetry is invisible from inside either.** ADR-0093 put
an orientation into the MCP handshake, and both harnesses tested there used it. Neither of them stayed
on that surface: every agent reaches for the terminal eventually, and the terminal would not describe
itself.

## Why not a `help` command

`help`, `--help` and per-command usage text are the three obvious answers and all three are refused,
each on its own ground rather than on one blanket rule:

- **A `help` command** is a seventeenth name in a tree §9 fixes at sixteen, *flat, one noun group, no
  aliases and no hidden commands*. The count is load-bearing — ADR-0088 refused a seventeenth on
  exactly that ground — and it would be the first name in the tree that `CONTEXT.md` does not define.
- **A `--help` flag** is a fourth global, where §9 closes the globals at three, and it would be a flag
  every command has to accept and none of them acts on. §9's three configure an invocation; this would
  be one that replaces it.
- **Per-command usage text** is the larger commitment and answers a question nobody was asking. The
  observed failures were all *what are the commands*, not *what are this command's arguments* — and a
  positional that resolves to nothing already answers the second in the only way that helps, by naming
  the command that enumerates the namespace it resolved against (§9). What stands in place of a manual
  page here is Discovery: three commands whose whole job is answering *which Provider, which Operation,
  how do I call it*.

None of the three is needed, because the defect is not a missing command. The message that already
exists for exactly this case — the branch `Main` already owns, at the exit code it already returns —
said `<command>` and stopped. **It costs one string.**

## What the page carries, and why the groups stayed

The **bare names** are what end the foraging. The **positionals** are what stop the next invocation
being a guess: an agent that learns `review` exists and not that it takes an artefact spends its next
turn finding that out at exit `2`. §9's table carries both, so rendering the table is the obvious
thing and the cheapest to keep true.

The **six groups** stayed, rather than flattening to sixteen lines. They are how §9 teaches the tree —
*which Provider, which Operation, how do I call it* is a run of three questions asked in order and not
three unrelated names — and a page that dropped them would be shorter by six lines and would have
taught the shape of the surface to nobody. The three commands outside the tree are rendered as a
seventh group and labelled as what they are, because a caller who cannot tell `version` from `records`
has learned the names and not the surface.

## One list, not a fourth copy

`completions` already knew every name, and the reason the surface is one compiled-in list is drift:
three shells describing one surface from three transcriptions of it would disagree the day the
seventeenth command lands (issue #104). A usage page assembled from a fourth transcription would be
that same defect with one more reader.

So the groups and the positionals moved **into** that list rather than beside it. `tree.go` now states
§9's table as the table — groups, names, positionals — and the flat sixteen every other reader wants
is derived from it. The page renders the groups; the completion scripts flatten them; a command §9 adds
reaches all four readers by one edit.

## Consequences

- **The invocation is unchanged in every respect but its text.** Still a usage error, still exit `2`,
  still stderr, still reading no part of the process — no working directory is resolved, no environment
  is read, no gate fires. A page that reached for any of those would be a command by another name.
- **`hyper` now writes twenty-eight lines where it wrote one.** That is the accepted cost, and it is
  paid only by the invocation that asked. A caller who types a command gets what that command says.
- **The tree is legible and still closed.** §9's own words are *no hidden commands*; this prints the
  list, which is the opposite of hiding one. Nothing is added, nothing is aliased, and `help` remains a
  word that names nothing.
- **The page is a thing this repository is now on the hook for keeping true.** It is held to §9's table
  by a case that transcribes the table from the specification and reads the page back
  (`internal/cli/usage_test.go`), and to the completion scripts' own list by a second one — a page that
  drifted from either would teach a caller a command that does not exist.
- **`CONTEXT.md` gains no term.** A usage message is not a domain concept.
