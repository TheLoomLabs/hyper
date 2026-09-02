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

An effectful Run syncs the Store once, at Run start, and that sync **is** the push of its open Journal
entry — one reach at the remote rather than two, and the earliest moment at which the Run can know it
will be able to record what it does.

**A read-only Run attempts the sync and tolerates its failure.** It proceeds against whatever branch the
clone holds and pushes when it can; it Refuses `store-absent` only where no branch is in hand *after*
the attempt; and it is never `75` for a sync it could not complete, that code being the effectful Run's
([ADR-0083](../adr/0083-a-read-only-run-attempts-the-sync-and-tolerates-its-failure.md)). Proceeding
offline is surviving a reach that failed rather than declining to make one — a runner's clone holds no
Store until a fetch brings one, so a read-only Run that never reached the remote would Refuse on every
scheduled occurrence, and could not be in a position to tell *unreachable now* from *never existed*
either. It says so on stderr before its first Step: the condition and what it did about it, naming no
remote. That is narration and not a Refusal — no `error_code`, no row, and stdout carries none of it.

The two Runs part here because their syncs gate different things. An effectful Run's is the push of its
own entry, so a sync it could not complete means it cannot record an effect it is about to cause. A
read-only Run's is a fetch, and nothing a `read` Step does is gated on the record — `skip-if-recorded`
is `mutate`-only and run-once is effectful-only (§12) — so the worst a stale Store costs it is a
redundant Record version standing beside the one it duplicates.

**The sync takes the tip and no history.** Where the branch is absent from the clone — which is every
runner, `actions/checkout` taking one ref (§10) — `hyper` creates it with a depth-1 fetch of that one
ref. Where the branch is present it fetches incrementally and names no depth, so a clone that holds the
Store whole keeps it whole. `hyper` never deepens the Store and never shortens one it did not create,
which decides the depth exactly once, at the moment there is nothing to preserve
([ADR-0074](../adr/0074-the-store-branch-is-fetched-shallow-and-whole.md)). It is a depth-1 fetch and
never a filtered one: no blob filter, no tree filter, no partial clone. The Store's history is never
read and its content always is.

Nothing `hyper` answers about the record is defined over the branch's commits. Finding a Head is a
directory listing, finding a Step's previous Run is a backward scan through date partitions, and
append-only makes a year-old Run a read of the tip like any other; Provenance names revisions on the
**code** branch and none on this one (ADR-0011, and *Files are authoritative* below). Content is the
other way round: a version's `written_at` sits inside the file, so ordering a series opens every version
of it, and under a filter each of those is a lazy fetch — which would make *a read-only Run proceeds
offline* false wherever the network is.

`hyper` names the ref explicitly rather than relying on the remote's configured refspec. ADR-0071 gave
the reason as *a checkout leaves it pinned to the one ref it took*, which #246 measured and found false —
`git remote add` writes the wildcard — so the practice is right and the reason for it was not (§10,
[ADR-0132](../adr/0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md)).

### Where the Store sits locally

Nowhere. `hyper` never checks the Store out: it reads the branch's files out of git tree objects and
writes them with `hash-object`, `commit-tree` and `update-ref`, so no byte of Store content is ever an
ordinary file on disk and no worktree, temporary directory or hidden checkout exists to leave behind
([ADR-0075](../adr/0075-hyper-never-checks-the-store-out.md)). *Files are authoritative* below is
unchanged by this: the files are the ones in the tree, and finding a Head is a listing of one.

**The Store therefore has no uncommitted local state, ever.** A write is committed as the call that
produced it confirms — one commit per confirmed write — and pushed after every effectful Step, which is
what §6's crash guarantee rests on. Between two pushes there is no dirty directory a later Run would
have to tell from a hand-edit, and a crashed Run's local branch tip is exactly what it confirmed.

The local ref is `refs/heads/hyper-store`, so a human's `git checkout hyper-store` works on the clone
they already have. There is nothing to keep private: the record being in the open is the thesis, and a
worktree would take that checkout away — `git worktree add` locks the branch to itself.

What does sit on disk is `hyper`'s own, under `.git/hyper/`: the lock §6 states, and any derived state
that makes a Head lookup or a backward scan faster. It is never committed and never part of the record.

**`git` is a subprocess, and it is the one external tool the binary requires.** It is the same `git`
that resolves the credential a checkout left behind (§10), which is what `hyper` fetches and pushes with
(§11, §13).

An effectful Run that cannot complete that sync is `failed` with the contention exit code (`75`, §12)
and is not a Refusal. A read-only Run is never here — its failed sync is tolerated above, and the code
is the effectful Run's. It is the third way a Run loses the Store, beside the lock (§6) and the push
below, and it belongs with them rather than with the guardrails: `77` says a verbatim retry will refuse
identically (§12), and a Run that could not reach the remote succeeds on the same retry five minutes
later. What separates the two codes is whether an act of yours is required — an artefact edit, a `store
init`, a newer binary — and a network coming back is not one (ADR-0061). What it wrote before it
stopped stands locally and goes out with the next Run that syncs.

