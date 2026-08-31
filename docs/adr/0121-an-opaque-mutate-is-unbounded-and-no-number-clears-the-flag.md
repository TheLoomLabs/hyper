# An opaque `mutate` is unbounded, and no number clears the flag

**`UNBOUNDED` has a third form: it renders on an `opaque` `mutate` Step whatever that Step declares.**
`check` still accepts the Bound, the Bound still counts Records, and `bound-illegal` is still the
`destroy` rule alone. What moved is the one surface whose job is to say that a number and a blast
radius are different things.

The flag is the tool's only editorial claim (ADR-0026), and until this it had a form an author could
clear by writing a number that changed nothing about what the flag was for. That is not an economy or
an omission — the two failures §12's `unbounded` bullet already argues against — it is the surface
teaching an edit that buys nothing, which is the one thing a mark that exists to make somebody look may
not do.

## What was wrong (issue #241)

§4 states the `destroy` half in as many words: an `opaque` `destroy` carries no Bound, one written
there is `bound-illegal`, and the reason is that *a count of the commands it ran says nothing about
what any of them did*. §5 states the same thing with the rendering attached — `1` *would stand in the
gutter and in `FLAGS` reading at most one thing will be destroyed while `rm -rf /` is magnitude one* —
and §12's `unbounded` set had exactly two members.

**There is no third member in that set, and an `opaque` `mutate` is the case that falls between the
two there were.** It takes a Bound, `check` accepts it, and both marks — the flag and the gutter's `!`
— go away, while the command line the Bound says nothing about is unchanged.

## The evidence: the repair the surface offered

The sealed acceptance run of 2026-08-31 on `change-window`, read in
[ADR-0120](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md)
(issues #238, #240). The session's first draft carried no `bound:`, and `review` answered:

```
  mutate!  opaque  local    │   - id: grant
                            │     args:
                            │       command: [sh, -c, 'cat requests/pending >> firewall/allow && : > requests/pending']

  OPAQUE     line 8  step grant  mutate reaches an effect hyper cannot describe
  UNBOUNDED  line 8  step grant  mutate with no declared bound
```

It appended `bound: 1` to both effectful Steps, re-read the same artefact, and got `mutate` in the
gutter and one flag. Its reasoning, in the report it handed the reviewer:

> `bound:` is mandatory only on a `destroy` and refused on an opaque `destroy`, so I was not certain an
> opaque `mutate` would take one. I added `bound: 1` to each effectful step and re-checked: accepted,
> the flag cleared, and the gutter dropped to `mutate`. Each step issues one command and mints one
> Record, so `1` is the honest number.

**It is the honest number, and that is the fault.** The Step mints one Record and `1` counts it. What
the reviewer of the second rendering lost is the two marks that said *this Step's magnitude is not
bounded by anything*, on a Step whose sibling flag one line up says `hyper` cannot describe what it
reaches: the command appends two firewall rules and truncates a file, and `bound: 1` is true of the
Record and silent about all of it.

This is [ADR-0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)'s
precedent used again — an agent's disclosed reasoning read as evidence about a surface — and it is the
first transcript of the review surface teaching an edit rather than a fact.

## The argument, one Kind down

The reason a Bound is refused on an `opaque` `destroy` is not that `destroy` is severe. Severity is why
the case was noticed; the argument is that **a count of Records says nothing about what an opaque
command did**, and that premise holds identically on a `mutate`. Only the conclusion had been drawn.

Where the two Kinds part is what may be *written*, not what may be *read off it*:

- On an `opaque` `destroy` there is no honest count to write at all — the only value a single command
  could carry is `1`, and it would render as a promise the Step cannot make. So the check refuses it.
- On an `opaque` `mutate` the count is honest. `check` has nothing to refuse, and the whole of the
  problem is what stands beside the number. So the flag says it.

**One argument, landing on the check at one Kind and on the rendering at the other.** That is why this
is not the `destroy` rule softened and not the `destroy` rule extended.

## What was decided, and against what

**The flag gains a third form.** It renders on an `opaque` `mutate` regardless of a declared Bound,
with its own text — *a bound on an opaque mutate counts records, not what the commands did* — which is
the sentence the transcript's author needed and did not have. It is the `opaque` `destroy` form's own
footing: implied by the marks the gutter already carries beside the line (the Kind, and `opaque`), so
§12's supply rule holds unchanged — `FLAGS` names no fact the gutter does not annotate in place.

**`bound-illegal` on an `opaque` `mutate` was rejected.** It is the strongest reading of §4's argument
and it refuses a value that is true. `check` would decline an artefact that says something correct
about itself, which is a different kind of surface from the one every other code in §4 belongs to; and
§9's derived Bound fact is a three-member enumeration on the *Operation* (`mandatory`, `illegal`,
`none`), so every `opaque` `mutate` Operation would move from `none` to `illegal` and the MCP tool's
member with it — a wire change to say something the review already says better, beside the line it is
about. It would also take `bound-exceeded` off those Steps, which is a real static check on an authored
`over:` `values:` list and the one thing a Bound on such a Step does buy.

