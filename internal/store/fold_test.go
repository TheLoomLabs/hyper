package store_test

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Case is the one place a reader's two environments genuinely differ, a
// laptop's filesystem being usually case-insensitive and a runner's not, so the
// fold is `hyper`'s rather than the filesystem's (§7). This ticket builds the
// reading half: given an identity, does the Store already hold one that folds
// onto it. Where the check fires and whether it Refuses or halts is §6's
// (issue #130).

// TestCollision_AnswersTheIdentityTheStoreHoldsUnderTheFold. Two identities
// that differ only in case are one under the fold, and what comes back is the
// spelling the Store holds — which is what a Refusal has to render beside the
// one that was about to be written.
func TestCollision_AnswersTheIdentityTheStoreHoldsUnderTheFold(t *testing.T) {
	stored := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "Preview-42.Example.com"}
	_, held := seededStore(t, aVersion(t, stored, theEntryRunID, 1, theInstant))

	arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "preview-42.example.COM"}
	collides, found, err := held.Collision(arriving)
	if err != nil {
		t.Fatalf("Collision: %v", err)
	}
	if !found {
		t.Fatalf("%q does not collide with the stored %q", arriving.Name, stored.Name)
	}
	if collides != stored {
		t.Errorf("the collision is %+v, want the identity the Store holds, %+v", collides, stored)
	}
}

// TestCollision_FoldsEveryComponentOfTheIdentity. A Record's identity is
// (Target, Definition, name) and the fold is over the identity, not over its
// last third.
func TestCollision_FoldsEveryComponentOfTheIdentity(t *testing.T) {
	stored := store.Identity{Target: "Cloudflare-Prod", Definition: "Preview-DNS", Name: theSeries.Name}
	_, held := seededStore(t, aVersion(t, stored, theEntryRunID, 1, theInstant))

	for _, arriving := range []store.Identity{
		{Target: "cloudflare-prod", Definition: "Preview-DNS", Name: theSeries.Name},
		{Target: "Cloudflare-Prod", Definition: "preview-dns", Name: theSeries.Name},
	} {
		collides, found, err := held.Collision(arriving)
		if err != nil {
			t.Fatalf("Collision: %v", err)
		}
		if !found || collides != stored {
			t.Errorf("%+v collides with %+v (%v), want the stored %+v", arriving, collides, found, stored)
		}
	}
}

// TestCollision_IsNotTheSeriesItself. An identity the Store already holds
// exactly is that series, and a further version of it is an ordinary write.
// Only two identities that must be distinct being one under the fold is a
// collision.
func TestCollision_IsNotTheSeriesItself(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	if collides, found, err := held.Collision(theSeries); err != nil || found {
		t.Errorf("Collision(%+v) = %+v, %v, %v; a series colliding with itself is a further version of it", theSeries, collides, found, err)
	}
}

// TestCollision_AnswersNothingWhereTheStoreHoldsNoSuchIdentity, which is every
// first write of every series.
func TestCollision_AnswersNothingWhereTheStoreHoldsNoSuchIdentity(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "preview-99.example.com"}
	if collides, found, err := held.Collision(arriving); err != nil || found {
		t.Errorf("Collision(%+v) = %+v, %v, %v; want no collision", arriving, collides, found, err)
	}
}

// TestCollision_IsDecidedByReadingAndNeverByAttemptingTheWrite. A git tree
// entry is a byte string and case-sensitive everywhere, so the write always
// succeeds — this seeds both spellings, which stand side by side as two files,
// and the check still answers that they are one identity (§7, ADR-0075).
func TestCollision_IsDecidedByReadingAndNeverByAttemptingTheWrite(t *testing.T) {
	lower := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "preview-42.example.com"}
	upper := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "PREVIEW-42.EXAMPLE.COM"}
	r, held := seededStore(t,
		aVersion(t, lower, theEntryRunID, 1, theInstant),
		aVersion(t, upper, theSecondRunID, 1, theInstant),
	)

	tree := r.storeTree(r.root)
	if _, both := tree[pathOf(aVersion(t, lower, theEntryRunID, 1, theInstant))]; !both {
		t.Fatalf("the branch does not hold the lowercase spelling; it holds %v", tree)
	}
	if _, both := tree[pathOf(aVersion(t, upper, theSecondRunID, 1, theInstant))]; !both {
		t.Fatalf("the branch does not hold the uppercase spelling; the write always succeeds (ADR-0075)")
	}

	collides, found, err := held.Collision(upper)
	if err != nil || !found {
		t.Fatalf("Collision = %+v, %v, %v; want the lowercase series", collides, found, err)
	}
	if collides != lower {
		t.Errorf("the collision is %+v, want %+v", collides, lower)
	}
}

