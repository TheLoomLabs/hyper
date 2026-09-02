# The projected job ran on a runner, and the deepen step fetched the Store whole

**The job `hyper project` writes has executed on a GitHub-hosted runner.** It checked out, deepened,
fetched `hyper-0.0.1-alpha-x86_64-linux.tar.gz`, verified it against the digest frozen into
`hyper.yaml`, untarred it, and ran `hyper` on a machine created for the job and destroyed after it —
four times, three green and one red on purpose. Three Runs are on the Store branch of a public
repository and are readable from a clone. Two effectful jobs triggered together serialised on
`hyper-store` with no interleaving and no deadlock.

**Two things were found, and both are about the world rather than about the generator.** The deepen
step fetches the Store branch in full on every Run, which is the cost
[ADR-0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md) declined
`fetch-depth: 0` in order to avoid — §11, ADR-0071 and
[ADR-0074](0074-the-store-branch-is-fetched-shallow-and-whole.md) all rest on a claim about
`actions/checkout` that is not true. And the `hyper changes` step cannot fail, so a Comparison that
does not render leaves a green job. Issue #246.

## What ran, and what did not

**GitHub delivered no `schedule` event to the fixture repository at all** — not one in the hour the
three projected workflows stood on the default branch at `*/5 * * * *`, and none since. This is not
the generated file: a hand-written `canary.yml` outside hyper's namespace, carrying the same
expression, had its ticks dropped identically while its `push` run queued in about a second. The
workflows were `active`, the repository public and not a fork, `allowed_actions: all`.

**A projected file may not be edited to add a trigger.** `hyper run` compares it against a
regeneration at run start, before the first Step, and the comparison is whole-file:

    refused · exit 77 · run 01a06206-4644-7bd5-a530-d6dc301543ba
      = file: .github/workflows/hyper-observe.yml
      = checked at run start, before the first step
      = hyper project

That is §10 working as written — the reviewed artefact declares a recurrence *and no second occasion
for a Run to start* — and it closes the obvious way round the scheduler.

So the jobs were dispatched from **copies outside the `hyper-*.yml` namespace**, where `project` does
not speak for them and the projection check does not see them; `hyper check` answered *checked 8
artefacts: no problems found* throughout. Everything from `permissions:` down is byte-for-byte the
projected file — verified by hashing that span on both — and the whole of the difference is the
header comment, `name:` and the `on:` block. **What executed is therefore the generated job and not
the generated file**, and every claim below should be read as being about the former. The three
`hyper-*.yml` files were never edited.

## Claim 1 — a read-only Run, on a runner, in the Store

`dispatch observe`, green in fourteen seconds on `ubuntu-24.04`:

    hyper.tar.gz: OK
    run 01a06208-f5d3-7143-8557-1f1297c2fb14
    step 1/1 name-the-machine
    STEP  ID                KIND  DISPOSITION  RECORDS
    1     name-the-machine  read  ran          1

    completed · exit 0 · run 01a06208-f5d3-7143-8557-1f1297c2fb14

`hyper changes observe` rendered on the same job under `if: always()`, reporting `THE WORLD MOVED  1
observation` against `no baseline — first Run of observe`. The read-only workflow carries no
`concurrency:` block, which is the projection's other branch and is what this job is.

## Claim 2 — the Records are on the branch, and the branch reads back

Twelve commits on `hyper-store`, four per Run, authored by `hyper <hyper@hyper.invalid>` — hyper's
own identity, not the runner's, which never set one:

    511a4e6  End run 01a0620a-7a2f-7c78-b381-3cba0ba0cf36
    a50fdc0  Step 1 hold of run 01a0620a-…: ran
    b4a5577  Record local/host-effects/["sh","-c","sleep 45; echo alpha held the store"] at run 01a0620a-… step 1
    98c6b88  Begin run 01a0620a-7a2f-7c78-b381-3cba0ba0cf36

and the same four for `mark-beta` and for `observe`. Fetched into a clone afterwards, `hyper runs`
reads all three:

    RUN             STARTED                   TRIGGER      OUTCOME    CONTESTED  PROCEDURE   TARGETS  HYPER
    01a0620a-7a2f…  2026-09-02T12:14:08.943Z  TheLoomLabs  completed             mark-alpha  local    0.0.1-alpha
    01a06209-96de…  2026-09-02T12:13:10.750Z  TheLoomLabs  completed             mark-beta   local    0.0.1-alpha
    01a06208-f5d3…  2026-09-02T12:12:29.523Z  TheLoomLabs  completed             observe     local    0.0.1-alpha

The push went out over the credential `actions/checkout` left behind, which is what
`persist-credentials: true` is written out for, and the `HYPER` column is the pin the job installed.

## Claim 3 — `hyper-store` serialised two jobs that were triggered together

Both effectful workflows were dispatched in the same second. The group held:

    mark-beta   created 12:13:04Z  started 12:13:06Z  completed 12:14:02Z  success
    mark-alpha  created 12:14:03Z  started 12:14:05Z  completed 12:14:58Z  success

**alpha's job was not created until one second after beta's had completed.** Not interleaved, not
deadlocked, and neither cancelled — `cancel-in-progress: false` is why the second is held rather than
discarded. Each Step slept 45 seconds so that an overlap would have been visible as one; the Store's
own commit order agrees, beta's four commits standing below alpha's with no interleaving.

