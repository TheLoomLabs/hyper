package cli

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// The `AUTHORITY` block's own vocabulary: its caption, the token a cell with no
// supply carries, and the one a supply that names nothing does.
const (
	// authorityCaption is what stands above the columns, and it says where
	// the rows came from: this is the one rendering on the screen assembled
	// from more than one artefact, and a reviewer reading it is reading
	// something that is not in the file open beside them (§8, ADR-0026).
	authorityCaption = "AUTHORITY" + reviewCaptionGap + "assembled from definitions/ and targets/"
	// authorityNoSet is what a set-shaped cell with nothing in it carries.
	// For a set-shaped fact an absent key and an empty list are one value
	// and this is it, which is the rule §8 states for the Comparison's cells
	// citing this table's own `DESTROY OPS` as the instance. It reads on all
	// four set columns: an intersection that admits no Kind is a pairing
	// that reaches nothing, which is a fact worth a cell rather than the
	// blank this screen may not draw.
	authorityNoSet = "—"
	// authorityFieldGap is what separates two Kinds inside one cell. A cell
	// holds a set and not an enumeration a reader counts commas in, and the
	// columns beside it are already separated by two.
	authorityFieldGap = " "
)

// authorityColumns is the table's header, and it does not move between the
// three artefacts that render it: the filter changes and the table does not,
// which is what keeps one renderer behind them. A column carrying one repeated
// value stays — a Definition review already renders `DEFINITION` once per row,
// and eliding it on a Target declaration would make the same table two shapes
// (§8, ADR-0069).
var authorityColumns = []string{"DEFINITION", "TARGET", "DEFINITION KINDS", "TARGET KINDS", "EFFECTIVE", "DESTROY OPS"}

// authorityEmptyStates is the explicit empty state, one per end of the relation
// — and each says what is true of *that* artefact, which is the whole of what
// makes the line worth its place. A Definition claiming no Target and a Target
// nothing claims are two different facts, and one sentence for both would state
// the wrong one on two of the three screens.
//
// It renders where an edit somewhere in the repository would produce a row. An
// absent block would be ambiguous between *there is no pairing* and *the
// renderer had nothing to say*, and the first is a fact worth the line: a
// granted `destroy` with no claimant is either a Target awaiting its Definition
// or one whose Definition was deleted (§8, ADR-0012, ADR-0069).
//
// A Manifest and a Repository declaration have no entry and need none: no edit
// to any file produces a row there, so the block does not render at all and
// there is no sentence to write (ADR-0069).
var authorityEmptyStates = map[string]string{
	artefact.KindDefinition:        "this Definition claims no Target",
	artefact.KindTargetDeclaration: "no Definition claims this Target",
	artefact.KindProcedure:         "no Step binds a Definition to a Target",
}

// authorityRow is one pairing on the wire: §5's two claims, their intersection,
// and the `destroy` Operations the Definition names.
//
// The three Kind members carry arrays of full names and never the page's
// initials. The initials are a notation this screen renders an intersection in
// and the names are the values, exactly as `envelope ✓` and `"envelope ok"` are
// one fact in two notations (§8, ADR-0026).
//
// The four list members are pointers because empty is a value here and absent
// is a different one. A Definition that loaded and names no `destroy` Operation
// carries `destroy_operations: []` — the supply is there and names nothing —
// where one that did not load carries no such key at all, the ordinary absence
// rule a reader reads (§7). That is the distinction the whole table is built
// on, and a member merely missing would collapse the two into one reading; it
// is `artefactRow.Rate`'s reason applied to a set rather than a number.
type authorityRow struct {
	Type              string    `json:"type"`
	Definition        string    `json:"definition"`
	Target            string    `json:"target"`
	DefinitionKinds   *[]string `json:"definition_kinds,omitempty"`
	TargetKinds       *[]string `json:"target_kinds,omitempty"`
	Effective         *[]string `json:"effective,omitempty"`
	DestroyOperations *[]string `json:"destroy_operations,omitempty"`
}

