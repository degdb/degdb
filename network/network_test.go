package network

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/d4l3k/messagediff"
	"github.com/degdb/degdb/protocol"
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

func TestMinimumCoveringPeers(t *testing.T) {
	testData := []struct {
		keyspaces []*protocol.Keyspace
		cover     bool
	}{
		{
			nil,
			false,
		},
		{
			[]*protocol.Keyspace{
				{1, 20},
				{20, 1},
			},
			true,
		},
		{
			[]*protocol.Keyspace{
				{1, 20},
				{20, 30},
				{30, 50},
				{50, 1},
			},
			true,
		},
	}
	for i, td := range testData {
		s := &Server{Peers: make(map[string]*Conn)}
		for j, keyspace := range td.keyspaces {
			id := strconv.Itoa(j)
			s.Peers[id] = &Conn{
				Peer: &protocol.Peer{
					Keyspace: keyspace,
					Id:       id,
				}}
		}
		min := s.MinimumCoveringPeers()
		sort.Sort(sortConnByKeyspace(min))
		union := &protocol.Keyspace{}

		// Twice to make sure there aren't any order issues with 1 space gaps.
		for i := 0; i < 2; i++ {
			for _, peer := range min {
				union = union.Union(peer.Peer.Keyspace)
			}
		}
		if union.Maxed() != td.cover {
			t.Errorf("%d. s.MinimumCoveringPeers().Union().Maxed() != %+v, %+v, %+v", i, td.cover, math.MaxUint64-union.Mag(), min)
		}
	}
}

type sortConnByKeyspace []*Conn

func (s sortConnByKeyspace) Len() int {
	return len(s)
}
func (s sortConnByKeyspace) Less(i, j int) bool {
	return s[i].Peer.Keyspace.Start < s[j].Peer.Keyspace.Start
}
func (s sortConnByKeyspace) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func TestLocalPeer(t *testing.T) {
	testData := []struct {
		ip      string
		port    int
		serving bool
		want    *protocol.Peer
	}{
		{
			"127.0.0.1", 7946, false,
			&protocol.Peer{
				Id:       "127.0.0.1:7946",
				Serving:  false,
				Keyspace: &protocol.Keyspace{Start: 0x5677ecc49097f350, End: 0xd677ecc49097f34e},
			},
		},
		{
			"127.0.0.2", 7947, true,
			&protocol.Peer{
				Id:       "127.0.0.2:7947",
				Serving:  true,
				Keyspace: &protocol.Keyspace{Start: 0xfa367fcf18de76bd, End: 0x7a367fcf18de76bb},
			},
		},
	}
	for i, td := range testData {
		s := Server{
			IP:      td.ip,
			Port:    td.port,
			Serving: td.serving,
		}
		out := s.LocalPeer()
		if diff, ok := messagediff.PrettyDiff(td.want, out); !ok {
			t.Errorf("%d. %#v.LocalPeer() = %#v; diff %s", i, s, out, diff)
		}
	}
}
