# A rule the orientation states is stated with its exception

**The orientation states the Bound rule as the checker holds it — mandatory on a `destroy`, refused
on an opaque one, and `shell:` is what makes a `destroy` opaque — in the sentence that already stood.**
It is one clause added to one sentence, and a case holds that sentence to what `check` actually
answers, so the two cannot come to disagree again.

Nothing about the Bound moved. `check` is untouched, §5 already stated both halves, and the worked
example the orientation carries already declared a Bound on the `http` `destroy` it teaches. What
changed is a document that stated half a rule the binary holds whole.

## The claim as it stood, and the two rules underneath it

The text an agent is oriented with said this, on both channels — the `instructions` field of MCP's
`initialize` and the `AGENTS.md` `project` writes where a repository holds none (ADR-0093, ADR-0095):

> An effectful Step may declare a `bound:`, the maximum Records it may affect; on a `destroy` it is
> **mandatory**.

The binary states two rules over the same key, and one of them is the opposite of that
(`checkStepBound`):

- a `destroy` Step carrying no `bound:` is `bound-missing` — **unless** the `destroy` is opaque;
- an **opaque** `destroy` Step carrying one is `bound-illegal`, and one carrying none is correct.

An opaque `destroy` is a `destroy` whose request block is `shell:` (`IsOpaqueDestroy`), which is
**every `destroy` a fresh repository can author at all**: `providers/` is empty until somebody writes
a Manifest, and the built-in `shell` Provider is the whole of what stands there (§12, ADR-0039). So
the orientation's rule was not merely incomplete on an edge; it was wrong on the first `destroy` its
reader would write.

Why the exception exists is [ADR-0053](0053-an-opaque-destroy-names-its-population.md)'s and §5's, and
it is not the rule softened. A Bound counts the Records an Expansion resolved to, and a count of the
commands an opaque Step ran says nothing about what any of them did — the only Bound one command could
carry is `1`, which would read *at most one thing will be destroyed* while `rm -rf /` is magnitude
one. Truthful and still misleading is the worse failure on the most severe Step the tool runs, so the
Step carries an `over:` selector and three review flags instead, `UNBOUNDED` among them.

## The evidence: a repair loop the orientation paid for

