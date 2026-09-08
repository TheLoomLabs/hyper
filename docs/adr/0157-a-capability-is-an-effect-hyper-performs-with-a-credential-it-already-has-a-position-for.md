# A Capability is an effect `hyper` performs with a credential it already has a position for

The Capability set has been closed at two since ADR-0004, and this is the sentence that keeps it
closed. A candidate is admitted where it is an effect `hyper` itself performs and every credential
it needs occupies a position `hyper` already owns; where either half fails it is refused, and it is
refused **by the sentence** rather than beside it.

`http` is admitted by both halves: the effect is a request `hyper` makes on a Manifest's behalf, and
the credential is a header a Provider author named and a Target declaration filled (§12, ADR-0031).
`shell` is admitted by the first half and meets the second vacuously: a credential is a property of
`hyper` reaching a host, a `shell` Operation reaches none, and `hyper` therefore holds nothing for
it to have a position for (§12, ADR-0039). What the *command* then reaches is the machine's business
and not the Capability's. That is not a hole either — the clause bites on a candidate needing a
credential `hyper` would hold, and all four below need one.

Four things are refused by the sentence, before any criterion is reached. **SSH** needs a private
key, and a private key is not a header — there is no position in any artefact where one could be
written, and inventing one is inventing an Auth scheme. **Placing bytes on a machine** fails the
same clause one step later: reaching the machine is the credential, and the local machine is
`shell`. It is refused by the sentence like the other three and is the one of the four kept on file
rather than closed, for the reason the section below gives. **A client certificate** is a property
of a connection rather than a position in a request, which is the ground ADR-0031 refused it on and
the ground it is refused on here. **A secret arriving in a request body** puts the credential in a
position a Step's arguments choose, where the whole point of a placement is that `hyper` knows the
position mechanically and suppresses by it (§3, ADR-0031).

We chose this because ADR-0004 closed the set and then said, of the process by which it grows,
*"What remains unowned is the process by which the closed set grows without becoming an open one by
attrition."* That sentence has stood since the set was closed, and §12 and §13 have both advertised
it as unowned ever since. Attrition is not a risk that arrives all at once: every candidate arrives
with a real author behind it, a described API it cannot reach, and a merit, and a set with no
admission sentence loses one member's worth of closure per plausible request. The remedy is not a
stricter disposition. It is a sentence a refusal can be *by*, which is the only device in this
project that has ever held a set closed — ADR-0031 fixed the Auth schemes at two by *a header and a
placement, never a protocol*, and that sentence does real work, refusing request signing, OAuth2 and
a client certificate without a judgement about how many schemes are enough. The Capability set gets
the same instrument rather than a better-argued inventory.

## How the sentence is applied

Five criteria, and they are conjunctive. A candidate passes all five or it is refused; there is no
score, and four of five is a refusal and not a near miss.

1. **Describable.** Its effects are stated in the artefact, so a reviewer reads what it will do
   before it does it. `http` passes — a method, a host, a path and a body, every one of them written
   down. `shell` fails, and is the named exception rather than the precedent: opacity is a property
   of the Capability (§12), and what makes the exception affordable is criterion 4, not a tolerance
   a second candidate inherits.
2. **Recordable.** What it reached has an identity `hyper` owns — a Target it bound, an argv it ran,
   an object it projected — so the Comparison has something to join a Record to (§8, ADR-0030).
   `shell` passes at the floor rather than failing here: an argv and an exit code are *that it ran*
   and not what it printed (ADR-0143), and the machine is the one a class-`local` Target names. What
   fails is a candidate reaching a machine named in its own arguments, which records an outcome with
   nothing `hyper` can attach it to.
3. **Not composable.** It cannot be reached by the primitives that exist. A candidate an author can
   already write — differently, or less pleasantly — is not a missing effect.
4. **No reach a third party gains.** Either it is safe to grant an Extension, whose Manifest anyone
   may author and `hyper` vouches for nothing about (ADR-0004), or it joins `shell` as reserved.
   Which of the two is **decided when the Capability is added and never later**: reserving one
   afterwards takes reach back from Manifests already written and is a change no admission sentence
   should be able to require.
5. **Credential-positioned.** Every credential it needs occupies a position `hyper` already owns, or
   the candidate is an Auth scheme change wearing a Capability's clothes. This is the criterion that
   keeps ADR-0031 from being reopened by accident, and the one all four candidates below fail or
   dodge.

**The converse holds and is the point.** A candidate failing 3 is refused *even where it passes the
other four* — even where it is describable, recordable, safe to grant and needs no credential at
all. That will read as pedantry the first time it bites, because a candidate in that state is one
whose argument is *we could do this more conveniently*, and convenience is exactly the mechanism
ADR-0004 named. A set that admits on convenience is open; it merely takes longer to notice.

## The first application

The candidates this decision was written against, each assessed against all five:

| candidate | describable | recordable | not composable | third-party reach | credential-positioned |
|---|---|---|---|---|---|
| `file` — place bytes on a machine | pass | pass | **fail** | would need reserving | **fail** |
| `ssh` / remote exec | **fail** | exit code only | **fail** | **fail** | **fail** |
| agent over HTTPS | pass | pass | **fail** | **fail** | pass |
| secret request-body input | pass | n/a | **fail** | — | **fail** |

