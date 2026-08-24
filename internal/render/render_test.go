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

// The truncation marker is the terminal row's third shape (§9, §12, issue
// #162). `false` and `true` are the two the row has carried since the stream
// landed, and the object beside them is what the Inspection commands write:
// they range over the record's two axes and each of them has parameters that
// narrow the one a limit cut, so the marker names the axis, the counts, and the
// parameters that would make the next call a narrower question.
//
// The three are asserted here as one table rather than one case each, because
// what the member is is *one of three shapes* and the way to see that is to put
// them side by side.
func TestResultRow_TheThreeShapesOfTheTruncationMember(t *testing.T) {
	for name, c := range map[string]struct {
		row  render.ResultRow
		line string
	}{
		"a result nothing cut": {
			row:  render.NewResultRow(false),
			line: `{"type":"result","truncated":false}`,
		},
		"a result a limit cut, on a namespace with no axis to name": {
			row:  render.NewResultRow(true),
			line: `{"type":"result","truncated":true}`,
		},
		"a result a limit cut on an axis": {
			row: render.NewTruncatedResultRow(render.TruncationMarker{
				Axis:     render.AxisTime,
				Returned: 200,
				Dropped:  2840,
				Hint:     "narrow with --since, or --between on changes",
			}),
			line: `{"type":"result","truncated":{"axis":"time","returned":200,"dropped":2840,"hint":"narrow with --since, or --between on changes"}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := render.WriteJSON(&buf, nil, c.row); err != nil {
				t.Fatal(err)
			}
			if got, want := buf.String(), c.line+"\n"; got != want {
				t.Errorf("WriteJSON() = %q, want %q", got, want)
			}
		})
	}
}

// TestTruncationMarker_CarriesTheIdentityAxisByItsOwnName is §12's other
// member, which is the whole of what the closed pair has to say about spelling:
// `identity` is §7's word for the (Target, Definition, name) triple, and the
// wire carries that word and not `name`.
func TestTruncationMarker_CarriesTheIdentityAxisByItsOwnName(t *testing.T) {
	var buf bytes.Buffer
	marker := render.TruncationMarker{
		Axis:     render.AxisIdentity,
		Returned: 50,
		Dropped:  3950,
		Hint:     "narrow with --target, --definition or --name",
	}

	if err := render.WriteJSON(&buf, nil, render.NewTruncatedResultRow(marker)); err != nil {
		t.Fatal(err)
	}

	want := `{"type":"result","truncated":{"axis":"identity","returned":50,"dropped":3950,"hint":"narrow with --target, --definition or --name"}}` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

// TestTruncationMarker_AMarkerThatIsNotOneStopsTheStream is what holds the
// marker's four members to what a marker means, and it is the closed pair's
// enforcement among them: a --limit is a positive integer, so a cut returned at
// least one row and dropped at least one, and §12 fixes two axes and no third.
// Every one of these would be a truncated result that reads as complete, or a
// remedy naming nothing — which is the pair of things §9 says this surface may
// never hand back.
//
// It stops the stream where it stands, which is WriteJSON's stated behaviour
// for a row that will not encode. The terminal row is the last row, so what
// reaches the consumer is a stream with no terminal row at all: the wire's own
// signal that it was cut off, and a louder report than the marker would be.
func TestTruncationMarker_AMarkerThatIsNotOneStopsTheStream(t *testing.T) {
	whole := render.TruncationMarker{
		Axis:     render.AxisIdentity,
		Returned: 50,
		Dropped:  3950,
		Hint:     "narrow with --target, --definition or --name",
	}
	for name, c := range map[string]struct {
		marker render.TruncationMarker
		says   string
	}{
		"an axis outside §12's pair": {
			marker: func() render.TruncationMarker {
				m := whole
				m.Axis = render.TruncationAxis("ordinal")
				return m
			}(),
			says: "ordinal",
		},
		"no axis at all": {
			marker: func() render.TruncationMarker {
				m := whole
				m.Axis = ""
				return m
			}(),
			says: "axis",
		},
		"a cut that returned nothing": {
			marker: func() render.TruncationMarker {
				m := whole
				m.Returned = 0
				return m
			}(),
			says: "returned",
		},
		"a cut that dropped nothing": {
			marker: func() render.TruncationMarker {
				m := whole
				m.Dropped = 0
				return m
			}(),
			says: "dropped",
		},
		"a remedy naming nothing": {
			marker: func() render.TruncationMarker {
				m := whole
				m.Hint = ""
				return m
			}(),
			says: "hint",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer

			err := render.WriteJSON(&buf, nil, render.NewTruncatedResultRow(c.marker))
			if err == nil {
				t.Fatalf("WriteJSON() wrote %q and answered no error", buf.String())
			}
			if got := err.Error(); !strings.Contains(got, c.says) {
				t.Errorf("WriteJSON() error = %q, want it to name %q — the member it refused", got, c.says)
			}
		})
	}
}

// TestTruncationMarker_WritesEveryMemberEvenWhereOneIsSmall holds the marker
// off the ordinary absence rule (§7). A member a row does not carry is absent
// from the object, and a marker carries all four always: the counts are numbers
// a reader subtracts and compares, and one omitted at its smallest value would
// be read as *unknown* where the fact is *one*.
func TestTruncationMarker_WritesEveryMemberEvenWhereOneIsSmall(t *testing.T) {
	var buf bytes.Buffer
	marker := render.TruncationMarker{Axis: render.AxisTime, Returned: 1, Dropped: 1, Hint: "narrow with --since"}

	if err := render.WriteJSON(&buf, nil, render.NewTruncatedResultRow(marker)); err != nil {
		t.Fatal(err)
	}

	want := `{"type":"result","truncated":{"axis":"time","returned":1,"dropped":1,"hint":"narrow with --since"}}` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

// TestTruncatedResultRow_HasNoLineOnThePage is the terminal row's own rule
// holding for the shape that carries the most: the marker's human counterpart
// is a line its command writes on stderr, and a page that grew a row here would
// be putting narration on stdout, where only the answer goes (§9).
func TestTruncatedResultRow_HasNoLineOnThePage(t *testing.T) {
	row := render.NewTruncatedResultRow(render.TruncationMarker{Axis: render.AxisIdentity, Returned: 1, Dropped: 2, Hint: "narrow with --name"})
	if cells := row.Cells(); len(cells) != 0 {
		t.Errorf("Cells() = %q, want none", cells)
	}
}

// TestResultRow_ATruncationNobodySetIsTheBareFalse is the member's zero value,
// and it is the shape the boolean this replaced has always defaulted to. The
// terminal row is the wire's framing, so an unset one may write `false` — the
// stream carried everything — and may never write `null`, which is a terminal
// row saying nothing about whether the answer is whole.
func TestResultRow_ATruncationNobodySetIsTheBareFalse(t *testing.T) {
	var buf bytes.Buffer
	if err := render.WriteJSON(&buf, nil, render.ResultRow{Type: "result"}); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), `{"type":"result","truncated":false}`+"\n"; got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}
}

// TestWriteAligned_AStackedCellOpensOutAndTheColumnsStayAligned is the one
// geometry §8's `THE CODE MOVED` needs and no page before it had (issue #171).
//
// **The column widens and the row wraps**: a Cadence's cell stacks the
// expression, the phrase and the rate, and a selector's stacks its form over
// its members, and neither is truncated. What that costs the renderer is that a
// cell carrying a newline contributes one physical line per segment while the
// cells beside it stand empty beneath their own first — a stacked cell that
// reached the tabwriter whole would take the width of its longest segment and
// put the columns to its right on a line of their own.
func TestWriteAligned_AStackedCellOpensOutAndTheColumnsStayAligned(t *testing.T) {
	var buf bytes.Buffer
	err := render.WriteAligned(&buf, "  ", [][]string{
		{"SUBJECT", "FACT", "FROM", "TO"},
		{"procedure retire", "cadence", "0 0 1 * *\n00:00 UTC on the 1st\n1 run/month", "*/5 * * * *\nevery 5 minutes\n≈8800 runs/month"},
		{"—", "repository revision", "1f0a3d7", "88bc402"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := "  SUBJECT           FACT                 FROM                  TO\n" +
		"  procedure retire  cadence              0 0 1 * *             */5 * * * *\n" +
		"                                         00:00 UTC on the 1st  every 5 minutes\n" +
		"                                         1 run/month           ≈8800 runs/month\n" +
		"  —                 repository revision  1f0a3d7               88bc402\n"
	if got := buf.String(); got != want {
		t.Errorf("WriteAligned() =\n%s\nwant\n%s", got, want)
	}
}

// TestWriteAligned_ABlockWithNothingStackedComesBackAsItWentIn is the fence
// around that: every page that landed before the third table carries no cell
// with a newline in it, and none of them may have moved by a byte.
func TestWriteAligned_ABlockWithNothingStackedComesBackAsItWentIn(t *testing.T) {
	var buf bytes.Buffer
	if err := render.WriteAligned(&buf, "", [][]string{{"a", "bb"}, {"ccc", "d"}}); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "a    bb\nccc  d\n"; got != want {
		t.Errorf("WriteAligned() = %q, want %q", got, want)
	}
}
