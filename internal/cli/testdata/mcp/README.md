# The second surface, driven as cases

This corpus is §9's MCP server as a fixture (ADR-0088, issue #195). A case here
holds a `call` where every other case holds an `argv`:

```json
{"tool": "providers", "arguments": {}}
```

and its golden is `envelope.golden` — the whole return envelope, as it came
back off the wire — where every other case holds a `stdout.golden`, a
`stderr.golden` and an `exit.golden`. Everything else about a case is
unchanged: the same `repo/`, the same `env`, the same `now`, `mint`, `actor`,
`hostname`, `bin/`, `serve/` and git-fixture inputs, read by the same harness
in `../../golden_test.go`.

**The call is real; only the client is in-process.** The case is driven through
the server over the SDK's in-memory transports, so the handshake, the framing
and the JSON of every row are the wire's — the same principle
`golden_serve_test.go` states for the TLS fixture, where the call is real and
only the name resolution is a fixture.

**A case names its repository in the environment and never in its arguments.**
§9 is flat about it — *no tool takes an override argument of any kind, under
any name* — so there is no `--repo-dir` to splice into a call the way the
harness splices one into an argv. A case that carries a `repo/` has
`HYPER_REPO_DIR` pointed at it; a case that shares a repository writes the
variable itself, in its own `env`, which is what the two `five-artefact-demo`
cases do.

The subtree is one directory per **tool**, named as §9 names it, with one
directory per case beneath — the same convention every command's corpus keeps,
and the same fence holds it (`TestGoldenCorpora_ACasesDirectorySaysWhichCommandItExercises`).
A tool is named for the command it carries, so `providers/` here and
`../providers/` are the two surfaces over one command and are meant to be read
against each other: the rows in an `envelope.golden` are the rows in the
`--json` twin's `stdout.golden`, and a fence holds them to it
(`TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites`).

`providers/truncated/repo` carries fifty Manifests, which is one more Provider
than the default limit admits once the built-in is counted. That is the only
way this surface can reach a truncated result at all: `providers()` takes no
arguments, the `--limit` its command carries is not offered here, and the cut
is therefore the default's. What comes back is fifty rows, `truncated: true`
carrying the bare boolean — a namespace listing having no axis to name — and a
text block that says so, there being no stderr on this surface for the line the
CLI writes beside its table.

The paths that decline — a Refusal, a name matching nothing, an argument the
schema does not admit — have no case here yet; they are issue #196's.
