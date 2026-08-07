# `hyper` has no configuration files

There is no `~/.config/hyper/`, no repository `.hyper.yaml`, and no per-project settings file of any
kind. Configuration is three layers — **flags → environment → defaults** — and governs presentation
only: `--repo-dir`/`HYPER_REPO_DIR`, `--no-color`/`NO_COLOR`, and `--json`. Repository root
resolution walks up from the working directory, bounded by the git root.

We chose this because, by the time the question was asked, nothing load-bearing was left to put in a
file. Credentials are environment-resolved and never written anywhere (ADR-0007); concurrency caps
and deadlines are Manifest-declared with no override; retention lives in a reviewed artefact
(ADR-0011); Cadence lives in the Procedure (ADR-0005). Swamp's documented four-layer model — repo
file, user file, environment, flags — is a good model for a tool that has settings, and the honest
answer here is that the settings were removed one ticket at a time rather than that the ordering was
wrong.

The two file layers are worse than redundant. A **user-level** file makes `hyper` behave differently
on one machine than another, which is the environment-as-authority-axis that the safety model
deleted: the guardrails are identical on a laptop and on a runner precisely so that a laptop can test
them. A **repository-level** file is an unreviewed artefact competing for authority with the reviewed
ones, at which point "the artefact is the review surface" acquires a footnote.

## Consequences

- **Nothing `hyper` reads can change what a Run does except a reviewed artefact and the
  environment's credentials.** This is the property the decision exists to protect, and it is why
  deleting two layers is a stronger answer than ordering five.
- **Every setting that survives is invisible to the outcome.** Colour, width and output format cannot
  change what reaches the world, so their precedence is uninteresting by construction.
- **There is no place to put a future setting**, and that is deliberate friction. A setting that
  matters belongs in a reviewed artefact; one that does not belongs in a flag. Anything that fits
  neither is a signal that the design went wrong somewhere earlier.
- **Ergonomic cost, accepted:** a repeated `--repo-dir` cannot be made sticky, and there is no way to
  set a personal default for anything. `HYPER_REPO_DIR` and a shell alias are the whole answer.
