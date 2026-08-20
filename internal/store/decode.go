package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"
)

// The read half. A Store file is decoded to the shape it was written from and
// to nothing else: the five shapes are closed, their keys are closed, and a
// file carrying something outside them is a file this package did not write
// (§7).
//
// Two rules do most of the work. The **schema version is read first**, and a
// file above the reader's ceiling answers SchemaUnsupported rather than a guess
// or a partial value (ADR-0028) — checked before the keys, since a later
// shape's file is expected to carry keys this reader does not know. And the
// bytes are **re-encoded and compared** as the last act of every decode: a
// decode that could not write back what it read has read a file the canonical
// encoding did not produce, and the Store holds no other kind.

// decodeFile reads one Store file. It parses the bytes, checks the version
// against this reader's ceiling, hands the mapping to the shape's own reader,
// and then holds the result against the bytes it came from.
//
// It is one function over the five shapes rather than five copies of the same
// five steps, which is what keeps the version check from being the one a shape
// forgets.
func decodeFile[T interface{ Encode() []byte }](data []byte, known int, read func(*fields, *T)) (T, error) {
	var zero, shape T

	value, err := parse(data)
	if err != nil {
		return zero, err
	}
	mapping, ok := value.(Mapping)
	if !ok {
		return zero, fmt.Errorf("a Store file is a JSON object")
	}

	f := newFields(mapping, nil)
	version := f.count("schema_version")
	f.require("schema_version")
	if f.err != nil {
		return zero, f.err
	}
	if version < 1 {
		return zero, fmt.Errorf("%d is not a schema version: every shape's integer starts at 1", version)
	}
	if version > known {
		return zero, SchemaUnsupported{Written: version, Known: known}
	}

	read(f, &shape)
	f.closed()
	if f.err != nil {
		return zero, f.err
	}

	// The canonical check: what came back writes the bytes that went in, or
	// the file was not written by this encoding. It catches what no
	// key-by-key read can — a duplicated key, an escaped umlaut, a number
	// spelled another way, an indent that moved — on a branch where a
	// version is minted wherever the bytes did.
	//
	// The cost is stated rather than hidden: every read of a Store file
	// pays for an encode of it, which roughly doubles what a decode costs
	// and is paid once per file on a surface that scans many. What it buys
	// is that *the Store holds canonical files* is checked rather than
	// assumed — and every rule §7 states about the encoding is enforced at
	// the read by the code that writes it, so there is no second reader to
	// disagree with the encoder about what canonical means.
	if written := shape.Encode(); !bytes.Equal(written, data) {
		return zero, fmt.Errorf("the file is not in the canonical encoding: it re-encodes to %d bytes and was read from %d", len(written), len(data))
	}
	return shape, nil
}

// parse reads canonical JSON to a Value. It is the door every decoded value
// comes through, and what it will not admit is null: a field's presence is a
// fact stated by its presence, and there is no null anywhere in the Store (§7,
// §12).
func parse(data []byte) (Value, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("the file is not JSON: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("the file carries a second value after the first")
	}
	return convert(raw)
}

// convert turns what encoding/json handed back into this package's own value
// model, which is the closed set the encoder writes and nothing wider.
func convert(raw any) (Value, error) {
	switch raw := raw.(type) {
	case map[string]any:
		mapping := make(Mapping, len(raw))
		for key, value := range raw {
			converted, err := convert(value)
			if err != nil {
				return nil, fmt.Errorf("at %q: %w", key, err)
			}
			mapping[key] = converted
		}
		return mapping, nil
	case []any:
		array := make(Array, len(raw))
		for i, value := range raw {
			converted, err := convert(value)
			if err != nil {
				return nil, fmt.Errorf("at element %d: %w", i, err)
			}
			array[i] = converted
		}
		return array, nil
	case string:
		return String(raw), nil
	case bool:
		return Bool(raw), nil
	case json.Number:
		return ParseNumber(raw.String())
	}
	return nil, fmt.Errorf("the Store holds no null")
}

// fields is one mapping being read back into a shape. It answers each member,
// remembers which ones were asked for, and carries the first fault rather than
// the last, so a shape's reader is a list of its members instead of a list of
// its members interleaved with error checks.
//
// Remembering what was asked for is what closes the shape: a key nobody asked
// for is a key this shape does not have, and a file carrying one was written by
// something else — or by a later version of this shape, which the version check
// has already turned away by the time any of this runs.
type fields struct {
	mapping Mapping
	taken   map[string]bool
	err     error
}

// newFields opens a reader over one mapping, carrying whatever fault the file
// has already found: a block's fault is the file's, and a reader that started
// clean inside a file that had not would report the second fault as the first.
func newFields(mapping Mapping, err error) *fields {
	return &fields{mapping: mapping, taken: map[string]bool{}, err: err}
}

// fault records the first thing wrong with the file. Later ones are dropped:
// a decode reports what a reader must fix, and the first fault is usually the
// cause of the rest.
func (f *fields) fault(format string, args ...any) {
	if f.err == nil {
		f.err = fmt.Errorf(format, args...)
	}
}

// take answers a member and marks it read, whether it is there or not — asking
// for a key the file does not carry is still this shape asking for it.
func (f *fields) take(key string) Value {
	f.taken[key] = true
	return f.mapping[key]
}

// require faults where a member the shape always carries is absent. Each shape
// states its mandatory set in one line, which reads as the shape's own rule
// rather than as a flag repeated at every getter.
func (f *fields) require(keys ...string) {
	for _, key := range keys {
		if _, present := f.mapping[key]; !present {
			f.fault("the file carries no %q", key)
		}
	}
}

