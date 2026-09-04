# Security

## Reporting a vulnerability

Email **igor@theloomlabs.org** with the subject **`vulnerability report`**.

Include what you did, what happened, and what you expected — a repository of
artefacts and the command line you ran is the best possible report, because it
is what the tool is built to make readable.

**Please do not open a public issue for a vulnerability.** Everything else
belongs in [the tracker](https://github.com/TheLoomLabs/hyper/issues).

### What to expect

| | |
| --- | --- |
| Acknowledgement | within 3 working days |
| An assessment — accepted, not a vulnerability, or need more information | within 10 working days |
| Public disclosure | when the fix ships, or 90 days after your report, whichever is sooner |

If a report is accepted and the deadline arrives with no fix, the report is
disclosed anyway, with what mitigations exist. A deadline that moves whenever
the fix is late is not a deadline.

Credit in the disclosure by default; say so if you would rather not be named.

## Supported versions

**No release of `hyper` has been published yet**, so there is nothing to
backport to and no version table to keep. Fixes land on `main`.

What that means for shipping a fix is the version pin, which
[§11](docs/spec/12-distribution-and-version-pinning.md) owns: a repository names
the version of `hyper` that may act on it, and `hyper` never updates itself and
never asks whether a newer one exists
([ADR-0019](docs/adr/0019-hyper-never-updates-itself.md)). So there is no
mechanism by which a fix reaches you unasked, and no mechanism by which anything
else does either. Upgrading past a fix is an edit to a reviewed artefact, made by
a person, visible in a diff.

## What the design already promises

Three properties are worth stating before you look, because a report that lands
against one of them is a report against something real, and much of the usual
attack surface is simply absent.

**`hyper` never holds a secret.** There is no vault, no keychain, no encryption
at rest, because nothing is at rest — `hyper` reads credentials from the process
environment, which is how it works with every secret manager without integrating
with any of them. A Target declaration names the *environment variables* its
credentials resolve from; `hyper` resolves them once per Run, holds them in
memory, and writes no credential *value* anywhere
([ADR-0007](docs/adr/0007-hyper-never-stores-a-secret.md)). The variable *names*
do reach the record, and that is the whole of what does: a `credential-absent`
Refusal names the variable the environment did not hold, and a Refusal is written
to the Journal like any other. The rule holds for outputs as well as inputs: a
Manifest declares which output fields are secret, and the Record carries a
redaction marker with *presence only* — no digest, no length, because a digest of
a human-chosen secret is an offline-crackable oracle. Secret-valued output
requires a sink the invocation supplies, and a Run reaching such a Step with none
Refuses before its first Step (`secret-sink-absent`,
[ADR-0148](docs/adr/0148-a-secret-sink-is-a-directory-hyper-makes-and-one-file-holds-one-value.md)).
The sink is a **directory `hyper` creates** `0700`, holding one `0600` file per
value at `<nnnn>/<name>/<field>` — the Step's position, the Record's name and the
declared field — and it is the one route by which a secret value leaves `hyper`
at all. Three paths are refused: `-`, because stdout lands in the same pipe a CI
job logs; a path inside the repository working tree, because a secret written
there is one `git add` away from the record; and a path something is already
standing at, so that every file under a sink is that Run's and no stale value can
be read as fresh. The file holds the value and nothing `hyper` added, and nothing
reads it back — no Run parses it and it never reaches the Store.

What `hyper` does write is a *reference*, never a value: `project` generates a
workflow whose `env:` block names each variable as `${{ secrets.NAME }}`,
derived from the bindings ([§10](docs/spec/11-cadence-and-projection.md)). That
is a reviewed file naming a secret `hyper` never resolves — the runner resolves
it, before `hyper` starts.

**There is no third-party code.** A Provider is a Manifest and nothing else —
no plugin binary, no extension language, no sandboxed module, no transitive
dependency graph, no build step
([ADR-0004](docs/adr/0004-extensions-are-data-not-code.md)). Every effect a
Manifest describes is performed by `hyper` itself, from a closed set of
Capabilities that only `hyper` defines — a set closed at two, one of them
reserved to the Providers `hyper` ships: a third party can never publish a
Provider that runs commands on your machine. The claim that survives an adversary is stated in
[§11](docs/spec/12-distribution-and-version-pinning.md): **a hostile Extension
reaches nothing you did not grant, and you can read everything it can reach.**

**`install` verifies a digest and claims nothing else.** Fetched bytes either
match the digest published for the ref or they are `origin-digest-mismatch` and
nothing is written. `hyper` never verifies that an Extension is benign, and
treats no registry-side scan, signature, or badge as evidence that one is.

### Where the boundary is written down

- [§5 — Authority and safety](docs/spec/06-authority-and-safety.md): the two
  keys, the envelope, `opaque` and `destroy`, the Bound, and why there is no
  bypass.
- [§6 — Execution](docs/spec/07-execution.md): the fixed order a Run begins in,
  and every gate that declines before Step 1 — including credential resolution
  and the Secret sink.
- [§11 — Distribution and version pinning](docs/spec/12-distribution-and-version-pinning.md):
  the pin, the digest, `install`, and the two things an Extension may never do.

### What is out of scope

- **What an Operation you authorised then did.** A Capability you granted and a
  Target that accepts `destroy` are a Run doing what the artefacts said. That is
  the tool working; the review is where it is caught.
- **Anything reached through the `shell` Capability.** It is opaque by
  declaration — `hyper` cannot describe what a command does, says so on the
  review surface as `OPAQUE`, and is not claiming to contain it.
- **The honest limits.** [§13](docs/spec/14-non-goals-and-honest-limits.md) is a
  chapter of what `hyper` does not know and does not cover, written down
  deliberately. A report that one of them is true is welcome as an issue, and is
  not a vulnerability.
