package protocol

import (
	"sort"

	"github.com/spaolacci/murmur3"
)

//go:generate protoc --gogoslick_out=. protocol.proto

// Includes checks if the provided uint64 is inside the keyspace.
func (k *Keyspace) Includes(hash uint64) bool {
	if k == nil {
		return false
	}
	a := hash
	s := k.Start
	e := k.End
	return s <= a && a < e || a < e && e < s || e < s && s <= a
}

// Mag returns the size of the keyspace.
func (k *Keyspace) Mag() uint64 {
	if k == nil {
		return 0
	}
	return k.End - k.Start
}

// Union returns the union of the keyspace. They must overlap otherwise nil is returned.
func (k *Keyspace) Union(a *Keyspace) *Keyspace {
	if a == nil && k == nil {
		return nil
	} else if a == nil {
		return k.Clone()
	} else if k == nil {
		return a.Clone()
	}
	aSI := k.Includes(a.Start) || k.End == a.Start
	aEI := k.Includes(a.End) || k.Start == a.End
	kSI := a.Includes(k.Start) || a.End == k.Start
	kEI := a.Includes(k.End) || a.Start == k.End

	switch {
	// Complete keyspace
	case aSI && aEI && kSI && kEI:
		return &Keyspace{Start: k.Start, End: k.Start - 1}

	// k encompasses a
	case aSI && aEI:
		return k.Clone()

	// a encompasses k
	case kSI && kEI:
		return a.Clone()

	// a.Start is in k
	case aSI:
		return &Keyspace{Start: k.Start, End: a.End}

	// a.End is in k
	case aEI:
		return &Keyspace{Start: a.Start, End: k.End}
	}
	return nil
}

// Maxed returns whether the keyspace encompasses the entire keyspace.
func (k *Keyspace) Maxed() bool {
	return k != nil && k.End == k.Start-1
}

// Clone makes a copy of the keyspace.
func (k *Keyspace) Clone() *Keyspace {
	return &Keyspace{Start: k.Start, End: k.End}
}

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
