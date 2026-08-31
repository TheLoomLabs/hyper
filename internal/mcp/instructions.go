package mcp

import "fmt"

// Instructions is the orientation: what `hyper` is, the five artefacts, the
// loop an agent drives them through, where the record lives, the three commands
// that are the human's and why, that a Refusal is final, and a worked example
// of each artefact (§9, ADR-0093, ADR-0095, ADR-0096, ADR-0101, ADR-0113,
// issues #209, #211, #212, #218 and #233).
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
// **And a rule stated here is stated with its exception, or not stated.** The
// Bound sentence is where that was learned: it said *mandatory on a `destroy`*
// and stopped, which is true and is half the rule — an opaque `destroy` refuses
// a Bound — so an agent authoring the only `destroy` a fresh repository can
// write was taught the artefact `check` declines. That claim is now held to
// `check` itself rather than to a reader, in one sentence, by
// TestInstructions_TheBoundRuleIsTheOneCheckHolds (§9, ADR-0101, issue #218).
//
// **Its length is a design constraint rather than an aesthetic one.** It is
// paid for on every session in every harness — as a handshake field whether or
// not the model reads it, and as a file the harness reads up front — so it
// carries what an agent cannot get any other way and stops. Anything a tool
// call or a command already answers stays one. The example is the shape to
// author against and not a tour: one whole repository, two request shapes, and
// the rules that do not show up in either stated as prose beside them.
//
// **Which shape is carried whole is a decision and not a preference.** The
// effectful one is, because its rules are the ones nothing else states — a
// `record:` fixed by Kind, an `identity:` that must resolve before the call, a
// `destroy:` claim naming Operations, a selector, a Bound — and because the
// multi-host `read` is the task a fresh repository's first agent is asked for,
// which an example carrying it whole answers by transcription rather than by
// teaching. The `read` is the fragment beside it, and **both shapes whole is
// the length #211 cut, re-acquired** (ADR-0096, issue #212,
// internal/cli/instructions_test.go).
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
   ` + "`local`" + `, no Definition — for one answer before a Procedure exists, and with ` + "`--response`" + ` it makes
   no call at all and reads a ` + "`record:`" + ` against a response you fetched yourself.

Read the record back with ` + "`runs`" + `, ` + "`show`" + `, ` + "`records`" + ` and ` + "`changes`" + `. ` + "`changes`" + ` answers *what moved*,
field by field, against the last Run. **The record is a branch in this repository** — ` + "`hyper-store`" + `,
append-only, never checked out — so the working tree shows nothing of it, and it travels with a clone like
any other branch. ` + "`runs`" + ` and ` + "`records`" + ` say so on every answer; ` + "`git log hyper-store`" + ` reads it directly.

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
declare a ` + "`bound:`" + `, the maximum Records it may affect; on a ` + "`destroy`" + ` it is **mandatory**, and on an
**opaque** one — a ` + "`destroy`" + ` whose request is ` + "`shell:`" + `, which every ` + "`destroy`" + ` on the built-in ` + "`shell`" + `
Provider is — it is **refused** instead, a count of the commands it ran saying nothing about what any
of them did. **The credential is never in an artefact** — the Target names the environment variable,
` + "`hyper`" + ` puts it in the header the Manifest's ` + "`auth:`" + ` names, and no rendering prints it.

## Two request shapes

