# A Requirement roots at any projected field, and the value goes on the line

**The orientation now says what a `require:` may root at, and that where the fact under review has a
value the value belongs on the `require:` line.** The worked example is unchanged — `sha256sum -c`
says its verdict in its status, its `stdout` being a per-file report rather than a value, and an
example comparing a value would stop teaching that an exit code is a field like any other. What was
missing was never the example; it was the sentence beside it saying there was a choice.

Nothing in the binary moves. `check` accepted both spellings before this and accepts both after, both
Steps are `OPAQUE` either way, and the Run halts identically. What moves is whether the predicate a
reviewer is being asked to approve is stated on the `require:` line, or written into a quoted command
argument that line says nothing about.

## What was wrong (issue #242)

§8's whole design is that **the artefact is the review surface and the gutter annotates what is in
this file**. A Requirement is the purest case of it: the line carries no marker at all, because there
is no derived fact to hold — *its whole content is on the line being read: an `id:`, a Step of this
same file, one field name and one operator*.

**A predicate hidden in a shell string defeats that exactly where it matters most**, on the artefact
whose entire job is to stop the Run. And the orientation's only worked `require:` gates on
`exit_code`, so an agent that transcribes it — which is what a worked example is for — writes the
spelling that hides the fact, at no cost in calls and with nothing in the transcript that looks like a
problem.

