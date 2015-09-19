package crypto

import (
	"testing"

	"github.com/degdb/degdb/protocol"
)

func TestFingerprint(t *testing.T) {
	testData := []struct {
		triple *protocol.Triple
		length int
	}{{
		&protocol.Triple{
			Subj: "foo",
			Pred: "bar",
			Obj:  "duck",
		},
		20,
	}}
	for i, td := range testData {
		hash, err := FingerprintTriple(td.triple)
		if err != nil {
			t.Error(err)
		}
		if len(hash) != td.length {
			t.Errorf("%d. FingerprintTriple(%#v) = %#v; not len %#v", i, td.triple, hash, td.length)
		}
	}
}

func BenchmarkFingerprint(b *testing.B) {
	triple := &protocol.Triple{
		Subj: "/m/02mjmr",
		Pred: "/type/object/name",
		Obj:  "Barack Obama",
		Lang: "en",
	}
	key, err := GenerateKey()
	if err != nil {
		b.Fatal(err)
	}
	if err := key.SignTriple(triple); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FingerprintTriple(triple)
	}
}
