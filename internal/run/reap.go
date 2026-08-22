package run

import (
	"errors"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The reap: closing every open entry the Journal holds (§6, §7, ADR-0003,
// ADR-0076, issue #154).
//
// **A second interrupt kills the process and leaves an entry holding no account
// at all.** There is no reaper, no daemon and no heartbeat: an abandoned entry
// is noticed by the next Run that looks, and until it is, a Step that was in
// flight reads as *never reached* — which re-runs an effect nobody vouched for
// the moment run-once is in the tool (§6, once.go).
//
// **It rides inside the push that sends `run.json`.** The fetch has landed one
// layer up, so every open entry on the branch is readable, and each one's
// closing write goes out **beside** `run.json` rather than taking a reach of
// its own. They are one commit for the same reason: the reap is decided from
// one fetched tip at one instant, and a branch that could hold the entry
// without the inferences it was drawn beside would be a state no Run ever put
// it in (§6, ADR-0076).
//
// **A Run that then declines at a gate has already reaped**, which is the same
// reason §6 puts `run.json` before the gates at all.
//
// **Only an effectful Run reaps.** A read-only Run holds the shared lock, so it
// can find a live effectful Run's entry open with no way to tell it from an
// abandoned one; it reads and never reaps (§6, lock.go).
//
// *Effectful* is the same reading that decided the lock and the push rhythm —
// the Kinds the Procedure's Steps bind — and it is deliberately **not** read
// against `--dry-run`. A rehearsal of an effectful Procedure takes the
// exclusive lock and pushes its entry like any other, so it holds exactly the
// guarantee the sentence above is about; and what it writes here is a true fact
// about a Run that is not it — this Run read the Journal and found that entry
// holding no account — inside the dead Run's own entry rather than its own. The
// rehearsal exception ADR-0001 is bought against is a rehearsal's own record
// being read back as evidence of an effect, which a closing write is not
// (§7, ADR-0001, lock.go).
//
// **A wrong reap costs nothing.** The closing write lands at a path only its
// writer can reach, so a Run that was alive after all finishes on its own terms
// and the entry ends up **contested** rather than decided by whichever Run
// reached the remote first. Both accounts stand and `hyper` picks no side
// (§7, ADR-0076).

// reaping is the closing writes this Run makes: one per open entry the Journal
// holds, and nothing at all on a read-only Run.
//
// **It reaps every open entry it finds and not a subset**, which is the rule
// store.OpenEntries is written under and enforced by there being nothing here
// to filter with (§6, ADR-0076).
//
// It is called before `run.json` is appended, which is what keeps this Run's
// own entry out of what it reads: an entry that does not exist yet is not one
// this Run can find open.
func (r run) reaping() ([]store.Write, error) {
	if !r.effectful {
		return nil, nil
	}
	open, err := r.request.Store.OpenEntries()
	switch {
	case errors.Is(err, store.ErrUnreadable):
		// **A file this binary cannot read is not a reap that failed.**
		// §6 puts the Store schema test one gate further on and
		// quantifies it over the Journal *whole*, so the condition this
		// read just met is the one that gate exists to report — with
		// `store-schema-unsupported`, the path it sits at, and the `77`
		// §12 fixes for it (gates.go, store.Readable). Reaping nothing
		// leaves every entry exactly as open as it was, which is what a
		// binary that cannot read the record ought to leave behind, and
		// the Run declines one line later with the code and the file
		// rather than with a fault about a reap nobody asked for.
		//
		// **It is that condition and no other.** Every other way this
		// read stops — a branch git would not list, an entry sitting at
		// a path its own run.json does not build — is one no gate goes
		// on to report, so tolerating it would be a Run completing at
		// `0` having quietly reaped nothing (§7, store.OpenEntries).
		return nil, nil
	case err != nil:
		return nil, err
	}

	// The instant read once for the whole reap rather than once per entry: a
	// Run that closed four entries drew one inference at one moment, which
	// is what it did.
	inferred := r.request.Now()
	read := reading(r.request.RepoRoot)
	writes := make([]store.Write, 0, len(open))
	for _, entry := range open {
		code, err := wentQuietOn(entry, read)
		if err != nil {
			return nil, err
		}
		writes = append(writes, store.Write{
			Path: entry.At().ClosedByPath(r.id),
			Content: store.ClosedBy{
				// The closing Run's instant on the closing Run's
				// clock, which is why a reaped entry renders no
				// duration at all: subtracting the dead Run's
				// `started_at` from it is the cross-entry
				// subtraction §7 forbids (§7, §8).
				EndedAt: inferred,
				// The Step the dead Run went quiet on: the one
				// after the highest ordinal its entry holds. A
				// Run that wrote no Step file at all went quiet
				// on Step 1, which is this arithmetic and not a
				// case of its own (§7).
				Step:     entry.Last + 1,
				StepCode: code,
			}.Encode(),
		})
	}
	return writes, nil
}

// wentQuietOn is the Step's code facts as the dead Run's own revision resolves
// them, and the zero value where it does not.
//
// **Which Step it was is derived and not guessed.** `run.json` names the
// Procedure and the **repository** revision to load it at, and the highest
// ordinal the entry holds is the last Step that finished — so the Step is the
// one after it, read out of the sequence that revision builds. It is
// `repo_revision` and never `procedure_revision`: reconstructing the sequence
// means loading every Procedure the top-level one invokes and the artefacts
// their Steps bind, which a commit resolves and a blob id cannot (§7).
//
// **It carries what the reaper knows and omits what it cannot establish.**
// Three readings reach the zero value and each is an honest absence rather than
// a failure:
//
//   - The dead Run recorded `repo_dirty`, so the commit it named is not the
//     code it ran. §7 names this one outright — *absent where it does not,
//     which is every Run that recorded `repo_dirty`*.
//   - This clone does not hold that commit, which is every Run recorded on a
//     runner whose code branch a laptop never fetched (revision.ArtefactsAt).
//   - The revision resolves and the Step does not: an invocation naming
//     nothing or a cycle leaves a sequence whose ordinals name different Steps
//     than the dead Run's did, and an ordinal past the end of one names no Step
//     at all.
//   - The Step resolves and its binding does not — a Definition, a Target or an
//     Operation that revision does not hold, which is a repository `check`
//     refuses.
//
// **The id and the code facts are one group**, written together or omitted
// together. §7 states them as one — *the Step's `id` and its code facts where
// the dead Run's revision resolves them, and absent where it does not* — so
// there is no fourth shape in which some of them stand.
//
// A git failure is not one of the four and is answered as the error it is: by
// the time a reap runs, this repository has already had the lock taken in it,
// the Store synced through it and `HEAD` resolved against it, so a git that
// will not list a commit it just resolved is the world resisting rather than
// something a reaper may write around.
func wentQuietOn(entry store.OpenEntry, read revisions) (store.StepCode, error) {
	if entry.Provenance.RepoDirty {
		return store.StepCode{}, nil
	}
	loaded, held, err := read.at(entry.Provenance.RepoRevision)
	if err != nil || !held {
		return store.StepCode{}, err
	}

	walked := flatten(loaded, entry.Procedure)
	// A sequence the walk could not reach in full is one whose positions do
	// not line up with the dead Run's: an invocation naming nothing
	// contributes no Steps and every Step after it moves up. The ordinal is
	// then a number about a different Procedure, and §7's *derived, not
	// guessed* is the whole reason to write nothing rather than that.
	if !walked.Whole || entry.Last+1 > len(walked.Steps) {
		return store.StepCode{}, nil
	}
	authored := walked.Steps[entry.Last]

	bound, err := resolve(loaded, authored)
	if err != nil {
		// The id and the code facts are one group and are written or
		// omitted together: §7 says *the Step's `id` and its code facts
		// where the dead Run's revision resolves them, and absent where
		// it does not*, and a revision whose Definition, Target or
		// Manifest does not resolve is one that does not. Writing the
		// id alone would be a shape the section does not name, from a
		// revision `check` would refuse.
		return store.StepCode{}, nil
	}
	return store.StepCode{
		ID:         authored.ID,
		Definition: authored.Definition,
		Operation:  authored.Operation,
		Provider:   bound.manifest.Name,
		Target:     authored.Target,
		Kind:       store.Kind(bound.operation.Kind),
	}, nil
}

// revisions is the code branch as one reap read it, and what it has already
// asked git about.
//
// It is one read per **revision** rather than one per entry: two Runs killed on
// one afternoon named one commit, and asking git the same question twice is two
// subprocesses for one answer — the same trade every batch read in
// internal/store makes. A revision this clone does not hold is remembered too:
// *not held* is an answer, and a Journal of fifty entries at one absent commit
// should cost one `rev-parse` and not fifty.
type revisions struct {
	root string
	read map[string]revisionRead
}

// revisionRead is what one revision answered: the repository it held, and
// whether this clone holds it at all. It is the (value, held) pair every reader
// in this package answers, remembered rather than asked again.
type revisionRead struct {
	loaded repository.Loaded
	held   bool
}

func reading(root string) revisions {
	return revisions{root: root, read: map[string]revisionRead{}}
}

// at is the repository as one commit held it, and whether this clone holds that
// commit at all. A revision it does not hold is answered rather than errored:
// that is an ordinary fact about the clone — a Run recorded on a runner whose
// code branch a laptop never fetched — and never the world resisting
// (revision.ArtefactsAt).
func (v revisions) at(commit string) (repository.Loaded, bool, error) {
	if answered, asked := v.read[commit]; asked {
		return answered.loaded, answered.held, nil
	}
	sources, held, err := revision.ArtefactsAt(v.root, commit, repository.IsArtefact)
	if err != nil {
		return repository.Loaded{}, false, err
	}
	answered := revisionRead{held: held}
	if held {
		answered.loaded = repository.LoadFrom(artefactSources(sources))
	}
	v.read[commit] = answered
	return answered.loaded, answered.held, nil
}

// artefactSources is the revision's files in the shape a load reads them. The
// two types are one pair of members and are deliberately not one type:
// internal/revision answers what git holds and internal/repository answers what
// an artefact is, and neither is the other's.
func artefactSources(files []revision.File) []repository.Source {
	sources := make([]repository.Source, len(files))
	for i, file := range files {
		sources[i] = repository.Source{Path: file.Path, Bytes: file.Bytes}
	}
	return sources
}
