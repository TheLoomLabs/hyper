# §8 — Review and Comparison

Two surfaces carry the half of the thesis that says nothing changes unseen: a Definition review, before
anything runs, and a Comparison, after. This chapter states both, together with the Refusal rendering
that is the whole path back from a declined Step, the line every Run ends with, and the row stream all
of them emit under `--json`.
The renderings below are what `hyper` produces rather than an illustration of it; the commands that
produce them and the exit codes they carry are §9's.

None of it is a guardrail; every check standing between an authored artefact and the world is static
and sits before the Run (§4, §5). What that costs is stated at the end of this chapter (ADR-0010).

Colour and width are the only differences between an interactive rendering and one in a CI log. Every
fact on every surface is legible with colour off and through a pipe; colour re-emphasises what a marker
or a word already says and never carries a fact of its own (ADR-0015).

## The Definition review

The object under review is the artefact itself, annotated in place by a gutter, with a table for what is
assembled from elsewhere and one editorial surface (ADR-0026). Three renderings sit inside one screen,
and each has its own rule.

The **gutter** marks, beside the lines that make the claim, each Step's Kind, the Target it binds, its
Bound, and its envelope check. A `mutate` Step with no declared Bound is marked `mutate!`: its absence
is not a static check's business and it is rendered here instead (§4). A comment renders verbatim
beside the line it sits on and is read for nothing else (§3). A nested Procedure invocation renders
under the invoking Step's path with the transitive envelope §3 states. Where the artefact has a
previous revision, the review renders the range and the gutter marks every line that moved.

**`AUTHORITY`** is the one table, because it is assembled from a Definition and a Target declaration
together and no gutter on this file could hold it: the claimed Kinds against the accepted Kinds, their
intersection, and the `destroy` Operations the Definition names (§5). Granularity following severity
(§12), the claimed-Kinds column carries `destroy` where the Definition names any: it is derived at
that one position rather than read.

**`FLAGS`** is the one editorial surface. Every row cites a line the gutter already marked and
introduces no claim of its own; a flag citing a line the gutter did not mark is a defect in the
renderer rather than a rendering.

A review resolves no credential, reaches no network, and invokes nothing.

```
$ hyper review procedures/retire-preview-envs.yaml

  BLAST RADIUS      │  procedures/retire-preview-envs.yaml     a91f0c2 → working tree
  ──────────────────┼──────────────────────────────────────────────────────────────
                    │   kind: procedure
                    │   procedure: retire-preview-envs
  envelope ✓        │   targets: [local, staging]
                    │   steps:
  read     local    │     - id: probe
                    │       definition: uptime
                    │       operation: check_http
                    │       target: local
                    │       over:
                    │         observations:
                    │           - field: name
                    │             starts_with: preview-
                    │
  mutate!  staging  │     - id: label
                    │       definition: hetzner-staging
                    │       operation: set_server_labels
                    │       target: staging
                    │       over:
                    │         assets:
                    │           - field: labels.role
                    │             equals: preview
                    │
  DESTROY  staging  │     - id: retire
                    │       definition: hetzner-staging
                    │       operation: delete_server
                    │       target: staging
                    │       over:
                    │         assets:
                    │           - field: labels.role
                    │             equals: preview
                    │           - field: created_at
                    │             older_than: 14d
                    │ -     bound: 3
                    │ +     bound: 5

  AUTHORITY   assembled from definitions/ and targets/
  DEFINITION       TARGET   DEFINITION KINDS     TARGET KINDS         EFFECTIVE  DESTROY OPS
  uptime           local    read                 read                 r          —
  hetzner-staging  staging  mutate destroy       read mutate destroy  m d        delete_server

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  UNBOUNDED  line 14  step label    mutate with no declared bound
  DESTROY    line 23  step retire   delete_server, bound 5
  WIDENED    line 33  step retire   bound 3 → 5 since a91f0c2
  ENVELOPE   line 3   ok            no step reaches a target outside [local, staging]
```

## The Comparison

A Comparison is retrospective and has no prospective counterpart: it renders a Run that has happened
against the Run before it, and nothing in `hyper` renders a proposed change before it happens
(ADR-0010). It is organised by which of three actors did the changing.

### The window

The baseline is the previous Run of the same Procedure, so a monitoring Run is never compared against a
provisioning one. That window is total rather than partial: every Run is a Run of a Procedure
(ADR-0036), so no Run reaches the world outside some Procedure's Comparison. `since <t>` is sugar for *take the last Run before that instant and fold everything
after it into one rendering*; `between` names two Runs directly; a whole-Store mode compares across
every Procedure at once. Those parameters — `since` or `between`, `target`, `kind`, `limit` — are typed
and closed, and there is no predicate dialect over them: a caller wanting an arbitrary filter takes the
rows and applies it themselves (ADR-0013).

