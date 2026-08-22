package run

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// When a projection does not resolve, and when one resolves and collides (§6,
// §7, §8, ADR-0017, ADR-0072, issue #144).
//
// **A response can arrive and still not be readable, and this is the one way a
// `read` fails at all.** Nothing static decides it — no artefact states what an
// API returns — so it is decided where it can be: against the response in hand,
// after the call has gone out, and with no Refusal available. That is the whole
// of why the faults here are halts and the Expansion's two identity comparands
// one file over are Refusals (expand.go, ADR-0072).
//
// **Which path failed is what decides.** A path a recorded field is read from
// resolving to nothing is an absence and not a fault at all: the field is not
// written on that version, the bytes moved, and a surface reads the field going
// quiet as a change like any other (§6, §7, internal/projection). What halts is
// the path a Record's identity is read from and the path an Operation of
// `series` cardinality reads its Records from — without the first `hyper`
// cannot say which Record it is holding, and without the second it cannot tell
// a collection that was empty from a path that was wrong.
//
// **Neither carries an `error_code`.** Nothing declined: the call went out, the
// answer came back, and what failed is `hyper`'s reading of it — so the Step's
// Disposition is *ran* and what the file names instead is the path, under
// `projection_failed_path` (§7, §12). That path is held there and nowhere else,
// and **no surface shows the response it failed against** (ADR-0017).

// readingFault is a Run halted by `hyper`'s reading of an answer that arrived:
// the wording a surface renders, and — where a path is what failed — the path
// the Step file holds under `projection_failed_path`.
//
// It is a type rather than a plain error because two things downstream turn on
// *what kind of fault this was* and neither can be read off a message. The Step
// file carries the path (§7), and the drain writes the Records a faulted member
// **did** project: what a half-projected response puts in doubt is the claim
// that its Records are all of them, and that claim lives in the identity set
// rather than in any Record (§6, step.go).
type readingFault struct {
	// path is the path that failed to project, and "" where the fault is a
	// collision. A colliding identity **resolved** — what is wrong with it
	// is what it resolved to — so there is no failed path to name and §7
	// puts none on the file.
	path    string
	message string
}

func (f readingFault) Error() string { return f.message }

// unreadable is a path that did not resolve against what came back.
func unreadable(path, message string, args ...any) error {
	return readingFault{path: path, message: fmt.Sprintf(message, args...)}
}

// collided is an identity that resolved and is one another Record already
// holds. It names no path: see readingFault.path.
func collided(message string, args ...any) error {
	return readingFault{message: fmt.Sprintf(message, args...)}
}

// wroteWhatProjected says whether a member that faulted still writes the
// Records it did project.
//
// It is true of exactly the faults here, and the distinction is §6's: a
// response `hyper` read part of is a response that arrived, and what projected
// is written. A member that reached the Operation's deadline holds no such
// answer — the call went out and nothing came back — and the drain skips it as
// it always did (§6, drain.go).
func wroteWhatProjected(fault error) bool {
	var reading readingFault
	return errors.As(fault, &reading)
}

// failedPath is what the Step file holds under `projection_failed_path`: the
// path that failed to project on the fault the Run carries, and "" for every
// other way a Step halts.
//
// It is read off **the fault the Run carries** and not off whichever member
// happened to fail a projection, so the file names what halted the Run rather
// than a second fault the entry says nothing else about (§6, §7).
func failedPath(fault error) string {
	var reading readingFault
	if errors.As(fault, &reading) {
		return reading.path
	}
	return ""
}

// identityHolders is the two comparands an identity that resolved **from a
// response** is held against, walked in the order §6 fixes: whatever held the
// identity first keeps it, and each comparand supplies the order that decides
// which that was (§6, ADR-0072).
//
// The sibling comparand is Expansion order across an Expansion — which on a
// `read` is read off the drain rather than off a completion order — and the
// collection's own order across one `series` response. Both are one walk here,
// the drain reading the members back in Expansion order and each member's
// Records in the order the collection stated them (step.go, drain.go).
//
// The Store comparand supplies no order at all: the standing series was written
// by an earlier Run and nothing in the Store is removable. It is read **once**,
// before the first version of this Step goes down, so a member of this Run's
// own Expansion is the sibling comparand rather than a series that was already
// standing.
//
// **It is not expand.go's two comparands under another name**, and the shape is
// retyped rather than shared because the two answer different questions. The
// Expansion's pass runs over a resolved set and reports **every** decline it
// finds, a Refusal's array being the checks that declined together (§7,
// ADR-0061); this one decides one identity at a time as the drain reaches it and
// the **first** fault is what the Run carries, a halt being one fault and not a
// list. What they share is store.Folded, which is where the fold that decides a
// collision does live once (§7).
type identityHolders struct {
	// first is what projected each folded identity first, keyed by the fold
	// and never by the identity: two identities that must be distinct being
	// one under the fold is the collision, and `Foo` beside `foo` is its
	// whole content (§7, store.Folded).
	first map[store.Identity]projectedIdentityBy
	// standing is what the Store already held, keyed by the identity as it
	// resolved — Collisions' own key, and empty for the members that
	// collide with none.
	standing map[store.Identity]store.Identity
}

