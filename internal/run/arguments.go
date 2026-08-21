package run

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// An `args:` value at a Step's turn: a literal read against the type declared
// at its position, and a reference resolved against what the Expansion is
// ranging over (§3, §6, §12, ADR-0081, issue #139).
//
// It sits beside the Expansion rather than inside it because the subject is a
// different one. What a selector resolves to is a question about a population;
// this is a question about **one value meeting one declared type**, and the
// answer is the same reading `check` performs over an authored scalar at load —
// characters against the declared type's text form, never a comparison of the
// value's own JSON type.

// arguments reads the Step's `args:` for one member: every declared input
// filled, literals read against the type declared at that position and
// references resolved against what this Expansion is ranging over.
//
// **This is where a stored value meets the type an input declares.** It is read
// exactly as an authored scalar is read at load — characters against the
// declared type's text form — so a stored `"2592000"` fills an `integer` input
// and a stored `"thirty"` does not, and a Refusal does not turn on whether a
// remote API answered a string or a number (§6, ADR-0081). One that will not
// read is `schema-mismatch`, the same code §4 fires where the value is on the
// page; a reference resolving to nothing supplies no value at all, which is the
// same code again.
//
// The error beside the Refusal is a fault `check` has already reported and this
// milestone cannot reach: an input nothing supplies, and an authored literal
// whose characters do not read. Both are `schema-mismatch` at load, and a Run
// that reached Step 1 is a Run whose artefacts passed (§6, ADR-0064).
func (r run) arguments(operation artefact.OperationInfo, authored sequenced, resolving member, cited citation) (map[string]schema.Scalar, *Refusal, error) {
	read := make(map[string]schema.Scalar, len(authored.Args))
	for _, name := range slices.Sorted(keys(operation.Inputs)) {
		node := authored.Args[name]
		if node == nil {
			return nil, nil, fmt.Errorf("step %s supplies no %s, which %s declares — hyper check reports it", named(authored), name, authored.Operation)
		}
		declared := schema.Type(operation.Inputs[name].Type)

		text := node.Value
		if node.Kind == yaml.MappingNode {
			referenced, resolved := r.reference(authored, resolving, node)
			if !resolved {
				declined := r.refusal(schema.CodeMismatch,
					fmt.Sprintf("%s resolves to nothing on %s, and every input %s declares is supplied", referenceText(node), expansionMember(resolving.Name, cited), authored.Operation),
					cited.at(node.Line, "args."+name))
				return nil, &declined, nil
			}
			held, isScalar := scalarText(referenced)
			if !isScalar {
				declined := r.refusal(schema.CodeMismatch,
					fmt.Sprintf("%s resolves to %s on %s, and %s declares %s a %s", referenceText(node), describe(referenced), expansionMember(resolving.Name, cited), authored.Operation, name, declared),
					cited.at(node.Line, "args."+name))
				return nil, &declined, nil
			}
			text = held
		}

		value, reads := schema.ReadScalar(declared, text)
		if !reads {
			if node.Kind != yaml.MappingNode {
				return nil, nil, fmt.Errorf("step %s writes %s: %s, which does not read as the %s %s declares it",
					named(authored), name, node.Value, declared, authored.Operation)
			}
			declined := r.refusal(schema.CodeMismatch,
				fmt.Sprintf("%s resolves to %q on %s, which does not read as the %s %s declares %s", referenceText(node), text, expansionMember(resolving.Name, cited), declared, authored.Operation, name),
				cited.at(node.Line, "args."+name))
			return nil, &declined, nil
		}
		read[name] = value
	}
	return read, nil, nil
}

// reference resolves one of the format's two reference forms, which are the
// only two there are (§3).
//
// **They resolve at one moment and against two things.** `{item:}` names the
// Record the Step is ranging over, which the Expansion has in hand; `{step:,
// path:}` names the Record an earlier Step of **this Run** acted on, which is
// the condition's root arriving one key over (§3, §12, condition.go). Both
// resolve here, before the Step's first call goes out, and one that resolves to
// nothing supplies no value at all — `schema-mismatch`, rather than a halt.
func (r run) reference(authored sequenced, resolving member, node *yaml.Node) (store.Value, bool) {
	if step, path, isStep := stepPath(node); isStep {
		return soleRecord(r.acted[stepKey{authored.Namespace, step}], path)
	}
	return resolving.reference(node)
}

