# §10 — Cadence and projection

A Procedure may declare a **Cadence**: how often it is meant to run. `hyper` keeps no clock of its own
and never runs it. It *projects* the declaration into an executor's clock — a workflow file it
generates, that a human reads in a diff, and that a static check holds to the declaration it came from
(ADR-0005).

This chapter states the grammar a Cadence is written in, what it promises and what it does not, the
gloss every surface renders it through, the workflow `project` generates, the check that fails when
that file no longer matches, and the job summary the projection writes.

## The declaration

A Cadence is authored on a Procedure, in `procedures/`, as one field of the artefact §3 states. A
Procedure declaring none is run by hand and generates no workflow; there is no default recurrence and
nothing inherits one.

One Procedure declares at most one Cadence. Two recurrences are two Procedures, with any shared body
factored into a nested one — which stays one Run (§6) — so *prod every five minutes, staging hourly* is
two artefacts rather than one artefact with two clocks. A Procedure invoked by another may declare a
Cadence of its own; it is projected like any other, and its invocation from elsewhere is still one Run.

A Cadence is a code fact: it appears in `THE CODE MOVED` when it moves between two Runs, on the same
footing as a Bound (§8, §12). It is a blast-radius multiplier, which is why it is declared in the
reviewed artefact and the executor's copy derived from it (ADR-0005).

### The grammar

A Cadence is a five-field POSIX cron expression, UTC only (ADR-0005). The fields are space-separated
and in one order — minute (`0`–`59`), hour (`0`–`23`), day of month (`1`–`31`), month (`1`–`12`), day
of week (`0`–`6`, Sunday `0`).

Each field is one of four **item** forms — `*`, a number, a range `a-b`, or a step over either, `*/n` or
`a-b/n` — or a comma-separated **list** of them. A list's members are items and not merely numbers, so
`9-11,14-16/2` is a field like any other: the restriction POSIX does not make is one this grammar has no
reason to invent, and it is the form the corpus teaches an AI author to write (§3). A field's items may
overlap, repeat, and arrive in any order; what a field means is the set of values its items select
together.

Nothing else is in the grammar. A sixth seconds field, an `@hourly`-style nickname, a month or
day *name*, `?`, `L`, `W`, `#`, and any timezone or offset are all rejected at load
(`cadence-malformed`, §12). A name is rejected on the ground the nickname is: it is a second spelling
of a Cadence the numeric form already states.

Where the day-of-month and day-of-week fields are both restricted, POSIX matches a day satisfying
*either*, and the gloss below says so in words. Restricted means *not `*`*, and it is the spelling that
decides rather than the values: `0-6` selects every day of the week and is still a restriction, so
`0 0 1 * 0-6` unions to every day of the month. The gloss says that too, in the same words.

There is nowhere to name another timezone — no field, no flag, and no file (ADR-0014). *3am my time* is
therefore unexpressible, which is stated here as the cost it is and carried forward to §13.

An expression finer than the executor's floor loads like any other. What it costs is legible in the
rate the gloss derives, on the review surface where every other blast-radius fact is read.

## What a Cadence promises

A Cadence is a lower bound on staleness rather than a promise of coverage (ADR-0005). It bounds how
long a fact may go unrefreshed; it says nothing about any particular window having been served.

A missed window is never made up. There is no catch-up, no backlog, and no queue of skipped
occurrences: re-invocation is decided by Repeatability against the Journal's evidence and never by a
clock (§6, ADR-0005).

Because that evidence is what decides, a Cadence and a **run-once** Step are refused together
(`cadence-run-once`, §4, ADR-0038). Run-once Refuses where the Journal already holds the Step as *ran*,
and a Refusal is terminal, so every occurrence after the first would stop there and carry the rest of
the Procedure down with it: the clock would be attached to a body with a lifespan of one occurrence.
The check walks every Procedure reachable from the one declaring the Cadence, to any depth, because the
paragraph above makes a nested Procedure the ordinary home of a shared body — which is exactly where a
run-once Step hides from the artefact carrying the clock. What is authored instead is the split that
rule already describes: the run-once Steps in a Procedure run by hand, the recurring ones in the
Procedure that keeps the Cadence.

