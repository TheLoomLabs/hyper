# When a repair obliges a sealed re-run

The suite cannot answer the question the acceptance harness exists to ask, and this file is the rule
that follows from that (issue #250).

`cmd/hyper/acceptance_test.go` asserts properties of the **harness** — the seal held, `providers/` is
absent, the repository checks clean, `AGENTS.md` is this binary's orientation. Issue #221 fixes that
boundary deliberately: **nothing in the suite may assert what an agent did.** So a repair to a
sentence an agent reads is verified by the suite as *text that changed*, and by nothing else.

Issue #241 is the worked example. An agent read `UNBOUNDED`, made the one edit that clears the flag
without touching what the flag was about, and explained its reasoning in its report. No reading of
the diff would have found that. A transcript found it in one line.

## The rule

**A repair is *taught* or it is *enforced*, and only the taught kind obliges a run.**

- **Enforced** — the binary now Refuses, a schema now declines, a rendering now carries a field it
  did not. `check`, the goldens and the package cases hold it. No run is owed.
- **Taught** — the change's whole subject is what an agent *reads and then decides*: a clause in the
  orientation (`internal/mcp/instructions.go`), the wording of a `check` row or a Refusal, a
  `review` rendering, a tool description or the shape of its structured output. **Nothing in the
  suite can fail if this repair does not work.** A run is owed.

A repair that is both is taught: the enforcement is fenced, the teaching is not.

## What the ticket that lands a taught repair must do

**Name the run it owes, and either buy it or record why not.** These runs are a handful a year and
each costs a session and real money (#221, #250) — so the obligation is to *decide*, in writing, not
to spend.

- **Which task.** The one whose transcript produced the repair. Where the repair came from somewhere
  else, the task whose surface it touches; where none does, say so — that is a gap in the task set,
  and adding a task file is the whole of what fencing it takes (#222).
- **When.** After the repair lands, not before: the run is the measurement of the repair, and a run
  against the surface being changed measures the old one.
- **Deferring is allowed and is not silent.** *This repair is taught, the run it owes is
  `<task>`, and it is not being bought because …* is a complete answer. Several deferred repairs
  landing on one task is itself the argument for buying that task next.

## Reading a run back

Every prior run was read into an ADR, and that stays the shape. Two things the read must state:

- **What the agent did, off the transcript rather than off the repository it left.** The order of
  the calls is the evidence; the diff is the outcome.
- **Whether the repair landed, plainly.** A run where the taught clause did its job and nothing new
  appeared is a **good result and the first evidence the repair has** — it is not a wasted run and
  it is not written up as one. ADR-0120 is the worked example of a positive result stated as one.

## Running one

`CONTRIBUTING.md`'s *The acceptance harness* section owns the how, and
`scripts/acceptance/run.sh`'s header owns what the seal covers and why. Put the output directory
outside the checkout and outside its parent — the script refuses otherwise, and a previous run's
directory is covered by search rather than by memory, so old ones may stay where they are.
