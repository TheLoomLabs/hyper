package cli_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// flagsOf is the `FLAGS` block a review rendered, read off the page: its
// caption to the end of the rendering, each line with the screen's own indent
// taken off.
//
// It finds the block by its caption, exactly as authorityOf finds the table
// above it — and unlike that one it never answers nothing: the block renders on
// all five artefacts, an absent one being ambiguous between *nothing to flag*
// and *the renderer had nothing to say* (§12).
func flagsOf(page string) []string {
	lines := strings.Split(strings.TrimSuffix(page, "\n"), "\n")
	for n, line := range lines {
		if !strings.HasPrefix(strings.TrimLeft(line, " "), "FLAGS") {
			continue
		}
		block := make([]string, 0, len(lines)-n)
		for _, rest := range lines[n:] {
			block = append(block, strings.TrimPrefix(rest, "  "))
		}
		return block
	}
	return nil
}

// TestRunReview_EveryProcedureRendersTheEnvelopeFlag is the one row a Procedure
// is guaranteed, and the one flag name with an all-clear form: a review does
// not run `check`, so the flag has two states rather than one and both render
// (§12, ADR-0054).
func TestRunReview_EveryProcedureRendersTheEnvelopeFlag(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: look
    definition: things-observed
    operation: delete_everything
    target: staging
`)

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review", exit, stderr)
	}
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"ENVELOPE  line 3  ok  no step reaches a target outside [staging]",
	}
	if got := flagsOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_AnEnvelopeTheStepsExceedRendersItsOtherStateAndExitsZero is the
// second of the two states, and the exit code beside it: a review does not run
// `check`, so an artefact carrying `envelope-exceeded` renders like any other
// and exits 0 however many flags it carried (§9, §12).
func TestRunReview_AnEnvelopeTheStepsExceedRendersItsOtherStateAndExitsZero(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: []
steps:
  - id: retire
    definition: things
    operation: end_thing
    target: staging
    bound: 5
`)

	stdout, stderr, exit := runReview(t, root, "subject")
	if exit != 0 || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q, want a clean review — a flag indexes a fault without becoming one", exit, stderr)
	}
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"DESTROY   line 5  step retire  end_thing, bound 5",
		"ENVELOPE  line 3  exceeded     a step reaches outside []",
	}
	if got := flagsOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_DestroyReadsOnEveryArtefactWhoseGutterMarksIt is the first
// standing name, and the whole of what makes the five names five rather than
// five per artefact: `destroy` authority claimed, granted or exercised, on
// whichever line of whichever artefact carries it (§12).
func TestRunReview_DestroyReadsOnEveryArtefactWhoseGutterMarksIt(t *testing.T) {
	root := rosterRepo(t)
	// The Definition claims `destroy` the way §3 fixes: by naming an
	// Operation, `destroy` being no member of `kinds:` (§3, ADR-0069).
	writeFile(t, root+"/definitions/things.yaml",
		"kind: definition\ndefinition: things\nprovider: things\nkinds: [mutate]\ndestroy: [end_thing]\ntargets: [staging]\n")

	for _, c := range []struct{ named, want string }{
		{"subject", "DESTROY   line 5  step retire  end_thing, bound 5"},
		{"definitions/things.yaml", "DESTROY  line 5  destroy claimed for end_thing"},
		{"staging", "DESTROY  line 4  destroy accepted"},
		{"providers/things.yaml", "DESTROY  line 27  end_thing declares destroy"},
	} {
		stdout, stderr, exit := runReview(t, root, c.named)
		if exit != 0 || stderr != "" {
			t.Fatalf("exit = %d, stderr = %q reviewing %s, want a clean review", exit, stderr, c.named)
		}
		if got := flagsOf(stdout); !slices.Contains(got, c.want) {
			t.Errorf("the block on %s reads\n%q\nwant a row %q", c.named, got, c.want)
		}
	}
}

// TestRunReview_OpaqueReadsOnTheStepTheOperationAndTheOptIn is the second, on
// the three lines §12 names: a Step invoking an Opaque Operation, the Manifest
// Operation whose request uses an Opaque Capability, and the opt-in by which a
// Target declaration admits an `opaque` `destroy` at all (§4, §12).
func TestRunReview_OpaqueReadsOnTheStepTheOperationAndTheOptIn(t *testing.T) {
	root := opaqueDestroyRepo(t)

	for _, c := range []struct{ named, want string }{
		{"subject", "OPAQUE     line 5  step scrub  destroy reaches an effect hyper cannot describe"},
		{"local", "OPAQUE   line 6  an opaque destroy admitted"},
		{"shell", "OPAQUE   line 7   read reaches an effect hyper cannot describe"},
	} {
		stdout, stderr, exit := runReview(t, root, c.named)
		if exit != 0 || stderr != "" {
			t.Fatalf("exit = %d, stderr = %q reviewing %s, want a clean review", exit, stderr, c.named)
		}
		if got := flagsOf(stdout); !slices.Contains(got, c.want) {
			t.Errorf("the block on %s reads\n%q\nwant a row %q", c.named, got, c.want)
		}
	}
}

