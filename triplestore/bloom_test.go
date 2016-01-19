package triplestore

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"testing"

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

	db.Insert(testTriples)

	additionalTriples := make([]*protocol.Triple, 10000)
	for i, _ := range additionalTriples {
		additionalTriples[i] = &protocol.Triple{
			Subj: "/m/0test",
			Pred: "/type/object/name",
			Obj:  "Bloom " + strconv.Itoa(i),
		}
	}
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

	filter2, err := db.Bloom(&protocol.Keyspace{0, 0})
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
}