The floor is the executor's — five minutes on Actions — and delivery is best-effort.

A Cadence is also the declaration the last Journal entry is read against, which is what makes staleness
readable at all. It is derived when something looks, and nothing watches for it.

## The gloss

Cron is write-only for humans and agents alike, so no surface renders a Cadence as the expression
alone. Wherever a Cadence renders, the gloss renders with it (ADR-0005, ADR-0021) — `0 3 * * 1` as:

```
03:00 UTC every Monday   ·   ≈4.3 runs/month   ·   last ran 41 days ago
```

The **phrase** states the times of day, the days, and the months the expression selects. The **rate**
states how often that comes to, in runs per month. Both are derived from the five fields and from
nothing else — no clock is read, no calendar of record is consulted, and neither summarises (ADR-0066).
Each is stated in full below.

### The phrase

The phrase is a **total function of the five fields**: every expression the grammar above admits gets
one, and an expression that glosses awkwardly gets an awkward phrase rather than a fallback to
something shorter. There is no phrasebook of recognised shapes, because a phrasebook stops being total
exactly where the expression is hardest to read, which is where the gloss was needed. Nothing is ever
truncated, on either of the surfaces that render two phrases in a table cell: a truncated gloss drops,
and §8's truncation marker names *an axis dropped*, which has no meaning inside a phrase.

It is composed of three clauses in fixed order — the **time**, the **day**, the **month** — so the eye
lands in the same place on every rendering. An unrestricted field renders nothing, with one exception
stated below. `UTC` attaches once, at the first clause carrying a clock or a calendar value; `* * * * *`
is the only expression in the grammar that carries neither and therefore names no timezone. Naming one
there would qualify a statement that is true in every timezone, which is a word added.

**Each field renders in the form it was written in.** The phrase preserves the expression's own
structure rather than recomputing a canonical one, which bounds its length by the expression's own
length and lets a reader map every clause back to the field that produced it — which is what makes
ADR-0063's *a reader who disagrees with a gloss can check it* mechanically true rather than merely
plausible.

| the field | the phrase |
| --- | --- |
| `*` | *every* |
| a number | the value |
| a list | its members, ascending, duplicates collapsed |
| `a-b` | *from a to b* |
| `*/n` | *every n* on the minute and hour fields where `n` divides 60 or 24; its members everywhere else |
| `a-b/n` | its members, always |

Order and repetition are the only things normalised, because cron attaches no meaning to either: a
field is the set of values its items select, so rendering `5,1,3` ascending drops nothing a reader could
map back. The **form** is never normalised. `0-59` stays a range and glosses as one, awkwardly, rather
than collapsing into *every minute* — somebody wrote a range where `*` was meant, and that is a fact
about the artefact on the surface built to show facts about the artefact. A form selecting exactly one
value is repaired **grammatically** and no further: `*/1` renders *every minute*, `1-1` renders `:01`.

**A step is not an interval, and the phrase says *every n* only where it is exactly true** (ADR-0066).
`*/7` on minutes selects 0, 7, 14, 21, 28, 35, 42, 49 and 56, and then waits **four** minutes — so
*every 7 minutes* is a false sentence rather than a second notation, and it would be the first thing a
gloss ever introduced. Interval language is therefore admitted on the minute and hour fields alone,
where the span is fixed and its minimum is zero, so the grid reads unambiguously off the phrase. It is
never admitted on day of month, whose span varies with the month — `*/10` is the 1st, 11th, 21st and
31st, and most months have no 31st. It is never admitted on month or day of week either, and there by
the same rule paying off rather than by exception: *every 3 months* does not say **which** three, where
naming January, April, July and October is complete and shorter both.

