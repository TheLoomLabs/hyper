package cli_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// §10's two facts, copied byte for byte from issue #175 — the specification
// that writes the two sentences out, §10 itself stating them as prose. Every
// surface that renders a gloss renders these beside it, so they are written
// once here and every case in this file is held to the same two sentences.
const (
	defaultBranchFact = "scheduled runs happen on the default branch only"
	hourBoundaryFact  = ":00 is the executor's busiest minute — delivery there is likeliest to be delayed or dropped"
)

// TestRunReview_TheDefaultBranchFactRendersOnEveryCadence is the unconditional
// half on the surface it matters most on: a reviewer approving `*/5 * * * *` on
// a feature branch is approving a workflow that will not fire once, and the
// line that says so is the one beside the rate they read it from (§10).
func TestRunReview_TheDefaultBranchFactRendersOnEveryCadence(t *testing.T) {
	for _, cadence := range []string{"0 3 * * 1", "*/5 * * * *", "5 * * * *", "1-59 * * * *", "0 0 29 2 *"} {
		root := newRepo(t)
		writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring(cadence))

		stdout, _, exit := runReview(t, root, "nightly")
		if exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", cadence, exit)
		}
		header := headerOf(t, stdout)
		if len(header) != 2 {
			t.Fatalf("%s: the header is %q, want two lines", cadence, header)
		}
		if !strings.Contains(header[1], " · "+defaultBranchFact) {
			t.Errorf("%s glossed\n %q\nwant the default-branch fact beside it", cadence, header[1])
		}
	}
}

// TestRunReview_TheHourBoundaryFactRendersWhereTheMinuteSelectsZero is the
// conditional half, on the same line and read off the same field. What decides
// is whether the minute field selects `0` at all, and never how it was spelled.
func TestRunReview_TheHourBoundaryFactRendersWhereTheMinuteSelectsZero(t *testing.T) {
	for _, want := range []struct {
		cadence string
		lands   bool
	}{
		{"0 3 * * 1", true},
		{"0,30 * * * *", true},
		{"*/5 * * * *", true},
		{"* * * * *", true},
		{"0-29 * * * *", true},
		{"5 * * * *", false},
		{"1-59 * * * *", false},
		{"30 4 * * *", false},
	} {
		root := newRepo(t)
		writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring(want.cadence))

		stdout, _, exit := runReview(t, root, "nightly")
		if exit != cli.ExitClean {
			t.Fatalf("%s: exit = %d, want a clean review", want.cadence, exit)
		}
		header := headerOf(t, stdout)
		if len(header) != 2 {
			t.Fatalf("%s: the header is %q, want two lines", want.cadence, header)
		}
		if got := strings.Contains(header[1], hourBoundaryFact); got != want.lands {
			t.Errorf("%s carried the hour-boundary fact %v, want %v — the line was\n %q",
				want.cadence, got, want.lands, header[1])
		}
	}
}

// TestRunReview_TheFactsShareTheGlossesOwnLine holds where they go: a header's
// parts share one `·`-separated line, the expression already being on the
// `cadence:` line below (§10). They are not a second line, not a flag and not a
// gutter mark — an artefact with no Cadence has no line to hang them on at all.
func TestRunReview_TheFactsShareTheGlossesOwnLine(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring("0 3 * * 1"))
	writeFile(t, root+"/targets/local.yaml", "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read]\n")

	stdout, _, exit := runReview(t, root, "nightly")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	header := headerOf(t, stdout)
	if len(header) != 2 {
		t.Fatalf("the header is %q, want two lines", header)
	}
	want := "03:00 UTC every Monday · ≈4.3 runs/month · " + defaultBranchFact + " · " + hourBoundaryFact
	if header[1] != want {
		t.Errorf("the gloss line is\n %q\nwant\n %q", header[1], want)
	}

	// An artefact declaring no Cadence renders neither fact anywhere on
	// its page: the facts stand beside a gloss, and there is none.
	writeFile(t, root+"/procedures/by-hand.yaml", "kind: procedure\nprocedure: by-hand\ntargets: [local]\nsteps: []\n")
	stdout, _, exit = runReview(t, root, "by-hand")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if strings.Contains(stdout, defaultBranchFact) || strings.Contains(stdout, hourBoundaryFact) {
		t.Errorf("an artefact with no Cadence carried a fact:\n%s", stdout)
	}
}

