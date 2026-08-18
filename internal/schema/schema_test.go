package schema

import (
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
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
		t.Fatalf("Check() = %+v, want no problems", got)
	}
}

func mustOne(t *testing.T, got []problem.Problem, code, field string) {
	t.Helper()
	if len(got) != 1 {
		t.Fatalf("Check() = %+v, want exactly one problem", got)
	}
	if got[0].ErrorCode != code {
		t.Errorf("ErrorCode = %q, want %q", got[0].ErrorCode, code)
	}
	if got[0].Field != field {
		t.Errorf("Field = %q, want %q", got[0].Field, field)
	}
}

// scalarPosition is the schema a lone scalar test reads a value against: an
// object with one required member, "value", of the type under test — the
// smallest position that exercises the reading rule without a whole
// artefact's schema around it.
func scalarPosition(t Type) Schema {
	return Schema{
		Type: Object,
		Properties: []Property{
			{Name: "value", Required: true, Schema: Schema{Type: t}},
		},
	}
}

func TestReadingRule_StringIgnoresQuoting(t *testing.T) {
	// The quoting YAML required is lexical rather than part of the value
	// (ADR-0081) — quoted and bare read as the same string.
	for _, doc := range []string{"value: hello", "value: \"hello\""} {
		mustNone(t, Check(parse(t, doc), scalarPosition(String), "f.yaml"))
	}
}

func TestReadingRule_Integer(t *testing.T) {
	// "2592000" and 2592000 are one value at an integer position; a
	// leading zero reads rather than refuses — "0755" reads as 755, ADR-0081's
	// consequence, not a defended feature.
	for _, doc := range []string{"value: 2592000", "value: \"2592000\"", "value: 0755"} {
		mustNone(t, Check(parse(t, doc), scalarPosition(Integer), "f.yaml"))
	}
}

func TestReadingRule_IntegerRejectsNonDigits(t *testing.T) {
	got := Check(parse(t, "value: 12.5"), scalarPosition(Integer), "f.yaml")
	mustOne(t, got, CodeMismatch, "value")
}

func TestReadingRule_Number(t *testing.T) {
	for _, doc := range []string{"value: 1", "value: 1.5", "value: \"1.5\"", "value: -2.25", "value: 1e10"} {
		mustNone(t, Check(parse(t, doc), scalarPosition(Number), "f.yaml"))
	}
	got := Check(parse(t, "value: not-a-number"), scalarPosition(Number), "f.yaml")
	mustOne(t, got, CodeMismatch, "value")
}

func TestReadingRule_BooleanTextIsExactlyTrueAndFalse(t *testing.T) {
	for _, doc := range []string{"value: true", "value: false"} {
		mustNone(t, Check(parse(t, doc), scalarPosition(Boolean), "f.yaml"))
	}
}

func TestReadingRule_BooleanRejectsCaseVariants(t *testing.T) {
	// §3 and §12 both fix boolean's text as exactly true and false and
	// nothing else — not a case-insensitive pair — so a capitalised spelling
	// reads as nothing at all, the same as NO does.
	for _, doc := range []string{"value: True", "value: TRUE", "value: False", "value: FALSE"} {
		got := Check(parse(t, doc), scalarPosition(Boolean), "f.yaml")
		mustOne(t, got, CodeMismatch, "value")
	}
}

func TestReadingRule_NOAtBooleanPositionReadsAsNothing(t *testing.T) {
	// The Norway problem stays dead by construction: NO is not one of
	// boolean's two words, so it reads as nothing at all.
	got := Check(parse(t, "value: \"NO\""), scalarPosition(Boolean), "f.yaml")
	mustOne(t, got, CodeMismatch, "value")

	got = Check(parse(t, "value: NO"), scalarPosition(Boolean), "f.yaml")
	mustOne(t, got, CodeMismatch, "value")
}

func TestReadingRule_DurationIsOneIntegerOneUnit(t *testing.T) {
	for _, doc := range []string{"value: 90d", "value: \"90d\"", "value: 1s", "value: 4h"} {
		mustNone(t, Check(parse(t, doc), scalarPosition(Duration), "f.yaml"))
	}
}

func TestReadingRule_DurationRejectsCompoundingAndForeignUnits(t *testing.T) {
	for _, doc := range []string{"value: 1d12h", "value: 2w", "value: 1mo", "value: 1y"} {
		got := Check(parse(t, doc), scalarPosition(Duration), "f.yaml")
		mustOne(t, got, CodeMismatch, "value")
	}
}

func TestReadingRule_TimestampRequiresUTCZ(t *testing.T) {
	mustNone(t, Check(parse(t, "value: \"2024-01-15T10:30:00Z\""), scalarPosition(Timestamp), "f.yaml"))
}

func TestReadingRule_TimestampOffsetFormIsRefused(t *testing.T) {
	got := Check(parse(t, "value: \"2024-01-15T10:30:00+02:00\""), scalarPosition(Timestamp), "f.yaml")
	mustOne(t, got, CodeMismatch, "value")
}

func TestReadingRule_Enum(t *testing.T) {
	s := scalarPosition(String)
	s.Properties[0].Schema.Enum = []string{"read", "mutate", "destroy"}

	mustNone(t, Check(parse(t, "value: mutate"), s, "f.yaml"))

	got := Check(parse(t, "value: delete"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "value")
}

func TestCheck_UnknownKey(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "kind", Required: true, Schema: Schema{Type: String}},
		},
	}
	got := Check(parse(t, "kind: repository-declaration\nextra: 1\n"), s, "f.yaml")
	mustOne(t, got, CodeUnknownKey, "extra")
}

