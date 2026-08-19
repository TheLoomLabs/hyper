package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
)

// §12's `FLAGS` vocabulary, in the spelling the wire carries: kebab-case, the
// closed set's own, where the page renders each name upper-case for the eye
// exactly as the gutter renders the Kind `destroy` as `DESTROY` (§8, §12).
//
// Two of them spell what a marker spells and are declared here anyway. A
// marker is §8's word for what the gutter marks a line with and a flag name is
// §12's name for what indexes it; they are two closed sets that happen to agree
// on two members, and a surface reading one from the other would make a
// rendering rule out of the coincidence.
//
// The three change names — `widened`, `narrowed`, `changed` — are absent
// rather than stubbed. All three read a baseline and no range opens in this
// milestone, so there is nothing for them to be a direction between (§12).
const (
	flagDestroy    = "destroy"
	flagOpaque     = "opaque"
	flagUnbounded  = "unbounded"
	flagEnvelope   = "envelope"
	flagUnresolved = "unresolved"
)

// The block's own words: what stands above the rows, and what stands where
// there are none.
const (
	// flagsCaption says what the block is, and it says it as a relation
	// rather than as a title: this is the one editorial surface in the tool,
	// and what keeps it an index is that every row cites a line the gutter
	// already marked (§8, ADR-0026).
	flagsCaption = "FLAGS" + reviewCaptionGap + "index into the gutter above — no flag states anything the gutter does not"
	// flagsEmptyState is what an artefact drawing no flag renders in place
	// of rows. Only a Procedure is guaranteed one, so on the other four an
	// absent block would be ambiguous between *nothing to flag* and *the
	// renderer had nothing to say* — the ambiguity `AUTHORITY`'s own empty
	// state already refuses to leave standing (§8, §12).
	//
	// It is one sentence for all five, where the table above needs three:
	// there the fact differs by which end of the relation the artefact
	// supplies, and here it is the same fact on every artefact — the gutter
	// is the whole supply, and nothing it marked draws a row.
	flagsEmptyState = "no line the gutter marked draws a flag"
	// flagStepCoordinate is what stands in front of a Step's id: in the
	// coordinate column, and it is the wire member's own name: a row
	// carrying `step` renders `step <id>`, and a row carrying no coordinate
	// renders whatever its own name has in that column.
	flagStepCoordinate = "step"
	// flagLineCitation is what stands in front of the line a row cites. The
	// number is the working tree's, counted from one over every line of the
	// file including blank ones — the numbering a `gutter` row's `line`
	// shares, which is the whole of what makes the citation resolvable (§8).
	flagLineCitation = "line"
)

// reviewFlag is one row of the index before it is composed: which name, the
// line it cites, the coordinate the wire carries where the flag has one, the
// cell the page renders in its own coordinate column, the row's text, and
// whether it is a claim about the file rather than about one line.
//
// The name is the wire's kebab-case and the page upper-cases it, which is one
// fact in two notations exactly as `envelope ✓` and `"envelope ok"` are: a
// second spelling on the row would be the doubling ADR-0026 refuses.
//
// The text is the page's alone. It is not on the wire, and that is §12's own
// decomposition rather than an omission: a `flag` row carries the name, the
// line it cites and the coordinate, which is what makes a flag citing a line no
// `gutter` row marked mechanically detectable — a consumer wanting what the
// line says reads the line.
type reviewFlag struct {
	name      string
	citesLine int
	// step is the wire's coordinate, "" on every flag that has none. Only a
	// Procedure has one today (§12).
	step string
	// coordinate is the page's own cell: `step <id>` where the row carries
	// one, and otherwise whatever that flag's own name puts there — which is
	// the envelope's state, and nothing at all for the rest.
	coordinate string
	text       string
	// fileLevel is a claim about the whole artefact rather than about the
	// line it cites, which is what pins it last. `envelope` is the only one
	// today: its subject line sits where §3's key order puts it, so sorting
	// it into the middle of a per-Step list would interleave a summary with
	// the rows it summarises (ADR-0054).
	fileLevel bool
}

