// Package store is hyper's record: the orphan branch §7 states, and git as the
// subprocess that reads and writes it.
//
// The Store is a branch of the repository the artefacts sit in, written by
// every environment that runs — the laptop and the runner alike (ADR-0006).
// Writing to it invokes no Operation, consumes no Capability and passes no
// two-key check: it sits beneath the layer Providers exist at, and it is not a
// Target.
//
// What milestone 4 landed is the branch's creation (issue #126), the
// canonical encoding every file on it is written in (issue #127), the path
// grammar every file on it is named by (issue #128), the five shapes it holds
// (issue #129) — a Record version, run.json, a Step file, outcome.json and a
// closing write, each with a schema version of its own — the read half (issue
// #130): the sync that puts the branch in hand, the series the branch holds,
// the Head derived from a listing and the case fold a collision is decided
// under — the removal (issue #131): Compaction's predicate, the one commit it
// writes, and the push that re-applies an unpushed set of path operations onto
// a fetched tip — and the Journal reader (issue #132): the entries the branch
// holds, the classification that says how each one ended, the Disposition read
// from a file or from a silence, and the backward scan through the date
// partitions. The git layer they all go through is here already, unexported,
// and stays that way until a caller outside this package earns it.
//
// The Journal reader had no CLI consumer in milestone 4 by construction. It is
// what milestone 5's Run and milestone 8's renderings both stand on, and it is
// here because the Journal is milestone 4's; `hyper run` is now the first
// caller of the backward scan, of Concluded and of the Head derivation alike.
//
// The shapes are the encoder's own case: §7 states rules no command reached
// while they were written, so they are verified at this package's own seam
// rather than through a command that did not exist yet.
package store

import (
	"errors"
	"time"
)

// The branch, and the ref it is. The name is fixed rather than chosen: there is
// no setting for it, no flag, and no file it could be configured from
// (ADR-0014). One repository has one Store, and finding it is knowing the name.
//
// The two are spelled once and derived, because `git checkout hyper-store` is
// what §7 promises a reader and `refs/heads/hyper-store` is what every plumbing
// call names — and the two must not be able to come apart.
const (
	BranchName = "hyper-store"
	Ref        = "refs/heads/" + BranchName
)

// RemoteName is the remote the Store is looked for on and pushed to. It is
// `origin` and it is not configurable, for BranchName's own reason.
const RemoteName = "origin"

// trackingRef is where a fetch puts what the remote holds: the ordinary
// remote-tracking ref, which is also what a `git clone` of a repository that
// already had a Store leaves behind.
//
// A fetch lands here rather than on the branch itself because the two are two
// facts — what the remote holds, and what this clone holds — and they part
// exactly where a Run wrote commits it could not push. Pointing the branch at
// what came is a separate act, and it happens only where it loses nothing (§7).
const trackingRef = "refs/remotes/" + RemoteName + "/" + BranchName

// Introduction is STORE.md, entire. It is written once, when the Store is
// created, and never again — a second `init` that rewrote it would be the one
// rewrite append-only forbids, arriving through the file that looks least
// dangerous to touch (§7, §12, ADR-0011).
//
// §7 fixes its three claims and not its words: that every other file on the
// branch is machine-written, that the branch is `hyper`'s account of the world
// rather than part of it, and that editing it by hand is editing evidence. What
// it carries beyond them is nothing — no schema version, no timestamp, no
// repository-specific fact — so it is byte-identical in every repository that
// has ever run `hyper store init`, which is also what makes it a golden.
const Introduction = "# The `hyper` Store\n" +
	"\n" +
	"This branch is the record. Every other file on it is machine-written: `hyper`\n" +
	"put it there, no file here was authored by hand, and none is meant to be.\n" +
	"\n" +
	"The branch is `hyper`'s account of the world, not part of it. Nothing on it is\n" +
	"configuration, nothing on it is reviewed, and nothing on it changes what a Run\n" +
	"does — it is what the Runs that have already happened left behind.\n" +
	"\n" +
	"Editing it by hand is editing evidence.\n"

// commitMessage is what `git log` on the branch says about its root. The branch
// is `hyper`'s account of the world and its commits are part of the account a
// human reads there (§7), so the root commit says what it is rather than
// carrying a tool's boilerplate.
const commitMessage = "Create the hyper Store"

