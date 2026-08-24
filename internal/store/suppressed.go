package store

import "slices"

// Which fields of a version were suppressed, derived on read (§7, ADR-0007,
// issue #166).
//
// Nothing in the Store says. A field a Manifest declared secret is written as a
// constant string in the position the value would have occupied — no digest, no
// length, no sibling list of what was suppressed — so §7's decoder answers that
// constant as the string it is and there is no key anywhere naming the fields
// it stood for. What is left is the marker itself, and comparing against it is
// the whole derivation.
//
// **The comparison is total rather than a heuristic precisely because the
// marker is a constant** — the same property that makes a rotation invisible. A
// projected value that reads the same is §7's own stated non-case: hyper does
// not disambiguate one, and the alternative that would is a digest or a length
// beside the marker, which is the thing ADR-0007 refused to write.

// SuppressedFields answers, for each version handed over, the names of the
// fields whose value is the secret marker — the presence-only fact `records`
// renders as a row's `secret_fields` (§9).
//
// The answer is positional: one entry per version, in the order they were
// handed over, so a caller pairs it with the listing it already holds rather
// than looking a version up by a key it would have to build. A version that
// suppressed nothing answers nil, which is the absence a row omits its member
// on rather than an empty list it would state one against (§7).
//
// It is a door of its own beside Read for the reason a Version carries no
// content: it hands back the names instead of the fields they were read out of,
// so a listing of a thousand rows never holds a thousand Records' content. Read
// is the door for one version whole; this is the door for one derivation over
// many.
//
// **It opens every one of these files a second time, and that is the honest
// cost rather than an oversight.** The listing that found them read them all
// already and kept only the metadata (readListing), because `written_at` sits
// inside the file and ordering a series opens it anyway. Holding the content
// through that listing to spare this read would put every Record's `fields` in
// memory for every caller, which is the trade the Version/RecordVersion split
// exists to refuse — and the second read is **one** batch for the whole answer,
// so what it costs is bytes and never a subprocess per row (readVersions).
func (s *Store) SuppressedFields(versions []Version) ([][]string, error) {
	read, err := s.readVersions(versions)
	if err != nil {
		return nil, err
	}

	suppressed := make([][]string, len(read))
	for i, version := range read {
		suppressed[i] = markedFields(version.Fields)
	}
	return suppressed, nil
}

// markedFields is the names, sorted by Unicode code point.
//
// It reads the top level of `fields` and no deeper. A Manifest declares
// `secret:` by field name and the writer marks the **whole** field, structure
// and all (internal/run's projected), so the marker stands at a top-level key
// or nowhere — and a mapping the marker suppressed is written as the marker
// rather than descended into. Walking further would be looking for a position
// no writer can put one in.
//
// The order is the identity set's own rather than the mapping's, a Go map
// having none: two renderings of one result are byte-identical and diffable
// (§9), which a member whose order came out of an iteration would not be.
func markedFields(fields Mapping) []string {
	var marked []string
	for name, value := range fields {
		if text, isText := value.(String); isText && string(text) == SecretMarker {
			marked = append(marked, name)
		}
	}
	slices.Sort(marked)
	return marked
}
