package yamlsubset

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

func TestScan_CleanFileHasNoProblems(t *testing.T) {
	data := []byte("kind: definition\ndefinition: uptime\n")
	got := Scan("definitions/uptime.yaml", data)
	if len(got) != 0 {
		t.Fatalf("Scan() = %v, want no problems", got)
	}
}

func TestScan_Anchor(t *testing.T) {
	data := []byte("base: &b\n  x: 1\nother: 2\n")
	got := Scan("definitions/x.yaml", data)
	mustOneViolation(t, got, 1, 7, "base")
}

func TestScan_Alias(t *testing.T) {
	data := []byte("base: &b\n  x: 1\nderived: *b\n")
	got := Scan("definitions/x.yaml", data)
	if len(got) != 2 {
		t.Fatalf("Scan() = %v, want 2 problems (anchor + alias)", got)
	}
	found := false
	for _, p := range got {
		if p.ErrorCode != ErrorCode {
			t.Errorf("ErrorCode = %q, want %q", p.ErrorCode, ErrorCode)
		}
		if p.Field == "derived" && p.Line == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("Scan() = %v, want a problem at derived (line 3)", got)
	}
}

func TestScan_MergeKey(t *testing.T) {
	data := []byte("base: &b\n  x: 1\nderived:\n  <<: *b\n  y: 2\n")
	got := Scan("definitions/x.yaml", data)
	var sawMerge bool
	for _, p := range got {
		if p.Field == "derived.<<" {
			sawMerge = true
		}
	}
	if !sawMerge {
		t.Fatalf("Scan() = %v, want a problem at derived.<<", got)
	}
}

func TestScan_MergeKeyWithoutAlias(t *testing.T) {
	data := []byte("derived:\n  <<:\n    z: 3\n  y: 2\n")
	got := Scan("definitions/x.yaml", data)
	mustOneViolation(t, got, 2, 3, "derived.<<")
}

func TestScan_ExplicitTag(t *testing.T) {
	data := []byte("value: !!str 123\n")
	got := Scan("definitions/x.yaml", data)
	mustOneViolation(t, got, 1, 8, "value")
}

func TestScan_MultiDocument(t *testing.T) {
	data := []byte("kind: definition\n---\nkind: procedure\n")
	got := Scan("definitions/x.yaml", data)
	var sawMultiDoc bool
	for _, p := range got {
		if p.Line == 2 && p.Field == "" {
			sawMultiDoc = true
		}
	}
	if !sawMultiDoc {
		t.Fatalf("Scan() = %v, want a multi-document problem at line 2", got)
	}
}

func TestScan_ImplicitNull_EmptyValue(t *testing.T) {
	data := []byte("retention:\nkind: definition\n")
	got := Scan("hyper.yaml", data)
	mustOneViolation(t, got, 1, 11, "retention")
}

func TestScan_ImplicitNull_Tilde(t *testing.T) {
	data := []byte("retention: ~\n")
	got := Scan("hyper.yaml", data)
	mustOneViolation(t, got, 1, 12, "retention")
}

func TestScan_ImplicitNull_LiteralNull(t *testing.T) {
	data := []byte("retention: null\n")
	got := Scan("hyper.yaml", data)
	mustOneViolation(t, got, 1, 12, "retention")
}

func TestScan_QuotedNullIsNotAViolation(t *testing.T) {
	data := []byte("retention: \"null\"\n")
	got := Scan("hyper.yaml", data)
	if len(got) != 0 {
		t.Fatalf("Scan() = %v, want no problems for a quoted \"null\"", got)
	}
}

func TestScan_PlainScalarsAreNotViolations(t *testing.T) {
	// Bare integers, booleans and durations are read against the schema at
	// their position (§3, ADR-0081) — a later ticket's rule, not this
	// package's. This package must not reject them.
	data := []byte("bound: 5\nconcurrency: 4\nreusable: false\nretention: 90d\nversion: 1.4.0\n")
	got := Scan("hyper.yaml", data)
	if len(got) != 0 {
		t.Fatalf("Scan() = %v, want no problems for plain scalars", got)
	}
}

