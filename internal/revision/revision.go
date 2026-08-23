// Package revision is what git says about the **code** branch: the commit at
// `HEAD`, the blob id of an artefact as the working tree holds it, and whether
// any artefact a Run read differs from `HEAD` or is untracked (§7, issue #136).
//
// It is the other half of internal/store, and the two are deliberately not one
// package. That one is the record — an orphan branch hyper writes and no human
// authors — and this one is the repository the artefacts sit in, which hyper
// only ever reads. Nothing here writes an object, moves a ref, reaches a
// remote or names a commit identity, so the environment its subprocesses run
// with carries none of those; and nothing in internal/store reads a path in the
// working tree, which is what keeps *hyper never checks the Store out*
// (ADR-0075) a fact about one package.
//
// What it answers is Provenance's own three members and nothing else:
// `repo_revision`, the two blob-id members through Blob, and the `repo_dirty`
// marker (§7, ADR-0043). It derives no Provenance value itself — which member
// goes on which file is internal/store's split, and which artefacts a Run read
// is the Run's own fact.
//
// A blob id is computed here rather than asked of git, because it is
// computable: git's object id is a SHA-1 over a header and the content, so an
// artefact hyper has already read is one subprocess it does not have to start.
// The test of that claim is `git hash-object` answering the same string.
package revision

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/git"
)

// File is one artefact a Run read: where it sits in the repository, relative to
// the root and with forward slashes, and the exact bytes it was read as.
//
// The bytes travel with the path because the comparison is against what the Run
// actually read and never against what the file says now: an artefact edited
// between the load and this call is a repository nobody was running, and
// re-reading it here would let one appear.
type File struct {
	Path  string
	Bytes []byte
}

// Facts is what the code branch says about the code a Run performed.
type Facts struct {
	// Head is the commit at `HEAD`, whole. It is what a reaper loads the
	// Procedure sequence at, which a blob id could not do (§7).
	Head string
	// Dirty says some artefact the Run read differs from `HEAD` or is
	// untracked — exactly the file set §8's catch-all row counts the moved
	// lines of, which is what makes the marker and the count agree by
	// construction (§7).
	Dirty bool
}

// Blob is the git blob id of content: SHA-1 over `blob <length>\0` and the
// bytes, which is git's object id and not a digest hyper chose.
//
// hyper names the algorithm where hyper chose it (§7), and here it did not: the
// two revision members are written bare because the algorithm is the
// repository's, and a reader verifies one with `git hash-object`.
func Blob(content []byte) string {
	sum := sha1.New()
	fmt.Fprintf(sum, "blob %d\x00", len(content))
	sum.Write(content)
	return hex.EncodeToString(sum.Sum(nil))
}

// Read answers the facts, over the artefacts the Run read.
//
// It answers an error where `HEAD` resolves to no commit, which is a repository
// with nothing committed at all. That is not an empty answer: `repo_revision`
// is a member every Run's Provenance carries (§7, ADR-0043), so a Run that
// cannot name one has nothing to write and stops before it has written
// anything.
//
// A file set of nothing is dirty-free by construction, and correctly: a Run
// that read no artefact — which no Run does, every Run being a Run of a
// Procedure (ADR-0036) — has no code to have moved.
func Read(repoRoot string, read []File) (Facts, error) {
	git := repository(repoRoot)

	head, err := git.text("rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err != nil || head == "" {
		return Facts{}, fmt.Errorf("the repository names no revision: HEAD resolves to no commit, so there is nothing for a Run's provenance to record")
	}

	dirty, err := git.differs(head, read)
	if err != nil {
		return Facts{}, err
	}
	return Facts{Head: head, Dirty: dirty}, nil
}

// differs says whether any of the files differs from what the commit holds at
// its path, an untracked file being one that differs from nothing there.
//
// It is one `ls-tree` over the paths rather than one call per file: the answer
// is a property of the set, and a Procedure spanning a dozen artefacts should
// not cost a dozen subprocesses.
func (g gitRepository) differs(commit string, files []File) (bool, error) {
	if len(files) == 0 {
		return false, nil
	}

	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	// -z so that a path git would otherwise quote arrives whole, and the
	// pathspec named explicitly so the listing is the Run's file set rather
	// than the repository's.
	listing, err := g.run(append([]string{"ls-tree", "-z", commit, "--"}, slices.Sorted(slices.Values(paths))...)...)
	if err != nil {
		return false, err
	}
	records, err := treeRecords(listing)
	if err != nil {
		return false, err
	}

	committed := map[string]string{}
	for _, record := range records {
		committed[record.path] = record.object
	}

	for _, file := range files {
		if committed[file.Path] != Blob(file.Bytes) {
			return true, nil
		}
	}
	return false, nil
}

