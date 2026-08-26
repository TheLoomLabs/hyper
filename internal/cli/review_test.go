package cli_test

import (
	"bytes"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// runReview drives `hyper review` against root with the arguments given, in an
// environment with nothing in it: the repository is named by the flag, which is
// what every case here means by "against this repository".
func runReview(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	exit = cli.RunReview(append([]string{"--repo-dir", root}, args...), cli.Streams(&out, &errs), reviewProcess(emptyEnvironment), root, "1.4.0")
	return out.String(), errs.String(), exit
}

// reviewProcess is the process a review is driven with: the environment the
// case supplies, and the suite's one fixed instant.
//
// The clock is the whole of what this command reaches for beyond the
// environment, and it reaches it for the age the header renders beside the
// gloss (§8, §10). Every other member is left nil, which is what says a review
// mints nothing, dials nothing and starts no child.
func reviewProcess(lookupenv func(string) (string, bool)) cli.Process {
	return cli.Process{LookupEnv: lookupenv, Now: func() time.Time { return fixedInstant }}
}

// sourceOf is the source a review rendered, read back off the page: the lines
// beneath the rule, each with the gutter that prefixes it taken off. The gutter
// is found by the bar it ends with, which is the one character separating a
// review's two halves on every line of the screen (§8).
//
// It exists so that a case about the *source* asserts against the file's own
// bytes rather than against a second copy of the layout: what the criterion
// asks is that every line renders verbatim, and a case that spelled the prefix
// out would fail the day the marker column widens for a reason that is not the
// source's.
func sourceOf(t *testing.T, page string) []string {
	t.Helper()

	var source []string
	past := false
	for _, line := range strings.Split(strings.TrimSuffix(page, "\n"), "\n") {
		if strings.Contains(line, "┼") {
			past = true
			continue
		}
		if !past {
			continue
		}
		if line == "" {
			// The source ends at the one line beneath the rule that
			// carries no gutter at all: the blank separating the
			// AUTHORITY block, which is a table of its own and not
			// one of the artefact's lines. A line the *artefact*
			// left empty is not this — it renders as its gutter,
			// trimmed, and still carries the bar (§8).
			break
		}
		bar := strings.Index(line, "│")
		if bar < 0 {
			t.Fatalf("line %q beneath the rule carries no gutter; every rendered line does", line)
		}
		source = append(source, strings.TrimPrefix(line[bar+len("│"):], " "))
	}
	return source
}

// artefactWithEveryShapeOfLine is a Repository declaration written the way an
// author writes one and then some: a comment above a key, a comment beside a
// key, a blank line inside the body, and a line indented under a key. Every one
// of them is a byte of the working tree, and §8 says a review renders all of
// them in place.
const artefactWithEveryShapeOfLine = `# The pin every Run in this repository is gated on.
kind: repository-declaration
version: 1.4.0

retention: 90d  # ninety days of Records
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
store:
  branch: hyper-store
`

// TestRunReview_RendersEveryLineOfTheArtefactVerbatim is the whole of what this
// command renders in this milestone: the working tree's file, comments in
// place, indentation unchanged, blank lines counted, nothing re-encoded and
// nothing truncated (§3, §8, ADR-0057).
func TestRunReview_RendersEveryLineOfTheArtefactVerbatim(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/hyper.yaml", artefactWithEveryShapeOfLine)

	stdout, stderr, exit := runReview(t, root, "hyper.yaml")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	// The expectation is the fixture cut at its own newlines by the standard
	// library rather than a second copy of it written out here: what is
	// asserted is that the page carries the file's own lines, so the other
	// side of the comparison may not be made by the code under test.
	want := strings.Split(strings.TrimSuffix(artefactWithEveryShapeOfLine, "\n"), "\n")
	if got := sourceOf(t, stdout); !slices.Equal(got, want) {
		t.Errorf("the source rendered as\n %q\nwant\n %q", got, want)
	}
}

// TestRunReview_ALineIsRenderedWithItsOwnWhitespace holds the verbatim rule at
// the one place a renderer is tempted to tidy: trailing spaces an author's
// editor left are the artefact's bytes and render with it. The padding the
// gutter writes is not, and a line the source ends does not end in ours.
func TestRunReview_ALineIsRenderedWithItsOwnWhitespace(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/hyper.yaml", "kind: repository-declaration\nversion: 1.4.0   \n\nretention: 90d\n")

	stdout, _, exit := runReview(t, root, "hyper.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	rendered := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if !slices.ContainsFunc(rendered, func(line string) bool { return strings.HasSuffix(line, "version: 1.4.0   ") }) {
		t.Errorf("no rendered line ends in the trailing spaces the file wrote:\n%s", stdout)
	}
	for _, line := range rendered {
		if strings.HasSuffix(line, "│ ") {
			t.Errorf("the blank line rendered as %q; a line the source left empty ends where the gutter does", line)
		}
	}
}

// TestRunReview_ANamePositionalResolvesAgainstTheArtefactsOwnName is the first
// of the positional's two forms: anything without a `/` and not ending `.yaml`
// is a name, matched against what the artefact declares for itself rather than
// against its filename (§9, ADR-0060).
func TestRunReview_ANamePositionalResolvesAgainstTheArtefactsOwnName(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/definitions/preview-dns.yaml", "kind: definition\ndefinition: preview-dns\nprovider: cloudflare-dns\nkinds: [mutate]\ntargets: [cloudflare-prod]\n")

	stdout, stderr, exit := runReview(t, root, "preview-dns")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	if !strings.Contains(stdout, "definitions/preview-dns.yaml") || !strings.Contains(stdout, "DEFINITION") {
		t.Errorf("the review reads\n%s\nwant the Definition the name resolves to", stdout)
	}
}

// TestRunReview_ANameDifferingOnlyInCaseResolvesToNothing is the fold being
// hyper's rather than the filesystem's: a macOS filesystem is case-insensitive
// and a runner's is not, so a name matched by an `open` would render on a
// laptop and exit 2 in CI (§9, ADR-0060).
func TestRunReview_ANameDifferingOnlyInCaseResolvesToNothing(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/deploy.yaml", "kind: procedure\nprocedure: deploy\ntargets: []\nsteps: []\n")

	stdout, stderr, exit := runReview(t, root, "Deploy")
	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d: a name differing in case matches nothing", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout carries %q; a usage error opens no stream in either mode", stdout)
	}
	if !strings.Contains(stderr, `"Deploy"`) || !strings.Contains(stderr, "namespaces") {
		t.Errorf("stderr reads %q, want the name that was typed and the namespaces it was resolved against", stderr)
	}
	if strings.Contains(stderr, "deploy") {
		t.Errorf("stderr reads %q; a usage error suggests no near miss (ADR-0047)", stderr)
	}
}

// TestRunReview_ANameInTwoNamespacesIsAUsageErrorNamingBoth is the one fault
// the name form has that the path form cannot: two artefacts of two kinds
// declaring one name. The command cannot pick, so it says which kinds it
// matched and points at the form that names a file.
func TestRunReview_ANameInTwoNamespacesIsAUsageErrorNamingBoth(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/definitions/shared.yaml", "kind: definition\ndefinition: shared\nprovider: p\nkinds: [read]\ntargets: [t]\n")
	writeFile(t, root+"/targets/shared.yaml", "kind: target-declaration\ntarget: shared\nclass: local\nkinds: [read]\n")

	stdout, stderr, exit := runReview(t, root, "shared")
	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d for a name matching two artefact kinds", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout carries %q; a usage error opens no stream in either mode", stdout)
	}
	for _, want := range []string{"Definition", "Target declaration", ".yaml"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr reads %q, want it to carry %q", stderr, want)
		}
	}
}

// TestRunReview_APathPositionalReachesTheArtefactWithNoName is the other half
// of why both forms are mandatory: `hyper.yaml` declares no name, so a path is
// the only thing that reaches it.
func TestRunReview_APathPositionalReachesTheArtefactWithNoName(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "hyper.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review of the Repository declaration", exit)
	}
	if !strings.Contains(stdout, "REPOSITORY") {
		t.Errorf("the review reads\n%s\nwant the Repository declaration's own kind heading the marker column", stdout)
	}
}

// TestRunReview_TheBuiltinManifestIsReachedByNameAndNeverByAPath is the first
// half of ADR-0068 rendered: the one artefact with no file in the repository is
// named, never opened, and the pseudo-path the load carries it under is not a
// positional anybody can type.
func TestRunReview_TheBuiltinManifestIsReachedByNameAndNeverByAPath(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "shell")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want the built-in Manifest rendered", exit)
	}
	if !strings.Contains(stdout, "kind: provider") || !strings.Contains(stdout, "provider: shell") {
		t.Errorf("the review reads\n%s\nwant the bytes compiled into the binary", stdout)
	}

	_, _, exit = runReview(t, root, "<built-in>/shell")
	if exit != cli.ExitUsage {
		t.Errorf("exit = %d for the pseudo-path, want %d: a path resolves against the load's files and the built-in has none", exit, cli.ExitUsage)
	}
}

// TestRunReview_TheBuiltinManifestsHeaderStatesTheSentenceAlone is ADR-0068's
// rendering: `path` is silenced because the sentence beside it already says
// where the bytes are, and the line collapses to its one field rather than
// leaving the path's width as blank run-up.
func TestRunReview_TheBuiltinManifestsHeaderStatesTheSentenceAlone(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "shell")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want the built-in Manifest rendered", exit)
	}

	// The header is read off the page rather than spelled out with its
	// gutter, on markersOf's own rule: what this case asks is what the
	// header's line carries, and a case that wrote the marker column's
	// width into its expectation would fail the day this Manifest's own
	// Operations widen it.
	if got, want := headerOf(t, stdout), []string{"no baseline — shell ships in the binary"}; !slices.Equal(got, want) {
		t.Errorf("the header is\n %q\nwant\n %q", got, want)
	}
}

