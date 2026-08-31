# The rehearsal marker is the entry's, and `records` joins for it

**`records` carries `dry_run` on every row, read off the Journal entry of the Run that wrote the
version.** It is written always, the bare `false` included — §7's one exception to the absence rule,
carried onto the surface that names the Run — and it is absent only where the branch holds no entry
for that Run at all, which the narration counts. On the page it is a `REHEARSAL` column carrying the
word `yes` where there is something to say and nothing where there is not, the reading `show`'s own
header already takes over the same marker.

**No Record version carries the marker, and §7 now says so.** The entry/version line has one side for
this fact, and it is the entry's.

## What was wrong (issue #234)

§6 and §7 are each explicit and they point opposite ways at one `run_id`.

§6: *a dry-run performs the reads it reaches and stops rather than simulating an effect (ADR-0010).
Those reads really happened, so they record Observations like any other.*

§7, of the entry: *that entry is evidence that a rehearsal happened and evidence of nothing else*, and
every consumer of Journal evidence filters it out.

Both are right. A reader who carries the second rule to the Record store — a different store, where
the first rule holds — discards exactly the versions holding the account they came for, and the field
that says which rule applies was written on one side only.

The evidence is the sealed acceptance run of 2026-08-30 on `fleet-rollout` (ADR-0110, issues #223 and
#232). The session rehearsed the rollout before running it, and **all four** pre-state Observations —
the whole of the task's first answer, *what each machine was on* — exist only under the rehearsal's
`run_id`, because the effecting Run's identical reads returned what the head already held and minted
no version. The session read `records`, could not tell what kind of Run `01a0517c` was, spent a `run
show` on it, and reported the join to the human unprompted:

> **The pre-state Records are attributed to the rehearsal**, run `01a0517c`, not the effecting run.
> […] it's just that the `run_id` on "web-02 was 1.3.9" is the rehearsal's, which is worth knowing
> before you read the Journal.

It cost a call in a twenty-seven-call run, and it costs one every time a reader meets an unfamiliar
`run_id` on a Record they care about.

## The row, and not the version file

The ticket left the shape open and named two candidates: a `dry_run` member on the `records` row, or
`dry_run` on the Record version file too, on §7's own ground — *a version file saying only `see Run
abc` would be unreadable in a browser and in a diff, which is exactly where this field set is read.*

That argument reaches this marker. A Record version sits under a Record path with no entry beside it,
and §7's own restatement rule — *a file restates what a reader cannot find in the same directory* —
would put the marker on the file. **What does not reach it is the recovery.**

Provenance has been on every Record version ever written. A marker added to the shape now answers only
the versions written after it, and every version already on a branch stays silent — so `records` must
read the entry to answer for those, and having read it, has the answer for the rest. A member on the
version file would then be a **second place to read one fact from, answering a subset of what the
first answers**, and the surface would need a rule for which to believe. That is the second
representation the Provenance split is written to avoid (ADR-0043), arriving on the marker §7 has
already singled out as the one a reader must never get wrong.

The cost of the file half is stated rather than waved at: it is `RecordSchemaVersion` moving to 2, the
read-down code ADR-0028 requires forever after, and a three-state marker in the Store's own type for
the versions written before it. What it buys over the row is a version file that reads whole in a
browser — real, and not worth those three when the row it is read through answers already.

**So the marker stays where a Run's own facts are written, and the join moves inside the call.** That
is the whole of the change a caller sees: the `run show` the session made by hand is now made by
`records`, once per listing rather than once per unfamiliar id.

## The third door on the Store

`records` reads the Journal through `store.Rehearsals`, which is narrow on both axes it can be narrow
on: **the Runs it is asked about**, and **one file under each** — the `run.json`, and never the
`outcome.json`, the closing writes or the Step files.

It is a third door beside `Entries` and `Listing` for their own stated reason: *which door a caller
needs is a cost*.

**It takes the Runs rather than answering the Journal**, and that is what keeps a `--limit 1` costing
what one answer costs. `records` cuts before it asks, so a listing of one Record opens one entry where
a walk of the whole Journal would make the narrowest question on this surface pay for a year of Runs.
It is the trade `Listing` already makes with its predicate: the branch is listed once either way, and
what a narrowing buys is files not opened.

**It opens one file under each**, which decides a failure mode. A Journal entry whose `outcome.json`
will not decode stops `Entries`, and there is no reason it should stop a command whose job is finding
a version: the kind of Run that wrote one is in the `run.json`, and a Run whose *end* nobody can read
is still a Run whose kind is known.

**Narrower is not laxer.** An entry this door does open answers the two rules every reader of a
`run.json` holds one to — the entry carries one, and it sits under the UTC date of the instant that
file itself carries — or the read faults, exactly as `decodeEntry` faults on the same two shapes. Both
rules are now written once and called from both doors, so what an entry *is* cannot come to be spelled
two ways. Answering *no marker* for a corrupt entry would spell a broken Store the same way as one
that simply never held that Run, which is the distinction the map below exists to keep.

That fault is not the refusal this decision declines below. A version naming a Run **the branch does
not hold** is reported and rendered; a `run.json` this command **read and could not** is a Store
`hyper` did not write, and `records` already ends the same way over a Record version in that state —
the record namespace it has always read is decoded on every listing, and one file that will not
decode ends the call. Extending that to the file it now reads is the same rule, not a second one.

It answers a **map and never a set**, because absence has to mean the other thing. A missing key says
*the branch holds no entry for this Run*, which is a Store that has lost evidence rather than a Run
that was not a rehearsal — and the one rule §7 states about this marker is that a reader taking its
absence for `false` gets a permanent wrong answer. A set would spell those two the same way.

## Three states on the wire, two on the page

The row carries `true`, `false`, or nothing. The third is not a rehearsal marker at all: a Run writes
its entry at Run start, before any Step, and Compaction removes no Journal entry (§7), so a version
naming a Run with no entry is a Store missing evidence. The count is narrated — *`n` versions name a
Run the Journal holds no entry for; REHEARSAL is read off the entries it does* — which is the reading
this command already takes over the same shape of absence, where a Definition that did not load leaves
`ORPHANED` unanswerable and the count is stated rather than the column quietly wrong (ADR-0069).

The page collapses the three to two, and the collapse is the same one `show` makes: the word where
there is something to say, a blank where there is not. A column carrying `no` down every row of an
ordinary listing is a column a reader stops seeing, and the wire — where a consumer reads the member
rather than scans it — is where the `false` has to survive.

## What was considered

**`dry_run` on the Record version file, alone or as well.** Refused above. Alone it cannot answer for
a version already written; as well, it is a second source for one fact that answers less than the
first, bought with the Store's first schema-version bump.

**Deriving the marker from Provenance already on the version.** There is nothing there to derive it
from. Provenance is which code performed the Run, and a rehearsal and the Run that follows it run the
same binary against the same revisions — that is the point of a rehearsal.

**Putting the marker on `runs` as well.** Out of scope here and left as it stands. `runs` orders the
Journal and `show` reads one entry whole; the ticket is about the surface that names a Run it did not
range over, which is `records`.

**Refusing a listing whose Records name Runs the Journal does not hold.** Refused. `records` writes
nothing and exits `0` whatever the Records it listed hold (§9); a Store missing an entry is a fact
about that Store, and reporting it is this command's job where declining to answer is not. It is a
different case from an entry that is *there* and will not read, which ends the call above.

**Answering the whole Journal and letting the surface pick.** Refused. It is the same answer for every
call, so a `records --name x` and a `records` of four thousand Records would read the same year of
entries — and the cut this command already applies is exactly the bound that makes the read
proportional to what it is about to render.

## Consequences

- **`records` reads the Journal, and it is the second thing it reads that is not the record.** The
  first is the repository, for `ORPHANED`. Both are columns the record cannot answer, and the Journal
  is on the branch the command already has open — one narrow listing, not a new input.
- **`store.Rehearsals` is a third door on the Journal**, and the first reader in the tool that opens a
  `run.json` without opening the entry around it. It reads one file per Run the answer names, so what
  a listing pays for this column is proportional to the listing rather than to the Journal.
- **The two rules a `run.json` is held to are written once** — `missingRunFile` and `misfiledEntry` —
  and `decodeEntry` now calls them too. Two readers of one file were two places for one message to
  drift.
- **Every `records` fixture store now holds a Journal.** The corpus was records-only because the
  command was; a store of Record versions written by Runs nothing recorded is not a Store `hyper`
  writes, and the goldens said so by rendering nothing in the new column.
- **The `records` page has a twelfth column and the wire a member after `step`.** The member sits
  beside the Run it describes and before `record_kind`, which is §9's own order.
- **The page spells two of the three states and the third is on stderr.** A blank cell covers an
  ordinary Run and a Run with no entry alike, and only the narration tells them apart — so `hyper
  records > out` keeps the ambiguity and loses the count. That is the cost of the `show` reading
  rather than an oversight, it is the same cost `ORPHANED` already pays over a Definition that did not
  load, and the wire is where the three states are kept apart for a consumer that must not confuse
  them.
- **`a-run-the-journal-does-not-hold` is the case for the third state**: two versions of one Record,
  one Run recorded and one not, one `yes`, one blank, and the narration counting the blank that is not
  an answer.
- **ADR-0110's first cost closes.** That transcript paid twice; this is the half about `records`.
  Issue #235 is the other, and it is not this decision.
