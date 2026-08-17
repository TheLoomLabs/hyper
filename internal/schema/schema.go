// Package schema is hyper's own schema engine — the reading rule and the
// object shape it reads against (§3, §12, ADR-0081). A scalar is read
// against the schema at its position, never compared with it: the value's
// characters are read as the declared type, in the text form §12 fixes, and
// the quoting YAML required is lexical rather than part of the value.
// additionalProperties: false is forced at every level rather than
// authored, so an object's Properties are its complete key set and nothing
// here ever widens them.
//
// This is not the input-schema subset a Manifest's Operation writes in
// (four keywords, authored, closed by §12 on its own). It is the schema
// hyper holds for its own five artefacts, compiled in rather than written
// anywhere a reviewer reads — Check is what "hyper reads it against hyper's
// own schema for that artefact" means.
package schema

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// The two codes hyper's own schema declines under (§4, §12).
// additionalProperties: false is forced rather than authored, so
// CodeUnknownKey refuses the one constraint hyper imposes and CodeMismatch
// refuses a value against what an author wrote — a value whose characters
// will not read as the declared type, a key the schema declares that the
// file does not supply, and a value outside its enum are all one check and
// this one code (ADR-0081).
const (
	CodeMismatch   = "schema-mismatch"
	CodeUnknownKey = "unknown-key"
)

// Type is one member of the closed scalar vocabulary (§12), plus the two
// structural types every object and array position is built from.
type Type string

const (
	String    Type = "string"
	Integer   Type = "integer"
	Number    Type = "number"
	Boolean   Type = "boolean"
	Object    Type = "object"
	Array     Type = "array"
	Duration  Type = "duration"
	Timestamp Type = "timestamp"
)

// Schema describes what hyper admits at one position.
type Schema struct {
	Type Type
	// Enum constrains a scalar Type's legal text; empty means unconstrained.
	Enum []string
	// Properties is legal only where Type == Object, and is the complete
	// key set at that position — additionalProperties: false is forced
	// rather than authored (§12).
	Properties []Property
	// Items is legal only where Type == Array, and describes every member.
	Items *Schema
}

// Property is one named member of an Object schema.
type Property struct {
	Name     string
	Schema   Schema
	Required bool
}

// Check reads root against s and returns one problem per position that does
// not fit. root is nil where the file supplied no document at all, which
// reads the same as an object that supplied none of its keys — a required
// property is schema-mismatch there like anywhere else, at the file's own
// position for lack of a line of its own to name.
func Check(root *yaml.Node, s Schema, file string) []problem.Problem {
	var problems []problem.Problem
	read(root, s, "", file, &problems)
	return problems
}

// read is Check's recursion. n is nil only where the position named by
// field was never supplied at all — an object's missing required key never
// reaches a node of its own, so its member schemas are read against nil
// too, in case that member is itself an object with required keys of its
// own to name individually rather than as one collapsed fault.
func read(n *yaml.Node, s Schema, field, file string, problems *[]problem.Problem) {
	switch s.Type {
	case Object:
		readObject(n, s, field, file, problems)
	case Array:
		readArray(n, s, field, file, problems)
	default:
		readScalar(n, s, field, file, problems)
	}
}

func readObject(n *yaml.Node, s Schema, field, file string, problems *[]problem.Problem) {
	if n != nil && n.Kind != yaml.MappingNode {
		emit(problems, file, field, n.Line, n.Column, CodeMismatch, "expected an object at this position")
		return
	}

	line, column := position(n)
	seen := make(map[string]bool, len(s.Properties))
	if n != nil {
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				// A malformed key — an anchor, a merge key, a tag — is
				// yamlsubset's rule to name; this package has nothing to
				// match it against and says nothing here.
				continue
			}
			prop, ok := property(s.Properties, key.Value)
			if !ok {
				emit(problems, file, join(field, key.Value), key.Line, key.Column, CodeUnknownKey,
					fmt.Sprintf("%q is not a key the schema at this position admits", key.Value))
				continue
			}
			seen[key.Value] = true
			read(val, prop.Schema, join(field, key.Value), file, problems)
		}
	}

	for _, prop := range s.Properties {
		if prop.Required && !seen[prop.Name] {
			emit(problems, file, join(field, prop.Name), line, column, CodeMismatch,
				fmt.Sprintf("the schema at this position declares %q, and this file does not supply it", prop.Name))
		}
	}
}

