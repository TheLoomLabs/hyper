# §7 — The record

Everything a Run leaves behind is written to one place, in the open, and read back by every later Run
and by every rendering §8 states. This chapter states where that place is, how a file gets into it, the
encoding every file is written in and every key each one carries, how the current version of a Record
is found with nothing pointing at it, and what may be removed from it later.

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

## Canonical JSON

Every JSON file in the Store is written in one encoding, and the encoding is stated in full because it
is not presentation: a version is minted only where the bytes moved, the identity digest below is taken
over these bytes, and a file name breaks a Head tie under them. An unstated separator is a fact two
writers can disagree about.

UTF-8, LF line endings, and a trailing LF. Two-space indent. Keys sorted by Unicode code point. `": "`
after a key; a comma immediately after a value, then the line ending; no trailing whitespace. Numbers
as the shortest decimal that round-trips. An array writes one element per line at the same indent, so a
set that gains a member gains a line and a git diff of it names what moved rather than reporting that
one long line changed. An empty mapping and an empty array are written inline, `{}` and `[]`.

Escaping is the minimum JSON requires, and a character outside ASCII is written as itself in UTF-8
rather than as an escape — so a Record whose name carries an umlaut is legible in a browser and hashes
as what it reads as. The trailing LF is written for the same reason the rest of this is: a file without
one puts `\ No newline at end of file` into every diff of a branch whose whole purpose is being read as
one.

A timestamp is RFC 3339, UTC, `Z` mandatory, with milliseconds always to three digits. The width is
fixed, so lexicographic order over a timestamp is chronological order, and the window in which two Runs
writing one series inside the same second fall through to the file-name tie-break is a thousandth of
what whole seconds would leave.

A key whose value would be an empty mapping or an empty list is absent rather than written empty. Two
places are exceptions and say so where they stand, both for the same reason: absence there already
carries a different meaning.

## Records

A Record version is one file, holding that version's projected content and its metadata together. One
artefact that is diffable and canonical at once.

The projected content nests under `fields`, the key the Manifest's projection is written under (§3),
rather than sitting beside the metadata. A projected field's name is a Provider author's to choose, and
flat would need a reserved list of metadata names for it to steer around — a list that grows, and one
that cannot grow safely here: a name added to it at schema version 2 collides with a field already
written into a version 1 file, and no file in the Store is ever rewritten (ADR-0011). Nested, the two
namespaces are disjoint forever and there is no check to state or forget.

Beside `fields` a version carries `schema_version`; its own identity as `target`, `definition` and
`name`, unencoded and in full; `record_type`, `observation` or `asset`; the `run_id`, `step` and
`operation` that wrote it; `written_at`; and its `provenance`. A version written by a Step reached
through a nested Procedure invocation carries that Step's `path` as well, as the Step file does below.
The identity is restated rather than read back out of the path because the path is
lossy — the Store path grammar truncates an over-long segment and suffixes a hash (§12) — and because
ADR-0011 requires the working tree to describe itself.

```json
{
  "definition": "preview-dns",
  "fields": {
    "created_on": "2026-08-06T11:03:19Z",
    "id": "372e67954025e0ba6aaa6d586b9e0b59",
    "name": "preview-42.example.com"
  },
  "name": "372e67954025e0ba6aaa6d586b9e0b59",
  "operation": "create_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "hyper_version": "1.4.0",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "record_type": "asset",
  "run_id": "01991ea6-b118-7c93-8d41-6b2f7ae05c19",
  "schema_version": 1,
  "step": 1,
  "target": "cloudflare-prod",
  "written_at": "2026-08-06T11:03:19.914Z"
}
```

The Record's `name` is the value its identity path resolved to, so it recurs inside `fields` wherever a
Manifest reads both from one path — which §3 states is the ordinary case and not two facts disagreeing.

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

