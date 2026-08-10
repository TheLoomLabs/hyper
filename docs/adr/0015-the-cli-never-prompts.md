# The CLI never prompts

`hyper` asks the operator nothing. There is no confirmation before a destroy, no "are you sure", no
interactive picker, no TTY-conditional behaviour anywhere except colour and terminal width. A command
either does what its arguments said or Refuses, and every surface reads identically in a terminal, in
a pipe, and in an Actions log.

We chose this because a confirmation prompt is a bypass with better manners. ADR-0001 removed every
`--force` and `--skip-checks` on the ground that the only way past a guardrail is editing a reviewed
artefact; a prompt puts the decision back at the keyboard, unreviewed, unrecorded, and answered in
the second before it is understood. It also cannot exist in the environment `hyper` was built for:
unattended effectful Runs on a schedule are normal (ADR-0005), so a prompt is either skipped there —
making the guardrails different on a runner than on a laptop, the axis the safety model deleted — or
it hangs a scheduled Run until it times out.

The pre-flight summary is the same decision wearing a milder face and is rejected with it. `hyper
run` renders nothing before executing: a summary with no question after it is decoration that reads
like a checkpoint, and in CI it scrolls past unread. Review happens at review time, on the artefact,
through `hyper review` — before the commit, not before the process.

## Consequences

- **`--json` implies nothing about interactivity**, because there is no interactivity to suppress.
  Swamp's convention that `--json` skips confirmation prompts has nothing to attach to here.
- **A destructive Run started by mistake is not catchable at the keyboard.** What stands in its place
  is entirely static and entirely before the invocation: the two-key check, the named-Operation
  requirement on destroy, the mandatory Bound, and the Definition review. This is a deliberate
  relocation of the safety net, not a gap in it, and it is worth stating plainly because the moment a
  human most wants a prompt is exactly the moment this decision denies them one.
- **Ctrl-C is the only interactive control**, and it is handled as a signal rather than a question:
  the Run drains — the Step in flight finishes and is recorded as having run — closes its own Journal
  entry `failed`, and exits 130.
