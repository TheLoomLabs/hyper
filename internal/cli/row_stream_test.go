package cli_test

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/compare"
)

// **§9 states three rules over the whole row stream, and this file holds each of
// them over the whole corpus** (§8, §9, ADR-0026, ADR-0047, issue #172).
//
// A per-case golden says what that case wrote, and a hundred goldens that
// happen to agree say nothing about the hundred-and-first. The three rules are
// not facts about one command's rendering — they are facts about *the wire*,
// and a command that lands in a later milestone and gets one of them wrong
// would otherwise arrive with a green corpus of its own. So each is asserted by
// walking the checked-in golden files rather than by driving anything: a case
// is found by the shape of its directory, and tells these drivers nothing about
// which command it exercises or where its inputs live.
//
// Each driver **fails a third way**, beside the two directions of its own rule:
// where the corpus holds nothing for it to range over. A rule that passes over
// nothing is the failure these are written to catch — the reason to write them
// now, over a corpus of eighty-three paired cases, rather than to have them
// later over two hundred by which time something has already violated one.
//
// Two of §9's rules already stand in golden_test.go, written where the row
// stream landed: `TestGoldenCorpora_AJSONStreamIsTypedRowsEndingInItsTerminalRow`
// is the terminal row over every stream, and
// `TestGoldenCorpora_EveryFlagCitesALineTheGutterMarked` is the detectable
// violation §9 names for the page-and-wire mapping. What is here is what they
// do not reach.

// jsonStreams is every case in the corpus that opens a row stream: one whose
// argv carries `--json` and whose stdout.golden is not empty. It hands each
// one's name and decoded rows to visit, terminal row included.
//
// A case that wrote nothing to stdout opened no stream at all — a usage error,
// and the Refusals a command that is not a Run makes — so there is nothing here
// to range over and nothing missing from it (§9, ADR-0060).
func jsonStreams(t *testing.T, visit func(name string, rows []map[string]any)) {
	t.Helper()

	for _, c := range goldenCases(t) {
		if !c.opensARowStream() {
			continue
		}
		stdout := readFile(t, filepath.Join(c.dir, "stdout.golden"))
		if stdout == "" {
			continue
		}

		var rows []map[string]any
		for i, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
			var row map[string]any
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				t.Errorf("%s: line %d is not one JSON object: %v", c.name, i+1, err)
				continue
			}
			rows = append(rows, row)
		}
		visit(c.name, rows)
	}
}

