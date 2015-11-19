package query

import (
	"encoding/json"
	"errors"

	"github.com/degdb/degdb/protocol"
	"github.com/spaolacci/murmur3"
)

var (
	ErrNotImplemented = errors.New("query protocol type is not implemented")
	ErrUnRooted       = errors.New("unrooted queries are not implemented")
	ErrArrayOp        = errors.New("triplestore QueryArrayOp is not implemented")
)

func Parse(query string) ([]*protocol.Triple, error) {
	var filters []*protocol.Triple
	if err := json.Unmarshal([]byte(query), &filters); err != nil {
		return nil, err
	}
	return filters, nil
}

func ShardQueryByHash(step *protocol.ArrayOp) map[uint64]*protocol.ArrayOp {
	m := make(map[uint64]*protocol.ArrayOp)
	// TODO(d4l3k): better query hash splitting
	if step.Mode == protocol.OR && len(step.Triples) > 0 {
		for _, triple := range step.Triples {
			if len(triple.Subj) == 0 {
				// TODO(d4l3k): Handle people sneaking in unrooted queries.
				continue
			}
			hash := murmur3.Sum64([]byte(triple.Subj))
			m[hash] = step
		}
	} else {
		m[0] = step
	}
	return m
}
