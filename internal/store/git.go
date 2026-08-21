package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/git"
)

// git is a subprocess, and it is the one external tool the binary requires
// (§7, §13). This file is the whole of that layer: the calls the Store is read
// and written with, and nothing about what the Store means.
//
// It is unexported and it stays that way. Milestone 8 reads code-branch objects
// at a revision for §8's review range and may want a layer of its own —
// extracting this one then is a move with a caller, and doing it now is a
// package with none (issue #124).
//
// Every call here is plumbing: `hash-object`, `mktree`, `commit-tree`,
// `update-ref`, `show-ref`, `ls-remote`, `fetch`, `push`, `rev-parse`,
// `rev-list`, `log`, `ls-tree`, `diff-tree` and `cat-file`. Each is one git
// invocation and knows nothing about
// the Store; store.go, sync.go and series.go compose them into the Store's own
// acts, which is why a method like createStore hangs off this file's type and
// lives in one of those. There is no worktree, no temporary directory, no
// hidden checkout and no byte of Store content as an ordinary file on disk, so
// no command below reads or writes the working tree and no `git checkout`
// appears anywhere in the package (ADR-0075).

// repository is one repository as this package reaches it: where it sits, and
// the environment every git subprocess is run with. Both are resolved once, by
// open, so no call below decides for itself where it is standing.
type repository struct {
	root string
	env  []string
}

// ErrNoRepository is the repository root that holds no git repository. It is a
// sentinel rather than a message because it is the one fault here that is a
// usage error and not the world resisting: there is no branch to create and no
// repository to refuse on behalf of, so its caller exits 2 (§9, issue #124).
//
// #124 said this answers "on resolveRepoRoot's existing message", and it does
// not, deliberately. That message is the *walk* finding no git root and it ends
// *pass --repo-dir or set HYPER_REPO_DIR* — which is the one remedy that cannot
// apply here, since the only way to reach this fault at all is to have named a
// root already: with neither global set, the walk resolves the git root or the
// command never gets this far. A message telling a caller to do the thing they
// just did is worse than the code it carries.
//
// It names no path, which is the other half: it is written to stderr and
// compared byte for byte by the corpus, and a repository root is an absolute
// path that differs on every machine.
var ErrNoRepository = errors.New("the repository root holds no git repository; the Store is an orphan branch of the repository the artefacts sit in")

// open resolves the repository at repoRoot, or answers ErrNoRepository where
// there is none there.
//
// The test is a `.git` entry at the root itself and never a walk upwards, which
// is the same reading repository.FindGitRoot gives — and here it is
// load-bearing rather than tidy. The repository root is what --repo-dir named
// or what that walk already found, and asking git to resolve a toplevel from it
// would climb out of a directory the caller pointed at and create a branch in
// whatever repository lay above it (§9, issue #126). A worktree's `.git` is a
// file rather than a directory, so the entry is stat-ed for existence and not
// for its kind.
func open(repoRoot string, now time.Time) (repository, error) {
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
		return repository{}, ErrNoRepository
	}
	return repository{root: repoRoot, env: environment(now)}, nil
}

// The identity every commit `hyper` writes carries, author and committer alike.
//
// It is `hyper`'s own constant and is never read from the repository's git
// configuration, for two reasons that arrive from opposite ends (§7, issue
// #124): a runner whose checkout never set `user.email` would otherwise be
// unable to write the record at all, and *who ran something* is already the
// Journal's `trigger.actor` — a second, weaker copy of it on every commit would
// be a fact with two spellings and no reader.
//
// The address is under `.invalid`, which RFC 2606 reserves for exactly this: an
// address that is well-formed, obviously not a mailbox, and could never be
// delivered to by accident.
const (
	CommitName  = "hyper"
	CommitEmail = "hyper@hyper.invalid"
)

