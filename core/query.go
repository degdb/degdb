package core

import (
	"sync"
	"time"

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

			// External request and is already sharded.
			if q.Sharded {
				return s.ts.QueryArrayOp(step, int(q.Limit))
			}

			var wg sync.WaitGroup
			var triplesLock sync.RWMutex
			triples = nil
			shards := query.ShardQueryByHash(step)

			// Unrooted queries
			if arrayOp, ok := shards[0]; ok {
				// TODO localnode
				set := s.network.MinimumCoveringPeers()
				s.Printf("Minimum covering set %+v", set)
				_ = arrayOp
				wg.Add(len(set))
				req := basicReq(arrayOp)
				var err error
				for _, conn := range set {
					conn := conn
					go func() {
						var msg *protocol.Message
						msg, err = conn.Request(req)
						if err != nil {
							return
						}
						triplesLock.Lock()
						// TODO(d4l3k): Deduplicate triples
						triples = append(triples, msg.GetQueryResponse().Triples...)
						triplesLock.Unlock()
						done := make(chan bool, 1)
						go func() {
							wg.Done()
							done <- true
						}()
						go func() {
							time.Sleep(10 * time.Second)
							done <- true
						}()
						<-done
					}()
				}
				wg.Wait()
				return triples, err
			}

			// Rooted queries
			for hash, arrayOp := range shards {
				if hash == 0 {
					return nil, query.ErrUnRooted
				}
				if s.network.LocalPeer().Keyspace.Includes(hash) {
					trips, err := s.ts.QueryArrayOp(arrayOp, int(q.Limit))
					if err != nil {
						return nil, err
					}
					triples = append(triples, trips...)
					continue
				}
			Peers:
				for _, conn := range s.network.Peers {
					if conn == nil || conn.Peer == nil {
						continue
					}
					if conn.Peer.Keyspace.Includes(hash) {
						req := basicReq(arrayOp)
						// TODO(d4l3k) Parallelize
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

func basicReq(arrayOp *protocol.ArrayOp) *protocol.Message {
	return &protocol.Message{Message: &protocol.Message_QueryRequest{
		QueryRequest: &protocol.QueryRequest{
			Type:    protocol.BASIC,
			Steps:   []*protocol.ArrayOp{arrayOp},
			Sharded: true,
		}}}
}
