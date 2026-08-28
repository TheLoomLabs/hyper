# The shape carried whole is the effectful one, and the worked example is not the acceptance task

**The orientation's two request shapes swap places.** The Manifest it carries whole is now effectful —
a `mutate` beside a `destroy`, an `auth:` scheme, a `record:` whose `identity:` is a hole, a Step with
a selector and a Bound — and the multi-host `read` that was the whole example becomes the fragment
beside it, three keys of an Operation.

This is a defect in what ADR-0095 shipped rather than a new idea. Its decision — one text, two
channels, `project` writing the file — stands untouched. What moves is which of the two shapes is
written out.

## The evidence: the criterion was met by transcription

A Claude Code session, 2026-08-28, in a fresh repository at `bea4560` — quickstart shape, `hyper.yaml`
pinned by hand, **no `providers/`**, MCP wired at local scope, `AGENTS.md` written by `project` and
unedited. The instruction was #209's and #211's: an extension that gets the HTTP status of a list of
websites and says whether they are online.

**ADR-0095's criterion is met, and the second channel did its job.** Seventeen calls, and not one of
them is binary archaeology, against twenty-eight calls with six `strings` invocations in the transcript
#211 was opened on. It read `AGENTS.md` **before its first tool call**, stopped at the diff without
calling `run`, and closed on `baseline_absent` by naming `store init` as the human's and not running
it — the surface-neutral wording of ADR-0095 working under the exact conditions it was written for.

**And the Manifest it wrote is the orientation's, verbatim.** Twenty-two of twenty-two lines identical;
the Target, the Definition and the Procedure identical but the host list; the filenames, the Provider
name, the Operation name and the Step-id convention all the example's. The only substantive edits were
dropping a field it was not asked for and reflowing one inline mapping.

So the run **cannot distinguish *the orientation is sufficient* from *the orientation contains the
answer***. The canonical regression task and the worked example had become the same task, which is
ADR-0095's own doing: before it, the primary example was a Proxmox Manifest and the multi-host `read`
was the *second* one, added under #209 because an agent asked for exactly this had only a single-host
`mutate` to work from.

## The fragment was carrying the rules, and had never been run

The shape demoted to four lines is the one whose rules an agent cannot infer. Absent from it, and so
absent from the whole text: `kind: mutate` beside `repeatability: skip-if-recorded`, an `input:` schema
for a request with a body, a `bound:` on a Step, a `destroy` Operation at all, and — the one that
matters most — **a `skip-if-recorded` Operation's `identity:` is a hole and not a `$.` path**, because
the test reads the head of the series the call would write under before deciding whether to call (§3).
The old Proxmox Manifest showed that. Nothing did afterwards.

§4 catches one half of getting that wrong and cannot catch the other: an `identity:` that is a response
path is `manifest-inconsistent`, while a hole naming the wrong *input* checks clean — and then checks
and records under a key nothing else will find. An agent authoring an effectful Operation had to compose the
fragment's request block with the read Manifest's scaffolding and the prose about `bound:`. That may
work; it had never been run.

## What was considered

**Fixing only the domain collision** — keeping the `read` whole and moving it to a domain no acceptance
task names. It is the smaller change and it was the right thing to try first: if an agent composes a
correct `mutate` from the fragment as it stands, the fragment is sufficient. That run has not happened
(it needs a fresh session in a fresh repository, which is what `/home/idabic/dev/hyper-effectful` is
set up for), and the swap is the change that does not depend on its outcome — a `read` composed from a
fragment is the shape #209 already proved teachable from three keys, where the effectful rules above
have never been taught by anything but a whole Manifest.

**Carrying both shapes whole** is refused. That is the 13,191 characters #211 cut, re-acquired, and the
text is paid for on every session in every harness — once as a handshake field whether or not the model
reads it, once as a file the harness reads up front.

## What it costs

The text goes from 9,396 characters to 11,398 — **+21%, and 14% below the 13,191 #211 started from**,
counted in characters as ADR-0095 counts them. It is not the wash issue #212 hoped for, and the reason
is not prose. The effectful shape has more
that must be written down: two Operations rather than one, an `auth:` scheme, a `destroy:` claim, a
selector, a Bound, and five sentences of rules that exist only on that side of the line. #211's
reduction stands in the sense that matters — no fact was re-acquired to get here, and the second shape
is still a fragment.

## Consequences

- **§9 fixes which shape is whole**, where it previously fixed only that both are carried. An
  orientation that carried the acceptance task as its example would satisfy that sentence and teach
  nothing, which is what happened.
- **Two cases hold the swap** (`internal/cli/instructions_test.go`): the example still checks clean,
  and the Manifest carried whole declares a `mutate`, a `destroy`, `skip-if-recorded` and an
  `identity:` that is a hole — while no artefact in it carries `host-input:`, which is what keeps the
  `read` a fragment and the second Manifest from growing back.
- **Acceptance is two transcripts, and both are outstanding.** Neither may name the example's domain.
  The multi-host `read` must still land without `strings`, now composed from a fragment rather than
  copied — that is the risk this decision takes on. An effectful Operation must create and delete over
  HTTP with header auth, with a correct `record.identity` and a `bound:` on the `destroy`.
- **ADR-0095's account of the reduction is amended** where it names which shape became the fragment.
  Nothing else in it moves: the channel, the create-if-absent write, the wording for two readers and
  the never-overwrite rule are all untouched.
- **The example's domain changes with it**, from website uptime to DNS records. That is the point
  rather than a side effect: the domain an acceptance task names must not be the domain the example
  is written in.
