package schema_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// TestReadScalar is §12's scalar table read in both directions, which is what
// ADR-0081 states it is: the text column is what a value writes in a path, a
// query, a header or a command, and the body column is what it writes as JSON.
// One table, one reading, and a quote that is YAML's rather than the value's —
// so "2592000" and 2592000 are one value at an integer position.
func TestReadScalar(t *testing.T) {
	for _, c := range []struct {
		name      string
		declared  schema.Type
		value     string
		wantText  string
		wantJSON  string
		wantReads bool
	}{
		{"a string is itself at both sinks", schema.String, "ci-runner", "ci-runner", `"ci-runner"`, true},
		{"a string carrying digits stays a string", schema.String, "2592000", "2592000", `"2592000"`, true},
		{"a string is never HTML-escaped", schema.String, "a<b&c", "a<b&c", `"a<b&c"`, true},
		{"an integer is digits and a JSON number", schema.Integer, "2592000", "2592000", "2592000", true},
		{"a leading zero is not part of the value", schema.Integer, "0755", "755", "755", true},
		{"a negative integer keeps its sign", schema.Integer, "-12", "-12", "-12", true},
		{"an integer past a float64 is exact", schema.Integer, "123456789012345678901234567890", "123456789012345678901234567890", "123456789012345678901234567890", true},
		{"an integer position refuses a decimal point", schema.Integer, "1.5", "", "", false},
		{"an integer position refuses a word", schema.Integer, "thirty", "", "", false},
		{"a number is the shortest decimal that round-trips", schema.Number, "1.0", "1", "1", true},
		{"a number keeps its exponent form where it is shorter", schema.Number, "1e-7", "1e-7", "1e-7", true},
		{"a boolean is two words and a bare token", schema.Boolean, "true", "true", "true", true},
		{"a boolean position refuses NO", schema.Boolean, "NO", "", "", false},
		{"a boolean position refuses True", schema.Boolean, "True", "", "", false},
		{"a duration is byte-identical to what was authored", schema.Duration, "14d", "14d", `"14d"`, true},
		{"a duration position refuses a compound", schema.Duration, "1d12h", "", "", false},
		{"a timestamp is RFC 3339 in UTC", schema.Timestamp, "2026-04-02T09:41:14Z", "2026-04-02T09:41:14Z", `"2026-04-02T09:41:14Z"`, true},
		{"a timestamp position refuses an offset form", schema.Timestamp, "2026-04-02T09:41:14+02:00", "", "", false},
		{"an object reads as nothing at all", schema.Object, "{}", "", "", false},
		{"an array reads as nothing at all", schema.Array, "[]", "", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			read, reads := schema.ReadScalar(c.declared, c.value)
			if reads != c.wantReads {
				t.Fatalf("ReadScalar(%s, %q) read = %v, want %v", c.declared, c.value, reads, c.wantReads)
			}
			if !reads {
				return
			}
			if got := read.Text(); got != c.wantText {
				t.Errorf("Text() = %q, want %q", got, c.wantText)
			}
			if got := read.JSON(); got != c.wantJSON {
				t.Errorf("JSON() = %q, want %q", got, c.wantJSON)
			}
		})
	}
}
