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
// references resolved against what this Expansion is ranging over — and, on a
// `shell` Operation, the argv beside them.
//
// What one position reads to is argument's below, and this is the walk over the
// positions an Operation declares. The one fault it answers itself is an input
// nothing supplies, which is `schema-mismatch` at load and a Run that reached
// Step 1 could not have (§6, ADR-0064).
func (r run) arguments(operation artefact.OperationInfo, authored sequenced, resolving member, cited citation) (resolvedArgs, *Refusal, error) {
	read := resolvedArgs{Inputs: make(map[string]schema.Scalar, len(authored.Args))}
	for _, name := range slices.Sorted(keys(operation.Inputs)) {
		node := authored.Args[name]
		if node == nil {
			return resolvedArgs{}, nil, fmt.Errorf("step %s supplies no %s, which %s declares — hyper check reports it", named(authored), name, authored.Operation)
		}

		// The `shell` Capability's one input is the argv, and it is a
		// list rather than a value: read as a scalar it would be a
		// declared input nothing could supply. It fills no position on
		// §12's two-column table — `hyper` execs the list rather than
		// serialising it — so it is resolved here and held beside the
		// inputs rather than among them (§3, §12, ADR-0051, ADR-0081).
		if operation.IsShell && name == artefact.ShellCommandInput {
			argv, declined, err := r.argv(authored, resolving, node, cited)
			if err != nil || declined != nil {
				return resolvedArgs{}, declined, err
			}
			read.Argv = argv
			continue
		}

		value, declined, err := r.argument(schema.Type(operation.Inputs[name].Type), name, node, authored, resolving, cited)
		if err != nil || declined != nil {
			return resolvedArgs{}, declined, err
		}
		read.Inputs[name] = value
	}
	return read, nil, nil
}

// resolvedArgs is one member's `args:` at its turn: the inputs read against the
// types their positions declare, and — on a `shell` Operation — the argv its
// `command:` resolved to.
//
// The argv is a member of its own rather than an input among the others
// because it is not one: every other input is a scalar meeting §12's two-column
// table, and this is a list `hyper` hands to a process. A map of scalars with a
// list wedged into it would be a shape every reader of the inputs would have to
// know about, and none of them does.
type resolvedArgs struct {
	Inputs map[string]schema.Scalar
	// Argv is nil on every Operation but a `shell` one, which is the only
	// place a list reaches anything (§12, schema.ReadScalar).
	Argv []string
}

// argv is a `shell` Step's `command:` resolved for one member: the argv words
// in the order they were authored, first word first (§3, ADR-0051).
//
// **The first member is the reach axis and is a literal.** A reference there
// would put the choice of binary in a value the world supplied, which is the
// arrival ADR-0029 closed for a host reappearing on the one Capability no grant
// bounds — so `command-malformed` refuses it at load, offline, and nothing here
// re-decides it. What stands here for the shape `check` already refused is a
// halt saying so rather than an argv `hyper` assembled from a reference anyway
// (ADR-0064).
//
// Every member after the first is referenceable, which is what makes an
// Expansion writable at all, and each is read exactly as a `string` input is
// read: characters against the declared type, one function over both (ADR-0081).
func (r run) argv(authored sequenced, resolving member, node *yaml.Node, cited citation) ([]string, *Refusal, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, nil, fmt.Errorf("step %s writes a command: that is not a list of argv words — hyper check reports it", named(authored))
	}
	if len(node.Content) == 0 {
		return nil, nil, fmt.Errorf("step %s writes an empty command:, and there is no executable to name — hyper check reports it", named(authored))
	}
	if head := node.Content[0]; head.Kind != yaml.ScalarNode {
		return nil, nil, fmt.Errorf("step %s writes a command: whose first member is not a literal, and the first member is the reach axis — hyper check reports it", named(authored))
	}

	words := make([]string, 0, len(node.Content))
	for at, item := range node.Content {
		word, declined, err := r.argument(schema.String, fmt.Sprintf("%s[%d]", artefact.ShellCommandInput, at), item, authored, resolving, cited)
		if err != nil || declined != nil {
			return nil, declined, err
		}
		words = append(words, word.Text())
	}
	return words, nil, nil
}

// argument reads one authored `args:` position for one member: a literal
// against the type declared there, and a reference resolved against what the
// Expansion is ranging over.
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
// milestone cannot reach: an authored literal whose characters do not read.
// That is `schema-mismatch` at load, and a Run that reached Step 1 is a Run
// whose artefacts passed (§6, ADR-0064).
//
// name is the position as a message and a citation name it — an input's own
// name, or `command[2]` for one argv word — so that a Refusal says which of a
// Step's lines it was.
func (r run) argument(declared schema.Type, name string, node *yaml.Node, authored sequenced, resolving member, cited citation) (schema.Scalar, *Refusal, error) {
	text := node.Value
	if node.Kind == yaml.MappingNode {
		referenced, isResolved := r.reference(authored, resolving, node)
		if !isResolved {
			declined := r.refusal(schema.CodeMismatch,
				fmt.Sprintf("%s resolves to nothing on %s, and every input %s declares is supplied", referenceText(node), expansionMember(resolving.Name, cited), authored.Operation),
				cited.at(node.Line, "args."+name))
			return schema.Scalar{}, &declined, nil
		}
		held, isScalar := scalarText(referenced)
		if !isScalar {
			declined := r.refusal(schema.CodeMismatch,
				fmt.Sprintf("%s resolves to %s on %s, and %s declares %s a %s", referenceText(node), describe(referenced), expansionMember(resolving.Name, cited), authored.Operation, name, declared),
				cited.at(node.Line, "args."+name))
			return schema.Scalar{}, &declined, nil
		}
		text = held
	}

	value, reads := schema.ReadScalar(declared, text)
	if !reads {
		if node.Kind != yaml.MappingNode {
			return schema.Scalar{}, nil, fmt.Errorf("step %s writes %s: %s, which does not read as the %s %s declares it",
				named(authored), name, node.Value, declared, authored.Operation)
		}
		declined := r.refusal(schema.CodeMismatch,
			fmt.Sprintf("%s resolves to %q on %s, which does not read as the %s %s declares %s", referenceText(node), text, expansionMember(resolving.Name, cited), declared, authored.Operation, name),
			cited.at(node.Line, "args."+name))
		return schema.Scalar{}, &declined, nil
	}
	return value, nil, nil
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
