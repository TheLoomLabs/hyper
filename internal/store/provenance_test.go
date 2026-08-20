package store_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Provenance splits by scope, and the split is a rule about where a member
// *may* be written: at the level where it has exactly one value, and nowhere
// else (§7, ADR-0043). What decides whether a level restates a member it may
// write is where the reader stands — a Step file sits beside run.json and reads
// the Run-wide members one file over, and a Record version sits under a Record
// path with no entry beside it (issue #129).

func TestProvenance_IsWrittenAtTheLevelWhereItHasExactlyOneValue(t *testing.T) {
	// The six members, and the three files. Each row says which file each
	// member belongs to, and the assertion below is both halves at once:
	// present where it belongs, absent everywhere else.
	runWide := []string{"hyper_version", "procedure_revision", "repo_revision"}
	stepWide := []string{"definition_revision", "manifest_digest", "origin_digest"}

	whole := theProvenance
	whole.Step.OriginDigest = "sha256:1e5a4b2c9d80f7361a4e8b02c95d7f13ae6082b47c1d9350fe27a8b61c0d4e93"

	version := recordVersion(t)
	version.Provenance = whole

	run := runFile(t)
	run.Provenance = whole.Run

	for name, tc := range map[string]struct {
		file    string
		carries []string
		omits   []string
	}{
		"a Record version carries the whole of it": {
			file:    string(version.Encode()),
			carries: append(append([]string{}, runWide...), stepWide...),
		},
		"run.json carries the Run-wide members": {
			file:    string(run.Encode()),
			carries: runWide,
			omits:   stepWide,
		},
		"a Step file carries the Step's": {
			file:    string(stepFile().Encode()),
			carries: []string{"definition_revision", "manifest_digest"},
			omits:   runWide,
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, member := range tc.carries {
				if !strings.Contains(tc.file, `"`+member+`"`) {
					t.Errorf("file:\n%s\nwant it to carry %q", tc.file, member)
				}
			}
			for _, member := range tc.omits {
				if strings.Contains(tc.file, `"`+member+`"`) {
					t.Errorf("file:\n%s\nwant it to carry no %q", tc.file, member)
				}
			}
		})
	}
}

func TestProvenance_OmitsOriginDigestForAProviderWithNoUpstream(t *testing.T) {
	// Absent for a built-in Provider and for a locally authored one,
	// neither having an upstream to have come from (§11, ADR-0073).
	if got := string(stepFile().Encode()); strings.Contains(got, "origin_digest") {
		t.Errorf("step file:\n%s\nwant no origin_digest", got)
	}

	file := stepFile()
	file.Provenance = store.StepProvenance{
		DefinitionRevision: theProvenance.Step.DefinitionRevision,
		ManifestDigest:     theProvenance.Step.ManifestDigest,
		OriginDigest:       "sha256:1e5a4b2c9d80f7361a4e8b02c95d7f13ae6082b47c1d9350fe27a8b61c0d4e93",
	}
	if got := string(file.Encode()); !strings.Contains(got, `"origin_digest": "sha256:1e5a`) {
		t.Errorf("step file:\n%s\nwant the registry digest install verified", got)
	}
}

func TestProvenance_WritesRepoDirtyOnlyWhereItApplies(t *testing.T) {
	// It follows the ordinary absence rule rather than dry_run's exception:
	// one renderer reads it, and reading it wrong costs a `git diff` that
	// does not reproduce rather than a Procedure that refuses forever (§7).
	clean := runFile(t)
	if got := string(clean.Encode()); strings.Contains(got, "repo_dirty") {
		t.Errorf("run.json:\n%s\nwant no repo_dirty key at all", got)
	}

	dirty := clean
	dirty.Provenance.RepoDirty = true
	if got := string(dirty.Encode()); !strings.Contains(got, `"repo_dirty": true`) {
		t.Errorf("run.json:\n%s\nwant the marker", got)
	}
}
