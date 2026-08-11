# A Cadence and a run-once Step are refused together

Run-once Refuses where the Journal holds the Step as *ran*, and a Refusal is terminal for the Run. Put
a Cadence on a Procedure containing one and the arithmetic is short: occurrence one runs, occurrence two
Refuses at that Step, and every Step after it in the Procedure is *never reached* — at that occurrence
and at every occurrence for as long as the clock keeps firing. **`check` refuses the combination**
(`cadence-run-once`, §4), walking every Procedure reachable from the one declaring the Cadence, to any
depth.

The reading a competent implementer reaches unaided is that run-once is a default like any other and a
default needs no check. It is a reasonable instinct and it ships a Procedure that works exactly once.
Three things make that outcome worse than an ordinary authoring mistake:

- **Nobody is present when it happens.** ADR-0005 puts the clock on GitHub Actions, so the second
  occurrence Refuses unattended, some night after the first one worked.
- **There is no way out but an artefact edit** (ADR-0001), and the artefact that needs editing may not
  be one the repository owns: `repeatability:` lives in a Manifest, and an installed Manifest is
  verified by digest (§11).
- **It arrives through an omission rather than a decision.** Run-once is what an Operation gets when its
  Provider author writes nothing, and the author who wrote nothing is not the author who wrote the
  Cadence. The two artefacts are separately correct and the combination is fatal.

The check has no false positive worth the name. Nothing an author can write makes a run-once Step under
a Cadence come out well: it is not a bootstrap Step that fires once and then lets the rest proceed,
because the Refusal takes the rest down with it. The nearest thing to an exception is a run-once Step
behind a `when:` that never holds, which `check` cannot evaluate and therefore refuses too — and that
Step is not benign either, only postponed. The morning its condition first holds is the morning the
Procedure stops running, which is refusing a delayed certainty rather than a guess.

## Considered options

- **Make an undeclared `repeatability:` a load error on an effectful Operation.** The obvious fix — it
  attacks the omission at its source — and it is rejected. The primary author of a Manifest is an AI
  (ADR-0023), a required field gets filled by reflex, and the reflex value is `repeatable`. That trades
  a silence which fails safe and loudly for a guess which fails open and silently: the current failure is
  a named Refusal with a rendered remediation path, the traded one is an effect repeated on a Cadence
  because a field had to be filled. Run-once by omission stays what §12 calls it, the strict one, and
  this check is what makes the strictness survivable.
- **Let a Definition or Step override the Manifest's Repeatability.** Rejected twice over: it is the
  fact §6 gives the Provider author precisely because the Definition author would be guessing, and at
  the Step it is authority arriving after review (ADR-0008). It also would not help the case that
  motivates it — a consumer overriding a third party's omission is a consumer asserting that invoking
  something twice is safe, which is the one thing they are least placed to know.
- **Catch it at `project` rather than at `check`.** The Cadence becomes real when the workflow is
  generated, so this reads naturally. Rejected because it splits one truth across two commands: `check`
  would pass on a repository `project` refuses, and `check` is the surface §4 built to be the one that
  can be trusted standalone. `projection-stale` is already a Cadence rule that `check` carries.
- **Render it rather than refuse it** — a line beside the Cadence gloss saying this Procedure runs once.
  Rejected. A rendering cannot stop anything (§0), and ADR-0021 keeps `hyper` from introducing a claim
  of its own where a Cadence renders — the same rule that stops it saying *overdue*. This is not a
  judgement about the world, though: it is two reviewed artefacts contradicting each other, which is
  what `check` is for.
- **Refuse only from the second occurrence, leaving the first legal.** Rejected: a repository that
  passes every check and works once is the failure this decision exists to remove, moved rather than
  fixed. `check` is the pre-flight of every Run, so the combination cannot be legal at the first
  occurrence and illegal at the second without `check` learning what occurrence it is in — which is the
  is-CI axis §5 deleted.
- **Reserve the clock for `read` Procedures.** Rejected already by §10, and it would not close this: with
  ADR-0037 a `read` can no longer be run-once, but the Procedures that most want a Cadence are the ones
  that also mutate, and unattended effectful Runs are what the two keys and the mandatory Bound exist to
  make safe.

## Consequences

- **One new `error_code`, `cadence-run-once`,** counted with §10's rather than §4's, on
  `projection-stale`'s precedent: the check runs in `check` and the fact it is about is the Cadence. The
  set goes to thirty-nine.
- **The walk is `envelope-exceeded`'s.** §10 makes *any shared body factored into a nested one* the way
  two recurrences are authored, so the run-once Step will ordinarily sit in a nested Procedure rather
  than in the one carrying the clock. A shallow check would miss exactly the case the idiom creates.
- **`run-once-recorded` stays, and stays reachable.** A Procedure with no Cadence, invoked by hand twice,
  still Refuses at its run-once Step, which is what run-once is for. This decision removes the case where
  nobody is there to read the Refusal, not the Refusal.
- **A named limit in §13.** Where the omission is in an installed Manifest, the consumer cannot correct
  it at all — the digest verification makes a local edit a failure rather than a fix, and no artefact
  downstream may override a Manifest's declared facts. What the check buys is when they find out:
  offline, at authoring time, naming the Operation and the file.
- **The replacement idiom is two Procedures**, and it is stated where the refusal is: the run-once Steps
  in one that is run by hand, the recurring Steps in one that carries the Cadence. §10 already has the
  shape for it, factoring shared body into a nested Procedure.
