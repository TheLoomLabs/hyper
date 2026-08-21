package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// The lock: what stops two Runs standing on each other, and the one piece of
// `hyper`'s own state that sits on a disk rather than on the branch (§6, §7,
// ADR-0075, issue #138).
//
// A Run holds it for its duration — **exclusive if it contains any effectful
// Step, shared if every Step is `read`**. Which mode a Run takes is decided
// from the Kinds `check` already computes, and deciding it is internal/run's,
// which states what the two modes buy; taking it is here, beside the branch it
// is a lock on.
//
// Contention is not a Refusal. No guardrail declined and nobody has anything to
// edit: the other Run ends and this one succeeds on the same invocation five
// minutes later, which is exactly what sorts `75` from `77` (§12, ADR-0061).
// This package reports the condition and maps no code, as it does for
// ErrPushExhausted one file over.
//
// **It is one filesystem's lock, and two executors share no filesystem.** What
// stands in for it across runners is the concurrency group a projection writes
// (§10); between a laptop and a runner nothing stands in for it at all, and §13
// carries that as the limit it is.

// LockMode is which of the two locks a Run takes. Both land together because
// which one a Run takes is one decision, read off the Kinds before any Step
// runs — and only the shared one is exercised while every Step this binary
// performs is a `read`.
type LockMode int

const (
	// Shared is the read-only Run's. Any number of them may hold the Store
	// at once: none of them changes the world, and what each writes is a
	// path only it can reach (ADR-0076).
	Shared LockMode = iota
	// Exclusive is the lock of a Run carrying any effectful Step. It
	// excludes every other Run, read-only ones included: an effectful Run
	// closes another Run's open entry (§6), and a read-only Run reading the
	// Journal underneath one would be reading a record mid-write.
	Exclusive
)

// ErrContended is another Run holding the Store. It is a condition this package
// reports and never a code it maps: its caller is the one that knows a Run is
// `failed` at 75 here, and that this is neither a Refusal nor a failure of the
// work (§9, §12, ADR-0061).
var ErrContended = errors.New("another Run holds the " + BranchName + " lock in this repository; a Run holds it for its duration, and this one takes it or stops")

// localStateDir is where `hyper`'s own local state sits: under `.git/`, which
// is the whole of what keeps it out of the record. git ignores that path by
// construction rather than by a rule anybody wrote, there is no working-tree
// file to commit by accident, and nothing here is ever part of the branch (§7,
// ADR-0075).
const localStateDir = "hyper"

// lockFile is the file the lock is taken on. It is one file for both modes
// because they are two modes of one lock: two files could each be held while
// the other was, which is the whole thing the exclusive mode exists to prevent.
const lockFile = "lock"

// Lock is the Store lock, held. It is released by Release and by the process
// ending, whichever comes first.
type Lock struct {
	// file is the open file the kernel's lock hangs off, and nil once the
	// lock has been released. The lock is the file **description**'s rather
	// than the file's, which is what makes the two guarantees below true.
	file *os.File
}

// Acquire takes the lock in the mode named, and answers ErrContended where
// another Run holds it in a mode that excludes this one.
//
// **It is an advisory lock the kernel holds, not a file whose existence is the
// lock**, and the difference is the one property a Run cannot do without: a
// `hyper` killed outright — the second interrupt §6 states, an executor's grace
// period running out, a laptop shutting its lid — releases it on the spot. A
// lock whose existence were the lock would be one a crash left behind forever,
// and §6 states in as many words that there is no reaper, no daemon and no
// heartbeat to clear one. So `.git/hyper/lock` outlives the Run as an empty
// file and the **lock** outlives nothing: what a Run holds ends when the Run
// does, which is what its removal was for.
//
// It never blocks. A Run that waited would be a Run whose Cadence silently
// became *whenever the other one finishes*, and §6 makes contention an outcome
// rather than a queue.
//
// It answers ErrNoRepository where repoRoot holds no git repository — the same
// fault the rest of this package answers for the same cause, and the one here
// that is the invocation being wrong rather than another Run being alive.
func Acquire(repoRoot string, mode LockMode) (*Lock, error) {
	gitDir, err := gitDir(repoRoot)
	if err != nil {
		return nil, err
	}
	local := filepath.Join(gitDir, localStateDir)
	if err := os.MkdirAll(local, 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(local, lockFile), os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), flockOp(mode)|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrContended
		}
		return nil, fmt.Errorf("taking the %s lock: %w", BranchName, err)
	}
	return &Lock{file: file}, nil
}

// flockOp is the flock operation each mode is: the two the kernel already has,
// and the reason no counting, no lease and no lock file format is written here.
func flockOp(mode LockMode) int {
	if mode == Exclusive {
		return syscall.LOCK_EX
	}
	return syscall.LOCK_SH
}

// Release gives the lock up.
//
// It is safe to call on a lock already released and on one that was never
// taken. Its caller is a `defer` written beside the Acquire it pairs with, and
// the day a path releases early and unwinds through that defer as well is the
// day a second release has to be nothing rather than a fault about a lock
// nobody holds.
//
// It answers an error because releasing can fail and a caller that wants to
// know may ask. `hyper run` does not: a Run whose lock would not come off has
// already done everything it was going to do, and the process ending releases
// it whatever this answers.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	// Closing the descriptor releases the lock on its own, so the explicit
	// unlock is what makes the two acts separable: an unlock that failed is
	// reported, and the descriptor is closed either way rather than leaked
	// behind the report.
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if closed := file.Close(); err == nil {
		err = closed
	}
	return err
}
