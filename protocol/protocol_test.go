package protocol

import "testing"

func TestCloneTriples(t *testing.T) {
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
