# The projection's executor is compiled in, never authored

The runner label, the `actions/checkout` commit, and the release-artefact and checksums URLs the
generated workflow carries are constants inside the binary. No artefact declares one, no flag supplies
one, and the Repository declaration does not admit them. They move only when the binary moves, which
means only through the three acts ADR-0020 already fixes: install, `project`, read the diff. Four
constants, stated as a set in §11, and that set is the whole of what a `hyper` release can change in a
repository that edited nothing.

We chose this because `runs-on` is the environment-as-authority axis in its most plausible disguise.
ADR-0020's own admission criterion appears to invite it — a runner label governs the repository as a
whole and belongs to no Procedure, Definition, or Target — and every CI-adjacent tool in existence
makes it a setting, so it is the reading an implementer reaches unaided. What it buys is two
repositories against one Store naming two platforms and fetching two differently-built binaries, which
is precisely the skew the pin exists to prevent, arriving through packaging rather than through policy.
It also brings back the digest table §11 refuses: one pin means one artefact only while the platform is
fixed.

## Considered options

- **`runs-on` authorable in the Repository declaration.** The reading ADR-0020 leaves open. Rejected:
  it reinstates the axis the safety model deleted, multiplies the frozen digest into a table, and makes
  *which bytes ran* answerable only by reading a field beside the pin rather than the pin.
- **`runs-on` and the artefact platform authorable separately.** Honest about self-hosted runners,
  whose label says nothing about their architecture. Rejected twice over: for the same reason, and for
  handing an author two facts that must agree with nothing to check the agreement against — the
  disagreement would surface as a digest mismatch on a runner rather than as a Refusal on a laptop.
- **Stamping the release digest into the binary at build time**, so `project` reaches the network
  never. Attractive against a project that dislikes fetching. Rejected because it deletes
  `release-artefact-absent` and the trust-on-first-use paragraph ADR-0020 argued for, and because a
  binary carrying the digest of a tarball that is not itself is a cross-stamp no reviewer can check
  against anything.
- **Dropping `actions/checkout` for a generated `git` clone**, on the ground §11 uses to refuse a setup
  action. Rejected because the two acts are not the same one: a setup action decides which bytes execute
  as `hyper`, and `checkout` does not — the digest in the reviewed file decides that. The action is
  pinned by commit, while `runs-on` names an image GitHub rebuilds continuously and that image supplies
  the shell the install step runs in, so refusing the stricter pin while trusting the looser one is
  straining at a gnat.

## Consequences

- **The trust boundary is the runner.** It is drawn there because the executor is trusted to execute at
  all, and what stands inside it is the refusal to let anything but a digest choose the binary. That
  sentence is now stated in §11 rather than left to be inferred from a workflow example.
- **`hyper` schedules on GitHub-hosted x86-64 Linux or not at all.** Self-hosted, ARM, and pinned-image
  runners have no projection. What stands in for one is `hyper run` invoked by that executor's own
  clock, which records `local` and `manual` because it reads its own environment — so the Journal
  cannot say a clock fired it. Stated as a non-goal in §13.
- **The set is countable, and being countable is the point.** Everything else in a generated workflow —
  the recurrence, the job name, the `env:` block, the Bound behind every Step — derives from something
  somebody wrote and reviewed. Four constants do not, and a reader can hold four.
- **No constant can move without the version moving.** A constant lives in the binary, a differing
  binary is a differing version, and the file carries the version in four places already, so
  `projection-stale` fires on no occasion the pin was not firing on anyway. A third-party action's pin
  arriving in a diff a human had to read regardless is a supply-chain fact arriving free.
- **A third party's clock is now something a repository waits on.** An `actions/checkout`
  vulnerability, or a retired runner image, is repaired by a `hyper` release and by nothing else, since
  a hand-edit to the generated file is caught like any other. It joins §13's ceiling as the one victim
  there that is waiting on somebody outside this project.
- **`persist-credentials: true` is written out.** The projection states the dependency rather than
  resting on the action's default, and what the dependency buys is that `hyper` never holds the Store's
  credential at all: it inherits an authenticated remote and pushes through it, so there is no slot, no
  resolution, and nothing to suppress in a rendering.
- **`project` reads a checksums file, not the artefact.** Both are the same mutable source read in the
  same instant, so hashing bytes this machine will never execute buys nothing; `release-artefact-absent`
  gains its three shapes — no release, no checksums file, no line for the artefact the template names.
