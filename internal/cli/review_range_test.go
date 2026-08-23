package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The review's range, where a golden cannot hold it (§8, ADR-0067, ADR-0071,
// issue #164).
//
// Four of the six artefacts in testdata/ranged-review/repo anchor on **that
// file's blob under the supplying Run's `repo_revision`** — a Target
// declaration, a Manifest, a Repository declaration and a Procedure reached
// only by invocation — and a `repo_revision` is a commit. A commit id is a
// function of the tree, the message, the identity and the dates, so the fixture
// mints one at run time and no case directory can name it. These cases
// materialise the repository, read its `HEAD`, and seed the Journal against it,
// which is also what lets one seeded Journal be reviewed twice.

// rangedRepository is the fixture those cases share: the ranged-review
// repository materialised as a real git repository, with its own `HEAD` in
// hand and its Store branch left for the case to seed.
type rangedRepository struct {
	c    goldenCase
	run  goldenRun
	head string
}

// ranged materialises it. The case directory carries a repo-from, a git marker
// and a now and no argv at all: what it drives is decided here, one review at a
// time, which is the whole reason these are not goldens.
func ranged(t *testing.T) rangedRepository {
	t.Helper()

	c := corpusCase(t, "review/a-journal-seeded-at-run-time", "hyper", "review", "hyper.yaml")
	run := c.invocation(t)
	return rangedRepository{c: c, run: run, head: run.fixture.text(t, run.fixture.root, "rev-parse", "HEAD")}
}

// journal writes the Store branch: one file per path given, committed
// parentless and pointed at by the ref, exactly as a case's store/ directory
// would be. A case calls it once and may call it with nothing but the marker
// file, which is a Store that answers and holds no entry.
func (r rangedRepository) journal(t *testing.T, files map[string]string) {
	t.Helper()

	seed := t.TempDir()
	written := map[string]string{"STORE.md": "# hyper store\n"}
	for path, content := range files {
		written[path] = content
	}
	for path, content := range written {
		full := filepath.Join(seed, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := r.run.fixture.root
	r.run.fixture.run(t, root, "update-ref", store.Ref, r.run.fixture.commitAbove(t, seed, ""))
}

// review drives one review of the materialised repository through the one
// entry point, and answers its page.
func (r rangedRepository) review(t *testing.T, positional string) (stdout, stderr string, exit int) {
	t.Helper()

	var out, errs bytes.Buffer
	args := []string{"review", "--repo-dir", r.run.fixture.root, positional}
	exit = cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t))
	return out.String(), errs.String(), exit
}

// rangeOf is the range a review stated, read back off the header's first line:
// whatever stands to the right of the path column. The case asserts against the
// whole of it, so an absence and a range are told apart by what the header says
// rather than by what the case looked for.
//
// The path is found by the gap behind it rather than by being parsed: this
// screen's least gap between two things on one line is two spaces, and a
// repository-relative path holds none. On the one artefact with no file the
// line collapses to its one field and there is no gap to find, which is
// ADR-0068's rendering read back — the sentence stands alone.
func rangeOf(t *testing.T, page string) string {
	t.Helper()

	first := strings.TrimSpace(headerOf(t, page)[0])
	if _, stated, padded := strings.Cut(first, "  "); padded {
		return strings.TrimSpace(stated)
	}
	return first
}

