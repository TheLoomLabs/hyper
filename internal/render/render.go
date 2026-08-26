// Package render is hyper's row stream (issue #110). A command builds an
// ordered list of typed rows, and one function writes them as the --json NDJSON
// stream and one writes them as the human table. The two are two forms of one
// thing (ADR-0026): both are written from one list, so the two surfaces cannot
// state different things.
//
// What is here is the stream and nothing else. A row type belongs to the
// command that writes it, and so does the shape of that command's page — its
// columns, its headers, the line that stands where there are no rows. One
// renderer means one path from rows to bytes, not one table layout for every
// command.
package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// A Row is one row of a stream: one object on the --json wire, and one line on
// the page where its command tabulates it.
//
// A row's wire shape is its own JSON encoding, and three things about that
// encoding are contracts rather than conveniences (§8, §9):
//
//   - type is the first key of the object, and the remaining keys follow the
//     row's declaration order — which is what makes --json stable without a
//     schema. encoding/json marshals a struct's fields in declaration order, so
//     a row holds both halves by declaring its type first. Nothing here holds
//     it at write time: a stream that stopped mid-flight to report a badly
//     declared row would leave the wire cut off, which is the failure it would
//     be defending against. The corpus holds it instead, over the checked-in
//     golden files of every command that opens a stream.
//   - A member the row does not carry is absent from the object entirely rather
//     than written as null or "", the ordinary absence rule being a fact a
//     reader reads (§7). A row declares such a member omitempty, or as a
//     pointer where its zero value is a value it must be able to state.
//   - Nothing is abbreviated. Every digest, every revision and every id goes
//     out whole (§8, ADR-0047): a consumer resolves what it is handed against a
//     git object or a sha256sum, and a shortened value is one it has to go
//     somewhere else to complete. That is a property of what a row carries, so
//     it is held by each row type rather than by the stream.
//
// Cells is the line the row contributes to its command's table, in that table's
// column order. It is empty where the page renders the row some other way or
// does not render it at all — the terminal row below is the row every stream
// carries and no page has a line for.
type Row interface {
	Cells() []string
}

// ResultRow is the terminal row of every stream that is not a Run's (§9). The
// terminal row is always written, including after zero rows: its absence is
// what says the stream was cut off, so a stream that ends without one is a
// stream a consumer must not trust.
//
// truncated is the marker of a result a limit cut short, and it is the whole of
// what this row carries: it is written always, the bare false included, a
// result row with no marker having nothing left to say.
type ResultRow struct {
	Type      string     `json:"type"`
	Truncated Truncation `json:"truncated"`
}

// A Truncation is what `truncated` carries, and §9 fixes three shapes of it:
// the bare false, the bare true, and the marker object (issue #162).
//
// Which shape a command writes is a fact about what it ranges over rather than
// a choice. A namespace listing — `providers`, `targets` — writes the bare
// boolean, because neither namespace has a narrowing parameter and an axis
// there would name nothing a caller could act on. The Inspection commands range
// over the record's two axes and every one of them has parameters that narrow
// the one a limit cut, so they write the marker (§9, §12, ADR-0065).
//
// Its members are unexported and its two doors are the constructors below, so
// the three shapes are all there are: a member free to hold anything at all is
// a member whose --json contract is whatever its last caller passed. Its **zero
// value is the bare false**, which is what the boolean this replaced has always
// defaulted to — a terminal row that reached the wire unset writes `false` and
// never `null`, an absent marker being the one thing this member may not say.
type Truncation struct {
	// bare is the boolean shape, and it is the whole of what goes out
	// where there is no marker: a namespace listing has no axis to name,
	// so the boolean is the entire answer there.
	bare bool
	// marker is the object shape, and where it stands it is the whole
	// answer — the boolean beside it is not set, a marker being already a
	// result something cut and the pair being one fact in two
	// representations that can disagree (§8, ADR-0059). It is a pointer
	// because *no marker* is a state this value must be able to hold.
	marker *TruncationMarker
}

