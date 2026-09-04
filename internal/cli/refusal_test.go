package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// **The caret excerpt shows the offending line in its own context, and the
// caret sits at the value** (§8, issue #169).
//
// The corpus drives the whole rendering — every case under `testdata/run/` that
// Refuses draws one, and `testdata/probe/a-host-outside-the-grant` draws one
// with no Run behind it. What is here is the two decisions no golden isolates:
// how far back the context reaches, and where on the cited line the caret goes.
//
// Both are decided from the file's own characters, so a table over one small
// file is what puts the cases beside each other. A golden shows one answer at a
// time and cannot show that the walk **stops at the enclosing key** — that fact
// is the difference between two cases, and the difference is what is being
// stated.

// excerptFixture is a file whose shape carries every case below: a top-level
// key with a nested mapping under it, a sequence entry deeper still, a key with
// no value on its line, and the first line of the file.
const excerptFixture = `kind: target-declaration
auth:
  token: {env: STAGING_TOKEN}
steps:
  - id: retire
    over:
      assets:
        - field: created_at
          older_than: 14d
    bound: 5
`

// TestReadExcerpt_TheContextAndTheCaret walks the two decisions cell by cell.
func TestReadExcerpt_TheContextAndTheCaret(t *testing.T) {
	root := t.TempDir()
	write(t, root, "artefact.yaml", excerptFixture)

	for _, c := range []struct {
		name  string
		line  int
		first int
		caret string
	}{
		{
			// The walk stops at the enclosing key: `auth:` is
			// shallower than `token:`, so the context is one line
			// and not two.
			name: "the context stops at the enclosing key", line: 3, first: 2,
			caret: "  token: {env: STAGING_TOKEN}\n         ^",
		},
		{
			// Nothing above `bound:` is shallower than it within
			// two lines, so the cap decides instead — and what the
			// two lines carry is the `over:` block the Bound is
			// about, which is the whole point of showing them.
			name: "the context is capped at two lines", line: 10, first: 8,
			caret: "    bound: 5\n           ^",
		},
		{
			// A line with no value on it — a key opening a block —
			// takes the caret at its first character, there being
			// no value for it to point at.
			name: "a key with no value takes the caret at the key", line: 7, first: 6,
			caret: "      assets:\n      ^",
		},
		{
			// The first line of a file has nothing above it, and
			// the walk answers the citation alone rather than
			// reaching past the start.
			name: "the first line of a file stands alone", line: 1, first: 1,
			caret: "kind: target-declaration\n      ^",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			excerpt := readExcerpt(root, "bound-exceeded", "artefact.yaml", c.line)
			if !excerpt.rendered() {
				t.Fatal("no excerpt was drawn")
			}
			if excerpt.first != c.first {
				t.Errorf("excerpt opens at line %d; want %d", excerpt.first, c.first)
			}
			cited := excerpt.lines[len(excerpt.lines)-1]
			drawn := cited + "\n" + strings.Repeat(" ", excerpt.column) + "^"
			if drawn != c.caret {
				t.Errorf("caret drawn as\n%s\nwant\n%s", drawn, c.caret)
			}
		})
	}
}

// TestReadExcerpt_DrawsNothing walks the citations that draw no caret at all.
// Each falls back to the coordinate as a `=` note, so what matters here is that
// none of them draws lines.
func TestReadExcerpt_DrawsNothing(t *testing.T) {
	root := t.TempDir()
	write(t, root, "artefact.yaml", excerptFixture)

	for _, c := range []struct {
		name string
		code string
		file string
		line int
	}{
		// The one code that renders no caret however readable its file
		// is: a Store file is evidence, and editing it is editing
		// evidence (ADR-0011). The path here is a real file so that
		// what the test states is the **rule** and not the absence.
		{"a Store file is evidence and never a caret", "store-schema-unsupported", "artefact.yaml", 3},
		{"a file the working tree does not hold", "bound-exceeded", "gone.yaml", 3},
		{"a check with no line to point at", "bound-exceeded", "artefact.yaml", 0},
		{"a line past the end of the file", "bound-exceeded", "artefact.yaml", 400},
	} {
		t.Run(c.name, func(t *testing.T) {
			if excerpt := readExcerpt(root, c.code, c.file, c.line); excerpt.rendered() {
				t.Fatalf("an excerpt was drawn: %q", excerpt.lines)
			}
		})
	}
}

