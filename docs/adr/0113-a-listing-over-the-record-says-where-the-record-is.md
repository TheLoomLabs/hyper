# A listing over the record says where the record is

**`runs` and `records` begin every answer with one sentence: the record is the `hyper-store` branch of
this repository, it is never checked out, and it travels with a clone.** On the terminal it stands above
the table, and on the MCP surface it stands beneath the summary line in the `text` block — the two places
each surface has for saying something about the answer as a whole. It carries no row and no member on the
wire, the branch being fixed by §7 rather than found by a call.

The orientation states the same fact in the paragraph about **reading the record back**, beside the four
commands that do it, rather than in a subordinate clause of the paragraph about the `--response` file.

`show` and `changes` say nothing about it and stay silent: neither is a search.

## What was wrong (issue #233)

Nothing an agent is allowed to call said where the account is, and an agent asked what its repository's
account amounted to answered that the account is not in the repository.

Every route to the fact was blocked or unluckily placed.

- **The reading surfaces render content and never location.** `runs`, `show`, `records` and `changes` say
  what the account holds; none of them said where it is held or under what ref.
- **The working tree shows nothing.** `hyper` never checks the Store out (§7, ADR-0075), so `ls`, `git
  status` and `git status --ignored` are all silent, and there is no `.hyper/`, no `store/` and no
  `records/` to find.
- **The three commands that would name it are the three an agent may not run.** `install`, `store init`
  and `compact` are on the far side of the line ADR-0104 draws, and none has a tool on the MCP surface at
  all (§9).
- **The orientation stated it once, in a paragraph about something else.** The clause was *a Store that is
  append-only and travels in the repository*, and it sat inside the warning about the `--response` file
  and why not to author a throwaway Operation to look at a body — the paragraph an agent reads when it is
  thinking about projections, not about the Record.

