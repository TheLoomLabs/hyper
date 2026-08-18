package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// cloudflareProdTarget is §3's own worked Target declaration: one host, the
// Kinds it accepts, the Capability it grants, and one credential slot naming
// the variable it resolves from.
const cloudflareProdTarget = `kind: target-declaration
target: cloudflare-prod
class: cloudflare
kinds: [read, mutate, destroy]
capabilities: [http]
hosts: [api.cloudflare.com]
auth:
  token: {env: CLOUDFLARE_API_TOKEN}
`

// localTargetDeclaration is the other of §3's two, and the Target a Probe
// binds: authored by the repository in targets/ like every other, carrying no
// auth: block at all (§4, ADR-0041).
const localTargetDeclaration = `kind: target-declaration
target: local
class: local
kinds: [read]
capabilities: [http]
hosts: [status.hyper.dev, cert.hyper.dev]
`

// targetsRepo is a repository holding hyper.yaml and one targets/ file per
// entry, keyed by the basename the file takes.
func targetsRepo(t *testing.T, declarations map[string]string) string {
	t.Helper()
	root := newRepo(t)
	for basename, content := range declarations {
		writeFile(t, filepath.Join(root, "targets", basename), content)
	}
	return root
}

// runTargets drives `hyper targets` against root with the arguments given, in
// an environment with nothing in it — which is what a case that says nothing
// about a variable means: absent.
func runTargets(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	return runTargetsIn(t, root, nil, args...)
}

// runTargetsIn drives the command against an environment the case states: a
// name the map carries is set, to that value, and a name it does not carry is
// not set at all. The two are different answers on this command's presence
// column, which is why the environment is a lookup rather than a getter.
func runTargetsIn(t *testing.T, root string, env map[string]string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	lookupenv := func(name string) (string, bool) {
		value, present := env[name]
		return value, present
	}
	exit = cli.RunTargets(append([]string{"--repo-dir", root}, args...), &out, &errs, lookupenv, root, "1.4.0")
	return out.String(), errs.String(), exit
}

// targetRowFixture is one row of the --json stream as a case reads it back. It
// is declared here rather than shared with the command, so that a test
// asserting a member's name is asserting the wire's own spelling — hosts, and
// not endpoint.
type targetRowFixture struct {
	Type               string                 `json:"type"`
	Name               string                 `json:"name"`
	Hosts              []string               `json:"hosts"`
	AcceptsKinds       []string               `json:"accepts_kinds"`
	GrantsCapabilities []string               `json:"grants_capabilities"`
	Credentials        []credentialRowFixture `json:"credentials"`
}

// credentialRowFixture is one credential entry as a case reads it back.
// Present is a pointer because absent-from-the-object and false are two
// different readings: a slot naming no variable has no presence to state.
type credentialRowFixture struct {
	Slot    string `json:"slot"`
	Env     string `json:"env"`
	Present *bool  `json:"present"`
}

// readTargetRows reads the target rows off a --json stream, leaving the
// terminal row to the cases that read it themselves.
func readTargetRows(t *testing.T, stdout string) []targetRowFixture {
	t.Helper()
	var rows []targetRowFixture
	for _, line := range jsonLines(t, stdout) {
		var row targetRowFixture
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
		if row.Type == "target" {
			rows = append(rows, row)
		}
	}
	return rows
}

// targetNames is the row set's names in the order they arrived, which is the
// fact §9 makes normative.
func targetNames(rows []targetRowFixture) []string {
	var got []string
	for _, row := range rows {
		got = append(got, row.Name)
	}
	return got
}

// TestRunTargets_WritesOneRowPerTargetDeclaration is the command's whole
// answer: what the repository grants, one row per declaration, and exit 0.
func TestRunTargets_WritesOneRowPerTargetDeclaration(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"cloudflare-prod.yaml": cloudflareProdTarget,
		"local.yaml":           localTargetDeclaration,
	})

	stdout, stderr, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}

	rows := readTargetRows(t, stdout)
	if got, want := targetNames(rows), []string{"cloudflare-prod", "local"}; !slices.Equal(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if got, want := rows[0].AcceptsKinds, []string{"read", "mutate", "destroy"}; !slices.Equal(got, want) {
		t.Errorf("accepts_kinds = %v, want the declaration's kinds: %v", got, want)
	}
	if got, want := rows[0].GrantsCapabilities, []string{"http"}; !slices.Equal(got, want) {
		t.Errorf("grants_capabilities = %v, want the declaration's capabilities: %v", got, want)
	}
}

