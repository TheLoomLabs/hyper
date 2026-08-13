# Naming nothing is a usage error, fetching nothing is not

Nine of the sixteen commands take a name positionally, and eight of the nine answer a name that
resolves to nothing with **exit `2`, no `error_code`, and no row stream at all**. `install <ref>` is the
ninth and exits `1`. The line between them is not what the name looks like but whose namespace it
belongs to: the eight resolve against something this repository already holds, and `install` resolves
against a registry over a network.

The reading an implementer reaches unaided is that not-found is a load failure. It is the POSIX habit —
`cat nosuchfile` exits `1`, and `2` is what a shell tool spends on a bad flag — and §9 appears to
confirm it, since `review` already says that "only an artefact that would not load exits 1, and what it
writes then is `check`'s row". Reaching for `1` gets all three of this decision's parts wrong in one
move: it puts every command on the same code, so the `install` split never arises; it gives the fault
`check`'s rendering; and it makes the version pin gate and an absent Store contend with it on
undefined terms.

**`check`'s rendering cannot hold it.** A `check` row is a file, a line, a column, a field, an
`error_code` and a message, positioned so that the next act is an edit (§9). A name that matches
nothing has no file and no line — that is the whole content of the fault — so the row arrives with both
coordinates empty and an `error_code` that §12 defines as *the identifier of a check that declined*.
Nothing declined. There is no artefact to attribute a check to, and minting a code for the absence of
one would put a member in the closed set that names no check anywhere, which is the property the set is
closed on.

**`77` makes a promise that is false here.** §12 pairs `75` and `77` as `EX_TEMPFAIL` and `EX_NOPERM`
precisely to carry *retry me* against *a verbatim retry will refuse identically*, and §9 states that a
Refusal's remediation points at an artefact to edit. The remedy for a mistyped Procedure name is to
type a different one, or to write a Procedure that does not exist yet. Both are the caller's next act
and neither is an edit to a reviewed artefact, so the `77` rendering would end by naming artefacts it
cannot name.

What is left is [ADR-0036](0036-every-run-is-a-run-of-a-procedure.md)'s principle applied at its full
width rather than to the one case that provoked it: **a Refusal is the artefacts declining an act; a
usage error is there being no act to decline.** That ADR priced `run` handed a Definition — an artefact
that exists and is the wrong kind. A name matching nothing at all is the same fault with less in hand,
and it belongs to every command with a positional rather than to `run`.

**The `install` split is where the principle stops, and it stops on evidence rather than on taste.**
Exit `2` is a claim that the invocation was wrong, and `hyper` can make that claim about the other
eight from what it already holds: the working tree, a Store branch, a `stat`. For a ref it cannot. The
same command line resolves on Tuesday and 404s on Wednesday, resolves on a laptop and not behind a
proxy, and resolves for one account and not another — and offline it cannot be judged at all. An exit
code that says *you typed it wrong* must not depend on a round trip, and a `404` sits beside the
timeout and the unreachable host, which are unarguably `1`. Keeping `install` at `2` for consistency
would buy a tidier sentence by making the tidiest code in the set contingent on the network.

**Case is `hyper`'s to decide, not the filesystem's.** A name matches byte-exact over UTF-8,
case-sensitive, compared against the artefact's own `name:` rather than settled by whether an `open`
succeeded. §12's string rule is already byte-exact, and `name-mismatch` already pins an artefact's name
to its file's basename. §7 has met this exact problem and answered it in the same shape: case "is the
one place the two environments genuinely differ, a laptop's filesystem being usually case-insensitive
and a runner's not, so the rule is `hyper`'s rather than the filesystem's", decided by reading rather
than "by attempting the write and seeing what happens". A lookup settled by whether an `open` succeeded
is that rejected move at the other end of the same command — and left there, `hyper review Deploy`
renders on a laptop and exits `2` in CI, a divergence between the only two environments this tool has,
produced by a rule nobody wrote.

## Considered options

- **`1` for all nine, on `check`'s row.** Rejected: the row has no file and no line to carry, and the
  `error_code` it would need names no check. It also collapses into `check`'s *problems found* the
  case where the repository is fine and the invocation is not.