`ssh` fails every criterion it can fail, which is what makes it the clean case: the effect is opaque
like `shell` without `shell`'s reservation argument, what it records is an exit code, the machine it
reaches is reachable by provisioning it configured, granting it to an Extension hands a third party
a shell on somebody else's host, and its key has no position. Its *recordable* cell reads **exit
code only** rather than a refusal for that reason: the outcome is `shell`'s, and the host it landed
on is a word in an argument rather than a Target `hyper` bound. An **agent reached over HTTPS** is
the subtle one. It passes credential-positioned outright, the credential being a header like any
other, and it is refused on 3 and 4 together: an HTTPS request to a host a Target grants is `http`,
so what would be added is not an effect but a name for a use of one. A **secret request-body input**
is in the table for completeness and is refused by the sentence above, before any criterion is
reached.

**`file` is kept on file rather than refused.** It is the only candidate that passes describable and
recordable cleanly — bytes at a path is a fact with an identity and a digest, which is more than
`shell` manages — and it fails *not composable* only because a machine provisioned already
configured has nothing left to place, which is a fact about how machines are made today rather than
about the effect. The day a machine must be **fixed rather than replaced**, `file` passes 3, and the
argument resumes with only credential-positioned still standing against it — a resumption that is
then an Auth scheme question, which is ADR-0031's and not this record's. Recording it as deferred is
not softness. It is so that the next person to propose it finds the reasoning at the point it
stopped rather than repeating the whole of it to arrive there.

## Considered options

- **Leave it unowned**, and decide each candidate on its merits when it arrives. The status quo, and
  the thing §12 and §13 have been honestly advertising. It is rejected because *on its merits* is
  the mechanism rather than the alternative to it: every candidate has merits, the ones with none
  never get proposed, and a decision procedure that reads only merits admits monotonically. ADR-0004
  named the failure; declining to own it a second time is answering the question with the sentence
  that raised it.
- **An inventory** — a written list of effects that will never be Capabilities. Rejected on the
  ground ADR-0039 already rejected an inventory for the built-in roster: a list is not closed
  against a candidate nobody thought to write down, and it inverts the burden, so an effect's
  absence from the refusals reads as a case for it.
- **The five criteria and no sentence.** Nearly this decision, and the tempting economy — the
  criteria do the work, so why the sentence? Rejected because criteria without a sentence are a
  scorecard. A candidate passing four opens a negotiation about the fifth, and a negotiation whose
  subject is one criterion at a time is attrition with a procedure. The sentence is what a refusal
  *is*; the criteria are how it is applied, and they are downstream of it.
- **A demand test** — admit an effect once enough authors hit the wall on it. Rejected: it makes the
  set's closure a function of how hard it is pushed on, which is attrition restated as policy. The
  `capability-reserved` deaths #227 records are evidence that the corpus is illegible about what
  exists, which is a documentation finding, and they are not evidence for admission.
- **A second general-purpose Capability**, granted narrowly — one effect that does whatever the
  machine can do, for the cases the four candidates above cover between them. Rejected: that
  Capability is `shell`, it exists, and it is reserved. Adding a second and granting it to
  Extensions is ADR-0004's WASM argument arriving through a door labelled differently — the residual
  power of a third party's declaration is the whole question, and the answer does not change with
  the label.

## Consequences

- **ADR-0004's last unowned consequence is discharged**, and the two places that advertised it stop
  saying so. §12's opening, which recorded the growth process as undecided, and the sentence closing
  §13's ceiling, which recorded that §12 recorded it, are both struck in the change that lands this
  record — so neither live spec line advertises a hole this record says is filled. The consequence
  in ADR-0004 that named it stands as written, under the amendment marker CONTRIBUTING requires,
  which is where the corpus keeps a passage a later decision answers.
- **ADR-0062's growth route is available to other sets in §12 and not to this one.** That record
  grew the Dispositions for the first time on the sentence *closed against authors adding to it,
  which is not the same as closed against the specification naming a state of the world it had not
  written down* — and the seventh Disposition was a state that had been occurring all along with no
  word for it. No effect `hyper` does not perform is already happening. So the one precedent for a
  §12 set growing cannot be borrowed here, and the distinction is worth stating rather than leaving
  for someone to reach for: a naming is free and a power is not.
- **ADR-0039's two growth stories meet inside criterion 4.** Its split — *the roster and the
  reserved Capabilities are two closed sets with two growth stories… Only the second is a decision
  about power* — is upstream of this record, and criterion 4 is where it lands inside a single
  admission: a Capability an Extension may declare widens what every Manifest anyone writes can do,
  where a reserved one widens only `hyper` and costs a name in the roster's namespace. ADR-0039's
  own criterion for shipping a built-in is untouched and now reads from a set with an admission
  test.
- **A refusal is legible without this document.** SSH, file placement, mTLS and a secret
  request-body input are each refused by one sentence a reader can hold, in the manner ADR-0031
  refuses OAuth2, and an author who reaches for one of them learns why in a line rather than in a
  survey of five criteria.
- **`file` is deferred rather than dead**, with a stated condition — a machine that must be fixed
  rather than replaced — and the condition is about the world rather than about anyone's willingness
  to reconsider.
- **Nothing moves.** The Capability set is still `http` and `shell`, no artefact gains a key, no
  closed set gains a member, `check` reads nothing it did not read before, and there is no code in
  this decision at all. What changed is that the next proposal is assessed rather than argued.