// TestRunTargets_OrdersByNameAndNotByPath is §9's normative order, held by a
// fixture whose files sort in the opposite direction to the names they declare.
func TestRunTargets_OrdersByNameAndNotByPath(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"aaa-last.yaml":  namedTarget("zulu"),
		"zzz-first.yaml": namedTarget("alpha"),
	})

	stdout, _, _ := runTargets(t, root, "--json")

	if got, want := targetNames(readTargetRows(t, stdout)), []string{"alpha", "zulu"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v; the order is the name's, not the path's", got, want)
	}
}

// TestRunTargets_TheOrderIsByteExactOverUTF8 is the comparison §9 fixes, held
// by names an ASCII-only fixture could not tell apart from any other order: the
// bytes are compared as they stand, with no locale collation and no
// normalisation — the same fold §9 fixes for matching a name, and the same one
// a Record identity is compared under (§7, §12).
func TestRunTargets_TheOrderIsByteExactOverUTF8(t *testing.T) {
	// Both names are ordered by a locale's collation nothing like the way
	// their bytes are: Ångström sorts among the A's under collation and
	// after every ASCII name by bytes, its first byte being 0xC3; zürich
	// sorts beside zorn under collation and after it by bytes, ü being two
	// bytes where o is one.
	root := targetsRepo(t, map[string]string{
		"zurich.yaml":   namedTarget("zürich"),
		"zorn.yaml":     namedTarget("zorn"),
		"angstrom.yaml": namedTarget("Ångström"),
		"alpha.yaml":    namedTarget("alpha"),
	})

	stdout, _, _ := runTargets(t, root, "--json")

	want := []string{"alpha", "zorn", "zürich", "Ångström"}
	if got := targetNames(readTargetRows(t, stdout)); !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v — byte-exact over UTF-8, not a locale's collation", got, want)
	}
}

// namedTarget is §3's local declaration under a name of the caller's choosing:
// the cases about order care which name a declaration carries and nothing else
// about it.
func namedTarget(name string) string {
	return strings.Replace(localTargetDeclaration, "target: local", "target: "+name, 1)
}

// TestRunTargets_HostsIsAnArrayCarryingEveryHostTheDeclarationGrants is one of
// #106's four flagged resolutions: §9 said *its endpoint* and §3's declaration
// has no such key, and §9 now names it hosts on both its surfaces. A Target
// granting several hosts grants all of them, and a grant silently reduced to
// its first member is not a grant (ADR-0024, ADR-0029).
func TestRunTargets_HostsIsAnArrayCarryingEveryHostTheDeclarationGrants(t *testing.T) {
	root := targetsRepo(t, map[string]string{"multi.yaml": `kind: target-declaration
target: multi
class: hostco
kinds: [read]
capabilities: [http]
hosts: [zulu.hostco.dev, alpha.hostco.dev, mike.hostco.dev]
`})

	stdout, _, _ := runTargets(t, root, "--json")
	rows := readTargetRows(t, stdout)
	if len(rows) != 1 {
		t.Fatalf("rows = %v, want one", targetNames(rows))
	}
	want := []string{"zulu.hostco.dev", "alpha.hostco.dev", "mike.hostco.dev"}
	if !slices.Equal(rows[0].Hosts, want) {
		t.Errorf("hosts = %v, want %v — every host, in the declaration's own order", rows[0].Hosts, want)
	}

	table, _, _ := runTargets(t, root)
	for _, host := range want {
		if !strings.Contains(table, host) {
			t.Errorf("the table does not carry %q; a grant reduced to some of its members is not a grant\n%s", host, table)
		}
	}
}

