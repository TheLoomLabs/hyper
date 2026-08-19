# Five commands, one repository

`repo/` is milestone 1's five-artefact demonstration repository — §3's own worked
artefacts, two Manifests, two Target declarations, two Definitions, a Procedure
and a `hyper.yaml` (issue #99). Milestone 1 proves it checks clean. Milestone 2
asks it the four questions §9's **Discovery** and **The repository** sections
state, in the order an agent would ask them (issue #116):

```
hyper providers                                   → cloudflare-dns, shell, uptime
hyper provider cloudflare-dns                     → Authorization: Bearer <secret>, and its Operations
hyper operation cloudflare-dns delete_dns_record  → the Manifest's own lines, and bound: mandatory
hyper targets                                     → cloudflare-prod, its credential slot absent; local, two hosts, read
```

An agent that has never opened a `providers/` file learns from those four which
Provider to name, which Operation it exposes, what a call to it needs, and what
the repository will let it reach — offline, credential-free, and without a single
HTTP request existing in the codebase. Each of the four runs in both the human
and the `--json` mode, and all eight exit `0` with their answer on stdout and
nothing on stderr.

Milestone 3 asks it the fifth question — *what is about to be approved* — of the
artefacts in it and of the one Provider that has no file anywhere (issue #118):

```
hyper review procedures/retire-preview-dns.yaml  → the Procedure under its header, marked in the gutter
hyper review preview-dns                         → the same, resolved by the name it declares
hyper review hyper.yaml                          → the artefact a path is the only way to reach
hyper review shell                               → the bytes compiled into the binary
```

Every way that positional can fail lives beside them — a name matching nothing, a
name differing only in case, a path matching nothing, the pseudo-path no caller
can type, no positional and two, and a flag `review` does not have — and so does
the pin gate firing ahead of a bad name. All of them read this same repository
and none carries one of its own.

## Why the repository sits here and not in a case

It is one repository and twenty-nine cases, in six corpora: the four commands'
eight above, `check/`'s two clean cases, the two beside `targets`'s own that run
it again under an environment supplying `CLOUDFLARE_API_TOKEN` — where the
credential column reads present and nothing else in the answer moves — and
`review/`'s seventeen.

A copy per case is how it began, and a Provider whose Operations moved under one
command's golden file and not another's is exactly the drift that would have
followed — eight files to change in step, and no test that would notice if seven
of them changed. `review/` is what that copy would have cost: it renders these
files byte for byte, so a fixture edit that did not reach every copy would show
up as a review of an artefact no other command could see.

So the fixture belongs to no command, which is why it is not under any command's
corpus. The cases stay in the per-command subtrees #101 fixed, each named for the
command its argv invokes; what they share is the repository and not a directory.

## How a case names it

A case with a `repo/` of its own has `--repo-dir` resolved to it by the harness.
These carry none, so they name it themselves, with the `--repo-dir` an operator
would type — relative to the case's own directory, which is where a case's paths
are read from:

```
hyper providers --repo-dir ../../five-artefact-demo/repo
```

That is the shape `../exemption/` established for three invocations against one
repository (issue #105). The difference is that its three stand *in* the
repository and these do not: nothing here is a claim about a working directory,
so no case names one.

`check/five-artefact-demo-faulty/` is a different repository — the same five
artefacts with one deliberate fault in each — and stays where it is. Nothing this
side of it is its home.
