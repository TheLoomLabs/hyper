package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **The `RECORDS` column takes one of three forms, and the `step` row carries
// the counts they are read from** (§8, ADR-0030, ADR-0062, issue #140).
//
// The corpus drives each form end to end — `2` on the tracer bullet, `0` on the
// Expansion that resolved to nothing, the dash on a refused Step, `2 of 3` on
// the Expansion that drained — and what is here is the three of them side by
// side, over the value the engine answers rather than over a page. A cell form
// is decided by two members of one small value, and the way to see that the
// dash and the zero are different answers is to put them in one table.

// TestStepRow_TheThreeCellForms walks §8's cell forms and the row members each
// one writes.
func TestStepRow_TheThreeCellForms(t *testing.T) {
	for name, c := range map[string]struct {
		step run.Step
		cell string
		// members is the JSON the row writes for the two counts, so that
		// *absent* is told from *zero* by the bytes rather than by a
		// pointer nobody looked at.
		members string
	}{
		"the set is all the Step reached": {
			step:    run.Step{Disposition: store.DispositionRan, Records: 2, Concluded: true},
			cell:    "2",
			members: `"records":2`,
		},
		"an Expansion that resolved to nothing": {
			// A *ran* Step whose set is written empty. It is `0` and
			// never the dash: the Step looked and there was nothing
			// there, which is a conclusion (§8, ADR-0030).
			step:    run.Step{Disposition: store.DispositionRan, Records: 0, Concluded: true},
			cell:    "0",
			members: `"records":0`,
		},
		"a Step that stopped short of its Expansion": {
			step:    run.Step{Disposition: store.DispositionRan, Records: 2, Concluded: true, Expanded: 3},
			cell:    "2 of 3",
			members: `"records":2,"expanded":3`,
		},
		"a Step that concluded about none of its Expansion": {
			// The drained Step whose every member faulted, and the
			// Step carrying no `over:` that halted on its one call:
			// expanded to one, accounted for none.
			step:    run.Step{Disposition: store.DispositionRan, Records: 0, Concluded: true, Expanded: 1},
			cell:    "0 of 1",
			members: `"records":0,"expanded":1`,
		},
		"no set exists at all": {
			// *refused* concluded about nothing, by construction
			// rather than by circumstance, so no number is written
			// and the dash is what tells it from the zero above.
			step:    run.Step{Disposition: store.DispositionRefused},
			cell:    noRecords,
			members: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := stepRowOf(c.step)

			if got := row.Cells()[4]; got != c.cell {
				t.Errorf("the RECORDS cell reads %q, want %q", got, c.cell)
			}

			encoded, err := json.Marshal(row)
			if err != nil {
				t.Fatal(err)
			}
			for _, member := range []string{`"records"`, `"expanded"`} {
				held := strings.Contains(string(encoded), member)
				if held != strings.Contains(c.members, member) {
					t.Errorf("the row %s %s: %s", map[bool]string{true: "carries", false: "omits"}[held], member, encoded)
				}
			}
			if c.members != "" && !strings.Contains(string(encoded), c.members) {
				t.Errorf("the row reads %s, want it to carry %s", encoded, c.members)
			}
		})
	}
}