func TestCheck_MissingRequiredKey(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "kind", Required: true, Schema: Schema{Type: String}},
			{Name: "digest", Required: true, Schema: Schema{Type: String}},
		},
	}
	got := Check(parse(t, "kind: repository-declaration\n"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "digest")
}

func TestCheck_OptionalKeyMayBeOmitted(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "kind", Required: true, Schema: Schema{Type: String}},
			{Name: "retention", Required: false, Schema: Schema{Type: Duration}},
		},
	}
	mustNone(t, Check(parse(t, "kind: repository-declaration\n"), s, "f.yaml"))
}

func TestCheck_NilRootReportsEveryRequiredKey(t *testing.T) {
	// An empty file is valid YAML — zero documents — and reads the same as
	// an object supplying none of its keys.
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "kind", Required: true, Schema: Schema{Type: String}},
			{Name: "version", Required: true, Schema: Schema{Type: String}},
			{Name: "digest", Required: true, Schema: Schema{Type: String}},
			{Name: "retention", Required: false, Schema: Schema{Type: Duration}},
		},
	}
	got := Check(nil, s, "f.yaml")
	if len(got) != 3 {
		t.Fatalf("Check(nil) = %+v, want 3 problems (one per required key)", got)
	}
	for _, p := range got {
		if p.ErrorCode != CodeMismatch {
			t.Errorf("ErrorCode = %q, want %q", p.ErrorCode, CodeMismatch)
		}
		if p.Line != 1 || p.Column != 1 {
			t.Errorf("position = %d:%d, want 1:1", p.Line, p.Column)
		}
	}
}

func TestCheck_OpenObjectAdmitsAnyKeys(t *testing.T) {
	// A mapping keyed by name — a Target declaration's auth: — is not
	// enumerated here; the generic engine only checks it is a mapping and
	// leaves its members to whatever reads a name-keyed mapping's own rule.
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "auth", Required: false, Schema: Schema{Type: Object, Open: true}},
		},
	}
	mustNone(t, Check(parse(t, "auth:\n  token: {env: TOKEN}\n  second: {env: OTHER}\n"), s, "f.yaml"))
}

func TestCheck_OpenObjectStillRejectsANonMapping(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "auth", Required: false, Schema: Schema{Type: Object, Open: true}},
		},
	}
	got := Check(parse(t, "auth: nope\n"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "auth")
}

func TestCheck_NonObjectRoot(t *testing.T) {
	s := Schema{Type: Object, Properties: []Property{{Name: "kind", Required: true, Schema: Schema{Type: String}}}}
	got := Check(parse(t, "- a\n- b\n"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "")
}

func TestCheck_NestedObjectAndArray(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "hosts", Required: true, Schema: Schema{Type: Array, Items: &Schema{Type: Integer}}},
			{Name: "auth", Required: false, Schema: Schema{
				Type: Object,
				Properties: []Property{
					{Name: "env", Required: true, Schema: Schema{Type: String}},
				},
			}},
		},
	}

	mustNone(t, Check(parse(t, "hosts: [1, 2]\nauth:\n  env: TOKEN\n"), s, "f.yaml"))

	got := Check(parse(t, "hosts: [1, nope]\nauth:\n  env: TOKEN\n"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "hosts[1]")

	got = Check(parse(t, "hosts: []\nauth:\n  env: TOKEN\n  bogus: 1\n"), s, "f.yaml")
	mustOne(t, got, CodeUnknownKey, "auth.bogus")
}

func TestCheck_FieldPathsAreDotAndBracketJoined(t *testing.T) {
	s := Schema{
		Type: Object,
		Properties: []Property{
			{Name: "steps", Required: true, Schema: Schema{Type: Array, Items: &Schema{
				Type: Object,
				Properties: []Property{
					{Name: "bound", Required: true, Schema: Schema{Type: Integer}},
				},
			}}},
		},
	}
	got := Check(parse(t, "steps:\n  - bound: nope\n"), s, "f.yaml")
	mustOne(t, got, CodeMismatch, "steps[0].bound")
}

// TestDurationSeconds_ReadsTheGrammarThePatternAdmits: the four units the
// authoring format closes the set at, converted; and false for everything the
// pattern refuses, which is what a surface rendering a duration in seconds
// reads as *there is no duration here* rather than as zero (§3, §9).
func TestDurationSeconds_ReadsTheGrammarThePatternAdmits(t *testing.T) {
	for value, want := range map[string]int{
		"30s": 30, "0s": 0, "2m": 120, "1h": 3600, "1d": 86400, "90m": 5400,
	} {
		got, authored := DurationSeconds(value)
		if !authored || got != want {
			t.Errorf("DurationSeconds(%q) = (%d, %v), want (%d, true)", value, got, authored, want)
		}
	}

	for _, value := range []string{"", "soon", "30", "s", "30 s", "-30s", "+30s", "1.5h", "30S", "1w"} {
		if got, authored := DurationSeconds(value); authored {
			t.Errorf("DurationSeconds(%q) = (%d, true); the grammar is digits and one of smhd", value, got)
		}
	}
}

// TestDurationSeconds_AgreesWithTheTypeThatRefusesTheValue is the one grammar
// read from both ends: a value this converts is a value a duration position
// admits, and a value it refuses is schema-mismatch there.
func TestDurationSeconds_AgreesWithTheTypeThatRefusesTheValue(t *testing.T) {
	for _, value := range []string{"30s", "0s", "1d", "soon", "30", "-1s", "1w"} {
		_, authored := DurationSeconds(value)
		if admitted := readsAs(Duration, value); authored != admitted {
			t.Errorf("DurationSeconds(%q) read it as %v and the duration type admits it as %v", value, authored, admitted)
		}
	}
}
