package mcp

import "fmt"

// Instructions is the orientation: what `hyper` is, the five artefacts, the
// loop an agent drives them through, the three commands that are the human's
// and why, that a Refusal is final, and a worked example of each artefact (§9,
// ADR-0093, ADR-0095, issues #209 and #211).
//
// **It reaches an agent two ways, and the text is one text.** The `initialize`
// handshake carries it in the field the protocol has for exactly this —
// *instructions describing how to use the server and its features* — and
// `hyper project` writes it to `AGENTS.md` where a repository has none. Neither
// channel is sufficient alone: a client decides when it surfaces `instructions`
// and one harness carries it only inside a tool search, while a file reaches
// only a repository somebody has already run `project` in. Both are the same
// bytes, from here, because two orientations disagree the first time either is
// edited (ADR-0095, internal/cli's RunProject).
//
// **So it is worded for a reader on either surface**, and that is a constraint
// on every sentence in it. It names commands rather than tools — `show` and not
// `run_show` — and it puts `install`, `store init` and `compact` out of reach as
// *the human's*, which is true of both surfaces, rather than as *absent from
// this one*, which is true only of the server. A text that said *no tool here
// writes a Definition* would be read as permission by the agent holding a
// terminal.
//
// **It is a function of the version, and that is load-bearing rather than
// tidy.** The Repository declaration below pins which version of `hyper` may
// act, and the version that would act is the version of the binary the reader
// is standing next to (§9, ADR-0020). A constant here would teach every agent
// to author a pin that Refuses the gate on every repository but the one the
// text was written in.
//
// **It is exported because internal/cli writes it**, `project` being the second
// channel — and the corpus one package over reads it for a second reason, to
// write the artefacts below into a repository and run `check` over them: what
// the orientation teaches is held to checking clean rather than to reading well
// (internal/cli's RunProject, internal/cli/instructions_test.go). The dependency
// runs one way, `internal/cli` → `internal/mcp`, and the surface still knows no
// command (server.go, ADR-0093).
func Instructions(version string) string {
	return fmt.Sprintf(orientation, version)
}

