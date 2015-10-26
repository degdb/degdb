package core

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"github.com/degdb/degdb/protocol"
	"github.com/degdb/degdb/query"
	"github.com/spaolacci/murmur3"
)

func (s *server) initHTTP() error {
	s.network.Mux.HandleFunc("/", s.handleNotFound)
	s.network.Mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(rice.MustFindBox("../static").HTTPBox())))

	// HTTP endpoints
	s.network.Mux.HandleFunc("/api/v1/insert", s.handleInsertTriple)
	s.network.Mux.HandleFunc("/api/v1/query", s.handleQuery)
	s.network.Mux.HandleFunc("/api/v1/triples", s.handleTriples)
	s.network.Mux.HandleFunc("/api/v1/peers", s.handlePeers)

	return nil
}

// handleNotFound renders a 404 page for missing pages.
func (s *server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, fmt.Sprintf("degdb: file not found %s", r.URL), 404)
}

// handleInsertTriple inserts a triple into the graph.
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
	// TODO(d4l3k): This should ideally be refactored and force the client to presign the triple.
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
	if err := s.network.Broadcast(&hash, msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if s.network.LocalPeer().Keyspace.Includes(hash) {
		s.ts.Insert(msg.GetInsertTriples().Triples)
	}
	http.Redirect(w, r, "/static/insert.html", 307)
}

// handleQuery executes a query against the graph.
func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	q := r.FormValue("q")
	log.Printf("Query: %s", q)
	triple, err := query.Parse(q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	query := &protocol.QueryRequest{
		Type:   protocol.BASIC,
		Filter: triple[0],
	}
	triples, err := s.ExecuteQuery(query)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	json.NewEncoder(w).Encode(triples)
}

// handleTriples is a debug method to dump the triple DB into a JSON blob.
func (s *server) handleTriples(w http.ResponseWriter, r *http.Request) {
	triples, err := s.ts.Query(&protocol.Triple{}, -1)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(triples)
}

// handlePeers is a debug method to dump the current known peers.
func (s *server) handlePeers(w http.ResponseWriter, r *http.Request) {
	var peers []*protocol.Peer
	for _, peer := range s.network.Peers {
		peers = append(peers, peer.Peer)
	}
	json.NewEncoder(w).Encode(peers)
}