The sealed acceptance run of 2026-08-29 — `scripts/acceptance/run.sh`, `snapshot-lifecycle`, the run
recorded in [ADR-0100](0100-a-reviews-page-travels-in-the-structured-content.md) (issue #217). The
session authored an opaque `destroy`, wrote the Bound the orientation told it was mandatory, and was
declined. It worked the real rule out of `review`'s `UNBOUNDED` flag and wrote this in its handback:

> AGENTS.md says a bound is *mandatory* on a destroy; this provider reports `bound: "illegal"` and
> review confirms an opaque destroy takes none. The doc and the binary disagree, and the binary wins

That the session recovered is not a mitigation. It recovered from `review` and from `operation`'s
derived `bound` member — surfaces that were right — after the one surface whose entire job is to be
read before the first tool call had sent it the wrong way. **The orientation is the product on this
surface**: there is no `docs/spec/` on a user's machine (ADR-0099), so what the handshake and the
`AGENTS.md` say is the whole of what an agent knows before it starts guessing.

## What was decided, and against what

**A rule stated in the orientation is stated with its exception, or it is not stated.** That is the
general form, and §9 now carries it beside the length budget it has to live inside. A rule carrying
half of itself is worse than an absent rule: an absent one leaves an agent to call `operation`, which
answers `bound: illegal` for exactly this Operation; a half one is believed, authored against, and
declined.

**A second sentence about Bounds was rejected.** The length is a design constraint rather than an
aesthetic one (ADR-0093, ADR-0096) — this text is paid for on every session in every harness, as a
handshake field whether or not the model reads it and as a file the harness reads up front — and the
correction is a defect repair, not a licence to grow the section it landed in. The exception is a
clause of the sentence that already stood, and the case below holds it to exactly one sentence.

**The exception is stated as an exception.** *Mandatory on a `destroy`* and *refused on an opaque
one* are two clauses about overlapping populations — an opaque `destroy` is a `destroy` — so the
second says **refused instead**, which is the reading §5 states at length and the one clause has room
for. Restating the first as *mandatory on a non-opaque `destroy`* was the other way and was rejected:
it puts the exception in the common case's own clause, where every reader pays for it.

**Naming the Capability rather than only the word *opaque* was deliberate.** *Opaque* is a term
`CONTEXT.md` carries and the reviews render, and an agent that has read neither is told which
`destroy` it has: the one whose request is `shell:`. The clause names both, because the word is what
the flags and `operation` will say back and the request block is what the author is looking at.

**Teaching the Target's `opaque-destroy:` opt-in here was rejected as out of scope.** It is a second
rule with a second code (`opaque-destroy-not-granted`, §5), its own message says what it wants, and
the grain of that opt-in is a live question of its own (issue #220). One sentence about Bounds states
the Bound rule.

## The seam the fence is expressed at

`TestInstructions_TheBoundRuleIsTheOneCheckHolds`, in `internal/cli` for the reason the worked
example's case is there: a `check` is not reachable from `internal/mcp` and must not become so, the
surface handing a Call to a dispatch and knowing no command.

It drives **four repositories through `check`** and asserts the four answers — the orientation's own
worked example with its Bound and without it (clean, `bound-missing`), and the smallest opaque
`destroy` there is with a Bound and without it (`bound-illegal`, clean). Then it pairs each answer
`check` declines with the word the orientation must carry for an agent to have avoided authoring it:
`bound-missing` requires *mandatory*, `bound-illegal` requires *refused*. A rule that moves in the
binary and not in the text fails on the pairing, which is the direction it moved last time.

Two things beside it carry the rest of the claim. `boundSentence` asserts that **exactly one** sentence
of the orientation states the rule, which is the budget held as a fence rather than as an intention;
and `TestInstructions_TheBoundRuleNamesWhatMakesADestroyOpaque` reads the Capability off the binary
(`artefact.ReservedCapability`) rather than spelling it, so a renamed Capability points at the text.

**What the fence cannot do is read.** Prose has no schema, and a sentence that carried both words and
still misled would pass. What it holds is that the orientation names every outcome `check` has for a
`destroy` Step's Bound, in one sentence, in the vocabulary the other surfaces answer in — which is the
specific failure that happened, and the one that would otherwise happen again silently.

## Consequences

- **The orientation grows by one clause**, on both channels at once, the two being one text
  (ADR-0095). No golden carries it, `project`'s behaviour is unchanged, and an `AGENTS.md` already
  written keeps its old text until somebody regenerates it — which is `project` never overwriting a
  file that stands, working as decided.
- **§9 gains a rule and the count of what the orientation states is unchanged.** *A rule is stated
  with its exception* is a constraint on the nine, not a tenth thing to state.
- **`CONTEXT.md`'s **Bound** entry carried the same half-rule and is corrected with it.** *Mandatory
  on a `destroy` Step* and nothing after it is the same sentence one file over, in the document
  CONTRIBUTING sends every author to for vocabulary. No term is added and none is renamed; the entry
  gains its exception, in the register the entries are written in.
- **The worked example is unchanged.** Its `destroy` is an `http` one and its `bound: 5` was always
  correct; the fence now reads that Step both ways rather than only the way it ships.
- **`check`, `review` and `operation` are untouched.** All three were already right, and the derived
  `bound` member's three names — `mandatory`, `illegal`, `none` — are the vocabulary the new clause
  was written to match.
- **The next transcript is what says whether this worked.** ADR-0100's run had to reconstruct the rule
  from a flag; the claim here is that the text now states it. One run in which nobody hit the Bound
  would say nothing either way.
