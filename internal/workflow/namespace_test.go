package workflow_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// TestPath_CarriesTheProcedureNameVerbatim is §10's own sentence: one file per
// Procedure, at `.github/workflows/hyper-<procedure>.yml`.
func TestPath_CarriesTheProcedureNameVerbatim(t *testing.T) {
	if got, want := workflow.Path("retire-preview-envs"), ".github/workflows/hyper-retire-preview-envs.yml"; got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

// TestProcedureOf_IsPathReadBackwards holds the two halves of the namespace to
// each other: whatever Path writes, ProcedureOf reads back, so *which file is
// this Procedure's* has one answer.
func TestProcedureOf_IsPathReadBackwards(t *testing.T) {
	for _, procedure := range []string{"retire-preview-envs", "a", "on", "one.two"} {
		name, inside := workflow.ProcedureOf(workflow.Path(procedure))
		if !inside || name != procedure {
			t.Errorf("ProcedureOf(Path(%q)) = %q, %v; want %q, true", procedure, name, inside, procedure)
		}
	}
}

// TestProcedureOf_AnswersFalseOutsideTheNamespace is the other half of what
// `project` owns: a path it does not speak for is one it must not remove.
func TestProcedureOf_AnswersFalseOutsideTheNamespace(t *testing.T) {
	outside := []string{
		".github/workflows/release.yml",
		".github/workflows/hyper-nightly.yaml",
		".github/workflows/nested/hyper-nightly.yml",
		".github/hyper-nightly.yml",
		"hyper-nightly.yml",
		".github/workflows/hyper-nightly.yml.bak",
	}
	for _, path := range outside {
		if name, inside := workflow.ProcedureOf(path); inside {
			t.Errorf("ProcedureOf(%q) = %q, true; want it outside the namespace", path, name)
		}
	}
}
