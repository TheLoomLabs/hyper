package run

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// What an effectful call does with an answer (§6, §7, §12, ADR-0010, ADR-0050,
// ADR-0062, issue #148).
//
// **This is the whole of what a Kind changes about a call.** An effectful Step
// resolves, expands, calls and projects exactly as a `read` does — step.go is
// one path — and the difference is here: a `read` records whatever came back,
// and an effectful Step decides whether what came back means its effect
// happened.
//
// **It completes on `2xx` and halts on everything else, `3xx` included, a
// `destroy` completing on `404` besides.** A call the server did not accept did
// not do what the Step said, and `hyper` does not read the shape of an error
// body to decide whether its own effect happened; the redirect halts because
// `hyper` follows none, a redirect target being reach arriving from data
// (ADR-0029, capability.client). The `404` completes a `destroy` because a
// `destroy` told there is nothing there **has reached the state it exists to
// reach**, and because the alternative halts that Step identically on every
// re-run, leaving an Asset that can never be Tombstoned and the Steps after it
// *never reached* for good (§6, issue #150). **A Step halted
// by a status carries no `error_code`** — nothing declined, and a failure has
// none — and its Disposition is *ran* whether the status was `400` or `500`.
// The residual doubt about whether a `500` left something behind is real and is
// **not** what *attempted, outcome unknown* carries: that value means no answer
// came back at all.
//
// **The two `attempted` Dispositions are distinct on one axis, and it is
// whether anything is in doubt.** A deadline reached is ambiguous — the call
// went out and no answer came back — so the Step is *attempted, outcome
// unknown*. A request that provably never left is *attempted, world untouched*,
// and it is the safest state in the tool rather than a failure with an empty
// set: nothing was concluded about anything by construction, which is why it
// carries no identity set and renders the dash (§8, ADR-0062).
//
// **`answered` is written here and on no `read`.** Its presence is the fact
// that something other than the ordinary answer decided this Step; a `read`'s
// status is the answer and belongs in the Record wherever its Manifest
// projected it, and a Journal copy would add only a claim that `hyper` thought
// a `503` was untoward, which on a `read` it does not (§7, ADR-0010). It covers
// the three cases §6 makes of an answer that was not the ordinary one — the
// halt, the `404` that completes a `destroy`, and the request that never left —
// and no others: a deadline carries none, there being no answer to name.
//
// **On a `destroy` it is the whole of what tells a Tombstone written on `404`
// from one written on `204`.** The Record says the thing is gone and nothing
// there says how `hyper` learned it, which is the line ADR-0010 draws: what
// `hyper` is accountable for is that the thing is gone, and recording *already
// gone* as a fact about the Asset would be the reconciliation it declined to
// build (§7).
//
// **Retry is unaffected in both directions.** No status is ever retried — a
// status is an answer and never an error (ADR-0050) — and the three no-answer
// cases are the class ADR-0018 retries because the request provably never left.
// An exhausted retry leaves the response object for this to read, which is why
// the judgement is made where the projection is and not inside the call
// (capability.NeverSent, pattern.go).

// effectFault is a Run halted by what an effectful call answered: the wording a
// surface renders, the Disposition the Step takes, and the `answered` its file
// carries.
//
// It is a type rather than a plain error for readingFault's reason one file
// over — two things downstream turn on *what kind of fault this was* and
// neither can be read off a message. Here they are the Step's own Disposition
// and the `answered` block, both of them §7's and neither of them derivable
// from the text a reader sees.
type effectFault struct {
	// disposition is what became of the Step, and it is one of the three
	// §12 values an effectful halt can reach: *ran*, *attempted, outcome
	// unknown*, or *attempted, world untouched*.
	disposition store.Disposition
	// answered is what the call gave back where it did not give the
	// ordinary answer, and nil where there is nothing to name — the
	// deadline, which reached no answer at all.
	answered store.Answered
	message  string
}

func (f effectFault) Error() string { return f.message }

// answeredOtherwise is a call that came back and did not complete the Step: the
// status outside `2xx`, and the Disposition *ran* — the call went out, the
// answer came back, and what stopped the Step is what that answer said.
func answeredOtherwise(answered store.Answered, message string, args ...any) error {
	return effectFault{disposition: store.DispositionRan, answered: answered, message: fmt.Sprintf(message, args...)}
}

// neverLeft is a request that provably never left: no response arrived at all,
// so the Step is *attempted, world untouched* and its `answered` names the host
// it reached for with no status beside it (§7).
func neverLeft(answered store.Answered, message string, args ...any) error {
	return effectFault{disposition: store.DispositionAttemptedWorldUntouched, answered: answered, message: fmt.Sprintf(message, args...)}
}

// outcomeUnknown is a deadline reached on an effectful call: the call went out,
// nothing came back, and neither *it happened* nor *it did not* is a thing
// `hyper` may write down. It carries no `answered` — there is no answer to
// name, and the key exists to say what one was.
func outcomeUnknown(message string, args ...any) error {
	return effectFault{disposition: store.DispositionAttemptedOutcomeUnknown, message: fmt.Sprintf(message, args...)}
}

