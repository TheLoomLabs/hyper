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

§9 states where the gate sits and which commands stand outside it — `version` and `completions`, which
read no repository, and `project`, which is the pin's only writer and cannot be gated on what it
writes. What it declines under is named here. A binary whose version differs from the pin in either
direction is `version-pin-mismatch`; a command that needs the pin and finds none is
`version-pin-absent`, naming `hyper project` exactly as an absent Store names `store init` (§12). Both
are Refusals and both exit `77`.

Deleting `hyper.yaml` is therefore a dead end rather than a bypass, which is what ADR-0001 requires of
anything gate-shaped, and a repository with no pin gets one from its first `project`.

Nothing is exempted for being read-only. `check` predicts what the pinned binary will do (§4) and
`changes` renders what it did (§8); either one answering under a different binary answers a question
nobody asked. `project`'s exemption is not that one and does not open the door to it: it is exempt for
*writing* the pin, and a repository with no pin gets one from its first `project` exactly because
nothing gates that command.

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
beside the version and the URL (§10). What it reads is the checksums file published under the release
tag, and the line in it naming the artefact the template below resolves to — a few hundred bytes rather
than the artefact itself, both being the same mutable source read in the same instant, so hashing bytes
nothing on this machine will ever execute buys nothing over reading the checksum beside them. The
digest a runner checks against is therefore derived from the reviewed declaration rather than resolved
again per workflow, and a hand-edited digest in a generated file fails the projection check §10 states
like any other hand-edit.

Resolving that checksum is the only thing the pin ever reaches the network for, and it happens
attended, at review time, landing in a diff — trust on first use, named rather than glossed. Freezing
the checksum is what converts a release tag, which is a mutable pointer, into an immutable reviewed
fact (ADR-0020). Re-projection resolves nothing: only a version change fetches.

`project` Refuses where it cannot resolve a published artefact for its own version
(`release-artefact-absent`, §12, §9). Three shapes reach that one code: no release under the tag, no
checksums file beside it, and no line in that file for the artefact the template names. An unreleased
binary therefore runs and checks and cannot project, which is the same statement as: every pin in every
repository names a version somebody can download.

**A fourth shape reaches it and cannot be told from the first two.** This read carries no credential
and never will (ADR-0007), so a release that is published where nothing unauthenticated may read it —
a private repository, a private fork — answers exactly as an absent one does. `hyper` classifies it
with them, which `77` allows on its own criterion: a verbatim retry Refuses identically. What the
Refusal may not do is state as fact the thing the answer could not establish, so the remedy turns on
whether the checksums file was read (ADR-0127). Where it arrived and named no artefact, the release
was readable and its contents were seen: the remedy names publishing a release for this version and
installing a binary some release does name. Where nothing arrived, the remedy names those two and,
between them, making an existing release readable unauthenticated — the third possibility named
rather than resolved.

**A fetch that did not complete is exit `1` and not that code**, which is `install`'s own rule one
command over (§9, ADR-0060): a host that did not respond, a resolution that timed out, no network at
all — and equally a release host that answered a rate limit or a bad gateway, which is an answer that
arrived and is still not an answer about the artefact. What sorts the two is `77`'s promise that a
verbatim retry Refuses identically, which the three shapes above keep and a rate limit does not.
Nothing is written on either path.

Only the platform the workflow's `runs-on` names is ever fetched, so one digest is recorded rather than
a table of them (§10) — `runs-on` and the artefact's platform being one compiled-in fact rather than
two. What a release publishes *beyond* those two files is the release process's business and no
property of the tool; what it publishes *them* as is not, since the binary names both by a template it
holds and cannot be argued out of. A disagreement between the two is what
`release-artefact-absent` reports, attended, on a laptop, at review time — rather than as a fetch that
404s on a runner at three in the morning.

## How the binary arrives

### The laptop

Unconstrained, and deliberately: a package manager, a release download, `go install`, a build from
source. The laptop was never a reviewed artefact. Under the gate above its *version* is governed
however it got there, which is the fact that mattered.