// entryFiles is one closed, non-rehearsal Journal entry of the Procedure
// `watch`, in the shape §7 writes one: run.json, outcome.json, and the two Step
// files the Run's two Steps wrote — the second of them reached through the
// nested Procedure `probe`, which is what puts `probe` in a Step file's `path`.
//
// The revisions are the caller's, so a case can seed the entry that read the
// repository as it stands and the entry that names something this clone does
// not hold from one writer.
func entryFiles(repoRevision string, dirty bool) map[string]string {
	const at = "journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/"
	marker := ""
	if dirty {
		marker = `
    "repo_dirty": true,`
	}
	return map[string]string{
		at + "run.json": `{
  "dry_run": false,
  "procedure": "watch",
  "provenance": {
    "hyper_version": "1.4.0",
    "procedure_revision": "5639c68a1e0a79e88a92cfd1153dd40d4febd1cf",` + marker + `
    "repo_revision": "` + repoRevision + `"
  },
  "run_id": "01984c9f-3a20-7b41-8c05-2d6e9f31a7b4",
  "schema_version": 1,
  "started_at": "2026-06-26T03:00:00.000Z",
  "trigger": {
    "actor": "igor",
    "cause": "cron",
    "executor": "local",
    "host": "thinkpad"
  }
}
`,
		at + "outcome.json": `{
  "ended_at": "2026-06-26T03:01:44.000Z",
  "outcome": "completed",
  "schema_version": 1
}
`,
		at + "steps/0001.json": stepFileText(1, "beat", ""),
		at + "steps/0002.json": stepFileText(2, "again", "probe.again"),
	}
}

// stepFileText is one Step file of that entry: the Definition it named, the
// Manifest it loaded, the Target it bound, and — on the second — the invocation
// chain it was reached through.
func stepFileText(step int, id, path string) string {
	chain := ""
	if path != "" {
		chain = `
  "path": "` + path + `",`
	}
	return `{
  "definition": "heartbeat",
  "disposition": "ran",
  "ended_at": "2026-06-26T03:00:` + [...]string{"", "12", "24"}[step] + `.000Z",
  "id": "` + id + `",
  "kind": "read",
  "operation": "check_http",` + chain + `
  "provenance": {
    "definition_revision": "295fea3b5d37d11f4007541e1721ebcc5fd40030",
    "manifest_digest": "sha256:2b7e5c81f0a394d6e2b7c051a83f6d940e17c5b28a306f4d91e75b0c2f8a63d1"
  },
  "provider": "uptime",
  "schema_version": 1,
  "started_at": "2026-06-26T03:00:00.000Z",
  "step": ` + [...]string{"", "1", "2"}[step] + `,
  "target": "staging"
}
`
}

// TestReviewRange_TheThreeAndANestedProcedureOpenAtTheFilesBlobUnderRepoRevision
// is ADR-0067's other arm: the four artefacts carrying no revision of their own
// open at **that file's blob** under the same Run's `repo_revision`, resolved at
// render and stored nowhere.
//
// It asserts a blob and not merely a value, which is the whole of what #56
// refused: a commit in that position invites the reading where a repository
// revision moves because a README did, and the range sits on the line beside
// one artefact's path.
func TestReviewRange_TheThreeAndANestedProcedureOpenAtTheFilesBlobUnderRepoRevision(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	for _, artefact := range []string{"targets/staging.yaml", "providers/uptime.yaml", "hyper.yaml", "procedures/probe.yaml"} {
		blob := r.run.fixture.text(t, r.run.fixture.root, "rev-parse", r.head+":"+artefact)

		stdout, stderr, exit := r.review(t, artefact)
		if exit != cli.ExitClean || stderr != "" {
			t.Fatalf("%s: exit = %d, stderr = %q, want a clean review", artefact, exit, stderr)
		}
		if got, want := rangeOf(t, stdout), blob[:7]+" → working tree"; got != want {
			t.Errorf("%s states the range as %q, want %q", artefact, got, want)
		}
		if strings.Contains(stdout, r.head[:7]) {
			t.Errorf("%s renders the commit %s; a range names one file's bytes (ADR-0067)", artefact, r.head[:7])
		}
	}
}

// TestReviewRange_ATopLevelProcedureAnchorsOnItsOwnRevisionAndTheNestedOneDoesNot
// is the reason ADR-0067 keys on the member and not on the `kind:`: one kind
// falls on both sides of the line at once, and the artefact that fails the test
// is the same artefact that passes it one invocation up.
func TestReviewRange_ATopLevelProcedureAnchorsOnItsOwnRevisionAndTheNestedOneDoesNot(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	top, _, _ := r.review(t, "procedures/watch.yaml")
	if got, want := rangeOf(t, top), "5639c68 → working tree"; got != want {
		t.Errorf("the top-level Procedure states %q, want its own procedure_revision %q", got, want)
	}
	nested, _, _ := r.review(t, "procedures/probe.yaml")
	blob := r.run.fixture.text(t, r.run.fixture.root, "rev-parse", r.head+":procedures/probe.yaml")
	if got, want := rangeOf(t, nested), blob[:7]+" → working tree"; got != want {
		t.Errorf("the nested Procedure states %q, want the file's blob under repo_revision %q", got, want)
	}
}

