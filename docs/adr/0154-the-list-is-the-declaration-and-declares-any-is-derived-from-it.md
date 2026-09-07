# The list is the declaration, and *declares any* is derived from it

`artefact.OperationInfo` carried *does this Operation declare `secret:` output* as a stored `bool`
beside the two members that already answered it. The bool is deleted. The question has one answer,
`OperationInfo.DeclaresSecret`, derived from `Secret` — the list as authored — and read by all three
walks that ask it: the §6 sink gate, the Cadence walk, and the Run's own per-Step reading. Issue #278.

Nothing moves for an artefact that checks clean: no artefact key, no closed-set member, no
`error_code`, no wire member, no golden. **One `check` row moves for one that does not** — a `secret:`
item that is not a name — and the Consequences below say which row and what stays behind it.

## What the state was

Three members held one `secret:` declaration, all written in one loop in `operationInfoFromNode`:

| member | shape | what it was set from |
| --- | --- | --- |
| `HasSecret` | `bool` | `len(secretVal.Content) > 0` — every item, scalar or not |
| `SecretFields` | `map[string]bool` | one entry per scalar item |
| `Secret` | `[]string` | the same scalars, in the Manifest's own order ([ADR-0152](0152-secret-is-named-on-every-surface-that-renders-an-operation.md)) |

`Secret` is information the other two do not carry: the **order**, which is what every surface that
renders an Operation writes and what a set cannot hold. `HasSecret` carried nothing — it was
`len(Secret) > 0` for every input `secret:`'s schema admits, that schema being an array of string.

**And its three readers did not read one member.** `internal/run/repeat.go`'s
`producesSecret` read `HasSecret`; `internal/run/gates.go`'s `secretOutputOf` read
`len(SecretFields) > 0`; `internal/artefact/procedure_graph.go`'s Cadence walk read `HasSecret` again.
Two readings of one declaration, nothing holding them equal, and no record saying which was canonical
— [ADR-0150](0150-a-skipped-members-absent-secret-is-a-line-and-not-a-check.md), which the comments
beside both readings cite, decides something else entirely.

**A test comment stated the invariant and named the wrong member.**
`TestProducesSecret_ReadsTheDeclaration` asserted *what it reads is `HasSecret` and never the set
beside it … the §6 gate reads the first*. The §6 gate read the second. The rule was right and its
subject was wrong, which `CONTRIBUTING.md` calls a defect rather than a nit — and the case fencing it
pinned `producesSecret` to the one member the gate did not read.

**It was not reachable from a Manifest that reaches a Run.** `secret:` is an array of string, so a
non-scalar item is `schema-mismatch`, and both readings in `internal/run` are downstream of a clean
`check` — the gate's own comment says it runs after `check` re-ran. The one thing that could tell the
members apart at a Run was the test case asserting they were different.

The members *did* already disagree in-process, for a Manifest that does not check clean: a mapping
inside `secret:` counted toward `HasSecret` and toward neither of the other two. Only one reader is
positioned to see that — the Cadence walk, which is `check`'s own rather than a Run's — and what it
did with it is below, under what this deletion moves. No surface read `HasSecret` at all.

## The decision, and why it is a deletion

**A fact two members already carry is derived and not stored.** A stored answer to a question its
neighbours also answer is a member that can drift from them; deriving it makes the disagreement
unrepresentable rather than fenced. That is the whole difference between this and the two repairs
beside it: they both leave an invariant standing and add something to hold it, and this one removes
the invariant.

`DeclaresSecret` reads `Secret` rather than `SecretFields`, and off any Manifest read the two answer
alike. `Secret` is the closer of the two to the key: it is the list as authored, and the set is that
list with its order and any repetition discarded. *Does this Operation declare any* is a fact about
what was written, so it is read off the member that still says what was written.

`SecretFields` stays, unchanged and unpromoted. It answers a different question of a different operand
— *is this one field secret* — which is what a predicate's own `field:` and a projection ask of one
name (§12, [ADR-0142](0142-a-declared-secret-field-is-suppressed-on-every-surface-that-renders-a-projection.md)).
Its emptiness coinciding with the list's is a consequence of both being read from one key, not a
second answer.