// TestCollision_FoldsBeyondASCIIAndReachesNoFilesystem. The fold is `hyper`'s
// own: it is the same answer on both platforms because nothing about it asks a
// platform — the Store is never checked out, so these names reach no filesystem
// at all (ADR-0075). The Kelvin sign is the case a fold has and a lowercasing
// of ASCII does not.
func TestCollision_FoldsBeyondASCIIAndReachesNoFilesystem(t *testing.T) {
	for name, pair := range map[string][2]string{
		"a Latin letter with a diaeresis": {"Über-vm", "über-vm"},
		"the Kelvin sign":                 {"Krypton", "krypton"},
		"a Greek final sigma":             {"ΟΔΟΣ", "οδος"},
	} {
		t.Run(name, func(t *testing.T) {
			stored := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: pair[0]}
			_, held := seededStore(t, aVersion(t, stored, theEntryRunID, 1, theInstant))

			arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: pair[1]}
			collides, found, err := held.Collision(arriving)
			if err != nil {
				t.Fatalf("Collision: %v", err)
			}
			if !found || collides != stored {
				t.Errorf("%q collides with %+v (%v), want the stored %q", pair[1], collides, found, pair[0])
			}
		})
	}
}

// TestCollision_TellsApartTwoNamesThatAreNotOneUnderTheFold, which is the half
// that has to be false as often as the other is true.
func TestCollision_TellsApartTwoNamesThatAreNotOneUnderTheFold(t *testing.T) {
	_, held := seededStore(t, aVersion(t, theSeries, theEntryRunID, 1, theInstant))

	arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "preview-42.example.co"}
	if collides, found, err := held.Collision(arriving); err != nil || found {
		t.Errorf("Collision(%q) = %+v, %v, %v; two names that are not one under the fold are two identities", arriving.Name, collides, found, err)
	}
}

// TestCollision_DoesNotFoldBytesThatAreNotUTF8OntoOneCharacter. The identity
// this is asked about has not been written yet — it is a Manifest-declared
// field of an upstream response, which is hostile input and need not be UTF-8
// at all — while everything the Store holds is, the canonical encoding
// admitting nothing else. Reading an unpaired byte as the replacement character
// would fold such a name onto a stored one that carries that character, which
// is a collision reported over two names sharing nothing.
func TestCollision_DoesNotFoldBytesThatAreNotUTF8OntoOneCharacter(t *testing.T) {
	stored := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "vm-\ufffd"}
	_, held := seededStore(t, aVersion(t, stored, theEntryRunID, 1, theInstant))

	arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: "vm-\xff"}
	if collides, found, err := held.Collision(arriving); err != nil || found {
		t.Errorf("Collision(%q) = %+v, %v, %v; an unpaired byte is itself and not the replacement character", arriving.Name, collides, found, err)
	}
}

// TestCollision_ReadsTheIdentityFromTheFile, which is where the answer to
// *which series is this* lives: an over-long identity is truncated in the path
// and whole in the file, so a check that folded path segments would answer
// nothing about exactly the identities the encoding exists to survive (§12).
func TestCollision_ReadsTheIdentityFromTheFile(t *testing.T) {
	long := strings.Repeat("Preview-", 40) + "42.Example.com"
	stored := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: long}
	_, held := seededStore(t, aVersion(t, stored, theEntryRunID, 1, theInstant))

	arriving := store.Identity{Target: theSeries.Target, Definition: theSeries.Definition, Name: strings.ToLower(long)}
	collides, found, err := held.Collision(arriving)
	if err != nil {
		t.Fatalf("Collision: %v", err)
	}
	if !found || collides != stored {
		t.Errorf("the collision is %+v (%v), want the stored identity whole, %+v", collides, found, stored)
	}
}