// Cells is the row's line on the page, in authorityColumns' order — the page
// written from the row rather than beside it, so the two surfaces cannot state
// different things (ADR-0026). Every member the wire carries has a column: this
// row has no fact a consumer filters on that a reviewer would not also want.
func (r authorityRow) Cells() []string {
	return []string{
		r.Definition,
		r.Target,
		authorityCell(r.DefinitionKinds),
		authorityCell(r.TargetKinds),
		effectiveCell(r.Effective),
		authorityCell(r.DestroyOperations),
	}
}

// authorityCell is one set-shaped cell: §8's one word where the supply behind
// it is absent, the em dash where it is present and names nothing, and the
// members otherwise.
//
// The word is the gutter's own — a name that resolves to nothing is
// `unresolved` on both surfaces of this screen — and it covers a supply that
// resolved to nothing and one that resolved to something unreadable alike, the
// two differing in nothing this table can act on (§8, ADR-0064).
func authorityCell(members *[]string) string {
	if members == nil {
		return markerUnresolved
	}
	if len(*members) == 0 {
		return authorityNoSet
	}
	return strings.Join(*members, authorityFieldGap)
}

// effectiveCell is the same cell in the notation §8's own block renders the
// intersection in — `r`, `m d` — which is the whole of what that column says:
// the Kinds are already spelled out in the two columns it is read from, and a
// third copy of the words is width spent restating them.
//
// An initial is the Kind's first rune and not a mapping from the closed set: a
// Kind outside that set is check's to name and renders here like any other,
// where a mapping would drop it silently (§12, ADR-0064).
func effectiveCell(kinds *[]string) string {
	if kinds == nil || len(*kinds) == 0 {
		return authorityCell(kinds)
	}
	short := make([]string, 0, len(*kinds))
	for _, kind := range *kinds {
		if initial, width := utf8.DecodeRuneInString(kind); width > 0 {
			short = append(short, string(initial))
		}
	}
	return strings.Join(short, authorityFieldGap)
}

// authorityBlock is the table as this screen holds it: what the relation read,
// and how many rows could not be discovered.
//
// Neither is on a row. A rendering that emits no rows emits no rows, so there
// is nothing on the wire for the absence to be carried on; and the count is a
// fact about a discovery that failed rather than about a pairing, which is the
// one thing every row here is about (§8, ADR-0069).
type authorityBlock struct {
	table     artefact.AuthorityTable
	notLoaded int
}

// readAuthority is the relation read on the artefact under review, together
// with the count the page terminates the table with.
//
// The two namespaces it reads are the load's own folds rather than a second
// walk: every artefact is already in memory and matched byte-exact on its own
// `name:`, never on whether an `open` succeeded, so reading the relation from
// its right end costs nothing and is identical on a laptop and a runner
// (ADR-0064, ADR-0069).
func readAuthority(found resolvedArtefact, loaded repository.Loaded) authorityBlock {
	supply := artefact.Authority{
		Definitions: definitionFacts(loaded),
		Targets:     targetFacts(loaded),
	}
	block := authorityBlock{table: supply.Table(found.kind.wire, found.artefact.Root)}
	if block.table.Discovered {
		block.notLoaded = definitionsNotLoaded(loaded)
	}
	return block
}

// definitionFacts and targetFacts are the two namespaces as this table reads
// them, read off the declarations the load folded each name to. The fold is the
// load's single decision about which file a name means, so the file this table
// states a row off and the file a Step's `definition:` resolves to are the same
// one by construction rather than by two walks agreeing (issue #109).
func definitionFacts(loaded repository.Loaded) map[string]artefact.DefinitionFacts {
	facts := make(map[string]artefact.DefinitionFacts, len(loaded.DefinitionDeclarations))
	for name, root := range loaded.DefinitionDeclarations {
		facts[name] = artefact.ReadDefinitionFacts(root)
	}
	return facts
}

func targetFacts(loaded repository.Loaded) map[string]artefact.TargetFacts {
	facts := make(map[string]artefact.TargetFacts, len(loaded.TargetDeclarations))
	for name, root := range loaded.TargetDeclarations {
		facts[name] = artefact.ReadTargetFacts(root)
	}
	return facts
}