// TestRunReview_TheAbsenceIsRankedAsAPipeline is §8's two live stages: the
// built-in Manifest answers `built-in` and every artefact with a file answers
// `no-store`, which is the true answer for every repository until `store init`
// exists. Each is reachable only where the one before it did not fire.
func TestRunReview_TheAbsenceIsRankedAsAPipeline(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/targets/local.yaml", "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\n")

	for _, c := range []struct{ named, want string }{
		{"shell", `"baseline_absent":"built-in"`},
		{"local", `"baseline_absent":"no-store"`},
		{"hyper.yaml", `"baseline_absent":"no-store"`},
	} {
		stdout, _, exit := runReview(t, root, c.named, "--json")
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s --json exited %d, want a clean review", c.named, exit)
		}
		if !strings.Contains(stdout, c.want) {
			t.Errorf("hyper review %s --json wrote\n%s\nwant a row carrying %s", c.named, stdout, c.want)
		}
	}
}

// TestRunReview_TheStreamIsTheHeaderRowTheGutterAndTheTerminalRow is what
// `--json` carries in this milestone, whole: the header, one `gutter` row per
// rendered line the gutter has something to say about, and the terminal row —
// whose marker is false because nothing on this screen is a result set (§8,
// §9).
//
// The source is not on it and no row for it is: a review does not decompose
// into rows the way the change tables do, the consumer already having the file.
// The artefact is a Repository declaration with no `retention:`, so the one
// mark is the pin: a line that is not in the file has no cell, which is
// different from a line rendering a blank one (§8, issue #122).
func TestRunReview_TheStreamIsTheHeaderRowTheGutterAndTheTerminalRow(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "hyper.yaml", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := []string{
		`{"type":"artefact","kind":"repository-declaration","path":"hyper.yaml","baseline_absent":"no-store"}`,
		`{"type":"gutter","line":2,"marker":"1.4.0"}`,
		`{"type":"result","truncated":false}`,
	}
	if got := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); !slices.Equal(got, want) {
		t.Errorf("the stream is\n %q\nwant\n %q", got, want)
	}
}

// TestRunReview_PathIsAbsentExactlyWhereTheBaselineIsBuiltIn is §8's rule on
// the wire: an artefact with no file has no path, and no Run could have
// recorded a revision of what has none — one fact, and the name already on the
// row is the discriminator (ADR-0068).
func TestRunReview_PathIsAbsentExactlyWhereTheBaselineIsBuiltIn(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "shell", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want the built-in Manifest rendered", exit)
	}
	if strings.Contains(stdout, `"path"`) {
		t.Errorf("the stream carries a path:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"baseline_absent":"built-in"`) {
		t.Errorf("the stream is\n%s\nwant the built-in absence beside the absent path", stdout)
	}
}

// TestRunReview_TakesNoLimit is §9's rule for a command that names one thing:
// nothing on this screen is a result set, so `--limit` is not a flag `review`
// has and naming one is a usage error like any other unknown flag.
func TestRunReview_TakesNoLimit(t *testing.T) {
	root := newRepo(t)

	stdout, stderr, exit := runReview(t, root, "hyper.yaml", "--limit", "2")
	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d for a flag review does not have", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout carries %q; a usage error opens no stream", stdout)
	}
	if !strings.Contains(stderr, "unknown flag --limit") {
		t.Errorf("stderr reads %q, want the unknown-flag message every command writes", stderr)
	}
}

// TestRunReview_AnArtefactThatWillNotLoadWritesChecksRowAndExitsOne is §9's
// second exit code: found and faulty is not a rendering, and what it writes is
// the row `check` writes for the same file.
func TestRunReview_AnArtefactThatWillNotLoadWritesChecksRowAndExitsOne(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/definitions/broken.yaml", "kind: definition\ndefinition: [unclosed\n")

	stdout, stderr, exit := runReview(t, root, "definitions/broken.yaml")
	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d for an artefact that will not load", exit, cli.ExitProblems)
	}
	if stderr != "" {
		t.Errorf("stderr carries %q; the rows are the answer and go to stdout", stderr)
	}
	if !strings.Contains(stdout, "definitions/broken.yaml") {
		t.Errorf("stdout reads %q, want check's row for the file", stdout)
	}
}

// TestRunReview_AnArtefactThatNamesOneThatIsNotThereRendersAndExitsZero is
// ADR-0064's line: an artefact that loads and names something missing is
// neither of the other two exits — the fault is `check`'s to report and this
// surface's to annotate.
func TestRunReview_AnArtefactThatNamesOneThatIsNotThereRendersAndExitsZero(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/definitions/orphan.yaml", "kind: definition\ndefinition: orphan\nprovider: not-there\nkinds: [read]\ntargets: [nowhere]\n")

	stdout, stderr, exit := runReview(t, root, "orphan")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want the artefact rendered", exit, stderr)
	}
	if !strings.Contains(stdout, "provider: not-there") {
		t.Errorf("the review reads\n%s\nwant the line naming what is not there rendered like any other", stdout)
	}
}

// TestRunReview_APathMatchingNothingIsAUsageError is the path form's own
// unresolved answer: a usage error, no stream, and a message naming what was
// typed and where an artefact's file can be.
func TestRunReview_APathMatchingNothingIsAUsageError(t *testing.T) {
	root := newRepo(t)

	stdout, stderr, exit := runReview(t, root, "definitions/typo.yaml")
	if exit != cli.ExitUsage {
		t.Fatalf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if stdout != "" {
		t.Errorf("stdout carries %q; a usage error opens no stream", stdout)
	}
	if !strings.Contains(stderr, `"definitions/typo.yaml"`) {
		t.Errorf("stderr reads %q, want the path that was typed", stderr)
	}
}

// TestRunReview_NoPositionalAndTwoPositionalsAreBothUsageErrors is the arity
// fault decided from the argument list alone, before any repository is
// resolved: one caller forgot the artefact and the other named it twice
// (ADR-0060).
func TestRunReview_NoPositionalAndTwoPositionalsAreBothUsageErrors(t *testing.T) {
	root := newRepo(t)

	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "names no artefact"},
		{[]string{"hyper.yaml", "shell"}, "takes one artefact, got 2"},
	} {
		stdout, stderr, exit := runReview(t, root, c.args...)
		if exit != cli.ExitUsage {
			t.Errorf("hyper review %v exited %d, want %d", c.args, exit, cli.ExitUsage)
		}
		if stdout != "" {
			t.Errorf("hyper review %v wrote %q to stdout; a usage error opens no stream", c.args, stdout)
		}
		if !strings.Contains(stderr, c.want) {
			t.Errorf("hyper review %v wrote %q, want it to carry %q", c.args, stderr, c.want)
		}
	}
}

// TestRunReview_TheThreeGlobalsBehaveAsTheyDoEverywhereElse holds §9's
// configuration layers at this command: --repo-dir and its environment
// spelling name the repository, and --no-color and NO_COLOR produce identical
// bytes because no page carries colour of its own to suppress (§9, ADR-0014).
// There is no fourth global, which is what the unknown-flag arm above says.
func TestRunReview_TheThreeGlobalsBehaveAsTheyDoEverywhereElse(t *testing.T) {
	root := newRepo(t)
	elsewhere := t.TempDir() // a working directory with no repository of its own

	fromEnv := func(name string) (string, bool) {
		if name == "HYPER_REPO_DIR" {
			return root, true
		}
		return "", false
	}
	var named, environed bytes.Buffer
	if exit := cli.RunReview([]string{"--repo-dir", root, "hyper.yaml"}, cli.Streams(&named, &named), reviewProcess(emptyEnvironment), root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("--repo-dir exited %d, want a clean review", exit)
	}
	if exit := cli.RunReview([]string{"hyper.yaml"}, cli.Streams(&environed, &environed), reviewProcess(fromEnv), elsewhere, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("HYPER_REPO_DIR exited %d, want a clean review", exit)
	}
	if named.String() != environed.String() {
		t.Errorf("the flag rendered\n%s\nand the variable rendered\n%s", named.String(), environed.String())
	}

	noColorSet := func(name string) (string, bool) {
		if name == "NO_COLOR" {
			return "1", true
		}
		return "", false
	}
	var flagged, environedColour bytes.Buffer
	if exit := cli.RunReview([]string{"--repo-dir", root, "--no-color", "hyper.yaml"}, cli.Streams(&flagged, &flagged), reviewProcess(emptyEnvironment), root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("--no-color exited %d, want a clean review", exit)
	}
	if exit := cli.RunReview([]string{"--repo-dir", root, "hyper.yaml"}, cli.Streams(&environedColour, &environedColour), reviewProcess(noColorSet), root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("NO_COLOR exited %d, want a clean review", exit)
	}
	if flagged.String() != named.String() || environedColour.String() != named.String() {
		t.Errorf("--no-color and NO_COLOR do not render what the plain invocation does:\n%q\n%q\n%q",
			flagged.String(), environedColour.String(), named.String())
	}
}

// TestRunReview_ResolvesNoCredential is §9's claim about this surface asserted
// rather than left to hold by habit: a review reaches nothing, so the only
// variables it reads are the two globals every command reads. A credential's
// variable is named by a Target declaration and this command never asks the
// environment for one — which is what makes `review` answerable on a fresh
// clone with nothing exported.
func TestRunReview_ResolvesNoCredential(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/targets/cloudflare-prod.yaml",
		"kind: target-declaration\ntarget: cloudflare-prod\nclass: cloudflare\nkinds: [read]\ncapabilities: [http]\nhosts: [api.cloudflare.com]\nauth:\n  token: {env: CLOUDFLARE_API_TOKEN}\n")

	var asked []string
	watching := func(name string) (string, bool) {
		asked = append(asked, name)
		return "", false
	}
	var out bytes.Buffer
	if exit := cli.RunReview([]string{"--repo-dir", root, "cloudflare-prod"}, cli.Streams(&out, &out), reviewProcess(watching), root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	globals := map[string]bool{"NO_COLOR": true, "HYPER_REPO_DIR": true}
	for _, name := range asked {
		if !globals[name] {
			t.Errorf("the review read %s from the environment; it resolves no credential", name)
		}
	}
}

// procedureDeclaring is a Procedure written around one Cadence: the least
// artefact that carries the header's second line, so that a case about the
// gloss is about the gloss.
func procedureDeclaring(cadence string) string {
	return "kind: procedure\nprocedure: nightly\ntargets: [local]\ncadence: \"" + cadence + "\"\nsteps: []\n"
}

