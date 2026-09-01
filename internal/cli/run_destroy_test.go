package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **The serial `destroy`, watched on the wire, and the two Tombstones that read
// alike** (§6, §7, ADR-0010, ADR-0011, issue #150).
//
// Three of the claims this milestone lands are ones a golden cannot hold, for
// [run_concurrency_test.go](run_concurrency_test.go)'s reason: two of them are
// about **when** the calls happened and one is about two *cases* holding the
// same bytes, and a branch and a page record neither.
//
// So the wire is watched for the first two, through the same watch the
// concurrency cases use, and the third is driven as what it is: the same
// Procedure over the same seeded branch under the same Run id, answered `204`
// in one case and `404` in the other, with the Record files held against each
// other byte for byte.

// TestDestroy_AnEffectfulExpansionIsSerial is §6's rule with no artefact
// anywhere claiming otherwise: a `destroy` Expansion of five stands **one**
// connection at a time.
//
// A `concurrency:` limit is a `read`'s — `check` refuses one declared on any
// other Kind (§4, ADR-0045) — so there is no Manifest key to vary here and the
// claim is that the number is not consulted at all. The environment is loaded
// with every name an implementer might reach for, so that *no override* is an
// observation rather than a claim.
func TestDestroy_AnEffectfulExpansionIsSerial(t *testing.T) {
	t.Setenv("HYPER_CONCURRENCY", "5")
	t.Setenv("HYPER_MAX_IN_FLIGHT", "5")
	t.Setenv("CONCURRENCY", "5")

	// A width of two, so a second connection standing beside the first
	// would be seen. None does, and each connection pays the hold out.
	_, _, watch := drive(t, "a-destroy-expansion-is-serial", 2, 30*time.Millisecond, nil)

	if watch.peak != 1 {
		t.Errorf("%d connections stood together on a destroy Expansion, want 1 — a mutate or destroy Expansion runs strictly serially", watch.peak)
	}
	if len(watch.arrived) != 5 {
		t.Errorf("%d connections were made, want 5 — one per member of the Expansion", len(watch.arrived))
	}
}

// TestDestroy_AHaltStopsAtTheFirstErrorAndCallsNoMore is the other half of the
// same sentence, and the half serial dispatch buys: an effectful Expansion
// stops at the first error rather than draining, so the member after the one
// that answered `500` is never called at all.
//
// The branch says what was confirmed and `expanded_to` says what was resolved;
// what neither can say is that no fifth request went out, and that is what this
// reads. Its counterpart is `a-member-that-reaches-the-deadline`, where a
// `read` Expansion drains and **every** member is attempted (§6, drain.go).
func TestDestroy_AHaltStopsAtTheFirstErrorAndCallsNoMore(t *testing.T) {
	exit, _, watch := driven(t, "a-destroy-halted-at-the-fourth-of-five")

	if exit != cli.ExitProblems {
		t.Fatalf("exit = %d, want %d — the world resisted at the fourth member", exit, cli.ExitProblems)
	}
	if len(watch.arrived) != 4 {
		t.Errorf("%d connections were made, want 4 — the fifth member of a halted effectful Expansion is never called", len(watch.arrived))
	}
	if watch.peak != 1 {
		t.Errorf("%d connections stood together, want 1", watch.peak)
	}
}

// TestDestroy_ATombstoneOnA404ReadsAsOneOnA204 is ADR-0010's line drawn at the
// one place it is visible: what `hyper` is accountable for is that the thing is
// gone, and recording *already gone* as a fact about the Asset would be the
// reconciliation it declined to build.
//
// The two cases are the same Procedure over the same seeded branch under the
// same Run id and the same clock, so every byte either Record file holds is a
// byte the other holds — unless something in the version says how `hyper`
// learned the Asset was gone, which is exactly what may not be there. The
// status that confirmed it lives on the Step file under `answered`, and the
// second half of this reads that it is there and that it is the **only**
// difference between the two branches.
func TestDestroy_ATombstoneOnA404ReadsAsOneOnA204(t *testing.T) {
	confirmed := branchOf(t, "a-destroy-step-tombstones-an-asset")
	absent := branchOf(t, "a-destroy-answered-a-404-is-still-gone")

	const version = "records/cloudflare-prod/preview-dns/preview-42.example.com/01991e21-3c9f-7b04-9d18-5c7e2a94f083-0001.json"
	on204, on404 := fileIn(t, confirmed, version), fileIn(t, absent, version)
	if on204 != on404 {
		t.Errorf("the Tombstone written on a 404 differs from the one written on a 204:\n got:  %q\n want: %q", on404, on204)
	}
	if !strings.Contains(on204, `"tombstone": true`) {
		t.Errorf("the version this case is about is not a Tombstone:\n%s", on204)
	}

	// And the difference is where §7 puts it, and nowhere else: the Step
	// file's `answered`, which is a fact about the Run rather than about the
	// thing.
	const step = "journal/2026/04/02/01991e21-3c9f-7b04-9d18-5c7e2a94f083/steps/0001.json"
	ran, told := fileIn(t, confirmed, step), fileIn(t, absent, step)
	if strings.Contains(ran, `"answered"`) {
		t.Errorf("a destroy answered 2xx carries `answered`:\n%s", ran)
	}
	if !strings.Contains(told, `"status": 404`) {
		t.Errorf("a destroy answered 404 does not carry the status that confirmed it:\n%s", told)
	}
	if withoutAnswered(told) != ran {
		t.Errorf("the two Step files differ in more than `answered`:\n got:  %q\n want: %q", withoutAnswered(told), ran)
	}
}