**The time clause merges the minute and hour fields into clock times only where the merge does not
lengthen it** — that is, where the minute selects one value and the hour enumerates, a single value or a
list. `0 3 * * 1` is `03:00 UTC every Monday` and `0 9,17 * * *` is `09:00 and 17:00 UTC every day`; the
merged form has one member per hour and the hour clause had that many already. A range, a step or `*` on
the hour keeps the two fields in separate clauses — `0 9-17 * * *` is `at :00 past every hour from 09:00
to 17:00 UTC` — and so does a minute field selecting more than one value. Merging further would take the
cross product, and `*/5 9-17 * * *` is 108 clock times, which is the one way a structure-preserving
phrase could still explode.

**The day clause is the one unrestricted field that speaks.** Where both day fields are `*` it renders
*every day*, but only where the time clause names clock times; where the time clause already recurs
within the day, the recurrence is stated and *every day* would restate it. So `0 3 * * *` is `03:00 UTC
every day` and `*/15 * * * *` is `every 15 minutes`. Without that exception the commonest expression in
the grammar would render `03:00 UTC` and read as something that happens once. A day-of-month clause says
*of the month* only where the month clause is silent, the months it names doing that work otherwise.

**Where both day fields are restricted the clause is a disjunction, and it is written `or`**, with both
sides in full: `0 0 1 * 1` is `00:00 UTC on the 1st of the month or any Monday`. Never a comma and never
*and* — *and* states the intersection, which is the wrong answer and the one that reads like the right
one. This is the single most misread thing in cron, and it is the reason the phrase spends the words:
that expression fires twelve times a year **plus** fifty-two, which the rate beside it then says out
loud.

**The vocabulary is full English names and ordinals** — `Monday`, `January`, `the 1st`, `the 22nd` —
fixed, with no locale and nothing to configure (ADR-0014). They are exactly the words the grammar above
refuses to read on input, and that is the point rather than a tension: a name is a second spelling of a
number an artefact must not be written in, and a gloss is the reading that number has. Abbreviating them
would also make the phrase look like something that could be pasted back in.

```
0 3 * * 1          03:00 UTC every Monday
0 0 1 * *          00:00 UTC on the 1st of the month
0 0 * * *          00:00 UTC every day
*/5 * * * *        every 5 minutes
*/7 * * * *        at :00, :07, :14, :21, :28, :35, :42, :49 and :56 past every hour
0-59 * * * *       every minute from :00 to :59 past every hour
0 9,17 * * 1-5     09:00 and 17:00 UTC every day from Monday to Friday
0 9-17 * * *       at :00 past every hour from 09:00 to 17:00 UTC, every day
0 0 1 * 1          00:00 UTC on the 1st of the month or any Monday
0 0 1 */3 *        00:00 UTC on the 1st in January, April, July and October
0 0 29 2 *         00:00 UTC on the 29th of February
```

### The rate

The rate is the expression's matches over one full **Gregorian cycle**, divided by 4,800. The cycle is
400 years: 146,097 days, 4,800 months, and exactly 20,871 weeks — so it repeats the leap pattern and
the weekday alignment together, and every one of both appears in it in its true proportion.

That is what makes the denominator **derived rather than named** (ADR-0066). A calendar year is the
reading the arithmetic invites and it has no defensible value: a leap year and a common year disagree,
and so do two common years starting on different weekdays, so `0 3 * * 1` is 4.33 or 4.42 depending on
which one somebody picked. Counting over *this* year is worse than picking one, because the number then
changes on 1 January with no edit anywhere, and a laptop and a runner reading the same artefact across
that boundary would render different rates for the same Cadence. Over the cycle there is nothing to
pick and no clock to read: `0 3 * * 1` is 20,871 ÷ 4,800 = 4.348, forever and in both environments.

It renders to **two significant figures**, with the unit fixed at runs per month on every surface — a
unit that varied with the value would destroy the comparison the rate exists for. Two figures is what
the range needs: one decimal place would render the rarest expression the grammar admits, `0 0 29 2 *`
at 0.0202, as `0.0 runs/month` beside a Procedure that does run, which is a gloss dropping the fact it
was rendered to carry.

The `≈` renders where the number was rounded and is **absent where it is exact**. `0 0 1 * *` is
`1 run/month`, exactly, and saying so is free; `0 0 * * *` is `≈30 runs/month`. The singular is used at
exactly 1 and nowhere else.

