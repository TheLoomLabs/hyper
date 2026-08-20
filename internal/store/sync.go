package store

import (
	"errors"
	"time"
)

// The two acts that put the branch in hand: the sync that brings it down from
// the remote, and the handle that reads what the clone holds (§7, issue #130).
//
// They are two calls rather than one because their failures are two facts and
// the caller decides what each one costs. A read-only Run proceeds offline, so
// a sync that could not reach the remote is not the end of it; a Run that
// cannot find the Store at all Refuses (`store-absent`), and that is what
// ErrAbsent says. Folding them together would make an unreachable network read
// as *there is nothing recorded*, which is the reading that disarms every test
// run-once and `skip-if-recorded` perform (§7, ADR-0006).

// ErrAbsent is the Store that is not there: neither in the clone nor, where a
// remote is configured, on it. It is the condition a caller renders as
// `store-absent` (§12); this package holds no Run and renders no Refusal.
//
// The remedy it names is the only one there is. The branch is created by an
// explicit act and never by a Run, read-only Runs included, because a fetch
// that failed mid-flight and a branch that never existed look identical from
// the inside (§7).
var ErrAbsent = errors.New("the repository holds no " + BranchName + " branch; the Store is created by `hyper store init` and never by a Run")

// Sync brings the Store branch down from the remote, and is what puts it in
// hand on a clone that lacks it.
//
// **It decides the depth exactly once, at the moment there is nothing to
// preserve** (§7, ADR-0074). Where the clone does not hold the branch — which
// is every runner, `actions/checkout` taking one ref — this is the branch's
// arrival and it is a depth-1 fetch of that one ref. Where the clone holds it
// already the fetch names no depth, so a Store held whole stays whole and one
// held shallow is never deepened. Neither is ever a filtered fetch: a version's
// `written_at` sits inside the file, so ordering a series opens every version
// of it, and under a blob or tree filter each of those is a lazy fetch that
// makes *a read-only Run proceeds offline* false wherever the network is.
//
// *Holding the branch* is the clone's own ref or the remote-tracking one, and
// the second is not a refinement: an ordinary `git clone` leaves every branch
// as a remote-tracking ref and the Store's history whole, and a depth-1 fetch
// there would cut a history the clone already had — `hyper` shortening a Store
// it did not create, through the one path where it looks like an arrival.
//
// A repository with no remote configured reaches no network at all and reads
// the branch it has. The absent remote is not a failure: a repository that has
// never had one is not a repository whose Store is missing. There the question
// is the branch itself and not the tracking ref: nothing can be fetched, and a
// tracking ref left by a remote that is gone is not a Store this clone holds.
//
// What comes down lands on the remote-tracking ref, and the local branch is
// pointed at it only where that loses nothing — see adopt, which is where a Run
// that wrote and could not push is kept whole.
//
// It answers ErrAbsent where neither side holds the branch, ErrNoRepository
// where repoRoot holds no git repository, and an ordinary error where the world
// resisted — a remote that could not be reached, read as a remote holding
// nothing, is exactly how a second orphan root gets minted (ADR-0074).
func Sync(repoRoot string, now time.Time) error {
	repo, err := open(repoRoot, now)
	if err != nil {
		return err
	}

	local, err := repo.holdsRef(Ref)
	if err != nil {
		return err
	}
	remote, err := repo.hasRemote(RemoteName)
	if err != nil {
		return err
	}
	if !remote {
		if local {
			return nil
		}
		return ErrAbsent
	}

	tracked, err := repo.holdsRef(trackingRef)
	if err != nil {
		return err
	}
	if local || tracked {
		if err := repo.fetchIncremental(RemoteName, Ref, trackingRef); err != nil {
			return err
		}
		return repo.adopt()
	}

	// The one look before the arrival. An empty listing is *the remote does
	// not hold it* and an error is the world resisting, and the two are
	// never folded together.
	published, err := repo.remoteHoldsRef(RemoteName, Ref)
	if err != nil {
		return err
	}
	if !published {
		return ErrAbsent
	}
	return repo.arrive()
}

// arrive brings the branch down to a clone that does not hold it: the tip and
// no history, and the local ref pointed at what came.
func (g repository) arrive() error {
	if err := g.fetchShallow(RemoteName, Ref, trackingRef); err != nil {
		return err
	}
	return g.adopt()
}

// adopt points the local branch at what the fetch brought, where doing so loses
// nothing.
//
// The Store is append-only and never force-pushed, so the ordinary case is that
// the fetched tip is a descendant of the local branch and pointing at it is a
// fast-forward — or that there is no local branch at all, which is the arrival.
//
// **Where the local branch holds commits the remote does not, it is left
// exactly where it is.** That is a Run that wrote and could not push, and §7
// states what happens to it: what it wrote stands locally and goes out with the
// next Run that pushes, which re-applies the whole unpushed set onto the
// fetched tip. Moving the ref here would discard that set, and failing here
// would make the state §7 calls ordinary a Store nothing could sync again. The
// re-application itself is the push's, and the push is milestone 4.7's.
func (g repository) adopt() error {
	tip, err := g.resolveRef(trackingRef)
	if err != nil {
		return err
	}
	held, err := g.holdsRef(Ref)
	if err != nil {
		return err
	}
	if !held {
		return g.createRef(Ref, tip)
	}

	local, err := g.resolveRef(Ref)
	if err != nil {
		return err
	}
	if local == tip {
		return nil
	}
	unpushed, err := g.carriesCommitsOutside(local, tip)
	if err != nil || unpushed {
		return err
	}
	return g.moveRef(Ref, tip, local)
}

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
}

// Open answers a handle on the Store the clone holds, and ErrAbsent where it
// holds none.
//
// It reaches no network and never creates anything: putting the branch in hand
// is Sync's act and creating one is `hyper store init`'s. A caller that wants
// both syncs first and opens after, which is also the order in which their two
// failures mean different things.
//
// now is the clock the caller threaded. Nothing a read does consumes it; it is
// the environment every git subprocess in this package is run with, and the
// commits a later write makes through this handle take both their dates from it
// (§7).
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
	return &Store{repo: repo, commit: commit}, nil
}
