package cli

import (
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/verify"
)

// The gutter's own vocabulary: the words and sigils the marker column is
// written in.
//
// One marker class has two notations and the rest have one. `envelope ✓` on the
// page is `envelope ok` on the wire — one fact in the two notations, exactly as
// the change column's `~` and `changed: true` are — and everything else goes
// out as the page's own string with its alignment padding collapsed (§8).
//
// `DESTROY` is not a second pair. It is upper-case on both surfaces, §8's own
// row carrying `"DESTROY staging"`: the upper case is how this vocabulary
// spells that Kind rather than a rendering of a word the wire states otherwise.
// What §12 spells in kebab-case is a `flag` row's name, which is a different
// surface's member and lands with it.
const (
	// markerUnresolved is §8's one mark for a name the gutter must follow
	// that resolves to nothing. One name and not four: the gutter marks and
	// does not classify, and which name failed is `FLAGS`' text.
	//
	// It is a Procedure's and no other artefact's, and that is the supply
	// rule holding rather than a roster left short: the other four derive
	// every mark they carry from their own lines, so there is no name for
	// the gutter to fail to follow (§8, ADR-0064, issue #122).
	markerUnresolved = "unresolved"
	// markerOpaque is the token standing between a Kind and its Target on a
	// Step invoking an Operation whose request uses an Opaque Capability,
	// and the token in front of `DESTROY` on the Target declaration opt-in
	// that admits one.
	markerOpaque = "opaque"
	// markerDestroy is how this vocabulary spells the Kind `destroy`:
	// upper-case, on the strongest fact the column carries, and upper-case
	// on both surfaces (§8). It reads wherever that Kind is marked and on
	// every artefact that marks one — a Step's, an Operation's, a Kind a
	// Definition claims or a Target declaration accepts, and the `destroy:`
	// line naming Operations.
	markerDestroy = "DESTROY"
	// markerUnbounded is what a `mutate` Step with no declared Bound carries
	// after its Kind. A `destroy` Step with none carries no `!`: that is
	// `bound-missing`, a static check, and `check`'s to report (§4, §12).
	markerUnbounded = "!"
	// markerEnvelope heads the Procedure's own envelope check, which stands
	// beside its `targets:` line and takes no part in the Kind/Target
	// alignment, being a different marker class.
	markerEnvelope = "envelope"
	// markerMemberGap is what separates two words inside one marker — two
	// Kinds a Definition claims, two hosts a Target grants, the opacity in
	// front of the Kind it admits. It is one space, where two separate one
	// *field* from the next: a marker is one derived fact in one cell, and
	// spacing what is inside it as widely as the fields would make one cell
	// read as several (§8).
	markerMemberGap = " "
)

// envelopeStates is the envelope mark's two states, each in the sigil the page
// draws and the word the wire and the `FLAGS` block state. Both render: a
// review does not run `check` (§9), so an envelope the Steps exceed renders
// like any other artefact's and the review still exits 0 — the mark is what
// says so, and a mark that went silent there would leave the all-clear state
// indistinguishable from an unmarked line.
//
// The all-clear pair is §8's own. The exceeded pair is minted here, on the
// relation §8 fixes between the two notations rather than on a rendering it
// states: §8 renders the state that holds and §12 says the name has two, so
// what the other one looks like is under-determined and this is the reading
// that keeps one sigil answering one word.
//
// The state is a member rather than a substring of the wire's marker because
// three renderings read it: the marker cell, the row it goes out on, and the
// coordinate column of the `envelope` flag that indexes the same line (§12).
var envelopeStates = map[bool]envelopeState{
	true:  {"✓", "ok"},
	false: {"✗", "exceeded"},
}

// envelopeState is one of the two, in the two notations one mark and one flag
// row render it in.
type envelopeState struct{ sigil, state string }

