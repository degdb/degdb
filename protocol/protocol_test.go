package protocol

import "testing"

func TestIncludes(t *testing.T) {
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
	}
	for i, td := range testData {
		if out := td.ks.Includes(td.hash); out != td.want {
			t.Errorf("%d. %#v.Includes(%#v) = %#v not %#v", i, td.ks, td.hash, out, td.want)
		}
	}
}