// orientation is the text, with one hole in it: the Repository declaration's
// version pin.
//
// **What it may not contain is a claim the tools do not keep.** It is prose
// with no schema, and nothing validates it against §9. What the cases beside it
// hold is the **list** — that each of §9's orientation facts is stated at all
// (instructions_test.go) — and what a `check` holds is the worked example
// (internal/cli/instructions_test.go). Every sentence between those is on a
// reader, and a sentence that drifts from the surface is worse than an absent
// one: an agent believes it, and spends its next turns repairing an artefact
// this text taught it. Cite §9 when editing one.
//
// **Its length is a design constraint rather than an aesthetic one.** It is
// paid for on every session in every harness — as a handshake field whether or
// not the model reads it, and as a file the harness reads up front — so it
// carries what an agent cannot get any other way and stops. Anything a tool
// call or a command already answers stays one. The example is the shape to
// author against and not a tour: one whole repository, two request shapes, and
// the rules that do not show up in either stated as prose beside them.
const orientation = `# hyper

` + "`hyper`" + ` runs infrastructure automation you author, a human reviews, and it records. **You write the
artefacts; a human approves the diff; only then does anything run.** A Provider knows how to talk to a
kind of system and exposes Operations; a Definition is a named, authority-scoped use of one; a Procedure
sequences Steps; every invocation acts against a Target and produces Records.

A Provider is **data, never code** — reviewing the artefact is reviewing the whole of what will run —
and every effect is performed by ` + "`hyper`" + ` from a closed set of Capabilities only ` + "`hyper`" + ` defines.

You will be asked to **author** something new, to **change** something already reviewed, to **retire**
something, or to **operate** — run a Procedure and read the record back. The loop is the same for all
four, and each has one fact below that you need before you start.

## The loop

1. **Read** — ` + "`providers`" + `, ` + "`targets`" + `, ` + "`provider`" + `. ` + "`operation`" + ` answers **the Manifest's own lines,
   verbatim**: that is the format you author in.
2. **Author with your own file tools.** No ` + "`hyper`" + ` command writes a reviewed artefact: ` + "`hyper`" + ` writes
   what it derives, and you write what is reviewed.
3. **` + "`check`" + `** — offline, no credential, answers file, line, column and an ` + "`error_code`" + `. Repair until
   it is clean.
4. **` + "`review`" + `** — the rendering of what is about to be approved. An artefact you changed is reviewed
   again, against the last Run that read it.
5. **Stop. Hand the diff to the human.** Nothing you authored has authority until somebody has read it,
   and there is no approval command: approval is not a thing an agent grants itself.
6. **` + "`run`" + `**, once they have. ` + "`probe`" + ` is the throwaway question that writes nothing — a ` + "`read`" + ` against
   ` + "`local`" + `, no Definition — for one answer before a Procedure exists.

Read the record back with ` + "`runs`" + `, ` + "`show`" + `, ` + "`records`" + ` and ` + "`changes`" + `. ` + "`changes`" + ` answers *what moved*,
field by field, against the last Run.

**Add a ` + "`cadence:`" + ` and you are not finished**: an unprojected recurrence is ` + "`projection-stale`" + ` at
` + "`check`" + `, and ` + "`project`" + ` is the repair.

**Deleting a Definition does not destroy what it made.** Its Assets become Orphaned — still recorded,
now unreachable by anything ` + "`hyper`" + ` can do, and reported for as long as they stand. Say so before you
delete one; the operator may want them destroyed through a Step first.

## The five artefacts

| | | |
| --- | --- | --- |
| **Manifest** | ` + "`providers/`" + ` | A Provider whole: class, Capabilities, auth scheme, and Operations — each with a Kind (` + "`read`" + `, ` + "`mutate`" + `, ` + "`destroy`" + `), a request, an input schema and a ` + "`record:`" + ` projection. |
| **Target declaration** | ` + "`targets/`" + ` | One system that may be acted on: Kinds admitted, Capabilities granted, hosts permitted, credential source. |
| **Definition** | ` + "`definitions/`" + ` | A named, authority-scoped use of a Provider: Kinds claimed, Targets it may act on. It observes or it effects, never both. |
| **Procedure** | ` + "`procedures/`" + ` | Ordered Steps: one Operation, one Definition, one Target, args beside it. |
| **Repository declaration** | ` + "`hyper.yaml`" + ` | Which ` + "`hyper`" + ` version may act here, how long Records are kept. **Written by ` + "`project`" + `** — never hand-write the ` + "`digest:`" + `. |

**Format** — a strict YAML subset: no anchors, aliases, merge keys, tags or expression language. A
` + "`{hole}`" + ` in a request is filled from a Step's ` + "`args:`" + `; ` + "`$.body.…`" + ` in a ` + "`record:`" + ` is a path into the
response, and an ` + "`over:`" + ` beside it projects a collection into one Record each. An effectful Step may
declare a ` + "`bound:`" + `, the maximum Records it may affect; on a ` + "`destroy`" + ` it is **mandatory**. **The
credential is never in an artefact** — the Target names the environment variable, ` + "`hyper`" + ` puts it in the
header the Manifest's ` + "`auth:`" + ` names, and no rendering prints it.

## Two request shapes

Single host — ` + "`host: \"{from-target}\"`" + ` resolves to the one host the bound Target grants, and ` + "`auth:`" + `
names the header its credential goes in:

` + "```" + `yaml
http: {method: POST, host: "{from-target}", path: /zones/{zone}/records,
       body: {name: "{name}", content: "{address}"}}
auth: {header: {name: Authorization, prefix: "Bearer "}}
` + "```" + `

Many hosts — the same hole expands to every host the Target grants, and ` + "`host-input:`" + ` names which input
picks one per Step. This is the shape for anything that *checks* rather than *creates*.

` + "`providers/site-uptime.yaml`" + `

` + "```" + `yaml
kind: provider
provider: site-uptime
schema-version: 1
class: website
capabilities: [http]
operations:
  check_site:
    kind: read
    repeatability: repeatable
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
      fields: {host: $.host, status: $.status, days_left: $.tls.days_left}
` + "```" + `

Nothing declares which statuses are acceptable, and that is not an omission: a ` + "`read`" + ` never halts on
one, so ` + "`status: $.status`" + ` records a ` + "`503`" + ` as readily as a ` + "`200`" + ` and a later Step's ` + "`when:`" + ` decides what
to do about it. A host the Target does **not** grant comes from an ` + "`enumerations:`" + ` block of the
Operation's own instead.

## The rest of the repository

The Target grants what the Manifest requires; ` + "`hosts:`" + ` and an ` + "`http`" + ` Capability go together or not at all.

` + "`targets/websites.yaml`" + `

` + "```" + `yaml
kind: target-declaration
target: websites
class: website
kinds: [read]
capabilities: [http]
hosts: [example.com, status.example.org]
` + "```" + `

` + "`definitions/uptime-checks.yaml`" + `

` + "```" + `yaml
kind: definition
definition: uptime-checks
provider: site-uptime
kinds: [read]
targets: [websites]
` + "```" + `

` + "`procedures/check-uptime.yaml`" + `

` + "```" + `yaml
kind: procedure
procedure: check-uptime
targets: [websites]
steps:
  - id: example-com
    definition: uptime-checks
    operation: check_site
    target: websites
    args: {host: example.com}
  - id: status-example-org
    definition: uptime-checks
    operation: check_site
    target: websites
    args: {host: status.example.org}
` + "```" + `

**` + "`hyper.yaml`" + ` is written by ` + "`project`" + `** and is the one artefact here to read rather than copy: the pin
comes from the binary that ran it, and the sixty-four zeros stand in for a digest only ` + "`project`" + ` can
know. An invented digest is inert to everything you can call, and **not** inert in a generated workflow,
where it is the line a runner checks fetched bytes against.

` + "`hyper.yaml`" + `

` + "```" + `yaml
kind: repository-declaration
version: %s
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
` + "```" + `

## What halts, and what is merely an answer

**A ` + "`read`" + ` never halts on what came back.** Any status is recorded, and where *no response arrived at
all* — a refused connection, a name that does not resolve — the response object is the host alone. The
Step still mints an Observation and is still ` + "`ran`" + `: a projected field gone quiet is how *down* is
recorded. The Run does **not** stop, and later Steps are not skipped.

**An effectful Operation is the opposite**: it completes on ` + "`2xx`" + ` and halts on everything else, a
` + "`destroy`" + ` completing on ` + "`404`" + ` besides. Where no response arrived it is *attempted, world untouched*.

What does halt a ` + "`read`" + ` is its **projection** — a path that found nothing. An empty collection and a
wrong path are not the same fact.

**Narration is not an outcome.** ` + "`probe`" + ` prints ` + "`no response arrived: …`" + ` and exits ` + "`0`" + `; a Step whose
host was unreachable narrates the same way and is ` + "`ran`" + `. Read the Disposition, never the prose beside
it, and never generalise from a ` + "`probe`" + ` to a Run.

## Three of the sixteen commands are the human's

` + "`install`" + `, ` + "`store init`" + ` and ` + "`compact`" + ` have no tool on the MCP surface, and are not yours to type into a
terminal either. One line puts all three on the far side: you may read the record and add to it, and
**may not create it, prune it, or bring anything new into the repository**.

- ` + "`install`" + ` is the single point at which third-party data enters — installing and running in one turn is
  a supply-chain sequence with no human between acquisition and effect.
- ` + "`store init`" + ` creates the record. Creating it is the human's act; your part is to say it has not happened.
- ` + "`compact`" + ` removes from the record permanently: the one command that would let you prune the account
  you are held to.

**Do not reach around this.** Where a call Refuses naming one of the three, say so and stop; the human
runs it.

## A Refusal is final

A Refusal is a guardrail declining a well-formed call, and it carries its own remediation. **The same
call retried refuses identically.** No argument anywhere widens your authority — no bypass flag, no
approval command, no per-Run exemption. Change what it names, or say what needs to happen and let the
human do it. A malformed call is different: that is an error carrying the sentence a person would have
read, and it is worth fixing and repeating.

## Leave a note for the next agent

` + "`hyper project`" + ` writes this text to ` + "`AGENTS.md`" + ` where a repository has none, and never overwrites one
that stands. **So where an ` + "`AGENTS.md`" + ` exists and says nothing about ` + "`hyper`" + `, offer to add a section to
it** — that is the one case in which nothing here reaches the next agent.

An ` + "`AGENTS.md`" + ` is **not a reviewed artefact**: none of the five, no authority, nothing about a Run reads
it, and ` + "`check`" + ` does not count it. It is a note, and it lands in a diff like every other file you write.
`
