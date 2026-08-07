# `hyper` never updates itself

`hyper` has no self-update, and it never checks whether a newer version exists. Upgrading is three
acts a human performs in the open: install a new binary, run `hyper project`, review the diff.

We chose this because ADR-0004 removed the code from Providers, which leaves `hyper` as the only code
that runs — so replacing the binary replaces every guardrail at once. A self-updating binary changes
Expansion ordering, Bound checking and Repeatability evaluation between two Runs with no artefact
edit, which is the precise state ADR-0001 exists to make unreachable. It is also, exactly, a program
that fetches code over a network and executes it; the objection to that is why this project exists
rather than a footnote to it.

The update *check* is a separate act and dies separately. It carries nothing about a Run and would be
easy to wave through as a courtesy, but it is egress performed on nobody's behalf by a tool that
otherwise reaches the network only where a reviewed artefact asked it to. ADR-0016 declined telemetry
on that ground and this is the same ground.

## Consequences

- **There is no `hyper upgrade` and no `hyper self-update`.** The sixteen commands stand; the upgrade
  path is a package manager, a release download, or `go install`, none of which is `hyper`'s
  business.
- **Learning that a version exists is not `hyper`'s job.** Whether anything in this project ever
  speaks first is the notification question, and the answer there does not get to smuggle a version
  check in beside it.
- **A stale pin is silent.** Nothing in the tool will ever tell you that you are three releases
  behind. That is the accepted cost of both halves together, and it is cheaper than a binary that can
  change itself.
