# Two code commits, and a Journal seeded against both

The fixture [`THE CODE MOVED`](../README.md) is driven at where a golden cannot
hold it (issue #171). It carries **no `argv`**, so `TestGolden` never picks it
up: what is driven against it is decided in
[changes_code_test.go](../../../changes_code_test.go), one Comparison at a time.

- **`code-baseline/`** is the earlier revision, committed **below** `repo/`.
  It is the harness input this ticket landed: the eight
  artefact-authored classes and the catch-all's whole count read bytes at *two*
  revisions, and a fixture that makes one commit has one revision to read.
- **`repo/`** is the later revision, and the working tree the command stands
  in. The two commits' trees are exactly the two directories, so
  `targets/production.yaml` — which only `repo/` holds — is a creation the diff
  counts like any other.
- **`git`** materialises the repository. The Store branch is seeded at run
  time, the entries naming commit ids only the run knows.

A `repo_revision` is a commit, and a commit id is a function of the tree, the
message, the identity and the dates. Nothing in this directory names one, and
nothing here has to be regenerated when one moves.

## What the two revisions move between them

One edit per class, so that every one of §12's nine emits at least one row and
the table can be asserted whole:

| class | where it moves |
| --- | --- |
| declared Kinds | `definitions/ci-keys.yaml` gains `destroy`; `targets/production.yaml` arrives declaring three |
| Target set | the Procedure's envelope gains `production`, the Definition's bindable Targets gain it, and the retiring Step's `target:` moves onto it |
| Bounds | the retiring Step's `bound: 5` is taken away — an absent `bound:` is unbounded, and the cell renders `–` rather than naming what the absence means |
| Cadence | `0 0 1 * *` → `*/5 * * * *`, ADR-0005's own pair, whose ≈8,800× is legible in the two rates and in nothing else on the page |
| selector | the `values:` selector gains a member **at the end**, and the predicate selector gains a conjunct |
| required Capabilities | the Manifest gains `shell` |
| the credential source | `staging` moves from `TAILSCALE_STAGING` to `TAILSCALE_PROD` |
| the Operation set | the Manifest gains `get_key` — **fifteen lines of block, one line of value**, which is what the catch-all's subtraction is asserted on |
| the digests | all four that a Step file and a `run.json` supply |

`hyper.yaml`'s `retention:` moves too, and it is deliberately **not** a class:
its lines fall to the catch-all's count, which is the word *other* holding.
`providers/unread.yaml` is there for the same argument one step further out —
its `capabilities:` moves and **no Step file names it**, so it draws no row at
all: which Manifests a Run read is the Step files' `provider`, and a table that
enumerated the Manifests any other way would be reporting a Provider nobody
ran.

The seeded Journal gives each Run the revision it read, and the retiring Step
the Target that revision's Procedure binds — so the two entries are the two
Runs those two revisions actually performed.
