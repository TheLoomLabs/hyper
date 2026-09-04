package artefact

import (
	"slices"
	"strings"
	"testing"
)

// widgetManifestWithComments is a Manifest authored the way a repository author
// writes one: three Operations, one of them documented by a comment above its
// key and another by a comment inside its body, blank lines between the blocks,
// and a top-level key after operations: — which is the block's end for the
// Operation authored last in it.
const widgetManifestWithComments = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    deadline: 30s
    http: {method: GET, host: "{from-target}", path: /widgets}

  # Archives the widget rather than deleting it: the API has no delete.
  # The Kind is mutate all the same — what it does to the asset is what counts.
  delete_widget:
    kind: mutate
    deadline: 30s
    # A POST, and not a DELETE, because the endpoint is /archive.
    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/archive"}

  create_widget:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http: {method: POST, host: "{from-target}", path: /widgets}


# The Auth scheme is written after the Operations in this Manifest.
auth:
  header: {name: Authorization, prefix: "Bearer "}
`

// linesBetween is the fixture's own lines, from the one that reads first
// through the one that reads last, extracted with the standard library. Every
// case here states the range it expects this way rather than as a second copy
// of the bytes: what the criterion asks is that the command hands back the
// file's own range, so the expectation has to be taken from the file by
// something other than the code under test (issue #114).
func linesBetween(t *testing.T, manifest, first, last string) string {
	t.Helper()
	all := strings.Split(manifest, "\n")
	from := slices.Index(all, first)
	if from < 0 {
		t.Fatalf("the fixture has no line %q", first)
	}
	// The far end is looked for from the near one, a line like
	// `stderr: $.stderr` reading identically in every Operation of the
	// built-in's Manifest.
	to := slices.Index(all[from:], last)
	if to < 0 {
		t.Fatalf("the fixture has no line %q after %q", last, first)
	}
	return strings.Join(all[from:from+to+1], "\n") + "\n"
}

// mustSource is the source of an Operation the fixture declares, the lookup
// having answered.
func mustSource(t *testing.T, manifest, name string) string {
	t.Helper()
	source, declared := OperationSource([]byte(manifest), parse(t, manifest), name)
	if !declared {
		t.Fatalf("OperationSource(%q) found nothing; the fixture declares it", name)
	}
	return source
}

// TestOperationSource_IsTheOperationsOwnLines is what the command exists for:
// an Operation in the middle of a block is written back from its key line
// through the last line of its mapping, and the range is the file's own bytes
// rather than a re-encoding of what they parsed to (§9, §12).
func TestOperationSource_IsTheOperationsOwnLines(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "list_widgets")
	want := linesBetween(t, widgetManifestWithComments,
		"  list_widgets:",
		`    http: {method: GET, host: "{from-target}", path: /widgets}`)

	if got != want {
		t.Errorf("source =\n%q\nwant\n%q", got, want)
	}
}

// TestOperationSource_EndsAtTheLineBeforeTheNextOperationsKey is the range's
// far end where another Operation follows: nothing of the next Operation is in
// it, its documenting comment included — a comment above a key documents the
// Operation it stands above (§3).
func TestOperationSource_EndsAtTheLineBeforeTheNextOperationsKey(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "list_widgets")

	for _, next := range []string{"delete_widget", "Archives the widget"} {
		if strings.Contains(got, next) {
			t.Errorf("source =\n%q\nwant nothing of the Operation that follows; it carries %q", got, next)
		}
	}
}

// TestOperationSource_IncludesACommentImmediatelyPrecedingTheKey: comments are
// permitted on any line and rendered verbatim in place, and a comment above an
// Operation documents that Operation (§3, issue #114).
func TestOperationSource_IncludesACommentImmediatelyPrecedingTheKey(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "delete_widget")
	want := linesBetween(t, widgetManifestWithComments,
		"  # Archives the widget rather than deleting it: the API has no delete.",
		`    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/archive"}`)

	if got != want {
		t.Errorf("source =\n%q\nwant\n%q", got, want)
	}
	if !strings.HasPrefix(got, "  # Archives the widget") {
		t.Errorf("source opens %q, want the comment above the key", strings.SplitN(got, "\n", 2)[0])
	}
}

