package protocol

import (
	"sort"

	"github.com/spaolacci/murmur3"
)

//go:generate protoc --gogoslick_out=. protocol.proto

func (msg *Message) Hash() uint64 {
	data, _ := msg.Marshal()
	return murmur3.Sum64(data)
}

// CloneTriples makes a shallow copy of the triples.
func CloneTriples(triples []*Triple) []*Triple {
	ntrips := make([]*Triple, len(triples))
	for i, triple := range triples {
		t := *triple
		ntrips[i] = &t
	}
	return ntrips
}

// SortTriples sorts a slice of triples by Subj, Pred, Obj
func SortTriples(triples []*Triple) {
	sort.Sort(TripleSlice(triples))
}

// See SortTriples
type TripleSlice []*Triple

func (p TripleSlice) Len() int { return len(p) }
func (p TripleSlice) Less(i, j int) bool {
	a := p[i]
	b := p[j]
	return a.Subj < b.Subj && a.Pred < b.Pred && a.Obj < b.Obj
}
func (p TripleSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