// headerOf is the header a review rendered, read back off the page: the lines
// above the rule, each with the gutter that prefixes it taken off.
func headerOf(t *testing.T, page string) []string {
	t.Helper()

	var header []string
	for _, line := range strings.Split(strings.TrimSuffix(page, "\n"), "\n") {
		if strings.Contains(line, "┼") {
			return header
		}
		bar := strings.Index(line, "│")
		if bar < 0 {
			t.Fatalf("line %q above the rule carries no gutter; every rendered line does", line)
		}
		header = append(header, strings.TrimPrefix(line[bar+len("│"):], "  "))
	}
	t.Fatalf("the page carries no rule:\n%s", page)
	return nil
}

// TestRunReview_TheWorkedExpressionsRenderOnTheHeadersSecondLine drives §10's
// eleven worked expressions through a Procedure header, which is the gloss's
// first consumer and the surface the rule was written for (§8, ADR-0063).
//
// The phrases are the specification's own, copied byte for byte: a phrase that
// differs by a byte is a different sentence, and a reader who checks a gloss
// against §10 is checking exactly this.
func TestRunReview_TheWorkedExpressionsRenderOnTheHeadersSecondLine(t *testing.T) {
	for _, worked := range []struct{ cadence, gloss string }{
		{"0 3 * * 1", "03:00 UTC every Monday · ≈4.3 runs/month"},
		{"0 0 1 * *", "00:00 UTC on the 1st of the month · 1 run/month"},
		{"0 0 * * *", "00:00 UTC every day · ≈30 runs/month"},
		{"*/5 * * * *", "every 5 minutes · ≈8800 runs/month"},
		{"*/7 * * * *", "at :00, :07, :14, :21, :28, :35, :42, :49 and :56 past every hour · ≈6600 runs/month"},
		{"0-59 * * * *", "every minute from :00 to :59 past every hour · ≈44000 runs/month"},
		{"0 9,17 * * 1-5", "09:00 and 17:00 UTC every day from Monday to Friday · ≈43 runs/month"},
		{"0 9-17 * * *", "at :00 past every hour from 09:00 to 17:00 UTC, every day · ≈270 runs/month"},
		{"0 0 1 * 1", "00:00 UTC on the 1st of the month or any Monday · ≈5.2 runs/month"},
		{"0 0 1 */3 *", "00:00 UTC on the 1st in January, April, July and October · ≈0.33 runs/month"},
		{"0 0 29 2 *", "00:00 UTC on the 29th of February · ≈0.020 runs/month"},
	} {
		root := newRepo(t)
		writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring(worked.cadence))

		stdout, stderr, exit := runReview(t, root, "nightly")
		if exit != cli.ExitClean || stderr != "" {
			t.Fatalf("%s: exit = %d, stderr = %q, want a clean review", worked.cadence, exit, stderr)
		}
		header := headerOf(t, stdout)
		if len(header) != 2 {
			t.Fatalf("%s: the header is %q, want two lines", worked.cadence, header)
		}
		// Every one of the eleven opens its minute field on `0`, on
		// `*` or on a step over the whole span, so all eleven carry
		// both of §10's facts beside the gloss. What this case asserts
		// is the gloss; the facts are held where they are derived and
		// on each surface that renders them (review_facts_test.go).
		want := worked.gloss + " · " + defaultBranchFact + " · " + hourBoundaryFact
		if header[1] != want {
			t.Errorf("%s glossed\n %q\nwant\n %q", worked.cadence, header[1], want)
		}
	}
}

// TestRunReview_AnArtefactWithNoCadenceRendersAOneLineHeader is the other half
// of the rule: the gloss's absence takes the line rather than leaving one blank
// (§8). A Procedure declaring none and every artefact that could not declare one
// answer the same way, the header keying on the supply rather than on the kind.
func TestRunReview_AnArtefactWithNoCadenceRendersAOneLineHeader(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/by-hand.yaml", "kind: procedure\nprocedure: by-hand\ntargets: [local]\nsteps: []\n")
	writeFile(t, root+"/targets/local.yaml", "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\n")

	for _, named := range []string{"by-hand", "local", "hyper.yaml", "shell"} {
		stdout, _, exit := runReview(t, root, named)
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s exited %d, want a clean review", named, exit)
		}
		if header := headerOf(t, stdout); len(header) != 1 {
			t.Errorf("hyper review %s rendered the header %q, want one line", named, header)
		}
	}
}

// TestRunReview_TheGlossIsAboveTheRule is where §8 puts it and why: below the
// rule it would sit directly over `kind: procedure` and read as an annotation
// of that line, which is the one thing the header may not be (ADR-0063).
func TestRunReview_TheGlossIsAboveTheRule(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring("0 3 * * 1"))

	stdout, _, exit := runReview(t, root, "nightly")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	gloss := strings.Index(stdout, "03:00 UTC every Monday")
	rule := strings.Index(stdout, "┼")
	if gloss < 0 || rule < 0 || gloss > rule {
		t.Errorf("the gloss is not above the rule:\n%s", stdout)
	}
	if source := sourceOf(t, stdout); slices.ContainsFunc(source, func(line string) bool {
		return strings.Contains(line, "runs/month")
	}) {
		t.Errorf("the gloss rendered among the source lines:\n%s", stdout)
	}
}

// TestRunReview_TheWireCarriesTheGlossesPartsAndNeverTheComposedString is §8's
// `artefact` row with the Cadence on it: the expression as written, the phrase,
// and the rate as a JSON number at the two significant figures the page rounded
// to. Carrying the composed string beside them would be one fact in two
// representations that can disagree (§8, §10, ADR-0063).
//
// `last_run` is absent: one Journal absence serves both readings of the missing
// entry, so the header states it once on the range's line and this line carries
// only what the artefact's own bytes supply.
func TestRunReview_TheWireCarriesTheGlossesPartsAndNeverTheComposedString(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring("0 3 * * 1"))

	stdout, _, exit := runReview(t, root, "nightly", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := []string{
		`{"type":"artefact","kind":"procedure","path":"procedures/nightly.yaml","baseline_absent":"no-store","cadence":"0 3 * * 1","phrase":"03:00 UTC every Monday","rate":4.3}`,
		// The envelope mark rides between them, a Procedure declaring
		// no Steps still declaring the envelope none of them exceeds
		// (issue #120).
		`{"type":"gutter","line":3,"marker":"envelope ok"}`,
		// And the flag that indexes it, which is the row every
		// Procedure review carries (issue #123).
		`{"type":"flag","flag":"envelope","cites_line":3}`,
		`{"type":"result","truncated":false}`,
	}
	if got := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); !slices.Equal(got, want) {
		t.Errorf("the stream is\n %q\nwant\n %q", got, want)
	}
	if strings.Contains(stdout, "last_run") {
		t.Errorf("the row carries last_run:\n%s", stdout)
	}
}

// TestRunReview_AnUnreadableCadenceRendersNoGloss holds the division of labour
// between the two offline surfaces: `cadence-malformed` is `check`'s, over a
// repository whole, and a review of one artefact refuses nothing. The review
// renders the artefact and exits clean, with no gloss to render and no line
// taken for one.
func TestRunReview_AnUnreadableCadenceRendersNoGloss(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring("@hourly"))

	stdout, stderr, exit := runReview(t, root, "nightly")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want the artefact rendered and nothing refused", exit, stderr)
	}
	if header := headerOf(t, stdout); len(header) != 1 {
		t.Errorf("the header is %q, want the one line an artefact with no readable Cadence renders", header)
	}
	if !slices.Contains(sourceOf(t, stdout), `cadence: "@hourly"`) {
		t.Errorf("the artefact's own line is missing:\n%s", stdout)
	}
}

// TestRunReview_ACadenceOnAnArtefactThatIsNotAProcedureIsNotOne holds the
// supply: a Cadence is a Procedure's, and a key of that name anywhere else is
// not a recurrence for this header to gloss (§10).
func TestRunReview_ACadenceOnAnArtefactThatIsNotAProcedureIsNotOne(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncadence: \"0 3 * * 1\"\n")

	stdout, _, exit := runReview(t, root, "local")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if header := headerOf(t, stdout); len(header) != 1 {
		t.Errorf("the header is %q, want one line: a Target declaration declares no Cadence", header)
	}
}

// markersOf is the marker column a review rendered, read back off the page:
// every line beneath the rule that carries something left of the bar, keyed by
// the line of the artefact it stands beside.
//
// It reads the column rather than the whole line for the reason sourceOf reads
// the other half: what a case about a marker asks is what the cell says, and a
// case that spelled the padding out would fail the day another marker in the
// same rendering widens the column.
func markersOf(t *testing.T, page string) map[int]string {
	t.Helper()

	markers := map[int]string{}
	line := 0
	past := false
	for _, rendered := range strings.Split(strings.TrimSuffix(page, "\n"), "\n") {
		if strings.Contains(rendered, "┼") {
			past = true
			continue
		}
		if !past {
			continue
		}
		if rendered == "" {
			// The gutter ends at the blank separating the AUTHORITY
			// block, whose lines are a table of their own and mark
			// nothing. A line the artefact left empty still carries
			// the bar (§8).
			break
		}
		line++
		bar := strings.Index(rendered, "│")
		if bar < 0 {
			t.Fatalf("line %q beneath the rule carries no gutter; every rendered line does", rendered)
		}
		if marker := strings.TrimSpace(rendered[:bar]); marker != "" {
			markers[line] = marker
		}
	}
	return markers
}

// providerDeclaring is a Manifest whose one Operation is named against its
// Kind: `delete_everything` declaring `read` is what a gutter reading the
// Manifest renders `read` and a gutter guessing from the name renders
// otherwise (§12).
const providerDeclaring = `kind: provider
provider: things
schema-version: 1
class: things
capabilities: [http]
operations:
  delete_everything:
    kind: read
    http:
      method: GET
      host: "{from-target}"
      path: /things
    record:
      identity: $.id
      fields: {id: $.id}
  make_thing:
    kind: mutate
    http:
      method: POST
      host: "{from-target}"
      path: /things
    record:
      identity: $.id
      fields: {id: $.id}
  end_thing:
    kind: destroy
    http:
      method: DELETE
      host: "{from-target}"
      path: /things/{thing_id}
`