// TestRunTargets_TheFieldIsNamedHostsInBothRenderings is the other half of that
// resolution: §12's opening rule — one fact reaching two wires reaches them
// under one name — decides it in favour of the artefact's own key, and
// milestone 11's MCP tool follows this name.
func TestRunTargets_TheFieldIsNamedHostsInBothRenderings(t *testing.T) {
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	stream, _, _ := runTargets(t, root, "--json")
	if !strings.Contains(stream, `"hosts":["api.cloudflare.com"]`) {
		t.Errorf("the stream is %q, want the grant under the key hosts", stream)
	}
	if strings.Contains(stream, "endpoint") {
		t.Errorf("the stream names an endpoint: %q; §3's declaration has no such key", stream)
	}

	table, _, _ := runTargets(t, root)
	header := strings.SplitN(table, "\n", 2)[0]
	if !strings.Contains(header, "HOSTS") || strings.Contains(strings.ToUpper(header), "ENDPOINT") {
		t.Errorf("the table's header is %q, want HOSTS", header)
	}
}

// TestRunTargets_EachCredentialSlotIsPairedWithItsVariable is why a bare list
// of names will not do: a declaration may carry slots for more than one scheme,
// and a list of names alone does not say which fills what (§9).
func TestRunTargets_EachCredentialSlotIsPairedWithItsVariable(t *testing.T) {
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	stdout, _, _ := runTargets(t, root, "--json")
	rows := readTargetRows(t, stdout)
	if len(rows[0].Credentials) != 1 {
		t.Fatalf("credentials = %v, want one entry", rows[0].Credentials)
	}
	got := rows[0].Credentials[0]
	if got.Slot != "token" || got.Env != "CLOUDFLARE_API_TOKEN" {
		t.Errorf("credential = %+v, want the slot token paired with CLOUDFLARE_API_TOKEN", got)
	}

	table, _, _ := runTargets(t, root)
	if !strings.Contains(table, "token=CLOUDFLARE_API_TOKEN") {
		t.Errorf("the table is %q, want the pair token=CLOUDFLARE_API_TOKEN rather than a bare variable name", table)
	}
}

// TestRunTargets_ADeclarationCarryingTwoSlotsReportsBoth is the same rule where
// it bites: basic:'s two slots are two variables, and which one is the username
// is not recoverable from a list of names.
func TestRunTargets_ADeclarationCarryingTwoSlotsReportsBoth(t *testing.T) {
	root := targetsRepo(t, map[string]string{"hostco.yaml": twoSlotDeclaration})

	stdout, _, _ := runTargetsIn(t, root, map[string]string{"HOSTCO_USERNAME": "someone"}, "--json")
	rows := readTargetRows(t, stdout)
	if len(rows[0].Credentials) != 2 {
		t.Fatalf("credentials = %v, want both slots", rows[0].Credentials)
	}
	for _, want := range []credentialRowFixture{
		{Slot: "username", Env: "HOSTCO_USERNAME", Present: boolPtr(true)},
		{Slot: "password", Env: "HOSTCO_PASSWORD", Present: boolPtr(false)},
	} {
		found := false
		for _, got := range rows[0].Credentials {
			if got.Slot != want.Slot {
				continue
			}
			found = true
			if got.Env != want.Env || got.Present == nil || *got.Present != *want.Present {
				t.Errorf("slot %s = %+v, want env %s present %v", got.Slot, got, want.Env, *want.Present)
			}
		}
		if !found {
			t.Errorf("no entry for the slot %s; every slot the declaration carries is reported", want.Slot)
		}
	}
}

// twoSlotDeclaration carries slots for both halves of a basic: scheme, which is
// the case a bare list of variable names cannot state.
const twoSlotDeclaration = `kind: target-declaration
target: hostco
class: hostco
kinds: [read]
capabilities: [http]
hosts: [api.hostco.dev]
auth:
  username: {env: HOSTCO_USERNAME}
  password: {env: HOSTCO_PASSWORD}
`

func boolPtr(b bool) *bool { return &b }