// The three shapes an id, a revision or a digest takes **whole**, which is what
// every one of them goes out as (§9, ADR-0047): a git object name is forty
// lowercase hex digits, a digest is `sha256:` and sixty-four, and a Run id is a
// UUIDv7 in the five groups its text form has.
var (
	wholeRevision = regexp.MustCompile(`^[0-9a-f]{40}$`)
	wholeDigest   = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	wholeRunID    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// The same three **short of whole**. Each is the whole form's own shape cut off
// early, which is what the page draws and what this surface never does: a
// consumer resolves what it is handed against a git object or a `sha256sum`,
// and a shortened value is one it has to go somewhere else to complete (§9).
//
// `git` resolves an object name from seven digits, which is where the page's
// abbreviation starts and therefore where this begins to look; a digest and a
// Run id are recognised by their own openings and need no such floor.
var (
	shortRevision = regexp.MustCompile(`^[0-9a-f]{7,39}$`)
	shortDigest   = regexp.MustCompile(`^sha256:[0-9a-f]{0,63}$`)
	shortRunID    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}(-[0-9a-f]{4}){0,2}$`)
)

// abbreviation is the page's own mark for a value it cut short, and a value
// carrying it on the wire is abbreviated whatever else it looks like.
const abbreviation = "…"

// TestGoldenCorpora_NothingIsAbbreviatedOnTheWire is §9's first rule over every
// stream at once: **no id and no digest is abbreviated anywhere on the wire.**
//
// Every revision, every digest and every Run id goes out whole — a `provenance`
// row's `definition_revision`, `manifest_digest` and `origin_digest`, an
// `artefact` row's `baseline`, a `window` row's `procedure_revision`, a `code`
// row's `from` and `to`. The page abbreviates a fact to be *recognised*
// (ADR-0047); this surface carries none to be recognised, and a shortened value
// is one a consumer has to go somewhere else to complete.
//
// **The one stated exception is the catch-all row's `command`**, and it is
// exempted here by the name §9 gives it rather than by the case that carries
// it: it is a command a reader runs rather than an id the row reports, and
// `git` resolves it short. A row that grew an abbreviated id under some other
// key would fail here on the day it was written.
//
// It reads inside a string rather than only at its ends, because an id short of
// whole is as unusable embedded in a sentence as it is standing alone — and
// because reading only whole values is what would make the exemption above
// decorative, `git diff 1f0a3d7 88bc402` being a sentence with two of them in
// it.
func TestGoldenCorpora_NothingIsAbbreviatedOnTheWire(t *testing.T) {
	var whole int

	jsonStreams(t, func(name string, rows []map[string]any) {
		for _, row := range rows {
			rowType, _ := row["type"].(string)
			fact, _ := row["fact"].(string)
			catchAll := rowType == "code" && fact == string(compare.FactOtherLines)
			walkLeaves(row, nil, func(path []string, leaf any) {
				value, isText := leaf.(string)
				if !isText {
					return
				}
				// The one exemption §9 states, read off the row
				// it belongs to rather than off the case that
				// carries it: the catch-all's `command` is a
				// command a reader runs, not an id the row
				// reports, and `git` resolves it short.
				if catchAll && len(path) == 1 && path[0] == "command" {
					return
				}
				for _, token := range strings.Fields(value) {
					token = strings.Trim(token, `.,;:!?"'()[]{}`)
					switch {
					case wholeRevision.MatchString(token),
						wholeDigest.MatchString(token),
						wholeRunID.MatchString(token):
						whole++
					case strings.Contains(token, abbreviation),
						shortRevision.MatchString(token),
						shortDigest.MatchString(token),
						shortRunID.MatchString(token):
						t.Errorf("%s: a %s row's %s carries %q, which is short of whole; nothing is abbreviated on the wire (§9, ADR-0047)",
							name, rowType, strings.Join(path, "."), token)
					}
				}
			})
		}
	})

	// A corpus whose streams carry no id, revision or digest at all would
	// hold this vacuously. The fence is that the walk found values to
	// judge, not how many: cases are added freely, and a number here would
	// be a registration by another name.
	if whole == 0 {
		t.Fatal("no --json stdout golden in any corpus carries an id, a revision or a digest; the rule was held over nothing")
	}
}

// walkLeaves hands every scalar leaf of a decoded row to visit, along with the
// path of keys that reached it. A list contributes its members under the key
// that holds it, an index being a position rather than a name.
//
// A leaf arrives as decoded — a string, or the `float64` every JSON number
// decodes to — and what each caller does with it is its own: an abbreviation is
// a property of text alone, where a page's columns are written from numbers as
// readily as from names. A boolean is not a leaf here: `true` renders as the
// marker its column draws (`~`, `yes`, `dry-run`), which is one fact in two
// notations rather than a value carried across (§9, ADR-0026).
func walkLeaves(value any, path []string, visit func(path []string, leaf any)) {
	switch held := value.(type) {
	case map[string]any:
		for key, member := range held {
			if key == "type" {
				continue
			}
			walkLeaves(member, append(path, key), visit)
		}
	case []any:
		for _, member := range held {
			walkLeaves(member, path, visit)
		}
	case string, float64:
		visit(path, held)
	}
}

// twinnedCase is a case and its `--json` twin: one invocation in its two
// renderings, which is what makes the page and the wire comparable at all.
type twinnedCase struct{ page, wire goldenCase }

