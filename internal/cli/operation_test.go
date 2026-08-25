package cli_test

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/cli"
)

// widgetManifestWithComments is a Manifest authored the way a repository author
// writes one: Operations separated by blank lines, one documented by a comment
// above its key and by another inside its body, and one authored last in the
// block with a top-level key after it.
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

auth:
  header: {name: Authorization, prefix: "Bearer "}
`

// runOperation drives `hyper operation` against root with the arguments given,
// in an environment with nothing in it: the repository is named by the flag,
// which is what every case here means by "against this repository".
func runOperation(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	exit = cli.RunOperation(append([]string{"--repo-dir", root}, args...), &out, &errs, emptyEnvironment, root, "1.4.0")
	return out.String(), errs.String(), exit
}

// operationDetailRowFixture is the row as a case reads it back. It is declared
// here rather than shared with the command, so that a test asserting a member's
// name is asserting the wire's own spelling.
type operationDetailRowFixture struct {
	Type   string `json:"type"`
	Source string `json:"source"`
}

// readOperationDetailRow is the stream's one row, which every clean answer
// opens with.
func readOperationDetailRow(t *testing.T, stdout string) operationDetailRowFixture {
	t.Helper()
	first := jsonLines(t, stdout)[0]
	var row operationDetailRowFixture
	if err := json.Unmarshal([]byte(first), &row); err != nil {
		t.Fatalf("%s: %v", first, err)
	}
	if row.Type != "operation_detail" {
		t.Fatalf("the stream opens with a %q row, want operation_detail", row.Type)
	}
	return row
}

// widgetRepo is a repository whose one Extension is the commented Manifest
// above, written to disk so a case can read the file back and take its own
// range out of it.
func widgetRepo(t *testing.T) string {
	t.Helper()
	return providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithComments})
}

// TestRunOperation_WritesTheOperationDetailRowThenTheResultRow is the command's
// whole answer: the Operation's declaring lines, the terminal row, and exit 0
// (§9).
func TestRunOperation_WritesTheOperationDetailRowThenTheResultRow(t *testing.T) {
	root := widgetRepo(t)

	stdout, stderr, exit := runOperation(t, root, "--json", "widget", "delete_widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	lines := jsonLines(t, stdout)
	if len(lines) != 2 {
		t.Fatalf("the stream carries %d rows, want the detail row and the terminal row:\n%s", len(lines), stdout)
	}
	if !strings.HasPrefix(lines[0], `{"type":"operation_detail",`) {
		t.Errorf("the stream opens %q, want the operation_detail row", lines[0])
	}
	if got, want := lines[1], `{"type":"result","truncated":false}`; got != want {
		t.Errorf("the stream ends %q, want %q", got, want)
	}
}

// TestRunOperation_TheSourceIsByteForByteTheManifestFilesOwnRange is the whole
// point of the command, and it is asserted against the file rather than against
// a golden alone: the range is extracted from the bytes on disk with the
// standard library and held against what the row carried. A re-encoding that
// produced equivalent YAML would pass a golden file it was regenerated from and
// fail here (§12, issue #114).
func TestRunOperation_TheSourceIsByteForByteTheManifestFilesOwnRange(t *testing.T) {
	root := widgetRepo(t)

	stdout, _, exit := runOperation(t, root, "--json", "widget", "delete_widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}

	onDisk, err := os.ReadFile(filepath.Join(root, "providers", "widget.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Split(string(onDisk), "\n")
	from, to := indexOfLine(t, all, "  # Archives the widget rather than deleting it: the API has no delete."),
		indexOfLine(t, all, `    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/archive"}`)
	want := strings.Join(all[from:to+1], "\n") + "\n"

	if got := readOperationDetailRow(t, stdout).Source; got != want {
		t.Errorf("source =\n%q\nwant the file's own range\n%q", got, want)
	}
	if !strings.Contains(string(onDisk), readOperationDetailRow(t, stdout).Source) {
		t.Error("the source is not a range of the file's bytes")
	}
}

// indexOfLine is the 0-based index of a line the fixture writes, so a case
// names an anchor rather than a number that moves when the fixture is edited.
func indexOfLine(t *testing.T, lines []string, text string) int {
	t.Helper()
	for i, line := range lines {
		if line == text {
			return i
		}
	}
	t.Fatalf("the fixture has no line %q", text)
	return 0
}

