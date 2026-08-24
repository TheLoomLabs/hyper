package artefact_test

import (
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// The change column's supply (§8, issue #168): which of the working tree's
// lines the range touched, and where a deletion is cited.

// touchedProcedure is the Procedure §8 renders, as the working tree holds it.
// Its shape is what the anchor rule is stated against — a `bound:` at a Step's
// own column, a Step at the sequence's — so the cases below edit it rather than
// minting a file per case.
const touchedProcedure = `kind: procedure
procedure: retire
targets: [local, staging]
cadence: "0 3 * * 1"
steps:
  - id: probe
    definition: uptime
    operation: check_http
    target: local

  - id: retire
    definition: hetzner
    operation: delete_server
    target: staging
    over:
      assets:
        - field: labels.role
          equals: preview
    bound: 5
`

// touched reads the range over that Procedure, the working tree parsed as
// `hyper` parses one.
func touched(t *testing.T, baseline, working string) artefact.Touched {
	t.Helper()

	root, _, ok := yamlsubset.Parse("procedures/retire.yaml", []byte(working))
	if !ok {
		t.Fatalf("the working tree did not parse")
	}
	return artefact.ReadTouched(artefact.SourceLines([]byte(baseline)), artefact.SourceLines([]byte(working)), root)
}

// marked is the lines the column marks, as a set a case can state whole: what
// the column says is *these lines and no others*, and a case asserting one mark
// would pass on a renderer that marked the file.
func marked(t *testing.T, got artefact.Touched, want ...int) {
	t.Helper()

	for _, line := range want {
		if !got.Marked(line) {
			t.Errorf("line %d is unmarked; the range touched it", line)
		}
	}
	wanted := map[int]bool{}
	for _, line := range want {
		wanted[line] = true
	}
	for line := range got.Lines {
		if !wanted[line] {
			t.Errorf("line %d is marked; nothing on it moved", line)
		}
	}
}

// TestTouched_AnUneditedArtefactMarksNothing is the state a range opens in on
// every artefact nobody has touched since it last ran: the column is drawn,
// because a range opened, and it marks not one line.
func TestTouched_AnUneditedArtefactMarksNothing(t *testing.T) {
	marked(t, touched(t, touchedProcedure, touchedProcedure))
}

// TestTouched_AnEditedLineIsMarkedAndItsNeighboursAreNot is §8's own worked
// example: the Bound the agent widened, and nothing else on the screen.
func TestTouched_AnEditedLineIsMarkedAndItsNeighboursAreNot(t *testing.T) {
	baseline := replaceLine(t, touchedProcedure, "    bound: 5", "    bound: 3")
	marked(t, touched(t, baseline, touchedProcedure), 19)
}

// TestTouched_ANewLineIsMarked reads the second of the column's three cases: a
// line that is new has no counterpart to differ from and is marked all the
// same.
func TestTouched_ANewLineIsMarked(t *testing.T) {
	baseline := removeLine(t, touchedProcedure, "    bound: 5")
	marked(t, touched(t, baseline, touchedProcedure), 19)
}

// TestTouched_ARemovedBoundIsCitedAtItsStep is the anchor rule's first arm: a
// removed line has no number, having no line, and what the deletion is cited at
// is the opening line of the nearest enclosing structure — the Step's own
// `- id:`, which did not itself move.
func TestTouched_ARemovedBoundIsCitedAtItsStep(t *testing.T) {
	working := removeLine(t, touchedProcedure, "    bound: 5")
	got := touched(t, touchedProcedure, working)
	marked(t, got, 11)
	if anchor := got.Anchors[19]; anchor != 11 {
		t.Errorf("the removed bound: is cited at line %d, want the Step's own line 11", anchor)
	}
}

// TestTouched_ARemovedStepIsCitedAtSteps is the rule's other arm, and the one
// that fixes why the anchor is the enclosing structure rather than whatever
// text sits adjacent: the Step is gone, so the line a reader edits to put it
// back is the one that names the sequence.
func TestTouched_ARemovedStepIsCitedAtSteps(t *testing.T) {
	working := `kind: procedure
procedure: retire
targets: [local, staging]
cadence: "0 3 * * 1"
steps:
  - id: probe
    definition: uptime
    operation: check_http
    target: local
`
	got := touched(t, touchedProcedure, working)
	marked(t, got, 5)
	if anchor := got.Anchors[11]; anchor != 5 {
		t.Errorf("the removed Step is cited at line %d, want the steps: line 5", anchor)
	}
}

// TestTouched_ASurplusRemovalInsideOneEditIsStillADeletion is the anchor rule
// holding where an edit did two things at once: this one rewrote `kinds:` and
// deleted `destroy:` in one gap, and only the first has a line it moved to.
//
// The pairing runs from the top and the surplus falls through to the structure,
// which is what keeps the removed claim off *whatever text happens to sit
// adjacent* — the reading §8 names and refuses.
func TestTouched_ASurplusRemovalInsideOneEditIsStillADeletion(t *testing.T) {
	baseline := `kind: definition
definition: preview
provider: cloudflare-dns
kinds: [mutate]
destroy: [delete_dns_record]
targets: [cloudflare-prod]
`
	working := `kind: definition
definition: preview
provider: cloudflare-dns
kinds: [mutate, destroy]
targets: [cloudflare-prod]
`
	got := touched(t, baseline, working)
	marked(t, got, 1, 4)
	if anchor := got.Anchors[4]; anchor != 4 {
		t.Errorf("the rewritten kinds: is cited at line %d, want the line it moved to", anchor)
	}
	if anchor := got.Anchors[5]; anchor != 1 {
		t.Errorf("the removed destroy: is cited at line %d, want the artefact's own first line", anchor)
	}
}

// TestTouched_ARemovedTopLevelKeyIsCitedAtTheArtefactsOwnFirstLine is the
// document read as the structure it is: a key at the top level is a member of
// the artefact, so its deletion is cited where the artefact opens.
func TestTouched_ARemovedTopLevelKeyIsCitedAtTheArtefactsOwnFirstLine(t *testing.T) {
	working := removeLine(t, touchedProcedure, `cadence: "0 3 * * 1"`)
	got := touched(t, touchedProcedure, working)
	marked(t, got, 1)
	if anchor := got.Anchors[4]; anchor != 1 {
		t.Errorf("the removed cadence: is cited at line %d, want the artefact's own first line", anchor)
	}
}

// TestTouched_ARemovedConjunctIsCitedAtTheFormThatListsIt keeps the rule
// honest one level in: a conjunct is a member of the selector's own sequence,
// which `assets:` names.
func TestTouched_ARemovedConjunctIsCitedAtTheFormThatListsIt(t *testing.T) {
	working := `kind: procedure
procedure: retire
targets: [local, staging]
cadence: "0 3 * * 1"
steps:
  - id: retire
    definition: hetzner
    operation: delete_server
    target: staging
    over:
      assets:
        - field: labels.role
          equals: preview
        - field: created_at
          older_than: 14d
    bound: 5
`
	baseline := working
	shortened := removeLine(t, removeLine(t, working, "        - field: created_at"), "          older_than: 14d")
	got := touched(t, baseline, shortened)
	marked(t, got, 11)
}

// TestTouched_ARewrittenArtefactMarksEveryLineOfIt is the far end of the same
// reading: nothing survived, so nothing is unmarked.
func TestTouched_ARewrittenArtefactMarksEveryLineOfIt(t *testing.T) {
	working := "kind: definition\ndefinition: heartbeat\nprovider: uptime\n"
	marked(t, touched(t, touchedProcedure, working), 1, 2, 3)
}

// replaceLine is the artefact with one of its lines written differently, and a
// fatal where the line it names is not in the file: a case editing a line that
// moved is a case asserting nothing.
func replaceLine(t *testing.T, source, line, with string) string {
	t.Helper()
	return editLine(t, source, line, with)
}

// removeLine is the artefact with one of its lines taken out.
func removeLine(t *testing.T, source, line string) string {
	t.Helper()
	return editLine(t, source, line, "")
}

// editLine replaces one whole line, dropping it where the replacement is empty.
func editLine(t *testing.T, source, line, with string) string {
	t.Helper()

	lines := artefact.SourceLines([]byte(source))
	edited := make([]string, 0, len(lines))
	found := false
	for _, held := range lines {
		if held == line && !found {
			found = true
			if with == "" {
				continue
			}
			edited = append(edited, with)
			continue
		}
		edited = append(edited, held)
	}
	if !found {
		t.Fatalf("the fixture holds no line %q", line)
	}
	text := ""
	for _, held := range edited {
		text += held + "\n"
	}
	return text
}