// TestReviewRange_ARehearsalSuppliesNoRange is §7's disqualification read on
// this surface: rehearsing a widened `destroy` Bound would otherwise retire the
// flag that was the warning, and a rehearsal disarming a review surface is a
// shape that rules everywhere rather than on the artefact it named.
func TestReviewRange_ARehearsalSuppliesNoRange(t *testing.T) {
	r := ranged(t)
	rehearsed := entryFiles(r.head, false)
	for path, content := range rehearsed {
		rehearsed[path] = strings.Replace(content, `"dry_run": false`, `"dry_run": true`, 1)
	}
	r.journal(t, rehearsed)

	// All five kinds at once, the one entry there is having read every one
	// of them: a rehearsal supplies no range **at all**, which is the one
	// filter that does not vary with what the artefact anchors on.
	for _, artefact := range []string{"procedures/watch.yaml", "procedures/probe.yaml", "definitions/heartbeat.yaml", "targets/staging.yaml", "providers/uptime.yaml", "hyper.yaml"} {
		stdout, _, exit := r.review(t, artefact)
		if exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", artefact, exit)
		}
		if got := rangeOf(t, stdout); !strings.HasPrefix(got, "no baseline — ") {
			t.Errorf("%s states %q, want the absence a rehearsal leaves", artefact, got)
		}
	}
}

// TestReviewRange_ADirtyEntrySuppliesNoRangeToAnArtefactWithNoRevisionOfItsOwn
// is the criterion stated as it is written: **two reviews over one seeded
// Journal, differing only in the artefact under review**.
//
// The blob under that commit resolves perfectly and names bytes that did not
// run, so a Target declaration's gutter would mark a line that did not move and
// miss one that did — the one screen ADR-0026 says may not lie. The two members
// that survive a dirty tree are hashes of what ran, so the Definition beside it
// ranges normally.
func TestReviewRange_ADirtyEntrySuppliesNoRangeToAnArtefactWithNoRevisionOfItsOwn(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, true))

	declaration, _, exit := r.review(t, "targets/staging.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("the Target declaration exited %d, want a clean review", exit)
	}
	if got, want := rangeOf(t, declaration), "no baseline — nothing has bound staging"; got != want {
		t.Errorf("the Target declaration states %q, want %q", got, want)
	}

	definition, _, exit := r.review(t, "definitions/heartbeat.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("the Definition exited %d, want a clean review", exit)
	}
	if got, want := rangeOf(t, definition), "295fea3 → working tree"; got != want {
		t.Errorf("the Definition states %q, want the definition_revision that dirty entry recorded: %q", got, want)
	}
}

// TestReviewRange_NotRunsSentenceDiffersByKindAndTheWireNameDoesNot is §8's
// payment for a closed set's stability: the wire carries one name and the page
// names the act that would supply a range, which differs by kind. A reader told
// *`staging` has not run* learns neither what happened nor what to do.
func TestReviewRange_NotRunsSentenceDiffersByKindAndTheWireNameDoesNot(t *testing.T) {
	r := ranged(t)
	r.journal(t, nil)

	sentences := map[string]string{
		"procedures/watch.yaml":      "no baseline — watch has not run",
		"targets/staging.yaml":       "no baseline — nothing has bound staging",
		"definitions/heartbeat.yaml": "no baseline — nothing has named heartbeat",
		"providers/uptime.yaml":      "no baseline — nothing has loaded uptime",
		// The fifth form, and the rule holding rather than an exception to
		// it: a Repository declaration is read by every Run there is and
		// declares no name, so the act is any Run at all and the sentence
		// carries no name to state.
		"hyper.yaml": "no baseline — nothing has run",
	}
	said := map[string]bool{}
	for artefact, want := range sentences {
		stdout, _, exit := r.review(t, artefact)
		if exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", artefact, exit)
		}
		if got := rangeOf(t, stdout); got != want {
			t.Errorf("%s states %q, want %q", artefact, got, want)
		}
		said[rangeOf(t, stdout)] = true
	}
	if len(said) != len(sentences) {
		t.Errorf("the five kinds said %d different things; the act differs by kind (§8)", len(said))
	}
}