// TestRunOperation_PreservesTheOriginalIndentation: the lines are written back
// unchanged, not dedented and not re-encoded, because a Manifest is written in
// the format the caller is expected to author Definitions in (§3, §9).
func TestRunOperation_PreservesTheOriginalIndentation(t *testing.T) {
	root := widgetRepo(t)

	stdout, _, _ := runOperation(t, root, "--json", "widget", "delete_widget")
	source := readOperationDetailRow(t, stdout).Source
	for _, line := range strings.Split(strings.TrimSuffix(source, "\n"), "\n") {
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("source carries the line %q; the Operation is authored two columns in", line)
		}
	}
	if !strings.Contains(source, "\n    kind: mutate\n") {
		t.Errorf("source =\n%q\nwant the mapping's own four-column members", source)
	}
}

// TestRunOperation_CarriesCommentsVerbatimAndInPlace is §3's rule at the one
// command that renders a Manifest's own bytes: a comment above the key
// documents the Operation and is part of its range, and a comment inside the
// body stands where it was written.
func TestRunOperation_CarriesCommentsVerbatimAndInPlace(t *testing.T) {
	root := widgetRepo(t)

	stdout, _, _ := runOperation(t, root, "--json", "widget", "delete_widget")
	source := readOperationDetailRow(t, stdout).Source

	if !strings.HasPrefix(source, "  # Archives the widget") {
		t.Errorf("source opens %q, want the comment above the key", strings.SplitN(source, "\n", 2)[0])
	}
	if !strings.Contains(source, "    deadline: 30s\n    # A POST, and not a DELETE, because the endpoint is /archive.\n") {
		t.Errorf("source =\n%q\nwant the body's comment in place", source)
	}
}

// TestRunOperation_TheRangeStopsWhereTheNextOperationBegins is the far end
// where another Operation follows, and where none does: the Operation authored
// last in a block ends at the end of the block rather than at a next key that
// is not there, and neither range carries a blank line it did not write.
func TestRunOperation_TheRangeStopsWhereTheNextOperationBegins(t *testing.T) {
	root := widgetRepo(t)

	middle, _, _ := runOperation(t, root, "--json", "widget", "list_widgets")
	source := readOperationDetailRow(t, middle).Source
	for _, next := range []string{"delete_widget", "Archives the widget"} {
		if strings.Contains(source, next) {
			t.Errorf("list_widgets's source carries %q; the range ends at the line before the next Operation's key", next)
		}
	}

	last, _, _ := runOperation(t, root, "--json", "widget", "create_widget")
	source = readOperationDetailRow(t, last).Source
	if !strings.HasPrefix(source, "  create_widget:") {
		t.Errorf("create_widget's source opens %q", strings.SplitN(source, "\n", 2)[0])
	}
	for _, beyond := range []string{"auth:", "Bearer"} {
		if strings.Contains(source, beyond) {
			t.Errorf("create_widget's source carries %q; the range ends at the end of the operations: block", beyond)
		}
	}
	for _, name := range []string{"list_widgets", "delete_widget", "create_widget"} {
		stdout, _, _ := runOperation(t, root, "--json", "widget", name)
		if source := readOperationDetailRow(t, stdout).Source; strings.HasSuffix(source, "\n\n") {
			t.Errorf("%s's source ends in a blank line; trailing blank lines are trimmed", name)
		}
	}
}

// TestRunOperation_TheBuiltInRendersFromTheCompiledInBytes is the one Manifest
// a repository author cannot read any other way: the built-in shell Provider
// has no file, and its source is the same range taken over the bytes compiled
// into the binary (§12, ADR-0039).
func TestRunOperation_TheBuiltInRendersFromTheCompiledInBytes(t *testing.T) {
	stdout, stderr, exit := runOperation(t, newRepo(t), "--json", "shell", "mutate_once")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	source := readOperationDetailRow(t, stdout).Source
	if !strings.Contains(artefact.BuiltinShellProviderYAML, source) {
		t.Errorf("source =\n%q\nis not a range of the compiled-in bytes", source)
	}
	if !strings.HasPrefix(source, "  mutate_once:") {
		t.Errorf("source opens %q, want the Operation's key line", strings.SplitN(source, "\n", 2)[0])
	}
	if strings.Contains(source, "mutate_skip_if_recorded") {
		t.Error("the range runs into the Operation that follows")
	}
}

