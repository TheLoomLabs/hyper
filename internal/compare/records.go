package compare

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The two Record tables (§8, issue #170): `YOU DID THIS` and `THE WORLD MOVED`.
//
// They land together because they share the whole of the derivation —
// eligibility, the endpoints, `FIELDS`, the ordinal, the sorting — and differ
// only in their three change names and in the Tombstone rule. Splitting them
// would build one derivation twice.
//
// **A row is a Record identity, and there is one per identity per table**
// (ADR-0058). Its change name is read from what the baseline end held against
// what the subject end holds and never from the versions between them, so a
// Record a single Run both changed and destroyed is one `destroyed` row
// spanning both.

// Change is the name a row carries, and §8 closes the set at five words over
// two tables: Assets render `created`, `changed` and `destroyed`, Observations
// `appeared`, `changed` and `vanished`, exclusive within each table.
//
// The word is the same in both tables where it is the same fact, which is why
// there are five and not six: `changed` is *the endpoints differ* whichever
// actor the table names.
type Change string

const (
	// ChangeCreated is an Asset the baseline end held no version of.
	ChangeCreated Change = "created"
	// ChangeChanged is two endpoints that differ. It is the one name read
	// from the Records alone in both tables.
	ChangeChanged Change = "changed"
	// ChangeDestroyed is a subject end holding a Tombstone the baseline end
	// did not, a Tombstone being a marker inside the Asset table rather
	// than a class of its own (§7, ADR-0033).
	ChangeDestroyed Change = "destroyed"
	// ChangeAppeared is an identity the subject Run concluded about and the
	// baseline Run did not. It is Disposition-derived, which is what makes
	// it beat ChangeChanged wherever both fire.
	ChangeAppeared Change = "appeared"
	// ChangeVanished is an identity the baseline Run concluded about and
	// the subject Run did not: a thing that stopped being there, with the
	// baseline end's fields beside it because the Store holds no version
	// minted for a disappearance.
	ChangeVanished Change = "vanished"
)

// End is one end of the window for one Record: the version a reader would have
// called the Head had they looked at that end's instant, and what that version
// projected.
//
// Held is what says the end has a version at all, and it is a member rather
// than a nil check because both of the states it distinguishes are ordinary: a
// `created` row's baseline end holds none, and so does the baseline end of the
// series a Tombstone opened.
//
// Fields is the version's projected content, read by the caller. It is nil on
// an end holding no version, and nil on a Tombstone opening the series it ends
// — the absence §7 reads as *`hyper` destroyed this and never observed what it
// was* (ADR-0033).
type End struct {
	Held    bool
	Version store.Version
	Fields  store.Mapping
}

// Record is one eligible identity with both ends of the window read for it.
//
// It is the caller's read and this package's derivation: which identities are
// eligible is Eligible below, which version stands at each end is Endpoint, and
// the bytes behind them are `internal/store`'s.
type Record struct {
	Identity store.Identity
	Baseline End
	Subject  End
}

// Eligible is the identities a window may draw a row for, in `(Target,
// Definition, name)` order.
//
// **They come from the identity sets and never from the Store** (ADR-0058). A
// row exists for an identity some Step of the subject Run or of the baseline
// Run concluded about (§7), which is what keeps another Procedure's work out of
// this Procedure's tables — the same evidence that already decides *vanished*,
// *appeared* and *nothing moved* deciding eligibility outright.
//
// The endpoints are then read without asking whose Run wrote them: a Record
// this Step concluded about, which another Procedure moved in between, renders
// `changed` and the gap shows in `ORDINAL`. Reading `run_id` to name a row
// would make this surface report authorship across Procedures, which is the
// join the window rule exists to prevent.
//
// A Step's Target and Definition are what turn its set's names into
// identities: a set holds the names a Step concluded about, and the Step it
// sits in is what says which series each of them belongs to (§7).
func Eligible(window Window) []store.Identity {
	baseline, subject := readingsOf(window)
	concluded := map[store.Identity]bool{}
	for _, side := range []reading{baseline, subject} {
		for id := range side.held {
			concluded[id] = true
		}
	}

	ids := make([]store.Identity, 0, len(concluded))
	for id := range concluded {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, byIdentity)
	return ids
}

// readingsOf is both sides' accounts of what they concluded about, read off one
// window.
//
// It is one door rather than a call per side because the two are only readable
// against each other: whether a side is silent about an identity is answered by
// the **other** side's account of which Steps concluded about it (reading.silent).
func readingsOf(window Window) (baseline, subject reading) {
	return readingOf(window.Baseline), readingOf(window.Subject)
}

// Endpoint is §7's Head derivation with one side's instant as a cutoff: the
// version a reader would have called the Head had they looked then.
//
// It is a prefix of the same ordering the Head comes off — `written_at`, ties
// broken by the file name — so it is that derivation read short rather than a
// second one, and a version written after the instant is not consulted at all.
//
// A side that is not there holds no version. That is the baseline of a first
// Run, where every Asset the subject holds is `created` and every Observation
// `appeared`, and it is the answer this returns rather than a caller's special
// case.
func Endpoint(side Side, series store.Series) (store.Version, bool) {
	if !side.Present {
		return store.Version{}, false
	}
	instant := side.Instant()
	var stood store.Version
	found := false
	for _, version := range series.Versions {
		if version.WrittenAt.After(instant) {
			break
		}
		stood, found = version, true
	}
	return stood, found
}

// reading is one side's account of what it concluded about: which identities
// its Steps carried, which Step carried each of them, and what the side is able
// to say about a Step it does not hold an identity under.
//
// The three maps are one value because the last of them is only readable
// against the first two: *this side says nothing about that identity* is a fact
// about the Steps that would have covered it, and those are named by the
// **other** side's account of the same identity.
type reading struct {
	// present is whether this side of the window exists at all.
	present bool
	// held maps an identity to the authored ids of the Steps of this side
	// that concluded about it.
	held map[store.Identity][]string
	// complete is the authored ids whose record on this side carries a set
	// this side read in full — a set with no failed projection path beside
	// it (§6, §7).
	complete map[string]bool
	// recorded is the authored ids this side holds a record for at all,
	// whether or not that record carries a set.
	recorded map[string]bool
	// ranToTheEnd is whether the Run reached every Step its Procedure
	// declared, which is what its `completed` outcome says. It is what
	// tells a Step this Run's revision did not have from one it never
	// reached: both are an absent record, and only the second is §12's
	// seventh Disposition (§7).
	ranToTheEnd bool
}

// readingOf reads one side's account off its Step records.
//
// The members are the caller's: a set whose digest did not move is written in
// an earlier entry, and reading it back is a walk of the Journal rather than a
// derivation (§7, ADR-0055). A record carrying a digest reaches here with its
// members resolved.
func readingOf(side Side) reading {
	read := reading{
		present:  side.Present,
		held:     map[store.Identity][]string{},
		complete: map[string]bool{},
		recorded: map[string]bool{},
	}
	if !side.Present {
		return read
	}
	if outcome, closed := side.Entry.Outcome(); closed && outcome == store.OutcomeCompleted {
		read.ranToTheEnd = true
	}
	for _, step := range side.Steps {
		read.recorded[step.ID] = true
		if step.Identities.Digest == "" {
			// Three of §12's seven Dispositions carry no set and a
			// fourth writes no file, and a closing write carries
			// none either (§7, ADR-0076). Such a Step contributes
			// no identity to its side and removes none from the
			// other (ADR-0058).
			continue
		}
		if step.ProjectionFailedPath == "" {
			read.complete[step.ID] = true
		}
		for _, name := range step.Identities.Members {
			id := store.Identity{Target: step.Target, Definition: step.Definition, Name: name}
			read.held[id] = append(read.held[id], step.ID)
		}
	}
	return read
}

// holds answers whether this side concluded about an identity, and which of its
// Steps did.
func (r reading) holds(id store.Identity) ([]string, bool) {
	steps, held := r.held[id]
	return steps, held
}

// silent answers whether this side's account says nothing about an identity it
// does not hold — where the Steps that would have covered it are named by the
// other side's account of the same identity.
//
// **A partial set is read for what it holds and never for what it omits.**
// Where a Step's Disposition carries the path a projection failed on, an
// identity missing from its set is one `hyper` did not read rather than one the
// world removed: it draws no `vanished` row as subject, and no `appeared` row
// as baseline. **A Step whose Disposition carries no set at all** — a
// `closed-by/` file, every *attempted, world untouched*, a `when:` that did not
// hold — is that rule at its limit and is read the same way (§8, ADR-0058).
//
// **A Step this side holds no record for at all is silent only where the Run
// stopped short.** Inside an entry that did not run to its end that absence is
// §12's seventh Disposition, *never reached*, which carries no set like the
// other three. Inside a `completed` one it is a Step that Run's revision did
// not have — which is not a silence but an absence of sight, and an absence of
// sight is exactly what `appeared` and `vanished` report. Telling the two apart
// by the outcome costs no byte: a Run that completed reached every Step it
// declared.
//
// A side that is not there is not silent either. The baseline of a first Run
// never had the identity in view, and `appeared` is what §8 says about a Record
// this Procedure is seeing for the first time.
func (r reading) silent(steps []string) bool {
	if !r.present {
		return false
	}
	for _, id := range steps {
		switch {
		case r.complete[id]:
			return false
		case r.recorded[id]:
			continue
		case r.ranToTheEnd:
			return false
		}
	}
	return true
}

// changeRowsOf is the two Record tables' rows, Assets first, each table's rows
// in `(Target, Definition, name)` order.
//
// **Rows sort by the identity and never by the change name.** A rendering that
// puts the destructions first ranks its own rows, which is the one thing only
// `FLAGS` may do (ADR-0054), and it makes the eye read the `CHANGE` column
// twice. Ordering by `written_at` is refused because the laptop and the runner
// do not share a clock (§7) — a refusal this table can afford because it ranges
// over identities and has a name axis to order on instead.
//
// **The three tables never join an Observation series to an Asset series**,
// which is the drift detection `hyper` has no engine for (ADR-0010): a row's
// table is the record type of the version standing at it, and no identity
// reaches both.
func changeRowsOf(window Window, records []Record) []render.Row {
	sorted := slices.Clone(records)
	slices.SortFunc(sorted, func(a, b Record) int { return byIdentity(a.Identity, b.Identity) })

	baseline, subject := readingsOf(window)
	var assets, observations []render.Row
	for _, record := range sorted {
		row, drawn := changeRowOf(record, baseline, subject)
		if !drawn {
			continue
		}
		if row.Type == string(store.RecordAsset) {
			assets = append(assets, row)
			continue
		}
		observations = append(observations, row)
	}
	return append(assets, observations...)
}

// changeRowOf reads one identity's row off its two endpoints, and answers that
// no row is drawn where nothing moved.
func changeRowOf(record Record, baseline, subject reading) (ChangeRow, bool) {
	kind, known := recordTypeOf(record)
	if !known {
		// Neither end holds a version, so there is nothing for a table
		// to be about: an identity a Step concluded about and no Run
		// ever minted is a Record the Store does not hold.
		return ChangeRow{}, false
	}

	change, drawn := ChangeChanged, false
	if kind == store.RecordAsset {
		change, drawn = assetChange(record)
	} else {
		change, drawn = observationChange(record, baseline, subject)
	}
	if !drawn {
		return ChangeRow{}, false
	}

	row := ChangeRow{
		Type:       string(kind),
		Change:     string(change),
		Target:     record.Identity.Target,
		Definition: record.Identity.Definition,
		Name:       record.Identity.Name,
		Fields:     fieldsOf(change, record),
	}
	row.FromOrdinal, row.ToOrdinal = ordinalsOf(change, record)
	if change == ChangeDestroyed {
		// The Tombstone's own `written_at`, which §7 states is when
		// destruction was confirmed.
		row.ConfirmedAt = store.InstantText(record.Subject.Version.WrittenAt)
	}
	return row, true
}

// recordTypeOf is which of the two tables a row belongs to, read off the
// version standing at an end of the window rather than off the Step that
// concluded about it.
//
// The version is the fact. A Step's Kind says what it was permitted to do and
// the version says what was written, and the second is what a table of Records
// is a table of (§7, ADR-0025).
func recordTypeOf(record Record) (store.RecordType, bool) {
	switch {
	case record.Subject.Held:
		return record.Subject.Version.RecordType, true
	case record.Baseline.Held:
		return record.Baseline.Version.RecordType, true
	}
	return "", false
}

// assetChange is the Asset table's three names, which are read from the
// Records alone.
//
// **A series whose first version is a Tombstone renders `destroyed` and never
// `created`**, though the baseline holds no version of it and the subject does:
// what the subject holds is a destruction, and reading *absent, then present*
// as a creation would report the opposite of what happened (§7, ADR-0033). It
// needs no marker to be told apart — an ordinary `destroyed` row carries the
// last known state's fields and this one has none.
//
// The Asset table has no `appeared` and no `vanished`, so an identity that left
// this Procedure's sight with its version standing draws no row: `changed`
// there would assert that the baseline Run had this in view and saw something
// else, which is the one thing that is false.
func assetChange(record Record) (Change, bool) {
	if !record.Subject.Held {
		return "", false
	}
	stood := record.Baseline.Held && record.Baseline.Version.File == record.Subject.Version.File
	switch {
	case stood:
		return "", false
	case record.Subject.Version.Tombstone:
		return ChangeDestroyed, true
	case !record.Baseline.Held:
		return ChangeCreated, true
	}
	return ChangeChanged, true
}

// observationChange is the Observation table's three names.
//
// *Vanished*, *appeared* and *nothing moved* are derived from the identity sets
// each Step's Disposition carries rather than from the Records, which is what
// buys a disappearance a row at all: an unchanged Record and a Record that
// stopped existing both write nothing (§7, ADR-0058).
//
// **Where `appeared` and `changed` are both true, `appeared` wins** — an
// identity absent from the baseline Run's set, present in the subject's, whose
// series holds an older version that has since moved. The Disposition-derived
// name beats the Record-derived one, which is one precedence rule rather than a
// per-pair one and never arises in the Asset table.
//
// **A silence takes away the Disposition-derived name and never the
// Record-derived one.** §8 says an identity missing from a partial set draws no
// row *where it would otherwise render vanished*, and no row as baseline *where
// the same identity returning would otherwise render appeared* — those two
// names and no third. A Record whose endpoints differ still differs from when
// this Procedure last looked, whoever moved it, so the row falls through to
// `changed` rather than being dropped: reading the endpoints is what ADR-0058
// makes eligibility and naming two separate questions for.
func observationChange(record Record, baseline, subject reading) (Change, bool) {
	inBaseline, heldByBaseline := baseline.holds(record.Identity)
	inSubject, heldBySubject := subject.holds(record.Identity)

	switch {
	case heldByBaseline && !heldBySubject && record.Baseline.Held && !subject.silent(inBaseline):
		return ChangeVanished, true
	case heldBySubject && !heldByBaseline && record.Subject.Held && !baseline.silent(inSubject):
		return ChangeAppeared, true
	case record.Baseline.Held && record.Subject.Held && record.Baseline.Version.File != record.Subject.Version.File:
		return ChangeChanged, true
	}
	return "", false
}

// ordinalsOf is the row's two ordinals, and nothing for the side that has
// nothing for this row to name.
//
// **`–` means this side has nothing for this row to name.** On an Asset row
// that is a side holding no version: it repeats what `CHANGE` already says on a
// `created` row, and earns its place on the `destroyed` row of a series whose
// first version is a Tombstone, where `– → 1` reads as *`hyper` ended a thing
// it never built*.
//
// On an `appeared` or a `vanished` row what the side lacks is a **view** rather
// than a version, so they render `– → n` and `n → –`: an identity that left
// this Procedure's sight and returned unchanged has a version standing on both
// sides, and printing the standing ordinal on the side the Procedure could not
// see puts two numbers on a row and invites a reader to difference them across
// exactly the boundary the row exists to report.
//
// **Nothing marks a gap and nothing counts what a window hides.** Both would be
// sound in the Asset table, Compaction never removing an Asset version, and
// unsound in the Observation table beside it, which is two guarantees under one
// column head (§8).
func ordinalsOf(change Change, record Record) (from, to *int) {
	if change != ChangeVanished && record.Subject.Held {
		to = ordinal(record.Subject.Version.Ordinal)
	}
	if change != ChangeAppeared && record.Baseline.Held {
		from = ordinal(record.Baseline.Version.Ordinal)
	}
	return from, to
}

// ordinal is one ordinal as the row carries it. It is a pointer because the
// member's absence is the fact the column renders `–` for, and `0` is not an
// ordinal (§7).
func ordinal(position int) *int { return &position }

// byIdentity orders two identities as §7 orders an identity set's members and
// §6 orders an Expansion: Target, then Definition, then name, each by Unicode
// code point, the columns read left to right.
//
// It is that rule reused rather than reinvented, so two renderings of one
// window are byte-identical and diffable (§8, ADR-0044).
func byIdentity(a, b store.Identity) int {
	return cmp.Or(
		strings.Compare(a.Target, b.Target),
		strings.Compare(a.Definition, b.Definition),
		strings.Compare(a.Name, b.Name),
	)
}

// ChangeRow is one row of `YOU DID THIS` or of `THE WORLD MOVED`, and its
// `type` is which of the two.
//
// **One type carries both tables** because §8 gives them one derivation: they
// differ in their three change names and in `confirmed_at`, and a second
// declaration would be the same members in the same order written twice, free
// to drift in either. The members are §8's own, in §8's order.
//
// **`from_ordinal` and `to_ordinal` are absent exactly where the column renders
// `–`** — §7's absence rule saying *nothing to name on this side* by writing no
// key, which is what a `vanished` row carries in place of a subject ordinal.
//
// **`confirmed_at` stands on a `destroyed` row and nowhere else.** It is the
// Tombstone's own `written_at`, which §7 states is when destruction was
// confirmed, and it is the instant the page renders as `† confirmed 11:02`.
//
// **`fields` carries every value whole**, including the ones the page rendered
// `changed` (ADR-0059): the elision is that column's geometry and never a fact
// either surface states. It is written always, the empty mapping included,
// because an empty one is what says *`hyper` destroyed this and never observed
// what it was* on the one row that can hold it (§7, ADR-0033).
type ChangeRow struct {
	Type        string   `json:"type"`
	Change      string   `json:"change"`
	Target      string   `json:"target"`
	Definition  string   `json:"definition"`
	Name        string   `json:"name"`
	FromOrdinal *int     `json:"from_ordinal,omitempty"`
	ToOrdinal   *int     `json:"to_ordinal,omitempty"`
	ConfirmedAt string   `json:"confirmed_at,omitempty"`
	Fields      FieldSet `json:"fields"`
}

// Cells is the row's line on its table, in that table's column order:
// `CHANGE`, `TARGET`, `DEFINITION`, `RECORD`, `ORDINAL`, `FIELDS`.
//
// Every cell is derived from the row's own members, so the page cannot render a
// fact the stream did not carry (ADR-0026). The two that render differently
// here are the ordinal, whose absent side is a `–` where the wire writes no
// key, and `FIELDS`, whose elision is this column's geometry (ADR-0059).
func (r ChangeRow) Cells() []string {
	return []string{
		r.Change,
		r.Target,
		r.Definition,
		r.Name,
		ordinalCell(r.FromOrdinal, r.ToOrdinal),
		r.Fields.cell(r.confirmed()),
	}
}

// confirmed is the `† confirmed <time>` marker a `destroyed` row opens its
// `FIELDS` cell with, and the empty string on every other row.
//
// The time is the hour and the minute, which is the grain a reader compares two
// destructions at on one page; the instant whole is on the wire beside it, one
// fact in the two notations. It is read back off the row's own member so that
// the page cannot render an instant the stream did not carry, and an instant
// that would not parse renders itself rather than a gap.
func (r ChangeRow) confirmed() string {
	if r.ConfirmedAt == "" {
		return ""
	}
	instant, err := time.Parse(time.RFC3339, r.ConfirmedAt)
	if err != nil {
		return tombstoneMark + " confirmed " + r.ConfirmedAt
	}
	return tombstoneMark + " confirmed " + instant.UTC().Format("15:04")
}

// ordinalCell is the `ORDINAL` column: two positions with an arrow between
// them, and `–` for the side with nothing to name.
func ordinalCell(from, to *int) string {
	return ordinalText(from) + " " + fieldArrow + " " + ordinalText(to)
}

// ordinalText is one side of the ordinal cell.
func ordinalText(position *int) string {
	if position == nil {
		return sideNothing
	}
	return strconv.Itoa(*position)
}
