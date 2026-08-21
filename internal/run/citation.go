package run

import (
	"fmt"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Where a Refusal at a Step's Expansion points, and how it names what it found
// (§7, §8, ADR-0061, issue #139).
//
// The four checks that decide at an Expansion are the only ones in the closed
// set that cite a Step that **ran** — every other member declines before Step 1,
// where the Step a Refusal names has no file in the entry at all. That is a
// property of those four codes and not of the coordinate: `step` is an artefact
// coordinate everywhere, and this file builds one the same way for all of them.
//
// It is beside the Expansion rather than inside it because the two change for
// different reasons: what a selector resolves to is §5's and §6's, and where a
// Refusal points and what it says is §7's and §8's.

// expansionMember is how a collision names what resolved an identity: the
// member, and the Step itself where there is no selector and so no member to
// name — the comparand that reaches a Step carrying no `over:` (§6).
func expansionMember(name string, cited citation) string {
	if name != "" {
		return name
	}
	if cited.id != "" {
		return "step " + cited.id
	}
	return "this step"
}

// citation is where a Refusal at an Expansion points: the Procedure the Step
// was authored in, the Step's coordinate in it, and what that Step was bound
// to.
//
// The Step it carries is an **artefact coordinate and never an execution
// fact**, which is the rule every Refusal is written under — here the Step it
// names does have a file in the entry, and that is a property of these four
// codes rather than of the coordinate (§7, ADR-0061).
type citation struct {
	file       string
	step       int
	id         string
	index      int
	line       int
	field      string
	operation  string
	target     string
	selectorAt int
}

// citation is the Step's coordinate, read once per Step rather than per member.
//
// The selector arrives already read rather than being parsed a second time from
// the node: one reading of `over:` per Step is what keeps the line a Refusal
// cites and the form the Expansion resolved from being two answers about one
// key.
//
// **The file and the index are the Step's own, not the Run's.** A Step reached
// through a nested invocation was authored in the invoked Procedure's file and
// sits at its own position in that file's `steps:`, and that pair is what an
// edit would reach — where `step` beside them is the position in the Run, which
// is the number the Step table's first column renders (§7, §8, issue #141).
func (r run) citation(authored sequenced, position int, over selector) citation {
	return citation{
		file: authored.Declared.Path, step: position, id: authored.ID, index: authored.Index,
		line: authored.Line, operation: authored.Operation, target: authored.Target,
		selectorAt: over.Line,
	}
}

// at is the citation pointing at one line inside the Step, under the field path
// §8's remediation notation writes: `steps[2].over.assets[1].older_than`.
func (c citation) at(line int, field string) citation {
	c.line, c.field = line, fmt.Sprintf("steps[%d].%s", c.index, field)
	return c
}

// selector is the citation pointing at the Step's `over:` line, which is what
// an identity collision cites: the population is what an edit would narrow, and
// on a Step carrying no selector it is the Step's own line.
func (c citation) selector() citation {
	c.field = fmt.Sprintf("steps[%d]", c.index)
	if c.selectorAt != 0 {
		c.line, c.field = c.selectorAt, c.field+".over"
	}
	return c
}

// refusal is one check declining at a Step's Expansion, in the shape §7 holds a
// Refusal member in and §8 renders a row from.
func (r run) refusal(code, message string, cited citation) Refusal {
	return Refusal{
		RefusalMember: store.RefusalMember{
			ErrorCode: code,
			File:      cited.file,
			Line:      cited.line,
			Field:     cited.field,
			Message:   message,
			Step:      cited.step,
			StepID:    cited.id,
		},
		Operation: cited.operation,
		Target:    cited.target,
	}
}
