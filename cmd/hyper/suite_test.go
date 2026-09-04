package main

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// What runs the suite, and what these cases are for (issue #243).
//
// Until this file there was one workflow in this repository, a tag was its
// whole trigger, and it published. Nothing ran `go test` but a person at a
// terminal, and `docs/build/releasing.md`'s first act — *land everything, on
// `main`, with the tests green* — was an instruction to that person and an
// assertion nothing checked. A tag pushed against a red tree built,
// version-checked and published, and the one gate in front of the one thing
// here that reaches the world compared the label on the tin: that the binary
// reports its tag.
//
// The cases below read `.github/workflows/` the way
// `TestRelease_TheTagRunsTheScriptTheseCasesRun` reads it — **the steps and
// never the file**, a workflow's own comments being the one place every string
// these cases look for could appear while the job did none of it. What they
// hold is the four properties the workflows cannot assert about themselves:
// that the suite runs on the two events a change arrives by, that the toolchain
// is go.mod's directive rather than an action's opinion, that every action is
// pinned by commit, and that a tag cannot publish a tree the suite fails on.
//
// The fifth property is not a question about the file. A runner with no
// `bubblewrap` installed makes `TestAcceptance_…` **skip in silence**, and the
// seam that fences every acceptance task is then the one thing CI does not run
// — issue #222's rot arriving through a new door, un-fencing every task in
// `scripts/acceptance/tasks/` at once. So the workflow installs the tool and claims, in `prepared`, that it
// did; `unavailable` below is what turns that claim into a red job rather than
// a green one nobody reads.

// prepared is the environment variable a machine sets to say it was prepared to
// run every case in this package: the tools are installed and a namespace can
// be built, so a case that says it cannot run here is a defect in the
// preparation rather than a property of the machine to be tolerated.
//
// It is set by `.github/workflows/suite.yml` and by nothing else. A laptop
// leaves it unset and keeps the skips, which is the honest answer there — a
// suite that failed on a machine without `bwrap` would be asserting something
// about the machine.
const prepared = "SUITE_PREPARED"

