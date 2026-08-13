# One supply is stated once, and the member it silences is not omitted

The review header's members each render from one supply (ADR-0063). Where one supply answers two of
them, the header states it once and the second member does not render. That is **subsumption** and not
omission, and the difference is which member has a supply of its own: an omitted member has one and is
dropped, a silenced member is one the line has already answered.

One artefact answers to it. A built-in Provider's Manifest ships inside the binary and has no file in
the repository (§4, §11), so it has no path, and no Run could have recorded a revision of what has no
file — one absence, reaching two members. §8 renders the range's named absence,
`no baseline — shell ships in the binary`, and `path` is silent beside it, that sentence already saying
where the bytes are. The rule keys on **there being no file**, not on the artefact's kind and not on its
being built in: ADR-0067's decomposition, applied to the other half of the same line.

The reading an implementer reaches unaided is to render something in that column, and the corpus's own
discipline is what drives them there. §8 states four times that this screen may not lie by omission
(ADR-0026) — an absent `FLAGS` block is ambiguous between *nothing to flag* and *the renderer had
nothing to say*, a dropped `AUTHORITY` row says the Target was never claimed, a blank marker cell says
`read`, a blank change column would excuse the artefact. An implementer carrying that into
`hyper review shell` concludes a blank is the one thing they may not draw, and mints `<built-in>/shell`.
The wrong answer is produced *by* the rule rather than in spite of it, which is why it has to be decided
rather than left to the member list.

What refuses it is a distinction the corpus already performs and has never stated. §8 argues it for the
range and *last ran* — "one fact in two notations", so the header says it once and `last_run` goes
absent on the wire for all three absences — but argues it there as an observation about one Journal
entry rather than as a rule a member list can be read against. Stated, it does two things: it lets an
enumerated list hold a member that sometimes does not render without the list becoming optional, and it
keeps a renderer that drops a member no other supply answered a defect rather than a judgement call.

## Considered options

- **The Provider's name, `shell`.** Rejected: it repeats the name the sentence beside it carries and the
  marker column already typed, so it adds nothing a reader does not have. Worse, a bare token in a column
  whose other four values are repository-relative paths reads as a file — the one thing it is not.
- **A synthetic locator, `<built-in>/shell`.** Rejected, and it is the reading this decision exists to
  refuse. It is a new shape of value on a member whose other values are openable files, and what it names
  is something no `check` cites and no editor opens: honest about there being no file only to a reader
  who already knew.
- **The binary and its version, `hyper 0.4.1`.** The most defensible of the three — it is literally where
  the bytes are, and it is the coordinate a reader acts on, `hyper_version` being the pin. Rejected on
  doubling: the sentence beside it already says the bytes are in the binary, and one line saying so twice
  is exactly the failure §8 refuses between the range and *last ran*. It also quietly shrinks §13's
  stated limit by moving the tool's coordinate onto a screen whose subject is the artefact.
- **Making `path` optional on the header's member list.** The same rendering under a worse rule, and
  rejected for the rule. An optional member is one a renderer may drop; a subsumed member is one another
  supply answered. Only the second is checkable against a rendering, which is what ADR-0063 wrote the
  list down for.
- **Rendering the path's width as blank run-up, with the range's sentence in its own column.** Rejected:
  whitespace where a member was is omission wearing a rendering, and it is the shape the gutter's marker
  cell is forbidden for. The line collapses to its one field instead.

## Consequences

- **ADR-0063's member rule and its written list are unchanged.** This is the rule under which the list
  survives a member that does not always render; nothing is added to it and nothing becomes conditional.
- **The class is *no file in the repository*, and it has one member today.** ADR-0039 bounds the built-in
  roster by a criterion rather than a list, so the class grows only where the reserved Capabilities do.
- **The wire follows without a second key.** `review --json`'s `artefact` row omits `path` exactly where
  `baseline_absent` is `built-in`. The two are biconditional, so the name already on the row is the
  discriminator, and §8 states the omission rather than leaving it to fall out of which artefact happens
  to have a file.
- **#70's sentence is untouched**, byte for byte. It is the one supply, and rewording it to carry the
  path's half would break the lead-in §8 fixes across all four absence sentences.
- **ADR-0057 is untouched.** Its subject is that one side of the range renders and never both; a built-in
  has no baseline to have drawn, so only §8's word for where the bytes came from widens — the working
  tree for four artefacts, the embedded bytes for the fifth.
- **§12 gains nothing.** No name travels: `path` is a wire key and its absence is a rendering, and §12
  enumerates neither.
- **§13's limit is stated completely rather than extended.** `hyper review shell` withholds two things
  and not one — what changed, and where to go — and they are one limit seen from two sides.