An outcome does not disqualify a baseline, a refused Run's completed Steps having reached the world
like any other's. A dry-run entry is disqualified as baseline and as subject alike (§7), and a Probe
writes no Journal entry and can never be either (ADR-0009). Where no baseline exists the header says so
as a named state — *no baseline — first Run of `<Procedure>`* — with every Record rendering as created
or appeared.

The header names both Runs, each with its id, its Trigger, when it started, its outcome, and how long
it took. A duration derives within one Journal entry; two entries' timestamps are never subtracted (§7).

### Three tables

One table per actor, Assets first: `YOU DID THIS`, `THE WORLD MOVED`, `THE CODE MOVED`. The split is by
actor rather than by field, and Asset against Observation is two tables rather than one column with two
values (ADR-0026).

Assets render `created`, `changed`, or `destroyed`; Observations render `appeared`, `changed`, or
`vanished`. A Tombstone is a marker inside the Asset table rather than a class of its own. A series
whose first version is a Tombstone (§7) renders `destroyed` and never `created`, though the baseline
holds no version of it and the subject does: what the subject holds is a destruction, and reading
*absent, then present* as a creation would report the opposite of what happened. It needs no marker to
be told apart from the destruction of an Asset `hyper` built — an ordinary `destroyed` row carries the
last known state's fields and this one has none, so *destroyed, and `hyper` never saw what it was* reads
off the empty column rather than out of a second rendering of the same fact. There is no
rename class: identity is a Manifest-declared field of an upstream response (§7), so a rename is an
unfamiliar name appearing and a familiar one going quiet, and it renders honestly as both.

Every row in all three tables carries its Target, its Definition, its name, and its change. Asset and
Observation rows add the fields that moved: a scalar leaf renders `path: old → new`, truncated with a
marker where it is long, and anything nested or large renders `path: changed`.

*Vanished*, *appeared*, and *nothing moved* are derived from the identity sets each Step's Disposition
carries (§7) rather than from the Records, which is what buys a disappearance a row at all — an
unchanged Record and a Record that stopped existing both write nothing. No Record gains state for it and
nothing is reconciled.

A partial set is read for what it holds and never for what it omits. Where a Step's Disposition carries
the path a projection failed on (§6, §7), an identity missing from its set is one `hyper` did not read
rather than one the world removed: it gets no row as subject, where it would otherwise render
*vanished*, and none as baseline, where the same identity returning in the next Run would otherwise
render *appeared*. The entry stands as a baseline like any other and what that Step did write renders
like any other Run's; it is that one set's absences that say nothing.

The three tables never join an Observation series to an Asset series. That join is the drift detection
`hyper` has no engine for and never performs (ADR-0010).

A Refusal gets no row. Nothing reached the world, so it is not a change; it appears in the header as a
Run in the window and is one `runs` query away. A Run whose every Step skipped renders no rows at all,
which is how the Dispositions read back distinguish it from a Run that did the work (§6).

Pattern facts — a retry's attempts, a poll's iterations, a paginated read's pages — do not enter the
three tables. They render on `runs` and only where they are not the trivial single call (§7).

`THE CODE MOVED` reports over the closed enumeration of code facts §12 defines, terminated by the
mandatory catch-all row §12 states, counting every other line of every reviewed artefact that moved.

```
$ hyper changes --since 2026-08-04T09:12:00Z

  retire-preview-envs
  BASELINE  01991c3a-7d40…  cron           Tue 4 Aug 09:12  completed  1m48s  rev a91f0c2
  SUBJECT   01991ea6-b118…  igor@thinkpad  Thu 6 Aug 11:03  completed  2m31s  rev 4d7e118

  YOU DID THIS   5 assets
  CHANGE     TARGET   DEFINITION       RECORD        VERSION  FIELDS
  destroyed  staging  hetzner-staging  preview-8801  4 → 5    † confirmed 11:02
  destroyed  staging  hetzner-staging  preview-8802  3 → 4    † confirmed 11:02
  destroyed  staging  hetzner-staging  preview-8806  7 → 8    † confirmed 11:03
  created    staging  hetzner-staging  preview-8821  – → 1    server_type: cx22 · region: fsn1
  changed    staging  hetzner-staging  preview-8815  9 → 10   labels.retire-after: 2026-08-18 → 2026-08-25

  THE WORLD MOVED   2 observations
  CHANGE   TARGET  DEFINITION  RECORD            VERSION  FIELDS
  changed  local   uptime      status.hyper.dev  22 → 23  status: 200 → 503
  changed  local   uptime      cert.hyper.dev    22 → 23  days_left: 41 → 34

  THE CODE MOVED   3 facts
  DEFINITION           FACT                 FROM     TO
  retire-preview-envs  definition revision  a91f0c2  4d7e118
  retire-preview-envs  step retire · bound  3        5
  —                    repository revision  1f0a3d7  88bc402
  2 other lines changed · git diff 1f0a3d7 88bc402

  TOTALS  7 changes · 5 asset · 2 observation · 3 tombstone · the code moved
```