**The wire carries the number the page renders**, rounded once — on the `artefact` row's `rate` and on a
`code` row's `from_rate` and `to_rate` alike (§8). Carrying the unrounded value beside the rendered one
would be two representations of one derived fact that can disagree, which is the refusal this
specification makes for stored durations (§7), for output schemas (§3) and for a Head marker (§7); and
it buys nothing, two significant figures being far more than the judgement the rate is read for needs.

The rate is the number that matters beside a Procedure that destroys, which is why it is rendered rather
than left to be inferred from the phrase. It is also what makes ADR-0005's argument legible on the
screen: `0 0 1 * *` → `*/5 * * * *` renders `1 run/month` against `≈8800 runs/month`, and the ≈8,800×
is read off two numbers rather than reconstructed from ten cron fields.

### Beside them

Beside them renders the **last Journal entry** — the most recent Run of that Procedure, dry-run entries
filtered out like every other reading of Journal evidence (§7). It is a fact placed next to the gloss
rather than a third member of it (ADR-0021), which is why the two part company on the surfaces below.
What an overdue reading is made of is that entry, its Trigger (§7), and its age against the declared
Cadence — facts placed beside each other, with the human doing the subtraction.

`hyper` never says *overdue*, and nothing refuses on it. Any threshold would be a claim of the tool's
own on a surface built to admit none (ADR-0021, ADR-0026), and being overdue is a fact about the
executor rather than about the artefact or the world.

### Where it renders

**Wherever a Cadence renders, the gloss renders with it, and there is no surface exempt.** Cron is
write-only wherever it is read, so the rule is total rather than a property of the review screen
(ADR-0063). Four surfaces render one today:

- **A review's header**, on the line beneath the path and the range, with the last Journal entry beside
  it (§8). The gloss is a fact about the artefact as a whole, which is the header's own subject, and it
  is not a gutter mark: a Cadence governs every Step rather than making a claim about its own line.
- **A review's `FLAGS` row**, where the Cadence moved inside the range — both expressions glossed,
  which is the only way ADR-0005's blast-radius argument survives on the screen it was written for. A
  gloss is a notation and not a claim, so ADR-0026's one-editorial-surface rule is untouched (§8).
- **`THE CODE MOVED`'s `cadence` row**, both expressions again, in a Comparison (§8).
- **`project`'s rows**, one per Procedure, where the agent that just wrote the workflow reads back what
  it projected (§9).

The last Journal entry renders on the first of those and on none of the others. It is a fact about what
stands now, and the three below are about a value moving between two revisions or about a file just
written — a surface with no artefact-under-review has no side to hang it on.

**How the three parts are arranged is the surface's, and what they are is not.** On a header they share
one line, separated by `·`, the expression already being on the `cadence:` line below. In a table cell
the expression, the phrase and the rate **stack**, the gloss under the cron it glosses, and the cell
wraps rather than shortening anything (§8). The rate never becomes a row or a column of its own: it
cannot move while the Cadence stands still, so a row for it would be one fact rendered twice.

**A review's absent entry is one absence, not two.** The range and *last ran* read the same entry under
the same filter, so §8 names it once, on the range's line, and the gloss line carries the phrase and the
rate — which need no Store and render offline as they always have. `review` reads the Journal where
there is one and never requires one, which is what keeps the offline authoring loop intact (§9).

### Two facts the gloss carries

Two facts about how the executor will treat the declaration render beside the three above, wherever the
gloss renders. Neither is a problem with the artefact: neither carries an `error_code`, neither fails
`check`, and neither is a claim `hyper` makes about what the world holds.

**Scheduled workflows run only on the default branch.** A Cadence on a feature branch is inert, and
saying so where the Cadence renders is what keeps it from being discovered three weeks later.

**An hour-boundary collision.** Actions names the start of every hour as its high-load window, so a
Cadence landing there is the one most likely to be delayed or dropped. `project` never adjusts a
declared time to avoid it: a generated time nobody wrote would be a small lie in a file whose whole
value is that it says what was declared.

## The projection