A field a Manifest declares secret is written as the string `"<secret>"` in the position the value
would occupy — no digest, no length, no sibling list of what was suppressed — so no secret reaches the
Store at all (ADR-0007). The marker is a constant, which is what keeps the byte comparison above
honest: a rotated secret writes identical bytes and correctly mints no version. It is the same constant
every rendering §8 produces uses, so there is one thing to recognise rather than two. A projected value
that happens to read the same is not a case `hyper` disambiguates: `secret:` is declared in a reviewed
Manifest (§3), which is authoritative over what a reader would otherwise infer from a value.

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
a Manifest identity change, which is a code change and therefore reviewed. The one collision that is
authored rather than projected — two members of one `values:` list that are one identity under this
fold — is the same check and the same code, fired at load with no Store in hand (§3).

## The Head

Nothing in the Store points at the current version of a Record. The Head is derived by ordering a
series' versions on the `written_at` each version carries — UTC, from the writer, ties broken by the
file name — so finding it is a directory listing rather than the reading of a marker, and two
environments writing one series contend over nothing (ADR-0011). A version identifier is therefore
mintable by either writer alone and never a counter: a Run id is a UUIDv7 (§12).

The number §8 renders beside a Record is not that identifier and is stored nowhere. It is a version's
**ordinal** position in this ordering, derived from the same listing and unstable under it — a version
arriving beneath one already rendered moves every ordinal above it, and Compaction moves them again
(ADR-0049). Nothing takes one as input: naming a version is naming its Run.

The version that is current is visible to anyone reading the branch, in a fresh checkout or a browser,
with no git plumbing and no tool.

A Tombstone is an ordinary version of the series, and the four things it carries are three ordinary
keys and one marker: `tombstone: true` for the destruction (ADR-0011), the previous Head's `fields`
copied forward for the Asset's last known state, and the `operation`, `run_id` and `step` every version
carries anyway for what destroyed it and what confirmed it. Its `written_at` is when destruction was
confirmed, and §8 renders it as that rather than reading a fifth key. The `fields` were projected by
some earlier Operation and the `operation` names the one that destroyed it, which is the one place in
the Store those two keys describe different calls.

The series is the one the Expansion acted on, so a Tombstone is written under the Asset's own identity
rather than under a projection of the destroying Operation's response, which need not carry one. It is
terminal for the Asset's life and not for the series: a further version above it makes the Head alive
again, which is what makes destroy-then-recreate behave as §6 states under `skip-if-recorded`.

Where the Expansion was an `over:` `values:` list, a member may name no series at all — that being what
the form exists for — and the Tombstone opens one under that member as the Record `name` (ADR-0033).
There is no branch on whether a series was already there: where the literal matches one, the Tombstone
is an ordinary further version of it, and the Store cannot afterwards tell a resource `hyper` built from
one it only ever ended, which is correct because nothing distinguishes them. Such a Tombstone is the
series' first version and it carries **no `fields`** — there is no previous Head to copy forward, and
the key's absence there means `hyper` destroyed this and never observed what it was. A Tombstone is the
one version whose `fields` can be missing for no other reason, so the absence needs no marker beside it.

```json
{
  "definition": "preview-dns",
  "name": "5b2d84f16c0a39e7d5182bfa604c7e93",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "hyper_version": "1.4.0",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "record_type": "asset",
  "run_id": "01991ea6-b118-7c93-8d41-6b2f7ae05c19",
  "schema_version": 1,
  "step": 2,
  "target": "cloudflare-prod",
  "tombstone": true,
  "written_at": "2026-08-06T11:05:41.302Z"
}
```

It is the shortest file the Store holds, and every key in it is one an ordinary version carries. The
`name` came from the Procedure rather than from a response, `record_type` is `asset` because `hyper`'s
effect reached the thing, and the only key an ordinary Tombstone has that this one lacks is `fields`.

This is the one Record `name` in `hyper` whose origin is an author rather than a Manifest-declared field
of an upstream response. Where the two disagree — the author writing a spelling the `identity:` path
would not have projected — a series opens under the spelling, the Tombstone lands in it, and a real
Asset series for the same resource stays standing and reads alive. Nothing catches this: it would take
knowing what the API returns, which is the question §4 states it has no oracle for, and it is carried to
§13 as the limit it is rather than defended by a check that cannot be written.

## The Journal

A Journal entry is a directory, one per Run, under a date partition (§12). `run.json` is written at Run
start, a file per Step is written as that Step reaches its Disposition, and `outcome.json` is written
when the Run ends.