// TestRunTargets_PresenceIsComputedWhenTheCommandRuns is the acceptance
// criterion stated as the experiment it describes: one repository, two
// environments, two outputs differing in the presence column and nowhere else.
func TestRunTargets_PresenceIsComputedWhenTheCommandRuns(t *testing.T) {
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	without, _, _ := runTargets(t, root, "--json")
	with, _, _ := runTargetsIn(t, root, map[string]string{"CLOUDFLARE_API_TOKEN": "a-value-nothing-reads"}, "--json")

	if without == with {
		t.Fatalf("the two environments produced identical output %q; presence is asked of the environment at the moment of the call", without)
	}
	if got, want := without, strings.Replace(with, `"present":true`, `"present":false`, 1); got != want {
		t.Errorf("the two outputs differ outside the presence member:\n %q\n %q", got, want)
	}
}

// TestRunTargets_AnEmptyStringVariableIsPresent is the whole of what presence
// asks: is this variable set. Whether an empty credential works is the
// endpoint's business, and hyper has no opinion about a value it does not look
// at.
func TestRunTargets_AnEmptyStringVariableIsPresent(t *testing.T) {
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	stdout, _, _ := runTargetsIn(t, root, map[string]string{"CLOUDFLARE_API_TOKEN": ""}, "--json")

	rows := readTargetRows(t, stdout)
	if got := rows[0].Credentials[0].Present; got == nil || !*got {
		t.Errorf("present = %v for a variable set to the empty string, want true", got)
	}
}

// TestRunTargets_NoCredentialValueReachesTheOutput is ADR-0007 on the one
// surface that asks the environment anything: the question is whether the
// variable is set, and the value behind a present name is never read, rendered
// or logged.
func TestRunTargets_NoCredentialValueReachesTheOutput(t *testing.T) {
	const secret = "s3cr3t-value-that-must-not-be-rendered"
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	for _, mode := range [][]string{{}, {"--json"}} {
		stdout, stderr, _ := runTargetsIn(t, root, map[string]string{"CLOUDFLARE_API_TOKEN": secret}, mode...)
		if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
			t.Errorf("%v: a credential's value reached the output:\n stdout=%q\n stderr=%q", mode, stdout, stderr)
		}
	}
}

// TestRunTargets_PresenceIsReportedForEverySlotTheDeclarationCarries is the
// limit stated on the surface's own terms: a Run resolves the slots its
// bindings require, and this command has no Procedure in hand to narrow by — so
// a slot no Definition binds is reported like any other, and an absence here is
// not by itself a Run that will Refuse.
func TestRunTargets_PresenceIsReportedForEverySlotTheDeclarationCarries(t *testing.T) {
	root := targetsRepo(t, map[string]string{"hostco.yaml": twoSlotDeclaration})
	writeFile(t, filepath.Join(root, "definitions", "reads.yaml"),
		"kind: definition\ndefinition: reads\nprovider: shell\nkinds: [read]\ntargets: [hostco]\n")

	stdout, _, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if got := readTargetRows(t, stdout)[0].Credentials; len(got) != 2 {
		t.Errorf("credentials = %v, want both slots however few of them a Definition binds", got)
	}
}

// TestRunTargets_LocalIsListedLikeEveryOtherTarget is ADR-0041 made visible:
// the Target a Probe binds is authored by the repository like all the others,
// and the reserved name's one visible consequence is an empty credential
// column.
func TestRunTargets_LocalIsListedLikeEveryOtherTarget(t *testing.T) {
	root := targetsRepo(t, map[string]string{"local.yaml": localTargetDeclaration})

	stdout, _, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	rows := readTargetRows(t, stdout)
	if got, want := targetNames(rows), []string{"local"}; !slices.Equal(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	if len(rows[0].Credentials) != 0 {
		t.Errorf("credentials = %v, want none: a declaration named local carries no auth: block", rows[0].Credentials)
	}
	if got, want := rows[0].Hosts, []string{"status.hyper.dev", "cert.hyper.dev"}; !slices.Equal(got, want) {
		t.Errorf("hosts = %v, want %v: local grants enumerated hosts like any other Target", got, want)
	}
}

// TestRunTargets_ASlotNamingNoVariableStatesNoPresence is the ordinary absence
// rule on this row: a slot whose env: is malformed resolves from no variable,
// so there is no variable to ask the environment about and no presence to
// state. The slot is still reported — the declaration carries it, and what is
// wrong with it is check's to name.
func TestRunTargets_ASlotNamingNoVariableStatesNoPresence(t *testing.T) {
	root := targetsRepo(t, map[string]string{"malformed.yaml": `kind: target-declaration
target: malformed
class: hostco
kinds: [read]
capabilities: [http]
hosts: [api.hostco.dev]
auth:
  token: PLAINTEXT
`})

	stdout, _, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	got := readTargetRows(t, stdout)[0].Credentials
	if len(got) != 1 || got[0].Slot != "token" {
		t.Fatalf("credentials = %+v, want the slot the declaration carries", got)
	}
	if got[0].Env != "" || got[0].Present != nil {
		t.Errorf("credential = %+v, want no variable and no presence stated", got[0])
	}
}

// TestRunTargets_NoTargetsDirectoryWritesZeroRowsAndTheTerminalRow is a
// repository that has declared no Target: an absent directory is not an error,
// and the stream still terminates.
func TestRunTargets_NoTargetsDirectoryWritesZeroRowsAndTheTerminalRow(t *testing.T) {
	root := newRepo(t)

	stdout, stderr, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}
	if len(readTargetRows(t, stdout)) != 0 {
		t.Errorf("stdout = %q, want no target row", stdout)
	}
	if stdout != `{"type":"result","truncated":false}`+"\n" {
		t.Errorf("stdout = %q, want the terminal row alone", stdout)
	}

	page, _, _ := runTargets(t, root)
	if page == "" {
		t.Error("the page is empty; what stands where there are no rows is a sentence, not a header over nothing")
	}
}