// reviewRepo is a repository with a Provider, a Definition claiming every Kind
// and a Target declaration granting them, so that a case about a Procedure's
// gutter is about the Procedure. procedure is written to
// procedures/subject.yaml, which is what every case below reviews.
func reviewRepo(t *testing.T, procedure string) string {
	t.Helper()
	root := newRepo(t)
	writeFile(t, root+"/providers/things.yaml", providerDeclaring)
	writeFile(t, root+"/targets/staging.yaml",
		"kind: target-declaration\ntarget: staging\nclass: things\nkinds: [read, mutate, destroy]\ncapabilities: [http]\nhosts: [api.things.dev]\n")
	// Two Definitions against one Provider, because a Definition observes
	// or effects and never both (ADR-0032): the Records a series holds take
	// their type from the Kinds claimed, and one claim spanning both would
	// write an Observation and an Asset into one series.
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate]\ndestroy: [end_thing]\ntargets: [staging]\n")
	writeFile(t, root+"/definitions/things-observed.yaml",
		"kind: definition\ndefinition: things-observed\nprovider: things\nkinds: [read]\ntargets: [staging]\n")
	writeFile(t, root+"/procedures/subject.yaml", procedure)
	return root
}

// TestRunReview_AStepsKindIsReadFromTheManifestAndMarkedBesideItsLine is the
// gutter's first fact: the Kind an Operation declares, marked beside the line
// that binds it, and never inferred from the Operation's name (§8, §12).
func TestRunReview_AStepsKindIsReadFromTheManifestAndMarkedBesideItsLine(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`)

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	if got, want := markersOf(t, stdout)[5], "read  staging"; got != want {
		t.Errorf("the marker beside line 5 is %q, want %q", got, want)
	}
}

// TestRunReview_AMutateStepWithNoBoundIsMarkedWithTheBang is §8's one mark
// whose absence no static check reports: a `mutate` Step with no declared Bound
// is `mutate!`, and one carrying a Bound is `mutate`. `check` is silent on it by
// design, so the gutter is where the fact is rendered at all (§4, §8).
//
// The two Steps sit in one rendering, which is also where the Kind field's
// padding shows: `mutate` and `mutate!` line their Targets up under each other.
func TestRunReview_AMutateStepWithNoBoundIsMarkedWithTheBang(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: unbounded
    definition: things
    operation: make_thing
    target: staging
  - id: bounded
    definition: things
    operation: make_thing
    target: staging
    bound: 3
`)

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	markers := markersOf(t, stdout)
	if got, want := markers[5], "mutate!  staging"; got != want {
		t.Errorf("the Step with no bound: is marked %q, want %q", got, want)
	}
	if got, want := markers[9], "mutate   staging"; got != want {
		t.Errorf("the Step carrying a bound: is marked %q, want %q", got, want)
	}
}

// TestRunReview_ADestroyStepRendersDESTROYAndCarriesNoBang is the other half of
// the same rule. `destroy` renders upper-case for the eye, and a `destroy` Step
// with no `bound:` gets no `!`: that is `bound-missing`, a static check, and
// `check`'s to report — there is no `DESTROY!` marker (§4, §8, §12).
func TestRunReview_ADestroyStepRendersDESTROYAndCarriesNoBang(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: retire
    definition: things
    operation: end_thing
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[5], "DESTROY  staging"; got != want {
		t.Errorf("the destroy Step is marked %q, want %q", got, want)
	}
}

// TestRunReview_TheProceduresTargetsLineCarriesTheEnvelopeMark is the one mark
// that is not a Step's: the envelope check quantifies over every Step's
// `target:` at once, so it stands beside the line that declares the envelope
// and takes no part in the Kind/Target alignment (§8).
func TestRunReview_TheProceduresTargetsLineCarriesTheEnvelopeMark(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[3], "envelope ✓"; got != want {
		t.Errorf("the targets: line is marked %q, want %q", got, want)
	}
}

// TestRunReview_AnEnvelopeTheStepsExceedStillRendersAndExitsZero is what
// separates this surface from `check`: a review does not run it and does not
// decline, so an artefact carrying `envelope-exceeded` renders like any other
// and the mark is what says so (§8, §9, ADR-0064).
func TestRunReview_AnEnvelopeTheStepsExceedStillRendersAndExitsZero(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: []
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`)

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want the artefact rendered and nothing refused", exit, stderr)
	}
	markers := markersOf(t, stdout)
	if got, want := markers[3], "envelope ✗"; got != want {
		t.Errorf("the targets: line is marked %q, want %q", got, want)
	}
	if got, want := markers[5], "read  staging"; got != want {
		t.Errorf("the Step outside the envelope is marked %q, want %q — the Step's own claim is unchanged", got, want)
	}
}

// TestRunReview_AStepInvokingAnOpaqueOperationIsMarkedOpaqueBesideItsKind is
// §8's third gutter fact: opacity is a Manifest fact exactly as a Kind is, and
// the gutter carries it for the reason it carries the Kind — what `hyper`
// cannot describe is not readable from the Step's own lines. The token sits
// between the Kind and the Target (§8, §12).
func TestRunReview_AStepInvokingAnOpaqueOperationIsMarkedOpaqueBesideItsKind(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [local]
steps:
  - id: run
    definition: commands
    operation: read
    target: local
`)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncapabilities: [shell]\n")
	writeFile(t, root+"/definitions/commands.yaml",
		"kind: definition\ndefinition: commands\nprovider: shell\nkinds: [read]\ntargets: [local]\n")

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[5], "read  opaque  local"; got != want {
		t.Errorf("the Step invoking an Opaque Operation is marked %q, want %q", got, want)
	}
}

// TestRunReview_ANestedInvocationRendersItsTransitiveEnvelope is §8's
// composition rule in the gutter: what a nested invocation's own line derives
// is the transitive envelope §3 states — what everything it invokes may touch,
// to any depth — and that envelope reaches this Procedure's own envelope mark.
func TestRunReview_ANestedInvocationRendersItsTransitiveEnvelope(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: inner
    procedure: deeper
`)
	writeFile(t, root+"/procedures/deeper.yaml", `kind: procedure
procedure: deeper
targets: [staging]
steps:
  - id: onwards
    procedure: deepest
`)
	writeFile(t, root+"/procedures/deepest.yaml", "kind: procedure\nprocedure: deepest\ntargets: [staging]\nsteps: []\n")

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	markers := markersOf(t, stdout)
	if got, want := markers[5], "staging"; got != want {
		t.Errorf("the invocation is marked %q, want %q", got, want)
	}
	if got, want := markers[3], "envelope ✓"; got != want {
		t.Errorf("the targets: line is marked %q, want %q", got, want)
	}
}

// TestRunReview_ATransitiveEnvelopeOutsideTheDeclaredOneReachesTheMark is the
// other side of it, one level down: `deepest` binds a Target `subject` never
// declared, and the envelope mark on `subject`'s own `targets:` line is what
// says so — which is the walk `envelope-exceeded` already makes, read again
// rather than made twice.
func TestRunReview_ATransitiveEnvelopeOutsideTheDeclaredOneReachesTheMark(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: inner
    procedure: deeper
`)
	writeFile(t, root+"/procedures/deeper.yaml", `kind: procedure
procedure: deeper
targets: [staging]
steps:
  - id: onwards
    procedure: deepest
`)
	writeFile(t, root+"/procedures/deepest.yaml", "kind: procedure\nprocedure: deepest\ntargets: [production]\nsteps: []\n")
	writeFile(t, root+"/targets/production.yaml",
		"kind: target-declaration\ntarget: production\nclass: things\nkinds: [read]\ncapabilities: [http]\nhosts: [api.things.dev]\n")

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	markers := markersOf(t, stdout)
	if got, want := markers[5], "production staging"; got != want {
		t.Errorf("the invocation is marked %q, want %q", got, want)
	}
	if got, want := markers[3], "envelope ✗"; got != want {
		t.Errorf("the targets: line is marked %q, want %q", got, want)
	}
}

// TestRunReview_ANameResolvingToNothingIsMarkedUnresolved is §8's fourth mark
// and the one that fires where the other three cannot: a review does not run
// `check` and does not decline, so a Step whose `definition:`, `operation:` or
// bound Provider is not there renders like any other line with the mark its
// derivation would have carried — one name and not four, the gutter marking and
// not classifying, and a blank cell would say `read` by omission on the one
// screen that may not (§8, §12, ADR-0026, ADR-0064).
func TestRunReview_ANameResolvingToNothingIsMarkedUnresolved(t *testing.T) {
	// Every unresolved Step binds a Target this Procedure never declared,
	// so the envelope mark below is the assertion that an unresolved Step
	// contributes nothing to the check: were the mark reading a `target:`
	// whose Step it could derive nothing else about, it would read `✗`.
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: no-definition
    definition: not-there
    operation: delete_everything
    target: elsewhere
  - id: no-operation
    definition: things
    operation: not-there
    target: elsewhere
  - id: no-provider
    definition: orphan
    operation: delete_everything
    target: elsewhere
  - id: no-procedure
    procedure: not-there
`)
	writeFile(t, root+"/definitions/orphan.yaml",
		"kind: definition\ndefinition: orphan\nprovider: not-there\nkinds: [read]\ntargets: [staging]\n")

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want the artefact rendered and nothing refused", exit, stderr)
	}
	markers := markersOf(t, stdout)
	for _, line := range []int{5, 9, 13, 17} {
		if got, want := markers[line], "unresolved"; got != want {
			t.Errorf("line %d is marked %q, want %q", line, got, want)
		}
	}
	if got, want := markers[3], "envelope ✓"; got != want {
		t.Errorf("the targets: line is marked %q, want %q — an unresolved Step contributes no envelope", got, want)
	}
}

// columnWidthOf is the marker column's own width, read back off the rule that
// spans it: the rule opens with the screen's indent and runs to the junction,
// carrying the column and the gap that separates it from the bar.
func columnWidthOf(t *testing.T, page string) int {
	t.Helper()

	for _, line := range strings.Split(page, "\n") {
		junction := strings.Index(line, "┼")
		if junction < 0 {
			continue
		}
		return utf8.RuneCountInString(line[:junction]) - len("  ") - len("  ")
	}
	t.Fatalf("the page carries no rule:\n%s", page)
	return 0
}

// TestRunReview_NoMarkerIsTruncatedHoweverLongTheTargetName is §8's rule about
// what this screen may not do: a Target name is an identity in a reviewed
// artefact, and eliding a character of one is the review stating something
// other than what is about to be approved. Nothing here is sized to the
// terminal — §9's truncation discipline governs a result set, which has an
// order and a limit, and an artefact has neither.
func TestRunReview_NoMarkerIsTruncatedHoweverLongTheTargetName(t *testing.T) {
	long := strings.Repeat("staging-", 30) + "eu-central"
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [`+long+`]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: `+long+`
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[5], "read  "+long; got != want {
		t.Errorf("the marker is %q, want %q", got, want)
	}
	if got, want := columnWidthOf(t, stdout), utf8.RuneCountInString("read  "+long); got != want {
		t.Errorf("the marker column is %d wide, want %d — as wide as the widest marker in the rendering", got, want)
	}
}

// TestRunReview_TheMarkerColumnFallsBackToTheKindHeading is the other half of
// the width rule: the column is as wide as the widest marker, or the word
// heading it where that is wider, so the heading is never the thing that gets
// clipped (§8).
func TestRunReview_TheMarkerColumnFallsBackToTheKindHeading(t *testing.T) {
	// No targets: line, so no envelope mark, and one Step whose whole
	// marker is narrower than `PROCEDURE`. The missing key is a fault
	// `check` reports and this surface renders past (ADR-0064).
	root := reviewRepo(t, `kind: procedure
procedure: subject
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: eu
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[4], "read  eu"; got != want {
		t.Errorf("the marker is %q, want %q", got, want)
	}
	if got, want := columnWidthOf(t, stdout), utf8.RuneCountInString("PROCEDURE"); got != want {
		t.Errorf("the marker column is %d wide, want %d — the kind heading is wider than every marker", got, want)
	}
}