// twinnedCases is every such pair in the corpus, found by the convention the
// corpora already keep: a case named `<name>-json` beside a case named `<name>`
// whose argv is the same command line with `--json` on it. No corpus is named
// here and none could be — a pair that lands in a later milestone is found by
// having been written.
//
// A `--json` case with no plain twin is not a pair and is not an error: a
// stream whose page has no case is a case somebody has not written, which is a
// gap in coverage rather than a breach of the mapping.
func twinnedCases(t *testing.T) []twinnedCase {
	t.Helper()

	byName := map[string]goldenCase{}
	for _, c := range goldenCases(t) {
		byName[c.name] = c
	}

	var pairs []twinnedCase
	for _, wire := range goldenCases(t) {
		plain, is := strings.CutSuffix(wire.name, "-json")
		if !is {
			continue
		}
		page, found := byName[plain]
		if !found {
			continue
		}
		// The flag comes off wherever it was typed. §9 fixes no position
		// for it — `hyper provider --json shell` and `hyper provider
		// shell --json` are one invocation — so what makes a pair is the
		// command line beneath the flag and not where the flag sits.
		typed := slices.Clone(wire.argv)
		if at := slices.Index(typed, "--json"); at >= 0 {
			typed = slices.Delete(typed, at, at+1)
		}
		if !slices.Equal(typed, page.argv) {
			t.Errorf("case %s is named the --json twin of %s and is driven from %q, which is not %q with --json on it", wire.name, page.name, wire.argv, page.argv)
			continue
		}
		pairs = append(pairs, twinnedCase{page: page, wire: wire})
	}
	return pairs
}

// columnHeader matches a cell of a table's column header: §9's pages spell them
// upper-case, and nothing a row carries is rendered that way.
var columnHeader = regexp.MustCompile(`^[A-Z][A-Z_ ]*$`)

// columnGap is what separates two cells of a rendered line: the renderer pads
// to the column and no cell it writes carries two spaces of its own
// (render.WriteAligned).
var columnGap = regexp.MustCompile(`\s{2,}`)

// columns is a rendered line split at its column gaps: what the page put in
// each column, in order, with the empty ones dropped.
func columns(line string) []string {
	var cells []string
	for _, cell := range columnGap.Split(strings.TrimSpace(line), -1) {
		if cell = strings.TrimSpace(cell); cell != "" {
			cells = append(cells, cell)
		}
	}
	return cells
}

// pageCells is one rendered line as its cells, each also opened out along the
// two separators a cell stacks values with — `, ` in a list and ` · ` in a
// gloss — so that a row's own member is comparable to what the page put in the
// column beside its neighbours.
func pageCells(line string) []string {
	var cells []string
	for _, cell := range columns(line) {
		cells = append(cells, cell)
		for _, separator := range []string{" · ", ", "} {
			if !strings.Contains(cell, separator) {
				continue
			}
			for _, part := range strings.Split(cell, separator) {
				if part = strings.TrimSpace(part); part != "" {
					cells = append(cells, part)
				}
			}
		}
	}
	return cells
}

// isColumnHeader says whether a rendered line is a table's column header, which
// is where a table's rows begin.
func isColumnHeader(line string) bool {
	cells := columns(line)
	return len(cells) > 1 && !slices.ContainsFunc(cells, func(cell string) bool { return !columnHeader.MatchString(cell) })
}

// tableLines is the page's table body lines: every line beneath a column header,
// up to the blank that ends the block or to the stated line that terminates it.
//
// **A line with one cell is not a row of the table**, and that is what ends a
// block that no blank line closes: §8 renders a stated line directly beneath a
// table more than once — the Comparison's catch-all, the review's *N
// definitions did not load* — and each names no column, which is why it is
// written beneath the rows rather than as one of them (changes.writeCodeTable).
func tableLines(page string) []string {
	var lines []string
	rendered := strings.Split(strings.TrimSuffix(page, "\n"), "\n")
	for i := 0; i < len(rendered); i++ {
		if !isColumnHeader(rendered[i]) {
			continue
		}
		for i++; i < len(rendered) && len(columns(rendered[i])) > 1; i++ {
			lines = append(lines, rendered[i])
		}
	}
	return lines
}

// rendersValue says whether a rendered cell is the page's rendering of one of a
// row's values: the value itself, or the abbreviation the page draws of it.
//
// The page abbreviates a fact to be recognised (ADR-0047), so a cell that is a
// prefix of the value is its rendering where the cell carries the page's own
// ellipsis or where what stands there is an object name — the two forms §8
// draws. A bare prefix of an ordinary word is not: `read` is not the page's
// rendering of `readable`.
func rendersValue(cell, value string) bool {
	held, whole := strings.ToLower(cell), strings.ToLower(value)
	if held == whole {
		return true
	}
	cut := strings.TrimRight(held, abbreviation+"+*")
	if len(cut) < 7 || !strings.HasPrefix(whole, cut) {
		return false
	}
	return cut != held || objectName.MatchString(cut)
}

