# A Probe is not a Run

`hyper` answers a one-off question without a Definition, through a **Probe**: a `read` Operation
invoked against the reserved `local` Target, writing no Record and no Journal entry. It has no
Trigger, no Provenance, no Disposition and no outcome triple, because it is a lookup rather than an
execution. This is the single exception to the rule that nothing is invoked except through a
Definition, and it is bounded so tightly that the rule's reasons do not reach it.

We chose this because the review model dies by volume. A Definition is mandatory so that every Record
has an owner and every effect has an authored claim behind it; but if asking *is this site up* also
requires an artefact, the set of Definitions stops being something a human reads, and the surface
whose reviewability is the entire point becomes noise. The restrictions are what make it a door
rather than a loophole: with no Record written there is no `(Target, Definition, name)` key to need,
so the ownership reason does not apply; and against `local`, which holds no credentials and grants
nothing, the two-key authority check is vacuous rather than skipped — there is no authority to
intersect because there is nothing to protect.

## Consequences

- **A Probe cannot be scheduled, sequenced, or diffed.** It may not carry a Cadence, may not appear as
  a Step in a Procedure, and can never be a diff baseline. It is weaker in every direction than a
  dry-run, which at least writes a marked Journal entry.
- **A Probe is invisible afterwards.** Nothing records that it happened, which is correct for a look
  and wrong for anything you want to know twice. That asymmetry is self-enforcing: ask the same
  question next week and the record will say nothing was observed, because nothing was. Wanting the
  answer *repeatedly* is exactly the point at which a Definition becomes the right artefact.
- **Anything needing a credential still needs a Definition.** *Is this site up* and *when does this
  cert expire* are `local` and are smoothed; *what are my VM IPs* is not, and must be authored. This
  is the correct line rather than an unfinished one — the case that needs a credential is precisely
  the case where the authority review earns its keep.
- **`local` acquires a second job.** It was introduced as the credential-free Target meaning this
  machine and the public internet; it is now also the boundary that makes Probes safe. Widening what
  `local` grants would therefore widen Probes, which is a consequence worth remembering before
  granting it anything.
