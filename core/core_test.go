package core

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

const (
	retryCount    = 10
	diskAllocated = 1e9 // 1GB
)

func launchSwarm(nodeCount int, t *testing.T) []*server {
	port := 11200
	var nodes []*server

	var wg sync.WaitGroup
	wg.Add(nodeCount)

	var peers []string
	for i := 0; i < nodeCount; i++ {
		port := port + i
		s, err := newServer(port, peers, diskAllocated)
		if err != nil {
			t.Error(err)
		}
		go func() {
			wg.Done()
			if err := s.network.Listen(); err != nil {
				t.Log(err)
			}
		}()
		peers = append(peers, fmt.Sprintf("localhost:%d", s.network.Port))
		nodes = append(nodes, s)
		time.Sleep(20 * time.Millisecond)
	}
	wg.Wait()

	for i := 0; i < retryCount; i++ {
		var errors []error
		for _, node := range nodes {
			for _, peer := range nodes {
				protoPeer := peer.network.LocalPeer()
				if node.network.Peers[protoPeer.Id] == nil && node != peer {
					errors = append(errors, fmt.Errorf("node %+v missing peer %+v", node.network.LocalPeer(), protoPeer))
				}
			}
		}
		if len(errors) == 0 {
			break
		}
		if i < retryCount-1 {
			log.Printf("Rechecking peer discovery... %d times", retryCount-i)
			time.Sleep(200 * time.Millisecond)
			continue
		}
		for _, err := range errors {
			t.Error(err)
		}
	}
	return nodes
}

func killSwarm(nodes []*server) {
	for _, node := range nodes {
		node.Stop()
	}
	time.Sleep(200 * time.Millisecond)
}

func TestCoreDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	nodes := launchSwarm(5, t)
	defer killSwarm(nodes)
}
