# hyper

`hyper` is a tool for AI-authored, human-reviewable infrastructure automation: an agent writes the
artefacts, a human reviews them like code, and every effect on the world is recorded so that what
changed between one run and the next can be read back.

Its spine is procedural, not desired-state. A **Provider** knows how to talk to a kind of system and
exposes **Operations**; a **Definition** is a named, configured use of one; a **Procedure** sequences
**Steps**; every invocation acts against a **Target** and produces **Records**.

## Language

### Capability

**Provider**:
A named capability for talking to one kind of system — its schemas, its Operations, and the
Capabilities it requires. A Provider is a Manifest and nothing else; `hyper` supplies every effect it
describes.
_Avoid_: Model, Driver, Plugin, Connector

**Manifest**:
The whole of a Provider: data rather than code, stating its schemas, Operations, and required
Capabilities. There is no implementation behind it to review, which is why reviewing it is enough.
_Avoid_: Schema, Spec, Descriptor

**Operation**:
A single callable exposed by a Provider, carrying a declared Kind.
_Avoid_: Method, Action, Command

**Kind**:
An Operation's declared blast radius — `read`, `mutate`, or `destroy`. Always declared in the
Manifest, never inferred from an Operation's name.
_Avoid_: Type, Verb, Category

**Repeatability**:
An Operation's declared behaviour when a Procedure is run again: `repeatable` (invoke it again),
`skip-if-recorded` (skip while the Asset it would produce still stands), or, when undeclared,
run-once. Declared in the Manifest, never inferred.
_Avoid_: Idempotency, Retry policy, Rerun mode

**Opaque**:
A trait on an Operation whose effects `hyper` cannot describe, such as an arbitrary shell command.
Orthogonal to Kind, so an Opaque Operation still declares whether it destroys.
_Avoid_: Shell, Untyped, Raw, Escape hatch

**Capability**:
One effect `hyper` can perform on a Manifest's behalf, drawn from a closed set that only `hyper`
defines. A Manifest declares the Capabilities it requires and a Target declaration grants them; an
Operation reaches only what both name.
_Avoid_: Permission, Grant, Scope

**Pattern**:
A behaviour `hyper` performs around a call — pagination, polling to a terminal condition, retry —
which a Manifest parameterises but does not implement. A Pattern may never change an Operation's Kind
or the number of Records it affects, and retry may follow only a failure that provably happened before
the request was sent, so no Pattern can change the number of times the world was touched.
_Avoid_: Strategy, Policy, Middleware, Hook

**Auth scheme**:
A way of authenticating a request, implemented by `hyper` and chosen by name in a Manifest. The
Manifest supplies its parameters and the Target its credentials, so Provider authors never handle a
secret and cannot invent a scheme. Because the set is closed, `hyper` knows which positions of a
request carry the secret and never renders them.
_Avoid_: Auth method, Credential type, Signer

**Extension**:
A Provider authored and distributed by someone other than `hyper` itself. Being a Manifest, it
contains no code; it may not shadow a built-in Provider's name, and the Capabilities reserved to
built-ins are never granted to it.
_Avoid_: Plugin, Package, Module

### Authored artefacts

**Definition**:
A named, configured use of a Provider, declaring which Targets it may act on. The durable artefact an
agent authors and a human reviews; nothing is invoked except through one.
_Avoid_: Model, Instance, Config, Binding, Profile

**Procedure**:
An ordered set of Steps, declaring the full set of Targets it may touch. Procedures contain Steps
directly — there is no grouping level between them — and compose by invoking one another.
_Avoid_: Workflow, Playbook, Pipeline, Job

**Step**:
One entry in a Procedure: a single Operation, invoked through a Definition, against one Target.
_Avoid_: Task, Job, Stage

**Bound**:
The maximum number of Records an effectful Step may affect, declared by the Step's author. Mandatory
on a `destroy` Step, where an absent Bound means unbounded rather than unchecked.
_Avoid_: Limit, Cap, Quota, Threshold

**Cadence**:
A Procedure's declared recurrence, stated as a UTC cron expression. It is a lower bound on staleness
rather than a promise of coverage, and `hyper` projects it into an external executor's clock rather
than keeping one of its own.
_Avoid_: Schedule, Trigger, Interval, Frequency

### The world

**Target**:
A concrete system an Operation acts on, and the unit of both blast radius and credentials. A
Definition declares by name the Targets it accepts; an invocation binds one.
_Avoid_: Environment, Account, Host, Context

**Target declaration**:
The reviewed half of a Target: which Kinds it accepts, which Capabilities it grants, which endpoint
it names. An artefact in the repository, holding no credentials, so every static check runs without
them.
_Avoid_: Target config, Connection, Profile

**Target credentials**:
The unreviewed half of a Target: the secrets its Auth scheme requires, named by the declaration as
environment variables and resolved once per Run. They live wherever the environment already keeps
them, never in the repository and never at rest inside `hyper`.
_Avoid_: Vault, Secret store, Keychain

