package cli_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// widgetManifest is a Manifest fixture in the shape §3 fixes, parameterised by
// its own name so a repository can hold several. It is written to a file whose
// basename the caller chooses, which is what lets a case prove the order
// follows the provider: rather than the path.
func widgetManifest(name string) string {
	return fmt.Sprintf(`kind: provider
provider: %s
schema-version: 1
class: widgetco
capabilities: [http]
operations:
  list_widgets:
    kind: read
    deadline: 30s
    http:
      method: GET
      host: "{from-target}"
      path: /widgets
    input:
      type: object
      properties:
        zone: {type: string}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id}
`, name)
}

// providersRepo is a repository holding hyper.yaml and one providers/ file per
// entry, keyed by the basename the file takes.
func providersRepo(t *testing.T, manifests map[string]string) string {
	t.Helper()
	root := newRepo(t)
	for basename, content := range manifests {
		writeFile(t, filepath.Join(root, "providers", basename), content)
	}
	return root
}

// runProviders drives `hyper providers` against root with the arguments given,
// and returns its two streams and its exit code. The environment is empty: the
// repository is named by the flag, which is what every case here means by
// "against this repository".
func runProviders(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	exit = cli.RunProviders(append([]string{"--repo-dir", root}, args...), &out, &errs, func(string) string { return "" }, root, "1.4.0")
	return out.String(), errs.String(), exit
}

// providerRowFixture is one row of the --json stream as a case reads it back.
// It is declared here rather than shared with the command, so that a test
// asserting a member's name is asserting the wire's own spelling.
type providerRowFixture struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Origin         string `json:"origin"`
	Summary        string `json:"summary"`
	OperationCount int    `json:"operation_count"`
	Digest         string `json:"digest"`
}

// readProviderRows reads the provider rows off a --json stream, leaving the
// terminal row to the cases that read it themselves.
func readProviderRows(t *testing.T, stdout string) []providerRowFixture {
	t.Helper()
	var rows []providerRowFixture
	for _, line := range jsonLines(t, stdout) {
		var row providerRowFixture
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		if row.Type == "provider" {
			rows = append(rows, row)
		}
	}
	return rows
}

// jsonLines is the stream's lines, each of which is one object (§8).
func jsonLines(t *testing.T, stdout string) []string {
	t.Helper()
	if stdout == "" {
		t.Fatal("the stream is empty; every stream carries at least its terminal row")
	}
	return strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
}

// names is the row set's names in the order they arrived, which is the fact
// §9 makes normative.
func names(rows []providerRowFixture) []string {
	var got []string
	for _, row := range rows {
		got = append(got, row.Name)
	}
	return got
}

// TestRunProviders_WritesOneRowPerProviderBuiltInAndExtensionAlike is the
// command's whole answer: every Provider hyper can load, the one that ships
// inside the binary among them, and exit 0.
func TestRunProviders_WritesOneRowPerProviderBuiltInAndExtensionAlike(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})

	stdout, stderr, exit := runProviders(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	rows := readProviderRows(t, stdout)
	if got, want := names(rows), []string{"shell", "widget"}; !slices.Equal(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if rows[0].Origin != "built-in" {
		t.Errorf("the built-in shell Provider's origin = %q, want %q", rows[0].Origin, "built-in")
	}
	if rows[1].Origin != "extension" {
		t.Errorf("a providers/ file's origin = %q, want %q", rows[1].Origin, "extension")
	}
	if rows[1].OperationCount != 1 {
		t.Errorf("operation_count = %d, want 1", rows[1].OperationCount)
	}
	if rows[1].Summary == "" {
		t.Error("the row carries no summary; §9 says every row carries one, derived")
	}
}

// TestRunProviders_OrdersByNameAndNotByPath is §9's normative order, held by a
// fixture whose files sort in the opposite direction to the names they declare.
func TestRunProviders_OrdersByNameAndNotByPath(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"aaa-last.yaml":  widgetManifest("zulu"),
		"zzz-first.yaml": widgetManifest("alpha"),
	})

	stdout, _, _ := runProviders(t, root, "--json")

	if got, want := names(readProviderRows(t, stdout)), []string{"alpha", "shell", "zulu"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v; the order is the name's, not the path's", got, want)
	}
}