// TestRunTargets_ADeclarationThatWillNotParseContributesNoRow is ADR-0064's
// rule, already how the Target namespace is built: a file that will not parse
// names nothing, and its faults are check's to report rather than this
// command's to half-render.
func TestRunTargets_ADeclarationThatWillNotParseContributesNoRow(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"local.yaml":  localTargetDeclaration,
		"broken.yaml": "kind: target-declaration\ntarget: broken\n  bad indentation: [\n",
	})

	stdout, stderr, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr)
	}
	if got, want := targetNames(readTargetRows(t, stdout)), []string{"local"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v; an unparseable declaration contributes no row", got, want)
	}
}

// TestRunTargets_ATruncatedResultMarksItselfOnBothSurfaces is the rule a
// truncated result must never look complete: the marker on the terminal row,
// and a line on stderr naming what came back and what did not — in both modes,
// narration being stderr's in either (§9).
func TestRunTargets_ATruncatedResultMarksItselfOnBothSurfaces(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"alpha.yaml": namedTarget("alpha"),
		"zulu.yaml":  namedTarget("zulu"),
	})

	for _, mode := range [][]string{{"--json", "--limit", "1"}, {"--limit", "1"}} {
		stdout, stderr, _ := runTargets(t, root, mode...)

		if want := "returned 1 of 2 Targets; 1 dropped by --limit 1"; !strings.Contains(stderr, want) {
			t.Errorf("%v: stderr = %q, want it to carry %q", mode, stderr, want)
		}
		if mode[0] == "--json" {
			if want := `{"type":"result","truncated":true}`; !strings.Contains(stdout, want) {
				t.Errorf("%v: stdout = %q, want the terminal row %s", mode, stdout, want)
			}
			if got, want := targetNames(readTargetRows(t, stdout)), []string{"alpha"}; !slices.Equal(got, want) {
				t.Errorf("%v: names = %v, want %v — the first N of the order", mode, got, want)
			}
		}
	}
}

// TestRunTargets_TheMarkerNamesNoAxis is m2.5's rule reused: §12 closes the
// truncation axes at two, identity and time, and both are the record's — a
// Target namespace has neither.
func TestRunTargets_TheMarkerNamesNoAxis(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"alpha.yaml": namedTarget("alpha"),
		"zulu.yaml":  namedTarget("zulu"),
	})

	stdout, _, _ := runTargets(t, root, "--json", "--limit", "1")

	lines := jsonLines(t, stdout)
	var marker map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &marker); err != nil {
		t.Fatal(err)
	}
	if _, named := marker["axis"]; named {
		t.Errorf("the terminal row is %v; a Target namespace has neither of §12's two axes", marker)
	}
	if got, want := len(marker), 2; got != want {
		t.Errorf("the terminal row carries %d members (%v), want %d — type and truncated", got, marker, want)
	}
}