**Local**:
The reserved Target meaning this machine and the public internet, holding no credentials.
_Avoid_: Default, None, Empty

**Expansion**:
The resolution of a Step's selector to the concrete Records it will act on, scoped by Kind: a `read`
Step may expand over Observations, an effectful one only over Assets. Anything `hyper` did not create
must therefore be named by literal identifier before it can be changed.
_Avoid_: Resolution, Matching, Fan-out, Globbing

### The record

**Record**:
An immutable, versioned series of what an Operation produced, identified by its Target, its
Definition, and a name. Every Record is either an Observation or an Asset.
_Avoid_: Data, Artifact, Output

**Head**:
The current version of a Record, derived from the order of the versions themselves rather than
declared by a marker. Nothing in the Store points at it, which is what lets two environments write
the same series without contending.
_Avoid_: Latest, Current, Tip

**Observation**:
A Record of a fact read from the world at a point in time. `hyper` is not accountable for what it
describes, and never reconciles it against an Asset.
_Avoid_: Reading, Sample, Fact, Resource

**Asset**:
A Record of something `hyper` created and is therefore accountable for. Having been created by
`hyper` is the whole test — a thing merely observed is never an Asset.
_Avoid_: Resource, Holding, Managed resource

**Orphaned Asset**:
An Asset whose Definition no longer exists. Expansion needs a Definition, so nothing in `hyper` can
reach it again; it is never collected and is reported for as long as it stands.
_Avoid_: Dangling resource, Leaked resource, Abandoned resource

**Tombstone**:
The version of an Asset recording that what it described was destroyed, and what its last known state
was. Terminal for the Asset's life rather than for the series: recreating under the same identity
writes a further version above it.
_Avoid_: Deletion marker, Soft delete

**Secret sink**:
The destination an invocation supplies for output an Operation declares secret. A Step producing such
output Refuses when none is supplied, which is a fact about the invocation rather than about the
environment it runs in.
_Avoid_: Secret output, Vault write, Capture file

**Run**:
A single execution of an Operation or a Procedure, and the unit against which change is reviewed.
_Avoid_: Execution, Invocation, Job

**Probe**:
A `read` Operation invoked against `local` without a Definition, writing no Record and no Journal
entry. It is a lookup rather than a Run, so it has no Trigger, no Provenance and no Disposition, and
it can never be scheduled, sequenced into a Procedure, or used as a Comparison baseline.
_Avoid_: Query, Check, Ad-hoc run, One-shot

**Trigger**:
What caused a Run to happen — a clock or a person — and which executor it happened on. A fact about
the occasion rather than about the code, and the only thing that distinguishes a world that has not
changed from one nobody has looked at.
_Avoid_: Source, Cause, Origin, Event

**Refusal**:
A terminal Run outcome in which a guardrail declined a Step before any effect reached the world.
Distinct from failure, which means the world resisted.
_Avoid_: Rejection, Denial, Block, Abort

**Journal**:
The append-only series of Run entries — one per Run, carrying its outcome, its Provenance, and every
Step's Disposition. The only place a Refusal or an unconfirmed attempt is recorded, since neither
writes a Record. An entry carrying no outcome is **open**: the Run may be in flight or its process may
be gone, and `hyper` never guesses which.
_Avoid_: Log, Audit trail, Run state, Checkpoint

**Disposition**:
What a Step did in a Run: ran, skipped as already recorded, refused, never reached, or attempted with
its outcome unknown — together with the Record identities it acted on and what `hyper` itself did to
reach that outcome, which is the only account of a Pattern's attempts, pages and poll iterations. Held
by the Journal rather than by any Record.
_Avoid_: Status, State, Result, Outcome

**Store**:
Where Records and the Journal live: a branch of the repository, written by every environment that
runs. It is `hyper`'s account of the world rather than part of it, so it is never a Target and
reaching it costs no Capability.
_Avoid_: Database, State, Backend, Cache

**Compaction**:
The removal of interior Observation versions, to the extent a reviewed artefact permits it. It never
removes evidence — no Asset, no Tombstone, no Journal entry — and it reclaims tree size and scan cost
rather than clone size.
_Avoid_: GC, Pruning, Vacuum, Cleanup, Retention

**Provenance**:
The record, carried by every Record version, of which code produced it: Definition revision, Manifest
digest, Extension digest, repository revision, and the version of `hyper` that performed it — which,
Providers being data, is the only code that ran.
_Avoid_: Audit, History, Lineage

**Comparison**:
The rendering of one Run against the previous Run of the same Procedure: the Assets `hyper` changed,
the Observations the world changed, and the code that changed between the two. Retrospective by construction, so it
reports what happened rather than proposing what would.
_Avoid_: Diff, Drift, Plan, Changelog, Delta
