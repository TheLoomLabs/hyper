package cli

import (
	"slices"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/cadence"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// The gutter's change column and `FLAGS`' three change names (§8, §12,
// ADR-0057, issue #168).
//
// With a range open (issue #164) the review can finally say which lines moved
// since the artefact last took part in reaching the world. The column marks
// them and the three names read a direction off the lines it marked — **which
// is the gutter's supply and not a reach past it**: a review renders the
// working tree, so the value a direction is measured against is never on
// screen, and the flag's text is where it renders (ADR-0057).
//
// **Direction is claimed exactly where it is mechanically decidable, and never
// where it is not.** That is numeric comparison for a Bound and set inclusion
// wherever the fact compares as a set and one side contains the other. It is
// not available for a selector, a credential source or a Cadence, each of which
// takes `changed` and its full before-and-after text: predicate subsumption is
// undecidable in general, so a surface calling `equals: preview` →
// `starts_with: preview-` a widening would be inventing the one thing it may
// not invent (§12).

// The change column's own vocabulary: one sigil for the page, one boolean for
// the wire.
const (
	// markerChanged is what the change column draws on a line the range
	// touched. `changed` goes out as `true` rather than as this — the sigil
	// and the boolean being one fact in two notations, exactly as
	// `envelope ✓` and `"envelope ok"` are (§8).
	markerChanged = "~"
	// changeColumnGap is the change column and the space that separates it
	// from the source, which is what the source sits two characters left of
	// where a review has no range. A blank column one character wide is the
	// one thing this screen may not draw (§8).
	changeColumnGap = " "
)

// reviewChanges is what the range said about one artefact: the lines its change
// column marks, and the flags that read a direction off them.
//
// The two travel together for the reason the markers and their flags do: every
// row cites a line the gutter marked, and what makes that hold by construction
// rather than by two readings agreeing is that both come out of one comparison
// (§8, §12, ADR-0026).
type reviewChanges struct {
	touched artefact.Touched
	flags   []reviewFlag
}

// readChanges compares the artefact across the range: the working tree as the
// load read it, and the baseline as the one git object the range opens at.
//
// **A review with no range reads nothing here**, which is the column having no
// content and no width and the three names having nothing to be a direction
// across. The four absences are the header's and are ranked there; what arrives
// here is a range that opened or nothing at all.
//
// **A baseline that will not parse marks its lines and flags none of them.**
// The column is a fact about text and needs no parse; a direction is a fact
// about two values, and there is no value to read off bytes `hyper` cannot
// read. That is the same discipline the gutter already holds — a line with
// nothing derived is a line left unmarked rather than one marked empty — and
// what is wrong with the *working tree* is `check`'s to report (ADR-0064).
func readChanges(reviewed reviewedArtefact, opened reviewRange) reviewChanges {
	if opened.blob == "" {
		return reviewChanges{}
	}
	baseline := artefact.SourceLines(opened.bytes)
	changes := reviewChanges{touched: artefact.ReadTouched(baseline, reviewed.source, reviewed.root)}

	was, _, readable := yamlsubset.Parse(reviewed.path, opened.bytes)
	if !readable {
		return changes
	}
	changes.flags = changeFlags(
		artefact.ReadChangeFacts(reviewed.kind.wire, was),
		artefact.ReadChangeFacts(reviewed.kind.wire, reviewed.root),
		drawnLines{touched: changes.touched, marked: markerLines(reviewed.markers)},
		abbreviatedRevision(opened.blob),
	)
	return changes
}

// drawnLines is what a citation may land on: the lines the change column
// marked, and the lines the marker column stands beside.
//
// The two are one thing to a flag and are two supplies. **A `gutter` row is one
// rendered line, not one marked cell** — a line with content in either column
// gets a row — so what makes a citation resolvable is a row and not a `~`, and
// a flag reading only the change column would rank a line the marker column
// already drew below one nothing drew at all (§8, §12).
type drawnLines struct {
	touched artefact.Touched
	marked  map[int]bool
}

// drawn reports whether that line draws a `gutter` row at all, which is the
// relation every flag row is held to (ADR-0026).
func (g drawnLines) drawn(line int) bool {
	return line > 0 && (g.touched.Marked(line) || g.marked[line])
}

// markerLines is the lines this artefact's marker column stands beside, read
// off the same markers the rows are composed from — a marker that composed to
// nothing draws no row and is not one of them (§8).
func markerLines(markers []reviewMarker) map[int]bool {
	lines := make(map[int]bool, len(markers))
	for _, m := range markers {
		if m.wire() != "" {
			lines[m.line] = true
		}
	}
	return lines
}

// changeFlags is the index into the change column: one row per `(subject,
// fact)` pair that moved, in the order the working tree writes them.
//
// **The two sides are paired by subject and never merged.** The artefact is a
// subject on both sides always, so its own keys are compared whether either
// side wrote them — an absent key and an empty value are one value, and `–` is
// it (§8). A **Step** and a **credential slot** are subjects that may exist on
// one side only, and one that does has no before-and-after to render: every
// line of a Step the working tree gained is marked and its marker column
// carries its Kind and Target, where a Step the working tree lost has no line
// to cite at all.
//
// §12's Bounds class reads *a Step's `bound:`, its appearance and its
// disappearance included*, and that is what a Step present on both sides
// carries here — a `bound:` written where there was none, and one taken away.
// What it is not is the Step's own appearance: the class is a kind of fact
// about a subject, and a subject that is not in the working tree is not one
// this screen can cite. A Comparison reads two Runs and has a column for a side
// with nothing; a review renders one file, and the whole of what it says about
// a Step that is gone is the `~` on the line its deletion anchored to (§8).
func changeFlags(was, is []artefact.ChangeFact, drawn drawnLines, revision string) []reviewFlag {
	baseline := map[string]artefact.ChangeFact{}
	for _, fact := range was {
		baseline[factKey(fact)] = fact
	}

	var flags []reviewFlag
	for _, fact := range is {
		before, paired := baseline[factKey(fact)]
		if !paired || before.Same(fact) {
			continue
		}
		flags = append(flags, changeFlag(before, fact, drawn, revision))
	}
	return flags
}

// factKey is the pair a fact is compared under: the subject inside the artefact
// and the key the fact is written at.
func factKey(fact artefact.ChangeFact) string { return fact.Step + "\x00" + fact.Key }

// changeFlag is one moved fact as a row: which direction, the line it cites,
// the coordinate that locates it, and the before-and-after text.
//
// **The coordinate and the text name the fact exactly once between them.**
// Where the fact belongs to a Step the coordinate is that Step and the text
// opens with the key — §8's own `step retire` and `bound 3 → 5 since a91f0c2`;
// where it belongs to the artefact the key locates itself and the coordinate
// carries it, which is §8's own `cadence` standing where `step retire` stands.
//
// The revision is named in the text rather than beside every touched line: the
// column marks a line and says nothing about what it is relative to, and the
// range is one fact for the whole screen (§8).
func changeFlag(before, after artefact.ChangeFact, drawn drawnLines, revision string) reviewFlag {
	head := ""
	coordinate := after.Key
	step := ""
	if after.Step != "" {
		head, coordinate, step = after.Key+" ", stepCoordinate(after.Step), after.Step
	}
	head += sideText(before) + " → " + sideText(after) + " since " + revision

	return reviewFlag{
		name:       changeName(before, after),
		citesLine:  citedLine(before, after, drawn),
		step:       step,
		coordinate: coordinate,
		text:       head,
		from:       stackedSide(before),
		to:         stackedSide(after),
	}
}

// changeName is which of the three the row carries.
//
// A set where one side contains the other has a direction and one that both
// gains and loses a member has none — which falls out of the two inclusions
// rather than needing a clause of its own: neither holds there, and `changed`
// is what is left. `narrowed` earns its place on symmetry rather than on
// usefulness: rendering `widened` while folding every narrowing into `changed`
// would be the surface deciding that one direction is worth a name, which is a
// judgement about severity and not a fact the gutter carries (§12).
func changeName(before, after artefact.ChangeFact) string {
	switch after.Shape {
	case artefact.FactSet:
		switch {
		case containsAll(after.Members, before.Members):
			return flagWidened
		case containsAll(before.Members, after.Members):
			return flagNarrowed
		}
	case artefact.FactBound:
		return boundDirection(before, after)
	}
	return flagChanged
}

// containsAll reports whether every member of the second set is a member of the
// first. It is set inclusion over the sorted, deduplicated members a fact
// carries, which is what makes the rule quantified over the shape rather than
// listed over the classes: it reaches a set-shaped fact a list would have had
// to be edited for (§12).
func containsAll(set, members []string) bool {
	for _, member := range members {
		if !slices.Contains(set, member) {
			return false
		}
	}
	return true
}

// boundDirection is the direction a Bound moved in, and `changed` where there
// is none to read.
//
// **An absent Bound is unbounded** (§5), which is greater than every magnitude
// there is — so a Bound that appeared narrowed what the Step may reach and one
// that disappeared widened it. That is numeric comparison with the value the
// format states by omission read as the value it states, and not a second rule:
// what §12 refuses a direction to is a fact no comparison decides, and this one
// is decided.
//
// A Bound written in a shape `hyper` cannot read as a magnitude takes
// `changed`. There is nothing to compare, `bound:` is `check`'s to report, and
// a direction claimed off a value nobody could read is the invention §12
// forbids (ADR-0064).
func boundDirection(before, after artefact.ChangeFact) string {
	was, wasLegible := boundMagnitude(before)
	is, isLegible := boundMagnitude(after)
	if !wasLegible || !isLegible {
		return flagChanged
	}
	switch {
	case was == nil && is != nil:
		return flagNarrowed
	case was != nil && is == nil:
		return flagWidened
	case was != nil && is != nil && *is > *was:
		return flagWidened
	case was != nil && is != nil && *is < *was:
		return flagNarrowed
	}
	return flagChanged
}

// boundMagnitude is a Bound as a number, and nil for the Bound an absent
// `bound:` states — the unbounded one, which no number stands for. The second
// return says the Bound was legible at all, which is what tells that absence
// from a `bound:` carrying something that is not a magnitude: the first is a
// value the format states by omission and the second is nothing at all.
func boundMagnitude(fact artefact.ChangeFact) (*int, bool) {
	if !fact.Written() {
		return nil, true
	}
	magnitude, err := strconv.Atoi(strings.TrimSpace(fact.Value))
	if err != nil {
		return nil, false
	}
	return &magnitude, true
}

// citedLine is the line the row cites: **the line carrying its subject, and one
// the gutter drew a row for**.
//
// It is four readings in rank, and the rank is what keeps the citation both
// resolvable and true. The **first line the fact is written across that the
// change column marked** is §8's own worked example — the `bound:` line, and the
// conjunct's line where a selector moved one level in. Where the working tree
// carries no such line, because the key is gone, it is **the anchor that
// deletion was cited at**, which the column marked for exactly this. Beneath
// those two stand the two the marker column already drew: **a line of the fact
// the marker column stands beside**, and then **the fact's subject** — a Step's
// own `- id:`, which carries a marker on every Step there is.
//
// The last two are reachable only where a fact moved and no line it is written
// across did, which takes two lines of one artefact holding the same bytes and
// the diff pairing them across subjects. They are a rank rather than an
// assertion that it cannot happen: a citation of 0 is no citation, and a row
// dropped for want of one would be the omission ADR-0026 forbids (§8, §12).
func citedLine(before, after artefact.ChangeFact, drawn drawnLines) int {
	for _, line := range after.Lines {
		if drawn.touched.Marked(line) {
			return line
		}
	}
	for _, line := range before.Lines {
		if anchor := drawn.touched.Anchors[line]; anchor != 0 {
			return anchor
		}
	}
	for _, line := range after.Lines {
		if drawn.drawn(line) {
			return line
		}
	}
	if drawn.drawn(after.SubjectLine) || !after.Written() {
		return after.SubjectLine
	}
	return after.Lines[0]
}

// sideText is one side of the change as the row's head line carries it: the
// members of a set, the scalar as written, the Cadence's own expression, the
// form heading a selector — and `–` for a side with nothing.
//
// **A side with nothing renders `–`** wherever the format states a value by
// omission: an absent `bound:` being unbounded, an absent `over:` a Step
// invoked once, an absent `cadence:` no recurrence. Naming what an absence
// means is a claim and not a value, and the flag standing beside it is where a
// claim belongs (§8).
func sideText(fact artefact.ChangeFact) string {
	switch {
	case fact.Shape == artefact.FactSet && len(fact.Members) > 0:
		return strings.Join(fact.Members, factMemberGap)
	case fact.Shape != artefact.FactSet && fact.Value != "":
		return fact.Value
	}
	return factNothing
}

// stackedSide is one side of the change as the lines beneath the head carry it,
// and nothing at all where the head said the whole of it.
//
// Two shapes stack. A **Cadence** carries its gloss, because cron is write-only
// for humans and agents alike and a reviewer reading `0 0 1 * *` → `*/5 * * * *`
// would otherwise read the rate that matters for one side of the edit and cron
// for the other, on the screen the mandatory gloss was written for (§10,
// ADR-0005, ADR-0063) — and it carries the two facts §10 places beside a gloss
// wherever one renders, which is a rule of §10's own and not of the gloss's
// licence. A **selector** carries its members, in the Comparison's own
// notation: a ` · `-separated run of names for a `values:` selector, whose
// order is the fact, and one `field operator operand` line per conjunct for a
// predicate one. One notation across both surfaces and not two — it is the same
// fault read off a file here and off two Runs there (§8).
//
// An expression outside §10's grammar contributes no line, exactly as the
// header's own gloss does not render for one: a gloss is a reading of the
// grammar, and what is not in the grammar has no reading. The arrow beside the
// second block is what says which side a lone gloss belongs to.
func stackedSide(fact artefact.ChangeFact) []string {
	switch fact.Shape {
	case artefact.FactCadence:
		gloss, readable := cadence.Read(fact.Value)
		if !readable {
			return nil
		}
		// §10's two facts stack under the rate, one line each, on the
		// side they are a reading of. Each side carries its own pair
		// rather than the cell carrying one: the hour-boundary fact is
		// a reading of that expression's minute field, so a Cadence
		// moving onto or off the hour is a line appearing or
		// disappearing beside the arrow — which is the whole of what
		// this row exists to render. The default-branch fact is the
		// same sentence on both sides and stands on both for the rule
		// rather than for the news: it renders wherever a gloss does,
		// and a side glossed with nothing beside it would be the one
		// place a reviewer reads a Cadence and learns less about it
		// than one line above (§10).
		return append([]string{gloss.Phrase + " · " + gloss.RateText}, cadence.Facts(fact.Value)...)
	case artefact.FactSelector:
		if fact.Value == "values" {
			return []string{strings.Join(fact.Members, factMemberGap)}
		}
		return fact.Members
	}
	return nil
}

// The notation both surfaces render a value in, which is the Comparison's and
// not a second one (§8).
const (
	// factMemberGap separates two members of one set-shaped value, and the
	// members of a `values:` selector. It is wider than the gap the gutter
	// puts inside one marker because these two stand inside a sentence,
	// where a marker is a cell of its own.
	factMemberGap = " · "
	// factNothing is a side with nothing: an absent key and an empty list
	// are one value and this is it, which is what the review's `AUTHORITY`
	// table already renders for a Definition claiming no `destroy`
	// Operations (§8).
	factNothing = "–"
)