// page and wire are the envelope mark in the two notations the marker column
// carries: the name and its sigil for the eye, the name and its state for the
// wire.
func (s envelopeState) page() string { return markerEnvelope + markerMemberGap + s.sigil }
func (s envelopeState) wire() string { return markerEnvelope + markerMemberGap + s.state }

// reviewMarker is one line's marker cell before it is composed: the line it
// stands beside, and either the fields the marker composes from or a whole-cell
// marker that takes no part in their alignment.
//
// It is decomposed here and composed once below, which is the opposite of what
// a `gutter` row does and for the reason that row states: the row carries the
// string the page renders, because a decomposition on the wire is a second
// rendering of one fact and the second one can be wrong about the first (§8).
// The fields exist on this side of the composition because the alignment is a
// fact about the whole rendering rather than about one marker — a field is
// padded to the widest value at that position in *this* rendering — so nothing
// can be composed until every marker is known.
//
// Two artefacts compose a marker from fields and the rest carry whole cells,
// and which they are is a fact about what they mark rather than about their
// kind: a Step's Kind, its opacity and its Target are one claim rendered in one
// cell, and an Operation's Kind, its effective Repeatability and its opacity
// are the same shape one artefact along. A `hosts:` line's grant is one derived
// fact and stands in the cell entire (§8, issue #122).
type reviewMarker struct {
	line int
	// whole is a marker occupying the cell entire, in the page's notation
	// and the wire's. The two differ on the envelope mark alone and are
	// equal everywhere else; both are set, so nothing downstream has to know
	// which class it is holding.
	whole, wholeWire string
	// fields are the parts an aligned marker composes from, in the order the
	// cell renders them, each holding its position: a field this line
	// supplies nothing for is "" and is padded rather than dropped, which is
	// what keeps the field behind it a column.
	fields []string
}

// reviewMarks is one reading of the artefact under review: the cells its marker
// column carries, and the flags that index them.
//
// The two travel together because they come out of one reading and must: every
// `FLAGS` row cites a line the gutter marked, and what makes that hold by
// construction rather than by two readings agreeing is that both are derived
// from the same marks (§8, §12, ADR-0026).
type reviewMarks struct {
	markers []reviewMarker
	flags   []reviewFlag
}

// readMarks is the artefact under review read into both, each in the order its
// own surface renders it — the markers in line order, which is the order the
// rows go out in, a consumer being unable to re-sort what it has already
// printed (§8, §9); the flags in line order too, with a file-level row last
// (ADR-0054).
//
// The five rosters are five, and the dispatch is the whole of what differs
// between them: what each artefact marks is §8's and what it flags is §12's,
// and every one of them composes, aligns and goes out through the same
// functions below. That is what keeps the change column present on all five and
// Procedure-only on none of them (§8, issue #122).
//
// Only a Procedure is read against the repository. Its Kind comes from a
// Manifest two directories away and its envelope quantifies over every Step's
// `target:` at once; the other four derive every mark they carry from the file
// being read, which is why they are handed a root and nothing else.
//
// A Repository declaration reads a roster and flags none of it. Its two marks
// are the pin every Run is gated on and the retention policy bounding
// Compaction, and neither is blast radius — so the block renders its empty
// state, which is the fact rather than a roster left short (§8, §12).
func readMarks(found resolvedArtefact, loaded repository.Loaded) reviewMarks {
	root := found.artefact.Root

	var marks reviewMarks
	switch found.kind.wire {
	case artefact.KindProcedure:
		// The transitive walk needs every procedures/ file at once, which
		// is the same graph `check` builds once per run and for the same
		// reason: a nested invocation's own file, to any depth (issue #96).
		graph := verify.ProcedureGraph(loaded)
		read := artefact.ReadProcedureMarks(root, loaded.Providers, loaded.Definitions, loaded.Targets, graph)
		marks = reviewMarks{markers: procedureMarkers(read), flags: procedureFlags(read, manifestPathIn(loaded))}
	case artefact.KindDefinition:
		read := artefact.ReadDefinitionMarks(root)
		marks = reviewMarks{markers: definitionMarkers(read), flags: definitionFlags(read)}
	case artefact.KindTargetDeclaration:
		read := artefact.ReadTargetDeclarationMarks(root)
		marks = reviewMarks{markers: targetDeclarationMarkers(read), flags: targetDeclarationFlags(read)}
	case artefact.KindProvider:
		read := artefact.ReadManifestMarks(root)
		marks = reviewMarks{markers: manifestMarkers(read), flags: manifestFlags(read)}
	case artefact.KindRepositoryDeclaration:
		marks = reviewMarks{markers: repositoryDeclarationMarkers(artefact.ReadRepositoryDeclarationMarks(root))}
	}
	slices.SortStableFunc(marks.markers, func(a, b reviewMarker) int { return a.line - b.line })
	sortFlags(marks.flags)
	return marks
}

