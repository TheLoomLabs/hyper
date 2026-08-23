package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Inspection commands' shared way in to the record (§9, issue #163).
//
// Four commands read the Store back and none of them writes — `runs`, `show`,
// `changes`, `records` — and what they share is the two acts that stand between
// the pin gate and their own work: putting the branch in hand, and saying what
// it means when it could not be put there. It is stated once here so that the
// four cannot come to disagree about what an absent branch costs.

// openStoreForReading puts the Store in hand for a command that only reads it,
// and answers the exit code where it could not.
//
// **It syncs before it opens**, which is not an act these commands could skip:
// `store-absent` means the branch is on neither side (store.ErrAbsent), and
// only the sync reaches the side this clone is not. A runner's clone holds no
// Store until a fetch brings one, so a command that never reached the remote
// would Refuse on every scheduled machine — and §9 requires all four of these
// to Refuse where the branch is missing rather than return an empty answer,
// which is a claim about the record and not about this clone.
//
// The failure it tolerates and the reason are syncForReading's, one function
// down; the two failures are two calls because what each costs is the caller's
// (store.Sync).
func openStoreForReading(command, repoRoot string, instant time.Time, stderr io.Writer) (*store.Store, int) {
	syncForReading(command, "this command", repoRoot, instant, stderr)
	held, err := store.Open(repoRoot, instant)
	if err != nil {
		return nil, reportReadStoreFault(command, stderr, err)
	}
	return held, 0
}

// syncForReading attempts the sync a caller that only reads may proceed
// without, and narrates the failure it is going to tolerate.
//
// **A read tolerates a sync it could not complete** and proceeds against
// whatever branch the clone holds, which is the rhythm ADR-0083 states for a
// read-only Run and which every command that reads the record shares: a fetch
// that could not land is a network fact rather than a missing record.
//
// What it says is the condition and what it did about it, and never git's own
// words: a fetch's error names a remote by URL, which is a fact about the
// machine and not about the record. It is narration and not a Refusal — no
// `error_code`, no row, and stdout carries none of it (§9, §12).
//
// ErrAbsent is the one answer it says nothing about, because it is not a
// failure: the sync ran, reached whatever there was to reach, and found no
// branch on either side. The caller's Open answers the same thing a line later
// and Refuses `store-absent` in the words that name the remedy.
//
// reader is how the caller names itself in that line — *this Run* on a
// read-only Run, *this command* on an Inspection command — and it is the only
// thing the two sites do not share.
func syncForReading(command, reader, repoRoot string, instant time.Time, stderr io.Writer) {
	if err := store.Sync(repoRoot, instant); err == nil || errors.Is(err, store.ErrAbsent) {
		return
	}
	fmt.Fprintf(stderr, "hyper %s: the Store could not be synced; %s reads the branch this clone holds\n", command, reader)
}

// reportReadStoreFault renders whichever way the record stopped a command that
// reads it, and answers the exit code (§9, §12).
//
// The three are told apart by what it would take to clear them. A branch
// neither side holds Refuses `store-absent` at 77 naming `hyper store init` —
// the remedy is an act of somebody's, which is exactly what sorts 77 from 75
// (ADR-0061). A file written above this reader's ceiling Refuses
// `store-schema-unsupported` at the same code, the remedy there being a
// different binary (ADR-0028). Everything else is the world resisting at 1: a
// remote unreachable, a git object that would not read, an entry whose files
// disagree with one another. Never 75, which is a Run that lost the Store and
// which none of these commands is.
//
// command names the command in the one message that is not a Refusal, which is
// the whole reason it is a parameter: one renderer, and a caller still reads
// their own command's name.
func reportReadStoreFault(command string, stderr io.Writer, err error) int {
	var unsupported store.SchemaUnsupported
	switch {
	case errors.Is(err, store.ErrAbsent):
		return refuse(stderr, storeAbsentCode, "no "+store.BranchName+" branch in this repository — hyper store init")
	case errors.As(err, &unsupported):
		// The message carries the path the store package named, which
		// is the file the Refusal cites — §8 states that this code
		// cites a Store file, and it is the one Refusal whose subject
		// is evidence rather than an artefact.
		return refuse(stderr, store.SchemaUnsupportedCode, fmt.Sprintf("%s — install a hyper that reads it", err))
	}
	fmt.Fprintf(stderr, "hyper %s: %s\n", command, err)
	return ExitProblems
}
