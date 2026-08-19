package cli_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// runReview drives `hyper review` against root with the arguments given, in an
// environment with nothing in it: the repository is named by the flag, which is
// what every case here means by "against this repository".
func runReview(t *testing.T, root string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	var out, errs bytes.Buffer
	exit = cli.RunReview(append([]string{"--repo-dir", root}, args...), &out, &errs, emptyEnvironment, root, "1.4.0")
	return out.String(), errs.String(), exit
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

	first := strings.SplitN(stdout, "\n", 2)[0]
	if got, want := first, "  MANIFEST  │  no baseline — shell ships in the binary"; got != want {
		t.Errorf("the header's first line is\n %q\nwant\n %q", got, want)
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

// TestRunReview_TheStreamIsTheHeaderRowAndTheTerminalRow is what `--json`
// carries in this milestone: nothing is annotated yet, so the annotations are
// the header alone, and the terminal row's marker is false because nothing on
// this screen is a result set (§8, §9).
func TestRunReview_TheStreamIsTheHeaderRowAndTheTerminalRow(t *testing.T) {
	root := newRepo(t)

	stdout, _, exit := runReview(t, root, "hyper.yaml", "--json")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	want := []string{
		`{"type":"artefact","kind":"repository-declaration","path":"hyper.yaml","baseline_absent":"no-store"}`,
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
	if exit := cli.RunReview([]string{"--repo-dir", root, "hyper.yaml"}, &named, &named, emptyEnvironment, root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("--repo-dir exited %d, want a clean review", exit)
	}
	if exit := cli.RunReview([]string{"hyper.yaml"}, &environed, &environed, fromEnv, elsewhere, "1.4.0"); exit != cli.ExitClean {
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
	if exit := cli.RunReview([]string{"--repo-dir", root, "--no-color", "hyper.yaml"}, &flagged, &flagged, emptyEnvironment, root, "1.4.0"); exit != cli.ExitClean {
		t.Fatalf("--no-color exited %d, want a clean review", exit)
	}
	if exit := cli.RunReview([]string{"--repo-dir", root, "hyper.yaml"}, &environedColour, &environedColour, noColorSet, root, "1.4.0"); exit != cli.ExitClean {
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
	if exit := cli.RunReview([]string{"--repo-dir", root, "cloudflare-prod"}, &out, &out, watching, root, "1.4.0"); exit != cli.ExitClean {
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
		if header[1] != worked.gloss {
			t.Errorf("%s glossed\n %q\nwant\n %q", worked.cadence, header[1], worked.gloss)
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
		`{"type":"result","truncated":false}`,
	}
	if got := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); !slices.Equal(got, want) {
		t.Errorf("the stream is\n %q\nwant\n %q", got, want)
	}
	if strings.Contains(stdout, "last_run") {
		t.Errorf("the row carries last_run:\n%s", stdout)
	}
}

// TestRunReview_AnUnreadableCadenceRendersNoGloss holds what this milestone
// does not do: `cadence-malformed` is §12's static check and is not implemented
// here, so an expression outside §10's grammar is not refused. The review
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
