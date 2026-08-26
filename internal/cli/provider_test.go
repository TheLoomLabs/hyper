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

// widgetManifestWithAuth is a Manifest carrying everything the header row
// reports: a header: scheme, a Capability, a schema version, an origin: block
// hyper itself wrote at install, and three Operations authored out of name
// order — one of them named for a blast radius it does not declare.
const widgetManifestWithAuth = `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
origin:
  ref: ghcr.io/widgetco/widget:1.2.0
  digest: sha256:1111111111111111111111111111111111111111111111111111111111111111
operations:
  list_widgets:
    kind: read
    deadline: 30s
    http: {method: GET, host: "{from-target}", path: /widgets}
    record:
      over: $.body.result
      identity: $.id
      fields: {id: $.id}
  delete_widget:
    kind: mutate
    deadline: 30s
    http: {method: POST, host: "{from-target}", path: "/widgets/{id}/archive"}
    record:
      identity: "{id}"
      fields: {id: $.body.id}
  create_widget:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 30s
    http: {method: POST, host: "{from-target}", path: /widgets}
    record:
      identity: "{name}"
      fields: {id: $.body.id}
`

// runProvider drives `hyper provider` against root with the arguments given,
// in an environment with nothing in it: the repository is named by the flag,
// which is what every case here means by "against this repository".
func runProvider(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	exit = cli.RunProvider(append([]string{"--repo-dir", root}, args...), cli.Streams(&out, &errs), emptyEnvironment, root, "1.4.0")
	return out.String(), errs.String(), exit
}

// manifestRowFixture is the header row as a case reads it back. It is declared
// here rather than shared with the command, so that a test asserting a
// member's name or its absence is asserting the wire's own spelling.
//
// SchemaVersion and the two origin members are pointers and strings a case
// tests for absence: both origin members are absent together where the
// Manifest carries no origin: block, and absent is not the same answer as
// empty (§7).
type manifestRowFixture struct {
	Type                 string   `json:"type"`
	AuthScheme           string   `json:"auth_scheme"`
	CapabilitiesRequired []string `json:"capabilities_required"`
	Digest               string   `json:"digest"`
	SchemaVersion        *int     `json:"schema_version"`
	OriginRef            *string  `json:"origin_ref"`
	OriginDigest         *string  `json:"origin_digest"`
}

// operationRowFixture is one Operation row as a case reads it back.
type operationRowFixture struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Opaque  bool   `json:"opaque"`
	Summary string `json:"summary"`
}

// readManifestRow is the stream's header row, which every clean answer opens
// with.
func readManifestRow(t *testing.T, stdout string) manifestRowFixture {
	t.Helper()
	var row manifestRowFixture
	if err := json.Unmarshal([]byte(jsonLines(t, stdout)[0]), &row); err != nil {
		t.Fatalf("%s: %v", jsonLines(t, stdout)[0], err)
	}
	if row.Type != "manifest" {
		t.Fatalf("the stream opens with a %q row, want the manifest row emitted first", row.Type)
	}
	return row
}

// readOperationRows reads the Operation rows off a --json stream, leaving the
// header and terminal rows to the cases that read them themselves.
func readOperationRows(t *testing.T, stdout string) []operationRowFixture {
	t.Helper()
	var rows []operationRowFixture
	for _, line := range jsonLines(t, stdout) {
		var row operationRowFixture
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		if row.Type == "operation" {
			rows = append(rows, row)
		}
	}
	return rows
}

// operationRowNames is the Operation rows' names in the order they arrived,
// which is the fact §9 makes normative.
func operationRowNames(rows []operationRowFixture) []string {
	var got []string
	for _, row := range rows {
		got = append(got, row.Name)
	}
	return got
}

