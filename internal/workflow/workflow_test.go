package workflow_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// workedExample is §10's own inputs for the file testdata/ holds: the
// Procedure the section projects, its Cadence, a Procedure that effects, the
// one credential slot its bindings require, and the version and digest the
// example pins. The digest is spelled as `hyper.yaml` spells it, algorithm
// inline (§3), which is where `project` reads it from.
var workedExample = workflow.Facts{
	Procedure: "retire-preview-envs",
	Cadence:   "0 3 * * 1",
	Effects:   true,
	Variables: []string{"STAGING_TOKEN"},
	Version:   "0.4.1",
	Digest:    "sha256:a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2",
}

// golden is testdata/'s copy of the section's fenced block — the independent
// source of truth this package is held to, read rather than restated.
func golden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading the worked example: %v", err)
	}
	return string(data)
}

// TestGenerate_TheWorkedExampleIsReproducedByteForByte is the whole claim in
// one assertion. §10 states the file and this package writes it; a byte that
// differs is a `projection-stale` on every repository at once, so the
// comparison is the file and never a substring of it.
func TestGenerate_TheWorkedExampleIsReproducedByteForByte(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	want := golden(t, "retire-preview-envs.yml")
	if got != want {
		t.Errorf("Generate() wrote\n%s\nwant\n%s", got, want)
	}
}

// TestGenerate_TheSameFactsWriteTheSameBytes is what generate-and-verify rests
// on: the check regenerates and compares, so a generator that answered twice
// would fail a repository nobody had edited.
func TestGenerate_TheSameFactsWriteTheSameBytes(t *testing.T) {
	first := string(workflow.Generate(workedExample))
	second := string(workflow.Generate(workedExample))
	if first != second {
		t.Error("Generate() answered two different files for one set of facts")
	}
}

// readOnly is the worked example's own facts with the one member the
// concurrency block turns on inverted — two files differing in one fact
// rather than two unrelated repositories.
func readOnly() workflow.Facts {
	facts := workedExample
	facts.Effects = false
	return facts
}

func TestGenerate_AReadOnlyProcedureCarriesNoConcurrencyBlock(t *testing.T) {
	got := string(workflow.Generate(readOnly()))
	if strings.Contains(got, "concurrency:") {
		t.Errorf("a read-only Procedure carried a concurrency block:\n%s", got)
	}
	if strings.Contains(got, "hyper-store") {
		t.Errorf("a read-only Procedure named the Store's group:\n%s", got)
	}
}

func TestGenerate_AnEffectfulProcedureTakesTheStoresGroup(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	want := "concurrency:\n  group: hyper-store\n  cancel-in-progress: false\n"
	if !strings.Contains(got, want) {
		t.Errorf("Generate() wrote\n%s\nwant it to carry\n%s", got, want)
	}
}

// TestGenerate_EveryVariantIsAFileThatParses guards the one way an absent
// block can go wrong that a substring test cannot see: a blank line left where
// the block was, or one eaten from around it.
func TestGenerate_EveryVariantIsAFileThatParses(t *testing.T) {
	for name, facts := range map[string]workflow.Facts{
		"the worked example": workedExample,
		"read-only":          readOnly(),
		"no credential slot": noSlots(),
		"two slots":          twoSlots(),
	} {
		var read map[string]any
		if err := yaml.Unmarshal(workflow.Generate(facts), &read); err != nil {
			t.Errorf("%s: the generated file does not parse: %v", name, err)
		}
	}
}

func noSlots() workflow.Facts {
	facts := workedExample
	facts.Variables = nil
	return facts
}

func twoSlots() workflow.Facts {
	facts := workedExample
	facts.Variables = []string{"STAGING_TOKEN", "CLOUDFLARE_TOKEN"}
	return facts
}

// TestGenerate_AProcedureRequiringNoSlotCarriesNoEnvKey holds the absence rule
// §10 states: an absent block is *this Procedure requires nothing*, where an
// empty mapping asserts a lookup that happened and found nothing.
func TestGenerate_AProcedureRequiringNoSlotCarriesNoEnvKey(t *testing.T) {
	got := string(workflow.Generate(noSlots()))
	if strings.Contains(got, "env:") {
		t.Errorf("a Procedure requiring no slot carried an env: key:\n%s", got)
	}
	if strings.Contains(got, "secrets.") {
		t.Errorf("a Procedure requiring no slot named an executor secret:\n%s", got)
	}
}

