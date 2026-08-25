# The `hyper` version is pinned by the repository

A **Repository declaration** names the version of `hyper` that may act on the repository. Every
command compares itself against that pin and Refuses on mismatch — on a laptop and on a runner
alike — with `version` and `completions` exempt for reading no repository, and `hyper project` exempt
for being the pin's only writer. `hyper project` is the door: it *derives* the pin from the binary
that ran it, so changing the version means installing a binary, running one command, and reading the
diff it writes.

We chose this because after ADR-0004 the binary is the only code that runs, which makes a version
bump the largest behaviour change available in the system — larger than any Definition edit, since it
moves Bound checking, Expansion ordering, Repeatability evidence and redaction at once. ADR-0005
already required the generated workflow to pin the version. Stopping there would have pinned the
runner and left the laptop free, and the two write the same Store: that is the
environment-as-authority-axis the safety model deleted, arriving through packaging rather than
through policy.

## Considered options

- **Pin only in the generated workflow.** The reading ADR-0005 left open. Rejected twice over: it
  makes the guardrail an environment axis, and a repository whose Procedures declare no Cadence
  generates no workflow at all, so the pin would have nowhere to live in exactly the repositories a
  human drives by hand.
- **Record the version in Provenance and render it, but enforce nothing.** Honest, cheap, and already
  half-built — Provenance carries the version and the Comparison renders code facts. Rejected because
  recording is not preventing; the Comparison is retrospective by construction and reports a
  guardrail that moved after it has moved.
- **Compare digests rather than version strings at the gate.** Attractive against a project that
  prefers derived facts to declared ones, and wrong here: a self-hash is a check the checked party
  performs on itself, so a binary that lies about its version lies about its hash. The digest earns
  its place only where bytes cross a network from a mutable source, which is the fetch, not the gate.
- **An authored pin.** Lets a bump be staged before the binary exists, at the cost of a pin naming a
  version nobody has and a digest somebody must paste. The derived pin puts the version on the side
  of the line where `hyper` writes what it derives and the agent writes what is reviewed.

## Consequences

- **The Repository declaration exists, and this amends ADR-0014.** That decision anticipated it — *a
  setting that matters belongs in a reviewed artefact* — and it is not a configuration file: it is
  authority, it is reviewed, and `hyper review` renders it. To stop it becoming the junk drawer
  ADR-0014 killed, it admits **only facts that govern the repository as a whole and belong to no
  Procedure, Definition, or Target declaration**. Two qualify today: the version pin, and the
  retention policy ADR-0011 had already put in an unnamed reviewed artefact.
- **Digest at the fetch, string at the gate.** The generated workflow carries the version, the URL,
  and the digest of the artefact for the platform its `runs-on` names — the pin and its verification
  in one reviewed file, with nothing resolved at run time. This is `install`'s rule applied to the
  binary itself: registry as source, repository as record, digest only and never intent.
- **Freezing the digest is trust-on-first-use, and that is its whole value.** A release tag is a
  mutable pointer, because the asset under it can be replaced after publication. `project` fetches
  the published checksum for its own version once, attended, on a laptop, and freezes it into a file
  a human reads — converting a mutable pointer into an immutable reviewed fact. Re-projection is
  offline; only a version change needs the network.
- **Projection is repo-wide and all-or-nothing.** There is no `project <procedure>`, because
  per-Procedure projection would let two Procedures pin different versions against one Store — the
  skew this decision exists to prevent, re-entering as a convenience flag.
- **An absent pin Refuses.** A command that needs the pin and does not find one Refuses (77) naming
  `hyper project`, exactly as an absent Store Refuses naming `store init`. Deleting the file is a
  dead end rather than a bypass, which is what ADR-0001 requires of anything gate-shaped.
- **The version number promises nothing.** No compatibility is inferred from it; there is no "surely
  a patch release is fine". The only mechanical compatibility fact is the Store's schema integer,
  which increments when a file's shape changes and is independent of the release version.
- **A hand-edited pin is caught twice.** `check` fails on projection drift, and the fetched binary
  Refuses against a pin it does not match — two independent detections of the same edit.
- **An unreleased binary works locally and cannot project.** `project` Refuses when it cannot resolve
  a published artefact for its own version, which is precisely the case where it would otherwise
  write a workflow fetching a binary nobody can download.
- **Ergonomic cost, accepted: you cannot read an old commit's Store without that commit's binary.**
  Checking out history moves the pin under you, and there is no auto-download to soften it. This is
  the same bargain a toolchain directive makes, and the price of refusing the axis.
