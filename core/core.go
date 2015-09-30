// Package core contains the rewritten degdb code.
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/degdb/degdb/crypto"
	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/protocol"
	"github.com/degdb/degdb/triplestore"
	"github.com/spaolacci/murmur3"
)

type server struct {
	diskAllocated int
	port          int
	network       *network.Server
	ts            *triplestore.TripleStore
	crypto        *crypto.PrivateKey

	*log.Logger
}

func Main(port int, peers []string, diskAllocated int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s := &server{
		Logger:        log.New(os.Stdout, fmt.Sprintf(":%d ", port), log.Flags()),
		diskAllocated: diskAllocated,
		port:          port,
	}

	if err := s.init(); err != nil {
		s.Fatal(err)
	}

	go s.connectPeers(peers)
	s.Fatal(s.network.Listen())
}

func (s *server) connectPeers(peers []string) {
	for _, peer := range peers {
		peer := peer
		time.Sleep(200 * time.Millisecond)
		go func() {
			s.Printf("Connecting to peer %s", peer)
			if err := s.network.Connect(peer); err != nil {
				s.Printf("ERR connecting to peer: %s", err)
			}
		}()
	}
}

func (s *server) init() error {
	s.Printf("Initializing crypto...")
	keyFile := fmt.Sprintf("degdb-%d.key", s.port)
	privKey, err := crypto.ReadOrGenerateKey(keyFile)
	if err != nil {
		return err
	}
	s.crypto = privKey

	s.Printf("Initializing triplestore...")
	s.Printf("Max DB size = %d bytes.", s.diskAllocated)
	dbFile := fmt.Sprintf("degdb-%d.db", s.port)
	ts, err := triplestore.NewTripleStore(dbFile)
	if err != nil {
		return err
	}
	s.ts = ts

	s.Printf("Initializing network...")
	ns, err := network.NewServer(s.Logger, s.port)
	if err != nil {
		return err
	}
	s.network = ns

	s.network.Mux.HandleFunc("/", s.handleNotFound)
	s.network.Mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(rice.MustFindBox("../static").HTTPBox())))

	// HTTP endpoints
	s.network.Mux.HandleFunc("/api/v1/peers", s.handlePeers)
	s.network.Mux.HandleFunc("/api/v1/triples", s.handleTriples)
	s.network.Mux.HandleFunc("/api/v1/insert", s.handleInsertTriple)
	s.network.Mux.HandleFunc("/api/v1/query", s.handleQuery)

	// Binary endpoints
	s.network.Handle("InsertTriples", s.handleInsertTriples)

	return nil
}

func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	q := r.FormValue("q")
	log.Printf("Query: %s", q)
}

func (s *server) handleInsertTriples(conn *network.Conn, msg *protocol.Message) {
	// TODO(d4l3k): Verify correct keyspace
	s.ts.Insert(msg.GetInsertTriples().Triples)
}

func (s *server) handlePeers(w http.ResponseWriter, r *http.Request) {
	var peers []*protocol.Peer
	for _, peer := range s.network.Peers {
		peers = append(peers, peer.Peer)
	}
	json.NewEncoder(w).Encode(peers)
}

func (s *server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("degdb: file not found %s", r.URL), 404)
}

func (s *server) handleInsertTriple(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "endpoint needs POST", 400)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	subj := r.FormValue("subj")
	pred := r.FormValue("pred")
	obj := r.FormValue("obj")
	lang := r.FormValue("lang")
	triple := &protocol.Triple{
		Subj: subj,
		Pred: pred,
		Obj:  obj,
		Lang: lang,
	}
	if err := s.crypto.SignTriple(triple); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	msg := &protocol.Message{
		Message: &protocol.Message_InsertTriples{
			InsertTriples: &protocol.InsertTriples{
				Triples: []*protocol.Triple{triple},
			},
		},
		Gossip: true,
	}
	hash := murmur3.Sum64([]byte(triple.Subj))
	if err := s.network.Broadcast(hash, msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/static/insert.html", 307)
}
func (s *server) handleTriples(w http.ResponseWriter, r *http.Request) {
	triples, err := s.ts.Query(&protocol.Triple{}, -1)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(triples)
}
