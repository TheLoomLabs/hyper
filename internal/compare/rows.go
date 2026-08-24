package compare

import (
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Comparison's rows (§8, issues #167, #170 and #171). One list, written
// twice —
// as the page and as the `--json` stream — so the two surfaces cannot state
// different things (ADR-0026).

// Rows answers the ordered rows of one window: the `window` row, then the rows
// of `YOU DID THIS`, then the rows of `THE WORLD MOVED`, then the rows of
// `THE CODE MOVED` and the catch-all that terminates it.
//
// records is what the caller read for the identities Eligible named — the two
// endpoints of each, and what they projected. Nothing here opens a file: which
// identities are eligible and which version stands at each end are this
// package's (Eligible, Endpoint), and the bytes behind them are the Store's.
//
// **The order is the page's and it is a contract rather than a consequence**
// (§8, ADR-0026): a row goes out on its own line, there is no cursor behind the
// stream, and a consumer cannot re-sort what it has already printed.
//
// code is what the caller read for the third table: the reviewed artefacts at
// both revisions and what git says moved between them (code.go). `TOTALS` gets
// no row here or anywhere — §8's stream carries the rows of the tables and the
// `window` row above them and no `totals` object, that line being those rows
// counted rather than a fact of its own (internal/cli's totalsLine).
func Rows(window Window, records []Record, code Code) []render.Row {
	rows := append([]render.Row{windowRowOf(window)}, changeRowsOf(window, records)...)
	return append(rows, CodeRows(window, code)...)
}

// WindowRow is the Comparison's header: which two Runs are being compared, and
// everything the header says about each of them.
//
// It is **one row and not one per line**, which is the shape §8 fixes and the
// precedent the review's own `artefact` row was written on: a header cites no
// line, so one row per rendered line would invent an anchor the surface does
// not have. It names its content — the window — rather than its position on a
// page, as every type in this stream does.
//
// `baseline` is absent where there is none. A `window` row has exactly one way
// of having no baseline — the subject is the first Run of its Procedure — so
// the member's absence carries the whole of it, where the review's `artefact`
// row needs `baseline_absent` because four different absences reach that line
// (§8, §12).
type WindowRow struct {
	Type      string   `json:"type"`
	Procedure string   `json:"procedure"`
	Baseline  *SideRow `json:"baseline,omitempty"`
	Subject   *SideRow `json:"subject"`
}

// Cells is empty: the header is a block of two labelled lines rather than a
// line in a table of like rows, and the page renders it as `changes` writes it
// (ADR-0026, and the review's `artefact` row one package over).
func (r WindowRow) Cells() []string { return nil }

// SideRow is one end of the window as the wire carries it: the six facts §8
// says the header names each Run with, and the two markers that qualify them.
//
// **`ended` stands where the page renders a duration.** §7 is precise that no
// duration is stored anywhere — a stored duration is a second representation of
// what the timestamps already carry, and the two can disagree — and that
// reading holds on a stream as it holds in the Store. The page subtracts inside
// one entry and renders `1m48s`; the wire carries the two instants it
// subtracted and never the subtraction, which is the same one-fact-two-notations
// rule that puts `repo_dirty` here and a `+` on the page.
//
// **Its absence is what the page renders `reaped` for.** A reaped entry's only
// account is a `closed-by/` file, whose `ended_at` is the *closing* Run's
// instant on the closing Run's clock, so no duration derives (§7); the member is
// omitted and `closed_by` beside it says why. That is the ordinary absence rule
// carrying a fact rather than hiding one.
//
// **`outcome` is the entry's own and is written always.** A window never names
// an open entry (StandingOf), so there is always one: the owner's where the
// entry holds an `outcome.json` — on a contested entry included — and `failed`
// where a closing write is its only account (§7).
//
// **`closed_by` is every inference another Run drew**, and it stands beside the
// outcome rather than inside it. Putting a reaper's account in the outcome
// would be the surface deciding between two accounts of what the world did,
// which §7 is precise `hyper` does not do; leaving it off would be the tool
// holding a disagreement it never shows anyone. It is the same member `show`
// carries under the same name, and the page renders it as one stated line per
// file beneath the header.
//
// **`repo_dirty` is written where the entry recorded it**, rather than the `+`
// suffix the page draws on the revision beside it: the bytes that Run read are
// nowhere in git, and this is the marker that stops a consumer resolving the
// revision and believing it read what ran (§7, §8).
//
// **Nothing is abbreviated.** The Run id and the revision go out whole, as
// every id and every digest on this wire does (§8, ADR-0047).
type SideRow struct {
	Run string `json:"run"`
	// Trigger is the composed string and never the mapping, which is §8's
	// own `window` row and the same reading `runs` takes of the same fact:
	// a clock or a person, which is the whole of what §7 says a Trigger
	// distinguishes. `show` is the surface that carries the four members an
	// executor writes, its job being one entry read whole.
	Trigger           string      `json:"trigger"`
	Started           string      `json:"started"`
	Outcome           string      `json:"outcome"`
	Ended             string      `json:"ended,omitempty"`
	ProcedureRevision string      `json:"procedure_revision"`
	RepoDirty         bool        `json:"repo_dirty,omitempty"`
	ClosedBy          []CloserRow `json:"closed_by,omitempty"`
}

// CloserRow is one closing write on a side of the window: another Run's
// inference that this entry's Run had died, and the whole of what the contest
// line beneath the header renders.
//
// The Run is the file's name and not one of its members — a closing write
// carries none naming its author, its path being that member (§7, ADR-0076) —
// and `outcome` is what §7 fixes every closing write as, written out rather
// than left implied because the page states it, and a fact the page states and
// the wire does not is the two surfaces disagreeing (ADR-0026).
type CloserRow struct {
	Run     string `json:"run"`
	Outcome string `json:"outcome"`
	Step    int    `json:"step,omitempty"`
	Ended   string `json:"ended"`
}

// windowRowOf reads the header off the window.
func windowRowOf(window Window) WindowRow {
	row := WindowRow{Type: "window", Procedure: window.Procedure, Subject: sideRowOf(window.Subject)}
	if window.Baseline.Present {
		row.Baseline = sideRowOf(window.Baseline)
	}
	return row
}

// sideRowOf reads one end of the window off its entry.
func sideRowOf(side Side) *SideRow {
	if !side.Present {
		return nil
	}
	entry := side.Entry
	row := &SideRow{
		Run:               entry.Run.String(),
		Trigger:           entry.Trigger.Text(),
		Started:           store.InstantText(entry.StartedAt),
		ProcedureRevision: entry.Provenance.ProcedureRevision,
		RepoDirty:         entry.Provenance.RepoDirty,
	}
	// The outcome is the Store's own derivation over the files present —
	// the owner's wherever one exists, `failed` where a closing write is
	// the entry's only account — read here rather than restated (§7).
	if outcome, closed := entry.Outcome(); closed {
		row.Outcome = string(outcome)
	}
	// The end is the owner's and never a closer's, for Instant's reason one
	// file over: a closing write's instant is on the closing Run's clock.
	if entry.Owner.Outcome != "" {
		row.Ended = store.InstantText(entry.Owner.EndedAt)
	}
	for _, closer := range entry.Closers {
		row.ClosedBy = append(row.ClosedBy, CloserRow{
			Run:     closer.Run.String(),
			Outcome: string(store.OutcomeFailed),
			Step:    closer.Step,
			Ended:   store.InstantText(closer.EndedAt),
		})
	}
	return row
}
