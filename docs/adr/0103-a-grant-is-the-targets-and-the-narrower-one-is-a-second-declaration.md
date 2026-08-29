# A grant is the Target's, and the narrower one is a second declaration

**`opaque-destroy:` stays a boolean on the Target declaration.** It is a property of the Target and
never of the `(Definition, Target)` pairing that took it, so a declaration carrying it admits an
`opaque` `destroy` from every Definition bound there. What narrows it is a second class-local
declaration, which the model already admits and which nothing said where an author would meet it —
so `opaque-destroy-not-granted` now states both edits and what the wider one costs.

Nothing in the checker moved. No schema changed, no check fires on anything new, and §12's closed
set holds the same members. What changed is a wall that named one edit where the model has two.

## The question, and why it was a question (issue #220)

`opaque-destroy:` is a bare boolean, read as `== "true"` and checked per `(Definition, Target)` pair
as *has this Target opted in at all*. A repository that needs one Definition to run an opaque
`destroy` against `local` opts `local` in — and from then on any Definition bound to `local` may
claim an opaque `destroy` with no further edit to `targets/local.yaml`.

Against the sentence the model is built on — *authority is authored, and widening is a visible edit*
— that reads as one visible edit where the wider claim is the second one. §3's own severity rule
sharpens it: `kinds:` for `read` and `mutate`, **named Operations** for `destroy`, a granularity that
gets finer as severity rises. The most severe thing the tool does is then held at the coarsest grain
in the model, on the one Capability whose reach no grant bounds.

That is the strongest form of the objection and it is worth stating in full, because the answer is
not that the grain is unimportant. It is that the grain is the Target.

## Why the grain is right

**The widening *is* a visible edit, on the end §3 puts it.** A Definition claiming an opaque
`destroy` names the Operation, names the Target, renders `DESTROY` and `OPAQUE` in its own gutter and
appears in `AUTHORITY` (§8). Authority is one relation with two ends, and both are reviewed artefacts
in the same repository under the same review — there is no unedited widening, only a widening whose
edit is in `definitions/`.

**The Target's own review already answers *who took this grant*.** A Target declaration renders a row
for every Definition that claims it, discovered across `definitions/` rather than authored
(ADR-0069), and that rendering exists for precisely this question: the screen whose gutter marks what
it grants and where nothing else says who took it. `hyper review targets/local.yaml` lists every
claimant beside its `DESTROY OPS`, under a `FLAGS` row reading *an opaque destroy admitted*. The
sealed run of 2026-08-29 read exactly that and said so.

**Every authority fact about a pairing is derived from its two ends, never authored at one.**
`AUTHORITY`'s `EFFECTIVE` column is an intersection; §5's two keys are an intersection; a Step names
both ends and grants neither. No artefact in the model states a grant over a pair. A
`opaque-destroy: [host-ops]` list would be the first, authored at the end that names no Definition
anywhere else: a Target declares `kinds:`, `capabilities:`, `hosts:` and
a `class:` — *what*, never *who*. It would put a name in `targets/` that must resolve into
`definitions/`, a lookup direction no artefact takes, and leave a grant naming nothing whenever a
Definition is renamed or deleted (ADR-0012).

**And it would buy containment only where the two artefacts have different reviewers.** Nothing in
`hyper` gives them different reviewers: both are files in one repository, reviewed as code, and
`hyper` holds no reviewer model of its own and no approval step (ADR-0001, ADR-0015). A repository
that wants `targets/` reviewed by different eyes than `definitions/` has that in the review system it
already uses; a per-claimant list inside the artefact would look like that guarantee without being
it, which is the failure mode this model is built to avoid.

**The narrower grant already exists, in the model's own currency.** More than one declaration may
claim `class: local` — two names for the machine `hyper` runs on, each with its own accepted Kinds,
its own credential slots and its own `opaque-destroy:` (§3, ADR-0041). One opting in and one not
confines command-`destroy` authority to the Definitions that bind the first. It is the same plurality
§13 uses to buy back the credential-slot removal that a `local`-named declaration cannot make, and it
costs a reviewer nothing new to read: one more file in `targets/`, with one more `AUTHORITY` table
saying who binds it.

## What was actually wrong

The wall, not the schema. `check` answered:

> `local` has not opted into `opaque-destroy:`, and this Definition claims an opaque destroy Operation

which is true, and names the one edit that widens the grant to every Definition on that Target. The
other edit is written in an ADR, in §3's `local` paragraph and in §13's environment paragraph — none
of which is on the machine an agent authors on, there being no `docs/spec/` there (ADR-0099).

The evidence is the sealed acceptance run of 2026-08-29 (ADR-0099, ADR-0100). It needed an opaque
`destroy` for one Definition, met this row, reached for `opaque-destroy: [snapshots]`, was answered
`schema-mismatch`, and declined to ship rather than hand back a containment guarantee it could not
scope:

> It's a blanket grant. The Target review now lists `host-ops` alongside `snapshots`, and any future
> Definition could claim an opaque destroy against `local` without touching that file again.

