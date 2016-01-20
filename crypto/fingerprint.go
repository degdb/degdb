package crypto

import (
	"crypto/sha1"

	"github.com/degdb/degdb/protocol"
)

// FingerprintTriple generates a SHA-1 hash of the triple.
func FingerprintTriple(t *protocol.Triple) ([]byte, error) {
	data, err := t.Marshal()
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum(data)
	return sum[:], nil
}