**What a binary's version is depends on how it was built, and two things can name one.** The
release stamps it with the linker, which decides wherever it wrote; where nothing stamped, the
version is the one Go derived from the repository the source sat in and recorded in the build
information — the tag where the source is that tag, that tag marked `+dirty` where the tree carried
edits, and a pseudo-version where the commit is no release (ADR-0138). So a `go install` at a tag
names the tag whether or not the flag was given, and a build from an edited tree or an unreleased
commit names something no release published and is Refused by every repository, exactly as the pin
gate above states. A binary with neither — a `go test` binary is the one that exists — is `unknown`,
which is nobody's release.

Both are facts about the bytes rather than about the machine, which is the line this chapter draws:
nothing is read from a file, a flag or an environment variable at run time, and no version is
resolved after the build. What the fallback replaced was not a stricter check but a build with no
answer at all.

### The runner

One shell step in the workflow `project` generates: fetch the release artefact from the URL, check the
bytes against the frozen digest, unpack, and invoke the binary from the working directory (§10).
Version, URL and digest are literal in one reviewed file, and nothing is resolved when the job runs.

No setup action, first-party included. An action is code running in the job before `hyper` does,
holding the same secrets and resolved through a second distribution channel, which puts the
supply-chain problem back inside the artefact whose reviewability is the claim (ADR-0007). `go install`
would put a toolchain on the runner, and a container image would reintroduce the runtime the binary
does without.

**`actions/checkout` is the one action the projection names, and its exemption is stated rather than
assumed.** What a setup action would do is decide which bytes execute as `hyper`; `checkout` does not —
the digest in the reviewed file decides that, and it is checked against bytes fetched from a literal URL
by a step whose script lives in the workflow rather than in the tree. The action is pinned by commit
SHA, which is a stricter pin than `hyper` gives itself: `runs-on: ubuntu-24.04` names an image GitHub
rebuilds continuously, and that image supplies the `bash`, `curl`, `tar` and `sha256sum` the install
step runs and the `git` the deepen step before it runs (§10). Refusing a SHA-pinned first-party action while trusting a rolling image would be straining at
a gnat. **The trust boundary is the runner**, drawn there because the executor is trusted to execute at
all, and what stands inside it is the refusal to let anything but a digest choose the binary
(ADR-0046).

**The Store's credential is the checkout's, and the projection declares it.** `persist-credentials:
true` is written into the generated file rather than left to the action's default: it is what leaves an
authenticated remote behind for `hyper` to fetch and push the Store branch with (§10, §7), and a
byte-exact generated file resting silently on a default that belongs to somebody else's release cycle is
the same defect as an unstated constant, one layer down. What it buys is worth naming: `hyper` never
holds the Store's credential at all. It inherits a configured remote and pushes through it, which is
ADR-0007's claim in its sharpest form anywhere in the system — there is no slot, no resolution, and
nothing to suppress in a rendering, because the token never reaches the process.

**`hyper` reads credentials and never acquires them, so OIDC federation is not implemented**
(ADR-0007). A federated cloud reached from CI therefore needs a long-lived credential in the
executor's secrets, which is worse than the ecosystem norm and is the accepted price of not building
speculatively. If a Target ever needs federation it arrives as an Auth scheme `hyper` owns, never as a
third-party action inside a workflow `hyper` generated. Carried forward to §13.

## The projection's constants

The workflow §10 states carries facts no artefact declares and nothing in the repository derives: which
runner it names, which action it checks out with, and where the bytes it fetches live. They are
**compiled into the binary**. There is no file to put them in (ADR-0014), the Repository declaration
admits only facts that govern every Run and belong to no Procedure, Definition, or Target (ADR-0020),
and regeneration reaches no network — so the binary is the only thing left that could hold them, and it
does (ADR-0046).

Four, and this is the whole set:

- **The runner** — `runs-on: ubuntu-24.04`.
- **The checkout** — `actions/checkout@<commit>`, a commit and never a tag, a tag being a mutable
  pointer for exactly the reason a release tag is.