## The Refusal

A Refusal is the most verbose surface in the tool (ADR-0026). No flag, no confirmation and no override
reaches one (ADR-0001), so this rendering is the entire path back to a passing review.

It renders three things. The Step table says what each Step did, in the Disposition vocabulary §6 and
§12 fix, and how many Records each wrote — a Refusal withholds Records for the refused Step and for no
other, so the Steps before it wrote what they wrote and it is stated in words rather than left to
inference. The caret excerpt shows the offending line in its own context, which is what lets a reader
judge whether raising the Bound is the right fix at all or whether the selector is the thing that is
wrong. The `EDIT ONE OF` table says where to edit. Both remediations are rendered, and the narrowed
selector is speculatively re-expanded so its count is on the page — a read `hyper` did not have to
perform, worth it because the alternative is a reviewer widening a destroy Bound for want of the other
number.

Under those three sits the terminal line every Run ends with, in the form the next section states: on
this path it carries the remediation pointer, `show --expansion` being where the whole expansion is read
back rather than the count the caret gives.

```
$ hyper run retire-preview-envs

  STEP  ID      KIND     DISPOSITION  RECORDS
  1     probe   read     ran          12
  2     label   mutate   ran          8
  3     retire  destroy  refused      0

  refused: bound exceeded — nothing was deleted

    procedures/retire-preview-envs.yaml:30
     28 │           - field: created_at
     29 │             older_than: 14d
     30 │       bound: 5
        │              ^ expansion resolved 23 assets on staging
        │
        = checked at expansion, before the first call
        = no flag overrides this (ADR-0001) — the way past is an artefact edit

  EDIT ONE OF
  FILE                                 LINE  FIELD           FROM  TO
  procedures/retire-preview-envs.yaml  30    steps[2].bound  5     ≥ 23
  procedures/retire-preview-envs.yaml  29    steps[2].over   14d   30d   expands to 4

  steps 1-2 ran and their 20 records are written. step 3 wrote nothing.

  refused · exit 77 · hyper show 01991ea6-b118-7c93-8d41-6b2f7ae05c19 --expansion   for all 23
```

## The terminal line

Every Run's rendering ends with one line naming what happened, the exit code §9 fixes, and the entry
that holds it — on the completed path exactly as on the refused one. It is the human counterpart of the
`outcome` row below, and it exists because that row does: one renderer produces both forms (ADR-0026),
so what the row carries the line carries — a fact the stream states and the page does not is the two
surfaces disagreeing about what happened rather than differing in shape.

```
  completed · exit 0 · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19
  completed · dry-run · exit 0 · run 019921f4-3c07-70b2-a15e-2d9c4e83b661
```

A rehearsal carries the marker its entry carries (§7). A `--dry-run` that reached the end completed and
exited 0 like any other Run, so without it the line a Run that reached the world writes and the line a
rehearsal writes are the same bytes, on the one path where the difference is the whole point. It is
`dry_run`'s rule arriving on a surface: the reader that takes its absence for `false` is the reader that
pays.

The exit code is on the page rather than left to the shell, because the shell is not always there to
read it. The job summary §10 states is these same renderings relocated, and what lands there is stdout
and nothing else: a red job says a Run did not complete and never which of the four `failed` codes it
exited with, which is the whole reason that space is finer than the triple (§9).

A Refusal's line absorbs the remediation pointer rather than restating the id beneath it — the form
above, where the `show` command stands where a completed Run names its id — because a Refusal is the one
outcome with a next command to name, its rendering being the entire path back (ADR-0001). A completed
Run names none. What to look at next is the Comparison, and saying so here would editorialise on a
surface that reports, `FLAGS` above being the one place that does (ADR-0026).

**The Run id renders whole.** Every other id on every other human surface is abbreviated for the eye —
the Comparison's header, a `runs` row, a Provenance revision (§7) — and this is the one an operator
retypes, so it renders as the argument the next command takes rather than as a fact to recognise
(ADR-0047). What that leaves standing is stated rather than discovered: an id read out of a table cannot
be retyped, and what supplies one whole is `--json`, which abbreviates nothing.