This is the stand-in §10 describes doing what it is for: §6's single-store lock is a lock on a
filesystem, and these two jobs shared none.

## Claim 4 — the digest check runs, and it is load-bearing

The positive is in every green job: `hyper.tar.gz: OK`. That it is not decorative was shown by
changing **one hex character** of the frozen digest in a fourth copy — `d9a6…` to `e9a6…` — and
dispatching it:

    sha256sum: WARNING: 1 computed checksum did NOT match
    hyper.tar.gz: FAILED
    ##[error]Process completed with exit code 1.

    4  install hyper 0.0.1-alpha: failure
    5  hyper run observe: skipped

The step failed, the job failed, and `hyper` never ran. The check gates the binary rather than
reporting on it.

## Claim 5 — the deepen step fetches the Store branch, whole, on every Run

Every one of the three runs logged this under **deepen the checkout**, before `hyper` was installed:

    From https://github.com/TheLoomLabs/hyper-runner-fixture
     * [new branch]      hyper-store -> origin/hyper-store

**Three places say that cannot happen.** §11:

> `checkout` leaves `remote.origin.fetch` pinned to the single ref it checked out, so an `--unshallow`
> after it inherits that refspec and reaches nothing else. That is written down rather than relied on.

ADR-0071 says it in the same words, and ADR-0074 rests a consequence on it. **The claim is false, and
the mechanism is visible in the runner's own log.** `actions/checkout` does not clone. It runs

    git init /home/runner/work/hyper-runner-fixture/hyper-runner-fixture
    git remote add origin https://github.com/TheLoomLabs/hyper-runner-fixture
    git -c protocol.version=2 fetch --no-tags --prune --no-recurse-submodules --depth=1 \
        origin +c3d423f…:refs/remotes/origin/main

and `git remote add` writes the **wildcard** refspec, `+refs/heads/*:refs/remotes/origin/*`. The
explicit refspec on the fetch is an argument to that one command and configures nothing. A bare
`git fetch --unshallow` afterwards therefore inherits the wildcard and takes every branch, with
complete history. Reproduced outside Actions, in a clone built the same way: 13 of the Store's 13
commits arrive, and `remote.origin.fetch` reads back as the wildcard. A `git clone --depth 1
--branch main` **does** pin the refspec to one ref, which is the shape the claim describes and is not
the shape a runner is in.

Two consequences follow, and both are the opposite of what is written down:

- **`fetch-depth: 0` was declined to avoid exactly this, and the deepen step does it anyway.** §14
  costs a runner at *the whole live Store on every scheduled occurrence of every Procedure*; what it
  actually pays is the whole **history**, which §14 assigns to the rarer act of cloning. At a
  five-minute cadence, with the Journal the term Compaction cannot reclaim, that is the curve §14
  says nothing recurs on.
- **ADR-0074's *the Store fetch re-creates `.git/shallow` after the deepen step has cleared it*
  does not happen.** After the deepen the branch is already whole, so hyper's shallow fetch has
  nothing to bring: measured in a runner-shaped clone, no `.git/shallow` exists after the Run.

**Nothing is repaired here.** The false sentence is corrected in place at the three sites, that being
what a measured falsehood earns; what to *do* about a deepen step that fetches the Store is a
decision with §11, §14, ADR-0071 and ADR-0074 in it, and it is filed rather than taken.

## Claim 6 — `hyper changes` cannot fail the step it is written into

The control run's `./hyper` never existed, the untar having never run. Step 6 executed anyway, under
`if: always()`, and was reported **success**:

    /home/runner/work/_temp/27b51bc1-….sh: line 2: ./hyper: No such file or directory

    6  hyper changes observe: success

`writeChanges` writes `./hyper changes <procedure> | tee -a "$GITHUB_STEP_SUMMARY"` and stops there.
A pipeline's status is its last command's, so `tee`'s `0` is the step's, whatever `hyper` did.
`writeRun` has precisely the machinery that would prevent it — `code=${PIPESTATUS[0]}`, with `set +e`
around it so the closing fence is still written — and its doc comment explains why it is there.

On this job it cost nothing: the job was already red from the install step. **The case it does cost
is the one where the Run succeeds and the Comparison does not render** — a `git` failure, a Store
that cannot be read — and there the summary carries an empty fence between two backtick lines and the
job is green. Filed rather than fixed, the fix being one line but the question *should a failed
Comparison fail a Run that succeeded* being a real one.

## What this does not establish

- **The `schedule` trigger has still never fired.** Everything above was reached by
  `workflow_dispatch` on a copy. The generated file's own trigger — the only one it has — remains
  unexercised end to end, and #246's *push, let the job run* was not what happened. That the executor
  dropped every tick for an hour is itself worth knowing against §10's cadence gloss, and is not
  evidence that it always would.
- **No credential was exercised.** Both Procedures are `shell` over a `local` Target, so the `env:`
  block — the one part of the job derived from a Target's credential slots — was absent from all four
  files. A projected job carrying `${{ secrets.… }}` has still never run.
- **Nothing contended for a Record.** The two effectful jobs serialised, so the push path never met a
  remote that had moved, and `ErrPushExhausted`, the re-application and the `CONTESTED` column are
  where they were.
- **One runner image, one platform.** `ubuntu-24.04`, `x86_64-linux`, four jobs. The other three
  archives §11 names remain unexercised (#247).