Nothing on a runner is durable until it is pushed, so §6's guarantee that a crash loses at most the Step
in flight is a fact about when the pushes happen: the open Journal entry is pushed at Run start, the
Store is pushed after every effectful Step, and a Run's reads batch to its end (ADR-0006).

A push rejected as non-fast-forward fetches, re-applies and retries, three times, after which the Run
is `failed` with the contention exit code (`75`, §12) — the same code §6's lock contention takes, both
being a Run that lost the Store rather than a guardrail declining or the world resisting. It is not
`git rebase`, which needs a worktree there is none of (ADR-0075): `hyper` knows exactly which paths are
unpushed — every path in every local commit the remote does not hold, this Run's and any earlier Run's
that stopped before it could push — so it re-applies that set onto the fetched tip's tree and commits.
It is the whole unpushed set and not this Run's alone, which is what makes *what it wrote before it
stopped stands locally and goes out with the next Run that syncs* above true rather than aspirational.

**Every retry is clean**, with no exception anywhere in the Store, and that is a fact about the operation
rather than an observation about how a merge behaves. Disjoint path sets cannot collide, and every path
the Store holds carries the id of the Run that wrote it (§12,
[ADR-0076](../adr/0076-every-store-path-carries-the-id-of-the-run-that-wrote-it.md)) — so two Runs cannot
write one path, two Runs cannot mint one `<run-id>`, and a stranded unpushed write stays pushable
forever rather than colliding on every later sync from that machine.

The one path form that once broke this was the closing write, which named the **dead** Run's id in a
file it wrote itself. It now names both (below), so two Runs closing one entry write two files rather
than one path twice.

The re-application reaches nothing below a shallow boundary, and that is a consequence of append-only
rather than a hope. The branch is only ever appended to and never force-pushed, so every fetch lands on
a descendant of the boundary the previous one set; a local unpushed commit is rooted at the tip `hyper`
last saw, so the merge base of it and the new remote tip is at or above that boundary and is present.
The push is a fast-forward whose parent the remote already holds.

## Append-only

No file in the Store is ever rewritten, in any path, by anyone (ADR-0011). Every write creates a path
that did not exist: a Record version is a new file, and a Journal entry is a directory that gains a file
as each Step reaches a Disposition and one more when the Run ends. Nothing is edited in place, nothing
is truncated, and nothing carries a marker a later write moves.

Which paths exist, what each segment holds, and how an identity hostile to a filesystem is encoded into
one are a closed grammar — the Store path grammar, named here and defined in §12, and distinct from the
value paths §3 states.

## Canonical JSON

Every JSON value the Store holds is written in one encoding, and the encoding is stated in full because
it is not presentation: a version is minted only where the bytes moved, the identity digest below is
taken over these bytes, and a file name breaks a Head tie under them. An unstated separator is a fact
two writers can disagree about.

UTF-8, LF line endings, and a trailing LF. Two-space indent. Keys sorted by Unicode code point. `": "`
after a key; a comma immediately after a value, then the line ending; no trailing whitespace. Numbers
as the shortest decimal that round-trips. An array writes one element per line at the same indent, so a
set that gains a member gains a line and a git diff of it names what moved rather than reporting that
one long line changed. An empty mapping and an empty array are written inline, `{}` and `[]`.

**The encoding is defined over a value, and a file is the case where the value is the whole file**
([ADR-0079](../adr/0079-the-canonical-encoding-is-a-property-of-a-value.md)). A value encoded on its own
is encoded exactly as it would be were it that file's whole content — an array alone opens at no indent
and writes its elements two spaces in, a nested mapping is a mapping like any other and is never
compacted onto one line — which is what the identity digest below is taken over. It governs what the
Store holds and what `hyper` hashes and nothing else: §8's row stream is a second encoding, and is
stated there rather than here.

Escaping is the minimum JSON requires, and a character outside ASCII is written as itself in UTF-8
rather than as an escape — so a Record whose name carries an umlaut is legible in a browser and hashes
as what it reads as. The trailing LF is written for the same reason the rest of this is: a file without
one puts `\ No newline at end of file` into every diff of a branch whose whole purpose is being read as
one.

Every hexadecimal digit `hyper` writes is lowercase — every digest below, every one on §8's wire, and
the SHA-256 that suffixes an over-long path segment (§12). The one exception is a percent-escape in that
same grammar, which is uppercase because RFC 3986 says so, and which is the reason this is stated at all:
one rule with a stated exception is readable, where two conventions and no rule is what an implementer
guesses at.

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
    "created_on": "2026-08-06T09:41:14Z",
    "id": "372e67954025e0ba6aaa6d586b9e0b59",
    "name": "preview-42.example.com"
  },
  "name": "preview-42.example.com",
  "operation": "create_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "hyper_version": "1.4.0",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "record_type": "asset",
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "step": 1,
  "target": "cloudflare-prod",
  "written_at": "2026-08-06T09:41:14.221Z"
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

