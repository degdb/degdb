package core

import (
	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/protocol"
	"github.com/spaolacci/murmur3"
)

// initBinary initalizes the binary endpoints.
func (s *server) initBinary() error {
	s.network.Handle("InsertTriples", s.handleInsertTriples)
	s.network.Handle("QueryRequest", s.handleQueryRequest)

	return nil
}

func (s *server) handleInsertTriples(conn *network.Conn, msg *protocol.Message) {
	triples := msg.GetInsertTriples().Triples
	localKS := s.network.LocalPeer().Keyspace

	var validTriples []*protocol.Triple
	idHashes := make(map[string]uint64)
	for _, triple := range triples {
		hash, ok := idHashes[triple.Subj]
		if !ok {
			hash = murmur3.Sum64([]byte(triple.Subj))
			idHashes[triple.Subj] = hash
		}
		if !localKS.Includes(hash) {
			s.Printf("ERR insert triple dropped due to keyspace %#v from %#v", triple, conn.Peer)
			// TODO(d4l3k): Follow up on bad triple by reannouncing keyspace.
			continue
		}
		validTriples = append(validTriples, triple)
	}
	s.ts.Insert(validTriples)
}

func (s *server) handleQueryRequest(conn *network.Conn, msg *protocol.Message) {
	triples, err := s.ExecuteQuery(msg.GetQueryRequest())
	resp := &protocol.Message{
		Message: &protocol.Message_QueryResponse{
			QueryResponse: &protocol.QueryResponse{
				Triples: triples,
			},
		},
	}
	if err != nil {
		resp.Error = err.Error()
	}
	if err := conn.RespondTo(msg, resp); err != nil {
		s.Printf("ERR send QueryResponse %s", err)
	}
}