// TestDestroy_AVersionAboveATombstoneReadsAliveAgain is §7's *terminal for the
// Asset's life and not for the series*, read off the branch the corpus case
// left: three versions of one series, the Tombstone in the middle, and the
// Head the one above it.
//
// The Head is derived and never a marker anybody wrote (§7, ADR-0011), so what
// this reads is the ordering: the last version of the series is the `mutate`'s,
// and it carries no `tombstone` key at all.
func TestDestroy_AVersionAboveATombstoneReadsAliveAgain(t *testing.T) {
	branch := branchOf(t, "a-destroy-then-a-create-reads-alive-again")

	const series = "records/cloudflare-prod/preview-dns/preview-42.example.com/"
	versions := versionsIn(branch, series)
	if len(versions) != 3 {
		t.Fatalf("the series holds %d versions, want 3 — the Asset an earlier Run created, its Tombstone, and the version above it:\n%v", len(versions), versions)
	}
	if head := fileIn(t, branch, versions[len(versions)-1]); strings.Contains(head, `"tombstone"`) {
		t.Errorf("the Head of the series reads dead after a further version was written above the Tombstone:\n%s", head)
	}
	if buried := fileIn(t, branch, versions[len(versions)-2]); !strings.Contains(buried, `"tombstone": true`) {
		t.Errorf("the version beneath the Head is not the Tombstone:\n%s", buried)
	}
}

// driven runs one corpus case to its end and answers all three things the cases
// above read: the exit code, the branch the Run left, and the watch its
// connections passed through.
//
// It tolerates the exit code, which is what separates it from `drive` beside it
// in [run_concurrency_test.go](run_concurrency_test.go): every case there
// completes, and one of the cases here is about a Run the world resisted.
//
// The watch stands a width of two, so a second connection standing beside the
// first would be seen, and a hold short enough that members which never stand
// together cost the suite milliseconds rather than seconds.
func driven(t *testing.T, name string) (int, string, *dialWatch) {
	t.Helper()

	c := corpusCase(t, "run/"+name)
	invocation := c.invocation(t)
	process := c.process(t, invocation)

	dial, watch := watching(process.Dial, 2, 30*time.Millisecond)
	process.Dial = dial

	var stdout, stderr bytes.Buffer
	exit := cli.Main(invocation.args, &stdout, &stderr, process, c.facts(t))
	return exit, invocation.fixture.render(t, invocation.fixture.root), watch
}

// branchOf is the branch one corpus case left, which is what two cases are held
// against each other by: the files inside them rather than their goldens.
func branchOf(t *testing.T, name string) string {
	t.Helper()

	_, branch, _ := driven(t, name)
	return branch
}

// fileIn is one file's content out of a rendered branch, named by its Store
// path. A path the branch does not hold fails the case rather than answering
// the empty string, which would compare equal to another absence.
func fileIn(t *testing.T, branch, path string) string {
	t.Helper()

	for _, section := range strings.Split(branch, "\n=== ") {
		name, content, found := strings.Cut(strings.TrimPrefix(section, "=== "), "\n")
		if !found {
			continue
		}
		if held, _, _ := strings.Cut(name, " ("); held == path {
			return content
		}
	}
	t.Fatalf("the branch holds no %s:\n%s", path, branch)
	return ""
}

// versionsIn is the paths a rendered branch holds under one prefix, in the
// order the rendering lists them — which is the branch's own path order, and
// for one series is the order its versions were written in.
func versionsIn(branch, prefix string) []string {
	var held []string
	for _, line := range strings.Split(branch, "\n") {
		if !strings.HasPrefix(line, "=== "+prefix) {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(line, "=== "), " (")
		held = append(held, name)
	}
	return held
}

// withoutAnswered is a Step file with its `answered` list taken out, which is
// what the two cases above are held against each other with. The list sits at
// the top, `answered` sorting before every other member a Step file carries,
// and it is one entry per member of the Expansion that was not answered the
// ordinary way (§7, ADR-0126).
func withoutAnswered(step string) string {
	opening := strings.Index(step, `  "answered": [`)
	if opening < 0 {
		return step
	}
	closing := strings.Index(step[opening:], "\n  ],\n")
	if closing < 0 {
		return step
	}
	return step[:opening] + step[opening+closing+len("\n  ],\n"):]
}
