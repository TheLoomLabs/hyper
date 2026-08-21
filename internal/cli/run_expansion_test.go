package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **An identity set is written as a digest, and its members only where that
// digest moved** (§7, ADR-0030, ADR-0055, issue #139).
//
// No golden can hold this either, and for a different reason from the push
// beside it: what it is about is what the *second* entry writes given the
// first, and a case drives one Run. The claim needs four — a set that lands, a
// set that did not move, a set that moved, and a set that did not move again —
// over one branch, so the arithmetic a reader does off the entry is the one
// this asserts.
//
// It drives the corpus case [testdata/run/four-runs-of-one-step] four times
// through one materialised repository, editing the Step's `values:` list
// between the second Run and the third. That edit is the ordinary occasion for
// a set to move: an Expansion is what the Step concluded about, so narrowing
// the population narrows the set, and nothing else in the artefact or the world
// has to change for it to.

// TestExpansion_TheIdentitySetIsWrittenWhereItMoved drives the four Runs and
// reads what each entry wrote.
func TestExpansion_TheIdentitySetIsWrittenWhereItMoved(t *testing.T) {
	dir := filepath.Join("testdata", "run", "four-runs-of-one-step")
	c := goldenCase{dir: dir, name: "run/four-runs-of-one-step", argv: readArgv(t, filepath.Join(dir, "argv"))}
	invocation := c.invocation(t)
	// One process for the four Runs, so that the mint answers the four ids
	// the case names in the order it names them — a fresh one per Run would
	// answer the first id four times and write one entry over another.
	process := c.process(t, invocation)

	narrowed := filepath.Join(invocation.fixture.root, "procedures", "watch-values.yaml")
	authored, err := os.ReadFile(narrowed)
	if err != nil {
		t.Fatal(err)
	}

	for run, expected := range []struct {
		members []string
		moved   bool
	}{
		{members: []string{"cert.hyper.dev", "status.hyper.dev"}, moved: true},
		{members: []string{"cert.hyper.dev", "status.hyper.dev"}, moved: false},
		{members: []string{"status.hyper.dev"}, moved: true},
		{members: []string{"status.hyper.dev"}, moved: false},
	} {
		if run == 2 {
			// The edit: one member of the two, which is what moves
			// the set. It is written into the working tree and not
			// committed, so the Runs past it record repo_dirty —
			// which is the truth about them and is asserted nowhere
			// here.
			edited := strings.Replace(string(authored),
				"values: [status.hyper.dev, cert.hyper.dev]", "values: [status.hyper.dev]", 1)
			if edited == string(authored) {
				t.Fatalf("the fixture's values: list is not the one this test narrows:\n%s", authored)
			}
			if err := os.WriteFile(narrowed, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		var stdout, stderr bytes.Buffer
		if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
			t.Fatalf("run %d: exit = %d; stderr: %s", run+1, exit, stderr.String())
		}

		file := stepFileOf(t, c, invocation, run+1)
		if file.Identities.Digest != store.IdentityDigest(expected.members) {
			t.Errorf("run %d carries digest %s over %v", run+1, file.Identities.Digest, expected.members)
		}
		switch {
		case expected.moved && !equalNames(file.Identities.Members, expected.members):
			t.Errorf("run %d writes members %v, want %v in full — the digest moved", run+1, file.Identities.Members, expected.members)
		case !expected.moved && file.Identities.Members != nil:
			t.Errorf("run %d writes members %v; the set did not move, so the digest stands alone", run+1, file.Identities.Members)
		}
		// The set is sorted wherever it is written, which is what makes
		// the digest a fact about the set rather than about the order a
		// response happened to arrive in — where `expanded_to` beside it
		// is the account of a sequence (§6, §7).
		if names := file.Selector.ExpandedTo; run < 2 && !equalNames(names, []string{"status.hyper.dev", "cert.hyper.dev"}) {
			t.Errorf("run %d expanded to %v, want the authored order", run+1, names)
		}
	}
}

// stepFileOf reads Step 1's file out of the entry the nth Run of the fixture
// wrote, naming the Run by the id the case's mint file answered for it.
func stepFileOf(t *testing.T, c goldenCase, invocation goldenRun, nth int) store.StepFile {
	t.Helper()

	ids := strings.Fields(readFile(t, filepath.Join(c.dir, "mint")))
	if nth > len(ids) {
		t.Fatalf("the case names %d Run ids and this is Run %d", len(ids), nth)
	}
	id, err := store.ParseRunID(ids[nth-1])
	if err != nil {
		t.Fatal(err)
	}

	entry := store.JournalEntry{Run: id, Started: c.instant(t)}
	branch := invocation.fixture.render(t, invocation.fixture.root)
	content, held := blobOf(branch, entry.StepPath(1))
	if !held {
		t.Fatalf("run %d wrote no %s; the branch holds:\n%s", nth, entry.StepPath(1), branch)
	}
	file, err := store.DecodeStepFile([]byte(content))
	if err != nil {
		t.Fatalf("run %d's Step file: %v", nth, err)
	}
	return file
}

// blobOf reads one file out of a rendered branch, which is the shape the branch
// goldens are written in: a `=== <path> (<n> bytes)` header per file, and its
// content beneath.
func blobOf(branch, path string) (string, bool) {
	blocks := strings.Split(branch, "=== ")
	for _, block := range blocks {
		header, content, split := strings.Cut(block, "\n")
		if split && strings.HasPrefix(header, path+" (") {
			return content, true
		}
	}
	return "", false
}

// equalNames compares two name lists element by element, the empty list and nil
// being two answers everywhere the Store writes one (§7).
func equalNames(held, want []string) bool {
	if len(held) != len(want) {
		return false
	}
	for i := range held {
		if held[i] != want[i] {
			return false
		}
	}
	return true
}
