# §8 — Review and Comparison

Two surfaces carry the half of the thesis that says nothing changes unseen: a Definition review, before
anything runs, and a Comparison, after. This chapter states both, together with the Step table every Run
renders, the Refusal rendering that is the whole path back from a declined Step, the line every Run ends
with, and the row stream all of them emit under `--json`.
The renderings below are what `hyper` produces rather than an illustration of it; the commands that
produce them and the exit codes they carry are §9's.

None of it is a guardrail; every check standing between an authored artefact and the world is static
and sits before the Run (§4, §5). What that costs is stated at the end of this chapter (ADR-0010).

Colour and width are the only differences between an interactive rendering and one in a CI log. Every
fact on every surface is legible with colour off and through a pipe; colour re-emphasises what a marker
or a word already says and never carries a fact of its own (ADR-0015).

## The Definition review

The object under review is the artefact itself, annotated in place by a gutter, with a table for what is
assembled from elsewhere and one editorial surface (ADR-0026), under a header naming what is being read.
Four renderings sit inside one screen, and each has its own rule. ADR-0026 fixes three disciplines —
annotate in place, aggregate what is assembled from elsewhere, editorialise only by citation — and how
many surfaces instantiate one is a separate question (ADR-0063). Three of the four read on all five
reviewed artefacts and `AUTHORITY` reads on three of them; each rendering below states its own roster,
and there is no artefact on which a rendering is silently withheld.

**The header states facts about the artefact as a whole**, each read from one supply, and it cites no
line. That last clause is what separates it from `FLAGS` below: a fact pointing at a line is an index
into the gutter and belongs to the surface built for one, and a header permitted to point would be the
editorial voice ADR-0026 removed, arriving at the top of the screen instead of the bottom. It ranks
nothing and introduces no claim of its own.

Its members are the artefact's kind, its path, the range the review is read across, and — on a Procedure
declaring a Cadence — the gloss §10 states and the last Journal entry beside it. The rule is what admits
a further member and the list is what keeps the set checkable against a rendering; an intensional rule
nobody can enumerate is a set that has stopped being closed (§12). A whole-artefact fact no supply on
this screen already holds is not a header member for want of a home.

**A member renders from its own supply, and goes silent where another member's supply has answered for
it on the same line** ([ADR-0068](../adr/0068-one-supply-is-stated-once-and-the-member-it-silences-is-not-omitted.md)).
That is not the list going optional, and the distinction is the whole of what keeps omission the defect
it is everywhere else on this screen (ADR-0026): an omitted member is one with a supply of its own,
dropped; a silenced member is one the line already answered, and the header states a supply once rather
than twice. Two instances are below, and both are one artefact's absence serving two readings.

**It is a block, one fact per line, inside the same rule.** The kind heads the marker column below, the
path and the range share the first line, and the gloss takes the second. Composed onto the first line it
would make the screen's width depend on how awkwardly an expression glosses, and the grammar §10 states
admits expressions that gloss at very different lengths. Below the rule it would sit directly above
`kind: procedure` and read as an annotation of that line, which is the one thing the header may not be.

**A Cadence is a header fact and not a gutter mark**, though it is a blast-radius multiplier like the
Bounds the gutter carries (ADR-0005). The gutter annotates a line's claim about itself and a Cadence
governs every Step in the artefact; and the gloss reaches a supply the gutter does not have, the last
Journal entry being no part of the artefact across the range (ADR-0057). Its line is source like every
other and renders in place, and the change column marks it when the range touched it.

The **gutter** marks, beside the lines that make the claim, each Step's Kind, the Target it binds, its
Bound, its opacity, and its envelope check. A `mutate` Step with no declared Bound is marked `mutate!`:
its absence is not a static check's business and it is rendered here instead (§4). A Step invoking an
Operation whose request uses an Opaque Capability is marked opaque beside its Kind — opacity is a
Manifest fact, exactly as a Kind is, and the gutter carries it for the reason it carries the Kind: what
`hyper` cannot describe is not readable from the Step's own lines. A nested Procedure invocation
renders under the invoking Step's path with the transitive envelope §3 states.

**Where a name the gutter must follow resolves to nothing, the marker is `unresolved`.** A review does
not run `check` (§9) and does not decline, so a Step whose `definition:`, `operation:` or bound
Provider is not there renders like any other line with the mark its derivation would have carried:
there is no Kind to read, no opacity, and no envelope check, and a cell left blank would say `read`
by omission on the one screen that may not (ADR-0026, ADR-0064). One name and not four — the gutter
marks and does not classify, exactly as the change column marks a touched line without saying how —
and which name failed, and the path `hyper` looked for, is `FLAGS`' text below (§12). A nested
invocation whose `procedure:` is not there carries it for its transitive envelope, which is the same
absence one level up.

**Three of the five artefacts have no name to follow, and a Definition has one that reaches nothing on
this screen.** A Target declaration, a Manifest and a Repository declaration name no artefact at all. A
Definition names its Provider — and nothing rendered here is derived from a Manifest: the gutter marks
the Kinds it claims, the Targets it may bind and the `destroy` Operations it names, all of them authored
in the file being read, and `AUTHORITY` below is assembled from this artefact and a Target declaration.
So a Definition whose `provider:` is not there renders complete and unmarked, and the surface with
something to say about it is `check`, whose next act is an edit. That is the gutter's supply rule
holding rather than the review missing something: this screen annotates what `hyper` derived from these
lines, and nothing it derived is missing.

**Nothing an author wrote enters the gutter.** A comment renders verbatim in place, inside the line it
was written on, exactly as every other byte of the artefact does, and is read for nothing else (§3). It
is source rather than annotation: the gutter carries what `hyper` derived about a line, and a column
two authors write into is one where a marker and a comment contend for a cell that holds one of them —
which, since the gutter may not lie by omission (ADR-0026), has no answer that is not a rule about
whose text is dropped. The rule closes the class rather than that one instance: there is nothing else
in the format today an author could put there, and were there one, it would render in place too.

**The gutter is two columns.** A **marker** column carries the marks above; a **change** column, one
character wide, carries whether the range touched the line. They are separate because reading down the
marker column *is* the step table (ADR-0026), and a fact about the artefact's history interleaved into
it — on only the lines that happen to have both — is that column ceasing to be one table's. A marker
composes within itself, a Kind, its opacity and its Target being one Step's claim rendered in one cell;
the change mark is not that Step's claim and never joins them.

The marker column is as wide as the widest marker in this rendering, and a marker is never truncated: a
Target name is an identity in a reviewed artefact, and eliding a character of one is this screen
stating something other than what is about to be approved. Its header is the artefact's kind —
`PROCEDURE`, `DEFINITION`, `TARGET`, `MANIFEST`, `REPOSITORY` — which is true on all five, where a
header naming blast radius describes a Procedure's marks and a Definition's and misdescribes a Target
declaration's endpoint and a Repository declaration's retention policy. Nothing on this screen is sized
to the terminal: §9's truncation discipline governs a result set, which has an order and a limit, and an
artefact has neither.

**A review renders the working tree and no removed lines** (ADR-0057), and on the one artefact with no
file in the working tree it renders the bytes embedded in the binary (§11). What ADR-0057 fixes is that
one side of the range renders and never both; a built-in has no baseline to have drawn, so the rule
reaches it unchanged and only the word for where the bytes came from widens. Where the artefact has a
previous revision, the review renders the range and the change column marks `~` on every line the range
touched — a line whose content differs from the baseline, a line that is new, and the line a deletion
anchors to. One mark and not three: the gutter marks and does not classify, and a direction is `FLAGS`'
text below (§12). A deletion anchors to the opening line of the nearest enclosing structure — a removed
`bound:` to its Step's `- id:`, a removed Step to `steps:` — which is the line carrying the subject a
flag cites anyway, and the only anchor that is not whatever text happens to sit adjacent. Where the
review has no range the column has no content and no width, and the source sits two characters left.

**The gutter reads on all five reviewed artefacts, not only a Procedure**, and every marker it carries
is what `FLAGS` below may index — so what a review can say about an artefact is fixed here and nowhere
else. On a **Definition** it marks the Kinds claimed, the Targets bindable, and the `destroy`
Operations named; on a **Target declaration** the Kinds accepted, the Capabilities granted, the
endpoint, each credential slot's environment variable, and the opt-in admitting an `opaque` `destroy`
(§4); on a **Manifest** each Operation's Kind, its Repeatability, its opacity, the auth scheme, and the
Capabilities required; on a **Repository declaration** the `hyper` version pin and the retention
policy. Only a Procedure has Steps, so only a Procedure carries a Kind, Target, Bound or envelope mark,
and the last of those is why only a Procedure is guaranteed a flag.

**The change column reads on all five**, and that is a statement about which artefacts have a range
rather than about which renderings carry one. No kind is exempt from the column: where it is empty the
supply is missing and not the artefact excused, which is a different thing from `envelope` below being
Procedure-only because only a Procedure *has* Steps. An artefact with no range would have carried marks
had a Run supplied one.

### Where a range opens

**A range opens at the last Run that read the artefact, and never at `HEAD`.** The revision on the left
of `a91f0c2 → working tree` is what that Run supplies about this file (§7), so the range reads *since
this last took part in reaching the world* and the gutter's marks survive any number of commits between.
Against `HEAD` they would not: an agent that authored the widened Bound and committed it leaves the two
sides equal, and the review renders nothing to mark on the one branch a human is about to approve. The
same agent commits `env: STAGING_TOKEN` → `env: PROD_TOKEN`, so that argument is not a Procedure's and
`HEAD` is refused on all five.

**A dry-run entry supplies no range**, exactly as it is disqualified as the Comparison's baseline (§7).
Rehearsing a widened `destroy` Bound would otherwise retire the `WIDENED` flag that was the warning,
which is what the disqualification exists to prevent, and a rehearsal disarming a review surface is the
shape that rules everywhere rather than on the artefact the rehearsal named.