// TestRunReview_ACommentRendersInPlaceAndEntersNoMarker is §8's rule about
// whose text the column carries: the gutter carries what `hyper` derived about
// a line, and a column two authors write into is one where a marker and a
// comment contend for a cell that holds one of them. A comment is source, it
// renders verbatim inside the line it was written on, and it is read for
// nothing else (§3, §8).
func TestRunReview_ACommentRendersInPlaceAndEntersNoMarker(t *testing.T) {
	source := `kind: procedure
procedure: subject
targets: [staging]
steps:
  # the one Step, and what an author thought of it
  - id: look  # reads and writes nothing
    definition: things-observed
    operation: delete_everything
    target: staging
`
	root := reviewRepo(t, source)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	if got := sourceOf(t, stdout); !slices.Equal(got, want) {
		t.Errorf("the source rendered as\n %q\nwant\n %q", got, want)
	}
	markers := markersOf(t, stdout)
	if got, want := markers[6], "read  staging"; got != want {
		t.Errorf("the Step's own line is marked %q, want %q", got, want)
	}
	if marker, marked := markers[5]; marked {
		t.Errorf("the comment's own line is marked %q; nothing an author wrote enters the marker column", marker)
	}
}

// TestRunReview_TheChangeColumnStaysSeparateAndZeroWide is §8's two-column
// rule at the one milestone where the second column has no supply: no range
// opens, so the change column has no content **and no width**, and the source
// sits one character right of the bar. No marker composes a change mark into
// itself — reading down the marker column *is* the step table, and a fact about
// the artefact's history interleaved into it is that column ceasing to be one
// table's.
func TestRunReview_TheChangeColumnStaysSeparateAndZeroWide(t *testing.T) {
	source := `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`
	root := reviewRepo(t, source)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	// One character stands between the bar and the artefact's own line, and
	// it is the gap: the column that would have carried the change mark is
	// not drawn at all, so the source sits two characters left of where a
	// ranged review would put it.
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	written := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	// The source opens at the rule rather than closing at the last line of
	// the page: what stands beneath it is the AUTHORITY block (§8).
	rule := slices.IndexFunc(lines, func(line string) bool { return strings.Contains(line, "┼") })
	rendered := lines[rule+1 : rule+1+len(written)]
	for i, line := range rendered {
		bar := strings.Index(line, "│")
		if bar < 0 {
			t.Fatalf("the line %q carries no gutter", line)
		}
		if got, want := line[bar+len("│"):], " "+written[i]; got != want {
			t.Errorf("the line %q stands to the right of the bar, want %q", got, want)
		}
	}
	if strings.Contains(stdout, "~") {
		t.Errorf("the page carries a change mark; nothing marks a line until the change column lands (issue #168):\n%s", stdout)
	}
}

// TestRunReview_TheWireCarriesOneGutterRowPerRenderedLineWithContent is §8's
// `gutter` row: one rendered **line** and not one marked cell, carrying the
// marker as the string the page renders with its alignment padding collapsed to
// single spaces. `envelope ✓` goes out `envelope ok` and `DESTROY` goes out as
// it renders — the sigil and the word are one fact in two notations, exactly as
// `~` and `changed: true` will be.
//
// The rows are the annotations and never the source: the consumer already has
// the file (§8).
func TestRunReview_TheWireCarriesOneGutterRowPerRenderedLineWithContent(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: make
    definition: things
    operation: make_thing
    target: staging
  - id: retire
    definition: things
    operation: end_thing
    target: staging
    bound: 5
`)

	stdout, _, exit := runReview(t, root, "subject", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := []string{
		`{"type":"artefact","kind":"procedure","path":"procedures/subject.yaml","baseline_absent":"no-store"}`,
		`{"type":"gutter","line":3,"marker":"envelope ok"}`,
		`{"type":"gutter","line":5,"marker":"mutate! staging"}`,
		`{"type":"gutter","line":9,"marker":"DESTROY staging"}`,
		`{"type":"authority","definition":"things","target":"staging","definition_kinds":["mutate","destroy"],"target_kinds":["read","mutate","destroy"],"effective":["mutate","destroy"],"destroy_operations":["end_thing"]}`,
		`{"type":"flag","flag":"unbounded","cites_line":5,"step":"make"}`,
		`{"type":"flag","flag":"destroy","cites_line":9,"step":"retire"}`,
		`{"type":"flag","flag":"envelope","cites_line":3}`,
		`{"type":"result","truncated":false}`,
	}
	if got := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); !slices.Equal(got, want) {
		t.Errorf("the stream is\n %q\nwant\n %q", got, want)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "definition: things") {
			t.Errorf("the stream carries the source: %q", line)
		}
	}
}

// TestRunReview_ALineWithNothingInEitherColumnGetsNoRow is the other half of
// that rule, and it is what keeps the stream the annotations rather than the
// file: a line the gutter says nothing about is a line with no row, so the
// count of rows is the count of facts.
func TestRunReview_ALineWithNothingInEitherColumnGetsNoRow(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := strings.Count(stdout, `"type":"gutter"`), 2; got != want {
		t.Errorf("the stream carries %d gutter rows, want %d — one for targets: and one for the Step:\n%s", got, want, stdout)
	}
}

// TestRunReview_AnUnresolvedAndAnOpaqueMarkerGoOutAsThePageComposesThem holds
// the collapse at the two markers whose page notation has padding of more than
// one kind: a whole-cell marker that takes no part in the Kind/Target
// alignment, and one carrying the opacity field between them.
func TestRunReview_AnUnresolvedAndAnOpaqueMarkerGoOutAsThePageComposesThem(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [local]
steps:
  - id: run
    definition: commands
    operation: destroy
    target: local
  - id: lost
    definition: not-there
    operation: read
    target: local
`)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [destroy]\ncapabilities: [shell]\nopaque-destroy: true\n")
	writeFile(t, root+"/definitions/commands.yaml",
		"kind: definition\ndefinition: commands\nprovider: shell\ndestroy: [destroy]\ntargets: [local]\n")

	stdout, _, exit := runReview(t, root, "subject", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	for _, want := range []string{
		`{"type":"gutter","line":5,"marker":"DESTROY opaque local"}`,
		`{"type":"gutter","line":9,"marker":"unresolved"}`,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the stream is\n%s\nwant a row carrying %s", stdout, want)
		}
	}
}

// The other four artefacts' rosters (issue #122). Only a Procedure has Steps,
// so only a Procedure carries a Kind, a Target, a Bound or an envelope mark;
// what the other four mark is their own, and §8 fixes it artefact by artefact.

// providerWithAScheme is a Manifest carrying every line its roster marks: an
// Auth scheme, the Capabilities its Operations require, and one Operation of
// each Kind — two of them stating their Repeatability by omission. The scheme
// is the mark whose line does not name it: `auth:` says there is one and the
// key beneath it says which (§13).
const providerWithAScheme = `kind: provider
provider: things
schema-version: 1
class: things
capabilities: [http]
auth:
  header: {name: Authorization, prefix: "Bearer "}
operations:
  list_things:
    kind: read
    http:
      method: GET
      host: "{from-target}"
      path: /things
    record:
      identity: $.id
      fields: {id: $.id}
  make_thing:
    kind: mutate
    http:
      method: POST
      host: "{from-target}"
      path: /things
    record:
      identity: $.id
      fields: {id: $.id}
  end_thing:
    kind: destroy
    repeatability: repeatable
    http:
      method: DELETE
      host: "{from-target}"
      path: /things/{thing_id}
`

// rosterRepo is a repository with one artefact of each of §12's five kinds in
// it, each written so that every line its own roster marks is there: a
// Repository declaration with a retention policy, a Manifest with a scheme and
// three Operations, a Target declaration granting hosts and a credential slot,
// a Definition claiming both effectful Kinds, and a Procedure binding them.
//
// It checks clean, which every case reading it depends on: an artefact the load
// found a fault in is `check`'s row and exit 1 rather than a rendering (§9).
func rosterRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root+"/hyper.yaml", "kind: repository-declaration\nversion: 1.4.0\n"+
		"digest: sha256:0000000000000000000000000000000000000000000000000000000000000000\nretention: 90d\n")
	writeFile(t, root+"/providers/things.yaml", providerWithAScheme)
	writeFile(t, root+"/targets/staging.yaml",
		"kind: target-declaration\ntarget: staging\nclass: things\nkinds: [read, mutate, destroy]\n"+
			"capabilities: [http]\nhosts: [api.things.dev, api.things.eu]\nauth:\n  token: {env: THINGS_API_TOKEN}\n")
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate, destroy]\ndestroy: [end_thing]\ntargets: [staging]\n")
	writeFile(t, root+"/procedures/subject.yaml",
		"kind: procedure\nprocedure: subject\ntargets: [staging]\nsteps:\n  - id: retire\n"+
			"    definition: things\n    operation: end_thing\n    target: staging\n    bound: 5\n")
	return root
}