// objectName matches what a git object or a digest looks like abbreviated: the
// characters those names are drawn from and nothing else.
var objectName = regexp.MustCompile(`^[0-9a-f][0-9a-f:.-]*$`)

// TestGoldenCorpora_ThePageAndTheWireCarryTheSameRows is §9's second rule over
// every twinned case at once: **the page and the wire carry the same rows.**
//
// The mapping is total — every row of every table is one object, each header is
// one object, and nothing rendered is left out — because both forms come out of
// one list of rows (ADR-0026). What can still go wrong is what each surface
// does with that list: a page that filters before it tabulates can drop a row
// the wire carries, and a page that composes a line of its own can state a fact
// no row does. Both are held here, and in the two directions the rule has:
//
//   - **A rendered line with no row behind it.** Every table body line on the
//     page carries a value some non-terminal row of its twin carries. This is
//     the direction the spec states outright — *nothing rendered is left out* —
//     and it is asserted over the table lines because that is where §9 states
//     it: a table's rows are its rows, where the blocks and stated lines beside
//     them are each command's own rendering of what a row carries. **One column
//     is enough**, and it has to be: a page composes cells as readily as it
//     copies them — `4 → 5` out of two ordinals, `—` where a fact has no
//     subject, `days_left: 41 → 34` out of a field and its two values — and a
//     line held to every column would be a driver asserting each command's
//     composition rules. What those cells say byte for byte is the case's own
//     golden; what this adds is that the line belongs to a row at all.
//
//   - **A row with no line.** Every non-terminal row of a type the page renders
//     is rendered. The type is what makes this askable at all: the wire
//     legitimately carries rows the page has no line for — `run`'s two
//     `provenance` rows are the whole of the Provenance and the Step table
//     shows none of it — so what fails here is a row whose *own* type reaches
//     the page and which does not, which is a row the page dropped.
//
// The two directions are matched at different grains, and deliberately. A table
// line is matched **cell by cell**, the page having written those cells from
// the row's own members. A row is matched by what tells it from the rows beside
// it — a phrase of its own read as a run of words, a name of its own read as a
// column, or a word only it carries — because a page renders one row across
// several columns of one line, a `gutter` row's `read staging` being two cells,
// and across several lines of a block, which is `show`'s whole page. Matching a
// row by its cells alone would fail on every page that is not a table; matching
// a table line by its words would pass on any line sharing a word with
// anything.
//
// A pair's two cases are two invocations and not one, so a value either of them
// mints is its own: every `run` twin mints a Run id of its own, which is why
// nothing here is held on a value being equal across the pair.
//
// **Where a page renders an artefact's own source it holds this rule loosely,
// and it is the source that makes it so.** `review`'s page draws the artefact
// beside its gutter and a Refusal's caret excerpts the lines it cites, so a
// Definition's name stands on that page whether or not the row naming it was
// rendered. Nothing read off such a page can say otherwise, and inventing a
// stricter reading of it would put this driver in the business of parsing each
// command's layout — which is the registration by another name that the corpora
// are walked to avoid. What the rule is bought for is the tabulated page, where
// §9 states it: every row of every table is one object.
func TestGoldenCorpora_ThePageAndTheWireCarryTheSameRows(t *testing.T) {
	pairs := twinnedCases(t)
	var judgedLines, judgedRows int

	// Which row types the corpus shows a page rendering, read per command
	// rather than per case: the wire legitimately carries rows a page has no
	// line for — `run`'s `provenance` rows are the whole of the Provenance
	// and the Step table shows none of it — and which those are is a fact
	// about the command's page, not about one of its cases. Gathering it
	// over the command's cases is what makes a page that dropped a whole
	// table visible: one case's `step` rows vanishing is caught by every
	// other `run` case rendering theirs.
	renders := map[string]bool{}
	for _, pair := range pairs {
		page := readFile(t, filepath.Join(pair.page.dir, "stdout.golden"))
		rows := decodedRows(t, pair.wire.name, readFile(t, filepath.Join(pair.wire.dir, "stdout.golden")))
		for at, row := range rows {
			if tellingOf(rows, at).onPage(page) {
				renders[pair.page.argv[0]+" "+row.rowType] = true
			}
		}
	}

	for _, pair := range pairs {
		page := readFile(t, filepath.Join(pair.page.dir, "stdout.golden"))
		stream := readFile(t, filepath.Join(pair.wire.dir, "stdout.golden"))
		if strings.TrimSpace(stream) == "" {
			// Neither surface opened: a usage error, and the page
			// is silent beside it (§9, ADR-0060).
			continue
		}

		rows := decodedRows(t, pair.wire.name, stream)
		if len(rows) == 0 {
			// The stream is its terminal row alone: a result over
			// no rows, whose page is the command's own sentence
			// saying so.
			continue
		}

		var values []string
		for _, row := range rows {
			values = append(values, row.values...)
		}

		for _, line := range tableLines(page) {
			judgedLines++
			if !slices.ContainsFunc(pageCells(line), func(cell string) bool {
				return slices.ContainsFunc(values, func(value string) bool { return rendersValue(cell, value) })
			}) {
				t.Errorf("%s renders %q, and no row of %s carries a value it puts in a column; nothing rendered is left out of the wire (§9, ADR-0026)",
					pair.page.name, strings.TrimSpace(line), pair.wire.name)
			}
		}

		command := pair.page.argv[0]
		for at, row := range rows {
			held := tellingOf(rows, at)
			if !held.judgeable() || !renders[command+" "+row.rowType] {
				continue
			}
			judgedRows++
			if held.onPage(page) {
				continue
			}
			t.Errorf("%s carries a %s row the page renders no line for, on a command whose page renders %s rows: %s (§9, ADR-0026)",
				pair.wire.name, row.rowType, row.rowType, row.encoded)
		}
	}

	// Three ways to hold this vacuously, and each is its own failure: a
	// corpus with no twinned case at all, one whose pages tabulate nothing,
	// and one whose streams carry only their terminal rows. The fence is
	// that the walk found pairs, lines and rows to judge, not how many.
	if len(pairs) == 0 {
		t.Error("no case in any corpus has a --json twin; the mapping was held over nothing")
	}
	if judgedLines == 0 {
		t.Error("no twinned case in any corpus renders a table line; the page was held against nothing")
	}
	if judgedRows == 0 {
		t.Error("no twinned case in any corpus carries a non-terminal row; the wire was held against nothing")
	}
}

