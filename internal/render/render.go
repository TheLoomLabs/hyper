// Package render is hyper's row stream (issue #110). A command builds an
// ordered list of typed rows, and one function writes them as the --json NDJSON
// stream and one writes them as the human table. The two are two forms of one
// thing (ADR-0026): both are written from one list, so the two surfaces cannot
// state different things.
//
// What is here is the stream and nothing else. A row type belongs to the
// command that writes it, and so does the shape of that command's page — its
// columns, its headers, the line that stands where there are no rows. One
// renderer means one path from rows to bytes, not one table layout for every
// command.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// A Row is one row of a stream: one object on the --json wire, and one line on
// the page where its command tabulates it.
//
// A row's wire shape is its own JSON encoding, and three things about that
// encoding are contracts rather than conveniences (§8, §9):
//
//   - type is the first key of the object, and the remaining keys follow the
//     row's declaration order — which is what makes --json stable without a
//     schema. encoding/json marshals a struct's fields in declaration order, so
//     a row holds both halves by declaring its type first. Nothing here holds
//     it at write time: a stream that stopped mid-flight to report a badly
//     declared row would leave the wire cut off, which is the failure it would
//     be defending against. The corpus holds it instead, over the checked-in
//     golden files of every command that opens a stream.
//   - A member the row does not carry is absent from the object entirely rather
//     than written as null or "", the ordinary absence rule being a fact a
//     reader reads (§7). A row declares such a member omitempty, or as a
//     pointer where its zero value is a value it must be able to state.
//   - Nothing is abbreviated. Every digest, every revision and every id goes
//     out whole (§8, ADR-0047): a consumer resolves what it is handed against a
//     git object or a sha256sum, and a shortened value is one it has to go
//     somewhere else to complete. That is a property of what a row carries, so
//     it is held by each row type rather than by the stream.
//
// Cells is the line the row contributes to its command's table, in that table's
// column order. It is empty where the page renders the row some other way or
// does not render it at all — the terminal row below is the row every stream
// carries and no page has a line for.
type Row interface {
	Cells() []string
}

// ResultRow is the terminal row of every stream that is not a Run's (§9). The
// terminal row is always written, including after zero rows: its absence is
// what says the stream was cut off, so a stream that ends without one is a
// stream a consumer must not trust.
//
// truncated is the marker of a result a limit cut short, and it is the whole of
// what this row carries: it is written always, false included, a result row
// with no marker having nothing left to say.
type ResultRow struct {
	Type      string `json:"type"`
	Truncated bool   `json:"truncated"`
}

// NewResultRow is the terminal row for a stream that carried everything it
// found, or that a limit cut short.
func NewResultRow(truncated bool) ResultRow {
	return ResultRow{Type: "result", Truncated: truncated}
}

// Cells is empty: the terminal row is the wire's framing and has no line on the
// page. What a truncated result says to a human is a line its command writes,
// on the stream stderr carries the narration on (§9).
func (r ResultRow) Cells() []string { return nil }

// WriteJSON writes the stream as NDJSON: one compact object per line — no space
// after a separator — terminating in terminal, which is a parameter rather than
// something a caller remembers to append, a stream that opened and did not
// terminate being the one thing this surface may not do (§8). A row that will
// not encode stops the stream where it stands, and the unterminated wire is
// then the true report: the stream was cut off.
//
// HTML escaping is off. The wire carries an artefact's own bytes, and a message
// quoting a & or a < is a message a consumer reads back as it was written.
func WriteJSON(w io.Writer, rows []Row, terminal Row) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return enc.Encode(terminal)
}

// WriteTable writes the stream as the human page: header, then one line per row
// that has one, aligned in columns. header is the command's own, and so is the
// order of the cells beneath it — what the renderer holds is that the page and
// the wire are written from one list of rows.
//
// A page whose rows contribute no line at all is written as nothing, header
// included: what stands in place of an empty table — a confirmation line, a
// count, a sentence naming what was looked for — is the command's own and is
// not a header over no rows.
func WriteTable(w io.Writer, header []string, rows []Row) error {
	var lines [][]string
	for _, row := range rows {
		if cells := row.Cells(); len(cells) > 0 {
			lines = append(lines, cells)
		}
	}
	if len(lines) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, cells := range lines {
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}