`project` generates one workflow file per Procedure that declares a Cadence, at
`.github/workflows/hyper-<procedure>.yml`, carrying the Procedure's name verbatim — one file per
Procedure, so run history in the Actions UI is per-Procedure and nothing is ambiguous about which
expression fired. Generation being whole-file, repo-wide and all-or-nothing (§9), a Procedure that has
dropped its Cadence loses its file to the same command that writes the rest.

### The generated workflow

```yaml
# generated by hyper 0.4.1 — edits are overwritten by `hyper project`
name: retire-preview-envs

on:
  schedule:
    - cron: '0 3 * * 1'

permissions:
  contents: write

concurrency:
  group: hyper-store
  cancel-in-progress: false

jobs:
  run:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
        with:
          persist-credentials: true

      - name: deepen the checkout
        run: |
          if [ -f .git/shallow ]; then git fetch --unshallow; fi

      - name: install hyper 0.4.1
        run: |
          curl -fsSL -o hyper.tar.gz \
            https://github.com/TheLoomLabs/hyper/releases/download/v0.4.1/hyper-0.4.1-x86_64-linux.tar.gz
          echo 'a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2  hyper.tar.gz' \
            | sha256sum -c -
          tar -xzf hyper.tar.gz

      - name: hyper run retire-preview-envs
        env:
          STAGING_TOKEN: ${{ secrets.STAGING_TOKEN }}
        run: |
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          set +e
          ./hyper run retire-preview-envs | tee -a "$GITHUB_STEP_SUMMARY"
          code=${PIPESTATUS[0]}
          set -e
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          exit $code

      - name: hyper changes retire-preview-envs
        if: always()
        run: |
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
          ./hyper changes retire-preview-envs | tee -a "$GITHUB_STEP_SUMMARY"
          printf '```\n' >> "$GITHUB_STEP_SUMMARY"
