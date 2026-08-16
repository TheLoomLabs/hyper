package problem

import "testing"

func TestSort_OrdersByFileThenLineThenColumnThenErrorCode(t *testing.T) {
	problems := []Problem{
		{File: "b.yaml", Line: 1, Column: 1, ErrorCode: "z"},
		{File: "a.yaml", Line: 2, Column: 1, ErrorCode: "z"},
		{File: "a.yaml", Line: 1, Column: 5, ErrorCode: "z"},
		{File: "a.yaml", Line: 1, Column: 1, ErrorCode: "z"},
		{File: "a.yaml", Line: 1, Column: 1, ErrorCode: "a"},
	}

	Sort(problems)

	want := []Problem{
		{File: "a.yaml", Line: 1, Column: 1, ErrorCode: "a"},
		{File: "a.yaml", Line: 1, Column: 1, ErrorCode: "z"},
		{File: "a.yaml", Line: 1, Column: 5, ErrorCode: "z"},
		{File: "a.yaml", Line: 2, Column: 1, ErrorCode: "z"},
		{File: "b.yaml", Line: 1, Column: 1, ErrorCode: "z"},
	}

	if len(problems) != len(want) {
		t.Fatalf("Sort() produced %d problems, want %d", len(problems), len(want))
	}
	for i := range want {
		if problems[i] != want[i] {
			t.Errorf("problems[%d] = %+v, want %+v", i, problems[i], want[i])
		}
	}
}
