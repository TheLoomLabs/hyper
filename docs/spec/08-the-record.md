# §7 — The record

Everything a Run leaves behind is written to one place, in the open, and read back by every later Run
and by every rendering §8 states. This chapter states where that place is, how a file gets into it,
what each file holds, how the current version of a Record is found with nothing pointing at it, what a
Journal entry carries, and what may be removed from it later.

## The Store

The Store is an orphan branch of the same repository, written by every environment that runs — the
laptop and the runner alike (ADR-0006). Writing to it invokes no Operation, consumes no Capability,
and passes no two-key check, the Store not being a Target: it sits beneath the layer Providers exist
at. What a runner needs in order to write it — `contents: write` on the token it was handed — is a
fact about the executor and belongs to the projection §10 states.

The branch is named `hyper-store`, fixed rather than chosen: there is no setting for it, no flag, and
no file it could be configured from (ADR-0014). One repository has one Store, and finding it is knowing
the name.

The branch is created by an explicit act and never by a Run, read-only Runs included. A Run that cannot
find it Refuses (`store-absent`, §12) rather than proceeding against an empty record: a fetch that
failed mid-flight and a branch that never existed look identical from the inside, and reading either as
*there is nothing recorded* disarms every test run-once and `skip-if-recorded` perform (§6, ADR-0006).
Offline-tolerant means the branch exists and is unreachable now; it never means it has never existed.

The branch introduces itself in prose. `STORE.md` is written once, when the Store is created, and says
that every other file on the branch is machine-written, that the branch is the account of the world
rather than part of it, and that editing it by hand is editing evidence (ADR-0011).

### Syncing and durability

An effectful Run syncs the Store before its first effect and Refuses if it cannot (`store-unsynced`,
§12). A read-only Run proceeds offline and pushes when it can.

Nothing on a runner is durable until it is pushed, so §6's guarantee that a crash loses at most the Step
in flight is a fact about when the pushes happen: the open Journal entry is pushed at Run start, the
Store is pushed after every effectful Step, and a Run's reads batch to its end (ADR-0006).

A push rejected as non-fast-forward fetches, rebases, and retries, three times, after which the Run is
`failed` with the contention exit code (`75`, §12) — the same code §6's lock contention takes, both
being a Run that lost the Store rather than a guardrail declining or the world resisting. Almost every
one of those rebases is trivially clean, since every path carries the id of the Run that wrote it and
no two Runs target one path. The one conflict that is not clean — two Runs closing the same open entry
with different outcomes — stands as a conflict rather than being resolved in either direction: it is a
disagreement about what happened, and picking a side is the tool editing evidence.

## Append-only

No file in the Store is ever rewritten, in any path, by anyone (ADR-0011). Every write creates a path
that did not exist: a Record version is a new file, and a Journal entry is a directory that gains a file
as each Step reaches a Disposition and one more when the Run ends. Nothing is edited in place, nothing
is truncated, and nothing carries a marker a later write moves.

Which paths exist, what each segment holds, and how an identity hostile to a filesystem is encoded into
one are a closed grammar — the Store path grammar, named here and defined in §12, and distinct from the
value paths §3 states.

## Records

A Record version is one file, holding that version's projected content and its metadata together, in
canonical JSON: UTF-8, LF line endings, two-space indent, keys sorted by Unicode code point, no
trailing whitespace, and numbers as the shortest decimal that round-trips. One artefact that is
diffable and canonical at once.

A version is written only where the bytes moved. An Operation returning what the head version already
holds mints nothing, and the canonical encoding is what makes *the bytes moved* an exact test rather
than an approximate one. The cost of that — a Record that vanished and a Record that did not change
both write nothing — is paid in the Journal below rather than by adding state to the Record, and §8
states what it buys.

A field a version does not carry is absent from the file, and nothing stands in its place: no empty
value, no marker, and no key.

There are no binary Records, no streaming writes, and no appending inside a version. A Record is the
projection its Manifest declared (§3), and a blob nobody reviews has no business on a branch whose
whole point is that it can be read.

A field a Manifest declares secret is written as a presence-only marker in the position the value would
occupy — no digest, no length, no sibling list of what was suppressed — so no secret reaches the Store
at all (ADR-0007). The marker is a constant, which is what keeps the byte comparison above honest: a
rotated secret writes identical bytes and correctly mints no version. What renders from that, and the
one place a raw response is visible, are §8's.

### Identity on a filesystem

A Record's identity is `(Target, Definition, name)` (§2), and the `name` half is a Manifest-declared
field of an upstream response, which makes it hostile input. How an identity becomes a path segment —
the encoding, the preserved case, the truncation of an over-long one — is part of the grammar §12
states.

