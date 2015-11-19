package triplestore

import (
	"io/ioutil"
	"log"
	"os"
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
