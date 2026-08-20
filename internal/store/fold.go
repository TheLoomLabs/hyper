package store

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// The case fold, and the collision it decides (§7). A Record whose identity
// collides case-insensitively with one already in the Store may not be written,
// and the fold that decides it is `hyper`'s own: a laptop's filesystem is
// usually case-insensitive and a runner's is not, and the branch is written by
// both.
//
// It is decided by **reading** the Store and never by attempting the write and
// seeing what happens — which `hyper` could not do if it wanted to, a git tree
// entry being a byte string and case-sensitive everywhere, so the write always
// succeeds (ADR-0075). The filesystem enters only where a human checks the
// branch out, which is the reading the Head promises them.
//
// Where the check fires, and whether it Refuses or halts, is §6's: this is the
// question, not the guardrail.

// Collision answers the identity the Store already holds that is one with the
// identity handed under the fold, and whether there is one.
//
// An identity the Store holds *exactly* is not a collision: that is the series
// itself, and a further version of it is an ordinary write. What this reports
// is two identities that must be distinct being one under the fold, which is
// `record-identity-collision` (§12).
//
// It reads every Record series on the branch, no index existing here or under
// `.git/hyper/` (§7). Where more than one stored identity folds onto the one
// handed — which takes a Store that already holds a collision — the first in
// identity order is the answer, so two reads of one branch report one identity.
func (s *Store) Collision(id Identity) (Identity, bool, error) {
	records, err := s.Records()
	if err != nil {
		return Identity{}, false, err
	}

	wanted := folded(id)
	for _, series := range records {
		if series.Identity != id && wanted == folded(series.Identity) {
			return series.Identity, true, nil
		}
	}
	return Identity{}, false, nil
}

// folded is an identity under the fold: the three components each folded, and
// still three, because joining them into one string would need a separator no
// component is guaranteed to be free of — and a joining rule that is not
// injective makes two identities one that nothing could ever tell apart (§7,
// and the `shell` projection's own argument in §12).
func folded(id Identity) Identity {
	return Identity{
		Target:     fold(id.Target),
		Definition: fold(id.Definition),
		Name:       fold(id.Name),
	}
}

// fold writes one identity component in its folded form: each rune replaced by
// the least code point it is case-equivalent to under Unicode's simple case
// folding, which is `hyper`'s rule and not a locale's, a filesystem's or a
// language's.
//
// Simple folding rather than a lowercasing, because the two disagree on
// characters a Manifest-declared name really carries: the Kelvin sign folds
// onto `k` and a Greek final sigma onto its medial form, where lowercasing
// answers three distinct characters. The least member of a fold orbit is a
// representative rather than a rendering — it is compared and never written, and
// case is preserved everywhere the Store writes a name (§12).
//
// A byte that is not UTF-8 folds to itself. The identity this is asked about
// has not been written yet — it is a Manifest-declared field of an upstream
// response, which is hostile input and need not be UTF-8 at all — while every
// name the Store holds is, the canonical encoding admitting nothing else (§7).
// Reading an unpaired byte as the replacement character would fold such a name
// onto a stored one carrying that character, which is a collision reported over
// two names sharing nothing: the opposite fault to the one this exists to
// catch, arriving on exactly the input the encoding was written for.
func fold(component string) string {
	var b strings.Builder
	b.Grow(len(component))
	for i := 0; i < len(component); {
		r, width := utf8.DecodeRuneInString(component[i:])
		if r == utf8.RuneError && width == 1 {
			b.WriteString(component[i : i+1])
			i++
			continue
		}
		b.WriteRune(foldRune(r))
		i += width
	}
	return b.String()
}

// foldRune is the least code point in a rune's fold orbit. unicode.SimpleFold
// walks the orbit and returns to where it started, so the walk is finite and
// the least member is a canonical representative: two runes fold together
// exactly where they are in one orbit, which is the same relation
// strings.EqualFold decides pairwise.
func foldRune(r rune) rune {
	least := r
	for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
		if next < least {
			least = next
		}
	}
	return least
}
