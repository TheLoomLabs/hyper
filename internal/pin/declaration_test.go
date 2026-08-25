package pin_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/pin"
)

const frozen = "sha256:9f2c1b7a4e6d038c5b1f92a7de40cb83f5710e2d9a6c4b83fe012d75c9a4e6b1"

// TestWritten_EditsTwoScalarsAndNothingElse is §11's own sentence about
// `hyper.yaml`: it is edited, never regenerated. `retention:` is authored, the
// comments and the layout are the author's, and what `project` writes is two
// derived facts into a reviewed artefact.
func TestWritten_EditsTwoScalarsAndNothingElse(t *testing.T) {
	authored := `# our repository's own declaration — read the diff when this moves
kind: repository-declaration
version: 1.3.0  # bumped by hyper project
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000

# ninety days of interior versions, agreed in RFC-14
retention: 90d
`
	want := `# our repository's own declaration — read the diff when this moves
kind: repository-declaration
version: 1.4.0  # bumped by hyper project
digest: ` + frozen + `

# ninety days of interior versions, agreed in RFC-14
retention: 90d
`

	if got := string(pin.Written([]byte(authored), true, "1.4.0", frozen)); got != want {
		t.Errorf("Written() wrote\n%s\nwant\n%s", got, want)
	}
}

// TestWritten_LeavesTheQuotingItFound is the same rule one level down: a
// scalar's spelling is the author's too, so a pin they quoted stays quoted.
func TestWritten_LeavesTheQuotingItFound(t *testing.T) {
	authored := "kind: repository-declaration\nversion: \"1.3.0\"\ndigest: 'sha256:0000'\n"
	want := "kind: repository-declaration\nversion: \"1.4.0\"\ndigest: '" + frozen + "'\n"

	if got := string(pin.Written([]byte(authored), true, "1.4.0", frozen)); got != want {
		t.Errorf("Written() wrote %q, want %q", got, want)
	}
}

// TestWritten_ARepositoryWithNoDeclarationGetsOneAndNoRetention is the
// other half: a repository that has not stated a policy has not agreed to lose
// anything, and `project` does not author one on its behalf (§3, §11).
func TestWritten_ARepositoryWithNoDeclarationGetsOneAndNoRetention(t *testing.T) {
	written := string(pin.Written(nil, false, "1.4.0", frozen))

	want := "kind: repository-declaration\nversion: 1.4.0\ndigest: " + frozen + "\n"
	if written != want {
		t.Errorf("Written() created %q, want %q", written, want)
	}
	if strings.Contains(written, "retention") {
		t.Errorf("Written() created %q, want no retention: at all", written)
	}
}

// TestWritten_WritingWhatIsAlreadyThereChangesNoByte is what makes
// re-projection a no-op on the file every command reads: the two facts are
// already the two facts, so the diff is empty rather than whitespace.
func TestWritten_WritingWhatIsAlreadyThereChangesNoByte(t *testing.T) {
	authored := "kind: repository-declaration\nversion: 1.4.0\ndigest: " + frozen + "\nretention: 90d\n"

	if got := string(pin.Written([]byte(authored), true, "1.4.0", frozen)); got != authored {
		t.Errorf("Written() wrote %q, want the bytes it was handed, %q", got, authored)
	}
}

// TestDeclared_IsThePinTheBytesCarry is what decides whether anything is
// fetched at all: a pin equal to the binary's version resolves nothing (§11).
func TestDeclared_IsThePinTheBytesCarry(t *testing.T) {
	for _, c := range []struct {
		name, data, want string
	}{
		{"a pin", "kind: repository-declaration\nversion: 1.4.0\n", "1.4.0"},
		{"no version key", "kind: repository-declaration\n", ""},
		{"nothing that parses", "kind: [\n", ""},
		{"no file at all", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := pin.Declared([]byte(c.data)); got != c.want {
				t.Errorf("Declared(%q) = %q, want %q", c.data, got, c.want)
			}
		})
	}
}

// TestWritten_AKeyTheFileDoesNotCarryIsWritten is the answer to a shape
// `project` cannot reach: `hyper.yaml`'s schema makes both keys required, and
// `project` writes nothing where `check` would report anything — so a
// declaration missing one has already been Refused by the time this could see
// it. What is asserted is that the function still answers a declaration carrying
// both facts rather than silently dropping one (§4, ADR-0064).
func TestWritten_AKeyTheFileDoesNotCarryIsWritten(t *testing.T) {
	written := string(pin.Written([]byte("kind: repository-declaration\n"), true, "1.4.0", frozen))

	want := "kind: repository-declaration\nversion: 1.4.0\ndigest: " + frozen + "\n"
	if written != want {
		t.Errorf("Written() wrote %q, want %q", written, want)
	}
}

// TestWritten_AKeyWithNoScalarSpanIsLeftAlone is the other unreachable shape,
// and it is left alone rather than repaired: what is wrong with a `version:`
// carrying a mapping, or carrying nothing at all, is a `schema-mismatch` a
// reader is owed rather than a spelling this may invent for them.
func TestWritten_AKeyWithNoScalarSpanIsLeftAlone(t *testing.T) {
	for _, authored := range []string{
		"kind: repository-declaration\nversion:\n  major: 1\ndigest: " + frozen + "\n",
		"kind: repository-declaration\nversion:\ndigest: " + frozen + "\n",
	} {
		if got := string(pin.Written([]byte(authored), true, "1.4.0", frozen)); got != authored {
			t.Errorf("Written() wrote %q, want the bytes it was handed, %q", got, authored)
		}
	}
}

// TestWritten_AValueOfADifferentLengthMovesOnlyItself is the span arithmetic
// held to what it is for. Every value `project` replaces in practice is the same
// length as the one it replaces — a version for a version, sixty-four hex digits
// for sixty-four — so the case that says the rest of the line survives a value
// that grew or shrank has to be written rather than waited for.
func TestWritten_AValueOfADifferentLengthMovesOnlyItself(t *testing.T) {
	authored := "kind: repository-declaration\nversion: 1.10.0-rc.1  # the candidate we are on\ndigest: sha256:00\n"
	want := "kind: repository-declaration\nversion: 2.0  # the candidate we are on\ndigest: " + frozen + "\n"

	if got := string(pin.Written([]byte(authored), true, "2.0", frozen)); got != want {
		t.Errorf("Written() wrote %q, want %q", got, want)
	}
}
