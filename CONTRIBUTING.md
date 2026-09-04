# Contributing to `hyper`

Most of what you need to know is not in this file. It is in
[`docs/spec/`](docs/spec/), which is the authority, and in
[`docs/adr/`](docs/adr/), which is why. This file is the conventions that live
nowhere else: how to build it, how to run the suite, and what a change is
expected to look like when it lands.

## Build and test

```bash
go build ./...
go vet ./...
go test ./...
```

Go 1.25 or newer (`go.mod` carries the directive).

A binary built this way names no version at the link step, which is correct and
is not a bug — the suite builds without the flag throughout. What such a binary
*reports* is the version Go recorded for the commit it was built from, marked
`+dirty` while you are mid-change and a pseudo-version between releases
([ADR-0138](docs/adr/0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)),
and no repository's pin names either. So when you want to *use* `hyper` rather
than test it, stamp it: [`docs/build/releasing.md`](docs/build/releasing.md)
owns the invocation.

**`internal/cli` is ~110s on its own; every other package is seconds.** It is
the golden corpus, and it drives every case under `testdata/` through
`cli.Main`. While you are working inside one command, run its cases rather than
the package:

```bash
go test ./internal/cli -run 'TestGolden/check/'
```

Run `go test ./...` whole before you open a change.

## What runs it besides you