**Which Runs read an artefact is read off the Journal and never inferred from the working tree.** A
Procedure is read by the Run whose `run.json` names it; a Definition by every Run with a Step file
naming it in `definition`, a Manifest by every Run naming it in `provider`, a Target declaration by
every Run naming it in `target`, and a nested Procedure by every Run carrying it in a Step file's `path`
(§7). A Repository declaration is read by every Run there is, governing all of them (§11). Several Runs
qualify for all but the first, and the most recent of them supplies the range: a Run reads one working
tree, so every Step of one Run naming one artefact agrees about its revision, and the multiplicity is
across entries rather than inside one.

**What that Run supplies is decided by whether the artefact carries a revision of its own, not by its
kind** ([ADR-0067](../adr/0067-a-range-is-anchored-by-whether-the-artefact-carries-a-revision-not-by-its-kind.md)).
Two do. A top-level **Procedure**'s range opens at the `procedure_revision` that Run recorded and a
**Definition**'s at the `definition_revision` on the Step file that named it — both git blob ids over the
bytes that Run actually read, computable with `git hash-object` and wrong under no condition. The other
three carry no member of their own, and neither does a Procedure reached only by invocation, ADR-0048
carrying the top-level one and no other: their range opens at **that file's blob under the
`repo_revision` the same Run recorded**. It is resolved to a blob rather than rendered as the commit
because the range sits on the line beside the artefact's path and names one file's bytes there — a
commit in that position invites the reading #56 refused, a repository revision moving when a README does.
The resolution is derived at render and stored nowhere, as `ORDINAL` is (ADR-0049).

**A `repo_dirty` entry supplies no range to an artefact with no revision of its own.** The derivation
above is valid only over a clean tree: the bytes that Run read are not the bytes at that commit and are
nowhere in git (§7), so the blob resolves perfectly and names something else. The range is then rendered
against the wrong side and the gutter marks the wrong lines — a `~` on a line that did not move and none
on a line that did, which is the one screen ADR-0026 says may not lie. The supplying Run is therefore
the most recent that is neither a rehearsal nor dirty, and the absence below where there is none. It
costs an author who habitually runs against uncommitted edits their Target-declaration ranges, and that
is the right cost: the two members that survive a dirty tree are hashes of what ran, and marking a
revision `+` says the bytes differ where the failure here is an individually wrong mark that no suffix
makes readable.

**That entry is the one the gloss reads.** §10's *last ran* is the most recent Run of that Procedure
with dry-run entries filtered out, which is the baseline above under the same filter, so the header
renders one Journal entry twice — as a revision on the range, and as an age beside the gloss. One
lookup, two notations.

### The four absences

Four absences are named rather than rendered as one, and none of them refuses. A review resolves
nothing and reaches nothing, and the surface a human reads an agent's first artefact on has to work in a
fresh clone; but a header that rendered any two of these identically would omit one (ADR-0026). §12
carries their names.

- **A built-in Manifest has no range at any point in its life.** It ships inside the binary and has no
  file in the repository (§4, §11), so no Run can record a revision for it and none ever will — an
  absence that is permanent where the other three are repaired by an act. What it points at is where the
  fact *is* readable: a built-in's bytes move only when the binary does, which is `hyper_version`, which
  is the pin, which is a one-line edit to a Repository declaration that has a range of its own.
- **The Store is unreachable**, so nothing can be asked.
- **No Run read the artefact.** The Journal answered and holds nothing that anchors this file.
- **The clone does not contain the revision.** A Run answered, it is the right Run, and it named a
  revision — and the object is not here to read
  ([ADR-0071](../adr/0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md)).

**They rank `built-in`, `no-store`, `not-run`, `not-in-clone`, and the order is a pipeline rather than
four facts compared against each other**: each is reachable only where the one before it did not fire —
no file at all, then nothing to ask, then asked and empty, then answered and the bytes absent. Stating
it as the pipeline is what lets a reader check a rendering without reading the implementation, where the
two comparisons underneath it — a permanent fact beats a transient one, an unanswerable supply beats an
answered-and-empty one — say why the stages run in that order and not what the order is. Rendering *no
Store* on a built-in promises a range that repairing the Store will never produce, which is the defect
the four names exist to refuse, one layer in; and a rule that holds by accident is one no reader can
check a rendering against.

**The fourth is about the clone and the other three are about the Journal**, which is what makes it the
last stage and also what makes it the only one whose absence does not travel. It fires because a blob id
is content-addressed: the object is present in a shallow clone exactly where the artefact's bytes have
not moved since that Run, and absent exactly where they have — so on a shallow clone the range is
missing on every review that would have had something to mark. That is not a corner to render politely,
and what answers it is the supply rather than the sentence: the projection deepens the runner's clone
before anything runs (§10), and this absence is the safety net under it.

**Under `not-run` the sentence names the act that would supply a range, and the act differs by kind**:
*has not run* for a Procedure, *nothing has bound* for a Target declaration, *nothing has named* for a
Definition, *nothing has loaded* for a Manifest. The wire name is one for all of them (§12) and pays the
cost of a closed set's stability; the sentence pays no such cost, and *`staging` has not run* is false —
nothing runs a Target declaration, and a reader told so learns neither what happened nor what to do.

**`not-in-clone` names no act at all and renders the revision instead**, which is the one place the four
sentences diverge in shape rather than in wording. Three causes reach it and no one act repairs all
three: a shallow clone wants `--unshallow`, a partial clone wants a refetch or the filter dropped, and a
code branch whose history was rewritten wants nothing that exists — the object is gone for good. `hyper`
cannot tell the three apart without asking a remote, which it does not do here, so any act it named
would be a guess, and a sentence promising a repair that does not work is the defect this section
refuses one layer in. The revision is what the reader has to work with, and it renders whole
(ADR-0047).

```
  no baseline — retire-preview-envs has not run
  no baseline — nothing has bound staging
  no baseline — no Store
  no baseline — shell ships in the binary
  no baseline — a91f0c2 is not in this clone
```

The lead-in is one phrase across all five lines. A built-in's absence is permanent and the other three
are not, and that difference is carried by what the sentence says rather than by a second form of words:
two lead-ins would make the eye sort the absences before reading them, which is the ranking `FLAGS` is
the one surface permitted to do (ADR-0026).

**Where the absence is the Journal's, one absence serves both readings of that entry.** Where the entry
is missing it is missing for the range and for *last ran* alike, so the header states it once, on the
range's line, and the gloss line carries only what the artefact's own bytes supply — which needs no
Store and renders offline as it always has (§9). Rendering *no baseline — `retire-preview-envs` has not
run* above and *last ran: never* below is one fact twice: the argument for naming four absences is that
they are four different facts, and these are one fact in two notations.

**`not-in-clone` is the exception, and it is stated rather than left to be read off the rule above.**
The lookup succeeded there — the entry is present, dated, and the right one — and what failed is
downstream of it, so *last ran: 3 days ago* renders in full beside a range that cannot open. Suppressing
it for symmetry with the other three would drop a fact the Store held for the sake of a tidier header,
which is the omission ADR-0026 forbids everywhere else on this screen.

**And on an artefact with no file in the repository, the path goes with it.** The rule keys on there
being no file rather than on the artefact's kind or on its being built in — the decomposition ADR-0067
fixed for the range one paragraph up, and today one artefact answers to it, the roster being bounded by
a criterion rather than a list (ADR-0039, §11). Where there is no file there is no path to state, and
the sentence beside it already says where the bytes are, so the header's first line carries that
sentence alone. `path` is silent because the supply behind it is on the line, not dropped for want of a
home (ADR-0068). Rendering `shell` there repeats the name the sentence carries and the marker column
typed; rendering `<built-in>/shell` puts a value in a column of openable files that no `check` cites and
no editor opens; rendering the binary and its version says *in the binary* twice on one line, which is
the doubling the paragraph above refuses. The line collapses to its one field rather than leaving the
path's width as blank run-up — whitespace where a member was is omission wearing a rendering, on the one
screen that may not (ADR-0026).

```
$ hyper review shell

  MANIFEST          │  no baseline — shell ships in the binary
  ──────────────────┼─────────────────────────────────────────
```

**A Target declaration's header is the rule's other half rendered**, and it is two lines because there is
no Cadence beneath it — the gloss is a Procedure's, and its absence takes the line rather than leaving
one blank:

```
$ hyper review staging

  TARGET            │  targets/staging.yaml                    7c1e93b → working tree
  ──────────────────┼──────────────────────────────────────────────────────────────
```

`7c1e93b` is a blob id like the Procedure's above and not the commit it was resolved from, so the two
headers say the same kind of thing about the two files. What differs is only what supplied it.

**`AUTHORITY`** is the one table, because it is assembled from a Definition and a Target declaration
together and no gutter on this file could hold it: the claimed Kinds against the accepted Kinds, their
intersection, and the `destroy` Operations the Definition names (§5). Granularity following severity
(§12), the claimed-Kinds column carries `destroy` where the Definition names any: it is derived at
that one position rather than read. It is the one *table*, and not the one surface reading past this
file: the header above reads the Journal for its range and its gloss. What makes a fact a table is
being assembled from more than one artefact, and a table is heavier than a header line for a reason —
`AUTHORITY` has two artefacts' columns to align and the header has one artefact's value to state.

**It is one relation read from whichever end the artefact under review supplies, and three of the five
supply one** ([ADR-0069](../adr/0069-authority-is-one-relation-read-from-whichever-end-the-artefact-supplies.md)).
A **Definition** supplies the left end and renders a row for every Target it claims. A **Target
declaration** supplies the right end and renders a row for every Definition that claims it. A
**Procedure** supplies neither end and binds pairs, rendering one row per distinct `(Definition,
Target)` its Steps bind — the claims no Step here binds are checked against the Definition's own
`targets:` list (§3) and render on that Definition's review, and a row for one would sit directly above
an `ENVELOPE` flag that reads as contradicting it. The columns, their order and the caption do not move
between
the three: the filter changes and the table does not, which is what keeps one renderer behind them. A
column carrying one repeated value stays, a Definition review already rendering `DEFINITION` once per
row, and eliding it on a Target declaration would make the same table two shapes.

