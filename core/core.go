// Package core contains the rewritten degdb code.
package core

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/triplestore"
)

type server struct {
	diskAllocated int
	network       *network.Server
	ts            *triplestore.TripleStore

	*log.Logger
}

func Main(port int, peers []string, diskAllocated int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s := &server{
		Logger:        log.New(os.Stdout, fmt.Sprintf(":%d ", port), log.Flags()),
		diskAllocated: diskAllocated,
	}

	s.Printf("Initializing triplestore...")
	s.Printf("Max DB size = %d bytes.", diskAllocated)
	db := fmt.Sprintf("degdb-%d.db", port)
	ts, err := triplestore.NewTripleStore(db)
	if err != nil {
		s.Fatal(err)
	}
	s.ts = ts

	s.Printf("Initializing network...")
	ns, err := network.NewServer(s.Logger, port)
	if err != nil {
		s.Fatal(err)
	}
	s.network = ns

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