// TestRunTargets_TheDefaultLimitSaysItIsTheDefault is what a caller who typed
// no --limit is told when one cut their result anyway: that there is a default,
// what it is, and what widens it.
func TestRunTargets_TheDefaultLimitSaysItIsTheDefault(t *testing.T) {
	declarations := map[string]string{}
	for i := range 60 {
		name := fmt.Sprintf("target-%02d", i)
		declarations[name+".yaml"] = namedTarget(name)
	}
	root := targetsRepo(t, declarations)

	stdout, stderr, exit := runTargets(t, root, "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitClean)
	}
	if got := len(readTargetRows(t, stdout)); got != 50 {
		t.Errorf("returned %d rows, want the default limit of 50", got)
	}
	if want := "returned 50 of 60 Targets; 10 dropped by the default limit of 50 — name a larger --limit for the rest"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want it to carry %q", stderr, want)
	}
}

// TestRunTargets_TakesNoPositionalAndHasNoCursor is §9's tree read the other
// way: `targets` enumerates a namespace and resolves no name in one, and there
// is no way to ask for the next N.
func TestRunTargets_TakesNoPositionalAndHasNoCursor(t *testing.T) {
	root := targetsRepo(t, map[string]string{"local.yaml": localTargetDeclaration})

	stdout, stderr, exit := runTargets(t, root, "local")
	if exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want it silent; a usage error opens no row stream", stdout)
	}
	if !strings.Contains(stderr, "takes no positional argument") {
		t.Errorf("stderr = %q, want it to say the command takes no positional", stderr)
	}

	for _, flag := range []string{"--cursor", "--page", "--offset", "--after"} {
		stdout, stderr, exit := runTargets(t, root, flag, "1")
		if exit != cli.ExitUsage {
			t.Errorf("%s: exit = %d, want %d", flag, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("%s: stdout = %q, want it silent", flag, stdout)
		}
		if !strings.Contains(stderr, "unknown flag "+flag) {
			t.Errorf("%s: stderr = %q, want it to name the unknown flag", flag, stderr)
		}
	}
}

// TestRunTargets_TheGateFiresBeforeTheRepositoryIsLoaded is the pin gate on
// this command: a mismatched pin Refuses 77 with stdout silent, in both modes,
// and the Refusal renders on stderr because a Refusal is not a row (§9,
// ADR-0020).
func TestRunTargets_TheGateFiresBeforeTheRepositoryIsLoaded(t *testing.T) {
	root := targetsRepo(t, map[string]string{"local.yaml": localTargetDeclaration})
	writeFile(t, filepath.Join(root, "hyper.yaml"),
		"kind: repository-declaration\nversion: 9.9.9\ndigest: sha256:0000000000000000000000000000000000000000000000000000000000000000\n")

	for _, mode := range [][]string{{}, {"--json"}} {
		stdout, stderr, exit := runTargets(t, root, mode...)
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

// TestRunTargets_CannotExitOne is the shape of the command: it reports facts,
// not problems found, so the code that means "problems found" is unreachable —
// including against a repository every other command has something to say
// about.
func TestRunTargets_CannotExitOne(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"local.yaml":       localTargetDeclaration,
		"broken.yaml":      "kind: target-declaration\ntarget: broken\n  bad: [\n",
		"unknown-key.yaml": "kind: target-declaration\ntarget: unknown-key\nbogus: 1\n",
	})
	writeFile(t, filepath.Join(root, "definitions", "typo.yaml"), "kind: definition\ndefinition: typo\nprovider: nowhere\n")

	for _, mode := range [][]string{{}, {"--json"}, {"--limit", "1"}} {
		if _, _, exit := runTargets(t, root, mode...); exit == cli.ExitProblems {
			t.Errorf("%v: exit = %d; `hyper targets` reports facts, not problems found", mode, exit)
		}
	}
}

