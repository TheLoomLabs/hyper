# Contributing to `hyper`

Most of what you need to know is not in this file. It is in
[`docs/spec/`](docs/spec/), which is the authority, and in
[`docs/adr/`](docs/adr/), which is why. This file is the conventions that live
nowhere else: how to build it, how to run the suite, and what a change is
expected to look like when it lands.

## Build and test

```bash
go build ./...
go test ./...
```

Go 1.25 or newer (`go.mod` carries the directive).

A binary built this way is unstamped, which is correct and is not a bug — the
suite runs against unstamped builds throughout. When you want to *use* `hyper`
rather than test it, stamp it:
[`docs/build/releasing.md`](docs/build/releasing.md) owns the invocation.

**`internal/cli` is ~110s on its own; every other package is seconds.** It is
the golden corpus, and it drives every case under `testdata/` through
`cli.Main`. While you are working inside one command, run its cases rather than
the package:

```bash
go test ./internal/cli -run 'TestGolden/check/'
```

Run `go test ./...` whole before you open a change.

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