// TestRunReview_ADefinitionsGutterMarksItsThreeClaims is §8's roster on a
// Definition: the Kinds claimed, the `destroy` Operations named and the Targets
// bindable, each beside its own line and all three authored in the file being
// read (§8, issue #122).
func TestRunReview_ADefinitionsGutterMarksItsThreeClaims(t *testing.T) {
	root := rosterRepo(t)

	stdout, stderr, exit := runReview(t, root, "definitions/things.yaml")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	want := map[int]string{
		4: "mutate DESTROY",
		5: "DESTROY end_thing",
		6: "staging",
	}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v", got, want)
	}
}

// TestRunReview_ADefinitionWhoseProviderIsNotThereRendersCompleteAndUnmarked is
// the gutter's supply rule holding rather than the review missing something:
// nothing rendered on this screen is derived from a Manifest, so there is no
// name here for the gutter to fail to follow, the `provider:` line carries no
// mark, and the review exits 0 (§8, ADR-0064).
func TestRunReview_ADefinitionWhoseProviderIsNotThereRendersCompleteAndUnmarked(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/definitions/orphan.yaml",
		"kind: definition\ndefinition: orphan\nprovider: not-there\nkinds: [read]\ntargets: [nowhere]\n")

	stdout, stderr, exit := runReview(t, root, "orphan")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	markers := markersOf(t, stdout)
	if marker, marked := markers[3]; marked {
		t.Errorf("the provider: line is marked %q; nothing on this screen is derived from a Manifest", marker)
	}
	if got, want := markers[4], "read"; got != want {
		t.Errorf("the kinds: line is marked %q, want %q — the Definition's own claim renders as it always does", got, want)
	}
	for line, marker := range markers {
		if strings.Contains(marker, "unresolved") {
			t.Errorf("line %d is marked %q; unresolved is a Procedure's mark and no other artefact's", line, marker)
		}
	}
}

// TestRunReview_ALineThatDerivedNothingRendersUnmarked is the other absence the
// roster collapses into a line with no cell, and it is a different fact about
// the file: a Definition writing `destroy: []` wrote the line and named no
// Operation, so there is nothing derived to mark. `DESTROY` alone there would
// name a grant this Definition never took, and a review does not run `check`
// (§9) — what it renders is what `hyper` derived from these lines and nothing
// else (§8, ADR-0064).
func TestRunReview_ALineThatDerivedNothingRendersUnmarked(t *testing.T) {
	root := rosterRepo(t)
	writeFile(t, root+"/definitions/observed.yaml",
		"kind: definition\ndefinition: observed\nprovider: things\nkinds: [read]\ndestroy: []\ntargets: [staging]\n")

	stdout, stderr, exit := runReview(t, root, "observed")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	want := map[int]string{4: "read", 6: "staging"}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v — the destroy: line names none and carries no mark", got, want)
	}
}

// TestRunReview_ATargetDeclarationsGutterMarksEveryGrant is §8's roster on a
// Target declaration: the Kinds accepted, the Capabilities granted, the hosts
// granted, and each credential slot's environment variable — the variable's
// name and never its value, this surface resolving no credential at all (§7,
// §8, ADR-0007).
func TestRunReview_ATargetDeclarationsGutterMarksEveryGrant(t *testing.T) {
	root := rosterRepo(t)

	stdout, stderr, exit := runReview(t, root, "staging")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	want := map[int]string{
		4: "read mutate DESTROY",
		5: "http",
		6: "api.things.dev api.things.eu",
		8: "THINGS_API_TOKEN",
	}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v", got, want)
	}
}

// TestRunReview_TheOptInAdmittingAnOpaqueDestroyIsMarked is the one grant a
// Target declaration makes that no other line of it makes: the opt-in §4 fixes,
// marked in the two tokens this vocabulary already carries.
func TestRunReview_TheOptInAdmittingAnOpaqueDestroyIsMarked(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/targets/box.yaml",
		"kind: target-declaration\ntarget: box\nclass: local\nkinds: [read, destroy]\n"+
			"capabilities: [shell]\nopaque-destroy: true\n")

	stdout, _, exit := runReview(t, root, "box")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := markersOf(t, stdout)[6], "opaque DESTROY"; got != want {
		t.Errorf("the opt-in is marked %q, want %q", got, want)
	}
}

// TestRunReview_ATargetDeclarationWithNoAuthBlockRendersNoCredentialCell is the
// absence rule this roster is read under: where a line is simply not in the
// file there is no cell, which is a different thing from a line rendering a
// blank one (§8).
func TestRunReview_ATargetDeclarationWithNoAuthBlockRendersNoCredentialCell(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\ncapabilities: [shell]\n")

	stdout, _, exit := runReview(t, root, "local")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := map[int]string{4: "read", 5: "shell"}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v — a declaration with no auth: has no slot to mark", got, want)
	}
}

// TestRunReview_AManifestsGutterMarksItsSchemeItsCapabilitiesAndEachOperation
// is §8's roster on a Manifest, and the Operations are the half of it a
// reviewer with the file open cannot see: a Kind is declared and never inferred
// from the name, and the Repeatability in force has no spelling in the source
// at all (§12).
//
// The `auth:` line is marked with which of §12's two schemes this Manifest
// names, which is the one thing that line does not say — the scheme is the key
// nested beneath it, and the header it composes is what `hyper provider`
// renders (§9, §13).
func TestRunReview_AManifestsGutterMarksItsSchemeItsCapabilitiesAndEachOperation(t *testing.T) {
	root := rosterRepo(t)

	stdout, stderr, exit := runReview(t, root, "providers/things.yaml")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	want := map[int]string{
		5:  "http",
		6:  "header",
		9:  "read     repeatable",
		18: "mutate   run-once",
		27: "DESTROY  repeatable",
	}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v", got, want)
	}
}