// TestRunProvider_WritesTheManifestRowFirstThenOneRowPerOperation is the
// command's whole answer: the Manifest's own facts as a header row, the
// Operations it exposes, the terminal row, and exit 0.
func TestRunProvider_WritesTheManifestRowFirstThenOneRowPerOperation(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, stderr, exit := runProvider(t, root, "--json", "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	lines := jsonLines(t, stdout)
	if len(lines) != 5 {
		t.Fatalf("the stream carries %d rows, want the header, three Operations and the terminal row:\n%s", len(lines), stdout)
	}
	if !strings.HasPrefix(lines[0], `{"type":"manifest",`) {
		t.Errorf("the stream opens %q, want the manifest row emitted first", lines[0])
	}
	if got, want := lines[4], `{"type":"result","truncated":false}`; got != want {
		t.Errorf("the stream ends %q, want %q", got, want)
	}
	if got := len(readOperationRows(t, stdout)); got != 3 {
		t.Errorf("%d operation rows, want 3", got)
	}
}

// TestRunProvider_TheHeaderRowStatesWhatTheManifestDeclares is the row this
// command exists for: the Auth scheme it composes, the Capabilities it
// requires, its digest, its schema version, and its origin: block — the one
// part of a Manifest no other surface renders (§9).
func TestRunProvider_TheHeaderRowStatesWhatTheManifestDeclares(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, _, exit := runProvider(t, root, "--json", "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}

	row := readManifestRow(t, stdout)
	if got, want := row.AuthScheme, "Authorization: Bearer <secret>"; got != want {
		t.Errorf("auth_scheme = %q, want %q", got, want)
	}
	if got, want := row.CapabilitiesRequired, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("capabilities_required = %v, want %v", got, want)
	}
	if got, want := row.Digest, artefact.ManifestDigest([]byte(widgetManifestWithAuth)); got != want {
		t.Errorf("digest = %q, want %q — manifest_digest over the file's exact bytes", got, want)
	}
	if row.SchemaVersion == nil || *row.SchemaVersion != 1 {
		t.Errorf("schema_version = %v, want 1", row.SchemaVersion)
	}
	if row.OriginRef == nil || *row.OriginRef != "ghcr.io/widgetco/widget:1.2.0" {
		t.Errorf("origin_ref = %v, want the ref the origin: block carries", row.OriginRef)
	}
	if row.OriginDigest == nil || !strings.HasPrefix(*row.OriginDigest, "sha256:1111") {
		t.Errorf("origin_digest = %v, want the digest the origin: block carries", row.OriginDigest)
	}
}

// TestRunProvider_TheTwoOriginMembersAreAbsentTogether is the ordinary absence
// rule over the pair: the same Manifest with its origin: block removed writes
// neither, and neither is written empty. Absent together they say the Manifest
// makes no digest claim, which is the whole of what distinguishes an installed
// Extension from one an author wrote (§9, ADR-0073).
func TestRunProvider_TheTwoOriginMembersAreAbsentTogether(t *testing.T) {
	withBlock := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	stdout, _, _ := runProvider(t, withBlock, "--json", "widget")
	if row := readManifestRow(t, stdout); row.OriginRef == nil || row.OriginDigest == nil {
		t.Fatalf("origin_ref = %v and origin_digest = %v, want both written", row.OriginRef, row.OriginDigest)
	}

	_, block, found := strings.Cut(widgetManifestWithAuth, "origin:\n")
	if !found {
		t.Fatal("the fixture carries no origin: block to remove")
	}
	_, rest, _ := strings.Cut(block, "operations:\n")
	authored := strings.Replace(widgetManifestWithAuth, "origin:\n"+strings.TrimSuffix(block, rest), "", 1)

	without := providersRepo(t, map[string]string{"widget.yaml": authored})
	stdout, _, exit := runProvider(t, without, "--json", "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	row := readManifestRow(t, stdout)
	if row.OriginRef != nil || row.OriginDigest != nil {
		t.Errorf("origin_ref = %v and origin_digest = %v, want neither", row.OriginRef, row.OriginDigest)
	}
	if line := jsonLines(t, stdout)[0]; strings.Contains(line, "origin_ref") || strings.Contains(line, "origin_digest") {
		t.Errorf("the header row is %s; both members are omitted from the object rather than written empty", line)
	}
}

// TestRunProvider_TheBuiltInShellProviderRendersWithNoOriginMembers is the
// Provider hyper ships: no origin: block, so no digest claim, and its six
// Operations (§3, ADR-0073).
func TestRunProvider_TheBuiltInShellProviderRendersWithNoOriginMembers(t *testing.T) {
	root := newRepo(t)

	stdout, stderr, exit := runProvider(t, root, "--json", "shell")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	row := readManifestRow(t, stdout)
	if row.OriginRef != nil || row.OriginDigest != nil {
		t.Errorf("origin_ref = %v and origin_digest = %v, want neither", row.OriginRef, row.OriginDigest)
	}
	if got, want := row.AuthScheme, "none"; got != want {
		t.Errorf("auth_scheme = %q, want %q", got, want)
	}
	if got, want := row.Digest, artefact.ManifestDigest([]byte(artefact.BuiltinShellProviderYAML)); got != want {
		t.Errorf("digest = %q, want the digest over the compiled-in bytes %q", got, want)
	}
	if got := len(readOperationRows(t, stdout)); got != 6 {
		t.Errorf("%d operation rows, want the built-in's six", got)
	}
}

// TestRunProvider_OperationRowsAreOrderedByNameAscending is §9's normative
// order: a Manifest whose Operations are authored out of alphabetical order is
// ranged over in name order, which is what makes two renderings diffable.
func TestRunProvider_OperationRowsAreOrderedByNameAscending(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, _, exit := runProvider(t, root, "--json", "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}

	want := []string{"create_widget", "delete_widget", "list_widgets"}
	if got := operationRowNames(readOperationRows(t, stdout)); !slices.Equal(got, want) {
		t.Errorf("operations = %v, want %v — the Manifest authors them in another order", got, want)
	}
}

// TestRunProvider_KindIsReadFromTheManifestAndNotFromTheName is §12's rule at
// the row that carries Kind on every Operation: delete_widget declares mutate
// and is a mutate.
func TestRunProvider_KindIsReadFromTheManifestAndNotFromTheName(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, _, _ := runProvider(t, root, "--json", "widget")
	for _, row := range readOperationRows(t, stdout) {
		if row.Name == "delete_widget" && row.Kind != "mutate" {
			t.Errorf("delete_widget's kind = %q, want mutate — its kind: says so", row.Kind)
		}
	}
}

// TestRunProvider_OpacityIsReadFromTheRequestAndDeclaredByNothing is §12's
// derived fact on both surfaces: shell is opaque, http is not, and no artefact
// anywhere writes the word.
func TestRunProvider_OpacityIsReadFromTheRequestAndDeclaredByNothing(t *testing.T) {
	if strings.Contains(artefact.BuiltinShellProviderYAML, "opaque") {
		t.Error("the built-in Manifest carries the word opaque; opacity is derived and declared beside nothing")
	}

	builtin, _, _ := runProvider(t, newRepo(t), "--json", "shell")
	for _, row := range readOperationRows(t, builtin) {
		if !row.Opaque {
			t.Errorf("the built-in's %s is not opaque; its request is shell", row.Name)
		}
	}

	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	extension, _, _ := runProvider(t, root, "--json", "widget")
	for _, row := range readOperationRows(t, extension) {
		if row.Opaque {
			t.Errorf("%s is opaque; its request is http, which hyper describes", row.Name)
		}
	}
}

// TestRunProvider_EveryOperationRowCarriesADerivedSummary is the row's fourth
// member: derived from what the Manifest states, and never a key an author
// could write (§9).
func TestRunProvider_EveryOperationRowCarriesADerivedSummary(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, _, _ := runProvider(t, root, "--json", "widget")
	for _, row := range readOperationRows(t, stdout) {
		if row.Summary == "" {
			t.Errorf("%s carries no summary", row.Name)
		}
	}
}

// TestRunProvider_ANameMatchingNothingIsAUsageError is ADR-0060 at this
// command: a Refusal is the artefacts declining an act and a usage error is
// there being no act to decline. Nothing was reviewed, so there is no
// error_code, and no row stream opens in either mode.
func TestRunProvider_ANameMatchingNothingIsAUsageError(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	for _, mode := range [][]string{{"widgt"}, {"--json", "widgt"}} {
		stdout, stderr, exit := runProvider(t, root, mode...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d", mode, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent; no row stream opens on a usage error", mode, stdout)
		}
		if strings.Contains(stderr, "error_code") || strings.Contains(stderr, "refused:") {
			t.Errorf("%v: stderr = %q, want no error_code and no Refusal: nothing declined", mode, stderr)
		}
		if !strings.Contains(stderr, `"widgt"`) {
			t.Errorf("%v: stderr = %q, want it to name the name that was typed", mode, stderr)
		}
		if !strings.Contains(stderr, "hyper providers") {
			t.Errorf("%v: stderr = %q, want it to name the command that enumerates the namespace", mode, stderr)
		}
		if !strings.Contains(stderr, "Provider namespace") {
			t.Errorf("%v: stderr = %q, want it to name the namespace the name was resolved against", mode, stderr)
		}
	}
}

// TestRunProvider_ANameMatchingNothingListsNoCandidateAndSuggestsNoNearMiss is
// ADR-0047 carried into the message: a suggestion is a partial name resolved on
// the caller's behalf, and a human who accepts one has run something they did
// not type.
func TestRunProvider_ANameMatchingNothingListsNoCandidateAndSuggestsNoNearMiss(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"widget.yaml": widgetManifestWithAuth,
		"gadget.yaml": widgetManifest("gadget"),
	})

	_, stderr, _ := runProvider(t, root, "widgt")
	for _, candidate := range []string{"widget", "gadget", "shell"} {
		if strings.Contains(stderr, candidate) {
			t.Errorf("stderr = %q, want it to name no candidate; it names %q", stderr, candidate)
		}
	}
}