// procedureFlags is §12's roster on a Procedure, which is every name in the
// vocabulary: one Step's flags per Step, and the envelope's beneath them.
//
// The envelope's row is appended last and sorted last, which are two different
// rules agreeing: it is a file-level row wherever its cited line falls, and on
// a Procedure §3's key order puts `targets:` above every Step anyway
// (ADR-0054).
//
// A Procedure declaring no `targets:` draws no envelope row. There is no line
// to cite, so the gutter marks nothing there either, and a row anchored to a
// line that is not in the file would be the defect this surface is checkable
// against; the missing key is `check`'s to report (§8, ADR-0064).
func procedureFlags(marks artefact.ProcedureMarks, manifestPath func(string) string) []reviewFlag {
	var flags []reviewFlag
	for _, step := range marks.Steps {
		flags = append(flags, stepFlags(step, manifestPath)...)
	}
	if marks.EnvelopeLine > 0 {
		flags = append(flags, envelopeFlag(marks))
	}
	return flags
}

// stepFlags is one Step's rows, in the order §5 enumerates them where a Step
// draws all three: `DESTROY`, `OPAQUE`, `UNBOUNDED`. That order is a tie-break
// inside one cited line and never a sort key across lines — the block is line
// ordered, and what a name outranks is a claim no gutter carries (ADR-0054).
//
// An unresolved Step draws that one row and nothing else, exactly as its marker
// carries nothing else: there is no Kind to read, no opacity and no Bound, so
// every other name has nothing to index (§8).
func stepFlags(step artefact.StepMark, manifestPath func(string) string) []reviewFlag {
	at := func(name, text string) reviewFlag {
		return reviewFlag{
			name:       name,
			citesLine:  step.Line,
			step:       step.ID,
			coordinate: stepCoordinate(step.ID),
			text:       text,
		}
	}
	if step.Unresolved {
		return []reviewFlag{at(flagUnresolved, absentNameText(step.Absent, manifestPath))}
	}

	// The Kinds are written as §12 spells them and never as this file's own
	// flag names, which two of them collide with: what a Step declares is a
	// Kind, and `destroy` the flag is the name that indexes it (§8, §12).
	var flags []reviewFlag
	if step.Kind == "destroy" {
		flags = append(flags, at(flagDestroy, destroyStepText(step)))
	}
	if step.Opaque {
		flags = append(flags, at(flagOpaque, opaqueText(step.Operation)))
	}
	// The `opaque` `destroy` row is implied by the two above it and renders
	// regardless: *unbounded* is the accurate word for a Step where a Bound
	// is refused, and a surface silent on the strongest instance of the fact
	// it indexes is omitting rather than economising (§5, §12).
	switch {
	case step.Kind == "destroy" && step.Opaque:
		flags = append(flags, at(flagUnbounded, "an opaque destroy takes no bound"))
	case step.Kind == "mutate" && !step.Bounded:
		flags = append(flags, at(flagUnbounded, "mutate with no declared bound"))
	}
	return flags
}

// stepCoordinate is the page's coordinate cell for a Step's row: the wire
// member's own name in front of the id: the Step declares. A Step whose id: is
// not legible has no coordinate on either surface — the row still renders,
// the fact it indexes being about the line rather than about the name.
func stepCoordinate(id string) string {
	if id == "" {
		return ""
	}
	return flagStepCoordinate + " " + id
}

// destroyStepText is §8's own row: the Operation the Step invokes, and the
// Bound standing behind it where the Step declares one.
//
// A Step with no legible Bound renders the Operation alone rather than naming
// the absence. Where the absence is the fact — a `mutate` with none, an
// `opaque` `destroy` where one is refused — `unbounded` is the row that says
// so, and saying it twice on one line would be two rows for one fact (§12).
func destroyStepText(step artefact.StepMark) string {
	if step.Bound == "" {
		return step.Operation
	}
	return step.Operation + ", bound " + step.Bound
}

// opaqueText is the one sentence this name renders wherever it reads on an
// Operation, on a Step and on a Manifest alike: opacity is a Manifest fact and
// the two lines make one claim about it, so one form of words serves both
// (§12).
func opaqueText(operation string) string {
	return operation + " reaches an effect hyper cannot describe"
}

