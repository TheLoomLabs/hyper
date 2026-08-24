package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// The gutter's change column and the three change names, where a golden cannot
// hold them (§8, §12, issue #168).
//
// Three of the five artefacts anchor on **that file's blob under the supplying
// Run's `repo_revision`**, and a `repo_revision` is a commit — a function of the
// tree, the message, the identity and the dates, which no case directory can
// name. So the cases here materialise the five-artefact demonstration
// repository, read its `HEAD`, seed the Journal against it and then edit the
// working tree, which is the one thing a checked-in `uncommitted/` cannot do
// for an artefact whose range opens at a commit.
//
// What a golden holds instead is the two artefacts that anchor on a blob:
// `review/the-three-change-names-on-a-procedure`, which is §8's own worked
// example rendered, and `review/a-definition-widened-and-narrowed` beside it.

// demoRanged is the five-artefact demonstration repository materialised, its
// Journal left for the case to seed.
func demoRanged(t *testing.T) rangedRepository {
	t.Helper()
	return rangedIn(t, "review/a-demo-journal-seeded-at-run-time")
}

// demoEntry is one closed, non-rehearsal Journal entry of `retire-preview-dns`
// against that repository: the Run that read every artefact in it, so that one
// seeding serves a review of any of them.
func demoEntry(repoRevision string) map[string]string {
	const at = "journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/"
	return map[string]string{
		at + "run.json": `{
  "dry_run": false,
  "procedure": "retire-preview-dns",
  "provenance": {
    "hyper_version": "1.4.0",
    "procedure_revision": "63e064ae887bcdf2c78a8ede0d647d411da0e56c",
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
		at + "steps/0001.json": `{
  "definition": "preview-dns",
  "disposition": "ran",
  "ended_at": "2026-06-26T03:00:12.000Z",
  "id": "retire",
  "kind": "destroy",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "faa5bb20bbfc9936fae25486223ae095826dbf4d",
    "manifest_digest": "sha256:2b7e5c81f0a394d6e2b7c051a83f6d940e17c5b28a306f4d91e75b0c2f8a63d1"
  },
  "provider": "cloudflare-dns",
  "schema_version": 1,
  "started_at": "2026-06-26T03:00:00.000Z",
  "step": 1,
  "target": "cloudflare-prod"
}
`,
	}
}

// edit rewrites one artefact of the materialised working tree, replacing one
// run of text with another and failing where the text it names is not there: a
// case editing bytes that moved is a case asserting nothing.
//
// It is the working tree and never the commit, which is the state the change
// column exists for — an agent's edit, uncommitted, on the branch a human is
// about to approve.
func (r rangedRepository) edit(t *testing.T, path, from, to string) {
	t.Helper()

	full := filepath.Join(r.run.fixture.root, filepath.FromSlash(path))
	held, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(held), from) {
		t.Fatalf("%s holds no %q", path, from)
	}
	edited := strings.Replace(string(held), from, to, 1)
	if err := os.WriteFile(full, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
}

// blobOf writes content into the materialised repository's object store and
// answers its blob id, so that a case can seed a Journal naming bytes of its
// own choosing. It writes no file into the tree: an object is content-addressed
// and a range opens at one, whatever commit does or does not carry it.
func (r rangedRepository) blobOf(t *testing.T, content string) string {
	t.Helper()

	at := filepath.Join(t.TempDir(), "object")
	if err := os.WriteFile(at, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return r.run.fixture.text(t, r.run.fixture.root, "hash-object", "-w", "--no-filters", "--", at)
}

// reviewJSON drives one review in the `--json` mode, which is the mode §8 says
// the relation between a flag and the gutter is detectable in.
func (r rangedRepository) reviewJSON(t *testing.T, positional string) (stdout, stderr string, exit int) {
	t.Helper()

	var out, errs bytes.Buffer
	args := []string{"review", "--repo-dir", r.run.fixture.root, positional, "--json"}
	exit = cli.Main(args, &out, &errs, r.c.process(t, r.run), r.c.facts(t))
	return out.String(), errs.String(), exit
}

// holds reports whether one of those lines is the text given, whole.
func holds(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

// TestReviewChanges_ATargetDeclarationsCredentialSourceAndItsAcceptedKinds is
// two of the eight classes on the artefact §12 says the credential source
// belongs to, and the two directions it decides between.
//
// **The credential source takes `changed` and never a direction**, its value
// being a name and not a magnitude; the accepted Kinds compare as a set, and
// one side containing the other is a direction. It is names only and never
// values: §6's whole reason for naming variables explicitly is that
// `env: STAGING_TOKEN` → `env: PROD_TOKEN` is a visible one-line edit (§12,
// ADR-0007).
func TestReviewChanges_ATargetDeclarationsCredentialSourceAndItsAcceptedKinds(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "targets/cloudflare-prod.yaml", "kinds: [read, mutate, destroy]", "kinds: [read, mutate]")
	r.edit(t, "targets/cloudflare-prod.yaml", "env: CLOUDFLARE_API_TOKEN", "env: CLOUDFLARE_PROD_TOKEN")

	stdout, stderr, exit := r.review(t, "targets/cloudflare-prod.yaml")
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	flags := flagsOf(stdout)
	revision := strings.Fields(rangeOf(t, stdout))[0]
	for _, want := range []string{
		"NARROWED  line 4  kinds             destroy · mutate · read → mutate · read since " + revision,
		"CHANGED   line 8  credential token  CLOUDFLARE_API_TOKEN → CLOUDFLARE_PROD_TOKEN since " + revision,
	} {
		if !holds(flags, want) {
			t.Errorf("the block holds no %q; it holds\n%s", want, strings.Join(flags, "\n"))
		}
	}
}

// TestReviewChanges_AManifestsCapabilitiesAndOperationSet is the two classes a
// Manifest supplies, and the second direction read off set inclusion.
//
// The Operation set is the keys of `operations:` and nothing beneath them: what
// moved when a request changed is the digest a Run records, which is `the
// digests` and has no line in any artefact (§12).
func TestReviewChanges_AManifestsCapabilitiesAndOperationSet(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "providers/cloudflare-dns.yaml", "capabilities: [http]", "capabilities: [http, shell]")

	stdout, _, exit := r.review(t, "providers/cloudflare-dns.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	revision := strings.Fields(rangeOf(t, stdout))[0]
	if want := "WIDENED  line 5   capabilities  http → http · shell since " + revision; !holds(flagsOf(stdout), want) {
		t.Errorf("the block holds no %q; it holds\n%s", want, strings.Join(flagsOf(stdout), "\n"))
	}
}

// TestReviewChanges_ASetThatBothGainsAndLosesAMemberTakesChanged is the rule
// falling out of the two inclusions rather than needing a clause of its own:
// neither holds there, and `changed` is what is left (§12).
func TestReviewChanges_ASetThatBothGainsAndLosesAMemberTakesChanged(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "definitions/preview-dns.yaml", "kinds: [mutate]", "kinds: [read]")

	stdout, _, exit := r.review(t, "definitions/preview-dns.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	revision := strings.Fields(rangeOf(t, stdout))[0]
	if want := "CHANGED  line 4  kinds  mutate → read since " + revision; !holds(flagsOf(stdout), want) {
		t.Errorf("the block holds no %q; it holds\n%s", want, strings.Join(flagsOf(stdout), "\n"))
	}
}

// TestReviewChanges_ASelectorTakesChangedHoweverItMoved is the one place the
// rule is a refusal rather than a reading: this edit adds a conjunct and
// removes none, which is a narrowing by every reading a human would make and by
// none a machine can. Predicate subsumption is undecidable in general, so a
// surface calling it a direction would be inventing the one thing it may not
// invent (§12).
func TestReviewChanges_ASelectorTakesChangedHoweverItMoved(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "procedures/retire-preview-dns.yaml", "        - field: created_on\n          older_than: 14d\n", "")

	stdout, _, exit := r.review(t, "procedures/retire-preview-dns.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	for _, line := range flagsOf(stdout) {
		if strings.HasPrefix(line, "WIDENED") || strings.HasPrefix(line, "NARROWED") {
			t.Errorf("a selector took a direction: %q", line)
		}
	}
	if !holds(flagsOf(stdout), "CHANGED    line 37  step retire   over assets → assets since 63e064a") {
		t.Errorf("the selector drew no changed row; the block holds\n%s", strings.Join(flagsOf(stdout), "\n"))
	}
}

