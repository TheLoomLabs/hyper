// Package projection is §12's path grammar resolved (issue #133): a path in
// the closed grammar — `$`, `.member`, `["member"]`, and nothing else —
// evaluated against a Capability's response object, and the projected fields
// an Operation's record: block reads out of one.
//
// internal/artefact already validates that a path is well-formed, which is
// check's half of this: a path is refused at load where its characters are not
// the grammar (§4). This is the other half — where one resolves — and the
// distinction the two halves share nothing about is the one this package
// exists for: **resolved to nothing** is a different answer from **resolved to
// null**. A body carrying `{"error":null}` answers `$.body.error` with a value,
// and a body carrying `{}` answers it with no value at all, and every surface
// above reads the difference: a recorded field resolving to nothing is not
// written on the version (§6, issue #144).
//
// Both cardinalities are here. An Operation of `one` cardinality projects one
// set of fields out of the response object; an Operation of `series`
// cardinality projects many Records out of one response, reading from **two
// roots** — `over:` from the response, and `identity:` and every `fields:` path
// from each member of the collection it named. Both are written `$`, and the
// position decides which root it means: the grammar gains no fourth production
// for it (§3, §12).
//
// What a projection that does **not** resolve does to a Run is
// internal/run/reading.go's, and so is the `projection_failed_path` a halted
// Step's file carries (issue #144). What is here is the resolution itself and
// the two answers it has — *resolved to nothing* and *resolved to null*.
package projection

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// segmentPattern matches one segment of the grammar: `.member` or
// `["member"]`. It is the same grammar internal/artefact refuses a malformed
// path against, written to capture what each segment names rather than to
// answer whether the whole reads — the two questions the two halves ask.
var segmentPattern = regexp.MustCompile(`^(?:\.([A-Za-z_][A-Za-z0-9_-]*)|\["([^"]*)"\])`)

// Projection is what an Operation's record: block projects out of one
// response: its recorded fields, in the order the Manifest authored them.
//
// The order is authored rather than sorted because it is the order the answer
// renders in — a Probe's FIELD/VALUE table is this list — and a Manifest author
// writing `host`, `status`, `days_left` has written the order a reader reads
// them in. Nothing downstream depends on it: the Store sorts a Record version's
// keys by code point like every other mapping it writes (§7, ADR-0079).
type Projection struct {
	Fields []FieldPath
	// Over is the `over:` path naming the collection an Operation of
	// `series` cardinality projects its Records out of, and "" on an
	// Operation of `one` cardinality — the two being told apart by that
	// key's presence and by nothing else (§3, ADR-0037).
	//
	// It roots at the **response** where every path beside it roots at a
	// member of what it named. That is the two roots in one value, and it
	// is why it sits here rather than being read a second time by whoever
	// walks the collection.
	Over string
}

// FieldPath is one entry of a record:'s fields: mapping — the name the Record
// holds the value under, and the path it is read from.
type FieldPath struct{ Name, Path string }

// Field is one projected field: the name, and what its path resolved to. A
// field whose path resolved to nothing is not one of these at all — absence is
// the answer, and it is carried by the field not being here (§6, §7).
type Field struct {
	Name  string
	Value any
}

// Fields is what one response projected to: the recorded fields that
// resolved, in the Manifest's own order.
//
// It is a type of its own for the encoding: what a Probe puts on the wire is
// *the shape a Record would have held* (§9), which is a mapping of field name
// to value, and the order it renders in is the order the page beneath it
// renders — one answer, two surfaces (ADR-0026). encoding/json over a Go map
// would sort the keys and over this slice would write an array of objects, and
// neither is the shape.
type Fields []Field

// MarshalJSON writes the fields as one compact mapping in their own order. A
// projection that resolved nothing writes {} rather than null: a Record
// carrying only its identity is a perfectly good Record, and a reader asking
// what was projected is answered *nothing* rather than *the question was not
// asked* (§3, §7).
//
// It is the response object's own encoder under a second name, which is what
// keeps the two mappings a Probe renders written by one rule: an ordered
// mapping is an ordered mapping, and a second loop here is a second place for
// the key order or the escaping to drift (§8, ADR-0026).
func (f Fields) MarshalJSON() ([]byte, error) {
	mapping := make(capability.Object, 0, len(f))
	for _, field := range f {
		mapping = append(mapping, capability.Member{Name: field.Name, Value: field.Value})
	}
	return mapping.MarshalJSON()
}

// Read reads an Operation's record: fields: off the node that Operation is
// declared by. Which node that is is internal/artefact's to answer
// (artefact.OperationNode): this package knows what a record: block means and
// not where one lives.
//
// It judges nothing and drops what it cannot read, which is every reader's rule
// in this tool: what is wrong with a Manifest is check's to report and never a
// reader's to guess at (ADR-0064).
func Read(operation *yaml.Node) Projection {
	record := mappingValue(operation, "record")

	var read Projection
	read.Over = scalarValue(mappingValue(record, "over"))

	fields := mappingValue(record, "fields")
	if fields == nil || fields.Kind != yaml.MappingNode {
		return read
	}

	for i := 0; i+1 < len(fields.Content); i += 2 {
		key, value := fields.Content[i], fields.Content[i+1]
		if key.Kind != yaml.ScalarNode || value.Kind != yaml.ScalarNode {
			continue
		}
		read.Fields = append(read.Fields, FieldPath{Name: key.Value, Path: value.Value})
	}
	return read
}

