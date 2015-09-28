package network

import (
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

func TestPeerDiscovery(t *testing.T) {
	stunOnce.Do(func() {})
	stunWG.Done()
	host = "localhost"

	nodeCount := 5
	port := 8181
	var nodes []*Server

	var wg sync.WaitGroup
	wg.Add(nodeCount)

	for i := 0; i < nodeCount; i++ {
		port := port + i
		logger := log.New(os.Stdout, fmt.Sprintf(":%d ", port), log.Flags())
		s, err := NewServer(logger, port)
		if err != nil {
			t.Error(err)
		}
		go func() {
			wg.Done()
			if err := s.Listen(); err != nil {
				t.Error(err)
			}
		}()
		nodes = append(nodes, s)
	}
	wg.Wait()
	for i, s := range nodes[1:] {
		i := i
		s := s
		go func() {
			err := s.Connect(fmt.Sprintf("localhost:%d", nodes[i].Port))
			if err != nil {
				t.Error(err)
			}
		}()
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	for _, node := range nodes {
		for _, peer := range nodes {
			protoPeer := peer.LocalPeer()
			if node.Peers[protoPeer.Id] == nil && node != peer {
				t.Errorf("node %+v missing peer %+v", node.LocalPeer(), protoPeer)
			}
		}
	}

}
