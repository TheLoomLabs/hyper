# A Secret sink is a directory `hyper` makes, and one file holds one value

The Secret sink is a **directory** the invocation names and `hyper` creates `0700`. Under it one file
holds one value, at

```
<sink>/<nnnn>/<name>/<field>
```

`0600`, the value's own bytes and nothing `hyper` added. `<nnnn>` is the Step's position in the Run's
written order, `<name>` the Record's name and `<field>` the field the Manifest declared `secret:` — the
triple, percent-encoded exactly as §12 encodes a Store path segment. The path must not already exist:
every file under a sink is that Run's. `secret-sink-absent` returns to §12's closed set in
`secret-sink-unwritten`'s place, and §8's fourth remediation class has its member back. Issue #270.

This is the half [ADR-0146](0146-a-sink-nothing-writes-is-a-refusal-and-not-a-completed-run.md) parked.
That record ended the silence — a Run that would produce a secret Refuses rather than destroying it —
and left the format open, on the ground that an on-disk format for secrets decided in passing on the
way to fixing a warning is the wrong way round. This is that decision, taken on its own.

## What the format had to express

**One Run produces more than one secret, in three independent ways**, and no two of them are the same
axis:

- **Two secret-producing Steps in one Procedure.** `check` permits it — nothing anywhere refuses the
  combination, and `internal/cli/testdata/run/repo-two-secrets` is that repository.
- **One Step that expands.** An Expansion holds one Record series per member (§6), and the suppression
  runs per Record, so a Step over five members produces five values.
- **One Operation declaring two `secret:` fields.** `SecretFields` is a set on the Operation, so the
  leaf is a *field of a Record version* and not a Record.

So the addressable thing is the triple **Step → Record → field**, and it is not reducible: no two of the
three collapse into one another.

**It is written once and never read back.** `hyper` does not parse it, no Run reads it, and it never
reaches the Store — which is the whole point (ADR-0007, ADR-0011). The constraint is therefore
**legibility rather than round-tripping**: an operator or a wrapper script has to get a value out
without a parser, and two secrets must never become ambiguous.

## The decision

**A directory, and the filesystem carries the structure.** There is no key to design, no parser to run,
and `cat "$sink/0001/db-1/password"` is the whole of what a wrapper does. The three axes above become
three path segments, so the shape that expresses them exactly is the one that needs no format at all.

**`<nnnn>` names the Step and an authored id does not.** A Step's `id:` is what its author called it and
it is unique inside one Procedure; a Procedure invoking one other Procedure twice gives both
invocations' Steps one id *and* one path (`internal/run/sequence.go`). The position is the Run's own
counter and unique by construction, it is what §12 already numbers a Step file and a Record version by,
and it is on the Step table the operator is reading. A key that is not unique is a key that collides the
day someone writes the case it does not cover, which is the failure this whole record is about.

**`<name>` is the Record's name and the Target and Definition are not repeated.** A Step binds one of
each, so within one Step a Record is named by its name alone — and that name is the third segment of
§12's own `records/<target>/<definition>/<name>/`, so it is a name the reader can carry straight back to
`hyper records --name`. It goes through the Store's own encoder rather than a second copy of it:
`store.EncodeSegment` is exported for this, because a Record whose name is not a filename must be one
name under `cat` and under `hyper records`, and one rule spelled in two packages is where the day comes
that they disagree.

**The file holds the value and nothing `hyper` added.** No trailing newline, no quoting, no wrapper. A
newline `hyper` invented is a byte the endpoint never issued, and a wrapper that did not strip it would
send a credential that is not the credential; `$(cat …)` strips one either way, so the shape that is
right for a reader that strips is also the only one right for a reader that does not. A value that is
not a string is written as the JSON it is, which is what a projected value reads as on every other
surface (`projection.Text`, §9).

**The sink must not already be there, and `hyper` makes it.** A directory `hyper` did not make may hold
an earlier Run's files, and a file this Run does not write is one an operator reading the sink takes for
this Run's — a **stale credential read as fresh**, which is strictly worse than the empty file ADR-0146
refused for reading as filled. `os.Mkdir` is the test and the act at once. Making it also puts the mode
in `hyper`'s hands, which is what lets §9 promise `0700` and `0600` rather than describe them.

