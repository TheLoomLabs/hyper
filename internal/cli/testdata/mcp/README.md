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
presence when it runs, so `cloudflare-prod` is `presence: "set"` in a case whose
`env` fills the variable, `presence: "empty"` in one that sets it to nothing and
`presence: "absent"` in one that does not set it at all. What the
fence holds is that an envelope's row is **one of** the renderings the stream
writes for that identity — keys in order, values as stated.

**Where a case has a `--json` twin, the whole list is held in order.** The rows
in its `envelope.golden` are that twin's rows, in that twin's order, less the
terminal row — one list, two surfaces, checked
(`TestGoldenCorpora_AnEnvelopeIsTheStreamLessItsTerminalRow`, issue #204). The
truncation marker's `hint` is the assertion's one stated exception, naming the
arguments a tool call can type where the terminal names its flags, and the fence
states that exception rather than tolerating it: the terminal's hint opens `--`
and the envelope's carries no flag spelling anywhere, so a hint that regressed
to naming flags fails rather than passing a comparison the key was deleted from.

**A case whose twin writes a page and opens no stream is passed over, and the
fence names it.** There is no row list on the other side to compare with, which
is a `-json` twin to write rather than a rule to weaken — so the count and the
cases are logged rather than dropped quietly, and `go test ./internal/cli -run
TestGoldenCorpora_AnEnvelopeIsTheStreamLessItsTerminalRow -v` is where the list
of twins still owed is read.

**`tools.golden` is the one file here that is not a case.** It holds what
`tools/list` publishes — every tool's description and its two schemas, in the
name order a client receives them — regenerated behind the corpus's one
`-update` like every other golden. A `call` case holds what one call *answered*;
nothing else here holds what a client is *told it may ask for*, so an argument
widened from an enum to a bare string, a required member made optional, or an
output member dropped from a closed object would pass every case that does not
happen to exercise it. A schema is the contract an agent writes its calls
against, and a schema that drifts between two releases is the one way this
surface can break a caller without any answer changing.

**Two fences hold the corpus rather than any answer**, on
`../error_code_coverage_test.go`'s own shape: every one of the thirteen tools
has a case, and every one of §9's six envelope shapes does — the ordinary
answer, the answer with problems found, the guardrail declining, the Run
refusing, the Run failing, and the protocol error (`../mcp_coverage_test.go`).

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
`isError: true` and the whole Refusal as `text` — with the sentence saying a
verbatim retry refuses identically, which is the only place the protocol leaves
for saying it (ADR-0001). The `version-pin-mismatch` half of the gate is filed
with the invocations it is contrasted with, in `../exemption/provider`, against
the repository the three of them share.

**Those envelopes carry no `structuredContent` key at all**, and that absence is
the whole of ADR-0102 in the corpus: the command declined before it opened a row
stream, so there is no result for the `outputSchema` this tool published to be
true of. A half written there anyway would say `rows: []`, which reads as *this
tool ranged over a namespace and found nothing* where the fact is that it never
looked. Eight cases hold the shape, across five tools — the gate on `provider`
(three of them, one filed under `../exemption/`) and on `review`, a Target
granting no host on `probe` (two), an absent Store on `runs`, and no release
under the tag on `project`. What holds every *other* envelope here to the schema
its tool publishes is
`TestToolSet_EveryAnswerConformsToTheSchemaItsToolPublished`, with the validator
the MCP Go SDK itself validates with.

`review/version-pin-mismatch` is the one of those worth reading beside the three
`review` cases above it. The artefact is in the repository and the page is still
withheld, which is what makes `rendering` being `required` on `review` honest:
the member is absent, and so is the half it would have been absent from.

`provider/version-pin-mismatch-and-a-bad-name` is an ordering contract read
across the two surfaces. The gate fires before the positional is resolved
everywhere, so its twin in `../provider/` exits `77` and not `2` — and the same
call here is therefore a **Refusal envelope and not a protocol error**. A
mapping that read the positional first would invert it, and the two cases are
one claim written twice.

`providers/truncated-at-the-default-limit/repo` carries fifty Manifests, which
is one more Provider than the default limit admits once the built-in is counted.
That is the only way this surface can reach a truncated result at all:
`providers()` takes no arguments, the `--limit` its command carries is not
offered here, and the cut is therefore the default's. **It is the one case here
named for no case one directory up**, and the name is what makes that true:
`../providers/truncated` is a two-Manifest repository cut with `--limit 2`, so a
case sharing its name would be paired against a fixture it is not — which is
what `TestGoldenCorpora_AnEnvelopeIsTheStreamLessItsTerminalRow` pairs by.

What comes back is fifty rows, `truncated: true` carrying the bare boolean — a
namespace listing having no axis to name — and a text block that says so, there
being no stderr on this surface for the line the CLI writes beside its table.

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
`presence` is `"empty"` rather than either of the other two — the state an
upstream produces, and the one a Run binding that slot now Refuses under
(`credential-empty`, §12, ADR-0145). No case anywhere holds a credential value, because
nothing on this surface ever reads one. `two-slot-declaration` is why the pair
is a pair: one declaration carrying two slots, one of them set and one not,
which a flat list of names could not have said. `no-targets-directory` is the
shape §9 states and Go makes easy to get wrong — `rows: []` where the command
found nothing, and a text block that says `no rows`.

There is no truncated `targets` case, and that is the tool rather than a gap:
`targets()` takes no arguments, so the only cut it could reach is the default's,
where the `../targets/truncated` case one directory up reaches its own with a
`--limit` this surface does not offer.
`providers/truncated-at-the-default-limit` is where the cut result is held.

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
Which is why the **text block carries them**: a summary line, a blank line, and
then the table its twin one directory up writes to stdout, byte for byte
(ADR-0097, issue #214). `TestGoldenCorpora_AChecksTextBlockIsItsSummaryLineAndThenWhatTheCLIWroteOnStdout`
holds the two against each other, and holds the other arm on the cases that
found nothing — the summary line alone, with no table beneath it. `clean` is
that side, `rows: []` with the bit false. `ordering` is the
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
(`TestGoldenCorpora_AReviewsTextBlockIsWhatTheCLIWroteOnStdout`). **The same
page is in `structuredContent.rendering`**, which is why every `review` golden
here that carries a page holds it twice: one page, composed once, written to
both channels, because a page that lived only in the block would live in the
half nothing obliges a client to read (ADR-0100, issue #217). The fence holds
all three strings against each other. It carries `FLAGS` rows and is
`isError: false`, a flag being a fact about the artefact rather than a problem
with it.
`gutter-marks-procedure` is the artefact that **names** ones that are not there:
it renders, marks `unresolved` in the gutter and in the index, and is not an
error either — the fault is `check`'s to report and this surface's to annotate
(ADR-0064).

`review/version-pin-mismatch` is the one `review` case the stdout pairing passes
over, and it is not a gap: the gate declined, so §9's **fourth** text-block row
is what the block carries and there is no page on either surface to hold the
other against. What holds it is the stderr pairing, which collects a Refusal by
its opening across the whole corpus.

`review/an-artefact-that-will-not-load` is the third of review's exit codes on
this surface, and the case that says the text-block table is keyed on the
**tool**: found and faulty writes `check`'s rows and `check`'s table, so the
text block is that table, `rendering` is that table, and `isError` is true.
`name-matching-nothing` is the fourth — an artefact that is not there at all has
no row to write, which is the usage error, and it arrives as a protocol error
paired with its twin one directory up.

`runs/`, `run_show/`, `changes/` and `records/` are §9's Inspection four, and
every case here **reuses the four commands' own Stores** — a `repo-from` and a
`store-from` naming the corpus one directory up, so what an envelope states and
what a `--json` stream states are two readings of one seeded Journal (issue
#199). `run_show` is the one directory not named for its command: §9 names the
tool differently, a client holding every server's tools in one flat namespace
where a bare `show` names nothing, and the pairings above follow the tool to
`../../show/` rather than looking for a corpus that does not exist.

**Every `runs/` and `records/` envelope here that answers at all carries a second
line in its text block**, beneath the summary line and separated from it by a
blank one: *the record is the `hyper-store` branch of this repository — never
checked out, and it travels with a clone* (ADR-0113, issue #233). It is §9's
fourth text-block row and the one composed from neither the rows nor the
structured half, so its whole trace on this surface is the `content` block —
`structuredContent` is byte-identical to what it was. `runs/store-absent` is the
exception and is not one: a Refusal carries the Refusal rendering alone.
`run_show/` and `changes/` carry nothing of the kind, for the reason §9 states.

**Every `records/` row carries `dry_run`, the bare `false` included** (ADR-0114,
issue #234). It is read off the Journal entry of the Run that wrote the version,
which is why the corpus one directory up now seeds one: `records/the-heads-listed`
renders `true` on the Observation that a rehearsal wrote and `false` on the four
rows beside it, which is the whole of what the ticket asked the surface to say
without a second call.

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

All four are also what those three tools **publish**: `truncated` on `runs`,
`changes` and `records` declares the bare `false` or this marker, which is what
each of them writes and what the boolean standing there before them did not
admit. That was a live non-conformance on an ordinary return rather than a
Refusal — a client validating one of these four answers against the old schema
was told the server had broken its contract on the one answer §9 says must never
look complete — and it is what writing the conformance fence found (ADR-0102).

`changes/a-window-and-its-header` is the window with nothing in its Record
tables and `the-other-forms-of-a-row` is the other side of it, carrying all four
change forms across both tables — which is where the `observation` rows are, and
the reason it is here rather than left to the `asset` rows the two narrowed
cases carry.

`changes/a-rehearsal-is-the-subject-when-named` is the second surface's half of
issue #235, and it is here rather than left to the CLI corpus because **that is
the condition the ticket names**: the agent that found the gap had only this
surface, and `records` and `run_show` carry no field values by contract
(ADR-0115). One rehearsal in the Journal, no effectful Run anywhere, `subject`
naming it, and the two Observations come back with their values — `dry_run: true`
on the one side the window has, and no `baseline` member at all.

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
back `isError: true`, with no structured half and the Refusal rendered whole,
naming `hyper store init` — which is correct rather than awkward, creating the
record being the human's act and an agent's part in it being to say that it has
not happened.

The usage cases are the pairs §9's malformed set puts on this surface:
`runs/usage-an-outcome-outside-the-triple` is a value outside a closed set the
**command** closes where the flag is read, `records/usage-since-without-history`,
`changes/usage-since-and-between` and `changes/usage-subject-and-between` are the
argument pairs the commands refuse together, and `run_show/an-unknown-run-id` is a positional that satisfies
every schema and names nothing. Each carries the sentence its command wrote and
each is paired with its twin one directory up.

`run/` is the first of §9's Execution half, and it is the tool that closes the
loop: an agent that cannot run also cannot read back the Record it just caused
(issue #200).
Every case here **reuses the `run` corpus one directory up** — a `repo-from`
naming the same fixture repository, beside the same `store/` seed, the same
`mint`, the same `serve/` and the same `env` — and each is **named for the
argv case it is the same Run as**. That naming is the fence: a case here holds
a `store.golden` and so does its twin, and the two are compared byte for byte
(`TestGoldenCorpora_AToolLeavesTheBranchItsCommandLeaves`). One
Run, two doors, one branch — *ergonomics is the whole of the difference between
the two*, held as a comparison of bytes rather than as a claim.

`what-the-run-wrote-reaches-the-remote` is that claim over the third place an
invocation lands, and the one that carries the record off the machine: it stands
a `git` fixture with an `origin` and holds a `remote.golden` its argv twin's
matches byte for byte (`TestGoldenCorpora_AToolPushesWhatItsCommandPushes`,
issue #204). A branch golden says what a Run left locally and a tree golden what
it wrote in the working tree; neither says whether the entry reached the remote
a second clone reads, which is the half of §7's transport a Run pushes for.

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

`probe/` is §9's other Execution tool, and it is the one that protects the
review surface: a throwaway question that costs a reviewed Definition is what
ends with a repository full of Definitions nobody read (issue #201). Every case
here is named for the case one directory up it drives the same Probe as, and
each reuses that case's repository — `../../five-artefact-demo/repo` for the
seven that share it, and the one-edge repository the others carry — named in an
`env` because no tool takes an argument that names one. The two cases that never
reach a command name none at all, which is the claim rather than an omission:
what refuses them is the argument, one layer above any repository.

`a-503-and-nothing-else` is the ordinary return, and what it is worth reading
for is what is **not** in it: no `outcome` key, a Probe having no outcome triple
to report, and `truncated: false` rather than `null` — one Probe is one answer,
and the terminal row it lifts that from is `result` and not a Run's. Its text
block is `1 Probe`, the glossary's word for the thing and not the wire's
discriminator. `a-typed-input-filling-a-query-hole` is the `inputs` object doing
what the repeated flag does: `minutes` arrives as the JSON number `15` and is
read against the `integer` the Operation declares at that position, which is the
whole of ADR-0081 on this surface — the spelling crosses and the typing does
not. `a-host-that-answered-nothing` is the third `uptime check_http` rendering
and the one that says a `503` and a host that answered nothing are one piece of
news here: `isError: false`, the response object carrying `host` and nothing
else, and the sentence the CLI wrote on stderr about the refused connection
dropped, narration being what this surface drops.

`writes-nothing-at-all` is ADR-0009 held as bytes rather than as a claim. It
stands a Store branch, Probes beside it, and holds a `store.golden` that its
argv twin's matches byte for byte — no Record, no Journal entry, no Store write,
on the one path a golden can actually see it.

`a-host-outside-the-grant` and `a-repository-declaring-no-local` are the
Refusal, in both the shapes the repository leaves available: positioned on the
`hosts:` line that did not grant the host, and unpositioned where there is no
declaration to point at. Both come back `isError: true` with the Refusal
rendered whole and no rows, which is the reach coming from an artefact even
where no artefact named the Operation.

The eight declining cases split the way `run`'s ten do. Six are the command's
own sentences forwarded: the opaque Operation twice, *whatever any Target
grants*; the effectful one; the declared input left out; the input the Operation
does not declare, which carries **both** faults the command found, the second
being that nothing supplied the one it does declare; and the value that will not
read as its declared type. Every one of them arrives as a JSON-RPC error
carrying no `error_code`, which is ADR-0060's line: a code names a check that
declined an **artefact**, and a value supplied at a call is not one.

The other two never reach a command at all, and they are the shape this argument
adds: `inputs` is the one object on this surface whose keys are not the
schema's, so its `propertyNames` is where a key's own claim is stated and the
server is where it is made true. `usage-an-input-named-by-the-empty-string` is
the `minLength` half — a key that is well-typed and names no input — and
`usage-an-input-that-is-not-a-scalar` is the member type: every type §12
declares is a scalar, and an `array` reads as nothing at every position a hole
fills.

**What a client hears while a Run works, and what a client that gives up leaves
behind, are not cases either** — for run_signal_test.go's reason one directory
up. A progress notification and a cancellation are facts about *when*, which no
file beside a `call` can state, so both are driven past the goldens instead:
`../../mcp_progress_test.go` drives `run/a-skip-propagates` twice, once under a
progress token and once without, and `../../mcp_cancelled_test.go` drives the
same case with the call cancelled from inside its first Step. They are the same
cases and the same server; what the driver supplies is the one input a directory
cannot (issue #202). *Nothing between calls* is one package further over, in
`internal/mcp`, where two calls share one session and everything the client read
is checked in the order it arrived — a gap a corpus driving one call at a time
has no way to look into.

`project/` is §9's one Lifecycle tool, and the only tool here that **writes a
file into the working tree** (issue #203). The six cases that reach a command
are each named for the case one directory up they are the same invocation as,
and each carries a `repo/` of its own rather than an `env` naming a shared one:
a `tree.golden` is read against the tree the case was driven in, so a shared
repository would be a golden with no case to belong to — the rule the `project`
corpus already keeps, and the reason these six repositories are copies rather
than pointers. The seventh reaches no command and carries none, which is the
usual shape of a `usage-` case here.

The `tree.golden` is what the pairing is really for.
`TestGoldenCorpora_AToolWritesTheTreeItsCommandWrites` holds each of them to its
argv twin's byte for byte, which is
`TestGoldenCorpora_AToolLeavesTheBranchItsCommandLeaves` read over the other
place an invocation lands: one act, two doors, one tree. No case here holds a
`store.golden`, and the absence is the fact — `project` derives from reviewed
artefacts and touches no Store.

`writes-the-workflow` is the ordinary return: one `workflow` row carrying the
gloss's **parts** — `cadence`, `phrase` and `rate`, never the composed
phrase-and-rate line the page stacks — and a text block reading `1 workflow`,
the file and not the Procedure the glossary reserves the word away from.
`two-procedures-one-read-only` is the pair, which is where the two glosses are
two. `a-dropped-cadence-loses-its-file` is the row a removal writes: `path` and
`procedure` and no gloss at all, beside the row of the file the same act
rewrote.

**No envelope here carries an `outcome` key and none carries a last Journal
entry.** The first is the tool table's doing — `project` is not a Run, so it
declares no execution half — and the second is §10's one stated absence: this
surface writes a file and reports what it wrote, and what stands in the Store is
no part of that.

`the-pin-the-binary-disagrees-with` and `a-repository-with-no-pin` are the
exemption reached through this door. Every other tool passes the version pin
gate exactly as its command does; this one passes none, being the pin's only
writer — a writer gated on what it writes is a bootstrap with no bootstrap — and
the two cases are a repository pinning a version this binary is not and a
repository pinning nothing at all, both projecting cleanly and both leaving the
tree their twins leave.

`no-release-under-the-tag` is the other half: `release-artefact-absent` arriving
as `isError: true` with the Refusal rendered whole, and a `tree.golden` that is
the repository untouched — the fetch stands before the first write, so a Refusal
there leaves no half-written projection behind. `usage-a-procedure-is-not-an-argument`
is the closure: `project()` takes no arguments, there being no per-Procedure
projection for one to name, so a `procedure` argument is refused by the schema
one layer above the sentence the command writes for its own positional.

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
