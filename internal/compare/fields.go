package compare

import (
	"bytes"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// `FIELDS` (§8, issue #170, ADR-0059). **A value renders whole or renders
// `changed`**, and there is no truncated form: two values agreeing for their
// first hundred characters and diverging after would render as identical bytes
// on a row asserting they differ.

const (
	// valueBudget is how many characters a scalar may occupy and still
	// render whole. It is a **stated constant and not the terminal's
	// width**: colour and width are the only differences between an
	// interactive rendering and a piped one, and a width-derived budget
	// makes the two disagree about content (§8, ADR-0059).
	//
	// Guessing at a number is affordable here in the way ADR-0045 said it
	// was not for the Store: a wrong Store limit discards evidence
	// irrecoverably, where a wrong rendering budget costs a `changed` where
	// a value would have fitted, and the value is one `hyper show` away.
	valueBudget = 120
	// valueChanged is what a value the budget disqualified renders as on a
	// two-sided row. It is the whole of what the page can honestly carry
	// there, and `--json` carries the value regardless.
	valueChanged = "changed"
	// fieldGap separates two fields of one cell, and the `† confirmed`
	// marker from the fields behind it. It is the notation the Comparison
	// renders a run of values in everywhere.
	fieldGap = " · "
	// fieldArrow stands between the two sides of a value that moved and
	// between the two sides of the ordinal.
	fieldArrow = "→"
	// sideNothing is a side with nothing: the ordinal of an end holding no
	// version, and the value of a field one end did not carry.
	sideNothing = "–"
	// tombstoneMark opens a `destroyed` row's cell, the dagger being the
	// notation §8 renders a confirmed destruction in.
	tombstoneMark = "†"
)

// FieldChange is one field of a row's `FIELDS`: the path, and what stood at
// each end of the window.
//
// A side that carried no such field holds nil, which is the ordinary absence:
// a field whose path resolved to nothing is not in the version at all, and the
// field not being written is what carries that (§6, §7).
type FieldChange struct {
	Path string
	From store.Value
	To   store.Value
}

// value is the one value a one-sided row's field holds. A one-sided row has
// exactly one end, so whichever member is filled is the whole of it.
func (f FieldChange) value() store.Value {
	if f.To != nil {
		return f.To
	}
	return f.From
}

// FieldSet is a row's `FIELDS` as both surfaces read it: the fields it renders,
// **sorted by Unicode code point**, and which of the two shapes the wire writes
// them in.
//
// The order is the Store's own canonical encoding read out rather than a second
// ordering, and there is **no cap on how many render**: a wide cell is a
// Manifest author's projection choice rendered honestly, and capping it is
// `hyper` guessing at a number in the way ADR-0045 declined (§8).
//
// Paired is the two-sided shape — `{"path":[from,to]}` — and is what a
// `changed` row carries. A one-sided row carries `{"path":value}`, there being
// no other side and the cell describing what the thing is rather than how it
// differs.
type FieldSet struct {
	Paired bool
	Fields []FieldChange
}

// MarshalJSON writes the fields as one object, in the order they are held.
//
// It is assembled rather than handed to a map because the order is the fact:
// two renderings of one window are byte-identical and diffable, which a member
// whose order came out of a Go map iteration would not be (§8).
//
// **Every value goes out whole**, the ones the page rendered `changed`
// included: the elision is the column's geometry and never a fact either
// surface states (ADR-0059). A value the Store holds nested goes out as the
// artefact's own parsed shape rather than as anything the page composed.
//
// On a `changed` row a side that carried no such field goes out as `null`. It
// is the one place on this wire an absence is written rather than left out:
// §7's rule is stated over a **key**, and a two-element array has no key to
// omit — the array being §8's own shape for the pair. `null` there is the same
// *nothing on this side* the page renders `–` for.
func (s FieldSet) MarshalJSON() ([]byte, error) {
	var written bytes.Buffer
	written.WriteByte('{')
	for i, field := range s.Fields {
		if i > 0 {
			written.WriteByte(',')
		}
		written.Write(store.Unframed(store.String(field.Path)))
		written.WriteByte(':')
		if !s.Paired {
			written.Write(wireValue(field.value()))
			continue
		}
		written.WriteByte('[')
		written.Write(wireValue(field.From))
		written.WriteByte(',')
		written.Write(wireValue(field.To))
		written.WriteByte(']')
	}
	written.WriteByte('}')

	// The canonical encoding indents and breaks lines, and this wire is
	// compact — no space after a separator (§8). The two encodings are
	// different shapes of one value, and this is the door between them
	// (store.Unframed).
	var compact bytes.Buffer
	if err := json.Compact(&compact, written.Bytes()); err != nil {
		return nil, err
	}
	return compact.Bytes(), nil
}

// wireValue is one value as this stream carries it, and `null` where the side
// carried none.
func wireValue(value store.Value) []byte {
	if value == nil {
		return []byte("null")
	}
	return store.Unframed(value)
}

// cell is the `FIELDS` column's text, opened by the marker a `destroyed` row
// carries and empty where the row renders neither.
//
// **`FIELDS` follows the change name.** A `changed` row renders the fields that
// moved, `path: old → new`. A `created` or `appeared` row renders every
// projected field of the one version, `path: value`, with no arrow — there is
// no other side, and the cell is describing what the thing is rather than how
// it differs. A `destroyed` row renders `† confirmed <time>` and then the last
// known fields off the Tombstone itself. A `vanished` row renders the baseline
// end's fields with no marker: a row saying a thing stopped being there while
// declining to say what it was leaves its reader no other page to go to (§8).
func (s FieldSet) cell(marker string) string {
	parts := make([]string, 0, len(s.Fields)+1)
	if marker != "" {
		parts = append(parts, marker)
	}
	for _, field := range s.Fields {
		parts = append(parts, s.fieldText(field))
	}
	return strings.Join(parts, fieldGap)
}

// fieldText is one field of the cell.
//
// **A value renders whole or renders `changed`** (ADR-0059). A scalar over the
// stated budget, one carrying a newline, and anything nested are one class with
// one rendering: `path: changed` on a two-sided row, and the bare `path` on a
// one-sided one, where `changed` would be false and the field's name is the
// whole of what the page can honestly carry.
//
// On a two-sided row it is the **pair** that is disqualified, not one half of
// it: a cell rendering one side whole and the other as a word would read as a
// value that moved to nothing.
func (s FieldSet) fieldText(field FieldChange) string {
	if !s.Paired {
		text, renderable := valueText(field.value())
		if !renderable {
			return field.Path
		}
		return field.Path + ": " + text
	}
	from, fromRenderable := sideText(field.From)
	to, toRenderable := sideText(field.To)
	if !fromRenderable || !toRenderable {
		return field.Path + ": " + valueChanged
	}
	return field.Path + ": " + from + " " + fieldArrow + " " + to
}

// sideText is one side of a two-sided field, and `–` where that side carried no
// such field at all — the same *nothing to name on this side* the ordinal
// column renders.
func sideText(value store.Value) (string, bool) {
	if value == nil {
		return sideNothing, true
	}
	return valueText(value)
}

// valueText is a value as the page writes it, and whether the page may write it
// at all.
//
// The three disqualifications are one class: a scalar over the budget, one
// carrying a newline, and anything nested. **The newline test is absolute
// rather than a length in disguise**, because the built-in `shell` Provider
// projects `stdout` and `stderr` as unparsed text with no cap between a chatty
// command and the Store, so a short two-line value would otherwise be free to
// rewrite its own table's geometry (ADR-0052, ADR-0059).
//
// A scalar renders as the value and never as its encoding: a String writes its
// text without the quotes the canonical form carries, the page being read by an
// eye rather than parsed. Everything the Store holds nested — a mapping, an
// array — is disqualified by construction.
func valueText(value store.Value) (string, bool) {
	var text string
	switch value := value.(type) {
	case store.String:
		text = string(value)
	case store.Number:
		text = value.Text()
	case store.Bool:
		text = strconv.FormatBool(bool(value))
	default:
		return "", false
	}
	if strings.ContainsRune(text, '\n') || utf8.RuneCountInString(text) > valueBudget {
		return "", false
	}
	return text, true
}

// fieldsOf is the fields a row renders, in code-point order.
//
// A `changed` row renders **the fields that moved** and nothing else, which is
// what the two-sided cell is for; every other name renders every projected
// field of the one version it is about.
func fieldsOf(change Change, record Record) FieldSet {
	switch change {
	case ChangeChanged:
		return FieldSet{Paired: true, Fields: movedFields(record.Baseline.Fields, record.Subject.Fields)}
	case ChangeVanished:
		// The baseline end's, the Store holding no version minted for a
		// disappearance (§8).
		return FieldSet{Fields: wholeFields(record.Baseline.Fields)}
	}
	// `created`, `appeared` and `destroyed` all render the subject end's
	// version — on a `destroyed` row the Tombstone's own fields, which
	// copied the previous Head's forward, so a Record changed and then
	// destroyed in one window shows the state it was in when it ended (§7).
	return FieldSet{Fields: wholeFields(record.Subject.Fields)}
}

// wholeFields is every field of one version, in code-point order.
func wholeFields(fields store.Mapping) []FieldChange {
	whole := make([]FieldChange, 0, len(fields))
	for path, value := range fields {
		whole = append(whole, FieldChange{Path: path, To: value})
	}
	return sortedFields(whole)
}

// movedFields is the fields whose value differs between the two ends, in
// code-point order.
//
// The comparison is the canonical encoding's — the same *the bytes moved* test
// that decides whether a version was minted at all (§7) — rather than a second
// reading of what a value is. A field one end carried and the other did not has
// moved: an absent field is the ordinary absence §6 states, and a value
// arriving or going away is a difference the row exists to report.
func movedFields(from, to store.Mapping) []FieldChange {
	paths := map[string]bool{}
	for path := range from {
		paths[path] = true
	}
	for path := range to {
		paths[path] = true
	}

	moved := make([]FieldChange, 0, len(paths))
	for path := range paths {
		before, after := from[path], to[path]
		if !differs(before, after) {
			continue
		}
		moved = append(moved, FieldChange{Path: path, From: before, To: after})
	}
	return sortedFields(moved)
}

// differs answers whether two values moved, an absent value on either side
// being a difference from any value at all.
func differs(before, after store.Value) bool {
	switch {
	case before == nil && after == nil:
		return false
	case before == nil || after == nil:
		return true
	}
	return !bytes.Equal(store.Unframed(before), store.Unframed(after))
}

// sortedFields puts a cell's fields in Unicode code point order, which is the
// Store's own key order read out rather than a second ordering (§7, §8).
func sortedFields(fields []FieldChange) []FieldChange {
	slices.SortFunc(fields, func(a, b FieldChange) int { return strings.Compare(a.Path, b.Path) })
	return fields
}
