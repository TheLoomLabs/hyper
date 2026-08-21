# A version carrying no fields is not a Tombstone's alone

A Record version writes no `fields` key where **every** path its Manifest
projected resolved to nothing. It is an ordinary version and it is minted like
any other, and what tells it from a Tombstone that carries none is the
`tombstone` marker beside it rather than the absence itself.

## The problem this decides

§7 states the absence in one sentence, about the Tombstone that opens the series
it ends:

> Such a Tombstone is the series' first version and it carries **no `fields`**
> […]. A Tombstone is the one version whose `fields` can be missing for no other
> reason, so the absence needs no marker beside it.

Read as a rule about which versions may lack the key, that sentence says *only a
Tombstone may*, and the implementation read it that way: a version arriving with
no fields was `hyper`'s own arithmetic being wrong, and the encoder refused it
outright.

Two other sentences, in two other sections, put an ordinary version in exactly
that state.

§6, on a projection that does not resolve:

> A path a recorded field is read from resolving to nothing is absence: the
> field is not written on that version […] and it is not silent — the bytes
> moved, so a version is minted and the field going quiet renders as a change
> like any other.

And §12, twice, on the `shell` Capability. The response object where the command
**could not be started at all** is `command` and nothing else; and the built-in
`shell` Provider's projection is `identity: $.command` with `exit_code`,
`stdout` and `stderr` as its three fields — none of which is `$.command`.

Put together: a `shell` `read` Step whose binary is not there records an
Observation (§6 — a `read` never halts, and the attempt is the answer) whose
three projected paths all resolve to nothing (§6 — each is an absence) and which
therefore carries no `fields` at all. The specification requires that version to
exist and, on one reading of one sentence, forbids it from being written.

The conflict is not reachable before the `shell` Capability lands. Every `http`
projection that has ever been written projects `$.host`, which survives a call
that got no answer for the same reason `command` does — so *no answer* has
always left at least one field behind, and the sentence was true of everything
in the corpus at the time it was written.

## The decision

**An ordinary version may carry no `fields`, and the key is simply absent.**

- The absence means what it looks like: `hyper` made the call, and every path
  the Manifest projected read nothing back off the answer. It is §6's ordinary
  field absence applied to all of a projection at once rather than a shape of
  its own.
- **The two absences are never confused**, because the Tombstone's is not what
  identifies it: `tombstone: true` is a written marker, so a reader asks the
  marker and never the key. §7's *the absence needs no marker beside it* is
  about the Tombstone needing no second marker for its **fields** — not about
  the fields' absence identifying a Tombstone.
- **Such a version is minted like any other.** The bytes moved — there was no
  version before it, or the one before it carried fields — so a Record versions
  on the change exactly as §7 states, and the fields going quiet renders as a
  change like any other (§8).
- **`exists` and `absent` read it unchanged.** A predicate over such a version
  finds every field absent, which is the answer, and no operator needed a new
  case.

## Considered options

- **Project `$.command` as a field on the built-in `shell` Provider.** It would
  guarantee at least one field on every shell version and the conflict would
  vanish. Rejected: §12 states the built-in's Manifest in full and this is not
  it, and the field would be a second copy of the Record's own name — which is
  the duplication §7 refuses everywhere else, an identity restated inside the
  content it identifies.
- **Record nothing where a command could not be started.** Rejected: it
  contradicts §6 outright. A `read` never halts on what came back and the
  attempt is the answer; a Step that recorded nothing would be indistinguishable
  from one that never ran, on the one Capability where *the binary is not there*
  is the ordinary operational finding.
- **Write `fields: {}`.** Rejected: it is a second spelling of absent, and §7's
  canonical encoding suppresses an empty mapping everywhere else. Two encodings
  of one state on a branch nothing rewrites is the mistake ADR-0011 is written
  against.
- **Mark it — a key saying *this projected nothing*.** Rejected: it is the
  catch-all bucket ADR-0017 closed, arriving one file over. Which paths resolved
  to nothing is readable from the Manifest and the version side by side, and a
  marker would be `hyper` writing down its own reading of an answer rather than
  the answer.

## Consequences

- **§7 gains a sentence and loses none.** The Tombstone's absence still means
  *hyper destroyed this and never observed what it was*, and it is still the
  only absence that means that.
- **The encoder no longer refuses the shape**, and the decoder no longer
  requires the key on a non-Tombstone. Both were reading the sentence above as a
  rule about which versions exist rather than about what an absence means.
- **No closed set moves.** No `error_code`, no Disposition, no outcome, no
  operator. A version that projected nothing is a version, and every surface
  that renders one already renders a field that is not there.
- **A `shell` `read` against a binary that is not installed is a first-class
  observation.** It records the command it tried, it versions when that stops
  being true, and a Comparison reports the day the fields came back.
