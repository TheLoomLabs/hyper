package artefact

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// OperationSource is the Manifest lines declaring one Operation, verbatim: a
// range of the file's own bytes, taken from the Operation's key line through
// the last line of its mapping, with the comment above the key included and the
// blank lines after it trimmed.
//
// It is the one reader here that answers with bytes rather than with facts, and
// that is the whole of why it exists. A Manifest is written in the format the
// caller is expected to author Definitions in (§3), so handing the lines back
// unchanged teaches that format at the moment the caller needs it — and §12
// closes the identity: `operation` writes these lines back unchanged, and what
// a reviewer reads is what manifest_digest covers. A re-encoding that produced
// equivalent YAML would break that silently, the digest still being right over
// bytes the reviewer never saw. So nothing here parses a value, re-indents a
// line or re-wraps one: the parse tree is read for where the range is and the
// bytes are copied out of the file.
//
// manifest is the exact bytes root parsed from, which is what the load keeps
// beside every Manifest for this reason and for manifest_digest's (§7). Passing
// the pair rather than re-reading the file is what makes the built-in shell
// Provider answerable at all: it has no file, and its bytes are the constant
// compiled into the binary (§12, ADR-0039).
//
// It finds nothing where the name is not a key of a legible operations: block,
// which is the lookup a usage error is written off (§9, ADR-0060), and where
// the Manifest has no such block to look in at all — what is wrong with it is
// check's to name and never this reader's to guess at (ADR-0064).
func OperationSource(manifest []byte, root *yaml.Node, name string) (string, bool) {
	operations := operationsMapping(root)
	if operations == nil {
		return "", false
	}
	lines := readSourceLines(manifest)

	for i := 0; i+1 < len(operations.Content); i += 2 {
		key := operations.Content[i]
		if key.Kind != yaml.ScalarNode || key.Value != name {
			continue
		}

		// The far end is the line before whatever is written next: the
		// following Operation's own range, which begins at its documenting
		// comment rather than at its key, or — for the Operation authored
		// last — the end of the operations: block.
		last := lines.blockEnd(key.Line, indentOf(key))
		if i+2 < len(operations.Content) {
			next := operations.Content[i+2]
			last = lines.headStart(next.Line, indentOf(next)) - 1
		}
		return lines.through(lines.headStart(key.Line, indentOf(key)),
			lines.trimTrailingBlanks(key.Line, last)), true
	}
	return "", false
}

// indentOf is how far a node is written in, in spaces, which is what decides
// what a line belongs to: the members of the operations: block are the lines
// indented past it, and a comment heads a key where it is written at that key's
// own indentation. The parse tree numbers columns from 1.
func indentOf(n *yaml.Node) int { return n.Column - 1 }

// sourceLines is a file's own bytes indexed by the lines they occupy, which is
// what a range stated in the parse tree's line numbers is resolved against. It
// holds offsets rather than copies: every answer this file gives is a slice of the
// bytes that came in, so no line is ever rebuilt from parts and no line ending
// is normalised on the way through.
type sourceLines struct {
	source []byte
	// starts holds the offset each line begins at, so line n — numbered from
	// 1, as the parse tree numbers them — begins at starts[n-1]. A file
	// ending in a newline has no further line: the terminator belongs to the
	// line it ends.
	starts []int
}

func readSourceLines(source []byte) sourceLines {
	starts := []int{0}
	for at, b := range source {
		if b == '\n' && at+1 < len(source) {
			starts = append(starts, at+1)
		}
	}
	return sourceLines{source: source, starts: starts}
}

func (l sourceLines) count() int { return len(l.starts) }

// text is line n with whatever terminator it carries, and "" for a line number
// outside the file — which is what a parse tree pointing past the bytes it came
// from would ask for, and a fault this file may not panic on.
func (l sourceLines) text(n int) string {
	if n < 1 || n > l.count() {
		return ""
	}
	if n == l.count() {
		return string(l.source[l.starts[n-1]:])
	}
	return string(l.source[l.starts[n-1]:l.starts[n]])
}

// headStart is where the range documenting the thing on line n begins: n
// itself, or the first of the comment lines standing immediately above it at
// its own indentation. Comments are permitted on any line and rendered verbatim
// in place (§3), and a comment above a key documents what the key declares — so
// it is part of that Operation's source and part of no other's.
//
// Two things end the run, and both are the difference between a comment that
// documents this key and one that belongs to what came before. A blank line: a
// comment separated from a key by one is not standing above it. And a
// difference in indentation: a comment written further in than the key is
// inside the block above it — the note an author leaves at the end of an
// Operation's body — and taking it as a head comment would move a line out of
// one Operation's source and into the next one's, or, where the next thing is
// written at the top level, out of the Manifest's rendering altogether.
func (l sourceLines) headStart(n, indent int) int {
	for n > 1 {
		above := l.text(n - 1)
		if !strings.HasPrefix(strings.TrimSpace(above), "#") || leadingSpaces(above) != indent {
			break
		}
		n--
	}
	return n
}

// blockEnd is the last line of the block an Operation on line from belongs to,
// where no Operation follows it: the block ends where the file dedents out of
// it, and the end of the file where it never does.
//
// It is measured against the Operation's own indentation rather than read off
// the key written after operations: in the parse tree, because what ends the
// block need not be a key at all. A comment written at the top level, a
// document marker, a second document — each is outside the block by its
// indentation alone, and none of them is a line of the Operation above.
//
// Blank lines are passed over rather than ending the block: an Operation's
// mapping may have one inside it, and the trailing ones are trimmed anyway.
func (l sourceLines) blockEnd(from, indent int) int {
	for n := from + 1; n <= l.count(); n++ {
		text := l.text(n)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if leadingSpaces(text) < indent {
			return n - 1
		}
	}
	return l.count()
}

// leadingSpaces is how far a line is written in. Spaces alone: YAML indents
// with spaces, and a line indented with a tab is a strict-yaml-violation the
// loader has already reported (§3, ADR-0023).
func leadingSpaces(text string) int {
	return len(text) - len(strings.TrimLeft(text, " "))
}

// trimTrailingBlanks is the range's far end with the blank lines taken off it:
// the blank line an author leaves between two Operations separates them and
// belongs to neither, so a range ends at the last line the Operation wrote.
//
// key is the floor, and it is what stops the trimming from eating the
// Operation's own key line. It is also the answer where last is below it, which
// is the degenerate reading of an Operation authored in flow style,
// `operations: {read: {…}}`: the key shares its line with whatever else was
// written there, so the range is that one line. It is the line declaring the
// Operation and it is still the file's own bytes — but where a second Operation
// was written on it, that Operation's declaration is on it too, which is the
// cost of asking for the lines of something authored without any.
func (l sourceLines) trimTrailingBlanks(key, last int) int {
	if last < key {
		return key
	}
	if last > l.count() {
		last = l.count()
	}
	for last > key && strings.TrimSpace(l.text(last)) == "" {
		last--
	}
	return last
}

// through is the bytes of lines first to last, terminators included: the range
// is copied out of the file rather than rebuilt from its lines, so nothing here
// can normalise a line ending or invent a final newline the file never had.
func (l sourceLines) through(first, last int) string {
	end := len(l.source)
	if last < l.count() {
		end = l.starts[last]
	}
	return string(l.source[l.starts[first-1]:end])
}
