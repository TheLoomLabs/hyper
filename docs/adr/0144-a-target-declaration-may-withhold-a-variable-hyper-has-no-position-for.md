# A Target declaration may withhold a variable `hyper` has no position for

A Target declaration may carry `withhold:`, a list of plain environment variable names. The set a
`shell` Operation's child is denied is the union, over every Target declaration in the repository, of
the credential slots and the `withhold:` entries. A declaration named `local` may carry one.

We chose this because §13 named a residue and had no way to write it down. `hyper` removes a
credential from a command's environment by the position it occupies — a Target declaration's slot —
which is the same knowledge that lets it suppress rather than scan (ADR-0007). A secret the repository
never named occupies no position, so it passes to any command and appears on no surface. The common
case is not exotic: an invocation wrapped in `op run --` or `aws-vault exec --` — which is the vault
integration `hyper` has, the process environment being the interface every secret manager speaks —
leaves that tool's *own* credential in the environment beside the one it fetched. `OP_SERVICE_ACCOUNT_TOKEN`
is strictly more powerful than the `HCLOUD_TOKEN` it was there to produce, and it is a name the author
knows and `hyper` cannot.

It grants nothing, which is what makes it cheap. `auth:` says where a credential is *resolved from*,
and `env:` is reserved to that position because a general ambient-input channel is authority arriving
after review under another name (ADR-0008). `withhold:` says only that a variable is *removed*: it
resolves nothing, reads nothing, and carries no value, so there is no authority for a hole to arrive
through and nothing for `env:` to stand in. It is a subtraction, and the only thing a wrong one costs
is a command that cannot see a variable.

The scope is the repository rather than the bound Target, which is not a widening for safety but the
rule §11 already states for credential slots, transferred verbatim: decided that way the set is a fact
about the tree a reviewer reads off it, where a set derived from the Steps a Run walked would differ
between two Runs of one repository and would grow a hole the day a Procedure stopped binding a Target
whose variable was still set.

## Considered options

- **Nothing — §13 states the limit honestly, and the remedy is already there.** §13 pointed at declaring
  the variable as a credential slot on a second class-`local` declaration. Rejected: that makes `hyper`
  resolve a credential it has no use for and compose it into a header nothing sends. It is a lie in a
  reviewed artefact told to get a side effect, and a reviewer reading `auth: {token: {env:
  OP_SERVICE_ACCOUNT_TOKEN}}` would reasonably conclude that something authenticates with it.
- **An allowlist: a child gets the variables a declaration names and nothing else.** Correct in
  principle and rejected on cost. It breaks every command needing `PATH`, `HOME`, `SSH_AUTH_SOCK`, or a
  cloud CLI's own configuration, and the list that fixes that is a configuration surface ADR-0014
  forbids — arriving, worse, as something an author must get exhaustively right rather than something
  they add to.
- **Glob patterns — `OP_*`.** Rejected: a glob is a small language, and §12's posture is closed sets. The
  failure modes are asymmetric, which decides it. A literal an author forgot leaks one named variable; a
  pattern subtly wrong silently withholds `PATH`, and what they debug is a command that cannot find its
  own binary.
- **Scan the child's environment for values that look like credentials.** Rejected on ADR-0007's ground,
  where it has been rejected every other time it was proposed.

## Consequences

- **This does not close the residue, it makes it writable.** A variable no declaration names still
  reaches any command. What changed is that naming one no longer requires claiming it authenticates
  something.
- **No new error code.** `withhold:` joins the Target declaration's closed key set, so a scalar where a
  list belongs is `schema-mismatch` at the line it was written on, like every other key. A name matching
  no variable in the environment is not a fault at all — subtracting what is not there is the same
  environment, and a check that reported it would be reading the invoking environment to judge an
  artefact.
- **A reviewer reads the removal off the tree.** It is the union over the repository, so the question
  *what does a command not see* is answered by the declarations rather than by tracing which Steps a Run
  reached.
- **`hyper` still never reads what it withholds.** The key carries names, and the removal is by name.
  Nothing here resolves a value, which is why a `withhold:` entry costs no credential gate, no Refusal,
  and no line in a Journal entry.