```
  AUTHORITY   assembled from definitions/ and targets/
  DEFINITION            TARGET           DEFINITION KINDS  TARGET KINDS         EFFECTIVE  DESTROY OPS
  preview-dns           cloudflare-prod  mutate destroy    read mutate destroy  m d        delete_dns_record
  preview-dns-observed  cloudflare-prod  read              read mutate destroy  r          —
```

**A Manifest and a Repository declaration are members of no pair, so the table has no end to read from
and does not render at all** — no header, no empty body, and no sentence. A Manifest's Operations
declare the Kinds a Definition may claim rather than claiming any, which is one column's supply and not
a membership; assembling it two hops out, over every Definition naming this Provider, would render rows
about pairings this artefact is not in and would traverse backwards the one edge this screen does not
traverse, the same edge whose absence leaves a Definition with a missing `provider:` complete and
unmarked above (ADR-0064). A Repository declaration governs every Run and pairs with nothing, and a
table of what its pin governs is an inventory rather than an intersection, which no discipline ADR-0026
fixes admits. The absence is stated here rather than rendered, exactly as `envelope` and `unbounded`
being Procedure-only are stated in §12 and render nowhere.

**Absent and empty are different, and the line between them is whether an edit could produce a row.** A
Target declaration nothing claims renders the header with an explicit empty state, on the argument that
carries `FLAGS`' empty block (§12): an edit to any file in `definitions/` puts a row here, so an absent
block would be ambiguous between *nothing claims this Target* and *the renderer had nothing to say* —
and the first is a fact worth the line, a granted `destroy` with no claimant being either a Target
awaiting its Definition or one whose Definition was deleted (ADR-0012). On a Manifest no edit to any
file produces a row, so there is no ambiguity to refuse and an empty block would assert a supply that
does not exist.

**Rows sort by `(Target, Definition)`, each by Unicode code point.** It is the ordering §7 sorts an
identity set's `members` by and the Comparison's tables reuse below, taken again rather than
reinvented, so two renderings of one review are byte-identical and it degenerates to Definition order
where the Target column is constant. Step order is refused even where it exists: reading down the
marker column *is* the step table (ADR-0026), so a table beneath it in step order is a second copy of
an ordering the reviewer already has, where a sorted one is a second index into the same rows. What
that costs is a reviewer relating a `DESTROY` mark to its row by name rather than by position — and the
name is on the Step's own `definition:` line, two lines from the mark.

**On a Definition, an absent declaration empties two of a row's cells rather than the row.** Where a
`targets:` member names nothing, the accepted-Kinds column and the intersection have no supply and
carry `unresolved`; the claimed Kinds and the named `destroy` Operations are this artefact's own and
render as they always do. Dropping the row would be omission on the one table for the reason the gutter
may not omit — a Definition claiming three Targets and rendering two rows says the third was never
claimed (ADR-0026). The fact therefore renders twice, as the `targets:` line's gutter mark and as this
row's empty half, which is the doubling the header already accepts between the range and the gloss: the
gutter says *this line*, and the table says *this pairing*.

**A supply that did not load renders `unresolved` where it has a cell, and a line beneath the table
where it has none.** A `targets:` member naming a file that is there and will not parse has no accepted
Kinds to render, and the cell carries the same name as one whose supply resolved to nothing: the two
absences differ in nothing this table can act on, and what a reader does about either is `check`'s row
on that file's own line (ADR-0064). A **discovered** row has no such cell — a Definition claiming this
Target that did not parse contributes nothing at all — so the table terminates with the count and where
to take it:

```
  3 definitions did not load · hyper check
```

It renders only where the count is non-zero and it names no member, on the shape of the `git diff` row
below. `review` does not decline for it: §9 fixes exit `1` for the artefact *under review* failing to
load, and a fault in a file the reviewer did not ask about is `check`'s (ADR-0064). What the line
refuses to leave standing is the omission. The row set on this one artefact is discovered rather than
authored, so a discovery failure removes a row outright where every other absence on this screen leaves
a marked one — and a three-row table read as the whole answer is the one table lying by omission
(ADR-0026).

**`FLAGS`** is the one editorial surface, and it is an index rather than a voice: it ranks nothing,
claims nothing, and points. Every row cites a line the gutter already marked and introduces no claim of
its own; a flag citing a line the gutter did not mark is a defect in the renderer rather than a
rendering. The vocabulary is seven names and §12 states them.

**A flag may read the text of any line the gutter marked, and cites the one line carrying its
subject.** The bound that matters is that it never leaves the gutter, not that it never leaves one row
of it — what ADR-0026 closes is a surface assembled from somewhere the reviewer is not looking, and the
gutter is by construction the thing they are looking at. That is what admits `ENVELOPE`, whose subject
is the `targets:` line and whose claim quantifies over every Step's `target:`, each of them marked.

**The gutter's supply is the artefact across the range** (ADR-0057): a marked line, and its counterpart
at the revision the header names. That is what admits `WIDENED`, which reads a direction off two values
only one of which is on screen — the baseline is the same artefact at the revision the whole screen is
scoped to, not a second file assembled beside it. A surface permitted to say *widened* has read the
baseline already; withholding it the numbers would keep the claim and drop the evidence for it, which
is the wrong half of ADR-0026 to enforce.

**A flag glosses what it renders** (ADR-0063). A `changed` row on the `cadence:` line carries the full
before-and-after text §12 states, and both expressions render through the gloss §10 states, exactly as
the header renders the working tree's. A gloss is the value in a second notation rather than a claim
about it, so it introduces nothing `FLAGS` may not introduce; and the alternative fails ADR-0005 on its
own example — a reviewer looking at `0 0 1 * *` → `*/5 * * * *` would read the rate that matters for one
side of the edit and cron for the other, on the screen the mandatory gloss was written for. What renders
here is the gloss and not the Journal fact beside it in the header: the entry is one, and a row about
two revisions has no side to hang it on.

The row keeps the one-line head every other flag has and carries the two glosses beneath it, the arrow
marking which side each belongs to. It wraps like the Comparison cell above and shortens nothing:

```
  CHANGED  line 8  cadence  0 0 1 * * → */5 * * * *
                            00:00 UTC on the 1st of the month · 1 run/month
                          → every 5 minutes · ≈8800 runs/month
```

Rows render in **line order**, with a file-level row last (ADR-0054). A review whose artefact draws no
flag renders the block with an explicit empty state rather than omitting it — an absent block would be
ambiguous between *nothing to flag* and *the renderer had nothing to say*, which is the ambiguity the
two named absences above already refuse to leave standing.

**Every line number this chapter renders is the working tree's**, counted from one over every line of
the file including blank ones. A flag's citation, a `gutter` row's `line`, a Refusal's caret excerpt
and its `EDIT ONE OF` rows are one numbering, and it is the numbering of the file the reader has
checked out — which is the whole of what a citation is for. A removed line has no number, having no
line; what a deletion is cited at is the anchor above. On a built-in Manifest it is the numbering of
the bytes this screen renders, which are the bytes `operation` writes back unchanged (§9, §11): there is
no file to check out, and a citation that resolved against one would have no referent at all.

A review resolves no credential, reaches no network, and invokes nothing.

**That claim is enforced on every git object read rather than left to hold by habit.** On a partial
clone an ordinary object read is a lazy fetch, so a review on one would reach the network without a line
of `hyper` intending to and the sentence above would be false where nobody would notice. Every object
read this chapter performs — a review's baseline, the five byte-reading classes of `THE CODE MOVED`
below, and its catch-all count — runs with lazy fetching off, and a promisor object that would need
fetching renders `not-in-clone` like any other absent one. The one place `hyper` reaches a git remote on
purpose is the Store's sync (§7), which this does not touch. The line is between an object read that
silently becomes a fetch and a branch sync `hyper` chose, and it is stated as that line rather than as
one environment variable, so an implementer reading git through a library is bound by the same rule.

