# `hyper` never checks the Store out

The Store is an orphan branch of the same repository (ADR-0006), fetched shallow and whole (ADR-0074),
and its files are authoritative over every answer `hyper` gives about the record (ADR-0011, §7). No
section said where those files sit on the machine doing the reading. They sit nowhere: `hyper` reads the
Store out of git tree objects and writes it with `hash-object`, `commit-tree` and `update-ref`, and no
byte of Store content is ever an ordinary file on disk. *Files are authoritative* stays exactly true —
the files are the ones in the tree, and finding a Head is a listing of one.

What follows from it is a stronger guarantee than the one it was chosen for: **the Store has no
uncommitted local state, ever**. One commit per confirmed write, pushed after every effectful Step (§7).

## The reading an implementer reaches unaided

**A second git worktree.** It is the obvious answer, it is what `git worktree` is for, and §7 primes the
reader for it: *the version that is current is visible to anyone reading the branch, in a fresh checkout
or a browser, with no git plumbing and no tool.* That sentence has been read as a claim about the
checkout `hyper` keeps. It is not — the browser clause is what gives it away. It quantifies over
**humans** reading the branch, and it is a promise about what the branch *is*, not about how `hyper`
reads it.

And the worktree is the one shape that takes the promise away. `git worktree add` locks the branch to
that worktree, so a human on their own laptop then gets `fatal: 'hyper-store' is already checked out at
…` for the checkout §7 offers them.

## Why the crash guarantee decides it

§6 requires a serial effectful Expansion to leave what it confirmed rather than nothing — *three
Tombstones where the fourth call never returned, which the next Run reads*. §7 pushes after every
effectful **Step**, so those three are written between pushes, and what they are between pushes is the
whole question.

In a worktree they are **uncommitted files a crashed process left**, and the next Run cannot tell them
from a hand-edit of evidence — `STORE.md` says editing the branch by hand is editing evidence (ADR-0011),
and a sweep that commits whatever it finds is the tool doing exactly that on someone else's behalf. It
must therefore either commit files it did not write or discard what §6 promised, and neither is
available.

Committing per confirmed write removes the state rather than adjudicating it. There is nothing on disk
to leave behind, nothing to reconcile, and §7's *what it wrote before it stopped stands locally and goes
out with the next Run that syncs* becomes literally true of the branch tip. A torn write cannot be
observed either: `hash-object -w` lands its object atomically, and an object nothing references is
invisible rather than half-present.

The one thing that must not happen under this shape is a per-call write that nothing points at — a loose
object no commit reaches is unreachable, and `git gc` eats it. That is why the commit is per confirmed
write and not per Step: the granularity is what makes §6's sentence survive, not an optimisation.

## The costs the worktree carries and this does not

- **A third curve.** §13 prices the Store twice — a clone pays the history, `hyper`'s sync pays the live
  tree, and the sync is paid on every scheduled occurrence of every Procedure (ADR-0074). Materialising
  adds writing that whole live tree to disk, immediately after fetching it, in order to read the handful
  of files a Run touches.
- **An ignore rule for a path nobody reviewed.** The only files `hyper` writes into the working tree are
  the workflows `project` regenerates and the binary `install` unpacks, both reviewed. A Store worktree
  would be a third and the only unreviewed one, and `.gitignore` is not among the five artefacts.
- **A channel into the record that the model does not otherwise have.** A `shell` Operation's working
  directory is the repository root (§3). A materialised Store is readable from there; a Store that is
  only ever tree objects is not reachable by a command at all.

## What it costs

**`git` is a subprocess, and it is the one external tool the binary requires.** The surface goes from
two commands to a dozen, and the alternative — an embedded implementation — would have to resolve git's
config, credential helpers and URL rewrites in order to inherit the token §10's `persist-credentials:
true` leaves behind, which is the part of git nobody should reimplement. §11's exemption covers what the
**generated workflow** consumes on a runner; this is what the **binary** consumes on a laptop, so §13
names it rather than letting the exemption stretch. No minimum version is stated: every command on this
path predates 2010, and a pin would be a fifth projection constant ADR-0046 closed at four.

## Considered options

- **A second git worktree.** Rejected above: it takes §7's checkout away from the human it was promised
  to, and it is the shape whose crash residue §6 cannot be honoured against.
- **A materialised directory under `.git/`.** It keeps the branch unlocked, so the human's checkout
  survives — but every other cost above is unchanged, the crash residue most of all. It is the worktree's
  problem with the worktree's one benefit removed.
- **A fresh temporary checkout per Run.** Rejected: it pays the third curve on every Run rather than
  once, and a crash still leaves a directory, now one nobody would think to look in.
- **A sparse checkout of only the paths a Run touches.** Rejected: which paths a Run reads is not known
  before it reads them — a backward scan through date partitions is a search, not a fetch list.

## Consequences

- **The local ref is `refs/heads/hyper-store`**, not a private namespace. `hyper` needs no privacy here —
  the whole thesis is that the record is in the open — and a private ref would mean a human's checkout has
  to fetch a branch their clone already holds.
- **`hyper`'s own local state lives under `.git/hyper/`.** The lock §6 states and any derived state that
  makes a Head lookup or a backward scan faster. §7's *any index is local, ignored by git* was a claim
  about ignore rules, and under `.git/` there are none to make: git ignores that path by construction
  rather than by a rule anybody wrote. The noun goes with it — "index" already carries a second sense
  across §8 and §12, and `git`'s own index is now a third.
- **The push retry is a path-set union, and the claim it makes gets stronger.** §7 hedged that *almost
  every one of those rebases is trivially clean*, which is an observation about how git's merge happens to
  behave. Re-applying a known path set onto a fetched tree cannot collide where the sets are disjoint, and
  every path carries the id of the Run that wrote it, so the only unclean case is the one §7 already
  names. `git rebase` is unavailable on this path anyway, needing a worktree.
- **The surviving conflict is detected exactly rather than textually.** Two Runs closing one open entry
  write the same two paths — the in-flight Step file at the next `<nnnn>` and `outcome.json`, whose
  `closed_by_run` and `ended_at` always differ. In a worktree that left conflict markers inside a file
  specified as canonical JSON, which was never a resolution. It is a same-path write, the losing Run is
  `failed` with `75` having pushed nothing, and what the Store ends up holding is a question about
  evidence rather than about materialisation.
- **The case-fold check stops being a rule the write could contradict.** §7 grounded
  `record-identity-collision` in a laptop's case-insensitive filesystem against a runner's. `hyper` now
  never puts a Record's identity on a filesystem at all — a git tree entry is a byte string, case-sensitive
  everywhere — so *never by attempting the write and seeing what happens* is structural rather than
  forbidden: the write always succeeds. The check survives untouched, because a human's checkout of the
  branch on a case-insensitive laptop is precisely the reading §7 promises, and that is now the only place
  a filesystem sees these names.
- **§12's sets do not move.** A local object write that fails is not a guardrail declining before a call
  goes out, which is what every `error_code` is (ADR-0072), so it is a halt or an ordinary `failed` and
  mints nothing. The set held because a rule about what a member *is* made the question unaskable.
