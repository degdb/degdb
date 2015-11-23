package protocol

import "github.com/spaolacci/murmur3"

//go:generate protoc --gogoslick_out=. protocol.proto

// Includes checks if the provided uint64 is inside the keyspace.
func (k *Keyspace) Includes(hash uint64) bool {
	a := hash
	s := k.Start
	e := k.End
	return s <= a && a < e || a < e && e < s || e < s && s <= a
}

// Mag returns the size of the keyspace.
func (k *Keyspace) Mag() uint64 {
	return k.End - k.Start
}

// Union returns the union of the keyspace. They must overlap otherwise nil is returned.
func (k *Keyspace) Union(a *Keyspace) *Keyspace {
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
		return &Keyspace{Start: k.Start, End: k.End}

	// a encompasses k
	case kSI && kEI:
		return &Keyspace{Start: a.Start, End: a.End}

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
	return k.End == k.Start-1
}

func (msg *Message) Hash() uint64 {
	data, _ := msg.Marshal()
	return murmur3.Sum64(data)
}