// Initialised is what Init did: whether it created the branch in this
// repository, and whether it pushed it to the remote.
//
// The two are separate answers rather than one summary because they are the two
// facts the command's row carries, and either can stand without the other: a
// branch fetched from a remote that already held it is created here and pushed
// nowhere, and a repository with no remote configured creates one and pushes
// nothing.
type Initialised struct {
	// Created says this call created the Store: it minted the parentless
	// root and wrote STORE.md into it. It is false in both of the other two
	// cases, because in both of them the Store already existed and no file
	// was written — the branch was already here, or it was on the remote and
	// came down from there.
	Created bool
	// Pushed says the branch went to the remote. It is only ever true of the
	// root this call minted: a branch fetched from the remote is already
	// there, and a branch that was already local is not looked at.
	Pushed bool
}

// Init creates the Store, and does nothing else. No configuration is written,
// no example Definition is scaffolded, and no file in the working tree is
// touched — the branch is a parentless commit built from git objects and
// nothing about it is ever checked out, so it runs against a dirty tree like
// any read command (§9, ADR-0075).
//
// It looks before it creates, and the order is the load-bearing rule (§7,
// ADR-0074): the local ref, then — where a remote is configured — `origin`'s. A
// branch already here is created again by nothing; a branch on `origin` and not
// here is fetched, the tip and no history, the ref named explicitly; and only
// where neither holds it is a parentless root built. Skipping that second look
// makes two clones each mint an orphan root, which produces two histories that
// can never fast-forward into one another and a second operator whose every
// push fails forever with nothing to diagnose it by.
//
// Creating is not the whole of what it does, and the second half is the one an
// operator cannot perform any other way: **where a remote is configured and does
// not hold the branch, it is pushed there** — including where the branch was
// already local and this call created nothing. A runner's clone never holds the
// Store and fetches it from the remote (§7), so a Store that exists only on the
// laptop that ran `init` refuses every scheduled Run forever, and no command in
// §9's tree other than this one would ever send it. That case is reachable
// without anybody doing anything wrong: an `init` whose push was rejected leaves
// exactly that state, and if a second `init` were a no-op on finding the branch
// underfoot there would be no way back from it at all.
//
// The cost is stated rather than hidden: where a remote is configured, this
// reaches it on every invocation, so a second `init` on a laptop with no network
// answers that the world resisted rather than that there is already a Store.
// That is the honest answer — the postcondition is a Store here *and* on the
// remote, and only one of the two could be checked.
//
// It does not compare the two branches, only ask whether the remote has one. A
// local branch ahead of the remote's is a sync rather than a creation, and
// belongs to the Run that wrote the commits (§7).
//
// now is the clock the caller threaded, and both of the commit's dates come
// from it: a fixture's branch is then reproducible, and `git log` on the Store
// is honest about when the record began.
//
// It answers ErrNoRepository where repoRoot holds no git repository, which its
// caller reads as a usage error; every other error is the world resisting.
func Init(repoRoot string, now time.Time) (Initialised, error) {
	repo, err := open(repoRoot, now)
	if err != nil {
		return Initialised{}, err
	}

	local, err := repo.holdsRef(Ref)
	if err != nil {
		return Initialised{}, err
	}
	remote, err := repo.hasRemote(RemoteName)
	if err != nil {
		return Initialised{}, err
	}

	// With no remote configured there is one question and one act: the
	// branch is here, or it is built. Nothing reaches a network on this arm,
	// which is what a repository that has never had a remote deserves.
	if !remote {
		if local {
			return Initialised{}, nil
		}
		if err := repo.createStore(); err != nil {
			return Initialised{}, err
		}
		return Initialised{Created: true}, nil
	}

	held, err := repo.remoteHoldsRef(RemoteName, Ref)
	if err != nil {
		return Initialised{}, err
	}

	switch {
	case local && held:
		// Both sides hold it. The caller asked for there to be a Store
		// and there is one, in both of the places one has to be.
		return Initialised{}, nil

	case local:
		// The Store is here and nowhere else, which is the state every
		// scheduled Run reads as no Store at all. Nothing is created and
		// the branch goes out.
		if err := repo.pushRef(RemoteName, Ref); err != nil {
			return Initialised{}, err
		}
		return Initialised{Pushed: true}, nil

	case held:
		// The shape every runner's fresh clone is in. The branch is
		// fetched rather than re-created, and nothing is written. The
		// depth is bringBranch's decision and not this command's: an
		// ordinary `git clone` holds the Store's history under a
		// remote-tracking ref, and `init` is not a licence to cut it.
		if err := repo.bringBranch(); err != nil {
			return Initialised{}, err
		}
		return Initialised{}, nil
	}

	if err := repo.createStore(); err != nil {
		return Initialised{}, err
	}
	if err := repo.pushRef(RemoteName, Ref); err != nil {
		// The root stands locally, which is what the error leaves behind
		// — and what the next `store init` sends, this command having
		// just become the way back from here.
		return Initialised{Created: true}, err
	}
	return Initialised{Created: true, Pushed: true}, nil
}

