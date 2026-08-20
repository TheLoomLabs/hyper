package store

import (
	"fmt"
	"iter"
	"slices"
)

// The identity set as it is carried and as it is read back. What a Step
// concluded about is written as a digest and — only where that digest moved —
// as the sorted members in full, so an unchanged listing of five hundred
// Records costs one line (§7).
//
// What makes that affordable is that reading a set back is **total**. Every
// entry either holds the members or, by holding a digest, names a set an
// earlier entry holds in full, terminating at the Run where that Step first
// carried one. Nothing removes the entries in between — Compaction touches
// interior Observation versions and never a Journal entry — so a set and its
// count are recoverable from any entry the Store holds, which is why neither is
// stored a second time.

// Identities is what a Step concluded a recorded conclusion about — what it
// projected from a response under read and mutate, and what it confirmed
// destroyed under destroy. It is not what the Step wrote and not what it saw: a
// Record that came back unchanged mints no file and is in the set, which is the
// case the whole mechanism exists for (ADR-0030).
type Identities struct {
	// Digest is sha256: over the canonical encoding of the sorted array —
	// the array as it would be written alone, at no indent, and never as it
	// sits inside a Step file, where Members carries four spaces of it
	// (ADR-0079).
	Digest string
	// Members is the sorted set in full, written whenever the digest moved
	// and absent where it did not. Nil is that absence; the empty slice is
	// a set that moved to empty and is written `[]`, which is one of the
	// two exceptions to the absence rule and earns it — absence here
	// already means *the digest did not move*, so a reader would otherwise
	// decode *we looked and saw nothing* from recognising a constant.
	Members []string
}

// write puts the identity set into its block.
//
// A set with members and no digest is not one: the digest is taken over the
// members and is computed wherever a set is, so it cannot be the half that is
// missing. Concluded is the only door that builds one and never produces that,
// which is what makes this hyper's own arithmetic rather than a caller's input.
func (i Identities) write(m members) {
	if i.Members != nil && i.Digest == "" {
		impossible("an identity set of %d members carries no digest (§7)", len(i.Members))
	}
	m.text("digest", i.Digest)
	if i.Members != nil {
		m["members"] = Always(namesArray(i.Members))
	}
}

// Concluded is the identity set a Step carries, given the names it concluded
// about and the digest the last Run in which that Step carried a set held.
//
// previous is the empty string where there is no such Run: a Step's first, and
// a Step whose authored id moved, which is a different Step with no digest
// behind it and writes its set in full like any other first Run (ADR-0055).
// Finding that digest is the caller's — it is a backward walk over the Journal,
// and which Dispositions carry a set at all is §6's.
//
// The names are a set, Go having no type that says so, and a name repeated is
// one name: a duplicate that reached the digest would give one set two digests,
// which is a spurious version minted on the next Run.
func Concluded(names []string, previous string) Identities {
	sorted := slices.Compact(slices.Sorted(slices.Values(names)))
	if sorted == nil {
		// The empty set is written `[]` where it is written at all, and
		// a nil slice is the absence one key over. They are one value
		// everywhere else in this package and two here.
		sorted = []string{}
	}

	digest := IdentityDigest(sorted)
	if digest == previous {
		return Identities{Digest: digest}
	}
	return Identities{Digest: digest, Members: sorted}
}

// ReadIdentitySet answers the set a Step concluded about, from the entry in
// hand and as many earlier ones as it takes.
//
// steps yields Step files newest first, beginning with the entry the set is
// being read off — the order a backward scan through the Journal's date
// partitions gives (§7). It yields whatever the scan reaches and this walk
// selects: a file is a candidate where its authored id is the one asked for and
// it carries an identity set at all, three of §12's seven Dispositions carrying
// none and a fourth writing no file. Filtering dry-run entries out is the
// caller's, that being a fact about the Run and not about the Step (§6, §7,
// ADR-0001).
//
// It reads no further than the entry holding the set, so a set read off a
// recent entry costs one file and one off an old one costs the entries between.
func ReadIdentitySet(id string, steps iter.Seq2[StepFile, error]) ([]string, error) {
	digest := ""
	for step, err := range steps {
		if err != nil {
			return nil, err
		}
		if step.ID != id || step.Identities.Digest == "" {
			continue
		}

		set := step.Identities
		if digest == "" {
			digest = set.Digest
		}
		if set.Digest != digest {
			// Every entry between the one in hand and the one
			// holding the set carries the digest that did not move,
			// so a different one is two files disagreeing about
			// which set this Step concluded about — and picking one
			// of them is this package guessing.
			return nil, fmt.Errorf("the entry in hand carries %s and an earlier one carries %s: no Run between them wrote the members", digest, set.Digest)
		}
		if set.Members == nil {
			continue
		}
		if held := IdentityDigest(set.Members); held != digest {
			return nil, fmt.Errorf("the entry holding the members carries %s over a set whose digest is %s", digest, held)
		}
		return set.Members, nil
	}

	if digest == "" {
		return nil, fmt.Errorf("no entry the walk reached carries an identity set for step %q", id)
	}
	return nil, fmt.Errorf("the walk ended before the Run that wrote the members of %s", digest)
}
