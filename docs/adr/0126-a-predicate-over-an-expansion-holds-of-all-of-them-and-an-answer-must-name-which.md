# A predicate over an Expansion holds of all of them, and an answer must name which

**Two keys were written for a Step that produces one Record, and both met a Step that produced
several. The predicate resolved it silently and correctly — a `require:` rooted at an expanding Step
holds of every Record it acted on — and `answered` resolved it silently and wrongly, holding one of
the answers with nothing saying which member it belonged to. §3 now states the first, §7 states the
second, and `answered` is a list.**

## What was wrong (issues #251, #252)

Both came out of one session — the attended run against a real vendor recorded in
[ADR-0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md)
— and both were observed and not repaired there, on that ticket's no-repair rule. They are one ticket's
worth of shape: **an Expansion arriving at a sentence written for one call.**

**#251.** `hyper check` accepts a Requirement whose `step:` names a Step of `series` cardinality, and
`review` renders the line as though it were sound. It was reached twice — once as a probe, once by an
authoring session that met it cold while satisfying an ordinary brief. §4 makes a *reference* naming
such a Step `series-reference`, on the ground that pairing an expanding Step against a stored series is
a join by identity that is never performed, and the ticket's question was whether a `require:` is the
same fault wearing a different key. **What no session established is what a Run does with one.** No Run
in that session reached a series-rooted Requirement, so whether it Refuses at Run start, halts at the
predicate, or silently picks a member was unknown, and the ticket named finding out as the first thing
to do.

**#252.** A `destroy` Step expanded over two Assets, made two `DELETE` calls, and wrote one `answered`:

```json
"answered": { "host": "api.hetzner.cloud", "status": 404 },
"selector":  { "expanded_to": ["<a>", "<b>"] }
```

One answer for two calls, with no member named. §7 defines the key in the singular — *the host it
reached and the status it got* — and says nothing about the expanding case, so the ticket's own reading
was that this might be a gap in the specification rather than in the implementation.

## What a Run actually does, driven rather than reasoned

Five cases were added to the `run` corpus and are the evidence for everything below.

**A series-rooted `require:` is an AND over the Expansion.** It does not Refuse, does not halt for want
of a root, and does not pick a member: the predicate is asked of every Record the named Step acted on
and holds where each of them satisfies it. Against a three-member series read, one member `degraded`
halts the Run and leaves the Step after it *never reached* at exit `1`; all three `ready` and the Run
goes on. That is the reading a `when:` has had since it was written — the two are the same predicate at
the same root, read by the same reader — and `series-reference` was never about it: **a value takes one
thing and a filter takes a population**, so the rule that has to name one Record is the one written
where a scalar was expected.

**`answered` held the first non-ordinary answer in Expansion order, or the halt's where one halted.**
That was deliberate in the implementation and stated in no section. Three cases, each a two-member
`destroy`:

| the members answered | the entry held |
|---|---|
| `204` then `404` | `{host, status: 404}` |
| `404` then `204` | `{host, status: 404}` |
| `404` then `500` | `{host, status: 500}` |

**The first two produced byte-identical Store branches**, modulo the Run id: two worlds in which a
different Asset had already gone left one evidence. In the third the member answered `404` was
Tombstoned and the `404` that says so was overwritten by the next member's halt — a Tombstone in the
Store whose provenance is unrecoverable, and nothing anywhere saying a second answer existed.

## What that decides

**#251 is not a defect in `check`, and the repair is a sentence.** The behaviour is determinate,
order-independent and already the reading the operator set has; what was missing is that no section
said so, which `condition.go` had admitted in as many words: *§12 states the rule for neither — every
earlier Step in its text has one Record*. §3 now
states the population rule at the two Record roots and says outright that `series-reference` declines
the reference and not the predicate, so a reader meets a decision rather than a silence.

**#252 is a defect, and the ticket's own hypothesis is the half that does not survive.** §7 is silent
and the silence is real, but stating the implemented rule would have been the specification blessing a
loss rather than describing one. [ADR-0050](0050-a-status-is-an-answer-not-an-error.md) did not mention
this key in passing: it kept *already gone* off the Asset — recording it there is the reconciliation
[ADR-0010](0010-hyper-has-no-plan.md) declined — and justified that silence by
**relocating** the distinction, in its own words: *The distinction is not lost, it is relocated:
`answered` holds the status on the Step file, so the Journal says how `hyper` learned the thing was
gone and the Record says only that it is.* A per-Step key keeps that promise for a Step whose Expansion
is one member, and **a `destroy` over one member is not the ordinary shape of a `destroy`** — the
corpus's own canonical case expands over five.