- **The release artefact** —
  `https://github.com/TheLoomLabs/hyper/releases/download/v<version>/hyper-<version>-x86_64-linux.tar.gz`,
  what the runner fetches.
- **The checksums file** — `checksums.txt` under the same tag, what `project` reads once, attended, to
  freeze the digest.

The version is the only variable, and the platform is not one: it appears in the artefact's path and in
its filename, and everything else is literal. Both URLs are stated here because the projection check is
byte-exact (§10) — a template shown in an example and stated nowhere is a thing every reader has to
guess at identically for the check to mean anything.

**A script the projection writes is not a fifth constant.** The install step's `curl`, `sha256sum` and
`tar`, and the deepen step's guarded `git fetch --unshallow origin "$GITHUB_REF"` (§10), are the file's
shape rather than facts about the world outside it: they name no version, no host and no third party,
and they change only when the binary's idea of what the job must do changes. The ref that step fetches
is not a constant either — the executor supplies it, as it supplies `$GITHUB_STEP_SUMMARY` and every
`${{ secrets.… }}` in the file — and no branch name is compiled in for it to be one
([ADR-0134](../adr/0134-the-deepen-step-names-one-ref-and-what-deepens-the-code-branch-is-the-clones-own-boundary.md)).
What they do consume is the image's tools, which is the exemption two paragraphs up and not a new one.

**The binary's own `git` is a different claim and is not this exemption.** `hyper` reads and writes the
Store by invoking `git` as a subprocess (§7, ADR-0075), and it does so on a laptop as much as on a
runner, where the sentence above is about what a **generated workflow** consumes on an image `hyper`
names. It is the one external tool the binary requires, no version is pinned for it — every command on
that path predates 2010, and a pin would be a fifth constant this section closed at four — and §13
carries it as a limit rather than letting the exemption stretch to cover it.

The generated file's *shape* is the binary's throughout — that is what generate-and-verify means — and
its content divides in two. Everything the file says about this repository derives from something
somebody wrote and reviewed: the recurrence, the job and workflow names, the `env:` block, the
concurrency group, the version and the digest (§10). These four are what it says about the world
outside both the binary and the repository — one runner, one third party's action, two URLs — and
**they are therefore the complete list of what a `hyper` release can change in a repository that edited
nothing.**

A constant moves only when the binary moves, and a binary that differs is a version that differs — so
no constant can go stale on its own. The file already carries the version in four places, and the
repair is the ritual ADR-0020 fixes: install, `project`, read the diff. A third-party action's pin
arriving in a diff a human had to read anyway is a supply-chain fact arriving free, which is the whole
of what this costs on an ordinary upgrade.

What it costs on an extraordinary one is real and is stated in §13: a checkout commit that needs to
move for a reason of its own — a vulnerability in the action, an image GitHub retires — moves only in a
`hyper` release, and a hand-edit to the generated file is caught by `projection-stale` like any other.

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
is a tracked file in `providers/`. That is the origin `providers` reports, a two-member set §12 states,
and it is the whole of the difference — a Manifest's powers do not depend on who wrote it, only on what
it declares and what a Target grants (§5).

**An Extension is a Provider somebody other than `hyper` authored, whether or not it was fetched.** A
Manifest an author typed into `providers/` this morning is one, exactly as an installed Manifest is:
the two differ in whether they claim an upstream, which the `origin:` block below carries and which is
a fact about a Manifest's provenance rather than about where its bytes load from (ADR-0073). The wire
keeps the two apart — `origin` says which of the two places the bytes are, and `provider` reports the
block's ref and digest beside the Manifest's other declared facts (§9).

### What ships built in

One Provider ships inside the binary, `shell`, and §12 states it in full. What fixes the set is a
criterion rather than a list: **`hyper` ships a Provider only where the Capability it needs is one
nobody else may declare** (ADR-0039). `shell` is that Capability and the only one, so the roster grows
only where the reserved set does. Nothing ships for convenience — no vendor Manifest and no starter
Manifest — because a Provider `hyper` embeds for convenience is one its consumers could have written
themselves, and embedding it ties their correctness to a release cycle, takes a name out of the
namespace permanently, and puts `hyper` in the position of vouching for a description of somebody
else's API.