Where no entry was written the id is absent, and its absence is the fact. Two paths decline before a Run
is identified at all — the version pin gate and the bootstrap `store-absent`, which renders, exits `77`,
and writes nothing (§9) — and on both the line says what happened and names nothing to look up, there
being nothing to look up. Every other Refusal has an id by the time it declines, `run.json` being written
at Run start (§7).

## The row stream

`--json` switches every surface above to NDJSON: one JSON object per line, each carrying a `type`
discriminator. Both forms come out of one renderer (ADR-0026), and the mapping is total — every row of
every table above is one object, a Comparison's header is one object, and nothing rendered is left out.
A long Run streams.

The last row is always the terminal row, and its absence means the stream was cut off. There are two,
and the type is itself the discriminator: a Run emits `outcome`, carrying the outcome, the exit code
(§9), and the `run_id` of the entry it wrote, plus the `error_code` of the check that declined it where
the outcome is `refused` — a `completed` or a `failed` Run carries none at all, a failure having no
check to name (§12). Everything else emits `result`, carrying the truncation marker. Both are always
emitted, `failed` and `refused` included.

`run_id` is written whole and is absent exactly where the line above names nothing: the two paths that
decline before a Run is identified. The row is emitted there regardless — `run` terminates with
`outcome` on every path it takes, a terminal type that flipped to `result` according to how early the
tool declined being one fact arriving under two contracts. `dry_run` rides beside them and is written
always, `false` included, which is §7's one exception to the absence rule holding on the wire for the
reason it holds in the Store: what a reader that takes its absence for `false` gets wrong is
unrecoverable. The line renders the marker only where it is true, an eye reading a line rather than
scanning it for an absent key.

Consumers filter rows rather than queries. There is no expression language over the stream and none
behind it (ADR-0013): `hyper changes --json | jq 'select(.type=="asset")'` is the shape of every
arbitrary predicate.

A review does not decompose into rows the way the three change tables do, so `review --json` emits the
annotations and never the source — the consumer already has the file. Each `flag` row carries the line
it cites, which makes a flag citing a line no `gutter` row marked a detectable violation rather than a
prose mistake.

```
$ hyper review procedures/retire-preview-envs.yaml --json
{"type":"gutter","line":3,"marker":"envelope ok"}
{"type":"gutter","line":5,"marker":"read local"}
{"type":"gutter","line":13,"marker":"mutate! staging"}
{"type":"gutter","line":21,"marker":"DESTROY staging"}
{"type":"gutter","line":30,"marker":"changed since a91f0c2"}
{"type":"authority","definition":"uptime","target":"local","definition_kinds":["read"],"target_kinds":["read"],"effective":["read"],"destroy_operations":[]}
{"type":"authority","definition":"hetzner-staging","target":"staging","definition_kinds":["mutate","destroy"],"target_kinds":["read","mutate","destroy"],"effective":["mutate","destroy"],"destroy_operations":["delete_server"]}
{"type":"flag","flag":"UNBOUNDED","cites_line":13,"step":"label"}
{"type":"flag","flag":"DESTROY","cites_line":21,"step":"retire"}
{"type":"flag","flag":"WIDENED","cites_line":30,"step":"retire"}
{"type":"flag","flag":"ENVELOPE","cites_line":3}
{"type":"result","truncated":false}
```

```
$ hyper changes --since 2026-08-04T09:12:00Z --json
{"type":"window","procedure":"retire-preview-envs","baseline":{"run":"01991c3a-7d40-7a11-9c2e-4f0b8d61a3e7","trigger":"cron","started":"2026-08-04T09:12:03Z","outcome":"completed","revision":"a91f0c2"},"subject":{"run":"01991ea6-b118-7c93-8d41-6b2f7ae05c19","trigger":"igor@thinkpad","started":"2026-08-06T11:03:18Z","outcome":"completed","revision":"4d7e118"}}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8801","from_version":4,"to_version":5,"confirmed_at":"2026-08-06T11:02:41Z"}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8802","from_version":3,"to_version":4,"confirmed_at":"2026-08-06T11:02:52Z"}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8806","from_version":7,"to_version":8,"confirmed_at":"2026-08-06T11:03:09Z"}
{"type":"asset","change":"created","target":"staging","definition":"hetzner-staging","name":"preview-8821","to_version":1,"fields":{"server_type":"cx22","region":"fsn1"}}
{"type":"asset","change":"changed","target":"staging","definition":"hetzner-staging","name":"preview-8815","from_version":9,"to_version":10,"fields":{"labels.retire-after":["2026-08-18","2026-08-25"]}}
{"type":"observation","change":"changed","target":"local","definition":"uptime","name":"status.hyper.dev","from_version":22,"to_version":23,"fields":{"status":[200,503]}}
{"type":"observation","change":"changed","target":"local","definition":"uptime","name":"cert.hyper.dev","from_version":22,"to_version":23,"fields":{"days_left":[41,34]}}
{"type":"code","definition":"retire-preview-envs","fact":"definition revision","from":"a91f0c2","to":"4d7e118"}
{"type":"code","definition":"retire-preview-envs","fact":"step retire · bound","from":3,"to":5}
{"type":"code","fact":"repository revision","from":"1f0a3d7","to":"88bc402"}
{"type":"code","fact":"other lines changed","count":2,"command":"git diff 1f0a3d7 88bc402"}
{"type":"result","truncated":false}
```

