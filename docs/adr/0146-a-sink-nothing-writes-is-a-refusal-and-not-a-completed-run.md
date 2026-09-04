# A sink nothing writes is a Refusal and not a completed Run

A Run reaching a Step whose Operation declares `secret:` output Refuses, at Run start, naming every
such Step at once — **whether or not `--secret-out` was supplied**. The code is
`secret-sink-unwritten` and its remedy is a different binary. `secret-sink-absent` leaves §12's closed
set until there is a file for a sink to be absent from; the count is unchanged, the two never standing
in that set together. Issue #266.

`--secret-out` stays exactly as it is: accepted on `run` and no other command, `-` refused, a
flag-shaped value refused, a path inside the working tree refused. What it no longer does is decide
anything — no path reaches `internal/run` at all, and `run.Request` carries no `SecretSink`.

## What was wrong (issue #266)

Nothing wrote the file. The whole use of the supplied path was its emptiness: the §6 gate returned
early where a sink was named, and `SecretSink` reached the CLI flag, `run.Request`, that gate, and
nowhere else. There was no `os.Create`, no `WriteFile` and no `0600` outside a test.

So a Run given a sink **completed**. It performed the Step, projected the response, wrote the
declared-secret field to the Store as the constant marker ([ADR-0007](0007-hyper-never-stores-a-secret.md),
[ADR-0142](0142-a-declared-secret-field-is-suppressed-on-every-surface-that-renders-a-projection.md)),
and discarded the value. A Provider author who wrote `secret: [root_password]` got a clean Run at exit
`0`, a Record that read correctly, and no password — with no warning on any surface.

**Nothing leaked; the failure mode is loss.** That is the whole reason this sat unnoticed: every
guarantee held, and the thing that did not happen was the only thing the operator wanted. It read as
done from every angle — `--secret-out` in `--help`, in §9, in the `run` tool's schema, and a green
golden (`run/a-secret-field-is-the-marker`) ratifying the completed Run as correct behaviour.

The trail out of the source comment ran into #133 and #137, which are **closed spec milestones**. A
reader following it landed on completed work.

## The decision

Two things were tangled, and only one of them is blocked.

**The format is blocked and stays parked.** What the file holds is undecided: one secret in one file
cannot express a Procedure reaching two secret-producing Steps, which `check` permits; a keyed document
has to key to a *member*, an Expansion over five producing five; a directory makes the Step/member
structure the filesystem's and `0600` per file. It is written once and read by a human or a wrapper,
never by `hyper`, so the constraint is legibility rather than round-tripping — and it is an on-disk
format for secrets, which is worth deciding slowly and in a record of its own.

**The silence is not blocked, and it is what this record ends.** A build that cannot write a sink does
not accept one and proceed. Refusing is the posture every other guardrail here takes, it needs no
format decision, and it converts a silent loss into a stated limit.

**It is one code and not two, and that is the load-bearing half.** The supply that is missing is the
*binary's* and not the invocation's, so a sink named and a sink withheld are the same Run. Keeping
`secret-sink-absent` beside it would have put a code in the closed set that no build can produce —
`unwritten` is decided first, on the stronger fact — or, worse, kept its remedy alive: *the same
invocation again with `--secret-out <path>`* sends an operator round a loop that ends on another `77`.
That is precisely the failure [ADR-0145](0145-an-empty-credential-is-its-own-refusal-on-both-surfaces.md)
split the credential codes to prevent, arriving from the other direction: there, one code could only
offer the wrong remedy to one half; here, one remedy would be wrong for both.

**The flag is kept and the engine is not handed it.** The path is still worth refusing before it is
worth writing to — `-` and a path in the working tree are faults whoever eventually writes the file —
and §9 has documented the flag since it was written. What is removed is the field: `run.Request` held
a path the gate no longer reads, and a value carried into a package to be dropped is how this defect
was built in the first place.

## Considered options

- **Write the sink now and decide the format here.** Rejected as scope: the format is one decision and
  the silence is another, the second is not blocked by the first, and an on-disk format for secrets
  decided in passing on the way to fixing a warning is the wrong way round. This record leaves the
  three shapes on the table exactly as issue #266 states them.
