package cli

import "github.com/TheLoomLabs/hyper/internal/store"

// Provenance on the wire (§7, ADR-0043, issue #166).
//
// It reaches three surfaces in two positions — a row of its own on `run` and on
// `hyper show`, and one member of a `records` row — and it is declared here so
// that its seven names, their order and their absence rules are one statement
// rather than one per surface. Which position a rendering uses is that
// surface's; what Provenance says is this file's.

// provenanceRow is which code performed the Run, at one of the two scopes §7
// splits it across: the Run-wide members, and one row per Step file written.
//
// Which scope a row is is read off the row itself and never off a key naming
// one. A Step's row carries `step` and the Run's does not, which is exactly the
// split the Store already writes — a member is written at the level where it
// has one value and omitted from every level where it has none (ADR-0043) — and
// a discriminator beside it would carry that fact twice.
//
// Nothing is abbreviated: every revision and every digest goes out whole (§8,
// ADR-0047).
//
// Its members are provenanceBlock's, embedded rather than restated, because
// Provenance reaches the wire at two positions: a row of its own here, and a
// member of a `records` row, which carries the whole of it under one key (§9,
// issue #166). Embedding is what keeps the seven names one list — encoding/json
// writes an embedded struct's fields inline, in declaration order, so the key
// order is the same wherever the block stands — where a second declaration
// would be seven names two surfaces could come to spell differently.
type provenanceRow struct {
	Type string `json:"type"`
	Step *int   `json:"step,omitempty"`
	provenanceBlock
}

// provenanceBlock is Provenance's members and nothing else: the Run-wide half
// and the Step's half, in §7's own order, each following the ordinary absence
// rule (§7, ADR-0043).
//
// It carries no `type` and no `step`, those being facts about where a row
// stands rather than about the code that performed a Run — which is why a
// `records` row holds one under `provenance` and a stream holds one as a row of
// its own, from one declaration.
type provenanceBlock struct {
	HyperVersion      string `json:"hyper_version,omitempty"`
	ProcedureRevision string `json:"procedure_revision,omitempty"`
	RepoRevision      string `json:"repo_revision,omitempty"`
	RepoDirty         bool   `json:"repo_dirty,omitempty"`

	DefinitionRevision string `json:"definition_revision,omitempty"`
	ManifestDigest     string `json:"manifest_digest,omitempty"`
	OriginDigest       string `json:"origin_digest,omitempty"`
}

// runProvenanceRow is the Run-wide scope: the members that have exactly one
// value across a Run, however many Definitions its Steps span (ADR-0043).
func runProvenanceRow(p store.RunProvenance) provenanceRow {
	return provenanceRow{
		Type: "provenance",
		provenanceBlock: provenanceBlock{
			HyperVersion:      p.HyperVersion,
			ProcedureRevision: p.ProcedureRevision,
			RepoRevision:      p.RepoRevision,
			RepoDirty:         p.RepoDirty,
		},
	}
}

// stepProvenanceRow is one Step's scope, and the `step` it carries is what
// tells the two apart on the wire.
//
// It takes the position and the Store's own half rather than a Step value,
// because the two commands that render one hold the Step in two shapes — a Run
// reports what it just did and `hyper show` reads a Step file back — and the
// row is the same row (ADR-0026).
func stepProvenanceRow(position int, p store.StepProvenance) provenanceRow {
	return provenanceRow{
		Type: "provenance",
		Step: &position,
		provenanceBlock: provenanceBlock{
			DefinitionRevision: p.DefinitionRevision,
			ManifestDigest:     p.ManifestDigest,
			OriginDigest:       p.OriginDigest,
		},
	}
}

// Cells is empty: Provenance is not on §8's Step table. What renders it to a
// human is `hyper show`, which reads the entry back (§9); the page a Run writes
// is the table and the terminal line, and a row with no line is the shape the
// terminal row already has (ADR-0026).
func (r provenanceRow) Cells() []string { return nil }
