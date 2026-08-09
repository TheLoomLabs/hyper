# §11 — Distribution and version pinning

Two things reach a machine from outside the repository: the binary, and any Extension the repository
uses. Neither is trusted for having been fetched. Which version of each is allowed to act is a fact the
repository holds in a file a human reads in a diff; where bytes cross a network from a mutable source
they are checked against a digest recorded in that same file; and nothing is resolved again when a Run
happens.

This chapter states where the version pin lives and what enforces it, how the binary and an Extension
arrive, what verification each gets, and what `hyper` does when the binary that is running is not the
one the repository pins.

## The version pin

### Where it lives

The Repository declaration names the version of `hyper` that may act on the repository (§3, ADR-0020).
It is one artefact, `hyper.yaml` at the repository root, and the pin sits in it beside the retention
policy — the two facts that govern every Run and belong to no Procedure, Definition, or Target
declaration.

The pin is derived rather than authored. `hyper project` writes the version of the binary that ran it,
and nothing else writes it, so changing the version is *install a binary, run one command, read the
diff* — three acts in the open, each leaving something behind (ADR-0020). A version is a fact about the
binary rather than a claim about the world, which puts it on the side of the line where `hyper` writes
what it derives and the agent writes what is reviewed (§9).

The declaration carries the digest of the released artefact beside the version, written by the same
command (§3). What that digest is for, and why it is not the running binary's, is below.

### The gate

§9 states where the gate sits and which two commands stand outside it. What it declines under is named
here. A binary whose version differs from the pin in either direction is `version-pin-mismatch`; a
command that needs the pin and finds none is `version-pin-absent`, naming `hyper project` exactly as an
absent Store names `store init` (§12). Both are Refusals and both exit `77`.

Deleting `hyper.yaml` is therefore a dead end rather than a bypass, which is what ADR-0001 requires of
anything gate-shaped, and a repository with no pin gets one from its first `project`.

Nothing is exempted for being read-only. `check` predicts what the pinned binary will do (§4) and
`changes` renders what it did (§8); either one answering under a different binary answers a question
nobody asked.

### The comparison is exact

The gate compares version strings for equality. There is no range, no minimum, no compatible-release
operator, and no reading under which a patch release is close enough. The version number promises
nothing and no compatibility is inferred from it in either direction (ADR-0020) — which is what stops
the pin being softened later by a semver argument nobody reviewed.

### The digest at the fetch, the version at the gate

`hyper` never hashes itself, and the gate compares a version string rather than a digest (ADR-0020).
The digest does its work where bytes cross a network from a mutable source, which is the runner's
fetch and never the binary already on the machine.

`project` resolves the published checksum for the version it is recording, writes it into the
Repository declaration beside the pin (§3), and the projection copies it into each generated workflow
beside the version and the URL (§10). The digest a runner checks against is therefore derived from the
reviewed declaration rather than resolved again per workflow, and a hand-edited digest in a generated
file fails the projection check §10 states like any other hand-edit.

Resolving that checksum is the only thing the pin ever reaches the network for, and it happens
attended, at review time, landing in a diff — trust on first use, named rather than glossed. Freezing
the checksum is what converts a release tag, which is a mutable pointer, into an immutable reviewed
fact (ADR-0020). Re-projection resolves nothing: only a version change fetches.

`project` Refuses where it cannot resolve a published artefact for its own version
(`release-artefact-absent`, §12, §9). An unreleased binary therefore runs and checks and cannot
project, which is the same statement as: every pin in every repository names a version somebody can
download.

Only the platform the workflow's `runs-on` names is ever fetched, so one digest is recorded rather than
a table of them (§10). Which platform artefacts a release publishes is a release-process fact and not a
property of the tool.

## How the binary arrives

### The laptop

Unconstrained, and deliberately: a package manager, a release download, `go install`, a build from
source. The laptop was never a reviewed artefact. Under the gate above its *version* is governed
however it got there, which is the fact that mattered.

### The runner

One shell step in the workflow `project` generates: fetch the release artefact from the URL, check the
bytes against the frozen digest, unpack, and invoke the binary from the working directory (§10).
Version, URL and digest are literal in one reviewed file, and nothing is resolved when the job runs.

No setup action, first-party included. An action is code running in the job before `hyper` does,
holding the same secrets and resolved through a second distribution channel, which puts the
supply-chain problem back inside the artefact whose reviewability is the claim (ADR-0007). `go install`
would put a toolchain on the runner, and a container image would reintroduce the runtime the binary
does without.

**`hyper` reads credentials and never acquires them, so OIDC federation is not implemented**
(ADR-0007). A federated cloud reached from CI therefore needs a long-lived credential in the
executor's secrets, which is worse than the ecosystem norm and is the accepted price of not building
speculatively. If a Target ever needs federation it arrives as an Auth scheme `hyper` owns, never as a
third-party action inside a workflow `hyper` generated. Carried forward to §13.

## Skew

The pin makes laptop and runner provably the same code against one Store: the runner runs the version
the workflow fetched, the workflow was generated from the pin, and the laptop is gated on that same
pin. What follows is what each direction of a difference does.

**A binary newer than the pin Refuses.** This is the ordinary upgrade, seen from before its second
step: the new binary is installed and the repository has not yet been told. `project` is what moves the
pin, and the diff it writes is where the largest behaviour change in the system gets reviewed
(ADR-0020). Nothing about the Refusal is special-cased for the newer direction, since a binary is not
trusted for being new.

**A binary older than the pin Refuses.** Three ways to arrive there, one outcome. A machine that has
not been upgraded runs an older binary against a repository somebody else projected. Checking out an
older commit moves the pin under a current binary. And a workflow left behind by an older projection
fetches the version it names, which then Refuses against the pin it finds rather than acting on the
repository with a guardrail set nobody reviewed (§10).