// opaqueDestroyRepo is a repository whose one Step is the strongest Step the
// tool runs: an `opaque` `destroy`, against a Target declaration that admits
// one, through the Provider `hyper` ships (§5, ADR-0039, ADR-0053).
func opaqueDestroyRepo(t *testing.T) string {
	t.Helper()
	root := newRepo(t)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [destroy]\ncapabilities: [shell]\nopaque-destroy: true\n")
	writeFile(t, root+"/definitions/commands.yaml",
		"kind: definition\ndefinition: commands\nprovider: shell\ndestroy: [destroy]\ntargets: [local]\n")
	writeFile(t, root+"/procedures/subject.yaml", `kind: procedure
procedure: subject
targets: [local]
steps:
  - id: scrub
    definition: commands
    operation: destroy
    target: local
    over:
      assets:
        - field: stdout
          starts_with: preview-
    args:
      command: [rm, -rf, /srv/preview]
`)
	return root
}

// TestRunReview_UnboundedReadsOnAMutateStepCarryingNoBound is §8's own row, on
// the one mark whose absence no static check reports: a `mutate` Step with no
// `bound:` is `mutate!` in the gutter, and this is the surface that puts it on
// one screen with everything else worth looking at (§4, §8).
func TestRunReview_UnboundedReadsOnAMutateStepCarryingNoBound(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: make
    definition: things
    operation: make_thing
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := "UNBOUNDED  line 5  step make  mutate with no declared bound"
	if got := flagsOf(stdout); !slices.Contains(got, want) {
		t.Errorf("the block reads\n%q\nwant a row %q", got, want)
	}
}

// TestRunReview_UnboundedReadsOnAnOpaqueDestroyStepRegardless is the row §5
// argues for outright: it is implied by `destroy` and `opaque` together on
// every such Step, having no other form to take, and it renders anyway — a
// surface indexing what is unbounded and silent on the one Step where nothing
// can be bounded is omitting rather than economising (§5, §12).
func TestRunReview_UnboundedReadsOnAnOpaqueDestroyStepRegardless(t *testing.T) {
	root := opaqueDestroyRepo(t)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"DESTROY    line 5  step scrub  destroy",
		"OPAQUE     line 5  step scrub  destroy reaches an effect hyper cannot describe",
		"UNBOUNDED  line 5  step scrub  an opaque destroy takes no bound",
		"ENVELOPE   line 3  ok          no step reaches a target outside [local]",
	}
	if got := flagsOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_UnboundedReadsOnAnOpaqueMutateWhateverItDeclares is the third
// form, and the one an author can otherwise edit away without touching what it
// was about: an `opaque` `mutate` takes a Bound — `check` accepts it, a Bound
// counts Records and the Step mints one — and the count says nothing about what
// the command did, which is §4's own argument on the Kind below `destroy`. The
// two renderings are the same row (§5, §12, issue #241).
func TestRunReview_UnboundedReadsOnAnOpaqueMutateWhateverItDeclares(t *testing.T) {
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"OPAQUE     line 5  step grant  mutate reaches an effect hyper cannot describe",
		"UNBOUNDED  line 5  step grant  a bound on an opaque mutate counts records, not what the commands did",
		"ENVELOPE   line 3  ok          no step reaches a target outside [local]",
	}
	for _, bound := range []bool{false, true} {
		stdout, _, exit := runReview(t, opaqueMutateRepo(t, bound), "subject")
		if exit != 0 {
			t.Fatalf("exit = %d, want a clean review", exit)
		}
		if got := flagsOf(stdout); !slices.Equal(got, want) {
			t.Errorf("the block on a step carrying a bound=%v reads\n%q\nwant\n%q", bound, got, want)
		}
	}
}

// opaqueMutateRepo is the Step issue #241 was found on, command and all: it
// appends two firewall rules and truncates a file, mints the one Record a
// `bound: 1` would count, and the count is what the flag beside it is about.
// The Bound is written where bound is true and left out where it is false,
// which are the two artefacts the sealed run produced either side of the edit.
func opaqueMutateRepo(t *testing.T, bound bool) string {
	t.Helper()
	root := newRepo(t)
	writeFile(t, root+"/targets/local.yaml",
		"kind: target-declaration\ntarget: local\nclass: local\nkinds: [mutate]\ncapabilities: [shell]\n")
	writeFile(t, root+"/definitions/commands.yaml",
		"kind: definition\ndefinition: commands\nprovider: shell\nkinds: [mutate]\ntargets: [local]\n")
	declared := ""
	if bound {
		declared = "    bound: 1\n"
	}
	writeFile(t, root+"/procedures/subject.yaml", `kind: procedure
procedure: subject
targets: [local]
steps:
  - id: grant
    definition: commands
    operation: mutate
    target: local
    args:
      command: [sh, -c, 'cat requests/pending >> firewall/allow && : > requests/pending']
`+declared)
	return root
}

