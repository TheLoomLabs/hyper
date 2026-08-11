# Every Run is a Run of a Procedure

`hyper run` takes the name of a Procedure and nothing else. There is no way to invoke a single Operation
directly through a Definition, and a Run therefore always has an ordered `steps:` list, an authored
Target envelope, and a reviewed artefact behind every call it makes (§3, §9). A one-off act against a
credentialled Target is written as a Procedure of one Step, or it is not written.

The reading a competent implementer reaches unaided is that **a tool built around single Operations
should let you invoke one**. It is what the neighbouring tools do — `ansible -m`, `kubectl create`, every
CLI that wraps an API — and the friction it removes is real: you have a Definition, you have reviewed it,
and the Procedure that wraps one call looks like ceremony. The reading is also what an earlier draft of §9
said, in the sentence this decision deletes: *`run` has two forms and the difference between them is
authority, not sugar*.

That sentence was wrong on its own terms. The direct form's authority is exactly the Definition's, which
is what a one-Step Procedure declares — so the difference is sugar, and it is sugar bought at a price
paid in four places nobody had costed:

- **It has no shape in the Store.** `run.json` carries `procedure`, and a Step file carries the Step's
  authored `id` beside its `definition`, `operation` and `target` (§7). A direct invocation has neither, so
  it writes a Journal entry naming a Procedure that does not exist or it writes one with a hole in it.
- **It can never be Compared.** The baseline is the previous Run of the same Procedure (§8). A Run with no
  Procedure is neither a baseline nor a subject, so whatever it did to the world reaches no `YOU DID THIS`
  row on any surface. That is *nothing changes unseen* failing, on an effectful path, silently.
- **It cannot `destroy`.** A `destroy` Step's Bound is mandatory and an absent Bound is unbounded, which is
  refused before anything runs (§5). There is nowhere to write a Bound, so half the tool's stated purpose
  was already unreachable through this form by a rule nobody had applied to it.
- **Repeatability has nothing to key on.** Run-once refuses on what the Journal holds *for that Step* (§6).
  There is no Step, so the evidence the whole re-run model reads has no subject.

We chose to delete the form rather than narrow it because narrowing fixes only the fifth problem — the
one that raised the question, an Operation with a required input having nowhere to read one from — and
leaves the four above standing. What survives a narrowing is an Operation with no required input, no
selector, no Bound, no condition, `read` or `mutate` only, once, against one Target. Against `local` and
`read`, that is already a Probe (ADR-0009). Against anything else, the artefact costs three lines and
buys the Record, the Comparison row, the gutter the review annotates, and a Bound where the Kind demands
one. A door that narrow, opening onto a Run the record cannot describe, does not earn its place.

Stating the rule at the model rather than at the command line is the point of it. As a CLI restriction it
leaves the Store's shape accidentally sound; as *every Run is a Run of a Procedure* the three holes above
close by construction, and each becomes a property an implementer can rely on rather than a case that
happens not to arise.

## Considered options

- **Narrow the form to Operations with no required input.** The ticket's first candidate, and the
  smallest change. Rejected on the four costs above, none of which the narrowing touches, and on what it
  would add: either a thirty-ninth `error_code` for a Refusal, or a usage error that has to explain why
  half the Operations a Provider declares are unreachable through a documented command.
- **Let the values come from somewhere reviewed.** The ticket's second candidate. The only reviewed home
  for an argument value is a Step's `args:`, and a Step lives in a Procedure (§3), so this *is*
  `run <procedure>` over a Procedure of one Step. It is not a rejected option so much as the chosen one
  under a name that hides it — and keeping the direct form as its spelling would be a second
  representation of one act, refused everywhere else in this model.
- **Take the values at invocation, typed by the Operation's input schema**, as a Probe does. Rejected by
  ADR-0008 before this ticket existed: an input is authority arriving after review, Step behaviour on no
  reviewed line, and the review surface cannot annotate what is not in the file. A Probe's `--input`
  survives that argument only because a Probe is not a Run — `read` Kind, `local` Target, no Record and no
  Journal entry — so its input chooses what is looked at and can change nothing a later Run reads.
- **Widen the Probe to credentialled Targets and to `mutate`**, so the ad-hoc case has a home. Rejected
  as the deleted door reopening under another name. Every property that makes a Probe safe to exempt from
  the record is the property being removed, and ADR-0017 additionally forbids the wire half of it.

## Consequences

- **A one-off act against a credentialled Target requires an artefact first**, and nothing grows to cover
  it. This is ADR-0008's *parameterisation is duplication, deliberately* applied to occasions instead of
  parameters, and it is stated as a non-goal in §13 rather than as a ceiling victim: nothing here is
  unwritable, only unwritten. The ritual is author, check, review, run — and this was the one door that
  skipped the first three.
- **Handing `run` two positionals is a usage error, exit `2`**, not a Refusal. A Refusal is the artefacts
  declining an act and has a check to name; here nothing was reviewed, so nothing refused. Exit `77`
  additionally promises that the remedy is an artefact edit (§9), and the remedy here is a different
  command or a Procedure that does not exist yet.
- **One positional naming a Definition is a usage error too**, on the kind mismatch: `definitions/` and
  `procedures/` are different directories and the kind is known before anything loads (§3). This is the
  case a reader of the old §9 will type from memory, and the message names the Procedure they want to
  write.
- **The MCP `run` tool loses its union**, taking a `procedure` and nothing else. A call carrying a
  `definition` is an argument violating a schema, which is a protocol error — what that surface has in
  place of exit `2` (§9). The tool count is unchanged at thirteen, as the command count is at sixteen: a
  form died, not a verb.
- **The Comparison window is total.** No Run can reach the world outside some Procedure's Comparison,
  which is a stronger claim than §8 could previously make and is half the thesis.
- **`Run` in the glossary means one thing.** It read *a single execution of an Operation or a Procedure*,
  which was already loose — a Probe executes an Operation and is not a Run — and is now exact.
