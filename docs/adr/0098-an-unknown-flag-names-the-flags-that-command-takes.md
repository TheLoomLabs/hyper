# An unknown flag names the flags that command takes

`hyper <command> --flag`, where the flag is not one that command takes, writes a **second line naming
the namespace it was resolved against**: the command's own parameters, and the three configuration
flags §9 closes at three. The namespace is written out rather than pointed at, because it is a handful
of words. A command that takes no parameters of its own says so.

A token spelled with **one** hyphen resolves against that same namespace, so `hyper check -h` is this
message rather than a path that does not exist.

No command is added, no flag is added, and no per-command usage text is written. `--help` is still not
a flag, still reaches the branch every unknown flag reaches, and still exits `2`. What changed is that
the message that branch already wrote now says something.

## What ADR-0094 fixed, and where it stopped

ADR-0094 gave the **command** namespace a second line and left the **flag** namespace without one. The
two messages sat one word apart:

```console
$ hyper --help
hyper: unknown command "--help"
  the nineteen commands are that namespace, and hyper with no arguments lists them
                                                                          exit 2

$ hyper check --help
hyper check: unknown flag --help                                          exit 2
```

The second line is the whole of ADR-0094's fix and the second message did not have it. §9 states the
rule generally — what a name that resolves to nothing writes is **the name that was typed, the
namespace it was resolved against, and the command that enumerates that namespace** — and a flag is a
name resolved against a namespace like any other. `unknown flag --help` names the first of the three
and stops.

## The transcript ADR-0094 did not have

ADR-0094 refused per-command usage text on the ground that nobody had been observed asking for it:

> The observed failures were all *what are the commands*, not *what are this command's arguments*

That was true of the transcripts it had. It was not true of the session issue #214 was opened on, where
the surface was MCP-only, the agent already knew the command names, and what it typed was
`check --help` — three times, learning nothing from any of them. It is ADR-0094's own evidence one
level down: three attempts, nothing learned, and in ADR-0094's case what came next was *reading
directories outside the repository*.

**The refusal of per-command usage text stands.** This is not a manual page and not a `usage:` block
per command: it is one line, on the branch that already existed, at the exit code it already returned,
naming a namespace the parser already held. The three things ADR-0094 refused are still refused —
a `help` command would be a seventeenth name in a tree §9 fixes at sixteen, a `--help` flag would be a
fourth global where §9 closes them at three, and neither is what this is.

## What the line names

Three things could have gone on it, and the first is what went:

- **The flags this command takes.** The most useful, and the only one that needs a spelling for *this
  command takes none*.
- **The three globals.** What a caller on a command with no parameters of its own actually needs to
  hear, and nothing at all to a caller on `records`.
- **ADR-0094's pointer verbatim**, sending them to the tree page. The page names the three globals and
  no command's parameters, so it answers the second question and not the first.

It names the first **and** the second, in one sentence, because together they are the namespace and
either alone is a partial account of it:

```console
$ hyper records --help
hyper records: unknown flag --help
  records takes --limit, --since, --target, --definition, --name and --history, past --json, --repo-dir and --no-color

$ hyper check --help
hyper check: unknown flag --help
  check takes no flags of its own, past --json, --repo-dir and --no-color
```

**It is the page and not the pointer, which is the one place it departs from ADR-0094.** ADR-0094
points at the tree because the tree is twenty-eight lines and a caller who missed a keystroke did not
ask for a tour. A command's flags are five words, they are enumerated nowhere else in the tool, and a
line that pointed at a second invocation would cost the round trip this exists to remove. The
asymmetry is in the namespaces, not in the rule.

**A command taking none says so.** *takes , past --json* is a sentence with a hole in it, and the fact
a caller needs from `check` is that the hole is the whole answer.

## No near miss, and the ground is not inherited