**The empty-file objection is what the directory shape answers.** ADR-0146 recorded why a file created
empty at Run start is wrong: *a file created empty at Run start would be a sink an operator could read
as filled*. An empty **directory** has nothing to read — `cat "$sink/0001/db-1/password"` fails loudly
where `cat file` returns the empty string. That asymmetry is what makes the third axis of the decision
free: values are written **at the Step that produced them**, and a Run that halts part-way leaves
exactly the secrets it actually produced, legibly, with the rest absent rather than blank.
`run/a-halt-leaves-the-secrets-it-already-wrote` is that case rather than that sentence — Step 1's
value in the sink, Step 2 on its deadline, nothing under `0002/`, and the Run `failed` at `1`.

**The mode is set and not requested.** `os.Mkdir` and `os.WriteFile` take a mode the process umask
subtracts from, so a `hyper` running under `umask 0700` would create a sink its own operator cannot
read — and §9 promises `0700` and `0600` rather than *at most* them. A `Chmod` follows each, which is
what makes *the mode is in `hyper`'s hands* a fact; without it the bits a corpus golden holds would be
the umask of whoever ran the suite.

**A path already there is a usage error and not a Refusal**, beside `-` and a path inside the working
tree. It is a fault of the *flag*, decidable from the invocation, and it is stated whatever the
Procedure holds — `hyper run watch-status --secret-out -` is already a usage error on a Procedure that
produces no secret at all. A fault that turned on what a Procedure happened to declare would be one an
artefact author could introduce into somebody else's invocation.

**The directory is made at §6's gate, not at the first Step that fills it.** A sink that cannot be made
— a parent that is not there, a directory this process may not write — then stops a Run that has not yet
touched anything, rather than one that has already mutated three Assets and has nowhere to put the
fourth's credential. It is made **only where the Run reaches a Step that declares secret output**, so a
sink named against a Procedure producing none leaves no empty directory behind.

## Considered options

- **One secret, one file.** Simplest, and `cat` gets the value with no parsing at all. Rejected because
  it cannot express any of the three axes above, so shipping it means shipping a new §4 rule and a new
  static code refusing a second secret-producing Step — and nothing at all could be done about the Step
  that expands, an Expansion's width not being knowable offline. **Narrowing the authoring format in
  order to keep a file format simple is the trade backwards**: the format is the thing that is cheap to
  get right, and §4's rules are the thing under review.
- **A keyed document.** One file, keyed to reach a member; `jq -r '…'` in the shell that was about to
  `export` it. Rejected on the stated constraint. It puts a parser between the operator and the value at
  exactly the moment a wrapper is least able to fail safely, and it obliges the key to be defended: a key
  that is not the whole triple collides the day someone writes the case it does not cover, and a key that
  *is* the triple is the directory shape with a document wrapped round it and `jq` in the way. It also
  forces the second decision — write per secret or write at the end — to be answered *at the end*, since
  a partially written JSON document is not a document, which loses every secret a halted Run already
  produced.
- **A directory whose members are keyed by Step id rather than position.** Rejected above: two
  invocations of one Procedure give two Steps one id. Adopting it would have meant either a new
  uniqueness rule in §4 or a silent overwrite of one secret by another, which is the loss again with a
  smaller blast radius.
- **`<sink>/<target>/<definition>/<name>/<field>`, mirroring the Store exactly.** Rejected as a fold that
  loses the Step: two Steps of one Run binding one (Target, Definition) and projecting one identity —
  a rotate followed by a read — would write one path twice, and the second value would land on the first.
  §12's Store grammar avoids this by carrying `<run-id>-<nnnn>` in the *filename*; the sink carries the
  position in the leading segment for the same reason.
- **Write everything at the end of the Run.** Rejected: a Run that halts at Step 4 of 6 has already
  produced Steps 1–3's secrets and would discard them, which is ADR-0146's loss arriving one gate later.
  It also holds every secret of the Run in memory until the last Step, where writing at the Step holds
  each for the few lines between the projection and the file.
- **Requiring the sink to exist and be empty, rather than making it.** Rejected. `mktemp -d` is one word
  shorter that way, and the cost is that the mode is then the operator's: `hyper` cannot promise `0700`
  over a directory it did not create. *Empty* is also a weaker test than *absent* and a racier one.

## Consequences

- **`secret-sink-absent` returns to §12's closed set and `secret-sink-unwritten` leaves it.** Fifty-two
  members either way; the two never stand in the set together, exactly as ADR-0146 said. That is
  **breaking on the wire again and in the same way**: a consumer matching `secret-sink-unwritten` matches
  nothing, which is the intended failure — the state that string named is not a state any Run is in.
- **§8's fourth remediation class, *a different invocation*, has its member back**, and it is the only
  remedy in the set the Run's own operator can take without leaving the shell: *the same command again
  with `--secret-out <path>`*. It is still the one class no generated workflow can take at all, which is
  why `cadence-secret-output` stands unchanged (§4, ADR-0077).