The rule the test comment stated is kept and moved to where it can be stated about members that
exist: `DeclaresSecret`'s own doc comment, which is now the one place *this question has three askers
and one reading* is written down.

## What was considered

- **Keep `HasSecret` and make it canonical**, moving `gates.go` onto it. Rejected: it repairs the
  comment and keeps the arrangement the comment was worried about. The member still holds nothing
  `Secret` does not, two members still have to agree, and the next reader still has to be told which
  one to read — by a comment, which is what failed here.
- **Keep both and hold them equal with a case over a real Manifest read.** Rejected as the wrong
  fence. Such a case passes for every input the schema admits, so it asserts a property of
  `operationInfoFromNode` and not of the readers, and the divergence it is written against is reachable
  only by hand or through a Manifest `check` refuses — the case would fence the constructor while
  leaving every caller free to pick a member. It is the cheapest repair only if there is a reason for `HasSecret` nobody has written down;
  the ticket asked for that reason and the search for one found none.
- **Three call sites each spelling `len(op.Secret) > 0`.** Rejected, narrowly, and it is what the
  standards review suggested. One member read three ways cannot disagree, so this is correct; a
  predicate is better only because the *why* — one question, three askers, derived on purpose — needs
  somewhere to live, and a package that already carries `IsRunOnce` and `IsOpaqueDestroy` has a
  spelling for exactly this.

## Consequences

- **`OperationInfo` loses a field.** Anything constructing one by hand and setting `HasSecret` no
  longer compiles, which is the intended failure. There were two such constructions in the tree, both
  in the case above: one now names `Secret`, and one is deleted with the member.
- **The test case that pinned the wrong member is deleted**, not rewritten. It fenced a difference
  between two members, and there is one member. The two cases beside it stand, `secret: []` included
  — a list naming no field declares nothing, which is the reading a `bool` and a length agree on and
  the case is worth keeping for the Step it describes rather than for the member it reads.
- **A new case holds the four shapes off a Manifest read.**
  `TestOperationInfo_DeclaresSecretIsOneReadingOfOneMember` reads a Manifest declaring `secret:` with
  two names, with none, not at all, and with an item that is not a name, and asserts the predicate and
  the set agree on all four — which is the deletion's cost, stated rather than assumed. The fourth row
  is the shape the answers used to differ on, so the case covers the input this change moves rather
  than only the three a clean Manifest can carry.
- **One `check` over an artefact that already fails loses a row.** A `secret:` whose item is a mapping
  counted toward `HasSecret` and toward neither of the other two, so the Cadence walk read it as a
  declaration and a recurrence over it drew `cadence-secret-output` beside the `schema-mismatch` the
  item earns. It no longer does: an item that is not a name declares no field, so there is nothing for
  the walk to carry up. The Manifest is refused either way, on the mismatch, and the Cadence row
  returns the moment the item becomes a name. That is the whole of the behaviour this deletion moves,
  and it is the same fact from the other side — the disagreement between the members was only ever
  reachable through a Manifest `check` refuses. It is held by a case of its own,
  `TestCheckProcedureGraph_ASecretItemThatIsNotANameDeclaresNothingToCarryUp`, rather than left to be
  noticed: a walk that stated a declaration it could not read is what moved.
- **No sealed run is owed.** No `check` message is reworded, no rendering changes, and no clause of
  the orientation moves. What an author loses is a row, not a sentence, and it is the second of two
  rows about one mistake: `schema-mismatch` still fires on the item itself, citing
  `operations.<name>.secret[0]` and *expected a string at this position*, which is the row that says
  what to fix. The message that spells `secret: [token]` out for an author is a different check on a
  different artefact — a marking written inside `record: fields:`
  ([ADR-0151](0151-a-fields-value-is-a-path-and-a-sink-nothing-can-fill-is-a-refusal.md),
  `checkRecordField`) — and it is untouched here. This is enforced rather than taught in
  [`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md)'s sense, held by the package
  cases and the existing sink goldens, and it does not join the queue standing against
  `push-credential` ([#277](https://github.com/TheLoomLabs/hyper/issues/277)).