[`.github/workflows/suite.yml`](.github/workflows/suite.yml) runs `go build`,
`go vet` and `go test` over everything, on every push to `main` and on every
pull request, and [`release.yml`](.github/workflows/release.yml) calls that same
file and waits on it — so a tag cannot publish a tree the suite fails on
([ADR-0123](docs/adr/0123-the-suite-is-run-by-a-machine-and-a-prepared-machine-may-not-skip.md),
[#243](https://github.com/TheLoomLabs/hyper/issues/243)).

The runner installs `bubblewrap`, because the acceptance seam below needs it,
and the test step sets **`SUITE_PREPARED=1`**. That variable is one claim: *the
tools are installed and a namespace can be built here*. The gates in `cmd/hyper`
that would otherwise skip — a tool missing from `PATH`, a `bwrap` that cannot
build a namespace — fail instead when it is set, because on a machine that
promised the tool, its absence is a defect in the promise. Leave it unset
locally, where a skip is the honest answer: a suite that failed on a machine
without `bwrap` would be asserting something about the machine.

## The golden corpus

Goldens are checked in and regenerated behind **one flag**:

```bash
go test ./internal/cli -update
```

One flag serves every corpus. A corpus that regenerated behind a switch of its
own would be one a `-update` run silently left stale.

The conventions the harness enforces, so that you meet them as a fence rather
than as a surprise:

- **`testdata/` is directories, and a case's directory says which command it
  exercises.** A case sits in a directory named for its command — as
  [§9](docs/spec/10-surfaces.md) names it — or one directly beneath one, which
  is how `check/clean/` and `mcp/providers/<case>/` are both in order. The
  harness does not need this; it reads the command out of the argv. It is a rule
  for readers, which is why
  `TestGoldenCorpora_ACasesDirectorySaysWhichCommandItExercises` holds it.
- **A case is an `argv` or a `call`, and never both.** An `argv` is a command
  line; a `call` is an MCP tool and its arguments. That is the two surfaces over
  one core arriving in the corpus — everything else about a case is the same
  input read the same way.
- **Everything a Run would read off the machine is supplied by the case.** The
  clock (`now`), the Run ids (`mint`), the actor and the hostname, the fixture
  repository (`repo/` or `repo-from`), the environment (`env`). A case that
  reached the machine for one of them would be a case that fails on somebody
  else's.
- **A corpus documents itself.** Most subtrees carry a `README.md` saying what
  they drive and why the fixtures are shaped as they are; write one when you add
  a subtree.
  [`internal/cli/testdata/mcp/README.md`](internal/cli/testdata/mcp/README.md)
  is the worked example, and
  [`internal/cli/testdata/run/README.md`](internal/cli/testdata/run/README.md)
  is the longest.

Regenerating is not review. `-update` will happily write a wrong answer into a
golden; read the diff it produces the way you would read any other.

## The acceptance harness

The suite says whether the code does what the spec says. It cannot say whether
an **agent** can author a correct artefact against the surface, and that
question is answered by transcripts:

```bash
scripts/acceptance/run.sh scripts/acceptance/tasks/<task>.md /somewhere/outside/the/checkout
```

It builds a stamped binary, materialises a repository in the README's quickstart
shape with `providers/` absent, writes its `AGENTS.md` with `hyper project`, and
runs one headless agent session against it — **inside a mount namespace where no
`hyper` source checkout exists**. That last part is the point
([ADR-0099](docs/adr/0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md)):
three earlier runs left the repository and read `docs/spec/`, and a transcript
that was handed the specification records a success the shipped product cannot
reproduce, there being no `docs/spec/` on the machine a user installs `hyper` on.
The script asserts the seal and stops if it does not hold.

The output directory has to be outside this checkout, since the checkout is what
the seal hides. `TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse`
runs the setup half in the suite, so the harness cannot rot between the handful
of runs a year it is used for.

**The output directory is covered too**, so a setup script may leave in it what
it likes: a `--tmpfs` goes over it, and what binds back on top is the
repository, `bin/hyper`, `mcp.json`, and any path a task's `endpoint.env` names
inside the directory
([ADR-0109](docs/adr/0109-the-seal-covers-the-output-directory-the-harness-writes.md)).
`bin/hyper` and `mcp.json` cannot be hidden and never could: the MCP server is a
child of the sealed session, which reads `mcp.json` from inside the namespace
and execs the binary it names — so what the harness claims is the narrower *no
source checkout, no second binary, no fixture internals*. Everything else it
writes there — the fixture's binary, its credential, the reports, the logs and
the transcript being written as the session runs — is not reachable from inside.

**A previous run's output directory is covered as well**, found by searching for
the `mcp.json` this harness writes rather than by remembering a path, so a
machine that has run the harness before does not hand the next session an older
run's answer key. If you keep transcripts around, they stay where they are and
the harness works around them.

**And `$HOME` is covered wholesale**, with three paths bound back on top of the
tmpfs: the Claude Code binary, the credential it authenticates with, and a
`.claude.json` carrying the onboarding flag
([ADR-0130](docs/adr/0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md),
[#257](https://github.com/TheLoomLabs/hyper/issues/257)). The cover used to be a
list of the places this project's text collects, and what a list cannot keep up
with is the material an attended session leaves behind — a throwaway repository,
a transcript, a by-hand answer to the task about to be run, which several
tickets now require somebody to produce. So the seal names what to keep instead.
**Leave that material wherever you like**: nothing about where it sits is
load-bearing any more, and the harness neither reads it nor refuses because of
it.

**A repair to what an agent reads owes a run.** The suite asserts what the
harness did and never what an agent did (#221), so a clause added to the
orientation, or a `check` row or `review` rendering reworded, is fenced by
nothing here — the ticket that lands one names the run it owes and either buys
it or writes down why not
([`docs/agents/acceptance-re-runs.md`](docs/agents/acceptance-re-runs.md),
[#250](https://github.com/TheLoomLabs/hyper/issues/250)).

**A task is fenced by existing.** That test ranges over every task file in
`scripts/acceptance/tasks/` rather than naming one, so adding a task file — and
the `.setup.sh` beside it, which is part of the same artefact — is the whole of
what fencing it takes. A setup script that fails, or that leaves a repository
`hyper check` does not call clean, fails the suite under the task's own name.
**Commit the setup script executable**: `run.sh` runs it only if it is, and one
committed without its bit is skipped in silence, which the fence asserts against
rather than inherits.

**The fixture has a Store**, so a task may ask for a Run
([ADR-0104](docs/adr/0104-the-acceptance-fixture-ships-a-store.md)). One that does
has to **grant the approval in its own text**: the orientation tells an agent to
stop and hand the diff to a human before running anything, and the prompt is the
only human a headless session has.

**A task may bring a service with it.** `run.sh` hands the setup script the output
directory beside the repository, and reads two files back out of it afterwards:
`endpoint.pid`, whose process it kills in a trap on the way out, and
`endpoint.env`, whose `NAME=value` lines it folds into the environment the MCP
server runs with. That is how the three lookout tasks reach an HTTPS endpoint from
inside the seal — a local TLS server built from `scripts/acceptance/lookout`,
trusted through `SSL_CERT_FILE` in the `hyper` process's environment and through
nothing any artefact could name
([ADR-0105](docs/adr/0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)).
That service takes its initial state by `-fixture <name>`, so a task that needs
its own fiction gets one and observes no other task's world; the states and the
argument for each are in `scripts/acceptance/lookout/api.go`, and the
documentation every one of them installs into the repository is the one file
`scripts/acceptance/lookout/api.md`.
`endpoint.env` is a fixture's other half, and one task empties a line of it
rather than filling it: `monitor-coverage-empty-credential` is
`monitor-coverage`'s setup run as it stands with `LOOKOUT_API_TOKEN` exported to
nothing, which is the one arrangement that puts an agent in front of the third
member of credential presence
([ADR-0145](docs/adr/0145-an-empty-credential-is-its-own-refusal-on-both-surfaces.md),
[#268](https://github.com/TheLoomLabs/hyper/issues/268)).
The lifetime is `run.sh`'s rather than the setup script's because the fence runs
the setup half on every `go test ./cmd/hyper`. A **path** `endpoint.env` names
inside the output directory is bound back through the seal, since the process
that opens it is the sealed session's own child; that is how `SSL_CERT_FILE`
reaches a certificate the cover would otherwise hide.

## The spec is the authority

Where the code and [`docs/spec/`](docs/spec/) disagree, **the spec is right and
the code is the defect.** The same holds for the process document that sits
beside it — [`docs/build/releasing.md`](docs/build/releasing.md) says so of
itself.

`docs/spec/` plus `docs/adr/` plus [`CONTEXT.md`](CONTEXT.md) are about 270k
tokens. **Never run a tool over `docs/spec/` in one pass** — it does not fit,
and the sections are layers where any one change is a slice through all of them.
Read the sections your change touches, and the ADRs those sections cite.

Use [`CONTEXT.md`](CONTEXT.md)'s vocabulary in anything you write — an issue
title, a test name, a doc comment. Each term lists the synonyms it deliberately
avoids; drifting to one of them is how two names for one thing get started.

## ADRs

Decisions live in [`docs/adr/`](docs/adr/), numbered, one per file, each stating
what was decided, why, what was considered, and what it costs.

Two kinds of edit, and they are not the same edit:

- **A factual error is corrected in place, unmarked.** An ADR is a record of a
  decision, not a record of what was once believed about the facts around it.
- **A genuine change of decision gets an amendment marker** on the passage it
  revises — `_ADR-00NN amends this:_` — with the original left standing. See
  [ADR-0007](docs/adr/0007-hyper-never-stores-a-secret.md) and
  [ADR-0053](docs/adr/0053-an-opaque-destroy-names-its-population.md) for the
  shape.

If a change you are making contradicts an existing ADR, say so explicitly rather
than overriding it quietly.

## Doc comments carry the why

The house style is not decoration. Read almost any file in `internal/` before
you write one.

A doc comment says **why the thing is shaped as it is** — not what it does,
which the code already says. It cites the spec section it implements and the
issue it arrived under, and where a decision had a losing alternative it names
it, so the next reader does not re-derive the choice.

**A comment asserting a property the code does not hold is a defect, not a
nit.** These comments are load-bearing: they are how a reader who cannot hold
270k tokens finds out that this function is the one place a rule is spelled, or
that a signature is narrow on purpose.

## Commit messages

A conventional-commit prefix, a scope naming the packages touched, and the issue
number in the subject:

```
feat(mcp,cli,spec,adr): progress at the Step boundary, and the cancelled call (issue #202)
```

The body explains **the decision, not the diff**. What was chosen, what it was
chosen over, and what a reader would otherwise have to reconstruct from the code.
`git log` is the third place the why is written down, after the ADRs and the doc
comments, and it is the one that is attached to the change itself.

Commit messages end at the body — no trailers.

## Issues

Issues live as GitHub Issues in
[`TheLoomLabs/hyper`](https://github.com/TheLoomLabs/hyper/issues), managed
through the `gh` CLI. The label set and what each label means are in
[`docs/agents/triage-labels.md`](docs/agents/triage-labels.md).

## Reporting a vulnerability

Not here. See [`SECURITY.md`](SECURITY.md).