// closed faults where the file carries a member this shape never asked for.
func (f *fields) closed() {
	var unknown []string
	for key := range f.mapping {
		if !f.taken[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		slices.Sort(unknown)
		f.fault("the file carries %q, which this shape does not have", unknown[0])
	}
}

// member answers one member of the file at one of the encoder's own value
// types, and that type's zero value where the file carries none. A member of
// another type is a fault: the shapes are closed, and so is what each key
// holds.
//
// It is one function over the five types the shapes read rather than five
// copies of one type switch, which is what keeps a fault message from being the
// one a type spells differently.
func member[T Value](f *fields, key, what string) T {
	var zero T
	value := f.take(key)
	if value == nil {
		return zero
	}
	typed, ok := value.(T)
	if !ok {
		f.fault("%q is not %s", key, what)
		return zero
	}
	return typed
}

// text answers a string member, and the empty string where the file carries
// none — the same absence the encoder's own text writes nothing for.
func (f *fields) text(key string) string {
	return string(member[String](f, key, "a string"))
}

// count answers an integer member, and zero where the file carries none. A
// Number the file does not carry holds no literal, and one it does always
// holds one, so the two are told apart without asking twice.
func (f *fields) count(key string) int {
	number := member[Number](f, key, "a number")
	if number.literal == "" {
		return 0
	}
	count, err := strconv.Atoi(number.literal)
	if err != nil {
		f.fault("%q is %s, which is not a whole number", key, number.literal)
		return 0
	}
	return count
}

// position answers a 1-indexed member — a Step's place in a Run's written
// order, a line in a file — and faults where the file carries one below it. The
// zero is the absence, which the encoder writes nothing for, so a number below
// one is a number this package did not write.
func (f *fields) position(key string) int {
	count := f.count(key)
	if f.carries(key) && count < 1 {
		f.fault("%q is %d, and a position starts at 1", key, count)
	}
	return count
}

// answer reads an integer that may not have arrived, which is count with the
// zero admitted: 0 is an exit code a command really gives.
func (f *fields) answer(key string) Answer {
	if _, present := f.mapping[key]; !present {
		f.take(key)
		return Answer{}
	}
	return Arrived(f.count(key))
}

// at answers an instant, and the zero Time where the file carries none. The
// layout admits an offset so that a file written with one is read as the
// instant it names — and then fails the canonical check, which is where a
// timestamp outside §7's one form is refused rather than quietly re-based.
func (f *fields) at(key string) time.Time {
	text := f.text(key)
	if text == "" {
		return time.Time{}
	}
	instant, err := time.Parse("2006-01-02T15:04:05.000Z07:00", text)
	if err != nil {
		f.fault("%q is %q, which is not RFC 3339 with milliseconds to three digits", key, text)
		return time.Time{}
	}
	return instant.UTC()
}

// mark answers a boolean member, and false where the file carries none.
func (f *fields) mark(key string) bool {
	return bool(member[Bool](f, key, "a boolean"))
}

// value answers a member the shape holds as an arbitrary value: a Record's
// projected content, a selector as authored, the two values a check compared.
func (f *fields) value(key string) Value { return f.take(key) }

// nested answers a member the shape holds as a mapping of arbitrary content,
// and nil where the file carries none.
func (f *fields) nested(key string) Mapping {
	return member[Mapping](f, key, "a mapping")
}

// names answers a list of strings, nil where the file carries none and the
// empty slice where it carries the empty list. The two are told apart because
// on two members the difference is the whole point (§7).
func (f *fields) names(key string) []string {
	array, present := f.array(key)
	if !present {
		return nil
	}
	names := make([]string, len(array))
	for i, element := range array {
		text, ok := element.(String)
		if !ok {
			f.fault("%q holds something that is not a string", key)
			return nil
		}
		names[i] = string(text)
	}
	return names
}

// array answers a list member and whether the file carries one. Presence is
// asked before the value, an empty list being a value the shapes mean.
func (f *fields) array(key string) (Array, bool) {
	present := f.carries(key)
	return member[Array](f, key, "a list"), present
}

// block answers a nested mapping as a reader of its own, so that the members
// inside it are closed exactly as the file's own are. It answers nil where the
// file carries no such block.
func (f *fields) block(key string) *fields {
	mapping := f.nested(key)
	if mapping == nil {
		return nil
	}
	return newFields(mapping, f.err)
}

// join folds a nested reader's fault back into this one once its members have
// been read, a block's fault being the file's.
func (f *fields) join(nested *fields, key string) {
	if nested == nil {
		return
	}
	nested.closed()
	if nested.err != nil && f.err == nil {
		f.err = fmt.Errorf("in %q: %w", key, nested.err)
	}
}

// oneOf answers a member of a closed set: the spelling the file carries,
// checked against the members the Store holds and no others. An absent key
// answers the zero value — whether the shape carries the member at all is
// require's question and not this one's.
func oneOf[T ~string](f *fields, key string, values ...T) T {
	text := f.text(key)
	if text == "" {
		return ""
	}
	for _, value := range values {
		if string(value) == text {
			return value
		}
	}
	f.fault("%q is %q, which is outside the closed set", key, text)
	return ""
}

// carries answers whether the file holds a member, without asking for it. Two
// shapes branch on which of two member sets they were written with, and a
// branch is not a read.
func (f *fields) carries(key string) bool {
	_, present := f.mapping[key]
	return present
}

// run answers a Run id member, checked as one. An id that reached a file
// unchecked would name a directory nothing could find again by looking for the
// id it was told (§12).
func (f *fields) run(key string) RunID {
	text := f.text(key)
	if text == "" {
		return RunID{}
	}
	id, err := ParseRunID(text)
	if err != nil {
		f.fault("%q: %s", key, err)
		return RunID{}
	}
	return id
}
