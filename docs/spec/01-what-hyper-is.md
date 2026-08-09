# §0 — What `hyper` is

> **Nothing reaches the world unreviewed; nothing changes unseen.**

An agent writes the artefact; you verify it offline before anything runs, and read exactly what
changed after — including what the agent changed about the artefact.

That is a design thesis rather than a market claim: the property `hyper` is built to hold, stated
from the position of one user and compared against nothing. Every chapter after this one is
downstream of it, and this one adds no requirement of its own.

## Two clauses, each covering the other's blind spot

The two clauses are not a two-item feature list. Each names an accountability the other structurally
cannot supply.

**Nothing reaches the world unreviewed.** A Run reaches the world only through the five reviewed
artefacts — Manifest, Target declaration, Definition, Procedure, Repository declaration (§2) —
written in one format with no expression language anywhere in it (§3, ADR-0022). A Probe carries no
Definition, and a Probe is a `read` against `local` that writes no Record and is not a Run (§2,
ADR-0009). What stands between an authored artefact and the world is stated in full by §4 and §5,
and every part of it is static: the two keys, the named Operation a `destroy` claim requires, the
mandatory Bound, the envelope. Nothing overrides any of it at invocation, and the way past a Refusal
is an edit to the artefact, put back through review (§5, ADR-0001).

Its blind spot is time. Static verification reads reviewed text against reviewed text (§4), so it
knows nothing about whether the world moved since anyone last looked, and a Manifest that passes
every rule can still be wrong about what it describes (§13).

**Nothing changes unseen.** A Run writes what it did to a branch of the same repository (§7,
ADR-0006), and it is read back as one Run against the Run before it, split by which actor did the
changing: the Assets `hyper` changed, the Observations the world changed, and the code that changed
between the two (§8). The third table is what the repository buys — an agent widening a `destroy`
Bound between two Runs is a change of the same class as a server going quiet.

Its blind spot is direction in time. A Comparison is retrospective, and nothing anywhere in the tool
renders a proposed change before it happens (§8, §13, ADR-0010).

Neither clause is accountability on its own. One acts on what has not happened yet and knows nothing
about the world; the other accounts for what has happened and stops nothing.

## The artefact is the whole of what there is to read

The thesis holds only where reading the artefact is sufficient, which is why there is nothing behind
one. A Provider is a Manifest and nothing else (§2, ADR-0004): every effect it describes is performed
by `hyper`, from the closed set of Capabilities `hyper` alone defines (ADR-0004, §12). An Extension —
a Provider authored and distributed by someone other than `hyper` — is a Manifest too, so `install`
moves data and there is nothing behind it to fetch, build, or isolate (§11). Nor does `hyper` author
one: it writes what it derives, and the agent writes what is reviewed (§9).

That line is drawn against a supply chain, and the evidence it is drawn on is
[`docs/research/swamp-prior-art.md`](../research/swamp-prior-art.md) — on `main`, frozen at the
commit it was read at, and openable. Three facts from it: a lockfile resolving 200 npm packages
against 18 JSR ones; extension dependencies inlined into the bundle on the author's machine at
publish time, so they never enter the consumer's lockfile and version pinning is advisory; and zero
occurrences of `Deno.permissions` in a tree hosted on a runtime that has one. ADR-0004 is where
those weigh against the alternatives, and §13 states what removing the code costs.

## The loop

The chapters are ordered by the loop the thesis names, and each depends on the ones before it. §2
through §5 — the model, the format, what `check` refuses, and what stands before the world — are
*nothing reaches the world unreviewed*. §6 through §8 — the Run, the record it leaves, and the
renderings that read it back — are *nothing changes unseen*. The rest is what both halves need in
order to be reachable, repeatable and honest: the surfaces (§9), Cadence (§10), distribution (§11),
the closed sets (§12), and the limits (§13).

## One screen

Half of the thesis is free to watch. `check` and a Definition review reach nothing outside the
repository (§4, §8), so what follows runs against a clone, with no credential and no infrastructure.
The other half costs two Runs against real systems, and §13 says so.

An agent widens the `destroy` Step's Bound from 3 to 5 in a Procedure that retires preview
environments. `check` reports nothing: a Bound is declared, so `bound-missing` does not apply, and
whether an Expansion exceeds one is not decidable from the artefacts at all (§4). The edit is legal,
and it is not invisible.

Below is §8's rendering of that working tree, abridged to the Step the agent touched — §8 states it
in full, with the `AUTHORITY` table and the two Steps dropped here.

```
$ hyper review procedures/retire-preview-envs.yaml

  BLAST RADIUS      │  procedures/retire-preview-envs.yaml     a91f0c2 → working tree
  ──────────────────┼──────────────────────────────────────────────────────────────
  envelope ✓        │   targets: [local, staging]

  DESTROY  staging  │     - id: retire
                    │       definition: hetzner-staging
                    │       operation: delete_server
                    │       over:
                    │         assets:
                    │           - field: labels.role
                    │             equals: preview
                    │           - field: created_at
                    │             older_than: 14d
                    │ -     bound: 3
                    │ +     bound: 5

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  DESTROY    line 21  step retire   delete_server, bound 5
  WIDENED    line 30  step retire   bound 3 → 5 since a91f0c2
  ENVELOPE   line 3   ok            no step reaches a target outside [local, staging]
```

`WIDENED` is the review surface reporting that an agent widened a destroy Bound — before anything
ran, against no infrastructure, beside the line that made the claim. The same fact after two Runs is
a row of `THE CODE MOVED` (§8). Who may make the edit stick is who may merge it, and there is no
second authority axis inside the tool (§13).
