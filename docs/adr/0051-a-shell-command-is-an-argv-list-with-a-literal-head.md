# A shell command is an argv list with a literal head

A `shell:` request block carries one key, `command:`. **It is a list of argv words `hyper` execs
directly, with no interpreter between the artefact and the process, and its first member is a literal.**
The words arrive in a Step's `args:` rather than in the Manifest, `hyper` being the only author a
`shell` Operation can have and knowing nothing whatever about the command.

The reading a competent implementer reaches unaided is the one every scripting tool ships: `command:`
is a string, and `hyper` hands it to `/bin/sh -c`. It is what the Capability's own name suggests, it
makes a pipe work, and it renders in the gutter as a line the reviewer recognises from their terminal.
It is also the one shape that lets the world choose what runs. Every non-Capability-relevant position
resolves to an Operation input (§12), and a Step's argument may be a reference to an earlier Step's
Record — so under the string reading `command: "deploy {tag}"` with `tag` read off an API response
runs whatever that response said, semicolon included. ADR-0029 closed exactly this for a host, where a
grant still stood behind the decision; here it would arrive on the Capability §13 states has no grant
at all.

An argv list closes it structurally rather than by escaping. A hole fills one word whatever characters
the value holds, for the same reason a parameterised query is not string concatenation, and there is no
rule about metacharacters to state, get wrong, or maintain. What survives is a list a reviewer reads
member by member — which is what §13 means by *the words a reviewer read in the Procedure*, and it is
more nearly true of a list than of a line.

The head is separate because it is the reach axis. `host:` is the one Capability-relevant position in
an `http` request, and the shell equivalent of choosing a host is choosing a binary; a reference there
is the same arrival by a different door. It cannot be made Capability-relevant in §12's sense, that
position resolving only to a declared enumeration or to `from-target`, and `hyper` cannot enumerate the
executables on a machine it has never seen. So it is a literal instead — a Step-level requirement,
checked against the Procedure alone, with no Store, no credential and no network. Members after the
first stay referenceable, which is the whole of what makes an Expansion over a `values:` list writable.

## Considered options

- **A string handed to `/bin/sh -c`.** The unaided reading, rejected above. It also makes the format's
  one guarantee about holes unstateable: `hyper` can say a hole fills a position, and under a shell it
  cannot say what position a value ends up in.
- **A string, with `hyper` escaping every interpolated hole.** Rejected as the same decision with an
  escaping rule bolted on. The rule would be a closed set of its own in a specification that keeps
  those in one place, it differs per shell, and the failure mode of getting it subtly wrong is arbitrary
  command execution rather than a wrong answer.
- **An argv list with no literal-head rule.** Rejected on ADR-0029's ground alone. It is a smaller hole
  than the string reading and the same hole: a reference in position zero is the world choosing the
  program, and nothing downstream would notice.
- **Make the head Capability-relevant and enumerate permitted executables in the Target declaration.**
  Genuinely attractive — it would give `shell` the grant §13 says it lacks. Rejected because the grant
  would be a lie at the width that matters: an enumeration containing `/bin/sh` or any interpreter
  grants everything, and one that does not is a list a repository must maintain against a machine
  `hyper` never inspects. §13 states the absence of a bound honestly instead of installing one that can
  be stepped around in one word.
- **Keep `cwd:`, `stdin:` and `env:` as authored keys.** Rejected together. Each is a further position a
  hole could fill on the Capability with no grant, and an authored `env:` in particular would route a
  secret through an argument list, which is the one place §7 can neither suppress nor recognise it. All
  three are fixed instead: the repository root, empty, and the environment §11 already states.

## Consequences

- **No pipe, no redirection, no glob, no `&&`**, and §13 carries it. What replaces them is a script in
  the repository invoked as one word, which moves the shell logic from a line nobody reviews into a file
  reviewed like any other — at the cost of being a file at all.
- **One new `error_code`, `command-malformed`**, covering a `command:` that is empty and one whose first
  member is a reference. One code because it is one check — *a shell Step names its executable
  literally* — on the shape `credential-slot-malformed` already has. §12 goes to forty-five with
  ADR-0053's.
- **The built-in's six Operations share one request**, `command: "{command}"` over an input schema of
  `{type: array, items: {type: string}}`. Nothing in the Manifest varies by Operation except the two
  facts the roster exists to vary, and §12 renders it in full.
- **`hyper` binds the child's process group.** Not a consequence of the argv decision as such, but it
  arrives with the exec: without it a terminal's interrupt reaches the child directly and §6's drain is
  a sentence the implementation contradicts. A deadline then kills the group rather than the pid, and
  §13 carries what a second interrupt leaves behind.
