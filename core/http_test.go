package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/d4l3k/messagediff"
	"github.com/degdb/degdb/protocol"
	"github.com/spaolacci/murmur3"
)

func init() {
	newTmpDir()
	protocol.SortTriples(testTriples)
}

func newTmpDir() {
	dir, err := ioutil.TempDir("", "degdb-test-files")
	if err != nil {
		log.Fatal(err)
	}
	KeyFilePath = dir + "/degdb-%d.key"
	DatabaseFilePath = dir + "/degdb-%d.db"
}

func testServer(t *testing.T) *server {
	newTmpDir()
	s, err := newServer(0, nil, diskAllocated)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestHTTP(t *testing.T) {
	t.Parallel()

	s := testServer(t)
	go s.network.Listen()
	time.Sleep(10 * time.Millisecond)
	base := fmt.Sprintf("http://localhost:%d", s.network.Port)

	localPeerJSON, err := json.Marshal(s.network.LocalPeer())
	if err != nil {
		t.Fatal(err)
	}
	hosts, err := net.LookupHost("localhost")
	if err != nil {
		t.Fatal(err)
	}

	testData := []struct {
		path string
		want []string
	}{
		{
			"/api/v1/info",
			[]string{string(localPeerJSON)},
		},
		{
			"/api/v1/myip",
			hosts,
		},
		{
			"/api/v1/peers",
			[]string{"[]"},
		},
	}

	for i, td := range testData {
		url := base + td.path
		resp, err := http.Get(url)
		if err != nil {
			t.Error(err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		bodyTrim := strings.TrimSpace(string(body))
		found := false
		for _, want := range td.want {
			wantTrim := strings.TrimSpace(want)
			if bodyTrim == wantTrim {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%d. http.Get(%+v) = %+v; not one of %+v", i, td.path, bodyTrim, td.want)
		}
	}
}

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

// subjInKeyspace appends an increasing number to the prefix until it finds a
// string that will hash into the given keyspace.
func subjInKeyspace(keyspace *protocol.Keyspace, prefix string) string {
	subj := prefix
	i := 1
	for !keyspace.Includes(murmur3.Sum64([]byte(subj))) {
		subj = prefix + strconv.Itoa(i)
		i++
	}
	return subj
}

// testTriplesKeyspcae returns a set of triples in the current keyspace.
func testTriplesKeyspace(keyspace *protocol.Keyspace) []*protocol.Triple {
	subjs := make(map[string]string)
	var triples []*protocol.Triple
	for _, triple := range testTriples {
		triple := *triple
		if subj, ok := subjs[triple.Subj]; ok {
			triple.Subj = subj
		} else {
			subj = subjInKeyspace(keyspace, triple.Subj)
			subjs[triple.Subj] = subj
			triple.Subj = subj
		}
		triples = append(triples, &triple)
	}
	protocol.SortTriples(triples)
	return triples
}

func TestInsertAndRetreiveTriples(t *testing.T) {
	t.Parallel()

	s := testServer(t)
	go s.network.Listen()

	time.Sleep(10 * time.Millisecond)
	base := fmt.Sprintf("http://localhost:%d", s.network.Port)

	testTriples := testTriplesKeyspace(s.network.LocalKeyspace())

	triples, err := json.Marshal(testTriples)
	if err != nil {
		t.Error(err)
	}

	buf := bytes.NewBuffer(triples)

	resp, err := http.Post(base+"/api/v1/insert", "application/json", buf)
	if err != nil {
		t.Error(err)
	}
	out, _ := ioutil.ReadAll(resp.Body)
	if !bytes.Contains(out, []byte(strconv.Itoa(len(testTriples)))) {
		t.Errorf("http.Post(/api/v1/insert) = %+v; missing %+v", string(out), len(testTriples))
	}

	// Takes a bit of time to write
	time.Sleep(100 * time.Millisecond)

	var signedTriples []*protocol.Triple

	resp, err = http.Get(base + "/api/v1/triples")
	if err != nil {
		t.Error(err)
	}
	err = json.NewDecoder(resp.Body).Decode(&signedTriples)
	if err != nil {
		t.Error(err)
	}
	strippedTriples := stripCreated(stripSigning(signedTriples))
	protocol.SortTriples(strippedTriples)

	if diff, equal := messagediff.PrettyDiff(testTriples, strippedTriples); !equal {
		t.Errorf("http.Get(/api/v1/insert) = %+v\n;not %+v\n%s", strippedTriples, testTriples, diff)
	}
}

func stripCreated(triples []*protocol.Triple) []*protocol.Triple {
	triples = protocol.CloneTriples(triples)
	for _, triple := range triples {
		triple.Created = 0
	}
	return triples
}
