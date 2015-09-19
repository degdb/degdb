package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/degdb/degdb/protocol"
	"github.com/spaolacci/murmur3"
)

var (
	ellipticCurve = elliptic.P256()
)

type PrivateKey ecdsa.PrivateKey

func GenerateKey() (*PrivateKey, error) {
	key, err := ecdsa.GenerateKey(ellipticCurve, rand.Reader)
	return (*PrivateKey)(key), err
}

func ReadKey(path string) (*PrivateKey, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParseECPrivateKey(buf)
	return (*PrivateKey)(key), err
}

func ReadOrGenerateKey(path string) (*PrivateKey, error) {
	key, err := ReadKey(path)
	// TODO(d4l3k): Better way of checking key existence.
	if key != nil {
		return key, err
	}
	key, err = GenerateKey()
	if err != nil {
		return nil, err
	}
	key.Write(path)
	return key, nil
}

func (key *PrivateKey) Write(path string) error {
	buf, err := x509.MarshalECPrivateKey((*ecdsa.PrivateKey)(key))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, buf, 0644); err != io.EOF {
		return err
	}
	return nil
}

func (key *PrivateKey) SignTriple(t *protocol.Triple) error {
	var err error
	t.Author, err = key.AuthorID()
	if err != nil {
		return err
	}
	fingerprint, err := FingerprintTriple(t)
	if err != nil {
		return err
	}

	r, s, err := ecdsa.Sign(rand.Reader, (*ecdsa.PrivateKey)(key), fingerprint)
	if err != nil {
		return err
	}

	t.Sig = string(r.Bytes()) + string(s.Bytes())
	return nil
}

// AuthorID generates a unique ID based on the murmur hash of the public key.
func (key *PrivateKey) AuthorID() (string, error) {
	hasher := murmur3.New64()
	buf, err := x509.MarshalPKIXPublicKey((*ecdsa.PrivateKey)(key).PublicKey)
	if err != nil {
		return "", err
	}
	hasher.Write(buf)
	return "degdb:author_" + strconv.Itoa(int(hasher.Sum64())), nil
}
