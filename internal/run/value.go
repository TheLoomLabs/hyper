package run

import (
	"encoding/json"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// A projected value crossing into the record (§7, §12, issue #136).
//
// What comes back off a Capability is whatever the wire held — the shapes
// internal/capability assembles a response object out of, and whatever a JSON
// body parsed to. What a Record version holds is §7's closed set of canonical
// values. This is the one crossing between them, so the mapping is stated once
// and a shape nobody anticipated is answered rather than guessed at.

// stored is a projected value as a Record version holds it, and false where the
// Store has no value for it.
//
// There are exactly two things it answers false about, and both are stated
// rather than defensive. A JSON **null** is one: §12 closes the scalar types at
// eight and states there is no `null` — a field's presence is a fact the
// `exists` and `absent` operators state, never a nullable type — so a path that
// resolved to one has resolved to a value no version can carry. And a Go value
// of a shape this crossing has never heard of is the other, which is a
// Capability having assembled something new: answering false leaves the field
// off the version rather than writing a rendering of a Go type into evidence.
//
// A number keeps its literal rather than passing through a float64, which is
// why there is no `float64` arm at all: internal/capability decodes a body with
// UseNumber, so every number that reaches here is the text the wire carried. An
// integer past a float64's exact range is a Record identity on plenty of
// upstreams, and one that moved under a re-encode would mint a version on every
// Run (§7).
func stored(value any) (store.Value, bool) {
	switch value := value.(type) {
	case string:
		return store.String(value), true
	case bool:
		return store.Bool(value), true
	case int:
		return store.Int(int64(value)), true
	case json.Number:
		number, err := store.ParseNumber(value.String())
		return number, err == nil
	case capability.Object:
		return storedObject(value)
	case map[string]any:
		mapping := store.Mapping{}
		for name, held := range value {
			if stored, holdable := stored(held); holdable {
				mapping[name] = stored
			}
		}
		return mapping, true
	case map[string]string:
		mapping := store.Mapping{}
		for name, held := range value {
			mapping[name] = store.String(held)
		}
		return mapping, true
	case []any:
		array := make(store.Array, 0, len(value))
		for _, held := range value {
			stored, holdable := stored(held)
			if !holdable {
				// An array is a sequence and dropping a member
				// would move every member after it, so a list
				// carrying something the Store cannot hold is
				// not held at all (§7).
				return nil, false
			}
			array = append(array, stored)
		}
		return array, true
	default:
		return nil, false
	}
}

// storedObject is a response object's own ordered mapping as a Record version
// holds it: a mapping, its order forgotten.
//
// The order is deliberately lost. §12 states the response object's members in
// an order because the object is an answer a surface renders (§9, ADR-0017);
// the Store sorts a mapping's keys by code point because a git diff of it is
// read by a human (§7, ADR-0079). They are two encodings with two readers, and
// a projected `$.tls` crossing here becomes the second.
func storedObject(object capability.Object) (store.Value, bool) {
	mapping := store.Mapping{}
	for _, member := range object {
		if held, holdable := stored(member.Value); holdable {
			mapping[member.Name] = held
		}
	}
	return mapping, true
}
