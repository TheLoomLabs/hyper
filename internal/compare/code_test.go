package compare_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// `THE CODE MOVED`'s rules, where two revisions of one artefact are the whole
// of the case (§8, §12, issue #171).
//
// The end-to-end table — the nine classes over a real repository at two real
// commits, the catch-all's arithmetic against `git diff` itself, and the two
// suppressions — is `internal/cli`'s, a `repo_revision` being a commit no case
// directory can name. What is here is what a golden could not state cleanly
// either way: a fact whose **bytes** moved and whose **value** did not, the
// order the rows come out in, and the four forms of the row that terminates the
// table.

// codeWindow is a window over two Runs of one Procedure, each having read one
// revision of the artefacts.
func codeWindow(baseline, subject store.RunProvenance) compare.Window {
	was := run("11", "sync-ci-keys", at(9, 0), at(9, 2))
	was.Provenance = baseline
	is := run("13", "sync-ci-keys", at(11, 0), at(11, 2))
	is.Provenance = subject
	return compare.Window{
		Procedure: "sync-ci-keys",
		Baseline:  compare.Side{Present: true, Entry: was},
		Subject:   compare.Side{Present: true, Entry: is},
	}
}

// provenance is one Run's Run-wide half, with the members a case does not care
// about held equal across the window so that only what it varies emits a row.
func provenance(repoRevision, procedureRevision string) store.RunProvenance {
	return store.RunProvenance{
		HyperVersion:      "1.4.0",
		ProcedureRevision: procedureRevision,
		RepoRevision:      repoRevision,
	}
}

// steady is the window every case below varies one thing against: two Runs with
// **nothing in the Provenance moved**, so that `the digests` draws no row and
// what the table holds is what the case put in the artefacts.
//
// The two ends name one revision because the Provenance is what a window's
// digest rows are read off; which revisions the *bytes* were read at is the
// Code value's own pair, and the catch-all's command comes off that (codeOf).
func steady() compare.Window {
	return codeWindow(provenance("1f0a3d7", "a91f0c2"), provenance("1f0a3d7", "a91f0c2"))
}

// procedureAt is one revision of a Procedure as the caller reads it: the source
// parsed, and the facts its own lines carry.
func procedureAt(t *testing.T, source string) compare.CodeArtefact {
	t.Helper()
	return artefactAt(t, artefact.KindProcedure, compare.SubjectProcedure, "sync-ci-keys", "procedures/sync-ci-keys.yaml", source)
}

// artefactAt is one revision of any artefact, read the way `internal/cli` reads
// it out of a revision's tree.
func artefactAt(t *testing.T, wire, subject, name, path, source string) compare.CodeArtefact {
	t.Helper()

	root, _, readable := yamlsubset.Parse(path, []byte(source))
	if !readable {
		t.Fatalf("the case's own %s will not parse:\n%s", path, source)
	}
	return compare.CodeArtefact{Kind: subject, Name: name, Path: path, Facts: artefact.ReadChangeFacts(wire, root)}
}

// codeOf is the two sides of one window's code, both revisions in the clone.
func codeOf(baseline, subject []compare.CodeArtefact) compare.Code {
	return compare.Code{
		Baseline: compare.CodeSide{Revision: "1f0a3d7", InClone: true, Artefacts: baseline},
		Subject:  compare.CodeSide{Revision: "88bc402", InClone: true, Artefacts: subject},
	}
}

// facts is the rows a window drew, less the catch-all, as `SUBJECT  FACT` and
// the two cells behind them — the page's own line with its padding collapsed,
// so that a case states what it means rather than a column width.
func facts(rows []render.Row) []string {
	var drawn []string
	for _, row := range rows {
		code, is := row.(compare.CodeRow)
		if !is || code.CatchAll() {
			continue
		}
		drawn = append(drawn, strings.Join(code.Cells(), " | "))
	}
	return drawn
}

// catchAll is the row that terminates the table.
func catchAll(t *testing.T, rows []render.Row) compare.CodeRow {
	t.Helper()

	for _, row := range rows {
		if code, is := row.(compare.CodeRow); is && code.CatchAll() {
			return code
		}
	}
	t.Fatal("the table carries no catch-all; §12 states it terminates the table and is not optional")
	return compare.CodeRow{}
}