**Leaving it, and saying so in §4, was rejected.** It is the option the ticket offers for the reading
where a `mutate` is not a `destroy` and `OPAQUE` still stands beside it. But `OPAQUE` says *`hyper`
cannot describe this effect* and `UNBOUNDED` says *nothing bounds its magnitude*; they are two facts,
and the second was the one the author edited away. A sentence in §4 would document the asymmetry for a
reader of the specification and would do nothing for the author reading a rendering, who is who the
flag is for.

**The gutter is untouched.** An `opaque` `mutate` carrying a Bound is still marked `mutate`, not
`mutate!`: the `!` is §8's mark for an absent `bound:`, it is read off the file exactly as written, and
a marker that said `!` where a Bound stands would make the gutter derive rather than mark — the line
ADR-0026 draws between the two columns. The flag is where a claim about what the Bound is worth
belongs, and the second form has always worked this way, an `opaque` `destroy` drawing `DESTROY` and
`OPAQUE` in the gutter and `UNBOUNDED` from the pair.

**The orientation states it.** The Bound sentence carried *mandatory* and *refused* and left the third
Kind to inference; the transcript inferred it in the safe direction, then edited in the unsafe one. It
now ends *and on an **opaque** `mutate` it is **accepted and buys nothing**: the same count, so
`review` flags that Step `UNBOUNDED` whether you write one or not* — one sentence still, which is
[ADR-0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)'s rule and the budget
ADR-0093 fixes.

## The seam the fence is expressed at

`TestRunReview_UnboundedReadsOnAnOpaqueMutateWhateverItDeclares`, in `internal/cli`, drives one
repository twice — the transcript's own Step with `bound: 1` and without it — and asserts the whole
`FLAGS` block is the same both times. It fails on the rendering the ticket was filed about rather than
on a helper's return value, which is where the fault was legible.

`TestInstructions_TheBoundRuleSaysWhatAnOpaqueMutateCarries` holds the text to the binary in both
directions, which is what `boundSentence`'s existing fence does for the other two Kinds: the sentence
must name `mutate` and `UNBOUNDED`, `check` must accept the artefact it describes, and `review` must
draw the row it promises. A rule that moves in the binary and not in the text fails there, and so does
a sentence that promises a flag the renderer does not draw.

## Consequences

- **The review's page changes on one shape**, an `opaque` `mutate` carrying a `bound:`, which now draws
  a row it did not draw before. No wire member moves: a `flag` row carries the name, the line and the
  coordinate, and the text is the page's own (§12).
- **`check`, the schemas and §9's derived facts are untouched.** `bound-illegal` still names one
  Operation shape, the Bound fact still has three members, and `bound-exceeded` still guards an
  authored `over:` `values:` list on these Steps.
- **§4, §5, §8, §9, §12 and §13 agree on the set.** §4 says the Bound is accepted and the rendering is
  where the reason lands, §5 carries the argument, §8 says why the `!` reads the key and never what the
  key is worth, §9 says why the derived Bound fact stays `none` on such an Operation and where the
  difference is said instead, §12's bullet reads three forms rather than two, and §13's *unbounded,
  carrying no Bound at all* — true of a shell `destroy` and written of shell Steps — now says what the
  word means on each Kind. `CONTEXT.md`'s **Bound** entry gains the same clause, and §9's own account
  of the orientation's Bound rule reads *a third of the rule* where it read *half*.
- **The `unbounded` name still renders only where the fact holds**, `envelope` remaining the one name
  with an all-clear form. Nothing was added to §12's five standing names.
- **What this does not fix is the reading that produced it.** An author who believes a cleared flag is
  an improved artefact will find another mark to clear; what changed is that this one no longer
  rewards it. The next transcript to author an opaque command Step is what says whether the sentence
  landed.