- **`cadence-secret-output`'s remedy is now a working Run.** §4's split — the secret-producing Steps in a
  Procedure a person invokes with `--secret-out` — cleared the *Cadence* fault before and left the
  hand-invoked Run Refusing anyway. It completes now. §4, §10 and §13 said so was pending and now say it
  is done.
- **`run.Request` carries `SecretSink` again.** ADR-0146 removed it precisely so that nothing carried a
  path in order to drop it; the writer is what earns it back. It arrives **absolute**, resolved by the
  surface against that surface's working directory, so the engine still reaches no process fact of its
  own.
- **`0600` is implemented, forced and asserted.** §9 has said *a sink is to be written `0600`* since it
  was written, and nothing wrote any mode until now. `internal/run/sink_test.go` holds the bits on both node
  kinds, and the corpus's new `sink.golden` renders every node's mode — a `0600` file inside a `0755`
  directory would be a secret whose directory anyone can list, so the guarantee is about the tree.
- **The corpus gains a fourth golden, and it is the axis that could not exist before.** `store.golden`,
  `tree.golden` and the two text streams are all blind to whether a secret arrived: a Run that produces
  one reports a Step that *ran* and writes a Record holding the constant marker whether the value landed
  anywhere or nowhere. **That is exactly the reading that ratified the defect** — `run/a-secret-field-is-
  the-marker` was green while the Run destroyed what it was invoked for (issue #266). `sink.golden` is
  the only golden that can tell those two apart. A case naming a sink must hold one; the single exception
  carries a `sink-occupied` marker and is the case driving the *already there* fault, whose path the
  corpus itself checked in.
- **The corpus gains five cases and loses none.** `a-sink-supplied-refuses-like-one-withheld` becomes
  `a-secret-reaches-the-sink` at exit `0`, with its `-json` and MCP twins and the `serve/` fixture
  ADR-0146 deleted; `two-secret-steps-fill-two-directories` and
  `an-expansion-fills-one-directory-per-member` drive the two axes *one secret, one file* could not have
  expressed; `a-rehearsal-fills-the-sink-it-was-given` is the completing half of
  `a-rehearsal-earns-no-sink-exemption`; `a-halt-leaves-the-secrets-it-already-wrote` is the case the
  issue named as deciding when the file is written; and `usage-secret-out-at-a-path-already-there` is
  the third fault. `repo-two-secrets`' Operation drops to `deadline: 1s` on `repo-drain`'s precedent,
  so the halt is a number a case can drive to. `a-secret-sink-names-every-step` keeps its name and its argv and now Refuses under the other
  code.
- **`store.EncodeSegment` and `store.StepNumber` are exported.** They were §12's path grammar and are now
  §12's and §9's, spelled once. Nothing else about the Store's paths moves, and the sink is not a path in
  that grammar: it is not on the branch and never reaches one, which §13 now says where it states the
  grammar.
- **A member that made no call writes no file, and that is a stated limit rather than a silence.** A
  `skip-if-recorded` Step whose member the Store already holds concluded about that Record without asking
  the world for it, so there is no value to write; a `mutate` declaring `secret:` output that skips
  therefore completes with nothing in the sink for that member. The **absence is the answer** and it is
  legible — the directory shape's own dividend — where a keyed document would have carried a key with
  nothing under it or no key at all. Whether §4 should refuse `skip-if-recorded` on a secret-producing
  Operation outright is a separate decision and is not taken here; it is not the loss ADR-0146 named,
  because the value was never produced.
- **The run this repair owes is deferred, and this is the record of that.** Per
  `docs/agents/acceptance-re-runs.md` the repair is **both** taught and enforced, so it counts as taught:
  the Refusal, the write and the modes are fenced by the corpus and by `internal/run/sink_test.go`, and
  what is taught is the `run` tool's `secret_sink` description — an agent that reads *hyper creates it as
  a directory and fills it* and then names a path that already exists gets a `2` naming the reason, and
  nothing in the suite fails. **No task in the acceptance set reaches a Step whose Operation declares
  `secret:` output**, which is the same gap ADR-0146 recorded and did not buy. There is still no
  transcript this repair could be measured in, so the gap is a task file rather than an unspent run
  (#222, #250), and it is not being bought here either. Two deferrals now stand on the same missing task;
  that is the argument for writing it. **It was written**: `push-credential` (#271) reaches a Step
  declaring `secret:` output and is arranged around the round trip past the Refusal, so the run both
  repairs owe is a run of it. Adding the task is what fenced the gap; the run is still unbought.