// envelopeFlag is the Procedure's file-level row: its declared envelope, and
// whether every Step is inside it.
//
// It is the one name with an all-clear form, and both forms render. A review
// does not run `check`, so an artefact carrying `envelope-exceeded` renders
// like any other and the review still exits 0 — the row is what says so, and a
// row that went silent there would leave the all-clear state indistinguishable
// from a block that had nothing to say (§9, §12).
//
// The two texts are not symmetric and the asymmetry is the mark's supply. §8
// renders the all-clear as *no step reaches a target outside* the declared
// list; the exceeded state is minted here, and it names no half, the mark
// behind it being one verdict that either the Target half or the Kind half
// sets.
func envelopeFlag(marks artefact.ProcedureMarks) reviewFlag {
	declared := "[" + strings.Join(marks.EnvelopeTargets, ", ") + "]"
	text := "a step reaches outside " + declared
	if marks.EnvelopeHolds {
		text = "no step reaches a target outside " + declared
	}
	return reviewFlag{
		name:       flagEnvelope,
		citesLine:  marks.EnvelopeLine,
		coordinate: envelopeStates[marks.EnvelopeHolds].state,
		text:       text,
		fileLevel:  true,
	}
}

// flagRow is one row of the index on the wire: the name in §12's kebab-case,
// the line it cites, and the coordinate the flag cites where it has one.
//
// `cites_line` is on every row, and that is what makes ADR-0026's rule
// checkable rather than merely stated: a flag citing a line no `gutter` row
// marked is mechanically detectable in the stream, where a per-case eyeball
// would find a breach only where somebody happened to look (§8, §12).
//
// The row carries no state and no text. `envelope` has two states and what
// says which is the `gutter` row on the same line — a flag introduces no claim
// of its own, so a state here would be the surface stating a fact twice and the
// second one could be wrong about the first (§8, ADR-0026).
type flagRow struct {
	Type      string `json:"type"`
	Flag      string `json:"flag"`
	CitesLine int    `json:"cites_line"`
	Step      string `json:"step,omitempty"`

	// coordinateText and text are the page's own notation, off the wire for
	// `gutterRow.markerText`'s reason and on the row for the same one: the
	// page is written from the rows, so what the block prints and what the
	// stream carries come out of one composition (ADR-0026).
	coordinateText, text string
}

// Cells is empty: the block is drawn by writeFlagsBlock rather than tabulated,
// having no column headings to tabulate under (ADR-0026).
func (r flagRow) Cells() []string { return nil }

// flagRows is the index as the rows both surfaces are written from: one per
// flag, in the order the block renders them.
func flagRows(flags []reviewFlag) []render.Row {
	rows := make([]render.Row, 0, len(flags))
	for _, flag := range flags {
		rows = append(rows, flagRow{
			Type:           "flag",
			Flag:           flag.name,
			CitesLine:      flag.citesLine,
			Step:           flag.step,
			coordinateText: flag.coordinate,
			text:           flag.text,
		})
	}
	return rows
}

// sortFlags puts the rows in the order both surfaces carry them: ascending by
// the line each cites, with a file-level row last (ADR-0054).
//
// It is stable, which is what keeps two flags citing one line in the order the
// roster appended them — the tie-break stepFlags states — and it sorts by
// nothing else. Severity order would be the surface deciding what matters,
// which is the one thing an index may not do: `hyper` fixes no ranking over the
// vocabulary, and a consumer wanting a subset filters the row stream.
func sortFlags(flags []reviewFlag) {
	slices.SortStableFunc(flags, func(a, b reviewFlag) int {
		if a.fileLevel != b.fileLevel {
			if a.fileLevel {
				return 1
			}
			return -1
		}
		return a.citesLine - b.citesLine
	})
}