// MarshalJSON writes whichever of the three shapes this is: the marker where
// there is one, and the bare boolean where there is not.
func (t Truncation) MarshalJSON() ([]byte, error) {
	if t.marker != nil {
		return json.Marshal(*t.marker)
	}
	return json.Marshal(t.bare)
}

// TruncationAxis is §12's closed pair: the record's two axes, one of which
// every Inspection command orders on and a limit therefore cuts (ADR-0065).
//
// `identity` is §7's word for the (Target, Definition, name) triple, and it is
// preferred to `name` because the name is the last column of the key and rarely
// the cut that helps. `time` is Runs or the versions of one Record, ordered
// newest-first.
//
// It is a named string with two constants, which is the shape §12's other
// closed sets already have here (store.Outcome's triple), and the pair is
// closed where the marker is written rather than by the type: a conversion can
// spell a third name and nothing that spells one reaches the wire.
type TruncationAxis string

const (
	// AxisIdentity is Records, ordered by (Target, Definition, name).
	// --target, --definition and --name narrow it.
	AxisIdentity TruncationAxis = "identity"
	// AxisTime is Runs or versions, ordered newest-first. --since narrows
	// it, and --between does on `changes`.
	AxisTime TruncationAxis = "time"
)

// TruncationMarker is the shape §9 fixes for a result an Inspection command's
// limit cut: which axis was cut, what came back, what did not, and what would
// make the next call a narrower question. There is no cursor behind this stream
// and no way to ask for the next N — the remedy for a truncated result is a
// narrower question, and the marker is what names the parameters that ask one
// (§9, ADR-0065).
//
// All four members are written always: they are counts a reader subtracts and
// compares, and the ordinary absence rule (§7) would leave a consumer reading a
// missing key as *unknown* where the fact is *none*.
//
// Hint is the command's own words, for truncationLine's reason one package
// over: the parameters that narrow an axis differ by which command was called —
// `--between` is `changes`'s and nobody else's — and naming a flag the caller's
// command does not take would point the remedy at an argument they would go
// looking for in their own command line.
type TruncationMarker struct {
	Axis     TruncationAxis `json:"axis"`
	Returned int            `json:"returned"`
	Dropped  int            `json:"dropped"`
	Hint     string         `json:"hint"`
}

// MarshalJSON writes the marker, and refuses one that is not one: a marker
// stands only where a limit cut a result, so every one of its members has a
// value it cannot honestly take.
//
// The axis is §12's pair and nothing else — a third name would name a cut with
// no parameter behind it, which is the one thing the axis is on the wire to
// prevent. A `--limit` is a positive integer (readLimit, one package over), so
// a cut returned at least one row and dropped at least one, and a marker
// carrying either count at zero is a truncated result that reads as complete —
// the one thing §9 says this surface may never do. A hint naming nothing is the
// same failure at the remedy: §12 states the axis is what makes the hint *more
// than manners*, and an empty one hands back a narrower question with no
// parameter in it, on a surface with no cursor to page with.
//
// It refuses rather than repairing. The counts and the words are the command's,
// and nothing here can know what it cut; a marker falling back to a default
// axis would send the next call at a parameter that narrows something else.
//
// **This is a value held at write time, which the row rule above declines to do
// for a row's key order, and the difference is where the row stands.** A stream
// that stopped mid-flight to report a badly declared row would cut the wire off
// to report a smaller fault than the cut. The terminal row is the last row: the
// rows already written are already out, nothing is lost but the framing, and a
// stream ending without a terminal row is the stream's own defined signal that
// it was cut off (ResultRow above). Refusing here reports the fault in the one
// notation this surface guarantees a consumer reads, where writing the marker
// anyway would hand back a complete-looking answer naming an axis that is not
// one. The key-order rule stands as it is stated; nothing here is written
// against a row's declaration.
func (m TruncationMarker) MarshalJSON() ([]byte, error) {
	switch {
	case m.Axis != AxisIdentity && m.Axis != AxisTime:
		return nil, fmt.Errorf("truncation axis %q is neither %q nor %q", string(m.Axis), AxisIdentity, AxisTime)
	case m.Returned < 1:
		return nil, fmt.Errorf("a truncation marker returned %d rows: a limit cuts a result it returned some of", m.Returned)
	case m.Dropped < 1:
		return nil, fmt.Errorf("a truncation marker dropped %d rows: a result nothing cut carries no marker", m.Dropped)
	case m.Hint == "":
		return nil, errors.New("a truncation marker carries no hint: the axis names the cut, and the hint names what narrows it")
	}
	// The alias sheds this method and nothing else, so the members, their
	// order and their tags are the type's own and the encoding below is the
	// ordinary one.
	type members TruncationMarker
	return json.Marshal(members(m))
}

