# The server is started by `mcp`, and it stands outside the tree

The MCP server §9 states is started by **`hyper mcp`**, which takes no arguments at all. `mcp` is
the **third command outside §9's tree of sixteen**, beside `version` and `completions`, and not a
seventeenth command in it. It passes the test those two pass — it reads no repository, and it says
nothing about `hyper`'s domain — and `serve`, `mcp serve` and a global flag are each refused on a
rule already written down.

## The gap, and why it cannot be left to the implementation

§9 states the server in one sentence and never names the invocation that starts it: *the server is
the same binary, started by the client over stdio: one process per client, dying with it*. Earlier
in the same section it fixes **sixteen commands, flat, every name a word the glossary already
defines**, with **no aliases, and no hidden commands**, beside the two that stand outside it because
*neither reads a repository and neither says anything about `hyper`'s domain*. Both sentences are
the spec's; between them there is no name a client can put in a config file.

A ref could be left to the implementation and be wrong once (ADR-0087); this cannot be left there at
all. The invocation is the one part of this surface a **human types by hand into a file `hyper`
never reads** — `{ "hyper": { "command": "hyper", "args": ["mcp"] } }` — so it is a name in somebody
else's tracked file the day the first client is configured, and changing it later breaks every
machine that was. It is decided here, ahead of the ticket that builds the server, so that ticket
inherits a name rather than choosing one.

## Why it is not a seventeenth command

Both halves of §9's own test hold, and the second is the one that decides it.

**It reads no repository.** The pin gate fires **per tool**, at the moment a tool resolves one —
which is the same moment the command it carries does — so the process itself compares nothing
against a pin at startup. That is the shape `version` and `completions` have, arrived at
differently: they never resolve a repository at all, and `mcp` resolves one only inside an act that
is already gated. It is therefore **not a fourth exemption from the gate**, and ADR-0020's three
stand exactly as they are — every tool declines against a skewed binary precisely as its command
does.

**It says nothing about `hyper`'s domain.** `mcp` is the protocol's name. `CONTEXT.md` does not
define it, and the tree's rule is that **every name in it is a word the glossary already defines** —
so a seventeenth command would be the first that is not. What the process does once started is the
thirteen tools, and each of those is named for a command already in the tree: the domain is reached
*through* this name, never *by* it.

**And the count is load-bearing.** §9 counts to sixteen five times — the table's own sentence,
*fifteen of the sixteen* at the gate, *`project` is the sixteenth*, *nine of the sixteen take a name
positionally*, and the tool set's *three of the sixteen commands are absent*. Moving the count would
move all five, and it would move them for a word that is not a verb over Definitions, Runs or the
Record. The three outside the tree are what the corpus counts separately for exactly this reason.

## Considered options

- **`serve`.** The one word §9 and §13 each spend a paragraph refusing — *there is no `serve`, no
  daemon, and no remote transport*, and *`--server` and `--token` belong to a `serve` that does not
  exist*. §9 then goes out of its way to say the MCP server *is not the `serve` above wearing
  another name*. Reusing the word for a stdio process that dies with its client would make that
  sentence read as though it had been withdrawn, and it would invite the port, the flag and the
  token back one release later, each as a small extension of a name that already promised them. The
  refusal is not of the mechanism — this mechanism is fine and is the one §9 states — but of the
  word.
- **`mcp serve`.** A second noun group where §9 admits **one noun group and no other nesting**. It
  also buys nothing: `store init` is a group because `store` has state a verb acts on, where `mcp`
  would carry exactly one sub-verb forever, which is a group with nothing to group. And it would put
  `serve` back in the surface with a qualifier in front of it, which is the previous option refused
  more quietly.
- **A global flag** — `hyper --mcp`, or a `--serve` that turns any invocation into a server. Refused
  twice over. §9 closes the globals at **three** — *there are three globals and no more* — and
  ADR-0014 says what they are for: they govern **presentation only**, and everything that survives
  that rule is *invisible to the outcome*. A flag that decides whether the process is a command or a
  server changes what the binary **is**, not how it presents — which is the one thing the three are
  characterised by not doing. It would also be offerable after every command in three completion
  scripts, describing sixteen invocations that do not exist.
- **A separate binary** — `hyper-mcp`, shipped beside `hyper`. Refused on §9's own sentence: the
  server is *the same binary*. Two artefacts is a second thing to install, a second thing to pin,
  and a second digest in the Repository declaration, in exchange for a name this decision gets for
  free.
- **Say nothing, and let the milestone that builds the server decide.** The status quo. It leaves
  the first client config in the world to a choice made under a deadline, in a package comment,
  where no reader of `docs/adr/` would find the reasoning — and the choice is unusually expensive to
  revisit, being typed into files `hyper` neither reads nor can migrate.

## Consequences

- **`tree.go`'s `outsideTree` carries `mcp`**, so all three completion scripts offer it at position
  one and offer **no global after it** — which is what that list already means: *what the globals
  govern*, not *what the gate skips*. The same edit reaches bash, zsh and fish, which is why the
  list exists.
- **`hyper mcp` answers `unknown command` and exits `2` until the server lands.** That is the
  accepted cost `tree.go` states for every name the spec fixes ahead of the milestone that builds
  it, arriving for the first time at a name outside the tree.
- **No exemption moves and no closed set moves.** ADR-0020's three gate exemptions stand; §9's three
  globals stand; the tree stays at sixteen and the tool set at thirteen. What changes is one
  sentence in §9, *two more commands* becoming three.
- **§13's *No `serve`, no daemon, and no remote API* is unweakened.** Nothing listens on a port and
  nothing outlives the invocation that started it: this decision names an invocation, and an
  invocation that dies with its client is the thing that sentence permits rather than the thing it
  refuses.
- **The name is not `hyper`'s configuration.** It is typed into the client's file, which `hyper`
  neither reads nor is affected by — so ADR-0014's *there is no place to put a future setting*
  stands, and there is nothing here for a second machine to have set differently.
- **`CONTEXT.md` gains no term.** `mcp` is the name of a protocol and of the invocation that speaks
  it, not a domain object — which is the whole content of this decision, seen from the glossary.
