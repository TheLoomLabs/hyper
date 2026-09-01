# The suite is run by a machine, and a prepared machine may not skip

**`.github/workflows/suite.yml` runs `go build ./...`, `go vet ./...` and `go test ./...` on every
push to `main` and every pull request; `release.yml` calls that same file and `publish` waits on it;
and the runner installs `bubblewrap` and sets `SUITE_PREPARED=1`, which turns the acceptance seam's
skip into a failure.** Nothing about the product moves. What moves is who has run the suite before a
change lands, and whether a tag can publish a tree nobody ran it on.

## What was wrong (issue #243)

`.github/workflows/` held one file. A tag was its whole trigger, and its steps were: checkout, read
the version from the tag, pin the toolchain, `scripts/release.sh`, check that the binary inside the
artefact reports the tag, `gh release create`. There was no `go test` anywhere in the repository's
automation. **Nothing ran the suite except a person at a terminal.**

[`docs/build/releasing.md`](../build/releasing.md) opened its ritual with *land everything, on
`main`, with the tests green* — an instruction to that person, and an assertion nothing checked. A
tag pushed against a red tree built, version-checked and published.

**It is this repository's own thesis, unapplied to itself.** *Nothing reaches the world unreviewed*
is a claim about what a Run may do, and the one thing here that reaches the world is `gh release
create`. What stood in front of it was one string comparison, and `release.yml` says in its own
comment why that one is there: a mis-stamped artefact Refuses in every repository that installs it
and the remedy is another release. **The same sentence is true of behaviour and not only of the
stamp.** A release whose `check` is wrong Refuses nothing, is discovered by an author, and the remedy
is the same. The gate that existed guarded the label on the tin.

Every ticket from #213 to #242 was implemented, reviewed and merged with the suite run by hand. That
worked, and it worked because one person was doing all of it in sequence. It was a habit, not a
property this repository held.

## The trap: a runner with no `bubblewrap` un-fences six tasks

`TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse` opens with
`needTools(t, "bash", "bwrap", "git", "go", "python3")`, and `needSeal` skips where `bwrap` cannot
build a namespace. Both are right: a suite that failed on a machine without the tool would be
asserting something about the machine rather than about the code.

`ubuntu-24.04` ships no `bubblewrap`. On such a runner those two gates mean the acceptance case
**skips in silence** — and that case is the fence that ranges over `tasks/*.md` rather than naming a
task, which is [issue #222](https://github.com/TheLoomLabs/hyper/issues/222)'s repair: adding a task
file is the whole of what fencing it takes. A job that skips it un-fences all six at once, and the
job is green. That is #222's rot arriving through a new door.

So the runner installs the tool — and something has to make the claim checkable, because *the case
did not run* and *the case passed* look identical in a job's summary.

## What was decided, and against what

**`SUITE_PREPARED` is a claim the machine makes, and the gates read it.** One environment variable,
set by `suite.yml`'s test step and by nothing else, meaning: *the tools are installed and a namespace
can be built here.* `unavailable` (`cmd/hyper/suite_test.go`) is the one place the decision is made
— `needTools`, `needSeal` and `hostPlatform` all route through it — and where the variable is set it
fails, naming the variable, rather than skipping. A laptop leaves it unset and keeps the skips, which
is the honest answer there.

The alternative was **to read `go test -json` in the job** and fail on a `skip` action for that test
name. It was rejected on two counts. It puts the assertion in YAML, where nothing under review holds
it and no case can exercise it, and it either loses human-readable output or needs a second tool to
render it. And it does not generalise honestly: fifteen cases in this suite skip, and most of them
should — permission bits under `root`, a platform with no `sleep`, a temp directory inside the git
repository. A blanket *no skips* rule would be false, and a rule naming one test by string in a
workflow file is a fence that a rename walks straight through. The variable puts the rule where the
gates are, and `TestUnavailable_APreparedMachineFailsWhereAnotherSkips` holds it.

**The suite is a callable workflow, and `release.yml` calls it.** The tag path cannot be covered by
the `main` path: neither `push` nor `pull_request` fires for a tag, and the commit a tag points at is
not the one the last `main` run tested — a tag can be pushed at any commit, including one no pull
request ever saw. Two ways to close that were available. Transcribing the three commands into
`release.yml` would be a second copy of the job to keep in agreement, which is the thing that file
already refuses to do with `scripts/release.sh`. **Calling the same file** is the same decision:
`uses: ./.github/workflows/suite.yml` is this commit's copy of it, which is also why that one form of
`uses:` needs no commit pin.

**`bwrap` is verified rather than assumed.** Ubuntu 24.04 restricts unprivileged user namespaces
through AppArmor. The package ships a profile that permits them; where that has not taken effect the
job lifts the restriction and tries again, in a step of its own. Doing it there rather than leaving
it to `needSeal` means the failure names its cause — a runner that cannot build a namespace — instead
of surfacing as a test that could not run.

## The seam the fence is expressed at

Four properties of these files are held by cases in `cmd/hyper/suite_test.go`, which read the parsed
steps and never the file's text — a workflow's own comments being the one place every string those
cases look for could appear while the job did none of it. That is
`TestRelease_TheTagRunsTheScriptTheseCasesRun`'s method, applied to the second workflow: the three
commands under the two triggers, the toolchain line compared against `release.yml`'s own rather than
spelled a third time, every `uses:` in the directory pinned by a 40-character commit, and the
publishing job listing a caller of `suite.yml` in its `needs:`.

The fifth property is not a question about the file, and is the one above: a green job cannot be
distinguished from a skipped case by reading YAML, so it is asserted in Go.

## Consequences

- **A pull request and `main` both run the whole suite**, including the acceptance seam's setup half
  for all six tasks, on a runner that has `bubblewrap`.
- **A tag waits for the suite.** The release pays the suite's minutes before it publishes, which is
  the same wait a pull request pays. A red tree fails before `gh release create` is reached.
- **A gate that routes through `unavailable` is now a CI failure**, which is `needTools` and
  `needSeal` — the two that name something a preparation could have supplied. `hostPlatform` is
  deliberately not one of them: the release publishing nothing for this architecture is a fact no
  runner setup could change. Whoever adds a case to `cmd/hyper` that legitimately cannot run on a
  prepared machine reaches past those two, and says why.
- **What is held is the gates and not the case.** A `t.Skip` written inside the acceptance case, or
  the case being deleted or renamed, passes this rule untouched — those are review's to catch, and
  the fence #222 built is the one that makes an unfenced *task* impossible.
- **Every other skip is untouched**, `cmd/hyper/main_test.go`'s own included. Nothing here claims *no
  test skips*, which would be false: fifteen cases in this suite skip and most of them should. The
  claim is narrower and true, and `CONTRIBUTING.md` states it in those terms.
- **`docs/build/releasing.md`'s step 1 is checked.** It stays in the document, because it is still
  the order to work in — but a tag pushed against a red tree no longer publishes.