**A hand-edited pin is caught twice**, by the projection check and by the fetched binary's own gate,
neither detection depending on the other having run (§10).

### What the pin does not cover

The Store outlives every pin, so it holds files written by every version ever pinned, each stating its
own schema version and read down from the reader's own (§7, ADR-0028). That integer is the system's
only mechanical compatibility fact, and it answers a different question from the pin: it keeps every
environment able to read what an older one wrote, where the pin keeps two environments running the
same code at once. Neither substitutes for the other.

**Reading an old commit's Store needs that commit's binary.** Checking out history moves the pin, every
command that reads the record is gated, and there is no auto-download to soften it. This is the same
bargain a toolchain directive makes and the direct price of refusing to make the environment an
authority axis. Carried forward to §13.

## `hyper` never updates itself

There is no `hyper upgrade` and no self-update, and nothing in the binary checks whether a newer
version exists (ADR-0019). Upgrading is the three acts the pin already fixes — install, `project`,
review the diff — and it happens entirely outside the binary. The update check dies separately and for
its own reason (ADR-0016, ADR-0019): it is egress performed on nobody's behalf by a tool that otherwise
reaches the network only where a reviewed artefact asked it to.

**A stale pin is therefore silent.** Nothing will ever tell you that you are three releases behind, and
no surface says so — the notification path being a Step you author rather than a channel `hyper` speaks
on (ADR-0021). Carried forward to §13.

## Extensions

An Extension is data with no code in it, so what distribution moves is a Manifest and there is nothing
behind it to fetch, build, or isolate (ADR-0004). The only code that runs is the binary the pin above
governs, which is why this chapter has one version to pin rather than one per Provider.

Built-in Providers and Extensions are the same object under the same grammar and the same checks (§4).
The difference is where the Manifest comes from: a built-in ships inside the binary, and an Extension
is a tracked file in `providers/`. That is the origin `providers` reports (§9), and it is the whole of
the difference — a Manifest's powers do not depend on who wrote it, only on what it declares and what a
Target grants (§5).

### `install`

`install <ref>` resolves the ref against a registry, fetches the Manifest, verifies the bytes against
the digest published for that ref, and writes the file into `providers/`. Bytes that do not match are
`extension-digest-mismatch` (§12) and nothing is written.

Registry as source, repository as record. The registry is where an Extension is discovered and
fetched; what executes is the file in the tree, reviewed in the same commit as the Definition that uses
it, and updated by an `install` whose whole effect is a diff. A Run resolves nothing from a registry —
a run-time fetch would be shared mutable state between review and execution.

`install` records the ref it resolved and the digest it verified in the Manifest's own origin block,
which the Manifest schema defines and only `hyper` writes — the rule that lets `project` write the pin
(§3). The digest covers the published bytes, which are the file without the block naming them, since a
digest cannot cover itself. `check` recomputes it and reports `extension-digest-mismatch` where it no
longer holds (§4), which is what makes the fetch's verification repeatable offline, by anyone reading
the repository, long after the machine that performed it is gone.

A Manifest carrying no origin block is a locally authored Provider: checked like any other and making
no digest claim. Editing an installed Manifest therefore means re-installing it or dropping the block,
and dropping it is one more visible edit to a tracked file — which is the only mechanism this path has
ever relied on.

### Digest only, never intent

`hyper` verifies that fetched bytes match a published digest. It never verifies that an Extension is
benign, and it treats no registry-side scan, signature, or badge as evidence that one is (ADR-0004).

The claim it makes instead is the one that survives an adversary: **a hostile Extension reaches nothing
you did not grant, and you can read everything it can reach.**

### The two things an Extension may never do

**It may never shadow a built-in Provider's name.** A collision is a load failure
(`provider-name-collision`, §12) and never a precedence rule, precedence being how a Definition
reviewed as one thing runs as another.

**It may never hold a Capability reserved to built-ins.** Some Capabilities are declared by built-in
Manifests and never granted to an Extension — the one behind an `opaque` shell Operation among them, so
that a third party can never ship a Provider that runs commands on your machine (ADR-0004). Which
Capabilities are reserved is fixed with the set itself (§12). A Manifest loaded from `providers/`
declaring one is `capability-reserved` (§12), refused at load rather than at the moment it would have
run.

Both are mechanical, both are decidable from the file alone, and neither depends on who published it.

### The Manifest's schema version

A Manifest is the one artefact carrying an explicit schema version, being the one authored outside this
repository's pin (§3, ADR-0023). What it means to read across that version is the rule a Store file's
version already carries, applied here: `hyper` reads any Manifest at or below the schema version it
knows, and Refuses on one written above it (`manifest-schema-unsupported`, §12) rather than guessing at
a shape it does not recognise (ADR-0028).

Guessing is the expensive failure in this one place. A Manifest read on a partial understanding of its
own shape would have its declared-equals-derived Capability check run against keys the reader could not
see (§4), which is the check the whole extension model rests on.

## What distribution does not add

There is no registry as a product, no publishing command, and no account: `install` consumes a
registry, and what a registry is beyond a place bytes and checksums are published is not `hyper`'s
concern. Nothing about a Provider's availability is a promise this tool makes. Carried forward to §13.

There is no vendoring mode, no offline mirror, and no cache directory. What an Extension amounts to is
a tracked file in the repository, which is already the vendored copy — cloning the repository brings
every Provider it uses, and a Run reaches the network only where a reviewed artefact asked it to.

There is no dependency between Extensions. A Manifest names no other Manifest, so there is no
transitive graph to resolve, to lock, or to review — which is what the whole of ADR-0004 buys, stated
from the distribution side.
