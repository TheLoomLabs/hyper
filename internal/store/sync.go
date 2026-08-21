package store

import "time"

// The sync: bringing the branch down from the remote, and the depth that one
// act decides forever (§7, ADR-0074).
//
// It is a call of its own rather than something Open does, because its failure
// and Open's are two facts and the caller decides what each one costs. A
// read-only Run proceeds offline, so a sync that could not reach the remote is
// not the end of it; a Run that cannot find the Store at all Refuses
// (`store-absent`). Folding them together would make an unreachable network
// read as *there is nothing recorded*, which is the reading that disarms every
// test run-once and `skip-if-recorded` perform (§7, ADR-0006).

// Sync brings the Store branch down from the remote, and is what puts it in
// hand on a clone that lacks it — which is every runner.
//
// What it decides is whether there is a Store to bring at all; how it is
// brought, and the depth that decision fixes forever, is bringBranch's (§7,
// ADR-0074). What comes down lands on the remote-tracking ref, and the local
// branch is pointed at it only where that loses nothing — see adopt, which is
// where a Run that wrote and could not push is kept whole.
//
// A repository with no remote configured reaches no network at all and reads
// the branch it has. The absent remote is not a failure: a repository that has
// never had one is not a repository whose Store is missing. Neither is a remote
// that holds no branch while this clone does — the Store here stands, and
// publishing it is the next push's.
//
// It answers ErrAbsent where neither side holds the branch, ErrNoRepository
// where repoRoot holds no git repository, and an ordinary error where the world
// resisted.
//
// **What a failure costs is entirely the caller's**, and the two Runs spend it
// differently: an effectful Run is `failed` at 75, its sync being the push of
// its own open entry, and a read-only Run tolerates the failure outright and
// proceeds against whatever branch the clone holds. §7 leaves the second
// under-determined and internal/cli's locateStore is where it is resolved and
// written down (issue #138).
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
		// With no remote there is one question and one answer, and
		// nothing on this arm reaches a network: the branch is here, or
		// there is no Store. A tracking ref is not asked after here —
		// removing a remote takes its tracking refs with it, so with no
		// remote configured there is none to hold anything.
		if local {
			return nil
		}
		return ErrAbsent
	}

	held, err := repo.holdsBranch()
	if err != nil {
		return err
	}

	if !held {
		// The one look before the arrival, and it decides between two
		// answers rather than confirming one. An empty listing is *the
		// remote does not hold it* and an error is the world resisting;
		// a remote that could not be reached, read as a remote holding
		// nothing, is exactly how a second orphan root gets minted
		// (ADR-0074).
		published, err := repo.remoteHoldsRef(RemoteName, Ref)
		if err != nil {
			return err
		}
		if !published {
			return ErrAbsent
		}
	}

	if err := repo.bringBranch(); err != nil {
		if !local {
			return err
		}
		// The branch is here and the remote has none. That is not a
		// failed sync: the Store this clone holds stands, and the push
		// that publishes it is the next one's — an `init` whose push
		// was rejected leaves exactly this state. Anything else is the
		// world resisting and the error stands, which is why the remote
		// is asked again rather than git's own words read. It is the
		// second reach at the remote and it is on the failing path
		// alone, so an ordinary sync is one.
		published, asked := repo.remoteHoldsRef(RemoteName, Ref)
		if asked != nil || published {
			return err
		}
		return nil
	}
	return nil
}

// holdsBranch says whether the clone holds the Store branch at all: its own
// ref, or the remote-tracking ref left by a clone of a repository that already
// had one.
//
// The second is not a refinement. An ordinary `git clone` leaves every branch
// as a remote-tracking ref and the Store's history whole, so a clone that holds
// it only that way is a clone holding the Store — and a depth-1 fetch there
// would cut a history it already had, `hyper` shortening a Store it did not
// create through the one path where it looks like an arrival (ADR-0074).
func (g repository) holdsBranch() (bool, error) {
	for _, ref := range []string{Ref, trackingRef} {
		held, err := g.holdsRef(ref)
		if err != nil || held {
			return held, err
		}
	}
	return false, nil
}

// bringBranch brings the branch down from the remote and points the local ref
// at what came, deciding the depth on whether the clone already holds it.
//
// **The depth is decided exactly once, at the moment there is nothing to
// preserve** (§7, ADR-0074). A clone that does not hold the branch takes the
// tip and no history: this is the arrival, and it is every runner's, an
// `actions/checkout` taking one ref. A clone that holds it names no depth, so a
// Store held whole stays whole and one held shallow is never deepened.
//
// It is the one place either decision is made, so `store init` finding the
// branch on the remote and a Run syncing at start cannot answer it differently.
func (g repository) bringBranch() error {
	held, err := g.holdsBranch()
	if err != nil {
		return err
	}
	fetch := g.fetchShallow
	if held {
		fetch = g.fetchIncremental
	}
	if err := fetch(RemoteName, Ref, trackingRef); err != nil {
		return err
	}
	return g.adopt()
}

// adopt points the local branch at what the fetch brought, where doing so loses
// nothing.
//
// The Store is append-only and never force-pushed, so the ordinary case is that
// the fetched tip is a descendant of the local branch and pointing at it is a
// fast-forward — or that there is no local branch at all, which is the arrival
// and the whole of what a fetch into a tracking ref leaves to do.
//
// **Where the local branch holds commits the remote does not, it is left
// exactly where it is.** That is a Run that wrote and could not push, and §7
// states what happens to it: what it wrote stands locally and goes out with the
// next Run that pushes, which re-applies the whole unpushed set onto the
// fetched tip. Moving the ref here would discard that set — the one act
// append-only forbids, arriving at the ref rather than at a file — so the two
// lines that detect it are not the push retry arriving early but what keeps
// this one from being a write. The re-application is milestone 4.7's.
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
	behind, err := g.isAncestor(local, tip)
	if err != nil || !behind {
		return err
	}
	return g.moveRef(Ref, tip, local)
}