// streamRow is one non-terminal row as the mapping reads it: what it is, what
// it carries, and the words its values are drawn from.
type streamRow struct {
	rowType string
	encoded string
	values  []string
	words   []string
}

// telling is what the page can be held to for one row: what that row carries
// that a line rendering it would have to show, and that a line rendering some
// other row of the same stream would not.
//
// **A row with none of it is not judged**, and that is a statement about what a
// corpus can prove rather than a hole in the rule. A `gutter` row carrying
// `{"line":13,"marker":"staging"}` beside another carrying `staging` at line 6
// differs from it in a position and in nothing else — and the review's page
// draws no line numbers at all, the gutter standing against the source it
// annotates. Nothing read off the page could tell which of the two a `staging`
// in the margin is, so neither is judged, and what holds them is the relation
// between a flag and the gutter that golden_test.go asserts.
type telling struct {
	// phrases are the row's values of more than one word. A page renders
	// one across columns — a `gutter` row's `read staging` is two cells of
	// one line — so it is read as a run of words rather than as a cell, and
	// it needs no distinctiveness: a phrase of several words is already the
	// row's own.
	phrases [][]string
	// names are the row's values that no other row of the stream carries,
	// each a name rather than a position. A cell is held against these, so
	// the page is free to abbreviate one (ADR-0047).
	names []string
	// words are the words no other row's values are drawn from — what a
	// page says in words of its own can only have got from this row. The
	// Comparison's catch-all carries `other lines changed` and the page
	// renders *other lines could not be counted*: one fact in the wire's
	// vocabulary and in the page's, sharing no value and two words.
	words map[string]bool
}

// position matches a value that is a place in an order rather than a name: an
// ordinal, a Step number, the line a flag cites. Two rows of a stream differ in
// one constantly, and a page that renders the thing they number renders the
// number itself far more rarely — so it tells this row from that one on the
// wire and cannot be looked for on the page.
var position = regexp.MustCompile(`^-?[0-9]+$`)

