package protocol

import "github.com/spaolacci/murmur3"

//go:generate protoc --gogoslick_out=. protocol.proto

func (k *Keyspace) Includes(hash uint64) bool {
	a := hash
	s := k.Start
	e := k.End
	return s <= a && a < e || a < e && e < s || e < s && s <= a
}

func (msg *Message) Hash() uint64 {
	data, _ := msg.Marshal()
	return murmur3.Sum64(data)
}
