package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

func TestWriteJSON_EmitsOneCompactRowPerProblemThenResult(t *testing.T) {
	var buf bytes.Buffer
	problems := []problem.Problem{
		{File: "procedures/retire.yaml", Line: 34, Column: 7, Field: "steps[2].bound", ErrorCode: "strict-yaml-violation", Message: "an anchor is not part of the authoring format"},
	}
	if err := WriteJSON(&buf, problems); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}

	wantProblem := `{"type":"problem","file":"procedures/retire.yaml","line":34,"column":7,"field":"steps[2].bound","error_code":"strict-yaml-violation","message":"an anchor is not part of the authoring format"}`
	if lines[0] != wantProblem {
		t.Errorf("row 0 = %q, want %q", lines[0], wantProblem)
	}

	wantResult := `{"type":"result","truncated":false}`
	if lines[1] != wantResult {
		t.Errorf("row 1 = %q, want %q", lines[1], wantResult)
	}
}

func TestWriteJSON_EmitsTerminalRowEvenWhenClean(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	got := strings.TrimRight(buf.String(), "\n")
	want := `{"type":"result","truncated":false}`
	if got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

func TestWriteTable_EmptyWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("WriteTable() wrote %q, want nothing for a clean run", buf.String())
	}
}

func TestWriteTable_CarriesFileLineFieldCodeMessageNotColumn(t *testing.T) {
	var buf bytes.Buffer
	problems := []problem.Problem{
		{File: "procedures/retire.yaml", Line: 34, Column: 7, Field: "steps[2].bound", ErrorCode: "strict-yaml-violation", Message: "boom"},
	}
	if err := WriteTable(&buf, problems); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "procedures/retire.yaml") ||
		!strings.Contains(out, "34") ||
		!strings.Contains(out, "steps[2].bound") ||
		!strings.Contains(out, "strict-yaml-violation") ||
		!strings.Contains(out, "boom") {
		t.Fatalf("WriteTable() = %q, missing an expected field", out)
	}
	if strings.Contains(out, "\t7\t") {
		t.Errorf("WriteTable() rendered the column, want it wire-only: %q", out)
	}
}
