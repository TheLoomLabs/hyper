package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
)

// The seam between a command's answer and the bytes it becomes (§9, ADR-0026,
// issue #194).
//
// The corpus is what proves the CLI's destination writes what the two writers
// and the `--json` flag wrote — every checked-in golden is that assertion, and
// it is why this refactor could only be wrong one way. What is here is the
// three claims a golden case cannot make: that the **form** is the
// destination's and reaches it from the one place the flag is read; that a
// Refusal's own form is carried rather than read back off its members; and that
// an answer which could not be written is reported where it is reported. A case
// is an argv against a command, and none of the three is a command's behaviour.

// answered is a destination's answer as bytes, with the narration beside it.
// The two are read apart because the whole of what this seam fixes is which of
// them a thing goes to.
func answered(t *testing.T, args []string, rows []render.Row) (answer, told string) {
	t.Helper()
	var out, errs bytes.Buffer
	_, to, code := parseArgs("runs", args, parameters{limit: defaultListLimit}, environment(nil), streams{stdout: &out, stderr: &errs})
	if code != 0 {
		t.Fatalf("parseArgs() = %d, want 0; the narration said %q", code, errs.String())
	}
	page := func(w io.Writer, rows []render.Row) error {
		_, err := io.WriteString(w, "the page\n")
		return err
	}
	if code := writeAnswer("runs", to, rows, render.NewResultRow(false), page); code != 0 {
		t.Fatalf("writeAnswer() = %d, want 0; the narration said %q", code, errs.String())
	}
	return out.String(), errs.String()
}

// A destination the caller named --json on writes the rows; one they did not
// writes the command's page. Both readings come off the same parse, which is
// the point: the flag names a form, the form is the destination's, and the
// command in between holds no opinion about either.
func TestParseArgs_AnswersTheFormTheCallerNamed(t *testing.T) {
	rows := []render.Row{render.NewResultRow(false)}

	page, told := answered(t, nil, rows)
	if page != "the page\n" {
		t.Errorf("with no --json the answer is %q, want the command's page", page)
	}
	if told != "" {
		t.Errorf("the narration carries %q; an answer that was written says nothing beside it", told)
	}

	wire, told := answered(t, []string{"--json"}, rows)
	if strings.Contains(wire, "the page") {
		t.Errorf("with --json the answer is %q, want the row stream", wire)
	}
	if told != "" {
		t.Errorf("the narration carries %q; an answer that was written says nothing beside it", told)
	}
	if lines := strings.Count(wire, "\n"); lines != 2 {
		t.Errorf("the row stream is %q — %d lines, want the row and its terminal row", wire, lines)
	}
}

// A Refusal is not a row, so it goes to the narration and the answer stays
// silent — in **both** forms, which is the half of the rule a `--json` caller
// would otherwise discover by parsing an empty stream (§9, gate.go).
func TestRefuse_LeavesTheAnswerSilentInBothForms(t *testing.T) {
	for name, args := range map[string][]string{
		"the page form": nil,
		"the wire form": {"--json"},
	} {
		t.Run(name, func(t *testing.T) {
			var out, errs bytes.Buffer
			_, to, code := parseArgs("runs", args, parameters{limit: defaultListLimit}, environment(nil), streams{stdout: &out, stderr: &errs})
			if code != 0 {
				t.Fatalf("parseArgs() = %d, want 0", code)
			}

			if code := refuse(to, "store-absent", "no hyper/store branch in this repository — hyper store init"); code != ExitRefused {
				t.Errorf("refuse() = %d, want %d", code, ExitRefused)
			}
			if out.Len() != 0 {
				t.Errorf("the answer carries %q; a Refusal is not a row and opens no stream", out.String())
			}
			if want := "refused: store-absent\n  no hyper/store branch"; !strings.HasPrefix(errs.String(), want) {
				t.Errorf("the narration is %q, want it to begin %q", errs.String(), want)
			}
		})
	}
}

// A Refusal's form is carried and never read back off its members. A check
// that cites an artefact and happens to carry no line still renders §8's
// notes form rather than being downgraded to the two-line one — the caret and
// the `EDIT ONE OF` are the whole path back to a passing review, and a Refusal
// that lost them to a missing coordinate would lose the remedy with them
// (refusalForm).
func TestRefuseProblems_KeepsItsFormWithNoCoordinate(t *testing.T) {
	var out, errs bytes.Buffer
	to := streams{stdout: &out, stderr: &errs}

	code := refuseProblems(to, t.TempDir(), []problem.Problem{{
		ErrorCode: "schema-mismatch",
		Message:   "a check with nowhere to point",
	}})

	if code != ExitRefused {
		t.Errorf("refuseProblems() = %d, want %d", code, ExitRefused)
	}
	if out.Len() != 0 {
		t.Errorf("the answer carries %q; a Refusal is not a row", out.String())
	}
	if want := "refused: schema-mismatch\n\n"; !strings.HasPrefix(errs.String(), want) {
		t.Fatalf("the narration is %q, want it to begin %q — the two-line form is a\n"+
			"fact about an invocation, and this is a check about an artefact", errs.String(), want)
	}
	if want := "= a check with nowhere to point"; !strings.Contains(errs.String(), want) {
		t.Errorf("the narration is %q, want the message as §8's note where no caret was drawn", errs.String())
	}
}

// declining is a writer that takes nothing, which is the one way an answer that
// was built can still fail to be written.
type declining struct{}

func (declining) Write([]byte) (int, error) { return 0, errors.New("the stream would not take it") }

// An answer that could not be written is reported on the narration and exits 2,
// and it is writeAnswer that holds that rule rather than each of the nineteen
// commands behind it (§9).
func TestWriteAnswer_ReportsAStreamItCouldNotWrite(t *testing.T) {
	var errs bytes.Buffer
	to := streams{stdout: declining{}, stderr: &errs}
	page := func(w io.Writer, rows []render.Row) error {
		_, err := io.WriteString(w, "the page\n")
		return err
	}

	if code := writeAnswer("runs", to, nil, render.NewResultRow(false), page); code != ExitUsage {
		t.Errorf("writeAnswer() = %d, want %d", code, ExitUsage)
	}
	if want := "hyper runs: the stream would not take it\n"; errs.String() != want {
		t.Errorf("the narration is %q, want %q", errs.String(), want)
	}
}
