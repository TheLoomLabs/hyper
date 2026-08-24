package artefact

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// What the range touched: the working tree's lines the review's change column
// marks, and the line each removed line is cited at (§8, ADR-0057, issue #168).
//
// **A review renders the working tree and no removed lines**, so every line
// number here is the working tree's — counted from one over every line of the
// file including blank ones — and **a removed line has no number, having no
// line**. What a deletion is cited at is the anchor above it: the opening line
// of the nearest enclosing structure, a removed `bound:` to its Step's `- id:`
// and a removed Step to `steps:`, which is the line carrying the subject a flag
// cites anyway and the only anchor that is not whatever text happens to sit
// adjacent (§8).
//
// It reads the two sides as lines and the working tree as a parsed artefact,
// because those are two questions: *which lines moved* is a fact about text and
// *what encloses a line that is gone* is a fact about the shape the text is
// written in. Nothing here judges either side — a review does not run `check`,
// and what is wrong with the artefact is `check`'s to report (ADR-0064).

// Touched is one reading of the range over one artefact.
type Touched struct {
	// Lines are the working tree's lines the change column marks: a line
	// whose content differs from the baseline, a line that is new, and the
	// line a deletion anchors to. **One mark and not three** — the gutter
	// marks and does not classify, and a direction is `FLAGS`' text (§8).
	Lines map[int]bool
	// Anchors is each removed line's own number in the baseline against the
	// working-tree line its deletion is cited at. It is what lets a flag
	// about a fact the working tree no longer carries cite a line the gutter
	// marked, which is the whole of what a citation is for (§8, §12).
	Anchors map[int]int
}

// Marked reports whether the change column marks that line.
func (t Touched) Marked(line int) bool { return t.Lines[line] }

// ReadTouched reads the range over one artefact: the baseline's lines, the
// working tree's, and the working tree as `hyper` parsed it.
//
// The parse is the working tree's alone. The baseline is read as text here
// because that is the whole of what the column needs, and a baseline that will
// not parse still marks the lines that moved — which is the same discipline the
// header holds one member over: an absence is named where it is, and nothing
// downstream of it is silently dropped (§8, ADR-0064).
func ReadTouched(baseline, working []string, root *yaml.Node) Touched {
	touched := Touched{Lines: map[int]bool{}, Anchors: map[int]int{}}

	kept := commonLines(baseline, working)
	matched := make(map[int]bool, len(kept))
	for _, pair := range kept {
		matched[pair[1]] = true
	}
	for i := range working {
		if !matched[i] {
			touched.Lines[i+1] = true
		}
	}

	structures := readStructures(root, working)
	// The gaps between what survived, each read for the two edits it holds.
	// A removed line standing against an added one is a **modification** —
	// the added line is the line whose content differs from the baseline, it
	// carries the mark, and the removed line is cited there; §8's own worked
	// example is one, `bound: 3` → `bound: 5` marking the `bound:` line and
	// not the Step above it. A removed line with no added line against it is
	// a **deletion**, and that is the one the enclosing structure answers
	// for.
	//
	// Which is which inside one gap is the count: a gap removing three lines
	// and adding one rewrote the first and deleted the other two, so the
	// pairing runs from the top and the surplus falls through. A gap that
	// paired everything it removed would be *whatever text happens to sit
	// adjacent* standing in for the anchor, which is the reading §8 names
	// and refuses (§8).
	for _, gap := range gapsBetween(kept, len(baseline), len(working)) {
		paired := min(gap.to-gap.from, gap.addedTo-gap.added)
		for i := 0; i < paired; i++ {
			touched.Anchors[gap.from+1+i] = gap.added + 1 + i
		}
		if gap.from+paired >= gap.to {
			continue
		}
		// The surplus stands after everything the gap added, which is
		// where a reader looking for what is no longer written has just
		// finished reading.
		after := gap.after
		if gap.addedTo > gap.added {
			after = gap.addedTo
		}
		anchor := anchorFor(baseline[gap.from+paired:gap.to], structures, after)
		if anchor == 0 {
			continue
		}
		touched.Lines[anchor] = true
		for line := gap.from + paired + 1; line <= gap.to; line++ {
			touched.Anchors[line] = anchor
		}
	}
	return touched
}

