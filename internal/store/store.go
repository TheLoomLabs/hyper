// Package store is hyper's record: the orphan branch §7 states, and git as the
// subprocess that reads and writes it.
//
// The Store is a branch of the repository the artefacts sit in, written by
// every environment that runs — the laptop and the runner alike (ADR-0006).
// Writing to it invokes no Operation, consumes no Capability and passes no
// two-key check: it sits beneath the layer Providers exist at, and it is not a
// Target.
//
// What this milestone has landed is the branch's creation (issue #126) and the
// canonical encoding every file on it is written in (issue #127). The read
// half, the Head, the Journal, the path grammar and the push retry are the
// milestone's later tickets; the git layer they all go through is here already,
// unexported, and stays that way until a caller outside this package earns it.
//
// The encoder is the same shape: §7 states rules no command in this milestone
// can reach, so they are verified at this package's own seam rather than
// through a command that does not exist yet.
package store

import "time"

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

// IntroductionPath is where the branch introduces itself, and it is the one
// path in the whole Store that carries no Run id: every other path names the
// Run that wrote it, and this file is written by no Run (§12, ADR-0076).
const IntroductionPath = "STORE.md"

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
		// fetched rather than re-created, and nothing is written.
		if err := repo.fetchRef(RemoteName, Ref); err != nil {
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
