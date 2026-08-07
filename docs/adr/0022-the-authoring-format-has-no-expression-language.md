# The authoring format has no expression language

There is no expression language anywhere in `hyper`'s five artefacts. No interpolation syntax, no
operators, no functions, no arithmetic, no regular expressions, no boolean algebra. What replaces it
is three closed forms: a **path** for naming a value that lives elsewhere, a **closed operator set**
for predicates, and a **template hole** whose legality is decided by the position it appears in.
Everything a Definition, Procedure, Manifest, Target declaration or Repository declaration needs to
say is said in the format's own data syntax.

We chose this because every other tool in this space has an expression language and every one of them
pays for it in the same place. Swamp carries three separate CEL environments, and the best decision in
its repository is that the third one — for security decisions — is a permanently *sealed*, strictly
weaker dialect of the other two. ADR-0004 took the principle one step further for authority checks:
the security decision has no evaluator at all, because a grammar that does not exist cannot grow. This
ADR takes the same step for the rest of the format. The pressure to add a helper, an operator, a
coercion, one more builtin is constant and each addition is individually reasonable; a language that
was never started cannot accumulate them.

The argument that actually decided it is about **review**, not about simplicity. A one-line expression
reads identically whether it widened by a little or by everything — `age > 14d` becomes `age > 1d` in
one character, on one line, and the reviewer's eye has nothing to catch on. A structured predicate
list gives every predicate its own line, which is exactly what the review gutter was built to
annotate and what the Comparison needs in order to render a selector change as anything more
informative than "the selector changed". The format's shape and the review surface's shape are the
same decision.

## Considered options

- **A general expression language (CEL or similar) in all four positions** — Manifest projection,
  Expansion selector, Step condition, Step argument. Rejected on the review argument above, and on
  Swamp's own case law: its accessors read live disk state on every evaluation, which the design docs
  frame as a feature and which means two Steps in one run can observe different values. Once an
  expression can call something, it can call something that looks at the world, and the invariant that
  every fact influencing a Run is visible in the artefact is gone.
- **Four purpose-built total micro-grammars**, one per position, each carrying only its own
  vocabulary. Genuinely defensible, and rejected for being four things to learn, four things to
  version, and four places for the same helper to be requested. The closed-form answer covers all four
  positions with one path grammar and one operator set.
- **Regular expressions in predicates.** Rejected specifically rather than as a side effect. A regex
  is an expression language with a catastrophic-backtracking footgun attached, and it defeats the
  review argument more completely than anything else considered: `.*` is two characters that mean
  everything. `starts_with` and `ends_with` are the bounded form of the real use case, which is
  prefix-matching on names.
- **Disjunction (`any_of`) in predicates.** Rejected on safety rather than simplicity, which is the
  reasoning most at risk of being lost. A **Bound** is per-Step. An `OR` inside one selector puts two
  disjoint populations under a single count; splitting it into two Steps gives each disjunct its own
  Bound, its own gutter row, and its own line in the Comparison. Refusing disjunction is the Bound
  working correctly, not a limitation reluctantly accepted.

## Consequences

- **A path is a selection, never an evaluation.** The grammar is `$`, `.member`, `["member"]` and
  nothing else. Recursive descent (`..`) is rejected because a path whose meaning depends on data the
  reviewer cannot see while reviewing is the same objection as a YAML alias and a JSON Schema `$ref` —
  the third appearance of one argument. Array indexing (`[n]`) is rejected because an index into an
  upstream array *is* the "identity is array position" hazard that made Record identity a declared
  Manifest fact. Iteration is not in the grammar at all: it is declared once, in an Operation's
  `record.over`, and implied by a cardinality of `series`.
- **A reference is a mapping, not a string.** `{step: <id>, path: $.…}` and `{item: $.…}` are the only
  two reference kinds. Because typing is schema-directed, a mapping where the schema expects a scalar
  is unambiguously a reference and a mapping where the schema expects an object is a literal object —
  so "is this a literal or a reference" is a type question rather than a parsing question. A whole
  object can therefore never be referenced; references sit at scalar leaves.
- **The ceiling gains three more named victims.** An author who needs disjunction in a selector,
  a regex match, or arithmetic on a response field is stuck until `hyper` ships something. This is the
  wall stated from the other side, exactly as ADR-0004 accepts it, and it joins OIDC federation and
  load-shaped retry on the list that argues for a process by which the closed set grows.
- **The format is teachable from a Manifest dump.** With no expression language there is no second
  syntax an agent must learn separately from the artefact it is reading, which is what the MCP
  surface's decision to return Manifest **source text** was betting on.
