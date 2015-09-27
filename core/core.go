// Package core contains the rewritten degdb code.
package core

import (
	"log"
	"os"
	"strconv"

	"github.com/degdb/degdb/network"
)

type server struct {
	diskAllocated int
	network       *network.Server

	*log.Logger
}

func Main(port int, peers []string, diskAllocated int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	s := &server{
		Logger:        log.New(os.Stderr, ":"+strconv.Itoa(port)+" ", log.Flags()),
		diskAllocated: diskAllocated,
	}

	s.Printf("Initializing...")
	s.Printf("Allocated %d bytes.", diskAllocated)

	ns, err := network.NewServer(s.Logger)
	if err != nil {
		s.Fatal(err)
	}
	s.network = ns

	for _, peer := range peers {
		s.Printf("Connecting to peer %s", peer)
		s.network.Connect(peer)
	}

	s.Fatal(s.network.Listen(port))
}
