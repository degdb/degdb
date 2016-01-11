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

var testTriples = []*protocol.Triple{
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

func TestTripleDuplicates(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile(os.TempDir(), "triplestore.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	db, err := NewTripleStore(file.Name(), log.New(os.Stdout, "", log.Flags()))
	if err != nil {
		t.Fatal(err)
	}

	db.Insert(testTriples)
	// Insert twice to ensure no duplicates.
	db.Insert(testTriples)

	info, err := db.Size()
	if err != nil {
		t.Fatal(err)
	}
	if info.Triples != uint64(len(testTriples)) {
		t.Errorf("Size() = %#v; not %d", info, len(testTriples))
	}
}

func TestTripleStore(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile(os.TempDir(), "triplestore.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	db, err := NewTripleStore(file.Name(), log.New(os.Stdout, "", log.Flags()))
	if err != nil {
		t.Fatal(err)
	}

	db.Insert(testTriples)

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
		triples, err := db.Query(td.query, 100)
		if err != nil {
			t.Error(err)
		}
		if diff, ok := messagediff.PrettyDiff(td.want, triples); !ok {
			t.Errorf("%d. Query(%#v, -1) = %#v; diff %s", i, td.query, triples, diff)
		}
	}
}

func TestArrayOpToSQL(t *testing.T) {
	t.Parallel()

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
func BenchmarkTripleInsertBatch1000(b *testing.B) {
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
		triples := make([]*protocol.Triple, 1000)
		for j := 0; j < 1000; j++ {
			triples[j] = &protocol.Triple{
				Subj: "foo" + strconv.Itoa(i*1000+j),
				Pred: "some subject! woooooo",
				Obj:  "toasters are delicious",
			}
		}
		db.Insert(triples)
	}
}

func TestTripleStoreQueryArrayOp(t *testing.T) {
	t.Parallel()

	file, err := ioutil.TempFile(os.TempDir(), "triplestore.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())
	db, err := NewTripleStore(file.Name(), log.New(os.Stdout, "", log.Flags()))
	if err != nil {
		t.Fatal(err)
	}

	db.Insert(testTriples)

	testData := []struct {
		query *protocol.ArrayOp
		want  []*protocol.Triple
	}{
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{{
					Subj: "/m/02mjmr",
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
		{
			&protocol.ArrayOp{
				Triples: []*protocol.Triple{
					{
						Subj: "/m/02mjmr",
					},
					{
						Subj: "/m/0hume",
					},
				},
			},
			testTriples,
		},
		{
			&protocol.ArrayOp{
				Mode: protocol.AND,
				Triples: []*protocol.Triple{
					{
						Subj: "/m/02mjmr",
					},
					{
						Subj: "/m/0hume",
					},
				},
			},
			nil,
		},
		{
			&protocol.ArrayOp{
				Mode: protocol.NOT,
				Triples: []*protocol.Triple{
					{
						Subj: "/m/0hume",
					},
				},
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
		triples, err := db.QueryArrayOp(td.query, 100)
		if err != nil {
			t.Error(err)
		}
		if diff, ok := messagediff.PrettyDiff(td.want, triples); !ok {
			t.Errorf("%d. Query(%#v, -1) = %#v; diff %s", i, td.query, triples, diff)
		}
	}
}
