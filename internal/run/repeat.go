package run

import (
	"errors"
	"fmt"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Repeatability at the Run: `skip-if-recorded`'s test, and the two Dispositions
// it decides between (§6, §7, §8, §12, ADR-0011, ADR-0056, issue #152).
//
// **It skips while the Asset a call would produce still stands**, which is a
// fact the head version of that Record's series carries (§12). The test
// therefore decides at the granularity of the key it reads, which is **per
// Record and not per Step**: a Step's Expansion holds one such series per
// member, and whether the Asset a call would produce still stands is a question
// about that member (ADR-0056). A Step may skip two members and call for one.
//
// **Three member states, and each of them is decided.** A member whose head
// stands is skipped. A member whose head is a **Tombstone** runs, the series
// standing for nothing — so create, destroy, and create again is three Runs
// that each do what they say rather than a third that reports `completed`
// having built nothing (§7, ADR-0011). A member naming **no series at all**
// runs, there being nothing for it to have been recorded as. A Step carrying no
// selector is that same test over the one series it would write, which is the
// rule reaching the commonest Step shape with no special case (§6).
//
// **The test is taken at each member's turn**, which is what puts it here and
// not at the Expansion. An effectful Expansion is serial and its versions go
// down between one call and the next, so a head this test reads is the branch
// as it stands when that member's turn comes — and two members that resolve one
// identity are then *the first runs and the rest skip*, the test having become
// true of them, rather than two calls decided against one branch neither had
// written to yet (§4, §6). Nothing is dropped for standing either: what the
// Store shortens is a `destroy`'s `values:` list, and only for what it knows is
// gone, so every member the selector resolved to is in `expanded_to` whether
// its call went out or not (§5, §7, expand.go).
//
// **It reads the Store's head version and never the Journal.** Unlike run-once
// it consumes no Disposition, which is why the Disposition a mixed Step carries
// decides nothing about a later Run and is a fact for a reader rather than an
// input to anything (§6, §12).
//
// **Two static guards stand in front of it and neither is re-read here.**
// `skip-if-recorded` is a `mutate`-only value, and a `skip-if-recorded` Step
// expanding over `assets:` is `skip-if-recorded-unreachable` — every member of
// such an Expansion stands by construction, so the test could only ever answer
// *skip* (§4, §5, internal/artefact). A `values:` list is the form that names a
// population `hyper` may not yet have built, which is the population this value
// exists to fill in.
//
// **The value's honest limit is unchanged and is now paid per member.**
// `skip-if-recorded` trusts the record over the world (§13), so a member
// hand-deleted between Runs is skipped rather than rebuilt. What the per-member
// test changes is that a member the artefact **newly asks for** is no longer
// hidden behind the members it already had.

// skipsIfRecorded says this Step's Operation declares `skip-if-recorded`.
//
// It reads the **declared** value rather than the effective one, and the two
// cannot disagree: `skip-if-recorded` is a value no Kind defaults to, run-once
// and `repeatable` being the only two a silence means (§12,
// artefact.OperationDetail).
func (b binding) skipsIfRecorded() bool {
	return b.operation.Repeatability == "skip-if-recorded"
}

// recorded is the Record name `skip-if-recorded` found still standing for one
// member, and "" where that member's call is to go out — which on every
// Operation that does not declare the value is every member.
//
// **The identity has to resolve before the call**, the test reading the head of
// the series the call would write under, and `check` holds a `skip-if-recorded`
// Operation to an `identity:` that does (`manifest-inconsistent`, §4,
// ADR-0056). Two forms have that property and both are read here: a template
// hole, which the Expansion already filled from this member's inputs, and
// `$.command` on a `shell` Operation, which is in the response object precisely
// because it is a fact about the **call** rather than about the answer (§3).
//
// The second is resolved here rather than at identityBeforeTheCall one file
// over, and the two are not one function. That one answers **which identities
// the Expansion's collision comparands may be run over**, and a `$.command`
// reaching it would move an existing halt on a colliding `shell` `mutate` to a
// Refusal on a Step this ticket says nothing about (§6, expand.go). What is
// shared is the spelling of the command, which is capability.Command's and is
// read from there by both the projection and this test.
//
// An identity that resolves to nothing **halts**, and it is a state `check`
// refuses rather than one a Run reaches: an Operation declaring this value with
// nothing to test is a Manifest declaring a test it cannot perform, and
// answering *run* for it would silently turn a guarded Step into an unguarded
// one (§4, ADR-0064).
func (r run) recorded(bound binding, authored sequenced, resolving member) (string, error) {
	if !bound.skipsIfRecorded() {
		return "", nil
	}
	name := skipIdentity(bound, resolving)
	if name == "" {
		return "", beforeTheCall(fmt.Errorf("step %s binds %s %s, which declares skip-if-recorded and whose identity: %s resolves to nothing before the call — hyper check reports it",
			named(authored), bound.manifest.Name, authored.Operation, bound.operation.Identity))
	}

	standing, err := r.stands(authored.identity(name))
	if err != nil {
		return "", beforeTheCall(err)
	}
	if !standing {
		return "", nil
	}
	return name, nil
}

// skipIdentity is the name the series this member's call would write under is
// held by, resolved before the call, and "" where nothing resolved one.
//
// The two arms are §4's two forms and there is no third: the hole the Expansion
// filled, and the argv a `shell` Operation names its Record by. The argv is
// spelled by capability.Command and never joined here — `[echo, "a b"]` and
// `[echo, a, b]` are two commands and must be two identities, and a second
// spelling of one of them would be a skip test reading a series the projection
// never writes to (§12, ADR-0051).
func skipIdentity(bound binding, resolving member) string {
	if bound.operation.Identity == "$.command" && bound.operation.IsShell {
		if len(resolving.Argv) == 0 {
			return ""
		}
		return capability.Command{Argv: resolving.Argv}.Text()
	}
	return resolving.Identity
}

// stands says the Asset this series holds still stands: the branch holds the
// series and its head version is not a Tombstone.
//
// It is the head version and no other. *Any version* would have the test reach
// a thing for what it used to be, which is the reading a selector's predicates
// are already held to one file over (§5, expand.go).
//
// The two false answers are one answer here and two sentences in §6 — a series
// the branch does not hold, and one whose head is a Tombstone — because what
// the caller does with them is identical: the member runs. What tells them
// apart is what the Run then writes, a version over a Tombstone being the one
// that makes the Head alive again (§7, ADR-0011).
func (r run) stands(id store.Identity) (bool, error) {
	head, standing, err := r.request.Store.Head(id)
	if err != nil || !standing {
		return false, err
	}
	return !head.Tombstone, nil
}

// skipFault is a member's turn that ended **before its call**: the Store the
// skip test could not read, and the identity that resolved to nothing.
//
// It is a type rather than a plain error because of where it is raised. The
// skip test runs inside the walk, which is the one place in a Step where an
// error means *this member's call answered badly* — and neither of these is a
// call or an answer: no request was made, so there is nothing for a Disposition
// to describe, and a Step file written *ran* over one would claim a conclusion
// `hyper` never reached (§12, store.DispositionRan). What stops the Run here is
// what stops it at heldBy one file over, so it travels back through the walk
// marked as what it is and leaves the Step by the Run's own door (step.go).
type skipFault struct{ err error }

func (f skipFault) Error() string { return f.err.Error() }
func (f skipFault) Unwrap() error { return f.err }

// beforeTheCall marks a fault the skip test raised in front of a member's call.
func beforeTheCall(err error) error { return skipFault{err: err} }

// haltedBeforeTheCall is the fault a member's turn carried in front of its
// call, and nil for every other way a member ends — a call that answered, a
// call that did not, and a member that skipped.
func haltedBeforeTheCall(fault error) error {
	var raised skipFault
	if errors.As(fault, &raised) {
		return raised.err
	}
	return nil
}