// dispositionAfter is the Disposition a Step reaches, given the fault that
// halted it and nil where nothing did.
//
// **Every other way a Step ends its Expansion is *ran***, a `read`'s deadline
// and a projection that did not resolve included: the calls went out and the
// answers that came back were recorded, and what the set holds is the members
// it concluded about (§6, §7, step.go).
func dispositionAfter(fault error) store.Disposition {
	var effect effectFault
	if errors.As(fault, &effect) {
		return effect.disposition
	}
	return store.DispositionRan
}

// whatAnswered is what the Step file holds under `answered`: what the fault the
// Run carries gave back, and — where nothing halted — what the **first member
// in Expansion order** that completed on something other than the ordinary
// answer was told.
//
// The two arms are one key because §7 makes them one sentence: its presence is
// the fact that something other than the ordinary answer decided this Step, and
// which of §6's three cases it was is read from the Disposition beside it.
//
// **A Step that halted answers with the halt**, read off the fault the Run
// carries rather than off whichever member happened to answer one — so the file
// names what ended the Step, whatever an earlier member of its Expansion was
// told. That is failedPath's own sentence one file over (§6, §7, reading.go).
//
// It answers with the halt **even where the halt names none**, which is the
// deadline: no answer came back, and a `404` an earlier member was told would
// read against the *attempted, outcome unknown* beside it as an answer this
// Step ended on. §7's key says which of §6's three cases decided the Step, and
// the Disposition is what tells them apart — so the two must not disagree.
func whatAnswered(fault error, completed store.Answered) store.Answered {
	if fault == nil {
		return completed
	}
	var effect effectFault
	if errors.As(fault, &effect) {
		return effect.answered
	}
	return nil
}

// effectful says this Step touches the world: its Operation declares a Kind
// other than `read`.
//
// The Kind is the Operation's and never the Step's, a Kind being declared per
// Operation in a Manifest and never inferred (ADR-0025) — which is the reading
// LockMode already takes over the same fact one file over (lock.go).
func (b binding) effectful() bool {
	return store.Kind(b.operation.Kind) != store.KindRead
}

// recordType is what a version this Step writes is a version of: an Observation
// under `read`, an Asset under an effectful Kind (§7).
//
// **Everything else about the write is the `read` path's**, unchanged — the
// identity restated unencoded, the projected content under `fields`, the whole
// of Provenance, and a version minted only where the bytes moved. A `mutate`
// returning what the head already holds mints nothing and its Record is still
// in the identity set, which is the case the whole mechanism exists for
// (ADR-0030).
//
// A `destroy` answers the same Asset: a Tombstone's `record_type` is `asset`
// because `hyper`'s effect reached the thing, and the marker beside it is what
// says the effect was the end of it (tombstones below).
func (b binding) recordType() store.RecordType {
	if b.effectful() {
		return store.RecordAsset
	}
	return store.RecordObservation
}

// tombstones says a version this Step writes is the destruction: this Step's
// Operation declares `destroy`, and every version it writes is a Tombstone
// (§7, ADR-0037).
//
// There is no second condition. A `destroy` writes on confirmed destruction
// only — what does not confirm writes nothing at all, so there is no
// half-Tombstone for a state to be read off — and confirmation is judged one
// file's function away (judged below).
func (b binding) tombstones() bool {
	return store.Kind(b.operation.Kind) == store.KindDestroy
}

// destroyed is what one member of a `destroy`'s Expansion concluded: the Asset's
// **own** identity, and no projected content at all.
//
// The series is the one the Expansion acted on and never a projection of the
// destroying Operation's response, which need not carry one: a `destroy`
// projects nothing and declares no identity (§3, §7, ADR-0037). So the name is
// the member's — the Record `name` an `assets:` selector resolved, and the
// literal itself where the selector is a `values:` list — and there is nothing
// to read out of what came back.
//
// The `fields` a Tombstone carries are not here either. They are the previous
// Head's, copied forward for the Asset's last known state, and they are read at
// the write rather than at the call: what a Tombstone says the Asset was is
// what the Store held about it and not what the destroying call answered, which
// is the one place in the Store `operation` and `fields` describe different
// calls (§7, step.go).
func destroyed(resolving member) []conclusion {
	return []conclusion{{name: resolving.Name}}
}

// dispatched is how many members of this Step's Expansion may be in flight at
// once: the Operation's Manifest-declared `concurrency:` limit on a `read`, and
// **one** on an effectful Step.
//
// **Concurrency is a function of Kind and is fixed by `hyper`** (§6, ADR-0045,
// drain.go). There is no authored knob, no flag and no environment override,
// and an effectful Step does not consult the limit at all: a number a Provider
// author measured for reads has no business widening how much destruction is in
// flight, and serial is what makes *three of five, then halt* a determinate
// fact a reviewer can read rather than a race.
//
// It is not a second code path. The dispatch's own rule is that a limit below
// one runs the Expansion one member at a time, so *serial* and *limit one* are
// one mechanism (drain.go).
func (b binding) dispatched() int {
	if b.effectful() {
		return 1
	}
	return b.detail.ConcurrencyLimit
}