// manifestPathIn is where a Provider's Manifest loaded from, by the name a
// Definition binds it under — the load's own fold, so the file a flag names and
// the file a Step's Operation resolved against are the same one by construction
// (issue #109). It answers "" for a name the fold does not hold, which is a name
// that resolved to nothing.
func manifestPathIn(loaded repository.Loaded) func(string) string {
	return func(provider string) string { return loaded.Manifests[provider].Path }
}

// procedureMarkers is §8's roster on a Procedure: the envelope check beside the
// `targets:` line that declares it, and one marker per Step.
func procedureMarkers(marks artefact.ProcedureMarks) []reviewMarker {
	var markers []reviewMarker
	if line := marks.EnvelopeLine; line > 0 {
		state := envelopeStates[marks.EnvelopeHolds]
		markers = append(markers, reviewMarker{line: line, whole: state.page(), wholeWire: state.wire()})
	}
	for _, step := range marks.Steps {
		markers = append(markers, newStepMarker(step))
	}
	return markers
}

// newStepMarker is one Step's marker: `unresolved` where a name it must follow
// resolved to nothing, and otherwise its Kind, its opacity and its Target.
//
// A nested invocation carries no Kind and its Target field is the transitive
// envelope it reaches, the Targets joined by a space: what a Procedure's own
// line derives is what everything it invokes may touch, which is a set where a
// Step's is one name (§3, §8).
func newStepMarker(step artefact.StepMark) reviewMarker {
	if step.Unresolved {
		return wholeMarker(step.Line, markerUnresolved)
	}
	return reviewMarker{
		line:   step.Line,
		fields: []string{kindToken(step), opaqueToken(step.Opaque), setMarker(step.Targets)},
	}
}

// kindToken is a Step's Kind field: the Kind the Manifest declares, with
// `destroy` upper-cased and a `mutate` carrying no Bound marked `!`. It is ""
// on a nested invocation, which declares no Kind — the field is still padded
// there, so what a Procedure invokes lines up under the Targets its own Steps
// bind.
func kindToken(step artefact.StepMark) string {
	if step.Kind == "mutate" && !step.Bounded {
		return step.Kind + markerUnbounded
	}
	return kindMarker(step.Kind)
}

// definitionMarkers is §8's roster on a Definition: the Kinds it claims, the
// `destroy` Operations it names, and the Targets it may bind.
//
// All three are authored in the file being read, and the Provider it names
// carries no mark at all: nothing rendered here is derived from a Manifest, so
// a Definition whose `provider:` resolves to nothing renders complete and
// unmarked and the surface with something to say about it is `check`. That is
// the gutter's supply rule holding rather than the review missing something
// (§8, ADR-0064).
func definitionMarkers(marks artefact.DefinitionMarks) []reviewMarker {
	var markers []reviewMarker
	markers = appendWhole(markers, marks.Kinds.Line, kindsMarker(marks.Kinds.Values))
	markers = appendWhole(markers, marks.Destroy.Line, destroyOperationsMarker(marks.Destroy.Values))
	markers = appendWhole(markers, marks.Targets.Line, setMarker(marks.Targets.Values))
	return markers
}

