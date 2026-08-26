// Package yamlsubset is the loader's one rule for issue #88: the strict YAML
// subset §3 states. YAML is parsed strictly — an anchor, an alias, a merge
// key, a tag, a multi-document file, or implicit type resolution is
// strict-yaml-violation, positioned at the file, line and column it occurs on
// (ADR-0023). This package knows nothing about any artefact schema — that is
// internal/schema and internal/artefact's rule (issue #89) — but it exposes
// the node its own decode produces, Parse, so a caller checking both this
// package's grammar and an artefact's schema reads the file once.
package yamlsubset

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// ErrorCode is the one code this package emits.
const ErrorCode = "strict-yaml-violation"

// Parse decodes data as YAML and returns the first document's root content
// node together with the load-time problems this package reports at that
// granularity. ok is false where the file will not parse at all — a hard
// YAML syntax error — which is the one case "loading a file is the first
// check, and failing it stops every check after" names (§4): root is nil,
// problems holds exactly one entry, and neither this package's own walk nor
// the schema check that reads the same node (issue #89) runs. ok is true and
// root is nil where the file is empty: zero documents is valid YAML, and
// whether it is a valid artefact is a schema question this package does not
// answer. A file holding more than one YAML document is ok, with a problem
// for the second document and the first document's node returned all the
// same.
//
// Parse walks nothing itself, so a caller that needs both this package's
// grammar rules and the schema check reads the file once rather than twice:
// Scan below calls Violations on the node it returns.
func Parse(file string, data []byte) (root *yaml.Node, problems []problem.Problem, ok bool) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []*yaml.Node
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, []problem.Problem{parseErrorProblem(file, err)}, false
		}
		docs = append(docs, &doc)
	}

	if len(docs) == 0 {
		return nil, nil, true
	}

	if len(docs) > 1 {
		second := docs[1]
		problems = append(problems, problem.Problem{
			File:      file,
			Line:      second.Line,
			Column:    second.Column,
			ErrorCode: ErrorCode,
			Message:   "a file may hold exactly one YAML document",
		})
	}

	first := docs[0]
	if len(first.Content) == 0 {
		return nil, problems, true
	}
	return first.Content[0], problems, true
}

// Violations walks root for the strict-subset grammar and returns one
// problem per rejected construct — Scan's own walk, exposed so a caller that
// already holds root from Parse does not decode the file a second time
// (issue #89).
func Violations(root *yaml.Node, file string) []problem.Problem {
	var problems []problem.Problem
	walk(root, "", file, &problems)
	return problems
}

// Scan reads data as YAML and returns one problem per construct the strict
// subset rejects. A file that will not parse at all — a hard YAML syntax
// error — yields exactly one problem and scanning stops there, since the
// parser has nothing more to walk (issue #88's "an artefact whose own file
// will not parse still declares its name and yields exactly one problem").
// A syntactically valid file that uses a forbidden construct more than once
// yields one problem per occurrence.
func Scan(file string, data []byte) []problem.Problem {
	root, problems, ok := Parse(file, data)
	if !ok {
		return problems
	}
	if root != nil {
		problems = append(problems, Violations(root, file)...)
	}
	return problems
}

// parseErrorLine matches go-yaml's "yaml: line N: <reason>" error shape so
// the one problem a hard parse failure produces still carries a position.
var parseErrorLine = regexp.MustCompile(`^yaml: line (\d+): (.*)$`)

func parseErrorProblem(file string, err error) problem.Problem {
	line := 1
	message := err.Error()
	if m := parseErrorLine.FindStringSubmatch(message); m != nil {
		if n, convErr := strconv.Atoi(m[1]); convErr == nil {
			line = n
		}
		message = m[2]
	}
	return problem.Problem{
		File:      file,
		Line:      line,
		Column:    1,
		ErrorCode: ErrorCode,
		Message:   message,
	}
}