// TestRunTargets_TwoInvocationsAgainstOneRepositoryAreByteIdentical is what
// makes the answer diffable, and what makes truncation mean something: the
// order is fixed, so nothing here depends on a map's iteration.
func TestRunTargets_TwoInvocationsAgainstOneRepositoryAreByteIdentical(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"cloudflare-prod.yaml": cloudflareProdTarget,
		"hostco.yaml":          twoSlotDeclaration,
		"local.yaml":           localTargetDeclaration,
	})

	for _, mode := range [][]string{{"--json"}, {}} {
		first, _, _ := runTargetsIn(t, root, map[string]string{"HOSTCO_PASSWORD": ""}, mode...)
		for i := range 8 {
			again, _, _ := runTargetsIn(t, root, map[string]string{"HOSTCO_PASSWORD": ""}, mode...)
			if again != first {
				t.Fatalf("%v: invocation %d wrote different bytes:\n %q\n %q", mode, i+2, first, again)
			}
		}
	}
}

// TestRunTargets_TheRowSetIsTheTargetNamespace is the fence on the fold: the
// rows are the names a Definition's targets: resolves against, so a Target
// reachable from a Definition and absent from the list that is supposed to
// enumerate it is a failure here rather than a discovery in milestone 5.
func TestRunTargets_TheRowSetIsTheTargetNamespace(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"aaa-last.yaml": namedTarget("zulu"),
		"alpha.yaml":    namedTarget("alpha"),
		"broken.yaml":   "kind: target-declaration\ntarget: broken\n  bad: [\n",
		"nameless.yaml": "kind: target-declaration\nclass: local\n",
	})

	loaded, err := repository.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	want := slices.Sorted(maps.Keys(loaded.Targets))

	stdout, _, _ := runTargets(t, root, "--json")
	if got := targetNames(readTargetRows(t, stdout)); !slices.Equal(got, want) {
		t.Errorf("the rows name %v and the Target namespace holds %v", got, want)
	}
}

// TestRunTargets_TheTableAndTheStreamStateTheSameFacts is ADR-0026 held over
// this command: one row set written twice, so the two surfaces cannot say
// different things.
func TestRunTargets_TheTableAndTheStreamStateTheSameFacts(t *testing.T) {
	root := targetsRepo(t, map[string]string{
		"cloudflare-prod.yaml": cloudflareProdTarget,
		"local.yaml":           localTargetDeclaration,
	})
	env := map[string]string{"CLOUDFLARE_API_TOKEN": "unread"}

	stream, _, _ := runTargetsIn(t, root, env, "--json")
	table, _, _ := runTargetsIn(t, root, env)

	tableLines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	rows := readTargetRows(t, stream)
	if got, want := len(tableLines), len(rows)+1; got != want {
		t.Fatalf("the table has %d lines and the stream %d rows; want a header and one line per row", got, want-1)
	}
	for i, row := range rows {
		line := tableLines[i+1]
		cells := append([]string{row.Name}, row.Hosts...)
		cells = append(cells, row.AcceptsKinds...)
		cells = append(cells, row.GrantsCapabilities...)
		for _, credential := range row.Credentials {
			cells = append(cells, credential.Slot+"="+credential.Env)
		}
		for _, cell := range cells {
			if !strings.Contains(line, cell) {
				t.Errorf("table line %q does not carry %q", line, cell)
			}
		}
	}
}

// TestRunTargets_ReadsNothingOfTheEnvironmentButTheGlobalsAndTheNamedVariables
// is "reaches nothing" as this command can state it: it is the one surface that
// asks the environment anything, and what it asks about is the variables the
// declarations themselves name, beside §9's two globals. No Store is reached
// because nothing here knows where one would be, and no request is made because
// the answer is the repository load and one lookup per slot.
func TestRunTargets_ReadsNothingOfTheEnvironmentButTheGlobalsAndTheNamedVariables(t *testing.T) {
	root := targetsRepo(t, map[string]string{"cloudflare-prod.yaml": cloudflareProdTarget})

	var asked []string
	lookupenv := func(name string) (string, bool) {
		asked = append(asked, name)
		return "", false
	}
	var stdout, stderr bytes.Buffer
	if exit := cli.RunTargets([]string{"--repo-dir", root}, &stdout, &stderr, lookupenv, root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want %d; stderr=%q", exit, cli.ExitClean, stderr.String())
	}

	allowed := map[string]bool{"HYPER_REPO_DIR": true, "NO_COLOR": true, "CLOUDFLARE_API_TOKEN": true}
	for _, name := range asked {
		if !allowed[name] {
			t.Errorf("the command read %s; the only variables it reads are the two globals' and the ones the declarations name", name)
		}
	}
	if !slices.Contains(asked, "CLOUDFLARE_API_TOKEN") {
		t.Error("the credential's variable was never looked up; presence is asked of the environment at the moment of the call")
	}
}