// targetDeclarationMarkers is §8's roster on a Target declaration: the Kinds it
// accepts, the Capabilities and the hosts it grants, the environment variable
// each credential slot resolves from, and the opt-in that admits an `opaque`
// `destroy` (§4, §8).
//
// A declaration with no `auth:` block renders no credential-slot cell, there
// being no line to carry one — which is the absence rule this whole roster is
// read under and not an exemption: where a line is simply not in the file there
// is no cell, and that is a different thing from a line rendering a blank one.
func targetDeclarationMarkers(marks artefact.TargetDeclarationMarks) []reviewMarker {
	var markers []reviewMarker
	markers = appendWhole(markers, marks.Kinds.Line, kindsMarker(marks.Kinds.Values))
	markers = appendWhole(markers, marks.Capabilities.Line, setMarker(marks.Capabilities.Values))
	markers = appendWhole(markers, marks.Hosts.Line, setMarker(marks.Hosts.Values))
	for _, slot := range marks.Credentials {
		markers = appendWhole(markers, slot.Line, setMarker(slot.Values))
	}
	// The opt-in's mark is what it admits, spelled in the two tokens this
	// vocabulary already carries: the opacity a Step is marked with and the
	// Kind it admits (§4, §8).
	markers = appendWhole(markers, marks.OpaqueDestroy, markerOpaque+markerMemberGap+markerDestroy)
	return markers
}

// manifestMarkers is §8's roster on a Manifest: the Auth scheme it names, the
// Capabilities its Operations require, and each Operation's Kind, effective
// Repeatability and opacity beside the line its key is written on.
//
// The Operations are the one roster on this artefact that composes from fields,
// and they align exactly as a Procedure's Steps do: reading down the column is
// the Operation table, which a field appearing on some lines and not others
// would break at the one place the eye is reading (§8).
func manifestMarkers(marks artefact.ManifestMarks) []reviewMarker {
	var markers []reviewMarker
	markers = appendWhole(markers, marks.Auth.Line, setMarker(marks.Auth.Values))
	markers = appendWhole(markers, marks.Capabilities.Line, setMarker(marks.Capabilities.Values))
	for _, op := range marks.Operations {
		markers = append(markers, reviewMarker{
			line:   op.Line,
			fields: []string{kindMarker(op.Kind), op.Repeatability, opaqueToken(op.Opaque)},
		})
	}
	return markers
}

// repositoryDeclarationMarkers is §8's roster on a Repository declaration: the
// `hyper` version every Run in this repository is gated on, and the retention
// policy that bounds Compaction. A repository declaring no `retention:` renders
// no retention cell, there being no line to carry one (§8, §11).
func repositoryDeclarationMarkers(marks artefact.RepositoryDeclarationMarks) []reviewMarker {
	var markers []reviewMarker
	markers = appendWhole(markers, marks.Version.Line, setMarker(marks.Version.Values))
	markers = appendWhole(markers, marks.Retention.Line, setMarker(marks.Retention.Values))
	return markers
}

// appendWhole appends one whole-cell marker where there is a line to mark and
// something derived to mark it with, and appends nothing otherwise.
//
// The two absences it collapses are one thing to this column and are two
// different facts about the file: a key the artefact never wrote has no line,
// and a key it wrote in a shape `hyper` could not read has a line and nothing
// derived. Both leave the line unmarked, which is a line the gutter says
// nothing about rather than one it marks empty — and what is wrong with the
// second is `check`'s to report (§8, ADR-0064).
func appendWhole(markers []reviewMarker, line int, text string) []reviewMarker {
	if line == 0 || text == "" {
		return markers
	}
	return append(markers, wholeMarker(line, text))
}