// writeFlagsBlock writes the index beneath everything else on the screen: a
// blank line, the caption, and then the rows or this artefact's explicit empty
// state.
//
// It renders on all five artefacts and in both states, unlike the table above
// it: only a Procedure is guaranteed a row, so an absent block would be
// ambiguous between *nothing to flag* and *the renderer had nothing to say*
// (§8, §12).
//
// The rows are aligned here rather than by the renderer every table in this
// package is aligned by, for the reason the gutter is: this block has no column
// headings, and a field position no row in this rendering supplies takes no
// width at all — a blank column is the one thing this screen may not draw (§8).
func writeFlagsBlock(w io.Writer, rows []render.Row) error {
	flags := flagRowsOf(rows)

	lines := []string{"", reviewIndent + flagsCaption}
	if len(flags) == 0 {
		lines = append(lines, reviewIndent+flagsEmptyState)
	}
	widths := flagWidths(flags)
	for _, flag := range flags {
		lines = append(lines, reviewIndent+flag.line(widths))
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

// flagRowsOf is the index read back off the row list, which is the one place
// the page reads a `flag` row: the block is written from the rows, so the
// stream and the block cannot carry different flags (ADR-0026).
func flagRowsOf(rows []render.Row) []flagRow {
	var flags []flagRow
	for _, row := range rows {
		if flag, drawn := row.(flagRow); drawn {
			flags = append(flags, flag)
		}
	}
	return flags
}

// fields is one row's cells in the order the block renders them: the name
// upper-cased, the line it cites, the coordinate where it has one, and the
// row's own text.
func (r flagRow) fields() []string {
	return []string{
		strings.ToUpper(r.Flag),
		fmt.Sprintf("%s %d", flagLineCitation, r.CitesLine),
		r.coordinateText,
		r.text,
	}
}

// line is the row as the block draws it: each field padded to this rendering's
// width for that position and separated by the least gap this screen puts
// between two things on one line, ending where its last field does.
//
// A position no row in this rendering supplies is not drawn at all, which is
// the coordinate column on the four artefacts that have none: a flag's
// coordinate is a Procedure's `step` today, and a column of nothing is what §8
// refuses one column left of here.
func (r flagRow) line(widths []int) string {
	var cells []string
	for i, field := range r.fields() {
		if widths[i] == 0 {
			continue
		}
		cells = append(cells, padTo(field, widths[i]))
	}
	return strings.TrimRight(strings.Join(cells, reviewFieldGap), " ")
}

// flagWidths is each position's width in this rendering: the widest value any
// row supplies there, and 0 for a position none of them does.
func flagWidths(flags []flagRow) []int {
	var widths []int
	for _, flag := range flags {
		fields := flag.fields()
		for len(widths) < len(fields) {
			widths = append(widths, 0)
		}
		for i, field := range fields {
			if width := utf8.RuneCountInString(field); width > widths[i] {
				widths[i] = width
			}
		}
	}
	return widths
}

// definitionFlags is §12's roster on a Definition, which is one name: the
// `destroy` authority the file claims, cited on every line that claims it.
//
// The line that carries the claim is `destroy:`, and that is not the same
// statement as *the line that says the word*: §3 keeps `destroy` out of
// `kinds:` precisely so that naming an Operation is what claims it,
// granularity following severity (§8, ADR-0069). A Definition claiming
// `destroy` need write no `kinds:` at all, so a roster reading that line alone
// would leave the strongest claim on the screen unindexed.
//
// A `kinds:` line that writes it anyway draws its own row. §3 refuses that
// spelling and `check` reports it, and until it is fixed the line claims
// `destroy` and the gutter marks it `DESTROY` — a review annotates what `hyper`
// derived from these lines, and an index silent on a marked claim is omitting
// (§8, ADR-0064).
//
// A `destroy:` line naming no Operation draws no row, exactly as it draws no
// mark: there is no claim to index, and `DESTROY` alone would name a grant this
// Definition never took (§8).
func definitionFlags(marks artefact.DefinitionMarks) []reviewFlag {
	var flags []reviewFlag
	if marks.Kinds.Line > 0 && slices.Contains(marks.Kinds.Values, "destroy") {
		flags = append(flags, reviewFlag{
			name:      flagDestroy,
			citesLine: marks.Kinds.Line,
			text:      "destroy claimed",
		})
	}
	if marks.Destroy.Line > 0 && len(marks.Destroy.Values) > 0 {
		flags = append(flags, reviewFlag{
			name:      flagDestroy,
			citesLine: marks.Destroy.Line,
			text:      "destroy claimed for " + setMarker(marks.Destroy.Values),
		})
	}
	return flags
}

// targetDeclarationFlags is §12's roster on a Target declaration: the `destroy`
// its `kinds:` line accepts, and the opt-in by which it admits an `opaque`
// `destroy` at all (§4).
//
// The two are two rows on two lines and never one. A declaration accepting
// `destroy` need not admit an opaque one, and one that admits it grants nothing
// on that line — the grant is on the `kinds:` line above, which is where the
// first row cites.
func targetDeclarationFlags(marks artefact.TargetDeclarationMarks) []reviewFlag {
	var flags []reviewFlag
	if marks.Kinds.Line > 0 && slices.Contains(marks.Kinds.Values, "destroy") {
		flags = append(flags, reviewFlag{
			name:      flagDestroy,
			citesLine: marks.Kinds.Line,
			text:      "destroy accepted",
		})
	}
	if marks.OpaqueDestroy > 0 {
		flags = append(flags, reviewFlag{
			name:      flagOpaque,
			citesLine: marks.OpaqueDestroy,
			text:      "an opaque destroy admitted",
		})
	}
	return flags
}

// manifestFlags is §12's roster on a Manifest: each Operation's declared Kind
// where it is `destroy`, and its opacity where its request uses an Opaque
// Capability.
//
// Both cite the line the Operation's key is written on, which is the line the
// gutter marks and the line that binds the claim — an Operation's body being
// everything indented beneath its name (§8).
func manifestFlags(marks artefact.ManifestMarks) []reviewFlag {
	var flags []reviewFlag
	for _, op := range marks.Operations {
		if op.Kind == "destroy" {
			flags = append(flags, reviewFlag{
				name:      flagDestroy,
				citesLine: op.Line,
				text:      op.Name + " declares destroy",
			})
		}
		if op.Opaque {
			flags = append(flags, reviewFlag{
				name:      flagOpaque,
				citesLine: op.Line,
				text:      opaqueText(op.Name),
			})
		}
	}
	return flags
}

// absentNameText is the `unresolved` row's own text: which name failed, and
// where `hyper` looked for it (§12).
//
// The two halves are what separate this row from its marker. The gutter marks
// one word for four absences because it marks and does not classify; the flag
// is the surface that says which, and it says it in the words `check` reports
// the same absence in — one phrasing for one fault, read off a Step here and
// off a repository pass there (§8, ADR-0064).
//
// A key carrying no legible name at all resolves against nothing, so there is
// no path to have looked in: what failed is that nothing was written, which is
// `schema-mismatch` and `check`'s to report.
func absentNameText(absent artefact.AbsentName, manifestPath func(string) string) string {
	if absent.Name == "" {
		return absent.Key + ": no name to resolve"
	}
	return absent.Key + ": " + absent.Name + " — " + lookedIn(absent, manifestPath)
}

// lookedIn is where `hyper` looked for the name that failed. Three of the four
// resolve against a location §12 fixes and name the file that would have
// carried them, composed off the same kind-to-location table the resolution
// reads; the fourth is a key inside a Manifest that was found, so what it names
// is that Manifest — the file the reviewer opens to see which Operations there
// are.
func lookedIn(absent artefact.AbsentName, manifestPath func(string) string) string {
	switch absent.Key {
	case "definition":
		return "no " + artefactFile(artefact.KindDefinition, absent.Name)
	case "procedure":
		return "no " + artefactFile(artefact.KindProcedure, absent.Name)
	case "provider":
		// A Provider name resolves against the built-ins before any file,
		// so naming the file alone would state half the namespace it was
		// looked for in (§11, ADR-0039).
		return "no built-in Provider and no " + artefactFile(artefact.KindProvider, absent.Name)
	case "operation":
		if path := manifestPath(absent.Provider); path != "" {
			return "no such Operation in " + path
		}
		return "no such Operation on Provider " + absent.Provider
	}
	return ""
}

// artefactFile is where an artefact of that kind declaring that name is
// written: the location §12 maps the kind to, and the name as the file carries
// it.
func artefactFile(kind, name string) string {
	return locationOf(kind) + name + ".yaml"
}
