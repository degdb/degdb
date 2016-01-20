package protocol

import (
	"testing"

	"github.com/d4l3k/messagediff"
)

func TestCloneTriples(t *testing.T) {
	t.Parallel()

	triples := []*Triple{
		{
			Subj: "a",
		},
	}
	ntrips := CloneTriples(triples)
	ntrips[0].Subj = "b"
	if triples[0].Subj != "a" {
		t.Error("CloneTriples() failed to make a copy")
	}
}

func TestSortTriples(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a, want []*Triple
	}{
		{
			[]*Triple{
				{
					Subj: "b",
				},
				{
					Subj: "c",
				},
				{
					Subj: "a",
					Pred: "b",
				},
				{
					Subj: "a",
					Pred: "a",
				},
			},
			[]*Triple{
				{
					Subj: "a",
					Pred: "a",
				},
				{
					Subj: "a",
					Pred: "b",
				},
				{
					Subj: "b",
				},
				{
					Subj: "c",
				},
			},
		},
	}
	for i, td := range testData {
		out := CloneTriples(td.a)
		SortTriples(out)
		if diff, equal := messagediff.PrettyDiff(td.want, out); !equal {
			t.Errorf("%d. SortTriples(%+v) = %+v not %+v\n%s", i, td.a, out, td.want, diff)
		}
	}
}
