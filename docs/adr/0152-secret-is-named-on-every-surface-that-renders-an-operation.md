# `secret:` is named on every surface that renders an Operation

Four surfaces render an Operation and none of them named the key it declares its secret output under.
All four do now: `provider`'s summary carries a clause, `operation`'s derived block carries
`secret_fields`, `review` marks the Operation's key line and indexes it with a `SECRET` flag, and the
orientation states where the key goes and where it does not. Issue #276.

This is [ADR-0149](0149-the-sink-was-supplied-both-secrets-were-lost-and-check-was-clean-throughout.md)'s
second finding, and the half [ADR-0151](0151-a-fields-value-is-a-path-and-a-sink-nothing-can-fill-is-a-refusal.md)
recorded as *not closed here and not this record's*. That one is enforcement: the wrong spelling now
fails. This is teaching: the right spelling is now findable.

## What the state was

A sealed `push-credential` run (#272) had to author an Operation declaring `secret:` output. It got the
spelling wrong, lost two live credentials to a `check` that then accepted the artefact, and spent
**thirteen `Bash` calls** finding the right one: `hyper --help`, `hyper run --help`, then eight calls of
`strings` and `nm` over `bin/hyper` hunting the word `secret` in the binary's text and symbol table,
before building a six-candidate matrix and driving it through `check` and `probe`.

**The thing being reverse-engineered was the authoring grammar of the artefact the whole tool is
about.** Reading the binary is not a seal artefact and is not discounted as one: `bin/hyper` is
reachable by design and cannot be otherwise
([ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md)), and it is the
same binary on a user's machine. A user who spells this wrong has exactly those calls available and no
more — `hyper` ships no specification on their disk, which is
[ADR-0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md)'s whole point.

The four surfaces are `provider`, `operation`, `review`, and the orientation the binary carries in
`internal/mcp/instructions.go` and writes to `AGENTS.md`. The task's own `docs/lookout-api.md` was in
hand too and correctly documents the API rather than the Manifest
([ADR-0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)), so it
was never the place for this.

## Why all four rather than one

The ticket put three candidates on the table and left which of them to take as the decision. The answer
is all of them and the fourth beside them, because they are not three renderings of one repair: they are
four readers at four moments, and each is the only surface reaching its own.

- **The orientation is the only one that reaches an author with nothing to copy.** In a fresh
  repository no Manifest declares `secret:` — the built-in `shell` Provider declares none and that is a
  decision rather than an omission (§3) — so every rendering-side repair is silent at exactly the
  moment the sealed session needed one. Without this clause the gap the ticket names stays open in the
  case that produced it.
- **`provider` is the listing an agent reads *before* it opens an Operation.** Its summary already
  carries the three facts a row's other members do not answer; *this Operation hands out a value the
  Store will not hold* is a fourth of the same kind, and a listing silent on it sends a reader looking
  for the fact one command further on.
- **`operation` renders what the Operation declares, and this is a declaration with consequences at Run
  time.** The ticket's own *arguably it should already* is the whole argument. The source above the
  block is verbatim, so the block's job is not the spelling — it is the consequence, which is what
  makes the empty answer load-bearing (below).
- **`review` is the human's.** A `secret:` list sits at the foot of an Operation's body, several lines
  below the key that binds the claim, where a Kind is one line down and opacity is nowhere in the file
  at all. The gutter puts *this Operation hands a value out of the Store* on the line a reviewer is
  already reading, and the flag block puts every such line on one screen — which on a Manifest of
  twenty Operations is the whole reason the block exists.

**A repair on one of the four would have been defensible and would have left the sentence in the ticket
true.** *Four surfaces render Operations and none of them names the key* is a statement about a set, and
answering it on three of four leaves a reader who reached for the fourth exactly where the sealed
session was. `provider` is the surface the ticket's candidate list omitted and its gap statement did
not; it costs one clause, and leaving it out would have needed an argument nobody has.

## What each surface says

`provider`'s summary gains a fourth clause where the Operation declares one, beside the projection
rather than inside it — which is §3's own arrangement of the two keys, `secret:` being a list beside
`record:` and not a marking within it (ADR-0151):

```
NAME          KIND  OPAQUE  SUMMARY
read_session  read  false   GET /session; repeatable; projects one Record; secret output: token
```

`operation` gains `secret_fields` in the derived block, standing with the Record pair rather than after
them: it is the third fact about the projection — which of the fields named there reaches the Store as
§7's constant instead of a value. It goes out under the name `records` already carries for the same
fact read off a written Record (§8), so the two surfaces name one thing once.

```
RECORD CARDINALITY  one
RECORD IDENTITY     $.host
SECRET FIELDS       token
```

`review` marks the Operation's key line with a fourth field and indexes it — the gutter line and the
flag row, out of a screen that renders the whole file between them:

```
  read  repeatable  secret token  │   read_session:

  FLAGS   index into the gutter above — no flag states anything the gutter does not
  SECRET  line 7  read_session declares secret output: token
```

And the orientation states the position in prose, beside the sentence that already said a credential is
never *in* an artefact — because this is that fact read the other way, the credential a call hands back:

> **A credential a call hands *back* is the converse and the Manifest declares it**: `secret: [token]`
> beside that Operation's `record:`, naming projected fields — and never inside `fields:`, which
> `check` refuses. A field named there reaches the Store as a marker no surface renders and no
> `require:` may read, and a Run reaching such a Step hands the values to a sink the invocation names or
> Refuses.

## Why the clause states both positions

**A rule stated in the orientation is stated with its exception, or it is not stated**
([ADR-0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)). Naming the position and
stopping would leave the wrong position exactly as findable as it was — which is the state ADR-0149
recorded, a Manifest that checked clean, declared no secret output, and destroyed two credentials at
exit `0`. So the clause carries `fields:` too, and both halves are held to the `check` that decides them
by `TestInstructions_TheSecretOutputSpellingIsTheOneCheckHolds`: two Manifests differing in one line,
the same field, the same path, the same projection, marked in the two positions an author picks between.

It names no flag. The orientation's length is a design constraint rather than an aesthetic one — it is
paid for on every session in every harness — and *anything a tool call or a command already answers
stays one* (ADR-0093, ADR-0096). `--secret-out` is answered by the `run` tool's own argument and by the
Refusal that names it; the `secret:` spelling was answered by nothing, which is what earns it the space.

## Why the empty answer is written and not dropped

`secret_fields` is `[]` rather than absent where the Operation declares none. That is
`patterns_resolved`'s rule, and the reason it holds here is stronger than symmetry: **the empty answer is
one a Run acts on.** A Run reaching such an Operation needs no sink, and a Run handed one anyway Refuses
under `secret-sink-unfilled` (ADR-0151). A member that went absent would leave a caller unable to tell
*this Operation declares no secret output* from *this block was not asked*, which are the two sides of
that Refusal.

The page says the same thing by having no line, which is the rule every labelled value on that screen
follows.

## What was considered

- **The orientation alone.** Rejected. It is the half that closes the gap for a fresh author and it
  closes nothing for the reader holding an existing Manifest — which is every subsequent agent, every
  reviewer, and the author of the second Operation in a Provider that already has one.
- **The renderings alone.** Rejected for the mirror reason. They speak only where a declaration already
  stands, and in a fresh repository none does.
- **A count rather than the names**, on `provider` and in the gutter. Rejected. *One field is secret* is
  the fact and *which* is the question, and a reviewer told the first goes back down the body for the
  second — which is the reading the mark exists to save.
- **Sorting the names.** Rejected. The order is the Manifest's own, on the rule every set-shaped mark
  already renders under: a claim silently re-sorted is not the claim the reviewer has open beside it
  (§8).
- **A `secret` marker with no flag.** Not available, and that is the vocabulary working rather than a
  constraint worked around: every marker class the gutter carries indexes into `FLAGS` (§12), so a
  marker class arriving in §8 brings its flag. `unresolved` was the first name to arrive that way
  ([ADR-0064](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md)) and
  `secret` is the second.
- **Naming the key in the `run` tool's `secret_sink` description.** Rejected when ADR-0151 declined it
  and rejected again here: one argument's description reaches an agent that has already decided to run,
  and reaches no author at all.
- **Putting it in `docs/spec/`.** Not available. `hyper` ships no specification on a user's machine
  (ADR-0099), and what teaches this has to be on a surface the binary itself carries.
- **A new `error_code` or a new `check`.** Not this record's. ADR-0151 took the enforcement and took it
  against the value rather than the key, which closes the class rather than the one door.

## Consequences

- **`FLAGS` grows to nine names, six of them standing.** It grows by §12's own rule rather than by a
  decision: the gutter gained a marker class and the listing followed it. `secret` reads on a Manifest
  Operation and on nothing else, and that is the fact having one place to be declared rather than the
  roster coming up short — `secret:` is a Provider author's claim about output that author
  understands, and no artefact downstream restates it.
- **`operation`'s derived block grows to nine members**, and the MCP tool's `outputSchema` requires
  `secret_fields` on every answer. That is a contract change on the wire: a consumer reading the block
  gets the member on every Operation, empty where none is declared.
- **The corpus reads one repository from four directions.** `secret-output-demo/` was `probe`'s alone;
  it is now `provider`'s, `operation`'s, `mcp/operation`'s and `review`'s as well, on the argument that
  already put it there — a copy per case is a fixture edit that reaches one golden and not another, and
  what these four cases are for is being read against each other.
- **Nothing else in the corpus moved but the `operation` goldens**, which gained `"secret_fields":[]`.
  Every other surface renders the fact only where an Operation declares one, and until now none did
  outside `run/` and `check/`.
- **The repair is taught, and the run it owes is `push-credential` (#271).** Nothing in the suite can
  fail if this repair does not work: whether an agent handed the clause then writes `secret:` in the
  right place is a thing only a transcript can say (`docs/agents/acceptance-re-runs.md`). **It is not
  bought here, and this is the last deferral against that task.** Four axes now stand on it — #272's
  two unobserved sink usage errors, ADR-0150's `skip-if-recorded` empty-sink line, ADR-0151's `check`
  message and converse Refusal, and this — and *several deferred repairs landing on one task is itself
  the argument for buying that task next*. The spend decision is #277, filed with this record on the
  precedent of #268 and #272: it becomes buyable the moment this lands, which is now. It carries one
  thing this record could not — **the fourth axis is not buyable as the task stands.** ADR-0150's line
  needs a *third* Run of the Procedure and a prompt that makes the Repeatability the session's natural
  choice, which is a change to what `push-credential.md` measures rather than a run of it.
