package store

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"
)

// The Head, derived (§7, ADR-0011). Nothing in the Store points at the current
// version of a Record: a series' versions are ordered on the `written_at` each
// file carries, ties broken by the file name, so finding the Head is a listing
// and two environments writing one series contend over nothing.
//
// Every answer here is defined over the files themselves and over nothing else.
// The path is read for where a file sits and never for what it holds — the
// grammar truncates an over-long identity segment and suffixes a digest (§12),
// so *which series is this* is answered by the identity the file restates,
// unencoded and in full. Ordering opens every version of a series, which is
// exactly why the fetch is never filtered (ADR-0074).

// Version is one version of a series as a listing answers it: what the file
// says about itself, where it sits, and where it falls in the ordering.
//
// It is not the file. RecordVersion is the file — the metadata and the content
// together, which is what an encode writes and a decode reads — and this is
// what the branch holds one of, at the grain the Head is derived at. Read is
// the door from one to the other.
//
// It carries no `fields`. Ordering a series and naming a version need every
// member of the metadata and no byte of the content, so a listing of five
// hundred versions holds five hundred of these and Read opens the one a caller
// went on to ask for.
type Version struct {
	Metadata
	// File is the Store path this version sits at, which is what names it
	// to git and what breaks a tie in the ordering.
	File string
	// Ordinal is this version's position in the ordering, from 1. It is
	// derived here, stored nowhere, and unstable by construction: a version
	// arriving beneath one already rendered moves every ordinal above it,
	// and so does a Compaction. Nothing takes one as input — naming a
	// version is naming its Run (ADR-0049).
	Ordinal int
}

// Series is one Record's versions, in the order the ordering puts them: oldest
// first, so a version's Ordinal is its position in this list.
//
// A series the Store does not hold is this with no versions rather than an
// error. *hyper has never recorded this* is the answer every first Run of every
// Step reads, and it is not a fault.
type Series struct {
	Identity Identity
	Versions []Version
}

// Head is the current version of the Record, and whether the series holds one
// at all.
//
// It is the last of the ordering and not a marker anybody wrote. A series whose
// Head carries Tombstone reads dead; a further version above it makes it read
// alive again, which is what makes destroy-then-recreate behave as §6 states
// under `skip-if-recorded` (§7). Nothing here editorialises beyond that: what
// the Head is and what it says about itself are the two facts, and reading them
// together is the caller's.
func (s Series) Head() (Version, bool) {
	if len(s.Versions) == 0 {
		return Version{}, false
	}
	return s.Versions[len(s.Versions)-1], true
}

// Series answers one Record's versions, ordered.
//
// It lists the one directory the identity names and reads every file in it,
// which is what deriving a Head costs: a version's `written_at` sits inside the
// file. The identity is then read back out of each file rather than trusted
// from the directory it was found in — the encoding is lossy in the direction
// that matters, so a file's own account of which series it belongs to is the
// only one.
func (s *Store) Series(id Identity) (Series, error) {
	files, err := s.repo.listTree(s.commit, seriesPrefix(id))
	if err != nil {
		return Series{}, err
	}
	versions, err := s.readListing(files)
	if err != nil {
		return Series{}, err
	}

	held := make([]Version, 0, len(versions))
	for _, version := range versions {
		// Two identities share a directory only where one of them was
		// truncated onto the other's digest, which is a SHA-256
		// collision away. The file decides, at whatever that costs.
		if version.Identity == id {
			held = append(held, version)
		}
	}
	return Series{Identity: id, Versions: order(held)}, nil
}

// Head answers the current version of one Record, and whether the Store holds
// the series at all. It is Series with the ordering read off it, which is the
// query almost every caller makes and the one §6 asks per member of an
// Expansion.
func (s *Store) Head(id Identity) (Version, bool, error) {
	series, err := s.Series(id)
	if err != nil {
		return Version{}, false, err
	}
	head, found := series.Head()
	return head, found, nil
}

// Records answers every Record series the branch holds, each with its versions
// ordered, sorted by identity.
//
// It reads every Record file on the branch, which is what an enumeration over
// files costs where no index exists — and none does, here or under
// `.git/hyper/`: §7 permits derived state that makes this faster and states
// that no answer depends on one existing, so every answer is a fresh read (§7,
// issue #124).
//
// The order is the identity's own — Target, then Definition, then name, by code
// point — and never the listing's. Escaping drags every escaped character to
// the left of every unreserved one, so a path order is an order over the
// encoding rather than over the names anybody wrote (§12, ADR-0044).
func (s *Store) Records() ([]Series, error) {
	files, err := s.repo.listTree(s.commit, recordsPrefix)
	if err != nil {
		return nil, err
	}
	versions, err := s.readListing(files)
	if err != nil {
		return nil, err
	}

	held := map[Identity][]Version{}
	for _, version := range versions {
		held[version.Identity] = append(held[version.Identity], version)
	}

	records := make([]Series, 0, len(held))
	for id, versions := range held {
		records = append(records, Series{Identity: id, Versions: order(versions)})
	}
	slices.SortFunc(records, func(a, b Series) int {
		return cmp.Or(
			strings.Compare(a.Identity.Target, b.Identity.Target),
			strings.Compare(a.Identity.Definition, b.Identity.Definition),
			strings.Compare(a.Identity.Name, b.Identity.Name),
		)
	})
	return records, nil
}

// Read answers one version whole: its metadata and the content a listing does
// not carry.
//
// It opens the file a second time rather than every listing holding every
// Record's content in memory, which is the trade the two shapes exist for. The
// version is named by the file a listing found it at, so a caller cannot ask
// for one this Store never answered.
func (s *Store) Read(version Version) (RecordVersion, error) {
	read, err := s.readVersions([]Version{version})
	if err != nil {
		return RecordVersion{}, err
	}
	return read[0], nil
}