A built-in's Manifest is ordinary YAML in the binary's own source, embedded verbatim: `operation` writes
those lines back unchanged (§9), and `manifest_digest` covers exactly those bytes (§7). It carries
`schema-version:` like every other Manifest — the field's justification is about who may write a
Manifest rather than about which ones carry it, and a row shape branching on origin would be worse than
a field that is trivially satisfied. It carries no `origin:` block and therefore makes no digest claim
against a registry, which is what a locally authored Provider does too (§7). There is no trusted path
in: the embedded bytes go through the same loader and the same checks §4 states, and a built-in that
fails one is a defect in the binary rather than a Refusal an author can do anything about.

**A built-in is forkable in form and not in power.** Its source is readable, and copying it into
`providers/` is an ordinary edit to a tracked file — but the copy must be renamed, an Extension being
unable to shadow a built-in name, and once renamed it may not declare `shell`.

A `shell` Operation runs its command as a child of the process `hyper` runs in, and what that child
inherits is the invoking environment with **every variable any Target declaration names as a credential
slot removed** — every one in the repository, not only those this Run resolved, so the set is decided
offline and does not turn on which Steps a Run reached. `hyper` knows those names by position (§3),
which is the same knowledge that lets it suppress a credential rather than scan for one (ADR-0007), used
here to keep the credentials it resolved out of a process it cannot describe. Everything else in the
environment is the command's, and `hyper` neither reads it nor records it; §13 states what that costs.

### `install`

`install <ref>` resolves the ref against a registry, fetches the Manifest, verifies the bytes against
the digest published for that ref, and writes the file into `providers/`. Bytes that do not match are
`origin-digest-mismatch` (§12) and nothing is written.

**A ref the registry does not hold exits `1`, not `2`**, which is where this command departs from every
other positional in the tree (§9, ADR-0060). The other eight resolve a name against something this
repository holds — an artefact, a Provider, a Store entry, a path — and `hyper` can say the name is
wrong on evidence it already had. A ref names something in a registry's namespace, and *matches
nothing* is an answer that had to be fetched: it can differ between two invocations of an identical
command line, it is unavailable offline, and it arrives beside the answers — a registry that did not
respond, a resolution that timed out — that are unambiguously the world resisting. `1` is where those
already live, and it keeps exit `2` decidable without a network. `install` therefore carries three
codes: `2` for an invocation the ref grammar rejects, `1` for a ref the registry does not hold or a
fetch that did not complete, and `77` for `origin-digest-mismatch`, which is a check declining bytes
that did arrive.

Registry as source, repository as record. The registry is where an Extension is discovered and
fetched; what executes is the file in the tree, reviewed in the same commit as the Definition that uses
it, and updated by an `install` whose whole effect is a diff. A Run resolves nothing from a registry —
a run-time fetch would be shared mutable state between review and execution.

`install` records the ref it resolved and the digest it verified in the Manifest's own origin block,
which the Manifest schema defines and only `hyper` writes — the rule that lets `project` write the pin
(§3). The digest covers the published bytes, which are the file without the block naming them, since a
digest cannot cover itself. `check` recomputes it and reports `origin-digest-mismatch` where it no
longer holds (§4), which is what makes the fetch's verification repeatable offline, by anyone reading
the repository, long after the machine that performed it is gone.

A Manifest carrying no origin block is a locally authored Provider: checked like any other and making
no digest claim. Editing an installed Manifest therefore means re-installing it or dropping the block,
and dropping it is one more visible edit to a tracked file — which is the only mechanism this path has
ever relied on. It is also a readable edit rather than only a visible one: `provider` reports the ref
and digest where the block is there and reports neither where it is not (§9), so *this Manifest stopped
claiming an upstream* is a fact a caller reads off the surface and not only a diff a human noticed.

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