// NewResultRow is the terminal row for a stream that carried everything it
// found, or that a limit cut short on a namespace with no axis to name.
func NewResultRow(truncated bool) ResultRow {
	return ResultRow{Type: "result", Truncated: Truncation{bare: truncated}}
}

// NewTruncatedResultRow is the terminal row for a result a limit cut on one of
// the record's two axes: the marker in place of the bare true, on a command
// whose parameters can narrow what it cut.
func NewTruncatedResultRow(marker TruncationMarker) ResultRow {
	return ResultRow{Type: "result", Truncated: Truncation{marker: &marker}}
}

// Cells is empty: the terminal row is the wire's framing and has no line on the
// page. What a truncated result says to a human is a line its command writes,
// on the stream stderr carries the narration on (§9).
func (r ResultRow) Cells() []string { return nil }

// WriteJSON writes the stream as NDJSON: one compact object per line — no space
// after a separator — terminating in terminal, which is a parameter rather than
// something a caller remembers to append, a stream that opened and did not
// terminate being the one thing this surface may not do (§8). A row that will
// not encode stops the stream where it stands, and the unterminated wire is
// then the true report: the stream was cut off.
//
// Each row is MarshalRow's, which is what makes this stream and the MCP
// surface's array one reading of one row (ADR-0026): what is here is the
// framing — a newline between the lines — and nothing about what a row says.
func WriteJSON(w io.Writer, rows []Row, terminal Row) error {
	write := func(row Row) error {
		encoded, err := MarshalRow(row)
		if err != nil {
			return err
		}
		_, err = w.Write(append(encoded, '\n'))
		return err
	}
	for _, row := range rows {
		if err := write(row); err != nil {
			return err
		}
	}
	return write(terminal)
}

// MarshalRow is one row as it goes on the wire: the compact object §8's stream
// carries, with no trailing newline and no framing of any kind.
//
// It exists because the row stream is no longer the only place a row is
// serialised. The MCP surface serves §8's row set as an **array** rather than
// as a line stream, and the two must be one reading of one row: *there is one
// renderer behind both forms, so the terminal and this surface cannot drift
// apart* (§9, ADR-0026). One function is what makes that structural rather than
// a rule two callers are asked to remember: a row in an envelope and the same
// row on a `--json` stream are one encoding of one row, differing at most in
// what the transport carrying them escapes on its own account.
//
// HTML escaping is off, which is the rule that had to move with it: the wire
// carries an artefact's own bytes, and a message quoting a & or a < is a
// message a consumer reads back as it was written. It is stated here now
// because this is where the encoder is.
func MarshalRow(row Row) ([]byte, error) {
	var encoded bytes.Buffer
	enc := json.NewEncoder(&encoded)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(row); err != nil {
		return nil, err
	}
	// Encode terminates every value with a newline; a row's framing belongs
	// to whoever writes it down, an array's members carrying none.
	return bytes.TrimSuffix(encoded.Bytes(), []byte("\n")), nil
}

