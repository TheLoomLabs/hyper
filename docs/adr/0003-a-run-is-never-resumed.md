# A Run is never resumed

When a Run halts partway, `hyper` has no resume: no `--from`, no downstream-reset graph, no
run-state file to continue out of. **Re-running the Procedure is the retry.** What makes that safe is
not a checkpoint but a declaration — each Operation's Repeatability, stated in the Manifest, tells
`hyper` whether a Step already done should be re-invoked, skipped, or refused.

We chose this because the alternative contradicts two things `hyper` has already committed to: CI runs
are fully self-contained with no state shared with the laptop, and the binary is stateless. A resume
file is precisely shared state, and it would make the two environments behave differently. It also
collides with the safety model — resuming a halted destroy means resuming into the state we
deliberately refused to reason about. The property that makes resumption unnecessary is that
everything a Step produces is already durable in the Record store, written per Step as it completes,
so a fresh Run reads exactly the state a resumed Run would have rebuilt.

## Consequences

- State does not disappear, it changes character. The Journal records each Run historically — its
  outcome, its Provenance, every Step's Disposition — but it is a record of the past, never
  resumable state.
- Safety on re-run rests on Repeatability being declared honestly by Provider authors. The default
  for an undeclared Operation is run-once, which refuses on re-run rather than repeating an effect
  nobody vouched for.
- A killed process leaves an *open* Journal entry rather than no entry. The next invocation closes it
  as `failed`, with the in-flight Step marked attempted-outcome-unknown. There is no reaper and no
  daemon: it is noticed by the next Run that looks, which is the only moment it matters.
