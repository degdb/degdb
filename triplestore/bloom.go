package triplestore

import (
	"github.com/spaolacci/murmur3"
	"github.com/tylertreat/BoomFilters"

	"github.com/degdb/degdb/protocol"
)

// DefaultTripleBatchSize is the default number of triples to use when streaming.
var DefaultTripleBatchSize = 1000

// Bloom returns a ScalableBloomFilter containing all the triples the current node has in the optional keyspace.
func (ts *TripleStore) Bloom(keyspace *protocol.Keyspace) (*boom.ScalableBloomFilter, error) {
	filter := boom.NewDefaultScalableBloomFilter(BloomFalsePositiveRate)

	results, errs := ts.EachTripleBatch(DefaultTripleBatchSize)
	for triples := range results {
		for _, triple := range triples {
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
	for err := range errs {
		return nil, err
	}
	return filter, nil
}

// TriplesMatchingBloom streams triples in batches of 1000 that match the bloom filter.
func (ts *TripleStore) TriplesMatchingBloom(filter *boom.ScalableBloomFilter) (<-chan []*protocol.Triple, <-chan error) {
	c := make(chan []*protocol.Triple, 10)
	cerr := make(chan error, 1)
	go func() {
		triples := make([]*protocol.Triple, 0, DefaultTripleBatchSize)
		results, errs := ts.EachTripleBatch(DefaultTripleBatchSize)
		for resultTriples := range results {
			for _, triple := range resultTriples {
				data, err := triple.Marshal()
				if err != nil {
					cerr <- err
					break
				}
				if !filter.Test(data) {
					continue
				}
				triples = append(triples, triple)
				if len(triples) >= DefaultTripleBatchSize {
					c <- triples
					triples = make([]*protocol.Triple, 0, DefaultTripleBatchSize)
				}
			}
			if len(triples) > 0 {
				c <- triples
			}
		}
		for err := range errs {
			cerr <- err
		}
		close(c)
		close(cerr)
	}()
	return c, cerr
}
