# `AUTHORITY` is one relation read from whichever end the artefact supplies

`AUTHORITY` states one relation — a Definition's claimed Kinds against a Target declaration's accepted
Kinds, their intersection, and the `destroy` Operations the Definition names (§5). **The artefact under
review supplies one end of it, and which end decides the filter and nothing else.** A Definition
supplies the left end and renders a row for every Target it claims; a **Target declaration supplies the
right end and renders a row for every Definition that claims it**; a Procedure supplies neither and
binds pairs, rendering one row per distinct `(Definition, Target)` its Steps bind. A Manifest and a
Repository declaration are members of no pair, so the table has no end to read from and **does not
render at all** — where a Target declaration nothing claims renders empty.

Both halves are the reading a competent implementer does not reach unaided, and they fail in opposite
directions.

**The table is not Definition-side.** ADR-0026 introduced it as "built from a Definition and a Target
declaration together", and §8 stated its row rule inside a paragraph about a Definition's `targets:`
list. Every unaided reading of that builds a Definition-side table and renders nothing on `hyper review
staging` — leaving the artefact whose one-line credential edit §12 makes a named change class with no
surface answering *who reaches this Target, with what*. Its gutter marks what it grants; nothing on
that screen says who took the grant, and that is precisely the fact ADR-0026 says no single file's
gutter could ever show. The relation is symmetric by construction: §5's two-key check is an
intersection, and an intersection privileges neither operand.

**Absent and empty are not the same absence.** §12 renders an empty `FLAGS` block on the four
non-Procedure artefacts, and the unaided generalisation is that every rendering announces itself
everywhere. That argument is *nothing to report* against *the renderer had nothing to say*, and both
readings are live only where an edit somewhere in the repository would produce a row. On a Target
declaration one would, so the empty block earns its line — a granted `destroy` with no claimant is
either a Target awaiting its Definition or one whose Definition was deleted
([ADR-0012](0012-deleting-a-definition-abandons-its-assets.md)). On a Manifest none would, and an empty
block there asserts a supply that does not exist. What keeps the silence honest is that the roster is
written in §8 beside each rendering, which is how `envelope` and `unbounded` have been Procedure-only
since §12 was written without either being announced on the other four screens.

The two halves are one decision. Reading the relation from either end is what makes the roster three
artefacts rather than one; the roster being short is what forces the absence to be ruled rather than
discovered.

## Considered options

- **`AUTHORITY` renders only where the artefact under review is a Definition.** Rejected: it is the
  Definition-side reading above, and it withholds the table from the artefact that most needs it while
  leaving ADR-0026's justification — the fact no single file's gutter can show — true of a screen that
  does not render it.
- **A Target declaration renders the table's header with an empty body.** Rejected as a rendering that
  states the relation exists and then declines to read it. If the rows are assemblable the table
  assembles them; if they are not, the header is a promise with nothing behind it.
- **A Manifest renders the table two hops out** — every Definition naming this Provider, crossed with
  the Targets each claims. Rejected twice. The Manifest appears in none of the table's columns, so the
  rows are about pairings the artefact under review is not a member of; and it traverses the
  Manifest→Definition edge backwards, which is the one edge this screen does not traverse
  ([ADR-0064](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md) leaves a
  Definition with a missing `provider:` reviewing complete and unmarked on exactly that ground).
- **A Repository declaration renders what its pin governs.** Rejected: that is an inventory of the
  repository rather than an intersection of two claims, and no discipline ADR-0026 fixes admits one.
- **Every rendering announces itself on every artefact, empty where it has nothing.** Rejected above:
  it makes the screen assert a supply that no edit can fill, and it retires the distinction that lets
  §12 hold three Procedure-only flag names without a word on the other four screens.
- **Rows sort in step order on a Procedure.** Rejected: step order does not exist on two of the three
  artefacts that render the table, so it is three rules for one table; and reading down the marker
  column already *is* the step table (ADR-0026), which makes a step-ordered table beneath it a second
  copy of an ordering the reviewer has, where a sorted one is a second index into the same rows.

## Consequences

- **The roster is written per rendering and the matrix falls out.** §8 already states the gutter's
  roster and the header's; `AUTHORITY` gains the one it lacked, and the chapter now says at the top
  that no rendering is silently withheld on any artefact. A matrix was rejected as a second statement
  of the same facts beside the prose that carries them.
- **The row set on a Target declaration is discovered rather than authored**, which is the first
  surface in the tool that answers a question the artefact under review did not ask. It costs nothing
  at load: `check` already loads every artefact and matches byte-exact on its own `name:`, never on
  whether an `open` succeeded (ADR-0064), so the direction of the lookup is free and identical on a
  laptop and a runner.
- **A discovery failure is named beneath the table.** A row that could not be discovered has no cell to
  carry `unresolved`, so where a file in `definitions/` did not parse the table terminates with
  `3 definitions did not load · hyper check`. `review` does not decline for it — §9's exit `1` is for
  the artefact under review — and without the line the one table would lie by omission in the one way
  ADR-0026 forbids. A cell that *does* exist carries `unresolved` whether its supply resolved to
  nothing or resolved to something unreadable: the two differ in nothing this table can act on.
- **The row count on a Target declaration is bounded by what other authors wrote**, not by this one.
  Nothing on a review is sized to the terminal (§8), so a Target claimed by forty Definitions renders
  forty rows. That is accepted as the honest rendering of forty claims on one Target rather than
  capped, on ADR-0045's argument against `hyper` guessing at a number.
- **ADR-0026 is not amended.** Nothing is withdrawn and no discipline is added: reading a symmetric
  relation from its other end is the second discipline doing what it already did, and "built from a
  Definition and a Target declaration together" is true of every row this decision admits.
- **§9 and §12 do not move.** The `authority` rows of `review --json` keep their keys and emit none on
  the two artefacts where the table is absent. §12 enumerates vocabularies whose names travel as
  values, and nothing travels here — a review does not decline, and the header's absences needed names
  because they render as text in a value position. A rendering that emits no rows emits no rows.