// TestOperationSource_CarriesACommentInsideTheBodyVerbatimAndInPlace is the
// other half of the same rule: a comment between two keys of the Operation's
// own mapping is a line of the range like any other.
func TestOperationSource_CarriesACommentInsideTheBodyVerbatimAndInPlace(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "delete_widget")

	want := "    deadline: 30s\n    # A POST, and not a DELETE, because the endpoint is /archive.\n"
	if !strings.Contains(got, want) {
		t.Errorf("source =\n%q\nwant the body's comment in place: %q", got, want)
	}
}

// TestOperationSource_TheLastOperationInABlockEndsAtTheEndOfTheBlock is the
// range's far end where no Operation follows: it ends where the operations:
// block does, and not at a next key that is not there. The fixture writes a
// top-level auth: after the block, with a comment of its own above it, and
// neither is the last Operation's (issue #114).
func TestOperationSource_TheLastOperationInABlockEndsAtTheEndOfTheBlock(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "create_widget")
	want := linesBetween(t, widgetManifestWithComments,
		"  create_widget:",
		`    http: {method: POST, host: "{from-target}", path: /widgets}`)

	if got != want {
		t.Errorf("source =\n%q\nwant\n%q", got, want)
	}
	for _, beyond := range []string{"auth:", "The Auth scheme is written"} {
		if strings.Contains(got, beyond) {
			t.Errorf("source =\n%q\nwant nothing past the operations: block; it carries %q", got, beyond)
		}
	}
}

// TestOperationSource_TrimsTrailingBlankLines: the blank line an author leaves
// between two Operations belongs to neither, so the range ends at the last line
// the Operation wrote (issue #114).
func TestOperationSource_TrimsTrailingBlankLines(t *testing.T) {
	for _, name := range []string{"list_widgets", "delete_widget", "create_widget"} {
		got := mustSource(t, widgetManifestWithComments, name)
		if strings.HasSuffix(got, "\n\n") || strings.TrimSpace(got) == "" {
			t.Errorf("%s's source ends %q, want no trailing blank line", name, got[max(0, len(got)-8):])
		}
	}
}

// TestOperationSource_PreservesTheOriginalIndentation: the lines are written
// back unchanged, not dedented and not re-wrapped — a Manifest is written in
// the format the caller is expected to author Definitions in, so what comes
// back has to be authorable as it stands (§3, §9).
func TestOperationSource_PreservesTheOriginalIndentation(t *testing.T) {
	got := mustSource(t, widgetManifestWithComments, "delete_widget")

	for _, line := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("source carries the line %q; the Operation is authored two columns in", line)
		}
	}
	if !strings.Contains(got, "\n    kind: mutate\n") {
		t.Errorf("source =\n%q\nwant the mapping's own four-column members", got)
	}
}

// TestOperationSource_IsByteForByteTheFilesOwnRange is the criterion stated
// whole: the answer is a slice of the Manifest's bytes, so it is found in them
// at one offset and it is what the standard library extracts from the same
// range. A re-encoding that produced equivalent YAML would break the identity
// silently — manifest_digest would still be right, and the thing the reviewer
// read would no longer be the thing it covered (§12).
func TestOperationSource_IsByteForByteTheFilesOwnRange(t *testing.T) {
	manifest := []byte(widgetManifestWithComments)

	for _, name := range []string{"list_widgets", "delete_widget", "create_widget"} {
		got := mustSource(t, widgetManifestWithComments, name)
		if strings.Count(widgetManifestWithComments, got) != 1 {
			t.Errorf("%s's source occurs %d times in the Manifest, want the one range it was taken from",
				name, strings.Count(widgetManifestWithComments, got))
		}
		at := strings.Index(string(manifest), got)
		if at < 0 || string(manifest[at:at+len(got)]) != got {
			t.Errorf("%s's source is not a range of the file's bytes", name)
		}
	}
}

// TestOperationSource_ReadsTheBuiltInFromTheCompiledInBytes is the one Manifest
// a repository author cannot read any other way: the built-in shell Provider
// has no file, and its source is the same range taken over the bytes compiled
// into the binary (§12, ADR-0039).
func TestOperationSource_ReadsTheBuiltInFromTheCompiledInBytes(t *testing.T) {
	got := mustSource(t, BuiltinShellProviderYAML, "mutate_once")
	want := linesBetween(t, BuiltinShellProviderYAML, "  mutate_once:", "        exit_code: $.exit_code")

	if got != want {
		t.Errorf("source =\n%q\nwant\n%q", got, want)
	}
	if !strings.Contains(BuiltinShellProviderYAML, got) {
		t.Error("the built-in's source is not a range of the compiled-in bytes")
	}
	if strings.Contains(got, "mutate_skip_if_recorded") {
		t.Error("the range runs into the Operation that follows")
	}
}

