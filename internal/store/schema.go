package store

import (
	"fmt"
	"time"
)

// The five schema versions, one per shape the Store holds — and five rather
// than one because a shape's version says that *that* shape moved (§7,
// ADR-0028). One integer across the Store would move a Record version's number
// when a Step file's shape moved, and an older binary would then Refuse a
// Record file it could read perfectly.
//
// Each starts at 1 and each moves alone. They are five constants rather than
// one array or one map because five independent integers that could be indexed
// by a shape are five integers with a place to be moved together from.
//
// `STORE.md` carries none, being prose written once.
const (
	// RecordSchemaVersion is a Record version's, Tombstones included: a
	// Tombstone is an ordinary version of the series and not a shape of its
	// own (§7).
	RecordSchemaVersion = 1
	// RunSchemaVersion is run.json's.
	RunSchemaVersion = 1
	// StepSchemaVersion is a Step file's.
	StepSchemaVersion = 1
	// OutcomeSchemaVersion is outcome.json's.
	OutcomeSchemaVersion = 1
	// ClosedBySchemaVersion is a closed-by/ file's.
	ClosedBySchemaVersion = 1
)

// SchemaUnsupportedCode is the error_code a Run Refuses with where a file it
// must read was written in a shape above the one this binary knows (§12).
const SchemaUnsupportedCode = "store-schema-unsupported"

// SchemaUnsupported is the condition a decode answers instead of a guess: the
// file says it was written in a shape this binary does not know, and reading it
// anyway is reading a shape nobody defined (ADR-0028).
//
// It is a condition rather than a Refusal because the decoder is not where a
// Run declines. This package holds no Run, renders no terminal line and knows
// no path: the caller read the file, so the caller names it and the caller
// renders the row, carrying SchemaUnsupportedCode into it.
type SchemaUnsupported struct {
	// Written is the version the file carries, and Known the highest this
	// binary reads. Written is always above Known: at or below it, the file
	// decodes.
	Written, Known int
}

func (e SchemaUnsupported) Error() string {
	return fmt.Sprintf("the file was written at schema version %d and this hyper reads %d", e.Written, e.Known)
}

// members is a file's mapping under construction. Every setter drops a member
// the file does not carry rather than writing it empty, so the absence rule is
// stated once here instead of at each of the five shapes' every key (§7).
//
// It is the shapes' own rule and not the encoder's: canonical.go drops an empty
// mapping and an empty list, which is what a *value* being empty means, and
// this is what a *file* not carrying a member means. The two meet on a Record's
// fields, where both agree.
type members Mapping

// text writes a string member, and writes nothing where it is empty. No member
// of any of the five shapes carries the empty string as a value it means, so
// there is no case where this drops one that was intended.
func (m members) text(key, value string) {
	if value != "" {
		m[key] = String(value)
	}
}

// count writes an integer member, and writes nothing where it is zero. Every
// counted thing in the Store starts at one — a Step's position, a Pattern's
// attempts, a Bound, a `line` — so zero is the absence and never a value.
func (m members) count(key string, value int) {
	if value != 0 {
		m[key] = Int(int64(value))
	}
}

// at writes an instant, and writes nothing at the zero one. A reaper's file
// carries no `started_at` and a Step that never began has no instant to write,
// so the zero Time is *hyper did not observe this* rather than the epoch.
func (m members) at(key string, value time.Time) {
	if !value.IsZero() {
		m[key] = Timestamp(value)
	}
}

// mark writes a boolean member where it is true and nothing where it is false.
// The two that reach it — `tombstone` and `repo_dirty` — are markers rather than
// fields: each says a thing is so, and its absence says the ordinary case (§7).
// `dry_run` is not one of them and does not come through here.
func (m members) mark(key string, value bool) {
	if value {
		m[key] = Bool(true)
	}
}

// answer writes an integer answer, and writes nothing where none arrived. It is
// count with the zero admitted: 0 is a shell exit code a command really gives,
// so the absence is carried by the Answer rather than by the value.
func (m members) answer(key string, value Answer) {
	if value.arrived {
		m[key] = Int(int64(value.code))
	}
}

// value writes a value a caller supplied, and writes nothing where there is
// none. An empty mapping or list reaching here is dropped by the encoder on the
// absence rule, which is the same answer arrived at one layer down.
func (m members) value(key string, value Value) {
	if value != nil {
		m[key] = value
	}
}

// block writes a nested mapping built by fill, and writes nothing where fill
// filled none of it — the absence rule applied to a member that is itself a
// file's worth of members.
func (m members) block(key string, fill func(members)) {
	nested := members{}
	fill(nested)
	if len(nested) > 0 {
		m[key] = Mapping(nested)
	}
}

// namesArray is a list of names as the Store writes one: one string per
// element, in the order it was handed, which is Expansion order on a selector
// and code point order on an identity set.
func namesArray(values []string) Array {
	array := make(Array, len(values))
	for i, value := range values {
		array[i] = String(value)
	}
	return array
}

// require holds a file to the members its shape always carries, and is the
// encoder's half of a rule the decoder states in the same words over the same
// list. One list read from both directions is what makes the round trip total:
// a shape that could be written without a member the read requires is a shape
// whose own bytes it would refuse.
//
// It is impossible rather than an error for the reason paths.go's is. None of
// these arrive from the world — a Run's Procedure, a Step's Kind and a
// version's identity are hyper's own facts — and what a caller would otherwise
// be handed is a file already on its way into an append-only branch.
func (m members) require(subject string, keys []string) {
	for _, key := range keys {
		if value, written := m[key]; !written || omitted(value) {
			impossible("%s carries no %q, which every one of them carries (§7)", subject, key)
		}
	}
}

// file writes one of the five shapes: the shape's own members, and the schema
// version beside them, which is the one key every file in the Store carries and
// no shape restates for itself.
func file(version int, subject string, required []string, fill func(members)) []byte {
	m := members{"schema_version": Int(int64(version))}
	fill(m)
	m.require(subject, required)
	return Encode(Mapping(m))
}
