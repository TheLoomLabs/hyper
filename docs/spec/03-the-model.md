# §2 — The model

A Provider is a Manifest and nothing else, with no implementation behind it to review (ADR-0004).
Every Operation it exposes carries a declared Kind, stated in the Manifest and never inferred from the
Operation's name or any other property `hyper` could derive on its own (ADR-0025). The Kind
enumeration is closed and lives in §12.

Five artefacts are authored and reviewed, and together they say everything about what runs and what it
may touch. A Definition claims a Provider's Kinds against named Targets, with no argument value of its
own. A Procedure sequences Steps in the order written, with no dependency edge anywhere (ADR-0002) and
no grouping level beneath it; it composes with other Procedures by invoking them, not by containing
their Steps. Each Step is where a Definition, an Operation, and a Target meet, and where the argument
values a Definition withheld are supplied. A Target declaration is the reviewed half of a Target — the
Kinds it accepts, the Capabilities it grants, the endpoint it names — holding no credential; its
counterpart, the Target credentials, is resolved once per Run and never enters the repository or comes
to rest inside `hyper` (ADR-0007). A Repository declaration governs the repository as a whole rather
than any one artefact within it: which version of `hyper` may act on it, and how long Records are kept.

A Target is therefore the unit of both blast radius and credentials at once: every Step binds exactly
one, and a Definition limits which it may bind. `local` is the one reserved Target holding no
credentials, and it is bound by the same rule as any other — it declares by name the hosts it grants,
so no Target reaches without limit (ADR-0024).

What a Step produces is a Record, and every Record is an Observation or an Asset — two Record types,
never a status field on one (ADR-0025). `hyper` is accountable for an Asset, since it created it, and
for nothing an Observation merely describes; neither ever becomes the other. A Record's identity —
`(Target, Definition, name)` — excludes the Run that wrote it, so a Definition invoked twice against
the same Target writes a further version into one series rather than starting a new one (ADR-0025); the
Run itself survives only as Provenance, carried by every version: the Definition revision, Manifest
digest, Extension digest, repository revision, and the version of `hyper` that wrote it.

A Run is a single execution of an Operation or a Procedure. A Probe is not one: it is a `read`
Operation invoked against `local` without a Definition, and having written no Record it has no
Provenance, no Disposition, and no Trigger — it cannot be scheduled, sequenced into a Procedure, or
serve as a Comparison baseline (ADR-0009).
