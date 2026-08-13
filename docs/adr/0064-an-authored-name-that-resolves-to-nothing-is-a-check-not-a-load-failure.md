# An authored name that resolves to nothing is a check, not a load failure

An artefact names another artefact in four positions — a Definition's `provider:` and its `targets:`
members, a Step's `definition:`, and a nested invocation's `procedure:` — and names a *member* of one in
four more: a Step's `operation:`, a Definition's `destroy:` members, a `field:` at either Record root,
and the `step:` half of a reference. **A name that resolves to nothing in any of them is a `check` rule
with a file and a line.** The repository still loads; the artefact still reviews; the row points at the
line the name was written on. Two codes carry it, split by namespace: `artefact-absent` where the
namespace is the repository's artefacts, `reference-unresolvable` where it is what an artefact declares.

[ADR-0060](0060-naming-nothing-is-a-usage-error-fetching-nothing-is-not.md) decided the other half —
a name the **user** typed, which is a usage error at exit `2` because `check`'s row has no file and no
line to carry and no check to name. Here both coordinates exist, so the reason `2` won there is exactly
the reason it loses here, and the two ADRs state one rule between them: **who wrote the name decides
where the fault goes.** The user's is an invocation with no artefact to point at; the author's is an
artefact declining, on the line that must be edited.

We chose this because every unaided reading of the fault is a *load* failure, and each one costs
something the corpus is built on.

**A Step's `target:` is not a lookup.** The unaided reading is that a name is a name and you resolve it,
so a Step's `target:` reaches `targets/` like a Definition's `targets:` member does. §3 already says
otherwise — it "is checked for membership of its Definition's `targets:` list" — and reading it as a
lookup makes one absent `targets/prod.yaml` bound by six Steps render seven rows, six of them on lines
that are not wrong. A Target's existence is asked once, where a Definition claims it. The membership
question that remains had no code at all, and now has one: `target-not-claimed`, on
`operation-not-claimed`'s shape.

**A file that will not parse is still present.** The unaided reading is that an artefact you could not
build is an artefact that is not there, so every Definition naming a malformed Manifest reports the
Manifest missing. That inverts the discipline the row exists for: one broken file would emit a row on
every artefact downstream of it and bury the single row that names the actual edit. An artefact
declares a name by carrying one, and a fault inside it is reported once, on its own line.

**Failing to resolve does not stop the load.** The unaided reading is the implementation's: you cannot
build the object graph, so you throw. But `check` is specified to load every artefact and evaluate every
rule together, and a resolution fault is one rule among thirty — a repository with a missing Provider
must still report its `bound-missing`, its `envelope-exceeded` and everything else in one pass, because
a checker that stops at the first dangling name makes the author fix the repository one row per run.

**`review` renders, and does not decline.** The unaided reading is that a screen which cannot derive a
Step's Kind has failed. §9 already says a review does not run `check` and that only an artefact that
*would not load* exits `1` — and this one loads. So the gutter marks `unresolved` where the derivation
would have gone, `AUTHORITY` renders every row with its unsupplied cells named, and the review exits 0.
§12 had already worked out the shape: a review does not decline, so an absence travels as a **name in a
value position** rather than as an `error_code`, which is what `not-run` and `no-store` are.

Matching is ADR-0060's rule unchanged — byte-exact over UTF-8, case-sensitive, against the artefact's
own `name:`, never settled by whether an `open` succeeded. A lookup the filesystem decides renders on a
laptop and fails in CI, which is the one divergence between the only two environments this tool has.

## Considered options

- **One code for all eight positions**, the row's `field:` column carrying which name failed. Rejected:
  it collapses two remedies that differ in who can perform them. A missing artefact may be the reader's
  to write; a missing member is a key inside an artefact that already exists, and where that artefact is
  a built-in or somebody else's Extension it is not theirs to write at all. That is the distinction
  `name-mismatch` was taken out of `kind-mismatch` to preserve — which of two files the next act
  touches.