```

### What the projection carries

**The workflow and its job are named after the Procedure**, because that string is the subject line of
the executor's own failure email (ADR-0005) and the only part of a failure visible on a phone. GitHub
addresses that email to whoever last committed the projection, which is whoever reviewed the Cadence
(ADR-0021).

**`on:` carries the recurrence and nothing else.** The reviewed artefact declares a recurrence and
nothing more, so the projection derives no second occasion for a Run to start from; a Run started by a
person is started from a laptop and records that Trigger (§7).

**`permissions: contents: write`** is what a runner needs in order to write the Store, and the whole of
what the projection grants. Writing the Store is not an Operation and costs no Capability (§7,
ADR-0006).

**The `concurrency` group is the Store's**, one group for the repository rather than one per Procedure,
with `cancel-in-progress: false`. It is what stands in for the single-store lock §6 states across two
runners that share no filesystem (ADR-0005). A second effectful workflow firing in the same minute
queues behind the first, and where a third piles up the oldest pending run is dropped — which the lower
bound above already accepts. A Procedure whose every Step is `read` carries no group at all: it takes
the shared lock and reaps nothing, and serialising it would starve the five-minute cadence behind the
forty-minute provision (§6, ADR-0006).

**The `env:` block is derived from the bindings the Procedure makes.** Every credential slot required by
a (Definition, Target) pair reachable through the Procedure, to any depth, appears on the `run` step
and nowhere else in the file, each carrying an executor secret named exactly as the environment
variable that Target declaration resolves the slot from (ADR-0007). It is the bindings rather than the
declarations because a Target may carry slots for a scheme this Procedure never uses (§3), and writing
those into the job would put a secret in the runner that no Step could reach — the same narrowing §6
makes for the presence check, for the same reason. The
runtime binary resolves an environment variable exactly as it does on a laptop, so nothing about the
executor enters the decision; adding a Target to a Procedure makes a new secret appear in the diff
`project` writes rather than in YAML nobody reviewed; and an executor holding no secret of that name
Refuses before the first Step (`credential-absent`, §12) rather than failing part-way through one.

**The checkout leaves the token behind**, which is what `hyper` fetches and pushes the Store branch with
— that sync is `hyper`'s own work rather than a step of the workflow (§7). `persist-credentials: true`
is written out rather than left to the action's default, a byte-exact file being no place to rest
silently on a default somebody else may change (§11). `actions/checkout` is the one action the
projection names, pinned by commit SHA, and `runs-on` names one pinned platform: both are compiled into
the binary rather than authored anywhere, and §11 states that whole set and what follows from it
(ADR-0046).

**The deepen step is what makes the Comparison legible on a runner**, and it is there because
`actions/checkout` defaults to one commit. `hyper changes` reads bytes at the baseline Run's revisions,
and a blob id is content-addressed, so on a shallow clone the object is present exactly where the
artefact did not move and absent exactly where it did: five of the nine code classes and the whole line
count would go unread on every window that had something in it, and a Target declaration — which carries
no Provenance member at all — would move without a single row saying so (§8,
[ADR-0071](../adr/0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md)).

**It deepens the code branch and not the Store.** `fetch-depth: 0` is the obvious spelling and it is
*all history for all branches and tags* by the action's own documentation, which fetches the Store
branch — the one branch §13 names as growing without bound — on every scheduled Run of every Procedure.
`checkout` leaves `remote.origin.fetch` pinned to the single ref it checked out, so an `--unshallow`
after it inherits that refspec and reaches nothing else. That is written down rather than relied on.

**What the Store gets instead is `hyper`'s own depth-1 fetch, naming the ref**, the checkout having left
neither the branch nor a refspec that reaches it (§7,
[ADR-0074](../adr/0074-the-store-branch-is-fetched-shallow-and-whole.md)). It lands after this step, so
the `.git/shallow` this step just cleared comes back with a boundary naming the Store branch alone: the
code branch stays whole, and nothing later in the file reads that path. The two depths are opposite
because the two branches are read differently — the code branch is read *at revisions the Store names*,
and the Store is read at its tip and nowhere else.

**The guard is `.git/shallow` and there is no `|| true`.** `--unshallow` errors on a repository that is
already complete, which a self-hosted or pre-warmed runner may hand it, so the test is what keeps the
step total. Its failure fails the step, and the step sits before the `run` invocation: nothing has
reached the world when it dies, so the cost is one skipped window against a Comparison that cannot see a
credential source move. §8's `not-in-clone` absence stays as the safety net beneath this, and is not the
path a projected workflow is expected to take.

**The install step carries the version, the URL, and the digest** of the artefact for the platform
`runs-on` names — the pin and its verification in one reviewed file, with nothing resolved at run time
(§11, ADR-0020). The URL is a template the binary holds, with the version its only variable. The binary
the runner fetches compares itself against the same pin before it reads a second file (§9), so a
workflow left behind by an older projection Refuses rather than acting.

**Two `hyper` invocations, and no branching.** `run` first, then `changes` under `if: always()`. One
invocation would put the Step narration on the summary page and leave the Comparison — half of what
there is to see — unrendered. A Refusal returns the full Refusal rendering on stdout and exits `77`
(§8, §9), so the first invocation already puts the remediation surface on the page and no conditional
is needed to get it there (ADR-0021).

**`${PIPESTATUS[0]}` is what fails the job.** The status the step exits with is the `hyper`
invocation's rather than `tee`'s, which is what makes a Refusal, a failure, and Store contention land
as a red job carrying the code §12 fixes. The `set +e` around it exists so the closing fence is written
before the step exits.

The runtime binary is told nothing about any of this. Only `project` knows the executor, which is what
keeps `hyper run <procedure>` producing the same bytes on a laptop as on a runner (ADR-0021).

## The check

A generated file that is not what `project` would write now fails `check` (`projection-stale`, §12) —
the verification half of generate-and-verify, and the reason the declaration stays the source rather
than becoming a comment on the executor's copy (ADR-0005). The comparison is whole-file and byte-exact
against a fresh regeneration, so it catches every way the two can part: a Cadence edited and not
projected, a hand-edit to a generated file, a generated file deleted, one left behind by a Procedure
that no longer declares a Cadence, and a hand-edited version pin — which is therefore caught twice,
here and by the fetched binary's own pin gate (ADR-0020).

Byte-exactness costs nothing extra for the constants §11 states. One of those moves only when the
binary does, a binary that differs is a version that differs, and the file already carries the version
in four places — so the check fires on no occasion the pin was not firing on anyway, and what a
compiled-in constant changes is what the diff says rather than how often there is one (ADR-0046).

It is one of `check`'s rules and runs with the rest of them, standalone and in the pre-flight of every
Run alike (§4, §9); it is stated here because it compares a reviewed artefact against a file derived
from it, and the derivation is stated here. Regeneration reaches no network — a version change is the
only thing that ever does (ADR-0020) — so the check runs wherever `check` runs.

What repairs it is `project`, always, and never a flag on the check itself (§9). A hand-edit to a
generated file does not survive the next `project`, which is correct rather than regrettable: it is
authority living outside every reviewed artefact.

## The job summary

The projection writes the ordinary renderings to `$GITHUB_STEP_SUMMARY`, fenced rather than converted
to Markdown, so the review gutter and the aligned columns survive and the page carries what the
terminal carries (§9, ADR-0021).

What lands there is the Step table `run` writes — one row per Step carrying its index, its id, its
Kind, the count of Records it concluded about (§8, ADR-0030), and its **Disposition**, one of the seven
§12 defines. Where a guardrail declined, §8's Refusal rendering follows in full, remediation table
included. Then the
Comparison, under `if: always()`, so a Run that failed still renders what reached the world before it
stopped.

The Dispositions are why the page is worth writing. A green check means the Run finished, not that
anything happened: a Run whose every Step was skipped as already recorded completes and exits `0` like
any other (§6), no exit code distinguishes it, and this page is where the difference is legible — as it
is where *ran* three times and *never reached* twice reads as the partial Run it was. The counts beside
them say how much each Disposition covered, a Step that skipped five hundred Assets rendering five
hundred rather than nothing (§8).

`tee` rather than a redirect, because a summary over the executor's per-step limit is dropped *without
failing the step*: the Actions log is always complete and the summary is a convenience copy of it
(ADR-0021).

## No clock of its own

`hyper` never sleeps, never daemonises, and never watches a clock. There is no scheduler in the binary,
no timer, no background thread, no `serve`, and nothing that outlives the invocation that started it
(§9).

Scheduled recurrence is a property of the executor. The laptop authors, runs ad hoc, and reviews; the
runner is the only thing that keeps time. Nothing is lost to that division — whatever the laptop holds
credentials for it can run by hand, and that Run is a first-class Run recording its own Trigger (§7).

A scheduled Run may be of any Kind. Unattended destruction is what the two keys, the mandatory Bound on
`destroy`, named-Operation granularity, and the absence of any bypass exist to make safe (§4, §5,
ADR-0001), and reserving the clock for `read` Procedures would undo all of it.

## What a Cadence does not buy

**Continuous monitoring is not what this is.** `hyper` claims periodic checking against a floor the
executor sets, with best-effort delivery and no made-up window. A sub-minute prober is a hosted service
rather than a stateless binary an external clock invokes. Named here as a non-goal and carried forward
to §13.

**Alert-on-transition does not exist.** Nothing watches a Record for a value crossing a line and
nothing emits when one does. The Comparison renders `status: 200 → 503` when someone reads it (§8), and
the notification you want is a Step you author — a Definition against a Slack or PagerDuty Target,
reviewed, Bounded and recorded like everything else (ADR-0021). It can announce that something
happened; it can never announce that something failed to happen. Carried forward to §13.

**Two executor failures `hyper` can neither see nor prevent.** A scheduled workflow auto-disabled after
60 days of repository inactivity produces no run, no error in the Actions tab, and no banner — only
GitHub's own warning email, addressed to whoever last enabled it, with the gloss's *last ran* the only
thing that shows it, and only when somebody looks. And an oversized job summary is dropped silently,
which is what the `tee` above exists to survive. A tool that only ever renders cannot see a run that did
not happen; both are stated rather than mitigated, and carried forward to §13.
