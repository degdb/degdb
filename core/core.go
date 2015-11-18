// Package core contains the rewritten degdb code.
package core

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fatih/color"

	"github.com/degdb/degdb/crypto"
	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/triplestore"
)

type server struct {
	diskAllocated int
	port          int
	network       *network.Server
	ts            *triplestore.TripleStore
	crypto        *crypto.PrivateKey

	*log.Logger
}

func Main(port int, peers []string, diskAllocated int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s := &server{
		Logger: log.New(os.Stdout,
			color.CyanString(":%d ", port),
			log.Flags()),
		diskAllocated: diskAllocated,
		port:          port,
	}

	if err := s.init(); err != nil {
		s.Fatal(err)
	}

	go s.connectPeers(peers)
	s.Fatal(s.network.Listen())
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
	keyFile := fmt.Sprintf("degdb-%d.key", s.port)
	privKey, err := crypto.ReadOrGenerateKey(keyFile)
	if err != nil {
		return err
	}
	s.crypto = privKey

	s.Printf("Initializing triplestore...")
	s.Printf("Max DB size = %d bytes.", s.diskAllocated)
	dbFile := fmt.Sprintf("degdb-%d.db", s.port)
	ts, err := triplestore.NewTripleStore(dbFile)
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
