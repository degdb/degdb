package protocol

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

// Union returns the union of the keyspaces. They must overlap otherwise nil is returned.
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

// Intersection returns the intersection of the keyspaces. If there are
// multiple intersections, it returns the first.
func (k *Keyspace) Intersection(a *Keyspace) *Keyspace {
	if a == nil && k == nil {
		return nil
	} else if a == nil {
		return nil
	} else if k == nil {
		return nil
	}
	aSI := k.Includes(a.Start) || k.End == a.Start
	aEI := k.Includes(a.End) || k.Start == a.End
	kSI := a.Includes(k.Start) || a.End == k.Start
	kEI := a.Includes(k.End) || a.Start == k.End

	switch {
	// Both complete keyspaces
	case k.Maxed() && a.Maxed():
		return k.Clone()

	// Overlapping keyspaces, ugh. Returns the sane default.
	// TODO(d4l3k): This should return two keyspaces.
	case aSI && aEI && kSI && kEI:
		return &Keyspace{Start: k.Start, End: a.End}

	// k encompasses a
	case aSI && aEI:
		return a.Clone()

	// a encompasses k
	case kSI && kEI:
		return k.Clone()

	// a.Start is in k
	case aSI:
		return &Keyspace{Start: a.Start, End: k.End}

	// a.End is in k
	case aEI:
		return &Keyspace{Start: k.Start, End: a.End}
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