// TestRunReview_UnboundedAndUnresolvedRenderOnNoArtefactButAProcedure is the
// two Procedure-only names, and their reasons are two: `bound:` is a Step's
// key, and a Definition names a Provider too — but nothing on a Definition's
// screen is derived from a Manifest, so an absent one costs that rendering
// nothing and there is no name there for the gutter to fail to follow (§8,
// §12, ADR-0064).
func TestRunReview_UnboundedAndUnresolvedRenderOnNoArtefactButAProcedure(t *testing.T) {
	root := rosterRepo(t)
	writeFile(t, root+"/definitions/orphan.yaml",
		"kind: definition\ndefinition: orphan\nprovider: not-installed\nkinds: [read]\ntargets: [staging]\n")

	for _, named := range []string{"definitions/things.yaml", "orphan", "staging", "providers/things.yaml", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named)
		if exit != 0 {
			t.Fatalf("exit = %d reviewing %s, want a clean review", exit, named)
		}
		for _, absent := range []string{"UNBOUNDED", "UNRESOLVED"} {
			if block := strings.Join(flagsOf(stdout), "\n"); strings.Contains(block, absent) {
				t.Errorf("the block on %s reads\n%s\nwant no %s row — the name is a Procedure's", named, block, absent)
			}
		}
	}

	// The Definition naming a Provider that is not there draws no flag at
	// all, which is the same absence read from the other side: its gutter
	// marks nothing about that name, so there is nothing to index.
	stdout, _, _ := runReview(t, root, "orphan")
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"no line the gutter marked draws a flag",
	}
	if got := flagsOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block on the Definition reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_UnresolvedNamesWhichNameFailedAndThePathHyperLookedFor is what
// separates this row from its marker: the gutter marks one word for four
// absences because it marks and does not classify, and this is the surface that
// says which (§8, §12).
//
// The four are the four a Step can carry: its own `definition:` and
// `operation:`, the Provider the Definition it found names, and a nested
// invocation's `procedure:`.
func TestRunReview_UnresolvedNamesWhichNameFailedAndThePathHyperLookedFor(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: no-such-definition
    definition: not-there
    operation: list_things
    target: staging
  - id: no-such-operation
    definition: things
    operation: not_there
    target: staging
  - id: no-such-provider
    definition: orphan
    operation: list_things
    target: staging
  - id: no-such-procedure
    procedure: not-there
`)
	writeFile(t, root+"/definitions/orphan.yaml",
		"kind: definition\ndefinition: orphan\nprovider: not-installed\nkinds: [read]\ntargets: [staging]\n")

	stdout, _, exit := runReview(t, root, "subject")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	for _, want := range []string{
		"definition: not-there — no definitions/not-there.yaml",
		"operation: not_there — no such Operation in providers/things.yaml",
		"provider: not-installed — no built-in Provider and no providers/not-installed.yaml",
		"procedure: not-there — no procedures/not-there.yaml",
	} {
		if block := strings.Join(flagsOf(stdout), "\n"); !strings.Contains(block, want) {
			t.Errorf("the block reads\n%s\nwant a row ending %q", block, want)
		}
	}
}

// TestRunReview_RowsRenderInLineOrderWithTheFileLevelRowLast is ADR-0054, and
// the fixture is written so that severity order and line order disagree: the
// `destroy` Step stands above the unbounded `mutate`, so a block that ranked
// its rows would put them the other way round.
//
// `ENVELOPE` is last wherever its cited line falls, which on a Procedure is
// always the top of the file: it is the summary of the rows above it, and a
// summary that arrives before its evidence reads as a verdict (ADR-0054).
func TestRunReview_RowsRenderInLineOrderWithTheFileLevelRowLast(t *testing.T) {
	root := reviewRepo(t, `kind: procedure
procedure: subject
targets: [staging]
steps:
  - id: retire
    definition: things
    operation: end_thing
    target: staging
    bound: 5
  - id: make
    definition: things
    operation: make_thing
    target: staging
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		"FLAGS   index into the gutter above — no flag states anything the gutter does not",
		"DESTROY    line 5   step retire  end_thing, bound 5",
		"UNBOUNDED  line 10  step make    mutate with no declared bound",
		"ENVELOPE   line 3   ok           no step reaches a target outside [staging]",
	}
	if got := flagsOf(stdout); !slices.Equal(got, want) {
		t.Errorf("the block reads\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_EveryRowCitesALineTheGutterMarked is the relation the whole
// surface rests on, asserted on the artefact carrying every mark there is: a
// flag citing a line the gutter did not mark is a defect in the renderer rather
// than a rendering (§8, ADR-0026).
func TestRunReview_EveryRowCitesALineTheGutterMarked(t *testing.T) {
	root := opaqueDestroyRepo(t)

	stdout, _, exit := runReview(t, root, "subject", "--json")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}

	marked := map[float64]bool{}
	var cited []float64
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("%q is not one JSON object: %v", line, err)
		}
		switch row["type"] {
		case "gutter":
			marked[row["line"].(float64)] = true
		case "flag":
			cited = append(cited, row["cites_line"].(float64))
		}
	}
	if len(cited) == 0 {
		t.Fatal("the stream carries no flag row; the relation held vacuously")
	}
	for _, line := range cited {
		if !marked[line] {
			t.Errorf("a flag cites line %v, which no gutter row marked:\n%s", line, stdout)
		}
	}
}