// TestRunTargets_TheThreeGlobalsAreTheOnesSectionNineCloses: --repo-dir has its
// environment spelling, --no-color has its, and neither spelling of the latter
// changes a byte — there being no colour on this page to suppress.
func TestRunTargets_TheThreeGlobalsAreTheOnesSectionNineCloses(t *testing.T) {
	root := targetsRepo(t, map[string]string{"local.yaml": localTargetDeclaration})
	elsewhere := t.TempDir()

	var viaEnv bytes.Buffer
	lookupenv := func(name string) (string, bool) {
		if name == "HYPER_REPO_DIR" {
			return root, true
		}
		return "", false
	}
	if exit := cli.RunTargets(nil, &viaEnv, &viaEnv, lookupenv, elsewhere, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("HYPER_REPO_DIR: exit = %d, want %d; output=%q", exit, cli.ExitClean, viaEnv.String())
	}

	viaFlag, _, _ := runTargets(t, root)
	if viaEnv.String() != viaFlag {
		t.Errorf("HYPER_REPO_DIR wrote %q and --repo-dir wrote %q", viaEnv.String(), viaFlag)
	}

	flagged, _, _ := runTargets(t, root, "--no-color")
	if flagged != viaFlag {
		t.Errorf("--no-color changed the bytes:\n %q\n %q", viaFlag, flagged)
	}

	noColorEnv, _, _ := runTargetsIn(t, root, map[string]string{"NO_COLOR": "1"})
	if noColorEnv != viaFlag {
		t.Errorf("NO_COLOR changed the bytes:\n %q\n %q", viaFlag, noColorEnv)
	}
}

// TestRunTargets_ReachesNoNetworkNoStoreAndInvokesNothing fences the command's
// own file, on the shape `version` and `completions` are fenced by: what a
// command can reach is what it imports, and this one imports its streams, the
// repository load, the artefact facts and the renderer. No net, no os/exec, no
// Store — the whole answer is the load and one lookup per credential slot.
//
// The fence matters more here than on its neighbours rather than less: this is
// the one command in the milestone that reaches outside the repository at all,
// and what it is allowed to ask the environment is whether a name is set (§9,
// ADR-0007).
func TestRunTargets_ReachesNoNetworkNoStoreAndInvokesNothing(t *testing.T) {
	allowed := map[string]bool{
		`"fmt"`:     true,
		`"io"`:      true,
		`"maps"`:    true,
		`"slices"`:  true,
		`"strings"`: true,
		`"github.com/TheLoomLabs/hyper/internal/artefact"`:   true,
		`"github.com/TheLoomLabs/hyper/internal/render"`:     true,
		`"github.com/TheLoomLabs/hyper/internal/repository"`: true,
	}

	file, err := parser.ParseFile(token.NewFileSet(), "targets.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range file.Imports {
		if !allowed[imp.Path.Value] {
			t.Errorf("internal/cli/targets.go imports %s; `hyper targets` reaches no network, reads no Store, and invokes nothing", imp.Path.Value)
		}
	}
}

// TestTargetsCorpus_NoCaseExitsOne is the corpus half of the rule the command
// states: `hyper targets` reports facts, not problems found, so exit 1 is
// unreachable from it however faulty the repository it read.
func TestTargetsCorpus_NoCaseExitsOne(t *testing.T) {
	cases, err := os.ReadDir(filepath.Join("testdata", "targets"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("the targets corpus is empty; the invariant would hold vacuously")
	}

	for _, entry := range cases {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join("testdata", "targets", entry.Name(), "exit.golden")
		recorded, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if exit := strings.TrimSpace(string(recorded)); exit == strconv.Itoa(cli.ExitProblems) {
			t.Errorf("%s records exit %s; `hyper targets` reports facts, not problems found", path, exit)
		}
	}
}
