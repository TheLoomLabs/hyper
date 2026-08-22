package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/repository"
)

// **A Procedure invoking another does not start a second Run** (§6, issue
// #141). The invoked Procedure's Steps are Steps of the one Run, in the one
// written order, under a path — and the invocation itself is not a Step.
//
// The corpus drives what that looks like from a page and a branch. What is held
// here is the walk: the order, the path, the file each Step was authored in,
// and the two shapes that leave it short — which no Run can reach, `check`
// refusing both, and which the lock is read off before `check` has run.

// TestFlatten_CountsANestedProceduresStepsInTheOneWrittenOrder is the sequence
// itself: the Steps around an invocation and the Steps inside it, in the order
// they run.
func TestFlatten_CountsANestedProceduresStepsInTheOneWrittenOrder(t *testing.T) {
	walked := flatten(sequenceFixture(t), "outer")

	var order []string
	for _, step := range walked.Steps {
		order = append(order, named(step))
	}
	want := []string{"before", "inner.first", "inner.deeper.deep", "inner.second", "after"}
	if len(order) != len(want) {
		t.Fatalf("the walk reached %v, want %v", order, want)
	}
	for at, name := range want {
		if order[at] != name {
			t.Errorf("step %d is %s, want %s", at+1, order[at], name)
		}
	}
}

// TestFlatten_WritesNoPathOnATopLevelStep is §7's other half: a Step reached
// through no invocation carries none, its own id being the whole of what names
// it.
func TestFlatten_WritesNoPathOnATopLevelStep(t *testing.T) {
	for _, step := range flatten(sequenceFixture(t), "outer").Steps {
		if step.ID == "before" || step.ID == "after" {
			if step.Path != "" {
				t.Errorf("top-level step %s carries path %q", step.ID, step.Path)
			}
		}
	}
}

// TestFlatten_CitesTheFileEachStepWasAuthoredIn is what a Refusal points at: a
// Step reached through an invocation was authored in the invoked Procedure's
// file, at its own position in that file's `steps:`, and neither is the Run's
// flattened position.
func TestFlatten_CitesTheFileEachStepWasAuthoredIn(t *testing.T) {
	for _, step := range flatten(sequenceFixture(t), "outer").Steps {
		want, index := "procedures/outer.yaml", 0
		switch step.ID {
		case "after":
			index = 2
		case "first":
			want = "procedures/inner.yaml"
		case "second":
			want, index = "procedures/inner.yaml", 2
		case "deep":
			want = "procedures/deeper.yaml"
		}
		if step.Declared.Path != want || step.Index != index {
			t.Errorf("step %s is cited at %s steps[%d], want %s steps[%d]", named(step), step.Declared.Path, step.Index, want, index)
		}
	}
}

// TestFlatten_ScopesEachInvocationSeparately is what a `when:` resolves an id
// against. An id is unique inside one Procedure and says nothing across two, so
// two invocations of one Procedure are two scopes — and the second's Steps read
// the second's Records rather than the first's.
func TestFlatten_ScopesEachInvocationSeparately(t *testing.T) {
	walked := flatten(sequenceFixture(t), "twice")

	scopes := map[int]int{}
	for _, step := range walked.Steps {
		scopes[step.Namespace]++
	}
	if len(scopes) != 2 {
		t.Fatalf("two invocations of one Procedure resolved %d scopes, want 2", len(scopes))
	}
}

// TestFlatten_ReadsEveryProcedureItWalked is the file set `repo_dirty` is
// decided over: a nested Procedure is an artefact the Run read (§7, §8).
func TestFlatten_ReadsEveryProcedureItWalked(t *testing.T) {
	var read []string
	for _, procedure := range flatten(sequenceFixture(t), "outer").Procedures {
		read = append(read, procedure.Path)
	}

	want := []string{"procedures/outer.yaml", "procedures/inner.yaml", "procedures/deeper.yaml"}
	if len(read) != len(want) {
		t.Fatalf("the walk read %v, want %v", read, want)
	}
	for at, path := range want {
		if read[at] != path {
			t.Errorf("read %s, want %s", read[at], path)
		}
	}
}

