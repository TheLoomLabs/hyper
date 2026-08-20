package store

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Compaction: the one act in the tool that removes anything, and the predicate
// that bounds it (§7, issue #131).
//
// Retention is read-time and lives in the Repository declaration alone. Nothing
// in this file reads a policy, and nothing anywhere may widen one: what a
// caller hands over is what a reviewed artefact declared, so a flag or an
// environment variable that let one invocation remove more than the repository
// agreed to has no way in (ADR-0001, ADR-0014).
//
// What may be removed is stated once, as the predicate below, and everything
// else on the branch stands: no Asset version, no version of an Asset series,
// no Tombstone, no Journal entry, no Provenance and never `STORE.md`. Pruning
// the entry that says a Step *ran* makes it *never reached* again, which is a
// bypass under a maintenance name (ADR-0006, ADR-0001) — so the predicate is
// written as the whole of what is removable rather than as a list of what is
// spared, and a shape it has never heard of is untouched by construction.

// Retention is the policy a Compaction acts under: what the Repository
// declaration says, and what that means as an age.
//
// Both halves are carried because both are read. The age is what the predicate
// measures against the clock, and the declared text is what the commit message
// and the command's own line name — a policy rendered as `2160h0m0s` where the
// artefact says `90d` would send a reader looking for a line nobody wrote (§3,
// §8).
type Retention struct {
	// Declared is the artefact's own spelling of the policy, `90d`.
	Declared string
	// Age is what it names. A version is removable where it is older than
	// this, so a version exactly this old stands: the policy is a length of
	// time the repository agreed to keep, and the boundary is inside it.
	Age time.Duration
}

// Removed is one version a Compaction took off the branch: what the file said
// about itself, and where it sat.
//
// It carries no ordinal, and the absence is the rule rather than an omission. A
// version's ordinal is its position in a series' ordering, and Compaction is
// precisely the thing that moves one — removing an interior version renumbers
// every version above it — so a value that carried one here would be handing a
// renderer the one number nothing may name a version by (ADR-0049). A removed
// version is named by its Run and its Step, which are the two segments of its
// file name and are stable forever.
type Removed struct {
	Metadata
	// File is the Store path the version sat at, which is what names it to
	// git and what a report of the removals is ordered by.
	File string
}

// Compaction is what one Compaction did: the versions it removed, and how many
// series it left alone.
//
// Untouched counts every Record series the branch holds that lost nothing — an
// Asset series, a series of one version, a series whose every interior version
// is younger than the policy. It is reported beside the count because the two
// answer the question an operator actually asks: *what did this leave*.
type Compaction struct {
	// Removed is every version taken off the branch, in path order.
	Removed []Removed
	// Untouched is the number of series nothing was removed from.
	Untouched int
}

// Compact removes the interior Observation versions the policy permits, commits
// the removal, and publishes it.
//
// **The predicate, stated once.** For every Observation series the branch
// holds, a version is removable exactly where it is not the Head, not the
// series' first version, and its `written_at` is older than the policy measured
// against the clock this handle was opened at. Everything else stands.
//
// It is one commit, and the commit message is its account: `git log` on the
// branch is what says what a Compaction removed (§7, §13), and it survives a
// push re-application intact. No Journal entry is written — Compaction is not a
// Run — and a Compaction that finds nothing to remove writes no commit at all,
// so a repository whose every interior version is young reaches this and leaves
// the branch byte-identical.
//
// The push follows every other write's path and its failures are the caller's
// to map: a push exhausted after three attempts answers ErrPushExhausted, which
// a command that is not a Run reports as the world resisting (§9).
func (s *Store) Compact(retention Retention) (Compaction, error) {
	records, err := s.Records()
	if err != nil {
		return Compaction{}, err
	}

	compaction := Compaction{}
	for _, series := range records {
		taken := removable(series, retention.Age, s.now)
		if len(taken) == 0 {
			compaction.Untouched++
			continue
		}
		compaction.Removed = append(compaction.Removed, taken...)
	}
	slices.SortFunc(compaction.Removed, func(a, b Removed) int {
		return strings.Compare(a.File, b.File)
	})

	if len(compaction.Removed) == 0 {
		return compaction, nil
	}
	if err := s.remove(compaction.Removed, retention); err != nil {
		// Nothing is answered where the removal did not land: what a
		// caller would render is a list of versions that are still there.
		return Compaction{}, err
	}
	if err := s.repo.publish(); err != nil {
		// The removal *did* land, and the branch this clone holds is the
		// compacted one — so what happened is answered beside the error
		// that says it went nowhere. It is the same fact ErrPushExhausted
		// states in words: what was written stands locally (§7).
		return compaction, err
	}
	return compaction, nil
}

