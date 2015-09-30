package protocol

//go:generate protoc --gogoslick_out=. protocol.proto

func (k *Keyspace) Includes(hash uint64) bool {
	a := hash
	s := k.Start
	e := k.End
	return s <= a && a < e || a < e && e < s || e < s && s <= a
}
