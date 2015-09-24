package old

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/spaolacci/murmur3"
)

func initCrypto(name string) error {
	privatepath := *dbDir + "/private-" + name + ".key"
	if _, err := os.Stat(privatepath); os.IsNotExist(err) {
		log.Printf("Keys not found, generating")
		err := generateCryptoKeys()
		if err != nil {
			return err
		}
		return writeCryptoKeys(privatepath)
	}
	return readCryptoKeys(privatepath)
}

var privatekey *ecdsa.PrivateKey

func generateCryptoKeys() error {
	pubkeyCurve := elliptic.P256()

	var err error
	privatekey, err = ecdsa.GenerateKey(pubkeyCurve, rand.Reader)
	if err != nil {
		return err
	}

	return nil
}

func writeCryptoKeys(privatepath string) error {
	buf, err := x509.MarshalECPrivateKey(privatekey)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(privatepath, buf, 0644); err != io.EOF {
		return err
	}
	return nil
}

func readCryptoKeys(privatepath string) error {
	buf, err := ioutil.ReadFile(privatepath)
	if err != nil {
		return err
	}
	privatekey, err = x509.ParseECPrivateKey(buf)
	if err != nil {
		return err
	}
	return nil
}

func authorID() (string, error) {
	hasher := murmur3.New64()
	buf, err := x509.MarshalPKIXPublicKey(&privatekey.PublicKey)
	if err != nil {
		return "", err
	}
	hasher.Write(buf)
	return "degdb:author_" + strconv.Itoa(int(hasher.Sum64())), nil
}

func (t *Triple) Fingerprint() []byte {
	h := sha1.New()
	io.WriteString(h, t.Subj)
	io.WriteString(h, t.Pred)
	io.WriteString(h, t.Obj)
	io.WriteString(h, t.Lang)
	io.WriteString(h, t.Author)
	return h.Sum(nil)
}

func (t *Triple) Sign() error {
	var err error
	t.Author, err = authorID()
	if err != nil {
		return err
	}
	fingerprint := t.Fingerprint()

	r, s, err := ecdsa.Sign(rand.Reader, privatekey, fingerprint)
	if err != nil {
		return err
	}

	t.Sig = string(r.Bytes()) + string(s.Bytes())
	return nil
}
