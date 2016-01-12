package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/network/http"
	"github.com/degdb/degdb/protocol"
	"github.com/degdb/degdb/query"
	"github.com/spaolacci/murmur3"
)

func (s *server) initHTTP() error {
	fileServer := customhttp.NewFileServer(rice.MustFindBox("../static").HTTPBox())
	fileServer.IndexTemplate = network.IndexTemplate
	fileServer.ErrorTemplate = network.ErrorTemplate
	fileServer.PathPrefix = "/static"

	s.network.HTTPHandle("/static/", http.StripPrefix("/static/", fileServer))

	// HTTP endpoints
	s.network.HTTPHandleFunc("/api/v1/info", s.handleInfo)
	s.network.HTTPHandleFunc("/api/v1/insert", s.handleInsertTriple)
	s.network.HTTPHandleFunc("/api/v1/query", s.handleQuery)
	s.network.HTTPHandleFunc("/api/v1/triples", s.handleTriples)
	s.network.HTTPHandleFunc("/api/v1/peers", s.handlePeers)
	s.network.HTTPHandleFunc("/api/v1/myip", s.handleMyIP)

	return nil
}

// handleInsertTriple inserts set of triples into the graph.
func (s *server) handleInsertTriple(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "endpoint needs POST", 400)
		return
	}
	body, _ := ioutil.ReadAll(r.Body)
	var triples []*protocol.Triple
	if err := json.Unmarshal(body, &triples); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	hashes := make(map[uint64][]*protocol.Triple)
	unix := time.Now().Unix()
	for _, triple := range triples {
		// TODO(d4l3k): This should ideally be refactored and force the client to presign the triple.
		if err := s.crypto.SignTriple(triple); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		triple.Created = unix
		hash := murmur3.Sum64([]byte(triple.Subj))
		hashes[hash] = append(hashes[hash], triple)
	}

	for hash, triples := range hashes {
		msg := &protocol.Message{
			Message: &protocol.Message_InsertTriples{
				InsertTriples: &protocol.InsertTriples{
					Triples: triples,
				}},
			Gossip: true,
		}
		if err := s.network.Broadcast(&hash, msg); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if s.network.LocalPeer().Keyspace.Includes(hash) {
			s.ts.Insert(triples)
		}
	}
	w.Write([]byte(fmt.Sprintf("Inserted %d triples.", len(triples))))
}

// handleQuery executes a query against the graph.
func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	q := r.FormValue("q")
	s.Printf("Query: %s", q)
	triple, err := query.Parse(q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	query := &protocol.QueryRequest{
		Type: protocol.BASIC,
		Steps: []*protocol.ArrayOp{{
			Triples: triple,
		}},
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
	peers := make([]*protocol.Peer, 0, len(s.network.Peers))
	for _, peer := range s.network.Peers {
		peers = append(peers, peer.Peer)
	}
	json.NewEncoder(w).Encode(peers)
}

// handleInfo return information about the local node.
func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.network.LocalPeer())
}

// handleMyIP returns the requesters IP address.
func (s *server) handleMyIP(w http.ResponseWriter, r *http.Request) {
	addr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	w.Write([]byte(addr.IP.String()))
}
