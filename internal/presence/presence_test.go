package presence_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/presence"
)

// TestOf_ReadsTheThreeAndNothingElse is the whole of the package: os.LookupEnv's
// pair in, one of §12's three members out.
//
// The words are written as literals rather than as the constants they name, so
// that a rename reaching every call site and every golden would still fail here:
// what the CLI column prints and what the MCP tool's enum publishes are these
// strings, and a test spelling them through the constant would assert only that
// the constant equals itself (§9, §12, ADR-0145).
func TestOf_ReadsTheThreeAndNothingElse(t *testing.T) {
	for _, c := range []struct {
		what    string
		value   string
		present bool
		want    presence.Presence
	}{
		{"a variable nothing exported", "", false, "absent"},
		{"a variable exported to nothing", "", true, "empty"},
		{"a variable that is filled", "a-value-nothing-reads", true, "set"},
		{"a variable holding only a space", " ", true, "set"},
	} {
		if got := presence.Of(c.value, c.present); got != c.want {
			t.Errorf("%s: Of(%q, %v) = %q, want %q", c.what, c.value, c.present, got, c.want)
		}
	}
}

// TestOf_AValueIsNotJudgedPastItsLength is ADR-0007 stated as the experiment it
// forbids: `set` is *has characters* and never *looks like a credential*, so two
// values nothing would accept read alike and read like a real one.
//
// A whitespace-only variable is the case worth naming, because trimming it is
// the reading a well-meaning change would reach for first — and trimming is a
// judgement about a value's contents, which needs the value and belongs to the
// endpoint.
func TestOf_AValueIsNotJudgedPastItsLength(t *testing.T) {
	for _, value := range []string{" ", "\t\n", "0", "null", "undefined", "x"} {
		if got := presence.Of(value, true); got != presence.Set {
			t.Errorf("Of(%q, true) = %q, want %q; the one thing read off a value is whether it has any characters", value, got, presence.Set)
		}
	}
}
