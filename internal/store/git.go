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
)

// git is a subprocess, and it is the one external tool the binary requires
// (§7, §13). This file is the whole of that layer: the calls the Store is read
// and written with, and nothing about what the Store means.
//
// It is unexported and it stays that way. Milestone 4.7 adds the removals and
// the push retry, and milestone 8 reads code-branch objects at a revision for
// §8's review range and may want a layer of its own — extracting this one then
// is a move with a caller, and doing it now is a package with none (issue
// #124).
//
// Every call here is plumbing: `hash-object`, `mktree`, `commit-tree`,
// `update-ref`, `show-ref`, `ls-remote`, `fetch`, `push`, `rev-parse`,
// `ls-tree` and `cat-file`. Each is one git invocation and knows nothing about
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
// The process's environment is inherited rather than replaced, and that is the
// one place this layer is deliberately not hermetic. The git `hyper` shells out
// to is the same git that resolves the credential a checkout left behind (§7,
// §11), so its configuration, its credential helpers and its SSH agent are all
// reached the way the operator already set them up. What is overridden is
// overridden by being written last: os/exec keeps the final value of a repeated
// name, so the identity and the dates below win over anything the caller's
// environment happened to set.
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
	return append(inheritable(os.Environ()),
		"GIT_AUTHOR_NAME="+CommitName,
		"GIT_AUTHOR_EMAIL="+CommitEmail,
		"GIT_AUTHOR_DATE="+stamp,
		"GIT_COMMITTER_NAME="+CommitName,
		"GIT_COMMITTER_EMAIL="+CommitEmail,
		"GIT_COMMITTER_DATE="+stamp,
		"GIT_TERMINAL_PROMPT=0",
	)
}

// redirecting names the environment variables that decide *which* repository
// git acts on, rather than how it behaves while acting. They are dropped from
// the environment every call here is made with, because the repository is the
// one the repository root names and never one an ambient variable does.
//
// This is not hypothetical: git sets GIT_DIR and GIT_INDEX_FILE for every hook
// it runs, so a `hyper store init` invoked from a pre-commit hook would inherit
// them and write its branch through whatever they point at — silently, and into
// a repository the caller never named. Nothing else in the environment is
// touched, the credential helpers and the SSH agent being exactly what §7 says
// this git is reached for.
var redirecting = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_COMMON_DIR",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

// inheritable is the process's environment with those dropped. A variable is
// matched on the name before its first "=", which is what an environment entry
// is; anything malformed enough to have none is passed through untouched rather
// than guessed at.
func inheritable(env []string) []string {
	kept := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, named := strings.Cut(entry, "=")
		if named && slices.Contains(redirecting, name) {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
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
	_, err := g.run(nil, "fetch", "--quiet", "--depth=1", "--no-tags", "--", remote, src+":"+dst)
	return err
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
	_, err := g.run(nil, "fetch", "--quiet", "--no-tags", "--", remote, src+":"+dst)
	return err
}

// carriesCommitsOutside says whether ref holds commits that other does not —
// the question *is this ref behind, or has it got something of its own*.
//
// It is `rev-list ref ^other` and a look at whether anything came back, rather
// than `merge-base --is-ancestor`, which answers the same question and is a
// decade younger than every other call in this file (§13). An empty answer is
// *ref is an ancestor of other*, which is a fast-forward.
func (g repository) carriesCommitsOutside(ref, other string) (bool, error) {
	listing, err := g.text(nil, "rev-list", ref, "^"+other)
	if err != nil {
		return false, err
	}
	return listing != "", nil
}

// resolveRef answers the commit a ref stands at. It is `^{commit}` rather than
// the bare name so that a ref pointing at anything else is a fault here rather
// than a tree read that answers nothing further down.
func (g repository) resolveRef(ref string) (string, error) {
	return g.text(nil, "rev-parse", "--verify", ref+"^{commit}")
}

// treeFile is one file on the branch as a read finds it: where it sits, and the
// blob standing there. It is treeEntry read rather than written, and the two are
// separate types because a write names a blob it has just made and a read names
// one it is about to open.
type treeFile struct {
	path string
	blob string
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
func (g repository) listTree(commit, prefix string) ([]treeFile, error) {
	listing, err := g.run(nil, "ls-tree", "-r", "-z", commit, "--", prefix)
	if err != nil {
		return nil, err
	}

	var files []treeFile
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
		files = append(files, treeFile{path: path, blob: fields[2]})
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

// treeEntry is one file in a tree the package builds: where it sits, and the
// blob whose bytes stand there.
type treeEntry struct {
	path string
	blob string
}

// writeTree assembles a tree from entries and answers its id. It goes through
// `mktree`, which reads its entries on stdin and writes an object: there is no
// index file to point at, no `read-tree`, and nothing on disk at any point.
func (g repository) writeTree(entries []treeEntry) (string, error) {
	var listing strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&listing, "%s blob %s\t%s\n", fileMode, entry.blob, entry.path)
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