```
$ hyper review procedures/retire-preview-envs.yaml

  PROCEDURE         │  procedures/retire-preview-envs.yaml     a91f0c2 → working tree
                    │  03:00 UTC every Monday · ≈4.3 runs/month · last ran 41 days ago
  ──────────────────┼──────────────────────────────────────────────────────────────
                    │   kind: procedure
                    │   procedure: retire-preview-envs
  envelope ✓        │   targets: [local, staging]
                    │   cadence: "0 3 * * 1"
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
                    │ ~     bound: 5

  AUTHORITY   assembled from definitions/ and targets/
  DEFINITION       TARGET   DEFINITION KINDS     TARGET KINDS         EFFECTIVE  DESTROY OPS
  uptime           local    read                 read                 r          —
  hetzner-staging  staging  mutate destroy       read mutate destroy  m d        delete_server

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  UNBOUNDED  line 15  step label    mutate with no declared bound
  DESTROY    line 24  step retire   delete_server, bound 5
  WIDENED    line 34  step retire   bound 3 → 5 since a91f0c2
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
writes no Journal entry and can never be either (ADR-0009). **An open entry is neither** — one with no
`outcome.json` (§7), whose Run may be in flight or may be gone, and which `hyper` never guesses about.
It is not disqualified the way a rehearsal is; it is not yet an entry a window can name, the header
below rendering an outcome it does not have. Where no baseline exists the header says so as a named
state — *no baseline — first Run of `<Procedure>`* — with every Record rendering as created or appeared.

**Each side of the window is an instant, and it is that entry's own last one** — `outcome.json`'s
`ended_at` where the Run wrote it, and the last Step file's where the entry's only account is a
`closed-by/` file (§7). That second is the instant §7 already names as "when the Run went quiet", read as
a timestamp and never as a verdict, and it is used here for the reason a duration may not use a
`closed-by/` file's: that `ended_at` is the *closing* Run's instant on the closing Run's clock, so a Run
reaped a week later would otherwise sweep every intervening Run's versions into its side of the window.

A **contested** entry (§7) is not that case and takes no special reading here: it holds an `outcome.json`
its own Run wrote, so the window's side is that `ended_at` like any other Run's. The `closed-by/` file
beside it names another Run's clock and is never an endpoint.

The header names both Runs, each with its id, its Trigger, when it started, its outcome, how long it
took, and the `procedure_revision` it recorded (§7). A duration derives within one Journal entry; two
entries' timestamps are never subtracted (§7). **A reaped entry's duration cell renders the word
`reaped`.** No duration exists to render there and the entry's account being a `closed-by/` file is what
says so (§7); a column whose every other cell is a duration and this one names why there is none is the
same discipline as the two named absences above, applied to one cell. A bare dash there reads as a
renderer that broke, and the outcome cell beside it says `failed` on plenty of Runs whose duration
derives perfectly well.

**A contested entry's outcome and duration cells are the owner's, unqualified**, and the contest is a
line rather than a cell value. An entry holding both an `outcome.json` and a `closed-by/` file renders
its own Run's outcome and its own Run's duration exactly as if no reaper had been there, and carries
beneath the header one stated line per `closed-by/` file — *contested — Run `<id>` recorded this entry
`failed` at `<instant>`*. Nothing is minted for it: every fact in that line is read off the file's own
keys, and the entry's account is present rather than absent, so this is not the discipline that produced
`reaped`, `not-run`, `no-store` and the `baseline_absent` members. Putting it in the outcome cell would
be the surface deciding between two accounts, which §7 is precise that it does not do; putting it
nowhere would be the tool holding a disagreement it never shows anyone.

The revision is named in full as a Procedure's — an unqualified `rev` sitting one table above a row
reading `repository revision` is two facts inviting one reading — and it renders on both lines whether
or not it moved, the header orienting a reader where the table below reports only what changed. **A
revision supplied by an entry that recorded `repo_dirty` renders with a `+` suffix**, `a91f0c2+`, on
this line and everywhere else this chapter renders a revision that entry supplied. The bytes that Run
read are not the bytes at that revision and are nowhere in git (§7); the marker is what stops the header
asserting otherwise, and what it costs the table below is stated with the catch-all.

### Three tables

One table per actor, Assets first: `YOU DID THIS`, `THE WORLD MOVED`, `THE CODE MOVED`. The split is by
actor rather than by field, and Asset against Observation is two tables rather than one column with two
values (ADR-0026). **All three render on every Comparison, header and count, whether or not they hold a
row.** An absent block is ambiguous between *nothing to report* and *the renderer had nothing to say*,
which is the ambiguity the two named absences above already refuse to leave standing, and three
zero-row tables under a `TOTALS` of zeros is how a Run whose every Step skipped reads.

**A row is a Record identity, and there is one per identity per table** (ADR-0058). Its change name is
read from what the baseline end held against what the subject end holds, and never from the versions
between them: a Record a single Run both changed and destroyed is one `destroyed` row spanning both,
not two rows. What stands at each end is §7's Head derivation with that entry's instant above as a
cutoff — the version a reader would have called the Head had they looked then.

**Which identities are eligible comes from the identity sets, not from the Store** (ADR-0058). A row
exists for an identity some Step of the subject Run or of the baseline Run concluded about (§7), which
is what keeps another Procedure's work out of this Procedure's tables — the same evidence that already
decides *vanished*, *appeared* and *nothing moved* below, deciding eligibility outright. The endpoints
are then read without asking whose Run wrote them: a Record this Step concluded about, which another
Procedure moved in between, renders `changed` and the gap shows in `ORDINAL`. This surface says *this
differs from when we last looked* and nothing else, and reading `run_id` to name a row would make it
report authorship across Procedures, which is the join the window rule exists to prevent.

Assets render `created`, `changed`, or `destroyed`; Observations render `appeared`, `changed`, or
`vanished`. The three in each table are exclusive. **Where `appeared` and `changed` are both true —
an identity absent from the baseline Run's set, present in the subject's, whose series holds an older
version that has since moved — `appeared` wins.** The Disposition-derived name beats the Record-derived
name wherever both fire, which is one precedence rule rather than a per-pair one and never arises in
the Asset table, `created` and `destroyed` both being Record facts. `changed` there would assert that
the baseline Run had this in view and saw something else, which is the one thing that is false.

A Tombstone is a marker inside the Asset table rather than a class of its own. A series
whose first version is a Tombstone (§7) renders `destroyed` and never `created`, though the baseline
holds no version of it and the subject does: what the subject holds is a destruction, and reading
*absent, then present* as a creation would report the opposite of what happened. It needs no marker to
be told apart from the destruction of an Asset `hyper` built — an ordinary `destroyed` row carries the
last known state's fields and this one has none, so *destroyed, and `hyper` never saw what it was* reads
off the empty column rather than out of a second rendering of the same fact. There is no
rename class: identity is a Manifest-declared field of an upstream response (§7), so a rename is an
unfamiliar name appearing and a familiar one going quiet, and it renders honestly as both.

Every row in all three tables carries its Target, its Definition, its name, and its change. Asset and
Observation rows add a `FIELDS` cell, and what it holds follows the change name.

- **`changed`** renders the fields that moved, `path: old → new` each.
- **`created`** and **`appeared`** render every projected field of the one version, `path: value`, with
  no arrow — there is no other side, and the cell is describing what the thing is rather than how it
  differs.
- **`destroyed`** renders `† confirmed <time>` and then the last known fields. Those come off the
  Tombstone itself, which copied the previous Head's `fields` forward (§7), so a Record changed and then
  destroyed in one window shows the state it was in when it ended. A series whose first version is a
  Tombstone has no `fields` to copy and renders the marker alone, which is the empty column the section
  above reads *`hyper` never saw what it was* off.
- **`vanished`** renders the baseline end's fields, with no marker. It is the Observation table's mirror
  of `destroyed`: a row saying a thing stopped being there while declining to say what it was leaves its
  reader no other page to go to, the Store holding no version minted for a disappearance.

Fields render **sorted by Unicode code point**, which is the Store's own canonical encoding read out
rather than a second ordering, and there is **no cap on how many render**. A wide cell is a Manifest
author's projection choice rendered honestly, and capping it is `hyper` guessing at a number in the way
ADR-0045 declined.

**A value renders whole or renders `changed`** (ADR-0059). A scalar over 120 characters, one carrying a
newline, and anything nested are one class with one rendering: `path: changed` on a `changed` row, and
the bare `path` on a one-sided one, where `changed` would be false and the field's name is the whole of
what the page can honestly carry. There is no truncated form. Two values agreeing for their first
hundred characters and diverging after would render as identical bytes on a row asserting they differ,
and the newline test is absolute rather than a length in disguise, because `shell` projects `stdout` and
`stderr` as unparsed text with no cap between a chatty command and the Store (ADR-0052). The budget is a
stated constant and not the terminal's width: colour and width are the only differences between an
interactive rendering and a piped one, and a width-derived budget makes the two disagree about content.
`--json` carries every value whole regardless — the elision is this column's geometry and not a fact
either surface states.

**Rows render sorted by `(Target, Definition, name)`, each by Unicode code point** — the columns read
left to right. It is the ordering rule §7 sorts an identity set's `members` by and §6 orders an
Expansion by, reused rather than reinvented, so two renderings of one window are byte-identical and
diffable. Grouping by change name is refused on ADR-0054's argument: a rendering that puts the
destructions first ranks its own rows, which is the one thing only `FLAGS` may do, and it makes the eye
read the `CHANGE` column twice. Ordering by `written_at` is refused because the laptop and the runner do
not share a clock (§7) — a refusal this table can afford because it ranges over identities and has a name
axis to order on instead. §9 states the general rule and names the surface with no such fallback: `runs`
ranges over Journal entries, where time is the only axis there is, and it pays the skew (ADR-0065, §13).

### The ordinal

Asset and Observation rows carry an `ORDINAL` beside the change, `4 → 5`. It is each version's ordinal
position in the `written_at` ordering of what the Store holds when the Comparison is read, derived from
the same directory listing that derives the Head (§7). It is stored nowhere, and it is never the version's
identifier, which is the Run that wrote it (ADR-0011, ADR-0049).

What it buys is **adjacency**. A Definition is reachable from more than one Procedure, and `--since` folds
several Runs into one rendering, so `4 → 7` is the only place a Comparison admits that versions exist
between its two sides that it is not showing. It is not a count of times the Record was checked: a version
is minted only where the bytes moved (§7), so a Record read hourly for a month and never changing sits at
`1`.

An ordinal is unstable, in both tables and under one rule. Compaction removes interior Observation
versions and renumbers every version above them (§7). A read-only Run proceeds offline and pushes when it
can (§7), so a laptop's Observations slot in beneath versions a runner already pushed and the ordinal
moves with no Compaction anywhere — the quieter of the two, Compaction being an explicit command with a
`git log` behind it. Two renderings of one Store can therefore disagree and neither is wrong. That is
affordable for exactly one reason, and it is a rule rather than a property of today's flags: **a version
is named by its Run**, and no surface accepts an ordinal as input.

Nothing marks a gap and nothing counts what a window hides. Both would be sound in the Asset table,
Compaction never removing an Asset version, and unsound in the Observation table beside it, which is two
guarantees under one column head.

**`–` means this side has nothing for this row to name.** On an Asset row that is a side holding no
version: it repeats what `CHANGE` already says on a `created` row, and earns its place on the
`destroyed` row of a series whose first version is a Tombstone (§7, ADR-0033), where `– → 1` reads as
*`hyper` ended a thing it never built* in the column the eye is already in. On an `appeared` or a
`vanished` row what the side lacks is a **view** rather than a version, so they render `– → n` and
`n → –`: an identity that left this Procedure's sight and returned unchanged has a version standing on
both sides, and a disappearance mints none at all, and in each case printing the standing ordinal on the
side the Procedure could not see puts two numbers on a row and invites a reader to difference them
across exactly the boundary the row exists to report.

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

A Step whose Disposition carries **no set at all** is that rule at its limit and is read the same way:
it contributes no identity to its side and removes none from the other. That is the reaper's
*attempted, outcome unknown* — a `closed-by/` file, which carries `step`, `disposition`, the `id` and
the code facts and no `identities` (§7). It is a different animal from the same Disposition written by
the Run itself, which carries the conclusions it did reach, and since ADR-0076 the difference is which
file it is rather than which key it holds. It is also every *attempted, world untouched*, whose
set is absent by construction (§7). Nothing renders *vanished* on their account and an identity
returning next Run renders no *appeared* on it either.

The three tables never join an Observation series to an Asset series. That join is the drift detection
`hyper` has no engine for and never performs (ADR-0010).

A Refusal gets no row. Nothing reached the world, so it is not a change; it appears in the header as a
Run in the window and is one `runs` query away. A Run whose every Step skipped renders no rows at all,
which is how the Dispositions read back distinguish it from a Run that did the work (§6).

Pattern facts — a retry's attempts, a poll's iterations, a paginated read's pages — do not enter the
three tables. They render on `runs` and only where they are not the trivial single call (§7).

`THE CODE MOVED` reports over the closed enumeration of code facts §12 defines, terminated by the
mandatory catch-all row §12 states, counting every other line of every reviewed artefact that moved.
Its header count is the classed rows and never the catch-all, which is why the table can read `0 facts`
and still carry a row.

The `procedure revision` row is `the digests` class emitting one of its members rather than a tenth
class: that class is *every member of the Provenance* (§12), so a member joining the field set brings
its row with it, and a class defined as every member with one carved out is an enumeration that has
stopped being checkable. It and the `bound` row are not one fact rendered twice — an edit that moves no
classed fact at all still moves the revision, and that row is the one saying a reviewed artefact is not
the one that ran last.

**The first column is `SUBJECT`, and it carries a kind-qualified name** — `procedure
retire-preview-envs`, `target staging`, `definition hetzner-staging`, `manifest hetzner`, `repository
hyper.yaml`. A header reading `DEFINITION` misdescribes every row whose fact belongs to something else,
which is the defect the gutter's own header was fixed for above, and a bare name is ambiguous across
kinds; §12 fixes each `kind:` to one directory, so the kind and the name together are a whole path in
one short cell. **A class emits one row per `(subject, fact)` pair**, always — a class is a kind of fact
and not a row, so a Definition's declared Kinds and its Target's accepted ones moving in one window is
two rows under one class name. Of the nine, the selector, the Bounds, the Cadence and a Step's `target:`
are a **Procedure**'s; the required Capabilities and the exposed Operation set are a **Manifest**'s; the
credential source is a **Target**'s; a Definition's claimed Kinds, its bindable Targets and its named
`destroy` Operations are a **Definition**'s. On `the digests` each member takes its own —
`procedure_revision` the Procedure, `definition_revision` the Definition, `manifest_digest` and
`origin_digest` the Manifest, `hyper_version` the **Repository declaration**, since the pin gate refuses
any binary whose version differs from the repository's in either direction (§11) and that member moving
*is* the pin moving. `repo_revision` alone belongs to no artefact and renders `—`, which is therefore
the one cell in the table that has it.

**Which Manifests a Run read is the Step files' `provider`** (§7), and the two Manifest classes cannot be
computed without it: `manifest_digest` names the bytes that ran and never which Provider they were, so
enumerating the Manifests to diff would otherwise mean resolving each Step's `definition` to its
`provider:` at that Step's own revision — a git object per Step, in the surface that already has both
revisions of the artefacts themselves. It is the same member a review's range on a Manifest reads, and
a field two surfaces need is not one added for either's convenience.

**The `cadence` row's two expressions are glossed**, each under the cron it glosses in its own cell, on
§10's rule that no surface renders a Cadence as the expression alone (ADR-0005, ADR-0063). Cron is
write-only for humans and agents alike wherever it is read, and this row is the one place the whole
tool renders a Cadence *moving* — ADR-0005's own `0 0 1 * *` → `*/5 * * * *`, whose ≈8,800× is legible
in the two rates and in nothing else on the page. What renders is the phrase and the rate; there is no
Journal fact beside them here, the header above already naming both Runs and this row being about two
revisions rather than about what stands now.

**The cell stacks the expression, the phrase and the rate, and wraps rather than shortening any of
them** (§10). A phrase is a total function of the five fields with no short form and no truncation, so
the column widens and the row runs to as many lines as the two expressions need:

```
  SUBJECT                        FACT     FROM                     TO
  procedure retire-preview-envs  cadence  0 0 1 * *                */5 * * * *
                                          00:00 UTC on the 1st     every 5 minutes
                                            of the month           ≈8800 runs/month
                                          1 run/month