// TestRunProviders_DigestIsSHA256OverTheManifestFilesExactBytes is the
// acceptance criterion that the digest is verifiable: the expectation is the
// standard library's hash of the same file on disk, not a value this package
// produced.
func TestRunProviders_DigestIsSHA256OverTheManifestFilesExactBytes(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})

	onDisk, err := os.ReadFile(filepath.Join(root, "providers", "widget.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(onDisk)
	want := "sha256:" + hex.EncodeToString(sum[:])

	stdout, _, _ := runProviders(t, root, "--json")
	for _, row := range readProviderRows(t, stdout) {
		if row.Name != "widget" {
			continue
		}
		if row.Digest != want {
			t.Errorf("digest = %q, want %q — sha256 over the file's exact bytes", row.Digest, want)
		}
		return
	}
	t.Fatal("no row for the widget Provider")
}

// TestRunProviders_TheBuiltInDigestIsOverTheCompiledInBytes is the one Provider
// with no blob in the repository, and the one whose digest nothing else could
// produce.
func TestRunProviders_TheBuiltInDigestIsOverTheCompiledInBytes(t *testing.T) {
	sum := sha256.Sum256([]byte(artefact.BuiltinShellProviderYAML))
	want := "sha256:" + hex.EncodeToString(sum[:])

	stdout, _, _ := runProviders(t, newRepo(t), "--json")
	rows := readProviderRows(t, stdout)
	if len(rows) != 1 || rows[0].Name != "shell" {
		t.Fatalf("rows = %v, want the built-in shell Provider alone", names(rows))
	}
	if rows[0].Digest != want {
		t.Errorf("digest = %q, want %q — sha256 over the bytes compiled into the binary", rows[0].Digest, want)
	}
}

// TestRunProviders_NoProvidersDirectoryListsTheBuiltInAlone is a repository
// that has authored no Extension: an absent directory is not an error, and the
// answer is the Provider that is always there.
func TestRunProviders_NoProvidersDirectoryListsTheBuiltInAlone(t *testing.T) {
	stdout, stderr, exit := runProviders(t, newRepo(t), "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}
	if got, want := names(readProviderRows(t, stdout)), []string{"shell"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

// TestRunProviders_AManifestThatWillNotParseContributesNoRow is ADR-0064's
// rule, already how the Provider index is built: a file that will not parse
// names nothing in the namespace, and its faults are check's to report rather
// than this command's to half-render.
func TestRunProviders_AManifestThatWillNotParseContributesNoRow(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"widget.yaml": widgetManifest("widget"),
		"broken.yaml": "kind: provider\nprovider: broken\n  bad indentation: [\n",
	})

	stdout, stderr, exit := runProviders(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}
	if got, want := names(readProviderRows(t, stdout)), []string{"shell", "widget"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v; an unparseable Manifest contributes no row", got, want)
	}
}

// TestRunProviders_LimitKeepsTheFirstNOfTheOrder is what the normative order
// buys: a bounded return is the first N of one answer rather than an arbitrary
// sample of one.
func TestRunProviders_LimitKeepsTheFirstNOfTheOrder(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"alpha.yaml": widgetManifest("alpha"),
		"zulu.yaml":  widgetManifest("zulu"),
	})

	stdout, _, exit := runProviders(t, root, "--json", "--limit", "2")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if got, want := names(readProviderRows(t, stdout)), []string{"alpha", "shell"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v — the first N of the order, not the last", got, want)
	}
}

