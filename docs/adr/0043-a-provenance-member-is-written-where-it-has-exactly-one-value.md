# A Provenance member is written where it has exactly one value

Provenance is one field set projected by scope rather than a field set per file. A member is written at
the level where it has exactly one value and omitted from every level where it has none: `hyper_version`,
`repo_revision` and `repo_dirty` on `run.json`; `definition_revision`, `manifest_digest` and
`origin_digest` on a Step file; the whole of it on every Record version. Nothing at Run level names a
Definition, so a Procedure whose Steps span two of them has no revision it must invent.

We chose this because the unaided reading is that `run.json` carries a `definition_revision`, and this
corpus reached that reading four times before anyone noticed a Procedure has no single Definition to
name it after. §8's `hyper run --json` stream emitted one `provenance` row carrying all five members for
a Run whose own review example spans `uptime` and `hetzner-staging`; §9 called it *the Run's own
`provenance`*, called a `show` rendering *the entry's Provenance*, and twice specified a surface as
emitting §8's row *unchanged*. The rule was already latent in ADR-0034, which fixed the predicate instant
at the Run's start on the ground that it is "Run-wide — it sits with `repo_revision` and `hyper_version`
in Provenance: one Run, one reading", naming the same two members `run.json` now carries and drawing the
same line a ticket before anyone stated it.

What makes the Step file's copy load-bearing rather than convenient is a built-in Provider.
`manifest_digest` for one is taken over bytes embedded in the binary, which have no blob in the
repository at all, so `repo_revision` cannot resolve it and there is nowhere else the fact could be
recovered from. The same holds one step weaker for `definition_revision` on a Run that recorded
`repo_dirty`, which is exactly the case §7's reaper already falls back on.

This is not a second representation of one fact, which is the shape this specification refuses
elsewhere. A second representation can disagree with the first; this one cannot, because every member has
one value and appears only where that value is single. What varies across the three files is which
members are present, never what a present member means.

## Considered options

- **A set of revisions at Run level.** `run.json` carries `definition_revision` as an array where the
  Procedure spans several. Rejected because it makes Provenance's *shape* depend on the Run: a reader
  and a schema must handle a member that is sometimes a string and sometimes a list, and the list has no
  ordering anyone could act on. It also answers the wrong question — a reader wanting to know which code
  ran a given Step is handed every Definition the Run touched and left to guess.
- **The member absent everywhere but a Record version.** Rejected because it weakens the reason
  `run.json` carries Provenance at all. A Refusal writes no Record, and the Journal entry is then the one
  place the code that would have run is named; dropping the Definition-scoped members from the Journal
  entirely leaves a refused Run unable to say which Definition it declined.
- **Run-level Provenance as a smaller field set of its own.** Rejected as the second representation
  above. A second field set is a second thing to keep in step, and the two can drift; a projection of one
  set cannot.
- **Every file self-describing, carrying the whole of Provenance.** The instinct an implementer reaches
  for, and rejected on locality: a Step file sits beside `run.json` in the same entry directory, so
  restating the Run-wide members there gives one Journal entry two copies of `hyper_version` for no
  reader who could not already find it. A Record version restates them because it sits under a Record
  path with no entry beside it, which is the distinction §7 now states.

## Consequences

- **Hard to reverse, which is why this is written down.** No file in the Store is ever rewritten
  (ADR-0011) and migration in place is impossible, so a Journal file's Provenance is what it is forever
  and a change of shape can only arrive as a schema version on a Store that accretes format handling
  (ADR-0028).
- **The wire splits the way the Store does.** §8's `provenance` row becomes one Run-wide row plus one per
  Step file written, distinguished by `step`. One renderer produces both forms (ADR-0026), so `run` and
  `show` emit the same shape and there is no second rendering to keep in step.
- **A member with no value at a level is absent, not null.** §7's absence rule already says so; the
  visible instance is `origin_digest` on a Step whose Provider is built in or locally authored.
- **A Procedure's own revision was left where it stood, and has since been decided.** The rule says
  where a member goes once it exists and said nothing about whether Provenance should have a Procedure
  revision at all. What it supplied was the test that decision had to answer: a Run spans nested
  Procedures as one Run (§6), so a Procedure revision is single-valued at Run level only for the
  top-level one. ADR-0048 takes exactly that reading — `procedure_revision` is the top-level Procedure's
  and nothing beneath it — and this rule placed it with no amendment.