- **`77` for all nine, as a guardrail declining.** Rejected: nothing was reviewed, so nothing refused,
  and `77` promises a remedy — an artefact edit — that does not exist here. A caller that retries on
  `77` is told by §9 that a verbatim retry refuses identically, which is true; the useful thing to say
  about a typo is that a *different* invocation will work.
- **`2` for all nine, `install` included.** Rejected on the network argument above: it makes the one
  code that should be decidable from the invocation depend on a fetch, and puts a `404` on a different
  code from the timeout that preceded it.
- **A tenth `error_code`, minted for the fault and carried on `1` or `77`.** Rejected: every member of
  §12's set is the identifier of a check that declined, attributed where that check is stated. There
  is no check and no artefact to attribute one to, and the set holds at forty-six.
- **Exit `0` with an empty result for `show <run-id>`**, on the ground that `runs`, `records` and
  `changes` all return zero rows and exit clean when their filters match nothing. Rejected: §9 makes
  the positional on `changes` a name rather than a filter — it "decides which rendering you get rather
  than filtering the rows of one" — and `show` promises one entry in full. A `show` that renders
  nothing and exits `0` is indistinguishable from success in a CI log.
- **A near-miss suggestion, or a listing of what does exist.** Rejected:
  [ADR-0047](0047-an-id-a-human-retypes-renders-whole.md) refused partial resolution of a Run id, and a
  suggestion is that same resolution moved from the argument parser into the error message — a human
  who accepts one has run something they did not type. Enumerating a namespace is the listing
  commands' job, and they carry `--limit` and a truncation marker because an unbounded return is the
  hazard.
- **Returning it on MCP as a domain answer** — `isError: true`, no `outcome` key. Rejected: that is
  exactly the shape a guardrail declining already returns, so a usage error would become
  indistinguishable from a Refusal on the one surface with no exit code to separate them.

## Consequences

- **`2` covers eight commands and every kind of name they take** — an artefact, a Provider, an
  Operation, a Run id, a path — and carries no `error_code`. §12's set holds at forty-six.
- **No row stream opens on a usage error.** stdout is silent in both modes and the rendering goes to
  stderr. §9's *the last row is always the terminal row, and its absence means the stream was cut off*
  is unviolated because nothing opened one, and its claim that `run` is on the `outcome` side "on every
  path it takes" is corrected to every path on which a Run was **attempted**: a usage error is not a
  path `run` takes, it is `run` never starting.
- **A positional resolves against its own namespace, and whatever that namespace requires is in place
  before the lookup can happen.** The pin gate fires first for all sixteen. A working-tree name needs
  nothing beyond it, so `hyper run typo` is `2` on a repository with no Store. `show` is the exception
  that states the rule: the Store is its namespace, so `store-absent` (`77`) necessarily precedes the
  lookup.
- **`install` carries three codes** — `2` where the ref grammar rejects the invocation, `1` where the
  registry does not hold the ref or the fetch did not complete, and `77` for
  `origin-digest-mismatch`, a check declining bytes that did arrive.
- **`check` stats its paths before loading anything.** A path naming no file is `2` and no problems are
  reported, which closes a live hazard: with `check` reporting only the problems positioned in the
  paths named, a path matching nothing would filter to zero problems and exit `0` clean.
- **`probe` is the ninth command, and §9 had priced only its inputs.** Its two positionals resolve
  against the Provider set like `operation`'s, and a Provider or Operation matching nothing is this
  usage error rather than a fourth thing.
- **MCP's definition of a malformed call grows a third member** — an argument that is well-typed and
  names nothing — under §9's existing rule that every usage error the CLI states arrives there as a
  protocol error. `install` has no tool, so its `1` has no MCP half.
- **The lookup does not fold case, and §7's Record identity does.** The two are the same methodology
  and not the same rule: §7 folds to stop a `Foo` and a `foo` coexisting in one Store, which is a
  collision rule. A lookup has nothing to collide with, and folding it would let a typed name find an
  artefact it does not spell — a mistyped `run` silently running something. What both rules share is
  that `hyper` decides, on what it read, identically on both platforms.
- **A name that an author wrote into an artefact is a different fault and is not decided here.** That
  is a name `check` must position by file and line, and it has both.