// Project resolves every recorded field against the root the position names and
// answers the ones that resolved, in the Manifest's own order. A path that
// resolved to nothing contributes no field, which is what makes a field going
// quiet an absence a surface reads rather than an error it reports (§6, §12).
//
// The root is the response object on an Operation of `one` cardinality and one
// member of the collection `over:` named on an Operation of `series`. It is one
// function for both because it is one walk over one grammar: what differs is
// what `$` is, and that is decided by the position a path was written in and
// never by the path (§3, §12).
func (p Projection) Project(root any) Fields {
	var projected Fields
	for _, field := range p.Fields {
		value, resolved := Resolve(field.Path, root)
		if !resolved {
			continue
		}
		projected = append(projected, Field{Name: field.Name, Value: value})
	}
	return projected
}

// Resolve evaluates one path against the root it was written to root at, and
// answers what it resolved to. resolved is false where the path names something
// the root does not carry — which is a different answer from a path that
// resolved to a null the response did carry, that being a value with `resolved`
// true and `value` nil.
//
// The root is the response object at every position a Manifest writes a path
// except one: on an Operation of `series` cardinality the `identity:` and every
// `fields:` entry root at a **member** of the collection `over:` named, which
// is whatever that member parsed to. The distinction the answer draws is
// unchanged and so is the grammar; only what `$` names moves (§3, §12).
//
// A path that is not in the grammar resolves to nothing rather than to an
// error: check refuses a malformed path at load (§4), and a resolver that
// answered a second way about the same fault would be a second opinion on an
// artefact nobody reviewed (ADR-0064).
func Resolve(path string, root any) (value any, resolved bool) {
	names, inGrammar := Segments(path)
	if !inGrammar {
		return nil, false
	}

	current := root
	for _, name := range names {
		current, resolved = member(current, name)
		if !resolved {
			return nil, false
		}
	}
	return current, true
}

// Segments is the members a path names, in order, and false where its
// characters are not the grammar. `$` alone names none, which is the whole
// response object and — where a path roots at a Record rather than at a
// response — the member itself (§3, §12).
//
// It is exported because a second root reads the same grammar against something
// that is not a response object at all: an `{item:}` reference resolves against
// the Record its Step is ranging over, whose fields are the Store's own values
// rather than a Capability's (§6, issue #139). The walk differs and the grammar
// does not, so the grammar is read here and the hop is the caller's.
func Segments(path string) ([]string, bool) {
	rest, ok := strings.CutPrefix(path, "$")
	if !ok {
		return nil, false
	}

	var names []string
	for rest != "" {
		segment := segmentPattern.FindStringSubmatch(rest)
		if segment == nil {
			return nil, false
		}
		// The two productions name a member the same way and differ only
		// in what characters a member may be spelled with: `.member` is
		// an identifier and `["member"]` is anything, which is how a
		// header name carrying a hyphen or a dot is reachable at all.
		names = append(names, segment[1]+segment[2])
		rest = rest[len(segment[0]):]
	}
	return names, true
}

// member is one hop: what a named member of the value in hand holds, and false
// where the value carries no such member or is not something a member can be
// read off at all.
//
// The three shapes are the three a response object is built from: the object
// itself, the headers mapping, and whatever a JSON body parsed to. A scalar
// answers nothing, which is the grammar reaching no further inside one than
// inside a string anywhere else — the sentence §12 writes about `$.stdout` and
// which holds here for the same reason.
func member(value any, name string) (any, bool) {
	switch value := value.(type) {
	case capability.Object:
		return value.Lookup(name)
	case map[string]string:
		held, carried := value[name]
		return held, carried
	case map[string]any:
		held, carried := value[name]
		return held, carried
	default:
		return nil, false
	}
}

// Text is what a projected value reads as on a page: a string as itself, and
// everything else as the JSON it is. A page rendering a string in quotes would
// be quoting a host name; a page rendering an object without them would be
// writing something that is not any notation at all.
func Text(value any) string {
	if text, isString := value.(string); isString {
		return text
	}

	encoded, err := capability.CompactJSON(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}

// Collection is the members an Operation of `series` cardinality projects its
// Records out of: what the `over:` path resolved to, read as a sequence.
//
// resolved is false where the path names something the response does not carry,
// which is the fault §6 halts a Run on — without it `hyper` cannot tell a
// collection that was empty from a path that was wrong, which is the *I
// recorded nothing* ADR-0017 declined to leave a reader diagnosing off an
// absent wire.
//
// A path that resolved to something that is **not** a sequence answers the
// empty collection and resolved, which is the ordinary reading of a value the
// grammar reached and found nothing inside: a member is a member of a list, and
// a scalar has none. It is not a second fault to report — §6's halt is *the
// path did not resolve*, and this one did.
func Collection(path string, response capability.Object) ([]any, bool) {
	value, resolved := Resolve(path, response)
	if !resolved {
		return nil, false
	}
	members, isSequence := value.([]any)
	if !isSequence {
		return nil, true
	}
	return members, true
}

// scalarValue is a node's text, and "" where the node is absent or is not a
// plain scalar — the same drop rule Read follows over the mapping around it.
func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

// mappingValue is the value one key of a mapping holds, and nil where the node
// is not a mapping or does not carry that key.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if k := node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
