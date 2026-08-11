# An HTTP response is an object `hyper` builds, not the body it returned

A Manifest's projection reads from an Operation's response, and that response is an object `hyper`
assembles from the call it made: the host it reached, the status, the headers, the parsed body, and the
peer certificate's facts where the scheme was HTTPS. A path roots at that object, so a body field is
`$.body.result.id` rather than `$.result.id`.

We chose this because the body is not the answer, and reading it as though it were fails on the
workload this tool opens with. A site that is down answers `503` with no body at all: under the
unaided reading there is nothing for a path to resolve against, so *is this site up* is a question
`hyper` can ask and never record the interesting answer to. The same reading makes the specification's
own canonical Provider unwritable — `uptime` projects a status code and a certificate's remaining days,
and neither has ever been in a body.

The certificate is what fixes the membership rather than leaving it to taste. Arithmetic is refused
everywhere in the format (ADR-0022), so no artefact can compute a remaining-days figure from an expiry
timestamp; either `hyper` supplies it or the most ordinary check anyone would write is unwritable. It
is not a second representation of `not_after`, which is the objection this design has sustained
elsewhere: it is derived from the certificate *and* the instant fixed at the Run's start (ADR-0034),
which is the one value no Manifest can name.

## Considered options

- **The parsed JSON body, and nothing else.** The reading an implementer reaches unaided, and the one
  every worked example in this specification currently reads as. Rejected above.
- **A second root marker for the envelope** — `$status`, or a sibling grammar beside `$`. This is a
  fourth production in a grammar that has three, and the position-decides-the-root rule already carries
  every other case of one path meaning two things.
- **The raw response bytes, addressable.** Rejected on the ground ADR-0017 already settled for
  rendering: a dump reinstates the catch-all bucket the record's shape exists to close, here on the
  surface that decides what a `destroy` reaches rather than on a screen.
- **A latency or duration member.** Rejected because a Record versions only on change, so a timing
  field would mint a version on every Run of every Operation that projected it, and the record would
  fill with evidence that `hyper` ran rather than that anything moved. A duration is computed inside a
  Journal entry, which is where this one already lives.

## Consequences

- **The object carries the host the call reached**, which is the one member that is a fact about the
  request rather than the answer. Without it an Operation whose answer carries no identity of its own —
  a check, the ordinary case for a `read` against several granted hosts — has nowhere to project a
  Record identity from, and one series would be written where there should be one per host.
- **`body` is absent rather than an error where the response is not JSON.** Otherwise no Manifest could
  check a website, an HTML page being the ordinary answer from the hosts an uptime check is pointed at.
  What follows is a limit rather than a hole: an API that answers in XML can be called and cannot be
  projected.
- **Every path in every Manifest gains a `body.` segment**, including the ones a polling Pattern's
  `until:` writes, which root at the same object.
- **A header can carry a credential.** `Set-Cookie` is the ordinary case, and what handles it is the
  machinery that already exists — a Manifest declares which projected fields are secret and the Record
  carries the marker — rather than a rule about headers specifically.
- **Whether a status is an error is a separate question.** This decision makes a `503` projectable; it
  says nothing about whether the Step that saw one continues, which is what an Operation's own
  declaration has to answer.