// TestCodeRows_AFactThatDidNotMoveEmitsNoRowHoweverItsBytesMoved is the
// comparison being by the fact's own equality and never by the text.
//
// Reordering `targets: [staging, local]`, or reordering two conjuncts of one
// predicate selector, changes the file and changes nothing this table reports;
// those lines fall to the catch-all's count. That is §12's own argument for
// refusing a one-member `in:` — one filter with two spellings, "which would
// render as a change in `THE CODE MOVED` with nothing moved" — applied to the
// two second-spellings the format does not refuse (§8).
func TestCodeRows_AFactThatDidNotMoveEmitsNoRowHoweverItsBytesMoved(t *testing.T) {
	for _, moved := range []struct {
		name          string
		before, after string
		want          []string
	}{
		{
			name:   "a reordered Target set",
			before: "kind: procedure\nprocedure: sync-ci-keys\ntargets: [staging, local]\n",
			after:  "kind: procedure\nprocedure: sync-ci-keys\ntargets: [local, staging]\n",
		},
		{
			name: "two reordered conjuncts of one selector",
			before: "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: retire\n    over:\n      assets:\n" +
				"        - field: expires\n          older_than: 0s\n        - field: created\n          older_than: 30d\n",
			after: "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: retire\n    over:\n      assets:\n" +
				"        - field: created\n          older_than: 30d\n        - field: expires\n          older_than: 0s\n",
		},
		{
			name: "a reordered values: selector, whose order is the fact",
			before: "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: issue\n    over:\n" +
				"      values: [ci-arm64, ci-x86]\n",
			after: "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: issue\n    over:\n" +
				"      values: [ci-x86, ci-arm64]\n",
			want: []string{"procedure sync-ci-keys | step issue · over | values\nci-arm64 · ci-x86 | values\nci-x86 · ci-arm64"},
		},
	} {
		t.Run(moved.name, func(t *testing.T) {
			rows := compare.CodeRows(steady(), codeOf(
				[]compare.CodeArtefact{procedureAt(t, moved.before)},
				[]compare.CodeArtefact{procedureAt(t, moved.after)},
			))
			got := facts(rows)
			if len(got) != len(moved.want) {
				t.Fatalf("the table draws %v, want %v", got, moved.want)
			}
			for i, want := range moved.want {
				if got[i] != want {
					t.Errorf("the row is\n%q\nwant\n%q", got[i], want)
				}
			}
		})
	}
}

