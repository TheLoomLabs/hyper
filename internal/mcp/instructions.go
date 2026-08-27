package mcp

import "fmt"

// Instructions is the orientation the `initialize` handshake carries: what
// `hyper` is, the five artefacts, the loop an agent drives them through, the
// three commands that have no tool and why, that a Refusal is final, and one
// worked example of each artefact (§9, ADR-0093, issue #209).
//
// **It is the only thing on this surface that reaches an agent before its first
// tool call.** Every tool carries a description, and `operation` goes further —
// it answers *the Manifest's own lines, verbatim*, which teaches the authoring
// format at the moment a caller needs it (§9). Both arrive with a call already
// in mind. What none of them can say is that `hyper` is here at all, what the
// five artefacts are, or which order to do things in, and an agent that does not
// know those shells out or invents a config file. MCP's `initialize` result has
// a field for exactly this — *instructions describing how to use the server and
// its features* — and filling it puts the orientation in the protocol, with no
// file in the user's repository and no setup beyond the client config they
// already need.
//
// **It is not `hyper` speaking first.** It is a field of the response to a
// request the client made, so nothing is initiated and ADR-0021's own test is
// not engaged; the argument is ADR-0093's and is not restated here.
//
// **It is a function of the version, and that is load-bearing rather than
// tidy.** The Repository declaration below pins which version of `hyper` may
// act, and the version that would act is the version of the server the client
// started (§9, ADR-0020). A constant here would teach every agent to author a
// pin that Refuses the gate on every repository but the one the text was
// written in.
//
// It is exported for the corpus one package over, which writes the artefacts
// below into a repository and runs `check` over them: what the handshake
// teaches is held to checking clean rather than to reading well
// (internal/cli/instructions_test.go).
func Instructions(version string) string {
	return fmt.Sprintf(orientation, version)
}

