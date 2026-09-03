package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// What these cases hold, and the defect that bought them.
//
// `docs/adr/` is the largest corpus in this repository and, until ADR-0140, the
// only significant subtree with no `README.md` — against CONTRIBUTING's own rule
// that a corpus documents itself. What the README pointed at it with was the
// sentence *ninety-odd records of why*, written once against a corpus that had
// since reached 139 and recomputed by nobody.
//
// **Neither half of that was checkable.** A count in prose is a number a human
// maintains, and a directory nothing indexes has no claim to be wrong about. A
// list of links is a different thing: it can be compared against the directory
// it claims to cover, and a record added without a line in it is then a red
// suite rather than a page that quietly stops being true.
//
// So the index carries three claims and these cases hold all three — the list is
// the directory, every link in it resolves, and the count it states is the
// number of files. None of them reads an ADR's contents past its heading, which
// is deliberate: what is fenced here is that the corpus is navigable, not what
// any record says.

// adrRecord is one ADR as both halves name it: the file on disk, and the title
// its own `# ` heading gives it. The index restates the second, so the second is
// worth comparing — a heading edited without the index following it is the same
// class of drift as a record added without a line.
type adrRecord struct {
	number string
	title  string
	file   string
}

// adrRecordLine is the shape of a line in the index's *Every record* list. It is
// strict on purpose: the list is generated from the directory, and a line a
// human hand-wrote into a different shape is a line the next generation will
// silently drop.
var adrRecordLine = regexp.MustCompile(`^- \*\*(\d{4})\*\* · \[(.+)\]\((\d{4}-[a-z0-9-]+\.md)\)$`)

// adrLink is any link to a record, anywhere on the page — the reading paths and
// *Start here* are curated by hand and are where a typo lands.
var adrLink = regexp.MustCompile(`\]\((\d{4}-[a-z0-9-]+\.md)\)`)

// adrCount is the one number the page states in prose. It is held rather than
// removed: a count nothing checks is what ADR-0140 is about, and a count a case
// reads is a fact like any other.
var adrCount = regexp.MustCompile("\\*\\*This corpus holds (\\d+) records\\*\\*, numbered `(\\d{4})`–`(\\d{4})`")

// adrDir is where the corpus lives, and adrIndex is the page under test.
func adrDir(t *testing.T) string { t.Helper(); return filepath.Join(root(t), "docs", "adr") }
func adrIndex(t *testing.T) string {
	t.Helper()
	page, err := os.ReadFile(filepath.Join(adrDir(t), "README.md"))
	if err != nil {
		t.Fatalf("reading the ADR index: %v — docs/adr/README.md is the way into the corpus and ADR-0140 requires it to stand", err)
	}
	return string(page)
}

// onDisk is every record in the directory, in the order the numbering fixes.
// The filenames are zero-padded to four digits, so lexical order is numeric
// order and no case has to sort them a second way.
func onDisk(t *testing.T) []adrRecord {
	t.Helper()
	entries, err := os.ReadDir(adrDir(t))
	if err != nil {
		t.Fatal(err)
	}

	var records []adrRecord
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") || len(name) < 5 || !isFourDigits(name[:4]) {
			continue
		}
		records = append(records, adrRecord{number: name[:4], title: headingOf(t, filepath.Join(adrDir(t), name)), file: name})
	}
	return records
}

