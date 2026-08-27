package cli_test

import (
	"slices"
	"strings"
	"testing"
)

// Two closed sets, each reached by a case (§9, issue #204).
//
// This is error_code_coverage_test.go's own shape one surface over: *every
// member of a closed set reached by a case*, named where it is not. What that
// file asks of §12's error codes, this asks of §9's two closed sets — the
// thirteen tools, and the shapes an answer can take — and it asks it the same
// way, off what the corpus has checked in rather than off the code that would
// produce it.
//
// A coverage fence is worth its own file for the reason that one is: it is not
// an assertion about any answer, it is an assertion about the corpus, and the
// day it fails the edit is a case to write rather than a golden to regenerate.

// TestGoldenCorpora_EveryToolHasACase names the thirteen and fails where one is
// driven by nothing.
//
// **It reads the tool set rather than a list**, which is what makes it
// self-maintaining: the day a fourteenth tool is declared, this fails naming
// it, and the case that answers the failure is the case that tool needed. The
// set itself is transcribed and held against §9 one file over
// (mcp_tools_test.go, internal/mcp's tool_set_test.go), so nothing here has to
// restate which thirteen they are.
func TestGoldenCorpora_EveryToolHasACase(t *testing.T) {
	driven := map[string]int{}
	for _, c := range goldenCases(t) {
		if c.call != nil {
			driven[c.call.Tool]++
		}
	}

	published := map[string]bool{}
	var missing []string
	for _, tool := range listing(t) {
		published[tool.Name] = true
		if driven[tool.Name] == 0 {
			missing = append(missing, tool.Name)
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Errorf("%d of %d tools have no case under testdata/mcp/: %v", len(missing), len(published), missing)
	}

	// The other half of the same question, and the one a walk of the corpus
	// is the only thing that can answer: a case naming a tool the server
	// does not publish. It would fail its own golden today, and it would
	// fail it saying the tool is unknown rather than saying the corpus
	// names something §9 does not.
	for tool := range driven {
		if !published[tool] {
			t.Errorf("a case under testdata/mcp/ calls %q, which tools/list does not answer", tool)
		}
	}
}

// envelopeShapes is §9's closed set of things an answer can be, in the order
// the specification's own mapping walks them: the ordinary return, the answer
// that carries problems the command found, the guardrail declining, and the
// three §12 outcomes a Run can end in — of which `completed` is the ordinary
// return and the other two are their own shapes. The protocol error closes it,
// being the one answer that is no envelope at all.
//
// Each is a **path through envelopeOf** rather than a value of any one member,
// which is why the set is worth enumerating: `isError` is one bit and cannot
// carry three states, `outcome` is absent on ten of the thirteen tools, and no
// single key tells a Refusal from a report of problems found. What tells them
// apart is the combination, and a corpus with no case for one of them is a
// combination this surface has never actually answered.
var envelopeShapes = []string{
	"ordinary",
	"problems found",
	"guardrail declining",
	"refused",
	"failed",
	"protocol error",
}

// TestGoldenCorpora_EveryEnvelopeShapeHasACase names each shape and fails where
// one is reached by no case (§9).
//
// It reads what is **checked in**, on the coverage fences' own footing one
// package over: a shape is what a golden says, so a mapping that stopped
// producing one would leave this failing rather than leave a case quietly
// asserting something else.
func TestGoldenCorpora_EveryEnvelopeShapeHasACase(t *testing.T) {
	reached := map[string][]string{}
	for _, c := range goldenCases(t) {
		if c.call == nil {
			continue
		}
		shape := envelopeShapeOf(t, c)
		reached[shape] = append(reached[shape], c.name)
	}

	var missing []string
	for _, shape := range envelopeShapes {
		if len(reached[shape]) == 0 {
			missing = append(missing, shape)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of %d envelope shapes have no case under testdata/mcp/: %q", len(missing), len(envelopeShapes), missing)
	}
}

// envelopeShapeOf reads one call case's checked-in answer and says which of
// §9's shapes it is.
//
// The reading is §9's mapping table, taken in the order that makes each arm
// decidable. `outcome` comes first because it is the discriminator §9 names —
// it is `refused` or `failed` or it is a completed Run — and what is left is
// the ten tools that carry no `outcome` key at all, where `isError` says
// whether the caller got what they asked for and the text block says why: a
// rendering opening `refused:` is a guardrail declining, and anything else is a
// command reporting problems it found (§9, refusal.go).
func envelopeShapeOf(t *testing.T, c goldenCase) string {
	t.Helper()

	if c.answersAProtocolError() {
		return "protocol error"
	}

	held := readEnvelope(t, c.dir)
	switch outcome := held.StructuredContent.Outcome; outcome {
	case "refused", "failed":
		return outcome
	case "completed":
		// §12's third, which is the ordinary return wearing a Run's
		// terminal fact. A rehearsal that halted at its first effect is
		// one of these: the answer is partial and says so in the
		// Dispositions rather than in the outcome (§7, §12).
		return "ordinary"
	case "":
		// The ten tools that are not a Run, which carry no `outcome` key
		// at all.
	default:
		// An outcome outside §12's triple is a golden this reading
		// cannot classify, which is a fault rather than a shape: the
		// enumeration below would gain a member nobody named.
		t.Fatalf("case %s carries the outcome %q, which is no member of §12's triple", c.name, outcome)
	}

	if !held.IsError {
		return "ordinary"
	}
	if strings.HasPrefix(textBlock(t, c.dir), refusalOpening) {
		return "guardrail declining"
	}
	return "problems found"
}
