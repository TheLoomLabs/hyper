package cli_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// **A `destroy` by literal identifier, and the two absences that are not one**
// (§5, §6, §7, ADR-0033, ADR-0084, issue #151).
//
// The corpus holds what a branch and a page can hold: the Tombstone that opens
// a series, the member the Store dropped, the member a `mutate` reached, and
// the Refusal a misspelt literal earns. What it cannot hold is a claim about
// two files *held against each other* — that the Tombstone opening a series
// differs from an ordinary one in the `fields` it does not carry and in nothing
// else, and that an ordinary version carrying no `fields` is not read as a
// Tombstone for it.
//
// Both are claims about **absence**, which is exactly what a golden asserts
// least well: a key that is not there looks the same in a golden as a key
// nobody thought to write. So they are driven here, off the same corpus cases,
// and the second is held against a case from another milestone entirely — the
// `shell` command that could not be started at all, which is where §7's other
// fieldless version comes from (issue #142).

// The two cases this file reads past their goldens: one `destroy` over a
// `values:` list the Store held nothing for, and one over a list whose second
// member names a series a previous Run created.
const (
	openedSeries   = "a-destroy-by-literal-opens-the-series-it-ends"
	standingSeries = "a-literal-that-matches-a-standing-series"
)

// TestLiteralDestroy_NoSecondMarkerTellsTheTwoTombstonesApart is ADR-0033's
// *no branch on whether a series already exists*, read off one Run that took
// both routes at once.
//
// The case destroys two literals: one naming a series the Store never held, and
// one naming a series a previous Run created. Both Tombstones are written by
// one Step of one Run under one clock, so every member either file carries is a
// member the other carries — the identity's own `name` apart — unless something
// in the version says which route reached it. What may differ is the `fields`:
// the opened series has no previous Head to copy forward, and the standing one
// has. What may not differ is anything else, `tombstone: true` being on the
// version already and a reader asking that key rather than a second one.
func TestLiteralDestroy_NoSecondMarkerTellsTheTwoTombstonesApart(t *testing.T) {
	branch := branchOf(t, standingSeries)

	const run = "01991f10-b118-7c93-8d41-6b2f7ae05e02"
	opened := fileIn(t, branch, "records/cloudflare-prod/preview-dns/preview-41.example.com/"+run+"-0001.json")
	further := fileIn(t, branch, "records/cloudflare-prod/preview-dns/preview-42.example.com/"+run+"-0001.json")

	if strings.Contains(opened, `"fields"`) {
		t.Errorf("the Tombstone opening a series carries fields, and there was no previous Head to copy forward:\n%s", opened)
	}
	if !strings.Contains(further, `"fields"`) {
		t.Errorf("the Tombstone over a standing Asset carries no fields, and its previous Head held some:\n%s", further)
	}
	for _, held := range []string{opened, further} {
		if !strings.Contains(held, `"tombstone": true`) {
			t.Errorf("a version this case is about is not a Tombstone:\n%s", held)
		}
	}

	// And nothing else moves. `name` is the identity's own and `fields` is
	// the last known state; a third member differing would be a second
	// representation of a fact `tombstone` already carries.
	if differs := membersDiffering(t, opened, further); !slices.Equal(differs, []string{"fields", "name"}) {
		t.Errorf("the two Tombstones differ in %v, want [fields name] — no second marker says which route reached one", differs)
	}
}

// TestLiteralDestroy_AnOrdinaryVersionWithNoFieldsIsNotATombstone holds §7's
// two absences apart at the one place they meet: the reader.
//
// A Tombstone opening a series carries no `fields` because there was no
// previous Head to copy forward. An ordinary version carries none because every
// path its Manifest projected resolved to nothing — a `shell` command that
// could not be started at all being where the corpus already has one (ADR-0084,
// issue #142). The two files look alike in the one respect this is about, and
// what tells them apart is the written marker and never the missing key, so
// what this reads is the decode: what the Store answers when asked what each
// version is.
func TestLiteralDestroy_AnOrdinaryVersionWithNoFieldsIsNotATombstone(t *testing.T) {
	ended := versionAt(t, branchOf(t, openedSeries),
		"records/cloudflare-prod/preview-dns/preview-41.example.com/01991f10-b118-7c93-8d41-6b2f7ae05e01-0001.json")
	quiet := versionAt(t, branchOf(t, "a-command-that-could-not-be-started"),
		"records/local/host-checks/%5B%22disk-free%22%2C%22%2Fsrv%22%5D/01991ea6-b118-7c93-8d41-6b2f7ae05d03-0001.json")

	if ended.Fields != nil || quiet.Fields != nil {
		t.Fatalf("this is about two versions carrying no fields, and one of them carries some: %v and %v", ended.Fields, quiet.Fields)
	}
	if !ended.Tombstone {
		t.Error("the version a destroy by literal identifier wrote does not read as a Tombstone")
	}
	if quiet.Tombstone {
		t.Error("a version whose every projected path resolved to nothing reads as a Tombstone — the marker and not the missing key is what identifies one")
	}
	if ended.RecordType != store.RecordAsset || quiet.RecordType != store.RecordObservation {
		t.Errorf("the two versions are a %s and a %s, want an asset and an observation", ended.RecordType, quiet.RecordType)
	}
}

// versionAt is one Record version out of a rendered branch, decoded — which is
// what a claim about *what the Store says this version is* has to be read off,
// a key that is absent being indistinguishable in the text from one nobody
// wrote.
//
// The rendering separates one file from the next with the newline the file
// itself ends on, so the last byte of every section but the last is the
// separator's. It is put back here rather than read around: the decode is a
// canonical one and holds the bytes it read against the bytes it would write.
func versionAt(t *testing.T, branch, path string) store.RecordVersion {
	t.Helper()

	held := strings.TrimRight(fileIn(t, branch, path), "\n") + "\n"
	version, err := store.DecodeRecordVersion([]byte(held))
	if err != nil {
		t.Fatalf("the branch holds a %s the Store cannot read back: %v", path, err)
	}
	return version
}

// membersDiffering is the top-level members two Record versions disagree
// about, sorted — a member one carries and the other does not included, which
// is the half of the answer this file is about.
//
// Each member is compared in the bytes it was written as. A comparison over
// members needs the shape and not the values, and re-encoding either side would
// be this test deciding what equal means.
func membersDiffering(t *testing.T, one, other string) []string {
	t.Helper()

	held, beside := membersOf(t, one), membersOf(t, other)
	var differs []string
	for name, value := range held {
		if !slices.Equal(value, beside[name]) {
			differs = append(differs, name)
		}
	}
	for name := range beside {
		if _, carried := held[name]; !carried {
			differs = append(differs, name)
		}
	}
	slices.Sort(differs)
	return differs
}

// membersOf is one version file's top-level members, undecoded.
func membersOf(t *testing.T, file string) map[string]json.RawMessage {
	t.Helper()

	var members map[string]json.RawMessage
	if err := json.Unmarshal([]byte(file), &members); err != nil {
		t.Fatalf("a version file the branch holds is not JSON: %v\n%s", err, file)
	}
	return members
}