// definitionsNotLoaded is how many rows the table could not discover: the files
// in definitions/ that did not load at all.
//
// A file that loaded and declares no name is not one of them. It is in no
// namespace, so it claims no Target and no row is missing for it; what is wrong
// with it is `schema-mismatch` and check's to name (ADR-0064). What this counts
// is the file whose bytes never became a claim at all.
//
// The caller asks it only where the row set is discovered. On the other two
// filters a file that did not load leaves a row with a cell carrying
// `unresolved` — the pairing is written in the artefact under review — and a
// count there would be naming an absence the table has already marked (§8).
func definitionsNotLoaded(loaded repository.Loaded) int {
	count := 0
	for _, a := range loaded.Artefacts {
		if !a.OK && strings.HasPrefix(a.Path, locationOf(artefact.KindDefinition)) {
			count++
		}
	}
	return count
}

// authorityRows is the table as the rows both surfaces are written from: one
// per pairing the artefact under review's end of the relation supplies, already
// sorted, and none at all on the two artefacts that are members of no pair.
func authorityRows(block authorityBlock) []render.Row {
	rows := make([]render.Row, 0, len(block.table.Rows))
	for _, pairing := range block.table.Rows {
		row := authorityRow{
			Type:       "authority",
			Definition: pairing.Definition,
			Target:     pairing.Target,
		}
		// A nil list is an end with no supply and the member goes with
		// it, where a supply that names nothing writes the empty list:
		// the ordinary absence rule, and the distinction the page
		// renders as `unresolved` against an em dash (§7, §8).
		if pairing.DefinitionKinds != nil {
			row.DefinitionKinds = &pairing.DefinitionKinds
			row.DestroyOperations = &pairing.DestroyOperations
		}
		if pairing.TargetKinds != nil {
			row.TargetKinds = &pairing.TargetKinds
		}
		if pairing.Effective != nil {
			row.Effective = &pairing.Effective
		}
		rows = append(rows, row)
	}
	return rows
}

// writeAuthorityBlock writes the table beneath the artefact's own lines: a
// blank line, the caption, and then the rows or this artefact's own empty
// state, terminating in the discovery count where there is one.
//
// It writes nothing at all on the two artefacts that are members of no pair —
// no header, no empty body and no sentence — where an artefact that is a member
// of some pair and has none writes the caption and says so. The line between
// the two is whether an edit somewhere in the repository could produce a row
// (§8, ADR-0069).
//
// The table is aligned by the renderer every other page in this package is
// aligned by and then indented into the screen. That is what keeps its columns
// the two-space columns the rest of the tool draws, and it is why the block is
// written from the rows rather than beside them.
func writeAuthorityBlock(w io.Writer, kind string, rows []render.Row, block authorityBlock) error {
	if !block.table.Renders {
		return nil
	}
	lines := []string{"", reviewIndent + authorityCaption}

	tabulated, err := authorityTable(rows)
	if err != nil {
		return err
	}
	if len(tabulated) == 0 {
		tabulated = []string{authorityEmptyStates[kind]}
	}
	for _, line := range tabulated {
		lines = append(lines, reviewIndent+line)
	}
	if block.notLoaded > 0 {
		lines = append(lines, reviewIndent+definitionsNotLoadedLine(block.notLoaded))
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// authorityTable is the header and the rows aligned, as lines, and nothing at
// all where no row has a line: what stands in place of an empty table is this
// command's own and is not a header over no rows.
func authorityTable(rows []render.Row) ([]string, error) {
	var tabulated bytes.Buffer
	if err := render.WriteTable(&tabulated, authorityColumns, rows); err != nil {
		return nil, err
	}
	if tabulated.Len() == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(tabulated.String(), "\n"), "\n"), nil
}

// definitionsNotLoadedLine is the line a table with an undiscoverable row
// terminates in: how many, and where to take it.
//
// It names no member, on the shape §8 gives the `git diff` row beneath the
// Comparison. What it refuses to leave standing is the omission — the row set
// on this artefact is discovered rather than authored, so a discovery failure
// removes a row outright where every other absence on this screen leaves a
// marked one, and a table read as the whole answer would be the one table lying
// by omission (§8, ADR-0026, ADR-0069).
func definitionsNotLoadedLine(notLoaded int) string {
	plural := "s"
	if notLoaded == 1 {
		plural = ""
	}
	return fmt.Sprintf("%d definition%s did not load · hyper check", notLoaded, plural)
}