// TestRunProviders_ATruncatedResultMarksItselfOnBothSurfaces is the rule a
// truncated result must never look complete: the marker on the terminal row,
// and a line on stderr naming what came back and what did not — in both modes,
// narration being stderr's in either (§9).
func TestRunProviders_ATruncatedResultMarksItselfOnBothSurfaces(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"alpha.yaml": widgetManifest("alpha"),
		"zulu.yaml":  widgetManifest("zulu"),
	})

	for _, mode := range [][]string{{"--json", "--limit", "1"}, {"--limit", "1"}} {
		stdout, stderr, _ := runProviders(t, root, mode...)

		if want := "returned 1 of 3 Providers; 2 dropped by --limit 1"; !strings.Contains(stderr, want) {
			t.Errorf("%v: stderr = %q, want it to carry %q", mode, stderr, want)
		}
		if strings.Contains(mode[0], "json") {
			if want := `{"type":"result","truncated":true}`; !strings.Contains(stdout, want) {
				t.Errorf("%v: stdout = %q, want the terminal row %s", mode, stdout, want)
			}
		}
	}
}

// TestRunProviders_AnUntruncatedResultSaysSo is the terminal row's other value,
// written always rather than only where it is true: a result row with no marker
// has nothing left to say.
func TestRunProviders_AnUntruncatedResultSaysSo(t *testing.T) {
	stdout, stderr, _ := runProviders(t, newRepo(t), "--json")

	if want := `{"type":"result","truncated":false}`; !strings.HasSuffix(stdout, want+"\n") {
		t.Errorf("stdout = %q, want it to end in %s", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want it silent where nothing was dropped", stderr)
	}
}

// TestRunProviders_TheMarkerNamesNoAxis is one of #106's four flagged
// resolutions: §12 closes the truncation axes at two, identity and time, and
// both are the record's — a Provider namespace has neither and neither
// narrowing parameter, so an axis member here would name a set member that does
// not describe it.
func TestRunProviders_TheMarkerNamesNoAxis(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})

	stdout, _, _ := runProviders(t, root, "--json", "--limit", "1")

	lines := jsonLines(t, stdout)
	terminal := lines[len(lines)-1]
	var marker map[string]any
	if err := json.Unmarshal([]byte(terminal), &marker); err != nil {
		t.Fatal(err)
	}
	if _, named := marker["axis"]; named {
		t.Errorf("the terminal row is %s; a Provider namespace has neither of §12's two axes", terminal)
	}
	if got, want := len(marker), 2; got != want {
		t.Errorf("the terminal row carries %d members (%s), want %d — type and truncated", got, terminal, want)
	}
}

// TestRunProviders_HasNoCursorAndNoPageParameter is §9's flat refusal of
// pagination: walking three thousand rows a page at a time is the same disaster
// arriving politely, so there is no way to ask for the next N.
func TestRunProviders_HasNoCursorAndNoPageParameter(t *testing.T) {
	root := newRepo(t)
	for _, flag := range []string{"--cursor", "--page", "--offset", "--after"} {
		stdout, stderr, exit := runProviders(t, root, flag, "1")
		if exit != cli.ExitUsage {
			t.Errorf("%s: exit = %d, want %d", flag, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%s: stdout = %q, want it silent; a usage error opens no row stream", flag, stdout)
		}
		if !strings.Contains(stderr, "unknown flag "+flag) {
			t.Errorf("%s: stderr = %q, want it to name the unknown flag", flag, stderr)
		}
	}
}

// TestRunProviders_LimitTakesAPositiveIntegerOrNothing is the flag's own
// grammar: a limit of none is the flag left off, and a limit of zero or below
// is a question with no answer in it.
func TestRunProviders_LimitTakesAPositiveIntegerOrNothing(t *testing.T) {
	root := newRepo(t)
	for _, value := range []string{"0", "-1", "all", ""} {
		_, stderr, exit := runProviders(t, root, "--limit", value)
		if exit != cli.ExitUsage {
			t.Errorf("--limit %q: exit = %d, want %d", value, exit, cli.ExitUsage)
		}
		if !strings.Contains(stderr, "want a positive integer") {
			t.Errorf("--limit %q: stderr = %q, want it to say what a limit is", value, stderr)
		}
	}

	if _, _, exit := runProviders(t, root, "--limit"); exit != cli.ExitUsage {
		t.Errorf("a bare --limit: exit = %d, want %d", exit, cli.ExitUsage)
	}
}

// TestRunProviders_TakesNoPositional is §9's tree read the other way: nine of
// the sixteen take a name positionally and this is not one of them, so a name
// here is the neighbouring command mistyped.
func TestRunProviders_TakesNoPositional(t *testing.T) {
	stdout, stderr, exit := runProviders(t, newRepo(t), "shell")
	if exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it silent", stdout)
	}
	if !strings.Contains(stderr, "takes no positional argument") {
		t.Errorf("stderr = %q, want it to say the command takes no positional", stderr)
	}
}

