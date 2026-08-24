package cli

import (
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
)

// What `THE CODE MOVED` reads out of the clone (§8, §12, issue #171).
//
// The derivation is `internal/compare`'s and this is the reads it is handed:
// the reviewed artefacts at each end of the window, and what git says moved
// between the two revisions. That package opens no file and starts no
// subprocess, so the two `git` calls per window happen here — one `ls-tree`
// with its batch read per revision (revision.ArtefactsAt) and one `git diff`
// across the pair (revision.Between).
//
// **Nothing here fails on a revision the clone does not hold.** A Run recorded
// on a runner names a commit a laptop may never have fetched, and *not held* is
// an ordinary fact about the clone rather than the world resisting — what the
// table does with it is render the one §12 absence true of it (ADR-0071). A git
// that would not run at all arrives at the same sentence, on `readRange`'s own
// argument one file over: `not-in-clone` is the only one of the four names true
// of *the object is not readable here*, and a fifth minted for a subprocess
// that failed would be a name no reader could act on differently.

// codeReads is the clone as one invocation reads it: what each revision held,
// and what moved between each pair of them.
//
// It is one read per **revision** and per **pair** rather than one per window,
// which is `internal/run`'s reap making the same trade for the same reason: a
// fold across several Procedures — `--since` over a Store with more than one,
// or the whole-Store mode — renders one block each, and two Procedures whose
// Runs read one commit should cost git one `ls-tree` and not two. A revision
// this clone does not hold is remembered too: *not held* is an answer, and a
// fold over a dozen Procedures at one absent commit should cost one `rev-parse`
// (revision.ArtefactsAt, internal/run's revisions).
type codeReads struct {
	root      string
	revisions map[string]revisionRead
	between   map[string]movedRead
}

// revisionRead is what one revision answered: the repository it held, and
// whether this clone holds it at all.
type revisionRead struct {
	loaded repository.Loaded
	held   bool
}

// movedRead is what one pair of revisions answered: what git said moved between
// them, and whether it could be asked at all.
type movedRead struct {
	moved revision.Moved
	held  bool
}

// readingCode opens the reads for one invocation.
func readingCode(repoRoot string) codeReads {
	return codeReads{root: repoRoot, revisions: map[string]revisionRead{}, between: map[string]movedRead{}}
}

// readCode is what one window's third table is written from.
//
// **A window with no baseline reads nothing.** Its subject Run is the first Run
// of its Procedure, so there is no earlier revision for code to have moved
// from, no pair for `git diff` to name, and nothing for a count to count.
func (c codeReads) readCode(window compare.Window) compare.Code {
	if !window.Baseline.Present || !window.Subject.Present {
		return compare.Code{}
	}

	reach := reachedArtefacts(window)
	code := compare.Code{
		Baseline: c.sideAt(window.Baseline, reach),
		Subject:  c.sideAt(window.Subject, reach),
	}
	if !code.Baseline.InClone || !code.Subject.InClone {
		return code
	}

	// One `git diff` for the window, over the same file set the two reads
	// above were filtered by: §7's `repo_dirty` marks that file set too, "so
	// the marker and the count agree on what code is by construction" (§8).
	moved, held := c.moved(code.Baseline.Revision, code.Subject.Revision)
	if !held {
		code.Baseline.InClone, code.Subject.InClone = false, false
		return code
	}
	code.Count = moved.Count
	code.Baseline.Moved, code.Subject.Moved = moved.Before, moved.After
	return code
}

// sideAt is one end of the window's code: the revision that entry recorded, and
// the artefacts it held that either Run read.
func (c codeReads) sideAt(side compare.Side, reach codeReach) compare.CodeSide {
	provenance := side.Entry.Provenance
	read := compare.CodeSide{Revision: provenance.RepoRevision, Dirty: provenance.RepoDirty}
	if read.Revision == "" {
		return read
	}

	held := c.at(read.Revision)
	if !held.held {
		return read
	}
	read.InClone = true
	read.Artefacts = codeArtefacts(held.loaded, reach)
	return read
}

// at is the repository as one commit held it, asked of git once however many
// windows name that commit.
func (c codeReads) at(commit string) revisionRead {
	if answered, asked := c.revisions[commit]; asked {
		return answered
	}
	answered := revisionRead{}
	sources, held, err := revision.ArtefactsAt(c.root, commit, repository.IsArtefact)
	if err == nil && held {
		answered = revisionRead{loaded: repository.LoadFrom(artefactSources(sources)), held: true}
	}
	c.revisions[commit] = answered
	return answered
}

// moved is what git says moved between one pair of revisions, asked once.
//
// A pair git could not answer for is remembered like any other, for `at`'s
// reason: *not held* is an answer, and a fold whose every window names one
// absent commit should cost one `rev-parse` and not one per Procedure. The
// second member is what says which it was, an empty Moved being also what a
// pair that moved nothing answers.
func (c codeReads) moved(before, after string) (revision.Moved, bool) {
	key := before + "\x00" + after
	if answered, asked := c.between[key]; asked {
		return answered.moved, answered.held
	}
	answered := movedRead{}
	moved, held, err := revision.Between(c.root, before, after, repository.IsArtefact)
	if err == nil && held {
		answered = movedRead{moved: moved, held: true}
	}
	c.between[key] = answered
	return answered.moved, answered.held
}

// artefactSources is a revision's files in the shape a load reads them. The two
// types are one pair of members and are deliberately not one type:
// internal/revision answers what git holds and internal/repository answers what
// an artefact is, and neither is the other's (internal/run's reap does the same
// hand-off for the same reason).
func artefactSources(files []revision.File) []repository.Source {
	sources := make([]repository.Source, len(files))
	for i, file := range files {
		sources[i] = repository.Source{Path: file.Path, Bytes: file.Bytes}
	}
	return sources
}