// Contents answers the versions named, whole, in the order they were named, in
// one batch read.
//
// It is Read's shape for many versions rather than one, and it exists for the
// reason SuppressedFields does: a caller reading the content of a hundred
// versions one at a time pays a subprocess a hundred times, where the door
// behind both of them reads the lot in one (readVersions). §8's Comparison is
// the caller — it reads the two endpoint versions of every eligible identity,
// so the count is a Run's identity set rather than one Record.
//
// The versions are named by the files a listing found them at, so a caller
// cannot ask for one this Store never answered.
func (s *Store) Contents(versions []Version) ([]RecordVersion, error) {
	return s.readVersions(versions)
}

// readVersions opens the versions named and answers them whole, in the order
// they were named, in one batch read.
//
// It is the door behind Read and behind SuppressedFields, which is what keeps
// *a version's content costs a second open* one fact rather than two: the
// listing that found these versions read every one of their files already and
// kept only the metadata (readListing), so anything reaching for `fields` opens
// them again, and doing that per version would be a subprocess per version.
//
// The version is named by the file a listing found it at, so a caller cannot
// ask for one this Store never answered.
func (s *Store) readVersions(versions []Version) ([]RecordVersion, error) {
	blobs := make([]string, len(versions))
	for i, version := range versions {
		// `<commit>:<path>` is git's own way of naming one file in one
		// tree, which is what this handle is pinned to.
		blobs[i] = s.commit + ":" + version.File
	}
	contents, err := s.repo.readBlobs(blobs)
	if err != nil {
		return nil, err
	}

	read := make([]RecordVersion, len(versions))
	for i, version := range versions {
		decoded, err := decodeVersion(version.File, contents[i])
		if err != nil {
			return nil, err
		}
		read[i] = decoded
	}
	return read, nil
}

// readListing decodes every file a listing found, in the order it found them.
//
// It is one batch read for the whole listing rather than a subprocess per file,
// which is what keeps *finding a Head opens every version of the series* a cost
// in bytes rather than in processes.
//
// Every file is read whole and only its metadata is kept, and that is the
// honest cost rather than an oversight: `written_at` sits inside the file, so
// ordering opens all of them anyway, and the decode holds the bytes against a
// re-encode of what it read, which needs the content. What the split buys is
// memory and a caller's attention — an enumeration does not hold every Record's
// content, and nothing that does not need `fields` is handed them.
func (s *Store) readListing(files []treeEntry) ([]Version, error) {
	blobs := make([]string, len(files))
	for i, file := range files {
		blobs[i] = file.blob
	}
	contents, err := s.repo.readBlobs(blobs)
	if err != nil {
		return nil, err
	}

	versions := make([]Version, len(files))
	for i, file := range files {
		decoded, err := decodeVersion(file.path, contents[i])
		if err != nil {
			return nil, err
		}
		// The file's own identity, Run and Step build the path it sits
		// at, and a file where they do not is a file this package did
		// not write. It is a fault rather than a skip: a version
		// nothing can find again by the identity it carries is one an
		// enumeration would report and a Head lookup would miss, which
		// is two answers about one Store that disagree (§7).
		//
		// What that costs is stated rather than hidden: one file under
		// records/ that `hyper` did not write fails every read of the
		// branch, not just a read of its own series. That is the same
		// answer the schema ceiling gives one shape over — a file this
		// reader cannot account for is surfaced rather than read or
		// skipped — and §12 closes records/ at one form, so there is no
		// file there this package is entitled to pass over.
		if built := RecordPath(decoded.Identity, decoded.Run, decoded.Step); built != file.path {
			return nil, fmt.Errorf("%q holds a version the grammar names %q: a Record version sits at the path its own identity builds (§12)", file.path, built)
		}
		versions[i] = Version{Metadata: decoded.Metadata, File: file.path}
	}
	return versions, nil
}

// decodeVersion reads one Record version and names the file it came out of.
//
// Naming it is the whole of what this adds, and it is what the schema ceiling
// needs: a file written above this reader's ceiling surfaces the
// `store-schema-unsupported` condition rather than being read or skipped, and
// the caller renders the Refusal over a path this package knows and does not
// hold (§7, ADR-0028).
func decodeVersion(file string, content []byte) (RecordVersion, error) {
	decoded, err := DecodeRecordVersion(content)
	if err != nil {
		return RecordVersion{}, fmt.Errorf("%s: %w", file, err)
	}
	return decoded, nil
}

// order puts a series' versions in the order the Head is derived from, and
// numbers them. It sorts in place and hands back the slice it was given.
//
// The instant first, and the file name where two versions share one. §7 says
// the file name and ADR-0011 says the Run id; they are the same rule at two
// grains and the finer one is the rule, since the file name is
// `<run-id>-<nnnn>` and two Steps of one Run writing one identity write two
// paths the coarser reading could not order. The comparison is byte-wise over
// the name, which is the same *the bytes moved* test the encoding runs
// everywhere else.
//
// The name and not the whole path: every version of one series sits in one
// directory, so the two answer alike, and the name is what §7 states.
func order(versions []Version) []Version {
	slices.SortFunc(versions, func(a, b Version) int {
		return cmp.Or(
			a.WrittenAt.Compare(b.WrittenAt),
			strings.Compare(path.Base(a.File), path.Base(b.File)),
		)
	})
	for i := range versions {
		versions[i].Ordinal = i + 1
	}
	return versions
}
