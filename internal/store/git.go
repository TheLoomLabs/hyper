package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// git is a subprocess, and it is the one external tool the binary requires
// (§7, §13). This file is the whole of that layer: the calls the Store is read
// and written with, and nothing about what the Store means.
//
// It is unexported and it stays that way. Milestone 4.6 adds the read half and
// 4.7 the removals and the push retry, and milestone 8 reads code-branch
// objects at a revision for §8's review range and may want a layer of its own —
// extracting this one then is a move with a caller, and doing it now is a
// package with none (issue #124).
//
// Every call here is plumbing: `hash-object`, `mktree`, `commit-tree`,
// `update-ref`, `show-ref`, `ls-remote`, `fetch` and `push`. Each is one git
// invocation and knows nothing about the Store; store.go composes them into the
// Store's own acts, which is why a method like createStore hangs off this file's
// type and lives in that one. There is no
// worktree, no temporary directory, no hidden checkout and no byte of Store
// content as an ordinary file on disk, so no command below reads or writes the
// working tree and no `git checkout` appears anywhere in the package
// (ADR-0075).

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

// fetchRef brings one ref down from the remote, the tip and no history.
//
// It is depth-1 and it is never filtered: no blob filter, no tree filter, no
// partial clone. The Store's history is never read and its content always is,
// so a filter would turn every version's `written_at` into a lazy fetch and
// make *a read-only Run proceeds offline* false wherever the network is (§7,
// ADR-0074). The depth is decided here, at the one moment there is nothing to
// preserve: this is the branch's arrival in the clone, and `hyper` never
// deepens a Store and never shortens one it did not create.
func (g repository) fetchRef(remote, ref string) error {
	_, err := g.run(nil, "fetch", "--quiet", "--depth=1", "--no-tags", "--", remote, ref+":"+ref)
	return err
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

// treeEntry is one file in a tree the package builds: where it sits, and the
// blob whose bytes stand there. Every entry is a regular file at 100644 — the
// Store holds files and no directories of its own making, no symlinks and
// nothing executable — so the mode is this package's constant rather than an
// entry's field (§12).
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
		fmt.Fprintf(&listing, "100644 blob %s\t%s\n", entry.blob, entry.path)
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
