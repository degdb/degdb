package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/gob"
	"log"
	"os"
)

func initCrypto(name string) error {
	privatepath := *dbDir + "/private-" + name + ".key"
	publicpath := *dbDir + "/public-" + name + ".key"
	if _, err := os.Stat(privatepath); os.IsNotExist(err) {
		log.Printf("Keys not found, generating")
		err := generateCryptoKeys()
		if err != nil {
			return err
		}
		return writeCryptoKeys(privatepath, publicpath)
	}
	return readCryptoKeys(privatepath, publicpath)
}

var publickey ecdsa.PublicKey
var privatekey ecdsa.PrivateKey

func generateCryptoKeys() error {
	pubkeyCurve := elliptic.P256()

	privatekeygen, err := ecdsa.GenerateKey(pubkeyCurve, rand.Reader)

	if err != nil {
		return err
	}

	privatekey = *privatekeygen
	publickey = privatekey.PublicKey
	return nil
}

func writeCryptoKeys(privatepath, publicpath string) error {
	privatekeyfile, err := os.Create(privatepath)
	if err != nil {
		return err
	}
	privatekeyencoder := gob.NewEncoder(privatekeyfile)
	privatekeyencoder.Encode(privatekey)
	privatekeyfile.Close()

	publickeyfile, err := os.Create(publicpath)
	if err != nil {
		return err
	}

	publickeyencoder := gob.NewEncoder(publickeyfile)
	publickeyencoder.Encode(publickey)
	publickeyfile.Close()

	return nil
}

func readCryptoKeys(privatepath, publicpath string) error {
	privatekeyfile, err := os.Open(privatepath)
	if err != nil {
		return err
	}
	gob.NewDecoder(privatekeyfile).Decode(&privatekey)
	privatekeyfile.Close()

	publickeyfile, err := os.Open(publicpath)
	if err != nil {
		return err
	}

	gob.NewDecoder(publickeyfile).Decode(&publickey)
	publickeyfile.Close()

	return nil
}