// TestReviewRange_TheWireCarriesOneNameForEveryOneOfThoseSentences is the other
// half of the paragraph above, and the reason the page pays and the wire does
// not: a consumer filtering on `baseline_absent` wants one stable string.
func TestReviewRange_TheWireCarriesOneNameForEveryOneOfThoseSentences(t *testing.T) {
	r := ranged(t)
	r.journal(t, nil)

	for _, artefact := range []string{"procedures/watch.yaml", "targets/staging.yaml", "definitions/heartbeat.yaml", "providers/uptime.yaml", "hyper.yaml"} {
		var out, errs bytes.Buffer
		args := []string{"review", "--repo-dir", r.run.fixture.root, artefact, "--json"}
		if exit := cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t)); exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", artefact, exit)
		}
		header := strings.SplitN(out.String(), "\n", 2)[0]
		if !strings.Contains(header, `"baseline_absent":"not-run"`) {
			t.Errorf("%s carries %s, want baseline_absent not-run on every one of them", artefact, header)
		}
		if strings.Contains(header, `"baseline"`) {
			t.Errorf("%s carries a baseline beside the absence; exactly one of the two is written: %s", artefact, header)
		}
		if strings.Contains(header, `"last_run"`) {
			t.Errorf("%s carries last_run; one Journal absence serves both readings of that entry (§8): %s", artefact, header)
		}
	}
}

// TestReviewRange_TheMostRecentQualifyingRunSupplies is the multiplicity §8
// resolves: several Runs qualify for all but the first artefact, and a Run
// reads one working tree, so the range is the most recent one's and the
// multiplicity is across entries rather than inside one.
func TestReviewRange_TheMostRecentQualifyingRunSupplies(t *testing.T) {
	r := ranged(t)
	entries := entryFiles(r.head, false)
	// An older entry of the same Procedure, naming a Definition revision
	// this clone does not hold. It is the one the range would open at if the
	// walk took the first entry it found rather than the most recent.
	const (
		newer = "journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/"
		older = "journal/2026/06/01/019818e0-1c40-7a02-9d31-5f8b2c6e4a19/"
	)
	stale := strings.NewReplacer(
		"01984c9f-3a20-7b41-8c05-2d6e9f31a7b4", "019818e0-1c40-7a02-9d31-5f8b2c6e4a19",
		"2026-06-26", "2026-06-01",
		"295fea3b5d37d11f4007541e1721ebcc5fd40030", "1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01",
	)
	seeded := map[string]string{}
	for path, content := range entries {
		seeded[path] = content
		seeded[older+strings.TrimPrefix(path, newer)] = stale.Replace(content)
	}
	r.journal(t, seeded)

	stdout, _, exit := r.review(t, "definitions/heartbeat.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := rangeOf(t, stdout), "295fea3 → working tree"; got != want {
		t.Errorf("the range states %q, want the most recent qualifying Run's %q", got, want)
	}
}

