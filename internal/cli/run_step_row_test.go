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
//
// One row is `skip-if-recorded`'s and is here rather than in the corpus because
// its content is a **size** (§8, ADR-0056, issue #152): a Step that skipped five
// hundred renders `500` and not the dash, and a case seeding five hundred Assets
// to say so would be a corpus nobody runs. The mixed Step's `n` is not here —
// the row it answers is a *ran* Step's with a set, indistinguishable from the
// first case below — and what says the engine never writes `n of m` on one is
// [testdata/run/a-values-list-skips-two-and-calls-for-one], whose page reads `3`.

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
		"a Step whose every member skipped": {
			// *skipped as already recorded* carries a set — the
			// head versions the skip test read — so a Step that
			// made no call renders a **count** and not the dash,
			// and it may be larger than a neighbouring Step's that
			// did. A Step that skipped five hundred Assets did not
			// do nothing (§8, ADR-0056).
			step:    run.Step{Disposition: store.DispositionSkippedAsAlreadyRecorded, Records: 500, Concluded: true},
			cell:    "500",
			members: `"records":500`,
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

// **A rehearsal's withheld Step is on that Step's row** (§8, §9, ADR-0091,
// issue #206).
//
// The page has said so in prose since milestone 5 — *stopped at publish* under
// the table — and neither machine surface carried it: the withheld Step's row
// was the *never reached* row it shared with every Step behind it. The member
// is what closes that, and what it has to get right is the discrimination the
// page's sentence already makes, which is why the three answers below are
// three: the Step a rehearsal stopped at, the rehearsal that stopped at none,
// and the Run whose *never reached* rows are the world's rather than a
// rehearsal's.
//
// It is over `runRows` rather than `stepRowOf` because the operand is the
// Answer's, and over the bytes rather than the field because absence is half of
// what the member says: §7's absence rule makes carrying no key the
// discriminator, so a `false` written out would be every Step claiming to be a
// Step some rehearsal did not withhold.

// TestRunRows_TheWithheldStepIsTheOneRowThatSaysSo walks the three answers a
// `step` row can be written under and holds the member to the one Step of the
// one Run that has it.
func TestRunRows_TheWithheldStepIsTheOneRowThatSaysSo(t *testing.T) {
	const member = `"withheld":true`

	for name, c := range map[string]struct {
		answer run.Answer
		// on is the Step position whose row carries the member, and zero
		// where no row does.
		on int
	}{
		"a rehearsal that stopped at the first effect": {
			answer: run.Answer{
				Steps: []run.Step{
					{Position: 1, ID: "status", Kind: store.KindRead, Disposition: store.DispositionRan, Records: 1, Concluded: true},
					{Position: 2, ID: "publish", Kind: store.KindMutate, Disposition: store.DispositionNeverReached},
					{Position: 3, ID: "confirm", Kind: store.KindRead, Disposition: store.DispositionNeverReached},
				},
				Withheld: 2,
			},
			on: 2,
		},
		"a rehearsal that reached the end": {
			// Every Step of a read-only Procedure ran, so the
			// rehearsal withheld nothing and there is no position for
			// the member to be written at (§9, ADR-0010).
			answer: run.Answer{
				Steps: []run.Step{
					{Position: 1, ID: "status", Kind: store.KindRead, Disposition: store.DispositionRan, Records: 1, Concluded: true},
					{Position: 2, ID: "cert", Kind: store.KindRead, Disposition: store.DispositionRan, Records: 1, Concluded: true},
				},
			},
			on: 0,
		},
		"a Run the world resisted": {
			// The case the obvious inference gets wrong. A halt
			// leaves *never reached* rows exactly as a rehearsal
			// does, and a consumer that read the first of them as
			// the withheld Step would report the boundary of a
			// partial answer under a Run that failed (withheldStep).
			answer: run.Answer{
				Steps: []run.Step{
					{Position: 1, ID: "status", Kind: store.KindRead, Disposition: store.DispositionRan, Records: 1, Concluded: true},
					{Position: 2, ID: "publish", Kind: store.KindMutate, Disposition: store.DispositionAttemptedWorldUntouched},
					{Position: 3, ID: "confirm", Kind: store.KindRead, Disposition: store.DispositionNeverReached},
				},
			},
			on: 0,
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, row := range runRows(c.answer, t.TempDir()) {
				step, is := row.(stepRow)
				if !is {
					continue
				}
				encoded, err := json.Marshal(step)
				if err != nil {
					t.Fatal(err)
				}
				held := strings.Contains(string(encoded), member)
				if want := step.Step == c.on; held != want {
					t.Errorf("step %d %s %s, want it %s: %s", step.Step,
						map[bool]string{true: "carries", false: "omits"}[held], member,
						map[bool]string{true: "carried", false: "omitted"}[want], encoded)
				}
			}
		})
	}
}
