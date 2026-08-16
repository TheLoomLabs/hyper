// Package problem defines the one shape a static-verification finding takes:
// a value, not a message. §12 fixes the members — file, line, column, field,
// error_code, message — and both of hyper check's renderings (§9) come from
// sorting and formatting one slice of this type.
package problem

import "sort"

// Problem is one row of hyper check's output: a single fault at a single
// position. It carries no message-only shape — file, line and field are what
// make the next act an edit rather than a search (issue #88).
type Problem struct {
	// File is the artefact's path, relative to the repository root, with
	// forward slashes regardless of platform.
	File string
	// Line and Column are 1-indexed positions in File. Column rides on the
	// wire only — the human table does not render it (§9).
	Line   int
	Column int
	// Field is a path into the artefact, in the notation §8's remediation
	// table uses: "steps[2].bound", "auth.token". It is empty where the
	// fault has no position more specific than the file itself.
	Field string
	// ErrorCode is one member of §12's closed error_code set.
	ErrorCode string
	// Message is free text describing the fault.
	Message string
}

// Sort orders problems by file path, then by line, then by column, then by
// error_code — the order §9's "check [path...]" states.
func Sort(problems []Problem) {
	sort.SliceStable(problems, func(i, j int) bool {
		a, b := problems[i], problems[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.ErrorCode < b.ErrorCode
	})
}
