package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// checkCorpus is the check command's own slice of testdata/ — its cases, and no
// sibling's. Nothing here drives them: the one harness in golden_test.go does
// that, from each case's argv (issue #108). What is read here is what the check
// corpus's golden files say, which is a question about this milestone's closed
// set of error codes and not about how a command runs.
const checkCorpus = "testdata/check"

// milestoneOneErrorCodes is the closed set issue #87 fixes for this
// milestone: thirty-seven error_code members, and no others, reach `hyper
// check` (docs/build/milestones.md's milestone 1, "Which codes land, and
// which do not"). It is:
//
//   - §4's thirty-two static codes, in full. The last of them,
//     procedure-cycle, is a member this milestone was written without: §6
//     said a cyclic invocation graph was rejected before the first Step and
//     §12 held no code for one, so nothing rejected it until issue #141 made
//     the state reachable and issue #146 named the check (§4, §12).
//   - The offline halves of three codes §4 and §6 share — bound-exceeded (an
//     authored values: list longer than the Bound), predicate-type-mismatch
//     (the authored operand faults), and record-identity-collision (its §3
//     load site and its §4 wiring site). All three now have their run-time
//     halves at Expansion too, driven under testdata/run/: two of them with
//     issue #139, and bound-exceeded's with issue #149.
//   - §10's two Cadence walk codes — cadence-run-once and
//     cadence-secret-output — both stated by §4 and riding
//     envelope-exceeded's transitive walk.
//
// version-pin-mismatch and version-pin-absent are milestone 0's gate, which
// `check` inherits rather than contributes; they are exercised by the
// version-pin-* cases under testdata/check/ but are deliberately not members
// of this set.
var milestoneOneErrorCodes = []string{
	"strict-yaml-violation",
	"unknown-key",
	"schema-mismatch",
	"kind-mismatch",
	"name-mismatch",
	"schema-unsupported",
	"credential-slot-malformed",
	"hole-illegal",
	"series-reference",
	"artefact-absent",
	"reference-unresolvable",
	"capability-mismatch",
	"manifest-inconsistent",
	"target-inconsistent",
	"header-reserved",
	"local-reserved",
	"identity-undeclared",
	"target-class-mismatch",
	"definition-kinds-mixed",
	"kind-not-granted",
	"capability-not-granted",
	"operation-not-claimed",
	"target-not-claimed",
	"envelope-exceeded",
	"opaque-destroy-not-granted",
	"bound-missing",
	"bound-illegal",
	"host-not-granted",
	"command-malformed",
	"opaque-destroy-unscoped",
	"skip-if-recorded-unreachable",
	"bound-exceeded",
	"predicate-type-mismatch",
	"record-identity-collision",
	"cadence-run-once",
	"cadence-secret-output",
	"procedure-cycle",
}

