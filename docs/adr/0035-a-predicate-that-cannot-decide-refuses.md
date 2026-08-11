# A predicate that cannot decide Refuses

A projected field has no declared type. §3 gives an Operation's output no schema at all — only the
projection a Manifest declares — so what a `fields:` entry actually holds is discovered when it is
read and never before. A predicate can therefore be handed a value it cannot compare: `older_than:
14d` against a number, `greater_than: 10` against a string, `starts_with: preview-` against an
object. **Where that happens, `hyper` Refuses. It never treats the value as not matching.** The same
answer covers the two ways an author can write a predicate whose truth does not depend on the value at
all — `in: []`, which can never hold, and `starts_with: ""`, which always does — and both of those are
refused at load, being authored rather than discovered.

The reading a competent implementer reaches unaided is **not-matching**, arrived at without deciding
anything: a filter returns a boolean, the comparison fails, the item is excluded, the loop moves on.
Alongside it come two habits from the same source — evaluating the conjuncts of an AND lazily, and
letting a mismatched type mean *not equal*. All three are the ordinary way to write a filter, and
together they mean **an API changing a field's type silently changes what a `destroy` Step reaches.**
`not_equals: "active"` against a field that moved from a string to a number is *true* under the unaided
reading, so a selector written to exclude the live resources stops excluding them, on the morning the
vendor shipped, with nothing declined and nothing rendered. The narrowing direction is no better, only
quieter: a selector that reaches nothing reports a Run that completed.

`hyper` can afford to Refuse here because of where the values come from. A selector reads the Store and
a condition reads the Records earlier Steps of this Run acted on, and both resolve **before the Step's
call goes out** (§6), so declining is available in the sense §5 requires — a guardrail declining before
any effect reached the world. A polling `until:` is the one root where it is not: it reads a response
mid-call, after the world was already touched. There the existing rule applies unchanged — §6's *when a
projection does not resolve*, which halts the Run, carries no `error_code` because nothing declined,
and names the path and what it found. So two of the three roots contribute a Refusal and the third
contributes a halt, and that is one rule read against three positions rather than three rules.

## Considered options

- **Not-matching, silently.** Rejected above. It is worth naming that the objection is not tidiness:
  the failure is invisible on every surface the tool has, because a Record that quietly failed to
  compare is indistinguishable from one that compared and did not match.
- **Not-matching, with the excluded Records reported.** A softer version, and it fails on the same
  ground the first does the moment a Run is unattended (§10) — a report nobody reads is the
  `allowFailure` §6 declined, wearing a note.
- **Refusing only for the operators where the widening direction is reachable** — `not_equals`,
  `absent`. Rejected because it makes the rule a table a reviewer must hold rather than a sentence, and
  because the narrowing direction is a silent failure too: it is a `destroy` Step reporting success
  having reached nothing.
- **Declaring types on a Manifest's `fields:`, making the whole question static.** Rejected as the
  second representation of a declared fact this specification has refused everywhere else — the type
  would live beside the path that produces it and the two can disagree, which is §7's argument against
  stored durations and §3's against output schemas. It also cannot be complete: it would state what a
  Provider author believes the API returns, checked against nothing.

## Consequences

- **One `error_code`, `predicate-type-mismatch`, stated by §4 and §6 both.** It is the second member
  two sections state, on `bound-exceeded`'s exact argument: what names a Refusal is the check that
  declined and never the moment it ran. §4 fires it where the fault is authored and knowable offline —
  a `timestamp` operand under `greater_than`, `exists: false`, an `in:` that is empty, of one member, or
  of mixed types, a `starts_with:` or `ends_with:` of the empty string, a predicate against a field the
  Manifest declares secret — and §6 fires it against a stored value at Expansion.
- **A predicate list does not short-circuit.** Every conjunct is evaluated against every candidate.
  Short-circuiting would make *whether the Run Refuses* depend on the order the author happened to
  write two conjuncts in and on which Records the Store happened to hold that morning — the same
  artefact behaving differently for a reason no reviewer can see on the line — and it would restore
  the silence this decision removes for any Record an earlier conjunct caught first.
- **A predicate against a declared-secret field is refused at load.** Such a field is written to the
  Store as the constant `"<secret>"` (§7), so the value cannot depend on the Record and the predicate
  reaches the whole series or none of it. It is the constant-predicate case arriving without the author
  doing anything that looks wrong, and it is free to catch: `secret:` is a declared list beside a
  declared field set.
- **A `field:` naming a field no Operation of the Provider projects stays `reference-unresolvable` and
  grows no new code** (§4). It is the same check a reference already gets, and it must be a load error
  rather than a predicate that never holds: `absent` over an undeclared field is true for every version
  in the series, so one typo turns a filtered `destroy` into an unfiltered one with only the Bound in
  front of it.
- **No operator coerces.** A value is compared as the type it is, which is what keeps this rule a
  statement about types rather than about a coercion table. A timestamp arrives in a Store file as a
  string, so `starts_with: "2026-"` against one is legal and does what it looks like — `hyper` is not
  pretending to know more about that value than that it is a string.