// TestReviewRange_TheAbsencesRankAsAPipelineOnceTheStoreCanAnswer walks the
// three stages a Store makes reachable, each one only where the one before it
// did not fire (§8, §12): no branch at all, then a branch holding nothing that
// anchors this file, then a Run that answered and named a revision the clone
// does not hold.
func TestReviewRange_TheAbsencesRankAsAPipelineOnceTheStoreCanAnswer(t *testing.T) {
	r := ranged(t)

	stdout, _, _ := r.review(t, "procedures/watch.yaml")
	if got, want := rangeOf(t, stdout), "no baseline — no Store"; got != want {
		t.Errorf("with no branch the header states %q, want %q", got, want)
	}

	r.journal(t, nil)
	stdout, _, _ = r.review(t, "procedures/watch.yaml")
	if got, want := rangeOf(t, stdout), "no baseline — watch has not run"; got != want {
		t.Errorf("with an empty Journal the header states %q, want %q", got, want)
	}

	absent := entryFiles(r.head, false)
	for path, content := range absent {
		absent[path] = strings.Replace(content, "5639c68a1e0a79e88a92cfd1153dd40d4febd1cf", "1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01", 1)
	}
	r.journal(t, absent)
	stdout, _, _ = r.review(t, "procedures/watch.yaml")
	if got, want := rangeOf(t, stdout), "no baseline — 1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01 is not in this clone"; got != want {
		t.Errorf("with the object absent the header states %q, want %q", got, want)
	}
}

// TestReviewRange_BuiltInOutranksAStoreThatCanAnswer is the ranking's whole
// point read where it now bites: the artefact with no file has no range at any
// point in its life, so rendering *no Store* there — or, with a Store in hand,
// *nothing has loaded shell* — promises a range no act will ever produce (§8,
// ADR-0067).
func TestReviewRange_BuiltInOutranksAStoreThatCanAnswer(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	stdout, _, exit := r.review(t, "shell")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want the built-in Manifest rendered", exit)
	}
	if got, want := rangeOf(t, stdout), "no baseline — shell ships in the binary"; got != want {
		t.Errorf("the built-in states %q, want %q", got, want)
	}
}

// TestReviewRange_LastRunIsAbsentUnderTheFirstThreeAbsences is §8's rule with
// its one exception held out: where the absence is the Journal's it is missing
// for the range and for *last ran* alike, so the header states it once — and
// `not-in-clone`, where the entry was found and only its bytes were not, is
// asserted one case up (ADR-0071).
func TestReviewRange_LastRunIsAbsentUnderTheFirstThreeAbsences(t *testing.T) {
	r := ranged(t)

	// `built-in` on the one artefact with no file, `no-store` with no branch
	// at all, and `not-run` with a branch that answered and holds nothing.
	cases := []struct{ positional, absent string }{
		{"shell", "built-in"},
		{"procedures/watch.yaml", "no-store"},
	}
	for _, c := range cases {
		assertNoLastRun(t, r, c.positional, c.absent)
	}
	r.journal(t, nil)
	assertNoLastRun(t, r, "procedures/watch.yaml", "not-run")
}

// assertNoLastRun holds one review's header row to carrying the absence named
// and no Journal entry beside it.
func assertNoLastRun(t *testing.T, r rangedRepository, positional, absent string) {
	t.Helper()

	var out, errs bytes.Buffer
	args := []string{"review", "--repo-dir", r.run.fixture.root, positional, "--json"}
	if exit := cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t)); exit != cli.ExitClean {
		t.Fatalf("%s: exit = %d, want a clean review", positional, exit)
	}
	header := strings.SplitN(out.String(), "\n", 2)[0]
	if !strings.Contains(header, `"baseline_absent":"`+absent+`"`) {
		t.Fatalf("%s carries %s, want baseline_absent %s", positional, header, absent)
	}
	if strings.Contains(header, `"last_run"`) {
		t.Errorf("%s carries last_run under %s; one absence serves both readings (§8): %s", positional, absent, header)
	}
}

