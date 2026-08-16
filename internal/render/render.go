// Package render is hyper check's one renderer. The human table and the
// --json NDJSON stream are two forms of one thing (ADR-0026): both are built
// from the same sorted []problem.Problem, so the two surfaces cannot state
// different things.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// problemRow is the --json shape of one problem, field order fixed to match
// §9's example verbatim: {"type":"problem","file":...,"line":...,
// "column":...,"field":...,"error_code":...,"message":...}. encoding/json
// marshals struct fields in declaration order, which is what fixes the
// renderer's key order on the wire.
type problemRow struct {
	Type      string `json:"type"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Field     string `json:"field"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// resultRow is the terminal row of every non-Run row stream (§9). check
// takes no --limit and has no truncation axis, so truncated is always false.
type resultRow struct {
	Type      string `json:"type"`
	Truncated bool   `json:"truncated"`
}

// WriteJSON writes one compact NDJSON object per line — no space after a
// separator — terminating in the result row. It is always called, even where
// problems is empty: the terminal row is what says the stream was not cut
// off.
func WriteJSON(w io.Writer, problems []problem.Problem) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, p := range problems {
		row := problemRow{
			Type:      "problem",
			File:      p.File,
			Line:      p.Line,
			Column:    p.Column,
			Field:     p.Field,
			ErrorCode: p.ErrorCode,
			Message:   p.Message,
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return enc.Encode(resultRow{Type: "result", Truncated: false})
}

// WriteTable writes the human rendering: the file, the line, the field, the
// error_code and the message. column rides on the wire only (§9). Nothing is
// written where there are no problems — a clean run's stdout is empty, not a
// header over no rows.
func WriteTable(w io.Writer, problems []problem.Problem) error {
	if len(problems) == 0 {
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "FILE\tLINE\tFIELD\tERROR_CODE\tMESSAGE")
	for _, p := range problems {
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", p.File, p.Line, p.Field, p.ErrorCode, p.Message)
	}
	return tw.Flush()
}
