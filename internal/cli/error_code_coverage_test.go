package cli_test

import (
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// checkCorpus is the check command's own slice of testdata/ — its cases, and no
// sibling's. Nothing here drives them: the one harness in golden_test.go does
// that, from each case's argv (issue #108). What is read here is what the check
// corpus's golden files say, which is a question about this milestone's closed
// set of error codes and not about how a command runs.
const checkCorpus = "testdata/check"

// milestoneOneErrorCodes is the closed set issue #87 fixes for this
// milestone: thirty-six error_code members, and no others, reach `hyper
// check` (docs/build/milestones.md's milestone 1, "Which codes land, and
// which do not"). It is:
//
//   - §4's thirty-one static codes, in full.
//   - The offline halves of three codes §4 and §6 share — bound-exceeded (an
//     authored values: list longer than the Bound), predicate-type-mismatch
//     (the authored operand faults), and record-identity-collision (its §3
//     load site and its §4 wiring site). Their run-time halves at Expansion
//     are milestone 5's and 6's.
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
}

// TestMilestoneOneErrorCodes_EveryMemberHasAFailingFixture walks every golden
// file in the check corpus and confirms each of the thirty-six error_code
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
