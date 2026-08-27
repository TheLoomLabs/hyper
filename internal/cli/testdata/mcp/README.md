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
answering. **No case here carries a `wd`**, and their absence is the claim: a
path argument is read against the repository root and never against the
directory the client started the server in (ADR-0089), so every one of these
cases runs from a working directory that is not the repository at all and names
its files exactly as §9's sketch says it may. Each carried a `wd` while the
command read a path against the process's directory, which pinned the one case
where the two roots agree — a client standing in the repository it is asking
about — and that is what issue #205 settled and this corpus stopped needing.

`path-not-found`, `a-path-outside-the-repository` and
`an-empty-path-names-nothing` are the three malformed halves of one argument.
The first two are the command's own usage errors, each paired with the case of
the same name one directory up: a path naming no file in the repository, and a
path resolving outside it — which resolves to a real file on the disk beside the
fixture and is refused for naming nothing this repository holds. The third never
reaches a command at all: `paths: [""]` is well-typed and names no path, and the
schema's `minLength` is what refuses it here, one layer above the sentence the
command writes for the same argument.

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

`runs/`, `run_show/`, `changes/` and `records/` are §9's Inspection four, and
every case here **reuses the four commands' own Stores** — a `repo-from` and a
`store-from` naming the corpus one directory up, so what an envelope states and
what a `--json` stream states are two readings of one seeded Journal (issue
#199). `run_show` is the one directory not named for its command: §9 names the
tool differently, a client holding every server's tools in one flat namespace
where a bare `show` names nothing, and the pairings above follow the tool to
`../../show/` rather than looking for a corpus that does not exist.

The truncated cases are what the ticket is really about, and there are four of
them — one per marker the four commands can write.
`runs/a-cut-listing-names-the-time-axis`, `changes/the-limit-cuts-the-tables`,
`records/a-cut-listing-names-the-identity-axis` and
`records/a-series-is-cut-at-the-version-cap` each carry the marker with all four
members and a **hint naming the tool's arguments**: *narrow with `since` or
`target`*, where the terminal writes `--since` or `--target`. `changes` is the
one worth reading twice — its hint says `record_kind` where its command says
`--kind`, which is a rewording no textual edit of the command's sentence could
have reached, and it is the only member of any answer that differs between the
two surfaces. The last of the four is `records`' **second** marker: the limit
dropped nothing and the cap on versions per series did, so the axis is `time`
and the counts are versions.

`changes/a-window-and-its-header` is the window with nothing in its Record
tables and `the-other-forms-of-a-row` is the other side of it, carrying all four
change forms across both tables — which is where the `observation` rows are, and
the reason it is here rather than left to the `asset` rows the two narrowed
cases carry.

`run_show/an-entry-read-back` and `run_show/an-expansion-asked-for` are one
Store driven twice, and the pair is the whole of what `expansion` does: the
entry's second Step carries a selector, no row shows one without the argument,
and every row does with it. `a-halted-destroy-under-expansion` is the richer
entry beside them — a `bound`, a relative operand glossed under `resolved`, and
an `answered` status that is not the ordinary answer — and `a-refused-entry` is
the entry whose Run recorded a Refusal, which is where the `refusal` and
`remediation` rows are. `a-projection-failure-names-the-path` is the one member
§6 gives a Step of its own: a `projection_failed_path`, beside the partial
`records` it leaves.

`runs/store-absent` is §9's line about the far side of the tool set: **a tool
finding no Store Refuses naming a command its caller cannot reach.** It comes
back `isError: true`, with no `outcome` key and the Refusal rendered whole,
naming `hyper store init` — which is correct rather than awkward, creating the
record being the human's act and an agent's part in it being to say that it has
not happened.

The four usage cases are the pairs §9's malformed set puts on this surface:
`runs/usage-an-outcome-outside-the-triple` is a value outside a closed set the
**command** closes where the flag is read, `records/usage-since-without-history`
and `changes/usage-since-and-between` are the two argument pairs the commands
refuse together, and `run_show/an-unknown-run-id` is a positional that satisfies
every schema and names nothing. Each carries the sentence its command wrote and
each is paired with its twin one directory up.

`run/` is §9's Execution half, and it is the tool that closes the loop: an agent
that cannot run also cannot read back the Record it just caused (issue #200).
Every case here **reuses the `run` corpus one directory up** — a `repo-from`
naming the same fixture repository, beside the same `store/` seed, the same
`mint`, the same `serve/` and the same `env` — and each is **named for the
argv case it is the same Run as**. That naming is the fence: a case here holds
a `store.golden` and so does its twin, and the two are compared byte for byte
(`TestGoldenCorpora_ARunThroughTheToolWritesTheStoreTheCommandWrites`). One
Run, two doors, one branch — *ergonomics is the whole of the difference between
the two*, held as a comparison of bytes rather than as a claim.

`the-tracer-bullet` is the ordinary return: `outcome: completed`, the entry
named whole, `dry_run: false` beside it, and a text block that is §8's terminal
line arriving as a sentence. The exit code is the one member of that line the
summary does not compose — the terminal compensates for the outcome arriving
last by carrying a code, and this surface has none — so what says *past this
lies time* is `failed` standing where `refused` would have. A Refusal is the
other way round: its text block is the command's **page**, forwarded whole, and
the page's own terminal line goes over with its code and its `hyper show`
pointer in it, because the rendering is not this surface's to edit. `a-skip-propagates` is the same
shape over a Procedure whose Steps skipped, and `a-destroy-halted-at-the-fourth-of-five`
is the other side of the triple: `failed`, `isError: true`, and the Step row
carrying `3 of 5` in its two counts.

`a-rehearsal-stops-at-the-first-effect` is `dry_run: true`, and it reports
**`completed`** — a halted rehearsal is the correct outcome of a correct
operation, so the answer is partial and says so in the Dispositions rather than
in the outcome. The marker rides in the structured half and in the text block
both, which is §7's one exception to the absence rule holding here for the
reason it holds in the Store: without it the sentence a Run that reached the
world writes and the sentence a rehearsal writes are the same bytes.

**Where that answer stops is on a row**, and this case is what holds it there:
the `publish` Step carries `withheld: true` and the *never reached* `read`
behind it carries no such key (§8, ADR-0091, issue #206). It is the second half
of what a `dry_run` call asks, and until it was a member it was on the page
alone — the text block here is `run`'s summary line rather than the page, so the
structured half was the only place it could land. The row object is the one
`run/a-rehearsal-stops-at-the-first-effect-json` writes, member for member, which
is the one renderer behind both forms visible in two checked-in files — what
differs is the indent and the trailing comma an envelope's array puts around it
(ADR-0026).

**A Refusal comes back rendered in full**, and `run` is where that rendering is
the command's own page rather than its stderr. `an-expansion-past-its-bound` is
the whole of it — the Step table, the caret excerpt, the `=` notes, `EDIT ONE OF`
with its `FROM` and `TO`, the terminal line's pointer at `hyper show
--expansion`, and the sentence saying a verbatim retry refuses identically.
`check-refuses-the-run` is the same shape with five members and no Step table at
all, *nothing ran. no step was reached.* standing in its place.
`version-pin-mismatch` and `a-repository-with-no-store` are the two paths that
decline **before a Run is identified**: both carry `outcome: refused` with no
`run_id`, which is §8's rule that what is missing there is the id and never the
key beside it.

`a-shell-mutate-lands-an-asset` is the `shell` Capability through the tool: the
case carries its own `bin/`, so the argv a Step runs resolves against the
fixture and not against the machine, and what the tool reaches is a real child
process. Two cases here carry a `-json` twin that was written for them —
this one and `a-secret-field-is-the-marker` — because the row-pairing fence
holds an envelope's row against the streams the corpus writes, and a fixture
only this surface drives has none.

`a-secret-sink-names-every-step` is the guardrail the sink exists for: the
invocation supplied none, so the Run Refuses before Step 1 naming **every** Step
that would have needed one. `a-secret-field-is-the-marker` is the other side —
a `secret_sink` supplied, the Run completing — and what it is really for is what
is *not* in its envelope: no secret, under any key, the output schema being
closed over the members §9 names. Its sink is a relative path because a golden
cannot hold an absolute one; §9 describes the argument as absolute, and what the
command does with either is resolve it against the process's working directory.

The ten `usage-` cases are §9's malformed set for this tool, and they split in
two. `usage-a-definition-is-not-an-argument`, `usage-inputs-is-not-an-argument`
and `usage-a-target-is-not-an-argument` never reach a command: the schema is
closed over three properties, so each is `json: unknown field` — and the closure
is what refuses the bypass nobody has thought of yet, which no list of forbidden
words could. `usage-an-empty-procedure-names-nothing` and
`usage-an-empty-sink-names-no-path` are well-typed arguments that name nothing,
the schema's `minLength` made true where it is enforceable; the second is
load-bearing rather than tidy, an empty sink read as *no sink* being the very
Refusal above arriving from a typo. The other five are the command's own
sentences, forwarded: a positional that is not a Procedure, one that is a
Definition, one written as a path — this tool takes a Procedure's **name** and
`review`'s two forms are `review`'s (ADR-0090) — a sink named `-`, and a sink
inside the repository working tree. Each of the last two names `--secret-out`
where the argument is `secret_sink`, which is the rule rather than a rough edge
— a usage error's message is the sentence a person would have read, and the
truncation marker's hint is the only wording §9 spells differently between the
two surfaces.

**The three ways a Run loses the Store are not here**, for the reason their argv
twins are not: a lock is held by a live process and git's account of an
unreachable remote names a temp directory, so neither is a directory of files.
They are in `../../run_store_lost_test.go`, beside the argv cases they double,
and they are the one place the two surfaces cannot say the same thing — the lock
and the sync at Run start write no terminal row at all, so the envelope carries
§12's reading of the code the command would have returned.

What a declining case is held against beyond its own golden is the rendering
the CLI writes, and one fence holds both halves
(`TestGoldenCorpora_WhatDeclinesInAnEnvelopeIsWhatTheCLIWroteOnStderr`). A
usage error is paired with the case one directory up that names it —
`provider/name-matching-nothing` here and `../provider/name-matching-nothing`
are one sentence on two surfaces. A Refusal is paired against the whole corpus,
as a row is: a text block that opens `refused:` has to open a Refusal some case
under `testdata/` writes on stderr, whole, with only the retry sentence after
it.