This is the other failure mode of the channel
[ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)
measured and
[ADR-0117](0117-a-closed-key-set-is-stated-as-closed-and-an-invocation-is-a-fixed-block.md) repaired.
There an undocumented corner cost about half a session in Refusals (issue #237). Here a documented
corner teaches the weaker of two available spellings and costs nothing at all, which is why it needed a
transcript to find.

## The evidence: three pairs, and a wall stated nowhere

The sealed acceptance run of 2026-08-31 on `change-window`, read in
[ADR-0120](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md)
(issues #238, #240). `change-window`'s wall is three files whose contents are the facts:
`control/window` reads `open`, `control/freeze` must be empty, `control/approver` names somebody. Each
is a value a `require:` could have compared. The session followed the example three times:

```yaml
  - id: read-window
    definition: change-control
    operation: read
    target: control
    args:
      command: [sh, -c, 'test "$(cat control/window)" = open']

  - id: window-open
    require: {step: read-window, field: exit_code, equals: 0}
```

**The Requirement itself is the run's strongest positive result** and is not what this record is
about: the shape was authored right first time, from `AGENTS.md` alone, and the session named and
declined the effectful-halt workaround it replaced (ADR-0116, ADR-0120). What `review` renders of it is
three `OPAQUE` flags and three `require:` lines saying an exit code was zero:

```
  read  opaque  control  │   - id: read-window
                         │     definition: change-control
                         │     operation: read
                         │     target: control
                         │     args:
                         │       command: [sh, -c, 'test "$(cat control/window)" = open']
                         │
                         │   - id: window-open
                         │     require: {step: read-window, field: exit_code, equals: 0}

  OPAQUE  line 5  step read-window  read reaches an effect hyper cannot describe
```

**The predicate under review is `= open`, and it is inside the command string.** Nothing in the
gutter, the `FLAGS` block or the `require:` line carries it. A reviewer reading the artefact as the
review presents it learns that a check ran and that its exit code had to be zero.

## The stronger spelling works, and costs nothing

Against the same fixture, `check` clean over eleven artefacts and the Run completing:

```yaml
  - id: read-window
    definition: change-control
    operation: read
    target: control
    args:
      command: [cat, control/window]

  - id: window-open
    require: {step: read-window, field: stdout, equals: "open\n"}
```

and `review` renders the predicate where a diff can carry it:

```
  read  opaque  control  │   - id: read-window
                         │     definition: change-control
                         │     operation: read
                         │     target: control
                         │     args:
                         │       command: [cat, control/window]
                         │
                         │   - id: window-open
                         │     require: {step: read-window, field: stdout, equals: "open\n"}

  OPAQUE  line 5  step read-window  read reaches an effect hyper cannot describe
```

The Step is still `OPAQUE` — a `cat` is as opaque as a `test`, and nothing here is about the shell
Provider's opacity — but the fact under review is now on a line the rendering shows verbatim, and a
change to it is a change to the diff rather than to a quoted string inside one.

**Neither spelling is refusable and neither should be.** Both are well-formed, both halt on the same
world, and `hyper` cannot tell that `test "$(cat …)" = open` and `cat …` plus `equals: "open\n"` are
two spellings of one intent — that is exactly what an Opaque request means. So this is an authoring
fact, and the only place an authoring fact reaches an agent is the orientation.

## The predicate set was invisible too

A `require:` roots at any field the earlier Step **projected**, and the set of operators is eleven:
`equals`, `not_equals`, `in`, `exists`, `absent`, `starts_with`, `ends_with`, `greater_than`,
`less_than`, `older_than`, `newer_than` (§12). The orientation showed one operator against one field,
and [ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)'s
session found the list by probing `check` until it enumerated it in a Refusal.

**The list is not what was added.** `check` answers it — `exactly one of equals, not_equals, … must be
present` — to anyone who writes a predicate carrying none, and the orientation's own budget rule is
that anything a command already answers stays one (ADR-0093, ADR-0096). What no Refusal answers is the
*root*: an author who writes `field: exit_code` and a working operator never learns there was another
field, because nothing declines. That is the sentence, and the operator set is left where it is.

## What was decided, and against what

**A sentence beside the example.** The orientation gains one paragraph stating that a `require:` roots
at any field the Step it names projected — the field's own name, never a `$.` path, and never one the
Manifest declares `secret:` — and that where the fact under review has a value, the value is what goes
on the line, with the two spellings of `control/window` set beside each other and `review`'s treatment
of each named.

**The `secret:` clause is there because the rule is stated in that text.**
[ADR-0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)'s rule is that a rule
stated in the orientation is stated with its exception or not stated, and *any projected field* has
exactly one: a field declared `secret:` reaches the Store as a constant, and a predicate over one is
`predicate-type-mismatch` wherever it stands (§5, ADR-0035). Four words, and the alternative is a
sentence that promises a spelling `check` refuses — which is the Bound sentence's own fault reproduced
one paragraph down.

**Changing the example was rejected.** It is the cheaper edit and the more expensive one to get right.
`sha256sum -c` is in the example precisely *because* it is an exit-code verdict. A shell `read`
projects `stdout` and `stderr` beside `exit_code` on every Step, this one included — what
`sha256sum -c` writes there is a per-file report, and the answer a check wants out of it is the
status. An example that compared a value would teach the better spelling and stop teaching that an
exit code is a field like any other, which is the fact that makes the honest cases honest. The example
is also the fragment `TestInstructions_TheSharedCheckItTeachesIsOneCheckAccepts` writes into a
repository and checks, and its virtue there is that it is the smallest legible shared check.

**Saying nothing was rejected.** The ticket offers it with a real argument: `shell` Steps are `OPAQUE`
either way and a reviewer must read the command line regardless. But *must read the command line* is
what a reviewer does when there is no alternative, and here there is one — the same halt, the same
`check`, the predicate on a rendered line. A surface that offers two spellings and teaches the weaker
one is not neutral about which gets written; ADR-0120 is the measurement of that, three pairs in one
session.

**Nothing in the binary changed.** No new code, no new flag, no new rendering. This is a documentation
decision, and the one thing that makes it enforceable is that the claim is held to `check` rather than
to a reader.

## The seam the fence is expressed at

`TestInstructions_TheFieldARequirementRootsAtIsTheOneCheckHolds`, in `internal/cli`, takes the one
sentence of the orientation that says what a `require:` roots at and drives the same repository four
times: the value-comparing spelling checks clean, a field no Operation projects is
`reference-unresolvable`, the field written as a path is `schema-mismatch`, and a projected field
declared `secret:` is `predicate-type-mismatch`. The last of the four is why the repository carries a
second Provider — the built-in `shell` one projects no secret, so the case the exception exists for
cannot be written against it. Each answer `check` declines is paired with the words the sentence must
carry for an author not to have written it, which
is the pairing
[ADR-0117](0117-a-closed-key-set-is-stated-as-closed-and-an-invocation-is-a-fixed-block.md)'s case
established: a rule that moves in the binary and not in the text fails there, and so does a sentence
promising a spelling the checker declines.

`TestInstructions_SayWhatARequirementRootsAtAndWhereThePredicateGoes`, in `internal/mcp`, is the claim
on §9's list — that the text states it at all — on the footing every other orientation fact stands on.

Both stand beside `TestInstructions_TheSharedCheckItTeachesIsOneCheckAccepts`, whose
`taughtRequirement` still asserts **exactly one** fenced block writing a `require:`. That is why the
second spelling is prose with the YAML inline rather than a second fenced block: a second one is a
manual growing inside a text paid for on every session in every harness.

## Consequences

- **The orientation is one paragraph longer**, which is the budget ADR-0093 fixes being spent rather
  than an omission being repaired for free. What it buys is the one authoring choice on the artefact
  the review surface exists for, and it is spent on the axis §9's exception rule already licenses: a
  rule stated here is stated with what it costs to get wrong.
- **The example is untouched**, so `TestInstructions_TheSharedCheckItTeachesIsOneCheckAccepts` and
  `taughtRequirement` are untouched, and the shape an agent transcribes is the shape it was.
- **§5 carries the argument and §9 carries the decision.** §5 says the predicate is where the
  reviewable fact belongs and why that is the same fault as the authority one beside it; §9 says the
  text states it, why the example stays, and which case holds it. §3, §6, §8 and §12 are unchanged —
  every rule this states was already stated there, which is the point: the gap was in the one text
  that is read.
- **`check`, the schemas, the renderings and the record are untouched.** Both spellings were legal
  before and are legal now.
- **What this does not fix is an author who has already written the weaker one.** Nothing declines it
  and no flag marks it; the two artefacts differ only in where the same fact is written, which is why
  this is a sentence in the orientation and not a code in §4. The next sealed run against a fixture
  whose wall has values in it is what says whether it landed.
