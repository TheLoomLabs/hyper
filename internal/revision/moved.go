package revision

import (
	"strconv"
	"strings"
)

// What git says moved between two revisions (§8, §12, issue #171).
//
// It is this package's third reading of the code branch, and it is the one the
// Comparison's catch-all row names its own evidence for: `N other lines
// changed · git diff <rev> <rev>`. **The unit is the command's** — added and
// removed as git counts them, a modified line being two — because the row names
// the command, and any other unit makes the row disagree with its own evidence.
//
// The lines themselves come back beside the count because the row is *other*
// lines: `hyper` maps each classed fact to its lines at both revisions and
// subtracts them, so the enumeration above and the count beneath it sum to the
// whole rather than overlap. A count with no line numbers behind it could not
// be subtracted from without guessing which lines a fact occupied.

// Moved is what one `git diff` said: how many lines moved over the file set the
// caller wanted, and which lines they were at each of the two revisions.
//
// The two line sets are keyed by repository path and are separate because a
// fact occupies different lines at the two revisions — a `bound:` that moved
// down four lines is one line at each end, and subtracting one for both would
// leave the catch-all reporting a line it never counted.
type Moved struct {
	// Count is git's own total: every `+` and every `-` line of every hunk
	// of every wanted path.
	Count int
	// Before and After are the lines each path's hunks touched, at the
	// first revision and at the second. A path with no hunk is absent from
	// both, which is the ordinary absence: a file that did not move has no
	// line that did.
	Before, After map[string]map[int]bool
}

// Between answers what moved between two revisions over the paths wanted.
//
// The selection is the caller's, exactly as it is for ArtefactsAt one file
// over: what counts as an artefact is internal/repository's rule and this
// package's subject is git, and a listing filtered here would be the five
// artefact locations spelled in a second place. **The generated workflow falls
// outside that rule and is therefore out** — it is projected rather than
// authored and byte-exact against what `project` would write (ADR-0046), so a
// change in it is a `hyper` version move already in Provenance, a Procedure
// move already classed, or a hand-edit, and a hand-edit is `check`'s Refusal
// rather than a row here.
//
// **A revision this clone does not hold is answered and never errored**, which
// is this package's rule wherever it names an object somebody else's Run
// recorded: what the caller does with it is render the absence §8 names, and
// the count is the part that needed the bytes (ADR-0071).
//
// Renames are off, so a moved file is a deletion and an addition and its lines
// are counted as git counts them there. Following a rename would report one
// path's lines under another path's name, which is exactly the map the caller
// subtracts a fact's own lines out of.
func Between(repoRoot, before, after string, wanted func(path string) bool) (Moved, bool, error) {
	git := repository(repoRoot)
	for _, commit := range []string{before, after} {
		resolved, err := git.text("rev-parse", "--verify", "--quiet", commit+"^{commit}")
		if err != nil || resolved == "" {
			return Moved{}, false, nil
		}
	}

	// `core.quotePath=false` so that a path outside ASCII arrives as itself
	// rather than as git's escapes, and `--unified=0` so that every line a
	// hunk names is a line that moved: with context the hunk header would
	// count lines that stood still, and this count is the row's own evidence.
	patch, err := git.run(
		"-c", "core.quotePath=false", "diff", "--no-color", "--no-ext-diff",
		"--no-renames", "--unified=0", before, after, "--",
	)
	if err != nil {
		return Moved{}, false, err
	}
	return readPatch(string(patch), wanted), true, nil
}

// readPatch reads one unified diff into the count and the two line sets.
//
// The path is read off the `---` and `+++` lines rather than out of the
// `diff --git a/x b/x` header, which names two paths on one line and cannot be
// cut where either of them holds the separator. A creation names `/dev/null` on
// one of the two and the other is the path.
//
// A file whose patch this cannot read contributes nothing rather than half of
// itself: what a caller does with the count is subtract each classed row's own
// lines from it, and a path whose lines are unknown would have a fact's lines
// subtracted out of a count they were never in.
func readPatch(patch string, wanted func(path string) bool) Moved {
	moved := Moved{Before: map[string]map[int]bool{}, After: map[string]map[int]bool{}}
	before, after, path := "", "", ""
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			before, after, path = "", "", ""
		case strings.HasPrefix(line, "--- "):
			before = patchPath(strings.TrimPrefix(line, "--- "), "a/")
		case strings.HasPrefix(line, "+++ "):
			after = patchPath(strings.TrimPrefix(line, "+++ "), "b/")
			if path = after; path == "" {
				path = before
			}
			if !wanted(path) {
				path = ""
			}
		case strings.HasPrefix(line, "@@ "):
			if path == "" {
				continue
			}
			moved.readHunk(path, line)
		}
	}
	return moved
}

// readHunk reads one `@@ -l,s +l,s @@` header: the lines it removed at the
// first revision and the lines it added at the second.
//
// The header is the whole of the reading and the body is not read at all. Under
// `--unified=0` a hunk's two spans *are* its `-` and `+` lines, and git's own
// arithmetic is what the catch-all row's unit is defined as.
func (m *Moved) readHunk(path, header string) {
	fields := strings.Fields(header)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return
	}
	m.Count += mark(m.Before, path, fields[1][1:])
	m.Count += mark(m.After, path, fields[2][1:])
}

// mark reads one side of a hunk header — `l` or `l,s` — records the lines it
// names against that path, and answers how many there were.
//
// **A span of zero names no line** and is the one form that has to be told from
// a span of one: a hunk that only added lines writes `-l,0`, and the `l` there
// is the line the addition sits after rather than a line that moved. Such a
// side records nothing at all, so a path that moved on one side only is absent
// from the other — the ordinary absence, and what a caller reads as *no line of
// that file moved there*.
func mark(sides map[string]map[int]bool, path, field string) int {
	start, length := field, "1"
	if at, count, split := strings.Cut(field, ","); split {
		start, length = at, count
	}
	first, firstErr := strconv.Atoi(start)
	held, heldErr := strconv.Atoi(length)
	if firstErr != nil || heldErr != nil || held <= 0 {
		return 0
	}

	lines, marked := sides[path]
	if !marked {
		lines = map[int]bool{}
		sides[path] = lines
	}
	for line := first; line < first+held; line++ {
		lines[line] = true
	}
	return held
}

// patchPath is one path off a `---` or `+++` line: the prefix off, and "" for
// the `/dev/null` a creation or a deletion names.
func patchPath(field, prefix string) string {
	field = strings.TrimSpace(field)
	if field == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(field, prefix)
}