Case is the one place a reader's two environments genuinely differ, a laptop's filesystem being usually
case-insensitive and a runner's not, so the rule is `hyper`'s rather than the filesystem's. A Record
whose identity collides case-insensitively with one already in the Store may not be written
(`record-identity-collision`, §12), decided by reading the Store, identically on both platforms, and
never by attempting the write and seeing what happens — which `hyper` could not do if it wanted to, a
git tree entry being a byte string and case-sensitive everywhere, so the write always succeeds
(ADR-0075). The filesystem enters only where a **human** checks the branch out, which is exactly the
reading the Head section promises them. The remedy for a genuine `Foo` beside a `foo` is
a Manifest identity change, which is a code change and therefore reviewed. The one collision that is
authored rather than projected — two members of one `values:` list that are one identity under this
fold — is the same check and the same code, fired at load with no Store in hand (§3).

**The check is stated here and fired at §6's Expansion**, which is what decides whether it Refuses or
halts. Where the identity resolves before the call, the Store is read at Expansion beside the members'
comparison against each other, and a collision Refuses with nothing touched. Where `identity:` reads from
the response, the name arrives with the answer and the call has gone out, so the Refusal is unavailable
and the Run halts instead, carrying no `error_code` (§6, ADR-0072). It is never decided at the write: a
guardrail that declines after a call has gone out is not a Refusal, and the write is downstream of every
call that could produce a name to collide.

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
the key's absence there means `hyper` destroyed this and never observed what it was. It needs no second
marker saying so: `tombstone: true` is on the version already, and a reader asks that key rather than
this one.

An **ordinary** version can carry no `fields` too, and it means something else: every path its Manifest
projected resolved to nothing, which is §6's ordinary field absence applied to all of a projection at
once ([ADR-0084](../adr/0084-a-version-carrying-no-fields-is-not-a-tombstones-alone.md)). A `shell`
`read` whose command could not be started at all is where it arrives — the response object is `command`
and nothing else (§3), and the built-in Provider projects `exit_code`, `stdout` and `stderr` (§12) — and
the version is minted, compared and rendered like any other. The two absences are never read as one,
the marker and not the key being what identifies a Tombstone.

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
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "step": 3,
  "target": "cloudflare-prod",
  "tombstone": true,
  "written_at": "2026-08-06T09:43:36.512Z"
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
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "started_at": "2026-08-06T09:41:12.508Z",
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
(§6), the identity digest below, the Comparison's baseline, and the subject the Comparison chooses
for itself (§8) — and a reader that takes absence for `false` refuses every run-once Step in the
Procedure it rehearsed, permanently, with nothing but an artefact edit left (ADR-0001). The
exception is bought by what getting it wrong costs, not by the shape of the field. The fourth is the
only one a caller reaches past, by naming the rehearsal as the subject itself, and what comes back
is what that Run read rather than a claim about what the world became (§8, ADR-0115).

### The Step file

A Step file carries `schema_version`, the `step` position, the Step's authored `id`, its `definition`,
`operation`, `provider`, `target` and `kind`, its `disposition`, `started_at` and `ended_at`, its
`provenance`, and the three things a Disposition holds below. A Step reached through a nested Procedure invocation
carries `path` as well — the invocation chain, `retire.probe` — beside its own `id`; a top-level Step
carries none.

```json
{
  "definition": "preview-dns",
  "disposition": "ran",
  "ended_at": "2026-08-06T09:43:38.105Z",
  "id": "retire",
  "identities": {
    "digest": "sha256:6f1c8d0a4b93e527f10c6ba8d34e79521f0badc6e84397b210f5cd6e0a4b7f38",
    "members": [
      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "kind": "destroy",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1"
  },
  "provider": "cloudflare-dns",
  "schema_version": 1,
  "selector": {
    "bound": 5,
    "declared": {
      "assets": [
        {
          "field": "name",
          "starts_with": "preview-"
        },
        {
          "field": "created_on",
          "older_than": "14d"
        }
      ]
    },
    "expanded_to": [
      "preview-17.example.com",
      "preview-42.example.com",
      "preview-8.example.com"
    ]
  },
  "started_at": "2026-08-06T09:43:35.890Z",
  "step": 3,
  "target": "cloudflare-prod"
}
```

`kind` is held here rather than read back from the Manifest because it is the Kind that was in force
when the Step ran, which is the fact §8's third table exists to report as moving — and a Journal whose
Dispositions cannot be read without fetching three artefacts at the revision that Run names is evidence
with a dependency. It is the argument that puts the selector in the file as authored, applied to the
facts in the same position.