// tellingOf reads what the page can be held to for the row at one position,
// against the rest of its stream.
func tellingOf(rows []streamRow, at int) telling {
	elsewhere, spelled := map[string]bool{}, map[string]bool{}
	for i, row := range rows {
		if i == at {
			continue
		}
		for _, value := range row.values {
			elsewhere[value] = true
		}
		for _, word := range row.words {
			spelled[word] = true
		}
	}

	held := telling{words: map[string]bool{}}
	for _, value := range rows[at].values {
		if words := streamWords(value); len(words) > 1 {
			held.phrases = append(held.phrases, words)
		} else if !elsewhere[value] && !position.MatchString(value) {
			held.names = append(held.names, value)
		}
	}
	for _, word := range rows[at].words {
		if !spelled[word] && !position.MatchString(word) {
			held.words[word] = true
		}
	}
	return held
}

// judgeable says whether the page can be held to this row at all.
func (h telling) judgeable() bool {
	return len(h.phrases) > 0 || len(h.names) > 0 || len(h.words) > 0
}

// onPage says whether the page renders this row, on one of its lines: a phrase
// of the row's spelled out in a run, a name of the row's standing in a column,
// or a word the page could only have got from this row.
func (h telling) onPage(page string) bool {
	for _, line := range strings.Split(page, "\n") {
		rendered := streamWords(line)
		if slices.ContainsFunc(h.phrases, func(phrase []string) bool { return carries(rendered, phrase) }) {
			return true
		}
		if slices.ContainsFunc(rendered, func(word string) bool { return h.words[word] }) {
			return true
		}
		cells := pageCells(line)
		if slices.ContainsFunc(h.names, func(name string) bool {
			return slices.ContainsFunc(cells, func(cell string) bool { return rendersValue(cell, name) })
		}) {
			return true
		}
	}
	return false
}

// carries says whether a rendered line spells a value: its words, in order and
// together, wherever the alignment put its columns.
func carries(rendered, value []string) bool {
	if len(value) == 0 {
		return false
	}
	for at := 0; at+len(value) <= len(rendered); at++ {
		if slices.Equal(rendered[at:at+len(value)], value) {
			return true
		}
	}
	return false
}

// decodedRows is a stream's non-terminal rows, decoded. The terminal row is
// dropped: it is the wire's framing and no page has a line for it — §8's
// terminal line is what the page writes in its place, and it is the command's
// own (render.ResultRow).
func decodedRows(t *testing.T, name, stream string) []streamRow {
	t.Helper()

	lines := strings.Split(strings.TrimSuffix(stream, "\n"), "\n")
	var rows []streamRow
	for i, line := range lines[:len(lines)-1] {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("%s: line %d is not one JSON object: %v", name, i+1, err)
		}
		rowType, _ := decoded["type"].(string)
		row := streamRow{rowType: rowType, encoded: line}
		walkLeaves(decoded, nil, func(_ []string, leaf any) {
			value, isText := leaf.(string)
			if !isText {
				// A number is a value the page renders from
				// too — a `probe` row's projection puts its
				// statuses and its counts in the `VALUE`
				// column — and it reaches the page as the text
				// its own encoding writes.
				value = strconv.FormatFloat(leaf.(float64), 'f', -1, 64)
			}
			row.values = append(row.values, value)
			row.words = append(row.words, streamWords(value)...)
		})
		rows = append(rows, row)
	}
	return rows
}

// streamWords is the words a rendering is drawn from: what remains after the
// separators a page lays out with — whitespace, a list's comma, a gloss's
// interpunct, the review's gutter rule — with the one-character tokens dropped,
// a bare ordinal matching anything that happens to count.
func streamWords(text string) []string {
	var words []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return unicode.IsSpace(r) || r == ',' || r == '·' || r == '|' || r == '│'
	}) {
		if len([]rune(word)) > 1 {
			words = append(words, strings.ToLower(word))
		}
	}
	return words
}

