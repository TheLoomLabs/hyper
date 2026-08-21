package store_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The schema test §6 runs at Run start (issue #137). What is asserted here is
// its **scope** and its answer: which files it opens, which it does not, and
// what it says about one it cannot read.

// thePair is the (Definition, Target) pair theSeries sits under, which is what
// a Procedure whose Step names that Definition and that Target makes.
var thePair = store.Pair{Target: theSeries.Target, Definition: theSeries.Definition}

// bumped is one Store file rewritten at a schema version above every ceiling
// this binary knows. It is a textual substitution over a file the encoder wrote
// rather than a file typed out here, so the case stays canonical in every
// respect but the one it is about — a hand-written file would fail the decode
// before the version was ever compared.
func bumped(content string) string {
	return strings.Replace(content, `"schema_version": 1`, `"schema_version": 2`, 1)
}

// TestReadable_ClearsAStoreEveryFileOfWhichThisBinaryReads is the ordinary
// answer: a Journal and a Record series written by this binary clear the gate,
// and the Run goes on to its first Step.
func TestReadable_ClearsAStoreEveryFileOfWhichThisBinaryReads(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	unreadable, found, err := held.Readable([]store.Pair{thePair})
	if err != nil {
		t.Fatalf("Readable: %v", err)
	}
	if found {
		t.Errorf("the gate declined on %s, and every file on the branch was written by this binary", unreadable.File)
	}
}

// TestReadable_DeclinesOnARecordHeadUnderAPairTheProcedureMakes is the
// Refusal's own path: a Record file above this binary's ceiling, under a pair
// the Procedure binds, named by the path the Refusal cites.
func TestReadable_DeclinesOnARecordHeadUnderAPairTheProcedureMakes(t *testing.T) {
	version := aVersion(t, theSeries, theEntryRunID, 1, theInstant)
	r, _ := seededStore(t)
	r.seedFiles(r.root, map[string]string{pathOf(version): bumped(string(version.Encode()))})
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	unreadable, found, err := held.Readable([]store.Pair{thePair})
	if err != nil {
		t.Fatalf("Readable: %v", err)
	}
	if !found {
		t.Fatal("the gate cleared a Record version written at schema version 2, and this binary reads 1")
	}
	if unreadable.File != pathOf(version) {
		t.Errorf("the Refusal cites %q, want %q — §8 states this code cites a Store file", unreadable.File, pathOf(version))
	}
	if unreadable.Written != 2 || unreadable.Known != store.RecordSchemaVersion {
		t.Errorf("the condition reads written %d known %d, want 2 and %d", unreadable.Written, unreadable.Known, store.RecordSchemaVersion)
	}
}

// TestReadable_ReadsNoRecordOutsideThePairsTheProcedureMakes is §6's scope
// stated as what it does not open: a Target serving two Providers does not
// oblige a Run to read a series no Step of it could reach, exactly as it does
// not oblige it to hold a credential no Step could send.
func TestReadable_ReadsNoRecordOutsideThePairsTheProcedureMakes(t *testing.T) {
	elsewhere := store.Identity{Target: "staging", Definition: "other-definition", Name: "x"}
	version := aVersion(t, elsewhere, theEntryRunID, 1, theInstant)
	r, _ := seededStore(t)
	r.seedFiles(r.root, map[string]string{pathOf(version): bumped(string(version.Encode()))})
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	unreadable, found, err := held.Readable([]store.Pair{thePair})
	if err != nil {
		t.Fatalf("Readable: %v", err)
	}
	if found {
		t.Errorf("the gate declined on %s, which sits under a pair this Procedure never binds", unreadable.File)
	}
}

// TestReadable_ReadsTheJournalWhole is the other half of §6's sentence: the
// Journal is in scope in full, with no pair to narrow it by, because a Run
// reads back the evidence of Steps under any binding it has ever had.
func TestReadable_ReadsTheJournalWhole(t *testing.T) {
	entry := anEntry{run: runFileAt(t, theEntryRunID, theInstant)}
	r, _ := seededStore(t)
	files := map[string]string{}
	for path, content := range entry.files(t) {
		files[path] = bumped(content)
	}
	r.seedFiles(r.root, files)
	held, err := store.Open(r.root, theInstant)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	unreadable, found, err := held.Readable(nil)
	if err != nil {
		t.Fatalf("Readable: %v", err)
	}
	if !found {
		t.Fatal("the gate cleared a run.json written at schema version 2, and a Run reads the Journal whole")
	}
	if !strings.HasPrefix(unreadable.File, "journal/") {
		t.Errorf("the Refusal cites %q, want a Journal path", unreadable.File)
	}
}
