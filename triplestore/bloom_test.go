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

func TestBloom(t *testing.T) {
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

	tripleCount := 5000

	additionalTriples := make([]*protocol.Triple, 0, tripleCount+len(testTriples))
	for i := 0; i < tripleCount; i++ {
		additionalTriples = append(additionalTriples, &protocol.Triple{
			Subj: "/m/0test",
			Pred: "/type/object/name",
			Obj:  "Bloom " + strconv.Itoa(i),
		})
	}
	additionalTriples = append(additionalTriples, testTriples...)
	protocol.SortTriples(additionalTriples)

	db.Insert(additionalTriples)

	filter, err := db.Bloom(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, triple := range additionalTriples {
		data, err := triple.Marshal()
		if err != nil {
			t.Error(err)
			continue
		}
		if !filter.Test(data) {
			t.Errorf("Bloom filter missing triple %+v", triple)
		}
	}

	filter2, err := db.Bloom(&protocol.Keyspace{})
	if err != nil {
		t.Fatal(err)
	}

	for _, triple := range additionalTriples {
		data, err := triple.Marshal()
		if err != nil {
			t.Error(err)
			continue
		}
		if filter2.Test(data) {
			t.Errorf("Bloom filter incorrectly has %+v", triple)
		}
	}

	var resultTriples []*protocol.Triple
	results, errs := db.TriplesMatchingBloom(filter)
	for triples := range results {
		resultTriples = append(resultTriples, triples...)
	}
	for err := range errs {
		t.Error(err)
	}
	protocol.SortTriples(resultTriples)
	if diff, ok := messagediff.PrettyDiff(additionalTriples, resultTriples); !ok {
		t.Errorf("TriplesMatchingBloom(filter) = %#v; diff %s", resultTriples, diff)
	}

	resultTriples = nil
	results, errs = db.TriplesMatchingBloom(filter2)
	for triples := range results {
		resultTriples = append(resultTriples, triples...)
	}
	for err := range errs {
		t.Error(err)
	}
	for _, triple := range resultTriples {
		t.Errorf("TriplesMatchingBLoom(nil) incorrectly has %+v", triple)
	}
}