Every sentence of that is correct, and the narrower spelling it wanted was in the model the whole
time. The row is where an author meets this fact, and it is the surface that can carry the second
edit at the moment it is wanted: a `FLAGS` row may not, its caption promising that no flag states
anything the gutter does not, and the extent of a grant is not on the line the opt-in is written on.

So the message states both edits, and says which of them is wider:

> `local` has not opted into `opaque-destroy:`, and this Definition claims an opaque destroy
> Operation — the opt-in is the Target's and admits one from every Definition bound to `local`; a
> narrower grant is a second class-local declaration, bound here instead

*bound here instead* points at the `targets:` member the row already cites, which is the line whose
edit fixes it either way.

## Considered options

- **Close it and change nothing.** `wontfix` was a live answer and the design is argued for it. It is
  rejected on the transcript: an agent met the wall, reached for a spelling that does not exist, and
  never found the one that does. A model whose narrowing is unreachable from the surface that refuses
  is, in practice, the blanket grant the ticket described.
- **`opaque-destroy:` takes a list of Definition names.** What the run reached for, and the only
  option that makes the second claimant's arrival an edit to the granting file. Rejected on the two
  grounds above — the first authored per-pair fact in the model, authored at the end that names no
  Definitions — and on a third: it makes `AUTHORITY` no longer readable as an intersection, the table
  having a fact on one end that the other end cannot see.
- **`opaque-destroy:` takes a list of Operation names**, mirroring a Definition's own `destroy:` and
  §3's severity rule exactly. Rejected because Operation names live in a Provider's namespace and a
  Target reads none — it names a `class:`, never a Provider, and two Providers may both declare
  `destroy`. What it would narrow is nothing anyway: the built-in `shell` Provider's two opaque
  `destroy` Operations differ in Repeatability and not in what they may destroy.
- **Render the extent on the Target's review.** Rejected twice. `FLAGS` may not carry it (§8's own
  caption), and marking *which* claimants took the opaque grant in `AUTHORITY` needs the
  Definition→Provider hop, putting a third namespace into a table ADR-0069 fixed at two ends — to say
  what the row above it already says, on a screen the author who is being refused is not looking at.
- **Teach the opt-in in the orientation.** ADR-0101 rejected this as out of scope on the ground that
  its own message says what it wants. That holds harder now: the message says what it wants *and*
  what it costs, and the orientation's length is paid on every session in every harness.

## The fences

`TestCheckDefinition_OpaqueDestroyNotGrantedNamesTheGrantsExtent` holds the row to the two facts
beyond the fault — the extent, and the narrower spelling in the vocabulary the corpus already uses
for it (§4, §12, §13, ADR-0041).

`TestCheckDefinition_ASecondClassLocalDeclarationConfinesTheOptIn` is what holds that message honest.
Two class-local declarations, one opted in: the Definition binding it is clean, and the Definition
binding the other is refused **with the opted-in declaration sitting in the same namespace**. A
message advertising a narrowing the checker had stopped performing would fail here rather than in a
transcript a year later.

`check/opaque-destroy-confined-by-a-second-declaration` is that shape as a page, and it is the worked
example the corpus lacked: `host-ops` binds `local-teardown` and checks clean, `snapshots` binds
`local` and is refused, one row, exit `1`. The two declarations differ in the opt-in line and in
nothing else, so the page reads as the opt-in confining it rather than as two unlike grants; and the
Step that uses the confined grant is in the fixture, as `opaque-destroy-clean` carries one, because a
narrowing nothing runs through is a narrowing nobody has checked is usable.

## Consequences

- **The grant's extent is written down for the first time.** It was true, checked and unstated: §5
  said the Target's declaration must opt in and stopped. It now says what opting in admits, what the
  grain is, and what narrows it — one paragraph in the chapter that already owns the opt-in.
- **The message is among the longest `check` writes**, and the row it stands beside is the precedent
  rather than the excuse: the one for a `values:` list whose members all resolve to one Record
  identity ends *wire the member into the input `identity:` reads, or write the calls out as Steps*.
  That is the other row whose remedy has two shapes, and it names both.
- **`error-code-coverage`'s golden moves and nothing else does.** No `error_code` is added or
  removed, no check changes what it fires on, and `opaque-destroy-clean` is byte-identical — the
  repository it holds is the one this decision says is correct.
- **The credential half is untouched and its grain is the same one.** §5's second opt-in is per
  Target too, resolved at Run start, and no static check performs it. Nothing here changes it, and
  the same argument would apply to it if anything did.
- **ADR-0041's consequence is now the answer to a ticket.** *The `opaque-destroy:` opt-in is no longer
  one switch over every command* was written as a consequence of authoring `local` in the repository;
  it is the whole of the narrowing this decision points authors at, and it took an agent failing to
  find it for that to be worth stating twice.
- **The next transcript is what says whether this worked**, as it was for ADR-0101. A run that needs
  a scoped opaque `destroy` either finds the second declaration from the row that refused it, or does
  not.
