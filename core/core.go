// Package core contains the rewritten degdb code.
package core

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/degdb/degdb/network"
	"github.com/degdb/degdb/protocol"
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
		Logger:        log.New(os.Stderr, fmt.Sprintf(":%d ", port), log.Flags()),
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
	ns, err := network.NewServer(s.Logger)
	if err != nil {
		s.Fatal(err)
	}
	s.network = ns

	s.network.Handle("PeerRequest", func(conn *network.Conn, msg *protocol.Message) {
		s.Printf("PeerRequest %#v", *msg.GetPeerRequest())
	})

	for _, peer := range peers {
		time.Sleep(200 * time.Millisecond)
		s.Printf("Connecting to peer %s", peer)
		if err := s.network.Connect(peer); err != nil {
			s.Printf("ERR connecting to peer: %s", err)
		}
	}

	s.Fatal(s.network.Listen(port))
}
