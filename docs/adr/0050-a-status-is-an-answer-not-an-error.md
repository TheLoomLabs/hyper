# A status is an answer, not an error

A response that arrived is a result. **A `read` never halts on what came back; an effectful Operation
completes on `2xx`, a `destroy` on `404` besides, and halts on everything else.** No artefact declares
which statuses are acceptable, and where no response arrived at all the response object is the host and
nothing else.

The reading a competent implementer reaches unaided is the reflex of every HTTP client ever written:
below `400` is fine and `400` and above is an error. It is also what §6 promised before this decision —
*whether an unreachable host or a `503` is a result or an error is the Provider's declaration* — and
both readings fail on the workload §0 opens with. The `uptime` Provider projects `$.status` and §8
compares `status: 200 → 503`: under the fixed rule the specification's own canonical Provider cannot
record the answer it exists to record, and under the declared one it would have to enumerate every
status a website might ever answer with, which is not a finite list.

What makes the declaration unnecessary rather than merely awkward is that the guard it would install
already exists one layer down. A `read` whose answer is unreadable halts on its **projection**:
`list_records` against a `401` has no `$.body.result`, and §6 already halts a `series` Operation whose
collection path does not resolve, because a collection that was empty and a path that was wrong are not
the same fact. The status was never what decided a `read`; the projection was, and it decides for a
stated reason against the response in hand.

On the effectful Kinds the projection is not enough and `hyper` fixes the rule instead. A `destroy`
projects nothing at all (ADR-0037), so there is nothing there to fail; and on a `mutate` the guard is
incidental, since whether an identity path resolves against an error body is a property of that API's
error body — data no reviewer sees while reviewing. Where `hyper` is accountable for an effect it does
not read the answer's shape to decide whether the effect happened. It reads the status, and the status
it accepts is `2xx`. `3xx` is on the halting side because ADR-0029 already put it there: a redirect
target is reach arriving from data, so `hyper` follows none, and an effectful call answered with one did
not do what the Step said.

`404` completes a `destroy` because the alternative is not a limit but a trap. An Asset whose resource
is already gone would halt its Step, and halt it identically on every re-run: the Asset could never be
Tombstoned, every later Step would be *never reached* forever, and the only exit would be deleting the
Definition, which orphans every Asset it owns (ADR-0012). A `destroy` told there is nothing there has
reached the state it exists to reach, and it writes an ordinary Tombstone. `CONTEXT.md` has always
allowed one where a destruction reached something `hyper` had no record of, so a Tombstone has never
claimed that `hyper` caused the destruction — only that its effect reached and the thing is gone.

## Considered options

- **Fix it at `< 400` on every Kind.** The unaided reading, rejected above: it makes the canonical
  `uptime` Provider unwritable, and it makes an API answering `404` for *absent* — the ordinary REST
  idiom — undescribable, the Operation that wanted to record an absence halting instead.
- **The Manifest declares which statuses succeed**, which is what §6 promised. Rejected on three counts.
  The enumeration is unbounded on the `read` side. A status list is authority — *what counts as
  acceptable* — living in the artefact a reviewer reads least, where the same decision written as a
  `when:` on a recorded status sits on the line the gutter annotates. And it is a further per-Operation
  declaration and a new closed vocabulary in §12, bought for a guard the projection already supplies.
- **A status-class vocabulary** — `success`, `client-error`, `server-error` — declared per Operation.
  Rejected as too coarse for the case that motivates it: `404` meaning *absent* and `401` meaning *your
  token is wrong* are one class, and a declaration would have to swallow both to reach either.
- **Split the halt's Disposition by class**, `4xx` as *ran* and `5xx` as *attempted, outcome unknown*.
  Rejected: it is the closed vocabulary above arriving through the Journal instead of the Manifest, and
  it changes nothing about what re-runs, §6 refusing a run-once Step on either value. §12's own
  definitions decide it without a partition — *attempted, outcome unknown* means no answer came back,
  and a `500` is an answer.
- **An `error` member on the response object**, carrying what went wrong where no response arrived.
  Rejected on the ground ADR-0017 settled for rendering: it reinstates the catch-all bucket the record's
  shape exists to close, here on the object every projection reads from.

## Consequences

- **Where no response arrived the response object is `host` and nothing else** — `status`, `headers`,
  `body` and `tls` all absent, on the rule `body` already carried (ADR-0040). A `read` records a refused
  connection as an Observation whose `status` has gone quiet, which renders as a change like any other,
  and an effectful Operation halts for free, no status being not `2xx`. `uptime` records *down* whether
  the host answered `503` or did not answer at all, and the only reason it can is that `host` is a fact
  about the call rather than about the answer.
- **A `read` can still halt, and only through its projection.** That leaves §6's drain rule scoped to
  exactly one case, so it is stated there rather than left to be composed: the Expansion drains and the
  Run halts after it. Halting at the *first* failure would make which Observations were recorded depend
  on a concurrent Expansion's completion order, which §6 forbids deriving from.
- **Every call is judged, a Pattern's included.** There is no final call a rule could privilege without
  inventing one, and a Pattern may not change what an Operation does. Retry is untouched: ADR-0018
  confines it to failures that provably preceded the request, so no status is ever retried, and an
  exhausted retry on a `read` leaves the object above for the projection to read.
- **A status halt carries no `error_code`, and its Disposition is *ran*.** Nothing declined, so there is
  no check to name (§12), and a response arrived, which is what *ran* means — `400` and `500` alike, the
  residual doubt about a `500` being real and not what *attempted, outcome unknown* carries. Where **no
  response arrived** the Disposition is decided by neither this decision nor anything else in the
  corpus: what settles it is whether the request went out, which is ADR-0018's line rather than a fact
  about a status, and none of the six values fits a call that provably never left. That is a hole this
  decision makes reachable in one more way rather than one it opens, and it is ticketed rather than
  answered here.
- **One key on the Step file, `answered`, and no closed set grows.** An effectful Step whose call
  answered anything but `2xx` carries the host it reached and the status it got — the halting case and
  the `destroy` `404` alike, the status itself absent where nothing came back. It is on the shape
  `projection_failed_path` already has, and it is effectful-only because it exists to record that a
  non-`2xx` answer changed what `hyper` did, which is the same ground the Pattern account stands on
  (ADR-0018). A `read`'s status is the answer and belongs in the Record, where its Manifest projected
  it. §12 gains no `error_code`, no response-object member and no Disposition; two members gain an
  absence rule. §4 gains nothing, nothing being declared for it to check.
- **Nothing distinguishes a Tombstone written on `404` from one written on `204`.** Recording *already
  gone* as a fact about the Asset would be the reconciliation signal `hyper` declined to build
  (ADR-0010) and would reopen the Tombstone's four facts (§7). The distinction is not lost, it is
  relocated: `answered` holds the status on the Step file, so the Journal says how `hyper` learned the
  thing was gone and the Record says only that it is.
- **A seventeenth victim at the wall.** An effectful Operation against an API that answers anything but
  `2xx` is unwritable — a create answering `303 See Other`, or one answering a transient `5xx` mid-poll.
- **`probe` exits `0` whatever came back.** A Probe is a `read`, so nothing halts it; a nonzero exit
  would be `hyper` deciding a `503` is bad news, which is the judgement §9 already refuses when it
  renders a Cadence's staleness and never says *overdue*. The exit code says whether the command did
  what it was asked and the rendering says what came back.
