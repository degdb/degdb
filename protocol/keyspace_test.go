package protocol

import (
	"math"
	"testing"

	"github.com/d4l3k/messagediff"
)

func TestKeyspaceIncludes(t *testing.T) {
	t.Parallel()

	testData := []struct {
		ks   *Keyspace
		hash uint64
		want bool
	}{
		{
			&Keyspace{
				Start: 1,
				End:   100,
			},
			50,
			true,
		},
		{
			&Keyspace{
				Start: 1,
				End:   100,
			},
			150,
			false,
		},
		{
			&Keyspace{
				Start: 100,
				End:   1,
			},
			150,
			true,
		},
		{
			&Keyspace{
				Start: 100,
				End:   1,
			},
			50,
			false,
		},
		{
			&Keyspace{
				Start: 100,
				End:   50,
			},
			25,
			true,
		},
		{
			&Keyspace{
				Start: 100,
				End:   50,
			},
			75,
			false,
		},
		{
			nil,
			0,
			false,
		},
	}
	for i, td := range testData {
		if out := td.ks.Includes(td.hash); out != td.want {
			t.Errorf("%d. %#v.Includes(%#v) = %#v not %#v", i, td.ks, td.hash, out, td.want)
		}
	}
}

func TestKeyspaceUnion(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a, b, want *Keyspace
	}{
		{
			&Keyspace{1, 10},
			&Keyspace{20, 30},
			nil,
		},
		{
			&Keyspace{1, 10},
			&Keyspace{10, 20},
			&Keyspace{1, 20},
		},
		{
			&Keyspace{10, 20},
			&Keyspace{1, 10},
			&Keyspace{1, 20},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{5, 10},
			&Keyspace{1, 20},
		},
		{
			&Keyspace{5, 10},
			&Keyspace{1, 20},
			&Keyspace{1, 20},
		},
		{
			&Keyspace{math.MaxUint64 - 5, math.MaxUint64 - 1},
			&Keyspace{math.MaxUint64 - 1, 20},
			&Keyspace{math.MaxUint64 - 5, 20},
		},
		{
			&Keyspace{math.MaxUint64 - 1, 20},
			&Keyspace{math.MaxUint64 - 5, math.MaxUint64 - 1},
			&Keyspace{math.MaxUint64 - 5, 20},
		},
		{
			&Keyspace{math.MaxUint64 - 5, 1},
			&Keyspace{1, 20},
			&Keyspace{math.MaxUint64 - 5, 20},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{math.MaxUint64 - 5, 1},
			&Keyspace{math.MaxUint64 - 5, 20},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{20, 1},
			&Keyspace{1, 0},
		},
		{
			nil, nil, nil,
		},
		{
			&Keyspace{1, 2},
			nil,
			&Keyspace{1, 2},
		},
		{
			nil,
			&Keyspace{1, 2},
			&Keyspace{1, 2},
		},
	}
	for i, td := range testData {
		if out := td.a.Union(td.b); !out.Equal(td.want) {
			t.Errorf("%d. %+v.Union(%+v) = %+v not %+v", i, td.a, td.b, out, td.want)
		}
	}
}

func TestKeyspaceIntersection(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a, b, want *Keyspace
	}{
		{
			&Keyspace{1, 10},
			&Keyspace{20, 30},
			nil,
		},
		{
			&Keyspace{1, 10},
			&Keyspace{10, 20},
			&Keyspace{10, 10},
		},
		{
			&Keyspace{10, 20},
			&Keyspace{1, 10},
			&Keyspace{10, 10},
		},
		{
			&Keyspace{1, 15},
			&Keyspace{10, 20},
			&Keyspace{10, 15},
		},
		{
			&Keyspace{10, 20},
			&Keyspace{1, 15},
			&Keyspace{10, 15},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{5, 10},
			&Keyspace{5, 10},
		},
		{
			&Keyspace{5, 10},
			&Keyspace{1, 20},
			&Keyspace{5, 10},
		},
		{
			&Keyspace{math.MaxUint64 - 5, math.MaxUint64 - 1},
			&Keyspace{math.MaxUint64 - 1, 20},
			&Keyspace{math.MaxUint64 - 1, math.MaxUint64 - 1},
		},
		{
			&Keyspace{math.MaxUint64 - 1, 20},
			&Keyspace{math.MaxUint64 - 5, math.MaxUint64 - 1},
			&Keyspace{math.MaxUint64 - 1, math.MaxUint64 - 1},
		},
		{
			&Keyspace{math.MaxUint64 - 5, 1},
			&Keyspace{1, 20},
			&Keyspace{1, 1},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{math.MaxUint64 - 5, 1},
			&Keyspace{1, 1},
		},
		{
			&Keyspace{1, 20},
			&Keyspace{20, 1},
			&Keyspace{1, 1},
		},
		{
			nil, nil, nil,
		},
		{
			&Keyspace{1, 2},
			nil,
			nil,
		},
		{
			nil,
			&Keyspace{1, 2},
			nil,
		},
	}
	for i, td := range testData {
		if out := td.a.Intersection(td.b); !out.Equal(td.want) {
			t.Errorf("%d. %+v.Intersection(%+v) = %+v not %+v", i, td.a, td.b, out, td.want)
		}
	}
}

func TestKeyspaceMag(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a    *Keyspace
		want uint64
	}{
		{
			&Keyspace{1, 10},
			9,
		},
		{
			&Keyspace{math.MaxUint64 - 5, 1},
			7,
		},
	}
	for i, td := range testData {
		if out := td.a.Mag(); out != td.want {
			t.Errorf("%d. %+v.Mag() = %+v not %+v", i, td.a, out, td.want)
		}
	}
}

func TestKeyspaceMaxed(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a    *Keyspace
		want bool
	}{
		{
			&Keyspace{1, 10},
			false,
		},
		{
			&Keyspace{2, 1},
			true,
		},
		{
			(&Keyspace{1, 20}).Union(&Keyspace{20, 1}),
			true,
		},
	}
	for i, td := range testData {
		if out := td.a.Maxed(); out != td.want {
			t.Errorf("%d. %+v.Maxed() = %+v not %+v", i, td.a, out, td.want)
		}
	}
}

func TestKeyspaceComplement(t *testing.T) {
	t.Parallel()

	testData := []struct {
		a, want *Keyspace
	}{
		{
			&Keyspace{1, 10},
			&Keyspace{10, 1},
		},
		{
			nil,
			&Keyspace{1, 0},
		},
		{
			&Keyspace{1, 0},
			nil,
		},
	}
	for i, td := range testData {
		out := td.a.Complement()
		if diff, equal := messagediff.PrettyDiff(td.want, out); !equal {
			t.Errorf("%d. %+v.Complement() = %+v not %+v\n%s", i, td.a, out, td.want, diff)
		}
	}
}