- **Keep completing, and warn.** Rejected on [ADR-0001](0001-no-bypass-for-safety-refusals.md)'s
  ground and on §8's: a warning on a `completed` Run at exit `0` is a line that scrolls past in CI,
  and the Run it annotates has already destroyed the value. `hyper` has one way of saying *I will not
  do this*, and it exits `77`.
- **Refuse `--secret-out` itself as a usage error.** Rejected. It states the limit on the surface that
  is not the problem: a Run that withholds the flag loses the secret just as completely, so the fault
  would still be silent for exactly the invocation nobody typed a flag on. It also amputates the flag
  from `--help`, §9 and the MCP schema for the length of the deferral, to be re-added unchanged.
- **Two codes: `secret-sink-absent` when none was named, `secret-sink-unwritten` when one was.**
  Rejected above. The remedies are identical — there is nothing the operator can do either way — and
  §12 splits codes where the *remedies* differ, never where the inputs do.
- **Leave `secret-sink-absent` in §12 as the name of a check that cannot fire.** Rejected: the closed
  set is what a consumer matches on, and a member no binary produces is a set describing a different
  program. It returns with the format, and §12 says so where it once stated the member.

## Consequences

- **`error_code` is renamed within a fixed count.** Fifty-two members, with `secret-sink-unwritten`
  where `secret-sink-absent` stood. That is **breaking on the wire**: a consumer matching the old
  string matches nothing, which is the intended failure — the state it named is not the state a Run is
  now in.
- **§8's fourth remediation class is empty until the sink is written.** *A different invocation* had
  one member and it was this one. `secret-sink-unwritten`'s remedy is a **different binary**, beside
  `store-schema-unsupported` and `manifest-schema-unsupported`: *a hyper that writes a Secret sink —
  nothing in the repository is the fault*. The class returns with the format, and §8 records that it
  is empty rather than deleting it.
- **`cadence-secret-output`'s remedy is a shape and not yet a working Run.** §4's split — the
  secret-producing Steps in a Procedure a person invokes with `--secret-out` — clears the *Cadence*
  fault, and that hand-invoked Run then Refuses like any other until the file is written. §4, §10 and
  §13 say so rather than leaving a reader to discover it. The static rule itself does not move: a
  recurrence with no invocation that could ever supply a sink is still refused before its first
  occurrence.
- **A Run can no longer reach the marker.** `internal/run`'s suppression of a declared-secret field
  into `store.Secret` stands and is correct, and nothing exercises it end to end while this gate holds:
  every Run that would has Refused. The marker's other surfaces — `probe`, `provider`, the Comparison
  and the Record renderings — are unaffected and keep their fixtures (ADR-0142).
- **The corpus loses a completed Run and gains a Refusal.**
  `run/a-secret-field-is-the-marker` is now `run/a-sink-supplied-refuses-like-one-withheld`, with its
  `-json` and MCP twins: exit `77`, one member, and the response fixture it no longer reaches deleted.
  `a-rehearsal-refuses-the-sink-it-was-not-given` is `a-rehearsal-earns-no-sink-exemption`, the name
  no longer implying that giving one would have helped. The claim the old case made — *the secret
  reaches the Store as the marker and the sink as nothing* — was true, and it was the ratification the
  ticket names: a green golden standing behind a Run that quietly destroyed what it was invoked for.
- **It is enforced in the half that matters and taught in one clause, and the run it owes is
  deferred.** The Refusal is fenced by the corpus on both surfaces — the page, the `--json` stream and
  the entry all carry the new code — so an agent that ignores every sentence here still cannot lose a
  secret. What is taught is the `run` tool's `secret_sink` description, which now says supplying a path
  rescues no Run; nothing in the suite fails if an agent reads that and supplies one anyway, and what
  happens then is a `77` naming the reason. Per `docs/agents/acceptance-re-runs.md` a taught repair
  names its run: **no task in the set reaches a Step whose Operation declares `secret:` output**, so
  there is no transcript this repair could have been measured in, and the gap is a task file rather
  than an unspent run (#222, #250). It is not being bought here.