// isFourDigits is what separates a record from `README.md`, and it is spelled
// out rather than folded into the link pattern because the directory walk and
// the page parse have to agree on which files are records at all.
func isFourDigits(s string) bool {
	if len(s) != 4 {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// headingOf is a record's own title: the first line, which every ADR opens with
// as `# `. A file that does not is a defect this case is entitled to name.
func headingOf(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first, _, _ := strings.Cut(string(body), "\n")
	if !strings.HasPrefix(first, "# ") {
		t.Fatalf("%s opens with %q, and every ADR opens with a `# ` heading — the index quotes that heading as the record's title", filepath.Base(path), first)
	}
	return strings.TrimSpace(strings.TrimPrefix(first, "# "))
}

// indexed is what the index's *Every record* list claims, in the order it claims
// it. Only lines under that heading count: the reading paths above it link the
// same files and are explicitly not a partition, so counting them would make
// the curated half a completeness obligation it does not have.
func indexed(t *testing.T) []adrRecord {
	t.Helper()
	page := adrIndex(t)
	_, list, found := strings.Cut(page, "\n## Every record\n")
	if !found {
		t.Fatal("the ADR index has no `## Every record` section — that list is the complete one, and the reading paths above it are explicitly not")
	}

	var records []adrRecord
	for _, line := range strings.Split(list, "\n") {
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		match := adrRecordLine.FindStringSubmatch(line)
		if match == nil {
			t.Errorf("the ADR index carries a list line the generator would not have written:\n\t%s\nwant the shape `- **0001** · [Title](0001-slug.md)`", line)
			continue
		}
		records = append(records, adrRecord{number: match[1], title: match[2], file: match[3]})
	}
	return records
}

// TestDocs_TheADRIndexNamesEveryRecord is the case the *ninety-odd* sentence
// could not have had. It compares the index's complete list against the
// directory, entry for entry — the record's number, the title its own heading
// gives it, and the file the link resolves to — so that adding an ADR without
// indexing it, or renaming one without following it here, fails the suite
// (ADR-0140).
func TestDocs_TheADRIndexNamesEveryRecord(t *testing.T) {
	disk, page := onDisk(t), indexed(t)

	if len(disk) != len(page) {
		t.Errorf("docs/adr/ holds %d records and the index lists %d", len(disk), len(page))
	}

	for i := 0; i < len(disk) && i < len(page); i++ {
		switch {
		case disk[i].file != page[i].file:
			t.Errorf("index position %d links %s, and the directory has %s there — the list is in numeric order and the directory is the authority", i+1, page[i].file, disk[i].file)
		case disk[i].number != page[i].number:
			t.Errorf("%s is listed as **%s** and its filename says %s", disk[i].file, page[i].number, disk[i].number)
		case disk[i].title != page[i].title:
			t.Errorf("%s is listed as %q and its own heading reads %q — the index quotes the heading and a heading that moved has to be followed here", disk[i].file, page[i].title, disk[i].title)
		}
	}

	// The two ends, named rather than left to the loop, because a record
	// missing from one end is the likeliest shape of this failing and the
	// positional message above would name the wrong file for all of them.
	for _, missing := range difference(disk, page) {
		t.Errorf("docs/adr/%s is in the corpus and not in the index — add it to `## Every record`", missing)
	}
	for _, extra := range difference(page, disk) {
		t.Errorf("the index lists %s and docs/adr/ has no such record", extra)
	}
}

// TestDocs_TheADRIndexLinksNothingThatIsNotThere covers the half of the page a
// human writes. *Start here* and the reading paths are curated, non-exhaustive
// and full of hand-typed slugs of the length this project's filenames run to,
// which is exactly where a typo goes unnoticed — a dead link on a page whose job
// is being the way in.
func TestDocs_TheADRIndexLinksNothingThatIsNotThere(t *testing.T) {
	present := map[string]bool{}
	for _, record := range onDisk(t) {
		present[record.file] = true
	}

	seen := map[string]bool{}
	for _, match := range adrLink.FindAllStringSubmatch(adrIndex(t), -1) {
		target := match[1]
		if present[target] || seen[target] {
			seen[target] = true
			continue
		}
		seen[target] = true
		t.Errorf("the ADR index links %s and no such file exists in docs/adr/", target)
	}
}

// TestDocs_TheADRIndexStatesTheCountItHolds keeps the number in the prose and
// makes it a fact. The sentence it replaces — *ninety-odd records* — was wrong by
// forty-odd and nothing could say so; this one is read off the same directory
// the list is compared against, so it is wrong for exactly one commit.
func TestDocs_TheADRIndexStatesTheCountItHolds(t *testing.T) {
	disk := onDisk(t)
	if len(disk) == 0 {
		t.Fatal("docs/adr/ holds no records, which is not a state this repository can be in")
	}

	match := adrCount.FindStringSubmatch(adrIndex(t))
	if match == nil {
		t.Fatal("the ADR index states no count — want a sentence of the shape **This corpus holds N records**, numbered `0001`–`0NNN`")
	}

	if got, want := match[1], strconv.Itoa(len(disk)); got != want {
		t.Errorf("the ADR index says it holds %s records and docs/adr/ holds %s", got, want)
	}
	if got, want := match[2], disk[0].number; got != want {
		t.Errorf("the ADR index says the corpus starts at %s and the first record is %s", got, want)
	}
	if got, want := match[3], disk[len(disk)-1].number; got != want {
		t.Errorf("the ADR index says the corpus ends at %s and the last record is %s", got, want)
	}

	// *With no gaps* is the other half of that sentence, and it is the claim
	// the numbering rests on: the number is the order a record was written
	// in, so a hole in the sequence is a record that went missing rather
	// than a number nobody used.
	for i, record := range disk {
		if want := pad(i + 1); record.number != want {
			t.Fatalf("the %d%s record in docs/adr/ is numbered %s, want %s — the index claims the sequence has no gaps", i+1, ordinal(i+1), record.number, want)
		}
	}
}

// difference is the records in the first list that the second does not name, by
// file. It exists so the two directions read as two sentences rather than one
// symmetric-difference dump a reader has to work out the direction of.
func difference(from, against []adrRecord) []string {
	named := map[string]bool{}
	for _, record := range against {
		named[record.file] = true
	}

	var missing []string
	for _, record := range from {
		if !named[record.file] {
			missing = append(missing, record.file)
		}
	}
	return missing
}

func pad(n int) string {
	s := strconv.Itoa(n)
	return strings.Repeat("0", max(0, 4-len(s))) + s
}

func ordinal(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return "th"
	case n%10 == 1:
		return "st"
	case n%10 == 2:
		return "nd"
	case n%10 == 3:
		return "rd"
	default:
		return "th"
	}
}

// The second document this repository publishes rather than keeps: a release's
// body.
//
// `release.yml` used to publish with `--generate-notes`, which lists the commit
// subjects since the last tag. For this repository that is twenty-odd lines
// written for a reviewer reading a diff — every one of them accurate, and none
// of them what a person deciding whether to install this needs to read. The body
// is now a file in the tree the tag names, so it is reviewed in the same diff as
// the change it describes (ADR-0141).
//
// **What that trades is one failure for another.** A generated body cannot be
// missing; a written one can, and a tag pushed without one would have published
// a release with an empty description. So the workflow refuses, and these cases
// hold that it still does — a guard nobody exercises until the day it matters is
// a guard worth a case.

// releaseNotesDir is where a release's body lives, relative to the repository
// root, and `notesPath` is how the workflow spells the same thing. They are
// compared rather than assumed: the workflow interpolates the tag, and a
// directory renamed on one side alone is a release that fails at the last step.
const (
	releaseNotesDir = "docs/build/release-notes"
	notesPath       = releaseNotesDir + "/$GITHUB_REF_NAME.md"
)

// publishStep is the one step of `release.yml` that creates the release. It is
// found by what it runs rather than by its name, for the reason every case in
// this package reads steps that way: a name is a label and `run` is the act.
func publishStep(t *testing.T) string {
	t.Helper()

	var found []string
	for _, run := range runsOf(workflowOf(t, "release.yml")) {
		if strings.Contains(run, "gh release create") {
			found = append(found, run)
		}
	}
	if len(found) != 1 {
		t.Fatalf("release.yml runs `gh release create` in %d steps, want exactly 1 — two of them is two answers to what a release publishes", len(found))
	}
	return found[0]
}

// TestDocs_AReleasePublishesNotesFromTheTree holds the mechanism: the body comes
// from a file this repository carries, at the path the notes actually live at,
// and GitHub's commit dump is not a fallback anywhere in the step. A release
// that quietly went back to `--generate-notes` would publish something plausible
// and wrong, which is the shape of defect nobody reports.
func TestDocs_AReleasePublishesNotesFromTheTree(t *testing.T) {
	step := publishStep(t)

	if !strings.Contains(step, "--notes-file") {
		t.Errorf("the publish step does not pass --notes-file; it runs:\n%s", step)
	}
	if strings.Contains(step, "--generate-notes") {
		t.Errorf("the publish step still passes --generate-notes, which publishes the commit subjects since the last tag; it runs:\n%s", step)
	}
	if !strings.Contains(step, notesPath) {
		t.Errorf("the publish step does not name %s, which is where this repository keeps a release's body; it runs:\n%s", notesPath, step)
	}
	if !strings.Contains(step, "exit 1") {
		t.Errorf("the publish step does not refuse a tag with no notes file — without that a tag pushed before the notes were written publishes a release with an empty body; it runs:\n%s", step)
	}

	if _, err := os.Stat(filepath.Join(root(t), releaseNotesDir)); err != nil {
		t.Fatalf("%s does not stand, and the publish step reads a file under it: %v", releaseNotesDir, err)
	}
}

// TestDocs_EveryReleaseNotesFileNamesItsOwnVersion is the copy-paste fence. The
// notes for one release are the obvious starting point for the next, and a file
// renamed without its contents following it is a release describing the one
// before it — accurate prose about the wrong bytes, which nothing else here
// could catch.
func TestDocs_EveryReleaseNotesFileNamesItsOwnVersion(t *testing.T) {
	directory := filepath.Join(root(t), releaseNotesDir)
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}

	var found int
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			t.Errorf("%s/%s is a directory, and the publish step reads this one as a flat set of files", releaseNotesDir, name)
			continue
		}
		if !strings.HasPrefix(name, "v") || !strings.HasSuffix(name, ".md") {
			t.Errorf("%s/%s is not named for a tag — the publish step reads `$GITHUB_REF_NAME.md`, and a tag here carries the `v`", releaseNotesDir, name)
			continue
		}
		found++

		body, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			t.Errorf("%s/%s is empty, and the publish step refuses an empty file rather than publishing one", releaseNotesDir, name)
			continue
		}

		// The version without the `v`, which is what every filename under
		// the tag carries and what the prose refers to the release as.
		version := strings.TrimSuffix(strings.TrimPrefix(name, "v"), ".md")
		if !strings.Contains(string(body), version) {
			t.Errorf("%s/%s never names %s — notes copied from an earlier release describe the wrong bytes", releaseNotesDir, name, version)
		}
	}

	if found == 0 {
		t.Errorf("%s holds no release notes, and the publish step reads one out of it for every tag", releaseNotesDir)
	}
}