func readArray(n *yaml.Node, s Schema, field, file string, problems *[]problem.Problem) {
	if n == nil || n.Kind != yaml.SequenceNode {
		line, column := position(n)
		emit(problems, file, field, line, column, CodeMismatch, "expected an array at this position")
		return
	}
	for i, item := range n.Content {
		read(item, *s.Items, fmt.Sprintf("%s[%d]", field, i), file, problems)
	}
}

func readScalar(n *yaml.Node, s Schema, field, file string, problems *[]problem.Problem) {
	if n == nil || n.Kind != yaml.ScalarNode {
		line, column := position(n)
		emit(problems, file, field, line, column, CodeMismatch,
			fmt.Sprintf("expected %s at this position", article(s.Type)))
		return
	}
	if !readsAs(s.Type, n.Value) {
		emit(problems, file, field, n.Line, n.Column, CodeMismatch,
			fmt.Sprintf("%q does not read as %s at this position", n.Value, article(s.Type)))
		return
	}
	if len(s.Enum) > 0 && !member(s.Enum, n.Value) {
		emit(problems, file, field, n.Line, n.Column, CodeMismatch,
			fmt.Sprintf("%q is outside the enum at this position", n.Value))
	}
}

// The scalar text forms §12 fixes, read rather than compared: a node's own
// implicit tag never enters this — the characters are read as t regardless
// of what go-yaml's resolver made of them, which is what makes "2592000"
// and 2592000 one value at an integer position and NO at a boolean one read
// as nothing at all (ADR-0081).
//
// numberPattern reads leniently, on the same ground integerPattern does:
// what a number renders as — the shortest decimal that round-trips (§12) —
// is a property of a writer this milestone does not yet have. check never
// re-emits a value; it only reads one against the schema at its position,
// and nothing here is authority for how a later writer canonicalises one
// (§7, ADR-0079).
var (
	integerPattern  = regexp.MustCompile(`^-?[0-9]+$`)
	numberPattern   = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)
	booleanPattern  = regexp.MustCompile(`^(true|false)$`)
	durationPattern = regexp.MustCompile(`^[0-9]+[smhd]$`)
)

func readsAs(t Type, value string) bool {
	switch t {
	case String:
		return true
	case Integer:
		return integerPattern.MatchString(value)
	case Number:
		return numberPattern.MatchString(value)
	case Boolean:
		return booleanPattern.MatchString(value)
	case Duration:
		return durationPattern.MatchString(value)
	case Timestamp:
		return readsAsTimestamp(value)
	default:
		return false
	}
}

// readsAsTimestamp reads value as RFC 3339 in UTC with Z mandatory — an
// offset form is refused at an authored position even where it parses,
// since the suffix check runs first.
func readsAsTimestamp(value string) bool {
	if !strings.HasSuffix(value, "Z") {
		return false
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func property(props []Property, name string) (Property, bool) {
	for _, p := range props {
		if p.Name == name {
			return p, true
		}
	}
	return Property{}, false
}

func member(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func join(field, name string) string {
	if field == "" {
		return name
	}
	return field + "." + name
}

// position reports where to point a problem that has no node of its own to
// point at — an absent required key, an absent object or array entirely.
// (1, 1) is the file itself, the same fallback a hard parse failure uses.
func position(n *yaml.Node) (int, int) {
	if n == nil {
		return 1, 1
	}
	return n.Line, n.Column
}

func article(t Type) string {
	switch t {
	case Object, Integer, Array:
		return "an " + string(t)
	default:
		return "a " + string(t)
	}
}

func emit(problems *[]problem.Problem, file, field string, line, column int, code, message string) {
	*problems = append(*problems, problem.Problem{
		File:      file,
		Line:      line,
		Column:    column,
		Field:     field,
		ErrorCode: code,
		Message:   message,
	})
}
