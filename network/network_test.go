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

const retryCount = 10

func launchSwarm(nodeCount int, t *testing.T) []*Server {
	stunOnce.Do(func() {
		stunWG.Done()
	})
	stunHost = "localhost"

	port := 11100
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
				t.Log(err)
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

	for i := 0; i < retryCount; i++ {
		var errors []error
		for _, node := range nodes {
			for _, peer := range nodes {
				protoPeer := peer.LocalPeer()
				if node.Peers[protoPeer.Id] == nil && node != peer {
					errors = append(errors, fmt.Errorf("node %+v missing peer %+v", node.LocalPeer(), protoPeer))
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

func killSwarm(nodes []*Server) {
	for _, node := range nodes {
		node.Stop()
	}
	time.Sleep(200 * time.Millisecond)
}

func TestPeerDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	nodes := launchSwarm(5, t)
	defer killSwarm(nodes)
}

func TestMinimumCoveringPeers(t *testing.T) {
	t.Parallel()

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
		if len(min) > len(td.keyspaces) {
			t.Errorf("%d. len(s.MinimumCoveringPeers()) = %d > num keyspaces = %d", i, len(min), len(td.keyspaces))
		}
		sort.Sort(sortConnByKeyspace(min))
		union := &protocol.Keyspace{}

		var keyspaces []protocol.Keyspace

		// Multiple times to make sure there aren't any order issues with 1 space gaps.
		for i := 0; i < len(min); i++ {
			for _, peer := range min {
				if i == 0 {
					keyspaces = append(keyspaces, *peer.Peer.Keyspace)
				}
				union = union.Union(peer.Peer.Keyspace)
			}
		}
		if union.Maxed() != td.cover {
			t.Errorf("%d. s.MinimumCoveringPeers().Union().Maxed() != %+v, %+v, %+v", i, td.cover, math.MaxUint64-union.Mag(), keyspaces)
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
	t.Parallel()

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

func TestGetHost(t *testing.T) {
	resetStun()

	host := getHost()
	if len(host) == 0 {
		t.Error("getHost() len = 0")
	}
	want := "test"
	stunHost = want
	host2 := getHost()
	if host2 != want {
		t.Errorf("getHost() = %+v; not %+v", host2, want)
	}
}

func resetStun() {
	var testOnce sync.Once
	var testWG sync.WaitGroup
	testWG.Add(1)
	stunOnce = testOnce
	stunWG = testWG
	stunHost = ""
}