```

The rate gets no row and no column of its own. A class emits one row per `(subject, fact)` pair and the
rate is not a fact that can move while the Cadence stands still, so a row for it would be the same fact
rendered twice — and a column would be one every other row in the table left blank.

Rows render sorted by `(SUBJECT, FACT)` on Unicode code point, with the `—` subject after every named
one and the catch-all last of all. `—` means *this fact belongs to no artefact you can open*, so it
sorts away from the rows that do, and §12 already fixes the catch-all as terminating the table.

**The catch-all counts `git diff` lines** — added and removed as git counts them, a modified line being
two — over the reviewed five and nothing else, minus the lines a classed row above already reports. The
unit is the command's because the row names the command, and any other unit makes the row disagree with
its own evidence; the word *other* is what makes the enumeration and the count sum to the whole rather
than overlap, so `hyper` maps each classed fact to its lines at both revisions. It already loads both
revisions of the reviewed artefacts — four of the nine classes read off two Journal entries, and the
Cadence, the required Capabilities, the credential source, the declared Kinds and the Operation set do
not — so that map costs nothing beyond what the table already pays. Nothing outside the reviewed five
counts, which is §7's rule arriving from the other side: `repo_dirty` marks that same file set "so the
marker and the count agree on what code is by construction". The generated workflow is out
particularly — it is projected rather than authored and byte-exact against what `project` would write
(ADR-0046), so a change in it is a `hyper` version move already in Provenance, a Procedure move already
classed, or a hand-edit, and a hand-edit is `check`'s Refusal rather than a row here.

**Where either side recorded `repo_dirty`, the command is suppressed** and the row renders
`N other lines changed` alone. The bytes that Run read are nowhere in git — a dirty baseline's working
tree is gone for good — so `git diff <rev> <rev>` does not reproduce what moved, and printing a command
that does not reproduce is worse than printing none. `N` and the classed rows are still computed between
the two committed revisions, which is what the Provenance names and what a reader can check out; the one
thing the table cannot then vouch for is marked in every cell that names a revision, by the `+` the
header states. §7 already accepts this shape from the other end, a `closed-by/` file omitting its code
facts "where the dead Run's revision resolves them, and absent where it does not, which is every Run
that recorded `repo_dirty`".

**Where the clone does not contain a revision the window names, this table degrades partially rather
than absent.** Four of the nine classes read off two Journal entries and are unaffected; the Cadence,
the required Capabilities, the credential source, the declared Kinds and the Operation set read bytes,
and so does the catch-all count. What renders is every row the surviving classes produce, and the
catch-all row is **replaced** by one line naming what could not be read:

```
  other lines could not be counted · git diff 1f0a3d7 88bc402
```

It names no member, on the shape of the `3 definitions did not load` row above, and it replaces rather
than joins the catch-all, two terminating lines being the doubling this chapter refuses everywhere
else. **The command it carries is `git diff` and not the act that would repair the clone**, which is
where this line and the review's header part company: a reviewer is standing in the deficient clone,
having just run `hyper review` in it, where the reader of a job summary is very often on a phone and
the deficient clone is a runner's that no longer exists. `git diff 1f0a3d7 88bc402` is what *that*
reader can run, and it returns strictly more than the count would have. The existing `repo_dirty`
suppression stacks on top unchanged, the two suppressions being about different commands and reaching
this row for different reasons.

The reason this is a safety net and not the ordinary case is that the ordinary case would be
intolerable. A **Target declaration** carries no Provenance member at all — no revision, no digest — so
on a shallow clone an `env: STAGING_TOKEN` → `env: PROD_TOKEN` edit produces no classed row, no
catch-all count, and, before the rule below, a `TOTALS` line reading *the code did not move*. That is
the credential source class §12 minted, rendered as its own negation, on the half of the thesis that
says nothing changes unseen. §10's projection is what stops it happening.

**`TOTALS` counts rows** (ADR-0058), so the line totals what is on the page above it and a Record
changed and destroyed in one window counts once. `changes` is the asset rows plus the observation rows;
the tombstone count is a **subset** of the asset count and is never added to it; code facts enter
neither number. All four numbers render, `0` included, so the line is scanned rather than parsed. The
last segment is a phrase and not a count — summing a classed fact, a repository revision and a line
count into one integer is three incommensurable things under one head — and it renders its negative
explicitly, *the code did not move*, on the same argument the empty tables above render.

**It has three forms and they are tested in order**: any classed row rendered → *the code moved*;
otherwise the absence line above rendered → *the code could not be fully read*; otherwise → *the code
did not move*. The order is what makes the line honest rather than merely careful. A surviving classed
row is positive proof, where the absence line is proof of nothing either way, so a window in which a
Bound moved *and* a Target declaration could not be read is reported by the fact — the line above
already naming what went uncounted. What the ordering removes is the one reading this table may never
produce: the negative asserted over bytes nobody read.

A fold across several Procedures — `--since` over a Store with more than one, or the whole-Store
mode — renders one block per Procedure, each with its own header, its own three tables and its own
`TOTALS`, in Procedure-name code-point order. There is no grand total: it would sum across windows with
different baselines, which is the cross-Procedure reading the window rule refuses, and a reader looking
for which Procedure did the damage is served by the per-block line and misled by the sum.

```
$ hyper changes --since 2026-08-04T09:12:00Z

  retire-preview-envs
  BASELINE  01991c3a-7d40…  cron           Tue 4 Aug 09:12  completed  1m48s  procedure rev a91f0c2
  SUBJECT   01991ea6-b118…  igor@thinkpad  Thu 6 Aug 11:03  completed  2m31s  procedure rev b0c94f1

  YOU DID THIS   5 assets
  CHANGE     TARGET   DEFINITION       RECORD        ORDINAL  FIELDS
  destroyed  staging  hetzner-staging  preview-8801  4 → 5    † confirmed 11:02 · region: fsn1 · server_type: cx22
  destroyed  staging  hetzner-staging  preview-8802  3 → 4    † confirmed 11:02 · region: fsn1 · server_type: cx22
  destroyed  staging  hetzner-staging  preview-8806  7 → 8    † confirmed 11:03 · region: fsn1 · server_type: cx22
  changed    staging  hetzner-staging  preview-8815  9 → 10   labels.retire-after: 2026-08-18 → 2026-08-25
  created    staging  hetzner-staging  preview-8821  – → 1    region: fsn1 · server_type: cx22

  THE WORLD MOVED   2 observations
  CHANGE   TARGET  DEFINITION  RECORD            ORDINAL  FIELDS
  changed  local   uptime      cert.hyper.dev    22 → 23  days_left: 41 → 34
  changed  local   uptime      status.hyper.dev  22 → 23  status: 200 → 503

  THE CODE MOVED   3 facts
  SUBJECT                        FACT                 FROM     TO
  procedure retire-preview-envs  procedure revision   a91f0c2  b0c94f1
  procedure retire-preview-envs  step retire · bound  3        5
  —                              repository revision  1f0a3d7  88bc402
  2 other lines changed · git diff 1f0a3d7 88bc402

  TOTALS  7 changes · 5 asset · 2 observation · 3 tombstone · the code moved