// soleRecord is the value a `{step:, path:}` reference resolves to over what the
// Step it names acted on: the path read against **the** Record, and nothing at
// all where there is not exactly one.
//
// A Step that acted on no Record is the skip §6 states — it was skipped by
// either Disposition, or never reached — and a reference to it resolves to
// nothing. A Step that acted on several has no one Record for a reference to
// name: **the shape that is writable names one Record**, which is why `check`
// already refuses a reference to a Step of `series` cardinality outright (§3,
// §4, `series-reference`). An expanding Step of `one` cardinality is that same
// fact arriving at a Run rather than at a load, and it answers the same
// *resolves to nothing*.
//
// **A condition rooted at the same Step reads it differently, and deliberately
// so** (condition.go): a predicate is a filter and a filter over a population
// is an AND, where a reference is a value and a value is one thing. The
// position decides it, as every other legality question in this format does
// (§3) — and §12 states the rule for neither, both Steps in its text having one
// Record.
func soleRecord(acted []store.Mapping, path string) (store.Value, bool) {
	if len(acted) != 1 {
		return nil, false
	}
	return resolvePath(acted[0], path)
}

// reference resolves an `{item:}` reference against this member: `$` is the
// whole of the member, and a path with segments reads the head version of the
// series the member names (§3, §12).
func (m member) reference(node *yaml.Node) (store.Value, bool) {
	path, isItem := itemPath(node)
	if !isItem {
		return nil, false
	}
	if segments, inGrammar := projection.Segments(path); inGrammar && len(segments) == 0 {
		// `$` is the whole of what the member is: a `values:` member's
		// own scalar, and — where the selector ranges over series — the
		// whole Record, which is not a scalar and so fills no position
		// a reference may stand in (§3, ADR-0078). Answering the Record
		// rather than nothing is what makes the Refusal say which of
		// the two it was.
		if m.Item != nil {
			return m.Item, true
		}
		return m.Head, m.Head != nil
	}
	return resolvePath(m.Head, path)
}

// resolvePath reads a path in §12's grammar against one Record's fields, and
// answers false where the grammar rejects it or a segment names nothing.
//
// It is one walk for both reference forms because a Record is one shape at both
// roots: the Record a Step is ranging over and the Record an earlier Step acted
// on are the same fields under the same names, and two walks is where the day
// comes that one of them descends differently.
func resolvePath(fields store.Mapping, path string) (store.Value, bool) {
	segments, inGrammar := projection.Segments(path)
	if !inGrammar {
		return nil, false
	}
	if len(segments) == 0 {
		return fields, fields != nil
	}

	var current store.Value = fields
	for _, name := range segments {
		mapping, isMapping := current.(store.Mapping)
		if !isMapping {
			return nil, false
		}
		held, carried := mapping[name]
		if !carried {
			return nil, false
		}
		current = held
	}
	return current, true
}

// itemPath is the path an `{item:}` reference names, and false where the
// mapping is not one — a `{step:, path:}` reference, or a shape `check` has
// already refused.
func itemPath(node *yaml.Node) (string, bool) {
	if node.Kind != yaml.MappingNode || len(node.Content) != 2 {
		return "", false
	}
	key, value := node.Content[0], node.Content[1]
	if key.Kind != yaml.ScalarNode || key.Value != "item" || value.Kind != yaml.ScalarNode {
		return "", false
	}
	return value.Value, true
}

// stepPath is the Step and the path a `{step:, path:}` reference names, and
// false where the mapping is not one.
//
// The two keys are read by name rather than by position: a mapping's key order
// is the author's, and `{path: $.host, step: first}` is the same reference
// written the other way round.
func stepPath(node *yaml.Node) (step, path string, isStep bool) {
	if node.Kind != yaml.MappingNode {
		return "", "", false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || value.Kind != yaml.ScalarNode {
			continue
		}
		switch key.Value {
		case "step":
			step = value.Value
		case "path":
			path = value.Value
		}
	}
	return step, path, step != ""
}

// referenceText is a reference as a message names it, which is the mapping as
// it was authored: `{item: $.ttl}`.
func referenceText(node *yaml.Node) string {
	var written strings.Builder
	written.WriteString("{")
	for i := 0; i+1 < len(node.Content); i += 2 {
		if i > 0 {
			written.WriteString(", ")
		}
		fmt.Fprintf(&written, "%s: %s", node.Content[i].Value, node.Content[i+1].Value)
	}
	written.WriteString("}")
	return written.String()
}

// scalarText is a stored value as the characters an input's declared type is
// read against, and false where the value is not a scalar at all — an object or
// a list, which no scalar position admits and which a reference may not fill
// (§3, ADR-0078).
func scalarText(value store.Value) (string, bool) {
	switch held := value.(type) {
	case store.String:
		return string(held), true
	case store.Number:
		return held.Text(), true
	case store.Bool:
		if held {
			return "true", true
		}
		return "false", true
	case store.Timestamp:
		return store.InstantText(time.Time(held)), true
	default:
		return "", false
	}
}
