// Package core contains the rewritten degdb code.
package core

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/degdb/degdb/bitcoin"
	"github.com/degdb/degdb/crypto"
	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/triplestore"
)

var (
	KeyFilePath      = "degdb-%d.key"
	DatabaseFilePath = "degdb-%d.db"
)

type server struct {
	diskAllocated int
	port          int
	network       *network.Server
	ts            *triplestore.TripleStore
	crypto        *crypto.PrivateKey

	*log.Logger
}

// Main launches a node with the specified parameters.
func Main(port int, peers []string, diskAllocated int) {
	s, err := newServer(port, peers, diskAllocated)
	if err != nil {
		log.Fatal(err)
	}

	bitcoin.NewClient()

	s.Fatal(s.network.Listen())
}

func newServer(port int, peers []string, diskAllocated int) (*server, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s := &server{
		Logger: log.New(os.Stdout,
			color.CyanString(":%d ", port),
			log.Flags()),
		diskAllocated: diskAllocated,
		port:          port,
	}

	if err := s.init(); err != nil {
		return nil, err
	}
	go s.connectPeers(peers)
	return s, nil
}

func (s *server) connectPeers(peers []string) {
	for _, peer := range peers {
		peer := peer
		time.Sleep(200 * time.Millisecond)
		go func() {
			s.Printf("Connecting to peer %s", peer)
			if err := s.network.Connect(peer); err != nil {
				s.Printf("ERR connecting to peer: %s", err)
			}
		}()
	}
}

func (s *server) init() error {
	s.Printf("Initializing crypto...")
	keyFile := fmt.Sprintf(KeyFilePath, s.port)
	privKey, err := crypto.ReadOrGenerateKey(keyFile)
	if err != nil {
		return err
	}
	s.crypto = privKey

	s.Printf("Initializing triplestore...")
	s.Printf("Max DB size = %d bytes.", s.diskAllocated)
	dbFile := fmt.Sprintf(DatabaseFilePath, s.port)
	ts, err := triplestore.NewTripleStore(dbFile, s.Logger)
	if err != nil {
		return err
	}
	s.ts = ts

	s.Printf("Initializing network...")
	ns, err := network.NewServer(s.Logger, s.port)
	if err != nil {
		return err
	}
	s.network = ns

	if err := s.initHTTP(); err != nil {
		return err
	}
	if err := s.initBinary(); err != nil {
		return err
	}

	return nil
}

// Stop stops the server and closes all open sockets.
func (s *server) Stop() {
	s.network.Stop()
}