```

## The Step table

Every Run renders one row per Step: its position, its authored `id`, its Kind, its Disposition in the
vocabulary §6 and §12 fix, and how many Records it concluded about. Under it sits the terminal line
every Run ends with, in the form stated two sections below.

```
$ hyper run retire-preview-envs

  STEP  ID      KIND     DISPOSITION  RECORDS
  1     probe   read     ran          12
  2     label   mutate   ran          8
  3     retire  destroy  ran          3

  completed · exit 0 · run 01991ea6-b118-7c93-8d41-6b2f7ae05c19
```

`RECORDS` is the size of the identity set that Step's Disposition carries (§7): what the Step concluded
about, and not the versions it wrote. The two are different numbers and differ every time a Record comes
back unchanged, which under a Cadence is nearly every Run (ADR-0030). The Run above is the one the
Comparison renders: `probe` concluded about twelve Observations and two of them moved, `label` about
eight Assets and two of them moved, `retire` confirmed three destroyed — twenty-three conclusions and
the seven rows the `TOTALS` line above counts (ADR-0058).

The written count is the less useful of the two here, and on a `read` it is barely a fact at all: a Step
that checked twelve hosts and found none of them moved wrote nothing, so a column reporting versions
renders a Procedure that checks a hundred hosts hourly as a table of zeros, indistinguishable from a Step
that read nothing at all.
What each Record did is the Comparison's rendering rather than the Run's; this column says how much each
Step touched, which is the fact only this page carries.

A cell takes one of three forms.

**`n`** — the set is all the Step reached.

**`n of m`** — the Expansion reached `m` and the Step concluded about `n`, the rest unaccounted for: a
Step whose process died mid-sequence, one a projection failure halted, or a `read` Expansion that drained
and then halted (§6, §7). It is §7's arithmetic rendered in the column the eye is already in, and it names
no member — which Assets those are is `expanded_to`'s and nowhere else.

**`–`** — no set exists, rather than a set with nothing in it. *refused*, *skipped by condition*,
*never reached* and *attempted, world untouched* conclude about nothing (ADR-0030), and the dash is what
tells them from the *ran* Step whose Expansion resolved to nothing, whose set is written empty and whose
cell reads `0`.

The fourth of those is the one a reader may expect to render `0 of m`: a `destroy` whose Expansion
resolved to five and whose first request never left did name five Assets and touch none of them. It
renders `–` all the same. `n of m` means *unaccounted for*, and the whole content of that Disposition is
that nothing is unaccounted for — rendering the safest state in the tool through the column form
reserved for doubt would invert it (ADR-0062). The five names are on the entry in `expanded_to` (§7)
either way.

*skipped as already recorded* carries a set — the head versions the skip test read — so a Step that made
no call renders a count, and it may be larger than a neighbouring Step's that did. That is the
evidentiary content of that Disposition rather than an artefact of it: the number says how much the skip
covered, and a Step that skipped five hundred Assets did not do nothing.

A Step that skipped some members and called for others renders `n`, not `n of m`. The skip test reached
a conclusion about every member (§6, ADR-0056), so nothing is unaccounted for, and `n of m` says a Step
stopped short of its Expansion. How much of that `n` was a call is the Comparison's to render: the
members that ran are the rows in `YOU DID THIS`, and the members that skipped wrote no version and have
none.

## The Refusal

A Refusal is the most verbose surface in the tool (ADR-0026). No flag, no confirmation and no override
reaches one (ADR-0001), so this rendering is the entire path back to a passing review.

It renders three things. The Step table above, with the refused Step's cell reading `–`: it concluded
about nothing, and a Refusal withholds Records for that Step and for no other, so the Steps before it
carry the counts they earned. The caret excerpt shows the offending line in its own context, which is
what lets a reader judge whether raising the Bound is the right fix at all or whether the selector is
the thing that is wrong. The `EDIT ONE OF` table says where to edit. Both remediations are rendered,
and the narrowed selector is speculatively re-expanded so its count is on the page — a read `hyper` did
not have to perform, worth it because the alternative is a reviewer widening a destroy Bound for want of
the other number.

**A relative predicate is glossed with the instant it resolved to** (ADR-0034), on this surface and on
the one other where a Run renders a selector: `show --expansion`, where each Step carries the selector
it resolved (§9). It renders as a `=`
note beneath the caret, first among them, because it reads the excerpt above it where the others read
the check; and beside the value in the `EDIT ONE OF` row, whose trailing column already carries what
`hyper` derived about that remediation. The proposal is glossed as the current value is: it resolves
against the same instant, which is this Run's start and not the reader's clock (ADR-0034).

This is the only gloss a Run's surfaces render, and none of it reaches a review or a `check`: both are
offline, neither has a Run, and an instant is what a Run supplies (ADR-0063). The gloss §10 states runs
the other way — it is on every surface that renders a Cadence, a review included, and needs no Run at
all.

The sentence beneath the table names no count. What it carries is that the Steps before the refused one
ran and that nothing rewinds them (§6), which is a different fact from how many Records they touched and
the one a reader of this surface needs; the counts are in the column directly above it.

Under those three sits the terminal line every Run ends with, in the form the next section states: on
this path it carries the remediation pointer, `show --expansion` being where the whole expansion is read
back rather than the count the caret gives.

```
$ hyper run retire-preview-envs

  STEP  ID      KIND     DISPOSITION  RECORDS
  1     probe   read     ran          12
  2     label   mutate   ran          8
  3     retire  destroy  refused      –

  refused: bound exceeded — nothing was deleted

    procedures/retire-preview-envs.yaml:34
     32 │           - field: created_at
     33 │             older_than: 14d
     34 │       bound: 5
        │              ^ expansion resolved 23 assets on staging
        │
        = older_than: 14d resolved to 2026-07-23T11:03:18Z
        = checked at expansion, before the first call
        = no flag overrides this (ADR-0001) — the way past is an artefact edit

  EDIT ONE OF
  FILE                                 LINE  FIELD           FROM  TO
  procedures/retire-preview-envs.yaml  34    steps[2].bound  5     ≥ 23
  procedures/retire-preview-envs.yaml  33    steps[2].over   14d   30d   expands to 4 · 30d is 2026-07-07T11:03:18Z

  steps 1-2 ran and what they did stands. step 3 wrote nothing.

  refused · exit 77 · hyper show 0199206d-4e15-7c30-9b8a-52d9ea01f7b4 --expansion   for all 23
```

### Where no Step was reached

Most of the closed `error_code` set declines before Step 1 — `check` re-runs in full at Run start, so
every static code reaches a Run this way, beside the credential pass, the sink gate and the Store
schema test (§6,
ADR-0061). Each of the three parts above is rendered where it has content and omitted where it has
none.

**The Step table is omitted entirely**, and the sentence beneath it becomes a flat statement that
nothing ran. An empty table asserts *we looked at the Steps*, which is false, and the fact that no Step
was reached is the most important thing on the page — an absence cannot carry it.

```
  nothing ran. no step was reached.