`run.json` carries `schema_version`, the `run_id`, the `procedure`, the `trigger`, `started_at`,
`dry_run`, and the Run's `provenance` — so a Run that wrote no Record still says which code performed
it.

```json
{
  "dry_run": false,
  "procedure": "retire-preview-dns",
  "provenance": {
    "hyper_version": "1.4.0",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "run_id": "01991ea6-b118-7c93-8d41-6b2f7ae05c19",
  "schema_version": 1,
  "started_at": "2026-08-06T11:03:18.204Z",
  "trigger": {
    "actor": "igor",
    "cause": "manual",
    "executor": "local",
    "host": "thinkpad"
  }
}
```

`dry_run` is written on every entry, `false` included, and is the one marker in the Store that does not
follow the absence rule above. Four independent readers filter rehearsals out — run-once Repeatability
(§6), the identity digest below, and the Comparison as baseline and as subject (§8) — and a reader that
takes absence for `false` refuses every run-once Step in the Procedure it rehearsed, permanently, with
nothing but an artefact edit left (ADR-0001). The exception is bought by what getting it wrong costs,
not by the shape of the field.

### The Step file

A Step file carries `schema_version`, the `step` position, the Step's authored `id`, its `definition`,
`operation`, `target` and `kind`, its `disposition`, `started_at` and `ended_at`, its `provenance`, and
the three things a Disposition holds below. A Step reached through a nested Procedure invocation
carries `path` as well — the invocation chain, `retire.probe` — beside its own `id`; a top-level Step
carries none.

```json
{
  "definition": "preview-dns",
  "disposition": "ran",
  "ended_at": "2026-08-06T11:05:44.117Z",
  "id": "retire",
  "identities": {
    "digest": "sha256:6f1c8d0a4b93e527f10c6ba8d34e79521f0badc6e84397b210f5cd6e0a4b7f38",
    "members": [
      "372e67954025e0ba6aaa6d586b9e0b59",
      "9a4c1f0d3b7e5286c1d09f4a7b3e6152",
      "c07b3e91d4a2f5860b3c19e75d2a4f83"
    ]
  },
  "kind": "destroy",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1"
  },
  "schema_version": 1,
  "selector": {
    "bound": 5,
    "declared": {
      "assets": [
        {"field": "name", "starts_with": "preview-"},
        {"field": "created_on", "older_than": "14d"}
      ]
    },
    "expanded_to": [
      "372e67954025e0ba6aaa6d586b9e0b59",
      "9a4c1f0d3b7e5286c1d09f4a7b3e6152",
      "c07b3e91d4a2f5860b3c19e75d2a4f83"
    ]
  },
  "started_at": "2026-08-06T11:05:41.902Z",
  "step": 2,
  "target": "cloudflare-prod"
}
```

`kind` is held here rather than read back from the Manifest because it is the Kind that was in force
when the Step ran, which is the fact §8's third table exists to report as moving — and a Journal whose
Dispositions cannot be read without fetching three artefacts at the revision that Run names is evidence
with a dependency. It is the argument that puts the selector in the file as authored, applied to the
one other fact in the same position.

A Step that is a nested Procedure invocation writes no file of its own. An invocation is not a Step,
none of the six Dispositions describes one, and its own Steps each write a file carrying `path`.

`outcome.json` carries `schema_version`, the `outcome` §6's triple fixes, `ended_at`, and `closed_by_run`
where another Run closed the entry.

```json
{
  "ended_at": "2026-08-06T11:05:49.331Z",
  "outcome": "completed",
  "schema_version": 1
}
```

No exit code is written. `1`, `75`, `130` and `143` are a mapping the CLI applies to `failed` (§12), and
the Store does not restate a rendering. The cost is real and worth naming: a Run that lost the Store
lock and a Run stopped by an interrupt are told apart by their Step files — the first has none — rather
than by a field.

### The open entry

An open entry is one with no `outcome.json`, and that absence is the whole representation — there is no
state field to leave stale and no growing file to rewrite. The Run may be in flight or its process may
be gone, and `hyper` never guesses which.

A Step that was never reached writes no file either, and within a *closed* entry that absence is its
whole representation in the same way. Six Dispositions, five borne by a file and one read from a
silence. A forty-Step Procedure that halted at Step 3 would otherwise write thirty-seven files saying
that nothing happened.

