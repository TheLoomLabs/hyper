package run

import (
	"errors"
	"fmt"

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
// **It completes on `2xx` and halts on everything else, `3xx` included.** A
// call the server did not accept did not do what the Step said, and `hyper`
// does not read the shape of an error body to decide whether its own effect
// happened; the redirect halts because `hyper` follows none, a redirect target
// being reach arriving from data (ADR-0029, capability.client). **A Step halted
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
// the halt and the request that never left, and no others: a deadline carries
// none, there being no answer to name.
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

// answeredBy is what the Step file holds under `answered`, and nil for every
// way a Step ends that did not answer otherwise.
//
// It is read off **the fault the Run carries** and not off whichever member
// happened to answer one, so the file names what halted the Run — which is
// failedPath's own sentence one file over (§6, §7, reading.go).
func answeredBy(fault error) store.Answered {
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
// A `destroy` reaches here and answers the same Asset, which is true of the
// version it writes and not yet of the marker on it: a Tombstone is an ordinary
// version of the Asset's own series carrying `tombstone: true` and the previous
// Head's fields copied forward, and that is issue #150's.
func (b binding) recordType() store.RecordType {
	if b.effectful() {
		return store.RecordAsset
	}
	return store.RecordObservation
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
// **A `destroy`'s `404` is not here.** A `destroy` told there is nothing there
// has reached the state it exists to reach, so it completes on `404` besides —
// and the alternative halts that Step identically on every re-run, leaving an
// Asset that can never be Tombstoned. That, and the Tombstone the completion
// writes, are issue #150's.
//
// **The `shell` half is not here.** An effectful `shell` Operation completes on
// `0` alone and its `answered` carries the command and the exit code, there
// being no `404` for a command to answer with and no exit code meaning *already
// absent* in any vocabulary `hyper` knows. That is the Kind semantics on top of
// a Capability that needs nothing, and it is issue #156's — so an effectful
// `shell` Step still records what came back, as it did before this ticket.
func (b binding) judged(authored sequenced, response capability.Object) error {
	if !b.effectful() || b.operation.IsShell {
		return nil
	}

	host, _ := memberOf[string](response, capability.MemberHost)
	status, arrived := memberOf[int](response, capability.MemberStatus)
	if !arrived {
		return neverLeft(store.HTTPAnswer{Host: host},
			"step %s: no response arrived from %s, so the request never left and the world is untouched",
			named(authored), host)
	}
	if status/100 == 2 {
		return nil
	}
	return answeredOtherwise(store.HTTPAnswer{Host: host, Status: store.Arrived(status)},
		"step %s: %s answered %d, and a %s Step completes on 2xx alone — hyper follows no redirect and does not read the shape of an error body to decide whether its own effect happened",
		named(authored), host, status, b.operation.Kind)
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
