package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **A Step that concluded about a Record without calling for it wrote no secret
// for it, and both surfaces say so** (§8, §9, ADR-0150, issue #273).
//
// The absence in the sink is correct — nothing was produced, so nothing was
// lost — and it was legible to nobody: the operator asked for values on disk,
// got part of a tree, and the honest cause was three artefacts away. What is
// here is the pair of renderings that end that, held over the shapes a corpus
// case cannot reach cheaply: the wholly skipped Step beside the mixed one, the
// singular beside the plural, and two such Steps in one Run.
//
// The page's *placement* is not here and is the corpus's — a block's position
// between the table and the terminal line is what a golden holds — and neither
// is the engine's count, which is `perform`'s and is driven end to end.

// TestSecretsSkipped_TheRowCarriesTheCountAndTheAbsenceIsZero holds §7's absence
// rule on the member: a count where the sink is short, and no key at all where
// it is not.
//
// Zero needs no pointer here and `records` does, which is the distinction the
// two members are worth reading side by side for. A `records` of zero is a Step
// that looked and concluded about nothing, so the key has to be written; a
// `secrets_skipped` of zero is *nothing is missing*, which is exactly what
// carrying no key means.
func TestSecretsSkipped_TheRowCarriesTheCountAndTheAbsenceIsZero(t *testing.T) {
	const member = `"secrets_skipped"`

	for name, c := range map[string]struct {
		step run.Step
		want string
	}{
		"a Step whose every member skipped": {
			step: run.Step{
				Position: 1, ID: "mint", Kind: store.KindMutate,
				Disposition: store.DispositionSkippedAsAlreadyRecorded,
				Records:     3, Concluded: true, SecretsSkipped: 3,
			},
			want: `"secrets_skipped":3`,
		},
		"a mixed Step": {
			// The harder of the two to read, and the reason the
			// member is not gated on the Disposition: this row says
			// `ran` and a count of three, which is what the Step
			// that wrote every one of its values says too.
			step: run.Step{
				Position: 2, ID: "mint", Kind: store.KindMutate,
				Disposition: store.DispositionRan,
				Records:     3, Concluded: true, SecretsSkipped: 2,
			},
			want: `"secrets_skipped":2`,
		},
		"a Step that skipped nothing": {
			step: run.Step{
				Position: 1, ID: "mint", Kind: store.KindMutate,
				Disposition: store.DispositionRan, Records: 3, Concluded: true,
			},
			want: "",
		},
		"a Step whose Operation declares no secret at all": {
			// The engine writes the count on a secret-producing
			// Step and on no other, so this row is every skipping
			// Step in every Procedure that produces no secret.
			step: run.Step{
				Position: 1, ID: "publish", Kind: store.KindMutate,
				Disposition: store.DispositionSkippedAsAlreadyRecorded,
				Records:     3, Concluded: true,
			},
			want: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(stepRowOf(c.step))
			if err != nil {
				t.Fatal(err)
			}
			held := strings.Contains(string(encoded), member)
			if want := c.want != ""; held != want {
				t.Errorf("the row %s %s, want it %s: %s",
					map[bool]string{true: "carries", false: "omits"}[held], member,
					map[bool]string{true: "carried", false: "omitted"}[want], encoded)
			}
			if c.want != "" && !strings.Contains(string(encoded), c.want) {
				t.Errorf("the row reads %s, want it to carry %s", encoded, c.want)
			}
		})
	}
}

// TestSecretsSkipped_ThePageNamesEveryStepAndTheReasonOnce holds what the
// sentence beneath the table says, and the one thing about it that is a
// decision rather than a rendering: the reason is the Run's and is written
// once, however many Steps carry the fact.
func TestSecretsSkipped_ThePageNamesEveryStepAndTheReasonOnce(t *testing.T) {
	for name, c := range map[string]struct {
		steps []run.Step
		want  []string
	}{
		"one Step, one Record": {
			steps: []run.Step{{Position: 1, Disposition: store.DispositionSkippedAsAlreadyRecorded, Records: 1, Concluded: true, SecretsSkipped: 1}},
			want: []string{
				"step 1 skipped 1 record and wrote no secret for it.",
				secretsSkippedReason,
			},
		},
		"one Step, several Records": {
			steps: []run.Step{{Position: 2, Disposition: store.DispositionRan, Records: 3, Concluded: true, SecretsSkipped: 2}},
			want: []string{
				"step 2 skipped 2 records and wrote no secret for them.",
				secretsSkippedReason,
			},
		},
		"two Steps of one Run": {
			// `check` permits two secret-producing Steps in one
			// Procedure (ADR-0148), so two of them skipping is a
			// Run that exists. Each Step's count is its own and the
			// reason is the same sentence about the same
			// Repeatability value.
			steps: []run.Step{
				{Position: 1, Disposition: store.DispositionSkippedAsAlreadyRecorded, Records: 1, Concluded: true, SecretsSkipped: 1},
				{Position: 2, Disposition: store.DispositionRan, Records: 4, Concluded: true, SecretsSkipped: 3},
			},
			want: []string{
				"step 1 skipped 1 record and wrote no secret for it.",
				"step 2 skipped 3 records and wrote no secret for them.",
				secretsSkippedReason,
			},
		},
		"a Run whose sink is complete": {
			steps: []run.Step{{Position: 1, Disposition: store.DispositionRan, Records: 3, Concluded: true}},
			want:  nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			rows := make([]render.Row, 0, len(c.steps))
			for _, step := range c.steps {
				rows = append(rows, stepRowOf(step))
			}
			got := secretsSkipped(rows)
			if len(got) != len(c.want) {
				t.Fatalf("the page writes %d lines, want %d: %q", len(got), len(c.want), got)
			}
			for i, line := range c.want {
				if got[i] != line {
					t.Errorf("line %d reads %q, want %q", i+1, got[i], line)
				}
			}
		})
	}
}