`provider` is the second of them, and it is the Provider's **name** where `manifest_digest` beside it is
the Provider's **bytes**. The digest identifies what ran and answers nothing about what it was: finding
the Runs that read `providers/cloudflare-dns.yaml` without this member means resolving every Step's
`definition` to its `provider:` at that Step's own `definition_revision`, which is the dependency the
paragraph above refuses, one artefact deeper and once per Step. Two surfaces need the answer — a review's
range on a Manifest, and the required-Capabilities and Operation-set classes of §8's third table, both of
which enumerate the Manifests a Run read before they can diff one. It is a Definition that names the
Provider and a Step that names the Definition, so the name is derivable and the point of holding it is
that deriving it costs git objects a shallow clone does not have.

A Step that is a nested Procedure invocation writes no file of its own. An invocation is not a Step,
none of the seven Dispositions describes one, and its own Steps each write a file carrying `path`.

**A Requirement writes none either**, on the same three grounds and with one further consequence: it is
the entry a Run can *end* at while writing nothing about itself. Where its predicate does not hold the
Run halts, and the entry says so where a halt is always said — `outcome.json` is `failed`, and the
Steps after it wrote no file, having never been reached. What the halt *names* — the Requirement, the
field, the Step it read — is on the Run's own fault beside every other halt's, and not in the entry: a
halt carries no `error_code`, and a Requirement made no call for the record to hold (§6, ADR-0116).

`outcome.json` carries `schema_version`, the `outcome` §6's triple fixes, `ended_at`, and `refusal` where
the outcome is `refused` — the whole of what declined it, in the form the next section states. It is
written by the Run whose entry it is and by no other, so it carries no member naming its author: the
`<run-id>` in its path is that member, and another Run's account of the entry is a `closed-by/` file
below.

```json
{
  "ended_at": "2026-08-06T09:43:43.319Z",
  "outcome": "completed",
  "schema_version": 1
}
```

No exit code is written. `1`, `75`, `130` and `143` are a mapping the CLI applies to `failed` (§12), and
the Store does not restate a rendering. The cost is real and worth naming: a Run that lost the Store
lock and a Run stopped by an interrupt are told apart by their Step files — the first has none — rather
than by a field.

### The open entry

An open entry is one holding **no account at all** — neither an `outcome.json` its own Run wrote nor a
`closed-by/` file another Run wrote — and that absence is the whole representation. There is still no
state field to leave stale and no growing file to rewrite: closing has two *forms*, not two writes to
one path. The Run may be in flight or its process may be gone, and `hyper` never guesses which.

A Step that was never reached writes no file either, and within a *closed* entry that absence is its
whole representation in the same way. Seven Dispositions, six borne by a file and one read from a
silence. A forty-Step Procedure that halted at Step 3 would otherwise write thirty-seven files saying
that nothing happened.

Closing one is §6's rule and stays append-only here: the Run that closes another's entry creates **one**
path and edits nothing the dead Run wrote (ADR-0011). It writes
`journal/<yyyy>/<mm>/<dd>/<run-id>/closed-by/<closer-run-id>.json` — inside the dead Run's entry and
under the dead Run's date partition, so *is this entry closed* stays a listing of one directory, and
named by the Run making the claim
([ADR-0076](../adr/0076-every-store-path-carries-the-id-of-the-run-that-wrote-it.md)). It writes neither
`outcome.json` nor a file under `steps/`: those are the owner's paths, and a closer that could take one
is the same-path write §12's grammar now makes impossible.

That file carries what the reaper knows and omits what it cannot establish. Always `schema_version`,
`ended_at`, `step` and `disposition`; the Step's `id` and its code facts where the dead Run's revision
resolves them, and absent where it does not, which is every Run that recorded `repo_dirty`. The
Disposition is `attempted-outcome-unknown` and no other value can appear there — without it §6's rule
has nowhere to land and the crashed Step reads as never reached, which re-runs an effect nobody vouched
for. It is written out rather than assumed from the file's existence, `outcome` being the key that is
not: the entry's outcome is a question about the **entry**, which this file's existence answers in full,
where a Disposition is a fact about a **Step** that this file is the only carrier of, and §8 reads
Dispositions generically across all seven values. The file carries no member naming its author either,
its path being that member, and never `started_at` — the reaper does not know when the Step began, and
filling it would be `hyper` asserting something about a Run it did not perform, on the surface built to
hold what happened.

Which Step it was is not a guess: `run.json` names the Procedure and the *repository* revision to load it
at, and the highest `<nnnn>` present is the last one that finished, so `step` is the one after it. That
is `repo_revision` and never `procedure_revision`, the two doing different work and not standing in for
one another: reconstructing the Step sequence means loading every Procedure the top-level one invokes,
which a commit resolves and a blob id cannot. The Provenance member says which Procedure file *moved*;
the repository revision says where to *find* one.

The last Step file's `ended_at` is when the Run went quiet, and it is the only evidence a Run that never
came back leaves. It is read as a timestamp and never as a verdict.

