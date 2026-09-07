# A `fields:` value is a path, and a sink nothing can fill is a Refusal

Two checks, one loss. `check` refuses a `record: fields:` value that is not a scalar, under
`schema-mismatch`, and names the Operation-level `secret:` spelling in the message where the value
carries that key. A Run given a Secret sink that reaches no Step declaring secret output Refuses before
Step 1, under a new member of §12's closed set, `secret-sink-unfilled`. Issue #275.

This is the repair
[ADR-0149](0149-the-sink-was-supplied-both-secrets-were-lost-and-check-was-clean-throughout.md)
deferred in the sentence that recorded the defect: *what to do about the Open position and the absent
converse gate has options worth weighing […] it is recorded below and taken by a ticket*. The ticket
put three on the table and called two of them belt and braces. Both are taken, and the third is folded
into the first as its message rather than standing as a rule of its own.

## What the state was

A `secret` key written inside a Manifest's `record: fields:` was accepted by `check`, dropped by the
projection, and produced a Run that **declared no secret output**. So the sink gate never fired, a
supplied `--secret-out` was ignored, the sink directory was never created, and the value was destroyed
at exit `0` while the Record carried no field at all — not `<secret>`, absent.

It reproduced in two commands against an ordinary Operation:

```
$ hyper check providers/repro.yaml
checked 12 artefacts: no problems found

$ hyper probe repro issue --response r.json
FIELD      VALUE
identity:  ledger
id         cred_x
```

The token was in the supplied response and not in the projection, and neither surface said a word.

**Both halves were individually right, which is why it stood.** `fields:` is an `Open` object, so the
schema declined to have an opinion about a mapping-valued field; and the projection reader takes
scalars and skips the rest, because *it judges nothing and drops what it cannot read […] what is wrong
with a Manifest is `check`'s to report and never a reader's to guess at*
([ADR-0064](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md)). The
division of labour is correct. The hole was that the other half never looked.

**The sealed session's own experiment is the sharpest statement of the shape.** Six wrong spellings
through `check`, splitting by **position** rather than by shape: the two written inside `fields:` passed
clean and dropped the token, and the four written anywhere else were caught with a position named —
`unknown-key` at `record.secret`, `schema-mismatch` at an Operation-level `secret:` that is not an
array. `check` was not weak in general. It was blind at exactly one position, and that position is
where an author reaching for *this field is secret* naturally writes it.

## Why the check is written against the value and the message against the key

The ticket's option 1 refuses a non-scalar `fields:` value; its option 2 refuses a `secret:` key inside
`fields:` specifically, with the better message. **The rule is option 1 and the message is option 2**,
and they are not in competition once the rule is stated against the value.

**§3 already said it.** *`fields:` values stay uniformly scalar, so a mapping in that position keeps
meaning a reference and nothing else* ([ADR-0022](0022-the-authoring-format-has-no-expression-language.md)).
That sentence has stood since §3 was written and nothing enforced it. So this is not a new constraint
on authors; it is §4 finally holding a rule §3 states, which is exactly what §4 is for — *this chapter
is where each rejection gets the name it is refused under*.

**A rule written against the key would fix one door and leave the position open.** `secret:` is the
spelling an author reaches for today; the next one is `sensitive:`, or `redact:`, or a mapping carrying
a path and a comment. Every one of them would check clean, drop the field, and destroy the value by the
same three steps. The value-shaped rule closes the class.

**The key still earns the message, and that is where option 2's whole value is.** A `schema-mismatch`
reading *expected a scalar* is correct and costs the reader the thing the sealed session spent thirteen
calls against `strings` and `nm` finding. So a mapping carrying `secret` is told where the declaration
goes instead:

```
FILE                    LINE  FIELD                                            ERROR_CODE       MESSAGE
providers/lookout.yaml  24    operations.issue_credential.record.fields.token  schema-mismatch  secret: is declared beside record: as secret: [token] and never inside fields: — a marking written here is dropped by the projection and the value is destroyed
```

That is a message about the key and a check about the value, which is the arrangement that survives the
next spelling: a mapping naming no `secret` gets the plain sentence and is refused just as hard.

**It mints no `error_code`.** `schema-mismatch` is *a value that does not satisfy the schema at its
position*, and a `fields:` value that is not a path is precisely that. A member of its own would have
been a second name for the check §4 already states, and §12's set is kept small by refusing exactly
that (§4, [ADR-0081](0081-a-value-is-read-against-the-schema-at-its-position.md)).