Closing one is §6's rule and stays append-only here: the Run that closes another's entry creates two
paths and edits nothing the dead Run wrote (ADR-0011). It writes the in-flight Step's file first, at
the next `<nnnn>`, `attempted-outcome-unknown` and carrying `closed_by_run`, and then `outcome.json`.
Without that file §6's rule has nowhere to land and the crashed Step reads as never reached, which
re-runs an effect nobody vouched for. Which Step it was is not a guess: `run.json` names the Procedure
and the *repository* revision to load it at, and the highest `<nnnn>` present is the last one that
finished. That is `repo_revision` and never `procedure_revision`, the two doing different work and not
standing in for one another: reconstructing the Step sequence means loading every Procedure the
top-level one invokes, which a commit resolves and a blob id cannot. The Provenance member says which
Procedure file *moved*; the repository revision says where to *find* one.

That file carries what the reaper knows and omits what it cannot establish. Always `schema_version`,
`step`, `disposition` and `closed_by_run`; the Step's `id` and its code facts where the dead Run's
revision resolves them, and absent where it does not, which is every Run that recorded `repo_dirty`.
Never `started_at` — the reaper does not know when the Step began, and filling it would be `hyper`
asserting something about a Run it did not perform, on the surface built to hold what happened.

The last Step file's `ended_at` is when the Run went quiet, and it is the only evidence a Run that never
came back leaves. It is read as a timestamp and never as a verdict.

### Times, not durations

Every file stamps the instant it was written: `run.json` the Run's start, each Step file the instants
that Step began and ended, `outcome.json` the Run's end. No duration is stored anywhere — a stored
duration is a second representation of what the timestamps already carry, and the two can disagree.

Durations derive at render, and only within one entry. Timestamps from two entries are never
subtracted, and no rendering presents a cross-entry interval as a measurement, because the laptop and
the runner do not share a clock.

A reaped entry therefore renders no duration at all. Its `ended_at` is the closing Run's instant on the
closing Run's clock, so subtracting the dead Run's `started_at` from it is the cross-entry subtraction
this rule forbids, wearing one entry's directory. `closed_by_run` being present is what says so; there
is no second flag.

### What a Disposition holds

A Step's Disposition — one of the six §6 names and §12 defines — is held here rather than by any Record,
and each carries up to three things beyond its value: the identities the Step reached a recorded
conclusion about, the selector it resolved together with what that selector expanded to and the Bound
it was counted against, and what `hyper` itself did to reach the outcome. Two more arise in one case
each, and are stated below with them.

**The identity set is what the Step concluded about**, not what it wrote and not what it saw: what it
projected from a response under `read` and `mutate`, and what it confirmed destroyed under `destroy`,
which projects nothing and declares no identity (§3). A Record that came back unchanged mints no file
and is in the set; that is the case the whole mechanism exists for, and it is why *what it wrote* is
the wrong reading (ADR-0030).

It is written as `identities`, a `digest` and — only where that digest differs from the one the same Step
carried in the last Run of the Procedure that carried one — the sorted `members` in full. An unchanged
listing of five hundred Records costs one line; a changed one costs the set, and the set it changed from
is findable as the last Run whose digest moved. `members` is written whenever the digest moved, an empty
list included: this is one of the two exceptions to the empty-value rule above, and it earns it, since
absence there already means *the digest did not move* and a reader would otherwise decode *we looked
and saw nothing* from recognising the digest of `[]` as a constant.

The Run compared against is the last one in which that Step carried a set at all, and never simply the
previous Run (ADR-0055). Two of the six Dispositions below carry no set and a third writes no file, so
the previous Run frequently holds no digest for a Step to differ from, and treating that absence as a
difference would write `members` where nothing moved — which is the one thing their presence says. The
Step is matched by its authored `id` (§3); an `id` that moved is a different Step, with no digest behind
it, writing its set in full on its first Run like any other.

The walk that reads a set back is therefore total. Every entry either holds `members` or, by holding a
digest, names a set an earlier entry holds in full, terminating at the Run where that Step first carried
one. Nothing removes the entries in between — Compaction touches interior Observation versions and never
a Journal entry — so a set and its count are recoverable from any entry the Store holds, which is why
neither is stored a second time.