// TestRunOperation_AProviderMatchingNothingIsAUsageError is ADR-0060 at the
// first of the two positionals: it resolves against the repository's Provider
// namespace, so the message names that namespace and `hyper providers`, and the
// Operation lookup is never attempted — there being no Manifest to look in.
func TestRunOperation_AProviderMatchingNothingIsAUsageError(t *testing.T) {
	root := widgetRepo(t)

	for _, mode := range [][]string{{"widgt", "delete_widget"}, {"--json", "widgt", "delete_widget"}} {
		stdout, stderr, exit := runOperation(t, root, mode...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d", mode, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent; no row stream opens on a usage error", mode, stdout)
		}
		if !strings.Contains(stderr, `"widgt"`) || !strings.Contains(stderr, "Provider namespace") {
			t.Errorf("%v: stderr = %q, want the name that was typed and the namespace it was resolved against", mode, stderr)
		}
		if !strings.Contains(stderr, "hyper providers") {
			t.Errorf("%v: stderr = %q, want the command that enumerates the Provider namespace", mode, stderr)
		}
		if strings.Contains(stderr, "Operation namespace") || strings.Contains(stderr, "delete_widget") {
			t.Errorf("%v: stderr = %q, want the Operation lookup never attempted; a bad Provider names no Manifest to look in", mode, stderr)
		}
	}
}

// TestRunOperation_AnOperationMatchingNothingIsAUsageError is the same rule at
// the second positional, which resolves against that Manifest's own Operation
// namespace: two namespaces, so two messages, and this one names the command
// that enumerates its own — `hyper provider <that provider>` (§9).
func TestRunOperation_AnOperationMatchingNothingIsAUsageError(t *testing.T) {
	root := widgetRepo(t)

	for _, mode := range [][]string{{"widget", "delete_widgets"}, {"--json", "widget", "delete_widgets"}} {
		stdout, stderr, exit := runOperation(t, root, mode...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d", mode, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent", mode, stdout)
		}
		if !strings.Contains(stderr, `"delete_widgets"`) || !strings.Contains(stderr, "Operation namespace") {
			t.Errorf("%v: stderr = %q, want the name that was typed and the namespace it was resolved against", mode, stderr)
		}
		if !strings.Contains(stderr, "hyper provider widget") {
			t.Errorf("%v: stderr = %q, want the command that enumerates that Manifest's Operations", mode, stderr)
		}
	}
}

// TestRunOperation_NeitherMessageCarriesAnErrorCodeOrOpensARowStream is what
// separates a usage error from a Refusal on both surfaces: nothing was
// reviewed, so nothing was refused, and there is no terminal row to be missing
// because no stream opened (§9, ADR-0060).
func TestRunOperation_NeitherMessageCarriesAnErrorCodeOrOpensARowStream(t *testing.T) {
	root := widgetRepo(t)

	for _, args := range [][]string{{"widgt", "delete_widget"}, {"widget", "delete_widgets"}} {
		for _, mode := range [][]string{args, append([]string{"--json"}, args...)} {
			stdout, stderr, _ := runOperation(t, root, mode...)
			if stdout != "" {
				t.Errorf("%v: stdout = %q, want it completely silent in both modes", mode, stdout)
			}
			if strings.Contains(stderr, "error_code") || strings.Contains(stderr, "refused:") {
				t.Errorf("%v: stderr = %q, want no error_code and no Refusal", mode, stderr)
			}
			if !strings.HasPrefix(stderr, "hyper operation: ") {
				t.Errorf("%v: stderr = %q, want it to name the command", mode, stderr)
			}
		}
	}
}

// TestRunOperation_MatchingIsByteExactAndCaseSensitive is §9's rule at both
// positionals, against the Manifest's own keys rather than settled by whether a
// filesystem open succeeded.
func TestRunOperation_MatchingIsByteExactAndCaseSensitive(t *testing.T) {
	root := widgetRepo(t)

	for _, args := range [][]string{{"Widget", "delete_widget"}, {"widget", "Delete_widget"}, {"widget", "delete_widget "}} {
		stdout, _, exit := runOperation(t, root, args...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d; the fold is hyper's, not the filesystem's", args, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent", args, stdout)
		}
	}
}

// TestRunOperation_TakesExactlyTwoPositionals: `operation` names one Operation
// of one Provider, so one positional and three are both usage errors decided
// from the argument list alone (ADR-0060).
func TestRunOperation_TakesExactlyTwoPositionals(t *testing.T) {
	root := widgetRepo(t)

	for _, args := range [][]string{{}, {"widget"}, {"widget", "delete_widget", "shell"}} {
		stdout, stderr, exit := runOperation(t, root, args...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d", args, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent", args, stdout)
		}
		if !strings.HasPrefix(stderr, "hyper operation: ") {
			t.Errorf("%v: stderr = %q, want it to name the command", args, stderr)
		}
	}
}

