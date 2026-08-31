package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// §9's one conditional narration line, held to the fact it is about (§7,
// ADR-0119, issue #239).

// TestRunNarration_TheWarningAndTheMarkerAreOneFact is what keeps the line
// honest: it says the code this Run is performing is in no commit, and the
// entry the same Run writes marks exactly that with `repo_dirty`. Two readings
// of one read of the code branch, so a case narrating one without the other
// would be a warning about a state the record does not agree it was in.
//
// It is asserted over the corpus's own files rather than by driving a Run,
// because both halves are already checked in: the harness holds `stderr.golden`
// and `store.golden` byte for byte, and what has no other home is that the two
// move together.
func TestRunNarration_TheWarningAndTheMarkerAreOneFact(t *testing.T) {
	// One case per answer, named rather than enumerated: what a corpus-wide
	// walk would add is every case that seeds a Journal it did not write,
	// where the marker on the branch is an entry from before this Run.
	for _, name := range []string{
		"an-untracked-artefact-is-dirty",
		"a-working-tree-that-moved",
		"a-working-tree-edited-since-check-passed",
		"the-tracer-bullet",
		"a-run-on-a-runner",
	} {
		dir := filepath.Join("testdata", "run", name)
		marked := strings.Contains(readGolden(t, dir, "store.golden"), `"repo_dirty": true`)
		warned := strings.Contains(readGolden(t, dir, "stderr.golden"), uncommittedNarration)
		if marked != warned {
			t.Errorf("%s marks repo_dirty=%t and narrates the warning=%t; the two are one read of the code branch", name, marked, warned)
		}
	}
}

// readGolden is one of a case's checked-in files, read whole. The case is
// named rather than enumerated and the file is read rather than driven: what is
// asserted here is that two files a harness already checks byte for byte agree
// with each other, so re-running the case would be a third opinion about
// something already held twice.
func readGolden(t *testing.T, dir, name string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("%s: %v", filepath.Join(dir, name), err)
	}
	return string(content)
}