**A contested entry** holds both — an `outcome.json` its own Run wrote and a `closed-by/` file another
Run wrote — and it is what a reap of a Run that was alive after all leaves behind. Both stand and
neither is removed: the reaper's file truthfully records that a Run inferred death at an instant, and
the owner's truthfully records what the Run went on to do. **The entry's outcome is the owner's wherever
one exists.** An `outcome.json` is its own Run's observation; a `closed-by/` file is another Run's
inference about that Run, drawn from a silence. Where the two disagree the observation is what happened
and the inference stays true of the Run that drew it. `hyper` picks no side between two accounts of
*what the world did* and never will; this is not that, and holding both files is what keeps it from
becoming that.

Where an entry holds `closed-by/` files and no `outcome.json`, the Run really did not come back: the
entry is `failed`, reaped exactly as before, and where several closers landed the close instant is the
**earliest** `ended_at` among them — the first inference, later ones adding nothing but their own
existence. All of them stand.

### Times, not durations

Every file stamps the instant it was written: `run.json` the Run's start, each Step file the instants
that Step began and ended, `outcome.json` the Run's end. No duration is stored anywhere — a stored
duration is a second representation of what the timestamps already carry, and the two can disagree.

Durations derive at render, and only within one entry. Timestamps from two entries are never
subtracted, and no rendering presents a cross-entry interval as a measurement, because the laptop and
the runner do not share a clock.

A reaped entry therefore renders no duration at all, and §8's header names that absence in the cell.
Its `ended_at` is the closing Run's instant on the closing Run's clock, so subtracting the dead Run's
`started_at` from it is the cross-entry subtraction this rule forbids, wearing one entry's directory.
**The entry's account being a `closed-by/` file is what says so**, and there is no second flag: the file
that holds the instant is the file whose path names a different Run, so the two facts arrive together
and cannot come apart. The same reasoning is why a Comparison takes its window's ends from the last Step
file's `ended_at` on such an entry rather than from this one (§8) — a cutoff on the closing Run's clock
reaches every Run in between.

None of it reaches a **contested** entry. There the account is the owner's `outcome.json`, written on the
owner's clock inside the owner's entry, so the duration derives normally and the window end is that
`ended_at` like any other Run's. The `closed-by/` file beside it is not an endpoint of anything: it is
another Run's instant, and this rule is exactly what stops it being read as one.

### What a Disposition holds

A Step's Disposition — one of the seven §6 names and §12 defines — is held here rather than by any Record,
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
previous Run (ADR-0055). Three of the seven Dispositions below carry no set and a fourth writes no file, so
the previous Run frequently holds no digest for a Step to differ from, and treating that absence as a
difference would write `members` where nothing moved — which is the one thing their presence says. The
Step is matched by its authored `id` (§3); an `id` that moved is a different Step, with no digest behind
it, writing its set in full on its first Run like any other.

The walk that reads a set back is therefore total. Every entry either holds `members` or, by holding a
digest, names a set an earlier entry holds in full, terminating at the Run where that Step first carried
one. Nothing removes the entries in between — Compaction touches interior Observation versions and never
a Journal entry — so a set and its count are recoverable from any entry the Store holds, which is why
neither is stored a second time.

The digest is `sha256:` over the canonical JSON encoding of the sorted array, trailing LF included — the
array as it would be written **alone**, at no indent, and never as it sits inside this entry, where
`members` carries four spaces of it. Where a set is written in full a reader recomputes it with
`sha256sum` over those exact bytes and nothing else:

```
[
  "ci-macos",
  "ci-riscv",
  "ci-x86",
  "über-vm"
]
```

`sha256:a118a517431e241eac83559919ae969346bf5a3bf6e06c6db3e636f378fcdf12`. The umlaut is in the example
rather than a fourth ASCII name because it pins what the prose above cannot: a character outside ASCII
is written as itself, so an implementation escaping it to `\u00fc` produces a plausible digest and no
signal at all, and code point order puts it last, which `LC_ALL=C sort` reproduces exactly, UTF-8 byte
order being code point order. The empty set is `[]` under the inline rule above, so its digest is the
constant `sha256:37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570` — printed here
because a set that moved to empty writes `members` in full beside it, so recognising the constant tells
a reader nothing the file did not already say.

Sorting is by Unicode code point, the rule canonical JSON already uses for keys
rather than a second ordering — the same rule §6 orders an Expansion by, over the same names — which
also makes the digest a fact about the set rather than about the order a response happened to arrive
in. It is deliberately not the sequence: what the Step did in what order is `expanded_to` below, and
the two lists differ in order wherever a `values:` list is the selector.

Three Dispositions carry a set and four do not. *ran* carries one. *skipped as already recorded* carries
one — the skip test read a head version, which is a conclusion about that identity. Under
`skip-if-recorded` those two are one set at two granularities: the test decides per member (§6,
ADR-0056), so a Step that called for one member and skipped two holds all three, and its digest does
not move as a `values:` list fills in one member at a time. *attempted, outcome
unknown* carries the conclusions it did reach and not the ones it did not: a destroy that confirmed
three of five holds three. *refused* carries none, nothing having been concluded about anything; *skipped
by condition* and *never reached* carry none, and the second writes no file to carry it in.

