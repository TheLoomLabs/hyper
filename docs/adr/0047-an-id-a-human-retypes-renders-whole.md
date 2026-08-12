# An id a human retypes renders whole

The Run id §8's terminal line carries renders in full, and nothing anywhere resolves a partial one: a
command taking a `<run-id>` matches it whole or matches nothing. Every other id on every other human
surface stays abbreviated — the Comparison's header, a `runs` row, a Provenance revision (§7, §8) —
because those are read and this one is typed.

We chose this because the reading an implementer reaches unaided is the other one, and the corpus
depicted it: §8's Refusal footer read `hyper show 01991ea6-b118… --expansion`, a command that cannot be
run as written, in the surface that is the entire path back from a declined Step (ADR-0001). The
abbreviation convention is right where it came from — a revision is a fact to recognise, and git taught
every reader to shorten one — and there are only two ways to make a command carrying an ellipsis
correct: print the id whole, or resolve a prefix. A prefix resolves against the Store, and the Store
grows monotonically and is written by two environments (ADR-0006, ADR-0011), so the same command is
unambiguous in August and ambiguous in November with nothing in the repository having changed. That is
state-dependent behaviour in the position where an operator is copying a string in order to read what a
`destroy` did, and it is the convenience this tool refuses elsewhere for the same reason — `runs
--limit 1` is not a way to find your own Run either, on a Store the runner also pushes to.

## Considered options

- **A unique prefix, git-style, an ambiguous one being an error.** The cheapest way to make the corpus
  as written true, and the interaction every operator already knows. Rejected: it makes a command's
  meaning a function of how much history the Store holds and which environment last pushed. The
  collision is not even the sharp edge — a prefix that resolves today, resolves to a different entry
  next month, and reports nothing unusual either time is silent, and what it addresses is the evidence
  a human is about to read as the account of an effect.
- **Abbreviated for the eye, whole inside the suggested command.** Rejected as the worst of the three:
  two renderings of one id within one screen, and no help on the completed path at all, which has no
  command to hide a whole id inside — §8 gives a Refusal a next command and a completed Run none.
- **Every id whole on every surface.** Consistent, and it retires the residue below. Rejected on width:
  `runs` and the Comparison's header are tables read down a column, where thirty-six characters a row
  buys recognition nobody is short of — the eye there is matching one entry against another, not
  transcribing either.
- **A short `hyper`-minted run name beside the id** — a counter, or a word pair. Rejected: a Run id is a
  UUIDv7 exactly so either environment can mint one alone with nothing to contend over (§7), and a
  second, shorter name for the same Run is a second identity that must be unique across two writers,
  which is the contention the design removed, returning as ergonomics.

## Consequences

- **An id read out of a table cannot be retyped.** This is the price, and §8 states it rather than
  leaving it to be met: what supplies an id whole is the terminal line of the Run that wrote it, or
  `--json`, which abbreviates nothing anywhere.
- **The terminal line is a working surface rather than a courtesy.** It is where an operator gets an id
  from, which is the reason it renders the one fact whole that every neighbouring rendering shortens.
- **No ambiguity case exists, so nothing has to explain one.** No error code names a partial match, no
  rule decides which of two entries wins, and §12's closed set grows by nothing. What a command does
  when the id it was handed matches nothing is a different question, belonging to every positional
  argument rather than to this one.
- **Widening it later is possible; narrowing it is not.** Accepting prefixes is a change no existing
  command's output stops being valid under, and it can be made the day a Store is large enough for the
  ergonomics to bite. Withdrawing prefix resolution once scripts rest on it cannot be.
