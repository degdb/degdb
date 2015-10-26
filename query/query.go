package query

import (
	"encoding/json"
	"errors"

	"github.com/degdb/degdb/protocol"
)

var ErrNotImplemented = errors.New("query protocol type is not implemented")

func Parse(query string) ([]*protocol.Triple, error) {
	var filters []*protocol.Triple
	if err := json.Unmarshal([]byte(query), &filters); err != nil {
		return nil, err
	}
	return filters, nil
}
