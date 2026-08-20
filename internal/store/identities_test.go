package store_test

import (
	"iter"
	"reflect"
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The identity set: the digest's place in a Step file, and the backward walk
// that reads a set back from any entry the Store holds (§7, ADR-0055, issue
// #129).
//
// An unchanged listing of five hundred Records costs one line; a changed one
// costs the set. What makes that affordable is that the walk is **total** —
// every entry either holds members or, by holding a digest, names a set an
// earlier entry holds in full — so neither the set nor its count is ever stored
// a second time.

// carrying is a Step file holding one identity set, which is all the walk reads
// off one.
func carrying(id string, identities store.Identities) store.StepFile {
	return store.StepFile{StepCode: store.StepCode{ID: id}, Identities: identities}
}

// backward is a walk over entries, newest first and starting with the entry in
// hand — the order the Journal's date partitions are scanned in (§7).
func backward(files ...store.StepFile) iter.Seq2[store.StepFile, error] {
	return func(yield func(store.StepFile, error) bool) {
		for _, file := range files {
			if !yield(file, nil) {
				return
			}
		}
	}
}

func TestConcluded_WritesTheSetInFullExactlyWhereTheDigestMoved(t *testing.T) {
	names := []string{"ci-x86", "über-vm", "ci-macos", "ci-riscv"}
	const digest = "sha256:a118a517431e241eac83559919ae969346bf5a3bf6e06c6db3e636f378fcdf12"

	t.Run("no digest behind it", func(t *testing.T) {
		// A Step's first Run, and a Step whose authored id moved: a
		// different Step, with no digest behind it, writing its set in
		// full like any other first Run (§7, ADR-0055).
		got := store.Concluded(names, "")

		want := store.Identities{Digest: digest, Members: []string{"ci-macos", "ci-riscv", "ci-x86", "über-vm"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("identities = %+v, want %+v", got, want)
		}
	})

	t.Run("the digest did not move", func(t *testing.T) {
		got := store.Concluded(names, digest)

		if want := (store.Identities{Digest: digest}); !reflect.DeepEqual(got, want) {
			t.Errorf("identities = %+v, want %+v", got, want)
		}
	})

	t.Run("the digest moved to the empty set", func(t *testing.T) {
		// Written in full, the empty list included: absence already
		// means *the digest did not move*, so a reader would otherwise
		// decode *we looked and saw nothing* from a constant (§7).
		got := store.Concluded(nil, digest)

		want := store.Identities{Digest: store.IdentityDigest(nil), Members: []string{}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("identities = %+v, want %+v", got, want)
		}
	})
}

func TestReadIdentitySet_ReadsTheSetOffTheEntryInHandWhereItHoldsOne(t *testing.T) {
	set := []string{"preview-17.example.com", "preview-42.example.com"}

	got, err := store.ReadIdentitySet("retire", backward(carrying("retire", store.Concluded(set, ""))))
	if err != nil {
		t.Fatalf("ReadIdentitySet = %v, want the set read", err)
	}
	if !slices.Equal(got, set) {
		t.Errorf("set = %q, want %q", got, set)
	}
}

func TestReadIdentitySet_WalksBackToTheRunThatHoldsTheSetInFull(t *testing.T) {
	set := []string{"preview-17.example.com", "preview-42.example.com"}
	full := store.Concluded(set, "")
	unmoved := store.Concluded(set, full.Digest)

	// Two Runs whose digest did not move, and the Run before them where it
	// last did. Nothing removes the entries in between — Compaction touches
	// interior Observation versions and never a Journal entry (§7).
	got, err := store.ReadIdentitySet("retire", backward(
		carrying("retire", unmoved),
		carrying("retire", unmoved),
		carrying("retire", full),
	))
	if err != nil {
		t.Fatalf("ReadIdentitySet = %v, want the set read", err)
	}
	if !slices.Equal(got, set) {
		t.Errorf("set = %q, want %q", got, set)
	}
}

func TestReadIdentitySet_MatchesTheStepByItsAuthoredID(t *testing.T) {
	// The Run compared against is the last one in which **that** Step
	// carried a set at all, and never simply the previous Run (ADR-0055).
	set := []string{"preview-42.example.com"}
	full := store.Concluded(set, "")

	got, err := store.ReadIdentitySet("retire", backward(
		carrying("retire", store.Concluded(set, full.Digest)),
		carrying("announce", store.Concluded([]string{"something-else"}, "")),
		store.StepFile{StepCode: store.StepCode{ID: "retire"}},
		carrying("retire", full),
	))
	if err != nil {
		t.Fatalf("ReadIdentitySet = %v, want the set read", err)
	}
	if !slices.Equal(got, set) {
		t.Errorf("set = %q, want %q", got, set)
	}
}

func TestReadIdentitySet_ReadsNoFurtherThanTheEntryThatHoldsTheSet(t *testing.T) {
	// The backward scan stops at the first match, which is what keeps
	// reading a set off an old entry from being a scan of everything (§7).
	set := []string{"preview-42.example.com"}
	full := store.Concluded(set, "")

	read := 0
	walk := func(yield func(store.StepFile, error) bool) {
		for _, file := range []store.StepFile{
			carrying("retire", store.Concluded(set, full.Digest)),
			carrying("retire", full),
			carrying("retire", store.Concluded([]string{"older"}, "")),
		} {
			read++
			if !yield(file, nil) {
				return
			}
		}
	}

	if _, err := store.ReadIdentitySet("retire", walk); err != nil {
		t.Fatalf("ReadIdentitySet = %v, want the set read", err)
	}
	if read != 2 {
		t.Errorf("read %d entries, want it to stop at the one holding the set", read)
	}
}

func TestReadIdentitySet_RefusesAWalkThatNeverReachesASetHeldInFull(t *testing.T) {
	set := []string{"preview-42.example.com"}
	full := store.Concluded(set, "")

	if _, err := store.ReadIdentitySet("retire", backward(carrying("retire", store.Concluded(set, full.Digest)))); err == nil {
		t.Errorf("ReadIdentitySet = no error, want the walk reported as ending short")
	}
}

func TestReadIdentitySet_RefusesEntriesThatContradictOneAnother(t *testing.T) {
	// A digest naming a set an earlier entry does not hold is a Store that
	// disagrees with itself, and answering the earlier set anyway would be
	// this package guessing which of the two files is right.
	set := []string{"preview-42.example.com"}

	if _, err := store.ReadIdentitySet("retire", backward(
		carrying("retire", store.Identities{Digest: store.IdentityDigest(set)}),
		carrying("retire", store.Concluded([]string{"a-different-set"}, "")),
	)); err == nil {
		t.Errorf("ReadIdentitySet = no error, want the contradiction reported")
	}
}

func TestReadIdentitySet_RefusesASetItsOwnDigestDoesNotName(t *testing.T) {
	if _, err := store.ReadIdentitySet("retire", backward(carrying("retire", store.Identities{
		Digest:  store.IdentityDigest([]string{"preview-42.example.com"}),
		Members: []string{"preview-17.example.com"},
	}))); err == nil {
		t.Errorf("ReadIdentitySet = no error, want the file reported as disagreeing with itself")
	}
}
