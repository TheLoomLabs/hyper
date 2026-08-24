package store_test

import (
	"reflect"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Which fields a version suppressed, read back off the one thing the Store
// holds about them: the constant marker standing in the position the value
// would have occupied (§7, ADR-0007, issue #166).
//
// §7's decoder does not read back which fields were suppressed — nothing in the
// Store says — so the comparison here is against the constant itself. It is
// total rather than a heuristic for exactly that reason, and it is the same
// property that makes a rotation invisible.

// aSuppressedVersion is one version whose Manifest declared two of its three
// fields secret: what the writer produced, which is the marker in the position
// the value would occupy and nothing else about it.
func aSuppressedVersion(t *testing.T) store.RecordVersion {
	t.Helper()

	version := aVersion(t, theSeries, theEntryRunID, 1, theInstant)
	version.Fields = store.Mapping{
		"api_key":    store.Secret(store.String("hunter2")),
		"account_id": store.Secret(store.String("acct-9")),
		"region":     store.String("fsn1"),
	}
	return version
}

func TestSuppressedFields_ReadsTheMarkedFieldsBackOffTheMarker(t *testing.T) {
	_, held := seededStore(t, aSuppressedVersion(t))

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	suppressed, err := held.SuppressedFields(series.Versions)
	if err != nil {
		t.Fatalf("SuppressedFields: %v", err)
	}

	want := [][]string{{"account_id", "api_key"}}
	if !reflect.DeepEqual(suppressed, want) {
		t.Errorf("SuppressedFields = %v, want %v — the marked fields, by code point", suppressed, want)
	}
}

// A version that suppressed nothing carries no names at all, and nil is that
// absence: the row that renders this omits the member rather than writing an
// empty list against it (§7).
func TestSuppressedFields_AnswersNilWhereNothingWasSuppressed(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	suppressed, err := held.SuppressedFields(series.Versions)
	if err != nil {
		t.Fatalf("SuppressedFields: %v", err)
	}

	if len(suppressed) != 1 || suppressed[0] != nil {
		t.Errorf("SuppressedFields = %#v, want one nil — a version that suppressed nothing names nothing", suppressed)
	}
}

// A Tombstone opening the series it ends carries no `fields` at all, which is
// the one version whose absence needs no marker beside it (ADR-0033). Reading
// its suppressed fields is a read over nothing rather than a fault.
func TestSuppressedFields_ReadsAVersionCarryingNoFieldsAtAll(t *testing.T) {
	opening := aTombstone(t, theSeries, theEntryRunID, 1, theInstant)
	opening.Fields = nil
	_, held := seededStore(t, opening)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	suppressed, err := held.SuppressedFields(series.Versions)
	if err != nil {
		t.Fatalf("SuppressedFields: %v", err)
	}

	if len(suppressed) != 1 || suppressed[0] != nil {
		t.Errorf("SuppressedFields = %#v, want one nil — a Tombstone opening a series holds no fields", suppressed)
	}
}

// The answer is positional: the names for the version at each index, so a
// caller pairs them with the versions it handed over rather than looking one up
// by a key it would have to build.
func TestSuppressedFields_AnswersOnePositionPerVersionHanded(t *testing.T) {
	suppressed := aSuppressedVersion(t)
	plain := aVersion(t, theSeries, theSecondRunID, 1, theLaterInstant)
	_, held := seededStore(t, suppressed, plain)

	series, err := held.Series(theSeries)
	if err != nil {
		t.Fatalf("Series: %v", err)
	}
	marked, err := held.SuppressedFields(series.Versions)
	if err != nil {
		t.Fatalf("SuppressedFields: %v", err)
	}

	want := [][]string{{"account_id", "api_key"}, nil}
	if !reflect.DeepEqual(marked, want) {
		t.Errorf("SuppressedFields = %#v, want %#v — the oldest version first, as the series is ordered", marked, want)
	}
}

// Nothing handed over is nothing read: no git subprocess runs and the answer is
// empty, which is what an identity narrowing that matched no series costs.
func TestSuppressedFields_ReadsNothingForNoVersions(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	suppressed, err := held.SuppressedFields(nil)
	if err != nil {
		t.Fatalf("SuppressedFields: %v", err)
	}
	if len(suppressed) != 0 {
		t.Errorf("SuppressedFields = %#v, want nothing", suppressed)
	}
}
