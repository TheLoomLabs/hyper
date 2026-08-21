package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The lock is what stops two Runs standing on each other, and its whole
// contract is which pairs of Runs may hold it at once (§6, §7, issue #138).
//
// Every case here takes the lock against a real repository built in a temp
// directory, for store_test.go's own reason: the lock is a fact about a
// filesystem, and the test of a rule that reaches one is what it left there.

// TestLock_TwoReadOnlyRunsMayHoldItAtOnce is the whole point of there being two
// modes. A five-minute monitoring cadence is not starved behind another
// monitoring Run, and it is not starved behind itself.
func TestLock_TwoReadOnlyRunsMayHoldItAtOnce(t *testing.T) {
	r := newRepo(t)

	first, err := store.Acquire(r.root, store.Shared)
	if err != nil {
		t.Fatalf("the first shared Acquire: %v", err)
	}
	defer first.Release()

	second, err := store.Acquire(r.root, store.Shared)
	if err != nil {
		t.Fatalf("the second shared Acquire: %v; two read-only Runs may hold the lock at once", err)
	}
	if err := second.Release(); err != nil {
		t.Errorf("Release: %v", err)
	}
}

// TestLock_AReadOnlyRunAndAnEffectfulOneMayNot is the other half of the same
// sentence, read from both ends: whichever of the two got there first, the
// second one is contended.
func TestLock_AReadOnlyRunAndAnEffectfulOneMayNot(t *testing.T) {
	for name, pair := range map[string][2]store.LockMode{
		"the read-only Run first": {store.Shared, store.Exclusive},
		"the effectful Run first": {store.Exclusive, store.Shared},
		"two effectful Runs":      {store.Exclusive, store.Exclusive},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo(t)

			held, err := store.Acquire(r.root, pair[0])
			if err != nil {
				t.Fatalf("the first Acquire: %v", err)
			}
			defer held.Release()

			second, err := store.Acquire(r.root, pair[1])
			if !errors.Is(err, store.ErrContended) {
				t.Errorf("the second Acquire = %v, want ErrContended", err)
				second.Release()
			}
		})
	}
}

// TestLock_IsReleasedWhenTheRunEnds is the lock not outliving the Run that took
// it: what a Run releases, the next Run takes.
func TestLock_IsReleasedWhenTheRunEnds(t *testing.T) {
	r := newRepo(t)

	held, err := store.Acquire(r.root, store.Exclusive)
	if err != nil {
		t.Fatalf("the first Acquire: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	next, err := store.Acquire(r.root, store.Exclusive)
	if err != nil {
		t.Fatalf("the Acquire after the Release: %v; a released lock is nobody's", err)
	}
	if err := next.Release(); err != nil {
		t.Errorf("the second Release: %v", err)
	}
}

// TestLock_ReleaseIsIdempotent is the `defer` every caller writes: a Run that
// released its lock and then unwound through the same defer releases nothing
// twice, and a lock nobody took releases nothing at all.
func TestLock_ReleaseIsIdempotent(t *testing.T) {
	r := newRepo(t)

	held, err := store.Acquire(r.root, store.Shared)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("the first Release: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Errorf("the second Release: %v, want nil", err)
	}

	var never *store.Lock
	if err := never.Release(); err != nil {
		t.Errorf("releasing a lock that was never taken: %v, want nil", err)
	}
}

// TestLock_LivesUnderGitAndReachesNeitherTheBranchNorTheWorkingTree is where it
// sits and what it must not touch. `.git/hyper/` is git's own directory, so the
// lock is ignored by construction rather than by a rule anybody wrote — no
// working-tree file appears, and no branch gains anything (§7, ADR-0075).
func TestLock_LivesUnderGitAndReachesNeitherTheBranchNorTheWorkingTree(t *testing.T) {
	r := newRepo(t)
	r.seedStore(r.root, store.IntroductionPath, store.Introduction)
	before := r.text("rev-parse", store.Ref)

	held, err := store.Acquire(r.root, store.Shared)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer held.Release()

	local := filepath.Join(r.root, ".git", "hyper")
	entries, err := os.ReadDir(local)
	if err != nil {
		t.Fatalf("reading %s: %v; the lock lives under .git/hyper/", local, err)
	}
	if len(entries) == 0 {
		t.Errorf("%s holds nothing; the lock lives there", local)
	}

	if dirty := r.text("status", "--porcelain"); dirty != "" {
		t.Errorf("the working tree carries %q; the lock never reaches it", dirty)
	}
	if after := r.text("rev-parse", store.Ref); after != before {
		t.Errorf("%s moved from %s to %s; the lock is never part of the record", store.Ref, before, after)
	}
}

// TestLock_NamesARepositoryRootThatIsNotOne is the one fault here that is the
// invocation being wrong rather than another Run holding the lock: there is no
// `.git` to put it under, and the caller is told which of the two it met.
func TestLock_NamesARepositoryRootThatIsNotOne(t *testing.T) {
	held, err := store.Acquire(t.TempDir(), store.Shared)
	if !errors.Is(err, store.ErrNoRepository) {
		t.Errorf("Acquire on a directory that is no repository = %v, want ErrNoRepository", err)
		held.Release()
	}
}
