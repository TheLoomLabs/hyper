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
Capabilities it requires — together with an implementation.
_Avoid_: Model, Driver, Plugin, Connector

**Manifest**:
The declared half of a Provider: data rather than code, stating the Provider's schemas, Operations,
and required Capabilities. It is what a human reviews in place of the implementation.
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
A permission a Provider must declare in its Manifest in order to be granted it.
_Avoid_: Permission, Grant, Scope

**Extension**:
A Provider authored and distributed by someone other than the tool itself.
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

### The world

**Target**:
A concrete system an Operation acts on, and the unit of both blast radius and credentials. A
Definition declares the Targets it accepts; an invocation binds one.
_Avoid_: Environment, Account, Host, Context

**Local**:
The reserved Target meaning this machine and the public internet, holding no credentials.
_Avoid_: Default, None, Empty

**Expansion**:
The resolution of a Step's selector to the concrete Records it will act on. Only Assets are reachable
by Expansion; anything `hyper` did not create must be named by literal identifier.
_Avoid_: Resolution, Matching, Fan-out, Globbing

### The record

**Record**:
An immutable, versioned series of what an Operation produced, identified by its Target, its
Definition, and a name. Every Record is either an Observation or an Asset.
_Avoid_: Data, Artifact, Output

**Observation**:
A Record of a fact read from the world at a point in time. `hyper` is not accountable for what it
describes, and never reconciles it against an Asset.
_Avoid_: Reading, Sample, Fact, Resource

**Asset**:
A Record of something `hyper` created and is therefore accountable for. Having been created by
`hyper` is the whole test — a thing merely observed is never an Asset.
_Avoid_: Resource, Holding, Managed resource

**Tombstone**:
The terminal version of an Asset, recording that what it described was destroyed and what its last
known state was.
_Avoid_: Deletion marker, Soft delete

**Run**:
A single execution of an Operation or a Procedure, and the unit against which change is reviewed.
_Avoid_: Execution, Invocation, Job

**Refusal**:
A terminal Run outcome in which a guardrail declined a Step before any effect reached the world.
Distinct from failure, which means the world resisted.
_Avoid_: Rejection, Denial, Block, Abort

**Journal**:
The append-only series of Run entries — one per Run, carrying its outcome, its Provenance, and every
Step's Disposition. The only place a Refusal or an unconfirmed attempt is recorded, since neither
writes a Record.
_Avoid_: Log, Audit trail, Run state, Checkpoint

**Disposition**:
What a Step did in a Run: ran, skipped as already recorded, refused, never reached, or attempted with
its outcome unknown. Held by the Journal rather than by any Record.
_Avoid_: Status, State, Result, Outcome

**Provenance**:
The record, carried by every Record version, of which code produced it: Definition revision, Manifest
digest, Extension digest, and repository revision.
_Avoid_: Audit, History, Lineage
