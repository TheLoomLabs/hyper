# `hyper` never stores a secret

`hyper` has no vault. A Target declaration names the environment variables its Auth scheme's
credential slots are filled from; `hyper` resolves them once per Run, holds them in memory, and never
writes one anywhere. Nothing is encrypted at rest because nothing is at rest. Where a secret must
come from a password manager, the invocation is wrapped by the tool that already owns it — `op run --
hyper run …`, `direnv`, `aws-vault exec --` — so the resolution happens outside `hyper`'s process
entirely.

We chose this because the alternative is a product we do not want to be in. Storing a secret means
owning a KDF choice, key rotation, recovery, escrow, and permission checking on every read, and none
of that is infrastructure automation. Swamp shows what the half-built version costs: PBKDF2 paid on
every get, no AAD on the envelope, no rotation and no recovery at all, an auto-generated key that is
two concatenated UUIDs, and a key file whose permissions are validated when written and never when
read — so losing one file permanently destroys every secret it held. The second reason is the one
that made it easy: with credentials resolved from the environment in both places, the laptop and the
GitHub Actions runner are not two code paths with matching guardrails, they are the same code path.
That is what makes the rule that the environment is never an axis (ADR-0001's neighbour in the safety
model) true rather than aspirational.

## Consequences

- **No secret ever enters the Store.** The Store is a branch that gets pushed, so the rule has to
  hold for outputs as well as inputs: a Manifest declares which of an Operation's output fields are
  secret, and `hyper` writes the Record with a redaction marker carrying *presence only* — no digest,
  no length. A digest would let a diff report that a credential rotated, which is genuinely useful,
  and it would also publish an offline-crackable oracle for any secret a human chose rather than a
  machine generated. The diff says *redacted, unknown* instead, which is the honest answer.
- **A secret-valued output requires a sink, and the absence of one is a Refusal.** The invocation
  supplies a path for secret outputs or it does not; a Step whose Operation declares one Refuses when
  it does not. The obvious phrasing — refuse when unattended — is illegal here, because that is an
  `is-CI` guess and the environment is never an axis. Expressed as a property of the invocation it is
  testable on a laptop by omitting the flag, and in Actions the generated workflow simply supplies
  nothing. The sink is written `0600`, and a path resolving inside the repository working tree is
  refused: a generated root password written beside the artefacts is one `git add -A` from being
  permanent.
- **Some Assets are not re-readable.** An Asset whose only handle is the secret it contains — an API
  key returned once at creation — cannot be recovered from the record. You rotate rather than
  recover. `skip-if-recorded` is unaffected, because the Record exists; only the value is gone.
- **Credential positions are suppressed positionally, never scrubbed.** Because Auth schemes are a
  closed set `hyper` implements, `hyper` knows exactly which header, query parameter, or body position
  it filled, and never renders those in an error, a Journal entry, or the NDJSON stream. A
  scrubber that pattern-matched for secrets would be an advisory check, which is the category
  ADR-0004 exists to eliminate. This only holds because a Provider cannot invent a scheme.
- **A literal in a credential position is a load-time error.** Positional and mechanical, in the same
  family as declared-equals-derived Capabilities. It makes committing a secret structurally
  impossible in the credential slot and does nothing about one pasted into an unrelated field; that
  remains a pre-commit hook's job, not `hyper`'s.
- **The CI secret surface is generated, not hand-written.** The workflow `hyper` projects for a
  Procedure derives its `env:` block from the Target declarations of the Targets that Procedure
  transitively touches, and the Actions secret and the environment variable share one name. Adding a
  Target to a Procedure therefore makes a new secret appear in a diff rather than in YAML nobody
  reviewed.
- **`hyper` reads credentials and does not acquire them.** OIDC federation is not implemented. If a
  Target ever needs it, it arrives as Auth schemes `hyper` owns, not as a third-party action inside a
  workflow `hyper` generated — that would put the supply-chain problem back into the artefact whose
  reviewability is the point. Until then, a federated cloud in CI needs a long-lived credential in
  Actions secrets, which is worse than the ecosystem norm and is the accepted price of not building
  speculatively.
- **There is no audit log for credential reads.** Every Journal entry already carries each Step's
  Target and Disposition, so which credentials were reached, when, and by which Run is a query over
  the Journal rather than a second append-only artefact needing its own storage, retention, and
  tamper-evidence.