// gap is one edit between two lines both sides kept: the baseline lines it
// removed, the working lines it added, and the working line it stands after.
//
// Both runs are half-open ranges of their own side's indices, so a gap that
// removed nothing has from equal to to and is not a gap at all.
type gap struct {
	from, to       int
	added, addedTo int
	after          int
}

// gapsBetween is the edits read off what survived: everything between one
// matched pair and the next, on both sides at once.
//
// It answers only the gaps that removed something. A gap that added lines and
// removed none is already marked line by line above — every added line is a
// line that is new — and has nothing to be cited at.
func gapsBetween(kept [][2]int, baseline, working int) []gap {
	var gaps []gap
	nextBaseline, nextWorking, precedes := 0, 0, 0
	edit := func(toBaseline, toWorking int) {
		if toBaseline <= nextBaseline {
			return
		}
		gaps = append(gaps, gap{
			from: nextBaseline, to: toBaseline,
			added: nextWorking, addedTo: toWorking,
			after: precedes,
		})
	}
	for _, pair := range kept {
		edit(pair[0], pair[1])
		nextBaseline, nextWorking, precedes = pair[0]+1, pair[1]+1, pair[1]+1
	}
	edit(baseline, working)
	return gaps
}

// commonLines is the longest run of lines the two sides share in order, as the
// pairs of indices it matches — a line matched here is a line neither added nor
// removed, and everything else is one or the other.
//
// The common prefix and suffix are taken first and the table is built over what
// is left. An artefact is edited a line at a time far more often than it is
// rewritten, so the ordinary case costs a scan of the file and the table is
// what a rewrite falls back to.
func commonLines(a, b []string) [][2]int {
	head := 0
	for head < len(a) && head < len(b) && a[head] == b[head] {
		head++
	}
	tail := 0
	for tail < len(a)-head && tail < len(b)-head && a[len(a)-1-tail] == b[len(b)-1-tail] {
		tail++
	}

	ma, mb := a[head:len(a)-tail], b[head:len(b)-tail]
	// table[i][j] is the length of the longest common run of ma[i:] and
	// mb[j:], filled from the end so that the walk below reads forwards.
	table := make([][]int, len(ma)+1)
	for i := range table {
		table[i] = make([]int, len(mb)+1)
	}
	for i := len(ma) - 1; i >= 0; i-- {
		for j := len(mb) - 1; j >= 0; j-- {
			switch {
			case ma[i] == mb[j]:
				table[i][j] = table[i+1][j+1] + 1
			case table[i+1][j] >= table[i][j+1]:
				table[i][j] = table[i+1][j]
			default:
				table[i][j] = table[i][j+1]
			}
		}
	}

	pairs := make([][2]int, 0, head+tail)
	for i := 0; i < head; i++ {
		pairs = append(pairs, [2]int{i, i})
	}
	for i, j := 0, 0; i < len(ma) && j < len(mb); {
		switch {
		case ma[i] == mb[j]:
			pairs = append(pairs, [2]int{head + i, head + j})
			i, j = i+1, j+1
		case table[i+1][j] >= table[i][j+1]:
			i++
		default:
			j++
		}
	}
	for k := 0; k < tail; k++ {
		pairs = append(pairs, [2]int{len(a) - tail + k, len(b) - tail + k})
	}
	return pairs
}

// structure is one collection in the working tree as an anchor reads it: the
// line a deletion inside it is cited at, the column its members are written at,
// and whether its members are sequence entries or mapping keys.
//
// The opening line is the key naming the collection rather than the collection
// itself, which is why a removed Step is cited at `steps:` and not at the first
// Step that survived it: the line a reader edits to put the Step back is the one
// that names the sequence (§8).
type structure struct {
	opening  int
	indent   int
	sequence bool
}