// TestExcerpt_GlossesTheRelativeOperandsItRenders holds ADR-0034's rule at the
// place §8 puts it: the `=` note reads the **excerpt** where every other note
// reads the check, so an operand two lines above the citation is glossed and
// one the excerpt does not reach is not.
func TestExcerpt_GlossesTheRelativeOperandsItRenders(t *testing.T) {
	root := t.TempDir()
	write(t, root, "artefact.yaml", excerptFixture)
	started := time.Date(2026, 8, 6, 11, 3, 18, 0, time.UTC)

	glossed := readExcerpt(root, "bound-exceeded", "artefact.yaml", 10).glosses(started)
	if len(glossed) != 1 {
		t.Fatalf("the Bound's excerpt glossed %d operands; want the one it renders", len(glossed))
	}
	if want := "older_than: 14d resolved to 2026-07-23T11:03:18Z"; glossed[0].note() != want {
		t.Errorf("note is %q; want %q", glossed[0].note(), want)
	}

	// The `auth:` excerpt renders no relative operand, so it earns no gloss
	// — the note follows the text and not the code.
	if glossed := readExcerpt(root, "credential-absent", "artefact.yaml", 3).glosses(started); len(glossed) != 0 {
		t.Errorf("an excerpt with no relative operand glossed %d", len(glossed))
	}

	// A surface with no Run has no instant, and a gloss is derived
	// arithmetic rather than a claim: it renders where its supply is
	// (ADR-0063). This is the Probe.
	if glossed := readExcerpt(root, "bound-exceeded", "artefact.yaml", 10).glosses(time.Time{}); len(glossed) != 0 {
		t.Errorf("an excerpt with no Run behind it glossed %d", len(glossed))
	}
}

// TestRefusalRemedies_AreTheSetWithNoRemediationTable holds the one fact
// membership of that map decides twice: a code in it renders the remedy as its
// last `=` note and renders **no** `EDIT ONE OF` table, and a code outside it
// renders ADR-0001's note and a table. The two readings are one map so that
// they can never disagree about which set a code is in (§8).
func TestRefusalRemedies_AreTheSetWithNoRemediationTable(t *testing.T) {
	member := refusalRow{ErrorCode: "credential-absent", File: "targets/staging.yaml", Line: 12, Field: "auth.token"}
	if rows := remediationsFor(member, nil, time.Time{}); len(rows) != 0 {
		t.Errorf("a code whose remedy is not an edit rendered %d remediation rows", len(rows))
	}
	notes := member.notes(refusalPhase(member))
	if last := notes[len(notes)-1]; last != refusalRemedies["credential-absent"] {
		t.Errorf("last note is %q; want the remedy", last)
	}

	edit := refusalRow{ErrorCode: "unknown-key", File: "hyper.yaml", Line: 5, Field: "retain-forever"}
	if rows := remediationsFor(edit, nil, time.Time{}); len(rows) != 1 {
		t.Errorf("a code whose remedy is an edit rendered %d remediation rows; want 1", len(rows))
	}
	notes = edit.notes(refusalPhase(edit))
	if last := notes[len(notes)-1]; last != noBypassNote {
		t.Errorf("last note is %q; want ADR-0001's", last)
	}
}

// TestRefusalPhase_IsTheStepTheMemberCites walks §8's phase note over the fact
// that decides it: **the Step the member cites**, and the one code that cites a
// Step it did not reach.
//
// The corpus drives each arm — a Bound at an Expansion, a `check` re-run before
// Step 1 — and what is here is the pair beside each other, plus the exception
// no golden isolates: `secret-sink-absent` names every reachable Step whose
// Operation declares secret output, and names them **before Step 1**.
func TestRefusalPhase_IsTheStepTheMemberCites(t *testing.T) {
	step := 3
	for _, c := range []struct {
		name   string
		member refusalRow
		want   string
	}{
		{"a check that cites a Step was found at its Expansion",
			refusalRow{ErrorCode: "bound-exceeded", Step: &step}, phaseAtExpansion},
		{"a check that cites no Step declined before Step 1",
			refusalRow{ErrorCode: "credential-absent"}, phaseAtRunStart},
		{"the sink gate cites Steps it never reached",
			refusalRow{ErrorCode: "secret-sink-absent", Step: &step}, phaseAtRunStart},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := refusalPhase(c.member); got != c.want {
				t.Errorf("phase is %q; want %q", got, c.want)
			}
		})
	}
}

// write puts one file under root, and is here so the tables above read as
// tables.
func write(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