The evidence is the sealed acceptance run of 2026-08-30 on `fleet-rollout` (ADR-0110, issues #223 and
#232). The task's second closing question asks what the repository's account of the rollout now amounts
to. The session answered the *what is in it* half correctly off `changes`, then went looking for where it
lives:

```
ls -a
git status --short --ignored
for d in .hyper .hyper/store store records; do [ -e "$d" ] && echo "EXISTS: $d"; done
```

All three came back empty, and it reported to the human:

> There is no store directory anywhere under the working tree — the Records are readable through
> `runs`/`records`/`changes`, but they are not in your diff. So right now the repository can say all of
> this, but only on this machine; **a clone would get the Procedure and not the history.**

**That last sentence is false.** `git log hyper-store` in that same repository shows a commit per Record
version and a commit per Step of both Runs, and a clone gets every one of them (ADR-0110 counts them).

The session had read `AGENTS.md` in full at its fifth call, so it had been told. It never ran `git
branch`, and nothing prompted it to.

**It is the one fault in that run that reached the human as a false statement about durability**, and it
is the kind that changes what an operator does next: someone told their audit trail is machine-local backs
it up, stops trusting it, or re-runs the rollout somewhere it will *really* be recorded. It also lands on
what the tool is for — `hyper` records so that a repository can account for what was done to it, and a
session that cannot find the account concludes there isn't one.

The agent behaved correctly at every step. It read the orientation, searched the place a store would
obviously be, and reported honestly what it found. The gap is that the search space it chose was the
working tree, and nothing it was allowed to call would have redirected it.

## Both halves, because each is insufficient alone

The ticket offered three shapes: say it where an agent will read it, put it on a surface, or both.

**Moving the clause alone does nothing for a session that skims**, and this session did not skim — it read
the file whole and the clause still did not survive to its answer. A text is read once, at the start, and
the question it answers is asked at the end.

**A surface alone is met only by a session that lists.** An agent that reasons about the record without
calling `runs` or `records` — planning, or answering from a Run's own terminal line — meets nothing.

So both, and they are one sentence in two registers rather than two facts: the orientation's is prose in
the paragraph that teaches the loop, the surface's is `store.Location`, spelled once in the `store`
package and written by the page and the text block alike.

## The three claims, and why each is stated

The sentence says three things and each of them was a wrong answer somebody gave.

**The branch**, because nothing on the surface named it. `hyper-store` is fixed (§7, ADR-0014), so naming
it costs nothing and hands the reader a `git log hyper-store` they can run themselves — the fact is
verifiable from the sentence rather than merely asserted by it.

**Never checked out**, because that is why the search returned empty. A reader told *the record is a
branch* and not told it is never checked out looks for it in the working tree exactly as this session did,
finds nothing, and is now confused by a surface that just told them where to look.

**Travels with a clone**, because portability is what was denied. *There is a record* and *the record
survives leaving this machine* are two claims, and the session got the first one right; a sentence that
established only existence would leave the false half standing. The ticket requires the words, and the
cases hold them.

## Two of the four commands, not all four

`runs` and `records` are the two whose job is *finding* something in the Store. They range over a
namespace, and both already named the branch on the page they wrote when they found nothing in it — a
listing that names the namespace it ranged over when it is empty has no reason to stop naming it when it
is not. That is the whole change on the terminal: the same sentence, read where the list is not empty.

`show` resolves a Run id its caller already holds, and `changes` compares two Runs they already found.
Neither is a search, and a session that ran all four — which is what the transcript did — would read one
sentence four times.

## Prose on the page, and no row on the wire

The sentence states a **constant of the design** rather than a result. The branch is fixed by §7, the
checkout is fixed by ADR-0075, and portability is a property of git. A row carrying that would be a row
every NDJSON consumer parses on every call to be told what the specification already states, and its
members would be the same bytes forever.

So the CLI's `--json` stream is unchanged, and the structured half of the MCP envelope is unchanged. What
does carry it on the MCP surface is the `text` block, which is that surface's analogue of the page — and
§9 already argues the case for `check`: *a client is not obliged to surface `structuredContent` to the
model behind it, and most do not*. A fact carried only there is a fact an agent may never meet, which is
the failure this decision exists to close.

It is the first member of §9's asymmetric table composed from neither the rows nor the structured half,
and it differs from `check`'s row in one way that matters: **it is written even where there are no rows.**
`check` promises the rows and an empty set keeps that promise by standing where they would; this sentence
promises nothing about the rows and is needed most where there are none, an empty listing being the call
most easily read as *there is no record*.

## What was considered

**A `store` row at the head of `runs` and `records`.** Refused above: a constant repeated on every call is
not a result, and §9's own reason for putting `check`'s rows in the `text` block — most clients do not
surface the structured half — applies with more force to a fact that has no other channel.

**A seventeenth command that answers where the Store is.** Refused. §9 fixes sixteen commands, flat, every
name a word the glossary defines, and a command answering one constant would be a command whose whole
output is a sentence this one now writes for free.

**Moving `store init` or `compact` across the line, so an agent could ask them.** Refused, and the ticket
refuses it too: that line is ADR-0104's and this does not reopen it. The three stay where they are, and
the orientation's paragraph about them is untouched.

**Narrating it on stderr instead.** Refused on a mechanical ground rather than an aesthetic one: a tool's
narration goes to `io.Discard` (§9), so the MCP surface would never carry it — and the MCP surface is
where the session that got this wrong was standing.

**Putting it on all four Inspection commands.** Refused above. One sentence four times in one session is
how a surface teaches a reader to skip it.

## Consequences

- **`store.Location` is the sentence, spelled once.** Both surfaces write it, and a sentence maintained
  twice disagrees with itself the first time either is edited.
- **The empty pages no longer name the branch a second time.** `no Journal entry in this repository's
  Store` and `no Record in this repository's Store` lost the clause that named the namespace, the line
  above them now naming it. `TestInspectionListings_NameTheBranchOnce` holds it.
- **§9's asymmetric text-block table has five rows rather than four**, and `textBlock` has a fourth member
  for the tool half of it. A tool that declares nothing there is still the ordinary case.
- **Every `runs` and `records` golden that renders a page gained a line, and no `--json` golden changed.**
  The cases that render none — `store-absent`, `version-pin-mismatch`, the usage errors — are unchanged,
  a command that never opened the Store having nothing to say about where it is. That asymmetry is the
  decision itself, visible in the corpus.
- **The orientation is one sentence longer in the reading-back paragraph and one clause shorter in the
  `--response` warning**, which now points at *the append-only record above* rather than restating what it
  is. Length is a design constraint on that text, and this pays part of its own way.
- **`TestInstructions_SayWhereTheRecordLives` holds the placement and not only the words.** It asserts
  that the sentence stands after *Read the record back with*, because where the fact is said is the whole
  of what this ticket was about.
- **ADR-0110's open question closes.** That ADR declined to decide this on one transcript and left it to
  issue #233's own acceptance criteria; this is that decision.
