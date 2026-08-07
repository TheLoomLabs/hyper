# A Store file's schema version is its own, not the branch's

Every file in the Store carries its own schema-version integer, not the branch. `hyper` reads any file
at or below the schema version it knows, and Refuses, never guesses, on a file written by a schema
version newer than its own. ADR-0020 already established that this integer is independent of the
`hyper` release version recorded in Provenance and is the system's only mechanical compatibility fact;
what it left unstated is where the integer lives and what "compatible" means in practice — per file
rather than per branch, and read-down-refuse-on-newer rather than best-effort.

Provenance already records the `hyper` version on every file it writes, so a future reader will
assume that field already answers "can I read this," and that a second integer is redundant
bookkeeping nobody pruned. It is not the same fact, and treating it as one is the exact mistake
ADR-0020 already heads off; what remained open was whether the integer that does answer the question
lives once per branch or once per file, and that is this decision.

We chose per file because the first shape for this idea put a single version marker in a branch-level
introduction file, borrowing the pattern a Manifest already uses for its own explicit schema version
and its own unknown-key rejection. The Store's append-only rule makes that reading exact everywhere
but one place: a branch-level marker changing shape means rewriting the one file that describes every
other file in the branch, which is the same class of edit the append-only rule exists to forbid,
arriving through the file that looks like the least dangerous one to touch. Making the version per
file instead makes append-only literal rather than aspirational: every file, forever, states the shape
it was written in, and a schema change adds new files rather than editing an old one.

## Considered options

- **One schema-version field in a branch-level introduction file.** The original shape, correct on
  every point but one: it is a marker that must be rewritten the moment the schema changes, and
  nothing in an append-only branch may be rewritten.
- **Reuse the `hyper` version already carried in Provenance as the compatibility signal.** Attractive
  because the fact already exists on every version file, and wrong on its own terms: it conflates
  *what code wrote this* with *what shape is this*. Reusing it would make every patch release,
  including ones that touch no file's shape, read as a potential format change to anything scanning
  the Store, and would let ADR-0020's hard version pin be softened later by an implicit semver
  argument nobody reviewed.

## Consequences

- **`hyper` must read every schema version at or below its own, forever.** There is no in-place
  migration, so the reader accretes format-handling code rather than the Store accreting migrations of
  its own history.
- **A file newer than the running binary is a Refusal, not a best-effort read**, on the same
  unknown-key grounds a Manifest already applies to a load it does not recognise: guessing at an
  unfamiliar shape is exactly the failure mode a schema version exists to catch.
- **The schema integer moves far less often than the release version.** It increments only when a
  file's shape changes, which is rare enough that most `hyper` releases touch none of them.
- **This is not the version-skew guardrail.** The exact-match version pin is what keeps two
  environments running the same code against one Store; the schema integer is what keeps every
  environment able to read what an older one wrote, including after that pin has moved on.
- **Reading old history still needs the binary that wrote it, not merely one that recognises the
  schema.** A file `hyper` can parse is not a file `hyper` can act on the same way a different version
  once did — the schema integer is a compatibility fact about shape, and says nothing about behaviour.
