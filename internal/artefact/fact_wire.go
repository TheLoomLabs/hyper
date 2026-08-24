package artefact

import (
	"bytes"
	"encoding/json"
	"strconv"

	"gopkg.in/yaml.v3"
)

// A code fact's value as §8's row stream carries it (§8, issue #171).
//
// **A `from` or `to` that is not a scalar carries the artefact's own parsed
// shape** — `{"values":["ci-arm64","ci-x86"]}`, `{"assets":[{"field":
// "labels.role","equals":"preview"}]}`, `["read","mutate"]` — in the order the
// page renders it. The page's notation is that chapter's geometry and never a
// fact either surface states (ADR-0059), so nothing composed goes out composed:
// a ` · `-separated run and a `field operator operand` line are renderings, and
// what the wire carries is what the author wrote.
//
// It is built here because this is where the nodes are. A reader composing the
// wire out of the rendered members would have to parse its own rendering back —
// and could not: an `in:` list and a bare operand render alike once a conjunct
// is one line of text.

// jsonScalar is one scalar node as JSON, in the type the artefact wrote it.
//
// The tag is the fact and never the text: a Bound is `3` on the wire because
// the author wrote an integer, and `"3"` would be this surface re-typing a
// value it was handed. A tag whose literal will not read as the type it claims
// falls back to the string, which is what a consumer can still do something
// with — and what is wrong with the value is `check`'s to report (ADR-0064).
func jsonScalar(node *yaml.Node) json.RawMessage {
	switch node.Tag {
	case "!!int", "!!float":
		if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return json.RawMessage(node.Value)
		}
	case "!!bool":
		switch node.Value {
		case "true", "false":
			return json.RawMessage(node.Value)
		}
	case "!!null":
		return json.RawMessage("null")
	}
	return jsonText(node.Value)
}

// jsonText is one string as the wire writes it.
//
// **HTML escaping is off**, which is the stream's own rule and not this file's:
// the wire carries an artefact's own bytes, and a predicate operand quoting a
// `&` or a `<` is one a consumer reads back as it was written (§8,
// render.WriteJSON). `json.Marshal` escapes all three, so the encoder is used
// with escaping turned off rather than the shorthand.
func jsonText(text string) json.RawMessage {
	var written bytes.Buffer
	encoder := json.NewEncoder(&written)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(text); err != nil {
		// A Go string is always encodable; the branch is here so that
		// nothing in this file can return bytes that are not JSON.
		return json.RawMessage(`""`)
	}
	// The encoder terminates every value with a newline and this is one
	// value inside a larger object.
	return json.RawMessage(bytes.TrimSuffix(written.Bytes(), []byte("\n")))
}

// jsonNode is one node's subtree as JSON: the artefact's own parsed shape, with
// a mapping's keys in the order the file wrote them.
//
// A node this cannot read — an alias, a tag the subset does not admit — carries
// `null` rather than half a value: what is wrong with it is `check`'s to
// report, and a wire that wrote half a mapping would be a consumer's parse of
// something nobody authored.
func jsonNode(node *yaml.Node) json.RawMessage {
	if node == nil {
		return json.RawMessage("null")
	}
	switch node.Kind {
	case yaml.ScalarNode:
		return jsonScalar(node)
	case yaml.SequenceNode:
		written := []byte{'['}
		for i, item := range node.Content {
			if i > 0 {
				written = append(written, ',')
			}
			written = append(written, jsonNode(item)...)
		}
		return json.RawMessage(append(written, ']'))
	case yaml.MappingNode:
		written := []byte{'{'}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, val := node.Content[i], node.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			if len(written) > 1 {
				written = append(written, ',')
			}
			written = append(written, jsonText(key.Value)...)
			written = append(written, ':')
			written = append(written, jsonNode(val)...)
		}
		return json.RawMessage(append(written, '}'))
	}
	return json.RawMessage("null")
}

// jsonNames is a set-shaped fact's members as the wire carries them: the
// sorted, deduplicated run the page renders, as an array of strings.
//
// It is built from the members rather than from the node because the members
// *are* the value: a set compares by set equality, so the order the author
// happened to write it in is not a fact, and the wire carries the order the two
// cells are read against each other in (§8).
func jsonNames(members []string) json.RawMessage {
	written := []byte{'['}
	for i, member := range members {
		if i > 0 {
			written = append(written, ',')
		}
		written = append(written, jsonText(member)...)
	}
	return json.RawMessage(append(written, ']'))
}

// jsonForm is a selector as the wire carries it: the form it was written in,
// and its members beneath — the conjuncts in the order the page renders them,
// which is the order this row's two cells can be differenced by eye in (§8).
func jsonForm(form string, members []json.RawMessage) json.RawMessage {
	written := append(jsonText(form), ':', '[')
	written = append([]byte{'{'}, written...)
	for i, member := range members {
		if i > 0 {
			written = append(written, ',')
		}
		written = append(written, member...)
	}
	return json.RawMessage(append(written, ']', '}'))
}