// TestRunReview_TheFactsReachNoWireMember is the other half of *page-only*: §9
// closes the `artefact` row at the gloss's three parts, and both facts are
// derived from `cadence` and `phrase`, which the row already carries — so a
// consumer derives them exactly as the page does and the row does not widen
// (§8, §10).
func TestRunReview_TheFactsReachNoWireMember(t *testing.T) {
	root := newRepo(t)
	writeFile(t, root+"/procedures/nightly.yaml", procedureDeclaring("0 3 * * 1"))

	stdout, _, exit := runReview(t, root, "--json", "nightly")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	if strings.Contains(stdout, defaultBranchFact) || strings.Contains(stdout, hourBoundaryFact) {
		t.Errorf("the wire carried a fact the page renders:\n%s", stdout)
	}
	for _, part := range []string{`"cadence":"0 3 * * 1"`, `"phrase":"03:00 UTC every Monday"`, `"rate":4.3`} {
		if !strings.Contains(strings.ReplaceAll(stdout, ", ", ","), part) {
			t.Errorf("the row dropped %s:\n%s", part, stdout)
		}
	}
}

// TestReviewChanges_TheFactsStackUnderEachSidesRate is the second surface: the
// `FLAGS` row where a Cadence moved inside the range. In a table cell the parts
// stack, and the two facts stack under the rate of the side they belong to
// (§8, §10).
//
// **Each side carries its own pair**, which is what makes the hour-boundary
// fact legible across the edit: it is a reading of one expression's minute
// field, so a move onto or off the hour is a line appearing or disappearing
// beside the arrow. The default-branch fact is the same sentence on both sides
// and renders on both for the rule's sake — it stands beside a gloss wherever
// one renders, and a side rendering a gloss with nothing beside it would be the
// one place a reviewer reads a Cadence and learns less about it than they do
// one line above.
func TestReviewChanges_TheFactsStackUnderEachSidesRate(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "procedures/retire-preview-dns.yaml", `cadence: "0 3 * * 1"`, `cadence: "30 4 * * *"`)

	stdout, _, exit := r.review(t, "procedures/retire-preview-dns.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	block := flagsOf(stdout)
	for _, want := range []string{
		"03:00 UTC every Monday · ≈4.3 runs/month",
		defaultBranchFact,
		hourBoundaryFact,
		"→ 04:30 UTC every day · ≈30 runs/month",
		defaultBranchFact,
	} {
		if !slices.ContainsFunc(block, func(line string) bool { return strings.Contains(line, want) }) {
			t.Fatalf("the FLAGS block carries no %q:\n%s", want, strings.Join(block, "\n"))
		}
	}

	// The moved-off-the-hour side carries the default-branch fact and not
	// the hour-boundary one: `30 4 * * *` selects no `:00`, and the fact
	// disappearing beside the arrow is the edit being read.
	from, to := factsBesideEachSide(t, block)
	if !slices.Equal(from, []string{defaultBranchFact, hourBoundaryFact}) {
		t.Errorf("the side before the edit carries %q, want both facts", from)
	}
	if !slices.Equal(to, []string{defaultBranchFact}) {
		t.Errorf("the side after the edit carries %q, want the default-branch fact alone", to)
	}
}

// factsBesideEachSide reads the two sides of a stacked `cadence` row back off
// the page: the fact lines above the arrow, and the ones below it. The arrow is
// what says which side a stacked line belongs to (§8).
func factsBesideEachSide(t *testing.T, block []string) (from, to []string) {
	t.Helper()

	arrived := false
	for _, line := range block {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "→ ") {
			arrived = true
			continue
		}
		if trimmed != defaultBranchFact && trimmed != hourBoundaryFact {
			continue
		}
		if arrived {
			to = append(to, trimmed)
			continue
		}
		from = append(from, trimmed)
	}
	return from, to
}

// TestReviewChanges_TheFlagRowReachesNoWireMember holds the same closure one
// row type over: the `flag` row carries what it carried, and a consumer derives
// both facts from the expression it already has (§8, §10).
func TestReviewChanges_TheFlagRowReachesNoWireMember(t *testing.T) {
	r := demoRanged(t)
	r.journal(t, demoEntry(r.head))
	r.edit(t, "procedures/retire-preview-dns.yaml", `cadence: "0 3 * * 1"`, `cadence: "*/5 * * * *"`)

	stdout, _, exit := r.reviewJSON(t, "procedures/retire-preview-dns.yaml")
	if exit != cli.ExitClean {
		t.Fatalf("--json exited %d, want a clean review", exit)
	}
	if strings.Contains(stdout, defaultBranchFact) || strings.Contains(stdout, hourBoundaryFact) {
		t.Errorf("the wire carried a fact the page renders:\n%s", stdout)
	}
}
