package core

import (
	"github.com/degdb/degdb/protocol"
	"github.com/degdb/degdb/query"
)

func (s *server) ExecuteQuery(q *protocol.QueryRequest) ([]*protocol.Triple, error) {
	var triples []*protocol.Triple
	switch q.Type {
	case protocol.BASIC:
		for i, step := range q.Steps {
			if i != 0 {
				var midTriples []*protocol.Triple
				for _, triple := range triples {
					midTriples = append(midTriples, &protocol.Triple{
						Subj: triple.Obj,
					})
				}
				step = &protocol.ArrayOp{
					Arguments: []*protocol.ArrayOp{step},
					Mode:      protocol.AND,
					Triples:   midTriples,
				}
			}
			triples = nil
			for hash, q := range query.ShardQueryByHash(step) {
				if hash == 0 {
					// TODO(d4l3k): Unrooted queries
					return nil, query.ErrUnRooted
				}
				if s.network.LocalPeer().Keyspace.Includes(hash) {
					trips, err := s.ts.QueryArrayOp(q)
					if err != nil {
						return nil, err
					}
					triples = append(triples, trips...)
					continue
				}
			Peers:
				for _, conn := range s.network.Peers {
					if conn.Peer.Keyspace.Includes(hash) {
						req := &protocol.Message{Message: &protocol.Message_QueryRequest{
							QueryRequest: &protocol.QueryRequest{
								Type:  protocol.BASIC,
								Steps: []*protocol.ArrayOp{q},
							}}}
						msg, err := conn.Request(req)
						if err != nil {
							return nil, err
						}
						triples = append(triples, msg.GetQueryResponse().Triples...)
						break Peers
					}
				}
			}
		}

	//case protocol.GREMLIN:
	//case protocol.MQL:
	default:
		return nil, query.ErrNotImplemented
	}
	return triples, nil
}