// readStructures is the working tree's collections, innermost last within each
// branch and in the order the walk reaches them.
//
// A collection's opening line is the key that names it, and a sequence entry
// that is itself a collection opens on its own entry line — which is what makes
// `- id: retire` the anchor of everything written beneath it and `steps:` the
// anchor of the entry itself.
func readStructures(root *yaml.Node, working []string) []structure {
	var found []structure
	var walk func(node *yaml.Node, opening int)
	walk = func(node *yaml.Node, opening int) {
		if node == nil || len(node.Content) == 0 || opening < 1 || opening > len(working) {
			return
		}
		switch node.Kind {
		case yaml.MappingNode:
			found = append(found, structure{opening: opening, indent: node.Content[0].Column - 1})
			for i := 0; i+1 < len(node.Content); i += 2 {
				walk(node.Content[i+1], node.Content[i].Line)
			}
		case yaml.SequenceNode:
			found = append(found, structure{opening: opening, indent: node.Content[0].Column - 1, sequence: true})
			for _, entry := range node.Content {
				walk(entry, entry.Line)
			}
		}
	}
	walk(root, rootLine(root))
	return found
}

// rootLine is the artefact's own opening line: the line its first top-level key
// is written on, which is line 1 on every artefact that opens with one and the
// first line of content on one that opens with a comment. It is what a deletion
// of a top-level key is cited at, the document being the structure that
// encloses it.
func rootLine(root *yaml.Node) int {
	if root == nil || len(root.Content) == 0 {
		return 0
	}
	return root.Content[0].Line
}

// anchorFor is the line one deletion is cited at: the opening line of the
// nearest enclosing structure, read off the removed text's own shape.
//
// The shape is two facts and both are read from the first removed line that
// says anything — the column its content stands at, and whether it is a
// sequence entry. Those two name the structure the removed line was a member
// of: a `bound:` at a Step's own column is that Step's mapping, and a `- id:`
// at the same column is the sequence `steps:` names, the entry marker being the
// whole of what separates them.
//
// Where the removed text names no such structure — a run of blank lines, a
// column nothing in the working tree is written at — the nearest structure
// above the deletion answers, and where there is none at all the artefact's own
// first line does. What that refuses is a deletion with nowhere to be cited:
// every flag cites a line the gutter marked, and a citation of 0 is no
// citation (§8, §12).
func anchorFor(removed []string, structures []structure, after int) int {
	indent, sequence, legible := removedShape(removed)
	nearest := 0
	for _, s := range structures {
		if s.opening > after || s.opening <= nearest {
			continue
		}
		if legible && (s.indent != indent || s.sequence != sequence) {
			continue
		}
		nearest = s.opening
	}
	if nearest == 0 && legible {
		return anchorFor(nil, structures, after)
	}
	if nearest == 0 && len(structures) > 0 {
		return structures[0].opening
	}
	return nearest
}

// removedShape is where a deletion's content stood: the column it is written
// at, whether it opens a sequence entry, and whether anything was legible there
// at all.
//
// It reads the first line that carries content, blank lines standing for
// nothing on either side of an edit. The column counts the entry markers in,
// so that a `- id:` two spaces in stands at the column its own first key does —
// which is the column YAML gives that key and the one a structure is measured
// by.
func removedShape(removed []string) (indent int, sequence, legible bool) {
	for _, line := range removed {
		text := strings.TrimRight(line, " \t")
		if strings.TrimSpace(text) == "" {
			continue
		}
		indent = len(text) - len(strings.TrimLeft(text, " "))
		rest := text[indent:]
		for strings.HasPrefix(rest, "- ") {
			indent, rest, sequence = indent+2, rest[2:], true
		}
		return indent, sequence, true
	}
	return 0, false, false
}