*attempted, world untouched* carries none, and it is the fourth on that side rather than a *ran* Step
whose set happens to be empty: it is the value only where no call this Step made reached the world (§6,
ADR-0062), so nothing was concluded about anything, by construction rather than by circumstance. That
distinction is the one §8's dash renders, and it is why the same fact makes this the one failure that is
not Repeatability evidence — a Step that concluded about nothing is a Step no later Run can read
evidence off.

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

An **effectful** Step carries one thing more, under `answered`: **one entry per member of its Expansion
whose call did not answer `2xx`**, in Expansion order, each holding the host it reached, the status it
got, and — where the Step resolved a selector — the `member` of `expanded_to` it is about. It covers the
three cases §6 makes of an answer that was not the ordinary one — the halt, the `404` that completes a
`destroy`, and the request that never left at all — and no others, so an entry's presence is the fact
that something other than the ordinary answer reached that member, and **which of the three ended the
Step** is read from the Disposition beside the list rather than off any entry in it. Where no response
arrived the `status` inside an entry is absent, on the rule the response object carries (§3, ADR-0050),
so a member whose request never left writes the host alone. It says the request did not arrive and never
which of ADR-0018's members stopped it (§12, §13, ADR-0062).

```json
"answered": [
  {"member": "preview-01.example.com", "host": "api.cloudflare.com", "status": 404},
  {"member": "preview-02.example.com", "host": "api.cloudflare.com", "status": 500}
]
```

**It is a list because the Step is one and the calls are many** (ADR-0126). A `destroy` expanding over
five Assets is answered five times, and a key holding one of the five says only that a non-`2xx` answer
reached this Step: which of the Tombstones it wrote was written on a `404` would not be recoverable, and
two Runs in which a different Asset had already gone would leave byte-identical entries. That fact is
carried here and nowhere else — ADR-0050 put it off the Record deliberately and onto this key — so a
per-Step key kept the relocation only for the Step whose Expansion is one member, which is not the
ordinary shape of a `destroy`.

A member that halted the Step writes the **last** entry, an effectful Expansion stopping at the first
error with everything in front of it already committed (§6). A halt that names no answer at all — the
deadline — writes none, and the entries in front of it stand as what those members were told and make no
claim about what ended the Step: the Disposition is where that has always been read from, and a key that
is per member no longer has to drop an earlier member's answer to keep a per-Step reading honest.

A Step that resolved no selector writes one entry naming no member, which is the silence `expanded_to`
keeps on the same Step: there is no member for it to name, and an entry naming one would name something
the entry does not otherwise hold.

A `shell` Step's entries carry the same `member` with its own Capability's members beside it: the command
it ran and the code it exited with, and the `exit_code` absent where the command could not be started at
all (§3, §6). Its threshold is `0` rather than `2xx`, and it covers two of the three cases rather than
all of them, there being no `404` for a command to answer with: a nonzero exit that halted the Step, and
a child that never started, which is *attempted, world untouched* under the other Capability and carries
the command alone.

```json
"answered": [{"command": "[\"rm\",\"-rf\",\"/srv/app/releases/r41\"]", "exit_code": 1}]
```

`command` is written rather than left to the identity set beside it for two reasons. A `destroy`
projects nothing and declares no identity (§3), so on the Kind where this key matters most there is no
projected `command` anywhere in the entry. And it is what keeps an entry from ever being written empty:
a failed exec would otherwise leave `{}`, which the encoding above suppresses outright, and the fact
that something other than the ordinary answer decided this Step would vanish exactly where it is least
ordinary. A `member` alone does not save it — the entry would then say which member and nothing about
what it was told — so an entry naming neither a host nor a command is one `hyper` will not write.

It is here rather than on any Record for the reason the Pattern account is (ADR-0018): what it holds is
that a non-`2xx` answer changed what `hyper` did, which is `hyper`'s own conduct rather than the world's
state. That is also why it is effectful-only. A `read`'s status is the answer, and the answer belongs in
the Record wherever its Manifest projected it — an `uptime` version carrying `status: 503` says
everything a second copy in the Journal would, and the one thing a Journal copy would add is a claim
that `hyper` thought a `503` was untoward, which on a `read` it does not (§6). And on a `destroy` it is
the whole of what tells a Tombstone written on `404` from one written on `204`: the Record says the
thing is gone, and nothing there says how `hyper` learned it, which is the line ADR-0010 draws. The
`member` is what carries that across an Expansion, and it is a name off `expanded_to` rather than a
second copy of the Record for the same reason the key is here at all — the Record says what stands, and
the entry says which of this Step's calls was answered that way.

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
held, and it is held in full. It is held on **`outcome.json` and nowhere else**, under `refusal`, and
the Step file carries none of it (ADR-0061). A Run has at most one Refusal ever, the outcome being
terminal, so what declines a Run is a fact about the Run rather than about any Step — and most of the
closed `error_code` set declines before Step 1, where there is no Step for it to be a fact about (§6).
The reader that knows a Run refused is already holding the file that says why.