```

**The caret excerpt survives**, and every code that reaches a Run this way cites a line: a static code
its own artefact and line, `cadence-run-once` and `cadence-secret-output` the `cadence:` line,
`credential-absent` the `env:` line of the Target declaration whose slot the environment did not fill,
and `secret-sink-absent` one Step line per Step that would have needed a sink. The two Cadence codes
cite the `cadence:` line although the fault is a Manifest's — an undeclared `repeatability:`, a
declared `secret:` — because the walk goes to the Manifest and the citation comes back to the artefact
whose author can act. That is also what keeps a caret off a Manifest verified by digest, which would be
`store-schema-unsupported`'s problem below arriving on an artefact rather than on evidence. The `=` note beneath it carries the
phase §7 derives from the code — `= checked at run start, before the first step` — which is what tells
a reader this Refusal preceded execution rather than halted it.

The one exception is `store-schema-unsupported`, which **renders no caret**. The file it cites is a
Store file: evidence rather than an artefact, and editing it is editing evidence (ADR-0011). A caret
excerpt over a Store path would render, in the surface whose whole purpose is *here is the line to
change*, a file the reader must not touch. Its file and field go in the `=` notes instead.

**The remediation table is rendered only where an artefact edit is the way past**, and its header stays
`EDIT ONE OF` on every path where it appears. The set it leaves behind is larger than one code and has
three remedies, none of them an edit and all of them keeping `77`'s promise that a verbatim retry
refuses identically: a **command** (`projection-stale` → `hyper project`, `store-absent` → `hyper store
init`), a **different binary** (`store-schema-unsupported`, `manifest-schema-unsupported`,
`version-pin-mismatch`), an **act on the environment** (`credential-absent`), and a **different
invocation** (`secret-sink-absent` → the same command with `--secret-out <path>`). The last is the
only remedy on this list that the Run's own operator can take without leaving the shell, and it is
also the only one no generated workflow can take at all, the projection being byte-exact with a
compiled-in executor — which is why a Cadence over such a Step is refused at `check` rather than
allowed to reach this rendering unattended (§4, ADR-0077). On all of them the
`=` notes carry the remedy, and they **name the remedying command verbatim**. A check knows its own
remedy, so naming it states a fact rather than editorialising: `FLAGS` is the one surface ADR-0026
restricts, and it is restricted for summarising other lines, which this does not do. Prose that
describes a command without naming it is the same fact rendered worse.

Where a Refusal carries more than one member — the two phases that evaluate many checks together (§7) —
**every one of them renders**, each with its own caret excerpt and its own remediation table where it
has one, in the array's order. The Refusal is the entire path back (ADR-0001): rendering the first of
five costs an operator five round trips, each ending in another `77`. What the terminal line names is
the first.

```
$ hyper run retire-preview-envs

  nothing ran. no step was reached.

  refused: credential absent — 2 targets

    targets/staging.yaml:12
     11 │ auth:
     12 │   token: {env: STAGING_TOKEN}
        │          ^ the environment does not hold STAGING_TOKEN
        │
        = checked at run start, before the first step
        = set it in the environment, or wrap the invocation — op run --, direnv, aws-vault exec --

    targets/cloudflare-prod.yaml:9
      8 │ auth:
      9 │   token: {env: CLOUDFLARE_API_TOKEN}
        │          ^ the environment does not hold CLOUDFLARE_API_TOKEN
        │
        = checked at run start, before the first step
        = set it in the environment, or wrap the invocation — op run --, direnv, aws-vault exec --

  refused · exit 77 · run 0199206d-4e15-7c30-9b8a-52d9ea01f7b4
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

**The pointer is earned by truncation, not by the outcome.** `show --expansion` stands on the line
above because the caret cannot fit twenty-three assets in an excerpt; where the page is complete — every
Refusal that declined before Step 1, whose caret and remediation table are the whole of it — the line
falls back to the completed form, `refused · exit 77 · run 0199206d-…`, and the Run id renders whole
there as it does everywhere (ADR-0047). Pointing at `show` would send a reader to look up what they are
already looking at.

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

**An entry that was written is named, whether or not it reached the remote.** An effectful Run that lost
the Store at its Run-start sync exits `75` having already written `run.json` locally (§7), so the line
names that id: the rule is *no entry was written*, and one was. It is resolvable by `hyper show` on the
machine printing the line, and it goes out with the next Run that syncs. On a runner the workspace
evaporates and nobody reads it again — which is a fact about that executor, not a reason to withhold a
true id from the operator in front of the one that is there.

## The row stream

`--json` switches every surface above to NDJSON: one JSON object per line, each carrying a `type`
discriminator. Both forms come out of one renderer (ADR-0026), and the mapping is total — every row of
every table above is one object, each of the two headers above is one object, and nothing rendered is
left out. A long Run streams.

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

**No id and no digest is abbreviated anywhere on the wire.** Every revision, every digest and every
Run id goes out whole — a `provenance` row's `definition_revision`, `manifest_digest` and
`origin_digest`, an `artefact` row's `baseline`, a `window` row's `procedure_revision`, a `code` row's
`from` and `to`. The page abbreviates a fact to be recognised (ADR-0047), and this surface carries
none to be recognised: a consumer resolves what it is handed against a git object or a `sha256sum`,
and a shortened value is one it has to go somewhere else to complete. The one string that keeps the
page's abbreviation is the catch-all row's `command`, which is a command a reader runs rather than an
id the row reports, and which `git` resolves short.

Consumers filter rows rather than queries. There is no expression language over the stream and none
behind it (ADR-0013): `hyper changes --json | jq 'select(.type=="asset")'` is the shape of every
arbitrary predicate.

**The stream's order is the table's, and here it is a contract rather than a consequence.** One renderer
means the rows arrive in the order the page renders them (ADR-0026), but this is the surface that makes
that order normative: a row goes out on its own line, there is no cursor behind the stream, and a consumer
cannot re-sort what it has already printed. §9 states the orders and the axis a truncation marker names.

**The review's header is one `artefact` row**, on the `window` row's precedent below rather than the
`gutter` row's: a header cites no line, so one row per rendered line would invent an anchor the surface
does not have. It is named for what it carries — what is under review — because every other type in the
stream names its content and none names a position on a page. It carries the kind, the path, the
baseline revision, and on a Procedure declaring a Cadence the expression, the phrase, the rate as a
number, and the last entry the header rendered an age from.

**It carries the gloss's parts and never the composed string.** The `gutter` row goes the other way and
says why — a decomposition is a second rendering of the same fact, and the second one can be wrong about
the first — but a marker is one derived fact rendered in one cell, where the gloss is several facts with
several supplies sharing a line, which is the `window` row's shape and not the marker's. Carrying both
the parts and the string would be the failure `gutter` names. §12 states the `flag` rows are what a
reviewing agent reads to decide whether to escalate; a rate it must re-parse out of a phrase is that
agent doing the renderer's arithmetic backwards.

**`rate` is the number the page renders**, rounded to the two significant figures §10 states rather than
carried at full precision beside a rounded rendering — which would be the same fact in two
representations that can disagree, and the `≈` on the page is what says it was rounded. It is a JSON
number and never a band or a word: a rate whose only form is prose puts the escalating agent back where
this row exists to take it out of.

**The baseline's absence is named on the wire as it is on the page.** `baseline` is written where there
is one; where there is not, `baseline_absent` carries which of the four §12 names it is, since a key
merely missing would collapse the distinction the header renders four sentences for. It carries the
name the header ranked and never the set of what was true, one absence rendering on the page and one
going out. `last_run` is absent on the first three, one entry supplying both readings — and **present
on `not-in-clone`**, which is the exception the header states above arriving on the wire: the entry was
found and only its bytes were not. **`path` is absent exactly where `baseline_absent` is `built-in`**,
and the two are one fact — an artefact with no file has no path, and no Run could have recorded a
revision of what has none — so the name already on the row is the discriminator and a second key would
carry that fact twice. It is stated here rather than left to fall out of which artefact happens to have
a file: a rule that holds by accident is one no reader can check a rendering against.

**`not-in-clone` is one name in two positions.** The Comparison's catch-all row carries it beside the
`command` it keeps, dropping `count` — `{"type":"code","fact":"other lines changed","baseline_absent":
"not-in-clone","command":"git diff 1f0a3d7 88bc402"}` — rather than changing `fact` or nulling the
count, an omitted `count` being exactly the key merely missing that the paragraph above refuses. A
consumer filtering on the name gets both surfaces, which is `not-run`'s shape one table over: one name
over a fact with more than one site.

A review does not decompose into rows the way the three change tables do, so `review --json` emits the
annotations and never the source — the consumer already has the file. Each `flag` row carries the line
it cites, which makes a flag citing a line no `gutter` row marked a detectable violation rather than a
prose mistake. A flag's name goes on the wire in kebab-case like any other member of a closed set
(§12); the block above renders it upper-case for the eye, exactly as the gutter renders the Kind
`destroy` as `DESTROY`.

**A `gutter` row is one rendered line, not one marked cell.** It carries `line`, the marker column's
rendered text as `marker` where that cell has content, and `changed` where the change column marked the
line; a line with content in either column gets a row and a line with neither gets none. One row per
rendered row is what makes the two surfaces unable to state different things (ADR-0026), so the row
carries the marker as the string the page renders rather than decomposing it into a Kind, a Target and
a Bound — a decomposition is a second rendering of the same fact, and the second one can be wrong about
the first. `changed` is written `true` rather than as the `~` the column draws, the sigil and the
boolean being one fact in the two notations exactly as `envelope ✓` and `"envelope ok"` are; the
revision it is relative to is named in the header and in each flag row's text, never once per touched
line.

```
$ hyper review procedures/retire-preview-envs.yaml --json
{"type":"artefact","kind":"procedure","path":"procedures/retire-preview-envs.yaml","baseline":"a91f0c2d5b83e47196c0af2b1d7e63840f5a92c1","cadence":"0 3 * * 1","phrase":"03:00 UTC every Monday","rate":4.3,"last_run":{"run":"01991c3a-7d40-7a11-9c2e-4f0b8d61a3e7","ended":"2026-07-01T03:01:44Z"}}
{"type":"gutter","line":3,"marker":"envelope ok"}
{"type":"gutter","line":6,"marker":"read local"}
{"type":"gutter","line":15,"marker":"mutate! staging"}
{"type":"gutter","line":24,"marker":"DESTROY staging"}
{"type":"gutter","line":34,"changed":true}
{"type":"authority","definition":"uptime","target":"local","definition_kinds":["read"],"target_kinds":["read"],"effective":["read"],"destroy_operations":[]}
{"type":"authority","definition":"hetzner-staging","target":"staging","definition_kinds":["mutate","destroy"],"target_kinds":["read","mutate","destroy"],"effective":["mutate","destroy"],"destroy_operations":["delete_server"]}
{"type":"flag","flag":"unbounded","cites_line":15,"step":"label"}
{"type":"flag","flag":"destroy","cites_line":24,"step":"retire"}
{"type":"flag","flag":"widened","cites_line":34,"step":"retire"}
{"type":"flag","flag":"envelope","cites_line":3}
{"type":"result","truncated":false}
```

