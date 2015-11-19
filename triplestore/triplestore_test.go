package triplestore

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/d4l3k/messagediff"

	"github.com/degdb/degdb/protocol"
)

func TestTripleStore(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "triplestore.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	db, err := NewTripleStore(file.Name(), log.New(os.Stdout, "", log.Flags()))
	if err != nil {
		t.Fatal(err)
	}

	triples := []*protocol.Triple{
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
		{
			Subj: "/m/0hume",
			Pred: "/type/object/name",
			Obj:  "Hume",
		},
		{
			Subj: "/m/0hume",
			Pred: "/type/object/type",
			Obj:  "/organization/team",
		},
	}

	db.Insert(triples)
	// Insert twice to ensure no duplicates.
	db.Insert(triples)

	testData := []struct {
		query *protocol.Triple
		want  []*protocol.Triple
	}{
		{
			&protocol.Triple{
				Subj: "/m/02mjmr",
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
		{
			&protocol.Triple{
				Pred: "/type/object/type",
			},
			[]*protocol.Triple{
				{
					Subj: "/m/02mjmr",
					Pred: "/type/object/type",
					Obj:  "/people/person",
				},
				{
					Subj: "/m/0hume",
					Pred: "/type/object/type",
					Obj:  "/organization/team",
				},
			},
		},
		{
			&protocol.Triple{
				Pred: "/type/object/name",
				Obj:  "Barack Obama",
			},
			[]*protocol.Triple{
				{
					Subj: "/m/02mjmr",
					Pred: "/type/object/name",
					Obj:  "Barack Obama",
				},
			},
		},
	}

	for i, td := range testData {
		triples, err := db.Query(td.query, -1)
		if err != nil {
			t.Error(err)
		}
		if diff, ok := messagediff.PrettyDiff(td.want, triples); !ok {
			t.Errorf("%d. Query(%#v, -1) = %#v; diff %s", i, td.query, triples, diff)
		}
	}

	info, err := db.Size()
	if err != nil {
		t.Fatal(err)
	}
	if info.Triples != uint64(len(triples)) {
		t.Errorf("Size() = %#v; not %d", info, len(triples))
	}
}

func TestArrayOpToSQL(t *testing.T) {
	testData := []struct {
		op   *protocol.ArrayOp
		want []string
	}{
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{Subj: "subj", Pred: "pred", Obj: "obj", Lang: "lang", Author: "author"},
					{Subj: "subj1"},
				},
				Mode: protocol.AND,
			},
			[]string{"(subj = ? AND pred = ? AND obj = ? AND lang = ? AND author = ?) AND (subj = ?)",
				"subj", "pred", "obj", "lang", "author", "subj1"},
		},
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{Subj: "subj1"},
					{Subj: "subj2"},
				},
				Mode: protocol.OR,
			},
			[]string{"(subj = ?) OR (subj = ?)", "subj1", "subj2"},
		},
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{Subj: "subj"},
				},
				Mode: protocol.NOT,
			},
			[]string{"NOT (subj = ?)", "subj"},
		},
		{
			&protocol.ArrayOp{
				Arguments: []*protocol.ArrayOp{{
					Triples: []*protocol.Triple{
						{Subj: "subj"},
					},
				}},
				Mode: protocol.NOT,
			},
			[]string{"NOT ((subj = ?))", "subj"},
		},
		{
			&protocol.ArrayOp{
				Arguments: []*protocol.ArrayOp{
					{
						Triples: []*protocol.Triple{
							{Subj: "subj1"},
						},
					},
					{
						Triples: []*protocol.Triple{
							{Subj: "subj2"},
						},
					},
				},
				Mode: protocol.AND,
			},
			[]string{"((subj = ?)) AND ((subj = ?))", "subj1", "subj2"},
		},
		{
			&protocol.ArrayOp{
				Arguments: []*protocol.ArrayOp{
					{
						Triples: []*protocol.Triple{
							{Subj: "subj1"},
						},
					},
					{
						Triples: []*protocol.Triple{
							{Subj: "subj2"},
						},
					},
				},
				Mode: protocol.OR,
			},
			[]string{"((subj = ?)) OR ((subj = ?))", "subj1", "subj2"},
		},
	}

	for i, td := range testData {
		sql := ArrayOpToSQL(td.op)
		if diff, ok := messagediff.PrettyDiff(td.want, sql); !ok {
			t.Errorf("%d. ArrayOpToSQL(%#v) = %#v; diff %s", i, td.op, sql, diff)
		}
	}
}

func BenchmarkTripleInsert(b *testing.B) {

	file, err := ioutil.TempFile(os.TempDir(), "triplestore.db")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(file.Name())
	db, err := NewTripleStore(file.Name(), log.New(os.Stdout, "", log.Flags()))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		triple := &protocol.Triple{
			Subj: "foo" + strconv.Itoa(i),
			Pred: "some subject! woooooo",
			Obj:  "toasters are delicious",
		}
		db.Insert([]*protocol.Triple{triple})
	}
}