// orientation is the text, with one hole in it: the Repository declaration's
// version pin.
//
// **What it may not contain is a claim the tools do not keep.** It is prose
// with no schema, and nothing validates it against §9. What the cases beside it
// hold is the **list** — that each of §9's five orientation facts is stated at
// all (instructions_test.go) — and what a `check` holds is the worked example
// (internal/cli/instructions_test.go). Every sentence between those is on a
// reader, and a sentence that drifts from the surface is worse than an absent
// one: an agent believes it, and spends its next turns repairing an artefact
// this text taught it. Cite §9 when editing one.
//
// It is deliberately shorter than the documentation it stands in front of. A
// client pays for it on every session whether or not the model reads it, so it
// carries what an agent cannot get any other way and stops: the five facts §9
// names, and one example to author against. Anything a tool call already
// answers stays a tool call.
const orientation = `# hyper

` + "`hyper`" + ` is a tool for AI-authored, human-reviewable infrastructure automation. **You author
the artefacts; a human reviews them like code before anything runs; every effect on the
world is recorded.** Its spine is procedural rather than desired-state: a Provider knows how
to talk to a kind of system and exposes Operations, a Definition is a named, authority-scoped
use of one, a Procedure sequences Steps, and every invocation acts against a Target and
produces Records.

A Provider is **data, never code** — reviewing the artefact is reviewing the whole of what
will run — and every effect it describes is performed by ` + "`hyper`" + ` itself from a closed set of
Capabilities that only ` + "`hyper`" + ` defines.

## What you will be asked to do

Four verbs, and the loop below is the same for all four. Each has one fact you need before
you start.

- **Author** something new — a Provider, a Definition, a Procedure, a Target declaration.
- **Change** something that exists. A reviewed artefact is reviewed again: ` + "`review`" + ` renders it
  against the last Run that read it, so what you altered is what a human reads.
- **Retire** something. **Deleting a Definition does not destroy what it made.** Its Assets
  become Orphaned — still recorded, still ` + "`hyper`" + `'s account of things it created, and now
  unreachable by anything ` + "`hyper`" + ` can do. They are reported for as long as they stand. Say
  that before you delete one; the operator may want the Assets destroyed through a Step first.
- **Operate** — run a Procedure, then read the record back with ` + "`runs`" + `, ` + "`run_show`" + `, ` + "`records`" + `
  and ` + "`changes`" + `. ` + "`changes`" + ` is the one that answers *what moved*, field by field, against the
  last Run as its baseline.

**If you add a ` + "`cadence:`" + ` to a Procedure, you are not finished.** A declared recurrence with no
projected workflow beside it is ` + "`projection-stale`" + ` at ` + "`check`" + `. Call ` + "`project`" + `, which is on this
surface precisely so you can repair what you caused.

## The loop

1. **Read what is here.** ` + "`providers`" + ` lists every Provider and ` + "`targets`" + ` lists what this
   repository may reach. ` + "`provider`" + ` reports one Manifest's facts and the Operations it exposes;
   ` + "`operation`" + ` answers **the Manifest's own lines, verbatim** — which is the format you are
   expected to author in.
2. **Author with your own file tools.** No tool here writes a Definition, a Procedure or a
   Target declaration: ` + "`hyper`" + ` writes what it derives, and you write what is reviewed.
3. **` + "`check`" + `.** It is offline, needs no credential, and answers file, line, column and an
   ` + "`error_code`" + ` — positioned so that the next act is an edit. Repair and check again until
   it is clean.
4. **` + "`review`" + `.** The rendering of what is about to be approved, artefact by artefact.
5. **Stop there and hand the diff to the human.** This is the step to get right. Nothing you
   authored has authority until somebody has read it, and there is no approval tool here
   because approval is not a thing an agent grants itself.
6. **` + "`run`" + `**, once they have. ` + "`probe`" + ` is the throwaway question that writes nothing — a
   ` + "`read`" + ` Operation against ` + "`local`" + `, with no Definition — for when you want to see one answer
   before a Procedure exists.

## The five artefacts

| | | |
| --- | --- | --- |
| **Manifest** | ` + "`providers/`" + ` | A Provider whole: its class, the Capabilities it requires, its auth scheme, and its Operations — each with a Kind (` + "`read`" + `, ` + "`mutate`" + `, ` + "`destroy`" + `), a request, an input schema and a ` + "`record:`" + ` projection. |
| **Target declaration** | ` + "`targets/`" + ` | One system that may be acted on: which Kinds it admits, which Capabilities it grants, which hosts it permits, and where its credential is read from. |
| **Definition** | ` + "`definitions/`" + ` | A named, authority-scoped use of a Provider: which Kinds it claims and which Targets it may act on. It observes or it effects, never both. |
| **Procedure** | ` + "`procedures/`" + ` | An ordered set of Steps, each one Operation through one Definition against one Target, with the arguments beside it. |
| **Repository declaration** | ` + "`hyper.yaml`" + ` | Which version of ` + "`hyper`" + ` may act here, and how long Records are kept. |

The format is a strict YAML subset: no anchors, no aliases, no merge keys, no tags, no
expression language. A ` + "`{hole}`" + ` in a request is filled from a Step's ` + "`args:`" + `; ` + "`$.body.…`" + ` in a
` + "`record:`" + ` is a path into the response. An effectful Step may declare a ` + "`bound:`" + ` — the maximum
number of Records it may affect — and on a ` + "`destroy`" + ` Step it is mandatory.

## What is not reachable from here, and why

Thirteen of ` + "`hyper`" + `'s sixteen commands have a tool here, each named for the command it carries
— ` + "`run_show`" + ` for ` + "`show`" + ` is the one name that differs, a bare ` + "`show`" + ` naming nothing in a
client's flat namespace. The other three have **no tool at all**, and one line puts all three
on the far side of it: an agent may read the record and add to it, and **may not create it,
prune it, or bring anything new into the repository**.

- ` + "`install`" + ` is the single point at which third-party data enters the repository. An agent
  that can install can author against what it installed and run it in the same turn, which
  is a whole supply-chain sequence with no human between acquisition and effect.
- ` + "`store init`" + ` creates the record. Creating it is the human's act, and your part in it is to
  say that it has not happened.
- ` + "`compact`" + ` removes from the record permanently — the one command that would let you prune
  the account you are held to.

**Do not shell out to them.** The absence is the guardrail, and reaching around it with a
terminal is the exact bypass it exists to prevent. Where a tool Refuses naming one of the
three, say so and stop; the human runs it.

## What halts, and what is merely an answer

This is the rule agents get wrong, and it decides how you read a result.

**A ` + "`read`" + ` never halts on what came back.** Any status is recorded — a ` + "`503`" + ` as readily as a
` + "`200`" + ` — and where **no response arrived at all** (a refused connection, a name that does not
resolve) the response object is **the host and nothing else**. The Step still mints an
Observation and its Disposition is still ` + "`ran`" + `: an answer that is an absence is still an answer,
and a projected field that has gone quiet is how *down* is recorded. A Run does **not** stop
because a host was unreachable, and Steps after it are not skipped.

**An effectful Operation is the opposite**: it completes on ` + "`2xx`" + ` and halts on everything else, a
` + "`destroy`" + ` completing on ` + "`404`" + ` besides. Where no response arrived its Disposition is *attempted,
world untouched*.

What still halts a ` + "`read`" + ` is its **projection** — a path that found nothing, a collection that was
not there. An empty collection and a wrong path are not the same fact.

**Narration is not an outcome.** ` + "`probe`" + ` prints ` + "`no response arrived: …`" + ` and exits ` + "`0`" + `; a Step
whose host was unreachable narrates the same way and is ` + "`ran`" + `. Read the Disposition and the
outcome, never the prose beside them, and never generalise from a ` + "`probe`" + ` to what a Run will do.

## A Refusal is final

A Refusal is a guardrail declining a well-formed call, and it comes back carrying its own
remediation. **The same call retried refuses identically.** Retrying it unchanged is not a
strategy, and no argument anywhere widens your own authority — there is no bypass flag, no
approval tool, and no per-Run exemption. Read what the Refusal says to change, change that,
and call again; or say what needs to happen and let the human do it.

An error is different: a malformed call comes back as a protocol error carrying the sentence
a person would have read, and that one is worth fixing and repeating.

## A worked example

Five artefacts that check clean, for a Provider that talks HTTP to a Proxmox server. Nothing
about them is special — they are what ` + "`check`" + ` accepts, written out — and the shape is what to
copy: the request block with its holes, the ` + "`auth:`" + ` scheme naming a header, the ` + "`record:`" + `
projection, and the Target that grants what the Manifest requires.

**` + "`hyper.yaml`" + ` is written by ` + "`project`" + `, not by hand**, and it is the one artefact below to read
rather than copy. ` + "`project`" + ` derives the pin from the binary that ran it and freezes the digest of
the release beside it; the sixty-four zeros here stand in for a value only ` + "`project`" + ` can know.
A digest you invent is inert to everything you can call, and it is **not** inert in a generated
workflow, where the digest is the line a runner checks fetched bytes against.

` + "`hyper.yaml`" + `

` + "```yaml" + `
kind: repository-declaration
version: %s
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
` + "```" + `

` + "`providers/proxmox.yaml`" + `

` + "```yaml" + `
kind: provider
provider: proxmox
schema-version: 1
class: proxmox
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "PVEAPIToken="}
operations:
  list_vms:
    kind: read
    repeatability: repeatable
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /api2/json/nodes/{node}/qemu
    input:
      type: object
      properties:
        node: {type: string}
    record:
      over: $.body.data
      identity: $.vmid
      fields: {vmid: $.vmid, name: $.name, status: $.status}
  create_vm:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 5m
    http:
      method: POST
      host: "{from-target}"
      path: /api2/json/nodes/{node}/qemu
      body: {vmid: "{vmid}", name: "{name}", cores: "{cores}", memory: "{memory}"}
    input:
      type: object
      properties:
        node: {type: string}
        vmid: {type: string}
        name: {type: string}
        cores: {type: string}
        memory: {type: string}
    record:
      identity: "{vmid}"
      fields: {task: $.body.data}
  destroy_vm:
    kind: destroy
    repeatability: repeatable
    deadline: 5m
    http:
      method: DELETE
      host: "{from-target}"
      path: /api2/json/nodes/{node}/qemu/{vmid}
    input:
      type: object
      properties:
        node: {type: string}
        vmid: {type: string}
` + "```" + `

A second Manifest, because **the request shape above is not the only one**. Where an Operation
reaches many hosts rather than one, ` + "`host: \"{from-target}\"`" + ` expands to every host the bound
Target grants and ` + "`host-input:`" + ` names which of the Operation's inputs picks one per Step. This is
the shape to copy for anything that *checks* rather than *creates*.

` + "`providers/site-uptime.yaml`" + `

` + "```yaml" + `
kind: provider
provider: site-uptime
schema-version: 1
class: website
capabilities: [http]
operations:
  check_site:
    kind: read
    deadline: 10s
    http:
      method: GET
      host: "{from-target}"
      path: /
      host-input: host
    input:
      type: object
      properties:
        host: {type: string}
    record:
      identity: $.host
      fields:
        host: $.host
        status: $.status
        days_left: $.tls.days_left
` + "```" + `

` + "`targets/websites.yaml`" + `

` + "```yaml" + `
kind: target-declaration
target: websites
class: website
kinds: [read]
capabilities: [http]
hosts: [example.com, status.example.org]
` + "```" + `

Nothing there declares which statuses are acceptable, and that is not an omission — a ` + "`read`" + ` never
halts on one, so ` + "`status: $.status`" + ` records a ` + "`503`" + ` as readily as a ` + "`200`" + ` and a later Step's
` + "`when:`" + ` decides what to do about it. Where the host is **not** one the Target grants — an
enumeration of the Operation's own — declare it under ` + "`enumerations:`" + ` in the Manifest and let the
hole draw on that instead.

` + "`targets/proxmox-lab.yaml`" + `

` + "```yaml" + `
kind: target-declaration
target: proxmox-lab
class: proxmox
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [pve.lab.example.com]
auth:
  token: {env: PROXMOX_API_TOKEN}
` + "```" + `

` + "`definitions/lab-vms.yaml`" + `

` + "```yaml" + `
kind: definition
definition: lab-vms
provider: proxmox
kinds: [mutate]
destroy: [destroy_vm]
targets: [proxmox-lab]
` + "```" + `

` + "`procedures/provision-lab-vm.yaml`" + `

` + "```yaml" + `
kind: procedure
procedure: provision-lab-vm
targets: [proxmox-lab]
steps:
  - id: create
    definition: lab-vms
    operation: create_vm
    target: proxmox-lab
    args:
      node: pve1
      vmid: "9001"
      name: lab-9001
      cores: "2"
      memory: "4096"
    bound: 1
` + "```" + `

The credential is never in an artefact. The Target names the environment variable it is read
from, ` + "`hyper`" + ` puts it in the header the Manifest's ` + "`auth:`" + ` scheme names, and no rendering
anywhere prints it.
`
