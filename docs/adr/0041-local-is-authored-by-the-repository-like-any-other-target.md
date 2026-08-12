# `local` is authored by the repository like any other Target

`hyper` ships no Target declaration. `local` is a file in `targets/` that a repository writes, reviews
and diffs like every other Target declaration, and what its name reserves is one thing: it is the Target
a Probe binds. A declaration named `local` declares `class: local` and carries no `auth:` block, and one
doing either is refused. Nothing else about the file is special, which includes its count — more than
one declaration claims `class: local` where a repository has reason to.

We chose this because a host grant is not a fact `hyper` derives. ADR-0024 confined every host to an
artefact so that *which hosts may this repository reach without credentials* becomes a reviewed fact, and
a set supplied by the binary makes that sentence false in its subject: it is then a fact about the binary,
reaching the world and appearing in nobody's diff, which is the shape ADR-0020 refused on the version pin.
There is also no set to supply. Non-empty is `hyper` choosing a repository's reach; the honest wildcard is
what ADR-0024 killed; empty is indistinguishable from the file's absence. ADR-0039 settled the same
question one artefact-class over — `hyper` ships a Provider only where the Capability it needs is one
nobody else may declare — and anybody can write this file, which makes shipping it convenience.

## Considered options

- **`hyper` supplies the declaration.** What "reserved" reads as, and the reading a competent
  implementer reaches unaided. Rejected on the arguments above, and on what it would have to contain:
  every candidate set is either a choice `hyper` has no standing to make or the wildcard the format no
  longer has.
- **The host grant moves to the Repository declaration.** Rejected without needing a decision. That
  artefact's admission rule takes only facts governing every Run and belonging to no Procedure,
  Definition or Target declaration, and hosts belong to a Target declaration by construction.
- **Exactly one declaration claims `class: local`.** ADR-0039's reading, and the one §12's roster
  sentence implied. Rejected because the argument there was about binding a **non**-local Target — one
  naming a remote endpoint and holding its credentials, where the gutter would read `shell
  cloudflare-prod` for a command running here. Two class-local Targets both mean the machine `hyper` runs
  on, so that lie is not available, and every actual authority is per-declaration while `class:` only ever
  rejects a mismatch.
- **Refusing `auth:` on every class-local declaration rather than on the reserved name.** Rejected as
  wider than the guarantee it protects: what needs to hold is that a Probe resolves no credential, and a
  Probe binds `local`.

## Consequences

- **The credential-free property is a reservation again, not a consequence.** ADR-0031's slot-coverage
  result had `local` credential-free because its declaration covers no slots — sound on the unstated
  premise that nobody could write that declaration's `auth:` block. With the author holding the pen, an
  `auth:` block plus a class-local Provider declaring a scheme passes coverage, and a Probe becomes an
  unrecorded credentialled call: no Definition, no Record, no Journal entry, and ADR-0017 handing back
  the raw wire. The refusal restores the property structurally, and the Journal-is-the-audit-log claim
  stays true because there are no credentialled calls it cannot see.
- **`capabilities:` on a Target becomes a grant that something checks.** A Capability an Operation's
  request names and the bound Target does not grant had no check and no `error_code` anywhere. It has both
  now, per binding, on slot coverage's shape — which makes omitting `shell` from every class-local
  declaration a one-line reviewed switch stopping every command a repository could run.
- **The `opaque-destroy:` opt-in is no longer one switch over every command.** ADR-0039 accepted that
  coarseness on the ground that granularity lives in a Definition's named-Operation claim. It lives in the
  Target too: a class-local declaration opting in and another not confines command-destroy authority to
  the Definitions that name the first.
- **`hosts:` is present exactly where `http` is granted.** Both directions are vacuous otherwise, and the
  Target now has an analogue of the Manifest's own-keys-disagree check.
- **An absent `local` is a grant of nothing rather than a fault.** A fresh repository holds no such file
  and is complete without it; a class-local Provider compels one only through a binding a Definition
  makes. What an *artefact* naming an absent artefact does is a wider question than this one and is not
  answered here.
- **The environment a command inherits is stripped of nothing when it binds `local`**, that declaration
  naming no credential slot. §13 states this beside the reach limit rather than leaving it to be found.