`refusal` is an **ordered array of at least one member**, and the array is the checks that declined
this Refusal rather than several Refusals. Each member carries what a `check` problem row carries — the
`error_code` (§12), the `file`, the `line`, the `field` and the `message` — plus what a Run adds:
`step` and `step_id` where the check cites a Step, and `declared` against `observed` where the check
compared two values (§5). Nothing is invented to fill a member that does not apply: a check that
compared nothing writes no `declared`, on the absence rule above, rather than reporting a value for
`observed` it never had.

A Refusal and a `check` problem are one shape because they are one thing arriving through two commands:
what `check` reports offline is what stops a Run online. The key is `file` and not `artefact` — the word
`hyper` reserves for the five reviewed artefacts (§3) — because two codes cite a file that is not one:
`projection-stale` cites a generated workflow and `store-schema-unsupported` cites a Store file. It is
also the word the `EDIT ONE OF` table's column already carries (§8).

```json
"refusal": [
  {
    "declared": 5,
    "error_code": "bound-exceeded",
    "field": "steps[2].bound",
    "file": "procedures/retire-preview-envs.yaml",
    "line": 33,
    "message": "expansion resolved 23 assets on staging",
    "observed": 23,
    "step": 3,
    "step_id": "retire"
  }
]
```

`step` is an **artefact coordinate and never an execution fact**: `bound-missing` cites `steps[2]` in a
Run where no Step was ever reached, and a Step it names may have no file in the entry at all. A Refusal
before Step 1 writes no Step file, none having reached a Disposition, so such an entry is `run.json` and
`outcome.json` and nothing else.

**The array has more than one member only where the phase evaluates many checks together.** Two do:
`check` re-run at Run start, which reports every problem it finds rather than the first (§9), and the
credential pass, which resolves every slot the Run's bindings require in one go (§6) and so knows every
absent variable at once. Everywhere else it has exactly one member, a Refusal being terminal — there is
no second check to reach. The order is the order `check` prints in, by file path and then by line,
and what the terminal line and the `outcome` row name is the first member's `error_code` (§8). That head
is derived and never stored: a stored head is a second representation of the array's first member and the
two can disagree, which is the reason no exit code, no duration and no Head marker is stored either.

An attempt whose outcome never came back is held on its Step file as fully and for the same reason, less
the `error_code`: nothing declined it, so there is no check to name (§9).

The file and the line are written rather than derived from the artefact's own revision. The Journal is
evidence, and evidence that needs a checkout at the right revision to be legible is evidence with a
dependency; the source at that revision is not in the Store at all, so nothing is being said twice. No
phase is written beside them either: each member of the closed `error_code` set names one check and a
check happens at one phase, so §8's `phase` derives from the code and is not a set of its own to drift
from the one it shadows. That rule is what fixes `store-schema-unsupported` at Run start (§6) rather
than wherever a read happens to land: a code whose phase depends on which Step read first has no phase
to derive.

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
above, and the Comparison's baseline (§8). A rehearsal that counted as evidence would permanently
refuse every run-once Step in the Procedure it rehearsed, with no bypass to recover through and
nothing but an artefact edit left (ADR-0001): the review aid would disarm the tool.

**The Comparison's *subject* is the one end a caller may name a rehearsal as.** No rule ever picks one
for either end: the subject a window chooses for itself passes a rehearsal over, and so does the
baseline behind any subject. What `changes --subject <run-id>` asks is *what did this rehearsal read*,
which the paragraph below is the reason there is an answer to, and the answer is a claim about a read
rather than about what the world became (§8, ADR-0115). Every filter above is untouched.

**The marker falls on the entry's side of the entry/version line, and no Record version carries one.**
It is a question asked of a version all the same: a dry-run performs the reads it reaches, so its
Observations are versions like any other (§6), and where a rehearsal preceded the Run that acted, the
pre-state a reader came for can exist under the rehearsal's `run_id` alone. A reader who carries the
paragraph above to the Record store — a different store, where §6 says the opposite — discards exactly
the versions holding the account. The version restates Provenance because it sits under a Record path
with no entry beside it, and that argument reaches this marker too; what does not reach it is the
recovery. Provenance has been on every version ever written, where a marker added to the shape now
would answer only the versions written after it and leave every version already on a branch silent —
so the surface that renders it reads the entry regardless, and a member on the file would be a second
place to read one fact from, answering less than the first. `records` makes that join inside the one
call and renders the answer on its row (§9, ADR-0114), and the Comparison renders the versions
themselves for the Run a caller names as its subject (§8, ADR-0115).

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