// TestReviewRange_NotInCloneRendersTheRevisionWholeAndNamesNoAct is ADR-0071's
// own shape: three causes reach it — a shallow clone, a partial one, a history
// that was rewritten — and no one act repairs all three, so any act the
// sentence named would be a guess.
func TestReviewRange_NotInCloneRendersTheRevisionWholeAndNamesNoAct(t *testing.T) {
	const named = "1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01"

	r := ranged(t)
	absent := entryFiles(r.head, false)
	for path, content := range absent {
		absent[path] = strings.Replace(content, "5639c68a1e0a79e88a92cfd1153dd40d4febd1cf", named, 1)
	}
	r.journal(t, absent)

	stdout, _, exit := r.review(t, "procedures/watch.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got := rangeOf(t, stdout); !strings.Contains(got, named) {
		t.Errorf("the header states %q, want the revision whole", got)
	}
	for _, act := range []string{"--unshallow", "fetch", "hyper ", "run "} {
		if strings.Contains(rangeOf(t, stdout), act) {
			t.Errorf("the sentence names %q; three causes reach this absence and no one act repairs all three", act)
		}
	}
	// The exception §8 states, and the one absence that does not travel: the
	// entry was found and only its bytes were not, so *last ran* renders in
	// full beside a range that cannot open.
	if got := headerOf(t, stdout)[1]; !strings.HasSuffix(got, "· last ran 41 days ago") {
		t.Errorf("the gloss line reads %q, want *last ran* rendering beside a range that could not open", got)
	}
}

// TestReviewRange_ARangeThatCannotOpenNeverMakesReviewDecline is §9's exit codes
// held over every way this reading stops: `review` exits 1 for the artefact
// under review failing to load and for nothing else, so an absent Store, an
// empty Journal and a missing object are all renderings.
func TestReviewRange_ARangeThatCannotOpenNeverMakesReviewDecline(t *testing.T) {
	r := ranged(t)

	absent := entryFiles(r.head, false)
	for path, content := range absent {
		absent[path] = strings.Replace(content, "5639c68a1e0a79e88a92cfd1153dd40d4febd1cf", "1f0a3d7c9b2e4a6d8f0b1c3e5a7d9f2b4c6e8a01", 1)
	}
	seeds := []func(){func() {}, func() { r.journal(t, nil) }, func() { r.journal(t, absent) }}
	for _, seed := range seeds {
		seed()
		stdout, stderr, exit := r.review(t, "procedures/watch.yaml")
		if exit != cli.ExitClean {
			t.Errorf("exit = %d, want 0 — a range that cannot open is a rendering", exit)
		}
		if stderr != "" {
			t.Errorf("stderr carries %q; an absent range is stated on the page and nowhere else", stderr)
		}
		if !strings.Contains(stdout, "kind: procedure") {
			t.Error("the page dropped the artefact's own lines; a review renders the working tree whatever the range did")
		}
	}
}

// TestReviewRange_TheRevisionIsABlobOnAllFiveArtefactsAndNeverACommit is the
// criterion the whole anchor rule exists for. Two artefacts carry a blob id in
// Provenance and four resolve one at render, and what a reader must never be
// handed on that line is a commit: a repository revision moves when a README
// does, which is the reading #56 refused.
func TestReviewRange_TheRevisionIsABlobOnAllFiveArtefactsAndNeverACommit(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	for _, artefact := range []string{"procedures/watch.yaml", "procedures/probe.yaml", "definitions/heartbeat.yaml", "targets/staging.yaml", "providers/uptime.yaml", "hyper.yaml"} {
		var out, errs bytes.Buffer
		args := []string{"review", "--repo-dir", r.run.fixture.root, artefact, "--json"}
		if exit := cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t)); exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", artefact, exit)
		}

		// The wire abbreviates nothing, so the whole revision is on the
		// row and git can be asked what kind of object it is.
		header := strings.SplitN(out.String(), "\n", 2)[0]
		_, rest, carried := strings.Cut(header, `"baseline":"`)
		if !carried {
			t.Fatalf("%s carries no baseline: %s", artefact, header)
		}
		baseline, _, _ := strings.Cut(rest, `"`)
		if len(baseline) != 40 {
			t.Errorf("%s carries baseline %q; the wire abbreviates nothing (ADR-0047)", artefact, baseline)
		}
		if kind := r.run.fixture.text(t, r.run.fixture.root, "cat-file", "-t", baseline); kind != "blob" {
			t.Errorf("%s opens at a %s; a range names one file's bytes (ADR-0067)", artefact, kind)
		}
	}
}

