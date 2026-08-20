package store

import (
	"errors"
	"fmt"
)

// The push, and the re-application a rejected one goes through (§7, ADR-0075,
// ADR-0076).
//
// A push rejected as non-fast-forward fetches, re-applies the unpushed set onto
// the fetched tip and retries, three times. It is not `git rebase`, which needs
// a worktree there is none of: `hyper` knows exactly which paths are unpushed —
// every path operation in every local commit the remote does not hold, this
// caller's and any earlier one's that stopped before it could push — and
// re-applies that set onto the fetched tip's tree.
//
// §7 describes that set in terms of writes because until now every write was
// one. `compact` is the first thing in the tool that **removes**, so the set is
// a set of path *operations*, and a removal whose path the fetched tip no
// longer holds is a no-op rather than a conflict (issue #131). The cleanliness
// argument is unchanged: every path in the Store carries the id of the Run that
// wrote it and two Runs cannot mint one id, so two writers cannot contend for
// one path (ADR-0076).
//
// It re-applies commit by commit rather than collapsing the unpushed set into
// one commit, and that is what keeps `git log` on the branch the account §7
// says it is: a Run's own commit and a Compaction's own commit each survive a
// retry with the message they were written with.

// pushAttempts is how many times a push is made before the condition is
// reported: three, which is §7's own number. It is a count of pushes and not of
// re-applications — the third rejection is the end, and nothing is rebuilt
// after it that nothing would send.
const pushAttempts = 3

// ErrPushExhausted is the push that could not complete in three attempts: the
// remote moved under each of them, and what this clone holds is a branch ahead
// of the one on the remote.
//
// It is a condition this package reports and never a code it maps. Where the
// caller is a Run this is `failed` at 75; where the caller is `compact` or
// `store init` — commands with no outcome triple to map onto — it is the world
// resisting at 1 (§9, §12, issue #131).
//
// What it leaves behind is stated rather than hidden: every commit stands
// locally, re-applied onto the last tip that was fetched, and goes out with the
// next push that gets through. Nothing is unwound.
var ErrPushExhausted = errors.New("the push was rejected three times running; what was written stands locally and goes out with the next push")

// publish sends the Store branch to the remote, where there is one.
//
// A repository with no remote configured reaches no network at all and is not a
// failure: a repository that has never had one is not a repository whose Store
// could not be published.
func (g repository) publish() error {
	remote, err := g.hasRemote(RemoteName)
	if err != nil || !remote {
		return err
	}
	return g.pushStore()
}

// pushStore pushes the branch, re-applying and retrying where the remote moved
// under it.
//
// The two ways a push fails are told apart by asking the remote rather than by
// reading git's words: the branch is fetched, and whether the fetched tip holds
// anything this clone does not is the whole question. A remote that moved is
// the non-fast-forward and is re-applied; a remote that did not is a push that
// failed for its own reason — a credential, a network, a hook — and that error
// stands, because retrying it would be three copies of one failure and a
// message naming the wrong cause.
func (g repository) pushStore() error {
	for attempt := 1; ; attempt++ {
		err := g.pushRef(RemoteName, Ref)
		if err == nil {
			return nil
		}
		if attempt == pushAttempts {
			return fmt.Errorf("%w: %s", ErrPushExhausted, err)
		}

		moved, asked := g.remoteMoved()
		if asked != nil {
			// Both reaches at the remote failed, and the operator needs
			// both: the push error is what they asked for and stays the
			// error, and the fetch that could not diagnose it is stated
			// beside it rather than dropped.
			return fmt.Errorf("%w; and the fetch that followed it: %v", err, asked)
		}
		if !moved {
			return err
		}
		if err := g.reapply(); err != nil {
			return err
		}
	}
}

// remoteMoved fetches and answers whether the remote holds a commit this clone
// does not — which is *the push was rejected as non-fast-forward*, asked of the
// world rather than inferred from a message.
//
// The fetch goes through bringBranch, so the depth rules are the sync's and not
// a second decision made here: a clone that holds the branch fetches
// incrementally and one that does not takes the tip (§7, ADR-0074). It cannot
// move the local ref out from under the commits being re-applied — adopt leaves
// a branch holding commits the remote does not exactly where it is, which is
// the case this is only ever reached in.
func (g repository) remoteMoved() (bool, error) {
	if err := g.bringBranch(); err != nil {
		return false, err
	}
	behind, err := g.isAncestor(trackingRef, Ref)
	if err != nil {
		return false, err
	}
	return !behind, nil
}

// reapply rebuilds every local commit the remote does not hold on top of the
// fetched tip, and points the branch at the result.
//
// Each commit is replayed as what it did rather than as what it held: its path
// operations are applied to the tree beneath it, so a commit that wrote one
// file writes that one file onto a tip carrying a hundred others, and a commit
// that removed one removes it from wherever it now stands. Replacing the tree
// wholesale would take every path the remote gained back off the branch, which
// is the discard append-only forbids arriving through a retry.
//
// The order is oldest first, which is the order they happened: a path an
// earlier commit wrote and a later one removed ends up removed, and no rule
// beyond replaying them in sequence is needed to say so.
func (g repository) reapply() error {
	tip, err := g.resolveRef(trackingRef)
	if err != nil {
		return err
	}
	local, err := g.resolveRef(Ref)
	if err != nil {
		return err
	}
	unpushed, err := g.commitsSince(Ref, trackingRef)
	if err != nil {
		return err
	}

	for _, commit := range unpushed {
		operations, err := g.operationsIn(commit)
		if err != nil {
			return err
		}
		message, err := g.messageOf(commit)
		if err != nil {
			return err
		}
		tree, err := g.applyOnto(tip, operations)
		if err != nil {
			return err
		}
		if tip, err = g.commitOnto(tree, tip, message); err != nil {
			return err
		}
	}
	return g.moveRef(Ref, tip, local)
}