// TestGenerate_EnvEntriesAreOrderedByVariableName holds the block to being a
// function of the repository rather than of the walk that found the pairs: the
// caller hands over what its Steps bind, in whatever order it walked them, and
// the file is the same file either way.
func TestGenerate_EnvEntriesAreOrderedByVariableName(t *testing.T) {
	want := "        env:\n" +
		"          CLOUDFLARE_TOKEN: ${{ secrets.CLOUDFLARE_TOKEN }}\n" +
		"          STAGING_TOKEN: ${{ secrets.STAGING_TOKEN }}\n"

	forward := twoSlots()
	backward := forward
	backward.Variables = []string{"STAGING_TOKEN", "CLOUDFLARE_TOKEN"}
	slices.Reverse(backward.Variables)

	for _, facts := range []workflow.Facts{forward, backward} {
		got := string(workflow.Generate(facts))
		if !strings.Contains(got, want) {
			t.Errorf("Generate() wrote\n%s\nwant it to carry\n%s", got, want)
		}
	}
}

// TestGenerate_OneVariableNamedTwiceIsOneEntry is what two Definitions binding
// one Target under one scheme come to: one slot, one variable, and a mapping
// that may not carry a key twice.
func TestGenerate_OneVariableNamedTwiceIsOneEntry(t *testing.T) {
	facts := workedExample
	facts.Variables = []string{"STAGING_TOKEN", "STAGING_TOKEN"}

	got := string(workflow.Generate(facts))
	if n := strings.Count(got, "STAGING_TOKEN: "); n != 1 {
		t.Errorf("STAGING_TOKEN was written %d times, want 1:\n%s", n, got)
	}
}

// TestGenerate_TheEnvBlockSitsOnTheRunStepAndNowhereElse holds §10's placement:
// the credentials the Steps need are on the invocation that runs them, and the
// Comparison that follows reads the repository and needs none.
func TestGenerate_TheEnvBlockSitsOnTheRunStepAndNowhereElse(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	if n := strings.Count(got, "env:"); n != 1 {
		t.Errorf("env: appears %d times, want 1:\n%s", n, got)
	}
	run := strings.Index(got, "- name: hyper run ")
	changes := strings.Index(got, "- name: hyper changes ")
	env := strings.Index(got, "env:")
	if !(run < env && env < changes) {
		t.Errorf("env: is not on the run step:\n%s", got)
	}
}

// TestGenerate_TheCronExpressionIsAlwaysSingleQuoted holds the one scalar whose
// quoting is fixed rather than derived, as §10's example fixes it — including
// the expressions a plain scalar would carry perfectly well, which is what
// makes the rule a rule.
func TestGenerate_TheCronExpressionIsAlwaysSingleQuoted(t *testing.T) {
	for _, expression := range []string{"0 3 * * 1", "*/15 * * * *", "30 4 * * *", "0 0 1 1 0"} {
		facts := workedExample
		facts.Cadence = expression

		want := "    - cron: '" + expression + "'\n"
		if got := string(workflow.Generate(facts)); !strings.Contains(got, want) {
			t.Errorf("Generate() wrote\n%s\nwant it to carry\n%s", got, want)
		}
	}
}

// TestGenerate_OnCarriesTheRecurrenceAndNothingElse: the reviewed artefact
// declares a recurrence and no second occasion for a Run to start.
func TestGenerate_OnCarriesTheRecurrenceAndNothingElse(t *testing.T) {
	var read struct {
		On struct {
			Schedule []struct {
				Cron string `yaml:"cron"`
			} `yaml:"schedule"`
		} `yaml:"on"`
	}
	if err := yaml.Unmarshal(workflow.Generate(workedExample), &read); err != nil {
		t.Fatalf("the generated file does not parse: %v", err)
	}
	if len(read.On.Schedule) != 1 || read.On.Schedule[0].Cron != workedExample.Cadence {
		t.Errorf("on: = %+v, want the one recurrence %q", read.On, workedExample.Cadence)
	}

	got := string(workflow.Generate(workedExample))
	for _, absent := range []string{"workflow_dispatch", "push:", "pull_request", "repository_dispatch"} {
		if strings.Contains(got, absent) {
			t.Errorf("on: carried %s, which no artefact declared:\n%s", absent, got)
		}
	}
}

