# The second surface, driven as cases

This corpus is §9's MCP server as a fixture (ADR-0088, issue #195). A case here
holds a `call` where every other case holds an `argv`:

```json
{"tool": "providers", "arguments": {}}
```

and its golden is `envelope.golden` — the whole return envelope, as it came
back off the wire — or `error.golden`, where the call was malformed and came
back as no envelope at all (below); every other case holds a `stdout.golden`, a
`stderr.golden` and an `exit.golden` instead. Everything else about a case is
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
(`TestGoldenCorpora_ARowInAnEnvelopeIsTheRowTheStreamWrites`). The pairing is
by the row's own identity across the whole corpus rather than by directory, and
one identity can have more than one rendering: `targets` computes credential
presence when it runs, so `cloudflare-prod` is `present: true` in a case whose
`env` sets the variable and `present: false` in one that does not. What the
fence holds is that an envelope's row is **one of** the renderings the stream
writes for that identity — keys in order, values as stated.

**A case that declines holds an `error.golden` where the others hold an
`envelope.golden`**, and which of the two it is says which half of §9's mapping
the case is about (issue #196). A JSON-RPC error is not an envelope with a bit
set, so there is nothing to lay out: the file is the message the command wrote
where a person would have read it on stderr, and nothing beside it — the code
is the SDK's own mapping of a handler error rather than a number hyper chooses.
`provider/name-matching-nothing` is the positional that matches nothing and
`providers/limit-is-not-an-argument` is the argument no schema admits; both are
malformed calls, and both are the CLI's exit `2` with no exit code to spend.

`provider/version-pin-absent` is the other side of it. A guardrail declining is
an **answer** to a well-formed call, so what comes back is an envelope carrying
`isError: true`, no rows, and the whole Refusal as `text` — with the sentence
saying a verbatim retry refuses identically, which is the only place the
protocol leaves for saying it (ADR-0001). The `version-pin-mismatch` half of the
gate is filed with the invocations it is contrasted with, in
`../exemption/provider`, against the repository the three of them share.

`provider/version-pin-mismatch-and-a-bad-name` is an ordering contract read
across the two surfaces. The gate fires before the positional is resolved
everywhere, so its twin in `../provider/` exits `77` and not `2` — and the same
call here is therefore a **Refusal envelope and not a protocol error**. A
mapping that read the positional first would invert it, and the two cases are
one claim written twice.

`providers/truncated/repo` carries fifty Manifests, which is one more Provider
than the default limit admits once the built-in is counted. That is the only
way this surface can reach a truncated result at all: `providers()` takes no
arguments, the `--limit` its command carries is not offered here, and the cut
is therefore the default's. What comes back is fifty rows, `truncated: true`
carrying the bare boolean — a namespace listing having no axis to name — and a
text block that says so, there being no stderr on this surface for the line the
CLI writes beside its table.

`operation/` is §9's third discovery question on this surface, and its cases are
the derived block's own rules rather than the command's (issue #197).
`five-artefact-demo` is a `destroy`: `bound: "mandatory"`, `patterns_resolved`
as `[]` where the Operation declares none, and the Record pair absent
altogether, a `destroy` projecting no Record of its own.
`a-paginated-read-over-a-series` is the other side of each — `bound: "none"`,
two Patterns named, `record_cardinality` and `record_identity` beside them —
and `a-builtin-shell-destroy` is the third member of a fact that is not a
boolean: an opaque `destroy`, whose Bound is `illegal` rather than absent.
`a-mutate-declaring-no-repeatability` is the effective value with no spelling in
the source at all, `run-once` derived from a Kind that omitted the key, and
`a-mutate-declaring-skip-if-recorded` is the third member of the same set, this
one authored. `deadline_seconds` is a number on every one of them, the wire
fixing the unit so that nothing downstream parses a suffix.

**Either positional naming nothing is a protocol error**, which is why there are
two of them: `operation/provider-matching-nothing` and
`operation/operation-matching-nothing` resolve against two different namespaces
— the repository's, and that Manifest's own — so each carries the sentence its
own lookup wrote, and the pairing above holds both against the case one
directory up.

`targets/` is the repository's own row, and what its cases are about is the
credential half. `five-artefact-demo-credentialed` carries both shapes at once:
`cloudflare-prod`, whose one slot names a variable the case's `env` sets, and
`local`, which declares no `auth:` block and so carries no `credentials` member
at all. It is also where `hosts` is read as an array in the declaration's own
order — `local` grants two, and both are there. `empty-string-variable` is the
line between a name and a value: the variable is set to the empty string and
`present` is `true`, whether an empty credential works being the endpoint's
business and not hyper's. No case anywhere holds a credential value, because
nothing on this surface ever reads one. `two-slot-declaration` is why the pair
is a pair: one declaration carrying two slots, one of them set and one not,
which a flat list of names could not have said. `no-targets-directory` is the
shape §9 states and Go makes easy to get wrong — `rows: []` where the command
found nothing, and a text block that says `no rows`.

There is no truncated `targets` case, and that is the tool rather than a gap:
`targets()` takes no arguments, so the only cut it could reach is the default's,
where the `../targets/truncated` case one directory up reaches its own with a
`--limit` this surface does not offer. `providers/truncated` is where the cut
result is held.

`check/` and `review/` are §9's Authoring pair, and they are the two tools that
reach nothing: no credential resolves, no network is touched, and nothing is
invoked (issue #198). No case here sets any environment variable but the one
that fixes the repository, which is the claim itself rather than a convention —
`TestGoldenCorpora_TheAuthoringToolsAreDrivenWithNothingButARepository` holds
it. They also reuse the repositories the two commands' own corpora already
carry, one directory up, so that what an envelope states and what a table or a
`--json` stream states are two readings of one fixture.

`check/five-artefact-demo-faulty` is the ordinary failing run: six rows ordered
by file path then line, `isError: true`, and — the part worth reading — **no
`outcome` key and no `error_code` on the envelope**. A command reporting
problems it found is §9's mapping row for exit `1`: the caller did not get what
they asked for, it is not a Refusal, and the remedy is the rows themselves.
`clean` is the other side, `rows: []` with the bit false. `ordering` is the
half neither of those reaches: its repository has one file carrying **two**
problems, so the rows are ordered by file path and then by line rather than by
file path alone.

`paths-narrow-the-report` calls the same repository with two of its six files
named, and is meant to be read against `five-artefact-demo-faulty` beside it:
one load, one rule set, four rows fewer. `paths-prove-the-full-load` is the
claim that makes it honest — the file it names is clean **only because the
artefacts it references loaded too**, so `rows: []` there is the full load
answering. Every case here that names a path carries a `wd` as well as an
`env`, and the `wd` is doing real work: `check` resolves a path argument
against the **process's working directory** before it stats one, exactly as its
command does, so a case whose working directory is not the repository would
have every path it names refused. The `wd` is a client started in the
repository it is asking about, which is the case §9's sketch has in mind when
it calls the argument *repository-relative* — the two spellings of that one
argument are named in `internal/mcp/tools.go` and are not this corpus's to
settle.

`path-not-found` and `an-empty-path-names-nothing` are the two malformed halves
of one argument. The first is the command's own usage error, paired with
`../../check/path-not-found` one directory up. The second never reaches a
command at all: `paths: [""]` is well-typed and names no path, and the schema's
`minLength` is made true on the server because it has to be — the command
resolves the empty string to the working directory itself, stats it clean as a
directory stats clean, and then matches no problem's file, so `check([""])`
would answer *no problems found* over a repository full of them.

`review/five-artefact-demo-procedure` is what the tool is for. Its `text` is the
**whole rendered review surface** — the gutter, `AUTHORITY`, `FLAGS` — byte for
byte what `../../review/five-artefact-demo-procedure` writes to stdout, and a
fence holds the pairing
(`TestGoldenCorpora_AReviewsTextBlockIsWhatTheCLIWroteOnStdout`). It carries
`FLAGS` rows and is `isError: false`, a flag being a fact about the artefact
rather than a problem with it. `gutter-marks-procedure` is the artefact that
**names** ones that are not there: it renders, marks `unresolved` in the gutter
and in the index, and is not an error either — the fault is `check`'s to report
and this surface's to annotate (ADR-0064).

`review/an-artefact-that-will-not-load` is the third of review's exit codes on
this surface, and the case that says the text-block table is keyed on the
**tool**: found and faulty writes `check`'s rows and `check`'s table, so the
text block is that table and `isError` is true. `name-matching-nothing` is the
fourth — an artefact that is not there at all has no row to write, which is the
usage error, and it arrives as a protocol error paired with its twin one
directory up.

What a declining case is held against beyond its own golden is the rendering
the CLI writes, and one fence holds both halves
(`TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr`). A
usage error is paired with the case one directory up that names it —
`provider/name-matching-nothing` here and `../provider/name-matching-nothing`
are one sentence on two surfaces. A Refusal is paired against the whole corpus,
as a row is: a text block that opens `refused:` has to open a Refusal some case
under `testdata/` writes on stderr, whole, with only the retry sentence after
it.