```
$ hyper changes --since 2026-08-04T09:12:00Z --json
{"type":"window","procedure":"retire-preview-envs","baseline":{"run":"01991c3a-7d40-7a11-9c2e-4f0b8d61a3e7","trigger":"cron","started":"2026-08-04T09:12:03Z","outcome":"completed","procedure_revision":"a91f0c2d5b83e47196c0af2b1d7e63840f5a92c1"},"subject":{"run":"01991ea6-b118-7c93-8d41-6b2f7ae05c19","trigger":"igor@thinkpad","started":"2026-08-06T11:03:18Z","outcome":"completed","procedure_revision":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe"}}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8801","from_ordinal":4,"to_ordinal":5,"confirmed_at":"2026-08-06T11:02:41Z","fields":{"region":"fsn1","server_type":"cx22"}}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8802","from_ordinal":3,"to_ordinal":4,"confirmed_at":"2026-08-06T11:02:52Z","fields":{"region":"fsn1","server_type":"cx22"}}
{"type":"asset","change":"destroyed","target":"staging","definition":"hetzner-staging","name":"preview-8806","from_ordinal":7,"to_ordinal":8,"confirmed_at":"2026-08-06T11:03:09Z","fields":{"region":"fsn1","server_type":"cx22"}}
{"type":"asset","change":"changed","target":"staging","definition":"hetzner-staging","name":"preview-8815","from_ordinal":9,"to_ordinal":10,"fields":{"labels.retire-after":["2026-08-18","2026-08-25"]}}
{"type":"asset","change":"created","target":"staging","definition":"hetzner-staging","name":"preview-8821","to_ordinal":1,"fields":{"region":"fsn1","server_type":"cx22"}}
{"type":"observation","change":"changed","target":"local","definition":"uptime","name":"cert.hyper.dev","from_ordinal":22,"to_ordinal":23,"fields":{"days_left":[41,34]}}
{"type":"observation","change":"changed","target":"local","definition":"uptime","name":"status.hyper.dev","from_ordinal":22,"to_ordinal":23,"fields":{"status":[200,503]}}
{"type":"code","subject_kind":"procedure","subject":"retire-preview-envs","fact":"procedure revision","from":"a91f0c2d5b83e47196c0af2b1d7e63840f5a92c1","to":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe"}
{"type":"code","subject_kind":"procedure","subject":"retire-preview-envs","fact":"step retire · bound","from":3,"to":5}
{"type":"code","fact":"repository revision","from":"1f0a3d78c2e5b91467af03d28b5c9e610473fa8d","to":"88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"}
{"type":"code","fact":"other lines changed","count":2,"command":"git diff 1f0a3d7 88bc402"}
{"type":"result","truncated":false}
```

A change row's `fields` carries every value **whole**, including the ones the page rendered `changed`
(ADR-0059), and `from_ordinal` and `to_ordinal` are absent exactly where the column renders `–` — the
absence rule (§7) saying *nothing to name on this side* by writing no key, which is what a `vanished`
row carries in place of a subject ordinal. A `code` row carries `subject_kind` and `subject` where the
fact has an artefact subject and neither where it does not, the pair being the one cell the table
renders `—` in; the catch-all's `command` is likewise absent where either side recorded `repo_dirty`,
and the `window` object carries `repo_dirty: true` on the side that recorded it rather than the `+` the
header draws, one fact in the two notations exactly as `changed` and `~` are above.

A `code` row whose fact is `cadence` carries `from_phrase` and `to_phrase` beside its two expressions,
and `from_rate` and `to_rate` as numbers rounded exactly as `rate` is — the parts and never the composed
cell, which is the `artefact` row's rule above and the rule a `flag` row's glossed before-and-after
follows too.

```
$ hyper run retire-preview-envs --json
{"type":"step","index":1,"id":"probe","kind":"read","disposition":"ran","records":12}
{"type":"step","index":2,"id":"label","kind":"mutate","disposition":"ran","records":8}
{"type":"step","index":3,"id":"retire","kind":"destroy","disposition":"refused"}
{"type":"refusal","error_code":"bound-exceeded","step":3,"step_id":"retire","operation":"delete_server","target":"staging","declared":5,"observed":23,"file":"procedures/retire-preview-envs.yaml","line":34,"field":"steps[2].bound","message":"expansion resolved 23 assets on staging","resolved":{"14d":"2026-07-23T11:03:18Z"}}
{"type":"remediation","file":"procedures/retire-preview-envs.yaml","line":34,"field":"steps[2].bound","from":5,"to":23}
{"type":"remediation","file":"procedures/retire-preview-envs.yaml","line":33,"field":"steps[2].over","hint":"narrow the selector","example_expansion":4,"resolved":{"30d":"2026-07-07T11:03:18Z"}}
{"type":"provenance","procedure_revision":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe","repo_revision":"88bc402f71d3e6a95c0428be1f7d3a09c5e64b12","hyper_version":"1.4.0"}
{"type":"provenance","step":1,"definition_revision":"c3a17b09e4d582f61b7c0a94d3e85f207c6b1d43","manifest_digest":"sha256:2b7e5c81f0a394d6e2b7c051a83f6d940e17c5b28a306f4d91e75b0c2f8a63d1"}
{"type":"provenance","step":2,"definition_revision":"4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a","manifest_digest":"sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1","origin_digest":"sha256:e40a91c73f5b28d604e1a7c95b038f26d417ea5390c62b81f7d045a3e926c8b7"}
{"type":"provenance","step":3,"definition_revision":"4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a","manifest_digest":"sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1","origin_digest":"sha256:e40a91c73f5b28d604e1a7c95b038f26d417ea5390c62b81f7d045a3e926c8b7"}
{"type":"outcome","outcome":"refused","code":77,"error_code":"bound-exceeded","dry_run":false,"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}
```

**One `refusal` row per problem**, in the array's order (§7). Where a Refusal carries five, five rows go
out and the single `outcome` row's `error_code` is the first one's. The stream does not nest: one row
carrying an array would make this the one surface that does, and a consumer's
`select(.type=="refusal")` would stop returning one problem per line. The array exists in the Store
because a file is not a stream — the same reason `expanded_to` is a list in the entry and a count in the
column above. A row carries the same members its Store counterpart does, plus the `operation` and
`target` the Step it cites was bound to; a Refusal that reached no Step carries neither, and carries no
`step` row before it, there being no Step table on that page.

A `refusal` and a `remediation` row each carry `resolved` where the text they render holds a relative
predicate, mapping the operand to the instant it resolved to — the `=` note and the trailing cell in the
two notations, and absent where the page rendered no such operand. It is derived from `run.json`'s
`started_at` and stored nowhere (ADR-0034), which is why it rides on the rows that render it rather than
on the `provenance` row beside them.

```
$ hyper run retire-preview-envs --json
{"type":"refusal","error_code":"credential-absent","file":"targets/staging.yaml","line":12,"field":"auth.token","message":"the environment does not hold STAGING_TOKEN"}
{"type":"refusal","error_code":"credential-absent","file":"targets/cloudflare-prod.yaml","line":9,"field":"auth.token","message":"the environment does not hold CLOUDFLARE_API_TOKEN"}
{"type":"provenance","procedure_revision":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe","repo_revision":"88bc402f71d3e6a95c0428be1f7d3a09c5e64b12","hyper_version":"1.4.0"}
{"type":"outcome","outcome":"refused","code":77,"error_code":"credential-absent","dry_run":false,"run_id":"0199206d-4e15-7c30-9b8a-52d9ea01f7b4"}
```

A `step` row's `records` is the count the column renders, and it is absent where the column renders `–`:
a Disposition carrying no set has no number to report, and §7's absence rule says so by writing nothing
rather than `0`, which is the value the *ran* Step with an empty set carries. Where the column renders
`n of m` the row carries `expanded` beside it — `m`, and only where it differs from `records`. The row
exists for a Step that was never reached like any other, that Step having a cell on the page.

The same count reaches `run_show` as the identities themselves, under the same key: `records` is a
number where a Run reports on itself and the members where an entry is read back (§9), one set in the
two shapes the two surfaces are for.

Provenance splits on the wire exactly as it splits in the Store (§7): one `provenance` row carrying the
Run-wide members and one per Step file written, distinguished by `step` the way §7 distinguishes the two
files themselves. A member with no value at a level is absent rather than `null` — `origin_digest` above
on the Step whose Provider is locally authored — which is §7's absence rule and not a rendering of its
own. There is no `provenance` row for a Step that was never reached, that Disposition writing no file to
render.

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

A Run halted by a status is rendered the same way and for the same reason: it names the host it reached
and the status it got, carries no `error_code` since nothing declined, and goes to stderr as an error's
rendering rather than into the row stream (§6, ADR-0050). Two facts are the whole of it because two
facts are the whole of what `hyper` may show — the response behind them is a credentialled Target's and
no surface renders one. On a `read` there is nothing here to render at all: no status halts one, and the
status is in the Record wherever the Manifest projected it, which is a row in `THE WORLD MOVED` rather
than an error.

## What these surfaces do not say

The Comparison prevents nothing. It is an accountability instrument and never a guardrail, and it
reports what changed rather than what is wrong (ADR-0010). It never says *this differs from what we
intended*, only *this differs from when we last looked*.

There is no rendering before a destructive Step. What stands in its place is the Definition review above
and the Bound (§5), both acting on the authored claim rather than on the world.

Three tables buy the actor split at the cost of a single chronological scan: there is no one list of
everything that happened, in order. Named here as the accepted cost it is, and carried forward to §13
with the limit above it.