// TestOperationSource_ANameMatchingNothingFindsNothing is the lookup's own
// answer, which the command turns into a usage error: matching is byte-exact
// over UTF-8 and case-sensitive against the Manifest's own key (§9, ADR-0060).
func TestOperationSource_ANameMatchingNothingFindsNothing(t *testing.T) {
	for _, name := range []string{"list_widget", "List_widgets", "", "operations"} {
		source, declared := OperationSource([]byte(widgetManifestWithComments), parse(t, widgetManifestWithComments), name)
		if declared || source != "" {
			t.Errorf("OperationSource(%q) = %q, %v; want nothing found", name, source, declared)
		}
	}
}

// TestOperationSource_AManifestWithNoLegibleOperationsBlockFindsNothing is
// ADR-0064's rule at this reader: what cannot be read has no source to hand
// back, and what is wrong with the Manifest is check's to name.
func TestOperationSource_AManifestWithNoLegibleOperationsBlockFindsNothing(t *testing.T) {
	for _, manifest := range []string{
		"kind: provider\nprovider: widget\n",
		"kind: provider\nprovider: widget\noperations: []\n",
		"kind: provider\nprovider: widget\noperations: read\n",
		"",
	} {
		source, declared := OperationSource([]byte(manifest), parse(t, manifest), "read")
		if declared || source != "" {
			t.Errorf("OperationSource over %q = %q, %v; want nothing found", manifest, source, declared)
		}
	}
}

// TestOperationSource_AFileNotEndingInANewlineKeepsItsLastLine: the range is
// bytes rather than lines, so an author whose editor wrote no final newline
// gets their last line back and no newline hyper invented for them.
func TestOperationSource_AFileNotEndingInANewlineKeepsItsLastLine(t *testing.T) {
	manifest := strings.TrimSuffix(`kind: provider
provider: widget
operations:
  read:
    kind: read
`, "\n")

	got := mustSource(t, manifest, "read")
	if want := "  read:\n    kind: read"; got != want {
		t.Errorf("source = %q, want %q", got, want)
	}
}

// manifestWithATrailingBodyComment writes the comment an author leaves at the
// end of an Operation's body — indented into that Operation's mapping, and
// standing immediately above the next thing written. Both of the Operations
// carry one, so both branches of the range's far end are exercised: the one
// followed by another Operation, and the one that ends the block.
const manifestWithATrailingBodyComment = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    # The deadline is the API's own, and the reason it is not 30s.
    deadline: 45s
  delete_widget:
    kind: mutate
    deadline: 30s
    # Archived rather than deleted: the API has no delete.
auth:
  header: {name: Authorization, prefix: "Bearer "}
`

// TestOperationSource_ACommentEndingABodyStaysWithTheOperationItIsWrittenIn is
// the other half of the comment rule, and the half indentation decides: a
// comment written inside an Operation's mapping is a line of that Operation
// however close it stands to what comes next. Only a comment written at the
// key's own indentation heads the key below it — the run that headStart walks
// is a documenting comment, and a comment indented into the block above is not
// one (§3, issue #114).
func TestOperationSource_ACommentEndingABodyStaysWithTheOperationItIsWrittenIn(t *testing.T) {
	followed := mustSource(t, manifestWithATrailingBodyComment, "list_widgets")
	if !strings.HasSuffix(followed, "    deadline: 45s\n") {
		t.Errorf("list_widgets's source =\n%q\nwant it to end at its own last line", followed)
	}
	if !strings.Contains(followed, "    # The deadline is the API's own, and the reason it is not 30s.\n") {
		t.Errorf("list_widgets's source =\n%q\nwant the comment written inside its body", followed)
	}

	ending := mustSource(t, manifestWithATrailingBodyComment, "delete_widget")
	if !strings.HasPrefix(ending, "  delete_widget:") {
		t.Errorf("delete_widget's source opens %q, want its own key line", strings.SplitN(ending, "\n", 2)[0])
	}
	if strings.Contains(ending, "The deadline is the API's own") {
		t.Errorf("delete_widget's source =\n%q\ncarries the comment ending the Operation above it", ending)
	}
	if !strings.HasSuffix(ending, "    # Archived rather than deleted: the API has no delete.\n") {
		t.Errorf("delete_widget's source =\n%q\nwant the comment ending its own body, which no other Operation's range may take", ending)
	}
}

// manifestEndingInItsOperationsBlock is the other shape a Manifest takes: the
// operations: block is the last thing in it, so the block ends where the file
// dedents out of it rather than at a next key. The file goes on afterwards all
// the same — a comment written at the top level, which belongs to the file and
// not to the Operation above it.
const manifestEndingInItsOperationsBlock = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    deadline: 30s

# Authored by hand: this Manifest carries no origin: block.
`

