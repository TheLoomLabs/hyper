# An effectful shell Operation records that it ran and not what it printed

The built-in `shell` Provider's three effectful Operations — `mutate`, `mutate_once` and
`mutate_skip_if_recorded` — project `exit_code` and nothing else. `read` still projects `stdout` and
`stderr` beside it, and `destroy` and `destroy_once` carry no `record:` block at all, as every
`destroy` does (ADR-0037). The identity is unchanged on all four that project: a shell Record is
named by the command that produced it.

We chose this because of what the two Kinds write. A shell `read`'s output is an Observation, and
Compaction reclaims interior versions of one; a shell `mutate`'s was an Asset, and Compaction never
removes an Asset, so a command's stdout was on the branch permanently and by construction. §13 named
that asymmetry as a cost and named its worst case beside it — *a command that prints a credential puts
it in the Store, and nothing notices* — and the half that made a printed credential unrecoverable
rather than merely present was the permanence, not the printing.

The second reason is that the Record was an overclaim. `shell` is the Opaque Capability: `hyper` is
the Provider author and knows nothing whatever about the command, which is why the built-in declares
no `secret:` and could not usefully declare one, and why a Step reaching it draws the review's
`unbounded` flag whatever `bound:` it carries. Folding that command's stdout into an Asset — the
Record type meaning *`hyper`'s own effect reached this and `hyper` is accountable for it* — is the same
overclaim at the Store, `hyper` vouching permanently for bytes it cannot describe. An exit code is a
fact about the call that `hyper` does own.

## Considered options

- **Take `stdout` from every shell Operation, `read` included.** Considered first and rejected: it guts
  the case that was never the problem. Capturing a command's output is often the whole reason for
  reaching for `shell`, and a `read` writes the one Record type Compaction can reclaim — the volume is
  bounded by a policy an author may state, where the permanence was bounded by nothing.
- **Leave it and rely on `retention:`.** Rejected: `retention:` reclaims interior Observation versions
  and touches no Asset at all, so it is not the lever. Permanence is not a volume problem.
- **Let Compaction remove interior Asset versions whose fields came from an opaque source.** The
  surgical answer, and rejected as the most dangerous of them: it puts a deletion rule on the one
  Record type whose whole guarantee is that nothing deletes it, and ADR-0001's *no bypass* is exactly
  the shape of argument that would then have to be re-made for every future exception.
- **Recognise a credential in the output and refuse or mask it.** Rejected on ADR-0007's ground, and it
  is worth restating: suppression here is positional, and a command's stdout is not a position `hyper`
  owns. A scan would also be blind to the case §13 names, which is a secret the repository never named.

## Consequences

- **This is a breaking change for a Procedure whose Requirement predicates on an effectful Step's
  `stdout`.** The field is no longer projected, so the path resolves to nothing and the predicate has
  no root. What is available instead is `exit_code`, and what a Procedure wanting the output should
  reach for is a `read` Step beside the `mutate`.
- **An Asset still says which command ran and how it exited.** `identity: $.command` is unchanged, so
  what the Record loses is the transcript and not the fact — *what the command did* survives, *what it
  printed* does not.
- **The volume half is untouched.** A chatty shell `read` still writes its stdout into an Observation,
  and Compaction reclaims those versions only where the repository declared a `retention:` — omitted,
  nothing is ever removed. No cap stands between a chatty command and the Store, and a byte limit
  remains a number `hyper` would be guessing at (ADR-0045).
- **`git` history still holds what was already written.** History is not editable, so this bounds what
  the branch's live tree accumulates from here and reclaims nothing already on it. For a credential a
  command printed before this, rotation remains the only remedy that is one.
- **The Manifest digest changed, and Provenance carries it.** Every Record and Journal entry written
  against the built-in from here names a different `manifest_digest`, which is the mechanism working:
  the Provider is data, and its bytes are the code that ran.