// TestReviewChanges_TheMarksSurviveCommitsBetweenTheRangeAndTheWorkingTree is
// the whole argument for opening the range at the last Run rather than at
// `HEAD` (§8). Against `HEAD` an agent that authored the edit and committed it
// leaves the two sides equal, and the review renders nothing to mark on the one
// branch a human is about to approve.
func TestReviewChanges_TheMarksSurviveCommitsBetweenTheRangeAndTheWorkingTree(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "definitions/preview-dns.yaml", "kinds: [mutate]", "kinds: [mutate, destroy]")
	r.run.fixture.run(t, r.run.fixture.root, "add", "--all")
	r.run.fixture.run(t, r.run.fixture.root, "commit", "--quiet", "--message", "the agent's own commit")

	stdout, _, exit := r.review(t, "definitions/preview-dns.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	revision := strings.Fields(rangeOf(t, stdout))[0]
	if want := "WIDENED  line 4  kinds  mutate → destroy · mutate since " + revision; !holds(flagsOf(stdout), want) {
		t.Errorf("the committed edit draws no flag; the block holds\n%s", strings.Join(flagsOf(stdout), "\n"))
	}
	if !strings.Contains(stdout, "│ ~ kinds: [mutate, destroy]") {
		t.Errorf("the change column marks nothing on the committed edit:\n%s", stdout)
	}
}