// TestOperationSource_TheBlockEndsWhereTheFileDedentsOutOfIt is the far end for
// an Operation authored last in a block that is itself the last thing in the
// Manifest: the range ends with the Operation, not with the file. What ends the
// block need not be a key at all — a comment written at the top level is
// outside the block by its indentation alone (§3, issue #114).
func TestOperationSource_TheBlockEndsWhereTheFileDedentsOutOfIt(t *testing.T) {
	got := mustSource(t, manifestEndingInItsOperationsBlock, "list_widgets")
	want := linesBetween(t, manifestEndingInItsOperationsBlock, "  list_widgets:", "    deadline: 30s")

	if got != want {
		t.Errorf("source =\n%q\nwant\n%q", got, want)
	}
	if strings.Contains(got, "Authored by hand") {
		t.Error("the range runs past the end of the operations: block and into the file's own comment")
	}
}

// TestSourceLines_IsEveryLineOfTheFileVerbatim is the promotion `hyper review`
// asked for: `operation` reads a range of a file's lines and a review reads all
// of them, off the same index (issue #118). Every line of the working tree's
// file renders — comments in place, indentation unchanged, blank lines counted
// — so what this answers is the file's own bytes cut at its own newlines and
// nothing else.
func TestSourceLines_IsEveryLineOfTheFileVerbatim(t *testing.T) {
	lines := SourceLines([]byte(widgetManifestWithComments))

	// The expectation is taken from the fixture by the standard library
	// rather than written out a second time: what the criterion asks is that
	// the reader hands back the file's own lines, so the other side of the
	// comparison may not be a copy of them made by hand.
	want := strings.Split(strings.TrimSuffix(widgetManifestWithComments, "\n"), "\n")
	if !slices.Equal(lines, want) {
		t.Errorf("SourceLines gave %d lines, want %d:\n got:  %q\n want: %q", len(lines), len(want), lines, want)
	}
}

// TestSourceLines_CountsFromOneOverEveryLineIncludingBlankOnes is the numbering
// every citation on the review surface resolves against (§8): line n of the
// file is lines[n-1], and a blank line is a line like any other.
func TestSourceLines_CountsFromOneOverEveryLineIncludingBlankOnes(t *testing.T) {
	lines := SourceLines([]byte(widgetManifestWithComments))

	// Line 11 of the fixture is the blank line between the first Operation
	// and the comment heading the second, counted off the constant above.
	if got := lines[10]; got != "" {
		t.Errorf("line 11 is %q, want the blank line the fixture writes there", got)
	}
	if got, want := lines[11], "  # Archives the widget rather than deleting it: the API has no delete."; got != want {
		t.Errorf("line 12 is %q, want %q", got, want)
	}
}

// TestSourceLines_ALastLineWithNoNewlineIsStillALine holds the two endings a
// file can have against one numbering: an author whose editor wrote no final
// newline has the same number of lines as one whose editor did, and neither
// gains an empty line at the end that the review would render as a blank one.
func TestSourceLines_ALastLineWithNoNewlineIsStillALine(t *testing.T) {
	terminated := SourceLines([]byte("kind: repository-declaration\nversion: 1.4.0\n"))
	bare := SourceLines([]byte("kind: repository-declaration\nversion: 1.4.0"))

	want := []string{"kind: repository-declaration", "version: 1.4.0"}
	if !slices.Equal(terminated, want) {
		t.Errorf("a file ending in a newline gave %q, want %q", terminated, want)
	}
	if !slices.Equal(bare, want) {
		t.Errorf("a file ending without one gave %q, want %q", bare, want)
	}
}

// TestSourceLines_AnEmptyFileHasNoLines is the degenerate reading: zero
// documents is valid YAML and a file with no bytes has no line to render, which
// is a review of nothing rather than a review of one blank line.
func TestSourceLines_AnEmptyFileHasNoLines(t *testing.T) {
	if lines := SourceLines(nil); len(lines) != 0 {
		t.Errorf("SourceLines of an empty file gave %q, want no lines at all", lines)
	}
}
