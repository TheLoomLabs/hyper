# A value is read against the schema at its position

A scalar is **read** as the type the schema at its position declares — its characters against that
type's text form — and never compared with a type of its own. The reading is the same at load, where
the value is authored, and at Expansion, where it arrives from a reference. A value that will not read,
an input the schema declares that nobody supplied, and a value outside its `enum` are one check and one
code, `schema-mismatch`. Every input an Operation declares is supplied.

## The problem this decides

ADR-0023 states that scalars are typed by the schema at their position, never by what they look like,
and cites the resulting error as though it existed: *schema-directed typing reports* expected string,
got object *at that line, which is already a good error*.

It does not exist. `unknown-key` names the **key** half of schema conformance — `additionalProperties:
false` being forced at every level — and nothing in fourteen sections names the **instance** half. No
code covers a value that fails the schema at its position: not `type`, not `required`, not `enum`, not
`const`, and not at any position of any artefact. §3 says a Step names *the `args:` the Operation's
input schema requires* and §9 says a Probe's `--input` values are *each typed by the Operation's
declared input schema*; neither is a check, and §4's grammar list has no member to be refused under.

ADR-0078 is what made it urgent rather than untidy. Before it, an `args:` value disagreeing with its
input's declared type was harmless — everything reached the wire as text. After it the declared type
decides the bytes, so a string landing in an `integer` input either produces a JSON number that is not
one or a serialisation `hyper` cannot perform, on the one path §4 says has no oracle and in the
direction that is silent.

## The decision

**Reading, not comparison.** ADR-0023's *typed by the schema, never by what it looks like* is a reading
rule and this states what it reads: the value's characters as the declared type, in the text form §12
fixes for each. The quoting YAML required is lexical and not part of the value, so `"2592000"` and
`2592000` are one value at an `{type: integer}` position. The Norway problem stays dead by construction
rather than by a blocklist: `boolean`'s text is `true` and `false` and nothing else, so `NO` at a
boolean position reads as nothing at all.

**One check, one code, two sites.** *A value satisfies the schema at its position* is `schema-mismatch`
(`error_code` 49 → 50), stated by §4 where the value is authored and offline and by §6 where an `args:`
value arrives from a reference and meets the type its input declares. It is `predicate-type-mismatch`'s
shape and `bound-exceeded`'s, both already split across those two sections. *Satisfies* means the same
reading at both, which is what makes it one check rather than two under one name.

**Every declared input is supplied**, so `required` leaves the input-schema subset, and `const` leaves
with it as `enum` of one member. Six keywords become four: `type`, `enum`, `properties`, `items`.

**The `shell:` block carries no keys**, written `shell: {}`, and the argv is the Operation input named
`command`.

**A Probe's `--input` stays a usage error**, exit `2`, carrying no `error_code`, joining the two faults
§9 already states there.

## Considered options

- **Compare the value's YAML core type against the declared one.** The reading a competent implementer
  reaches unaided, and the one this rejects. It makes `"2592000"` and `2592000` two values at a position
  ADR-0078 has just finished saying the quote is silent at — ADR-0023 mandates that quote for a parsing
  reason, so it was never available to carry meaning. It also puts the format's most common AI-authoring
  slip, a stray quote, on the Refusal path while changing nothing about what the artefact means.
- **Bind the stored value's own JSON type at the §6 site.** Defensible on its own — §7's canonical
  encoding makes string-versus-number a real distinction in the Store — and it splits the check: §4
  lenient about quoting, §6 strict, under one code. It also makes a Refusal turn on whether a remote API
  answered `"2592000"` or `2592000`, which is a fact about the API and one §4 states `hyper` has no
  oracle for. §12 already reads a stored value leniently across a type line for the same reason:
  *`integer` and `number` are two scalar types where an input schema constrains what a caller supplies,
  and one comparison where a value has already come back*.
- **A code specific to an Operation's input.** Leaves `bound: five`, `deadline: forever` and
  `schema-version: "1"` unnamed, which is the state that produced this decision, and would be widened by
  the first ticket that authors one of them wrong.
- **Absorb `unknown-key`.** One schema, one check, taken to its end. Rejected on §12's own line:
  `additionalProperties: false` is **forced rather than authored**, so it is not a keyword of the subset
  at all. The two also send a reader to different places — every `schema-mismatch` names a schema
  position to go and read, and an unknown key names none.
- **Let an unsupplied optional input omit its sink** — the body key, the query parameter, the header, the
  argv word. Rejected on ADR-0078's own sentence: *a reviewer reads a request's structure off the
  Provider and its values off the Procedure*. This is a Step's `args:` deciding which keys a request
  has.
- **Narrow ADR-0078's array ban so a hole may carry a list into an argv.** The smaller fix for the
  `shell` contradiction, and it reopens the rule for one caller. Deleting the key is smaller still: a
  key whose only legal content is `"{command}"` is a second spelling that can only ever disagree with
  the first, which is why `opaque`, an output schema and a per-Operation `capabilities:` do not exist
  either.
- **Exempt the built-in Providers from `check`.** Refused by ADR-0004: a Provider is data, and data
  `check` may not read is §10.5's advisory analyzer wearing the tool's own badge.

## Consequences

- **The built-in `shell` Provider did not load, and that is how this was found.** All six of its
  Operations wrote `shell: {command: "{command}"}` against `input: {properties: {command: {type: array,
  items: {type: string}}}}` — a hole naming an input declared `array`, refused by §4 as
  `manifest-inconsistent`, stated as a decision by ADR-0078, and marked *refused* in §12's own scalar
  table. Three statements, one worked Manifest, and the Manifest is the one Provider `hyper` ships.
- **`manifest-inconsistent` goes eleven shapes to twelve**: an input the Operation declares that no
  position of its request reaches — no hole, no `host-input:`, and not the argv the `shell` Capability
  names. Harmless before this decision and not after, since every declared input must now be supplied
  into something that goes nowhere.
- **The input-schema subset shrinks and nothing replaces the clause.** Nothing nests below `type`,
  `enum`, `properties` and `items`: no hole may name an `object` or `array` input, `host-input:` names a
  whole host, and the one array sink is the `shell` argv — so a nested `properties:` is refused by the
  quantified check above rather than by a list here saying which nestings are illegal, on the finding
  ADR-0080 closed one ticket earlier.
- **§12's text-form column is read in both directions.** It was written for what leaves on the wire and
  is now also what a scalar is read *as*. One rule, two directions, rather than an authoring table
  minted beside a serialisation one.
- **A stored `"0755"` fills an `integer` input with 755**, and there is no line for a reviewer to read.
  ADR-0078's leading-zero trap has the same defence and no better one on the authored half — it is on
  the page — and this half has it in the Store. Stated as a limit in §13 rather than defended.
- **An API with a genuinely optional parameter is two Operations**, and the cost is combinatorial: two
  optional parameters are three Operations and five are thirty-one. It joins §13's wall, whose count had
  read *seventeen* against nineteen entries since ADR-0078 added two without moving the word.
- **`error_code` grows by one and by a property.** Fifty members; thirty-one from §4's static checks;
  three members stated by §4 and §6 both; five requiring a Step to have been reached. `schema-mismatch`
  satisfies the set's standing property — it declines before any call goes out, both its sites resolving
  at or before Expansion (ADR-0072).
