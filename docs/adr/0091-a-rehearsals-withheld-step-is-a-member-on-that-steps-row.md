# A rehearsal's withheld Step is a member on that Step's row

The Step a `--dry-run` stopped at carries **`withheld: true`** on its `step` row, and no other row
carries the key. One renderer writes both forms, so `--json` and the MCP envelope gain it together.
The page keeps its sentence — the row carries the position, the prose stays prose.

## A page fact on a page and nowhere else

A rehearsal that stops renders which Step it withheld. §9:

> A dry-run that stopped renders that it stopped and why — the withheld Step whose output the next one
> would have read — with the Steps after it *never reached*.

On the page that is one sentence beneath the Step table:

```
STEP  ID       KIND    DISPOSITION    RECORDS
1     status   read    ran            1
2     publish  mutate  never-reached  –
3     confirm  read    never-reached  –

stopped at publish. a rehearsal performs the reads it reaches and withholds the first effectful step rather than simulating it.
```

Neither machine surface carried it. The `--json` stream wrote three `step` rows, two of them
*never reached* and indistinguishable from one another; the envelope carried the same three rows and a
summary line reading `completed · dry-run · run <id>`. Which Step was **withheld**, as against merely
unreached behind it, was on the page and nowhere else — and it had been since milestone 5, `run` over
MCP inheriting the gap rather than opening it (issue #200, issue #206).

The reason it stayed a page fact is that no Disposition is one. §12's seven are closed and the withheld
Step is *never reached*: the Run ended before it, it wrote no file, and *never reached* is what an entry
reads back from that silence (ADR-0010). So the fact lives on `run.Answer` as a **position**, and until
now the only thing that read it was the sentence.

## Why it is a row's member and not a footnote

**§8 already states the principle, in the other direction.** Of the terminal line and the `outcome` row:

> a fact the stream states and the page does not is the two surfaces disagreeing about what happened
> rather than differing in shape

Read symmetrically, that is this. One renderer produces both forms (ADR-0026) so that neither can hold a
fact the other lacks, and a fact reaching only `runPage` is the same defect with the surfaces swapped.

**The obvious inference is the one the code refuses to make.** A consumer could take the first
*never reached* Step as the withheld one. `withheldStep` does not, and says why:

> a page that inferred a rehearsal from them would put the sentence under a Run that failed

A Run the world resisted leaves *never reached* rows too. The inference is sound only under
`outcome: completed` **and** `dry_run: true`, and nothing on either machine surface said that it was
sound at all. A consumer that got it wrong reported the boundary of a partial answer in the wrong place.

**A rehearsal's answer is partial by construction.** §9 is flat about the same shape one axis over — *a
truncated result must never look complete* — and a truncation carries a marker for exactly this reason.
The partiality here was legible on the page and inferable on the wire.

**It is what a `dry_run` call is for.** An agent calling `run(dry_run: true)` asks *what would this do,
and where does it stop*, and the second half of the question was the half the envelope did not answer.

## The absence rule is the whole of the semantics

The member is written `true` on one Step and carries no key anywhere else — §7's absence rule, which is
what makes the member itself the discriminator rather than a value to be compared. Three consequences
follow and all three are what was wanted:

- **A Run the world resisted writes the key nowhere**, whatever its *never reached* rows say. The
  position is only ever set where a rehearsal withheld a Step, so a halt leaves nothing to write it at.
- **A rehearsal that reached the end writes it nowhere** — a read-only Procedure withholds nothing, and
  there is no Step at position zero for the comparison to land on.
- **`false` is never written.** A `withheld: false` on every other row would be each Step claiming to be
  a Step some rehearsal did not withhold, which on a Run that is not a rehearsal is a claim about
  nothing. It is the opposite of `dry_run`, which is §7's one exception and is written always: what a
  reader that takes `dry_run`'s absence for `false` gets wrong is unrecoverable, and what a reader that
  takes this member's absence for `false` gets right is every row it will ever see.

**`runRows` writes it, not `stepRowOf`.** The fact is the Answer's rather than the Step's: a Step knows
its Disposition, and *never reached* is the one it shares with everything behind it.

**And the page and the row ask one predicate**, `Answer.Withholds`, rather than each writing the
comparison out. The sentence beneath the Step table and the member on the row are two renderings of the
same question, so making them literally one call is what stops them drifting — a copy in each surface
would be the two-surface disagreement this ADR exists to close, reintroduced one layer down (ADR-0026).

## Considered options

- **A member on the terminal `outcome` row** — `withheld_step: 2`. It is arguably a Run-wide fact, the
  terminal row is where a Run's own account of how it ended lives, and on MCP it would ride up into the
  structured content beside `dry_run` for free. Rejected: the fact is a Step's, §8 fixes the `outcome`
  row's members tightly, and it would be the one member of that row that is not about the Run's end. It
  also puts a position on one row and the Step it names on another, so a consumer reads two rows to
  learn one thing — and a row cannot point backwards.
- **Carry the page sentence in the MCP text block.** No row changes and no `--json` golden moves.
  Rejected: it fixes one machine surface and leaves the other, which is the drift ADR-0026 exists to
  prevent. It also needs a seam that does not exist — the fact reaches `runPage` and nothing else, so it
  would have to cross the `destination` interface for one command's sake. And `run`'s text block is the
  summary line rather than the page (§9), so there is no table beneath which the sentence would sit.
- **Do nothing, and record it in §13** as a known limit of both machine surfaces, with the
  `completed && dry_run` inference written down as the supported reading. Honest and cheap. Rejected: it
  leaves the boundary of a partial answer to a rule a consumer has to know rather than read, and the
  rule has a case it gets wrong that looks exactly like the case it gets right.

## Consequences

- **§8 states the member** where it states `records` and `expanded`, and §9 states it in three places: the
  `--dry-run` paragraph, where the page's sentence already was; the `run` tool's output sketch; and the
  paragraph beneath it saying what the member is for on a surface with no page.
- **Two goldens moved and no others.** `run/a-rehearsal-stops-at-the-first-effect-json` and its
  `mcp/run/` twin gained the key on one row, byte-identical across the two files. That the other
  rehearsal goldens did **not** move is the assertion, not an omission: `-performs-the-reads-it-reaches-json`
  is the rehearsal that withheld nothing, and `a-halt-inside-a-nested-procedure-json` is the Run the
  world resisted whose *never reached* rows the inference would have misread.
- **The page is unchanged.** No cell moves, no sentence is rewritten, and the rehearsal cases that drive
  the page alone hold their bytes. The member is what the row gained; the prose was already right.
- **`run_show` does not carry it.** The entry has no such member to read back — the position is derived
  while the Run is in flight and stored nowhere — and inventing one there would be a second
  representation of a fact the Journal states by silence (§7, ADR-0043). What `run_show` reads back is
  *never reached* on the Step and on every Step behind it, which is what happened.
- **The text block still does not say it.** §9's text-block table is closed at three rows and `run`'s is
  the summary line. The fact is in the structured half, which is where a machine surface answers.