// wholeMarker is a marker occupying the cell entire, in one notation on both
// surfaces — which is every whole-cell marker but the envelope's, and that one
// is composed where its two states are.
func wholeMarker(line int, text string) reviewMarker {
	return reviewMarker{line: line, whole: text, wholeWire: text}
}

// kindsMarker is a set of Kinds as one cell: each in this vocabulary's own
// spelling, in the artefact's own order. The order is the file's because a
// claim silently re-sorted is not the claim the reviewer has open beside it
// (§3, §8).
func kindsMarker(kinds []string) string {
	marked := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		marked = append(marked, kindMarker(kind))
	}
	return setMarker(marked)
}

// kindMarker is one Kind in this vocabulary's spelling: `destroy` upper-cased
// wherever it is marked, on every artefact, and every other Kind as the
// artefact declares it — including one outside §12's closed set, which states
// what it states here and earns its problem from `check` (§8, ADR-0064).
func kindMarker(kind string) string {
	if kind == "destroy" {
		return markerDestroy
	}
	return kind
}

// destroyOperationsMarker is a Definition's `destroy:` line: the Kind those
// Operations carry, and the Operations named.
//
// The Kind stands in front of the names because that is what the line means and
// not what it says — a list of Operation names is a list of Operation names
// wherever it appears, and this one is the claim that reaches the strongest
// Kind there is (§5, §8). A line naming none carries no mark: there is no claim
// to spell, and `DESTROY` alone would name a grant this Definition never took.
func destroyOperationsMarker(operations []string) string {
	if len(operations) == 0 {
		return ""
	}
	return markerDestroy + markerMemberGap + setMarker(operations)
}

// setMarker is a set of derived values as one cell: the members separated by
// the one gap this vocabulary puts between two members of one fact, and nothing
// at all where the set is empty.
func setMarker(members []string) string {
	return strings.Join(members, markerMemberGap)
}

// opaqueToken is the opacity field's own text: the token where the request uses
// an Opaque Capability, and nothing where it does not. It is a field on a Step
// and on an Operation alike — opacity is a Manifest fact, and what carries it
// is the line making the claim (§8, §12).
func opaqueToken(opaque bool) string {
	if opaque {
		return markerOpaque
	}
	return ""
}

// gutterRow is one **rendered line** of the review and not one marked cell: it
// carries the line, the marker column's rendered text where that cell has
// content, and `changed` where the change column marked the line. A line with
// content in either column gets a row and a line with neither gets none (§8).
//
// `changed` is written `true` rather than as the `~` the column draws, the
// sigil and the boolean being one fact in the two notations exactly as
// `envelope ✓` and `"envelope ok"` are. The revision it is relative to is named
// in the header and in each flag row's text, never once per touched line: the
// column marks a line and says nothing about what it is relative to, and the
// range is one fact for the whole screen (§8, issue #168).
//
// The marker goes out as the string the page renders with its alignment padding
// collapsed to single spaces, rather than decomposed into the fields it
// composed from. A decomposition is a second rendering of the same fact and the
// second one can be wrong about the first, which is exactly what the `artefact`
// row goes the other way about and says why: a marker is one derived fact in
// one cell, where the gloss is several facts with several supplies sharing a
// line (§8, ADR-0063).
type gutterRow struct {
	Type    string `json:"type"`
	Line    int    `json:"line"`
	Marker  string `json:"marker,omitempty"`
	Changed bool   `json:"changed,omitempty"`

	// markerText is the marker in the notation the page renders it in — the
	// alignment padding, and the sigils `✓` and `DESTROY` the wire spells in
	// words. It is off the wire because the wire carries the collapsed
	// string, and it is on the row because the page is written from the rows
	// (ADR-0026): both come out of one composition, so what the page prints
	// and what the row carries cannot disagree.
	markerText string
}

// Cells is empty: the gutter is drawn beside the artefact's own lines rather
// than tabulated, and writeReviewPage is what puts a marker where it belongs
// (ADR-0026).
func (r gutterRow) Cells() []string { return nil }