## Why the converse gate is worth a code, and what it cites

`internal/run/gates.go` opened the sink gate with *if a sink was named, there is nothing to refuse*, and
returned from the whole block where no Step declared secret output. `internal/run/sink.go` stated the
intent plainly: *a Run that reaches no Step declaring secret output never calls it, so a sink named
against a Procedure that produces none leaves no empty directory behind.*

**That reasoning is sound and it is what made this loss invisible.** `hyper` held both facts at run
start — a sink was named, nothing produces a secret — and the tidiness argument for silence was written
before there was a reason to speak. Either half of this record alone would have caught the sealed run's
mistake before its first call.

**It is a code of its own on the set's own test.** A reader handed `secret-sink-absent` for a sink they
named goes looking for the flag they already typed and is out of moves — the same test that split
`credential-absent` from `credential-empty`
([ADR-0145](0145-an-empty-credential-is-its-own-refusal-on-both-surfaces.md)). It is the **occasion's**
supply like the other three §9 contributes, and it is checked at Run start beside them.

**It cites the Procedure the invocation named, at the line naming it, and no Step.** Every Step of such
a Run is correct as authored: none declares secret output and none was asked to. A citation on one
would send a reader to edit the one thing that is not wrong. What the two operands are is an invocation
and a Procedure, and the Procedure is the half that has a file — the line the operator's own argument
resolved to.

```
refused: secret-sink-unfilled

  procedures/watch-two.yaml:2
    1 │ kind: procedure
    2 │ procedure: watch-two
      │            ^ this Run was given a Secret sink and reaches no Step whose Operation declares secret: output — either the sink is a flag this invocation did not need, or an Operation is missing the secret: that would fill it
      │
      = checked at run start, before the first step
      = declare secret: [<field>] beside the Operation's record:, or the same command again without --secret-out
```

**Its remedy names an artefact edit, and it is the only note in §8's not-an-edit set that does.**
`hyper` holds both operands and
cannot say which half is wrong. One reading is an operator who named a sink this Procedure never needed,
whose way past is the invocation. The other is an author who meant a value to be kept and wrote the
declaration where the projection does not read it, whose way past is the Manifest. **A note offering
only the invocation would send that author back to the loss** — the same Run again, without the flag,
completing at exit `0` with the value destroyed — so the edit is named first and the flag second. The
asymmetry is deliberate: being told to fix a Manifest that is already right costs a reader a moment,
and being told to drop a flag that was the only thing standing between them and a lost credential costs
them the credential.

It renders no `EDIT ONE OF` all the same, and by the table's own rule rather than as an exception to
it: the edit it names is a key the Manifest does not carry, so there is no line to point at, and a row
on the Procedure would name the one file that is not the fault.

**It carries no `--dry-run` exemption**, on `secret-sink-absent`'s own ground: a rehearsal performs the
reads it reaches, so whether a Run would produce a secret is the same question under either.

## Why the corpus could not hold this before

**Every golden was downstream of a Manifest the corpus itself authored correctly.** `sink.golden` is
the one golden that can tell a landed secret from a lost one
([ADR-0148](0148-a-secret-sink-is-a-directory-hyper-makes-and-one-file-holds-one-value.md)), and it can
only do so for a Manifest that declares one. A case for this defect has to author the **wrong** Manifest
and assert on the silence, which is a shape the corpus did not have. It has one now, under `check/`
where the wrong Manifest belongs, and the run corpus gains the gate's other direction.

## The blast radius, which belongs in the record

`run-once` is the right rule and it is what made this loss permanent. The Store correctly recorded that
the effect happened; that the Run discarded the effect's product is not a fact the Store has, the
projection having never produced one. In the sealed run, recovery cost two new Steps, minted two fresh
credentials, and left two inert ones standing on a service whose API has no revoke route — only a
delete that would take the monitor with it. Neither half of this record makes that recoverable. Both
make it unreachable.

## What was considered

- **Only the `check` half.** Rejected as the whole answer. It closes the door the sealed run walked
  through and it closes it at authoring time, which is the earliest and best place — but it is a rule
  about one position in one artefact, and the loss it prevents has a shape wider than its cause: any
  future route to *a sink was named and nothing will fill it* is caught by the gate and by nothing else.