The digest is `sha256:` over the canonical JSON encoding of the sorted array, trailing LF included — so
where a set is written in full a reader recomputes its digest with `sha256sum` over those exact bytes
and nothing else. Sorting is by Unicode code point, the rule canonical JSON already uses for keys
rather than a second ordering — the same rule §6 orders an Expansion by, over the same names — which
also makes the digest a fact about the set rather than about the order a response happened to arrive
in. It is deliberately not the sequence: what the Step did in what order is `expanded_to` below, and
the two lists differ in order wherever a `values:` list is the selector.

Four Dispositions carry a set and two do not. *ran* carries one. *skipped as already recorded* carries
one — the skip test read a head version, which is a conclusion about that identity. *attempted, outcome
unknown* carries the conclusions it did reach and not the ones it did not: a destroy that confirmed
three of five holds three. *refused* carries none, nothing having been concluded about anything; *skipped
by condition* and *never reached* carry none, and the second writes no file to carry it in.

The second is the selector, held as authored beside what it resolved to, so that what a Step reached is
readable back from the entry long after the Run without a checkout at the revision that Run's
Provenance names. It is what a Refusal's remediation points at (§8) and what `show --expansion` reads
(§9); a Step carrying no selector (§3) resolved none and holds none. `expanded_to` is written whenever
a selector exists, an empty list included — the other exception, for the reason the first one is: an
Expansion that resolved to nothing is not a Step with no selector. It is written in Expansion order
(§6) and not sorted: on a serial `destroy` it is the only place the halt point is legible, and *which
three of the five* is read off it by position.

The Assets a Run may or may not have destroyed are named in `expanded_to` and nowhere else. The
arithmetic a reader does is *expanded to five, concluded about three, two unaccounted for*, and `hyper`
does not say which of the two was in flight: §6 attaches the uncertainty to the attempt rather than to
the thing, and naming one would undo that.

A `values:` selector is held as authored like any other, which gives that entry a second arithmetic no
predicate can offer: the declared list is a list of names, so a member present in `declared` and absent
from `expanded_to` is one the Store already held a Tombstone for (§5). Three authored, two expanded to,
one already gone — readable off the entry without a checkout, and the reason the declared form is kept
beside what it resolved to rather than replaced by it.

The third is `hyper`'s own account of the work — a Pattern's attempts, its pages, its poll iterations —
supplied by no Provider (ADR-0018). It is what makes *attempted, outcome unknown* after five attempts a
different fact on the page from the same Disposition after one. It is written where a Pattern did more
than the trivial single call and absent otherwise, which is the rule §8 renders it under stated once
rather than twice — except on *attempted, outcome unknown*, where it is written whenever a Pattern was
declared at all. How many times `hyper` may have touched the world is the fact that Disposition exists
to carry, and *one attempt* and *no retry declared* are the same silence everywhere else and must not be
here.

An **effectful** Step whose call answered anything but `2xx` carries one thing more, under `answered`:
the host it reached and the status it got. It covers the two cases §6 makes of a non-`2xx` answer — the
halt, and the `404` that completes a `destroy` — and no others, so its presence is the fact that
something other than the ordinary answer decided this Step, and which of the two it was is read from the
Disposition beside it. Where no response arrived at all the `status` inside it is absent, on the rule the
response object carries (§3, ADR-0050).

```json
"answered": {"host": "api.cloudflare.com", "status": 500}
```

A `shell` Step carries the same key with its own Capability's members: the command it ran and the code
it exited with, and the `exit_code` absent where the command could not be started at all (§3, §6). Its
threshold is `0` rather than `2xx`, and it covers one case rather than two, there being no `404` for a
command to answer with.

```json
"answered": {"command": "[\"rm\",\"-rf\",\"/srv/app/releases/r41\"]", "exit_code": 1}
```

`command` is written rather than left to the identity set beside it for two reasons. A `destroy`
projects nothing and declares no identity (§3), so on the Kind where this key matters most there is no
projected `command` anywhere in the entry. And it is what keeps the key from ever being written empty:
a failed exec would otherwise leave `answered: {}`, which the encoding above suppresses outright, and
the fact that something other than the ordinary answer decided this Step would vanish exactly where it
is least ordinary.

