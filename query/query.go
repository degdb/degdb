package query

import (
	"encoding/json"

	"github.com/degdb/degdb/protocol"
)

func Parse(query string) ([]*protocol.Triple, error) {
	var filters []*protocol.Triple
	if err := json.Unmarshal([]byte(query), &filters); err != nil {
		return nil, err
	}
	return filters, nil
}
