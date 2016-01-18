package core

import (
	"testing"

	"github.com/d4l3k/messagediff"
	"github.com/degdb/degdb/protocol"
)

func TestQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	nodes := launchSwarm(5, t)
	defer killSwarm(nodes)

	primary := nodes[0]
	triples := protocol.CloneTriples(testTriples)
	if err := primary.signAndInsertTriples(triples, primary.crypto); err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		query *protocol.QueryRequest
		want  []*protocol.Triple
	}{
		{
			&protocol.QueryRequest{
				Type: protocol.BASIC,
				Steps: []*protocol.ArrayOp{{
					Triples: []*protocol.Triple{{
						Subj: "/m/02mjmr",
					}},
				}},
			},
			[]*protocol.Triple{
				{
					Subj: "/m/02mjmr",
					Pred: "/type/object/name",
					Obj:  "Barack Obama",
				},
				{
					Subj: "/m/02mjmr",
					Pred: "/type/object/type",
					Obj:  "/people/person",
				},
			},
		},
	}
	for i, td := range testData {
		trips, err := primary.ExecuteQuery(td.query)
		if err != nil {
			t.Error(err)
		}
		trips = stripCreated(stripSigning(trips))

		if diff, equal := messagediff.PrettyDiff(td.want, trips); !equal {
			t.Errorf("%d. s.ExecuteQuery(%+v) = %+v\n%s", i, td.query, trips, diff)
		}
	}
}

// stripSigning returns a copy of the triples with the signing information stripped.
func stripSigning(triples []*protocol.Triple) []*protocol.Triple {
	triples = protocol.CloneTriples(triples)
	for _, triple := range triples {
		triple.Author = ""
		triple.Sig = ""
	}
	return triples
}