// environment is what every git subprocess is run with: the process's own, plus
// the identity and the two dates.
//
// The process's environment is inherited rather than replaced, less the
// variables that would redirect git at another repository — internal/git states
// that rule and holds it for both packages that start a git subprocess. What is
// overridden here is overridden by being written last: os/exec keeps the final
// value of a repeated name, so the identity and the dates below win over
// anything the caller's environment happened to set.
//
// A git date is whole seconds, so a clock carrying milliseconds reaches a
// commit truncated. It is written in git's own raw form — the seconds since the
// epoch and an explicit zero offset — because that is the one spelling no
// locale, no `TZ` and no date parser can read two ways.
//
// GIT_TERMINAL_PROMPT is off. `hyper` never prompts (ADR-0015), and a push that
// stopped to ask a scheduled runner for a password would hang rather than fail
// — which is the one failure mode a Cadence cannot recover from.
func environment(now time.Time) []string {
	stamp := fmt.Sprintf("%d +0000", now.Unix())
	return append(git.Inheritable(os.Environ()),
		"GIT_AUTHOR_NAME="+CommitName,
		"GIT_AUTHOR_EMAIL="+CommitEmail,
		"GIT_AUTHOR_DATE="+stamp,
		"GIT_COMMITTER_NAME="+CommitName,
		"GIT_COMMITTER_EMAIL="+CommitEmail,
		"GIT_COMMITTER_DATE="+stamp,
		"GIT_TERMINAL_PROMPT=0",
	)
}

// gitError is a git call that failed, carrying what was run and what git said
// about it. git's own stderr is the diagnosis in almost every case here — a
// rejected push, a remote that would not answer, an object that would not write
// — and a caller that swallowed it would leave an operator with an exit code
// and nothing else.
type gitError struct {
	args   []string
	stderr string
	err    error
}

func (e *gitError) Error() string {
	if trimmed := strings.TrimSpace(e.stderr); trimmed != "" {
		return fmt.Sprintf("git %s: %s: %s", strings.Join(e.args, " "), e.err, trimmed)
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.args, " "), e.err)
}

func (e *gitError) Unwrap() error { return e.err }

// run runs one git command in the repository and returns its stdout untouched.
//
// Both of git's streams are captured. stdout is an object id or a listing this
// package reads, and stderr is git's narration — *From origin*, *[new branch]* —
// which must never reach `hyper`'s own stderr: that stream is the command's
// narration and the corpus compares it byte for byte (§9).
func (g repository) run(stdin []byte, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	cmd.Env = g.env
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, &gitError{args: args, stderr: stderr.String(), err: err}
	}
	return stdout.Bytes(), nil
}

