package crypto

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/degdb/degdb/protocol"
)

const (
	tmpKeyDir  = "crypto_test"
	tmpKeyFile = "test.key"
)

func TestKeyGeneration(t *testing.T) {
	t.Parallel()

	if _, err := GenerateKey(); err != nil {
		t.Error(err)
	}
}

func TestReadOrGenerateKey(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", tmpKeyDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	file := path.Join(dir, tmpKeyFile)
	key, err := ReadOrGenerateKey(file)
	if err != nil {
		t.Fatal(err)
	}
	key2, err := ReadOrGenerateKey(file)
	if err != nil {
		t.Fatal(err)
	}
	author1, err := key.AuthorID()
	if err != nil {
		t.Fatal(err)
	}
	author2, err := key2.AuthorID()
	if err != nil {
		t.Fatal(err)
	}
	if author1 != author2 {
		t.Fatalf("ReadOrGenerateKey is not properly saving and restoring key. 1: %#v 2: %#v", key, key2)
	}
}

func TestSignTriple(t *testing.T) {
	t.Parallel()

	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	triple := &protocol.Triple{}
	if err := key.SignTriple(triple); err != nil {
		t.Fatal(err)
	}
	author, err := key.AuthorID()
	if err != nil {
		t.Fatal(err)
	}
	if len(triple.Author) == 0 {
		t.Errorf("triple.Author not set")
	}
	if triple.Author != author {
		t.Errorf("triple.Author = %s; not %s", triple.Author, author)
	}
	if len(triple.Sig) == 0 {
		t.Errorf("triple.Sig not set")
	}
}