```
$ hyper run retire-preview-envs --json
{"type":"step","index":1,"id":"probe","kind":"read","disposition":"ran","records":12}
{"type":"step","index":2,"id":"label","kind":"mutate","disposition":"ran","records":8}
{"type":"step","index":3,"id":"retire","kind":"destroy","disposition":"refused","records":0}
{"type":"refusal","error_code":"bound-exceeded","step":3,"step_id":"retire","operation":"delete_server","target":"staging","declared":5,"observed":23,"artefact":"procedures/retire-preview-envs.yaml","line":30}
{"type":"remediation","artefact":"procedures/retire-preview-envs.yaml","line":30,"field":"steps[2].bound","from":5,"to":23}
{"type":"remediation","artefact":"procedures/retire-preview-envs.yaml","line":29,"field":"steps[2].over","hint":"narrow the selector","example_expansion":4}
{"type":"provenance","repo_revision":"88bc402","hyper_version":"1.4.0"}
{"type":"provenance","step":1,"definition_revision":"c3a17b0","manifest_digest":"sha256:2b7e…"}
{"type":"provenance","step":2,"definition_revision":"4d7e118","manifest_digest":"sha256:9c1f…","origin_digest":"sha256:e40a…"}
{"type":"provenance","step":3,"definition_revision":"4d7e118","manifest_digest":"sha256:9c1f…","origin_digest":"sha256:e40a…"}
{"type":"outcome","outcome":"refused","code":77,"error_code":"bound-exceeded","dry_run":false,"run_id":"01991ea6-b118-7c93-8d41-6b2f7ae05c19"}
```

Provenance splits on the wire exactly as it splits in the Store (§7): one `provenance` row carrying the
Run-wide members and one per Step file written, distinguished by `step` the way §7 distinguishes the two
files themselves. A member with no value at a level is absent rather than `null` — `origin_digest` above
on the Step whose Provider is locally authored — which is §7's absence rule and not a rendering of its
own. There is no row for a Step that was never reached, that Disposition writing no file to render.

## Redaction and the wire

A field a Manifest declares secret is held in the Store as a constant presence-only marker in the
position the value would occupy (§7), so every surface above renders the marker and can render nothing
else. The tables are structurally incapable of leaking a value, and equally incapable of reporting that
a secret rotated: the marker is a constant, identical bytes mint no version, and a rotation renders only
where a Manifest also projects the non-secret metadata that moves with it — a key id, a fingerprint, a
`created_at`.

The wire is visible only where no credential was used (ADR-0017). A Probe may surface the raw response
beside the projection `hyper` derived from it; against a credentialled Target nothing does, on any
surface, under any flag. What fills that gap is precision rather than volume: a projection that failed
names the path that failed to project (§6). That is the whole of what it carries — a failure names no
check, so no `error_code` stands beside it (§12) — and it is positional in the sense ADR-0007 is
positional rather than a scan of a body no surface may show. It is an error's rendering, so it goes to
stderr in both modes like any other (§9) rather than into the row stream above. That is the halting
Run's surface for it; the Step's Disposition holds the same path in the Journal (§7), and reading it
back later is `show`'s (§9) rather than an error's rendering at all.

## What these surfaces do not say

The Comparison prevents nothing. It is an accountability instrument and never a guardrail, and it
reports what changed rather than what is wrong (ADR-0010). It never says *this differs from what we
intended*, only *this differs from when we last looked*.

There is no rendering before a destructive Step. What stands in its place is the Definition review above
and the Bound (§5), both acting on the authored claim rather than on the world.

Three tables buy the actor split at the cost of a single chronological scan: there is no one list of
everything that happened, in order. Named here as the accepted cost it is, and carried forward to §13
with the limit above it.
