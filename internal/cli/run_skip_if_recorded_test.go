package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **Under `skip-if-recorded` the two Dispositions are one set at two
// granularities** (§6, §7, §8, ADR-0056, issue #152).
//
// No golden can hold this: what it is about is what the *second* and *third*
// entries write given the first, and a case drives one Run. The claim needs
// three over one branch — a Step that skipped one member and called for two, a
// Step whose every member skipped, and the same again — so the arithmetic a
// reader does off the entries is the one this asserts.
//
// It drives the corpus case [testdata/run/three-runs-of-one-values-list] three
// times through one materialised repository, editing nothing between them. The
// artefact is fixed at three members and the **branch** is what moves: the
// seeded one holds the first standing and the second Tombstoned, so the first
// Run skips one member, creates the other two, and leaves a branch on which the
// two Runs after it can do nothing at all.
//
// What that shows is the digest: the identity set holds every member the skip
// test concluded about, so it is the same three every Run and the digest is
// byte-identical across all three — which is the behaviour it exists for, and
// the one that would be lost if the set shrank to whatever ran.

// TestSkipIfRecorded_TheDigestDoesNotMoveAsTheListFillsIn drives the three Runs
// and reads what each entry wrote.
func TestSkipIfRecorded_TheDigestDoesNotMoveAsTheListFillsIn(t *testing.T) {
	c := corpusCase(t, "run/three-runs-of-one-values-list")
	invocation := c.invocation(t)
	// One process for the three Runs, so that the mint answers the three
	// ids the case names in the order it names them — a fresh one per Run
	// would answer the first id three times and write one entry over
	// another.
	process := c.process(t, invocation)

	// The `values:` list as the artefact authors it, which is the order
	// `expanded_to` holds and never the order the set is written in.
	authored := []string{
		"docs.preview-42.example.com",
		"api.preview-42.example.com",
		"cdn.preview-42.example.com",
	}
	for run, expected := range []struct {
		disposition store.Disposition
		moved       bool
		// versions is how many Record versions stand on the branch once
		// this Run has ended: the two the case seeded, plus the two the
		// first Run created and the nothing the two after it did.
		versions int
	}{
		{store.DispositionRan, true, 4},
		{store.DispositionSkippedAsAlreadyRecorded, false, 4},
		{store.DispositionSkippedAsAlreadyRecorded, false, 4},
	} {
		var stdout, stderr bytes.Buffer
		if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
			t.Fatalf("run %d: exit = %d; stderr: %s", run+1, exit, stderr.String())
		}

		file := stepFileOf(t, c, invocation, run+1)
		if file.Disposition != expected.disposition {
			t.Errorf("run %d is %s, want %s", run+1, file.Disposition, expected.disposition)
		}
		// **The digest does not move.** The set is all three members
		// every Run, whether their calls went out or not, so the Step
		// that skipped everything carries the digest the Step that
		// called for two carried.
		identitySetWritten(t, run+1, file.Identities, store.Names(authored), expected.moved)
		// **Every member the selector resolved to is in `expanded_to`**,
		// whether its call went out or not: nothing is dropped for
		// standing, and the order is the authored one (§6, §7).
		if names := file.Selector.ExpandedTo; !equalNames(names, authored) {
			t.Errorf("run %d expanded to %v, want the authored %v", run+1, names, authored)
		}

		// **A Step that skipped made no call**, which the branch is
		// what says: a call that went out would have been answered —
		// the case serves the host — and the version it minted would
		// stand here.
		branch := invocation.fixture.render(t, invocation.fixture.root)
		if held := strings.Count(branch, "=== records/"); held != expected.versions {
			t.Errorf("run %d leaves %d Record versions on the branch, want %d", run+1, held, expected.versions)
		}
	}
}

// **`$.command` on a `shell` Operation resolves before the call, so the built-in
// `mutate_skip_if_recorded` performs its own test** (§4, ADR-0056, issue #152).
//
// It needs two Runs and so cannot be a golden either. The first creates both
// Assets and the second must skip both, which is the whole claim: the name the
// skip test reads the head under and the name the projection writes under are
// one string, spelled by capability.Command and by nothing else. Were they two
// spellings the test would read a series nothing ever writes to, every Run would
// be *ran*, and no single Run's page would say so.
func TestSkipIfRecorded_AShellCommandIsItsOwnIdentity(t *testing.T) {
	c := corpusCase(t, "run/a-shell-step-skips-the-command-it-recorded")
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	// The argv each member resolves to, spelled as §12's `command` member
	// is: the list JSON-encoded on one line, which is what makes `[mark, "a
	// b"]` and `[mark, a, b]` two identities rather than one.
	marked := []string{
		`["mark","/srv/app/releases/r41"]`,
		`["mark","/srv/app/releases/r42"]`,
	}

	for run, expected := range []struct {
		disposition store.Disposition
		moved       bool
		versions    int
	}{
		{store.DispositionRan, true, 2},
		{store.DispositionSkippedAsAlreadyRecorded, false, 2},
	} {
		var stdout, stderr bytes.Buffer
		if exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t)); exit != 0 {
			t.Fatalf("run %d: exit = %d; stderr: %s", run+1, exit, stderr.String())
		}

		file := stepFileOf(t, c, invocation, run+1)
		if file.Disposition != expected.disposition {
			t.Errorf("run %d is %s, want %s", run+1, file.Disposition, expected.disposition)
		}
		identitySetWritten(t, run+1, file.Identities, store.Names(marked), expected.moved)

		// What says no call went out is the **Disposition** above and
		// not this count: the command is deterministic, so a second
		// call would project what the head already holds and mint
		// nothing either way. The count is here for the other half —
		// that the first Run opened one series per member, under the
		// argv rather than under anything else.
		branch := invocation.fixture.render(t, invocation.fixture.root)
		if held := strings.Count(branch, "=== records/"); held != expected.versions {
			t.Errorf("run %d leaves %d Record versions on the branch, want %d", run+1, held, expected.versions)
		}
	}
}
