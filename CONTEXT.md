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
run-once on an effectful Operation and `repeatable` on a `read`. Declared in the Manifest, never
inferred, and never overridden downstream. Which values an Operation may declare follows its Kind,
two of the three deciding by reading a projection: `skip-if-recorded` is `mutate`-only, and run-once
cannot be written at all. `skip-if-recorded` decides per Record rather than per Step, an Expansion
holding one series per member, so a Step may skip the members whose Assets stand and call for the rest.
A run-once Step is refused under a Cadence, its second occurrence having nobody present to read the
Refusal.
_Avoid_: Idempotency, Retry policy, Rerun mode

**Opaque**:
A property of a Capability whose effects `hyper` cannot describe, such as running a command, carried
by every Operation whose request uses it and declared beside none of them. Orthogonal to Kind, so an
Opaque Operation still declares whether it destroys.
_Avoid_: Shell, Untyped, Raw, Escape hatch

**Capability**:
One effect `hyper` can perform on a Manifest's behalf, drawn from a closed set that only `hyper`
defines. A Manifest declares the Capabilities it requires and a Target declaration grants them; an
Operation reaches only what both name. One is Opaque and reserved to Providers `hyper` ships, so what
an Operation cannot describe and who may write one are one fact.
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
secret and cannot invent a scheme or a slot. A scheme decorates a request and never performs one, and
what it writes is always a request header — so `hyper` knows the position the secret occupies and never
renders it. A Provider naming no scheme sends no credential.
_Avoid_: Auth method, Credential type, Signer, Protocol

**Extension**:
A Provider authored and distributed by someone other than `hyper` itself. Being a Manifest, it
contains no code; it may not shadow a built-in Provider's name, and the Capabilities reserved to
built-ins are never granted to it.
_Avoid_: Plugin, Package, Module

### Authored artefacts

**Definition**:
A named, authority-scoped use of a Provider: which Kinds it claims and which Targets it may act on. It
observes or it effects, never both, since the Records it writes take their type from the Kinds it claims.
It carries no argument values — those belong to the Step, where they are read beside the Bound. The
durable artefact an agent authors and a human reviews; nothing is invoked except through one.
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
than keeping one of its own. It is also the declaration the last Journal entry is read against, which
is what makes staleness readable rather than merely bounded.
_Avoid_: Schedule, Trigger, Interval, Frequency

**Repository declaration**:
The reviewed artefact granting authority over the repository as a whole: which version of `hyper` may
act on it, and how long Records are kept. It admits only facts that govern every Run and belong to no
Procedure, Definition, or Target.
_Avoid_: Config, Settings, Manifest, Policy, Lockfile

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
The Target meaning this machine, whose declaration the repository authors like any other's — declaring
the hosts it grants, so there is no unconstrained-reach Target. Its name is reserved rather than its
file: it is the Target a Probe binds, it carries no credential slot, and other declarations may name its
class, each a further name for the same machine with its own grant.
_Avoid_: Default, None, Empty

**Expansion**:
The resolution of a Step's selector to the concrete Records it will act on, scoped to the Step's own
Definition and Target, and by Kind: a `read` Step may expand over Observations, an effectful one only
over Assets. Anything `hyper` did not create must therefore be named by literal identifier, which is a
selector form of its own. It is ordered as it resolves — by the artefact where the selector is a literal
list, otherwise by the Record name — so *which three of the five* is a fact and not a race. Its members
are one Record identity each: every one projects an identity no other member of that Expansion projects,
and two that are one identity under the fold are refused before the first call or halt the Run at the
projection. They are not one *Record* each, though — an Operation of `series` cardinality projects many
out of one response — so what a Step concluded about counts Records where `expanded_to` counts members.
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
A Record of something `hyper`'s own effect reached and is therefore accountable for. That effect is the
whole test — usually because `hyper` created the thing, equally where it changed one it did not, and
equally where it ended one it never saw — and a thing merely observed is never an Asset.
_Avoid_: Resource, Holding, Managed resource

**Orphaned Asset**:
An Asset whose Definition no longer exists. Expansion needs a Definition, so nothing in `hyper` can
reach it again; it is never collected and is reported for as long as it stands.
_Avoid_: Dangling resource, Leaked resource, Abandoned resource