// projectedIdentityBy is one identity that resolved, and what projected it. The
// two travel together because a collision reports both: the name is what
// collides and the projector is what a reader looks at to find it.
type projectedIdentityBy struct {
	name string
	by   string
}

// heldBy reads the Store comparand once for the identities a Step's calls
// projected, and answers the pair the drain walks.
//
// It reads nothing where the Operation's `identity:` resolves before the call:
// the Expansion has already run both comparands over those identities and
// Refused, with nothing touched (§6, expand.go). That is projectedAfterTheCall's
// answer below, and it is the whole of what decides whether the branch is
// enumerated at all.
func (r run) heldBy(bound binding, answers []answer, authored sequenced) (identityHolders, error) {
	held := identityHolders{first: map[store.Identity]projectedIdentityBy{}}

	projected := projectedAfterTheCall(bound, answers, authored)
	if len(projected) == 0 {
		return held, nil
	}
	standing, err := r.request.Store.Collisions(projected)
	if err != nil {
		return identityHolders{}, err
	}
	held.standing = standing
	return held, nil
}

// take answers the collision where this identity is one another Record already
// holds, and records it as the holder where it is not.
//
// The sibling comparand is asked first, for the reason the Expansion's Refusal
// names it first: it is reproducible from this Run alone and points at
// something a reader can reach, where the Store comparand points at a series an
// earlier Run wrote (§6, expand.go).
func (h *identityHolders) take(id store.Identity, by string) error {
	folded := store.Folded(id)
	if earlier, taken := h.first[folded]; taken {
		return collided("%s resolved the identity %s, and %s already resolved %s — the two are one Record identity",
			by, id.Name, earlier.by, earlier.name)
	}
	if standing, holds := h.standing[id]; holds {
		return collided("%s resolved the identity %s, and the Store already holds %s under %s/%s — the two are one Record identity",
			by, id.Name, standing.Name, standing.Target, standing.Definition)
	}
	h.first[folded] = projectedIdentityBy{name: id.Name, by: by}
	return nil
}

// projectedBy is how a collision names what projected an identity: the
// Expansion member, and the Record's position in the collection where the
// Operation projects many Records out of one response.
//
// A Step carrying no selector has no member to name, and what it names instead
// is **the answer** — `what came back`, the phrase the projection faults above
// already use. It is not the Step: the message opens with the Step, and a
// second naming of it there would say nothing the reader is not already
// holding.
//
// It says nothing whatever about the response beyond the position. A collision
// is reported over two names and a place to look for them, and no surface shows
// the response a projection was read against (ADR-0017).
//
// The position counts from 1 and runs across the whole of one member's walk, so
// a paginated `series` Operation numbers its Records the way a reader counting
// them would rather than restarting at each page — the pages being one
// collection arriving in instalments (§3, pattern.go).
func projectedBy(member string, at int) string {
	named := member
	if named == "" {
		named = "what came back"
	}
	if at == 0 {
		return named
	}
	return fmt.Sprintf("record %d of %s", at, named)
}

// projectedAfterTheCall is the identities a Step's calls projected, in
// Expansion order, and **nothing at all** for the one shape the Expansion
// already held.
//
// That shape is an Operation of `one` cardinality whose `identity:` resolves
// before the call: one member, one Record, and both comparands already run over
// that name with nothing touched (§6, expand.go). Asking a second time would be
// one enumeration of the branch per Step for an answer already given.
//
// **An Operation of `series` cardinality reaches here whichever way its
// `identity:` resolves**, and that is not a hedge. The Expansion's pass holds
// one identity per **member** (ADR-0070), and a `series` Operation puts many
// Records under one member — so a `series` Operation whose `identity:` is a
// template hole projects every Record of one response under the one name that
// hole filled to, which is several versions of one series and a collision no
// pre-call pass could have seen. §6 says a `series` Operation reads its
// identities from a response by construction; §4 does not refuse the Manifest
// that does otherwise, and a fault no check produced is one this must not read
// past (ADR-0064).
//
// It reads every answer, faulted or not: a member that half-projected holds the
// Records it did read, and each of them is an identity a version will be
// written under.
func projectedAfterTheCall(bound binding, answers []answer, authored sequenced) []store.Identity {
	if !bound.operation.HasSeries && !strings.HasPrefix(bound.operation.Identity, "$") {
		return nil
	}
	var projected []store.Identity
	for _, held := range answers {
		for _, concluded := range held.records {
			projected = append(projected, authored.identity(concluded.name))
		}
	}
	return projected
}
