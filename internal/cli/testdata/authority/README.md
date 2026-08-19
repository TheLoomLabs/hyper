# One relation, read from both its ends

`repo/` is the repository the `AUTHORITY` table's own cases read (issue #121).
It exists because the two repositories beside it cannot show what this one
shows: `five-artefact-demo/` checks clean, and three of the cells this table can
carry are only reachable where a claim resolves to nothing or to something
unreadable, a grant is narrower than the claim, or a file in `definitions/` did
not load at all.

The relation is §5's two-key check rendered: a Definition claims Kinds and
Targets, a Target declaration accepts Kinds and grants Capabilities, and what an
Operation may reach is what both name. The artefact under review supplies one
end of it, and which end decides the filter and nothing else
([ADR-0069](../../../../docs/adr/0069-authority-is-one-relation-read-from-whichever-end-the-artefact-supplies.md)).

## What is in it

| artefact | why it is here |
| --- | --- |
| `definitions/things.yaml` | claims four Targets — one that grants everything it claims, one that grants less, one that will not parse, and one that is not there |
| `definitions/things-observed.yaml` | the second claimant on `staging`, so the right end's filter has more than one row to sort |
| `definitions/half-written.yaml`, `definitions/also-half-written.yaml` | will not parse, so two rows that would have been discovered are not |
| `targets/staging.yaml` | accepts every Kind, and is the Target both Definitions claim |
| `targets/observed-only.yaml` | accepts `read` alone, so the intersection with a `mutate destroy` claim is empty |
| `targets/wont-parse.yaml` | is there and will not parse, which renders as the same absence as a name that is not there at all |
| `targets/unclaimed.yaml` | nothing claims it, which is the explicit empty state |

The Definitions come in a pair — `things` beside `things-observed` — because a
Definition observes or effects and never both (ADR-0032).

## The cells the demonstration repository cannot reach

```
hyper review definitions/things.yaml
```

| row | cell | what it is |
| --- | --- | --- |
| `nowhere` | `unresolved` | a `targets:` member naming nothing — the row stays, and two of its cells empty |
| `wont-parse` | `unresolved` | a member naming a file that is there and will not parse: the same word, in the same cells |
| `observed-only` | `—` | a set that is supplied and names nothing: the claim reaches this Target for no Kind at all |
| `staging` | `m d` | the intersection, in the initials §8's own block renders it in |

`unresolved` is the gutter's own word, and the two rows carrying it render
identically on purpose: a supply that resolved to nothing and one that resolved
to something unreadable differ in nothing this table can act on, and what a
reader does about either is `check`'s row on that file's own line (ADR-0064).

Dropping either row would be the omission ADR-0026 forbids — a Definition
claiming four Targets and rendering two says the other two were never claimed.

## The end the table is not usually read from

```
hyper review staging     → the two Definitions that claim it, sorted, and the count beneath
hyper review unclaimed   → the header, and an explicit empty state
```

`staging` is the rendering an unaided reading withholds. ADR-0026 introduced the
table as *built from a Definition and a Target declaration together* and §8
states its row rule inside a paragraph about a Definition's `targets:`, so the
natural implementation builds a Definition-side table and renders nothing here —
on the screen whose gutter marks what it grants and where nothing says who took
the grant.

Both cases end in `2 definitions did not load · hyper check`. The row set on a
Target declaration is discovered rather than authored, so a discovery failure
removes a row outright where every other absence on this screen leaves a marked
one — and the review still exits `0`, §9's exit `1` being for the artefact under
review and a fault in a file the reviewer did not ask about being `check`'s.

`unclaimed` is the line between absent and empty. An edit to any file in
`definitions/` puts a row there, so the block renders and says there is none: a
granted `destroy` with no claimant is either a Target awaiting its Definition or
one whose Definition was deleted (ADR-0012).

The sentence is the Target declaration's own. Each end of the relation has one,
because *no Definition claims this Target* and *this Definition claims no
Target* are two different facts and one sentence for both would state the wrong
one on two of the three screens. A Manifest and a Repository declaration have
none and render no block at all, which the demonstration repository's own cases
hold: no edit to any file produces a row there.

## Why it is a repository and not a case

Four review cases read it — the Definition, the Target declaration, the same as
NDJSON, and the Target nothing claims — on the argument `five-artefact-demo/`
already carries: a copy per case is a fixture edit that reaches one golden file
and not another, and this table's whole subject is that one relation renders the
same facts from either end. The cases stay in `review/`, named for the command
their argv invokes, and name this repository with the `--repo-dir` an operator
would type.
