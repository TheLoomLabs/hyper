package compare

import (
	"maps"
	"slices"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The window: which two Runs a Comparison names (§8, issue #167).

// Window is one Procedure's Comparison: the Procedure it is of, and the two
// ends it is between.
//
// There is one per Procedure and never one across several. The baseline is the
// previous Run of the **same** Procedure, so a monitoring Run is never compared
// against a provisioning one — and that window is total rather than partial:
// every Run is a Run of a Procedure (ADR-0036), so no Run reaches the world
// outside some Procedure's Comparison.
type Window struct {
	Procedure string
	Baseline  Side
	Subject   Side
}

// Selection is what the caller named: which Procedure, and which of the two
// ways of naming a window they used.
//
// `--between` is not here. It names two Runs directly, so it resolves two ids
// against the Journal and builds a Window outright; what this type carries is
// the selection that has a *rule* behind it rather than two names (§8, §9).
type Selection struct {
	// Procedure is the Procedure named positionally, and "" is the
	// whole-Store mode: naming nothing compares across every Procedure at
	// once, which is why the Procedure is positional on `changes` and a
	// parameter on `runs` — it decides which rendering you get rather than
	// filtering the rows of one (§9).
	Procedure string
	// Since and SinceNamed are `--since`: take the last Run before that
	// instant and fold everything after it into one rendering. The second
	// member says a window was asked for at all, the zero instant being a
	// point in the year 1 rather than *no bound* (internal/cli's own
	// reading of the same flag).
	Since      time.Time
	SinceNamed bool
}

// Select answers the windows a Comparison renders, one per Procedure, in
// Procedure-name code-point order.
//
// entries is the Journal as the Store listed it and the order it is in does not
// matter: the two ends are chosen on `started_at`, which is the axis §9 orders
// Runs on and the only axis Runs have (ADR-0065).
//
// **The subject is the newest nameable Run and the baseline is the one before
// it.** `--since` moves the baseline and never the subject: the baseline is the
// last nameable Run to have *started* before the instant, and everything after
// it folds into one rendering, so the subject stays the newest. Where no
// nameable Run started at or after the instant there is no window at all —
// nothing happened in the span the caller asked about, and a window whose two
// ends were one Run would render a Comparison of a Run against itself.
//
// **A rehearsal and an open entry are passed over**, for the two different
// reasons StandingOf states, and a Procedure whose every entry is one of those
// names no window.
//
// **Where no baseline exists the window carries none**, and the header states
// that as a named state — *no baseline — first Run of `<Procedure>`* — rather
// than rendering a line about a Run that is not there (§8).
func Select(entries []store.Entry, asked Selection) []Window {
	nameable := map[string][]store.Entry{}
	for _, entry := range entries {
		if StandingOf(entry) != Nameable {
			continue
		}
		if asked.Procedure != "" && entry.Procedure != asked.Procedure {
			continue
		}
		nameable[entry.Procedure] = append(nameable[entry.Procedure], entry)
	}

	windows := make([]Window, 0, len(nameable))
	for _, procedure := range slices.Sorted(maps.Keys(nameable)) {
		of := nameable[procedure]
		// Newest first, on `started_at` with the Run id descending as
		// the tie-break: §9's ordering for Runs, restated here because
		// this reads a listing rather than making one, and a caller
		// handing entries in some other order must not get some other
		// window.
		slices.SortFunc(of, newest)
		if window, named := windowOf(procedure, of, asked); named {
			windows = append(windows, window)
		}
	}
	return windows
}

// windowOf chooses one Procedure's two ends out of its own nameable entries,
// newest first.
func windowOf(procedure string, of []store.Entry, asked Selection) (Window, bool) {
	if len(of) == 0 {
		return Window{}, false
	}
	window := Window{Procedure: procedure, Subject: Side{Present: true, Entry: of[0]}}
	if !asked.SinceNamed {
		if len(of) > 1 {
			window.Baseline = Side{Present: true, Entry: of[1]}
		}
		return window, true
	}

	// `--since` is a lower bound and includes the instant it names, exactly
	// as it does on `runs` and `records`: a timestamp copied off a page
	// selects the Run it was copied from (internal/cli's withinWindow).
	folded := 0
	for folded < len(of) && !of[folded].StartedAt.Before(asked.Since) {
		folded++
	}
	if folded == 0 {
		return Window{}, false
	}
	if folded < len(of) {
		window.Baseline = Side{Present: true, Entry: of[folded]}
	}
	return window, true
}

// newest orders two entries as §9 orders Runs: `started_at` descending, ties
// broken on the Run id descending — a time key with a name behind it, and a
// UUIDv7 total over the tie (ADR-0065).
func newest(a, b store.Entry) int {
	if !a.StartedAt.Equal(b.StartedAt) {
		if a.StartedAt.After(b.StartedAt) {
			return -1
		}
		return 1
	}
	switch {
	case a.Run.String() > b.Run.String():
		return -1
	case a.Run.String() < b.Run.String():
		return 1
	}
	return 0
}
