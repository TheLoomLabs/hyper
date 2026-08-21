// Package problem defines the one shape a static-verification finding takes:
// a value, not a message. §12 fixes the members — file, line, column, field,
// error_code, message — and both of hyper check's renderings (§9) come from
// sorting and formatting one slice of this type.
package problem

import (
	"cmp"
	"sort"
	"strings"
)

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
		return Compare(problems[i], problems[j]) < 0
	})
}

// Compare is that same order as a comparison, for the one caller that orders
// something which is not a Problem: a Run's Refusal is an ordered array whose
// order §7 fixes as "the order check prints in", and a second spelling of this
// comparison is where the day comes that a Refusal and a `check` over the same
// repository list the same faults differently (§7, §9).
func Compare(a, b Problem) int {
	switch {
	case a.File != b.File:
		return strings.Compare(a.File, b.File)
	case a.Line != b.Line:
		return cmp.Compare(a.Line, b.Line)
	case a.Column != b.Column:
		return cmp.Compare(a.Column, b.Column)
	}
	return strings.Compare(a.ErrorCode, b.ErrorCode)
}
