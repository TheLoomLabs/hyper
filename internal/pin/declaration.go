package pin

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
)

// Written is `hyper.yaml`'s bytes carrying version and digest — the two
// derived facts `hyper project` writes into the Repository declaration and
// nothing else in the tool writes at all (§3, §11, ADR-0020).
//
// **The file is edited, never regenerated.** `retention:` is authored, the
// comments and the layout are the author's, and the whole-file rule that
// governs a projected workflow protects files that are *entirely* derived — so
// what happens here is two scalars replaced in place and every other byte
// carried through untouched. A declaration is a reviewed artefact that carries
// two derived facts, not a generated file.
//
// present is false where the repository holds no `hyper.yaml` at all, and what
// it gets then is a declaration carrying `kind:`, `version:`, `digest:` and
// **no `retention:`**: a repository that has not stated a policy has not agreed
// to lose anything, and `project` does not author one on its behalf (§3).
//
// It judges neither value. A version is what the binary says it is and a digest
// is what the release published, and the diff this writes is where both are
// read (§9, ADR-0020).
func Written(data []byte, present bool, version, digest string) []byte {
	if !present {
		return []byte("kind: " + artefact.KindRepositoryDeclaration + "\n" +
			"version: " + version + "\n" +
			"digest: " + digest + "\n")
	}
	// One key at a time, and the second edit reads the first's answer: a
	// value that changed length moved every column after it on its own line,
	// so the second key's position is found in the bytes the first edit
	// produced rather than in the ones it was handed.
	return written(written(data, "version", version), "digest", digest)
}

// Declared is the version pin these bytes carry, and "" where they carry none —
// no `version:` key, a value that is not a plain scalar, or a file that will not
// parse at all.
//
// It is what decides whether `project` reaches the network: a pin already equal
// to the binary's version resolves nothing, and the digest already in the
// declaration is copied into every workflow (§11). Where it answers "" — the
// repository that has never been projected — the two differ and the checksum is
// resolved, which is the same act an upgrade performs.
//
// It answers a value where Check answers a decision, and the two read one member
// through one struct so that *what the pin says* and *what the gate does about
// it* cannot come apart.
func Declared(data []byte) string {
	var decl repositoryDeclaration
	if err := yaml.Unmarshal(data, &decl); err != nil {
		return ""
	}
	return decl.Version
}

// written is data with one top-level scalar carrying value, and everything else
// exactly as it stood.
//
// **The edit is the scalar's own span and not its line**, which is what carries
// a trailing comment and the author's quoting through: the parse says where the
// value begins, the value it parsed to says how long it is, and what is replaced
// is that substring. A pin the author wrote `"1.3.0"` stays quoted, and the
// `# bumped by hyper project` after it stays where they put it.
//
// Two shapes have no span to edit, and neither is reachable through `project` —
// `hyper.yaml`'s schema makes both keys required, and `project` writes nothing
// where `check` would report anything. **A key the file does not carry at all is
// appended**, because doing nothing there would answer a declaration missing a
// fact this was asked to write. **A key carrying anything else is left exactly
// as it stands** — an empty scalar, a mapping, a sequence — because what is
// wrong with it is a `schema-mismatch` a reader is owed rather than a spelling
// this may invent for them (§4, ADR-0064).
func written(data []byte, key, value string) []byte {
	held := valueOf(topLevel(data), key)
	if held == nil {
		return []byte(appended(string(data), key, value))
	}

	lines := strings.SplitAfter(string(data), "\n")
	at := held.Line - 1
	if held.Kind != yaml.ScalarNode || held.Value == "" || at < 0 || at >= len(lines) {
		return data
	}

	// The scalar's span, searched for from where the parse says it opens.
	// `from` is inside the line for every value the parse can have found
	// there; the bound is what keeps a Column the file does not have from
	// being an index rather than a miss.
	line, from := lines[at], held.Column-1
	if from < 0 || from > len(line) {
		return data
	}
	span := strings.Index(line[from:], held.Value)
	if span < 0 {
		// A value spelled across more lines than the one it opens on.
		return data
	}

	from += span
	lines[at] = line[:from] + value + line[from+len(held.Value):]
	return []byte(strings.Join(lines, ""))
}

// appended is data with `key: value` written after its last line, for the key it
// does not carry. It is written as its own line whatever the file ended with, so
// a declaration with no trailing newline does not swallow the key into its last
// one.
func appended(data, key, value string) string {
	if data != "" && !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	return data + key + ": " + value + "\n"
}

// topLevel is the mapping `hyper.yaml`'s bytes parse to, and nil where they
// parse to anything else — no document, a sequence, a scalar, or nothing that
// parses at all. Every caller reads nil the way a lookup into a nil mapping
// already reads.
func topLevel(data []byte) *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}
	if root := doc.Content[0]; root.Kind == yaml.MappingNode {
		return root
	}
	return nil
}

// valueOf is what one top-level key carries, and nil where the mapping does not
// carry that key at all — which is the one state that is an absence rather than
// a shape, and the only one an edit can answer by writing the key.
func valueOf(root *yaml.Node, key string) *yaml.Node {
	if root == nil {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if name, value := root.Content[i], root.Content[i+1]; name.Value == key {
			return value
		}
	}
	return nil
}