ADR-0094 suggests no near miss for a mistyped command, on ADR-0047's ground that a suggestion is a
partial name resolved on the caller's behalf. That ground is about a Run id and a human who runs
something they did not type; a flag cannot be run by accident, so it does not carry over unexamined.

The ground here is stronger and it is this message's own: **the whole namespace is already on the
line.** `--limi` → *did you mean `--limit`* would pick one name out of a list the caller is reading in
the same breath. There is nothing for a suggestion to add, and a caller who accepted one would have
accepted it over the enumeration standing beside it.

## One hyphen is a flag

`hyper check -h` fell through the `--` prefix arm to the positionals and answered:

```console
$ hyper check -h
hyper check: -h: no such file or directory
```

A filesystem error for a question about the interface, and `-h` is exactly what a caller who has
already tried `--help` types next. It is now the same message `--h` gets, and it costs nothing to make
it one: §9 has no single-hyphen flag anywhere for `-h` to be, so a token spelled that way names nothing
in the namespace it is resolved against and says so.

**`--` is what names a file whose own name begins with a hyphen**, and it did before this and does
after it. A bare `-` is left to the positionals: it is a whole conventional file name, and nothing is
being resolved when it is typed.

## One list, not a second transcription

The message is composed from `parameters` — §9's own word for what a command takes past the three
globals, and the value each command already states at its own dispatch. **No command carries a list of
its own flags**, which is the property ADR-0094 spent a section on arriving one namespace down: sixteen
per-command lists is sixteen things to forget, and there are none.

What is written twice is a **spelling**, not a list: the parser names each flag in the arm that reads it,
and the table this message renders from names it again. That is one line against one line and it is held
by a fence rather than left to care — every spelling is driven through the parser, and one that comes
back *unknown* fails the suite. The alternative was a parse loop rebuilt as a table of descriptors, which
would have collapsed the two into one at the cost of rewriting the one function every command's arguments
pass through, for a defect a test catches.

Two things follow from that and both are deliberate:

- **Four parameters that never reach the parser are members of the value anyway.** `probe`'s `--input`,
  `run`'s `--secret-out` and `--dry-run`, and `show`'s `--expansion` are taken off the argument list
  before the globals are parsed, each for the reason its own splitter states — a parser that knew about
  all four is one every other command's signature would have to admit, and `hyper compact --dry-run`
  would stop being the unknown flag it is. What the value states is the command's **flag namespace**
  and not the parser's list of cases, so they are named there; a `run` answering *takes no flags of its
  own* would be naming a namespace two of whose members it had accepted three lines earlier.
- **The flags are rendered in the members' own order**, not in the order §9's prose happens to name
  each command's. Following the prose would be exactly the second transcription this avoids.

The three globals come off `tree.go`'s list — the spellings the usage page renders and the completion
scripts offer — rather than being written out again here, for the reason that list exists at all.

## Consequences

- **The invocation is unchanged in every respect but its text.** Still a usage error, still exit `2`,
  still stderr, still reading no part of the process: no working directory is resolved, no environment
  is read, no gate fires, and stdout carries nothing in either form.
- **The first line is untouched.** This adds a line; it does not rewrite the one that names the flag,
  and the corpus holds both.
- **A hyphenated positional is now reachable only past `--`.** That was already the documented way to
  name one and the corpus had no case that did it otherwise, so the cost is a rule stated where it was
  previously true by accident.
- **The message is a thing this repository is now on the hook for keeping true**, and two fences hold
  it from the two directions it can drift (`internal/cli/flags_test.go`). One holds every member of
  `parameters` to a spelling and every spelling to its member's name, so a parameter added and left
  unspelled fails rather than shipping a namespace that under-reports itself. The other holds every
  spelling to the **parser** — which spells each flag a second time, in the arm that reads it — so a
  flag renamed in one place and not the other fails rather than shipping a line naming a flag the
  command refuses. A third case drives `--help` against all sixteen commands in the tree.
- **`CONTEXT.md` gains no term.** A usage message is not a domain concept.