// awkwardNames are the names the quoting rule exists for: each is a plain
// scalar some YAML reader resolves to something that is not the string, and
// each is a name a Procedure may carry, `check` holding a name to matching its
// filename and to nothing else.
//
// The want beside each is the whole `name:` line, byte for byte, because the
// rule is about bytes: what a second reader does with them is the point, and
// this is the only place in the corpus that states which bytes are written.
var awkwardNames = []struct {
	procedure string
	want      string
}{
	{"retire-preview-envs", "name: retire-preview-envs\n"},
	{"on", "name: 'on'\n"},
	{"true", "name: 'true'\n"},
	{"null", "name: 'null'\n"},
	{"12:30", "name: '12:30'\n"},
	{"0x10", "name: '0x10'\n"},
	{"no", "name: 'no'\n"},
	{"off", "name: 'off'\n"},
	{"yes", "name: 'yes'\n"},
	{"~", "name: '~'\n"},
	{"2024-01-01", "name: '2024-01-01'\n"},
	{"0777", "name: '0777'\n"},
	{"1.5", "name: '1.5'\n"},
	{"-lead", "name: '-lead'\n"},
	{"it's", "name: it's\n"},
	{"-it's", "name: '-it''s'\n"},
	{"", "name: ''\n"},
}

// TestGenerate_TheQuotingRuleIsTotal is the rule in its two halves at once:
// the bytes written, and that they parse back as the name that was written.
func TestGenerate_TheQuotingRuleIsTotal(t *testing.T) {
	for _, name := range awkwardNames {
		facts := workedExample
		facts.Procedure = name.procedure

		got := string(workflow.Generate(facts))
		if !strings.Contains(got, name.want) {
			t.Errorf("a Procedure named %q wrote\n%s\nwant it to carry %q", name.procedure, got, name.want)
		}

		var read struct {
			Name string `yaml:"name"`
		}
		if err := yaml.Unmarshal([]byte(got), &read); err != nil {
			t.Errorf("a Procedure named %q wrote a file that does not parse: %v", name.procedure, err)
			continue
		}
		if read.Name != name.procedure {
			t.Errorf("name: parsed back as %q, want %q", read.Name, name.procedure)
		}
	}
}

// TestGenerate_TheVersionAppearsInTheFourPlacesTheSectionCounts holds §11's
// count: the header comment, the install step's own name, the release tag and
// the artefact's filename. A version in three of them is a file that installs
// one binary and says it installed another.
func TestGenerate_TheVersionAppearsInTheFourPlacesTheSectionCounts(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	for _, place := range []string{
		"# generated by hyper 0.4.1 — edits are overwritten by `hyper project`\n",
		"      - name: install hyper 0.4.1\n",
		"/releases/download/v0.4.1/",
		"hyper-0.4.1-x86_64-linux.tar.gz",
	} {
		if !strings.Contains(got, place) {
			t.Errorf("Generate() wrote\n%s\nwant it to carry %q", got, place)
		}
	}
	if n := strings.Count(got, "0.4.1"); n != 4 {
		t.Errorf("the version appears %d times, want 4", n)
	}
}

// TestGenerate_TheDigestIsWrittenAsTheChecksumLineTakesIt: `hyper.yaml` carries
// the algorithm inline so a reviewer reads which one produced the digest, and
// `sha256sum -c -` takes the hex alone. Stripping it here is one rule in one
// place rather than a split every caller repeats.
func TestGenerate_TheDigestIsWrittenAsTheChecksumLineTakesIt(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	want := "          echo 'a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2  hyper.tar.gz' \\\n" +
		"            | sha256sum -c -\n"
	if !strings.Contains(got, want) {
		t.Errorf("Generate() wrote\n%s\nwant it to carry\n%s", got, want)
	}
	if strings.Contains(got, "sha256:") {
		t.Errorf("the algorithm prefix reached the checksum line:\n%s", got)
	}
}

// TestGenerate_ADigestCarryingNoAlgorithmIsWrittenWhole: the pin's schema makes
// the digest a string and nothing narrower, and this package judges nothing —
// what it is handed is what it writes.
func TestGenerate_ADigestCarryingNoAlgorithmIsWrittenWhole(t *testing.T) {
	facts := workedExample
	facts.Digest = "deadbeef"

	if got := string(workflow.Generate(facts)); !strings.Contains(got, "echo 'deadbeef  hyper.tar.gz'") {
		t.Errorf("Generate() wrote\n%s\nwant it to carry the digest whole", got)
	}
}

// TestGenerate_TheJobSummaryIsShellAndTheBinaryIsToldNothing holds the line the
// safety model draws: `$GITHUB_STEP_SUMMARY`, the fences, the `tee` and
// `${PIPESTATUS[0]}` are bytes in this file, and the invocations they wrap are
// the same invocations a laptop makes.
func TestGenerate_TheJobSummaryIsShellAndTheBinaryIsToldNothing(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	for _, want := range []string{
		"          ./hyper run retire-preview-envs | tee -a \"$GITHUB_STEP_SUMMARY\"\n",
		"          code=${PIPESTATUS[0]}\n",
		"          exit $code\n",
		"        if: always()\n",
		"          ./hyper changes retire-preview-envs | tee -a \"$GITHUB_STEP_SUMMARY\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Generate() wrote\n%s\nwant it to carry %q", got, want)
		}
	}
	for _, absent := range []string{"--json", "--markdown", "GITHUB_ACTIONS", "if: success()", "if: failure()"} {
		if strings.Contains(got, absent) {
			t.Errorf("the file carried %q — the executor axis the safety model deleted:\n%s", absent, got)
		}
	}
}