So `answered` becomes **one entry per member of the Expansion whose call did not answer `2xx`**, in
Expansion order, each naming the `member` of `expanded_to` it is about. It is the shape the entry
already uses for everything else that is per member — `identities.members` and `expanded_to` are lists
in the same file — and the member is a name off `expanded_to` rather than a second copy of the Record,
which keeps ADR-0010's line where it is: the Record says what stands, and the entry says which of this
Step's calls was answered that way.

## Considered options

- **State the implemented rule in §7 and change nothing**, which is what #252 proposed. Rejected on
  ADR-0050's own consequence: the sentence to be written is *and on an expanding `destroy` the
  distinction ADR-0050 relocated here is not recoverable*, which is a limit for §13 rather than a
  description, and it is a limit on the ordinary case rather than a corner.
- **Add a `member` key and keep `answered` singular.** Rejected as half the fix. It separates the first
  two cases above and leaves the third exactly as it was — a `404` still overwritten by a later
  member's halt, with nothing saying a second answer existed.
- **Decline a series-rooted `require:` in `check`, as `series-reference` already declines a
  reference.** Rejected: the predicate has a determinate meaning at that root and it is the one the
  operator set already gives it. Declining it would remove a legal authoring — *every member of this
  list is still what I expect* is a thing an operator says — to protect against an author meaning
  something narrower, which `hyper` cannot know and would be guessing at.
- **Make the halt name the member that failed.** Rejected on
  [ADR-0035](0035-a-predicate-that-cannot-decide-refuses.md)'s ground, which the surrounding code
  already keeps: a sentence carrying one observed value is naming whichever member came first. The
  halt carries a **count** instead — how many Records the root held and how many satisfied the test —
  which names no member and no value and is the fact an author needs.

## Consequences

- **`answered` is a list on every Step file, one entry per member, and the singular object is gone.**
  A Step that resolved no selector writes one entry naming no member, which is the silence
  `expanded_to` already keeps on that Step. An entry naming neither a host nor a command is refused by
  the encoder as the empty one always was, and a `member` alone does not save it.
- **The halt's answer is one member's like any other**, written last, beside the entries the members in
  front of it wrote rather than in place of them. A halt that names no answer — the deadline — writes
  none, and the entries in front of it make no claim about what ended the Step: the Disposition is
  where *which of §6's three cases* has always been read from. Before the key was per member an
  earlier member's answer had to be dropped to keep a per-Step reading honest, and that suppression is
  gone.
- **Two Runs in which a different Asset had already gone are now two entries.** The corpus holds the
  pair, `204`-then-`404` against `404`-then-`204`, which were byte-identical before this and differ in
  the member each names now.
- **§9's `show` grows a `MEMBER` line per group** and the `run_show` tool publishes `answered` as an
  array. A Step whose Expansion is one member with no selector renders exactly as it did.
- **§3 states the population rule, and it is the same rule at both Record roots.** A `require:` and a
  `when:` are one predicate, and neither is declined for rooting at a Step that expanded. `CONTEXT.md`'s
  Requirement entry says *everything* an earlier Step acted on rather than *what* it acted on, which is
  the same sentence made unambiguous.
- **An author who roots a `require:` at a list read has written a stricter test than they probably
  meant, and nothing declines it.** It halts more and never less, so the failure is legible rather than
  silent, and the halt now says how many Records the root held and how many satisfied the test — which
  is the difference between *the thing I created is still watched* and *everything on this account is*.
  Where that costs a Run is a real cost and it is stated in §3 rather than defended.
- **The suite would still not have caught either.** ADR-0122's Requirement evidence comes from
  `change-window`, whose reads are `shell` Steps of `one` cardinality; `monitor-coverage` is the only
  task with a `series` read and asks for no `require:`, and its effectful Step does not expand over
  Assets. No sealed transcript has ever had an expanding effectful Step meet a non-`2xx`.

## The run this owes

**`monitor-retirement`, and it is deferred until [#255](https://github.com/TheLoomLabs/hyper/issues/255)
lands.** [#253](https://github.com/TheLoomLabs/hyper/issues/253) decided that the surface both defects
sit on is fenced by no task — the ADR-0125 session ran deliberately outside the seal, against a vendor
no task file has — and #255 is the task that closes it: a Requirement between a `mutate` and a
`destroy`, and a `destroy` Step that expands over two Assets and meets `404` on one and `204` on the
other. Both halves of this repair land on it.

Under [`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md) the repair is **taught**,
because part of it is: the halt's new sentence and `show`'s new `MEMBER` line are text an agent reads
and then decides on, and nothing in the suite can fail if they do not teach. The enforced half does not
wait on the run and has not — the encoder, the decoder, the published schema and five corpus cases hold
the list, the member and the two orders. **It is not being bought now because there is no run to buy:**
the task that would measure it does not exist yet, which is the gap #253 was filed to say out loud, and
buying a run against `change-window` or `monitor-coverage` would measure a surface neither of them
reaches.