// TestMilestoneOneErrorCodes_EveryMemberHasAFailingFixture walks every golden
// file in the check corpus and confirms each of the thirty-seven error_code
// members this milestone contributes appears in at least one of them — the
// human table's ERROR_CODE column or the --json stream's "error_code" field,
// either rendering being fine since both come from one renderer (ADR-0026).
// It fails naming every code an uncovered member, so the closed set and the
// fixture set can never drift apart silently (issue #99).
func TestMilestoneOneErrorCodes_EveryMemberHasAFailingFixture(t *testing.T) {
	var haystack []byte
	for _, c := range goldenCases(t) {
		if filepath.Dir(c.dir) != checkCorpus {
			continue
		}
		for _, golden := range []string{"stdout.golden", "stderr.golden"} {
			haystack = append(haystack, readFile(t, filepath.Join(c.dir, golden))...)
			haystack = append(haystack, '\n')
		}
	}

	var missing []string
	for _, code := range milestoneOneErrorCodes {
		// Bounded on both sides by something other than a lowercase letter
		// or a hyphen, so a code is never credited by appearing as a
		// substring of a longer, different one — every member of this set
		// is a distinct kebab-case token and none is a hyphen-delimited
		// prefix or suffix of another.
		pattern := `(^|[^a-z-])` + regexp.QuoteMeta(code) + `([^a-z-]|$)`
		if !regexp.MustCompile(pattern).Match(haystack) {
			missing = append(missing, code)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d of %d milestone-1 error_code members have no failing fixture under %s/: %v", len(missing), len(milestoneOneErrorCodes), checkCorpus, missing)
	}
}

// runCorpus is the run command's own slice of testdata/. What is read out of it
// below is the same question this file asks of `check`'s corpus, one command
// over: which members of the closed set have a fixture, and which do not.
const runCorpus = "testdata/run"

// codesReachingARun is what §4's thirty-two codes reaching a Run is proved by:
// a representative spread rather than the whole set, and at least one code from
// each of the five artefact kinds (issue #137).
//
// The spread rather than the whole set is the ticket's own reading and it is
// worth stating why it is enough. **Nothing is implemented for these codes at
// Run start.** Milestone 1 built every one and holds every one to a fixture
// under testdata/check — the list above this one — and what a Run adds is the
// *path*: `check` is re-run in full with nothing skipped, so a repository that
// `check` refuses is a repository a Run refuses (§6, ADR-0061). A corpus
// re-driving all thirty-two through `run` would assert internal/verify runs
// thirty-two times over, which is one claim and not thirty-two.
//
// What a spread does have to cover is the **shape** of the path, and that is
// what the five artefact kinds are for: the walk reaches every location, and a
// Refusal cites the file the reader must edit whichever of them it came from.
//
// The three that are not §4's ride here beside them because they reach a Run
// the same way and through the same rendering: the Store schema test's, the
// credential pass's and the sink gate's (§6, §9, §12).
//
// bound-exceeded is the fourth that is not §4's, and it is here on a different
// footing from all of them: it is not a code arriving through the re-run of
// `check` but §6's own, an Expansion's count being a number no file holds
// (§5, §6, issue #149). Its offline half has a fixture under testdata/check/
// and a member in the list above; what this one holds is that the run-time
// half is driven too, and that both spell one code.
var codesReachingARun = []string{
	"unknown-key",
	"credential-slot-malformed",
	"header-reserved",
	"artefact-absent",
	"reference-unresolvable",
	"kind-not-granted",
	"envelope-exceeded",
	"store-schema-unsupported",
	"credential-absent",
	"secret-sink-absent",
	"bound-exceeded",
}

// artefactKindsCitedByARefusal is the five reviewed artefacts by where each one
// lives (§3, §12). A Refusal cites the file whose author can act, so a corpus
// that never refused against one of these has never driven that arm of the
// walk.
//
// `hyper.yaml` is a path and the other four are directories, which is the one
// asymmetry §3 already carries: the Repository declaration is the artefact
// keyed by its filename.
var artefactKindsCitedByARefusal = []string{
	"hyper.yaml",
	"targets/",
	"providers/",
	"definitions/",
	"procedures/",
}

// TestCodesReachingARun_EveryMemberHasARefusingFixture holds the spread above
// against the run corpus's own golden files, on the footing the milestone-1
// test states: what is read is what the checked-in goldens say, and a code with
// no fixture is a claim nothing drives.
func TestCodesReachingARun_EveryMemberHasARefusingFixture(t *testing.T) {
	haystack := goldensUnder(t, runCorpus)

	var missing []string
	for _, code := range codesReachingARun {
		pattern := `(^|[^a-z-])` + regexp.QuoteMeta(code) + `([^a-z-]|$)`
		if !regexp.MustCompile(pattern).Match(haystack) {
			missing = append(missing, code)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("%d of %d codes have no Refusing fixture under %s/: %v", len(missing), len(codesReachingARun), runCorpus, missing)
	}
}

// TestCodesReachingARun_EveryArtefactKindIsCited is the other half of the
// ticket's reading: `check` re-runs over the repository whole, so a Refusal has
// to be able to cite any of the five artefacts — and a corpus that only ever
// refused against a Procedure would leave four arms of that walk undriven.
func TestCodesReachingARun_EveryArtefactKindIsCited(t *testing.T) {
	// The branch goldens rather than the pages, because a Refusal's `file`
	// is a key there rather than a column: `outcome.json` holds the array in
	// full and in the canonical encoding, so a match on `"file": "targets/`
	// is the citation itself and never a path that happened to appear in a
	// message (§7).
	var haystack []byte
	for _, c := range goldenCases(t) {
		if filepath.Dir(c.dir) != runCorpus {
			continue
		}
		haystack = append(haystack, readFile(t, filepath.Join(c.dir, "store.golden"))...)
		haystack = append(haystack, '\n')
	}

	var uncited []string
	for _, where := range artefactKindsCitedByARefusal {
		if !bytes.Contains(haystack, []byte(`"file": "`+where)) {
			uncited = append(uncited, where)
		}
	}
	if len(uncited) > 0 {
		t.Errorf("no Refusal under %s/ cites %v; a Run's `check` re-runs over every artefact and must be able to cite each", runCorpus, uncited)
	}
}

// goldensUnder is every stdout and stderr golden of one corpus, concatenated —
// the one read both assertions above are made against, and the same read the
// milestone-1 test makes of `check`'s.
func goldensUnder(t *testing.T, corpus string) []byte {
	t.Helper()

	var haystack []byte
	for _, c := range goldenCases(t) {
		if filepath.Dir(c.dir) != corpus {
			continue
		}
		for _, golden := range []string{"stdout.golden", "stderr.golden"} {
			haystack = append(haystack, readFile(t, filepath.Join(c.dir, golden))...)
			haystack = append(haystack, '\n')
		}
	}
	return haystack
}

// comparingCode is the one member of the closed `error_code` set that compares
// two values, and therefore the only one that ever writes `declared` and
// `observed`: an Expansion's count against the Bound the Step's author declared
// (§5, §7, issue #149).
//
// It is spelled here rather than imported from the package that fires it, on
// this file's own footing: what every assertion below reads is what the
// checked-in goldens say, and a constant imported from the code under test
// would move with it.
const comparingCode = "bound-exceeded"

// TestRefusal_OnlyTheComparingCodeWritesDeclaredAndObserved holds §7's absence
// rule over the two members a Refusal carries only where a check compared two
// values: nothing is invented to fill a member that does not apply, so a check
// that compared nothing writes neither — and the check that did writes both.
//
// It reads the corpus's checked-in branch goldens rather than the code, on the
// milestone-1 test's own footing: what is asserted is what the Journal actually
// holds across every Run the corpus drives, which is the claim a reader of an
// entry relies on. The `--json` stream is held to the same rule beside it, the
// two surfaces being one list of rows rendered twice (ADR-0026).
func TestRefusal_OnlyTheComparingCodeWritesDeclaredAndObserved(t *testing.T) {
	compared := 0
	for _, c := range goldenCases(t) {
		for _, member := range refusalMembersOf(t, filepath.Join(c.dir, "store.golden")) {
			held := member.Declared != nil || member.Observed != nil
			switch {
			case held && member.ErrorCode != comparingCode:
				t.Errorf("%s: %s writes declared/observed and compared no two values", c.name, member.ErrorCode)
			case held && (member.Declared == nil || member.Observed == nil):
				t.Errorf("%s: %s writes one of declared and observed; a comparison has two operands", c.name, member.ErrorCode)
			case !held && member.ErrorCode == comparingCode:
				t.Errorf("%s: %s writes neither declared nor observed; it is the check that compares two", c.name, member.ErrorCode)
			case held:
				compared++
			}
		}
		for _, row := range refusalRowsOf(t, filepath.Join(c.dir, "stdout.golden")) {
			code, _ := row["error_code"].(string)
			_, declared := row["declared"]
			_, observed := row["observed"]
			switch {
			case (declared || observed) && code != comparingCode:
				t.Errorf("%s: the %s row carries declared/observed and compared no two values", c.name, code)
			case (declared || observed) && !(declared && observed):
				t.Errorf("%s: the %s row carries one of declared and observed; a comparison has two operands", c.name, code)
			case !declared && !observed && code == comparingCode:
				t.Errorf("%s: the %s row carries neither declared nor observed; it is the check that compares two", c.name, code)
			}
		}
	}
	if compared == 0 {
		t.Errorf("no Refusal in the corpus compares two values; %s has no fixture and this test asserts nothing", comparingCode)
	}
}

// refusalMembersOf is every Refusal member held on an `outcome.json` in one
// case's branch golden, read through the format's own parser and decoded
// through the Store's own reader — so what is asserted above is what a consumer
// of the entry gets rather than what a text search finds. A case with no branch
// golden contributes none.
func refusalMembersOf(t *testing.T, golden string) []store.RefusalMember {
	t.Helper()

	rendered := readFile(t, golden)
	if rendered == absentBranch {
		// A Run that found no branch wrote no entry, so there is no
		// `outcome.json` to read — which is a case contributing
		// nothing rather than a golden this parser cannot read.
		return nil
	}

	var held []store.RefusalMember
	for _, file := range parseRendering(t, rendered) {
		if !strings.HasSuffix(file.path, "/outcome.json") {
			continue
		}
		outcome, err := store.DecodeOutcomeFile([]byte(file.bytes))
		if err != nil {
			t.Fatalf("%s holds a %s this binary cannot read: %v", golden, file.path, err)
		}
		held = append(held, outcome.Refusal...)
	}
	return held
}

// refusalRowsOf is every `refusal` row on one case's stdout, and none at all
// where that stdout is a page rather than a stream — a page carries no line
// opening `{"type":"refusal"`, `type` being the row's first key by contract
// (§8, internal/render).
//
// It decodes into a bare mapping rather than into cli's own row type, which is
// unexported and would in any case be the shape under test answering a question
// about itself: what is asserted is which **keys** the checked-in wire carries.
func refusalRowsOf(t *testing.T, golden string) []map[string]any {
	t.Helper()

	var rows []map[string]any
	for _, line := range strings.Split(readFile(t, golden), "\n") {
		if !strings.HasPrefix(line, `{"type":"refusal"`) {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%s holds a refusal row that is not an object: %v", golden, err)
		}
		rows = append(rows, row)
	}
	return rows
}