// TestGenerate_TheDeepenStepIsGuardedAndCarriesNoOrTrue: `--unshallow` errors on
// a repository that is already complete, and the guard is what keeps the step
// total while leaving a real failure fatal.
func TestGenerate_TheDeepenStepIsGuardedAndCarriesNoOrTrue(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	if !strings.Contains(got, "          if [ -f .git/shallow ]; then git fetch --unshallow; fi\n") {
		t.Errorf("Generate() wrote\n%s\nwant the guarded deepen step", got)
	}
	if strings.Contains(got, "|| true") {
		t.Errorf("the deepen step carried || true:\n%s", got)
	}
	if strings.Contains(got, "fetch-depth") {
		t.Errorf("the checkout named a depth, which would fetch the Store branch:\n%s", got)
	}
	if deepen, run := strings.Index(got, "deepen the checkout"), strings.Index(got, "./hyper run "); deepen > run {
		t.Errorf("the deepen step sits after the run invocation:\n%s", got)
	}
}

// TestGenerate_ThePermissionsAreTheWholeOfWhatWritingTheStoreNeeds.
func TestGenerate_ThePermissionsAreTheWholeOfWhatWritingTheStoreNeeds(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	if !strings.Contains(got, "permissions:\n  contents: write\n") {
		t.Errorf("Generate() wrote\n%s\nwant permissions: contents: write", got)
	}
	if strings.Contains(got, "persist-credentials: false") || !strings.Contains(got, "persist-credentials: true") {
		t.Errorf("persist-credentials is not written out as true:\n%s", got)
	}
}

// TestConstants_TheFourAreCompiledInAndTheVersionIsTheirOnlyVariable holds §11's
// closed set: the runner and the checkout are bytes in the file above, and the
// two URLs are the package's own with the version their one variable — the
// platform being the same compiled-in fact `runs-on` is.
func TestConstants_TheVersionIsTheOnlyVariableInEitherURL(t *testing.T) {
	if got, want := workflow.ArtefactURL("1.2.3"), "https://github.com/TheLoomLabs/hyper/releases/download/v1.2.3/hyper-1.2.3-x86_64-linux.tar.gz"; got != want {
		t.Errorf("ArtefactURL() = %q, want %q", got, want)
	}
	if got, want := workflow.ArtefactName("1.2.3"), "hyper-1.2.3-x86_64-linux.tar.gz"; got != want {
		t.Errorf("ArtefactName() = %q, want %q", got, want)
	}
	if got, want := workflow.ChecksumsURL("1.2.3"), "https://github.com/TheLoomLabs/hyper/releases/download/v1.2.3/checksums.txt"; got != want {
		t.Errorf("ChecksumsURL() = %q, want %q", got, want)
	}
}

func TestGenerate_TheRunnerAndTheCheckoutAreTheOnesTheSectionNames(t *testing.T) {
	got := string(workflow.Generate(workedExample))
	for _, want := range []string{
		"    runs-on: ubuntu-24.04\n",
		"      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Generate() wrote\n%s\nwant it to carry %q", got, want)
		}
	}
}

// permittedImports is what a package that opens no file, reaches no network,
// starts no subprocess and reads no clock may import. The list is the test:
// purity is a property of what this package can reach, and a reviewer reading
// an import block is reading the whole of it.
var permittedImports = []string{"fmt", "regexp", "sort", "strings"}

// TestPackage_OpensNoFileReachesNoNetworkStartsNoSubprocessAndReadsNoClock is
// the structural half of *facts in, exact bytes out*. It is what lets the
// generator and the projection check be one function called from two places:
// a generator that could read a clock would answer differently on the second
// call, and generate-and-verify would be comparing a file against a guess.
func TestPackage_OpensNoFileReachesNoNetworkStartsNoSubprocessAndReadsNoClock(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(f os.FileInfo) bool {
		return !strings.HasSuffix(f.Name(), "_test.go")
	}, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("reading the package's own imports: %v", err)
	}

	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, imported := range file.Imports {
				path := strings.Trim(imported.Path.Value, `"`)
				if !slices.Contains(permittedImports, path) {
					t.Errorf("%s imports %q, which is not one of %v", name, path, permittedImports)
				}
			}
		}
	}
}