// TestReviewChanges_EveryFlagCitesALineSomeGutterRowMarked is the relation §8
// states, asserted the way §8 says it is detectable: **on the wire**, where a
// flag citing a line no `gutter` row marked is mechanically detectable rather
// than a prose mistake.
//
// It runs over every artefact of a repository whose working tree has moved on
// all of them at once, which is what makes the assertion about the rule rather
// than about one rendering.
func TestReviewChanges_EveryFlagCitesALineSomeGutterRowMarked(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "procedures/retire-preview-dns.yaml", "bound: 5", "bound: 9")
	r.edit(t, "procedures/retire-preview-dns.yaml", `cadence: "0 3 * * 1"`, `cadence: "*/5 * * * *"`)
	r.edit(t, "definitions/preview-dns.yaml", "destroy: [delete_dns_record]\n", "")
	r.edit(t, "targets/cloudflare-prod.yaml", "kinds: [read, mutate, destroy]", "kinds: [read, mutate]")
	r.edit(t, "providers/cloudflare-dns.yaml", "capabilities: [http]", "capabilities: [http, shell]")
	r.edit(t, "hyper.yaml", "retention: 90d", "retention: 365d")

	directions := 0
	for _, named := range []string{
		"procedures/retire-preview-dns.yaml",
		"definitions/preview-dns.yaml",
		"targets/cloudflare-prod.yaml",
		"providers/cloudflare-dns.yaml",
		"hyper.yaml",
	} {
		marked := map[int]bool{}
		var cited []int
		for _, row := range reviewStream(t, r, named) {
			switch row["type"] {
			case "gutter":
				marked[int(row["line"].(float64))] = true
			case "flag":
				cited = append(cited, int(row["cites_line"].(float64)))
				if name, _ := row["flag"].(string); name == "widened" || name == "narrowed" || name == "changed" {
					directions++
				}
			}
		}
		if len(marked) == 0 {
			t.Errorf("%s draws no gutter row at all; every artefact here moved", named)
		}
		for _, line := range cited {
			if !marked[line] {
				t.Errorf("%s carries a flag citing line %d, which no gutter row marked", named, line)
			}
		}
	}
	if directions == 0 {
		t.Errorf("no change name reached the stream; the assertion would hold over a review that read no range at all")
	}
}

// TestReviewChanges_ARepositoryDeclarationMarksItsLinesAndFlagsNone is the
// enumeration holding rather than a roster left short: `version:` is the pin,
// which is `the digests`' `hyper_version` and has no class here, and
// `retention:` is one of the lines §12's catch-all counts. Both move on this
// screen and neither is a fact this vocabulary names.
func TestReviewChanges_ARepositoryDeclarationMarksItsLinesAndFlagsNone(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "hyper.yaml", "retention: 90d", "retention: 365d")

	stdout, _, exit := r.review(t, "hyper.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if !strings.Contains(stdout, "│ ~ retention: 365d") {
		t.Errorf("the change column marks nothing on the edited line:\n%s", stdout)
	}
	if got := flagsOf(stdout); !holds(got, "no line the gutter marked draws a flag") {
		t.Errorf("the block holds %q, want the empty state", got)
	}
}

// TestReviewChanges_ABaselineThatWillNotParseMarksItsLinesAndFlagsNone is the
// column and the names read apart. The column is a fact about text and needs no
// parse; a direction is a fact about two values, and there is none to read off
// bytes `hyper` cannot read.
func TestReviewChanges_ABaselineThatWillNotParseMarksItsLinesAndFlagsNone(t *testing.T) {
	r := demoRanged(t)
	entry := demoEntry(r.head)
	// A blob of this repository's own that is not an artefact: the Store's
	// marker file, which is a Markdown heading and no YAML mapping at all.
	// It is named as the Definition's revision, so the range opens and what
	// it opens at will not parse.
	blob := r.blobOf(t, "# not an artefact\n\tand not YAML either\n")
	entry["journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/steps/0001.json"] = strings.Replace(
		entry["journal/2026/06/26/01984c9f-3a20-7b41-8c05-2d6e9f31a7b4/steps/0001.json"],
		"faa5bb20bbfc9936fae25486223ae095826dbf4d", blob, 1)
	r.journal(t, entry)

	stdout, _, exit := r.review(t, "definitions/preview-dns.yaml")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if !strings.Contains(stdout, "│ ~ kind: definition") {
		t.Errorf("the change column marks nothing against a baseline that will not parse:\n%s", stdout)
	}
	if got := flagsOf(stdout); !holds(got, "DESTROY  line 5  destroy claimed for delete_dns_record") {
		t.Errorf("the marker column's own flags went with it; the block holds\n%s", strings.Join(got, "\n"))
	}
	for _, line := range flagsOf(stdout) {
		for _, name := range []string{"WIDENED", "NARROWED", "CHANGED"} {
			if strings.HasPrefix(line, name) {
				t.Errorf("a direction was read off a baseline that will not parse: %q", line)
			}
		}
	}
}

// reviewStream is one review's `--json` rows, decoded.
func reviewStream(t *testing.T, r rangedRepository, named string) []map[string]any {
	t.Helper()

	stdout, _, _ := r.reviewJSON(t, named)
	var rows []map[string]any
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%s: %v on %q", named, err, line)
		}
		rows = append(rows, row)
	}
	return rows
}