// mergeTag is the tag go-yaml's implicit resolver assigns to a plain,
// untagged scalar spelled exactly "<<" — the merge key sigil (YAML 1.1
// §10.2.1.3, kept alive by YAML 1.2 core for compatibility).
const mergeTag = "!!merge"

// nullTag is the tag go-yaml's implicit resolver assigns to an empty value,
// "~", or "null"/"Null"/"NULL" spelled plain. hyper's scalar vocabulary (§12)
// has no null anywhere, so a scalar that resolves to it has nowhere to be
// read as — the one case "implicit type resolution" can mean independently
// of any schema, which is what makes it this package's rule rather than the
// reading rule's (§89, ADR-0081).
const nullTag = "!!null"

func walk(n *yaml.Node, field, file string, problems *[]problem.Problem) {
	if n == nil {
		return
	}

	for _, msg := range violations(n) {
		emit(problems, file, field, n, msg)
	}

	switch n.Kind {
	case yaml.MappingNode:
		// The names this mapping has already given, and the whole of the
		// duplicate-key rule.
		//
		// It is the one rejected construct that is a property of a
		// **mapping** rather than of a node, which is why it is walked
		// here rather than listed in violations below. §3 states it as a
		// property and names no check for it — *a mapping makes a name
		// unique by construction, so there is no duplicate-name rule to
		// state* — and go-yaml decoding into a node keeps both spellings
		// and resolves the mapping to the last, which is exactly the
		// shape the strict subset exists to refuse: one line's meaning
		// depending on another line (§3, ADR-0023).
		//
		// It is per mapping and never per file: the same name under two
		// parents is two names, and every Manifest whose Operations each
		// declare a `kind:` has several. And it is reported on the
		// **second** spelling, which is the line an author deletes; the
		// first is the one they meant.
		named := make(map[string]bool, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i]
			val := n.Content[i+1]

			if key.Kind == yaml.ScalarNode && key.Tag == mergeTag {
				emit(problems, file, join(field, "<<"), key, "a merge key is not part of the authoring format")
			} else {
				for _, msg := range violations(key) {
					emit(problems, file, field, key, msg)
				}
				if key.Kind == yaml.ScalarNode {
					if named[key.Value] {
						emit(problems, file, join(field, key.Value), key, fmt.Sprintf("a mapping names each key once, and %q is named again here", key.Value))
					}
					named[key.Value] = true
				}
			}

			childField := field
			if key.Kind == yaml.ScalarNode {
				childField = join(field, key.Value)
			}
			walk(val, childField, file, problems)
		}
	case yaml.SequenceNode:
		for i, item := range n.Content {
			walk(item, fmt.Sprintf("%s[%d]", field, i), file, problems)
		}
	}
}

// violations reports every rejected construct a single node exhibits. A node
// can carry more than one at once — an anchored, explicitly tagged scalar —
// and each is its own occurrence.
func violations(n *yaml.Node) []string {
	var msgs []string
	if n.Style&yaml.TaggedStyle != 0 {
		msgs = append(msgs, "an explicit tag is not part of the authoring format")
	}
	if n.Anchor != "" {
		msgs = append(msgs, "an anchor is not part of the authoring format")
	}
	if n.Kind == yaml.AliasNode {
		msgs = append(msgs, "an alias is not part of the authoring format")
	}
	if n.Kind == yaml.ScalarNode && n.Tag == nullTag {
		msgs = append(msgs, "there is no null in the scalar vocabulary — quote the value or write it out")
	}
	return msgs
}

// emit appends one problem at n's position — the one shape every violation
// this package finds takes.
func emit(problems *[]problem.Problem, file, field string, n *yaml.Node, message string) {
	*problems = append(*problems, problem.Problem{
		File:      file,
		Line:      n.Line,
		Column:    n.Column,
		Field:     field,
		ErrorCode: ErrorCode,
		Message:   message,
	})
}

func join(field, name string) string {
	if field == "" {
		return name
	}
	return field + "." + name
}
