# A command's stdout is text, never a parsed object

The `shell` Capability has a response object of its own, closed at four members — `command`,
`exit_code`, `stdout`, `stderr`. **`stdout` and `stderr` are text and are never parsed**, so
`$.stdout.result.id` is not a path and a shell projection reaches `$.exit_code`, `$.stdout` and
`$.stderr` and nothing finer.

The reading a competent implementer reaches unaided is the symmetry: `body` is the parsed JSON body,
absent where the response carried none or carried something else (ADR-0040), so `stdout` is the parsed
JSON output, absent where the command printed something else. It is one line of code, it costs no new
rule, and it makes `aws ec2 describe-instances --output json` project as cleanly as an API call. That
last clause is the objection rather than the argument for it.

`opaque` is a claim about what `hyper` can describe. A Manifest describes an `http` Operation
completely — a method, a host, a path, a body, every one of them in the artefact — and the review of
that description is what the whole tool rests on. A command is the case where no such description
exists, which is why the Capability is reserved to `hyper` (ADR-0039) and why §13 states its reach is
bounded by nothing but the words in the Procedure. Parsing its output is describing it: it says this
command answers in a shape, that the shape has a `result`, that `result` has an `id`. Nothing
established any of that. The Provider author who would vouch for it is `hyper`, which has never seen
the command.

The consequence of the unaided reading is worse than a category error. If a command's output projects
like an API's, then `curl` is a better Provider than a Manifest — the same reach, the same projection,
none of the declaration, no capability grant, no static check, and nothing for a reviewer to read but
a command line. Every incentive the format creates points at writing Manifests, and it points there
only while the escape hatch is worse at the job than the road. Keeping stdout opaque is what keeps
`shell` a last resort rather than a shortcut around the entire design.

Nothing new enforces it. The path grammar has three productions and none of them reaches inside a
scalar (§12), so the restriction is the grammar the format already had, applied to a member that
happens to be a string. A shell response holds no collection either, so `over:` has nothing to name and
every shell Operation is of `one` cardinality by construction rather than by a rule.

## Considered options

- **Parse stdout as JSON where it parses, absent otherwise.** The unaided reading, rejected above. It
  has a second fault beside the category error: whether a projection resolves would depend on what the
  command happened to print that day, so a Manifest correct on Monday fails on Tuesday when a warning
  reaches stdout, and the failure is a halted Run rather than a wrong field.
- **A declared output format per Operation** — `json`, `lines`, `text`. Rejected as the output schema
  §3 refuses, arriving on the one Provider whose author cannot fill it in. It also would not be
  declarable where it matters: the six Operations are `hyper`'s, and the format is a property of the
  command a Step supplies.
- **Project only `exit_code`, and carry `stdout` and `stderr` on the Step file** the way `answered` is.
  Rejected on growth, which was the concern that motivated it: Observations compact and Journal entries
  never do (§7), so moving the output out of the Record makes the cost permanent instead of
  reclaimable. It also puts what the world said in the place reserved for what `hyper` did.
- **Project `exit_code` and `stdout`, dropping `stderr`** as diagnostics rather than state. Rejected
  because it leaves a failing `read` mute — exit `1` and nothing about why, with no raw wire to fall
  back on, ADR-0017 having removed that everywhere.
- **A compiled-in size cap with truncation.** Rejected as a guessed number, on the ground ADR-0045 left
  `concurrency:` at 1: every value `hyper` could choose is a guess about output nobody described. §12's
  over-long path truncation is not the precedent it looks like — that one serves a filesystem limit,
  not a preference.
- **Four more Operations, so an author picks recording or not.** Rejected on the same argument that
  keeps the roster at six for the facts it does vary: ten Operations on the Provider that is meant to be
  the last resort is a large price for an opt-out from a cost §13 can simply state.

## Consequences

- **A command's structured output is a blob**, and §13 says so. The cost is uneven by Kind, which is the
  half worth knowing: a shell `read`'s output is an Observation and Compaction reclaims its interior
  versions, while a shell `mutate`'s is an Asset and Compaction never removes one.
- **`command` is the identity, and it is `host`'s argument one Capability over.** An Operation whose
  answer carries no identity of its own has nowhere else to project one from, so the member that is a
  fact about the call rather than about the answer is what the Record is named by. It is JSON-encoded
  on one line rather than joined, because a joining rule makes `[echo, "a b"]` and `[echo, a, b]` one
  series and `record-identity-collision` can never catch it — the two names being genuinely equal.
- **A shell Record has three fields and one name, for every Definition in every repository.** The
  Manifest is `hyper`'s and no artefact downstream overrides a declared fact (§13), so `identity:
  $.command` with `exit_code`, `stdout` and `stderr` is the whole of what a command can ever record.
- **The built-in declares no `secret:`**, and §13 states what that costs. A `secret:` list is a Provider
  author's claim about output that author understands; declaring `stdout` secret on every command would
  be `hyper` asserting a fact it cannot have, and it would make `[systemctl, is-active, nginx]` Refuse
  without a secret sink. The honest reading of the same ignorance is to declare none — which qualifies
  ADR-0007 rather than sitting beside it, a command's stdout being the one place a credential can reach
  the Store through no position `hyper` owns.
- **§12 holds two response objects rather than one**, and which a projection reads from is decided by
  the key the Operation's request is written under. They share no member: `http` describes what it did
  and `shell` describes nothing, so what each can be asked about afterwards differs the same way.
