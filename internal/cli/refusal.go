package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/pin"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/verify"
)

// §8's Refusal rendering: the caret excerpt, the `=` notes and the `EDIT ONE
// OF` table (§8, issue #169).
//
// **A Refusal is the most verbose surface in the tool** (ADR-0026), and this
// file is why. No flag, no confirmation and no override reaches one (ADR-0001),
// so what is rendered here is the entire path back to a passing review — not a
// report of a fault but the instructions for clearing it, and a rendering that
// left something out would leave an operator with a `77` and nowhere to go.
//
// It replaces the problem table `check` renders, which stood on three pages
// until this milestone: a Run's, a `hyper show` of a refused entry's, and a
// Probe's. Every fact §8 requires was already on those pages — file, line,
// field, code, message — and what lands here is the shape. The three call it
// rather than each carrying a copy, which is the rule that keeps them from
// coming to render one Refusal three ways (gate.go, run.go, show.go).
//
// **What is here is a rendering and never a decision.** Nothing in this file
// decides that a Run refused, which check declined it, or what an edit would
// be; those are the engine's and arrive already made. What is decided here is
// which lines of a file to show, where the caret sits, and which of §8's notes
// each member earns.

// refusalIndent is the page indent of a caret excerpt's file line, and
// refusalGutter the width the line numbers are right-aligned into. Together
// they put the `│` in one column whether the file has eight lines or eight
// hundred — the excerpt reads as one block rather than as a ladder that shifts
// when a number grows a digit (§8).
const (
	refusalIndent = "  "
	refusalGutter = 5
)

// noBypassNote is §8's note beneath every excerpt whose remedy is an artefact
// edit, and it is the sentence ADR-0001 is: there is no flag, no confirmation
// and no override, so a reader looking for one is told so here rather than
// finding out by trying three.
const noBypassNote = "no flag overrides this (ADR-0001) — the way past is an artefact edit"

// The three phases a Refusal can be found in, as §8's `=` note words them. Which
// one a member carries is what tells a reader whether this Refusal preceded
// execution or halted it, and it is the most load-bearing note on the page for
// a reader deciding whether anything reached the world (§7, §8). The third
// arrived with the Requirement, which decides after the Step it reads has run
// and before any Step after it has — a moment neither of the other two words
// (§6, issue #236, ADR-0116).
const (
	phaseAtExpansion   = "checked at expansion, before the first call"
	phaseAtRunStart    = "checked at run start, before the first step"
	phaseAtRequirement = "checked at a requirement, after the step it reads and before any step after it"
)

// refusalPhase is which of the three this member carries, and it is **derived
// per member** rather than once for the page: §7 fixes the array as one phase's
// finding, and a rule that reads each member holds whether or not that stays
// true.
//
// **The Step it cites is what decides.** Five checks decide at or before a
// Step's Expansion and they are the only members of the closed set that cite a
// Step reached at all; every other check declines between `run.json` and Step 1
// (§6, §7, ADR-0061, internal/run/citation.go). So *does this member name a
// Step* is the same question as *was this found at an Expansion* — and it is
// answered off the member rather than off a map of codes, which would be a
// second reading that can disagree with the citation beside it, and which one
// code would already break: `schema-mismatch` is §4's static check and also
// §6's, where an `args:` value arriving from a reference will not read as its
// input's type.
//
// **`secret-sink-absent` is the exception and is the only one.** It is the
// **invocation's** gate, stated at Run start because both its operands are
// already in hand, and it names every reachable Step whose Operation declares
// secret output — as an artefact coordinate, before Step 1, writing no Step
// file (§6, §9, internal/run/gates.go). It is the one code that cites a Step it
// did not reach.
//
// **A Requirement is the third phase**, and it is told apart by the one shape
// only a Requirement produces: an `id:` and no position. A Requirement takes no
// position in the sequence — it writes no Step file and reaches no Disposition
// (§6, ADR-0116) — so it names what an edit would reach and nothing a Journal
// entry counts, and the note has to say what neither of the other two says: the
// Step it read has already run, and no Step after it has (§7, §8, ADR-0072).
func refusalPhase(member refusalRow) string {
	switch {
	case member.Step == nil && member.StepID != "":
		return phaseAtRequirement
	case member.Step != nil && member.ErrorCode != run.CodeSecretSinkAbsent:
		return phaseAtExpansion
	default:
		return phaseAtRunStart
	}
}