Case is the one place the two environments genuinely differ, a laptop's filesystem being usually
case-insensitive and a runner's not, so the rule is `hyper`'s rather than the filesystem's. Writing a
Record whose identity collides case-insensitively with one already in the Store is a Refusal
(`record-identity-collision`, §12), decided by reading the Store, identically on both platforms, and
never by attempting the write and seeing what happens. The remedy for a genuine `Foo` beside a `foo` is
a Manifest identity change, which is a code change and therefore reviewed.

## The Head

Nothing in the Store points at the current version of a Record. The Head is derived by ordering a
series' versions on the `written_at` each version carries — UTC, from the writer, ties broken by the
file name — so finding it is a directory listing rather than the reading of a marker, and two
environments writing one series contend over nothing (ADR-0011). A version identifier is therefore
mintable by either writer alone and never a counter: a Run id is a UUIDv7 (§12).

The version that is current is visible to anyone reading the branch, in a fresh checkout or a browser,
with no git plumbing and no tool.

A Tombstone is an ordinary version of the series, carrying the fact that what it described was
destroyed, the Asset's last known state, the Operation that destroyed it, and the Run that confirmed
it. The series is the one the Expansion acted on, so a Tombstone is written under the Asset's own
identity rather than under a projection of the destroying Operation's response, which need not carry
one. It is terminal for the Asset's life and not for the series: a further version above it makes the
Head alive again, which is what makes destroy-then-recreate behave as §6 states under
`skip-if-recorded`.

## The Journal

A Journal entry is a directory, one per Run, under a date partition (§12). `run.json` is written at Run
start and carries the Run id, the Procedure, the Trigger, whether the Run is a dry-run, and the Run's
Provenance — so a Run that wrote no Record still says which code performed it. A file per Step is
written as that Step reaches its Disposition. `outcome.json` is written when the Run ends and carries
the outcome triple §6 states.

### The open entry

An open entry is one with no `outcome.json`, and that absence is the whole representation — there is no
state field to leave stale and no growing file to rewrite. The Run may be in flight or its process may
be gone, and `hyper` never guesses which. Closing one is §6's rule and stays append-only here: the Run
that closes another's entry creates `outcome.json` and edits nothing the dead Run wrote (ADR-0011).

The last Step file's `ended_at` is when the Run went quiet, and it is the only evidence a Run that never
came back leaves. It is read as a timestamp and never as a verdict.

### Times, not durations

Every file stamps the instant it was written: `run.json` the Run's start, each Step file the instants
that Step began and ended, `outcome.json` the Run's end. No duration is stored anywhere — a stored
duration is a second representation of what the timestamps already carry, and the two can disagree.

Durations derive at render, and only within one entry. Timestamps from two entries are never
subtracted, and no rendering presents a cross-entry interval as a measurement, because the laptop and
the runner do not share a clock.

### What a Disposition holds

A Step's Disposition — one of the six §6 names and §12 defines — is held here rather than by any Record,
and each carries three things beyond its value: the Record identities the Step acted on, the selector
it resolved together with what that selector expanded to and the Bound it was counted against, and
what `hyper` itself did to reach the outcome. A fourth arises in one case only, and is stated below
with it.

The identity set is written as a digest, and in full only where that digest differs from the same Step's
digest in the previous Run of the Procedure. An unchanged listing of five hundred Records costs one
line; a changed one costs the set, and the set it changed from is findable as the last Run whose digest
moved. Every Disposition carries it — a `read` Step's like any other, which is what lets §8 tell a
Record that vanished from one that did not change with no reconciliation and no new state on any
Record, and *attempted, outcome unknown* included, which is the Disposition that knows least and the
one whose identities matter most: the Assets a Run may or may not have destroyed are named there and
nowhere else.

The second is the selector, held as it was authored beside what it resolved to, so that what a Step
reached is readable back from the entry long after the Run and against the artefact revision that
Run's Provenance names. It is what a Refusal's remediation points at (§8) and what `show --expansion`
reads (§9); a Step carrying no selector (§3) resolved none and holds none.

The third is `hyper`'s own account of the work — a Pattern's attempts, its pages, its poll iterations —
supplied by no Provider (ADR-0018). It is what makes *attempted, outcome unknown* after five attempts a
different fact on the page from the same Disposition after one.

A Step halted by a projection that did not resolve (§6) carries the identities it wrote and no others,
and one thing more: the path that failed to project. The set is partial and the path is what says so.
The digest is taken over what the set holds like any other and moves under the same rule, so it says
nothing about partiality either way — two Runs failing on the same member hold the same nine identities
and the digest does not move — and it is the path a reader reads partiality from. That path is held
here and nowhere else: a rendering goes to a terminal that scrolls, and no surface shows the response
it failed against (ADR-0017).