// TestRunReview_TheRepeatabilityMarkedIsTheEffectiveOne is §12's derivation
// rendered rather than the Manifest's own key read back: an Operation omitting
// `repeatability:` is run-once where it effects and repeatable where it reads,
// and `run-once` is a word no artefact may write — so a reader who scanned the
// source for it would find nothing (§12, ADR-0037).
func TestRunReview_TheRepeatabilityMarkedIsTheEffectiveOne(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/providers/things.yaml", `kind: provider
provider: things
schema-version: 1
class: things
capabilities: [http]
operations:
  list_things:
    kind: read
    http:
      method: GET
      host: "{from-target}"
      path: /things
    record:
      identity: $.id
      fields: {id: $.id}
  end_thing:
    kind: destroy
    http:
      method: DELETE
      host: "{from-target}"
      path: /things/{thing_id}
`)

	stdout, _, exit := runReview(t, root, "things")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	markers := markersOf(t, stdout)
	if got, want := markers[7], "read     repeatable"; got != want {
		t.Errorf("the read is marked %q, want %q — a read declaring nothing is repeatable", got, want)
	}
	if got, want := markers[16], "DESTROY  run-once"; got != want {
		t.Errorf("the destroy is marked %q, want %q — an effectful Operation declaring nothing is run-once", got, want)
	}
	if strings.Contains(strings.Join(sourceOf(t, stdout), "\n"), "run-once") {
		t.Errorf("the source names run-once; it is a word no artefact may write:\n%s", stdout)
	}
}

// TestRunReview_AManifestWithNoAuthBlockRendersNoSchemeCell is the Manifest's
// half of the absence rule: `none` is what a row renders for a Provider that
// sends no credential, and a line that is not in the file has no cell for it
// (§8, §13).
func TestRunReview_AManifestWithNoAuthBlockRendersNoSchemeCell(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/providers/things.yaml", providerDeclaring)

	stdout, _, exit := runReview(t, root, "things")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	for line, marker := range markersOf(t, stdout) {
		if strings.Contains(marker, "none") || strings.Contains(marker, "header") {
			t.Errorf("line %d is marked %q; this Manifest names no scheme and has no line to mark", line, marker)
		}
	}
}

// TestRunReview_TheBuiltinManifestMarksAllSixOfItsOperations is the Manifest
// roster read over the bytes compiled into the binary, which is the case that
// needs the marker composition whole: every one of the six Operations is
// Opaque, two of them `destroy`, and two state their Repeatability by omission
// (§11, ADR-0039).
func TestRunReview_TheBuiltinManifestMarksAllSixOfItsOperations(t *testing.T) {
	root := newRepo(t)

	stdout, stderr, exit := runReview(t, root, "shell")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want the built-in Manifest rendered", exit, stderr)
	}

	operations := 0
	destroys := 0
	for _, marker := range markersOf(t, stdout) {
		if !strings.Contains(marker, "opaque") {
			continue
		}
		operations++
		if strings.Contains(marker, "DESTROY") {
			destroys++
		}
	}
	if operations != 6 || destroys != 2 {
		t.Errorf("%d Operations are marked opaque and %d of them DESTROY, want 6 and 2:\n%s", operations, destroys, stdout)
	}
	if got, want := markersOf(t, stdout)[5], "shell"; got != want {
		t.Errorf("the capabilities: line is marked %q, want %q", got, want)
	}
}

// TestRunReview_ARepositoryDeclarationsGutterMarksThePinAndTheRetention is §8's
// roster on the artefact that governs every Run: the version pin every Run is
// gated on, and the retention policy that bounds Compaction (§3, §11).
func TestRunReview_ARepositoryDeclarationsGutterMarksThePinAndTheRetention(t *testing.T) {
	root := rosterRepo(t)

	stdout, stderr, exit := runReview(t, root, "hyper.yaml")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}

	want := map[int]string{2: "1.4.0", 4: "90d"}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v — the digest is hyper's own writing and carries no mark", got, want)
	}
}

// TestRunReview_ARepositoryDeclarationWithNoRetentionRendersNoRetentionCell is
// the last of the four absences: a repository whose Records are never removed
// writes no `retention:`, so there is no line and therefore no cell (§8).
func TestRunReview_ARepositoryDeclarationWithNoRetentionRendersNoRetentionCell(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "hyper.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := map[int]string{2: "1.4.0"}
	if got := markersOf(t, stdout); !maps.Equal(got, want) {
		t.Errorf("the marker column is\n %v\nwant\n %v", got, want)
	}
}

// TestRunReview_TheMarkerColumnIsHeadedByTheArtefactsOwnKind is §8's heading on
// all five: the kind of the artefact being read, where a header naming blast
// radius would describe a Procedure's marks and a Definition's and misdescribe
// a Target declaration's hosts and a Repository declaration's retention policy.
func TestRunReview_TheMarkerColumnIsHeadedByTheArtefactsOwnKind(t *testing.T) {
	root := rosterRepo(t)

	for _, c := range []struct{ named, want string }{
		{"definitions/things.yaml", "DEFINITION"},
		{"procedures/subject.yaml", "PROCEDURE"},
		{"providers/things.yaml", "MANIFEST"},
		{"staging", "TARGET"},
		{"hyper.yaml", "REPOSITORY"},
	} {
		stdout, _, exit := runReview(t, root, c.named)
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s exited %d, want a clean review", c.named, exit)
		}
		if got := headingOf(t, stdout); got != c.want {
			t.Errorf("hyper review %s heads the marker column %q, want %q", c.named, got, c.want)
		}
	}
}

// headingOf is the word at the top of the marker column, read back off the
// page: whatever stands left of the bar on the header's first line.
func headingOf(t *testing.T, page string) string {
	t.Helper()

	first := strings.SplitN(page, "\n", 2)[0]
	bar := strings.Index(first, "│")
	if bar < 0 {
		t.Fatalf("the header's first line %q carries no gutter; every rendered line does", first)
	}
	return strings.TrimSpace(first[:bar])
}

// TestRunReview_TheChangeColumnIsAColumnOnAllFiveAndZeroWideOnAllFive is §8's
// statement about which artefacts *have* a range rather than about which
// renderings carry one: no kind is exempt from the column, and in this
// milestone the supply is missing everywhere, so the source sits one character
// right of the bar on every one of the five.
func TestRunReview_TheChangeColumnIsAColumnOnAllFiveAndZeroWideOnAllFive(t *testing.T) {
	root := rosterRepo(t)

	for _, named := range []string{"definitions/things.yaml", "procedures/subject.yaml", "providers/things.yaml", "staging", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named)
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s exited %d, want a clean review", named, exit)
		}
		for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
			bar := strings.Index(line, "│")
			if bar < 0 {
				continue
			}
			if rest := line[bar+len("│"):]; rest != "" && !strings.HasPrefix(rest, " ") {
				t.Errorf("hyper review %s renders %q, want one character between the bar and the line", named, line)
			}
		}
		if strings.Contains(stdout, "~") {
			t.Errorf("hyper review %s carries a change mark; nothing marks a line until the change column lands (issue #168):\n%s", named, stdout)
		}
	}
}

// TestRunReview_TheWireCarriesGutterRowsOnAllFourOtherArtefacts is §8's
// `gutter` row on the four rosters this ticket lands, on the same contract as a
// Procedure's: one row per rendered line with content, carrying the string the
// page renders with its alignment padding collapsed to single spaces, and never
// a decomposition of it (§8, ADR-0063).
func TestRunReview_TheWireCarriesGutterRowsOnAllFourOtherArtefacts(t *testing.T) {
	root := rosterRepo(t)

	for _, c := range []struct {
		named string
		want  []string
	}{
		{"definitions/things.yaml", []string{
			`{"type":"gutter","line":4,"marker":"mutate DESTROY"}`,
			`{"type":"gutter","line":5,"marker":"DESTROY end_thing"}`,
			`{"type":"gutter","line":6,"marker":"staging"}`,
		}},
		{"staging", []string{
			`{"type":"gutter","line":4,"marker":"read mutate DESTROY"}`,
			`{"type":"gutter","line":5,"marker":"http"}`,
			`{"type":"gutter","line":6,"marker":"api.things.dev api.things.eu"}`,
			`{"type":"gutter","line":8,"marker":"THINGS_API_TOKEN"}`,
		}},
		{"providers/things.yaml", []string{
			`{"type":"gutter","line":5,"marker":"http"}`,
			`{"type":"gutter","line":6,"marker":"header"}`,
			`{"type":"gutter","line":9,"marker":"read repeatable"}`,
			`{"type":"gutter","line":18,"marker":"mutate run-once"}`,
			`{"type":"gutter","line":27,"marker":"DESTROY repeatable"}`,
		}},
		{"hyper.yaml", []string{
			`{"type":"gutter","line":2,"marker":"1.4.0"}`,
			`{"type":"gutter","line":4,"marker":"90d"}`,
		}},
	} {
		stdout, _, exit := runReview(t, root, c.named, "--json")
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s --json exited %d, want a clean review", c.named, exit)
		}

		var rows []string
		for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
			if strings.Contains(line, `"type":"gutter"`) {
				rows = append(rows, line)
			}
		}
		if !slices.Equal(rows, c.want) {
			t.Errorf("hyper review %s --json carries\n %q\nwant\n %q", c.named, rows, c.want)
		}
	}
}

// TestRunReview_UnresolvedRendersOnNoArtefactButAProcedure is the other side of
// the supply rule: three of the five artefacts name no artefact at all and a
// Definition names one that reaches nothing on this screen, so `unresolved` —
// §8's one mark for a name the gutter must follow that resolves to nothing —
// has nothing to fire on but a Step (§8, ADR-0064).
func TestRunReview_UnresolvedRendersOnNoArtefactButAProcedure(t *testing.T) {
	root := newRepo(t)
	// Every name one of these four artefacts writes for another, pointed at
	// nothing: a Definition's Provider and its Targets, a Target
	// declaration's class, a Procedure's own targets:.
	writeFile(t, root+"/definitions/orphan.yaml",
		"kind: definition\ndefinition: orphan\nprovider: not-there\nkinds: [read]\ntargets: [nowhere]\n")
	writeFile(t, root+"/targets/lonely.yaml",
		"kind: target-declaration\ntarget: lonely\nclass: not-there\nkinds: [read]\ncapabilities: [shell]\n")
	writeFile(t, root+"/providers/things.yaml", providerDeclaring)

	for _, named := range []string{"orphan", "lonely", "things", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named)
		if exit != cli.ExitClean {
			t.Fatalf("hyper review %s exited %d, want a clean review", named, exit)
		}
		for line, marker := range markersOf(t, stdout) {
			if strings.Contains(marker, "unresolved") {
				t.Errorf("hyper review %s marks line %d %q; unresolved is a Procedure's mark and no other artefact's", named, line, marker)
			}
		}
	}
}

// authorityOf is the AUTHORITY block read back off the page: every line from
// its caption to the blank line beneath it, each with the screen's own indent
// taken off, and nothing at all where the block did not render.
//
// It finds the block by its caption rather than by counting lines from the
// bottom, which is what lets a case assert that the block is *absent* — the
// thing §8 distinguishes from an empty one (ADR-0069). It ends at the blank
// line for the same reason: the `FLAGS` block stands beneath this one on every
// artefact, and a reading that ran to the end of the page would make a case
// about the table a case about both.
func authorityOf(page string) []string {
	lines := strings.Split(strings.TrimSuffix(page, "\n"), "\n")
	for n, line := range lines {
		if !strings.HasPrefix(strings.TrimLeft(line, " "), "AUTHORITY") {
			continue
		}
		block := make([]string, 0, len(lines)-n)
		for _, rest := range lines[n:] {
			if rest == "" {
				break
			}
			block = append(block, strings.TrimPrefix(rest, "  "))
		}
		return block
	}
	return nil
}

// authorityRepo is a repository with both ends of the relation in it: one
// Target two Definitions claim, one Target nothing claims, and a Procedure
// binding one of the pairs twice.
func authorityRepo(t *testing.T) string {
	t.Helper()
	root := newRepo(t)
	writeFile(t, root+"/providers/things.yaml", providerDeclaring)
	writeFile(t, root+"/targets/staging.yaml",
		"kind: target-declaration\ntarget: staging\nclass: things\nkinds: [read, mutate, destroy]\ncapabilities: [http]\nhosts: [api.things.dev]\n")
	writeFile(t, root+"/targets/unclaimed.yaml",
		"kind: target-declaration\ntarget: unclaimed\nclass: things\nkinds: [read, destroy]\ncapabilities: [http]\nhosts: [api.things.dev]\n")
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate]\ndestroy: [end_thing]\ntargets: [staging]\n")
	writeFile(t, root+"/definitions/things-observed.yaml",
		"kind: definition\ndefinition: things-observed\nprovider: things\nkinds: [read]\ntargets: [staging]\n")
	return root
}

// TestRunReview_ADefinitionRendersARowPerTargetItClaims is the left end of the
// relation, and the whole of what one row states: the claimed Kinds with
// `destroy` derived at that column, the accepted Kinds, their intersection as
// initials, and the `destroy` Operations named (§5, §8).
func TestRunReview_ADefinitionRendersARowPerTargetItClaims(t *testing.T) {
	root := authorityRepo(t)

	stdout, stderr, exit := runReview(t, root, "definitions/things.yaml")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"DEFINITION  TARGET   DEFINITION KINDS  TARGET KINDS         EFFECTIVE  DESTROY OPS",
		"things      staging  mutate destroy    read mutate destroy  m d        end_thing",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_ATargetDeclarationRendersARowPerDefinitionThatClaimsIt is the
// rendering an unaided reading withholds: the columns, their order and the
// caption do not move, and no column is elided because its value repeats
// (ADR-0069).
func TestRunReview_ATargetDeclarationRendersARowPerDefinitionThatClaimsIt(t *testing.T) {
	root := authorityRepo(t)

	stdout, stderr, exit := runReview(t, root, "staging")
	if exit != cli.ExitClean || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"DEFINITION       TARGET   DEFINITION KINDS  TARGET KINDS         EFFECTIVE  DESTROY OPS",
		"things           staging  mutate destroy    read mutate destroy  m d        end_thing",
		"things-observed  staging  read              read mutate destroy  r          —",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_ATargetDeclarationNothingClaimsRendersAnExplicitEmptyState is
// the line between absent and empty: an edit to any file in definitions/ puts a
// row here, so the block renders and says there is none — a granted `destroy`
// with no claimant being either a Target awaiting its Definition or one whose
// Definition was deleted (§8, ADR-0012, ADR-0069).
func TestRunReview_ATargetDeclarationNothingClaimsRendersAnExplicitEmptyState(t *testing.T) {
	root := authorityRepo(t)

	stdout, _, exit := runReview(t, root, "unclaimed")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"no Definition claims this Target",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_AManifestAndARepositoryDeclarationRenderNoTable is the absence
// that is ruled rather than discovered: neither artefact is a member of any
// pair, so no edit to any file produces a row and an empty block there would
// assert a supply that does not exist (§8, ADR-0069).
func TestRunReview_AManifestAndARepositoryDeclarationRenderNoTable(t *testing.T) {
	root := authorityRepo(t)

	for _, named := range []string{"providers/things.yaml", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named)
		if exit != cli.ExitClean {
			t.Fatalf("exit = %d reviewing %s, want a clean review", exit, named)
		}
		if strings.Contains(stdout, "AUTHORITY") || strings.Contains(stdout, "claims this Target") {
			t.Errorf("the review of %s reads\n%s\nwant no table at all — no header, no empty body, no sentence", named, stdout)
		}
	}
}

// TestRunReview_AProcedureRendersOneRowPerDistinctPairItsStepsBind is the
// artefact that supplies neither end, and the ordering that is refused on it:
// two Steps sharing a pairing collapse to one row, and the rows sort by
// (Target, Definition) rather than following the marker column (§8, ADR-0069).
func TestRunReview_AProcedureRendersOneRowPerDistinctPairItsStepsBind(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/procedures/subject.yaml", `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: end
    definition: things
    operation: end_thing
    target: staging
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
  - id: end-again
    definition: things
    operation: end_thing
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"DEFINITION       TARGET   DEFINITION KINDS  TARGET KINDS         EFFECTIVE  DESTROY OPS",
		"things           staging  mutate destroy    read mutate destroy  m d        end_thing",
		"things-observed  staging  read              read mutate destroy  r          —",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q — sorted, and one row per distinct pairing", got, want)
	}
}