It is here rather than on any Record for the reason the Pattern account is (ADR-0018): what it holds is
that a non-`2xx` answer changed what `hyper` did, which is `hyper`'s own conduct rather than the world's
state. That is also why it is effectful-only. A `read`'s status is the answer, and the answer belongs in
the Record wherever its Manifest projected it — an `uptime` version carrying `status: 503` says
everything a second copy in the Journal would, and the one thing a Journal copy would add is a claim
that `hyper` thought a `503` was untoward, which on a `read` it does not (§6). And on a `destroy` it is
the whole of what tells a Tombstone written on `404` from one written on `204`: the Record says the
thing is gone, and nothing there says how `hyper` learned it, which is the line ADR-0010 draws.

A Step halted by a projection that did not resolve (§6) carries the identities it concluded about and no
others, and one thing more, under `projection_failed_path`: the path that failed to project. The set is
partial and the path is what says so.
The digest is taken over what the set holds like any other and moves under the same rule, so it says
nothing about partiality either way — two Runs failing on the same member hold the same nine identities
and the digest does not move — and it is the path a reader reads partiality from. That path is held
here and nowhere else: a rendering goes to a terminal that scrolls, and no surface shows the response
it failed against (ADR-0017).

### A Refusal in the Journal

A Refusal writes no Record — nothing happened to the world — so the Journal is the only place it is
held, and it is held in full: the Step file carries the check that declined it under its `error_code`
(§12), the Step, the Target it would have bound, and what was `declared` against what was `observed`
(§5), beside the `artefact` and `line` the remediation points at. An attempt whose outcome never came
back is held there as fully and for the same reason, less the `error_code`: nothing declined it, so
there is no check to name (§9).

The file and the line are written rather than derived from the Step's own revision. The Journal is
evidence, and evidence that needs a checkout at the right revision to be legible is evidence with a
dependency; the source at that revision is not in the Store at all, so nothing is being said twice. No
phase is written beside them either: each member of the closed `error_code` set names one check and a
check happens at one phase, so §8's `phase` derives from the code and is not a set of its own to drift
from the one it shadows.

### The Trigger

A Run's Trigger names what caused the Run and which executor it happened on, and it carries which
occasion on that executor. It is a mapping rather than a string: four facts whose shape differs by
executor do not pack into one without a grammar and a parser, and a job URL carries every separator
such a packing would use.

`cause` is `cron` or `manual` and `executor` is `github-actions` or `local`, both closed sets §12
holds. `actor` is written on both — the Actions actor, or the operating system user — and `host` on
`local` only. On Actions the occasion is the executor's own `run_id`, its `run_attempt`, and the
`job_url`. §8's header renders `cron` from `cause` and `igor@thinkpad` from `actor` and `host`, so
both forms come from stored facts with nothing invented at render.

`hyper` reads the executor's environment to fill the Trigger and branches on nothing it finds. The
environment is not an authority axis (§5): recording which executor ran is not that act, behaving
differently on one is, and no rule anywhere in this specification reads these two fields.

It is written into the Store rather than sent anywhere (ADR-0021), and it is what links an entry to the
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
wrote it: `procedure_revision`, `definition_revision`, `manifest_digest`, `origin_digest`,
`repo_revision`, and `hyper_version` — which, Providers being data, is the only code that ran
(ADR-0004). A version file saying only *see Run `abc`* would be unreadable in a browser and in a diff,
which is exactly where this field set is read.

`definition_revision` is the git blob id of the Definition file: content-addressed, computable offline
from the working tree, unmoved by a rebase, and equal exactly where the content is.
`procedure_revision` is the same fact about the Procedure file. `repo_revision` is the commit at
`HEAD`. All three are written whole and rendered abbreviated (§8).

