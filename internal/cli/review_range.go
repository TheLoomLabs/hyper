package cli

import (
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The review's range: where it opens, and what it opens at (§8, ADR-0067,
// ADR-0071, issue #164).
//
// A range opens at the last Run that read the artefact, and never at `HEAD`.
// Against `HEAD` an agent that authored a widened Bound and committed it leaves
// the two sides equal, and the review renders nothing to mark on the one branch
// a human is about to approve; the same agent commits `env: STAGING_TOKEN` →
// `env: PROD_TOKEN`, so the argument is not a Procedure's and `HEAD` is refused
// on all five artefacts.
//
// **Nothing here decides anything from the working tree.** Which Runs read an
// artefact is a question about the Journal — `run.json`'s `procedure`, and a
// Step file's `definition`, `provider`, `target` or `path` — and what that Run
// supplies is a question about its Provenance. Neither varies with the
// artefact's `kind:` except by coincidence, which is the decomposition ADR-0067
// exists to name.
//
// **It never fails.** Every way this reading can stop is one of §12's two
// Journal-side absences: a Store that could not answer is `no-store` and a
// revision the clone does not hold is `not-in-clone`. That is not a swallowed
// error, it is §9's exit codes read backwards — `review` exits 1 for the
// artefact under review failing to load and for nothing else, so a range that
// cannot open has nowhere to go but the header's own sentence.

// reviewRange is what the Journal and the clone together supplied: whether the
// Store answered at all, the entry that read the artefact, the revision that
// entry named for it, and the blob the range opens at.
//
// The named revision and the blob are two members rather than one because they
// are two facts and only one of them always exists. On the two artefacts
// carrying a revision of their own they are the same string; on the other four
// the Run named a commit and the blob is that file's bytes under it, resolved
// at render and stored nowhere (ADR-0067). Where the clone holds no such
// object, `not-in-clone` renders what the Run named, which is the whole of what
// the reader has to work with.
type reviewRange struct {
	// stored says the Store answered. It is false where there is no branch,
	// where the repository root holds no git repository at all, and where
	// the branch is there and could not be read — a file written above this
	// binary's ceiling, or an entry that does not sit at the path its own
	// run.json builds. All three are the Store unreachable, which is the
	// one thing `no-store` says; none of them is *asked and empty* (§12).
	stored bool
	// supplied says an entry read this artefact and qualified to anchor it.
	supplied bool
	// entry is that entry, and the zero value where none supplied. It is
	// the one the gloss reads as well: the range and *last ran* are one
	// lookup in two notations (§8, §10).
	entry store.Entry
	// named is the revision that entry recorded for this artefact — a blob
	// id on the two artefacts that carry one, and the `repo_revision`
	// commit on the other four.
	named string
	// blob is the blob the range opens at, and "" where this clone does not
	// hold the object named.
	blob string
	// bytes are that blob's own bytes: the artefact as the supplying Run
	// read it, which is the whole of the gutter's supply across the range
	// (ADR-0057). They are read in the same call the blob is, one object
	// name answering both halves of one question.
	bytes []byte
}

// readRange is the range read for one artefact: the Store opened, the Journal
// walked backwards, and the one object the answer needs read out of the clone.
//
// It syncs nothing. A review resolves no credential, reaches no network and
// invokes nothing (§8), so it opens whatever branch this clone holds and never
// the one a fetch would bring — which is the one place an Inspection command's
// way in to the record (inspect.go) is deliberately not reused: those four
// Refuse where the branch is missing, and this one renders an absence and
// carries on.
//
// The artefact with no file in the repository is not asked about at all. No Run
// could have recorded a revision of what has none, so `built-in` fires one
// stage above this and a lookup here would be a question with a known answer
// (ADR-0068).
func readRange(reviewed reviewedArtefact, repoRoot string, now time.Time) reviewRange {
	if reviewed.path == "" {
		return reviewRange{}
	}
	held, err := store.Open(repoRoot, now)
	if err != nil {
		return reviewRange{}
	}

	found, supplied, err := supplyingEntry(held, reviewed)
	if err != nil {
		// A branch that is there and whose files will not be read is the
		// Store unreachable rather than the Store answering empty: what
		// `not-run` means is *asked and holds nothing that anchors this
		// file*, and nothing was asked here (§12).
		return reviewRange{}
	}
	if !supplied {
		return reviewRange{stored: true}
	}

	// The object the range opens at, named in whichever of the two forms
	// this artefact's anchor takes: the recorded blob id itself, or the
	// file's blob under the commit the same Run recorded.
	object := found.named
	if !found.own {
		object = revision.AtPath(found.named, reviewed.path)
	}
	opened := reviewRange{stored: true, supplied: true, entry: found.entry, named: found.named}
	at, inClone, err := revision.Held(repoRoot, object)
	if err != nil || !inClone {
		// A read the clone could not perform and one that answered
		// *missing* arrive at the same sentence, and that is the closed
		// set rather than a conflation: §12 names four absences and
		// `not-in-clone` is the only one true of *the object is not
		// readable here* — the Store answered, the Run is the right one,
		// and what failed is downstream of both. A fifth name would have
		// to be minted for a git that would not run, and what §9 leaves
		// no room for is the third option, declining (§9, §12).
		return opened
	}
	opened.blob, opened.bytes = at.Blob, at.Bytes
	return opened
}

// supplyingEntry is the Journal walked backwards for the most recent entry that
// read this artefact and qualifies to anchor it: the entry, the revision it
// named, and whether that revision is the artefact's own.
//
// **A rehearsal is out**, exactly as it is disqualified as the Comparison's
// baseline (§7): rehearsing a widened `destroy` Bound would otherwise retire
// the flag that was the warning, and a rehearsal disarming a review surface is
// a shape that rules everywhere rather than on the artefact the rehearsal
// named.
//
// **A `repo_dirty` entry is out for an artefact with no revision of its own.**
// The blob under that commit resolves perfectly and names bytes that did not
// run, so the gutter would mark a line that did not move and miss one that did
// — the one screen ADR-0026 says may not lie. The two members that survive a
// dirty tree are hashes of what ran and are unaffected, so a dirty entry
// anchors a top-level Procedure and a Definition like any other.
//
// It stops at the first entry that qualifies, which is what makes the walk cost
// the entries between rather than the Journal: a range read off a recent Run
// costs that Run's files (§7).
func supplyingEntry(held *store.Store, reviewed reviewedArtefact) (anchor, bool, error) {
	entries, err := held.Entries()
	if err != nil {
		return anchor{}, false, err
	}

	for _, candidate := range entries {
		if candidate.DryRun {
			continue
		}
		// The `run.json` reading first, and the Step files only where it
		// did not answer: a Repository declaration never reaches them,
		// and a Procedure reaches them only on the entries of Runs that
		// were not its own. On a Journal of a year of Runs that is the
		// difference between one listing and all of them (§7).
		found, anchored := anchorInRunFile(candidate, reviewed)
		if !anchored {
			records, err := held.Dispositions(candidate)
			if err != nil {
				return anchor{}, false, err
			}
			found, anchored = anchorInSteps(candidate, records.Steps, reviewed)
		}
		if !anchored {
			continue
		}
		if !found.own && candidate.Provenance.RepoDirty {
			continue
		}
		return found, true, nil
	}
	return anchor{}, false, nil
}

// anchor is what one entry supplies for one artefact: the entry, the revision
// it recorded for that artefact, and whether that revision is the artefact's
// own or the commit the whole Run read.
//
// The three travel together because the last decides what the first two mean.
// `own` is what says whether the revision is already the blob the range opens
// at or a commit one file has still to be resolved under, and it is the same
// bit the dirty filter turns on — so a caller holding the revision without it
// is a caller one step from resolving the wrong thing (ADR-0067).
type anchor struct {
	entry store.Entry
	named string
	own   bool
}

// atRepoRevision is the answer the four artefacts carrying no revision of their
// own give: the commit the whole Run read, which the caller resolves this
// file's blob under.
//
// It is written once because it is one answer given four times — a Target
// declaration's, a Manifest's, a Repository declaration's and a
// nested-only Procedure's — and what differs between them is only which record
// says the Run read the artefact, never what the entry then supplies.
func atRepoRevision(entry store.Entry) (anchor, bool) {
	return anchor{entry: entry, named: entry.Provenance.RepoRevision}, entry.Provenance.RepoRevision != ""
}

// anchorInRunFile is what an entry's `run.json` alone says: a **Repository
// declaration** is read by every Run there is, governing all of them (§11), and
// a **top-level Procedure** is read by the Run whose `run.json` names it.
//
// Those two and the four below are one rule split at the file that answers it,
// which is ADR-0067's own observation made structural: the Procedure appears on
// both sides, because ADR-0048 gives `procedure_revision` to the top-level
// Procedure and to no other and the same artefact fails the test one invocation
// down.
func anchorInRunFile(entry store.Entry, reviewed reviewedArtefact) (anchor, bool) {
	switch reviewed.kind.wire {
	case artefact.KindRepositoryDeclaration:
		return atRepoRevision(entry)
	case artefact.KindProcedure:
		if entry.Procedure != reviewed.name || reviewed.name == "" {
			return anchor{}, false
		}
		recorded := entry.Provenance.ProcedureRevision
		return anchor{entry: entry, named: recorded, own: true}, recorded != ""
	}
	return anchor{}, false
}

// anchorInSteps is what the entry's Step records say: a **Definition** is read
// by every Run with a record naming it in `definition`, a **Manifest** by every
// Run naming it in `provider`, a **Target declaration** by every Run naming it
// in `target`, and a **nested Procedure** by every Run carrying it in a
// record's `path` (§8).
//
// **The anchor is whether the artefact carries a revision of its own, not its
// kind** (ADR-0067). Only the Definition does, on the record that named it, and
// it is a git blob id over the bytes that Run actually read; the other three
// take the commit the whole Run read.
//
// A record naming the artefact but carrying no revision for it does not anchor,
// and the walk goes on to the entry before it. That is a closing write's
// reading of a Definition: it names what the Step was going to do and carries
// no Provenance, the reaper having established none (§7).
//
// All four match on the artefact's own declared name, and the empty string is
// the absence of a name rather than a name to match on: a Step file records the
// artefact it bound by name, and matching "" would answer every record that
// recorded none.
func anchorInSteps(entry store.Entry, records []store.StepFile, reviewed reviewedArtefact) (anchor, bool) {
	if reviewed.name == "" {
		return anchor{}, false
	}
	for _, record := range records {
		switch reviewed.kind.wire {
		case artefact.KindDefinition:
			if record.Definition == reviewed.name && record.Provenance.DefinitionRevision != "" {
				return anchor{entry: entry, named: record.Provenance.DefinitionRevision, own: true}, true
			}
		case artefact.KindProvider:
			if record.Provider == reviewed.name {
				return atRepoRevision(entry)
			}
		case artefact.KindTargetDeclaration:
			if record.Target == reviewed.name {
				return atRepoRevision(entry)
			}
		case artefact.KindProcedure:
			if invokes(record.Path, reviewed.name) {
				return atRepoRevision(entry)
			}
		}
	}
	return anchor{}, false
}

// invokes says whether a Step record's invocation chain was reached through the
// Procedure named.
//
// The chain is the **Procedures** invoked and never the ids that invoked them,
// ending in the Step's own authored id — `retire.probe` is the Step `probe` of
// the Procedure `retire` (§7, internal/run's sequence). So the components
// before the last are the Procedures this Step was reached through, and a chain
// whose last component happened to equal a Procedure's name is a Step id and
// not an invocation.
func invokes(chain, procedure string) bool {
	components := strings.Split(chain, ".")
	if len(components) < 2 {
		return false
	}
	for _, through := range components[:len(components)-1] {
		if through == procedure {
			return true
		}
	}
	return false
}