// TestRunProviders_TwoInvocationsAgainstOneRepositoryAreByteIdentical is what
// makes the answer diffable, and what makes truncation mean something: the
// order is fixed, so nothing here depends on a map's iteration.
func TestRunProviders_TwoInvocationsAgainstOneRepositoryAreByteIdentical(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"alpha.yaml":  widgetManifest("alpha"),
		"mike.yaml":   widgetManifest("mike"),
		"zulu.yaml":   widgetManifest("zulu"),
		"yankee.yaml": widgetManifest("yankee"),
	})

	for _, mode := range [][]string{{"--json"}, {}} {
		first, _, _ := runProviders(t, root, mode...)
		for i := range 8 {
			again, _, _ := runProviders(t, root, mode...)
			if again != first {
				t.Fatalf("%v: invocation %d wrote different bytes:\n %q\n %q", mode, i+2, first, again)
			}
		}
	}
}

// TestRunProviders_NoDigestIsAbbreviatedInEitherRendering is §8's rule and the
// reason the table is wide: a digest is verified with sha256sum rather than
// recognised by eye, so ADR-0047's abbreviation does not reach it.
func TestRunProviders_NoDigestIsAbbreviatedInEitherRendering(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})

	stream, _, _ := runProviders(t, root, "--json")
	for _, row := range readProviderRows(t, stream) {
		if len(row.Digest) != len("sha256:")+64 {
			t.Errorf("%s: digest = %q, want sha256: and 64 hex characters", row.Name, row.Digest)
		}
		if !strings.Contains(stream, row.Digest) {
			t.Errorf("%s: the stream does not carry the digest whole", row.Name)
		}

		table, _, _ := runProviders(t, root)
		if !strings.Contains(table, row.Digest) {
			t.Errorf("%s: the table carries no whole digest; got %q", row.Name, table)
		}
	}
}

// TestRunProviders_TheTableAndTheStreamStateTheSameFacts is ADR-0026 held over
// this command: one row set written twice, so the two surfaces cannot say
// different things.
func TestRunProviders_TheTableAndTheStreamStateTheSameFacts(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"alpha.yaml": widgetManifest("alpha"),
		"zulu.yaml":  widgetManifest("zulu"),
	})

	stream, _, _ := runProviders(t, root, "--json")
	table, _, _ := runProviders(t, root)

	tableLines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	if got, want := len(tableLines), len(readProviderRows(t, stream))+1; got != want {
		t.Fatalf("the table has %d lines and the stream %d rows; want a header and one line per row", got, want-1)
	}
	for i, row := range readProviderRows(t, stream) {
		line := tableLines[i+1]
		for _, cell := range []string{row.Name, row.Origin, row.Summary, row.Digest} {
			if !strings.Contains(line, cell) {
				t.Errorf("table line %q does not carry %q", line, cell)
			}
		}
	}
}

