package core

import (
	"github.com/degdb/degdb/protocol"
	"github.com/degdb/degdb/query"
)

func (s *server) ExecuteQuery(q *protocol.QueryRequest) ([]*protocol.Triple, error) {
	switch q.Type {
	case protocol.BASIC:
		return s.ts.Query(q.Filter, -1)
	case protocol.GREMLIN:
	case protocol.MQL:
	}
	return nil, query.ErrNotImplemented
}
