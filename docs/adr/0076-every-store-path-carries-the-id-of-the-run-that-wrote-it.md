# Every Store path carries the id of the Run that wrote it

The Store is append-only, and a push rejected as non-fast-forward fetches, re-applies the path set the
Run wrote onto the fetched tip, and commits (§7, ADR-0011, ADR-0075). That re-application is clean where
the path sets are disjoint, and §12's grammar very nearly guarantees they are: a Record version is
`records/<target>/<definition>/<name>/<run-id>-<nnnn>.json`, and a Journal entry is
`journal/<yyyy>/<mm>/<dd>/<run-id>/…`. Three of the five path forms carry the id of the Run that wrote
them, and no two Runs can mint one `<run-id>`.

The closing write was the sole breach. When a Run closes an entry it does not own it writes the in-flight
Step file at the next `<nnnn>` and `outcome.json` — under the **dead** Run's `<run-id>`, not its own — so
those two paths are the only ones in the Store a second writer can reach. That is why there was exactly
one unclean case in the whole push protocol, and it is the whole of why there was one.

**The closing write now carries the closer's id too.** It is one file,
`journal/<yyyy>/<mm>/<dd>/<run-id>/closed-by/<closer-run-id>.json`, inside the dead Run's entry and under
the dead Run's date partition — so *is this entry closed* stays a listing of one directory — and named by
the Run making the claim. Two Runs cannot write one path, in any circumstance, anywhere in the Store.

## What it was chosen against

**The reading an implementer reaches unaided is that closing an entry means writing its `outcome.json`.**
That is what closing is, the file is already specified, and §12 listed it as written *when the Run ends,
or by the Run that closes it*. Under that shape the disagreement is real and the protocol resolves it by
push order: whoever reached the remote first is what the Store holds, and the loser is `failed` with `75`
having pushed nothing.

§7 said that disagreement *stands as a conflict rather than being resolved in either direction* — *it is
a disagreement about what happened, and picking a side is the tool editing evidence*. First-push-wins is
a resolution, in favour of whoever was earlier. The claim and the mechanism disagreed, in the one section
whose subject is not editing evidence.

## The case it is really about

Not two reapers. Two Runs reaping an abandoned entry write content ADR-0003 fixes entirely — `failed`,
in-flight Step *attempted, outcome unknown* — and differ only in who closed it and when, which are facts
about the **closer**. Neither is a rival account of what happened to the dead Run.

The case is a **live-but-slow Run closing its own entry while another Run has already reaped it**. One is
an observation and the other an inference, and they contradict. §6's lock is one filesystem's and §10's
`concurrency` group covers runner-to-runner, so the population is exactly one shape: an effectful Run on
the laptop overlapping an effectful Run on a runner. There is no recency test and §6 refuses one on
purpose — *`hyper` never guesses which* — so overlap does not race toward the reap, it guarantees it.

Under the old shape that live Run did not merely lose the argument. The reaper writes the in-flight Step
file at the next `<nnnn>`, which is precisely the number the owner writes when that Step finishes, so the
owner **died at its next Step push** — `75`, having pushed nothing, its confirmed Records for that Step
going with it, and every Step after it never running. Reverse the push order and it is the reaper that
takes the `75`: a Run with work of its own, killed by a housekeeping act performed for a Run that was not
there.

## Consequences

- **There is no unclean case left.** §7's *every retry is clean but one* becomes *every retry is clean*.
  The `75` on a push keeps the two meanings it always also had: three retries against a remote that keeps
  moving, and a remote it could not reach.
- **A disagreement is held rather than adjudicated.** A **contested** entry is one holding both an
  `outcome.json` its own Run wrote and a `closed-by/` file another Run wrote. Both stand, both are true
  — the reaper truthfully records that it inferred death at an instant — and nothing is deleted. The
  entry's outcome is the **owner's** wherever one exists, because an observation is what an inference was
  an inference about.
- **A wrong reap costs nothing.** The owner's paths cannot be taken, so it finishes on its own terms and
  never learns it was contested. §6's outcome triple loses *an entry closed by a later Run* as a cause of
  `failed`, there being no longer any way for one Run's reap to reach another Run's outcome.
- **`closed_by_run` leaves `outcome.json`.** Its whole job was to mark a file as not written by its
  entry's own Run, which the path now says, and a key restating its own path is the second representation
  §7 refuses for durations. The `closed-by/` file does not carry it either, for the same reason.
- **The file carries no `outcome`.** ADR-0003 fixes it to `failed` always, so the file's existence is the
  whole answer. `disposition` stays despite likewise holding one value: the entry's outcome is a question
  about the **entry** that a file's existence answers, where a Disposition is a fact about a **Step** that
  this file is the only carrier of, and §8 reads Dispositions generically across all seven.
- **§12's grammar goes from five path forms to six, and the schema integers from four to five.** That is
  the price, and it buys a set the grammar can be checked against: every path carries the id of the Run
  that wrote it, and `STORE.md` — written once, by no Run — is the only exception.
- **The reap folds into the sync.** It is decided from the same fetched tip the Run is about to build on,
  and its files go out in the push of `run.json` (§6, §7), so the reach count at the remote is unchanged.
  A Run reaps every open entry it finds: under this ADR a wrong inference is harmless, and a rule reaping
  some and not others would need a criterion, the only candidates being age and liveness — both of them
  the guess §6 declines.

## What it does not fix

Two effectful Runs across the laptop and a runner are still serialised by nothing, and the reap was only
ever the symptom that made it visible. What this ADR guarantees is that the **record** stays truthful and
complete under that overlap. What is unguarded is the **world**: two Runs may mutate one Target at once,
and the remedy is the lock server ADR-0006 rules out. §13 carries it.