// machine is the part of *testing.T the gates that cannot run a case use. It
// exists so that the rule below is exercised by a case rather than only by the
// job that depends on it: *testing.T satisfies it, and so does a recorder that
// says which of the two was called.
type machine interface {
	Helper()
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// unavailable is what a case says where the machine is missing something a
// preparation could have supplied. It is the one place that decision is made,
// so that `needTools` and `needSeal` cannot drift into two answers to *what is
// a machine allowed not to have*.
//
// Where `prepared` is set the answer is that it is allowed nothing: the case
// fails, naming the variable, because something that promised to install the
// tool did not.
//
// **A machine's architecture is not one of these.** `hostPlatform` skips
// through `testing` directly, because the release publishing nothing for this
// GOOS/GOARCH is a fact no preparation could change, and a rule that failed on
// it would be asserting about the machine — the thing the skips exist to
// avoid.
func unavailable(t machine, format string, args ...any) {
	t.Helper()
	if os.Getenv(prepared) != "" {
		t.Fatalf("%s: %s is set, so this machine was prepared to run every case and a skip here is the preparation's defect", fmt.Sprintf(format, args...), prepared)
		return
	}
	t.Skipf(format, args...)
}

// TestUnavailable_APreparedMachineFailsWhereAnotherSkips holds the rule the CI
// job's fifth acceptance criterion rests on. Nothing else in the suite can
// observe it: a case that actually skipped would report itself as skipped, and
// a green run is exactly what a silent skip looks like.
func TestUnavailable_APreparedMachineFailsWhereAnotherSkips(t *testing.T) {
	t.Setenv(prepared, "")
	unprepared := &recorder{}
	unavailable(unprepared, "bwrap is not here")
	if unprepared.failed != "" || !strings.Contains(unprepared.skipped, "bwrap is not here") {
		t.Errorf("with %s unset the case skipped %q and failed %q; want the skip alone", prepared, unprepared.skipped, unprepared.failed)
	}

	t.Setenv(prepared, "1")
	claimed := &recorder{}
	unavailable(claimed, "bwrap is not here")
	if claimed.skipped != "" || !strings.Contains(claimed.failed, "bwrap is not here") || !strings.Contains(claimed.failed, prepared) {
		t.Errorf("with %s set the case skipped %q and failed %q; want a failure naming the variable and the reason", prepared, claimed.skipped, claimed.failed)
	}
}

// recorder answers which of the two a gate reached. The real methods do not
// return — they end the case — so nothing here runs after a call in a real run,
// and the recorder's returning is what lets one case exercise both branches.
type recorder struct{ skipped, failed string }

func (r *recorder) Helper()                        {}
func (r *recorder) Skipf(format string, a ...any)  { r.skipped = fmt.Sprintf(format, a...) }
func (r *recorder) Fatalf(format string, a ...any) { r.failed = fmt.Sprintf(format, a...) }

// TestSuite_TheWorkflowRunsWhatAContributorRuns is the first criterion: the
// three commands CONTRIBUTING opens with, on the two events by which a change
// reaches `main`, plus the `workflow_call` that lets the release wait on this
// same file rather than on a second transcription of it.
func TestSuite_TheWorkflowRunsWhatAContributorRuns(t *testing.T) {
	suite := workflowOf(t, "suite.yml")

	for _, trigger := range []string{"push", "pull_request", "workflow_call"} {
		if _, fires := suite.On[trigger]; !fires {
			t.Errorf("the suite does not fire on %s; it fires on %v", trigger, slices.Sorted(maps.Keys(suite.On)))
		}
	}
	var push struct {
		Branches []string `yaml:"branches"`
	}
	if on, fires := suite.On["push"]; fires {
		if err := on.Decode(&push); err != nil {
			t.Fatalf("the suite's push trigger does not parse: %v", err)
		}
		if !slices.Contains(push.Branches, "main") {
			t.Errorf("the suite's push trigger names %v, and `main` is the branch every change lands on", push.Branches)
		}
	}

	steps := runsOf(suite)
	for _, want := range []string{"go build ./...", "go vet ./...", "go test ./..."} {
		if !slices.ContainsFunc(steps, func(run string) bool { return strings.Contains(run, want) }) {
			t.Errorf("no step of the suite runs %q; its steps are %q", want, steps)
		}
	}
}

// TestSuite_TheToolchainIsGoModsDirectiveAndNoActionDecidesIt is release.yml's
// rule applied to the second workflow, for release.yml's reason: left alone the
// Go that builds and tests would be whichever the image ships wherever that
// satisfies the directive, which is an input to the answer that nothing in this
// repository decided.
//
// The two files carry the same line, and the case compares them rather than
// spelling the expression twice — a third copy here is a third thing to keep in
// agreement with go.mod.
func TestSuite_TheToolchainIsGoModsDirectiveAndNoActionDecidesIt(t *testing.T) {
	var pins []string
	for _, run := range runsOf(workflowOf(t, "release.yml")) {
		if strings.Contains(run, "GOTOOLCHAIN") {
			pins = append(pins, strings.TrimSpace(run))
		}
	}
	// Exactly one, because `runsOf` ranges over a map and the last of two
	// would be whichever the map handed over — a case that flakes rather than
	// one that fails. Two steps writing `GOTOOLCHAIN` is also two answers to
	// which Go builds the release, which is the fault this case is about.
	if len(pins) != 1 {
		t.Fatalf("the release workflow pins the toolchain in %d steps, want exactly 1: %q", len(pins), pins)
	}
	pin := pins[0]

	suite := runsOf(workflowOf(t, "suite.yml"))
	if !slices.ContainsFunc(suite, func(run string) bool { return strings.TrimSpace(run) == pin }) {
		t.Errorf("no step of the suite runs %q, which is how the release pins the toolchain; its steps are %q", pin, suite)
	}

	for name, parsed := range allWorkflows(t) {
		for _, uses := range usesOf(parsed) {
			if strings.Contains(uses, "setup-go") {
				t.Errorf("%s uses %s; the toolchain is go.mod's directive and no setup action decides which bytes build the binary", name, uses)
			}
		}
	}
}

// TestSuite_TheAcceptanceSeamRunsRatherThanSkipping is the one trap in this
// work. `TestAcceptance_TheSealedHarnessHandsAnAgentTheQuickstartAndNothingElse`
// opens with `needTools(t, "bash", "bwrap", …)` and `needSeal`, both correct,
// and on a runner without the tool they mean the fence that ranges over
// `tasks/*.md` does not run at all — every task un-fenced by a green job.
//
// So the runner installs `bubblewrap` and the test step claims `prepared`,
// which is what makes the gate above fail instead of skip.
func TestSuite_TheAcceptanceSeamRunsRatherThanSkipping(t *testing.T) {
	suite := workflowOf(t, "suite.yml")

	if !slices.ContainsFunc(runsOf(suite), func(run string) bool { return strings.Contains(run, "bubblewrap") }) {
		t.Errorf("no step of the suite installs bubblewrap; without it the acceptance seam skips and every acceptance task is fenced by nothing")
	}

	claimed := false
	for _, job := range suite.Jobs {
		for _, step := range job.Steps {
			if !strings.Contains(step.Run, "go test") {
				continue
			}
			if value, set := step.Env[prepared]; set && value.Value != "" {
				claimed = true
			}
		}
	}
	if !claimed {
		t.Errorf("the suite's `go test` step does not set %s; a skipped acceptance case would leave the job green", prepared)
	}
}

// TestWorkflows_EveryActionIsPinnedByCommit is release.yml's rule for
// release.yml's reason, held over every file in the directory rather than the
// one it was written in: a tag is a name somebody else can move, and an action
// resolved by tag is bytes this repository never reviewed running against a
// token that can publish.
//
// A reusable workflow of this repository's own is named by path, which is this
// commit's copy of it and is the point of naming it that way.
func TestWorkflows_EveryActionIsPinnedByCommit(t *testing.T) {
	commit := regexp.MustCompile(`^[^@]+@[0-9a-f]{40}$`)

	for name, parsed := range allWorkflows(t) {
		for _, uses := range usesOf(parsed) {
			if strings.HasPrefix(uses, "./") {
				continue
			}
			if !commit.MatchString(uses) {
				t.Errorf("%s uses %q, which is not pinned by commit", name, uses)
			}
		}
	}
}

// TestSuite_ATagCannotPublishATreeTheSuiteFailsOn is the last criterion, and
// the reason the suite is a callable workflow rather than only a triggered one.
// A job on `main` says nothing about the tree a tag names: tags are pushed at
// commits, `push` and `pull_request` do not fire for one, and a green `main` is
// not the tree being published anyway.
//
// So the release calls this same file and waits on it. What the case holds is
// the wait rather than the call — a gate that is not in `needs:` is a job that
// runs beside the publication instead of before it.
func TestSuite_ATagCannotPublishATreeTheSuiteFailsOn(t *testing.T) {
	release := workflowOf(t, "release.yml")

	var gates []string
	for name, job := range release.Jobs {
		if job.Uses == "./.github/workflows/suite.yml" {
			gates = append(gates, name)
		}
	}
	if len(gates) == 0 {
		t.Fatal("no job of the release workflow calls ./.github/workflows/suite.yml; a tag would publish a tree nothing tested")
	}

	published := false
	for name, job := range release.Jobs {
		if !slices.ContainsFunc(job.Steps, func(published step) bool { return strings.Contains(published.Run, "gh release create") }) {
			continue
		}
		published = true
		if !slices.ContainsFunc(gates, func(gate string) bool { return slices.Contains(job.Needs, gate) }) {
			t.Errorf("the release's %s job publishes and needs %v; it has to wait on one of %v", name, job.Needs, gates)
		}
	}
	if !published {
		t.Fatal("no job of the release workflow publishes; this case reads the publication as the thing that must wait")
	}
}

// workflowFile is the little of GitHub's format these cases read: the triggers, the
// jobs, and each job's steps. It is deliberately not the schema — what stands
// here is that certain commands run under certain triggers, and a struct that
// modelled the format would be a second and worse copy of somebody else's.
type workflowFile struct {
	On   map[string]yaml.Node `yaml:"on"`
	Jobs map[string]struct {
		Uses  string `yaml:"uses"`
		Needs names  `yaml:"needs"`
		Steps []step `yaml:"steps"`
	} `yaml:"jobs"`
}

// step is a step's three fields that say what it does: what it runs, what
// action it runs instead, and the environment it runs under. `Env` is read as
// nodes because a value there is whatever YAML the author wrote — `1` and `'1'`
// are one claim, and a decode into a string would fail the parse on the first
// of them and report it as a workflow that does not parse.
type step struct {
	Uses string               `yaml:"uses"`
	Run  string               `yaml:"run"`
	Env  map[string]yaml.Node `yaml:"env"`
}

// names is a field GitHub spells either way — `needs: suite` and
// `needs: [suite]` are one thing — read as both here so that the cases hold the
// property rather than the spelling this repository happens to use today.
type names []string

func (n *names) UnmarshalYAML(node *yaml.Node) error {
	var one string
	if err := node.Decode(&one); err == nil {
		*n = names{one}
		return nil
	}
	var many []string
	if err := node.Decode(&many); err != nil {
		return err
	}
	*n = many
	return nil
}

// workflowOf parses one file under `.github/workflows/` by name.
func workflowOf(t *testing.T, name string) workflowFile {
	t.Helper()

	source, err := os.ReadFile(filepath.Join(root(t), ".github", "workflows", name))
	if err != nil {
		t.Fatalf("this repository has no %s: %v", name, err)
	}
	var parsed workflowFile
	if err := yaml.Unmarshal(source, &parsed); err != nil {
		t.Fatalf("%s does not parse: %v", name, err)
	}
	return parsed
}

// allWorkflows is every workflow this repository ships, by filename, for the
// rules that are the directory's rather than one file's.
func allWorkflows(t *testing.T) map[string]workflowFile {
	t.Helper()

	directory := filepath.Join(root(t), ".github", "workflows")
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	parsed := map[string]workflowFile{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		parsed[entry.Name()] = workflowOf(t, entry.Name())
	}
	if len(parsed) == 0 {
		t.Fatal(".github/workflows holds no workflow; a rule that ranges over nothing is green for the wrong reason")
	}
	return parsed
}

// runsOf is every command a workflow runs, in no particular order: what these
// cases ask of a workflow is that something in it does a thing, and which step
// does it is the file's business.
func runsOf(parsed workflowFile) []string {
	var runs []string
	for _, job := range parsed.Jobs {
		for _, step := range job.Steps {
			if step.Run != "" {
				runs = append(runs, step.Run)
			}
		}
	}
	return runs
}

// usesOf is every action and called workflow a workflow names, jobs and steps
// alike — `uses:` at both levels is one rule's subject.
func usesOf(parsed workflowFile) []string {
	var uses []string
	for _, job := range parsed.Jobs {
		if job.Uses != "" {
			uses = append(uses, job.Uses)
		}
		for _, step := range job.Steps {
			if step.Uses != "" {
				uses = append(uses, step.Uses)
			}
		}
	}
	return uses
}
