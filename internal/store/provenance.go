package store

// Provenance is which code performed a Run, and it splits by scope: a member is
// written at the level where it has exactly one value and omitted from every
// level where it has none (§7, ADR-0043).
//
// The split is a rule about where a member *may* be written, and it is stated
// here as two types rather than as one type filtered three ways. A Step file
// that could carry `hyper_version` is a Journal entry holding two copies of it,
// which is the second representation the split exists to avoid — and a filter
// is a rule with a place to be got wrong, where a type is a rule with none.
//
// What decides whether a level *restates* a member it may write is where the
// reader stands. A Step file sits beside run.json and reads the Run-wide
// members one file over, so it carries none of them; a Record version sits
// under a Record path with no entry beside it, and carries the whole.

// The members each half always carries. OriginDigest is in neither, being
// absent for a built-in Provider and for a locally authored one, and RepoDirty
// is in neither, being a marker that says the ordinary case by its absence.
var (
	runProvenanceMembers  = []string{"hyper_version", "procedure_revision", "repo_revision"}
	stepProvenanceMembers = []string{"definition_revision", "manifest_digest"}
)

// RunProvenance is the Run-wide half: the members that have exactly one value
// across a Run, however many Definitions its Steps span. It is what makes a Run
// that wrote no Record still say which code performed it.
type RunProvenance struct {
	// HyperVersion is always a release string. The pin gate refuses any
	// binary whose version differs from the repository's in either
	// direction (§11), so there is no development form to write.
	HyperVersion string
	// ProcedureRevision is the git blob id of the **top-level** Procedure —
	// the file run.json's `procedure` names. A Run spans nested Procedures
	// as one Run, so that is the only reading with exactly one value, and
	// it has one for every Run (ADR-0036, ADR-0048).
	ProcedureRevision string
	// RepoRevision is the commit at HEAD. It is what a reaper loads the
	// Procedure sequence at, which a blob id could not do (§7).
	RepoRevision string
	// RepoDirty is true where any reviewed artefact the Run read differs
	// from HEAD or is untracked — exactly the file set §8's catch-all row
	// counts the moved lines of. It follows the ordinary absence rule
	// rather than dry_run's exception: one renderer reads it, and reading
	// it wrong costs a `git diff` that does not reproduce.
	RepoDirty bool
}

// StepProvenance is the Step's half: the members a Step has exactly one value
// for, a Step naming one Definition, one Operation and one Provider. Nothing at
// Run level names a Definition, so a Procedure whose Steps span several has
// nothing to disambiguate.
type StepProvenance struct {
	// DefinitionRevision is the git blob id of the Definition file:
	// content-addressed, computable offline, unmoved by a rebase, and equal
	// exactly where the content is.
	DefinitionRevision string
	// ManifestDigest is SHA-256 over the Manifest's exact bytes — the file
	// in providers/ for an installed or locally authored Provider, the
	// embedded bytes for a built-in. Over the bytes rather than a canonical
	// form of what they parse to, so that a reader checks it with
	// `sha256sum` and never by re-encoding a parse tree.
	ManifestDigest string
	// OriginDigest is the registry digest install verified. It is absent
	// for a built-in Provider and for a locally authored one, neither
	// having an upstream to have come from (§11, ADR-0073). It is not
	// ManifestDigest under another name even where both are present: that
	// one covers the file as it stands, this one the published bytes.
	OriginDigest string
}

// Provenance is the whole of it, and only a Record version carries the whole.
// A version file saying only *see Run abc* would be unreadable in a browser and
// in a diff, which is exactly where this field set is read.
type Provenance struct {
	Run  RunProvenance
	Step StepProvenance
}

// write puts the Run-wide members into a provenance block. hyper names the
// algorithm where hyper chose it, so the digests carry `sha256:` inline and the
// two revisions are bare, the algorithm there being the repository's.
//
// All three of runMembers have a value for every Run — the pin gate leaves no
// binary without a release version, every Run is a Run of a Procedure
// (ADR-0036), and HEAD is a commit — so RepoDirty is the only member of this
// half that is ever absent on its own.
func (p RunProvenance) write(m members) {
	m.text("hyper_version", p.HyperVersion)
	m.text("procedure_revision", p.ProcedureRevision)
	m.text("repo_revision", p.RepoRevision)
	m.mark("repo_dirty", p.RepoDirty)
	m.require("a Run's Provenance", runProvenanceMembers)
}

// write puts the Step's members into a provenance block. A Step names one
// Definition and one Provider, so both of stepMembers always have a value and
// only OriginDigest is ever absent on its own.
func (p StepProvenance) write(m members) {
	m.text("definition_revision", p.DefinitionRevision)
	m.text("manifest_digest", p.ManifestDigest)
	m.text("origin_digest", p.OriginDigest)
	m.require("a Step's Provenance", stepProvenanceMembers)
}

// write puts the whole of Provenance into a block, which is what a Record
// version writes and what no Journal file does.
func (p Provenance) write(m members) {
	p.Run.write(m)
	p.Step.write(m)
}

// readRunProvenance reads the Run-wide half back out of a provenance block.
func readRunProvenance(f *fields) RunProvenance {
	return RunProvenance{
		HyperVersion:      f.text("hyper_version"),
		ProcedureRevision: f.text("procedure_revision"),
		RepoRevision:      f.text("repo_revision"),
		RepoDirty:         f.mark("repo_dirty"),
	}
}

// readStepProvenance reads the Step's half back out of a provenance block.
func readStepProvenance(f *fields) StepProvenance {
	return StepProvenance{
		DefinitionRevision: f.text("definition_revision"),
		ManifestDigest:     f.text("manifest_digest"),
		OriginDigest:       f.text("origin_digest"),
	}
}