// refusalRemedies is the set §8 leaves behind: the codes whose way past is
// **not** an artefact edit, each mapped to the remedy that is.
//
// Membership is the discriminator for the remediation table as well as the
// source of the note — a code in this map renders no `EDIT ONE OF` and a code
// outside it renders one — so the two can never disagree about which set a code
// is in (§8, ADR-0061).
//
// **Each names the remedying command verbatim.** A check knows its own remedy,
// so naming it states a fact rather than editorialising: `FLAGS` is the one
// surface ADR-0026 restricts and it is restricted for summarising other lines,
// which this does not do. Prose that describes a command without naming it is
// the same fact rendered worse.
//
// The four classes §8 states are all here. A **command** — `hyper project`,
// `hyper store init`. A **different binary**, which is the one class with no
// command to name, its remedy being an install rather than an invocation. An
// **act on the environment**, whose note names the wrappers an operator
// actually reaches for rather than telling them to export something. And a
// **different invocation**, which is the only remedy on the list the Run's own
// operator can take without leaving the shell — and the only one no generated
// workflow can take at all (§4, ADR-0077).
var refusalRemedies = map[string]string{
	run.CodeCredentialAbsent:               "set it in the environment, or wrap the invocation — op run --, direnv, aws-vault exec --",
	run.CodeSecretSinkAbsent:               "the same invocation again with --secret-out <path>",
	verify.CodeProjectionStale:             "hyper project",
	storeAbsentCode:                        "hyper store init",
	store.SchemaUnsupportedCode:            "a hyper that reads this schema version — nothing in the repository is the fault",
	artefact.CodeManifestSchemaUnsupported: "a hyper that reads this schema version — nothing in the repository is the fault",
	pin.CodeMismatch:                       "the hyper the Repository declaration pins",
}

// **Two of the seven are not reachable through this renderer today**, and they
// are in the map anyway. `store-absent` and `version-pin-mismatch` decline
// before a Run is identified at all and render as the two-line form gate.go
// states. The map is §8's own list of *the remedies that are not an edit*, and
// a list that dropped a member because the check that fires it is not built is
// a partial copy of a closed set.
//
// `projection-stale` was the fourth until issue #179 built the check, and
// `manifest-schema-unsupported` the third until issue #190 built this one.
// Their arrival is what the ground above was held for: a Run's pre-flight
// declines on each like any other static code, and what a reader gets is a
// command to run, or a binary to install, rather than a table pointing at a
// generated file or at a Manifest whose shape this binary does not know (§10,
// §11).
//
// The Manifest's code is read from the check that decides it
// (artefact.CodeManifestSchemaUnsupported) rather than spelled again here, on
// the rule every member of the closed set now keeps: one declaration site. It
// is **not** internal/artefact's `schema-unsupported`, which the tool also
// emits — that one is an input schema reaching outside §4's four-keyword subset
// and its remedy is an ordinary artefact edit, which is why it earns no entry
// in this map at all. The two read alike and land on opposite sides of the one
// question this map answers.

// relativeOperand is `older_than:` or `newer_than:` written against a duration,
// found in the text an excerpt renders.
//
// §8 puts the gloss on the surfaces **that render the operand**, which is why
// this reads the excerpt rather than the check: the `=` note beneath a caret
// reads the lines above it where every other note reads the check that declined
// (ADR-0034). A Refusal citing a `bound:` renders the `over:` block above it in
// context, and the operand a reader sees there is the one that gets glossed.
//
// **It finds the operand and does not judge it.** The duration grammar is
// internal/schema's — the one place §3's `[0-9]+[smhd]` is spelled, and the
// place the loader holds an authored value to — so what this matches is *a
// token in the operand position* and what says whether that token is a duration
// is schema.DurationSeconds. A second spelling of the grammar here would be a
// second reading of one closed set, and the day a unit is added to it this would
// silently decline to gloss the operands written with it.
var relativeOperand = regexp.MustCompile(`\b(older_than|newer_than):\s+"?([^"\s]+)"?\s*$`)