// TestReviewRange_TheHeaderRendersOneJournalEntryTwiceFromOneLookup is §8's
// *one lookup, two notations*: the range and *last ran* read the same entry
// under the same filter, so the revision on the first line and the age on the
// second are one Journal entry rendered twice rather than two entries once
// each.
func TestReviewRange_TheHeaderRendersOneJournalEntryTwiceFromOneLookup(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	stdout, _, exit := r.review(t, "procedures/watch.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	header := headerOf(t, stdout)
	if got, want := rangeOf(t, stdout), "5639c68 → working tree"; got != want {
		t.Errorf("the range states %q, want %q", got, want)
	}
	if got, want := header[1], "03:00 UTC every Monday · ≈4.3 runs/month · last ran 41 days ago"; got != want {
		t.Errorf("the gloss line reads %q, want %q", got, want)
	}

	// The wire carries the entry the header rendered an age from: the Run
	// whole, and the instant it ended rather than the age, which is stale
	// the moment it is written.
	var out, errs bytes.Buffer
	args := []string{"review", "--repo-dir", r.run.fixture.root, "procedures/watch.yaml", "--json"}
	if exit := cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t)); exit != cli.ExitClean {
		t.Fatalf("--json exited %d, want a clean review", exit)
	}
	if want := `"last_run":{"run":"01984c9f-3a20-7b41-8c05-2d6e9f31a7b4","ended":"2026-06-26T03:01:44.000Z"}`; !strings.Contains(out.String(), want) {
		t.Errorf("the header row is %s, want it to carry %s", strings.SplitN(out.String(), "\n", 2)[0], want)
	}
}

// TestReviewRange_AnArtefactWithNoCadenceCarriesNoAgeAndNoLastRun is the other
// half of that rule: §8 gives the header the last Journal entry **beside the
// gloss**, so an artefact with no Cadence beneath it has no line to hang an age
// on — and the row carries no instant either, the two being one member in two
// notations.
func TestReviewRange_AnArtefactWithNoCadenceCarriesNoAgeAndNoLastRun(t *testing.T) {
	r := ranged(t)
	r.journal(t, entryFiles(r.head, false))

	stdout, _, exit := r.review(t, "targets/staging.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if header := headerOf(t, stdout); len(header) != 1 {
		t.Errorf("the header is %q; an artefact with no Cadence renders a one-line header", header)
	}
	if strings.Contains(stdout, "last ran") {
		t.Errorf("the page carries an age with no gloss to render it beside:\n%s", stdout)
	}

	var out, errs bytes.Buffer
	args := []string{"review", "--repo-dir", r.run.fixture.root, "targets/staging.yaml", "--json"}
	if exit := cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t)); exit != cli.ExitClean {
		t.Fatalf("--json exited %d, want a clean review", exit)
	}
	if strings.Contains(out.String(), `"last_run"`) {
		t.Errorf("the row carries last_run where no age rendered: %s", strings.SplitN(out.String(), "\n", 2)[0])
	}
}

// TestReviewRange_AnOpenEntrySuppliesARangeAndNoAge is the split the header's
// two lines make where an entry holds no account of its end: its run.json
// carries the revision like any other's, and *last ran* is a rendering of when
// the Run **ended** — which an open entry has none of, and which inventing from
// its start would assert that a Run still in flight is over (§7, §8).
func TestReviewRange_AnOpenEntrySuppliesARangeAndNoAge(t *testing.T) {
	r := ranged(t)
	open := entryFiles(r.head, false)
	delete(open, "journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/outcome.json")
	r.journal(t, open)

	stdout, _, exit := r.review(t, "procedures/watch.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if got, want := rangeOf(t, stdout), "5639c68 → working tree"; got != want {
		t.Errorf("the range states %q, want the open entry's own procedure_revision %q", got, want)
	}
	if got, want := headerOf(t, stdout)[1], "03:00 UTC every Monday · ≈4.3 runs/month"; got != want {
		t.Errorf("the gloss line reads %q, want %q — an open entry has no end to render an age from", got, want)
	}
}
