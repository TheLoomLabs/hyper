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

// milestoneOneErrorCodes is the closed set issue #87 fixes for that
// milestone: the thirty-seven error_code members milestone 1 brings to `hyper
// check`. It was every code reaching `check` until milestone 9 added
// one, and it is milestone 1's contribution rather than the whole set now —
// what `check` owes a fixture for is checkCorpusErrorCodes below. It is:
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
	"destroy-unscoped",
	"skip-if-recorded-unreachable",
	"bound-exceeded",
	"predicate-type-mismatch",
	"record-identity-collision",
	"cadence-run-once",
	"cadence-secret-output",
	"procedure-cycle",
}

// milestoneNineCheckCodes is what milestone 9 adds to the same set: §10's two
// static Cadence codes that are not the transitive walk's.
//
// `cadence-malformed` is here rather than in the list above because the grammar
// was stated and unenforced until the milestone that projects a Cadence into a
// workflow — an expression outside it loaded clean and rendered no gloss, which
// is the one surface built to show a reviewer the blast radius of a recurrence
// saying nothing about the recurrence it could not read (§10, §12, issue #174).
//
// `projection-stale` is §10's other static code and the verification half of
// generate-and-verify: `hyper project` writes the file (issue #177) and this
// compares what stands against a fresh regeneration (issue #179). It is the one
// member of the whole set whose fixture cites a file that is **not** an
// artefact, which is what testdata/check/projection-stale is for — one code,
// three shapes, each citing a path in the namespace `project` owns.
var milestoneNineCheckCodes = []string{
	"cadence-malformed",
	"projection-stale",
}

// milestoneTenErrorCodes is what milestone 10 adds: §11's Extension codes,
// each one a thing a Manifest in providers/ may never be.
//
// `provider-name-collision` is a Manifest taking a built-in Provider's name. It
// is here rather than in the milestone-1 list because it is the one code in the
// set whose subject is a **namespace** rather than an artefact — *this name is
// taken* is a fact no single file's checks can see — and because until this
// milestone the fold answered it the one way §11 forbids: the later Manifest
// won, and the built-in's Operations left the namespace with no row to say so
// (§11, §12, issue #185).
//
// `capability-reserved` is the other thing an Extension may never be: a
// Manifest in providers/ holding the Capability reserved to the Providers hyper
// ships. It is here rather than in the milestone-1 list because milestone 1
// enforced the Capability check a Manifest can pass — declared against derived —
// and this is the one it cannot: a Manifest may agree with itself exactly,
// declare `shell`, carry `shell:` request blocks, and exec argv on the machine
// hyper runs on, and until this milestone `check` reported no problems found
// (§11, §12, ADR-0004, issue #186).
//
// The two are one milestone's list and not two because they close one sentence:
// what an Extension may never be. They are also closed by one criterion — hyper
// ships a Provider only where the Capability it needs is one nobody else may
// declare (ADR-0039) — so the roster this list's first member reads and the
// reserved set its second reads grow together or not at all.
//
// `origin-digest-mismatch` is the third, and it closes a different sentence:
// not what an Extension may never be, but what an installed one must still be —
// the bytes `install` verified against the digest recorded beside them. It is
// the one member of this list `check` shares with another command, `install`
// answering the same code over the same fact at the moment of the fetch; what
// this half adds is that the verification is **repeatable offline, by anyone
// reading the repository**, which is the claim §11 rests the whole distribution
// mechanism on and which nothing enforced until this milestone (§11, §12,
// issue #189).
//
// `manifest-schema-unsupported` is the fourth, and it closes the sentence the
// three above leave open: what an Extension may never be and what an installed
// one must still be both presuppose a file this binary can read at all. A
// Manifest is the one artefact carrying an explicit schema version, being the
// one authored outside this repository's pin (ADR-0023), and until this
// milestone that integer was decoration — a Manifest declaring `schema-version:
// 7` was read as though it declared `1`, with its declared-equals-derived
// Capability check run against keys the reader could not see, which is the one
// guess §11 calls expensive by name (§11, §12, ADR-0028, issue #190).
//
// It is also the one member of this list whose remedy is **not an artefact
// edit**: *a hyper that reads this schema version — nothing in the repository is
// the fault*. So its Refusal renders a named remedy where the three above render
// an `EDIT ONE OF` table.
var milestoneTenErrorCodes = []string{
	"provider-name-collision",
	"capability-reserved",
	"origin-digest-mismatch",
	"manifest-schema-unsupported",
}

