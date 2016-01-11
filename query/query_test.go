package query

import (
	"testing"

	"github.com/d4l3k/messagediff"
	"github.com/degdb/degdb/protocol"
)

func TestParse(t *testing.T) {
	t.Parallel()

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

func TestShardQueryByHash(t *testing.T) {
	t.Parallel()

	testData := []struct {
		step *protocol.ArrayOp
		want map[uint64]*protocol.ArrayOp
	}{
		{
			nil,
			nil,
		},
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{Subj: "foo"},
					{Subj: "bar"},
				},
			},
			map[uint64]*protocol.ArrayOp{
				0xe271865701f54561: {
					Triples: []*protocol.Triple{
						{Subj: "foo"},
						{Subj: "bar"},
					},
				},
				0x923658dbfd3ae604: {
					Triples: []*protocol.Triple{
						{Subj: "foo"},
						{Subj: "bar"},
					},
				},
			},
		},
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{Pred: "bar"},
				},
			},
			map[uint64]*protocol.ArrayOp{
				0: {
					Triples: []*protocol.Triple{
						{Pred: "bar"},
					},
				},
			},
		},
	}
	for i, td := range testData {
		out := ShardQueryByHash(td.step)
		if diff, eq := messagediff.PrettyDiff(td.want, out); !eq {
			t.Errorf("%d. Parse(%#v) = %#v\ndiff %s", i, td.step, out, diff)
		}
	}
}