// codeArtefacts is one revision's load as the table's subjects: the kind word
// §8's `SUBJECT` column qualifies a name with, the name, the path the moved
// lines are held under, and the facts the file's own lines carry.
//
// An artefact that would not parse contributes no fact and no subject. What is
// wrong with it is `check`'s to report, and a row derived from bytes `hyper`
// could not read would be this surface asserting a fact nobody wrote
// (ADR-0064); its lines fall to the catch-all's count like any other.
func codeArtefacts(loaded repository.Loaded, reach codeReach) []compare.CodeArtefact {
	var held []compare.CodeArtefact
	for _, found := range loaded.Artefacts {
		kind, known := kindOf(found.Path)
		if !known || !found.OK {
			continue
		}
		name := artefact.DeclaredName(found.Root, kind.nameKey)
		if kind.wire == artefact.KindRepositoryDeclaration {
			// The one artefact that declares no name of its own, and
			// the one keyed by its filename (§3, §12). It is the
			// subject `hyper_version` takes, the pin being the
			// repository's declaration (§11).
			name = repository.DeclarationPath
		}
		if !reach.read(kind.wire, name) {
			continue
		}
		held = append(held, compare.CodeArtefact{
			Kind:  kind.subject,
			Name:  name,
			Path:  repositoryPath(found.Path),
			Facts: artefact.ReadChangeFacts(kind.wire, found.Root),
		})
	}
	return held
}

// repositoryPath is where an artefact's file sits, and "" for the one Manifest
// that has none: the built-in `shell` Provider's bytes are compiled into the
// binary and its pseudo-path names no blob (§11, ADR-0039). A path no `git
// diff` can name is one no classed row subtracts a line at.
func repositoryPath(loadedPath string) string {
	if loadedPath == artefact.BuiltinShellProviderPath {
		return ""
	}
	return loadedPath
}

// codeReach is which artefacts the two Runs read: the names they named, under
// §12's own `kind:` value for each.
//
// **The table ranges over the artefacts the window's two Runs read and not over
// the repository.** §8 fixes that for the two Manifest classes outright —
// *which Manifests a Run read is the Step files' `provider`*, `manifest_digest`
// naming the bytes that ran and never which Provider they were — and the same
// evidence answers the other three kinds off the same files. What it keeps out
// is another Procedure's artefacts, which is the rule the window itself exists
// for; what those lines still get is the catch-all's count, which ranges over
// the reviewed five whole and is what makes the enumeration and the count sum
// to it (§8, §12).
//
// It is keyed by the `kind:` value rather than held as one set per kind because
// the kinds are §12's closed five and not four members and a special case: a
// roster with a field each would be the same fold written four times and a
// switch to pick between them, and the one artefact that is read without asking
// would still not be among them.
//
// **What the evidence cannot supply is an artefact only an unreached Step would
// have named**, and it is stated here rather than left to be found: a Step that
// was never reached writes no file (§7), so a Definition or a Target
// declaration bound by that Step alone is outside the reach and its moved lines
// fall to the catch-all's count instead of drawing a classed row. Nothing is
// dropped — the word *other* is what guarantees that — and the alternative is
// resolving each Procedure's declared Steps at each revision, which is the
// second supply §8 declines for the Manifests and would leave one kind read off
// the Journal and three off the bytes.
type codeReach map[string]map[string]bool

// read reports whether either Run read that artefact. The Repository
// declaration is read by every Run there is, governing all of them (§11), so it
// is answered without asking.
func (r codeReach) read(wire, name string) bool {
	return wire == artefact.KindRepositoryDeclaration || r[wire][name]
}

// mark records a name a Run read under the kind it is, and records nothing for
// a record that named none — the empty string being the absence of a name
// rather than a name (§7).
func (r codeReach) mark(wire, name string) {
	if name == "" {
		return
	}
	if r[wire] == nil {
		r[wire] = map[string]bool{}
	}
	r[wire][name] = true
}

// reachedArtefacts folds both entries' `run.json` and Step files into that
// reach: the top-level Procedure each Run named, the nested Procedures its
// Steps were reached through, and the Definition, Target declaration and
// Manifest each Step bound.
//
// It is the union across the window rather than one side's. An artefact one Run
// read and the other did not is exactly the change this table exists to report
// — a Step that stopped binding `staging` and started binding `production` is
// two Target declarations, and a side that never read one renders `–`.
func reachedArtefacts(window compare.Window) codeReach {
	reach := codeReach{}
	for _, side := range []compare.Side{window.Baseline, window.Subject} {
		if !side.Present {
			continue
		}
		reach.mark(artefact.KindProcedure, side.Entry.Procedure)
		for _, step := range side.Steps {
			for _, through := range invokedProcedures(step.Path) {
				reach.mark(artefact.KindProcedure, through)
			}
			reach.mark(artefact.KindDefinition, step.Definition)
			reach.mark(artefact.KindTargetDeclaration, step.Target)
			reach.mark(artefact.KindProvider, step.Provider)
		}
	}
	return reach
}

// invokedProcedures is the Procedures a Step was reached through, read off its
// invocation chain.
//
// The chain is the **Procedures** invoked and never the ids that invoked them,
// ending in the Step's own authored id — `retire.probe` is the Step `probe` of
// the Procedure `retire` (§7) — so the components before the last are the
// Procedures, and a chain of one component is a Step id and no invocation at
// all. It is `invokes`' reading one file over, answering which rather than
// whether.
func invokedProcedures(chain string) []string {
	components := strings.Split(chain, ".")
	if len(components) < 2 {
		return nil
	}
	return components[:len(components)-1]
}
