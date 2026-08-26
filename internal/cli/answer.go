package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// writeAnswer writes a command's answer to its destination and answers the exit
// code its caller returns: 0 clears the caller to go on to its own, and
// anything else is the code it returns, the answer having not been written.
//
// It is one function rather than the same lines in each command because what it
// holds is a rule and not a convenience: stdout is the answer and nothing else
// ever goes there, so an answer that could not be written is reported on the
// narration and exits 2 (§9). A command writing that block for itself is a
// command free to get the terminal row or the stream it reports on wrong, and
// there are nineteen of them.
func writeAnswer(command string, to destination, rows []render.Row, terminal render.Row, page func(io.Writer, []render.Row) error) int {
	if err := to.answer(rows, terminal, page); err != nil {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", command, err)
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
// writeAnswer's own reason above: a page whose values are one each
// rather than a table of like rows is a shape two commands already have,
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

// yesCell is how a table renders a marker the wire carries as a boolean: the
// word where it stands, and nothing at all where it does not.
//
// A blank is the page's reading of the ordinary absence rule (§7) — the row
// omits the member and the column shows nothing against it — and the word is
// what a reader scans a column for. It is one function rather than three lines
// at each marker so that two tables cannot come to spell one `yes` differently.
func yesCell(marked bool) string {
	if marked {
		return "yes"
	}
	return ""
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
// out of what, what did not, and what to do about it. noun is what was counted,
// in the plural and capitalised as the glossary spells it — the one part of the
// count that is the command's own.
//
// Which form the count takes turns on whether the caller named the cap, because
// the two send a reader somewhere different — a caller who wrote --limit 2 is
// told what their own number cut, and one who wrote nothing is told there is a
// default at all. Naming a flag the caller never typed would point the remedy
// at an argument they would go looking for in their own command line.
//
// narrowing is the remedy, and it is the caller's own words for the same reason
// the noun is: the parameters that narrow an axis differ by which command was
// called (render.TruncationMarker). A command that has none passes the empty
// string, and there the remedy is a larger cap — which is what a listing over a
// namespace with nothing to narrow has left. **Where a command has one, the
// narrowing is the whole remedy and no larger --limit is offered beside it**:
// this line is the page's half of the marker on the wire, and the marker names
// a narrower question rather than a bigger answer (§9, §12, ADR-0065).
//
// Neither form offers a next page. There is no cursor behind this stream, and a
// truncated result must never look complete.
func truncationLine(noun string, returned, found int, parsed commandArgs, narrowing string) string {
	cut, remedy := fmt.Sprintf("--limit %d", parsed.limit), ""
	if !parsed.limitNamed {
		cut, remedy = fmt.Sprintf("the default limit of %d", parsed.limit), " — name a larger --limit for the rest"
	}
	if narrowing != "" {
		remedy = " — " + narrowing
	}
	return fmt.Sprintf("returned %d of %d %s; %d dropped by %s%s",
		returned, found, noun, found-returned, cut, remedy)
}
