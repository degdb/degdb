package query

import (
	"testing"

	"github.com/d4l3k/messagediff"
	"github.com/degdb/degdb/protocol"
)

func TestParse(t *testing.T) {
	testData := []struct {
		in   string
		want []*protocol.Triple
	}{{
		`[{"subj":"foo", "pred":"bar", "obj":"moo"}, {}]`,
		[]*protocol.Triple{
			{
				Subj: "foo",
				Pred: "bar",
				Obj:  "moo",
			},
			{},
		},
	}}
	for i, td := range testData {
		out, err := Parse(td.in)
		if err != nil {
			t.Error(err)
		}
		if diff, eq := messagediff.PrettyDiff(td.want, out); !eq {
			t.Errorf("%d. Parse(%#v) = %#v\ndiff %s", i, td.in, out, diff)
		}
	}
}