// gutterRows is the gutter as the rows both surfaces are written from: each
// marker composed once, against the widths this rendering fixes, carried in the
// page's notation and the wire's, and beside it whether the range touched that
// line.
//
// The widths are read off the whole rendering rather than off one marker: each
// field is padded to the widest value at that position, then separated by the
// gap this screen puts between two things on one line, which is what makes
// `read` and `mutate!` line their Targets up under each other (§8).
//
// A field position no marker in this rendering supplies takes no width at all
// and is not drawn, which is the discipline the change column already holds one
// column left: a blank column is the one thing this screen may not draw. So
// §8's own renderings, which carry no opaque Step, come out exactly as its
// two-field rule gives — and a Manifest whose Operations are all opaque draws
// the field on every one of them.
//
// A marker that composed to nothing and a line the range did not touch are one
// row between them and that row is not drawn: the line has content in neither
// column, and a `gutter` row for it would be an anchor with nothing anchored to
// it (§8).
func gutterRows(markers []reviewMarker, touched artefact.Touched) []render.Row {
	widths := markerWidths(markers)

	// One row per rendered line, which is what makes a marked line the range
	// also touched one row rather than two: the row is the line, and the two
	// columns are what it carries (§8).
	marked := map[int]string{}
	pages := map[int]string{}
	for _, m := range markers {
		if wire := m.wire(); wire != "" {
			marked[m.line], pages[m.line] = wire, m.page(widths)
		}
	}
	lines := make([]int, 0, len(marked)+len(touched.Lines))
	for line := range marked {
		lines = append(lines, line)
	}
	for line := range touched.Lines {
		if _, both := marked[line]; !both {
			lines = append(lines, line)
		}
	}
	slices.Sort(lines)

	rows := make([]render.Row, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, gutterRow{
			Type:       "gutter",
			Line:       line,
			Marker:     marked[line],
			Changed:    touched.Marked(line),
			markerText: pages[line],
		})
	}
	return rows
}

// markerWidths is this rendering's field widths, read off the markers that
// compose from fields. Whole-cell markers supply none of it — they take no part
// in the alignment, being a different marker class — though they still widen
// the column itself, which the page reads off the composed cells (§8).
func markerWidths(markers []reviewMarker) []int {
	rows := make([][]string, 0, len(markers))
	for _, m := range markers {
		rows = append(rows, m.fields)
	}
	return columnWidths(rows)
}

// page is the marker as the screen draws it: a whole-cell marker as it stands,
// and otherwise its fields aligned against this rendering's own widths, the
// way every aligned block on this screen is composed.
func (m reviewMarker) page(widths []int) string {
	if m.whole != "" {
		return m.whole
	}
	return alignedFields(m.fields, widths)
}

// wire is the same marker with its alignment padding collapsed to single
// spaces: the page's string, field for field. A whole-cell marker answers with
// its own wire notation instead, which is the envelope mark and nothing else
// (§8).
//
// What that costs is that a marker's fields are not recoverable from the string
// — a nested invocation reaching two Targets goes out `"production staging"`,
// which reads like a Kind and a Target — and §8 takes the cost outright: the
// row carries the string the page renders rather than a decomposition, because
// a decomposition is a second rendering of one fact and the second one can be
// wrong about the first. A consumer wanting the fields reads the artefact.
func (m reviewMarker) wire() string {
	if m.whole != "" {
		return m.wholeWire
	}
	stated := make([]string, 0, len(m.fields))
	for _, field := range m.fields {
		if field != "" {
			stated = append(stated, field)
		}
	}
	// The separator is one space because that is what the padding
	// collapses to, and not because it is the gap between two members of a
	// set: the page's own composition puts two between one field and the
	// next, and this is that string with the run-up taken out (§8).
	return strings.Join(stated, " ")
}
