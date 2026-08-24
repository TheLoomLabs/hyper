package compare

import (
	"bytes"
	"cmp"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/cadence"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// `THE CODE MOVED` (§8, §12, issue #171): the Comparison's third table, over
// §12's closed enumeration of nine code facts, terminated by the mandatory
// catch-all.
//
// **A class emits one row per `(subject, fact)` pair**, always — a class is a
// kind of fact and not a row, so a Definition's declared Kinds and its Target's
// accepted ones moving in one window is two rows under one class name. §12's
// nine class names are the grouping and never appear on this screen: a cell
// naming the class would misdescribe every row whose value is shaped unlike it.
//
// **A row's rendering and its comparison both follow the shape of the value at
// that row, never the class above it.** That shape is `internal/artefact`'s own
// reading of the artefact's lines — the vocabulary §12 fixes once and both this
// surface and the review's gutter read, the difference between them being that
// a review reads one file and a Comparison reads two revisions of it.
//
// **Where this implementation reads a Step's `over:`, `bound:` and `target:`
// differs from §8's own account, deliberately.** §8 says four of the nine
// classes read off two Journal entries — `the digests` and the three facts a
// Step file records beside them — and the eight here are read off the artefacts
// at the two revisions instead, `the digests` alone off the entries. Three
// reasons, and the third is the one that decides it: one reader answers the
// review and the Comparison, so a selector cannot be read one way on one screen
// and another way on the other; a Step file is written only by a Step that was
// reached, so a Bound that moved on a Step neither Run reached would emit no
// row at all where the artefacts carry every Step both revisions declared; and
// the row stream carries *the artefact's own parsed shape* for a value that is
// not a scalar (§8), which the Store's canonical encoding — keys in code-point
// order, authored order discarded — cannot supply. What it costs is stated
// where it is paid: a revision the clone does not hold takes eight classes down
// rather than five, leaving `the digests` and the catch-all's own absence line.

// The five subject kinds §8's `SUBJECT` column qualifies a name with. **A
// header reading `DEFINITION` misdescribes every row whose fact belongs to
// something else**, and a bare name is ambiguous across kinds; §12 fixes each
// `kind:` to one directory, so the kind and the name together are a whole path
// in one short cell.
//
// They are the page's words rather than §12's `kind:` values — `target` where
// the wire says `target-declaration`, `manifest` where it says `provider` —
// which is the same two-spellings-of-one-fact the review's own marker column
// header already is (§8, internal/cli's artefactKinds).
const (
	SubjectProcedure  = "procedure"
	SubjectDefinition = "definition"
	SubjectTarget     = "target"
	SubjectManifest   = "manifest"
	SubjectRepository = "repository"
)

// The six facts `the digests` names, one per Provenance member (§7, §12).
//
// **The class is stated intensionally on purpose**: a member joining the field
// set brings a row here without §12's enumeration moving, which is how the
// Procedure revision arrived (ADR-0048) and why there are still nine. It is the
// one class with no key to name, being Run-recorded with no line in any
// artefact, so its rows name their member.
const (
	FactProcedureRevision  = "procedure revision"
	FactDefinitionRevision = "definition revision"
	FactManifestDigest     = "manifest digest"
	FactOriginDigest       = "origin digest"
	FactRepositoryRevision = "repository revision"
	FactHyperVersion       = "hyper version"
)

const (
	// FactOtherLines is the catch-all's own name, and it is the name on the
	// wire in both of that row's forms: the absence carries
	// `baseline_absent` beside the `command` it keeps and drops `count`,
	// **rather than changing `fact`** (§8).
	FactOtherLines = "other lines changed"
	// NotInClone is the one §12 absence this table can carry, and the same
	// name the review's header renders: the Store answered, the window is
	// the right one, and what failed is the clone not holding an object
	// somebody else's Run recorded (ADR-0071).
	NotInClone = "not-in-clone"
	// repositoryDeclaration is where the Repository declaration sits, and
	// the name `hyper_version`'s subject carries — the pin gate refuses any
	// binary whose version differs from the repository's in either
	// direction (§11), so that member moving *is* the pin moving.
	//
	// It is spelled here rather than taken from internal/repository because
	// this package reaches no file and therefore no walk; the two are held
	// together by the corpus, which renders the row.
	repositoryDeclaration = "hyper.yaml"
	// codeRevisionDigits is how much of a git object name the page draws.
	// **The page abbreviates a fact to be recognised** (ADR-0047) and git
	// resolves a short revision, so a revision is drawn short and the wire
	// carries it whole. A `sha256:` digest is not abbreviated at all: no
	// tool resolves a short one, and `show` renders a Manifest digest whole
	// one command over.
	codeRevisionDigits = 7
	// codeStack separates the lines of a stacked cell — a Cadence's
	// expression, phrase and rate, and a selector's form over its members.
	// The renderer opens it out and the column widens (render.WriteAligned).
	codeStack = "\n"
)

// Code is what the caller read for `THE CODE MOVED`: the reviewed artefacts at
// each end of the window, and what git says moved between the two revisions.
//
// It is handed in whole, this package opening no file and starting no
// subprocess: the git reads are `internal/revision`'s and the parse is
// `internal/repository`'s, made by `internal/cli` (compare.go).
//
// A window with no baseline carries none of it. There is no earlier revision
// for code to have moved from, so the table renders no row and no catch-all —
// the count being a count between two revisions, and a command naming one
// reproducing nothing.
type Code struct {
	Baseline CodeSide
	Subject  CodeSide
	// Count is git's own count of the lines that moved over the reviewed
	// five, before the classed rows subtract their own: added and removed
	// as git counts them, a modified line being two (§12).
	Count int
}

// CodeSide is one end of the window's code: the revision that entry recorded,
// whether this clone holds it, and the artefacts it held.
type CodeSide struct {
	// Revision is the `repo_revision` the entry recorded — the commit the
	// catch-all's `git diff` names and the tree the artefacts were read at.
	Revision string
	// InClone says this clone holds that revision. **A revision it does not
	// hold is an ordinary fact about the clone** rather than the world
	// resisting: a Run recorded on a runner names a commit a laptop may
	// never have fetched (ADR-0071).
	InClone bool
	// Dirty is the `repo_dirty` this entry recorded. The bytes that Run
	// read are nowhere in git, which is what suppresses the catch-all's
	// command — a `git diff <rev> <rev>` that does not reproduce what moved
	// is worse than no command at all (§7, §8).
	Dirty bool
	// Artefacts are the reviewed artefacts at that revision, each with the
	// facts its own lines carry.
	Artefacts []CodeArtefact
	// Moved is which line of which path git says moved, read at this
	// revision. It is what a classed row subtracts its own lines out of.
	Moved map[string]map[int]bool
}

// CodeArtefact is one artefact at one revision: the subject it is on the page,
// where it sits in the repository, and the facts `internal/artefact` read off
// its lines.
type CodeArtefact struct {
	// Kind is one of the five subject words above.
	Kind string
	// Name is the name the artefact declares for itself, and the path for
	// the one artefact that declares none.
	Name string
	// Path is where the file sits, relative to the repository root — the
	// key the moved lines are held under, and "" for the one Manifest with
	// no file in the repository (§11, ADR-0039).
	Path  string
	Facts []artefact.ChangeFact
}

// CodeValue is one side of a code row: what the page renders and what the wire
// carries, which are one value in two notations rather than two facts.
//
// **A side with nothing renders `–`**, including where the format states a
// value by omission — an absent `bound:`, an absent `over:`, an absent
// `cadence:`. Naming what an absence means is a claim and not a value, and
// `FLAGS` one surface over is the one editorial place in the tool (§8).
type CodeValue struct {
	// Written says this side states a value at all.
	Written bool
	// Shape decides both what the cell renders and what makes two sides
	// differ. It follows the value at the row and never the class above it:
	// the Target set class alone carries a set for a Procedure's envelope
	// and a scalar for the `target:` a Step binds (§8).
	Shape artefact.FactShape
	// Members are a set's members sorted by code point, a `values:`
	// selector's members **as authored** — its order is the fact — and a
	// predicate selector's conjuncts one per line, sorted by code point on
	// the rendered line (§6, §8).
	Members []string
	// Text is a scalar as written, and a selector's form: `values`, `assets`
	// or `observations`. A cell dropping the form could not tell an `assets`
	// selector from an `observations` one, which is the difference between
	// ranging over what `hyper` built and over what it read (§5, §8).
	Text string
	// Phrase and RateText are the Cadence gloss, and they stand on no other
	// row: cron is write-only for humans and agents alike wherever it is
	// read, and this row is the one place the whole tool renders a Cadence
	// *moving* (§10, ADR-0005, ADR-0063).
	Phrase, RateText string
	// Rate is the number the page rounded into RateText, carried for the
	// wire: the parts and never the composed cell, which is the `artefact`
	// row's rule one command over (§8, §10).
	Rate *float64
	// Wire is the value in the artefact's own parsed shape, and nil where
	// the page renders `–` — the key absent rather than null, which is
	// `from_ordinal`'s rule two row types up (§7, §8).
	Wire json.RawMessage
}

// same reports whether two sides are the same fact, which is what decides
// whether a row is emitted at all.
//
// **A fact that did not move emits no row, however its bytes moved.**
// Reordering `targets: [staging, local]`, or reordering two conjuncts of one
// selector, changes the file and changes nothing this table reports; those
// lines fall to the catch-all's count. That is the comparison being by the
// fact's own equality and never by the text (§8, §12).
//
// **`Written` is not consulted, and that is the rule rather than an omission.**
// For a set-shaped fact an absent key and an empty list are one value — the
// same `–` the review's `AUTHORITY` table already renders for a Definition
// claiming no `destroy` Operations — and for every other shape a key written at
// something this cannot read as a value renders `–` too (cell above). Two sides
// the page renders identically are two sides a row would assert a difference
// between while showing none, which is the one thing ADR-0026 forbids; what is
// wrong with the unreadable key is `check`'s to report (ADR-0064).
func (v CodeValue) same(other CodeValue) bool {
	return v.Text == other.Text && slices.Equal(v.Members, other.Members)
}

// cell is the value as the column draws it, and `–` for a side with nothing.
//
// **A `FROM`/`TO` cell renders its value whole**, and the column widens and the
// row wraps to as many lines as the two values need. Nothing is truncated and
// nothing is elided: ADR-0059's whole-or-`changed` rule governs the `FIELDS`
// column and does not reach here — nor could it be extended to, since it
// disqualifies anything nested and a selector is nested by construction, so the
// extension would render every selector `changed`, which is the one word this
// column can never carry (§8).
func (v CodeValue) cell(abbreviate bool) string {
	if !v.Written {
		return sideNothing
	}
	switch v.Shape {
	case artefact.FactSet:
		if len(v.Members) == 0 {
			return sideNothing
		}
		return strings.Join(v.Members, fieldGap)
	case artefact.FactSelector:
		if v.Text == "" {
			return sideNothing
		}
		return strings.Join(append([]string{v.Text}, v.selectorLines()...), codeStack)
	case artefact.FactCadence:
		// An expression outside §10's grammar has no reading and
		// contributes no gloss, exactly as the review's own header does
		// not gloss one; what is left is the scalar under the shape
		// rule, and the guard beneath catches the one written at
		// nothing at all.
		if v.Text != "" && v.Phrase != "" {
			return strings.Join([]string{v.Text, v.Phrase, v.RateText}, codeStack)
		}
	}
	if v.Text == "" {
		return sideNothing
	}
	if abbreviate {
		return abbreviatedObject(v.Text)
	}
	return v.Text
}

// abbreviatedObject is a git object name or a digest as the page draws it: the
// algorithm kept where `hyper` named one, and the hex shortened behind it.
//
// **The page abbreviates a fact to be recognised** and the wire carries every
// one of them whole (§8, ADR-0047). The algorithm is kept because it is not the
// part being recognised: `hyper` names the algorithm where `hyper` chose it
// (§7), and a cell reading seven hex digits with no algorithm in front of them
// would be a digest and a revision rendered identically in one column.
func abbreviatedObject(name string) string {
	algorithm, hex, named := strings.Cut(name, ":")
	if !named {
		if len(name) <= codeRevisionDigits {
			return name
		}
		return name[:codeRevisionDigits]
	}
	if len(hex) <= codeRevisionDigits {
		return name
	}
	return algorithm + ":" + hex[:codeRevisionDigits]
}

// selectorLines is a selector's members beneath the form heading it: one
// ` · `-separated run for a `values:` selector, whose order is the fact, and
// one `field operator operand` line per conjunct for a predicate one (§8).
func (v CodeValue) selectorLines() []string {
	if len(v.Members) == 0 {
		return nil
	}
	if v.Text == "values" {
		return []string{strings.Join(v.Members, fieldGap)}
	}
	return v.Members
}

// CodeRow is one row of `THE CODE MOVED`, and the catch-all that terminates the
// table is one of these too.
//
// **One type carries both** because §12 states the catch-all as a row of this
// table rather than as a line beneath it, and because a consumer filtering
// `select(.type=="code")` gets the enumeration and the count that makes it sum
// to the whole. The two are told apart by `fact`, which is the name §8 fixes
// and which the absence form keeps.
//
// **`subject_kind` and `subject` stand where the fact has an artefact subject
// and neither stands where it does not** — the pair being the one cell the
// table renders `—` in, which `repo_revision` alone earns: it belongs to no
// artefact a reader can open (§8).
type CodeRow struct {
	SubjectKind string
	Subject     string
	Fact        string
	From, To    CodeValue
	// Count is the catch-all's own, and a pointer because `0` is a count
	// this row must be able to state: a window in which every moved line is
	// reported by a classed row above reads `0 other lines changed`.
	Count *int
	// BaselineAbsent is `not-in-clone` on the catch-all's replacement form
	// and "" everywhere else. It stands **in place of** `count`, the count
	// being the part that needed the bytes (§8, §12).
	BaselineAbsent string
	// Command is `git diff <rev> <rev>`, abbreviated as the page draws it —
	// the one string on this wire that keeps the page's abbreviation, being
	// a command a reader runs rather than an id the row reports, and one
	// git resolves short (§8). It is "" where either side recorded
	// `repo_dirty`.
	Command string
}

// MarshalJSON writes the row in §8's own key order, and writes a member only
// where the row carries one.
//
// It is assembled rather than tagged because three of the members are values
// rather than strings — a `from` may be a number, an array or a mapping, and
// the Cadence gloss's parts ride beside it — and because the absence rule is
// stated over a **key**: a side with nothing writes no key, exactly as
// `from_ordinal` does two row types up (§7, §8).
//
// The canonical encoding indents and breaks lines and this wire is compact, so
// the assembled object goes through one compaction: a value read off the Store
// and a value read off an artefact arrive in one notation
// (FieldSet.MarshalJSON, one file over).
func (r CodeRow) MarshalJSON() ([]byte, error) {
	var written bytes.Buffer
	written.WriteString(`{"type":"code"`)
	member := func(key string, value json.RawMessage) {
		if value == nil {
			return
		}
		written.WriteByte(',')
		written.Write(jsonText(key))
		written.WriteByte(':')
		written.Write(value)
	}
	if r.SubjectKind != "" {
		member("subject_kind", jsonText(r.SubjectKind))
		member("subject", jsonText(r.Subject))
	}
	member("fact", jsonText(r.Fact))
	member("from", r.From.Wire)
	member("to", r.To.Wire)
	member("from_phrase", phraseWire(r.From))
	member("to_phrase", phraseWire(r.To))
	member("from_rate", rateWire(r.From))
	member("to_rate", rateWire(r.To))
	if r.Count != nil {
		member("count", json.RawMessage(strconv.Itoa(*r.Count)))
	}
	if r.BaselineAbsent != "" {
		member("baseline_absent", jsonText(r.BaselineAbsent))
	}
	if r.Command != "" {
		member("command", jsonText(r.Command))
	}
	written.WriteByte('}')

	var compact bytes.Buffer
	if err := json.Compact(&compact, written.Bytes()); err != nil {
		return nil, err
	}
	return compact.Bytes(), nil
}

// phraseWire is a Cadence's phrase as the wire carries it, and nothing on every
// other row: a phrase is a total function of the five fields, so a row that has
// one always has one (§10).
func phraseWire(value CodeValue) json.RawMessage {
	if value.Phrase == "" {
		return nil
	}
	return jsonText(value.Phrase)
}

// rateWire is the rate as a JSON number, rounded exactly as the page rounded
// it. **It is never a band or a word**: a rate whose only form is prose puts an
// escalating agent back where the number exists to take it out of (§8, §10).
func rateWire(value CodeValue) json.RawMessage {
	if value.Rate == nil {
		return nil
	}
	written, err := json.Marshal(*value.Rate)
	if err != nil {
		return nil
	}
	return written
}

// jsonText is one string as this wire writes it.
//
// It is `internal/store`'s own string writer rather than `encoding/json`'s,
// which is what the `FIELDS` column one file over already writes a path with:
// **HTML escaping is off** on this stream — the wire carries an artefact's own
// bytes, and a value quoting a `&` or a `<` is one a consumer reads back as it
// was written (§8, render.WriteJSON) — and `json.Marshal` escapes all three.
func jsonText(text string) json.RawMessage {
	return store.Unframed(store.String(text))
}

// CatchAll reports whether this is the row that terminates the table: the
// count, or the line that stands where the bytes could not be read.
func (r CodeRow) CatchAll() bool { return r.Fact == FactOtherLines }

// Cells is the row's line under `SUBJECT`, `FACT`, `FROM`, `TO`.
//
// The catch-all has none: it is a stated line beneath the table rather than a
// row of it, on the shape §8 renders and the shape the review's own
// `3 definitions did not load` line already has. What it says is Line below.
func (r CodeRow) Cells() []string {
	if r.CatchAll() {
		return nil
	}
	abbreviate := r.abbreviates()
	return []string{r.subjectCell(), r.Fact, r.From.cell(abbreviate), r.To.cell(abbreviate)}
}

// subjectCell is the kind-qualified name, and `—` for the one fact that belongs
// to no artefact a reader can open.
func (r CodeRow) subjectCell() string {
	if r.SubjectKind == "" {
		return codeNoSubject
	}
	return r.SubjectKind + " " + r.Subject
}

// abbreviates reports whether this row's two values are git object names the
// page draws short.
//
// It is read off the row's own `fact`, which is the member that says what the
// value is: five of the six Provenance members are object names an eye compares
// by their first digits, and the sixth is a `hyper` version — a release string
// that is already its whole self, and one whose two halves a reader has to see
// to tell `1.4.0` from `1.14.0` (§8, ADR-0047).
func (r CodeRow) abbreviates() bool {
	switch r.Fact {
	case FactProcedureRevision, FactDefinitionRevision, FactRepositoryRevision, FactManifestDigest, FactOriginDigest:
		return true
	}
	return false
}

// Line is the catch-all's own line, and "" on every classed row.
//
// **Two suppressions, and they stack.** Where either side recorded `repo_dirty`
// the command is suppressed and the row renders `N other lines changed` alone,
// `git diff <rev> <rev>` not reproducing what moved. Where the clone does not
// contain a revision the window names, the count is replaced by the line naming
// what could not be read — which keeps the command, the reader of a job summary
// rarely being in the clone that came up short (§8).
func (r CodeRow) Line() string {
	if !r.CatchAll() {
		return ""
	}
	head := "other lines could not be counted"
	if r.Count != nil {
		// §12 writes the row as `N other lines changed`, and the `N`
		// there is a placeholder rather than a claim about grammar: one
		// line takes the singular, exactly as the count beside a
		// table's head does one package over (internal/cli's countOf).
		head = strconv.Itoa(*r.Count) + " other line" + plural(*r.Count) + " changed"
	}
	if r.Command == "" {
		return head
	}
	return head + fieldGap + r.Command
}

// plural is the `s` a count of one does not take.
func plural(counted int) string {
	if counted == 1 {
		return ""
	}
	return "s"
}

const (
	// codeNoSubject is the `—` `repo_revision` alone renders: *this fact
	// belongs to no artefact you can open*. It is an em dash where a side
	// with nothing is an en dash, the two being different absences — one is
	// a fact with no subject and the other is a subject with no value (§8).
	codeNoSubject = "—"
)

// CodeRows is `THE CODE MOVED`: the classed rows in `(SUBJECT, FACT)` order and
// the catch-all last of all.
//
// **Rows sort by `(SUBJECT, FACT)` on Unicode code point, with the `—` subject
// after every named one.** `—` means *this fact belongs to no artefact you can
// open*, so it sorts away from the rows that do, and §12 already fixes the
// catch-all as terminating the table (§8).
//
// A window with no baseline draws nothing at all. Its subject Run is the first
// Run of its Procedure, so there is no earlier revision for code to have moved
// from, no pair of revisions for `git diff` to name, and nothing for a count to
// be a count of.
func CodeRows(window Window, code Code) []render.Row {
	if !window.Baseline.Present || !window.Subject.Present {
		return nil
	}

	drawn := drawnPairs(window, code)
	rows := make([]render.Row, 0, len(drawn)+1)
	for _, pair := range drawn {
		rows = append(rows, pair.row())
	}
	return append(rows, catchAllRow(code, drawn))
}

// drawnPairs is every `(subject, fact)` pair that moved, sorted as the page
// renders them.
func drawnPairs(window Window, code Code) []codePair {
	pairs := append(digestPairs(window), artefactPairs(code)...)
	drawn := make([]codePair, 0, len(pairs))
	for _, pair := range pairs {
		if !pair.from.value.same(pair.to.value) {
			drawn = append(drawn, pair)
		}
	}
	slices.SortFunc(drawn, byCodePair)
	return drawn
}

// codePair is one `(subject, fact)` pair with both ends of the window read for
// it: what each side's value is, and which lines of which file it occupies
// there.
type codePair struct {
	subject  codeSubject
	fact     string
	from, to codeEnd
}

// codeSubject is a row's subject: the kind word and the name, or the empty pair
// for the one fact that belongs to no artefact.
type codeSubject struct{ kind, name string }

// codeEnd is one end of a pair: the value, and where it is written.
type codeEnd struct {
	value CodeValue
	// path is the artefact's repository path and lines are the lines the
	// value occupies there. They are what the catch-all subtracts, and both
	// are empty on a fact with no line in any artefact — `the digests`,
	// which is Run-recorded (§12).
	path  string
	lines []int
}

// row is the pair as the surface carries it.
func (p codePair) row() CodeRow {
	return CodeRow{
		SubjectKind: p.subject.kind,
		Subject:     p.subject.name,
		Fact:        p.fact,
		From:        p.from.value,
		To:          p.to.value,
	}
}

// byCodePair orders two rows as the page renders them: the rendered subject by
// code point with the `—` subject after every named one, then the fact.
func byCodePair(a, b codePair) int {
	if (a.subject.kind == "") != (b.subject.kind == "") {
		if a.subject.kind == "" {
			return 1
		}
		return -1
	}
	return cmp.Or(
		strings.Compare(a.subject.kind+" "+a.subject.name, b.subject.kind+" "+b.subject.name),
		strings.Compare(a.fact, b.fact),
	)
}

// digestPairs is `the digests`: every member of the Provenance, read off the
// two Journal entries.
//
// **Each member takes its own subject.** The Procedure revision's is the
// Procedure, `definition_revision`'s the Definition and `manifest_digest`'s and
// `origin_digest`'s the Manifest — one row per name either Run's Step files
// carried — `hyper_version`'s the Repository declaration whose pin it cannot
// differ from (§11), and `repo_revision`'s no artefact at all, which is the one
// cell in the table that renders `—` (§8, §12).
func digestPairs(window Window) []codePair {
	baseline, subject := window.Baseline.Entry.Provenance, window.Subject.Entry.Provenance
	pairs := []codePair{
		digestPair(codeSubject{SubjectProcedure, window.Procedure}, FactProcedureRevision, baseline.ProcedureRevision, subject.ProcedureRevision),
		digestPair(codeSubject{SubjectRepository, repositoryDeclaration}, FactHyperVersion, baseline.HyperVersion, subject.HyperVersion),
		digestPair(codeSubject{}, FactRepositoryRevision, baseline.RepoRevision, subject.RepoRevision),
	}

	// The Definitions and the Manifests each Run named, folded from both
	// sides' Step files. **Which Manifests a Run read is the Step files'
	// `provider`** (§7): `manifest_digest` names the bytes that ran and
	// never which Provider they were, so a table enumerating the Manifests
	// any other way would be resolving each Step's `definition` to its
	// `provider:` at that Step's own revision — a git object per Step, on a
	// surface that already holds both revisions of the artefacts (§8).
	was, is := stepProvenance(window.Baseline), stepProvenance(window.Subject)
	for _, name := range namesAcross(was.definitions, is.definitions) {
		pairs = append(pairs, digestPair(codeSubject{SubjectDefinition, name}, FactDefinitionRevision, was.definitions[name], is.definitions[name]))
	}
	for _, name := range namesAcross(was.manifests, is.manifests) {
		pairs = append(pairs,
			digestPair(codeSubject{SubjectManifest, name}, FactManifestDigest, was.manifests[name], is.manifests[name]),
			digestPair(codeSubject{SubjectManifest, name}, FactOriginDigest, was.origins[name], is.origins[name]),
		)
	}
	return pairs
}

// stepDigests is what one side's Step files recorded, by the name each member
// belongs to.
type stepDigests struct{ definitions, manifests, origins map[string]string }

// stepProvenance folds one side's Step files into the three maps.
//
// A record naming an artefact and carrying no revision for it contributes
// nothing, which is a closing write's reading of a Definition: it names what
// the Step was going to do and carries no Provenance, the reaper having
// established none (§7).
func stepProvenance(side Side) stepDigests {
	read := stepDigests{definitions: map[string]string{}, manifests: map[string]string{}, origins: map[string]string{}}
	if !side.Present {
		return read
	}
	for _, step := range side.Steps {
		if step.Definition != "" && step.Provenance.DefinitionRevision != "" {
			read.definitions[step.Definition] = step.Provenance.DefinitionRevision
		}
		if step.Provider == "" {
			continue
		}
		if step.Provenance.ManifestDigest != "" {
			read.manifests[step.Provider] = step.Provenance.ManifestDigest
		}
		if step.Provenance.OriginDigest != "" {
			read.origins[step.Provider] = step.Provenance.OriginDigest
		}
	}
	return read
}

// namesAcross is every key either side carried, in code-point order. It is one
// fold for the three things this file pairs across a window — the artefacts,
// their facts and the names a Provenance member belongs to — because all three
// are *what either end held*, and a second fold is a second place for one of
// them to drop a name only one side has.
func namesAcross[V any](was, is map[string]V) []string {
	held := map[string]bool{}
	for name := range was {
		held[name] = true
	}
	for name := range is {
		held[name] = true
	}
	return slices.Sorted(maps.Keys(held))
}

// digestPair is one member of the Provenance as a pair: a scalar on both sides,
// with no line in any artefact for the catch-all to subtract.
func digestPair(subject codeSubject, fact, was, is string) codePair {
	return codePair{
		subject: subject,
		fact:    fact,
		from:    codeEnd{value: digestValue(was)},
		to:      codeEnd{value: digestValue(is)},
	}
}

// digestValue is one recorded member, and the side with nothing where the
// entry recorded none — an `origin_digest` on a Provider with no upstream to
// have come from, and every member of a side whose Step files never named that
// artefact (§7, ADR-0073).
func digestValue(recorded string) CodeValue {
	if recorded == "" {
		return CodeValue{}
	}
	return CodeValue{Written: true, Shape: artefact.FactScalar, Text: recorded, Wire: jsonText(recorded)}
}

// artefactPairs is the eight classes authored in the artefacts' own lines, read
// at both revisions.
//
// Artefacts are paired by `(kind, name)` and their facts by the key the fact is
// written at, which is what makes a Definition added or deleted between the two
// revisions render its facts against a side with nothing rather than fall out
// of the table: a deleted Definition appears here alongside every other
// artefact change (ADR-0012).
func artefactPairs(code Code) []codePair {
	if !code.Baseline.InClone || !code.Subject.InClone {
		// **These classes read bytes, and a revision the clone does not
		// hold has none to read.** They are dropped rather than rendered
		// against a side with nothing: `– → destroy · mutate` on a
		// window whose baseline was never fetched asserts that a Kind
		// *appeared*, which is a claim about bytes nobody read — the one
		// reading this table may never produce (§8, ADR-0071). What
		// says so instead is the catch-all's replacement line, which
		// names the gap where a dropped row would not.
		return nil
	}
	was, is := indexArtefacts(code.Baseline), indexArtefacts(code.Subject)
	var pairs []codePair
	for _, key := range namesAcross(was, is) {
		baseline, heldByBaseline := was[key]
		subject, heldBySubject := is[key]
		named := baseline
		if !heldByBaseline {
			named = subject
		}
		facts := factsBy(baseline, heldByBaseline)
		later := factsBy(subject, heldBySubject)
		for _, fact := range namesAcross(facts, later) {
			pairs = append(pairs, codePair{
				subject: codeSubject{named.Kind, named.Name},
				fact:    fact,
				from:    artefactEnd(baseline, facts[fact]),
				to:      artefactEnd(subject, later[fact]),
			})
		}
	}
	return pairs
}

// indexArtefacts is one side's artefacts by the pair that identifies them
// across the window: their kind and the name they declare.
func indexArtefacts(side CodeSide) map[string]CodeArtefact {
	index := make(map[string]CodeArtefact, len(side.Artefacts))
	for _, found := range side.Artefacts {
		index[found.Kind+"\x00"+found.Name] = found
	}
	return index
}

// factsBy is one artefact's facts by key, **less the ones a Step owns**: a
// Step's `target:`, `bound:` and `over:` are qualified by a coordinate and are
// paired below, where a Step present on one side only can be told from a key
// the artefact never wrote.
func factsBy(held CodeArtefact, present bool) map[string]artefact.ChangeFact {
	facts := map[string]artefact.ChangeFact{}
	if !present {
		return facts
	}
	for _, fact := range held.Facts {
		key := fact.Key
		if fact.Step != "" {
			key = "step " + fact.Step + fieldGap + fact.Key
		}
		facts[key] = fact
	}
	return facts
}

// artefactEnd is one side of an artefact-authored fact: the value the file
// states, and the lines it occupies there.
func artefactEnd(held CodeArtefact, fact artefact.ChangeFact) codeEnd {
	end := codeEnd{path: held.Path, lines: fact.Lines}
	if !fact.Written() {
		return end
	}
	end.value = CodeValue{
		Written: true,
		Shape:   fact.Shape,
		Members: fact.Members,
		Text:    fact.Value,
		Wire:    fact.Wire,
	}
	if fact.Shape == artefact.FactCadence {
		// **Two rules meet in that cell and neither is the other.** A
		// Cadence's value is a cron expression, which is a scalar and
		// renders as one under the shape rule; what stacks the cell is
		// the mandatory gloss, a fact about cron being write-only
		// rather than about the value being compound (§8, §10,
		// ADR-0005, ADR-0063). An expression outside §10's grammar has
		// no reading and carries none, exactly as the review's own
		// header does not gloss one.
		if gloss, readable := cadence.Read(fact.Value); readable {
			rate := gloss.Rate
			end.value.Phrase, end.value.RateText, end.value.Rate = gloss.Phrase, gloss.RateText, &rate
		}
	}
	return end
}

// catchAllRow terminates the table: the lines that moved and that no row above
// reports, or the line that stands where the bytes could not be read.
//
// **The catch-all counts `git diff` lines** — added and removed as git counts
// them, a modified line being two — over the reviewed five and nothing else,
// minus the lines a classed row above already reports. The word *other* is what
// makes the enumeration and the count sum to the whole rather than overlap, so
// each classed fact is mapped to its lines at both revisions (§8, §12).
//
// **What a classed row subtracts is the lines its own value occupies and
// nothing else.** A Manifest gaining an Operation moves the whole block that
// Operation is written as, and the `operations` row reports one name: the key
// line is subtracted, and the request, the projection and the declared Kind
// beneath it are not, being reported by no row above. Subtracting the block
// would have the catch-all silently drop lines it exists to guarantee are never
// dropped, which is the one thing the word *other* forbids.
func catchAllRow(code Code, drawn []codePair) CodeRow {
	row := CodeRow{Fact: FactOtherLines, Command: diffCommand(code)}
	if !code.Baseline.InClone || !code.Subject.InClone {
		// The eight artefact classes and the count read bytes, and a
		// revision this clone does not hold has none to read. The row
		// is **replaced** rather than joined, two terminating lines
		// being the doubling this chapter refuses everywhere else, and
		// it names no member — on the shape of the review's own
		// `3 definitions did not load` line (§8, §12, ADR-0071).
		row.BaselineAbsent = NotInClone
		return row
	}
	// The clamp cannot fire and is not there to be relied on: every line a
	// row subtracts is one a hunk marked, and a line is subtracted once
	// however many facts are written across it (reported below), so the
	// subtrahend is a subset of the count by construction. It stands
	// because a negative count would render as one, and the one thing this
	// row may never say is a number no reader can check against the command
	// beside it.
	counted := max(0, code.Count-reported(code, drawn))
	row.Count = &counted
	return row
}

// diffCommand is the command the row names, and "" where either side recorded
// `repo_dirty`.
//
// **The bytes that Run read are nowhere in git** — a dirty baseline's working
// tree is gone for good — so `git diff <rev> <rev>` does not reproduce what
// moved, and printing a command that does not reproduce is worse than printing
// none. `N` and the classed rows are still computed between the two committed
// revisions, which is what the Provenance names and what a reader can check out
// (§7, §8).
func diffCommand(code Code) string {
	if code.Baseline.Dirty || code.Subject.Dirty {
		return ""
	}
	if code.Baseline.Revision == "" || code.Subject.Revision == "" {
		return ""
	}
	// The command keeps the page's abbreviation on the wire as well, alone
	// among the strings this stream carries: it is a command a reader runs
	// rather than an id the row reports, and git resolves it short (§8).
	return "git diff " + abbreviatedObject(code.Baseline.Revision) + " " + abbreviatedObject(code.Subject.Revision)
}

// reported is how many of the moved lines the classed rows above already
// account for.
//
// A line is subtracted once however many facts are written across it: a flow
// sequence puts a key and its members on one line, and subtracting it twice
// would have the count report fewer lines than moved — the same failure in the
// other direction from subtracting a block.
func reported(code Code, drawn []codePair) int {
	subtracted := map[string]bool{}
	counted := 0
	for _, pair := range drawn {
		for _, side := range []struct {
			end   codeEnd
			moved map[string]map[int]bool
			at    string
		}{
			{pair.from, code.Baseline.Moved, "from"},
			{pair.to, code.Subject.Moved, "to"},
		} {
			if side.end.path == "" {
				continue
			}
			for _, line := range side.end.lines {
				if !side.moved[side.end.path][line] {
					continue
				}
				key := side.at + "\x00" + side.end.path + "\x00" + strconv.Itoa(line)
				if subtracted[key] {
					continue
				}
				subtracted[key] = true
				counted++
			}
		}
	}
	return counted
}

// CodeRowsIn is the code rows read back off a row list, which is how a page
// reads them: the block is written from the rows, so the stream and the page
// cannot carry different facts (ADR-0026).
func CodeRowsIn(rows []render.Row) []CodeRow {
	var held []CodeRow
	for _, row := range rows {
		if code, is := row.(CodeRow); is {
			held = append(held, code)
		}
	}
	return held
}

// CodeMovedPhrase is `TOTALS`' last segment, and **its three forms are tested
// in order**: any classed row rendered → *the code moved*; otherwise the
// absence line rendered → *the code could not be fully read*; otherwise → *the
// code did not move*.
//
// **The order is what makes the line honest rather than merely careful.** A
// surviving classed row is positive proof where the absence line is proof of
// nothing either way, so a window in which a Bound moved *and* a Target
// declaration could not be read is reported by the fact — the line above having
// already named what went uncounted. What the ordering removes is the one
// reading this table may never produce: the negative asserted over bytes nobody
// read (§8).
//
// It is a phrase and not a count: summing a classed fact, a repository revision
// and a line count into one integer is three incommensurable things under one
// head.
func CodeMovedPhrase(rows []CodeRow) string {
	absent := false
	for _, row := range rows {
		if !row.CatchAll() {
			return "the code moved"
		}
		absent = absent || row.BaselineAbsent != ""
	}
	if absent {
		return "the code could not be fully read"
	}
	return "the code did not move"
}