// createStore builds the Store's first commit and points the branch at it: one
// blob, a tree holding it and nothing else, and a commit with no parent.
// Nothing is checked out and no working tree is touched at any point — the
// bytes go straight into the object database and the tree is assembled from
// their ids (ADR-0075).
func (g repository) createStore() error {
	blob, err := g.writeBlob([]byte(Introduction))
	if err != nil {
		return err
	}
	tree, err := g.writeTree([]treeEntry{{path: IntroductionPath, blob: blob}})
	if err != nil {
		return err
	}
	commit, err := g.commitParentlessTree(tree, commitMessage)
	if err != nil {
		return err
	}
	return g.createRef(Ref, commit)
}

// ErrAbsent is the Store that is not there: neither in the clone nor, where a
// remote is configured, on it. It is the condition a caller renders as
// `store-absent` (§12); this package holds no Run and renders no Refusal.
//
// The remedy it names is the only one there is. The branch is created by an
// explicit act and never by a Run, read-only Runs included, because a fetch
// that failed mid-flight and a branch that never existed look identical from
// the inside (§7).
var ErrAbsent = errors.New("the repository holds no " + BranchName + " branch; the Store is created by `hyper store init` and never by a Run")

// Store is a handle on the record as it stands: the repository the branch is a
// branch of, and the commit the handle was opened at.
//
// Every answer it gives is read out of that commit's tree, freshly, with
// nothing cached between two of them and nothing derived kept anywhere. §7
// permits local state under `.git/hyper/` that makes a Head lookup faster and
// states that no answer depends on one existing; this builds none, so *the
// files are authoritative* is what the code does rather than what it promises.
//
// The commit is resolved once so that two answers about one Store are answers
// about one Store. A Run that syncs, reads a series and then reads another is
// reading the branch it found, not the branch a second environment pushed to in
// between.
type Store struct {
	repo   repository
	commit string
	// now is the clock this handle was opened at, and it is a member rather
	// than a parameter of the one call that reads it: two answers about one
	// Store are answers at one instant, exactly as they are answers about
	// one commit. Retention is an age, so a Compaction measures against
	// this and never against a clock it reached for itself (§7, ADR-0034's
	// own rule one layer down).
	now time.Time
}

// Open answers a handle on the Store the clone holds, and ErrAbsent where it
// holds none.
//
// It reaches no network and never creates anything: putting the branch in hand
// is Sync's act and creating one is `hyper store init`'s. A caller that wants
// both syncs first and opens after, which is also the order in which their two
// failures mean different things.
//
// now is the clock the caller threaded. It is the environment every git
// subprocess in this package is run with, so the commits a write makes through
// this handle take both their dates from it, and it is the instant a Compaction
// measures retention against — one clock, read once, for everything this handle
// goes on to answer (§7).
func Open(repoRoot string, now time.Time) (*Store, error) {
	repo, err := open(repoRoot, now)
	if err != nil {
		return nil, err
	}
	held, err := repo.holdsRef(Ref)
	if err != nil {
		return nil, err
	}
	if !held {
		return nil, ErrAbsent
	}
	commit, err := repo.resolveRef(Ref)
	if err != nil {
		return nil, err
	}
	return &Store{repo: repo, commit: commit, now: now}, nil
}