// TestGoldenCorpora_ARunTerminatesInAnOutcomeRowTheTwoDeclinesIncluded is §9's
// third rule where it is hardest to hold: **`run` is on the `outcome` side on
// every path on which a Run was attempted, the two that decline before a Run is
// identified included — what is missing there is the row's `run_id` and never
// the row** (§9, §10).
//
// A terminal type that flipped according to how early the tool declined would
// be one fact arriving under two contracts, and a Run that declined and wrote
// nothing at all would be indistinguishable from a Run whose stream was cut
// off — which is the one thing the terminal row exists to tell apart.
//
// The two declines are the version pin gate and the bootstrap `store-absent`,
// neither of which has an entry to name, and a corpus with no case for either
// fails below: without one the rule is asserted only where an id was there to
// write, which is the half that was never in question.
//
// **Two paths are outside it, and the driver exempts each by what it is.** A
// usage error opens no stream at all and is not a path the command takes (the
// driver below). A lock another Run holds and a Store this one could not reach
// are paths on which no Run was **attempted** — both stand before `run.json` —
// so each is silent on stdout and `run_id` stays absent exactly where §9 says
// it is. Neither of those can be a golden today, a live lock and git's account
// of an unreachable remote being what run_store_lost_test.go exists for; the
// exemption is stated all the same, because a driver and the code it reads
// stating opposite rules is a trap for whoever writes the case that could.
func TestGoldenCorpora_ARunTerminatesInAnOutcomeRowTheTwoDeclinesIncluded(t *testing.T) {
	var terminated, unidentified int

	for _, c := range goldenCases(t) {
		if c.subject() != "run" || !c.opensARowStream() {
			continue
		}
		exit, err := strconv.Atoi(strings.TrimSpace(readFile(t, filepath.Join(c.dir, "exit.golden"))))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		stdout := readFile(t, filepath.Join(c.dir, "stdout.golden"))
		switch {
		case exit == cli.ExitUsage:
			// No Run was attempted and no stream opened; the
			// driver below is where that is held.
			continue
		case stdout == "" && exit == cli.ExitStoreLost:
			// A Run that lost the Store *before it began* — to
			// the lock, or to the sync at Run start — and neither
			// was a Run attempted on nor a decline, so it answers
			// nothing. A 75 that ran and could not push its work
			// is the other shape and does answer, which is why
			// this exempts the silence rather than the exit.
			continue
		case stdout == "":
			t.Errorf("%s exits %d and opens no stream; a Run answers on stdout at every exit on which a Run was attempted (§9, §10)", c.name, exit)
			continue
		}

		lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		var row struct {
			Type  string `json:"type"`
			RunID string `json:"run_id"`
		}
		if err := json.Unmarshal([]byte(lines[len(lines)-1]), &row); err != nil {
			t.Errorf("%s: the last row is not one JSON object: %v", c.name, err)
			continue
		}
		if row.Type != "outcome" {
			t.Errorf("%s ends in a %q row; a Run's stream terminates in outcome, whatever declined it (§9)", c.name, row.Type)
			continue
		}
		terminated++
		if row.RunID == "" {
			unidentified++
		}
	}

	if terminated == 0 {
		t.Error("no run case in any corpus opens a --json stream; the rule was held over nothing")
	}
	// The shape the rule is bought for: a Run that declined before it had
	// an id, whose outcome row is emitted regardless and carries no
	// `run_id`. Without one in the corpus the paragraph above is asserted
	// only where an id was there to write.
	if unidentified == 0 {
		t.Error("no run case in any corpus declined before a Run was identified; the row whose run_id is absent is driven by nothing")
	}
}

// TestGoldenCorpora_AUsageErrorOpensNoStream is the other half of the terminal
// row's rule, and the reason its absence is unambiguous: **a usage error opens
// no stream at all**, so there is no terminal row to be absent from and nothing
// was cut off (§9, §10, ADR-0060).
//
// stdout carries nothing in either mode and the rendering goes to stderr like
// every other human rendering of an error. A usage error is not a path a
// command takes — it is the command never starting — and the exit code is what
// says so.
func TestGoldenCorpora_AUsageErrorOpensNoStream(t *testing.T) {
	var judged int

	forEachGoldenTriple(t, func(dir string, exit int) {
		if exit != cli.ExitUsage {
			return
		}
		judged++
		if stdout := readFile(t, filepath.Join(dir, "stdout.golden")); stdout != "" {
			t.Errorf("%s exits %d and wrote %q to stdout; a usage error opens no row stream at all (§9, ADR-0060)",
				dir, cli.ExitUsage, stdout)
		}
	})

	// A corpus holding no usage error would hold this vacuously. The fence
	// is that the walk found the exits it is here to judge, not a count of
	// them: cases are added freely, and a number here would be a
	// registration by another name.
	if judged == 0 {
		t.Fatal("no case in any corpus exits on a usage error; the invariant held vacuously")
	}
}