// treeRecord is one line of an `ls-tree -z` listing: what git holds at a path,
// what kind of object it is, and where it sits.
//
// The kind is read rather than assumed. A submodule is a `commit` record naming
// an object this repository does not hold, and under `-r` a tree does not
// appear at all — so a caller that wants blobs says so and a repository holding
// either still reads.
type treeRecord struct {
	kind, object, path string
}

// treeRecords reads an `ls-tree -z` listing into its records.
//
// It is one reader for both listings this package asks for — the file set a Run
// read, and the artefacts one revision held — because they are one format, and
// a second parser of it is a second place for the same record to be read wrong.
func treeRecords(listing []byte) ([]treeRecord, error) {
	var records []treeRecord
	for _, line := range nulSeparated(listing) {
		meta, path, named := strings.Cut(line, "\t")
		fields := strings.Fields(meta)
		if !named || len(fields) != 3 {
			return nil, fmt.Errorf("git ls-tree wrote %q, which is not a <mode> <type> <object>\\t<path> record", line)
		}
		records = append(records, treeRecord{kind: fields[1], object: fields[2], path: path})
	}
	return records, nil
}

// gitRepository is one repository as this package reaches it: where it sits,
// and the environment its git subprocesses run with.
type gitRepository struct {
	root string
	env  []string
}

// repository resolves the repository at repoRoot. There is no existence check
// here and none is wanted: every caller has already resolved the root and — the
// Store being a branch of the same repository — has already had a git repository
// answer for it, so a second stat would be a second opinion about a fact one
// call up (§9).
func repository(repoRoot string) gitRepository {
	return gitRepository{root: repoRoot, env: environment()}
}

// run runs one git command in the repository and returns its stdout untouched.
// Both of git's streams are captured: stdout is a listing this package reads,
// and git's narration must never reach hyper's own stderr, which the corpus
// compares byte for byte (§9).
func (g gitRepository) run(args ...string) ([]byte, error) {
	return g.with(nil, args...)
}

// with runs one git command with something on its stdin, which is what the one
// batch read here needs and what nothing else here does. It is the whole of
// run, so a command reading stdin and one that does not run the same way — the
// same directory, the same environment, and git's narration captured either
// way.
func (g gitRepository) with(stdin []byte, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	cmd.Env = g.env
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// text runs a git command whose whole answer is a line to read back, and
// returns it with the surrounding whitespace off.
func (g gitRepository) text(args ...string) (string, error) {
	out, err := g.run(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// environment is the process's own, less the variables that would redirect git
// at another repository — internal/git states that rule and holds it for both
// packages that start a git subprocess — and with lazy fetching off.
//
// No identity and no date is added: this package writes no object and makes no
// commit, which are the two things internal/store's own environment exists for.
//
// **Lazy fetching is off on every subprocess this package starts**, which is
// how *a review resolves no credential, reaches no network, and invokes
// nothing* is enforced rather than left to hold by habit (§8, ADR-0071). On a
// partial clone an ordinary object read is a lazy fetch, so a package whose
// whole subject is reading the code branch would reach a remote without a line
// of hyper intending to, and the failure would arrive as latency rather than as
// an absence anyone named. It is set here, on the environment every read below
// is run with, rather than at each call: a reader added later inherits the rule
// instead of having to remember it.
//
// The switch itself is internal/git's, for the reason the redirecting list is:
// both packages that start a git subprocess read objects and both are bound by
// the same rule, and one spelled in one copy and not the other would leave one
// of them reaching the network and the other not.
func environment() []string {
	return append(git.Inheritable(os.Environ()), git.NoLazyFetch)
}

// nulSeparated reads a listing git wrote NUL-separated, which is how the one
// listing here is asked for: a path git would otherwise quote arrives whole,
// and a path carrying a newline cannot be mistaken for two.
func nulSeparated(listing []byte) []string {
	trimmed := strings.TrimSuffix(string(listing), "\x00")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\x00")
}