// haltedByDeadline is the halt an Operation's `deadline:` is, at the Disposition
// the Step's Kind gives it. It is named for what it answers and never for the
// key it is about — the declared bound itself is `bound.detail.Deadline`, one
// field away, and an accessor's name on a function that answers a fault is
// where the day comes that a caller reaches for the wrong one.
//
// **A deadline reached on a `read` fails the Step and it is *ran***: the call
// went out under a bound an artefact declared, and nothing is in doubt that a
// Disposition could carry. On an effectful Step the same silence is the
// ambiguity *attempted, outcome unknown* exists for, and the wording does not
// change with it — what the message says is that the deadline was reached and
// no answer came back, which is the whole of the ambiguity (§6, §12).
func (b binding) haltedByDeadline(message string, args ...any) error {
	if !b.effectful() {
		return fmt.Errorf(message, args...)
	}
	return outcomeUnknown(message, args...)
}

// judged is what this Step's Kind makes of one response, and nil where the
// response is the ordinary answer — which on a `read` is every response there
// is (§6, ADR-0050).
//
// It is read off the response object and never off a socket or an error, which
// is what puts it here rather than inside the call: a retry Pattern follows a
// failure that provably preceded the request, and what an exhausted retry
// leaves is the object with the host and no status — the same object a single
// refused connection leaves, judged the same way (ADR-0018, pattern.go).
//
// **A `destroy`'s `404` completes it**, and what it answers instead of a fault
// is the `answered` the Step file carries: the call did not give the ordinary
// answer, and the Step went on all the same (§7).
//
// **The `shell` half is not here.** An effectful `shell` Operation completes on
// `0` alone and its `answered` carries the command and the exit code, there
// being no `404` for a command to answer with and no exit code meaning *already
// absent* in any vocabulary `hyper` knows. That is the Kind semantics on top of
// a Capability that needs nothing, and it is issue #156's — so an effectful
// `shell` Step still records what came back, as it did before this ticket.
func (b binding) judged(authored sequenced, response capability.Object) (store.Answered, error) {
	if !b.effectful() || b.operation.IsShell {
		return nil, nil
	}

	host, _ := memberOf[string](response, capability.MemberHost)
	status, arrived := memberOf[int](response, capability.MemberStatus)
	if !arrived {
		return nil, neverLeft(store.HTTPAnswer{Host: host},
			"step %s: no response arrived from %s, so the request never left and the world is untouched",
			named(authored), host)
	}
	answer := store.HTTPAnswer{Host: host, Status: store.Arrived(status)}
	if status/100 == 2 {
		return nil, nil
	}
	if status == http.StatusNotFound && b.tombstones() {
		return answer, nil
	}
	return nil, answeredOtherwise(answer,
		"step %s: %s answered %d, and a %s Step completes on %s — hyper follows no redirect and does not read the shape of an error body to decide whether its own effect happened",
		named(authored), host, status, b.operation.Kind, b.completesOn())
}

// completesOn is what this Step's Kind accepts, said the way a reader of the
// halt needs it: the threshold itself, so the sentence a halted `destroy`
// renders is true of a `destroy` (§6).
//
// It is here rather than written into the message because the message is one
// sentence for both effectful Kinds and the threshold is the only word in it
// that moves. A message naming the Kind and then stating the other Kind's
// threshold is the one thing a reader cannot recover from: they would read that
// their `404` should have halted too.
func (b binding) completesOn() string {
	if b.tombstones() {
		return "2xx and on 404 besides"
	}
	return "2xx alone"
}

// memberOf reads one member of a response object at the type §12 states it at,
// and answers whether it was there to read: `host` is a string and `status` an
// integer, both written by internal/capability and by nothing else.
//
// **The absence is the answer**, and on `status` it is the whole of *no response
// arrived*: a status of `0` read out of a member the object does not carry is
// exactly the reading store.Answer exists to prevent (§7, ADR-0050). A member
// carrying something other than its stated type answers the same absence, a
// value hyper cannot read being one it does not have.
//
// It is here rather than beside capability.Object.Lookup because what it adds
// is a **type**, and which type each of these two members has is §12's fact
// about the two members this file reads — not a widening of the response
// object's own interface, which answers *what does this member hold* and is
// read by a projection that resolves paths rather than names (ADR-0040).
func memberOf[T any](response capability.Object, name string) (T, bool) {
	var absent T
	value, held := response.Lookup(name)
	if !held {
		return absent, false
	}
	typed, stated := value.(T)
	if !stated {
		return absent, false
	}
	return typed, true
}