// TestRunOperation_TakesNoLimit: it names one Operation, so there is no result
// set for a cap to cut and --limit is an unknown flag (§9).
func TestRunOperation_TakesNoLimit(t *testing.T) {
	root := widgetRepo(t)

	for _, spelling := range []string{"--limit", "--limit=2"} {
		stdout, stderr, exit := runOperation(t, root, spelling, "2", "widget", "delete_widget")
		if exit != cli.ExitUsage {
			t.Errorf("%s: exit = %d, want %d", spelling, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%s: stdout = %q, want it silent", spelling, stdout)
		}
		if !strings.Contains(stderr, "unknown flag") {
			t.Errorf("%s: stderr = %q, want an unknown flag", spelling, stderr)
		}
	}
}

// TestRunOperation_TheGateFiresBeforeEitherPositionalIsResolved is the ordering
// every gated command shares: a mismatched pin plus a positional matching
// nothing is 77 and not 2, because the gate fires first for fifteen of the
// sixteen (§9, ADR-0020).
func TestRunOperation_TheGateFiresBeforeEitherPositionalIsResolved(t *testing.T) {
	root := widgetRepo(t)
	writeFile(t, filepath.Join(root, "hyper.yaml"),
		"kind: repository-declaration\nversion: 9.9.9\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	for _, args := range [][]string{{"nowhere", "delete_widget"}, {"widget", "nowhere"}} {
		stdout, stderr, exit := runOperation(t, root, args...)
		if exit != cli.ExitRefused {
			t.Errorf("%v: exit = %d, want %d", args, exit, cli.ExitRefused)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent; a Refusal is not a row", args, stdout)
		}
		if !strings.HasPrefix(stderr, "refused: version-pin-mismatch") {
			t.Errorf("%v: stderr = %q, want the Refusal", args, stderr)
		}
	}
}

// TestRunOperation_CannotExitOne is the shape of the command: it reports facts,
// not problems found, so the code that means "problems found" is unreachable —
// including against a repository every other command has something to say
// about.
func TestRunOperation_CannotExitOne(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"widget.yaml": widgetManifestWithComments,
		"broken.yaml": "kind: provider\nprovider: broken\n  bad: [\n",
	})
	writeFile(t, filepath.Join(root, "definitions", "typo.yaml"), "kind: definition\ndefinition: typo\nprovider: nowhere\n")

	for _, args := range [][]string{
		{"widget", "delete_widget"},
		{"--json", "widget", "delete_widget"},
		{"nowhere", "delete_widget"},
		{"widget", "nowhere"},
		{"broken", "read"},
	} {
		if _, _, exit := runOperation(t, root, args...); exit == cli.ExitProblems {
			t.Errorf("%v: exit = %d; `hyper operation` reports facts, not problems found", args, exit)
		}
	}
}