// caretExcerpt is the offending line in its own context, read from the working
// tree.
//
// **The line numbers are the working tree's**, which is the same supply the
// review's gutter and its `EDIT ONE OF` rows share: a reader is being told
// where to edit, and the only numbering an edit can be made against is the file
// on disk in front of them (§8, ADR-0057).
type caretExcerpt struct {
	// first is the line number the excerpt opens at and lines its rendered
	// lines, the cited one last.
	first int
	lines []string
	// column is the 0-indexed offset on the cited line the caret points at.
	column int
}

// rendered says this member has an excerpt to draw. An excerpt with no lines is
// a file that could not be read or a code that renders no caret at all, and
// both fall back to the same shape: the coordinate as a note.
func (c caretExcerpt) rendered() bool { return len(c.lines) > 0 }

// readExcerpt is the caret excerpt for one citation, and false where none can
// be drawn.
//
// **Two codes render no caret and both checks are here** rather than falling
// out of what the working tree happens to hold. Each cites a file that is not an
// artefact to edit, which is the one thing this surface must not point a reader
// at: `store-schema-unsupported` cites evidence, and editing it is editing
// evidence (ADR-0011); `projection-stale` cites a generated file, whose every
// byte is derived from artefacts elsewhere and whose hand-edits do not survive
// the next `project` (§10). Neither would draw one today — a Store file is
// absent from the working tree and a projection problem cites no line — and a
// rule that holds by accident is one no reading can be checked against, so both
// are stated (§7, §8, ADR-0028).
//
// A file that cannot be read draws nothing either, and says so by drawing
// nothing: the coordinate then renders as a note, which is the same shape the
// code above takes. A Refusal whose artefact was deleted between the Run and the
// rendering is rare and is not an error — what the page owes a reader there is
// the file and the line, which it still has.
func readExcerpt(root, code, file string, line int) caretExcerpt {
	if code == store.SchemaUnsupportedCode || code == verify.CodeProjectionStale || file == "" || line < 1 {
		return caretExcerpt{}
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return caretExcerpt{}
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if line > len(lines) {
		return caretExcerpt{}
	}

	cited := lines[line-1]
	first := line
	// The context above, back to the line that encloses the cited one and
	// no further than two. What the excerpt is for is judging whether
	// raising the Bound is the right fix at all or whether the selector is
	// the thing that is wrong (§8), and that judgement is made against the
	// block the line sits in — so the walk stops at the first line shallower
	// than the citation, that being where the block began.
	for taken := 0; taken < 2 && first > 1; taken++ {
		first--
		if indentOf(lines[first-1]) < indentOf(cited) {
			break
		}
	}

	return caretExcerpt{first: first, lines: lines[first-1 : line], column: caretColumn(cited)}
}

// indentOf is how far a line is indented, in characters. YAML admits no tab in
// indentation, so a character is a column.
func indentOf(text string) int { return len(text) - len(strings.TrimLeft(text, " ")) }

// caretColumn is where the caret sits on the cited line: **at the value**, and
// at the line's first character where the line holds no value.
//
// The value is what a Refusal is about on every line it ever cites — a `bound:`
// that is too small, a `token:` whose variable is unset, an `operation:` naming
// nothing — so a caret under the key would point at the one half of the line
// that is not the fault.
func caretColumn(text string) int {
	if at := strings.Index(text, ": "); at >= 0 {
		return at + 2
	}
	return indentOf(text)
}

// glosses is the relative operands the excerpt's text holds, mapped to the
// instants they resolved against, in the order the lines render.
//
// It is empty where the excerpt renders no such operand and where there is no
// instant to resolve against — a surface with no Run has none, and a gloss is
// derived arithmetic rather than a claim, so it renders where its supply is and
// nowhere else (ADR-0034, ADR-0063).
func (c caretExcerpt) glosses(instant time.Time) []gloss {
	if instant.IsZero() {
		return nil
	}
	var found []gloss
	for _, line := range c.lines {
		match := relativeOperand.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		// An operand that is not a duration names an instant of its
		// own — `older_than: 2026-01-01T00:00:00Z` — and there is
		// nothing to resolve: the artefact already wrote the instant,
		// and a note restating it would gloss a value with itself.
		resolved := resolvedInstant(match[2], instant)
		if resolved == "" {
			continue
		}
		found = append(found, gloss{operator: match[1], operand: match[2], instant: resolved})
	}
	return found
}

// selectorGlosses is the relative operands one **rendered selector** holds, in
// the order the line renders them.
//
// It is a second scanner beside the excerpt's because the two read two
// notations of one thing: an excerpt is the artefact's own bytes, `older_than:
// 14d`, and a rendered selector is §8's notation over the entry's canonical
// value, `created_at older_than 14d` with the colons dropped (selectorText).
// A regexp that admitted both would admit neither precisely.
//
// It reads the text the page renders rather than the value beneath it for the
// reason the excerpt's does: §8 puts the gloss on the surfaces **that render
// the operand**, so what is glossed is what a reader can see (ADR-0034).
func selectorGlosses(rendered string, instant time.Time) []gloss {
	if instant.IsZero() {
		return nil
	}
	var found []gloss
	words := strings.Fields(rendered)
	for i, word := range words {
		if word != "older_than" && word != "newer_than" {
			continue
		}
		if i+1 >= len(words) {
			continue
		}
		// An operand that is not a duration names an instant of its
		// own, and there is nothing to resolve: the artefact already
		// wrote the instant.
		if resolved := resolvedInstant(words[i+1], instant); resolved != "" {
			found = append(found, gloss{operator: word, operand: words[i+1], instant: resolved})
		}
	}
	return found
}

// gloss is one relative operand and the instant it resolved to.
type gloss struct {
	operator, operand, instant string
}

// note is the `=` note this gloss renders: the operand as the artefact wrote
// it, and what it resolved to.
func (g gloss) note() string {
	return fmt.Sprintf("%s: %s resolved to %s", g.operator, g.operand, g.instant)
}

// resolvedInstant is the instant a duration operand names, counted back from
// the Run's start, in the notation §3 mandates for an authored one — and "" for
// an operand that is not a duration, which no caller reaches.
//
// It renders to the second. A gloss is a fact for an eye rather than a value
// anything compares, and the sub-second precision the Store writes an instant
// at would be three digits nobody reads on a line whose subject is a fortnight.
func resolvedInstant(operand string, started time.Time) string {
	seconds, isDuration := schema.DurationSeconds(operand)
	if !isDuration {
		return ""
	}
	return started.Add(-time.Duration(seconds) * time.Second).UTC().Format(time.RFC3339)
}

// excerpted is the member with what the page derived about it filled in: the
// caret excerpt read from the working tree, and the glosses that excerpt earned
// against the Run's start.
//
// It is one reading at one moment rather than a page and a wire each reading
// the file for themselves, which is ADR-0026 at the one place in this rendering
// where a fact comes from outside the Run: the `=` note beneath the caret and
// the row's `resolved` are the same operands in two notations, and a second
// read of a file somebody is editing is where the two come to disagree.
//
// started is the Run's start, and the zero instant where the surface has no Run
// — a Probe. A gloss is derived arithmetic rather than a claim, so it renders
// where its supply is and nowhere else (ADR-0034, ADR-0063).
func excerpted(member refusalRow, root string, started time.Time) refusalRow {
	member.excerpt = readExcerpt(root, member.ErrorCode, member.File, member.Line)
	member.resolvedOperands = member.excerpt.glosses(started)
	if len(member.resolvedOperands) == 0 {
		return member
	}
	member.Resolved = map[string]string{}
	for _, held := range member.resolvedOperands {
		member.Resolved[held.operand] = held.instant
	}
	return member
}

// writeExcerpt writes one member's caret excerpt and its `=` notes.
//
// The two are one block because a note with no excerpt above it is a sentence
// about a line the reader cannot see: where no caret is drawn the coordinate
// itself becomes the first note, so the block still says where and still says
// what.
func writeExcerpt(w io.Writer, member refusalRow, phase string) error {
	notes := member.notes(phase)
	if !member.excerpt.rendered() {
		return writeNotes(w, refusalIndent, notes)
	}

	excerpt := member.excerpt
	width := max(refusalGutter, len(strconv.Itoa(excerpt.first+len(excerpt.lines)-1)))
	gutter := strings.Repeat(" ", width)

	if _, err := fmt.Fprintf(w, "%s%s:%d\n", refusalIndent, member.File, member.Line); err != nil {
		return err
	}
	for i, text := range excerpt.lines {
		if _, err := fmt.Fprintf(w, "%*d │ %s\n", width, excerpt.first+i, text); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%s │ %s^ %s\n", gutter, strings.Repeat(" ", excerpt.column), member.Message); err != nil {
		return err
	}
	// The empty gutter line between the caret and the notes. It is what
	// keeps the notes legible as commentary on the excerpt rather than as
	// more of it, on a block that is already the longest thing on the page.
	if _, err := fmt.Fprintf(w, "%s │\n", gutter); err != nil {
		return err
	}
	return writeNotes(w, gutter+" ", notes)
}

// writeNotes writes the `=` notes at one indent.
func writeNotes(w io.Writer, indent string, notes []string) error {
	for _, note := range notes {
		if _, err := fmt.Fprintf(w, "%s= %s\n", indent, note); err != nil {
			return err
		}
	}
	return nil
}

// notes is the `=` notes this member earns, in §8's order.
//
// **A relative predicate's resolved instant comes first**, because it reads the
// excerpt above it where the others read the check (ADR-0034). The phase
// follows, and the last note is either ADR-0001's — there is no way past but an
// edit — or the remedy for the codes where an edit is not the way past. The two
// are exclusive by construction: a code in refusalRemedies renders no `EDIT ONE
// OF`, so the note that points at one would be pointing at nothing.
//
// Where no caret was drawn the coordinate leads: the message first, since it
// has no caret to sit beside, then the file and the field §8 says go here
// instead.
func (r refusalRow) notes(phase string) []string {
	var notes []string
	if !r.excerpt.rendered() {
		notes = append(notes, r.Message)
		if r.File != "" {
			notes = append(notes, "file: "+r.File)
		}
		if r.Field != "" {
			notes = append(notes, "field: "+r.Field)
		}
	}
	for _, held := range r.resolvedOperands {
		notes = append(notes, held.note())
	}
	if phase != "" {
		notes = append(notes, phase)
	}
	if remedy, isRemedy := refusalRemedies[r.ErrorCode]; isRemedy {
		return append(notes, remedy)
	}
	return append(notes, noBypassNote)
}

// remediationColumns is the `EDIT ONE OF` table's header **for the rows it
// holds**: the coordinate always, the two value columns where a check compared
// two values, and an unnamed sixth where `hyper` derived something about a row.
//
// A column no row fills is not rendered, which is §7's absence rule one axis
// over: a page carrying a `FROM` and a `TO` over nothing states that this check
// had a replacement to offer and declined to name it. Most of the closed set has
// none — an `unknown-key` names a file, a line and a field, and what to put
// there is the author's judgement rather than arithmetic — so on most of this
// table's appearances the coordinate is the whole of it, and the heading is the
// instruction (§8).
//
// The sixth column has no name because it is not a column of the table: it is
// what `hyper` derived about that remediation — a speculative expansion's count,
// a proposal's resolved instant — riding beside the row it is about. A header
// over it would promise every row carries one.
func remediationColumns(rows []render.Row) []string {
	columns := []string{"FILE", "LINE", "FIELD"}
	values, derived := false, false
	for _, row := range rows {
		held, is := row.(remediationRow)
		if !is {
			continue
		}
		values = values || held.from != "" || held.to != ""
		derived = derived || held.derived != ""
	}
	if values {
		columns = append(columns, "FROM", "TO")
	}
	if derived {
		columns = append(columns, "")
	}
	return columns
}

// remediationHeading is the table's own heading, above its columns. It is
// `EDIT ONE OF` even where one row stands beneath it: the header is what the
// table is for rather than a count of what it holds, and a heading that changed
// with the row count would be a second thing for a reader to parse (§8).
const remediationHeading = "EDIT ONE OF"

// remediationRow is one edit past this check, on §8's wire.
//
// It carries **either** a `from` and a `to` **or** a `hint` and the example
// expansion, which is the split between the two things a remediation can be: a
// value to replace, where the check compared two and the replacement is
// arithmetic; and a direction, where narrowing is a judgement and what `hyper`
// contributes is a worked example of one.
//
// `from` and `to` ride as the canonical bytes the entry holds for the values
// they came from, so the number on this row and the number in `outcome.json`
// are one reading (§7, ADR-0026). The `≥` the page draws beside the `to` is the
// page's notation and is not here: what the check found is a count, and what a
// Bound must clear is a fact a reader derives from it (ADR-0059).
//
// `resolved` rides where the remediation renders a relative operand — the
// proposal, glossed against the same instant the current value was, which is
// this Run's start and not the reader's clock (ADR-0034).
type remediationRow struct {
	Type             string            `json:"type"`
	File             string            `json:"file"`
	Line             int               `json:"line,omitempty"`
	Field            string            `json:"field,omitempty"`
	From             json.RawMessage   `json:"from,omitempty"`
	To               json.RawMessage   `json:"to,omitempty"`
	Hint             string            `json:"hint,omitempty"`
	ExampleExpansion *int              `json:"example_expansion,omitempty"`
	Resolved         map[string]string `json:"resolved,omitempty"`
	// derived is the trailing cell: what the page renders beside the row
	// about the row. It is not on the wire, every part of it already being
	// there as a member of its own — a composed string beside its own parts
	// is the second rendering of a fact that can be wrong about the first
	// (§8, ADR-0059).
	derived string
	// from and to are what the page's two value columns render, which is
	// the notation over the values above rather than a second reading of
	// them.
	from, to string
}

// Cells is the row's line in the `EDIT ONE OF` table: where to edit, and what
// to put there.
// Cells is the row's line in the `EDIT ONE OF` table, in that table's widest
// form: the coordinate, the two value columns, and the trailing cell. A row
// that fills fewer of them is trimmed to the header the table actually renders,
// which is writeRefusal's — WriteAligned aligns what it is handed, so a row
// carrying more cells than its header would open a column the header does not
// name (§8, render.WriteAligned).
func (r remediationRow) Cells() []string {
	line := ""
	if r.Line != 0 {
		line = strconv.Itoa(r.Line)
	}
	return []string{r.File, line, r.Field, r.from, r.to, r.derived}
}

// remediationsFor is the edits past one check, in the order `EDIT ONE OF`
// renders them, and nothing at all where an artefact edit is not the way past.
//
// **The general shape is one row: the coordinate the check already cites.**
// That is the whole of what most of the closed set can offer — an
// `unknown-key`, a `reference-unresolvable`, a `header-reserved` each name a
// file, a line and a field, and what to put there is the author's judgement
// rather than arithmetic. The table renders it all the same, because the header
// is the instruction: this is the line, go and edit it.
//
// **A check that compared two values renders them**, which is `bound-exceeded`
// and no other member of the set (§7): the Bound its author declared against the
// count the Expansion resolved to, and the `≥` that says what a Bound admitting
// this Expansion would have to be.
//
// **The narrowed selector rides beside it** where the engine derived one, so
// that a reviewer choosing between the two edits has both counts on the page.
// Without it the only edit offered is the one that widens a `destroy` (§8,
// internal/run/narrow.go).
func remediationsFor(member refusalRow, narrowed *run.Narrowing, started time.Time) []render.Row {
	if _, isRemedy := refusalRemedies[member.ErrorCode]; isRemedy || member.File == "" {
		return nil
	}

	edit := remediationRow{Type: "remediation", File: member.File, Line: member.Line, Field: member.Field}
	if member.Declared != nil && member.Observed != nil {
		edit.From, edit.To = member.Declared, member.Observed
		edit.from, edit.to = string(member.Declared), "≥ "+string(member.Observed)
	}
	rows := []render.Row{edit}
	if narrowed == nil {
		return rows
	}

	expansion := narrowed.Expansion
	proposal := remediationRow{
		Type: "remediation", File: member.File, Line: narrowed.Line, Field: narrowed.Field,
		Hint: "narrow the selector", ExampleExpansion: &expansion,
		from: narrowed.From, to: narrowed.To,
	}
	// The proposal is glossed as the current value is, against the same
	// instant: this Run's start and not the reader's clock (ADR-0034).
	if resolved := resolvedInstant(narrowed.To, started); resolved != "" {
		proposal.Resolved = map[string]string{narrowed.To: resolved}
		proposal.derived = fmt.Sprintf("expands to %d · %s is %s", expansion, narrowed.To, resolved)
	} else {
		proposal.derived = fmt.Sprintf("expands to %d", expansion)
	}
	return append(rows, proposal)
}

// refusalForm is which of §8's two Refusal renderings a set of members takes.
//
// **It is carried and never inferred**, which is the one rule this type exists
// to hold. What sorts the two is what the check had to point at, and that is a
// fact the caller holds rather than one a member's own fields can be read for:
// a `problem.Problem` may carry no file, and a positioned Refusal read back as
// the two-line form would silently drop the caret excerpt and the `EDIT ONE OF`
// table that are the whole path back to a passing review (§8, gate.go).
type refusalForm int

const (
	// aboutTheInvocation is the two-line form: a fact about **this
	// invocation** — a binary the Repository declaration does not pin, a
	// Store branch neither side holds — with no artefact coordinate in it.
	// A caret excerpt needs a file and a line and an `EDIT ONE OF` needs a
	// field, and inventing one to reach the richer rendering would point a
	// reader at an edit that is not the remedy.
	aboutTheInvocation refusalForm = iota
	// citingAnArtefact is §8's caret excerpt, its `=` notes and its `EDIT
	// ONE OF` table: the fault has a file, a line and a field, and the
	// remedy is an edit there.
	citingAnArtefact
)

// writeRefusalMembers writes §8's Refusal for the members a Refusal arrives as,
// in the form its caller named, and it is the whole of what the CLI's
// destination does with one (destination.go).
//
// The members arrive already read against the working tree, because the root an
// excerpt is read against is the caller's (excerpted). What is derived here is
// each member's remediations, which are a function of the member alone.
func writeRefusalMembers(w io.Writer, form refusalForm, members []refusalRow) error {
	if form == aboutTheInvocation {
		for _, member := range members {
			if _, err := fmt.Fprintf(w, "refused: %s\n  %s\n", member.ErrorCode, member.Message); err != nil {
				return err
			}
		}
		return nil
	}

	rows := make([]render.Row, 0, 2*len(members))
	for _, member := range members {
		rows = append(rows, member)
		rows = append(rows, remediationsFor(member, nil, time.Time{})...)
	}
	// phased is false: a phase note says which of §6's phases a check
	// declined in, and a Run is what has phases. Neither form written here
	// is one — a fact about an invocation has no Step to have preceded, and
	// a Probe is not a Run (refusalPhase).
	return writeRefusal(w, rows, false)
}

// writeRefusal writes §8's Refusal: every member of the array, each with its
// own caret excerpt and its own remediation table where it has one, in the
// array's order.
//
// **Every one of them renders.** The Refusal is the entire path back
// (ADR-0001), and rendering the first of five costs an operator five round
// trips, each ending in another `77` (§8).
//
// members and remediations arrive as the rows the stream emits, in the order it
// emits them: a member's remediations are the `remediation` rows between it and
// the next `refusal` row, which is what makes the block the page draws and the
// rows the wire carries one reading of one Refusal (ADR-0026).
//
// phased says the surface has a Run behind it, and false is the Probe: there is
// no Step for a phase to have preceded, so no member of this Refusal carries a
// phase note at all.
func writeRefusal(w io.Writer, rows []render.Row, phased bool) error {
	first := true
	for i, row := range rows {
		member, is := row.(refusalRow)
		if !is {
			continue
		}
		if !first {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		first = false

		if _, err := fmt.Fprintf(w, "refused: %s\n\n", member.ErrorCode); err != nil {
			return err
		}
		phase := ""
		if phased {
			phase = refusalPhase(member)
		}
		if err := writeExcerpt(w, member, phase); err != nil {
			return err
		}
		remediations := remediationsAfter(rows[i+1:])
		if len(remediations) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(w, "\n%s\n", remediationHeading); err != nil {
			return err
		}
		columns := remediationColumns(remediations)
		lines := [][]string{columns}
		for _, row := range remediations {
			lines = append(lines, row.Cells()[:len(columns)])
		}
		if err := render.WriteAligned(w, "", lines); err != nil {
			return err
		}
	}
	return nil
}

// remediationsAfter is the `remediation` rows standing between one `refusal`
// row and the next, which is the pairing the stream's order already states: the
// rows go out in the page's order, and a row's place in it is what says which
// member it is about (§8).
func remediationsAfter(rows []render.Row) []render.Row {
	var found []render.Row
	for _, row := range rows {
		switch row.(type) {
		case refusalRow:
			return found
		case remediationRow:
			found = append(found, row)
		}
	}
	return found
}
