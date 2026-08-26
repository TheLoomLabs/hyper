# Three invocations and a tool, one repository, opposite answers

This corpus is §9's exemption paragraph as a fixture (ADR-0020, issue #105).
`repo/` pins 9.9.9 and the binary the four cases beside it are handed is
1.4.0, so:

- `check/` — the command inside the tree of sixteen — Refuses
  `version-pin-mismatch` at 77, with stdout silent and the Refusal naming both
  versions and both remedies.
- `version/` and `completions/` — two of the three commands outside it — exit
  0 with their whole output on stdout and nothing on stderr. The third is
  `mcp`, which this fixture does not drive and never will: starting the server
  resolves no repository, so it is not a fourth exemption and there is no
  invocation here to contrast (ADR-0088).
- `provider/` — a **tool** against this same repository, and the fourth case —
  answers §9's Refusal envelope: `isError: true`, no `outcome` key, and the
  whole rendering `check/` writes on stderr as its `text`, with the sentence
  saying a verbatim retry refuses identically after it (issue #196). That the
  two renderings are one is held by a fence rather than by this paragraph
  (`TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr`, which
  pairs a Refusal against every Refusal the corpus writes on stderr). What the
  gate compares against this pin on the MCP surface is each tool, at the moment
  it resolves a repository, which is what this case is here to show — the
  invocation that started the server passed no gate, and this call passes the
  same one `check` does.

The difference between them is the exemption and nothing else. That the gate
Refuses is already proven six times over in `../check/version-pin-*`; what is
proven only here is the contrast.

The four are ordinary cases, driven by the one harness in `golden_test.go`
like every other case under `testdata/`, and they share the one repository
rather than each carrying a copy of it. The three invocations stand in it —
each names `../repo` as its `wd`, which is the *one working directory* half of
the claim — and only `check/` names it as a repository, with the `--repo-dir .`
an operator standing there would type. `provider/` names it the one way §9
leaves a tool: `HYPER_REPO_DIR` in its own `env`, no tool taking an override
argument of any kind, under any name. `version/` and `completions/` name none,
and the entry points behind them are handed neither a working directory nor an
environment to find one with: the exemption is not a branch they take but a
repository they cannot reach, so the `wd` they stand in is one they never ask
for.

`check/` spells the repository out rather than letting root resolution walk up
to it because a checked-in fixture cannot carry a `.git`, and a walk from here
would climb straight past `repo/` and resolve hyper's own repository — proving
something about this tree rather than about the pin.

`check/facts.json`, `version/facts.json` and `provider/facts.json` are the same
build, which is what makes the contrast one binary's and not three: the version
the Refusal quotes as *this binary is*, the version the page states on its
first line, and the version the server announces itself at are three readings
of one constant (§9, §11, issue #103). Change one and change the others.