// TestRunOperation_ThePageAndTheStreamStateTheSameFacts is ADR-0026 at this
// command: the two renderings are one row written twice, so the lines the page
// opens with are the lines the wire carries, and every derived fact under them
// is the fact the derived block carried.
func TestRunOperation_ThePageAndTheStreamStateTheSameFacts(t *testing.T) {
	root := widgetRepo(t)

	page, _, exit := runOperation(t, root, "widget", "delete_widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	stream, _, _ := runOperation(t, root, "--json", "widget", "delete_widget")

	source := readOperationDetailRow(t, stream).Source
	if !strings.HasPrefix(page, source) {
		t.Errorf("the page opens\n%q\nand the wire carried\n%q", page, source)
	}

	derived := readDerived(t, root, "delete_widget")
	for label, value := range map[string]string{
		"CAPABILITIES":      strings.Join(derived.Capabilities, ", "),
		"BOUND":             derived.Bound,
		"REPEATABILITY":     derived.Repeatability,
		"CONCURRENCY LIMIT": "1",
	} {
		if got := labelledValue(t, page, label); got != value {
			t.Errorf("the page states %s %q and the wire carried %q", label, got, value)
		}
	}
	if strings.Contains(page, "PATTERNS RESOLVED") {
		t.Errorf("the page\n%q\nlabels a Pattern set nothing is in; the wire says it with an empty list and the page by having no line", page)
	}
}

// TestRunOperation_TheThreeGlobalsAreTheOnesSectionNineCloses: --repo-dir has
// its environment spelling, --no-color has its, and neither spelling of the
// latter changes a byte — there being no colour on this page to suppress.
func TestRunOperation_TheThreeGlobalsAreTheOnesSectionNineCloses(t *testing.T) {
	root := widgetRepo(t)
	elsewhere := t.TempDir()

	var viaEnv bytes.Buffer
	lookupenv := func(key string) (string, bool) {
		if key == "HYPER_REPO_DIR" {
			return root, true
		}
		return "", false
	}
	if exit := cli.RunOperation([]string{"widget", "delete_widget"}, &viaEnv, &viaEnv, lookupenv, elsewhere, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("HYPER_REPO_DIR: exit = %d, want %d; output=%q", exit, cli.ExitClean, viaEnv.String())
	}

	viaFlag, _, _ := runOperation(t, root, "widget", "delete_widget")
	if viaEnv.String() != viaFlag {
		t.Errorf("HYPER_REPO_DIR wrote %q and --repo-dir wrote %q", viaEnv.String(), viaFlag)
	}

	flagged, _, _ := runOperation(t, root, "--no-color", "widget", "delete_widget")
	if flagged != viaFlag {
		t.Errorf("--no-color changed the bytes:\n %q\n %q", viaFlag, flagged)
	}

	var noColorEnv bytes.Buffer
	cli.RunOperation([]string{"--repo-dir", root, "widget", "delete_widget"}, &noColorEnv, &noColorEnv, func(key string) (string, bool) {
		if key == "NO_COLOR" {
			return "1", true
		}
		return "", false
	}, root, "1.4.0")
	if noColorEnv.String() != viaFlag {
		t.Errorf("NO_COLOR changed the bytes:\n %q\n %q", viaFlag, noColorEnv.String())
	}
}

// TestRunOperation_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals is the
// half of "reaches nothing" that is assertable from here: the command writes a
// Manifest's own lines back, so the environment variables a Target declaration
// names are never looked up (§9, ADR-0007).
func TestRunOperation_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals(t *testing.T) {
	root := widgetRepo(t)
	writeFile(t, filepath.Join(root, "targets", "widgetco-prod.yaml"), `kind: target-declaration
target: widgetco-prod
class: widgetco
kinds: [read]
capabilities: [http]
hosts: [api.widgetco.example]
auth:
  token: {env: WIDGETCO_API_TOKEN}
`)

	var asked []string
	lookupenv := func(key string) (string, bool) {
		asked = append(asked, key)
		return "", false
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.RunOperation([]string{"--repo-dir", root, "widget", "delete_widget"}, &stdout, &stderr, lookupenv, root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}
	for _, key := range asked {
		if key != "HYPER_REPO_DIR" && key != "NO_COLOR" {
			t.Errorf("the command read %s; the only environment §9 gives it is the two globals'", key)
		}
	}
	if strings.Contains(stdout.String(), "WIDGETCO_API_TOKEN") {
		t.Errorf("stdout = %q, want no credential anywhere in it", stdout.String())
	}
}

// TestRunOperation_ReachesNoNetworkNoStoreAndInvokesNothing fences the
// command's own file, on the shape `provider`, `targets` and `providers` are
// fenced by: what a command can reach is what it imports, and this one imports
// its streams, the repository load, the artefact read and the renderer. No net,
// no os/exec, no Store — the whole answer is the load, two lookups and a range
// of bytes.
func TestRunOperation_ReachesNoNetworkNoStoreAndInvokesNothing(t *testing.T) {
	allowed := map[string]bool{
		`"fmt"`:            true,
		`"io"`:             true,
		`"strconv"`:        true,
		`"strings"`:        true,
		`"text/tabwriter"`: true,
		`"github.com/TheLoomLabs/hyper/internal/artefact"`:   true,
		`"github.com/TheLoomLabs/hyper/internal/render"`:     true,
		`"github.com/TheLoomLabs/hyper/internal/repository"`: true,
	}

	file, err := parser.ParseFile(token.NewFileSet(), "operation.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("internal/cli/operation.go imports %s; `hyper operation` reaches no network, reads no Store, and invokes nothing", imp.Path.Value)
		}
	}
}

// TestOperationCorpus_NoCaseExitsOne is the corpus half of the rule the command
// states: `hyper operation` reports facts, not problems found, so exit 1 is
// unreachable from it however faulty the repository it read.
func TestOperationCorpus_NoCaseExitsOne(t *testing.T) {
	corpusReportsFactsNotProblems(t, "operation")
}

// widgetManifestWithDerivedFacts is a Manifest carrying one Operation per fact
// the derived block has to answer for: a paginated `read` over a series with a
// declared concurrency limit, a `mutate` under each Repeatability value it may
// declare and one under neither, a `read` declaring neither, and the non-opaque
// `destroy` — the built-in shell Provider carrying the opaque one.
const widgetManifestWithDerivedFacts = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    repeatability: repeatable
    deadline: 30s
    concurrency: 4
    patterns:
      pagination:
        cursor: {from: $.body.cursor, into: {query: cursor}}
      retry: {attempts: 3}
    http: {method: GET, host: "{from-target}", path: /widgets}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id}
  get_widget:
    kind: read
    deadline: 2m
    http: {method: GET, host: "{from-target}", path: "/widgets/{id}"}
    record:
      identity: $.body.id
      fields: {id: $.body.id}
  create_widget:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    http: {method: POST, host: "{from-target}", path: /widgets}
    record:
      identity: "{name}"
      fields: {id: $.body.id}
  rotate_widget:
    kind: mutate
    deadline: 1d
    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/rotate"}
    record:
      identity: "{id}"
      fields: {id: $.body.id}
  delete_widget:
    kind: destroy
    repeatability: repeatable
    deadline: 30s
    http: {method: DELETE, host: "{from-target}", path: "/widgets/{id}"}
