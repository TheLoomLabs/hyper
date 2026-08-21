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
func (r run) arguments(operation artefact.OperationInfo, authored artefact.Step, resolving member, cited citation) (map[string]schema.Scalar, *Refusal, error) {
	read := make(map[string]schema.Scalar, len(authored.Args))
	for _, name := range slices.Sorted(keys(operation.Inputs)) {
		node := authored.Args[name]
		if node == nil {
			return nil, nil, fmt.Errorf("step %s supplies no %s, which %s declares — hyper check reports it", authored.ID, name, authored.Operation)
		}
		declared := schema.Type(operation.Inputs[name].Type)

		text := node.Value
		if node.Kind == yaml.MappingNode {
			referenced, resolved := resolving.reference(node)
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
					authored.ID, name, node.Value, declared, authored.Operation)
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

// reference resolves an `{item:}` reference against this member: `$` is the
// whole of the member, and a path with segments reads the head version of the
// series the member names (§3, §12).
//
// A `{step:, path:}` reference reaches nothing here and answers false, which is
// the same *resolves to nothing* the path grammar answers: reading an earlier
// Step's Record is the condition's root and arrives with it (issue #141). It is
// unreachable while NotBuilt declines such a Step before Step 1.
func (m member) reference(node *yaml.Node) (store.Value, bool) {
	path, isItem := itemPath(node)
	if !isItem {
		return nil, false
	}
	segments, inGrammar := projection.Segments(path)
	if !inGrammar {
		return nil, false
	}

	if len(segments) == 0 {
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
	var current store.Value = m.Head
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
