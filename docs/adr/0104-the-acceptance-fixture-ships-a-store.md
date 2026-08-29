# The acceptance fixture ships a Store

**`scripts/acceptance/run.sh` runs `hyper store init`, and the repository every sealed session is
handed has a Store.** That was already the line the script ran; what it did not have was a reason, and
issue #221 booked one — *the Store's presence in the fixture is a decision, not an oversight, and it is
settled inside the ticket that first needs a Run*. This is that ticket's answer (issue #223).

**Nothing else in the fixture changes.** `providers/` is still absent, the Target still grants
`read, mutate, destroy`, and the quickstart shape is still the quickstart shape.

## The wall and the deliverable are not both available

A repository with no Store refuses every Run `store-absent`, remediation *hyper store init*. An agent
cannot take that remediation. The orientation puts `store init` on the far side of the line it draws
with `install` and `compact`, in as many words: *`store init` creates the record. Creating it is the
human's act; your part is to say it has not happened.* And then, of all three: *Do not reach around
this. Where a call Refuses naming one of the three, say so and stop; the human runs it.*

So a Store-less fixture and a task whose deliverable is a Run compose into a task whose **correct**
completion is a Refusal on the first call. The transcript it produced would be a good one and a useless
one: it would say that the agent read one row and obeyed it, and it would reach none of Execution, the
Record, or the branch that holds them — §6, §7 and the Store, which no transcript has ever reached and
which are the whole of why a run-capable task is being written. There is no version of *keep the wall*
under which the Run happens.

The fixture is also **one repository shared by every task**, materialised by one script from one shape.
Keeping the wall is therefore not a property of the task that wants it; it is a first row levied on
`fleet-rollout`, on the second-Run task behind it (issue #224), and on every run-capable task after
that, each paying a sealed session's opening call to be told the same thing.

## What it costs, stated plainly

The wall would have bought exactly one measurement, and it is #221's fourth user story: whether an
agent reads `store-absent`'s own remediation and says so, or reaches around it. That is a real
question — reaching around a Refusal is the failure mode ADR-0001 exists about, and the orientation
names all three of the human's commands rather than calling them absent from this surface precisely
because the agent on the other surface holds a terminal (`internal/mcp/instructions.go`).

It is also a **one-row** question. It is answered by the first tool call of any run-capable task, it
tells you nothing further, and under this harness the price of asking it is a whole sealed session that
reaches nothing else. If it is worth asking it is worth a task of its own, and that task wants the
fixture *without* a Store, which is a switch `run.sh` does not have and would be a few lines to grow.
It cannot be a task's own `.setup.sh`: the setup script runs before `git init` and before `store init`,
so a task can add to the shape and cannot un-initialise what the script has not built yet. Growing the
switch is deferred until something asks for it, on the ground that a knob with no caller is a knob
nothing holds true.

The realism spent is smaller than the phrase *the quickstart shape* makes it sound. `store init` is run
once, by a human, and the repository has a Store for the rest of its life. **The initialised repository
is the representative one**; the fresh one is the first minute.

## What was considered

**Keep the wall and let the first run-capable task spend its opening call on it.** Refused above: the
agent is instructed not to take the remediation, so the call is not an opening but an ending.

**Keep the wall and let the *harness* run `store init` between the refusal and the Run** — the human's
act performed by the human's script, mid-session. Refused: there is no such seam. `run.sh` starts one
headless session with one prompt and reads its transcript afterwards; a human in the loop is the thing
a headless run does not have, and simulating one would put the harness inside the measurement.

**Say so in the task text instead** — *the Store is not there; tell me and stop*. That is a legitimate
task and it is the one-row task above, not this one.

**Grow the switch now, defaulting to a Store.** Refused for now as unbuilt configuration: the two tasks
in flight both want the Store, and a flag whose only value is the default is a claim nothing checks.

## Consequences

- **`run.sh` states this beside the line**, next to the shape it builds, so the next reader of the
  script does not have to find this file to know why the Store is there.
- **A run-capable acceptance task is now writable**, which is what
  `scripts/acceptance/tasks/fleet-rollout.md` is (issue #223) and what issue #224 is written against.
- **`store-absent` is unreachable in a sealed transcript** until something grows the switch. Nothing in
  the suite or in the harness measures whether an agent reaches around it; that gap is recorded here
  rather than left to be rediscovered.
- **The fence reaches the pin and stops there.** `TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse`
  runs the setup half for every task, and `store init` Refuses on an unpinned repository, so a fixture
  that lost its pin fails the harness's own script rather than the first Run of a session nobody is
  watching. It says nothing about the commit: `store init` writes a parentless commit straight to the
  object database and does not read `HEAD` (`internal/store/store.go`), so it succeeds in a repository
  with no commit at all. What a lost `git commit` would break is a Run's `repo_revision`, one gate
  later, and nothing here catches it.
- **The approval the orientation withholds is granted in the task text.** An agent is told to stop and
  hand the diff to a human before running, and there is no human in a headless session — so a task
  whose deliverable is a Run says in its own words that the approval is given. The prompt is the human
  speaking, which is the one place in this harness a human is.