// removable is the predicate over one series, and the whole of it.
//
// A series that holds an Asset version anywhere in it loses nothing, at any
// age: `record_type: asset` is `hyper`'s effect having reached the thing, and a
// Tombstone carries it too, so *no Asset version, no version of an Asset series
// and no Tombstone* is one test rather than three. That the test reads the
// whole series and not the version in front of it is deliberate — an
// Observation version sitting under a Head that Tombstones it is part of the
// account of an Asset, and removing it would leave the destruction of a thing
// nothing ever described.
//
// The first version and the Head are excluded by the range: a series of one
// version and a series of two both offer nothing, which is what makes *never
// the Head* and *never the first* structural rather than two comparisons that
// could disagree at the ends.
func removable(series Series, retention time.Duration, now time.Time) []Removed {
	if len(series.Versions) < 3 {
		return nil
	}
	for _, version := range series.Versions {
		if version.RecordType != RecordObservation || version.Tombstone {
			return nil
		}
	}

	var taken []Removed
	for _, version := range series.Versions[1 : len(series.Versions)-1] {
		// Older *than* the policy, so a version exactly the policy's age
		// stands: the repository agreed to keep its Records that long,
		// and the instant the agreement runs out is still inside it.
		if now.Sub(version.WrittenAt) <= retention {
			continue
		}
		taken = append(taken, Removed{Metadata: version.Metadata, File: version.File})
	}
	return taken
}

// remove writes the removal: one commit whose tree is the branch's with those
// paths off it, and the branch pointed at it. Publishing it is Compact's next
// act and not this one's — what the branch holds and what the remote holds are
// two facts, and they part exactly where a push could not complete (§7).
//
// The tree is derived by applying the removals as **path operations**, which is
// the same call a rejected push re-applies an unpushed commit with. That is not
// a convenience: a Compaction *is* a set of path operations, and expressing it
// as one is what makes it re-appliable at all (issue #131). Every surviving
// path keeps the blob it already had, so a Compaction rewrites no file and can
// mint no version.
//
// The branch is moved with its old value named, which is update-ref's own guard
// — a branch that moved between the read and the write is a second writer, and
// overwriting it is the act append-only forbids arriving at the ref. The handle
// is then pointed at what it wrote, so a read taken through it after a
// Compaction answers about the branch that now stands rather than about the one
// the versions were removed from.
func (s *Store) remove(removed []Removed, retention Retention) error {
	operations := make([]pathOperation, len(removed))
	for i, version := range removed {
		// A removal is the absence of a blob rather than a flag beside
		// one, so naming the path is the whole of stating it.
		operations[i] = pathOperation{path: version.File}
	}

	tree, err := s.repo.applyOnto(s.commit, operations)
	if err != nil {
		return err
	}
	commit, err := s.repo.commitOnto(tree, s.commit, compactionMessage(len(removed), retention))
	if err != nil {
		return err
	}
	if err := s.repo.moveRef(Ref, commit, s.commit); err != nil {
		return err
	}
	s.commit = commit
	return nil
}

// compactionMessage is what `git log` on the branch says about a Compaction: the
// count, and the retention policy it acted under. Nothing else — the paths are
// in the diff, and a message that listed them would be a second copy of what
// the commit already is.
//
// It names the policy as the artefact declared it, so a reader who runs `git
// log` and then opens `hyper.yaml` reads one string in both places (§3, §7).
func compactionMessage(count int, retention Retention) string {
	return fmt.Sprintf("Compact the %s: %s removed, retention %s", BranchName, InteriorVersions(count), retention.Declared)
}

// InteriorVersions is what a count of removed versions is called, in the words
// the predicate is stated in: they are interior, they are Observations, and
// they are versions rather than Records.
//
// It is exported because the commit message and the command's own line count
// one thing and must count it in one phrase — the message being a rendering
// with a reader, and the line being the same fact on the other surface (§7,
// §9).
func InteriorVersions(count int) string {
	if count == 1 {
		return "1 interior observation version"
	}
	return fmt.Sprintf("%d interior observation versions", count)
}
