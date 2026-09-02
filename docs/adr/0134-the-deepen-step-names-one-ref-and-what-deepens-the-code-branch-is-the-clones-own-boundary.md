# The deepen step names one ref, and what deepens the code branch is the clone's own boundary

The projected workflow's deepen step is

    if [ -f .git/shallow ]; then git fetch --unshallow origin "$GITHUB_REF"; fi

which is the line it always was with a remote and a refspec on the end of it. That closes #258: a
**bare** `--unshallow` inherits `remote.origin.fetch`, on a runner that is the wildcard
`+refs/heads/*:refs/remotes/origin/*` that `git remote add` wrote, and so the step took the Store
branch with complete history on every Run — the cost
[ADR-0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md) declined
`fetch-depth: 0` in order to avoid, paid by the line written to avoid it (#246,
[ADR-0132](0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md)).

**The repair costs one argument because a refspec argument configures nothing.** That is the same
property of `git fetch` that produced the fault: `actions/checkout`'s own `--depth=1` fetch names an
explicit refspec, which is why the wildcard `git remote add` left is still there afterwards. Read from
the other end it is the fix — a fetch can say what it wants without writing what it wants down, so
there is nothing to set and nothing to unset, and the step stays one line inside one `if`.

## What the argument does, and what it does not

**It is not what deepens the code branch.** `--unshallow` deepens whatever the clone already holds
shallowly, whatever refspec it is given; the refspec decides only which refs are brought down beside
that. Measured in a runner-shaped clone — `git init`, `git remote add`, one `--depth=1` fetch with an
explicit refspec, `checkout -B` — against a remote holding `main` at 20 commits and `hyper-store` at 13:

| the deepen step runs | `main` | `hyper-store` | `.git/shallow` |
|---|---|---|---|
| `git fetch --unshallow` | 20/20 | **13/13** | removed |
| `git fetch --unshallow origin "$GITHUB_REF"` | 20/20 | **absent** | removed |
| `git fetch --unshallow origin refs/heads/hyper-store` | 20/20 | 13/13 | removed |

The third row is the demonstration: naming a ref that has nothing to do with the checked-out branch
still deepens the checked-out branch to 20. **The argument's whole work is to stop the wildcard**, and
the code branch is deepened by the boundary the clone is already carrying.

**`hyper`'s own Store fetch then behaves as ADR-0074 says it does.** Run after the middle row —
`git fetch --no-tags --depth=1 -- origin refs/heads/hyper-store:refs/remotes/origin/hyper-store`, which
is `fetchShallow` — the branch arrives at **1 commit of 13**, `.git/shallow` comes back naming the Store's
tip and nothing else, and `main` is still 20 with the oldest blob on it readable. That consequence of
[ADR-0074](0074-the-store-branch-is-fetched-shallow-and-whole.md) was written true, measured false on a
runner (ADR-0132), and is true again.

## Why the ref is the executor's own

The generator does not know the branch name and this decision does not give it one. §11's compiled-in
set is four constants — two URLs, the runner image, the checkout SHA — and a branch name would be a
fifth that is a fact about somebody's repository rather than about the job. Authoring one puts a git
ref into a reviewed artefact that says nothing else about git, and §10 has no field to put it in.

**The executor names it, because the file's only trigger makes it name one.** The projection derives no
second occasion for a Run to start (§10), so the trigger is the Cadence, and a `schedule` event runs on
`refs/heads/<default branch>` and says so in `GITHUB_REF`. It is `$GITHUB_REF` rather than
`$GITHUB_REF_NAME` because the fully-qualified form cannot be a tag that shares a branch's name.

**Where nothing names one, the fallback is not the wildcard.** Measured: with the variable unset the
step runs `git fetch --unshallow origin ""`, git reads that as the remote's default branch, `main`
deepens to 20 and the Store does not arrive. So the failure mode of an absent executor variable is a
correct fetch rather than the fault this decision removes — which is worth knowing and is not what the
step rests on.

This is the third executor-supplied name in the file, beside `$GITHUB_STEP_SUMMARY` and every
`${{ secrets.… }}` the bindings write. None of them reaches the binary: `hyper` is told nothing about
the executor and resolves an environment variable on a runner exactly as it does on a laptop (§10, §11).

## Considered options

- **Accept it, and correct §13's two curves to say the whole history recurs.** Rejected. The ground
  offered for it — a runner's clone is discarded and the wire cost is the executor's — is true about the
  disk and false about the wire: the bytes cross it on every scheduled occurrence of every Procedure,
  and the branch they come from is the one §13 names as growing without bound, with the Journal term
  Compaction cannot reclaim inside it. At a five-minute Cadence that is the curve §13 says nothing
  recurs on, and two words remove it. An honest limit is for a cost that cannot be paid down, not for
  one nobody got round to.
- **Set `remote.origin.fetch` before the `--unshallow`.** Rejected. It needs the same ref this does, so
  it buys nothing on that axis; it costs a second line in a file where every line is argued for; and it
  *writes* the clone's configuration, which every later step then runs under — including `hyper`'s own
  fetches, which would silently start relying on a narrowing they do not ask for. A refspec argument
  configures nothing, and leaving the runner's git exactly as `checkout` left it is the smaller claim.
- **`--negotiation-tip`, or a `--shallow-since` window.** Rejected. Both are heuristics over history
  where the thing actually wanted is a statement of *which branch*, and both put the amount fetched at
  the mercy of what the remote and the clone can negotiate — a network dependency in the same family
  ADR-0074 refused a blob filter for.
- **A narrowed refspec with the branch name compiled in or authored.** Rejected above: it is a fifth
  constant or a new authored field, and the executor already knows the answer.
- **`fetch-depth: 0`, or dropping the deepen step.** Rejected in ADR-0071 and unchanged here: the first
  repairs the code branch by fetching the Store branch, and the second gives up eight of the nine code
  classes and the whole line count on every window that had anything in it.

## Consequences

- **Every projected file's bytes change, so every repository holding one is `projection-stale` until
  `hyper project` is run again.** That is the ordinary blast radius of a generator change and the check
  names it exactly (§10); it is stated here because the step's *behaviour* changing is the point and
  the file changing is the price.
- **ADR-0071's decision stands and its reason is replaced.** The deepen step is still the supply-side
  repair for `not-in-clone`, and it is still one guarded line. What it no longer rests on is *the
  checkout left a narrow refspec behind*; it rests on naming one, which is true of any clone the step
  meets.
- **§13's two curves stand as written, and now say the line that holds them.** `hyper`'s sync pays for
  the live tree once per environment that lacks the branch, and the whole history is the clone's rarer
  bill. Nothing on a runner enforces that: it is one argument in a generated file, and a hand-written
  workflow that deepens its own checkout pays the old cost with nothing in `hyper` to say so.
- **What recurs on a runner is now the code branch's history.** The deepen step still takes all of it,
  on every Run, and that cost is unchanged and was always the deal ADR-0071 struck. The difference is
  which curve it is on: a code branch grows with commits a human makes and a Store branch grows with
  the Cadence, so the recurring fetch is now bounded by the repository rather than by the schedule.
- **This repair is enforced, not taught** (`docs/agents/acceptance-re-runs.md`). The worked example in
  `internal/workflow/testdata/` is compared byte-for-byte, the `project` goldens carry the file whole,
  and a package case fails on the bare form specifically. Nothing here is a sentence an agent reads and
  then decides about, so **no sealed acceptance run is owed**.
