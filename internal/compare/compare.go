// Package compare is the Comparison (§8, issue #167): one Run rendered against
// the Run before it, as an ordered list of rows.
//
// **It is pure.** It opens no file, starts no subprocess and reads no clock.
// Two Journal entries, the Store reads those entries need and the artefacts at
// both revisions are handed in; what comes back is the row list, which
// `internal/render` writes as the page and as the `--json` stream (ADR-0026).
// The git reads are `internal/revision`'s and the Store reads are
// `internal/store`'s, made by the caller.
//
// The reason it is a package and not a file under `internal/cli` is size and
// not taste. What lives here is the window rule, the endpoint Head-at-instant,
// identity-set eligibility, six change names with one precedence rule between
// them, the ordinal in four forms, nine code-fact classes and `TOTALS`' three
// ordered forms — a derivation `internal/cli/changes.go` should be able to
// call rather than contain, exactly as it calls `internal/run` and
// `internal/store`.
//
// What stands here is the window and the header (#167), the two Record tables
// beneath them (#170), and `THE CODE MOVED` with `TOTALS`' last segment
// (#171).
package compare

import (
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Side is one end of a window: the Journal entry that end is, and the records
// that entry holds about its Steps.
//
// The Step records travel with the entry because the endpoint instant needs
// them — a reaped entry's is the last Step file's `ended_at` (Instant below) —
// and because the tables above will read the identity sets off the same files.
// They are the caller's read: `store.Store.Dispositions` answers them, and this
// package is handed the answer rather than the Store.
//
// Present is what says the end exists at all. A Comparison always has a
// subject; a baseline is what the first Run of a Procedure has none of, and the
// header states that absence as a named state rather than rendering an empty
// line (§8).
type Side struct {
	Present bool
	Entry   store.Entry
	// Steps is the entry's Step records, in the Run's own written order —
	// the Step files it wrote and, where a reaper closed it, the reading
	// its earliest closing write carries beside them (§7). It is a field
	// rather than a constructor argument because Select answers the two
	// entries a window names before the caller has read anything about
	// their Steps: the selection costs one listing, and the Step files of
	// two entries per Procedure are read once those two are known.
	Steps []store.StepFile
}

// Instant is this side of the window: the entry's **own last** instant.
//
// It is `outcome.json`'s `ended_at` where the Run wrote one, and the last Step
// file's `ended_at` where the entry's only account is a `closed-by/` file — the
// instant §7 names as *when the Run went quiet*, read as a timestamp and never
// as a verdict.
//
// **It is never the closing write's**, which is the whole reason this is not
// `store.Entry.Ended`. That instant is the *closing* Run's on the closing Run's
// clock, so a Run reaped a week later would sweep every intervening Run's
// versions into its side of the window — the same cross-entry reading §7
// forbids a duration for, applied to an endpoint (§8).
//
// A contested entry takes no special reading: it holds an `outcome.json` its
// own Run wrote, so its side is that `ended_at` like any other Run's, and the
// `closed-by/` file beside it is never an endpoint.
//
// An entry with neither — an open one — has no instant, and Select never puts
// one on either side of a window (StandingOf below).
func (s Side) Instant() time.Time {
	if s.Entry.Owner.Outcome != "" {
		return s.Entry.Owner.EndedAt
	}
	var quiet time.Time
	for _, record := range s.Steps {
		if record.EndedAt.After(quiet) {
			quiet = record.EndedAt
		}
	}
	return quiet
}

// Reaped answers whether this side's only account of how it ended is a closing
// write, which is what says no duration derives for it (§7, store.Entry.Duration).
func (s Side) Reaped() bool { return s.Entry.Account() == store.AccountReaped }

// Standing is whether a Journal entry may be a side of a window, and where it
// may not, which of the two reasons it is.
//
// The two are separate values because §8 is precise that they are separate
// facts. A rehearsal is **disqualified**: it performed the reads it reached and
// withheld the first effectful Step, and letting one be a baseline would retire
// the warning a real Run earned. An open entry is **not yet nameable**: it is
// not disqualified at all, it is an entry whose Run may be in flight or may be
// gone, and naming it would have the header render an outcome the entry does
// not have (§7, §8, ADR-0001).
//
// A Probe is neither, because a Probe writes no Journal entry and can never
// reach this (ADR-0009).
type Standing int

const (
	// Nameable is an entry a window may name. **An outcome does not
	// disqualify one**: a refused Run's completed Steps reached the world
	// like any other's, so the triple is not consulted here at all.
	Nameable Standing = iota + 1
	// Rehearsal is an entry marked `dry_run`. It is disqualified as a
	// **baseline** under every form, and as the subject no rule may choose
	// — which is what Select and Preceding read this for. It is not
	// disqualified as a subject a caller **named**: `changes --subject`
	// reads past this value deliberately, asking what that Run read rather
	// than what the world became (§7, §8, ADR-0115).
	Rehearsal
	// Unclosed is an entry holding no account of how it ended: neither its
	// own Run's `outcome.json` nor another Run's closing write. It is
	// spelled for the state and not for the file, an entry closed by a
	// reaper being closed.
	Unclosed
)

// StandingOf answers which of the three one entry is.
func StandingOf(entry store.Entry) Standing {
	switch {
	case entry.DryRun:
		return Rehearsal
	case entry.Account() == store.AccountOpen:
		return Unclosed
	}
	return Nameable
}
