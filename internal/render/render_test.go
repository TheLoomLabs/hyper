package render_test

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/render"
)

// stubRow is a row type declared outside the renderer, which is where row types
// live: a command owns the rows it writes and the renderer owns the stream they
// go out on. It is named for being a stub rather than for anything §8 puts on
// the wire — no row type in the tool is spelt "stub", so nothing here can be
// mistaken for the shape of a row a command actually writes.
//
// It carries a member written only where the fact is there, which is the
// ordinary absence rule (§7, §9), and it renders fewer members on the page than
// it carries on the wire, which every row type in the tool is free to do.
type stubRow struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Note  string `json:"note,omitempty"`
}

func (r stubRow) Cells() []string { return []string{r.Name, strconv.Itoa(r.Count)} }

func newStubRow(name string, count int) stubRow {
	return stubRow{Type: "stub", Name: name, Count: count}
}

func TestWriteJSON_WritesOneCompactObjectPerRowAndTerminatesInTheTerminalRow(t *testing.T) {
	var buf bytes.Buffer
	rows := []render.Row{newStubRow("uptime", 2), newStubRow("shell", 6)}

	if err := render.WriteJSON(&buf, rows, render.NewResultRow(false)); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`{"type":"stub","name":"uptime","count":2}`,
		`{"type":"stub","name":"shell","count":6}`,
		`{"type":"result","truncated":false}`,
	}
	if got := lines(t, buf.String()); !slices.Equal(got, want) {
		t.Errorf("WriteJSON() =\n%q\nwant\n%q", got, want)
	}
}

func TestWriteJSON_WritesTheTerminalRowAfterZeroRows(t *testing.T) {
	var buf bytes.Buffer
	if err := render.WriteJSON(&buf, nil, render.NewResultRow(false)); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), `{"type":"result","truncated":false}`+"\n"; got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

func TestWriteJSON_OmitsAMemberTheRowDoesNotCarryRatherThanWritingItEmpty(t *testing.T) {
	var buf bytes.Buffer
	carried := newStubRow("uptime", 2)
	carried.Note = "installed"
	absent := newStubRow("shell", 6)

	if err := render.WriteJSON(&buf, []render.Row{carried, absent}, render.NewResultRow(false)); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`{"type":"stub","name":"uptime","count":2,"note":"installed"}`,
		`{"type":"stub","name":"shell","count":6}`,
		`{"type":"result","truncated":false}`,
	}
	if got := lines(t, buf.String()); !slices.Equal(got, want) {
		t.Errorf("WriteJSON() =\n%q\nwant\n%q\nan absent member is absent, not null and not empty", got, want)
	}
}

func TestWriteJSON_LeavesHTMLPunctuationAsItWasWritten(t *testing.T) {
	var buf bytes.Buffer
	rows := []render.Row{newStubRow(`a <b> & c`, 1)}

	if err := render.WriteJSON(&buf, rows, render.NewResultRow(false)); err != nil {
		t.Fatal(err)
	}

	want := []string{
		`{"type":"stub","name":"a <b> & c","count":1}`,
		`{"type":"result","truncated":false}`,
	}
	if got := lines(t, buf.String()); !slices.Equal(got, want) {
		t.Errorf("WriteJSON() =\n%q\nwant\n%q", got, want)
	}
}

func TestNewResultRow_CarriesTheTruncationMarkerItWasGiven(t *testing.T) {
	var buf bytes.Buffer
	if err := render.WriteJSON(&buf, nil, render.NewResultRow(true)); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), `{"type":"result","truncated":true}`+"\n"; got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

func TestWriteTable_WritesTheHeaderAndOneAlignedLinePerRow(t *testing.T) {
	var buf bytes.Buffer
	rows := []render.Row{newStubRow("uptime", 2), newStubRow("cloudflare-dns", 11)}

	if err := render.WriteTable(&buf, []string{"NAME", "OPERATIONS"}, rows); err != nil {
		t.Fatal(err)
	}

	want := "NAME            OPERATIONS\n" +
		"uptime          2\n" +
		"cloudflare-dns  11\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteTable() =\n%q\nwant\n%q", got, want)
	}
}

// TestWriteTable_NoLineEndsInPadding is what a cell a row does not carry looks
// like on the page: the columns before it are aligned and the line stops where
// its last written cell does. Padding a reader cannot see is noise in a diff
// and a trailing space in a pipe, and a row whose last cell is empty is
// ordinary — a Target declaring no credential is §9's own example.
func TestWriteTable_NoLineEndsInPadding(t *testing.T) {
	var buf bytes.Buffer
	rows := []render.Row{cellRow{"cloudflare-prod", "token=CLOUDFLARE_API_TOKEN"}, cellRow{"local", ""}}

	if err := render.WriteTable(&buf, []string{"NAME", "CREDENTIALS"}, rows); err != nil {
		t.Fatal(err)
	}

	want := "NAME             CREDENTIALS\n" +
		"cloudflare-prod  token=CLOUDFLARE_API_TOKEN\n" +
		"local\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteTable() =\n%q\nwant\n%q", got, want)
	}
}

// cellRow is a row that is nothing but its cells, for the cases about the page
// rather than about the wire.
type cellRow []string

func (r cellRow) Cells() []string { return r }

func TestWriteTable_WritesNothingWhereNoRowHasALine(t *testing.T) {
	var buf bytes.Buffer
	if err := render.WriteTable(&buf, []string{"NAME", "OPERATIONS"}, nil); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "" {
		t.Errorf("WriteTable() = %q, want nothing: what stands where a page has no rows is its command's own line", got)
	}
}

func TestWriteTable_SkipsARowThatContributesNoLine(t *testing.T) {
	var buf bytes.Buffer
	rows := []render.Row{newStubRow("uptime", 2), render.NewResultRow(false)}

	if err := render.WriteTable(&buf, []string{"NAME", "OPERATIONS"}, rows); err != nil {
		t.Fatal(err)
	}

	want := "NAME    OPERATIONS\nuptime  2\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteTable() =\n%q\nwant\n%q", got, want)
	}
}

// lines is the stream as the lines a consumer reads off it, which is the one
// shape every assertion above holds the wire in: NDJSON is a line-oriented
// format, and a stream that does not end in a newline has a last row nothing
// terminated.
func lines(t *testing.T, stream string) []string {
	t.Helper()
	if stream == "" {
		return nil
	}
	if !strings.HasSuffix(stream, "\n") {
		t.Fatalf("the stream does not end in a newline: %q", stream)
	}
	return strings.Split(strings.TrimSuffix(stream, "\n"), "\n")
}