- **Only the converse gate.** Rejected for the mirror reason. It catches the class and it catches it
  late — at a Run, on the operator's clock, after review — where the `fields:` rule catches this
  instance in `check`, which is where an agent authoring a Manifest is already looking.
- **Refusing a named sink no Step can fill *at `check`*.** Rejected: `check` reads artefacts and a sink
  is the invocation's, so there is nothing on any page to refuse. The nearest thing that is authored is
  `cadence-secret-output`, which §4 already refuses for exactly this reason — a recurrence whose every
  occurrence would Refuse, the projected workflow supplying no sink (ADR-0077).
- **Making the converse a widening of `secret-sink-absent`.** Rejected on §8's rule that one code holds
  one remedy: the two remedies are opposite acts, and a single code could only ever have offered the
  wrong one to whichever half it was not written for (ADR-0145).
- **Refusing the whole non-scalar rule and reporting only `secret:`.** Rejected above: it is the option
  that fixes the door and not the class.
- **A `manifest-inconsistent` for the `secret:`-inside-`fields:` spelling.** Rejected. That code is *a
  Manifest disagreeing with itself* — a `pagination` outside an `over:`, a `record:` on a `destroy` —
  and this Manifest does not disagree with itself. It writes a value the grammar at that position does
  not admit, which is `schema-mismatch` and is the same fault as writing a mapping where a duration goes.

## Consequences

- **`error_code` grows by one, to fifty-three.** `secret-sink-unfilled` joins §9's contributions, which
  are now four rather than three: three the occasion supplies and one the occasion supplied and should
  not have. The `check` half mints nothing.
- **§3's sentence about `fields:` is enforced for the first time.** *Values stay uniformly scalar* has
  been true of every Manifest anyone wrote and false of the checker since milestone 1. Nothing in the
  corpus moved when it landed, which is the fence the rule wanted: every Manifest `hyper` ships and
  every fixture the corpus holds was already right.
- **The sink gate is symmetric and is stated as one gate with two operands.** §6's order, §9's
  `--secret-out` section and §12's set all say so in those terms, so a reader meeting one direction
  finds the other beside it rather than finding half a rule.
- **The corpus gains two cases and loses none.** `check/a-secret-declared-inside-fields` is the wrong
  Manifest beside the right one — one Operation carrying `token: {path: $…, secret: true}` and its
  sibling carrying `secret: [token]`, so the golden shows the refusal next to the spelling that works.
  `run/a-sink-no-step-can-fill` names `--secret-out` over `repo-two-reads`, two ordinary `read` Steps
  and no secret anywhere, and is the one Refusal in the corpus whose `=` remedy names an edit and an
  invocation both.
- **The `run` tool's `secret_sink` description carries both halves of the gate and no more.** It says
  the argument earns a Refusal in either direction, and it deliberately does **not** say where `secret:`
  is written. That is #276's subject and a repair across four surfaces; putting a clause about it in one
  argument's description would be doing a fraction of that ticket badly, and would spend the deferral
  argument below on the same breath that makes it.
- **`probe` is unchanged and is still the technique that found this.** The sealed session's own summary
  — *I found it by testing candidates against `probe`, which withholds the response only when the
  declaration is genuinely honoured* — is `probe` earning its keep ([ADR-0009](0009-a-probe-is-not-a-run.md)).
  What this record removes is the need to reach for it: `check` now answers the same question at the
  position the author is standing in.
- **The teaching gap is not closed here and is not this record's.** `secret:` is named by no surface an
  agent is handed — not the orientation, not `operation`, not `provider`, not `review` — and that is
  issue #276. What lands here is a message an author meets *after* writing the wrong thing; #276 is
  about their not writing it. The two are complementary and neither is the other's substitute.
- **The repair is both enforced and taught, so it is taught, and the run it owes is deferred in
  writing.** Per `docs/agents/acceptance-re-runs.md` the enforcement is fenced — two corpus cases, the
  package tests over the gate's four combinations, and the closed-set coverage test — and the teaching
  is not: whether an author handed *secret: is declared beside record: as `secret: [token]`* then writes
  it correctly is a thing only a transcript can say. The run it owes is `push-credential` (#271), the
  task whose transcript produced the defect. **It is not being bought here.** The surface an agent
  reads is about to move again under #276, and a run bought now would measure a surface that is one
  ticket from changing — which is the same reason ADR-0150's run was deferred, and the second deferred
  repair now standing against that task. Two is the argument for buying it next.