// TestFlatten_IsNotWholeWhereItCouldNotDescend is the walk's own honesty. An
// invocation naming a Procedure that is not there is `artefact-absent` at
// `check`, and a cycle is rejected before the first Step (ADR-0002) — so
// neither reaches a Run. The lock is read before either has been checked, and
// it reads this (lock.go).
func TestFlatten_IsNotWholeWhereItCouldNotDescend(t *testing.T) {
	loaded := sequenceFixture(t)

	if flatten(loaded, "outer").Whole != true {
		t.Error("a Procedure whose every invocation resolved is not whole")
	}
	for _, procedure := range []string{"absent", "cyclic"} {
		if flatten(loaded, procedure).Whole {
			t.Errorf("%s walked whole, and it does not", procedure)
		}
	}
}

// TestFlatten_NamesACycleRatherThanTruncatingIt holds the arm above as the
// property it is there for: the walk answers rather than recursing forever, and
// it says which Procedure closed the loop.
//
// `check` refuses a cycle before Step 1 — `procedure-cycle`, cited at the
// invocation entry that closes the loop (§4, issue #146) — so what the name is
// for is Perform's own precondition past that gate: it stops rather than
// performing a Procedure with the recursive invocation quietly dropped.
func TestFlatten_NamesACycleRatherThanTruncatingIt(t *testing.T) {
	walked := flatten(sequenceFixture(t), "cyclic")

	if len(walked.Steps) != 1 || walked.Steps[0].ID != "round" {
		t.Errorf("the walk reached %d Steps, want the one that is not the invocation", len(walked.Steps))
	}
	if walked.Cycle != "cyclic" {
		t.Errorf("the walk named %q as the cycle, want cyclic", walked.Cycle)
	}
	if named := flatten(sequenceFixture(t), "absent").Cycle; named != "" {
		t.Errorf("an invocation naming nothing was reported as the cycle %q; that is artefact-absent", named)
	}
}

// sequenceFixture is the Procedures these cases walk. It carries no Provider,
// no Definition and no Target: the walk resolves `procedure:` and reads
// `steps:`, and judges neither — what is wrong with a Step is `check`'s
// (ADR-0064).
func sequenceFixture(t *testing.T) repository.Loaded {
	t.Helper()

	step := func(id string) string {
		return "  - {id: " + id + ", definition: d, operation: o, target: local}\n"
	}
	root := t.TempDir()
	for path, content := range map[string]string{
		"hyper.yaml": "kind: repository-declaration\nversion: 1.4.0\nretention: 90d\n",
		"procedures/outer.yaml": "kind: procedure\nprocedure: outer\ntargets: [local]\nsteps:\n" +
			step("before") + "  - {id: in, procedure: inner}\n" + step("after"),
		"procedures/inner.yaml": "kind: procedure\nprocedure: inner\ntargets: [local]\nsteps:\n" +
			step("first") + "  - {id: down, procedure: deeper}\n" + step("second"),
		"procedures/deeper.yaml": "kind: procedure\nprocedure: deeper\ntargets: [local]\nsteps:\n" + step("deep"),
		"procedures/twice.yaml": "kind: procedure\nprocedure: twice\ntargets: [local]\nsteps:\n" +
			"  - {id: once, procedure: deeper}\n  - {id: again, procedure: deeper}\n",
		"procedures/absent.yaml": "kind: procedure\nprocedure: absent\ntargets: [local]\nsteps:\n" +
			"  - {id: nowhere, procedure: no-such-procedure}\n",
		"procedures/cyclic.yaml": "kind: procedure\nprocedure: cyclic\ntargets: [local]\nsteps:\n" +
			step("round") + "  - {id: again, procedure: cyclic}\n",
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := repository.Load(root)
	if err != nil {
		t.Fatalf("loading the fixture: %v", err)
	}
	return loaded
}