- **One code per position** — `provider-absent`, `target-absent`, `definition-absent`,
  `procedure-absent`, `operation-absent`, `step-absent`. Rejected as six names for one check. The
  discrimination they would buy is already carried by the row's `file`, `line` and `field`, and §12's
  members are checks rather than positions.
- **`reference-unresolvable` widened to carry all eight.** Rejected: §3 defines *reference* as a
  specific mapping form, and a reader handed that code on a `provider:` key would go looking for a
  `{step, path}`. The widening that *is* correct stops at a namespace boundary — §3 already applies the
  code to a bare `field:`, which is no reference either, so the scope was always the namespace and never
  the syntax.
- **`target-not-claimed` folded into `operation-not-claimed`**, as *a Step naming something its
  Definition does not claim*. Rejected on that code's own test: a reader handed *operation not claimed*
  on a `target:` line edits `destroy:`, which is the wrong file's wrong key.
- **`review` exits `1` with `check`'s row**, widening *would not load* to mean *would not resolve*.
  Rejected twice over: it makes `review` run a check §9 says it does not run, and the artefact in
  question loads perfectly — *would not load* means found and faulty, and this one is found and
  complete.
- **The absence named in the review's header**, on the `not-run` / `no-store` precedent. Rejected: that
  pair is the right precedent for *how* an absence is named and the wrong one for *where*. A range is a
  whole-artefact fact; a dangling `definition:` is one line's, and §8 admits a header member only where
  no supply on the screen already holds the fact. On a Procedure with twenty Steps a header line also
  loses the only thing the reader needs, which is *which* one.
- **A near-miss suggestion — *did you mean `preview-dns`?*** Rejected on ADR-0047's and ADR-0060's
  shared ground: partial resolution moved from the resolver into the error message is still partial
  resolution, and a human who accepts one has approved an artefact they did not read. Naming the path
  `hyper` looked for is not that — it is total, derived from the kind and the name, and it is where the
  second of the two edits lives.

## Consequences

- **§12 grows by two, from forty-five to forty-seven** — `artefact-absent` and `target-not-claimed`.
  Half of what this decision covers cost nothing: `reference-unresolvable` was widened rather than
  minted, its scope having always been a namespace and not a syntax. §4's contribution goes from
  twenty-eight to thirty. The `-not-claimed` pair becomes a family: a Definition claims Operations and
  it claims Targets, and a Step naming outside either claim reads the same way.
- **`FLAGS` lists eight rather than seven**, and the set did not grow by decision. §12's rule is that
  every marker class the gutter carries indexes there, so `unresolved` arriving in §8 brought its flag
  with it — the first name to arrive that way since the rule was written, and evidence the rule works
  rather than an act of growing the vocabulary.
- **An uninstalled Extension is a `check` fault, offline.** A `provider:` naming what neither the
  built-ins nor `providers/` holds is `artefact-absent` with no network reached — not `install`'s exit
  `1`, which ADR-0060 reserved for a name in somebody else's namespace. A repository cloned without its
  `providers/` files says so on the line that needs the Provider.
- **#50's `local` question closes on one line, in two halves that disagree on purpose.** Where nobody
  named the Target — a Probe — an absent declaration is a grant of nothing and the request declines
  `host-not-granted` ([ADR-0042](0042-a-probe-is-bounded-by-the-grant-it-binds.md)). Where an author
  wrote `targets: [local]`, it is `artefact-absent`. One rule about who wrote the name, not two rules
  about `local`.
- **A Definition whose `provider:` is absent reviews complete and unmarked.** Nothing on that screen is
  derived from a Manifest: the gutter marks what the Definition authored, and `AUTHORITY` is assembled
  from the Definition and a Target declaration. `unresolved` is therefore Procedure-only, for a reason
  that is not `unbounded`'s — not that the key lives on a Step, but that a Definition's screen never
  needed the artefact that went missing.
- **Exit codes are inherited and not decided.** This is a §4 check like any other, so it is `1` from
  `check`, `77` from a Run's pre-flight, and nothing at all from `review`. The ticket's fork —
  Refusal or load failure — dissolves once the fault is a check: §4's codes have never had one exit
  code, they have the code of the command that ran them.
