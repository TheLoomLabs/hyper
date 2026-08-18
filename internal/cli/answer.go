package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// writeAnswer writes a command's answer to stdout in whichever mode was asked
// for: the --json row stream, terminated by the terminal row the caller
// supplies, or the command's own page. Both are written from one list of rows,
// which is the whole of ADR-0026 — the two surfaces cannot state different
// things because there is one set of rows behind them.
//
// It is one function rather than the same eight lines in each command because
// what it holds is a rule and not a convenience: stdout is the answer and
// nothing else ever goes there, so a stream that could not be written is
// reported on stderr and exits 2 (§9). A command writing that block for itself
// is a command free to get the mode, the terminal row or the stream it reports
// on wrong, and this milestone alone lands four callers.
//
// page is the command's own, because the shape of a page is: its columns, its
// header, and the line that stands where there are no rows are facts about that
// command and not about the renderer. What is shared is the path from rows to
// bytes, not the layout.
//
// A return of 0 clears the caller to go on to its exit code; anything else is
// the code it returns, the answer having not been written.
func writeAnswer(command string, stdout, stderr io.Writer, asJSON bool, rows []render.Row, terminal render.Row, page func(io.Writer, []render.Row) error) int {
	var err error
	if asJSON {
		err = render.WriteJSON(stdout, rows, terminal)
	} else {
		err = page(stdout, rows)
	}
	if err != nil {
		fmt.Fprintf(stderr, "hyper %s: %s\n", command, err)
		return ExitUsage
	}
	return 0
}

// labelledValue is one line of a block of labelled values: the label, and what
// the row states against it. A value the row does not carry writes no line at
// all, which is the ordinary absence rule the wire applies to the same member —
// a page carrying a label against nothing would state a claim the artefact
// never made (§7, ADR-0064).
type labelledValue struct{ label, value string }

// writeLabelledValues writes a block of them, aligned, in the order they are
// given — which is the order of the row's own members, so a reader moving
// between the page and the wire reads the same facts in the same sequence.
//
// It is one function rather than the same eight lines in each command for
// writeAnswer's own reason: a page whose values are one each rather than a
// table of like rows is a shape two commands in this milestone already have,
// and a command spelling the alignment for itself is a command free to get the
// absence rule or the padding wrong.
//
// No line ends in padding: each value is its line's last cell, which tabwriter
// leaves unaligned, so nothing here needs the trim WriteTable performs.
func writeLabelledValues(w io.Writer, values []labelledValue) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, stated := range values {
		if stated.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", stated.label, stated.value); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// truncate keeps the first N of a command's own order and says how many it
// dropped. The first N of a normative order is the answer to a question rather
// than an arbitrary sample of one, which is what makes a bounded return usable
// at all (§9, ADR-0065). There is no cursor, no page parameter and no way to ask for
// the next N: the remedy for a truncated result is a narrower question, and on
// a namespace with no narrowing parameter it is a larger --limit.
//
// It lives here beside writeAnswer for writeAnswer's own reason: the cut is
// applied before either rendering, so the table and the --json stream are one
// row set cut in one place (ADR-0026), and a command holding that sequence for
// itself is a command free to get it wrong.
func truncate(rows []render.Row, limit int) (kept []render.Row, dropped int) {
	if limit <= 0 || len(rows) <= limit {
		return rows, 0
	}
	return rows[:limit], len(rows) - limit
}

// truncationLine is what a truncated result says to a human: what came back,
// out of what, and what did not. noun is what was counted, in the plural and
// capitalised as the glossary spells it — the one part of this line that is the
// command's own.
//
// Which of the two forms it takes turns on whether the caller named the cap,
// because the two send a reader somewhere different — a caller who wrote
// --limit 2 is told what their own number cut, and one who wrote nothing is
// told there is a default at all and what widens it. Naming a flag the caller
// never typed would point the remedy at an argument they would go looking for
// in their own command line.
//
// Neither form names an axis, and neither offers a next page: there is no
// cursor behind this stream, and neither namespace a listing command
// enumerates has a narrowing parameter to suggest (§9, §12).
func truncationLine(noun string, returned, found int, parsed commandArgs) string {
	dropped := found - returned
	if parsed.limitNamed {
		return fmt.Sprintf("returned %d of %d %s; %d dropped by --limit %d",
			returned, found, noun, dropped, parsed.limit)
	}
	return fmt.Sprintf("returned %d of %d %s; %d dropped by the default limit of %d — name a larger --limit for the rest",
		returned, found, noun, dropped, parsed.limit)
}