func TestScan_SequenceIndexInField(t *testing.T) {
	data := []byte("steps:\n  - id: probe\n  - id: label\n    bound: &b 5\n")
	got := Scan("procedures/x.yaml", data)
	mustOneViolation(t, got, 4, 12, "steps[1].bound")
}

func TestScan_UnparseableFileYieldsExactlyOneProblem(t *testing.T) {
	data := []byte("a: [1, 2\n")
	got := Scan("definitions/broken.yaml", data)
	if len(got) != 1 {
		t.Fatalf("Scan() = %v, want exactly one problem", got)
	}
	if got[0].ErrorCode != ErrorCode {
		t.Errorf("ErrorCode = %q, want %q", got[0].ErrorCode, ErrorCode)
	}
	if got[0].File != "definitions/broken.yaml" {
		t.Errorf("File = %q, want definitions/broken.yaml", got[0].File)
	}
}

func TestScan_EmptyFileHasNoProblems(t *testing.T) {
	got := Scan("hyper.yaml", []byte(""))
	if len(got) != 0 {
		t.Fatalf("Scan() = %v, want no problems for an empty file", got)
	}
}

func TestParse_UnparseableFileIsNotOK(t *testing.T) {
	root, problems, ok := Parse("definitions/broken.yaml", []byte("a: [1, 2\n"))
	if ok {
		t.Fatalf("Parse() ok = true, want false for a hard syntax error")
	}
	if root != nil {
		t.Errorf("Parse() root = %v, want nil", root)
	}
	if len(problems) != 1 || problems[0].ErrorCode != ErrorCode {
		t.Fatalf("Parse() problems = %v, want exactly one strict-yaml-violation", problems)
	}
}

func TestParse_EmptyFileIsOKWithNilRoot(t *testing.T) {
	root, problems, ok := Parse("hyper.yaml", []byte(""))
	if !ok {
		t.Fatalf("Parse() ok = false, want true for an empty file")
	}
	if root != nil {
		t.Errorf("Parse() root = %v, want nil", root)
	}
	if len(problems) != 0 {
		t.Errorf("Parse() problems = %v, want none", problems)
	}
}

func TestParse_ReturnsTheDecodedRoot(t *testing.T) {
	root, problems, ok := Parse("hyper.yaml", []byte("kind: repository-declaration\n"))
	if !ok || root == nil {
		t.Fatalf("Parse() = %v, %v, %v, want a root node and ok", root, problems, ok)
	}
	if root.Kind != yaml.MappingNode {
		t.Errorf("Parse() root.Kind = %v, want a mapping", root.Kind)
	}
}

func TestParse_MultiDocumentIsOKWithTheFirstDocumentsRoot(t *testing.T) {
	root, problems, ok := Parse("hyper.yaml", []byte("kind: repository-declaration\n---\nkind: definition\n"))
	if !ok {
		t.Fatalf("Parse() ok = false, want true")
	}
	if root == nil || root.Kind != yaml.MappingNode {
		t.Fatalf("Parse() root = %v, want the first document's mapping", root)
	}
	if len(problems) != 1 || problems[0].Line != 2 {
		t.Fatalf("Parse() problems = %v, want one problem at line 2", problems)
	}
}

func TestViolations_MatchesWhatScanWalksFor(t *testing.T) {
	data := []byte("base: &b\n  x: 1\nderived: *b\n")
	root, parseProblems, ok := Parse("definitions/x.yaml", data)
	if !ok {
		t.Fatal("Parse() ok = false")
	}
	got := append(append([]problem.Problem{}, parseProblems...), Violations(root, "definitions/x.yaml")...)
	want := Scan("definitions/x.yaml", data)
	if len(got) != len(want) {
		t.Fatalf("Parse+Violations = %v, want the same problems Scan finds: %v", got, want)
	}
}

func mustOneViolation(t *testing.T, got []problem.Problem, line, column int, field string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Scan() = %v, want exactly one problem", got)
	}
	p := got[0]
	if p.Line != line || p.Column != column {
		t.Errorf("position = %d:%d, want %d:%d", p.Line, p.Column, line, column)
	}
	if p.Field != field {
		t.Errorf("Field = %q, want %q", p.Field, field)
	}
	if p.ErrorCode != ErrorCode {
		t.Errorf("ErrorCode = %q, want %q", p.ErrorCode, ErrorCode)
	}
}
