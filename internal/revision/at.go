package revision

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// The artefacts as one revision held them (§7, ADR-0076, issue #154).
//
// It is this package's second reading of the code branch and it is the same
// subject as the first: `Read` answers what git says about the working tree
// against `HEAD`, and this answers what git held at a commit somebody else's
// Run named. Neither reads a path in the working tree at a revision that is not
// the one standing there, which is what keeps *a reaper loads the Procedure at
// the dead Run's `repo_revision`* a read of the record rather than a read of
// whatever the machine happens to hold now.
//
// **It is `repo_revision` and never `procedure_revision`.** Reconstructing a
// Run's Step sequence means loading every Procedure the top-level one invokes,
// and the Definitions, Target declarations and Manifests those Steps bind — a
// commit resolves all of them and a blob id resolves one file (§7).

// ArtefactsAt answers the artefacts one revision held, and whether this clone
// holds that revision at all.
//
// The selection is the caller's, because what counts as an artefact is
// internal/repository's rule and this package's subject is git: a listing
// filtered here would be the five artefact locations spelled in a second place,
// which is where the day comes that the walk and the revision disagree about
// what a repository is.
//
// **A revision this clone does not hold is answered and never errored.** A Run
// recorded on a runner names a commit a laptop may never have fetched, and a
// shallow clone holds a Store branch whole while holding one code commit — so
// *not held* is an ordinary fact about the clone rather than the world
// resisting, and what a reaper does with it is write the closing write it can
// establish and omit the code facts it cannot (§7).
//
// The order is git's own, which is the tree's: sorted by path, one answer for
// two reads of one commit.
func ArtefactsAt(repoRoot, commit string, wanted func(path string) bool) ([]File, bool, error) {
	git := repository(repoRoot)

	// The revision is resolved before the tree is listed, because the two
	// answer different questions with one word: `ls-tree` of a commit that
	// does not resolve is an error, and this call has to tell that from a
	// commit that resolves and holds nothing wanted.
	//
	// `--verify --quiet` stops in exactly one way, which is what makes the
	// fold below honest: it exits non-zero and says nothing where the
	// revision does not resolve. A git that could not answer *at all* is
	// something every caller here has already had answered — the Store is a
	// branch of this repository and `Read` resolved `HEAD` against it — so
	// what is left for this call to stop on is the revision itself.
	resolved, err := git.text("rev-parse", "--verify", "--quiet", commit+"^{commit}")
	if err != nil || resolved == "" {
		return nil, false, nil
	}

	listing, err := git.run("ls-tree", "-r", "-z", resolved)
	if err != nil {
		return nil, false, err
	}
	records, err := treeRecords(listing)
	if err != nil {
		return nil, false, err
	}

	// Blobs and nothing else, and only the paths the caller wanted: a
	// repository holds files that are not artefacts, and a read that brought
	// them back would be one whose cost grew with the repository rather than
	// with the artefacts.
	var paths, blobs []string
	for _, record := range records {
		if record.kind != "blob" || !wanted(record.path) {
			continue
		}
		paths = append(paths, record.path)
		blobs = append(blobs, record.object)
	}
	if len(paths) == 0 {
		return nil, true, nil
	}

	contents, err := git.blobs(blobs)
	if err != nil {
		return nil, false, err
	}
	files := make([]File, len(paths))
	for i, path := range paths {
		files[i] = File{Path: path, Bytes: contents[i]}
	}
	return files, true, nil
}

// blobs reads the objects named, in the order named, in one `cat-file --batch`.
//
// One subprocess for the set rather than one per file is the trade this package
// already makes one `ls-tree` for in `differs`: the answer is a property of the
// set, and a Procedure spanning a dozen artefacts should not cost a dozen
// processes.
//
// The batch protocol is git's own — `<object> SP <type> SP <size> LF`, the
// content, and one LF — and it is read by the size git states rather than by
// scanning for a separator, because an artefact may hold any byte at all, a
// newline among them.
func (g gitRepository) blobs(objects []string) ([][]byte, error) {
	if len(objects) == 0 {
		return nil, nil
	}
	batch, err := g.with([]byte(strings.Join(objects, "\n")+"\n"), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}

	contents := make([][]byte, 0, len(objects))
	rest := batch
	for _, object := range objects {
		header, body, split := bytes.Cut(rest, []byte("\n"))
		if !split {
			return nil, fmt.Errorf("git cat-file answered no header for %s", object)
		}
		fields := strings.Fields(string(header))
		if len(fields) != 3 {
			return nil, fmt.Errorf("git cat-file wrote %q for %s, which is not <object> <type> <size>", header, object)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size+1 > len(body) {
			return nil, fmt.Errorf("git cat-file wrote %q for %s, which is not a size this answer carries", header, object)
		}
		contents = append(contents, body[:size])
		rest = body[size+1:]
	}
	return contents, nil
}
