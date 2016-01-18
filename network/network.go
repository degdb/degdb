// Package network takes care of networking for degdb.
package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/spaolacci/murmur3"

	"github.com/degdb/degdb/network/ip"
	"github.com/degdb/degdb/protocol"
)

type protocolHandler func(conn *Conn, msg *protocol.Message)

var (
	ErrNoRecipients = errors.New("no recipients for the message were found")

	stunOnce sync.Once
	stunWG   sync.WaitGroup
	stunHost string
)

func init() {
	stunWG.Add(1)
}

func getHost() string {
	stunOnce.Do(func() {
		// TODO(d4l3k): Fetch IP by talking to other nodes.
		ip, err := ip.IP()
		if err != nil {
			log.Fatal(err)
		}
		stunHost = ip
		stunWG.Done()
	})
	stunWG.Wait()

	return stunHost
}

// Server handles all network traffic.
type Server struct {
	IP   string
	Port int

	Serving bool

	HTTP          *http.Server
	mux           *http.ServeMux
	httpEndpoints []string

	Peers     map[string]*Conn
	peersLock sync.RWMutex

	netListener net.Listener
	// listeningWG waits for the server to start listening and accepting connections.
	listeningWG sync.WaitGroup

	handlers map[string]protocolHandler
	listener *httpListener
	*log.Logger
}

// NewServer creates a new server with routing information. If log is nil, stdout is used.
func NewServer(logger *log.Logger, port int) (*Server, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "", log.Flags())
	}
	s := &Server{
		Logger:   logger,
		Port:     port,
		Peers:    make(map[string]*Conn),
		handlers: make(map[string]protocolHandler),
	}

	s.listeningWG.Add(1)

	s.IP = getHost()

	s.listener = &httpListener{
		accept: make(chan *httpConn, 10),
	}

	s.initHTTPRouting()

	// Handlers
	s.Handle("Handshake", s.handleHandshake)
	s.Handle("PeerRequest", s.handlePeerRequest)
	s.Handle("PeerNotify", s.handlePeerNotify)

	return s, nil
}

type Closable interface {
	Close() error
}

// Stop closes all connections and cleans up.
func (s *Server) Stop() {
	toClose := []Closable{s.netListener, s.listener}

	s.peersLock.Lock()
	defer s.peersLock.Unlock()
	for _, peer := range s.Peers {
		toClose = append(toClose, peer)
	}

	for _, close := range toClose {
		if close == nil {
			continue
		}
		if err := close.Close(); err != nil {
			s.Println(err)
		}
	}
}

// ListenWait waits for the server to start accepting connections before returning.
func (s *Server) ListenWait() {
	s.listeningWG.Wait()
}

// Connect to another server. `addr` should be in the format "google.com:80".
func (s *Server) Connect(addr string) error {
	netConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	tcpConn := netConn.(*net.TCPConn)
	tcpConn.SetKeepAlive(true)

	conn := s.NewConn(tcpConn)

	if err := s.sendHandshake(conn, protocol.HANDSHAKE_INITIAL); err != nil {
		return err
	}

	go s.handleConnection(conn)
	return nil
}

// Listen for incoming connections on the specified port.
func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	s.netListener = ln
	addr := ln.Addr().(*net.TCPAddr)
	if s.Port == 0 {
		s.Port = addr.Port
	}

	go s.listenHTTP(addr)
	s.Printf("Listening: 0.0.0.0:%d, ip: %s", s.Port, s.IP)
	s.listeningWG.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		s.Printf("New connection from %s", conn.RemoteAddr())
		go s.handleConnection(s.NewConn(conn))
	}
}

// Handle registers a handler for a specific protobuf message type.
func (s *Server) Handle(typ string, f protocolHandler) {
	s.handlers[typ] = f
}

