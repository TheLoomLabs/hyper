package cli_test

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
// every command shares: a mismatched pin plus a positional matching nothing is
// 77 and not 2, because the gate fires first for all sixteen (§9, ADR-0020).
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

// TestRunOperation_ThePageAndTheStreamCarryOneSource is ADR-0026 at this
// command: the two renderings are one row written twice, so the lines the page
// writes are the lines the wire carries.
func TestRunOperation_ThePageAndTheStreamCarryOneSource(t *testing.T) {
	root := widgetRepo(t)

	page, _, exit := runOperation(t, root, "widget", "delete_widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	stream, _, _ := runOperation(t, root, "--json", "widget", "delete_widget")

	if got, want := page, readOperationDetailRow(t, stream).Source; got != want {
		t.Errorf("the page wrote\n%q\nand the wire carried\n%q", got, want)
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
		`"fmt"`:     true,
		`"io"`:      true,
		`"strings"`: true,
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
