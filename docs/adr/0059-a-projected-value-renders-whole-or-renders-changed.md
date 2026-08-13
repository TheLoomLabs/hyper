# A projected value renders whole or renders `changed`

The Comparison's `FIELDS` column renders a projected value in full or does not render it at all. There
is no truncated form: a scalar longer than 120 characters, or carrying a single newline, renders
`path: changed` on a two-sided row and its bare `path` on a one-sided one. `--json` carries every value
whole regardless — the elision is the human form's geometry and never a fact either surface states.

This permanently corrects a sentence §8 already carried: "a scalar leaf renders `path: old → new`,
truncated with a marker where it is long". That sentence is also the reading an implementer reaches
unaided, since eliding an over-long cell with an ellipsis is what every table in every tool does, and a
future reader finding truncation removed is owed the reason.

**A truncated pair can render two different values as identical bytes.** `path: old → new` exists to
show a difference. Where the two values agree for their first hundred characters and diverge after —
which is the ordinary shape of a URL, a path, a JSON blob, a command line, a certificate chain — the cell
renders the same string twice, on a row whose `CHANGE` column asserts they differ. That is a surface
stating something other than what happened, which is the failure §8's other rules are written to
prevent, and no marker placed after the cut repairs it. The one-sided rows are not exposed to this and
are not the reason for the rule; they follow it so that one rule governs the column.

**A character budget says nothing about a newline.** [ADR-0052](0052-a-commands-stdout-is-text-never-a-parsed-object.md) makes the built-in
`shell` Provider project `stdout` and `stderr` as text, never parsed, and no cap stands between a chatty
command and the Store — a byte limit being a number `hyper` would be guessing at. So a row in
`YOU DID THIS` or `THE WORLD MOVED` may hold an arbitrarily large multi-line text field that **changed**,
and `hyper`'s own Manifest is the one that does this: there is no Provider author downstream to fix it by
choosing better fields (§3). A truncation rule stated in characters leaves a short two-line value free to
rewrite its own table's geometry, so the newline test is absolute and independent of length.

The budget itself is a guessed constant, and guessing here is affordable in a way ADR-0045 said it was
not for the Store. A wrong Store limit discards evidence irrecoverably; a wrong rendering budget costs a
`changed` where a value would have fitted, and the value is one `hyper show` away. It is stated as a
number rather than derived from the terminal because §8's opening rule is that colour and width are the
only differences between an interactive rendering and one in a CI log — a width-derived budget makes the
two disagree about content.

## Considered options

- **Truncate with a marker, as §8 stated.** Rejected: on a two-sided row it can render a difference as
  identical bytes, and it has no answer to a value containing a newline.
- **Truncate on one-sided rows and elide on two-sided ones**, since only the pair can lie. Rejected as
  two rules for one column, bought with the smaller half of the problem — the multi-line case is
  unhandled on both.
- **No length test at all, disqualifying only newlines and nesting.** Rejected: it renders a
  single-line 200KB `stdout` whole into a terminal row, which is the geometry failure arriving through
  the door the newline test closed.
- **A budget derived from the terminal width.** Rejected: it makes a piped rendering and an interactive
  one carry different content, which §8's first rule forbids.
- **Eliding on the wire too**, on the ground that one renderer produces both forms (ADR-0026). Rejected:
  what ADR-0026 requires the two forms to agree on is which fields moved and under which change name,
  which they do. §8 already fixes the precedent in this direction — every id renders abbreviated on a
  page and whole under `--json`, "which abbreviates nothing" — and a consumer piping to `jq` asked for
  the bytes.

## Consequences

- **The `FIELDS` column has one disqualification rule** — over 120 characters, or containing a newline,
  or nested — and one rendering for all three.
- **A one-sided row's unrenderable field renders its bare `path`.** `changed` is false on a `created`,
  `appeared` or `vanished` row, and the field's name is the whole of what the page can honestly carry.
- **`--json` and the page disagree in volume and never in fact.** A consumer reading `fields` gets every
  value in full, including a `shell` Observation's whole `stdout`.
- **A `shell` `read` whose output moved renders `stdout: changed`** and nothing more, on every
  Comparison. What it actually said is `hyper show`'s, which is where the Store's own bytes are read
  back.