// TestRunProviders_TheGateFiresBeforeTheRepositoryIsLoaded is the pin gate on a
// command that is not check: a mismatched pin Refuses 77 with stdout silent, in
// both modes, and the Refusal renders on stderr because a Refusal is not a row
// (§9, ADR-0020).
func TestRunProviders_TheGateFiresBeforeTheRepositoryIsLoaded(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})
	writeFile(t, filepath.Join(root, "hyper.yaml"),
		"kind: repository-declaration\nversion: 9.9.9\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	for _, mode := range [][]string{{}, {"--json"}} {
		stdout, stderr, exit := runProviders(t, root, mode...)
		if exit != cli.ExitRefused {
			t.Errorf("%v: exit = %d, want %d", mode, exit, cli.ExitRefused)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent; a Refusal is not a row", mode, stdout)
		}
		if !strings.HasPrefix(stderr, "refused: version-pin-mismatch") {
			t.Errorf("%v: stderr = %q, want the Refusal", mode, stderr)
		}
	}
}

// TestRunProviders_ARepositoryWithNoDeclarationRefuses is the other half of the
// gate: no pin at all Refuses naming `hyper project` rather than listing
// anything (§4, ADR-0020).
func TestRunProviders_ARepositoryWithNoDeclarationRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "providers", "widget.yaml"), widgetManifest("widget"))

	stdout, stderr, exit := runProviders(t, root)
	if exit != cli.ExitRefused {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitRefused)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it silent", stdout)
	}
	if !strings.HasPrefix(stderr, "refused: version-pin-absent") {
		t.Errorf("stderr = %q, want version-pin-absent", stderr)
	}
}

// TestRunProviders_TheThreeGlobalsAreTheOnesSectionNineCloses: --repo-dir has
// its environment spelling, --no-color has its, and neither spelling of the
// latter changes a byte — there being no colour on this page to suppress, which
// is why both produce identical output rather than one of them being untested.
func TestRunProviders_TheThreeGlobalsAreTheOnesSectionNineCloses(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})
	elsewhere := t.TempDir()

	var viaEnv bytes.Buffer
	getenv := func(k string) string {
		if k == "HYPER_REPO_DIR" {
			return root
		}
		return ""
	}
	if exit := cli.RunProviders(nil, &viaEnv, &viaEnv, getenv, elsewhere, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("HYPER_REPO_DIR: exit = %d, want %d; output=%q", exit, cli.ExitClean, viaEnv.String())
	}

	viaFlag, _, _ := runProviders(t, root)
	if viaEnv.String() != viaFlag {
		t.Errorf("HYPER_REPO_DIR wrote %q and --repo-dir wrote %q", viaEnv.String(), viaFlag)
	}

	flagged, _, _ := runProviders(t, root, "--no-color")
	if flagged != viaFlag {
		t.Errorf("--no-color changed the bytes:\n %q\n %q", viaFlag, flagged)
	}

	var noColorEnv bytes.Buffer
	cli.RunProviders([]string{"--repo-dir", root}, &noColorEnv, &noColorEnv, func(k string) string {
		if k == "NO_COLOR" {
			return "1"
		}
		return ""
	}, root, "1.4.0")
	if noColorEnv.String() != viaFlag {
		t.Errorf("NO_COLOR changed the bytes:\n %q\n %q", viaFlag, noColorEnv.String())
	}
}

// TestRunProviders_CannotExitOne is the shape of the command: it reports facts,
// not problems found, so the code that means "problems found" is unreachable —
// including against a repository every other command has something to say
// about.
func TestRunProviders_CannotExitOne(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"widget.yaml":     widgetManifest("widget"),
		"broken.yaml":     "kind: provider\nprovider: broken\n  bad: [\n",
		"unknown-key.yml": "kind: provider\nprovider: nope\nbogus: 1\n",
	})
	writeFile(t, filepath.Join(root, "definitions", "typo.yaml"), "kind: definition\ndefinition: typo\nprovider: nowhere\n")

	for _, mode := range [][]string{{}, {"--json"}, {"--limit", "1"}} {
		if _, _, exit := runProviders(t, root, mode...); exit == cli.ExitProblems {
			t.Errorf("%v: exit = %d; `hyper providers` reports facts, not problems found", mode, exit)
		}
	}
}

