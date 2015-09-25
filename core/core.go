// Package core contains the rewritten degdb code.
package core

import (
	"log"
	"os"
	"strconv"

	"github.com/degdb/degdb/network"
)

func Main(port int, peers []string, diskAllocated int) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log := log.New(os.Stderr, ":"+strconv.Itoa(port), log.Flags())

	log.Printf("Initializing...")
	s, err := network.NewServer(log)
	if err != nil {
		s.Fatal(err)
	}

	log.Printf("Allocated %d bytes.", diskAllocated)

	go setup(s, peers)
	s.Fatal(s.Listen(port))
}

func setup(s *network.Server, peers []string) {
	for _, peer := range peers {
		log.Printf("Connecting to peer %s", peer)
		s.Connect(peer)
	}
}
