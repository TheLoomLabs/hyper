package artefact

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

func parse(t *testing.T, doc string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &n); err != nil {
		t.Fatalf("Unmarshal(%q) = %v", doc, err)
	}
	if len(n.Content) == 0 {
		return nil
	}
	return n.Content[0]
}

func mustNone(t *testing.T, got []problem.Problem) {
	t.Helper()
	if len(got) != 0 {
		t.Fatalf("CheckRepositoryDeclaration() = %+v, want no problems", got)
	}
}

func mustCode(t *testing.T, got []problem.Problem, code string) problem.Problem {
	t.Helper()
	for _, p := range got {
		if p.ErrorCode == code {
			return p
		}
	}
	t.Fatalf("CheckRepositoryDeclaration() = %+v, want a %s problem", got, code)
	return problem.Problem{}
}

const clean = "kind: repository-declaration\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"

func TestCheckRepositoryDeclaration_Clean(t *testing.T) {
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, clean)))
}

func TestCheckRepositoryDeclaration_RetentionIsOptional(t *testing.T) {
	doc := clean + "retention: 90d\n"
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
}

func TestCheckRepositoryDeclaration_KindMismatch(t *testing.T) {
	doc := "kind: definition\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, CodeKindMismatch)
	if p.Field != "kind" {
		t.Errorf("Field = %q, want kind", p.Field)
	}
}

func TestCheckRepositoryDeclaration_UnknownKey(t *testing.T) {
	doc := clean + "name: my-repo\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeUnknownKey)
	if p.Field != "name" {
		t.Errorf("Field = %q, want name", p.Field)
	}
}

func TestCheckRepositoryDeclaration_MissingDigestIsSchemaMismatch(t *testing.T) {
	doc := "kind: repository-declaration\nversion: 1.4.0\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "digest" {
		t.Errorf("Field = %q, want digest", p.Field)
	}
}

func TestCheckRepositoryDeclaration_BadDurationIsSchemaMismatch(t *testing.T) {
	doc := clean + "retention: 1d12h\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	p := mustCode(t, got, schema.CodeMismatch)
	if p.Field != "retention" {
		t.Errorf("Field = %q, want retention", p.Field)
	}
}

func TestCheckRepositoryDeclaration_RetentionQuotedOrBareIsOneValue(t *testing.T) {
	for _, doc := range []string{clean + "retention: 90d\n", clean + "retention: \"90d\"\n"} {
		mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
	}
}

func TestCheckRepositoryDeclaration_KeyOrderIsFree(t *testing.T) {
	// Key order within a file is free and nothing checks it (§3).
	doc := "digest: sha256:0000000000000000000000000000000000000000000000000000000000000000\nversion: 1.4.0\nkind: repository-declaration\n"
	mustNone(t, CheckRepositoryDeclaration("hyper.yaml", parse(t, doc)))
}

func TestCheckRepositoryDeclaration_EmptyFileReportsThreeMissingKeys(t *testing.T) {
	got := CheckRepositoryDeclaration("hyper.yaml", nil)
	if len(got) != 3 {
		t.Fatalf("CheckRepositoryDeclaration(nil) = %+v, want 3 problems", got)
	}
	for _, p := range got {
		if p.ErrorCode != schema.CodeMismatch {
			t.Errorf("ErrorCode = %q, want %q", p.ErrorCode, schema.CodeMismatch)
		}
	}
}

func TestCheckRepositoryDeclaration_KindMismatchDoesNotFireWhenKindItselfFailsSchema(t *testing.T) {
	// kind: written as a mapping fails the schema (schema-mismatch); the
	// kind check has nothing legible to compare and says nothing more about
	// the same line.
	doc := "kind:\n  nested: true\nversion: 1.4.0\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n"
	got := CheckRepositoryDeclaration("hyper.yaml", parse(t, doc))
	for _, p := range got {
		if p.ErrorCode == CodeKindMismatch {
			t.Fatalf("CheckRepositoryDeclaration() = %+v, want no kind-mismatch alongside schema-mismatch", got)
		}
	}
	mustCode(t, got, schema.CodeMismatch)
}
