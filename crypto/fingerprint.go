package crypto

import (
	"crypto/sha1"

	"github.com/degdb/degdb/protocol"
)

// FingerprintTriple generates a SHA-1 hash of the triple.
func FingerprintTriple(t *protocol.Triple) ([]byte, error) {
	h := sha1.New()
	data, err := t.Marshal()
	if err != nil {
		return nil, err
	}
	if _, err := h.Write(data); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
