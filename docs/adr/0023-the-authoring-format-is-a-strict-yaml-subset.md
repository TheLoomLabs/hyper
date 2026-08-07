# The authoring format is a strict YAML subset

All five reviewed artefacts — Definition, Procedure, Manifest, Target declaration, Repository
declaration — are written in YAML 1.2 core with a large part of the language **rejected at load**: no
anchors, no aliases, no merge keys, no tags, no multi-document files, no implicit type resolution, and
unknown keys refused rather than ignored. Scalars are typed by the schema at their position, never by
what they look like. Files are named `.yaml` and are YAML syntactically; they are not YAML
semantically, and that is a knowing trade for editor and LSP support.

We chose this because the primary author of these artefacts is an AI, which makes **corpus a
first-class design input** rather than a matter of familiarity. A format with no training data behind
it throws away the one thing that makes AI authoring cheap, and the MCP surface has already bet on the
format being learnable from a Manifest source dump with no other teaching. Against that, YAML's
footguns are real — but every one of them is removable by *rejecting*, which is the move this project
makes everywhere else: unknown-key rejection, closed enumerations, closed Capability sets, no bypass.
A restriction is reviewable; an invention is unlearnable.

Banning anchors and aliases is not tidiness. It is the review model: the line you read must be the
value that was used. An alias means it is not, and the same objection independently rejected `$ref` in
the schema subset and recursive descent in the path grammar.

## Considered options

- **HCL.** Line-oriented, block-structured, no significant whitespace, and a real corpus from
  Terraform — on the face of it the best fit. Rejected for a specific reason rather than on taste: the
  corpus *is* Terraform's, so a model trained on it writes `${var.x}`, `for` expressions and `try()` by
  reflex. Adopting HCL would mean banning precisely the constructs the corpus teaches, which is worse
  than having no corpus. It also imports the desired-state and `plan` mental model that ADR-0010
  exists to reject.
- **A `hyper`-native line-oriented syntax.** Total control and zero footguns, bought with zero corpus
  and an implementation nobody else can help with. The footguns were the weaker half of the argument
  once rejection was on the table.
- **JSON or JSONC.** Unambiguous and canonical, and the worst review surface of the four: closing
  braces occupy lines the gutter has nothing to say about, and quoting noise fights the artefact-is-
  the-review-surface rule.
- **Implicit typing minus the dangerous subset** — keeping `true`/`false`/numbers while blocking
  `yes`/`no`/`on`/`off`, sexagesimals and leading-zero octals. Rejected because a blocklist is an
  advisory check, which is the shape the extension model exists to eliminate. Schema-directed typing
  kills the Norway problem by construction instead.

## Consequences

- **There is no untyped position anywhere in any artefact**, which is what makes schema-directed
  typing total. Operation inputs use a closed JSON Schema subset with `additionalProperties: false`
  forced, and the artefact schemas are `hyper`'s own. That same totality is what later lets a mapping
  in a scalar position be read unambiguously as a reference.
- **Operation outputs get no schema at all** — only the projection. A Manifest already declares Record
  cardinality, the identity field and a set of named projection paths; an output schema would be a
  second representation of the same fact that can disagree with the first, which is the reasoning that
  refused stored durations in the Journal.
- **`$ref` and the schema combinators (`allOf`, `oneOf`, `if`/`then`/`else`) are rejected**, on the
  anchors argument in a different hat.
- **A template hole at the start of a value must be quoted.** `host: {region}.api.example.com`
  unquoted is a YAML flow mapping. This needs no special-casing: schema-directed typing reports
  *expected string, got object* at that line, which is already a good error.
- **The layout is fixed and not configurable** — `definitions/`, `procedures/`, `targets/`,
  `providers/`, and `hyper.yaml` at the root — with a mandatory `kind:` key that must agree with the
  directory through a fixed mapping. Declared *and* derived, on the extension model's own rule.
  Directory alone would mean the review surface's first line does not say what you are reading;
  `kind:` alone would mean any stray YAML in the repository becomes an artefact by having a line
  pasted into it. A configurable layout is the axis ADR-0014 deleted.
- **Only a Manifest carries a format version.** ADR-0020 pins the binary repository-wide and enforces
  it in both environments, so every artefact in the working tree is read by exactly the binary the
  repository pins and there is no skew to version against. The line is not Store-versus-artefact but
  *inside versus outside the pin's reach*: a Manifest is the only artefact authored by someone who
  cannot see this repository's pin. This discharges the extension model's requirement for an explicit
  schema version on the four repo-authored artefacts, which predated ADR-0020.