// WriteTable writes the stream as the human page: header, then one line per row
// that has one, aligned in columns. header is the command's own, and so is the
// order of the cells beneath it — what the renderer holds is that the page and
// the wire are written from one list of rows.
//
// A page whose rows contribute no line at all is written as nothing, header
// included: what stands in place of an empty table — a confirmation line, a
// count, a sentence naming what was looked for — is the command's own and is
// not a header over no rows.
//
// No line ends in padding. A row whose last cell is empty is ordinary — a
// Target that declares no credential is §9's own example — and the alignment it
// leaves behind is whitespace no reader sees, noise in a diff, and a trailing
// space in whatever the page is piped into. So the aligned page is written
// through a buffer and each line ends where its last written cell does.
func WriteTable(w io.Writer, header []string, rows []Row) error {
	var lines [][]string
	for _, row := range rows {
		if cells := row.Cells(); len(cells) > 0 {
			lines = append(lines, cells)
		}
	}
	if len(lines) == 0 {
		return nil
	}

	return WriteAligned(w, "", append([][]string{header}, lines...))
}

// WriteAligned writes a block of cells aligned into columns, each line prefixed
// by indent. It is what WriteTable is once the header is one more line, and it
// is exported because §8 renders two block-shaped pages that have no column
// header at all — a Comparison's two-line window, whose `BASELINE` and
// `SUBJECT` are the rows' own labels rather than a heading over them.
//
// It is one path from cells to bytes rather than one per page: what a command
// still decides is which cells, in which order, under what heading, and the
// alignment beneath every one of them is the renderer's (ADR-0026).
//
// **No line ends in padding.** A cell that is empty at the end of a line is
// ordinary — a Target that declares no credential is §9's own example — and the
// alignment it leaves behind is whitespace no reader sees, noise in a diff, and
// a trailing space in whatever the page is piped into. So the block is written
// through a buffer and each line ends where its last written cell does. The
// indent is applied after that trim, so a line with nothing on it is the indent
// and not the indent plus a column's worth of spaces.
func WriteAligned(w io.Writer, indent string, lines [][]string) error {
	var aligned bytes.Buffer
	tw := tabwriter.NewWriter(&aligned, 0, 2, 2, ' ', 0)
	for _, cells := range opened(lines) {
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	for _, line := range strings.Split(strings.TrimSuffix(aligned.String(), "\n"), "\n") {
		if _, err := fmt.Fprintln(w, indent+strings.TrimRight(line, " ")); err != nil {
			return err
		}
	}
	return nil
}

// opened is the block with every stacked cell opened out: a cell carrying a
// newline contributes one physical line per segment, and the cells beside it
// stand empty beneath their own first segment.
//
// It is here rather than in the one command that stacks a cell because it is a
// fact about the alignment: a stacked cell that reached the tabwriter whole
// would take the width of its longest segment and put the columns to its right
// on a line of their own, which is the alignment failing rather than a page
// deciding something. **The column widens and the row wraps** — §8's
// `THE CODE MOVED` renders a Cadence's expression, phrase and rate stacked in
// one cell, and a selector's form over its members, and neither is truncated
// (ADR-0059 governs `FIELDS` and reaches neither).
//
// A block with nothing stacked comes back as it went in, which is every page
// that landed before this one.
func opened(lines [][]string) [][]string {
	stacked := false
	for _, cells := range lines {
		for _, cell := range cells {
			stacked = stacked || strings.Contains(cell, "\n")
		}
	}
	if !stacked {
		return lines
	}

	var written [][]string
	for _, cells := range lines {
		segments := make([][]string, len(cells))
		deep := 1
		for i, cell := range cells {
			segments[i] = strings.Split(cell, "\n")
			deep = max(deep, len(segments[i]))
		}
		for row := range deep {
			line := make([]string, len(cells))
			for i, held := range segments {
				if row < len(held) {
					line[i] = held[row]
				}
			}
			written = append(written, line)
		}
	}
	return written
}