// TestCodeRows_ASideWithNothingRendersTheDashIncludingWhereTheFormatStatesAValueByOmission
// is the absence rule at the one place naming it would be a claim.
//
// An absent `bound:` is unbounded (§5), an absent `over:` a Step invoked once
// (§3), an absent `cadence:` no recurrence — and the cell still renders `–`.
// Naming what an absence means is a claim and not a value, and `FLAGS` is the
// one editorial surface in the tool: a reader of `step retire · bound  3  –`
// has already read that the Bound was removed (§8).
func TestCodeRows_ASideWithNothingRendersTheDashIncludingWhereTheFormatStatesAValueByOmission(t *testing.T) {
	before := "kind: procedure\nprocedure: sync-ci-keys\ncadence: \"0 0 1 * *\"\nsteps:\n  - id: retire\n    bound: 3\n    over:\n      values: [a]\n"
	after := "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: retire\n"

	rows := compare.CodeRows(steady(), codeOf(
		[]compare.CodeArtefact{procedureAt(t, before)},
		[]compare.CodeArtefact{procedureAt(t, after)},
	))
	want := []string{
		"procedure sync-ci-keys | cadence | 0 0 1 * *\n00:00 UTC on the 1st of the month\n1 run/month\n" +
			defaultBranchFact + "\n" + hourBoundaryFact + " | –",
		"procedure sync-ci-keys | step retire · bound | 3 | –",
		"procedure sync-ci-keys | step retire · over | values\na | –",
	}
	got := facts(rows)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the table draws\n%s\n\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestCodeRows_AnAbsentKeyAndAnEmptyListAreOneValue is the same rule read from
// the other end: for a set-shaped fact the two are one value and `–` is it, so
// a `destroy:` that went from absent to empty emits no row at all (§8).
func TestCodeRows_AnAbsentKeyAndAnEmptyListAreOneValue(t *testing.T) {
	definition := func(destroy string) []compare.CodeArtefact {
		return []compare.CodeArtefact{artefactAt(t, artefact.KindDefinition, compare.SubjectDefinition, "ci-keys", "definitions/ci-keys.yaml",
			"kind: definition\ndefinition: ci-keys\nkinds: [mutate]\n"+destroy)}
	}
	rows := compare.CodeRows(steady(), codeOf(definition(""), definition("destroy: []\n")))
	if got := facts(rows); len(got) != 0 {
		t.Errorf("the table draws %v, want no row: an absent key and an empty list are one value", got)
	}
}

// TestCodeRows_RowsSortBySubjectAndFactWithTheDashSubjectAfterEveryNamedOne is
// the order the page renders, which is a contract rather than a consequence:
// a row goes out on its own line and a consumer cannot re-sort what it has
// already printed (§8, ADR-0026).
func TestCodeRows_RowsSortBySubjectAndFactWithTheDashSubjectAfterEveryNamedOne(t *testing.T) {
	window := codeWindow(provenance("1f0a3d7", "a91f0c2"), provenance("88bc402", "b0c94f1"))
	code := codeOf(
		[]compare.CodeArtefact{
			procedureAt(t, "kind: procedure\nprocedure: sync-ci-keys\ntargets: [staging]\n"),
			artefactAt(t, artefact.KindDefinition, compare.SubjectDefinition, "ci-keys", "definitions/ci-keys.yaml", "kind: definition\ndefinition: ci-keys\nkinds: [mutate]\n"),
		},
		[]compare.CodeArtefact{
			procedureAt(t, "kind: procedure\nprocedure: sync-ci-keys\ntargets: [staging, production]\n"),
			artefactAt(t, artefact.KindDefinition, compare.SubjectDefinition, "ci-keys", "definitions/ci-keys.yaml", "kind: definition\ndefinition: ci-keys\nkinds: [destroy, mutate]\n"),
		},
	)

	var order []string
	for _, row := range compare.CodeRows(window, code) {
		code, is := row.(compare.CodeRow)
		if !is {
			continue
		}
		if code.CatchAll() {
			order = append(order, "<catch-all>")
			continue
		}
		order = append(order, code.Cells()[0]+" / "+code.Cells()[1])
	}
	want := []string{
		"definition ci-keys / kinds",
		"procedure sync-ci-keys / procedure revision",
		"procedure sync-ci-keys / targets",
		"— / repository revision",
		"<catch-all>",
	}
	if strings.Join(order, "\n") != strings.Join(want, "\n") {
		t.Errorf("the rows come out\n%s\n\nwant\n%s", strings.Join(order, "\n"), strings.Join(want, "\n"))
	}
}

// TestCodeRows_TheCatchAllHasFourFormsAndTheTwoSuppressionsStack is the row
// that terminates the table.
//
// **Where either side recorded `repo_dirty` the command is suppressed**, `git
// diff <rev> <rev>` not reproducing what moved. **Where the clone does not
// contain a revision the window names** the count is replaced by the line
// naming what could not be read, which keeps the command. The two are about
// different halves of the row and reach it for different reasons, so they stack
// (§8, §12).
func TestCodeRows_TheCatchAllHasFourFormsAndTheTwoSuppressionsStack(t *testing.T) {
	for _, form := range []struct {
		name           string
		dirty, inClone bool
		want           string
	}{
		{name: "the count and its command", inClone: true, want: "7 other lines changed · git diff 1f0a3d7 88bc402"},
		{name: "a dirty side suppresses the command", dirty: true, inClone: true, want: "7 other lines changed"},
		{name: "an absent revision replaces the count", want: "other lines could not be counted · git diff 1f0a3d7 88bc402"},
		{name: "both, stacked", dirty: true, want: "other lines could not be counted"},
	} {
		t.Run(form.name, func(t *testing.T) {
			code := compare.Code{
				Baseline: compare.CodeSide{Revision: "1f0a3d7", InClone: form.inClone, Dirty: form.dirty},
				Subject:  compare.CodeSide{Revision: "88bc402", InClone: form.inClone},
				Count:    7,
			}
			row := catchAll(t, compare.CodeRows(steady(), code))
			if got := row.Line(); got != form.want {
				t.Errorf("the catch-all reads %q, want %q", got, form.want)
			}
		})
	}
}

// TestCodeRows_TheCountSubtractsTheLinesEachRowsOwnValueOccupiesAndNoMore is
// the word *other* holding.
//
// A Manifest gaining an Operation moves the whole block that Operation is
// written as, and the `operations` row reports one name: **the key line is
// subtracted, and the request, the projection and the declared Kind beneath it
// are not**, being reported by no row above. Subtracting the block would have
// the catch-all silently drop lines it exists to guarantee are never dropped
// (§8).
func TestCodeRows_TheCountSubtractsTheLinesEachRowsOwnValueOccupiesAndNoMore(t *testing.T) {
	const path = "providers/tailscale.yaml"
	before := "kind: provider\nprovider: tailscale\ncapabilities: [http]\noperations:\n  list_keys:\n    kind: read\n"
	after := "kind: provider\nprovider: tailscale\ncapabilities: [http]\noperations:\n  list_keys:\n    kind: read\n" +
		"  get_key:\n    kind: read\n    deadline: 10s\n"

	code := codeOf(
		[]compare.CodeArtefact{artefactAt(t, artefact.KindProvider, compare.SubjectManifest, "tailscale", path, before)},
		[]compare.CodeArtefact{artefactAt(t, artefact.KindProvider, compare.SubjectManifest, "tailscale", path, after)},
	)
	// git added three lines: the Operation's own key line and the two
	// beneath it. Only the first is the `operations` row's own value.
	code.Count = 3
	code.Subject.Moved = map[string]map[int]bool{path: {7: true, 8: true, 9: true}}

	row := catchAll(t, compare.CodeRows(steady(), code))
	if got, want := row.Line(), "2 other lines changed · git diff 1f0a3d7 88bc402"; got != want {
		t.Errorf("the catch-all reads %q, want %q: the key line is subtracted and the two beneath it are not", got, want)
	}
}

// TestCodeRows_OneLineCarryingTwoFactsIsSubtractedOnce is the same arithmetic
// in the other direction: a flow sequence puts a key and its members on one line,
// and subtracting it twice would have the count report fewer lines than moved.
func TestCodeRows_OneLineCarryingTwoFactsIsSubtractedOnce(t *testing.T) {
	const path = "definitions/ci-keys.yaml"
	definition := func(kinds, targets string) []compare.CodeArtefact {
		return []compare.CodeArtefact{artefactAt(t, artefact.KindDefinition, compare.SubjectDefinition, "ci-keys", path,
			"kind: definition\ndefinition: ci-keys\nkinds: ["+kinds+"]\ntargets: ["+targets+"]\n")}
	}
	code := codeOf(definition("mutate", "staging"), definition("destroy, mutate", "production, staging"))
	// Four lines moved: the two keys at each of the two revisions, and both
	// facts are written across their own line only.
	code.Count = 4
	code.Baseline.Moved = map[string]map[int]bool{path: {3: true, 4: true}}
	code.Subject.Moved = map[string]map[int]bool{path: {3: true, 4: true}}

	row := catchAll(t, compare.CodeRows(steady(), code))
	if got, want := row.Line(), "0 other lines changed · git diff 1f0a3d7 88bc402"; got != want {
		t.Errorf("the catch-all reads %q, want %q", got, want)
	}
}

// TestCodeMovedPhrase_HasThreeFormsTestedInOrder is `TOTALS`' last segment.
//
// A surviving classed row is positive proof where the absence line is proof of
// nothing either way, so a window in which a fact moved *and* a revision could
// not be read is reported by the fact. What the ordering removes is the one
// reading this table may never produce: the negative asserted over bytes nobody
// read (§8).
func TestCodeMovedPhrase_HasThreeFormsTestedInOrder(t *testing.T) {
	classed := compare.CodeRow{SubjectKind: compare.SubjectProcedure, Subject: "sync-ci-keys", Fact: compare.FactProcedureRevision}
	counted := 0
	for _, form := range []struct {
		name string
		rows []compare.CodeRow
		want string
	}{
		{"a classed row rendered", []compare.CodeRow{classed, {Fact: compare.FactOtherLines, Count: &counted}}, "the code moved"},
		{"a classed row beside the absence line", []compare.CodeRow{classed, {Fact: compare.FactOtherLines, BaselineAbsent: compare.NotInClone}}, "the code moved"},
		{"the absence line alone", []compare.CodeRow{{Fact: compare.FactOtherLines, BaselineAbsent: compare.NotInClone}}, "the code could not be fully read"},
		{"neither", []compare.CodeRow{{Fact: compare.FactOtherLines, Count: &counted}}, "the code did not move"},
	} {
		t.Run(form.name, func(t *testing.T) {
			if got := compare.CodeMovedPhrase(form.rows); got != form.want {
				t.Errorf("the phrase is %q, want %q", got, form.want)
			}
		})
	}
}

// TestCodeRows_AWindowWithNoBaselineDrawsNothing is the first Run of a
// Procedure: there is no earlier revision for code to have moved from, no pair
// for `git diff` to name, and nothing for a count to count.
func TestCodeRows_AWindowWithNoBaselineDrawsNothing(t *testing.T) {
	subject := run("13", "sync-ci-keys", at(11, 0), at(11, 2))
	subject.Provenance = provenance("88bc402", "b0c94f1")
	window := compare.Window{Procedure: "sync-ci-keys", Subject: compare.Side{Present: true, Entry: subject}}

	if rows := compare.CodeRows(window, compare.Code{}); len(rows) != 0 {
		t.Errorf("the table draws %d rows over a window with no baseline, want none", len(rows))
	}
}

// TestCodeRows_NoSelectorRendersChanged is ADR-0059's rule not reaching this
// column, and not being extensible to it: it disqualifies anything nested, and
// a selector is nested by construction, so the extension renders every selector
// `changed` — the one word this column can never carry (§8).
func TestCodeRows_NoSelectorRendersChanged(t *testing.T) {
	before := "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: retire\n    over:\n      assets:\n        - field: expires\n          older_than: 0s\n"
	after := "kind: procedure\nprocedure: sync-ci-keys\nsteps:\n  - id: retire\n    over:\n      observations:\n        - field: status\n          equals: \"503\"\n"

	rows := compare.CodeRows(steady(), codeOf(
		[]compare.CodeArtefact{procedureAt(t, before)},
		[]compare.CodeArtefact{procedureAt(t, after)},
	))
	want := "procedure sync-ci-keys | step retire · over | assets\nexpires older_than 0s | observations\nstatus equals 503"
	got := facts(rows)
	if len(got) != 1 || got[0] != want {
		t.Errorf("the table draws\n%q\n\nwant\n%q", strings.Join(got, "\n"), want)
	}
}

// §10's two facts, copied byte for byte from issue #175 — the specification
// that writes the two sentences out, §10 itself stating them as prose.
const (
	defaultBranchFact = "scheduled runs happen on the default branch only"
	hourBoundaryFact  = ":00 is the executor's busiest minute — delivery there is likeliest to be delayed or dropped"
)

// TestCodeRows_TheCadenceCellStacksTheTwoFactsUnderTheRate is §10's rule on the
// third surface that renders a gloss: a `cadence` cell carries the expression,
// the phrase, the rate and the two facts, stacked, and nothing in it is
// shortened — the column widens to the longest line it holds (§8).
//
// **The hour-boundary fact is a reading of one side's minute field**, so a
// Cadence moving off the hour renders it in the `FROM` cell and not in the `TO`
// one, which is the difference this row exists to show. The default-branch fact
// stands in both, being a fact about every Cadence there is.
func TestCodeRows_TheCadenceCellStacksTheTwoFactsUnderTheRate(t *testing.T) {
	procedure := func(cadence string) []compare.CodeArtefact {
		return []compare.CodeArtefact{procedureAt(t,
			"kind: procedure\nprocedure: sync-ci-keys\ncadence: \""+cadence+"\"\n")}
	}
	rows := compare.CodeRows(steady(), codeOf(procedure("0 0 1 * *"), procedure("30 4 * * *")))
	want := []string{"procedure sync-ci-keys | cadence | " +
		strings.Join([]string{"0 0 1 * *", "00:00 UTC on the 1st of the month", "1 run/month", defaultBranchFact, hourBoundaryFact}, "\n") +
		" | " +
		strings.Join([]string{"30 4 * * *", "04:30 UTC every day", "≈30 runs/month", defaultBranchFact}, "\n")}
	if got := facts(rows); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the row is\n%s\n\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestCodeRows_TheCadenceRowsWireCarriesNoFact is the closure §9 states: the
// `code` row is closed at the gloss's three parts, and both facts are derived
// from `cadence` and `phrase`, which the row already carries. A consumer
// derives them exactly as the page does (§8, §10).
func TestCodeRows_TheCadenceRowsWireCarriesNoFact(t *testing.T) {
	procedure := func(cadence string) []compare.CodeArtefact {
		return []compare.CodeArtefact{procedureAt(t,
			"kind: procedure\nprocedure: sync-ci-keys\ncadence: \""+cadence+"\"\n")}
	}
	rows := compare.CodeRows(steady(), codeOf(procedure("0 0 1 * *"), procedure("*/5 * * * *")))
	for _, row := range rows {
		wire, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(wire), defaultBranchFact) || strings.Contains(string(wire), hourBoundaryFact) {
			t.Errorf("the wire carried a fact the page renders: %s", wire)
		}
	}
}