### A Refusal in the Journal

A Refusal writes no Record — nothing happened to the world — so the Journal is the only place it is
held, and it is held in full: the Step file carries the check that declined it under its `error_code`
(§12), the Step, the Target it would have bound, and what was declared against what was found (§5),
beside the entry's own Provenance. An attempt whose outcome never came back is held there as fully and
for the same reason, less the `error_code`: nothing declined it, so there is no check to name (§9).

### The Trigger

A Run's Trigger names what caused the Run and which executor it happened on, and it carries which
occasion on that executor: on Actions the run id, the attempt number, and the job URL; on a laptop the
hostname. It is a string written into the Store rather than egress, and it is what links an entry to the
narration that produced it — without it a Run id and the job that emitted it are unrelatable.

### Dry-run entries

A dry-run writes an entry marked as one (§6): `run.json`, a file per Step it reached, and
`outcome.json`. That entry is evidence that a rehearsal happened and evidence of nothing else, and
every consumer of Journal evidence filters it out — run-once Repeatability (§6), the identity digest
above, and the Comparison as baseline and as subject alike (§8). A rehearsal that counted as evidence
would permanently refuse every run-once Step in the Procedure it rehearsed, with no bypass to recover
through and nothing but an artefact edit left (ADR-0001): the review aid would disarm the tool.

## Provenance

Every Record version carries Provenance, and carries it in full rather than by reference to the Run that
wrote it: the Definition revision, the Manifest digest, the Extension digest, the repository revision,
and the version of `hyper` that performed the write — which, Providers being data, is the only code that
ran (ADR-0004). A version file saying only *see Run `abc`* would be unreadable in a browser and in a
diff, which is exactly where this field set is read.

## Retention and Compaction

Retention is read-time. The policy lives in the Repository declaration (§3) and nowhere else: no flag,
no environment variable, no per-invocation override. A `--keep-versions` would let one invocation remove
more than the repository ever agreed to, which is the shape ADR-0001 removed elsewhere; widening
retention is instead a one-line edit to a reviewed artefact, and it renders as one (§8).

Compaction is an explicit command. It never runs automatically and never on a Cadence, and it removes
interior Observation versions only — never a Head, never a series' first version, never an Asset, never
a Tombstone, never a Journal entry, and never any Provenance. Evidence is what the record exists to
hold: pruning the entry that says a Step *ran* makes it *never reached* again, which is a bypass under a
maintenance name (ADR-0006, ADR-0001). Compaction is an ordinary commit on the Store branch, so `git
log` is its own account of what it removed.

What Compaction reclaims is tree size and scan cost. It reclaims nothing from a clone: git history is
not editable, so every byte ever written to the branch is still fetched by the next clone of it. The
Store therefore grows monotonically, forever, and there is no rollover to a fresh branch — carrying one
forward would have to carry enough Disposition evidence with it to keep run-once refusing, or it is the
same bypass again. This is named here as the honest limit it is and carried forward to §13.

## Orphaned Assets

Deleting a Definition that still owns live, un-tombstoned Assets is legal. `hyper` neither blocks the
deletion nor destroys what the Definition owned, and the Assets it owned stay in the Store as Orphaned
Assets — Expansion needing a Definition (§5), nothing `hyper` can now do reaches them (ADR-0012).

They are reported for as long as they stand rather than once, at the moment they are orphaned:
otherwise a forgotten resource becomes invisible by way of a tidy-up commit. Recovering one means
restoring the Definition, or authoring a new one that declares the same Target and names the resource
by literal identifier (§12); there is no adoption path, adoption being the reconciliation `hyper` does
not perform.

## The schema version

Every file in the Store carries its own schema version, an integer, and the branch carries none.
`hyper` reads any file at or below the version it knows, and Refuses on one written above it
(`store-schema-unsupported`, §12) rather than guessing at a shape it does not recognise (ADR-0028).
Append-only makes migration in place impossible, so the reader accretes format handling and the Store
accretes nothing: a schema change adds new files rather than editing old ones.

It is not the `hyper` version Provenance carries, and it is not the version pin §11 states. It moves
only when a file's shape moves, and no compatibility is inferred from a release version in either
direction.

## Files are authoritative

Every answer `hyper` gives about the record is defined over the files themselves. Finding a Head is a
directory listing; finding the previous Run of a given Step is a backward scan through the date
partitions, stopping at the first match (ADR-0011). Any index is local, ignored by git, droppable, and
rebuildable by a full scan of the branch, and no answer depends on one existing — an index makes those
two workloads faster, never different, and a lost index costs a rebuild rather than data.
