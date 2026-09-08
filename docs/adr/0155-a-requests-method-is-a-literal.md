# A request's `method:` is a literal

An `http:` block's `method:` admits no template hole. A hole there is `hole-illegal` (§4), refused at
the position rather than at the source — the name inside it is never read — which puts `method:` beside
an Auth scheme's parameters as the second of §12's two positions where a hole is refused outright. It
follows that `method:` reaches no Operation input, so an input named only in a hole there is
`manifest-inconsistent` like any other input nothing reaches. Issue [#279](https://github.com/TheLoomLabs/hyper/issues/279).

Nothing moves for an artefact that checks clean: no artefact key, no closed-set member, no new
`error_code`, no wire member, no rendering. **Two `check` rows appear for one that does not**, and
both are new.

## What the state was

`method:` was an ordinary template-hole position on the load side and no position at all on the call
side, and each half was individually defensible.

`checkHTTPRequest` ran `checkOrdinaryHoles` over it, so a hole naming an input the Operation declares
checked clean and a hole naming nothing earned `hole-illegal` — `check` was not ignoring `method:`, it
was validating it and naming the position. `collectReachedNames` collected its holes beside `path:`'s,
so an input named only there counted as **reached** and escaped the *no position of this Operation's
request reaches it* row. And `Request.Build` copied the method through unfilled, on the line above the
four that call `Fill`, `fillParameters` and `serialiseBody`:

```go
call := Call{Host: host, Method: r.Method}
```

So a Manifest writing `method: "{verb}"` with `verb` declared was accepted by every check, supplied by
every Step binding the Operation (ADR-0081), read by nothing, and reached `net/http` with the braces
still in it. `{` and `}` are not `tchar`, so the token never reached the wire and the Run died on a Go
error with no `error_code` — a guardrail's job done by the standard library, in the one place §6 says a
Refusal belongs to the Run (ADR-0061).

`Build`'s own doc comment stated the invariant this broke: *every hole in a request names an Operation
input and every declared input is supplied, so a hole with nothing behind it is a Manifest `check` has
already refused.* For `method:` the second clause was false — `check` refused nothing, and `Build` did
not look.

**The corpus had not decided this; it inherited it.** §12's positions closed at three and named two of
them, `host:` and `auth:`; the third was *every other position resolves only to an Operation input*, a
general rule that reached `method:` by omission. §3's `http:` paragraph described `method:` only as a
key the block carries. No record named the position, and every fixture in the tree wrote a literal
method in either direction — so there was no golden that could have caught it.

## The decision

**What a call *does* is written in the artefact a reviewer reads.** That is ADR-0029's decision for a
host and ADR-0051's for a command's first argv word, arriving a third time on the request's verb, and
the answer is the same one.

The reason it is not a matter of taste is the Kind. A Manifest declares an Operation's `kind:`, and
every guardrail in §5 hangs off that declaration: the Target's `kinds:` grant is checked against it,
the review draws `DESTROY` from it, and `destroy`'s mandatory Bound, its `over:` selector and its
named-Operation requirement all key on it. Fill the hole and the **verb** comes from outside the
artefact while the **Kind** stays inside it, so a `kind: read` Operation whose `method:` is a
Step-supplied input sends `DELETE`, passes the `read` grant, draws no flag, reaches none of `destroy`'s
requirements, and the world loses a record. That is not a hole in a position; it is the blast-radius
axis moved out of the file the reviewer is reading.

Refusing it costs nothing anyone has written. No fixture, no shipped Manifest and no line of the
specification used a hole there, which is why the decision is cheap now and would not have been later.

**The refusal is at the position, not at the source.** `checkMethodHoles` never reads the name inside
the hole, and the case that holds it writes a hole naming a declared input — the arrangement an author
actually produces, and the one every ordinary position admits. A check that refused only holes naming
nothing would be `checkOrdinaryHoles` again under a new name.

**`collectReachedNames` drops `method:` in the same change.** It is the quieter half of one defect and
not a second repair: a position that reaches no input cannot mark one reached, and leaving it in would
silence the `manifest-inconsistent` row on the strength of a hole that has already been refused. An
author who writes the hole now gets both rows — the position that refused it, and the input left
declared and read by nothing — which is exactly the shape an Auth scheme's refused hole already has.

## What was considered

- **Fill it.** One `Fill` call at `internal/capability/http.go:229`, and the reading §12's general rule
  already implied. Rejected on the Kind argument above. It is the cheaper diff and the more surprising
  rule: nothing else in the format lets a value from outside an artefact change what a call does, and a
  reviewer reading `kind: read` beside `method: "{verb}"` has been told two things that do not agree.
- **Fill it, and check the filled verb against the Kind at Run time.** Rejected as a guardrail in the
  wrong place. It would need a mapping from verb to Kind that HTTP does not have — `POST` is a mutate
  at one API and a destroy at another — and it moves a decision `check` can make offline into a Run,
  which is the direction §4 exists to push against. It also fails the review: a flag drawn from what a
  Run discovers is a flag the reviewer of the diff never saw.
- **Refuse, but as a code of its own.** Rejected. `hole-illegal` is *a hole where one may not be*, and
  `auth:` already carries the outright-refusal case under it. A second code would make a reader think
  the two refusals differ in kind when they differ only in which key.
- **Leave it, and fence the Go error into a Refusal with an `error_code`.** Rejected: it names the
  symptom. The Manifest is knowably wrong offline, and a Run that Refuses where `check` was clean is
  the arrangement ADR-0061 and §4 both spend their length avoiding.

## Consequences

- **Two `check` rows on a Manifest that used to pass.** `hole-illegal` at `<op>.http.method`, and
  `manifest-inconsistent` at the input the hole named. `internal/cli/testdata/check/a-hole-written-into-method`
  is the golden: one Provider, one hole, two rows. Both codes already had fixtures, so the closed set
  and the coverage test are untouched.
- **A hole naming an input declared `object` or `array` changes code at this one position.** It was
  `manifest-inconsistent` (ADR-0078's rule, read by `checkOrdinaryHoles`) and is now `hole-illegal`,
  because the position is refused before the name is read. That is the intended ordering: the fault is
  the hole, and the type of a name nothing may read is not the thing to report.
- **`Build` keeps its line and gains the reason for it.** `Method` is copied through rather than
  filled, and the doc comment now says that is the position's meaning rather than an omission — the
  invariant it always asserted is true again.
- **A `method:` that is a literal but not an HTTP token is left where it is.** `method: "GET "` still
  reaches `net/http` and still fails there. It is a different fault with a different shape — an author
  writing a malformed literal, not a value arriving from outside the artefact — and this record
  decides the second and not the first.

  _ADR-0156 amends this:_ the limit was measured and closed. A literal that is not RFC 9110's `token`
  is `manifest-inconsistent` at the same key, on `checkPathDelimiters`' argument rather than on this
  record's. What is written here about the two faults being different in shape stays true — it is why
  they are two checks on one key and not one — and what changes is that the second is no longer
  standing.
- **No sealed run is owed.** The change is enforced rather than taught in
  [`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md)'s sense: `check` now Refuses,
  and the golden and the package cases hold it. No clause of the orientation moves, no existing row is
  reworded, and no task in `scripts/acceptance/tasks/` writes a hole into `method:` — the rows are new
  text, but nothing an agent already reads has changed under it.