// TestRunReview_AnAbsentTargetDeclarationEmptiesTwoCellsNotTheRow is §8's own
// rule: dropping the row would say the third Target was never claimed, and the
// claimed Kinds and the named `destroy` Operations are this artefact's own
// (ADR-0026).
func TestRunReview_AnAbsentTargetDeclarationEmptiesTwoCellsNotTheRow(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate]\ndestroy: [end_thing]\ntargets: [staging, nowhere]\n")

	stdout, _, exit := runReview(t, root, "definitions/things.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"DEFINITION  TARGET   DEFINITION KINDS  TARGET KINDS         EFFECTIVE   DESTROY OPS",
		"things      nowhere  mutate destroy    unresolved           unresolved  end_thing",
		"things      staging  mutate destroy    read mutate destroy  m d         end_thing",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_ATargetDeclarationThatWillNotParseCarriesTheSameWord is the
// other absence: a supply that is present and will not parse has no accepted
// Kinds to render, and the two differ in nothing this table can act on
// (ADR-0064, ADR-0069).
func TestRunReview_ATargetDeclarationThatWillNotParseCarriesTheSameWord(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/targets/staging.yaml", "kind: target-declaration\ntarget: staging\n  broken: [")

	stdout, _, exit := runReview(t, root, "definitions/things.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review — the fault is in a file the reviewer did not ask about", exit)
	}
	if got := authorityOf(stdout); len(got) != 3 || !strings.Contains(got[2], "unresolved") {
		t.Errorf("the block reads\n%q\nwant one row carrying unresolved", got)
	}
}

// TestRunReview_ADiscoveryFailureIsCountedBeneathTheTable is the one absence
// with no cell to carry it: a Definition claiming this Target that did not
// parse contributes nothing at all, so the table says how many and where to
// take it — and the review still exits 0, exit 1 being for the artefact under
// review (§8, §9, ADR-0069).
func TestRunReview_ADiscoveryFailureIsCountedBeneathTheTable(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/definitions/half-written.yaml", "kind: definition\ndefinition: half\n  targets: [")

	stdout, _, exit := runReview(t, root, "staging")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	block := authorityOf(stdout)
	if len(block) == 0 {
		t.Fatalf("the review carries no AUTHORITY block:\n%s", stdout)
	}
	last := block[len(block)-1]
	if want := "1 definition did not load · hyper check"; last != want {
		t.Errorf("the block ends %q, want %q", last, want)
	}
	if strings.Contains(last, "half") {
		t.Errorf("the line reads %q; it names no member", last)
	}
}

// TestRunReview_TheDiscoveryCountRendersOnlyWhereItIsNonZero is the other half
// of that line: a repository whose definitions/ all loaded says nothing, on the
// shape §8 gives the `git diff` row below it.
func TestRunReview_TheDiscoveryCountRendersOnlyWhereItIsNonZero(t *testing.T) {
	root := authorityRepo(t)

	stdout, _, exit := runReview(t, root, "staging")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if strings.Contains(stdout, "did not load") {
		t.Errorf("the review reads\n%s\nwant no discovery count — every definitions/ file loaded", stdout)
	}
}

// TestRunReview_TheWireCarriesOneAuthorityRowPerRenderedRow is §8's own row:
// the Kinds as arrays of their full names rather than the page's initials, and
// `destroy_operations` written empty rather than omitted where the Definition
// names none — the supply is there and names nothing (§7, §8).
func TestRunReview_TheWireCarriesOneAuthorityRowPerRenderedRow(t *testing.T) {
	root := authorityRepo(t)

	stdout, _, exit := runReview(t, root, "staging", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	for _, want := range []string{
		`{"type":"authority","definition":"things","target":"staging","definition_kinds":["mutate","destroy"],"target_kinds":["read","mutate","destroy"],"effective":["mutate","destroy"],"destroy_operations":["end_thing"]}`,
		`{"type":"authority","definition":"things-observed","target":"staging","definition_kinds":["read"],"target_kinds":["read","mutate","destroy"],"effective":["read"],"destroy_operations":[]}`,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the stream is\n%s\nwant a row carrying %s", stdout, want)
		}
	}
}

// TestRunReview_AnUnsuppliedCellIsAbsentFromTheWire is the ordinary absence
// rule where the page writes `unresolved`: a member with no supply is absent
// from the object entirely, where one that is supplied and empty is written
// empty — the two being the distinction the whole table is built on (§7, §8).
func TestRunReview_AnUnsuppliedCellIsAbsentFromTheWire(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate]\ndestroy: [end_thing]\ntargets: [nowhere]\n")

	stdout, _, exit := runReview(t, root, "definitions/things.yaml", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := `{"type":"authority","definition":"things","target":"nowhere","definition_kinds":["mutate","destroy"],"destroy_operations":["end_thing"]}`
	if !strings.Contains(stdout, want) {
		t.Errorf("the stream is\n%s\nwant a row carrying %s", stdout, want)
	}
}

// TestRunReview_NoAuthorityRowIsEmittedWhereTheTableIsAbsent is the wire half
// of the two artefacts that are members of no pair: a rendering that emits no
// rows emits no rows (ADR-0069).
func TestRunReview_NoAuthorityRowIsEmittedWhereTheTableIsAbsent(t *testing.T) {
	root := authorityRepo(t)

	for _, named := range []string{"providers/things.yaml", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named, "--json")
		if exit != cli.ExitClean {
			t.Fatalf("exit = %d reviewing %s, want a clean review", exit, named)
		}
		if strings.Contains(stdout, `"type":"authority"`) {
			t.Errorf("the stream for %s is\n%s\nwant no authority row", named, stdout)
		}
	}
}

// TestRunReview_TwoRunsAgainstOneRepositoryAreByteIdentical is what the sort is
// for: the row set on a Target declaration is discovered across a map, and a
// rendering that followed the walk would differ between two runs of one binary
// against one repository (§8).
func TestRunReview_TwoRunsAgainstOneRepositoryAreByteIdentical(t *testing.T) {
	root := authorityRepo(t)

	first, _, _ := runReview(t, root, "staging")
	for range 8 {
		if again, _, _ := runReview(t, root, "staging"); again != first {
			t.Fatalf("two renderings of one review differ:\n%s\n---\n%s", first, again)
		}
	}
}

// TestRunReview_ADefinitionClaimingNoTargetSaysSoInItsOwnWords is the empty
// state on the end of the relation it was not written for: an edit to *this*
// file puts a row here, so the block renders — and what it says is true of a
// Definition. The Target declaration's sentence would state the wrong fact on
// this screen (§8, ADR-0069).
func TestRunReview_ADefinitionClaimingNoTargetSaysSoInItsOwnWords(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/definitions/unclaiming.yaml",
		"kind: definition\ndefinition: unclaiming\nprovider: things\nkinds: [read]\ntargets: []\n")

	stdout, _, exit := runReview(t, root, "definitions/unclaiming.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"this Definition claims no Target",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_AProcedureBindingNoPairSaysSoInItsOwnWords is the third
// sentence, on the artefact that supplies neither end: a Procedure whose only
// Step is a nested invocation binds no pairing of its own, and an edit to its
// steps: puts a row here (§3, §8).
func TestRunReview_AProcedureBindingNoPairSaysSoInItsOwnWords(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/procedures/inner.yaml", "kind: procedure\nprocedure: inner\ntargets: []\nsteps: []\n")
	writeFile(t, root+"/procedures/subject.yaml", `kind: procedure
procedure: subject
targets: []
steps:
  - id: nested
    procedure: inner
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"AUTHORITY   assembled from definitions/ and targets/",
		"no Step binds a Definition to a Target",
	}
	if got := authorityOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_ADefinitionsKindsRenderAsWritten is ReadDefinitionFacts's rule
// held at the column: the claim is not reduced or re-sorted, and the one thing
// guarded is the derived member — a Definition that writes `destroy` in kinds:
// and names a `destroy` Operation states the claim once (§3, §8).
func TestRunReview_ADefinitionsKindsRenderAsWritten(t *testing.T) {
	root := authorityRepo(t)
	writeFile(t, root+"/definitions/loud.yaml",
		"kind: definition\ndefinition: loud\nprovider: things\nkinds: [destroy, read]\ndestroy: [end_thing]\ntargets: [staging]\n")

	stdout, _, exit := runReview(t, root, "definitions/loud.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	block := authorityOf(stdout)
	if len(block) != 3 || !strings.Contains(block[2], "destroy read ") {
		t.Errorf("the block reads\n%q\nwant the kinds: list as written, with no second destroy appended", block)
	}
}