**A blob id is computed from the working tree, so it resolves exactly where the artefact was
committed** — and `hyper` never writes one. The id is right about the content either way: it is git's
own object name over the bytes that ran, so it compares equal to a later `git hash-object` of the same
file whether or not any object database holds it. What it will not do, where nothing committed those
bytes, is come back from a `git show`: the Record then names a revision that is in no commit, and the
`repo_dirty` marker below is the same fact stated on the entry. `hyper` writing the blob itself would
make every id resolve for as long as git kept it and no longer — an object nothing references is
unreachable, and unreachable loose objects are what `git gc` prunes — so the promise would expire on a
clock nobody set, on a member whose whole point is that a rebase does not move it. Provenance is
therefore what it says it is and no more, and the act that makes it resolve is the author's: commit the
artefacts, and then run. The orientation says so in the loop, `run` warns on stderr before its first
Step where the tree it read has not been committed, and a `review` whose baseline is such a revision
says so rather than saying the clone is behind (§8, §9, ADR-0119).

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
reader checks bytes with `sha256sum` and a canonical form only where something has written that form
out for them. The identity digest above is the one place anything has: its set is in the entry, in the
encoding it was hashed in. A Manifest's parse tree is nowhere, and re-encoding one to check a digest is
work only `hyper` can do. Reformatting a
Manifest moves it and moves every later Record's Provenance with it, which renders as a code change
(§8) — correct rather than noisy: the reviewed artefact moved.

`origin_digest` is the registry digest `install` verified, the same value §3's installed Manifest carries
in its `origin:` block and the same value `provider` reports beside it (§9). It is absent for a built-in
Provider and for a locally authored one, neither having an upstream to have come from — which is a
different fact from a Provider's origin, that one naming where the bytes load from and having two
answers rather than three (§12, ADR-0073). It is not `manifest_digest` under another name even where both
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
diff` command that does not reproduce rather than a Procedure that refuses forever. What the renderer
that reads it right does — suffix every revision that entry supplies and suppress the command — is §8's.

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
log` is its own account of what it removed — as far back as the clone reaches, which for a Store
`hyper` fetched shallow is one commit. The two meet on no path `hyper` projects: this command never
runs on a Cadence and the shallow Store is the runner's (§10, ADR-0074).

What it moves is the ordinal §8 renders: removing an interior version renumbers every version above it,
so a Comparison read before a Compaction and one read after report different numbers for the same two
versions, and a gap that was real closes. Nothing consumes an ordinal (ADR-0049), so what this costs is a
stale rendering rather than an answer.

What Compaction reclaims is tree size, scan cost, and the size of every later shallow fetch — the sync
above taking the tip, what it pays for is what Compaction just made smaller. It reclaims nothing from a
clone: git history is not editable, so every byte ever written to the branch is still fetched by the
next clone of it. The Store therefore grows monotonically, forever, and there is no rollover to a fresh
branch — carrying one forward would have to carry enough Disposition evidence with it to keep run-once
refusing, or it is the same bypass again.

What it reclaims at no depth is the **Journal**. This command removes interior Observation versions and
nothing else, and a version is minted only where the bytes moved, so a Record read every five minutes
and never changing costs one file — while every one of those Runs writes an entry, and every entry
stands forever. The term that grows with the Cadence rather than with the world is the term nothing
reclaims. This is named here as the honest limit it is and carried forward to §13, which states it as
the two curves and two payers it is.

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
(`store-schema-unsupported`, §12) rather than guessing at a shape it does not recognise (ADR-0028). A
Run tests it at Run start over the files it will read (§6) rather than wherever a read lands, so the
code has one phase like every other member of the set.

The rule Refuses on a shape a reader does not recognise; it does not stop that reader writing the shapes
it does. A Run declining with this code has already written its own entry into the Store beside the
files it could not read, which is exactly what five independent integers exist to permit — and the trace
matters most here, one environment upgrading being the likeliest reason the other stopped (ADR-0061).

There are five integers, not one — a Record version's, `run.json`'s, a Step file's, `outcome.json`'s and
a `closed-by/` file's — each independent and each starting at `1`, matching the explicit version a
Manifest carries (§3).
`STORE.md` carries none, being prose written once. One integer across the Store would move a Record
version's number when a Step file's shape moved, and an older binary would then Refuse a Record file it
could read perfectly; ADR-0028's rule is that the integer moves only when a file's shape moves, and five
ceilings for the reader is what that sentence costs to be literally true.
Append-only makes migration in place impossible, so the reader accretes format handling and the Store
accretes nothing: a schema change adds new files rather than editing old ones.

It is not the `hyper` version Provenance carries, and it is not the version pin §11 states. It moves
only when a file's shape moves, and no compatibility is inferred from a release version in either
direction.

## Files are authoritative

Every answer `hyper` gives about the record is defined over the files themselves. Finding a Head is a
directory listing; finding the previous Run of a given Step is a backward scan through the date
partitions, stopping at the first match (ADR-0011). `hyper` may keep local derived state that makes
those two workloads faster: it lives under `.git/hyper/`, is never committed, is always droppable, and
is rebuildable by a full scan of the branch. No answer depends on one existing — it makes those two
workloads faster, never different, and losing it costs a rebuild rather than data. It is nameless on
purpose: it is optional, nothing else refers to it, and the word this sentence used to spend on it is
already carrying the gutter's sense in §8 and §12 and `git`'s own in the section above.
