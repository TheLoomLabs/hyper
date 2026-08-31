package compare_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/compare"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The rows one window answers (§8, issue #167): the `window` row, with the two
// Record tables beneath it in records_test.go and `THE CODE MOVED` in
// code_test.go.

// wire is one row as the --json stream carries it.
func wire(t *testing.T, row render.Row) string {
	t.Helper()

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(row); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestRows_TheWindowRowCarriesBothSidesWhole(t *testing.T) {
	subject := run("13", "watch", at(11, 0), at(11, 2))
	subject.Provenance.ProcedureRevision = "b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe"
	subject.Trigger = store.Trigger{Cause: store.CauseCron, Executor: store.ExecutorLocal, Actor: "igor", Host: "thinkpad"}

	baseline := run("11", "watch", at(9, 0), at(9, 2))
	baseline.Provenance.ProcedureRevision = "a91f0c2d5b83e47196c0af2b1d7e63840f5a92c1"

	rows := compare.Rows(compare.Window{
		Procedure: "watch",
		Baseline:  compare.Side{Present: true, Entry: baseline},
		Subject:   compare.Side{Present: true, Entry: subject},
	}, nil, compare.Code{})

	want := `{"type":"window","procedure":"watch",` +
		`"baseline":{"run":"01992011-0000-7000-8000-000000000000","trigger":"igor@thinkpad","started":"2026-08-06T09:00:00.000Z","dry_run":false,"outcome":"completed","ended":"2026-08-06T09:02:00.000Z","procedure_revision":"a91f0c2d5b83e47196c0af2b1d7e63840f5a92c1"},` +
		`"subject":{"run":"01992013-0000-7000-8000-000000000000","trigger":"cron","started":"2026-08-06T11:00:00.000Z","dry_run":false,"outcome":"completed","ended":"2026-08-06T11:02:00.000Z","procedure_revision":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe"}}` + "\n"
	if got := wire(t, rows[0]); got != want {
		t.Errorf("the window row is\n%s\nwant\n%s", got, want)
	}
}

func TestRows_ADirtySideCarriesRepoDirtyAndNeverThePagesSuffix(t *testing.T) {
	subject := run("13", "watch", at(11, 0), at(11, 2))
	subject.Provenance.ProcedureRevision = "b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe"
	subject.Provenance.RepoDirty = true

	rows := compare.Rows(compare.Window{Procedure: "watch", Subject: compare.Side{Present: true, Entry: subject}}, nil, compare.Code{})
	got := wire(t, rows[0])
	if !bytes.Contains([]byte(got), []byte(`"procedure_revision":"b0c94f1e73a852d6b4f09c318e2a70d5c86b41fe","repo_dirty":true`)) {
		t.Errorf("the window row is\n%s\nwant repo_dirty beside the revision it qualifies, and the revision whole", got)
	}
	if bytes.Contains([]byte(got), []byte("+")) {
		t.Errorf("the window row is\n%s\nwant no + suffix; that is the page's notation for the same fact", got)
	}
}

func TestRows_NoBaselineWritesNoBaselineMember(t *testing.T) {
	rows := compare.Rows(compare.Window{
		Procedure: "watch",
		Subject:   compare.Side{Present: true, Entry: run("11", "watch", at(9, 0), at(9, 2))},
	}, nil, compare.Code{})
	if got := wire(t, rows[0]); bytes.Contains([]byte(got), []byte("baseline")) {
		t.Errorf("the window row is\n%s\nwant no baseline member at all where there is no baseline", got)
	}
}

func TestRows_AReapedSideCarriesItsClosersAndNoEnd(t *testing.T) {
	entry := run("11", "watch", at(9, 0), at(9, 2))
	entry.Owner = store.OutcomeFile{}
	entry.Closers = []store.Closer{{Run: runID("09"), ClosedBy: store.ClosedBy{EndedAt: at(23, 0), Step: 3}}}

	rows := compare.Rows(compare.Window{
		Procedure: "watch",
		Subject:   compare.Side{Present: true, Entry: entry, Steps: []store.StepFile{{Step: 1, EndedAt: at(9, 1)}}},
	}, nil, compare.Code{})
	got := wire(t, rows[0])
	if !bytes.Contains([]byte(got), []byte(`"outcome":"failed","procedure_revision"`)) {
		t.Errorf("the window row is\n%s\nwant no ended member between them: no duration derives, which is what the page renders reaped for", got)
	}
	want := `"closed_by":[{"run":"01992009-0000-7000-8000-000000000000","outcome":"failed","step":3,"ended":"2026-08-06T23:00:00.000Z"}]`
	if !bytes.Contains([]byte(got), []byte(want)) {
		t.Errorf("the window row is\n%s\nwant %s", got, want)
	}
	if !bytes.Contains([]byte(got), []byte(`"outcome":"failed"`)) {
		t.Errorf("the window row is\n%s\nwant the entry's own outcome, which a closing write fixes at failed (§7)", got)
	}
}

func TestRows_AContestedSideCarriesTheOwnersOutcomeAndItsClosersBeside(t *testing.T) {
	entry := run("11", "watch", at(9, 0), at(9, 2))
	entry.Closers = []store.Closer{{Run: runID("09"), ClosedBy: store.ClosedBy{EndedAt: at(23, 0), Step: 3}}}

	rows := compare.Rows(compare.Window{Procedure: "watch", Subject: compare.Side{Present: true, Entry: entry}}, nil, compare.Code{})
	got := wire(t, rows[0])
	if !bytes.Contains([]byte(got), []byte(`"outcome":"completed","ended":"2026-08-06T09:02:00.000Z"`)) {
		t.Errorf("the window row is\n%s\nwant the owner's outcome and end, unqualified", got)
	}
	if !bytes.Contains([]byte(got), []byte(`"closed_by":`)) {
		t.Errorf("the window row is\n%s\nwant the contest beside them rather than inside the outcome", got)
	}
}

func TestWindowRow_HasNoLineOfItsOwnInATable(t *testing.T) {
	rows := compare.Rows(compare.Window{Procedure: "watch", Subject: compare.Side{Present: true, Entry: run("11", "watch", at(9, 0), at(9, 2))}}, nil, compare.Code{})
	if cells := rows[0].Cells(); len(cells) != 0 {
		t.Errorf("Cells() = %v, want none: the header is a block and not a row of a table of like rows", cells)
	}
}

func TestRows_ARehearsalSubjectCarriesDryRunAndAnOrdinaryRunCarriesItFalse(t *testing.T) {
	// §7's one exception to the absence rule, on the surface that names two
	// Runs: a reader that takes the marker's absence for `false` is the
	// reader that pays, and `--subject` is what puts a rehearsal on this
	// wire at all (§7, §8, ADR-0114).
	rehearsal := run("13", "watch", at(11, 0), at(11, 2))
	rehearsal.DryRun = true

	rows := compare.Rows(compare.Window{
		Procedure: "watch",
		Baseline:  compare.Side{Present: true, Entry: run("11", "watch", at(9, 0), at(9, 2))},
		Subject:   compare.Side{Present: true, Entry: rehearsal},
	}, nil, compare.Code{})

	got := wire(t, rows[0])
	if !bytes.Contains([]byte(got), []byte(`"started":"2026-08-06T11:00:00.000Z","dry_run":true`)) {
		t.Errorf("the window row is\n%s\nwant dry_run true on the subject", got)
	}
	if !bytes.Contains([]byte(got), []byte(`"started":"2026-08-06T09:00:00.000Z","dry_run":false`)) {
		t.Errorf("the window row is\n%s\nwant dry_run written on the baseline too, the bare false included", got)
	}
}