// Broadcast sends a message to all peers with that have the hash in their keyspace.
// If there is no peer that can receive the message, ErrNoRecipients is returned.
func (s *Server) Broadcast(hash *uint64, msg *protocol.Message) error {
	alreadySentTo := make(map[uint64]bool)
	if msg.Gossip {
		for _, to := range msg.SentTo {
			alreadySentTo[to] = true
		}
	}
	sentTo := []uint64{murmur3.Sum64([]byte(s.LocalID()))}
	var toPeers []*Conn
	for _, peer := range s.Peers {
		peerHash := murmur3.Sum64([]byte(peer.Peer.Id))
		if (hash == nil || peer.Peer.GetKeyspace().Includes(*hash)) && !alreadySentTo[peerHash] {
			sentTo = append(sentTo, peerHash)
			toPeers = append(toPeers, peer)
		}
	}
	if msg.Gossip {
		msg.SentTo = append(msg.SentTo, sentTo...)
	}
	if len(toPeers) == 0 {
		return ErrNoRecipients
	}
	for _, peer := range toPeers {
		s.Printf("Broadcasting to %s", peer.Peer.Id)
		if err := peer.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleConnection(conn *Conn) error {
	var err error
	for {
		header := make([]byte, 4)
		_, err = conn.Read(header)
		if err != nil {
			break
		}
		if string(header) == "GET " || string(header) == "POST" {
			s.Printf("Incoming HTTP connection.")
			s.handleHTTPConnection(header, conn)
			return nil
		}
		length := binary.BigEndian.Uint32(header)
		if length > 10000000 {
			err = fmt.Errorf("Packet larger than 10MB! len = %s", humanize.SI(float64(length), "B"))
			break
		}
		buf := make([]byte, length)
		_, err = conn.Read(buf)
		if err != nil {
			break
		}

		req := &protocol.Message{}
		if err = req.Unmarshal(buf); err != nil {
			break
		}
		s.Printf("Message: <- %s, %+v", conn.PrettyID(), req.GetMessage())
		if req.ResponseTo != 0 {
			if c, ok := conn.expectedMessages[req.ResponseTo]; ok {
				c <- req
				continue
			} else {
				err = fmt.Errorf("response sent to invalid request %d", req.ResponseTo)
				break
			}
		}
		rawType := reflect.TypeOf(req.GetMessage()).Elem().Name()
		typ := strings.TrimPrefix(rawType, "Message_")
		handler, ok := s.handlers[typ]
		if !ok {
			err = fmt.Errorf("no handler for message type %s", typ)
			break
		}
		go handler(conn, req)
	}
	s.Printf("Connection closed. %s", err)
	conn.Close()
	if conn.Peer != nil {
		delete(s.Peers, conn.Peer.Id)
	}
	return err
}

// LocalPeer returns a peer object of the current server.
func (s *Server) LocalPeer() *protocol.Peer {
	return &protocol.Peer{
		Id:       s.LocalID(),
		Serving:  s.Serving,
		Keyspace: s.LocalKeyspace(),
	}
}

// LocalKeyspace returns the keyspace that the local node represents.
func (s *Server) LocalKeyspace() *protocol.Keyspace {
	center := murmur3.Sum64([]byte(s.LocalID()))
	return &protocol.Keyspace{
		Start: center - math.MaxUint64/4,
		End:   center + math.MaxUint64/4,
	}
}

// LocalID returns the local machines ID.
func (s *Server) LocalID() string {
	return net.JoinHostPort(s.IP, strconv.Itoa(s.Port))
}

// MinimumCoveringPeers returns a set of peers that minimizes overlap. This is similar to the Set Covering Problem and is NP-hard.
// This is a greedy algorithm. While the keyspace is not entirely covered, scan through all peers and pick the peer that will add the most to the set while still having the start in the selected set.
// TODO(wiz): Make this more optimal.
// TODO(wiz): achieve n-redundancy
func (s *Server) MinimumCoveringPeers() []*Conn {
	usedPeers := make(map[string]bool)
	var peers []*Conn
	var keyspace *protocol.Keyspace
	for i := 0; i < len(s.Peers) && !keyspace.Maxed(); i++ {
		var bestPeer *Conn
		var increase uint64
		// By definition, ranging through peer map will go in random order.
	Peers:
		for id, conn := range s.Peers {
			if conn == nil || conn.Peer == nil || usedPeers[id] {
				continue
			}
			peer := conn.Peer
			if keyspace == nil {
				peers = append(peers, conn)
				keyspace = peer.Keyspace
				break Peers
			}
			incr := keySpaceIncrease(keyspace, peer.Keyspace)
			if incr > increase {
				increase = incr
				bestPeer = conn
			}
		}
		if bestPeer != nil {
			peers = append(peers, bestPeer)
			keyspace = keyspace.Union(bestPeer.Peer.Keyspace)
			usedPeers[bestPeer.Peer.Id] = true
			// break?
		}
	}
	return peers
}

// keySpaceIncrease calculates the increase in keyspace if b was to be unioned.
func keySpaceIncrease(a, b *protocol.Keyspace) uint64 {
	unionMag := a.Union(b).Mag()
	aMag := a.Mag()
	if unionMag > aMag {
		return unionMag - aMag
	}
	return 0
}