// TestRunReview_TheWireCarriesOneFlagRowPerRenderedRow is §12's decomposition:
// the name in kebab-case — the closed set's own spelling, where the page
// renders it upper-case for the eye — the line it cites, and the coordinate
// where the flag has one.
//
// The coordinate is `step` on a Procedure and there is no other today, which is
// why the four rows below carry one and a Manifest's carry none.
func TestRunReview_TheWireCarriesOneFlagRowPerRenderedRow(t *testing.T) {
	root := opaqueDestroyRepo(t)

	stdout, _, exit := runReview(t, root, "subject", "--json")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	want := []string{
		`{"type":"flag","flag":"destroy","cites_line":5,"step":"scrub"}`,
		`{"type":"flag","flag":"opaque","cites_line":5,"step":"scrub"}`,
		`{"type":"flag","flag":"unbounded","cites_line":5,"step":"scrub"}`,
		`{"type":"flag","flag":"envelope","cites_line":3}`,
	}
	var got []string
	for _, line := range strings.Split(strings.TrimSuffix(stdout, "\n"), "\n") {
		if strings.HasPrefix(line, `{"type":"flag"`) {
			got = append(got, line)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("the flag rows are\n%q\nwant\n%q", got, want)
	}
}

// TestRunReview_TheThreeChangeNamesAreNotImplemented is what this ticket does
// not carry. All three read a **marked** line as well as the baseline, and the
// change column that marks one lands with them (issue #168) — so a range being
// open (issue #164) leaves nothing yet for a direction to index (§8, §12,
// ADR-0057).
func TestRunReview_TheThreeChangeNamesAreNotImplemented(t *testing.T) {
	root := rosterRepo(t)

	for _, named := range []string{"subject", "definitions/things.yaml", "staging", "providers/things.yaml", "hyper.yaml"} {
		stdout, _, exit := runReview(t, root, named, "--json")
		if exit != 0 {
			t.Fatalf("exit = %d reviewing %s, want a clean review", exit, named)
		}
		for _, absent := range []string{"widened", "narrowed", "changed"} {
			if strings.Contains(stdout, absent) {
				t.Errorf("the stream on %s carries %q:\n%s", named, absent, stdout)
			}
		}
	}
}

// TestRunReview_ADestroyRowNamesNoBoundOnAnOpaqueStep is §5's argument held on
// this surface: a Bound counts the Records an Expansion resolved to, and a
// count of the calls an opaque Step made says nothing about what any of them
// did — so the only Bound such a Step could carry "would stand in the gutter
// and in `FLAGS` reading *at most one thing will be destroyed* while `rm -rf /`
// is magnitude one".
//
// Writing one is `bound-illegal` and `check`'s to report. What this holds is
// that the review does not repeat the refused claim while the `unbounded` row
// directly beneath it says the opposite.
func TestRunReview_ADestroyRowNamesNoBoundOnAnOpaqueStep(t *testing.T) {
	root := opaqueDestroyRepo(t)
	writeFile(t, root+"/procedures/subject.yaml", `kind: procedure
procedure: subject
targets: [local]
steps:
  - id: scrub
    definition: commands
    operation: destroy
    target: local
    over:
      assets:
        - field: stdout
          starts_with: preview-
    args:
      command: [rm, -rf, /srv/preview]
    bound: 5
`)

	stdout, _, exit := runReview(t, root, "subject")
	if exit != 0 {
		t.Fatalf("exit = %d, want a clean review", exit)
	}
	block := flagsOf(stdout)
	if !slices.Contains(block, "DESTROY    line 5  step scrub  destroy") {
		t.Errorf("the block reads\n%q\nwant a DESTROY row naming the Operation alone", block)
	}
	if strings.Contains(strings.Join(block, "\n"), "bound 5") {
		t.Errorf("the block reads\n%q\nwant no Bound named on an opaque destroy", block)
	}
}
