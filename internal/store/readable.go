package store

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// The schema test §6 runs at Run start, before the first Step (issue #137).
//
// A Store file written in a shape above this binary's is not read and not
// guessed at (ADR-0028), and the moment that is decided is fixed rather than
// left to wherever a read happens to land: a code whose phase depends on which
// Step read first has no phase to derive, and §7 derives a Refusal's phase from
// its code (§7, §8). So the files a Run **must** read are checked together,
// once, before anything runs — and this is the walk that asks the decoders
// milestone 4 already wrote.
//
// The scope is §6's own sentence and not a wider one: the Journal, and the
// Record heads under the (Definition, Target) pairs the Procedure makes. A file
// no Step of this Run could read never declines it, which is exactly how slot
// coverage and the credential pass are scoped — two Run-start gates scoped by
// one sentence is one rule.
//
// **A Record head is derived and not stored, so reading one opens the series.**
// A version's `written_at` sits inside the file and the ordering is defined over
// the files themselves (ADR-0011), so *the head of this series* is an answer no
// listing gives and every version has to be decoded to reach. Walking a series
// whole is therefore what checking its head costs, here and at every other read
// in this package — `mints` asks `Head` for the same series mid-Step and pays
// the same price. A superseded version above this binary's ceiling stops a Run
// either way; what this gate decides is **when**, which is the whole of why §6
// fixes the moment at all (§7, §8).
//
// **The pair is the finest scope decidable at Run start.** A series is named by
// an identity, and an identity's third segment is what the Operation's
// `identity:` projected — which on a `$`-rooted path arrives with the response
// and so is not knowable before the first call (§3). §6 scopes the test by pair
// rather than by identity for that reason, and this walks the pair.

// Pair is one (Definition, Target) pair a Procedure makes: the scope both the
// schema test here and §6's credential pass are quantified over.
//
// It is §6's own noun. It is deliberately not called a binding: CONTEXT.md
// keeps that word off a Definition, and internal/run already spends it on
// something else — the artefacts one Step resolved to, which is a Manifest, an
// Operation and a Target declaration rather than two names.
//
// It is a pair rather than a Step because a Procedure of ten Steps against one
// Definition and one Target makes one pair, and the walk is over what the Run
// reads rather than over what it does.
type Pair struct{ Target, Definition string }

// Unreadable is one Store file the Run must read that this binary cannot: the
// path it sits at, and the condition its decode answered.
//
// It carries the path because SchemaUnsupported does not: the decoders answer
// the condition and this package's readers name the file, which is the same
// split every other read here makes. §8 states that this is the one Refusal
// whose subject is **evidence rather than an artefact**, so the path is what
// the Refusal cites and there is no line and no field beneath it.
type Unreadable struct {
	File string
	SchemaUnsupported
}

func (u Unreadable) Error() string { return u.File + ": " + u.SchemaUnsupported.Error() }

// Readable holds every Store file this Run must read to the schema versions
// this binary knows, and answers the first it cannot read.
//
// The order is the walk's and the walk's order is fixed: the Journal, whole and
// in path order, and then the Record series under each pair, the pairs sorted
// and each series in path order. §7 states that a Refusal's array is
// ordered and that the terminal line names its first member, and a walk whose
// order came off a map would name a different file on two runs of one command.
//
// Every file is **decoded** rather than having its `schema_version` read off in
// passing. The decoders are where a shape's ceiling is stated, and a second
// reader of that number is a second place for it to be wrong.
//
// Decoding brings the rest of a decode with it — a file that is not JSON, a
// key the shape does not have, bytes that do not re-encode to themselves — and
// that is a consequence rather than a second check this gate is making. Each of
// those already stops the Run at whatever read first meets it; what changes is
// that they now stop it at Run start, before a Step, which is where a Store
// this binary cannot read is cheapest to find out about (§7, ADR-0079).
//
// None of them is a Refusal. `store-schema-unsupported` names a check that
// declined and the ceiling is that check; a file this binary cannot parse at
// all named no check and is answered as an error, which is `failed` and not
// `77` (§12, ADR-0061).
func (s *Store) Readable(pairs []Pair) (Unreadable, bool, error) {
	prefixes := []string{journalPrefix}
	for _, pair := range sortedPairs(pairs) {
		prefixes = append(prefixes, recordsPrefix+encodeSegment(pair.Target)+"/"+encodeSegment(pair.Definition)+"/")
	}

	for _, prefix := range prefixes {
		files, err := s.repo.listTree(s.commit, prefix)
		if err != nil {
			return Unreadable{}, false, err
		}
		unreadable, found, err := s.readable(files)
		if found || err != nil {
			return unreadable, found, err
		}
	}
	return Unreadable{}, false, nil
}

// readable decodes one listing by the shape each path names, and answers the
// first file above this binary's ceiling.
//
// It reads the whole listing in one batch, which is what every other read in
// this package costs: a scan of a year of Runs is one `cat-file --batch` and
// not one subprocess per entry.
func (s *Store) readable(files []treeEntry) (Unreadable, bool, error) {
	blobs := make([]string, len(files))
	for i, file := range files {
		blobs[i] = file.blob
	}
	contents, err := s.repo.readBlobs(blobs)
	if err != nil {
		return Unreadable{}, false, err
	}

	for i, file := range files {
		parsed, err := ParsePath(file.path)
		if err != nil {
			return Unreadable{}, false, err
		}
		if err := decodeForm(parsed.Form, contents[i]); err != nil {
			var unsupported SchemaUnsupported
			if errors.As(err, &unsupported) {
				return Unreadable{File: file.path, SchemaUnsupported: unsupported}, true, nil
			}
			return Unreadable{}, false, fmt.Errorf("%s: %w", file.path, err)
		}
	}
	return Unreadable{}, false, nil
}

// decodeForm reads one file at the shape its own path names, and answers
// whatever that shape's decoder answered.
//
// The introduction is the one form with no schema version and no decoder:
// STORE.md is prose written once, so there is nothing to hold to a ceiling
// (§7).
func decodeForm(form Form, content []byte) error {
	switch form {
	case FormRecord:
		_, err := DecodeRecordVersion(content)
		return err
	case FormRun:
		_, err := DecodeRunFile(content)
		return err
	case FormStep:
		_, err := DecodeStepFile(content)
		return err
	case FormOutcome:
		_, err := DecodeOutcomeFile(content)
		return err
	case FormClosedBy:
		_, err := DecodeClosedBy(content)
		return err
	}
	return nil
}

// sortedPairs is the pairs a Procedure makes, deduplicated and in a fixed
// order: by Target and then by Definition, which is the identity's own order
// and the order `records/` is laid out in (§12).
func sortedPairs(pairs []Pair) []Pair {
	sorted := slices.Clone(pairs)
	slices.SortFunc(sorted, func(a, b Pair) int {
		return cmp.Or(
			strings.Compare(a.Target, b.Target),
			strings.Compare(a.Definition, b.Definition),
		)
	})
	return slices.Compact(sorted)
}