Single host — ` + "`host: \"{from-target}\"`" + ` resolves to the one host the bound Target grants, and ` + "`auth:`" + `
names the header its credential goes in. This is the shape for anything that *creates* or *ends*
something.

` + "`providers/preview-dns.yaml`" + `

` + "```" + `yaml
kind: provider
provider: preview-dns
schema-version: 1
class: cloudflare
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  create_record:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http:
      method: POST
      host: "{from-target}"
      path: /zones/{zone}/records
      body: {type: A, name: "{name}", content: "{address}"}
    input:
      type: object
      properties: {zone: {type: string}, name: {type: string}, address: {type: string}}
    record:
      identity: "{name}"
      fields: {id: $.body.result.id, created_on: $.body.result.created_on}
  delete_record:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http:
      method: DELETE
      host: "{from-target}"
      path: /zones/{zone}/records/{record_id}
    input:
      type: object
      properties: {zone: {type: string}, record_id: {type: string}}
` + "```" + `

Three rules there that nothing else will teach you. **A ` + "`record:`" + ` is mandatory on a ` + "`read`" + ` and a
` + "`mutate`" + `, and forbidden on a ` + "`destroy`" + `** — a Tombstone lands under the Asset's own identity. **An
Operation declaring ` + "`skip-if-recorded`" + ` takes its ` + "`identity:`" + ` from a hole, not from a ` + "`$.`" + ` path**: the
test reads the head of the series the call would write under *before* deciding whether to call, so an
identity that exists only once the response is back is a test the Manifest cannot perform. Take it from
what you sent. And **` + "`repeatability:`" + ` is declared here or defaulted here**, never downstream — an
effectful Operation declaring none is run-once.

Many hosts — the same hole expands to every host the Target grants, and ` + "`host-input:`" + ` names which input
picks one per Step. This is the shape for anything that *checks*, and it is three keys of a ` + "`read`" + `
Operation sitting where the two above sit.

` + "```" + `yaml
http: {method: GET, host: "{from-target}", path: /, host-input: host}
input: {type: object, properties: {host: {type: string}}}
record: {identity: $.host, fields: {host: $.host, status: $.status}}
` + "```" + `

A ` + "`read`" + ` projects what came back, so its ` + "`identity:`" + ` is a path, and a host the Target does **not**
grant comes from an ` + "`enumerations:`" + ` block of the Operation's own instead.

**Nothing offline tells you whether a ` + "`record:`" + ` path resolves.** No artefact states what an API
returns, so ` + "`check`" + ` reaches the grammar and stops there. Fetch one response with whatever client you
have, write it out as the response object — ` + "`host`" + `, ` + "`status`" + `, ` + "`headers`" + `, ` + "`body`" + `,
` + "`tls`" + ` — and hand it back: ` + "`probe <provider> <operation> --response <path>`" + ` makes **no call**,
and answers the Records that projection would have written with the paths that resolved to nothing named
beneath them. It is the only way to see a ` + "`mutate`" + `'s projection at all, a Probe never invoking one.
The file is scratch and ` + "`hyper`" + ` neither writes it nor loads it — **put it in ` + "`.gitignore`" + `**, a
saved response being the one place a token can end up in the tree by accident. **Do not author a
throwaway Operation projecting ` + "`$.body`" + ` to look at a response**: that writes a whole body into the
append-only record above, and only ` + "`compact`" + ` takes it out again.

## The rest of the repository

The Target grants what the Manifest requires, and ` + "`hosts:`" + ` and an ` + "`http`" + ` Capability go together or not
at all.

` + "`targets/cloudflare-prod.yaml`" + `

` + "```" + `yaml
kind: target-declaration
target: cloudflare-prod
class: cloudflare
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.cloudflare.com]
auth:
  token: {env: CLOUDFLARE_API_TOKEN}
` + "```" + `

A Target's ` + "`kinds:`" + ` admits ` + "`destroy`" + ` and **a Definition's ` + "`kinds:`" + ` has no such member**: there a
` + "`destroy:`" + ` claim names the Operations it allows one by one, granularity following severity. Reading
back what this Definition creates is a second Definition — ` + "`kinds: [read]`" + `, same Provider, same
Target — because one observes or it effects.

` + "`definitions/preview-dns.yaml`" + `

` + "```" + `yaml
kind: definition
definition: preview-dns
provider: preview-dns
kinds: [mutate]
destroy: [delete_record]
targets: [cloudflare-prod]
` + "```" + `

A Step's ` + "`over:`" + ` is the selector naming which Records it acts on — a different key from the ` + "`over:`" + `
inside a ` + "`record:`" + ` — and ` + "`{item: …}`" + ` addresses whatever it resolved to. A Step declaring none is invoked
once, which is the shape for creating something; a ` + "`destroy`" + ` Step always declares one.

` + "`procedures/refresh-preview-dns.yaml`" + `

` + "```" + `yaml
kind: procedure
procedure: refresh-preview-dns
targets: [cloudflare-prod]
steps:
  - id: publish
    definition: preview-dns
    operation: create_record
    target: cloudflare-prod
    args: {zone: example.com, name: preview-42.example.com, address: 203.0.113.10}
  - id: retire
    definition: preview-dns
    operation: delete_record
    target: cloudflare-prod
    over:
      assets:
        - field: created_on
          older_than: 14d
    args: {zone: example.com, record_id: {item: $.id}}
    bound: 5
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

## A shared check halts; it does not hand a verdict back

**A Procedure invokes another** with a ` + "`procedure:`" + ` in place of the binding — ` + "`- {id: verify, procedure: verify-archive}`" + `,
` + "`id:`" + ` and ` + "`procedure:`" + ` and nothing else — and what it invokes runs as Steps of the one Run. **No ` + "`when:`" + `
and no reference reaches across that boundary**: what an invoked Procedure did is not a fact its caller
can condition on, and its ` + "`targets:`" + ` count against the caller's declared envelope.

So a check that has to stop the work does it from the inside, with a ` + "`require:`" + ` entry of its own:

` + "```" + `yaml
  - id: archive-sound
    definition: archive-audit      # kinds: [read]
    operation: read
    target: archive
    args: {command: [sh, -c, "sha256sum -c /srv/archive/SHA256SUMS"]}

  - id: sound
    require: {step: archive-sound, field: exit_code, equals: 0}
` + "```" + `

A ` + "`require:`" + ` is a ` + "`when:`" + `'s predicate read for the other answer: a ` + "`when:`" + ` that does not hold **skips**
the Step it is written on, and a ` + "`require:`" + ` that does not hold **halts the Run** — here and in whatever
invoked this Procedure, one Run having one outcome however deep the invocation goes. It takes an ` + "`id:`" + `
and a ` + "`require:`" + ` and no other key, and it **claims no Kind and binds no Target**. Never reach for an
effectful Step that exits non-zero in order to make a check able to fail: that puts ` + "`mutate`" + ` in the
authority table of the one artefact whose point is that it writes nothing.

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
