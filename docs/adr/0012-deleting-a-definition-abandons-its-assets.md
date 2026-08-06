# Deleting a Definition abandons its Assets

Deleting a Definition that still owns live, un-tombstoned Assets is legal. `hyper` neither blocks the
deletion nor destroys what it owned. The Assets become **Orphaned Assets**: still recorded, still
`hyper`'s account of things it created, and — because Expansion needs a Definition — unreachable by
anything `hyper` can now do. They are reported for as long as they stand.

We chose this because the two alternatives are worse in ways this project has already ruled on.
Cascading destruction would make deleting a text file a destructive Operation with no Step, no Bound,
no declared Kind and no Target check — every guardrail in the safety model bypassed by an act that
does not look like running anything. Blocking the deletion would make the Store's contents veto an
edit to the repository, which inverts the direction authority flows in: the artefact is what gets
reviewed, and the record is an account of the world, not a lock on the code.

## Consequences

- **The deletion is visible where edits are visible.** Removing a Definition is a code change, so it
  appears in `THE CODE MOVED` alongside every other artefact change.
- **Orphaned Assets are permanently reported**, not reported once. Silence would let a forgotten VM
  become invisible by way of a tidy-up commit, which is exactly the failure the record exists to
  prevent.
- **Recovering one means restoring the Definition**, or authoring a new one that declares the same
  Target and can name the resource by literal identifier. There is no adoption path, because adoption
  is reconciliation and the domain model declined it.
- **This closes the under-reach half of the orphan risk.** A mistagged Asset left standing by a
  selector is still unowned, but an Asset left standing by a deleted Definition now has a name, a
  rendering, and a rule.
