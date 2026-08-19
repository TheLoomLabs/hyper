# Every mark the gutter carries, on one screen

`repo/` is the repository the marker column's own cases read (issue #120). It
exists because the five-artefact demonstration repository next door cannot show
what this one shows: that repository checks clean, and four of §8's marker
classes are only reachable on an artefact `check` has something to say about.

`procedures/subject.yaml` is the screen. Reading down its marker column is the
step table §8 says it is:

```
hyper review procedures/subject.yaml
```

| line | marker | what it is |
| --- | --- | --- |
| `targets:` | `envelope ✓` | the check that quantifies over every Step's `target:` at once |
| `enumerate` | `read     staging` | the Kind declared two directories away, and the Target bound |
| `make` | `mutate!  staging` | a `mutate` with no declared Bound — `check` is silent on it by design (§4) |
| `make-bounded` | `mutate   staging` | the same Operation, with a Bound |
| `retire` | `DESTROY  staging` | `destroy`, upper-case for the eye, and no `!`: an absent Bound there is `bound-missing` |
| `scrub` | `DESTROY  opaque  local` | an Opaque request, declared beside no Operation anywhere |
| `no-such-definition` | `unresolved` | the `definition:` names nothing |
| `no-such-operation` | `unresolved` | the `operation:` names nothing |
| `no-such-provider` | `unresolved` | the Definition resolved and its `provider:` did not |
| `nested` | `staging` | a nested invocation's transitive envelope, walked to any depth |
| `no-such-procedure` | `unresolved` | the same absence one level up |

Three of those four `unresolved` lines are `check`'s to report and none of them
is this surface's to decline: the review renders and exits `0`, which is what
ADR-0064 fixes and what the case's `exit.golden` holds.

`procedures/beyond.yaml` is the envelope mark's other state — a Step reaching a
Target its own Procedure never declared — and it exits `0` too. A review does
not run `check` (§9), so an artefact carrying `envelope-exceeded` renders like
any other and the mark is the whole of what says so.

The Definitions come in pairs — `things` beside `things-observed`, `commands`
beside `commands-destroy` — because a Definition observes or effects and never
both (ADR-0032), and a `read` Step and an effectful one against one Provider and
one Target are two Definitions rather than one.

`procedures/inner.yaml` is there to be invoked. It declares the envelope
`subject`'s `nested` line renders and nothing else.

## Why it is a repository and not a case

Three review cases read it — the Procedure rendered, the same as NDJSON, and the
envelope's exceeded state — on the argument `five-artefact-demo/` already
carries: a copy per case is a fixture edit that reaches one golden file and not
another, and this repository's whole subject is that the two surfaces render the
same facts. The cases stay in `review/`, named for the command their argv
invokes, and name this repository with the `--repo-dir` an operator would type.