`

// derivedRepo is a repository whose one Extension is the Manifest above.
func derivedRepo(t *testing.T) string {
	t.Helper()
	return providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithDerivedFacts})
}

// derivedFixture is the derived block as a case reads it back, declared here
// rather than shared with the command so that a test asserting a member's name
// is asserting the wire's own spelling. Every member is a pointer or a slice or
// carries omitempty on the command's side; here they are read as they arrive,
// so a member the row omitted reads as its zero value and a case asserting
// absence asserts it against the bytes.
type derivedFixture struct {
	Capabilities      []string `json:"capabilities"`
	Bound             string   `json:"bound"`
	PatternsResolved  []string `json:"patterns_resolved"`
	RecordCardinality string   `json:"record_cardinality"`
	RecordIdentity    string   `json:"record_identity"`
	Repeatability     string   `json:"repeatability"`
	DeadlineSeconds   *int     `json:"deadline_seconds"`
	ConcurrencyLimit  *int     `json:"concurrency_limit"`
}

// readDerived is the derived block of the stream's one row, for the case that
// names one Operation of the Manifest above.
func readDerived(t *testing.T, root, name string) derivedFixture {
	t.Helper()
	stdout, stderr, exit := runOperation(t, root, "--json", "widget", name)
	if exit != cli.ExitClean {
		t.Fatalf("%s: exit = %d, want %d; stderr=%q", name, exit, cli.ExitClean, stderr)
	}
	var row struct {
		Derived derivedFixture `json:"derived"`
	}
	first := jsonLines(t, stdout)[0]
	if err := json.Unmarshal([]byte(first), &row); err != nil {
		t.Fatalf("%s: %v", first, err)
	}
	return row.Derived
}

// TestRunOperation_TheDerivedBlockStandsBesideTheSource is the row's shape: the
// Manifest's own lines, and beside them the facts the source does not carry in
// that form, on one row rather than two — §9 writes the shape out once and
// milestone 11's MCP tool reuses this contract rather than minting a second one.
func TestRunOperation_TheDerivedBlockStandsBesideTheSource(t *testing.T) {
	root := derivedRepo(t)

	stdout, _, exit := runOperation(t, root, "--json", "widget", "list_widgets")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}

	lines := jsonLines(t, stdout)
	if len(lines) != 2 {
		t.Fatalf("the stream carries %d rows, want the detail row and the terminal row:\n%s", len(lines), stdout)
	}
	if !strings.HasPrefix(lines[0], `{"type":"operation_detail","source":`) {
		t.Errorf("the row opens %.48q; type first, then the source it is written beside", lines[0])
	}
	if !strings.Contains(lines[0], `,"derived":{"capabilities":["http"],"bound":"none","patterns_resolved":["pagination","retry"],`) {
		t.Errorf("the row carries %q; want §9's own members in §9's own order", lines[0])
	}
}

// TestRunOperation_TheCapabilityIsTheOneTheRequestIsWrittenUnder: an Operation
// uses exactly one Capability and the request block's key is it, so the member
// carries exactly one member and never a second (§12).
func TestRunOperation_TheCapabilityIsTheOneTheRequestIsWrittenUnder(t *testing.T) {
	if got := readDerived(t, derivedRepo(t), "list_widgets").Capabilities; len(got) != 1 || got[0] != "http" {
		t.Errorf("capabilities = %v, want exactly [http]", got)
	}

	var row struct {
		Derived derivedFixture `json:"derived"`
	}
	stdout, _, _ := runOperation(t, newRepo(t), "--json", "shell", "destroy")
	if err := json.Unmarshal([]byte(jsonLines(t, stdout)[0]), &row); err != nil {
		t.Fatal(err)
	}
	if got := row.Derived.Capabilities; len(got) != 1 || got[0] != "shell" {
		t.Errorf("the built-in shell destroy's capabilities = %v, want exactly [shell]", got)
	}
}

// TestRunOperation_BoundCarriesThreeMembersAndNoBooleanSpellingOfIt is the
// resolution this ticket carries: §5 gives the fact three states, and a boolean
// would carry *you need not write one* and *writing one is refused* under one
// value, on the most severe Operation the tool runs.
func TestRunOperation_BoundCarriesThreeMembersAndNoBooleanSpellingOfIt(t *testing.T) {
	root := derivedRepo(t)

	if got := readDerived(t, root, "delete_widget").Bound; got != "mandatory" {
		t.Errorf("a non-opaque destroy's bound = %q, want mandatory", got)
	}
	for _, name := range []string{"list_widgets", "create_widget"} {
		if got := readDerived(t, root, name).Bound; got != "none" {
			t.Errorf("%s's bound = %q, want none", name, got)
		}
	}

	opaque, _, _ := runOperation(t, newRepo(t), "--json", "shell", "destroy")
	if !strings.Contains(opaque, `"bound":"illegal"`) {
		t.Errorf("the opaque destroy's row = %q, want bound illegal", opaque)
	}
	for _, stream := range []string{opaque, mustStream(t, root, "delete_widget")} {
		if strings.Contains(stream, "bound_required") {
			t.Errorf("the row = %q; the field is named bound, and no boolean spelling of it exists anywhere", stream)
		}
	}
}

// mustStream is one Operation's --json stream, for a case that reads the bytes
// rather than the decoded members.
func mustStream(t *testing.T, root, name string) string {
	t.Helper()
	stdout, stderr, exit := runOperation(t, root, "--json", "widget", name)
	if exit != cli.ExitClean {
		t.Fatalf("%s: exit = %d, want %d; stderr=%q", name, exit, cli.ExitClean, stderr)
	}
	return stdout
}

// TestRunOperation_PatternsResolvedIsEmptyRatherThanAbsent: a caller asking
// which Patterns run around this call is answered *none of them*, which is a
// fact, where an absent member would say the question was not asked (§9).
func TestRunOperation_PatternsResolvedIsEmptyRatherThanAbsent(t *testing.T) {
	root := derivedRepo(t)

	if got := readDerived(t, root, "list_widgets").PatternsResolved; !slices.Equal(got, []string{"pagination", "retry"}) {
		t.Errorf("patterns_resolved = %v, want the members the Operation declares", got)
	}
	if !strings.Contains(mustStream(t, root, "create_widget"), `"patterns_resolved":[],`) {
		t.Errorf("the row = %q, want an empty patterns_resolved rather than an absent one", mustStream(t, root, "create_widget"))
	}
}

// TestRunOperation_TheRecordPairIsAbsentTogetherOnADestroy: a destroy carries
// no record: and declares no identity, what it writes being a Tombstone under
// the series its Expansion acted on — absent rather than empty, the ordinary
// absence rule being a fact a reader reads (§3, §7, ADR-0037).
func TestRunOperation_TheRecordPairIsAbsentTogetherOnADestroy(t *testing.T) {
	root := derivedRepo(t)

	series := readDerived(t, root, "list_widgets")
	if series.RecordCardinality != "series" || series.RecordIdentity != "$.id" {
		t.Errorf("the record pair = (%q, %q), want (series, $.id)", series.RecordCardinality, series.RecordIdentity)
	}
	if got := readDerived(t, root, "create_widget"); got.RecordCardinality != "one" || got.RecordIdentity != "{name}" {
		t.Errorf("the record pair = (%q, %q), want (one, {name}) — the template hole verbatim", got.RecordCardinality, got.RecordIdentity)
	}

	for _, stream := range []string{mustStream(t, root, "delete_widget"), builtinStream(t, "destroy")} {
		for _, member := range []string{"record_cardinality", "record_identity"} {
			if strings.Contains(stream, member) {
				t.Errorf("a destroy's row carries %s; both are omitted from the object, not written as null or \"\":\n%s", member, stream)
			}
		}
	}
}

// builtinStream is one built-in shell Operation's --json stream, read against a
// repository with no Extension in it at all.
func builtinStream(t *testing.T, name string) string {
	t.Helper()
	stdout, stderr, exit := runOperation(t, newRepo(t), "--json", "shell", name)
	if exit != cli.ExitClean {
		t.Fatalf("shell %s: exit = %d, want %d; stderr=%q", name, exit, cli.ExitClean, stderr)
	}
	return stdout
}

// TestRunOperation_RepeatabilityIsTheEffectiveValue: run-once is rendered even
// though no artefact may write that word, which makes it exactly parallel to
// opaque — a fact no artefact declares and every surface renders (§12).
func TestRunOperation_RepeatabilityIsTheEffectiveValue(t *testing.T) {
	root := derivedRepo(t)

	for name, want := range map[string]string{
		"list_widgets":  "repeatable",
		"create_widget": "skip-if-recorded",
		"rotate_widget": "run-once",
		"get_widget":    "repeatable",
	} {
		if got := readDerived(t, root, name).Repeatability; got != want {
			t.Errorf("%s's repeatability = %q, want %q", name, got, want)
		}
	}
}

// TestRunOperation_TheDeadlineIsSecondsOnTheWireAndTheAuthoredSpellingOnThePage
// — §9 fixed the wire name and its unit with it, and the human table renders
// what the source beside it says.
func TestRunOperation_TheDeadlineIsSecondsOnTheWireAndTheAuthoredSpellingOnThePage(t *testing.T) {
	root := derivedRepo(t)

	seconds := readDerived(t, root, "get_widget").DeadlineSeconds
	if seconds == nil || *seconds != 120 {
		t.Fatalf("deadline_seconds = %v, want 120", seconds)
	}

	page, _, _ := runOperation(t, root, "widget", "get_widget")
	if !strings.Contains(page, "2m") {
		t.Errorf("the page = %q, want the authored spelling 2m", page)
	}
	if strings.Contains(page, "120") {
		t.Errorf("the page = %q, want no second spelling of the deadline on it", page)
	}
}

// TestRunOperation_TheConcurrencyLimitIsPresentOnEveryOperation is ADR-0045 on
// the wire: a caller asking *how many at once* gets a number for every
// Operation, and the rule about which Kinds may author the key stays in §3.
func TestRunOperation_TheConcurrencyLimitIsPresentOnEveryOperation(t *testing.T) {
	root := derivedRepo(t)

	for name, want := range map[string]int{
		"list_widgets":  4,
		"get_widget":    1,
		"create_widget": 1,
		"rotate_widget": 1,
		"delete_widget": 1,
	} {
		got := readDerived(t, root, name).ConcurrencyLimit
		if got == nil {
			t.Errorf("%s's concurrency_limit is absent; it is present on every Operation without exception", name)
			continue
		}
		if *got != want {
			t.Errorf("%s's concurrency_limit = %d, want %d", name, *got, want)
		}
	}
	for _, name := range []string{"read", "mutate", "destroy"} {
		if !strings.Contains(builtinStream(t, name), `"concurrency_limit":1`) {
			t.Errorf("the built-in %s's row = %q, want concurrency_limit 1", name, builtinStream(t, name))
		}
	}
}

// TestRunOperation_AnExplicitOneAndAnOmittedKeyWriteTheSameAnswer: 1 is an
// ordinary member of an integer's value set, so an author who established that
// an API refuses concurrency may write it — and what they wrote means what the
// omission means (ADR-0045).
func TestRunOperation_AnExplicitOneAndAnOmittedKeyWriteTheSameAnswer(t *testing.T) {
	explicit := providersRepo(t, map[string]string{
		"widget.yaml": strings.Replace(widgetManifestWithDerivedFacts,
			"  get_widget:\n    kind: read\n    deadline: 2m\n",
			"  get_widget:\n    kind: read\n    deadline: 2m\n    concurrency: 1\n", 1),
	})

	omitted := mustStream(t, derivedRepo(t), "get_widget")
	written := mustStream(t, explicit, "get_widget")
	if strings.Contains(written, "concurrency: 1") == false {
		t.Fatal("the fixture did not write an explicit concurrency: 1")
	}
	if got, want := derivedOf(t, written), derivedOf(t, omitted); got != want {
		t.Errorf("an explicit concurrency: 1 derived %s and an omitted key derived %s", got, want)
	}
}

// derivedOf is the derived object of a stream's first row, as bytes, for a case
// comparing two answers rather than reading one.
func derivedOf(t *testing.T, stream string) string {
	t.Helper()
	_, derived, found := strings.Cut(jsonLines(t, stream)[0], `,"derived":`)
	if !found {
		t.Fatalf("the row carries no derived block: %s", stream)
	}
	return derived
}

// labelledValue is what the page states against one label of its derived block,
// with the alignment taken off — the page aligns its values into a column and a
// case reads the value rather than the padding.
func labelledValue(t *testing.T, page, label string) string {
	t.Helper()
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, label+" ") {
			return strings.TrimSpace(strings.TrimPrefix(line, label))
		}
	}
	t.Fatalf("the page\n%q\ncarries no %s line", page, label)
	return ""
}