// text runs a git command whose whole answer is a line to read back — an object
// id, a ref listing — and returns it with the surrounding whitespace off.
func (g repository) text(stdin []byte, args ...string) (string, error) {
	out, err := g.run(stdin, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// holdsRef says whether the repository holds the named ref. It is the one call
// whose non-zero exit is an answer rather than a fault, so it does not go
// through run: `show-ref --verify --quiet` exits 1 for a ref that is not there
// and writes nothing, and anything else is a repository that could not be read
// at all.
func (g repository) holdsRef(ref string) (bool, error) {
	var stderr bytes.Buffer
	args := []string{"show-ref", "--verify", "--quiet", ref}
	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	cmd.Env = g.env
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, &gitError{args: args, stderr: stderr.String(), err: err}
}

// hasRemote says whether a remote of that name is configured. `git remote`
// lists the configured names and exits 0 whether there are any or none, so the
// question is answered by reading its listing rather than by an exit code — a
// remote that is not configured and a git that could not be run are two
// different answers, and only the second is a fault.
func (g repository) hasRemote(name string) (bool, error) {
	listing, err := g.run(nil, "remote")
	if err != nil {
		return false, err
	}
	for _, configured := range strings.Fields(string(listing)) {
		if configured == name {
			return true, nil
		}
	}
	return false, nil
}

// remoteHoldsRef asks the remote whether it holds the named ref. The ref is
// named explicitly rather than left to the remote's configured refspec, a
// checkout having left that pinned to the one ref it took (§7, §10, ADR-0071).
//
// An empty listing is *the remote does not hold it*; an error is the world
// resisting, and the two are never folded together — a remote that could not be
// reached, read as a remote holding nothing, is exactly how a second orphan
// root gets minted (ADR-0074).
func (g repository) remoteHoldsRef(remote, ref string) (bool, error) {
	listing, err := g.text(nil, "ls-remote", "--", remote, ref)
	if err != nil {
		return false, err
	}
	return listing != "", nil
}

// fetchShallow brings one ref down from the remote, the tip and no history.
//
// It is depth-1 and it is never filtered: no blob filter, no tree filter, no
// partial clone. The Store's history is never read and its content always is,
// so a filter would turn every version's `written_at` into a lazy fetch and
// make *a read-only Run proceeds offline* false wherever the network is (§7,
// ADR-0074). The depth is decided here, at the one moment there is nothing to
// preserve: this is the branch's arrival in the clone, and `hyper` never
// deepens a Store and never shortens one it did not create.
func (g repository) fetchShallow(remote, src, dst string) error {
	return g.fetch(remote, src, dst, "--depth=1")
}

// fetchIncremental brings one ref up to date and names no depth, which is the
// whole of what separates it from the call above.
//
// Naming no depth is what leaves a clone that holds the Store whole holding it
// whole, and leaves one that holds it shallow exactly as shallow as it was:
// git carries the shallow boundary forward rather than deciding it again, so
// `hyper` never deepens a Store and never shortens one it did not create
// (ADR-0074). It is unfiltered for the reason above.
func (g repository) fetchIncremental(remote, src, dst string) error {
	return g.fetch(remote, src, dst)
}

// fetch is the call both are: one ref, named explicitly on both sides rather
// than left to the remote's configured refspec — a checkout having left that
// pinned to the one ref it took (ADR-0071) — and whatever depth the caller
// named, which is the only thing that separates them.
//
// No filter is written here and none may be: the two above are the whole of how
// this package fetches, so *never a filtered fetch* is a fact about one line.
func (g repository) fetch(remote, src, dst string, depth ...string) error {
	args := append([]string{"fetch", "--quiet", "--no-tags"}, depth...)
	_, err := g.run(nil, append(args, "--", remote, src+":"+dst)...)
	return err
}

// isAncestor says whether ref is an ancestor of the commit named — the question
// *is this ref behind, so that moving it there loses nothing*.
//
// It is `rev-list ref ^of` and a look at whether anything came back, rather
// than `merge-base --is-ancestor`, which answers the same question and is a
// decade younger than every other call in this file (§13). An empty answer is
// *ref holds no commit that is not already there*, which is a fast-forward.
func (g repository) isAncestor(ref, of string) (bool, error) {
	listing, err := g.text(nil, "rev-list", ref, "^"+of)
	if err != nil {
		return false, err
	}
	return listing == "", nil
}

// resolveRef answers the commit a ref stands at. It is `^{commit}` rather than
// the bare name so that a ref pointing at anything else is a fault here rather
// than a tree read that answers nothing further down.
func (g repository) resolveRef(ref string) (string, error) {
	return g.text(nil, "rev-parse", "--verify", ref+"^{commit}")
}

// listTree lists every file under prefix in a commit's tree, recursively.
//
// It reads the tree objects and never a working tree: there is no checkout, no
// temporary directory and no byte of Store content on disk at any point
// (ADR-0075). `-z` is what makes the listing exact — git quotes an unusual path
// in its ordinary output, and a quoted path is a second spelling of a name this
// package builds and parses in one form.
//
// A prefix naming nothing answers an empty listing rather than a fault: a
// branch that holds no Record series at all is a Store with nothing recorded
// yet, which is the state every Store begins in (§7).
//
// Every entry is held to being a regular file, which is the read half of the
// rule writeTree states: the Store holds files at 100644, no directories of its
// own making, no symlinks and nothing executable (§12). Anything else on the
// branch was put there by something that is not `hyper`, and reading it as a
// Store file is giving it a meaning nothing gave it.
func (g repository) listTree(commit, prefix string) ([]treeEntry, error) {
	// An empty prefix is the whole tree and is spelled by naming no
	// pathspec at all: git refuses the empty string as one outright, and
	// substituting "." for it would be this file inventing a second
	// spelling of *everything* for the one caller that wants it.
	args := []string{"ls-tree", "-r", "-z", commit}
	if prefix != "" {
		args = append(args, "--", prefix)
	}
	listing, err := g.run(nil, args...)
	if err != nil {
		return nil, err
	}

	var files []treeEntry
	for _, entry := range strings.Split(strings.TrimSuffix(string(listing), "\x00"), "\x00") {
		if entry == "" {
			continue
		}
		// `<mode> SP <type> SP <object> TAB <path>`, which is git's
		// oldest listing and the one every version of it writes.
		head, path, named := strings.Cut(entry, "\t")
		fields := strings.Fields(head)
		if !named || len(fields) != 3 {
			return nil, fmt.Errorf("git ls-tree wrote %q, which is not <mode> <type> <object> and a path", entry)
		}
		if fields[0] != fileMode || fields[1] != "blob" {
			return nil, fmt.Errorf("%q is a %s at mode %s: the Store holds regular files and nothing else (§12)", path, fields[1], fields[0])
		}
		files = append(files, treeEntry{path: path, blob: fields[2]})
	}
	return files, nil
}

// readBlobs answers the bytes of every blob named, in the order they were
// named.
//
// It is one `cat-file --batch` for the whole listing rather than one call per
// file, which is the difference between a Head lookup costing a process and a
// series of five hundred versions costing five hundred of them. Ordering a
// series opens every version of it (§7), so this is the call every read in the
// package ends at.
func (g repository) readBlobs(objects []string) ([][]byte, error) {
	if len(objects) == 0 {
		return nil, nil
	}
	batch, err := g.run([]byte(strings.Join(objects, "\n")+"\n"), "cat-file", "--batch")
	if err != nil {
		return nil, err
	}

	contents := make([][]byte, 0, len(objects))
	rest := batch
	for _, object := range objects {
		// `<object> SP <type> SP <size> LF`, the content, and one LF.
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

// pushRef sends one ref to the remote, named explicitly on both sides for
// ls-remote's own reason.
func (g repository) pushRef(remote, ref string) error {
	_, err := g.run(nil, "push", "--quiet", "--", remote, ref+":"+ref)
	return err
}

// writeBlob writes one file's bytes into the object database and answers its
// id. The content is handed over on stdin rather than named as a path, so
// nothing is written to disk to be hashed and no clean filter the repository
// configures can reach it: what goes into a tree is the bytes the caller
// supplied (§7, ADR-0075).
func (g repository) writeBlob(content []byte) (string, error) {
	return g.text(content, "hash-object", "-w", "--stdin")
}

// fileMode is the mode every entry in a Store tree carries. The Store holds
// files and no directories of its own making, no symlinks and nothing
// executable, so it is this package's constant rather than an entry's field —
// written by writeTree and required by listTree (§12).
const fileMode = "100644"

// treeEntry is one file in a Store tree: where it sits, and the blob whose
// bytes stand there. It is what writeTree is handed and what listTree answers,
// one type because it is one fact — a tree is a set of these, and which
// direction it was travelling in is the caller's business.
type treeEntry struct {
	path string
	blob string
}

// treeMode is the mode a subtree entry carries. It stands beside fileMode for
// the same reason: the Store holds files, and the directories above them are
// git's own structure rather than anything §12 names, so neither mode is ever a
// value a caller supplies.
const treeMode = "040000"

// writeTree assembles a tree from entries whose paths may carry slashes, and
// answers the root tree's id.
//
// It goes through `mktree`, which reads its entries on stdin and writes an
// object: there is no index file to point at, no `read-tree`, and nothing on
// disk at any point (ADR-0075). `mktree` writes one tree level and refuses a
// path with a slash in it outright, so the nesting is done here — the entries
// are grouped by their first segment and each group becomes a subtree of its
// own, bottom up, until one tree stands over all of them.
//
// The alternative is a temporary index file, which is what a `read-tree` plus
// `update-index` plus `write-tree` costs. It is rejected for what it is rather
// than for what it does: an index is a file on disk holding the shape of the
// Store, which is the one thing this package promises never to write. The
// recursion below costs one `mktree` per directory in the tree and leaves
// nothing behind at all.
//
// Entry order is not normalised here because `mktree` normalises it — git's
// tree ordering sorts a directory as though its name ended in a slash, which is
// exactly the rule an implementation gets wrong by hand.
func (g repository) writeTree(entries []treeEntry) (string, error) {
	// The listing is built in one pass and the subtrees in a second, so
	// that a directory named twice is one recursion rather than two: the
	// order the names were first seen decides nothing (mktree sorts), and
	// holding them is what keeps the walk deterministic under test.
	var files []treeEntry
	var dirs []string
	nested := map[string][]treeEntry{}
	for _, entry := range entries {
		name, rest, below := strings.Cut(entry.path, "/")
		if !below {
			files = append(files, treeEntry{path: name, blob: entry.blob})
			continue
		}
		if _, seen := nested[name]; !seen {
			dirs = append(dirs, name)
		}
		nested[name] = append(nested[name], treeEntry{path: rest, blob: entry.blob})
	}

	var listing strings.Builder
	for _, entry := range files {
		fmt.Fprintf(&listing, "%s blob %s\t%s\n", fileMode, entry.blob, entry.path)
	}
	for _, dir := range dirs {
		subtree, err := g.writeTree(nested[dir])
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&listing, "%s tree %s\t%s\n", treeMode, subtree, dir)
	}
	return g.text([]byte(listing.String()), "mktree")
}

// commitParentlessTree commits a tree with no parent — the orphan root a Store
// begins at — and answers the commit's id. The identity and both dates come
// from the environment open resolved, so nothing about the commit is read from
// the repository or from the wall clock.
func (g repository) commitParentlessTree(tree, message string) (string, error) {
	return g.text(nil, "commit-tree", tree, "-m", message)
}

// commitOnto commits a tree with one parent and answers the commit's id. It is
// every commit the Store takes after its root: a Run's write, a Compaction's
// removal, and a commit re-applied onto a fetched tip.
//
// The identity and both dates come from the environment open resolved, which is
// what makes a fixture's branch reproducible and `git log` on the Store honest
// about when a Compaction ran (§7).
func (g repository) commitOnto(tree, parent, message string) (string, error) {
	return g.text(nil, "commit-tree", tree, "-p", parent, "-m", message)
}

// commitsSince lists the commits a ref holds that another does not, oldest
// first — *every local commit the remote does not hold*, which is the set §7
// defines the push re-application over.
//
// Oldest first because they are re-applied in the order they were written: a
// path a later commit removed and an earlier one wrote must end up removed, and
// replaying them in the order they happened is what decides that without a
// rule of its own.
func (g repository) commitsSince(ref, excluding string) ([]string, error) {
	listing, err := g.text(nil, "rev-list", "--reverse", ref, "^"+excluding)
	if err != nil {
		return nil, err
	}
	return strings.Fields(listing), nil
}

// messageOf answers a commit's message, whole and untouched.
//
// %B is the subject and the body together, exactly as the commit carries them.
// A re-applied commit keeps the message the commit it re-applies was written
// with, because `git log` on the Store is the account of what happened to it
// (§7, §13) — and an account that lost a Run's own line to a rebase would be
// the record's own history rewritten by a retry.
func (g repository) messageOf(commit string) (string, error) {
	message, err := g.run(nil, "log", "-1", "--format=%B", commit)
	if err != nil {
		return "", err
	}
	// `git log` ends %B with a newline of its own, and commit-tree adds one
	// where the message lacks it; trimming the trailing run keeps a message
	// that survives one round trip identical to itself.
	return strings.TrimRight(string(message), "\n"), nil
}

// pathOperation is one thing a commit did to one path: wrote it, or removed it.
// It is the grain §7's re-application is defined at — *every path operation in
// every local commit the remote does not hold* — and the noun is operation
// rather than write because `compact` is the first thing in the tool that
// removes (issue #131).
type pathOperation struct {
	path string
	// blob is what stands at the path afterwards, and is empty where the
	// operation is the removal. A removal is the absence of a blob rather
	// than a flag beside one, so there is no state where the two disagree.
	blob string
}

// removes says the operation takes the path off the tree.
func (op pathOperation) removes() bool { return op.blob == "" }

// applyOnto answers the tree a set of path operations makes of a commit's tree.
//
// A write names the blob the operation already carries: the object is in the
// database, so applying a write is naming an id in a tree and never hashing
// bytes again. **A removal whose path the tree no longer holds is a no-op** —
// which is what a Compaction is built out of, and what makes one re-appliable
// onto a tip another environment has already compacted (§7, issue #131).
//
// It is the one place a Store tree is derived from another, and both callers
// are the same act at different grains: `compact` applies the removals it
// decided to the commit it read, and a rejected push applies each unpushed
// commit's operations to the tip it fetched.
func (g repository) applyOnto(commit string, operations []pathOperation) (string, error) {
	files, err := g.listTree(commit, "")
	if err != nil {
		return "", err
	}

	held := make(map[string]string, len(files))
	for _, entry := range files {
		held[entry.path] = entry.blob
	}
	for _, operation := range operations {
		if operation.removes() {
			delete(held, operation.path)
			continue
		}
		held[operation.path] = operation.blob
	}

	entries := make([]treeEntry, 0, len(held))
	for path, blob := range held {
		entries = append(entries, treeEntry{path: path, blob: blob})
	}
	// Sorted so that one tree is built from one listing however the map was
	// walked. `mktree` normalises the order it is given, so this decides
	// nothing about the object — it decides that the calls are the same
	// calls twice, which is what a test of them can hold.
	slices.SortFunc(entries, func(a, b treeEntry) int { return strings.Compare(a.path, b.path) })
	return g.writeTree(entries)
}

// operationsIn answers what one commit did to the tree, path by path, against
// its parent — or against nothing at all, where it is a root.
//
// It is `diff-tree` in git's raw form rather than `--name-status`, because the
// object id on the right-hand side is exactly what a re-application needs: the
// blob is already written, so replaying a write is naming it in a tree and
// never hashing bytes again.
func (g repository) operationsIn(commit string) ([]pathOperation, error) {
	// --no-renames because a rename record carries two paths rather than
	// one and this parser reads pairs: rename detection is a repository
	// setting, so it is turned off here rather than assumed off.
	listing, err := g.run(nil, "diff-tree", "-r", "-z", "--no-commit-id", "--no-renames", "--root", commit)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSuffix(string(listing), "\x00")
	if trimmed == "" {
		return nil, nil
	}
	// Every record is a header and a path, so an odd count is a listing
	// that stopped mid-record. It is a fault rather than a record dropped:
	// this answer decides what a retry re-applies, and a re-application
	// short one operation is a write or a removal silently lost.
	records := strings.Split(trimmed, "\x00")
	if len(records)%2 != 0 {
		return nil, fmt.Errorf("git diff-tree answered %d fields for %s, which is not a whole number of raw diff records", len(records), commit)
	}

	operations := make([]pathOperation, 0, len(records)/2)
	for i := 0; i < len(records); i += 2 {
		// `:<srcmode> <dstmode> <srcsha> <dstsha> <status>` and then the
		// path, each NUL-terminated. A status of D leaves the
		// destination id all zeros, which is the removal.
		fields := strings.Fields(records[i])
		if len(fields) != 5 || !strings.HasPrefix(fields[0], ":") {
			return nil, fmt.Errorf("git diff-tree wrote %q, which is not a raw diff record", records[i])
		}
		operation := pathOperation{path: records[i+1]}
		if fields[4] != "D" {
			operation.blob = fields[3]
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

// createRef points a ref at a commit, and only where the ref is not there
// already. The empty old value is the whole of that: `update-ref` takes it as
// *this ref must not currently exist* and fails the update otherwise, so a
// branch that appeared between the look and the write is a failure rather than
// a silent overwrite of somebody else's root.
func (g repository) createRef(ref, commit string) error {
	_, err := g.run(nil, "update-ref", ref, commit, "")
	return err
}

// moveRef points a ref that is already there at a commit, and only where it
// still stands where the caller last saw it. The old value is named for
// createRef's own reason one state over: a ref that moved between the look and
// the write is a second writer, and overwriting it is the act append-only
// forbids arriving at the ref rather than at a file.
func (g repository) moveRef(ref, commit, from string) error {
	_, err := g.run(nil, "update-ref", ref, commit, from)
	return err
}
