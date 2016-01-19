package triplestore

import (
	"github.com/spaolacci/murmur3"
	"github.com/tylertreat/BoomFilters"

	"github.com/degdb/degdb/protocol"
)

// Bloom returns a ScalableBloomFilter containing all the triples the current node has in the optional keyspace.
func (ts *TripleStore) Bloom(keyspace *protocol.Keyspace) (*boom.ScalableBloomFilter, error) {
	filter := boom.NewDefaultScalableBloomFilter(BloomFalsePositiveRate)

	dbq := ts.db.Where(&protocol.Triple{}).Limit(1000)

	var results []*protocol.Triple
	for i := 0; i == 0 || len(results) > 0; i++ {
		results = results[0:0]

		if err := dbq.Offset(i * 1000).Find(&results).Error; err != nil {
			return nil, err
		}
		for _, triple := range results {
			if keyspace != nil {
				hash := murmur3.Sum64([]byte(triple.Subj))
				if !keyspace.Includes(hash) {
					continue
				}
			}
			data, err := triple.Marshal()
			if err != nil {
				return nil, err
			}
			filter.Add(data)
		}
	}
	return filter, nil
}