// checkCorpusErrorCodes is every error_code the check corpus owes a failing
// fixture: the three milestones' contributions, concatenated. They are three
// lists and one assertion because which milestone a code arrived with is worth
// recording and is not worth a second walk of the corpus.
func checkCorpusErrorCodes() []string {
	codes := append([]string{}, milestoneOneErrorCodes...)
	codes = append(codes, milestoneNineCheckCodes...)
	return append(codes, milestoneTenErrorCodes...)
}

// TestCheckCorpusErrorCodes_EveryMemberHasAFailingFixture walks every golden
// file in the check corpus and confirms each error_code member `check`
// contributes appears in at least one of them — the human table's ERROR_CODE
// column or the --json stream's "error_code" field, either rendering being fine
// since both come from one renderer (ADR-0026). It fails naming every code an
// uncovered member, so the closed set and the fixture set can never drift apart
// silently (issue #99).
func TestCheckCorpusErrorCodes_EveryMemberHasAFailingFixture(t *testing.T) {
	codes := checkCorpusErrorCodes()
	if missing := codesWithNoFixture(t, checkCorpus, codes); len(missing) > 0 {
		t.Errorf("%d of %d error_code members have no failing fixture under %s/: %v", len(missing), len(codes), checkCorpus, missing)
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
// The four that are not §4's ride here beside them because they reach a Run
// the same way and through the same rendering: the Store schema test's, the
// credential pass's two and the sink gate's (§6, §9, §12).
//
// The sink gate's is `secret-sink-absent`, which stood down for one release in
// favour of `secret-sink-unwritten` and is back. The two are one gate at two
// moments in its life: while nothing wrote the file, a Run reaching a Step that
// declares secret output Refused whether a sink was named or not, so the absent
// one was a state no binary could produce and a member here with no fixture is
// exactly what this list exists to catch. The writer separates the operands
// again, and what this binary cannot produce is now the other one (§9, §12,
// ADR-0146, ADR-0148).
//
// The credential pass contributes two because it reads one variable three ways,
// and the two halves are the two that decline: `credential-absent` where the
// environment does not hold the variable, `credential-empty` where it holds it
// and sets it to nothing. They are two codes rather than two messages under one
// (§12, ADR-0145), so a corpus holding one of them would leave the other's
// rendering — its own `=` remedy note, and its own line on `outcome.json` —
// driven by nothing (issue #264).
//
// bound-exceeded is the fourth that is not §4's, and it is here on a different
// footing from all of them: it is not a code arriving through the re-run of
// `check` but §6's own, an Expansion's count being a number no file holds
// (§5, §6, issue #149). Its offline half has a fixture under testdata/check/
// and a member in the list above; what this one holds is that the run-time
// half is driven too, and that both spell one code.
//
// run-once-recorded is the fifth, and it is §6's own with no offline half at
// all: what decides it is what the Journal holds for the Step, which is a fact
// no artefact in the repository states and `check` therefore cannot read (§6,
// §12, issue #153). It is a member here and in no list above for that reason.
//
// cadence-malformed is one of the two members milestone 9 adds, and it is here
// on the ordinary footing: §10's grammar is closed so that an expression no
// executor's clock could read never reaches one, and a rule that held for
// `check` and not for a Run would let the Run be the way past it (§6, issue
// #174).
//
// provider-name-collision is milestone 10's first arrival, and it is here on
// the same footing as the two above: what an Extension may never be is decided
// at load, and a rule that held for `check` and not for a Run would make the
// Run the way past it — a Definition reviewed against the built-in running
// against whatever took its name being exactly the failure §11 refuses (§6,
// §11, issue #185).
//
// capability-reserved is milestone 10's second, and it is here for the reason
// the three above are and one of its own: what a Run would otherwise reach is
// the Capability itself. A rule that held for `check` and not for a Run would
// leave the file that declares it one `hyper run` away from exec'ing argv on
// the machine — which is the guarantee §11 states in words and this milestone
// enforces (§6, §11, issue #186).
//
// origin-digest-mismatch is milestone 10's third, and it is here on the footing
// the two above it are: an installed Manifest whose bytes have moved since
// `install` verified them is a Manifest under review that is not the one that
// was reviewed, and a rule that held for `check` and not for a Run would make
// the Run the way past it — a Run against an Extension nobody's digest covers
// being exactly what §11's origin: block exists to prevent (§6, §11, issue
// #189).
//
// manifest-schema-unsupported is milestone 10's fourth, and it is here on the
// footing the three above it are and one of its own: what a Run would otherwise
// do is read a Manifest on a partial understanding of its own shape, which is
// the failure §11 calls expensive by name — the declared-equals-derived
// Capability check run against keys the reader could not see (§6, §11, issue
// #190). Its Refusal is also the second place §8's named-remedy rendering is
// asserted against an **artefact** citation — `a-credential-the-environment-
// does-not-hold` is the first — a caret on the `schema-version:` scalar, and a
// remedy that is a binary to install rather than an `EDIT ONE OF` table.
//
// projection-stale is the other, and it is the member this list was hardest to
// leave out: a Run started by a projected workflow is a Run started by a file
// that must be what the artefacts ask for, and a rule that held on a laptop and
// not on the runner would exempt the one occasion nobody is watching (§6, §10,
// issue #179). Its Refusal is also where the rendering §8 owes it is asserted —
// no caret, and a remedy that is a command rather than an `EDIT ONE OF` table.
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
	"credential-empty",
	"secret-sink-absent",
	"bound-exceeded",
	"run-once-recorded",
	"cadence-malformed",
	"projection-stale",
	"provider-name-collision",
	"capability-reserved",
	"origin-digest-mismatch",
	"manifest-schema-unsupported",
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
	if missing := codesWithNoFixture(t, runCorpus, codesReachingARun); len(missing) > 0 {
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

// codesWithNoFixture is the question all four assertions in this file ask, and
// it is one function because the fourth caller is where the same eight lines
// copied four times stops being a coincidence: which members of a closed set
// have no fixture in one corpus, named, sorted, so that the closed set and the
// fixture set can never drift apart silently.
//
// What each caller keeps is its own sentence about what a missing fixture
// means — `check` owes a **failing** one and the other three owe a **Refusing**
// one, which is a real difference between reporting problems and declining
// before any effect — and the walk beneath them is the same walk (§12).
//
// A code is matched bounded on both sides by something other than a lowercase
// letter or a hyphen, so it is never credited by appearing as a substring of a
// longer, different one — every member of every set here is a distinct
// kebab-case token and none is a hyphen-delimited prefix or suffix of another.
func codesWithNoFixture(t *testing.T, corpus string, codes []string) []string {
	t.Helper()

	haystack := goldensUnder(t, corpus)

	var missing []string
	for _, code := range codes {
		pattern := `(^|[^a-z-])` + regexp.QuoteMeta(code) + `([^a-z-]|$)`
		if !regexp.MustCompile(pattern).Match(haystack) {
			missing = append(missing, code)
		}
	}
	sort.Strings(missing)
	return missing
}

// goldensUnder is every stdout and stderr golden of one corpus, concatenated —
// the one read every assertion above is made against.
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

// projectCorpus is `project`'s own slice of testdata/, and the third corpus this
// file asks the same question of.
const projectCorpus = "testdata/project"

// codesRefusedByProject is the closed set's members `hyper project` contributes:
// one, and it is distribution's rather than §4's.
//
// `release-artefact-absent` is `project` unable to resolve a published artefact
// for the version it is recording — no release under the tag, no checksums file
// beside it, or no line in that file for the artefact the compiled-in template
// names (§11, §12, issue #178). The three shapes are one code because the remedy
// for each is a released binary rather than an edit, and because two of them
// arrive as one answer: a tag with no release and a release with no checksums
// file are both a request for something that is not there.
//
// The pin gate's two codes are deliberately not members. `project` is the one
// command in §9's tree that calls no gate, for the reason RunProject states, so
// this corpus holds no Refusal under either — which is what the guard beside
// TestGolden asserts (§9, ADR-0020).
// They are spelled out rather than imported from the package that fires them,
// on this file's own footing: what every assertion here reads is what the
// checked-in goldens say, and a constant imported from the code under test would
// move with it.
var codesRefusedByProject = []string{
	"release-artefact-absent",
}

// TestCodesRefusedByProject_EveryMemberHasARefusingFixture holds that set
// against `project`'s corpus, on the footing the two tests above state: what is
// read is what the checked-in goldens say, and a code with no fixture is a claim
// nothing drives.
func TestCodesRefusedByProject_EveryMemberHasARefusingFixture(t *testing.T) {
	if missing := codesWithNoFixture(t, projectCorpus, codesRefusedByProject); len(missing) > 0 {
		t.Errorf("%d of %d codes have no Refusing fixture under %s/: %v", len(missing), len(codesRefusedByProject), projectCorpus, missing)
	}
}

// installCorpus is `install`'s own slice of testdata/, and the fourth corpus
// this file asks the same question of. It is also what install_test.go's own
// read of these cases' exit codes is scoped by — the corpus names are declared
// together here, so that *which directory is which command's* is answered in
// one place.
const installCorpus = "testdata/install"

// codesRefusedByInstall is the closed set's members `hyper install`
// contributes: one, and it is the only thing this command tells apart from
// every other way its two reads can fail to produce verified bytes.
//
// `origin-digest-mismatch` is bytes that arrived and are not the bytes the
// publisher published. It is a `77` rather than the `1` beside it because the
// read completed, the digest was published, and a verbatim retry declines
// identically — the remedy is the publisher's rather than another attempt
// (§11, §12, ADR-0060, issue #188).
//
// **The codes `install` does not carry are the assertion beside it.** A ref the
// registry does not hold, a checksums file naming every published file but this
// one, a rate limit, a bad gateway, a connection nothing accepted, a name that
// did not resolve, a handshake that did not complete and a body over the cap are
// each exit `1` with **no** `error_code` at all: §11 puts *matches nothing* and
// *the fetch did not complete* on one code deliberately, and a member here for
// any of them would be that paragraph reinvented. What holds that half is
// TestInstallCorpus_ItsWholeCodeSetIsThree, one file over.
//
// It is spelled out rather than imported from the package that fires it, on
// this file's own footing: what every assertion here reads is what the
// checked-in goldens say, and a constant imported from the code under test
// would move with it.
var codesRefusedByInstall = []string{
	"origin-digest-mismatch",
}

// TestCodesRefusedByInstall_EveryMemberHasARefusingFixture holds that set
// against `install`'s corpus, on the footing the three tests above state: what
// is read is what the checked-in goldens say, and a code with no fixture is a
// claim nothing drives.
func TestCodesRefusedByInstall_EveryMemberHasARefusingFixture(t *testing.T) {
	if missing := codesWithNoFixture(t, installCorpus, codesRefusedByInstall); len(missing) > 0 {
		t.Errorf("%d of %d codes have no Refusing fixture under %s/: %v", len(missing), len(codesRefusedByInstall), installCorpus, missing)
	}
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