**Tombstone**:
The version of an Asset recording that what it described was destroyed, and what its last known state
was. Terminal for the Asset's life rather than for the series: recreating under the same identity
writes a further version above it. It may be a series' first version, where a destruction reached
something `hyper` had no record of, and then it carries no last known state at all.
_Avoid_: Deletion marker, Soft delete

**Secret sink**:
The destination an invocation supplies for output an Operation declares secret. A Run reaching a Step
that produces such output Refuses when none is supplied, which is a fact about the invocation rather
than about the environment it runs in.
_Avoid_: Secret output, Vault write, Capture file

**Run**:
A single execution of a Procedure, and the unit against which change is reviewed. There is no other
kind: a single Operation is reached only through a Step, and a Probe executes one without being a Run.
_Avoid_: Execution, Invocation, Job

**Probe**:
A `read` Operation invoked against `local` without a Definition, writing no Record and no Journal
entry. It is a lookup rather than a Run, so it has no Trigger, no Provenance and no Disposition, and
it can never be scheduled, sequenced into a Procedure, or used as a Comparison baseline. Its reach is
the one thing it does not escape: the host it asks for is the Target's to grant, as a Step's is.
_Avoid_: Query, Check, Ad-hoc run, One-shot

**Trigger**:
What caused a Run to happen — a clock or a person — and which executor it happened on. A fact about
the occasion rather than about the code, and the only thing that distinguishes a world that has not
changed from one nobody has looked at.
_Avoid_: Source, Cause, Origin, Event

**Refusal**:
A terminal Run outcome in which a guardrail declined before any effect reached the world. Usually
before any Step exists at all, a Run re-running every static check at its start — so it is a fact about
the Run rather than about a Step, held on the Run's outcome and never on a Step's, and the Step it may
cite is an artefact coordinate rather than something that ran. Distinct from failure, which means the
world resisted, and from losing the Store, which needs no act of anyone's to clear.
_Avoid_: Rejection, Denial, Block, Abort

**Journal**:
The append-only series of Run entries — one per Run, carrying its outcome, its Provenance, and every
Step's Disposition. The only place a Refusal or an unconfirmed attempt is recorded, since neither
writes a Record. An entry carrying no account of how it ended is **open**: the Run may be in flight or
its process may be gone, and `hyper` never guesses which. An entry may be closed by its own Run or by a
later one, and holding both is a Contested entry.
_Avoid_: Log, Audit trail, Run state, Checkpoint

**Contested entry**:
A Journal entry holding two accounts of how a Run ended: its own Run's, and a later Run's inference that
it had died. It is what a Run that was reaped while still alive leaves behind, and both accounts stand —
the inference is a true fact about the Run that made it. The entry's outcome is its own Run's, an
observation being what an inference was an inference about; `hyper` never chooses between two accounts
of what the *world* did, and holding both files is what keeps this from being that.
_Avoid_: Conflict, Disputed run, Split entry, Zombie run

**Disposition**:
What a Step did in a Run: ran, skipped as already recorded, skipped by condition, refused, never
reached, attempted with its outcome unknown, or attempted with the world provably untouched — the two
skips being distinct because only the first is Repeatability evidence, and the two attempts because only
the first leaves any doubt that the world was touched — together with the Record identities it acted on
and what `hyper` itself did to reach that outcome, which is the only account of a Pattern's attempts,
pages and poll iterations. Held by the Journal rather than by any Record.
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
The record of which code produced something: Procedure revision, Definition revision, Manifest digest,
origin digest, repository revision, and the version of `hyper` that performed it — which, Providers
being data, is the only code that ran. The Procedure revision is the top-level Procedure's, the only one
a Run has exactly one of. Every Record version carries the whole of it; the Journal carries it split by
scope, each member written where it has exactly one value, so a Run that wrote no Record still says
which code performed it and a Procedure spanning two Definitions has no revision it must invent.
_Avoid_: Audit, History, Lineage

**Comparison**:
The rendering of one Run against the previous Run of the same Procedure: the Assets `hyper` changed,
the Observations the world changed, and the code that changed between the two. Retrospective by construction, so it
reports what happened rather than proposing what would.
_Avoid_: Diff, Drift, Plan, Changelog, Delta