// TestRunProvider_MatchingIsByteExactAndCaseSensitive is ADR-0060's own rule:
// a name matches against the Manifest's own provider: rather than by whether a
// filesystem open succeeded, so `hyper provider Widget` exits 2 on a laptop
// whose filesystem would have opened the file.
func TestRunProvider_MatchingIsByteExactAndCaseSensitive(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	stdout, _, exit := runProvider(t, root, "Widget")
	if exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d; case is hyper's to decide, not the filesystem's", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it silent", stdout)
	}
}

// TestRunProvider_ANameMatchingNothingAnswersInARepositoryWithNoStore is
// ADR-0060's consequence: a working-tree name needs nothing beyond the working
// tree, so the typo is repaired before anything else is missed. No fixture
// here has a Store branch, and this command would not know where to look for
// one.
func TestRunProvider_ANameMatchingNothingAnswersInARepositoryWithNoStore(t *testing.T) {
	root := newRepo(t)

	_, stderr, exit := runProvider(t, root, "widget")
	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitUsage, stderr)
	}
}

// TestRunProvider_TheGateFiresBeforeThePositionalIsResolved is the ordering
// every command shares: a mismatched pin plus a name matching nothing is 77 and
// not 2, because the gate fires first everywhere (§9, ADR-0020).
func TestRunProvider_TheGateFiresBeforeThePositionalIsResolved(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	writeFile(t, filepath.Join(root, "hyper.yaml"),
		"kind: repository-declaration\nversion: 9.9.9\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	for _, mode := range [][]string{{"nowhere"}, {"--json", "nowhere"}} {
		stdout, stderr, exit := runProvider(t, root, mode...)
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

// TestRunProvider_TakesExactlyOnePositional: `provider` names a Manifest, so
// naming none and naming two are both usage errors (ADR-0060).
func TestRunProvider_TakesExactlyOnePositional(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	for _, args := range [][]string{{}, {"widget", "shell"}} {
		stdout, stderr, exit := runProvider(t, root, args...)
		if exit != cli.ExitUsage {
			t.Errorf("%v: exit = %d, want %d", args, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q, want it silent", args, stdout)
		}
		if !strings.HasPrefix(stderr, "hyper provider: ") {
			t.Errorf("%v: stderr = %q, want it to name the command", args, stderr)
		}
	}
}

// TestRunProvider_TakesNoLimit: `provider` names a Manifest rather than
// ranging over a namespace, so there is nothing for a cap to cut and --limit is
// an unknown flag (§9).
func TestRunProvider_TakesNoLimit(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	for _, spelling := range []string{"--limit", "--limit=2"} {
		stdout, stderr, exit := runProvider(t, root, spelling, "2", "widget")
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

// TestRunProvider_CannotExitOne is the shape of the command: it reports facts,
// not problems found, so the code that means "problems found" is unreachable —
// including against a repository every other command has something to say
// about.
func TestRunProvider_CannotExitOne(t *testing.T) {
	root := providersRepo(t, map[string]string{
		"widget.yaml": widgetManifestWithAuth,
		"broken.yaml": "kind: provider\nprovider: broken\n  bad: [\n",
	})
	writeFile(t, filepath.Join(root, "definitions", "typo.yaml"), "kind: definition\ndefinition: typo\nprovider: nowhere\n")

	for _, args := range [][]string{{"widget"}, {"--json", "widget"}, {"nowhere"}, {"broken"}} {
		if _, _, exit := runProvider(t, root, args...); exit == cli.ExitProblems {
			t.Errorf("%v: exit = %d; `hyper provider` reports facts, not problems found", args, exit)
		}
	}
}

// TestRunProvider_TheTableAndTheStreamStateTheSameFacts is ADR-0026 at this
// command: the two renderings are one list of rows written twice, so every
// fact on the wire is on the page.
func TestRunProvider_TheTableAndTheStreamStateTheSameFacts(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	page, _, exit := runProvider(t, root, "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	stream, _, _ := runProvider(t, root, "--json", "widget")

	header := readManifestRow(t, stream)
	for _, fact := range []string{
		header.AuthScheme,
		header.Digest,
		*header.OriginRef,
		*header.OriginDigest,
		strings.Join(header.CapabilitiesRequired, ", "),
	} {
		if !strings.Contains(page, fact) {
			t.Errorf("the page states no %q; the wire does", fact)
		}
	}
	for _, row := range readOperationRows(t, stream) {
		for _, fact := range []string{row.Name, row.Kind, row.Summary} {
			if !strings.Contains(page, fact) {
				t.Errorf("the page states no %q; the wire does", fact)
			}
		}
	}
}

// TestRunProvider_NoDigestIsAbbreviatedInEitherRendering is ADR-0047's other
// half: a digest is verified with sha256sum rather than recognised by eye, so
// a shortened one is a value the reader has to go somewhere else to complete.
func TestRunProvider_NoDigestIsAbbreviatedInEitherRendering(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	whole := artefact.ManifestDigest([]byte(widgetManifestWithAuth))

	for _, args := range [][]string{{"widget"}, {"--json", "widget"}} {
		stdout, _, _ := runProvider(t, root, args...)
		if !strings.Contains(stdout, whole) {
			t.Errorf("%v: stdout = %q, want the digest whole: %s", args, stdout, whole)
		}
		if strings.Contains(stdout, "…") || strings.Contains(stdout, "...") {
			t.Errorf("%v: stdout = %q, want nothing abbreviated", args, stdout)
		}
	}
}

// TestRunProvider_TwoInvocationsAgainstOneRepositoryAreByteIdentical is what
// the normative order buys: two renderings of one repository diff to nothing.
func TestRunProvider_TwoInvocationsAgainstOneRepositoryAreByteIdentical(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})

	first, _, _ := runProvider(t, root, "--json", "widget")
	second, _, _ := runProvider(t, root, "--json", "widget")
	if first != second {
		t.Errorf("two invocations differ:\n %q\n %q", first, second)
	}
}

// TestRunProvider_TheThreeGlobalsAreTheOnesSectionNineCloses: --repo-dir has
// its environment spelling, --no-color has its, and neither spelling of the
// latter changes a byte — there being no colour on this page to suppress.
func TestRunProvider_TheThreeGlobalsAreTheOnesSectionNineCloses(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	elsewhere := t.TempDir()

	var viaEnv bytes.Buffer
	lookupenv := func(key string) (string, bool) {
		if key == "HYPER_REPO_DIR" {
			return root, true
		}
		return "", false
	}
	if exit := cli.RunProvider([]string{"widget"}, cli.Streams(&viaEnv, &viaEnv), lookupenv, elsewhere, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("HYPER_REPO_DIR: exit = %d, want %d; output=%q", exit, cli.ExitClean, viaEnv.String())
	}

	viaFlag, _, _ := runProvider(t, root, "widget")
	if viaEnv.String() != viaFlag {
		t.Errorf("HYPER_REPO_DIR wrote %q and --repo-dir wrote %q", viaEnv.String(), viaFlag)
	}

	flagged, _, _ := runProvider(t, root, "--no-color", "widget")
	if flagged != viaFlag {
		t.Errorf("--no-color changed the bytes:\n %q\n %q", viaFlag, flagged)
	}

	var noColorEnv bytes.Buffer
	cli.RunProvider([]string{"--repo-dir", root, "widget"}, cli.Streams(&noColorEnv, &noColorEnv), func(key string) (string, bool) {
		if key == "NO_COLOR" {
			return "1", true
		}
		return "", false
	}, root, "1.4.0")
	if noColorEnv.String() != viaFlag {
		t.Errorf("NO_COLOR changed the bytes:\n %q\n %q", viaFlag, noColorEnv.String())
	}
}

// TestRunProvider_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals is the
// half of "reaches nothing" that is assertable from here: the command resolves
// no credential — the Auth scheme is rendered as the header it composes, with
// the marker in the credential's position — so the environment variables a
// Target declaration names are never looked up.
func TestRunProvider_ResolvesNoCredentialAndReadsNothingButTheTwoGlobals(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
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
	if exit := cli.RunProvider([]string{"--repo-dir", root, "widget"}, cli.Streams(&stdout, &stderr), lookupenv, root, "1.4.0"); exit != cli.ExitClean {
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

// TestRunProvider_AManifestDeclaringNoOperationSaysSo is the page's own empty
// case: a header over no rows would state less than a sentence does, which is
// the rule `check`'s clean run and `targets`'s empty repository already
// follow (issue #99).
func TestRunProvider_AManifestDeclaringNoOperationSaysSo(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
operations: {}
`})

	stdout, _, exit := runProvider(t, root, "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if !strings.Contains(stdout, "no Operations") {
		t.Errorf("stdout = %q, want it to say the Manifest declares no Operation", stdout)
	}
}

// TestRunProvider_ReachesNoNetworkNoStoreAndInvokesNothing fences the
// command's own file, on the shape `targets` and `providers` are fenced by:
// what a command can reach is what it imports, and this one imports its
// streams, the repository load, the artefact facts and the renderer. No net,
// no os/exec, no Store — the whole answer is the load and one lookup into the
// Provider namespace.
//
// It resolves no credential either, which is the fence's other half here: the
// Auth scheme this command renders is the header a Manifest composes, with the
// credential's position marked, and nothing behind it reads a variable, a file
// or a keychain (§9, ADR-0007).
func TestRunProvider_ReachesNoNetworkNoStoreAndInvokesNothing(t *testing.T) {
	allowed := map[string]bool{
		`"fmt"`:            true,
		`"io"`:             true,
		`"slices"`:         true,
		`"strconv"`:        true,
		`"strings"`:        true,
		`"text/tabwriter"`: true,
		`"github.com/TheLoomLabs/hyper/internal/artefact"`:   true,
		`"github.com/TheLoomLabs/hyper/internal/render"`:     true,
		`"github.com/TheLoomLabs/hyper/internal/repository"`: true,
	}

	file, err := parser.ParseFile(token.NewFileSet(), "provider.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("internal/cli/provider.go imports %s; `hyper provider` reaches no network, reads no Store, and invokes nothing", imp.Path.Value)
		}
	}
}

// TestRunProvider_TheMarkerIsSharedRatherThanReSpelled is §7's constant held
// against the surface that renders it: the marker in the credential's position
// is the one constant every rendering uses, so a second spelling here would be
// a second thing to recognise — and one that could drift from the one the
// Store writes.
func TestRunProvider_TheMarkerIsSharedRatherThanReSpelled(t *testing.T) {
	source, err := os.ReadFile("provider.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), `"`+artefact.SecretMarker+`"`) {
		t.Errorf("internal/cli/provider.go spells %q for itself; the marker is §7's one constant", artefact.SecretMarker)
	}

	root := providersRepo(t, map[string]string{"widget.yaml": widgetManifestWithAuth})
	stdout, _, _ := runProvider(t, root, "--json", "widget")
	if !strings.Contains(readManifestRow(t, stdout).AuthScheme, artefact.SecretMarker) {
		t.Errorf("auth_scheme = %q, want the credential's position marked by %q",
			readManifestRow(t, stdout).AuthScheme, artefact.SecretMarker)
	}
}

// TestProviderCorpus_NoCaseExitsOne is the corpus half of the rule the command
// states: `hyper provider` reports facts, not problems found, so exit 1 is
// unreachable from it however faulty the repository it read.
func TestProviderCorpus_NoCaseExitsOne(t *testing.T) {
	corpusReportsFactsNotProblems(t, "provider")
}

// TestRunProvider_AnAuthBlockHyperCannotReadStatesNothing is the third answer
// the Auth scheme has, and the one §12's two renderings do not cover: a
// Manifest carrying an auth: block that names no scheme has declared something
// hyper cannot read, and neither rendering may stand in for it. `none` is a
// claim — no credential is sent — and hyper cannot make it about a scheme it
// failed to parse, so the member is absent from the object and the page carries
// no line, exactly as a Manifest that stated no Capability carries no
// capabilities_required. What is wrong with the block is schema-mismatch, which
// `check` reports (§7, §12, ADR-0064).
func TestRunProvider_AnAuthBlockHyperCannotReadStatesNothing(t *testing.T) {
	root := providersRepo(t, map[string]string{"widget.yaml": `kind: provider
provider: widget
schema-version: 1
class: widgetco
capabilities: [http]
auth: {}
operations:
  list_widgets:
    kind: read
    deadline: 30s
    http: {method: GET, host: "{from-target}", path: /widgets}
`})

	stream, _, exit := runProvider(t, root, "--json", "widget")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if got := readManifestRow(t, stream).AuthScheme; got != "" {
		t.Errorf("auth_scheme = %q, want none written: the block names no scheme", got)
	}
	if line := jsonLines(t, stream)[0]; strings.Contains(line, "auth_scheme") {
		t.Errorf("the header row is %s; the member is omitted rather than written empty", line)
	}
	if strings.Contains(jsonLines(t, stream)[0], `"none"`) {
		t.Error("the row says none; none is the claim that no credential is sent, which this Manifest did not make")
	}

	page, _, _ := runProvider(t, root, "widget")
	if strings.Contains(page, "AUTH SCHEME") {
		t.Errorf("the page is %q, want no AUTH SCHEME line where the row carries no member", page)
	}
}
