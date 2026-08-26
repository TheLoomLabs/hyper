# Three invocations, one repository, opposite answers

This corpus is §9's exemption paragraph as a fixture (ADR-0020, issue #105).
`repo/` pins 9.9.9 and the binary the three cases beside it are handed is
1.4.0, so:

- `check/` — the command inside the tree of sixteen — Refuses
  `version-pin-mismatch` at 77, with stdout silent and the Refusal naming both
  versions and both remedies.
- `version/` and `completions/` — two of the three commands outside it — exit
  0 with their whole output on stdout and nothing on stderr. The third is
  `mcp`, which this fixture does not drive and never will: starting the server
  resolves no repository, so it is not a fourth exemption and there is no
  invocation here to contrast (ADR-0088). What the gate compares against this
  pin on that surface is each **tool**, at the moment it resolves a repository,
  and the case for that is a `call` against this repository — which lands with
  the paths that decline (issue #196).

The difference between them is the exemption and nothing else. That the gate
Refuses is already proven six times over in `../check/version-pin-*`; what is
proven only here is the contrast.

The three are ordinary cases, driven by the one harness in `golden_test.go`
like every other case under `testdata/`, and they share the one repository
rather than each carrying a copy of it. All three stand in it — each names
`../repo` as its `wd`, which is the *one working directory* half of the claim —
and only `check/` names it as a repository, with the `--repo-dir .` an operator
standing there would type. The other two name none, and the entry points behind
them are handed neither a working directory nor an environment to find one
with: the exemption is not a branch they take but a repository they cannot
reach, so the `wd` they stand in is one they never ask for.

`check/` spells the repository out rather than letting root resolution walk up
to it because a checked-in fixture cannot carry a `.git`, and a walk from here
would climb straight past `repo/` and resolve hyper's own repository — proving
something about this tree rather than about the pin.

`check/facts.json` and `version/facts.json` are the same build, which is what
makes the contrast one binary's and not two: the version the Refusal quotes as
*this binary is* and the version the page states on its first line are two
readings of one constant (§11, issue #103). Change one and change the other.