// TestRunProviders_TheRowSetIsTheProviderNamespace is the fence on the one
// thing this command folds twice: the rows come off the loaded artefacts,
// because a row needs the bytes its digest covers, while every other artefact
// resolves a provider: against the namespace the load built. The two folds are
// one rule, and the day they stop agreeing a Provider is reachable from a
// Definition and absent from the list that is supposed to enumerate it.
func TestRunProviders_TheRowSetIsTheProviderNamespace(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"aaa-last.yaml": widgetManifest("zulu"),
		"alpha.yaml":    widgetManifest("alpha"),
		"broken.yaml":   "kind: provider\nprovider: broken\n  bad: [\n",
		"nameless.yaml": "kind: provider\nschema-version: 1\n",
	})

	loaded, err := repository.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := slices.Sorted(maps.Keys(loaded.Providers))

	stdout, _, _ := runProviders(t, root, "--json")
	if got := names(readProviderRows(t, stdout)); !slices.Equal(got, want) {
		t.Errorf("the rows name %v and the Provider namespace holds %v", got, want)
	}
}

// TestProvidersCorpus_NoCaseExitsOne is the corpus half of the rule the command
// states: `hyper providers` reports facts, not problems found, so exit 1 is
// unreachable from it however faulty the repository it read.
func TestProvidersCorpus_NoCaseExitsOne(t *testing.T) {
	cases, err := os.ReadDir(filepath.Join("testdata", "providers"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("the providers corpus is empty; the invariant would hold vacuously")
	}

	for _, entry := range cases {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join("testdata", "providers", entry.Name(), "exit.golden")
		recorded, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if exit := strings.TrimSpace(string(recorded)); exit == strconv.Itoa(cli.ExitProblems) {
			t.Errorf("%s records exit %s; `hyper providers` reports facts, not problems found", path, exit)
		}
	}
}

// TestRunProviders_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals is the
// half of "reaches nothing" that is assertable from here: the command resolves
// no credential, so the environment variables a Target declaration names are
// never looked up, and the only two variables read at all are the ones §9's
// globals define. The Store is not reached because nothing in this command
// knows where one would be, and no request is made because the whole answer is
// the repository load.
func TestRunProviders_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifest("widget")})
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
	getenv := func(key string) string {
		asked = append(asked, key)
		return ""
	}

	var stdout, stderr bytes.Buffer
	if exit := cli.RunProviders([]string{"--repo-dir", root}, &stdout, &stderr, getenv, root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}

	for _, key := range asked {
		if key != "HYPER_REPO_DIR" && key != "NO_COLOR" {
			t.Errorf("the command read %s; the only environment §9 gives it is the two globals'", key)
		}
	}
}

// TestRunProviders_TheDefaultLimitSaysItIsTheDefault is what a caller who typed
// no --limit is told when one cut their result anyway: that there is a default,
// what it is, and what widens it. Naming a flag they never wrote would send
// them looking for it in their own command line.
func TestRunProviders_TheDefaultLimitSaysItIsTheDefault(t *testing.T) {
	manifests := map[string]string{}
	for i := range 60 {
		name := fmt.Sprintf("widget-%02d", i)
		manifests[name+".yaml"] = widgetManifest(name)
	}
	root := providersRepo(t, manifests)

	stdout, stderr, exit := runProviders(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if got := len(readProviderRows(t, stdout)); got != 50 {
		t.Errorf("returned %d rows, want the default limit of 50", got)
	}
	if want := "returned 50 of 61 Providers; 11 dropped by the default limit of 50 — name a larger --limit for the rest"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want it to carry %q", stderr, want)
	}
	if !strings.Contains(stdout, `{"type":"result","truncated":true}`) {
		t.Errorf("stdout = %q, want the marker set", stdout)
	}
}