`procedure_revision` is the revision of the **top-level** Procedure — the file `run.json`'s `procedure`
names. A Run spans nested Procedures as one Run (§6), so that is the only reading with exactly one value
at Run level, and it has one for every Run, every Run being a Run of a Procedure (ADR-0036). The member
is therefore never absent. It earns its place because the Procedure is the artefact holding the Bound,
the selector, the `target:` a Step binds and every argument value: without it a Record's Provenance names
the code that performed the effect and not the code that decided its extent, while naming a revision for
the Definition, which holds neither (ADR-0048). What it does not carry is a nested Procedure's own
revision — a Bound widened inside one moves nothing here, and is reported by §12's `Bounds` class and
counted by the catch-all like any other line of any other artefact.

`manifest_digest` is SHA-256 over the Manifest's exact bytes — the file in `providers/` for an installed
or locally authored Provider, the embedded bytes for a built-in, which has no blob in the repository at
all. Over the bytes rather than a canonical form of what they parse to, because a second digest of one
Manifest is a second representation that can disagree with the one `install` verified, and because a
reader checks bytes with `sha256sum` and a canonical form with nothing but `hyper`. Reformatting a
Manifest moves it and moves every later Record's Provenance with it, which renders as a code change
(§8) — correct rather than noisy: the reviewed artefact moved.

`origin_digest` is the registry digest `install` verified, the same value §3's installed Manifest carries
in its `origin:` block. It is absent for a built-in Provider and for a locally authored one, neither
having an upstream to have come from. It is not `manifest_digest` under another name even where both
are present: that one covers the file as it stands, this one covers the published bytes, which are the
file without the block naming them (§11) — the file in the repository against the file that arrived.

`hyper` names the algorithm where `hyper` chose it. `manifest_digest`, `origin_digest` and the identity
digest carry `sha256:` inline; the two revisions are bare, the algorithm being the repository's rather
than a choice, and a reader verifies them with `git hash-object`. `hyper_version` is always a release
string: the pin gate refuses any binary whose version differs from the repository's in either direction
(§11), so a binary built from source either reports a released version that matches or never reaches
the Store, and there is no development form to write.

A sixth member is written where it applies: `repo_dirty: true`, where any reviewed artefact the Run read
differs from `HEAD` or is untracked. That is exactly the file set §8's catch-all row counts the moved
lines of, so the marker and the count agree on what code is by construction. It follows the ordinary
absence rule rather than `dry_run`'s exception: one renderer reads it, and reading it wrong costs a `git
diff` command that does not reproduce rather than a Procedure that refuses forever.

Provenance splits by scope across the three files, a member being written at the level where it has
exactly one value and omitted from every level where it has none (ADR-0043). A Record version carries all
of it. `run.json` carries the members that are Run-wide — `hyper_version`, `procedure_revision`,
`repo_revision`, and `repo_dirty` where it applies — which is what makes a Run that wrote no Record
still say which code performed it. A Step file carries the members that are the Step's —
`definition_revision`, `manifest_digest`, `origin_digest` — a Step naming one Definition, one Operation
and one Provider.
Nothing at Run level names a Definition, so a Procedure whose Steps span several has nothing to
disambiguate.

That rule says where a member *may* be written and not where it is restated, and what decides the
second is where the reader stands: a file restates what a reader cannot find in the same directory. A
Step file sits beside `run.json` and reads the Run-wide members one file over, so it carries none of
them; a Record version sits under a Record path with no entry beside it, and carries the whole of
Provenance for the reason this section opens with. The instinct to make every file self-describing is
the wrong one here — it would put `hyper_version` on a Step file and leave one Journal entry holding two
copies of it, which is the second representation the split exists to avoid.

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

What it moves is the ordinal §8 renders: removing an interior version renumbers every version above it,
so a Comparison read before a Compaction and one read after report different numbers for the same two
versions, and a gap that was real closes. Nothing consumes an ordinal (ADR-0049), so what this costs is a
stale rendering rather than an answer.

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

There are four integers, not one — a Record version's, `run.json`'s, a Step file's and `outcome.json`'s
— each independent and each starting at `1`, matching the explicit version a Manifest carries (§3).
`STORE.md` carries none, being prose written once. One integer across the Store would move a Record
version's number when a Step file's shape moved, and an older binary would then Refuse a Record file it
could read perfectly; ADR-0028's rule is that the integer moves only when a file's shape moves, and four
ceilings for the reader is what that sentence costs to be literally true.
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
