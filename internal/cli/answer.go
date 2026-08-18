package cli

import (
	"fmt"
	"io"

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
